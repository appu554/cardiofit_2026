-- 008_pbs_relational_indexes.sql
--
-- Indexes for the PBS relational graph. Apply AFTER bulk load.
-- Designed for the most common decision-support query shapes:
--   "give me all restrictions that apply to this drug"
--   "give me all prescribers allowed for this drug"
--   "give me the indication text for this restriction"
--   "give me the criteria + parameters for this restriction"

-- Item-keyed lookups (most queries start from a pbs_code)
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_atc_pbs        ON kb6_pbs_rel_item_atc (pbs_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_atc_atc        ON kb6_pbs_rel_item_atc (atc_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_prescribers_pbs     ON kb6_pbs_rel_prescribers (pbs_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_restr_pbs      ON kb6_pbs_rel_item_restrictions (pbs_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_restr_res      ON kb6_pbs_rel_item_restrictions (res_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_pt_pbs         ON kb6_pbs_rel_item_prescribing_texts (pbs_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_item_pt_pt          ON kb6_pbs_rel_item_prescribing_texts (prescribing_txt_id);

-- Restriction lookups (joined via res_code)
CREATE INDEX IF NOT EXISTS idx_kb6_rel_restr_res           ON kb6_pbs_rel_restrictions (res_code);

-- Prescribing text lookups (the central join hub)
CREATE INDEX IF NOT EXISTS idx_kb6_rel_pt_id               ON kb6_pbs_rel_prescribing_texts (prescribing_txt_id);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_pt_type             ON kb6_pbs_rel_prescribing_texts (prescribing_type);

-- Criteria + parameters tree
CREATE INDEX IF NOT EXISTS idx_kb6_rel_criteria_pt         ON kb6_pbs_rel_criteria (criteria_prescribing_txt_id);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_params_pt           ON kb6_pbs_rel_parameters (parameter_prescribing_txt_id);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_cp_criteria         ON kb6_pbs_rel_criteria_parameters (criteria_prescribing_txt_id);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_cp_parameter        ON kb6_pbs_rel_criteria_parameters (parameter_prescribing_txt_id);

-- Indication lookups
CREATE INDEX IF NOT EXISTS idx_kb6_rel_indications_pt      ON kb6_pbs_rel_indications (indication_prescribing_txt_id);

-- ATC dictionary
CREATE INDEX IF NOT EXISTS idx_kb6_rel_atc_code            ON kb6_pbs_rel_atc_codes (atc_code);
CREATE INDEX IF NOT EXISTS idx_kb6_rel_atc_parent          ON kb6_pbs_rel_atc_codes (atc_parent_code);

-- Programs (only 17 rows, but program_code is queried often)
CREATE INDEX IF NOT EXISTS idx_kb6_rel_programs_code       ON kb6_pbs_rel_programs (program_code);

-- Trigram index on restriction text (for substring search by clinicians)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm') THEN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
        CREATE INDEX IF NOT EXISTS idx_kb6_rel_pt_text_trgm
            ON kb6_pbs_rel_prescribing_texts USING gin (prescribing_txt gin_trgm_ops);
    END IF;
END $$;
