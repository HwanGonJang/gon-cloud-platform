#!/bin/bash

# Gon Cloud Platform (GCP) Common Utility Script
# Ubuntu 22.04 based

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration variables
CONTROL_PLANE_IP="121.36.55.1"
WORKER_NODE_1_IP="121.36.55.2"
WORKER_NODE_2_IP="121.36.55.3"
GCP_USER="gcp"
GCP_GROUP="gcp"
LOG_DIR="/var/log/gcp"
CONFIG_DIR="/etc/gcp"
DATA_DIR="/var/lib/gcp"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
    [ -d "$LOG_DIR" ] && echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_DIR/gcp-utils.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
    [ -d "$LOG_DIR" ] && echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_DIR/gcp-utils.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
    [ -d "$LOG_DIR" ] && echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_DIR/gcp-utils.log"
}

log_debug() {
    if [ "${GCP_DEBUG:-false}" = "true" ]; then
        echo -e "${BLUE}[DEBUG]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
        [ -d "$LOG_DIR" ] && echo "[DEBUG] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_DIR/gcp-utils.log"
    fi
}

log_success() {
    echo -e "${CYAN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
    [ -d "$LOG_DIR" ] && echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_DIR/gcp-utils.log"
}

# Progress indicator function
show_progress() {
    local current=$1
    local total=$2
    local task_name=$3
    local percentage=$((current * 100 / total))
    local progress_bar=""
    
    for ((i=0; i<percentage/2; i++)); do
        progress_bar+="█"
    done
    for ((i=percentage/2; i<50; i++)); do
        progress_bar+="░"
    done
    
    echo -ne "\r${BLUE}[PROGRESS]${NC} $task_name: [$progress_bar] ${percentage}% (${current}/${total})"
    if [ $current -eq $total ]; then
        echo ""
    fi
}

# Service status check function
check_service_status() {
    local service_name=$1
    local timeout=${2:-10}
    
    log_debug "Checking service status: $service_name"
    
    if timeout $timeout systemctl is-active --quiet $service_name; then
        log_info "✓ $service_name service is running"
        return 0
    else
        log_error "✗ $service_name service is not running"
        return 1
    fi
}

# Port check function
check_port() {
    local port=$1
    local service_name=$2
    local host=${3:-"localhost"}
    
    log_debug "Checking port: $service_name ($host:$port)"
    
    if netstat -tuln | grep -q ":$port "; then
        log_info "✓ $service_name port $port is open"
        return 0
    else
        log_error "✗ $service_name port $port is not open"
        return 1
    fi
}

# Network connectivity check function
check_connectivity() {
    local target_ip=$1
    local target_port=$2
    local service_name=$3
    local timeout=${4:-5}
    
    log_debug "Checking network connectivity: $service_name ($target_ip:$target_port)"
    
    if timeout $timeout bash -c "</dev/tcp/$target_ip/$target_port" 2>/dev/null; then
        log_info "✓ $service_name ($target_ip:$target_port) is reachable"
        return 0
    else
        log_error "✗ $service_name ($target_ip:$target_port) is not reachable"
        return 1
    fi
}

# HTTP health check function
check_http_health() {
    local url=$1
    local service_name=$2
    local expected_code=${3:-200}
    local timeout=${4:-10}
    
    log_debug "HTTP health check: $service_name ($url)"
    
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time $timeout "$url" 2>/dev/null)
    
    if [ "$response_code" = "$expected_code" ]; then
        log_info "✓ $service_name HTTP health check passed (response code: $response_code)"
        return 0
    else
        log_error "✗ $service_name HTTP health check failed (response code: $response_code, expected: $expected_code)"
        return 1
    fi
}

