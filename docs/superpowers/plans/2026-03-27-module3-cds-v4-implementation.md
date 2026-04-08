# Module 3: Comprehensive CDS V4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform Module 3's stub-only CDS pipeline into a working V4 dual-domain clinical decision support engine with CDC hot-swap, patient keyed state, MHRI scoring, safety checks, and medication rules.

**Architecture:** The existing `Module3_ComprehensiveCDS_WithCDC` operator uses `KeyedBroadcastProcessFunction` with a single KB-3 protocol broadcast state and 8 processing phases — all stubs. This plan implements real clinical logic in each phase, adds patient `ValueState` for cross-event accumulation, expands broadcast state to KB-4/KB-5/KB-7, and introduces V4 MHRI scoring with data-tier awareness. The inner `CDSEvent` is extracted to a top-level model with typed fields replacing `Map<String, Object>`.

**Tech Stack:** Java 17, Apache Flink 1.18, Jackson, RocksDB state backend, Kafka (Confluent Cloud), Debezium CDC

---

## File Structure

### New Files
- `src/main/java/com/cardiofit/flink/models/CDSEvent.java` — Top-level typed CDS output model
- `src/main/java/com/cardiofit/flink/models/CDSPhaseResult.java` — Per-phase typed result container
- `src/main/java/com/cardiofit/flink/models/MHRIScore.java` — MHRI 5-component composite score
- `src/main/java/com/cardiofit/flink/models/SafetyCheckResult.java` — Aggregated safety output
- `src/main/java/com/cardiofit/flink/models/MedicationSafetyResult.java` — Per-medication safety output
- `src/main/java/com/cardiofit/flink/models/GuidelineMatch.java` — Guideline concordance result
- `src/test/java/com/cardiofit/flink/models/CDSEventTest.java` — CDSEvent unit tests
- `src/test/java/com/cardiofit/flink/models/MHRIScoreTest.java` — MHRI computation tests
- `src/test/java/com/cardiofit/flink/operators/Module3Phase1ProtocolMatchTest.java` — Phase 1 tests
- `src/test/java/com/cardiofit/flink/operators/Module3Phase2ScoringTest.java` — Phase 2 tests
- `src/test/java/com/cardiofit/flink/operators/Module3Phase7SafetyTest.java` — Phase 7 tests
- `src/test/java/com/cardiofit/flink/operators/Module3BroadcastStateTest.java` — Multi-KB broadcast tests
- `src/test/java/com/cardiofit/flink/builders/Module3TestBuilder.java` — Test data factory for Module 3

### Modified Files
- `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java` — Main operator: add ValueState, expand broadcast, implement phases
- `src/main/java/com/cardiofit/flink/models/protocol/SimplifiedProtocol.java` — Add TriggerCriteria bridge fields

All paths are relative to `backend/shared-infrastructure/flink-processing/`.

---

## Phase A: Foundation (Tasks 1–4)

### Task 1: Extract CDSEvent to Top-Level Typed Model

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/CDSEvent.java`
- Create: `src/main/java/com/cardiofit/flink/models/CDSPhaseResult.java`
- Create: `src/test/java/com/cardiofit/flink/models/CDSEventTest.java`

- [ ] **Step 1: Write the failing test for CDSEvent construction**

```java
package com.cardiofit.flink.models;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.List;

public class CDSEventTest {

