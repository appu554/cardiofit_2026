# KB-3 Guidelines Service: Go Implementation Plan

## Executive Summary

**Objective**: Convert existing TypeScript implementation to Go and implement all missing features per README_kb3.md specification.

| Attribute | Current State | Target State |
|-----------|---------------|--------------|
| **Language** | TypeScript/Node.js | Go |
| **Port** | 8084/8085 | 8083 |
| **Temporal Operators** | Not implemented | Allen's Interval Algebra (13 operators) |
| **Pathway Engine** | Partial (GraphQL only) | Full state machine |
| **Protocols** | None defined | 17 protocols (6 acute, 6 chronic, 5 preventive) |
| **Scheduling** | Not implemented | Recurrence engine |
| **REST API** | 4 endpoints | 35+ endpoints per README |

**Timeline**: ~13 working days

---

## Phase 0: Archive Existing TypeScript (Day 1 - 2 hours)

### Objective
Move existing TypeScript code to a reference folder for conversion guidance.

### Actions
```bash
# Create archive folder
mkdir -p _reference_typescript

# Move existing source
mv src/ _reference_typescript/
mv database/ _reference_typescript/
mv tests/ _reference_typescript/
mv package.json _reference_typescript/
mv package-lock.json _reference_typescript/
mv tsconfig.json _reference_typescript/
mv simple_kb3_api.js _reference_typescript/
mv federation_endpoint.js _reference_typescript/

# Keep README for reference
cp README_kb3.md _reference_typescript/README_original.md
```

### Files to Archive
| File | Purpose | Conversion Priority |
|------|---------|---------------------|
| `src/engines/conflict_resolver.ts` | Base conflict resolution | High |
| `src/engines/production_conflict_resolver.ts` | 5-tier resolution rules | High |
| `src/engines/safety_override_engine.ts` | Safety override logic | High |
| `src/services/version_manager.ts` | Version lifecycle | High |
| `src/services/database_service.ts` | PostgreSQL operations | High |
| `src/services/neo4j_service.ts` | Graph database queries | High |
| `src/services/cache_service.ts` | Multi-layer caching | Medium |
| `src/services/audit_logger.ts` | Compliance logging | Medium |
| `src/api/guideline_service.ts` | Main business logic | High |
| `src/graphql/resolvers.ts` | GraphQL resolvers | Reference |
| `database/schema.sql` | Database schema | High |

---

## Phase 1: Go Project Structure (Day 1 - 4 hours)

### Initialize Go Module
```bash
go mod init github.com/cardiofit/kb-3-guidelines
```

### Directory Structure
```
kb-3-guidelines/
├── cmd/
│   └── server/
│       └── main.go                 # HTTP server entry point
├── pkg/
│   ├── temporal/                   # NEW: Temporal reasoning
│   │   ├── operators.go            # Allen's Interval Algebra
│   │   ├── pathway.go              # State machine engine
│   │   └── scheduling.go           # Recurrence patterns
│   ├── protocols/                  # NEW: Protocol library
│   │   ├── acute.go                # Sepsis, Stroke, STEMI, DKA, Trauma, PE
│   │   ├── chronic.go              # Diabetes, HF, CKD, Anticoag, COPD, HTN
│   │   ├── preventive.go           # Prenatal, Well Child, Adult, Cancer, Immunizations
│   │   └── registry.go             # Protocol registry
│   ├── governance/                 # CONVERT: From TypeScript
│   │   ├── conflict_resolver.go    # 5-tier resolution engine
│   │   ├── safety_override.go      # Safety override engine
│   │   └── version_manager.go      # Version lifecycle
│   ├── database/                   # CONVERT: From TypeScript
│   │   ├── postgres.go             # Connection pool, transactions
│   │   └── migrations.go           # Schema management
│   ├── graph/                      # CONVERT: From TypeScript
│   │   └── neo4j.go                # Graph queries
│   ├── cache/                      # CONVERT: From TypeScript
│   │   └── cache.go                # Multi-layer caching
│   ├── audit/                      # CONVERT: From TypeScript
│   │   └── logger.go               # Compliance logging
│   ├── api/                        # NEW: REST API
│   │   ├── handlers.go             # HTTP handlers
│   │   ├── middleware.go           # Auth, logging, CORS
│   │   └── routes.go               # Route definitions
│   ├── service/                    # CONVERT: Main business logic
│   │   └── guideline.go            # Guideline service
│   ├── models/                     # Data structures
│   │   ├── guideline.go
│   │   ├── conflict.go
│   │   ├── safety.go
│   │   ├── version.go
│   │   ├── pathway.go              # NEW
│   │   ├── protocol.go             # NEW
│   │   └── schedule.go             # NEW
│   └── config/
│       └── config.go               # Configuration management
├── migrations/
│   ├── 001_initial_schema.sql      # Base schema (from TypeScript)
│   └── 002_temporal_tables.sql     # NEW: Temporal tables
├── test/
│   ├── temporal_test.go
│   ├── pathway_test.go
│   ├── scheduling_test.go
│   ├── protocols_test.go
│   ├── conflict_test.go
│   ├── safety_test.go
│   ├── version_test.go
│   ├── api_test.go
│   └── integration_test.go
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
├── README.md                       # Updated to match implementation
├── README_kb3.md                   # Original spec (reference)
├── IMPLEMENTATION_PLAN.md          # This file
└── _reference_typescript/          # Archived TypeScript
```

### Go Dependencies
```go
// go.mod
module github.com/cardiofit/kb-3-guidelines

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/jackc/pgx/v5 v5.5.0
    github.com/neo4j/neo4j-go-driver/v5 v5.15.0
    github.com/redis/go-redis/v9 v9.3.0
    github.com/google/uuid v1.5.0
    github.com/sirupsen/logrus v1.9.3
    github.com/stretchr/testify v1.8.4
    github.com/spf13/viper v1.18.0
)
```

---

## Phase 2: Core Models (Days 1-2 - 8 hours)

### 2.1 Conflict Models (`pkg/models/conflict.go`)

```go
package models

import "time"

// Conflict represents a detected conflict between guidelines
type Conflict struct {
    ConflictID      string                 `json:"conflict_id"`
    Guideline1ID    string                 `json:"guideline1_id"`
    Guideline2ID    string                 `json:"guideline2_id"`
    Recommendation1 map[string]interface{} `json:"recommendation1"`
    Recommendation2 map[string]interface{} `json:"recommendation2"`
    Type            ConflictType           `json:"type"`
    Severity        Severity               `json:"severity"`
    Domain          string                 `json:"domain"`
    DetectedAt      time.Time              `json:"detected_at"`
}

type ConflictType string
const (
    ConflictTargetDifference    ConflictType = "target_difference"
    ConflictEvidenceDisagreement ConflictType = "evidence_disagreement"
    ConflictTreatmentPreference ConflictType = "treatment_preference"
)

type Severity string
const (
    SeverityCritical Severity = "critical"
    SeverityMajor    Severity = "major"
    SeverityMinor    Severity = "minor"
)

// Resolution represents a conflict resolution decision
type Resolution struct {
    Applicable           bool   `json:"applicable"`
    WinningGuideline     string `json:"winning_guideline,omitempty"`
    Action               any    `json:"action,omitempty"`
    Rationale            string `json:"rationale,omitempty"`
    SafetyOverride       bool   `json:"safety_override,omitempty"`
    OverrideID           string `json:"override_id,omitempty"`
    RequiresManualReview bool   `json:"requires_manual_review,omitempty"`
    RuleUsed             string `json:"rule_used,omitempty"`
}

// PatientContext for conflict resolution
type PatientContext struct {
    PatientID        string                 `json:"patient_id"`
    Age              int                    `json:"age"`
    Sex              string                 `json:"sex"`
    Region           string                 `json:"region,omitempty"`
    PregnancyStatus  string                 `json:"pregnancy_status,omitempty"`
    Labs             map[string]float64     `json:"labs"`
    ActiveConditions []string               `json:"active_conditions"`
    Medications      []string               `json:"medications"`
    Allergies        []string               `json:"allergies"`
    Comorbidities    []string               `json:"comorbidities"`
    RiskFactors      map[string]interface{} `json:"risk_factors"`
}
```

### 2.2 Safety Models (`pkg/models/safety.go`)

