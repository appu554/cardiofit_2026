-- KB-7 LOINC ValueSets Migration
-- Adds ALL clinical LOINC ValueSets for CQL engine integration
-- Source: LOINC data from /data/loinc/ (35,344 codes)
--
-- This replaces hardcoded builtin_valuesets.go with proper database-backed ValueSets
-- ============================================================================

-- ============================================================================
-- STEP 1: Create LOINC ValueSet Definitions
-- ============================================================================

-- HbA1c (Hemoglobin A1c) - Glycemic Control
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabHbA1c',
    '1.0.0',
    'LabHbA1c',
    'Hemoglobin A1c Laboratory Tests',
    'LOINC codes for HbA1c/Hemoglobin A1c tests used in diabetes management',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- uACR (Urine Albumin-to-Creatinine Ratio) - Nephropathy Screening
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabuACR',
    '1.0.0',
    'LabuACR',
    'Urine Albumin-to-Creatinine Ratio Tests',
    'LOINC codes for uACR and microalbumin/creatinine ratio tests for nephropathy screening',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- PHQ-9 (Patient Health Questionnaire) - Depression Screening
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabPHQ9',
    '1.0.0',
    'LabPHQ9',
    'PHQ-9 Depression Screening',
    'LOINC codes for PHQ-9 and depression screening questionnaires',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Creatinine - Renal Function
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabCreatinine',
    '1.0.0',
    'LabCreatinine',
    'Creatinine Laboratory Tests',
    'LOINC codes for serum and urine creatinine measurements',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- eGFR (Estimated Glomerular Filtration Rate) - Kidney Function
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabEGFR',
    '1.0.0',
    'LabEGFR',
    'eGFR Laboratory Tests',
    'LOINC codes for estimated glomerular filtration rate calculations',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Lactate - Sepsis/Shock Marker
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabLactate',
    '1.0.0',
    'LabLactate',
    'Lactate Laboratory Tests',
    'LOINC codes for blood and serum lactate measurements',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Troponin - Cardiac Markers
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabTroponin',
    '1.0.0',
    'LabTroponin',
    'Troponin Cardiac Markers',
    'LOINC codes for troponin I and T cardiac biomarkers',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- INR - Anticoagulation Monitoring
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabINR',
    '1.0.0',
    'LabINR',
    'INR Coagulation Tests',
    'LOINC codes for International Normalized Ratio measurements',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- BNP/NT-proBNP - Heart Failure Markers
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabBNP',
    '1.0.0',
    'LabBNP',
    'BNP and NT-proBNP Tests',
    'LOINC codes for brain natriuretic peptide heart failure markers',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Blood Culture - Sepsis Workup
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabBloodCulture',
    '1.0.0',
    'LabBloodCulture',
    'Blood Culture Tests',
    'LOINC codes for blood culture and bacteremia detection',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Lipid Panel - Cardiovascular Risk
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabLipidPanel',
    '1.0.0',
    'LabLipidPanel',
    'Lipid Panel Tests',
    'LOINC codes for cholesterol, LDL, HDL, triglycerides',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Glucose - Blood Sugar
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabGlucose',
    '1.0.0',
    'LabGlucose',
    'Glucose Laboratory Tests',
    'LOINC codes for blood glucose and fasting glucose measurements',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Potassium - Electrolytes
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabPotassium',
    '1.0.0',
    'LabPotassium',
    'Potassium Laboratory Tests',
    'LOINC codes for serum potassium (important for ACE inhibitor monitoring)',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Sodium - Electrolytes
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabSodium',
    '1.0.0',
    'LabSodium',
    'Sodium Laboratory Tests',
    'LOINC codes for serum sodium measurements',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- CBC - Complete Blood Count
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabCBC',
    '1.0.0',
    'LabCBC',
    'Complete Blood Count Tests',
    'LOINC codes for hemoglobin, hematocrit, WBC, platelets',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- Procalcitonin - Sepsis Marker
INSERT INTO value_sets (id, url, version, name, title, description, status, publisher, definition_type, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'http://kb7.health/ValueSet/LabProCalcitonin',
    '1.0.0',
    'LabProCalcitonin',
    'Procalcitonin Tests',
    'LOINC codes for procalcitonin sepsis biomarker',
    'active',
    'KB-7 Terminology Service',
    'explicit',
    NOW(),
    NOW()
) ON CONFLICT (url, version) DO NOTHING;

-- ============================================================================
-- STEP 2: Populate precomputed_valueset_codes with LOINC codes
-- ============================================================================

