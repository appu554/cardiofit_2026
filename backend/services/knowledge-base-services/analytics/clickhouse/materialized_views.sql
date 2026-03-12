-- ClickHouse Materialized Views for KB Analytics
-- Advanced materialized views for real-time analytics and reporting

-- Real-time system performance overview
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_system_performance_5min AS
SELECT
    toStartOfFiveMinute(timestamp) as time_window,
    count() as total_requests,
    countIf(failed_kb_calls = 0) as successful_requests,
    (countIf(failed_kb_calls = 0) * 100.0) / count() as success_rate_percent,
    
    -- Latency metrics
    avg(total_latency_ms) as avg_latency_ms,
    quantile(0.5)(total_latency_ms) as median_latency_ms,
    quantile(0.95)(total_latency_ms) as p95_latency_ms,
    quantile(0.99)(total_latency_ms) as p99_latency_ms,
    max(total_latency_ms) as max_latency_ms,
    
    -- Cache efficiency
    avg(cache_hit_rate) as avg_cache_hit_rate,
    
    -- Clinical metrics
    avg(guideline_adherence_score) as avg_guideline_adherence,
    avg(safety_score) as avg_safety_score,
    avg(confidence_score) as avg_confidence_score,
    
    -- Safety alerts
    countIf(length(safety_alerts) > 0) as decisions_with_alerts,
    (countIf(length(safety_alerts) > 0) * 100.0) / count() as alert_rate_percent,
    
    -- KB service utilization
    avg(total_kb_calls) as avg_kb_calls_per_decision,
    sum(total_kb_calls) as total_kb_calls,
    
    -- Unique metrics
    uniq(patient_id) as unique_patients,
    uniq(user_id) as unique_users,
    uniq(clinical_domain) as unique_clinical_domains
FROM clinical_decisions_analytics
WHERE timestamp >= now() - INTERVAL 24 HOUR
GROUP BY time_window
ORDER BY time_window DESC;

-- Individual KB service detailed performance
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kb_service_performance_hourly AS
SELECT
    toStartOfHour(timestamp) as hour,
    kb_calls.kb_name as kb_name,
    kb_calls.kb_version as kb_version,
    kb_calls.endpoint as endpoint,
    
    -- Request metrics
    count() as request_count,
    countIf(kb_calls.error_count = 0) as successful_requests,
    countIf(kb_calls.error_count > 0) as failed_requests,
    (countIf(kb_calls.error_count = 0) * 100.0) / count() as success_rate_percent,
    
    -- Latency analysis
    min(kb_calls.latency_ms) as min_latency_ms,
    max(kb_calls.latency_ms) as max_latency_ms,
    avg(kb_calls.latency_ms) as avg_latency_ms,
    quantile(0.5)(kb_calls.latency_ms) as median_latency_ms,
    quantile(0.95)(kb_calls.latency_ms) as p95_latency_ms,
    quantile(0.99)(kb_calls.latency_ms) as p99_latency_ms,
    
    -- Response size analysis
    avg(kb_calls.response_size) as avg_response_size,
    max(kb_calls.response_size) as max_response_size,
    sum(kb_calls.response_size) as total_response_bytes,
    
    -- Cache performance
    countIf(kb_calls.cache_hit = 1) as cache_hits,
    countIf(kb_calls.cache_hit = 0) as cache_misses,
    (countIf(kb_calls.cache_hit = 1) * 100.0) / count() as cache_hit_rate_percent,
    
    -- Quality metrics
    avg(kb_calls.confidence_score) as avg_confidence_score,
    avg(kb_calls.result_count) as avg_result_count,
    
    -- Error analysis
    sum(kb_calls.error_count) as total_errors,
    avg(kb_calls.error_count) as avg_error_count,
    
    -- Clinical context
    uniq(clinical_domain) as unique_clinical_domains,
    uniq(patient_id) as unique_patients_served,
    
    -- Version tracking
    uniq(version_set_id) as unique_version_sets_used
FROM clinical_decisions_analytics
ARRAY JOIN kb_calls
WHERE timestamp >= now() - INTERVAL 7 DAY
GROUP BY hour, kb_name, kb_version, endpoint
ORDER BY hour DESC, request_count DESC;