```go
package models

import "time"

// SafetyOverride represents a safety override rule
type SafetyOverride struct {
    OverrideID         string                  `json:"override_id"`
    Name               string                  `json:"name"`
    Description        string                  `json:"description"`
    TriggerConditions  SafetyTriggerConditions `json:"trigger_conditions"`
    OverrideAction     SafetyAction            `json:"override_action"`
    Priority           int                     `json:"priority"`
    Active             bool                    `json:"active"`
    AffectedGuidelines []string                `json:"affected_guidelines"`
    EffectiveDate      time.Time               `json:"effective_date"`
    ExpiryDate         *time.Time              `json:"expiry_date,omitempty"`
    RequiresSignature  bool                    `json:"requires_signature"`
    CreatedBy          string                  `json:"created_by"`
    ClinicalRationale  string                  `json:"clinical_rationale"`
}

type SafetyTriggerConditions struct {
    Pregnancy                *bool                   `json:"pregnancy,omitempty"`
    Pediatric                *bool                   `json:"pediatric,omitempty"`
    Geriatric                *bool                   `json:"geriatric,omitempty"`
    Conditions               []string                `json:"conditions,omitempty"`
    Medications              []string                `json:"medications,omitempty"`
    LabThresholds            map[string]LabThreshold `json:"lab_thresholds,omitempty"`
    AllergyContraindications []string                `json:"allergy_contraindications,omitempty"`
    SeverityThreshold        string                  `json:"severity_threshold,omitempty"`
    ClinicalContext          []string                `json:"clinical_context,omitempty"`
}

type LabThreshold struct {
    Operator string  `json:"operator"` // >, >=, <, <=, =
    Value    float64 `json:"value"`
    Unit     string  `json:"unit"`
}

type SafetyAction struct {
    ActionType              SafetyActionType `json:"action_type"`
    Description             string           `json:"description"`
    Parameters              map[string]any   `json:"parameters,omitempty"`
    MonitoringRequirements  []string         `json:"monitoring_requirements,omitempty"`
    AlternativeRecommendations []string      `json:"alternative_recommendations,omitempty"`
    EscalationRequired      bool             `json:"escalation_required,omitempty"`
}

type SafetyActionType string
const (
    ActionContraindicate    SafetyActionType = "contraindicate"
    ActionModifyDose        SafetyActionType = "modify_dose"
    ActionRequireMonitoring SafetyActionType = "require_monitoring"
    ActionSubstituteTherapy SafetyActionType = "substitute_therapy"
    ActionManualReview      SafetyActionType = "manual_review"
)

// SafetyAssessment result
type SafetyAssessment struct {
    PatientID             string                   `json:"patient_id"`
    SafetyScore           int                      `json:"safety_score"` // 0-100
    RiskFactors           []string                 `json:"risk_factors"`
    Contraindications     []Contraindication       `json:"contraindications"`
    Warnings              []Warning                `json:"warnings"`
    RequiredMonitoring    []string                 `json:"required_monitoring"`
    OverrideRecommendations []OverrideRecommendation `json:"override_recommendations"`
    AssessedAt            time.Time                `json:"assessed_at"`
}
```

### 2.3 Temporal Models (`pkg/models/pathway.go`) - NEW

```go
package models

import "time"

// PathwayInstance represents an active pathway for a patient
type PathwayInstance struct {
    InstanceID   string                 `json:"instance_id"`
    PathwayID    string                 `json:"pathway_id"`
    PatientID    string                 `json:"patient_id"`
    CurrentStage string                 `json:"current_stage"`
    Status       PathwayStatus          `json:"status"`
    StartedAt    time.Time              `json:"started_at"`
    CompletedAt  *time.Time             `json:"completed_at,omitempty"`
    Context      map[string]interface{} `json:"context"`
    Actions      []PathwayAction        `json:"actions"`
    AuditLog     []AuditEntry           `json:"audit_log"`
}

type PathwayStatus string
const (
    PathwayActive    PathwayStatus = "active"
    PathwayCompleted PathwayStatus = "completed"
    PathwaySuspended PathwayStatus = "suspended"
    PathwayCancelled PathwayStatus = "cancelled"
)

// PathwayAction represents an action within a pathway
type PathwayAction struct {
    ActionID      string           `json:"action_id"`
    Name          string           `json:"name"`
    Type          ActionType       `json:"type"`
    Status        ConstraintStatus `json:"status"`
    Deadline      time.Time        `json:"deadline"`
    GracePeriod   time.Duration    `json:"grace_period"`
    AlertThreshold time.Duration   `json:"alert_threshold"`
    CompletedAt   *time.Time       `json:"completed_at,omitempty"`
    CompletedBy   string           `json:"completed_by,omitempty"`
    Notes         string           `json:"notes,omitempty"`
}

type ActionType string
const (
    ActionMedication  ActionType = "medication"
    ActionLab         ActionType = "lab"
    ActionProcedure   ActionType = "procedure"
    ActionAssessment  ActionType = "assessment"
    ActionConsult     ActionType = "consult"
    ActionNotification ActionType = "notification"
)

// ConstraintStatus per README specification
type ConstraintStatus string
const (
    StatusPending       ConstraintStatus = "PENDING"
    StatusMet           ConstraintStatus = "MET"
    StatusApproaching   ConstraintStatus = "APPROACHING"
    StatusOverdue       ConstraintStatus = "OVERDUE"
    StatusMissed        ConstraintStatus = "MISSED"
    StatusNotApplicable ConstraintStatus = "NOT_APPLICABLE"
)

// ConstraintEvaluation result
type ConstraintEvaluation struct {
    ActionID        string           `json:"action_id"`
    ActionName      string           `json:"action_name"`
    Status          ConstraintStatus `json:"status"`
    Deadline        time.Time        `json:"deadline"`
    TimeRemaining   time.Duration    `json:"time_remaining,omitempty"`
    TimeOverdue     time.Duration    `json:"time_overdue,omitempty"`
    AlertLevel      string           `json:"alert_level,omitempty"`
}
```

### 2.4 Protocol Models (`pkg/models/protocol.go`) - NEW

```go
package models

import "time"

// Protocol represents a clinical protocol definition
type Protocol struct {
    ProtocolID      string        `json:"protocol_id"`
    Name            string        `json:"name"`
    Type            ProtocolType  `json:"type"`
    GuidelineSource string        `json:"guideline_source"`
    Version         string        `json:"version"`
    Stages          []Stage       `json:"stages"`
    Constraints     []TimeConstraint `json:"constraints"`
    EntryConditions []Condition   `json:"entry_conditions"`
    ExitConditions  []Condition   `json:"exit_conditions"`
    Active          bool          `json:"active"`
    EffectiveDate   time.Time     `json:"effective_date"`
}

type ProtocolType string
const (
    ProtocolAcute      ProtocolType = "acute"
    ProtocolChronic    ProtocolType = "chronic"
    ProtocolPreventive ProtocolType = "preventive"
)

// Stage represents a protocol stage
type Stage struct {
    StageID         string    `json:"stage_id"`
    Name            string    `json:"name"`
    Order           int       `json:"order"`
    Actions         []Action  `json:"actions"`
    EntryConditions []Condition `json:"entry_conditions"`
    ExitConditions  []Condition `json:"exit_conditions"`
    MaxDuration     time.Duration `json:"max_duration,omitempty"`
}

// Action within a stage
type Action struct {
    ActionID    string        `json:"action_id"`
    Name        string        `json:"name"`
    Type        ActionType    `json:"type"`
    Required    bool          `json:"required"`
    Deadline    time.Duration `json:"deadline"` // Relative to stage start
    GracePeriod time.Duration `json:"grace_period"`
    Parameters  map[string]any `json:"parameters,omitempty"`
}

// TimeConstraint for acute protocols
type TimeConstraint struct {
    ConstraintID   string        `json:"constraint_id"`
    Action         string        `json:"action"`
    Deadline       time.Duration `json:"deadline"`
    GracePeriod    time.Duration `json:"grace_period"`
    AlertThreshold time.Duration `json:"alert_threshold"`
    Severity       Severity      `json:"severity"`
    Reference      string        `json:"reference"` // Guideline reference
}

// Condition for entry/exit
type Condition struct {
    Type     string `json:"type"`     // lab, diagnosis, medication, age, etc.
    Field    string `json:"field"`
    Operator string `json:"operator"` // =, !=, >, <, >=, <=, contains, exists
    Value    any    `json:"value"`
}
```

### 2.5 Schedule Models (`pkg/models/schedule.go`) - NEW

```go
package models

import "time"

// ScheduledItem represents a scheduled care item
type ScheduledItem struct {
    ItemID         string             `json:"item_id"`
    PatientID      string             `json:"patient_id"`
    Type           ScheduleItemType   `json:"type"`
    Name           string             `json:"name"`
    Description    string             `json:"description,omitempty"`
    DueDate        time.Time          `json:"due_date"`
    Priority       int                `json:"priority"` // 1=highest, 5=lowest
    IsRecurring    bool               `json:"is_recurring"`
    Recurrence     *RecurrencePattern `json:"recurrence,omitempty"`
    Status         ScheduleStatus     `json:"status"`
    CompletedAt    *time.Time         `json:"completed_at,omitempty"`
    SourceProtocol string             `json:"source_protocol,omitempty"`
    CreatedAt      time.Time          `json:"created_at"`
}

type ScheduleItemType string
const (
    ScheduleLab         ScheduleItemType = "lab"
    ScheduleAppointment ScheduleItemType = "appointment"
    ScheduleMedication  ScheduleItemType = "medication"
    ScheduleProcedure   ScheduleItemType = "procedure"
    ScheduleScreening   ScheduleItemType = "screening"
    ScheduleAssessment  ScheduleItemType = "assessment"
)

type ScheduleStatus string
const (
    SchedulePending   ScheduleStatus = "pending"
    ScheduleCompleted ScheduleStatus = "completed"
    ScheduleOverdue   ScheduleStatus = "overdue"
    ScheduleCancelled ScheduleStatus = "cancelled"
    ScheduleSkipped   ScheduleStatus = "skipped"
)

// RecurrencePattern per README specification
type RecurrencePattern struct {
    Frequency      Frequency  `json:"frequency"`
    Interval       int        `json:"interval"`       // Every N frequency units
    DaysOfWeek     []int      `json:"days_of_week,omitempty"` // 0=Sunday, 6=Saturday
    DayOfMonth     int        `json:"day_of_month,omitempty"`
    MonthOfYear    int        `json:"month_of_year,omitempty"`
    EndDate        *time.Time `json:"end_date,omitempty"`
    MaxOccurrences int        `json:"max_occurrences,omitempty"`
}

type Frequency string
const (
    FreqDaily   Frequency = "daily"
    FreqWeekly  Frequency = "weekly"
    FreqMonthly Frequency = "monthly"
    FreqYearly  Frequency = "yearly"
)

// ChronicSchedule for chronic disease management
type ChronicSchedule struct {
    ScheduleID      string           `json:"schedule_id"`
    Name            string           `json:"name"`
    GuidelineSource string           `json:"guideline_source"`
    MonitoringItems []MonitoringItem `json:"monitoring_items"`
    FollowUpRules   []FollowUpRule   `json:"follow_up_rules"`
}

type MonitoringItem struct {
    ItemID     string            `json:"item_id"`
    Name       string            `json:"name"`
    Type       ScheduleItemType  `json:"type"`
    Recurrence RecurrencePattern `json:"recurrence"`
    Conditions []Condition       `json:"conditions,omitempty"` // When this applies
}

type FollowUpRule struct {
    RuleID    string            `json:"rule_id"`
    Trigger   Condition         `json:"trigger"`
    Action    string            `json:"action"`
    Timing    RecurrencePattern `json:"timing"`
}

// PreventiveSchedule for preventive care
type PreventiveSchedule struct {
    ScheduleID       string             `json:"schedule_id"`
    Name             string             `json:"name"`
    TargetPopulation PopulationCriteria `json:"target_population"`
    ScreeningItems   []ScreeningItem    `json:"screening_items"`
}

type PopulationCriteria struct {
    AgeMin      *int     `json:"age_min,omitempty"`
    AgeMax      *int     `json:"age_max,omitempty"`
    Sex         string   `json:"sex,omitempty"` // M, F, any
    Conditions  []string `json:"conditions,omitempty"`
    RiskFactors []string `json:"risk_factors,omitempty"`
}

type ScreeningItem struct {
    ItemID         string            `json:"item_id"`
    Name           string            `json:"name"`
    Recommendation string            `json:"recommendation"`
    StartAge       int               `json:"start_age"`
    EndAge         int               `json:"end_age"`
    Interval       RecurrencePattern `json:"interval"`
    Sex            string            `json:"sex"` // M, F, any
    EvidenceGrade  string            `json:"evidence_grade"`
    Source         string            `json:"source"` // USPSTF, ACIP, etc.
}
```

