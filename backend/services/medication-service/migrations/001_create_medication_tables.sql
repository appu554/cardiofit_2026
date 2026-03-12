-- Migration: Create Medication Service Tables
-- Migrates from Google Healthcare API to PostgreSQL with enhanced pharmaceutical intelligence
-- Version: 1.0
-- Date: 2024-01-15

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable JSONB GIN indexing
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- =====================================================
-- MEDICATIONS TABLE - Core pharmaceutical intelligence
-- =====================================================

CREATE TABLE medications (
    medication_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Standard Identifiers
    rxnorm_code VARCHAR(20) NOT NULL,
    ndc_codes TEXT[], -- Array of NDC codes
    generic_name VARCHAR(255) NOT NULL,
    brand_names TEXT[], -- Array of brand names
    synonyms TEXT[], -- Alternative names
    
    -- Clinical Classification
    therapeutic_class VARCHAR(100) NOT NULL,
    pharmacologic_class VARCHAR(100) NOT NULL,
    
    -- Safety Profile
    is_high_alert BOOLEAN DEFAULT FALSE,
    is_controlled_substance BOOLEAN DEFAULT FALSE,
    dea_schedule INTEGER CHECK (dea_schedule BETWEEN 1 AND 5),
    black_box_warning TEXT,
    
    -- Dosing Information
    dosing_type VARCHAR(50) NOT NULL CHECK (dosing_type IN (
        'fixed', 'weight_based', 'bsa_based', 'auc_based', 'tiered', 'protocol_based'
    )),
    
    -- Dose Ranges and Calculations
    standard_dose_min DECIMAL(10,4),
    standard_dose_max DECIMAL(10,4),
    weight_based_dose_mg_kg DECIMAL(8,4),
    bsa_based_dose_mg_m2 DECIMAL(10,4),
    max_single_dose DECIMAL(10,4),
    max_daily_dose DECIMAL(10,4),
    
    -- Adjustment Requirements
    renal_adjustment_required BOOLEAN DEFAULT FALSE,
    hepatic_adjustment_required BOOLEAN DEFAULT FALSE,
    therapeutic_drug_monitoring BOOLEAN DEFAULT FALSE,
    
    -- Pharmacokinetic Properties (JSONB for flexibility)
    pharmacokinetics JSONB,
    -- Example structure:
    -- {
    --   "absorption_rate": 2.5,
    --   "bioavailability": 0.85,
    --   "protein_binding": 95.0,
    --   "half_life_hours": 12.0,
    --   "metabolism_pathway": "hepatic",
    --   "active_metabolites": ["metabolite1", "metabolite2"]
    -- }
    
    -- Pharmacodynamic Properties
    pharmacodynamics JSONB,
    -- Example structure:
    -- {
    --   "mechanism_of_action": "ACE inhibition",
    --   "onset_of_action_hours": 1.0,
    --   "duration_of_action_hours": 24.0,
    --   "therapeutic_window": {"min": 10, "max": 20}
    -- }
    
    -- Safety and Monitoring
    safety_profile JSONB,
    -- Example structure:
    -- {
    --   "contraindications": ["pregnancy", "severe_renal_impairment"],
    --   "common_adverse_effects": ["dizziness", "cough"],
    --   "serious_adverse_effects": ["angioedema"],
    --   "drug_interactions": ["potassium_supplements", "nsaids"],
    --   "monitoring_requirements": ["renal_function", "potassium"],
    --   "pregnancy_category": "D",
    --   "lactation_safety": "compatible"
    -- }
    
    -- Age-Specific Dosing
    age_specific_dosing JSONB,
    -- Example structure:
    -- {
    --   "pediatric": {"dose_mg_kg": 15.0, "min_age_months": 6},
    --   "geriatric": {"adjustment_factor": 0.75, "max_dose": 40}
    -- }
    
    -- Clinical Indications
    clinical_indications TEXT[],
    off_label_uses TEXT[],
    
    -- Operational Data
    formulary_status VARCHAR(50) DEFAULT 'formulary',
    average_cost DECIMAL(10,2),
    availability_status VARCHAR(50) DEFAULT 'available',
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    
    -- Constraints
    CONSTRAINT uk_medications_rxnorm UNIQUE (rxnorm_code),
    CONSTRAINT chk_medications_dosing_data CHECK (
        (dosing_type = 'weight_based' AND weight_based_dose_mg_kg IS NOT NULL) OR
        (dosing_type = 'bsa_based' AND bsa_based_dose_mg_m2 IS NOT NULL) OR
        (dosing_type IN ('fixed', 'tiered') AND standard_dose_min IS NOT NULL) OR
        (dosing_type IN ('auc_based', 'protocol_based'))
    )
);

