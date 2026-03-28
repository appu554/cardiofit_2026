package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.PatternEvent;

import java.util.*;

/**
 * Test data factory for Module 4 Pattern Detection tests.
 */
public class Module4TestBuilder {

    // ── SemanticEvent builders ─────────────────────────────────

    /**
     * Baseline vital sign event — low risk, clinical significance ~0.15
     * NEWS2=2, qSOFA=0, acuity=2.0 → significance ≈ 0.19
     */
    public static SemanticEvent baselineVitalEvent(String patientId) {
        SemanticEvent se = new SemanticEvent();
        se.setId(UUID.randomUUID().toString());
        se.setPatientId(patientId);
        se.setEventTime(System.currentTimeMillis());
        se.setProcessingTime(System.currentTimeMillis());
        se.setEventType(EventType.VITAL_SIGN);

        Map<String, Object> annotations = new HashMap<>();
        annotations.put("clinical_significance", 0.19);
        annotations.put("risk_level", "low");
        se.setSemanticAnnotations(annotations);

        Map<String, Object> clinicalData = new HashMap<>();
        Map<String, Object> vitalSigns = new HashMap<>();
        vitalSigns.put("heart_rate", 78.0);
        vitalSigns.put("systolic_bp", 128.0);
        vitalSigns.put("diastolic_bp", 82.0);
        vitalSigns.put("respiratory_rate", 16.0);
        vitalSigns.put("oxygen_saturation", 97.0);
        vitalSigns.put("temperature", 37.0);
        clinicalData.put("vitalSigns", vitalSigns);
        se.setClinicalData(clinicalData);

        return se;
    }

    /**
     * Warning vital sign event — moderate risk, clinical significance ~0.55
     * NEWS2=5, qSOFA=1, acuity=5.0 → significance ≈ 0.60
     */
    @SuppressWarnings("unchecked")
    public static SemanticEvent warningVitalEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.getSemanticAnnotations().put("clinical_significance", 0.60);
        se.getSemanticAnnotations().put("risk_level", "moderate");

        Map<String, Object> vitalSigns = (Map<String, Object>) se.getClinicalData().get("vitalSigns");
        vitalSigns.put("heart_rate", 105.0);
        vitalSigns.put("systolic_bp", 95.0);
        vitalSigns.put("respiratory_rate", 22.0);
        vitalSigns.put("temperature", 38.5);
        vitalSigns.put("oxygen_saturation", 93.0);