---

## Phase 3: Convert Existing Services (Days 2-5)

### 3.1 Database Service (`pkg/database/postgres.go`)
**Source**: `_reference_typescript/src/services/database_service.ts`

**Key Methods to Convert**:
| TypeScript Method | Go Method |
|-------------------|-----------|
| `beginTransaction()` | `BeginTx(ctx) (*sql.Tx, error)` |
| `getActiveGuidelines(region?)` | `GetActiveGuidelines(ctx, region string) ([]Guideline, error)` |
| `getGuidelineById(id)` | `GetGuidelineByID(ctx, id string) (*Guideline, error)` |
| `getConflictsByGuidelineIds(ids)` | `GetConflictsByGuidelineIDs(ctx, ids []string) ([]Conflict, error)` |
| `getSafetyOverrides(activeOnly)` | `GetSafetyOverrides(ctx, activeOnly bool) ([]SafetyOverride, error)` |
| `logSafetyOverride(...)` | `LogSafetyOverride(ctx, log SafetyOverrideLog) error` |

**Estimated Time**: 6 hours

### 3.2 Neo4j Service (`pkg/graph/neo4j.go`)
**Source**: `_reference_typescript/src/services/neo4j_service.ts`

**Key Cypher Queries to Convert**:
- Conflict detection: `MATCH (r1:Recommendation)-[:CONFLICTS_WITH]-(r2:Recommendation)...`
- Guideline by condition: `MATCH (g:Guideline)-[:APPLIES_TO]->(c:Condition)...`
- Safety overrides: `MATCH (o:SafetyOverride)-[:AFFECTS]->(g:Guideline)...`
- Clinical pathway: `MATCH path = (g:Guideline)-[*]->()...`

**Estimated Time**: 8 hours

### 3.3 Cache Service (`pkg/cache/cache.go`)
**Source**: `_reference_typescript/src/services/cache_service.ts`

**Multi-Layer Cache Architecture**:
```
L1 (Memory)  → TTL: 5 min   → sync.Map
L2 (Redis)   → TTL: 30 min  → go-redis
L3 (Database)→ TTL: 24 hrs  → PostgreSQL
```

**Estimated Time**: 4 hours

### 3.4 Audit Logger (`pkg/audit/logger.go`)
**Source**: `_reference_typescript/src/services/audit_logger.ts`

**Features**:
- SHA256 checksum generation
- Event categorization (guideline_management, conflict_resolution, safety_override, etc.)
- Digital signature support
- Compliance logging with JSONB event data

**Estimated Time**: 3 hours

### 3.5 Conflict Resolver (`pkg/governance/conflict_resolver.go`)
**Source**: `_reference_typescript/src/engines/production_conflict_resolver.ts`

**5-Tier Resolution Rules**:
```go
func (r *ConflictResolver) ResolveConflict(conflict Conflict, ctx PatientContext) Resolution {
    // Tier 1: Safety Override - Always wins
    if override := r.checkSafetyOverrides(conflict, ctx); override != nil {
        return Resolution{SafetyOverride: true, OverrideID: override.OverrideID, ...}
    }

    // Tier 2: Regional Preference
    if ctx.Region != "" {
        if match := r.checkRegionalPreference(conflict, ctx.Region); match != "" {
            return Resolution{WinningGuideline: match, RuleUsed: "regional_preference", ...}
        }
    }

    // Tier 3: Evidence Strength (A > B > C > D)
    if winner := r.compareEvidenceGrades(conflict); winner != "" {
        return Resolution{WinningGuideline: winner, RuleUsed: "evidence_strength", ...}
    }

    // Tier 4: Publication Recency (>6 months difference)
    if winner := r.comparePublicationDates(conflict); winner != "" {
        return Resolution{WinningGuideline: winner, RuleUsed: "publication_recency", ...}
    }

    // Tier 5: Conservative Default
    winner := r.determineConservativeOption(conflict)
    return Resolution{WinningGuideline: winner, RuleUsed: "conservative_default", ...}
}
```

**Estimated Time**: 12 hours

### 3.6 Safety Override Engine (`pkg/governance/safety_override.go`)
**Source**: `_reference_typescript/src/engines/safety_override_engine.ts`

**Trigger Evaluation Logic**:
```go
func (e *SafetyOverrideEngine) EvaluateTrigger(override SafetyOverride, ctx PatientContext) bool {
    tc := override.TriggerConditions

    // Pregnancy check
    if tc.Pregnancy != nil && *tc.Pregnancy {
        if ctx.PregnancyStatus == "confirmed" || ctx.PregnancyStatus == "suspected" {
            return true
        }
    }

    // Pediatric check (<18 years)
    if tc.Pediatric != nil && *tc.Pediatric && ctx.Age < 18 {
        return true
    }

    // Geriatric check (>=65 years)
    if tc.Geriatric != nil && *tc.Geriatric && ctx.Age >= 65 {
        return true
    }

    // Condition matching
    if len(tc.Conditions) > 0 {
        for _, cond := range tc.Conditions {
            if contains(ctx.ActiveConditions, cond) {
                return true
            }
        }
    }

    // Lab threshold evaluation
    for labName, threshold := range tc.LabThresholds {
        if labValue, ok := ctx.Labs[labName]; ok {
            if e.evaluateLabThreshold(labValue, threshold) {
                return true
            }
        }
    }

    return false
}

// Safety Score: 100 - (25 × contraindications) - (10 × warnings) - (5 × risks)
func (e *SafetyOverrideEngine) CalculateSafetyScore(assessment SafetyAssessment) int {
    score := 100
    score -= len(assessment.Contraindications) * 25
    score -= len(assessment.Warnings) * 10
    score -= len(assessment.RiskFactors) * 5
    if score < 0 {
        score = 0
    }
    return score
}
```

**Estimated Time**: 10 hours

### 3.7 Version Manager (`pkg/governance/version_manager.go`)
**Source**: `_reference_typescript/src/services/version_manager.ts`

**Clinical Impact Scoring**:
```go
func (m *VersionManager) AssessClinicalImpact(changes []ChangeRecord) ClinicalImpact {
    score := 0
    for _, change := range changes {
        switch change.Field {
        case "evidence_grade":
            score += 15
        case "recommendation_text":
            score += 12
        case "target_value":
            score += 10
        case "safety_considerations":
            score += 20
        case "dosing":
            score += 8
        case "monitoring":
            score += 5
        case "population_criteria":
            score += 8
        }
    }

    level := m.determineImpactLevel(score)
    return ClinicalImpact{Score: score, Level: level, ...}
}

// Impact levels: Critical ≥30, Major ≥15, Minor ≥5, Cosmetic <5
```

**Approval Chain**:
- Always: `technical_lead`
- If clinical change: `clinical_lead`
- If major+critical: `medical_director`, `safety_committee`, `legal_review`
- Emergency: `medical_director` only (expedited)

**Estimated Time**: 12 hours

### 3.8 Guideline Service (`pkg/service/guideline.go`)
**Source**: `_reference_typescript/src/api/guideline_service.ts`

**Main Methods**:
- `GetGuidelines(query GuidelineQuery) (*GuidelineResponse, error)`
- `CompareGuidelines(guidelineIDs []string, domain string) (*ComparisonResult, error)`
- `GetClinicalPathway(conditions, contraindications []string, region string) (*ClinicalPathway, error)`
- `ValidateCrossKBLinks() (*ValidationReport, error)`

**Estimated Time**: 12 hours

---

## Phase 4: Temporal Operators (Days 5-6)

### 4.1 Allen's Interval Algebra (`pkg/temporal/operators.go`)

Per README specification, implement 13 temporal operators:

