# Gon Cloud Platform (GCP)

A lightweight cloud computing platform for managing virtual instances and networking infrastructure.

## Overview

GCP is a small-scale cloud platform built with Go and React that enables you to:

- Create and manage Virtual Private Clouds (VPCs)
- Launch virtual instances using KVM/QEMU or Docker containers
- Configure software-defined networking with Open vSwitch
- Set up security groups and firewall rules
- Manage everything through a web-based console

## Architecture

### System Components

```
Control Plane
├── API Server (Go/Gin)
├── Network Controller 
├── Instance Manager
├── Web Console (React)
├── PostgreSQL Database
└── RabbitMQ Message Broker

Worker Nodes
├── Hypervisor Agent (KVM/QEMU)
├── Network Agent (Open vSwitch)
├── Container Agent (Docker)
└── Storage Agent
```

### Network Topology

```
Internet
    │
    ▼
[Internet Gateway]
    │
    ▼
[Open vSwitch Bridge]
    │
    ├── [VPC-1] ── [Subnet-1] ── [Instance-1, Instance-2]
    ├── [VPC-2] ── [Subnet-2] ── [Instance-3, Instance-4]
    └── [VPC-N] ── [Subnet-N] ── [Instance-N]
```

## Tech Stack

- **Backend**: Go 1.21+, Gin, PostgreSQL, RabbitMQ
- **Virtualization**: KVM/QEMU, Docker, Open vSwitch
- **Frontend**: React 18, Redux Toolkit, Ant Design

## Quick Start

### Prerequisites
- Ubuntu 22.04 LTS (Least 3 servers)
- Hardware virtualization support
- Root/sudo access

### Installation

1. **Clone and setup environment**
```bash
git clone https://github.com/yourusername/gon-cloud-platform.git
cd gon-cloud-platform

# Install dependencies
sudo ./scripts/setup/install-deps.sh
sudo ./scripts/setup/setup-kvm.sh
sudo ./scripts/setup/setup-ovs.sh
sudo ./scripts/setup/init-database.sh
```

2. **Configure environment**
```bash
cp .env.example .env
# Edit .env with your settings
```

3. **Deploy Control Plane**
```bash
cd control-plane
make build && sudo make install

sudo systemctl enable --now gcp-api-server
sudo systemctl enable --now gcp-network-controller
sudo systemctl enable --now gcp-instance-manager
```

4. **Deploy Worker Nodes**
```bash
cd worker-node
make build && sudo make install

sudo systemctl enable --now gcp-hypervisor-agent
sudo systemctl enable --now gcp-network-agent
sudo systemctl enable --now gcp-container-agent
```

5. **Deploy Web Console**
```bash
cd web-console
npm install && npm run build
sudo cp -r dist/* /var/www/gcp-console/
```

## Development

### Build from Source
```bash
# Control Plane
cd control-plane
go mod download
make build

# Worker Node
cd worker-node
go mod download
make build

# Web Console
cd web-console
npm install
npm run dev
```

### Run Tests
```bash
make test                # Unit tests
make test-integration    # Integration tests
```

## Troubleshooting

**Check service status**
```bash
sudo systemctl status gcp-api-server
sudo journalctl -fu gcp-api-server
```

**Verify KVM support**
```bash
egrep -c '(vmx|svm)' /proc/cpuinfo
lsmod | grep kvm
```

**Check Open vSwitch**
```bash
sudo ovs-vsctl show
sudo systemctl status openvswitch-switch
```

## License

MIT License

Copyright © 2025 Gon Cloud Platform

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
