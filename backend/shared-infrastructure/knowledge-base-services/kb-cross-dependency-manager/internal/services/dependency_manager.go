package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CrossKBDependencyManager handles dependency tracking and management across KB services
type CrossKBDependencyManager struct {
	db     *gorm.DB
	logger *log.Logger
}

// DependencyTracker interface for dependency management operations
type DependencyTracker interface {
	RegisterDependency(ctx context.Context, dep *KBDependency) error
	DiscoverDependencies(ctx context.Context, lookbackHours int) (int, error)
	AnalyzeChangeImpact(ctx context.Context, change *ChangeRequest) (*ChangeImpactAnalysis, error)
	DetectConflicts(ctx context.Context, transactionID string, responses []KBResponse) ([]uuid.UUID, error)
	GetDependencyGraph(ctx context.Context, kbName string) (*DependencyGraph, error)
	ValidateDependencyHealth(ctx context.Context) (*HealthReport, error)
}

// KBDependency represents a dependency between KB services
type KBDependency struct {
	ID                     uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	SourceKB               string                 `json:"source_kb" gorm:"column:source_kb;not null"`
	SourceArtifactType     string                 `json:"source_artifact_type" gorm:"column:source_artifact_type;not null"`
	SourceArtifactID       string                 `json:"source_artifact_id" gorm:"column:source_artifact_id;not null"`
	SourceVersion          string                 `json:"source_version" gorm:"column:source_version;not null"`
	SourceEndpoint         *string                `json:"source_endpoint,omitempty" gorm:"column:source_endpoint"`
	TargetKB               string                 `json:"target_kb" gorm:"column:target_kb;not null"`
	TargetArtifactType     string                 `json:"target_artifact_type" gorm:"column:target_artifact_type;not null"`
	TargetArtifactID       string                 `json:"target_artifact_id" gorm:"column:target_artifact_id;not null"`
	TargetVersion          string                 `json:"target_version" gorm:"column:target_version;not null"`
	TargetEndpoint         *string                `json:"target_endpoint,omitempty" gorm:"column:target_endpoint"`
	DependencyType         string                 `json:"dependency_type" gorm:"column:dependency_type;not null"`
	DependencyStrength     string                 `json:"dependency_strength" gorm:"column:dependency_strength;default:medium"`
	RelationshipDescription *string               `json:"relationship_description,omitempty" gorm:"column:relationship_description"`
	RelationshipContext    map[string]interface{} `json:"relationship_context" gorm:"column:relationship_context;type:jsonb;default:'{}'"`
	TypicalUsageFrequency  *int                   `json:"typical_usage_frequency,omitempty" gorm:"column:typical_usage_frequency"`
	AverageResponseTimeMs  *int                   `json:"average_response_time_ms,omitempty" gorm:"column:average_response_time_ms"`
	FailureRatePercent     *float32               `json:"failure_rate_percent,omitempty" gorm:"column:failure_rate_percent"`
	Validated              bool                   `json:"validated" gorm:"column:validated;default:false"`
	ValidationTimestamp    *time.Time             `json:"validation_timestamp,omitempty" gorm:"column:validation_timestamp"`
	ValidationMethod       *string                `json:"validation_method,omitempty" gorm:"column:validation_method"`
	ValidationErrors       []string               `json:"validation_errors" gorm:"column:validation_errors;type:jsonb;default:'[]'"`
	ValidationWarnings     []string               `json:"validation_warnings" gorm:"column:validation_warnings;type:jsonb;default:'[]'"`
	LastVerified           *time.Time             `json:"last_verified,omitempty" gorm:"column:last_verified"`
	HealthStatus           string                 `json:"health_status" gorm:"column:health_status;default:unknown"`
	HealthCheckDetails     map[string]interface{} `json:"health_check_details" gorm:"column:health_check_details;type:jsonb;default:'{}'"`
	DiscoveredBy           string                 `json:"discovered_by" gorm:"column:discovered_by;not null"`
	DiscoveredAt           time.Time              `json:"discovered_at" gorm:"column:discovered_at;default:NOW()"`
	DiscoveryConfidence    float32                `json:"discovery_confidence" gorm:"column:discovery_confidence;default:0.5"`
	Active                 bool                   `json:"active" gorm:"column:active;default:true"`
	Deprecated             bool                   `json:"deprecated" gorm:"column:deprecated;default:false"`
	DeprecatedReason       *string                `json:"deprecated_reason,omitempty" gorm:"column:deprecated_reason"`
	DeprecatedAt           *time.Time             `json:"deprecated_at,omitempty" gorm:"column:deprecated_at"`
	ReplacementDependencyID *uuid.UUID            `json:"replacement_dependency_id,omitempty" gorm:"column:replacement_dependency_id"`
	CreatedAt              time.Time              `json:"created_at" gorm:"column:created_at;default:NOW()"`
	UpdatedAt              time.Time              `json:"updated_at" gorm:"column:updated_at;default:NOW()"`
	CreatedBy              string                 `json:"created_by" gorm:"column:created_by;not null"`
	LastModifiedBy         *string                `json:"last_modified_by,omitempty" gorm:"column:last_modified_by"`
}

