//! Simplified Test Harness for Unified Dose Calculation Engine
//! 
//! Basic test scenarios to validate core functionality

use std::collections::HashMap;
use std::time::Instant;
use anyhow::Result;

use crate::unified_clinical_engine::{
    UnifiedClinicalEngine, ClinicalRequest, PatientContext,
    RenalFunction, HepaticFunction, BiologicalSex, PregnancyStatus,
};

/// Simple test runner for basic validation
pub struct SimpleTestRunner {
    engine: UnifiedClinicalEngine,
}

impl SimpleTestRunner {
    pub async fn new() -> Result<Self> {
        let knowledge_base = std::sync::Arc::new(
            crate::unified_clinical_engine::KnowledgeBase::new().await?
        );
        let engine = UnifiedClinicalEngine::new(knowledge_base)?;
        
        Ok(Self { engine })
    }
    
    /// Create a basic healthy adult patient
    fn create_healthy_patient() -> PatientContext {
        PatientContext {
            age_years: 45,
            weight_kg: 70.0,
            height_cm: 170.0,
            sex: BiologicalSex::Male,
            pregnancy_status: PregnancyStatus::NotApplicable,
            renal_function: RenalFunction {
                egfr_ml_min_1_73m2: Some(90.0),
                serum_creatinine_mg_dl: Some(1.0),
                dialysis_status: None,
            },
            hepatic_function: HepaticFunction {
                child_pugh_class: None,
                ast_iu_l: Some(25.0),
                alt_iu_l: Some(30.0),
                total_bilirubin_mg_dl: Some(0.8),
            },
            active_medications: vec![],
            allergies: vec![],
            conditions: vec![],
            lab_values: HashMap::new(),
        }
    }
    
    /// Create a basic clinical request
    fn create_request(drug_id: &str, indication: &str) -> ClinicalRequest {
        ClinicalRequest {
            request_id: format!("TEST_{}", drug_id.to_uppercase()),
            drug_id: drug_id.to_string(),
            indication: indication.to_string(),
            patient_context: Self::create_healthy_patient(),
            timestamp: chrono::Utc::now(),
        }
    }
    
    /// Test basic engine initialization and response
    pub async fn test_basic_functionality(&self) -> Result<()> {
        println!("🧪 Testing basic engine functionality...");
        
        let start = Instant::now();
        
        // Test 1: Basic lisinopril calculation
        let request = Self::create_request("lisinopril", "hypertension");
        
        match self.engine.process_clinical_request(request).await {
            Ok(response) => {
                println!("✅ Basic calculation successful:");
                println!("   - Drug: {}", response.drug_id);
                println!("   - Proposed dose: {:.1} mg", response.calculation_result.proposed_dose_mg);
                println!("   - Final dose: {:.1} mg", response.final_recommendation.final_dose_mg);
                println!("   - Processing time: {} ms", response.processing_time_ms);
                println!("   - Safety findings: {}", response.safety_result.findings.len());
            },
            Err(e) => {
                println!("❌ Basic calculation failed: {}", e);
                return Err(e);
            }
        }
        
        let elapsed = start.elapsed();
        println!("⏱️  Total test time: {:.2}ms", elapsed.as_millis());
        
        Ok(())
    }
    
    /// Test multiple drug calculations
    pub async fn test_multiple_drugs(&self) -> Result<()> {
        println!("\n🧪 Testing multiple drug calculations...");
        
        let drugs = vec![
            ("lisinopril", "hypertension"),
            ("metformin", "diabetes"),
            ("amoxicillin", "infection"),
        ];
        
        let mut successful = 0;
        let mut total = 0;
        
        for (drug, indication) in drugs {
            total += 1;
            let request = Self::create_request(drug, indication);
            
            match self.engine.process_clinical_request(request).await {
                Ok(response) => {
                    successful += 1;
                    println!("✅ {}: {:.1} mg", drug, response.calculation_result.proposed_dose_mg);
                },
                Err(e) => {
                    println!("❌ {}: {}", drug, e);
                }
            }
        }
        
        println!("📊 Success rate: {}/{} ({:.1}%)", 
            successful, total, (successful as f64 / total as f64) * 100.0);
        
        if successful == 0 {
            return Err(anyhow::anyhow!("No drug calculations succeeded"));
        }
        
        Ok(())
    }
    
