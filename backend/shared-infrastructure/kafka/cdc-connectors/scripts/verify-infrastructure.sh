#!/bin/bash
# verify-infrastructure.sh
# Comprehensive infrastructure health verification for CDC deployment
# Prerequisites validation before connector deployment

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
KAFKA_CONTAINER_ID="3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754"
KAFKA_NETWORK="cardiofit-network"
KAFKA_CONNECT_URL="${KAFKA_CONNECT_URL:-http://localhost:8083}"
POSTGRES_HOSTS=(
  "localhost:5432"
  "localhost:5433"
  "localhost:5434"
  "localhost:5435"
  "localhost:5436"
)

# Logging function
log() {
  echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
  echo -e "${RED}[ERROR]${NC} $*" >&2
}

warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

# Check if command exists
check_command() {
  if ! command -v "$1" &> /dev/null; then
    error "$1 is not installed or not in PATH"
    return 1
  fi
  log "$1 is available"
  return 0
}

# Verify required tools
verify_tools() {
  log "Verifying required tools..."
  local missing=0

  for cmd in docker psql curl jq; do
    if ! check_command "$cmd"; then
      ((missing++))
    fi
  done

  if [ $missing -gt 0 ]; then
    error "$missing required tool(s) missing"
    return 1
  fi

  log "All required tools are available"
  return 0
}

# Verify Kafka container health
verify_kafka_container() {
  log "Verifying Kafka container health..."

  # Check if container exists and is running
  if ! docker ps --filter "id=$KAFKA_CONTAINER_ID" --format '{{.Status}}' | grep -q "Up"; then
    error "Kafka container $KAFKA_CONTAINER_ID is not running"
    return 1
  fi

  log "Kafka container is running"

  # Check network connectivity
  if ! docker inspect "$KAFKA_CONTAINER_ID" | jq -e ".[0].NetworkSettings.Networks[\"$KAFKA_NETWORK\"]" > /dev/null; then
    error "Kafka container is not connected to $KAFKA_NETWORK"
    return 1
  fi

  log "Kafka container is connected to $KAFKA_NETWORK"

  # Verify Kafka broker is responsive
  if ! docker exec "$KAFKA_CONTAINER_ID" kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null; then
    error "Kafka broker is not responsive"
    return 1
  fi

  log "Kafka broker is responsive"

  return 0
}

# Verify Kafka Connect cluster
verify_kafka_connect() {
  log "Verifying Kafka Connect cluster..."

  # Check if Kafka Connect is reachable
  if ! curl -sf "${KAFKA_CONNECT_URL}/" > /dev/null; then
    error "Kafka Connect is not reachable at ${KAFKA_CONNECT_URL}"
    return 1
  fi

  log "Kafka Connect is reachable"

  # Get cluster information
  local connect_info
  connect_info=$(curl -sf "${KAFKA_CONNECT_URL}/")

  log "Kafka Connect version: $(echo "$connect_info" | jq -r '.version')"
  log "Kafka Connect commit: $(echo "$connect_info" | jq -r '.commit')"

  # Verify Debezium connector plugin is available
  local plugins
  plugins=$(curl -sf "${KAFKA_CONNECT_URL}/connector-plugins")

  if ! echo "$plugins" | jq -e '.[] | select(.class == "io.debezium.connector.postgresql.PostgresConnector")' > /dev/null; then
    error "Debezium PostgreSQL connector plugin is not available"
    return 1
  fi

  log "Debezium PostgreSQL connector plugin is available"

  # Check existing connectors
  local connectors
  connectors=$(curl -sf "${KAFKA_CONNECT_URL}/connectors")
  local connector_count
  connector_count=$(echo "$connectors" | jq 'length')

  log "Existing connectors: $connector_count"

  if [ "$connector_count" -gt 0 ]; then
    warn "Found existing connectors: $(echo "$connectors" | jq -r '.[]' | tr '\n' ' ')"
  fi

  return 0
}

