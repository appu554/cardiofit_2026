// Package types provides core snapshot type definitions for the Safety Gateway Platform
package types

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ClinicalSnapshot represents an immutable snapshot of clinical data for safety validation
type ClinicalSnapshot struct {
	SnapshotID       string                 `json:"snapshot_id" validate:"required"`
	PatientID        string                 `json:"patient_id" validate:"required"`
	Data             *ClinicalSnapshotData  `json:"data" validate:"required"`
	CreatedAt        time.Time              `json:"created_at" validate:"required"`
	ExpiresAt        time.Time              `json:"expires_at" validate:"required"`
	Checksum         string                 `json:"checksum" validate:"required"`
	DataCompleteness float64                `json:"data_completeness" validate:"min=0,max=100"`
	AllowLiveFetch   bool                   `json:"allow_live_fetch"`
	AllowedLiveFields []string              `json:"allowed_live_fields"`
	Signature        string                 `json:"signature" validate:"required"`
	Version          string                 `json:"version" validate:"required"`
	Metadata         *SnapshotMetadata      `json:"metadata,omitempty"`
}

// ClinicalSnapshotData contains the actual clinical context data
type ClinicalSnapshotData struct {
	Demographics      *PatientDemographics `json:"demographics,omitempty"`
	ActiveMedications []*Medication        `json:"active_medications,omitempty"`
	Allergies         []*Allergy           `json:"allergies,omitempty"`
	Conditions        []*Condition         `json:"conditions,omitempty"`
	RecentVitals      []*VitalSign         `json:"recent_vitals,omitempty"`
	LabResults        []*LabResult         `json:"lab_results,omitempty"`
	ContextVersion    string               `json:"context_version"`
}

// SnapshotReference provides a compact reference to a clinical snapshot
type SnapshotReference struct {
	SnapshotID       string    `json:"snapshot_id" validate:"required"`
	Checksum         string    `json:"checksum" validate:"required"`
	CreatedAt        time.Time `json:"created_at" validate:"required"`
	DataCompleteness float64   `json:"data_completeness" validate:"min=0,max=100"`
	ExpiresAt        time.Time `json:"expires_at" validate:"required"`
}

// SnapshotMetadata contains additional information about snapshot creation and usage
type SnapshotMetadata struct {
	CreatedBy         string            `json:"created_by,omitempty"`
	CreationReason    string            `json:"creation_reason,omitempty"`
	DataSources       []string          `json:"data_sources,omitempty"`
	ProcessingTimeMs  int64             `json:"processing_time_ms,omitempty"`
	CompressionRatio  float64           `json:"compression_ratio,omitempty"`
	ValidationPassed  bool              `json:"validation_passed"`
	ValidationErrors  []string          `json:"validation_errors,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
}

// SnapshotValidationResult contains the result of snapshot validation
type SnapshotValidationResult struct {
	IsValid          bool                    `json:"is_valid"`
	ValidationTime   time.Time               `json:"validation_time"`
	Errors           []SnapshotValidationError `json:"errors,omitempty"`
	Warnings         []SnapshotValidationWarning `json:"warnings,omitempty"`
	ChecksumValid    bool                    `json:"checksum_valid"`
	SignatureValid   bool                    `json:"signature_valid"`
	ExpirationValid  bool                    `json:"expiration_valid"`
	CompletenessScore float64               `json:"completeness_score"`
}

// SnapshotValidationError represents a critical validation failure
type SnapshotValidationError struct {
	Field       string `json:"field"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Recoverable bool   `json:"recoverable"`
}

// SnapshotValidationWarning represents a non-critical validation issue
type SnapshotValidationWarning struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Impact  string `json:"impact"`
}

// SnapshotUsageReport tracks how snapshots are used for learning and optimization
type SnapshotUsageReport struct {
	SnapshotID      string        `json:"snapshot_id"`
	AccessedAt      time.Time     `json:"accessed_at"`
	AccessDuration  time.Duration `json:"access_duration"`
	CacheHit        bool          `json:"cache_hit"`
	CacheLevel      string        `json:"cache_level"` // "L1", "L2", "Context Gateway"
	EnginesUsed     []string      `json:"engines_used"`
	ValidationTime  time.Duration `json:"validation_time"`
	RetrievalTime   time.Duration `json:"retrieval_time"`
	DataFieldsUsed  []string      `json:"data_fields_used"`
	RequestMetadata map[string]interface{} `json:"request_metadata,omitempty"`
}

// Validation methods

// IsExpired checks if the snapshot has expired
func (cs *ClinicalSnapshot) IsExpired() bool {
	return time.Now().After(cs.ExpiresAt)
}

// IsValid performs basic validation checks on the snapshot
func (cs *ClinicalSnapshot) IsValid() error {
	if cs.SnapshotID == "" {
		return fmt.Errorf("snapshot ID is required")
	}
	
	if cs.PatientID == "" {
		return fmt.Errorf("patient ID is required")
	}
	
	if cs.Data == nil {
		return fmt.Errorf("clinical data is required")
	}
	
	if cs.CreatedAt.IsZero() {
		return fmt.Errorf("creation time is required")
	}
	
	if cs.ExpiresAt.IsZero() || cs.ExpiresAt.Before(cs.CreatedAt) {
		return fmt.Errorf("valid expiration time is required")
	}
	
	if cs.Checksum == "" {
		return fmt.Errorf("checksum is required")
	}
	
	if cs.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	
	if cs.DataCompleteness < 0 || cs.DataCompleteness > 100 {
		return fmt.Errorf("data completeness must be between 0 and 100")
	}
	
	return nil
}

