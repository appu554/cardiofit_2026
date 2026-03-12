# Phase 1: Clinical Safety Foundation - Implementation Specification
## KB-7 Terminology Service Enhancement

### 📋 Phase Overview
- **Timeline**: 4-6 weeks
- **Priority**: 🚨 Critical - Patient Safety Impact
- **Objective**: Establish clinical governance workflows to prevent unsafe terminology changes
- **Prerequisites**: Existing KB-7 Go service operational
- **Deliverables**: GitOps workflow, audit system, policy engine

---

## 🎯 Phase 1 Objectives

### Primary Goals
1. **Clinical Governance Workflow**: Every terminology change requires clinical review
2. **Complete Audit Trail**: W3C PROV-O compliant change tracking with provenance
3. **Policy Enforcement**: Clinical safety constraints through configurable policy flags
4. **Data Integrity**: SHA256 checksum validation for all terminology updates

### Success Criteria
- **100% clinical review compliance** for terminology changes
- **Complete audit trail** with <1 second provenance record creation
- **Policy flag effectiveness** preventing 100% of unsafe operations
- **Zero security incidents** related to unauthorized terminology changes

---

## 🗓️ Detailed Implementation Timeline

### Week 1: GitOps Clinical Governance Workflow

#### Days 1-2: GitHub Workflow Setup
**Deliverables**:
- `.github/workflows/terminology-review.yml`
- PR template for clinical reviews
- Branch protection rules

**Tasks**:
```yaml
# .github/workflows/terminology-review.yml
name: Clinical Terminology Review

on:
  pull_request:
    paths:
      - 'kb-7-terminology/data/**'
      - 'kb-7-terminology/mappings/**'
      - 'kb-7-terminology/internal/loaders/**'
    types: [opened, synchronize, reopened]

jobs:
  clinical-impact-assessment:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Clinical Impact Analysis
        id: impact-analysis
        run: |
          echo "Running clinical impact assessment..."
          # Analyze changed files for clinical significance
          changed_files=$(git diff --name-only ${{ github.event.pull_request.base.sha }} ${{ github.event.pull_request.head.sha }})
          echo "changed_files=$changed_files" >> $GITHUB_OUTPUT

          # Determine clinical impact level
          if echo "$changed_files" | grep -E "(drug|medication|interaction)" > /dev/null; then
            echo "clinical_impact=high" >> $GITHUB_OUTPUT
          elif echo "$changed_files" | grep -E "(mapping|concept)" > /dev/null; then
            echo "clinical_impact=medium" >> $GITHUB_OUTPUT
          else
            echo "clinical_impact=low" >> $GITHUB_OUTPUT
          fi

      - name: Require Clinical Review
        if: steps.impact-analysis.outputs.clinical_impact != 'low'
        uses: actions/github-script@v7
        with:
          script: |
            const { data: reviews } = await github.rest.pulls.listRequestedReviewers({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.issue.number,
            });

            const clinicalReviewers = ['clinical-informatics-lead', 'senior-ontologist'];
            const hasReviewers = reviews.users.some(user =>
              clinicalReviewers.includes(user.login)
            );

            if (!hasReviewers) {
              await github.rest.pulls.requestReviewers({
                owner: context.repo.owner,
                repo: context.repo.repo,
                pull_number: context.issue.number,
                reviewers: clinicalReviewers
              });
            }

  terminology-validation:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Validate Terminology Changes
        run: |
          cd kb-7-terminology
          go run ./cmd/validate-terminology-changes \
            --base-ref=${{ github.event.pull_request.base.sha }} \
            --head-ref=${{ github.event.pull_request.head.sha }} \
            --output-format=github-actions

      - name: Clinical Safety Checks
        run: |
          cd kb-7-terminology
          go run ./cmd/clinical-safety-validator \
            --changed-files="$(git diff --name-only ${{ github.event.pull_request.base.sha }} ${{ github.event.pull_request.head.sha }})" \
            --safety-level=strict
```

**Branch Protection Configuration**:
```yaml
# Configure via GitHub API or UI
branch_protection:
  pattern: "main"
  required_status_checks:
    strict: true
    contexts:
      - "clinical-impact-assessment"
      - "terminology-validation"
  enforce_admins: true
  required_pull_request_reviews:
    required_approving_review_count: 2
    dismiss_stale_reviews: true
    require_code_owner_reviews: true
    required_review_from_code_owners: true
  restrictions:
    users: []
    teams: ["clinical-informatics-team"]
```

#### Days 3-4: PR Template and Clinical Review Process
**Deliverable**: `.github/pull_request_template.md`

```markdown
## Clinical Terminology Change Request

### 📋 Change Summary
- **Terminology System**: [ ] SNOMED CT [ ] RxNorm [ ] LOINC [ ] ICD-10 [ ] AMT [ ] Other: ____
- **Change Type**: [ ] New Concepts [ ] Modified Mappings [ ] Deprecated Terms [ ] Policy Updates
- **Clinical Impact Level**: [ ] Low [ ] Medium [ ] High [ ] Critical

### 🏥 Clinical Justification
**Medical Rationale** (required for medium/high impact):
<!-- Explain the clinical need for this terminology change -->

**Patient Safety Assessment**:
<!-- Describe potential patient safety implications -->

**Drug Interaction Impact** (if applicable):
<!-- Assess impact on drug-drug interaction detection -->

### 📊 Technical Details
**Files Modified**:
- [ ] Core terminology data
- [ ] Mapping configurations
- [ ] Validation rules
- [ ] Policy flags

**Testing Performed**:
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] Manual clinical validation
- [ ] Performance impact assessed

### ✅ Clinical Review Checklist
- [ ] Clinical necessity documented
- [ ] Patient safety implications reviewed
- [ ] Drug interaction impacts assessed
- [ ] Regulatory compliance verified (if applicable)
- [ ] Rollback strategy documented
- [ ] Clinical team notification plan ready

### 📋 Reviewer Assignment
**Clinical Informatics Lead**: @clinical-informatics-lead
**Senior Ontologist**: @senior-ontologist
**Additional Clinical SME** (if high/critical impact): @clinical-sme

### 🚨 Emergency Override
If this is a critical patient safety issue requiring immediate implementation:
- [ ] Emergency override requested
- [ ] Chief Medical Officer approval: @cmo-approval
- [ ] Post-deployment clinical review scheduled within 24 hours

---
**Note**: All medium and high impact changes require clinical review approval before merge.
```

#### Days 5-7: Clinical Team Setup and Training
**Tasks**:
1. Create `CODEOWNERS` file with clinical review assignments
2. Set up clinical informatics team in GitHub
3. Create clinical reviewer onboarding documentation
4. Configure automated reviewer assignment rules

**CODEOWNERS Configuration**:
```
# Clinical Terminology Code Owners
/kb-7-terminology/data/                    @clinical-informatics-team
/kb-7-terminology/internal/loaders/        @clinical-informatics-team @senior-developer
/kb-7-terminology/mappings/                @clinical-informatics-team
/kb-7-terminology/policies/                @clinical-informatics-team @clinical-sme
/kb-7-terminology/docs/clinical/           @clinical-informatics-team

# Critical safety areas require additional review
/kb-7-terminology/data/drug-interactions/  @clinical-informatics-team @clinical-sme @pharmacist-reviewer
/kb-7-terminology/policies/safety-flags/   @clinical-informatics-team @clinical-sme @cmo-approval
```

### Week 2: Provenance & Audit System Database Design

#### Days 8-9: Database Schema Design
**Deliverable**: Enhanced PostgreSQL schema with audit tables

