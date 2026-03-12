# Module 3: CDS Alignment Plan
## Complete Roadmap to Match Clinical Decision Support Specification

**Document Version**: 2.0
**Created**: 2025-10-21
**Purpose**: Transform Module 3 from basic protocol library to full CDS-compliant Clinical Recommendation Engine

---

## Executive Summary

### Current State Analysis
Module 3 Clinical Recommendation Engine is **95% structurally complete** but only **60% functionally aligned** to the Clinical Decision Support (CDS) specification documented in:
- `cds.txt` (3,200+ lines) - Phase 1 implementation guide
- `Clinical_Knowledge_Base_Structure.txt` (5,800+ lines) - Complete architecture blueprint
- `ProtocolLoader.txt` (3,400+ lines) - Full Java implementation reference

### Critical Gap: Missing Runtime Intelligence
While Module 3 has all 16 clinical protocols (391 KB of YAML), it **lacks the runtime logic to USE them effectively**:

❌ **No Trigger Evaluation**: Protocols never activate automatically based on patient state
❌ **No Time Enforcement**: Cannot track "antibiotics within 1 hour" for sepsis bundles
❌ **No Medication Selection**: Cannot handle penicillin allergies or choose alternatives
❌ **No Confidence Scoring**: Cannot rank multiple matching protocols
❌ **No Validation**: Protocols loaded without schema validation or error reporting

**Impact**: System has comprehensive clinical knowledge but cannot apply it intelligently in real-time. This renders it **unsuitable for production clinical use**.

### Alignment Strategy
Transform Module 3 through **3 phases over 12-17 hours**:

**Phase 1 (Critical - 5-8 hours)**: Implement runtime intelligence core
- ConditionEvaluator.java - Trigger evaluation with AND/OR logic
- MedicationSelector.java - Rule-based selection with allergy checking
- TimeConstraintTracker.java - Deadline alerts and bundle compliance

**Phase 2 (High Priority - 4-6 hours)**: Add quality and performance features
- ConfidenceCalculator.java - Protocol confidence scoring
- ProtocolValidator.java - Schema validation with error reporting
- KnowledgeBaseManager.java - Singleton pattern with fast indexes

**Phase 3 (Medium Priority - 3-4 hours)**: Advanced clinical features
- EscalationRuleEvaluator.java - Auto-ICU transfer recommendations
- Enhanced YAML structure - Special populations, escalation rules
- Protocol migration - Update all 16 protocols to enhanced structure

---

## Table of Contents

1. [Current State vs Target State](#current-state-vs-target-state)
2. [Architecture Changes Overview](#architecture-changes-overview)
3. [Gap Analysis by Component](#gap-analysis-by-component)
4. [New Java Classes Required](#new-java-classes-required)
5. [Enhanced Protocol YAML Structure](#enhanced-protocol-yaml-structure)
6. [Protocol Migration Plan](#protocol-migration-plan)
7. [Implementation Roadmap](#implementation-roadmap)
8. [Acceptance Criteria](#acceptance-criteria)
9. [Risk Assessment](#risk-assessment)
10. [Effort Estimation](#effort-estimation)

---

## Current State vs Target State

### Current Module 3 Implementation

**What Exists (95% Structurally Complete)**:
```
✅ 16 Clinical Protocol YAML Files (391 KB)
   - Sepsis, STEMI, Stroke, ACS, DKA, COPD, Heart Failure, AKI
   - GI Bleeding, Anaphylaxis, Neutropenic Fever, HTN Crisis
   - Tachycardia, Metabolic Syndrome, Pneumonia, Respiratory Failure

✅ Basic ProtocolLoader.java (400 lines)
   - Loads YAML files using Jackson
   - Creates Protocol POJOs
   - Stores in HashMap

✅ Stub ProtocolMatcher.java (~150 lines)
   - matchProtocols() method exists but no trigger evaluation
   - Just returns empty list

✅ Basic ActionBuilder.java (~200 lines)
   - buildActions() method exists
   - No medication selection algorithm
   - No contraindication checking implementation

✅ Contraindication checking logic (basic)
   - Can detect contraindications
   - No alternative action generation
```

**What's Missing (40% Functional Gap)**:
```
❌ ConditionEvaluator.java (0/400 lines)
   - AND/OR logic for condition evaluation
   - Nested condition support
   - Operator handling (>=, <=, ==, !=, CONTAINS)

❌ ConfidenceCalculator.java (0/300 lines)
   - Base confidence score calculation
   - Modifier application (age, severity, comorbidities)
   - Activation threshold enforcement

❌ MedicationSelector.java (0/650 lines)
   - Rule-based medication selection
   - Allergy checking with alternatives
   - Renal/hepatic dose adjustments
   - MDR risk assessment

❌ TimeConstraintTracker.java (0/500 lines)
   - Hour-0/1/3 bundle tracking
   - Deadline calculation with offset_minutes
   - WARNING/CRITICAL alert generation
   - Bundle compliance monitoring

❌ KnowledgeBaseManager.java (0/750 lines)
   - Singleton pattern implementation
   - Category/specialty indexes for fast lookup
   - Hot reload capability
   - Thread-safe caching

❌ EscalationRuleEvaluator.java (0/350 lines)
   - Auto-escalation trigger evaluation
   - ICU transfer recommendation generation
   - Clinical deterioration detection

❌ ProtocolValidator.java (0/250 lines)
   - YAML schema validation
   - Error reporting with line numbers
   - Completeness checking (required fields)
```

### Target CDS Specification

**From cds.txt, Clinical_Knowledge_Base_Structure.txt, ProtocolLoader.txt**:

```java
// Complete runtime intelligence with 7 new components
public class ClinicalRecommendationEngine {

    // PHASE 1: Critical Runtime Logic
    private ConditionEvaluator conditionEvaluator;      // Trigger evaluation
    private MedicationSelector medicationSelector;      // Safe medication selection
    private TimeConstraintTracker timeTracker;          // Deadline enforcement

    // PHASE 2: Quality & Performance
    private ConfidenceCalculator confidenceCalculator;  // Protocol ranking
    private ProtocolValidator protocolValidator;        // Schema validation
    private KnowledgeBaseManager knowledgeBase;         // Fast indexed lookup

    // PHASE 3: Advanced Features
    private EscalationRuleEvaluator escalationEvaluator; // Auto-escalation

    public List<ClinicalRecommendation> generateRecommendations(
        EnrichedPatientContext context) {

        // 1. TRIGGER EVALUATION (NEW)
        List<Protocol> matchedProtocols = conditionEvaluator.evaluateTriggers(
            context, knowledgeBase.getAllProtocols());

        // 2. CONFIDENCE SCORING (NEW)
        Map<Protocol, Double> confidenceScores = confidenceCalculator.calculate(
            matchedProtocols, context);

        // 3. FILTER BY THRESHOLD (NEW)
        List<Protocol> activeProtocols = filterByConfidenceThreshold(
            matchedProtocols, confidenceScores);

        // 4. MEDICATION SELECTION (NEW)
        for (Protocol protocol : activeProtocols) {
            List<ProtocolAction> selectedActions = medicationSelector.selectActions(
                protocol.getActions(), context);
        }

        // 5. TIME CONSTRAINT TRACKING (NEW)
        TimeConstraintStatus timeStatus = timeTracker.evaluateConstraints(
            activeProtocols, context);

        // 6. ESCALATION EVALUATION (NEW)
        List<EscalationRecommendation> escalations =
            escalationEvaluator.evaluateEscalation(context, activeProtocols);

        // 7. BUILD RECOMMENDATIONS
        return buildRecommendations(activeProtocols, selectedActions,
            confidenceScores, timeStatus, escalations);
    }
}
```

**Enhanced YAML Protocol Structure**:
```yaml
# Current basic structure (what we have)
protocol_id: "SEPSIS-BUNDLE-001"
name: "Sepsis Management Bundle"
category: "INFECTIOUS"
actions:
  - action_id: "SEPSIS-ACT-001"
    medication:
      name: "Ceftriaxone"
      dose: "2 g"

# Target enhanced structure (what we need)
protocol_id: "SEPSIS-BUNDLE-001"
name: "Sepsis Management Bundle"
category: "INFECTIOUS"

# NEW: Trigger evaluation logic
trigger_criteria:
  match_logic: "ALL_OF"  # AND logic
  conditions:
    - condition_id: "SEPSIS-TRIG-001"
      parameter: "lactate"
      operator: ">="
      threshold: 2.0
      unit: "mmol/L"
    - condition_id: "SEPSIS-TRIG-002"
      match_logic: "ANY_OF"  # Nested OR logic
      conditions:
        - parameter: "systolic_bp"
          operator: "<"
          threshold: 90
        - parameter: "map"
          operator: "<"
          threshold: 65

# NEW: Confidence scoring
confidence_scoring:
  base_confidence: 0.85
  modifiers:
    - modifier_id: "SEPSIS-CONF-001"
      condition:
        parameter: "white_blood_count"
        operator: ">="
        threshold: 12000
      adjustment: 0.10
    - modifier_id: "SEPSIS-CONF-002"
      condition:
        parameter: "age"
        operator: ">="
        threshold: 65
      adjustment: 0.05
  activation_threshold: 0.70

# NEW: Medication selection algorithm
actions:
  - action_id: "SEPSIS-ACT-001"
    medication_selection:
      selection_criteria:
        - criteria_id: "NO_PENICILLIN_ALLERGY"
          primary_medication:
            name: "Ceftriaxone"
            dose: "2 g"
          alternative_medication:
            name: "Levofloxacin"
            dose: "750 mg"
            indication: "Penicillin allergy"

# NEW: Time constraint enforcement
time_constraints:
  - constraint_id: "SEPSIS-TIME-001"
    bundle_name: "Hour-1 Bundle"
    offset_minutes: 60
    critical: true
    actions:
      - "Blood cultures before antibiotics"
      - "Broad-spectrum antibiotics"
      - "Lactate measurement"

# NEW: Special populations
special_populations:
  - population_id: "ELDERLY"
    age_range: ">= 65"
    modifications:
      - action_id: "SEPSIS-ACT-001"
        dose_adjustment: "Consider 1 g if CrCl < 30"

# NEW: Escalation rules
escalation_rules:
  - rule_id: "SEPSIS-ESC-001"
    escalation_trigger:
      match_logic: "ANY_OF"
      conditions:
        - parameter: "lactate"
          operator: ">="
          threshold: 4.0
        - parameter: "systolic_bp"
          operator: "<"
          threshold: 90
    recommendation:
      escalation_level: "ICU_TRANSFER"
      rationale: "Septic shock requiring vasopressor support"
```

---

## Architecture Changes Overview

### Before: Simple Protocol Lookup System
```
Patient Context → ProtocolMatcher (stub) → Empty List
                → ActionBuilder (basic) → Generic Actions
                → No validation, no scoring, no intelligence
```

### After: Intelligent CDS Engine
```
Patient Context
  ↓
  → ConditionEvaluator (evaluates ALL protocols' triggers)
  ↓
  → ConfidenceCalculator (scores each match 0.0-1.0)
  ↓
  → Filter by activation_threshold (≥0.70)
  ↓
  → MedicationSelector (checks allergies, selects alternatives)
  ↓
  → TimeConstraintTracker (tracks deadlines, generates alerts)
  ↓
  → EscalationRuleEvaluator (checks ICU transfer criteria)
  ↓
  → ClinicalRecommendation (ranked by confidence, time-aware, safe)
```

### Component Integration Diagram

```
┌─────────────────────────────────────────────────────────────┐
│         KnowledgeBaseManager (Singleton)                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Protocol Storage: ConcurrentHashMap<String, Protocol>│   │
│  │ Category Index: Map<Category, List<Protocol>>        │   │
│  │ Specialty Index: Map<Specialty, List<Protocol>>      │   │
│  │ Hot Reload: FileWatcher for YAML updates             │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     ProtocolValidator (At Load Time)                        │
│  - Schema validation (required fields present)              │
│  - Reference validation (action_ids match)                  │
│  - Error reporting with line numbers                        │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     ConditionEvaluator (Trigger Matching)                   │
│  - Evaluate ALL_OF (AND) and ANY_OF (OR) logic              │
│  - Handle operators: >=, <=, ==, !=, CONTAINS               │
│  - Nested condition support (recursive evaluation)          │
│  - Returns: List<Protocol> matchedProtocols                 │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     ConfidenceCalculator (Protocol Ranking)                 │
│  - Base confidence from protocol                            │
│  - Apply modifiers based on patient state                   │
│  - Filter by activation_threshold                           │
│  - Returns: Map<Protocol, Double> confidenceScores          │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     MedicationSelector (Safe Action Selection)              │
│  - Evaluate selection_criteria rules                        │
│  - Check allergies → select alternatives                    │
│  - Renal/hepatic dose adjustments                           │
│  - MDR risk assessment                                      │
│  - Returns: List<ProtocolAction> selectedActions            │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     TimeConstraintTracker (Deadline Management)             │
│  - Track hour-0/1/3 bundles                                 │
│  - Calculate deadlines: triggerTime + offset_minutes        │
│  - Generate WARNING (30min remain) / CRITICAL (overdue)     │
│  - Returns: TimeConstraintStatus with alerts                │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│     EscalationRuleEvaluator (Auto-Escalation)               │
│  - Evaluate escalation_triggers                             │
│  - Generate ICU transfer recommendations                    │
│  - Clinical deterioration detection                         │
│  - Returns: List<EscalationRecommendation>                  │
└─────────────────────────────────────────────────────────────┘
```

---

## Gap Analysis by Component

### 1. ProtocolLoader.java

**Current Implementation** (400 lines):
```java
public class ProtocolLoader {
    private static final String PROTOCOL_DIRECTORY = "clinical-protocols/";
    private static final String[] PROTOCOL_FILES = { /* 16 files */ };

    public static Map<String, Protocol> loadProtocols() {
        Map<String, Protocol> protocols = new HashMap<>();
        ObjectMapper mapper = new ObjectMapper(new YAMLFactory());

        for (String filename : PROTOCOL_FILES) {
            InputStream inputStream = ProtocolLoader.class
                .getClassLoader()
                .getResourceAsStream(PROTOCOL_DIRECTORY + filename);
            Protocol protocol = mapper.readValue(inputStream, Protocol.class);
            protocols.put(protocol.getProtocolId(), protocol);
        }
        return protocols;
    }
}
```

**CDS Specification** (ProtocolLoader.txt - 1,605 lines):
```java
public class ProtocolLoader {
    private static final Logger logger = LoggerFactory.getLogger(ProtocolLoader.class);
    private ProtocolValidator validator;  // NEW

    public Map<String, Protocol> loadProtocols() throws ProtocolLoadException {
        Map<String, Protocol> protocols = new HashMap<>();
        ObjectMapper mapper = configureObjectMapper();  // NEW: Custom configuration

        for (String filename : PROTOCOL_FILES) {
            try {
                Protocol protocol = loadProtocol(filename, mapper);

                // NEW: Validation step
                ValidationResult validationResult = validator.validate(protocol);
                if (!validationResult.isValid()) {
                    logger.error("Protocol {} failed validation: {}",
                        filename, validationResult.getErrors());
                    throw new ProtocolLoadException(
                        "Invalid protocol: " + filename,
                        validationResult.getErrors());
                }

                protocols.put(protocol.getProtocolId(), protocol);
                logger.info("Loaded and validated protocol: {}", protocol.getName());

            } catch (IOException e) {
                logger.error("Failed to load protocol: {}", filename, e);
                throw new ProtocolLoadException("Failed to load: " + filename, e);
            }
        }

        // NEW: Cross-protocol validation
        validateProtocolReferences(protocols);

        return protocols;
    }

    // NEW: Custom ObjectMapper configuration
    private ObjectMapper configureObjectMapper() {
        ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
        mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        mapper.configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, true);
        mapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
        return mapper;
    }

    // NEW: Cross-protocol reference validation
    private void validateProtocolReferences(Map<String, Protocol> protocols)
        throws ProtocolLoadException {
        for (Protocol protocol : protocols.values()) {
            // Check action references
            for (ProtocolAction action : protocol.getActions()) {
                if (action.getActionId() == null || action.getActionId().isEmpty()) {
                    throw new ProtocolLoadException(
                        "Protocol " + protocol.getProtocolId() +
                        " has action with missing action_id");
                }
            }
        }
    }
}
```

**Gaps**:
- ❌ No validation before loading
- ❌ No error handling for malformed YAML
- ❌ No cross-protocol reference checking
- ❌ No logging of load success/failure
- ❌ No custom ObjectMapper configuration

**Effort**: 2-3 hours to add validation integration and error handling

---

### 2. ProtocolMatcher.java

**Current Implementation** (~150 lines - stub):
```java
public class ProtocolMatcher {
    public List<ClinicalRecommendation> matchProtocols(
        EnrichedPatientContext context,
        Map<String, Protocol> protocols) {

        // TODO: Implement trigger evaluation
        return new ArrayList<>(); // Returns empty list
    }
}
```

**CDS Specification** (cds.txt lines 801-1200):
```java
public class ProtocolMatcher {
    private ConditionEvaluator conditionEvaluator;  // NEW DEPENDENCY
    private ConfidenceCalculator confidenceCalculator;  // NEW DEPENDENCY

    public List<Protocol> matchProtocols(
        EnrichedPatientContext context,
        Map<String, Protocol> protocols) {

        List<Protocol> matchedProtocols = new ArrayList<>();

        for (Protocol protocol : protocols.values()) {
            // NEW: Evaluate trigger criteria
            if (protocol.getTriggerCriteria() != null) {
                boolean triggered = conditionEvaluator.evaluate(
                    protocol.getTriggerCriteria(),
                    context);

                if (triggered) {
                    // NEW: Calculate confidence score
                    double confidence = confidenceCalculator.calculateConfidence(
                        protocol,
                        context);

                    // NEW: Check activation threshold
                    double threshold = protocol.getConfidenceScoring() != null
                        ? protocol.getConfidenceScoring().getActivationThreshold()
                        : 0.70; // Default threshold

                    if (confidence >= threshold) {
                        matchedProtocols.add(protocol);
                        logger.debug("Protocol {} matched with confidence {}",
                            protocol.getProtocolId(), confidence);
                    } else {
                        logger.debug("Protocol {} triggered but confidence {} below threshold {}",
                            protocol.getProtocolId(), confidence, threshold);
                    }
                }
            }
        }

        // NEW: Sort by confidence (highest first)
        matchedProtocols.sort((p1, p2) -> {
            double conf1 = confidenceCalculator.calculateConfidence(p1, context);
            double conf2 = confidenceCalculator.calculateConfidence(p2, context);
            return Double.compare(conf2, conf1); // Descending order
        });

        return matchedProtocols;
    }
}
```

**Gaps**:
- ❌ No trigger evaluation logic (critical gap)
- ❌ No confidence scoring
- ❌ No filtering by activation threshold
- ❌ No sorting by confidence
- ❌ Missing dependencies: ConditionEvaluator, ConfidenceCalculator

**Effort**: 1-2 hours to integrate new components (components themselves: 5-8 hours)

---

### 3. ActionBuilder.java

**Current Implementation** (~200 lines - basic):
```java
public class ActionBuilder {
    public List<ProtocolAction> buildActions(
        Protocol protocol,
        EnrichedPatientContext context) {

        List<ProtocolAction> actions = new ArrayList<>();

        // Basic contraindication checking
        for (ProtocolAction action : protocol.getActions()) {
            boolean contraindicated = checkContraindications(action, context);
            if (!contraindicated) {
                actions.add(action);
            }
        }

        return actions;
    }

    private boolean checkContraindications(
        ProtocolAction action,
        EnrichedPatientContext context) {
        // Basic implementation exists
        return false;
    }
}
```

**CDS Specification** (cds.txt lines 1201-1600):
```java
public class ActionBuilder {
    private MedicationSelector medicationSelector;  // NEW DEPENDENCY
    private TimeConstraintTracker timeTracker;  // NEW DEPENDENCY

    public List<ProtocolAction> buildActions(
        Protocol protocol,
        EnrichedPatientContext context) {

        List<ProtocolAction> selectedActions = new ArrayList<>();

        for (ProtocolAction action : protocol.getActions()) {

            // NEW: Medication selection algorithm
            if (action.getMedicationSelection() != null) {
                ProtocolAction selectedAction = medicationSelector.selectMedication(
                    action,
                    context);

                // NEW: Apply dose adjustments
                selectedAction = applyDoseAdjustments(selectedAction, context);

                // NEW: Check contraindications with alternatives
                if (hasContraindication(selectedAction, context)) {
                    ProtocolAction alternative = findAlternativeAction(
                        action,
                        context);
                    if (alternative != null) {
                        selectedActions.add(alternative);
                        logger.info("Using alternative action due to contraindication: {}",
                            alternative.getActionId());
                    } else {
                        logger.warn("No alternative available for contraindicated action: {}",
                            action.getActionId());
                    }
                } else {
                    selectedActions.add(selectedAction);
                }
            } else {
                // No medication selection - use action as-is
                selectedActions.add(action);
            }
        }

        // NEW: Apply time constraints
        selectedActions = timeTracker.applyTimeConstraints(
            selectedActions,
            protocol.getTimeConstraints(),
            context);

        return selectedActions;
    }

    // NEW: Dose adjustment algorithms
    private ProtocolAction applyDoseAdjustments(
        ProtocolAction action,
        EnrichedPatientContext context) {

        // Renal dose adjustment (Cockcroft-Gault)
        if (action.getMedication().getDoseAdjustment() != null) {
            DoseAdjustment adjustment = action.getMedication().getDoseAdjustment();

            if (adjustment.getRenalAdjustment() != null) {
                double creatinineClearance = calculateCrCl(context);
                String adjustedDose = adjustment.getRenalAdjustment()
                    .getDoseForClearance(creatinineClearance);

                if (adjustedDose != null) {
                    action.getMedication().setDose(adjustedDose);
                    logger.info("Applied renal dose adjustment: {} (CrCl: {})",
                        adjustedDose, creatinineClearance);
                }
            }

            // Hepatic dose adjustment (Child-Pugh)
            if (adjustment.getHepaticAdjustment() != null) {
                String childPughScore = context.getPatientState().getChildPughScore();
                String adjustedDose = adjustment.getHepaticAdjustment()
                    .getDoseForChildPugh(childPughScore);

                if (adjustedDose != null) {
                    action.getMedication().setDose(adjustedDose);
                    logger.info("Applied hepatic dose adjustment: {} (Child-Pugh: {})",
                        adjustedDose, childPughScore);
                }
            }
        }

        return action;
    }

    // NEW: Alternative action selection
    private ProtocolAction findAlternativeAction(
        ProtocolAction action,
        EnrichedPatientContext context) {

        if (action.getMedicationSelection() != null) {
            for (SelectionCriteria criteria : action.getMedicationSelection().getSelectionCriteria()) {
                if (criteria.getAlternativeMedication() != null) {
                    // Check if alternative is contraindicated
                    ProtocolAction altAction = createActionFromMedication(
                        criteria.getAlternativeMedication());

                    if (!hasContraindication(altAction, context)) {
                        return altAction;
                    }
                }
            }
        }

        return null; // No safe alternative found
    }
}
```

**Gaps**:
- ❌ No medication selection algorithm (critical for safety)
- ❌ No dose adjustments (renal/hepatic)
- ❌ No alternative action selection when contraindicated
- ❌ No time constraint application
- ❌ Missing dependencies: MedicationSelector, TimeConstraintTracker

**Effort**: 2-3 hours to integrate new components (components themselves: 6-8 hours)

---

### 4. Missing Components Summary

| Component | Current Status | Target Lines | Critical? | Effort |
|-----------|---------------|--------------|-----------|--------|
| ConditionEvaluator.java | ❌ Missing | 400 | **YES** | 3-4 hours |
| ConfidenceCalculator.java | ❌ Missing | 300 | High | 2-3 hours |
| MedicationSelector.java | ❌ Missing | 650 | **YES** | 4-5 hours |
| TimeConstraintTracker.java | ❌ Missing | 500 | **YES** | 3-4 hours |
| KnowledgeBaseManager.java | ❌ Missing | 750 | High | 4-5 hours |
| EscalationRuleEvaluator.java | ❌ Missing | 350 | Medium | 2-3 hours |
| ProtocolValidator.java | ❌ Missing | 250 | High | 2 hours |
| **TOTAL** | **0/3,200 lines** | **3,200** | - | **20-27 hours** |

**Critical Path**: ConditionEvaluator + MedicationSelector + TimeConstraintTracker = **10-13 hours**

---

## New Java Classes Required

### 1. ConditionEvaluator.java
**Purpose**: Evaluate trigger criteria with AND/OR logic to determine if protocol should activate

**Key Responsibilities**:
- Evaluate ALL_OF (AND) and ANY_OF (OR) match logic
- Handle nested conditions recursively
- Support operators: >=, <=, ==, !=, CONTAINS
- Extract parameter values from EnrichedPatientContext

**Core Algorithm**:
```java
public boolean evaluate(TriggerCriteria trigger, EnrichedPatientContext context) {
    if (trigger.getMatchLogic() == MatchLogic.ALL_OF) {
        // AND logic - all conditions must be true
        for (Condition condition : trigger.getConditions()) {
            if (!evaluateCondition(condition, context)) {
                return false; // Short-circuit on first false
            }
        }
        return true;
    } else {
        // OR logic - at least one condition must be true
        for (Condition condition : trigger.getConditions()) {
            if (evaluateCondition(condition, context)) {
                return true; // Short-circuit on first true
            }
        }
        return false;
    }
}

private boolean evaluateCondition(Condition condition, EnrichedPatientContext context) {
    // Handle nested conditions (recursive)
    if (condition.getConditions() != null && !condition.getConditions().isEmpty()) {
        TriggerCriteria nestedTrigger = new TriggerCriteria();
        nestedTrigger.setMatchLogic(condition.getMatchLogic());
        nestedTrigger.setConditions(condition.getConditions());
        return evaluate(nestedTrigger, context); // RECURSION
    }

    // Leaf condition - extract value and compare
    Object actualValue = extractParameterValue(condition.getParameter(), context);
    Object expectedValue = condition.getThreshold();

    return compareValues(actualValue, expectedValue, condition.getOperator());
}
```

**Dependencies**: EnrichedPatientContext, Protocol.TriggerCriteria, Protocol.Condition

**Estimated Lines**: 400
**Estimated Effort**: 3-4 hours
**Priority**: **CRITICAL** (Phase 1)

---

### 2. ConfidenceCalculator.java
**Purpose**: Calculate confidence score for protocol match based on patient state

**Key Responsibilities**:
- Start with base confidence from protocol
- Apply modifiers based on patient conditions
- Enforce activation threshold
- Rank multiple matching protocols

**Core Algorithm**:
```java
public double calculateConfidence(Protocol protocol, EnrichedPatientContext context) {
    if (protocol.getConfidenceScoring() == null) {
        return 0.85; // Default confidence
    }

    ConfidenceScoring scoring = protocol.getConfidenceScoring();
    double confidence = scoring.getBaseConfidence();

    // Apply modifiers
    for (ConfidenceModifier modifier : scoring.getModifiers()) {
        if (conditionEvaluator.evaluateCondition(modifier.getCondition(), context)) {
            confidence += modifier.getAdjustment();
            logger.debug("Applied modifier {}: {} (new confidence: {})",
                modifier.getModifierId(), modifier.getAdjustment(), confidence);
        }
    }

    // Clamp to [0.0, 1.0]
    confidence = Math.max(0.0, Math.min(1.0, confidence));

    return confidence;
}

public boolean meetsActivationThreshold(Protocol protocol, double confidence) {
    double threshold = protocol.getConfidenceScoring() != null
        ? protocol.getConfidenceScoring().getActivationThreshold()
        : 0.70; // Default threshold

    return confidence >= threshold;
}
```

**Dependencies**: ConditionEvaluator (for modifier evaluation)

**Estimated Lines**: 300
**Estimated Effort**: 2-3 hours
**Priority**: High (Phase 2)

---

### 3. MedicationSelector.java
**Purpose**: Select appropriate medication based on patient allergies, renal function, MDR risk

**Key Responsibilities**:
- Evaluate selection criteria rules
- Check allergies → select alternatives
- Apply renal/hepatic dose adjustments
- Assess MDR (multi-drug resistant) risk

**Core Algorithm**:
```java
public ProtocolAction selectMedication(
    ProtocolAction action,
    EnrichedPatientContext context) {

    if (action.getMedicationSelection() == null) {
        return action; // No selection needed
    }

    MedicationSelection selection = action.getMedicationSelection();

    for (SelectionCriteria criteria : selection.getSelectionCriteria()) {
        boolean criteriaMet = evaluateCriteria(criteria.getCriteriaId(), context);

        if (criteriaMet) {
            // Use primary medication
            Medication selectedMed = criteria.getPrimaryMedication();

            // Check for contraindications/allergies
            if (hasAllergy(selectedMed, context)) {
                logger.warn("Patient allergic to {}, using alternative",
                    selectedMed.getName());
                selectedMed = criteria.getAlternativeMedication();
            }

            // Apply dose adjustments
            selectedMed = applyDoseAdjustments(selectedMed, context);

            // Create new action with selected medication
            ProtocolAction selectedAction = action.clone();
            selectedAction.setMedication(selectedMed);
            return selectedAction;
        }
    }

    // No criteria met - use default medication
    return action;
}

private boolean evaluateCriteria(String criteriaId, EnrichedPatientContext context) {
    switch (criteriaId) {
        case "NO_PENICILLIN_ALLERGY":
            return !context.getPatientState().getAllergies().contains("penicillin");

        case "CREATININE_CLEARANCE_GT_40":
            double crCl = calculateCrCl(context);
            return crCl > 40.0;

        case "MDR_RISK":
            return context.getPatientState().hasMDRRiskFactors();

        case "NO_BETA_BLOCKER_CONTRAINDICATION":
            return !hasBetaBlockerContraindication(context);

        default:
            logger.warn("Unknown criteria: {}", criteriaId);
            return false;
    }
}

private Medication applyDoseAdjustments(
    Medication medication,
    EnrichedPatientContext context) {

    // Renal adjustment (Cockcroft-Gault formula)
    double crCl = calculateCrCl(context);
    if (crCl < 60.0) {
        medication = adjustDoseForRenalFunction(medication, crCl);
    }

    // Hepatic adjustment (Child-Pugh score)
    String childPugh = context.getPatientState().getChildPughScore();
    if (childPugh.equals("B") || childPugh.equals("C")) {
        medication = adjustDoseForHepaticFunction(medication, childPugh);
    }

    return medication;
}

private double calculateCrCl(EnrichedPatientContext context) {
    // Cockcroft-Gault formula
    double age = context.getPatientState().getAge();
    double weight = context.getPatientState().getWeight();
    double creatinine = context.getPatientState().getCreatinine();
    boolean isFemale = context.getPatientState().getSex().equals("F");

    double crCl = ((140 - age) * weight) / (72 * creatinine);
    if (isFemale) {
        crCl *= 0.85;
    }

    return crCl;
}
```

**Dependencies**: EnrichedPatientContext.PatientState (allergies, labs, demographics)

**Estimated Lines**: 650
**Estimated Effort**: 4-5 hours
**Priority**: **CRITICAL** (Phase 1) - Required for patient safety

---

### 4. TimeConstraintTracker.java
**Purpose**: Track time-sensitive interventions and generate deadline alerts

**Key Responsibilities**:
- Track hour-0/1/3 bundles (sepsis, STEMI)
- Calculate deadlines: triggerTime + offset_minutes
- Generate WARNING (30min remaining), CRITICAL (deadline exceeded) alerts
- Monitor bundle compliance

**Core Algorithm**:
```java
public TimeConstraintStatus evaluateConstraints(
    Protocol protocol,
    EnrichedPatientContext context) {

    TimeConstraintStatus status = new TimeConstraintStatus();
    Instant triggerTime = context.getTriggerTime();
    Instant currentTime = Instant.now();

    if (protocol.getTimeConstraints() == null) {
        return status; // No constraints
    }

    for (TimeConstraint constraint : protocol.getTimeConstraints()) {
        Instant deadline = triggerTime.plus(constraint.getOffsetMinutes(), ChronoUnit.MINUTES);
        Duration timeRemaining = Duration.between(currentTime, deadline);

        ConstraintStatus constraintStatus = new ConstraintStatus();
        constraintStatus.setConstraintId(constraint.getConstraintId());
        constraintStatus.setBundleName(constraint.getBundleName());
        constraintStatus.setDeadline(deadline);
        constraintStatus.setTimeRemaining(timeRemaining);

        // Determine alert level
        if (timeRemaining.isNegative()) {
            // Deadline exceeded
            constraintStatus.setAlertLevel(AlertLevel.CRITICAL);
            constraintStatus.setMessage(String.format(
                "%s deadline exceeded by %d minutes",
                constraint.getBundleName(),
                Math.abs(timeRemaining.toMinutes())
            ));
        } else if (timeRemaining.toMinutes() <= 30) {
            // Within 30 minutes of deadline
            constraintStatus.setAlertLevel(AlertLevel.WARNING);
            constraintStatus.setMessage(String.format(
                "%s deadline in %d minutes",
                constraint.getBundleName(),
                timeRemaining.toMinutes()
            ));
        } else {
            // On track
            constraintStatus.setAlertLevel(AlertLevel.INFO);
            constraintStatus.setMessage(String.format(
                "%s: %d minutes remaining",
                constraint.getBundleName(),
                timeRemaining.toMinutes()
            ));
        }

        status.addConstraintStatus(constraintStatus);

        // Log critical deadlines
        if (constraint.isCritical() && timeRemaining.toMinutes() <= 30) {
            logger.warn("CRITICAL TIME CONSTRAINT: {}", constraintStatus.getMessage());
        }
    }

    return status;
}

public class TimeConstraintStatus {
    private List<ConstraintStatus> constraintStatuses = new ArrayList<>();

    public boolean hasCriticalAlerts() {
        return constraintStatuses.stream()
            .anyMatch(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL);
    }

    public List<ConstraintStatus> getCriticalAlerts() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL)
            .collect(Collectors.toList());
    }
}
```

**Dependencies**: Protocol.TimeConstraint, Java Time API

**Estimated Lines**: 500
**Estimated Effort**: 3-4 hours
**Priority**: **CRITICAL** (Phase 1) - Required for sepsis bundles, STEMI door-to-balloon

---

### 5. KnowledgeBaseManager.java
**Purpose**: Singleton pattern for protocol storage with fast indexed lookup

**Key Responsibilities**:
- Thread-safe protocol caching
- Category/specialty indexes for fast filtering
- Hot reload capability (watch YAML files for changes)
- Protocol query methods (getByCategory, getBySpecialty, search)

**Core Architecture**:
```java
public class KnowledgeBaseManager {
    private static final Logger logger = LoggerFactory.getLogger(KnowledgeBaseManager.class);
    private static volatile KnowledgeBaseManager instance;

    // Thread-safe storage
    private final ConcurrentHashMap<String, Protocol> protocols;
    private final Map<ProtocolCategory, List<Protocol>> categoryIndex;
    private final Map<String, List<Protocol>> specialtyIndex;

    // Hot reload support
    private final WatchService watchService;
    private final Path protocolDirectory;
    private volatile boolean isReloading = false;

    private KnowledgeBaseManager() {
        this.protocols = new ConcurrentHashMap<>();
        this.categoryIndex = new ConcurrentHashMap<>();
        this.specialtyIndex = new ConcurrentHashMap<>();
        this.watchService = initializeWatchService();
        this.protocolDirectory = Paths.get("clinical-protocols");

        loadAllProtocols();
        startWatchService();
    }

    public static KnowledgeBaseManager getInstance() {
        if (instance == null) {
            synchronized (KnowledgeBaseManager.class) {
                if (instance == null) {
                    instance = new KnowledgeBaseManager();
                }
            }
        }
        return instance;
    }

    private void loadAllProtocols() {
        try {
            Map<String, Protocol> loadedProtocols = ProtocolLoader.loadProtocols();

            // Clear existing data
            protocols.clear();
            categoryIndex.clear();
            specialtyIndex.clear();

            // Add to main storage
            protocols.putAll(loadedProtocols);

            // Build indexes
            buildIndexes();

            logger.info("Loaded {} protocols", protocols.size());

        } catch (Exception e) {
            logger.error("Failed to load protocols", e);
            throw new RuntimeException("Protocol loading failed", e);
        }
    }

    private void buildIndexes() {
        // Category index
        for (Protocol protocol : protocols.values()) {
            categoryIndex.computeIfAbsent(
                protocol.getCategory(),
                k -> new CopyOnWriteArrayList<>()
            ).add(protocol);

            // Specialty index
            if (protocol.getSpecialty() != null) {
                specialtyIndex.computeIfAbsent(
                    protocol.getSpecialty(),
                    k -> new CopyOnWriteArrayList<>()
                ).add(protocol);
            }
        }

        logger.info("Built indexes: {} categories, {} specialties",
            categoryIndex.size(), specialtyIndex.size());
    }

    // Query methods
    public Protocol getProtocol(String protocolId) {
        return protocols.get(protocolId);
    }

    public List<Protocol> getAllProtocols() {
        return new ArrayList<>(protocols.values());
    }

    public List<Protocol> getByCategory(ProtocolCategory category) {
        return categoryIndex.getOrDefault(category, Collections.emptyList());
    }

    public List<Protocol> getBySpecialty(String specialty) {
        return specialtyIndex.getOrDefault(specialty, Collections.emptyList());
    }

    public List<Protocol> search(String query) {
        String lowerQuery = query.toLowerCase();
        return protocols.values().stream()
            .filter(p ->
                p.getName().toLowerCase().contains(lowerQuery) ||
                p.getProtocolId().toLowerCase().contains(lowerQuery) ||
                p.getCategory().name().toLowerCase().contains(lowerQuery)
            )
            .collect(Collectors.toList());
    }

    // Hot reload
    private void startWatchService() {
        Thread watchThread = new Thread(() -> {
            while (true) {
                try {
                    WatchKey key = watchService.take();

                    for (WatchEvent<?> event : key.pollEvents()) {
                        if (event.kind() == StandardWatchEventKinds.ENTRY_MODIFY) {
                            Path changed = (Path) event.context();
                            if (changed.toString().endsWith(".yaml")) {
                                logger.info("Protocol file changed: {}", changed);
                                reloadProtocols();
                            }
                        }
                    }

                    key.reset();

                } catch (InterruptedException e) {
                    logger.error("Watch service interrupted", e);
                    Thread.currentThread().interrupt();
                    break;
                }
            }
        });

        watchThread.setDaemon(true);
        watchThread.setName("ProtocolWatcher");
        watchThread.start();
    }

    public synchronized void reloadProtocols() {
        if (isReloading) {
            logger.warn("Reload already in progress, skipping");
            return;
        }

        try {
            isReloading = true;
            logger.info("Starting protocol reload...");

            loadAllProtocols();

            logger.info("Protocol reload completed successfully");

        } catch (Exception e) {
            logger.error("Protocol reload failed", e);
        } finally {
            isReloading = false;
        }
    }
}
```

**Dependencies**: ProtocolLoader, FileWatcher (Java NIO)

**Estimated Lines**: 750
**Estimated Effort**: 4-5 hours
**Priority**: High (Phase 2) - Performance optimization

---

### 6. EscalationRuleEvaluator.java
**Purpose**: Evaluate escalation triggers and generate ICU transfer recommendations

**Key Responsibilities**:
- Evaluate escalation_rules from protocol
- Detect clinical deterioration
- Generate ICU transfer recommendations with rationale
- Support multiple escalation levels (CONSULT, TRANSFER, IMMEDIATE_TRANSFER)

**Core Algorithm**:
```java
public List<EscalationRecommendation> evaluateEscalation(
    Protocol protocol,
    EnrichedPatientContext context) {

    List<EscalationRecommendation> escalations = new ArrayList<>();

    if (protocol.getEscalationRules() == null) {
        return escalations;
    }

    for (EscalationRule rule : protocol.getEscalationRules()) {
        // Evaluate escalation trigger
        boolean triggered = conditionEvaluator.evaluate(
            rule.getEscalationTrigger(),
            context);

        if (triggered) {
            EscalationRecommendation escalation = new EscalationRecommendation();
            escalation.setRuleId(rule.getRuleId());
            escalation.setProtocolId(protocol.getProtocolId());
            escalation.setEscalationLevel(rule.getRecommendation().getEscalationLevel());
            escalation.setRationale(rule.getRecommendation().getRationale());
            escalation.setTimestamp(Instant.now());

            // Add clinical evidence
            escalation.setClinicalEvidence(gatherClinicalEvidence(rule, context));

            escalations.add(escalation);

            logger.warn("ESCALATION TRIGGERED: {} - {} - {}",
                protocol.getProtocolId(),
                rule.getRecommendation().getEscalationLevel(),
                rule.getRecommendation().getRationale());
        }
    }

    // Sort by escalation level (IMMEDIATE_TRANSFER first)
    escalations.sort(Comparator.comparing(
        EscalationRecommendation::getEscalationLevel));

    return escalations;
}

private Map<String, Object> gatherClinicalEvidence(
    EscalationRule rule,
    EnrichedPatientContext context) {

    Map<String, Object> evidence = new HashMap<>();

    // Extract relevant clinical parameters from trigger
    for (Condition condition : rule.getEscalationTrigger().getConditions()) {
        String parameter = condition.getParameter();
        Object value = context.getPatientState().getParameter(parameter);
        evidence.put(parameter, value);
    }

    return evidence;
}

public enum EscalationLevel {
    CONSULT,            // Specialist consultation recommended
    TRANSFER,           // ICU transfer recommended
    IMMEDIATE_TRANSFER  // Immediate ICU transfer required
}
```

**Dependencies**: ConditionEvaluator, Protocol.EscalationRule

**Estimated Lines**: 350
**Estimated Effort**: 2-3 hours
**Priority**: Medium (Phase 3) - Advanced feature

---

### 7. ProtocolValidator.java
**Purpose**: Validate protocol YAML structure and completeness

**Key Responsibilities**:
- Schema validation (required fields present)
- Reference validation (action_ids, condition_ids match)
- Completeness checking (evidence_source, contraindications)
- Error reporting with line numbers

**Core Algorithm**:
```java
public class ProtocolValidator {

    public ValidationResult validate(Protocol protocol) {
        ValidationResult result = new ValidationResult();

        // Required field validation
        validateRequiredFields(protocol, result);

        // Reference validation
        validateActionReferences(protocol, result);
        validateConditionReferences(protocol, result);

        // Completeness validation
        validateEvidenceSource(protocol, result);
        validateContraindications(protocol, result);

        // Logical validation
        validateConfidenceScoring(protocol, result);
        validateTimeConstraints(protocol, result);

        return result;
    }

    private void validateRequiredFields(Protocol protocol, ValidationResult result) {
        if (protocol.getProtocolId() == null || protocol.getProtocolId().isEmpty()) {
            result.addError("protocol_id is required");
        }

        if (protocol.getName() == null || protocol.getName().isEmpty()) {
            result.addError("name is required");
        }

        if (protocol.getCategory() == null) {
            result.addError("category is required");
        }

        if (protocol.getActions() == null || protocol.getActions().isEmpty()) {
            result.addError("At least one action is required");
        }
    }

    private void validateActionReferences(Protocol protocol, ValidationResult result) {
        Set<String> actionIds = new HashSet<>();

        for (ProtocolAction action : protocol.getActions()) {
            if (action.getActionId() == null || action.getActionId().isEmpty()) {
                result.addError("Action missing action_id");
            } else {
                if (actionIds.contains(action.getActionId())) {
                    result.addError("Duplicate action_id: " + action.getActionId());
                }
                actionIds.add(action.getActionId());
            }
        }
    }

    private void validateConfidenceScoring(Protocol protocol, ValidationResult result) {
        if (protocol.getConfidenceScoring() != null) {
            ConfidenceScoring scoring = protocol.getConfidenceScoring();

            if (scoring.getBaseConfidence() < 0.0 || scoring.getBaseConfidence() > 1.0) {
                result.addError("base_confidence must be between 0.0 and 1.0");
            }

            if (scoring.getActivationThreshold() < 0.0 || scoring.getActivationThreshold() > 1.0) {
                result.addError("activation_threshold must be between 0.0 and 1.0");
            }

            // Validate modifiers don't exceed bounds
            double maxPossible = scoring.getBaseConfidence();
            for (ConfidenceModifier mod : scoring.getModifiers()) {
                maxPossible += mod.getAdjustment();
            }

            if (maxPossible > 1.5) {
                result.addWarning("Confidence modifiers may exceed 1.0 (max possible: " + maxPossible + ")");
            }
        }
    }

    private void validateEvidenceSource(Protocol protocol, ValidationResult result) {
        if (protocol.getEvidenceSource() == null) {
            result.addWarning("evidence_source recommended for clinical validation");
        } else {
            if (protocol.getEvidenceSource().getPrimaryGuideline() == null) {
                result.addWarning("primary_guideline recommended");
            }
            if (protocol.getEvidenceSource().getEvidenceLevel() == null) {
                result.addWarning("evidence_level recommended");
            }
        }
    }
}

public class ValidationResult {
    private List<String> errors = new ArrayList<>();
    private List<String> warnings = new ArrayList<>();

    public boolean isValid() {
        return errors.isEmpty();
    }

    public void addError(String error) {
        errors.add(error);
    }

    public void addWarning(String warning) {
        warnings.add(warning);
    }

    public List<String> getErrors() {
        return errors;
    }

    public List<String> getWarnings() {
        return warnings;
    }
}
```

**Dependencies**: Protocol model classes

**Estimated Lines**: 250
**Estimated Effort**: 2 hours
**Priority**: High (Phase 2) - Quality assurance

---

## Enhanced Protocol YAML Structure

### Current Basic Structure (What We Have)
```yaml
protocol_id: "SEPSIS-BUNDLE-001"
name: "Sepsis Management Bundle"
category: "INFECTIOUS"
version: "1.0"
last_updated: "2024-01-15"

# Evidence base
evidence_source:
  primary_guideline: "Surviving Sepsis Campaign 2021"
  evidence_level: "STRONG"
  key_citations:
    - "Rhodes A, et al. Critical Care Med. 2017"

# Basic action list
actions:
  - action_id: "SEPSIS-ACT-001"
    type: "MEDICATION"
    priority: "CRITICAL"
    medication:
      name: "Ceftriaxone"
      dose: "2 g"
      route: "IV"
      frequency: "q24h"

  - action_id: "SEPSIS-ACT-002"
    type: "DIAGNOSTIC"
    priority: "CRITICAL"
    diagnostic:
      test_name: "Blood cultures"
      urgency: "STAT"
      timing: "Before antibiotics"

# Basic contraindications
contraindications:
  - contraindication_id: "SEPSIS-CONTRA-001"
    condition: "Severe penicillin allergy"
    severity: "CRITICAL"
    alternative_action:
      action_id: "SEPSIS-ACT-ALT-001"
```

### Target Enhanced Structure (What We Need)

```yaml
protocol_id: "SEPSIS-BUNDLE-001"
name: "Sepsis Management Bundle"
category: "INFECTIOUS"
specialty: "CRITICAL_CARE"
version: "2.0"
last_updated: "2025-01-15"

# ============================================================
# NEW SECTION 1: TRIGGER CRITERIA (Automatic Activation)
# ============================================================
trigger_criteria:
  match_logic: "ALL_OF"  # All conditions must be true (AND logic)
  conditions:
    # Lactate elevation
    - condition_id: "SEPSIS-TRIG-001"
      parameter: "lactate"
      operator: ">="
      threshold: 2.0
      unit: "mmol/L"
      source: "lab_results"

    # Hypotension OR elevated MAP
    - condition_id: "SEPSIS-TRIG-002"
      match_logic: "ANY_OF"  # At least one must be true (OR logic)
      conditions:
        - parameter: "systolic_bp"
          operator: "<"
          threshold: 90
          unit: "mmHg"
          source: "vital_signs"

        - parameter: "mean_arterial_pressure"
          operator: "<"
          threshold: 65
          unit: "mmHg"
          source: "vital_signs"

    # Infection suspected
    - condition_id: "SEPSIS-TRIG-003"
      parameter: "infection_suspected"
      operator: "=="
      threshold: true
      source: "clinical_assessment"

# ============================================================
# NEW SECTION 2: CONFIDENCE SCORING (Protocol Ranking)
# ============================================================
confidence_scoring:
  base_confidence: 0.85  # Starting confidence

  # Modifiers increase/decrease confidence based on patient state
  modifiers:
    - modifier_id: "SEPSIS-CONF-001"
      description: "Elevated white blood count increases confidence"
      condition:
        parameter: "white_blood_count"
        operator: ">="
        threshold: 12000
        unit: "cells/μL"
      adjustment: 0.10  # +10% confidence

    - modifier_id: "SEPSIS-CONF-002"
      description: "Elderly patients higher risk"
      condition:
        parameter: "age"
        operator: ">="
        threshold: 65
        unit: "years"
      adjustment: 0.05  # +5% confidence

    - modifier_id: "SEPSIS-CONF-003"
      description: "Procalcitonin >2.0 strongly suggests bacterial sepsis"
      condition:
        parameter: "procalcitonin"
        operator: ">="
        threshold: 2.0
        unit: "ng/mL"
      adjustment: 0.15  # +15% confidence

    - modifier_id: "SEPSIS-CONF-004"
      description: "Fever/hypothermia supports infection"
      condition:
        match_logic: "ANY_OF"
        conditions:
          - parameter: "temperature"
            operator: ">="
            threshold: 38.3
            unit: "°C"
          - parameter: "temperature"
            operator: "<="
            threshold: 36.0
            unit: "°C"
      adjustment: 0.08  # +8% confidence

  # Protocol only activates if confidence >= threshold
  activation_threshold: 0.70  # 70% confidence required

# ============================================================
# EXISTING SECTION: EVIDENCE BASE (Enhanced)
# ============================================================
evidence_source:
  primary_guideline: "Surviving Sepsis Campaign 2021"
  guideline_version: "SSC 2021"
  publication_date: "2021-10"
  evidence_level: "STRONG"  # GRADE system
  key_citations:
    - citation_id: "SSC-2021-001"
      authors: "Evans L, Rhodes A, Alhazzani W, et al."
      title: "Surviving Sepsis Campaign: International Guidelines for Management of Sepsis and Septic Shock 2021"
      journal: "Critical Care Medicine"
      year: 2021
      pmid: "34605781"
      doi: "10.1097/CCM.0000000000005337"

  last_review_date: "2024-06-01"
  next_review_date: "2025-06-01"

# ============================================================
# ENHANCED SECTION: ACTIONS WITH MEDICATION SELECTION
# ============================================================
actions:
  # Action 1: Antibiotic selection with allergy checking
  - action_id: "SEPSIS-ACT-001"
    type: "MEDICATION"
    priority: "CRITICAL"
    timing:
      window: "IMMEDIATE"
      max_delay_minutes: 60  # Hour-1 bundle

    # NEW: Medication selection algorithm
    medication_selection:
      selection_strategy: "RULE_BASED"

      selection_criteria:
        # Criteria 1: No penicillin allergy → use ceftriaxone
        - criteria_id: "NO_PENICILLIN_ALLERGY"
          description: "First-line therapy for patients without penicillin allergy"

          condition:
            parameter: "allergies"
            operator: "NOT_CONTAINS"
            threshold: "penicillin"

          primary_medication:
            name: "Ceftriaxone"
            dose: "2 g"
            route: "IV"
            frequency: "q24h"
            administration_instructions: "Infuse over 30 minutes"

          alternative_medication:
            name: "Levofloxacin"
            dose: "750 mg"
            route: "IV"
            frequency: "q24h"
            indication: "Severe penicillin allergy"
            administration_instructions: "Infuse over 90 minutes"

        # Criteria 2: Renal impairment → dose adjustment
        - criteria_id: "RENAL_IMPAIRMENT"
          description: "Dose adjustment for CrCl < 40"

          condition:
            parameter: "creatinine_clearance"
            operator: "<"
            threshold: 40.0
            unit: "mL/min"

          primary_medication:
            name: "Ceftriaxone"
            dose: "1 g"  # Reduced dose
            route: "IV"
            frequency: "q24h"
            administration_instructions: "Monitor renal function daily"

        # Criteria 3: MDR risk factors → broader coverage
        - criteria_id: "MDR_RISK"
          description: "Multi-drug resistant risk factors present"

          condition:
            match_logic: "ANY_OF"
            conditions:
              - parameter: "recent_hospitalization"
                operator: "=="
                threshold: true
              - parameter: "recent_antibiotics"
                operator: "=="
                threshold: true
              - parameter: "immunosuppressed"
                operator: "=="
                threshold: true

          primary_medication:
            name: "Meropenem"
            dose: "1 g"
            route: "IV"
            frequency: "q8h"
            indication: "MDR coverage"
            administration_instructions: "Infuse over 30 minutes"

  # Action 2: Blood cultures
  - action_id: "SEPSIS-ACT-002"
    type: "DIAGNOSTIC"
    priority: "CRITICAL"
    timing:
      window: "IMMEDIATE"
      max_delay_minutes: 60
      sequence: "BEFORE"  # Before antibiotics
      reference_action: "SEPSIS-ACT-001"

    diagnostic:
      test_name: "Blood cultures"
      sample_type: "blood"
      number_of_sets: 2
      collection_sites: ["peripheral", "central_line"]
      urgency: "STAT"
      instructions: "Collect before antibiotic administration"

  # Action 3: Lactate measurement
  - action_id: "SEPSIS-ACT-003"
    type: "DIAGNOSTIC"
    priority: "CRITICAL"
    timing:
      window: "IMMEDIATE"
      max_delay_minutes: 60

    diagnostic:
      test_name: "Lactate"
      sample_type: "arterial_blood"
      urgency: "STAT"
      repeat_interval_hours: 2
      repeat_condition:
        parameter: "initial_lactate"
        operator: ">="
        threshold: 2.0

  # Action 4: Fluid resuscitation
  - action_id: "SEPSIS-ACT-004"
    type: "MEDICATION"
    priority: "CRITICAL"
    timing:
      window: "IMMEDIATE"
      max_delay_minutes: 60

    medication:
      name: "Crystalloid (NS or LR)"
      dose: "30 mL/kg"
      route: "IV"
      administration_instructions: "Rapid bolus over first hour"

    # Contraindications for fluid bolus
    contraindications:
      - condition:
          parameter: "pulmonary_edema"
          operator: "=="
          threshold: true
        severity: "MODERATE"
        alternative_action: "Reduce bolus to 10 mL/kg and reassess"

# ============================================================
# NEW SECTION 3: TIME CONSTRAINTS (Bundle Tracking)
# ============================================================
time_constraints:
  # Hour-1 Bundle (Critical)
  - constraint_id: "SEPSIS-TIME-001"
    bundle_name: "Hour-1 Sepsis Bundle"
    description: "All interventions within 1 hour of recognition"
    offset_minutes: 60
    critical: true  # Missing this has mortality impact

    required_actions:
      - action_id: "SEPSIS-ACT-001"
        description: "Broad-spectrum antibiotics"
      - action_id: "SEPSIS-ACT-002"
        description: "Blood cultures (before antibiotics)"
      - action_id: "SEPSIS-ACT-003"
        description: "Lactate measurement"
      - action_id: "SEPSIS-ACT-004"
        description: "30 mL/kg crystalloid for hypotension or lactate ≥4"

    compliance_monitoring:
      track_completion: true
      alert_at_minutes: 30  # Warning at 30 min remaining
      critical_alert_at_minutes: 0  # Critical at deadline

  # Hour-3 Bundle (Important)
  - constraint_id: "SEPSIS-TIME-002"
    bundle_name: "Hour-3 Sepsis Bundle"
    description: "Reassessment and escalation if not improving"
    offset_minutes: 180
    critical: false

    required_actions:
      - action_id: "SEPSIS-ACT-REASSESS-001"
        description: "Repeat lactate if initial ≥2.0"
        type: "DIAGNOSTIC"
        diagnostic:
          test_name: "Lactate"
          urgency: "STAT"

      - action_id: "SEPSIS-ACT-REASSESS-002"
        description: "Reassess volume status and tissue perfusion"
        type: "CLINICAL_ASSESSMENT"

# ============================================================
# ENHANCED SECTION: CONTRAINDICATIONS WITH ALTERNATIVES
# ============================================================
contraindications:
  - contraindication_id: "SEPSIS-CONTRA-001"
    condition_description: "Severe penicillin allergy (anaphylaxis)"

    trigger_condition:
      match_logic: "ALL_OF"
      conditions:
        - parameter: "allergies"
          operator: "CONTAINS"
          threshold: "penicillin"
        - parameter: "allergy_severity"
          operator: "=="
          threshold: "SEVERE"

    severity: "CRITICAL"
    affected_actions:
      - "SEPSIS-ACT-001"  # Ceftriaxone

    alternative_action:
      action_id: "SEPSIS-ACT-ALT-001"
      type: "MEDICATION"
      medication:
        name: "Levofloxacin"
        dose: "750 mg"
        route: "IV"
        frequency: "q24h"
      rationale: "Fluoroquinolone alternative for beta-lactam allergy"

  - contraindication_id: "SEPSIS-CONTRA-002"
    condition_description: "Acute pulmonary edema (aggressive fluids contraindicated)"

    trigger_condition:
      parameter: "pulmonary_edema"
      operator: "=="
      threshold: true

    severity: "MODERATE"
    affected_actions:
      - "SEPSIS-ACT-004"  # Fluid bolus

    alternative_action:
      action_id: "SEPSIS-ACT-ALT-002"
      type: "MEDICATION"
      medication:
        name: "Norepinephrine"
        dose: "0.1 mcg/kg/min"
        route: "IV"
        titration: "Titrate to MAP ≥65 mmHg"
      rationale: "Early vasopressor support instead of aggressive fluids"

# ============================================================
# ENHANCED SECTION: MONITORING REQUIREMENTS
# ============================================================
monitoring_requirements:
  - monitoring_id: "SEPSIS-MON-001"
    parameter: "mean_arterial_pressure"
    target_range:
      min: 65
      max: null
      unit: "mmHg"
    frequency: "continuous"
    duration_hours: 24
    alert_condition:
      parameter: "mean_arterial_pressure"
      operator: "<"
      threshold: 65
    escalation_action:
      description: "Initiate vasopressor support"
      reference_action: "SEPSIS-ESC-001"

  - monitoring_id: "SEPSIS-MON-002"
    parameter: "lactate"
    target_range:
      min: null
      max: 2.0
      unit: "mmol/L"
    frequency: "q2h if elevated"
    duration_hours: 6
    alert_condition:
      match_logic: "ANY_OF"
      conditions:
        - parameter: "lactate"
          operator: ">="
          threshold: 4.0
        - parameter: "lactate_clearance"
          operator: "<"
          threshold: 10.0  # <10% clearance
    escalation_action:
      description: "Reassess resuscitation strategy, consider ICU"

# ============================================================
# NEW SECTION 4: SPECIAL POPULATIONS (Age/Pregnancy/etc)
# ============================================================
special_populations:
  # Elderly patients
  - population_id: "ELDERLY"
    description: "Patients ≥65 years"

    inclusion_criteria:
      parameter: "age"
      operator: ">="
      threshold: 65
      unit: "years"

    modifications:
      - action_id: "SEPSIS-ACT-001"
        modification_type: "DOSE_ADJUSTMENT"
        rationale: "Renal function declines with age"
        new_parameters:
          dose: "1 g if CrCl <30 mL/min"
          monitoring: "Monitor renal function daily"

      - action_id: "SEPSIS-ACT-004"
        modification_type: "CAUTION"
        rationale: "Increased risk of fluid overload"
        new_parameters:
          dose: "20 mL/kg initial bolus"
          monitoring: "Reassess volume status after 10 mL/kg"

  # Pregnancy
  - population_id: "PREGNANCY"
    description: "Pregnant patients"

    inclusion_criteria:
      parameter: "pregnancy_status"
      operator: "=="
      threshold: true

    modifications:
      - action_id: "SEPSIS-ACT-001"
        modification_type: "SAFETY_CONCERN"
        rationale: "Levofloxacin contraindicated in pregnancy"
        contraindications:
          - "Levofloxacin (use aztreonam instead)"
        alternative_medication:
          name: "Aztreonam"
          dose: "2 g"
          route: "IV"
          frequency: "q8h"

  # Immunocompromised
  - population_id: "IMMUNOCOMPROMISED"
    description: "Immunocompromised patients (neutropenic, transplant, etc)"

    inclusion_criteria:
      match_logic: "ANY_OF"
      conditions:
        - parameter: "absolute_neutrophil_count"
          operator: "<"
          threshold: 500
        - parameter: "transplant_recipient"
          operator: "=="
          threshold: true
        - parameter: "immunosuppressive_therapy"
          operator: "=="
          threshold: true

    modifications:
      - action_id: "SEPSIS-ACT-001"
        modification_type: "BROADER_COVERAGE"
        rationale: "Higher risk of resistant organisms and fungal infections"
        additional_medications:
          - name: "Meropenem"
            dose: "1 g"
            route: "IV"
            frequency: "q8h"
            indication: "Anti-pseudomonal coverage"

          - name: "Micafungin"
            dose: "100 mg"
            route: "IV"
            frequency: "q24h"
            indication: "Empiric antifungal if high risk"

# ============================================================
# NEW SECTION 5: ESCALATION RULES (Auto-Escalation)
# ============================================================
escalation_rules:
  # Septic shock requiring vasopressors
  - rule_id: "SEPSIS-ESC-001"
    escalation_trigger:
      match_logic: "ANY_OF"
      conditions:
        - parameter: "lactate"
          operator: ">="
          threshold: 4.0
          unit: "mmol/L"

        - match_logic: "ALL_OF"
          conditions:
            - parameter: "mean_arterial_pressure"
              operator: "<"
              threshold: 65
              unit: "mmHg"
            - parameter: "fluid_bolus_completed"
              operator: "=="
              threshold: true

    recommendation:
      escalation_level: "ICU_TRANSFER"
      urgency: "IMMEDIATE"
      rationale: "Septic shock requiring vasopressor support and invasive monitoring"

      required_interventions:
        - "Central venous access"
        - "Arterial line for continuous BP monitoring"
        - "Vasopressor initiation (norepinephrine)"
        - "Continuous lactate monitoring"

      specialist_consultation:
        - specialty: "CRITICAL_CARE"
          urgency: "STAT"

  # Persistent hypotension despite fluids
  - rule_id: "SEPSIS-ESC-002"
    escalation_trigger:
      match_logic: "ALL_OF"
      conditions:
        - parameter: "systolic_bp"
          operator: "<"
          threshold: 90
          unit: "mmHg"
          duration_minutes: 30

        - parameter: "fluid_volume_administered"
          operator: ">="
          threshold: 30.0
          unit: "mL/kg"

    recommendation:
      escalation_level: "CONSULT"
      urgency: "URGENT"
      rationale: "Persistent hypotension despite adequate fluid resuscitation"

      specialist_consultation:
        - specialty: "CRITICAL_CARE"
          urgency: "URGENT"
          recommendation: "Consider vasopressor initiation"

  # Multi-organ dysfunction
  - rule_id: "SEPSIS-ESC-003"
    escalation_trigger:
      description: "≥2 organ systems failing (SOFA score ≥2 per organ)"

      match_logic: "ANY_OF"
      conditions:
        - parameter: "sofa_score"
          operator: ">="
          threshold: 8

        - parameter: "number_of_failing_organs"
          operator: ">="
          threshold: 2

    recommendation:
      escalation_level: "ICU_TRANSFER"
      urgency: "IMMEDIATE"
      rationale: "Multi-organ dysfunction requiring intensive care"

      required_interventions:
        - "Organ support (ventilation, RRT, vasopressors)"
        - "Continuous monitoring"
        - "Source control assessment"

# ============================================================
# EXISTING SECTIONS (Unchanged)
# ============================================================
de_escalation_criteria:
  - criteria_id: "SEPSIS-DE-ESC-001"
    description: "Culture results available, narrow antibiotic spectrum"
    conditions:
      - parameter: "culture_results_available"
        operator: "=="
        threshold: true
      - parameter: "time_since_antibiotics"
        operator: ">="
        threshold: 48
        unit: "hours"
    recommendation:
      action: "Narrow antibiotic spectrum based on culture and susceptibility"
      rationale: "Antimicrobial stewardship"

outcome_tracking:
  primary_outcomes:
    - outcome_id: "SEPSIS-OUT-001"
      metric: "mortality_28_day"
      target: "<15%"
      benchmark_source: "SSC 2021"

    - outcome_id: "SEPSIS-OUT-002"
      metric: "hour_1_bundle_compliance"
      target: ">80%"
      benchmark_source: "SSC 2021"

  process_measures:
    - measure_id: "SEPSIS-PROC-001"
      description: "Time to antibiotic administration"
      target: "<60 minutes"

    - measure_id: "SEPSIS-PROC-002"
      description: "Blood cultures before antibiotics"
      target: ">95% compliance"

metadata:
  protocol_type: "CLINICAL_PATHWAY"
  implementation_complexity: "HIGH"
  requires_icu_level_care: true
  estimated_implementation_time_minutes: 180
  review_cycle_months: 12
```

---

## Protocol Migration Plan

### Migration Strategy: Phased Enhancement

**Phase 1: Add Trigger Criteria** (All 16 protocols - 4-6 hours)
- Start with simple triggers (single parameter)
- Gradually add complex triggers (nested AND/OR logic)
- Validate trigger evaluation with test cases

**Phase 2: Add Confidence Scoring** (All 16 protocols - 3-4 hours)
- Define base confidence from evidence strength
- Add 2-4 modifiers per protocol
- Set activation thresholds

**Phase 3: Add Medication Selection** (8 medication-heavy protocols - 5-7 hours)
- Sepsis, STEMI, Stroke, ACS, DKA, COPD, Heart Failure, Pneumonia
- Add selection criteria with alternatives
- Include allergy checking logic

**Phase 4: Add Time Constraints** (6 time-sensitive protocols - 3-4 hours)
- Sepsis (Hour-1/3 bundles)
- STEMI (door-to-balloon <90 min)
- Stroke (tPA <4.5 hours)
- Anaphylaxis (epinephrine immediate)
- DKA (insulin within 1 hour)
- ACS (antiplatelet within 24 hours)

**Phase 5: Add Special Populations** (All 16 protocols - 4-5 hours)
- Elderly (age ≥65)
- Pregnancy
- Immunocompromised
- Renal/hepatic impairment

**Phase 6: Add Escalation Rules** (All 16 protocols - 3-4 hours)
- Define ICU transfer criteria
- Specialist consultation triggers
- Clinical deterioration detection

**Total Migration Effort**: 22-30 hours

---

### Migration Workflow Per Protocol

**Step 1: Create Enhanced YAML Template**
```bash
cp sepsis-management.yaml sepsis-management-enhanced.yaml
```

**Step 2: Add Trigger Criteria Section**
```yaml
# Analyze existing activation_criteria comments
# Convert to structured trigger_criteria with match_logic
trigger_criteria:
  match_logic: "ALL_OF"
  conditions:
    - condition_id: "PROTOCOL-TRIG-001"
      parameter: "key_parameter"
      operator: ">="
      threshold: X
```

**Step 3: Add Confidence Scoring**
```yaml
# Define base confidence from evidence_level
# STRONG → 0.85-0.90, MODERATE → 0.75-0.80, WEAK → 0.60-0.70
confidence_scoring:
  base_confidence: 0.85
  modifiers:
    - modifier_id: "PROTOCOL-CONF-001"
      condition:
        parameter: "age"
        operator: ">="
        threshold: 65
      adjustment: 0.05
```

**Step 4: Enhance Action with Medication Selection**
```yaml
# For each medication action, add selection_criteria
actions:
  - action_id: "PROTOCOL-ACT-001"
    medication_selection:
      selection_criteria:
        - criteria_id: "NO_ALLERGY"
          primary_medication: { ... }
          alternative_medication: { ... }
```

**Step 5: Add Time Constraints**
```yaml
# Define bundle deadlines
time_constraints:
  - constraint_id: "PROTOCOL-TIME-001"
    bundle_name: "Hour-1 Bundle"
    offset_minutes: 60
    critical: true
    required_actions: ["ACT-001", "ACT-002"]
```

**Step 6: Add Special Populations**
```yaml
special_populations:
  - population_id: "ELDERLY"
    inclusion_criteria:
      parameter: "age"
      operator: ">="
      threshold: 65
    modifications:
      - action_id: "ACT-001"
        new_parameters: { dose: "adjusted" }
```

**Step 7: Add Escalation Rules**
```yaml
escalation_rules:
  - rule_id: "PROTOCOL-ESC-001"
    escalation_trigger:
      parameter: "severity_score"
      operator: ">="
      threshold: 8
    recommendation:
      escalation_level: "ICU_TRANSFER"
```

**Step 8: Validate Enhanced Protocol**
```bash
java -jar protocol-validator.jar sepsis-management-enhanced.yaml
# Should pass schema validation
# Should have no missing required fields
# Should have valid references
```

**Step 9: Replace Original**
```bash
mv sepsis-management-enhanced.yaml sepsis-management.yaml
```

---

### Protocol-Specific Migration Notes

#### 1. Sepsis Management (SEPSIS-BUNDLE-001)
**Priority**: **Critical** (most complex, time-sensitive)
**Effort**: 2-3 hours

**Key Enhancements**:
- Trigger: Lactate ≥2.0 + (SBP <90 OR MAP <65) + infection suspected
- Confidence modifiers: WBC ≥12K (+0.10), Age ≥65 (+0.05), Procalcitonin ≥2.0 (+0.15)
- Medication selection: Ceftriaxone vs Levofloxacin (penicillin allergy), Meropenem (MDR risk)
- Time constraints: Hour-1 bundle (critical), Hour-3 bundle (important)
- Special populations: Elderly (reduced doses), Immunocompromised (broader coverage)
- Escalation: Lactate ≥4.0 → ICU, MAP <65 after fluids → vasopressors

#### 2. STEMI Management (STEMI-PROTOCOL-001)
**Priority**: **Critical** (time-sensitive PCI)
**Effort**: 2-3 hours

**Key Enhancements**:
- Trigger: STEMI criteria on ECG + chest pain + troponin elevation
- Confidence modifiers: Classic symptoms (+0.10), Multiple leads (+0.08)
- Time constraint: Door-to-balloon <90 minutes (critical)
- Special populations: Elderly (bleeding risk), Pregnancy (radiation concerns)
- Escalation: Cardiogenic shock → ICU, PCI failure → CABG consult

#### 3. Stroke Management (STROKE-tPA-001)
**Priority**: **Critical** (time-sensitive tPA)
**Effort**: 2-3 hours

**Key Enhancements**:
- Trigger: NIHSS ≥4 + symptom onset <4.5h + no ICH on CT
- Confidence modifiers: NIHSS ≥10 (+0.10), Witnessed onset (+0.08)
- Time constraint: tPA <4.5 hours from onset (critical), <3 hours (ideal)
- Medication selection: Check INR, platelets, recent surgery contraindications
- Escalation: Hemorrhage → neurosurgery, Large vessel occlusion → thrombectomy

#### 4-16. Remaining Protocols
**Follow similar pattern**:
- Define clinical triggers from guideline criteria
- Add confidence modifiers from risk stratification tools
- Include medication alternatives for allergies/contraindications
- Define time-sensitive interventions
- Specify ICU transfer criteria

---

## Implementation Roadmap

### Phase 1: Critical Runtime Logic (5-8 hours)
**Goal**: Enable automatic protocol activation and safe medication selection

**Week 1 Tasks**:
1. **ConditionEvaluator.java** (3-4 hours)
   - [ ] Implement evaluate() method with ALL_OF/ANY_OF logic
   - [ ] Add evaluateCondition() with recursion support
   - [ ] Implement compareValues() with operator handling
   - [ ] Add extractParameterValue() for patient context
   - [ ] Unit tests: 10 test cases (simple AND, OR, nested, operators)

2. **MedicationSelector.java** (4-5 hours)
   - [ ] Implement selectMedication() with criteria evaluation
   - [ ] Add evaluateCriteria() for standard criteria (NO_PENICILLIN_ALLERGY, CREATININE_CLEARANCE_GT_40, MDR_RISK)
   - [ ] Implement calculateCrCl() (Cockcroft-Gault formula)
   - [ ] Add applyDoseAdjustments() for renal/hepatic
   - [ ] Implement hasAllergy() checking
   - [ ] Unit tests: 15 test cases (allergy, renal adjustment, MDR, alternatives)

3. **TimeConstraintTracker.java** (3-4 hours)
   - [ ] Implement evaluateConstraints() with deadline calculation
   - [ ] Add alert level logic (WARNING <30min, CRITICAL overdue)
   - [ ] Create TimeConstraintStatus class
   - [ ] Implement bundle compliance tracking
   - [ ] Unit tests: 8 test cases (on-time, warning, critical, bundle compliance)

**Phase 1 Acceptance Criteria**:
- ✅ Protocols activate automatically when triggers met
- ✅ Medication selection respects allergies and selects alternatives
- ✅ Time constraint alerts generated correctly
- ✅ All unit tests passing (33+ tests)
- ✅ Integration test: ROHAN-001 test case generates recommendations

---

### Phase 2: Quality & Performance (4-6 hours)
**Goal**: Add confidence ranking, validation, and optimized lookup

**Week 2 Tasks**:
1. **ConfidenceCalculator.java** (2-3 hours)
   - [ ] Implement calculateConfidence() with base + modifiers
   - [ ] Add meetsActivationThreshold() filtering
   - [ ] Implement confidence clamping [0.0, 1.0]
   - [ ] Unit tests: 8 test cases (base, modifiers, threshold, clamping)

2. **ProtocolValidator.java** (2 hours)
   - [ ] Implement validate() with required fields check
   - [ ] Add validateActionReferences() for duplicate IDs
   - [ ] Implement validateConfidenceScoring() range checks
   - [ ] Add validateEvidenceSource() completeness
   - [ ] Create ValidationResult class with errors/warnings
   - [ ] Unit tests: 6 test cases (valid, missing fields, invalid ranges)

3. **KnowledgeBaseManager.java** (4-5 hours)
   - [ ] Implement singleton pattern with double-checked locking
   - [ ] Add loadAllProtocols() with validation integration
   - [ ] Implement buildIndexes() for category/specialty
   - [ ] Add query methods (getByCategory, getBySpecialty, search)
   - [ ] Implement hot reload with FileWatcher
   - [ ] Unit tests: 10 test cases (singleton, indexes, query, reload)

4. **Integration: Update ProtocolMatcher.java** (1-2 hours)
   - [ ] Integrate ConditionEvaluator for trigger evaluation
   - [ ] Integrate ConfidenceCalculator for scoring
   - [ ] Add sorting by confidence (descending)
   - [ ] Unit tests: 5 test cases (matching, ranking, filtering)

5. **Integration: Update ActionBuilder.java** (1-2 hours)
   - [ ] Integrate MedicationSelector for action selection
   - [ ] Integrate TimeConstraintTracker for constraint application
   - [ ] Add alternative action selection for contraindications
   - [ ] Unit tests: 6 test cases (selection, alternatives, time)

**Phase 2 Acceptance Criteria**:
- ✅ Multiple matching protocols ranked by confidence
- ✅ Protocols validated at load time with error reporting
- ✅ Fast protocol lookup using indexes (<5ms)
- ✅ All unit tests passing (35+ additional tests)
- ✅ Integration test: Multiple protocols ranked correctly

---

### Phase 3: Advanced Features (3-4 hours)
**Goal**: Add special populations and auto-escalation

**Week 3 Tasks**:
1. **EscalationRuleEvaluator.java** (2-3 hours)
   - [ ] Implement evaluateEscalation() with rule evaluation
   - [ ] Add gatherClinicalEvidence() for rationale
   - [ ] Create EscalationRecommendation class
   - [ ] Implement escalation level sorting (IMMEDIATE first)
   - [ ] Unit tests: 6 test cases (triggers, levels, evidence)

2. **Enhanced YAML Migration** (3-4 hours)
   - [ ] Add special_populations section to 16 protocols
     - Elderly (age ≥65) with dose adjustments
     - Pregnancy with medication contraindications
     - Immunocompromised with broader coverage
   - [ ] Add escalation_rules section to 16 protocols
     - ICU transfer criteria
     - Specialist consultation triggers
   - [ ] Validate all enhanced protocols pass ProtocolValidator

3. **Integration: Update ClinicalRecommendationEngine.java** (1 hour)
   - [ ] Integrate EscalationRuleEvaluator
   - [ ] Add escalation recommendations to output
   - [ ] Unit tests: 4 test cases (escalation flow)

**Phase 3 Acceptance Criteria**:
- ✅ Special population modifications applied correctly
- ✅ Escalation recommendations generated for ICU criteria
- ✅ All 16 protocols enhanced with new sections
- ✅ All unit tests passing (10+ additional tests)
- ✅ End-to-end test: Septic shock patient → ICU escalation

---

### Complete Phased Timeline

| Phase | Duration | Components | Cumulative Total |
|-------|----------|------------|------------------|
| Phase 1 | 5-8 hours | ConditionEvaluator, MedicationSelector, TimeConstraintTracker | 5-8 hours |
| Phase 2 | 4-6 hours | ConfidenceCalculator, ProtocolValidator, KnowledgeBaseManager | 9-14 hours |
| Phase 3 | 3-4 hours | EscalationRuleEvaluator, YAML migration, integration | 12-18 hours |
| **Total** | **12-18 hours** | **7 new Java classes + 16 enhanced YAMLs** | **12-18 hours** |

**Note**: This excludes protocol YAML migration time (22-30 hours). If including full migration:
**Total Effort**: 34-48 hours (4-6 working days)

---

## Acceptance Criteria

### Functional Acceptance Criteria

#### AC1: Automatic Protocol Activation
**Given**: Patient with lactate 3.2 mmol/L, SBP 85 mmHg, infection suspected
**When**: EnrichedPatientContext processed by ClinicalRecommendationEngine
**Then**:
- ✅ Sepsis protocol activates (trigger criteria met)
- ✅ Confidence score ≥0.70 (base 0.85 + WBC modifier if applicable)
- ✅ Recommendation includes Hour-1 bundle actions

#### AC2: Medication Selection with Allergy
**Given**: Patient with penicillin allergy documented
**When**: Sepsis protocol recommendation generated
**Then**:
- ✅ Ceftriaxone NOT selected (penicillin cross-reactivity)
- ✅ Levofloxacin selected as alternative
- ✅ Recommendation includes allergy rationale

#### AC3: Renal Dose Adjustment
**Given**: Patient with CrCl 25 mL/min (calculated from age/weight/creatinine)
**When**: Sepsis protocol recommendation generated
**Then**:
- ✅ Ceftriaxone dose reduced to 1 g (from 2 g)
- ✅ Recommendation includes renal adjustment rationale
- ✅ Monitoring includes "Monitor renal function daily"

#### AC4: Time Constraint Tracking
**Given**: Sepsis recognized at T=0, current time T+45 minutes
**When**: Time constraints evaluated
**Then**:
- ✅ Hour-1 bundle shows 15 minutes remaining
- ✅ Alert level = WARNING (within 30 minutes)
- ✅ Message: "Hour-1 Sepsis Bundle deadline in 15 minutes"

**Given**: Current time T+65 minutes (deadline exceeded)
**Then**:
- ✅ Alert level = CRITICAL
- ✅ Message: "Hour-1 Sepsis Bundle deadline exceeded by 5 minutes"

#### AC5: Confidence Scoring and Ranking
**Given**: Patient matches both Sepsis (confidence 0.92) and Pneumonia (confidence 0.78)
**When**: Recommendations generated
**Then**:
- ✅ Both protocols activated (both ≥0.70 threshold)
- ✅ Recommendations sorted by confidence (Sepsis first)
- ✅ Each recommendation includes confidence score

#### AC6: Escalation Rule Triggering
**Given**: Patient with lactate 4.5 mmol/L (septic shock)
**When**: Escalation rules evaluated
**Then**:
- ✅ Escalation recommendation generated
- ✅ Escalation level = ICU_TRANSFER
- ✅ Rationale includes lactate value
- ✅ Required interventions listed (vasopressors, arterial line)

#### AC7: Protocol Validation
**Given**: Protocol YAML with missing action_id
**When**: Protocol loaded by ProtocolLoader
**Then**:
- ✅ Validation fails
- ✅ Error message: "Action missing action_id"
- ✅ Protocol NOT added to knowledge base

---

### Performance Acceptance Criteria

#### PC1: Protocol Lookup Performance
**Given**: KnowledgeBaseManager with 16 protocols loaded
**When**: getByCategory(INFECTIOUS) called
**Then**:
- ✅ Response time <5ms (indexed lookup)
- ✅ Correct protocols returned (Sepsis, Pneumonia, Neutropenic Fever)

#### PC2: Recommendation Generation Performance
**Given**: EnrichedPatientContext with complete patient data
**When**: generateRecommendations() called
**Then**:
- ✅ Total processing time <100ms
  - Trigger evaluation: <20ms (16 protocols)
  - Confidence calculation: <10ms
  - Medication selection: <30ms
  - Time tracking: <10ms
  - Escalation evaluation: <10ms
  - Recommendation building: <20ms

#### PC3: Hot Reload Performance
**Given**: Protocol YAML file modified
**When**: FileWatcher detects change
**Then**:
- ✅ Reload triggered within 5 seconds
- ✅ Validation completes <1 second
- ✅ Indexes rebuilt <500ms
- ✅ No service interruption (thread-safe)

---

### Quality Acceptance Criteria

#### QC1: Test Coverage
- ✅ Unit test coverage ≥85% for all new classes
- ✅ Integration test coverage ≥75% for workflow
- ✅ End-to-end test for ROHAN-001 test case passes

#### QC2: Code Quality
- ✅ No SonarQube critical issues
- ✅ Cyclomatic complexity <15 per method
- ✅ Code duplication <3%
- ✅ All public methods have Javadoc

#### QC3: Clinical Safety
- ✅ All medication selections respect documented allergies
- ✅ All contraindications trigger alternative actions
- ✅ All critical time constraints generate alerts
- ✅ All escalation criteria trigger correctly

---

## Risk Assessment

### Technical Risks

#### Risk 1: YAML Parsing Complexity
**Risk**: Enhanced YAML structure may cause Jackson deserialization errors
**Probability**: Medium
**Impact**: High
**Mitigation**:
- Create comprehensive unit tests for YAML parsing before migration
- Use ProtocolValidator to catch schema issues early
- Migrate protocols incrementally (test each before moving to next)

#### Risk 2: Performance Degradation
**Risk**: Evaluating 16 protocols with complex logic may exceed 100ms target
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Implement KnowledgeBaseManager indexes for fast filtering
- Use early termination in evaluators (short-circuit AND/OR)
- Profile performance and optimize bottlenecks

#### Risk 3: Nested Condition Recursion
**Risk**: Deep recursion in ConditionEvaluator may cause stack overflow
**Probability**: Low
**Impact**: High
**Mitigation**:
- Limit nesting depth to 4 levels (validate in ProtocolValidator)
- Add recursion depth counter with exception if exceeded
- Test with deeply nested protocols

---

### Clinical Risks

#### Risk 4: Medication Selection Logic Errors
**Risk**: Bug in allergy checking could recommend contraindicated medication
**Probability**: Low
**Impact**: **CRITICAL** (patient safety)
**Mitigation**:
- Extensive unit testing of all selection criteria combinations
- Manual review of medication selection logic by clinical expert
- Fail-safe: Default to NO medication if ANY selection error occurs
- Clinical validation with 100+ test cases before production

#### Risk 5: Time Constraint Miscalculation
**Risk**: Incorrect deadline calculation could miss critical interventions
**Probability**: Low
**Impact**: High
**Mitigation**:
- Unit test all time calculations with edge cases (leap seconds, DST, timezone)
- Use Instant (UTC) throughout, no local time conversions
- Add logging for all deadline calculations
- Clinical validation of sepsis bundle timing

#### Risk 6: False Positive Escalation
**Risk**: Escalation rules too sensitive, generating unnecessary ICU transfers
**Probability**: Medium
**Impact**: Medium (resource waste, alarm fatigue)
**Mitigation**:
- Set escalation thresholds based on clinical evidence (SOFA score, lactate)
- Track escalation recommendation acceptance rate
- Implement feedback mechanism for clinicians to report false positives
- Adjust escalation criteria based on real-world performance

---

### Migration Risks

#### Risk 7: Protocol Migration Errors
**Risk**: Manual YAML editing introduces syntax errors or logic bugs
**Probability**: Medium
**Impact**: Medium
**Mitigation**:
- Use ProtocolValidator to validate ALL migrated protocols
- Create migration checklist for each protocol
- Two-person review for critical protocols (Sepsis, STEMI, Stroke)
- Automated schema validation in CI/CD pipeline

#### Risk 8: Backward Compatibility
**Risk**: Enhanced structure breaks existing protocol loading
**Probability**: Low
**Impact**: High
**Mitigation**:
- Keep basic YAML structure backward compatible
- All new sections optional (gracefully handle absence)
- Comprehensive regression testing of existing functionality
- Feature flagging for gradual rollout

---

## Effort Estimation

### Development Effort Breakdown

| Component | Complexity | Estimated Effort | Confidence |
|-----------|------------|------------------|------------|
| **ConditionEvaluator.java** | High (recursion, operators) | 3-4 hours | 80% |
| **ConfidenceCalculator.java** | Medium (scoring math) | 2-3 hours | 85% |
| **MedicationSelector.java** | High (clinical logic, safety) | 4-5 hours | 75% |
| **TimeConstraintTracker.java** | Medium (time calculations) | 3-4 hours | 85% |
| **KnowledgeBaseManager.java** | Medium (singleton, indexes) | 4-5 hours | 80% |
| **EscalationRuleEvaluator.java** | Medium (rule evaluation) | 2-3 hours | 85% |
| **ProtocolValidator.java** | Low (schema validation) | 2 hours | 90% |
| **Integration (Matcher, Builder)** | Medium (component wiring) | 2-3 hours | 80% |
| **Unit Tests (all components)** | High (80+ test cases) | 4-5 hours | 75% |
| **Integration Tests** | Medium (10+ test scenarios) | 2-3 hours | 80% |
| **Documentation** | Low (Javadoc, README) | 1-2 hours | 90% |
| **TOTAL DEVELOPMENT** | - | **29-37 hours** | **81%** |

### Protocol Migration Effort

| Task | Protocols | Estimated Effort | Confidence |
|------|-----------|------------------|------------|
| **Add trigger_criteria** | 16 | 4-6 hours | 85% |
| **Add confidence_scoring** | 16 | 3-4 hours | 90% |
| **Add medication_selection** | 8 (medication-heavy) | 5-7 hours | 75% |
| **Add time_constraints** | 6 (time-sensitive) | 3-4 hours | 85% |
| **Add special_populations** | 16 | 4-5 hours | 80% |
| **Add escalation_rules** | 16 | 3-4 hours | 85% |
| **Validation & Testing** | 16 | 3-4 hours | 80% |
| **TOTAL MIGRATION** | - | **25-34 hours** | **83%** |

### Complete Project Effort

| Phase | Effort Range | Confidence |
|-------|--------------|------------|
| Development (Java classes) | 29-37 hours | 81% |
| Protocol Migration (YAML) | 25-34 hours | 83% |
| **Total Project** | **54-71 hours** | **82%** |
| **Pessimistic (+ 20% buffer)** | **65-85 hours** | **90%** |

**Recommended Timeline**: 8-11 working days (assuming 8-hour days)

---

### Effort Distribution Visualization

```
Development Effort (29-37 hours):
├─ Core Logic (12-16h): ConditionEvaluator, MedicationSelector, TimeTracker
├─ Quality Features (8-11h): ConfidenceCalculator, Validator, KnowledgeBase
├─ Advanced Features (4-6h): EscalationEvaluator, Integration
└─ Testing (5-7h): Unit + Integration tests

Migration Effort (25-34 hours):
├─ Critical Sections (12-17h): triggers, medication_selection, time_constraints
├─ Advanced Sections (7-9h): special_populations, escalation_rules
└─ Validation (6-8h): Testing, error fixing
```

---

## Conclusion

### Current State Summary
Module 3 has **95% structural completeness** (16 protocols, basic loader) but **60% functional alignment** to CDS specification. The system has comprehensive clinical knowledge but lacks the runtime intelligence to apply it effectively.

### Critical Gaps Requiring Immediate Attention
1. **No automatic protocol activation** (ConditionEvaluator missing)
2. **No safe medication selection** (MedicationSelector missing)
3. **No time-sensitive intervention tracking** (TimeConstraintTracker missing)

These gaps make the current system **unsuitable for production clinical use**.

### Alignment Path Forward
**Phase 1 (Critical - 5-8 hours)**: Implement runtime intelligence core to enable basic clinical decision support
**Phase 2 (High Priority - 4-6 hours)**: Add quality/performance features for production readiness
**Phase 3 (Medium Priority - 3-4 hours)**: Add advanced clinical features for comprehensive CDS

**Total Effort**: 12-18 hours development + 25-34 hours protocol migration = **37-52 hours** (5-7 working days)

### Success Criteria
- ✅ Protocols activate automatically based on patient state
- ✅ Medication selection respects allergies and selects safe alternatives
- ✅ Time-critical interventions tracked with deadline alerts
- ✅ Multiple protocols ranked by confidence
- ✅ Clinical safety validated with 100+ test cases
- ✅ Performance meets targets (<100ms per recommendation)

### Next Steps
1. **Review and approve this alignment plan**
2. **Prioritize phases** (Phase 1 critical, Phase 2/3 optional)
3. **Assign development resources** (backend Java developers)
4. **Begin Phase 1 implementation** (ConditionEvaluator, MedicationSelector, TimeConstraintTracker)
5. **Validate with ROHAN-001 test case**

---

**Document Status**: READY FOR REVIEW
**Recommended Action**: Proceed with Phase 1 Critical Implementation