func (KBDependency) TableName() string {
	return "kb_dependencies"
}

// ChangeRequest represents a requested change to a KB artifact
type ChangeRequest struct {
	KBName       string `json:"kb_name"`
	ArtifactID   string `json:"artifact_id"`
	ChangeType   string `json:"change_type"`
	OldVersion   string `json:"old_version,omitempty"`
	NewVersion   string `json:"new_version,omitempty"`
	Description  string `json:"description"`
	RequestedBy  string `json:"requested_by"`
}

// ChangeImpactAnalysis represents the analysis results of a change
type ChangeImpactAnalysis struct {
	ID                        uuid.UUID                    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	AnalysisTimestamp         time.Time                    `json:"analysis_timestamp" gorm:"column:analysis_timestamp;default:NOW()"`
	ChangedKB                 string                       `json:"changed_kb" gorm:"column:changed_kb;not null"`
	ChangedArtifactType       string                       `json:"changed_artifact_type" gorm:"column:changed_artifact_type;not null"`
	ChangedArtifactID         string                       `json:"changed_artifact_id" gorm:"column:changed_artifact_id;not null"`
	ChangeType                string                       `json:"change_type" gorm:"column:change_type;not null"`
	OldVersion                *string                      `json:"old_version,omitempty" gorm:"column:old_version"`
	NewVersion                *string                      `json:"new_version,omitempty" gorm:"column:new_version"`
	ChangeDescription         *string                      `json:"change_description,omitempty" gorm:"column:change_description"`
	ChangeScope               string                       `json:"change_scope" gorm:"column:change_scope;default:minor"`
	BreakingChange            bool                         `json:"breaking_change" gorm:"column:breaking_change;default:false"`
	BackwardCompatible        bool                         `json:"backward_compatible" gorm:"column:backward_compatible;default:true"`
	AnalysisStatus            string                       `json:"analysis_status" gorm:"column:analysis_status;default:pending"`
	DirectImpacts             []map[string]interface{}     `json:"direct_impacts" gorm:"column:direct_impacts;type:jsonb;default:'[]'"`
	IndirectImpacts           []map[string]interface{}     `json:"indirect_impacts" gorm:"column:indirect_impacts;type:jsonb;default:'[]'"`
	CascadeImpacts            []map[string]interface{}     `json:"cascade_impacts" gorm:"column:cascade_impacts;type:jsonb;default:'[]'"`
	TotalAffectedArtifacts    int                          `json:"total_affected_artifacts" gorm:"column:total_affected_artifacts;default:0"`
	AffectedKBServices        []string                     `json:"affected_kb_services" gorm:"column:affected_kb_services;type:text[]"`
	EstimatedPatientImpact    int                          `json:"estimated_patient_impact" gorm:"column:estimated_patient_impact;default:0"`
	EstimatedDowntimeMinutes  int                          `json:"estimated_downtime_minutes" gorm:"column:estimated_downtime_minutes;default:0"`
	RiskScore                 float32                      `json:"risk_score" gorm:"column:risk_score;default:0.0"`
	RiskLevel                 string                       `json:"risk_level" gorm:"column:risk_level;default:low"`
	RiskFactors               []map[string]interface{}     `json:"risk_factors" gorm:"column:risk_factors;type:jsonb;default:'[]'"`
	RecommendedActions        []map[string]interface{}     `json:"recommended_actions" gorm:"column:recommended_actions;type:jsonb;default:'[]'"`
	RollbackPlan              *string                      `json:"rollback_plan,omitempty" gorm:"column:rollback_plan"`
	TestingRequirements       []map[string]interface{}     `json:"testing_requirements" gorm:"column:testing_requirements;type:jsonb;default:'[]'"`
	RequiresApproval          bool                         `json:"requires_approval" gorm:"column:requires_approval;default:false"`
	ApprovalStatus            string                       `json:"approval_status" gorm:"column:approval_status;default:not_required"`
	ApprovedBy                *string                      `json:"approved_by,omitempty" gorm:"column:approved_by"`
	ApprovalTimestamp         *time.Time                   `json:"approval_timestamp,omitempty" gorm:"column:approval_timestamp"`
	ApprovalConditions        *string                      `json:"approval_conditions,omitempty" gorm:"column:approval_conditions"`
	ExecutionStatus           string                       `json:"execution_status" gorm:"column:execution_status;default:not_started"`
	ExecutionStartedAt        *time.Time                   `json:"execution_started_at,omitempty" gorm:"column:execution_started_at"`
	ExecutionCompletedAt      *time.Time                   `json:"execution_completed_at,omitempty" gorm:"column:execution_completed_at"`
	ExecutionNotes            *string                      `json:"execution_notes,omitempty" gorm:"column:execution_notes"`
	PreChangeValidation       map[string]interface{}       `json:"pre_change_validation" gorm:"column:pre_change_validation;type:jsonb;default:'{}'"`
	PostChangeValidation      map[string]interface{}       `json:"post_change_validation" gorm:"column:post_change_validation;type:jsonb;default:'{}'"`
	ValidationPassed          *bool                        `json:"validation_passed,omitempty" gorm:"column:validation_passed"`
	CreatedBy                 string                       `json:"created_by" gorm:"column:created_by;not null"`
	AssignedTo                *string                      `json:"assigned_to,omitempty" gorm:"column:assigned_to"`
	Priority                  int                          `json:"priority" gorm:"column:priority;default:5"`
	CreatedAt                 time.Time                    `json:"created_at" gorm:"column:created_at;default:NOW()"`
	UpdatedAt                 time.Time                    `json:"updated_at" gorm:"column:updated_at;default:NOW()"`
}

