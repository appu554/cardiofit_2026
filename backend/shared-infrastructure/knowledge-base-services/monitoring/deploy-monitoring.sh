#!/bin/bash

# KB Services Unified Monitoring Deployment Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONITORING_DIR="$SCRIPT_DIR"
LOG_FILE="$MONITORING_DIR/deploy.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${2:-$NC}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

# Error handling
handle_error() {
    log "❌ Error occurred in deployment script at line $1" "$RED"
    log "❌ Command: $2" "$RED"
    exit 1
}

trap 'handle_error $LINENO "$BASH_COMMAND"' ERR

# Banner
echo -e "${BLUE}"
cat << 'EOF'
╔══════════════════════════════════════════════════════════════════════════════╗
║                    KB Services Unified Monitoring                            ║
║                           Deployment Script                                  ║
╚══════════════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

log "🚀 Starting KB Services monitoring deployment" "$BLUE"
log "📁 Working directory: $MONITORING_DIR" "$BLUE"

# Check prerequisites
log "🔍 Checking prerequisites..." "$YELLOW"

# Check Docker
if ! command -v docker &> /dev/null; then
    log "❌ Docker is not installed or not in PATH" "$RED"
    exit 1
fi

# Check Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    log "❌ Docker Compose is not installed" "$RED"
    exit 1
fi

# Check if files exist
required_files=(
    "docker-compose.monitoring.yml"
    "prometheus-unified.yml" 
    "alert_rules.yml"
    "recording_rules.yml"
    "alertmanager.yml"
    "unified-kb-dashboard.json"
)

for file in "${required_files[@]}"; do
    if [[ ! -f "$MONITORING_DIR/$file" ]]; then
        log "❌ Required file missing: $file" "$RED"
        exit 1
    fi
done

log "✅ All prerequisite checks passed" "$GREEN"

# Create necessary directories
log "📁 Creating necessary directories..." "$YELLOW"

mkdir -p "$MONITORING_DIR/grafana/provisioning/dashboards"
mkdir -p "$MONITORING_DIR/grafana/provisioning/datasources" 
mkdir -p "$MONITORING_DIR/grafana/provisioning/alerting"
mkdir -p "$MONITORING_DIR/templates"
mkdir -p "$MONITORING_DIR/data"

# Create Grafana provisioning configurations
log "⚙️  Creating Grafana provisioning configurations..." "$YELLOW"

# Datasource configuration
cat > "$MONITORING_DIR/grafana/provisioning/datasources/datasource.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
    basicAuth: false
    
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    editable: true
    
  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
    editable: true
    
  - name: VictoriaMetrics
    type: prometheus
    access: proxy
    url: http://victoriametrics:8428
    editable: true
EOF

# Dashboard provisioning
cat > "$MONITORING_DIR/grafana/provisioning/dashboards/dashboard.yml" << 'EOF'
apiVersion: 1

providers:
  - name: 'KB Services Dashboards'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
      foldersFromFilesStructure: true
EOF

# Copy dashboard to provisioning directory
cp "$MONITORING_DIR/unified-kb-dashboard.json" "$MONITORING_DIR/grafana/provisioning/dashboards/"

# Create Loki configuration
log "📝 Creating Loki configuration..." "$YELLOW"

cat > "$MONITORING_DIR/loki-config.yaml" << 'EOF'
auth_enabled: false

server:
  http_listen_port: 3100

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory

query_scheduler:
  max_outstanding_requests_per_tenant: 32768

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

ruler:
  alertmanager_url: http://alertmanager:9093

limits_config:
  reject_old_samples: true
  reject_old_samples_max_age: 168h
  ingestion_rate_mb: 16
  ingestion_burst_size_mb: 32
  max_query_parallelism: 100
EOF

# Create Promtail configuration
cat > "$MONITORING_DIR/promtail-config.yml" << 'EOF'
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: kb-services-logs
    static_configs:
      - targets:
          - localhost
        labels:
          job: kb-services
          __path__: /var/log/**/*.log

  - job_name: system-logs
    static_configs:
      - targets:
          - localhost
        labels:
          job: system
          __path__: /var/log/{messages,secure,cron,maillog}

  - job_name: docker-logs
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: ['__meta_docker_container_label_monitoring_component']
        target_label: 'component'
      - source_labels: ['__meta_docker_container_name']
        target_label: 'container'
EOF

