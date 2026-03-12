// Safety Engines - Hybrid Protocol Engine and CAE implementation
//
// This library provides high-performance clinical safety evaluation engines
// designed to replace Python subprocess calls with native Rust implementations.
// 
// Architecture:
// - FFI interface for Go integration  
// - Clinical Assertion Engine (CAE) for safety evaluation
// - Protocol Engine for clinical pathway enforcement
// - FHIR-compatible data structures
// - Memory-safe clinical algorithms
// - Snapshot-driven deterministic evaluation
// - Comprehensive testing suite with chaos engineering capabilities

pub mod cae;
pub mod ffi;
pub mod types;
pub mod utils;

// Protocol Engine modules
#[cfg(feature = "protocol_engine")]
pub mod protocol;

// Comprehensive testing framework
#[cfg(any(test, feature = "testing"))]
pub mod testing;

// Re-export main types for convenience
pub use cae::{CAEEngine, CAEConfig, CAEError};
pub use types::{SafetyRequest, SafetyResult, SafetyStatus};

#[cfg(feature = "protocol_engine")]
pub use protocol::{
    ProtocolEngine, ProtocolEngineConfig, ProtocolEngineError,
    ProtocolEvaluationRequest, ProtocolEvaluationResult, ProtocolDecisionType,
};

// Re-export testing framework for external use
#[cfg(any(test, feature = "testing"))]
pub use testing::{
    ClinicalTestingSuite, ClinicalTestingSuiteConfig,
    TestResult, TestStatus, TestPriority, ClinicalSafetyClass, ClinicalTestContext,
};

pub use ffi::*;

use once_cell::sync::Lazy;
use std::sync::Mutex;

// Global engine instance (thread-safe)
static GLOBAL_CAE_ENGINE: Lazy<Mutex<Option<CAEEngine>>> = Lazy::new(|| Mutex::new(None));

/// Initialize the global CAE engine instance
pub fn initialize_global_engine(config: CAEConfig) -> Result<(), CAEError> {
    let engine = CAEEngine::new(config)?;
    let mut global_engine = GLOBAL_CAE_ENGINE.lock().unwrap();
    *global_engine = Some(engine);
    Ok(())
}

/// Get a reference to the global CAE engine
pub fn with_global_engine<F, R>(f: F) -> Result<R, CAEError>
where
    F: FnOnce(&CAEEngine) -> Result<R, CAEError>,
{
    let global_engine = GLOBAL_CAE_ENGINE.lock().unwrap();
    match global_engine.as_ref() {
        Some(engine) => f(engine),
        None => Err(CAEError::EngineNotInitialized),
    }
}

/// Shutdown the global CAE engine
pub fn shutdown_global_engine() {
    let mut global_engine = GLOBAL_CAE_ENGINE.lock().unwrap();
    *global_engine = None;
}

/// Initialize and run the clinical testing suite
#[cfg(any(test, feature = "testing"))]
pub async fn run_clinical_testing_suite() -> anyhow::Result<testing::reporting::TestSuiteResults> {
    let mut suite = testing::ClinicalTestingSuite::new();
    suite.run_full_suite().await
}

