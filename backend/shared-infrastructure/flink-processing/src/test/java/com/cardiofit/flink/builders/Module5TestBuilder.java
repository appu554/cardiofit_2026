package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.PatientMLState;
import com.cardiofit.flink.models.PatternEvent;

import java.util.*;

/**
 * Test data factory for Module 5 tests.
 * All data uses PRODUCTION key formats (lowercase-no-separator vitals, LOINC labs).
 */
public class Module5TestBuilder {

    // ── Patient ML State factories ──

    /** Stable patient: NEWS2=1, qSOFA=0, normal vitals and labs */
    public static PatientMLState stablePatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 78.0,
            "systolicbloodpressure", 128.0,
            "diastolicbloodpressure", 82.0,
            "respiratoryrate", 16.0,
            "oxygensaturation", 97.0,
            "temperature", 36.8
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.2,
            "creatinine", 0.9,
            "potassium", 4.1,
            "wbc", 7.5,
            "platelets", 220.0,
            "inr", 1.0
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(1.5);
        state.setTotalEventCount(5);
        state.setFirstEventTime(System.currentTimeMillis() - 86400000L);
        state.setRiskIndicators(stableRiskIndicators());
        state.setActiveAlerts(Collections.emptyMap());
        return state;
    }

    /** Sepsis patient: NEWS2=9, qSOFA=2, elevated lactate/WBC/temp, falling BP */
    public static PatientMLState sepsisPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 118.0,
            "systolicbloodpressure", 85.0,
            "diastolicbloodpressure", 52.0,
            "respiratoryrate", 26.0,
            "oxygensaturation", 91.0,
            "temperature", 39.2
        ));
        state.setLatestLabs(Map.of(
            "lactate", 4.8,
            "creatinine", 1.8,
            "potassium", 4.9,
            "wbc", 18.5,
            "platelets", 130.0,
            "inr", 1.3
        ));
        state.setNews2Score(9);
        state.setQsofaScore(2);
        state.setAcuityScore(7.5);
        state.setTotalEventCount(12);
        state.setFirstEventTime(System.currentTimeMillis() - 172800000L);
        state.setRiskIndicators(sepsisRiskIndicators());
        state.setActiveAlerts(sepsisAlerts());
        return state;
    }

    /**
     * AKI patient: NEWS2=1, qSOFA=0 (vitals normal!), but creatinine=3.2, K+=6.1.
     * This is the Gap 3 patient — lab-only emergency invisible to vitals-based scoring.
     */
    public static PatientMLState akiPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 82.0,
            "systolicbloodpressure", 135.0,
            "diastolicbloodpressure", 85.0,
            "respiratoryrate", 17.0,
            "oxygensaturation", 96.0,
            "temperature", 37.0
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.5,
            "creatinine", 3.2,
            "potassium", 6.1,
            "wbc", 9.0,
            "platelets", 180.0,
            "inr", 1.1
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(2.0);
        state.setTotalEventCount(8);
        state.setRiskIndicators(akiRiskIndicators());
        state.setActiveAlerts(akiAlerts());
        return state;
    }

    /** Drug-lab patient: Warfarin + INR 6.0 — NEWS2=1, qSOFA=0 */
    public static PatientMLState drugLabPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 76.0,
            "systolicbloodpressure", 130.0,
            "diastolicbloodpressure", 80.0,
            "respiratoryrate", 15.0,
            "oxygensaturation", 98.0,
            "temperature", 36.7
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.0,
            "creatinine", 1.0,
            "potassium", 4.2,
            "wbc", 7.0,
            "platelets", 55.0,
            "inr", 6.0
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(1.8);
        state.setRiskIndicators(drugLabRiskIndicators());
        state.setActiveAlerts(drugLabAlerts());
        return state;
    }

    /** State with null/missing labs — tests null-safety path */
    public static PatientMLState sparsePatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 80.0,
            "systolicbloodpressure", 120.0
        ));
        state.setLatestLabs(Collections.emptyMap());
        state.setNews2Score(0);
        state.setQsofaScore(0);
        state.setRiskIndicators(Collections.emptyMap());
        state.setActiveAlerts(Collections.emptyMap());
        state.setTotalEventCount(1);
        return state;
    }

    // ── Risk indicator factories ──

    private static Map<String, Object> stableRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("fever", false);
        risk.put("hypoxia", false);
        risk.put("elevatedLactate", false);
        risk.put("elevatedCreatinine", false);
        risk.put("hyperkalemia", false);
        risk.put("thrombocytopenia", false);
        risk.put("onAnticoagulation", false);
        risk.put("onVasopressors", false);
        risk.put("confidenceScore", 0.85);
        return risk;
    }

    private static Map<String, Object> sepsisRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("tachycardia", true);
        risk.put("hypotension", true);
        risk.put("fever", true);
        risk.put("elevatedLactate", true);
        risk.put("severelyElevatedLactate", true);
        risk.put("leukocytosis", true);
        risk.put("sepsisRisk", true);
        risk.put("confidenceScore", 0.92);
        return risk;
    }

    private static Map<String, Object> akiRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("elevatedCreatinine", true);
        risk.put("hyperkalemia", true);
        // Vitals-based flags are all false (normal vitals)
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("fever", false);
        risk.put("confidenceScore", 0.78);
        return risk;
    }

    private static Map<String, Object> drugLabRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("onAnticoagulation", true);
        risk.put("thrombocytopenia", true);
        // Flag may be overwritten by Module 3 (Gap 2)
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("confidenceScore", 0.70);
        return risk;
    }

    // ── Active alert factories (Gap 4) ──

    private static Map<String, Object> sepsisAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> sepsisAlert = new HashMap<>();
        sepsisAlert.put("severity", "CRITICAL");
        sepsisAlert.put("message", "Sepsis pattern detected: lactate 4.8, WBC 18.5, temp 39.2");
        alerts.put("SEPSIS_PATTERN", sepsisAlert);

        Map<String, Object> detAlert = new HashMap<>();
        detAlert.put("severity", "HIGH");
        detAlert.put("message", "Clinical deterioration: multi-system decline");
        alerts.put("DETERIORATION_PATTERN", detAlert);
        return alerts;
    }

    private static Map<String, Object> akiAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> akiAlert = new HashMap<>();
        akiAlert.put("severity", "HIGH");
        akiAlert.put("stage", 2);
        akiAlert.put("message", "AKI Risk: creatinine 3.2, potassium 6.1");
        alerts.put("AKI_RISK", akiAlert);

        Map<String, Object> hyperK = new HashMap<>();
        hyperK.put("severity", "CRITICAL");
        hyperK.put("message", "Hyperkalemia: K+ 6.1 mEq/L");
        alerts.put("HYPERKALEMIA_ALERT", hyperK);
        return alerts;
    }

    private static Map<String, Object> drugLabAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> anticoagAlert = new HashMap<>();
        anticoagAlert.put("severity", "CRITICAL");
        anticoagAlert.put("message", "Anticoagulation risk: INR 6.0");
        alerts.put("ANTICOAGULATION_RISK", anticoagAlert);

        Map<String, Object> bleedAlert = new HashMap<>();
        bleedAlert.put("severity", "HIGH");
        bleedAlert.put("message", "Bleeding risk: platelets 55k, INR 6.0");
        alerts.put("BLEEDING_RISK", bleedAlert);
        return alerts;
    }

    // ── Pattern event factories ──

    /** CRITICAL deterioration pattern with SEVERITY_ESCALATION tag */
    public static PatternEvent criticalEscalationPattern(String patientId) {
        PatternEvent event = new PatternEvent();
        event.setId("pattern-crit-" + UUID.randomUUID().toString().substring(0, 8));
        event.setPatientId(patientId);
        event.setPatternType("CLINICAL_DETERIORATION");
        event.setSeverity("CRITICAL");
        event.setConfidence(0.92);
        event.setDetectionTime(System.currentTimeMillis());
        event.setTags(Set.of("SEVERITY_ESCALATION", "MULTI_SOURCE_CONFIRMED"));
        event.setPriority(1);
        return event;
    }

    /** Moderate trend analysis pattern (should NOT trigger immediate inference) */
    public static PatternEvent moderateTrendPattern(String patientId) {
        PatternEvent event = new PatternEvent();
        event.setId("pattern-trend-" + UUID.randomUUID().toString().substring(0, 8));
        event.setPatientId(patientId);
        event.setPatternType("TREND_ANALYSIS");
        event.setSeverity("MODERATE");
        event.setConfidence(0.65);
        event.setDetectionTime(System.currentTimeMillis());
        event.setTags(Set.of("VITAL_SIGNS_TREND"));
        event.setPriority(3);
        return event;
    }
}
