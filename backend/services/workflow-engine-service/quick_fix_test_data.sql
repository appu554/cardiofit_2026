-- ============================================================
-- QUICK FIX: Add Test Data for Phase 5 Tests
-- Run this in Supabase SQL Editor
-- ============================================================

-- 1. Add missing Phase 5 fields to workflow_tasks (if not already added)
ALTER TABLE workflow_tasks 
ADD COLUMN IF NOT EXISTS escalation_level INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS escalated BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS escalation_data JSON DEFAULT '{}';

-- 2. Create test workflow definition
INSERT INTO workflow_definitions (id, fhir_id, name, version, bpmn_xml, status, created_at)
VALUES (1, 'workflow-def-1', 'Test Workflow', '1.0', '<bpmn>test</bpmn>', 'active', NOW())
ON CONFLICT (id) DO NOTHING;

-- 3. Create test workflow instances (using your actual column names)
INSERT INTO workflow_instances (
    id, external_id, fhir_id, definition_id, patient_id, status, start_time, updated_at,
    variables, active_gateways, completed_gateways, error_count, recovery_attempts
)
VALUES
(
    1, 'ext-workflow-1', 'workflow-instance-1', 1, 'test-patient-123', 'running', NOW(), NOW(),
    '{"test": true, "phase": 5}', '[]', '[]', 0, 0
),
(
    2, 'ext-workflow-2', 'workflow-instance-2', 1, 'test-patient-456', 'completed', NOW() - INTERVAL '1 hour', NOW(),
    '{"test": true, "phase": 5, "completed": true}', '[]', '["gateway-1"]', 0, 0
),
(
    3, 'ext-workflow-3', 'workflow-instance-3', 1, 'test-patient-789', 'error', NOW() - INTERVAL '30 minutes', NOW(),
    '{"test": true, "phase": 5, "has_error": true}', '[]', '[]', 2, 1
)
ON CONFLICT (id) DO NOTHING;

-- 4. Create test workflow tasks
INSERT INTO workflow_tasks (
    id, fhir_id, workflow_instance_id, name, status, created_at, updated_at,
    input_variables, output_variables, escalation_level, escalated, escalation_data
)
VALUES 
(
    1, 'task-1', 1, 'Test Task 1', 'created', NOW(), NOW(),
    '{"input": "test"}', '{}', 0, FALSE, '{}'
),
(
    2, 'task-2', 1, 'Test Task 2', 'in_progress', NOW(), NOW(),
    '{"input": "test2"}', '{}', 1, TRUE, '{"escalated_to": "supervisor"}'
),
(
    3, 'task-3', 2, 'Test Task 3', 'completed', NOW(), NOW(),
    '{"input": "test3"}', '{"output": "success"}', 0, FALSE, '{}'
),
(
    4, 'task-4', 3, 'Test Task 4', 'error', NOW(), NOW(),
    '{"input": "test4"}', '{}', 2, TRUE, '{"escalated_to": "manager", "error": "timeout"}'
)
ON CONFLICT (id) DO NOTHING;

-- 5. Verify data was created
SELECT 'Test data verification:' as info;
SELECT 'workflow_definitions' as table_name, COUNT(*) as count FROM workflow_definitions WHERE id = 1
UNION ALL
SELECT 'workflow_instances' as table_name, COUNT(*) as count FROM workflow_instances WHERE id IN (1,2,3)
UNION ALL
SELECT 'workflow_tasks' as table_name, COUNT(*) as count FROM workflow_tasks WHERE id IN (1,2,3,4);

-- Show sample data
SELECT 'Sample workflow instance:' as info;
SELECT id, definition_id, patient_id, status, error_count, recovery_attempts 
FROM workflow_instances WHERE id = 1;

SELECT 'Sample workflow task:' as info;
SELECT id, workflow_instance_id, name, status, escalation_level, escalated 
FROM workflow_tasks WHERE id = 1;

-- ============================================================
-- SUCCESS! You should now be able to run:
-- python test_phase5_features.py
-- ============================================================
