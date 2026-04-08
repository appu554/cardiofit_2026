package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Unified wrapper for all Module 6 input event types.
 * Exactly one of cdsEvent/patternEvent/prediction is non-null.
 */
public class ClinicalEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum Source { CDS, PATTERN, ML_PREDICTION }

    private String patientId;
    private long eventTime;
    private Source source;
    private CDSEvent cdsEvent;
    private PatternEvent patternEvent;
    private MLPrediction prediction;

    private ClinicalEvent() {}

    public static ClinicalEvent fromCDS(CDSEvent cds) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = cds.getPatientId();
        e.eventTime = cds.getEventTime();
        e.source = Source.CDS;
        e.cdsEvent = cds;
        return e;
    }

    public static ClinicalEvent fromPattern(PatternEvent pattern) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = pattern.getPatientId();
        e.eventTime = pattern.getDetectionTime();
        e.source = Source.PATTERN;
        e.patternEvent = pattern;
        return e;
    }

    public static ClinicalEvent fromPrediction(MLPrediction pred) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = pred.getPatientId();
        e.eventTime = pred.getPredictionTime();
        e.source = Source.ML_PREDICTION;
        e.prediction = pred;
        return e;
    }

    // ── Convenience accessors across all source types ──

    /** Sentinel value: NEWS2 score was not computed / not available. */
    public static final int NEWS2_ABSENT = -1;

    /**
     * Returns NEWS2 score, or {@link #NEWS2_ABSENT} if not available.
     * Callers must distinguish "score is 0" (patient stable) from "score absent"
     * (no vitals data to compute a score).
     */
    public int getNews2Score() {
        if (cdsEvent != null && cdsEvent.getPatientState() != null
                && cdsEvent.getPatientState().getNews2Score() != null) {
            return cdsEvent.getPatientState().getNews2Score();
        }
        return NEWS2_ABSENT;
    }

    public int getQsofaScore() {
        if (cdsEvent != null && cdsEvent.getPatientState() != null
                && cdsEvent.getPatientState().getQsofaScore() != null) {
            return cdsEvent.getPatientState().getQsofaScore();
        }
        return 0;
    }

    public boolean hasSepsisIndicators() {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return false;
        RiskIndicators ri = cdsEvent.getPatientState().getRiskIndicators();
        if (ri == null) return false;
        return ri.isFever() || ri.isElevatedLactate() || ri.isTachycardia();
    }

    /**
     * Check for an active alert by type and optional severity.
     * Searches PatientContextState.activeAlerts (Set of SimpleAlert).
     */
    public boolean hasActiveAlert(String alertType, String severity) {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return false;
        Set<SimpleAlert> alerts = cdsEvent.getPatientState().getActiveAlerts();
        if (alerts == null) return false;
        for (SimpleAlert alert : alerts) {
            String type = alert.getAlertType() != null ? alert.getAlertType().toString() : "";
            String sev = alert.getSeverity() != null ? alert.getSeverity().toString() : "";
            if (type.contains(alertType)) {
                if (severity == null || sev.equalsIgnoreCase(severity)) return true;
            }
            // Also check message field for alert type strings like "HYPERKALEMIA_ALERT"
            String msg = alert.getMessage() != null ? alert.getMessage() : "";
            if (msg.contains(alertType)) {
                if (severity == null || sev.equalsIgnoreCase(severity) || msg.contains(severity)) return true;
            }
        }
        return false;
    }

    public boolean hasActiveAlert(String alertType) {
        return hasActiveAlert(alertType, null);
    }

    public String getAlertDetail(String alertType, String detailKey) {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return null;
        Set<SimpleAlert> alerts = cdsEvent.getPatientState().getActiveAlerts();
        if (alerts == null) return null;
        for (SimpleAlert alert : alerts) {
            String msg = alert.getMessage() != null ? alert.getMessage() : "";
            String type = alert.getAlertType() != null ? alert.getAlertType().toString() : "";
            if (type.contains(alertType) || msg.contains(alertType)) {
                Map<String, Object> ctx = alert.getContext();
                if (ctx != null && ctx.containsKey(detailKey)) {
                    return String.valueOf(ctx.get(detailKey));
                }
            }
        }
        return null;
    }

    public boolean hasPattern(String patternType) {
        return patternEvent != null && patternType.equals(patternEvent.getPatternType());
    }

    public PatternEvent getPattern(String patternType) {
        if (hasPattern(patternType)) return patternEvent;
        return null;
    }

    public boolean hasPatternWithSeverity(String severity) {
        return patternEvent != null && severity.equalsIgnoreCase(patternEvent.getSeverity());
    }

    public boolean hasPrediction(String category) {
        return prediction != null && category.equalsIgnoreCase(prediction.getPredictionCategory());
    }

    public MLPrediction getPrediction(String category) {
        if (hasPrediction(category)) return prediction;
        return null;
    }

    public boolean hasAnyPredictionAbove(double threshold) {
        if (prediction == null || prediction.getCalibratedScore() == null) return false;
        return prediction.getCalibratedScore() >= threshold - 1e-9;
    }

    /**
     * Derive the clinical category for dedup keying.
     * CDS events → derived from active alerts or scores.
     * Pattern events → patternType.
     * ML predictions → predictionCategory.
     */
    public String getClinicalCategory() {
        if (patternEvent != null) return patternEvent.getPatternType();
        if (prediction != null) return prediction.getPredictionCategory() != null
                ? prediction.getPredictionCategory().toUpperCase() : "ML_PREDICTION";
        // CDS: derive from dominant risk
        if (cdsEvent != null) {
            if (hasActiveAlert("HYPERKALEMIA")) return "HYPERKALEMIA";
            if (hasActiveAlert("ANTICOAGULATION")) return "ANTICOAGULATION";
            if (hasActiveAlert("AKI")) return "AKI";
            if (getQsofaScore() >= 2 || hasActiveAlert("SEPSIS")) return "SEPSIS";
            int n2 = getNews2Score();
            if (n2 != NEWS2_ABSENT && n2 >= 7) return "DETERIORATION";
            return "CDS_GENERAL";
        }
        return "UNKNOWN";
    }

    // ── Standard getters ──
    public String getPatientId() { return patientId; }
    public long getEventTime() { return eventTime; }
    public Source getSource() { return source; }
    public CDSEvent getCdsEvent() { return cdsEvent; }
    public PatternEvent getPatternEvent() { return patternEvent; }
    public MLPrediction getPrediction() { return prediction; }
}
