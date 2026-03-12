-- TimescaleDB: Real-time Safety Monitoring Platform
-- Part II: Real-Time Analytics Platform - Safety Signal Detection

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Unified safety signals table across all KBs
CREATE TABLE IF NOT EXISTS safety_signals_unified (
    time TIMESTAMPTZ NOT NULL,
    signal_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    
    -- Signal classification
    signal_type VARCHAR(50) NOT NULL,
    kb_source VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'major', 'moderate', 'minor')),
    
    -- Context
    transaction_id VARCHAR(100),
    evidence_envelope_id UUID,
    patient_cohort VARCHAR(100),
    
    -- Signal details as JSONB for flexibility
    signal_data JSONB NOT NULL,
    /* Example structures:
    
    Drug-Drug Interaction Signal:
    {
      "type": "drug_interaction_detected",
      "drug1": {"name": "warfarin", "rxnorm": "11289"},
      "drug2": {"name": "simvastatin", "rxnorm": "36567"},
      "interaction_severity": "major",
      "mechanism": "CYP3A4_inhibition",
      "clinical_effect": "increased_bleeding_risk",
      "affected_patients": 23,
      "confidence": 0.94
    }
    
    Guideline Safety Conflict:
    {
      "type": "guideline_safety_conflict",
      "kb1": "kb_guidelines",
      "kb1_rule": "htn_ckd_acei_recommendation",
      "kb2": "kb_safety",
      "kb2_rule": "hyperkalemia_risk_assessment",
      "conflict": "ACEi recommended but high K+ risk",
      "affected_patients": 145,
      "detection_confidence": 0.92,
      "clinical_impact": "potential_hyperkalemia"
    }
    
    Dosing Safety Alert:
    {
      "type": "dose_safety_violation",
      "drug": {"name": "digoxin", "rxnorm": "3407"},
      "calculated_dose": 0.375,
      "max_safe_dose": 0.25,
      "safety_factor": "renal_impairment",
      "egfr": 28,
      "risk_level": "high",
      "recommendation": "reduce_dose_50_percent"
    }
    */
    
    -- Detection metadata
    detection_method VARCHAR(50) NOT NULL,
    detection_version VARCHAR(20),
    detection_algorithm JSONB,
    false_positive_probability DECIMAL(3,2),
    
    -- Clinical context
    patient_demographics JSONB,
    clinical_conditions JSONB DEFAULT '[]',
    concurrent_medications JSONB DEFAULT '[]',
    
    -- Impact assessment
    estimated_patient_impact INTEGER,
    clinical_significance_score DECIMAL(3,2),
    urgency_score DECIMAL(3,2),
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'new' CHECK (
        status IN ('new', 'investigating', 'confirmed', 'false_positive', 'resolved')
    ),
    investigated_by VARCHAR(100),
    investigated_at TIMESTAMPTZ,
    resolution_notes TEXT,
    
    -- Actions taken
    actions_taken JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "action": "automated_alert_sent",
        "timestamp": "2024-01-15T10:30:00Z",
        "target": "clinical_team",
        "details": {...}
      },
      {
        "action": "kb_rule_disabled",
        "timestamp": "2024-01-15T11:15:00Z",
        "kb": "kb_1_dosing",
        "rule_id": "digoxin_renal_adjustment_v2"
      }
    ]
    */
    
    -- Reviewer information
    reviewed_by VARCHAR(100),
    review_timestamp TIMESTAMPTZ,
    review_notes TEXT,
    action_taken TEXT,
    
    PRIMARY KEY (time, signal_id)
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable(
    'safety_signals_unified', 
    'time',
    chunk_time_interval => INTERVAL '1 hour',
    create_default_indexes => FALSE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_safety_signals_signal_type ON safety_signals_unified(signal_type, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_signals_kb_source ON safety_signals_unified(kb_source, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_signals_severity ON safety_signals_unified(severity, time DESC);
CREATE INDEX IF NOT EXISTS idx_safety_signals_status ON safety_signals_unified(status);
CREATE INDEX IF NOT EXISTS idx_safety_signals_transaction ON safety_signals_unified(transaction_id);
CREATE INDEX IF NOT EXISTS idx_safety_signals_envelope ON safety_signals_unified(evidence_envelope_id);

-- GIN index for signal data
CREATE INDEX IF NOT EXISTS idx_safety_signals_data_gin ON safety_signals_unified USING GIN(signal_data);
CREATE INDEX IF NOT EXISTS idx_safety_signals_actions_gin ON safety_signals_unified USING GIN(actions_taken);

-- Continuous aggregates for real-time monitoring

-- Hourly signal summary
CREATE MATERIALIZED VIEW safety_signals_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    kb_source,
    signal_type,
    severity,
    COUNT(*) as signal_count,
    COUNT(DISTINCT transaction_id) as unique_transactions,
    AVG(clinical_significance_score) as avg_clinical_significance,
    MAX(clinical_significance_score) as max_clinical_significance,
    AVG(false_positive_probability) as avg_fp_probability,
    SUM(estimated_patient_impact) as total_patient_impact,
    COUNT(*) FILTER (WHERE status = 'confirmed') as confirmed_signals,
    COUNT(*) FILTER (WHERE status = 'false_positive') as false_positive_signals,
    AVG(EXTRACT(EPOCH FROM (investigated_at - time))/60)::NUMERIC(10,2) as avg_investigation_time_minutes
FROM safety_signals_unified
WHERE time > NOW() - INTERVAL '7 days'
GROUP BY hour, kb_source, signal_type, severity
WITH NO DATA;

-- Daily signal trends
CREATE MATERIALIZED VIEW safety_signals_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    kb_source,
    signal_type,
    COUNT(*) as signal_count,
    COUNT(DISTINCT patient_cohort) as unique_cohorts,
    AVG(clinical_significance_score) as avg_clinical_significance,
    COUNT(*) FILTER (WHERE severity IN ('critical', 'major')) as high_severity_count,
    COUNT(*) FILTER (WHERE status = 'confirmed') as confirmed_count,
    COUNT(*) FILTER (WHERE status = 'false_positive') as false_positive_count,
    ROUND(
        COUNT(*) FILTER (WHERE status = 'false_positive')::DECIMAL / 
        NULLIF(COUNT(*) FILTER (WHERE status IN ('confirmed', 'false_positive')), 0) * 100, 
        2
    ) as false_positive_rate,
    SUM(estimated_patient_impact) as total_patient_impact,
    jsonb_object_agg(
        severity, 
        COUNT(*) FILTER (WHERE severity = safety_signals_unified.severity)
    ) as severity_breakdown
FROM safety_signals_unified
WHERE time > NOW() - INTERVAL '90 days'
GROUP BY day, kb_source, signal_type
WITH NO DATA;

-- Weekly KB performance summary
CREATE MATERIALIZED VIEW kb_safety_performance_weekly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 week', time) AS week,
    kb_source,
    COUNT(*) as total_signals,
    COUNT(*) FILTER (WHERE severity = 'critical') as critical_signals,
    COUNT(*) FILTER (WHERE severity = 'major') as major_signals,
    COUNT(*) FILTER (WHERE status = 'confirmed') as confirmed_signals,
    COUNT(*) FILTER (WHERE status = 'false_positive') as false_positive_signals,
    ROUND(
        COUNT(*) FILTER (WHERE status = 'false_positive')::DECIMAL / 
        NULLIF(COUNT(*) FILTER (WHERE status IN ('confirmed', 'false_positive')), 0) * 100, 
        2
    ) as false_positive_rate,
    AVG(clinical_significance_score) as avg_clinical_significance,
    AVG(false_positive_probability) as avg_predicted_fp_rate,
    SUM(estimated_patient_impact) as total_patient_impact,
    AVG(EXTRACT(EPOCH FROM (investigated_at - time))/3600)::NUMERIC(8,2) as avg_investigation_time_hours
FROM safety_signals_unified
WHERE time > NOW() - INTERVAL '1 year'
GROUP BY week, kb_source
WITH NO DATA;

-- Set up refresh policies
SELECT add_continuous_aggregate_policy('safety_signals_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '30 minutes');

SELECT add_continuous_aggregate_policy('safety_signals_daily',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '6 hours');

SELECT add_continuous_aggregate_policy('kb_safety_performance_weekly',
    start_offset => INTERVAL '2 weeks',
    end_offset => INTERVAL '1 week',
    schedule_interval => INTERVAL '1 day');

-- Real-time alerting table
CREATE TABLE IF NOT EXISTS safety_signal_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    signal_id UUID NOT NULL,
    alert_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Alert classification
    alert_type VARCHAR(50) NOT NULL,
    alert_priority VARCHAR(20) NOT NULL CHECK (alert_priority IN ('immediate', 'urgent', 'normal', 'low')),
    
    -- Target audience
    target_roles TEXT[] NOT NULL,
    target_users TEXT[],
    
    -- Alert content
    alert_title TEXT NOT NULL,
    alert_message TEXT NOT NULL,
    alert_details JSONB,
    
    -- Delivery tracking
    delivery_methods TEXT[] DEFAULT '{}', -- email, sms, push, dashboard
    sent_at TIMESTAMPTZ,
    delivery_status JSONB DEFAULT '{}',
    
    -- Response tracking
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    escalated BOOLEAN DEFAULT FALSE,
    escalated_at TIMESTAMPTZ,
    escalation_level INTEGER DEFAULT 0,
    
    -- Resolution
    resolved BOOLEAN DEFAULT FALSE,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT
);

-- Create indexes for alert management
CREATE INDEX IF NOT EXISTS idx_safety_signal_alerts_signal ON safety_signal_alerts(signal_id);
CREATE INDEX IF NOT EXISTS idx_safety_signal_alerts_timestamp ON safety_signal_alerts(alert_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_safety_signal_alerts_priority ON safety_signal_alerts(alert_priority, acknowledged);
CREATE INDEX IF NOT EXISTS idx_safety_signal_alerts_unresolved ON safety_signal_alerts(resolved, alert_timestamp DESC) WHERE NOT resolved;

-- Signal detection rules configuration
CREATE TABLE IF NOT EXISTS safety_signal_detection_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_description TEXT,
    
    -- Rule scope
    kb_sources TEXT[],
    signal_types TEXT[],
    
    -- Detection logic
    detection_query JSONB NOT NULL,
    /* Example structure:
    {
      "conditions": [
        {
          "field": "clinical_significance_score",
          "operator": "gt",
          "value": 0.8
        },
        {
          "field": "estimated_patient_impact",
          "operator": "gte",
          "value": 10
        }
      ],
      "time_window": "1 hour",
      "aggregation": "count",
      "threshold": 3
    }
    */
    
    -- Thresholds
    severity_thresholds JSONB NOT NULL,
    /* Structure:
    {
      "critical": {"min_significance": 0.9, "min_impact": 100},
      "major": {"min_significance": 0.7, "min_impact": 50},
      "moderate": {"min_significance": 0.5, "min_impact": 20},
      "minor": {"min_significance": 0.3, "min_impact": 5}
    }
    */
    
    -- Actions
    automated_actions JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "action": "send_alert",
        "parameters": {
          "roles": ["clinical_pharmacist", "attending_physician"],
          "priority": "urgent"
        }
      },
      {
        "action": "disable_kb_rule",
        "parameters": {
          "kb": "kb_1_dosing",
          "rule_pattern": "high_risk_dosing_*"
        }
      }
    ]
    */
    
    -- Configuration
    enabled BOOLEAN DEFAULT TRUE,
    run_frequency INTERVAL DEFAULT '5 minutes',
    last_run TIMESTAMPTZ,
    next_run TIMESTAMPTZ,
    
    -- Metadata
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Functions for safety signal management

-- Function to detect safety signals based on rules
CREATE OR REPLACE FUNCTION detect_safety_signals()
RETURNS INTEGER AS $$
DECLARE
    rule_record RECORD;
    detected_signals INTEGER := 0;
    signal_data JSONB;
BEGIN
    -- Process each enabled detection rule
    FOR rule_record IN 
        SELECT * FROM safety_signal_detection_rules 
        WHERE enabled = TRUE 
          AND (next_run IS NULL OR next_run <= NOW())
    LOOP
        -- This would contain the actual signal detection logic
        -- For now, we'll create a placeholder implementation
        
        -- Update the rule's last run timestamp
        UPDATE safety_signal_detection_rules
        SET 
            last_run = NOW(),
            next_run = NOW() + run_frequency
        WHERE id = rule_record.id;
        
        detected_signals := detected_signals + 1;
    END LOOP;
    
    RETURN detected_signals;
END;
$$ LANGUAGE plpgsql;

-- Function to create safety signal
CREATE OR REPLACE FUNCTION create_safety_signal(
    p_signal_type VARCHAR(50),
    p_kb_source VARCHAR(50),
    p_severity VARCHAR(20),
    p_signal_data JSONB,
    p_detection_method VARCHAR(50) DEFAULT 'automated',
    p_estimated_impact INTEGER DEFAULT 1
)
RETURNS UUID AS $$
DECLARE
    signal_id UUID;
BEGIN
    INSERT INTO safety_signals_unified (
        time,
        signal_type,
        kb_source,
        severity,
        signal_data,
        detection_method,
        estimated_patient_impact,
        clinical_significance_score
    ) VALUES (
        NOW(),
        p_signal_type,
        p_kb_source,
        p_severity,
        p_signal_data,
        p_detection_method,
        p_estimated_impact,
        CASE p_severity
            WHEN 'critical' THEN 1.0
            WHEN 'major' THEN 0.8
            WHEN 'moderate' THEN 0.6
            WHEN 'minor' THEN 0.4
            ELSE 0.5
        END
    ) RETURNING signal_id INTO signal_id;
    
    -- Trigger automated actions if configured
    PERFORM trigger_signal_actions(signal_id);
    
    RETURN signal_id;
END;
$$ LANGUAGE plpgsql;

-- Function to trigger automated actions for a signal
CREATE OR REPLACE FUNCTION trigger_signal_actions(p_signal_id UUID)
RETURNS VOID AS $$
DECLARE
    signal_record RECORD;
    rule_record RECORD;
    action_record JSONB;
BEGIN
    -- Get the signal details
    SELECT * INTO signal_record
    FROM safety_signals_unified
    WHERE signal_id = p_signal_id;
    
    -- Find applicable rules
    FOR rule_record IN
        SELECT * FROM safety_signal_detection_rules
        WHERE enabled = TRUE
          AND (kb_sources IS NULL OR signal_record.kb_source = ANY(kb_sources))
          AND (signal_types IS NULL OR signal_record.signal_type = ANY(signal_types))
    LOOP
        -- Process each automated action
        FOR action_record IN
            SELECT * FROM jsonb_array_elements(rule_record.automated_actions)
        LOOP
            CASE action_record->>'action'
                WHEN 'send_alert' THEN
                    -- Create alert
                    INSERT INTO safety_signal_alerts (
                        signal_id,
                        alert_type,
                        alert_priority,
                        target_roles,
                        alert_title,
                        alert_message,
                        alert_details
                    ) VALUES (
                        p_signal_id,
                        'safety_signal_detected',
                        COALESCE(action_record->'parameters'->>'priority', 'normal'),
                        ARRAY(SELECT jsonb_array_elements_text(action_record->'parameters'->'roles')),
                        'Safety Signal Detected: ' || signal_record.signal_type,
                        'A ' || signal_record.severity || ' safety signal has been detected.',
                        action_record->'parameters'
                    );
                
                WHEN 'log_event' THEN
                    -- Log the action (this could integrate with external logging systems)
                    NULL; -- Placeholder
                
                ELSE
                    -- Unknown action type
                    NULL;
            END CASE;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Set up retention policy (keep raw data for 1 year, aggregates longer)
SELECT add_retention_policy('safety_signals_unified', INTERVAL '1 year');

-- Comments for documentation
COMMENT ON TABLE safety_signals_unified IS 'Real-time safety signal detection across all KB services using TimescaleDB';
COMMENT ON TABLE safety_signal_alerts IS 'Alert management system for safety signals with delivery and acknowledgment tracking';
COMMENT ON TABLE safety_signal_detection_rules IS 'Configurable rules for automated safety signal detection and response';

COMMENT ON COLUMN safety_signals_unified.signal_data IS 'Flexible JSONB structure containing signal-specific details and context';
COMMENT ON COLUMN safety_signals_unified.detection_algorithm IS 'JSONB metadata about the algorithm used for signal detection';
COMMENT ON COLUMN safety_signals_unified.actions_taken IS 'JSONB array of automated and manual actions taken in response to the signal';