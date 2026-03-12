# Module 4: Clinical Pattern Engine - Enhancement Implementation Plan

## Document Overview

**Purpose**: Detailed implementation plan for enhancing Module 4 Pattern Detection with advanced clinical patterns, windowed analytics, and configuration management

**Timeline**: 8-10 hours total development time
**Priority**: High - Critical clinical decision support enhancements
**Dependencies**: Module 3 (Semantic Mesh) must be operational

---

## Executive Summary

### Current State
Module 4 exists with basic CEP pattern detection and windowed analytics:
- ✅ Basic deterioration patterns (clinical significance-based)
- ✅ Medication adherence tracking
- ✅ Vital signs trend detection
- ✅ Pathway compliance monitoring
- ✅ AKI pattern detection using RiskIndicators
- ✅ Basic anomaly detection and trend analysis

### Gaps Identified (from Implementation Guide)
- ❌ Advanced clinical patterns (sepsis early warning, rapid deterioration, drug-lab monitoring)
- ❌ MEWS (Modified Early Warning Score) calculation
- ❌ Lab trend analysis with clinical interpretations (creatinine, glucose)
- ❌ Vital sign variability detection with coefficient of variation
- ❌ Configuration externalization (hardcoded Kafka topics)
- ❌ Comprehensive clinical recommendations and interpretations

### Enhancement Goals
1. **Clinical Completeness**: Implement all patterns from official implementation guide
2. **Configuration Management**: Externalize all hardcoded values to environment variables
3. **Clinical Intelligence**: Add advanced scoring (MEWS), trend analysis, and variability detection
4. **Operational Readiness**: Production-grade error handling, monitoring, documentation

---

## Architecture Overview

### Current Architecture
```
┌─────────────────────────────────────────────────────────────┐
│              MODULE 4: PATTERN DETECTION (CURRENT)           │
└─────────────────────────────────────────────────────────────┘

Input Streams:
├─ SemanticEvent (from Module 3 Semantic Mesh)
└─ EnrichedEvent (from Module 2 Clinical Patterns)

CEP Patterns (5):
├─ Clinical Deterioration (significance-based)
├─ Medication Adherence
├─ Vital Signs Trend
├─ Pathway Compliance
└─ AKI Detection (KDIGO criteria)

Windowed Analytics (3):
├─ Trend Analysis (sliding windows)
├─ Anomaly Detection (tumbling windows)
└─ Protocol Monitoring

Output:
└─ PatternEvent → Multiple Kafka topics (hardcoded)
```

### Enhanced Architecture
```
┌─────────────────────────────────────────────────────────────┐
│              MODULE 4: PATTERN DETECTION (ENHANCED)          │
└─────────────────────────────────────────────────────────────┘

Input Streams (Configurable):
├─ SemanticEvent (semantic-mesh-updates.v1)
└─ EnrichedEvent (clinical-patterns.v1)

CEP Patterns (9):
├─ Clinical Deterioration (significance-based)
├─ Medication Adherence
├─ Vital Signs Trend
├─ Pathway Compliance
├─ AKI Detection (KDIGO criteria)
├─ ✨ Sepsis Early Warning (qSOFA-based)
├─ ✨ Rapid Clinical Deterioration (multi-vital)
├─ ✨ Drug-Lab Interaction Monitoring (ACE-I, Warfarin, etc.)
└─ ✨ Sepsis Pathway Compliance (bundle tracking)

Windowed Analytics (7):
├─ Trend Analysis (sliding windows)
├─ Anomaly Detection (tumbling windows)
├─ Protocol Monitoring
├─ ✨ MEWS Calculation (4-hour tumbling)
├─ ✨ Creatinine Trend Analysis (48-hour sliding)
├─ ✨ Glucose Trend Analysis (24-hour sliding)
└─ ✨ Vital Variability Detection (4-hour sliding)

Output (Configurable):
├─ pattern-events.v1 (all patterns)
├─ alert-management.v1 (deterioration)
├─ pathway-adherence-events.v1 (pathway compliance)
├─ safety-events.v1 (anomalies)
├─ clinical-reasoning-events.v1 (trends)
└─ ✨ critical-alerts.v1 (MEWS, severe deterioration)
```

---

## Phase 1: Configuration Externalization

### Objective
Make all Kafka configuration values externally configurable via environment variables with sensible fallback defaults (following Module 3 pattern).

### Current Hardcoded Values
```java
// In createSemanticEventSource()
.setTopics(KafkaTopics.SEMANTIC_MESH_UPDATES.getTopicName())  // Uses enum

// In createEnrichedEventSource()
.setTopics(KafkaTopics.CLINICAL_PATTERNS.getTopicName())  // Uses enum

// In createPatternEventsSink()
.setTopic(KafkaTopics.CLINICAL_PATTERNS.getTopicName())  // Hardcoded in enum

// In createDeteriorationPatternSink()
.setTopic(KafkaTopics.ALERT_MANAGEMENT.getTopicName())  // Hardcoded in enum

// In getBootstrapServers()
return KafkaConfigLoader.isRunningInDocker()
    ? "kafka1:29092,kafka2:29093,kafka3:29094"  // Hardcoded
    : "localhost:9092,localhost:9093,localhost:9094";  // Hardcoded
```

### Implementation Tasks

**Task 1.1: Add Helper Method**
```java
/**
 * Get Kafka topic name from environment variable with fallback default
 */
private static String getTopicName(String envVar, String defaultTopic) {
    String topic = System.getenv(envVar);
    return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
}
```

**Task 1.2: Update Kafka Source Configurations**
```java
// Replace
.setTopics(KafkaTopics.SEMANTIC_MESH_UPDATES.getTopicName())

// With
.setTopics(getTopicName("MODULE4_SEMANTIC_INPUT_TOPIC", "semantic-mesh-updates.v1"))

// Replace
.setTopics(KafkaTopics.CLINICAL_PATTERNS.getTopicName())

// With
.setTopics(getTopicName("MODULE4_ENRICHED_INPUT_TOPIC", "clinical-patterns.v1"))
```

**Task 1.3: Update Kafka Sink Configurations**
```java
// Main pattern events sink
.setTopic(getTopicName("MODULE4_PATTERN_EVENTS_TOPIC", "pattern-events.v1"))

// Deterioration patterns sink
.setTopic(getTopicName("MODULE4_DETERIORATION_TOPIC", "alert-management.v1"))

// Pathway adherence sink
.setTopic(getTopicName("MODULE4_PATHWAY_ADHERENCE_TOPIC", "pathway-adherence-events.v1"))

// Anomaly detection sink
.setTopic(getTopicName("MODULE4_ANOMALY_DETECTION_TOPIC", "safety-events.v1"))

// Trend analysis sink
.setTopic(getTopicName("MODULE4_TREND_ANALYSIS_TOPIC", "clinical-reasoning-events.v1"))
```

**Task 1.4: Update Bootstrap Servers**
```java
private static String getBootstrapServers() {
    String kafkaServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
    return (kafkaServers != null && !kafkaServers.isEmpty())
        ? kafkaServers
        : "localhost:9092";  // Simple default, no Docker detection
}
```

### Environment Variables Added
```bash
# Module 4 Input Topics
MODULE4_SEMANTIC_INPUT_TOPIC=semantic-mesh-updates.v1
MODULE4_ENRICHED_INPUT_TOPIC=clinical-patterns.v1

# Module 4 Output Topics
MODULE4_PATTERN_EVENTS_TOPIC=pattern-events.v1
MODULE4_DETERIORATION_TOPIC=alert-management.v1
MODULE4_PATHWAY_ADHERENCE_TOPIC=pathway-adherence-events.v1
MODULE4_ANOMALY_DETECTION_TOPIC=safety-events.v1
MODULE4_TREND_ANALYSIS_TOPIC=clinical-reasoning-events.v1

# Kafka Connection
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
```

### Deliverables
- ✅ `getTopicName()` helper method
- ✅ All Kafka topics configurable
- ✅ Bootstrap servers configurable
- ✅ Backward compatible (defaults match current values)