-- Cross-KB correlation and dependency analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kb_correlation_analysis AS
SELECT
    toStartOfHour(timestamp) as hour,
    kb1.kb_name as kb1_name,
    kb1.kb_version as kb1_version,
    kb2.kb_name as kb2_name,
    kb2.kb_version as kb2_version,
    
    -- Co-occurrence metrics
    count() as co_occurrence_count,
    uniq(transaction_id) as unique_transactions,
    uniq(patient_id) as unique_patients,
    
    -- Performance correlation
    corr(kb1.latency_ms, kb2.latency_ms) as latency_correlation,
    corr(kb1.confidence_score, kb2.confidence_score) as confidence_correlation,
    corr(kb1.result_count, kb2.result_count) as result_count_correlation,
    
    -- Combined performance metrics
    avg(kb1.latency_ms + kb2.latency_ms) as combined_avg_latency_ms,
    max(greatest(kb1.latency_ms, kb2.latency_ms)) as max_individual_latency_ms,
    avg(kb1.confidence_score * kb2.confidence_score) as combined_confidence_score,
    
    -- Error correlation
    countIf(kb1.error_count > 0 AND kb2.error_count > 0) as both_with_errors,
    countIf(kb1.error_count > 0 OR kb2.error_count > 0) as either_with_errors,
    
    -- Cache correlation
    countIf(kb1.cache_hit = 1 AND kb2.cache_hit = 1) as both_cache_hits,
    countIf(kb1.cache_hit = 1 OR kb2.cache_hit = 1) as either_cache_hit,
    
    -- Clinical domain analysis
    groupUniqArray(clinical_domain) as clinical_domains,
    countIf(length(safety_alerts) > 0) as transactions_with_safety_alerts
FROM clinical_decisions_analytics
ARRAY JOIN kb_calls as kb1
ARRAY JOIN kb_calls as kb2
WHERE kb1.kb_name != kb2.kb_name
  AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour, kb1_name, kb1_version, kb2_name, kb2_version
HAVING co_occurrence_count >= 5
ORDER BY hour DESC, co_occurrence_count DESC;

-- Clinical domain performance analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_clinical_domain_performance AS
SELECT
    toStartOfHour(timestamp) as hour,
    clinical_domain,
    request_type,
    
    -- Volume metrics
    count() as total_requests,
    uniq(patient_id) as unique_patients,
    uniq(user_id) as unique_clinicians,
    
    -- Performance metrics
    avg(total_latency_ms) as avg_total_latency_ms,
    quantile(0.95)(total_latency_ms) as p95_latency_ms,
    avg(total_kb_calls) as avg_kb_calls_per_request,
    avg(cache_hit_rate) as avg_cache_hit_rate,
    
    -- Clinical quality metrics
    avg(guideline_adherence_score) as avg_guideline_adherence,
    avg(safety_score) as avg_safety_score,
    avg(confidence_score) as avg_confidence_score,
    avg(completeness_score) as avg_completeness_score,
    
    -- Decision metrics
    countIf(decision_accepted = 1) as accepted_decisions,
    countIf(decision_accepted = 0) as rejected_decisions,
    (countIf(decision_accepted = 1) * 100.0) / count() as acceptance_rate_percent,
    
    -- Safety metrics
    countIf(length(safety_alerts) > 0) as requests_with_safety_alerts,
    (countIf(length(safety_alerts) > 0) * 100.0) / count() as safety_alert_rate_percent,
    flatten(groupArray(safety_alerts)) as all_safety_alerts,
    
    -- Override analysis
    countIf(length(clinical_overrides) > 0) as requests_with_overrides,
    (countIf(length(clinical_overrides) > 0) * 100.0) / count() as override_rate_percent,
    
    -- KB service utilization by domain
    groupUniqArray(arrayJoin(kb_calls.kb_name)) as kb_services_used,
    length(groupUniqArray(arrayJoin(kb_calls.kb_name))) as unique_kb_services_count,
    
    -- Outcome tracking
    countIf(clinical_outcome = 'positive') as positive_outcomes,
    countIf(clinical_outcome = 'negative') as negative_outcomes,
    countIf(clinical_outcome = 'neutral') as neutral_outcomes,
    countIf(clinical_outcome = 'unknown') as unknown_outcomes
FROM clinical_decisions_analytics
WHERE timestamp >= now() - INTERVAL 7 DAY
GROUP BY hour, clinical_domain, request_type
ORDER BY hour DESC, total_requests DESC;

