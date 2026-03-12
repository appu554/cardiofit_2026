-- Test Data for User Preference Service
-- Purpose: Comprehensive test data for user lookup and preference testing
-- Database: cardiofit_analytics, Schema: notification_service
-- Port: 5433

SET search_path TO notification_service, public;

-- ============================================================================
-- Clean up existing test data (optional, for fresh runs)
-- ============================================================================

-- DELETE FROM notification_service.user_preferences WHERE user_id LIKE 'test_%';

-- ============================================================================
-- Test Users: Attending Physicians
-- ============================================================================

INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'test_user_attending_icu_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": true, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS", "VOICE"],
        "HIGH": ["PAGER", "SMS", "PUSH"],
        "MODERATE": ["SMS", "PUSH"],
        "LOW": ["PUSH", "IN_APP"],
        "ML_ALERT": ["EMAIL", "PUSH"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    30,
    'fcm_token_attending_icu_001_test',
    '+1-555-1001',
    'dr.attending.icu@cardiofit-test.com',
    'PG-001-ICU',
    'en',
    'America/New_York'
),
(
    'test_user_attending_er_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": true, "IN_APP": false}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS", "VOICE"],
        "HIGH": ["PAGER", "SMS"],
        "MODERATE": ["SMS", "PUSH"],
        "LOW": ["PUSH"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    35,
    'fcm_token_attending_er_001_test',
    '+1-555-1002',
    'dr.attending.er@cardiofit-test.com',
    'PG-002-ER',
    'en',
    'America/New_York'
),
(
    'test_user_attending_cardio_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL", "PUSH"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    25,
    'fcm_token_attending_cardio_001_test',
    '+1-555-1003',
    'dr.attending.cardio@cardiofit-test.com',
    'PG-003-CARDIO',
    'en',
    'America/Chicago'
);

-- ============================================================================
-- Test Users: Charge Nurses
-- ============================================================================

INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'test_user_charge_nurse_icu_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    25,
    'fcm_token_charge_nurse_icu_001_test',
    '+1-555-2001',
    'charge.nurse.icu@cardiofit-test.com',
    'PG-101-ICU',
    'en',
    'America/New_York'
),
(
    'test_user_charge_nurse_er_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    30,
    'fcm_token_charge_nurse_er_001_test',
    '+1-555-2002',
    'charge.nurse.er@cardiofit-test.com',
    'PG-102-ER',
    'en',
    'America/New_York'
),
(
    'test_user_charge_nurse_cardio_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    20,
    'fcm_token_charge_nurse_cardio_001_test',
    '+1-555-2003',
    'charge.nurse.cardio@cardiofit-test.com',
    NULL,
    'en',
    'America/Chicago'
);

-- ============================================================================
-- Test Users: Primary Nurses
-- ============================================================================

INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'test_user_primary_nurse_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    true,
    22,
    6,
    20,
    'fcm_token_primary_nurse_001_test',
    '+1-555-3001',
    'primary.nurse.001@cardiofit-test.com',
    NULL,
    'en',
    'America/New_York'
),
(
    'test_user_primary_nurse_002',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    true,
    23,
    7,
    18,
    'fcm_token_primary_nurse_002_test',
    '+1-555-3002',
    'primary.nurse.002@cardiofit-test.com',
    NULL,
    'en',
    'America/New_York'
),
(
    'test_user_primary_nurse_003',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["PUSH", "IN_APP"],
        "MODERATE": ["IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    25,
    'fcm_token_primary_nurse_003_test',
    '+1-555-3003',
    'primary.nurse.003@cardiofit-test.com',
    NULL,
    'en',
    'America/Chicago'
),
(
    'test_user_primary_nurse_004',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["SMS", "PUSH"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    true,
    20,
    8,
    20,
    'fcm_token_primary_nurse_004_test',
    '+1-555-3004',
    'primary.nurse.004@cardiofit-test.com',
    NULL,
    'en',
    'America/Los_Angeles'
);

-- ============================================================================
-- Test Users: Residents
-- ============================================================================

INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'test_user_resident_icu_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    20,
    'fcm_token_resident_icu_001_test',
    '+1-555-4001',
    'resident.icu.001@cardiofit-test.com',
    'PG-201-ICU',
    'en',
    'America/New_York'
),
(
    'test_user_resident_icu_002',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    22,
    'fcm_token_resident_icu_002_test',
    '+1-555-4002',
    'resident.icu.002@cardiofit-test.com',
    'PG-202-ICU',
    'en',
    'America/New_York'
),
(
    'test_user_resident_er_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    25,
    'fcm_token_resident_er_001_test',
    '+1-555-4003',
    'resident.er.001@cardiofit-test.com',
    'PG-203-ER',
    'en',
    'America/New_York'
),
(
    'test_user_resident_cardio_001',
    '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["PAGER", "SMS"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH"],
        "LOW": ["PUSH"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    18,
    'fcm_token_resident_cardio_001_test',
    '+1-555-4004',
    'resident.cardio.001@cardiofit-test.com',
    'PG-204-CARDIO',
    'en',
    'America/Chicago'
);

-- ============================================================================
-- Test Users: Clinical Informatics Team
-- ============================================================================

INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    fcm_token, phone_number, email, pager_number, language, timezone
) VALUES
(
    'test_user_clinical_informatics_001',
    '{"SMS": false, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["EMAIL", "PUSH"],
        "HIGH": ["EMAIL", "PUSH"],
        "MODERATE": ["EMAIL", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL", "PUSH"]
    }'::jsonb,
    true,
    20,
    8,
    50,
    'fcm_token_informatics_001_test',
    NULL,
    'informatics.001@cardiofit-test.com',
    NULL,
    'en',
    'America/Los_Angeles'
),
(
    'test_user_clinical_informatics_002',
    '{"SMS": false, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["EMAIL", "PUSH"],
        "HIGH": ["EMAIL", "PUSH"],
        "MODERATE": ["EMAIL"],
        "LOW": ["EMAIL"],
        "ML_ALERT": ["EMAIL", "PUSH", "IN_APP"]
    }'::jsonb,
    true,
    19,
    7,
    60,
    'fcm_token_informatics_002_test',
    NULL,
    'informatics.002@cardiofit-test.com',
    NULL,
    'en',
    'America/New_York'
),
(
    'test_user_clinical_informatics_003',
    '{"SMS": false, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false, "IN_APP": true}'::jsonb,
    '{
        "CRITICAL": ["EMAIL", "PUSH"],
        "HIGH": ["EMAIL"],
        "MODERATE": ["EMAIL", "IN_APP"],
        "LOW": ["IN_APP"],
        "ML_ALERT": ["EMAIL", "PUSH"]
    }'::jsonb,
    false,
    NULL,
    NULL,
    40,
    'fcm_token_informatics_003_test',
    '+1-555-5003',
    'informatics.003@cardiofit-test.com',
    NULL,
    'en',
    'America/Chicago'
);

