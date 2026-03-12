-- Seed Test Data for Notification Service
-- Purpose: Populate test data for development and testing

SET search_path TO notification_service, public;

-- ============================================================================
-- User Preferences Test Data
-- ============================================================================

-- Critical care staff (wants all channels)
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'user_attending_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS", "VOICE"],
        "HIGH": ["PAGER", "SMS", "PUSH"],
        "MODERATE": ["SMS", "PUSH"],
        "LOW": ["PUSH"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    false,
    30,
    'fcm_token_attending_001_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    '+1-555-0101',
    'dr.attending@cardiofit.com',
    '1234567',
    'en',
    'America/New_York'
),
(
    'user_charge_nurse_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    false,
    25,
    'fcm_token_charge_nurse_001_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    '+1-555-0102',
    'charge.nurse@cardiofit.com',
    '1234568',
    'en',
    'America/New_York'
),
(
    'user_primary_nurse_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    true,
    20,
    'fcm_token_primary_nurse_001_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    '+1-555-0103',
    'primary.nurse@cardiofit.com',
    NULL,
    'en',
    'America/New_York'
);

UPDATE notification_service.user_preferences
SET quiet_hours_start = 22, quiet_hours_end = 6
WHERE user_id = 'user_primary_nurse_001';

-- Clinical informatics team (email-heavy)
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, max_alerts_per_hour,
    fcm_token, phone_number, email, language, timezone
) VALUES
(
    'user_clinical_informatics_001',
    '{"SMS": false, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false}'::jsonb,
    '{
        "CRITICAL": ["EMAIL", "PUSH"],
        "HIGH": ["EMAIL", "PUSH"],
        "MODERATE": ["EMAIL"],
        "LOW": ["EMAIL"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    true,
    50,
    'fcm_token_informatics_001_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    NULL,
    'informatics@cardiofit.com',
    'en',
    'America/Los_Angeles'
);

UPDATE notification_service.user_preferences
SET quiet_hours_start = 20, quiet_hours_end = 8
WHERE user_id = 'user_clinical_informatics_001';

-- Resident physician (moderate alert volume)
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'user_resident_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    false,
    20,
    'fcm_token_resident_001_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    '+1-555-0104',
    'resident@cardiofit.com',
    '1234569',
    'en',
    'America/Chicago'
);

-- ============================================================================
-- Notifications Test Data
-- ============================================================================

-- Critical sepsis alert - delivered and acknowledged
INSERT INTO notification_service.notifications (
    id, alert_id, user_id, channel, priority, message, status,
    external_id, created_at, sent_at, delivered_at, acknowledged_at,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_sepsis_001',
    'user_attending_001',
    'PAGER',
    1,
    'CRIT PAT-12345 SEPSIS 92% ICU-5A',
    'ACKNOWLEDGED',
    'PG_SM123456789',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '2 hours' + INTERVAL '2 seconds',
    NOW() - INTERVAL '2 hours' + INTERVAL '5 seconds',
    NOW() - INTERVAL '2 hours' + INTERVAL '3 minutes',
    '{"severity": "CRITICAL", "patient_id": "PAT-12345", "alert_type": "SEPSIS_ALERT", "risk_score": 0.92, "location": "ICU-5A"}'::jsonb
),
(
    gen_random_uuid(),
    'alert_sepsis_001',
    'user_attending_001',
    'SMS',
    1,
    'CRITICAL: PAT-12345 Sepsis Alert (92%) - ICU-5A',
    'ACKNOWLEDGED',
    'SM_TW987654321',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '2 hours' + INTERVAL '1 second',
    NOW() - INTERVAL '2 hours' + INTERVAL '3 seconds',
    NOW() - INTERVAL '2 hours' + INTERVAL '3 minutes',
    '{"severity": "CRITICAL", "patient_id": "PAT-12345", "alert_type": "SEPSIS_ALERT", "risk_score": 0.92, "location": "ICU-5A"}'::jsonb
);