# Verify PostgreSQL connectivity
verify_postgresql() {
  log "Verifying PostgreSQL connectivity..."
  local failed=0

  for pg_host in "${POSTGRES_HOSTS[@]}"; do
    local host="${pg_host%%:*}"
    local port="${pg_host##*:}"

    log "Testing connection to PostgreSQL at $host:$port..."

    # Test TCP connectivity
    if ! timeout 5 bash -c "cat < /dev/null > /dev/tcp/$host/$port" 2>/dev/null; then
      warn "Cannot connect to PostgreSQL at $host:$port (TCP check failed)"
      ((failed++))
      continue
    fi

    # Test PostgreSQL authentication (requires PGPASSWORD env var)
    if [ -n "${PGPASSWORD:-}" ]; then
      if ! PGPASSWORD="$PGPASSWORD" psql -h "$host" -p "$port" -U postgres -d postgres -c "SELECT version();" > /dev/null 2>&1; then
        warn "Cannot authenticate to PostgreSQL at $host:$port"
        ((failed++))
        continue
      fi

      log "Successfully connected to PostgreSQL at $host:$port"
    else
      warn "PGPASSWORD not set, skipping authentication test for $host:$port"
    fi
  done

  if [ $failed -eq ${#POSTGRES_HOSTS[@]} ]; then
    error "Failed to connect to any PostgreSQL instance"
    return 1
  fi

  log "PostgreSQL connectivity verified ($((${#POSTGRES_HOSTS[@]} - failed))/${#POSTGRES_HOSTS[@]} instances reachable)"
  return 0
}

# Verify Docker network
verify_network() {
  log "Verifying Docker network $KAFKA_NETWORK..."

  if ! docker network inspect "$KAFKA_NETWORK" > /dev/null 2>&1; then
    error "Docker network $KAFKA_NETWORK does not exist"
    return 1
  fi

  log "Docker network $KAFKA_NETWORK exists"

  # List containers on the network
  local network_containers
  network_containers=$(docker network inspect "$KAFKA_NETWORK" | jq -r '.[0].Containers | keys[]')
  local container_count
  container_count=$(echo "$network_containers" | wc -l)

  log "Containers on $KAFKA_NETWORK: $container_count"

  return 0
}

# Verify disk space
verify_disk_space() {
  log "Verifying disk space..."

  local required_mb=5000  # 5GB minimum
  local available_mb
  available_mb=$(df -m /var/lib/docker 2>/dev/null | awk 'NR==2 {print $4}')

  if [ -z "$available_mb" ]; then
    warn "Could not determine available disk space"
    return 0
  fi

  if [ "$available_mb" -lt "$required_mb" ]; then
    error "Insufficient disk space: ${available_mb}MB available, ${required_mb}MB required"
    return 1
  fi

  log "Sufficient disk space: ${available_mb}MB available"
  return 0
}

# Verify system resources
verify_system_resources() {
  log "Verifying system resources..."

  # Check available memory
  local available_mem_mb
  if command -v free &> /dev/null; then
    available_mem_mb=$(free -m | awk 'NR==2 {print $7}')
    log "Available memory: ${available_mem_mb}MB"

    if [ "$available_mem_mb" -lt 2000 ]; then
      warn "Low available memory: ${available_mem_mb}MB (recommend 2GB+)"
    fi
  fi

  # Check CPU load
  local load_avg
  load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
  log "System load average: $load_avg"

  return 0
}

# Generate verification report
generate_report() {
  local exit_code=$1

  log "=========================================="
  log "Infrastructure Verification Report"
  log "=========================================="
  log "Timestamp: $(date)"
  log "Kafka Container: $KAFKA_CONTAINER_ID"
  log "Kafka Network: $KAFKA_NETWORK"
  log "Kafka Connect: $KAFKA_CONNECT_URL"
  log "PostgreSQL Instances: ${#POSTGRES_HOSTS[@]}"
  log "=========================================="

  if [ $exit_code -eq 0 ]; then
    log "Status: ${GREEN}PASSED${NC}"
    log "Infrastructure is ready for CDC connector deployment"
  else
    error "Status: FAILED"
    error "Infrastructure verification failed. Fix issues before proceeding."
  fi

  log "=========================================="
}

# Main verification workflow
main() {
  log "Starting infrastructure verification for CDC deployment..."
  log "=========================================="

  local failed=0

  # Run all verification steps
  verify_tools || ((failed++))
  verify_disk_space || ((failed++))
  verify_system_resources
  verify_network || ((failed++))
  verify_kafka_container || ((failed++))
  verify_kafka_connect || ((failed++))
  verify_postgresql || ((failed++))

  # Generate report
  generate_report $failed

  return $failed
}

# Execute main function
main "$@"
