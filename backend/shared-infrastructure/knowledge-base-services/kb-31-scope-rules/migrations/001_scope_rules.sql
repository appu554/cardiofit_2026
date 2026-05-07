-- kb-31 migration 001: ScopeRule store
--
-- Stores versioned, jurisdiction-aware ScopeRules per Layer 3 v2 doc Part
-- 5.5.2. Mirrors kb-30 migration 001 (authorisation_rules) with a category
-- column and a status column (ACTIVE | DRAFT) so pilot rules can be staged
-- without entering the runtime authorisation path.
--
-- Versioning model: (rule_id, version) is unique. supersedes_ref is set
-- when a new version replaces an older one (lineage walk).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS scope_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id             TEXT NOT NULL,
    version             INTEGER NOT NULL,
    jurisdiction        TEXT NOT NULL,
    category            TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'DRAFT')),
    effective_start     TIMESTAMPTZ NOT NULL,
    effective_end       TIMESTAMPTZ,
    grace_days          INTEGER,
    payload_yaml        TEXT NOT NULL,
    payload_json        JSONB NOT NULL,
    content_sha         TEXT NOT NULL,
    supersedes_ref      UUID REFERENCES scope_rules(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (rule_id, version)
);

CREATE INDEX IF NOT EXISTS idx_scope_rules_jurisdiction_effective
    ON scope_rules(jurisdiction, effective_start, effective_end);

CREATE INDEX IF NOT EXISTS idx_scope_rules_category
    ON scope_rules(category);

CREATE INDEX IF NOT EXISTS idx_scope_rules_active
    ON scope_rules(jurisdiction)
    WHERE status = 'ACTIVE'
      AND (effective_end IS NULL OR effective_end > now());

CREATE INDEX IF NOT EXISTS idx_scope_rules_rule_id_version
    ON scope_rules(rule_id, version DESC);
