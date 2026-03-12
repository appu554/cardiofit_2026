# Module 3 Phase 7: Multi-Agent Orchestration Plan

**Date**: 2025-10-25
**Status**: 📋 PLANNING - Multi-Agent Execution Strategy
**Flink Version**: 2.1.0
**Estimated Duration**: 20-28 hours across 4 parallel agents

---

## 🎯 Multi-Agent Strategy Overview

Phase 7 will be implemented using **4 specialized agents** working in parallel on independent sub-systems, with coordination points for integration:

```
┌─────────────────────────────────────────────────────────────┐
│                    AGENT ORCHESTRATION                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent 1: Data Models        Agent 2: Protocol Library      │
│  (Backend Architect)         (System Architect)             │
│  ├─ ClinicalRecommendation   ├─ YAML Protocols (10)        │
│  ├─ StructuredAction         ├─ ProtocolLibraryLoader      │
│  ├─ MedicationDetails        ├─ EnhancedProtocolMatcher    │
│  ├─ DiagnosticDetails        └─ ActionBuilder              │
│  ├─ ContraindicationCheck                                   │
│  └─ AlternativeAction        ⏱️ 6-8 hours                   │
│                                                              │
│  ⏱️ 4-6 hours                                               │
│                                                              │
├──────────────────────────────┬───────────────────────────────┤
│                              │                               │
│  Agent 3: Safety & Clinical  │  Agent 4: Flink Integration  │
│  (Python Expert)             │  (Backend Architect)         │
│  ├─ SafetyValidator          │  ├─ Kafka Serializers        │
│  ├─ MedicationActionBuilder  │  ├─ ClinicalRecommendation-  │
│  ├─ AlternativeAction-       │  │   Processor              │
│  │   Generator                │  ├─ Module3_Clinical-        │
│  └─ Phase 6 Integration      │  │   RecommendationEngine   │
│                               │  └─ Pipeline Wiring          │
│  ⏱️ 4-6 hours                │                               │
│                              │  ⏱️ 4-6 hours                │
└──────────────────────────────┴───────────────────────────────┘
                              │
                              ↓
                    ┌──────────────────────┐
                    │  Integration Agent   │
                    │  (General Purpose)   │
                    │  ├─ End-to-end tests │
                    │  ├─ Clinical scenarios│
                    │  └─ Documentation    │
                    │                      │
                    │  ⏱️ 6-8 hours        │
                    └──────────────────────┘
```

---

## 🤖 Agent 1: Data Models & Core Structures

**Agent Type**: `backend-architect`
**Expertise**: Java data modeling, serialization, builder patterns
**Duration**: 4-6 hours
**Dependencies**: None (can start immediately)

### Deliverables

1. **ClinicalRecommendation.java** (~300 lines)
   - Comprehensive recommendation model with builder pattern
   - Fields: protocol info, actions, safety, alternatives, evidence
   - Jackson annotations for JSON serialization
   - Lombok annotations for boilerplate reduction

2. **StructuredAction.java** (~250 lines)
   - Detailed clinical action model
   - Enum: ActionType (DIAGNOSTIC, THERAPEUTIC, MONITORING, ESCALATION)
   - Nested: MedicationDetails, DiagnosticDetails
   - Timing and evidence fields

3. **MedicationDetails.java** (~200 lines)
   - Medication-specific details
   - Fields: name, dose, route, frequency, duration
   - Dosing calculation metadata
   - Safety warnings list

4. **DiagnosticDetails.java** (~150 lines)
   - Diagnostic order details
   - Fields: test type, specimen, urgency, timing

5. **ContraindicationCheck.java** (~100 lines)
   - Safety validation result model
   - Fields: medication, contraindication type, severity, rationale

6. **AlternativeAction.java** (~150 lines)
   - Alternative treatment option
   - Fields: medication, rationale, evidence

7. **ProtocolState.java** (~100 lines)
   - Flink state model for protocol tracking
   - Fields: last protocol ID, timestamp, recommendation count