        return se;
    }

    /**
     * Critical vital sign event — high risk, clinical significance ~0.85
     * NEWS2=9, qSOFA=2, acuity=8.5 → significance ≈ 0.87
     */
    @SuppressWarnings("unchecked")
    public static SemanticEvent criticalVitalEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.getSemanticAnnotations().put("clinical_significance", 0.87);
        se.getSemanticAnnotations().put("risk_level", "high");

        Map<String, Object> vitalSigns = (Map<String, Object>) se.getClinicalData().get("vitalSigns");
        vitalSigns.put("heart_rate", 125.0);
        vitalSigns.put("systolic_bp", 82.0);
        vitalSigns.put("respiratory_rate", 28.0);
        vitalSigns.put("temperature", 39.5);
        vitalSigns.put("oxygen_saturation", 88.0);

        Map<String, Object> labValues = new HashMap<>();
        labValues.put("lactate", 4.5);
        labValues.put("wbc_count", 18000);
        se.getClinicalData().put("labValues", labValues);

        return se;
    }

    /**
     * Medication ordered event
     */
    public static SemanticEvent medicationOrderedEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.MEDICATION_ORDERED);
        se.getSemanticAnnotations().put("clinical_significance", 0.3);
        return se;
    }

    /**
     * Medication missed event
     */
    public static SemanticEvent medicationMissedEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.MEDICATION_MISSED);
        se.getSemanticAnnotations().put("clinical_significance", 0.5);
        return se;
    }

    /**
     * Medication administered event
     */
    public static SemanticEvent medicationAdministeredEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.MEDICATION_ADMINISTERED);
        se.getSemanticAnnotations().put("clinical_significance", 0.2);
        return se;
    }

    /**
     * Patient admission event
     */
    public static SemanticEvent admissionEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.PATIENT_ADMISSION);
        se.getSemanticAnnotations().put("clinical_significance", 0.4);
        return se;
    }

    /**
     * Lab result event
     */
    public static SemanticEvent labResultEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.LAB_RESULT);
        se.getSemanticAnnotations().put("clinical_significance", 0.35);
        return se;
    }

    /**
     * Procedure scheduled event
     */
    public static SemanticEvent procedureScheduledEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setEventType(EventType.PROCEDURE_SCHEDULED);
        se.getSemanticAnnotations().put("clinical_significance", 0.3);
        return se;
    }

    /**
     * Glycaemic declining domain event (V4 cross-domain)
     */
    public static SemanticEvent glycaemicDecliningEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setClinicalDomain("GLYCAEMIC");
        se.setTrajectoryClass("DECLINING");
        se.getSemanticAnnotations().put("clinical_significance", 0.65);
        return se;
    }

    /**
     * Hemodynamic declining domain event (V4 cross-domain)
     */
    public static SemanticEvent hemodynamicDecliningEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        se.setClinicalDomain("HEMODYNAMIC");
        se.setTrajectoryClass("DECLINING");
        se.getSemanticAnnotations().put("clinical_significance", 0.70);
        return se;
    }

    // ── PatientContextState builders (for CDS→Semantic conversion tests) ──

    /**
     * Build a PatientContextState with NEWS2=5, qSOFA=1, acuity=5.5
     * representing a moderate-risk patient with vitals + labs
     */
    public static PatientContextState moderateRiskPatientState(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 102);
        state.getLatestVitals().put("systolicbp", 95);
        state.getLatestVitals().put("diastolicbp", 62);
        state.getLatestVitals().put("respiratoryrate", 22);
        state.getLatestVitals().put("temperature", 38.2);
        state.getLatestVitals().put("oxygensaturation", 94);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(2.8);
        lactate.setLabType("Lactate");
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        LabResult creatinine = new LabResult();
        creatinine.setLabCode("2160-0");
        creatinine.setValue(1.6);
        creatinine.setLabType("Creatinine");
        creatinine.setUnit("mg/dL");
        state.getRecentLabs().put("2160-0", creatinine);

        state.setNews2Score(5);
        state.setQsofaScore(1);
        state.setCombinedAcuityScore(5.5);

        return state;
    }

    /**
     * Build a PatientContextState with NEWS2=10, qSOFA=2, acuity=9.0
     * representing a high-risk septic patient
     */
    public static PatientContextState highRiskSepticPatientState(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 125);
        state.getLatestVitals().put("systolicbp", 82);
        state.getLatestVitals().put("diastolicbp", 50);
        state.getLatestVitals().put("respiratoryrate", 28);
        state.getLatestVitals().put("temperature", 39.5);
        state.getLatestVitals().put("oxygensaturation", 88);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(4.5);
        lactate.setLabType("Lactate");
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        LabResult wbc = new LabResult();
        wbc.setLabCode("6690-2");
        wbc.setValue(18.5);
        wbc.setLabType("WBC");
        wbc.setUnit("10^3/uL");
        state.getRecentLabs().put("6690-2", wbc);

        LabResult procalcitonin = new LabResult();
        procalcitonin.setLabCode("33959-8");
        procalcitonin.setValue(8.2);
        procalcitonin.setLabType("Procalcitonin");
        procalcitonin.setUnit("ng/mL");
        state.getRecentLabs().put("33959-8", procalcitonin);

        state.setNews2Score(10);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(9.0);

        return state;
    }

    /**
     * Build a PatientContextState with NEWS2=1, qSOFA=0, acuity=1.5
     * representing a healthy baseline patient
     */
    public static PatientContextState lowRiskPatientState(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 72);
        state.getLatestVitals().put("systolicbp", 122);
        state.getLatestVitals().put("diastolicbp", 78);
        state.getLatestVitals().put("respiratoryrate", 14);
        state.getLatestVitals().put("temperature", 36.8);
        state.getLatestVitals().put("oxygensaturation", 98);

        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setCombinedAcuityScore(1.5);

        return state;
    }

    // ── PatternEvent builders (for deduplication tests) ─────────

    /**
     * Build a HIGH severity deterioration pattern
     */
    public static PatternEvent deteriorationPattern(String patientId, String severity, double confidence) {
        PatternEvent pe = new PatternEvent();
        pe.setId(UUID.randomUUID().toString());
        pe.setPatternType("CLINICAL_DETERIORATION");
        pe.setPatientId(patientId);
        pe.setDetectionTime(System.currentTimeMillis());
        pe.setSeverity(severity);
        pe.setConfidence(confidence);
        pe.addInvolvedEvent("evt-" + UUID.randomUUID().toString().substring(0, 8));

        PatternEvent.PatternMetadata metadata = new PatternEvent.PatternMetadata();
        metadata.setAlgorithm("CEP_DETERIORATION");
        metadata.setVersion("1.0.0");
        metadata.setProcessingTime(1.5);
        pe.setPatternMetadata(metadata);

        pe.setRecommendedActions(List.of(
            "IMMEDIATE_ASSESSMENT_REQUIRED",
            "ESCALATE_TO_PHYSICIAN"
        ));
        return pe;
    }

    /**
     * Build a MEWS alert for mapper tests.
     * Uses urgency field to convey the risk level string.
     */
    public static MEWSAlert mewsAlert(String patientId, int score, String risk) {
        MEWSAlert alert = new MEWSAlert();
        alert.setPatientId(patientId);
        alert.setMewsScore(score);
        alert.setUrgency(risk);
        alert.setTimestamp(System.currentTimeMillis());
        return alert;
    }

    /**
     * Build a LabTrendAlert for mapper tests.
     * Sets percentChange (required by mapper's severity logic), interpretation (required for
     * patternDetails description), and representative first/last values.
     */
    public static LabTrendAlert labTrendAlert(String patientId, String labName, String direction) {
        LabTrendAlert alert = new LabTrendAlert();
        alert.setPatientId(patientId);
        alert.setLabName(labName);
        alert.setTrendDirection(direction);
        alert.setTimestamp(System.currentTimeMillis());
        alert.setFirstValue(0.9);
        alert.setLastValue(1.4);
        alert.setAbsoluteChange(0.5);
        alert.setPercentChange(55.6);
        alert.setInterpretation("Creatinine rising — monitor renal function");
        return alert;
    }

    /**
     * Build a VitalVariabilityAlert for mapper tests.
     * Note: field is vitalSignName (not vitalName).
     */
    public static VitalVariabilityAlert vitalVariabilityAlert(String patientId, String vitalName, double cv) {
        VitalVariabilityAlert alert = new VitalVariabilityAlert();
        alert.setPatientId(patientId);
        alert.setVitalSignName(vitalName);
        alert.setCoefficientOfVariation(cv);
        alert.setTimestamp(System.currentTimeMillis());
        alert.setVariabilityLevel("HIGH");
        return alert;
    }

    // ── Helpers for building CEP match maps ─────────────────────

    /**
     * Build a deterioration CEP match map: baseline → warning → critical
     */
    public static Map<String, List<SemanticEvent>> deteriorationMatchMap(String patientId) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent baseline = baselineVitalEvent(patientId);
        baseline.setEventTime(System.currentTimeMillis() - 3600_000); // 1h ago

        SemanticEvent warning = warningVitalEvent(patientId);
        warning.setEventTime(System.currentTimeMillis() - 1800_000); // 30min ago

        SemanticEvent critical = criticalVitalEvent(patientId);
        critical.setEventTime(System.currentTimeMillis());

        map.put("baseline", List.of(baseline));
        map.put("warning", List.of(warning));
        map.put("critical", List.of(critical));
        return map;
    }

    /**
     * Build a cross-domain CEP match map: glycaemic_decline → hemodynamic_decline
     */
    public static Map<String, List<SemanticEvent>> crossDomainMatchMap(String patientId) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent glycaemic = glycaemicDecliningEvent(patientId);
        glycaemic.setEventTime(System.currentTimeMillis() - 7200_000); // 2h ago

        SemanticEvent hemodynamic = hemodynamicDecliningEvent(patientId);
        hemodynamic.setEventTime(System.currentTimeMillis());

        map.put("glycaemic_decline", List.of(glycaemic));
        map.put("hemodynamic_decline", List.of(hemodynamic));
        return map;
    }

    /**
     * Build a medication adherence CEP match map: ordered → administered/missed
     */
    public static Map<String, List<SemanticEvent>> medicationMatchMap(String patientId, boolean wasMissed) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent ordered = medicationOrderedEvent(patientId);
        ordered.setEventTime(System.currentTimeMillis() - 3600_000);

        SemanticEvent administered = wasMissed
            ? medicationMissedEvent(patientId)
            : medicationAdministeredEvent(patientId);
        administered.setEventTime(System.currentTimeMillis());

        map.put("medication_ordered", List.of(ordered));
        map.put("administration_due", List.of(administered));
        return map;
    }

    /**
     * Build a vital trend CEP match map: vital1 → vital2 → vital3
     * If deteriorating=true, clinical significance increases across the 3 events
     */
    public static Map<String, List<SemanticEvent>> vitalTrendMatchMap(String patientId, boolean deteriorating) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent v1 = baselineVitalEvent(patientId);
        v1.setEventTime(System.currentTimeMillis() - 1800_000);
        v1.getSemanticAnnotations().put("clinical_significance", deteriorating ? 0.3 : 0.7);

        SemanticEvent v2 = baselineVitalEvent(patientId);
        v2.setEventTime(System.currentTimeMillis() - 900_000);
        v2.getSemanticAnnotations().put("clinical_significance", deteriorating ? 0.5 : 0.5);

        SemanticEvent v3 = baselineVitalEvent(patientId);
        v3.setEventTime(System.currentTimeMillis());
        v3.getSemanticAnnotations().put("clinical_significance", deteriorating ? 0.8 : 0.3);

        map.put("vital1", List.of(v1));
        map.put("vital2", List.of(v2));
        map.put("vital3", List.of(v3));
        return map;
    }

    /**
     * Build a pathway compliance CEP match map: admission → assessment → intervention
     */
    public static Map<String, List<SemanticEvent>> pathwayMatchMap(String patientId) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent admission = admissionEvent(patientId);
        admission.setEventTime(System.currentTimeMillis() - 7200_000);

        SemanticEvent assessment = labResultEvent(patientId);
        assessment.setEventTime(System.currentTimeMillis() - 3600_000);

        SemanticEvent intervention = procedureScheduledEvent(patientId);
        intervention.setEventTime(System.currentTimeMillis());

        map.put("admission", List.of(admission));
        map.put("assessment", List.of(assessment));
        map.put("intervention", List.of(intervention));
        return map;
    }

    // ── EnrichedEvent builders (for ClinicalPatterns AKI tests) ──────

    /**
     * Build an EnrichedEvent with baseline creatinine for AKI pattern testing.
     * Creatinine = 1.0 mg/dL (normal baseline).
     */
    public static EnrichedEvent baselineCreatinineEvent(String patientId) {
        EnrichedEvent event = new EnrichedEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventTime(System.currentTimeMillis() - 86400_000); // 24h ago

        Map<String, Object> payload = new HashMap<>();
        Map<String, Object> labValues = new HashMap<>();
        labValues.put("creatinine", 1.0);
        payload.put("labValues", labValues);
        event.setPayload(payload);

        RiskIndicators risk = new RiskIndicators();
        event.setRiskIndicators(risk);

        return event;
    }

    /**
     * Build an EnrichedEvent with elevated creatinine for AKI KDIGO Stage 1.
     * Creatinine = 1.8 mg/dL (1.8x baseline of 1.0 → Stage 1).
     */
    public static EnrichedEvent elevatedCreatinineEvent(String patientId, double creatinineValue) {
        EnrichedEvent event = new EnrichedEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventTime(System.currentTimeMillis());

        Map<String, Object> payload = new HashMap<>();
        Map<String, Object> labValues = new HashMap<>();
        labValues.put("creatinine", creatinineValue);
        payload.put("labValues", labValues);
        event.setPayload(payload);

        RiskIndicators risk = new RiskIndicators();
        event.setRiskIndicators(risk);

        return event;
    }

    /**
     * Build an EnrichedEvent with AKI risk factors (hypotension + nephrotoxic meds).
     */
    public static EnrichedEvent akiRiskFactorEvent(String patientId) {
        EnrichedEvent event = new EnrichedEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventTime(System.currentTimeMillis());

        Map<String, Object> payload = new HashMap<>();
        payload.put("labValues", new HashMap<>());
        event.setPayload(payload);

        RiskIndicators risk = new RiskIndicators();
        risk.setHypotension(true);
        risk.setOnNephrotoxicMeds(true);
        event.setRiskIndicators(risk);

        return event;
    }

    // ── Sepsis-specific SemanticEvent builders ──────────────────────

    /**
     * Build a sepsis baseline SemanticEvent with normal vitals.
     */
    @SuppressWarnings("unchecked")
    public static SemanticEvent sepsisBaselineEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        Map<String, Object> vitalSigns = (Map<String, Object>) se.getClinicalData().get("vitalSigns");
        vitalSigns.put("heart_rate", 80);
        vitalSigns.put("systolic_bp", 120);
        vitalSigns.put("respiratory_rate", 16);
        vitalSigns.put("temperature", 37.0);
        vitalSigns.put("oxygen_saturation", 98);

        Map<String, Object> labValues = new HashMap<>();
        labValues.put("lactate", 1.0);
        labValues.put("creatinine", 0.9);
        labValues.put("platelets", 250000);
        se.getClinicalData().put("labValues", labValues);

        return se;
    }

    /**
     * Build a sepsis early warning SemanticEvent with elevated vitals (qSOFA >= 2).
     * RR=24, SBP=95 → qSOFA = 2 (RR>=22 + SBP<=100).
     */
    @SuppressWarnings("unchecked")
    public static SemanticEvent sepsisWarningEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        Map<String, Object> vitalSigns = (Map<String, Object>) se.getClinicalData().get("vitalSigns");
        vitalSigns.put("heart_rate", 110);
        vitalSigns.put("systolic_bp", 95);
        vitalSigns.put("respiratory_rate", 24);
        vitalSigns.put("temperature", 38.5);
        vitalSigns.put("oxygen_saturation", 93);

        Map<String, Object> labValues = new HashMap<>();
        labValues.put("lactate", 2.5);
        labValues.put("creatinine", 1.2);
        labValues.put("platelets", 180000);
        se.getClinicalData().put("labValues", labValues);

        return se;
    }

    /**
     * Build a sepsis deterioration SemanticEvent with organ dysfunction.
     * Lactate > 4.0 → organ dysfunction = true.
     */
    @SuppressWarnings("unchecked")
    public static SemanticEvent sepsisDeteriorationEvent(String patientId) {
        SemanticEvent se = baselineVitalEvent(patientId);
        Map<String, Object> vitalSigns = (Map<String, Object>) se.getClinicalData().get("vitalSigns");
        vitalSigns.put("heart_rate", 130);
        vitalSigns.put("systolic_bp", 80);
        vitalSigns.put("respiratory_rate", 30);
        vitalSigns.put("temperature", 39.5);
        vitalSigns.put("oxygen_saturation", 88);

        Map<String, Object> labValues = new HashMap<>();
        labValues.put("lactate", 4.5);
        labValues.put("creatinine", 2.5);
        labValues.put("platelets", 80000);
        se.getClinicalData().put("labValues", labValues);

        return se;
    }

    /**
     * Build a rapid deterioration set of SemanticEvents.
     * HR baseline=80 → HR elevated=125 → RR elevated=30 → O2sat decreased=85.
     */
    @SuppressWarnings("unchecked")
    public static Map<String, List<SemanticEvent>> rapidDeteriorationMatchMap(String patientId) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();

        SemanticEvent hrBaseline = baselineVitalEvent(patientId);
        ((Map<String, Object>) hrBaseline.getClinicalData().get("vitalSigns")).put("heart_rate", 80.0);
        hrBaseline.setEventTime(System.currentTimeMillis() - 1800_000);

        SemanticEvent hrElevated = baselineVitalEvent(patientId);
        ((Map<String, Object>) hrElevated.getClinicalData().get("vitalSigns")).put("heart_rate", 125.0);
        hrElevated.setEventTime(System.currentTimeMillis() - 1200_000);

        SemanticEvent rrElevated = baselineVitalEvent(patientId);
        ((Map<String, Object>) rrElevated.getClinicalData().get("vitalSigns")).put("respiratory_rate", 30.0);
        rrElevated.setEventTime(System.currentTimeMillis() - 600_000);

        SemanticEvent o2satDecreased = baselineVitalEvent(patientId);
        ((Map<String, Object>) o2satDecreased.getClinicalData().get("vitalSigns")).put("oxygen_saturation", 85.0);
        o2satDecreased.setEventTime(System.currentTimeMillis());

        map.put("hr_baseline", List.of(hrBaseline));
        map.put("hr_elevated", List.of(hrElevated));
        map.put("rr_elevated", List.of(rrElevated));
        map.put("o2sat_decreased", List.of(o2satDecreased));
        return map;
    }

    /**
     * Build a drug-lab monitoring match map with a medication started event.
     */
    public static Map<String, List<SemanticEvent>> drugLabMonitoringMatchMap(String patientId, String medicationName) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();

        SemanticEvent medEvent = baselineVitalEvent(patientId);
        medEvent.setEventType(EventType.MEDICATION_ORDERED);
        Map<String, Object> medData = new HashMap<>();
        medData.put("medication_name", medicationName);
        medEvent.getClinicalData().put("medications", medData);

        map.put("high_risk_med_started", List.of(medEvent));
        return map;
    }

    /**
     * Build a sepsis pathway compliance match map.
     * @param cultureMinsAfterDx minutes from diagnosis to blood cultures
     * @param abxMinsAfterDx minutes from diagnosis to antibiotics
     */
    public static Map<String, List<SemanticEvent>> sepsisPathwayMatchMap(
            String patientId, double cultureMinsAfterDx, double abxMinsAfterDx) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        long baseTime = System.currentTimeMillis() - 7200_000;

        SemanticEvent diagnosis = baselineVitalEvent(patientId);
        diagnosis.setEventTime(baseTime);

        SemanticEvent cultures = baselineVitalEvent(patientId);
        cultures.setEventTime(baseTime + (long)(cultureMinsAfterDx * 60_000));

        SemanticEvent antibiotics = baselineVitalEvent(patientId);
        antibiotics.setEventTime(baseTime + (long)(abxMinsAfterDx * 60_000));

        map.put("sepsis_diagnosis", List.of(diagnosis));
        map.put("blood_cultures_ordered", List.of(cultures));
        map.put("antibiotics_started", List.of(antibiotics));
        return map;
    }

    /**
     * Build an AKI CEP match map for ClinicalPatterns.AKIPatternSelectFunction.
     */
    public static Map<String, List<EnrichedEvent>> akiMatchMap(String patientId, double elevatedCreatinine) {
        Map<String, List<EnrichedEvent>> map = new HashMap<>();
        map.put("baseline_creatinine", List.of(baselineCreatinineEvent(patientId)));
        map.put("elevated_creatinine", List.of(elevatedCreatinineEvent(patientId, elevatedCreatinine)));
        map.put("risk_factor_present", List.of(akiRiskFactorEvent(patientId)));
        return map;
    }

    /**
     * Build a sepsis CEP match map for ClinicalPatterns.SepsisPatternSelectFunction.
     */
    public static Map<String, List<SemanticEvent>> sepsisMatchMap(String patientId) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();
        SemanticEvent baseline = sepsisBaselineEvent(patientId);
        baseline.setEventTime(System.currentTimeMillis() - 3600_000);

        SemanticEvent warning = sepsisWarningEvent(patientId);
        warning.setEventTime(System.currentTimeMillis() - 1800_000);

        SemanticEvent deterioration = sepsisDeteriorationEvent(patientId);
        deterioration.setEventTime(System.currentTimeMillis());

        map.put("baseline", List.of(baseline));
        map.put("early_warning", List.of(warning));
        map.put("deterioration", List.of(deterioration));
        return map;
    }
}
