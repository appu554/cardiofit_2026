# Module 4 Pattern Detection — Gap Closure Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the identified gaps in Module 4 (Pattern Detection) — primarily test coverage for ~2500 lines of untested production code, deduplication severity-escalation handling, and production log cleanup. (AKI pattern re-enablement is out of scope — it was deliberately disabled and requires separate investigation.)

**Architecture:** Module 4 is a 3-layer pattern detection system: Layer 1 (instant state assessment), Layer 2 (Flink CEP temporal patterns), Layer 3 (ML prediction integration). It reads CDSEvent from Module 3 via `comprehensive-cds-events.v1`, converts to SemanticEvent internally, runs 6 CEP patterns + 3 windowed analytics + 3 advanced analytics engines, deduplicates via `PatternDeduplicationFunction`, classifies via `PatternClassificationFunction`, and sinks to 5 Kafka topics. All tests are JUnit 5 without Flink MiniCluster (testing static logic extracted from operators).

**Tech Stack:** Java 17, Flink 2.1.0 (flink-cep), JUnit 5, Jackson 2.17

---

## File Structure Overview

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── operators/Module4_PatternDetection.java        # MODIFY: clean debug logs, delegate scoring
│   ├── functions/PatternDeduplicationFunction.java     # MODIFY: severity escalation, configurable cooldown
│   └── patterns/ClinicalPatterns.java                  # READ ONLY (reference for test assertions)
├── test/java/com/cardiofit/flink/
│   ├── operators/Module4CDSConversionTest.java         # CREATE: CDS→Semantic + clinical significance tests
│   ├── operators/Module4CEPSelectFunctionTest.java     # CREATE: all 6 CEP select function tests
│   ├── operators/Module4WindowFunctionTest.java        # CREATE: 3 window function tests
│   ├── operators/Module4AlertMapperTest.java           # CREATE: MEWS/LabTrend/VitalVariability mapper tests
│   ├── operators/Module4DeduplicationTest.java         # CREATE: dedup with escalation + cooldown tests
│   ├── operators/Module4ClassificationTest.java        # CREATE: side-output routing tests
│   └── builders/Module4TestBuilder.java                # CREATE: test data factory
```

---

### Task 1: Module 4 Test Data Factory

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/builders/Module4TestBuilder.java`

**Context:** Module 4's main operator converts `Module3_ComprehensiveCDS.CDSEvent` to `SemanticEvent` internally. Tests need synthetic CDSEvents, SemanticEvents, PatternEvents, and alert objects. This factory provides all test data for Tasks 2-7. Follow the same pattern as `Module3TestBuilder.java`.

- [ ] **Step 1: Write the test factory class**

```java
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
     * Build a MEWS alert for mapper tests
     */
    public static MEWSAlert mewsAlert(String patientId, int score, String risk) {
        MEWSAlert alert = new MEWSAlert();
        alert.setPatientId(patientId);
        alert.setMewsScore(score);
        alert.setRiskLevel(risk);
        alert.setTimestamp(System.currentTimeMillis());
        return alert;
    }

    /**
     * Build a LabTrendAlert for mapper tests
     */
    public static LabTrendAlert labTrendAlert(String patientId, String labName, String direction) {
        LabTrendAlert alert = new LabTrendAlert();
        alert.setPatientId(patientId);
        alert.setLabName(labName);
        alert.setTrendDirection(direction);
        alert.setTimestamp(System.currentTimeMillis());
        return alert;
    }

    /**
     * Build a VitalVariabilityAlert for mapper tests
     */
    public static VitalVariabilityAlert vitalVariabilityAlert(String patientId, String vitalName, double cv) {
        VitalVariabilityAlert alert = new VitalVariabilityAlert();
        alert.setPatientId(patientId);
        alert.setVitalName(vitalName);
        alert.setCoefficientOfVariation(cv);
        alert.setTimestamp(System.currentTimeMillis());
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
}
```

- [ ] **Step 2: Run compile check**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS (or check for import issues with MEWSAlert/LabTrendAlert/VitalVariabilityAlert setters — adapt builder if needed)

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/builders/Module4TestBuilder.java
git commit -m "test(module4): add Module4TestBuilder factory for gap closure tests"
```

---

### Task 2: Test CDS→Semantic Conversion and Clinical Significance Calculation

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4CDSConversionTest.java`
- Read: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java:849-921` (calculateClinicalSignificance + determineRiskLevel)

**Context:** `calculateClinicalSignificance` and `determineRiskLevel` are private static methods in Module4_PatternDetection. They control CEP pattern matching thresholds (a significance of 0.6 matches "warning" in the deterioration pattern, 0.8 matches "critical"). Currently untested. Since they're private, test them indirectly via `convertCDSEventToSemanticEvent` which is also private — we'll need to extract them or use reflection. The cleanest approach: extract these two methods into a package-private helper class `Module4ClinicalScoring` so they're directly testable without reflection.

- [ ] **Step 1: Extract clinical scoring methods to testable class**

Create `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4ClinicalScoring.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.SimpleAlert;
import java.util.Set;

/**
 * Clinical significance and risk level scoring for Module 4 CEP pattern matching.
 * Extracted from Module4_PatternDetection for testability.
 *
 * Maps clinical scores to CEP pattern thresholds:
 * - 0.0-0.3: Low significance (baseline candidate)
 * - 0.3-0.6: Moderate significance (warning candidate)
 * - 0.6-0.8: High significance (early deterioration)
 * - 0.8-1.0: Critical significance (severe deterioration)
 */
class Module4ClinicalScoring {

    static double calculateClinicalSignificance(int news2Score, int qsofaScore, double acuityScore) {
        double significance = 0.0;

        // NEWS2 contribution (50% weight)
        if (news2Score >= 10) {
            significance += 0.5;
        } else if (news2Score >= 7) {
            significance += 0.4;
        } else if (news2Score >= 5) {
            significance += 0.35;
        } else if (news2Score > 0) {
            significance += 0.15;
        }

        // qSOFA contribution (30% weight)
        if (qsofaScore >= 2) {
            significance += 0.3;
        } else if (qsofaScore == 1) {
            significance += 0.15;
        }

        // Acuity contribution (20% weight)
        significance += (acuityScore / 10.0) * 0.2;

        return Math.min(1.0, Math.max(0.0, significance));
    }

