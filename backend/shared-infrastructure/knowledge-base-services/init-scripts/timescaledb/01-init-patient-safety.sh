#!/bin/bash

# KB-4 Patient Safety TimescaleDB Initialization Script
# This script sets up TimescaleDB for the Patient Safety Knowledge Base

set -e

echo "Initializing TimescaleDB for KB-4 Patient Safety..."

# Database connection parameters
DB_HOST=${TIMESCALE_HOST:-localhost}
DB_PORT=${TIMESCALE_PORT:-5432}
DB_NAME=${TIMESCALE_DB:-kb_patient_safety}
DB_USER=${TIMESCALE_USER:-kb4_user}
DB_PASSWORD=${TIMESCALE_PASSWORD:-password}

# Wait for TimescaleDB to be ready
echo "Waiting for TimescaleDB to be ready..."
until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do
  echo "TimescaleDB is unavailable - sleeping"
  sleep 2
done

echo "TimescaleDB is ready!"

# Create database if it doesn't exist
echo "Creating database if not exists..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME"

# Run the main schema migration
echo "Running schema migrations..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f /migrations/001_initial_schema.sql

# Insert sample data for development
if [ "$ENVIRONMENT" = "development" ]; then
    echo "Inserting sample data for development environment..."
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    -- Sample safety alerts
    INSERT INTO safety_alerts (
        time, patient_id, alert_type, severity, description, 
        source_system, triggering_values, recommendations
    ) VALUES 
    (
        NOW() - INTERVAL '2 hours',
        'patient-001',
        'critical_bp',
        'critical',
        'Critical hypertension detected: 190/120 mmHg',
        'vital_signs_monitor',
        '{\"systolic_bp\": 190, \"diastolic_bp\": 120}',
        '{\"immediate_action\": \"Contact physician immediately\", \"medication_review\": true}'
    ),
    (
        NOW() - INTERVAL '4 hours',
        'patient-002',
        'drug_interaction',
        'high',
        'High-risk drug interaction: Warfarin + Aspirin',
        'medication_system',
        '{\"drug1\": \"warfarin\", \"drug2\": \"aspirin\", \"interaction_level\": \"high\"}',
        '{\"action\": \"Review dosing\", \"monitoring\": \"PT/INR daily\"}'
    ),
    (
        NOW() - INTERVAL '1 day',
        'patient-003',
        'fall_risk',
        'medium',
        'Elevated fall risk score: 8/10',
        'risk_assessment',
        '{\"fall_score\": 8, \"age\": 78, \"medication_count\": 12}',
        '{\"interventions\": [\"bed alarm\", \"fall prevention protocol\"]}'
    ),
    (
        NOW() - INTERVAL '6 hours',
        'patient-001',
        'lab_critical',
        'critical',
        'Critical potassium level: 2.1 mEq/L',
        'lab_system',
        '{\"potassium\": 2.1, \"normal_range\": \"3.5-5.0\"}',
        '{\"immediate_action\": \"IV potassium replacement\", \"notify\": \"physician\"}'
    ),
    (
        NOW() - INTERVAL '30 minutes',
        'patient-004',
        'respiratory_distress',
        'critical',
        'Low oxygen saturation: 85% on room air',
        'vital_signs_monitor',
        '{\"spo2\": 85, \"respiratory_rate\": 28}',
        '{\"immediate_action\": \"Apply oxygen\", \"escalate\": \"rapid_response\"}'
    );

    -- Sample monitoring events
    INSERT INTO safety_monitoring_events (
        time, patient_id, event_type, event_source, event_data, risk_score, flagged
    ) VALUES
    (
        NOW() - INTERVAL '30 minutes',
        'patient-001',
        'vital_signs',
        'bedside_monitor',
        '{\"systolic_bp\": 185, \"diastolic_bp\": 110, \"heart_rate\": 95, \"temperature\": 98.6}',
        0.85,
        true
    ),
    (
        NOW() - INTERVAL '1 hour',
        'patient-002',
        'medication_administration',
        'emar_system',
        '{\"medication\": \"warfarin\", \"dose\": \"5mg\", \"route\": \"oral\", \"time\": \"08:00\"}',
        0.45,
        false
    ),
    (
        NOW() - INTERVAL '2 hours',
        'patient-003',
        'ambulation_attempt',
        'nursing_documentation',
        '{\"distance\": \"10_feet\", \"assistance\": \"minimal\", \"stability\": \"unsteady\"}',
        0.75,
        true
    ),
    (
        NOW() - INTERVAL '15 minutes',
        'patient-004',
        'respiratory_assessment',
        'nursing_documentation',
        '{\"respiratory_rate\": 24, \"spo2\": 92, \"oxygen_therapy\": \"2L_nasal_cannula\"}',
        0.65,
        true
    ),
    (
        NOW() - INTERVAL '45 minutes',
        'patient-005',
        'pain_assessment',
        'nursing_documentation',
        '{\"pain_score\": 8, \"pain_location\": \"chest\", \"quality\": \"sharp\"}',
        0.70,
        true
    );

    -- Sample patient risk profiles
    INSERT INTO patient_risk_profiles (
        patient_id, risk_scores, risk_factors, contraindications, safety_flags
    ) VALUES
    (
        'patient-001',
        '{\"fall_risk\": 0.65, \"readmission_risk\": 0.40, \"adverse_drug_event_risk\": 0.55, \"mortality_risk\": 0.25}',
        '{\"age\": 72, \"comorbidities\": [\"hypertension\", \"diabetes\"], \"medication_count\": 8}',
        ARRAY['contrast_agents', 'nsaids'],
        '{\"hypertension_alert\": true, \"poly_pharmacy\": true}'
    ),
    (
        'patient-002',
        '{\"fall_risk\": 0.30, \"readmission_risk\": 0.60, \"adverse_drug_event_risk\": 0.80, \"mortality_risk\": 0.20}',
        '{\"age\": 65, \"anticoagulant_therapy\": true, \"bleeding_history\": true}',
        ARRAY['antiplatelet_agents', 'thrombolytics'],
        '{\"bleeding_risk\": true, \"anticoagulation_monitoring\": true}'
    ),
    (
        'patient-003',
        '{\"fall_risk\": 0.85, \"readmission_risk\": 0.35, \"adverse_drug_event_risk\": 0.45, \"mortality_risk\": 0.15}',
        '{\"age\": 78, \"mobility_impaired\": true, \"cognitive_impairment\": \"mild\"}',
        ARRAY['sedatives', 'muscle_relaxants'],
        '{\"fall_prevention\": true, \"bed_alarm\": true}'
    );

    -- Sample drug safety contraindications
    INSERT INTO drug_safety_contraindications (
        drug_code, drug_name, contraindication_type, contraindication_code,
        contraindication_description, severity, evidence_level, clinical_context
    ) VALUES
    (
        'warfarin',
        'Warfarin Sodium',
        'drug_interaction',
        'aspirin_interaction',
        'Increased bleeding risk when combined with aspirin',
        'absolute',
        'high',
        '{\"mechanism\": \"synergistic_anticoagulation\", \"monitoring\": \"PT/INR_daily\"}'
    ),
    (
        'metformin',
        'Metformin HCl',
        'renal_impairment',
        'gfr_less_than_30',
        'Contraindicated in severe renal impairment (GFR < 30)',
        'absolute',
        'high',
        '{\"risk\": \"lactic_acidosis\", \"alternative\": \"insulin\"}'
    ),
    (
        'nsaids',
        'Non-steroidal Anti-inflammatory Drugs',
        'cardiovascular_risk',
        'recent_mi',
        'Increased risk of cardiovascular events post-MI',
        'relative',
        'moderate',
        '{\"timeframe\": \"within_6_months\", \"alternatives\": [\"acetaminophen\", \"topical_preparations\"]}'
    );
    "

    echo "Sample data inserted successfully!"
