#!/bin/bash
#
# CDC BroadcastStream & Neo4j Aliasing Deployment Script
# ======================================================
# Deploys the zero-downtime terminology versioning system including:
# - Debezium CDC connector for PostgreSQL
# - Neo4j database alias setup
# - Terminology notification service
# - Flink job with BroadcastStream
#
# Prerequisites:
# - Docker and Docker Compose installed
# - Kafka Connect running at KAFKA_CONNECT_URL
# - PostgreSQL database accessible with CDC configuration
# - Neo4j Enterprise (for database aliasing) at NEO4J_URI
# - Redis for state management at REDIS_URL
#
# Usage:
#   ./deploy-terminology-cdc.sh [setup|connector|neo4j|notification|flink|all|status|cleanup]
#
# Author: CDC Integration Team
# Version: 1.0
# Since: 2025-12-03

set -e

# ═══════════════════════════════════════════════════════════════════════════════
# CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════════

# Kafka Connect
KAFKA_CONNECT_URL="${KAFKA_CONNECT_URL:-http://localhost:8083}"

# PostgreSQL CDC source
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-kb_user}"
PG_PASSWORD="${PG_PASSWORD:-kb_password}"
PG_DATABASE="${PG_DATABASE:-kb_terminology}"

# Neo4j
NEO4J_URI="${NEO4J_URI:-bolt://localhost:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${NEO4J_PASSWORD:-kb7password}"

# Redis
REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

# GraphDB
GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/../configs"
SQL_DIR="${SCRIPT_DIR}/../sql"
SERVICES_DIR="${SCRIPT_DIR}/../services"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ═══════════════════════════════════════════════════════════════════════════════
# HELPER FUNCTIONS
# ═══════════════════════════════════════════════════════════════════════════════

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=${3:-30}
    local attempt=1

    log_info "Waiting for ${name} at ${url}..."

    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "${url}" > /dev/null 2>&1; then
            log_success "${name} is ready!"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    log_error "${name} did not become ready after ${max_attempts} attempts"
    return 1
}

# ═══════════════════════════════════════════════════════════════════════════════
# SETUP FUNCTIONS
# ═══════════════════════════════════════════════════════════════════════════════

setup_postgresql_cdc() {
    log_info "Setting up PostgreSQL CDC schema..."

    # Apply CDC schema
    if [ -f "${SQL_DIR}/kb7-releases-schema.sql" ]; then
        PGPASSWORD="${PG_PASSWORD}" psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "${PG_DATABASE}" \
            -f "${SQL_DIR}/kb7-releases-schema.sql"
        log_success "CDC schema applied successfully"
    else
        log_warning "CDC schema file not found at ${SQL_DIR}/kb7-releases-schema.sql"
    fi

    # Enable logical replication
    log_info "Enabling PostgreSQL logical replication..."
    PGPASSWORD="${PG_PASSWORD}" psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "${PG_DATABASE}" << 'EOF'
-- Ensure wal_level is set to logical (requires restart if changed)
-- ALTER SYSTEM SET wal_level = 'logical';

-- Create replication slot if not exists
SELECT pg_create_logical_replication_slot('kb7_releases_cdc_slot', 'pgoutput')
WHERE NOT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = 'kb7_releases_cdc_slot');

-- Create publication if not exists
CREATE PUBLICATION kb7_releases_cdc_publication FOR TABLE public.kb_releases
WHERE NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'kb7_releases_cdc_publication');

-- Grant replication permissions
GRANT USAGE ON SCHEMA public TO kb_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_user;
ALTER USER kb_user WITH REPLICATION;
EOF

    log_success "PostgreSQL CDC setup complete"
}