    /// Test performance with multiple requests
    pub async fn test_performance(&self) -> Result<()> {
        println!("\n🧪 Testing performance with multiple requests...");
        
        let start = Instant::now();
        let mut successful = 0;
        let num_requests = 10;
        
        for i in 0..num_requests {
            let request = ClinicalRequest {
                request_id: format!("PERF_{:03}", i),
                drug_id: "lisinopril".to_string(),
                indication: "hypertension".to_string(),
                patient_context: Self::create_healthy_patient(),
                timestamp: chrono::Utc::now(),
            };
            
            if self.engine.process_clinical_request(request).await.is_ok() {
                successful += 1;
            }
        }
        
        let elapsed = start.elapsed();
        let avg_time = elapsed.as_millis() as f64 / num_requests as f64;
        
        println!("📊 Performance results:");
        println!("   - Total requests: {}", num_requests);
        println!("   - Successful: {}", successful);
        println!("   - Total time: {:.2}ms", elapsed.as_millis());
        println!("   - Average time: {:.2}ms per request", avg_time);
        
        if successful < num_requests / 2 {
            return Err(anyhow::anyhow!("Too many requests failed: {}/{}", successful, num_requests));
        }
        
        if avg_time > 100.0 {
            return Err(anyhow::anyhow!("Average response time {:.2}ms exceeds 100ms target", avg_time));
        }
        
        Ok(())
    }
    
    /// Run all basic tests
    pub async fn run_all_tests(&self) -> Result<()> {
        println!("🚀 Starting Unified Dose Calculation Engine - Basic Test Suite");
        println!("{}", "=".repeat(70));
        
        let overall_start = Instant::now();
        let mut tests_passed = 0;
        let mut total_tests = 0;
        
        // Test 1: Basic functionality
        total_tests += 1;
        if self.test_basic_functionality().await.is_ok() {
            tests_passed += 1;
        }
        
        // Test 2: Multiple drugs
        total_tests += 1;
        if self.test_multiple_drugs().await.is_ok() {
            tests_passed += 1;
        }
        
        // Test 3: Performance
        total_tests += 1;
        if self.test_performance().await.is_ok() {
            tests_passed += 1;
        }
        
        let overall_elapsed = overall_start.elapsed();
        
        println!("\n{}", "=".repeat(70));
        println!("📊 FINAL TEST RESULTS");
        println!("{}", "=".repeat(70));
        println!("✅ Tests passed: {}/{}", tests_passed, total_tests);
        println!("❌ Tests failed: {}/{}", total_tests - tests_passed, total_tests);
        println!("⏱️  Total time: {:.2}s", overall_elapsed.as_secs_f64());
        
        let success_rate = (tests_passed as f64 / total_tests as f64) * 100.0;
        println!("📈 Success rate: {:.1}%", success_rate);
        
        if tests_passed == total_tests {
            println!("\n🎉 ALL TESTS PASSED! Basic engine functionality is working correctly.");
            Ok(())
        } else {
            println!("\n⚠️  Some tests failed. Please review the output above.");
            Err(anyhow::anyhow!("Test suite failed: {}/{} tests passed", tests_passed, total_tests))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_simple_runner_creation() {
        let runner = SimpleTestRunner::new().await;
        assert!(runner.is_ok(), "Simple test runner should initialize successfully");
    }
    
    #[tokio::test]
    async fn test_patient_creation() {
        let patient = SimpleTestRunner::create_healthy_patient();
        assert_eq!(patient.age_years, 45);
        assert_eq!(patient.weight_kg, 70.0);
        assert!(matches!(patient.sex, BiologicalSex::Male));
    }
    
    #[tokio::test]
    async fn test_request_creation() {
        let request = SimpleTestRunner::create_request("lisinopril", "hypertension");
        assert_eq!(request.drug_id, "lisinopril");
        assert_eq!(request.indication, "hypertension");
        assert_eq!(request.request_id, "TEST_LISINOPRIL");
    }
}
