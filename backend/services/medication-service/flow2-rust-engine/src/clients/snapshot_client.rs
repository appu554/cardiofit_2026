//! HTTP client for Context Gateway snapshot retrieval
//! 
//! Provides functionality for fetching, validating, and processing clinical data
//! snapshots from the Context Gateway service.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use chrono::{DateTime, Utc};
use reqwest::Client;
use tracing::{info, warn, error, debug};
use crate::EngineError;

/// Configuration for the snapshot client
#[derive(Debug, Clone)]
pub struct SnapshotClientConfig {
    pub context_gateway_url: String,
    pub timeout_seconds: u64,
    pub retry_attempts: u32,
    pub retry_delay_ms: u64,
    pub enable_integrity_verification: bool,
}

impl Default for SnapshotClientConfig {
    fn default() -> Self {
        Self {
            context_gateway_url: "http://localhost:8016".to_string(),
            timeout_seconds: 30,
            retry_attempts: 3,
            retry_delay_ms: 1000,
            enable_integrity_verification: true,
        }
    }
}

/// Clinical data snapshot from Context Gateway
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalSnapshot {
    pub snapshot_id: String,
    pub patient_id: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub checksum: String,
    pub signature: Option<String>,
    pub data: ClinicalSnapshotData,
    pub metadata: SnapshotMetadata,
}

/// Core clinical data within a snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalSnapshotData {
    pub patient_demographics: PatientDemographics,
    pub active_medications: Vec<ActiveMedication>,
    pub allergies: Vec<Allergy>,
    pub lab_values: Vec<LabValue>,
    pub conditions: Vec<MedicalCondition>,
    pub vital_signs: Vec<VitalSign>,
    pub clinical_notes: Vec<ClinicalNote>,
}

/// Patient demographic information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientDemographics {
    pub age_years: f64,
    pub weight_kg: f64,
    pub height_cm: f64,
    pub gender: String,
    pub race: Option<String>,
    pub ethnicity: Option<String>,
    pub bmi: Option<f64>,
    pub bsa_m2: Option<f64>,
    pub egfr: Option<f64>,
    pub creatinine_clearance: Option<f64>,
}

/// Active medication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActiveMedication {
    pub medication_id: String,
    pub medication_code: String,
    pub medication_name: String,
    pub dose_value: Option<f64>,
    pub dose_unit: Option<String>,
    pub frequency: Option<String>,
    pub route: Option<String>,
    pub start_date: Option<DateTime<Utc>>,
    pub prescriber_id: Option<String>,
}

/// Allergy information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Allergy {
    pub allergen_code: String,
    pub allergen_name: String,
    pub reaction: Option<String>,
    pub severity: Option<String>,
    pub onset_date: Option<DateTime<Utc>>,
    pub verified: bool,
}

/// Laboratory value
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabValue {
    pub test_code: String,
    pub test_name: String,
    pub value: f64,
    pub unit: String,
    pub reference_range: Option<String>,
    pub result_status: Option<String>,
    pub collected_at: DateTime<Utc>,
    pub reported_at: Option<DateTime<Utc>>,
}

/// Medical condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicalCondition {
    pub condition_code: String,
    pub condition_name: String,
    pub status: String, // active, inactive, resolved
    pub severity: Option<String>,
    pub onset_date: Option<DateTime<Utc>>,
    pub diagnosis_date: Option<DateTime<Utc>>,
}

/// Vital sign measurement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VitalSign {
    pub vital_type: String, // blood_pressure, heart_rate, temperature, etc.
    pub value: f64,
    pub unit: String,
    pub measured_at: DateTime<Utc>,
    pub measurement_method: Option<String>,
}

/// Clinical note entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalNote {
    pub note_id: String,
    pub note_type: String, // progress_note, assessment, plan, etc.
    pub author_id: String,
    pub created_at: DateTime<Utc>,
    pub content: String,
    pub tags: Vec<String>,
}

/// Snapshot metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotMetadata {
    pub version: String,
    pub data_sources: Vec<String>,
    pub assembly_duration_ms: u64,
    pub total_records: u32,
    pub data_quality_score: f64,
    pub completeness_flags: HashMap<String, bool>,
}

/// Snapshot integrity verification result
#[derive(Debug, Clone)]
pub struct IntegrityVerification {
    pub is_valid: bool,
    pub checksum_verified: bool,
    pub signature_verified: bool,
    pub is_expired: bool,
    pub verification_errors: Vec<String>,
}

/// HTTP client for snapshot operations
#[derive(Debug, Clone)]
pub struct SnapshotClient {
    client: Client,
    config: SnapshotClientConfig,
}

