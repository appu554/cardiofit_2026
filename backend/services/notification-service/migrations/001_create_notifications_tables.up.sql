-- Migration: Create notification service schema and tables
-- Version: 001
-- Description: Initial schema for notification service including notifications, user preferences, escalation log, and delivery metrics

-- Create schema for notification service
CREATE SCHEMA IF NOT EXISTS notification_service;

-- Set search path
SET search_path TO notification_service, public;

-- ============================================================================
-- Table: notifications
-- Purpose: Track all notification deliveries across channels
-- ============================================================================
CREATE TABLE notification_service.notifications (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id             VARCHAR(255) NOT NULL,
    user_id              VARCHAR(255) NOT NULL,
    channel              VARCHAR(50) NOT NULL,  -- SMS, EMAIL, PUSH, PAGER, VOICE
    priority             INTEGER NOT NULL CHECK (priority BETWEEN 1 AND 5),  -- 1 (highest) to 5 (lowest)
    message              TEXT NOT NULL,
    status               VARCHAR(50) NOT NULL DEFAULT 'PENDING',  -- PENDING, SENDING, SENT, DELIVERED, FAILED, ACKNOWLEDGED
    retry_count          INTEGER NOT NULL DEFAULT 0,
    external_id          VARCHAR(255),          -- Twilio SID, SendGrid message ID, FCM ID, etc.
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at              TIMESTAMP,
    delivered_at         TIMESTAMP,
    acknowledged_at      TIMESTAMP,
    error_message        TEXT,
    metadata             JSONB DEFAULT '{}'::jsonb,

    -- Constraints
    CONSTRAINT valid_channel CHECK (channel IN ('SMS', 'EMAIL', 'PUSH', 'PAGER', 'VOICE', 'IN_APP')),
    CONSTRAINT valid_status CHECK (status IN ('PENDING', 'SENDING', 'SENT', 'DELIVERED', 'FAILED', 'ACKNOWLEDGED', 'CANCELLED')),
    CONSTRAINT valid_timestamps CHECK (
        (sent_at IS NULL OR sent_at >= created_at) AND
        (delivered_at IS NULL OR delivered_at >= sent_at) AND
        (acknowledged_at IS NULL OR acknowledged_at >= delivered_at)
    )
);

-- Notifications indexes for performance
CREATE INDEX idx_notifications_alert_id ON notification_service.notifications(alert_id);
CREATE INDEX idx_notifications_user_id ON notification_service.notifications(user_id);
CREATE INDEX idx_notifications_status ON notification_service.notifications(status) WHERE status IN ('PENDING', 'SENDING', 'FAILED');
CREATE INDEX idx_notifications_created_at ON notification_service.notifications(created_at DESC);
CREATE INDEX idx_notifications_channel_status ON notification_service.notifications(channel, status);
CREATE INDEX idx_notifications_user_status ON notification_service.notifications(user_id, status) WHERE status != 'ACKNOWLEDGED';
CREATE INDEX idx_notifications_metadata ON notification_service.notifications USING GIN(metadata);

-- Comments
COMMENT ON TABLE notification_service.notifications IS 'Tracks all notification deliveries across multiple channels with delivery status and acknowledgment tracking';
COMMENT ON COLUMN notification_service.notifications.priority IS '1 = CRITICAL (immediate), 2 = HIGH, 3 = MODERATE, 4 = LOW, 5 = INFO';
COMMENT ON COLUMN notification_service.notifications.external_id IS 'Third-party service message ID for delivery tracking';
COMMENT ON COLUMN notification_service.notifications.metadata IS 'Additional context: severity, patient_id, alert_type, escalation_level, etc.';

