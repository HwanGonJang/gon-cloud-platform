#!/bin/bash

# Initialize Control Plane Go module
cd control-plane
go mod init github.com/HwanGonJang/gon-cloud-platform/control-plane
touch cmd/api-server/main.go
touch cmd/network-controller/main.go
touch cmd/instance-manager/main.go
touch Dockerfile

# Initialize Worker Node Go module
cd ../worker-node
go mod init github.com/HwanGonJang/gon-cloud-platform/worker-node
touch cmd/hypervisor-agent/main.go
touch cmd/network-agent/main.go
touch cmd/container-agent/main.go
touch Dockerfile

# Initialize Web Console
cd ../web-console
touch package.json vite.config.js Dockerfile .env.example
touch public/index.html public/favicon.ico
touch src/App.jsx src/main.jsx

# Create basic setup scripts
cd ../scripts/setup
touch install-deps.sh setup-kvm.sh setup-ovs.sh init-database.sh

# Create systemd service files
cd ../../deployments/systemd
touch gcp-api-server.service
touch gcp-network-controller.service
touch gcp-instance-manager.service
touch gcp-hypervisor-agent.service
touch gcp-network-agent.service
touch gcp-container-agent.service

echo "Initial files have been created successfully!"