### Testing
```bash
# Test with default configuration
mvn clean package -DskipTests
curl -X POST 'http://localhost:8081/jars/<jar-id>/run' \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module4_PatternDetection"}'

# Test with custom configuration
export MODULE4_PATTERN_EVENTS_TOPIC=dev-pattern-events
export MODULE4_DETERIORATION_TOPIC=dev-alerts
# Deploy and verify custom topics used
```

---

## Phase 2: Advanced CEP Patterns

### Objective
Implement 4 additional clinically-validated CEP patterns from the implementation guide.

---

### Pattern 2.1: Sepsis Early Warning Pattern

**Clinical Rationale**: Early sepsis detection using qSOFA criteria and laboratory markers can reduce mortality by 50% through timely antibiotic administration and fluid resuscitation.

**Pattern Logic**:
```
baseline → early_warning → deterioration
Within 6-hour window
```

**Conditions**:
1. **Baseline**: Normal or slightly elevated vitals
   - Heart rate: 60-110 bpm
   - Systolic BP: ≥90 mmHg
   - Temperature: 36-38°C

2. **Early Warning** (≥2 qSOFA criteria OR lab abnormalities):
   - Respiratory rate ≥22/min
   - Systolic BP ≤100 mmHg
   - Heart rate >90 bpm
   - Temperature >38.3°C or <36°C
   - Lactate >2.0 mmol/L
   - WBC >12,000 or <4,000

3. **Deterioration** (Severe sepsis indicators):
   - Systolic BP <90 mmHg (despite fluids)
   - Lactate >4.0 mmol/L
   - Organ dysfunction markers

**Implementation File**: `ClinicalPatterns.java` (already exists as `detectSepsisPattern()`)

**Integration Task**: Wire into Module4_PatternDetection pipeline
```java
// Add to createPatternDetectionPipeline()
PatternStream<SemanticEvent> sepsisPatterns =
    ClinicalPatterns.detectSepsisPattern(keyedSemanticEvents);

DataStream<PatternEvent> sepsisEvents = sepsisPatterns
    .select(new SepsisPatternSelectFunction())
    .uid("Sepsis Early Warning Events");
```

**Alert Output**:
```json
{
  "patternType": "SEPSIS_EARLY_WARNING",
  "severity": "CRITICAL",
  "confidence": 0.88,
  "qsofaScore": 2,
  "labAbnormalities": ["elevated_lactate", "leukocytosis"],
  "recommendedActions": [
    "STAT_BLOOD_CULTURES",
    "BROAD_SPECTRUM_ANTIBIOTICS_WITHIN_1H",
    "FLUID_RESUSCITATION_30ML_KG",
    "LACTATE_CLEARANCE_MONITORING",
    "ICU_CONSULTATION"
  ]
}
```

---

### Pattern 2.2: Rapid Clinical Deterioration Pattern

**Clinical Rationale**: Multi-organ deterioration patterns (cardiac + respiratory) require immediate intervention to prevent cardiac arrest or respiratory failure.

**Pattern Logic**:
```
hr_baseline → hr_elevated (+20 bpm) → rr_elevated (+30%) → o2sat_decreased (-5%)
Within 1-hour window
```

**Conditions**:
1. **HR Baseline**: Any heart rate measurement
2. **HR Elevated**: Heart rate increased by >20 bpm from baseline
3. **RR Elevated**: Respiratory rate >24/min (or 30% increase from baseline)
4. **O2 Sat Decreased**: SpO2 <92% (or 5% drop from baseline)

**Implementation** (New Pattern):
```java
public static Pattern<SemanticEvent, ?> detectRapidDeteriorationPattern() {
    return Pattern.<SemanticEvent>begin("hr_baseline")
        .where(event -> hasVitalSign(event, "heart_rate"))
        .followedBy("hr_elevated")
        .where((current, ctx) -> {
            SemanticEvent baseline = getFirst(ctx, "hr_baseline");
            double hrIncrease = getVitalValue(current, "heart_rate") -
                              getVitalValue(baseline, "heart_rate");
            return hrIncrease > 20;
        })
        .followedBy("rr_elevated")
        .where(event -> getVitalValue(event, "respiratory_rate") > 24)
        .followedBy("o2sat_decreased")
        .where(event -> getVitalValue(event, "oxygen_saturation") < 92)
        .within(Duration.ofHours(1));
}
```

**Alert Output**:
```json
{
  "patternType": "RAPID_CLINICAL_DETERIORATION",
  "severity": "CRITICAL",
  "confidence": 0.92,
  "deteriorationRate": "RAPID",
  "timespan_minutes": 45,
  "vitalChanges": {
    "heart_rate_increase_bpm": 28,
    "respiratory_rate": 26,
    "oxygen_saturation": 88
  },
  "recommendedActions": [
    "ACTIVATE_RAPID_RESPONSE_TEAM",
    "SUPPLEMENTAL_OXYGEN",
    "CONTINUOUS_MONITORING",
    "ABG_ANALYSIS",
    "ASSESS_AIRWAY_BREATHING_CIRCULATION"
  ]
}
```

---

### Pattern 2.3: Drug-Lab Interaction Monitoring Pattern

**Clinical Rationale**: Many medications require laboratory monitoring to prevent toxicity or organ damage. ACE inhibitors can cause hyperkalemia and renal dysfunction; warfarin requires INR monitoring; metformin requires renal function checks.

**Pattern Logic**:
```
high_risk_medication_started → notFollowedBy(monitoring_labs_ordered)
Within 48-hour window
```

**Monitored Drug Classes**:
| Drug Class | Required Labs | Monitoring Window | Rationale |
|------------|--------------|-------------------|-----------|
| ACE Inhibitors | K+, Creatinine | 48 hours | Hyperkalemia, AKI risk |
| Warfarin | INR/PT | 72 hours | Bleeding risk |
| Metformin | Creatinine, eGFR | 24 hours | Lactic acidosis risk |
| Digoxin | Digoxin level, K+ | 48 hours | Toxicity risk |
| Lithium | Lithium level, Creatinine | 7 days | Narrow therapeutic index |

**Implementation** (New Pattern):
```java
public static Pattern<SemanticEvent, ?> detectDrugLabMonitoringPattern() {
    return Pattern.<SemanticEvent>begin("high_risk_med_started")
        .where(event -> {
            if (!"MEDICATION_ORDERED".equals(event.getEventType())) return false;
            String medicationName = getMedicationName(event);
            return requiresLabMonitoring(medicationName);
        })
        .notFollowedBy("monitoring_labs_ordered")
        .where((event, ctx) -> {
            SemanticEvent medEvent = getFirst(ctx, "high_risk_med_started");
            String medicationName = getMedicationName(medEvent);
            String[] requiredLabs = getRequiredLabs(medicationName);

            if (!"LAB_RESULT".equals(event.getEventType())) return false;
            String labName = getLabName(event);
            return Arrays.asList(requiredLabs).contains(labName);
        })
        .within(Duration.ofHours(48));
}

private static boolean requiresLabMonitoring(String medicationName) {
    return medicationName.contains("ACE") ||
           medicationName.contains("Lisinopril") ||
           medicationName.contains("Enalapril") ||
           medicationName.contains("Warfarin") ||
           medicationName.contains("Metformin") ||
           medicationName.contains("Digoxin") ||
           medicationName.contains("Lithium");
}

private static String[] getRequiredLabs(String medicationName) {
    if (medicationName.contains("ACE") || medicationName.contains("Lisinopril")) {
        return new String[]{"Potassium", "Creatinine"};
    } else if (medicationName.contains("Warfarin")) {
        return new String[]{"INR", "PT"};
    } else if (medicationName.contains("Metformin")) {
        return new String[]{"Creatinine", "eGFR"};
    }
    // ... more mappings
    return new String[]{};
}
```

**Alert Output**:
```json
{
  "patternType": "DRUG_LAB_MONITORING_MISSED",
  "severity": "MODERATE",
  "confidence": 0.85,
  "medicationName": "Lisinopril 10mg",
  "requiredLabs": ["Potassium", "Creatinine"],
  "hoursElapsed": 36,
  "maxWindow": 48,
  "recommendedActions": [
    "ORDER_BASIC_METABOLIC_PANEL",
    "CHECK_POTASSIUM_LEVEL",
    "ASSESS_RENAL_FUNCTION",
    "REVIEW_MEDICATION_SAFETY"
  ]
}
```

---

### Pattern 2.4: Sepsis Pathway Compliance Pattern

**Clinical Rationale**: Sepsis bundle compliance (Surviving Sepsis Campaign guidelines) reduces mortality. Key components must be completed within strict timeframes: blood cultures before antibiotics, broad-spectrum antibiotics within 1 hour, lactate measurement, fluid resuscitation.

**Pattern Logic**:
```
sepsis_diagnosis → blood_cultures_ordered (within 1h) → antibiotics_started (within 1h)
```

**Conditions**:
1. **Sepsis Diagnosis**: ICD-10 code A41.x (sepsis) or qSOFA ≥2
2. **Blood Cultures Ordered**: Lab order within 1 hour
3. **Antibiotics Started**: Broad-spectrum antibiotic administration within 1 hour
4. **Optional**: Lactate measurement, fluid bolus administration

**Implementation** (New Pattern):
```java
public static Pattern<SemanticEvent, ?> detectSepsisPathwayCompliancePattern() {
    return Pattern.<SemanticEvent>begin("sepsis_diagnosis")
        .where(event -> {
            // Check for sepsis diagnosis
            Map<String, Object> clinicalData = event.getClinicalData();
            if (clinicalData.containsKey("diagnosis_codes")) {
                List<String> diagnoses = (List<String>) clinicalData.get("diagnosis_codes");
                return diagnoses.stream().anyMatch(code -> code.startsWith("A41"));
            }
            // Or qSOFA ≥2
            if (clinicalData.containsKey("qsofa_score")) {
                Integer qsofaScore = (Integer) clinicalData.get("qsofa_score");
                return qsofaScore != null && qsofaScore >= 2;
            }
            return false;
        })
        .followedBy("blood_cultures_ordered")
        .where(event -> {
            return "LAB_ORDER".equals(event.getEventType()) &&
                   getLabName(event).contains("Blood Culture");
        })
        .within(Duration.ofHours(1))
        .followedBy("antibiotics_started")
        .where(event -> {
            return "MEDICATION_ADMINISTERED".equals(event.getEventType()) &&
                   isAntibiotic(getMedicationName(event));
        })
        .within(Duration.ofHours(1));
}

private static boolean isAntibiotic(String medicationName) {
    String[] antibiotics = {
        "Ceftriaxone", "Vancomycin", "Piperacillin-Tazobactam",
        "Meropenem", "Cefepime", "Azithromycin", "Levofloxacin"
    };
    return Arrays.stream(antibiotics)
        .anyMatch(ab -> medicationName.contains(ab));
}
```

**Alert Output** (Compliance):
```json
{
  "patternType": "SEPSIS_PATHWAY_COMPLIANCE",
  "severity": "LOW",
  "confidence": 0.9,
  "bundleCompleted": true,
  "timings": {
    "diagnosis_to_cultures_minutes": 35,
    "diagnosis_to_antibiotics_minutes": 52
  },
  "complianceRate": 100,
  "message": "✅ Sepsis bundle completed within 1-hour window"
}
```

**Alert Output** (Non-Compliance):
```json
{
  "patternType": "SEPSIS_PATHWAY_DEVIATION",
  "severity": "HIGH",
  "confidence": 0.85,
  "bundleCompleted": false,
  "missingComponents": ["blood_cultures", "antibiotics"],
  "hoursElapsed": 2.5,
  "recommendedActions": [
    "STAT_BLOOD_CULTURES_NOW",
    "BROAD_SPECTRUM_ANTIBIOTICS_IMMEDIATELY",
    "LACTATE_MEASUREMENT",
    "FLUID_BOLUS_30ML_KG",
    "ESCALATE_TO_ATTENDING"
  ]
}
```

### Phase 2 Deliverables
- ✅ 4 new CEP patterns implemented
- ✅ Pattern select functions for each pattern
- ✅ Integration into Module4_PatternDetection pipeline
- ✅ Clinical recommendations for each pattern
- ✅ Comprehensive alert outputs

---

## Phase 3: Advanced Windowed Analytics

### Objective
Implement 4 advanced windowed analytics features for continuous clinical monitoring and risk scoring.

---

### Feature 3.1: MEWS (Modified Early Warning Score) Calculator

**Clinical Background**:
MEWS is a track-and-trigger system used to identify patients at risk of clinical deterioration. Scores ≥3 require increased monitoring; scores ≥5 require immediate medical review and possible ICU transfer.

**MEWS Scoring Table**:
| Parameter | 3 | 2 | 1 | 0 | 1 | 2 | 3 |
|-----------|---|---|---|---|---|---|---|
| Respiratory Rate | ≤8 | 9-14 | 15-20 | 21-29 | ≥30 | | |
| Heart Rate | ≤40 | 41-50 | 51-100 | 101-110 | 111-129 | ≥130 | |
| Systolic BP | ≤70 | 71-80 | 81-100 | 101-199 | ≥200 | | |
| Temperature (°C) | ≤35.0 | | 35.1-38.4 | ≥38.5 | | | |
| Consciousness (AVPU) | | | Alert | Voice | Pain | Unresponsive | |

**Window Configuration**:
- **Window Type**: Tumbling (non-overlapping)
- **Window Size**: 4 hours
- **Trigger**: Every 4 hours per patient
- **Output**: MEWS score with breakdown and recommendations

**Implementation**:

