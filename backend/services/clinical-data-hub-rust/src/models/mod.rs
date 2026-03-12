// Models module for Clinical Data Hub - Core data structures and types
pub mod cache;
pub mod data_source;
pub mod performance;
pub mod quality;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

/// Clinical data types supported by the hub
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum DataType {
    Demographics,
    Medications,
    Observations,
    Allergies,
    Conditions,
    Procedures,
    VitalSigns,
    LabResults,
    ImagingResults,
    ClinicalNotes,
}

/// Data source types for clinical information
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SourceType {
    Grpc { endpoint: String },
    Rest { base_url: String },
    GraphQL { endpoint: String },
    Fhir { store_url: String },
    Database { connection_string: String },
}

/// Priority levels for data sources
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub enum Priority {
    Low = 1,
    Medium = 2,
    High = 3,
    Critical = 4,
}

/// Compression types for data storage
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum CompressionType {
    None,
    Lz4,
    Zstd,
    MessagePack,
}

/// Update types for real-time data streaming
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum UpdateType {
    Create,
    Update,
    Delete,
    Batch,
}

/// Service health status
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum ServiceStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

/// Clinical data entry with metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalData {
    pub id: Uuid,
    pub patient_id: String,
    pub data_type: DataType,
    pub data: serde_json::Value,
    pub source: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub version: u64,
    pub checksum: String,
    pub metadata: HashMap<String, String>,
}

impl ClinicalData {
    /// Create a new clinical data entry
    pub fn new(
        patient_id: String,
        data_type: DataType,
        data: serde_json::Value,
        source: String,
    ) -> Self {
        let now = Utc::now();
        let checksum = Self::calculate_checksum(&data);
        
        Self {
            id: Uuid::new_v4(),
            patient_id,
            data_type,
            data,
            source,
            created_at: now,
            updated_at: now,
            version: 1,
            checksum,
            metadata: HashMap::new(),
        }
    }
    
    /// Calculate SHA-256 checksum of the data
    fn calculate_checksum(data: &serde_json::Value) -> String {
        use sha2::{Sha256, Digest};
        
        let data_string = serde_json::to_string(data).unwrap_or_default();
        let mut hasher = Sha256::new();
        hasher.update(data_string.as_bytes());
        format!("{:x}", hasher.finalize())
    }
    
    /// Update the data and increment version
    pub fn update(&mut self, new_data: serde_json::Value) {
        self.data = new_data;
        self.updated_at = Utc::now();
        self.version += 1;
        self.checksum = Self::calculate_checksum(&self.data);
    }
    
    /// Verify data integrity using checksum
    pub fn verify_integrity(&self) -> bool {
        self.checksum == Self::calculate_checksum(&self.data)
    }
    
    /// Get data age in seconds
    pub fn age_seconds(&self) -> i64 {
        (Utc::now() - self.updated_at).num_seconds()
    }
    
    /// Check if data is fresh within threshold
    pub fn is_fresh(&self, threshold_seconds: i64) -> bool {
        self.age_seconds() <= threshold_seconds
    }
}

/// Data aggregation request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregationRequest {
    pub patient_id: String,
    pub data_types: Vec<DataType>,
    pub sources: Vec<String>,
    pub parallel_execution: bool,
    pub max_concurrency: usize,
    pub timeout_ms: u64,
    pub allow_partial_results: bool,
    pub freshness_threshold_seconds: i64,
}

/// Aggregated clinical data result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregatedData {
    pub patient_id: String,
    pub data: HashMap<DataType, Vec<ClinicalData>>,
    pub sources_queried: Vec<String>,
    pub sources_successful: Vec<String>,
    pub sources_failed: Vec<String>,
    pub completeness_score: f64,
    pub consistency_score: f64,
    pub freshness_score: f64,
    pub aggregation_time_ms: u64,
    pub total_records: usize,
}

impl AggregatedData {
    /// Create a new aggregated data result
    pub fn new(patient_id: String) -> Self {
        Self {
            patient_id,
            data: HashMap::new(),
            sources_queried: Vec::new(),
            sources_successful: Vec::new(),
            sources_failed: Vec::new(),
            completeness_score: 0.0,
            consistency_score: 0.0,
            freshness_score: 0.0,
            aggregation_time_ms: 0,
            total_records: 0,
        }
    }
    
    /// Add data for a specific type
    pub fn add_data(&mut self, data_type: DataType, data: Vec<ClinicalData>) {
        self.total_records += data.len();
        self.data.insert(data_type, data);
    }
    
    /// Calculate overall quality score
    pub fn overall_quality_score(&self) -> f64 {
        (self.completeness_score + self.consistency_score + self.freshness_score) / 3.0
    }
    
    /// Get data for a specific type
    pub fn get_data(&self, data_type: &DataType) -> Option<&Vec<ClinicalData>> {
        self.data.get(data_type)
    }
    
    /// Check if aggregation is complete
    pub fn is_complete(&self) -> bool {
        !self.data.is_empty() && self.sources_failed.is_empty()
    }
}

/// Real-time data update event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataUpdateEvent {
    pub event_id: Uuid,
    pub patient_id: String,
    pub data_type: DataType,
    pub update_type: UpdateType,
    pub data: Option<serde_json::Value>,
    pub source: String,
    pub timestamp: DateTime<Utc>,
    pub metadata: HashMap<String, String>,
}

impl DataUpdateEvent {
    /// Create a new data update event
    pub fn new(
        patient_id: String,
        data_type: DataType,
        update_type: UpdateType,
        data: Option<serde_json::Value>,
        source: String,
    ) -> Self {
        Self {
            event_id: Uuid::new_v4(),
            patient_id,
            data_type,
            update_type,
            data,
            source,
            timestamp: Utc::now(),
            metadata: HashMap::new(),
        }
    }
    
    /// Check if event is recent within threshold
    pub fn is_recent(&self, threshold_seconds: i64) -> bool {
        (Utc::now() - self.timestamp).num_seconds() <= threshold_seconds
    }
}