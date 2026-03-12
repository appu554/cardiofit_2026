-- ClickHouse Analytics Tables for Clinical Events
-- Database: module8_analytics
-- Purpose: OLAP analytics on enriched clinical events with columnar storage

-- Main clinical events fact table
CREATE TABLE IF NOT EXISTS clinical_events_fact (
    event_id String,
    patient_id String,
    timestamp DateTime,
    event_type String,
    department_id String,
    heart_rate Nullable(UInt16),
    bp_systolic Nullable(UInt16),
    bp_diastolic Nullable(UInt16),
    spo2 Nullable(UInt16),
    temperature Nullable(Float32),
    news2_score Nullable(UInt8),
    qsofa_score Nullable(UInt8),
    risk_level String,
    event_data String  -- JSON as string for full event context
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (patient_id, timestamp)
TTL timestamp + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;

-- ML predictions fact table
CREATE TABLE IF NOT EXISTS ml_predictions_fact (
    event_id String,
    patient_id String,
    timestamp DateTime,
    sepsis_risk_24h Nullable(Float32),
    cardiac_risk_7d Nullable(Float32),
    readmission_risk_30d Nullable(Float32),
    prediction_data String  -- JSON with full prediction details
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (patient_id, timestamp)
SETTINGS index_granularity = 8192;

-- Clinical alerts fact table
CREATE TABLE IF NOT EXISTS alerts_fact (
    event_id String,
    patient_id String,
    timestamp DateTime,
    alert_type String,
    severity String,
    department_id String,
    response_time_seconds Nullable(UInt32)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp, severity)
SETTINGS index_granularity = 8192;

-- Materialized view for daily aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_patient_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (patient_id, day)
AS SELECT
    patient_id,
    toDate(timestamp) as day,
    count() as event_count,
    avg(heart_rate) as avg_heart_rate,
    avg(bp_systolic) as avg_bp_systolic,
    avg(news2_score) as avg_news2_score,
    countIf(risk_level IN ('HIGH', 'CRITICAL')) as high_risk_events
FROM clinical_events_fact
GROUP BY patient_id, day;

-- Materialized view for hourly department stats
CREATE MATERIALIZED VIEW IF NOT EXISTS hourly_department_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (department_id, hour)
AS SELECT
    department_id,
    toStartOfHour(timestamp) as hour,
    count() as event_count,
    countIf(risk_level = 'CRITICAL') as critical_events,
    countIf(risk_level = 'HIGH') as high_risk_events,
    avg(news2_score) as avg_news2_score
FROM clinical_events_fact
GROUP BY department_id, hour;