### Code Pattern Example

```java
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalRecommendation implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("recommendationId")
    private String recommendationId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("triggeredByAlert")
    private String triggeredByAlert;

    @JsonProperty("timestamp")
    private long timestamp;

    // Protocol Information
    @JsonProperty("protocolId")
    private String protocolId;

    @JsonProperty("protocolName")
    private String protocolName;

    @JsonProperty("protocolCategory")
    private String protocolCategory;  // INFECTION, CARDIOVASCULAR, etc.

    @JsonProperty("evidenceBase")
    private String evidenceBase;

    @JsonProperty("guidelineSection")
    private String guidelineSection;

    // Actions
    @JsonProperty("actions")
    private List<StructuredAction> actions;

    // Priority & Timing
    @JsonProperty("priority")
    private String priority;  // CRITICAL, HIGH, MEDIUM, LOW

    @JsonProperty("timeframe")
    private String timeframe;  // IMMEDIATE, <1hr, <4hr, ROUTINE

    @JsonProperty("urgencyRationale")
    private String urgencyRationale;

    // Safety Validation
    @JsonProperty("contraindicationsChecked")
    private List<ContraindicationCheck> contraindicationsChecked;

    @JsonProperty("safeToImplement")
    private boolean safeToImplement;

    @JsonProperty("warnings")
    private List<String> warnings;

    // Alternative Plans
    @JsonProperty("alternatives")
    private List<AlternativeAction> alternatives;

    // Monitoring Requirements
    @JsonProperty("monitoringRequirements")
    private List<String> monitoringRequirements;

    @JsonProperty("escalationCriteria")
    private String escalationCriteria;

    // Confidence & Source
    @JsonProperty("confidenceScore")
    private double confidenceScore;

    @JsonProperty("reasoningPath")
    private String reasoningPath;
}
```

### Testing Requirements
- Unit tests for each model class
- JSON serialization/deserialization tests
- Builder pattern validation
- **Target Coverage**: >90% line coverage

---

## 🤖 Agent 2: Protocol Library System

**Agent Type**: `system-architect`
**Expertise**: YAML parsing, protocol design, pattern matching
**Duration**: 6-8 hours
**Dependencies**: None (can start immediately)

### Deliverables

1. **10 YAML Protocol Definitions** (~200 lines each)
   - SEPSIS-BUNDLE-001: Sepsis Management (SSC 2021)
   - STEMI-001: STEMI Management (ACC/AHA 2013)
   - HF-ACUTE-001: Acute Heart Failure (ACC/AHA 2022)
   - DKA-001: Diabetic Ketoacidosis Management
   - ARDS-001: ARDS Management
   - STROKE-001: Acute Ischemic Stroke
   - ANAPHYLAXIS-001: Anaphylaxis Management
   - HYPERKALEMIA-001: Severe Hyperkalemia
   - ACS-NSTEMI-001: NSTEMI Management
   - HYPERTENSIVE-CRISIS-001: Hypertensive Emergency

2. **Protocol.java** (~200 lines)
   - Protocol data model
   - Fields: ID, name, category, evidence, trigger criteria, actions

3. **ProtocolLibraryLoader.java** (~300 lines)
   - YAML file loader using Jackson
   - Protocol validation
   - Caching and indexing
   - Error handling for malformed protocols

4. **EnhancedProtocolMatcher.java** (~400 lines)
   - Replaces hardcoded 6-protocol matcher
   - Alert-based matching
   - Clinical criteria matching (qSOFA, SIRS, etc.)
   - Priority scoring for multiple matches

5. **ActionBuilder.java** (~300 lines)
   - Converts protocol actions to StructuredAction objects
   - Integrates with MedicationActionBuilder
   - Handles action sequencing and timing

### YAML Protocol Example

