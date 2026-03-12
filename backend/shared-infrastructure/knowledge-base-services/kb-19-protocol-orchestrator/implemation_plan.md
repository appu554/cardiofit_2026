# KB-19 Clinical Protocol Orchestrator - Implementation Plan

## CTO/CMO Decision: CONFIRMED

**Choice: FLEXIBILITY → KB-19 as Separate Orchestrator Service**

### Rationale (4 Leadership Lenses)
1. **Clinical (CMO)**: Arbitration ≠ Truth evaluation - different mental models, different liability
2. **Architecture (CTO)**: Bounded context - independent versioning, governance, evolution
3. **Execution**: Don't embed, don't extract later - healthcare platforms don't refactor safety logic
4. **Regulatory**: KB-19 = "legal witness" - named accountable layer for recommendations

### Key Insight
> "Separate boundary = legal firewall"
> When a hospital board asks "which component issued this recommendation?", the answer is ONE word: **KB-19**

---

## Executive Summary

KB-19 is **NOT another protocol calculator**. It is the **decision synthesis brain** that:
- Consumes clinical truth from Vaidshala CQL Engine
- Orchestrates KB-3 (temporal), KB-8 (calculators), KB-12 (ordersets), KB-14 (governance)
- Performs **arbitration** when multiple protocols conflict
- Produces evidence-backed recommendations with ACC/AHA Class grading
- Generates audit trails for regulatory compliance

---

## Architecture Position

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KNOWLEDGE LAYER (Tier 4 CQL)                        │
│                    "The Truth" - What is clinically true?                   │
│                           (Vaidshala CQL Engine)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                         EXECUTION LAYER (Tier 6 Go)                         │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    KB-19 Protocol Orchestrator                       │   │
│  │              "The Brain" - Decision Synthesis Engine                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│         │              │              │              │                      │
│         ▼              ▼              ▼              ▼                      │
│     ┌───────┐     ┌───────┐     ┌────────┐    ┌────────┐                   │
│     │ KB-3  │     │ KB-8  │     │ KB-12  │    │ KB-14  │                   │
│     │Temporal│    │ Calc  │     │OrderSet│    │Govern  │                   │
│     └───────┘     └───────┘     └────────┘    └────────┘                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Domain Model Implementation

### Location
```
backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/
```

### Directory Structure
```
kb-19-protocol-orchestrator/
├── cmd/server/
│   └── main.go                          # Entry point with signal handling
├── internal/
│   ├── api/
│   │   ├── server.go                    # Gin server setup
│   │   ├── handlers.go                  # Core handlers
│   │   ├── protocol_handlers.go         # Protocol execution handlers
│   │   └── arbitration_handlers.go      # Arbitration endpoints
│   ├── config/
│   │   └── config.go                    # Environment-based config
│   ├── models/
│   │   ├── patient_context.go           # PatientContext struct
│   │   ├── protocol_descriptor.go       # ProtocolDescriptor struct
│   │   ├── protocol_evaluation.go       # ProtocolEvaluation struct
│   │   ├── evidence_envelope.go         # EvidenceEnvelope struct
│   │   ├── arbitrated_decision.go       # ArbitratedDecision struct
│   │   ├── recommendation_bundle.go     # RecommendationBundle struct
│   │   └── conflict_matrix.go           # ConflictMatrix struct
│   ├── arbitration/
│   │   ├── engine.go                    # Core arbitration engine
│   │   ├── priority_hierarchy.go        # Protocol priority rules
│   │   ├── conflict_detector.go         # Conflict detection logic
│   │   ├── safety_gatekeeper.go         # Safety veto logic
│   │   └── recommendation_grader.go     # ACC/AHA Class assignment
│   ├── clients/
│   │   ├── vaidshala_client.go          # CQL Engine client
│   │   ├── kb3_temporal_client.go       # KB-3 temporal client
│   │   ├── kb8_calculator_client.go     # KB-8 calculator client
│   │   ├── kb12_orderset_client.go      # KB-12 execution binding
│   │   └── kb14_governance_client.go    # KB-14 task escalation
│   ├── narrative/
│   │   └── generator.go                 # Clinical narrative generation
│   └── database/
│       └── postgres.go                  # Decision audit storage
├── pkg/
│   └── contracts/
│       └── api_contracts.go             # Request/Response contracts
├── migrations/
│   └── 001_initial_schema.sql           # PostgreSQL schema
├── tests/
│   ├── arbitration_test.go
│   └── integration_test.go
├── go.mod
├── Dockerfile
└── KB19-README.md
```