```go
package temporal

import "time"

// TemporalOperator represents Allen's Interval Algebra operators
type TemporalOperator string

const (
    OpBefore       TemporalOperator = "before"        // Target ends before reference starts
    OpAfter        TemporalOperator = "after"         // Target starts after reference ends
    OpSameAs       TemporalOperator = "same_as"       // Equivalent intervals
    OpMeets        TemporalOperator = "meets"         // Target ends exactly when reference starts
    OpOverlaps     TemporalOperator = "overlaps"      // Intervals share some time period
    OpWithin       TemporalOperator = "within"        // Target within offset of reference
    OpWithinBefore TemporalOperator = "within_before" // Target within offset before reference
    OpWithinAfter  TemporalOperator = "within_after"  // Target within offset after reference
    OpDuring       TemporalOperator = "during"        // Target contained within reference
    OpContains     TemporalOperator = "contains"      // Target contains reference
    OpStarts       TemporalOperator = "starts"        // Both start at same time
    OpEnds         TemporalOperator = "ends"          // Both end at same time
    OpEquals       TemporalOperator = "equals"        // Identical intervals
)

// Interval represents a time interval
type Interval struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}

// Before: Target ends before reference starts
func (i Interval) Before(other Interval) bool {
    return i.End.Before(other.Start)
}

// After: Target starts after reference ends
func (i Interval) After(other Interval) bool {
    return i.Start.After(other.End)
}

// Meets: Target ends exactly when reference starts
func (i Interval) Meets(other Interval) bool {
    return i.End.Equal(other.Start)
}

// Overlaps: Intervals share some time period
func (i Interval) Overlaps(other Interval) bool {
    return i.Start.Before(other.End) && i.End.After(other.Start)
}

// During: Target contained within reference
func (i Interval) During(other Interval) bool {
    return i.Start.After(other.Start) && i.End.Before(other.End)
}

// Contains: Target contains reference
func (i Interval) Contains(other Interval) bool {
    return i.Start.Before(other.Start) && i.End.After(other.End)
}

// Starts: Both start at same time
func (i Interval) Starts(other Interval) bool {
    return i.Start.Equal(other.Start)
}

// Ends: Both end at same time
func (i Interval) Ends(other Interval) bool {
    return i.End.Equal(other.End)
}

// Equals: Identical intervals
func (i Interval) Equals(other Interval) bool {
    return i.Start.Equal(other.Start) && i.End.Equal(other.End)
}

// Within: Target within offset of reference
func (i Interval) Within(other Interval, offset time.Duration) bool {
    return i.Start.After(other.Start.Add(-offset)) && i.End.Before(other.End.Add(offset))
}

// EvaluateTemporalRelation evaluates a temporal relationship
func EvaluateTemporalRelation(target, reference Interval, operator TemporalOperator, offset ...time.Duration) bool {
    switch operator {
    case OpBefore:
        return target.Before(reference)
    case OpAfter:
        return target.After(reference)
    case OpMeets:
        return target.Meets(reference)
    case OpOverlaps:
        return target.Overlaps(reference)
    case OpDuring:
        return target.During(reference)
    case OpContains:
        return target.Contains(reference)
    case OpStarts:
        return target.Starts(reference)
    case OpEnds:
        return target.Ends(reference)
    case OpEquals:
        return target.Equals(reference)
    case OpSameAs:
        return target.Equals(reference)
    case OpWithin:
        if len(offset) > 0 {
            return target.Within(reference, offset[0])
        }
        return false
    case OpWithinBefore:
        if len(offset) > 0 {
            return target.End.Before(reference.Start) &&
                   target.End.After(reference.Start.Add(-offset[0]))
        }
        return false
    case OpWithinAfter:
        if len(offset) > 0 {
            return target.Start.After(reference.End) &&
                   target.Start.Before(reference.End.Add(offset[0]))
        }
        return false
    default:
        return false
    }
}
```

**Estimated Time**: 6 hours

### 4.2 Pathway State Machine (`pkg/temporal/pathway.go`)

```go
package temporal

import (
    "context"
    "time"
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
    "github.com/google/uuid"
)

type PathwayEngine struct {
    db          *database.Service
    neo4j       *graph.Neo4jService
    protocols   *protocols.Registry
    audit       *audit.Logger
}

// StartPathway initiates a new pathway instance
func (e *PathwayEngine) StartPathway(ctx context.Context, pathwayID, patientID string, context map[string]interface{}) (*models.PathwayInstance, error) {
    // Get protocol definition
    protocol, err := e.protocols.GetProtocol(pathwayID)
    if err != nil {
        return nil, err
    }

    // Create instance
    instance := &models.PathwayInstance{
        InstanceID:   "INST-" + uuid.New().String()[:8],
        PathwayID:    pathwayID,
        PatientID:    patientID,
        CurrentStage: protocol.Stages[0].StageID,
        Status:       models.PathwayActive,
        StartedAt:    time.Now(),
        Context:      context,
        Actions:      e.createActionsFromProtocol(protocol, time.Now()),
    }

    // Persist
    if err := e.db.CreatePathwayInstance(ctx, instance); err != nil {
        return nil, err
    }

    // Audit
    e.audit.Log("pathway_started", map[string]interface{}{
        "instance_id": instance.InstanceID,
        "pathway_id":  pathwayID,
        "patient_id":  patientID,
    })

    return instance, nil
}

// EvaluateConstraints evaluates all constraints for a pathway instance
func (e *PathwayEngine) EvaluateConstraints(ctx context.Context, instanceID string) ([]models.ConstraintEvaluation, error) {
    instance, err := e.db.GetPathwayInstance(ctx, instanceID)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    var evaluations []models.ConstraintEvaluation

    for _, action := range instance.Actions {
        if action.Status == models.StatusMet || action.Status == models.StatusNotApplicable {
            continue
        }

        eval := models.ConstraintEvaluation{
            ActionID:   action.ActionID,
            ActionName: action.Name,
            Deadline:   action.Deadline,
        }

        if action.CompletedAt != nil {
            eval.Status = models.StatusMet
        } else if now.After(action.Deadline.Add(action.GracePeriod)) {
            eval.Status = models.StatusMissed
            eval.TimeOverdue = now.Sub(action.Deadline)
        } else if now.After(action.Deadline) {
            eval.Status = models.StatusOverdue
            eval.TimeOverdue = now.Sub(action.Deadline)
        } else if now.After(action.Deadline.Add(-action.AlertThreshold)) {
            eval.Status = models.StatusApproaching
            eval.TimeRemaining = action.Deadline.Sub(now)
        } else {
            eval.Status = models.StatusPending
            eval.TimeRemaining = action.Deadline.Sub(now)
        }

        evaluations = append(evaluations, eval)
    }

    return evaluations, nil
}

// GetPendingActions returns actions not yet completed
func (e *PathwayEngine) GetPendingActions(ctx context.Context, instanceID string) ([]models.PathwayAction, error) {
    instance, err := e.db.GetPathwayInstance(ctx, instanceID)
    if err != nil {
        return nil, err
    }

    var pending []models.PathwayAction
    for _, action := range instance.Actions {
        if action.Status == models.StatusPending || action.Status == models.StatusApproaching {
            pending = append(pending, action)
        }
    }
    return pending, nil
}

// GetOverdueActions returns overdue actions
func (e *PathwayEngine) GetOverdueActions(ctx context.Context, instanceID string) ([]models.PathwayAction, error) {
    instance, err := e.db.GetPathwayInstance(ctx, instanceID)
    if err != nil {
        return nil, err
    }

    var overdue []models.PathwayAction
    now := time.Now()
    for _, action := range instance.Actions {
        if action.CompletedAt == nil && now.After(action.Deadline) {
            overdue = append(overdue, action)
        }
    }
    return overdue, nil
}

// CompleteAction marks an action as completed
func (e *PathwayEngine) CompleteAction(ctx context.Context, instanceID, actionID, completedBy string) error {
    now := time.Now()
    return e.db.UpdatePathwayAction(ctx, instanceID, actionID, map[string]interface{}{
        "status":       models.StatusMet,
        "completed_at": now,
        "completed_by": completedBy,
    })
}

// AdvanceStage moves to the next stage
func (e *PathwayEngine) AdvanceStage(ctx context.Context, instanceID string) error {
    instance, err := e.db.GetPathwayInstance(ctx, instanceID)
    if err != nil {
        return err
    }

    protocol, err := e.protocols.GetProtocol(instance.PathwayID)
    if err != nil {
        return err
    }

    // Find next stage
    currentIdx := -1
    for i, stage := range protocol.Stages {
        if stage.StageID == instance.CurrentStage {
            currentIdx = i
            break
        }
    }

    if currentIdx == -1 || currentIdx >= len(protocol.Stages)-1 {
        // No more stages - complete pathway
        return e.db.UpdatePathwayInstance(ctx, instanceID, map[string]interface{}{
            "status":       models.PathwayCompleted,
            "completed_at": time.Now(),
        })
    }

    nextStage := protocol.Stages[currentIdx+1]
    return e.db.UpdatePathwayInstance(ctx, instanceID, map[string]interface{}{
        "current_stage": nextStage.StageID,
    })
}
```

**Estimated Time**: 8 hours

### 4.3 Scheduling Engine (`pkg/temporal/scheduling.go`)

