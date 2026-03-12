# SaMD Medication Service Platform v2.0 - Implementation Plan

## Executive Summary

This document outlines the implementation plan for enhancing the existing Knowledge Base Services to meet the SaMD (Software as Medical Device) Medication Service Platform v2.0 specifications. The plan identifies existing components, gaps, and provides a detailed roadmap for implementing missing features.

---

## Current State Analysis

### ✅ **Already Implemented Components**

#### 1. **Evidence Envelope v1.0**
- **Location**: `migrations/014_enhanced_evidence_envelope.sql`
- **Status**: Database schema exists with transaction tracking
- **Features**: 
  - Transaction ID tracking
  - KB version snapshots
  - Decision chain logging
  - Performance metrics
  - Audit trails

#### 2. **Safety Signal Detection System**
- **Location**: `migrations/015_safety_signals_unified.sql`
- **Status**: TimescaleDB infrastructure operational
- **Features**:
  - Real-time safety monitoring
  - Signal classification by severity
  - Automated alert generation
  - Continuous aggregates for analytics

#### 3. **Knowledge Base Services Architecture**
- **Status**: 7 KB services foundation deployed
- **Services**:
  - KB-Drug-Rules (port 8081) - TOML-based drug calculations
  - KB-DDI (port 8082) - Drug-drug interactions
  - KB-Patient-Safety (port 8083) - Safety profiles
  - KB-Clinical-Pathways (port 8084) - Decision pathways
  - KB-Formulary (port 8085) - Insurance/costs
  - KB-Terminology (port 8086) - Code mappings
  - KB-Drug-Master (port 8087) - Comprehensive drug database

#### 4. **API Gateway with Apollo Federation**
- **Location**: `api-gateway/src/index.ts`
- **Features**:
  - GraphQL federation
  - Basic orchestration
  - Version-aware data sources
  - Plugin architecture

#### 5. **Clinical Governance Foundation**
- **Status**: Implemented
- **Features**:
  - Ed25519 digital signatures
  - Content SHA256 hashing
  - Dual approval workflows
  - Regional compliance support (US/EU/CA/AU)

---

## 🎯 Gap Analysis: Missing SaMD v2.0 Components

### Critical Missing Components

#### 1. **Four-Phase Runtime with ORB Engine** ❌
- Intent Classification with Intent Manifest v2.0
- Clinical Safety Gates at each phase
- Sub-100ms total latency orchestration
- Emergency protocol handlers

#### 2. **Enhanced Evidence Envelope v2.0** ❌
- Clinical context with encounter types
- Safety attestations structure
- Clinical override capabilities
- Complete "decision DNA" tracing

#### 3. **Medication Safety Engine (Rust)** ❌
- Deterministic safety validation
- Absolute contraindication checks
- Dose boundary enforcement
- Interaction severity assessment

#### 4. **Clinical Intelligence Engine** ❌
- Multi-dimensional medication scoring
- Guideline adherence tracking
- Patient-specific factor analysis
- Cost-effectiveness calculations

#### 5. **Clinical Override Protocol** ❌
- Three-level override system (L1/L2/L3)
- 2FA for safety warnings
- Department head approval for contraindications
- Mandatory structured documentation

#### 6. **Progressive Deployment System** ❌
- Shadow mode testing
- Staged rollout (5% → 25% → 50% → 100%)
- Kill switch (<100ms activation)
- Automatic rollback capabilities

---

## 📋 Implementation Roadmap

### **Phase 1: Core SaMD Components (Weeks 1-2)**

#### 1.1 Four-Phase ORB Orchestrator
```
Directory: orchestrator/orb-engine/
```

**Components to Build:**
- `intent_classifier.go` - ML-based intent classification
- `manifest_generator.go` - Intent Manifest v2.0 generation
- `phase_controller.go` - Phase timing and coordination
- `safety_gates.go` - Clinical safety checkpoints

