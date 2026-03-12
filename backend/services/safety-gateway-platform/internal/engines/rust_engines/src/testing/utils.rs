//! # Testing Utilities
//!
//! This module provides utility functions and helpers for the clinical testing framework,
//! including test data generation, assertion helpers, timing utilities, and clinical
//! data validation functions.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use chrono::{DateTime, Utc};
use uuid::Uuid;
use anyhow::Result;
use std::sync::Arc;
use tokio::sync::RwLock;

use super::{TestResult, TestStatus, TestPriority, ClinicalSafetyClass, ClinicalTestContext};

/// Test data generator for creating synthetic clinical data
pub struct TestDataGenerator {
    seed: u64,
    patient_id_counter: Arc<RwLock<u64>>,
    encounter_id_counter: Arc<RwLock<u64>>,
}

impl TestDataGenerator {
    pub fn new() -> Self {
        Self {
            seed: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
            patient_id_counter: Arc::new(RwLock::new(1000)),
            encounter_id_counter: Arc::new(RwLock::new(2000)),
        }
    }

    pub fn with_seed(seed: u64) -> Self {
        Self {
            seed,
            patient_id_counter: Arc::new(RwLock::new(1000)),
            encounter_id_counter: Arc::new(RwLock::new(2000)),
        }
    }

    /// Generate synthetic patient data
    pub async fn generate_test_patient(&self) -> TestPatient {
        let mut counter = self.patient_id_counter.write().await;
        *counter += 1;
        let patient_id = *counter;

        TestPatient {
            id: format!("TEST-PAT-{:06}", patient_id),
            mrn: format!("MRN{:08}", patient_id * 7 % 99999999),
            first_name: self.generate_first_name(patient_id),
            last_name: self.generate_last_name(patient_id),
            date_of_birth: self.generate_birth_date(patient_id),
            gender: self.generate_gender(patient_id),
            allergies: self.generate_allergies(patient_id),
            medications: self.generate_medications(patient_id),
            conditions: self.generate_conditions(patient_id),
            vital_signs: self.generate_vital_signs(patient_id),
        }
    }

    /// Generate test encounter data
    pub async fn generate_test_encounter(&self, patient_id: &str) -> TestEncounter {
        let mut counter = self.encounter_id_counter.write().await;
        *counter += 1;
        let encounter_id = *counter;

        TestEncounter {
            id: format!("TEST-ENC-{:06}", encounter_id),
            patient_id: patient_id.to_string(),
            encounter_type: self.generate_encounter_type(encounter_id),
            status: "active".to_string(),
            start_time: Utc::now(),
            end_time: None,
            provider: self.generate_provider(encounter_id),
            location: self.generate_location(encounter_id),
            diagnosis: self.generate_diagnosis(encounter_id),
        }
    }

    /// Generate test medication data
    pub fn generate_test_medication(&self, seed: u64) -> TestMedication {
        let medications = [
            ("Aspirin", "81mg", "daily"),
            ("Metformin", "500mg", "twice daily"),
            ("Lisinopril", "10mg", "daily"),
            ("Atorvastatin", "20mg", "daily"),
            ("Metoprolol", "25mg", "twice daily"),
            ("Omeprazole", "20mg", "daily"),
            ("Amlodipine", "5mg", "daily"),
            ("Hydrochlorothiazide", "25mg", "daily"),
        ];

        let med_idx = (seed as usize) % medications.len();
        let (name, dose, frequency) = medications[med_idx];

        TestMedication {
            id: format!("TEST-MED-{:06}", seed),
            name: name.to_string(),
            dose: dose.to_string(),
            frequency: frequency.to_string(),
            route: "oral".to_string(),
            prescriber: self.generate_prescriber_name(seed),
            start_date: Utc::now(),
            end_date: None,
            instructions: format!("Take {} {}", dose, frequency),
        }
    }

