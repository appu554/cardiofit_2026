package com.cardiofit.flink.models;

import com.cardiofit.flink.protocols.ProtocolMatcher;
import com.fasterxml.jackson.annotation.JsonInclude;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Semantic Enrichment Container
 *
 * Holds clinical knowledge enrichment data from Module 3 Semantic Mesh.
 * This data augments the basic patient state with protocol recommendations,
 * drug interactions, evidence citations, and CEP pattern flags.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-10-28
 */
@JsonInclude(JsonInclude.Include.ALWAYS)
public class SemanticEnrichment implements Serializable {
    private static final long serialVersionUID = 1L;

    private Long enrichmentTimestamp;
    private String enrichmentVersion = "1.0.0";
    private String knowledgeBaseVersion;

    // Clinical Protocol Matching
    private List<MatchedProtocolDetail> matchedProtocols;

    // Drug Interaction Analysis
    private DrugInteractionAnalysis drugInteractionAnalysis;

    // Clinical Thresholds
    private Map<String, ClinicalThreshold> clinicalThresholds;

    // Care Pathway Recommendations
    private CarePathwayRecommendations carePathwayRecommendations;

    // Evidence-Based Alerts
    private List<EvidenceBasedAlert> evidenceBasedAlerts;

    // CEP Pattern Flags
    private Map<String, CEPPatternFlag> cepPatternFlags;

    // Semantic Tags
    private List<String> semanticTags;

    // Knowledge Base Sources
    private List<KnowledgeBaseSource> knowledgeBaseSources;

    public SemanticEnrichment() {
        this.enrichmentTimestamp = System.currentTimeMillis();
        this.matchedProtocols = new ArrayList<>();
        this.clinicalThresholds = new HashMap<>();
        this.evidenceBasedAlerts = new ArrayList<>();
        this.cepPatternFlags = new HashMap<>();
        this.semanticTags = new ArrayList<>();
        this.knowledgeBaseSources = new ArrayList<>();
    }

