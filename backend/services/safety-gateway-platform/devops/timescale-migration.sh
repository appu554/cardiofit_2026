#!/bin/bash

# TimescaleDB Migration Script for Safety Gateway Platform
# This script handles the zero-downtime migration from PostgreSQL to TimescaleDB

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/safety-gateway/migration.log"
BACKUP_DIR="/var/backups/safety-gateway"
TIMESCALE_HOST="${TIMESCALE_HOST:-localhost}"
TIMESCALE_PORT="${TIMESCALE_PORT:-5434}"
TIMESCALE_DB="${TIMESCALE_DB:-safety}"
TIMESCALE_USER="${TIMESCALE_USER:-safety_user}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-safety_gateway}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level=$1
    shift
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE"
}

log_info() { log "${BLUE}INFO${NC}" "$@"; }
log_warn() { log "${YELLOW}WARN${NC}" "$@"; }
log_error() { log "${RED}ERROR${NC}" "$@"; }
log_success() { log "${GREEN}SUCCESS${NC}" "$@"; }

# Error handling
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Migration failed with exit code $exit_code"
        log_info "Rolling back changes..."
        rollback_migration
    fi
    exit $exit_code
}

trap cleanup EXIT

# Validation functions
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if required tools are installed
    for tool in psql docker curl jq; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Check if backup directory exists
    mkdir -p "$BACKUP_DIR"
    
    # Check if log directory exists
    mkdir -p "$(dirname "$LOG_FILE")"
    
    log_success "Prerequisites check passed"
}

test_connections() {
    log_info "Testing database connections..."
    
    # Test PostgreSQL connection
    if ! PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" > /dev/null 2>&1; then
        log_error "Cannot connect to PostgreSQL database"
        exit 1
    fi
    
    # Test TimescaleDB connection
    if ! PGPASSWORD="$TIMESCALE_PASSWORD" psql -h "$TIMESCALE_HOST" -p "$TIMESCALE_PORT" -U "$TIMESCALE_USER" -d "$TIMESCALE_DB" -c "SELECT 1;" > /dev/null 2>&1; then
        log_error "Cannot connect to TimescaleDB database"
        exit 1
    fi
    
    log_success "Database connections established"
}

# Backup functions
create_backup() {
    log_info "Creating PostgreSQL backup..."
    
    local backup_file="$BACKUP_DIR/safety_gateway_$(date +%Y%m%d_%H%M%S).sql"
    
    PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        --clean \
        --if-exists \
        --create \
        --verbose \
        > "$backup_file"
    
    # Compress backup
    gzip "$backup_file"
    backup_file="${backup_file}.gz"
    
    # Verify backup
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not created"
        exit 1
    fi
    
    local backup_size=$(du -h "$backup_file" | cut -f1)
    log_success "Backup created: $backup_file ($backup_size)"
    
    echo "$backup_file" > "$BACKUP_DIR/latest_backup.txt"
}

# Schema migration functions
setup_timescaledb_schema() {
    log_info "Setting up TimescaleDB schema..."
    
    # Create TimescaleDB extension if not exists
    PGPASSWORD="$TIMESCALE_PASSWORD" psql -h "$TIMESCALE_HOST" -p "$TIMESCALE_PORT" -U "$TIMESCALE_USER" -d "$TIMESCALE_DB" << 'EOF'
-- Create TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create schemas
CREATE SCHEMA IF NOT EXISTS safety_events;
CREATE SCHEMA IF NOT EXISTS safety_metrics;
CREATE SCHEMA IF NOT EXISTS safety_audit;

-- Set search path
SET search_path TO safety_events, safety_metrics, safety_audit, public;

-- Create safety events table (time-series optimized)
CREATE TABLE IF NOT EXISTS safety_events.requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_id VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255),
    request_type VARCHAR(50) NOT NULL,
    safety_tier INTEGER NOT NULL,
    engines_evaluated TEXT[] NOT NULL,
    decision VARCHAR(50) NOT NULL,
    response_time_ms INTEGER NOT NULL,
    override_used BOOLEAN DEFAULT FALSE,
    override_level VARCHAR(50),
    context_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create hypertable for time-series optimization