**Key Features:**
```go
type ORBEngine struct {
    Phase1IntentClassification   *IntentClassifier  // ≤25ms
    Phase2ContextAssembly        *ContextAssembler  // ≤50ms
    Phase3IntelligenceEngine     *IntelligenceEngine // ≤75ms
    Phase4DeliveryOrchestrator   *DeliveryEngine    // ≤25ms
}

type IntentManifestV2 struct {
    TransactionID    string
    Intent          ClinicalIntent
    ConfidenceScore float64
    SafetyFlags     []SafetyFlag
    RequiredKBs     []KBRequirement
    ClinicalContext ClinicalContext
}
```

#### 1.2 Enhanced Evidence Envelope v2.0
```
File: api-gateway/src/models/EvidenceEnvelopeV2.ts
```

**Structure Enhancement:**
```typescript
interface EvidenceEnvelopeV2 {
    version: "2.0";
    transactionId: string;
    clinicalContext: {
        encounterType: "ambulatory" | "emergency" | "inpatient";
        urgency: "routine" | "urgent" | "emergent";
        decisionSupportLevel: "tier_1_automated" | "tier_2_augmented" | "tier_3_advisory";
    };
    provenance: {
        kbSnapshots: Record<string, KBSnapshot>;
        decisionChain: DecisionPoint[];
    };
    safetyAttestations: SafetyAttestation[];
    clinicalOverrides?: ClinicalOverride;
    performanceMetrics: PerformanceMetrics;
}
```

#### 1.3 Medication Safety Engine (Rust)
```
Directory: medication-safety-engine/
```

**Rust Implementation:**
```rust
// medication-safety-engine/src/lib.rs
pub struct MedicationSafetyEngine {
    version: String,
    rules: Vec<SafetyRule>,
    contraindication_db: ContraindicationDatabase,
}

impl MedicationSafetyEngine {
    pub fn validate(&self, proposal: &MedicationProposal) -> SafetyResult {
        let checks = vec![
            self.check_absolute_contraindications(),
            self.check_dose_boundaries(),
            self.check_interaction_severity(),
            self.check_duplicate_therapy(),
            self.check_renal_hepatic_adjustment(),
        ];
        
        SafetyResult {
            passed: checks.iter().all(|c| c.is_safe()),
            vetos: checks.iter().filter(|c| c.is_veto()).collect(),
            warnings: checks.iter().filter(|c| c.is_warning()).collect(),
            evidence_trail: self.generate_evidence_trail(checks),
        }
    }
}
```

---

### **Phase 2: Clinical Intelligence Layer (Weeks 3-4)**

#### 2.1 Clinical Decision Support Engine
```
Directory: clinical-intelligence/decision-engine/
```

**Components:**
- `scoring_engine.go` - Multi-factor medication scoring
- `guideline_adherence.go` - Guideline compliance tracking
- `patient_factors.go` - Patient-specific adjustments
- `proposal_generator.go` - Evidence-backed recommendations

**Implementation:**
```go
type ClinicalIntelligenceEngine struct {
    ScoringWeights ScoringConfiguration
    GuidelineDB    GuidelineDatabase
    FormularyDB    FormularyDatabase
}

func (c *ClinicalIntelligenceEngine) GenerateRecommendations(
    context PatientContext,
) ([]MedicationProposal, error) {
    candidates := c.generateCandidates(context)
    filtered := c.applySafetyFilters(candidates)
    scored := c.scoreProposals(filtered, ScoringWeights{
        GuidelineAdherence:     0.30,
        PatientSpecificFactors: 0.25,
        FormularyPreference:    0.15,
        CostEffectiveness:      0.15,
        AdherenceLikelihood:    0.15,
    })
    return c.generateEvidenceBackedProposals(scored)
}
```

#### 2.2 Safety Gates Implementation
```
Directory: orchestrator/safety-gates/
```

