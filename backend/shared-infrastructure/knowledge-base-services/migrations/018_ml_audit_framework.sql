-- PostgreSQL: ML Audit Framework and Governance
-- Part IV: ML/Advanced Logic Audit Framework

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ML Model Registry - Comprehensive tracking of all ML models in the system
CREATE TABLE IF NOT EXISTS ml_model_registry (
    model_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_name VARCHAR(200) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    
    -- Model metadata
    model_type VARCHAR(50) NOT NULL, -- 'classification', 'regression', 'clustering', 'recommendation'
    framework VARCHAR(50) NOT NULL, -- 'tensorflow', 'pytorch', 'scikit-learn', 'xgboost', 'custom'
    algorithm VARCHAR(100) NOT NULL, -- 'random_forest', 'neural_network', 'logistic_regression', etc.
    
    -- Clinical context
    clinical_domain VARCHAR(100) NOT NULL,
    intended_use TEXT NOT NULL,
    target_population TEXT,
    contraindications TEXT,
    
    -- Training metadata
    training_data_source TEXT,
    training_data_size INTEGER,
    training_date TIMESTAMPTZ,
    training_duration_hours DECIMAL(8,2),
    
    -- Model artifacts and paths
    model_artifact_path TEXT NOT NULL,
    model_config JSONB,
    model_weights_checksum VARCHAR(64),
    
    -- Feature information
    input_features JSONB NOT NULL,
    /* Structure:
    [
      {
        "name": "age",
        "type": "numeric",
        "range": [0, 120],
        "required": true,
        "description": "Patient age in years"
      },
      {
        "name": "systolic_bp",
        "type": "numeric", 
        "range": [70, 250],
        "required": true,
        "units": "mmHg"
      }
    ]
    */
    
    output_schema JSONB NOT NULL,
    /* Structure:
    {
      "type": "classification",
      "classes": ["low_risk", "medium_risk", "high_risk"],
      "confidence_threshold": 0.7,
      "output_format": "probability_distribution"
    }
    */
    
    -- Performance metrics
    training_accuracy DECIMAL(5,4),
    validation_accuracy DECIMAL(5,4),
    test_accuracy DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    auc_roc DECIMAL(5,4),
    
    -- Bias and fairness metrics
    fairness_metrics JSONB,
    /* Structure:
    {
      "demographic_parity": 0.95,
      "equalized_odds": 0.92,
      "calibration_score": 0.88,
      "subgroup_performance": {
        "age_65_plus": {"accuracy": 0.89, "recall": 0.85},
        "gender_female": {"accuracy": 0.91, "recall": 0.87}
      }
    }
    */
    
    -- Regulatory and compliance
    regulatory_status VARCHAR(50) DEFAULT 'development', -- 'development', 'validation', 'approved', 'deprecated'
    fda_status VARCHAR(50),
    hipaa_compliance BOOLEAN DEFAULT FALSE,
    clinical_validation_status VARCHAR(50) DEFAULT 'pending',
    
    -- Deployment information
    deployed_environments TEXT[] DEFAULT '{}', -- 'development', 'staging', 'production'
    deployment_date TIMESTAMPTZ,
    deployment_config JSONB,
    
    -- Monitoring thresholds
    performance_thresholds JSONB DEFAULT '{}',
    /* Structure:
    {
      "accuracy_threshold": 0.85,
      "drift_threshold": 0.1,
      "latency_threshold_ms": 100,
      "memory_threshold_mb": 512
    }
    */
    
    -- Lifecycle management
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'retired')),
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Approval workflow
    clinical_approval_required BOOLEAN DEFAULT TRUE,
    clinical_approved_by VARCHAR(100),
    clinical_approved_at TIMESTAMPTZ,
    technical_approved_by VARCHAR(100),
    technical_approved_at TIMESTAMPTZ,
    
    -- Model lineage
    parent_model_id UUID REFERENCES ml_model_registry(model_id),
    model_lineage TEXT,
    
    UNIQUE(model_name, model_version)
);

