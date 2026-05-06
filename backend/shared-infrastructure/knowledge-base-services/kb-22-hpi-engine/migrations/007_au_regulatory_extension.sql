-- ============================================================================
-- Migration 007: Australian Aged Care Regulatory Extension (Phase 1C-β + 1C-γ)
--
-- Implements:
--   1C-β: Extends clinical_sources schema with Layer 1 v2 §1.2 fields and
--         seeds 8 Australian regulatory source rows (Category C).
--   1C-γ: Creates regulatory_scope_rules table for jurisdiction-aware
--         authorisation rules, seeded with Victorian DPCS Amendment Act 2025
--         §36EA rules (PCW exclusion + role-by-schedule authorisation matrix).
--
-- All ScopeRule rows ship with activation_status='draft' and
-- requires_legal_review=TRUE. Promotion to 'active' requires explicit human
-- review by a clinical informatics lead with statutory text in hand.
--
-- Source spec: kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md
-- Source design: docs/superpowers/specs/2026-05-04-layer1c-procurement-design.md
-- Source plan: docs/superpowers/plans/2026-05-04-layer1c-procurement-plan.md
-- Date authored: 2026-05-04
-- ============================================================================

BEGIN;

-- ============================================================================
-- Section 1 — Extend clinical_sources for Australian Category C sources
-- ============================================================================

-- 1.1 Allow 'Australia' as a region
ALTER TABLE clinical_sources DROP CONSTRAINT IF EXISTS clinical_sources_region_check;
ALTER TABLE clinical_sources ADD CONSTRAINT clinical_sources_region_check
    CHECK (region IN ('global','UK','India','US','Europe','Asia-Pacific','Australia'));

-- 1.2 Extend type enum for legislation, regulations, frameworks
ALTER TABLE clinical_sources DROP CONSTRAINT IF EXISTS clinical_sources_type_check;
ALTER TABLE clinical_sources ADD CONSTRAINT clinical_sources_type_check
    CHECK (type IN (
        'guideline','book','journal','consensus','position_statement',
        'formulary','internal_kb',
        'legislation','regulation','regulatory_standard',
        'clinical_care_standard','framework'
    ));

-- 1.3 Add Layer 1 v2 §1.2 columns (all nullable for backwards compatibility)
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS regulatory_category CHAR(1)
    CHECK (regulatory_category IS NULL OR regulatory_category IN ('A','B','C'));
COMMENT ON COLUMN clinical_sources.regulatory_category IS
    'Layer 1 v2 §1.1 source category. A=clinical knowledge, B=patient state, C=regulatory/authority. NULL for pre-v2 canon sources.';

ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS jurisdiction TEXT;
COMMENT ON COLUMN clinical_sources.jurisdiction IS
    'Jurisdictional scope: ''national'', ISO-3166-2 subdivision (e.g., ''AU-VIC'', ''AU-TAS''), ''facility'', or NULL.';

ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS authority_tier SMALLINT
    CHECK (authority_tier IS NULL OR (authority_tier BETWEEN 1 AND 4));
COMMENT ON COLUMN clinical_sources.authority_tier IS
    'Layer 1 v2 §1.2: 1=primary regulator/legislature, 2=peak professional body, 3=academic/research, 4=facility-level policy.';

ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS effective_start DATE;
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS effective_end DATE;
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS replaces_source_id UUID
    REFERENCES clinical_sources(source_id);
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS replaced_by_source_id UUID
    REFERENCES clinical_sources(source_id);
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS reproduction_terms TEXT;
ALTER TABLE clinical_sources ADD COLUMN IF NOT EXISTS procurement_path TEXT;
COMMENT ON COLUMN clinical_sources.procurement_path IS
    'Filesystem path within knowledge-base-services to the procured PDF corpus for this source.';

-- ============================================================================
-- Section 2 — Seed Layer 1C Australian regulatory source rows
-- ============================================================================

