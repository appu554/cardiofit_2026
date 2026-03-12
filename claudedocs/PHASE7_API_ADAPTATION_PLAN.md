# Phase 7 API Adaptation Plan

**Status**: Ready for Implementation
**Date**: 2025-10-25
**Purpose**: Document actual model APIs and adaptation strategy for Agent 3/4 compilation fixes

---

## API Documentation Summary

### 1. Medication Model (Phase 6)

**Actual Structure**:
```java
class Medication {
    // Nested Monitoring object
    Monitoring monitoring;  // NOT getMonitoringParameters()

    static class Monitoring {
        List<String> labTests;              // NOT getParameters()
        String monitoringFrequency;
        List<String> vitalSigns;
        List<String> clinicalAssessment;
        String therapeuticRange;
    }

    // Nested Administration object
    Administration administration;  // NOT getAdministrationGuidelines()

    static class Administration {
        List<String> routes;
        String preferredRoute;
        Map<String, String> preparation;  // NOT getInstructions()
        String dilution;
        // ... more fields
    }

    // Nested AdverseEffects object
    AdverseEffects adverseEffects;  // Returns object, not List<String>

    static class AdverseEffects {
        Map<String, String> common;         // NOT a List!
        Map<String, String> serious;
        List<String> blackBoxWarnings;
        String monitoring;
        // NO isEmpty() method - use: common == null || common.isEmpty()
    }

    // Missing methods Agent 3 assumed:
    // - getMonitoringParameters() → Use getMonitoring().getLabTests()
    // - requiresTherapeuticMonitoring() → Use getMonitoring() != null
    // - getEvidenceLevel() → No direct field, use guidelineReferences
    // - getTypicalDuration() → Use adultDosing.standard.duration
}
```

**Adaptation Strategy**:
- Replace `med.getMonitoringParameters()` → `med.getMonitoring().getLabTests()`
- Replace `med.requiresTherapeuticMonitoring()` → `med.getMonitoring() != null`
- Replace `med.getAdverseEffects()` as List → Handle as nested object with common/serious Maps
- Replace `med.getEvidenceLevel()` → Use empty string or fetch from guidelines
- Replace `med.getTypicalDuration()` → `med.getAdultDosing().getStandard().getDuration()`
- Add null safety checks for all nested object access

### 2. CalculatedDose Model (Phase 6)

**Actual Structure**:
```java
class CalculatedDose {
    String calculatedDose;
    String calculatedFrequency;
    String adjustmentReason;
    String calculationNotes;  // DESCRIPTION of calculation
    // NO getCalculationMethod() - use calculationNotes instead
}
```

**Adaptation Strategy**:
- Replace `dose.getCalculationMethod()` → `dose.getCalculationNotes()`
- Method returns String description, not enum

### 3. PatientContextState Model

**Actual Structure**:
```java
class PatientContextState {
    Map<String, Object> latestVitals;  // NOT direct getWeight()
    Map<String, LabResult> recentLabs;  // NOT direct getCreatinineClearance()
    List<Condition> chronicConditions;   // NOT getDiagnoses()
    PatientDemographics demographics;    // Weight is nested here

    // NO direct getWeight() - must access demographics
    // NO direct getCreatinineClearance() - must calculate from labs
    // NO getDiagnoses() - use chronicConditions
}
```

**Adaptation Strategy**:
- Create helper method `getWeight(PatientContextState)` that accesses `demographics.weight`
- Create helper method `getCreatinineClearance(PatientContextState)` that:
  - Extracts creatinine from `recentLabs`
  - Calculates CrCl using Cockcroft-Gault formula
  - Returns calculated value
- Replace `patient.getDiagnoses()` → `patient.getChronicConditions()`
- Add null safety for nested access

### 4. ClinicalAction vs StructuredAction Relationship

**Actual Structure**:
```java
class ClinicalAction {
    String actionId;
    ActionType actionType;
    String description;
    MedicationDetails medicationDetails;
    DiagnosticDetails diagnosticDetails;
    // NO getStructuredAction() method!
    // ClinicalAction does NOT wrap StructuredAction
}

class StructuredAction {
    // Agent 1 created this as NEW action type
    // NOT related to ClinicalAction
}
```

**Adaptation Strategy**:
- **Option A (RECOMMENDED)**: Remove StructuredAction references from RecommendationEnricher
  - Work directly with ClinicalAction objects
  - Do not attempt to extract StructuredAction

- **Option B**: Convert ClinicalAction to StructuredAction
  - Create adapter method `convertToStructuredAction(ClinicalAction)`
  - Map fields from ClinicalAction → StructuredAction
  - Use StructuredAction in new logic

- **Implementation**: Use Option A for compatibility with existing code

### 5. ProtocolAction Inner Class Mismatches

**Actual Structure**:
```java
class ProtocolAction {
    static class MedicationDetails {  // INNER CLASS
        String medicationId;
        String dose;
        String route;
        // ... ProtocolAction-specific fields
    }

    static class DiagnosticDetails {  // INNER CLASS
        String testType;
        String testName;
        // ... ProtocolAction-specific fields
    }
}

// SEPARATE top-level classes:
class MedicationDetails {  // From Agent 1
    // Different structure!
}

class DiagnosticDetails {  // From Agent 1
    // Different structure!
}
```

**Adaptation Strategy**:
- When converting ProtocolAction → ClinicalAction:
  - Create new top-level MedicationDetails from ProtocolAction.MedicationDetails
  - Create new top-level DiagnosticDetails from ProtocolAction.DiagnosticDetails
  - Map fields appropriately

- Create converter method:
```java
MedicationDetails convertMedicationDetails(ProtocolAction.MedicationDetails protocolMed) {
    MedicationDetails details = new MedicationDetails();
    // Map common fields
    return details;
}
```