# System requirements check
check_system_requirements() {
    log_info "Checking system requirements..."
    
    local requirements_met=true
    
    # OS version check
    if grep -q "Ubuntu 22.04" /etc/os-release; then
        log_success "✓ Ubuntu 22.04 confirmed"
    else
        log_warn "Running on non-Ubuntu 22.04 system"
        requirements_met=false
    fi
    
    # Memory check (minimum 4GB recommended)
    local memory_gb=$(free -g | awk '/^Mem:/{print $2}')
    if [ $memory_gb -ge 4 ]; then
        log_success "✓ Memory: ${memory_gb}GB (sufficient)"
    else
        log_warn "Memory: ${memory_gb}GB (4GB or more recommended)"
        requirements_met=false
    fi
    
    # Disk space check (minimum 20GB recommended)
    local disk_gb=$(df / | awk 'NR==2{printf "%.0f", $4/1024/1024}')
    if [ $disk_gb -ge 20 ]; then
        log_success "✓ Disk space: ${disk_gb}GB available (sufficient)"
    else
        log_warn "Disk space: ${disk_gb}GB available (20GB or more recommended)"
        requirements_met=false
    fi
    
    # CPU cores check
    local cpu_cores=$(nproc)
    if [ $cpu_cores -ge 2 ]; then
        log_success "✓ CPU cores: ${cpu_cores} (sufficient)"
    else
        log_warn "CPU cores: ${cpu_cores} (2 or more recommended)"
        requirements_met=false
    fi
    
    # Virtualization support check
    if grep -q "vmx\|svm" /proc/cpuinfo; then
        log_success "✓ Hardware virtualization supported"
    else
        log_error "Hardware virtualization is not supported"
        requirements_met=false
    fi
    
    return $([ "$requirements_met" = true ] && echo 0 || echo 1)
}

# Control Plane health check
check_control_plane_health() {
    log_info "Starting Control Plane health check..."
    
    local all_healthy=true
    local check_count=0
    local total_checks=8
    
    # PostgreSQL check
    show_progress $((++check_count)) $total_checks "PostgreSQL check"
    if ! check_service_status postgresql; then
        all_healthy=false
    elif ! check_port 5432 "PostgreSQL"; then
        all_healthy=false
    elif ! sudo -u postgres psql -c "SELECT version();" > /dev/null 2>&1; then
        log_error "PostgreSQL database connection failed"
        all_healthy=false
    else
        log_success "✓ PostgreSQL working normally"
    fi
    
    # RabbitMQ check
    show_progress $((++check_count)) $total_checks "RabbitMQ check"
    if ! check_service_status rabbitmq-server; then
        all_healthy=false
    elif ! check_port 5672 "RabbitMQ"; then
        all_healthy=false
    elif ! rabbitmq-diagnostics -q ping > /dev/null 2>&1; then
        log_error "RabbitMQ connection failed"
        all_healthy=false
    else
        log_success "✓ RabbitMQ working normally"
    fi
    
    # etcd check
    show_progress $((++check_count)) $total_checks "etcd check"
    if ! check_service_status etcd; then
        all_healthy=false
    elif ! check_port 2379 "etcd"; then
        all_healthy=false
    elif ! etcdctl endpoint health > /dev/null 2>&1; then
        log_error "etcd health check failed"
        all_healthy=false
    else
        log_success "✓ etcd working normally"
    fi
    
    # Open vSwitch check
    show_progress $((++check_count)) $total_checks "Open vSwitch check"
    if ! check_service_status openvswitch-switch; then
        all_healthy=false
    elif ! ovs-vsctl show > /dev/null 2>&1; then
        log_error "Open vSwitch connection failed"
        all_healthy=false
    else
        log_success "✓ Open vSwitch working normally"
    fi
    
    # Bridge check
    show_progress $((++check_count)) $total_checks "Network bridge check"
    if sudo ovs-vsctl br-exists br-main; then
        log_success "✓ Open vSwitch br-main bridge exists"
    else
        log_error "Open vSwitch br-main bridge does not exist"
        all_healthy=false
    fi
    
    # GCP API server check
    show_progress $((++check_count)) $total_checks "API server check"
    if check_port 8080 "GCP API Server"; then
        if check_http_health "http://localhost:8080/health" "GCP API Server"; then
            log_success "✓ GCP API server working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    # Network Controller check
    show_progress $((++check_count)) $total_checks "Network Controller check"
    if check_port 8081 "Network Controller"; then
        if check_http_health "http://localhost:8081/health" "Network Controller"; then
            log_success "✓ Network Controller working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    # Instance Manager check
    show_progress $((++check_count)) $total_checks "Instance Manager check"
    if check_port 8082 "Instance Manager"; then
        if check_http_health "http://localhost:8082/health" "Instance Manager"; then
            log_success "✓ Instance Manager working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    if [ "$all_healthy" = true ]; then
        log_success "🎉 Control Plane is running normally!"
        return 0
    else
        log_error "❌ Control Plane has issues"
        return 1
    fi
}