```sql
-- Migration: 20250919_001_add_audit_tables.up.sql

-- Core audit trail table
CREATE TABLE terminology_changes (
    change_id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(100) NOT NULL,
    record_id BIGINT NOT NULL,
    operation VARCHAR(20) NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    change_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id VARCHAR(100) NOT NULL,
    session_id UUID NOT NULL,
    request_id VARCHAR(100), -- For tracing across services
    client_ip INET,
    user_agent TEXT,
    clinical_justification TEXT,
    approval_status VARCHAR(20) DEFAULT 'pending' CHECK (approval_status IN ('pending', 'approved', 'rejected', 'emergency_override')),
    approved_by VARCHAR(100),
    approved_at TIMESTAMP WITH TIME ZONE,
    clinical_impact_level VARCHAR(20) DEFAULT 'medium' CHECK (clinical_impact_level IN ('low', 'medium', 'high', 'critical')),
    rollback_change_id BIGINT REFERENCES terminology_changes(change_id), -- For rollback tracking

    -- Indexes for performance
    CONSTRAINT unique_session_operation UNIQUE (session_id, table_name, record_id, operation)
);

-- Add indexes for common queries
CREATE INDEX idx_terminology_changes_timestamp ON terminology_changes(change_timestamp);
CREATE INDEX idx_terminology_changes_user ON terminology_changes(user_id);
CREATE INDEX idx_terminology_changes_table ON terminology_changes(table_name);
CREATE INDEX idx_terminology_changes_approval ON terminology_changes(approval_status);
CREATE INDEX idx_terminology_changes_impact ON terminology_changes(clinical_impact_level);

-- W3C PROV-O compliant provenance tracking
CREATE TABLE change_provenance (
    provenance_id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES terminology_changes(change_id),

    -- PROV-O Entity (what was affected)
    prov_entity JSONB NOT NULL, -- {"id": "concept_123", "type": "snomed_concept", "attributes": {...}}

    -- PROV-O Activity (what was done)
    prov_activity JSONB NOT NULL, -- {"id": "update_activity_456", "type": "terminology_update", "startTime": "...", "endTime": "..."}

    -- PROV-O Agent (who/what did it)
    prov_agent JSONB NOT NULL, -- {"id": "user_789", "type": "Person", "name": "Dr. Smith", "role": "Clinical Informaticist"}

    -- Source data integrity
    source_checksum VARCHAR(64), -- SHA256 of source terminology file
    source_file_path TEXT,
    source_version VARCHAR(50),

    -- Additional PROV-O relationships
    was_generated_by VARCHAR(100), -- Activity that generated this entity
    was_derived_from VARCHAR(100), -- Previous version or source
    was_attributed_to VARCHAR(100), -- Responsible agent

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for provenance queries
CREATE INDEX idx_change_provenance_change_id ON change_provenance(change_id);
CREATE INDEX idx_change_provenance_entity ON change_provenance USING GIN (prov_entity);
CREATE INDEX idx_change_provenance_activity ON change_provenance USING GIN (prov_activity);
CREATE INDEX idx_change_provenance_agent ON change_provenance USING GIN (prov_agent);

-- Source file tracking and integrity
CREATE TABLE terminology_sources (
    source_id BIGSERIAL PRIMARY KEY,
    source_name VARCHAR(100) NOT NULL, -- SNOMED CT, RxNorm, LOINC, etc.
    version VARCHAR(50) NOT NULL,
    release_date DATE,
    download_url TEXT,
    download_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    sha256_checksum VARCHAR(64) NOT NULL,
    md5_checksum VARCHAR(32), -- Additional verification
    validation_status VARCHAR(20) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'valid', 'invalid', 'warning')),
    validation_errors JSONB, -- Detailed validation results

    -- Complete manifest data for reproducibility
    manifest_data JSONB NOT NULL, -- Complete sources.json entry

    -- Provenance information
    downloaded_by VARCHAR(100) NOT NULL,
    download_session_id UUID NOT NULL,

    -- Constraints
    UNIQUE(source_name, version, sha256_checksum)
);

-- Index for source tracking
CREATE INDEX idx_terminology_sources_name_version ON terminology_sources(source_name, version);
CREATE INDEX idx_terminology_sources_download_time ON terminology_sources(download_timestamp);
CREATE INDEX idx_terminology_sources_checksum ON terminology_sources(sha256_checksum);

-- Audit trigger function for automatic change tracking
CREATE OR REPLACE FUNCTION terminology_audit_trigger()
RETURNS TRIGGER AS $$
DECLARE
    change_id BIGINT;
    old_values JSONB;
    new_values JSONB;
BEGIN
    -- Convert row data to JSONB
    IF TG_OP = 'DELETE' THEN
        old_values = to_jsonb(OLD);
        new_values = NULL;
    ELSIF TG_OP = 'UPDATE' THEN
        old_values = to_jsonb(OLD);
        new_values = to_jsonb(NEW);
    ELSIF TG_OP = 'INSERT' THEN
        old_values = NULL;
        new_values = to_jsonb(NEW);
    END IF;

    -- Insert audit record
    INSERT INTO terminology_changes (
        table_name,
        record_id,
        operation,
        old_values,
        new_values,
        user_id,
        session_id,
        request_id
    ) VALUES (
        TG_TABLE_NAME,
        COALESCE(NEW.id, OLD.id), -- Use NEW.id for INSERT/UPDATE, OLD.id for DELETE
        TG_OP,
        old_values,
        new_values,
        current_setting('app.current_user', true),
        current_setting('app.session_id', true)::UUID,
        current_setting('app.request_id', true)
    ) RETURNING change_id INTO change_id;

    -- Create PROV-O provenance record
    INSERT INTO change_provenance (
        change_id,
        prov_entity,
        prov_activity,
        prov_agent
    ) VALUES (
        change_id,
        jsonb_build_object(
            'id', TG_TABLE_NAME || '_' || COALESCE(NEW.id, OLD.id),
            'type', TG_TABLE_NAME,
            'attributes', COALESCE(new_values, old_values)
        ),
        jsonb_build_object(
            'id', 'activity_' || change_id,
            'type', 'terminology_' || lower(TG_OP),
            'startTime', NOW()::text,
            'endTime', NOW()::text
        ),
        jsonb_build_object(
            'id', current_setting('app.current_user', true),
            'type', 'Person',
            'role', current_setting('app.user_role', true)
        )
    );

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers to terminology tables
CREATE TRIGGER terminology_concepts_audit
    AFTER INSERT OR UPDATE OR DELETE ON terminology_concepts
    FOR EACH ROW EXECUTE FUNCTION terminology_audit_trigger();

CREATE TRIGGER concept_mappings_audit
    AFTER INSERT OR UPDATE OR DELETE ON concept_mappings
    FOR EACH ROW EXECUTE FUNCTION terminology_audit_trigger();

CREATE TRIGGER value_sets_audit
    AFTER INSERT OR UPDATE OR DELETE ON value_sets
    FOR EACH ROW EXECUTE FUNCTION terminology_audit_trigger();

-- Helper function for setting session context
CREATE OR REPLACE FUNCTION set_audit_context(
    user_id TEXT,
    session_id UUID,
    request_id TEXT DEFAULT NULL,
    user_role TEXT DEFAULT 'user'
)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_user', user_id, false);
    PERFORM set_config('app.session_id', session_id::text, false);
    PERFORM set_config('app.request_id', COALESCE(request_id, gen_random_uuid()::text), false);
    PERFORM set_config('app.user_role', user_role, false);
END;
$$ LANGUAGE plpgsql;

-- View for easy audit trail querying
CREATE VIEW terminology_audit_trail AS
SELECT
    tc.change_id,
    tc.table_name,
    tc.record_id,
    tc.operation,
    tc.change_timestamp,
    tc.user_id,
    tc.clinical_justification,
    tc.approval_status,
    tc.approved_by,
    tc.approved_at,
    tc.clinical_impact_level,
    cp.prov_entity->>'id' as entity_id,
    cp.prov_entity->>'type' as entity_type,
    cp.source_checksum,
    cp.source_version
FROM terminology_changes tc
LEFT JOIN change_provenance cp ON tc.change_id = cp.change_id
ORDER BY tc.change_timestamp DESC;
```

#### Days 10-11: Go Service Audit Integration
**Deliverable**: Go audit service implementation

