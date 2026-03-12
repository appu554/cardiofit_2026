use flow2_rust_engine::{
    unified_clinical_engine::{
        UnifiedClinicalEngine,
        knowledge_base::KnowledgeBase,
        ClinicalRequest,
        PatientContext,
        BiologicalSex,
        PregnancyStatus,
        RenalFunction,
        HepaticFunction
    },
    api::{server::create_router, config::ServerConfig},
};
use std::{sync::Arc, collections::HashMap};
use chrono::Utc;
use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use tower::util::ServiceExt;
use serde_json::{json, Value};

/// Helper function to create a test unified engine
async fn create_test_engine() -> Arc<UnifiedClinicalEngine> {
    let knowledge_base = Arc::new(KnowledgeBase::new("./knowledge").await.unwrap());

    // Debug: Print loaded drug rules
    println!("Loaded drug rules:");
    for drug_id in knowledge_base.get_all_drug_ids() {
        println!("  - {}", drug_id);
    }

    Arc::new(UnifiedClinicalEngine::new(knowledge_base).unwrap())
}

/// Helper function to create a test patient context
fn create_test_patient_context() -> PatientContext {
    PatientContext {
        age_years: 45.0,
        weight_kg: 70.0,
        height_cm: 170.0,
        sex: BiologicalSex::Male,
        pregnancy_status: PregnancyStatus::NotPregnant,
        renal_function: RenalFunction {
            egfr_ml_min: Some(90.0),
            egfr_ml_min_1_73m2: Some(90.0),
            creatinine_mg_dl: Some(1.0),
            bun_mg_dl: Some(15.0),
            creatinine_clearance: Some(100.0),
            stage: Some("Normal".to_string()),
        },
        hepatic_function: HepaticFunction {
            child_pugh_class: Some("A".to_string()),
            alt_u_l: Some(25.0),
            ast_u_l: Some(30.0),
            bilirubin_mg_dl: Some(1.0),
            albumin_g_dl: Some(4.0),
        },
        active_medications: vec![],
        allergies: vec![],
        conditions: vec![],
        lab_values: HashMap::new(),
    }
}

/// Helper function to create a test clinical request
fn create_test_clinical_request(drug_id: &str) -> ClinicalRequest {
    ClinicalRequest {
        request_id: format!("test-{}", uuid::Uuid::new_v4()),
        drug_id: drug_id.to_string(),
        indication: "hypertension".to_string(),
        patient_context: create_test_patient_context(),
        timestamp: Utc::now(),
    }
}

#[tokio::test]
async fn test_unified_engine_initialization() {
    let engine = create_test_engine().await;

    // Test that the engine initializes successfully
    // Just verify the engine was created successfully
    assert!(Arc::strong_count(&engine) > 0);

    // Test basic functionality - process a simple clinical request
    let request = create_test_clinical_request("metformin");
    let result = engine.process_clinical_request(request).await;
    if let Err(e) = &result {
        println!("Engine initialization test error: {}", e);
    }
    assert!(result.is_ok());
}

#[tokio::test]
async fn test_clinical_request_processing() {
    let engine = create_test_engine().await;
    let request = create_test_clinical_request("metformin");
    
    // Test processing a clinical request
    let result = engine.process_clinical_request(request).await;
    if let Err(e) = &result {
        println!("Engine processing error: {}", e);
    }
    assert!(result.is_ok());
    
    let response = result.unwrap();
    assert!(!response.request_id.is_empty());
    assert_eq!(response.drug_id, "metformin");
    assert!(response.processing_time_ms > 0);
}

#[tokio::test]
async fn test_dose_optimization_endpoint() {
    let engine = create_test_engine().await;
    let config = ServerConfig::default();
    let app = create_router(engine, config);
    
    let request_body = json!({
        "request_id": "test-dose-123",
        "patient_id": "patient-456",
        "medication_code": "metformin",
        "clinical_parameters": {},
        "optimization_type": "standard",
        "clinical_context": {
            "age_years": 45,
            "weight_kg": 70.0,
            "height_cm": 170.0,
            "sex": "male",
            "egfr": 90.0
        },
        "processing_hints": {}
    });
    
    let request = Request::builder()
        .method("POST")
        .uri("/api/dose/optimize")
        .header("content-type", "application/json")
        .body(Body::from(request_body.to_string()))
        .unwrap();
    
    let response = app.oneshot(request).await.unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    
    let body = axum::body::to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let response_json: Value = serde_json::from_slice(&body).unwrap();
    
    assert_eq!(response_json["request_id"], "test-dose-123");
    assert!(response_json["optimized_dose"].as_f64().unwrap() > 0.0);
    assert!(response_json["execution_time_ms"].as_u64().unwrap() > 0);
}