# Worker Node health check
check_worker_node_health() {
    local control_plane_ip=${1:-$CONTROL_PLANE_IP}
    log_info "Starting Worker Node health check..."
    
    local all_healthy=true
    local check_count=0
    local total_checks=7
    
    # Docker check
    show_progress $((++check_count)) $total_checks "Docker check"
    if ! check_service_status docker; then
        all_healthy=false
    elif ! docker info > /dev/null 2>&1; then
        log_error "Cannot connect to Docker daemon"
        all_healthy=false
    else
        log_success "✓ Docker working normally"
    fi
    
    # KVM/Libvirt check
    show_progress $((++check_count)) $total_checks "KVM/Libvirt check"
    if ! check_service_status libvirtd; then
        all_healthy=false
    elif ! virsh list > /dev/null 2>&1; then
        log_error "Libvirt connection failed"
        all_healthy=false
    else
        log_success "✓ KVM/Libvirt working normally"
    fi
    
    # QEMU check
    show_progress $((++check_count)) $total_checks "QEMU check"
    if ! command -v qemu-system-x86_64 > /dev/null 2>&1; then
        log_error "QEMU is not installed"
        all_healthy=false
    else
        log_success "✓ QEMU installed"
    fi
    
    # Control Plane connectivity check
    show_progress $((++check_count)) $total_checks "Control Plane connectivity check"
    if ! check_connectivity $control_plane_ip 8080 "Control Plane API"; then
        all_healthy=false
    else
        log_success "✓ Control Plane is reachable"
    fi
    
    # Hypervisor Agent check
    show_progress $((++check_count)) $total_checks "Hypervisor Agent check"
    if check_port 9090 "Hypervisor Agent"; then
        if check_http_health "http://localhost:9090/health" "Hypervisor Agent"; then
            log_success "✓ Hypervisor Agent working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    # Network Agent check
    show_progress $((++check_count)) $total_checks "Network Agent check"
    if check_port 9091 "Network Agent"; then
        if check_http_health "http://localhost:9091/health" "Network Agent"; then
            log_success "✓ Network Agent working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    # Container Agent check
    show_progress $((++check_count)) $total_checks "Container Agent check"
    if check_port 9092 "Container Agent"; then
        if check_http_health "http://localhost:9092/health" "Container Agent"; then
            log_success "✓ Container Agent working normally"
        else
            all_healthy=false
        fi
    else
        all_healthy=false
    fi
    
    if [ "$all_healthy" = true ]; then
        log_success "🎉 Worker Node is running normally!"
        return 0
    else
        log_error "❌ Worker Node has issues"
        return 1
    fi
}

