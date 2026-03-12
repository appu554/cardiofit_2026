-- PostgreSQL: Automated Governance Workflows
-- Part V: Automated Governance Workflows with Event-Driven Architecture

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Governance Workflow Definitions - Define automated governance processes
CREATE TABLE IF NOT EXISTS governance_workflow_definitions (
    workflow_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_name VARCHAR(200) UNIQUE NOT NULL,
    workflow_description TEXT,
    workflow_version VARCHAR(20) NOT NULL DEFAULT '1.0',
    
    -- Workflow configuration
    workflow_type VARCHAR(50) NOT NULL, -- 'kb_deployment', 'safety_incident', 'model_validation', 'dependency_change'
    trigger_conditions JSONB NOT NULL,
    /* Structure:
    {
      "event_types": ["kb_rule_updated", "safety_signal_critical"],
      "conditions": [
        {
          "field": "severity",
          "operator": "equals",
          "value": "critical"
        },
        {
          "field": "estimated_patient_impact",
          "operator": "gte",
          "value": 100
        }
      ],
      "logical_operator": "AND"
    }
    */
    
    -- Workflow steps definition
    workflow_steps JSONB NOT NULL,
    /* Structure:
    [
      {
        "step_id": "notify_clinical_team",
        "step_type": "notification",
        "step_order": 1,
        "required": true,
        "timeout_minutes": 60,
        "parameters": {
          "recipients": ["clinical_pharmacist", "attending_physician"],
          "priority": "urgent",
          "template": "safety_signal_alert"
        },
        "conditions": {
          "execute_if": "signal_severity == 'critical'"
        }
      },
      {
        "step_id": "auto_disable_rules",
        "step_type": "kb_action", 
        "step_order": 2,
        "required": false,
        "timeout_minutes": 5,
        "parameters": {
          "action": "disable_related_rules",
          "kb_services": ["kb_1_dosing", "kb_2_interactions"],
          "confirmation_required": true
        }
      },
      {
        "step_id": "clinical_review",
        "step_type": "human_approval",
        "step_order": 3,
        "required": true,
        "timeout_minutes": 240,
        "parameters": {
          "approvers": ["clinical_lead", "pharmacy_director"],
          "approval_threshold": 2,
          "escalation_after_minutes": 120
        }
      }
    ]
    */
    
    -- Approval and escalation rules
    approval_rules JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "rule_name": "clinical_approval_required",
        "conditions": ["workflow_type == 'kb_deployment'", "clinical_impact == 'high'"],
        "required_roles": ["clinical_lead", "medical_director"],
        "approval_threshold": 2,
        "timeout_hours": 24
      }
    ]
    */
    
    escalation_rules JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "trigger": "step_timeout",
        "escalate_to": ["medical_director", "cio"],
        "escalation_delay_minutes": 60,
        "max_escalations": 3
      }
    ]
    */
    
    -- Workflow status and configuration
    enabled BOOLEAN DEFAULT TRUE,
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'critical')),
    max_concurrent_executions INTEGER DEFAULT 10,
    retry_policy JSONB DEFAULT '{"max_retries": 3, "retry_delay_minutes": 5}',
    
    -- Metadata
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Governance approval for the workflow definition itself
    approved BOOLEAN DEFAULT FALSE,
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ
);

-- Workflow Execution Instances - Track individual workflow executions
CREATE TABLE IF NOT EXISTS governance_workflow_executions (
    execution_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES governance_workflow_definitions(workflow_id),
    
    -- Execution metadata
    execution_status VARCHAR(20) DEFAULT 'running' CHECK (
        execution_status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled', 'timeout')
    ),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- Trigger information
    trigger_event_id UUID,
    trigger_event_type VARCHAR(50) NOT NULL,
    trigger_source VARCHAR(100),
    trigger_data JSONB,
    
    -- Execution context
    clinical_context JSONB,
    /* Structure:
    {
      "patient_cohort": "cardiac_surgery_post_op",
      "affected_kb_services": ["kb_1_dosing", "kb_3_guidelines"],
      "clinical_domain": "cardiology",
      "severity": "major",
      "estimated_impact": 50
    }
    */
    
    -- Current execution state
    current_step_id VARCHAR(100),
    current_step_order INTEGER,
    completed_steps INTEGER DEFAULT 0,
    total_steps INTEGER,
    
    -- Results and outputs
    execution_results JSONB DEFAULT '{}',
    final_outcome VARCHAR(100),
    outcome_reason TEXT,
    
    -- Error handling
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,
    last_error_at TIMESTAMPTZ,
    
    -- Approval tracking
    pending_approvals JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "approval_id": "clinical_review_001",
        "required_role": "clinical_lead",
        "status": "pending",
        "requested_at": "2024-01-15T10:30:00Z",
        "timeout_at": "2024-01-15T14:30:00Z"
      }
    ]
    */
    
    -- Escalations
    escalations_count INTEGER DEFAULT 0,
    last_escalation_at TIMESTAMPTZ,
    escalated_to JSONB DEFAULT '[]',
    
    -- Performance metrics
    execution_efficiency_score DECIMAL(3,2),
    human_intervention_required BOOLEAN DEFAULT FALSE,
    automation_percentage DECIMAL(5,2)
);