### 6. Contraindication Model Constructor/Methods

**Actual Structure**:
```java
class Contraindication {
    ContraindicationType contraindicationType;  // ENUM
    Severity severity;  // ENUM
    String contraindicationDescription;
    String evidence;  // NOT setRationale()
    String clinicalGuidance;  // NOT setMedicationName()

    // Constructor expects ENUM, not String:
    public Contraindication(ContraindicationType type, String description)

    // Severity enum:
    enum Severity {
        CRITICAL,  // NOT MODERATE!
        HIGH,
        // Check actual enum values
    }

    // NO setMedicationName() method
    // NO setRationale() method
    // Use setEvidence() and setContraindicationDescription()
}
```

**Adaptation Strategy**:
- Replace String type → `ContraindicationType.valueOf(type)` or create enum mapping
- Replace `setMedicationName(name)` → Store in `contraindicationDescription`
- Replace `setRationale(rationale)` → `setEvidence(rationale)`
- Fix Severity enum: Use `Severity.CRITICAL` or `Severity.HIGH` instead of non-existent `MODERATE`
- Add enum conversion helper:
```java
ContraindicationType mapToContraindicationType(String type) {
    switch(type.toUpperCase()) {
        case "ALLERGY": return ContraindicationType.ALLERGY;
        case "INTERACTION": return ContraindicationType.INTERACTION;
        // ... more mappings
        default: return ContraindicationType.ABSOLUTE;
    }
}
```

### 7. PatientContext setCurrentMedications Signature

**Actual Structure**:
```java
class PatientContext {
    EnrichedPatientContext enrichedContext;  // Contains PatientContextState

    // PatientContextState has:
    Map<String, Medication> activeMedications;  // Medication from PatientContextState

    // NOT setCurrentMedications(Map<String, Medication from Phase 6>)
}
```

**Adaptation Strategy**:
- Issue: Type mismatch between `com.cardiofit.flink.models.Medication` and `com.cardiofit.flink.knowledgebase.medications.model.Medication`
- Solution: Don't use setCurrentMedications - work with PatientContextState directly
- Alternative: Convert Phase 6 Medication → PatientContextState.Medication format

---

## Implementation Order

### Phase 1: Fix MedicationActionBuilder.java (15 errors)

**Files to modify**: `MedicationActionBuilder.java`

**Changes**:
1. Add helper methods for nested Medication access:
```java
private List<String> getMonitoringParameters(Medication med) {
    if (med.getMonitoring() == null) return new ArrayList<>();
    return med.getMonitoring().getLabTests();
}

private boolean requiresMonitoring(Medication med) {
    return med.getMonitoring() != null &&
           med.getMonitoring().getLabTests() != null &&
           !med.getMonitoring().getLabTests().isEmpty();
}

private String getAdministrationGuidance(Medication med) {
    if (med.getAdministration() == null) return "";
    return med.getAdministration().getPreferredRoute();
}

private List<String> getAdverseEffectsList(Medication med) {
    List<String> effects = new ArrayList<>();
    if (med.getAdverseEffects() != null) {
        if (med.getAdverseEffects().getCommon() != null) {
            effects.addAll(med.getAdverseEffects().getCommon().keySet());
        }
        if (med.getAdverseEffects().getSerious() != null) {
            effects.addAll(med.getAdverseEffects().getSerious().keySet());
        }
    }
    return effects;
}
```

2. Replace direct method calls with helper methods
3. Fix CalculatedDose.getCalculationMethod() → getCalculationNotes()
4. Add PatientContextState helper methods for weight/CrCl
5. Fix Medication.getEvidenceLevel() references
6. Fix Medication.getTypicalDuration() references

### Phase 2: Fix RecommendationEnricher.java (6 errors)

**Files to modify**: `RecommendationEnricher.java`

**Changes**:
1. Remove all `action.getStructuredAction()` calls
2. Work directly with ClinicalAction objects
3. Update logic to extract data directly from ClinicalAction fields

### Phase 3: Fix AlternativeActionGenerator.java (4 errors)

**Files to modify**: `AlternativeActionGenerator.java`

**Changes**:
1. Fix `getMonitoringParameters()` calls using helper methods
2. Remove or fix `setCurrentMedications()` call

### Phase 4: Fix ClinicalRecommendationProcessor.java (6 errors)

**Files to modify**: `ClinicalRecommendationProcessor.java`

**Changes**:
1. Create converter methods for ProtocolAction inner classes
2. Fix Contraindication constructor to use enum types
3. Fix `setMedicationName()` → use `setContraindicationDescription()`
4. Fix `setRationale()` → use `setEvidence()`
5. Fix Severity enum reference (use CRITICAL instead of MODERATE)

---

## Validation Checklist

After each file fix:
- [ ] Run `mvn clean compile -DskipTests`
- [ ] Verify error count decreases
- [ ] Document any new issues discovered
- [ ] Update INTEGRATION_STATUS.md

After all fixes:
- [ ] Full compilation succeeds
- [ ] No warnings related to our changes
- [ ] All adapter methods have null safety
- [ ] All adapter methods have logging for debugging

---

## Risk Mitigation

**Potential Issues**:
1. **Null Pointer Exceptions**: All nested object access must check for null
2. **Type Conversion Failures**: Enum conversions may fail for unexpected values
3. **Data Loss**: Converting between incompatible types may lose information
4. **Performance**: Additional adapter methods add overhead

**Mitigation Strategies**:
1. Add comprehensive null checks
2. Use try-catch for enum conversions with fallback defaults
3. Log warnings when data is lost in conversion
4. Mark adapter methods as @inline candidates for JIT optimization

---

**Next Step**: Begin Phase 1 - Fix MedicationActionBuilder.java