-- ML Inference Audit Log - Track all model predictions
CREATE TABLE IF NOT EXISTS ml_inference_audit_log (
    inference_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Model identification
    model_id UUID NOT NULL REFERENCES ml_model_registry(model_id),
    model_name VARCHAR(200) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    
    -- Request context
    request_id VARCHAR(100),
    transaction_id VARCHAR(100),
    evidence_envelope_id UUID,
    kb_source VARCHAR(50),
    
    -- Clinical context
    patient_id_hash VARCHAR(64), -- Hashed for privacy
    clinical_domain VARCHAR(100),
    encounter_type VARCHAR(50),
    user_id VARCHAR(100),
    
    -- Input data (anonymized)
    input_features JSONB NOT NULL,
    feature_hash VARCHAR(64), -- Hash of input features for duplicate detection
    
    -- Prediction results
    prediction_output JSONB NOT NULL,
    /* Structure:
    {
      "prediction": "high_risk",
      "confidence": 0.87,
      "probability_distribution": {
        "low_risk": 0.05,
        "medium_risk": 0.08,
        "high_risk": 0.87
      },
      "feature_importance": {
        "age": 0.35,
        "systolic_bp": 0.28,
        "diabetes": 0.22
      }
    }
    */
    
    prediction_confidence DECIMAL(5,4),
    prediction_category VARCHAR(100),
    
    -- Performance metrics
    inference_latency_ms INTEGER,
    memory_usage_mb INTEGER,
    cpu_usage_percent DECIMAL(5,2),
    
    -- Quality indicators
    data_quality_score DECIMAL(5,4),
    missing_features INTEGER DEFAULT 0,
    out_of_range_features INTEGER DEFAULT 0,
    
    -- Clinical integration
    clinical_decision_influenced BOOLEAN DEFAULT FALSE,
    recommendation_overridden BOOLEAN DEFAULT FALSE,
    override_reason TEXT,
    
    -- Feedback and outcomes
    feedback_provided BOOLEAN DEFAULT FALSE,
    feedback_value DECIMAL(5,4), -- 0.0 (incorrect) to 1.0 (correct)
    clinical_outcome VARCHAR(50), -- 'positive', 'negative', 'neutral', 'unknown'
    outcome_timestamp TIMESTAMPTZ,
    
    -- Environment and deployment
    deployment_environment VARCHAR(50),
    service_version VARCHAR(50),
    infrastructure_details JSONB
);

-- Partition inference audit log by month
CREATE TABLE ml_inference_audit_log_y2024m01 PARTITION OF ml_inference_audit_log
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
    
CREATE TABLE ml_inference_audit_log_y2024m02 PARTITION OF ml_inference_audit_log
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
    
-- Continue with monthly partitions as needed...

-- Feature Drift Monitoring - Track changes in input data distribution
CREATE TABLE IF NOT EXISTS ml_feature_drift_monitoring (
    drift_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Model identification
    model_id UUID NOT NULL REFERENCES ml_model_registry(model_id),
    model_name VARCHAR(200) NOT NULL,
    
    -- Drift measurement window
    measurement_window INTERVAL NOT NULL, -- '1 hour', '1 day', '1 week'
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    
    -- Feature-level drift metrics
    feature_drift_scores JSONB NOT NULL,
    /* Structure:
    {
      "age": {
        "ks_statistic": 0.12,
        "p_value": 0.05,
        "drift_detected": true,
        "drift_magnitude": "moderate"
      },
      "systolic_bp": {
        "ks_statistic": 0.03,
        "p_value": 0.78,
        "drift_detected": false,
        "drift_magnitude": "none"
      }
    }
    */
    
    -- Overall drift assessment
    overall_drift_score DECIMAL(5,4),
    drift_detected BOOLEAN,
    drift_severity VARCHAR(20) CHECK (drift_severity IN ('none', 'mild', 'moderate', 'severe')),
    
    -- Statistical test results
    statistical_tests JSONB,
    /* Structure:
    {
      "kolmogorov_smirnov": {"statistic": 0.15, "p_value": 0.02},
      "chi_squared": {"statistic": 12.5, "p_value": 0.01},
      "population_stability_index": 0.18
    }
    */
    
    -- Data characteristics
    sample_size_baseline INTEGER,
    sample_size_current INTEGER,
    missing_data_rate DECIMAL(5,4),
    
    -- Actions taken
    alert_triggered BOOLEAN DEFAULT FALSE,
    alert_level VARCHAR(20),
    automated_actions JSONB DEFAULT '[]',
    
    -- Analysis metadata
    analysis_config JSONB,
    analysis_duration_seconds INTEGER
);

