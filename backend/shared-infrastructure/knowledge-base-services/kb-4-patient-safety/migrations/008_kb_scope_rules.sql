-- 008_kb_scope_rules.sql
--
-- Tier 1 — kb_scope_rules table seeded with the 5 known regulatory windows
-- v2 Revision Mapping introduced. The future Authorisation evaluator queries
-- this table at runtime to answer "for this resident, this medicine, this
-- moment, who is authorised to do what?"
--
-- Lives in KB-4 (kb4_patient_safety) for proximity to the explicit-criteria
-- it gates. Future migration may rehome it to KB-2 or a new authorisation KB.
--
-- Reference: claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md
-- §"v2 jurisdictional regulatory windows"

CREATE TABLE IF NOT EXISTS kb_scope_rules (
    id                   BIGSERIAL PRIMARY KEY,
    rule_id              TEXT NOT NULL UNIQUE,
    rule_name            TEXT NOT NULL,
    jurisdiction         TEXT NOT NULL,            -- 'AU', 'AU/VIC', 'AU/TAS', etc.
    affected_role        TEXT NOT NULL,            -- 'PCW', 'RN_PRESCRIBER', 'ACOP', etc.
    affected_actions     TEXT[],                   -- ['administer', 'prescribe', ...]
    affected_medicines   TEXT[],                   -- e.g. ['S4','S8'] schedules
    constraint_type      TEXT NOT NULL,            -- 'EXCLUSION' | 'GATING' | 'PILOT' | 'EVIDENCE'
    effective_from       DATE NOT NULL,
    effective_to         DATE,                     -- NULL = indefinite
    grace_period_until   DATE,                     -- statutory grace (e.g. VIC PCW 90-day)
    summary              TEXT NOT NULL,
    statutory_reference  TEXT,
    source_url           TEXT,
    notes                TEXT,
    loaded_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kb_scope_rules_jurisdiction
    ON kb_scope_rules (jurisdiction);
CREATE INDEX IF NOT EXISTS idx_kb_scope_rules_affected_role
    ON kb_scope_rules (affected_role);
CREATE INDEX IF NOT EXISTS idx_kb_scope_rules_effective_window
    ON kb_scope_rules (effective_from, effective_to);

-- Seed: 5 regulatory windows from v2 Revision Mapping document
INSERT INTO kb_scope_rules (
    rule_id, rule_name, jurisdiction, affected_role, affected_actions,
    affected_medicines, constraint_type, effective_from, effective_to,
    grace_period_until, summary, statutory_reference, source_url
) VALUES
(
    'AU-VIC-PCW-S4S8-2026',
    'Victorian PCW S4/S8 administration exclusion',
    'AU/VIC',
    'PCW',
    ARRAY['administer'],
    ARRAY['S4', 'S8', 'drugs_of_dependence'],
    'EXCLUSION',
    '2026-07-01',
    NULL,
    '2026-09-29',
    'PCWs cannot administer S4/S8 + drugs of dependence to non-self-'
    'administering residents in Victorian RACHs. RN/EN/pharmacist/medical '
    'practitioner only. 90-day grace period to 29 Sept 2026.',
    'Drugs, Poisons and Controlled Substances Amendment Act 2025 (Vic)',
    NULL
),
(
    'AU-NMBA-RN-PRESCRIBER-2025',
    'NMBA Designated Registered Nurse Prescriber endorsement',
    'AU',
    'RN_PRESCRIBER',
    ARRAY['prescribe'],
    NULL,
    'GATING',
    '2025-09-30',
    NULL,
    NULL,
    'NMBA endorsement standard live since 30 Sept 2025; first endorsed '
    'prescribers expected mid-2026. Partnership-only with prescribing '
    'agreement and 6-month mentorship. Platform must verify prescribing '
    'agreement scope + mentorship status before allowing prescribe action.',
    'NMBA Standard for Designated Registered Nurse Prescribers (2025)',
    NULL
),
(
    'AU-DOH-ACOP-APC-2026',
    'ACOP mandatory APC training requirement',
    'AU',
    'ACOP',
    ARRAY['act_as_acop'],
    NULL,
    'GATING',
    '2026-07-01',
    NULL,
    NULL,
    'All ACOP-credentialed pharmacists must complete APC training from '
    '1 July 2026. Platform must verify APC credential currency before '
    'allowing ACOP-tier actions.',
    'Department of Health, Disability and Ageing — ACOP Tier 1/2 Rules (2026)',
    NULL
),
(
    'AU-TAS-PHARM-COPRESCRIBE-2026',
    'Tasmanian pharmacist co-prescribing pilot',
    'AU/TAS',
    'PHARMACIST',
    ARRAY['co_prescribe'],
    NULL,
    'PILOT',
    '2026-01-01',
    '2027-12-31',
    NULL,
    'Tasmanian pilot: pharmacists may co-prescribe in collaboration with '
    'GP per individual treatment plan. Australian-first pilot, $5M state '
    'budget, 12-month trial 2026-2027. Platform should be ready as digital '
    'substrate.',
    'Tasmanian State Budget 2025-26',
    NULL
),
(
    'AU-ACQSC-STD5-2026',
    'Strengthened Aged Care Quality Standard 5 (Clinical Care)',
    'AU',
    'ALL',
    ARRAY['document_evidence'],
    NULL,
    'EVIDENCE',
    '2026-01-01',
    NULL,
    NULL,
    'Standard 5 (Clinical Care) requires audit-grade evidence for every '
    'clinical-care decision. Layer-1 rules must produce evidence bundles '
    'that satisfy Standard 5.X requirements.',
    'Aged Care Quality and Safety Commission — Strengthened Quality Standards',
    NULL
)
ON CONFLICT (rule_id) DO NOTHING;