    fn generate_first_name(&self, seed: u64) -> String {
        let names = [
            "John", "Jane", "Michael", "Sarah", "David", "Lisa", "Robert", "Maria",
            "James", "Jennifer", "William", "Patricia", "Richard", "Linda", "Joseph", "Elizabeth",
        ];
        names[(seed as usize) % names.len()].to_string()
    }

    fn generate_last_name(&self, seed: u64) -> String {
        let names = [
            "Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
            "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Taylor",
        ];
        names[(seed as usize) % names.len()].to_string()
    }

    fn generate_birth_date(&self, seed: u64) -> DateTime<Utc> {
        let years_ago = 20 + (seed % 60); // Age between 20-80
        Utc::now() - chrono::Duration::days((years_ago * 365) as i64)
    }

    fn generate_gender(&self, seed: u64) -> String {
        match seed % 3 {
            0 => "male".to_string(),
            1 => "female".to_string(),
            _ => "other".to_string(),
        }
    }

    fn generate_allergies(&self, seed: u64) -> Vec<TestAllergy> {
        if seed % 3 == 0 {
            return vec![]; // No allergies
        }

        let allergies = [
            ("Penicillin", "severe", "rash, difficulty breathing"),
            ("Shellfish", "moderate", "hives, swelling"),
            ("Latex", "mild", "skin irritation"),
            ("Aspirin", "moderate", "gastrointestinal upset"),
            ("Sulfa drugs", "severe", "severe skin reaction"),
        ];

        let allergy_idx = (seed as usize) % allergies.len();
        let (allergen, severity, reaction) = allergies[allergy_idx];

        vec![TestAllergy {
            allergen: allergen.to_string(),
            severity: severity.to_string(),
            reaction: reaction.to_string(),
            verified: true,
        }]
    }

    fn generate_medications(&self, seed: u64) -> Vec<TestMedication> {
        let med_count = 1 + (seed % 4) as usize; // 1-4 medications
        (0..med_count)
            .map(|i| self.generate_test_medication(seed + i as u64))
            .collect()
    }

    fn generate_conditions(&self, seed: u64) -> Vec<TestCondition> {
        let conditions = [
            ("Hypertension", "I10", "active"),
            ("Type 2 Diabetes", "E11.9", "active"),
            ("Hyperlipidemia", "E78.5", "active"),
            ("Osteoarthritis", "M19.90", "active"),
            ("GERD", "K21.9", "active"),
        ];

        let condition_count = 1 + (seed % 3) as usize; // 1-3 conditions
        (0..condition_count)
            .map(|i| {
                let cond_idx = ((seed + i as u64) as usize) % conditions.len();
                let (name, code, status) = conditions[cond_idx];
                TestCondition {
                    name: name.to_string(),
                    icd10_code: code.to_string(),
                    status: status.to_string(),
                    onset_date: Utc::now() - chrono::Duration::days((seed % 1000) as i64),
                }
            })
            .collect()
    }

    fn generate_vital_signs(&self, seed: u64) -> TestVitalSigns {
        TestVitalSigns {
            systolic_bp: 110 + (seed % 40) as u32,    // 110-150
            diastolic_bp: 70 + (seed % 20) as u32,    // 70-90
            heart_rate: 60 + (seed % 40) as u32,      // 60-100
            temperature: 98.0 + ((seed % 4) as f32 * 0.5), // 98.0-100.0
            respiratory_rate: 12 + (seed % 8) as u32, // 12-20
            oxygen_saturation: 95 + (seed % 6) as u32, // 95-100
            weight_kg: 50.0 + ((seed % 100) as f32),  // 50-150 kg
            height_cm: 150.0 + ((seed % 50) as f32),  // 150-200 cm
        }
    }

    fn generate_encounter_type(&self, seed: u64) -> String {
        let types = ["inpatient", "outpatient", "emergency", "urgent_care"];
        types[(seed as usize) % types.len()].to_string()
    }

