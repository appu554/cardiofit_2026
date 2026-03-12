//! Cumulative Risk Assessment - Comprehensive polypharmacy safety analysis
//! 
//! This module implements sophisticated risk assessment for patients on multiple medications,
//! analyzing drug interactions, cumulative effects, and temporal risk patterns.

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use chrono::{DateTime, Utc, Duration};
use tracing::{info, warn, error, debug};

/// Cumulative risk assessment engine
#[derive(Debug)]
pub struct CumulativeRiskAssessment {
    risk_models: HashMap<String, Box<dyn RiskModel>>,
    interaction_matrix: InteractionMatrix,
    population_data: PopulationRiskData,
    config: RiskAssessmentConfig,
}

/// Risk assessment configuration
#[derive(Debug, Clone)]
pub struct RiskAssessmentConfig {
    pub enable_drug_interactions: bool,
    pub enable_temporal_analysis: bool,
    pub enable_population_comparison: bool,
    pub confidence_threshold: f64,
    pub max_risk_factors: u32,
    pub include_minor_interactions: bool,
}

impl Default for RiskAssessmentConfig {
    fn default() -> Self {
        Self {
            enable_drug_interactions: true,
            enable_temporal_analysis: true,
            enable_population_comparison: true,
            confidence_threshold: 0.8,
            max_risk_factors: 50,
            include_minor_interactions: false,
        }
    }
}

/// Comprehensive risk profile for a patient
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CumulativeRiskProfile {
    pub patient_id: String,
    pub assessment_date: DateTime<Utc>,
    pub overall_risk_score: f64,
    pub risk_level: RiskLevel,
    pub confidence_interval: (f64, f64),
    pub risk_factors: Vec<RiskFactor>,
    pub drug_interactions: Vec<DrugInteractionRisk>,
    pub temporal_patterns: Vec<TemporalRiskPattern>,
    pub population_comparison: PopulationComparison,
    pub mitigation_strategies: Vec<MitigationStrategy>,
    pub monitoring_recommendations: Vec<MonitoringRecommendation>,
}

/// Risk level categories
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum RiskLevel {
    Low,
    Moderate,
    High,
    VeryHigh,
    Critical,
}

/// Individual risk factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    pub factor_id: String,
    pub factor_type: RiskFactorType,
    pub description: String,
    pub risk_score: f64,
    pub confidence: f64,
    pub evidence_level: EvidenceLevel,
    pub contributing_medications: Vec<String>,
    pub patient_specific_modifiers: Vec<PatientModifier>,
}

/// Types of risk factors
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskFactorType {
    DrugInteraction,
    CumulativeEffect,
    OrganToxicity,
    MetabolicInterference,
    PharmacokineticInteraction,
    PharmacodynamicInteraction,
    AdverseEventRisk,
    TherapeuticFailure,
}

/// Evidence levels for risk factors
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceLevel {
    HighQuality,
    ModerateQuality,
    LowQuality,
    ExpertOpinion,
    Theoretical,
}

/// Patient-specific risk modifiers
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientModifier {
    pub modifier_type: ModifierType,
    pub description: String,
    pub risk_multiplier: f64,
    pub rationale: String,
}

/// Types of patient modifiers
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ModifierType {
    Age,
    RenalFunction,
    HepaticFunction,
    Genetics,
    Comorbidity,
    Adherence,
    Lifestyle,
}

/// Drug interaction risk
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugInteractionRisk {
    pub interaction_id: String,
    pub drug_a: String,
    pub drug_b: String,
    pub interaction_type: InteractionType,
    pub severity: InteractionSeverity,
    pub mechanism: String,
    pub clinical_effect: String,
    pub risk_score: f64,
    pub onset_time: Option<String>,
    pub duration: Option<String>,
    pub management_strategy: String,
}

/// Types of drug interactions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum InteractionType {
    Pharmacokinetic,
    Pharmacodynamic,
    Synergistic,
    Antagonistic,
    Additive,
}

/// Interaction severity levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum InteractionSeverity {
    Minor,
    Moderate,
    Major,
    Contraindicated,
}

/// Temporal risk pattern
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemporalRiskPattern {
    pub pattern_id: String,
    pub pattern_type: TemporalPatternType,
    pub description: String,
    pub time_window: String,
    pub risk_evolution: Vec<TimePoint>,
    pub critical_periods: Vec<CriticalPeriod>,
    pub intervention_windows: Vec<InterventionWindow>,
}

