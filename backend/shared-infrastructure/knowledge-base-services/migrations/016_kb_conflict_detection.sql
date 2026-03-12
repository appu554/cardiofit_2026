-- Inter-KB Conflict Detection System
-- Part II: Real-Time Analytics Platform - KB Conflict Monitoring

-- Real-time conflict detection between KBs
CREATE TABLE IF NOT EXISTS kb_conflict_detection (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    detection_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Conflict identification
    conflict_type VARCHAR(50) NOT NULL,
    conflict_category VARCHAR(30) NOT NULL CHECK (
        conflict_category IN ('logical', 'clinical', 'data', 'version', 'performance')
    ),
    
    -- Source KB information
    kb1_name VARCHAR(50) NOT NULL,
    kb1_version VARCHAR(50) NOT NULL,
    kb1_rule_id VARCHAR(200),
    kb1_endpoint VARCHAR(200),
    
    -- Target KB information
    kb2_name VARCHAR(50) NOT NULL,
    kb2_version VARCHAR(50) NOT NULL,
    kb2_rule_id VARCHAR(200),
    kb2_endpoint VARCHAR(200),
    
    -- Conflict details
    conflict_description TEXT NOT NULL,
    conflict_details JSONB NOT NULL,
    /* Example structures:
    
    Clinical Logic Conflict:
    {
      "type": "recommendation_conflict",
      "scenario": {
        "patient_context": {
          "age": 75,
          "conditions": ["hypertension", "ckd_stage_3"],
          "egfr": 42
        },
        "kb1_recommendation": {
          "drug": "ACE_inhibitor",
          "rationale": "hypertension_with_ckd_protection"
        },
        "kb2_recommendation": {
          "action": "avoid_ACE_inhibitor",
          "rationale": "hyperkalemia_risk_high"
        }
      },
      "conflict_severity": "major",
      "patient_safety_impact": "high"
    }
    
    Data Consistency Conflict:
    {
      "type": "data_inconsistency",
      "entity": "drug_interaction",
      "entity_id": "warfarin_simvastatin",
      "kb1_data": {
        "severity": "major",
        "mechanism": "CYP3A4_inhibition"
      },
      "kb2_data": {
        "severity": "moderate",
        "mechanism": "protein_binding_displacement"
      },
      "discrepancy": "severity_and_mechanism_mismatch"
    }
    
    Version Compatibility Conflict:
    {
      "type": "version_incompatibility",
      "breaking_change": true,
      "api_changes": ["endpoint_removed", "schema_modified"],
      "dependency_impact": ["kb_5_ddi", "kb_6_formulary"],
      "migration_required": true
    }
    */
    
    -- Impact assessment
    clinical_impact VARCHAR(20) NOT NULL CHECK (clinical_impact IN ('critical', 'major', 'moderate', 'minor')),
    patient_impact_estimate INTEGER DEFAULT 0,
    clinical_domains_affected TEXT[] DEFAULT '{}',
    
    -- Detection metadata
    detection_algorithm VARCHAR(100) NOT NULL,
    detection_confidence DECIMAL(3,2) NOT NULL,
    detection_version VARCHAR(20),
    
    -- Affected scope
    affected_drug_classes JSONB DEFAULT '[]',
    affected_conditions JSONB DEFAULT '[]',
    affected_patient_populations JSONB DEFAULT '[]',
    estimated_volume_impact INTEGER,
    
    -- Evidence and context
    supporting_evidence JSONB NOT NULL,
    /* Structure:
    {
      "evidence_sources": [
        {
          "type": "transaction_analysis",
          "sample_size": 1000,
          "conflict_rate": 0.15,
          "timeframe": "last_24_hours"
        },
        {
          "type": "rule_comparison",
          "rule_diff": {...},
          "semantic_analysis": {...}
        }
      ],
      "test_cases": [
        {
          "input": {...},
          "kb1_output": {...},
          "kb2_output": {...},
          "conflict_observed": true
        }
      ]
    }
    */
    
    -- Resolution tracking
    resolution_status VARCHAR(20) DEFAULT 'open' CHECK (
        resolution_status IN ('open', 'investigating', 'resolved', 'false_positive', 'deferred')
    ),
    resolution_strategy TEXT,
    resolution_details JSONB,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    
    -- Escalation
    escalated BOOLEAN DEFAULT FALSE,
    escalated_to VARCHAR(100),
    escalated_at TIMESTAMPTZ,
    escalation_reason TEXT,
    
    -- Governance
    assigned_to VARCHAR(100),
    assigned_at TIMESTAMPTZ,
    priority_score INTEGER DEFAULT 50,
    
    -- Related conflicts
    parent_conflict_id UUID REFERENCES kb_conflict_detection(id),
    related_conflicts UUID[] DEFAULT '{}',
    
    -- Monitoring
    last_occurrence TIMESTAMPTZ DEFAULT NOW(),
    occurrence_count INTEGER DEFAULT 1,
    first_detected TIMESTAMPTZ DEFAULT NOW(),
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_kb_conflict_timestamp ON kb_conflict_detection(detection_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_status ON kb_conflict_detection(resolution_status, clinical_impact);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_kbs ON kb_conflict_detection(kb1_name, kb2_name);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_type ON kb_conflict_detection(conflict_type, conflict_category);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_assigned ON kb_conflict_detection(assigned_to, resolution_status);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_escalated ON kb_conflict_detection(escalated, escalated_at) WHERE escalated = TRUE;

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_kb_conflict_details_gin ON kb_conflict_detection USING GIN(conflict_details);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_evidence_gin ON kb_conflict_detection USING GIN(supporting_evidence);
CREATE INDEX IF NOT EXISTS idx_kb_conflict_drug_classes_gin ON kb_conflict_detection USING GIN(affected_drug_classes);

-- ML model drift detection
CREATE TABLE IF NOT EXISTS ml_model_drift (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Model identification
    kb_name VARCHAR(50) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    model_type VARCHAR(50) NOT NULL, -- 'phenotype_classifier', 'risk_predictor', 'dose_optimizer', etc.
    
    -- Drift metrics
    drift_type VARCHAR(50) NOT NULL CHECK (
        drift_type IN ('data', 'concept', 'performance', 'prediction')
    ),
    drift_score DECIMAL(5,4) NOT NULL,
    drift_threshold DECIMAL(5,4) NOT NULL,
    drift_detected BOOLEAN GENERATED ALWAYS AS (drift_score > drift_threshold) STORED,
    
    -- Baseline vs current comparison
    baseline_period_start TIMESTAMPTZ NOT NULL,
    baseline_period_end TIMESTAMPTZ NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    
    -- Performance metrics comparison
    baseline_performance JSONB NOT NULL,
    current_performance JSONB NOT NULL,
    /* Structure:
    {
      "accuracy": 0.94,
      "precision": 0.92,
      "recall": 0.88,
      "f1_score": 0.90,
      "auc_roc": 0.96,
      "confidence_distribution": {
        "mean": 0.82,
        "std": 0.15,
        "percentiles": {"p25": 0.75, "p50": 0.85, "p75": 0.92}
      }
    }
    */
    
    -- Statistical tests
    statistical_tests JSONB NOT NULL,
    /* Structure:
    {
      "kolmogorov_smirnov": {
        "statistic": 0.23,
        "p_value": 0.001,
        "significant": true
      },
      "chi_square": {
        "statistic": 45.2,
        "p_value": 0.0001,
        "degrees_of_freedom": 10,
        "significant": true
      },
      "population_stability_index": {
        "psi": 0.18,
        "threshold": 0.1,
        "drift_detected": true
      },
      "adversarial_validation": {
        "auc": 0.72,
        "threshold": 0.5,
        "drift_detected": true
      }
    }
    */
    
    -- Data distribution analysis
    feature_drift_analysis JSONB,
    /* Structure:
    {
      "features": [
        {
          "name": "age",
          "drift_score": 0.05,
          "distribution_shift": "minimal",
          "baseline_stats": {"mean": 65.2, "std": 12.5},
          "current_stats": {"mean": 64.8, "std": 12.8}
        },
        {
          "name": "egfr",
          "drift_score": 0.34,
          "distribution_shift": "significant",
          "baseline_stats": {"mean": 68.5, "std": 22.3},
          "current_stats": {"mean": 72.1, "std": 19.8}
        }
      ],
      "correlation_changes": [
        {
          "feature_pair": ["age", "egfr"],
          "baseline_correlation": -0.45,
          "current_correlation": -0.38,
          "change_magnitude": 0.07
        }
      ]
    }
    */
    
    -- Impact assessment
    clinical_impact_assessment TEXT,
    recommended_action VARCHAR(100),
    urgency_level VARCHAR(20) DEFAULT 'medium' CHECK (
        urgency_level IN ('low', 'medium', 'high', 'critical')
    ),
    
    -- Root cause analysis
    potential_causes JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "cause": "population_shift",
        "likelihood": 0.7,
        "evidence": "significant_age_distribution_change",
        "description": "Patient population has shifted younger"
      },
      {
        "cause": "data_quality_degradation",
        "likelihood": 0.3,
        "evidence": "increased_missing_values",
        "description": "More missing values in key features"
      }
    ]
    */
    
    -- Status tracking
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    
    investigation_status VARCHAR(20) DEFAULT 'pending' CHECK (
        investigation_status IN ('pending', 'investigating', 'completed', 'deferred')
    ),
    investigated_by VARCHAR(100),
    investigation_notes TEXT,
    
    -- Actions taken
    action_taken TEXT,
    action_timestamp TIMESTAMPTZ,
    action_by VARCHAR(100),
    
    -- Follow-up
    retraining_required BOOLEAN DEFAULT FALSE,
    retraining_scheduled TIMESTAMPTZ,
    model_retired BOOLEAN DEFAULT FALSE,
    replacement_model VARCHAR(100)
);