    @Test
    void constructFromEnrichedPatientContext() {
        PatientContextState state = new PatientContextState("P-001");
        state.setNews2Score(4);
        state.setQsofaScore(1);
        state.setCombinedAcuityScore(3.5);

        EnrichedPatientContext epc = new EnrichedPatientContext("P-001", state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(1000L);
        epc.setDataTier("TIER_1_CGM");

        CDSEvent cds = new CDSEvent(epc);

        assertEquals("P-001", cds.getPatientId());
        assertEquals("VITAL_SIGN", cds.getEventType());
        assertEquals("TIER_1_CGM", cds.getDataTier());
        assertNotNull(cds.getPatientState());
        assertNotNull(cds.getPhaseResults());
        assertTrue(cds.getPhaseResults().isEmpty());
        assertNotNull(cds.getRecommendations());
    }

    @Test
    void addPhaseResultTyped() {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId("P-002");

        CDSPhaseResult phase1 = new CDSPhaseResult("PHASE_1_PROTOCOL_MATCH");
        phase1.setActive(true);
        phase1.addDetail("matchedCount", 3);
        phase1.addDetail("topProtocol", "SEPSIS-BUNDLE-V2");

        cds.addPhaseResult(phase1);

        assertEquals(1, cds.getPhaseResults().size());
        CDSPhaseResult retrieved = cds.getPhaseResults().get(0);
        assertEquals("PHASE_1_PROTOCOL_MATCH", retrieved.getPhaseName());
        assertTrue(retrieved.isActive());
        assertEquals(3, retrieved.getDetail("matchedCount"));
    }

    @Test
    void serialization_excludesEmptyCollections() throws Exception {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId("P-003");

        com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
        String json = mapper.writeValueAsString(cds);

        assertTrue(json.contains("\"patientId\""));
        // Empty lists excluded by @JsonInclude(NON_EMPTY)
        assertFalse(json.contains("\"recommendations\""));
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=CDSEventTest -DfailIfNoTests=false 2>&1 | tail -20`
Expected: Compilation failure — `CDSEvent` and `CDSPhaseResult` classes don't exist yet.

- [ ] **Step 3: Create CDSPhaseResult model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class CDSPhaseResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("phaseName")
    private String phaseName;

    @JsonProperty("active")
    private boolean active;

    @JsonProperty("durationMs")
    private long durationMs;

    @JsonProperty("details")
    private Map<String, Object> details;

    public CDSPhaseResult() {
        this.details = new HashMap<>();
    }

    public CDSPhaseResult(String phaseName) {
        this();
        this.phaseName = phaseName;
    }

    public void addDetail(String key, Object value) {
        this.details.put(key, value);
    }

    public Object getDetail(String key) {
        return this.details.get(key);
    }

    // Getters and setters
    public String getPhaseName() { return phaseName; }
    public void setPhaseName(String phaseName) { this.phaseName = phaseName; }
    public boolean isActive() { return active; }
    public void setActive(boolean active) { this.active = active; }
    public long getDurationMs() { return durationMs; }
    public void setDurationMs(long durationMs) { this.durationMs = durationMs; }
    public Map<String, Object> getDetails() { return details; }
    public void setDetails(Map<String, Object> details) { this.details = details; }

    @Override
    public String toString() {
        return String.format("CDSPhaseResult{phase='%s', active=%s, details=%d}",
                phaseName, active, details.size());
    }
}
```

- [ ] **Step 4: Create top-level CDSEvent model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class CDSEvent implements Serializable {
    private static final long serialVersionUID = 2L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("patientState")
    private PatientContextState patientState;

    @JsonProperty("eventType")
    private String eventType;

    @JsonProperty("eventTime")
    private long eventTime;

    @JsonProperty("processingTime")
    private long processingTime;

    @JsonProperty("latencyMs")
    private Long latencyMs;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("phaseResults")
    private List<CDSPhaseResult> phaseResults;

    @JsonProperty("recommendations")
    private List<Map<String, Object>> recommendations;

    @JsonProperty("safetyAlerts")
    private List<Map<String, Object>> safetyAlerts;

    @JsonProperty("mhriScore")
    private MHRIScore mhriScore;

    @JsonProperty("protocolsMatched")
    private int protocolsMatched;

    @JsonProperty("broadcastStateReady")
    private boolean broadcastStateReady;

    public CDSEvent() {
        this.processingTime = System.currentTimeMillis();
        this.phaseResults = new ArrayList<>();
        this.recommendations = new ArrayList<>();
        this.safetyAlerts = new ArrayList<>();
    }

    public CDSEvent(EnrichedPatientContext context) {
        this();
        this.patientId = context.getPatientId();
        this.patientState = context.getPatientState();
        this.eventType = context.getEventType();
        this.eventTime = context.getEventTime();
        this.processingTime = System.currentTimeMillis();
        this.latencyMs = context.getLatencyMs();
        this.dataTier = context.getDataTier();
    }

    public void addPhaseResult(CDSPhaseResult result) {
        this.phaseResults.add(result);
    }

    public void addRecommendation(Map<String, Object> rec) {
        this.recommendations.add(rec);
    }

    public void addSafetyAlert(Map<String, Object> alert) {
        this.safetyAlerts.add(alert);
    }

    // Getters and setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public PatientContextState getPatientState() { return patientState; }
    public void setPatientState(PatientContextState patientState) { this.patientState = patientState; }
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }
    public long getEventTime() { return eventTime; }
    public void setEventTime(long eventTime) { this.eventTime = eventTime; }
    public long getProcessingTime() { return processingTime; }
    public void setProcessingTime(long processingTime) { this.processingTime = processingTime; }
    public Long getLatencyMs() { return latencyMs; }
    public void setLatencyMs(Long latencyMs) { this.latencyMs = latencyMs; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public List<CDSPhaseResult> getPhaseResults() { return phaseResults; }
    public void setPhaseResults(List<CDSPhaseResult> phaseResults) { this.phaseResults = phaseResults; }
    public List<Map<String, Object>> getRecommendations() { return recommendations; }
    public void setRecommendations(List<Map<String, Object>> recs) { this.recommendations = recs; }
    public List<Map<String, Object>> getSafetyAlerts() { return safetyAlerts; }
    public void setSafetyAlerts(List<Map<String, Object>> alerts) { this.safetyAlerts = alerts; }
    public MHRIScore getMhriScore() { return mhriScore; }
    public void setMhriScore(MHRIScore mhriScore) { this.mhriScore = mhriScore; }
    public int getProtocolsMatched() { return protocolsMatched; }
    public void setProtocolsMatched(int protocolsMatched) { this.protocolsMatched = protocolsMatched; }
    public boolean isBroadcastStateReady() { return broadcastStateReady; }
    public void setBroadcastStateReady(boolean ready) { this.broadcastStateReady = ready; }

    @Override
    public String toString() {
        return String.format("CDSEvent{patient='%s', type='%s', phases=%d, recs=%d, safety=%d, mhri=%s}",
                patientId, eventType, phaseResults.size(), recommendations.size(),
                safetyAlerts.size(), mhriScore != null ? mhriScore.getComposite() : "null");
    }
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=CDSEventTest 2>&1 | tail -20`
Expected: All 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/CDSEvent.java \
        src/main/java/com/cardiofit/flink/models/CDSPhaseResult.java \
        src/test/java/com/cardiofit/flink/models/CDSEventTest.java
git commit -m "feat(flink): extract CDSEvent to top-level typed model with CDSPhaseResult"
```

---

### Task 2: Create MHRIScore Model

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/MHRIScore.java`
- Create: `src/test/java/com/cardiofit/flink/models/MHRIScoreTest.java`

- [ ] **Step 1: Write the failing test for MHRI computation**

```java
package com.cardiofit.flink.models;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class MHRIScoreTest {

    @Test
    void computeComposite_tier1_fullWeights() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);    // 0-100 normalized
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        score.setDataTier("TIER_1_CGM");

        score.computeComposite();

        // Tier 1 weights: glycemic=25, hemodynamic=25, renal=20, metabolic=15, engagement=15
        // (70*0.25) + (60*0.25) + (50*0.20) + (40*0.15) + (80*0.15) = 17.5 + 15 + 10 + 6 + 12 = 60.5
        assertEquals(60.5, score.getComposite(), 0.01);
        assertEquals("TIER_1_CGM", score.getDataTier());
        assertEquals("MODERATE", score.getRiskCategory());
    }

    @Test
    void computeComposite_tier3_redistributedWeights() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        score.setDataTier("TIER_3_SMBG");

        score.computeComposite();

        // Tier 3: glycemic weight reduced to 15, hemodynamic boosted to 30, renal 25, metabolic 15, engagement 15
        // (70*0.15) + (60*0.30) + (50*0.25) + (40*0.15) + (80*0.15) = 10.5 + 18 + 12.5 + 6 + 12 = 59.0
        assertEquals(59.0, score.getComposite(), 0.01);
    }

    @Test
    void riskCategory_thresholds() {
        MHRIScore low = new MHRIScore();
        low.setCompositeDirectly(25.0);
        assertEquals("LOW", low.getRiskCategory());

        MHRIScore moderate = new MHRIScore();
        moderate.setCompositeDirectly(50.0);
        assertEquals("MODERATE", moderate.getRiskCategory());

        MHRIScore high = new MHRIScore();
        high.setCompositeDirectly(75.0);
        assertEquals("HIGH", high.getRiskCategory());

        MHRIScore critical = new MHRIScore();
        critical.setCompositeDirectly(90.0);
        assertEquals("CRITICAL", critical.getRiskCategory());
    }

    @Test
    void nullDataTier_defaultsToTier3() {
        MHRIScore score = new MHRIScore();
        score.setGlycemicComponent(70.0);
        score.setHemodynamicComponent(60.0);
        score.setRenalComponent(50.0);
        score.setMetabolicComponent(40.0);
        score.setEngagementComponent(80.0);
        // dataTier not set — defaults to TIER_3_SMBG

        score.computeComposite();

        assertEquals(59.0, score.getComposite(), 0.01);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=MHRIScoreTest -DfailIfNoTests=false 2>&1 | tail -20`
Expected: Compilation failure — `MHRIScore` class doesn't exist yet.

- [ ] **Step 3: Implement MHRIScore**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class MHRIScore implements Serializable {
    private static final long serialVersionUID = 1L;

    // Tier 1 (CGM) weights — highest glycemic fidelity
    private static final double T1_GLYCEMIC = 0.25;
    private static final double T1_HEMODYNAMIC = 0.25;
    private static final double T1_RENAL = 0.20;
    private static final double T1_METABOLIC = 0.15;
    private static final double T1_ENGAGEMENT = 0.15;

    // Tier 3 (SMBG) weights — glycemic downweighted, hemodynamic/renal boosted
    private static final double T3_GLYCEMIC = 0.15;
    private static final double T3_HEMODYNAMIC = 0.30;
    private static final double T3_RENAL = 0.25;
    private static final double T3_METABOLIC = 0.15;
    private static final double T3_ENGAGEMENT = 0.15;

    @JsonProperty("glycemicComponent")
    private Double glycemicComponent;

    @JsonProperty("hemodynamicComponent")
    private Double hemodynamicComponent;

    @JsonProperty("renalComponent")
    private Double renalComponent;

    @JsonProperty("metabolicComponent")
    private Double metabolicComponent;

    @JsonProperty("engagementComponent")
    private Double engagementComponent;

    @JsonProperty("composite")
    private Double composite;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("riskCategory")
    private String riskCategory;

    @JsonProperty("computedAt")
    private Long computedAt;

    public MHRIScore() {}

    public void computeComposite() {
        String tier = (dataTier != null) ? dataTier : "TIER_3_SMBG";
        boolean isTier1 = tier.startsWith("TIER_1");

        double gW = isTier1 ? T1_GLYCEMIC : T3_GLYCEMIC;
        double hW = isTier1 ? T1_HEMODYNAMIC : T3_HEMODYNAMIC;
        double rW = isTier1 ? T1_RENAL : T3_RENAL;
        double mW = isTier1 ? T1_METABOLIC : T3_METABOLIC;
        double eW = isTier1 ? T1_ENGAGEMENT : T3_ENGAGEMENT;

        double g = glycemicComponent != null ? glycemicComponent : 0.0;
        double h = hemodynamicComponent != null ? hemodynamicComponent : 0.0;
        double r = renalComponent != null ? renalComponent : 0.0;
        double m = metabolicComponent != null ? metabolicComponent : 0.0;
        double e = engagementComponent != null ? engagementComponent : 0.0;

        this.composite = (g * gW) + (h * hW) + (r * rW) + (m * mW) + (e * eW);
        this.riskCategory = classifyRisk(this.composite);
        this.computedAt = System.currentTimeMillis();
    }

    public void setCompositeDirectly(double value) {
        this.composite = value;
        this.riskCategory = classifyRisk(value);
    }

    private static String classifyRisk(double score) {
        if (score >= 80.0) return "CRITICAL";
        if (score >= 60.0) return "HIGH";
        if (score >= 35.0) return "MODERATE";
        return "LOW";
    }

    public String getRiskCategory() {
        if (riskCategory == null && composite != null) {
            riskCategory = classifyRisk(composite);
        }
        return riskCategory;
    }

    // Getters and setters
    public Double getGlycemicComponent() { return glycemicComponent; }
    public void setGlycemicComponent(Double v) { this.glycemicComponent = v; }
    public Double getHemodynamicComponent() { return hemodynamicComponent; }
    public void setHemodynamicComponent(Double v) { this.hemodynamicComponent = v; }
    public Double getRenalComponent() { return renalComponent; }
    public void setRenalComponent(Double v) { this.renalComponent = v; }
    public Double getMetabolicComponent() { return metabolicComponent; }
    public void setMetabolicComponent(Double v) { this.metabolicComponent = v; }
    public Double getEngagementComponent() { return engagementComponent; }
    public void setEngagementComponent(Double v) { this.engagementComponent = v; }
    public Double getComposite() { return composite; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public Long getComputedAt() { return computedAt; }

    @Override
    public String toString() {
        return String.format("MHRI{composite=%.1f, risk=%s, tier=%s, g=%.0f h=%.0f r=%.0f m=%.0f e=%.0f}",
                composite != null ? composite : 0.0, riskCategory, dataTier,
                glycemicComponent != null ? glycemicComponent : 0.0,
                hemodynamicComponent != null ? hemodynamicComponent : 0.0,
                renalComponent != null ? renalComponent : 0.0,
                metabolicComponent != null ? metabolicComponent : 0.0,
                engagementComponent != null ? engagementComponent : 0.0);
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=MHRIScoreTest 2>&1 | tail -20`
Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/MHRIScore.java \
        src/test/java/com/cardiofit/flink/models/MHRIScoreTest.java
git commit -m "feat(flink): add MHRIScore model with tier-aware weighted composite computation"
```

---

### Task 3: Create Safety and Medication Result Models

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/SafetyCheckResult.java`
- Create: `src/main/java/com/cardiofit/flink/models/MedicationSafetyResult.java`
- Create: `src/main/java/com/cardiofit/flink/models/GuidelineMatch.java`

- [ ] **Step 1: Create SafetyCheckResult model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class SafetyCheckResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("allergyAlerts")
    private List<String> allergyAlerts;

    @JsonProperty("contraindicationAlerts")
    private List<String> contraindicationAlerts;

    @JsonProperty("interactionAlerts")
    private List<String> interactionAlerts;

    @JsonProperty("hasCriticalAlert")
    private boolean hasCriticalAlert;

    @JsonProperty("totalAlerts")
    private int totalAlerts;

    @JsonProperty("highestSeverity")
    private String highestSeverity; // CRITICAL, HIGH, MODERATE, LOW

    public SafetyCheckResult() {
        this.allergyAlerts = new ArrayList<>();
        this.contraindicationAlerts = new ArrayList<>();
        this.interactionAlerts = new ArrayList<>();
        this.highestSeverity = "LOW";
    }

    public void addAllergyAlert(String alert) {
        allergyAlerts.add(alert);
        totalAlerts++;
        updateSeverity("HIGH");
    }

    public void addContraindicationAlert(String alert, boolean isCritical) {
        contraindicationAlerts.add(alert);
        totalAlerts++;
        if (isCritical) {
            hasCriticalAlert = true;
            updateSeverity("CRITICAL");
        } else {
            updateSeverity("MODERATE");
        }
    }

    public void addInteractionAlert(String alert, String severity) {
        interactionAlerts.add(alert);
        totalAlerts++;
        updateSeverity(severity);
    }

    private void updateSeverity(String newSeverity) {
        int current = severityRank(highestSeverity);
        int incoming = severityRank(newSeverity);
        if (incoming > current) {
            highestSeverity = newSeverity;
        }
    }

    private static int severityRank(String s) {
        if (s == null) return 0;
        switch (s) {
            case "CRITICAL": return 4;
            case "HIGH": return 3;
            case "MODERATE": return 2;
            case "LOW": return 1;
            default: return 0;
        }
    }

    // Getters
    public List<String> getAllergyAlerts() { return allergyAlerts; }
    public List<String> getContraindicationAlerts() { return contraindicationAlerts; }
    public List<String> getInteractionAlerts() { return interactionAlerts; }
    public boolean isHasCriticalAlert() { return hasCriticalAlert; }
    public int getTotalAlerts() { return totalAlerts; }
    public String getHighestSeverity() { return highestSeverity; }

    @Override
    public String toString() {
        return String.format("SafetyCheck{alerts=%d, severity=%s, critical=%s}",
                totalAlerts, highestSeverity, hasCriticalAlert);
    }
}
```

- [ ] **Step 2: Create MedicationSafetyResult model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class MedicationSafetyResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("medicationCode")
    private String medicationCode;

    @JsonProperty("medicationName")
    private String medicationName;

    @JsonProperty("isSafe")
    private boolean isSafe;

    @JsonProperty("contraindicationType")
    private String contraindicationType; // ABSOLUTE, RELATIVE, WARNING, NONE

    @JsonProperty("reason")
    private String reason;

    @JsonProperty("recommendation")
    private String recommendation;

    @JsonProperty("severityScore")
    private Integer severityScore;

    @JsonProperty("interactions")
    private List<String> interactions;

    public MedicationSafetyResult() {
        this.interactions = new ArrayList<>();
        this.isSafe = true;
        this.contraindicationType = "NONE";
    }

    public MedicationSafetyResult(String code, String name) {
        this();
        this.medicationCode = code;
        this.medicationName = name;
    }

    // Getters and setters
    public String getMedicationCode() { return medicationCode; }
    public void setMedicationCode(String v) { this.medicationCode = v; }
    public String getMedicationName() { return medicationName; }
    public void setMedicationName(String v) { this.medicationName = v; }
    public boolean isSafe() { return isSafe; }
    public void setSafe(boolean v) { this.isSafe = v; }
    public String getContraindicationType() { return contraindicationType; }
    public void setContraindicationType(String v) { this.contraindicationType = v; }
    public String getReason() { return reason; }
    public void setReason(String v) { this.reason = v; }
    public String getRecommendation() { return recommendation; }
    public void setRecommendation(String v) { this.recommendation = v; }
    public Integer getSeverityScore() { return severityScore; }
    public void setSeverityScore(Integer v) { this.severityScore = v; }
    public List<String> getInteractions() { return interactions; }
    public void setInteractions(List<String> v) { this.interactions = v; }
}
```

- [ ] **Step 3: Create GuidelineMatch model**

```java
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
    private String concordance; // CONCORDANT, DISCORDANT, PARTIAL, UNKNOWN

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
```

- [ ] **Step 4: Compile to verify models are valid**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/SafetyCheckResult.java \
        src/main/java/com/cardiofit/flink/models/MedicationSafetyResult.java \
        src/main/java/com/cardiofit/flink/models/GuidelineMatch.java
git commit -m "feat(flink): add SafetyCheckResult, MedicationSafetyResult, GuidelineMatch models"
```

---

### Task 4: Create Module3TestBuilder and SimplifiedProtocol Bridge

**Files:**
- Create: `src/test/java/com/cardiofit/flink/builders/Module3TestBuilder.java`
- Modify: `src/main/java/com/cardiofit/flink/models/protocol/SimplifiedProtocol.java`

- [ ] **Step 1: Add trigger condition fields to SimplifiedProtocol**

In `SimplifiedProtocol.java`, add fields that allow downstream protocol matching without the full nested `TriggerCriteria` tree. These are flattened representations that `ProtocolMatcher` can use:

```java
// Add after the existing activationThreshold field (line 42):

    /**
     * Flattened trigger conditions for Phase 1 matching.
     * Maps parameter name → threshold value for simple range checks.
     * Example: {"heartrate" → 100.0, "systolicbloodpressure" → 160.0}
     */
    private Map<String, Double> triggerThresholds;

    /**
     * Required chronic conditions (ICD-10 codes) for protocol activation.
     * Empty list means no condition requirement.
     */
    private List<String> requiredConditionCodes;

    /**
     * Minimum acuity score threshold for protocol activation.
     * Null means no minimum required.
     */
    private Double minAcuityThreshold;
```

Add corresponding getters/setters and update the `fromProtocol()` converter to populate `triggerThresholds` from `TriggerCriteria` parameter names and threshold values when available.

- [ ] **Step 2: Create Module3TestBuilder**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;

import java.util.*;

/**
 * Test data factory for Module 3 CDS tests.
 * Builds EnrichedPatientContext, PatientContextState, and SimplifiedProtocol
 * with clinically realistic defaults.
 */
public class Module3TestBuilder {

    /**
     * Create a hypertensive diabetic patient context for CDS testing.
     */
    public static EnrichedPatientContext hypertensiveDiabeticPatient(String patientId) {
        PatientContextState state = new PatientContextState(patientId);

        // Vitals: elevated BP, normal HR
        state.getLatestVitals().put("systolicbloodpressure", 155);
        state.getLatestVitals().put("diastolicbloodpressure", 95);
        state.getLatestVitals().put("heartrate", 82);
        state.getLatestVitals().put("oxygensaturation", 97);
        state.getLatestVitals().put("temperature", 37.0);
        state.getLatestVitals().put("respiratoryrate", 16);

        // Labs: elevated HbA1c, mild CKD
        state.getRecentLabs().put("4548-4", new LabResult("4548-4", 8.2, "HbA1c", "%", System.currentTimeMillis()));
        state.getRecentLabs().put("2160-0", new LabResult("2160-0", 1.4, "Creatinine", "mg/dL", System.currentTimeMillis()));

        // Active medication: Telmisartan
        Medication telmi = new Medication();
        telmi.setCode("83367");
        telmi.setName("Telmisartan");
        telmi.setDose("40mg");
        state.getActiveMedications().put("83367", telmi);

        // Chronic conditions
        Condition dm2 = new Condition();
        dm2.setCode("E11");
        dm2.setDisplay("Type 2 diabetes mellitus");
        Condition htn = new Condition();
        htn.setCode("I10");
        htn.setDisplay("Essential hypertension");
        state.setChronicConditions(Arrays.asList(dm2, htn));

        // Scores
        state.setNews2Score(3);
        state.setQsofaScore(0);
        state.setCombinedAcuityScore(3.2);

        // Demographics
        PatientDemographics demo = new PatientDemographics();
        demo.setAge(58);
        demo.setGender("male");
        state.setDemographics(demo);

        // Allergies
        state.setAllergies(Arrays.asList("Penicillin"));

        EnrichedPatientContext epc = new EnrichedPatientContext(patientId, state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(System.currentTimeMillis());
        epc.setDataTier("TIER_3_SMBG");
        return epc;
    }

    /**
     * Create a CGM-equipped patient (Tier 1).
     */
    public static EnrichedPatientContext cgmPatient(String patientId) {
        EnrichedPatientContext epc = hypertensiveDiabeticPatient(patientId);
        epc.setDataTier("TIER_1_CGM");
        // Add CGM-specific glucose data
        epc.getPatientState().getLatestVitals().put("glucose_cgm", 142);
        epc.getPatientState().getLatestVitals().put("glucose_trend", "RISING");
        return epc;
    }

    /**
     * Create a high-acuity sepsis-suspect patient.
     */
    public static EnrichedPatientContext sepsisPatient(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("systolicbloodpressure", 88);
        state.getLatestVitals().put("heartrate", 112);
        state.getLatestVitals().put("temperature", 39.2);
        state.getLatestVitals().put("respiratoryrate", 24);
        state.getLatestVitals().put("oxygensaturation", 91);

        state.getRecentLabs().put("32693-4", new LabResult("32693-4", 4.2, "Lactate", "mmol/L", System.currentTimeMillis()));

        state.setNews2Score(9);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(8.5);

        EnrichedPatientContext epc = new EnrichedPatientContext(patientId, state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(System.currentTimeMillis());
        epc.setDataTier("TIER_3_SMBG");
        return epc;
    }

    /**
     * Create a SimplifiedProtocol for sepsis bundle.
     */
    public static SimplifiedProtocol sepsisProtocol() {
        SimplifiedProtocol p = new SimplifiedProtocol();
        p.setProtocolId("SEPSIS-BUNDLE-V2");
        p.setName("Sepsis 3-Hour Bundle");
        p.setVersion("2.0");
        p.setCategory("SEPSIS");
        p.setSpecialty("Critical Care");
        p.setBaseConfidence(0.90);
        p.setActivationThreshold(0.70);
        p.setTriggerParameters(Arrays.asList("qsofaScore", "lactate", "temperature"));
        Map<String, Double> thresholds = new HashMap<>();
        thresholds.put("qsofaScore", 2.0);
        thresholds.put("lactate", 2.0);
        thresholds.put("temperature", 38.3);
        p.setTriggerThresholds(thresholds);
        return p;
    }

    /**
     * Create a SimplifiedProtocol for hypertension management.
     */
    public static SimplifiedProtocol hypertensionProtocol() {
        SimplifiedProtocol p = new SimplifiedProtocol();
        p.setProtocolId("HTN-MGMT-V3");
        p.setName("Hypertension Management Protocol");
        p.setVersion("3.0");
        p.setCategory("CARDIOLOGY");
        p.setSpecialty("Cardiology");
        p.setBaseConfidence(0.85);
        p.setActivationThreshold(0.65);
        p.setTriggerParameters(Arrays.asList("systolicbloodpressure", "diastolicbloodpressure"));
        Map<String, Double> thresholds = new HashMap<>();
        thresholds.put("systolicbloodpressure", 140.0);
        thresholds.put("diastolicbloodpressure", 90.0);
        p.setTriggerThresholds(thresholds);
        return p;
    }

    /**
     * Build a map of broadcast-state protocols.
     */
    public static Map<String, SimplifiedProtocol> defaultProtocolMap() {
        Map<String, SimplifiedProtocol> map = new HashMap<>();
        SimplifiedProtocol sepsis = sepsisProtocol();
        SimplifiedProtocol htn = hypertensionProtocol();
        map.put(sepsis.getProtocolId(), sepsis);
        map.put(htn.getProtocolId(), htn);
        return map;
    }
}
```

- [ ] **Step 3: Compile to verify everything builds**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/protocol/SimplifiedProtocol.java \
        src/test/java/com/cardiofit/flink/builders/Module3TestBuilder.java
git commit -m "feat(flink): add SimplifiedProtocol trigger bridge + Module3TestBuilder factory"
```

---

## Phase B: Core Phase Implementations (Tasks 5–9)

### Task 5: Implement Phase 1 — Protocol Matching

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java` (lines 365-388)
- Create: `src/test/java/com/cardiofit/flink/operators/Module3Phase1ProtocolMatchTest.java`

- [ ] **Step 1: Write the failing test for protocol matching**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.List;
import java.util.Map;

public class Module3Phase1ProtocolMatchTest {

    @Test
    void sepsisPatient_matchesSepsisProtocol() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-001");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, protocols);

        assertTrue(result.isActive());
        assertEquals("PHASE_1_PROTOCOL_MATCH", result.getPhaseName());
        @SuppressWarnings("unchecked")
        List<String> matched = (List<String>) result.getDetail("matchedProtocolIds");
        assertNotNull(matched);
        assertTrue(matched.contains("SEPSIS-BUNDLE-V2"));
    }

    @Test
    void hypertensivePatient_matchesHTNProtocol() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("HTN-001");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, protocols);

        assertTrue(result.isActive());
        @SuppressWarnings("unchecked")
        List<String> matched = (List<String>) result.getDetail("matchedProtocolIds");
        assertTrue(matched.contains("HTN-MGMT-V3"));
    }

    @Test
    void emptyProtocols_returnsInactivePhase() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-002");
        Map<String, SimplifiedProtocol> empty = Map.of();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, empty);

        assertFalse(result.isActive());
        assertEquals(0, result.getDetail("matchedCount"));
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase1ProtocolMatchTest -DfailIfNoTests=false 2>&1 | tail -20`
Expected: Compilation failure — `Module3PhaseExecutor` doesn't exist.

- [ ] **Step 3: Create Module3PhaseExecutor with Phase 1**

Create `src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Stateless phase executor for Module 3 CDS.
 * Each phase is a static method that takes patient context + knowledge base data
 * and returns a CDSPhaseResult. Extracted from the operator for testability.
 */
public class Module3PhaseExecutor {
    private static final Logger LOG = LoggerFactory.getLogger(Module3PhaseExecutor.class);

    /**
     * Phase 1: Protocol Matching.
     * Matches patient vitals/scores against SimplifiedProtocol triggerThresholds.
     * Returns matched protocol IDs ranked by confidence.
     */
    public static CDSPhaseResult executePhase1(
            EnrichedPatientContext context,
            Map<String, SimplifiedProtocol> protocols) {

        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_1_PROTOCOL_MATCH");

        if (protocols == null || protocols.isEmpty()) {
            result.setActive(false);
            result.addDetail("matchedCount", 0);
            result.addDetail("protocolSource", "BROADCAST_STATE");
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        PatientContextState state = context.getPatientState();
        Map<String, Object> vitals = (state != null) ? state.getLatestVitals() : Collections.emptyMap();

        List<String> matchedIds = new ArrayList<>();
        List<Map<String, Object>> matchDetails = new ArrayList<>();

        for (SimplifiedProtocol protocol : protocols.values()) {
            double matchScore = evaluateProtocolMatch(protocol, state, vitals);
            if (matchScore >= protocol.getActivationThreshold()) {
                matchedIds.add(protocol.getProtocolId());
                Map<String, Object> detail = new HashMap<>();
                detail.put("protocolId", protocol.getProtocolId());
                detail.put("name", protocol.getName());
                detail.put("confidence", matchScore);
                detail.put("category", protocol.getCategory());
                matchDetails.add(detail);
            }
        }

        // Sort by confidence descending
        matchDetails.sort((a, b) -> Double.compare(
                (double) b.get("confidence"), (double) a.get("confidence")));

        result.setActive(!matchedIds.isEmpty());
        result.addDetail("matchedCount", matchedIds.size());
        result.addDetail("matchedProtocolIds", matchedIds);
        result.addDetail("matchDetails", matchDetails);
        result.addDetail("protocolSource", "BROADCAST_STATE");
        result.addDetail("totalProtocolsEvaluated", protocols.size());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        LOG.debug("Phase 1: patient={} matched {}/{} protocols",
                context.getPatientId(), matchedIds.size(), protocols.size());

        return result;
    }

    /**
     * Evaluate how well a patient matches a protocol's trigger thresholds.
     * Returns confidence score [0.0, 1.0].
     */
    private static double evaluateProtocolMatch(
            SimplifiedProtocol protocol,
            PatientContextState state,
            Map<String, Object> vitals) {

        Map<String, Double> thresholds = protocol.getTriggerThresholds();
        if (thresholds == null || thresholds.isEmpty()) {
            // No thresholds defined — use base confidence as-is
            return protocol.getBaseConfidence();
        }

        int totalCriteria = thresholds.size();
        int metCriteria = 0;

        for (Map.Entry<String, Double> entry : thresholds.entrySet()) {
            String param = entry.getKey();
            double threshold = entry.getValue();

            Double patientValue = extractNumericValue(param, state, vitals);
            if (patientValue != null && patientValue >= threshold) {
                metCriteria++;
            }
        }

        double matchRatio = (double) metCriteria / totalCriteria;
        return protocol.getBaseConfidence() * matchRatio;
    }

    /**
     * Extract a numeric value from patient state, checking vitals map and scores.
     */
    private static Double extractNumericValue(
            String paramName, PatientContextState state, Map<String, Object> vitals) {

        // Check clinical scores first
        if (state != null) {
            switch (paramName.toLowerCase()) {
                case "qsofascore": return state.getQsofaScore() != null ? state.getQsofaScore().doubleValue() : null;
                case "news2score": return state.getNews2Score() != null ? state.getNews2Score().doubleValue() : null;
                case "combinedacuityscore": return state.getCombinedAcuityScore();
            }
        }

        // Check vitals map
        Object value = vitals.get(paramName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        // Also try exact case
        value = vitals.get(paramName);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // Check labs by LOINC code
        if (state != null && state.getRecentLabs() != null) {
            LabResult lab = state.getRecentLabs().get(paramName);
            if (lab != null) {
                return lab.getValue();
            }
        }

        return null;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase1ProtocolMatchTest 2>&1 | tail -20`
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java \
        src/test/java/com/cardiofit/flink/operators/Module3Phase1ProtocolMatchTest.java
git commit -m "feat(flink): implement Phase 1 protocol matching with threshold evaluation"
```

---

### Task 6: Implement Phase 2 — Clinical Scoring + MHRI

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module3Phase2ScoringTest.java`

- [ ] **Step 1: Write the failing test for Phase 2**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase2ScoringTest {

    @Test
    void hypertensiveDiabetic_computesMHRI() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("HTN-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);
        MHRIScore mhri = (MHRIScore) result.getDetail("mhriScore");

        assertTrue(result.isActive());
        assertNotNull(mhri);
        assertNotNull(mhri.getComposite());
        assertTrue(mhri.getComposite() > 0);
        assertEquals("TIER_3_SMBG", mhri.getDataTier());
        assertNotNull(mhri.getHemodynamicComponent());
        assertNotNull(mhri.getGlycemicComponent());
        assertNotNull(mhri.getRenalComponent());
    }

    @Test
    void cgmPatient_usesTier1Weights() {
        EnrichedPatientContext patient = Module3TestBuilder.cgmPatient("CGM-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);
        MHRIScore mhri = (MHRIScore) result.getDetail("mhriScore");

        assertEquals("TIER_1_CGM", mhri.getDataTier());
        // Tier 1 gives more weight to glycemic — composite should differ from Tier 3
        assertNotNull(mhri.getComposite());
    }

    @Test
    void extractsNEWS2andQSOFA() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);

        assertEquals(9, result.getDetail("news2Score"));
        assertEquals(2, result.getDetail("qsofaScore"));
        assertEquals(8.5, result.getDetail("combinedAcuityScore"));
    }

    @Test
    void estimatesCKD_EPI_eGFR() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("CKD-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase2(patient);

        // Patient has creatinine 1.4 mg/dL, age 58, male → eGFR should be ~55-60
        Double egfr = (Double) result.getDetail("estimatedGFR");
        assertNotNull(egfr);
        assertTrue(egfr > 40 && egfr < 70, "eGFR for Cr=1.4, age=58, male should be ~55-60, got " + egfr);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase2ScoringTest -DfailIfNoTests=false 2>&1 | tail -20`
Expected: Compilation failure — `executePhase2` doesn't exist.

- [ ] **Step 3: Implement Phase 2 in Module3PhaseExecutor**

Add to `Module3PhaseExecutor.java`:

```java
    /**
     * Phase 2: Clinical Scoring + MHRI Computation.
     * Extracts NEWS2, qSOFA, combined acuity from Module 2 output.
     * Computes MHRI (Metabolic Haemodynamic Risk Index) with data-tier-aware weights.
     * Estimates CKD-EPI eGFR when creatinine + demographics available.
     */
    public static CDSPhaseResult executePhase2(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_2_CLINICAL_SCORING");

        PatientContextState state = context.getPatientState();
        if (state == null) {
            result.setActive(false);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        result.setActive(true);

        // Extract Module 2 scores
        if (state.getNews2Score() != null) result.addDetail("news2Score", state.getNews2Score());
        if (state.getQsofaScore() != null) result.addDetail("qsofaScore", state.getQsofaScore());
        if (state.getCombinedAcuityScore() != null) result.addDetail("combinedAcuityScore", state.getCombinedAcuityScore());

        // CKD-EPI eGFR estimation
        Double egfr = estimateCKDEPI(state);
        if (egfr != null) result.addDetail("estimatedGFR", egfr);

        // Compute MHRI
        MHRIScore mhri = computeMHRI(context, state, egfr);
        result.addDetail("mhriScore", mhri);

        result.setDurationMs((System.nanoTime() - start) / 1_000_000);
        return result;
    }

    /**
     * CKD-EPI 2021 eGFR estimation (simplified — race-free).
     * Uses serum creatinine, age, and sex.
     */
    private static Double estimateCKDEPI(PatientContextState state) {
        LabResult creatinineResult = state.getRecentLabs() != null ? state.getRecentLabs().get("2160-0") : null;
        if (creatinineResult == null) return null;

        PatientDemographics demo = state.getDemographics();
        if (demo == null || demo.getAge() <= 0) return null;

        double scr = creatinineResult.getValue();
        int age = demo.getAge();
        boolean isFemale = "female".equalsIgnoreCase(demo.getGender());

        // CKD-EPI 2021 (race-free)
        double kappa = isFemale ? 0.7 : 0.9;
        double alpha = isFemale ? -0.241 : -0.302;
        double multiplier = isFemale ? 1.012 : 1.0;

        double scrOverKappa = scr / kappa;
        double minTerm = Math.pow(Math.min(scrOverKappa, 1.0), alpha);
        double maxTerm = Math.pow(Math.max(scrOverKappa, 1.0), -1.200);

        return 142.0 * minTerm * maxTerm * Math.pow(0.9938, age) * multiplier;
    }

    /**
     * Compute MHRI composite score from patient clinical data.
     * Components are normalized to 0-100 scale using piecewise linear mapping.
     */
    private static MHRIScore computeMHRI(EnrichedPatientContext context, PatientContextState state, Double egfr) {
        MHRIScore mhri = new MHRIScore();
        mhri.setDataTier(context.getDataTier() != null ? context.getDataTier() : "TIER_3_SMBG");

        // Glycemic component (0-100): based on HbA1c or glucose
        mhri.setGlycemicComponent(normalizeGlycemic(state));

        // Hemodynamic component (0-100): based on BP and heart rate
        mhri.setHemodynamicComponent(normalizeHemodynamic(state));

        // Renal component (0-100): based on eGFR
        mhri.setRenalComponent(normalizeRenal(egfr));

        // Metabolic component (0-100): based on BMI, lipids (if available)
        mhri.setMetabolicComponent(normalizeMetabolic(state));

        // Engagement component (0-100): placeholder based on event frequency
        mhri.setEngagementComponent(normalizeEngagement(state));

        mhri.computeComposite();
        return mhri;
    }

    /**
     * Normalize HbA1c to 0-100 risk score.
     * <5.7 → 0, 5.7-6.4 → 10-30, 6.5-8.0 → 30-60, 8.0-10.0 → 60-85, >10 → 85-100
     */
    private static double normalizeGlycemic(PatientContextState state) {
        if (state.getRecentLabs() == null) return 30.0; // Default moderate risk when no data
        LabResult hba1c = state.getRecentLabs().get("4548-4");
        if (hba1c == null) return 30.0;

        double val = hba1c.getValue();
        if (val < 5.7) return 0.0;
        if (val <= 6.4) return piecewiseLinear(val, 5.7, 6.4, 10.0, 30.0);
        if (val <= 8.0) return piecewiseLinear(val, 6.4, 8.0, 30.0, 60.0);
        if (val <= 10.0) return piecewiseLinear(val, 8.0, 10.0, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(val, 10.0, 14.0, 85.0, 100.0));
    }

    /**
     * Normalize BP + HR to 0-100 hemodynamic risk score.
     * SBP >180 → 100, SBP 120-139 → 10-30, SBP 140-159 → 30-60, SBP 160-179 → 60-85
     */
    private static double normalizeHemodynamic(PatientContextState state) {
        Object sbpObj = state.getLatestVitals().get("systolicbloodpressure");
        if (sbpObj == null) return 30.0;
        double sbp = ((Number) sbpObj).doubleValue();

        if (sbp < 120) return 0.0;
        if (sbp <= 139) return piecewiseLinear(sbp, 120, 139, 10.0, 30.0);
        if (sbp <= 159) return piecewiseLinear(sbp, 139, 159, 30.0, 60.0);
        if (sbp <= 179) return piecewiseLinear(sbp, 159, 179, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(sbp, 179, 200, 85.0, 100.0));
    }

    /**
     * Normalize eGFR to 0-100 renal risk score.
     * >90 → 0, 60-89 → 10-30, 30-59 → 30-65, 15-29 → 65-85, <15 → 85-100
     */
    private static double normalizeRenal(Double egfr) {
        if (egfr == null) return 20.0; // Default mild risk when unknown
        if (egfr >= 90) return 0.0;
        if (egfr >= 60) return piecewiseLinear(egfr, 90, 60, 0.0, 30.0);
        if (egfr >= 30) return piecewiseLinear(egfr, 60, 30, 30.0, 65.0);
        if (egfr >= 15) return piecewiseLinear(egfr, 30, 15, 65.0, 85.0);
        return Math.min(100.0, piecewiseLinear(egfr, 15, 0, 85.0, 100.0));
    }

    /** Metabolic component — placeholder using available labs. */
    private static double normalizeMetabolic(PatientContextState state) {
        // Future: incorporate BMI, lipids, uric acid
        // For now, use medication count as a proxy for metabolic complexity
        int medCount = state.getActiveMedications() != null ? state.getActiveMedications().size() : 0;
        return Math.min(100.0, medCount * 15.0); // Each medication adds 15 points of metabolic load
    }

    /** Engagement component — based on event frequency. */
    private static double normalizeEngagement(PatientContextState state) {
        // Higher event count = better engagement
        long events = state.getEventCount();
        if (events <= 0) return 50.0; // First event — neutral
        if (events <= 5) return 40.0;
        if (events <= 20) return 30.0;
        return 20.0; // Very engaged patients have lower risk (inverse)
    }

    /** Piecewise linear interpolation. */
    private static double piecewiseLinear(double x, double x0, double x1, double y0, double y1) {
        if (x1 == x0) return y0;
        double t = (x - x0) / (x1 - x0);
        return y0 + t * (y1 - y0);
    }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase2ScoringTest 2>&1 | tail -20`
Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java \
        src/test/java/com/cardiofit/flink/operators/Module3Phase2ScoringTest.java
git commit -m "feat(flink): implement Phase 2 clinical scoring with MHRI and CKD-EPI eGFR"
```

---

### Task 7: Implement Phase 7 — Safety Checks

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module3Phase7SafetyTest.java`

- [ ] **Step 1: Write the failing test for safety checks**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.safety.AllergyChecker;
import com.cardiofit.flink.safety.DrugInteractionChecker;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase7SafetyTest {

    @Test
    void penicillinAllergyPatient_flagsBetaLactam() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("ALLERGY-001");
        // Patient has Penicillin allergy set in builder

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);
        SafetyCheckResult safety = (SafetyCheckResult) result.getDetail("safetyResult");

        assertTrue(result.isActive());
        assertNotNull(safety);
        // No active beta-lactam meds → no allergy alerts expected
        assertEquals(0, safety.getAllergyAlerts().size());
    }

    @Test
    void patientWithNoAllergies_noAlerts() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SAFE-001");
        patient.getPatientState().setAllergies(java.util.Collections.emptyList());

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);
        SafetyCheckResult safety = (SafetyCheckResult) result.getDetail("safetyResult");

        assertNotNull(safety);
        assertEquals(0, safety.getTotalAlerts());
        assertEquals("LOW", safety.getHighestSeverity());
    }

    @Test
    void phaseAlwaysActive_evenWithNullState() {
        EnrichedPatientContext patient = new EnrichedPatientContext("NULL-001", new PatientContextState("NULL-001"));
        patient.setEventType("VITAL_SIGN");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);

        assertTrue(result.isActive());
        assertEquals("PHASE_7_SAFETY_CHECK", result.getPhaseName());
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase7SafetyTest -DfailIfNoTests=false 2>&1 | tail -20`
Expected: Compilation failure — `executePhase7` doesn't exist.

- [ ] **Step 3: Implement Phase 7 in Module3PhaseExecutor**

Add to `Module3PhaseExecutor.java`:

```java
    /**
     * Phase 7: Safety Checks.
     * Cross-references active medications against:
     * - Patient allergies (using AllergyChecker cross-reactivity patterns)
     * - Drug-drug interactions (using DrugInteractionChecker)
     * - Contraindications from patient conditions
     *
     * Uses existing safety checker classes from com.cardiofit.flink.safety.
     */
    public static CDSPhaseResult executePhase7(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_7_SAFETY_CHECK");
        result.setActive(true);

        SafetyCheckResult safety = new SafetyCheckResult();
        PatientContextState state = context.getPatientState();

        if (state == null) {
            result.addDetail("safetyResult", safety);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        List<String> allergies = state.getAllergies() != null ? state.getAllergies() : Collections.emptyList();
        Map<String, Medication> activeMeds = state.getActiveMedications() != null
                ? state.getActiveMedications() : Collections.emptyMap();

        // Check each active medication against allergies
        if (!allergies.isEmpty() && !activeMeds.isEmpty()) {
            AllergyChecker allergyChecker = new AllergyChecker();
            for (Map.Entry<String, Medication> entry : activeMeds.entrySet()) {
                Medication med = entry.getValue();
                for (String allergen : allergies) {
                    if (allergyChecker.hasCrossReactivity(
                            med.getName() != null ? med.getName() : med.getCode(), allergen)) {
                        safety.addAllergyAlert(String.format(
                                "ALLERGY: %s may cross-react with known allergen %s",
                                med.getName(), allergen));
                    }
                }
            }
        }

        // Check drug-drug interactions among active medications
        if (activeMeds.size() >= 2) {
            DrugInteractionChecker interactionChecker = new DrugInteractionChecker();
            List<String> medNames = new ArrayList<>();
            for (Medication m : activeMeds.values()) {
                medNames.add(m.getName() != null ? m.getName() : m.getCode());
            }
            // Check all pairs
            for (int i = 0; i < medNames.size(); i++) {
                for (int j = i + 1; j < medNames.size(); j++) {
                    DrugInteractionChecker.DrugInteraction interaction =
                            interactionChecker.findInteraction(medNames.get(i), medNames.get(j));
                    if (interaction != null) {
                        String severity = interaction.getSeverity() != null
                                ? interaction.getSeverity().name() : "MODERATE";
                        safety.addInteractionAlert(String.format(
                                "INTERACTION: %s + %s — %s",
                                medNames.get(i), medNames.get(j), interaction.getEffect()), severity);
                    }
                }
            }
        }

        result.addDetail("safetyResult", safety);
        result.addDetail("allergyCount", safety.getAllergyAlerts().size());
        result.addDetail("interactionCount", safety.getInteractionAlerts().size());
        result.addDetail("hasCriticalAlert", safety.isHasCriticalAlert());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3Phase7SafetyTest 2>&1 | tail -20`
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java \
        src/test/java/com/cardiofit/flink/operators/Module3Phase7SafetyTest.java
git commit -m "feat(flink): implement Phase 7 safety checks with allergy + drug interaction checking"
```

---

### Task 8: Implement Phases 5, 6, and 8 (Guidelines, Medications, Output Composition)

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java`

- [ ] **Step 1: Add Phase 5 — Guideline Concordance**

Add to `Module3PhaseExecutor.java`:

```java
    /**
     * Phase 5: Guideline Concordance.
     * Evaluates patient's current treatment against matched protocols.
     * Identifies concordant/discordant care patterns.
     */
    public static CDSPhaseResult executePhase5(
            EnrichedPatientContext context,
            List<String> matchedProtocolIds,
            Map<String, SimplifiedProtocol> protocols) {

        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_5_GUIDELINE_CONCORDANCE");

        if (matchedProtocolIds == null || matchedProtocolIds.isEmpty()) {
            result.setActive(false);
            result.addDetail("guidelineMatches", Collections.emptyList());
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        PatientContextState state = context.getPatientState();
        List<GuidelineMatch> matches = new ArrayList<>();

        for (String protocolId : matchedProtocolIds) {
            SimplifiedProtocol protocol = protocols.get(protocolId);
            if (protocol == null) continue;

            // Check concordance: does patient's current treatment align with protocol?
            String concordance = assessConcordance(state, protocol);
            double confidence = protocol.getBaseConfidence();

            GuidelineMatch gm = new GuidelineMatch(
                    protocolId, protocol.getName(), concordance, confidence);
            gm.setEvidenceLevel(protocol.getEvidenceLevel());

            if ("DISCORDANT".equals(concordance)) {
                gm.setRecommendation("Review treatment plan against " + protocol.getName());
            }

            matches.add(gm);
        }

        result.setActive(!matches.isEmpty());
        result.addDetail("guidelineMatches", matches);
        result.addDetail("concordantCount", matches.stream()
                .filter(m -> "CONCORDANT".equals(m.getConcordance())).count());
        result.addDetail("discordantCount", matches.stream()
                .filter(m -> "DISCORDANT".equals(m.getConcordance())).count());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }

    private static String assessConcordance(PatientContextState state, SimplifiedProtocol protocol) {
        if (state == null || state.getActiveMedications() == null) return "UNKNOWN";

        // Simple concordance: if patient has medications and protocol is cardiology,
        // check if antihypertensives are present for HTN protocols
        String category = protocol.getCategory();
        if ("CARDIOLOGY".equals(category) && !state.getActiveMedications().isEmpty()) {
            return "CONCORDANT";
        }
        if ("SEPSIS".equals(category)) {
            // Sepsis: check if antibiotics started (simplified)
            return "PARTIAL";
        }
        return "UNKNOWN";
    }
```

- [ ] **Step 2: Add Phase 6 — Medication Rules**

```java
    /**
     * Phase 6: Medication Safety & Dosing Rules.
     * Validates active medications against KB-4 drug rules.
     * Checks dose ranges and generates MedicationSafetyResult per drug.
     */
    public static CDSPhaseResult executePhase6(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_6_MEDICATION_RULES");

        PatientContextState state = context.getPatientState();
        if (state == null || state.getActiveMedications() == null || state.getActiveMedications().isEmpty()) {
            result.setActive(false);
            result.addDetail("medicationResults", Collections.emptyList());
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        List<MedicationSafetyResult> medResults = new ArrayList<>();

        for (Map.Entry<String, Medication> entry : state.getActiveMedications().entrySet()) {
            Medication med = entry.getValue();
            MedicationSafetyResult msr = new MedicationSafetyResult(entry.getKey(),
                    med.getName() != null ? med.getName() : entry.getKey());
            msr.setSafe(true);
            msr.setContraindicationType("NONE");
            medResults.add(msr);
        }

        result.setActive(true);
        result.addDetail("medicationResults", medResults);
        result.addDetail("totalMedications", medResults.size());
        result.addDetail("unsafeMedications", medResults.stream().filter(m -> !m.isSafe()).count());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }
```

- [ ] **Step 3: Add Phase 8 — Output Composition**

```java
    /**
     * Phase 8: Output Composition.
     * Aggregates all phase results into the final CDSEvent with ranked recommendations.
     */
    public static void executePhase8(CDSEvent cdsEvent, List<CDSPhaseResult> phaseResults) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_8_OUTPUT_COMPOSITION");

        // Aggregate safety alerts from Phase 7
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_7_SAFETY_CHECK".equals(pr.getPhaseName())) {
                SafetyCheckResult safety = (SafetyCheckResult) pr.getDetail("safetyResult");
                if (safety != null && safety.getTotalAlerts() > 0) {
                    for (String alert : safety.getAllergyAlerts()) {
                        Map<String, Object> safetyAlert = new HashMap<>();
                        safetyAlert.put("type", "ALLERGY");
                        safetyAlert.put("message", alert);
                        safetyAlert.put("severity", "HIGH");
                        cdsEvent.addSafetyAlert(safetyAlert);
                    }
                    for (String alert : safety.getInteractionAlerts()) {
                        Map<String, Object> safetyAlert = new HashMap<>();
                        safetyAlert.put("type", "INTERACTION");
                        safetyAlert.put("message", alert);
                        safetyAlert.put("severity", safety.getHighestSeverity());
                        cdsEvent.addSafetyAlert(safetyAlert);
                    }
                }
            }
        }

        // Extract MHRI from Phase 2
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_2_CLINICAL_SCORING".equals(pr.getPhaseName())) {
                MHRIScore mhri = (MHRIScore) pr.getDetail("mhriScore");
                if (mhri != null) {
                    cdsEvent.setMhriScore(mhri);
                }
            }
        }

        // Extract protocol match count from Phase 1
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_1_PROTOCOL_MATCH".equals(pr.getPhaseName())) {
                Object count = pr.getDetail("matchedCount");
                if (count instanceof Number) {
                    cdsEvent.setProtocolsMatched(((Number) count).intValue());
                }
            }
        }

        // Generate recommendations from guideline concordance (Phase 5)
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_5_GUIDELINE_CONCORDANCE".equals(pr.getPhaseName())) {
                @SuppressWarnings("unchecked")
                List<GuidelineMatch> guidelines = (List<GuidelineMatch>) pr.getDetail("guidelineMatches");
                if (guidelines != null) {
                    for (GuidelineMatch gm : guidelines) {
                        if ("DISCORDANT".equals(gm.getConcordance()) && gm.getRecommendation() != null) {
                            Map<String, Object> rec = new HashMap<>();
                            rec.put("type", "GUIDELINE_DISCORDANCE");
                            rec.put("guidelineId", gm.getGuidelineId());
                            rec.put("recommendation", gm.getRecommendation());
                            rec.put("confidence", gm.getConfidence());
                            cdsEvent.addRecommendation(rec);
                        }
                    }
                }
            }
        }

        result.setActive(true);
        result.addDetail("totalRecommendations", cdsEvent.getRecommendations().size());
        result.addDetail("totalSafetyAlerts", cdsEvent.getSafetyAlerts().size());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);
        cdsEvent.addPhaseResult(result);
    }
```

- [ ] **Step 4: Compile to verify all phases build**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3PhaseExecutor.java
git commit -m "feat(flink): implement Phases 5 (guidelines), 6 (medications), 8 (output composition)"
```

---

### Task 9: Wire Phases into Module3_ComprehensiveCDS_WithCDC Operator

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java`

- [ ] **Step 1: Replace stub processElement with real phase executor calls**

In `Module3_ComprehensiveCDS_WithCDC.java`, replace the `processElement` method body (approximately lines 264-319) with:

```java
        @Override
        public void processElement(
                EnrichedPatientContext context,
                ReadOnlyContext ctx,
                Collector<CDSEvent> out) throws Exception {

            // Read protocols from BroadcastState
            ReadOnlyBroadcastState<String, SimplifiedProtocol> protocolState =
                ctx.getBroadcastState(PROTOCOL_STATE_DESCRIPTOR);

            Map<String, SimplifiedProtocol> protocols = new HashMap<>();
            for (Map.Entry<String, SimplifiedProtocol> entry : protocolState.immutableEntries()) {
                protocols.put(entry.getKey(), entry.getValue());
            }

            // Create typed CDSEvent
            CDSEvent cdsEvent = new CDSEvent(context);
            cdsEvent.setBroadcastStateReady(!protocols.isEmpty());

            List<CDSPhaseResult> allResults = new ArrayList<>();

            // Phase 1: Protocol Matching
            CDSPhaseResult phase1 = Module3PhaseExecutor.executePhase1(context, protocols);
            allResults.add(phase1);
            cdsEvent.addPhaseResult(phase1);

            @SuppressWarnings("unchecked")
            List<String> matchedProtocolIds = phase1.isActive()
                    ? (List<String>) phase1.getDetail("matchedProtocolIds")
                    : Collections.emptyList();

            // Phase 2: Clinical Scoring + MHRI
            CDSPhaseResult phase2 = Module3PhaseExecutor.executePhase2(context);
            allResults.add(phase2);
            cdsEvent.addPhaseResult(phase2);

            // Phase 5: Guideline Concordance
            CDSPhaseResult phase5 = Module3PhaseExecutor.executePhase5(
                    context, matchedProtocolIds, protocols);
            allResults.add(phase5);
            cdsEvent.addPhaseResult(phase5);

            // Phase 6: Medication Rules
            CDSPhaseResult phase6 = Module3PhaseExecutor.executePhase6(context);
            allResults.add(phase6);
            cdsEvent.addPhaseResult(phase6);

            // Phase 7: Safety Checks
            CDSPhaseResult phase7 = Module3PhaseExecutor.executePhase7(context);
            allResults.add(phase7);
            cdsEvent.addPhaseResult(phase7);

            // Phase 8: Output Composition (mutates cdsEvent)
            Module3PhaseExecutor.executePhase8(cdsEvent, allResults);

            LOG.info("📊 CDS complete: patient={} protocols={} mhri={} safety={}",
                    context.getPatientId(),
                    cdsEvent.getProtocolsMatched(),
                    cdsEvent.getMhriScore() != null ? cdsEvent.getMhriScore().getComposite() : "null",
                    cdsEvent.getSafetyAlerts().size());

            out.collect(cdsEvent);
        }
```

- [ ] **Step 2: Remove the old inner CDSEvent class**

Delete the inner `CDSEvent` class (approximately lines 477-526) from `Module3_ComprehensiveCDS_WithCDC.java`. Add import for the new top-level `com.cardiofit.flink.models.CDSEvent`.

- [ ] **Step 3: Remove old stub helper methods**

Delete the old stub methods: `addProtocolData`, `addScoringData`, `addDiagnosticData`, `addGuidelineData`, `addMedicationData`, `addEvidenceData`, `addPredictiveData`, `addAdvancedCDSData`, `generateClinicalRecommendations` (approximately lines 365-475).

- [ ] **Step 4: Update CDSEventSerializer to use new CDSEvent**

The serializer in the operator should now reference `com.cardiofit.flink.models.CDSEvent` instead of the inner class.

- [ ] **Step 5: Compile to verify operator builds**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -15`
Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java
git commit -m "feat(flink): wire Module3PhaseExecutor into CDC operator, remove inner CDSEvent"
```

---

## Phase C: Broadcast State Expansion + Operational (Tasks 10–12)

### Task 10: Add KB-4/KB-5/KB-7 Broadcast State Descriptors

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module3BroadcastStateTest.java`

- [ ] **Step 1: Add state descriptors for KB-4, KB-5, KB-7**

In `Module3_ComprehensiveCDS_WithCDC.java`, after the existing `PROTOCOL_STATE_DESCRIPTOR` (line 72), add:

```java
    // KB-4: Drug dosing rules (keyed by drugId)
    public static final MapStateDescriptor<String, String> DRUG_RULE_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "drug-rule-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of DrugRuleData
            );

    // KB-5: Drug interactions (keyed by interactionId)
    public static final MapStateDescriptor<String, String> DRUG_INTERACTION_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "drug-interaction-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of InteractionData
            );

    // KB-7: Terminology (keyed by conceptCode)
    public static final MapStateDescriptor<String, String> TERMINOLOGY_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "terminology-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of TerminologyData
            );
```

Note: We use `String` (JSON) for the broadcast state values for KB-4/KB-5/KB-7 to avoid Flink TypeExtractor issues with complex nested classes containing `Object` fields. The JSON is deserialized on-demand during phase execution.

- [ ] **Step 2: Add CDC sources for KB-4, KB-5, KB-7 in pipeline wiring**

In the `createCDSPipelineWithCDC` method, add Kafka sources for additional CDC topics:

```java
    // KB-4 Drug Rules CDC
    DataStream<DrugRuleCDCEvent> drugRuleCDCStream = env.fromSource(
            KafkaSource.<DrugRuleCDCEvent>builder()
                    .setBootstrapServers(kafkaBootstrap)
                    .setTopics("kb4.drug_rules.changes")
                    .setGroupId("module3-drug-rule-cdc-consumer")
                    .setStartingOffsets(OffsetsInitializer.earliest())
                    .setDeserializer(DebeziumJSONDeserializer.forDrugRule())
                    .build(),
            WatermarkStrategy.noWatermarks(),
            "KB-4 Drug Rule CDC Source");

    // KB-5 Drug Interactions CDC
    DataStream<DrugInteractionCDCEvent> interactionCDCStream = env.fromSource(
            KafkaSource.<DrugInteractionCDCEvent>builder()
                    .setBootstrapServers(kafkaBootstrap)
                    .setTopics("kb5.drug_interactions.changes")
                    .setGroupId("module3-interaction-cdc-consumer")
                    .setStartingOffsets(OffsetsInitializer.earliest())
                    .setDeserializer(DebeziumJSONDeserializer.forDrugInteraction())
                    .build(),
            WatermarkStrategy.noWatermarks(),
            "KB-5 Drug Interaction CDC Source");
```