-- High priority deterioration alert - delivered
INSERT INTO notification_service.notifications (
    id, alert_id, user_id, channel, priority, message, status,
    external_id, created_at, sent_at, delivered_at,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_deterioration_001',
    'user_primary_nurse_001',
    'SMS',
    2,
    'HIGH: PAT-67890 Deterioration - Review vitals - ER-3',
    'DELIVERED',
    'SM_TW555666777',
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '1 hour' + INTERVAL '1 second',
    NOW() - INTERVAL '1 hour' + INTERVAL '4 seconds',
    '{"severity": "HIGH", "patient_id": "PAT-67890", "alert_type": "DETERIORATION_ALERT", "location": "ER-3"}'::jsonb
),
(
    gen_random_uuid(),
    'alert_deterioration_001',
    'user_primary_nurse_001',
    'PUSH',
    2,
    'Patient Deterioration: PAT-67890 - HIGH - Review vital signs immediately',
    'DELIVERED',
    'FCM_msg_id_abc123xyz',
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '1 hour' + INTERVAL '1 second',
    NOW() - INTERVAL '1 hour' + INTERVAL '2 seconds',
    '{"severity": "HIGH", "patient_id": "PAT-67890", "alert_type": "DETERIORATION_ALERT", "location": "ER-3"}'::jsonb
);

-- Moderate vital sign anomaly - sent
INSERT INTO notification_service.notifications (
    id, alert_id, user_id, channel, priority, message, status,
    external_id, created_at, sent_at,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_vital_anomaly_001',
    'user_charge_nurse_001',
    'PUSH',
    3,
    'MODERATE: PAT-11111 Heart Rate abnormal - 135 bpm - Ward-2B',
    'SENT',
    'FCM_msg_id_def456uvw',
    NOW() - INTERVAL '30 minutes',
    NOW() - INTERVAL '30 minutes' + INTERVAL '1 second',
    '{"severity": "MODERATE", "patient_id": "PAT-11111", "alert_type": "VITAL_SIGN_ANOMALY", "parameter": "heart_rate", "value": 135, "location": "Ward-2B"}'::jsonb
);

-- Failed notification (retry exhausted)
INSERT INTO notification_service.notifications (
    id, alert_id, user_id, channel, priority, message, status,
    retry_count, created_at, error_message,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_lab_result_001',
    'user_resident_001',
    'EMAIL',
    4,
    'LOW: PAT-22222 Lab Result Available - Creatinine elevated',
    'FAILED',
    3,
    NOW() - INTERVAL '20 minutes',
    'SMTP connection timeout after 3 retry attempts',
    '{"severity": "LOW", "patient_id": "PAT-22222", "alert_type": "LAB_RESULT", "test": "creatinine"}'::jsonb
);

-- Pending notification (just created)
INSERT INTO notification_service.notifications (
    id, alert_id, user_id, channel, priority, message, status,
    created_at,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_ml_mortality_001',
    'user_clinical_informatics_001',
    'EMAIL',
    3,
    'ML Alert: PAT-33333 30-day mortality risk elevated to 78%',
    'PENDING',
    NOW() - INTERVAL '5 seconds',
    '{"severity": "MODERATE", "patient_id": "PAT-33333", "alert_type": "MORTALITY_RISK", "risk_score": 0.78, "model_version": "1.2.3"}'::jsonb
);

-- ============================================================================
-- Escalation Log Test Data
-- ============================================================================

-- Successful escalation (acknowledged at level 1)
INSERT INTO notification_service.escalation_log (
    id, alert_id, escalation_level, escalated_to_user, escalated_to_role,
    escalated_at, acknowledged_at, acknowledged_by, outcome, response_time_ms,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_sepsis_001',
    1,
    'user_attending_001',
    'Attending Physician',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '2 hours' + INTERVAL '3 minutes',
    'user_attending_001',
    'ACKNOWLEDGED',
    180000,
    '{"patient_id": "PAT-12345", "department": "ICU", "severity": "CRITICAL"}'::jsonb
);

