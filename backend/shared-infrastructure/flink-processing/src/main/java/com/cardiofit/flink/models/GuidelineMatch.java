package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class GuidelineMatch implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("guidelineId")
    private String guidelineId;

    @JsonProperty("guidelineName")
    private String guidelineName;

    @JsonProperty("concordance")
    private String concordance;

    @JsonProperty("confidence")
    private double confidence;

    @JsonProperty("recommendation")
    private String recommendation;

    @JsonProperty("evidenceLevel")
    private String evidenceLevel;

    public GuidelineMatch() {}

    public GuidelineMatch(String id, String name, String concordance, double confidence) {
        this.guidelineId = id;
        this.guidelineName = name;
        this.concordance = concordance;
        this.confidence = confidence;
    }

    // Getters and setters
    public String getGuidelineId() { return guidelineId; }
    public void setGuidelineId(String v) { this.guidelineId = v; }
    public String getGuidelineName() { return guidelineName; }
    public void setGuidelineName(String v) { this.guidelineName = v; }
    public String getConcordance() { return concordance; }
    public void setConcordance(String v) { this.concordance = v; }
    public double getConfidence() { return confidence; }
    public void setConfidence(double v) { this.confidence = v; }
    public String getRecommendation() { return recommendation; }
    public void setRecommendation(String v) { this.recommendation = v; }
    public String getEvidenceLevel() { return evidenceLevel; }
    public void setEvidenceLevel(String v) { this.evidenceLevel = v; }
}
