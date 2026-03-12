-- Migration Rollback: Drop notification service schema and tables
-- Version: 001
-- Description: Rollback for initial notification service schema

-- Set search path
SET search_path TO notification_service, public;

-- ============================================================================
-- Drop views
-- ============================================================================
DROP VIEW IF EXISTS notification_service.v_escalation_effectiveness;
DROP VIEW IF EXISTS notification_service.v_delivery_success_rates;
DROP VIEW IF EXISTS notification_service.v_pending_notifications;

-- ============================================================================
-- Drop triggers
-- ============================================================================
DROP TRIGGER IF EXISTS trg_escalation_response_time ON notification_service.escalation_log;
DROP TRIGGER IF EXISTS trg_user_preferences_updated_at ON notification_service.user_preferences;

-- ============================================================================
-- Drop functions
-- ============================================================================
DROP FUNCTION IF EXISTS notification_service.cleanup_expired_fatigue_history();
DROP FUNCTION IF EXISTS notification_service.calculate_escalation_response_time();
DROP FUNCTION IF EXISTS notification_service.update_user_preferences_timestamp();

-- ============================================================================
-- Drop tables (in reverse dependency order)
-- ============================================================================
DROP TABLE IF EXISTS notification_service.notification_templates CASCADE;
DROP TABLE IF EXISTS notification_service.alert_fatigue_history CASCADE;
DROP TABLE IF EXISTS notification_service.delivery_metrics CASCADE;
DROP TABLE IF EXISTS notification_service.escalation_log CASCADE;
DROP TABLE IF EXISTS notification_service.user_preferences CASCADE;
DROP TABLE IF EXISTS notification_service.notifications CASCADE;

-- ============================================================================
-- Drop schema
-- ============================================================================
DROP SCHEMA IF EXISTS notification_service CASCADE;

-- Rollback complete
SELECT 'Migration 001_create_notifications_tables.down.sql rollback completed successfully' as status;