-- =====================================================
-- MEDICATION FORMULATIONS - Available formulations
-- =====================================================

CREATE TABLE medication_formulations (
    formulation_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    medication_id UUID NOT NULL REFERENCES medications(medication_id) ON DELETE CASCADE,
    
    -- Formulation Details
    dosage_form VARCHAR(50) NOT NULL, -- tablet, capsule, injection, etc.
    strength DECIMAL(10,4) NOT NULL,
    strength_unit VARCHAR(20) NOT NULL,
    route_of_administration VARCHAR(20) NOT NULL,
    
    -- Release and Storage
    release_mechanism VARCHAR(50), -- immediate, extended, delayed
    storage_requirements TEXT,
    
    -- Additional Properties
    excipients TEXT[],
    stability_data JSONB,
    bioequivalence_data JSONB,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT chk_formulations_strength CHECK (strength > 0)
);

-- =====================================================
-- PRESCRIPTIONS - Two-phase prescription management
-- =====================================================

CREATE TABLE prescriptions (
    prescription_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    proposal_id UUID, -- Links to original proposal
    
    -- Patient and Medication
    patient_id VARCHAR(100) NOT NULL,
    medication_id UUID NOT NULL REFERENCES medications(medication_id),
    
    -- Prescription Details
    dose_value DECIMAL(10,4) NOT NULL,
    dose_unit VARCHAR(20) NOT NULL,
    route VARCHAR(50) NOT NULL,
    frequency JSONB NOT NULL,
    -- Example frequency structure:
    -- {
    --   "type": "scheduled",
    --   "interval_hours": 8,
    --   "times_per_day": 3,
    --   "display_string": "Three times daily"
    -- }
    
    duration_days INTEGER,
    quantity DECIMAL(10,2),
    quantity_unit VARCHAR(20),
    refills INTEGER DEFAULT 0,
    
    -- Clinical Information
    indication VARCHAR(500) NOT NULL,
    special_instructions TEXT,
    prescriber_id VARCHAR(100) NOT NULL,
    
    -- Two-Phase Lifecycle
    status VARCHAR(50) NOT NULL DEFAULT 'PROPOSED' CHECK (status IN ('PROPOSED', 'COMMITTED', 'CANCELLED')),
    
    -- Phase 1: Proposal
    proposal_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    proposal_context JSONB,
    -- Example proposal context:
    -- {
    --   "calculation_method": "weight_based",
    --   "patient_weight_kg": 70.0,
    --   "dose_mg_kg": 10.0,
    --   "confidence_score": 0.95,
    --   "warnings": ["mild_renal_impairment"],
    --   "recipe_id": "standard-dose-calculation-v1"
    -- }
    
    -- Phase 2: Commitment
    commit_timestamp TIMESTAMP WITH TIME ZONE,
    commit_metadata JSONB,
    -- Example commit metadata:
    -- {
    --   "committed_by": "workflow-engine-v1.2",
    --   "safety_validation_passed": true,
    --   "modifications_applied": {
    --     "dose_reduced": true,
    --     "reason": "drug_interaction_detected"
    --   }
    -- }
    
    committed_by VARCHAR(100), -- Workflow Engine identifier
    
    -- Audit Trail
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    
    -- Constraints
    CONSTRAINT chk_prescriptions_dose CHECK (dose_value > 0),
    CONSTRAINT chk_prescriptions_quantity CHECK (quantity IS NULL OR quantity > 0),
    CONSTRAINT chk_prescriptions_refills CHECK (refills >= 0),
    CONSTRAINT chk_prescriptions_commitment CHECK (
        (status = 'COMMITTED' AND commit_timestamp IS NOT NULL AND committed_by IS NOT NULL) OR
        (status != 'COMMITTED')
    )
);

