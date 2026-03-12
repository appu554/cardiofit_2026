package snapshot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// Validator handles snapshot integrity validation
type Validator struct {
	signingKey []byte
	logger     *logger.Logger
}

// NewValidator creates a new snapshot validator
func NewValidator(signingKey []byte, logger *logger.Logger) *Validator {
	return &Validator{
		signingKey: signingKey,
		logger:     logger,
	}
}

// ValidateIntegrity performs comprehensive snapshot validation
func (v *Validator) ValidateIntegrity(snapshot *types.ClinicalSnapshot) *types.SnapshotValidationResult {
	startTime := time.Now()
	result := &types.SnapshotValidationResult{
		Valid:          true,
		ValidationTime: startTime,
		Errors:         []string{},
		Warnings:       []string{},
		Metadata:       make(map[string]interface{}),
	}

	v.logger.Debug("Starting snapshot validation",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("patient_id", snapshot.PatientID),
	)

	// 1. Verify signature
	result.SignatureValid = v.verifySignature(snapshot, result)
	if !result.SignatureValid {
		result.Valid = false
	}

	// 2. Validate checksum
	result.ChecksumValid = v.validateChecksum(snapshot, result)
	if !result.ChecksumValid {
		result.Valid = false
	}

	// 3. Check expiration
	result.NotExpired = v.validateExpiration(snapshot, result)
	if !result.NotExpired {
		result.Valid = false
	}

	// 4. Verify required fields
	result.RequiredFieldsValid = v.validateRequiredFields(snapshot, result)
	if !result.RequiredFieldsValid {
		result.Valid = false
	}

	// 5. Additional validations
	v.validateDataCompleteness(snapshot, result)
	v.validateTemporalConsistency(snapshot, result)

	// Record validation metrics
	duration := time.Since(startTime)
	result.Metadata["validation_duration_ms"] = duration.Milliseconds()
	result.Metadata["validation_checks_performed"] = 6

	v.logger.Debug("Snapshot validation completed",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Bool("valid", result.Valid),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.Int("errors", len(result.Errors)),
		zap.Int("warnings", len(result.Warnings)),
	)

	return result
}

// verifySignature verifies the cryptographic signature of the snapshot
func (v *Validator) verifySignature(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) bool {
	if len(v.signingKey) == 0 {
		result.Warnings = append(result.Warnings, "Signature validation skipped - no signing key configured")
		return true // Consider valid if no key is configured
	}

	if snapshot.Signature == "" {
		result.Errors = append(result.Errors, "Snapshot signature is missing")
		return false
	}

	// Create signature payload (exclude signature field)
	signaturePayload := v.createSignaturePayload(snapshot)
	
	// Generate expected signature
	h := hmac.New(sha256.New, v.signingKey)
	h.Write([]byte(signaturePayload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if snapshot.Signature != expectedSignature {
		result.Errors = append(result.Errors, "Snapshot signature verification failed")
		v.logger.Warn("Snapshot signature mismatch",
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.String("expected", expectedSignature[:16]+"..."),
			zap.String("actual", snapshot.Signature[:16]+"..."),
		)
		return false
	}

	return true
}

// validateChecksum validates the data integrity checksum
func (v *Validator) validateChecksum(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) bool {
	if snapshot.Checksum == "" {
		result.Errors = append(result.Errors, "Snapshot checksum is missing")
		return false
	}

	// Calculate checksum of the clinical data
	expectedChecksum, err := v.calculateDataChecksum(snapshot.Data)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to calculate data checksum: %v", err))
		return false
	}

	if snapshot.Checksum != expectedChecksum {
		result.Errors = append(result.Errors, "Snapshot checksum validation failed")
		v.logger.Warn("Snapshot checksum mismatch",
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.String("expected", expectedChecksum),
			zap.String("actual", snapshot.Checksum),
		)
		return false
	}

	return true
}

// validateExpiration checks if the snapshot has expired
func (v *Validator) validateExpiration(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) bool {
	now := time.Now()
	
	if now.After(snapshot.ExpiresAt) {
		result.Errors = append(result.Errors, fmt.Sprintf("Snapshot expired at %v (now: %v)", 
			snapshot.ExpiresAt, now))
		return false
	}

	// Warning if expiring soon (within 5 minutes)
	if now.Add(5*time.Minute).After(snapshot.ExpiresAt) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Snapshot expires soon at %v", 
			snapshot.ExpiresAt))
	}

	return true
}

