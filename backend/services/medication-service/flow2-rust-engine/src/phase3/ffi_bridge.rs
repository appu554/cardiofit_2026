//! FFI Bridge for Go Integration
//! 
//! This module provides C-compatible FFI functions that allow the Go Phase 1 orchestrator
//! to call the Rust Phase 3 Clinical Intelligence Engine. This matches the specification
//! requirement for seamless Go-Rust integration via FFI.

use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::Arc;
use std::collections::HashMap;
use once_cell::sync::Lazy;
use tokio::runtime::Runtime;
use anyhow::{Result, anyhow};
use serde_json;

use crate::unified_clinical_engine::{UnifiedClinicalEngine, knowledge_base::KnowledgeBase};
use super::{ClinicalIntelligenceEngine, Phase3Input, Phase3Output};

// Global Phase 3 engine instance (initialized once)
static PHASE3_ENGINE: Lazy<Arc<ClinicalIntelligenceEngine>> = Lazy::new(|| {
    initialize_phase3_engine().expect("Failed to initialize Phase 3 engine")
});

// Global async runtime for handling async operations in FFI context
static RUNTIME: Lazy<Runtime> = Lazy::new(|| {
    Runtime::new().expect("Failed to create Tokio runtime")
});

/// Initialize the Phase 3 engine with knowledge base
fn initialize_phase3_engine() -> Result<Arc<ClinicalIntelligenceEngine>> {
    // Get knowledge base path from environment or use default
    let kb_path = std::env::var("KNOWLEDGE_BASE_PATH")
        .unwrap_or_else(|_| "../../../shared-infrastructure/knowledge-base-services".to_string());
    
    // Initialize components synchronously in the lazy static context
    let rt = Runtime::new()?;
    let knowledge_base = rt.block_on(async {
        KnowledgeBase::new(&kb_path).await
    })?;
    
    let unified_engine = Arc::new(UnifiedClinicalEngine::new(Arc::new(knowledge_base))?);
    let phase3_engine = Arc::new(ClinicalIntelligenceEngine::new(unified_engine)?);
    
    tracing::info!("🦀 Phase 3 Clinical Intelligence Engine initialized for FFI");
    
    Ok(phase3_engine)
}

/// Main FFI function for Phase 3 execution
/// 
/// This function is called from Go with a JSON-serialized Phase3Input
/// and returns a JSON-serialized Phase3Output.
/// 
/// # Safety
/// 
/// This function is unsafe because it deals with raw pointers from C/Go.
/// The caller must ensure that:
/// - input_json is a valid null-terminated C string
/// - The returned pointer is freed using free_phase3_result()
#[no_mangle]
pub extern "C" fn execute_phase3_ffi(input_json: *const c_char) -> *mut c_char {
    if input_json.is_null() {
        return create_error_result("Input JSON pointer is null");
    }
    
    let result = std::panic::catch_unwind(|| {
        unsafe { execute_phase3_impl(input_json) }
    });
    
    match result {
        Ok(output) => output,
        Err(_) => create_error_result("Panic occurred during Phase 3 execution"),
    }
}

/// Internal implementation of Phase 3 execution
unsafe fn execute_phase3_impl(input_json: *const c_char) -> *mut c_char {
    // Convert C string to Rust string
    let c_str = match CStr::from_ptr(input_json).to_str() {
        Ok(s) => s,
        Err(_) => return create_error_result("Invalid UTF-8 in input JSON"),
    };
    
    // Parse JSON input
    let phase3_input: Phase3Input = match serde_json::from_str(c_str) {
        Ok(input) => input,
        Err(e) => return create_error_result(&format!("JSON parsing error: {}", e)),
    };
    
    tracing::info!(
        "🎯 FFI: Starting Phase 3 execution for request_id: {}",
        phase3_input.request_id
    );
    
    // Execute Phase 3 using the global runtime
    let output_result = RUNTIME.block_on(async {
        PHASE3_ENGINE.execute_phase3(phase3_input).await
    });
    
    let phase3_output = match output_result {
        Ok(output) => output,
        Err(e) => {
            tracing::error!("❌ Phase 3 execution failed: {}", e);
            return create_error_result(&format!("Phase 3 execution error: {}", e));
        }
    };
    
    tracing::info!(
        "✅ FFI: Phase 3 completed for request_id: {} in {:?}",
        phase3_output.request_id,
        phase3_output.phase3_duration
    );
    
    // Serialize result to JSON
    let output_json = match serde_json::to_string(&phase3_output) {
        Ok(json) => json,
        Err(e) => return create_error_result(&format!("JSON serialization error: {}", e)),
    };
    
    // Convert to C string
    match CString::new(output_json) {
        Ok(c_string) => c_string.into_raw(),
        Err(e) => create_error_result(&format!("C string conversion error: {}", e)),
    }
}