-- Model Performance Monitoring - Track model performance over time
CREATE TABLE IF NOT EXISTS ml_model_performance_monitoring (
    performance_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Model identification
    model_id UUID NOT NULL REFERENCES ml_model_registry(model_id),
    model_name VARCHAR(200) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    
    -- Performance measurement period
    measurement_period INTERVAL NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    
    -- Basic performance metrics
    total_predictions INTEGER,
    successful_predictions INTEGER,
    failed_predictions INTEGER,
    avg_confidence DECIMAL(5,4),
    
    -- Accuracy metrics (when ground truth is available)
    accuracy DECIMAL(5,4),
    precision DECIMAL(5,4),
    recall DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    auc_roc DECIMAL(5,4),
    
    -- Performance by subgroup
    subgroup_performance JSONB,
    /* Structure:
    {
      "age_groups": {
        "18_35": {"accuracy": 0.89, "sample_size": 245},
        "36_65": {"accuracy": 0.92, "sample_size": 567},
        "65_plus": {"accuracy": 0.85, "sample_size": 189}
      },
      "gender": {
        "male": {"accuracy": 0.90, "sample_size": 501},
        "female": {"accuracy": 0.91, "sample_size": 500}
      }
    }
    */
    
    -- Operational metrics
    avg_latency_ms DECIMAL(8,2),
    p95_latency_ms DECIMAL(8,2),
    p99_latency_ms DECIMAL(8,2),
    max_latency_ms INTEGER,
    avg_memory_usage_mb INTEGER,
    max_memory_usage_mb INTEGER,
    
    -- Error analysis
    error_rate DECIMAL(5,4),
    error_categories JSONB,
    /* Structure:
    {
      "data_validation_errors": 15,
      "model_execution_errors": 3,
      "timeout_errors": 1,
      "memory_errors": 0
    }
    */
    
    -- Clinical impact metrics
    clinical_decisions_influenced INTEGER,
    decisions_overridden INTEGER,
    override_rate DECIMAL(5,4),
    patient_safety_alerts INTEGER,
    
    -- Comparison to baseline
    baseline_comparison JSONB,
    /* Structure:
    {
      "accuracy_change": -0.02,
      "latency_change_ms": 5,
      "confidence_change": -0.01,
      "trend": "declining"
    }
    */
    
    -- Quality scores
    model_health_score DECIMAL(5,4),
    deployment_readiness_score DECIMAL(5,4),
    
    -- Environment and infrastructure
    deployment_environment VARCHAR(50),
    infrastructure_config JSONB
);