-- ============================================================================
-- Test Patient-Nurse Assignments (for primary nurse lookups)
-- Note: In production, this would be a separate table
-- For testing, we'll document the assignments here
-- ============================================================================

-- Patient Assignments:
-- PAT-TEST-001 -> test_user_primary_nurse_001
-- PAT-TEST-002 -> test_user_primary_nurse_001
-- PAT-TEST-003 -> test_user_primary_nurse_002
-- PAT-TEST-004 -> test_user_primary_nurse_003
-- PAT-TEST-005 -> test_user_primary_nurse_004

-- ============================================================================
-- Test Users: Edge Cases
-- ============================================================================

-- User with minimal preferences
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, max_alerts_per_hour,
    phone_number, email, language, timezone
) VALUES
(
    'test_user_minimal_prefs',
    '{"SMS": true, "EMAIL": true}'::jsonb,
    '{
        "CRITICAL": ["SMS"],
        "HIGH": ["SMS"]
    }'::jsonb,
    false,
    10,
    '+1-555-9001',
    'minimal@cardiofit-test.com',
    'en',
    'UTC'
);

-- User with all channels disabled except email
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, max_alerts_per_hour,
    phone_number, email, language, timezone
) VALUES
(
    'test_user_email_only',
    '{"SMS": false, "EMAIL": true, "PUSH": false, "PAGER": false, "VOICE": false, "IN_APP": false}'::jsonb,
    '{
        "CRITICAL": ["EMAIL"],
        "HIGH": ["EMAIL"],
        "MODERATE": ["EMAIL"],
        "LOW": ["EMAIL"],
        "ML_ALERT": ["EMAIL"]
    }'::jsonb,
    false,
    100,
    NULL,
    'emailonly@cardiofit-test.com',
    'en',
    'UTC'
);

-- User with very restrictive quiet hours
INSERT INTO notification_service.user_preferences (
    user_id, channel_preferences, severity_channels,
    quiet_hours_enabled, quiet_hours_start, quiet_hours_end, max_alerts_per_hour,
    phone_number, email, language, timezone
) VALUES
(
    'test_user_strict_quiet_hours',
    '{"SMS": true, "EMAIL": true, "PUSH": true}'::jsonb,
    '{
        "CRITICAL": ["SMS"],
        "HIGH": ["PUSH"],
        "MODERATE": ["EMAIL"],
        "LOW": ["EMAIL"]
    }'::jsonb,
    true,
    18,
    10,
    5,
    '+1-555-9003',
    'quiethours@cardiofit-test.com',
    'en',
    'America/New_York'
);

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Verify all test users were created
SELECT
    'Test Users Created' as status,
    COUNT(*) as total_count,
    COUNT(CASE WHEN user_id LIKE '%attending%' THEN 1 END) as attending_count,
    COUNT(CASE WHEN user_id LIKE '%charge_nurse%' THEN 1 END) as charge_nurse_count,
    COUNT(CASE WHEN user_id LIKE '%primary_nurse%' THEN 1 END) as primary_nurse_count,
    COUNT(CASE WHEN user_id LIKE '%resident%' THEN 1 END) as resident_count,
    COUNT(CASE WHEN user_id LIKE '%informatics%' THEN 1 END) as informatics_count,
    COUNT(CASE WHEN user_id LIKE '%minimal%' OR user_id LIKE '%email_only%' OR user_id LIKE '%quiet_hours%' THEN 1 END) as edge_case_count
FROM notification_service.user_preferences
WHERE user_id LIKE 'test_%';

-- Show quiet hours configuration
SELECT
    user_id,
    quiet_hours_enabled,
    quiet_hours_start,
    quiet_hours_end,
    max_alerts_per_hour
FROM notification_service.user_preferences
WHERE user_id LIKE 'test_%' AND quiet_hours_enabled = true
ORDER BY user_id;

-- Show channel preferences summary
SELECT
    user_id,
    channel_preferences->>'SMS' as sms_enabled,
    channel_preferences->>'EMAIL' as email_enabled,
    channel_preferences->>'PUSH' as push_enabled,
    channel_preferences->>'PAGER' as pager_enabled
FROM notification_service.user_preferences
WHERE user_id LIKE 'test_%'
ORDER BY user_id
LIMIT 10;

-- ============================================================================
-- Cleanup Script (run separately when needed)
-- ============================================================================

-- To clean up all test data, uncomment and run:
-- DELETE FROM notification_service.user_preferences WHERE user_id LIKE 'test_%';
-- SELECT 'Test data cleaned up' as status;
