//! # DDI Adapter
//!
//! Drug-Drug Interaction adapter that integrates with existing DDI checking systems.
//! Provides batch checking capabilities with graceful degradation.

use crate::jit_safety::{domain::*, error::JitSafetyError};
use tracing::{debug, warn};

/// DDI adapter trait for checking drug interactions
pub trait DdiAdapter: Send + Sync {
    /// Check all interactions for a proposed drug against concurrent medications
    fn check_all(
        &self,
        proposed_drug_id: &str,
        concurrent_meds: &[ConcurrentMed],
    ) -> Result<Vec<DdiFlag>, JitSafetyError>;

    /// Check interaction between two specific drugs
    fn check_pair(
        &self,
        drug_a: &str,
        drug_b: &str,
    ) -> Result<Option<DdiFlag>, JitSafetyError>;
}

/// Memory-backed DDI adapter for fast lookups
pub struct MemoryDdiAdapter {
    interactions: Vec<DdiInteraction>,
}

/// DDI interaction definition
#[derive(Debug, Clone)]
pub struct DdiInteraction {
    pub drug_a: String,
    pub drug_b: String,
    pub severity: String,    // "minor", "moderate", "major", "contraindicated"
    pub action: String,
    pub code: String,
    pub rule_id: String,
    pub description: String,
}

impl MemoryDdiAdapter {
    /// Create a new memory-backed DDI adapter
    pub fn new() -> Self {
        Self {
            interactions: Self::load_default_interactions(),
        }
    }

    /// Create adapter with custom interactions
    pub fn with_interactions(interactions: Vec<DdiInteraction>) -> Self {
        Self { interactions }
    }

    /// Load default DDI interactions for testing/demo
    fn load_default_interactions() -> Vec<DdiInteraction> {
        vec![
            // ACE Inhibitor + ARB interaction
            DdiInteraction {
                drug_a: "lisinopril".to_string(),
                drug_b: "losartan".to_string(),
                severity: "major".to_string(),
                action: "avoid duplicate RAAS blockade".to_string(),
                code: "DDI-ACEI-ARB".to_string(),
                rule_id: "DDI-RAAS-001".to_string(),
                description: "Concurrent use of ACE inhibitor and ARB may increase risk of hyperkalemia, hypotension, and renal dysfunction".to_string(),
            },
            // ACE Inhibitor + ARNI interaction (contraindicated)
            DdiInteraction {
                drug_a: "lisinopril".to_string(),
                drug_b: "sacubitril_valsartan".to_string(),
                severity: "contraindicated".to_string(),
                action: "block".to_string(),
                code: "DDI-ACEI-ARNI".to_string(),
                rule_id: "DDI-RAAS-002".to_string(),
                description: "Concurrent use of ACE inhibitor and ARNI is contraindicated due to increased risk of angioedema".to_string(),
            },
            // Metformin + Contrast interaction
            DdiInteraction {
                drug_a: "metformin".to_string(),
                drug_b: "iodinated_contrast".to_string(),
                severity: "major".to_string(),
                action: "hold metformin before and after contrast".to_string(),
                code: "DDI-METFORMIN-CONTRAST".to_string(),
                rule_id: "DDI-METFORMIN-001".to_string(),
                description: "Metformin should be held before contrast administration to prevent lactic acidosis".to_string(),
            },
            // Warfarin + Aspirin interaction
            DdiInteraction {
                drug_a: "warfarin".to_string(),
                drug_b: "aspirin".to_string(),
                severity: "major".to_string(),
                action: "monitor INR closely".to_string(),
                code: "DDI-WARFARIN-ASPIRIN".to_string(),
                rule_id: "DDI-ANTICOAG-001".to_string(),
                description: "Concurrent use increases bleeding risk - monitor INR and bleeding signs".to_string(),
            },
        ]
    }

    /// Find interaction between two drugs (bidirectional)
    fn find_interaction(&self, drug_a: &str, drug_b: &str) -> Option<&DdiInteraction> {
        self.interactions.iter().find(|interaction| {
            (interaction.drug_a == drug_a && interaction.drug_b == drug_b) ||
            (interaction.drug_a == drug_b && interaction.drug_b == drug_a)
        })
    }
}

impl DdiAdapter for MemoryDdiAdapter {
    fn check_all(
        &self,
        proposed_drug_id: &str,
        concurrent_meds: &[ConcurrentMed],
    ) -> Result<Vec<DdiFlag>, JitSafetyError> {
        debug!("Checking DDI for '{}' against {} concurrent medications", 
               proposed_drug_id, concurrent_meds.len());

        let mut ddi_flags = Vec::new();

        for concurrent_med in concurrent_meds {
            if let Some(interaction) = self.find_interaction(proposed_drug_id, &concurrent_med.drug_id) {
                let ddi_flag = DdiFlag {
                    with_drug_id: concurrent_med.drug_id.clone(),
                    severity: interaction.severity.clone(),
                    action: interaction.action.clone(),
                    code: interaction.code.clone(),
                    rule_id: interaction.rule_id.clone(),
                };

                debug!("DDI found: {} with {} (severity: {})", 
                       proposed_drug_id, concurrent_med.drug_id, interaction.severity);

                ddi_flags.push(ddi_flag);
            }
        }

        debug!("DDI check completed: {} interactions found", ddi_flags.len());
        Ok(ddi_flags)
    }

