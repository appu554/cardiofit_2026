-- 014_snomed_au_rf2_indexes.sql
--
-- Indexes for kb7_snomed_* tables. Apply AFTER bulk COPY load
-- (see scripts/load_snomed_au_rf2.sh). Building indexes on
-- already-populated tables is 10-20x faster than maintaining
-- them during row-by-row COPY.

-- Concept: filter active concepts by module
CREATE INDEX IF NOT EXISTS idx_kb7_concept_active_module
    ON kb7_snomed_concept (active, module_id);

-- Description: lookup all descriptions for a concept (very hot path)
CREATE INDEX IF NOT EXISTS idx_kb7_description_concept
    ON kb7_snomed_description (concept_id, active);

-- Description: case-insensitive term search (for ACOP "find lab by name" queries)
CREATE INDEX IF NOT EXISTS idx_kb7_description_term_lower
    ON kb7_snomed_description (lower(term))
    WHERE active = 1;

-- Description: language + type filter (for "preferred term in en-au")
CREATE INDEX IF NOT EXISTS idx_kb7_description_lang_type
    ON kb7_snomed_description (language_code, type_id, active);

-- Relationship: find children of concept X (subsumption walk)
CREATE INDEX IF NOT EXISTS idx_kb7_relationship_source_type
    ON kb7_snomed_relationship (source_id, type_id, active);

-- Relationship: find parents of concept X (ancestor walk)
CREATE INDEX IF NOT EXISTS idx_kb7_relationship_dest_type
    ON kb7_snomed_relationship (destination_id, type_id, active);

-- Refset Simple: lookup refset memberships
CREATE INDEX IF NOT EXISTS idx_kb7_refset_simple_refset
    ON kb7_snomed_refset_simple (refset_id, active);

CREATE INDEX IF NOT EXISTS idx_kb7_refset_simple_component
    ON kb7_snomed_refset_simple (referenced_component_id, active);