**Three-Gate System:**
```go
// Gate 1: Intent Classification Safety
type SafetyGate1 struct {
    EmergencyProtocols []EmergencyProtocol
    SafetyThresholds   SafetyThresholds
}

// Gate 2: Context Assembly Safety
type SafetyGate2 struct {
    DataQualityChecks   []QualityCheck
    FallbackStrategies  []FallbackStrategy
}

// Gate 3: Intelligence Engine Safety
type SafetyGate3 struct {
    ClinicalValidation  ClinicalValidator
    ConservativeDefaults DefaultGenerator
}
```

#### 2.3 Clinical Override Protocol
```
Directory: governance/clinical-override/
```

**Override Levels:**
```go
type OverrideProtocol struct {
    Levels []OverrideLevel{
        {
            Level: "L1",
            Description: "Guideline deviation with reason",
            RequiredAuth: StandardAuth,
            AuditLevel: "standard",
        },
        {
            Level: "L2", 
            Description: "Safety warning override",
            RequiredAuth: TwoFactorAuth,
            AuditLevel: "enhanced",
            NotifyList: []string{"pharmacy", "safety_committee"},
        },
        {
            Level: "L3",
            Description: "Absolute contraindication override",
            RequiredAuth: DepartmentHeadApproval,
            AuditLevel: "critical",
            ReviewCycle: "72_hours",
        },
    }
}
```

---

### **Phase 3: Advanced Features (Weeks 5-6)**

#### 3.1 Graph-Based Guideline Engine
```
Directory: kb-guideline-evidence/graph-engine/
```

**Graph Navigation:**
```yaml
guideline_graph:
  id: "htn_ckd_2025"
  nodes:
    - id: "initial_assessment"
      type: "decision"
      question: "BP ≥ 140/90 on 2+ occasions?"
      edges:
        yes: "stage_classification"
        no: "lifestyle_counseling"
    
    - id: "stage_classification"
      type: "classification"
      criteria:
        stage_1: "140-159/90-99"
        stage_2: "≥160/≥100"
      edges:
        stage_1: "ckd_assessment"
        stage_2: "immediate_treatment"
```

#### 3.2 Real-Time Clinical Alerts
```
Directory: monitoring/clinical-alerts/
```

**Alert Engine:**
```go
type ClinicalAlertEngine struct {
    PriorityLevels []AlertPriority{
        {
            Type: "critical_interaction",
            Delivery: []string{"in_app_modal", "sms", "pharmacy_queue"},
            RequiresAck: true,
            TimeoutMs: 5000,
        },
        {
            Type: "renal_dose_adjustment",
            Trigger: "egfr_drop > 20%",
            Delivery: []string{"in_app_banner", "daily_summary"},
        },
    }
}
```

#### 3.3 Progressive Deployment System
```
Directory: deployment/progressive-rollout/
```

**Deployment Stages:**
```yaml
deployment_stages:
  stage_0_development:
    environment: "dev"
    data: "synthetic_only"
    validation: "automated_tests"
  
  stage_1_shadow:
    environment: "shadow"
    data: "anonymized_historical"
    validation: "retrospective_analysis"
    success_criteria:
      - agreement_with_clinicians: ">95%"
      - safety_event_detection: "100%"
  
  stage_2_pilot:
    environment: "pilot"
    users: ["early_adopter_clinicians"]
    monitoring: "enhanced"
    
  stage_3_production:
    rollout:
      - week_1: "5%_traffic"
      - week_2: "25%_traffic"
      - week_4: "50%_traffic"
      - week_8: "100%_traffic"
```

---

### **Phase 4: Production Hardening (Weeks 7-8)**

#### 4.1 Performance Optimization

**Target Metrics:**
| Component | Target | Strategy |
|-----------|--------|----------|
| Total Latency | <100ms | 4-phase parallel processing |
| Cache Hit Rate | >95% | 3-tier caching |
| Throughput | 10K RPS | Horizontal scaling |
| Error Rate | <0.1% | Circuit breakers |

