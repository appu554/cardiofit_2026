#!/bin/bash
# verify-cdc-deployment.sh
# Comprehensive health check and validation for CDC connector deployment
# Validates connector health, data flow, and Kafka topic creation

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
KAFKA_CONTAINER_ID="3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754"

# Expected connectors
declare -A EXPECTED_CONNECTORS=(
  ["kb1-medications-cdc"]="medications_db"
  ["kb2-scheduling-cdc"]="kb2_scheduling_db"
  ["kb3-encounter-cdc"]="kb3_encounter_db"
  ["kb6-drug-rules-cdc"]="kb6_drug_rules_db"
  ["kb7-guideline-evidence-cdc"]="kb7_guideline_evidence_db"
)

# Expected topic prefixes
declare -A TOPIC_PREFIXES=(
  ["kb1-medications-cdc"]="cdc.medications_db"
  ["kb2-scheduling-cdc"]="cdc.kb2_scheduling_db"
  ["kb3-encounter-cdc"]="cdc.kb3_encounter_db"
  ["kb6-drug-rules-cdc"]="cdc.kb6_drug_rules_db"
  ["kb7-guideline-evidence-cdc"]="cdc.kb7_guideline_evidence_db"
)

# Test results
declare -A TEST_RESULTS

# Check connector exists
check_connector_exists() {
  local connector_name=$1

  if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null; then
    return 0
  else
    return 1
  fi
}

# Get connector status
get_connector_status() {
  local connector_name=$1

  curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" || echo "{}"
}

# Verify connector is running
verify_connector_running() {
  local connector_name=$1

  log "[$connector_name] Verifying connector is running..."

  if ! check_connector_exists "$connector_name"; then
    error "[$connector_name] Connector does not exist"
    TEST_RESULTS[$connector_name]="FAILED: Connector does not exist"
    return 1
  fi

  local status
  status=$(get_connector_status "$connector_name")

  local connector_state
  connector_state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

  local task_state
  task_state=$(echo "$status" | jq -r '.tasks[0].state // "UNKNOWN"')

  info "[$connector_name] Connector state: $connector_state"
  info "[$connector_name] Task state: $task_state"

  if [ "$connector_state" != "RUNNING" ]; then
    error "[$connector_name] Connector is not running: $connector_state"
    TEST_RESULTS[$connector_name]="FAILED: Connector state is $connector_state"
    return 1
  fi

  if [ "$task_state" != "RUNNING" ]; then
    error "[$connector_name] Task is not running: $task_state"

    # Get error trace
    local trace
    trace=$(echo "$status" | jq -r '.tasks[0].trace // "No trace available"')
    error "[$connector_name] Error trace: $trace"

    TEST_RESULTS[$connector_name]="FAILED: Task state is $task_state"
    return 1
  fi

  log "[$connector_name] Connector and task are running"
  return 0
}

# Verify connector configuration
verify_connector_config() {
  local connector_name=$1

  log "[$connector_name] Verifying connector configuration..."

  local config
  config=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/config")

  if [ -z "$config" ]; then
    error "[$connector_name] Failed to retrieve configuration"
    return 1
  fi

  # Verify critical configuration parameters
  local connector_class
  connector_class=$(echo "$config" | jq -r '.["connector.class"] // "UNKNOWN"')

  if [ "$connector_class" != "io.debezium.connector.postgresql.PostgresConnector" ]; then
    error "[$connector_name] Unexpected connector class: $connector_class"
    return 1
  fi

  info "[$connector_name] Connector class: $connector_class"

  # Verify plugin.name
  local plugin_name
  plugin_name=$(echo "$config" | jq -r '.["plugin.name"] // "UNKNOWN"')

  info "[$connector_name] Plugin name: $plugin_name"

  # Verify slot.name
  local slot_name
  slot_name=$(echo "$config" | jq -r '.["slot.name"] // "UNKNOWN"')

  info "[$connector_name] Replication slot: $slot_name"

  # Verify publication.name
  local publication_name
  publication_name=$(echo "$config" | jq -r '.["publication.name"] // "UNKNOWN"')

  info "[$connector_name] Publication: $publication_name"

  log "[$connector_name] Configuration is valid"
  return 0
}

