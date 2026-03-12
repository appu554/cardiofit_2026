-- ============================================================
-- SUPABASE PHASE 5 SETUP SCRIPT
-- Run this script in your Supabase SQL Editor
-- ============================================================

-- 1. ADD MISSING PHASE 5 FIELDS TO EXISTING TABLES
-- ============================================================

-- Add escalation fields to workflow_tasks table
ALTER TABLE workflow_tasks 
ADD COLUMN IF NOT EXISTS escalation_level INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS escalated BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS escalation_data JSON DEFAULT '{}';

-- Add gateway tracking fields to workflow_instances table
ALTER TABLE workflow_instances 
ADD COLUMN IF NOT EXISTS active_gateways JSON DEFAULT '[]',
ADD COLUMN IF NOT EXISTS completed_gateways JSON DEFAULT '[]';

-- Add error tracking fields to workflow_instances table
ALTER TABLE workflow_instances 
ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS recovery_attempts INTEGER DEFAULT 0;

-- Create indexes for new escalation fields
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalated ON workflow_tasks(escalated);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalation_level ON workflow_tasks(escalation_level);

-- Create indexes for new workflow_instances fields
CREATE INDEX IF NOT EXISTS idx_workflow_instances_error_count ON workflow_instances(error_count);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_last_error_at ON workflow_instances(last_error_at);

-- 2. CREATE TEST WORKFLOW DEFINITION
-- ============================================================

