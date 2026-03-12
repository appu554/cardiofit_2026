-- KB-5 Phase 3 Schema Migration: Clinical Safety Engines
-- Implements Drug-Disease Contraindications, Allergy Cross-Reactivity, and Duplicate Therapy Detection
-- Version: 2025Q4.phase3

-- ============================================================================
-- DRUG-DISEASE CONTRAINDICATIONS
-- ============================================================================

-- Drug-disease contraindication rules table
CREATE TABLE IF NOT EXISTS ddi_drug_disease_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dataset_version TEXT NOT NULL,
  drug_code TEXT NOT NULL,
  drug_name TEXT NOT NULL,
  disease_code TEXT NOT NULL,
  disease_name TEXT NOT NULL,
  code_system TEXT NOT NULL CHECK (code_system IN ('ICD10', 'SNOMED-CT', 'ICD9')),
  contraindication_type TEXT NOT NULL CHECK (contraindication_type IN ('absolute', 'relative', 'conditional')),
  severity ddi_severity NOT NULL,
  clinical_rationale TEXT NOT NULL,
  management_strategy TEXT NOT NULL,
  alternative_drugs JSONB,
  monitoring_parameters JSONB,
  exceptions JSONB,
  evidence evidence_level NOT NULL DEFAULT 'C',
  confidence NUMERIC(3,2) DEFAULT 0.80,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for drug-disease lookups
CREATE INDEX IF NOT EXISTS idx_drug_disease_drug_code
  ON ddi_drug_disease_rules(dataset_version, drug_code);
CREATE INDEX IF NOT EXISTS idx_drug_disease_disease_code
  ON ddi_drug_disease_rules(dataset_version, disease_code);
CREATE INDEX IF NOT EXISTS idx_drug_disease_active
  ON ddi_drug_disease_rules(active) WHERE active = TRUE;

-- ============================================================================
-- ALLERGY CROSS-REACTIVITY
-- ============================================================================

-- Allergy cross-reactivity rules table
CREATE TABLE IF NOT EXISTS ddi_allergy_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dataset_version TEXT NOT NULL,
  allergen_code TEXT NOT NULL,
  allergen_name TEXT NOT NULL,
  allergen_type TEXT NOT NULL CHECK (allergen_type IN ('drug', 'class', 'ingredient', 'excipient')),
  cross_reactive_drug TEXT NOT NULL,
  cross_reactive_drug_name TEXT NOT NULL,
  cross_reactivity_rate NUMERIC(5,2),
  severity ddi_severity NOT NULL,
  reaction_type TEXT NOT NULL CHECK (reaction_type IN ('anaphylaxis', 'urticaria', 'angioedema', 'rash', 'bronchospasm', 'other')),
  clinical_notes TEXT,
  alternatives JSONB,
  monitoring_required BOOLEAN DEFAULT FALSE,
  evidence evidence_level NOT NULL DEFAULT 'C',
  confidence NUMERIC(3,2) DEFAULT 0.80,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for allergy lookups
CREATE INDEX IF NOT EXISTS idx_allergy_allergen_code
  ON ddi_allergy_rules(dataset_version, allergen_code);
CREATE INDEX IF NOT EXISTS idx_allergy_cross_reactive
  ON ddi_allergy_rules(dataset_version, cross_reactive_drug);
CREATE INDEX IF NOT EXISTS idx_allergy_active
  ON ddi_allergy_rules(active) WHERE active = TRUE;

-- ============================================================================
-- DUPLICATE THERAPY DETECTION
-- ============================================================================

-- Therapeutic class mappings for duplicate therapy detection
CREATE TABLE IF NOT EXISTS ddi_therapeutic_classes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dataset_version TEXT NOT NULL,
  drug_code TEXT NOT NULL,
  drug_name TEXT NOT NULL,
  atc_code TEXT NOT NULL,
  atc_level INT NOT NULL CHECK (atc_level BETWEEN 1 AND 5),
  therapeutic_class TEXT NOT NULL,
  pharmacological_group TEXT,
  chemical_subgroup TEXT,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Duplicate therapy rules table
CREATE TABLE IF NOT EXISTS ddi_duplicate_therapy_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dataset_version TEXT NOT NULL,
  atc_code TEXT NOT NULL,
  atc_level INT NOT NULL CHECK (atc_level BETWEEN 1 AND 5),
  therapeutic_class TEXT NOT NULL,
  severity ddi_severity NOT NULL,
  allow_combination BOOLEAN DEFAULT FALSE,
  max_drugs_allowed INT DEFAULT 1,
  clinical_rationale TEXT NOT NULL,
  exceptions JSONB,
  evidence evidence_level NOT NULL DEFAULT 'C',
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for duplicate therapy lookups
CREATE INDEX IF NOT EXISTS idx_therapeutic_drug_code
  ON ddi_therapeutic_classes(dataset_version, drug_code);
CREATE INDEX IF NOT EXISTS idx_therapeutic_atc_code
  ON ddi_therapeutic_classes(dataset_version, atc_code);
CREATE INDEX IF NOT EXISTS idx_duplicate_therapy_atc
  ON ddi_duplicate_therapy_rules(dataset_version, atc_code);

-- ============================================================================
-- AUDIT TRIGGERS
-- ============================================================================

-- Trigger function to update timestamps
CREATE OR REPLACE FUNCTION update_phase3_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers to all Phase 3 tables
DROP TRIGGER IF EXISTS trg_drug_disease_updated ON ddi_drug_disease_rules;
CREATE TRIGGER trg_drug_disease_updated
  BEFORE UPDATE ON ddi_drug_disease_rules
  FOR EACH ROW EXECUTE FUNCTION update_phase3_timestamp();

DROP TRIGGER IF EXISTS trg_allergy_updated ON ddi_allergy_rules;
CREATE TRIGGER trg_allergy_updated
  BEFORE UPDATE ON ddi_allergy_rules
  FOR EACH ROW EXECUTE FUNCTION update_phase3_timestamp();

DROP TRIGGER IF EXISTS trg_therapeutic_classes_updated ON ddi_therapeutic_classes;
CREATE TRIGGER trg_therapeutic_classes_updated
  BEFORE UPDATE ON ddi_therapeutic_classes
  FOR EACH ROW EXECUTE FUNCTION update_phase3_timestamp();

DROP TRIGGER IF EXISTS trg_duplicate_therapy_updated ON ddi_duplicate_therapy_rules;
CREATE TRIGGER trg_duplicate_therapy_updated
  BEFORE UPDATE ON ddi_duplicate_therapy_rules
  FOR EACH ROW EXECUTE FUNCTION update_phase3_timestamp();

-- Comments for documentation
COMMENT ON TABLE ddi_drug_disease_rules IS 'Drug-disease contraindication rules with ICD-10/SNOMED-CT support';
COMMENT ON TABLE ddi_allergy_rules IS 'Allergy cross-reactivity rules (e.g., penicillin-cephalosporin)';
COMMENT ON TABLE ddi_therapeutic_classes IS 'Drug-to-ATC therapeutic class mappings for duplicate detection';
COMMENT ON TABLE ddi_duplicate_therapy_rules IS 'Rules for detecting therapeutic duplication (e.g., multiple NSAIDs)';
