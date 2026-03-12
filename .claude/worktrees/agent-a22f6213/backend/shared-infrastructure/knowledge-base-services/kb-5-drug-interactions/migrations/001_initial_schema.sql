-- KB-5 Drug Interactions Database Schema
-- Comprehensive drug-drug interaction detection and management

-- Drug interaction database
CREATE TABLE drug_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id VARCHAR(100) UNIQUE NOT NULL, -- e.g., "WARFARIN_ASPIRIN_001"
    
    -- Drug information
    drug_a_code VARCHAR(100) NOT NULL,
    drug_a_name VARCHAR(200) NOT NULL,
    drug_b_code VARCHAR(100) NOT NULL,
    drug_b_name VARCHAR(200) NOT NULL,
    
    -- Interaction classification
    interaction_type VARCHAR(50) NOT NULL, -- 'pharmacokinetic', 'pharmacodynamic', 'duplicate_therapy'
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('contraindicated', 'major', 'moderate', 'minor')),
    evidence_level VARCHAR(20) NOT NULL CHECK (evidence_level IN ('established', 'probable', 'theoretical', 'unknown')),
    
    -- Clinical details
    mechanism TEXT NOT NULL,
    clinical_effect TEXT NOT NULL,
    time_to_onset VARCHAR(50), -- 'immediate', 'hours', 'days', 'weeks'
    duration VARCHAR(50), -- 'temporary', 'permanent', 'variable'
    
    -- Management recommendations
    management_strategy TEXT NOT NULL,
    monitoring_parameters JSONB,
    dose_adjustment_required BOOLEAN DEFAULT FALSE,
    alternative_drugs JSONB,
    
    -- Documentation
    literature_references JSONB,
    documentation_level VARCHAR(20) CHECK (documentation_level IN ('excellent', 'good', 'fair', 'poor')),
    
    -- Clinical context
    patient_populations JSONB, -- specific populations where interaction applies
    comorbidity_factors JSONB, -- conditions that increase risk
    
    -- Metadata
    active BOOLEAN DEFAULT TRUE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    frequency_score DECIMAL(5,4), -- how often this interaction occurs (0.0-1.0)
    clinical_significance DECIMAL(5,4), -- clinical impact score (0.0-1.0)
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

-- Indexes for drug interactions
CREATE INDEX idx_drug_interactions_drug_a ON drug_interactions(drug_a_code);
CREATE INDEX idx_drug_interactions_drug_b ON drug_interactions(drug_b_code);
CREATE INDEX idx_drug_interactions_severity ON drug_interactions(severity);
CREATE INDEX idx_drug_interactions_type ON drug_interactions(interaction_type);
CREATE INDEX idx_drug_interactions_active ON drug_interactions(active) WHERE active = TRUE;
CREATE INDEX idx_drug_interactions_lookup ON drug_interactions(drug_a_code, drug_b_code) WHERE active = TRUE;

-- Drug synonyms and mappings for comprehensive lookup
CREATE TABLE drug_synonyms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_drug_code VARCHAR(100) NOT NULL,
    synonym_code VARCHAR(100) NOT NULL,
    synonym_name VARCHAR(200) NOT NULL,
    synonym_type VARCHAR(50) NOT NULL, -- 'rxnorm', 'ndc', 'brand', 'generic', 'atc'
    mapping_confidence DECIMAL(3,2) NOT NULL DEFAULT 1.0, -- 0.0-1.0
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_drug_synonyms_synonym ON drug_synonyms(synonym_code);
CREATE INDEX idx_drug_synonyms_primary ON drug_synonyms(primary_drug_code);
CREATE INDEX idx_drug_synonyms_type ON drug_synonyms(synonym_type);

-- Interaction patterns for similar drug classes
CREATE TABLE interaction_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_name VARCHAR(200) UNIQUE NOT NULL,
    drug_class_a VARCHAR(100) NOT NULL, -- ATC class or therapeutic category
    drug_class_b VARCHAR(100) NOT NULL,
    interaction_template JSONB NOT NULL, -- template for generating specific interactions
    severity_default VARCHAR(20) NOT NULL,
    mechanism_template TEXT NOT NULL,
    management_template TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Patient-specific interaction history