---

## Phase 2: Core Domain Models

### 1. PatientContext (Input Brain)
```go
// File: internal/models/patient_context.go

type PatientContext struct {
    PatientID       uuid.UUID
    EncounterID     uuid.UUID
    Demographics    Demographics
    Vitals          VitalSigns
    Labs            []LabValue
    Diagnoses       []Diagnosis
    ICUStateSummary *ICUClinicalState  // From ICU Intelligence
    Comorbidities   []Comorbidity
    PregnancyStatus *PregnancyStatus
    MedicationList  []ActiveMedication
    CQLTruthFlags   map[string]bool    // From Vaidshala CQL Engine
    CalculatorScores map[string]float64 // From KB-8
    Timestamp       time.Time
}
```

### 2. ProtocolDescriptor (Metadata Only)
```go
// File: internal/models/protocol_descriptor.go

type ProtocolDescriptor struct {
    ID                   string
    Name                 string
    Category             ProtocolCategory  // Acute, Chronic, ICU, Emergency
    TriggerCriteria      []string          // CQL fact IDs that trigger this
    ContraindicationRules []string         // CQL fact IDs that block this
    PriorityClass        PriorityClass     // Emergency > Acute > Chronic
    GuidelineSource      string            // ACC/AHA, SSC, KDIGO, etc.
    GuidelineVersion     string
    RequiredCalculators  []string          // KB-8 calculators needed
}

type PriorityClass int
const (
    PriorityEmergency PriorityClass = 1  // Life-preserving
    PriorityAcute     PriorityClass = 2  // Organ stabilization
    PriorityChronic   PriorityClass = 3  // Long-term optimization
)
```

### 3. ProtocolEvaluation (Per-Protocol Truth Snapshot)
```go
// File: internal/models/protocol_evaluation.go

type ProtocolEvaluation struct {
    ProtocolID              string
    IsApplicable            bool
    ApplicabilityReason     string
    Contraindicated         bool
    ContraindicationReasons []string
    RecommendedActions      []AbstractAction
    RiskScoreImpact         float64
    CQLFactsUsed            []string  // Which CQL facts informed this
}
```

### 4. EvidenceEnvelope (Legal Protection)
```go
// File: internal/models/evidence_envelope.go

type EvidenceEnvelope struct {
    ID                  uuid.UUID
    RecommendationClass RecommendationClass  // I, IIa, IIb, III
    EvidenceLevel       EvidenceLevel        // HIGH, MODERATE, LOW
    GuidelineSource     string
    GuidelineVersion    string
    CitationAnchor      string               // DOI or reference
    InferenceChain      []InferenceStep      // How we got here
    KBVersions          map[string]string    // KB versions used
    Timestamp           time.Time
    Checksum            string               // SHA256 for integrity
}

type RecommendationClass string
const (
    ClassI   RecommendationClass = "I"    // Must do
    ClassIIa RecommendationClass = "IIa"  // Should do
    ClassIIb RecommendationClass = "IIb"  // Reasonable to consider
    ClassIII RecommendationClass = "III"  // Harmful → avoid
)
```