-- ML Model Validation Framework - Clinical and technical validation results
CREATE TABLE IF NOT EXISTS ml_model_validation_results (
    validation_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_id UUID NOT NULL REFERENCES ml_model_registry(model_id),
    
    -- Validation metadata
    validation_type VARCHAR(50) NOT NULL, -- 'clinical', 'technical', 'regulatory', 'performance'
    validation_phase VARCHAR(50) NOT NULL, -- 'development', 'pre_deployment', 'post_deployment'
    validation_date TIMESTAMPTZ DEFAULT NOW(),
    validator_id VARCHAR(100) NOT NULL,
    validator_role VARCHAR(50),
    
    -- Validation criteria and results
    validation_criteria JSONB NOT NULL,
    /* Structure:
    {
      "accuracy_threshold": 0.85,
      "fairness_requirements": ["demographic_parity", "equalized_odds"],
      "latency_requirement_ms": 100,
      "clinical_safety_requirements": ["no_false_negatives_critical"]
    }
    */
    
    validation_results JSONB NOT NULL,
    /* Structure:
    {
      "accuracy_test": {"result": "pass", "value": 0.89, "threshold": 0.85},
      "fairness_test": {"result": "fail", "demographic_parity": 0.82, "threshold": 0.90},
      "latency_test": {"result": "pass", "avg_latency": 78, "threshold": 100},
      "safety_test": {"result": "pass", "critical_false_negatives": 0}
    }
    */
    
    -- Overall validation outcome
    validation_status VARCHAR(20) NOT NULL CHECK (validation_status IN ('pass', 'fail', 'conditional')),
    overall_score DECIMAL(5,4),
    
    -- Issues and recommendations
    identified_issues JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    required_actions TEXT[],
    
    -- Clinical validation specifics
    clinical_expert_review TEXT,
    patient_safety_assessment TEXT,
    bias_assessment TEXT,
    interpretability_assessment TEXT,
    
    -- Follow-up requirements
    revalidation_required BOOLEAN DEFAULT FALSE,
    revalidation_timeline INTERVAL,
    monitoring_requirements JSONB,
    
    -- Approval chain
    approved BOOLEAN DEFAULT FALSE,
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,
    approval_conditions TEXT[]
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_ml_model_registry_name_version ON ml_model_registry(model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_ml_model_registry_status ON ml_model_registry(status, regulatory_status);
CREATE INDEX IF NOT EXISTS idx_ml_model_registry_clinical_domain ON ml_model_registry(clinical_domain);

CREATE INDEX IF NOT EXISTS idx_ml_inference_log_timestamp ON ml_inference_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_inference_log_model ON ml_inference_audit_log(model_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_inference_log_request ON ml_inference_audit_log(request_id);
CREATE INDEX IF NOT EXISTS idx_ml_inference_log_patient_hash ON ml_inference_audit_log(patient_id_hash);

CREATE INDEX IF NOT EXISTS idx_ml_drift_monitoring_model ON ml_feature_drift_monitoring(model_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_drift_monitoring_detected ON ml_feature_drift_monitoring(drift_detected, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_ml_performance_monitoring_model ON ml_model_performance_monitoring(model_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ml_performance_monitoring_period ON ml_model_performance_monitoring(period_start, period_end);

CREATE INDEX IF NOT EXISTS idx_ml_validation_model ON ml_model_validation_results(model_id, validation_date DESC);
CREATE INDEX IF NOT EXISTS idx_ml_validation_status ON ml_model_validation_results(validation_status, validation_type);

-- Functions for ML governance and monitoring

-- Function to register a new ML model
CREATE OR REPLACE FUNCTION register_ml_model(
    p_model_name VARCHAR(200),
    p_model_version VARCHAR(50),
    p_model_type VARCHAR(50),
    p_framework VARCHAR(50),
    p_algorithm VARCHAR(100),
    p_clinical_domain VARCHAR(100),
    p_intended_use TEXT,
    p_input_features JSONB,
    p_output_schema JSONB,
    p_created_by VARCHAR(100)
)
RETURNS UUID AS $$
DECLARE
    model_id UUID;
BEGIN
    INSERT INTO ml_model_registry (
        model_name, model_version, model_type, framework, algorithm,
        clinical_domain, intended_use, input_features, output_schema, created_by
    ) VALUES (
        p_model_name, p_model_version, p_model_type, p_framework, p_algorithm,
        p_clinical_domain, p_intended_use, p_input_features, p_output_schema, p_created_by
    ) RETURNING model_id INTO model_id;
    
    RETURN model_id;
END;
$$ LANGUAGE plpgsql;

-- Function to log ML inference
CREATE OR REPLACE FUNCTION log_ml_inference(
    p_model_id UUID,
    p_request_id VARCHAR(100),
    p_input_features JSONB,
    p_prediction_output JSONB,
    p_inference_latency_ms INTEGER,
    p_clinical_domain VARCHAR(100) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    inference_id UUID;
    model_name VARCHAR(200);
    model_version VARCHAR(50);
    feature_hash VARCHAR(64);
    prediction_confidence DECIMAL(5,4);
BEGIN
    -- Get model information
    SELECT mr.model_name, mr.model_version
    INTO model_name, model_version
    FROM ml_model_registry mr
    WHERE mr.model_id = p_model_id;
    
    -- Calculate feature hash for duplicate detection
    feature_hash := encode(digest(p_input_features::text, 'sha256'), 'hex');
    
    -- Extract confidence score from prediction output
    prediction_confidence := COALESCE((p_prediction_output->>'confidence')::DECIMAL(5,4), 0);
    
    INSERT INTO ml_inference_audit_log (
        model_id, model_name, model_version, request_id,
        input_features, feature_hash, prediction_output, 
        prediction_confidence, inference_latency_ms, clinical_domain
    ) VALUES (
        p_model_id, model_name, model_version, p_request_id,
        p_input_features, feature_hash, p_prediction_output,
        prediction_confidence, p_inference_latency_ms, p_clinical_domain
    ) RETURNING inference_id INTO inference_id;
    
    RETURN inference_id;
END;
$$ LANGUAGE plpgsql;

-- Function to detect feature drift
CREATE OR REPLACE FUNCTION detect_feature_drift(
    p_model_id UUID,
    p_measurement_window INTERVAL DEFAULT '1 day'
)
RETURNS BOOLEAN AS $$
DECLARE
    window_start TIMESTAMPTZ;
    window_end TIMESTAMPTZ;
    drift_detected BOOLEAN := FALSE;
    current_sample_size INTEGER;
    drift_score DECIMAL(5,4);
BEGIN
    window_end := NOW();
    window_start := window_end - p_measurement_window;
    
    -- Get current sample size
    SELECT COUNT(*)
    INTO current_sample_size
    FROM ml_inference_audit_log
    WHERE model_id = p_model_id
      AND timestamp BETWEEN window_start AND window_end;
    
    -- Only proceed if we have sufficient data
    IF current_sample_size < 100 THEN
        RETURN FALSE;
    END IF;
    
    -- This is a simplified drift detection - in practice, you'd implement
    -- statistical tests like Kolmogorov-Smirnov, Chi-squared, etc.
    -- For now, we'll use a placeholder calculation
    drift_score := RANDOM() * 0.3; -- Placeholder
    
    IF drift_score > 0.1 THEN
        drift_detected := TRUE;
        
        -- Log the drift detection
        INSERT INTO ml_feature_drift_monitoring (
            model_id, measurement_window, window_start, window_end,
            feature_drift_scores, overall_drift_score, drift_detected,
            drift_severity, sample_size_current
        ) VALUES (
            p_model_id, p_measurement_window, window_start, window_end,
            '{"placeholder": {"drift_score": 0.15, "drift_detected": true}}'::JSONB,
            drift_score, TRUE, 
            CASE 
                WHEN drift_score > 0.3 THEN 'severe'
                WHEN drift_score > 0.2 THEN 'moderate' 
                WHEN drift_score > 0.1 THEN 'mild'
                ELSE 'none'
            END,
            current_sample_size
        );
    END IF;
    
    RETURN drift_detected;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate model performance metrics
CREATE OR REPLACE FUNCTION calculate_model_performance(
    p_model_id UUID,
    p_measurement_period INTERVAL DEFAULT '1 day'
)
RETURNS VOID AS $$
DECLARE
    period_start TIMESTAMPTZ;
    period_end TIMESTAMPTZ;
    total_predictions INTEGER;
    successful_predictions INTEGER;
    failed_predictions INTEGER;
    avg_confidence DECIMAL(5,4);
    avg_latency DECIMAL(8,2);
    p95_latency DECIMAL(8,2);
    model_name VARCHAR(200);
    model_version VARCHAR(50);
BEGIN
    period_end := NOW();
    period_start := period_end - p_measurement_period;
    
    -- Get model information
    SELECT mr.model_name, mr.model_version
    INTO model_name, model_version
    FROM ml_model_registry mr
    WHERE mr.model_id = p_model_id;
    
    -- Calculate basic metrics
    SELECT 
        COUNT(*),
        COUNT(*) FILTER (WHERE prediction_output IS NOT NULL),
        COUNT(*) FILTER (WHERE prediction_output IS NULL),
        AVG(prediction_confidence),
        AVG(inference_latency_ms),
        PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY inference_latency_ms)
    INTO total_predictions, successful_predictions, failed_predictions,
         avg_confidence, avg_latency, p95_latency
    FROM ml_inference_audit_log
    WHERE model_id = p_model_id
      AND timestamp BETWEEN period_start AND period_end;
    
    -- Only insert if we have data
    IF total_predictions > 0 THEN
        INSERT INTO ml_model_performance_monitoring (
            model_id, model_name, model_version,
            measurement_period, period_start, period_end,
            total_predictions, successful_predictions, failed_predictions,
            avg_confidence, avg_latency_ms, p95_latency_ms
        ) VALUES (
            p_model_id, model_name, model_version,
            p_measurement_period, period_start, period_end,
            total_predictions, successful_predictions, failed_predictions,
            avg_confidence, avg_latency, p95_latency
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to validate model against criteria
CREATE OR REPLACE FUNCTION validate_ml_model(
    p_model_id UUID,
    p_validation_type VARCHAR(50),
    p_validator_id VARCHAR(100),
    p_validation_criteria JSONB
)
RETURNS UUID AS $$
DECLARE
    validation_id UUID;
    validation_results JSONB;
    validation_status VARCHAR(20);
    model_performance RECORD;
BEGIN
    -- Get recent model performance for validation
    SELECT 
        accuracy, precision, recall, f1_score,
        avg_latency_ms, error_rate
    INTO model_performance
    FROM ml_model_performance_monitoring
    WHERE model_id = p_model_id
    ORDER BY timestamp DESC
    LIMIT 1;
    
    -- Simplified validation logic - compare against thresholds
    validation_results := jsonb_build_object(
        'accuracy_test', jsonb_build_object(
            'result', CASE 
                WHEN COALESCE(model_performance.accuracy, 0) >= 
                     COALESCE((p_validation_criteria->>'accuracy_threshold')::DECIMAL, 0.8)
                THEN 'pass' 
                ELSE 'fail' 
            END,
            'value', COALESCE(model_performance.accuracy, 0),
            'threshold', COALESCE((p_validation_criteria->>'accuracy_threshold')::DECIMAL, 0.8)
        ),
        'latency_test', jsonb_build_object(
            'result', CASE 
                WHEN COALESCE(model_performance.avg_latency_ms, 999999) <= 
                     COALESCE((p_validation_criteria->>'latency_threshold_ms')::INTEGER, 100)
                THEN 'pass' 
                ELSE 'fail' 
            END,
            'value', COALESCE(model_performance.avg_latency_ms, 999999),
            'threshold', COALESCE((p_validation_criteria->>'latency_threshold_ms')::INTEGER, 100)
        )
    );
    
    -- Determine overall validation status
    validation_status := CASE 
        WHEN validation_results->'accuracy_test'->>'result' = 'pass' 
             AND validation_results->'latency_test'->>'result' = 'pass' 
        THEN 'pass'
        ELSE 'fail'
    END;
    
    INSERT INTO ml_model_validation_results (
        model_id, validation_type, validator_id, validation_criteria,
        validation_results, validation_status
    ) VALUES (
        p_model_id, p_validation_type, p_validator_id, p_validation_criteria,
        validation_results, validation_status
    ) RETURNING validation_id INTO validation_id;
    
    RETURN validation_id;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE ml_model_registry IS 'Comprehensive registry of all ML models with metadata, performance metrics, and governance tracking';
COMMENT ON TABLE ml_inference_audit_log IS 'Complete audit trail of all ML model predictions with clinical context and outcomes';
COMMENT ON TABLE ml_feature_drift_monitoring IS 'Real-time monitoring of input feature distributions to detect model degradation';
COMMENT ON TABLE ml_model_performance_monitoring IS 'Continuous tracking of model performance metrics and operational characteristics';
COMMENT ON TABLE ml_model_validation_results IS 'Clinical and technical validation results for ML models throughout their lifecycle';

COMMENT ON COLUMN ml_model_registry.input_features IS 'JSONB structure defining expected input features with validation rules and metadata';
COMMENT ON COLUMN ml_model_registry.fairness_metrics IS 'Bias and fairness assessment results across different demographic groups';
COMMENT ON COLUMN ml_inference_audit_log.prediction_output IS 'Complete model prediction including confidence scores and feature importance';
COMMENT ON COLUMN ml_feature_drift_monitoring.feature_drift_scores IS 'Statistical test results for each feature detecting distribution changes';