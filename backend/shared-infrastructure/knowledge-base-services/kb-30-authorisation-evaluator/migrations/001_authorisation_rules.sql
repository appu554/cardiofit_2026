-- kb-30 migration 001: AuthorisationRule store
--
-- Stores versioned, jurisdiction-aware Authorisation rules consumed by the
-- runtime evaluator (Layer 3 v2 doc Part 4.5.2).
--
-- Versioning model: (rule_id, version) is unique. supersedes_ref points at
-- the prior row when a new version replaces an older one (lineage walk).
-- Both the original YAML and the parsed JSON are kept; payload_yaml is
-- regulator-defensible source-of-truth, payload_json supports indexed query.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS authorisation_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id             TEXT NOT NULL,
    version             INTEGER NOT NULL,
    jurisdiction        TEXT NOT NULL,
    effective_start     TIMESTAMPTZ NOT NULL,
    effective_end       TIMESTAMPTZ,
    grace_days          INTEGER,
    payload_yaml        TEXT NOT NULL,
    payload_json        JSONB NOT NULL,
    content_sha         TEXT NOT NULL,
    supersedes_ref      UUID REFERENCES authorisation_rules(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by_role_ref UUID,
    UNIQUE (rule_id, version)
);

CREATE INDEX IF NOT EXISTS idx_auth_rules_jurisdiction_effective
    ON authorisation_rules(jurisdiction, effective_start, effective_end);

-- Active rules with no explicit end-date. Time-bounded "active" filtering
-- is handled at query time by idx_auth_rules_jurisdiction_effective; this
-- partial index narrows specifically to open-ended rules. (Plan 0.4 Task 3
-- removed `OR effective_end > now()` because now() is not IMMUTABLE and
-- cannot appear in a partial index predicate.)
CREATE INDEX IF NOT EXISTS idx_auth_rules_active
    ON authorisation_rules(jurisdiction)
    WHERE effective_end IS NULL;

CREATE INDEX IF NOT EXISTS idx_auth_rules_rule_id_version
    ON authorisation_rules(rule_id, version DESC);