INSERT INTO clinical_sources (
    code, name, type, source_category,
    edition, publication_year, publisher, region, license_type, runtime_allowed, update_cycle,
    regulatory_category, jurisdiction, authority_tier,
    effective_start, reproduction_terms, procurement_path, notes
) VALUES
('AU_AGED_CARE_ACT_2024',
 'Aged Care Act 2024 + Aged Care Rules 2025 + Strengthened Quality Standards',
 'legislation', 'canon',
 'commenced 2025-11-01', 2024, 'Commonwealth of Australia',
 'Australia', 'public', FALSE, 'iterative',
 'C', 'national', 1,
 DATE '2025-11-01',
 'Crown copyright Commonwealth. CC BY 4.0 per Federal Register of Legislation policy.',
 'kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/',
 'Layer 1 v2 §4.1. Aged Care Rules 2025 are 900+ pages and iterating. Procured PDFs include Act as-made + 2025-11-01 compilation, Rules + Explanatory Statement, and Strengthened Quality Standards Aug 2025.'),

('AU_VIC_DPCS_AMEND_2025',
 'Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025',
 'legislation', 'canon',
 'No. 37 of 2025', 2025, 'State of Victoria',
 'Australia', 'public', FALSE, 'one-time amendment Act',
 'C', 'AU-VIC', 1,
 DATE '2026-07-01',
 'Crown copyright Victoria. Open access on legislation.vic.gov.au. Commercial reproduction permitted with attribution.',
 'kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/',
 'Layer 1 v2 §4.4. Assented 2025-09-16. Inserts §36EA into DPCS Act 1981. Self-repeals 2027-07-01 per §12 (amendments persist). Reg 149Q exposure draft Feb 2026; final regulations expected before 2026-07-01 commencement. Spec notes 90-day grace period to 2026-09-29 before enforcement.'),

('AU_NMBA_DESIGNATED_RN_2025',
 'NMBA Registration Standard — Endorsement for Scheduled Medicines (Designated Registered Nurse Prescriber)',
 'regulatory_standard', 'canon',
 'effective 2025-09-30', 2025, 'Nursing and Midwifery Board of Australia (AHPRA)',
 'Australia', 'public', FALSE, 'periodic_review',
 'C', 'national', 2,
 DATE '2025-09-30',
 '© NMBA / AHPRA. Reference and quotation permitted with attribution. Commercial reproduction may require permission.',
 'kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber/',
 'Layer 1 v2 §4.5. First endorsed prescribers expected mid-2026. Eligibility: 5,000h clinical experience past 6y + NMBA-approved postgrad + 6-month mentorship.'),

('AU_TAS_COPRESCRIBING_PILOT',
 'Tasmanian Aged Care Pharmacist Co-Prescribing Pilot',
 'framework', 'canon',
 'pilot 2026-2027', 2026, 'State of Tasmania (Department of Health) + UTas School of Pharmacy',
 'Australia', 'internal', FALSE, 'pilot',
 'C', 'AU-TAS', 1,
 NULL,  -- not yet effective; engagement-required
 'TBD — depends on partnership terms.',
 'kb-3-guidelines/knowledge/au/regulatory/states/tas/co_prescribing_pilot_2026/',
 'Layer 1 v2 §4.6. ENGAGEMENT-REQUIRED. Australian-first pilot, $5M state budget. Engagement contacts: Salahudeen, Peterson, Curtain (UTas); Duncan McKenzie (TAS DH). v2 Revision Mapping Part 7 Move 1.'),

('AU_APC_ACOP_TRAINING',
 'APC Accreditation Standards for ACOP (Aged Care On-site Pharmacist) Training Programs',
 'regulatory_standard', 'canon',
 'mandatory from 2026-07-01', 2023, 'Australian Pharmacy Council',
 'Australia', 'public', FALSE, 'periodic_review',
 'C', 'national', 2,
 DATE '2026-07-01',
 '© APC. Reference and quotation permitted with attribution. Commercial reproduction may require permission.',
 'kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training/',
 'Layer 1 v2 §4.7. APC-accredited ACOP training mandatory for $350M ACOP measure participation from 2026-07-01.'),

