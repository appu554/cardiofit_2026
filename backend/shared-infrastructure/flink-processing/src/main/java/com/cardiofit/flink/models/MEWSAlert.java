package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Modified Early Warning Score (MEWS) alert for patient deterioration monitoring.
 *
 * MEWS is a track-and-trigger system that identifies patients at risk of clinical deterioration.
 * Scores ≥3 require increased monitoring, scores ≥5 require urgent medical review.
 *
 * Scoring system:
 * - Respiratory Rate: <9 (2), 9-14 (0), 15-20 (1), 21-29 (2), ≥30 (3)
 * - Heart Rate: <40 (2), 40-50 (1), 51-100 (0), 101-110 (1), 111-129 (2), ≥130 (3)
 * - Systolic BP: <70 (3), 70-80 (2), 81-100 (1), 101-199 (0), ≥200 (2)
 * - Temperature: <35 (2), 35-38.4 (0), ≥38.5 (2)
 * - AVPU Score: Alert (0), Voice (1), Pain (2), Unresponsive (3)
 */
public class MEWSAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private Integer mewsScore;
    private Map<String, Integer> scoreBreakdown;  // e.g., {"Respiratory_Rate": 2, "Heart_Rate": 1}
    private List<String> concerningVitals;
    private String urgency;  // e.g., "🔴 CRITICAL: Urgent medical review required"
    private String recommendations;
    private Long timestamp;
    private Long windowStart;
    private Long windowEnd;

    public MEWSAlert() {}

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public Integer getMewsScore() {
        return mewsScore;
    }

    public void setMewsScore(Integer mewsScore) {
        this.mewsScore = mewsScore;
    }

    public Map<String, Integer> getScoreBreakdown() {
        return scoreBreakdown;
    }

    public void setScoreBreakdown(Map<String, Integer> scoreBreakdown) {
        this.scoreBreakdown = scoreBreakdown;
    }

    public List<String> getConcerningVitals() {
        return concerningVitals;
    }

    public void setConcerningVitals(List<String> concerningVitals) {
        this.concerningVitals = concerningVitals;
    }

    public String getUrgency() {
        return urgency;
    }

    public void setUrgency(String urgency) {
        this.urgency = urgency;
    }

    public String getRecommendations() {
        return recommendations;
    }

    public void setRecommendations(String recommendations) {
        this.recommendations = recommendations;
    }

    public Long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Long timestamp) {
        this.timestamp = timestamp;
    }

    public Long getWindowStart() {
        return windowStart;
    }

    public void setWindowStart(Long windowStart) {
        this.windowStart = windowStart;
    }

    public Long getWindowEnd() {
        return windowEnd;
    }

    public void setWindowEnd(Long windowEnd) {
        this.windowEnd = windowEnd;
    }

    @Override
    public String toString() {
        return "MEWSAlert{" +
                "patientId='" + patientId + '\'' +
                ", mewsScore=" + mewsScore +
                ", scoreBreakdown=" + scoreBreakdown +
                ", concerningVitals=" + concerningVitals +
                ", urgency='" + urgency + '\'' +
                ", recommendations='" + recommendations + '\'' +
                ", timestamp=" + timestamp +
                ", windowStart=" + windowStart +
                ", windowEnd=" + windowEnd +
                '}';
    }
}