    fn generate_provider(&self, seed: u64) -> TestProvider {
        let first_names = ["Dr. Alice", "Dr. Bob", "Dr. Carol", "Dr. David", "Dr. Eve"];
        let last_names = ["Anderson", "Brown", "Clark", "Davis", "Evans"];
        let specialties = ["Internal Medicine", "Family Medicine", "Cardiology", "Endocrinology", "Emergency Medicine"];

        TestProvider {
            id: format!("PROV-{:04}", seed % 1000),
            name: format!("{} {}", 
                         first_names[(seed as usize) % first_names.len()],
                         last_names[((seed + 1) as usize) % last_names.len()]),
            specialty: specialties[(seed as usize) % specialties.len()].to_string(),
            npi: format!("{:010}", 1000000000 + (seed % 999999999)),
        }
    }

    fn generate_location(&self, seed: u64) -> String {
        let locations = [
            "Emergency Department", "Medical Floor 3", "ICU", "Outpatient Clinic A", 
            "Cardiology Suite", "Endocrinology Clinic", "Family Medicine"
        ];
        locations[(seed as usize) % locations.len()].to_string()
    }

    fn generate_diagnosis(&self, seed: u64) -> Vec<TestDiagnosis> {
        let diagnoses = [
            ("Acute myocardial infarction", "I21.9", "primary"),
            ("Pneumonia", "J18.9", "primary"),
            ("Chest pain", "R06.02", "secondary"),
            ("Hypertensive crisis", "I16.9", "primary"),
            ("Diabetic ketoacidosis", "E10.10", "primary"),
        ];

        let diag_idx = (seed as usize) % diagnoses.len();
        let (name, code, type_) = diagnoses[diag_idx];

        vec![TestDiagnosis {
            name: name.to_string(),
            icd10_code: code.to_string(),
            diagnosis_type: type_.to_string(),
        }]
    }

    fn generate_prescriber_name(&self, seed: u64) -> String {
        let names = ["Dr. Smith", "Dr. Johnson", "Dr. Williams", "Dr. Brown", "Dr. Jones"];
        names[(seed as usize) % names.len()].to_string()
    }
}

impl Default for TestDataGenerator {
    fn default() -> Self {
        Self::new()
    }
}