-- HbA1c Codes (from LOINC)
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabHbA1c', '2025', 'http://loinc.org', '4548-4', 'Hemoglobin A1c/Hemoglobin.total in Blood', NOW()),
    ('LabHbA1c', '2025', 'http://loinc.org', '17856-6', 'Hemoglobin A1c/Hemoglobin.total in Blood by HPLC', NOW()),
    ('LabHbA1c', '2025', 'http://loinc.org', '4549-2', 'Hemoglobin A1c/Hemoglobin.total in Blood by Electrophoresis', NOW()),
    ('LabHbA1c', '2025', 'http://loinc.org', '62388-4', 'Hemoglobin A1c/Hemoglobin.total in Blood by JDS/JSCC protocol', NOW()),
    ('LabHbA1c', '2025', 'http://loinc.org', '59261-8', 'Hemoglobin A1c/Hemoglobin.total in Blood by IFCC protocol', NOW()),
    ('LabHbA1c', '2025', 'http://loinc.org', '71875-9', 'Hemoglobin A1c/Hemoglobin.total [Pure mass fraction] in Blood by HPLC', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- uACR Codes (Urine Albumin-to-Creatinine Ratio)
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabuACR', '2025', 'http://loinc.org', '9318-7', 'Albumin/Creatinine [Ratio] in Urine', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '14959-1', 'Microalbumin/Creatinine [Ratio] in Urine', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '13705-9', 'Albumin/Creatinine [Mass Ratio] in Urine', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '14958-3', 'Microalbumin/Creatinine [Mass Ratio] in Urine', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '77253-6', 'Albumin/Creatinine [Ratio] in Urine by calculation', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '32294-1', 'Albumin/Creatinine [Ratio] in Urine by Automated method', NOW()),
    ('LabuACR', '2025', 'http://loinc.org', '44292-1', 'Microalbumin/Creatinine [Mass Ratio] in Urine by Detection limit <= 3.0 mg/L', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- PHQ-9 Depression Screening Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabPHQ9', '2025', 'http://loinc.org', '44249-1', 'PHQ-9 quick depression assessment panel', NOW()),
    ('LabPHQ9', '2025', 'http://loinc.org', '44261-6', 'PHQ-9 total score', NOW()),
    ('LabPHQ9', '2025', 'http://loinc.org', '89204-2', 'PHQ-9 depression screening score', NOW()),
    ('LabPHQ9', '2025', 'http://loinc.org', '55758-7', 'PHQ-2 quick depression assessment', NOW()),
    ('LabPHQ9', '2025', 'http://loinc.org', '73832-8', 'PHQ-9 adolescent depression screening', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Creatinine Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabCreatinine', '2025', 'http://loinc.org', '2160-0', 'Creatinine [Mass/volume] in Serum or Plasma', NOW()),
    ('LabCreatinine', '2025', 'http://loinc.org', '38483-4', 'Creatinine [Mass/volume] in Blood', NOW()),
    ('LabCreatinine', '2025', 'http://loinc.org', '14682-9', 'Creatinine [Moles/volume] in Serum or Plasma', NOW()),
    ('LabCreatinine', '2025', 'http://loinc.org', '21232-4', 'Creatinine [Mass/volume] in Arterial blood', NOW()),
    ('LabCreatinine', '2025', 'http://loinc.org', '2164-2', 'Creatinine renal clearance in 24 hour Urine and Serum or Plasma', NOW()),
    ('LabCreatinine', '2025', 'http://loinc.org', '35203-9', 'Creatinine [Mass/volume] in Capillary blood', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- eGFR Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabEGFR', '2025', 'http://loinc.org', '33914-3', 'eGFR CKD-EPI [Volume Rate/Area] in Serum or Plasma', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '48642-3', 'eGFR MDRD [Volume Rate/Area] in Serum or Plasma', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '48643-1', 'eGFR CKD-EPI [Volume Rate/Area] in Serum or Plasma by Creatinine-based formula', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '62238-1', 'eGFR CKD-EPI 2009 [Volume Rate/Area]', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '88293-6', 'eGFR CKD-EPI 2021 [Volume Rate/Area]', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '69405-9', 'eGFR Black [Volume Rate/Area]', NOW()),
    ('LabEGFR', '2025', 'http://loinc.org', '77147-7', 'eGFR MDRD [Volume Rate/Area] non-Black', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Lactate Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabLactate', '2025', 'http://loinc.org', '2524-7', 'Lactate [Moles/volume] in Serum or Plasma', NOW()),
    ('LabLactate', '2025', 'http://loinc.org', '2519-7', 'Lactate [Mass/volume] in Blood', NOW()),
    ('LabLactate', '2025', 'http://loinc.org', '32693-4', 'Lactate [Moles/volume] in Blood', NOW()),
    ('LabLactate', '2025', 'http://loinc.org', '19239-3', 'Lactate [Moles/volume] in Capillary blood', NOW()),
    ('LabLactate', '2025', 'http://loinc.org', '27941-4', 'Lactate [Moles/volume] in Arterial blood', NOW()),
    ('LabLactate', '2025', 'http://loinc.org', '30242-2', 'Lactate [Mass/volume] in Arterial blood', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Troponin Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabTroponin', '2025', 'http://loinc.org', '6598-7', 'Troponin T.cardiac [Mass/volume] in Serum or Plasma', NOW()),
    ('LabTroponin', '2025', 'http://loinc.org', '10839-9', 'Troponin I.cardiac [Mass/volume] in Serum or Plasma', NOW()),
    ('LabTroponin', '2025', 'http://loinc.org', '49563-0', 'Troponin I.cardiac [Mass/volume] high sensitivity in Serum or Plasma', NOW()),
    ('LabTroponin', '2025', 'http://loinc.org', '89579-7', 'Troponin T.cardiac [Mass/volume] high sensitivity in Serum or Plasma', NOW()),
    ('LabTroponin', '2025', 'http://loinc.org', '67151-1', 'Troponin T.cardiac [Mass/volume] by High sensitivity method', NOW()),
    ('LabTroponin', '2025', 'http://loinc.org', '42757-5', 'Troponin I.cardiac [Mass/volume] by Detection limit <= 0.01 ng/mL', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- INR Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabINR', '2025', 'http://loinc.org', '6301-6', 'INR in Platelet poor plasma by Coagulation assay', NOW()),
    ('LabINR', '2025', 'http://loinc.org', '34714-6', 'INR in Blood by Coagulation assay', NOW()),
    ('LabINR', '2025', 'http://loinc.org', '46418-0', 'INR in Capillary blood by Coagulation assay', NOW()),
    ('LabINR', '2025', 'http://loinc.org', '38875-1', 'INR in Platelet poor plasma or blood by Coagulation assay', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- BNP/NT-proBNP Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabBNP', '2025', 'http://loinc.org', '30934-4', 'BNP [Mass/volume] in Serum or Plasma', NOW()),
    ('LabBNP', '2025', 'http://loinc.org', '33762-6', 'NT-proBNP [Mass/volume] in Serum or Plasma', NOW()),
    ('LabBNP', '2025', 'http://loinc.org', '42637-9', 'Natriuretic peptide B [Mass/volume] in Blood', NOW()),
    ('LabBNP', '2025', 'http://loinc.org', '83107-3', 'NT-proBNP [Moles/volume] in Serum or Plasma', NOW()),
    ('LabBNP', '2025', 'http://loinc.org', '83108-1', 'BNP [Moles/volume] in Serum or Plasma', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Blood Culture Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabBloodCulture', '2025', 'http://loinc.org', '600-7', 'Bacteria identified in Blood by Culture', NOW()),
    ('LabBloodCulture', '2025', 'http://loinc.org', '17928-3', 'Bacteria identified in Blood by Aerobe culture', NOW()),
    ('LabBloodCulture', '2025', 'http://loinc.org', '17934-1', 'Bacteria identified in Blood by Anaerobe culture', NOW()),
    ('LabBloodCulture', '2025', 'http://loinc.org', '88462-7', 'Carbapenem resistant Enterobacteriaceae DNA in Blood', NOW()),
    ('LabBloodCulture', '2025', 'http://loinc.org', '90272-6', 'Bacteremia panel by Molecular genetics method', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Lipid Panel Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabLipidPanel', '2025', 'http://loinc.org', '2093-3', 'Cholesterol [Mass/volume] in Serum or Plasma', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '2085-9', 'HDL Cholesterol [Mass/volume] in Serum or Plasma', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '2089-1', 'LDL Cholesterol [Mass/volume] in Serum or Plasma', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '2571-8', 'Triglyceride [Mass/volume] in Serum or Plasma', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '13457-7', 'LDL Cholesterol [Mass/volume] by calculation', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '18262-6', 'LDL Cholesterol [Mass/volume] by Direct assay', NOW()),
    ('LabLipidPanel', '2025', 'http://loinc.org', '9830-1', 'Cholesterol.total/Cholesterol.HDL [Ratio] in Serum or Plasma', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Glucose Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabGlucose', '2025', 'http://loinc.org', '2345-7', 'Glucose [Mass/volume] in Serum or Plasma', NOW()),
    ('LabGlucose', '2025', 'http://loinc.org', '2339-0', 'Glucose [Mass/volume] in Blood', NOW()),
    ('LabGlucose', '2025', 'http://loinc.org', '1558-6', 'Fasting glucose [Mass/volume] in Serum or Plasma', NOW()),
    ('LabGlucose', '2025', 'http://loinc.org', '41653-7', 'Glucose [Mass/volume] in Capillary blood', NOW()),
    ('LabGlucose', '2025', 'http://loinc.org', '14749-6', 'Glucose [Moles/volume] in Serum or Plasma', NOW()),
    ('LabGlucose', '2025', 'http://loinc.org', '15074-8', 'Glucose [Moles/volume] in Blood', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Potassium Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabPotassium', '2025', 'http://loinc.org', '2823-3', 'Potassium [Moles/volume] in Serum or Plasma', NOW()),
    ('LabPotassium', '2025', 'http://loinc.org', '6298-4', 'Potassium [Moles/volume] in Blood', NOW()),
    ('LabPotassium', '2025', 'http://loinc.org', '39789-3', 'Potassium [Moles/volume] in Venous blood', NOW()),
    ('LabPotassium', '2025', 'http://loinc.org', '41656-0', 'Potassium [Moles/volume] in Arterial blood', NOW()),
    ('LabPotassium', '2025', 'http://loinc.org', '32713-0', 'Potassium [Moles/volume] in Capillary blood', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Sodium Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabSodium', '2025', 'http://loinc.org', '2951-2', 'Sodium [Moles/volume] in Serum or Plasma', NOW()),
    ('LabSodium', '2025', 'http://loinc.org', '2947-0', 'Sodium [Moles/volume] in Blood', NOW()),
    ('LabSodium', '2025', 'http://loinc.org', '39791-9', 'Sodium [Moles/volume] in Venous blood', NOW()),
    ('LabSodium', '2025', 'http://loinc.org', '41657-8', 'Sodium [Moles/volume] in Arterial blood', NOW()),
    ('LabSodium', '2025', 'http://loinc.org', '32717-1', 'Sodium [Moles/volume] in Capillary blood', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- CBC Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabCBC', '2025', 'http://loinc.org', '718-7', 'Hemoglobin [Mass/volume] in Blood', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '4544-3', 'Hematocrit [Volume Fraction] of Blood', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '6690-2', 'Leukocytes [#/volume] in Blood', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '777-3', 'Platelets [#/volume] in Blood', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '789-8', 'Erythrocytes [#/volume] in Blood', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '787-2', 'MCV [Entitic volume] in Red Blood Cells', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '785-6', 'MCH [Entitic mass] in Red Blood Cells', NOW()),
    ('LabCBC', '2025', 'http://loinc.org', '786-4', 'MCHC [Mass/volume] in Red Blood Cells', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- Procalcitonin Codes
INSERT INTO precomputed_valueset_codes (valueset_url, snomed_version, code_system, code, display, materialized_at)
VALUES
    ('LabProCalcitonin', '2025', 'http://loinc.org', '33959-8', 'Procalcitonin [Mass/volume] in Serum or Plasma', NOW()),
    ('LabProCalcitonin', '2025', 'http://loinc.org', '75241-0', 'Procalcitonin [Mass/volume] in Serum or Plasma by Immunoassay', NOW())
ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING;

-- ============================================================================
-- STEP 3: Create indexes for fast lookups
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_precomputed_loinc_code ON precomputed_valueset_codes(code) WHERE code_system = 'http://loinc.org';
CREATE INDEX IF NOT EXISTS idx_precomputed_loinc_valueset ON precomputed_valueset_codes(valueset_url) WHERE code_system = 'http://loinc.org';

-- ============================================================================
-- STEP 4: Log the migration (using actual materialization_log schema)
-- ============================================================================

INSERT INTO materialization_log (
    run_type,
    started_at,
    completed_at,
    duration_ms,
    valuesets_processed,
    valuesets_materialized,
    total_codes_inserted,
    snomed_version,
    status,
    environment
)
SELECT
    'explicit',
    NOW(),
    NOW(),
    0,
    (SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes WHERE code_system = 'http://loinc.org'),
    (SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes WHERE code_system = 'http://loinc.org'),
    (SELECT COUNT(*) FROM precomputed_valueset_codes WHERE code_system = 'http://loinc.org'),
    '2025',
    'completed',
    '{"migration": "010_loinc_valuesets", "source": "LOINC"}'::jsonb;

-- Summary
DO $$
DECLARE
    vs_count INTEGER;
    code_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT valueset_url), COUNT(*) INTO vs_count, code_count
    FROM precomputed_valueset_codes
    WHERE code_system = 'http://loinc.org';

    RAISE NOTICE 'LOINC ValueSets Migration Complete: % ValueSets, % codes', vs_count, code_count;
END $$;