-- Workflow Step Execution Log - Detailed log of each step
CREATE TABLE IF NOT EXISTS workflow_step_execution_log (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES governance_workflow_executions(execution_id),
    
    -- Step identification
    step_id VARCHAR(100) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    step_order INTEGER NOT NULL,
    
    -- Execution details
    step_status VARCHAR(20) NOT NULL CHECK (
        step_status IN ('pending', 'running', 'completed', 'failed', 'skipped', 'timeout', 'manual_override')
    ),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- Step inputs and outputs
    step_input JSONB,
    step_output JSONB,
    step_config JSONB,
    
    -- Results and metrics
    success_metrics JSONB,
    /* Structure:
    {
      "notifications_sent": 3,
      "notifications_acknowledged": 2,
      "response_time_minutes": 15,
      "completion_rate": 0.67
    }
    */
    
    -- Error handling
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,
    retry_strategy VARCHAR(50),
    
    -- Human interactions
    assigned_user VARCHAR(100),
    user_action VARCHAR(100), -- 'approved', 'rejected', 'escalated', 'modified'
    user_comments TEXT,
    user_action_at TIMESTAMPTZ,
    
    -- Automation details
    automated BOOLEAN DEFAULT TRUE,
    automation_confidence DECIMAL(3,2),
    human_override BOOLEAN DEFAULT FALSE,
    override_reason TEXT
);

-- Governance Events - Central event log for all governance-related events
CREATE TABLE IF NOT EXISTS governance_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Event classification
    event_type VARCHAR(50) NOT NULL,
    event_category VARCHAR(50) NOT NULL, -- 'kb_change', 'safety_signal', 'model_event', 'workflow_event'
    event_severity VARCHAR(20) NOT NULL CHECK (event_severity IN ('info', 'warning', 'major', 'critical')),
    
    -- Event source
    source_system VARCHAR(100) NOT NULL,
    source_service VARCHAR(100),
    source_user VARCHAR(100),
    
    -- Event content
    event_data JSONB NOT NULL,
    event_summary TEXT NOT NULL,
    
    -- Clinical context
    clinical_domain VARCHAR(100),
    clinical_impact_level VARCHAR(20),
    patient_impact_estimated INTEGER,
    
    -- Workflow triggers
    triggers_workflow BOOLEAN DEFAULT FALSE,
    triggered_workflows UUID[], -- Array of workflow_execution_ids that were triggered
    
    -- Correlation and causation
    related_events UUID[], -- Array of related event IDs
    root_cause_event_id UUID REFERENCES governance_events(event_id),
    
    -- Processing status
    processing_status VARCHAR(20) DEFAULT 'unprocessed' CHECK (
        processing_status IN ('unprocessed', 'processing', 'processed', 'ignored', 'failed')
    ),
    processed_at TIMESTAMPTZ,
    processing_duration_ms INTEGER,
    
    -- Metadata for analytics
    metadata JSONB DEFAULT '{}'
);