# Full cluster health check
check_cluster_health() {
    log_info "Starting full cluster health check..."
    
    local current_ip=$(hostname -I | awk '{print $1}')
    local is_control_plane=false
    local is_worker_node=false
    
    # Determine current node type
    if [ "$current_ip" = "$CONTROL_PLANE_IP" ]; then
        is_control_plane=true
        log_info "Current node: Control Plane ($current_ip)"
    elif [ "$current_ip" = "$WORKER_NODE_1_IP" ] || [ "$current_ip" = "$WORKER_NODE_2_IP" ]; then
        is_worker_node=true
        log_info "Current node: Worker Node ($current_ip)"
    else
        log_warn "Unknown node IP: $current_ip"
    fi
    
    local overall_health=true
    
    # System requirements check
    if ! check_system_requirements; then
        overall_health=false
    fi
    
    # Node-specific health checks
    if [ "$is_control_plane" = true ]; then
        if ! check_control_plane_health; then
            overall_health=false
        fi
    elif [ "$is_worker_node" = true ]; then
        if ! check_worker_node_health; then
            overall_health=false
        fi
    fi
    
    # Cluster inter-node connectivity check
    log_info "Checking cluster inter-node connectivity..."
    
    if [ "$current_ip" != "$CONTROL_PLANE_IP" ]; then
        if ! check_connectivity $CONTROL_PLANE_IP 8080 "Control Plane"; then
            overall_health=false
        fi
    fi
    
    if [ "$current_ip" != "$WORKER_NODE_1_IP" ]; then
        if ! check_connectivity $WORKER_NODE_1_IP 9090 "Worker Node 1"; then
            log_warn "Worker Node 1 not reachable (may be normal)"
        fi
    fi
    
    if [ "$current_ip" != "$WORKER_NODE_2_IP" ]; then
        if ! check_connectivity $WORKER_NODE_2_IP 9090 "Worker Node 2"; then
            log_warn "Worker Node 2 not reachable (may be normal)"
        fi
    fi
    
    if [ "$overall_health" = true ]; then
        log_success "🎉 Full cluster is running normally!"
        return 0
    else
        log_error "❌ Cluster has issues"
        return 1
    fi
}

# Directory setup function
setup_directories() {
    log_info "Setting up GCP directory structure..."
    
    local directories=("$LOG_DIR" "$CONFIG_DIR" "$DATA_DIR" "$DATA_DIR/instances" "$DATA_DIR/volumes" "$DATA_DIR/networks")
    
    for dir in "${directories[@]}"; do
        if [ ! -d "$dir" ]; then
            sudo mkdir -p "$dir"
            sudo chown -R $GCP_USER:$GCP_GROUP "$dir" 2>/dev/null || true
            log_info "✓ Directory created: $dir"
        else
            log_debug "Directory already exists: $dir"
        fi
    done
    
    # Log rotation setup
    sudo tee /etc/logrotate.d/gcp > /dev/null << EOF
$LOG_DIR/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    copytruncate
    create 0644 $GCP_USER $GCP_GROUP
}
EOF
    
    log_success "✓ GCP directory structure setup completed"
}

# Environment cleanup function
cleanup_environment() {
    log_info "Starting environment cleanup..."
    
    # Clean temporary files
    find /tmp -name "gcp-*" -type f -mtime +1 -delete 2>/dev/null || true
    
    # Clean log files (older than 30 days)
    find "$LOG_DIR" -name "*.log" -type f -mtime +30 -delete 2>/dev/null || true
    
    # Docker cleanup
    if command -v docker > /dev/null 2>&1; then
        docker system prune -f > /dev/null 2>&1 || true
        log_info "✓ Docker system cleanup completed"
    fi
    
    # Libvirt cleanup
    if command -v virsh > /dev/null 2>&1; then
        # Clean shutdown instances
        virsh list --all --name | grep -E "^gcp-instance-" | while read vm; do
            if [ -n "$vm" ] && virsh domstate "$vm" 2>/dev/null | grep -q "shut off"; then
                virsh undefine "$vm" > /dev/null 2>&1 || true
            fi
        done
        log_info "✓ Libvirt cleanup completed"
    fi
    
    log_success "✓ Environment cleanup completed"
}