/// Types of temporal patterns
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TemporalPatternType {
    Accumulation,
    Tolerance,
    Sensitization,
    Cyclical,
    Progressive,
}

/// Risk at specific time point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimePoint {
    pub time_offset_hours: u32,
    pub risk_score: f64,
    pub contributing_factors: Vec<String>,
}

/// Critical period with elevated risk
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CriticalPeriod {
    pub start_time: String,
    pub end_time: String,
    pub risk_elevation: f64,
    pub reason: String,
    pub monitoring_intensity: String,
}

/// Window for intervention
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InterventionWindow {
    pub window_start: String,
    pub window_end: String,
    pub intervention_type: String,
    pub expected_benefit: f64,
    pub urgency: String,
}

/// Population comparison data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PopulationComparison {
    pub patient_percentile: f64,
    pub cohort_description: String,
    pub similar_patients_count: u32,
    pub average_risk_score: f64,
    pub risk_distribution: Vec<RiskBin>,
    pub outcome_predictions: Vec<OutcomePrediction>,
}

/// Risk distribution bin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskBin {
    pub risk_range: (f64, f64),
    pub patient_count: u32,
    pub percentage: f64,
}

/// Outcome prediction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OutcomePrediction {
    pub outcome_type: String,
    pub probability: f64,
    pub time_horizon: String,
    pub confidence_interval: (f64, f64),
}

/// Mitigation strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MitigationStrategy {
    pub strategy_id: String,
    pub strategy_type: MitigationType,
    pub description: String,
    pub target_risk_factors: Vec<String>,
    pub expected_risk_reduction: f64,
    pub implementation_complexity: String,
    pub cost_effectiveness: Option<f64>,
    pub evidence_support: EvidenceLevel,
}

/// Types of mitigation strategies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MitigationType {
    DoseAdjustment,
    DrugSubstitution,
    TimingModification,
    AdditionalMonitoring,
    SupportiveTherapy,
    LifestyleModification,
    EducationalIntervention,
}

/// Monitoring recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRecommendation {
    pub monitoring_type: String,
    pub parameter: String,
    pub frequency: String,
    pub target_range: Option<(f64, f64)>,
    pub alert_thresholds: Option<(f64, f64)>,
    pub rationale: String,
    pub priority: MonitoringPriority,
}

/// Monitoring priority levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MonitoringPriority {
    Low,
    Medium,
    High,
    Critical,
}

/// Drug interaction matrix
#[derive(Debug, Clone)]
pub struct InteractionMatrix {
    interactions: HashMap<(String, String), InteractionData>,
}

/// Interaction data in matrix
#[derive(Debug, Clone)]
pub struct InteractionData {
    pub severity: InteractionSeverity,
    pub mechanism: String,
    pub risk_score: f64,
    pub evidence_level: EvidenceLevel,
}

/// Population risk data
#[derive(Debug, Clone)]
pub struct PopulationRiskData {
    pub cohorts: HashMap<String, CohortData>,
}

/// Cohort data for population comparison
#[derive(Debug, Clone)]
pub struct CohortData {
    pub cohort_id: String,
    pub description: String,
    pub patient_count: u32,
    pub risk_statistics: RiskStatistics,
    pub outcome_rates: HashMap<String, f64>,
}

/// Risk statistics for a cohort
#[derive(Debug, Clone)]
pub struct RiskStatistics {
    pub mean_risk_score: f64,
    pub median_risk_score: f64,
    pub std_deviation: f64,
    pub percentiles: HashMap<u8, f64>,
}

/// Risk model trait
pub trait RiskModel: Send + Sync + std::fmt::Debug {
    fn calculate_risk(&self, context: &RiskAssessmentContext) -> Result<f64>;
    fn model_name(&self) -> &str;
    fn applicable_risk_types(&self) -> Vec<RiskFactorType>;
}

/// Risk assessment context
#[derive(Debug, Clone)]
pub struct RiskAssessmentContext {
    pub patient_id: String,
    pub medications: Vec<MedicationContext>,
    pub patient_factors: PatientRiskFactors,
    pub laboratory_values: HashMap<String, f64>,
    pub comorbidities: Vec<String>,
    pub assessment_timepoint: DateTime<Utc>,
}