```go
package temporal

import (
    "context"
    "time"
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
)

type SchedulingEngine struct {
    db    *database.Service
    audit *audit.Logger
}

// AddScheduledItem adds a new scheduled item
func (s *SchedulingEngine) AddScheduledItem(ctx context.Context, patientID string, item models.ScheduledItem) error {
    item.PatientID = patientID
    item.Status = models.SchedulePending
    item.CreatedAt = time.Now()

    if item.ItemID == "" {
        item.ItemID = "SCHED-" + uuid.New().String()[:8]
    }

    return s.db.CreateScheduledItem(ctx, &item)
}

// GetPatientSchedule returns all scheduled items for a patient
func (s *SchedulingEngine) GetPatientSchedule(ctx context.Context, patientID string) ([]models.ScheduledItem, error) {
    return s.db.GetScheduledItems(ctx, patientID, nil)
}

// GetPendingItems returns pending items for a patient
func (s *SchedulingEngine) GetPendingItems(ctx context.Context, patientID string) ([]models.ScheduledItem, error) {
    status := models.SchedulePending
    return s.db.GetScheduledItems(ctx, patientID, &status)
}

// GetOverdueItems returns overdue items for a patient
func (s *SchedulingEngine) GetOverdueItems(ctx context.Context, patientID string) ([]models.ScheduledItem, error) {
    items, err := s.db.GetScheduledItems(ctx, patientID, nil)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    var overdue []models.ScheduledItem
    for _, item := range items {
        if item.Status == models.SchedulePending && now.After(item.DueDate) {
            item.Status = models.ScheduleOverdue
            overdue = append(overdue, item)
        }
    }
    return overdue, nil
}

// GetUpcoming returns items due within N days
func (s *SchedulingEngine) GetUpcoming(ctx context.Context, patientID string, days int) ([]models.ScheduledItem, error) {
    items, err := s.db.GetScheduledItems(ctx, patientID, nil)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    deadline := now.AddDate(0, 0, days)

    var upcoming []models.ScheduledItem
    for _, item := range items {
        if item.Status == models.SchedulePending &&
           item.DueDate.After(now) &&
           item.DueDate.Before(deadline) {
            upcoming = append(upcoming, item)
        }
    }
    return upcoming, nil
}

// CompleteItem marks an item as completed and schedules next occurrence if recurring
func (s *SchedulingEngine) CompleteItem(ctx context.Context, patientID, itemID string) error {
    item, err := s.db.GetScheduledItem(ctx, itemID)
    if err != nil {
        return err
    }

    now := time.Now()

    // Mark current as completed
    if err := s.db.UpdateScheduledItem(ctx, itemID, map[string]interface{}{
        "status":       models.ScheduleCompleted,
        "completed_at": now,
    }); err != nil {
        return err
    }

    // Schedule next occurrence if recurring
    if item.IsRecurring && item.Recurrence != nil {
        nextDue := s.CalculateNextOccurrence(item.DueDate, *item.Recurrence)

        // Check end conditions
        if item.Recurrence.EndDate != nil && nextDue.After(*item.Recurrence.EndDate) {
            return nil // Don't schedule past end date
        }

        nextItem := models.ScheduledItem{
            PatientID:      patientID,
            Type:           item.Type,
            Name:           item.Name,
            Description:    item.Description,
            DueDate:        nextDue,
            Priority:       item.Priority,
            IsRecurring:    true,
            Recurrence:     item.Recurrence,
            SourceProtocol: item.SourceProtocol,
        }

        return s.AddScheduledItem(ctx, patientID, nextItem)
    }

    return nil
}

// CalculateNextOccurrence calculates the next occurrence based on recurrence pattern
func (s *SchedulingEngine) CalculateNextOccurrence(from time.Time, pattern models.RecurrencePattern) time.Time {
    switch pattern.Frequency {
    case models.FreqDaily:
        return from.AddDate(0, 0, pattern.Interval)
    case models.FreqWeekly:
        return from.AddDate(0, 0, 7*pattern.Interval)
    case models.FreqMonthly:
        return from.AddDate(0, pattern.Interval, 0)
    case models.FreqYearly:
        return from.AddDate(pattern.Interval, 0, 0)
    default:
        return from.AddDate(0, 0, 1)
    }
}
```

**Estimated Time**: 6 hours

---

## Phase 5: Protocol Library (Days 6-8)

### 5.1 Acute Protocols (`pkg/protocols/acute.go`)

```go
package protocols

import (
    "time"
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// SepsisProtocol - Surviving Sepsis Campaign 2021, CMS SEP-1
var SepsisProtocol = models.Protocol{
    ProtocolID:      "SEPSIS-SEP1-2021",
    Name:            "Sepsis Bundle - CMS SEP-1",
    Type:            models.ProtocolAcute,
    GuidelineSource: "Surviving Sepsis Campaign 2021",
    Version:         "2021.1",
    Stages: []models.Stage{
        {
            StageID: "recognition",
            Name:    "Sepsis Recognition",
            Order:   1,
            Actions: []models.Action{
                {ActionID: "screen", Name: "Sepsis Screening", Type: models.ActionAssessment, Required: true},
                {ActionID: "lactate_initial", Name: "Initial Lactate", Type: models.ActionLab, Required: true, Deadline: 30 * time.Minute},
            },
        },
        {
            StageID: "3h_bundle",
            Name:    "3-Hour Bundle",
            Order:   2,
            Actions: []models.Action{
                {ActionID: "blood_cultures", Name: "Blood Cultures (before antibiotics)", Type: models.ActionLab, Required: true},
                {ActionID: "antibiotics", Name: "Broad-spectrum Antibiotics", Type: models.ActionMedication, Required: true, Deadline: 1 * time.Hour},
                {ActionID: "fluid_bolus", Name: "30 mL/kg Crystalloid (if hypotension/lactate ≥4)", Type: models.ActionMedication, Required: false, Deadline: 3 * time.Hour},
            },
        },
        {
            StageID: "6h_bundle",
            Name:    "6-Hour Bundle",
            Order:   3,
            Actions: []models.Action{
                {ActionID: "vasopressors", Name: "Vasopressors (if hypotension persists)", Type: models.ActionMedication, Required: false, Deadline: 6 * time.Hour},
                {ActionID: "lactate_repeat", Name: "Repeat Lactate (if initial >2)", Type: models.ActionLab, Required: false, Deadline: 6 * time.Hour},
                {ActionID: "reassess_volume", Name: "Reassess Volume Status", Type: models.ActionAssessment, Required: true, Deadline: 6 * time.Hour},
            },
        },
    },
    Constraints: []models.TimeConstraint{
        {ConstraintID: "abx_1h", Action: "Administer antibiotics", Deadline: 1 * time.Hour, Severity: models.SeverityCritical},
        {ConstraintID: "lactate_3h", Action: "Repeat lactate if initial >2", Deadline: 3 * time.Hour, Severity: models.SeverityMajor},
        {ConstraintID: "fluid_3h", Action: "30ml/kg crystalloid for hypotension/lactate≥4", Deadline: 3 * time.Hour, Severity: models.SeverityMajor},
    },
    EntryConditions: []models.Condition{
        {Type: "diagnosis", Field: "sepsis", Operator: "=", Value: true},
    },
}

// StrokeProtocol - AHA/ASA 2019
var StrokeProtocol = models.Protocol{
    ProtocolID:      "STROKE-AHA-2019",
    Name:            "Acute Ischemic Stroke - AHA/ASA 2019",
    Type:            models.ProtocolAcute,
    GuidelineSource: "AHA/ASA 2019",
    Constraints: []models.TimeConstraint{
        {ConstraintID: "ct_25min", Action: "Door-to-CT", Deadline: 25 * time.Minute, Severity: models.SeverityCritical},
        {ConstraintID: "tpa_60min", Action: "Door-to-needle (tPA)", Deadline: 60 * time.Minute, Severity: models.SeverityCritical},
        {ConstraintID: "tpa_window", Action: "tPA within window", Deadline: 270 * time.Minute, Severity: models.SeverityCritical}, // 4.5 hours
    },
}

// STEMIProtocol - ACC/AHA 2013
var STEMIProtocol = models.Protocol{
    ProtocolID:      "STEMI-ACC-2013",
    Name:            "STEMI - ACC/AHA 2013",
    Type:            models.ProtocolAcute,
    GuidelineSource: "ACC/AHA 2013",
    Constraints: []models.TimeConstraint{
        {ConstraintID: "ecg_10min", Action: "12-lead ECG", Deadline: 10 * time.Minute, Severity: models.SeverityCritical},
        {ConstraintID: "d2b_90min", Action: "Door-to-balloon", Deadline: 90 * time.Minute, Severity: models.SeverityCritical},
    },
}

// DKAProtocol - ADA 2024
var DKAProtocol = models.Protocol{
    ProtocolID:      "DKA-ADA-2024",
    Name:            "Diabetic Ketoacidosis - ADA 2024",
    Type:            models.ProtocolAcute,
    GuidelineSource: "ADA 2024",
    Constraints: []models.TimeConstraint{
        {ConstraintID: "k_before_insulin", Action: "K+ check before insulin", Deadline: 0, Severity: models.SeverityCritical},
        {ConstraintID: "overlap_transition", Action: "2h overlap on SC transition", Deadline: 2 * time.Hour, Severity: models.SeverityMajor},
    },
}

// TraumaProtocol - ATLS 10th Edition
var TraumaProtocol = models.Protocol{
    ProtocolID:      "TRAUMA-ATLS-10",
    Name:            "Trauma - ATLS 10th Edition",
    Type:            models.ProtocolAcute,
    GuidelineSource: "ATLS 10th Edition",
    Constraints: []models.TimeConstraint{
        {ConstraintID: "txa_3h", Action: "TXA administration", Deadline: 3 * time.Hour, Severity: models.SeverityCritical},
    },
}

// PEProtocol - ESC 2019
var PEProtocol = models.Protocol{
    ProtocolID:      "PE-ESC-2019",
    Name:            "Pulmonary Embolism - ESC 2019",
    Type:            models.ProtocolAcute,
    GuidelineSource: "ESC 2019",
    Constraints: []models.TimeConstraint{
        {ConstraintID: "anticoag_1h", Action: "Anticoagulation initiation", Deadline: 1 * time.Hour, Severity: models.SeverityCritical},
    },
}
```

