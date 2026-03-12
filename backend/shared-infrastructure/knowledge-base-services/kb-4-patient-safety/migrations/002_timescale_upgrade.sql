-- KB-4 Patient Safety: TimescaleDB Migration
-- Migrates from basic PostgreSQL to TimescaleDB with hypertables
-- Zero-downtime migration using dual-write pattern

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Enhanced safety alerts hypertable for time-series data
CREATE TABLE IF NOT EXISTS safety_alerts_v2 (
    event_id          UUID NOT NULL,
    ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
    patient_id        TEXT NOT NULL,
    therapy_id        TEXT NOT NULL,
    drug_code         TEXT NOT NULL,
    drug_class        TEXT NOT NULL,
    
    -- Core safety verdict
    safety_status     TEXT NOT NULL CHECK (safety_status IN ('PASS','WARN','VETO')),
    findings          JSONB NOT NULL DEFAULT '[]',
    
    -- Evidence and audit trail
    evidence_envelope JSONB NOT NULL DEFAULT '{}',
    decision_hash     TEXT NOT NULL,
    
    -- Clinical context snapshot
    patient_snapshot  JSONB NOT NULL DEFAULT '{}',
    concurrent_meds   TEXT[] DEFAULT '{}',
    
    -- Override state management
    override_state    TEXT NOT NULL DEFAULT 'none',
    override_id       UUID,
    
    -- Performance metrics
    evaluation_ms     INTEGER NOT NULL DEFAULT 0,
    cache_hit         BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    PRIMARY KEY (event_id, ts)
);

-- Create hypertable with daily chunks for optimal performance
SELECT create_hypertable(
    'safety_alerts_v2', 
    'ts', 
    chunk_time_interval => INTERVAL '1 day',
    create_default_indexes => TRUE,
    if_not_exists => TRUE
);

-- Compression policy for data older than 30 days
SELECT add_compression_policy(
    'safety_alerts_v2', 
    INTERVAL '30 days',
    if_not_exists => TRUE
);

-- Retention policy for 7-year compliance requirement
SELECT add_retention_policy(
    'safety_alerts_v2', 
    INTERVAL '7 years',
    if_not_exists => TRUE
);