func (ChangeImpactAnalysis) TableName() string {
	return "change_impact_analysis"
}

// KBResponse represents a response from a KB service
type KBResponse struct {
	KBName       string                 `json:"kb_name"`
	Version      string                 `json:"version"`
	Type         string                 `json:"type"`
	Confidence   float32                `json:"confidence"`
	Recommendation map[string]interface{} `json:"recommendation"`
}

// DependencyGraph represents the dependency relationships for a KB
type DependencyGraph struct {
	RootKB       string                    `json:"root_kb"`
	Dependencies []DependencyNode          `json:"dependencies"`
	Generated    time.Time                 `json:"generated"`
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	KB           string           `json:"kb"`
	ArtifactID   string           `json:"artifact_id"`
	Version      string           `json:"version"`
	Relationship string           `json:"relationship"`
	Strength     string           `json:"strength"`
	Children     []DependencyNode `json:"children,omitempty"`
}

// HealthReport represents the health status of KB dependencies
type HealthReport struct {
	Timestamp          time.Time                      `json:"timestamp"`
	OverallHealth      string                         `json:"overall_health"`
	TotalDependencies  int                            `json:"total_dependencies"`
	HealthyCount       int                            `json:"healthy_count"`
	DegradedCount      int                            `json:"degraded_count"`
	FailingCount       int                            `json:"failing_count"`
	UnknownCount       int                            `json:"unknown_count"`
	ServiceHealthMap   map[string]ServiceHealthStatus `json:"service_health_map"`
	CriticalIssues     []string                       `json:"critical_issues"`
	Recommendations    []string                       `json:"recommendations"`
}

// ServiceHealthStatus represents the health status of a specific KB service
type ServiceHealthStatus struct {
	Status             string                 `json:"status"`
	LastCheck          time.Time              `json:"last_check"`
	ResponseTime       *int                   `json:"response_time_ms,omitempty"`
	FailureRate        *float32               `json:"failure_rate,omitempty"`
	DependencyCount    int                    `json:"dependency_count"`
	CriticalDepsHealth map[string]string      `json:"critical_deps_health"`
	Issues             []string               `json:"issues,omitempty"`
	Metrics            map[string]interface{} `json:"metrics,omitempty"`
}