```yaml
protocolId: "SEPSIS-BUNDLE-001"
protocolName: "Sepsis Management Bundle"
protocolCategory: "INFECTION"
evidenceBase: "Surviving Sepsis Campaign 2021"
guidelineSection: "SSC 2021, Bundle Elements, pp. 1-15"
priority: "CRITICAL"

# Trigger Criteria
triggerCriteria:
  alertTypes:
    - "SEPSIS_CONFIRMED"
    - "SEPSIS_LIKELY"
  minimumPriority: "P0_CRITICAL"
  clinicalCriteria:
    qsofaScore: ">=2"
    sirsScore: ">=2"

# Exclusion Criteria
exclusionCriteria:
  - type: "RECENT_PROTOCOL"
    protocolId: "SEPSIS-BUNDLE-001"
    withinHours: 24
    rationale: "Bundle already initiated"

# Actions (in sequence)
actions:
  - sequenceOrder: 1
    actionType: "DIAGNOSTIC"
    actionId: "SEPSIS-001-D1"
    description: "Obtain blood cultures × 2 (aerobic + anaerobic) prior to antibiotics"
    urgency: "STAT"
    timeframe: "within 45 minutes"
    timeframeRationale: "Cultures obtained after antibiotics may be falsely negative"
    evidenceReference: "SSC 2021, Recommendation 3.1"
    evidenceStrength: "STRONG"
    prerequisiteChecks:
      - "Verify patient consent documented"
      - "Confirm adequate venous access"

  - sequenceOrder: 2
    actionType: "THERAPEUTIC"
    actionId: "SEPSIS-001-T1"
    medicationId: "MED-PIPT-001"  # Piperacillin-Tazobactam
    medicationName: "Piperacillin-Tazobactam"
    indication: "Empiric sepsis coverage (broad-spectrum)"
    urgency: "URGENT"
    timeframe: "within 1 hour"
    timeframeRationale: "Each hour delay increases mortality by 7.6%"
    evidenceReference: "SSC 2021, Recommendation 3.2.1"
    evidenceStrength: "STRONG"
    prerequisiteChecks:
      - "Verify no penicillin allergy"
      - "Check renal function for dosing"
    expectedOutcome: "Source control and bacterial clearance"
    monitoringParameters: "Clinical response, repeat cultures at 48-72h"

  - sequenceOrder: 3
    actionType: "DIAGNOSTIC"
    actionId: "SEPSIS-001-D2"
    description: "Measure serum lactate, repeat q2h until <2 mmol/L"
    urgency: "URGENT"
    timeframe: "within 1 hour"
    timeframeRationale: "Lactate clearance predicts resuscitation adequacy"
    evidenceReference: "SSC 2021, Recommendation 2.3"
    evidenceStrength: "MODERATE"
    expectedOutcome: "Lactate normalization within 6 hours"

  - sequenceOrder: 4
    actionType: "THERAPEUTIC"
    actionId: "SEPSIS-001-T2"
    description: "Crystalloid fluid resuscitation 30 mL/kg ideal body weight"
    conditions:
      - "Systolic BP < 90 mmHg OR"
      - "Lactate >= 4 mmol/L"
    urgency: "URGENT"
    timeframe: "within 3 hours"
    timeframeRationale: "Early goal-directed therapy improves outcomes"
    evidenceReference: "SSC 2021, Recommendation 4.1"
    evidenceStrength: "STRONG"
    prerequisiteChecks:
      - "Assess for fluid overload risk (heart failure, pulmonary edema)"
    monitoringParameters: "BP, urine output, lactate clearance"

  - sequenceOrder: 5
    actionType: "ESCALATION"
    actionId: "SEPSIS-001-E1"
    description: "ICU consultation if no improvement in 6 hours"
    conditions:
      - "Persistent hypotension OR"
      - "Lactate not clearing OR"
      - "Worsening organ dysfunction"
    urgency: "ROUTINE"
    timeframe: "within 6 hours"
    evidenceReference: "SSC 2021, Section 8.2"

# Alternative Actions (if primary contraindicated)
alternativeActions:
  - primaryActionId: "SEPSIS-001-T1"
    contraindicationType: "PENICILLIN_ALLERGY"
    alternativeMedicationId: "MED-MERO-001"  # Meropenem
    alternativeMedicationName: "Meropenem"
    rationale: "Carbapenem with broad coverage, lower cross-reactivity"

  - primaryActionId: "SEPSIS-001-T1"
    contraindicationType: "ALL_BETA_LACTAM_ALLERGY"
    alternativeMedicationId: "MED-CIPRO-001"  # Ciprofloxacin
    alternativeMedicationName: "Ciprofloxacin + Metronidazole"
    rationale: "Non-beta-lactam empiric coverage"

# Monitoring & Escalation
monitoringRequirements:
  - parameter: "Blood pressure"
    frequency: "Continuous (if hypotensive) or hourly"
  - parameter: "Lactate"
    frequency: "q2h until <2 mmol/L, then q6h"
  - parameter: "Urine output"
    frequency: "Hourly"
  - parameter: "Blood cultures"
    frequency: "Repeat at 48-72h if no improvement"

escalationCriteria:
  - condition: "Persistent hypotension despite 30 mL/kg fluids"
    action: "Initiate vasopressor therapy (norepinephrine)"
  - condition: "No lactate clearance in 6 hours"
    action: "Reassess source control and antibiotic coverage"
```