# Monitoring information collection
collect_monitoring_info() {
    log_info "Collecting system monitoring information..."
    
    local output_file="${LOG_DIR}/monitoring-$(date +%Y%m%d-%H%M%S).json"
    
    # Collect monitoring information in JSON format
    cat > "$output_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "hostname": "$(hostname)",
  "ip_address": "$(hostname -I | awk '{print $1}')",
  "system": {
    "os": "$(lsb_release -d | cut -f2)",
    "kernel": "$(uname -r)",
    "uptime": "$(uptime -p)",
    "load_average": "$(uptime | awk -F'load average:' '{print $2}')"
  },
  "resources": {
    "cpu_cores": $(nproc),
    "memory_total_gb": $(free -g | awk '/^Mem:/{print $2}'),
    "memory_used_gb": $(free -g | awk '/^Mem:/{print $3}'),
    "disk_usage": "$(df -h / | awk 'NR==2{print $5}')",
    "disk_available_gb": $(df / | awk 'NR==2{printf "%.0f", $4/1024/1024}')
  },
  "services": {
EOF

    # Add service status information
    local services=("postgresql" "rabbitmq-server" "etcd" "docker" "libvirtd" "openvswitch-switch")
    local service_count=0
    
    for service in "${services[@]}"; do
        if [ $service_count -gt 0 ]; then
            echo "," >> "$output_file"
        fi
        local status="inactive"
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            status="active"
        fi
        echo -n "    \"$service\": \"$status\"" >> "$output_file"
        ((service_count++))
    done
    
    cat >> "$output_file" << EOF

  },
  "network": {
    "interfaces": [
EOF

    # Network interface information
    ip -json addr show | jq -c '.[] | select(.flags | contains(["UP"])) | {name: .ifname, ip: [.addr_info[] | select(.family == "inet") | .local]}' | while IFS= read -r line; do
        echo "      $line," >> "$output_file"
    done
    
    # Remove last comma
    sed -i '$ s/,$//' "$output_file"
    
    cat >> "$output_file" << EOF
    ]
  }
}
EOF
    
    log_success "✓ Monitoring information collection completed: $output_file"
    echo "$output_file"
}

# Backup function
backup_configuration() {
    local backup_dir="${DATA_DIR}/backups/$(date +%Y%m%d-%H%M%S)"
    log_info "Starting configuration backup: $backup_dir"
    
    mkdir -p "$backup_dir"
    
    # Configuration files backup
    if [ -d "$CONFIG_DIR" ]; then
        cp -r "$CONFIG_DIR" "$backup_dir/"
        log_info "✓ Configuration files backup completed"
    fi
    
    # Database backup (Control Plane only)
    if systemctl is-active --quiet postgresql 2>/dev/null; then
        sudo -u postgres pg_dumpall > "$backup_dir/database.sql"
        log_info "✓ Database backup completed"
    fi
    
    # etcd backup (Control Plane only)
    if systemctl is-active --quiet etcd 2>/dev/null; then
        etcdctl snapshot save "$backup_dir/etcd-snapshot.db" 2>/dev/null || true
        log_info "✓ etcd backup completed"
    fi
    
    # Compression
    tar -czf "${backup_dir}.tar.gz" -C "$(dirname "$backup_dir")" "$(basename "$backup_dir")"
    rm -rf "$backup_dir"
    
    log_success "✓ Backup completed: ${backup_dir}.tar.gz"
    echo "${backup_dir}.tar.gz"
}

# Main function
main() {
    local command=${1:-"health"}
    
    case "$command" in
        "health"|"healthcheck")
            check_cluster_health
            ;;
        "control-plane")
            check_control_plane_health
            ;;
        "worker-node")
            check_worker_node_health "$2"
            ;;
        "system")
            check_system_requirements
            ;;
        "setup")
            setup_directories
            ;;
        "monitor")
            collect_monitoring_info
            ;;
        "backup")
            backup_configuration
            ;;
        "cleanup")
            cleanup_environment
            ;;
        "help"|"-h"|"--help")
            echo "GCP Platform Common Utility Tool"
            echo ""
            echo "Usage: $0 [command] [options]"
            echo ""
            echo "Commands:"
            echo "  health          Full cluster health check (default)"
            echo "  control-plane   Control Plane health check"
            echo "  worker-node     Worker Node health check"
            echo "  system          System requirements check"
            echo "  setup           Directory structure setup"
            echo "  monitor         Monitoring information collection"
            echo "  backup          Configuration backup"
            echo "  cleanup         Environment cleanup"
            echo "  help            Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  GCP_DEBUG=true  Enable debug mode"
            echo ""
            exit 0
            ;;
        *)
            log_error "Unknown command: $command"
            log_info "Run '$0 help' to see available commands"
            exit 1
            ;;
    esac
}

# Call main function only when script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi