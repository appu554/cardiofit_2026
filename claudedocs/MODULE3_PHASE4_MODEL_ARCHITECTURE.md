# Phase 4 Diagnostic Models - Architecture Overview

## Model Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Diagnostic Test Repository                    │
│                     (Phase 4 - Module 3)                         │
└─────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Clinical   │
                              │   Protocol   │
                              └──────┬───────┘
                                     │ recommends
                                     ▼
                        ┌────────────────────────┐
                        │  TestRecommendation    │ ◄──────┐
                        │  (388 lines)           │        │
                        │                        │        │
                        │  - Priority (P0-P3)    │        │
                        │  - Urgency (STAT-ROUTINE)│      │
                        │  - Clinical rationale   │       │
                        │  - Evidence support     │       │ links to
                        └────────┬───────────────┘        │
                                 │                         │
                    ┌────────────┴──────────────┐         │
                    │                            │         │
         orders     ▼                            ▼         │
         ┌──────────────────┐         ┌──────────────────┐│
         │    LabTest       │         │  ImagingStudy    ││
         │   (404 lines)    │         │  (394 lines)     ││
         │                  │         │                  ││
         │ ┌──────────────┐ │         │ ┌──────────────┐││
         │ │  Specimen    │ │         │ │ ACR Rating   │││
         │ │  Requirements│ │         │ │              │││
         │ └──────────────┘ │         │ └──────────────┘││
         │ ┌──────────────┐ │         │ ┌──────────────┐││
         │ │  Reference   │ │         │ │ Radiation    │││
         │ │  Ranges      │ │         │ │ Exposure     │││
         │ └──────────────┘ │         │ └──────────────┘││
         │ ┌──────────────┐ │         │ ┌──────────────┐││
         │ │  Ordering    │ │         │ │ Contrast     │││
         │ │  Rules       │ │         │ │ Safety       │││
         │ └──────────────┘ │         │ └──────────────┘││
         │ ┌──────────────┐ │         │ ┌──────────────┐││
         │ │  Quality     │ │         │ │ Safety       │││
         │ │  Factors     │ │         │ │ Checks       │││
         │ └──────────────┘ │         │ └──────────────┘││
         │ ┌──────────────┐ │         │ ┌──────────────┐││
         │ │  CDS Rules   │ │         │ │ CDS Rules    │││
         │ └──────────────┘ │         │ └──────────────┘││
         └─────────┬────────┘         └─────────┬───────┘│
                   │                             │        │
                   │                             │        │
                   │      produces result        │        │
                   └────────────┬────────────────┘        │
                                │                          │
                                ▼                          │
                   ┌─────────────────────────┐            │
                   │     TestResult          │            │
                   │    (432 lines)          │────────────┘
                   │                         │
                   │ ┌─────────────────────┐ │
                   │ │ Result              │ │
                   │ │ Interpretation      │ │
                   │ └─────────────────────┘ │
                   │ ┌─────────────────────┐ │
                   │ │ Reference Range     │ │
                   │ │ Comparison          │ │
                   │ └─────────────────────┘ │
                   │ ┌─────────────────────┐ │
                   │ │ Quality             │ │
                   │ │ Indicators          │ │
                   │ └─────────────────────┘ │
                   │ ┌─────────────────────┐ │
                   │ │ Trending Analysis   │ │
                   │ │ (Delta Checks)      │ │
                   │ └─────────────────────┘ │
                   │ ┌─────────────────────┐ │
                   │ │ Reflex Actions      │ │
                   │ └─────────────────────┘ │
                   └─────────────────────────┘
```

## Clinical Workflow

```
1. PROTOCOL EVALUATION
   ├─ Clinical Protocol identifies need
   └─ Generates TestRecommendation

2. TEST ORDERING
   ├─ TestRecommendation links to LabTest or ImagingStudy
   ├─ Check contraindications
   ├─ Validate prerequisites
   └─ Order placed

