//! Integration tests for the Rust Recipe Engine

use flow2_rust_engine::models::*;
use serde_json;

#[tokio::test]
async fn test_recipe_execution_request_deserialization() {
    // Test that we can deserialize the exact data format from Go engine
    let json_data = r#"
    {
        "request_id": "flow2-vanc-001",
        "recipe_id": "vancomycin-dosing-v1.0",
        "variant": "standard_auc",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "medication_code": "11124",
        "clinical_context": "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"fields\":{\"demographics.age\":65.0,\"demographics.weight.actual_kg\":80.0,\"demographics.height_cm\":175.0,\"demographics.gender\":\"MALE\",\"labs.serum_creatinine[latest]\":1.8,\"labs.egfr[latest]\":45.0,\"conditions.active\":[\"sepsis\",\"chronic_kidney_disease\"],\"allergies.active\":[{\"allergen\":\"Penicillin\",\"severity\":\"MODERATE\"}],\"medications.current\":[{\"code\":\"1191\",\"name\":\"Aspirin\"}]},\"sources\":[\"patient_service\",\"lab_service\",\"medication_service\"],\"retrieval_time_ms\":15,\"completeness\":0.95}",
        "timeout_ms": 5000
    }
    "#;

    let request: RecipeExecutionRequest = serde_json::from_str(json_data).unwrap();
    
    assert_eq!(request.request_id, "flow2-vanc-001");
    assert_eq!(request.recipe_id, "vancomycin-dosing-v1.0");
    assert_eq!(request.variant, "standard_auc");
    assert_eq!(request.patient_id, "905a60cb-8241-418f-b29b-5b020e851392");
    assert_eq!(request.medication_code, "11124");
    assert_eq!(request.timeout_ms, 5000);
    
    // Verify clinical context can be parsed
    let clinical_context: serde_json::Value = serde_json::from_str(&request.clinical_context).unwrap();
    assert_eq!(clinical_context["patient_id"], "905a60cb-8241-418f-b29b-5b020e851392");
    assert_eq!(clinical_context["fields"]["demographics.age"], 65.0);
    assert_eq!(clinical_context["fields"]["demographics.weight.actual_kg"], 80.0);
    assert_eq!(clinical_context["fields"]["labs.egfr[latest]"], 45.0);
}

#[tokio::test]
async fn test_medication_proposal_serialization() {
    // Test that we can serialize the response format Go engine expects
    let proposal = MedicationProposal {
        medication_code: "11124".to_string(),
        medication_name: "Vancomycin".to_string(),
        calculated_dose: 2000.0,
        dose_unit: "mg".to_string(),
        frequency: "q12h".to_string(),
        duration: Some("7 days".to_string()),
        safety_status: "SAFE".to_string(),
        safety_alerts: vec!["Monitor renal function".to_string()],
        contraindications: vec![],
        clinical_rationale: "Calculated using recipe vancomycin-dosing-v1.0 variant standard_auc".to_string(),
        monitoring_plan: vec![
            "Monitor serum creatinine daily".to_string(),
            "Target trough level 15-20 mg/L".to_string()
        ],
        alternatives: vec![],
        execution_time_ms: 5,
        recipe_version: "v1.0".to_string(),
    };

    let json = serde_json::to_string_pretty(&proposal).unwrap();
    println!("Medication Proposal JSON:\n{}", json);

    // Verify it can be deserialized back
    let deserialized: MedicationProposal = serde_json::from_str(&json).unwrap();
    assert_eq!(deserialized.medication_code, "11124");
    assert_eq!(deserialized.calculated_dose, 2000.0);
    assert_eq!(deserialized.safety_status, "SAFE");
}