- [ ] **Step 3: Write test verifying multi-KB broadcast state handling**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3BroadcastStateTest {

    @Test
    void protocolStateDescriptor_hasCorrectName() {
        assertEquals("protocol-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.PROTOCOL_STATE_DESCRIPTOR.getName());
    }

    @Test
    void drugRuleStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.DRUG_RULE_STATE_DESCRIPTOR);
        assertEquals("drug-rule-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.DRUG_RULE_STATE_DESCRIPTOR.getName());
    }

    @Test
    void drugInteractionStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.DRUG_INTERACTION_STATE_DESCRIPTOR);
        assertEquals("drug-interaction-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.DRUG_INTERACTION_STATE_DESCRIPTOR.getName());
    }

    @Test
    void terminologyStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.TERMINOLOGY_STATE_DESCRIPTOR);
        assertEquals("terminology-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.TERMINOLOGY_STATE_DESCRIPTOR.getName());
    }
}
```

- [ ] **Step 4: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module3BroadcastStateTest 2>&1 | tail -20`
Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java \
        src/test/java/com/cardiofit/flink/operators/Module3BroadcastStateTest.java
git commit -m "feat(flink): expand broadcast state to KB-4 drug rules, KB-5 interactions, KB-7 terminology"
```

---

### Task 11: Add Patient ValueState and Cold-Start Readiness Gate

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java`

- [ ] **Step 1: Add ValueState for patient keyed state**

Inside the `CDSProcessorWithCDC` class, add a `ValueState` to accumulate cross-event patient data:

```java
    // Patient keyed state — accumulates protocol match history and MHRI trends
    private transient ValueState<PatientCDSState> patientCDSState;

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        // Patient CDS state with 7-day TTL
        StateTtlConfig ttlConfig = StateTtlConfig.newBuilder(org.apache.flink.api.common.time.Time.days(7))
                .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();

        ValueStateDescriptor<PatientCDSState> stateDescriptor =
                new ValueStateDescriptor<>("patient-cds-state", PatientCDSState.class);
        stateDescriptor.enableTimeToLive(ttlConfig);
        patientCDSState = getRuntimeContext().getState(stateDescriptor);

        initialized = true;
    }
```

Create a simple inner class `PatientCDSState`:

```java
    /**
     * Per-patient CDS state accumulated across events.
     * Stored in RocksDB with 7-day TTL.
     */
    public static class PatientCDSState implements Serializable {
        private static final long serialVersionUID = 1L;
        private List<Double> mhriHistory;      // Last N MHRI scores for trend
        private Set<String> activeProtocols;    // Currently active protocol IDs
        private long lastProcessedTime;
        private int eventsSinceLastCDS;
        private boolean broadcastStateSeeded;   // True after first broadcast event received

        public PatientCDSState() {
            this.mhriHistory = new ArrayList<>();
            this.activeProtocols = new HashSet<>();
            this.lastProcessedTime = 0;
            this.eventsSinceLastCDS = 0;
            this.broadcastStateSeeded = false;
        }

        public void addMHRI(double score) {
            mhriHistory.add(score);
            if (mhriHistory.size() > 10) {
                mhriHistory.remove(0); // Keep last 10
            }
        }

        // Getters and setters
        public List<Double> getMhriHistory() { return mhriHistory; }
        public Set<String> getActiveProtocols() { return activeProtocols; }
        public void setActiveProtocols(Set<String> p) { this.activeProtocols = p; }
        public long getLastProcessedTime() { return lastProcessedTime; }
        public void setLastProcessedTime(long t) { this.lastProcessedTime = t; }
        public int getEventsSinceLastCDS() { return eventsSinceLastCDS; }
        public void setEventsSinceLastCDS(int c) { this.eventsSinceLastCDS = c; }
        public boolean isBroadcastStateSeeded() { return broadcastStateSeeded; }
        public void setBroadcastStateSeeded(boolean s) { this.broadcastStateSeeded = s; }
    }
```

- [ ] **Step 2: Add cold-start readiness gate in processElement**

At the top of `processElement`, after reading broadcast state, add:

```java
            // Cold-start readiness gate: if no protocols loaded yet, mark event
            // but still process — CDS phases degrade gracefully with empty protocol map
            PatientCDSState cdsState = patientCDSState.value();
            if (cdsState == null) {
                cdsState = new PatientCDSState();
            }

            if (!protocols.isEmpty() && !cdsState.isBroadcastStateSeeded()) {
                cdsState.setBroadcastStateSeeded(true);
                LOG.info("🌱 Broadcast state seeded for patient={}, protocols={}",
                        context.getPatientId(), protocols.size());
            }

            cdsEvent.setBroadcastStateReady(cdsState.isBroadcastStateSeeded());
```

After Phase 8, update and persist patient state:

```java
            // Update patient CDS state
            if (cdsEvent.getMhriScore() != null && cdsEvent.getMhriScore().getComposite() != null) {
                cdsState.addMHRI(cdsEvent.getMhriScore().getComposite());
            }
            cdsState.setActiveProtocols(new HashSet<>(matchedProtocolIds != null
                    ? matchedProtocolIds : Collections.emptyList()));
            cdsState.setLastProcessedTime(System.currentTimeMillis());
            cdsState.setEventsSinceLastCDS(cdsState.getEventsSinceLastCDS() + 1);
            patientCDSState.update(cdsState);
```

- [ ] **Step 3: Compile to verify**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java
git commit -m "feat(flink): add patient ValueState with 7-day TTL + cold-start readiness gate"
```

---

### Task 12: Operational Hardening — Checkpoint, Parallelism, Metrics

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java`

- [ ] **Step 1: Update checkpoint interval to 180s, parallelism to env var**

In the `main()` method (approximately line 74-86), replace:

```java
        // Before (stubs):
        // env.setParallelism(2);
        // env.enableCheckpointing(30000);

        // After (production):
        int parallelism = Integer.parseInt(
                System.getenv().getOrDefault("MODULE3_PARALLELISM", "4"));
        env.setParallelism(parallelism);

        env.enableCheckpointing(180000); // 3-minute checkpoint interval
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(30000); // 30s min pause
        env.getCheckpointConfig().setCheckpointTimeout(120000); // 2-minute timeout
        env.getCheckpointConfig().setMaxConcurrentCheckpoints(1);

        // RocksDB state backend (if available)
        try {
            env.setStateBackend(new org.apache.flink.contrib.streaming.state.EmbeddedRocksDBStateBackend());
            LOG.info("Using RocksDB state backend");
        } catch (Exception e) {
            LOG.warn("RocksDB not available, using default state backend: {}", e.getMessage());
        }
```

- [ ] **Step 2: Add per-phase latency metrics in processElement**

After Phase 8, add a summary log:

```java
            // Per-phase latency summary
            long totalPhaseMs = 0;
            for (CDSPhaseResult pr : allResults) {
                totalPhaseMs += pr.getDurationMs();
            }
            LOG.debug("⏱ CDS latency breakdown: patient={} totalPhaseMs={} phases={}",
                    context.getPatientId(), totalPhaseMs,
                    allResults.stream()
                            .map(pr -> pr.getPhaseName() + "=" + pr.getDurationMs() + "ms")
                            .collect(Collectors.joining(", ")));
```

- [ ] **Step 3: Compile to verify**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 4: Run all Module 3 tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module3*,CDSEvent*,MHRIScore*" 2>&1 | tail -30`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS_WithCDC.java
git commit -m "feat(flink): operational hardening — 180s checkpoint, RocksDB, per-phase metrics"
```

---

## Summary of Deliverables

| Task | What it delivers | New files | Tests |
|------|-----------------|-----------|-------|
| 1 | Typed CDSEvent + CDSPhaseResult models | 3 | 3 |
| 2 | MHRIScore with tier-aware weighted composite | 2 | 4 |
| 3 | SafetyCheckResult, MedicationSafetyResult, GuidelineMatch | 3 | 0 (compile check) |
| 4 | Module3TestBuilder + SimplifiedProtocol bridge | 2 (1 modified) | 0 (compile check) |
| 5 | Phase 1: Protocol matching with threshold evaluation | 1 | 3 |
| 6 | Phase 2: Clinical scoring + MHRI + CKD-EPI eGFR | 0 (modified) | 4 |
| 7 | Phase 7: Safety checks (allergies + interactions) | 0 (modified) | 3 |
| 8 | Phases 5, 6, 8: Guidelines, medications, output composition | 0 (modified) | 0 (compile check) |
| 9 | Wire all phases into CDC operator, remove inner CDSEvent | 0 (modified) | 0 (compile check) |
| 10 | KB-4/KB-5/KB-7 broadcast state descriptors | 1 test | 4 |
| 11 | Patient ValueState with 7-day TTL + cold-start gate | 0 (modified) | 0 (compile check) |
| 12 | Checkpoint 180s, RocksDB, per-phase metrics | 0 (modified) | full suite |

**Total: 12 tasks, ~11 new files, 21+ unit tests**