**File**: `com/cardiofit/flink/analytics/MEWSCalculator.java` (NEW)
```java
package com.cardiofit.flink.analytics;

import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

/**
 * MEWS (Modified Early Warning Score) Calculator
 * Continuous assessment of patient deterioration risk
 *
 * Clinical Reference: Royal College of Physicians (UK)
 * Evidence: Reduces cardiac arrests by 50%, ICU admissions by 25%
 */
public class MEWSCalculator {

    public static DataStream<MEWSAlert> calculateMEWS(
            DataStream<SemanticEvent> vitalStream) {

        return vitalStream
            .filter(event -> hasVitalSigns(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Time.hours(4)))
            .apply(new MEWSCalculationWindowFunction());
    }

    public static class MEWSCalculationWindowFunction
            implements WindowFunction<SemanticEvent, MEWSAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<MEWSAlert> out) {

            // Get most recent vitals within window
            Map<String, Double> latestVitals = extractLatestVitals(events);

            if (latestVitals.size() < 3) {
                return; // Need at least 3 vital signs for valid MEWS
            }

            // Calculate MEWS score
            int mewsScore = 0;
            Map<String, Integer> scoreBreakdown = new HashMap<>();
            List<String> concerningVitals = new ArrayList<>();

            // Respiratory Rate scoring
            if (latestVitals.containsKey("respiratory_rate")) {
                double rr = latestVitals.get("respiratory_rate");
                int rrScore = calculateRRScore(rr);
                mewsScore += rrScore;
                scoreBreakdown.put("Respiratory_Rate", rrScore);

                if (rrScore >= 2) {
                    concerningVitals.add(String.format("RR: %.0f/min (Score: %d)", rr, rrScore));
                }
            }

            // Heart Rate scoring
            if (latestVitals.containsKey("heart_rate")) {
                double hr = latestVitals.get("heart_rate");
                int hrScore = calculateHRScore(hr);
                mewsScore += hrScore;
                scoreBreakdown.put("Heart_Rate", hrScore);

                if (hrScore >= 2) {
                    concerningVitals.add(String.format("HR: %.0f bpm (Score: %d)", hr, hrScore));
                }
            }

            // Systolic Blood Pressure scoring
            if (latestVitals.containsKey("systolic_bp")) {
                double sbp = latestVitals.get("systolic_bp");
                int sbpScore = calculateBPScore(sbp);
                mewsScore += sbpScore;
                scoreBreakdown.put("Blood_Pressure", sbpScore);

                if (sbpScore >= 2) {
                    concerningVitals.add(String.format("SBP: %.0f mmHg (Score: %d)", sbp, sbpScore));
                }
            }

            // Temperature scoring
            if (latestVitals.containsKey("temperature")) {
                double temp = latestVitals.get("temperature");
                int tempScore = calculateTempScore(temp);
                mewsScore += tempScore;
                scoreBreakdown.put("Temperature", tempScore);

                if (tempScore >= 2) {
                    concerningVitals.add(String.format("Temp: %.1f°C (Score: %d)", temp, tempScore));
                }
            }

            // Consciousness Level (from GCS)
            if (latestVitals.containsKey("glasgow_coma_scale")) {
                double gcs = latestVitals.get("glasgow_coma_scale");
                int consciousnessScore = calculateConsciousnessScore(gcs);
                mewsScore += consciousnessScore;
                scoreBreakdown.put("Consciousness", consciousnessScore);

                if (consciousnessScore >= 2) {
                    concerningVitals.add(String.format("GCS: %.0f (Score: %d)", gcs, consciousnessScore));
                }
            }

            // Generate alert if MEWS ≥ 3
            if (mewsScore >= 3) {
                String urgency = determineUrgency(mewsScore);
                String recommendations = generateRecommendations(mewsScore);

                MEWSAlert alert = new MEWSAlert();
                alert.setPatientId(patientId);
                alert.setMewsScore(mewsScore);
                alert.setScoreBreakdown(scoreBreakdown);
                alert.setConcerningVitals(concerningVitals);
                alert.setUrgency(urgency);
                alert.setRecommendations(recommendations);
                alert.setTimestamp(System.currentTimeMillis());
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());

                out.collect(alert);
            }
        }

        // MEWS Scoring Methods
        private int calculateRRScore(double rr) {
            if (rr < 9) return 2;
            if (rr >= 9 && rr <= 14) return 0;
            if (rr >= 15 && rr <= 20) return 1;
            if (rr >= 21 && rr <= 29) return 2;
            return 3; // ≥30
        }

        private int calculateHRScore(double hr) {
            if (hr < 40) return 2;
            if (hr >= 40 && hr <= 50) return 1;
            if (hr >= 51 && hr <= 100) return 0;
            if (hr >= 101 && hr <= 110) return 1;
            if (hr >= 111 && hr <= 129) return 2;
            return 3; // ≥130
        }

        private int calculateBPScore(double sbp) {
            if (sbp < 70) return 3;
            if (sbp >= 70 && sbp <= 80) return 2;
            if (sbp >= 81 && sbp <= 100) return 1;
            if (sbp >= 101 && sbp <= 199) return 0;
            return 2; // ≥200
        }

        private int calculateTempScore(double temp) {
            if (temp < 35.0) return 2;
            if (temp >= 35.0 && temp <= 38.4) return 0;
            return 2; // ≥38.5
        }

        private int calculateConsciousnessScore(double gcs) {
            if (gcs == 15) return 0;                // Alert
            if (gcs >= 13 && gcs <= 14) return 1;   // Voice
            if (gcs >= 9 && gcs <= 12) return 2;    // Pain
            return 3;                                // Unresponsive (<9)
        }

        private String determineUrgency(int mewsScore) {
            if (mewsScore >= 5) {
                return "🔴 CRITICAL: Urgent medical review required within 15 minutes";
            } else if (mewsScore >= 3) {
                return "🟠 HIGH: Increased monitoring - notify physician within 30 minutes";
            } else {
                return "🟡 MODERATE: Enhanced monitoring";
            }
        }

        private String generateRecommendations(int mewsScore) {
            if (mewsScore >= 5) {
                return "1. Call Rapid Response Team IMMEDIATELY\n" +
                       "2. Continuous vital sign monitoring\n" +
                       "3. Consider ICU transfer\n" +
                       "4. Assess airway, breathing, circulation\n" +
                       "5. Obtain arterial blood gas";
            } else if (mewsScore >= 3) {
                return "1. Notify physician within 30 minutes\n" +
                       "2. Increase vital sign frequency to q1h\n" +
                       "3. Review recent labs\n" +
                       "4. Consider step-up level of care";
            } else {
                return "1. Continue current monitoring\n" +
                       "2. Reassess in 4 hours";
            }
        }
    }
}
```

**Data Model**: `com/cardiofit/flink/models/MEWSAlert.java` (NEW)
```java
public class MEWSAlert implements Serializable {
    private String patientId;
    private Integer mewsScore;
    private Map<String, Integer> scoreBreakdown;
    private List<String> concerningVitals;
    private String urgency;
    private String recommendations;
    private Long timestamp;
    private Long windowStart;
    private Long windowEnd;

    // Getters and setters
}
```

**Output Example**:
```json
{
  "patientId": "PAT-12345",
  "mewsScore": 5,
  "scoreBreakdown": {
    "Respiratory_Rate": 2,
    "Heart_Rate": 2,
    "Blood_Pressure": 1,
    "Temperature": 0,
    "Consciousness": 0
  },
  "concerningVitals": [
    "RR: 26/min (Score: 2)",
    "HR: 118 bpm (Score: 2)"
  ],
  "urgency": "🔴 CRITICAL: Urgent medical review required within 15 minutes",
  "recommendations": "1. Call Rapid Response Team IMMEDIATELY\n2. Continuous vital sign monitoring\n3. Consider ICU transfer\n4. Assess airway, breathing, circulation\n5. Obtain arterial blood gas",
  "timestamp": 1738281600000,
  "windowStart": 1738267200000,
  "windowEnd": 1738281600000
}
```

---

### Feature 3.2: Creatinine Trend Analysis (AKI Detection)

**Clinical Background**:
Acute Kidney Injury (AKI) detection using KDIGO criteria: serum creatinine increase ≥0.3 mg/dL within 48 hours OR ≥50% increase from baseline within 7 days. Early detection enables nephroprotective interventions.

**Window Configuration**:
- **Window Type**: Sliding
- **Window Size**: 48 hours
- **Slide Interval**: 1 hour
- **Analysis**: Linear regression trend, percent change from baseline

**Implementation**:

**File**: `com/cardiofit/flink/analytics/LabTrendAnalyzer.java` (NEW)
```java
public class LabTrendAnalyzer {

    public static DataStream<LabTrendAlert> analyzeCreatinineTrends(
            DataStream<SemanticEvent> labStream) {

        return labStream
            .filter(event -> isCreatinineLab(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Time.hours(48), Time.hours(1)))
            .apply(new CreatinineTrendWindowFunction());
    }

    public static class CreatinineTrendWindowFunction
            implements WindowFunction<SemanticEvent, LabTrendAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<LabTrendAlert> out) {

            List<LabValue> creatinineValues = extractCreatinineValues(events);

            if (creatinineValues.size() < 2) {
                return; // Need at least 2 values for trend
            }

            // Sort by timestamp
            creatinineValues.sort(Comparator.comparing(LabValue::getTimestamp));

            // Calculate trend metrics
            double firstValue = creatinineValues.get(0).getValue();
            double lastValue = creatinineValues.get(creatinineValues.size() - 1).getValue();
            double absoluteChange = lastValue - firstValue;
            double percentChange = ((lastValue - firstValue) / firstValue) * 100;

            // Linear regression for trend line
            TrendAnalysis trend = calculateLinearTrend(creatinineValues);

            // KDIGO AKI criteria
            boolean akiStage1 = absoluteChange >= 0.3 && absoluteChange < 1.0;  // Stage 1
            boolean akiStage2 = absoluteChange >= 1.0 && lastValue < (firstValue * 3);  // Stage 2
            boolean akiStage3 = lastValue >= (firstValue * 3) || lastValue >= 4.0;  // Stage 3

            // Check for concerning trends
            if (Math.abs(percentChange) > 25 || Math.abs(trend.getSlope()) > 0.1 ||
                akiStage1 || akiStage2 || akiStage3) {

                String akiStage = determineAKIStage(absoluteChange, firstValue, lastValue);
                String interpretation = interpretCreatinineTrend(
                    percentChange, trend, akiStage, firstValue, lastValue);

                LabTrendAlert alert = new LabTrendAlert();
                alert.setPatientId(patientId);
                alert.setLabName("Creatinine");
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());
                alert.setFirstValue(firstValue);
                alert.setLastValue(lastValue);
                alert.setAbsoluteChange(absoluteChange);
                alert.setPercentChange(percentChange);
                alert.setTrendSlope(trend.getSlope());
                alert.setTrendDirection(trend.getDirection());
                alert.setDataPoints(creatinineValues.size());
                alert.setRSquared(trend.getRSquared());
                alert.setAkiStage(akiStage);
                alert.setInterpretation(interpretation);

                out.collect(alert);
            }
        }

        private String determineAKIStage(double absoluteChange, double baseline, double current) {
            if (current >= (baseline * 3) || current >= 4.0) {
                return "AKI_STAGE_3";
            } else if (current >= (baseline * 2)) {
                return "AKI_STAGE_2";
            } else if (absoluteChange >= 0.3) {
                return "AKI_STAGE_1";
            } else {
                return "NO_AKI";
            }
        }

        private String interpretCreatinineTrend(double percentChange, TrendAnalysis trend,
                                               String akiStage, double baseline, double current) {
            StringBuilder interpretation = new StringBuilder();

            if ("AKI_STAGE_3".equals(akiStage)) {
                interpretation.append(String.format(
                    "🔴 CRITICAL: AKI Stage 3 detected. Creatinine %.2f → %.2f mg/dL (%.1f%% increase). " +
                    "Severe renal dysfunction. IMMEDIATE nephrology consultation required. " +
                    "Consider dialysis. Stop nephrotoxic medications.",
                    baseline, current, percentChange
                ));
            } else if ("AKI_STAGE_2".equals(akiStage)) {
                interpretation.append(String.format(
                    "🔴 CRITICAL: AKI Stage 2 detected. Creatinine %.2f → %.2f mg/dL (%.1f%% increase). " +
                    "Moderate renal dysfunction. Urgent nephrology consultation. " +
                    "Dose-adjust renally-excreted medications. Avoid nephrotoxic agents.",
                    baseline, current, percentChange
                ));
            } else if ("AKI_STAGE_1".equals(akiStage)) {
                interpretation.append(String.format(
                    "⚠️ WARNING: AKI Stage 1 detected. Creatinine %.2f → %.2f mg/dL (%.1f%% increase). " +
                    "Early renal dysfunction. Review medications (NSAIDs, ACE-I, diuretics). " +
                    "Optimize fluid status. Monitor urine output. Consider nephrology consult if worsening.",
                    baseline, current, percentChange
                ));
            } else if (trend.getDirection() == TrendDirection.RAPIDLY_INCREASING) {
                interpretation.append(String.format(
                    "⚠️ WARNING: Creatinine rapidly increasing (%.1f%% over 48h). " +
                    "Pre-renal azotemia vs early AKI. Check volume status, medication review. " +
                    "Trend concerning - close monitoring required.",
                    percentChange
                ));
            } else if (trend.getDirection() == TrendDirection.RAPIDLY_DECREASING) {
                interpretation.append(String.format(
                    "✅ IMPROVING: Creatinine decreasing %.1f%%. " +
                    "Renal function recovering. Continue supportive management. " +
                    "May liberalize medications as appropriate.",
                    Math.abs(percentChange)
                ));
            } else {
                interpretation.append(String.format(
                    "Creatinine trend stable. Baseline: %.2f mg/dL, Current: %.2f mg/dL.",
                    baseline, current
                ));
            }

            return interpretation.toString();
        }
    }
}
```

---

### Feature 3.3: Glucose Trend Analysis (Glycemic Variability)

**Clinical Background**:
Glucose variability (high coefficient of variation) is associated with increased mortality in ICU patients and diabetic complications. Hypoglycemia (<70 mg/dL) is a medical emergency; hyperglycemia (>200 mg/dL) indicates poor control.

**Window Configuration**:
- **Window Type**: Sliding
- **Window Size**: 24 hours
- **Slide Interval**: 4 hours
- **Analysis**: Statistical (mean, CV), hypoglycemia/hyperglycemia detection

**Implementation**: (Added to `LabTrendAnalyzer.java`)
```java
public static DataStream<LabTrendAlert> analyzeGlucoseTrends(
        DataStream<SemanticEvent> labStream) {

    return labStream
        .filter(event -> isGlucoseLab(event))
        .keyBy(SemanticEvent::getPatientId)
        .window(SlidingEventTimeWindows.of(Time.hours(24), Time.hours(4)))
        .apply(new GlucoseTrendWindowFunction());
}

public static class GlucoseTrendWindowFunction
        implements WindowFunction<SemanticEvent, LabTrendAlert, String, TimeWindow> {

    @Override
    public void apply(String patientId, TimeWindow window,
                     Iterable<SemanticEvent> events,
                     Collector<LabTrendAlert> out) {

        List<Double> glucoseValues = extractGlucoseValues(events);

        if (glucoseValues.size() < 3) {
            return; // Need at least 3 readings
        }

        // Calculate statistics
        double mean = glucoseValues.stream().mapToDouble(Double::doubleValue).average().orElse(0);
        double min = glucoseValues.stream().mapToDouble(Double::doubleValue).min().orElse(0);
        double max = glucoseValues.stream().mapToDouble(Double::doubleValue).max().orElse(0);

        // Calculate standard deviation
        double variance = glucoseValues.stream()
            .mapToDouble(v -> Math.pow(v - mean, 2))
            .average()
            .orElse(0);
        double stdDev = Math.sqrt(variance);

        // Calculate coefficient of variation (CV)
        double cv = (stdDev / mean) * 100;

        // Detect concerning patterns
        boolean hasHypoglycemia = min < 70;
        boolean hasHyperglycemia = max > 200;
        boolean highVariability = cv > 36;  // CV >36% = high variability

        if (hasHypoglycemia || hasHyperglycemia || highVariability) {
            String interpretation = interpretGlucosePattern(
                mean, min, max, cv, hasHypoglycemia, hasHyperglycemia, highVariability
            );

            LabTrendAlert alert = new LabTrendAlert();
            alert.setPatientId(patientId);
            alert.setLabName("Glucose");
            alert.setWindowStart(window.getStart());
            alert.setWindowEnd(window.getEnd());
            alert.setMeanValue(mean);
            alert.setMinValue(min);
            alert.setMaxValue(max);
            alert.setStandardDeviation(stdDev);
            alert.setCoefficientOfVariation(cv);
            alert.setDataPoints(glucoseValues.size());
            alert.setInterpretation(interpretation);

            out.collect(alert);
        }
    }

    private String interpretGlucosePattern(double mean, double min, double max,
                                          double cv, boolean hypoglycemia,
                                          boolean hyperglycemia, boolean highVariability) {
        StringBuilder interpretation = new StringBuilder();

        if (hypoglycemia) {
            interpretation.append(String.format(
                "🔴 HYPOGLYCEMIA DETECTED: Minimum glucose %.0f mg/dL (<70). " +
                "IMMEDIATE treatment required. Risk of seizures/loss of consciousness. " +
                "Administer glucose per protocol. Review insulin regimen.\n",
                min
            ));
        }

        if (hyperglycemia) {
            interpretation.append(String.format(
                "⚠️ HYPERGLYCEMIA: Maximum glucose %.0f mg/dL (>200). " +
                "Poor glycemic control. Consider insulin adjustment. " +
                "Risk of DKA if Type 1 diabetes. Assess for infection.\n",
                max
            ));
        }

        if (highVariability) {
            interpretation.append(String.format(
                "⚠️ HIGH GLUCOSE VARIABILITY: CV=%.1f%% (>36%%). " +
                "Erratic glucose control with swings from %.0f to %.0f mg/dL. " +
                "Increased mortality risk in ICU patients. " +
                "Review insulin dosing, meal timing, and correction factors.\n",
                cv, min, max
            ));
        }

        interpretation.append(String.format(
            "24h Summary: Mean %.0f mg/dL, Range %.0f-%.0f mg/dL, CV %.1f%%",
            mean, min, max, cv
        ));

        return interpretation.toString();
    }
}
```