```go
// internal/audit/service.go
package audit

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

type AuditService struct {
    db     *sql.DB
    logger *logrus.Logger
}

type ChangeContext struct {
    UserID               string    `json:"user_id"`
    SessionID            uuid.UUID `json:"session_id"`
    RequestID            string    `json:"request_id,omitempty"`
    ClientIP             string    `json:"client_ip,omitempty"`
    UserAgent            string    `json:"user_agent,omitempty"`
    UserRole             string    `json:"user_role"`
    ClinicalJustification string   `json:"clinical_justification,omitempty"`
}

type ChangeRecord struct {
    TableName             string                 `json:"table_name"`
    RecordID              int64                  `json:"record_id"`
    Operation             string                 `json:"operation"`
    OldValues             map[string]interface{} `json:"old_values,omitempty"`
    NewValues             map[string]interface{} `json:"new_values,omitempty"`
    ClinicalImpactLevel   string                 `json:"clinical_impact_level"`
    ApprovalStatus        string                 `json:"approval_status"`
}

type ProvenanceRecord struct {
    ChangeID       int64                  `json:"change_id"`
    Entity         map[string]interface{} `json:"entity"`
    Activity       map[string]interface{} `json:"activity"`
    Agent          map[string]interface{} `json:"agent"`
    SourceChecksum string                 `json:"source_checksum,omitempty"`
    SourceVersion  string                 `json:"source_version,omitempty"`
}

func NewAuditService(db *sql.DB, logger *logrus.Logger) *AuditService {
    return &AuditService{
        db:     db,
        logger: logger,
    }
}

func (as *AuditService) SetContext(ctx context.Context, changeCtx *ChangeContext) error {
    tx, err := as.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Set PostgreSQL session variables for audit triggers
    _, err = tx.ExecContext(ctx, "SELECT set_audit_context($1, $2, $3, $4)",
        changeCtx.UserID,
        changeCtx.SessionID,
        changeCtx.RequestID,
        changeCtx.UserRole,
    )
    if err != nil {
        return fmt.Errorf("failed to set audit context: %w", err)
    }

    return tx.Commit()
}

func (as *AuditService) TrackChange(ctx context.Context, record *ChangeRecord) (int64, error) {
    // Check if change requires clinical approval
    requiresApproval := as.requiresClinicalApproval(record)
    if requiresApproval && record.ApprovalStatus == "" {
        record.ApprovalStatus = "pending"
    }

    oldValuesJSON, _ := json.Marshal(record.OldValues)
    newValuesJSON, _ := json.Marshal(record.NewValues)

    var changeID int64
    err := as.db.QueryRowContext(ctx, `
        INSERT INTO terminology_changes (
            table_name, record_id, operation, old_values, new_values,
            clinical_impact_level, approval_status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING change_id`,
        record.TableName,
        record.RecordID,
        record.Operation,
        string(oldValuesJSON),
        string(newValuesJSON),
        record.ClinicalImpactLevel,
        record.ApprovalStatus,
    ).Scan(&changeID)

    if err != nil {
        return 0, fmt.Errorf("failed to insert change record: %w", err)
    }

    as.logger.WithFields(logrus.Fields{
        "change_id":       changeID,
        "table_name":      record.TableName,
        "operation":       record.Operation,
        "approval_status": record.ApprovalStatus,
    }).Info("Change tracked in audit system")

    return changeID, nil
}

func (as *AuditService) CreateProvenanceRecord(ctx context.Context, prov *ProvenanceRecord) error {
    entityJSON, _ := json.Marshal(prov.Entity)
    activityJSON, _ := json.Marshal(prov.Activity)
    agentJSON, _ := json.Marshal(prov.Agent)

    _, err := as.db.ExecContext(ctx, `
        INSERT INTO change_provenance (
            change_id, prov_entity, prov_activity, prov_agent,
            source_checksum, source_version
        ) VALUES ($1, $2, $3, $4, $5, $6)`,
        prov.ChangeID,
        string(entityJSON),
        string(activityJSON),
        string(agentJSON),
        prov.SourceChecksum,
        prov.SourceVersion,
    )

    if err != nil {
        return fmt.Errorf("failed to insert provenance record: %w", err)
    }

    return nil
}

func (as *AuditService) requiresClinicalApproval(record *ChangeRecord) bool {
    // Determine if change requires clinical approval based on:
    // 1. Clinical impact level
    // 2. Table being modified
    // 3. Type of operation

    if record.ClinicalImpactLevel == "high" || record.ClinicalImpactLevel == "critical" {
        return true
    }

    // Drug-related tables always require approval
    drugTables := []string{"drug_interactions", "medication_mappings", "contraindications"}
    for _, table := range drugTables {
        if record.TableName == table {
            return true
        }
    }

    return false
}

func (as *AuditService) ApproveChange(ctx context.Context, changeID int64, approverID string) error {
    _, err := as.db.ExecContext(ctx, `
        UPDATE terminology_changes
        SET approval_status = 'approved',
            approved_by = $1,
            approved_at = NOW()
        WHERE change_id = $2 AND approval_status = 'pending'`,
        approverID, changeID,
    )

    if err != nil {
        return fmt.Errorf("failed to approve change: %w", err)
    }

    as.logger.WithFields(logrus.Fields{
        "change_id":   changeID,
        "approved_by": approverID,
    }).Info("Change approved")

    return nil
}

func (as *AuditService) GetAuditTrail(ctx context.Context, filters *AuditFilters) ([]*AuditTrailEntry, error) {
    query := `
        SELECT change_id, table_name, record_id, operation, change_timestamp,
               user_id, clinical_justification, approval_status, approved_by,
               clinical_impact_level, entity_id, source_checksum
        FROM terminology_audit_trail
        WHERE 1=1`

    args := []interface{}{}
    argCount := 0

    if filters.StartTime != nil {
        argCount++
        query += fmt.Sprintf(" AND change_timestamp >= $%d", argCount)
        args = append(args, *filters.StartTime)
    }

    if filters.EndTime != nil {
        argCount++
        query += fmt.Sprintf(" AND change_timestamp <= $%d", argCount)
        args = append(args, *filters.EndTime)
    }

    if filters.UserID != "" {
        argCount++
        query += fmt.Sprintf(" AND user_id = $%d", argCount)
        args = append(args, filters.UserID)
    }

    if filters.TableName != "" {
        argCount++
        query += fmt.Sprintf(" AND table_name = $%d", argCount)
        args = append(args, filters.TableName)
    }

    query += " ORDER BY change_timestamp DESC"

    if filters.Limit > 0 {
        argCount++
        query += fmt.Sprintf(" LIMIT $%d", argCount)
        args = append(args, filters.Limit)
    }

    rows, err := as.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query audit trail: %w", err)
    }
    defer rows.Close()

    var entries []*AuditTrailEntry
    for rows.Next() {
        var entry AuditTrailEntry
        var approvedBy sql.NullString
        var clinicalJustification sql.NullString
        var entityID sql.NullString
        var sourceChecksum sql.NullString

        err := rows.Scan(
            &entry.ChangeID,
            &entry.TableName,
            &entry.RecordID,
            &entry.Operation,
            &entry.ChangeTimestamp,
            &entry.UserID,
            &clinicalJustification,
            &entry.ApprovalStatus,
            &approvedBy,
            &entry.ClinicalImpactLevel,
            &entityID,
            &sourceChecksum,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan audit trail entry: %w", err)
        }

        if approvedBy.Valid {
            entry.ApprovedBy = approvedBy.String
        }
        if clinicalJustification.Valid {
            entry.ClinicalJustification = clinicalJustification.String
        }
        if entityID.Valid {
            entry.EntityID = entityID.String
        }
        if sourceChecksum.Valid {
            entry.SourceChecksum = sourceChecksum.String
        }

        entries = append(entries, &entry)
    }

    return entries, rows.Err()
}

type AuditFilters struct {
    StartTime *time.Time
    EndTime   *time.Time
    UserID    string
    TableName string
    Limit     int
}

type AuditTrailEntry struct {
    ChangeID              int64     `json:"change_id"`
    TableName             string    `json:"table_name"`
    RecordID              int64     `json:"record_id"`
    Operation             string    `json:"operation"`
    ChangeTimestamp       time.Time `json:"change_timestamp"`
    UserID                string    `json:"user_id"`
    ClinicalJustification string    `json:"clinical_justification,omitempty"`
    ApprovalStatus        string    `json:"approval_status"`
    ApprovedBy            string    `json:"approved_by,omitempty"`
    ClinicalImpactLevel   string    `json:"clinical_impact_level"`
    EntityID              string    `json:"entity_id,omitempty"`
    SourceChecksum        string    `json:"source_checksum,omitempty"`
}
```

### Week 3: Clinical Policy Flags Implementation

#### Days 15-17: Policy Engine Design
**Deliverable**: Configurable policy engine with pluggable rules

```go
// internal/policy/engine.go
package policy

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

type PolicyEngine struct {
    rules    map[string]PolicyRule
    db       *sql.DB
    cache    cache.Cache
    logger   *logrus.Logger
}

type PolicyRule interface {
    Name() string
    Description() string
    Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error)
    IsEnabled() bool
    Priority() int // Higher numbers = higher priority
}

type PolicyDecision struct {
    Allowed      bool     `json:"allowed"`
    Reason       string   `json:"reason,omitempty"`
    Warnings     []string `json:"warnings,omitempty"`
    Requirements []string `json:"requirements,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type Concept struct {
    ID                string                 `json:"id"`
    Code              string                 `json:"code"`
    Display           string                 `json:"display"`
    System            string                 `json:"system"`
    PolicyFlags       map[string]interface{} `json:"policy_flags"`
    ClinicalContext   *ClinicalContext       `json:"clinical_context,omitempty"`
}

