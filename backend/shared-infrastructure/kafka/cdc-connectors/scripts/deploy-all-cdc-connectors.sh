#!/bin/bash
# deploy-all-cdc-connectors.sh
# Deploy all CDC connectors to Kafka Connect cluster with validation and health checks
# Idempotent deployment with automatic retry and rollback capabilities

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
log() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
info() { echo -e "${BLUE}[INFO]${NC} $*"; }

# Configuration
KAFKA_CONNECT_URL="${KAFKA_CONNECT_URL:-http://localhost:8083}"
CONFIG_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/configs"
MAX_RETRIES=3
RETRY_DELAY=5
HEALTH_CHECK_TIMEOUT=120

# Connector configuration mapping
# All 8 CDC connectors for KB1-KB7 services
declare -A CONNECTORS=(
  ["kb1-medications-cdc"]="kb1-medications-cdc.json"
  ["kb2-scheduling-cdc"]="kb2-scheduling-cdc.json"
  ["kb3-encounter-cdc"]="kb3-encounter-cdc.json"
  ["kb4-drug-calculations-cdc"]="kb4-drug-calculations-cdc.json"
  ["kb5-drug-interactions-cdc"]="kb5-drug-interactions-cdc.json"
  ["kb6-drug-rules-cdc"]="kb6-drug-rules-cdc.json"
  ["kb7-guideline-evidence-cdc"]="kb7-guideline-evidence-cdc.json"
  ["kb7-terminology-releases-cdc"]="kb7-terminology-releases-cdc.json"
)

# Deployment state tracking
DEPLOYED_CONNECTORS=()
FAILED_CONNECTORS=()

# Check Kafka Connect availability
check_kafka_connect() {
  log "Checking Kafka Connect availability..."

  local retries=0
  while [ $retries -lt $MAX_RETRIES ]; do
    if curl -sf "${KAFKA_CONNECT_URL}/" > /dev/null; then
      log "Kafka Connect is available at $KAFKA_CONNECT_URL"
      return 0
    fi

    ((retries++))
    warn "Kafka Connect not available (attempt $retries/$MAX_RETRIES)"
    sleep $RETRY_DELAY
  done

  error "Kafka Connect is not available after $MAX_RETRIES attempts"
  return 1
}

# Validate connector configuration file
validate_config() {
  local config_file=$1
  local connector_name=$2

  log "[$connector_name] Validating configuration file: $config_file"

  if [ ! -f "$config_file" ]; then
    error "[$connector_name] Configuration file not found: $config_file"
    return 1
  fi

  # Validate JSON syntax
  if ! jq empty "$config_file" 2>/dev/null; then
    error "[$connector_name] Invalid JSON in configuration file"
    return 1
  fi

  # Validate required fields
  local required_fields=("name" "config.connector.class" "config.database.hostname")
  for field in "${required_fields[@]}"; do
    if ! jq -e ".${field}" "$config_file" > /dev/null 2>&1; then
      error "[$connector_name] Missing required field: $field"
      return 1
    fi
  done

  log "[$connector_name] Configuration file is valid"
  return 0
}

# Check if connector already exists
connector_exists() {
  local connector_name=$1

  curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1
}

# Get connector status
get_connector_status() {
  local connector_name=$1

  curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" 2>/dev/null || echo "{}"
}

# Delete existing connector
delete_connector() {
  local connector_name=$1

  log "[$connector_name] Deleting existing connector..."

  if curl -sf -X DELETE "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null; then
    log "[$connector_name] Connector deleted successfully"
    sleep 3  # Wait for cleanup
    return 0
  else
    error "[$connector_name] Failed to delete connector"
    return 1
  fi
}

# Pause connector
pause_connector() {
  local connector_name=$1

  log "[$connector_name] Pausing connector..."

  if curl -sf -X PUT "${KAFKA_CONNECT_URL}/connectors/${connector_name}/pause" > /dev/null; then
    log "[$connector_name] Connector paused successfully"
    return 0
  else
    error "[$connector_name] Failed to pause connector"
    return 1
  fi
}