// NewCrossKBDependencyManager creates a new dependency manager instance
func NewCrossKBDependencyManager(db *gorm.DB, logger *log.Logger) *CrossKBDependencyManager {
	return &CrossKBDependencyManager{
		db:     db,
		logger: logger,
	}
}

// RegisterDependency registers a new dependency between KB services
func (dm *CrossKBDependencyManager) RegisterDependency(ctx context.Context, dep *KBDependency) error {
	// Validate dependency before registration
	if err := dm.validateDependency(dep); err != nil {
		return fmt.Errorf("invalid dependency: %w", err)
	}

	// Check if dependency already exists
	var existing KBDependency
	result := dm.db.WithContext(ctx).Where(
		"source_kb = ? AND source_artifact_id = ? AND source_version = ? AND target_kb = ? AND target_artifact_id = ? AND target_version = ?",
		dep.SourceKB, dep.SourceArtifactID, dep.SourceVersion,
		dep.TargetKB, dep.TargetArtifactID, dep.TargetVersion,
	).First(&existing)

	if result.Error == nil {
		// Update existing dependency
		dep.ID = existing.ID
		dep.CreatedAt = existing.CreatedAt
		dep.UpdatedAt = time.Now()
		if err := dm.db.WithContext(ctx).Save(dep).Error; err != nil {
			return fmt.Errorf("failed to update dependency: %w", err)
		}
		dm.logger.Printf("Updated existing dependency: %s -> %s", dep.SourceKB, dep.TargetKB)
	} else if result.Error == gorm.ErrRecordNotFound {
		// Create new dependency
		dep.ID = uuid.New()
		dep.CreatedAt = time.Now()
		dep.UpdatedAt = time.Now()
		if err := dm.db.WithContext(ctx).Create(dep).Error; err != nil {
			return fmt.Errorf("failed to create dependency: %w", err)
		}
		dm.logger.Printf("Registered new dependency: %s -> %s", dep.SourceKB, dep.TargetKB)
	} else {
		return fmt.Errorf("failed to check existing dependency: %w", result.Error)
	}

	return nil
}

// DiscoverDependencies automatically discovers dependencies from runtime transactions
func (dm *CrossKBDependencyManager) DiscoverDependencies(ctx context.Context, lookbackHours int) (int, error) {
	// Call the stored procedure to discover dependencies
	var discoveredCount int
	
	result := dm.db.WithContext(ctx).Raw(
		"SELECT discover_dependencies_from_transactions(?)",
		lookbackHours,
	).Scan(&discoveredCount)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to discover dependencies: %w", result.Error)
	}

	dm.logger.Printf("Discovered %d dependencies from %d hours of transaction history", discoveredCount, lookbackHours)
	return discoveredCount, nil
}

// AnalyzeChangeImpact analyzes the impact of a proposed change across the KB ecosystem
func (dm *CrossKBDependencyManager) AnalyzeChangeImpact(ctx context.Context, change *ChangeRequest) (*ChangeImpactAnalysis, error) {
	// Call the stored procedure to analyze impact
	var analysisID uuid.UUID
	
	result := dm.db.WithContext(ctx).Raw(
		"SELECT analyze_change_impact(?, ?, ?, ?, ?)",
		change.KBName, change.ArtifactID, change.ChangeType, change.OldVersion, change.NewVersion,
	).Scan(&analysisID)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to analyze change impact: %w", result.Error)
	}

	// Retrieve the analysis record
	var analysis ChangeImpactAnalysis
	if err := dm.db.WithContext(ctx).Where("id = ?", analysisID).First(&analysis).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve impact analysis: %w", err)
	}

	dm.logger.Printf("Analyzed change impact for %s:%s, risk level: %s", change.KBName, change.ArtifactID, analysis.RiskLevel)
	return &analysis, nil
}

// DetectConflicts detects conflicts between KB responses in a transaction
func (dm *CrossKBDependencyManager) DetectConflicts(ctx context.Context, transactionID string, responses []KBResponse) ([]uuid.UUID, error) {
	// Convert responses to JSONB for the stored procedure
	responsesJSON, err := json.Marshal(responses)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal responses: %w", err)
	}

	// Call the conflict detection function
	var conflictIDs []uuid.UUID
	rows, err := dm.db.WithContext(ctx).Raw(
		"SELECT detect_kb_conflicts(?, ?::jsonb)",
		transactionID, string(responsesJSON),
	).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var conflictID uuid.UUID
		if err := rows.Scan(&conflictID); err != nil {
			return nil, fmt.Errorf("failed to scan conflict ID: %w", err)
		}
		conflictIDs = append(conflictIDs, conflictID)
	}

	dm.logger.Printf("Detected %d conflicts in transaction %s", len(conflictIDs), transactionID)
	return conflictIDs, nil
}

