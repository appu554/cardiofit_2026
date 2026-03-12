package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// IngressValidator handles validation and sanitization of incoming requests
type IngressValidator struct {
	config      *config.Config
	logger      *logger.Logger
	rateLimiter *RateLimiter
	validator   *SchemaValidator
	mutex       sync.RWMutex
}

// NewIngressValidator creates a new ingress validator
func NewIngressValidator(cfg *config.Config, logger *logger.Logger) (*IngressValidator, error) {
	rateLimiter, err := NewRateLimiter(cfg.Security.RateLimiting)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}

	validator, err := NewSchemaValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create schema validator: %w", err)
	}

	return &IngressValidator{
		config:      cfg,
		logger:      logger,
		rateLimiter: rateLimiter,
		validator:   validator,
	}, nil
}

// ValidateRequest validates and sanitizes an incoming safety request
func (v *IngressValidator) ValidateRequest(ctx context.Context, req *types.SafetyRequest) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Size validation
	if err := v.validateRequestSize(req); err != nil {
		v.logger.Warn("Request size validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("error", err.Error()),
		)
		return err
	}

	// Rate limiting
	if err := v.rateLimiter.Allow(req.ClinicianID); err != nil {
		v.logger.Warn("Rate limit exceeded",
			zap.String("request_id", req.RequestID),
			zap.String("clinician_id", req.ClinicianID),
			zap.String("error", err.Error()),
		)
		return err
	}

	// Schema validation
	if err := v.validator.Validate(req); err != nil {
		v.logger.Warn("Schema validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("error", err.Error()),
		)
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Content validation
	if err := v.validateContent(req); err != nil {
		v.logger.Warn("Content validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("error", err.Error()),
		)
		return err
	}

	// Security validation
	if err := v.validateSecurity(req); err != nil {
		v.logger.Warn("Security validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("error", err.Error()),
		)
		return err
	}

	// Sanitize request
	v.sanitizeRequest(req)

	v.logger.Debug("Request validation successful",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("action_type", req.ActionType),
	)

	return nil
}

// validateRequestSize validates the size of the request
func (v *IngressValidator) validateRequestSize(req *types.SafetyRequest) error {
	// Serialize request to estimate size
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize request: %w", err)
	}

	maxSizeBytes := int64(v.config.Performance.MaxRequestSizeMB * 1024 * 1024)
	if int64(len(data)) > maxSizeBytes {
		return fmt.Errorf("request size %d bytes exceeds maximum %d bytes", len(data), maxSizeBytes)
	}

	return nil
}

// validateContent validates the content of the request
func (v *IngressValidator) validateContent(req *types.SafetyRequest) error {
	// Validate required fields
	if req.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	if req.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}

	if req.ClinicianID == "" {
		return fmt.Errorf("clinician_id is required")
	}

	if req.ActionType == "" {
		return fmt.Errorf("action_type is required")
	}

	// Validate action type
	validActionTypes := []string{
		"medication_order", "prescription", "medication_administration",
		"procedure_order", "lab_order", "diagnostic_order",
		"treatment_plan", "care_plan", "discharge_plan",
	}
	if !contains(validActionTypes, req.ActionType) {
		return fmt.Errorf("invalid action_type: %s", req.ActionType)
	}

	// Validate priority
	if req.Priority != "" {
		validPriorities := []string{"low", "normal", "high", "urgent", "emergency"}
		if !contains(validPriorities, req.Priority) {
			return fmt.Errorf("invalid priority: %s", req.Priority)
		}
	}

	// Validate IDs format (basic UUID validation)
	if err := v.validateUUID(req.RequestID, "request_id"); err != nil {
		return err
	}
	if err := v.validateUUID(req.PatientID, "patient_id"); err != nil {
		return err
	}
	if err := v.validateUUID(req.ClinicianID, "clinician_id"); err != nil {
		return err
	}

	// Validate medication IDs (allow medication names, not just UUIDs)
	for i, medID := range req.MedicationIDs {
		if medID == "" {
			return fmt.Errorf("medication_ids[%d] cannot be empty", i)
		}
		// Skip UUID validation for medication IDs - allow medication names
	}

	// Validate condition IDs (allow condition names, not just UUIDs)
	for i, condID := range req.ConditionIDs {
		if condID == "" {
			return fmt.Errorf("condition_ids[%d] cannot be empty", i)
		}
		// Skip UUID validation for condition IDs - allow condition names
	}

	// Validate allergy IDs (allow allergy names, not just UUIDs)
	for i, allergyID := range req.AllergyIDs {
		if allergyID == "" {
			return fmt.Errorf("allergy_ids[%d] cannot be empty", i)
		}
		// Skip UUID validation for allergy IDs - allow allergy names
	}

	// Validate timestamp
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	} else {
		// Ensure timestamp is not too far in the past or future
		now := time.Now()
		if req.Timestamp.Before(now.Add(-24*time.Hour)) || req.Timestamp.After(now.Add(1*time.Hour)) {
			return fmt.Errorf("timestamp is outside acceptable range")
		}
	}

	return nil
}