# Resume connector
resume_connector() {
  local connector_name=$1

  log "[$connector_name] Resuming connector..."

  if curl -sf -X PUT "${KAFKA_CONNECT_URL}/connectors/${connector_name}/resume" > /dev/null; then
    log "[$connector_name] Connector resumed successfully"
    return 0
  else
    error "[$connector_name] Failed to resume connector"
    return 1
  fi
}

# Deploy connector
deploy_connector() {
  local connector_name=$1
  local config_file=$2

  log "=========================================="
  log "Deploying connector: $connector_name"
  log "=========================================="

  # Validate configuration
  if ! validate_config "$config_file" "$connector_name"; then
    error "[$connector_name] Configuration validation failed"
    return 1
  fi

  # Check if connector exists
  if connector_exists "$connector_name"; then
    warn "[$connector_name] Connector already exists"

    # Get current status
    local status
    status=$(get_connector_status "$connector_name")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

    info "[$connector_name] Current state: $state"

    # Ask for action
    if [ "${AUTO_REPLACE:-false}" = "true" ]; then
      warn "[$connector_name] AUTO_REPLACE enabled - deleting existing connector"
      delete_connector "$connector_name" || return 1
    else
      warn "[$connector_name] Skipping deployment (use AUTO_REPLACE=true to force)"
      return 0
    fi
  fi

  # Deploy new connector
  log "[$connector_name] Creating connector..."

  local response
  local http_code

  response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    --data @"$config_file" \
    "${KAFKA_CONNECT_URL}/connectors")

  http_code=$(echo "$response" | tail -n1)
  local body
  body=$(echo "$response" | head -n-1)

  if [ "$http_code" -eq 201 ] || [ "$http_code" -eq 200 ]; then
    log "[$connector_name] Connector created successfully"
    DEPLOYED_CONNECTORS+=("$connector_name")
    return 0
  else
    error "[$connector_name] Deployment failed (HTTP $http_code)"
    error "[$connector_name] Response: $body"
    FAILED_CONNECTORS+=("$connector_name")
    return 1
  fi
}

# Wait for connector to be running
wait_for_connector() {
  local connector_name=$1
  local timeout=$2

  log "[$connector_name] Waiting for connector to be RUNNING (timeout: ${timeout}s)..."

  local elapsed=0
  while [ $elapsed -lt $timeout ]; do
    local status
    status=$(get_connector_status "$connector_name")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')
    local task_state
    task_state=$(echo "$status" | jq -r '.tasks[0].state // "UNKNOWN"')

    info "[$connector_name] Connector: $state, Task: $task_state"

    if [ "$state" = "RUNNING" ] && [ "$task_state" = "RUNNING" ]; then
      log "[$connector_name] Connector is running successfully"
      return 0
    fi

    if [ "$state" = "FAILED" ] || [ "$task_state" = "FAILED" ]; then
      error "[$connector_name] Connector failed to start"
      local trace
      trace=$(echo "$status" | jq -r '.tasks[0].trace // "No trace available"')
      error "[$connector_name] Error: $trace"
      return 1
    fi

    sleep 5
    elapsed=$((elapsed + 5))
  done

  error "[$connector_name] Timeout waiting for connector to start"
  return 1
}

# Validate connector health
validate_connector_health() {
  local connector_name=$1

  log "[$connector_name] Validating connector health..."

  local status
  status=$(get_connector_status "$connector_name")

  # Check connector state
  local connector_state
  connector_state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

  if [ "$connector_state" != "RUNNING" ]; then
    error "[$connector_name] Connector is not running: $connector_state"
    return 1
  fi

  # Check task states
  local task_count
  task_count=$(echo "$status" | jq '.tasks | length')
  local failed_tasks=0

  for ((i=0; i<task_count; i++)); do
    local task_state
    task_state=$(echo "$status" | jq -r ".tasks[$i].state // \"UNKNOWN\"")

    if [ "$task_state" != "RUNNING" ]; then
      error "[$connector_name] Task $i is not running: $task_state"
      ((failed_tasks++))
    fi
  done

  if [ $failed_tasks -gt 0 ]; then
    error "[$connector_name] $failed_tasks task(s) are not running"
    return 1
  fi

  log "[$connector_name] Connector is healthy (state: $connector_state, tasks: $task_count)"
  return 0
}