#### 4.2 Observability Stack

**Clinical Metrics Dashboard:**
```yaml
clinical_metrics:
  - medication_appropriateness_score
  - guideline_adherence_rate
  - safety_alert_override_rate
  - time_to_therapy_initiation
  - clinical_outcome_tracking

technical_metrics:
  - latency_per_phase: [p50, p95, p99]
  - kb_cache_hit_rates
  - error_rates_by_component
  - circuit_breaker_status

compliance_metrics:
  - audit_trail_completeness: "100%"
  - evidence_traceability: "100%"
  - data_retention_compliance: "100%"
```

#### 4.3 Testing Framework

**Test Categories:**
```
tests/
├── unit/                 # Component-level tests
├── integration/          # Cross-service tests
├── clinical-scenarios/   # Real-world clinical cases
├── replay/              # Historical decision replay
├── performance/         # Load and latency tests
├── compliance/          # SaMD compliance validation
└── chaos/               # Resilience testing
```

---

## 📁 Complete File Structure

```
backend/services/knowledge-base-services/
├── orchestrator/
│   ├── orb-engine/
│   │   ├── intent_classifier.go
│   │   ├── manifest_generator.go
│   │   ├── phase_controller.go
│   │   └── safety_coordinator.go
│   ├── safety-gates/
│   │   ├── gate1_intent.go
│   │   ├── gate2_context.go
│   │   ├── gate3_intelligence.go
│   │   └── emergency_protocol.go
│   └── context-assembly/
│       ├── parallel_fetcher.go
│       ├── phenotype_evaluator.go
│       └── cache_coordinator.go
│
├── medication-safety-engine/ (Rust)
│   ├── src/
│   │   ├── lib.rs
│   │   ├── contraindications.rs
│   │   ├── dose_boundaries.rs
│   │   ├── interactions.rs
│   │   ├── duplicate_therapy.rs
│   │   └── renal_hepatic.rs
│   ├── tests/
│   └── Cargo.toml
│
├── clinical-intelligence/
│   ├── decision-engine/
│   │   ├── scoring_engine.go
│   │   ├── guideline_adherence.go
│   │   ├── patient_factors.go
│   │   └── formulary_optimizer.go
│   ├── recommendation-engine/
│   │   ├── proposal_generator.go
│   │   ├── evidence_compiler.go
│   │   └── ranking_engine.go
│   └── models/
│       └── clinical_models.go
│
├── governance/
│   ├── clinical-override/
│   │   ├── override_protocol.go
│   │   ├── approval_workflow.go
│   │   ├── two_factor_auth.go
│   │   └── audit_handler.go
│   ├── compliance/
│   │   ├── samd_validator.go
│   │   ├── hipaa_compliance.go
│   │   └── regional_compliance.go
│   └── signatures/
│       └── enhanced_signing.go
│
├── monitoring/
│   ├── clinical-alerts/
│   │   ├── alert_engine.go
│   │   ├── delivery_router.go
│   │   ├── priority_handler.go
│   │   └── acknowledgment_tracker.go
│   ├── metrics/
│   │   ├── clinical_metrics.go
│   │   ├── performance_metrics.go
│   │   └── compliance_metrics.go
│   └── dashboards/
│       └── grafana_configs/
│
├── deployment/
│   ├── progressive-rollout/
│   │   ├── shadow_mode.go
│   │   ├── ab_testing.go
│   │   ├── kill_switch.go
│   │   └── rollback_controller.go
│   ├── infrastructure/
│   │   ├── kubernetes/
│   │   └── terraform/
│   └── testing/
│       ├── replay_framework.go
│       ├── clinical_scenarios.go
│       └── compliance_tests.go
│
└── evidence-envelope-v2/
    ├── models/
    │   └── envelope_v2.go
    ├── generators/
    │   └── evidence_generator.go
    └── validators/
        └── envelope_validator.go
```

