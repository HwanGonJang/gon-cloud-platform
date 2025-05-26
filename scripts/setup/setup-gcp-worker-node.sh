#!/bin/bash

# Gon Cloud Platform (GCP) Worker Node Setup Script
# Ubuntu 22.04 based

set -e

echo "=========================================="
echo "Starting GCP Worker Node setup..."
echo "=========================================="

# Variable configuration - received as script arguments
WORKER_NODE_IP=${1:-""}
NODE_NAME=${2:-"worker-$(hostname)"}
CONTROL_PLANE_IP="121.36.55.1"

if [ -z "$WORKER_NODE_IP" ]; then
    echo "Usage: $0 <WORKER_NODE_IP> [NODE_NAME]"
    echo "Example: $0 121.36.55.2 worker-1"
    exit 1
fi

echo "Worker Node IP: $WORKER_NODE_IP"
echo "Node Name: $NODE_NAME"
echo "Control Plane IP: $CONTROL_PLANE_IP"

# System update
echo "[1/7] Updating system packages..."
sudo apt update && sudo apt upgrade -y

# Install essential packages
echo "[2/7] Installing essential packages..."
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
    lsb-release \
    qemu-kvm \
    libvirt-daemon-system \
    libvirt-clients \
    virtinst \
    virt-manager \
    cpu-checker \
    libguestfs-tools \
    libosinfo-bin

# Check KVM virtualization support
echo "Checking KVM virtualization support..."
if kvm-ok | grep -q "KVM acceleration can be used"; then
    echo "✓ KVM virtualization is supported."
else
    echo "⚠ KVM virtualization is not supported or not enabled."
    echo "Please enable virtualization in BIOS or check nested virtualization."
fi

# Docker installation
echo "[3/7] Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker $USER

# Go 1.21 installation
echo "[4/7] Installing Go 1.21..."
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

# KVM and Libvirt configuration
echo "[5/7] Configuring KVM and Libvirt..."
sudo systemctl start libvirtd
sudo systemctl enable libvirtd
sudo usermod -aG libvirt $USER
sudo usermod -aG kvm $USER

# Disable libvirt default network (use GCP's own networking)
sudo virsh net-destroy default 2>/dev/null || true
sudo virsh net-undefine default 2>/dev/null || true

# Open vSwitch configuration
echo "[6/7] Configuring Open vSwitch..."
sudo systemctl start openvswitch-switch
sudo systemctl enable openvswitch-switch

# Create worker node bridge
sudo ovs-vsctl add-br br-worker
sudo ovs-vsctl set-controller br-worker tcp:$CONTROL_PLANE_IP:6640

# Connect physical interface to bridge (optional - manual configuration if needed)
# PRIMARY_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
# sudo ovs-vsctl add-port br-worker $PRIMARY_INTERFACE

# Create VM image storage directories
echo "Setting up VM and container storage..."
sudo mkdir -p /var/lib/gcp/{images,instances,volumes,snapshots}
sudo mkdir -p /var/lib/gcp/templates/{ubuntu,debian,centos}
sudo chown -R libvirt-qemu:kvm /var/lib/gcp/images
sudo chown -R libvirt-qemu:kvm /var/lib/gcp/instances
sudo chmod 755 /var/lib/gcp/{images,instances,volumes,snapshots}

# LVM configuration (for storage volume management)
echo "Setting up LVM storage..."
sudo apt install -y lvm2

# Create GCP platform directory structure
echo "[7/7] Creating GCP platform directory structure..."
sudo mkdir -p /opt/gcp/{bin,config,logs,data}
sudo mkdir -p /opt/gcp/config/{hypervisor-agent,network-agent,container-agent}
sudo chown -R $USER:$USER /opt/gcp

# Create environment configuration file
tee /opt/gcp/config/worker.env > /dev/null << EOF
# GCP Worker Node Configuration
WORKER_NODE_IP=$WORKER_NODE_IP
NODE_NAME=$NODE_NAME
CONTROL_PLANE_IP=$CONTROL_PLANE_IP
HYPERVISOR_AGENT_PORT=9090
NETWORK_AGENT_PORT=9091
CONTAINER_AGENT_PORT=9092
ETCD_ENDPOINTS=http://$CONTROL_PLANE_IP:2379
LOG_LEVEL=info