CREATE TABLE patient_interaction_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    interaction_id UUID NOT NULL REFERENCES drug_interactions(id),
    
    -- Alert context
    alert_triggered_at TIMESTAMPTZ DEFAULT NOW(),
    alert_severity VARCHAR(20) NOT NULL,
    clinical_context JSONB, -- patient-specific factors
    
    -- Drug details at time of alert
    drug_regimen JSONB NOT NULL, -- current drugs and dosages
    prescribing_physician VARCHAR(100),
    ordering_system VARCHAR(100),
    
    -- Alert disposition
    alert_status VARCHAR(20) DEFAULT 'active' CHECK (alert_status IN ('active', 'acknowledged', 'overridden', 'resolved')),
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    override_reason TEXT,
    resolution_action TEXT,
    
    -- Outcomes
    clinical_outcome VARCHAR(50), -- 'no_event', 'mild_event', 'serious_event', 'hospitalization'
    outcome_description TEXT,
    outcome_assessed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_patient_alerts_patient ON patient_interaction_alerts(patient_id);
CREATE INDEX idx_patient_alerts_interaction ON patient_interaction_alerts(interaction_id);
CREATE INDEX idx_patient_alerts_status ON patient_interaction_alerts(alert_status);
CREATE INDEX idx_patient_alerts_triggered ON patient_interaction_alerts(alert_triggered_at);

-- Drug interaction checking rules
CREATE TABLE interaction_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_type VARCHAR(50) NOT NULL, -- 'combination', 'dosage', 'timing', 'duration'
    
    -- Rule conditions
    conditions JSONB NOT NULL, -- complex rule logic
    drug_criteria JSONB NOT NULL, -- which drugs this rule applies to
    patient_criteria JSONB, -- patient populations
    
    -- Rule actions
    action_type VARCHAR(50) NOT NULL, -- 'alert', 'block', 'modify', 'monitor'
    alert_severity VARCHAR(20),
    alert_message TEXT,
    recommended_action TEXT,
    
    -- Rule metadata
    priority INTEGER DEFAULT 100, -- lower number = higher priority
    active BOOLEAN DEFAULT TRUE,
    effective_date TIMESTAMPTZ DEFAULT NOW(),
    expiry_date TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100)
);

CREATE INDEX idx_interaction_rules_type ON interaction_rules(rule_type);
CREATE INDEX idx_interaction_rules_active ON interaction_rules(active) WHERE active = TRUE;
CREATE INDEX idx_interaction_rules_priority ON interaction_rules(priority);

-- Drug formulation and route-specific interactions
CREATE TABLE drug_formulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code VARCHAR(100) NOT NULL,
    formulation_type VARCHAR(50) NOT NULL, -- 'tablet', 'injection', 'topical', 'inhaler'
    route_of_administration VARCHAR(50) NOT NULL, -- 'oral', 'iv', 'im', 'topical', 'inhalation'
    strength VARCHAR(100),
    
    -- Bioavailability and pharmacokinetics
    bioavailability DECIMAL(5,4), -- 0.0-1.0
    half_life_hours DECIMAL(8,2),
    time_to_peak_hours DECIMAL(6,2),
    
    -- Interaction modifiers
    interaction_modifiers JSONB, -- factors affecting interactions for this formulation
    
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_drug_formulations_drug ON drug_formulations(drug_code);
CREATE INDEX idx_drug_formulations_route ON drug_formulations(route_of_administration);

-- Clinical decision support configurations
CREATE TABLE cds_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(100) UNIQUE NOT NULL,
    healthcare_system VARCHAR(100) NOT NULL,
    
    -- Alerting thresholds
    severity_thresholds JSONB NOT NULL,
    /* Structure:
    {
      "contraindicated": {"enabled": true, "block_prescribing": true},
      "major": {"enabled": true, "require_acknowledgment": true},
      "moderate": {"enabled": true, "alert_only": true},
      "minor": {"enabled": false}
    }
    */
    
    -- Alert customization
    alert_templates JSONB,
    override_permissions JSONB,
    
    -- Integration settings
    external_systems JSONB, -- EMR integration configurations
    
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Interaction analytics and patterns
CREATE TABLE interaction_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_date DATE NOT NULL,
    interaction_id UUID REFERENCES drug_interactions(id),
    
    -- Usage statistics
    alert_frequency INTEGER DEFAULT 0,
    override_frequency INTEGER DEFAULT 0,
    clinical_events INTEGER DEFAULT 0,
    
    -- Patient demographics
    age_group_distribution JSONB,
    gender_distribution JSONB,
    comorbidity_patterns JSONB,
    
    -- Prescribing patterns
    prescriber_specialties JSONB,
    healthcare_settings JSONB,
    
    -- Outcomes
    adverse_event_rate DECIMAL(5,4),
    hospitalization_rate DECIMAL(5,4),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_interaction_analytics_date ON interaction_analytics(analysis_date);
CREATE INDEX idx_interaction_analytics_interaction ON interaction_analytics(interaction_id);