type ClinicalContext struct {
    DrugClass         string   `json:"drug_class,omitempty"`
    Interactions      []string `json:"interactions,omitempty"`
    Contraindications []string `json:"contraindications,omitempty"`
    SafetyWarnings    []string `json:"safety_warnings,omitempty"`
    RegulatoryStatus  string   `json:"regulatory_status,omitempty"`
}

func NewPolicyEngine(db *sql.DB, cache cache.Cache, logger *logrus.Logger) *PolicyEngine {
    pe := &PolicyEngine{
        rules:  make(map[string]PolicyRule),
        db:     db,
        cache:  cache,
        logger: logger,
    }

    // Register built-in rules
    pe.RegisterDefaultRules()
    return pe
}

func (pe *PolicyEngine) RegisterDefaultRules() {
    pe.RegisterRule(&DoNotAutoMapRule{})
    pe.RegisterRule(&ClinicalReviewRequiredRule{})
    pe.RegisterRule(&AustralianOnlyRule{})
    pe.RegisterRule(&DeprecationRule{})
    pe.RegisterRule(&SafetyLevelRule{})
    pe.RegisterRule(&DrugInteractionRule{})
}

func (pe *PolicyEngine) RegisterRule(rule PolicyRule) {
    pe.rules[rule.Name()] = rule
    pe.logger.WithFields(logrus.Fields{
        "rule_name":        rule.Name(),
        "rule_description": rule.Description(),
        "rule_priority":    rule.Priority(),
    }).Info("Policy rule registered")
}

func (pe *PolicyEngine) EvaluateOperation(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    // Get applicable rules sorted by priority
    applicableRules := pe.getApplicableRules(concept, operation)

    finalDecision := &PolicyDecision{
        Allowed:      true,
        Warnings:     []string{},
        Requirements: []string{},
        Metadata:     make(map[string]interface{}),
    }

    // Evaluate each rule
    for _, rule := range applicableRules {
        if !rule.IsEnabled() {
            continue
        }

        decision, err := rule.Evaluate(ctx, concept, operation)
        if err != nil {
            pe.logger.WithFields(logrus.Fields{
                "rule_name":   rule.Name(),
                "concept_id":  concept.ID,
                "operation":   operation,
                "error":       err.Error(),
            }).Error("Policy rule evaluation failed")

            // Continue with other rules, but log the error
            finalDecision.Warnings = append(finalDecision.Warnings,
                fmt.Sprintf("Rule %s evaluation failed: %s", rule.Name(), err.Error()))
            continue
        }

        // Aggregate decisions (AND logic for allowed, collect all warnings/requirements)
        if !decision.Allowed {
            finalDecision.Allowed = false
            finalDecision.Reason = decision.Reason
        }

        finalDecision.Warnings = append(finalDecision.Warnings, decision.Warnings...)
        finalDecision.Requirements = append(finalDecision.Requirements, decision.Requirements...)

        // Merge metadata
        for key, value := range decision.Metadata {
            finalDecision.Metadata[key] = value
        }
    }

    pe.logger.WithFields(logrus.Fields{
        "concept_id":        concept.ID,
        "operation":         operation,
        "final_allowed":     finalDecision.Allowed,
        "rules_evaluated":   len(applicableRules),
        "warnings_count":    len(finalDecision.Warnings),
        "requirements_count": len(finalDecision.Requirements),
    }).Debug("Policy evaluation completed")

    return finalDecision, nil
}

func (pe *PolicyEngine) getApplicableRules(concept *Concept, operation string) []PolicyRule {
    var rules []PolicyRule
    for _, rule := range pe.rules {
        rules = append(rules, rule)
    }

    // Sort by priority (higher priority first)
    sort.Slice(rules, func(i, j int) bool {
        return rules[i].Priority() > rules[j].Priority()
    })

    return rules
}

// Built-in Policy Rules

// DoNotAutoMapRule prevents automatic mapping operations
type DoNotAutoMapRule struct{}

func (r *DoNotAutoMapRule) Name() string { return "do_not_auto_map" }
func (r *DoNotAutoMapRule) Description() string {
    return "Prevents automatic mapping operations for concepts with doNotAutoMap flag"
}
func (r *DoNotAutoMapRule) Priority() int { return 100 }
func (r *DoNotAutoMapRule) IsEnabled() bool { return true }

func (r *DoNotAutoMapRule) Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    if doNotAutoMap, exists := concept.PolicyFlags["doNotAutoMap"]; exists {
        if autoMap, ok := doNotAutoMap.(bool); ok && autoMap {
            if strings.Contains(operation, "auto") || operation == "batch_update" {
                return &PolicyDecision{
                    Allowed: false,
                    Reason:  "Concept is flagged as 'doNotAutoMap' - manual review required",
                    Requirements: []string{"manual_clinical_review"},
                    Metadata: map[string]interface{}{
                        "rule_triggered": "do_not_auto_map",
                        "requires_manual_review": true,
                    },
                }, nil
            }
        }
    }

    return &PolicyDecision{Allowed: true}, nil
}

// ClinicalReviewRequiredRule enforces clinical review requirements
type ClinicalReviewRequiredRule struct{}

func (r *ClinicalReviewRequiredRule) Name() string { return "clinical_review_required" }
func (r *ClinicalReviewRequiredRule) Description() string {
    return "Enforces clinical review requirements for high-impact changes"
}
func (r *ClinicalReviewRequiredRule) Priority() int { return 90 }
func (r *ClinicalReviewRequiredRule) IsEnabled() bool { return true }

func (r *ClinicalReviewRequiredRule) Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    requiresReview := false

    // Check explicit flag
    if reviewFlag, exists := concept.PolicyFlags["requiresClinicalReview"]; exists {
        if required, ok := reviewFlag.(bool); ok && required {
            requiresReview = true
        }
    }

    // Check safety level
    if safetyLevel, exists := concept.PolicyFlags["safetyLevel"]; exists {
        if level, ok := safetyLevel.(string); ok && (level == "high" || level == "critical") {
            requiresReview = true
        }
    }

    // Check drug-related concepts
    if concept.ClinicalContext != nil && concept.ClinicalContext.DrugClass != "" {
        requiresReview = true
    }

    if requiresReview {
        return &PolicyDecision{
            Allowed: false,
            Reason:  "Clinical review required for this concept",
            Requirements: []string{
                "clinical_informaticist_approval",
                "clinical_justification_documented",
            },
            Warnings: []string{
                "This change requires clinical review before implementation",
            },
            Metadata: map[string]interface{}{
                "rule_triggered": "clinical_review_required",
                "review_type": "clinical_informaticist",
            },
        }, nil
    }

    return &PolicyDecision{Allowed: true}, nil
}

// AustralianOnlyRule restricts certain concepts to Australian deployments
type AustralianOnlyRule struct{}

func (r *AustralianOnlyRule) Name() string { return "australian_only" }
func (r *AustralianOnlyRule) Description() string {
    return "Restricts Australian-specific terminologies to appropriate deployments"
}
func (r *AustralianOnlyRule) Priority() int { return 80 }
func (r *AustralianOnlyRule) IsEnabled() bool { return true }

func (r *AustralianOnlyRule) Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    if australianOnly, exists := concept.PolicyFlags["australianOnly"]; exists {
        if restricted, ok := australianOnly.(bool); ok && restricted {
            // Check deployment region (this would come from configuration)
            deploymentRegion := "international" // This should be from config

            if deploymentRegion != "australia" {
                return &PolicyDecision{
                    Allowed: false,
                    Reason:  "Australian-only concept cannot be used in non-Australian deployments",
                    Metadata: map[string]interface{}{
                        "rule_triggered": "australian_only",
                        "deployment_region": deploymentRegion,
                        "required_region": "australia",
                    },
                }, nil
            }
        }
    }

    return &PolicyDecision{Allowed: true}, nil
}