/// Medication context for risk assessment
#[derive(Debug, Clone)]
pub struct MedicationContext {
    pub drug_id: String,
    pub drug_name: String,
    pub dose: f64,
    pub frequency: String,
    pub route: String,
    pub start_date: DateTime<Utc>,
    pub duration: Option<Duration>,
    pub indication: String,
}

/// Patient risk factors
#[derive(Debug, Clone)]
pub struct PatientRiskFactors {
    pub age_years: f64,
    pub weight_kg: f64,
    pub renal_function: f64, // eGFR
    pub hepatic_function: String, // Child-Pugh class
    pub genetic_variants: Vec<String>,
    pub smoking_status: String,
    pub alcohol_use: String,
    pub adherence_score: f64,
}

impl CumulativeRiskAssessment {
    /// Create a new cumulative risk assessment engine
    pub fn new() -> Self {
        let mut assessment = Self {
            risk_models: HashMap::new(),
            interaction_matrix: InteractionMatrix::new(),
            population_data: PopulationRiskData::new(),
            config: RiskAssessmentConfig::default(),
        };
        
        // Register default risk models
        assessment.register_default_models();
        
        info!("⚠️ Cumulative Risk Assessment initialized");
        info!("🧮 Risk models: {}", assessment.risk_models.len());
        
        assessment
    }
    
    /// Register default risk models
    fn register_default_models(&mut self) {
        self.risk_models.insert("drug_interaction".to_string(), Box::new(DrugInteractionRiskModel));
        self.risk_models.insert("cumulative_effect".to_string(), Box::new(CumulativeEffectRiskModel));
        self.risk_models.insert("organ_toxicity".to_string(), Box::new(OrganToxicityRiskModel));
        self.risk_models.insert("metabolic_interference".to_string(), Box::new(MetabolicInterferenceRiskModel));
    }
    
    /// Perform comprehensive risk assessment
    pub async fn assess_cumulative_risk(
        &self,
        context: RiskAssessmentContext,
    ) -> Result<CumulativeRiskProfile> {
        info!("🔍 Performing cumulative risk assessment for patient: {}", context.patient_id);
        
        let mut risk_factors = Vec::new();
        let mut total_risk_score = 0.0;
        
        // Run all risk models
        for (model_name, model) in &self.risk_models {
            match model.calculate_risk(&context) {
                Ok(risk_score) => {
                    if risk_score > 0.0 {
                        risk_factors.push(RiskFactor {
                            factor_id: format!("{}_{}", model_name, uuid::Uuid::new_v4()),
                            factor_type: model.applicable_risk_types()[0].clone(),
                            description: format!("Risk from {}", model_name),
                            risk_score,
                            confidence: 0.8,
                            evidence_level: EvidenceLevel::ModerateQuality,
                            contributing_medications: context.medications.iter()
                                .map(|m| m.drug_name.clone())
                                .collect(),
                            patient_specific_modifiers: vec![],
                        });
                        total_risk_score += risk_score;
                    }
                }
                Err(e) => {
                    warn!("Risk model {} failed: {}", model_name, e);
                }
            }
        }
        
        // Analyze drug interactions
        let drug_interactions = self.analyze_drug_interactions(&context).await?;
        
        // Analyze temporal patterns
        let temporal_patterns = if self.config.enable_temporal_analysis {
            self.analyze_temporal_patterns(&context).await?
        } else {
            vec![]
        };
        
        // Population comparison
        let population_comparison = if self.config.enable_population_comparison {
            self.perform_population_comparison(&context, total_risk_score).await?
        } else {
            PopulationComparison {
                patient_percentile: 50.0,
                cohort_description: "Not available".to_string(),
                similar_patients_count: 0,
                average_risk_score: 0.0,
                risk_distribution: vec![],
                outcome_predictions: vec![],
            }
        };
        
        // Determine risk level
        let risk_level = self.determine_risk_level(total_risk_score);
        
        // Generate mitigation strategies
        let mitigation_strategies = self.generate_mitigation_strategies(&risk_factors, &context).await?;
        
        // Generate monitoring recommendations
        let monitoring_recommendations = self.generate_monitoring_recommendations(&risk_factors, &context).await?;
        
        let profile = CumulativeRiskProfile {
            patient_id: context.patient_id.clone(),
            assessment_date: Utc::now(),
            overall_risk_score: total_risk_score,
            risk_level,
            confidence_interval: (total_risk_score * 0.8, total_risk_score * 1.2),
            risk_factors,
            drug_interactions,
            temporal_patterns,
            population_comparison,
            mitigation_strategies,
            monitoring_recommendations,
        };
        
        info!("✅ Risk assessment completed: {} (Risk Level: {:?})", 
              context.patient_id, profile.risk_level);
        
        Ok(profile)
    }
    