INSERT INTO workflow_definitions (id, fhir_id, name, version, bpmn_xml, status, created_at, updated_at)
VALUES (
    1,
    'workflow-def-1',
    'Test Workflow for Phase 5',
    '1.0',
    '<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="test-workflow" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:task id="Task_1" name="Test Task"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>',
    'active',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    fhir_id = EXCLUDED.fhir_id,
    name = EXCLUDED.name,
    version = EXCLUDED.version,
    bpmn_xml = EXCLUDED.bpmn_xml,
    status = EXCLUDED.status,
    updated_at = NOW();

-- 3. CREATE TEST WORKFLOW INSTANCES
-- ============================================================

INSERT INTO workflow_instances (
    id, fhir_id, definition_id, patient_id, status, start_time, updated_at,
    variables, active_gateways, completed_gateways, error_count, recovery_attempts
)
VALUES
(
    1, 'workflow-instance-1', 1, 'test-patient-123', 'running', NOW(), NOW(),
    '{"test": true, "phase": 5}', '[]', '[]', 0, 0
),
(
    2, 'workflow-instance-2', 1, 'test-patient-456', 'completed', NOW() - INTERVAL '1 hour', NOW(),
    '{"test": true, "phase": 5, "completed": true}', '[]', '["gateway-1"]', 0, 0
),
(
    3, 'workflow-instance-3', 1, 'test-patient-789', 'error', NOW() - INTERVAL '30 minutes', NOW(),
    '{"test": true, "phase": 5, "has_error": true}', '[]', '[]', 2, 1
)
ON CONFLICT (id) DO UPDATE SET
    fhir_id = EXCLUDED.fhir_id,
    definition_id = EXCLUDED.definition_id,
    patient_id = EXCLUDED.patient_id,
    status = EXCLUDED.status,
    variables = EXCLUDED.variables,
    active_gateways = EXCLUDED.active_gateways,
    completed_gateways = EXCLUDED.completed_gateways,
    error_count = EXCLUDED.error_count,
    recovery_attempts = EXCLUDED.recovery_attempts,
    updated_at = NOW();

-- 4. CREATE TEST WORKFLOW TASKS
-- ============================================================

INSERT INTO workflow_tasks (
    id, fhir_id, workflow_instance_id, task_definition_key, name, description, status,
    created_at, updated_at, priority, assignee, candidate_groups,
    input_variables, output_variables, escalation_level, escalated, escalation_data
)
VALUES
(
    1, 'task-1', 1, 'test-task-1', 'Test Task 1', 'First test task for Phase 5', 'created',
    NOW(), NOW(), 'normal', 'test-user', '["doctors", "nurses"]',
    '{"input": "test"}', '{}', 0, FALSE, '{}'
),
(
    2, 'task-2', 1, 'test-task-2', 'Test Task 2', 'Second test task for Phase 5', 'in_progress',
    NOW(), NOW(), 'high', 'test-user-2', '["doctors"]',
    '{"input": "test2"}', '{}', 1, TRUE, '{"escalated_to": "supervisor"}'
),
(
    3, 'task-3', 2, 'test-task-3', 'Test Task 3', 'Completed test task', 'completed',
    NOW() - INTERVAL '1 hour', NOW(), 'normal', 'test-user', '["nurses"]',
    '{"input": "test3"}', '{"output": "success"}', 0, FALSE, '{}'
),
(
    4, 'task-4', 3, 'test-task-4', 'Test Task 4', 'Error test task', 'error',
    NOW() - INTERVAL '30 minutes', NOW(), 'urgent', 'test-user-3', '["doctors"]',
    '{"input": "test4"}', '{}', 2, TRUE, '{"escalated_to": "manager", "error": "timeout"}'
)
ON CONFLICT (id) DO UPDATE SET
    fhir_id = EXCLUDED.fhir_id,
    workflow_instance_id = EXCLUDED.workflow_instance_id,
    task_definition_key = EXCLUDED.task_definition_key,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    priority = EXCLUDED.priority,
    assignee = EXCLUDED.assignee,
    candidate_groups = EXCLUDED.candidate_groups,
    input_variables = EXCLUDED.input_variables,
    output_variables = EXCLUDED.output_variables,
    escalation_level = EXCLUDED.escalation_level,
    escalated = EXCLUDED.escalated,
    escalation_data = EXCLUDED.escalation_data,
    updated_at = NOW();

-- 5. CREATE TEST WORKFLOW EVENTS
-- ============================================================

INSERT INTO workflow_events (
    id, workflow_instance_id, event_type, event_data, created_at, user_id
)
VALUES 
(
    1, 1, 'workflow_started', '{"started_by": "test-user", "reason": "phase5_test"}', NOW(), 'test-user'
),
(
    2, 1, 'task_created', '{"task_id": 1, "task_name": "Test Task 1"}', NOW(), 'system'
),
(
    3, 2, 'workflow_completed', '{"completed_by": "test-user", "duration": 3600}', NOW(), 'test-user'
),
(
    4, 3, 'error_occurred', '{"error_type": "timeout", "task_id": 4}', NOW(), 'system'
)
ON CONFLICT (id) DO UPDATE SET
    workflow_instance_id = EXCLUDED.workflow_instance_id,
    event_type = EXCLUDED.event_type,
    event_data = EXCLUDED.event_data,
    user_id = EXCLUDED.user_id;

-- 6. VERIFY DATA CREATION
-- ============================================================

-- Check that all test data was created successfully
SELECT 'workflow_definitions' as table_name, COUNT(*) as count FROM workflow_definitions WHERE id = 1
UNION ALL
SELECT 'workflow_instances' as table_name, COUNT(*) as count FROM workflow_instances WHERE id IN (1,2,3)
UNION ALL
SELECT 'workflow_tasks' as table_name, COUNT(*) as count FROM workflow_tasks WHERE id IN (1,2,3,4)
UNION ALL
SELECT 'workflow_events' as table_name, COUNT(*) as count FROM workflow_events WHERE id IN (1,2,3,4);

-- Show sample data
SELECT 'Sample workflow instance:' as info;
SELECT id, definition_id, patient_id, status, error_count, recovery_attempts
FROM workflow_instances WHERE id = 1;

SELECT 'Sample workflow task:' as info;
SELECT id, workflow_instance_id, name, status, escalation_level, escalated 
FROM workflow_tasks WHERE id = 1;

-- ============================================================
-- SETUP COMPLETE!
-- ============================================================
-- 
-- After running this script, you should have:
-- ✅ All Phase 5 fields added to existing tables
-- ✅ Test workflow definition (id=1)
-- ✅ Test workflow instances (id=1,2,3)
-- ✅ Test workflow tasks (id=1,2,3,4)
-- ✅ Test workflow events (id=1,2,3,4)
-- 
-- You can now run: python test_phase5_features.py
-- ============================================================