// DeprecationRule handles deprecated concepts
type DeprecationRule struct{}

func (r *DeprecationRule) Name() string { return "deprecation" }
func (r *DeprecationRule) Description() string {
    return "Manages deprecated concepts and suggests replacements"
}
func (r *DeprecationRule) Priority() int { return 70 }
func (r *DeprecationRule) IsEnabled() bool { return true }

func (r *DeprecationRule) Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    if deprecationDate, exists := concept.PolicyFlags["deprecationDate"]; exists {
        if dateStr, ok := deprecationDate.(string); ok {
            deprecatedDate, err := time.Parse("2006-01-02", dateStr)
            if err == nil && time.Now().After(deprecatedDate) {

                warnings := []string{
                    fmt.Sprintf("Concept deprecated as of %s", dateStr),
                }

                replacementConcept, hasReplacement := concept.PolicyFlags["replacementConcept"]
                if hasReplacement {
                    warnings = append(warnings,
                        fmt.Sprintf("Replacement concept available: %v", replacementConcept))
                }

                return &PolicyDecision{
                    Allowed: true, // Allow but warn
                    Warnings: warnings,
                    Metadata: map[string]interface{}{
                        "rule_triggered": "deprecation",
                        "deprecation_date": dateStr,
                        "replacement_concept": replacementConcept,
                    },
                }, nil
            }
        }
    }

    return &PolicyDecision{Allowed: true}, nil
}

// DrugInteractionRule evaluates drug interaction implications
type DrugInteractionRule struct{}

func (r *DrugInteractionRule) Name() string { return "drug_interaction" }
func (r *DrugInteractionRule) Description() string {
    return "Evaluates drug interaction implications for medication-related concepts"
}
func (r *DrugInteractionRule) Priority() int { return 95 }
func (r *DrugInteractionRule) IsEnabled() bool { return true }

func (r *DrugInteractionRule) Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error) {
    if concept.ClinicalContext == nil || len(concept.ClinicalContext.Interactions) == 0 {
        return &PolicyDecision{Allowed: true}, nil
    }

    // Check for high-severity interactions
    hasHighSeverityInteraction := false
    for _, interaction := range concept.ClinicalContext.Interactions {
        if strings.Contains(strings.ToLower(interaction), "major") ||
           strings.Contains(strings.ToLower(interaction), "contraindicated") {
            hasHighSeverityInteraction = true
            break
        }
    }

    if hasHighSeverityInteraction {
        return &PolicyDecision{
            Allowed: false,
            Reason:  "Concept has major drug interactions - requires specialized clinical review",
            Requirements: []string{
                "pharmacist_review",
                "drug_interaction_assessment",
                "clinical_informaticist_approval",
            },
            Warnings: []string{
                "This medication has major drug interactions",
                "Specialized pharmacist review required",
            },
            Metadata: map[string]interface{}{
                "rule_triggered": "drug_interaction",
                "interaction_count": len(concept.ClinicalContext.Interactions),
                "requires_pharmacist_review": true,
            },
        }, nil
    }

    return &PolicyDecision{Allowed: true}, nil
}
```

#### Days 18-19: Policy Flag Database Integration
**Deliverable**: Database schema updates and migration for policy flags

```sql
-- Migration: 20250919_002_add_policy_flags.up.sql

-- Add policy flags columns to existing tables
ALTER TABLE terminology_concepts ADD COLUMN IF NOT EXISTS policy_flags JSONB DEFAULT '{}';
ALTER TABLE concept_mappings ADD COLUMN IF NOT EXISTS policy_flags JSONB DEFAULT '{}';
ALTER TABLE value_sets ADD COLUMN IF NOT EXISTS policy_flags JSONB DEFAULT '{}';

-- Create indexes for policy flag queries
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_policy_flags ON terminology_concepts USING GIN (policy_flags);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_policy_flags ON concept_mappings USING GIN (policy_flags);
CREATE INDEX IF NOT EXISTS idx_value_sets_policy_flags ON value_sets USING GIN (policy_flags);

-- Policy configuration table for dynamic rule management
CREATE TABLE policy_rules (
    rule_id BIGSERIAL PRIMARY KEY,
    rule_name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 50,
    configuration JSONB DEFAULT '{}',
    applies_to_systems TEXT[], -- Array of terminology systems this rule applies to
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    updated_by VARCHAR(100) NOT NULL
);

-- Insert default policy rules
INSERT INTO policy_rules (rule_name, description, priority, applies_to_systems, created_by) VALUES
    ('do_not_auto_map', 'Prevents automatic mapping operations for flagged concepts', 100, '{"*"}', 'system'),
    ('clinical_review_required', 'Enforces clinical review requirements for high-impact changes', 90, '{"*"}', 'system'),
    ('australian_only', 'Restricts Australian-specific terminologies to appropriate deployments', 80, '{"AMT", "SNOMED-CT-AU", "ICD-10-AM"}', 'system'),
    ('deprecation', 'Manages deprecated concepts and suggests replacements', 70, '{"*"}', 'system'),
    ('drug_interaction', 'Evaluates drug interaction implications for medication concepts', 95, '{"RxNorm", "AMT", "SNOMED-CT"}', 'system');

-- Policy violation log for tracking blocked operations
CREATE TABLE policy_violations (
    violation_id BIGSERIAL PRIMARY KEY,
    concept_id VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    rule_name VARCHAR(100) NOT NULL REFERENCES policy_rules(rule_name),
    violation_reason TEXT NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    session_id UUID NOT NULL,
    client_ip INET,
    violation_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    policy_decision JSONB NOT NULL, -- Complete policy decision for audit
    override_requested BOOLEAN DEFAULT false,
    override_approved BOOLEAN DEFAULT false,
    override_approved_by VARCHAR(100),
    override_justification TEXT
);

-- Index for policy violation queries
CREATE INDEX idx_policy_violations_timestamp ON policy_violations(violation_timestamp);
CREATE INDEX idx_policy_violations_rule ON policy_violations(rule_name);
CREATE INDEX idx_policy_violations_user ON policy_violations(user_id);
CREATE INDEX idx_policy_violations_concept ON policy_violations(concept_id);