    fn check_pair(
        &self,
        drug_a: &str,
        drug_b: &str,
    ) -> Result<Option<DdiFlag>, JitSafetyError> {
        debug!("Checking DDI pair: '{}' vs '{}'", drug_a, drug_b);

        if let Some(interaction) = self.find_interaction(drug_a, drug_b) {
            let ddi_flag = DdiFlag {
                with_drug_id: drug_b.to_string(),
                severity: interaction.severity.clone(),
                action: interaction.action.clone(),
                code: interaction.code.clone(),
                rule_id: interaction.rule_id.clone(),
            };

            debug!("DDI pair found: {} (severity: {})", interaction.code, interaction.severity);
            Ok(Some(ddi_flag))
        } else {
            debug!("No DDI found between '{}' and '{}'", drug_a, drug_b);
            Ok(None)
        }
    }
}

/// External DDI service adapter (for integration with existing systems)
pub struct ExternalDdiAdapter {
    service_url: String,
    timeout_ms: u64,
}

impl ExternalDdiAdapter {
    pub fn new(service_url: impl Into<String>, timeout_ms: u64) -> Self {
        Self {
            service_url: service_url.into(),
            timeout_ms,
        }
    }
}

impl DdiAdapter for ExternalDdiAdapter {
    fn check_all(
        &self,
        proposed_drug_id: &str,
        concurrent_meds: &[ConcurrentMed],
    ) -> Result<Vec<DdiFlag>, JitSafetyError> {
        // In a real implementation, this would make HTTP calls to external DDI service
        // For now, return empty result with warning
        warn!("External DDI service not implemented - using empty result");
        debug!("Would call DDI service at {} for drug '{}' vs {} concurrent meds", 
               self.service_url, proposed_drug_id, concurrent_meds.len());
        
        Ok(Vec::new())
    }

    fn check_pair(
        &self,
        drug_a: &str,
        drug_b: &str,
    ) -> Result<Option<DdiFlag>, JitSafetyError> {
        // In a real implementation, this would make HTTP calls to external DDI service
        warn!("External DDI service not implemented - using empty result");
        debug!("Would call DDI service at {} for pair '{}' vs '{}'", 
               self.service_url, drug_a, drug_b);
        
        Ok(None)
    }
}

/// Mock DDI adapter for testing
#[cfg(test)]
pub struct MockDdiAdapter {
    should_fail: bool,
    interactions: Vec<DdiFlag>,
}

#[cfg(test)]
impl MockDdiAdapter {
    pub fn new() -> Self {
        Self {
            should_fail: false,
            interactions: Vec::new(),
        }
    }

    pub fn with_failure() -> Self {
        Self {
            should_fail: true,
            interactions: Vec::new(),
        }
    }

    pub fn with_interactions(interactions: Vec<DdiFlag>) -> Self {
        Self {
            should_fail: false,
            interactions,
        }
    }
}

#[cfg(test)]
impl DdiAdapter for MockDdiAdapter {
    fn check_all(
        &self,
        _proposed_drug_id: &str,
        _concurrent_meds: &[ConcurrentMed],
    ) -> Result<Vec<DdiFlag>, JitSafetyError> {
        if self.should_fail {
            Err(JitSafetyError::ddi_error("Mock DDI adapter failure"))
        } else {
            Ok(self.interactions.clone())
        }
    }

    fn check_pair(
        &self,
        _drug_a: &str,
        _drug_b: &str,
    ) -> Result<Option<DdiFlag>, JitSafetyError> {
        if self.should_fail {
            Err(JitSafetyError::ddi_error("Mock DDI adapter failure"))
        } else {
            Ok(self.interactions.first().cloned())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_memory_ddi_adapter() {
        let adapter = MemoryDdiAdapter::new();
        
        let concurrent_meds = vec![
            ConcurrentMed {
                drug_id: "losartan".to_string(),
                class_id: "ARB".to_string(),
                dose_mg: 50.0,
                interval_h: 24,
            }
        ];

        let result = adapter.check_all("lisinopril", &concurrent_meds);
        assert!(result.is_ok());
        
        let flags = result.unwrap();
        assert_eq!(flags.len(), 1);
        assert_eq!(flags[0].code, "DDI-ACEI-ARB");
        assert_eq!(flags[0].severity, "major");
    }

    #[test]
    fn test_no_interactions() {
        let adapter = MemoryDdiAdapter::new();
        
        let concurrent_meds = vec![
            ConcurrentMed {
                drug_id: "acetaminophen".to_string(),
                class_id: "ANALGESIC".to_string(),
                dose_mg: 500.0,
                interval_h: 6,
            }
        ];

        let result = adapter.check_all("lisinopril", &concurrent_meds);
        assert!(result.is_ok());
        
        let flags = result.unwrap();
        assert_eq!(flags.len(), 0);
    }

    #[test]
    fn test_contraindicated_interaction() {
        let adapter = MemoryDdiAdapter::new();
        
        let concurrent_meds = vec![
            ConcurrentMed {
                drug_id: "sacubitril_valsartan".to_string(),
                class_id: "ARNI".to_string(),
                dose_mg: 49.0,
                interval_h: 12,
            }
        ];

        let result = adapter.check_all("lisinopril", &concurrent_meds);
        assert!(result.is_ok());
        
        let flags = result.unwrap();
        assert_eq!(flags.len(), 1);
        assert_eq!(flags[0].code, "DDI-ACEI-ARNI");
        assert_eq!(flags[0].severity, "contraindicated");
    }

    #[test]
    fn test_bidirectional_interaction() {
        let adapter = MemoryDdiAdapter::new();
        
        // Test both directions of the same interaction
        let result1 = adapter.check_pair("lisinopril", "losartan");
        let result2 = adapter.check_pair("losartan", "lisinopril");
        
        assert!(result1.is_ok());
        assert!(result2.is_ok());
        
        let flag1 = result1.unwrap();
        let flag2 = result2.unwrap();
        
        assert!(flag1.is_some());
        assert!(flag2.is_some());
        
        assert_eq!(flag1.as_ref().unwrap().code, flag2.as_ref().unwrap().code);
    }
}