### Testing Requirements
- YAML parsing and validation tests
- Protocol matching logic tests
- Edge cases (multiple matches, no matches)
- **Target Coverage**: >85% line coverage

---

## 🤖 Agent 3: Safety Validation & Clinical Logic

**Agent Type**: `python-expert` (but writing Java - name is historical)
**Expertise**: Clinical safety logic, medication integration, Phase 6 reuse
**Duration**: 4-6 hours
**Dependencies**: Agent 1 (data models) must be complete

### Deliverables

1. **SafetyValidator.java** (~400 lines)
   - Orchestrates all Phase 6 safety checkers
   - Integrates: ContraindicationChecker, DrugInteractionChecker, AllergyChecker
   - Returns SafetyValidationResult with warnings

2. **SafetyValidationResult.java** (~150 lines)
   - Result model for safety validation
   - Fields: isSafe, warnings list, contraindications detected

3. **MedicationActionBuilder.java** (~350 lines)
   - Generates medication-specific StructuredAction
   - Integrates Phase 6 DoseCalculator
   - Handles renal/hepatic dose adjustments
   - Calculates weight-based dosing

4. **AlternativeActionGenerator.java** (~300 lines)
   - Uses Phase 6 TherapeuticSubstitutionEngine
   - Generates alternatives for contraindicated medications
   - Prioritizes alternatives by efficacy and safety

5. **RecommendationEnricher.java** (~250 lines)
   - Adds evidence attribution
   - Calculates urgency timeframes
   - Generates monitoring requirements

### Code Pattern Example

