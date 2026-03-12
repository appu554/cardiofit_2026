-- ClickHouse Schema for Large-Scale KB Analytics
-- Part II: Real-Time Analytics Platform - Large-scale cross-KB analytics

-- Clinical decisions analytics table - Main fact table
CREATE TABLE IF NOT EXISTS clinical_decisions_analytics (
    -- Primary keys and identifiers
    transaction_id String,
    evidence_envelope_id String,
    timestamp DateTime64(3) DEFAULT now64(),
    
    -- Request context
    patient_id String,
    user_id String,
    session_id String,
    encounter_type String,
    clinical_domain String,
    request_type String,
    
    -- Geographic and demographic context
    region String DEFAULT 'US',
    patient_age_group String, -- '0-17', '18-34', '35-64', '65-74', '75+'
    patient_gender String, -- 'M', 'F', 'Other', 'Unknown'
    
    -- Clinical context
    primary_conditions Array(String),
    secondary_conditions Array(String),
    active_medications Array(String),
    allergies Array(String),
    
    -- KB interactions - Nested structure for better analytics
    kb_calls Nested (
        kb_name String,
        kb_version String,
        endpoint String,
        request_size UInt32,
        response_size UInt32,
        latency_ms UInt32,
        cache_hit UInt8,
        error_count UInt16,
        confidence_score Float32,
        result_count UInt32
    ),
    
    -- Aggregated outcomes
    total_kb_calls UInt16,
    successful_kb_calls UInt16,
    failed_kb_calls UInt16,
    recommendations Array(String),
    safety_alerts Array(String),
    clinical_overrides Array(String),
    
    -- Performance metrics
    total_latency_ms UInt32,
    max_kb_latency_ms UInt32,
    cache_hit_rate Float32,
    evidence_envelope_size_bytes UInt32,
    
    -- Quality metrics
    guideline_adherence_score Float32,
    safety_score Float32,
    confidence_score Float32,
    completeness_score Float32,
    consistency_score Float32,
    
    -- Version tracking
    version_set_id String,
    kb_versions String, -- JSON string of version mapping
    version_override_used UInt8,
    
    -- Outcome tracking
    decision_accepted UInt8,
    override_reason String,
    clinical_outcome String, -- 'positive', 'negative', 'neutral', 'unknown'
    
    -- Partitioning and sorting
    date Date MATERIALIZED toDate(timestamp),
    hour DateTime MATERIALIZED toStartOfHour(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (clinical_domain, timestamp, transaction_id)
TTL date + INTERVAL 2 YEAR;

-- KB performance metrics table
CREATE TABLE IF NOT EXISTS kb_performance_metrics (
    -- Time dimensions
    timestamp DateTime64(3) DEFAULT now64(),
    hour DateTime MATERIALIZED toStartOfHour(timestamp),
    date Date MATERIALIZED toDate(timestamp),
    
    -- KB identification
    kb_name String,
    kb_version String,
    endpoint String,
    
    -- Request metrics
    request_count UInt32,
    successful_requests UInt32,
    failed_requests UInt32,
    
    -- Latency metrics (in milliseconds)
    min_latency_ms UInt16,
    max_latency_ms UInt32,
    avg_latency_ms Float32,
    p50_latency_ms Float32,
    p95_latency_ms Float32,
    p99_latency_ms Float32,
    
    -- Cache metrics
    cache_hit_count UInt32,
    cache_miss_count UInt32,
    cache_hit_rate Float32,
    
    -- Error analysis
    error_types Array(String),
    error_counts Array(UInt16),
    
    -- Resource utilization
    avg_cpu_usage Float32,
    avg_memory_usage Float32,
    concurrent_requests_avg Float32,
    concurrent_requests_max UInt16,
    
    -- Quality metrics
    avg_confidence_score Float32,
    avg_result_count Float32,
    
    -- Business metrics
    clinical_decisions_influenced UInt32,
    patient_safety_alerts_generated UInt16
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (kb_name, hour, kb_version)
TTL date + INTERVAL 1 YEAR;

-- Cross-KB correlation analysis table
CREATE TABLE IF NOT EXISTS kb_correlation_matrix (
    -- Time dimension
    timestamp DateTime64(3) DEFAULT now64(),
    date Date MATERIALIZED toDate(timestamp),
    
    -- KB pair identification
    kb1_name String,
    kb1_version String,
    kb2_name String,
    kb2_version String,
    
    -- Correlation metrics
    co_occurrence_count UInt32,
    correlation_type String, -- 'positive', 'negative', 'neutral', 'conflicting'
    
    -- Performance correlation
    latency_correlation Float32,
    cache_correlation Float32,
    error_correlation Float32,
    
    -- Clinical correlation
    recommendation_alignment Float32,
    safety_consistency Float32,
    
    -- Conflict analysis
    conflict_count UInt16,
    conflict_severity String, -- 'critical', 'major', 'moderate', 'minor'
    
    -- Patient outcomes
    combined_confidence_score Float32,
    decision_success_rate Float32
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (kb1_name, kb2_name, date)
TTL date + INTERVAL 1 YEAR;

-- Safety signal analytics table
CREATE TABLE IF NOT EXISTS safety_signal_analytics (
    -- Time dimensions
    timestamp DateTime64(3) DEFAULT now64(),
    date Date MATERIALIZED toDate(timestamp),
    hour DateTime MATERIALIZED toStartOfHour(timestamp),
    
    -- Signal identification
    signal_id String,
    signal_type String,
    kb_source String,
    severity String,
    
    -- Clinical context
    clinical_domain String,
    patient_demographics Map(String, String),
    conditions Array(String),
    medications Array(String),
    
    -- Signal details
    detection_method String,
    confidence_score Float32,
    false_positive_probability Float32,
    
    -- Impact metrics
    estimated_patient_impact UInt32,
    clinical_significance_score Float32,
    urgency_score Float32,
    
    -- Response metrics
    detection_to_acknowledgment_minutes UInt32,
    acknowledgment_to_resolution_minutes UInt32,
    total_resolution_time_minutes UInt32,
    
    -- Actions taken
    automated_actions_count UInt8,
    manual_actions_count UInt8,
    escalations_count UInt8,
    
    -- Outcomes
    resolution_status String,
    actual_outcome String, -- 'true_positive', 'false_positive', 'indeterminate'
    
    -- Geographic
    region String DEFAULT 'US'
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (signal_type, timestamp, signal_id)
TTL date + INTERVAL 3 YEAR;

-- Patient journey analytics table
CREATE TABLE IF NOT EXISTS patient_journey_analytics (
    -- Patient identification (hashed for privacy)
    patient_hash String,
    journey_id String,
    
    -- Time tracking
    timestamp DateTime64(3) DEFAULT now64(),
    date Date MATERIALIZED toDate(timestamp),
    
    -- Journey stage
    stage String, -- 'initial_assessment', 'diagnosis', 'treatment_planning', 'monitoring', 'outcome'
    stage_sequence UInt8,
    
    -- Clinical context
    primary_condition String,
    comorbidities Array(String),
    medications Array(String),
    
    -- KB interactions in this stage
    kb_services_used Array(String),
    total_kb_calls UInt16,
    stage_duration_minutes UInt32,
    
    -- Decision outcomes
    recommendations_followed Array(String),
    recommendations_overridden Array(String),
    safety_alerts_triggered Array(String),
    
    -- Quality metrics
    guideline_adherence_percentage Float32,
    safety_compliance_percentage Float32,
    
    -- Outcomes
    clinical_outcome String,
    patient_satisfaction_score Float32,
    readmission_risk_score Float32,
    
    -- Demographic context (anonymized)
    age_group String,
    gender String,
    region String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (patient_hash, stage_sequence, timestamp)
TTL date + INTERVAL 7 YEAR; -- Longer retention for longitudinal analysis

-- Version deployment impact analysis
CREATE TABLE IF NOT EXISTS version_deployment_impact (
    -- Deployment tracking
    deployment_id String,
    deployment_timestamp DateTime64(3),
    
    -- Version information
    kb_name String,
    old_version String,
    new_version String,
    environment String,
    
    -- Impact measurement window
    measurement_timestamp DateTime64(3) DEFAULT now64(),
    hours_since_deployment UInt16,
    
    -- Performance impact
    request_count_change Float32, -- Percentage change
    latency_change_ms Int32,
    error_rate_change Float32,
    cache_hit_rate_change Float32,
    
    -- Clinical impact
    recommendation_accuracy_change Float32,
    safety_alert_rate_change Float32,
    guideline_adherence_change Float32,
    
    -- Usage patterns
    adoption_rate Float32,
    rollback_requests UInt8,
    
    -- Quality metrics
    confidence_score_change Float32,
    consistency_score_change Float32,
    
    -- User feedback
    positive_feedback_count UInt16,
    negative_feedback_count UInt16,
    neutral_feedback_count UInt16
) ENGINE = MergeTree()
ORDER BY (kb_name, deployment_timestamp, hours_since_deployment)
TTL deployment_timestamp + INTERVAL 1 YEAR;

-- Real-time dashboard materialized views

-- Current system health dashboard
CREATE MATERIALIZED VIEW system_health_realtime AS
SELECT
    toStartOfMinute(timestamp) as minute,
    clinical_domain,
    count() as total_requests,
    avg(total_latency_ms) as avg_latency_ms,
    quantile(0.95)(total_latency_ms) as p95_latency_ms,
    countIf(failed_kb_calls > 0) as failed_requests,
    avg(cache_hit_rate) as avg_cache_hit_rate,
    avg(confidence_score) as avg_confidence_score,
    countIf(length(safety_alerts) > 0) as safety_alerts_count
FROM clinical_decisions_analytics
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY minute, clinical_domain
ORDER BY minute DESC;

-- KB service performance dashboard
CREATE MATERIALIZED VIEW kb_performance_realtime AS
SELECT
    toStartOfMinute(timestamp) as minute,
    kb_calls.kb_name as kb_name,
    kb_calls.kb_version as kb_version,
    count() as request_count,
    avg(kb_calls.latency_ms) as avg_latency_ms,
    quantile(0.95)(kb_calls.latency_ms) as p95_latency_ms,
    avg(kb_calls.cache_hit) as cache_hit_rate,
    countIf(kb_calls.error_count > 0) as error_count,
    avg(kb_calls.confidence_score) as avg_confidence
FROM clinical_decisions_analytics
ARRAY JOIN kb_calls
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY minute, kb_name, kb_version
ORDER BY minute DESC, kb_name;

-- Cross-KB interaction patterns
CREATE MATERIALIZED VIEW kb_interaction_patterns AS
SELECT
    toStartOfHour(timestamp) as hour,
    kb1.kb_name as kb1_name,
    kb2.kb_name as kb2_name,
    count() as co_occurrence_count,
    avg(kb1.latency_ms + kb2.latency_ms) as combined_latency_ms,
    avg(kb1.confidence_score * kb2.confidence_score) as combined_confidence,
    countIf(kb1.error_count > 0 OR kb2.error_count > 0) as combined_errors
FROM clinical_decisions_analytics
ARRAY JOIN kb_calls as kb1
ARRAY JOIN kb_calls as kb2
WHERE kb1.kb_name < kb2.kb_name
  AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour, kb1_name, kb2_name
HAVING co_occurrence_count >= 10
ORDER BY hour DESC, co_occurrence_count DESC;

-- Patient safety monitoring
CREATE MATERIALIZED VIEW safety_monitoring_realtime AS
SELECT
    toStartOfHour(timestamp) as hour,
    clinical_domain,
    count() as total_decisions,
    countIf(length(safety_alerts) > 0) as decisions_with_alerts,
    (countIf(length(safety_alerts) > 0) * 100.0) / count() as alert_rate_percent,
    avg(safety_score) as avg_safety_score,
    countIf(safety_score < 0.5) as low_safety_score_count,
    arraySort(groupUniqArray(arrayJoin(safety_alerts))) as common_alert_types
FROM clinical_decisions_analytics
WHERE timestamp >= now() - INTERVAL 6 HOUR
GROUP BY hour, clinical_domain
ORDER BY hour DESC;

-- Clinical outcome tracking
CREATE MATERIALIZED VIEW clinical_outcomes_tracking AS
SELECT
    date,
    clinical_domain,
    count() as total_decisions,
    countIf(decision_accepted = 1) as accepted_decisions,
    (countIf(decision_accepted = 1) * 100.0) / count() as acceptance_rate_percent,
    avg(guideline_adherence_score) as avg_guideline_adherence,
    countIf(clinical_outcome = 'positive') as positive_outcomes,
    countIf(clinical_outcome = 'negative') as negative_outcomes,
    countIf(clinical_outcome = 'neutral') as neutral_outcomes,
    countIf(length(clinical_overrides) > 0) as overridden_decisions
FROM clinical_decisions_analytics
WHERE timestamp >= today() - INTERVAL 30 DAY
GROUP BY date, clinical_domain
ORDER BY date DESC;

-- Analytical functions for business intelligence

-- Function to calculate KB service reliability score
-- Note: ClickHouse doesn't support stored procedures, so this would be implemented in application layer
-- But we can create a reusable query pattern

-- KB Reliability Score Query Template
-- SELECT 
--     kb_name,
--     kb_version,
--     (successful_requests * 100.0) / (successful_requests + failed_requests) as availability_percent,
--     CASE 
--         WHEN avg_latency_ms <= 10 THEN 100
--         WHEN avg_latency_ms <= 50 THEN 90
--         WHEN avg_latency_ms <= 100 THEN 80
--         WHEN avg_latency_ms <= 500 THEN 70
--         ELSE 60
--     END as performance_score,
--     CASE
--         WHEN cache_hit_rate >= 0.95 THEN 100
--         WHEN cache_hit_rate >= 0.90 THEN 90
--         WHEN cache_hit_rate >= 0.85 THEN 80
--         WHEN cache_hit_rate >= 0.75 THEN 70
--         ELSE 60
--     END as cache_efficiency_score,
--     (availability_percent * 0.4 + performance_score * 0.3 + cache_efficiency_score * 0.3) as overall_reliability_score
-- FROM kb_performance_metrics
-- WHERE date >= today() - INTERVAL 7 DAY
-- GROUP BY kb_name, kb_version;

-- Sample data insertion script (for testing)
-- INSERT INTO clinical_decisions_analytics (
--     transaction_id, evidence_envelope_id, patient_id, user_id,
--     clinical_domain, request_type, kb_calls.kb_name, kb_calls.kb_version,
--     kb_calls.latency_ms, kb_calls.cache_hit, total_latency_ms,
--     guideline_adherence_score, safety_score, confidence_score
-- ) VALUES (
--     'txn_test_001', 'env_test_001', 'patient_123', 'user_456',
--     'cardiology', 'medication_recommendation', ['kb_1_dosing', 'kb_3_guidelines'],
--     ['1.0.0', '1.2.0'], [15, 45], [1, 0], 60,
--     0.95, 0.88, 0.92
-- );

-- Comments for documentation
-- Clinical decisions analytics: Main fact table for all KB-driven clinical decisions
-- KB performance metrics: Aggregated performance data for monitoring and optimization  
-- Safety signal analytics: Comprehensive tracking of safety-related events and responses
-- Patient journey analytics: Longitudinal patient care tracking across KB interactions