-- Escalation chain (level 1 timeout, escalated to level 2, acknowledged)
INSERT INTO notification_service.escalation_log (
    id, alert_id, escalation_level, escalated_to_user, escalated_to_role,
    escalated_at, acknowledged_at, outcome, response_time_ms,
    metadata
) VALUES
(
    gen_random_uuid(),
    'alert_deterioration_002',
    1,
    'user_primary_nurse_001',
    'Primary Nurse',
    NOW() - INTERVAL '30 minutes',
    NULL,
    'TIMEOUT',
    NULL,
    '{"patient_id": "PAT-88888", "department": "Emergency", "severity": "HIGH"}'::jsonb
),
(
    gen_random_uuid(),
    'alert_deterioration_002',
    2,
    'user_charge_nurse_001',
    'Charge Nurse',
    NOW() - INTERVAL '25 minutes',
    NOW() - INTERVAL '22 minutes',
    'ACKNOWLEDGED',
    180000,
    '{"patient_id": "PAT-88888", "department": "Emergency", "severity": "HIGH"}'::jsonb
);

-- Active escalation (not yet acknowledged)
INSERT INTO notification_service.escalation_log (
    id, alert_id, escalation_level, escalated_to_user, escalated_to_role,
    escalated_at, metadata
) VALUES
(
    gen_random_uuid(),
    'alert_ml_mortality_001',
    1,
    'user_clinical_informatics_001',
    'Clinical Informatics',
    NOW() - INTERVAL '5 seconds',
    '{"patient_id": "PAT-33333", "department": "ICU", "severity": "MODERATE"}'::jsonb
);

-- ============================================================================
-- Alert Fatigue History Test Data
-- ============================================================================

-- Recent alerts for rate limiting test
INSERT INTO notification_service.alert_fatigue_history (
    user_id, alert_id, patient_id, alert_type, severity,
    suppressed, suppression_reason, created_at, expires_at
) VALUES
('user_primary_nurse_001', 'alert_vital_001', 'PAT-11111', 'VITAL_SIGN_ANOMALY', 'MODERATE',
 false, NULL, NOW() - INTERVAL '10 minutes', NOW() + INTERVAL '14 hours'),
('user_primary_nurse_001', 'alert_vital_002', 'PAT-11111', 'VITAL_SIGN_ANOMALY', 'MODERATE',
 true, 'DUPLICATE', NOW() - INTERVAL '8 minutes', NOW() + INTERVAL '16 hours'),
('user_primary_nurse_001', 'alert_lab_001', 'PAT-22222', 'LAB_RESULT', 'LOW',
 false, NULL, NOW() - INTERVAL '5 minutes', NOW() + INTERVAL '19 hours'),
('user_primary_nurse_001', 'alert_vital_003', 'PAT-11111', 'VITAL_SIGN_ANOMALY', 'MODERATE',
 true, 'BUNDLED', NOW() - INTERVAL '3 minutes', NOW() + INTERVAL '21 hours'),
('user_primary_nurse_001', 'alert_vital_004', 'PAT-11111', 'VITAL_SIGN_ANOMALY', 'MODERATE',
 true, 'BUNDLED', NOW() - INTERVAL '2 minutes', NOW() + INTERVAL '22 hours');

-- Rate limited alerts
INSERT INTO notification_service.alert_fatigue_history (
    user_id, alert_id, patient_id, alert_type, severity,
    suppressed, suppression_reason, created_at, expires_at
)
SELECT
    'user_charge_nurse_001',
    'alert_rate_limit_' || generate_series,
    'PAT-' || (10000 + generate_series)::text,
    'VITAL_SIGN_ANOMALY',
    'LOW',
    CASE WHEN generate_series > 20 THEN true ELSE false END,
    CASE WHEN generate_series > 20 THEN 'RATE_LIMIT' ELSE NULL END,
    NOW() - INTERVAL '50 minutes' + (generate_series || ' minutes')::interval,
    NOW() + INTERVAL '10 hours' + (generate_series || ' minutes')::interval
FROM generate_series(1, 25);

-- ============================================================================
-- Delivery Metrics Test Data
-- ============================================================================

-- Today's metrics (hourly breakdown)
INSERT INTO notification_service.delivery_metrics (
    date, channel, hour, total_sent, total_delivered, total_failed, total_acknowledged,
    avg_delivery_time_ms, p50_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
    error_count_by_type
) VALUES
(CURRENT_DATE, 'SMS', 8, 45, 43, 2, 38, 2500, 2200, 4500, 6000,
 '{"TIMEOUT": 1, "INVALID_NUMBER": 1}'::jsonb),