### 5.2 Chronic Schedules (`pkg/protocols/chronic.go`)

```go
package protocols

import (
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// DiabetesSchedule - ADA Standards 2024
var DiabetesSchedule = models.ChronicSchedule{
    ScheduleID:      "DIABETES-ADA-2024",
    Name:            "Diabetes Management - ADA 2024",
    GuidelineSource: "ADA Standards of Care 2024",
    MonitoringItems: []models.MonitoringItem{
        {ItemID: "hba1c", Name: "HbA1c", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 3}},
        {ItemID: "lipid_panel", Name: "Lipid Panel", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
        {ItemID: "eye_exam", Name: "Dilated Eye Exam", Type: models.ScheduleScreening,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
        {ItemID: "foot_exam", Name: "Comprehensive Foot Exam", Type: models.ScheduleAssessment,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
        {ItemID: "uacr", Name: "Urine Albumin/Creatinine Ratio", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
        {ItemID: "egfr", Name: "eGFR", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
    },
}

// HeartFailureSchedule - ACC/AHA/HFSA 2022
var HeartFailureSchedule = models.ChronicSchedule{
    ScheduleID:      "HF-ACCAHA-2022",
    Name:            "Heart Failure Management - ACC/AHA/HFSA 2022",
    GuidelineSource: "ACC/AHA/HFSA 2022",
    MonitoringItems: []models.MonitoringItem{
        {ItemID: "followup_7d", Name: "Post-discharge Follow-up", Type: models.ScheduleAppointment,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqDaily, Interval: 7}},
        {ItemID: "k_after_raas", Name: "K+ after RAAS initiation", Type: models.ScheduleLab},
        {ItemID: "bnp", Name: "BNP/NT-proBNP", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 3}},
    },
    FollowUpRules: []models.FollowUpRule{
        {RuleID: "k_raas", Trigger: models.Condition{Type: "medication", Field: "class", Operator: "=", Value: "RAAS"},
         Action: "Check K+ 3-7 days after initiation/dose change"},
    },
}

// CKDSchedule - KDIGO 2024
var CKDSchedule = models.ChronicSchedule{
    ScheduleID:      "CKD-KDIGO-2024",
    Name:            "CKD Management - KDIGO 2024",
    GuidelineSource: "KDIGO 2024",
    MonitoringItems: []models.MonitoringItem{
        // eGFR frequency varies by stage: G1-G2: yearly, G3a: q6mo, G3b-G4: q3mo, G5: monthly
        {ItemID: "egfr_g1g2", Name: "eGFR (Stage 1-2)", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1},
         Conditions: []models.Condition{{Type: "ckd_stage", Operator: "<=", Value: 2}}},
        {ItemID: "egfr_g3a", Name: "eGFR (Stage 3a)", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 6},
         Conditions: []models.Condition{{Type: "ckd_stage", Operator: "=", Value: "3a"}}},
        {ItemID: "nephrology_referral", Name: "Nephrology Referral", Type: models.ScheduleAppointment,
         Conditions: []models.Condition{{Type: "egfr", Operator: "<", Value: 30}}},
    },
}

// AnticoagSchedule - CHEST Guidelines
var AnticoagSchedule = models.ChronicSchedule{
    ScheduleID:      "ANTICOAG-CHEST",
    Name:            "Anticoagulation Management - CHEST Guidelines",
    GuidelineSource: "CHEST Guidelines",
    MonitoringItems: []models.MonitoringItem{
        {ItemID: "inr_routine", Name: "INR (routine)", Type: models.ScheduleLab,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqWeekly, Interval: 4}},
    },
    FollowUpRules: []models.FollowUpRule{
        {RuleID: "inr_dose_change", Trigger: models.Condition{Type: "dose_change", Operator: "=", Value: true},
         Action: "Recheck INR 3-7 days after dose change"},
    },
}

// COPDSchedule - GOLD 2024
var COPDSchedule = models.ChronicSchedule{
    ScheduleID:      "COPD-GOLD-2024",
    Name:            "COPD Management - GOLD 2024",
    GuidelineSource: "GOLD 2024",
    MonitoringItems: []models.MonitoringItem{
        {ItemID: "cat_score", Name: "CAT Score Assessment", Type: models.ScheduleAssessment,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 3}},
        {ItemID: "spirometry", Name: "Spirometry", Type: models.ScheduleProcedure,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1}},
    },
}

// HTNSchedule - ACC/AHA 2017
var HTNSchedule = models.ChronicSchedule{
    ScheduleID:      "HTN-ACCAHA-2017",
    Name:            "Hypertension Management - ACC/AHA 2017",
    GuidelineSource: "ACC/AHA 2017",
    MonitoringItems: []models.MonitoringItem{
        {ItemID: "bp_monthly", Name: "BP Check (until at goal)", Type: models.ScheduleAssessment,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 1},
         Conditions: []models.Condition{{Type: "bp_at_goal", Operator: "=", Value: false}}},
        {ItemID: "bp_maintenance", Name: "BP Check (at goal)", Type: models.ScheduleAssessment,
         Recurrence: models.RecurrencePattern{Frequency: models.FreqMonthly, Interval: 3},
         Conditions: []models.Condition{{Type: "bp_at_goal", Operator: "=", Value: true}}},
    },
}
```

### 5.3 Preventive Schedules (`pkg/protocols/preventive.go`)

```go
package protocols

import (
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// PrenatalSchedule
var PrenatalSchedule = models.PreventiveSchedule{
    ScheduleID: "PRENATAL",
    Name:       "Prenatal Care Schedule",
    TargetPopulation: models.PopulationCriteria{
        Sex:        "F",
        Conditions: []string{"pregnancy"},
    },
    ScreeningItems: []models.ScreeningItem{
        {ItemID: "first_visit", Name: "First Prenatal Visit", StartAge: 0, EndAge: 50,
         Recommendation: "Confirm pregnancy, dating ultrasound, initial labs"},
        {ItemID: "gct", Name: "Glucose Challenge Test", StartAge: 0, EndAge: 50,
         Recommendation: "GCT at 24-28 weeks gestation"},
        {ItemID: "gbs", Name: "GBS Screening", StartAge: 0, EndAge: 50,
         Recommendation: "GBS culture at 36 weeks"},
    },
}

// WellChildSchedule - EPSDT
var WellChildSchedule = models.PreventiveSchedule{
    ScheduleID: "WELLCHILD",
    Name:       "Well Child Care - EPSDT Schedule",
    TargetPopulation: models.PopulationCriteria{
        AgeMin: intPtr(0),
        AgeMax: intPtr(21),
    },
    ScreeningItems: []models.ScreeningItem{
        {ItemID: "newborn", Name: "Newborn Visit", StartAge: 0, EndAge: 0},
        {ItemID: "1mo", Name: "1 Month Visit", StartAge: 0, EndAge: 1},
        {ItemID: "2mo", Name: "2 Month Visit", StartAge: 2, EndAge: 2},
        {ItemID: "4mo", Name: "4 Month Visit", StartAge: 4, EndAge: 4},
        {ItemID: "6mo", Name: "6 Month Visit", StartAge: 6, EndAge: 6},
        {ItemID: "9mo", Name: "9 Month Visit", StartAge: 9, EndAge: 9},
        {ItemID: "12mo", Name: "12 Month Visit", StartAge: 12, EndAge: 12},
        {ItemID: "developmental", Name: "Developmental Screening", StartAge: 9, EndAge: 30,
         Recommendation: "ASQ-3 or equivalent at 9, 18, 24/30 months"},
    },
}

// AdultPreventiveSchedule - USPSTF
var AdultPreventiveSchedule = models.PreventiveSchedule{
    ScheduleID: "ADULT-PREV",
    Name:       "Adult Preventive Care - USPSTF",
    TargetPopulation: models.PopulationCriteria{
        AgeMin: intPtr(18),
    },
    ScreeningItems: []models.ScreeningItem{
        {ItemID: "bp", Name: "Blood Pressure Screening", StartAge: 18, EndAge: 120,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1},
         Sex: "any", EvidenceGrade: "A"},
        {ItemID: "lipid", Name: "Lipid Screening", StartAge: 40, EndAge: 75,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 5},
         Sex: "any", EvidenceGrade: "B"},
        {ItemID: "diabetes", Name: "Diabetes Screening", StartAge: 35, EndAge: 70,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 3},
         Sex: "any", EvidenceGrade: "B"},
    },
}

// CancerScreeningSchedule - USPSTF
var CancerScreeningSchedule = models.PreventiveSchedule{
    ScheduleID: "CANCER-SCREENING",
    Name:       "Cancer Screening - USPSTF",
    ScreeningItems: []models.ScreeningItem{
        {ItemID: "mammography", Name: "Mammography", StartAge: 50, EndAge: 74,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 2},
         Sex: "F", EvidenceGrade: "B", Source: "USPSTF"},
        {ItemID: "colonoscopy", Name: "Colonoscopy", StartAge: 45, EndAge: 75,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 10},
         Sex: "any", EvidenceGrade: "A", Source: "USPSTF"},
        {ItemID: "cervical", Name: "Cervical Cancer Screening", StartAge: 21, EndAge: 65,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 3},
         Sex: "F", EvidenceGrade: "A", Source: "USPSTF"},
        {ItemID: "lung_ct", Name: "Low-dose CT Lung (high-risk)", StartAge: 50, EndAge: 80,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1},
         Sex: "any", EvidenceGrade: "B", Source: "USPSTF",
         Recommendation: "20+ pack-year history, currently smoke or quit within 15 years"},
    },
}

// ImmunizationSchedule - ACIP
var ImmunizationSchedule = models.PreventiveSchedule{
    ScheduleID: "IMMUNIZATIONS",
    Name:       "Immunization Schedule - ACIP",
    ScreeningItems: []models.ScreeningItem{
        {ItemID: "flu", Name: "Influenza Vaccine", StartAge: 6, EndAge: 120,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 1},
         Sex: "any", Source: "ACIP"},
        {ItemID: "tdap", Name: "Tdap/Td", StartAge: 11, EndAge: 120,
         Interval: models.RecurrencePattern{Frequency: models.FreqYearly, Interval: 10},
         Sex: "any", Source: "ACIP"},
        {ItemID: "shingles", Name: "Shingrix (Shingles)", StartAge: 50, EndAge: 120,
         Sex: "any", Source: "ACIP", Recommendation: "2 doses, 2-6 months apart"},
        {ItemID: "pneumococcal", Name: "Pneumococcal Vaccine", StartAge: 65, EndAge: 120,
         Sex: "any", Source: "ACIP"},
    },
}

func intPtr(i int) *int { return &i }
```