-- Function to update policy flags with validation
CREATE OR REPLACE FUNCTION update_policy_flags(
    table_name TEXT,
    record_id BIGINT,
    new_flags JSONB,
    user_id TEXT,
    justification TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    current_flags JSONB;
    sql_query TEXT;
BEGIN
    -- Validate table name to prevent SQL injection
    IF table_name NOT IN ('terminology_concepts', 'concept_mappings', 'value_sets') THEN
        RAISE EXCEPTION 'Invalid table name: %', table_name;
    END IF;

    -- Get current policy flags
    sql_query := format('SELECT policy_flags FROM %I WHERE id = $1', table_name);
    EXECUTE sql_query INTO current_flags USING record_id;

    IF current_flags IS NULL THEN
        RAISE EXCEPTION 'Record not found: % with id %', table_name, record_id;
    END IF;

    -- Merge flags (new flags override existing ones)
    current_flags := current_flags || new_flags;

    -- Update the record
    sql_query := format('UPDATE %I SET policy_flags = $1, updated_at = NOW() WHERE id = $2', table_name);
    EXECUTE sql_query USING current_flags, record_id;

    -- Log the policy flag change
    INSERT INTO terminology_changes (
        table_name, record_id, operation, new_values, user_id, session_id,
        clinical_justification, clinical_impact_level
    ) VALUES (
        update_policy_flags.table_name,
        update_policy_flags.record_id,
        'UPDATE_POLICY_FLAGS',
        jsonb_build_object('policy_flags', new_flags),
        update_policy_flags.user_id,
        gen_random_uuid(),
        justification,
        CASE
            WHEN new_flags ? 'safetyLevel' AND new_flags->>'safetyLevel' IN ('high', 'critical') THEN 'high'
            WHEN new_flags ? 'requiresClinicalReview' AND (new_flags->>'requiresClinicalReview')::boolean THEN 'medium'
            ELSE 'low'
        END
    );

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Example policy flag configurations
COMMENT ON COLUMN terminology_concepts.policy_flags IS
'JSON object containing policy flags. Examples:
{
  "doNotAutoMap": true,
  "requiresClinicalReview": true,
  "safetyLevel": "high",
  "australianOnly": true,
  "deprecationDate": "2025-12-31",
  "replacementConcept": "SCTID:123456789",
  "regulatoryStatus": "approved",
  "clinicalReviewRequired": true,
  "drugInteractionWarning": true,
  "pbsListed": true,
  "tgaApproved": true
}';

-- View for policy-flagged concepts
CREATE VIEW policy_flagged_concepts AS
SELECT
    tc.id,
    tc.code,
    tc.display,
    tc.system,
    tc.policy_flags,
    CASE
        WHEN tc.policy_flags->>'safetyLevel' IN ('high', 'critical') THEN 'high_risk'
        WHEN tc.policy_flags->>'requiresClinicalReview' = 'true' THEN 'requires_review'
        WHEN tc.policy_flags->>'doNotAutoMap' = 'true' THEN 'manual_only'
        WHEN tc.policy_flags->>'deprecationDate' IS NOT NULL THEN 'deprecated'
        ELSE 'standard'
    END as risk_category,
    tc.created_at,
    tc.updated_at
FROM terminology_concepts tc
WHERE tc.policy_flags IS NOT NULL
  AND tc.policy_flags != '{}'::jsonb
ORDER BY
    CASE
        WHEN tc.policy_flags->>'safetyLevel' = 'critical' THEN 1
        WHEN tc.policy_flags->>'safetyLevel' = 'high' THEN 2
        WHEN tc.policy_flags->>'requiresClinicalReview' = 'true' THEN 3
        ELSE 4
    END,
    tc.updated_at DESC;
```

### Week 4: Integration Testing and API Enhancement

#### Days 22-24: REST API Integration with Audit and Policy
**Deliverable**: Enhanced REST endpoints with policy validation and audit tracking

```go
// internal/api/middleware/audit.go
package middleware

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "kb-7-terminology/internal/audit"
    "kb-7-terminology/internal/policy"
)

type AuditMiddleware struct {
    auditService  *audit.AuditService
    policyEngine  *policy.PolicyEngine
}

func NewAuditMiddleware(auditService *audit.AuditService, policyEngine *policy.PolicyEngine) *AuditMiddleware {
    return &AuditMiddleware{
        auditService: auditService,
        policyEngine: policyEngine,
    }
}

func (am *AuditMiddleware) SetAuditContext() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract user information from JWT token or headers
        userID := c.GetHeader("X-User-ID")
        if userID == "" {
            userID = "anonymous"
        }

        userRole := c.GetHeader("X-User-Role")
        if userRole == "" {
            userRole = "user"
        }

        sessionID := uuid.New()
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        // Set audit context
        changeCtx := &audit.ChangeContext{
            UserID:    userID,
            SessionID: sessionID,
            RequestID: requestID,
            ClientIP:  c.ClientIP(),
            UserAgent: c.GetHeader("User-Agent"),
            UserRole:  userRole,
        }

        // Store in request context
        ctx := context.WithValue(c.Request.Context(), "audit_context", changeCtx)
        c.Request = c.Request.WithContext(ctx)

        // Set database context for audit triggers
        if err := am.auditService.SetContext(ctx, changeCtx); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Failed to set audit context",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

func (am *AuditMiddleware) ValidatePolicy() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only validate for write operations
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
            c.Next()
            return
        }

        // Extract concept information from request
        // This would depend on the specific endpoint structure
        var concept *policy.Concept

        // For concept creation/update endpoints
        if c.Param("system") != "" && c.Param("code") != "" {
            concept = &policy.Concept{
                ID:     c.Param("system") + ":" + c.Param("code"),
                Code:   c.Param("code"),
                System: c.Param("system"),
                // Policy flags would be loaded from database or request
            }

            // Load existing concept with policy flags if updating
            if c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
                // Load from database
                existingConcept, err := am.loadConceptWithPolicyFlags(c.Request.Context(), concept.ID)
                if err != nil {
                    c.JSON(http.StatusInternalServerError, gin.H{
                        "error": "Failed to load existing concept for policy validation",
                    })
                    c.Abort()
                    return
                }
                concept = existingConcept
            }
        }

        if concept != nil {
            operation := am.mapHTTPMethodToOperation(c.Request.Method, c.FullPath())

            decision, err := am.policyEngine.EvaluateOperation(c.Request.Context(), concept, operation)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "Policy evaluation failed",
                })
                c.Abort()
                return
            }

            if !decision.Allowed {
                // Log policy violation
                am.logPolicyViolation(c, concept, operation, decision)

                c.JSON(http.StatusForbidden, gin.H{
                    "error":        "Operation blocked by policy",
                    "reason":       decision.Reason,
                    "warnings":     decision.Warnings,
                    "requirements": decision.Requirements,
                })
                c.Abort()
                return
            }

            // Store policy decision in context for later use
            c.Set("policy_decision", decision)
        }

        c.Next()
    }
}

func (am *AuditMiddleware) mapHTTPMethodToOperation(method, path string) string {
    switch method {
    case "POST":
        return "create"
    case "PUT":
        return "update"
    case "PATCH":
        return "partial_update"
    case "DELETE":
        return "delete"
    default:
        return "unknown"
    }
}

func (am *AuditMiddleware) loadConceptWithPolicyFlags(ctx context.Context, conceptID string) (*policy.Concept, error) {
    // Implementation would load concept from database with policy flags
    // This is a placeholder
    return &policy.Concept{
        ID: conceptID,
        PolicyFlags: make(map[string]interface{}),
    }, nil
}

func (am *AuditMiddleware) logPolicyViolation(c *gin.Context, concept *policy.Concept, operation string, decision *policy.PolicyDecision) {
    // Implementation would log to policy_violations table
    // This is a placeholder
}
```

**Enhanced API Endpoints**:
```go
// internal/api/handlers/concepts.go
package handlers

import (
    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"
    "kb-7-terminology/internal/audit"
    "kb-7-terminology/internal/policy"
    "kb-7-terminology/internal/services"
)

type ConceptHandler struct {
    conceptService   *services.ConceptService
    auditService     *audit.AuditService
    policyEngine     *policy.PolicyEngine
}

func NewConceptHandler(conceptService *services.ConceptService, auditService *audit.AuditService, policyEngine *policy.PolicyEngine) *ConceptHandler {
    return &ConceptHandler{
        conceptService: conceptService,
        auditService:   auditService,
        policyEngine:   policyEngine,
    }
}