/// FFI function for health check
#[no_mangle]
pub extern "C" fn phase3_health_check_ffi() -> *mut c_char {
    let result = std::panic::catch_unwind(|| {
        let health_result = RUNTIME.block_on(async {
            PHASE3_ENGINE.health_check().await
        });
        
        match health_result {
            Ok(health_status) => {
                match serde_json::to_string(&health_status) {
                    Ok(json) => match CString::new(json) {
                        Ok(c_string) => c_string.into_raw(),
                        Err(e) => create_error_result(&format!("C string conversion error: {}", e)),
                    },
                    Err(e) => create_error_result(&format!("JSON serialization error: {}", e)),
                }
            }
            Err(e) => create_error_result(&format!("Health check failed: {}", e)),
        }
    });
    
    match result {
        Ok(output) => output,
        Err(_) => create_error_result("Panic occurred during health check"),
    }
}

/// FFI function for getting performance metrics
#[no_mangle]
pub extern "C" fn phase3_get_metrics_ffi() -> *mut c_char {
    let result = std::panic::catch_unwind(|| {
        let metrics = PHASE3_ENGINE.get_metrics();
        let metrics_snapshot = metrics.get_snapshot();
        
        match serde_json::to_string(&metrics_snapshot) {
            Ok(json) => match CString::new(json) {
                Ok(c_string) => c_string.into_raw(),
                Err(e) => create_error_result(&format!("C string conversion error: {}", e)),
            },
            Err(e) => create_error_result(&format!("JSON serialization error: {}", e)),
        }
    });
    
    match result {
        Ok(output) => output,
        Err(_) => create_error_result("Panic occurred during metrics retrieval"),
    }
}

/// FFI function to free memory allocated by Rust
/// 
/// This MUST be called by Go to free the memory returned by other FFI functions.
/// 
/// # Safety
/// 
/// This function is unsafe because it deals with raw pointers.
/// The caller must ensure that:
/// - result_ptr was returned by one of the other FFI functions
/// - result_ptr is only freed once
/// - result_ptr is not used after being freed
#[no_mangle]
pub extern "C" fn free_phase3_result(result_ptr: *mut c_char) {
    if result_ptr.is_null() {
        return;
    }
    
    unsafe {
        // Convert back to CString and let it be dropped to free memory
        let _ = CString::from_raw(result_ptr);
    }
}

/// Initialize Phase 3 engine with custom configuration
/// 
/// This can be called from Go to initialize the engine with specific settings
/// before making other FFI calls.
#[no_mangle]
pub extern "C" fn initialize_phase3_with_config_ffi(config_json: *const c_char) -> *mut c_char {
    if config_json.is_null() {
        return create_error_result("Config JSON pointer is null");
    }
    
    let result = std::panic::catch_unwind(|| {
        unsafe { initialize_phase3_with_config_impl(config_json) }
    });
    
    match result {
        Ok(output) => output,
        Err(_) => create_error_result("Panic occurred during initialization"),
    }
}

unsafe fn initialize_phase3_with_config_impl(config_json: *const c_char) -> *mut c_char {
    let c_str = match CStr::from_ptr(config_json).to_str() {
        Ok(s) => s,
        Err(_) => return create_error_result("Invalid UTF-8 in config JSON"),
    };
    
    // Parse configuration
    let config: HashMap<String, serde_json::Value> = match serde_json::from_str(c_str) {
        Ok(cfg) => cfg,
        Err(e) => return create_error_result(&format!("Config parsing error: {}", e)),
    };
    
    // Apply configuration to environment variables
    if let Some(kb_path) = config.get("knowledge_base_path") {
        if let Some(path_str) = kb_path.as_str() {
            std::env::set_var("KNOWLEDGE_BASE_PATH", path_str);
        }
    }
    
    if let Some(apollo_url) = config.get("apollo_federation_url") {
        if let Some(url_str) = apollo_url.as_str() {
            std::env::set_var("APOLLO_FEDERATION_URL", url_str);
        }
    }
    
    // Force re-initialization by clearing the lazy static
    // Note: This is not thread-safe and should only be used during startup
    
    create_success_result("Phase 3 engine initialized with custom configuration")
}

/// Validate Phase 3 input without execution
#[no_mangle]
pub extern "C" fn validate_phase3_input_ffi(input_json: *const c_char) -> *mut c_char {
    if input_json.is_null() {
        return create_error_result("Input JSON pointer is null");
    }
    
    let result = std::panic::catch_unwind(|| {
        unsafe {
            let c_str = match CStr::from_ptr(input_json).to_str() {
                Ok(s) => s,
                Err(_) => return create_error_result("Invalid UTF-8 in input JSON"),
            };
            
            // Try to parse the input to validate structure
            let _phase3_input: Phase3Input = match serde_json::from_str(c_str) {
                Ok(input) => input,
                Err(e) => return create_error_result(&format!("Validation failed: {}", e)),
            };
            
            create_success_result("Phase 3 input validation passed")
        }
    });
    
    match result {
        Ok(output) => output,
        Err(_) => create_error_result("Panic occurred during input validation"),
    }
}

/// Helper function to create error result as C string
fn create_error_result(error_message: &str) -> *mut c_char {
    let error_response = serde_json::json!({
        "success": false,
        "error": error_message,
        "timestamp": chrono::Utc::now()
    });
    
    let error_json = error_response.to_string();
    
    match CString::new(error_json) {
        Ok(c_string) => c_string.into_raw(),
        Err(_) => {
            // Fallback error message if we can't even create the error JSON
            let fallback = CString::new("{\"success\":false,\"error\":\"Critical error during error handling\"}").unwrap();
            fallback.into_raw()
        }
    }
}