### 5.4 Protocol Registry (`pkg/protocols/registry.go`)

```go
package protocols

import (
    "fmt"
    "github.com/cardiofit/kb-3-guidelines/pkg/models"
)

type Registry struct {
    acute      map[string]models.Protocol
    chronic    map[string]models.ChronicSchedule
    preventive map[string]models.PreventiveSchedule
}

func NewRegistry() *Registry {
    r := &Registry{
        acute:      make(map[string]models.Protocol),
        chronic:    make(map[string]models.ChronicSchedule),
        preventive: make(map[string]models.PreventiveSchedule),
    }
    r.loadDefaults()
    return r
}

func (r *Registry) loadDefaults() {
    // Acute protocols
    r.acute["SEPSIS-SEP1-2021"] = SepsisProtocol
    r.acute["STROKE-AHA-2019"] = StrokeProtocol
    r.acute["STEMI-ACC-2013"] = STEMIProtocol
    r.acute["DKA-ADA-2024"] = DKAProtocol
    r.acute["TRAUMA-ATLS-10"] = TraumaProtocol
    r.acute["PE-ESC-2019"] = PEProtocol

    // Chronic schedules
    r.chronic["DIABETES-ADA-2024"] = DiabetesSchedule
    r.chronic["HF-ACCAHA-2022"] = HeartFailureSchedule
    r.chronic["CKD-KDIGO-2024"] = CKDSchedule
    r.chronic["ANTICOAG-CHEST"] = AnticoagSchedule
    r.chronic["COPD-GOLD-2024"] = COPDSchedule
    r.chronic["HTN-ACCAHA-2017"] = HTNSchedule

    // Preventive schedules
    r.preventive["PRENATAL"] = PrenatalSchedule
    r.preventive["WELLCHILD"] = WellChildSchedule
    r.preventive["ADULT-PREV"] = AdultPreventiveSchedule
    r.preventive["CANCER-SCREENING"] = CancerScreeningSchedule
    r.preventive["IMMUNIZATIONS"] = ImmunizationSchedule
}

func (r *Registry) GetProtocol(id string) (models.Protocol, error) {
    if p, ok := r.acute[id]; ok {
        return p, nil
    }
    return models.Protocol{}, fmt.Errorf("protocol not found: %s", id)
}

func (r *Registry) ListAcuteProtocols() []models.Protocol {
    result := make([]models.Protocol, 0, len(r.acute))
    for _, p := range r.acute {
        result = append(result, p)
    }
    return result
}

func (r *Registry) ListChronicSchedules() []models.ChronicSchedule {
    result := make([]models.ChronicSchedule, 0, len(r.chronic))
    for _, s := range r.chronic {
        result = append(result, s)
    }
    return result
}

func (r *Registry) ListPreventiveSchedules() []models.PreventiveSchedule {
    result := make([]models.PreventiveSchedule, 0, len(r.preventive))
    for _, s := range r.preventive {
        result = append(result, s)
    }
    return result
}
```

---

## Phase 6: REST API (Days 8-9)

### 6.1 Route Definitions (`pkg/api/routes.go`)

Per README_kb3.md specification:

```go
package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, h *Handlers) {
    // Health & Status
    r.GET("/health", h.Health)
    r.GET("/metrics", h.Metrics)
    r.GET("/version", h.Version)

    // Protocol Management
    r.GET("/protocols", h.ListProtocols)
    r.GET("/protocols/acute", h.ListAcuteProtocols)
    r.GET("/protocols/chronic", h.ListChronicSchedules)
    r.GET("/protocols/preventive", h.ListPreventiveSchedules)
    r.GET("/protocols/:type/:id", h.GetProtocol)

    // Pathway Operations
    r.POST("/pathways/start", h.StartPathway)
    r.GET("/pathways/:id", h.GetPathwayStatus)
    r.GET("/pathways/:id/pending", h.GetPendingActions)
    r.GET("/pathways/:id/overdue", h.GetOverdueActions)
    r.GET("/pathways/:id/constraints", h.EvaluateConstraints)
    r.GET("/pathways/:id/audit", h.GetPathwayAudit)
    r.POST("/pathways/:id/advance", h.AdvanceStage)
    r.POST("/pathways/:id/complete-action", h.CompleteAction)

    // Patient Operations
    r.GET("/patients/:id/pathways", h.GetPatientPathways)
    r.GET("/patients/:id/schedule", h.GetPatientSchedule)
    r.GET("/patients/:id/schedule-summary", h.GetScheduleSummary)
    r.GET("/patients/:id/overdue", h.GetPatientOverdue)
    r.GET("/patients/:id/upcoming", h.GetPatientUpcoming)
    r.GET("/patients/:id/export", h.ExportPatientData)
    r.POST("/patients/:id/start-protocol", h.StartProtocolForPatient)

    // Scheduling Operations
    r.GET("/schedule/:patientId", h.GetSchedule)
    r.GET("/schedule/:patientId/pending", h.GetSchedulePending)
    r.POST("/schedule/:patientId/add", h.AddScheduledItem)
    r.POST("/schedule/:patientId/complete", h.CompleteScheduledItem)

    // Temporal Operations
    r.POST("/temporal/evaluate", h.EvaluateTemporalRelation)
    r.POST("/temporal/next-occurrence", h.CalculateNextOccurrence)
    r.POST("/temporal/validate-constraint", h.ValidateConstraintTiming)

    // Alert Management
    r.POST("/alerts/process", h.ProcessAlerts)
    r.GET("/alerts/overdue", h.GetAllOverdue)

    // Batch Operations
    r.POST("/batch/start-protocols", h.BatchStartProtocols)

    // Governance (converted from TypeScript)
    r.GET("/guidelines", h.GetGuidelines)
    r.GET("/guidelines/:id", h.GetGuideline)
    r.POST("/conflicts/resolve", h.ResolveConflict)
    r.GET("/safety-overrides", h.GetSafetyOverrides)
    r.POST("/safety-overrides", h.CreateSafetyOverride)
    r.POST("/versions", h.CreateVersion)
    r.POST("/versions/:id/approve", h.ProcessApproval)
}
```

### 6.2 Server Entry Point (`cmd/server/main.go`)

```go
package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/cardiofit/kb-3-guidelines/pkg/api"
    "github.com/cardiofit/kb-3-guidelines/pkg/audit"
    "github.com/cardiofit/kb-3-guidelines/pkg/cache"
    "github.com/cardiofit/kb-3-guidelines/pkg/config"
    "github.com/cardiofit/kb-3-guidelines/pkg/database"
    "github.com/cardiofit/kb-3-guidelines/pkg/governance"
    "github.com/cardiofit/kb-3-guidelines/pkg/graph"
    "github.com/cardiofit/kb-3-guidelines/pkg/protocols"
    "github.com/cardiofit/kb-3-guidelines/pkg/service"
    "github.com/cardiofit/kb-3-guidelines/pkg/temporal"
)

func main() {
    // Load configuration
    cfg := config.Load()

    // Initialize services
    db, err := database.NewService(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    neo4j, err := graph.NewNeo4jService(cfg.Neo4jURL)
    if err != nil {
        log.Fatalf("Failed to connect to Neo4j: %v", err)
    }
    defer neo4j.Close()

    cacheService := cache.NewService(cfg.RedisURL)
    auditLogger := audit.NewLogger(db)

    // Initialize engines
    conflictResolver := governance.NewConflictResolver(db, neo4j, auditLogger)
    safetyEngine := governance.NewSafetyOverrideEngine(db, auditLogger)
    versionManager := governance.NewVersionManager(db, neo4j, cacheService, auditLogger)
    protocolRegistry := protocols.NewRegistry()
    pathwayEngine := temporal.NewPathwayEngine(db, neo4j, protocolRegistry, auditLogger)
    schedulingEngine := temporal.NewSchedulingEngine(db, auditLogger)

    // Initialize main service
    guidelineService := service.NewGuidelineService(
        db, neo4j, cacheService, conflictResolver, safetyEngine, versionManager, auditLogger,
    )

    // Initialize HTTP handlers
    handlers := api.NewHandlers(
        guidelineService, pathwayEngine, schedulingEngine, protocolRegistry,
        conflictResolver, safetyEngine, versionManager,
    )

    // Setup Gin router
    r := gin.Default()

    // Middleware
    r.Use(api.LoggingMiddleware())
    r.Use(api.CORSMiddleware())

    // Register routes
    api.RegisterRoutes(r, handlers)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8083"
    }

    log.Printf("KB-3 Temporal Service starting on port %s", port)
    if err := r.Run(":" + port); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

---

## Phase 7: Database Schema Enhancement (Day 9)

### New Tables (`migrations/002_temporal_tables.sql`)

```sql
-- Protocol definitions
CREATE TABLE IF NOT EXISTS protocols (
    protocol_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    protocol_type VARCHAR(20) NOT NULL CHECK (protocol_type IN ('acute', 'chronic', 'preventive')),
    guideline_source VARCHAR(255),
    version VARCHAR(20),
    definition JSONB NOT NULL,
    active BOOLEAN DEFAULT true,
    effective_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Pathway instances
CREATE TABLE IF NOT EXISTS pathway_instances (
    instance_id VARCHAR(64) PRIMARY KEY,
    pathway_id VARCHAR(64) NOT NULL,
    patient_id VARCHAR(64) NOT NULL,
    current_stage VARCHAR(64),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'suspended', 'cancelled')),
    context JSONB,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Pathway actions
