# Module 3 Reusable Assets - Quick Reference Map

## File Paths & Reusability Status

### Data Models - FULLY REUSABLE
```
├── /com/cardiofit/flink/models/
│   ├── SemanticEvent.java ✓ REUSE FULLY
│   │   └── Inner Classes: ClinicalAlert, GuidelineRecommendation, DrugInteraction, Contraindication, SemanticQuality
│   │
│   ├── PatientContext.java ✓ REUSE FULLY
│   │   └── Inner Classes: PatientDemographics, PatientLocation, ConditionEntry, MedicationEntry, LabResult, etc.
│   │
│   ├── PatientContextState.java ✓ REUSE FULLY (State Management Pattern)
│   │   └── Unified state with RocksDB persistence
│   │
│   ├── DrugInteraction.java ✓ REUSE FULLY
│   │   └── Builder pattern, all fields + methods implemented
│   │
│   ├── AllergyAlert.java ✓ REUSE FULLY
│   │   └── Builder pattern, ready for alert generation
│   │
│   ├── EnrichedEvent.java ✓ EXTEND
│   │   └── Input from Module 2; Module 3 should populate:
│   │       - potentialInteractions (List<DrugInteraction>)
│   │       - applicableProtocols (List<String>)
│   │
│   ├── EventType.java ✓ REUSE AS-IS
│   │   └── isMedicationRelated(), isClinical(), isCritical() methods
│   │
│   └── RiskIndicators.java ✓ REUSE AS-IS
│       └── Boolean flags for: tachycardia, hypertension, hypoxia, bradycardia, etc.
```

### Clinical Logic - HIGHLY REUSABLE
```
├── /com/cardiofit/flink/recommendations/
│   ├── RecommendationEngine.java ✓ INTEGRATE DIRECTLY
│   │   ├── generateImmediateActions()
│   │   ├── generateSuggestedLabs()
│   │   ├── determineMonitoringFrequency()
│   │   ├── generateReferrals()
│   │   └── generateEvidenceBasedInterventions()
│   │
│   └── Recommendations.java ✓ EXTEND FOR OUTPUT
│       └── Container for all recommendation types
```

### State Management - PATTERN TO REUSE
```
├── /com/cardiofit/flink/migration/
│   └── StateSchemaVersion.java ✓ EXTEND FOR NEW STATES
│       └── Example: ValueStateDescriptor<SemanticContext>
│
├── /com/cardiofit/flink/state/
│   └── HealthcareStateDescriptors.java ✓ EXTEND IF EXISTS
│       └── Centralized state descriptor registration
```

### Processor Framework - PARTIALLY REUSABLE
```
├── /com/cardiofit/flink/operators/
│   ├── Module3_SemanticMesh.java ✓ ENHANCE (Currently 40% complete)
│   │   ├── SemanticReasoningProcessor ✓ EXTEND
│   │   │   └── Already has: assessClinicalSignificance(), stratifyRisk(), etc.
│   │   │   └── Need to add: Call RecommendationEngine, populate recommendations
│   │   │
│   │   ├── ClinicalGuidelineProcessor ✓ ENHANCE
│   │   │   └── Currently: pass-through only
│   │   │   └── Need: Match protocols from KB3, apply guidelines
│   │   │
│   │   ├── DrugSafetyProcessor ✗ IMPLEMENT
│   │   │   └── Currently: stub only (checkDrugInteractions is empty)
│   │   │   └── Need: Real drug interaction checking, allergy matching
│   │   │
│   │   └── TerminologyStandardizationProcessor ✓ ENHANCE
│   │       └── Currently: basic mapping only
│   │       └── Need: Full KB7 terminology integration
│   │
│   └── Inner Classes in Module3_SemanticMesh:
│       ├── SemanticContext ✓ PATTERN TO COPY
│       ├── ClinicalRule ✓ PATTERN TO COPY
│       └── KnowledgeBaseUpdate ✓ PATTERN TO COPY
```

### Knowledge Base Access - NEEDS IMPLEMENTATION
```
├── /com/cardiofit/flink/clients/
│   ├── Neo4jGraphClient.java (exists) ✓ REFERENCE
│   │   └── Pattern for external KB access
│   │
│   └── KnowledgeBaseClient.java (check if exists) ? CREATE IF MISSING
│       ├── queryKB3_ClinicalProtocols()
│       ├── queryKB4_DrugCalculations()
│       ├── queryKB5_DrugInteractions()
│       ├── queryKB6_ValidationRules()
│       └── queryKB7_Terminology()
```

