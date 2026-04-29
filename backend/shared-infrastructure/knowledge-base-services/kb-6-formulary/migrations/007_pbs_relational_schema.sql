-- 007_pbs_relational_schema.sql
--
-- Relational PBS Schedule graph from the PBS API monthly CSV bundle.
-- These tables hold the AUTHORITATIVE relational data. The pre-existing
-- kb6_pbs_authorities/restrictions/indications/prescriber_types tables
-- (loaded from items.csv per-row flags) remain untouched as derived
-- per-item summaries.
--
-- All tables use TRUNCATE + COPY semantics for monthly idempotent reloads.
-- schedule_code identifies the PBS schedule revision the row belongs to;
-- a cross-month load would carry rows from multiple schedules.

-- ATC code dictionary
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_atc_codes (
    id              BIGSERIAL PRIMARY KEY,
    atc_code        TEXT,
    atc_description TEXT,
    atc_level       SMALLINT,
    atc_parent_code TEXT,
    schedule_code   INTEGER,
    loaded_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Item ↔ ATC linkage (many-to-many with priority)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_item_atc (
    id               BIGSERIAL PRIMARY KEY,
    atc_code         TEXT,
    pbs_code         TEXT,
    atc_priority_pct NUMERIC(7,4),
    schedule_code    INTEGER,
    loaded_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-item prescriber-type listings (full grain, vs. derived kb6_pbs_prescriber_types)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_prescribers (
    id               BIGSERIAL PRIMARY KEY,
    pbs_code         TEXT,
    prescriber_code  TEXT,           -- MP, NP, OP, DE, MW, etc.
    prescriber_type  TEXT,           -- "Medical Practitioner", "Nurse Practitioner", ...
    schedule_code    INTEGER,
    loaded_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indications attached to prescribing text IDs
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_indications (
    id                            BIGSERIAL PRIMARY KEY,
    indication_prescribing_txt_id BIGINT,
    condition                     TEXT,
    episodicity                   TEXT,
    severity                      TEXT,
    schedule_code                 INTEGER,
    loaded_at                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Restrictions (full text, replaces the 3-row degenerate kb6_pbs_restrictions)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_restrictions (
    id                          BIGSERIAL PRIMARY KEY,
    res_code                    TEXT,           -- composite key like "10041_6898_R"
    treatment_phase             TEXT,
    authority_method            TEXT,
    treatment_of_code           TEXT,
    restriction_number          TEXT,
    li_html_text                TEXT,           -- legal-instrument HTML (large)
    schedule_html_text          TEXT,           -- schedule HTML (large)
    note_indicator              BOOLEAN,
    caution_indicator           BOOLEAN,
    complex_authority_rqrd_ind  BOOLEAN,
    assessment_type_code        TEXT,
    criteria_relationship       TEXT,
    variation_rule_applied      TEXT,
    first_listing_date          DATE,
    written_authority_required  BOOLEAN,
    schedule_code               INTEGER,
    loaded_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prescribing texts (caution/note/criterion/parameter text bodies)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_prescribing_texts (
    id                          BIGSERIAL PRIMARY KEY,
    prescribing_txt_id          BIGINT,
    prescribing_type            TEXT,
    prescribing_txt             TEXT,
    prscrbg_txt_html            TEXT,
    complex_authority_rqrd_ind  BOOLEAN,
    assessment_type_code        TEXT,
    apply_to_increase_mq_flag   BOOLEAN,
    apply_to_increase_nr_flag   BOOLEAN,
    schedule_code               INTEGER,
    loaded_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Item ↔ Restriction (many-to-many with benefit type per linkage)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_item_restrictions (
    id                    BIGSERIAL PRIMARY KEY,
    res_code              TEXT,           -- composite key (FK to kb6_pbs_rel_restrictions.res_code)
    pbs_code              TEXT,
    benefit_type_code     TEXT,           -- A / S / R / U
    restriction_indicator TEXT,
    res_position          INTEGER,
    schedule_code         INTEGER,
    loaded_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Item ↔ Prescribing text
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_item_prescribing_texts (
    id                  BIGSERIAL PRIMARY KEY,
    pbs_code            TEXT,
    prescribing_txt_id  BIGINT,
    pt_position         INTEGER,
    schedule_code       INTEGER,
    loaded_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Criteria (clinical conditions for a restriction)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_criteria (
    id                            BIGSERIAL PRIMARY KEY,
    criteria_prescribing_txt_id   BIGINT,
    criteria_type                 TEXT,
    parameter_relationship        TEXT,
    schedule_code                 INTEGER,
    loaded_at                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Parameters (values plugged into criteria)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_parameters (
    id                            BIGSERIAL PRIMARY KEY,
    assessment_type               TEXT,
    parameter_prescribing_txt_id  BIGINT,
    parameter_type                TEXT,
    schedule_code                 INTEGER,
    loaded_at                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Criteria ↔ Parameter linkage
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_criteria_parameters (
    id                            BIGSERIAL PRIMARY KEY,
    criteria_prescribing_txt_id   BIGINT,
    parameter_prescribing_txt_id  BIGINT,
    pt_position                   INTEGER,
    schedule_code                 INTEGER,
    loaded_at                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- PBS programs (PB, EP, PL, CT, etc.)
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_programs (
    id            BIGSERIAL PRIMARY KEY,
    program_code  TEXT,
    program_title TEXT,
    schedule_code INTEGER,
    loaded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Load log specific to relational loader
CREATE TABLE IF NOT EXISTS kb6_pbs_rel_load_log (
    load_id       BIGSERIAL PRIMARY KEY,
    csv_name      TEXT,
    target_table  TEXT,
    rows_loaded   BIGINT,
    schedule_code INTEGER,
    loaded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes         TEXT
);
