# Module 3: CDS Recommendations Integration Plan

**Date**: October 28, 2025
**Status**: ⚠️ **IN PROGRESS** - Type conversion issues blocking integration
**Objective**: Integrate existing RecommendationEngine into Module 3 CDS output

---

## Current Situation

### ✅ What's Working
1. **Output Format Fixed**: Module 3 now produces complete patient state + phase metadata
2. **RecommendationEngine Exists**: Fully implemented recommendation engine in `/src/main/java/com/cardiofit/flink/recommendations/`
3. **Data Available**: All necessary clinical data flows through Module 3 processor

### ❌ What's Blocking
**Type Conversion Complexity**: Module 2's internal data types (`RiskIndicators`, `Set<SimpleAlert>`) don't match what RecommendationEngine expects.

**Type Mismatches**:
```java
// Module 2 uses:
- RiskIndicators (typed class)
- Set<SimpleAlert> (typed set)
- PatientContextState (typed class)

// RecommendationEngine expects:
- EnhancedRiskIndicators.RiskAssessment
- List<ClinicalAlert>
- PatientSnapshot
```

---

## Recommendation Engine Overview

### Location
`/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/recommendations/RecommendationEngine.java`

### What It Generates

**1. Immediate Actions** - Critical interventions based on:
- P0_CRITICAL and P1_URGENT alerts
- Specific high-risk conditions (hypertensive crisis, sepsis, etc.)
- Protocol requirements

**2. Suggested Labs** - Time-based testing recommendations:
- Missing baseline labs
- Condition-specific testing (diabetes → HbA1c, cardiac → troponin)
- Follow-up labs based on time since last test

**3. Monitoring Frequency** - Based on acuity level:
- CONTINUOUS: NEWS2 ≥ 7 or critical alerts
- HOURLY: NEWS2 5-6 or HIGH acuity
- Q4H (Every 4 hours): NEWS2 3-4 or MEDIUM acuity
- ROUTINE: NEWS2 < 3 or LOW acuity

**4. Referrals** - Specialist consultations from:
- Matched protocols
- Risk-based recommendations (cardiology, endocrinology, nephrology)

**5. Evidence-Based Interventions** - From similar patient outcomes

---

## Integration Approach: Two Options

### Option A: Direct Type Mapping (Complex)

**Pros**: Uses existing RecommendationEngine as-is
**Cons**: Requires extensive type conversion logic

**Steps**:
1. Create type converters for each mismatch:
   - `RiskIndicators` → `EnhancedRiskIndicators.RiskAssessment`
   - `Set<SimpleAlert>` → `List<ClinicalAlert>`
   - `PatientContextState` → `PatientSnapshot`
2. Handle missing fields and method signature differences
3. Test all conversion paths

**Estimated Complexity**: 3-4 hours of development + testing

---

### Option B: Simplified Inline Recommendation Logic (Recommended)

**Pros**: Faster implementation, no type conversion complexity
**Cons**: Duplicates some logic from RecommendationEngine

**Implementation**: Add simplified recommendation logic directly in Module 3:

