package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;

import java.util.*;

/**
 * Test data factory for Module 6 Clinical Action Engine tests.
 * Provides patient scenarios validated against passing E2E pipeline data.
 */
public class Module6TestBuilder {

    // ── ClinicalEvent builders from CDSEvent ──

    /**
     * Sepsis patient: NEWS2=13, qSOFA=2, elevated lactate/WBC.
     * Expected classification: HALT
     */
    public static ClinicalEvent sepsisHaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(13);
        state.setQsofaScore(2);
        state.getLatestVitals().put("heartrate", 130);
        state.getLatestVitals().put("systolicbloodpressure", 78);
        state.getLatestVitals().put("respiratoryrate", 30);
        state.getLatestVitals().put("temperature", 39.8);
        state.getLatestVitals().put("oxygensaturation", 87);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(4.8);
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        RiskIndicators ri = new RiskIndicators();
        ri.setTachycardia(true);
        ri.setHypotension(true);
        ri.setFever(true);
        ri.setElevatedLactate(true);
        state.setRiskIndicators(ri);

        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Hyperkalemia patient: NEWS2=2, qSOFA=0, K+=6.5.
     * Lab emergency invisible to vitals scoring.
     * Expected classification: HALT (lab-derived)
     */
    public static ClinicalEvent hyperkalemiaHaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(2);
        state.setQsofaScore(0);
        state.getLatestVitals().put("heartrate", 82);
        state.getLatestVitals().put("systolicbloodpressure", 128);

        LabResult potassium = new LabResult();
        potassium.setLabCode("2823-3");
        potassium.setValue(6.5);
        potassium.setUnit("mmol/L");
        state.getRecentLabs().put("2823-3", potassium);

        // Simulate activeAlert for HYPERKALEMIA
        SimpleAlert kAlert = new SimpleAlert();
        kAlert.setAlertType(AlertType.LAB_CRITICAL_VALUE);
        kAlert.setSeverity(AlertSeverity.CRITICAL);
        kAlert.setMessage("HYPERKALEMIA_ALERT CRITICAL K+ 6.5");
        kAlert.setPatientId(patientId);
        kAlert.setTimestamp(System.currentTimeMillis());
        state.addAlert(kAlert);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * AKI Stage 3 patient: NEWS2=3, qSOFA=0, Cr=4.2.
     * Expected classification: HALT
     */
    public static ClinicalEvent akiStage3HaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(3);
        state.setQsofaScore(0);

        LabResult creatinine = new LabResult();
        creatinine.setLabCode("2160-0");
        creatinine.setValue(4.2);
        creatinine.setUnit("mg/dL");
        state.getRecentLabs().put("2160-0", creatinine);

        SimpleAlert akiAlert = new SimpleAlert();
        akiAlert.setAlertType(AlertType.LAB_CRITICAL_VALUE);
        akiAlert.setSeverity(AlertSeverity.CRITICAL);
        akiAlert.setMessage("AKI_RISK STAGE_3");
        akiAlert.setPatientId(patientId);
        akiAlert.setTimestamp(System.currentTimeMillis());
        Map<String, Object> ctx = new HashMap<>();
        ctx.put("stage", "STAGE_3");
        akiAlert.setContext(ctx);
        state.addAlert(akiAlert);

        RiskIndicators ri = new RiskIndicators();
        ri.setElevatedCreatinine(true);
        state.setRiskIndicators(ri);

        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Moderate deterioration: NEWS2=7, qSOFA=1.
     * Expected classification: PAUSE
     */
    public static ClinicalEvent moderateDeteriorationPauseEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(7);
        state.setQsofaScore(1);
        state.getLatestVitals().put("heartrate", 105);
        state.getLatestVitals().put("systolicbloodpressure", 95);
        state.getLatestVitals().put("respiratoryrate", 22);
        state.getLatestVitals().put("temperature", 38.3);
        state.getLatestVitals().put("oxygensaturation", 93);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Mildly elevated scores: NEWS2=5, qSOFA=0.
     * Expected classification: SOFT_FLAG
     */
    public static ClinicalEvent mildElevationSoftFlagEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(5);
        state.setQsofaScore(0);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Stable patient: NEWS2=1, qSOFA=0, normal labs.
     * Expected classification: ROUTINE
     */
    public static ClinicalEvent stableRoutineEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.getLatestVitals().put("heartrate", 72);
        state.getLatestVitals().put("systolicbloodpressure", 122);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    // ── ClinicalEvent builders from PatternEvent ──

    public static ClinicalEvent criticalDeteriorationPatternEvent(String patientId) {
        PatternEvent pe = PatternEvent.builder()
            .patternType("CLINICAL_DETERIORATION")
            .patientId(patientId)
            .severity("CRITICAL")
            .confidence(0.92)
            .detectionTime(System.currentTimeMillis())
            .recommendedActions(List.of("IMMEDIATE_ASSESSMENT_REQUIRED", "ESCALATE_TO_PHYSICIAN"))
            .build();
        pe.addTag("SEVERITY_ESCALATION");
        return ClinicalEvent.fromPattern(pe);
    }

    public static ClinicalEvent highSeverityPatternEvent(String patientId, String patternType) {
        PatternEvent pe = PatternEvent.builder()
            .patternType(patternType)
            .patientId(patientId)
            .severity("HIGH")
            .confidence(0.85)
            .detectionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPattern(pe);
    }

    public static ClinicalEvent moderatePatternEvent(String patientId, String patternType) {
        PatternEvent pe = PatternEvent.builder()
            .patternType(patternType)
            .patientId(patientId)
            .severity("MODERATE")
            .confidence(0.70)
            .detectionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPattern(pe);
    }

    // ── ClinicalEvent builders from MLPrediction ──

    public static ClinicalEvent sepsisHighRiskPrediction(String patientId, double calibratedScore) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("sepsis")
            .calibratedScore(calibratedScore)
            .riskLevel(calibratedScore >= 0.60 ? "CRITICAL" : calibratedScore >= 0.35 ? "HIGH" : "MODERATE")
            .contextDepth("ESTABLISHED")
            .triggerSource("CDS_EVENT")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }

    public static ClinicalEvent deteriorationPrediction(String patientId, double calibratedScore) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("deterioration")
            .calibratedScore(calibratedScore)
            .riskLevel(calibratedScore >= 0.45 ? "HIGH" : "MODERATE")
            .contextDepth("ESTABLISHED")
            .triggerSource("CDS_EVENT")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }

    public static ClinicalEvent lowRiskPrediction(String patientId) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("readmission")
            .calibratedScore(0.15)
            .riskLevel("LOW")
            .contextDepth("ESTABLISHED")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }
}