### 5. ArbitratedDecision (The Heart Object)
```go
// File: internal/models/arbitrated_decision.go

type ArbitratedDecision struct {
    ID                uuid.UUID
    DecisionType      DecisionType  // DO, DELAY, AVOID, CONSIDER
    Target            string        // What action/medication
    Rationale         string        // Why this decision
    SafetyFlags       []SafetyFlag
    Dependencies      []string      // Other decisions this depends on
    Evidence          EvidenceEnvelope
    SourceProtocol    string        // Which protocol generated this
    ArbitrationReason string        // Why this won over alternatives
}

type DecisionType string
const (
    DecisionDo       DecisionType = "DO"
    DecisionDelay    DecisionType = "DELAY"
    DecisionAvoid    DecisionType = "AVOID"
    DecisionConsider DecisionType = "CONSIDER"
)
```

### 6. RecommendationBundle (Output)
```go
// File: internal/models/recommendation_bundle.go

type RecommendationBundle struct {
    ID                   uuid.UUID
    PatientID            uuid.UUID
    Timestamp            time.Time
    Decisions            []ArbitratedDecision
    ProtocolEvaluations  []ProtocolEvaluation
    NarrativeSummary     string                // Human-readable explanation
    ExecutionPlan        ExecutionPlan         // Bindings to KB-3/KB-12/KB-14
    ConflictsResolved    []ConflictResolution
    SafetyGatesApplied   []SafetyGate
}
```

---

## Phase 3: Arbitration Engine

### Core Arbitration Pipeline (8 Steps)

```go
// File: internal/arbitration/engine.go

type ArbitrationEngine struct {
    vaidshalaClient    *clients.VaidshalaClient
    kb3Client          *clients.KB3TemporalClient
    kb8Client          *clients.KB8CalculatorClient
    kb12Client         *clients.KB12OrderSetClient
    kb14Client         *clients.KB14GovernanceClient
    conflictDetector   *ConflictDetector
    priorityHierarchy  *PriorityHierarchy
    safetyGatekeeper   *SafetyGatekeeper
    narrativeGenerator *narrative.Generator
    log                *logrus.Entry
}

func (e *ArbitrationEngine) Execute(ctx context.Context,
    patientCtx *models.PatientContext) (*models.RecommendationBundle, error) {

    // Step 1: Collect candidate protocols
    candidates, err := e.collectCandidateProtocols(ctx, patientCtx)

    // Step 2: Filter ineligible protocols
    eligible := e.filterIneligible(candidates, patientCtx)

    // Step 3: Identify conflicts
    conflicts := e.conflictDetector.DetectConflicts(eligible)

    // Step 4: Apply priority hierarchy
    prioritized := e.priorityHierarchy.Resolve(eligible, conflicts)

    // Step 5: Apply safety gatekeepers
    safeDecisions := e.safetyGatekeeper.Apply(prioritized, patientCtx)

    // Step 6: Assign recommendation strength
    gradedDecisions := e.gradeRecommendations(safeDecisions)

    // Step 7: Produce narrative
    narrative := e.narrativeGenerator.Generate(gradedDecisions, conflicts)

    // Step 8: Bind execution to KB-3/KB-12/KB-14
    executionPlan := e.bindExecution(ctx, gradedDecisions)

    return &models.RecommendationBundle{
        ID:                  uuid.New(),
        PatientID:           patientCtx.PatientID,
        Timestamp:           time.Now(),
        Decisions:           gradedDecisions,
        NarrativeSummary:    narrative,
        ExecutionPlan:       executionPlan,
        ConflictsResolved:   conflicts,
    }, nil
}
```