-- Insert sample drug interactions
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_a_name, drug_b_code, drug_b_name,
    interaction_type, severity, evidence_level, mechanism, clinical_effect,
    management_strategy, dose_adjustment_required, frequency_score, clinical_significance
) VALUES 
(
    'WARFARIN_ASPIRIN_001',
    'WARFARIN', 'Warfarin',
    'ASPIRIN', 'Aspirin',
    'pharmacodynamic',
    'major',
    'established',
    'Both drugs affect hemostasis. Warfarin inhibits vitamin K-dependent clotting factors, while aspirin irreversibly inhibits platelet aggregation through COX-1 inhibition.',
    'Increased risk of bleeding, particularly gastrointestinal and intracranial hemorrhage. Risk may be 2-4 times higher than with warfarin alone.',
    'If combination necessary, use lowest effective aspirin dose (≤100mg daily), monitor INR closely (weekly initially), consider PPI for GI protection, educate patient on bleeding signs.',
    FALSE,
    0.75,
    0.95
),
(
    'DIGOXIN_AMIODARONE_001',
    'DIGOXIN', 'Digoxin',
    'AMIODARONE', 'Amiodarone',
    'pharmacokinetic',
    'major',
    'established',
    'Amiodarone inhibits P-glycoprotein transport and reduces renal clearance of digoxin, leading to 2-3 fold increase in digoxin levels.',
    'Digitalis toxicity including nausea, vomiting, arrhythmias, heart block, and potentially fatal ventricular arrhythmias.',
    'Reduce digoxin dose by 50% when initiating amiodarone. Monitor digoxin levels after 1-2 weeks and adjust accordingly. Target digoxin level 0.8-2.0 ng/mL.',
    TRUE,
    0.60,
    0.90
),
(
    'METHOTREXATE_NSAID_001',
    'METHOTREXATE', 'Methotrexate',
    'IBUPROFEN', 'Ibuprofen',
    'pharmacokinetic',
    'major',
    'established',
    'NSAIDs reduce renal clearance of methotrexate and may displace it from protein binding sites, leading to increased methotrexate toxicity.',
    'Severe methotrexate toxicity including bone marrow suppression, mucositis, hepatotoxicity, and potentially fatal pancytopenia.',
    'Avoid concurrent use when possible. If necessary, use lowest NSAID dose for shortest duration, monitor CBC, liver function, and consider leucovorin rescue.',
    TRUE,
    0.45,
    0.85
),
(
    'STATINS_GEMFIBROZIL_001',
    'SIMVASTATIN', 'Simvastatin',
    'GEMFIBROZIL', 'Gemfibrozil',
    'pharmacokinetic',
    'contraindicated',
    'established',
    'Gemfibrozil inhibits CYP2C8 and glucuronidation pathways, dramatically increasing statin levels and risk of rhabdomyolysis.',
    'Severe myopathy and rhabdomyolysis with risk of acute kidney injury and death. Risk increased >10-fold compared to statin monotherapy.',
    'Contraindicated combination. Use alternative fibrate (fenofibrate) or different statin. If patient develops muscle symptoms, discontinue immediately and check CK, renal function.',
    FALSE,
    0.25,
    0.98
);

-- Insert drug synonyms for comprehensive lookup
INSERT INTO drug_synonyms (primary_drug_code, synonym_code, synonym_name, synonym_type) VALUES
('WARFARIN', 'RX_WARFARIN', 'Warfarin Sodium', 'rxnorm'),
('WARFARIN', 'COUMADIN', 'Coumadin', 'brand'),
('WARFARIN', 'JANTOVEN', 'Jantoven', 'brand'),
('ASPIRIN', 'RX_ASPIRIN', 'Acetylsalicylic Acid', 'rxnorm'),
('ASPIRIN', 'BAYER', 'Bayer Aspirin', 'brand'),
('DIGOXIN', 'RX_DIGOXIN', 'Digoxin', 'rxnorm'),
('DIGOXIN', 'LANOXIN', 'Lanoxin', 'brand'),
('AMIODARONE', 'RX_AMIODARONE', 'Amiodarone HCl', 'rxnorm'),
('AMIODARONE', 'CORDARONE', 'Cordarone', 'brand'),
('METHOTREXATE', 'RX_METHOTREXATE', 'Methotrexate', 'rxnorm'),
('METHOTREXATE', 'RHEUMATREX', 'Rheumatrex', 'brand'),
('IBUPROFEN', 'RX_IBUPROFEN', 'Ibuprofen', 'rxnorm'),
('IBUPROFEN', 'ADVIL', 'Advil', 'brand'),
('IBUPROFEN', 'MOTRIN', 'Motrin', 'brand'),
('SIMVASTATIN', 'RX_SIMVASTATIN', 'Simvastatin', 'rxnorm'),
('SIMVASTATIN', 'ZOCOR', 'Zocor', 'brand'),
('GEMFIBROZIL', 'RX_GEMFIBROZIL', 'Gemfibrozil', 'rxnorm'),
('GEMFIBROZIL', 'LOPID', 'Lopid', 'brand');