-- Create indexes for ML drift monitoring
CREATE INDEX IF NOT EXISTS idx_ml_drift_model ON ml_model_drift(kb_name, model_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_drift_detected ON ml_model_drift(drift_detected, urgency_level, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_drift_type ON ml_model_drift(drift_type, kb_name);
CREATE INDEX IF NOT EXISTS idx_ml_drift_status ON ml_model_drift(investigation_status, acknowledged);

-- GIN indexes for JSONB analysis
CREATE INDEX IF NOT EXISTS idx_ml_drift_performance_gin ON ml_model_drift USING GIN(baseline_performance);
CREATE INDEX IF NOT EXISTS idx_ml_drift_current_perf_gin ON ml_model_drift USING GIN(current_performance);
CREATE INDEX IF NOT EXISTS idx_ml_drift_tests_gin ON ml_model_drift USING GIN(statistical_tests);
CREATE INDEX IF NOT EXISTS idx_ml_drift_features_gin ON ml_model_drift USING GIN(feature_drift_analysis);

-- Conflict pattern analysis table
CREATE TABLE IF NOT EXISTS kb_conflict_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_name VARCHAR(200) UNIQUE NOT NULL,
    pattern_description TEXT,
    
    -- Pattern definition
    pattern_conditions JSONB NOT NULL,
    /* Structure:
    {
      "kb_pairs": [
        {"kb1": "kb_1_dosing", "kb2": "kb_4_safety"},
        {"kb1": "kb_3_guidelines", "kb2": "kb_4_safety"}
      ],
      "conflict_types": ["recommendation_conflict", "safety_conflict"],
      "clinical_contexts": [
        {
          "conditions": ["hypertension", "ckd"],
          "age_range": [65, 85],
          "drug_classes": ["ACE_inhibitors", "ARBs"]
        }
      ],
      "frequency_threshold": 5,
      "time_window": "24 hours"
    }
    */
    
    -- Pattern metrics
    detection_count INTEGER DEFAULT 0,
    first_detected TIMESTAMPTZ,
    last_detected TIMESTAMPTZ,
    average_frequency_per_day DECIMAL(8,2),
    
    -- Impact analysis
    total_patient_impact INTEGER DEFAULT 0,
    clinical_severity_distribution JSONB DEFAULT '{}',
    resolution_time_stats JSONB DEFAULT '{}',
    
    -- Pattern status
    active BOOLEAN DEFAULT TRUE,
    monitoring_enabled BOOLEAN DEFAULT TRUE,
    alert_threshold INTEGER DEFAULT 10,
    
    -- Actions
    automated_response_enabled BOOLEAN DEFAULT FALSE,
    response_actions JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "action": "create_high_priority_alert",
        "parameters": {
          "recipients": ["clinical_informatics_team"],
          "escalation_minutes": 30
        }
      },
      {
        "action": "disable_conflicting_rules",
        "parameters": {
          "kb": "kb_1_dosing",
          "rule_pattern": "high_risk_*"
        }
      }
    ]
    */
    
    -- Metadata
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Functions for conflict detection and management

