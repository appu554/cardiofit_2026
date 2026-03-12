-- =============================================================================
-- MIGRATION 003: Production Hardening Guardrails
-- Purpose: Make KB projections physically immutable, add atomic activation
-- Reference: Clinical Platform Architecture Review - Hardening Recommendations
-- =============================================================================

BEGIN;

-- =============================================================================
-- CREATE GOVERNANCE SCHEMA (must come first, before governance.decision_lineage)
-- =============================================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_namespace WHERE nspname = 'governance') THEN
        CREATE SCHEMA governance;
    END IF;
END
$$;

-- =============================================================================
-- GAP 1: DATABASE-LEVEL WRITE GUARDRAILS
-- Make KB projection views physically immutable to prevent accidental writes
-- =============================================================================

-- Revoke all write permissions from PUBLIC on projection views
REVOKE INSERT, UPDATE, DELETE ON kb1_renal_dosing FROM PUBLIC;
REVOKE INSERT, UPDATE, DELETE ON kb4_safety_signals FROM PUBLIC;
REVOKE INSERT, UPDATE, DELETE ON kb5_interactions FROM PUBLIC;
REVOKE INSERT, UPDATE, DELETE ON kb6_formulary FROM PUBLIC;
REVOKE INSERT, UPDATE, DELETE ON kb16_lab_ranges FROM PUBLIC;

-- Create INSTEAD OF rules to silently block any write attempts
-- This is defense-in-depth: even if permissions fail, writes are blocked

CREATE OR REPLACE RULE kb1_readonly_insert AS
ON INSERT TO kb1_renal_dosing
DO INSTEAD NOTHING;

CREATE OR REPLACE RULE kb1_readonly_update AS
ON UPDATE TO kb1_renal_dosing
DO INSTEAD NOTHING;

CREATE OR REPLACE RULE kb1_readonly_delete AS
ON DELETE TO kb1_renal_dosing
DO INSTEAD NOTHING;

CREATE OR REPLACE RULE kb4_readonly_insert AS
ON INSERT TO kb4_safety_signals
DO INSTEAD NOTHING;

CREATE OR REPLACE RULE kb4_readonly_update AS
ON UPDATE TO kb4_safety_signals
DO INSTEAD NOTHING;

CREATE OR REPLACE RULE kb4_readonly_delete AS
ON DELETE TO kb4_safety_signals
DO INSTEAD NOTHING;

-- Note: kb5_interactions, kb6_formulary, kb16_lab_ranges are views over
-- denormalized tables (interaction_matrix, formulary_coverage, lab_reference_ranges)
-- Rules cannot be created on simple views, but permissions are revoked

-- =============================================================================
-- GAP 2: ATOMIC FACT ACTIVATION
-- Transaction-guarded function for safe fact status transitions
-- Ensures projections are always consistent with fact store
-- =============================================================================

-- Fact lifecycle state machine
-- DRAFT → APPROVED → ACTIVE → SUPERSEDED
--                  ↘ DEPRECATED

CREATE OR REPLACE FUNCTION activate_fact(
    p_fact_id UUID,
    p_activated_by VARCHAR(255) DEFAULT 'system'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_current_status fact_status;
    v_fact_type fact_type;
    v_rxcui VARCHAR(20);
    v_result JSONB;
BEGIN
    -- Lock the fact row to prevent concurrent activation
    SELECT status, fact_type, rxcui
    INTO v_current_status, v_fact_type, v_rxcui
    FROM clinical_facts
    WHERE fact_id = p_fact_id
    FOR UPDATE;

    -- Validate fact exists
    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'FACT_NOT_FOUND',
            'message', format('Fact %s does not exist', p_fact_id)
        );
    END IF;

    -- Validate state transition
    IF v_current_status NOT IN ('DRAFT', 'APPROVED') THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'INVALID_STATE_TRANSITION',
            'message', format('Cannot activate fact with status %s', v_current_status),
            'current_status', v_current_status::TEXT
        );
    END IF;

    -- Supersede any existing ACTIVE facts for same drug+type
    UPDATE clinical_facts
    SET status = 'SUPERSEDED',
        superseded_by = p_fact_id,
        effective_to = NOW(),
        updated_at = NOW()
    WHERE rxcui = v_rxcui
      AND fact_type = v_fact_type
      AND status = 'ACTIVE'
      AND fact_id != p_fact_id;

    -- Activate the new fact
    UPDATE clinical_facts
    SET status = 'ACTIVE',
        effective_from = NOW(),
        validated_by = p_activated_by,
        validated_at = NOW(),
        updated_at = NOW()
    WHERE fact_id = p_fact_id;

    -- Log the activation in audit trail
    INSERT INTO audit.fact_audit_log (
        fact_id,
        operation,
        old_values,
        new_values,
        changed_by
    ) VALUES (
        p_fact_id,
        'ACTIVATE',
        jsonb_build_object('status', v_current_status::TEXT),
        jsonb_build_object('status', 'ACTIVE', 'activated_by', p_activated_by),
        p_activated_by
    );

    -- Return success with activation details
    RETURN jsonb_build_object(
        'success', true,
        'fact_id', p_fact_id,
        'fact_type', v_fact_type::TEXT,
        'rxcui', v_rxcui,
        'previous_status', v_current_status::TEXT,
        'new_status', 'ACTIVE',
        'activated_by', p_activated_by,
        'activated_at', NOW()
    );
