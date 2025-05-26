#!/bin/bash

# Gon Cloud Platform (GCP) Control Plane Setup Script
# Ubuntu 22.04 based

set -e

echo "=========================================="
echo "Starting GCP Control Plane setup..."
echo "=========================================="

# Variable configuration
CONTROL_PLANE_IP="121.36.55.1"
POSTGRES_PASSWORD="gcp_secure_password_2024"
RABBITMQ_USER="gcp_admin"
RABBITMQ_PASSWORD="gcp_rabbitmq_password_2024"

# System update
echo "[1/8] Updating system packages..."
sudo apt update && sudo apt upgrade -y

# Install essential packages
echo "[2/8] Installing essential packages..."
sudo apt install -y \
    curl \
    wget \
    git \
    vim \
    htop \
    net-tools \
    bridge-utils \
    openvswitch-switch \
    openvswitch-common \
    iptables-persistent \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release

# Docker installation
echo "[3/8] Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker $USER

# Go 1.21 installation
echo "[4/8] Installing Go 1.21..."
GO_VERSION="1.21.5"
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm go${GO_VERSION}.linux-amd64.tar.gz

# Go environment variable setup
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# PostgreSQL 14 installation and configuration
echo "[5/8] Installing and configuring PostgreSQL 14..."
sudo apt install -y postgresql-14 postgresql-client-14 postgresql-contrib-14
sudo systemctl start postgresql
sudo systemctl enable postgresql

# PostgreSQL database and user creation
sudo -u postgres psql << EOF
CREATE DATABASE gcp_platform;
CREATE USER gcp_user WITH ENCRYPTED PASSWORD '$POSTGRES_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE gcp_platform TO gcp_user;
ALTER USER gcp_user CREATEDB;
\q
EOF

# PostgreSQL configuration file modification (allow external connections)
sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/14/main/postgresql.conf
echo "host    all             all             121.36.55.0/24          md5" | sudo tee -a /etc/postgresql/14/main/pg_hba.conf
sudo systemctl restart postgresql

# RabbitMQ installation and configuration
echo "[6/8] Installing and configuring RabbitMQ..."
sudo apt install -y rabbitmq-server
sudo systemctl start rabbitmq-server
sudo systemctl enable rabbitmq-server

# Enable RabbitMQ management plugin
sudo rabbitmq-plugins enable rabbitmq_management

# RabbitMQ user and permission setup
sudo rabbitmqctl add_user $RABBITMQ_USER $RABBITMQ_PASSWORD
sudo rabbitmqctl set_user_tags $RABBITMQ_USER administrator
sudo rabbitmqctl set_permissions -p / $RABBITMQ_USER ".*" ".*" ".*"

# etcd installation
echo "[7/8] Installing etcd..."
ETCD_VER=v3.5.10
GITHUB_URL=https://github.com/etcd-io/etcd/releases/download
DOWNLOAD_URL=${GITHUB_URL}
curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp
sudo mv /tmp/etcd-${ETCD_VER}-linux-amd64/etcd* /usr/local/bin/
rm -rf /tmp/etcd-${ETCD_VER}-linux-amd64*

# Create etcd service file
sudo tee /etc/systemd/system/etcd.service > /dev/null << EOF
[Unit]
Description=etcd distributed reliable key-value store
After=network.target
Wants=network-online.target

[Service]
Type=notify
User=etcd
ExecStart=/usr/local/bin/etcd \\
  --name=control-plane \\
  --data-dir=/var/lib/etcd \\
  --listen-client-urls=http://0.0.0.0:2379 \\
  --advertise-client-urls=http://$CONTROL_PLANE_IP:2379 \\
  --listen-peer-urls=http://0.0.0.0:2380 \\
  --initial-advertise-peer-urls=http://$CONTROL_PLANE_IP:2380 \\
  --initial-cluster=control-plane=http://$CONTROL_PLANE_IP:2380 \\
  --initial-cluster-token=gcp-cluster \\
  --initial-cluster-state=new
Restart=always
RestartSec=10s
LimitNOFILE=40000

[Install]
WantedBy=multi-user.target
EOF

# Create etcd user and directory setup
sudo useradd -r -s /bin/false etcd
sudo mkdir -p /var/lib/etcd
sudo chown etcd:etcd /var/lib/etcd
sudo systemctl daemon-reload
sudo systemctl enable etcd
sudo systemctl start etcd

# Open vSwitch configuration
echo "[8/8] Configuring Open vSwitch..."
sudo systemctl start openvswitch-switch
sudo systemctl enable openvswitch-switch

# Create main bridge
sudo ovs-vsctl add-br br-main
sudo ovs-vsctl set-manager ptcp:6640

# Create network configuration file
sudo tee /etc/netplan/99-gcp-network.yaml > /dev/null << EOF
network:
  version: 2
  renderer: networkd
  bridges:
    br-main:
      interfaces: []
      parameters:
        stp: false
      dhcp4: false
      addresses:
        - 10.0.0.1/16
EOF

# Create GCP platform directory structure
echo "Creating GCP platform directory structure..."
sudo mkdir -p /opt/gcp/{bin,config,logs,data}
sudo mkdir -p /opt/gcp/config/{api-server,network-controller,instance-manager}
sudo chown -R $USER:$USER /opt/gcp

# Create environment configuration file
tee /opt/gcp/config/gcp.env > /dev/null << EOF
# GCP Platform Configuration
CONTROL_PLANE_IP=$CONTROL_PLANE_IP
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=gcp_platform
POSTGRES_USER=gcp_user
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=$RABBITMQ_USER
RABBITMQ_PASSWORD=$RABBITMQ_PASSWORD
ETCD_ENDPOINTS=http://localhost:2379
API_SERVER_PORT=8080
NETWORK_CONTROLLER_PORT=8081
INSTANCE_MANAGER_PORT=8082
LOG_LEVEL=info
EOF

# Firewall configuration (UFW)
echo "Configuring firewall..."
sudo ufw enable
sudo ufw allow ssh
sudo ufw allow 8080/tcp  # API Server
sudo ufw allow 8081/tcp  # Network Controller  
sudo ufw allow 8082/tcp  # Instance Manager
sudo ufw allow 5432/tcp  # PostgreSQL
sudo ufw allow 5672/tcp  # RabbitMQ
sudo ufw allow 15672/tcp # RabbitMQ Management
sudo ufw allow 2379/tcp  # etcd client
sudo ufw allow 2380/tcp  # etcd peer
sudo ufw allow 6640/tcp  # Open vSwitch

# Check system service status
echo "Checking service status..."
sudo systemctl status postgresql --no-pager -l
sudo systemctl status rabbitmq-server --no-pager -l  
sudo systemctl status etcd --no-pager -l
sudo systemctl status openvswitch-switch --no-pager -l

echo "=========================================="
echo "Control Plane setup completed successfully!"
echo "=========================================="
echo "Please verify the following information:"
echo "- PostgreSQL: localhost:5432 (user: gcp_user)"
echo "- RabbitMQ: localhost:5672 (admin: $RABBITMQ_USER)"
echo "- RabbitMQ Management: http://$CONTROL_PLANE_IP:15672"
echo "- etcd: localhost:2379"
echo "- Open vSwitch bridge: br-main"
echo ""
echo "All services will start automatically after reboot."
echo "Please log out and log back in to apply Docker group permissions."