    /// Analyze drug interactions
    async fn analyze_drug_interactions(&self, context: &RiskAssessmentContext) -> Result<Vec<DrugInteractionRisk>> {
        let mut interactions = Vec::new();
        
        // Check all pairs of medications
        for i in 0..context.medications.len() {
            for j in (i + 1)..context.medications.len() {
                let drug_a = &context.medications[i];
                let drug_b = &context.medications[j];
                
                if let Some(interaction) = self.interaction_matrix.get_interaction(&drug_a.drug_id, &drug_b.drug_id) {
                    interactions.push(DrugInteractionRisk {
                        interaction_id: format!("{}_{}", drug_a.drug_id, drug_b.drug_id),
                        drug_a: drug_a.drug_name.clone(),
                        drug_b: drug_b.drug_name.clone(),
                        interaction_type: InteractionType::Pharmacokinetic,
                        severity: interaction.severity.clone(),
                        mechanism: interaction.mechanism.clone(),
                        clinical_effect: "Potential interaction".to_string(),
                        risk_score: interaction.risk_score,
                        onset_time: Some("Variable".to_string()),
                        duration: Some("While both drugs are used".to_string()),
                        management_strategy: "Monitor closely".to_string(),
                    });
                }
            }
        }
        
        Ok(interactions)
    }
    
    /// Analyze temporal risk patterns
    async fn analyze_temporal_patterns(&self, _context: &RiskAssessmentContext) -> Result<Vec<TemporalRiskPattern>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    /// Perform population comparison
    async fn perform_population_comparison(
        &self,
        _context: &RiskAssessmentContext,
        risk_score: f64,
    ) -> Result<PopulationComparison> {
        // Simplified implementation
        let percentile = if risk_score < 0.3 { 25.0 } else if risk_score < 0.6 { 50.0 } else { 75.0 };
        
        Ok(PopulationComparison {
            patient_percentile: percentile,
            cohort_description: "Similar age and medication count".to_string(),
            similar_patients_count: 1000,
            average_risk_score: 0.4,
            risk_distribution: vec![
                RiskBin { risk_range: (0.0, 0.3), patient_count: 250, percentage: 25.0 },
                RiskBin { risk_range: (0.3, 0.6), patient_count: 500, percentage: 50.0 },
                RiskBin { risk_range: (0.6, 1.0), patient_count: 250, percentage: 25.0 },
            ],
            outcome_predictions: vec![],
        })
    }
    
    /// Determine risk level from score
    fn determine_risk_level(&self, risk_score: f64) -> RiskLevel {
        match risk_score {
            s if s < 0.2 => RiskLevel::Low,
            s if s < 0.4 => RiskLevel::Moderate,
            s if s < 0.7 => RiskLevel::High,
            s if s < 0.9 => RiskLevel::VeryHigh,
            _ => RiskLevel::Critical,
        }
    }
    
    /// Generate mitigation strategies
    async fn generate_mitigation_strategies(
        &self,
        risk_factors: &[RiskFactor],
        _context: &RiskAssessmentContext,
    ) -> Result<Vec<MitigationStrategy>> {
        let mut strategies = Vec::new();
        
        for risk_factor in risk_factors {
            if risk_factor.risk_score > 0.5 {
                strategies.push(MitigationStrategy {
                    strategy_id: format!("mitigation_{}", uuid::Uuid::new_v4()),
                    strategy_type: MitigationType::AdditionalMonitoring,
                    description: format!("Enhanced monitoring for {}", risk_factor.description),
                    target_risk_factors: vec![risk_factor.factor_id.clone()],
                    expected_risk_reduction: 0.2,
                    implementation_complexity: "Low".to_string(),
                    cost_effectiveness: Some(0.8),
                    evidence_support: EvidenceLevel::ModerateQuality,
                });
            }
        }
        
        Ok(strategies)
    }
    