    static String determineRiskLevel(int news2Score, int qsofaScore, Set<SimpleAlert> alerts) {
        if (news2Score >= 10 || qsofaScore >= 2) {
            return "high";
        }

        if (alerts != null && !alerts.isEmpty()) {
            long criticalAlertCount = alerts.stream()
                .filter(alert -> alert.getSeverity() != null)
                .filter(alert -> alert.getSeverity().equals("CRITICAL") ||
                    (alert.getPriorityLevel() != null && alert.getPriorityLevel().equals("CRITICAL")))
                .count();
            if (criticalAlertCount >= 2) {
                return "high";
            }
        }

        if (news2Score >= 5 || qsofaScore >= 1) {
            return "moderate";
        }

        if (news2Score <= 4 && qsofaScore == 0) {
            return "low";
        }

        return "unknown";
    }
}
```

- [ ] **Step 2: Update Module4_PatternDetection to delegate to extracted class**

In `Module4_PatternDetection.java`, replace the two private methods (lines ~849-921) with delegation:

```java
    private static double calculateClinicalSignificance(int news2Score, int qsofaScore, double acuityScore) {
        return Module4ClinicalScoring.calculateClinicalSignificance(news2Score, qsofaScore, acuityScore);
    }

    private static String determineRiskLevel(int news2Score, int qsofaScore, java.util.Set<com.cardiofit.flink.models.SimpleAlert> alerts) {
        return Module4ClinicalScoring.determineRiskLevel(news2Score, qsofaScore, alerts);
    }
```

- [ ] **Step 3: Write the failing tests**

Create `Module4CDSConversionTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.SimpleAlert;
import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 clinical significance calculation and risk level determination.
 * These functions control CEP pattern matching thresholds.
 */
public class Module4CDSConversionTest {

    // ── Clinical Significance ─────────────────────────────────

    @Test
    void significance_lowRisk_allZeros() {
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(0, 0, 0.0);
        assertEquals(0.0, sig, 0.001, "All zeros should produce 0.0 significance");
    }

    @Test
    void significance_lowRisk_news2Under5() {
        // NEWS2=3 → 0.15, qSOFA=0, acuity=2.0 → 0.04 = 0.19
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(3, 0, 2.0);
        assertTrue(sig > 0.1 && sig < 0.3,
            "NEWS2=3, qSOFA=0, acuity=2.0 should be LOW range (0.1-0.3), got " + sig);
    }

    @Test
    void significance_moderate_news5_qsofa1() {
        // NEWS2=5 → 0.35, qSOFA=1 → 0.15, acuity=5.0 → 0.10 = 0.60
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(5, 1, 5.0);
        assertTrue(sig >= 0.55 && sig <= 0.65,
            "NEWS2=5, qSOFA=1, acuity=5.0 should be ~0.60, got " + sig);
    }

    @Test
    void significance_high_news7_qsofa2() {
        // NEWS2=7 → 0.40, qSOFA=2 → 0.30, acuity=7.0 → 0.14 = 0.84
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(7, 2, 7.0);
        assertTrue(sig >= 0.80 && sig <= 0.90,
            "NEWS2=7, qSOFA=2, acuity=7.0 should be ~0.84, got " + sig);
    }

    @Test
    void significance_critical_news10_qsofa2() {
        // NEWS2=10 → 0.50, qSOFA=2 → 0.30, acuity=9.0 → 0.18 = 0.98
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(10, 2, 9.0);
        assertTrue(sig >= 0.90 && sig <= 1.0,
            "NEWS2=10, qSOFA=2, acuity=9.0 should be CRITICAL (>0.90), got " + sig);
    }

    @Test
    void significance_capped_at_1() {
        // NEWS2=15 → 0.50, qSOFA=3 → 0.30, acuity=10.0 → 0.20 = 1.0
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(15, 3, 10.0);
        assertEquals(1.0, sig, 0.001, "Significance must not exceed 1.0");
    }

    // ── Risk Level Determination ──────────────────────────────