-- =====================================================
-- MEDICATION PROTOCOLS - Clinical protocol management
-- =====================================================

CREATE TABLE medication_protocols (
    protocol_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Protocol Identity
    protocol_name VARCHAR(255) NOT NULL,
    protocol_type VARCHAR(50) NOT NULL, -- chemotherapy, antibiotic, chronic_disease
    specialty VARCHAR(100),
    version VARCHAR(20) NOT NULL,
    
    -- Protocol Structure
    total_cycles INTEGER,
    cycle_length_days INTEGER,
    
    -- Protocol Medications (JSONB for complex structures)
    medications JSONB NOT NULL,
    -- Example structure:
    -- [
    --   {
    --     "medication_id": "uuid",
    --     "day": 1,
    --     "dose_mg_m2": 100,
    --     "route": "IV",
    --     "pre_medications": ["ondansetron", "dexamethasone"]
    --   }
    -- ]
    
    -- Monitoring and Safety
    monitoring_requirements JSONB,
    dose_modifications JSONB,
    stopping_criteria JSONB,
    
    -- Status
    active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT uk_protocols_name_version UNIQUE (protocol_name, version)
);

-- =====================================================
-- PROTOCOL ENROLLMENTS - Patient protocol tracking
-- =====================================================

CREATE TABLE protocol_enrollments (
    enrollment_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Patient and Protocol
    patient_id VARCHAR(100) NOT NULL,
    protocol_id UUID NOT NULL REFERENCES medication_protocols(protocol_id),
    
    -- Enrollment Status
    start_date DATE NOT NULL,
    end_date DATE,
    current_cycle INTEGER DEFAULT 1,
    current_day INTEGER DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'DISCONTINUED')),
    
    -- Modifications and Adjustments
    dose_modifications JSONB,
    -- Example structure:
    -- [
    --   {
    --     "medication_id": "uuid",
    --     "cycle": 2,
    --     "percentage": 75,
    --     "reason": "grade_2_neutropenia",
    --     "applied_at": "2024-01-15T10:30:00Z"
    --   }
    -- ]
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT chk_enrollments_cycles CHECK (current_cycle > 0),
    CONSTRAINT chk_enrollments_days CHECK (current_day > 0)
);

-- =====================================================
-- OUTBOX EVENTS - Reliable event publishing
-- =====================================================

CREATE TABLE outbox_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event Identity
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    
    -- Event Data
    event_data JSONB NOT NULL,
    
    -- Publishing Status
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT chk_outbox_event_type CHECK (event_type IN (
        'MedicationProposed', 'MedicationCommitted', 'MedicationModified',
        'MedicationDiscontinued', 'ProtocolInitiated', 'ProtocolCompleted',
        'ProtocolModified', 'EnrollmentStarted', 'EnrollmentCompleted'
    ))
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Medications indexes
CREATE INDEX idx_medications_therapeutic_class ON medications(therapeutic_class);
CREATE INDEX idx_medications_pharmacologic_class ON medications(pharmacologic_class);
CREATE INDEX idx_medications_dosing_type ON medications(dosing_type);
CREATE INDEX idx_medications_high_alert ON medications(is_high_alert) WHERE is_high_alert = TRUE;
CREATE INDEX idx_medications_controlled ON medications(is_controlled_substance) WHERE is_controlled_substance = TRUE;