CREATE TABLE IF NOT EXISTS pathway_actions (
    action_id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES pathway_instances(instance_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    action_type VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'met', 'approaching', 'overdue', 'missed', 'not_applicable')),
    deadline TIMESTAMP NOT NULL,
    grace_period INTERVAL,
    alert_threshold INTERVAL,
    completed_at TIMESTAMP,
    completed_by VARCHAR(64),
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Scheduled items
CREATE TABLE IF NOT EXISTS scheduled_items (
    item_id VARCHAR(64) PRIMARY KEY,
    patient_id VARCHAR(64) NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    due_date TIMESTAMP NOT NULL,
    priority INT DEFAULT 2 CHECK (priority BETWEEN 1 AND 5),
    is_recurring BOOLEAN DEFAULT false,
    recurrence JSONB,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'overdue', 'cancelled', 'skipped')),
    completed_at TIMESTAMP,
    source_protocol VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_protocols_type ON protocols(protocol_type);
CREATE INDEX IF NOT EXISTS idx_protocols_active ON protocols(active);
CREATE INDEX IF NOT EXISTS idx_pathway_instances_patient ON pathway_instances(patient_id);
CREATE INDEX IF NOT EXISTS idx_pathway_instances_status ON pathway_instances(status);
CREATE INDEX IF NOT EXISTS idx_pathway_instances_pathway ON pathway_instances(pathway_id);
CREATE INDEX IF NOT EXISTS idx_pathway_actions_instance ON pathway_actions(instance_id);
CREATE INDEX IF NOT EXISTS idx_pathway_actions_status ON pathway_actions(status);
CREATE INDEX IF NOT EXISTS idx_pathway_actions_deadline ON pathway_actions(deadline);
CREATE INDEX IF NOT EXISTS idx_scheduled_items_patient ON scheduled_items(patient_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_items_due ON scheduled_items(due_date);
CREATE INDEX IF NOT EXISTS idx_scheduled_items_status ON scheduled_items(status);
CREATE INDEX IF NOT EXISTS idx_scheduled_items_type ON scheduled_items(item_type);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_protocols_updated_at BEFORE UPDATE ON protocols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pathway_instances_updated_at BEFORE UPDATE ON pathway_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_items_updated_at BEFORE UPDATE ON scheduled_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Phase 8: Testing (Days 9-10)

### Test Structure
```
test/
├── temporal_test.go        # Temporal operators
├── pathway_test.go         # Pathway engine
├── scheduling_test.go      # Scheduling engine
├── protocols_test.go       # Protocol definitions
├── conflict_test.go        # Conflict resolver
├── safety_test.go          # Safety override
├── version_test.go         # Version manager
├── api_test.go             # API endpoints
└── integration_test.go     # End-to-end
```

### Key Test Cases

```go
// temporal_test.go
func TestInterval_Before(t *testing.T)
func TestInterval_After(t *testing.T)
func TestInterval_Meets(t *testing.T)
func TestInterval_Overlaps(t *testing.T)
func TestInterval_During(t *testing.T)
func TestInterval_Contains(t *testing.T)
func TestInterval_Within(t *testing.T)
func TestEvaluateTemporalRelation(t *testing.T)

// pathway_test.go
func TestStartPathway_Sepsis(t *testing.T)
func TestEvaluateConstraints_Pending(t *testing.T)
func TestEvaluateConstraints_Approaching(t *testing.T)
func TestEvaluateConstraints_Overdue(t *testing.T)
func TestEvaluateConstraints_Missed(t *testing.T)
func TestCompleteAction(t *testing.T)
func TestAdvanceStage(t *testing.T)

// scheduling_test.go
func TestCalculateNextOccurrence_Daily(t *testing.T)
func TestCalculateNextOccurrence_Weekly(t *testing.T)
func TestCalculateNextOccurrence_Monthly(t *testing.T)
func TestCalculateNextOccurrence_Quarterly(t *testing.T)
func TestGetOverdueItems(t *testing.T)
func TestCompleteRecurringItem(t *testing.T)

// api_test.go
func TestHealth(t *testing.T)
func TestListProtocols(t *testing.T)
func TestStartPathwayEndpoint(t *testing.T)
func TestGetPatientSchedule(t *testing.T)
func TestEvaluateTemporalRelation(t *testing.T)
```

---

## Phase 9: Docker & Deployment (Day 10)

### Dockerfile
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb3-temporal-service ./cmd/server

# Final image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/kb3-temporal-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8083

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

CMD ["./kb3-temporal-service"]
```

### docker-compose.yml
```yaml
version: '3.8'

services:
  kb3-guidelines:
    build: .
    container_name: kb3-guidelines
    ports:
      - "8083:8083"
    environment:
      - PORT=8083
      - DATABASE_URL=postgres://postgres:password@postgres:5432/kb3?sslmode=disable
      - NEO4J_URL=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=password
      - REDIS_URL=redis://redis:6379
      - LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
      neo4j:
        condition: service_healthy
      redis:
        condition: service_started
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8083/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    image: postgres:15-alpine
    container_name: kb3-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: kb3
    ports:
      - "5433:5432"
    volumes:
      - kb3_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  neo4j:
    image: neo4j:5-community
    container_name: kb3-neo4j
    environment:
      NEO4J_AUTH: neo4j/password
    ports:
      - "7475:7474"
      - "7688:7687"
    volumes:
      - kb3_neo4j_data:/data
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:7474"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: kb3-redis
    ports:
      - "6380:6379"
    volumes:
      - kb3_redis_data:/data

volumes:
  kb3_postgres_data:
  kb3_neo4j_data:
  kb3_redis_data:
```

### Makefile
```makefile
.PHONY: build run test docker clean

build:
	go build -o bin/kb3-temporal-service ./cmd/server

run: build
	./bin/kb3-temporal-service

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

docker-build:
	docker build -t kb3-temporal-service .

docker-run:
	docker-compose up -d

docker-stop:
	docker-compose down

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

migrate:
	go run ./cmd/migrate

lint:
	golangci-lint run

help:
	@echo "Available targets:"
	@echo "  build         - Build the service"
	@echo "  run           - Build and run locally"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with docker-compose"
	@echo "  docker-stop   - Stop docker-compose"
	@echo "  clean         - Clean build artifacts"
	@echo "  migrate       - Run database migrations"
	@echo "  lint          - Run linter"
```

---

## Summary

### Timeline Overview

| Phase | Description | Duration | Dependencies |
|-------|-------------|----------|--------------|
| 0 | Archive TypeScript | 2 hours | None |
| 1 | Go Project Structure | 4 hours | Phase 0 |
| 2 | Core Models | 8 hours | Phase 1 |
| 3 | Convert Services | 4 days | Phase 2 |
| 4 | Temporal Operators | 1.5 days | Phase 2 |
| 5 | Protocol Library | 2 days | Phase 4 |
| 6 | REST API | 1.5 days | Phases 3, 4, 5 |
| 7 | Database Schema | 4 hours | Phase 1 |
| 8 | Testing | 1.5 days | Phases 3-6 |
| 9 | Docker & Deployment | 4 hours | Phase 6 |
| **Total** | | **~13 days** | |

### Success Criteria

- [ ] All TypeScript functionality converted to Go
- [ ] All 35+ REST endpoints per README_kb3.md implemented
- [ ] 13 temporal operators (Allen's Algebra) working
- [ ] Pathway state machine functional with constraint evaluation
- [ ] 6 acute protocols defined (Sepsis, Stroke, STEMI, DKA, Trauma, PE)
- [ ] 6 chronic schedules defined (Diabetes, HF, CKD, Anticoag, COPD, HTN)
- [ ] 5 preventive schedules defined (Prenatal, WellChild, Adult, Cancer, Immunizations)
- [ ] Tests passing with >80% coverage
- [ ] Docker build successful
- [ ] Performance: <10ms pathway start, <5ms constraint eval, <50ms P95

### Performance Targets (from README)

| Metric | Target |
|--------|--------|
| Pathway Start | < 10ms |
| Constraint Evaluation | < 5ms |
| Schedule Query | < 5ms |
| P95 Latency | < 50ms |

---

## Next Steps

1. **Review this plan** and confirm approach
2. **Phase 0**: Archive existing TypeScript to `_reference_typescript/`
3. **Phase 1**: Initialize Go module and directory structure
4. Begin conversion and implementation following phases 2-9
