package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/infrastructure/clients"
)

// ContextGatewayService handles the integration between Recipe Resolution and Context Snapshots
// This implements Phase 2 of the Recipe Resolver workflow: Context Assembly via Snapshot
type ContextGatewayService struct {
	contextClient *clients.ContextGatewayClient
	logger        *zap.Logger
	config        ContextGatewayConfig
}

// ContextGatewayConfig contains configuration for the Context Gateway Service
type ContextGatewayConfig struct {
	// Snapshot settings
	DefaultSnapshotTTL          time.Duration            `json:"default_snapshot_ttl"`
	FreshnessRequirements       map[string]time.Duration `json:"freshness_requirements"`
	SnapshotCreationTimeout     time.Duration            `json:"snapshot_creation_timeout"`
	
	// Retry settings
	MaxRetries                  int                      `json:"max_retries"`
	RetryBackoffMultiplier      float64                  `json:"retry_backoff_multiplier"`
	InitialRetryDelay          time.Duration            `json:"initial_retry_delay"`
	
	// Quality settings
	MinRequiredQualityScore     float64                  `json:"min_required_quality_score"`
	RequiredFields             []string                 `json:"required_fields"`
	OptionalFields             []string                 `json:"optional_fields"`
	
	// Performance settings
	EnableAsyncSnapshotCreation bool                     `json:"enable_async_snapshot_creation"`
	SnapshotCreationWorkers     int                      `json:"snapshot_creation_workers"`
	
	// Validation settings
	EnableSnapshotValidation    bool                     `json:"enable_snapshot_validation"`
	ValidationLevel            string                   `json:"validation_level"`
}

// SnapshotCreationRequest contains the data needed to create a clinical snapshot
type SnapshotCreationRequest struct {
	PatientID             uuid.UUID                 `json:"patient_id"`
	RecipeID              uuid.UUID                 `json:"recipe_id"`
	RecipeResolution      *entities.RecipeResolution `json:"recipe_resolution"`
	PatientContext        entities.PatientContext    `json:"patient_context"`
	SnapshotType          string                    `json:"snapshot_type"`
	Priority              string                    `json:"priority"`
	CreatedBy             string                    `json:"created_by"`
	RequireValidation     bool                      `json:"require_validation"`
	CustomFreshness       map[string]time.Duration  `json:"custom_freshness,omitempty"`
}

// SnapshotCreationResult contains the result of snapshot creation
type SnapshotCreationResult struct {
	SnapshotID        string                      `json:"snapshot_id"`
	Status           string                      `json:"status"`
	CreatedAt        time.Time                   `json:"created_at"`
	ExpiresAt        time.Time                   `json:"expires_at"`
	QualityScore     float64                     `json:"quality_score"`
	ProcessingTime   time.Duration               `json:"processing_time"`
	ValidationResult *clients.ValidateSnapshotResponse `json:"validation_result,omitempty"`
	Warnings         []string                    `json:"warnings,omitempty"`
	DataSources      map[string]interface{}      `json:"data_sources"`
}

// NewContextGatewayService creates a new Context Gateway Service
func NewContextGatewayService(
	contextClient *clients.ContextGatewayClient,
	logger *zap.Logger,
	config ContextGatewayConfig,
) *ContextGatewayService {
	return &ContextGatewayService{
		contextClient: contextClient,
		logger:        logger,
		config:        config,
	}
}

// CreateSnapshotFromResolution creates a clinical snapshot from recipe resolution
// This is the main entry point for Phase 2: Context Assembly via Snapshot
func (c *ContextGatewayService) CreateSnapshotFromResolution(
	ctx context.Context,
	request *SnapshotCreationRequest,
) (*SnapshotCreationResult, error) {
	startTime := time.Now()
	
	// Validate request
	if err := c.validateSnapshotRequest(request); err != nil {
		c.logger.Error("Invalid snapshot creation request",
			zap.Error(err),
			zap.String("recipe_id", request.RecipeID.String()),
			zap.String("patient_id", request.PatientID.String()),
		)
		return nil, errors.Wrap(err, "invalid snapshot request")
	}

	// Create snapshot with retry logic
	var result *SnapshotCreationResult
	var err error
	
	if c.config.EnableAsyncSnapshotCreation {
		result, err = c.createSnapshotAsync(ctx, request)
	} else {
		result, err = c.createSnapshotSync(ctx, request)
	}
	
	if err != nil {
		c.logger.Error("Failed to create snapshot",
			zap.Error(err),
			zap.String("recipe_id", request.RecipeID.String()),
			zap.String("patient_id", request.PatientID.String()),
			zap.Duration("processing_time", time.Since(startTime)),
		)
		return nil, err
	}

	// Validate snapshot if required
	if c.config.EnableSnapshotValidation && request.RequireValidation {
		validationResult, err := c.validateSnapshot(ctx, result.SnapshotID)
		if err != nil {
			c.logger.Warn("Snapshot validation failed",
				zap.Error(err),
				zap.String("snapshot_id", result.SnapshotID),
			)
			// Don't fail the entire operation, but include the warning
			result.Warnings = append(result.Warnings, fmt.Sprintf("Validation failed: %s", err.Error()))
		} else {
			result.ValidationResult = validationResult
		}
	}

	result.ProcessingTime = time.Since(startTime)
	
	c.logger.Info("Snapshot created successfully",
		zap.String("snapshot_id", result.SnapshotID),
		zap.String("recipe_id", request.RecipeID.String()),
		zap.String("patient_id", request.PatientID.String()),
		zap.Float64("quality_score", result.QualityScore),
		zap.Duration("processing_time", result.ProcessingTime),
	)

	return result, nil
}