```java
private void generateSimplifiedRecommendations(
        EnrichedPatientContext context,
        CDSEvent cdsEvent,
        List<Protocol> matchedProtocols) {

    PatientContextState state = context.getPatientState();

    // 1. Immediate Actions from Critical Alerts
    List<String> immediateActions = new ArrayList<>();
    if (state.getActiveAlerts() != null) {
        for (SimpleAlert alert : state.getActiveAlerts()) {
            String priority = alert.getPriorityLevel();
            if (priority != null && (priority.contains("P0_CRITICAL") || priority.contains("P1_URGENT"))) {
                immediateActions.add(alert.getMessage());
            }
        }
    }

    // 2. Monitoring Frequency based on NEWS2
    String monitoringFrequency = "ROUTINE";
    if (state.getNews2Score() != null) {
        int news2 = state.getNews2Score();
        if (news2 >= 7) {
            monitoringFrequency = "CONTINUOUS";
            immediateActions.add("URGENT: ICU-level monitoring required - NEWS2 score " + news2);
        } else if (news2 >= 5) {
            monitoringFrequency = "HOURLY";
            immediateActions.add("Increase monitoring frequency - NEWS2 score " + news2);
        } else if (news2 >= 3) {
            monitoringFrequency = "Q4H";
        }
    }

    // 3. Suggested Labs based on Risk Indicators
    List<String> suggestedLabs = new ArrayList<>();
    RiskIndicators risks = state.getRiskIndicators();
    if (risks != null) {
        if (risks.isFever() && risks.isElevatedLactate()) {
            suggestedLabs.add("Blood cultures (before antibiotics)");
            suggestedLabs.add("Complete blood count with differential");
            suggestedLabs.add("Procalcitonin for sepsis evaluation");
        }
        if (risks.isHypoxia() || risks.isTachypnea()) {
            suggestedLabs.add("Arterial blood gas");
            suggestedLabs.add("Chest X-ray");
        }
        if (risks.isTachycardia() || risks.isElevatedTroponin()) {
            suggestedLabs.add("ECG");
            suggestedLabs.add("Troponin series");
            suggestedLabs.add("BNP/NT-proBNP");
        }
    }

    // 4. Referrals from Protocols
    List<String> referrals = new ArrayList<>();
    for (Protocol protocol : matchedProtocols) {
        if (protocol.getProtocolId().contains("SEPSIS")) {
            referrals.add("Infectious Disease consultation");
            referrals.add("ICU/Critical Care team");
        }
        if (protocol.getProtocolId().contains("CARDIAC")) {
            referrals.add("Cardiology consultation");
        }
        if (protocol.getProtocolId().contains("RESPIRATORY")) {
            referrals.add("Pulmonology consultation");
        }
    }

    // Add to CDS Event
    if (!immediateActions.isEmpty()) {
        cdsEvent.addCDSRecommendation("immediateActions", immediateActions);
    }
    if (!suggestedLabs.isEmpty()) {
        cdsEvent.addCDSRecommendation("suggestedLabs", suggestedLabs);
    }
    cdsEvent.addCDSRecommendation("monitoringFrequency", monitoringFrequency);
    if (!referrals.isEmpty()) {
        cdsEvent.addCDSRecommendation("referrals", referrals);
    }
}
```

**Estimated Complexity**: 1-2 hours of development + testing

---

## Expected Output Example

For the sepsis patient (PAT-ROHAN-001), Module 3 would generate:

```json
{
  "patientId": "PAT-ROHAN-001",
  "patientState": {...},
  "eventType": "VITAL_SIGN",
  "eventTime": 1760171000000,
  "phaseData": {...},
  "cdsRecommendations": {
    "immediateActions": [
      "SEPSIS LIKELY - SIRS criteria with elevated lactate (2.8 mmol/L), evaluate for infection source and organ dysfunction",
      "Hypoxia detected - Oxygen saturation 92% (normal ≥95%)",
      "NEWS2 score 8 - HIGH RISK: Emergency assessment required by critical care team",
      "URGENT: ICU-level monitoring required - NEWS2 score 8"
    ],
    "suggestedLabs": [
      "Blood cultures (before antibiotics)",
      "Complete blood count with differential",
      "Procalcitonin for sepsis evaluation",
      "Arterial blood gas",
      "Chest X-ray"
    ],
    "monitoringFrequency": "CONTINUOUS",
    "referrals": [
      "Infectious Disease consultation",
      "ICU/Critical Care team",
      "Pulmonology consultation"
    ]
  },
  "phaseDataCount": 19
}
```

---

## Recommendation: Option B (Simplified)

**Rationale**:
1. ✅ **Faster to implement** - No complex type conversions
2. ✅ **Focused on critical use cases** - Immediate actions, labs, monitoring
3. ✅ **Type-safe** - Works with actual internal types
4. ✅ **Maintainable** - Clear, straightforward logic
5. ✅ **Testable** - Easy to verify recommendations match clinical state