impl SnapshotClient {
    /// Create a new snapshot client with default configuration
    pub fn new() -> Result<Self, EngineError> {
        Self::with_config(SnapshotClientConfig::default())
    }

    /// Create a new snapshot client with custom configuration
    pub fn with_config(config: SnapshotClientConfig) -> Result<Self, EngineError> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(config.timeout_seconds))
            .user_agent(format!("{}/{}", crate::ENGINE_NAME, crate::VERSION))
            .build()
            .map_err(|e| EngineError::Http(e))?;

        Ok(Self { client, config })
    }

    /// Fetch a clinical snapshot by ID with retry logic
    pub async fn fetch_snapshot(&self, snapshot_id: &str) -> Result<ClinicalSnapshot, EngineError> {
        let mut attempt = 0;
        let mut last_error = None;

        while attempt < self.config.retry_attempts {
            attempt += 1;
            debug!("Fetching snapshot {} (attempt {}/{})", snapshot_id, attempt, self.config.retry_attempts);

            match self.fetch_snapshot_attempt(snapshot_id).await {
                Ok(snapshot) => {
                    info!("Successfully fetched snapshot {}", snapshot_id);
                    return Ok(snapshot);
                }
                Err(e) => {
                    last_error = Some(e);
                    if attempt < self.config.retry_attempts {
                        warn!("Snapshot fetch attempt {} failed, retrying in {}ms", 
                              attempt, self.config.retry_delay_ms);
                        tokio::time::sleep(
                            std::time::Duration::from_millis(self.config.retry_delay_ms)
                        ).await;
                    }
                }
            }
        }

        error!("Failed to fetch snapshot {} after {} attempts", 
               snapshot_id, self.config.retry_attempts);
        Err(last_error.unwrap_or_else(|| 
            EngineError::Generic("Failed to fetch snapshot".to_string())
        ))
    }

    /// Single attempt to fetch a snapshot
    async fn fetch_snapshot_attempt(&self, snapshot_id: &str) -> Result<ClinicalSnapshot, EngineError> {
        let url = format!("{}/api/snapshots/{}", self.config.context_gateway_url, snapshot_id);
        
        debug!("Making HTTP request to: {}", url);
        
        let response = self.client
            .get(&url)
            .header("Accept", "application/json")
            .header("X-Request-Source", "flow2-rust-engine")
            .send()
            .await
            .map_err(|e| {
                error!("HTTP request failed: {}", e);
                EngineError::Http(e)
            })?;

        if !response.status().is_success() {
            let status = response.status();
            let error_text = response.text().await.unwrap_or_else(|_| "Unknown error".to_string());
            error!("HTTP request failed with status {}: {}", status, error_text);
            return Err(EngineError::Generic(
                format!("Snapshot fetch failed: HTTP {}: {}", status, error_text)
            ));
        }

        let snapshot: ClinicalSnapshot = response.json().await
            .map_err(|e| {
                error!("Failed to parse snapshot JSON: {}", e);
                EngineError::Http(e)
            })?;

        debug!("Successfully parsed snapshot {}", snapshot.snapshot_id);
        Ok(snapshot)
    }

    /// Verify snapshot integrity (checksum and signature)
    pub fn verify_integrity(&self, snapshot: &ClinicalSnapshot) -> IntegrityVerification {
        let mut verification = IntegrityVerification {
            is_valid: true,
            checksum_verified: false,
            signature_verified: false,
            is_expired: false,
            verification_errors: Vec::new(),
        };

        // Check expiration
        let now = Utc::now();
        if now > snapshot.expires_at {
            verification.is_expired = true;
            verification.is_valid = false;
            verification.verification_errors.push(
                format!("Snapshot expired at {}, current time is {}", 
                       snapshot.expires_at, now)
            );
        }

        // Skip integrity checks if disabled
        if !self.config.enable_integrity_verification {
            warn!("Snapshot integrity verification is disabled");
            verification.checksum_verified = true;
            verification.signature_verified = true;
            return verification;
        }

        // Verify checksum
        match self.verify_checksum(snapshot) {
            Ok(()) => {
                verification.checksum_verified = true;
                debug!("Snapshot checksum verification passed");
            }
            Err(e) => {
                verification.is_valid = false;
                verification.verification_errors.push(
                    format!("Checksum verification failed: {}", e)
                );
                warn!("Snapshot checksum verification failed: {}", e);
            }
        }

        // Verify signature if present
        if let Some(ref signature) = snapshot.signature {
            match self.verify_signature(snapshot, signature) {
                Ok(()) => {
                    verification.signature_verified = true;
                    debug!("Snapshot signature verification passed");
                }
                Err(e) => {
                    verification.is_valid = false;
                    verification.verification_errors.push(
                        format!("Signature verification failed: {}", e)
                    );
                    warn!("Snapshot signature verification failed: {}", e);
                }
            }
        } else {
            verification.signature_verified = true; // No signature to verify
            debug!("No signature present in snapshot");
        }

        verification
    }

    /// Verify snapshot checksum
    fn verify_checksum(&self, snapshot: &ClinicalSnapshot) -> Result<(), String> {
        // Serialize the data for checksum calculation
        let data_json = serde_json::to_string(&snapshot.data)
            .map_err(|e| format!("Failed to serialize snapshot data: {}", e))?;
        
        // Calculate SHA-256 hash of the data
        use sha2::{Sha256, Digest};
        let mut hasher = Sha256::new();
        hasher.update(data_json.as_bytes());
        let calculated_hash = format!("{:x}", hasher.finalize());
        
        if calculated_hash == snapshot.checksum {
            Ok(())
        } else {
            Err(format!("Checksum mismatch: expected {}, calculated {}", 
                       snapshot.checksum, calculated_hash))
        }
    }

    /// Verify snapshot signature (placeholder implementation)
    fn verify_signature(&self, _snapshot: &ClinicalSnapshot, _signature: &str) -> Result<(), String> {
        // TODO: Implement proper digital signature verification
        // This would typically involve:
        // 1. Loading the public key from configuration or key store
        // 2. Verifying the signature against the snapshot data
        // 3. Ensuring the signing certificate is valid and trusted
        
        warn!("Signature verification not yet implemented - skipping");
        Ok(())
    }

    /// Fetch and verify snapshot in one operation
    pub async fn fetch_and_verify_snapshot(&self, snapshot_id: &str) -> Result<ClinicalSnapshot, EngineError> {
        let snapshot = self.fetch_snapshot(snapshot_id).await?;
        
        let verification = self.verify_integrity(&snapshot);
        if !verification.is_valid {
            error!("Snapshot {} failed integrity verification: {:?}", 
                   snapshot_id, verification.verification_errors);
            return Err(EngineError::Generic(
                format!("Snapshot integrity verification failed: {}", 
                       verification.verification_errors.join(", "))
            ));
        }

        info!("Snapshot {} successfully fetched and verified", snapshot_id);
        Ok(snapshot)
    }

    /// Check if the Context Gateway is available
    pub async fn health_check(&self) -> Result<bool, EngineError> {
        let url = format!("{}/health", self.config.context_gateway_url);
        
        let response = self.client
            .get(&url)
            .send()
            .await
            .map_err(EngineError::Http)?;
            
        Ok(response.status().is_success())
    }
}