-- ============================================================================
-- Table: user_preferences
-- Purpose: User notification preferences and channel settings
-- ============================================================================
CREATE TABLE notification_service.user_preferences (
    user_id                 VARCHAR(255) PRIMARY KEY,
    channel_preferences     JSONB NOT NULL DEFAULT '{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false, "VOICE": false}'::jsonb,
    severity_channels       JSONB NOT NULL DEFAULT '{
        "CRITICAL": ["PAGER", "SMS", "VOICE"],
        "HIGH": ["SMS", "PUSH"],
        "MODERATE": ["PUSH", "EMAIL"],
        "LOW": ["PUSH"],
        "INFO": ["EMAIL"]
    }'::jsonb,
    quiet_hours_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    quiet_hours_start       INTEGER CHECK (quiet_hours_start IS NULL OR (quiet_hours_start >= 0 AND quiet_hours_start <= 23)),
    quiet_hours_end         INTEGER CHECK (quiet_hours_end IS NULL OR (quiet_hours_end >= 0 AND quiet_hours_end <= 23)),
    max_alerts_per_hour     INTEGER NOT NULL DEFAULT 20 CHECK (max_alerts_per_hour > 0 AND max_alerts_per_hour <= 100),
    fcm_token               VARCHAR(512),
    phone_number            VARCHAR(20),
    email                   VARCHAR(255),
    pager_number            VARCHAR(50),
    language                VARCHAR(10) DEFAULT 'en',
    timezone                VARCHAR(50) DEFAULT 'UTC',
    created_at              TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP NOT NULL DEFAULT NOW()
);

-- User preferences indexes
CREATE INDEX idx_user_preferences_updated_at ON notification_service.user_preferences(updated_at DESC);
CREATE INDEX idx_user_preferences_quiet_hours ON notification_service.user_preferences(quiet_hours_enabled) WHERE quiet_hours_enabled = TRUE;

-- Comments
COMMENT ON TABLE notification_service.user_preferences IS 'User notification preferences including channel settings, quiet hours, and contact information';
COMMENT ON COLUMN notification_service.user_preferences.channel_preferences IS 'Global channel on/off settings per user';
COMMENT ON COLUMN notification_service.user_preferences.severity_channels IS 'Which channels to use for each alert severity level';
COMMENT ON COLUMN notification_service.user_preferences.quiet_hours_start IS 'Hour (0-23) when quiet hours begin';
COMMENT ON COLUMN notification_service.user_preferences.quiet_hours_end IS 'Hour (0-23) when quiet hours end';
COMMENT ON COLUMN notification_service.user_preferences.max_alerts_per_hour IS 'Rate limit for non-critical alerts per user';

-- ============================================================================
-- Table: escalation_log
-- Purpose: Track alert escalation events and outcomes
-- ============================================================================
CREATE TABLE notification_service.escalation_log (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id             VARCHAR(255) NOT NULL,
    escalation_level     INTEGER NOT NULL CHECK (escalation_level > 0 AND escalation_level <= 5),
    escalated_to_user    VARCHAR(255) NOT NULL,
    escalated_to_role    VARCHAR(100),
    escalated_at         TIMESTAMP NOT NULL DEFAULT NOW(),
    acknowledged_at      TIMESTAMP,
    acknowledged_by      VARCHAR(255),
    outcome              VARCHAR(50),  -- ACKNOWLEDGED, ESCALATED_FURTHER, TIMEOUT, CANCELLED
    response_time_ms     INTEGER,
    metadata             JSONB DEFAULT '{}'::jsonb,

    -- Constraints
    CONSTRAINT valid_outcome CHECK (outcome IS NULL OR outcome IN ('ACKNOWLEDGED', 'ESCALATED_FURTHER', 'TIMEOUT', 'CANCELLED', 'AUTO_RESOLVED')),
    CONSTRAINT valid_ack_timing CHECK (acknowledged_at IS NULL OR acknowledged_at >= escalated_at)
);

-- Escalation log indexes
CREATE INDEX idx_escalation_alert_id ON notification_service.escalation_log(alert_id);
CREATE INDEX idx_escalation_user_id ON notification_service.escalation_log(escalated_to_user);
CREATE INDEX idx_escalation_level ON notification_service.escalation_log(escalation_level);
CREATE INDEX idx_escalation_outcome ON notification_service.escalation_log(outcome) WHERE outcome IS NOT NULL;
CREATE INDEX idx_escalation_escalated_at ON notification_service.escalation_log(escalated_at DESC);
CREATE INDEX idx_escalation_pending ON notification_service.escalation_log(alert_id, escalation_level) WHERE acknowledged_at IS NULL;