// GetDependencyGraph builds a dependency graph for the specified KB using both foundational and runtime dependencies
func (dm *CrossKBDependencyManager) GetDependencyGraph(ctx context.Context, kbName string) (*DependencyGraph, error) {
	// Get foundational dependencies from Phase 0 dependency graph
	var foundationalDeps []struct {
		SourceKB       string `gorm:"column:source_kb"`
		TargetKB       string `gorm:"column:target_kb"`
		DependencyType string `gorm:"column:dependency_type"`
		Required       bool   `gorm:"column:required"`
		Criticality    string `gorm:"column:criticality"`
		Description    string `gorm:"column:description"`
	}
	
	if err := dm.db.WithContext(ctx).Table("kb_dependency_graph").Where(
		"(source_kb = ? OR target_kb = ?)",
		kbName, kbName,
	).Find(&foundationalDeps).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve foundational dependencies: %w", err)
	}

	// Get runtime dependencies from detailed tracking
	var runtimeDependencies []KBDependency
	if err := dm.db.WithContext(ctx).Where(
		"(source_kb = ? OR target_kb = ?) AND active = true",
		kbName, kbName,
	).Find(&runtimeDependencies).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve runtime dependencies: %w", err)
	}

	// Build the dependency graph combining both foundational and runtime data
	graph := &DependencyGraph{
		RootKB:       kbName,
		Dependencies: dm.buildEnhancedDependencyNodes(kbName, foundationalDeps, runtimeDependencies),
		Generated:    time.Now(),
	}

	dm.logger.Printf("Built enhanced dependency graph for %s with %d nodes (%d foundational, %d runtime)", 
		kbName, len(graph.Dependencies), len(foundationalDeps), len(runtimeDependencies))
	return graph, nil
}

// ValidateDependencyHealth performs a comprehensive health check of all dependencies
func (dm *CrossKBDependencyManager) ValidateDependencyHealth(ctx context.Context) (*HealthReport, error) {
	report := &HealthReport{
		Timestamp:        time.Now(),
		ServiceHealthMap: make(map[string]ServiceHealthStatus),
		CriticalIssues:   []string{},
		Recommendations:  []string{},
	}

	// Get all active dependencies
	var dependencies []KBDependency
	if err := dm.db.WithContext(ctx).Where("active = true").Find(&dependencies).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve dependencies: %w", err)
	}

	report.TotalDependencies = len(dependencies)

	// Count health statuses
	healthCounts := make(map[string]int)
	kbServices := make(map[string][]KBDependency)

	for _, dep := range dependencies {
		healthCounts[dep.HealthStatus]++
		
		// Group by source KB
		kbServices[dep.SourceKB] = append(kbServices[dep.SourceKB], dep)
		// Also group by target KB
		kbServices[dep.TargetKB] = append(kbServices[dep.TargetKB], dep)
	}

	report.HealthyCount = healthCounts["healthy"]
	report.DegradedCount = healthCounts["degraded"]
	report.FailingCount = healthCounts["failing"]
	report.UnknownCount = healthCounts["unknown"]

	// Determine overall health
	if report.FailingCount > 0 {
		report.OverallHealth = "critical"
		report.CriticalIssues = append(report.CriticalIssues, fmt.Sprintf("%d failing dependencies detected", report.FailingCount))
	} else if report.DegradedCount > report.HealthyCount/2 {
		report.OverallHealth = "degraded"
		report.Recommendations = append(report.Recommendations, "Multiple degraded dependencies need attention")
	} else {
		report.OverallHealth = "healthy"
	}

	// Build service health map
	for kbName, deps := range kbServices {
		status := dm.calculateServiceHealth(deps)
		report.ServiceHealthMap[kbName] = status
		
		if status.Status == "failing" {
			report.CriticalIssues = append(report.CriticalIssues, fmt.Sprintf("Service %s has failing dependencies", kbName))
		}
	}

	dm.logger.Printf("Health validation completed: %s overall health with %d critical issues", 
		report.OverallHealth, len(report.CriticalIssues))
	
	return report, nil
}

// Private helper methods