### Conflict Detection Matrix
```go
// File: internal/arbitration/conflict_detector.go

type ConflictMatrix struct {
    ProtocolA      string
    ProtocolB      string
    ConflictType   ConflictType
    ResolutionRule string
}

type ConflictType string
const (
    ConflictHemodynamic    ConflictType = "HEMODYNAMIC"    // Sepsis fluids vs HF
    ConflictAnticoagulation ConflictType = "ANTICOAGULATION" // AFib vs thrombocytopenia
    ConflictNephrotoxic    ConflictType = "NEPHROTOXIC"    // Drug vs AKI
    ConflictPregnancy      ConflictType = "PREGNANCY"      // Teratogenic risk
    ConflictNeurological   ConflictType = "NEUROLOGICAL"   // Anticoag vs ICH
)

// Pre-defined conflict rules
var KnownConflicts = []ConflictMatrix{
    {ProtocolA: "SEPSIS-FLUIDS", ProtocolB: "HF-DIURESIS",
     ConflictType: ConflictHemodynamic, ResolutionRule: "SEPSIS_WINS_IN_SHOCK"},
    {ProtocolA: "AFIB-ANTICOAG", ProtocolB: "THROMBOCYTOPENIA",
     ConflictType: ConflictAnticoagulation, ResolutionRule: "BLEEDING_SAFETY_WINS"},
    // ... more conflict rules
}
```

### Priority Hierarchy
```go
// File: internal/arbitration/priority_hierarchy.go

// Order of dominance:
// 1. Life-preserving / resuscitation
// 2. Organ-failure stabilization
// 3. Immediate morbidity prevention
// 4. Long-term chronic optimization

func (h *PriorityHierarchy) Resolve(protocols []ProtocolEvaluation,
    conflicts []ConflictMatrix) []ArbitratedDecision {

    // Sort by priority class
    sort.Slice(protocols, func(i, j int) bool {
        return protocols[i].PriorityClass < protocols[j].PriorityClass
    })

    // For each conflict, winner takes action, loser becomes DELAY/AVOID
    for _, conflict := range conflicts {
        winner, loser := h.determineWinner(conflict, protocols)
        // Mark loser as delayed with justification
    }

    return decisions
}
```

### Safety Gatekeeper (Global Veto Logic)
```go
// File: internal/arbitration/safety_gatekeeper.go

type SafetyGatekeeper struct {
    icuSafetyEngine *icu.ICUSafetyRulesEngine  // From Vaidshala
}

func (g *SafetyGatekeeper) Apply(decisions []ArbitratedDecision,
    ctx *PatientContext) []ArbitratedDecision {

    for i, decision := range decisions {
        // Check against ICU state
        if ctx.ICUStateSummary != nil {
            violations := g.icuSafetyEngine.EvaluateMedication(
                decision.Target, ctx.ICUStateSummary)

            if hasBlockingViolation(violations) {
                decisions[i].DecisionType = DecisionAvoid
                decisions[i].SafetyFlags = append(decisions[i].SafetyFlags,
                    SafetyFlag{Type: "ICU_HARD_BLOCK", Reason: violations[0].Reason})
                decisions[i].Evidence.RecommendationClass = ClassIII
            }
        }

        // Check pregnancy
        if ctx.PregnancyStatus != nil && ctx.PregnancyStatus.IsPregnant {
            // Pregnancy safety wins everything
        }

        // Check renal state
        // Check coagulation state
        // etc.
    }

    return decisions
}
```

---

## Phase 4: KB Client Integrations

### Vaidshala CQL Client (Critical)
```go
// File: internal/clients/vaidshala_client.go

type VaidshalaClient struct {
    baseURL    string
    httpClient *http.Client
    log        *logrus.Entry
}

// Get CQL truth flags for a patient
func (c *VaidshalaClient) EvaluateCQL(ctx context.Context,
    patientID uuid.UUID) (map[string]bool, error) {

    // Calls Vaidshala CQL Engine
    // Returns: {"HasHFrEF": true, "OnARNI": false, ...}
}

// Get clinical execution context (snapshot)
func (c *VaidshalaClient) GetClinicalContext(ctx context.Context,
    patientID uuid.UUID) (*contracts.ClinicalExecutionContext, error)
```