// validateRequiredFields ensures all required fields are present and valid
func (v *Validator) validateRequiredFields(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) bool {
	valid := true

	// Validate snapshot-level required fields
	if snapshot.SnapshotID == "" {
		result.Errors = append(result.Errors, "Snapshot ID is required")
		valid = false
	}
	
	if snapshot.PatientID == "" {
		result.Errors = append(result.Errors, "Patient ID is required")
		valid = false
	}
	
	if snapshot.Data == nil {
		result.Errors = append(result.Errors, "Clinical data is required")
		valid = false
		return valid
	}
	
	if snapshot.CreatedAt.IsZero() {
		result.Errors = append(result.Errors, "Creation timestamp is required")
		valid = false
	}
	
	if snapshot.ExpiresAt.IsZero() {
		result.Errors = append(result.Errors, "Expiration timestamp is required")
		valid = false
	}

	// Validate clinical data consistency
	if snapshot.Data.PatientID != snapshot.PatientID {
		result.Errors = append(result.Errors, "Patient ID mismatch between snapshot and clinical data")
		valid = false
	}

	// Check for minimum required clinical data
	if snapshot.Data.Demographics == nil {
		result.Warnings = append(result.Warnings, "Demographics data is missing")
	}

	return valid
}

// validateDataCompleteness checks data completeness percentage
func (v *Validator) validateDataCompleteness(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) {
	if snapshot.DataCompleteness < 0 || snapshot.DataCompleteness > 100 {
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Invalid data completeness value: %f (must be 0-100)", snapshot.DataCompleteness))
		return
	}

	// Warning for low completeness
	if snapshot.DataCompleteness < 50 {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Low data completeness: %.1f%%", snapshot.DataCompleteness))
	}

	result.Metadata["data_completeness"] = snapshot.DataCompleteness
}

// validateTemporalConsistency checks temporal consistency of the snapshot
func (v *Validator) validateTemporalConsistency(snapshot *types.ClinicalSnapshot, result *types.SnapshotValidationResult) {
	// Check creation time is not in the future
	now := time.Now()
	if snapshot.CreatedAt.After(now) {
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Snapshot creation time is in the future: %v", snapshot.CreatedAt))
		return
	}

	// Check expiration is after creation
	if !snapshot.ExpiresAt.After(snapshot.CreatedAt) {
		result.Errors = append(result.Errors, "Snapshot expiration time must be after creation time")
		return
	}

	// Validate clinical data assembly time
	if snapshot.Data != nil && !snapshot.Data.AssemblyTime.IsZero() {
		if snapshot.Data.AssemblyTime.After(snapshot.CreatedAt) {
			result.Warnings = append(result.Warnings, 
				"Clinical data assembly time is after snapshot creation time")
		}
	}
}

// createSignaturePayload creates the payload for signature verification
func (v *Validator) createSignaturePayload(snapshot *types.ClinicalSnapshot) string {
	// Create a signature payload that excludes the signature field
	payload := fmt.Sprintf("%s|%s|%s|%s|%f|%s",
		snapshot.SnapshotID,
		snapshot.PatientID,
		snapshot.Checksum,
		snapshot.CreatedAt.Format(time.RFC3339),
		snapshot.DataCompleteness,
		snapshot.Version,
	)
	
	return payload
}

// calculateDataChecksum calculates SHA256 checksum of clinical data
func (v *Validator) calculateDataChecksum(data *types.ClinicalContext) (string, error) {
	// Serialize clinical data to JSON for consistent checksum calculation
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal clinical data: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(dataBytes)
	return hex.EncodeToString(hash[:]), nil
}

// ValidateReference validates a snapshot reference without fetching the full snapshot
func (v *Validator) ValidateReference(ref *types.SnapshotReference) error {
	if ref.SnapshotID == "" {
		return fmt.Errorf("snapshot ID is required")
	}
	
	if ref.Checksum == "" {
		return fmt.Errorf("checksum is required")
	}
	
	if ref.CreatedAt.IsZero() {
		return fmt.Errorf("creation timestamp is required")
	}
	
	if ref.DataCompleteness < 0 || ref.DataCompleteness > 100 {
		return fmt.Errorf("invalid data completeness: %f", ref.DataCompleteness)
	}
	
	return nil
}

// GetValidationMetrics returns validation performance metrics
func (v *Validator) GetValidationMetrics() map[string]interface{} {
	return map[string]interface{}{
		"signing_key_configured": len(v.signingKey) > 0,
		"validator_version":      "1.0.0",
		"supported_algorithms":   []string{"HMAC-SHA256"},
	}
}