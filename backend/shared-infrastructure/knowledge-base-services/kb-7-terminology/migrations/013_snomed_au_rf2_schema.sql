-- 013_snomed_au_rf2_schema.sql
--
-- RF2-native tables for SNOMED CT-AU (and AMT, which uses the same RF2 shape).
-- Coexists with the abstract terminology_concepts schema from 001_initial_schema.sql:
-- the abstract schema serves FHIR ValueSet ops; these tables serve native
-- RF2 hierarchy/subsumption queries that ACOP rules consume.
--
-- Snapshot semantics: id is the natural primary key (only latest active row per id).
-- Module ID column distinguishes International (900000000000207008),
-- SNOMED CT-AU (32506021000036107), and AMT (900062011000036103).
--
-- Indexes are created in a separate migration (014) AFTER bulk load, to avoid
-- index-maintenance overhead during COPY.

CREATE TABLE IF NOT EXISTS kb7_snomed_concept (
    id                   BIGINT      PRIMARY KEY,
    effective_time       DATE        NOT NULL,
    active               SMALLINT    NOT NULL,
    module_id            BIGINT      NOT NULL,
    definition_status_id BIGINT      NOT NULL
);

CREATE TABLE IF NOT EXISTS kb7_snomed_description (
    id                    BIGINT      PRIMARY KEY,
    effective_time        DATE        NOT NULL,
    active                SMALLINT    NOT NULL,
    module_id             BIGINT      NOT NULL,
    concept_id            BIGINT      NOT NULL,
    language_code         VARCHAR(8)  NOT NULL,
    type_id               BIGINT      NOT NULL,
    term                  TEXT        NOT NULL,
    case_significance_id  BIGINT      NOT NULL
);

CREATE TABLE IF NOT EXISTS kb7_snomed_relationship (
    id                       BIGINT    PRIMARY KEY,
    effective_time           DATE      NOT NULL,
    active                   SMALLINT  NOT NULL,
    module_id                BIGINT    NOT NULL,
    source_id                BIGINT    NOT NULL,
    destination_id           BIGINT    NOT NULL,
    relationship_group       INTEGER   NOT NULL,
    type_id                  BIGINT    NOT NULL,
    characteristic_type_id   BIGINT    NOT NULL,
    modifier_id              BIGINT    NOT NULL
);

CREATE TABLE IF NOT EXISTS kb7_snomed_refset_simple (
    id                        UUID     PRIMARY KEY,
    effective_time            DATE     NOT NULL,
    active                    SMALLINT NOT NULL,
    module_id                 BIGINT   NOT NULL,
    refset_id                 BIGINT   NOT NULL,
    referenced_component_id   BIGINT   NOT NULL
);

CREATE TABLE IF NOT EXISTS kb7_snomed_load_log (
    load_id        SERIAL    PRIMARY KEY,
    release_date   DATE      NOT NULL,
    source_file    TEXT      NOT NULL,
    table_name     TEXT      NOT NULL,
    rows_loaded    BIGINT    NOT NULL,
    sha256         TEXT,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