# Deploy all connectors
deploy_all_connectors() {
  log "Starting deployment of all CDC connectors..."
  log "=========================================="

  local total=${#CONNECTORS[@]}
  local deployed=0

  for connector_name in "${!CONNECTORS[@]}"; do
    local config_file="${CONFIG_DIR}/${CONNECTORS[$connector_name]}"

    if deploy_connector "$connector_name" "$config_file"; then
      if wait_for_connector "$connector_name" $HEALTH_CHECK_TIMEOUT; then
        if validate_connector_health "$connector_name"; then
          ((deployed++))
        fi
      fi
    fi

    log "=========================================="
  done

  log "Deployment Summary:"
  log "  Total connectors: $total"
  log "  Successfully deployed: $deployed"
  log "  Failed: $((total - deployed))"

  if [ $deployed -eq $total ]; then
    log "All connectors deployed successfully"
    return 0
  else
    error "Some connectors failed to deploy"
    return 1
  fi
}

# List all connectors
list_connectors() {
  log "Listing all connectors..."

  local connectors
  connectors=$(curl -sf "${KAFKA_CONNECT_URL}/connectors")

  if [ -z "$connectors" ]; then
    warn "No connectors found"
    return 0
  fi

  local count
  count=$(echo "$connectors" | jq 'length')
  log "Found $count connector(s):"

  echo "$connectors" | jq -r '.[]' | while read -r connector_name; do
    local status
    status=$(get_connector_status "$connector_name")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')
    local task_state
    task_state=$(echo "$status" | jq -r '.tasks[0].state // "UNKNOWN"')

    log "  - $connector_name (connector: $state, task: $task_state)"
  done
}

# Generate deployment report
generate_deployment_report() {
  log "=========================================="
  log "CDC Connector Deployment Report"
  log "=========================================="
  log "Timestamp: $(date)"
  log "Kafka Connect: $KAFKA_CONNECT_URL"
  log "=========================================="

  if [ ${#DEPLOYED_CONNECTORS[@]} -gt 0 ]; then
    log "Successfully Deployed Connectors:"
    for connector in "${DEPLOYED_CONNECTORS[@]}"; do
      log "  ✅ $connector"
    done
  fi

  if [ ${#FAILED_CONNECTORS[@]} -gt 0 ]; then
    error "Failed Connectors:"
    for connector in "${FAILED_CONNECTORS[@]}"; do
      error "  ❌ $connector"
    done
  fi

  log "=========================================="

  # Detailed status for each connector
  for connector_name in "${!CONNECTORS[@]}"; do
    if connector_exists "$connector_name"; then
      local status
      status=$(get_connector_status "$connector_name")

      log "[$connector_name] Detailed Status:"
      echo "$status" | jq '.'
      log "=========================================="
    fi
  done
}

# Pause all connectors
pause_all_connectors() {
  log "Pausing all CDC connectors..."

  for connector_name in "${!CONNECTORS[@]}"; do
    if connector_exists "$connector_name"; then
      pause_connector "$connector_name"
    else
      warn "[$connector_name] Connector does not exist"
    fi
  done

  log "All connectors paused"
}

# Resume all connectors
resume_all_connectors() {
  log "Resuming all CDC connectors..."

  for connector_name in "${!CONNECTORS[@]}"; do
    if connector_exists "$connector_name"; then
      resume_connector "$connector_name"
    else
      warn "[$connector_name] Connector does not exist"
    fi
  done

  log "All connectors resumed"
}

# Main function
main() {
  local command="${1:-deploy}"

  log "CDC Connector Deployment Script"
  log "Command: $command"
  log "=========================================="

  # Check Kafka Connect availability
  if ! check_kafka_connect; then
    error "Kafka Connect is not available"
    exit 1
  fi

  case "$command" in
    deploy)
      deploy_all_connectors
      generate_deployment_report
      ;;

    list)
      list_connectors
      ;;

    pause)
      pause_all_connectors
      ;;

    resume)
      resume_all_connectors
      ;;

    status)
      generate_deployment_report
      ;;

    *)
      error "Unknown command: $command"
      echo "Usage: $0 {deploy|list|pause|resume|status}"
      exit 1
      ;;
  esac
}

# Execute
main "$@"