---

### Feature 3.4: Vital Sign Variability Detection

**Clinical Background**:
High variability in vital signs indicates physiologic instability and is associated with increased risk of adverse events. Coefficient of variation (CV) >15% for vitals is concerning.

**Window Configuration**:
- **Window Type**: Sliding
- **Window Size**: 4 hours
- **Slide Interval**: 1 hour
- **Analysis**: CV calculation for each vital sign type

**Implementation**:

**File**: `com/cardiofit/flink/analytics/VitalVariabilityAnalyzer.java` (NEW)
```java
package com.cardiofit.flink.analytics;

import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

public class VitalVariabilityAnalyzer {

    public static DataStream<VitalVariabilityAlert> analyzeVitalVariability(
            DataStream<SemanticEvent> vitalStream) {

        return vitalStream
            .filter(event -> hasVitalSigns(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Time.hours(4), Time.hours(1)))
            .apply(new VitalVariabilityWindowFunction());
    }

    public static class VitalVariabilityWindowFunction
            implements WindowFunction<SemanticEvent, VitalVariabilityAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<VitalVariabilityAlert> out) {

            // Group vitals by type
            Map<String, List<Double>> vitalsByType = new HashMap<>();

            for (SemanticEvent event : events) {
                Map<String, Object> vitals = getVitalSigns(event);
                for (Map.Entry<String, Object> entry : vitals.entrySet()) {
                    String vitalType = entry.getKey();
                    Double value = parseDouble(entry.getValue());

                    vitalsByType.computeIfAbsent(vitalType, k -> new ArrayList<>())
                               .add(value);
                }
            }

            // Calculate coefficient of variation for each vital type
            Map<String, Double> cvMap = new HashMap<>();
            List<String> unstableVitals = new ArrayList<>();

            for (Map.Entry<String, List<Double>> entry : vitalsByType.entrySet()) {
                String vitalType = entry.getKey();
                List<Double> values = entry.getValue();

                if (values.size() < 4) continue; // Need multiple readings

                double mean = values.stream().mapToDouble(Double::doubleValue).average().orElse(0);
                double variance = values.stream()
                    .mapToDouble(v -> Math.pow(v - mean, 2))
                    .average()
                    .orElse(0);
                double stdDev = Math.sqrt(variance);
                double cv = (stdDev / mean) * 100;

                cvMap.put(vitalType, cv);

                // Thresholds for concerning variability
                double cvThreshold = getCVThreshold(vitalType);
                if (cv > cvThreshold) {
                    unstableVitals.add(String.format("%s: CV=%.1f%% (threshold: %.0f%%)",
                        formatVitalName(vitalType), cv, cvThreshold));
                }
            }

            // Generate alert if any vital has high variability
            if (!unstableVitals.isEmpty()) {
                String interpretation = interpretVariability(unstableVitals, cvMap);

                VitalVariabilityAlert alert = new VitalVariabilityAlert();
                alert.setPatientId(patientId);
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());
                alert.setCvMap(cvMap);
                alert.setUnstableVitals(unstableVitals);
                alert.setInterpretation(interpretation);
                alert.setDataPointsPerVital(
                    vitalsByType.entrySet().stream()
                        .collect(Collectors.toMap(
                            Map.Entry::getKey,
                            e -> e.getValue().size()
                        ))
                );

                out.collect(alert);
            }
        }

        private double getCVThreshold(String vitalType) {
            // Clinical thresholds for concerning CV
            switch (vitalType) {
                case "heart_rate":
                    return 15.0;  // CV >15% concerning
                case "systolic_bp":
                case "diastolic_bp":
                    return 15.0;
                case "respiratory_rate":
                    return 20.0;  // RR more variable normally
                case "oxygen_saturation":
                    return 5.0;   // SpO2 should be very stable
                case "temperature":
                    return 2.0;   // Temperature should be very stable
                default:
                    return 15.0;
            }
        }

        private String interpretVariability(List<String> unstableVitals,
                                           Map<String, Double> cvMap) {
            StringBuilder interpretation = new StringBuilder();

            interpretation.append("⚠️ HIGH VITAL SIGN VARIABILITY DETECTED\n\n");
            interpretation.append("Unstable vital signs indicate physiologic instability and " +
                                "increased risk of adverse events:\n\n");

            for (String vital : unstableVitals) {
                interpretation.append("  • ").append(vital).append("\n");
            }

            interpretation.append("\nRECOMMENDATIONS:\n");
            interpretation.append("1. Increase vital sign monitoring frequency to q15-30min\n");
            interpretation.append("2. Assess for underlying cause (pain, anxiety, fluid status)\n");
            interpretation.append("3. Notify physician of instability\n");
            interpretation.append("4. Consider continuous monitoring\n");
            interpretation.append("5. Review recent interventions and medications\n");

            // Specific recommendations based on which vitals are unstable
            if (cvMap.containsKey("heart_rate") && cvMap.get("heart_rate") > 15) {
                interpretation.append("6. HR variability: Assess for arrhythmia, pain, anxiety\n");
            }
            if (cvMap.containsKey("systolic_bp") && cvMap.get("systolic_bp") > 15) {
                interpretation.append("6. BP variability: Review fluid status, vasopressor titration\n");
            }
            if (cvMap.containsKey("oxygen_saturation") && cvMap.get("oxygen_saturation") > 5) {
                interpretation.append("6. SpO2 variability: Assess respiratory status, consider ABG\n");
            }

            return interpretation.toString();
        }
    }
}
```

**Data Model**: `com/cardiofit/flink/models/VitalVariabilityAlert.java` (NEW)
```java
public class VitalVariabilityAlert implements Serializable {
    private String patientId;
    private Long windowStart;
    private Long windowEnd;
    private Map<String, Double> cvMap;
    private List<String> unstableVitals;
    private String interpretation;
    private Map<String, Integer> dataPointsPerVital;

    // Getters and setters
}
```

### Phase 3 Deliverables
- ✅ MEWS Calculator with scoring algorithm
- ✅ Creatinine trend analyzer with KDIGO AKI detection
- ✅ Glucose trend analyzer with variability detection
- ✅ Vital sign variability analyzer with CV calculations
- ✅ Alert data models for each analytics feature
- ✅ Clinical interpretations and recommendations
- ✅ Integration into Module4 pipeline

---

## Phase 4: Data Models & Support Classes

### Objective
Create all necessary data models and helper classes to support new features.

### Files to Create

1. **`MEWSAlert.java`** - MEWS alert data model (✅ Described above)
2. **`LabTrendAlert.java`** - Lab trend alert data model
3. **`VitalVariabilityAlert.java`** - Vital variability alert (✅ Described above)
4. **`DrugLabMonitoringAlert.java`** - Drug monitoring alert
5. **`TrendAnalysis.java`** - Trend calculation result
6. **`TrendDirection.java`** - Trend direction enum

### Implementation