    @Test
    void riskLevel_high_whenNEWS10OrMore() {
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(10, 0, null));
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(12, 0, null));
    }

    @Test
    void riskLevel_high_whenQSOFA2OrMore() {
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(3, 2, null));
    }

    @Test
    void riskLevel_high_when2CriticalAlerts() {
        Set<SimpleAlert> alerts = new HashSet<>();
        SimpleAlert a1 = new SimpleAlert();
        a1.setSeverity("CRITICAL");
        SimpleAlert a2 = new SimpleAlert();
        a2.setSeverity("CRITICAL");
        alerts.add(a1);
        alerts.add(a2);
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(4, 0, alerts));
    }

    @Test
    void riskLevel_moderate_whenNEWS5to9() {
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(5, 0, null));
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(9, 0, null));
    }

    @Test
    void riskLevel_moderate_whenQSOFA1() {
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(3, 1, null));
    }

    @Test
    void riskLevel_low_whenNEWS4OrLess_qSOFA0() {
        assertEquals("low", Module4ClinicalScoring.determineRiskLevel(4, 0, null));
        assertEquals("low", Module4ClinicalScoring.determineRiskLevel(0, 0, null));
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4CDSConversionTest -q 2>&1 | tail -10`
Expected: 12 tests PASS

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4ClinicalScoring.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4CDSConversionTest.java
git commit -m "test(module4): extract and test clinical significance + risk level scoring"
```

---

### Task 3: Test CEP Select Functions

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4CEPSelectFunctionTest.java`
- Read: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java:1131-1344` (all select functions)

**Context:** CEP select functions receive a `Map<String, List<SemanticEvent>>` of matched events and produce a `PatternEvent`. These are pure functions — no Flink runtime needed. We test them by constructing the match map directly and calling `select()`. There are 5 select functions in Module4_PatternDetection + 4 in ClinicalPatterns. Focus on the 5 in Module4 first (ClinicalPatterns ones are already implicitly covered by the Layer 3 integration test flow).

- [ ] **Step 1: Write the tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 CEP select functions.
 * Each function receives a CEP match map and produces a PatternEvent.
 */
public class Module4CEPSelectFunctionTest {

    // ── DeteriorationPatternSelectFunction ─────────────────────

    @Test
    void deterioration_producesHighSeverityPattern() throws Exception {
        var fn = new Module4_PatternDetection.DeteriorationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.deteriorationMatchMap("PAT-D001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("CLINICAL_DETERIORATION", result.getPatternType());
        assertEquals("PAT-D001", result.getPatientId());
        assertEquals("HIGH", result.getSeverity());
        assertEquals(0.85, result.getConfidence(), 0.01);
        assertEquals(3, result.getInvolvedEvents().size(), "Should reference baseline + warning + critical events");

        // Verify pattern details contain deterioration metrics
        assertNotNull(result.getPatternDetails().get("deterioration_rate"));
        assertNotNull(result.getPatternDetails().get("timespan_hours"));
        double rate = (double) result.getPatternDetails().get("deterioration_rate");
        assertTrue(rate > 0, "Deterioration rate should be positive (worsening)");

        // Verify recommended actions
        assertTrue(result.getRecommendedActions().contains("IMMEDIATE_ASSESSMENT_REQUIRED"));
        assertTrue(result.getRecommendedActions().contains("ESCALATE_TO_PHYSICIAN"));
    }

    // ── CrossDomainDeteriorationSelectFunction ─────────────────

    @Test
    void crossDomain_detectsGlycaemicAndHemodynamicDecline() throws Exception {
        var fn = new Module4_PatternDetection.CrossDomainDeteriorationSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.crossDomainMatchMap("PAT-CD001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("CROSS_DOMAIN_DECLINE", result.getPatternType());
        assertEquals("PAT-CD001", result.getPatientId());
        assertEquals("HIGH", result.getSeverity());
        assertEquals(2, result.getInvolvedEvents().size());

        // Verify cross-domain details
        assertEquals("DECLINING", result.getPatternDetails().get("glycaemic_trajectory"));
        assertEquals("DECLINING", result.getPatternDetails().get("hemodynamic_trajectory"));
        assertEquals("GLYCAEMIC", result.getPatternDetails().get("glycaemic_domain"));
        assertEquals("HEMODYNAMIC", result.getPatternDetails().get("hemodynamic_domain"));

        // V4-specific: MHRI recomputation action
        assertTrue(result.getRecommendedActions().contains("MHRI_RECOMPUTATION_NEEDED"));
    }

    // ── MedicationPatternSelectFunction ────────────────────────

    @Test
    void medication_missedDose_producesModerateAlert() throws Exception {
        var fn = new Module4_PatternDetection.MedicationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.medicationMatchMap("PAT-M001", true);

        PatternEvent result = fn.select(matchMap);

        assertEquals("MEDICATION_ADHERENCE", result.getPatternType());
        assertEquals("PAT-M001", result.getPatientId());
        assertEquals("MODERATE", result.getSeverity());
        assertEquals("MISSED", result.getPatternDetails().get("medication_status"));
        assertTrue(result.getRecommendedActions().contains("VERIFY_PATIENT_STATUS"));
    }

    @Test
    void medication_administered_producesLowAlert() throws Exception {
        var fn = new Module4_PatternDetection.MedicationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.medicationMatchMap("PAT-M002", false);

        PatternEvent result = fn.select(matchMap);

        assertEquals("MEDICATION_ADHERENCE", result.getPatternType());
        assertEquals("LOW", result.getSeverity());
        assertEquals("ADMINISTERED", result.getPatternDetails().get("medication_status"));
    }

    // ── VitalTrendPatternSelectFunction ────────────────────────

    @Test
    void vitalTrend_deteriorating_producesHighSeverity() throws Exception {
        var fn = new Module4_PatternDetection.VitalTrendPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.vitalTrendMatchMap("PAT-VT001", true);

        PatternEvent result = fn.select(matchMap);

        assertEquals("VITAL_SIGNS_TREND", result.getPatternType());
        assertEquals("HIGH", result.getSeverity());
        assertEquals("DETERIORATING", result.getPatternDetails().get("trend_direction"));
        assertEquals(3, result.getPatternDetails().get("reading_count"));
        assertTrue(result.getRecommendedActions().contains("INCREASE_VITAL_MONITORING"));
    }

    @Test
    void vitalTrend_improving_producesLowSeverity() throws Exception {
        var fn = new Module4_PatternDetection.VitalTrendPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.vitalTrendMatchMap("PAT-VT002", false);

        PatternEvent result = fn.select(matchMap);

        assertEquals("VITAL_SIGNS_TREND", result.getPatternType());
        assertEquals("LOW", result.getSeverity());
        assertEquals("IMPROVING", result.getPatternDetails().get("trend_direction"));
    }

    // ── PathwayCompliancePatternSelectFunction ─────────────────

    @Test
    void pathway_completed_producesLowSeverity() throws Exception {
        var fn = new Module4_PatternDetection.PathwayCompliancePatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.pathwayMatchMap("PAT-P001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("PATHWAY_COMPLIANCE", result.getPatternType());
        assertEquals("PAT-P001", result.getPatientId());
        assertEquals("LOW", result.getSeverity());
        assertTrue((boolean) result.getPatternDetails().get("pathway_completed"));
        assertNotNull(result.getPatternDetails().get("time_to_assessment"));
        assertNotNull(result.getPatternDetails().get("time_to_intervention"));
    }
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4CEPSelectFunctionTest -q 2>&1 | tail -10`
Expected: 7 tests PASS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4CEPSelectFunctionTest.java
git commit -m "test(module4): add CEP select function tests for all 5 pattern types"
```

---

### Task 4: Test Alert-to-PatternEvent Mappers

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4AlertMapperTest.java`
- Read: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java:1836-1990` (mapper inner classes)

**Context:** Three `private static` inner mapper classes convert analytics engine output (MEWSAlert, LabTrendAlert, VitalVariabilityAlert) to PatternEvent for the unified stream. Verified: all three are `private static class` (lines 1838, 1875, 1944). We must first change their visibility to package-private (`static class`) so tests in the same package can access them. This mirrors the select functions which are already `public static class`.

- [ ] **Step 1: Change mapper visibility from private to package-private**

In `Module4_PatternDetection.java`, change these three declarations:
- Line 1838: `private static class MEWSAlertToPatternEventMapper` → `static class MEWSAlertToPatternEventMapper`
- Line 1875: `private static class LabTrendAlertToPatternEventMapper` → `static class LabTrendAlertToPatternEventMapper`
- Line 1944: `private static class VitalVariabilityAlertToPatternEventMapper` → `static class VitalVariabilityAlertToPatternEventMapper`

- [ ] **Step 2: Write the tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 alert-to-PatternEvent mapper functions.
 * These convert analytics engine output into the unified PatternEvent stream.
 */
public class Module4AlertMapperTest {

    // Note: If inner classes are private, this test file must be in the same package
    // (com.cardiofit.flink.operators) to access package-private classes,
    // or we extract them. Adjust based on compilation results.

    @Test
    void mewsMapper_highScore_producesHighSeverity() throws Exception {
        // Given: MEWS score of 7 (high risk)
        MEWSAlert alert = Module4TestBuilder.mewsAlert("MEWS-001", 7, "HIGH");

        // When: mapped to PatternEvent
        // Access the inner class via reflection or direct if package-private
        var mapper = new Module4_PatternDetection.MEWSAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        // Then:
        assertNotNull(result);
        assertEquals("MEWS-001", result.getPatientId());
        assertEquals("EARLY_WARNING", result.getPatternType());
        assertNotNull(result.getPatternDetails().get("mewsScore"));
    }

    @Test
    void labTrendMapper_risingCreatinine_producesLabTrendPattern() throws Exception {
        LabTrendAlert alert = Module4TestBuilder.labTrendAlert("LAB-001", "Creatinine", "RISING");

        var mapper = new Module4_PatternDetection.LabTrendAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertNotNull(result);
        assertEquals("LAB-001", result.getPatientId());
        assertEquals("TREND_ANALYSIS", result.getPatternType());
    }

    @Test
    void vitalVariabilityMapper_highBPCV_producesVariabilityPattern() throws Exception {
        VitalVariabilityAlert alert = Module4TestBuilder.vitalVariabilityAlert("VV-001", "systolicBP", 22.5);

        var mapper = new Module4_PatternDetection.VitalVariabilityAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertNotNull(result);
        assertEquals("VV-001", result.getPatientId());
        assertEquals("VITAL_SIGNS_TREND", result.getPatternType());
    }
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4AlertMapperTest -q 2>&1 | tail -15`
Expected: 3 tests PASS

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4AlertMapperTest.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java
git commit -m "test(module4): add alert-to-PatternEvent mapper tests for MEWS, LabTrend, VitalVariability"
```

---

### Task 5: Fix Deduplication — Severity Escalation Passthrough

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/functions/PatternDeduplicationFunction.java`
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4DeduplicationTest.java`

**Context:** The current `PatternDeduplicationFunction` merges all same-type patterns within a 5-minute window. This is wrong for severity escalation: if a HIGH deterioration pattern fires at T=0 and a CRITICAL deterioration pattern fires at T=2min, the CRITICAL should be emitted immediately (not merged into the HIGH). The spec from the user's Module 4 briefing explicitly states: "Only emit a new alert if the severity has escalated (e.g., from HIGH to CRITICAL) — this ensures that a genuinely worsening situation still generates a new alert even within the cooldown window."

Additionally, the dedup function should track suppressed count and support configurable cooldown per pattern type.

**Testing scope note:** These tests validate the static helper logic (`isSeverityEscalation`, `severityIndex`, `computePatternKey`). The assembled `processElement` behavior (actual escalation passthrough through Flink keyed state) requires a Flink MiniCluster and is deferred to integration testing. The static helpers are the critical correctness unit — if `isSeverityEscalation("HIGH", "CRITICAL")` returns `true`, the `processElement` branch is straightforward.

- [ ] **Step 1: Write the failing tests first**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.functions.PatternDeduplicationFunction;
import com.cardiofit.flink.models.PatternEvent;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for PatternDeduplicationFunction improvements:
 * 1. Severity escalation passthrough (HIGH→CRITICAL emitted, not merged)
 * 2. Same-severity within window suppressed
 * 3. Different pattern types never suppressed
 */
public class Module4DeduplicationTest {

    @Test
    void severityEscalation_shouldBeDetectable() {
        // Verify the severity comparison logic that will drive escalation decisions
        // CRITICAL > HIGH > MODERATE > LOW
        assertTrue(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "CRITICAL"),
            "HIGH→CRITICAL is an escalation");
        assertTrue(PatternDeduplicationFunction.isSeverityEscalation("MODERATE", "HIGH"),
            "MODERATE→HIGH is an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("CRITICAL", "CRITICAL"),
            "Same severity is NOT an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "MODERATE"),
            "HIGH→MODERATE is a de-escalation, NOT an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "HIGH"),
            "Same severity is NOT an escalation");
    }

    @Test
    void severityIndex_ordersCorrectly() {
        // Verify the severity ordering used by the dedup function
        assertTrue(PatternDeduplicationFunction.severityIndex("CRITICAL") >
                   PatternDeduplicationFunction.severityIndex("HIGH"));
        assertTrue(PatternDeduplicationFunction.severityIndex("HIGH") >
                   PatternDeduplicationFunction.severityIndex("MODERATE"));
        assertTrue(PatternDeduplicationFunction.severityIndex("MODERATE") >
                   PatternDeduplicationFunction.severityIndex("LOW"));
        assertEquals(0, PatternDeduplicationFunction.severityIndex("UNKNOWN"));
    }

    @Test
    void patternKey_includesTypeOnly_notSeverity() {
        // The dedup key should be pattern type only (not severity) so that
        // escalations within the same type are detected
        PatternEvent highPattern = Module4TestBuilder.deteriorationPattern("P1", "HIGH", 0.85);
        PatternEvent criticalPattern = Module4TestBuilder.deteriorationPattern("P1", "CRITICAL", 0.95);

        String key1 = PatternDeduplicationFunction.computePatternKey(highPattern);
        String key2 = PatternDeduplicationFunction.computePatternKey(criticalPattern);

        assertEquals(key1, key2,
            "Same pattern type should have same dedup key regardless of severity");
        assertEquals("CLINICAL_DETERIORATION", key1);
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4DeduplicationTest -q 2>&1 | tail -10`
Expected: FAIL — `isSeverityEscalation`, `severityIndex`, `computePatternKey` don't exist yet

- [ ] **Step 3: Implement the deduplication improvements**

Update `PatternDeduplicationFunction.java` with the following changes:

```java
package com.cardiofit.flink.functions;

import com.cardiofit.flink.models.PatternEvent;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;

import java.util.*;

/**
 * Pattern Event Deduplication Function
 *
 * Prevents alert storms when multiple layers fire for the same patient.
 * Merges pattern events from different sources and boosts confidence.
 *
 * Improvements over v1:
 * - Severity escalation passthrough (HIGH→CRITICAL emitted immediately)
 * - Dedup key is pattern type only (not severity) for escalation detection
 * - Public static helpers for testability
 */
public class PatternDeduplicationFunction
    extends KeyedProcessFunction<String, PatternEvent, PatternEvent> {

    private transient ValueState<PatternEvent> lastPatternState;
    private transient MapState<String, Long> recentPatternsState;
    private transient MapState<String, String> recentSeverityState;

    private static final long DEDUP_WINDOW_MS = 5 * 60 * 1000;

    private static final List<String> SEVERITY_ORDER =
        Arrays.asList("LOW", "MODERATE", "HIGH", "CRITICAL");

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        ValueStateDescriptor<PatternEvent> lastPatternDescriptor =
            new ValueStateDescriptor<>("last-pattern", PatternEvent.class);
        lastPatternState = getRuntimeContext().getState(lastPatternDescriptor);

        MapStateDescriptor<String, Long> recentPatternsDescriptor =
            new MapStateDescriptor<>("recent-patterns", String.class, Long.class);
        recentPatternsState = getRuntimeContext().getMapState(recentPatternsDescriptor);

        MapStateDescriptor<String, String> recentSeverityDescriptor =
            new MapStateDescriptor<>("recent-severity", String.class, String.class);
        recentSeverityState = getRuntimeContext().getMapState(recentSeverityDescriptor);
    }

    @Override
    public void processElement(
        PatternEvent pattern,
        Context ctx,
        Collector<PatternEvent> out) throws Exception {

        long now = System.currentTimeMillis();
        String patternKey = computePatternKey(pattern);

        Long lastFiredTime = recentPatternsState.get(patternKey);
        String lastSeverity = recentSeverityState.get(patternKey);

        if (lastFiredTime != null && (now - lastFiredTime) < DEDUP_WINDOW_MS) {
            // Within dedup window — check for severity escalation
            if (lastSeverity != null && isSeverityEscalation(lastSeverity, pattern.getSeverity())) {
                // ESCALATION: emit immediately even within window
                pattern.addTag("SEVERITY_ESCALATION");
                out.collect(pattern);
                lastPatternState.update(pattern);
                recentPatternsState.put(patternKey, now);
                recentSeverityState.put(patternKey, pattern.getSeverity());
            } else {
                // Same or lower severity — merge
                PatternEvent lastPattern = lastPatternState.value();
                if (lastPattern != null && shouldMerge(lastPattern, pattern)) {
                    PatternEvent mergedPattern = mergePatterns(lastPattern, pattern);
                    out.collect(mergedPattern);
                    lastPatternState.update(mergedPattern);
                    recentPatternsState.put(patternKey, now);
                } else {
                    out.collect(pattern);
                    lastPatternState.update(pattern);
                    recentPatternsState.put(patternKey, now);
                    recentSeverityState.put(patternKey, pattern.getSeverity());
                }
            }
        } else {
            // New pattern outside window — emit
            out.collect(pattern);
            lastPatternState.update(pattern);
            recentPatternsState.put(patternKey, now);
            recentSeverityState.put(patternKey, pattern.getSeverity());
        }

        ctx.timerService().registerProcessingTimeTimer(now + DEDUP_WINDOW_MS);
    }

    // ═══ Public static helpers for testability ═══════════════

    /**
     * Compute dedup key from pattern — type only, NOT severity.
     * This ensures escalations within the same type are detected.
     */
    public static String computePatternKey(PatternEvent pattern) {
        return pattern.getPatternType();
    }

    /**
     * Returns true if newSeverity is strictly higher than oldSeverity.
     */
    public static boolean isSeverityEscalation(String oldSeverity, String newSeverity) {
        return severityIndex(newSeverity) > severityIndex(oldSeverity);
    }

    /**
     * Returns numeric index for severity comparison. Higher = more severe.
     */
    public static int severityIndex(String severity) {
        if (severity == null) return 0;
        int idx = SEVERITY_ORDER.indexOf(severity.toUpperCase());
        return idx >= 0 ? idx + 1 : 0;
    }

    // ═══ Private helpers (unchanged from v1) ═════════════════

    private boolean shouldMerge(PatternEvent existing, PatternEvent newPattern) {
        return existing.getPatternType().equals(newPattern.getPatternType());
    }

    private PatternEvent mergePatterns(PatternEvent existing, PatternEvent newPattern) {
        PatternEvent merged = new PatternEvent();
        merged.setId(existing.getId());
        merged.setPatientId(existing.getPatientId());
        merged.setEncounterId(existing.getEncounterId());
        merged.setPatternType(existing.getPatternType());
        merged.setCorrelationId(existing.getCorrelationId());

        merged.setSeverity(getHighestSeverity(existing.getSeverity(), newPattern.getSeverity()));

        double combinedConfidence = Math.min(1.0,
            existing.getConfidence() * 0.6 + newPattern.getConfidence() * 0.4);
        merged.setConfidence(combinedConfidence);

        merged.setDetectionTime(Math.min(existing.getDetectionTime(), newPattern.getDetectionTime()));

        merged.setPatternStartTime(Math.min(
            existing.getPatternStartTime() != null ? existing.getPatternStartTime() : Long.MAX_VALUE,
            newPattern.getPatternStartTime() != null ? newPattern.getPatternStartTime() : Long.MAX_VALUE));

        merged.setPatternEndTime(Math.max(
            existing.getPatternEndTime() != null ? existing.getPatternEndTime() : Long.MIN_VALUE,
            newPattern.getPatternEndTime() != null ? newPattern.getPatternEndTime() : Long.MIN_VALUE));

        Set<String> allInvolvedEvents = new HashSet<>();
        if (existing.getInvolvedEvents() != null) allInvolvedEvents.addAll(existing.getInvolvedEvents());
        if (newPattern.getInvolvedEvents() != null) allInvolvedEvents.addAll(newPattern.getInvolvedEvents());
        merged.setInvolvedEvents(new ArrayList<>(allInvolvedEvents));

        Set<String> allActions = new LinkedHashSet<>();
        if (existing.getRecommendedActions() != null) allActions.addAll(existing.getRecommendedActions());
        if (newPattern.getRecommendedActions() != null) allActions.addAll(newPattern.getRecommendedActions());
        merged.setRecommendedActions(new ArrayList<>(allActions));

        merged.setClinicalContext(existing.getClinicalContext());

        Map<String, Object> mergedDetails = new HashMap<>();
        if (existing.getPatternDetails() != null) mergedDetails.putAll(existing.getPatternDetails());
        if (newPattern.getPatternDetails() != null) mergedDetails.putAll(newPattern.getPatternDetails());
        mergedDetails.put("mergedSources", Arrays.asList(
            getSourceFromMetadata(existing), getSourceFromMetadata(newPattern)));
        mergedDetails.put("multiSourceConfirmation", true);
        merged.setPatternDetails(mergedDetails);

        PatternEvent.PatternMetadata mergedMetadata = new PatternEvent.PatternMetadata();
        mergedMetadata.setAlgorithm("MULTI_SOURCE_MERGED");
        mergedMetadata.setVersion("1.0.0");
        Map<String, Object> params = new HashMap<>();
        params.put("originalSource", getSourceFromMetadata(existing));
        params.put("confirmingSource", getSourceFromMetadata(newPattern));
        mergedMetadata.setAlgorithmParameters(params);
        double avgProcessingTime = (existing.getPatternMetadata().getProcessingTime() +
            newPattern.getPatternMetadata().getProcessingTime()) / 2.0;
        mergedMetadata.setProcessingTime(avgProcessingTime);
        mergedMetadata.setQualityScore("HIGH");
        merged.setPatternMetadata(mergedMetadata);

        Set<String> allTags = new HashSet<>();
        if (existing.getTags() != null) allTags.addAll(existing.getTags());
        if (newPattern.getTags() != null) allTags.addAll(newPattern.getTags());
        allTags.add("MULTI_SOURCE_CONFIRMED");
        merged.setTags(allTags);

        return merged;
    }

    private String getHighestSeverity(String sev1, String sev2) {
        int idx1 = severityIndex(sev1);
        int idx2 = severityIndex(sev2);
        return idx1 >= idx2 ? sev1 : sev2;
    }

    private String getSourceFromMetadata(PatternEvent pattern) {
        if (pattern.getPatternMetadata() != null && pattern.getPatternMetadata().getAlgorithm() != null) {
            return pattern.getPatternMetadata().getAlgorithm();
        }
        return "UNKNOWN_SOURCE";
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx, Collector<PatternEvent> out) throws Exception {
        Iterator<Map.Entry<String, Long>> iterator = recentPatternsState.iterator();
        while (iterator.hasNext()) {
            Map.Entry<String, Long> entry = iterator.next();
            if (timestamp - entry.getValue() > DEDUP_WINDOW_MS) {
                iterator.remove();
            }
        }
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4DeduplicationTest -q 2>&1 | tail -10`
Expected: 3 tests PASS

- [ ] **Step 5: Run all existing tests to verify no regressions**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -q 2>&1 | tail -15`
Expected: All tests PASS (the dedup function's signature is unchanged — only internals changed)

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/functions/PatternDeduplicationFunction.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4DeduplicationTest.java
git commit -m "fix(module4): dedup severity escalation passthrough + public helpers for testability"
```

---

### Task 6: Test PatternClassificationFunction Side-Output Routing

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4ClassificationTest.java`
- Read: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java:1527-1553` (PatternClassificationFunction)

**Context:** `PatternClassificationFunction` is a `MapFunction` that routes PatternEvents to side outputs based on pattern type. Since it's a map function (not a process function), it can't actually emit side outputs — it just adds metadata/tags for classification. The actual side-output routing happens in the `classifiedPatterns.getSideOutput()` calls. We test the classification logic (the map function) to ensure it correctly classifies pattern types.

- [ ] **Step 1: Write the tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 PatternClassificationFunction.
 * Verifies pattern type classification and metadata enrichment.
 */
public class Module4ClassificationTest {

    @Test
    void classification_deterioration_enrichedCorrectly() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-001", "HIGH", 0.85);

        PatternEvent result = fn.map(input);

        assertNotNull(result);
        assertEquals("CLINICAL_DETERIORATION", result.getPatternType());
        assertEquals("CLASS-001", result.getPatientId());
    }

    @Test
    void classification_preservesAllFields() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-002", "CRITICAL", 0.95);
        input.addTag("CEP_DETECTED");

        PatternEvent result = fn.map(input);

        assertEquals("CRITICAL", result.getSeverity());
        assertEquals(0.95, result.getConfidence(), 0.01);
        assertTrue(result.getTags().contains("CEP_DETECTED"));
    }
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4ClassificationTest -q 2>&1 | tail -10`
Expected: 2 tests PASS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4ClassificationTest.java
git commit -m "test(module4): add PatternClassificationFunction tests"
```

---

### Task 7: Test Window Functions (Trend, Anomaly, Protocol)

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4WindowFunctionTest.java`
- Read: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java:1346-1526` (window functions)

**Context:** Window functions operate on collections of events within a time window. `TrendAnalysisWindowFunction` (WindowFunction → `apply()`) uses linear regression on clinical significance over time. `AnomalyDetectionWindowFunction` (**ProcessWindowFunction** → `process()` — different signature!) computes mean/stddev for anomaly detection. `ProtocolMonitoringWindowFunction` (WindowFunction → `apply()`) monitors guideline compliance. We test them by calling `apply`/`process` directly with mock collectors — no Flink runtime needed. Note: if `AnomalyDetectionWindowFunction.process()` accesses `Context` methods (e.g., `context.window()`), we'll need to pass a real or mocked Context instead of null.

- [ ] **Step 1: Write the tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 window functions.
 * Uses direct invocation with mock collectors — no Flink runtime needed.
 */
public class Module4WindowFunctionTest {

    // ── TrendAnalysisWindowFunction ───────────────────────────

    @Test
    void trendAnalysis_deteriorating_emitsPattern() {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-001";

        // Build 5 events with increasing clinical significance (deteriorating trend)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3600_000;
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000)); // 10min apart
            se.getSemanticAnnotations().put("clinical_significance", 0.2 + (i * 0.15));
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        Collector<PatternEvent> collector = new ListCollector<>(collected);

        fn.apply(patientId, window, events, collector);

        assertFalse(collected.isEmpty(), "Deteriorating trend should emit a pattern event");
        PatternEvent result = collected.get(0);
        assertEquals("TREND_ANALYSIS", result.getPatternType());
        assertEquals(patientId, result.getPatientId());

        double slope = (double) result.getPatternDetails().get("trend_slope");
        assertTrue(slope > 0, "Positive slope indicates deterioration, got " + slope);
    }

    @Test
    void trendAnalysis_stable_emitsNothing() {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-002";

        // Build 4 events with similar clinical significance (stable)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3600_000;
        for (int i = 0; i < 4; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000));
            se.getSemanticAnnotations().put("clinical_significance", 0.35 + (i * 0.01)); // Negligible change
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "Stable trend should NOT emit a pattern event");
    }

    @Test
    void trendAnalysis_tooFewEvents_emitsNothing() {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-003";

        // Only 2 events — below the 3-point minimum
        List<SemanticEvent> events = new ArrayList<>();
        events.add(Module4TestBuilder.baselineVitalEvent(patientId));
        events.add(Module4TestBuilder.warningVitalEvent(patientId));

        TimeWindow window = new TimeWindow(0, 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "< 3 events should not produce trend analysis");
    }

    // ── AnomalyDetectionWindowFunction (ProcessWindowFunction — uses process(), not apply()) ──

    @Test
    void anomalyDetection_highStdDev_emitsAnomaly() throws Exception {
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();
        String patientId = "ANOM-001";

        // Build events with one outlier (significance spike)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        double[] significances = {0.2, 0.25, 0.22, 0.9, 0.23}; // 0.9 is the anomaly
        for (int i = 0; i < significances.length; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 300_000)); // 5min apart
            se.getSemanticAnnotations().put("clinical_significance", significances[i]);
            events.add(se);
        }

        // Note: AnomalyDetectionWindowFunction extends ProcessWindowFunction,
        // so we call process() with a null Context (it doesn't use it for logic).
        // If it does use Context, we'll need a mock — check at compile time.
        List<PatternEvent> collected = new ArrayList<>();
        fn.process(patientId, null, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Anomalous spike should emit an anomaly pattern");
        PatternEvent result = collected.get(0);
        assertEquals("ANOMALY_DETECTION", result.getPatternType());
        assertEquals(patientId, result.getPatientId());
    }

    @Test
    void anomalyDetection_uniform_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();
        String patientId = "ANOM-002";

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 300_000));
            se.getSemanticAnnotations().put("clinical_significance", 0.3); // All same
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process(patientId, null, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "Uniform readings should NOT trigger anomaly");
    }

    // ── ProtocolMonitoringWindowFunction ─────────────────────

    @Test
    void protocolMonitoring_noAssessment_emitsComplianceAlert() {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-001";

        // Admission event but no assessment within window → non-compliance
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 7200_000;
        events.add(Module4TestBuilder.admissionEvent(patientId));

        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Missing assessment should emit compliance alert");
        PatternEvent result = collected.get(0);
        assertEquals("PROTOCOL_MONITORING", result.getPatternType());
    }

    // ── Helper: ListCollector ─────────────────────────────────

    /**
     * Simple Collector implementation that stores results in a list.
     */
    private static class ListCollector<T> implements Collector<T> {
        private final List<T> list;
        ListCollector(List<T> list) { this.list = list; }
        @Override public void collect(T record) { list.add(record); }
        @Override public void close() {}
    }
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module4WindowFunctionTest -q 2>&1 | tail -10`
Expected: 6 tests PASS (3 trend + 2 anomaly + 1 protocol)

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module4WindowFunctionTest.java
git commit -m "test(module4): add window function tests for trend, anomaly, and protocol"
```

---

### Task 8: Clean Production Debug Logs

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java`

**Context:** The operator has numerous `LOG.info("🔍 CEP BASELINE CHECK ...")`, `LOG.info("DEBUG - Patient ...")`, and emoji-laden log lines that should be `LOG.debug(...)` in production. The `convertCDSEventToSemanticEvent` method alone has 6 debug log statements at INFO level. These will create massive log volumes in production at even moderate throughput.

- [ ] **Step 1: Downgrade debug logs to LOG.debug**

In `Module4_PatternDetection.java`, change the following patterns:

1. Lines ~137-139 (`🎯 CEP INPUT`): `LOG.info` → `LOG.debug`
2. Lines ~936-938 (`🔍 CEP BASELINE CHECK`): `LOG.info` → `LOG.debug`
3. Lines ~948-950 (`🔍 CEP WARNING CHECK`): `LOG.info` → `LOG.debug`
4. Lines ~961-963 (`🔍 CEP CRITICAL CHECK`): `LOG.info` → `LOG.debug`
5. Lines ~689-690 (`🔍 DEBUG - Processing patient`): `LOG.info` → `LOG.debug`
6. Lines ~709-712 (`✅ CEP DATA`): `LOG.info` → `LOG.debug`
7. Lines ~714-715 (`⚠️ MISSING DATA`): Keep as `LOG.warn` (this is a legitimate warning)
8. Lines ~826-835 (`DEBUG - Patient`): `LOG.info` → `LOG.debug`
9. Lines ~399-406 (`✅ COMPREHENSIVE IMMEDIATE PATTERN`): `LOG.info` → `LOG.debug`

Keep these at INFO level (they log once at startup, not per-event):
- Line ~91 (`Starting Module 4: Pattern Detection`)
- Line ~109 (`Creating pattern detection pipeline`)
- Line ~586 (`Pattern detection pipeline created successfully`)

- [ ] **Step 2: Verify compile succeeds**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Run all tests to verify no regressions**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -q 2>&1 | tail -15`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java
git commit -m "chore(module4): downgrade per-event debug logs from INFO to DEBUG"
```

---

### Task 9: Full Test Suite Validation

**Files:**
- No new files — validation only

**Context:** Final validation that all new tests pass alongside existing tests, and the total test count is correct.

- [ ] **Step 1: Run the complete Module 4 test suite**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -q 2>&1 | tail -20`
Expected: All tests PASS. Expected test count increase: +33 new tests (12 clinical scoring + 7 CEP select + 3 mapper + 3 dedup + 2 classification + 6 window)

- [ ] **Step 2: Verify test file inventory**

New test files created in this plan:
1. `Module4TestBuilder.java` — test data factory
2. `Module4CDSConversionTest.java` — 12 tests for clinical significance + risk level
3. `Module4CEPSelectFunctionTest.java` — 7 tests for CEP select functions
4. `Module4AlertMapperTest.java` — 3 tests for analytics mappers
5. `Module4DeduplicationTest.java` — 3 tests for dedup improvements
6. `Module4ClassificationTest.java` — 2 tests for classification
7. `Module4WindowFunctionTest.java` — 6 tests for window functions (trend, anomaly, protocol)

Total: **33 new tests** + existing 14 (9 ML integration + 5 BP pattern) = **47 Module 4 tests**

- [ ] **Step 3: Final commit with all passing**

```bash
git add -A backend/shared-infrastructure/flink-processing/src/test/
git commit -m "test(module4): complete gap closure — 33 new tests covering CEP, dedup, scoring, windows"
```

---

## Supplementary Tasks (added 2026-03-30 — live pipeline data review)

These tasks were added after reviewing live Flink pipeline output against the Module 4 implementation.
They address schema mismatches discovered in production data and add integration-level test coverage
for the CDS→Semantic conversion path and clinical deterioration trajectories.

### Supplementary Task 0: Production Key Format Fix + Real CDS Mapping Tests

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module4SemanticConverter.java` (vital key fix)
- Modify: `src/test/java/com/cardiofit/flink/builders/Module4TestBuilder.java` (update keys to production format)
- Create: `src/test/java/com/cardiofit/flink/operators/Module4RealCDSMappingTest.java` (7 tests)

**Context:** Production `PatientContextAggregator` stores vitals as `systolicbloodpressure`/`diastolicbloodpressure` but `normalizeVitalSigns()` only mapped `systolicbp`/`diastolicbp`. Blood pressure was silently dropped during Module 4 conversion. Also, production data includes `age`/`gender` in `latestVitals` — demographics must be excluded from vital sign normalization.

**Changes applied:**
- [x] `normalizeVitalSigns()` now checks `systolicbloodpressure` first, falls back to `systolicbp`
- [x] Demographics (age, gender) explicitly excluded from vital mapping
- [x] `Module4TestBuilder` vitals keys updated from `systolicbp` → `systolicbloodpressure`
- [x] 7 new tests: full conversion path (low/moderate/high risk), demographic exclusion, zero-score baseline, risk indicator extraction, lab triple-keying

### Supplementary Task 2 Additions: CEP Boundary Threshold Tests

**Files:**
- Modify: `src/test/java/com/cardiofit/flink/operators/Module4CDSConversionTest.java` (+5 tests)

**Context:** CEP patterns use significance >= 0.6 for "warning" and >= 0.8 for "critical" match. Tests verify exact NEWS2/qSOFA/acuity combinations that sit on these boundaries.

**Changes applied:**
- [x] `significance_boundaryAtWarningThreshold` — exact 0.6 boundary and below
- [x] `significance_boundaryAtCriticalThreshold` — exact 0.8 boundary and below
- [x] `significance_acuityAlone_neverCrossesCritical` — max acuity only contributes 0.20
- [x] `riskLevel_realProductionValues_allLow` — production baseline patients
- [x] `significance_zeroScores_zeroAcuity_producesZeroSignificance` — zero input regression

### Supplementary Task 9b: Deterioration Integration Tests

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module4DeteriorationIntegrationTest.java` (5 tests)

**Context:** Simulates a patient deteriorating through 3 timepoints (T0: baseline, T1: early deterioration, T2: severe) and verifies that clinical significance and risk level cross CEP thresholds at each stage. Also validates BP is correctly mapped at every timepoint (regression guard for the key format fix).

**Changes applied:**
- [x] T0 baseline: sig < 0.3, risk = "low"
- [x] T1 early deterioration: sig crosses 0.6 (warning), risk = "moderate"
- [x] T2 severe: sig crosses 0.8 (critical), risk = "high"
- [x] Monotonic escalation: sig(T0) < sig(T1) < sig(T2)
- [x] BP mapped at all timepoints with dropping SBP

### Supplementary Task 10: Window Function Coverage Closure

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module4WindowFunctionSupplementaryTest.java` (12 tests)

**Context:** Original 7 window tests covered happy paths. Supplementary tests close gaps in edge cases, boundary conditions, and behavioral correctness for all 3 window functions.

**Key corrections from proposed tests (review findings):**
1. `AnomalyDetectionWindowFunction` extends `ProcessWindowFunction` → uses `process()`, not `apply()`. Context parameter unused (null safe).
2. Anomaly minimum is 5 events (`if (events.size() < 5) return`). Tests below 5 verify the guard, not statistical reasoning.
3. Outlier self-contamination: with few events, spikes inflate their own mean/stddev, pushing threshold above the spike itself. Tests need sufficient anchoring data (6+ low values for single spike, 10+ for multi-spike).
4. `TrendAnalysisWindowFunction` uses index-based regression (`x = i`), NOT timestamp-based. Same-timestamp division-by-zero is impossible. Slope reflects significance change per event index, not temporal urgency.
5. `ProtocolMonitoringWindowFunction` counts `guidelineRecommendations.size()` — does NOT check admission→assessment→intervention sequences.
6. Improving (DECREASING) trends ARE emitted as LOW severity — only STABLE is suppressed.

**TrendAnalysis (4 new tests):**
- [x] Improving trend emits DECREASING direction with LOW severity (not suppressed)
- [x] Single event (< 3 minimum) emits nothing
- [x] Steep vs gradual deterioration: slope 0.2 > slope 0.05, steep is INCREASING
- [x] Exactly 3 events (minimum boundary) with clear deterioration emits INCREASING

**AnomalyDetection (4 new tests):**
- [x] 4 events (below 5 minimum) returns early — no emit even with spike present
- [x] 6 events with spike: 5 low anchors + 0.90 outlier exceeds mean+2σ threshold
- [x] Multiple spikes: 12 events (10 anchors + 2 spikes at 0.88, 0.92), anomaly_count=2
- [x] Gradual linear increase: max value stays below threshold — not flagged (trend, not anomaly)

**ProtocolMonitoring (4 new tests):**
- [x] Multiple events with recommendations: counts total across all events (2+1=3)
- [x] Mixed events: only events with non-null recommendations counted
- [x] Empty window: no throw, no emit
- [x] Null guidelineRecommendations: no NPE, count=0, no emit

**Architectural notes for follow-up (not blockers):**
- IEEE 754: significance at exact critical boundary (NEWS2=7, qSOFA=2, acuity=5.0) = 0.7999999999999999. Production CEP `>= 0.8` would miss this. Options: round to 4 decimals before CEP, or use 0.799 threshold.
- Index-based regression: identical significance changes over different time spans produce identical slopes. Temporal urgency not captured. Consider normalized-timestamp regression if deterioration rate matters clinically.
- Anomaly self-contamination: with small windows (5-6 events), single spikes may evade 2σ detection. Consider robust statistics (MAD) or minimum window size > 8.

**Updated totals:** +29 supplementary tests (7 + 5 + 5 + 12) = **95 Module 4 tests total**