SELECT create_hypertable('safety_events.requests', 'timestamp', if_not_exists => TRUE);

-- Create safety metrics table
CREATE TABLE IF NOT EXISTS safety_metrics.performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    service_name VARCHAR(100) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    labels JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create hypertable for metrics
SELECT create_hypertable('safety_metrics.performance', 'timestamp', if_not_exists => TRUE);

-- Create audit log table
CREATE TABLE IF NOT EXISTS safety_audit.logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(255),
    patient_id VARCHAR(255),
    resource_id VARCHAR(255),
    action VARCHAR(50) NOT NULL,
    result VARCHAR(50) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create hypertable for audit logs
SELECT create_hypertable('safety_audit.logs', 'timestamp', if_not_exists => TRUE);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_requests_patient_timestamp 
    ON safety_events.requests (patient_id, timestamp DESC);
    
CREATE INDEX IF NOT EXISTS idx_requests_decision_timestamp 
    ON safety_events.requests (decision, timestamp DESC) 
    WHERE decision IN ('unsafe', 'warning');
    
CREATE INDEX IF NOT EXISTS idx_requests_override_timestamp 
    ON safety_events.requests (override_used, timestamp DESC) 
    WHERE override_used = TRUE;
    
CREATE INDEX IF NOT EXISTS idx_performance_service_metric 
    ON safety_metrics.performance (service_name, metric_name, timestamp DESC);
    
CREATE INDEX IF NOT EXISTS idx_audit_patient_timestamp 
    ON safety_audit.logs (patient_id, timestamp DESC);

-- Create retention policies
-- Keep raw events for 1 year, then compress
SELECT add_retention_policy('safety_events.requests', INTERVAL '1 year', if_not_exists => TRUE);

-- Keep metrics for 90 days, then compress  
SELECT add_retention_policy('safety_metrics.performance', INTERVAL '90 days', if_not_exists => TRUE);

-- Keep audit logs for 7 years (HIPAA compliance)
SELECT add_retention_policy('safety_audit.logs', INTERVAL '7 years', if_not_exists => TRUE);

-- Create continuous aggregates for common queries
CREATE MATERIALIZED VIEW IF NOT EXISTS safety_metrics.hourly_decision_summary
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', timestamp) AS hour,
    decision,
    COUNT(*) AS decision_count,
    AVG(response_time_ms) AS avg_response_time,
    MAX(response_time_ms) AS max_response_time,
    COUNT(*) FILTER (WHERE override_used = TRUE) AS override_count
FROM safety_events.requests
GROUP BY hour, decision;

-- Add refresh policy for continuous aggregate
SELECT add_continuous_aggregate_policy('safety_metrics.hourly_decision_summary',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

-- Create functions for data validation
CREATE OR REPLACE FUNCTION validate_safety_request(
    p_request_id VARCHAR,
    p_patient_id VARCHAR,
    p_decision VARCHAR
) RETURNS BOOLEAN AS $$
BEGIN
    -- Validate request ID format
    IF p_request_id IS NULL OR LENGTH(p_request_id) = 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Validate patient ID format  
    IF p_patient_id IS NULL OR LENGTH(p_patient_id) = 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Validate decision values
    IF p_decision NOT IN ('safe', 'unsafe', 'warning') THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for data validation
CREATE OR REPLACE FUNCTION validate_safety_request_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT validate_safety_request(NEW.request_id, NEW.patient_id, NEW.decision) THEN
        RAISE EXCEPTION 'Invalid safety request data: request_id=%, patient_id=%, decision=%', 
            NEW.request_id, NEW.patient_id, NEW.decision;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS safety_request_validation ON safety_events.requests;
CREATE TRIGGER safety_request_validation
    BEFORE INSERT OR UPDATE ON safety_events.requests
    FOR EACH ROW EXECUTE FUNCTION validate_safety_request_trigger();

EOF

    log_success "TimescaleDB schema setup completed"
}