### KB-3 Temporal Client (Execution Binding)
```go
// File: internal/clients/kb3_temporal_client.go

func (c *KB3TemporalClient) BindTiming(ctx context.Context,
    decision ArbitratedDecision) (*TemporalBinding, error) {

    // Schedule follow-up, set deadlines, create alerts
}
```

### KB-12 OrderSet Client (Execution Binding)
```go
// File: internal/clients/kb12_orderset_client.go

func (c *KB12OrderSetClient) ActivateOrderSet(ctx context.Context,
    decision ArbitratedDecision) (*OrderSetActivation, error) {

    // Activate concrete orders based on decision
}
```

---

## Phase 5: API Endpoints

### Core Endpoints
```
POST /api/v1/execute
  → Execute protocol arbitration for patient
  → Input: PatientContext
  → Output: RecommendationBundle

POST /api/v1/evaluate
  → Evaluate specific protocol for patient
  → Input: PatientID + ProtocolID
  → Output: ProtocolEvaluation

GET /api/v1/protocols
  → List available protocols
  → Output: []ProtocolDescriptor

GET /api/v1/decisions/:patientId
  → Get recent decisions for patient
  → Output: []ArbitratedDecision

GET /health
GET /ready
GET /metrics
```

---

## Phase 6: Critical Files to Reference

| Component | Reference File |
|-----------|----------------|
| CQL Engine Integration | `vaidshala/clinical-runtime-platform/engines/cql_engine.go` |
| Evidence Envelope | `vaidshala/.../medication-advisor-engine/evidence/envelope.go` |
| KB Client Pattern | `vaidshala/.../kbclients/interfaces.go` |
| ICU Safety Rules | `vaidshala/.../advisor/icu_safety_rules.go` |
| ICU Clinical State | `vaidshala/.../advisor/icu_clinical_state.go` |
| KB-12 Structure | `kb-12-ordersets-careplans/` |
| KB-14 Governance | `kb-14-care-navigator/` |
| Standard KB Pattern | `kb-1-drug-rules/cmd/server/main.go` |

---

## Phase 0: CQL Content Strategy (PREREQUISITE)

### Strategic Insight: Adopt First, Author Last

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CQL SOURCING DECISION TREE                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│              Does validated CQL exist?                                      │
│                       │                                                     │
│       ┌───────────────┼───────────────┐                                    │
│      YES           PARTIAL            NO                                    │
│       │               │               │                                     │
│    ADOPT           ADAPT           AUTHOR                                   │
│    As-Is         & Extend          New CQL                                  │
│       │               │               │                                     │
│   Cost: 0 days   Cost: 1-2 wks   Cost: 4-8 wks                            │
│                                                                             │
│   Examples:      Examples:        Examples:                                 │
│   • CMS eCQM     • WHO DAK →      • ICMR-specific                          │
│   • AHRQ CDS       Indian codes   • NHM protocols                          │
│   • Da Vinci     • US thresholds  • Local formulary                        │
│                    → Asian pop                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Coverage by Demographic

| Demographic | Ready CQL | Adapt Needed | Author Needed |
|-------------|:---------:|:------------:|:-------------:|
| **Old Age** | 80% | 10% | 10% |
| **Chronic Care** | 70% | 20% | 10% |
| **Maternity** | 40% | 30% | 30% |
| **Pediatric** | 20% | 30% | 50% |

### 6-Tier CQL Library Architecture