# KVM/QEMU Configuration
QEMU_SYSTEM_PATH=/usr/bin/qemu-system-x86_64
LIBVIRT_URI=qemu:///system
VM_IMAGES_PATH=/var/lib/gcp/images
VM_INSTANCES_PATH=/var/lib/gcp/instances
VM_VOLUMES_PATH=/var/lib/gcp/volumes

# Docker Configuration
DOCKER_HOST=unix:///var/run/docker.sock
CONTAINER_RUNTIME=docker

# Network Configuration
OVS_BRIDGE=br-worker
OVS_CONTROLLER=tcp://$CONTROL_PLANE_IP:6640
EOF

# Create network configuration file
sudo tee /etc/netplan/99-gcp-worker-network.yaml > /dev/null << EOF
network:
  version: 2
  renderer: networkd
  bridges:
    br-worker:
      interfaces: []
      parameters:
        stp: false
      dhcp4: false
      addresses:
        - 10.1.0.1/24
EOF

# Firewall configuration (UFW)
echo "Configuring firewall..."
sudo ufw enable
sudo ufw allow ssh
sudo ufw allow 9090/tcp  # Hypervisor Agent
sudo ufw allow 9091/tcp  # Network Agent
sudo ufw allow 9092/tcp  # Container Agent
sudo ufw allow from $CONTROL_PLANE_IP  # Allow Control Plane access

# Docker network configuration (GCP dedicated)
echo "Configuring Docker network..."
sudo tee /etc/docker/daemon.json > /dev/null << EOF
{
  "bridge": "none",
  "iptables": false,
  "ip-forward": true,
  "storage-driver": "overlay2",
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF

sudo systemctl restart docker

# Check system service status
echo "Checking service status..."
sudo systemctl status libvirtd --no-pager -l
sudo systemctl status docker --no-pager -l
sudo systemctl status openvswitch-switch --no-pager -l

# Create VM template download script
tee /opt/gcp/bin/download-templates.sh > /dev/null << 'EOF'
#!/bin/bash

# Download Ubuntu Cloud Image
echo "Downloading Ubuntu 22.04 cloud image..."
cd /var/lib/gcp/templates/ubuntu
wget -O ubuntu-22.04-server-cloudimg-amd64.img \
  https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img

# Resize image (10GB)
sudo qemu-img resize ubuntu-22.04-server-cloudimg-amd64.img 10G

echo "VM template download completed!"
EOF

chmod +x /opt/gcp/bin/download-templates.sh

# Create resource monitoring script
tee /opt/gcp/bin/monitor-resources.sh > /dev/null << 'EOF'
#!/bin/bash

# System resource monitoring
echo "=== System Resource Monitoring ==="
echo "CPU Usage:"
top -bn1 | grep "Cpu(s)" | awk '{print $2 $3}' | sed 's/%us,/% user,/' | sed 's/%sy/% system/'

echo ""
echo "Memory Usage:"
free -h | grep -E "Mem|Swap"

echo ""
echo "Disk Usage:"
df -h | grep -E "/$|/var"

echo ""
echo "Running VMs:"
sudo virsh list --all

echo ""
echo "Running Containers:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "Open vSwitch Bridge Status:"
sudo ovs-vsctl show
EOF

chmod +x /opt/gcp/bin/monitor-resources.sh

echo "=========================================="
echo "Worker Node setup completed successfully!"
echo "=========================================="
echo "Please verify the following information:"
echo "- Worker Node IP: $WORKER_NODE_IP"
echo "- Node Name: $NODE_NAME"
echo "- Control Plane: $CONTROL_PLANE_IP"
echo "- Hypervisor Agent Port: 9090"
echo "- Network Agent Port: 9091"
echo "- Container Agent Port: 9092"
echo "- Open vSwitch Bridge: br-worker"
echo ""
echo "Additional tasks:"
echo "1. Download VM templates: /opt/gcp/bin/download-templates.sh"
echo "2. Monitor resources: /opt/gcp/bin/monitor-resources.sh"
echo ""
echo "All services will start automatically after reboot."
echo "Please log out and log back in to apply group permissions."