    // ========== INNER CLASSES ==========

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class MatchedProtocolDetail implements Serializable {
        private static final long serialVersionUID = 1L;

        private String protocolId;
        private String protocolName;
        private String category;
        private Double matchConfidence;
        private String matchReason;
        private List<String> triggerCriteria;
        private List<RecommendedAction> recommendedActions;
        private Map<String, String> escalationCriteria;

        public MatchedProtocolDetail() {
            this.triggerCriteria = new ArrayList<>();
            this.recommendedActions = new ArrayList<>();
            this.escalationCriteria = new HashMap<>();
        }

        // Getters and setters
        public String getProtocolId() { return protocolId; }
        public void setProtocolId(String protocolId) { this.protocolId = protocolId; }
        public String getProtocolName() { return protocolName; }
        public void setProtocolName(String protocolName) { this.protocolName = protocolName; }
        public String getCategory() { return category; }
        public void setCategory(String category) { this.category = category; }
        public Double getMatchConfidence() { return matchConfidence; }
        public void setMatchConfidence(Double matchConfidence) { this.matchConfidence = matchConfidence; }
        public String getMatchReason() { return matchReason; }
        public void setMatchReason(String matchReason) { this.matchReason = matchReason; }
        public List<String> getTriggerCriteria() { return triggerCriteria; }
        public void setTriggerCriteria(List<String> triggerCriteria) { this.triggerCriteria = triggerCriteria; }
        public List<RecommendedAction> getRecommendedActions() { return recommendedActions; }
        public void setRecommendedActions(List<RecommendedAction> recommendedActions) { this.recommendedActions = recommendedActions; }
        public Map<String, String> getEscalationCriteria() { return escalationCriteria; }
        public void setEscalationCriteria(Map<String, String> escalationCriteria) { this.escalationCriteria = escalationCriteria; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class RecommendedAction implements Serializable {
        private static final long serialVersionUID = 1L;

        private Integer priority;
        private String action;
        private String timeframe;
        private String evidenceCitation;
        private String evidenceLevel;

        // Getters and setters
        public Integer getPriority() { return priority; }
        public void setPriority(Integer priority) { this.priority = priority; }
        public String getAction() { return action; }
        public void setAction(String action) { this.action = action; }
        public String getTimeframe() { return timeframe; }
        public void setTimeframe(String timeframe) { this.timeframe = timeframe; }
        public String getEvidenceCitation() { return evidenceCitation; }
        public void setEvidenceCitation(String evidenceCitation) { this.evidenceCitation = evidenceCitation; }
        public String getEvidenceLevel() { return evidenceLevel; }
        public void setEvidenceLevel(String evidenceLevel) { this.evidenceLevel = evidenceLevel; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class DrugInteractionAnalysis implements Serializable {
        private static final long serialVersionUID = 1L;

        private Integer currentMedicationsAnalyzed;
        private Integer interactionsDetected;
        private Integer contraindicationsDetected;
        private Integer renalDoseAdjustmentsNeeded;
        private List<MedicationDetail> details;
        private List<InteractionWarning> interactionWarnings;

        public DrugInteractionAnalysis() {
            this.details = new ArrayList<>();
            this.interactionWarnings = new ArrayList<>();
        }

        // Getters and setters
        public Integer getCurrentMedicationsAnalyzed() { return currentMedicationsAnalyzed; }
        public void setCurrentMedicationsAnalyzed(Integer currentMedicationsAnalyzed) { this.currentMedicationsAnalyzed = currentMedicationsAnalyzed; }
        public Integer getInteractionsDetected() { return interactionsDetected; }
        public void setInteractionsDetected(Integer interactionsDetected) { this.interactionsDetected = interactionsDetected; }
        public Integer getContraindicationsDetected() { return contraindicationsDetected; }
        public void setContraindicationsDetected(Integer contraindicationsDetected) { this.contraindicationsDetected = contraindicationsDetected; }
        public Integer getRenalDoseAdjustmentsNeeded() { return renalDoseAdjustmentsNeeded; }
        public void setRenalDoseAdjustmentsNeeded(Integer renalDoseAdjustmentsNeeded) { this.renalDoseAdjustmentsNeeded = renalDoseAdjustmentsNeeded; }
        public List<MedicationDetail> getDetails() { return details; }
        public void setDetails(List<MedicationDetail> details) { this.details = details; }
        public List<InteractionWarning> getInteractionWarnings() { return interactionWarnings; }
        public void setInteractionWarnings(List<InteractionWarning> interactionWarnings) { this.interactionWarnings = interactionWarnings; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class InteractionWarning implements Serializable {
        private static final long serialVersionUID = 1L;

        private String protocolMedication;
        private String activeMedication;
        private String severity;
        private String clinicalEffect;
        private String management;
        private String onset;
        private Boolean blackBoxWarning;
        private List<String> evidencePMIDs;

        public InteractionWarning() {}

        // Getters and setters
        public String getProtocolMedication() { return protocolMedication; }
        public void setProtocolMedication(String protocolMedication) { this.protocolMedication = protocolMedication; }
        public String getActiveMedication() { return activeMedication; }
        public void setActiveMedication(String activeMedication) { this.activeMedication = activeMedication; }
        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }
        public String getClinicalEffect() { return clinicalEffect; }
        public void setClinicalEffect(String clinicalEffect) { this.clinicalEffect = clinicalEffect; }
        public String getManagement() { return management; }
        public void setManagement(String management) { this.management = management; }
        public String getOnset() { return onset; }
        public void setOnset(String onset) { this.onset = onset; }
        public Boolean getBlackBoxWarning() { return blackBoxWarning; }
        public void setBlackBoxWarning(Boolean blackBoxWarning) { this.blackBoxWarning = blackBoxWarning; }
        public List<String> getEvidencePMIDs() { return evidencePMIDs; }
        public void setEvidencePMIDs(List<String> evidencePMIDs) { this.evidencePMIDs = evidencePMIDs; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class MedicationDetail implements Serializable {
        private static final long serialVersionUID = 1L;

        private String medicationCode;
        private String medicationName;
        private String therapeuticClass;
        private List<String> contraindications;
        private List<String> monitoringRequired;

        public MedicationDetail() {
            this.contraindications = new ArrayList<>();
            this.monitoringRequired = new ArrayList<>();
        }

        // Getters and setters
        public String getMedicationCode() { return medicationCode; }
        public void setMedicationCode(String medicationCode) { this.medicationCode = medicationCode; }
        public String getMedicationName() { return medicationName; }
        public void setMedicationName(String medicationName) { this.medicationName = medicationName; }
        public String getTherapeuticClass() { return therapeuticClass; }
        public void setTherapeuticClass(String therapeuticClass) { this.therapeuticClass = therapeuticClass; }
        public List<String> getContraindications() { return contraindications; }
        public void setContraindications(List<String> contraindications) { this.contraindications = contraindications; }
        public List<String> getMonitoringRequired() { return monitoringRequired; }
        public void setMonitoringRequired(List<String> monitoringRequired) { this.monitoringRequired = monitoringRequired; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class ClinicalThreshold implements Serializable {
        private static final long serialVersionUID = 1L;

        private String normal;
        private String elevated;
        private String critical;
        private Object currentValue;
        private String currentCategory;
        private String clinicalSignificance;
        private String evidenceCitation;

        // Getters and setters
        public String getNormal() { return normal; }
        public void setNormal(String normal) { this.normal = normal; }
        public String getElevated() { return elevated; }
        public void setElevated(String elevated) { this.elevated = elevated; }
        public String getCritical() { return critical; }
        public void setCritical(String critical) { this.critical = critical; }
        public Object getCurrentValue() { return currentValue; }
        public void setCurrentValue(Object currentValue) { this.currentValue = currentValue; }
        public String getCurrentCategory() { return currentCategory; }
        public void setCurrentCategory(String currentCategory) { this.currentCategory = currentCategory; }
        public String getClinicalSignificance() { return clinicalSignificance; }
        public void setClinicalSignificance(String clinicalSignificance) { this.clinicalSignificance = clinicalSignificance; }
        public String getEvidenceCitation() { return evidenceCitation; }
        public void setEvidenceCitation(String evidenceCitation) { this.evidenceCitation = evidenceCitation; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class CarePathwayRecommendations implements Serializable {
        private static final long serialVersionUID = 1L;

        private String primaryPathway;
        private List<String> alternativePathways;
        private Map<String, Object> bundleCompliance;

        public CarePathwayRecommendations() {
            this.alternativePathways = new ArrayList<>();
            this.bundleCompliance = new HashMap<>();
        }

        // Getters and setters
        public String getPrimaryPathway() { return primaryPathway; }
        public void setPrimaryPathway(String primaryPathway) { this.primaryPathway = primaryPathway; }
        public List<String> getAlternativePathways() { return alternativePathways; }
        public void setAlternativePathways(List<String> alternativePathways) { this.alternativePathways = alternativePathways; }
        public Map<String, Object> getBundleCompliance() { return bundleCompliance; }
        public void setBundleCompliance(Map<String, Object> bundleCompliance) { this.bundleCompliance = bundleCompliance; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class EvidenceBasedAlert implements Serializable {
        private static final long serialVersionUID = 1L;

        private String alertId;
        private String alertType;
        private String severity;
        private String message;
        private Map<String, String> evidence;
        private String actionRequired;
        private String timeframe;

        public EvidenceBasedAlert() {
            this.evidence = new HashMap<>();
        }

        // Getters and setters
        public String getAlertId() { return alertId; }
        public void setAlertId(String alertId) { this.alertId = alertId; }
        public String getAlertType() { return alertType; }
        public void setAlertType(String alertType) { this.alertType = alertType; }
        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }
        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }
        public Map<String, String> getEvidence() { return evidence; }
        public void setEvidence(Map<String, String> evidence) { this.evidence = evidence; }
        public String getActionRequired() { return actionRequired; }
        public void setActionRequired(String actionRequired) { this.actionRequired = actionRequired; }
        public String getTimeframe() { return timeframe; }
        public void setTimeframe(String timeframe) { this.timeframe = timeframe; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class CEPPatternFlag implements Serializable {
        private static final long serialVersionUID = 1L;

        private Boolean flag;
        private Double confidence;
        private List<String> triggerComponents;
        private Boolean readyForCEP;
        private String expectedPattern;
        private String reason;

        public CEPPatternFlag() {
            this.triggerComponents = new ArrayList<>();
        }

        // Getters and setters
        public Boolean getFlag() { return flag; }
        public void setFlag(Boolean flag) { this.flag = flag; }
        public Double getConfidence() { return confidence; }
        public void setConfidence(Double confidence) { this.confidence = confidence; }
        public List<String> getTriggerComponents() { return triggerComponents; }
        public void setTriggerComponents(List<String> triggerComponents) { this.triggerComponents = triggerComponents; }
        public Boolean getReadyForCEP() { return readyForCEP; }
        public void setReadyForCEP(Boolean readyForCEP) { this.readyForCEP = readyForCEP; }
        public String getExpectedPattern() { return expectedPattern; }
        public void setExpectedPattern(String expectedPattern) { this.expectedPattern = expectedPattern; }
        public String getReason() { return reason; }
        public void setReason(String reason) { this.reason = reason; }
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class KnowledgeBaseSource implements Serializable {
        private static final long serialVersionUID = 1L;

        private String source;
        private String version;
        private String lastUpdated;
        private String citation;

        // Getters and setters
        public String getSource() { return source; }
        public void setSource(String source) { this.source = source; }
        public String getVersion() { return version; }
        public void setVersion(String version) { this.version = version; }
        public String getLastUpdated() { return lastUpdated; }
        public void setLastUpdated(String lastUpdated) { this.lastUpdated = lastUpdated; }
        public String getCitation() { return citation; }
        public void setCitation(String citation) { this.citation = citation; }
    }

    // ========== MAIN CLASS GETTERS AND SETTERS ==========

    public Long getEnrichmentTimestamp() { return enrichmentTimestamp; }
    public void setEnrichmentTimestamp(Long enrichmentTimestamp) { this.enrichmentTimestamp = enrichmentTimestamp; }

    public String getEnrichmentVersion() { return enrichmentVersion; }
    public void setEnrichmentVersion(String enrichmentVersion) { this.enrichmentVersion = enrichmentVersion; }

    public String getKnowledgeBaseVersion() { return knowledgeBaseVersion; }
    public void setKnowledgeBaseVersion(String knowledgeBaseVersion) { this.knowledgeBaseVersion = knowledgeBaseVersion; }

    public List<MatchedProtocolDetail> getMatchedProtocols() { return matchedProtocols; }
    public void setMatchedProtocols(List<MatchedProtocolDetail> matchedProtocols) { this.matchedProtocols = matchedProtocols; }

    public DrugInteractionAnalysis getDrugInteractionAnalysis() { return drugInteractionAnalysis; }
    public void setDrugInteractionAnalysis(DrugInteractionAnalysis drugInteractionAnalysis) { this.drugInteractionAnalysis = drugInteractionAnalysis; }

    public Map<String, ClinicalThreshold> getClinicalThresholds() { return clinicalThresholds; }
    public void setClinicalThresholds(Map<String, ClinicalThreshold> clinicalThresholds) { this.clinicalThresholds = clinicalThresholds; }

    public CarePathwayRecommendations getCarePathwayRecommendations() { return carePathwayRecommendations; }
    public void setCarePathwayRecommendations(CarePathwayRecommendations carePathwayRecommendations) { this.carePathwayRecommendations = carePathwayRecommendations; }

    public List<EvidenceBasedAlert> getEvidenceBasedAlerts() { return evidenceBasedAlerts; }
    public void setEvidenceBasedAlerts(List<EvidenceBasedAlert> evidenceBasedAlerts) { this.evidenceBasedAlerts = evidenceBasedAlerts; }

    public Map<String, CEPPatternFlag> getCepPatternFlags() { return cepPatternFlags; }
    public void setCepPatternFlags(Map<String, CEPPatternFlag> cepPatternFlags) { this.cepPatternFlags = cepPatternFlags; }

    public List<String> getSemanticTags() { return semanticTags; }
    public void setSemanticTags(List<String> semanticTags) { this.semanticTags = semanticTags; }

    public List<KnowledgeBaseSource> getKnowledgeBaseSources() { return knowledgeBaseSources; }
    public void setKnowledgeBaseSources(List<KnowledgeBaseSource> knowledgeBaseSources) { this.knowledgeBaseSources = knowledgeBaseSources; }
}