-- Governance Decisions Audit - Track all automated and manual governance decisions
CREATE TABLE IF NOT EXISTS governance_decisions_audit (
    decision_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Decision context
    execution_id UUID REFERENCES governance_workflow_executions(execution_id),
    workflow_step_id UUID REFERENCES workflow_step_execution_log(log_id),
    event_id UUID REFERENCES governance_events(event_id),
    
    -- Decision details
    decision_type VARCHAR(50) NOT NULL, -- 'approval', 'rejection', 'escalation', 'modification', 'automation'
    decision_category VARCHAR(50) NOT NULL, -- 'clinical', 'technical', 'safety', 'operational'
    
    -- Decision maker
    decision_maker VARCHAR(100) NOT NULL, -- User ID or 'system'
    decision_maker_role VARCHAR(50),
    automated_decision BOOLEAN DEFAULT FALSE,
    
    -- Decision logic
    decision_criteria JSONB,
    decision_reasoning TEXT,
    confidence_score DECIMAL(3,2),
    
    -- Decision outcome
    decision_outcome VARCHAR(100) NOT NULL,
    outcome_impact JSONB,
    /* Structure:
    {
      "kb_services_affected": ["kb_1_dosing", "kb_2_interactions"],
      "patients_potentially_impacted": 150,
      "estimated_risk_reduction": 0.25,
      "implementation_effort": "low"
    }
    */
    
    -- Approvals and overrides
    requires_additional_approval BOOLEAN DEFAULT FALSE,
    additional_approvers JSONB DEFAULT '[]',
    overridden BOOLEAN DEFAULT FALSE,
    override_reason TEXT,
    overridden_by VARCHAR(100),
    overridden_at TIMESTAMPTZ,
    
    -- Follow-up actions
    follow_up_required BOOLEAN DEFAULT FALSE,
    follow_up_actions JSONB DEFAULT '[]',
    follow_up_timeline INTERVAL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_governance_workflow_definitions_type ON governance_workflow_definitions(workflow_type, enabled);
CREATE INDEX IF NOT EXISTS idx_governance_workflow_executions_status ON governance_workflow_executions(execution_status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_governance_workflow_executions_trigger ON governance_workflow_executions(trigger_event_type, trigger_event_id);
CREATE INDEX IF NOT EXISTS idx_workflow_step_execution_log_execution ON workflow_step_execution_log(execution_id, step_order);
CREATE INDEX IF NOT EXISTS idx_workflow_step_execution_log_status ON workflow_step_execution_log(step_status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_governance_events_timestamp ON governance_events(event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_governance_events_type ON governance_events(event_type, event_category);
CREATE INDEX IF NOT EXISTS idx_governance_events_severity ON governance_events(event_severity, triggers_workflow);
CREATE INDEX IF NOT EXISTS idx_governance_decisions_audit_timestamp ON governance_decisions_audit(decision_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_governance_decisions_audit_type ON governance_decisions_audit(decision_type, automated_decision);

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_conditions_gin ON governance_workflow_definitions USING GIN(trigger_conditions);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_context_gin ON governance_workflow_executions USING GIN(clinical_context);
CREATE INDEX IF NOT EXISTS idx_governance_events_data_gin ON governance_events USING GIN(event_data);
CREATE INDEX IF NOT EXISTS idx_governance_decisions_criteria_gin ON governance_decisions_audit USING GIN(decision_criteria);

-- Functions for automated governance workflows

-- Function to publish governance event
CREATE OR REPLACE FUNCTION publish_governance_event(
    p_event_type VARCHAR(50),
    p_event_category VARCHAR(50),
    p_event_severity VARCHAR(20),
    p_source_system VARCHAR(100),
    p_event_data JSONB,
    p_event_summary TEXT,
    p_clinical_domain VARCHAR(100) DEFAULT NULL,
    p_patient_impact INTEGER DEFAULT 0
)
RETURNS UUID AS $$
DECLARE
    event_id UUID;
    workflow_record RECORD;
    execution_id UUID;
BEGIN
    -- Insert the governance event
    INSERT INTO governance_events (
        event_type, event_category, event_severity, source_system,
        event_data, event_summary, clinical_domain, patient_impact_estimated
    ) VALUES (
        p_event_type, p_event_category, p_event_severity, p_source_system,
        p_event_data, p_event_summary, p_clinical_domain, p_patient_impact
    ) RETURNING event_id INTO event_id;
    
    -- Check for workflows that should be triggered by this event
    FOR workflow_record IN
        SELECT workflow_id, workflow_name, trigger_conditions, workflow_steps
        FROM governance_workflow_definitions
        WHERE enabled = TRUE
          AND p_event_type = ANY(
              SELECT jsonb_array_elements_text(trigger_conditions->'event_types')
          )
    LOOP
        -- Start workflow execution (simplified trigger logic)
        execution_id := start_workflow_execution(
            workflow_record.workflow_id,
            event_id,
            p_event_type,
            p_source_system,
            p_event_data
        );
        
        -- Update the event to indicate it triggered workflows
        UPDATE governance_events
        SET 
            triggers_workflow = TRUE,
            triggered_workflows = COALESCE(triggered_workflows, '{}') || execution_id
        WHERE governance_events.event_id = event_id;
    END LOOP;
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;

-- Function to start workflow execution
CREATE OR REPLACE FUNCTION start_workflow_execution(
    p_workflow_id UUID,
    p_trigger_event_id UUID,
    p_trigger_event_type VARCHAR(50),
    p_trigger_source VARCHAR(100),
    p_trigger_data JSONB
)
RETURNS UUID AS $$
DECLARE
    execution_id UUID;
    workflow_steps JSONB;
    total_steps INTEGER;
BEGIN
    -- Get workflow definition
    SELECT gwd.workflow_steps
    INTO workflow_steps
    FROM governance_workflow_definitions gwd
    WHERE gwd.workflow_id = p_workflow_id;
    
    total_steps := jsonb_array_length(workflow_steps);
    
    -- Create workflow execution instance
    INSERT INTO governance_workflow_executions (
        workflow_id, trigger_event_id, trigger_event_type,
        trigger_source, trigger_data, total_steps,
        execution_status, current_step_order
    ) VALUES (
        p_workflow_id, p_trigger_event_id, p_trigger_event_type,
        p_trigger_source, p_trigger_data, total_steps,
        'running', 1
    ) RETURNING execution_id INTO execution_id;
    
    -- Start the first step
    PERFORM execute_workflow_step(execution_id, 1);
    
    RETURN execution_id;
END;
$$ LANGUAGE plpgsql;

-- Function to execute a workflow step
CREATE OR REPLACE FUNCTION execute_workflow_step(
    p_execution_id UUID,
    p_step_order INTEGER
)
RETURNS BOOLEAN AS $$
DECLARE
    workflow_id UUID;
    workflow_steps JSONB;
    step_config JSONB;
    step_id VARCHAR(100);
    step_type VARCHAR(50);
    log_id UUID;
    step_result BOOLEAN;
BEGIN
    -- Get workflow and step configuration
    SELECT gwe.workflow_id INTO workflow_id
    FROM governance_workflow_executions gwe
    WHERE gwe.execution_id = p_execution_id;
    
    SELECT gwd.workflow_steps INTO workflow_steps
    FROM governance_workflow_definitions gwd
    WHERE gwd.workflow_id = workflow_id;
    
    -- Get the specific step configuration
    step_config := workflow_steps->>(p_step_order - 1)::INTEGER;
    step_id := step_config->>'step_id';
    step_type := step_config->>'step_type';
    
    -- Log step start
    INSERT INTO workflow_step_execution_log (
        execution_id, step_id, step_type, step_order,
        step_status, step_config
    ) VALUES (
        p_execution_id, step_id, step_type, p_step_order,
        'running', step_config
    ) RETURNING log_id INTO log_id;
    
    -- Execute step based on type
    CASE step_type
        WHEN 'notification' THEN
            step_result := execute_notification_step(log_id, step_config);
        WHEN 'kb_action' THEN
            step_result := execute_kb_action_step(log_id, step_config);
        WHEN 'human_approval' THEN
            step_result := execute_human_approval_step(log_id, step_config);
        WHEN 'data_validation' THEN
            step_result := execute_data_validation_step(log_id, step_config);
        ELSE
            step_result := FALSE;
    END CASE;
    
    -- Update step completion
    UPDATE workflow_step_execution_log
    SET 
        step_status = CASE WHEN step_result THEN 'completed' ELSE 'failed' END,
        completed_at = NOW(),
        duration_seconds = EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
    WHERE log_id = log_id;
    
    -- Update workflow execution
    IF step_result THEN
        UPDATE governance_workflow_executions
        SET 
            completed_steps = completed_steps + 1,
            current_step_order = p_step_order + 1,
            execution_status = CASE 
                WHEN completed_steps + 1 >= total_steps THEN 'completed'
                ELSE 'running'
            END
        WHERE execution_id = p_execution_id;
    ELSE
        UPDATE governance_workflow_executions
        SET execution_status = 'failed'
        WHERE execution_id = p_execution_id;
    END IF;
    
    RETURN step_result;
END;
$$ LANGUAGE plpgsql;

-- Helper functions for different step types
CREATE OR REPLACE FUNCTION execute_notification_step(
    p_log_id UUID,
    p_step_config JSONB
)
RETURNS BOOLEAN AS $$
BEGIN
    -- Simplified notification execution
    -- In practice, this would integrate with email, Slack, or other notification systems
    
    UPDATE workflow_step_execution_log
    SET step_output = jsonb_build_object(
        'notifications_sent', 3,
        'notification_channels', ARRAY['email', 'slack'],
        'recipients', p_step_config->'parameters'->'recipients'
    )
    WHERE log_id = p_log_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION execute_kb_action_step(
    p_log_id UUID,
    p_step_config JSONB
)
RETURNS BOOLEAN AS $$
DECLARE
    action_type TEXT;
    kb_services JSONB;
BEGIN
    action_type := p_step_config->'parameters'->>'action';
    kb_services := p_step_config->'parameters'->'kb_services';
    
    -- Simplified KB action execution
    -- In practice, this would call KB service APIs to perform actions
    
    UPDATE workflow_step_execution_log
    SET step_output = jsonb_build_object(
        'action_executed', action_type,
        'affected_services', kb_services,
        'execution_time', NOW()
    )
    WHERE log_id = p_log_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION execute_human_approval_step(
    p_log_id UUID,
    p_step_config JSONB
)
RETURNS BOOLEAN AS $$
BEGIN
    -- This step requires human interaction, so we mark it as pending
    -- and create approval requests
    
    UPDATE workflow_step_execution_log
    SET 
        step_status = 'pending',
        step_output = jsonb_build_object(
            'approval_required', TRUE,
            'required_approvers', p_step_config->'parameters'->'approvers',
            'approval_threshold', p_step_config->'parameters'->'approval_threshold',
            'timeout_minutes', p_step_config->'parameters'->'timeout_minutes'
        )
    WHERE log_id = p_log_id;
    
    -- This step will be completed when approvals are received
    RETURN FALSE; -- Returns false to pause workflow execution
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION execute_data_validation_step(
    p_log_id UUID,
    p_step_config JSONB
)
RETURNS BOOLEAN AS $$
BEGIN
    -- Simplified data validation
    -- In practice, this would perform complex validation logic
    
    UPDATE workflow_step_execution_log
    SET step_output = jsonb_build_object(
        'validation_passed', TRUE,
        'validation_results', jsonb_build_object(
            'data_quality_score', 0.95,
            'completeness_score', 0.98,
            'consistency_score', 0.92
        )
    )
    WHERE log_id = p_log_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to record governance decision
CREATE OR REPLACE FUNCTION record_governance_decision(
    p_execution_id UUID,
    p_decision_type VARCHAR(50),
    p_decision_category VARCHAR(50),
    p_decision_maker VARCHAR(100),
    p_decision_outcome VARCHAR(100),
    p_decision_reasoning TEXT,
    p_automated BOOLEAN DEFAULT FALSE
)
RETURNS UUID AS $$
DECLARE
    decision_id UUID;
BEGIN
    INSERT INTO governance_decisions_audit (
        execution_id, decision_type, decision_category,
        decision_maker, decision_outcome, decision_reasoning,
        automated_decision
    ) VALUES (
        p_execution_id, p_decision_type, p_decision_category,
        p_decision_maker, p_decision_outcome, p_decision_reasoning,
        p_automated
    ) RETURNING decision_id INTO decision_id;
    
    RETURN decision_id;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE governance_workflow_definitions IS 'Configurable automated governance workflows with approval chains and escalation rules';
COMMENT ON TABLE governance_workflow_executions IS 'Individual instances of workflow executions with status tracking and context';
COMMENT ON TABLE workflow_step_execution_log IS 'Detailed execution log for each step in governance workflows';
COMMENT ON TABLE governance_events IS 'Central event log for all governance-related events that may trigger workflows';
COMMENT ON TABLE governance_decisions_audit IS 'Complete audit trail of all automated and manual governance decisions';

COMMENT ON COLUMN governance_workflow_definitions.workflow_steps IS 'JSONB array defining the sequence of steps, conditions, and parameters for workflow execution';
COMMENT ON COLUMN governance_workflow_executions.clinical_context IS 'Clinical context and impact information for the workflow execution';
COMMENT ON COLUMN governance_events.event_data IS 'Complete event payload with all relevant data for workflow processing';
COMMENT ON COLUMN governance_decisions_audit.outcome_impact IS 'JSONB structure tracking the impact and consequences of the governance decision';