    /// Generate monitoring recommendations
    async fn generate_monitoring_recommendations(
        &self,
        risk_factors: &[RiskFactor],
        _context: &RiskAssessmentContext,
    ) -> Result<Vec<MonitoringRecommendation>> {
        let mut recommendations = Vec::new();
        
        for risk_factor in risk_factors {
            if risk_factor.risk_score > 0.3 {
                recommendations.push(MonitoringRecommendation {
                    monitoring_type: "Laboratory".to_string(),
                    parameter: "Comprehensive metabolic panel".to_string(),
                    frequency: "Weekly".to_string(),
                    target_range: None,
                    alert_thresholds: None,
                    rationale: format!("Monitor for {}", risk_factor.description),
                    priority: if risk_factor.risk_score > 0.7 { 
                        MonitoringPriority::High 
                    } else { 
                        MonitoringPriority::Medium 
                    },
                });
            }
        }
        
        Ok(recommendations)
    }
}

impl InteractionMatrix {
    fn new() -> Self {
        Self {
            interactions: HashMap::new(),
        }
    }
    
    fn get_interaction(&self, drug_a: &str, drug_b: &str) -> Option<&InteractionData> {
        self.interactions.get(&(drug_a.to_string(), drug_b.to_string()))
            .or_else(|| self.interactions.get(&(drug_b.to_string(), drug_a.to_string())))
    }
}

impl PopulationRiskData {
    fn new() -> Self {
        Self {
            cohorts: HashMap::new(),
        }
    }
}

// Default risk model implementations
#[derive(Debug)]
struct DrugInteractionRiskModel;

impl RiskModel for DrugInteractionRiskModel {
    fn calculate_risk(&self, context: &RiskAssessmentContext) -> Result<f64> {
        // Simple risk calculation based on number of medications
        let medication_count = context.medications.len() as f64;
        let base_risk = (medication_count - 1.0) * 0.1;
        Ok(base_risk.min(1.0).max(0.0))
    }
    
    fn model_name(&self) -> &str {
        "DrugInteractionRiskModel"
    }
    
    fn applicable_risk_types(&self) -> Vec<RiskFactorType> {
        vec![RiskFactorType::DrugInteraction]
    }
}

#[derive(Debug)]
struct CumulativeEffectRiskModel;

impl RiskModel for CumulativeEffectRiskModel {
    fn calculate_risk(&self, context: &RiskAssessmentContext) -> Result<f64> {
        // Risk based on cumulative effects
        let age_factor = if context.patient_factors.age_years > 65.0 { 0.2 } else { 0.0 };
        let medication_factor = context.medications.len() as f64 * 0.05;
        Ok((age_factor + medication_factor).min(1.0))
    }
    
    fn model_name(&self) -> &str {
        "CumulativeEffectRiskModel"
    }
    
    fn applicable_risk_types(&self) -> Vec<RiskFactorType> {
        vec![RiskFactorType::CumulativeEffect]
    }
}

#[derive(Debug)]
struct OrganToxicityRiskModel;

impl RiskModel for OrganToxicityRiskModel {
    fn calculate_risk(&self, context: &RiskAssessmentContext) -> Result<f64> {
        // Risk based on organ function
        let renal_risk: f64 = if context.patient_factors.renal_function < 60.0 { 0.3 } else { 0.0 };
        let hepatic_risk: f64 = if context.patient_factors.hepatic_function != "A" { 0.2 } else { 0.0 };
        Ok((renal_risk + hepatic_risk).min(1.0))
    }
    
    fn model_name(&self) -> &str {
        "OrganToxicityRiskModel"
    }
    
    fn applicable_risk_types(&self) -> Vec<RiskFactorType> {
        vec![RiskFactorType::OrganToxicity]
    }
}

#[derive(Debug)]
struct MetabolicInterferenceRiskModel;

impl RiskModel for MetabolicInterferenceRiskModel {
    fn calculate_risk(&self, _context: &RiskAssessmentContext) -> Result<f64> {
        // Placeholder implementation
        Ok(0.1)
    }
    
    fn model_name(&self) -> &str {
        "MetabolicInterferenceRiskModel"
    }
    
    fn applicable_risk_types(&self) -> Vec<RiskFactorType> {
        vec![RiskFactorType::MetabolicInterference]
    }
}