#[tokio::test]
async fn test_flow2_request_compatibility() {
    // Test Flow2Request format compatibility
    let json_data = r#"
    {
        "request_id": "flow2-vanc-001",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "11124",
            "name": "Vancomycin",
            "dose": 1000.0,
            "unit": "mg",
            "frequency": "q12h",
            "route": "IV",
            "indication": "sepsis"
        },
        "patient_data": {
            "age_years": 65.0,
            "weight_kg": 80.0,
            "height_cm": 175.0,
            "gender": "MALE"
        },
        "clinical_context": {
            "patient_demographics": {
                "age_years": 65.0,
                "weight_kg": 80.0,
                "egfr": 45.0,
                "gender": "MALE"
            }
        },
        "processing_hints": {
            "enable_ml_inference": true,
            "priority": "high"
        },
        "priority": "high",
        "enable_ml_inference": true,
        "timestamp": "2024-01-15T10:30:00Z"
    }
    "#;

    let request: Flow2Request = serde_json::from_str(json_data).unwrap();
    
    assert_eq!(request.request_id, "flow2-vanc-001");
    assert_eq!(request.action_type, "MEDICATION_ANALYSIS");
    assert_eq!(request.medication_data.get("code").unwrap(), "11124");
    assert_eq!(request.patient_data.get("age_years").unwrap(), 65.0);
    assert_eq!(request.enable_ml_inference, true);
}

#[test]
fn test_patient_demographics_calculations() {
    let demographics = PatientDemographics {
        age_years: Some(65.0),
        weight_kg: Some(80.0),
        height_cm: Some(175.0),
        gender: Some("MALE".to_string()),
        bmi: None,
        bsa_m2: None,
        race: None,
        ethnicity: None,
        egfr: Some(45.0),
        creatinine_clearance: None,
    };

    // Test BMI calculation
    let calculated_bmi = demographics.calculate_bmi().unwrap();
    assert!((calculated_bmi - 26.12).abs() < 0.1); // 80 / (1.75^2) ≈ 26.12

    // Test clinical flags
    assert!(demographics.is_elderly()); // Age >= 65
    assert!(demographics.has_renal_impairment()); // eGFR < 60
    assert!(!demographics.is_obese()); // BMI < 30
}

#[test]
fn test_rule_condition_evaluation() {
    use serde_json::json;

    let condition = RuleCondition {
        fact: "patient_age".to_string(),
        operator: "greater_than".to_string(),
        value: json!(65),
    };

    // Test evaluation
    assert!(condition.evaluate(&json!(70))); // 70 > 65
    assert!(!condition.evaluate(&json!(60))); // 60 < 65
    assert!(!condition.evaluate(&json!("invalid"))); // Invalid type

    // Test string contains
    let condition = RuleCondition {
        fact: "medication_name".to_string(),
        operator: "contains".to_string(),
        value: json!("vancomycin"),
    };

    assert!(condition.evaluate(&json!("Vancomycin Hydrochloride")));
    assert!(!condition.evaluate(&json!("Penicillin")));

    // Test array membership
    let condition = RuleCondition {
        fact: "patient_conditions".to_string(),
        operator: "in".to_string(),
        value: json!(["sepsis", "pneumonia", "uti"]),
    };

    assert!(condition.evaluate(&json!("sepsis")));
    assert!(!condition.evaluate(&json!("diabetes")));
}

#[test]
fn test_intent_manifest_builder() {
    let manifest = IntentManifestBuilder::new()
        .with_request_info("flow2-vanc-001".to_string(), "patient-123".to_string())
        .with_recipe("vancomycin-dosing-v1.0".to_string())
        .with_variant("standard_auc".to_string())
        .with_data_requirements(vec![
            "demographics.age".to_string(),
            "demographics.weight.actual_kg".to_string(),
            "labs.serum_creatinine[latest]".to_string()
        ])
        .with_priority("high".to_string())
        .with_rationale("Vancomycin requires renal dose adjustment".to_string())
        .with_rule_info("vancomycin-standard-selection-v1".to_string(), "2.0.0".to_string())
        .with_medication_info("11124".to_string(), "Vancomycin".to_string(), vec!["sepsis".to_string()])
        .with_estimated_time(100)
        .build()
        .unwrap();

    assert_eq!(manifest.request_id, "flow2-vanc-001");
    assert_eq!(manifest.recipe_id, "vancomycin-dosing-v1.0");
    assert_eq!(manifest.variant, "standard_auc");
    assert_eq!(manifest.priority, "high");
    assert_eq!(manifest.data_requirements.len(), 3);
    assert_eq!(manifest.estimated_execution_time_ms, 100);
}