-- Function to detect conflicts between KB responses
CREATE OR REPLACE FUNCTION detect_kb_conflicts(
    transaction_id VARCHAR(100),
    kb_responses JSONB
)
RETURNS SETOF UUID AS $$
DECLARE
    kb1_record JSONB;
    kb2_record JSONB;
    conflict_id UUID;
    i INTEGER;
    j INTEGER;
    response_array JSONB[];
BEGIN
    -- Convert JSONB array to PostgreSQL array for easier iteration
    SELECT ARRAY(SELECT jsonb_array_elements(kb_responses)) INTO response_array;
    
    -- Compare each pair of KB responses
    FOR i IN 1..array_length(response_array, 1) LOOP
        FOR j IN (i+1)..array_length(response_array, 1) LOOP
            kb1_record := response_array[i];
            kb2_record := response_array[j];
            
            -- Check for recommendation conflicts
            IF (kb1_record->>'type' = 'recommendation' AND kb2_record->>'type' = 'recommendation') THEN
                -- Simple conflict detection logic (would be more sophisticated in practice)
                IF (kb1_record->'recommendation'->>'action' != kb2_record->'recommendation'->>'action') THEN
                    INSERT INTO kb_conflict_detection (
                        conflict_type,
                        conflict_category,
                        kb1_name,
                        kb1_version,
                        kb2_name,
                        kb2_version,
                        conflict_description,
                        conflict_details,
                        clinical_impact,
                        detection_algorithm,
                        detection_confidence,
                        supporting_evidence
                    ) VALUES (
                        'recommendation_conflict',
                        'clinical',
                        kb1_record->>'kb_name',
                        kb1_record->>'version',
                        kb2_record->>'kb_name',
                        kb2_record->>'version',
                        'Conflicting recommendations detected',
                        jsonb_build_object(
                            'kb1_recommendation', kb1_record->'recommendation',
                            'kb2_recommendation', kb2_record->'recommendation',
                            'transaction_id', transaction_id
                        ),
                        'moderate',
                        'automated_response_comparison',
                        0.85,
                        jsonb_build_object(
                            'detection_method', 'response_comparison',
                            'kb1_confidence', COALESCE(kb1_record->>'confidence', '0.5')::DECIMAL,
                            'kb2_confidence', COALESCE(kb2_record->>'confidence', '0.5')::DECIMAL
                        )
                    ) RETURNING id INTO conflict_id;
                    
                    RETURN NEXT conflict_id;
                END IF;
            END IF;
        END LOOP;
    END LOOP;
    
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate drift score for ML models
CREATE OR REPLACE FUNCTION calculate_model_drift(
    p_kb_name VARCHAR(50),
    p_model_id VARCHAR(100),
    p_baseline_start TIMESTAMPTZ,
    p_baseline_end TIMESTAMPTZ,
    p_current_start TIMESTAMPTZ,
    p_current_end TIMESTAMPTZ
)
RETURNS DECIMAL(5,4) AS $$
DECLARE
    drift_score DECIMAL(5,4);
    baseline_stats JSONB;
    current_stats JSONB;