```java
public class SafetyValidator {

    private final ContraindicationChecker contraindicationChecker;
    private final DrugInteractionChecker interactionChecker;
    private final AllergyChecker allergyChecker;
    private final MedicationDatabaseLoader medicationDB;

    public SafetyValidator() {
        // Initialize Phase 6 components
        this.medicationDB = MedicationDatabaseLoader.getInstance();
        this.contraindicationChecker = new ContraindicationChecker(medicationDB);
        this.interactionChecker = new DrugInteractionChecker(medicationDB);
        this.allergyChecker = new AllergyChecker(medicationDB);
    }

    /**
     * Validates a structured action for safety concerns.
     * Checks allergies, drug interactions, and contraindications.
     */
    public SafetyValidationResult validate(
            StructuredAction action,
            PatientContext patient) {

        List<String> warnings = new ArrayList<>();
        boolean safe = true;

        // Only validate medication actions
        if (action.getActionType() != ActionType.THERAPEUTIC ||
            action.getMedication() == null) {
            return SafetyValidationResult.safe();
        }

        String medicationId = action.getMedication().getMedicationId();
        Medication med = medicationDB.getMedication(medicationId);

        if (med == null) {
            warnings.add("❌ Medication not found in database: " + medicationId);
            return SafetyValidationResult.unsafe(warnings);
        }

        // 1. Allergy Check
        AllergyResult allergyResult = allergyChecker.checkAllergy(
            med,
            patient.getAllergies()
        );

        if (!allergyResult.isSafe()) {
            safe = false;
            warnings.add(String.format(
                "⚠️ ALLERGY ALERT: %s - %s",
                allergyResult.getAllergyType(),
                allergyResult.getWarning()
            ));
        }

        // 2. Drug Interaction Check
        List<String> patientMedications = new ArrayList<>(
            patient.getCurrentMedicationIds()
        );
        patientMedications.add(medicationId);

        List<InteractionResult> interactions = interactionChecker
            .checkPatientMedications(patientMedications);

        for (InteractionResult interaction : interactions) {
            if (interaction.getSeverity().equals("MAJOR")) {
                safe = false;
                warnings.add(String.format(
                    "⚠️ MAJOR INTERACTION: %s + %s - %s",
                    interaction.getDrug1Name(),
                    interaction.getDrug2Name(),
                    interaction.getManagement()
                ));
            } else if (interaction.getSeverity().equals("MODERATE")) {
                warnings.add(String.format(
                    "⚠️ MODERATE INTERACTION: %s + %s - %s",
                    interaction.getDrug1Name(),
                    interaction.getDrug2Name(),
                    interaction.getManagement()
                ));
            }
        }

        // 3. Contraindication Check
        ContraindicationResult contraResult = contraindicationChecker
            .checkContraindications(med, patient);

        if (contraResult.hasAbsoluteContraindication()) {
            safe = false;
            warnings.add(String.format(
                "❌ ABSOLUTE CONTRAINDICATION: %s",
                contraResult.getRationale()
            ));
        }

        if (contraResult.hasRelativeContraindication()) {
            warnings.add(String.format(
                "⚠️ RELATIVE CONTRAINDICATION: %s - %s",
                contraResult.getRationale(),
                contraResult.getManagement()
            ));
        }

        return new SafetyValidationResult(safe, warnings);
    }
}
```

### Testing Requirements
- Safety validation unit tests
- Phase 6 integration tests
- Clinical scenario tests (allergy, interaction, contraindication)
- **Target Coverage**: >85% line coverage

---

## 🤖 Agent 4: Flink Pipeline Integration

**Agent Type**: `backend-architect`
**Expertise**: Flink 2.1.0 DataStream API, Kafka integration, state management
**Duration**: 4-6 hours
**Dependencies**: Agents 1, 2, 3 must be complete

### Deliverables

1. **EnrichedPatientContextDeserializer.java** (~150 lines)
   - Kafka deserializer for Module 2 output
   - Jackson-based JSON deserialization
   - Error handling for malformed messages

2. **ClinicalRecommendationSerializer.java** (~150 lines)
   - Kafka serializer for Module 3 output
   - Jackson-based JSON serialization
   - Compact formatting for Kafka efficiency

3. **ClinicalRecommendationProcessor.java** (~500 lines)
   - Flink 2.1.0 KeyedProcessFunction
   - Orchestrates: ProtocolMatcher, ActionBuilder, SafetyValidator, AlternativeGenerator
   - Maintains protocol state in RocksDB
   - Emits ClinicalRecommendation events

4. **Module3_ClinicalRecommendationEngine.java** (~300 lines)
   - Main Flink job entry point
   - Kafka source/sink configuration
   - Pipeline topology definition
   - Checkpointing and state backend setup

### Code Pattern Example