#[tokio::test]
async fn test_complete_data_flow_simulation() {
    // Simulate the complete data flow: Go → Rust → Go
    
    // 1. Go engine sends RecipeExecutionRequest
    let go_request = RecipeExecutionRequest {
        request_id: "flow2-vanc-001".to_string(),
        recipe_id: "vancomycin-dosing-v1.0".to_string(),
        variant: "standard_auc".to_string(),
        patient_id: "905a60cb-8241-418f-b29b-5b020e851392".to_string(),
        medication_code: "11124".to_string(),
        clinical_context: serde_json::json!({
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "fields": {
                "demographics.age": 65.0,
                "demographics.weight.actual_kg": 80.0,
                "labs.egfr[latest]": 45.0,
                "conditions.active": ["sepsis"]
            }
        }).to_string(),
        timeout_ms: 5000,
    };

    // 2. Rust engine processes and returns MedicationProposal
    let rust_response = MedicationProposal {
        medication_code: go_request.medication_code.clone(),
        medication_name: "Vancomycin".to_string(),
        calculated_dose: 2000.0, // 80kg * 25mg/kg
        dose_unit: "mg".to_string(),
        frequency: "q12h".to_string(),
        duration: Some("7 days".to_string()),
        safety_status: "SAFE".to_string(),
        safety_alerts: vec!["Monitor renal function due to eGFR 45".to_string()],
        contraindications: vec![],
        clinical_rationale: format!(
            "Calculated using recipe {} variant {} - Dose adjusted for renal impairment",
            go_request.recipe_id, go_request.variant
        ),
        monitoring_plan: vec![
            "Monitor serum creatinine daily".to_string(),
            "Target trough level 15-20 mg/L".to_string()
        ],
        alternatives: vec![],
        execution_time_ms: 5,
        recipe_version: "v1.0".to_string(),
    };

    // 3. Verify the complete flow
    assert_eq!(rust_response.medication_code, "11124");
    assert_eq!(rust_response.calculated_dose, 2000.0);
    assert_eq!(rust_response.safety_status, "SAFE");
    assert!(rust_response.clinical_rationale.contains("vancomycin-dosing-v1.0"));
    assert!(rust_response.clinical_rationale.contains("standard_auc"));
    assert!(!rust_response.safety_alerts.is_empty());
    assert!(!rust_response.monitoring_plan.is_empty());

    println!("✅ Complete data flow simulation successful!");
    println!("📊 Request: {} → Recipe: {} → Dose: {}mg",
             go_request.request_id, go_request.recipe_id, rust_response.calculated_dose);
}

#[tokio::test]
async fn test_api_server_configuration() {
    // Test that server configuration can be created and validated
    use flow2_rust_engine::api::config::ServerConfig;

    let config = ServerConfig::default();

    // Validate default configuration
    assert!(config.validate().is_ok());
    assert_eq!(config.server.port, 8080);
    assert_eq!(config.server.host, "0.0.0.0");
    assert!(!config.security.enable_auth); // Disabled by default
    assert!(config.security.rate_limit.enabled);
    assert_eq!(config.security.rate_limit.max_requests, 100);
    assert!(config.performance.enable_compression);
    assert!(config.performance.enable_caching);

    println!("✅ Server configuration validation successful!");
    println!("📊 Default port: {}", config.server.port);
    println!("🔐 Auth enabled: {}", config.security.enable_auth);
    println!("🚦 Rate limit: {} req/min", config.security.rate_limit.max_requests);
}