**File**: `com/cardiofit/flink/models/LabTrendAlert.java` (NEW)
```java
package com.cardiofit.flink.models;

import java.io.Serializable;

public class LabTrendAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String labName;
    private Long windowStart;
    private Long windowEnd;
    private Double firstValue;
    private Double lastValue;
    private Double absoluteChange;
    private Double percentChange;
    private Double trendSlope;
    private String trendDirection;
    private Integer dataPoints;
    private Double rSquared;
    private String akiStage;  // For creatinine
    private Double meanValue;  // For glucose
    private Double minValue;
    private Double maxValue;
    private Double standardDeviation;
    private Double coefficientOfVariation;
    private String interpretation;

    // Default constructor
    public LabTrendAlert() {}

    // Getters and setters for all fields
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getLabName() { return labName; }
    public void setLabName(String labName) { this.labName = labName; }

    public Long getWindowStart() { return windowStart; }
    public void setWindowStart(Long windowStart) { this.windowStart = windowStart; }

    public Long getWindowEnd() { return windowEnd; }
    public void setWindowEnd(Long windowEnd) { this.windowEnd = windowEnd; }

    public Double getFirstValue() { return firstValue; }
    public void setFirstValue(Double firstValue) { this.firstValue = firstValue; }

    public Double getLastValue() { return lastValue; }
    public void setLastValue(Double lastValue) { this.lastValue = lastValue; }

    public Double getAbsoluteChange() { return absoluteChange; }
    public void setAbsoluteChange(Double absoluteChange) { this.absoluteChange = absoluteChange; }

    public Double getPercentChange() { return percentChange; }
    public void setPercentChange(Double percentChange) { this.percentChange = percentChange; }

    public Double getTrendSlope() { return trendSlope; }
    public void setTrendSlope(Double trendSlope) { this.trendSlope = trendSlope; }

    public String getTrendDirection() { return trendDirection; }
    public void setTrendDirection(String trendDirection) { this.trendDirection = trendDirection; }

    public Integer getDataPoints() { return dataPoints; }
    public void setDataPoints(Integer dataPoints) { this.dataPoints = dataPoints; }

    public Double getRSquared() { return rSquared; }
    public void setRSquared(Double rSquared) { this.rSquared = rSquared; }

    public String getAkiStage() { return akiStage; }
    public void setAkiStage(String akiStage) { this.akiStage = akiStage; }

    public Double getMeanValue() { return meanValue; }
    public void setMeanValue(Double meanValue) { this.meanValue = meanValue; }

    public Double getMinValue() { return minValue; }
    public void setMinValue(Double minValue) { this.minValue = minValue; }

    public Double getMaxValue() { return maxValue; }
    public void setMaxValue(Double maxValue) { this.maxValue = maxValue; }

    public Double getStandardDeviation() { return standardDeviation; }
    public void setStandardDeviation(Double standardDeviation) {
        this.standardDeviation = standardDeviation;
    }

    public Double getCoefficientOfVariation() { return coefficientOfVariation; }
    public void setCoefficientOfVariation(Double coefficientOfVariation) {
        this.coefficientOfVariation = coefficientOfVariation;
    }

    public String getInterpretation() { return interpretation; }
    public void setInterpretation(String interpretation) { this.interpretation = interpretation; }
}
```

**File**: `com/cardiofit/flink/models/TrendAnalysis.java` (NEW)
```java
package com.cardiofit.flink.models;

import java.io.Serializable;

public class TrendAnalysis implements Serializable {
    private static final long serialVersionUID = 1L;

    private double slope;
    private double intercept;
    private double rSquared;
    private TrendDirection direction;

    public TrendAnalysis() {}

    public TrendAnalysis(double slope, double intercept, double rSquared, TrendDirection direction) {
        this.slope = slope;
        this.intercept = intercept;
        this.rSquared = rSquared;
        this.direction = direction;
    }

    // Getters and setters
    public double getSlope() { return slope; }
    public void setSlope(double slope) { this.slope = slope; }

    public double getIntercept() { return intercept; }
    public void setIntercept(double intercept) { this.intercept = intercept; }

    public double getRSquared() { return rSquared; }
    public void setRSquared(double rSquared) { this.rSquared = rSquared; }

    public TrendDirection getDirection() { return direction; }
    public void setDirection(TrendDirection direction) { this.direction = direction; }

    @Override
    public String toString() {
        return String.format("TrendAnalysis{direction=%s, slope=%.4f, r²=%.4f}",
            direction, slope, rSquared);
    }
}
```

**File**: `com/cardiofit/flink/models/TrendDirection.java` (NEW)
```java
package com.cardiofit.flink.models;

public enum TrendDirection {
    STABLE,
    INCREASING,
    RAPIDLY_INCREASING,
    DECREASING,
    RAPIDLY_DECREASING;

    public static TrendDirection fromSlope(double slope) {
        if (Math.abs(slope) < 0.01) {
            return STABLE;
        } else if (slope > 0.1) {
            return RAPIDLY_INCREASING;
        } else if (slope > 0) {
            return INCREASING;
        } else if (slope < -0.1) {
            return RAPIDLY_DECREASING;
        } else {
            return DECREASING;
        }
    }
}
```

**File**: `com/cardiofit/flink/models/DrugLabMonitoringAlert.java` (NEW)
```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.List;

public class DrugLabMonitoringAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String medicationName;
    private String medicationCode;
    private Long medicationStartTime;
    private List<String> requiredLabs;
    private Long hoursElapsed;
    private Long maxWindowHours;
    private String severity;
    private String interpretation;
    private List<String> recommendedActions;

    public DrugLabMonitoringAlert() {}

    // Getters and setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getMedicationName() { return medicationName; }
    public void setMedicationName(String medicationName) { this.medicationName = medicationName; }

    public String getMedicationCode() { return medicationCode; }
    public void setMedicationCode(String medicationCode) { this.medicationCode = medicationCode; }

    public Long getMedicationStartTime() { return medicationStartTime; }
    public void setMedicationStartTime(Long medicationStartTime) {
        this.medicationStartTime = medicationStartTime;
    }

    public List<String> getRequiredLabs() { return requiredLabs; }
    public void setRequiredLabs(List<String> requiredLabs) { this.requiredLabs = requiredLabs; }

    public Long getHoursElapsed() { return hoursElapsed; }
    public void setHoursElapsed(Long hoursElapsed) { this.hoursElapsed = hoursElapsed; }

    public Long getMaxWindowHours() { return maxWindowHours; }
    public void setMaxWindowHours(Long maxWindowHours) { this.maxWindowHours = maxWindowHours; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getInterpretation() { return interpretation; }
    public void setInterpretation(String interpretation) { this.interpretation = interpretation; }

    public List<String> getRecommendedActions() { return recommendedActions; }
    public void setRecommendedActions(List<String> recommendedActions) {
        this.recommendedActions = recommendedActions;
    }
}
```

### Phase 4 Deliverables
- ✅ All alert data models created
- ✅ Helper classes (TrendAnalysis, TrendDirection) created
- ✅ Serializable implementations
- ✅ Proper getters/setters
- ✅ Ready for integration

---

## Phase 5: Integration & Pipeline Updates

### Objective
Wire all new patterns and analytics into the unified Module4 pipeline with proper routing.

### Integration Tasks

**Task 5.1: Update createPatternDetectionPipeline()**

Add new pattern streams:
```java
// ===== NEW: Advanced CEP Patterns =====

// Sepsis early warning pattern
PatternStream<SemanticEvent> sepsisPatterns =
    ClinicalPatterns.detectSepsisPattern(keyedSemanticEvents);

DataStream<PatternEvent> sepsisEvents = sepsisPatterns
    .select(new SepsisPatternSelectFunction())
    .uid("Sepsis Early Warning Events");

// Rapid clinical deterioration pattern
PatternStream<SemanticEvent> rapidDeteriorationPatterns =
    ClinicalPatterns.detectRapidDeteriorationPattern(keyedSemanticEvents);

DataStream<PatternEvent> rapidDeteriorationEvents = rapidDeteriorationPatterns
    .select(new RapidDeteriorationPatternSelectFunction())
    .uid("Rapid Deterioration Events");

// Drug-lab monitoring pattern
PatternStream<SemanticEvent> drugLabPatterns =
    ClinicalPatterns.detectDrugLabMonitoringPattern(keyedSemanticEvents);

DataStream<PatternEvent> drugLabEvents = drugLabPatterns
    .select(new DrugLabMonitoringPatternSelectFunction())
    .uid("Drug Lab Monitoring Events");

// Sepsis pathway compliance pattern
PatternStream<SemanticEvent> sepsisPathwayPatterns =
    ClinicalPatterns.detectSepsisPathwayCompliancePattern(keyedSemanticEvents);

DataStream<PatternEvent> sepsisPathwayEvents = sepsisPathwayPatterns
    .select(new SepsisPathwayCompliancePatternSelectFunction())
    .uid("Sepsis Pathway Compliance Events");
```