deploy_debezium_connector() {
    log_info "Deploying Debezium CDC connector..."

    wait_for_service "${KAFKA_CONNECT_URL}" "Kafka Connect"

    # Substitute environment variables in connector config
    local config_file="${CONFIG_DIR}/kb7-terminology-releases-cdc.json"

    if [ ! -f "${config_file}" ]; then
        log_error "Connector config not found at ${config_file}"
        return 1
    fi

    # Create temporary config with substituted variables
    local temp_config=$(mktemp)
    envsubst < "${config_file}" > "${temp_config}"

    # Check if connector already exists
    local connector_name="kb7-terminology-releases-cdc"
    local existing=$(curl -s "${KAFKA_CONNECT_URL}/connectors/${connector_name}" | grep -c "name" || true)

    if [ "$existing" -gt 0 ]; then
        log_info "Updating existing connector ${connector_name}..."
        curl -s -X PUT "${KAFKA_CONNECT_URL}/connectors/${connector_name}/config" \
            -H "Content-Type: application/json" \
            -d "$(cat ${temp_config} | jq '.config')"
    else
        log_info "Creating new connector ${connector_name}..."
        curl -s -X POST "${KAFKA_CONNECT_URL}/connectors" \
            -H "Content-Type: application/json" \
            -d "@${temp_config}"
    fi

    rm -f "${temp_config}"

    # Wait for connector to be running
    sleep 5
    local status=$(curl -s "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" | jq -r '.connector.state')

    if [ "$status" == "RUNNING" ]; then
        log_success "Debezium connector deployed and running"
    else
        log_warning "Connector status: ${status}"
    fi
}

setup_neo4j_alias() {
    log_info "Setting up Neo4j database alias..."

    # Create production alias pointing to default database
    cypher-shell -u "${NEO4J_USER}" -p "${NEO4J_PASSWORD}" -a "${NEO4J_URI}" << 'EOF'
// Create initial database if not exists
CREATE DATABASE kb7_v1 IF NOT EXISTS;

// Wait for database to be online
CALL apoc.util.sleep(2000);

// Create production alias
CREATE ALIAS kb7_production IF NOT EXISTS FOR DATABASE kb7_v1;

// Verify setup
SHOW ALIASES;
EOF

    log_success "Neo4j database alias configured"
}

start_notification_service() {
    log_info "Starting Terminology Notification Service..."

    # Check if service is already running
    if pgrep -f "terminology_notification_service.py" > /dev/null; then
        log_warning "Notification service already running"
        return 0
    fi

    # Set environment variables
    export KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
    export REDIS_URL="${REDIS_URL}"
    export KB1_WEBHOOK_URL="${KB1_WEBHOOK_URL:-http://localhost:8081/webhooks/terminology-update}"
    export KB2_WEBHOOK_URL="${KB2_WEBHOOK_URL:-http://localhost:8086/webhooks/terminology-update}"
    export KB3_WEBHOOK_URL="${KB3_WEBHOOK_URL:-http://localhost:8087/webhooks/terminology-update}"
    export KB4_WEBHOOK_URL="${KB4_WEBHOOK_URL:-http://localhost:8088/webhooks/terminology-update}"
    export KB5_WEBHOOK_URL="${KB5_WEBHOOK_URL:-http://localhost:8089/webhooks/terminology-update}"
    export KB6_WEBHOOK_URL="${KB6_WEBHOOK_URL:-http://localhost:8091/webhooks/terminology-update}"
    export KB7_WEBHOOK_URL="${KB7_WEBHOOK_URL:-http://localhost:8092/webhooks/terminology-update}"

    # Start service in background
    cd "${SERVICES_DIR}"
    nohup python3 terminology_notification_service.py > /tmp/terminology_notification_service.log 2>&1 &

    sleep 2

    if pgrep -f "terminology_notification_service.py" > /dev/null; then
        log_success "Notification service started (PID: $(pgrep -f 'terminology_notification_service.py'))"
    else
        log_error "Failed to start notification service. Check /tmp/terminology_notification_service.log"
    fi
}

