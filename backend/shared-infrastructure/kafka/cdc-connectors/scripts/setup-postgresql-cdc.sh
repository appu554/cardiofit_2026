#!/bin/bash
# setup-postgresql-cdc.sh
# Prepare PostgreSQL instances for CDC with WAL configuration, replication slots, and publications
# Idempotent script - safe to run multiple times

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging
log() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

# Configuration - matches CDC connector configs
# All 7 Knowledge Base databases (KB1-KB7)
set +u  # Temporarily disable unbound variable check for associative array
declare -A DATABASES=(
  ["KB1"]="localhost:5432:kb_drug_rules:kb_drug_rules_user:kb_password"
  ["KB2"]="localhost:5432:kb2_clinical_context:kb2_user:kb_password"
  ["KB3"]="localhost:5432:kb3_guidelines:kb3_user:password123"
  ["KB4"]="localhost:5432:kb4_drug_calculations:kb4_user:kb_password"
  ["KB5"]="localhost:5432:kb5_drug_interactions:kb5_user:kb_password"
  ["KB6"]="localhost:5432:kb_formulary:kb_formulary_user:kb_password"
  ["KB7"]="localhost:5432:kb_terminology:kb_terminology_user:kb_password"
)
set -u  # Re-enable unbound variable check

# WAL configuration settings
WAL_LEVEL="logical"
MAX_WAL_SENDERS="10"
MAX_REPLICATION_SLOTS="10"

# Check prerequisites
check_prerequisites() {
  log "Checking prerequisites..."

  if ! command -v psql &> /dev/null; then
    error "psql is not installed"
    return 1
  fi

  if ! command -v docker &> /dev/null; then
    error "docker is not installed"
    return 1
  fi

  log "Prerequisites satisfied"
  return 0
}

# Verify PostgreSQL WAL configuration
verify_wal_config() {
  local host=$1
  local port=$2
  local db=$3
  local user=$4
  local password=$5
  local kb_name=$6

  log "[$kb_name] Verifying WAL configuration on $host:$port..."

  local wal_level
  wal_level=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c "SHOW wal_level;" | tr -d ' ')

  if [ "$wal_level" != "logical" ]; then
    error "[$kb_name] WAL level is '$wal_level', expected 'logical'"
    warn "[$kb_name] Update postgresql.conf: wal_level = logical"
    warn "[$kb_name] Then restart PostgreSQL: docker restart <container>"
    return 1
  fi

  log "[$kb_name] WAL level is correctly set to 'logical'"

  # Verify max_wal_senders
  local max_senders
  max_senders=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c "SHOW max_wal_senders;" | tr -d ' ')

  if [ "$max_senders" -lt "$MAX_WAL_SENDERS" ]; then
    warn "[$kb_name] max_wal_senders is $max_senders (recommend >= $MAX_WAL_SENDERS)"
  else
    log "[$kb_name] max_wal_senders is sufficient: $max_senders"
  fi

  # Verify max_replication_slots
  local max_slots
  max_slots=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c "SHOW max_replication_slots;" | tr -d ' ')

  if [ "$max_slots" -lt "$MAX_REPLICATION_SLOTS" ]; then
    warn "[$kb_name] max_replication_slots is $max_slots (recommend >= $MAX_REPLICATION_SLOTS)"
  else
    log "[$kb_name] max_replication_slots is sufficient: $max_slots"
  fi

  return 0
}

# Create replication user if not exists
create_replication_user() {
  local host=$1
  local port=$2
  local db=$3
  local user=$4
  local password=$5
  local kb_name=$6

  log "[$kb_name] Creating replication user 'debezium'..."

  local user_exists
  user_exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT 1 FROM pg_roles WHERE rolname='debezium';" | tr -d ' ')

  if [ "$user_exists" = "1" ]; then
    log "[$kb_name] Replication user 'debezium' already exists"
  else
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" <<EOF
CREATE USER debezium WITH REPLICATION PASSWORD 'debezium_password_change_in_production';
GRANT CONNECT ON DATABASE $db TO debezium;
EOF
    log "[$kb_name] Created replication user 'debezium'"
  fi

  # Grant schema permissions
  PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" <<EOF
GRANT USAGE ON SCHEMA public TO debezium;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO debezium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO debezium;
EOF

  log "[$kb_name] Granted necessary permissions to debezium user"

  return 0
}

# Create replication slot
create_replication_slot() {
  local host=$1
  local port=$2
  local db=$3
  local user=$4
  local password=$5
  local kb_name=$6
  local slot_name="debezium_${kb_name,,}"

  log "[$kb_name] Creating replication slot '$slot_name'..."

  # Check if slot exists
  local slot_exists
  slot_exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT 1 FROM pg_replication_slots WHERE slot_name='$slot_name';" | tr -d ' ')

  if [ "$slot_exists" = "1" ]; then
    log "[$kb_name] Replication slot '$slot_name' already exists"

    # Check slot health
    local slot_active
    slot_active=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
      "SELECT active FROM pg_replication_slots WHERE slot_name='$slot_name';" | tr -d ' ')

    if [ "$slot_active" = "t" ]; then
      log "[$kb_name] Replication slot is active"
    else
      warn "[$kb_name] Replication slot is inactive"
    fi
  else
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
      "SELECT pg_create_logical_replication_slot('$slot_name', 'pgoutput');"

    log "[$kb_name] Created replication slot '$slot_name'"
  fi

  return 0
}