BEGIN
    -- This is a simplified drift calculation
    -- In practice, this would involve more sophisticated statistical analysis
    
    -- For demonstration, we'll calculate a simple drift score based on 
    -- prediction accuracy change between baseline and current periods
    
    -- Get baseline performance metrics (from evidence envelopes or model logs)
    SELECT jsonb_build_object(
        'accuracy', 0.94,
        'prediction_count', 1000,
        'avg_confidence', 0.82
    ) INTO baseline_stats;
    
    -- Get current performance metrics
    SELECT jsonb_build_object(
        'accuracy', 0.89,
        'prediction_count', 1200,
        'avg_confidence', 0.78
    ) INTO current_stats;
    
    -- Calculate drift score (difference in accuracy)
    drift_score := ABS(
        (baseline_stats->>'accuracy')::DECIMAL - 
        (current_stats->>'accuracy')::DECIMAL
    );
    
    RETURN drift_score;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to update conflict patterns
CREATE OR REPLACE FUNCTION update_conflict_patterns()
RETURNS INTEGER AS $$
DECLARE
    pattern_record RECORD;
    conflict_count INTEGER;
    updated_patterns INTEGER := 0;
BEGIN
    FOR pattern_record IN SELECT * FROM kb_conflict_patterns WHERE active = TRUE LOOP
        -- Count recent conflicts matching this pattern
        SELECT COUNT(*) INTO conflict_count
        FROM kb_conflict_detection
        WHERE detection_timestamp > NOW() - INTERVAL '24 hours'
          AND (
              pattern_record.pattern_conditions->'conflict_types' IS NULL OR
              conflict_type = ANY(
                  SELECT jsonb_array_elements_text(pattern_record.pattern_conditions->'conflict_types')
              )
          );
        
        -- Update pattern metrics
        UPDATE kb_conflict_patterns
        SET 
            detection_count = detection_count + conflict_count,
            last_detected = CASE WHEN conflict_count > 0 THEN NOW() ELSE last_detected END,
            average_frequency_per_day = (
                detection_count + conflict_count
            )::DECIMAL / GREATEST(
                EXTRACT(EPOCH FROM (NOW() - first_detected))/86400, 1
            ),
            updated_at = NOW()
        WHERE id = pattern_record.id;
        
        updated_patterns := updated_patterns + 1;
    END LOOP;
    
    RETURN updated_patterns;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE kb_conflict_detection IS 'Real-time detection and tracking of conflicts between KB services';
COMMENT ON TABLE ml_model_drift IS 'Monitoring and detection of ML model drift across KB services';
COMMENT ON TABLE kb_conflict_patterns IS 'Pattern recognition and automated response for recurring KB conflicts';

COMMENT ON COLUMN kb_conflict_detection.conflict_details IS 'JSONB structure containing detailed conflict information and context';
COMMENT ON COLUMN ml_model_drift.statistical_tests IS 'JSONB containing results of statistical tests for drift detection';
COMMENT ON COLUMN kb_conflict_patterns.pattern_conditions IS 'JSONB defining the conditions that identify this conflict pattern';