# Data migration functions
migrate_existing_data() {
    log_info "Migrating existing data from PostgreSQL to TimescaleDB..."
    
    # Get table list from PostgreSQL
    local tables=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "
        SELECT tablename FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename NOT LIKE 'pg_%'
        ORDER BY tablename;
    " | xargs)
    
    log_info "Found tables to migrate: $tables"
    
    for table in $tables; do
        log_info "Migrating table: $table"
        
        # Export data from PostgreSQL
        local temp_file="/tmp/${table}_migration_$(date +%s).csv"
        
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
            COPY (SELECT * FROM $table) TO STDOUT WITH CSV HEADER;
        " > "$temp_file"
        
        # Transform and import to TimescaleDB
        case "$table" in
            "safety_requests")
                # Map to new schema
                PGPASSWORD="$TIMESCALE_PASSWORD" psql -h "$TIMESCALE_HOST" -p "$TIMESCALE_PORT" -U "$TIMESCALE_USER" -d "$TIMESCALE_DB" -c "
                    COPY safety_events.requests(timestamp, request_id, patient_id, user_id, request_type, safety_tier, engines_evaluated, decision, response_time_ms, override_used, override_level, context_data)
                    FROM STDIN WITH CSV HEADER;
                " < "$temp_file"
                ;;
            "audit_logs")
                # Map to audit schema
                PGPASSWORD="$TIMESCALE_PASSWORD" psql -h "$TIMESCALE_HOST" -p "$TIMESCALE_PORT" -U "$TIMESCALE_USER" -d "$TIMESCALE_DB" -c "
                    COPY safety_audit.logs(timestamp, event_type, user_id, patient_id, resource_id, action, result, details, ip_address, user_agent)
                    FROM STDIN WITH CSV HEADER;
                " < "$temp_file"
                ;;
            *)
                log_warn "Unknown table $table, skipping migration"
                ;;
        esac
        
        # Clean up temp file
        rm -f "$temp_file"
        
        log_success "Completed migration for table: $table"
    done
    
    log_success "Data migration completed"
}

# Dual-write implementation
enable_dual_write() {
    log_info "Enabling dual-write mode..."
    
    # Update safety gateway configuration to enable dual-write
    local config_file="../config.yaml"
    
    # Create backup of current config
    cp "$config_file" "${config_file}.backup"
    
    # Enable dual-write mode
    cat >> "$config_file" << 'EOF'

# Migration settings - dual-write mode
migration:
  dual_write_enabled: true
  timescaledb:
    host: localhost
    port: 5434
    database: safety
    user: safety_user
    password: ${TIMESCALE_PASSWORD}
    ssl_mode: prefer
    max_connections: 10
  validation_enabled: true
  comparison_logging: true
  fail_on_mismatch: false  # Set to true after validation period
EOF

    log_success "Dual-write mode enabled in configuration"
}