// validateSecurity performs security validation
func (v *IngressValidator) validateSecurity(req *types.SafetyRequest) error {
	// Check for potential injection attacks in string fields
	dangerousPatterns := []string{
		"<script", "</script>", "javascript:", "vbscript:",
		"onload=", "onerror=", "onclick=", "onmouseover=",
		"eval(", "exec(", "system(", "shell_exec(",
		"DROP TABLE", "DELETE FROM", "INSERT INTO", "UPDATE SET",
		"UNION SELECT", "OR 1=1", "AND 1=1",
	}

	fields := []string{
		req.RequestID, req.PatientID, req.ClinicianID,
		req.ActionType, req.Priority, req.Source,
	}

	for _, field := range fields {
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(field), strings.ToLower(pattern)) {
				return fmt.Errorf("potentially malicious content detected")
			}
		}
	}

	// Validate context map for security
	for key, value := range req.Context {
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(key), strings.ToLower(pattern)) ||
				strings.Contains(strings.ToLower(value), strings.ToLower(pattern)) {
				return fmt.Errorf("potentially malicious content detected in context")
			}
		}
	}

	return nil
}

// sanitizeRequest sanitizes the request content
func (v *IngressValidator) sanitizeRequest(req *types.SafetyRequest) {
	// Trim whitespace from string fields
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.PatientID = strings.TrimSpace(req.PatientID)
	req.ClinicianID = strings.TrimSpace(req.ClinicianID)
	req.ActionType = strings.TrimSpace(req.ActionType)
	req.Priority = strings.TrimSpace(req.Priority)
	req.Source = strings.TrimSpace(req.Source)

	// Sanitize arrays
	req.MedicationIDs = sanitizeStringArray(req.MedicationIDs)
	req.ConditionIDs = sanitizeStringArray(req.ConditionIDs)
	req.AllergyIDs = sanitizeStringArray(req.AllergyIDs)

	// Sanitize context map
	for key, value := range req.Context {
		req.Context[key] = strings.TrimSpace(value)
	}

	// Set default priority if not specified
	if req.Priority == "" {
		req.Priority = "normal"
	}

	// Set source if not specified
	if req.Source == "" {
		req.Source = "unknown"
	}
}

// validateUUID performs basic UUID format validation
func (v *IngressValidator) validateUUID(id, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	// Basic UUID format check (36 characters with hyphens in correct positions)
	if len(id) != 36 {
		return fmt.Errorf("%s must be a valid UUID format", fieldName)
	}

	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return fmt.Errorf("%s must be a valid UUID format", fieldName)
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// sanitizeStringArray removes empty strings and trims whitespace
func sanitizeStringArray(arr []string) []string {
	var result []string
	for _, s := range arr {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