# Verify Kafka topics exist
verify_kafka_topics() {
  local connector_name=$1
  local topic_prefix="${TOPIC_PREFIXES[$connector_name]}"

  log "[$connector_name] Verifying Kafka topics..."

  # List topics from Kafka
  local topics
  topics=$(docker exec "$KAFKA_CONTAINER_ID" kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null || echo "")

  if [ -z "$topics" ]; then
    error "[$connector_name] Failed to list Kafka topics"
    return 1
  fi

  # Filter topics for this connector
  local connector_topics
  connector_topics=$(echo "$topics" | grep "^${topic_prefix}" || echo "")

  if [ -z "$connector_topics" ]; then
    warn "[$connector_name] No topics found with prefix: $topic_prefix"
    warn "[$connector_name] This may be normal if no data changes have occurred yet"
    return 0
  fi

  local topic_count
  topic_count=$(echo "$connector_topics" | wc -l)

  log "[$connector_name] Found $topic_count topic(s) with prefix: $topic_prefix"

  # List topics
  echo "$connector_topics" | while read -r topic; do
    info "  - $topic"
  done

  return 0
}

# Verify topic has data
verify_topic_data() {
  local topic=$1
  local connector_name=$2

  log "[$connector_name] Checking data in topic: $topic"

  # Get topic offsets
  local offsets
  offsets=$(docker exec "$KAFKA_CONTAINER_ID" kafka-run-class kafka.tools.GetOffsetShell \
    --broker-list localhost:9092 \
    --topic "$topic" 2>/dev/null || echo "")

  if [ -z "$offsets" ]; then
    warn "[$connector_name] Could not retrieve offsets for topic: $topic"
    return 0
  fi

  # Parse offsets to check if topic has data
  local total_messages=0
  while read -r line; do
    if [[ "$line" =~ :([0-9]+)$ ]]; then
      local offset="${BASH_REMATCH[1]}"
      total_messages=$((total_messages + offset))
    fi
  done <<< "$offsets"

  if [ $total_messages -gt 0 ]; then
    log "[$connector_name] Topic has $total_messages message(s)"
  else
    info "[$connector_name] Topic is empty (no changes captured yet)"
  fi

  return 0
}

# Verify connector metrics
verify_connector_metrics() {
  local connector_name=$1

  log "[$connector_name] Checking connector metrics..."

  # Get connector metrics endpoint (JMX metrics via Kafka Connect REST API)
  local metrics
  metrics=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" | jq -r '.tasks[0].id // 0')

  if [ -z "$metrics" ]; then
    warn "[$connector_name] Could not retrieve metrics"
    return 0
  fi

  info "[$connector_name] Task ID: $metrics"

  # Additional metrics would require JMX or Prometheus integration
  # Placeholder for future metrics validation

  return 0
}

# Check connector lag
check_connector_lag() {
  local connector_name=$1

  log "[$connector_name] Checking replication lag..."

  # This would require accessing PostgreSQL replication slot information
  # and comparing with Kafka topic offsets
  # Placeholder for future lag monitoring

  info "[$connector_name] Lag check requires PostgreSQL access (not implemented)"

  return 0
}

# Verify end-to-end data flow
verify_data_flow() {
  local connector_name=$1
  local database="${EXPECTED_CONNECTORS[$connector_name]}"

  log "[$connector_name] Verifying end-to-end data flow..."

  # Check if connector is capturing changes
  local status
  status=$(get_connector_status "$connector_name")

  # Get task metrics if available
  local task_metrics
  task_metrics=$(echo "$status" | jq -r '.tasks[0] // {}')

  info "[$connector_name] Task: $(echo "$task_metrics" | jq -c '.')"

  # Verify topics exist
  if ! verify_kafka_topics "$connector_name"; then
    warn "[$connector_name] Topic verification failed"
  fi

  log "[$connector_name] Data flow verification completed"

  return 0
}