// POST /v1/concepts/:system/:code/policy-flags
func (ch *ConceptHandler) UpdatePolicyFlags(c *gin.Context) {
    system := c.Param("system")
    code := c.Param("code")

    var request struct {
        PolicyFlags          map[string]interface{} `json:"policy_flags" binding:"required"`
        ClinicalJustification string                 `json:"clinical_justification"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Load existing concept
    concept, err := ch.conceptService.GetConcept(c.Request.Context(), system, code)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
        return
    }

    // Create policy concept for validation
    policyConcept := &policy.Concept{
        ID:          concept.ID,
        Code:        concept.Code,
        Display:     concept.Display,
        System:      concept.System,
        PolicyFlags: request.PolicyFlags,
    }

    // Validate policy change
    decision, err := ch.policyEngine.EvaluateOperation(c.Request.Context(), policyConcept, "update_policy_flags")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy validation failed"})
        return
    }

    if !decision.Allowed {
        c.JSON(http.StatusForbidden, gin.H{
            "error":        "Policy flag update blocked",
            "reason":       decision.Reason,
            "requirements": decision.Requirements,
        })
        return
    }

    // Update concept with new policy flags
    concept.PolicyFlags = request.PolicyFlags
    if err := ch.conceptService.UpdateConcept(c.Request.Context(), concept); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update concept"})
        return
    }

    // Track the change in audit system
    changeRecord := &audit.ChangeRecord{
        TableName:             "terminology_concepts",
        RecordID:              concept.ID,
        Operation:             "UPDATE_POLICY_FLAGS",
        NewValues:             map[string]interface{}{"policy_flags": request.PolicyFlags},
        ClinicalImpactLevel:   ch.determineClinicalImpactLevel(request.PolicyFlags),
        ApprovalStatus:        ch.determineApprovalStatus(request.PolicyFlags),
    }

    changeID, err := ch.auditService.TrackChange(c.Request.Context(), changeRecord)
    if err != nil {
        // Log error but don't fail the request
        c.Header("X-Warning", "Audit tracking failed")
    }

    c.JSON(http.StatusOK, gin.H{
        "message":     "Policy flags updated successfully",
        "change_id":   changeID,
        "policy_flags": request.PolicyFlags,
        "warnings":    decision.Warnings,
    })
}

// GET /v1/audit/trail
func (ch *ConceptHandler) GetAuditTrail(c *gin.Context) {
    filters := &audit.AuditFilters{}

    if startTime := c.Query("start_time"); startTime != "" {
        if t, err := time.Parse(time.RFC3339, startTime); err == nil {
            filters.StartTime = &t
        }
    }

    if endTime := c.Query("end_time"); endTime != "" {
        if t, err := time.Parse(time.RFC3339, endTime); err == nil {
            filters.EndTime = &t
        }
    }

    filters.UserID = c.Query("user_id")
    filters.TableName = c.Query("table_name")

    if limit := c.Query("limit"); limit != "" {
        if l, err := strconv.Atoi(limit); err == nil && l > 0 {
            filters.Limit = l
        }
    }

    entries, err := ch.auditService.GetAuditTrail(c.Request.Context(), filters)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit trail"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "audit_trail": entries,
        "count":       len(entries),
        "filters":     filters,
    })
}

// GET /v1/policy/violations
func (ch *ConceptHandler) GetPolicyViolations(c *gin.Context) {
    // Implementation for retrieving policy violations
    // This would query the policy_violations table

    c.JSON(http.StatusOK, gin.H{
        "message": "Policy violations endpoint - implementation pending",
    })
}

// POST /v1/audit/approve/:change_id
func (ch *ConceptHandler) ApproveChange(c *gin.Context) {
    changeIDStr := c.Param("change_id")
    changeID, err := strconv.ParseInt(changeIDStr, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid change ID"})
        return
    }

    var request struct {
        ClinicalJustification string `json:"clinical_justification" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Get user info from context
    auditCtx := c.Request.Context().Value("audit_context").(*audit.ChangeContext)

    if err := ch.auditService.ApproveChange(c.Request.Context(), changeID, auditCtx.UserID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve change"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":    "Change approved successfully",
        "change_id":  changeID,
        "approved_by": auditCtx.UserID,
    })
}

func (ch *ConceptHandler) determineClinicalImpactLevel(policyFlags map[string]interface{}) string {
    if safetyLevel, exists := policyFlags["safetyLevel"]; exists {
        if level, ok := safetyLevel.(string); ok {
            switch level {
            case "critical":
                return "critical"
            case "high":
                return "high"
            case "medium":
                return "medium"
            default:
                return "low"
            }
        }
    }

    if requiresReview, exists := policyFlags["requiresClinicalReview"]; exists {
        if required, ok := requiresReview.(bool); ok && required {
            return "medium"
        }
    }

    return "low"
}

func (ch *ConceptHandler) determineApprovalStatus(policyFlags map[string]interface{}) string {
    clinicalImpact := ch.determineClinicalImpactLevel(policyFlags)

    switch clinicalImpact {
    case "critical", "high":
        return "pending"
    case "medium":
        return "pending"
    default:
        return "approved"
    }
}
```

#### Days 25-28: Integration Testing and Documentation

**Integration Test Suite**:
```go
// tests/integration/phase1_integration_test.go
package integration

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "kb-7-terminology/internal/api"
    "kb-7-terminology/internal/audit"
    "kb-7-terminology/internal/policy"
)

type Phase1IntegrationTestSuite struct {
    router       *gin.Engine
    db           *sql.DB
    auditService *audit.AuditService
    policyEngine *policy.PolicyEngine
}

func (suite *Phase1IntegrationTestSuite) SetupTest() {
    // Setup test database and services
    // This would include setting up the test database with schema
    // and initializing all services
}

func (suite *Phase1IntegrationTestSuite) TearDownTest() {
    // Cleanup test data
}

func (suite *Phase1IntegrationTestSuite) TestClinicalGovernanceWorkflow() {
    // Test complete clinical governance workflow

    // 1. Create a concept that requires clinical review
    conceptData := map[string]interface{}{
        "code":    "test-drug-001",
        "display": "Test High-Risk Drug",
        "system":  "local",
        "policy_flags": map[string]interface{}{
            "safetyLevel": "high",
            "requiresClinicalReview": true,
            "drugInteractionWarning": true,
        },
    }

    conceptJSON, _ := json.Marshal(conceptData)
    req := httptest.NewRequest("POST", "/v1/concepts", bytes.NewBuffer(conceptJSON))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-User-ID", "test-user")
    req.Header.Set("X-User-Role", "clinical-informaticist")

    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)

    // Should be blocked due to clinical review requirement
    assert.Equal(suite.T(), http.StatusForbidden, w.Code)

    response := make(map[string]interface{})
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Contains(suite.T(), response["requirements"], "clinical_informaticist_approval")

    // 2. Test audit trail creation
    auditEntries, err := suite.auditService.GetAuditTrail(context.Background(), &audit.AuditFilters{
        UserID: "test-user",
        Limit:  10,
    })
    require.NoError(suite.T(), err)
    assert.NotEmpty(suite.T(), auditEntries)

    // 3. Test policy violation logging
    // Implementation would verify policy violations are logged correctly
}

func (suite *Phase1IntegrationTestSuite) TestAuditTrailCompleteness() {
    // Test that all operations are properly audited

    // Create initial concept
    concept := suite.createTestConcept("audit-test-001")

    // Update concept
    updateData := map[string]interface{}{
        "display": "Updated Test Concept",
    }
    suite.updateConcept(concept.ID, updateData)

    // Delete concept
    suite.deleteConcept(concept.ID)

    // Verify audit trail
    auditEntries, err := suite.auditService.GetAuditTrail(context.Background(), &audit.AuditFilters{
        TableName: "terminology_concepts",
        Limit:     100,
    })
    require.NoError(suite.T(), err)

    // Should have create, update, delete operations
    operations := make(map[string]bool)
    for _, entry := range auditEntries {
        if entry.RecordID == concept.ID {
            operations[entry.Operation] = true
        }
    }

    assert.True(suite.T(), operations["INSERT"])
    assert.True(suite.T(), operations["UPDATE"])
    assert.True(suite.T(), operations["DELETE"])

    // Verify PROV-O compliance
    for _, entry := range auditEntries {
        if entry.RecordID == concept.ID {
            assert.NotEmpty(suite.T(), entry.EntityID)
            assert.NotEmpty(suite.T(), entry.ChangeTimestamp)
            assert.NotEmpty(suite.T(), entry.UserID)
        }
    }
}

func (suite *Phase1IntegrationTestSuite) TestPolicyEngineRules() {
    // Test all policy rules individually

    testCases := []struct {
        name         string
        concept      *policy.Concept
        operation    string
        expectAllowed bool
        expectedRule string
    }{
        {
            name: "DoNotAutoMap rule blocks automatic operations",
            concept: &policy.Concept{
                ID:   "test-001",
                Code: "test-001",
                PolicyFlags: map[string]interface{}{
                    "doNotAutoMap": true,
                },
            },
            operation:     "auto_update",
            expectAllowed: false,
            expectedRule:  "do_not_auto_map",
        },
        {
            name: "ClinicalReview rule blocks high-safety concepts",
            concept: &policy.Concept{
                ID:   "test-002",
                Code: "test-002",
                PolicyFlags: map[string]interface{}{
                    "safetyLevel": "high",
                },
            },
            operation:     "update",
            expectAllowed: false,
            expectedRule:  "clinical_review_required",
        },
        {
            name: "Australian-only rule allows in AU deployment",
            concept: &policy.Concept{
                ID:   "test-003",
                Code: "test-003",
                PolicyFlags: map[string]interface{}{
                    "australianOnly": true,
                },
            },
            operation:     "create",
            expectAllowed: true, // Assuming test environment is AU
        },
    }

    for _, tc := range testCases {
        suite.T().Run(tc.name, func(t *testing.T) {
            decision, err := suite.policyEngine.EvaluateOperation(context.Background(), tc.concept, tc.operation)
            require.NoError(t, err)
            assert.Equal(t, tc.expectAllowed, decision.Allowed, "Policy decision allowed mismatch")

            if !tc.expectAllowed && tc.expectedRule != "" {
                assert.Contains(t, decision.Metadata["rule_triggered"], tc.expectedRule)
            }
        })
    }
}