# Set permissions
log "🔐 Setting appropriate permissions..." "$YELLOW"
chmod -R 755 "$MONITORING_DIR/grafana"
chmod 644 "$MONITORING_DIR"/*.yml "$MONITORING_DIR"/*.yaml

# Pull Docker images
log "📥 Pulling Docker images..." "$YELLOW"
docker-compose -f "$MONITORING_DIR/docker-compose.monitoring.yml" pull

# Start monitoring stack
log "🚀 Starting monitoring stack..." "$YELLOW"
docker-compose -f "$MONITORING_DIR/docker-compose.monitoring.yml" up -d

# Wait for services to be ready
log "⏳ Waiting for services to be ready..." "$YELLOW"

services=("prometheus:9090" "grafana:3000" "alertmanager:9093" "loki:3100" "jaeger:16686")
max_wait=300  # 5 minutes
wait_time=0

for service in "${services[@]}"; do
    IFS=':' read -r service_name port <<< "$service"
    log "   Waiting for $service_name..." "$YELLOW"
    
    while ! docker exec kb-$service_name wget --no-verbose --tries=1 --spider http://localhost:$port/health 2>/dev/null && docker exec kb-$service_name wget --no-verbose --tries=1 --spider http://localhost:$port/ 2>/dev/null; do
        if [[ $wait_time -ge $max_wait ]]; then
            log "❌ Timeout waiting for $service_name to be ready" "$RED"
            docker-compose -f "$MONITORING_DIR/docker-compose.monitoring.yml" logs $service_name
            exit 1
        fi
        sleep 5
        wait_time=$((wait_time + 5))
    done
    
    log "   ✅ $service_name is ready" "$GREEN"
done

# Import Grafana dashboard
log "📊 Importing Grafana dashboard..." "$YELLOW"
sleep 10  # Give Grafana time to fully initialize

# Create dashboard import script
cat > "$MONITORING_DIR/import-dashboard.sh" << 'EOF'
#!/bin/bash
curl -X POST \
  http://admin:kb-admin-2024@localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d @/etc/grafana/provisioning/dashboards/unified-kb-dashboard.json
EOF

chmod +x "$MONITORING_DIR/import-dashboard.sh"
docker exec kb-grafana /bin/sh -c "cd /etc/grafana/provisioning/dashboards && curl -X POST http://admin:kb-admin-2024@localhost:3000/api/dashboards/db -H 'Content-Type: application/json' -d @unified-kb-dashboard.json" || true

# Health check
log "🏥 Performing health checks..." "$YELLOW"

health_urls=(
    "http://localhost:9090/-/healthy:Prometheus"
    "http://localhost:3000/api/health:Grafana" 
    "http://localhost:9093/-/healthy:Alertmanager"
    "http://localhost:3100/ready:Loki"
    "http://localhost:16686/:Jaeger"
    "http://localhost:8428/health:VictoriaMetrics"
)

for url_service in "${health_urls[@]}"; do
    IFS=':' read -r url service_name <<< "$url_service"
    if curl -f -s "$url" > /dev/null; then
        log "   ✅ $service_name health check passed" "$GREEN"
    else
        log "   ⚠️  $service_name health check failed" "$YELLOW"
    fi
done

# Display access information
log "✅ Deployment completed successfully!" "$GREEN"
echo
log "📊 Access Information:" "$BLUE"
log "   Grafana Dashboard: http://localhost:3000" "$BLUE"
log "   Username: admin, Password: kb-admin-2024" "$BLUE"
log "   Prometheus: http://localhost:9090" "$BLUE"  
log "   Alertmanager: http://localhost:9093" "$BLUE"
log "   Jaeger Tracing: http://localhost:16686" "$BLUE"
log "   Loki Logs: http://localhost:3100" "$BLUE"
log "   VictoriaMetrics: http://localhost:8428" "$BLUE"
echo
log "📚 Key Dashboards:" "$BLUE"
log "   • Unified KB Dashboard: http://localhost:3000/d/kb-unified" "$BLUE"
log "   • Service Health: http://localhost:3000/d/kb-health" "$BLUE"
log "   • Database Performance: http://localhost:3000/d/kb-databases" "$BLUE"
echo
log "🔧 Management Commands:" "$BLUE"
log "   • View logs: docker-compose -f docker-compose.monitoring.yml logs -f" "$BLUE"
log "   • Stop stack: docker-compose -f docker-compose.monitoring.yml down" "$BLUE"
log "   • Restart: docker-compose -f docker-compose.monitoring.yml restart" "$BLUE"
echo
log "📝 Logs saved to: $LOG_FILE" "$BLUE"

# Create management script
cat > "$MONITORING_DIR/manage-monitoring.sh" << 'EOF'
#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.monitoring.yml"

case "$1" in
    start)
        echo "🚀 Starting KB Services monitoring stack..."
        docker-compose -f "$COMPOSE_FILE" up -d
        ;;
    stop)
        echo "🛑 Stopping KB Services monitoring stack..."
        docker-compose -f "$COMPOSE_FILE" down
        ;;
    restart)
        echo "🔄 Restarting KB Services monitoring stack..."
        docker-compose -f "$COMPOSE_FILE" restart
        ;;
    status)
        echo "📊 KB Services monitoring stack status:"
        docker-compose -f "$COMPOSE_FILE" ps
        ;;
    logs)
        echo "📝 KB Services monitoring logs:"
        docker-compose -f "$COMPOSE_FILE" logs -f "${@:2}"
        ;;
    health)
        echo "🏥 KB Services monitoring health checks:"
        services=("prometheus:9090" "grafana:3000" "alertmanager:9093")
        for service in "${services[@]}"; do
            IFS=':' read -r name port <<< "$service"
            if curl -f -s "http://localhost:$port/health" > /dev/null 2>&1 || curl -f -s "http://localhost:$port/" > /dev/null 2>&1; then
                echo "   ✅ $name is healthy"
            else
                echo "   ❌ $name is not responding"
            fi
        done
        ;;
    clean)
        echo "🧹 Cleaning up KB Services monitoring..."
        docker-compose -f "$COMPOSE_FILE" down -v
        docker system prune -f
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|health|clean}"
        echo
        echo "Commands:"
        echo "  start    - Start the monitoring stack"
        echo "  stop     - Stop the monitoring stack" 
        echo "  restart  - Restart the monitoring stack"
        echo "  status   - Show container status"
        echo "  logs     - Show logs (optionally specify service name)"
        echo "  health   - Check health of monitoring services"
        echo "  clean    - Stop and remove all containers and volumes"
        exit 1
        ;;
esac
EOF

chmod +x "$MONITORING_DIR/manage-monitoring.sh"

log "✨ Management script created: $MONITORING_DIR/manage-monitoring.sh" "$GREEN"
log "🎉 KB Services Unified Monitoring is now running!" "$GREEN"