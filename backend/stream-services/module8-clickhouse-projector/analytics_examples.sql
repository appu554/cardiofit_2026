-- ClickHouse Analytics Query Examples
-- Real-time OLAP analytics on clinical events

-- ========================================
-- 1. REAL-TIME DASHBOARDS
-- ========================================

-- Current ICU Risk Status (last hour)
SELECT
    department_id,
    risk_level,
    count() as patient_count,
    round(avg(news2_score), 1) as avg_news2,
    round(avg(heart_rate), 0) as avg_hr
FROM clinical_events_fact
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY department_id, risk_level
ORDER BY department_id,
    CASE risk_level
        WHEN 'CRITICAL' THEN 1
        WHEN 'HIGH' THEN 2
        WHEN 'MODERATE' THEN 3
        ELSE 4
    END;

-- ========================================
-- 2. TREND ANALYSIS
-- ========================================

-- Hourly Event Trends (last 24 hours)
SELECT
    toStartOfHour(timestamp) as hour,
    count() as total_events,
    countIf(risk_level = 'CRITICAL') as critical_events,
    countIf(risk_level = 'HIGH') as high_events,
    round(avg(news2_score), 1) as avg_news2,
    round(avg(qsofa_score), 1) as avg_qsofa
FROM clinical_events_fact
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC;

-- Daily Patient Event Volume (last 30 days)
SELECT
    day,
    event_count,
    avg_heart_rate,
    high_risk_events,
    round(high_risk_events * 100.0 / event_count, 1) as high_risk_percentage
FROM daily_patient_stats_mv
WHERE day >= today() - INTERVAL 30 DAY
ORDER BY day DESC;

-- ========================================
-- 3. DEPARTMENT PERFORMANCE
-- ========================================

-- Department Risk Distribution (last 7 days)
SELECT
    department_id,
    count() as total_events,
    countIf(risk_level = 'CRITICAL') as critical_count,
    countIf(risk_level = 'HIGH') as high_count,
    round(countIf(risk_level IN ('CRITICAL', 'HIGH')) * 100.0 / count(), 1) as high_risk_pct,
    round(avg(news2_score), 1) as avg_news2
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
GROUP BY department_id
ORDER BY high_risk_pct DESC;

-- Hourly Department Load (from materialized view)
SELECT
    department_id,
    toStartOfHour(hour) as hour,
    sum(event_count) as events,
    sum(critical_events) as critical,
    sum(high_risk_events) as high_risk
FROM hourly_department_stats_mv
WHERE hour >= now() - INTERVAL 24 HOUR
GROUP BY department_id, hour
ORDER BY department_id, hour DESC;

-- ========================================
-- 4. PATIENT COHORT ANALYSIS
-- ========================================

-- High-Risk Patient Cohort (last 7 days)
SELECT
    patient_id,
    count() as event_count,
    countIf(risk_level IN ('HIGH', 'CRITICAL')) as high_risk_events,
    round(avg(news2_score), 1) as avg_news2,
    round(avg(heart_rate), 0) as avg_hr,
    max(timestamp) as last_event
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
GROUP BY patient_id
HAVING high_risk_events > 0
ORDER BY high_risk_events DESC, avg_news2 DESC
LIMIT 20;

-- Patient Vital Sign Trends
SELECT
    patient_id,
    toStartOfHour(timestamp) as hour,
    round(avg(heart_rate), 0) as avg_hr,
    round(avg(bp_systolic), 0) as avg_systolic,
    round(avg(spo2), 0) as avg_spo2,
    round(avg(temperature), 1) as avg_temp,
    max(news2_score) as max_news2
FROM clinical_events_fact
WHERE patient_id = 'PAT-0001'
  AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY patient_id, hour
ORDER BY hour DESC;

-- ========================================
-- 5. ML PREDICTION ANALYSIS
-- ========================================

-- ML Prediction Risk Distribution
SELECT
    CASE
        WHEN sepsis_risk_24h >= 0.7 THEN 'High Risk (>70%)'
        WHEN sepsis_risk_24h >= 0.3 THEN 'Medium Risk (30-70%)'
        ELSE 'Low Risk (<30%)'
    END as risk_category,
    count() as patient_count,
    round(avg(sepsis_risk_24h), 3) as avg_risk,
    round(min(sepsis_risk_24h), 3) as min_risk,
    round(max(sepsis_risk_24h), 3) as max_risk
FROM ml_predictions_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
GROUP BY risk_category
ORDER BY avg_risk DESC;

-- ML Prediction Trends by Hour
SELECT
    toStartOfHour(timestamp) as hour,
    count() as prediction_count,
    round(avg(sepsis_risk_24h), 3) as avg_sepsis_risk,
    round(avg(cardiac_risk_7d), 3) as avg_cardiac_risk,
    round(avg(readmission_risk_30d), 3) as avg_readmission_risk,
    round(quantile(0.95)(sepsis_risk_24h), 3) as p95_sepsis_risk
