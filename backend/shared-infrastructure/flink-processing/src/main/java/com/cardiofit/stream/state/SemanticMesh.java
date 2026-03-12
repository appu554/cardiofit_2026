package com.cardiofit.stream.state;

import java.io.Serializable;
import java.util.Map;
import java.util.List;

public class SemanticMesh implements Serializable {
    private static final long serialVersionUID = 1L;

    private String meshId;
    private String domain;
    private Map<String, List<String>> relationships;
    private Map<String, Double> weights;
    private Map<String, Object> metadata;
    private String ontologyVersion;

    public SemanticMesh() {}

    public SemanticMesh(String meshId, String domain) {
        this.meshId = meshId;
        this.domain = domain;
    }

    public String getMeshId() { return meshId; }
    public void setMeshId(String meshId) { this.meshId = meshId; }

    public String getDomain() { return domain; }
    public void setDomain(String domain) { this.domain = domain; }

    public Map<String, List<String>> getRelationships() { return relationships; }
    public void setRelationships(Map<String, List<String>> relationships) { this.relationships = relationships; }

    public Map<String, Double> getWeights() { return weights; }
    public void setWeights(Map<String, Double> weights) { this.weights = weights; }

    public Map<String, Object> getMetadata() { return metadata; }
    public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

    public String getOntologyVersion() { return ontologyVersion; }
    public void setOntologyVersion(String ontologyVersion) { this.ontologyVersion = ontologyVersion; }

    // Methods required by ClinicalPathwayAdherenceFunction
    public String getVersion() {
        return this.ontologyVersion != null ? this.ontologyVersion : "1.0.0";
    }

    public List<Protocol> getApplicableProtocols(PatientContext patientContext) {
        // Placeholder implementation - in production this would use sophisticated
        // protocol matching based on patient conditions and context
        List<Protocol> protocols = new java.util.ArrayList<>();

        // Create default protocols based on patient context
        if (patientContext != null) {
            // Example: cardiology protocol for cardiac patients
            Protocol defaultProtocol = new Protocol("DEFAULT_CARDIO", "Standard Cardiology Care", "1.0");
            defaultProtocol.setPhases(java.util.Arrays.asList("assessment", "treatment", "monitoring"));
            protocols.add(defaultProtocol);
        }

        return protocols;
    }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public java.util.Set<String> getContraindications(String medication,
                                                      java.util.List<String> activeMedications,
                                                      java.util.List<String> conditions) {
        java.util.Set<String> contraindications = new java.util.HashSet<>();

        // Check drug-drug interactions
        if (activeMedications != null && medication != null) {
            for (String activeMed : activeMedications) {
                if (hasDrugInteraction(medication, activeMed)) {
                    contraindications.add("Drug interaction: " + medication + " with " + activeMed);
                }
            }
        }

        // Check drug-condition contraindications
        if (conditions != null && medication != null) {
            for (String condition : conditions) {
                if (hasConditionContraindication(medication, condition)) {
                    contraindications.add("Contraindicated for: " + condition);
                }
            }
        }

        return contraindications;
    }

    public java.util.List<String> getSaferAlternatives(String medication, PatientContext patientContext) {
        java.util.List<String> alternatives = new java.util.ArrayList<>();

        // Simple medication alternatives mapping
        if (medication != null) {
            switch (medication.toLowerCase()) {
                case "warfarin":
                    alternatives.add("apixaban");
                    alternatives.add("rivaroxaban");
                    break;
                case "aspirin":
                    alternatives.add("clopidogrel");
                    break;
                default:
                    alternatives.add("consult_pharmacist");
            }
        }

        return alternatives;
    }

    public boolean isLatestVersion() {
        // Check if this mesh is the latest version
        return ontologyVersion != null && !ontologyVersion.equals("deprecated");
    }

    private boolean hasDrugInteraction(String drug1, String drug2) {
        // Simplified drug interaction checking
        if (drug1 == null || drug2 == null) return false;

        // Known dangerous combinations
        return (drug1.toLowerCase().contains("warfarin") && drug2.toLowerCase().contains("aspirin")) ||
               (drug1.toLowerCase().contains("metformin") && drug2.toLowerCase().contains("contrast")) ||
               drug1.equals(drug2); // Same drug
    }

    private boolean hasConditionContraindication(String medication, String condition) {
        // Simplified condition contraindication checking
        if (medication == null || condition == null) return false;

        return (medication.toLowerCase().contains("beta-blocker") && condition.toLowerCase().contains("asthma")) ||
               (medication.toLowerCase().contains("nsaid") && condition.toLowerCase().contains("kidney"));
    }
}