# Create publication
create_publication() {
  local host=$1
  local port=$2
  local db=$3
  local user=$4
  local password=$5
  local kb_name=$6
  local pub_name="dbz_publication_${kb_name,,}"

  log "[$kb_name] Creating publication '$pub_name'..."

  # Check if publication exists
  local pub_exists
  pub_exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
    "SELECT 1 FROM pg_publication WHERE pubname='$pub_name';" | tr -d ' ')

  if [ "$pub_exists" = "1" ]; then
    log "[$kb_name] Publication '$pub_name' already exists"

    # Show publication details
    local pub_tables
    pub_tables=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
      "SELECT COUNT(*) FROM pg_publication_tables WHERE pubname='$pub_name';" | tr -d ' ')

    log "[$kb_name] Publication covers $pub_tables tables"
  else
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
      "CREATE PUBLICATION $pub_name FOR ALL TABLES;"

    log "[$kb_name] Created publication '$pub_name' for all tables"
  fi

  return 0
}

# Setup CDC for a single database
setup_database_cdc() {
  local kb_name=$1
  local config="${DATABASES[$kb_name]}"

  IFS=':' read -r host port db user password <<< "$config"

  log "=========================================="
  log "Setting up CDC for $kb_name"
  log "Database: $db at $host:$port"
  log "=========================================="

  # Test connection
  if ! PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c "SELECT 1;" > /dev/null 2>&1; then
    error "[$kb_name] Cannot connect to database"
    return 1
  fi

  log "[$kb_name] Database connection successful"

  # Verify WAL configuration
  if ! verify_wal_config "$host" "$port" "$db" "$user" "$password" "$kb_name"; then
    error "[$kb_name] WAL configuration verification failed"
    return 1
  fi

  # Create replication user
  if ! create_replication_user "$host" "$port" "$db" "$user" "$password" "$kb_name"; then
    error "[$kb_name] Failed to create replication user"
    return 1
  fi

  # Create replication slot
  if ! create_replication_slot "$host" "$port" "$db" "$user" "$password" "$kb_name"; then
    error "[$kb_name] Failed to create replication slot"
    return 1
  fi

  # Create publication
  if ! create_publication "$host" "$port" "$db" "$user" "$password" "$kb_name"; then
    error "[$kb_name] Failed to create publication"
    return 1
  fi

  log "[$kb_name] CDC setup completed successfully"
  return 0
}

# Cleanup replication artifacts (for rollback)
cleanup_database_cdc() {
  local kb_name=$1
  local config="${DATABASES[$kb_name]}"

  IFS=':' read -r host port db user password <<< "$config"

  local slot_name="debezium_${kb_name,,}"
  local pub_name="dbz_publication_${kb_name,,}"

  warn "[$kb_name] Cleaning up CDC artifacts..."

  # Drop publication
  PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
    "DROP PUBLICATION IF EXISTS $pub_name;" 2>/dev/null || true

  # Drop replication slot
  PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
    "SELECT pg_drop_replication_slot('$slot_name');" 2>/dev/null || true

  # Drop replication user
  PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -c \
    "DROP USER IF EXISTS debezium;" 2>/dev/null || true

  log "[$kb_name] CDC cleanup completed"
}

# Generate diagnostic report
generate_diagnostic_report() {
  log "=========================================="
  log "PostgreSQL CDC Diagnostic Report"
  log "=========================================="

  for kb_name in "${!DATABASES[@]}"; do
    local config="${DATABASES[$kb_name]}"
    IFS=':' read -r host port db user password <<< "$config"

    log "[$kb_name] Database: $db at $host:$port"

    # Check replication slots
    local slots
    slots=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
      "SELECT slot_name, active, confirmed_flush_lsn FROM pg_replication_slots;" 2>/dev/null || echo "ERROR")

    if [ "$slots" != "ERROR" ]; then
      log "[$kb_name] Replication slots:"
      echo "$slots" | while read -r line; do
        [ -z "$line" ] || log "  $line"
      done
    fi

    # Check publications
    local pubs
    pubs=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$db" -t -c \
      "SELECT pubname FROM pg_publication;" 2>/dev/null || echo "ERROR")

    if [ "$pubs" != "ERROR" ]; then
      log "[$kb_name] Publications:"
      echo "$pubs" | while read -r line; do
        [ -z "$line" ] || log "  $line"
      done
    fi

    log "=========================================="
  done
}

# Main function
main() {
  local mode="${1:-setup}"

  log "PostgreSQL CDC Setup Script"
  log "Mode: $mode"
  log "=========================================="

  if ! check_prerequisites; then
    error "Prerequisites check failed"
    exit 1
  fi

  case "$mode" in
    setup)
      local failed=0
      for kb_name in "KB1" "KB2" "KB3" "KB4" "KB5" "KB6" "KB7"; do
        if ! setup_database_cdc "$kb_name"; then
          ((failed++))
        fi
      done

      generate_diagnostic_report

      if [ $failed -gt 0 ]; then
        error "$failed database(s) failed CDC setup"
        exit 1
      fi

      log "All databases configured for CDC successfully"
      ;;

    cleanup)
      warn "Cleaning up CDC artifacts from all databases..."
      for kb_name in "KB1" "KB2" "KB3" "KB4" "KB5" "KB6" "KB7"; do
        cleanup_database_cdc "$kb_name"
      done
      log "CDC cleanup completed"
      ;;

    diagnostic)
      generate_diagnostic_report
      ;;

    *)
      error "Unknown mode: $mode"
      echo "Usage: $0 {setup|cleanup|diagnostic}"
      exit 1
      ;;
  esac
}

# Execute
main "$@"