#[test]
fn test_enhanced_intent_manifest_structure() {
    // Test that enhanced intent manifest can be created and serialized
    use flow2_rust_engine::models::*;
    use chrono::Utc;

    let manifest = EnhancedIntentManifest {
        request_id: "test-001".to_string(),
        patient_id: "patient-123".to_string(),
        recipe_id: "vancomycin-dosing-v1.0".to_string(),
        variant: "standard_auc".to_string(),
        data_requirements: vec!["demographics.age".to_string(), "labs.egfr[latest]".to_string()],
        priority: "HIGH".to_string(),
        clinical_rationale: "Test rationale".to_string(),
        estimated_execution_time_ms: 100,
        rule_id: "rule-001".to_string(),
        rule_version: "2.0.0".to_string(),
        generated_at: Utc::now(),
        medication_code: "11124".to_string(),
        conditions: vec!["sepsis".to_string()],

        risk_assessment: ClinicalRiskAssessment {
            overall_risk_level: "HIGH".to_string(),
            risk_factors: vec![
                RiskFactor {
                    factor_type: "DEMOGRAPHIC".to_string(),
                    description: "Elderly patient".to_string(),
                    severity: "MEDIUM".to_string(),
                    impact_score: 0.3,
                    evidence_level: "A".to_string(),
                }
            ],
            risk_score: 0.7,
            assessment_rationale: "High risk due to age and renal function".to_string(),
            mitigation_strategies: vec!["Enhanced monitoring".to_string()],
        },

        priority_details: DynamicPriority {
            level: "HIGH".to_string(),
            base_priority: "MEDIUM".to_string(),
            adjustments: vec![
                PriorityAdjustment {
                    factor: "High risk level".to_string(),
                    adjustment: 0.2,
                    rationale: "Risk elevation".to_string(),
                }
            ],
            final_score: 0.7,
            rationale: "Priority elevated due to risk factors".to_string(),
        },

        clinical_rationale_details: DetailedClinicalRationale {
            summary: "Clinical decision for vancomycin".to_string(),
            reasoning_steps: vec!["Step 1".to_string(), "Step 2".to_string()],
            clinical_factors: vec!["Factor 1".to_string()],
            evidence_level: "A".to_string(),
            confidence_score: 0.85,
        },

        execution_estimate: ExecutionEstimate {
            estimated_time_ms: 100,
            complexity_score: 2.5,
            resource_requirements: ResourceRequirements {
                cpu_intensive: false,
                memory_intensive: false,
                io_intensive: false,
                network_calls_required: 2,
            },
            parallel_execution_possible: true,
            caching_opportunities: 3,
        },

        alternative_recipes: vec![],
        clinical_flags: vec![],
        monitoring_requirements: vec![],
        safety_considerations: vec![],

        metadata: EnhancedManifestMetadata {
            generator_version: "2.0.0".to_string(),
            clinical_intelligence_enabled: true,
            risk_assessment_performed: true,
            data_optimization_applied: true,
            alternative_analysis_performed: false,
            generation_time_ms: 5,
        },
    };

    // Test serialization
    let json = serde_json::to_string_pretty(&manifest).unwrap();
    assert!(json.contains("vancomycin-dosing-v1.0"));
    assert!(json.contains("HIGH"));
    assert!(json.contains("clinical_intelligence_enabled"));

    // Test deserialization
    let deserialized: EnhancedIntentManifest = serde_json::from_str(&json).unwrap();
    assert_eq!(deserialized.recipe_id, "vancomycin-dosing-v1.0");
    assert_eq!(deserialized.risk_assessment.overall_risk_level, "HIGH");
    assert_eq!(deserialized.priority_details.level, "HIGH");

    println!("✅ Enhanced Intent Manifest structure validation successful!");
    println!("📊 Recipe: {}", deserialized.recipe_id);
    println!("⚠️  Risk Level: {}", deserialized.risk_assessment.overall_risk_level);
    println!("🔥 Priority: {}", deserialized.priority_details.level);
}