FROM ml_predictions_fact
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC;

-- ========================================
-- 6. ALERT ANALYSIS
-- ========================================

-- Alert Volume by Severity (last 7 days)
SELECT
    severity,
    count() as alert_count,
    round(avg(response_time_seconds), 0) as avg_response_seconds,
    round(quantile(0.95)(response_time_seconds), 0) as p95_response_seconds
FROM alerts_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
  AND response_time_seconds IS NOT NULL
GROUP BY severity
ORDER BY alert_count DESC;

-- Department Alert Performance
SELECT
    department_id,
    severity,
    count() as alerts,
    round(avg(response_time_seconds), 0) as avg_response,
    countIf(response_time_seconds > 300) as delayed_responses
FROM alerts_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
  AND response_time_seconds IS NOT NULL
GROUP BY department_id, severity
ORDER BY department_id, severity;

-- Hourly Alert Rates
SELECT
    toStartOfHour(timestamp) as hour,
    count() as total_alerts,
    countIf(severity = 'CRITICAL') as critical_alerts,
    countIf(severity = 'HIGH') as high_alerts,
    round(avg(response_time_seconds), 0) as avg_response_time
FROM alerts_fact
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC;

-- ========================================
-- 7. ADVANCED ANALYTICS
-- ========================================

-- Vital Sign Correlation Matrix
SELECT
    corr(heart_rate, bp_systolic) as hr_systolic_corr,
    corr(heart_rate, spo2) as hr_spo2_corr,
    corr(bp_systolic, bp_diastolic) as bp_corr,
    corr(news2_score, heart_rate) as news2_hr_corr
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY;

-- Risk Score Distribution (Histogram)
SELECT
    news2_score,
    count() as frequency,
    bar(count(), 0, max(count()) OVER (), 50) as histogram
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
  AND news2_score IS NOT NULL
GROUP BY news2_score
ORDER BY news2_score;

-- Time-to-Critical Analysis
WITH patient_first_event AS (
    SELECT
        patient_id,
        min(timestamp) as first_seen
    FROM clinical_events_fact
    WHERE timestamp >= today() - INTERVAL 7 DAY
    GROUP BY patient_id
),
critical_events AS (
    SELECT
        c.patient_id,
        c.timestamp as critical_time,
        f.first_seen
    FROM clinical_events_fact c
    JOIN patient_first_event f ON c.patient_id = f.patient_id
    WHERE c.risk_level = 'CRITICAL'
      AND c.timestamp >= today() - INTERVAL 7 DAY
)
SELECT
    patient_id,
    first_seen,
    critical_time,
    dateDiff('minute', first_seen, critical_time) as minutes_to_critical
FROM critical_events
WHERE minutes_to_critical > 0
ORDER BY minutes_to_critical
LIMIT 20;

-- ========================================
-- 8. DATA QUALITY METRICS
-- ========================================

-- Completeness Check
SELECT
    'heart_rate' as vital_sign,
    count() as total_records,
    countIf(heart_rate IS NOT NULL) as non_null_count,
    round(countIf(heart_rate IS NOT NULL) * 100.0 / count(), 1) as completeness_pct
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 1 DAY
UNION ALL
SELECT
    'bp_systolic',
    count(),
    countIf(bp_systolic IS NOT NULL),
    round(countIf(bp_systolic IS NOT NULL) * 100.0 / count(), 1)
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 1 DAY
UNION ALL
SELECT
    'spo2',
    count(),
    countIf(spo2 IS NOT NULL),
    round(countIf(spo2 IS NOT NULL) * 100.0 / count(), 1)
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 1 DAY;

-- Event Processing Latency (if event contains processing timestamps)
SELECT
    toStartOfHour(timestamp) as hour,
    count() as events,
    round(avg(JSONExtractFloat(event_data, 'processingLatencyMs')), 1) as avg_latency_ms
FROM clinical_events_fact
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour DESC;

-- ========================================
-- 9. PERFORMANCE MONITORING
-- ========================================

-- Table Statistics
SELECT
    table,
    formatReadableSize(sum(bytes)) as size,
    sum(rows) as total_rows,
    count() as parts_count
FROM system.parts
WHERE database = 'module8_analytics'
  AND active
GROUP BY table
ORDER BY sum(bytes) DESC;

-- Recent Queries Performance
SELECT
    query,
    type,
    event_time,
    query_duration_ms,
    read_rows,
    formatReadableSize(read_bytes) as read_size
FROM system.query_log
WHERE database = 'module8_analytics'
  AND event_time >= now() - INTERVAL 1 HOUR
  AND type = 'QueryFinish'
ORDER BY query_duration_ms DESC
LIMIT 10;