-- Performance indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_patient_ts 
    ON safety_alerts_v2 (patient_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_drug_ts 
    ON safety_alerts_v2 (drug_code, ts DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_status_ts 
    ON safety_alerts_v2 (safety_status, ts DESC);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_override 
    ON safety_alerts_v2 (override_state) WHERE override_state != 'none';

-- GIN index for JSONB fields
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_findings_gin 
    ON safety_alerts_v2 USING GIN (findings);
CREATE INDEX IF NOT EXISTS idx_safety_alerts_v2_evidence_gin 
    ON safety_alerts_v2 USING GIN (evidence_envelope);

-- Continuous aggregates for real-time analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', ts) AS hour,
    drug_code,
    drug_class,
    safety_status,
    COUNT(*) as alert_count,
    COUNT(*) FILTER (WHERE safety_status = 'VETO') as veto_count,
    COUNT(*) FILTER (WHERE safety_status = 'WARN') as warn_count,
    COUNT(*) FILTER (WHERE override_state != 'none') as override_count,
    AVG(evaluation_ms) as avg_evaluation_time,
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::FLOAT / COUNT(*) as cache_hit_rate
FROM safety_alerts_v2
GROUP BY hour, drug_code, drug_class, safety_status
ORDER BY hour DESC;

-- Refresh policy for continuous aggregates
SELECT add_continuous_aggregate_policy(
    'safety_alerts_hourly',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- Daily aggregates for longer-term trends
CREATE MATERIALIZED VIEW IF NOT EXISTS safety_alerts_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', ts) AS day,
    drug_code,
    drug_class,
    COUNT(*) as total_alerts,
    COUNT(*) FILTER (WHERE safety_status = 'VETO') as total_vetos,
    COUNT(*) FILTER (WHERE safety_status = 'WARN') as total_warnings,
    COUNT(*) FILTER (WHERE override_state = 'L3') as l3_overrides,
    AVG(evaluation_ms) as avg_evaluation_time,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY evaluation_ms) as p95_evaluation_time
FROM safety_alerts_v2
GROUP BY day, drug_code, drug_class
ORDER BY day DESC;

-- Refresh policy for daily aggregates
SELECT add_continuous_aggregate_policy(
    'safety_alerts_daily',
    start_offset => INTERVAL '7 days',
    end_offset => INTERVAL '1 day', 
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Migration function for existing data
CREATE OR REPLACE FUNCTION migrate_safety_alerts_to_v2()
RETURNS VOID AS $$
DECLARE
    batch_size INT := 1000;
    offset_val INT := 0;
    record_count INT;
BEGIN
    LOOP
        -- Insert batch from old table to new hypertable
        INSERT INTO safety_alerts_v2 (
            event_id, ts, patient_id, therapy_id, drug_code, drug_class,
            safety_status, findings, evidence_envelope, decision_hash,
            patient_snapshot, concurrent_meds, evaluation_ms, cache_hit
        )
        SELECT 
            COALESCE(id, gen_random_uuid()), -- Handle missing IDs
            COALESCE(created_at, now()),     -- Handle missing timestamps
            patient_id,
            COALESCE(therapy_id, 'LEGACY_' || id::text),
            drug_code,
            COALESCE(drug_class, 'UNKNOWN'),
            COALESCE(status, 'PASS'),
            COALESCE(alert_data, '[]'::jsonb),
            COALESCE(metadata, '{}'::jsonb),
            COALESCE(
                encode(sha256(concat(patient_id, drug_code, created_at)::bytea), 'hex'),
                md5(random()::text)
            ),
            COALESCE(context_data, '{}'::jsonb),
            string_to_array(COALESCE(active_medications, ''), ','),
            COALESCE(processing_time_ms, 50),
            false -- Default to cache miss for legacy data
        FROM safety_alerts
        ORDER BY created_at
        LIMIT batch_size OFFSET offset_val;
        
        GET DIAGNOSTICS record_count = ROW_COUNT;
        
        -- Exit when no more records
        IF record_count = 0 THEN
            EXIT;
        END IF;
        
        offset_val := offset_val + batch_size;
        
        -- Log progress every 10k records
        IF offset_val % 10000 = 0 THEN
            RAISE NOTICE 'Migrated % records', offset_val;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Migration completed. Total records migrated: %', offset_val;
END;
$$ LANGUAGE plpgsql;

-- Validation function to compare data integrity
CREATE OR REPLACE FUNCTION validate_migration_integrity()
RETURNS TABLE(
    table_name TEXT,
    record_count BIGINT,
    integrity_status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'safety_alerts'::TEXT,
        COUNT(*)::BIGINT,
        'original'::TEXT
    FROM safety_alerts
    
    UNION ALL
    
    SELECT 
        'safety_alerts_v2'::TEXT,
        COUNT(*)::BIGINT,
        'migrated'::TEXT
    FROM safety_alerts_v2;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for dual-write during migration period
CREATE OR REPLACE FUNCTION sync_to_safety_alerts_v2()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert into new hypertable on any insert to old table
    INSERT INTO safety_alerts_v2 (
        event_id, ts, patient_id, therapy_id, drug_code, drug_class,
        safety_status, findings, evidence_envelope, decision_hash,
        patient_snapshot, concurrent_meds, evaluation_ms, cache_hit
    ) VALUES (
        COALESCE(NEW.id, gen_random_uuid()),
        COALESCE(NEW.created_at, now()),
        NEW.patient_id,
        COALESCE(NEW.therapy_id, 'SYNC_' || NEW.id::text),
        NEW.drug_code,
        COALESCE(NEW.drug_class, 'UNKNOWN'),
        COALESCE(NEW.status, 'PASS'),
        COALESCE(NEW.alert_data, '[]'::jsonb),
        COALESCE(NEW.metadata, '{}'::jsonb),
        COALESCE(
            encode(sha256(concat(NEW.patient_id, NEW.drug_code, NEW.created_at)::bytea), 'hex'),
            md5(random()::text)
        ),
        COALESCE(NEW.context_data, '{}'::jsonb),
        string_to_array(COALESCE(NEW.active_medications, ''), ','),
        COALESCE(NEW.processing_time_ms, 50),
        false
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for dual-write
DROP TRIGGER IF EXISTS safety_alerts_sync_trigger ON safety_alerts;
CREATE TRIGGER safety_alerts_sync_trigger
    AFTER INSERT ON safety_alerts
    FOR EACH ROW
    EXECUTE FUNCTION sync_to_safety_alerts_v2();

-- Comments for documentation
COMMENT ON TABLE safety_alerts_v2 IS 'Enhanced safety alerts hypertable with time-series optimization';
COMMENT ON COLUMN safety_alerts_v2.evidence_envelope IS 'Contains tamper-evident evidence linking to KB-3 guidelines';
COMMENT ON COLUMN safety_alerts_v2.decision_hash IS 'SHA-256 hash for tamper detection';
COMMENT ON COLUMN safety_alerts_v2.patient_snapshot IS 'Complete patient context at decision time';
COMMENT ON COLUMN safety_alerts_v2.override_state IS 'Override authorization level: none, L1, L2, L3';

-- Grant appropriate permissions
GRANT SELECT, INSERT, UPDATE ON safety_alerts_v2 TO kb4_service;
GRANT SELECT ON safety_alerts_hourly TO kb4_service;
GRANT SELECT ON safety_alerts_daily TO kb4_service;
GRANT EXECUTE ON FUNCTION migrate_safety_alerts_to_v2() TO kb4_admin;
GRANT EXECUTE ON FUNCTION validate_migration_integrity() TO kb4_service;