func (dm *CrossKBDependencyManager) validateDependency(dep *KBDependency) error {
	if dep.SourceKB == "" || dep.TargetKB == "" {
		return fmt.Errorf("source and target KB names are required")
	}
	if dep.SourceArtifactID == "" || dep.TargetArtifactID == "" {
		return fmt.Errorf("source and target artifact IDs are required")
	}
	if dep.SourceVersion == "" || dep.TargetVersion == "" {
		return fmt.Errorf("source and target versions are required")
	}
	validTypes := map[string]bool{
		"references": true, "extends": true, "conflicts": true, 
		"overrides": true, "validates": true, "transforms": true,
	}
	if !validTypes[dep.DependencyType] {
		return fmt.Errorf("invalid dependency type: %s", dep.DependencyType)
	}
	validStrengths := map[string]bool{
		"critical": true, "strong": true, "medium": true, "weak": true, "optional": true,
	}
	if !validStrengths[dep.DependencyStrength] {
		return fmt.Errorf("invalid dependency strength: %s", dep.DependencyStrength)
	}
	return nil
}

// Enhanced dependency node building that combines foundational and runtime dependencies
func (dm *CrossKBDependencyManager) buildEnhancedDependencyNodes(rootKB string, foundationalDeps []struct {
	SourceKB       string `gorm:"column:source_kb"`
	TargetKB       string `gorm:"column:target_kb"`
	DependencyType string `gorm:"column:dependency_type"`
	Required       bool   `gorm:"column:required"`
	Criticality    string `gorm:"column:criticality"`
	Description    string `gorm:"column:description"`
}, runtimeDependencies []KBDependency) []DependencyNode {
	var nodes []DependencyNode
	nodeMap := make(map[string]*DependencyNode) // Use pointer for modification

	// Process foundational dependencies first (these are architectural/design-time)
	for _, dep := range foundationalDeps {
		var node DependencyNode
		var nodeKey string

		if dep.SourceKB == rootKB {
			// This KB depends on the target
			node = DependencyNode{
				KB:           dep.TargetKB,
				ArtifactID:   "foundational", // Foundational deps don't have specific artifacts
				Version:      "any",
				Relationship: dep.DependencyType,
				Strength:     dep.Criticality,
			}
			nodeKey = fmt.Sprintf("%s_%s", dep.TargetKB, dep.DependencyType)
		} else if dep.TargetKB == rootKB {
			// The source KB depends on this one (reverse dependency)
			node = DependencyNode{
				KB:           dep.SourceKB,
				ArtifactID:   "foundational",
				Version:      "any",
				Relationship: dep.DependencyType + "_reverse",
				Strength:     dep.Criticality,
			}
			nodeKey = fmt.Sprintf("%s_%s_reverse", dep.SourceKB, dep.DependencyType)
		} else {
			continue
		}

		// Store in map to avoid duplicates and allow enhancement
		nodeMap[nodeKey] = &node
	}

	// Enhance with runtime dependency data
	for _, runtimeDep := range runtimeDependencies {
		var nodeKey string

		if runtimeDep.SourceKB == rootKB {
			nodeKey = fmt.Sprintf("%s_%s", runtimeDep.TargetKB, runtimeDep.DependencyType)
		} else if runtimeDep.TargetKB == rootKB {
			nodeKey = fmt.Sprintf("%s_%s_reverse", runtimeDep.SourceKB, runtimeDep.DependencyType)
		} else {
			continue
		}

		// Check if we have a foundational node to enhance
		if existingNode, exists := nodeMap[nodeKey]; exists {
			// Enhance foundational node with runtime data
			existingNode.ArtifactID = runtimeDep.SourceArtifactID
			existingNode.Version = runtimeDep.SourceVersion
			if runtimeDep.DependencyStrength != "" {
				existingNode.Strength = runtimeDep.DependencyStrength
			}
		} else {
			// Create new runtime-only dependency node
			var node DependencyNode
			if runtimeDep.SourceKB == rootKB {
				node = DependencyNode{
					KB:           runtimeDep.TargetKB,
					ArtifactID:   runtimeDep.TargetArtifactID,
					Version:      runtimeDep.TargetVersion,
					Relationship: runtimeDep.DependencyType,
					Strength:     runtimeDep.DependencyStrength,
				}
			} else {
				node = DependencyNode{
					KB:           runtimeDep.SourceKB,
					ArtifactID:   runtimeDep.SourceArtifactID,
					Version:      runtimeDep.SourceVersion,
					Relationship: runtimeDep.DependencyType + "_reverse",
					Strength:     runtimeDep.DependencyStrength,
				}
			}
			nodeMap[nodeKey] = &node
		}
	}

	// Convert map to slice
	for _, node := range nodeMap {
		nodes = append(nodes, *node)
	}

	return nodes
}