/// Initialize clinical testing suite with custom configuration
#[cfg(any(test, feature = "testing"))]
pub async fn run_clinical_testing_suite_with_config(
    config: testing::ClinicalTestingSuiteConfig
) -> anyhow::Result<testing::reporting::TestSuiteResults> {
    let mut suite = testing::ClinicalTestingSuite::with_config(config);
    suite.run_full_suite().await
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::*;

    #[test]
    fn test_global_engine_lifecycle() {
        // Test initialization
        let config = CAEConfig::test_config();
        assert!(initialize_global_engine(config).is_ok());

        // Test usage
        let request = SafetyRequest {
            patient_id: "test-patient".to_string(),
            request_id: "test-001".to_string(),
            medication_ids: vec!["aspirin".to_string()],
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };

        let result = with_global_engine(|engine| engine.evaluate_safety(&request));
        assert!(result.is_ok());

        // Test shutdown
        shutdown_global_engine();

        // Engine should not be available after shutdown
        let result = with_global_engine(|engine| engine.evaluate_safety(&request));
        assert!(result.is_err());
    }

    #[cfg(feature = "testing")]
    #[tokio::test]
    async fn test_clinical_testing_suite_integration() {
        use crate::testing::*;
        
        // Create a minimal test configuration
        let mut config = ClinicalTestingSuiteConfig::default();
        config.chaos_engineering.enabled = false; // Disable chaos for basic test
        config.load_testing.enabled = false;      // Disable load testing
        config.security_testing.enable_penetration_testing = false; // Disable pen testing
        config.clinical_scenarios.enable_emergency_protocol_testing = false; // Disable emergency testing
        
        let mut suite = ClinicalTestingSuite::with_config(config);
        
        // Run a basic test suite
        let results = suite.run_full_suite().await;
        assert!(results.is_ok(), "Clinical testing suite should complete successfully");
        
        let test_results = results.unwrap();
        assert!(test_results.summary_statistics.total_tests > 0, "Should have executed some tests");
        
        // Verify no critical safety failures
        let critical_failures = test_results.test_results
            .iter()
            .filter(|r| r.priority == TestPriority::Critical && r.status == TestStatus::Failed)
            .count();
        assert_eq!(critical_failures, 0, "No critical tests should fail");
    }

    #[cfg(feature = "testing")]
    #[tokio::test]
    async fn test_security_testing_integration() {
        use crate::testing::*;
        
        let mut config = ClinicalTestingSuiteConfig::default();
        // Enable only security testing for this test
        config.chaos_engineering.enabled = false;
        config.load_testing.enabled = false;
        config.clinical_scenarios.enabled = false;
        config.integration_testing.enabled = false;
        config.performance_testing.enabled = false;
        config.security_testing.enabled = true;
        config.compliance_testing.enabled = false;
        
        let mut suite = ClinicalTestingSuite::with_config(config);
        let results = suite.run_full_suite().await;
        
        assert!(results.is_ok(), "Security testing should complete successfully");
        
        let test_results = results.unwrap();
        let security_tests = test_results.test_results
            .iter()
            .filter(|r| r.test_category == "security_testing" || r.test_category == "hipaa_compliance")
            .count();
        
        assert!(security_tests > 0, "Should have executed security tests");
    }

    #[cfg(feature = "testing")]
    #[test]
    fn test_clinical_test_context_creation() {
        use crate::testing::*;
        
        let context = ClinicalTestContext {
            patient_count: 100,
            provider_count: 25,
            clinical_scenarios: vec!["medication_safety".to_string(), "patient_identification".to_string()],
            fhir_resources_tested: vec!["Patient".to_string(), "Medication".to_string()],
            clinical_protocols_validated: vec!["drug_interaction_check".to_string()],
            safety_checks_performed: vec!["allergy_screening".to_string()],
            hipaa_safeguards: vec!["data_encryption".to_string(), "access_control".to_string()],
        };
        
        assert_eq!(context.patient_count, 100);
        assert_eq!(context.provider_count, 25);
        assert_eq!(context.clinical_scenarios.len(), 2);
        assert!(context.clinical_scenarios.contains(&"medication_safety".to_string()));
    }

    #[cfg(feature = "testing")]
    #[test]
    fn test_test_priority_and_safety_classification() {
        use crate::testing::*;
        
        // Test priority levels
        assert!(TestPriority::Critical > TestPriority::High);
        assert!(TestPriority::High > TestPriority::Medium);
        assert!(TestPriority::Medium > TestPriority::Low);
        
        // Test safety classifications
        let patient_safety = ClinicalSafetyClass::PatientSafetyCritical;
        let clinical_workflow = ClinicalSafetyClass::ClinicalWorkflow;
        let data_integrity = ClinicalSafetyClass::DataIntegrity;
        let performance = ClinicalSafetyClass::Performance;
        let non_clinical = ClinicalSafetyClass::NonClinical;
        
        // All should be valid enum variants
        assert_eq!(format!("{:?}", patient_safety), "PatientSafetyCritical");
        assert_eq!(format!("{:?}", clinical_workflow), "ClinicalWorkflow");
        assert_eq!(format!("{:?}", data_integrity), "DataIntegrity");
        assert_eq!(format!("{:?}", performance), "Performance");
        assert_eq!(format!("{:?}", non_clinical), "NonClinical");
    }
}