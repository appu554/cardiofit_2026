-- 007_role_scope_columns.sql
--
-- Tier 1 schema migration aligning KB-4 explicit-criteria with v2 Revision
-- Mapping requirements (8-12 role authority model + jurisdiction-aware
-- ScopeRules). Adds columns the future Authorisation evaluator (state
-- machine #1 in the v2 substrate) will read at runtime to gate rule
-- firing by who-may-act-now.
--
-- Reference: claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md
-- §"v2 Revision Mapping deltas"
--
-- Backward compatible — existing rules get sensible defaults so no rule
-- firing changes until the substrate is wired up.

-- Roles permitted to act on this rule (e.g. {GP, NP, PHARMACIST, ACOP, RN_PRESCRIBER}).
-- NULL or empty means "no role gating" (legacy behaviour).
ALTER TABLE kb4_explicit_criteria
    ADD COLUMN IF NOT EXISTS applicable_roles TEXT[];

-- Effective dates for rules with regulatory windows (VIC PCW exclusion,
-- TAS pilot, designated RN endorsement, ACOP APC training, etc.)
-- NULL = "always in force".
ALTER TABLE kb4_explicit_criteria
    ADD COLUMN IF NOT EXISTS effective_from DATE;
ALTER TABLE kb4_explicit_criteria
    ADD COLUMN IF NOT EXISTS effective_to DATE;

-- Free-form scope constraints (e.g. {care_setting: 'aged_care', frailty_stage_min: 4}).
-- Substrate Authorisation evaluator queries this at runtime.
ALTER TABLE kb4_explicit_criteria
    ADD COLUMN IF NOT EXISTS scope_constraints JSONB;

-- Indexes for typical filter shapes
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_applicable_roles
    ON kb4_explicit_criteria USING gin (applicable_roles)
    WHERE applicable_roles IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_kb4_explicit_effective_window
    ON kb4_explicit_criteria (effective_from, effective_to);

CREATE INDEX IF NOT EXISTS idx_kb4_explicit_scope_constraints
    ON kb4_explicit_criteria USING gin (scope_constraints)
    WHERE scope_constraints IS NOT NULL;

-- Document the columns so future readers know what they're for
COMMENT ON COLUMN kb4_explicit_criteria.applicable_roles IS
    'v2: roles permitted to act on this rule. NULL=no gating. '
    'Values: GP, NP, PHARMACIST, ACOP, RN_PRESCRIBER, RN, EN, PCW.';
COMMENT ON COLUMN kb4_explicit_criteria.effective_from IS
    'v2: rule effective start date (regulatory windows). NULL=always in force.';
COMMENT ON COLUMN kb4_explicit_criteria.effective_to IS
    'v2: rule effective end date (regulatory sunset). NULL=indefinite.';
COMMENT ON COLUMN kb4_explicit_criteria.scope_constraints IS
    'v2: free-form scope constraints (care_setting, frailty_stage_min, '
    'jurisdiction_extra) for runtime Authorisation evaluator.';