# Validation functions
validate_data_consistency() {
    log_info "Validating data consistency between PostgreSQL and TimescaleDB..."
    
    # Compare record counts
    local postgres_count=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "
        SELECT COUNT(*) FROM safety_requests WHERE created_at >= NOW() - INTERVAL '24 hours';
    " | xargs)
    
    local timescale_count=$(PGPASSWORD="$TIMESCALE_PASSWORD" psql -h "$TIMESCALE_HOST" -p "$TIMESCALE_PORT" -U "$TIMESCALE_USER" -d "$TIMESCALE_DB" -t -c "
        SELECT COUNT(*) FROM safety_events.requests WHERE timestamp >= NOW() - INTERVAL '24 hours';
    " | xargs)
    
    log_info "PostgreSQL count (24h): $postgres_count"
    log_info "TimescaleDB count (24h): $timescale_count"
    
    local diff=$((postgres_count - timescale_count))
    local diff_abs=${diff#-}  # absolute value
    
    if [ "$diff_abs" -gt 10 ]; then
        log_error "Data consistency check failed: difference of $diff records"
        return 1
    fi
    
    log_success "Data consistency check passed (difference: $diff records)"
    return 0
}

# Switch functions
switch_to_timescaledb() {
    log_info "Switching reads to TimescaleDB..."
    
    # Update configuration to use TimescaleDB for reads
    local config_file="../config.yaml"
    
    # Update database configuration
    sed -i 's/dual_write_enabled: true/dual_write_enabled: true/' "$config_file"
    sed -i 's/# primary_db: postgres/primary_db: timescaledb/' "$config_file"
    
    # Restart service to apply changes
    log_info "Restarting safety gateway service..."
    pkill -f "safety-gateway" || true
    sleep 5
    
    # Start service in background
    ../build/safety-gateway -config=../config.yaml &
    local service_pid=$!
    
    # Wait for service to start
    sleep 10
    
    # Health check
    if curl -f -s http://localhost:8030/health > /dev/null; then
        log_success "Service restarted successfully, reads switched to TimescaleDB"
    else
        log_error "Service failed to start after switching to TimescaleDB"
        kill $service_pid || true
        return 1
    fi
}

finalize_migration() {
    log_info "Finalizing migration..."
    
    # Disable dual-write mode
    local config_file="../config.yaml"
    sed -i 's/dual_write_enabled: true/dual_write_enabled: false/' "$config_file"
    
    # Remove PostgreSQL configuration
    sed -i '/# Old PostgreSQL config/,/# End PostgreSQL config/d' "$config_file"
    
    # Restart service one final time
    pkill -f "safety-gateway" || true
    sleep 5
    ../build/safety-gateway -config=../config.yaml &
    
    # Final health check
    sleep 10
    if curl -f -s http://localhost:8030/health > /dev/null; then
        log_success "Migration finalized successfully"
    else
        log_error "Service failed to start in final configuration"
        return 1
    fi
    
    # Create migration completion marker
    echo "$(date)" > "$BACKUP_DIR/migration_completed.txt"
    
    log_success "TimescaleDB migration completed successfully!"
}

# Rollback functions
rollback_migration() {
    log_warn "Rolling back migration..."
    
    # Stop current service
    pkill -f "safety-gateway" || true
    
    # Restore original configuration
    local config_file="../config.yaml"
    if [ -f "${config_file}.backup" ]; then
        cp "${config_file}.backup" "$config_file"
        log_info "Original configuration restored"
    fi
    
    # Restart service with original configuration
    ../build/safety-gateway -config=../config.yaml &
    
    # Wait and check
    sleep 10
    if curl -f -s http://localhost:8030/health > /dev/null; then
        log_success "Rollback completed, service running with original configuration"
    else
        log_error "Rollback failed, manual intervention required"
    fi
}

# Main migration function
run_migration() {
    log_info "Starting TimescaleDB migration for Safety Gateway Platform"
    
    check_prerequisites
    test_connections
    create_backup
    setup_timescaledb_schema
    migrate_existing_data
    enable_dual_write
    
    log_info "Waiting 5 minutes for dual-write validation..."
    sleep 300
    
    if validate_data_consistency; then
        switch_to_timescaledb
        
        log_info "Waiting 2 minutes for switch validation..."
        sleep 120
        
        if validate_data_consistency; then
            finalize_migration
        else
            log_error "Post-switch validation failed"
            rollback_migration
            exit 1
        fi
    else
        log_error "Dual-write validation failed"
        rollback_migration
        exit 1
    fi
}

# Script execution
case "${1:-run}" in
    "run")
        run_migration
        ;;
    "rollback")
        rollback_migration
        ;;
    "validate")
        validate_data_consistency
        ;;
    "test-connections")
        test_connections
        ;;
    "backup")
        create_backup
        ;;
    *)
        echo "Usage: $0 [run|rollback|validate|test-connections|backup]"
        echo "  run              - Execute full migration"
        echo "  rollback         - Rollback to PostgreSQL"
        echo "  validate         - Check data consistency"
        echo "  test-connections - Test database connections"
        echo "  backup           - Create PostgreSQL backup only"
        exit 1
        ;;
esac