// createSnapshotSync creates a snapshot synchronously with retry logic
func (c *ContextGatewayService) createSnapshotSync(
	ctx context.Context,
	request *SnapshotCreationRequest,
) (*SnapshotCreationResult, error) {
	var lastErr error
	retryDelay := c.config.InitialRetryDelay

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("Retrying snapshot creation",
				zap.Int("attempt", attempt),
				zap.Duration("delay", retryDelay),
				zap.String("recipe_id", request.RecipeID.String()),
			)
			
			select {
			case <-time.After(retryDelay):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := c.performSnapshotCreation(ctx, request)
		if err == nil {
			if attempt > 0 {
				c.logger.Info("Snapshot creation succeeded after retry",
					zap.Int("attempts", attempt+1),
					zap.String("snapshot_id", result.SnapshotID),
				)
			}
			return result, nil
		}

		lastErr = err
		
		// Check if error is retryable
		if !c.isRetryableError(err) {
			c.logger.Debug("Non-retryable error encountered",
				zap.Error(err),
				zap.Int("attempt", attempt),
			)
			break
		}

		// Calculate next retry delay with exponential backoff
		retryDelay = time.Duration(float64(retryDelay) * c.config.RetryBackoffMultiplier)
	}

	return nil, errors.Wrap(lastErr, "snapshot creation failed after retries")
}

// createSnapshotAsync creates a snapshot asynchronously
func (c *ContextGatewayService) createSnapshotAsync(
	ctx context.Context,
	request *SnapshotCreationRequest,
) (*SnapshotCreationResult, error) {
	// For now, implement as synchronous but with a separate goroutine for validation
	// This can be expanded to use a worker pool pattern
	result, err := c.createSnapshotSync(ctx, request)
	if err != nil {
		return nil, err
	}

	// Perform validation asynchronously if enabled
	if c.config.EnableSnapshotValidation && request.RequireValidation {
		go func() {
			validationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			_, validationErr := c.validateSnapshot(validationCtx, result.SnapshotID)
			if validationErr != nil {
				c.logger.Error("Async snapshot validation failed",
					zap.Error(validationErr),
					zap.String("snapshot_id", result.SnapshotID),
				)
			}
		}()
	}

	return result, nil
}

// performSnapshotCreation performs the actual snapshot creation via Context Gateway
func (c *ContextGatewayService) performSnapshotCreation(
	ctx context.Context,
	request *SnapshotCreationRequest,
) (*SnapshotCreationResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.SnapshotCreationTimeout)
	defer cancel()

	// Determine freshness requirements
	freshnessReqs := c.config.FreshnessRequirements
	if request.CustomFreshness != nil && len(request.CustomFreshness) > 0 {
		// Merge custom freshness requirements
		freshnessReqs = make(map[string]time.Duration)
		for k, v := range c.config.FreshnessRequirements {
			freshnessReqs[k] = v
		}
		for k, v := range request.CustomFreshness {
			freshnessReqs[k] = v
		}
	}

	// Create snapshot request for Context Gateway
	gatewayRequest := &clients.CreateSnapshotRequest{
		PatientID:             request.PatientID.String(),
		RecipeID:              request.RecipeID.String(),
		SnapshotType:          request.SnapshotType,
		FreshnessRequirements: freshnessReqs,
		RequiredFields:        c.config.RequiredFields,
		OptionalFields:        c.config.OptionalFields,
		Priority:              request.Priority,
		ExpiryDuration:        c.config.DefaultSnapshotTTL,
	}

	// Call Context Gateway to create snapshot
	gatewayResponse, err := c.contextClient.CreateSnapshot(ctx, gatewayRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create snapshot via context gateway")
	}

	// Check quality score requirement
	if gatewayResponse.QualityScore < c.config.MinRequiredQualityScore {
		c.logger.Warn("Snapshot quality below threshold",
			zap.Float64("quality_score", gatewayResponse.QualityScore),
			zap.Float64("min_required", c.config.MinRequiredQualityScore),
			zap.String("snapshot_id", gatewayResponse.SnapshotID),
		)
	}

	// Convert to result format
	result := &SnapshotCreationResult{
		SnapshotID:     gatewayResponse.SnapshotID,
		Status:         gatewayResponse.Status,
		CreatedAt:      gatewayResponse.CreatedAt,
		ExpiresAt:      gatewayResponse.ExpiresAt,
		QualityScore:   gatewayResponse.QualityScore,
		ProcessingTime: gatewayResponse.ProcessingTime,
		Warnings:       gatewayResponse.Warnings,
		DataSources:    gatewayResponse.DataSources,
	}

	return result, nil
}

