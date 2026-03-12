//! Evidence Envelope Management
//! 
//! Manages evidence tracking, KB version management, and audit compliance
//! for Phase 3 Clinical Intelligence Engine operations.

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use uuid::Uuid;
use sha2::{Sha256, Digest};
use anyhow::{Result, anyhow};

use super::models::EvidenceEnvelope;

/// Evidence manager for Phase 3 operations
pub struct EvidenceManager {
    current_versions: HashMap<String, String>,
}

impl EvidenceManager {
    /// Create new evidence manager
    pub fn new() -> Self {
        Self {
            current_versions: HashMap::new(),
        }
    }
    
    /// Create evidence envelope for Phase 3 operation
    pub fn create_evidence_envelope(
        &self,
        kb_versions: HashMap<String, String>,
        processing_chain: Vec<String>,
    ) -> Result<EvidenceEnvelope> {
        let envelope_id = Uuid::new_v4().to_string();
        let audit_id = Uuid::new_v4().to_string();
        
        // Create snapshot hash from KB versions
        let snapshot_hash = self.calculate_snapshot_hash(&kb_versions)?;
        
        Ok(EvidenceEnvelope {
            envelope_id,
            created_at: Utc::now(),
            kb_versions,
            snapshot_hash,
            signature: None, // TODO: Add cryptographic signature if needed
            audit_id,
            processing_chain,
        })
    }
    
    /// Validate evidence envelope integrity
    pub fn validate_evidence_envelope(&self, envelope: &EvidenceEnvelope) -> Result<bool> {
        // Verify snapshot hash
        let calculated_hash = self.calculate_snapshot_hash(&envelope.kb_versions)?;
        
        if calculated_hash != envelope.snapshot_hash {
            return Ok(false);
        }
        
        // TODO: Add signature verification if signatures are used
        
        Ok(true)
    }
    
    /// Update KB versions
    pub fn update_kb_versions(&mut self, versions: HashMap<String, String>) {
        self.current_versions = versions;
    }
    
    /// Get current KB versions
    pub fn get_current_versions(&self) -> &HashMap<String, String> {
        &self.current_versions
    }
    
    /// Calculate snapshot hash from KB versions
    fn calculate_snapshot_hash(&self, kb_versions: &HashMap<String, String>) -> Result<String> {
        let mut hasher = Sha256::new();
        
        // Sort KB versions for consistent hashing
        let mut sorted_versions: Vec<_> = kb_versions.iter().collect();
        sorted_versions.sort_by_key(|(k, _)| *k);
        
        for (kb_name, version) in sorted_versions {
            hasher.update(format!("{}:{}", kb_name, version).as_bytes());
        }
        
        Ok(format!("{:x}", hasher.finalize()))
    }
    
    /// Add processing step to envelope
    pub fn add_processing_step(
        &self,
        envelope: &mut EvidenceEnvelope,
        step: String,
    ) -> Result<()> {
        envelope.processing_chain.push(step);
        
        // Recalculate hash if needed
        envelope.snapshot_hash = self.calculate_snapshot_hash(&envelope.kb_versions)?;
        
        Ok(())
    }
}

/// Evidence collection for audit compliance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditTrail {
    pub operation_id: String,
    pub timestamp: DateTime<Utc>,
    pub evidence_envelope: EvidenceEnvelope,
    pub input_hash: String,
    pub output_hash: String,
    pub performance_metrics: PerformanceEvidence,
    pub decisions_made: Vec<DecisionEvidence>,
    pub kb_queries: Vec<KnowledgeBaseQuery>,
}

/// Performance evidence for audit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceEvidence {
    pub phase3_duration_ms: u64,
    pub sub_phase_timings: HashMap<String, u64>,
    pub sla_compliance: bool,
    pub resource_usage: ResourceUsage,
}

/// Resource usage metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceUsage {
    pub cpu_usage_percent: Option<f64>,
    pub memory_usage_mb: Option<u64>,
    pub network_requests: u32,
    pub cache_hits: u32,
    pub cache_misses: u32,
}

/// Decision evidence for audit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DecisionEvidence {
    pub decision_type: String,
    pub input_factors: Vec<String>,
    pub decision_logic: String,
    pub confidence_score: f64,
    pub alternatives_considered: Vec<String>,
    pub rationale: String,
}

/// Knowledge base query evidence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeBaseQuery {
    pub kb_name: String,
    pub kb_version: String,
    pub query_type: String,
    pub query_parameters: HashMap<String, serde_json::Value>,
    pub response_summary: String,
    pub execution_time_ms: u64,
}

impl AuditTrail {
    /// Create new audit trail
    pub fn new(
        operation_id: String,
        evidence_envelope: EvidenceEnvelope,
        input_hash: String,
        output_hash: String,
    ) -> Self {
        Self {
            operation_id,
            timestamp: Utc::now(),
            evidence_envelope,
            input_hash,
            output_hash,
            performance_metrics: PerformanceEvidence {
                phase3_duration_ms: 0,
                sub_phase_timings: HashMap::new(),
                sla_compliance: true,
                resource_usage: ResourceUsage {
                    cpu_usage_percent: None,
                    memory_usage_mb: None,
                    network_requests: 0,
                    cache_hits: 0,
                    cache_misses: 0,
                },
            },
            decisions_made: Vec::new(),
            kb_queries: Vec::new(),
        }
    }
    