-- Comments
COMMENT ON TABLE notification_service.escalation_log IS 'Audit trail of alert escalations with acknowledgment tracking and response times';
COMMENT ON COLUMN notification_service.escalation_log.escalation_level IS 'Escalation tier: 1 = primary contact, 2 = charge nurse, 3 = attending, 4 = senior physician, 5 = department head';
COMMENT ON COLUMN notification_service.escalation_log.response_time_ms IS 'Time from escalation to acknowledgment in milliseconds';

-- ============================================================================
-- Table: delivery_metrics
-- Purpose: Daily aggregated metrics for notification delivery analytics
-- ============================================================================
CREATE TABLE notification_service.delivery_metrics (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date                 DATE NOT NULL,
    channel              VARCHAR(50) NOT NULL,
    hour                 INTEGER CHECK (hour >= 0 AND hour <= 23),
    total_sent           INTEGER NOT NULL DEFAULT 0,
    total_delivered      INTEGER NOT NULL DEFAULT 0,
    total_failed         INTEGER NOT NULL DEFAULT 0,
    total_acknowledged   INTEGER NOT NULL DEFAULT 0,
    avg_delivery_time_ms INTEGER,
    p50_delivery_time_ms INTEGER,
    p95_delivery_time_ms INTEGER,
    p99_delivery_time_ms INTEGER,
    error_count_by_type  JSONB DEFAULT '{}'::jsonb,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_channel_metrics CHECK (channel IN ('SMS', 'EMAIL', 'PUSH', 'PAGER', 'VOICE', 'IN_APP', 'ALL')),
    CONSTRAINT valid_counts CHECK (
        total_sent >= 0 AND
        total_delivered >= 0 AND
        total_failed >= 0 AND
        total_acknowledged >= 0 AND
        total_delivered <= total_sent AND
        total_failed <= total_sent
    ),
    CONSTRAINT unique_date_channel_hour UNIQUE (date, channel, hour)
);

-- Delivery metrics indexes
CREATE INDEX idx_delivery_metrics_date_channel ON notification_service.delivery_metrics(date DESC, channel);
CREATE INDEX idx_delivery_metrics_date ON notification_service.delivery_metrics(date DESC);
CREATE INDEX idx_delivery_metrics_channel ON notification_service.delivery_metrics(channel);

-- Comments
COMMENT ON TABLE notification_service.delivery_metrics IS 'Aggregated delivery statistics for monitoring and analytics';
COMMENT ON COLUMN notification_service.delivery_metrics.hour IS 'Hour of day (0-23) for hourly aggregation, NULL for daily aggregate';
COMMENT ON COLUMN notification_service.delivery_metrics.error_count_by_type IS 'Error type breakdown: {"TIMEOUT": 5, "API_ERROR": 3, "INVALID_RECIPIENT": 2}';

-- ============================================================================
-- Table: alert_fatigue_history
-- Purpose: Track user alert fatigue patterns for suppression and bundling
-- ============================================================================
CREATE TABLE notification_service.alert_fatigue_history (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              VARCHAR(255) NOT NULL,
    alert_id             VARCHAR(255) NOT NULL,
    patient_id           VARCHAR(255),
    alert_type           VARCHAR(100) NOT NULL,
    severity             VARCHAR(50) NOT NULL,
    suppressed           BOOLEAN NOT NULL DEFAULT FALSE,
    suppression_reason   VARCHAR(100),  -- RATE_LIMIT, DUPLICATE, BUNDLED, QUIET_HOURS
    bundled_with         VARCHAR(255),  -- Reference to bundle notification_id
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at           TIMESTAMP NOT NULL,  -- For cleanup

    -- Constraints
    CONSTRAINT valid_severity_fatigue CHECK (severity IN ('CRITICAL', 'HIGH', 'MODERATE', 'LOW', 'INFO')),
    CONSTRAINT valid_suppression CHECK (
        (suppressed = FALSE AND suppression_reason IS NULL) OR
        (suppressed = TRUE AND suppression_reason IS NOT NULL)
    )
);

