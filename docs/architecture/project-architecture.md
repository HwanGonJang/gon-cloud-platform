gon-cloud-platform/
├── README.md
├── Makefile
├── docker-compose.yml
├── .gitignore
├── .env.example
│
├── docs/                           # Documentation
│   ├── api/                        # API documentation
│   ├── architecture/               # Architecture documentation
│   └── deployment/                 # Deployment guides
│
├── scripts/                        # Scripts
│   ├── setup/                      # Environment setup scripts
│   │   ├── install-deps.sh
│   │   ├── setup-kvm.sh
│   │   ├── setup-ovs.sh
│   │   └── init-database.sh
│   ├── deploy/                     # Deployment scripts
│   └── monitoring/                 # Monitoring scripts
│
├── configs/                        # Configuration files
│   ├── control-plane/
│   ├── worker-node/
│   ├── database/
│   └── networking/
│
├── control-plane/                  # Control Plane services
│   ├── cmd/                        # Executable files
│   │   ├── api-server/
│   │   │   └── main.go
│   │   ├── network-controller/
│   │   │   └── main.go
│   │   └── instance-manager/
│   │       └── main.go
│   │
│   ├── internal/                   # Internal packages
│   │   ├── api/                    # API handlers
│   │   │   ├── handlers/
│   │   │   │   ├── vpc.go
│   │   │   │   ├── subnet.go
│   │   │   │   ├── instance.go
│   │   │   │   ├── security_group.go
│   │   │   │   └── auth.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── cors.go
│   │   │   │   └── logging.go
│   │   │   └── routes/
│   │   │       └── router.go
│   │   │
│   │   ├── models/                 # Data models
│   │   │   ├── vpc.go
│   │   │   ├── subnet.go
│   │   │   ├── instance.go
│   │   │   ├── security_group.go
│   │   │   └── user.go
│   │   │
│   │   ├── database/               # Database related
│   │   │   ├── connection.go
│   │   │   ├── migrations/
│   │   │   └── repositories/
│   │   │       ├── vpc_repo.go
│   │   │       ├── subnet_repo.go
│   │   │       └── instance_repo.go
│   │   │
│   │   ├── services/               # Business logic
│   │   │   ├── vpc_service.go
│   │   │   ├── subnet_service.go
│   │   │   ├── instance_service.go
│   │   │   └── security_service.go
│   │   │
│   │   ├── network/                # Network management
│   │   │   ├── ovs_manager.go
│   │   │   ├── ip_allocator.go
│   │   │   ├── routing_table.go
│   │   │   └── firewall.go
│   │   │
│   │   ├── messaging/              # Message queue
│   │   │   ├── rabbitmq.go
│   │   │   ├── publisher.go
│   │   │   └── consumer.go
│   │   │
│   │   └── utils/                  # Utilities
│   │       ├── config.go
│   │       ├── logger.go
│   │       └── validator.go
│   │
│   ├── pkg/                        # Public packages
│   │   ├── errors/
│   │   ├── response/
│   │   └── constants/
│   │
│   ├── migrations/                 # DB migrations
│   │   ├── 001_create_vpcs.sql
│   │   ├── 002_create_subnets.sql
│   │   ├── 003_create_instances.sql
│   │   └── 004_create_security_groups.sql
│   │
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── worker-node/                    # Worker Node agents
│   ├── cmd/                        # Executable files
│   │   ├── hypervisor-agent/
│   │   │   └── main.go
│   │   ├── network-agent/
│   │   │   └── main.go
│   │   └── container-agent/
│   │       └── main.go
│   │
│   ├── internal/                   # Internal packages
│   │   ├── hypervisor/             # KVM/QEMU management
│   │   │   ├── kvm_manager.go
│   │   │   ├── qemu_manager.go
│   │   │   └── resource_monitor.go
│   │   │
│   │   ├── container/              # Docker management
│   │   │   ├── docker_manager.go
│   │   │   ├── image_manager.go
│   │   │   └── volume_manager.go
│   │   │
│   │   ├── network/                # Network agent
│   │   │   ├── interface_manager.go
│   │   │   ├── iptables_manager.go
│   │   │   └── traffic_monitor.go
│   │   │
│   │   ├── storage/                # Storage management
│   │   │   ├── lvm_manager.go
│   │   │   └── snapshot_manager.go
│   │   │
│   │   └── agent/                  # Agent common
│   │       ├── client.go
│   │       ├── heartbeat.go
│   │       └── metrics.go
│   │
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── web-console/                    # React web console
│   ├── public/
│   │   ├── index.html
│   │   └── favicon.ico
│   │
│   ├── src/
│   │   ├── components/             # React components
│   │   │   ├── common/
│   │   │   │   ├── Header.jsx
│   │   │   │   ├── Sidebar.jsx
│   │   │   │   └── Loading.jsx
│   │   │   │
│   │   │   ├── vpc/
│   │   │   │   ├── VPCList.jsx
│   │   │   │   ├── VPCCreate.jsx
│   │   │   │   └── VPCDetail.jsx
│   │   │   │
│   │   │   ├── instance/
│   │   │   │   ├── InstanceList.jsx
│   │   │   │   ├── InstanceCreate.jsx
│   │   │   │   └── InstanceDetail.jsx
│   │   │   │
│   │   │   └── security/
│   │   │       ├── SecurityGroupList.jsx
│   │   │       └── SecurityGroupCreate.jsx
│   │   │
│   │   ├── pages/                  # Page components
│   │   │   ├── Dashboard.jsx
│   │   │   ├── VPC.jsx
│   │   │   ├── Instances.jsx
│   │   │   └── Security.jsx
│   │   │
│   │   ├── store/                  # Redux store
│   │   │   ├── index.js
│   │   │   ├── slices/
│   │   │   │   ├── vpcSlice.js
│   │   │   │   ├── instanceSlice.js
│   │   │   │   └── authSlice.js
│   │   │   └── middleware/
│   │   │
│   │   ├── services/               # API services
│   │   │   ├── api.js
│   │   │   ├── vpcService.js
│   │   │   ├── instanceService.js
│   │   │   └── authService.js
│   │   │
│   │   ├── utils/
│   │   │   ├── constants.js
│   │   │   └── helpers.js
│   │   │
│   │   ├── styles/
│   │   │   ├── global.css
│   │   │   └── components/
│   │   │
│   │   ├── App.jsx
│   │   └── main.jsx
│   │
│   ├── package.json
│   ├── vite.config.js
│   ├── Dockerfile
│   └── .env.example
│
├── shared/                         # Shared libraries
│   ├── proto/                      # Protocol Buffers
│   │   ├── vpc.proto
│   │   ├── instance.proto
│   │   └── network.proto
│   │
│   └── schemas/                    # JSON schemas
│       ├── vpc.json
│       ├── instance.json
│       └── security_group.json
│
├── tests/                          # Tests
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── deployments/                    # Deployment related
│   ├── systemd/                    # systemd service files
│   │   ├── gcp-api-server.service
│   │   ├── gcp-network-controller.service
│   │   ├── gcp-instance-manager.service
│   │   ├── gcp-hypervisor-agent.service
│   │   ├── gcp-network-agent.service
│   │   └── gcp-container-agent.service
│   │
│   ├── kubernetes/                 # K8s manifests (optional)
│   └── terraform/                  # Infrastructure as code (optional)
│
└── monitoring/                     # Monitoring configuration
    ├── prometheus/
    ├── grafana/
    └── elk/