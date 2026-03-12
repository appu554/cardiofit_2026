#!/bin/bash
# rollback-cdc.sh
# Emergency rollback and disaster recovery for CDC connectors
# Supports full rollback, partial rollback, and connector pause/resume

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
BACKUP_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/backups"
KAFKA_CONTAINER_ID="3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754"

# Database connection strings
declare -A DATABASES=(
  ["kb1-medications-cdc"]="localhost:5432:medications_db:postgres:${PGPASSWORD_KB1:-postgres}"
  ["kb2-scheduling-cdc"]="localhost:5433:kb2_scheduling_db:postgres:${PGPASSWORD_KB2:-postgres}"
  ["kb3-encounter-cdc"]="localhost:5434:kb3_encounter_db:postgres:${PGPASSWORD_KB3:-postgres}"
  ["kb6-drug-rules-cdc"]="localhost:5435:kb6_drug_rules_db:postgres:${PGPASSWORD_KB6:-postgres}"
  ["kb7-guideline-evidence-cdc"]="localhost:5436:kb7_guideline_evidence_db:postgres:${PGPASSWORD_KB7:-postgres}"
)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup connector configuration
backup_connector_config() {
  local connector_name=$1

  log "[$connector_name] Backing up connector configuration..."

  local config
  config=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/config" 2>/dev/null || echo "")

  if [ -z "$config" ]; then
    warn "[$connector_name] Connector not found or configuration unavailable"
    return 1
  fi

  local backup_file="${BACKUP_DIR}/${connector_name}_$(date +%Y%m%d_%H%M%S).json"
  echo "$config" | jq '.' > "$backup_file"

  log "[$connector_name] Configuration backed up to: $backup_file"
  return 0
}

# Backup all connector configurations
backup_all_configs() {
  log "Backing up all connector configurations..."

  for connector_name in "${!DATABASES[@]}"; do
    backup_connector_config "$connector_name" || warn "[$connector_name] Backup failed"
  done

  log "Configuration backups completed"
}

# Pause connector
pause_connector() {
  local connector_name=$1

  log "[$connector_name] Pausing connector..."

  # Backup before pausing
  backup_connector_config "$connector_name"

  if curl -sf -X PUT "${KAFKA_CONNECT_URL}/connectors/${connector_name}/pause" > /dev/null; then
    log "[$connector_name] Connector paused successfully"

    # Verify pause
    sleep 2
    local status
    status=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

    if [ "$state" = "PAUSED" ]; then
      log "[$connector_name] Verified connector is paused"
      return 0
    else
      warn "[$connector_name] Connector state is $state (expected PAUSED)"
      return 1
    fi
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

    # Verify resume
    sleep 2
    local status
    status=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

    if [ "$state" = "RUNNING" ]; then
      log "[$connector_name] Verified connector is running"
      return 0
    else
      warn "[$connector_name] Connector state is $state (expected RUNNING)"
      return 1
    fi
  else
    error "[$connector_name] Failed to resume connector"
    return 1
  fi
}

# Delete connector
delete_connector() {
  local connector_name=$1

  log "[$connector_name] Deleting connector..."

  # Backup before deletion
  backup_connector_config "$connector_name"

  if curl -sf -X DELETE "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null; then
    log "[$connector_name] Connector deleted successfully"

    # Verify deletion
    sleep 2
    if ! curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
      log "[$connector_name] Verified connector is deleted"
      return 0
    else
      warn "[$connector_name] Connector still exists after deletion attempt"
      return 1
    fi
  else
    error "[$connector_name] Failed to delete connector"
    return 1
  fi
}