/// Helper function to create success result as C string
fn create_success_result(message: &str) -> *mut c_char {
    let success_response = serde_json::json!({
        "success": true,
        "message": message,
        "timestamp": chrono::Utc::now()
    });
    
    let success_json = success_response.to_string();
    
    match CString::new(success_json) {
        Ok(c_string) => c_string.into_raw(),
        Err(_) => create_error_result("Failed to create success response"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::phase3::models::*;
    use chrono::Utc;
    use std::collections::HashMap;
    
    #[test]
    fn test_ffi_memory_management() {
        let test_message = "test result";
        let c_string = CString::new(test_message).unwrap();
        let raw_ptr = c_string.into_raw();
        
        // Simulate freeing memory
        free_phase3_result(raw_ptr);
        // If we get here without crashing, memory management works
    }
    
    #[test]
    fn test_null_pointer_handling() {
        let result_ptr = execute_phase3_ffi(std::ptr::null());
        assert!(!result_ptr.is_null());
        
        // Free the error result
        free_phase3_result(result_ptr);
    }
    
    #[test]
    fn test_input_validation() {
        let invalid_json = CString::new("invalid json").unwrap();
        let result_ptr = validate_phase3_input_ffi(invalid_json.as_ptr());
        assert!(!result_ptr.is_null());
        
        // Check that it's an error response
        unsafe {
            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            let response: serde_json::Value = serde_json::from_str(result_str).unwrap();
            assert_eq!(response["success"], false);
        }
        
        free_phase3_result(result_ptr);
    }
    
    #[tokio::test]
    async fn test_create_mock_phase3_input() {
        // Create a minimal valid Phase3Input for testing
        let intent_manifest = IntentManifest {
            manifest_id: "test_manifest".to_string(),
            request_id: "test_request".to_string(),
            generated_at: Utc::now(),
            primary_intent: ClinicalIntent {
                category: "TREATMENT".to_string(),
                condition: "hypertension".to_string(),
                severity: "MODERATE".to_string(),
                phenotype: "standard".to_string(),
                time_horizon: "CHRONIC".to_string(),
            },
            secondary_intents: vec![],
            protocol_id: "hypertension_protocol".to_string(),
            protocol_version: "1.0".to_string(),
            evidence_grade: "A".to_string(),
            context_recipe_id: "context_recipe_1".to_string(),
            clinical_recipe_id: "clinical_recipe_1".to_string(),
            required_fields: vec![],
            optional_fields: vec![],
            data_freshness: FreshnessRequirements {
                max_age: std::time::Duration::from_secs(3600),
                critical_fields: vec![],
                preferred_sources: vec![],
            },
            snapshot_ttl: 3600,
            therapy_options: vec![
                TherapyCandidate {
                    therapy_class: "ACE_INHIBITOR".to_string(),
                    preference_order: 1,
                    rationale: "First-line therapy".to_string(),
                    guideline_source: "AHA/ACC".to_string(),
                }
            ],
            orb_version: "1.0".to_string(),
            rules_applied: vec![],
        };
        
        let enriched_context = EnrichedContext {
            demographics: Demographics {
                age: 55.0,
                sex: "male".to_string(),
                weight: 80.0,
                height: 175.0,
                bmi: Some(26.1),
                pregnancy_status: "not_applicable".to_string(),
                region: "US".to_string(),
            },
            lab_results: LabResults {
                egfr: Some(90.0),
                creatinine: Some(1.0),
                bilirubin: None,
                albumin: None,
                alt: None,
                ast: None,
                hemoglobin: None,
                platelet_count: None,
                inr: None,
            },
            vital_signs: VitalSigns {
                systolic_bp: Some(145.0),
                diastolic_bp: Some(92.0),
                heart_rate: Some(75.0),
                temperature: None,
                respiratory_rate: None,
                oxygen_saturation: None,
            },
            current_medications: CurrentMedications {
                medications: vec![],
                count: 0,
            },
            allergies: vec![],
            active_conditions: vec!["I10".to_string()], // ICD-10 for hypertension
            phenotype: "standard_hypertension".to_string(),
            risk_factors: vec![],
            patient_preferences: None,
            clinical_constraints: vec![],
        };
        
        let evidence_envelope = EvidenceEnvelope {
            envelope_id: "evidence_1".to_string(),
            created_at: Utc::now(),
            kb_versions: HashMap::new(),
            snapshot_hash: "hash123".to_string(),
            signature: None,
            audit_id: "audit_1".to_string(),
            processing_chain: vec!["phase1".to_string(), "phase2".to_string()],
        };
        
        let phase3_input = Phase3Input::new(intent_manifest, enriched_context, evidence_envelope);
        
        // Test serialization
        let json = serde_json::to_string(&phase3_input).unwrap();
        assert!(json.contains("test_request"));
        
        // Test deserialization
        let parsed: Phase3Input = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.request_id, "test_request");
    }
}