impl Default for SnapshotClient {
    fn default() -> Self {
        Self::new().expect("Failed to create default snapshot client")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_snapshot_client_config_default() {
        let config = SnapshotClientConfig::default();
        assert_eq!(config.context_gateway_url, "http://localhost:8016");
        assert_eq!(config.timeout_seconds, 30);
        assert_eq!(config.retry_attempts, 3);
        assert!(config.enable_integrity_verification);
    }

    #[test]
    fn test_clinical_snapshot_serialization() {
        let snapshot = ClinicalSnapshot {
            snapshot_id: "test-snapshot-123".to_string(),
            patient_id: "patient-456".to_string(),
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::hours(1),
            checksum: "abc123".to_string(),
            signature: None,
            data: ClinicalSnapshotData {
                patient_demographics: PatientDemographics {
                    age_years: 45.0,
                    weight_kg: 70.0,
                    height_cm: 170.0,
                    gender: "male".to_string(),
                    race: None,
                    ethnicity: None,
                    bmi: None,
                    bsa_m2: None,
                    egfr: Some(90.0),
                    creatinine_clearance: None,
                },
                active_medications: vec![],
                allergies: vec![],
                lab_values: vec![],
                conditions: vec![],
                vital_signs: vec![],
                clinical_notes: vec![],
            },
            metadata: SnapshotMetadata {
                version: "1.0".to_string(),
                data_sources: vec!["EHR".to_string()],
                assembly_duration_ms: 150,
                total_records: 25,
                data_quality_score: 0.95,
                completeness_flags: HashMap::new(),
            },
        };

        let json = serde_json::to_string(&snapshot).expect("Serialization should succeed");
        let deserialized: ClinicalSnapshot = serde_json::from_str(&json)
            .expect("Deserialization should succeed");
        
        assert_eq!(snapshot.snapshot_id, deserialized.snapshot_id);
        assert_eq!(snapshot.patient_id, deserialized.patient_id);
    }

    #[tokio::test]
    async fn test_snapshot_client_creation() {
        let client = SnapshotClient::new();
        assert!(client.is_ok());
    }
}