    /// Add decision evidence
    pub fn add_decision(
        &mut self,
        decision_type: String,
        input_factors: Vec<String>,
        decision_logic: String,
        confidence_score: f64,
        alternatives_considered: Vec<String>,
        rationale: String,
    ) {
        let decision = DecisionEvidence {
            decision_type,
            input_factors,
            decision_logic,
            confidence_score,
            alternatives_considered,
            rationale,
        };
        
        self.decisions_made.push(decision);
    }
    
    /// Add KB query evidence
    pub fn add_kb_query(
        &mut self,
        kb_name: String,
        kb_version: String,
        query_type: String,
        query_parameters: HashMap<String, serde_json::Value>,
        response_summary: String,
        execution_time_ms: u64,
    ) {
        let query = KnowledgeBaseQuery {
            kb_name,
            kb_version,
            query_type,
            query_parameters,
            response_summary,
            execution_time_ms,
        };
        
        self.kb_queries.push(query);
    }
    
    /// Update performance metrics
    pub fn update_performance(
        &mut self,
        phase3_duration_ms: u64,
        sub_phase_timings: HashMap<String, u64>,
        sla_compliance: bool,
    ) {
        self.performance_metrics.phase3_duration_ms = phase3_duration_ms;
        self.performance_metrics.sub_phase_timings = sub_phase_timings;
        self.performance_metrics.sla_compliance = sla_compliance;
    }
    
    /// Generate audit report
    pub fn generate_audit_report(&self) -> Result<String> {
        let report = serde_json::to_string_pretty(self)
            .map_err(|e| anyhow!("Failed to serialize audit trail: {}", e))?;
        
        Ok(report)
    }
}

/// Calculate hash of data for integrity checking
pub fn calculate_data_hash(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    format!("{:x}", hasher.finalize())
}

/// Validate data integrity using hash
pub fn validate_data_integrity(data: &[u8], expected_hash: &str) -> bool {
    let calculated_hash = calculate_data_hash(data);
    calculated_hash == expected_hash
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_evidence_manager_creation() {
        let manager = EvidenceManager::new();
        assert!(manager.current_versions.is_empty());
    }
    
    #[test]
    fn test_evidence_envelope_creation() {
        let manager = EvidenceManager::new();
        
        let mut kb_versions = HashMap::new();
        kb_versions.insert("kb_guidelines".to_string(), "v1.0".to_string());
        kb_versions.insert("kb_formulary".to_string(), "v2.1".to_string());
        
        let processing_chain = vec![
            "phase1".to_string(),
            "phase2".to_string(),
            "phase3".to_string(),
        ];
        
        let envelope = manager.create_evidence_envelope(kb_versions, processing_chain).unwrap();
        
        assert!(!envelope.envelope_id.is_empty());
        assert!(!envelope.snapshot_hash.is_empty());
        assert_eq!(envelope.kb_versions.len(), 2);
        assert_eq!(envelope.processing_chain.len(), 3);
    }
    
    #[test]
    fn test_snapshot_hash_calculation() {
        let manager = EvidenceManager::new();
        
        let mut kb_versions1 = HashMap::new();
        kb_versions1.insert("kb_a".to_string(), "v1.0".to_string());
        kb_versions1.insert("kb_b".to_string(), "v2.0".to_string());
        
        let mut kb_versions2 = HashMap::new();
        kb_versions2.insert("kb_b".to_string(), "v2.0".to_string());
        kb_versions2.insert("kb_a".to_string(), "v1.0".to_string()); // Different order
        
        let hash1 = manager.calculate_snapshot_hash(&kb_versions1).unwrap();
        let hash2 = manager.calculate_snapshot_hash(&kb_versions2).unwrap();
        
        assert_eq!(hash1, hash2); // Should be same despite different insertion order
    }
    
    #[test]
    fn test_evidence_envelope_validation() {
        let manager = EvidenceManager::new();
        
        let mut kb_versions = HashMap::new();
        kb_versions.insert("test_kb".to_string(), "v1.0".to_string());
        
        let envelope = manager.create_evidence_envelope(kb_versions, vec![]).unwrap();
        
        let is_valid = manager.validate_evidence_envelope(&envelope).unwrap();
        assert!(is_valid);
    }
    
    #[test]
    fn test_audit_trail_creation() {
        let mut kb_versions = HashMap::new();
        kb_versions.insert("test_kb".to_string(), "v1.0".to_string());
        
        let envelope = EvidenceEnvelope {
            envelope_id: "test_envelope".to_string(),
            created_at: Utc::now(),
            kb_versions,
            snapshot_hash: "test_hash".to_string(),
            signature: None,
            audit_id: "test_audit".to_string(),
            processing_chain: vec!["test_phase".to_string()],
        };
        
        let audit_trail = AuditTrail::new(
            "test_operation".to_string(),
            envelope,
            "input_hash".to_string(),
            "output_hash".to_string(),
        );
        
        assert_eq!(audit_trail.operation_id, "test_operation");
        assert_eq!(audit_trail.input_hash, "input_hash");
        assert_eq!(audit_trail.output_hash, "output_hash");
    }
    
    #[test]
    fn test_data_hash_calculation() {
        let data1 = b"test data";
        let data2 = b"test data";
        let data3 = b"different data";
        
        let hash1 = calculate_data_hash(data1);
        let hash2 = calculate_data_hash(data2);
        let hash3 = calculate_data_hash(data3);
        
        assert_eq!(hash1, hash2);
        assert_ne!(hash1, hash3);
    }
    
    #[test]
    fn test_data_integrity_validation() {
        let data = b"test data";
        let hash = calculate_data_hash(data);
        
        assert!(validate_data_integrity(data, &hash));
        assert!(!validate_data_integrity(b"modified data", &hash));
    }
}