3. SPECIMEN COLLECTION
   ├─ LabTest: SpecimenRequirements guide collection
   └─ ImagingStudy: Safety checks performed

4. TEST PERFORMANCE
   ├─ Lab performs test
   └─ Imaging performed with safety protocols

5. RESULT GENERATION
   ├─ TestResult created
   ├─ Automated interpretation
   ├─ Quality checks applied
   └─ Trending analysis

6. CLINICAL ACTION
   ├─ Alert if critical
   ├─ Physician notification if needed
   ├─ Reflex testing triggered
   └─ Follow-up recommendations
```

## Data Flow

```
Clinical Context
      │
      ▼
┌──────────────┐
│  Protocol    │
│  Engine      │
└──────┬───────┘
       │ generates
       ▼
┌──────────────────┐
│ Test             │
│ Recommendation   │ ───references───┐
└──────┬───────────┘                 │
       │ orders                      │
       ▼                             │
┌──────────────┐              ┌─────▼──────┐
│  Order       │              │  LabTest   │
│  Service     │◄─────────────│  or        │
└──────┬───────┘   validates  │  Imaging   │
       │                      └────────────┘
       │ collects
       ▼
┌──────────────┐
│  Lab/Imaging │
│  System      │
└──────┬───────┘
       │ produces
       ▼
┌──────────────────┐
│  TestResult      │
│  - interpretation│
│  - quality check │
│  - trending      │
└──────┬───────────┘
       │
       ├─────► Critical Alert
       ├─────► Physician Notification
       ├─────► Reflex Testing
       └─────► Follow-up Actions
```

## Model Complexity Analysis

### LabTest (404 lines)
```
Main Class:               50 lines
├─ SpecimenRequirements:  15 lines
├─ TestTiming:            15 lines
├─ ReferenceRange:        40 lines
│  ├─ NormalRange:        12 lines
│  └─ CriticalRange:      12 lines
├─ InterpretationGuidance: 15 lines
├─ OrderingRules:         20 lines
├─ QualityFactors:        15 lines
├─ CostData:              12 lines
├─ CDSRules:              15 lines
└─ Helper Methods:        180 lines (7 methods)
```

### ImagingStudy (394 lines)
```
Main Class:                      60 lines
├─ ACRAppropriatenessRating:     20 lines
├─ ImagingRequirements:          22 lines
├─ RadiationExposure:            20 lines
├─ ContrastSafety:               25 lines
├─ ImagingTiming:                18 lines
├─ OrderingRules:                20 lines
├─ SafetyChecks:                 20 lines
├─ CostData:                     15 lines
├─ CDSRules:                     15 lines
└─ Helper Methods:               150 lines (9 methods)
```

### TestRecommendation (388 lines)
```
Main Class:                   80 lines
├─ DecisionSupport:          20 lines
├─ OrderingInformation:      25 lines
├─ TestAlternative:          18 lines
├─ FollowUpGuidance:         20 lines
├─ Enumerations:             30 lines
│  ├─ TestCategory
│  ├─ Priority
│  └─ Urgency
└─ Helper Methods:           195 lines (15 methods)
```

### TestResult (432 lines)
```
Main Class:                   70 lines
├─ ReferenceRange:           20 lines
├─ QualityIndicators:        25 lines
├─ PreviousResults:          30 lines
│  └─ HistoricalValue:       10 lines
├─ ReflexActions:            15 lines
├─ Enumerations:             30 lines
│  ├─ TestType
│  ├─ ResultStatus
│  └─ ResultInterpretation
└─ Helper Methods:           232 lines (18 methods)
```

## Key Design Patterns

### 1. Builder Pattern (via Lombok)
```java
LabTest test = LabTest.builder()
    .testId("LAB-LACTATE-001")
    .testName("Serum Lactate")
    .specimen(SpecimenRequirements.builder()
        .specimenType("BLOOD")
        .volumeRequired(1.0)
        .build())
    .build();