**Trade-off**: Some duplication from RecommendationEngine, but focused on most critical recommendations.

**Future Enhancement**: Once working, can incrementally add more sophisticated logic from RecommendationEngine.

---

## Implementation Steps (Option B)

### Step 1: Add simplified recommendation method to Module3_ComprehensiveCDS.java
- Replace current `generateClinicalRecommendations()` method
- Use simplified inline logic (code above)
- Work directly with `RiskIndicators` and `SimpleAlert` types

### Step 2: Update protocol matching
- Change `protocolMatcher.matchProtocols(context)` signature check
- May need to pass `PatientContextState` instead of full `EnrichedPatientContext`

### Step 3: Rebuild and deploy
```bash
mvn clean package -DskipTests
# Upload to Flink
# Deploy new job
```

### Step 4: Test with sepsis patient
```bash
python3 send-full-module2-event.py
# Check output for cdsRecommendations
```

### Step 5: Verify recommendations
Expected output:
- ✅ 4+ immediate actions (from critical alerts + NEWS2 logic)
- ✅ 5+ suggested labs (sepsis workup + respiratory panel)
- ✅ "CONTINUOUS" monitoring frequency (NEWS2 = 8)
- ✅ 2-3 referrals (ID, ICU, Pulmonology)

---

## Type Reference

### RiskIndicators Class
```java
// Located in: com.cardiofit.flink.models.RiskIndicators
boolean isTachycardia()
boolean isHypotension()
boolean isFever()
boolean isHypoxia()
boolean isTachypnea()
boolean isElevatedLactate()
boolean isElevatedTroponin()
// ... more risk flags
```

### SimpleAlert Class
```java
// Located in: com.cardiofit.flink.models.SimpleAlert
String getMessage()
String getSeverity()
String getPriorityLevel()  // "P0_CRITICAL", "P1_URGENT", etc.
Double getPriorityScore()
```

### Protocol Class
```java
// Located in: com.cardiofit.flink.protocols.ProtocolMatcher.Protocol
String getProtocolId()
String getName()
Map<String, Object> getCriteria()
```

---

## Current Module 3 Files Modified

1. ✅ `/src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS.java`
   - CDSEvent model updated with patientState, eventType, processingTime, latencyMs
   - Added cdsRecommendations Map<String, Object>
   - Added generateClinicalRecommendations() method (needs simplification)

2. ⚠️ Type conversion methods (need replacement):
   - `convertRiskIndicators()` - 🔴 Has compilation errors
   - `convertAcuityScore()` - 🔴 Has compilation errors
   - `convertAlerts()` - 🔴 Has compilation errors
   - `convertPriority()` - 🔴 Has compilation errors

---

## Next Session Actions

1. **Remove failing type conversion methods**
2. **Implement simplified inline recommendation logic** (Option B code above)
3. **Fix protocol matching call** - Check method signature
4. **Rebuild and deploy**
5. **Test with sepsis patient event**
6. **Verify cdsRecommendations in output**

---

## Success Criteria

✅ **Module 3 produces output with populated cdsRecommendations**:
```json
{
  "cdsRecommendations": {
    "immediateActions": [...],  // 4+ actions
    "suggestedLabs": [...],     // 5+ labs
    "monitoringFrequency": "CONTINUOUS",
    "referrals": [...]          // 2+ referrals
  }
}
```

✅ **Recommendations are clinically appropriate** for sepsis patient:
- Immediate actions mention sepsis protocol, ICU monitoring
- Labs include blood cultures, lactate, ABG
- Monitoring is CONTINUOUS (NEWS2=8)
- Referrals include ID and ICU teams

✅ **No compilation errors**

✅ **Job runs without exceptions**

---

**Status**: Ready for implementation in next session
**Recommended Time**: 1-2 hours for Option B implementation + testing
**Generated by**: Claude Code
**Date**: October 28, 2025