```
TIER 0: FHIR Foundation
├── FHIRHelpers-4.3.000.cql
├── FHIRCommon-1.1.000.cql
└── USCoreCommon-1.0.000.cql

TIER 1: CQL Primitives
├── QICoreCommon-2.0.000.cql
├── QICorePatterns-1.0.000.cql
├── Status-1.6.000.cql           ← MISSING in current setup
└── UCUMUnits-1.0.000.cql

TIER 2: CQM Infrastructure
├── CQMCommon-2.0.000.cql
├── SupplementalDataElements-3.4.000.cql
├── CumulativeMedicationDuration-4.0.000.cql  ← CRITICAL for adherence
├── AdultOutpatientEncounters-4.0.000.cql     ← Used by 40+ measures
├── AdvancedIllnessAndFrailty-1.8.000.cql     ← Critical for Old Age
├── Hospice-6.9.000.cql
└── PalliativeCare-1.9.000.cql

TIER 3: Domain Commons (Build These)
├── CardiovascularCommon-1.0.000.cql
├── DiabetesCommon-1.0.000.cql
├── RenalCommon-1.0.000.cql
├── MaternalHealthCommon-1.0.000.cql
├── PediatricCommon-1.0.000.cql
└── ClinicalCalculatorsCommon-1.0.000.cql

TIER 4: Measures & Guidelines (Import from CMS/WHO)
├── CMS eCQM 2025 Bundle (60+ measures)
├── AHRQ CDS Libraries
├── WHO DAK Libraries (ANC, PNC, Immunization)
└── CDC Guideline Libraries

TIER 5: India Adaptation Layer
├── IndiaTerminologyAdapter-1.0.000.cql
├── IndiaDrugCodeAdapter-1.0.000.cql
├── IndiaANCAdapter-1.0.000.cql
├── IndiaImmunizationAdapter-1.0.000.cql
└── IndiaASCVDAdapter-1.0.000.cql

TIER 6: Application Layer (KB Engines)
├── DosingRulesEngine.cql        → KB-1
├── ClinicalContextEngine.cql    → KB-2
├── GuidelineEngine.cql          → KB-3
├── SafetyOverrideEngine.cql     → KB-4
├── DDIEngine.cql                → KB-5
├── FormularyEngine.cql          → KB-6
├── TerminologyEngine.cql        → KB-7
├── ClinicalCalculatorsEngine.cql → KB-8 (NEW)
├── CareGapsEngine.cql           → KB-9 (NEW)
└── CDSSOrchestrator.cql         → KB-19
```

### Immediate Action: Download CMS 2025 eCQM Bundle

```bash
# Day 1 task - zero authoring needed
wget https://ecqi.healthit.gov/sites/default/files/2025-eCQM-Specifications.zip
unzip 2025-eCQM-Specifications.zip -d vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/
```

### Priority CQL Imports (Week 1)

| Library | Source | Target KB | Coverage |
|---------|--------|-----------|----------|
| CMS122 (Diabetes HbA1c) | eCQI.gov | KB-3, KB-9 | Chronic |
| CMS165 (BP Control) | eCQI.gov | KB-3, KB-9 | Chronic |
| CMS144/135 (Heart Failure) | eCQI.gov | KB-3, KB-9 | Chronic |
| CMS2 (Depression) | eCQI.gov | KB-3, KB-9 | Mental Health |
| CMS139 (Falls Risk) | eCQI.gov | KB-8, KB-9 | Old Age |
| WHO ANC DAK | WHO | KB-3 (adapt) | Maternity |

### Files to Create/Modify
```
vaidshala/clinical-knowledge-core/
├── tier-0-fhir/                    # Download from HL7
│   ├── FHIRHelpers-4.3.000.cql
│   └── FHIRCommon-1.1.000.cql
├── tier-1-primitives/              # Download from CQF
│   ├── QICoreCommon-2.0.000.cql
│   ├── Status-1.6.000.cql          ← ADD THIS
│   └── UCUMUnits-1.0.000.cql
├── tier-2-cqm-infra/               # Download from CMS
│   ├── CQMCommon-2.0.000.cql
│   ├── AdvancedIllnessAndFrailty-1.8.000.cql  ← ADD THIS
│   └── CumulativeMedicationDuration-4.0.000.cql ← ADD THIS
├── tier-3-domain-commons/          # BUILD THESE
│   ├── CardiovascularCommon.cql
│   ├── DiabetesCommon.cql
│   └── MaternalHealthCommon.cql
├── tier-4-guidelines/
│   ├── cms-ecqm/                   # Import 60+ measures
│   ├── who-dak/                    # Import WHO DAKs
│   └── specialty/                  # Author custom
└── tier-5-india-adapters/          # BUILD THESE
    ├── IndiaTerminologyAdapter.cql
    └── IndiaANCAdapter.cql
```

