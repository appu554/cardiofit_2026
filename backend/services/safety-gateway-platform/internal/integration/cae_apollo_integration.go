package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/types"
	"safety-gateway-platform/pkg/logger"
)

// CAEApolloIntegration bridges the Phase 2 Safety Gateway with CAE Engine via Apollo Federation
type CAEApolloIntegration struct {
	apolloClient    *ApolloFederationClient
	caeClient       *CAEEngineClient
	snapshotManager *SnapshotManager
	logger          *logger.Logger
	config          *CAEIntegrationConfig
}

// CAEIntegrationConfig holds configuration for CAE integration
type CAEIntegrationConfig struct {
	ApolloFederationURL string        `yaml:"apollo_federation_url"`
	CAEServiceURL       string        `yaml:"cae_service_url"`
	SnapshotTTL         time.Duration `yaml:"snapshot_ttl"`
	EnableBatchCAE      bool          `yaml:"enable_batch_cae"`
	MaxConcurrentCAE    int           `yaml:"max_concurrent_cae"`
	KBVersionStrategy   string        `yaml:"kb_version_strategy"` // "latest", "pinned", "snapshot_locked"
}

// ClinicalSnapshot represents a point-in-time clinical context
type ClinicalSnapshot struct {
	ID          string                 `json:"id"`
	PatientID   string                 `json:"patient_id"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Data        map[string]interface{} `json:"data"`
	KBVersions  map[string]string      `json:"kb_versions"`
	Checksum    string                 `json:"checksum"`
	Completeness float64               `json:"completeness"`
	DataSources []string               `json:"data_sources"`
}

// CAERequest represents a request to the CAE Engine
type CAERequest struct {
	RequestID       string                 `json:"request_id"`
	SnapshotID      string                 `json:"snapshot_id"`
	ProposedAction  *ClinicalAction        `json:"proposed_action"`
	PatientContext  map[string]interface{} `json:"patient_context,omitempty"`
	KBVersions      map[string]string      `json:"kb_versions,omitempty"`
	EvaluationMode  string                 `json:"evaluation_mode"` // "standard", "what_if", "batch"
}

// CAEResponse represents a response from the CAE Engine
type CAEResponse struct {
	RequestID         string           `json:"request_id"`
	SnapshotID        string           `json:"snapshot_id"`
	Status            string           `json:"status"`
	Decision          string           `json:"decision"`
	RiskScore         float64          `json:"risk_score"`
	Findings          []CAEFinding     `json:"findings"`
	Recommendations   []Recommendation `json:"recommendations"`
	Explanations      []Explanation    `json:"explanations"`
	MLModulated       bool             `json:"ml_modulated"`
	MLRiskScore       float64          `json:"ml_risk_score,omitempty"`
	ProcessingTime    time.Duration    `json:"processing_time"`
	KBVersionsUsed    map[string]string `json:"kb_versions_used"`
	Provenance        *ProvenanceInfo   `json:"provenance"`
}

// CAEIntegrationOption configures the CAE integration
type CAEIntegrationOption func(*CAEApolloIntegration)

// WithSnapshotTTL sets the snapshot cache TTL
func WithSnapshotTTL(ttl time.Duration) CAEIntegrationOption {
	return func(cai *CAEApolloIntegration) {
		cai.config.SnapshotTTL = ttl
	}
}

// WithMaxCacheSize sets the maximum snapshot cache size
func WithMaxCacheSize(size int) CAEIntegrationOption {
	return func(cai *CAEApolloIntegration) {
		// This would be passed to snapshot manager options
	}
}

// WithBatchProcessing enables batch CAE processing
func WithBatchProcessing(enabled bool, maxConcurrent int) CAEIntegrationOption {
	return func(cai *CAEApolloIntegration) {
		cai.config.EnableBatchCAE = enabled
		cai.config.MaxConcurrentCAE = maxConcurrent
	}
}

// WithKBVersionStrategy sets the knowledge base version strategy
func WithKBVersionStrategy(strategy string) CAEIntegrationOption {
	return func(cai *CAEApolloIntegration) {
		cai.config.KBVersionStrategy = strategy
	}
}

// NewCAEApolloIntegration creates a new CAE Apollo integration
func NewCAEApolloIntegration(
	apolloURL string,
	caeURL string,
	logger *logger.Logger,
	opts ...CAEIntegrationOption,
) (*CAEApolloIntegration, error) {
	// Create default configuration
	config := &CAEIntegrationConfig{
		ApolloFederationURL: apolloURL,
		CAEServiceURL:       caeURL,
		SnapshotTTL:         30 * time.Minute,
		EnableBatchCAE:      true,
		MaxConcurrentCAE:    10,
		KBVersionStrategy:   "latest",
	}
	apolloClient, err := NewApolloFederationClient(config.ApolloFederationURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apollo client: %w", err)
	}

	caeClient, err := NewCAEEngineClient(config.CAEServiceURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAE client: %w", err)
	}

	snapshotManager := NewSnapshotManager(
		apolloClient,
		logger,
		WithCacheTTL(config.SnapshotTTL),
		WithMaxCacheSize(1000), // Default cache size
		WithChecksumValidation(true),
	)

	integration := &CAEApolloIntegration{
		apolloClient:    apolloClient,
		caeClient:       caeClient,
		snapshotManager: snapshotManager,
		logger:          logger,
		config:          config,
	}

	// Apply options
	for _, opt := range opts {
		opt(integration)
	}

	return integration, nil
}

// EvaluateWithSnapshot performs CAE evaluation using snapshot-based context
func (c *CAEApolloIntegration) EvaluateWithSnapshot(
	ctx context.Context,
	request *types.SafetyRequest,
) (*types.SafetyResponse, error) {
	c.logger.Info("Starting CAE evaluation with snapshot",
		zap.String("request_id", request.RequestID),
		zap.String("patient_id", request.PatientID),
	)

	// Step 1: Create or retrieve clinical snapshot
	snapshot, err := c.createClinicalSnapshot(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create clinical snapshot: %w", err)
	}

	// Step 2: Resolve KB versions for snapshot
	kbVersions, err := c.resolveKBVersions(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve KB versions: %w", err)
	}

	// Step 3: Create CAE request with snapshot context
	caeRequest := &CAERequest{
		RequestID:      request.RequestID,
		SnapshotID:     snapshot.ID,
		ProposedAction: c.convertToCAEAction(request),
		KBVersions:     kbVersions,
		EvaluationMode: "standard",
	}

	// Step 4: Execute CAE evaluation
	caeResponse, err := c.caeClient.Evaluate(ctx, caeRequest)
	if err != nil {
		return nil, fmt.Errorf("CAE evaluation failed: %w", err)
	}

	// Step 5: Convert CAE response to Safety Gateway response
	safetyResponse := c.convertToSafetyResponse(caeResponse, request, snapshot)

	c.logger.Info("CAE evaluation completed",
		zap.String("request_id", request.RequestID),
		zap.String("snapshot_id", snapshot.ID),
		zap.String("decision", caeResponse.Decision),
		zap.Float64("risk_score", caeResponse.RiskScore),
		zap.Duration("processing_time", caeResponse.ProcessingTime),
	)

	return safetyResponse, nil
}

// BatchEvaluateWithSnapshots performs batch CAE evaluation with snapshot optimization
func (c *CAEApolloIntegration) BatchEvaluateWithSnapshots(
	ctx context.Context,
	requests []*types.SafetyRequest,
) ([]*types.SafetyResponse, error) {
	if !c.config.EnableBatchCAE {
		// Fall back to individual evaluations
		return c.evaluateIndividually(ctx, requests)
	}

	c.logger.Info("Starting batch CAE evaluation",
		zap.Int("request_count", len(requests)),
	)

	// Step 1: Group requests by patient for snapshot optimization
	patientGroups := c.groupRequestsByPatient(requests)

	// Step 2: Create snapshots for each patient group
	snapshots := make(map[string]*ClinicalSnapshot)
	for patientID, patientRequests := range patientGroups {
		snapshot, err := c.createClinicalSnapshot(ctx, patientRequests[0]) // Use first request for context
		if err != nil {
			c.logger.Error("Failed to create snapshot for patient",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
			continue
		}
		snapshots[patientID] = snapshot
	}

	// Step 3: Create batch CAE requests
	var batchRequests []*CAERequest
	for patientID, patientRequests := range patientGroups {
		snapshot, exists := snapshots[patientID]
		if !exists {
			continue
		}

		kbVersions, err := c.resolveKBVersions(ctx, snapshot)
		if err != nil {
			c.logger.Error("Failed to resolve KB versions",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
			continue
		}

		for _, request := range patientRequests {
			batchRequests = append(batchRequests, &CAERequest{
				RequestID:      request.RequestID,
				SnapshotID:     snapshot.ID,
				ProposedAction: c.convertToCAEAction(request),
				KBVersions:     kbVersions,
				EvaluationMode: "batch",
			})
		}
	}

	// Step 4: Execute batch CAE evaluation
	batchResponses, err := c.caeClient.BatchEvaluate(ctx, batchRequests)
	if err != nil {
		return nil, fmt.Errorf("batch CAE evaluation failed: %w", err)
	}

	// Step 5: Convert responses back to Safety Gateway format
	var responses []*types.SafetyResponse
	for _, caeResponse := range batchResponses {
		// Find original request
		var originalRequest *types.SafetyRequest
		for _, req := range requests {
			if req.RequestID == caeResponse.RequestID {
				originalRequest = req
				break
			}
		}

		if originalRequest != nil {
			snapshot := c.findSnapshotByID(snapshots, caeResponse.SnapshotID)
			safetyResponse := c.convertToSafetyResponse(caeResponse, originalRequest, snapshot)
			responses = append(responses, safetyResponse)
		}
	}

	c.logger.Info("Batch CAE evaluation completed",
		zap.Int("request_count", len(requests)),
		zap.Int("response_count", len(responses)),
	)

	return responses, nil
}

// createClinicalSnapshot creates a comprehensive clinical snapshot via Apollo Federation
func (c *CAEApolloIntegration) createClinicalSnapshot(
	ctx context.Context,
	request *types.SafetyRequest,
) (*ClinicalSnapshot, error) {
	// Check if snapshot already exists and is valid
	if existingSnapshot := c.snapshotManager.GetCachedSnapshot(request.PatientID); existingSnapshot != nil {
		if !existingSnapshot.IsExpired() {
			return existingSnapshot, nil
		}
	}

	// Build comprehensive GraphQL query for Apollo Federation
	query := c.buildSnapshotGraphQLQuery(request)

	// Execute query against Apollo Federation
	apolloResponse, err := c.apolloClient.Query(ctx, query, map[string]interface{}{
		"patientId": request.PatientID,
	})
	if err != nil {
		return nil, fmt.Errorf("Apollo Federation query failed: %w", err)
	}

	// Create snapshot from Apollo response
	snapshot := &ClinicalSnapshot{
		ID:          c.generateSnapshotID(request.PatientID),
		PatientID:   request.PatientID,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(c.config.SnapshotTTL),
		Data:        apolloResponse.Data,
		DataSources: apolloResponse.DataSources,
		Completeness: c.calculateCompleteness(apolloResponse.Data),
	}

	// Generate checksum for integrity
	snapshot.Checksum = c.generateChecksum(snapshot.Data)

	// Cache the snapshot
	c.snapshotManager.CacheSnapshot(snapshot)

	return snapshot, nil
}

// buildSnapshotGraphQLQuery builds a comprehensive GraphQL query for clinical snapshot
func (c *CAEApolloIntegration) buildSnapshotGraphQLQuery(request *types.SafetyRequest) string {
	return fmt.Sprintf(`
		query GetClinicalSnapshot($patientId: ID!) {
			patient(id: $patientId) {
				id
				demographics {
					age
					gender
					weight
					height
					ethnicity
				}
				allergies {
					substance
					severity
					reaction
					status
				}
				conditions {
					code
					display
					onsetDate
					status
					severity
				}
				medications {
					code
					display
					dosage
					frequency
					startDate
					status
					prescriber
				}
				labResults {
					code
					display
					value
					unit
					referenceRange
					date
					status
				}
				vitalSigns {
					code
					display
					value
					unit
					date
				}
				procedures {
					code
					display
					performedDate
					status
				}
				encounters {
					id
					type
					startDate
					endDate
					provider
					location
				}
			}
			
			# Additional context for specific action types
			%s
		}
	`, c.buildActionSpecificQuery(request))
}

// buildActionSpecificQuery adds action-specific queries
func (c *CAEApolloIntegration) buildActionSpecificQuery(request *types.SafetyRequest) string {
	switch request.ActionType {
	case "medication_prescribe", "medication_interaction":
		return `
			medicationKnowledge {
				interactions(codes: $medicationCodes) {
					drug1
					drug2
					severity
					mechanism
				}
				contraindications(codes: $medicationCodes) {
					medication
					condition
					severity
				}
			}
		`
	case "lab_order":
		return `
			labKnowledge {
				normalRanges(age: $age, gender: $gender) {
					test
					min
					max
					unit
				}
			}
		`
	default:
		return ""
	}
}

// resolveKBVersions determines KB versions based on snapshot and strategy
func (c *CAEApolloIntegration) resolveKBVersions(
	ctx context.Context,
	snapshot *ClinicalSnapshot,
) (map[string]string, error) {
	switch c.config.KBVersionStrategy {
	case "latest":
		return c.getLatestKBVersions(ctx)
	case "pinned":
		return c.getPinnedKBVersions()
	case "snapshot_locked":
		return c.getSnapshotLockedVersions(snapshot)
	default:
		return c.getLatestKBVersions(ctx)
	}
}

// getLatestKBVersions retrieves the latest versions of all KBs
func (c *CAEApolloIntegration) getLatestKBVersions(ctx context.Context) (map[string]string, error) {
	query := `
		query GetKBVersions {
			knowledgeBases {
				kb1_dosing { version }
				kb3_guidelines { version }
				kb4_safety { version }
				kb5_ddi { version }
				kb7_terminology { version }
			}
		}
	`

	response, err := c.apolloClient.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get KB versions: %w", err)
	}

	versions := make(map[string]string)
	if kbs, ok := response.Data["knowledgeBases"].(map[string]interface{}); ok {
		for kbName, kbInfo := range kbs {
			if info, ok := kbInfo.(map[string]interface{}); ok {
				if version, ok := info["version"].(string); ok {
					versions[kbName] = version
				}
			}
		}
	}

	return versions, nil
}

// convertToCAEAction converts Safety Gateway request to CAE action
func (c *CAEApolloIntegration) convertToCAEAction(request *types.SafetyRequest) *ClinicalAction {
	return &ClinicalAction{
		Type:          request.ActionType,
		PatientID:     request.PatientID,
		MedicationIDs: request.MedicationIDs,
		ConditionIDs:  request.ConditionIDs,
		Priority:      request.Priority,
		Metadata:      request.Metadata,
	}
}

// convertToSafetyResponse converts CAE response to Safety Gateway response
func (c *CAEApolloIntegration) convertToSafetyResponse(
	caeResponse *CAEResponse,
	originalRequest *types.SafetyRequest,
	snapshot *ClinicalSnapshot,
) *types.SafetyResponse {
	// Convert CAE status to Safety Gateway status
	var status types.SafetyStatus
	switch caeResponse.Decision {
	case "SAFE", "APPROVE":
		status = types.SafetyStatusSafe
	case "WARNING", "CAUTION":
		status = types.SafetyStatusWarning
	case "UNSAFE", "REJECT":
		status = types.SafetyStatusUnsafe
	default:
		status = types.SafetyStatusError
	}

	// Convert findings to engine results
	var engineResults []types.EngineResult
	if len(caeResponse.Findings) > 0 {
		engineResult := types.EngineResult{
			EngineID:     "cae_engine",
			EngineName:   "Clinical Assertion Engine",
			Status:       status,
			RiskScore:    caeResponse.RiskScore,
			Violations:   c.convertFindingsToViolations(caeResponse.Findings),
			Confidence:   c.calculateConfidence(caeResponse),
			Duration:     caeResponse.ProcessingTime,
			Tier:         types.TierVetoCritical,
		}

		if caeResponse.MLModulated {
			engineResult.Metadata = map[string]interface{}{
				"ml_modulated":   true,
				"ml_risk_score":  caeResponse.MLRiskScore,
				"kb_versions":    caeResponse.KBVersionsUsed,
				"snapshot_id":    snapshot.ID,
			}
		}

		engineResults = append(engineResults, engineResult)
	}

	return &types.SafetyResponse{
		RequestID:      originalRequest.RequestID,
		Status:         status,
		RiskScore:      caeResponse.RiskScore,
		EngineResults:  engineResults,
		ProcessingTime: caeResponse.ProcessingTime,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"cae_decision":   caeResponse.Decision,
			"snapshot_id":    snapshot.ID,
			"kb_versions":    caeResponse.KBVersionsUsed,
			"ml_modulated":   caeResponse.MLModulated,
			"data_sources":   snapshot.DataSources,
			"completeness":   snapshot.Completeness,
		},
	}
}

// Helper types and methods...

type ClinicalAction struct {
	Type          string                 `json:"type"`
	PatientID     string                 `json:"patient_id"`
	MedicationIDs []string               `json:"medication_ids,omitempty"`
	ConditionIDs  []string               `json:"condition_ids,omitempty"`
	Priority      string                 `json:"priority"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type CAEFinding struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
}

type Recommendation struct {
	ActionText string `json:"action_text"`
	Rationale  string `json:"rationale"`
	Priority   string `json:"priority"`
}

type Explanation struct {
	CheckerName   string `json:"checker_name"`
	Reasoning     string `json:"reasoning"`
	Evidence      string `json:"evidence"`
	RiskFactors   []string `json:"risk_factors"`
}

type ProvenanceInfo struct {
	SnapshotID  string            `json:"snapshot_id"`
	KBVersions  map[string]string `json:"kb_versions"`
	Timestamp   time.Time         `json:"timestamp"`
	DataSources []string          `json:"data_sources"`
}

// Additional helper methods would be implemented here...
// - groupRequestsByPatient
// - findSnapshotByID
// - calculateCompleteness
// - generateSnapshotID
// - generateChecksum
// - convertFindingsToViolations
// - calculateConfidence
// - etc.