fi

# Setup retention and compression policies
echo "Setting up retention and compression policies..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
-- Add compression policy for safety_alerts (compress chunks older than 7 days)
SELECT add_compression_policy('safety_alerts', INTERVAL '7 days', if_not_exists => true);

-- Add compression policy for safety_monitoring_events (compress chunks older than 3 days)  
SELECT add_compression_policy('safety_monitoring_events', INTERVAL '3 days', if_not_exists => true);

-- Refresh continuous aggregates to populate initial data
CALL refresh_continuous_aggregate('safety_alerts_hourly', NULL, NULL);
CALL refresh_continuous_aggregate('safety_alerts_daily', NULL, NULL);
CALL refresh_continuous_aggregate('patient_risk_trends_daily', NULL, NULL);
"

# Create indexes for performance optimization
echo "Creating additional performance indexes..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
-- Additional composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_safety_alerts_patient_severity_time 
    ON safety_alerts (patient_id, severity, time DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_safety_alerts_type_unresolved_time 
    ON safety_alerts (alert_type, time DESC) 
    WHERE resolved = FALSE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_safety_monitoring_patient_flagged_time 
    ON safety_monitoring_events (patient_id, time DESC) 
    WHERE flagged = TRUE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_safety_monitoring_risk_score_time 
    ON safety_monitoring_events (risk_score, time DESC) 
    WHERE risk_score > 0.7;

-- Partial indexes for active/recent data
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_safety_alerts_recent_active 
    ON safety_alerts (time DESC, severity) 
    WHERE time > NOW() - INTERVAL '24 hours' AND resolved = FALSE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_patient_risk_profiles_high_risk 
    ON patient_risk_profiles (patient_id) 
    WHERE (risk_scores->>'mortality_risk')::float > 0.5 
       OR (risk_scores->>'adverse_drug_event_risk')::float > 0.7;
"

# Setup database monitoring and alerting
echo "Setting up database monitoring..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
-- Create monitoring view for database health
CREATE OR REPLACE VIEW v_timescale_health AS
SELECT 
    'TimescaleDB' as component,
    pg_size_pretty(pg_database_size(current_database())) as database_size,
    (SELECT COUNT(*) FROM timescaledb_information.hypertables) as hypertable_count,
    (SELECT COUNT(*) FROM timescaledb_information.chunks) as total_chunks,
    (SELECT COUNT(*) FROM timescaledb_information.compressed_chunks) as compressed_chunks,
    (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active') as active_connections,
    NOW() as checked_at;

-- Create alert summary view
CREATE OR REPLACE VIEW v_current_safety_status AS
SELECT 
    'Patient Safety Status' as dashboard,
    (SELECT COUNT(*) FROM safety_alerts WHERE resolved = FALSE) as active_alerts,
    (SELECT COUNT(*) FROM safety_alerts WHERE resolved = FALSE AND severity = 'critical') as critical_alerts,
    (SELECT COUNT(*) FROM safety_alerts WHERE resolved = FALSE AND severity = 'high') as high_alerts,
    (SELECT COUNT(DISTINCT patient_id) FROM safety_alerts WHERE time > NOW() - INTERVAL '24 hours') as patients_with_alerts_24h,
    (SELECT COUNT(*) FROM safety_monitoring_events WHERE processed = FALSE) as unprocessed_events,
    NOW() as status_time;
"

# Setup scheduled maintenance
echo "Creating maintenance functions..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
-- Create maintenance function
CREATE OR REPLACE FUNCTION perform_safety_db_maintenance()
RETURNS TEXT AS \$\$
DECLARE
    result TEXT := '';
    chunk_count INTEGER;
BEGIN
    -- Compress old chunks
    SELECT COUNT(*) INTO chunk_count
    FROM (
        SELECT compress_chunk(chunk)
        FROM show_chunks('safety_alerts')
        WHERE range_end < NOW() - INTERVAL '7 days'
        AND NOT is_compressed
    ) compressed;
    
    result := result || 'Compressed ' || chunk_count || ' safety_alerts chunks. ';
    
    -- Compress monitoring events
    SELECT COUNT(*) INTO chunk_count
    FROM (
        SELECT compress_chunk(chunk)
        FROM show_chunks('safety_monitoring_events')
        WHERE range_end < NOW() - INTERVAL '3 days'
        AND NOT is_compressed
    ) compressed;
    
    result := result || 'Compressed ' || chunk_count || ' monitoring_events chunks. ';
    
    -- Update table statistics
    ANALYZE safety_alerts;
    ANALYZE safety_monitoring_events;
    ANALYZE patient_risk_profiles;
    
    result := result || 'Updated table statistics. ';
    
    -- Log maintenance completion
    INSERT INTO safety_alert_audit (alert_id, action, performed_by, notes)
    VALUES (gen_random_uuid(), 'maintenance', 'system', result);
    
    RETURN result;
END;
\$\$ LANGUAGE plpgsql;

-- Grant necessary permissions
GRANT USAGE ON SCHEMA public TO $DB_USER;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO $DB_USER;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;
GRANT EXECUTE ON FUNCTION perform_safety_db_maintenance() TO $DB_USER;
"

# Create backup script
echo "Creating backup configuration..."
cat > /tmp/kb4_backup.sh << EOF
#!/bin/bash
# KB-4 Patient Safety Database Backup Script

BACKUP_DIR=\${BACKUP_DIR:-/backups/kb4}
TIMESTAMP=\$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="\${BACKUP_DIR}/kb4_patient_safety_\${TIMESTAMP}.sql"

# Create backup directory
mkdir -p \$BACKUP_DIR

# Perform backup
PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME > \$BACKUP_FILE

# Compress backup
gzip \$BACKUP_FILE

echo "Backup completed: \${BACKUP_FILE}.gz"

# Keep only last 7 days of backups
find \$BACKUP_DIR -name "kb4_patient_safety_*.sql.gz" -mtime +7 -delete

echo "Old backups cleaned up"
EOF

chmod +x /tmp/kb4_backup.sh

echo "TimescaleDB initialization completed successfully!"

# Display status information
echo "========================================="
echo "KB-4 Patient Safety TimescaleDB Status"
echo "========================================="

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
SELECT * FROM v_timescale_health;
SELECT * FROM v_current_safety_status;

-- Show recent activity
SELECT 'Recent Alerts' as info, COUNT(*) as count 
FROM safety_alerts 
WHERE time > NOW() - INTERVAL '1 hour';

SELECT 'Unprocessed Events' as info, COUNT(*) as count 
FROM safety_monitoring_events 
WHERE processed = FALSE;

SELECT 'High Risk Patients' as info, COUNT(*) as count 
FROM patient_risk_profiles 
WHERE (risk_scores->>'mortality_risk')::float > 0.5;
"

echo "========================================="
echo "Initialization complete!"
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo "User: $DB_USER"
echo "========================================="