```

### 2. Nested Static Classes
```java
public class LabTest {
    private SpecimenRequirements specimen;

    @Data
    @Builder
    public static class SpecimenRequirements implements Serializable {
        private String specimenType;
        // ...
    }
}
```

### 3. Enum Type Safety
```java
public enum Priority {
    P0_CRITICAL,
    P1_URGENT,
    P2_IMPORTANT,
    P3_ROUTINE
}
```

### 4. Fluent Interface
```java
if (recommendation.isHighPriority() &&
    recommendation.requiresImmediateAction() &&
    !recommendation.hasContraindication(patientConditions)) {
    // Order test
}
```

## Integration Patterns

### Pattern 1: Test Ordering
```java
// 1. Get recommendation from protocol
TestRecommendation rec = protocolEngine.recommend(patientContext);

// 2. Load test definition
LabTest test = testRepository.getLabTest(rec.getTestId());

// 3. Validate ordering
if (test.canOrder(patientContext) &&
    test.canReorder(lastOrderTime)) {
    // 4. Place order
    orderService.placeOrder(rec, test);
}
```

### Pattern 2: Result Interpretation
```java
// 1. Receive result
TestResult result = labSystem.getResult(orderId);

// 2. Load test definition
LabTest test = testRepository.getLabTest(result.getTestId());

// 3. Interpret result
String interpretation = test.interpretResult(
    result.getNumericValue(),
    patient.getPopulation()
);

// 4. Check for critical values
if (test.isCritical(result.getNumericValue(), patient.getPopulation())) {
    alertService.sendCriticalAlert(result);
}
```

### Pattern 3: Reflex Testing
```java
// 1. Check if result triggers reflex
if (result.getReflexActions().isReflexTriggered()) {

    // 2. Get reflex tests to order
    List<String> reflexTests = result.getReflexActions().getReflexTests();

    // 3. Generate recommendations for each
    for (String testId : reflexTests) {
        TestRecommendation reflexRec = recommendationBuilder
            .testId(testId)
            .priority(Priority.P1_URGENT)
            .urgency(Urgency.URGENT)
            .indication("Reflex testing based on " + result.getTestName())
            .build();

        orderService.placeOrder(reflexRec);
    }
}
```

## Performance Characteristics

### Memory Footprint
- **LabTest**: ~2KB per instance (with nested objects)
- **ImagingStudy**: ~2.5KB per instance
- **TestRecommendation**: ~1.5KB per instance
- **TestResult**: ~1KB per instance

### Serialization
- All classes implement Serializable
- Suitable for Flink state backend
- Compatible with Kafka serialization
- Redis caching compatible

### Lookup Performance
- Map-based reference ranges: O(1) lookup
- Enum-based status: O(1) comparison
- Helper methods: O(1) to O(n) depending on validation complexity

## Testing Strategy

### Unit Test Coverage Targets
- **Model Construction**: 100% (Builder pattern tests)
- **Helper Methods**: 100% (Business logic)
- **Nested Classes**: 90%+ (Complex structures)
- **Enum Operations**: 100% (Type safety)

### Integration Test Scenarios
1. YAML deserialization → Model instantiation
2. Test ordering workflow (Recommendation → Test → Order)
3. Result interpretation pipeline (Result → Interpretation → Alert)
4. Reflex testing automation (Result → Reflex → New Order)

## Conclusion

The Phase 4 diagnostic models provide a comprehensive, type-safe, and production-ready foundation for intelligent diagnostic test ordering and result interpretation. The architecture supports:

- ✅ Complete clinical metadata
- ✅ Automated interpretation logic
- ✅ Safety validation
- ✅ Evidence-based decision support
- ✅ Flink stream processing
- ✅ Extensible design

**Total Implementation**: 1,618 lines, 59KB, 4 main classes, 27 nested classes

**Status**: Ready for YAML data loading and recommender engine implementation.