-- Safety signal trend analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_safety_signal_trends AS
SELECT
    toStartOfHour(timestamp) as hour,
    signal_type,
    kb_source,
    severity,
    clinical_domain,
    
    -- Signal volume metrics
    count() as signal_count,
    uniq(signal_id) as unique_signals,
    
    -- Impact metrics
    sum(estimated_patient_impact) as total_estimated_impact,
    avg(estimated_patient_impact) as avg_estimated_impact,
    avg(clinical_significance_score) as avg_clinical_significance,
    avg(urgency_score) as avg_urgency_score,
    
    -- Quality metrics
    avg(confidence_score) as avg_confidence_score,
    avg(false_positive_probability) as avg_fp_probability,
    
    -- Response metrics
    avg(detection_to_acknowledgment_minutes) as avg_acknowledgment_time_min,
    avg(total_resolution_time_minutes) as avg_resolution_time_min,
    
    -- Status distribution
    countIf(resolution_status = 'resolved') as resolved_count,
    countIf(resolution_status = 'investigating') as investigating_count,
    countIf(resolution_status = 'open') as open_count,
    countIf(actual_outcome = 'true_positive') as true_positive_count,
    countIf(actual_outcome = 'false_positive') as false_positive_count,
    
    -- Action analysis
    avg(automated_actions_count) as avg_automated_actions,
    avg(manual_actions_count) as avg_manual_actions,
    countIf(escalations_count > 0) as signals_escalated,
    
    -- Geographic distribution
    groupUniqArray(region) as regions_affected,
    
    -- Detection method analysis
    groupUniqArray(detection_method) as detection_methods_used
FROM safety_signal_analytics
WHERE timestamp >= now() - INTERVAL 7 DAY
GROUP BY hour, signal_type, kb_source, severity, clinical_domain
ORDER BY hour DESC, signal_count DESC;

-- Patient journey performance tracking
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_patient_journey_performance AS
SELECT
    date,
    stage,
    primary_condition,
    age_group,
    gender,
    region,
    
    -- Volume metrics
    count() as stage_instances,
    uniq(patient_hash) as unique_patients,
    uniq(journey_id) as unique_journeys,
    
    -- Performance metrics
    avg(stage_duration_minutes) as avg_stage_duration_minutes,
    quantile(0.5)(stage_duration_minutes) as median_stage_duration_minutes,
    quantile(0.95)(stage_duration_minutes) as p95_stage_duration_minutes,
    
    -- KB utilization
    avg(total_kb_calls) as avg_kb_calls_per_stage,
    sum(total_kb_calls) as total_kb_calls,
    flatten(groupArray(kb_services_used)) as all_kb_services_used,
    length(groupUniqArray(arrayJoin(kb_services_used))) as unique_kb_services,
    
    -- Quality metrics
    avg(guideline_adherence_percentage) as avg_guideline_adherence,
    avg(safety_compliance_percentage) as avg_safety_compliance,
    avg(patient_satisfaction_score) as avg_patient_satisfaction,
    avg(readmission_risk_score) as avg_readmission_risk,
    
    -- Decision metrics
    avg(length(recommendations_followed)) as avg_recommendations_followed,
    avg(length(recommendations_overridden)) as avg_recommendations_overridden,
    avg(length(safety_alerts_triggered)) as avg_safety_alerts,
    
    -- Outcome distribution
    countIf(clinical_outcome = 'positive') as positive_outcomes,
    countIf(clinical_outcome = 'negative') as negative_outcomes,
    countIf(clinical_outcome = 'neutral') as neutral_outcomes,
    (countIf(clinical_outcome = 'positive') * 100.0) / count() as positive_outcome_rate
FROM patient_journey_analytics
WHERE timestamp >= today() - INTERVAL 30 DAY
GROUP BY date, stage, primary_condition, age_group, gender, region
ORDER BY date DESC, stage_instances DESC;

