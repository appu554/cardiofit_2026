package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Per-patient ML state maintained across events in Module 5.
 * Stored in Flink ValueState with 7-day TTL.
 */
public class PatientMLState implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final int HISTORY_SIZE = 10;
    public static final int MAX_PATTERN_BUFFER_SIZE = 20;

    private String patientId;

    // ── Latest clinical snapshot (from CDS events) ──
    private Map<String, Double> latestVitals;
    private Map<String, Double> latestLabs;
    private int news2Score;
    private int qsofaScore;
    private double acuityScore;
    private List<String> semanticTags;
    private Map<String, Object> riskIndicators;
    private Map<String, Object> activeAlerts;

    // ── Temporal features (ring buffers) ──
    private double[] news2History;
    private int news2HistoryIndex;
    private double[] acuityHistory;
    private int acuityHistoryIndex;
    private long firstEventTime;
    private int totalEventCount;

    // ── Pattern features (from Module 4) ──
    private List<PatternSummary> recentPatterns;
    private int deteriorationPatternCount;
    private int sepsisPatternCount;
    private String maxSeveritySeen;
    private boolean severityEscalationDetected;
    private long lastPatternTime;

    // ── Prediction tracking ──
    private Map<String, Double> lastPredictions;
    private long lastInferenceTime;

    public PatientMLState() {
        this.latestVitals = new HashMap<>();
        this.latestLabs = new HashMap<>();
        this.riskIndicators = new HashMap<>();
        this.activeAlerts = new HashMap<>();
        this.semanticTags = new ArrayList<>();
        this.news2History = new double[HISTORY_SIZE];
        this.acuityHistory = new double[HISTORY_SIZE];
        this.recentPatterns = new ArrayList<>();
        this.lastPredictions = new HashMap<>();
        this.maxSeveritySeen = "NONE";
    }

    // ── Ring buffer helpers ──

    public void pushNews2(int score) {
        news2History[news2HistoryIndex % HISTORY_SIZE] = score;
        news2HistoryIndex++;
    }

    public void pushAcuity(double score) {
        acuityHistory[acuityHistoryIndex % HISTORY_SIZE] = score;
        acuityHistoryIndex++;
    }

    public double news2Slope() {
        return calculateSlope(news2History, Math.min(news2HistoryIndex, HISTORY_SIZE));
    }

    public double acuitySlope() {
        return calculateSlope(acuityHistory, Math.min(acuityHistoryIndex, HISTORY_SIZE));
    }

    private static double calculateSlope(double[] buffer, int count) {
        if (count < 2) return 0.0;
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (int i = 0; i < count; i++) {
            sumX += i;
            sumY += buffer[i];
            sumXY += i * buffer[i];
            sumX2 += i * i;
        }
        double denom = count * sumX2 - sumX * sumX;
        if (Math.abs(denom) < 1e-9) return 0.0;
        return (count * sumXY - sumX * sumY) / denom;
    }

    // ── Pattern buffer ──

    public void addPattern(PatternSummary pattern) {
        if (recentPatterns.size() >= MAX_PATTERN_BUFFER_SIZE) {
            recentPatterns.remove(0);
        }
        recentPatterns.add(pattern);
        lastPatternTime = pattern.detectionTime();

        if ("CLINICAL_DETERIORATION".equals(pattern.patternType())) {
            deteriorationPatternCount++;
        }
        if ("SEPSIS_RISK".equals(pattern.patternType())
                || "SEPSIS".equalsIgnoreCase(pattern.patternType())) {
            sepsisPatternCount++;
        }

        int sevIdx = severityIndex(pattern.severity());
        if (sevIdx > severityIndex(maxSeveritySeen)) {
            maxSeveritySeen = pattern.severity();
        }
        if (pattern.tags() != null
                && pattern.tags().contains("SEVERITY_ESCALATION")) {
            severityEscalationDetected = true;
        }
    }

    public void clearPatternBuffer() {
        recentPatterns.clear();
        deteriorationPatternCount = 0;
        sepsisPatternCount = 0;
        maxSeveritySeen = "NONE";
        severityEscalationDetected = false;
    }

    public static int severityIndex(String severity) {
        if (severity == null) return 0;
        return switch (severity.toUpperCase()) {
            case "LOW" -> 1;
            case "MODERATE" -> 2;
            case "HIGH" -> 3;
            case "CRITICAL" -> 4;
            default -> 0;
        };
    }

    // ── Pattern summary record ──

    public record PatternSummary(
        String patternType,
        String severity,
        double confidence,
        long detectionTime,
        Set<String> tags
    ) implements Serializable {}

    // ── Standard getters/setters ──

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public Map<String, Double> getLatestVitals() { return latestVitals; }
    public void setLatestVitals(Map<String, Double> latestVitals) { this.latestVitals = latestVitals; }

    public Map<String, Double> getLatestLabs() { return latestLabs; }
    public void setLatestLabs(Map<String, Double> latestLabs) { this.latestLabs = latestLabs; }

    public int getNews2Score() { return news2Score; }
    public void setNews2Score(int news2Score) { this.news2Score = news2Score; }

    public int getQsofaScore() { return qsofaScore; }
    public void setQsofaScore(int qsofaScore) { this.qsofaScore = qsofaScore; }

    public double getAcuityScore() { return acuityScore; }
    public void setAcuityScore(double acuityScore) { this.acuityScore = acuityScore; }

    public List<String> getSemanticTags() { return semanticTags; }
    public void setSemanticTags(List<String> semanticTags) { this.semanticTags = semanticTags; }

    public Map<String, Object> getRiskIndicators() { return riskIndicators; }
    public void setRiskIndicators(Map<String, Object> riskIndicators) { this.riskIndicators = riskIndicators; }

    public Map<String, Object> getActiveAlerts() { return activeAlerts; }
    public void setActiveAlerts(Map<String, Object> activeAlerts) { this.activeAlerts = activeAlerts; }

    public double[] getNews2History() { return news2History; }
    public double[] getAcuityHistory() { return acuityHistory; }
    public int getNews2HistoryIndex() { return news2HistoryIndex; }
    public int getAcuityHistoryIndex() { return acuityHistoryIndex; }

    public long getFirstEventTime() { return firstEventTime; }
    public void setFirstEventTime(long firstEventTime) { this.firstEventTime = firstEventTime; }

    public int getTotalEventCount() { return totalEventCount; }
    public void setTotalEventCount(int totalEventCount) { this.totalEventCount = totalEventCount; }

    public List<PatternSummary> getRecentPatterns() { return recentPatterns; }
    public int getDeteriorationPatternCount() { return deteriorationPatternCount; }
    public int getSepsisPatternCount() { return sepsisPatternCount; }
    public String getMaxSeveritySeen() { return maxSeveritySeen; }
    public boolean isSeverityEscalationDetected() { return severityEscalationDetected; }
    public long getLastPatternTime() { return lastPatternTime; }

    public void setRecentPatterns(List<PatternSummary> recentPatterns) { this.recentPatterns = recentPatterns; }
    public void setDeteriorationPatternCount(int deteriorationPatternCount) { this.deteriorationPatternCount = deteriorationPatternCount; }
    public void setSepsisPatternCount(int sepsisPatternCount) { this.sepsisPatternCount = sepsisPatternCount; }
    public void setMaxSeveritySeen(String maxSeveritySeen) { this.maxSeveritySeen = maxSeveritySeen; }
    public void setSeverityEscalationDetected(boolean severityEscalationDetected) { this.severityEscalationDetected = severityEscalationDetected; }
    public void setLastPatternTime(long lastPatternTime) { this.lastPatternTime = lastPatternTime; }

    public Map<String, Double> getLastPredictions() { return lastPredictions; }
    public void setLastPredictions(Map<String, Double> lastPredictions) { this.lastPredictions = lastPredictions; }

    public long getLastInferenceTime() { return lastInferenceTime; }
    public void setLastInferenceTime(long lastInferenceTime) { this.lastInferenceTime = lastInferenceTime; }
}
