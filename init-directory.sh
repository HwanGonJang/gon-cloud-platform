#!/bin/bash

# Create project root directory
mkdir -p gon-cloud-platform
cd gon-cloud-platform

# Create basic files
touch README.md Makefile docker-compose.yml .gitignore .env.example

# Documentation directories
mkdir -p docs/{api,architecture,deployment}

# Script directories
mkdir -p scripts/{setup,deploy,monitoring}

# Configuration directories
mkdir -p configs/{control-plane,worker-node,database,networking}

# Control Plane structure
mkdir -p control-plane/cmd/{api-server,network-controller,instance-manager}
mkdir -p control-plane/internal/{api,models,database,services,network,messaging,utils}
mkdir -p control-plane/internal/api/{handlers,middleware,routes}
mkdir -p control-plane/internal/database/{migrations,repositories}
mkdir -p control-plane/pkg/{errors,response,constants}
mkdir -p control-plane/migrations

# Worker Node structure
mkdir -p worker-node/cmd/{hypervisor-agent,network-agent,container-agent}
mkdir -p worker-node/internal/{hypervisor,container,network,storage,agent}

# Web Console structure
mkdir -p web-console/{public,src}
mkdir -p web-console/src/{components,pages,store,services,utils,styles}
mkdir -p web-console/src/components/{common,vpc,instance,security}
mkdir -p web-console/src/store/{slices,middleware}
mkdir -p web-console/src/styles/components

# Shared libraries
mkdir -p shared/{proto,schemas}

# Test directories
mkdir -p tests/{unit,integration,e2e}

# Deployment directories
mkdir -p deployments/{systemd,kubernetes,terraform}

# Monitoring directories
mkdir -p monitoring/{prometheus,grafana,elk}

echo "GCP project directory structure has been created successfully!"