-- Version deployment impact tracking
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_version_deployment_impact AS
SELECT
    deployment_id,
    kb_name,
    old_version,
    new_version,
    environment,
    toStartOfHour(deployment_timestamp) as deployment_hour,
    hours_since_deployment,
    
    -- Aggregate impact metrics
    count() as measurement_count,
    avg(request_count_change) as avg_request_count_change_percent,
    avg(latency_change_ms) as avg_latency_change_ms,
    avg(error_rate_change) as avg_error_rate_change_percent,
    avg(cache_hit_rate_change) as avg_cache_hit_rate_change_percent,
    
    -- Clinical impact aggregation
    avg(recommendation_accuracy_change) as avg_accuracy_change_percent,
    avg(safety_alert_rate_change) as avg_safety_alert_rate_change_percent,
    avg(guideline_adherence_change) as avg_guideline_adherence_change_percent,
    
    -- Quality metrics
    avg(confidence_score_change) as avg_confidence_change_percent,
    avg(consistency_score_change) as avg_consistency_change_percent,
    
    -- Adoption metrics
    max(adoption_rate) as peak_adoption_rate,
    sum(rollback_requests) as total_rollback_requests,
    
    -- Feedback analysis
    sum(positive_feedback_count) as total_positive_feedback,
    sum(negative_feedback_count) as total_negative_feedback,
    sum(neutral_feedback_count) as total_neutral_feedback,
    
    -- Overall impact assessment
    CASE 
        WHEN avg(latency_change_ms) <= -10 AND avg(error_rate_change) <= -5 THEN 'Very Positive'
        WHEN avg(latency_change_ms) <= 0 AND avg(error_rate_change) <= 0 THEN 'Positive'
        WHEN avg(latency_change_ms) <= 10 AND avg(error_rate_change) <= 5 THEN 'Neutral'
        WHEN avg(latency_change_ms) <= 50 AND avg(error_rate_change) <= 15 THEN 'Negative'
        ELSE 'Very Negative'
    END as overall_impact_assessment
FROM version_deployment_impact
WHERE deployment_timestamp >= today() - INTERVAL 30 DAY
GROUP BY deployment_id, kb_name, old_version, new_version, environment, deployment_hour, hours_since_deployment
ORDER BY deployment_timestamp DESC, hours_since_deployment;

-- Real-time alerting views

-- Critical system health alerts
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_critical_health_alerts AS
SELECT
    now() as check_timestamp,
    'system_health' as alert_category,
    
    -- Latency alerts
    CASE 
        WHEN max(p95_latency_ms) > 1000 THEN 'CRITICAL: P95 latency exceeds 1000ms'
        WHEN max(p95_latency_ms) > 500 THEN 'WARNING: P95 latency exceeds 500ms'
        ELSE ''
    END as latency_alert,
    
    -- Error rate alerts
    CASE
        WHEN min(success_rate_percent) < 95 THEN 'CRITICAL: Success rate below 95%'
        WHEN min(success_rate_percent) < 98 THEN 'WARNING: Success rate below 98%'
        ELSE ''
    END as error_rate_alert,
    
    -- Cache performance alerts
    CASE
        WHEN min(avg_cache_hit_rate) < 0.80 THEN 'WARNING: Cache hit rate below 80%'
        WHEN min(avg_cache_hit_rate) < 0.90 THEN 'INFO: Cache hit rate below 90%'
        ELSE ''
    END as cache_alert,
    
    -- Safety alerts
    CASE
        WHEN max(alert_rate_percent) > 10 THEN 'CRITICAL: Safety alert rate exceeds 10%'
        WHEN max(alert_rate_percent) > 5 THEN 'WARNING: Safety alert rate exceeds 5%'
        ELSE ''
    END as safety_alert_rate_alert,
    
    -- Current metrics for context
    max(total_requests) as max_requests_5min,
    max(p95_latency_ms) as max_p95_latency_ms,
    min(success_rate_percent) as min_success_rate,
    min(avg_cache_hit_rate) as min_cache_hit_rate,
    max(alert_rate_percent) as max_safety_alert_rate
FROM mv_system_performance_5min
WHERE time_window >= now() - INTERVAL 30 MINUTE;

-- KB service health monitoring
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_kb_service_health AS
SELECT
    kb_name,
    kb_version,
    
    -- Health indicators
    CASE
        WHEN success_rate_percent < 95 THEN 'UNHEALTHY'
        WHEN success_rate_percent < 98 OR p95_latency_ms > 100 THEN 'DEGRADED'
        WHEN success_rate_percent >= 99 AND p95_latency_ms <= 50 THEN 'EXCELLENT'
        ELSE 'HEALTHY'
    END as health_status,
    
    -- Current performance metrics
    success_rate_percent,
    avg_latency_ms,
    p95_latency_ms,
    cache_hit_rate_percent,
    avg_confidence_score,
    
    -- Volume metrics
    request_count as requests_last_hour,
    unique_patients_served,
    unique_clinical_domains,
    
    -- Last update
    max(hour) as last_active_hour
FROM mv_kb_service_performance_hourly
WHERE hour >= now() - INTERVAL 1 HOUR
GROUP BY kb_name, kb_version, success_rate_percent, avg_latency_ms, p95_latency_ms, 
         cache_hit_rate_percent, avg_confidence_score, request_count, 
         unique_patients_served, unique_clinical_domains
ORDER BY health_status DESC, request_count DESC;