/// Test patient data structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestPatient {
    pub id: String,
    pub mrn: String,
    pub first_name: String,
    pub last_name: String,
    pub date_of_birth: DateTime<Utc>,
    pub gender: String,
    pub allergies: Vec<TestAllergy>,
    pub medications: Vec<TestMedication>,
    pub conditions: Vec<TestCondition>,
    pub vital_signs: TestVitalSigns,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestAllergy {
    pub allergen: String,
    pub severity: String,
    pub reaction: String,
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestMedication {
    pub id: String,
    pub name: String,
    pub dose: String,
    pub frequency: String,
    pub route: String,
    pub prescriber: String,
    pub start_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub instructions: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestCondition {
    pub name: String,
    pub icd10_code: String,
    pub status: String,
    pub onset_date: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestVitalSigns {
    pub systolic_bp: u32,
    pub diastolic_bp: u32,
    pub heart_rate: u32,
    pub temperature: f32,
    pub respiratory_rate: u32,
    pub oxygen_saturation: u32,
    pub weight_kg: f32,
    pub height_cm: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestEncounter {
    pub id: String,
    pub patient_id: String,
    pub encounter_type: String,
    pub status: String,
    pub start_time: DateTime<Utc>,
    pub end_time: Option<DateTime<Utc>>,
    pub provider: TestProvider,
    pub location: String,
    pub diagnosis: Vec<TestDiagnosis>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestProvider {
    pub id: String,
    pub name: String,
    pub specialty: String,
    pub npi: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestDiagnosis {
    pub name: String,
    pub icd10_code: String,
    pub diagnosis_type: String,
}

/// Timing utility for measuring test execution time
pub struct TestTimer {
    start_time: Instant,
    checkpoints: Vec<(String, Instant)>,
}

impl TestTimer {
    pub fn new() -> Self {
        Self {
            start_time: Instant::now(),
            checkpoints: Vec::new(),
        }
    }

    pub fn checkpoint(&mut self, name: &str) {
        self.checkpoints.push((name.to_string(), Instant::now()));
    }

    pub fn elapsed(&self) -> Duration {
        self.start_time.elapsed()
    }

    pub fn checkpoint_duration(&self, checkpoint_name: &str) -> Option<Duration> {
        let checkpoint_time = self.checkpoints
            .iter()
            .find(|(name, _)| name == checkpoint_name)?
            .1;
        Some(checkpoint_time - self.start_time)
    }

    pub fn duration_between_checkpoints(&self, start: &str, end: &str) -> Option<Duration> {
        let start_time = self.checkpoints
            .iter()
            .find(|(name, _)| name == start)?
            .1;
        let end_time = self.checkpoints
            .iter()
            .find(|(name, _)| name == end)?
            .1;
        Some(end_time - start_time)
    }

    pub fn get_all_checkpoints(&self) -> Vec<(String, Duration)> {
        self.checkpoints
            .iter()
            .map(|(name, time)| (name.clone(), *time - self.start_time))
            .collect()
    }
}

impl Default for TestTimer {
    fn default() -> Self {
        Self::new()
    }
}

/// Test assertion helpers
pub struct TestAssertions;

impl TestAssertions {
    /// Assert that a test result is successful
    pub fn assert_test_passed(result: &TestResult) -> Result<()> {
        if result.status != TestStatus::Passed {
            return Err(anyhow::anyhow!(
                "Test '{}' failed with status: {:?}. Error: {:?}",
                result.test_name,
                result.status,
                result.error_message
            ));
        }
        Ok(())
    }

    /// Assert that a test completed within expected time
    pub fn assert_within_time_limit(result: &TestResult, max_duration: Duration) -> Result<()> {
        match result.duration {
            Some(duration) if duration <= max_duration => Ok(()),
            Some(duration) => Err(anyhow::anyhow!(
                "Test '{}' took {:?}, which exceeds limit of {:?}",
                result.test_name,
                duration,
                max_duration
            )),
            None => Err(anyhow::anyhow!("Test '{}' has no recorded duration", result.test_name)),
        }
    }

    /// Assert that clinical safety requirements are met
    pub fn assert_clinical_safety_compliant(result: &TestResult) -> Result<()> {
        if result.safety_class == ClinicalSafetyClass::PatientSafetyCritical && 
           result.status != TestStatus::Passed {
            return Err(anyhow::anyhow!(
                "Patient safety critical test '{}' failed: {:?}",
                result.test_name,
                result.error_message
            ));
        }
        Ok(())
    }

    /// Assert that no patient safety incidents occurred
    pub fn assert_no_safety_incidents(results: &[TestResult]) -> Result<()> {
        let safety_incidents: Vec<_> = results
            .iter()
            .filter(|r| {
                r.safety_class == ClinicalSafetyClass::PatientSafetyCritical &&
                r.status == TestStatus::Failed
            })
            .collect();

        if !safety_incidents.is_empty() {
            return Err(anyhow::anyhow!(
                "Patient safety incidents detected in {} tests: {}",
                safety_incidents.len(),
                safety_incidents
                    .iter()
                    .map(|r| &r.test_name)
                    .cloned()
                    .collect::<Vec<_>>()
                    .join(", ")
            ));
        }
        Ok(())
    }

    /// Assert that compliance requirements are met
    pub fn assert_compliance_requirements_met(result: &TestResult) -> Result<()> {
        if !result.compliance_notes.is_empty() && result.status != TestStatus::Passed {
            return Err(anyhow::anyhow!(
                "Compliance test '{}' failed. Compliance notes: {:?}",
                result.test_name,
                result.compliance_notes
            ));
        }
        Ok(())
    }

    /// Assert that test results meet minimum success rate
    pub fn assert_success_rate(results: &[TestResult], min_success_rate: f64) -> Result<()> {
        if results.is_empty() {
            return Ok(());
        }

        let passed_count = results.iter().filter(|r| r.status == TestStatus::Passed).count();
        let success_rate = passed_count as f64 / results.len() as f64;

        if success_rate < min_success_rate {
            return Err(anyhow::anyhow!(
                "Success rate {:.2}% is below minimum required {:.2}%",
                success_rate * 100.0,
                min_success_rate * 100.0
            ));
        }
        Ok(())
    }

    /// Assert that critical tests all passed
    pub fn assert_critical_tests_passed(results: &[TestResult]) -> Result<()> {
        let failed_critical: Vec<_> = results
            .iter()
            .filter(|r| r.priority == TestPriority::Critical && r.status != TestStatus::Passed)
            .collect();

        if !failed_critical.is_empty() {
            return Err(anyhow::anyhow!(
                "Critical tests failed: {}",
                failed_critical
                    .iter()
                    .map(|r| &r.test_name)
                    .cloned()
                    .collect::<Vec<_>>()
                    .join(", ")
            ));
        }
        Ok(())
    }
}

/// Clinical data validation utilities
pub struct ClinicalDataValidator;

impl ClinicalDataValidator {
    /// Validate patient data completeness
    pub fn validate_patient_data(patient: &TestPatient) -> Result<()> {
        if patient.id.is_empty() {
            return Err(anyhow::anyhow!("Patient ID cannot be empty"));
        }
        if patient.mrn.is_empty() {
            return Err(anyhow::anyhow!("Patient MRN cannot be empty"));
        }
        if patient.first_name.is_empty() || patient.last_name.is_empty() {
            return Err(anyhow::anyhow!("Patient name cannot be empty"));
        }
        Ok(())
    }

    /// Validate medication data
    pub fn validate_medication_data(medication: &TestMedication) -> Result<()> {
        if medication.name.is_empty() {
            return Err(anyhow::anyhow!("Medication name cannot be empty"));
        }
        if medication.dose.is_empty() {
            return Err(anyhow::anyhow!("Medication dose cannot be empty"));
        }
        if medication.frequency.is_empty() {
            return Err(anyhow::anyhow!("Medication frequency cannot be empty"));
        }
        Ok(())
    }

    /// Validate vital signs are within reasonable ranges
    pub fn validate_vital_signs(vitals: &TestVitalSigns) -> Result<()> {
        if vitals.systolic_bp < 80 || vitals.systolic_bp > 200 {
            return Err(anyhow::anyhow!("Systolic BP {} is out of range", vitals.systolic_bp));
        }
        if vitals.diastolic_bp < 50 || vitals.diastolic_bp > 120 {
            return Err(anyhow::anyhow!("Diastolic BP {} is out of range", vitals.diastolic_bp));
        }
        if vitals.heart_rate < 40 || vitals.heart_rate > 150 {
            return Err(anyhow::anyhow!("Heart rate {} is out of range", vitals.heart_rate));
        }
        if vitals.temperature < 95.0 || vitals.temperature > 105.0 {
            return Err(anyhow::anyhow!("Temperature {} is out of range", vitals.temperature));
        }
        if vitals.oxygen_saturation < 80 || vitals.oxygen_saturation > 100 {
            return Err(anyhow::anyhow!("Oxygen saturation {} is out of range", vitals.oxygen_saturation));
        }
        Ok(())
    }

    /// Validate ICD-10 code format
    pub fn validate_icd10_code(code: &str) -> Result<()> {
        if code.is_empty() {
            return Err(anyhow::anyhow!("ICD-10 code cannot be empty"));
        }
        if code.len() < 3 || code.len() > 7 {
            return Err(anyhow::anyhow!("ICD-10 code '{}' has invalid length", code));
        }
        Ok(())
    }
}

/// Test environment utilities
pub struct TestEnvironment {
    pub base_url: String,
    pub timeout: Duration,
    pub retry_attempts: u32,
    pub test_data_path: String,
}

impl TestEnvironment {
    pub fn new() -> Self {
        Self {
            base_url: "http://localhost:8000".to_string(),
            timeout: Duration::from_secs(30),
            retry_attempts: 3,
            test_data_path: "./test_data".to_string(),
        }
    }

    pub fn from_env() -> Self {
        Self {
            base_url: std::env::var("TEST_BASE_URL")
                .unwrap_or_else(|_| "http://localhost:8000".to_string()),
            timeout: Duration::from_secs(
                std::env::var("TEST_TIMEOUT")
                    .unwrap_or_else(|_| "30".to_string())
                    .parse()
                    .unwrap_or(30),
            ),
            retry_attempts: std::env::var("TEST_RETRY_ATTEMPTS")
                .unwrap_or_else(|_| "3".to_string())
                .parse()
                .unwrap_or(3),
            test_data_path: std::env::var("TEST_DATA_PATH")
                .unwrap_or_else(|_| "./test_data".to_string()),
        }
    }
}

impl Default for TestEnvironment {
    fn default() -> Self {
        Self::new()
    }
}

/// Test metrics collector
pub struct TestMetricsCollector {
    metrics: Arc<RwLock<HashMap<String, serde_json::Value>>>,
}

impl TestMetricsCollector {
    pub fn new() -> Self {
        Self {
            metrics: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn record_metric(&self, key: &str, value: serde_json::Value) {
        let mut metrics = self.metrics.write().await;
        metrics.insert(key.to_string(), value);
    }

    pub async fn get_metric(&self, key: &str) -> Option<serde_json::Value> {
        let metrics = self.metrics.read().await;
        metrics.get(key).cloned()
    }

    pub async fn get_all_metrics(&self) -> HashMap<String, serde_json::Value> {
        let metrics = self.metrics.read().await;
        metrics.clone()
    }

    pub async fn clear_metrics(&self) {
        let mut metrics = self.metrics.write().await;
        metrics.clear();
    }
}

impl Default for TestMetricsCollector {
    fn default() -> Self {
        Self::new()
    }
}

/// Test fixture for reusable test setup
pub struct TestFixture {
    pub generator: TestDataGenerator,
    pub timer: TestTimer,
    pub metrics: TestMetricsCollector,
    pub environment: TestEnvironment,
}

impl TestFixture {
    pub fn new() -> Self {
        Self {
            generator: TestDataGenerator::new(),
            timer: TestTimer::new(),
            metrics: TestMetricsCollector::new(),
            environment: TestEnvironment::from_env(),
        }
    }

    pub async fn setup_test_data(&self) -> Result<Vec<TestPatient>> {
        let mut patients = Vec::new();
        for _i in 0..10 {
            patients.push(self.generator.generate_test_patient().await);
        }
        Ok(patients)
    }

    pub async fn cleanup(&self) {
        self.metrics.clear_metrics().await;
    }
}

impl Default for TestFixture {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_data_generator() {
        let generator = TestDataGenerator::with_seed(12345);
        let patient = generator.generate_test_patient().await;
        
        assert!(!patient.id.is_empty());
        assert!(!patient.first_name.is_empty());
        assert!(!patient.last_name.is_empty());
        assert!(!patient.mrn.is_empty());
        
        // Validate data
        assert!(ClinicalDataValidator::validate_patient_data(&patient).is_ok());
        assert!(ClinicalDataValidator::validate_vital_signs(&patient.vital_signs).is_ok());
    }

    #[tokio::test]
    async fn test_encounter_generation() {
        let generator = TestDataGenerator::with_seed(54321);
        let encounter = generator.generate_test_encounter("TEST-PAT-001").await;
        
        assert!(!encounter.id.is_empty());
        assert_eq!(encounter.patient_id, "TEST-PAT-001");
        assert!(!encounter.provider.name.is_empty());
        assert!(!encounter.provider.specialty.is_empty());
    }

    #[test]
    fn test_medication_generation() {
        let generator = TestDataGenerator::with_seed(98765);
        let medication = generator.generate_test_medication(123);
        
        assert!(!medication.name.is_empty());
        assert!(!medication.dose.is_empty());
        assert!(!medication.frequency.is_empty());
        
        // Validate medication data
        assert!(ClinicalDataValidator::validate_medication_data(&medication).is_ok());
    }

    #[test]
    fn test_timer_functionality() {
        let mut timer = TestTimer::new();
        
        std::thread::sleep(Duration::from_millis(10));
        timer.checkpoint("first");
        
        std::thread::sleep(Duration::from_millis(10));
        timer.checkpoint("second");
        
        assert!(timer.elapsed() >= Duration::from_millis(20));
        assert!(timer.checkpoint_duration("first").is_some());
        assert!(timer.duration_between_checkpoints("first", "second").is_some());
    }

    #[test]
    fn test_assertions() {
        let passed_result = TestResult {
            test_id: Uuid::new_v4(),
            test_name: "test_pass".to_string(),
            test_category: "unit".to_string(),
            status: TestStatus::Passed,
            priority: TestPriority::Critical,
            safety_class: ClinicalSafetyClass::PatientSafetyCritical,
            start_time: Utc::now(),
            end_time: Some(Utc::now()),
            duration: Some(Duration::from_secs(1)),
            error_message: None,
            metrics: HashMap::new(),
            clinical_context: ClinicalTestContext {
                patient_count: 0,
                provider_count: 0,
                clinical_scenarios: vec![],
                fhir_resources_tested: vec![],
                clinical_protocols_validated: vec![],
                safety_checks_performed: vec![],
                hipaa_safeguards: vec![],
            },
            compliance_notes: vec![],
        };

        assert!(TestAssertions::assert_test_passed(&passed_result).is_ok());
        assert!(TestAssertions::assert_within_time_limit(&passed_result, Duration::from_secs(2)).is_ok());
        assert!(TestAssertions::assert_clinical_safety_compliant(&passed_result).is_ok());
    }

    #[test]
    fn test_clinical_data_validation() {
        let patient = TestPatient {
            id: "TEST-001".to_string(),
            mrn: "MRN123456".to_string(),
            first_name: "John".to_string(),
            last_name: "Doe".to_string(),
            date_of_birth: Utc::now(),
            gender: "male".to_string(),
            allergies: vec![],
            medications: vec![],
            conditions: vec![],
            vital_signs: TestVitalSigns {
                systolic_bp: 120,
                diastolic_bp: 80,
                heart_rate: 70,
                temperature: 98.6,
                respiratory_rate: 16,
                oxygen_saturation: 99,
                weight_kg: 70.0,
                height_cm: 175.0,
            },
        };

        assert!(ClinicalDataValidator::validate_patient_data(&patient).is_ok());
        assert!(ClinicalDataValidator::validate_vital_signs(&patient.vital_signs).is_ok());
        assert!(ClinicalDataValidator::validate_icd10_code("I10").is_ok());
    }

    #[tokio::test]
    async fn test_metrics_collector() {
        let collector = TestMetricsCollector::new();
        
        collector.record_metric("test_count", serde_json::json!(5)).await;
        collector.record_metric("success_rate", serde_json::json!(0.95)).await;
        
        assert_eq!(collector.get_metric("test_count").await, Some(serde_json::json!(5)));
        assert_eq!(collector.get_metric("success_rate").await, Some(serde_json::json!(0.95)));
        
        let all_metrics = collector.get_all_metrics().await;
        assert_eq!(all_metrics.len(), 2);
    }

    #[tokio::test]
    async fn test_fixture_setup() {
        let fixture = TestFixture::new();
        let patients = fixture.setup_test_data().await.unwrap();
        
        assert_eq!(patients.len(), 10);
        for patient in &patients {
            assert!(ClinicalDataValidator::validate_patient_data(patient).is_ok());
        }
        
        fixture.cleanup().await;
    }
}