```java
public class ClinicalRecommendationProcessor
    extends KeyedProcessFunction<String, EnrichedPatientContext, ClinicalRecommendation> {

    // State
    private ValueState<ProtocolState> protocolStateHolder;

    // Components
    private ProtocolMatcher protocolMatcher;
    private ActionBuilder actionBuilder;
    private SafetyValidator safetyValidator;
    private AlternativeActionGenerator alternativeGenerator;

    @Override
    public void open(Configuration parameters) throws Exception {
        // Initialize Phase 6 medication database
        MedicationDatabaseLoader.getInstance().loadDatabase();

        // Initialize protocol library
        ProtocolLibraryLoader protocolLoader = new ProtocolLibraryLoader();
        List<Protocol> protocols = protocolLoader.loadProtocols();
        protocolMatcher = new EnhancedProtocolMatcher(protocols);

        // Initialize components
        actionBuilder = new ActionBuilder();
        safetyValidator = new SafetyValidator();
        alternativeGenerator = new AlternativeActionGenerator();

        // Initialize state
        ValueStateDescriptor<ProtocolState> descriptor =
            new ValueStateDescriptor<>("protocol-state", ProtocolState.class);
        protocolStateHolder = getRuntimeContext().getState(descriptor);
    }

    @Override
    public void processElement(
            EnrichedPatientContext context,
            Context ctx,
            Collector<ClinicalRecommendation> out) throws Exception {

        // 1. Match protocol based on alerts and patient state
        Protocol protocol = protocolMatcher.matchProtocol(context);

        if (protocol == null) {
            // No protocol matched
            return;
        }

        // Check if protocol already applied recently (avoid duplicates)
        ProtocolState state = protocolStateHolder.value();
        if (state != null &&
            state.getLastProtocolId().equals(protocol.getProtocolId()) &&
            (System.currentTimeMillis() - state.getLastRecommendationTime()) < 24 * 3600 * 1000) {
            // Protocol applied within 24 hours, skip
            return;
        }

        // 2. Generate actions from protocol definition
        List<StructuredAction> actions = actionBuilder.buildActions(
            protocol,
            context.getPatientState()
        );

        // 3. Validate safety for each action
        List<StructuredAction> validatedActions = new ArrayList<>();
        List<AlternativeAction> alternatives = new ArrayList<>();
        boolean allSafe = true;

        for (StructuredAction action : actions) {
            SafetyValidationResult safety = safetyValidator.validate(
                action,
                context.getPatientState()
            );

            if (safety.isSafe()) {
                validatedActions.add(action);
            } else {
                allSafe = false;

                // 4. Generate alternatives if medication contraindicated
                if (action.getActionType() == ActionType.THERAPEUTIC) {
                    List<AlternativeAction> alts = alternativeGenerator
                        .generateAlternatives(action, context.getPatientState());
                    alternatives.addAll(alts);
                }

                // Add warnings to action
                action.setWarnings(safety.getWarnings());
                validatedActions.add(action);
            }
        }

        // 5. Create clinical recommendation
        ClinicalRecommendation recommendation = ClinicalRecommendation.builder()
            .recommendationId(UUID.randomUUID().toString())
            .patientId(context.getPatientId())
            .triggeredByAlert(getTriggeredAlert(context))
            .timestamp(System.currentTimeMillis())
            .protocolId(protocol.getProtocolId())
            .protocolName(protocol.getProtocolName())
            .protocolCategory(protocol.getProtocolCategory())
            .evidenceBase(protocol.getEvidenceBase())
            .guidelineSection(protocol.getGuidelineSection())
            .actions(validatedActions)
            .alternatives(alternatives)
            .safeToImplement(allSafe)
            .priority(protocol.getPriority())
            .timeframe(protocol.getTimeframe())
            .confidenceScore(0.95)
            .build();

        // 6. Emit recommendation
        out.collect(recommendation);

        // 7. Update state
        ProtocolState newState = new ProtocolState();
        newState.setLastProtocolId(protocol.getProtocolId());
        newState.setLastRecommendationTime(System.currentTimeMillis());
        newState.setRecommendationCount((state != null ? state.getRecommendationCount() : 0) + 1);
        protocolStateHolder.update(newState);
    }

    private String getTriggeredAlert(EnrichedPatientContext context) {
        if (context.getPatientState().getActiveAlerts().isEmpty()) {
            return "NONE";
        }
        return context.getPatientState().getActiveAlerts().get(0).getAlertId();
    }
}
```