('AU_PHARMA_CARE_FRAMEWORK',
 'PHARMA-Care National Quality Framework',
 'framework', 'canon',
 'pilot v0.1', 2025, 'University of South Australia (Sluggett, Javanparast)',
 'Australia', 'internal', FALSE, 'pilot',
 'C', 'national', 3,
 NULL,  -- not yet effective; engagement-required
 'TBD — depends on EOI engagement terms.',
 'kb-3-guidelines/knowledge/au/regulatory/frameworks/pharma_care_unisa/',
 'Layer 1 v2 §4.8. ENGAGEMENT-REQUIRED. EOI ALH-PHARMA-Care@unisa.edu.au. v2 Revision Mapping Part 7 Move 2. KB-13 already has 5 placeholder indicator rows pending framework finalisation.'),

('AU_RESTRICTIVE_PRACTICE_2022',
 'Aged Care Royal Commission Response Act 2022 + Quality of Care Principles 2014 (restrictive practice provisions) + ACQSC operational guidance',
 'legislation', 'canon',
 'amended 2022; QoC repealed 2025-11-01', 2022, 'Commonwealth of Australia + ACQSC',
 'Australia', 'public', FALSE, 'iterative',
 'C', 'national', 1,
 DATE '2019-07-01',  -- restrictive practice framework in force since 2019
 'Crown copyright Commonwealth. CC BY 4.0.',
 'kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice/',
 'Layer 1 v2 §4.3. QoC Principles 2014 repealed 2025-11-01 by Aged Care Rules 2025; transition-period compilation captured. Successor restrictive-practice provisions in Aged Care Rules 2025 (see AU_AGED_CARE_ACT_2024).'),

('AU_MHR_SHARING_DEFAULT_2025',
 'Health Legislation Amendment (Modernising My Health Record—Sharing by Default) Act 2025 + Share-by-Default Rules 2025',
 'legislation', 'canon',
 'C2025A00008', 2025, 'Commonwealth of Australia',
 'Australia', 'public', FALSE, 'iterative',
 'C', 'national', 1,
 DATE '2025-02-14',  -- Royal Assent
 'Crown copyright Commonwealth. CC BY 4.0.',
 'kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025/',
 'Layer 1 v2 §4.10. Royal Assent 2025-02-14. Mandatory pathology + diagnostic imaging upload to MHR commences 2026-07-01. Civil penalties: 250 PU non-registration; 30 PU non-compliant upload. Subordinate Rules F2025L01569 registered 2025-12-09.')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- Section 3 — Create regulatory_scope_rules table
-- ============================================================================
-- This table holds jurisdiction-aware authorisation rules derived from Category C
-- sources. The Authorisation state machine (Phase 1C-δ) queries this table to
-- evaluate "may this person, in this jurisdiction, with these credentials,
-- perform this action with this medication, on this resident, in this location?"
--
-- Schema is data-not-code: NSW/QLD/SA following Victoria's lead becomes adding
-- rows, not engineering work — exactly the data-not-code architecture required
-- by Layer 1 v2 §4.4 ("ScopeRules architecture must be data-not-code").
-- ============================================================================