END;
$$;

COMMENT ON FUNCTION activate_fact IS
'Atomically activates a clinical fact, superseding any existing ACTIVE facts for the same drug+type.
This ensures projection views are always consistent with the canonical fact store.';

-- Batch activation function for bulk operations
CREATE OR REPLACE FUNCTION activate_facts_batch(
    p_fact_ids UUID[],
    p_activated_by VARCHAR(255) DEFAULT 'system'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_fact_id UUID;
    v_result JSONB;
    v_results JSONB[] := '{}';
    v_success_count INT := 0;
    v_failure_count INT := 0;
BEGIN
    -- Process each fact in the batch
    FOREACH v_fact_id IN ARRAY p_fact_ids
    LOOP
        v_result := activate_fact(v_fact_id, p_activated_by);
        v_results := array_append(v_results, v_result);

        IF (v_result->>'success')::boolean THEN
            v_success_count := v_success_count + 1;
        ELSE
            v_failure_count := v_failure_count + 1;
        END IF;
    END LOOP;

    RETURN jsonb_build_object(
        'success', v_failure_count = 0,
        'total', array_length(p_fact_ids, 1),
        'succeeded', v_success_count,
        'failed', v_failure_count,
        'results', to_jsonb(v_results),
        'activated_by', p_activated_by,
        'completed_at', NOW()
    );
END;
$$;

COMMENT ON FUNCTION activate_facts_batch IS
'Batch activation of clinical facts. All-or-nothing semantics within each fact.';

-- Deprecation function for fact withdrawal
CREATE OR REPLACE FUNCTION deprecate_fact(
    p_fact_id UUID,
    p_reason TEXT,
    p_deprecated_by VARCHAR(255) DEFAULT 'system'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_current_status fact_status;
BEGIN
    -- Lock and get current status
    SELECT status INTO v_current_status
    FROM clinical_facts
    WHERE fact_id = p_fact_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'FACT_NOT_FOUND'
        );
    END IF;

    -- Can deprecate from any state except already DEPRECATED
    IF v_current_status = 'DEPRECATED' THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'ALREADY_DEPRECATED'
        );
    END IF;

    -- Deprecate the fact
    UPDATE clinical_facts
    SET status = 'DEPRECATED',
        effective_to = NOW(),
        updated_at = NOW()
    WHERE fact_id = p_fact_id;

    -- Log the deprecation
    INSERT INTO audit.fact_audit_log (
        fact_id,
        operation,
        old_values,
        new_values,
        changed_by
    ) VALUES (
        p_fact_id,
        'DEPRECATE',
        jsonb_build_object('status', v_current_status::TEXT),
        jsonb_build_object('status', 'DEPRECATED', 'reason', p_reason),
        p_deprecated_by
    );

    RETURN jsonb_build_object(
        'success', true,
        'fact_id', p_fact_id,
        'reason', p_reason,
        'deprecated_by', p_deprecated_by,
        'deprecated_at', NOW()
    );
END;
$$;

COMMENT ON FUNCTION deprecate_fact IS
'Safely deprecates a clinical fact with audit trail. Use for withdrawn guidelines, obsolete data.';

-- =============================================================================
-- GAP 4: ENHANCED SCHEMA VERSION TRACKING
-- Add deployment context to schema versions for audit trail
-- =============================================================================