(CURRENT_DATE, 'SMS', 9, 52, 50, 2, 44, 2300, 2100, 4200, 5800,
 '{"TIMEOUT": 2}'::jsonb),
(CURRENT_DATE, 'EMAIL', 8, 30, 28, 2, 20, 8500, 7500, 15000, 25000,
 '{"SMTP_ERROR": 2}'::jsonb),
(CURRENT_DATE, 'EMAIL', 9, 35, 33, 2, 25, 8200, 7200, 14500, 24000,
 '{"SMTP_ERROR": 1, "INVALID_EMAIL": 1}'::jsonb),
(CURRENT_DATE, 'PUSH', 8, 80, 78, 2, 65, 1500, 1200, 2500, 3500,
 '{"FCM_ERROR": 2}'::jsonb),
(CURRENT_DATE, 'PUSH', 9, 95, 92, 3, 78, 1450, 1180, 2400, 3400,
 '{"FCM_ERROR": 2, "INVALID_TOKEN": 1}'::jsonb),
(CURRENT_DATE, 'PAGER', 8, 15, 15, 0, 12, 3000, 2800, 4000, 5000, '{}'::jsonb),
(CURRENT_DATE, 'PAGER', 9, 18, 18, 0, 15, 2900, 2700, 3900, 4800, '{}'::jsonb);

-- Yesterday's daily aggregate
INSERT INTO notification_service.delivery_metrics (
    date, channel, hour, total_sent, total_delivered, total_failed, total_acknowledged,
    avg_delivery_time_ms, p50_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
    error_count_by_type
) VALUES
(CURRENT_DATE - INTERVAL '1 day', 'SMS', NULL, 1200, 1150, 50, 1000, 2400, 2100, 4300, 5900,
 '{"TIMEOUT": 30, "INVALID_NUMBER": 20}'::jsonb),
(CURRENT_DATE - INTERVAL '1 day', 'EMAIL', NULL, 800, 750, 50, 600, 8400, 7400, 14800, 24500,
 '{"SMTP_ERROR": 40, "INVALID_EMAIL": 10}'::jsonb),
(CURRENT_DATE - INTERVAL '1 day', 'PUSH', NULL, 2000, 1950, 50, 1700, 1480, 1200, 2450, 3450,
 '{"FCM_ERROR": 35, "INVALID_TOKEN": 15}'::jsonb),
(CURRENT_DATE - INTERVAL '1 day', 'PAGER', NULL, 400, 395, 5, 350, 2950, 2750, 3950, 4900,
 '{"GATEWAY_ERROR": 5}'::jsonb),
(CURRENT_DATE - INTERVAL '1 day', 'ALL', NULL, 4400, 4245, 155, 3650, 3500, 2500, 10000, 20000,
 '{"TIMEOUT": 30, "INVALID_NUMBER": 20, "SMTP_ERROR": 40, "INVALID_EMAIL": 10, "FCM_ERROR": 35, "INVALID_TOKEN": 15, "GATEWAY_ERROR": 5}'::jsonb);

-- Last 7 days daily aggregates
INSERT INTO notification_service.delivery_metrics (
    date, channel, hour, total_sent, total_delivered, total_failed, total_acknowledged,
    avg_delivery_time_ms, p50_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms
)
SELECT
    CURRENT_DATE - (generate_series || ' days')::interval,
    'ALL',
    NULL,
    3500 + (generate_series * 100),
    3350 + (generate_series * 95),
    150 + (generate_series * 5),
    2900 + (generate_series * 80),
    3500,
    2500,
    10000,
    20000
FROM generate_series(2, 7);

-- ============================================================================
-- Summary
-- ============================================================================

SELECT 'Seed data loaded successfully' as status;
SELECT COUNT(*) as user_preferences_count FROM notification_service.user_preferences;
SELECT COUNT(*) as notifications_count FROM notification_service.notifications;
SELECT COUNT(*) as escalation_log_count FROM notification_service.escalation_log;
SELECT COUNT(*) as fatigue_history_count FROM notification_service.alert_fatigue_history;
SELECT COUNT(*) as delivery_metrics_count FROM notification_service.delivery_metrics;
SELECT COUNT(*) as notification_templates_count FROM notification_service.notification_templates;