// Legacy method for backward compatibility
func (dm *CrossKBDependencyManager) buildDependencyNodes(rootKB string, dependencies []KBDependency, visited map[string]bool) []DependencyNode {
	var nodes []DependencyNode
	nodeKey := fmt.Sprintf("%s", rootKB)
	
	if visited[nodeKey] {
		return nodes // Avoid infinite loops
	}
	visited[nodeKey] = true

	for _, dep := range dependencies {
		var node DependencyNode
		var targetKB string

		if dep.SourceKB == rootKB {
			// This KB depends on the target
			node = DependencyNode{
				KB:           dep.TargetKB,
				ArtifactID:   dep.TargetArtifactID,
				Version:      dep.TargetVersion,
				Relationship: dep.DependencyType,
				Strength:     dep.DependencyStrength,
			}
			targetKB = dep.TargetKB
		} else if dep.TargetKB == rootKB {
			// The source KB depends on this one
			node = DependencyNode{
				KB:           dep.SourceKB,
				ArtifactID:   dep.SourceArtifactID,
				Version:      dep.SourceVersion,
				Relationship: dep.DependencyType + "_reverse",
				Strength:     dep.DependencyStrength,
			}
			targetKB = dep.SourceKB
		} else {
			continue
		}

		// Recursively build children (limited depth to prevent infinite recursion)
		if len(visited) < 10 {
			node.Children = dm.buildDependencyNodes(targetKB, dependencies, visited)
		}
		
		nodes = append(nodes, node)
	}

	return nodes
}

func (dm *CrossKBDependencyManager) calculateServiceHealth(deps []KBDependency) ServiceHealthStatus {
	status := ServiceHealthStatus{
		Status:             "healthy",
		LastCheck:          time.Now(),
		DependencyCount:    len(deps),
		CriticalDepsHealth: make(map[string]string),
		Issues:             []string{},
		Metrics:            make(map[string]interface{}),
	}

	var totalResponseTime int
	var responseTimeCount int
	var totalFailureRate float32
	var failureRateCount int

	healthCounts := make(map[string]int)
	criticalFailures := 0

	for _, dep := range deps {
		healthCounts[dep.HealthStatus]++
		
		if dep.AverageResponseTimeMs != nil {
			totalResponseTime += *dep.AverageResponseTimeMs
			responseTimeCount++
		}
		
		if dep.FailureRatePercent != nil {
			totalFailureRate += *dep.FailureRatePercent
			failureRateCount++
		}

		if dep.DependencyStrength == "critical" && dep.HealthStatus != "healthy" {
			criticalFailures++
			status.CriticalDepsHealth[dep.TargetKB] = dep.HealthStatus
			if dep.HealthStatus == "failing" {
				status.Issues = append(status.Issues, fmt.Sprintf("Critical dependency %s is failing", dep.TargetKB))
			}
		}
	}

	// Calculate average metrics
	if responseTimeCount > 0 {
		avgResponseTime := totalResponseTime / responseTimeCount
		status.ResponseTime = &avgResponseTime
	}
	
	if failureRateCount > 0 {
		avgFailureRate := totalFailureRate / float32(failureRateCount)
		status.FailureRate = &avgFailureRate
		
		// Adjust status based on failure rate
		if avgFailureRate > 10.0 {
			status.Status = "failing"
		} else if avgFailureRate > 5.0 {
			status.Status = "degraded"
		}
	}

	// Determine overall status based on critical dependencies and health distribution
	if criticalFailures > 0 {
		status.Status = "failing"
	} else if healthCounts["failing"] > 0 {
		status.Status = "failing"
	} else if healthCounts["degraded"] > healthCounts["healthy"]/2 {
		status.Status = "degraded"
	}

	status.Metrics = map[string]interface{}{
		"healthy_count":   healthCounts["healthy"],
		"degraded_count":  healthCounts["degraded"],
		"failing_count":   healthCounts["failing"],
		"unknown_count":   healthCounts["unknown"],
		"critical_failures": criticalFailures,
	}

	return status
}