start_neo4j_sync_service() {
    log_info "Starting Neo4j Terminology Sync Service..."

    local sync_service_dir="${SCRIPT_DIR}/../../knowledge-base-services/kb-7-terminology/runtime-layer-MOVED-TO-SHARED/services"

    # Check if service is already running
    if pgrep -f "neo4j_sync_service.py" > /dev/null; then
        log_warning "Neo4j sync service already running"
        return 0
    fi

    # Set environment variables
    export NEO4J_URI="${NEO4J_URI}"
    export NEO4J_USER="${NEO4J_USER}"
    export NEO4J_PASSWORD="${NEO4J_PASSWORD}"
    export GRAPHDB_URL="${GRAPHDB_URL}"
    export REDIS_URL="${REDIS_URL}"

    if [ -f "${sync_service_dir}/neo4j_sync_service.py" ]; then
        cd "${sync_service_dir}"
        log_info "Checking Neo4j sync service status..."
        python3 neo4j_sync_service.py --status
    else
        log_warning "Neo4j sync service not found at ${sync_service_dir}"
    fi
}

deploy_flink_job() {
    log_info "Deploying Flink BroadcastStream job..."

    local flink_dir="${SCRIPT_DIR}/../../flink-processing"
    local jar_path="${flink_dir}/target/flink-ehr-intelligence-1.0.0.jar"

    if [ ! -f "${jar_path}" ]; then
        log_info "Building Flink job..."
        cd "${flink_dir}"
        mvn clean package -DskipTests
    fi

    # Check if Flink is running
    local flink_url="${FLINK_URL:-http://localhost:8081}"

    if ! curl -s -f "${flink_url}/overview" > /dev/null 2>&1; then
        log_warning "Flink JobManager not accessible at ${flink_url}"
        return 1
    fi

    # Upload JAR
    log_info "Uploading JAR to Flink..."
    local upload_response=$(curl -s -X POST "${flink_url}/jars/upload" \
        -H "Expect:" \
        -F "jarfile=@${jar_path}")

    local jar_id=$(echo "${upload_response}" | jq -r '.filename' | xargs basename)

    if [ -z "${jar_id}" ] || [ "${jar_id}" == "null" ]; then
        log_error "Failed to upload JAR"
        return 1
    fi

    log_info "JAR uploaded: ${jar_id}"

    # Submit terminology broadcast job
    log_info "Submitting Module_KB7_TerminologyBroadcast job..."
    curl -s -X POST "${flink_url}/jars/${jar_id}/run" \
        -H "Content-Type: application/json" \
        -d '{
            "entryClass": "com.cardiofit.flink.operators.Module_KB7_TerminologyBroadcast",
            "parallelism": 2
        }'

    log_success "Flink job submitted"
}

# ═══════════════════════════════════════════════════════════════════════════════
# STATUS AND MONITORING
# ═══════════════════════════════════════════════════════════════════════════════