---

## Phase 1: KB-19 Foundation (Week 1)

### Directory Structure
```
backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   ├── config/
│   ├── models/          # 6 domain models
│   ├── arbitration/     # Core engine
│   ├── clients/         # Vaidshala, KB-3, KB-8, KB-12, KB-14
│   └── narrative/
├── protocols/           # YAML/TOML protocol configs
├── conflicts/           # YAML conflict matrix
└── go.mod
```

### Tasks
- [ ] Create directory structure
- [ ] Implement 6 domain models (PatientContext, ProtocolDescriptor, ProtocolEvaluation, EvidenceEnvelope, ArbitratedDecision, RecommendationBundle)
- [ ] Set up main.go with signal handling
- [ ] Implement config loading with KB service URLs

---

## Phase 2: KB Clients (Week 2)

### Vaidshala Client (Critical Path)
```go
// internal/clients/vaidshala_client.go
type VaidshalaClient interface {
    EvaluateCQL(ctx, patientID) (map[string]interface{}, error)
    GetClinicalContext(ctx, patientID) (*ClinicalExecutionContext, error)
}
```

### Other Clients
- [ ] KB3TemporalClient - Schedule deadlines, create alerts
- [ ] KB8CalculatorClient - Get risk scores (CHA2DS2-VASc, SOFA, etc.)
- [ ] KB12OrderSetClient - Activate order bundles
- [ ] KB14GovernanceClient - Create audit tasks

---

## Phase 3: Arbitration Engine (Week 3)

### Core Components
1. **ConflictDetector** - Reads conflict matrix from YAML
2. **PriorityHierarchy** - Emergency > Acute > Chronic
3. **SafetyGatekeeper** - ICU state integration
4. **RecommendationGrader** - Assign Class I/IIa/IIb/III
5. **ArbitrationEngine** - Orchestrates all above

### Conflict Matrix (YAML-driven)
```yaml
# conflicts/hemodynamic.yaml
conflicts:
  - protocol_a: "SEPSIS-FLUIDS"
    protocol_b: "HF-DIURESIS"
    type: HEMODYNAMIC
    resolution:
      winner: "SEPSIS-FLUIDS"
      condition: "ShockState != NONE"
```

---

## Phase 4: API & Narrative (Week 4)

### Endpoints
```
POST /api/v1/execute           # Main orchestration endpoint
POST /api/v1/evaluate          # Evaluate single protocol
GET  /api/v1/protocols         # List available protocols
GET  /api/v1/decisions/:id     # Get decision history
GET  /health, /ready, /metrics # Ops endpoints
```

### Narrative Generator
- Clinical explanation of decisions
- Conflict resolution rationale
- Evidence citations

---

## Key Design Principles

1. **KB-19 is STATELESS** - No protocol logic lives here
2. **KB-19 DELEGATES truth** - CQL Engine owns clinical truth
3. **KB-19 OWNS synthesis** - Arbitration is the unique value
4. **KB-19 EXPLAINS itself** - Every decision has narrative + evidence
5. **KB-19 BINDS execution** - But never executes directly

---

## Port Assignment
```
KB-19 Protocol Orchestrator: 8099
```

---

## Dependencies
- Vaidshala CQL Engine (required)
- KB-3 Guidelines (temporal binding)
- KB-8 Calculator (risk scores)
- KB-12 OrderSets (order activation)
- KB-14 Governance (task escalation)
- PostgreSQL (decision audit storage)
- Redis (optional caching)