-- Full-text search on medication names
CREATE INDEX idx_medications_search_gin ON medications USING gin(
    to_tsvector('english', 
        coalesce(generic_name, '') || ' ' || 
        coalesce(array_to_string(brand_names, ' '), '') || ' ' ||
        coalesce(array_to_string(synonyms, ' '), '')
    )
);

-- JSONB indexes for complex queries
CREATE INDEX idx_medications_pharmacokinetics_gin ON medications USING gin(pharmacokinetics);
CREATE INDEX idx_medications_safety_profile_gin ON medications USING gin(safety_profile);

-- Formulations indexes
CREATE INDEX idx_formulations_medication_id ON medication_formulations(medication_id);
CREATE INDEX idx_formulations_route ON medication_formulations(route_of_administration);

-- Prescriptions indexes
CREATE INDEX idx_prescriptions_patient_date ON prescriptions(patient_id, proposal_timestamp DESC);
CREATE INDEX idx_prescriptions_medication_date ON prescriptions(medication_id, proposal_timestamp DESC);
CREATE INDEX idx_prescriptions_prescriber ON prescriptions(prescriber_id);
CREATE INDEX idx_prescriptions_status ON prescriptions(status);
CREATE INDEX idx_prescriptions_proposal_id ON prescriptions(proposal_id) WHERE proposal_id IS NOT NULL;

-- Protocols indexes
CREATE INDEX idx_protocols_type ON medication_protocols(protocol_type);
CREATE INDEX idx_protocols_specialty ON medication_protocols(specialty);
CREATE INDEX idx_protocols_active ON medication_protocols(active) WHERE active = TRUE;

-- Enrollments indexes
CREATE INDEX idx_enrollments_patient ON protocol_enrollments(patient_id);
CREATE INDEX idx_enrollments_protocol ON protocol_enrollments(protocol_id);
CREATE INDEX idx_enrollments_status ON protocol_enrollments(status);
CREATE INDEX idx_enrollments_dates ON protocol_enrollments(start_date, end_date);

-- Outbox indexes
CREATE INDEX idx_outbox_unpublished ON outbox_events(created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id);

-- =====================================================
-- TABLE PARTITIONING FOR SCALABILITY
-- =====================================================

-- Partition prescriptions by month for better performance
ALTER TABLE prescriptions PARTITION BY RANGE (proposal_timestamp);

-- Create partitions for current and future months
CREATE TABLE prescriptions_2024_01 PARTITION OF prescriptions
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE prescriptions_2024_02 PARTITION OF prescriptions
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Add more partitions as needed...

-- =====================================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- =====================================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to relevant tables
CREATE TRIGGER update_medications_updated_at BEFORE UPDATE ON medications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prescriptions_updated_at BEFORE UPDATE ON prescriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_protocols_updated_at BEFORE UPDATE ON medication_protocols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_enrollments_updated_at BEFORE UPDATE ON protocol_enrollments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON TABLE medications IS 'Core pharmaceutical intelligence with clinical properties and dosing guidelines';
COMMENT ON TABLE medication_formulations IS 'Available formulations for each medication';
COMMENT ON TABLE prescriptions IS 'Two-phase prescription management with proposal/commit lifecycle';
COMMENT ON TABLE medication_protocols IS 'Clinical protocols for complex medication regimens';
COMMENT ON TABLE protocol_enrollments IS 'Patient enrollment and tracking in medication protocols';
COMMENT ON TABLE outbox_events IS 'Reliable event publishing using outbox pattern';

COMMENT ON COLUMN medications.pharmacokinetics IS 'JSONB containing absorption, distribution, metabolism, excretion data';
COMMENT ON COLUMN medications.pharmacodynamics IS 'JSONB containing mechanism of action and therapeutic effects';
COMMENT ON COLUMN medications.safety_profile IS 'JSONB containing contraindications, adverse effects, and monitoring requirements';
COMMENT ON COLUMN prescriptions.proposal_context IS 'JSONB containing calculation details and business context from proposal phase';
COMMENT ON COLUMN prescriptions.commit_metadata IS 'JSONB containing validation results and modifications from commit phase';