#[tokio::test]
async fn test_medication_intelligence_endpoint() {
    let engine = create_test_engine().await;
    let config = ServerConfig::default();
    let app = create_router(engine, config);
    
    let request_body = json!({
        "request_id": "test-intel-123",
        "patient_id": "patient-456",
        "medications": [{
            "code": "metformin",
            "name": "Metformin",
            "dose": 500.0,
            "unit": "mg",
            "frequency": "BID",
            "route": "PO",
            "duration": "ongoing",
            "indication": "diabetes",
            "properties": {}
        }],
        "intelligence_type": "comprehensive",
        "analysis_depth": "detailed",
        "clinical_context": {
            "age_years": 45,
            "weight_kg": 70.0,
            "height_cm": 170.0,
            "sex": "male",
            "egfr": 90.0
        }
    });
    
    let request = Request::builder()
        .method("POST")
        .uri("/api/medication/intelligence")
        .header("content-type", "application/json")
        .body(Body::from(request_body.to_string()))
        .unwrap();
    
    let response = app.oneshot(request).await.unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    
    let body = axum::body::to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let response_json: Value = serde_json::from_slice(&body).unwrap();
    
    assert_eq!(response_json["request_id"], "test-intel-123");
    assert!(response_json["intelligence_score"].as_f64().unwrap() > 0.0);
    assert!(response_json["execution_time_ms"].as_u64().unwrap() > 0);
}

#[tokio::test]
async fn test_flow2_endpoint() {
    let engine = create_test_engine().await;
    let config = ServerConfig::default();
    let app = create_router(engine, config);
    
    let request_body = json!({
        "request_id": "test-flow2-123",
        "patient_id": "patient-456",
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "medication_code": "metformin",
            "indication": "diabetes"
        },
        "patient_data": {
            "age_years": 45,
            "weight_kg": 70.0
        },
        "clinical_context": {
            "age_years": 45,
            "weight_kg": 70.0,
            "height_cm": 170.0,
            "sex": "male",
            "egfr": 90.0
        },
        "processing_hints": {},
        "priority": "normal",
        "enable_ml_inference": false,
        "timestamp": "2024-01-15T10:30:00Z"
    });
    
    let request = Request::builder()
        .method("POST")
        .uri("/api/flow2/execute")
        .header("content-type", "application/json")
        .body(Body::from(request_body.to_string()))
        .unwrap();
    
    let response = app.oneshot(request).await.unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    
    let body = axum::body::to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let response_json: Value = serde_json::from_slice(&body).unwrap();
    
    assert_eq!(response_json["request_id"], "test-flow2-123");
    assert_eq!(response_json["overall_status"], "success");
    assert!(response_json["execution_time_ms"].as_u64().unwrap() > 0);
}

#[tokio::test]
async fn test_health_endpoint() {
    let engine = create_test_engine().await;
    let config = ServerConfig::default();
    let app = create_router(engine, config);
    
    let request = Request::builder()
        .method("GET")
        .uri("/health")
        .body(Body::empty())
        .unwrap();
    
    let response = app.oneshot(request).await.unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    
    let body = axum::body::to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let response_json: Value = serde_json::from_slice(&body).unwrap();
    
    assert_eq!(response_json["status"], "healthy");
    assert_eq!(response_json["engine"], "unified-clinical-engine");
}

#[tokio::test]
async fn test_concurrent_requests() {
    let engine = create_test_engine().await;
    
    // Test concurrent processing
    let mut handles = Vec::new();
    
    for _i in 0..10 {
        let engine = engine.clone();
        let handle = tokio::spawn(async move {
            let request = create_test_clinical_request("metformin");
            engine.process_clinical_request(request).await
        });
        handles.push(handle);
    }
    
    // Wait for all requests to complete
    for handle in handles {
        let result = handle.await.unwrap();
        assert!(result.is_ok());
    }
}

#[tokio::test]
async fn test_performance_requirements() {
    let engine = create_test_engine().await;
    let request = create_test_clinical_request("metformin");
    
    let start = std::time::Instant::now();
    let result = engine.process_clinical_request(request).await;
    let duration = start.elapsed();
    
    assert!(result.is_ok());
    // Verify sub-100ms performance requirement
    assert!(duration.as_millis() < 100, "Processing took {}ms, should be < 100ms", duration.as_millis());
}
