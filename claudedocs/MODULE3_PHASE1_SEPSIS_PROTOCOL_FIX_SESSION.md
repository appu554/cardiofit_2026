# Module 3 Phase 1 - Sepsis Protocol Matching Fix

**Session Date**: 2025-10-28
**Issue**: Sepsis protocol not matching despite patient having clear sepsis indicators
**Status**: ⚠️ IN PROGRESS - Protocol loads but doesn't match yet

---

## Problem Analysis

### Initial Symptoms
- Patient data shows clear sepsis: lactate 2.8, fever 39°C, HR 110, RR 28, SIRS 3/4
- Module 3 output: `"phase1_matched_protocols": 0`
- Sepsis protocol should match but doesn't trigger

### Root Causes Identified

#### Issue 1: Missing sepsisRisk Flag ✅ FIXED
**Location**: [Module2_ContextAssembly.java:1779-1782](../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java#L1779-L1782)

**Problem**: Module 2's `buildRiskIndicators()` never calculated or set the `sepsisRisk` field, even though it had all the individual flags (fever, tachycardia, elevatedLactate).

**Fix Applied**:
```java
// Build the indicators first to access calculated SIRS score
RiskIndicators indicators = builder.build();

// SEPSIS RISK CALCULATION (for Module 3 protocol matching)
// Set sepsisRisk=true if patient meets SIRS criteria (≥2) or has severe sepsis indicators
boolean sepsisRisk = indicators.calculateSIRS() >= 2 || indicators.hasSevereSepsisIndicators();
indicators.setSepsisRisk(sepsisRisk);
```

**Impact**: `sepsisRisk` now correctly set to `true` when SIRS ≥ 2

---

#### Issue 2: Sepsis Protocol Validation Failure ✅ FIXED
**Location**: [sepsis-management.yaml:18-22](../backend/shared-infrastructure/flink-processing/src/main/resources/clinical-protocols/sepsis-management.yaml#L18-L22)

**Problem**: ProtocolLoader enforces 8 required fields. Sepsis protocol was missing 3:
- `source` - Missing entirely
- `activation_criteria` - Missing entirely
- `priority_determination` - Missing entirely

**Error Log**:
```
Protocol validation failed for sepsis-management.yaml: structure validation errors
Protocol missing required field: source
```

**Fix Applied**:
```yaml
source: "Surviving Sepsis Campaign 2021 Guidelines"
activation_criteria: "See trigger_criteria section"
priority_determination: "HIGH"
```

**Impact**: Sepsis protocol now loads successfully (8 protocols instead of 7)

---

#### Issue 3: Protocol Matching Still Returns 0 ⚠️ UNRESOLVED

**Current Output**:
```json
{
  "riskIndicators": {
    "sepsisRisk": true,  // ✅ NOW TRUE
    "fever": true,
    "tachycardia": true,
    "tachypnea": true,
    "elevatedLactate": true
  },
  "phaseData": {
    "phase1_protocol_count": 8,  // ✅ NOW 8 (was 7)
    "phase1_matched_protocols": 0  // ❌ STILL 0
  }
}
```

**Analysis**:
- Sepsis protocol loads and validates ✅
- `sepsisRisk: true` ✅
- SIRS score should be 3 (fever + tachycardia + tachypnea) ✅
- But protocol doesn't match ❌

**Hypothesis**: ProtocolMatcher may not be executing, or there's a mismatch between how PatientState exposes SIRS score vs how ConditionEvaluator requests it.

---

## Technical Details

### Files Modified

#### 1. Module2_ContextAssembly.java
**Lines**: 1776-1786
**Change**: Added sepsis risk calculation after building RiskIndicators
```java
// Build the indicators first
RiskIndicators indicators = builder.build();

// Calculate sepsis risk
boolean sepsisRisk = indicators.calculateSIRS() >= 2 ||
                     indicators.hasSevereSepsisIndicators();
indicators.setSepsisRisk(sepsisRisk);
```

#### 2. PatientState.java
**Lines**: 250-282
**Change**: Added getSirsScore() method for protocol evaluation
```java
public Integer getSirsScore() {
    int score = 0;
    // Temperature: <36°C or >38°C
    if (temp != null && (temp < 36.0 || temp > 38.0)) score++;
    // Heart Rate: >90 bpm
    if (hr != null && hr > 90) score++;
    // Respiratory Rate: >20 breaths/min
    if (rr != null && rr > 20) score++;
    // WBC: <4000 or >12000 cells/mm³
    if (wbc != null && (wbc < 4.0 || wbc > 12.0)) score++;
    return score;
}
```

#### 3. ConditionEvaluator.java
**Lines**: 356-362
**Change**: Added sirs_score parameter mapping
```java
case "sirs":
case "sirs_score":
    return patientState.getSirsScore();
```

#### 4. sepsis-management.yaml
**Lines**: 18-22
**Change**: Added required validation fields
```yaml
source: "Surviving Sepsis Campaign 2021 Guidelines"
activation_criteria: "See trigger_criteria section"
priority_determination: "HIGH"
```

---

## Sepsis Protocol Trigger Criteria

The sepsis protocol should match based on Condition 3 (SEPSIS-TRIG-003):
```yaml
trigger_criteria:
  match_logic: "ANY_OF"
  conditions:
    - condition_id: "SEPSIS-TRIG-003"
      match_logic: "ALL_OF"
      conditions:
        - parameter: "sirs_score"
          operator: ">="
          threshold: 2
        - parameter: "infection_suspected"
          operator: "=="
          threshold: true
```

**Patient Data Evaluation**:
- `sirs_score`: 3 (calculated from fever=true, tachycardia=true, tachypnea=true) ≥ 2 ✅
- `infection_suspected`: maps to `sepsisRisk` = true ✅

**Expected**: Protocol should match
**Actual**: `phase1_matched_protocols: 0`

---

## Next Steps

1. **Add debug logging to ProtocolMatcher**: Log when protocols are being evaluated
2. **Verify ConditionEvaluator execution**: Check if parameter extraction is working
3. **Test with simplified protocol**: Create minimal test protocol to isolate matching issue
4. **Check PatientState serialization**: Ensure getSirsScore() is accessible during evaluation

---

## Deployment History

1. **Deployment 1** (15:10:28): Initial deployment with sepsisRisk fix → Protocol still failed validation
2. **Deployment 2** (15:21:19): Added `source` field → Still failed validation
3. **Deployment 3** (15:24:01): Clean rebuild → Still failed validation
4. **Deployment 4** (15:26:20): Added all 3 required fields → **Protocol loaded successfully ✅**
5. **Current**: Protocol loads but doesn't match patients

---

## Verification Commands

```bash
# Check protocol loading
docker logs flink-taskmanager-1-2.1 2>&1 | grep "SEPSIS-BUNDLE"

# Check protocol count
docker logs flink-taskmanager-1-2.1 2>&1 | grep "Protocol loading complete"

# Monitor Module 3 output
timeout 10 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cds-recommendations-v1 \
  --from-beginning --max-messages 1
```

---

## Related Documentation
- [Module 3 Phase 1 Implementation](./MODULE3_PHASE1_DATA_MODELS_COMPLETE.md)
- [Protocol Matcher Implementation](./MODULE3_CROSSCHECK_VERIFICATION.md)
- [Clinical Threshold Sources](./CLINICAL_THRESHOLD_DATA_SOURCES.md)
