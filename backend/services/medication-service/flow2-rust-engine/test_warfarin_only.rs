use flow2_rust_engine::unified_clinical_engine::UnifiedClinicalEngine;
use flow2_rust_engine::models::{ClinicalRequest, PatientContext, ActiveMedication};
use chrono::Utc;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Testing warfarin parsing only...");
    
    // Initialize engine
    let engine = UnifiedClinicalEngine::new("knowledge").await?;
    println!("Engine initialized successfully");
    
    // Create simple patient
    let mut patient = PatientContext {
        patient_id: "test-patient".to_string(),
        age_years: 65,
        weight_kg: 70.0,
        height_cm: 170.0,
        biological_sex: flow2_rust_engine::models::BiologicalSex::Male,
        pregnancy_status: flow2_rust_engine::models::PregnancyStatus::NotPregnant,
        renal_function: flow2_rust_engine::models::RenalFunction::Normal,
        hepatic_function: flow2_rust_engine::models::HepaticFunction::Normal,
        allergies: vec![],
        active_medications: vec![
            ActiveMedication {
                drug_id: "amiodarone".to_string(),
                dose_mg: 200.0,
                frequency: "daily".to_string(),
                route: "oral".to_string(),
                start_date: Utc::now(),
            }
        ],
        lab_values: vec![],
        conditions: vec![],
    };

    let request = ClinicalRequest {
        request_id: "test-warfarin-001".to_string(),
        patient_context: patient,
        drug_id: "warfarin".to_string(),
        indication: "atrial_fibrillation".to_string(),
        timestamp: Utc::now(),
    };

    println!("Calling engine with warfarin request...");
    
    match engine.process_clinical_request(request).await {
        Ok(response) => {
            println!("SUCCESS! Response: {:?}", response.safety_result.action);
        }
        Err(e) => {
            println!("ERROR: {}", e);
        }
    }
    
    Ok(())
}
