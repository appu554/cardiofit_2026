package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;
import java.util.List;

public class EvidenceEnvelope implements Serializable {
    private static final long serialVersionUID = 1L;

    private String envelopeId;
    private String patientId;
    private String evidenceType;
    private String source;
    private LocalDateTime timestamp;
    private Map<String, Object> evidenceData;
    private List<String> references;
    private double confidenceScore;
    private String qualityLevel;

    public EvidenceEnvelope() {}

    public EvidenceEnvelope(String envelopeId, String patientId, String evidenceType) {
        this.envelopeId = envelopeId;
        this.patientId = patientId;
        this.evidenceType = evidenceType;
        this.timestamp = LocalDateTime.now();
    }

    public String getEnvelopeId() { return envelopeId; }
    public void setEnvelopeId(String envelopeId) { this.envelopeId = envelopeId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEvidenceType() { return evidenceType; }
    public void setEvidenceType(String evidenceType) { this.evidenceType = evidenceType; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public LocalDateTime getTimestamp() { return timestamp; }
    public void setTimestamp(LocalDateTime timestamp) { this.timestamp = timestamp; }

    public Map<String, Object> getEvidenceData() { return evidenceData; }
    public void setEvidenceData(Map<String, Object> evidenceData) { this.evidenceData = evidenceData; }

    public List<String> getReferences() { return references; }
    public void setReferences(List<String> references) { this.references = references; }

    public double getConfidenceScore() { return confidenceScore; }
    public void setConfidenceScore(double confidenceScore) { this.confidenceScore = confidenceScore; }

    public String getQualityLevel() { return qualityLevel; }
    public void setQualityLevel(String qualityLevel) { this.qualityLevel = qualityLevel; }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public void setEventId(String eventId) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("event_id", eventId);
    }

    public void setProtocolId(String protocolId) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("protocol_id", protocolId);
    }

    public void setMeshVersion(String meshVersion) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("mesh_version", meshVersion);
    }

    public void setAnalysisType(String analysisType) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("analysis_type", analysisType);
    }

    public void setDeviation(boolean deviation) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("has_deviation", deviation);
    }

    public void setSeverity(com.cardiofit.stream.state.PathwayAnalysis.Severity severity) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("severity", severity != null ? severity.name() : null);
    }

    public void setRecommendations(java.util.List<String> recommendations) {
        this.references = recommendations; // Store recommendations as references
    }

    public void setConfidence(double confidence) {
        this.confidenceScore = confidence;
    }

    public void setCreatedAt(java.time.LocalDateTime createdAt) {
        this.timestamp = createdAt;
    }

    public void setInferenceChain(java.util.List<String> inferenceChain) {
        if (evidenceData == null) {
            evidenceData = new java.util.HashMap<>();
        }
        evidenceData.put("inference_chain", inferenceChain);
    }
}