// CalculateChecksum generates a SHA-256 checksum of the snapshot data
func (cs *ClinicalSnapshot) CalculateChecksum() (string, error) {
	dataBytes, err := json.Marshal(cs.Data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snapshot data: %w", err)
	}
	
	// Include key metadata in checksum
	checksumData := struct {
		SnapshotID string                `json:"snapshot_id"`
		PatientID  string                `json:"patient_id"`
		Data       *ClinicalSnapshotData `json:"data"`
		CreatedAt  time.Time             `json:"created_at"`
		ExpiresAt  time.Time             `json:"expires_at"`
		Version    string                `json:"version"`
	}{
		SnapshotID: cs.SnapshotID,
		PatientID:  cs.PatientID,
		Data:       cs.Data,
		CreatedAt:  cs.CreatedAt,
		ExpiresAt:  cs.ExpiresAt,
		Version:    cs.Version,
	}
	
	checksumBytes, err := json.Marshal(checksumData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal checksum data: %w", err)
	}
	
	hash := sha256.Sum256(checksumBytes)
	return hex.EncodeToString(hash[:]), nil
}

// ValidateChecksum verifies the snapshot's checksum matches its data
func (cs *ClinicalSnapshot) ValidateChecksum() error {
	expectedChecksum, err := cs.CalculateChecksum()
	if err != nil {
		return fmt.Errorf("failed to calculate expected checksum: %w", err)
	}
	
	if cs.Checksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, cs.Checksum)
	}
	
	return nil
}

// ValidateSignature verifies the snapshot's HMAC signature
func (cs *ClinicalSnapshot) ValidateSignature(secretKey []byte) error {
	if len(secretKey) == 0 {
		return fmt.Errorf("secret key is required for signature validation")
	}
	
	expectedSignature, err := cs.GenerateSignature(secretKey)
	if err != nil {
		return fmt.Errorf("failed to generate expected signature: %w", err)
	}
	
	if cs.Signature != expectedSignature {
		return fmt.Errorf("signature validation failed")
	}
	
	return nil
}

// GenerateSignature creates an HMAC-SHA256 signature for the snapshot
func (cs *ClinicalSnapshot) GenerateSignature(secretKey []byte) (string, error) {
	// Create signature payload
	signatureData := struct {
		SnapshotID string    `json:"snapshot_id"`
		PatientID  string    `json:"patient_id"`
		CreatedAt  time.Time `json:"created_at"`
		ExpiresAt  time.Time `json:"expires_at"`
		Checksum   string    `json:"checksum"`
		Version    string    `json:"version"`
	}{
		SnapshotID: cs.SnapshotID,
		PatientID:  cs.PatientID,
		CreatedAt:  cs.CreatedAt,
		ExpiresAt:  cs.ExpiresAt,
		Checksum:   cs.Checksum,
		Version:    cs.Version,
	}
	
	signatureBytes, err := json.Marshal(signatureData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal signature data: %w", err)
	}
	
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(signatureBytes)
	signature := mac.Sum(nil)
	
	return hex.EncodeToString(signature), nil
}

// ToReference creates a compact snapshot reference
func (cs *ClinicalSnapshot) ToReference() *SnapshotReference {
	return &SnapshotReference{
		SnapshotID:       cs.SnapshotID,
		Checksum:         cs.Checksum,
		CreatedAt:        cs.CreatedAt,
		DataCompleteness: cs.DataCompleteness,
		ExpiresAt:        cs.ExpiresAt,
	}
}

// GetRequiredFields returns the list of critical fields required for clinical safety
func (cs *ClinicalSnapshot) GetRequiredFields() []string {
	return []string{
		"demographics.age",
		"demographics.gender",
		"active_medications",
		"allergies",
		"conditions",
	}
}

// HasRequiredFields checks if all required clinical fields are present
func (cs *ClinicalSnapshot) HasRequiredFields() (bool, []string) {
	var missingFields []string
	
	if cs.Data == nil {
		return false, []string{"clinical_data"}
	}
	
	if cs.Data.Demographics == nil {
		missingFields = append(missingFields, "demographics")
	} else {
		if cs.Data.Demographics.Age <= 0 {
			missingFields = append(missingFields, "demographics.age")
		}
		if cs.Data.Demographics.Gender == "" {
			missingFields = append(missingFields, "demographics.gender")
		}
	}
	
	if cs.Data.ActiveMedications == nil {
		missingFields = append(missingFields, "active_medications")
	}
	
	if cs.Data.Allergies == nil {
		missingFields = append(missingFields, "allergies")
	}
	
	if cs.Data.Conditions == nil {
		missingFields = append(missingFields, "conditions")
	}
	
	return len(missingFields) == 0, missingFields
}

// CalculateDataCompleteness computes the completeness percentage of the snapshot
func (cs *ClinicalSnapshot) CalculateDataCompleteness() float64 {
	if cs.Data == nil {
		return 0.0
	}
	
	totalFields := 10.0 // Total expected fields for complete snapshot
	presentFields := 0.0
	
	if cs.Data.Demographics != nil {
		presentFields += 2.0 // Demographics counts as 2 fields
	}
	if len(cs.Data.ActiveMedications) > 0 {
		presentFields += 2.0
	}
	if len(cs.Data.Allergies) > 0 {
		presentFields += 2.0
	}
	if len(cs.Data.Conditions) > 0 {
		presentFields += 2.0
	}
	if len(cs.Data.RecentVitals) > 0 {
		presentFields += 1.0
	}
	if len(cs.Data.LabResults) > 0 {
		presentFields += 1.0
	}
	
	return (presentFields / totalFields) * 100.0
}