-- Enhance schema_migrations with deployment context
ALTER TABLE schema_migrations
ADD COLUMN IF NOT EXISTS deployed_by VARCHAR(255) DEFAULT 'system',
ADD COLUMN IF NOT EXISTS deployment_env VARCHAR(50) DEFAULT 'development',
ADD COLUMN IF NOT EXISTS git_commit_sha VARCHAR(40),
ADD COLUMN IF NOT EXISTS notes TEXT;

-- Create a function to record schema version with context
CREATE OR REPLACE FUNCTION record_schema_version(
    p_version INTEGER,
    p_name VARCHAR(255),
    p_deployed_by VARCHAR(255) DEFAULT 'system',
    p_deployment_env VARCHAR(50) DEFAULT 'development',
    p_git_commit_sha VARCHAR(40) DEFAULT NULL,
    p_notes TEXT DEFAULT NULL
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO schema_migrations (version, name, deployed_by, deployment_env, git_commit_sha, notes)
    VALUES (p_version, p_name, p_deployed_by, p_deployment_env, p_git_commit_sha, p_notes)
    ON CONFLICT (version) DO UPDATE SET
        deployed_by = EXCLUDED.deployed_by,
        deployment_env = EXCLUDED.deployment_env,
        git_commit_sha = EXCLUDED.git_commit_sha,
        notes = EXCLUDED.notes,
        applied_at = NOW();
END;
$$;

-- Create view for schema audit
CREATE OR REPLACE VIEW schema_audit AS
SELECT
    version,
    name,
    applied_at,
    deployed_by,
    deployment_env,
    git_commit_sha,
    notes,
    LAG(applied_at) OVER (ORDER BY version) as previous_version_at,
    applied_at - LAG(applied_at) OVER (ORDER BY version) as time_since_previous
FROM schema_migrations
ORDER BY version;

COMMENT ON VIEW schema_audit IS
'Audit view of schema migrations with deployment context and timing analysis.';

-- =============================================================================
-- GOVERNANCE: FACT LINEAGE TRACKING
-- Track which decisions were made with which schema/fact versions
-- =============================================================================

CREATE TABLE IF NOT EXISTS governance.decision_lineage (
    lineage_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_timestamp  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- What schema version produced this decision?
    schema_version      INTEGER NOT NULL,

    -- What facts were consulted?
    consulted_fact_ids  UUID[] NOT NULL,

    -- Decision context
    decision_type       VARCHAR(50) NOT NULL,  -- DDI_CHECK, RENAL_DOSE, FORMULARY_LOOKUP
    input_parameters    JSONB,
    output_result       JSONB,

    -- Correlation
    request_id          UUID,
    patient_context_id  VARCHAR(100),

    -- Retention policy
    expires_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '7 years'
);

CREATE INDEX idx_lineage_timestamp ON governance.decision_lineage(decision_timestamp DESC);
CREATE INDEX idx_lineage_facts ON governance.decision_lineage USING gin(consulted_fact_ids);
CREATE INDEX idx_lineage_request ON governance.decision_lineage(request_id);

COMMENT ON TABLE governance.decision_lineage IS
'Tracks which schema version and facts were used for each clinical decision.
Critical for audit: "Which schema version produced this decision?"';

-- Create governance schema if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_namespace WHERE nspname = 'governance') THEN
        CREATE SCHEMA governance;
    END IF;
END
$$;

-- Move table to governance schema if needed
-- Note: Table is created in governance schema above

-- =============================================================================
-- Record this migration
-- =============================================================================

SELECT record_schema_version(
    3,
    '003_hardening_guardrails'::VARCHAR(255),
    current_user::VARCHAR(255),
    'development'::VARCHAR(50),
    NULL::VARCHAR(40),
    'Production hardening: write guards, atomic activation, enhanced versioning'::TEXT
);

COMMIT;

-- =============================================================================
-- VERIFICATION
-- =============================================================================

SELECT 'Migration 003: Hardening Guardrails - COMPLETE' AS status;

-- Verify functions exist
SELECT routine_name, routine_type
FROM information_schema.routines
WHERE routine_schema = 'public'
AND routine_name IN ('activate_fact', 'activate_facts_batch', 'deprecate_fact', 'record_schema_version');

-- Verify rules exist
SELECT tablename, rulename
FROM pg_rules
WHERE schemaname = 'public'
AND rulename LIKE 'kb%_readonly%';