Add new windowed analytics:
```java
// ===== NEW: Advanced Windowed Analytics =====

// MEWS calculation
DataStream<MEWSAlert> mewsAlerts = MEWSCalculator.calculateMEWS(keyedSemanticEvents);

// Lab trend analysis
DataStream<LabTrendAlert> creatinineTrends =
    LabTrendAnalyzer.analyzeCreatinineTrends(keyedSemanticEvents);

DataStream<LabTrendAlert> glucoseTrends =
    LabTrendAnalyzer.analyzeGlucoseTrends(keyedSemanticEvents);

// Vital variability analysis
DataStream<VitalVariabilityAlert> vitalVariability =
    VitalVariabilityAnalyzer.analyzeVitalVariability(keyedSemanticEvents);
```

Update unified stream:
```java
// ===== Unified Pattern Stream =====
DataStream<PatternEvent> allPatternEvents = deteriorationEvents
    .union(medicationEvents)
    .union(vitalTrendEvents)
    .union(pathwayEvents)
    .union(akiEvents)
    .union(trendAnalysis)
    .union(anomalyDetection)
    .union(protocolMonitoring)
    .union(sepsisEvents)  // NEW
    .union(rapidDeteriorationEvents)  // NEW
    .union(drugLabEvents)  // NEW
    .union(sepsisPathwayEvents);  // NEW
```

**Task 5.2: Create Pattern Select Functions**

Implement select functions for each new pattern (similar to existing ones).

**Task 5.3: Update Routing Logic**

Add new sinks for critical alerts:
```java
// NEW: Critical alerts sink (MEWS, severe deterioration)
private static KafkaSink<MEWSAlert> createCriticalAlertsSink() {
    return KafkaSink.<MEWSAlert>builder()
        .setBootstrapServers(getBootstrapServers())
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic(getTopicName("MODULE4_CRITICAL_ALERTS_TOPIC", "critical-alerts.v1"))
            .setKeySerializationSchema((MEWSAlert alert) -> alert.getPatientId().getBytes())
            .setValueSerializationSchema(new MEWSAlertSerializer())
            .build())
        .setKafkaProducerConfig(KafkaConfigLoader.getAutoProducerConfig())
        .build();
}

// Route MEWS alerts to critical alerts topic
mewsAlerts
    .filter(alert -> alert.getMewsScore() >= 3)
    .sinkTo(createCriticalAlertsSink())
    .uid("MEWS Critical Alerts Sink");
```

### Phase 5 Deliverables
- ✅ All new patterns integrated into pipeline
- ✅ Pattern select functions implemented
- ✅ Routing logic updated
- ✅ New Kafka sinks created
- ✅ UID/name assignments for checkpointing

---

## Phase 6: Documentation & Testing

### Objective
Create comprehensive documentation and test scenarios for all new features.

### Documentation Files to Create

1. **`MODULE4_CONFIGURATION_ENVIRONMENT_VARIABLES.md`**
2. **`MODULE4_CLINICAL_PATTERNS_CATALOG.md`**
3. **`MODULE4_WINDOWED_ANALYTICS_GUIDE.md`**

### Testing Scenarios

Create test event JSONs for:
1. Sepsis early warning detection (3-event sequence)
2. MEWS calculation (score ≥3)
3. Creatinine trend (AKI Stage 1 detection)
4. Glucose variability detection (CV >36%)
5. Rapid deterioration pattern (multi-vital)
6. Drug-lab monitoring (ACE inhibitor without labs)
7. Sepsis pathway compliance (complete bundle)

### Phase 6 Deliverables
- ✅ Complete configuration documentation
- ✅ Clinical patterns catalog with rationale
- ✅ Windowed analytics guide with scoring
- ✅ Test scenarios with expected outputs
- ✅ Deployment instructions

---

## Build & Deployment

### Build Commands
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
mvn clean package -DskipTests
```

### Deployment Commands
```bash
# Upload JAR
curl -X POST -F "jarfile=@target/flink-ehr-intelligence-1.0.0.jar" \
  http://localhost:8081/jars/upload

# Get JAR ID
JAR_ID=$(curl -s http://localhost:8081/jars | jq -r '.files | sort_by(.uploaded) | reverse | .[0].id')

# Cancel old Module 4 job
OLD_JOB_ID=$(curl -s http://localhost:8081/jobs | jq -r '.jobs[] | select(.name | contains("Module 4")) | select(.status == "RUNNING") | .id')
curl -X PATCH "http://localhost:8081/jobs/$OLD_JOB_ID?mode=cancel"

# Deploy new Module 4
curl -X POST "http://localhost:8081/jars/$JAR_ID/run" \
  -H "Content-Type: application/json" \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module4_PatternDetection","parallelism":8}'
```

### Verification Commands
```bash
# Check job status
curl -s http://localhost:8081/jobs | jq '.jobs[] | select(.name | contains("Module 4"))'

# Check Kafka topics for output
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic critical-alerts.v1 --from-beginning --max-messages 5
```

---

## Success Criteria

### Functional Requirements
- ✅ All 9 CEP patterns detecting events correctly
- ✅ MEWS calculation producing alerts for scores ≥3
- ✅ Lab trends detecting AKI, glucose variability
- ✅ Vital variability identifying unstable patients
- ✅ All configuration externalized
- ✅ No hardcoded values remaining

### Performance Requirements
- ✅ Throughput: 10,000 events/second sustained
- ✅ Latency (p99): <2 seconds event-to-alert
- ✅ Pattern detection accuracy: >95%
- ✅ Checkpoint success rate: >99%

### Quality Requirements
- ✅ Build succeeds without errors
- ✅ All tests pass
- ✅ Documentation complete
- ✅ Clinical interpretations accurate
- ✅ Backward compatible with Module 3

---

## Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: Configuration | 30 min | All topics configurable |
| Phase 2: CEP Patterns | 2 hours | 4 new patterns integrated |
| Phase 3: Windowed Analytics | 3 hours | MEWS, lab trends, variability |
| Phase 4: Data Models | 1 hour | All alert models created |
| Phase 5: Integration | 1 hour | Pipeline fully wired |
| Phase 6: Documentation | 1 hour | Docs and tests complete |
| **TOTAL** | **8.5 hours** | **Production-ready Module 4** |

---

## Risk Mitigation

### Technical Risks
- **Risk**: Pattern complexity causes performance degradation
  - **Mitigation**: Windowed analytics use efficient aggregations, CEP uses indexed state

- **Risk**: Serialization issues with new data models
  - **Mitigation**: All models implement Serializable, test thoroughly

- **Risk**: Configuration errors in production
  - **Mitigation**: Sensible defaults, validation on startup, documentation

### Clinical Risks
- **Risk**: False positives overwhelming clinicians
  - **Mitigation**: Tune thresholds based on clinical validation, confidence scores

- **Risk**: Missed critical patterns (false negatives)
  - **Mitigation**: Conservative thresholds (favor sensitivity over specificity initially)

- **Risk**: Clinical interpretation inaccuracies
  - **Mitigation**: Evidence-based guidelines, clinical review of all interpretations

---

## Next Steps After Completion

1. **Clinical Validation**: Partner with clinical team to validate pattern accuracy
2. **Threshold Tuning**: Adjust detection thresholds based on real-world performance
3. **Alert Routing**: Configure alert delivery to appropriate care teams
4. **Dashboard Integration**: Connect patterns to monitoring dashboards
5. **Reporting**: Generate clinical outcome metrics and pattern analytics

---

**Implementation Ready**: This plan is ready for immediate execution. All components are well-defined with clear deliverables, timelines, and success criteria.