---

## 🚀 Implementation Priorities

### **Priority 1: Critical Safety Components** (Must Have)
1. **Four-Phase ORB Orchestrator** with safety gates
2. **Medication Safety Engine** (Rust) for deterministic validation
3. **Enhanced Evidence Envelope v2.0** with full traceability
4. **Clinical Override Protocol** with audit compliance

### **Priority 2: Clinical Intelligence** (Should Have)
5. **Clinical Decision Support Engine** with scoring
6. **Real-Time Clinical Alerts** with priority routing
7. **Graph-Based Guideline Engine** for complex pathways

### **Priority 3: Production Excellence** (Nice to Have)
8. **Progressive Deployment System** with kill switch
9. **Advanced Observability** with clinical metrics
10. **Comprehensive Testing Framework** with replay capability

---

## ✅ Success Criteria

### Technical Requirements
- [ ] Sub-100ms total latency achieved (P95)
- [ ] Cache hit rate >95% maintained
- [ ] 10K RPS throughput capability
- [ ] <0.1% error rate in production

### Clinical Safety Requirements
- [ ] Triple safety gates operational
- [ ] 100% contraindication detection rate
- [ ] Zero false negative safety events
- [ ] Complete decision reconstruction capability

### Compliance Requirements
- [ ] SaMD Class IIb compliance achieved
- [ ] HIPAA compliance validated
- [ ] FDA/EMA/TGA regional compliance
- [ ] 100% audit trail completeness

### Operational Requirements
- [ ] Progressive rollout completed successfully
- [ ] Kill switch activation <100ms
- [ ] Rollback capability <5 minutes
- [ ] 99.9% uptime maintained

---

## 📊 Risk Mitigation

### High-Risk Areas
1. **Medication Safety Engine Performance**
   - Mitigation: Rust implementation for deterministic performance
   - Fallback: Conservative defaults if timeout

2. **Clinical Override Misuse**
   - Mitigation: Multi-level authentication and audit trails
   - Monitoring: Real-time override tracking dashboard

3. **Integration Complexity**
   - Mitigation: Phased rollout with shadow mode testing
   - Testing: Comprehensive replay framework

4. **Regulatory Compliance**
   - Mitigation: Built-in compliance validation
   - Documentation: Complete evidence envelope trail

---

## 📅 Timeline

### Sprint 0 (Week 0): Foundation
- Set up repository structure
- Configure development environment
- Initialize Rust safety engine project
- Create base orchestrator framework

### Sprints 1-2 (Weeks 1-2): Core Components
- Implement 4-phase ORB orchestrator
- Build Evidence Envelope v2.0
- Develop Medication Safety Engine
- Create safety gates

### Sprints 3-4 (Weeks 3-4): Clinical Intelligence
- Build decision support engine
- Implement clinical override protocol
- Add guideline adherence tracking
- Create alert system

### Sprints 5-6 (Weeks 5-6): Advanced Features
- Implement graph-based guidelines
- Add progressive deployment
- Build monitoring dashboards
- Create testing framework

### Sprints 7-8 (Weeks 7-8): Production Hardening
- Performance optimization
- Security audit
- Compliance validation
- Documentation completion

---

## 🎯 Next Steps

1. **Review and approve this implementation plan**
2. **Allocate development resources**
3. **Set up development environment with required tools**
4. **Begin Sprint 0 foundation work**
5. **Schedule weekly progress reviews**
6. **Engage clinical advisory board for validation**

---

## 📚 References

- SaMD Guidance: FDA Software as Medical Device Guidelines
- Clinical Decision Support: HL7 CDS Hooks Specification
- FHIR Compliance: FHIR R4 Medication Resources
- Safety Standards: ISO 14971 Medical Device Risk Management
- Performance Benchmarks: Industry SaMD Performance Standards

---

*Document Version: 1.0*  
*Last Updated: 2025-08-25*  
*Status: Ready for Implementation*