# Restart connector tasks
restart_connector_tasks() {
  local connector_name=$1

  log "[$connector_name] Restarting connector tasks..."

  if curl -sf -X POST "${KAFKA_CONNECT_URL}/connectors/${connector_name}/restart" > /dev/null; then
    log "[$connector_name] Connector restarted successfully"

    # Wait for restart
    sleep 5

    # Verify restart
    local status
    status=$(curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status")
    local state
    state=$(echo "$status" | jq -r '.connector.state // "UNKNOWN"')

    if [ "$state" = "RUNNING" ]; then
      log "[$connector_name] Connector is running after restart"
      return 0
    else
      warn "[$connector_name] Connector state is $state after restart"
      return 1
    fi
  else
    error "[$connector_name] Failed to restart connector"
    return 1
  fi
}

# Clean up PostgreSQL replication slot
cleanup_replication_slot() {
  local connector_name=$1
  local config="${DATABASES[$connector_name]}"

  IFS=':' read -r host port db user password <<< "$config"

  local slot_name="debezium_${connector_name//-/_}"

  log "[$connector_name] Cleaning up replication slot: $slot_name"

  # Check if slot exists
  local slot_exists
  slot_exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT 1 FROM pg_replication_slots WHERE slot_name='$slot_name';" | tr -d ' ' || echo "")

  if [ "$slot_exists" != "1" ]; then
    info "[$connector_name] Replication slot does not exist"
    return 0
  fi

  # Check if slot is active
  local slot_active
  slot_active=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT active FROM pg_replication_slots WHERE slot_name='$slot_name';" | tr -d ' ' || echo "")

  if [ "$slot_active" = "t" ]; then
    warn "[$connector_name] Replication slot is active - cannot drop"
    warn "[$connector_name] Pause or delete the connector first"
    return 1
  fi

  # Drop replication slot
  if PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
    "SELECT pg_drop_replication_slot('$slot_name');" > /dev/null 2>&1; then
    log "[$connector_name] Replication slot dropped successfully"
    return 0
  else
    error "[$connector_name] Failed to drop replication slot"
    return 1
  fi
}

# Clean up publication
cleanup_publication() {
  local connector_name=$1
  local config="${DATABASES[$connector_name]}"

  IFS=':' read -r host port db user password <<< "$config"

  local pub_name="dbz_publication_${connector_name//-/_}"

  log "[$connector_name] Cleaning up publication: $pub_name"

  # Check if publication exists
  local pub_exists
  pub_exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT 1 FROM pg_publication WHERE pubname='$pub_name';" | tr -d ' ' || echo "")

  if [ "$pub_exists" != "1" ]; then
    info "[$connector_name] Publication does not exist"
    return 0
  fi

  # Drop publication
  if PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
    "DROP PUBLICATION IF EXISTS $pub_name;" > /dev/null 2>&1; then
    log "[$connector_name] Publication dropped successfully"
    return 0
  else
    error "[$connector_name] Failed to drop publication"
    return 1
  fi
}

# Delete Kafka topics
delete_kafka_topics() {
  local topic_prefix=$1
  local connector_name=$2

  log "[$connector_name] Deleting Kafka topics with prefix: $topic_prefix"

  # List topics
  local topics
  topics=$(docker exec "$KAFKA_CONTAINER_ID" kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | grep "^${topic_prefix}" || echo "")

  if [ -z "$topics" ]; then
    info "[$connector_name] No topics found with prefix: $topic_prefix"
    return 0
  fi

  local topic_count
  topic_count=$(echo "$topics" | wc -l)
  warn "[$connector_name] Found $topic_count topic(s) to delete"

  # Delete each topic
  echo "$topics" | while read -r topic; do
    warn "  Deleting topic: $topic"
    docker exec "$KAFKA_CONTAINER_ID" kafka-topics --bootstrap-server localhost:9092 --delete --topic "$topic" 2>/dev/null || true
  done

  log "[$connector_name] Kafka topics deleted"
  return 0
}

# Full rollback for a connector
full_rollback_connector() {
  local connector_name=$1

  log "=========================================="
  log "[$connector_name] Starting full rollback"
  log "=========================================="

  # Step 1: Pause connector (if running)
  if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
    pause_connector "$connector_name" || warn "[$connector_name] Failed to pause connector"
  fi

  # Step 2: Delete connector
  delete_connector "$connector_name" || warn "[$connector_name] Failed to delete connector"

  # Step 3: Clean up replication slot
  cleanup_replication_slot "$connector_name" || warn "[$connector_name] Failed to cleanup replication slot"

  # Step 4: Clean up publication
  cleanup_publication "$connector_name" || warn "[$connector_name] Failed to cleanup publication"

  # Step 5: Delete Kafka topics (optional - commented out for safety)
  # local topic_prefix="cdc.${DATABASES[$connector_name]#*:*:}"
  # delete_kafka_topics "$topic_prefix" "$connector_name"

  log "[$connector_name] Full rollback completed"
  log "=========================================="
}

# Emergency pause all connectors
emergency_pause_all() {
  warn "=========================================="
  warn "EMERGENCY PAUSE - Pausing all connectors"
  warn "=========================================="

  for connector_name in "${!DATABASES[@]}"; do
    if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
      pause_connector "$connector_name"
    else
      info "[$connector_name] Connector does not exist"
    fi
  done

  log "Emergency pause completed"
}

# Resume all connectors
resume_all() {
  log "Resuming all connectors..."

  for connector_name in "${!DATABASES[@]}"; do
    if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
      resume_connector "$connector_name"
    else
      warn "[$connector_name] Connector does not exist"
    fi
  done

  log "All connectors resumed"
}

# Restart all connectors
restart_all() {
  log "Restarting all connectors..."

  for connector_name in "${!DATABASES[@]}"; do
    if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
      restart_connector_tasks "$connector_name"
    else
      warn "[$connector_name] Connector does not exist"
    fi
  done

  log "All connectors restarted"
}

# Full rollback all connectors
full_rollback_all() {
  warn "=========================================="
  warn "FULL ROLLBACK - This will delete all CDC infrastructure"
  warn "=========================================="
  warn "This operation will:"
  warn "  - Delete all connectors"
  warn "  - Drop replication slots"
  warn "  - Drop publications"
  warn "Press Ctrl+C within 10 seconds to cancel..."
  warn "=========================================="

  sleep 10

  for connector_name in "${!DATABASES[@]}"; do
    full_rollback_connector "$connector_name"
  done

  log "Full rollback of all connectors completed"
}

# List connector backups
list_backups() {
  log "Available connector backups:"

  if [ ! -d "$BACKUP_DIR" ] || [ -z "$(ls -A "$BACKUP_DIR")" ]; then
    info "No backups found"
    return 0
  fi

  ls -lh "$BACKUP_DIR"/*.json 2>/dev/null || info "No JSON backups found"
}

# Restore connector from backup
restore_connector() {
  local connector_name=$1
  local backup_file=$2

  if [ ! -f "$backup_file" ]; then
    error "Backup file not found: $backup_file"
    return 1
  fi

  log "[$connector_name] Restoring connector from backup: $backup_file"

  # Delete existing connector if present
  if curl -sf "${KAFKA_CONNECT_URL}/connectors/${connector_name}" > /dev/null 2>&1; then
    warn "[$connector_name] Deleting existing connector"
    delete_connector "$connector_name"
    sleep 3
  fi

  # Create connector from backup
  local response
  response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$connector_name\",\"config\":$(cat "$backup_file")}" \
    "${KAFKA_CONNECT_URL}/connectors")

  local http_code
  http_code=$(echo "$response" | tail -n1)

  if [ "$http_code" -eq 201 ] || [ "$http_code" -eq 200 ]; then
    log "[$connector_name] Connector restored successfully"
    return 0
  else
    error "[$connector_name] Failed to restore connector (HTTP $http_code)"
    return 1
  fi
}

# Main function
main() {
  local command="${1:-}"

  if [ -z "$command" ]; then
    error "Command required"
    echo "Usage: $0 {pause-all|resume-all|restart-all|backup|rollback|rollback-all|list-backups|restore}"
    exit 1
  fi

  log "CDC Rollback and Recovery Script"
  log "Command: $command"
  log "=========================================="

  case "$command" in
    pause-all)
      emergency_pause_all
      ;;

    resume-all)
      resume_all
      ;;

    restart-all)
      restart_all
      ;;

    backup)
      backup_all_configs
      ;;

    rollback)
      local connector_name="${2:-}"
      if [ -z "$connector_name" ]; then
        error "Connector name required"
        echo "Usage: $0 rollback <connector-name>"
        exit 1
      fi
      full_rollback_connector "$connector_name"
      ;;

    rollback-all)
      full_rollback_all
      ;;

    list-backups)
      list_backups
      ;;

    restore)
      local connector_name="${2:-}"
      local backup_file="${3:-}"
      if [ -z "$connector_name" ] || [ -z "$backup_file" ]; then
        error "Connector name and backup file required"
        echo "Usage: $0 restore <connector-name> <backup-file>"
        exit 1
      fi
      restore_connector "$connector_name" "$backup_file"
      ;;

    *)
      error "Unknown command: $command"
      echo "Usage: $0 {pause-all|resume-all|restart-all|backup|rollback|rollback-all|list-backups|restore}"
      exit 1
      ;;
  esac

  log "=========================================="
  log "Command completed: $command"
}

# Execute
main "$@"