### New Packages to Create
```
├── /com/cardiofit/flink/clinical/ (NEW)
│   ├── ContraindicationChecker.java ✗ CREATE
│   │   ├── checkDrugDrugInteractions()
│   │   └── checkDrugAllergyContraindications()
│   │
│   ├── DrugInteractionEngine.java ✗ CREATE
│   │   ├── detectInteractions()
│   │   ├── calculateSeverity()
│   │   └── generateRecommendations()
│   │
│   ├── ClinicalProtocolMatcher.java ✗ CREATE
│   │   ├── matchProtocols()
│   │   ├── evaluateApplicability()
│   │   └── extractActionItems()
│   │
│   └── DosageValidator.java ✗ CREATE
│       ├── validateDosage()
│       ├── checkKidneyFunction()
│       └── checkDrugInteractionDosage()
│
├── /com/cardiofit/flink/enrichment/ (might exist)
│   ├── DrugInteractionEnricher.java ✗ CREATE
│   │   └── Adds drug interactions to SemanticEvent
│   │
│   └── ContraindicationEnricher.java ✗ CREATE
│       └── Adds contraindications to SemanticEvent
```

---

## Quick Integration Checklist

### Phase 1: Extend Existing Models (LOW EFFORT)
- [ ] Add fields to SemanticEvent: recommendations (Recommendations), additionalContraindications (List<Contraindication>)
- [ ] Verify PatientContext has allergies and activeMedications populated
- [ ] Extend PatientContextState to include recommendation cache (optional)

### Phase 2: Integrate RecommendationEngine (MEDIUM EFFORT)
- [ ] Create new processor or enhance SemanticReasoningProcessor
- [ ] Get patient snapshot from PatientContextState
- [ ] Get risk indicators from event
- [ ] Call RecommendationEngine.generateRecommendations()
- [ ] Populate SemanticEvent with recommendations

### Phase 3: Implement Drug Safety (MEDIUM-HIGH EFFORT)
- [ ] Create ContraindicationChecker
- [ ] Create DrugInteractionEngine
- [ ] Query KB5 for drug interactions
- [ ] Query PatientContext for allergies
- [ ] Enhance DrugSafetyProcessor with real implementation

### Phase 4: Protocol Matching (HIGH EFFORT)
- [ ] Create ClinicalProtocolMatcher
- [ ] Query KB3 for applicable protocols
- [ ] Implement protocol matching algorithm
- [ ] Add matched protocols to SemanticEvent.applicableProtocols

### Phase 5: Testing (MEDIUM EFFORT)
- [ ] Unit tests for ContraindicationChecker
- [ ] Unit tests for RecommendationEngine integration
- [ ] Integration tests with real KB data
- [ ] E2E tests with mock patients

---

## Key Classes to Study (as reference)

1. **RecommendationEngine.java** - Shows pattern for generating recommendations
2. **PatientContextState.java** - Shows state management pattern
3. **Module3_SemanticMesh.SemanticReasoningProcessor** - Shows processor implementation
4. **EnrichedEvent.java** - Shows how data flows through modules

---

## Kafka Topics Configuration

### Inputs (Read From)
```
- CLINICAL_PATTERNS (from Module 2)
- KB3_CLINICAL_PROTOCOLS
- KB4_DRUG_CALCULATIONS
- KB5_DRUG_INTERACTIONS
- KB6_VALIDATION_RULES
- KB7_TERMINOLOGY
```

### Outputs (Write To)
```
- SEMANTIC_MESH_UPDATES (main semantic events)
- SAFETY_EVENTS (drug interactions, contraindications)
- ALERT_MANAGEMENT (clinical alerts)
- CLINICAL_REASONING_EVENTS (guideline recommendations)
```

---

## Execution Parameters (Already Configured)
```java
env.setParallelism(4);
env.enableCheckpointing(30000);      // 30-second checkpoints
env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
```

---

## Data Size Estimates for Optimization

- **SemanticEvent**: ~2-5 KB (depends on nested lists)
- **PatientContextState**: ~10-20 KB (multiple collections)
- **Recommendations**: ~1-2 KB (just text recommendations)
- **DrugInteraction list**: 0.5 KB per interaction
- **State retention**: Time-windowed (24-48h for labs)

---

## Code Duplication Risk - Avoid

These classes exist in BOTH locations:
- `DrugInteraction.java` - exists in `/flink/models/` AND `/stream/models/`
- `AllergyAlert.java` - exists in `/flink/models/` AND `/stream/models/`
- `CanonicalEvent.java` - exists in `/flink/models/` AND `/stream/models/`

**Recommendation**: Use from `/flink/models/` for Module 3

---

## Timing Notes

- Module 2 produces EnrichedEvents continuously
- Module 3 must process within 30-second checkpoint window
- KB change streams are broadcast (low latency updates)
- Recommendation generation should be sub-second per event