-- Insert interaction patterns for drug classes
INSERT INTO interaction_patterns (
    pattern_name, drug_class_a, drug_class_b, 
    interaction_template, severity_default, 
    mechanism_template, management_template
) VALUES
(
    'ACE_Inhibitor_Potassium_Sparing',
    'ACE_INHIBITORS',
    'POTASSIUM_SPARING_DIURETICS',
    '{"type": "pharmacodynamic", "effect": "hyperkalemia", "monitoring": ["serum_potassium", "renal_function"]}',
    'moderate',
    'Both drug classes increase serum potassium levels through different mechanisms, leading to additive hyperkalemic effects.',
    'Monitor serum potassium and renal function closely, especially in elderly patients or those with renal impairment.'
),
(
    'Anticoagulant_Antiplatelet',
    'ANTICOAGULANTS',
    'ANTIPLATELETS',
    '{"type": "pharmacodynamic", "effect": "bleeding", "monitoring": ["bleeding_signs", "coagulation_studies"]}',
    'major',
    'Additive effects on hemostasis through different pathways increase bleeding risk significantly.',
    'Use combination only when benefits clearly outweigh risks. Consider gastroprotection and close monitoring for bleeding.'
),
(
    'CNS_Depressant_Combinations',
    'BENZODIAZEPINES',
    'OPIOIDS',
    '{"type": "pharmacodynamic", "effect": "respiratory_depression", "monitoring": ["respiratory_status", "sedation_level"]}',
    'major',
    'Additive CNS depression can lead to severe respiratory depression, coma, and death.',
    'Avoid combination when possible. If necessary, use lowest effective doses and provide close monitoring.'
);

-- Insert sample CDS configuration
INSERT INTO cds_configurations (
    config_name, healthcare_system, severity_thresholds, alert_templates
) VALUES
(
    'default_hospital_config',
    'General Hospital System',
    '{"contraindicated": {"enabled": true, "block_prescribing": true, "require_override": true}, "major": {"enabled": true, "require_acknowledgment": true, "show_alternatives": true}, "moderate": {"enabled": true, "alert_only": true}, "minor": {"enabled": false}}',
    '{"contraindicated": "CONTRAINDICATED: {drug_a} and {drug_b} combination is contraindicated due to {mechanism}. Consider alternatives: {alternatives}", "major": "MAJOR INTERACTION: {drug_a} and {drug_b} may cause {clinical_effect}. {management_strategy}", "moderate": "MODERATE INTERACTION: Monitor for {clinical_effect} when using {drug_a} with {drug_b}."}'
);

-- Create functions for interaction checking
CREATE OR REPLACE FUNCTION check_drug_interaction(drug_codes text[])
RETURNS TABLE (
    interaction_id text,
    drug_a text,
    drug_b text,
    severity text,
    clinical_effect text,
    management text
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        di.interaction_id::text,
        di.drug_a_name::text,
        di.drug_b_name::text,
        di.severity::text,
        di.clinical_effect::text,
        di.management_strategy::text
    FROM drug_interactions di
    WHERE di.active = true
        AND (
            (di.drug_a_code = ANY(drug_codes) AND di.drug_b_code = ANY(drug_codes))
            OR 
            (di.drug_b_code = ANY(drug_codes) AND di.drug_a_code = ANY(drug_codes))
        )
        AND di.drug_a_code != di.drug_b_code;
END;
$$ LANGUAGE plpgsql;

-- Create function for synonym resolution
CREATE OR REPLACE FUNCTION resolve_drug_synonyms(input_codes text[])
RETURNS TABLE (
    input_code text,
    primary_code text,
    confidence numeric
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        unnest(input_codes)::text as input_code,
        COALESCE(ds.primary_drug_code, unnest(input_codes))::text as primary_code,
        COALESCE(ds.mapping_confidence, 1.0) as confidence
    FROM unnest(input_codes) as input_code
    LEFT JOIN drug_synonyms ds ON ds.synonym_code = input_code AND ds.active = true;
END;
$$ LANGUAGE plpgsql;