-- Alert fatigue indexes
CREATE INDEX idx_fatigue_user_created ON notification_service.alert_fatigue_history(user_id, created_at DESC);
CREATE INDEX idx_fatigue_expires_at ON notification_service.alert_fatigue_history(expires_at) WHERE expires_at < NOW();
CREATE INDEX idx_fatigue_suppressed ON notification_service.alert_fatigue_history(user_id, suppressed) WHERE suppressed = TRUE;
CREATE INDEX idx_fatigue_patient ON notification_service.alert_fatigue_history(patient_id, alert_type, created_at) WHERE patient_id IS NOT NULL;

-- Comments
COMMENT ON TABLE notification_service.alert_fatigue_history IS 'Historical record of alerts per user for fatigue detection and duplicate suppression';
COMMENT ON COLUMN notification_service.alert_fatigue_history.expires_at IS 'Cleanup timestamp - typically NOW() + 24 hours';

-- ============================================================================
-- Table: notification_templates
-- Purpose: Reusable message templates for different alert types
-- ============================================================================
CREATE TABLE notification_service.notification_templates (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name        VARCHAR(100) NOT NULL UNIQUE,
    alert_type           VARCHAR(100) NOT NULL,
    channel              VARCHAR(50) NOT NULL,
    subject_template     TEXT,  -- For EMAIL
    body_template        TEXT NOT NULL,
    variables            JSONB DEFAULT '[]'::jsonb,  -- List of required variables
    active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_template_channel CHECK (channel IN ('SMS', 'EMAIL', 'PUSH', 'PAGER', 'VOICE', 'IN_APP'))
);

-- Notification templates indexes
CREATE INDEX idx_templates_alert_type ON notification_service.notification_templates(alert_type, channel) WHERE active = TRUE;
CREATE INDEX idx_templates_name ON notification_service.notification_templates(template_name);

-- Comments
COMMENT ON TABLE notification_service.notification_templates IS 'Message templates with variable substitution for consistent notification formatting';
COMMENT ON COLUMN notification_service.notification_templates.variables IS 'Array of variable names used in template: ["patient_id", "severity", "risk_score"]';

-- ============================================================================
-- Views for common queries
-- ============================================================================

-- View: Active unacknowledged notifications per user
CREATE VIEW notification_service.v_pending_notifications AS
SELECT
    user_id,
    channel,
    COUNT(*) as pending_count,
    MIN(created_at) as oldest_notification,
    MAX(priority) as highest_priority
FROM notification_service.notifications
WHERE status IN ('PENDING', 'SENDING', 'SENT', 'DELIVERED')
  AND acknowledged_at IS NULL
GROUP BY user_id, channel;

COMMENT ON VIEW notification_service.v_pending_notifications IS 'Summary of pending unacknowledged notifications per user and channel';

-- View: Delivery success rates by channel
CREATE VIEW notification_service.v_delivery_success_rates AS
SELECT
    channel,
    date,
    total_sent,
    total_delivered,
    total_failed,
    CASE WHEN total_sent > 0
        THEN ROUND((total_delivered::numeric / total_sent::numeric) * 100, 2)
        ELSE 0
    END as delivery_success_rate,
    CASE WHEN total_sent > 0
        THEN ROUND((total_failed::numeric / total_sent::numeric) * 100, 2)
        ELSE 0
    END as failure_rate
FROM notification_service.delivery_metrics
WHERE channel != 'ALL'
ORDER BY date DESC, channel;

COMMENT ON VIEW notification_service.v_delivery_success_rates IS 'Calculated success and failure rates per channel per day';

-- View: Escalation effectiveness
CREATE VIEW notification_service.v_escalation_effectiveness AS
SELECT
    escalation_level,
    COUNT(*) as total_escalations,
    COUNT(acknowledged_at) as acknowledged_count,
    COUNT(*) - COUNT(acknowledged_at) as unacknowledged_count,
    ROUND(AVG(response_time_ms), 0) as avg_response_time_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time_ms) as p95_response_time_ms