CREATE TABLE IF NOT EXISTS regulatory_scope_rules (
    rule_id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_code            TEXT NOT NULL UNIQUE,
    source_id            UUID NOT NULL REFERENCES clinical_sources(source_id),
    section_ref          TEXT NOT NULL,
    -- Subject of the rule
    jurisdiction         TEXT NOT NULL,
    role                 TEXT NOT NULL,
    role_qualifications  JSONB,  -- e.g. {"notation": false, "endorsement": "designated_rn_prescriber", "atsihp": true}
    -- Object of the rule
    medication_schedules TEXT[] NOT NULL,
    action               TEXT NOT NULL CHECK (action IN (
                            'administer','assist_self_administer','prescribe',
                            'supply','manage_administration')),
    -- Context
    resident_state       TEXT CHECK (resident_state IS NULL OR resident_state IN (
                            'self_administering','non_self_administering','any')),
    location             TEXT CHECK (location IS NULL OR location IN (
                            'on_site','off_site','any')),
    applicability_conditions JSONB,  -- additional structured preconditions
    -- Outcome
    permitted            BOOLEAN NOT NULL,
    rationale            TEXT NOT NULL,
    -- Temporal scope
    effective_from       DATE NOT NULL,
    grace_until          DATE,
    enforcement_from     DATE,
    effective_to         DATE,
    -- Governance
    activation_status    TEXT NOT NULL DEFAULT 'draft' CHECK (activation_status IN (
                            'draft','review','active','suspended','superseded')),
    requires_legal_review BOOLEAN NOT NULL DEFAULT TRUE,
    legal_review_notes   TEXT,
    reviewed_by          TEXT,
    reviewed_at          TIMESTAMPTZ,
    -- Lineage
    supersedes_rule_id   UUID REFERENCES regulatory_scope_rules(rule_id),
    superseded_by_rule_id UUID REFERENCES regulatory_scope_rules(rule_id),
    -- Audit
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scope_rules_jurisdiction ON regulatory_scope_rules(jurisdiction);
CREATE INDEX IF NOT EXISTS idx_scope_rules_role         ON regulatory_scope_rules(role);
CREATE INDEX IF NOT EXISTS idx_scope_rules_source       ON regulatory_scope_rules(source_id);
CREATE INDEX IF NOT EXISTS idx_scope_rules_active       ON regulatory_scope_rules(activation_status)
    WHERE activation_status = 'active';

COMMENT ON TABLE regulatory_scope_rules IS
'Jurisdiction-aware authorisation rules derived from regulatory sources. Consumed by the Authorisation state machine (Phase 1C-δ). Layer 1 v2 §4.4 mandates data-not-code architecture: adding NSW/QLD/SA jurisdictions is row insertion, not engineering. All rules ship in draft status with requires_legal_review=TRUE; promotion to active requires explicit human review against the cited statutory text.';

COMMENT ON COLUMN regulatory_scope_rules.role_qualifications IS
'JSONB structured qualifications layered on the role. Examples:
  EN with notation: {"notation": true}  (cannot administer)
  EN without notation + NMBA medication qualification: {"notation": false, "nmba_medication_qual": true}
  Designated RN Prescriber: {"endorsement": "designated_rn_prescriber"}
  ATSIHP: {"atsihp": true}';

COMMENT ON COLUMN regulatory_scope_rules.applicability_conditions IS
'JSONB additional preconditions beyond the primary fields. Examples:
  Prescribed supply: {"supply": "prescribed"}
  VAD permit holder: {"vad_permit": true}
  Reg 149Q exemption: {"unforeseen_event": true, "rn_clinical_judgement": true}';

-- ============================================================================
-- Section 4 — Seed Victorian DPCS Amendment Act 2025 §36EA ScopeRules
-- ============================================================================
-- The §36EA rule set covers eleven distinct (role × schedule × scenario) tuples
-- needed to evaluate "may X administer Y to a Victorian aged care resident?"
-- All rules ship draft + requires_legal_review until clinical informatics +
-- legal review confirm accuracy against the as-made Act and final Reg 149Q.
-- ============================================================================

DO $$
DECLARE
    vic_source_id UUID;
BEGIN
    SELECT source_id INTO vic_source_id
        FROM clinical_sources WHERE code = 'AU_VIC_DPCS_AMEND_2025';

    IF vic_source_id IS NULL THEN
        RAISE EXCEPTION 'AU_VIC_DPCS_AMEND_2025 source row not found — Section 2 seed must succeed first';
    END IF;

    -- Rule 1: PCW administering S4/S8/S9/DoD to non-self-administering resident on-site → PROHIBITED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from, grace_until,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_PCW_S4S8S9DOD_ADMIN_NONSELF',
        vic_source_id, 'DPCS Act 1981 §36EA(1) (as inserted by Act No. 37/2025 §9)',
        'AU-VIC', 'PCW', NULL,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'non_self_administering', 'on_site',
        '{"supply": "prescribed", "facility": "residential_aged_care_home", "funded": true}'::jsonb,
        FALSE,
        '§36EA(1) requires the registered provider to ensure that ONLY a person specified in §36EA(2) administers any drug of dependence, S9, S8 or S4 poison to a person who (a) is accessing funded aged care services in the residential aged care home, (b) for whom that drug or poison has been supplied on prescription, and (c) is at the residential aged care home when the drug or poison is administered. PCWs are not specified in §36EA(2). Violation: 100 penalty units (provider).',
        DATE '2026-07-01', DATE '2026-09-29',
        'draft', TRUE,
        'PRIORITY rule for Authorisation state machine. Verify exact text of as-made Act §36EA(1) prior to activation. Final regulation 149Q (currently exposure draft Feb 2026) may further qualify the permitted exemptions.'
    );

    -- Rule 2: PCW assisting self-administration → PERMITTED (carve-out)
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_PCW_ANY_ASSIST_SELF',
        vic_source_id, 'DPCS Act 1981 §36EA(3)(b) + DH Vic Reg 149Q draft guidance',
        'AU-VIC', 'PCW', NULL,
        ARRAY['S2','S3','S4','S8','S9','DoD']::TEXT[], 'assist_self_administer',
        'self_administering', 'on_site',
        '{"supply": "prescribed", "self_administration": true}'::jsonb,
        TRUE,
        '§36EA(3)(b): a registered provider does not contravene §36EA(1) if "the person for whom the drug or poison has been supplied self-administers the drug or poison". Per DH Vic guidance: "personal care workers and others can continue to assist or support a person to administer their own medication (for example, by taking the screw-cap lid off a container)".',
        DATE '2026-07-01',
        'draft', TRUE,
        'Verify scope of "assist" against National guiding principles for medication management in residential aged care; ensure not interpreted as "PCW administers" where resident has cognitive impairment.'
    );

    -- Rule 3: RN (and Nurse Practitioner, Designated RN Prescriber subsets) administering S4/S8/S9/DoD → PERMITTED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_RN_S4S8S9DOD_ADMIN',
        vic_source_id, 'DPCS Act 1981 §36EA(2)(a)',
        'AU-VIC', 'RN', NULL,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'on_site',
        '{"supply": "prescribed"}'::jsonb,
        TRUE,
        '§36EA(2)(a) specifies "a registered nurse" as a permitted administrator. Per DH Vic Reg 149Q draft guidance footnote 1: "Includes registered nurses, nurse practitioners, designated registered nurse prescribers, and enrolled nurses (ENs) without notation". Class includes Nurse Practitioners and Designated RN Prescribers as subsets — see VIC_NP_*, VIC_DRNP_* rules below for explicit specialisations.',
        DATE '2026-07-01',
        'draft', TRUE,
        'NP and DRNP subsets are encoded as separate rules for query specificity even though they fall under §36EA(2)(a). Confirm interpretation with NMBA registration class taxonomy.'
    );

    -- Rule 4: EN without notation administering S4/S8/S9/DoD → PERMITTED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_EN_NO_NOTATION_S4S8S9DOD_ADMIN',
        vic_source_id, 'DPCS Act 1981 §36EA(2)(b)',
        'AU-VIC', 'EN',
        '{"notation": false, "nmba_medication_qual": true}'::jsonb,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'on_site',
        '{"supply": "prescribed"}'::jsonb,
        TRUE,
        '§36EA(2)(b) specifies "an enrolled nurse who holds a qualification approved by the Nursing and Midwifery Board of Australia in relation to the administration of medication". Per DH Vic guidance footnote 1: ENs without notation hold a Board-approved qualification in administration of medicines.',
        DATE '2026-07-01',
        'draft', TRUE,
        'Credential verification at runtime requires cross-reference to AHPRA register notation field. Phase 1C-δ Credential ledger must capture EN notation status.'
    );

    -- Rule 5: EN WITH notation administering anything → PROHIBITED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_EN_WITH_NOTATION_ANY_ADMIN',
        vic_source_id, 'DPCS Act 1981 §36EA(2)(b) + AHPRA notation policy',
        'AU-VIC', 'EN',
        '{"notation": true}'::jsonb,
        ARRAY['S2','S3','S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'any',
        NULL,
        FALSE,
        'Per DH Vic Reg 149Q draft guidance footnote 1: "ENs with notation are identified by having a notation on their registration that states ''Does not hold a Board-approved qualification in administration of medicines'' and therefore cannot administer medication via any route". This is an AHPRA-level scope restriction that operates in addition to §36EA — applies to all schedules and all locations, not only Victoria.',
        DATE '2026-07-01',
        'draft', TRUE,
        'This rule actually applies nationally per AHPRA registration; encoded under VIC for completeness but should be replicated as national rule once the scope rule engine supports cross-jurisdictional inheritance.'
    );

    -- Rule 6: Medical practitioner administering S4/S8/S9/DoD → PERMITTED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_MED_PRAC_S4S8S9DOD_ADMIN',
        vic_source_id, 'DPCS Act 1981 §36EA(2)(c) + DPCSA general practitioner authorisation',
        'AU-VIC', 'medical_practitioner', NULL,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'on_site',
        '{"supply": "prescribed", "within_scope_of_practice": true}'::jsonb,
        TRUE,
        '§36EA(2)(c): "any other registered health practitioner who is authorised by or under this Act or the regulations to administer the drug or poison". Medical practitioners have general DPCSA authorisation to administer within scope of practice.',
        DATE '2026-07-01',
        'draft', TRUE,
        'Subject to scope-of-practice limits; not a blanket authorisation.'
    );

    -- Rule 7: Pharmacist administering S4/S8/S9/DoD → PERMITTED
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_PHARMACIST_S4S8S9DOD_ADMIN',
        vic_source_id, 'DPCS Act 1981 §36EA(2)(c) + pharmacist DPCSA authorisation',
        'AU-VIC', 'pharmacist', NULL,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'on_site',
        '{"supply": "prescribed", "within_scope_of_practice": true}'::jsonb,
        TRUE,
        '§36EA(2)(c) covers registered pharmacists with existing DPCSA authorisation to administer within scope.',
        DATE '2026-07-01',
        'draft', TRUE,
        'Pharmacist autonomous prescribing scope is expanding nationally (joint AdPha/Pharmacy Guild/PSA submission Oct 2025); revisit when AHPRA/Pharmacy Board frameworks evolve.'
    );

    -- Rule 8: ATSIHP administering S2/S3/S4/S8 → PERMITTED (note: S9 NOT included)
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_ATSIHP_S2S3S4S8_ADMIN',
        vic_source_id, 'DPCS Regulations 2017 (ATSIHP authorisation) + §36EA(2)(c)',
        'AU-VIC', 'ATSIHP',
        '{"atsihp": true}'::jsonb,
        ARRAY['S2','S3','S4','S8']::TEXT[], 'administer',
        'any', 'on_site',
        '{"supply": "prescribed", "within_scope_of_practice": true}'::jsonb,
        TRUE,
        'Per DH Vic Reg 149Q draft guidance: "Regulations enable registered Aboriginal and/or Torres Strait Islander Health Practitioners (ATSIHPs) to administer Schedules 2, 3, 4 and 8 medications. This authorisation is not impacted and ATSIHPs, as registered health practitioners, can continue to administer medication within their scope of their practice." Note S9 is NOT included.',
        DATE '2026-07-01',
        'draft', TRUE,
        'Confirm ATSIHP authorisation is in DPCS Regulations 2017 (currently captured in DPCS-Regulations-2017-compilation-2025-11.pdf) and not amended by future regulations. Verify S9 exclusion is intentional vs scrivener limitation.'
    );

    -- Rule 9: §36EA does not apply OFF-SITE (resident not at the home)
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_OFFSITE_36EA_NOT_APPLICABLE',
        vic_source_id, 'DPCS Act 1981 §36EA(1)(c) (negative scope)',
        'AU-VIC', 'any',
        '{"any_role": true}'::jsonb,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'off_site',
        '{"scope_carve_out": "not_at_residential_aged_care_home"}'::jsonb,
        TRUE,
        '§36EA(1)(c) limits the obligation to medication administered to a person "who is at the residential aged care home when the drug or poison is administered". Per DH Vic guidance: "Apply when a resident is not on site at a residential aged care home. This covers situations such as when a resident may be in the community such as on an outing, a medical appointment, or with family or friends. In these circumstances, the registered nurse responsible for managing medication administration continues to delegate medication administration in line with relevant codes, standards and guidelines issued by the Nursing and Midwifery Board of Australia." Off-site authorisation reverts to baseline DPCSA + NMBA delegation framework, not the §36EA exclusion regime.',
        DATE '2026-07-01',
        'draft', TRUE,
        'Off-site is a structural carve-out from §36EA, not a positive permission. Authorisation engine must check facility-presence before applying §36EA rules. permitted=TRUE here means "§36EA does not prohibit"; baseline DPCSA rules still apply.'
    );

    -- Rule 10: VAD permit substances → §36EA does not apply
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_VAD_PERMIT_36EA_EXEMPT',
        vic_source_id, 'DPCS Act 1981 §36EA(4) (as inserted by Act No. 37/2025 §9)',
        'AU-VIC', 'any',
        '{"vad_permit_holder": true}'::jsonb,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'any', 'on_site',
        '{"voluntary_assisted_dying_permit": true, "permit_subject": true}'::jsonb,
        TRUE,
        '§36EA(4): "Subsection (1) does not apply to the administration of any voluntary assisted dying substance specified in a voluntary assisted dying permit to a person who is (a) the subject of that permit; and (b) accessing funded aged care services in a residential aged care home." VAD authorisation regime (Voluntary Assisted Dying Act 2017 (Vic)) governs in place of §36EA.',
        DATE '2026-07-01',
        'draft', TRUE,
        'permitted=TRUE here means "§36EA does not prohibit"; VAD-specific authorisation requirements still apply. Authorisation engine must defer to VAD permit verification module.'
    );

    -- Rule 11: Reg 149Q "prescribed circumstances" — unforeseen event + RN clinical delegation
    INSERT INTO regulatory_scope_rules (
        rule_code, source_id, section_ref,
        jurisdiction, role, role_qualifications,
        medication_schedules, action,
        resident_state, location,
        applicability_conditions,
        permitted, rationale,
        effective_from,
        activation_status, requires_legal_review,
        legal_review_notes
    ) VALUES (
        'VIC_REG_149Q_UNFORESEEN_EXEMPT',
        vic_source_id,
        'DPCS Act 1981 §36EA(3)(c) + draft DPCS Regulations 2026 Reg 149Q',
        'AU-VIC', 'PCW',
        '{"or_other_non_specified_person": true}'::jsonb,
        ARRAY['S4','S8','S9','DoD']::TEXT[], 'administer',
        'non_self_administering', 'on_site',
        '{"unforeseen_event": true, "rn_managing_administration_determines_no_delay_acceptable": true, "rn_delegates_administration": true, "documented_per_reg_149Q_5": true}'::jsonb,
        TRUE,
        '§36EA(3)(c): "A registered provider does not contravene subsection (1) if a person other than a person specified in subsection (2) administers the drug or poison in PRESCRIBED CIRCUMSTANCES." Draft Reg 149Q defines prescribed circumstances as: (a) unforeseen event unexpectedly affects nurse availability AND (b) the RN managing medication administration determines that medication for an individual resident cannot wait and delegates to someone other than a nurse. Per DH Vic guidance: exemption is "limited, time-bound" and does NOT cover routine rostering gaps, known vacancies, or business-as-usual shortages.',
        DATE '2026-07-01',
        'draft', TRUE,
        'BLOCKING: Reg 149Q is currently exposure draft (Feb 2026) — final regulation must be re-procured and this rule re-verified before activation. Engine must require: (a) recorded unforeseen event, (b) RN clinical-judgement determination, (c) documentation per Reg 149Q(5). Without all three, default rule VIC_PCW_S4S8S9DOD_ADMIN_NONSELF prevails.'
    );

END $$;

COMMIT;

-- ============================================================================
-- Migration 007 — Acceptance check (run after applying)
-- ============================================================================
-- Expected results:
--   SELECT COUNT(*) FROM clinical_sources WHERE regulatory_category='C';  -- 8
--   SELECT COUNT(*) FROM regulatory_scope_rules WHERE jurisdiction='AU-VIC';  -- 11
--   SELECT COUNT(*) FROM regulatory_scope_rules WHERE activation_status='draft';  -- 11 (all draft)
--   SELECT COUNT(*) FROM regulatory_scope_rules WHERE requires_legal_review=TRUE;  -- 11 (all flagged)
-- ============================================================================