show_status() {
    echo ""
    echo "═══════════════════════════════════════════════════════════════════"
    echo "  CDC BroadcastStream & Neo4j Aliasing System Status"
    echo "═══════════════════════════════════════════════════════════════════"
    echo ""

    # Kafka Connect status
    echo -e "${BLUE}Kafka Connect:${NC}"
    if curl -s -f "${KAFKA_CONNECT_URL}" > /dev/null 2>&1; then
        echo -e "  Status: ${GREEN}RUNNING${NC}"
        local connector_status=$(curl -s "${KAFKA_CONNECT_URL}/connectors/kb7-terminology-releases-cdc/status" 2>/dev/null | jq -r '.connector.state' 2>/dev/null || echo "NOT_FOUND")
        echo -e "  CDC Connector: ${connector_status}"
    else
        echo -e "  Status: ${RED}NOT RUNNING${NC}"
    fi
    echo ""

    # Neo4j status
    echo -e "${BLUE}Neo4j:${NC}"
    if cypher-shell -u "${NEO4J_USER}" -p "${NEO4J_PASSWORD}" -a "${NEO4J_URI}" "RETURN 1" > /dev/null 2>&1; then
        echo -e "  Status: ${GREEN}CONNECTED${NC}"
        local alias_target=$(cypher-shell -u "${NEO4J_USER}" -p "${NEO4J_PASSWORD}" -a "${NEO4J_URI}" \
            "SHOW ALIASES WHERE name = 'kb7_production' RETURN properties.database AS target" 2>/dev/null | tail -1)
        echo -e "  Production Alias Target: ${alias_target:-NOT_SET}"
    else
        echo -e "  Status: ${RED}NOT CONNECTED${NC}"
    fi
    echo ""

    # Notification Service status
    echo -e "${BLUE}Terminology Notification Service:${NC}"
    if pgrep -f "terminology_notification_service.py" > /dev/null; then
        echo -e "  Status: ${GREEN}RUNNING${NC}"
        echo -e "  PID: $(pgrep -f 'terminology_notification_service.py')"
    else
        echo -e "  Status: ${YELLOW}NOT RUNNING${NC}"
    fi
    echo ""

    # Flink status
    echo -e "${BLUE}Flink:${NC}"
    local flink_url="${FLINK_URL:-http://localhost:8081}"
    if curl -s -f "${flink_url}/overview" > /dev/null 2>&1; then
        echo -e "  Status: ${GREEN}RUNNING${NC}"
        local running_jobs=$(curl -s "${flink_url}/jobs/overview" | jq '[.jobs[] | select(.state == "RUNNING")] | length')
        echo -e "  Running Jobs: ${running_jobs}"
    else
        echo -e "  Status: ${RED}NOT RUNNING${NC}"
    fi
    echo ""

    # Redis status
    echo -e "${BLUE}Redis:${NC}"
    if redis-cli -u "${REDIS_URL}" ping > /dev/null 2>&1; then
        echo -e "  Status: ${GREEN}CONNECTED${NC}"
        local current_version=$(redis-cli -u "${REDIS_URL}" GET "neo4j:current_version" 2>/dev/null)
        echo -e "  Current Terminology Version: ${current_version:-NOT_SET}"
    else
        echo -e "  Status: ${RED}NOT CONNECTED${NC}"
    fi
    echo ""
    echo "═══════════════════════════════════════════════════════════════════"
}

cleanup() {
    log_warning "Cleaning up CDC BroadcastStream system..."

    # Stop notification service
    if pgrep -f "terminology_notification_service.py" > /dev/null; then
        log_info "Stopping notification service..."
        pkill -f "terminology_notification_service.py"
    fi

    # Delete Debezium connector
    log_info "Deleting CDC connector..."
    curl -s -X DELETE "${KAFKA_CONNECT_URL}/connectors/kb7-terminology-releases-cdc" 2>/dev/null || true

    # Clean up Redis keys
    log_info "Cleaning up Redis keys..."
    redis-cli -u "${REDIS_URL}" DEL "neo4j:current_version" 2>/dev/null || true
    redis-cli -u "${REDIS_URL}" DEL "terminology:current_version" 2>/dev/null || true

    log_success "Cleanup complete"
}

# ═══════════════════════════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════════════════════════

case "${1:-all}" in
    setup)
        setup_postgresql_cdc
        ;;
    connector)
        deploy_debezium_connector
        ;;
    neo4j)
        setup_neo4j_alias
        start_neo4j_sync_service
        ;;
    notification)
        start_notification_service
        ;;
    flink)
        deploy_flink_job
        ;;
    all)
        log_info "Deploying complete CDC BroadcastStream & Neo4j Aliasing system..."
        echo ""
        setup_postgresql_cdc
        echo ""
        deploy_debezium_connector
        echo ""
        setup_neo4j_alias
        echo ""
        start_notification_service
        echo ""
        start_neo4j_sync_service
        echo ""
        deploy_flink_job
        echo ""
        show_status
        ;;
    status)
        show_status
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 [setup|connector|neo4j|notification|flink|all|status|cleanup]"
        echo ""
        echo "Commands:"
        echo "  setup       - Setup PostgreSQL CDC schema and replication"
        echo "  connector   - Deploy Debezium CDC connector"
        echo "  neo4j       - Setup Neo4j database alias and sync service"
        echo "  notification - Start terminology notification service"
        echo "  flink       - Deploy Flink BroadcastStream job"
        echo "  all         - Deploy complete system (default)"
        echo "  status      - Show system status"
        echo "  cleanup     - Clean up all components"
        exit 1
        ;;
esac