FROM notification_service.escalation_log
WHERE escalated_at >= NOW() - INTERVAL '30 days'
GROUP BY escalation_level
ORDER BY escalation_level;

COMMENT ON VIEW notification_service.v_escalation_effectiveness IS 'Escalation acknowledgment rates and response times by level';

-- ============================================================================
-- Functions for common operations
-- ============================================================================

-- Function: Update user preferences timestamp
CREATE OR REPLACE FUNCTION notification_service.update_user_preferences_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Auto-update user preferences timestamp
CREATE TRIGGER trg_user_preferences_updated_at
    BEFORE UPDATE ON notification_service.user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION notification_service.update_user_preferences_timestamp();

-- Function: Calculate response time on acknowledgment
CREATE OR REPLACE FUNCTION notification_service.calculate_escalation_response_time()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.acknowledged_at IS NOT NULL AND OLD.acknowledged_at IS NULL THEN
        NEW.response_time_ms = EXTRACT(EPOCH FROM (NEW.acknowledged_at - NEW.escalated_at)) * 1000;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Auto-calculate response time
CREATE TRIGGER trg_escalation_response_time
    BEFORE UPDATE ON notification_service.escalation_log
    FOR EACH ROW
    EXECUTE FUNCTION notification_service.calculate_escalation_response_time();

-- Function: Cleanup expired fatigue history
CREATE OR REPLACE FUNCTION notification_service.cleanup_expired_fatigue_history()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM notification_service.alert_fatigue_history
    WHERE expires_at < NOW() - INTERVAL '7 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION notification_service.cleanup_expired_fatigue_history() IS 'Remove fatigue history records older than 7 days past expiration';

-- ============================================================================
-- Grant permissions (adjust based on application user)
-- ============================================================================

-- Grant usage on schema
GRANT USAGE ON SCHEMA notification_service TO cardiofit;

-- Grant table permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA notification_service TO cardiofit;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA notification_service TO cardiofit;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA notification_service TO cardiofit;

-- Grant view permissions (views are tables, so this works)
GRANT SELECT ON notification_service.v_pending_notifications TO cardiofit;
GRANT SELECT ON notification_service.v_delivery_success_rates TO cardiofit;
GRANT SELECT ON notification_service.v_escalation_effectiveness TO cardiofit;

-- ============================================================================
-- Initial seed data
-- ============================================================================

-- Default notification templates
INSERT INTO notification_service.notification_templates (template_name, alert_type, channel, subject_template, body_template, variables) VALUES
('sepsis_alert_sms', 'SEPSIS_ALERT', 'SMS', NULL,
 'CRITICAL: {{patient_id}} Sepsis Risk {{risk_score}}% - {{location}}',
 '["patient_id", "risk_score", "location"]'::jsonb),

('sepsis_alert_email', 'SEPSIS_ALERT', 'EMAIL',
 'CRITICAL: Sepsis Alert - Patient {{patient_id}}',
 'Sepsis alert detected for patient {{patient_id}} with {{risk_score}}% confidence.\n\nLocation: {{location}}\n\nRecommended Actions:\n{{recommendations}}\n\nTimestamp: {{timestamp}}',
 '["patient_id", "risk_score", "location", "recommendations", "timestamp"]'::jsonb),

('deterioration_alert_push', 'DETERIORATION_ALERT', 'PUSH', NULL,
 'Patient Deterioration: {{patient_id}} - {{severity}} - Review vital signs immediately',
 '["patient_id", "severity"]'::jsonb),

('vital_sign_anomaly_sms', 'VITAL_SIGN_ANOMALY', 'SMS', NULL,
 '{{severity}}: {{patient_id}} {{parameter}} abnormal - {{value}} - {{location}}',
 '["severity", "patient_id", "parameter", "value", "location"]'::jsonb);

-- Migration complete
SELECT 'Migration 001_create_notifications_tables.up.sql completed successfully' as status;