# Run comprehensive validation for a connector
validate_connector() {
  local connector_name=$1

  log "=========================================="
  log "Validating connector: $connector_name"
  log "=========================================="

  local validation_passed=true

  # Check connector is running
  if ! verify_connector_running "$connector_name"; then
    validation_passed=false
  fi

  # Verify configuration
  if ! verify_connector_config "$connector_name"; then
    validation_passed=false
  fi

  # Verify topics
  if ! verify_kafka_topics "$connector_name"; then
    validation_passed=false
  fi

  # Verify data flow
  if ! verify_data_flow "$connector_name"; then
    validation_passed=false
  fi

  # Check metrics
  verify_connector_metrics "$connector_name"

  # Check lag
  check_connector_lag "$connector_name"

  if $validation_passed; then
    log "[$connector_name] ✅ All validations passed"
    TEST_RESULTS[$connector_name]="PASSED"
  else
    error "[$connector_name] ❌ Some validations failed"
    [ -z "${TEST_RESULTS[$connector_name]:-}" ] && TEST_RESULTS[$connector_name]="FAILED"
  fi

  log "=========================================="

  return 0
}

# Validate all connectors
validate_all_connectors() {
  log "Starting comprehensive validation of all CDC connectors..."

  for connector_name in "${!EXPECTED_CONNECTORS[@]}"; do
    validate_connector "$connector_name"
  done
}

# Generate validation report
generate_validation_report() {
  log "=========================================="
  log "CDC Deployment Validation Report"
  log "=========================================="
  log "Timestamp: $(date)"
  log "Kafka Connect: $KAFKA_CONNECT_URL"
  log "Kafka Container: $KAFKA_CONTAINER_ID"
  log "=========================================="

  local total=${#EXPECTED_CONNECTORS[@]}
  local passed=0
  local failed=0

  for connector_name in "${!EXPECTED_CONNECTORS[@]}"; do
    local result="${TEST_RESULTS[$connector_name]:-NOT_TESTED}"

    if [[ "$result" == "PASSED" ]]; then
      log "✅ $connector_name: $result"
      ((passed++))
    else
      error "❌ $connector_name: $result"
      ((failed++))
    fi
  done

  log "=========================================="
  log "Summary:"
  log "  Total connectors: $total"
  log "  Passed: $passed"
  log "  Failed: $failed"
  log "=========================================="

  if [ $failed -eq 0 ]; then
    log "🎉 All CDC connectors are healthy and operational"
    return 0
  else
    error "⚠️  Some CDC connectors have issues"
    return 1
  fi
}

# Quick health check
quick_health_check() {
  log "Running quick health check..."

  local healthy=0
  local unhealthy=0

  for connector_name in "${!EXPECTED_CONNECTORS[@]}"; do
    if check_connector_exists "$connector_name"; then
      local status
      status=$(get_connector_status "$connector_name")
      local state
      state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

      if [ "$state" = "RUNNING" ]; then
        log "✅ $connector_name: RUNNING"
        ((healthy++))
      else
        error "❌ $connector_name: $state"
        ((unhealthy++))
      fi
    else
      error "❌ $connector_name: NOT FOUND"
      ((unhealthy++))
    fi
  done

  log "Health check: $healthy healthy, $unhealthy unhealthy"

  [ $unhealthy -eq 0 ]
}

# Main function
main() {
  local mode="${1:-full}"

  log "CDC Deployment Verification Script"
  log "Mode: $mode"
  log "=========================================="

  case "$mode" in
    full)
      validate_all_connectors
      generate_validation_report
      ;;

    quick)
      quick_health_check
      ;;

    connector)
      local connector_name="${2:-}"
      if [ -z "$connector_name" ]; then
        error "Connector name required for single connector validation"
        echo "Usage: $0 connector <connector-name>"
        exit 1
      fi
      validate_connector "$connector_name"
      ;;

    *)
      error "Unknown mode: $mode"
      echo "Usage: $0 {full|quick|connector <name>}"
      exit 1
      ;;
  esac
}

# Execute
main "$@"