func (suite *Phase1IntegrationTestSuite) TestSHA256ChecksumValidation() {
    // Test source file integrity validation

    sourceData := []byte("test terminology data")
    expectedChecksum := sha256.Sum256(sourceData)

    // Create mock terminology source
    source := &audit.TerminologySource{
        SourceName:       "test-terminology",
        Version:          "1.0.0",
        FilePath:         "/tmp/test-terminology.txt",
        FileSize:         int64(len(sourceData)),
        SHA256Checksum:   hex.EncodeToString(expectedChecksum[:]),
        ValidationStatus: "pending",
    }

    // Test checksum validation
    isValid := suite.auditService.ValidateSourceChecksum(source, sourceData)
    assert.True(suite.T(), isValid, "Checksum validation should pass for correct data")

    // Test with corrupted data
    corruptedData := []byte("corrupted terminology data")
    isValid = suite.auditService.ValidateSourceChecksum(source, corruptedData)
    assert.False(suite.T(), isValid, "Checksum validation should fail for corrupted data")
}

func TestPhase1IntegrationSuite(t *testing.T) {
    suite.Run(t, new(Phase1IntegrationTestSuite))
}

// Helper functions for test setup
func (suite *Phase1IntegrationTestSuite) createTestConcept(code string) *Concept {
    // Implementation for creating test concepts
}

func (suite *Phase1IntegrationTestSuite) updateConcept(id string, data map[string]interface{}) {
    // Implementation for updating concepts
}

func (suite *Phase1IntegrationTestSuite) deleteConcept(id string) {
    // Implementation for deleting concepts
}
```

## 📊 Phase 1 Acceptance Criteria & Testing

### Success Metrics

#### Clinical Governance Workflow (Week 1)
- ✅ **100% clinical review compliance**: All terminology changes route through GitHub PR workflow
- ✅ **Automated reviewer assignment**: Clinical informatics team automatically assigned based on change impact
- ✅ **Branch protection enforced**: No direct commits to main branch, PR approval required
- ✅ **Clinical impact assessment**: Automated analysis of change significance

**Validation Method**: Manual testing of PR workflow + automated tests

#### Provenance & Audit System (Week 2)
- ✅ **Complete audit trail**: All CRUD operations tracked with <1 second latency
- ✅ **W3C PROV-O compliance**: Provenance records include Entity, Activity, Agent
- ✅ **SHA256 integrity**: All terminology sources verified with checksums
- ✅ **Cross-session audit**: Audit trail queryable across user sessions

**Validation Method**: Database audit verification + PROV-O compliance testing

#### Clinical Policy Flags (Week 3)
- ✅ **Policy flag effectiveness**: 100% prevention of flagged unsafe operations
- ✅ **Configurable rules**: Policy engine supports pluggable clinical safety rules
- ✅ **Clinical context awareness**: Drug interaction and safety level validation
- ✅ **Regional restrictions**: Australian-only concepts properly restricted

**Validation Method**: Policy rule testing + clinical scenario validation

#### API Integration (Week 4)
- ✅ **Middleware integration**: Audit and policy validation on all write operations
- ✅ **REST endpoint enhancement**: Policy flag management endpoints functional
- ✅ **Error handling**: Graceful policy violation and audit failure handling
- ✅ **Performance impact**: <50ms additional latency for policy/audit processing

**Validation Method**: Integration testing + performance benchmarking

### Quality Gates

#### Security Review
- All database operations use parameterized queries (SQL injection prevention)
- User input validation on all policy flag operations
- Audit trail immutability verified (no direct audit table modifications)
- Session management and authentication context properly handled

#### Clinical Safety Review
- Clinical informaticist validation of policy rules accuracy
- Drug interaction rule validation against clinical literature
- Australian terminology compliance verification
- Clinical workflow usability testing

#### Performance Validation
- Audit system performance under 1000 concurrent operations
- Policy engine evaluation latency <10ms per rule
- Database query optimization for audit trail retrieval
- Memory usage profiling for policy flag operations

### Documentation Deliverables

#### Technical Documentation
- **API Documentation**: Complete OpenAPI specification for new endpoints
- **Database Schema**: ERD and migration documentation
- **Policy Rule Guide**: Clinical policy configuration documentation
- **Audit System Manual**: PROV-O compliance and querying guide

#### Clinical Documentation
- **Clinical Workflow Guide**: Step-by-step clinical review process
- **Policy Flag Reference**: Clinical meaning and usage of all policy flags
- **Safety Procedure Manual**: Clinical safety escalation procedures
- **Compliance Guide**: Regulatory audit trail procedures

#### Operational Documentation
- **Deployment Guide**: Phase 1 deployment procedures and rollback plans
- **Monitoring Setup**: Audit system and policy engine monitoring
- **Troubleshooting Guide**: Common issues and resolution procedures
- **Backup & Recovery**: Audit trail backup and recovery procedures

## 🚀 Phase 1 Deployment Plan

### Pre-Deployment Checklist
- [ ] Database migrations tested in staging environment
- [ ] Clinical review team trained on new workflow
- [ ] Policy rules validated by clinical informaticist
- [ ] Audit system performance benchmarked
- [ ] Rollback procedures documented and tested
- [ ] Monitoring and alerting configured
- [ ] Security review completed
- [ ] Clinical safety review completed

### Deployment Sequence
1. **Database Migration** (Maintenance window: 30 minutes)
   - Apply audit table migrations
   - Create policy rule configurations
   - Set up audit triggers
   - Verify data integrity

2. **Application Deployment** (Blue-green deployment: 15 minutes)
   - Deploy new API with audit/policy middleware
   - Verify health checks pass
   - Test policy engine functionality
   - Validate audit trail creation

3. **GitHub Workflow Configuration** (15 minutes)
   - Deploy clinical review workflow
   - Configure branch protection rules
   - Test PR template and reviewer assignment
   - Verify clinical team access

4. **Validation & Smoke Testing** (30 minutes)
   - Create test terminology change PR
   - Verify policy engine blocks unsafe operations
   - Validate audit trail completeness
   - Confirm clinical review workflow

### Post-Deployment Validation
- Monitor policy violation rates for first 48 hours
- Verify audit trail performance under production load
- Clinical team feedback on workflow usability
- Performance impact assessment on existing operations
- Security scan and vulnerability assessment

### Rollback Plan
If any critical issues are discovered:
1. **Immediate**: Disable policy enforcement (allow-all mode)
2. **Database**: Rollback audit trigger installation
3. **Application**: Deploy previous version without audit middleware
4. **GitHub**: Temporarily disable branch protection
5. **Investigation**: Root cause analysis and fix preparation

---

## 📞 Phase 1 Team & Communication

### Core Phase 1 Team
- **Technical Lead**: Senior Go Developer (audit system, policy engine)
- **Clinical Lead**: Clinical Informaticist (workflow design, policy validation)
- **DevOps Lead**: Infrastructure and deployment automation
- **QA Lead**: Integration testing and clinical workflow validation

### Daily Standup Format
- **Progress**: What was completed yesterday
- **Blockers**: Clinical review delays, technical issues
- **Plan**: Today's priorities and dependencies
- **Risks**: Emerging issues that could impact timeline

### Weekly Clinical Review
- **Policy Rule Validation**: Clinical informaticist reviews new rules
- **Workflow Usability**: Clinical team feedback on PR process
- **Safety Assessment**: Review policy violations and safety incidents
- **Documentation Review**: Clinical procedure and workflow documentation

### Phase 1 Success Celebration
Upon successful completion:
- **Clinical Safety Milestone**: Patient safety foundation established
- **Technical Achievement**: Enterprise-grade audit and governance system
- **Team Recognition**: Clinical and technical collaboration success
- **Stakeholder Communication**: Executive update on Phase 1 completion

---

*Phase 1 Implementation Specification v1.0*
*Generated: 2025-09-19*
*Clinical Safety Priority: Critical*
*Next Phase: Semantic Intelligence Layer (Phase 2)*