// validateSnapshot validates a created snapshot
func (c *ContextGatewayService) validateSnapshot(
	ctx context.Context,
	snapshotID string,
) (*clients.ValidateSnapshotResponse, error) {
	validateRequest := &clients.ValidateSnapshotRequest{
		SnapshotID:      snapshotID,
		ValidationLevel: c.config.ValidationLevel,
		CheckTypes:      []string{"completeness", "accuracy", "freshness", "security"},
		Metadata: map[string]string{
			"service":  "medication-service-v2",
			"phase":    "context_assembly",
			"workflow": "recipe_resolution",
		},
	}

	return c.contextClient.ValidateSnapshot(ctx, validateRequest)
}

// GetSnapshot retrieves a snapshot by ID
func (c *ContextGatewayService) GetSnapshot(
	ctx context.Context,
	snapshotID string,
) (*clients.GetSnapshotResponse, error) {
	return c.contextClient.GetSnapshot(ctx, snapshotID)
}

// SupersedeSnapshot marks an old snapshot as superseded
func (c *ContextGatewayService) SupersedeSnapshot(
	ctx context.Context,
	oldSnapshotID string,
	newSnapshotID string,
	reason string,
	supersededBy string,
) error {
	request := &clients.SupersedeSnapshotRequest{
		OldSnapshotID: oldSnapshotID,
		NewSnapshotID: newSnapshotID,
		Reason:        reason,
		SupersededBy:  supersededBy,
	}

	return c.contextClient.SupersedeSnapshot(ctx, request)
}

// ListSnapshotsForPatient retrieves snapshots for a specific patient
func (c *ContextGatewayService) ListSnapshotsForPatient(
	ctx context.Context,
	patientID uuid.UUID,
	status string,
	limit int,
) (*clients.ListSnapshotsResponse, error) {
	request := &clients.ListSnapshotsRequest{
		PatientID: patientID.String(),
		Status:    status,
		Limit:     limit,
	}

	return c.contextClient.ListSnapshots(ctx, request)
}

// HealthCheck checks the health of the Context Gateway service
func (c *ContextGatewayService) HealthCheck(ctx context.Context) (*clients.ServiceHealth, error) {
	return c.contextClient.HealthCheck(ctx)
}

// validateSnapshotRequest validates the snapshot creation request
func (c *ContextGatewayService) validateSnapshotRequest(request *SnapshotCreationRequest) error {
	if request.PatientID == uuid.Nil {
		return fmt.Errorf("patient_id is required")
	}

	if request.RecipeID == uuid.Nil {
		return fmt.Errorf("recipe_id is required")
	}

	if request.RecipeResolution == nil {
		return fmt.Errorf("recipe_resolution is required")
	}

	if request.SnapshotType == "" {
		request.SnapshotType = "calculation" // Default type
	}

	if request.Priority == "" {
		request.Priority = "normal" // Default priority
	}

	if request.CreatedBy == "" {
		request.CreatedBy = "medication-service-v2" // Default creator
	}

	return nil
}

// isRetryableError determines if an error is retryable
func (c *ContextGatewayService) isRetryableError(err error) bool {
	// Consider network errors, timeout errors, and 5xx HTTP errors as retryable
	errStr := err.Error()
	
	// Network and timeout errors
	if containsAny(errStr, []string{"timeout", "connection", "network", "dial", "i/o timeout"}) {
		return true
	}
	
	// HTTP 5xx errors
	if containsAny(errStr, []string{"500", "502", "503", "504"}) {
		return true
	}
	
	// Context Gateway specific retryable errors
	if containsAny(errStr, []string{"service unavailable", "internal server error", "gateway timeout"}) {
		return true
	}
	
	return false
}

// containsAny checks if string contains any of the given substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) && s[:len(substr)] == substr {
			return true
		}
	}
	return false
}

// DefaultContextGatewayConfig returns default configuration
func DefaultContextGatewayConfig() ContextGatewayConfig {
	return ContextGatewayConfig{
		DefaultSnapshotTTL:          24 * time.Hour,
		FreshnessRequirements: map[string]time.Duration{
			"demographics":      7 * 24 * time.Hour,  // 7 days
			"vital_signs":       4 * time.Hour,       // 4 hours
			"lab_results":       24 * time.Hour,      // 24 hours
			"medications":       1 * time.Hour,       // 1 hour
			"allergies":         30 * 24 * time.Hour, // 30 days
			"conditions":        7 * 24 * time.Hour,  // 7 days
		},
		SnapshotCreationTimeout:     30 * time.Second,
		MaxRetries:                  3,
		RetryBackoffMultiplier:      2.0,
		InitialRetryDelay:          1 * time.Second,
		MinRequiredQualityScore:     0.7,
		RequiredFields: []string{
			"demographics", "medications", "allergies",
		},
		OptionalFields: []string{
			"vital_signs", "lab_results", "conditions", "procedures", "observations",
		},
		EnableAsyncSnapshotCreation: false,
		SnapshotCreationWorkers:     5,
		EnableSnapshotValidation:    true,
		ValidationLevel:            "standard",
	}
}