### Testing Requirements
- Kafka serialization/deserialization tests
- Flink processor integration tests
- End-to-end pipeline tests
- **Target Coverage**: >80% line coverage

---

## 🤖 Integration Agent: End-to-End Testing & Documentation

**Agent Type**: `general-purpose`
**Expertise**: Integration testing, clinical validation, documentation
**Duration**: 6-8 hours
**Dependencies**: All 4 agents must be complete

### Deliverables

1. **End-to-End Integration Tests**
   - Sepsis case: Blood cultures + Pip-Tazo + Lactate monitoring
   - STEMI case: Aspirin + Heparin + Cath lab activation
   - Allergy case: Penicillin allergy → Carbapenem alternative
   - Renal impairment: Dose adjustment for CrCl < 30

2. **Clinical Scenario Validation**
   - 8-patient test cohort (from technical review)
   - Protocol matching accuracy validation
   - Safety checking validation
   - Alternative generation validation

3. **Performance Testing**
   - Processing latency < 100ms per patient
   - Kafka throughput testing
   - State backend performance

4. **Documentation**
   - Phase 7 completion report
   - API documentation
   - Deployment guide
   - Clinical validation report

---

## 🔄 Coordination Points

### Synchronization Requirements

```yaml
Coordination_Point_1:
  name: "Data Models Complete"
  trigger: Agent 1 finishes
  blocked_agents: [Agent 3, Agent 4]
  action: "Unblock Agent 3 and Agent 4 to start implementation"

Coordination_Point_2:
  name: "Protocol Library Complete"
  trigger: Agent 2 finishes
  blocked_agents: [Agent 4]
  action: "Unblock Agent 4 for protocol matcher integration"

Coordination_Point_3:
  name: "All Components Complete"
  trigger: Agents 1-4 all finish
  blocked_agents: [Integration Agent]
  action: "Start Integration Agent for end-to-end testing"

Coordination_Point_4:
  name: "Phase 7 Complete"
  trigger: Integration Agent finishes
  action: "Deploy to staging environment"
```

### Communication Protocol

Each agent will create status files:
- `AGENT_1_STATUS.md` - Data models progress
- `AGENT_2_STATUS.md` - Protocol library progress
- `AGENT_3_STATUS.md` - Safety validation progress
- `AGENT_4_STATUS.md` - Flink integration progress
- `INTEGRATION_STATUS.md` - End-to-end testing progress

---

## 📊 Success Criteria (Phase 7 Complete)

### Technical Validation
- ✅ All 4 agents complete their deliverables
- ✅ Maven build succeeds with no errors
- ✅ Test coverage >80% across all new components
- ✅ Flink job deploys successfully

### Clinical Validation
- ✅ 10 protocols loaded and matched correctly
- ✅ Sepsis case generates correct recommendations
- ✅ Allergy detection and alternative generation working
- ✅ Dose adjustments applied correctly

### Integration Validation
- ✅ Module 2 → Module 3 pipeline connected
- ✅ Kafka topics producing/consuming correctly
- ✅ End-to-end latency < 100ms
- ✅ Exactly-once semantics maintained

### Documentation Validation
- ✅ Phase 7 completion report written
- ✅ API documentation complete
- ✅ Deployment guide created
- ✅ Clinical validation report published

---

## 🚀 Execution Command

To start multi-agent execution:

```bash
# Start Agent 1 (Data Models)
/sc:implement "Module 3 Phase 7 - Agent 1: Data Models" --persona backend-architect --parallel

# Start Agent 2 (Protocol Library) - parallel
/sc:implement "Module 3 Phase 7 - Agent 2: Protocol Library" --persona system-architect --parallel

# Agent 3 and 4 will be triggered automatically when Agent 1 completes
# Integration Agent will be triggered when all 4 agents complete
```

---

## 📁 Deliverable File Structure

```
backend/shared-infrastructure/flink-processing/
├── src/main/java/com/cardiofit/flink/
│   ├── models/                          [Agent 1]
│   │   ├── ClinicalRecommendation.java
│   │   ├── StructuredAction.java
│   │   ├── MedicationDetails.java
│   │   ├── DiagnosticDetails.java
│   │   ├── ContraindicationCheck.java
│   │   ├── AlternativeAction.java
│   │   └── ProtocolState.java
│   │
│   ├── protocols/                        [Agent 2]
│   │   ├── Protocol.java
│   │   ├── ProtocolLibraryLoader.java
│   │   ├── EnhancedProtocolMatcher.java
│   │   └── ActionBuilder.java
│   │
│   ├── clinical/                         [Agent 3]
│   │   ├── SafetyValidator.java
│   │   ├── SafetyValidationResult.java
│   │   ├── MedicationActionBuilder.java
│   │   ├── AlternativeActionGenerator.java
│   │   └── RecommendationEnricher.java
│   │
│   ├── operators/                        [Agent 4]
│   │   ├── ClinicalRecommendationProcessor.java
│   │   └── Module3_ClinicalRecommendationEngine.java
│   │
│   └── serialization/                    [Agent 4]
│       ├── EnrichedPatientContextDeserializer.java
│       └── ClinicalRecommendationSerializer.java
│
├── src/main/resources/protocols/         [Agent 2]
│   ├── SEPSIS-BUNDLE-001.yaml
│   ├── STEMI-001.yaml
│   ├── HF-ACUTE-001.yaml
│   └── [7 more protocols]
│
└── src/test/java/                        [Integration Agent]
    └── com/cardiofit/flink/
        ├── integration/
        │   ├── Phase7IntegrationTest.java
        │   ├── ClinicalScenarioTest.java
        │   └── PerformanceTest.java
        └── [unit tests per agent]

claudedocs/
├── AGENT_1_STATUS.md                     [Agent 1 progress]
├── AGENT_2_STATUS.md                     [Agent 2 progress]
├── AGENT_3_STATUS.md                     [Agent 3 progress]
├── AGENT_4_STATUS.md                     [Agent 4 progress]
├── INTEGRATION_STATUS.md                 [Integration Agent progress]
└── MODULE3_PHASE7_COMPLETION_REPORT.md   [Final deliverable]
```

---

## ⏱️ Timeline

```
Day 1 (Hours 1-8):
  ├─ Agent 1: Data Models (4-6h) → COMPLETE
  ├─ Agent 2: Protocol Library (6-8h) → IN PROGRESS
  └─ Status: 2 agents running in parallel

Day 2 (Hours 9-16):
  ├─ Agent 2: Protocol Library → COMPLETE
  ├─ Agent 3: Safety Validation (4-6h) → START → COMPLETE
  ├─ Agent 4: Flink Integration (4-6h) → START → COMPLETE
  └─ Status: 2 agents running in parallel

Day 3 (Hours 17-24):
  ├─ Integration Agent: Testing & Docs (6-8h) → START → COMPLETE
  └─ Status: Phase 7 COMPLETE

Total: 20-28 hours over 3 days with parallel execution
```

---

## 🎯 Next Actions

1. ✅ **Confirm multi-agent approach** with user
2. 🚀 **Launch Agent 1 & Agent 2** in parallel
3. 📊 **Monitor progress** via status files
4. 🔄 **Coordinate handoffs** at synchronization points
5. ✅ **Validate completion** with Integration Agent

---

**Ready to execute?** Type `yes` to launch multi-agent Phase 7 implementation.
