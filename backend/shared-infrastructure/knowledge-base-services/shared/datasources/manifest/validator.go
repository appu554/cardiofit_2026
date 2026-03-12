// Package manifest provides validation and management for Phase 3b authority manifests.
//
// This package addresses three critical production-hardening gaps:
//   - Gap 1: Dataset Provenance Locking - Tracks version, checksum, and download metadata
//   - Gap 2: Transformation Versioning - Tracks which transform scripts were applied
//   - Gap 3: Capability Exposure - Exposes authority coverage status at runtime
//
// Usage:
//
//	validator, err := manifest.NewValidator("path/to/datasources")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result := validator.Validate()
//	capabilities := validator.GetCapabilities()
package manifest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// MANIFEST SCHEMA
// =============================================================================

// Manifest represents the Phase 3b authority manifest (MANIFEST.yaml)
type Manifest struct {
	SchemaVersion string    `yaml:"schema_version"`
	GeneratedAt   string    `yaml:"generated_at"`
	Environment   string    `yaml:"environment"`
	Phase         string    `yaml:"phase"`
	PhaseName     string    `yaml:"phase_name"`
	Authorities   map[string]AuthorityEntry `yaml:"authorities"`
	Capabilities  CapabilitiesEntry         `yaml:"capabilities"`
	AuditTrail    AuditTrailEntry           `yaml:"audit_trail"`
	SyncHistory   []SyncHistoryEntry        `yaml:"sync_history"`
}

// AuthorityEntry represents a single authority source in the manifest
type AuthorityEntry struct {
	Description     string          `yaml:"description"`
	AuthorityLevel  string          `yaml:"authority_level"`
	LLMPolicy       string          `yaml:"llm_policy"`
	Priority        string          `yaml:"priority"`
	Source          SourceEntry     `yaml:"source"`
	Version         VersionEntry    `yaml:"version"`
	Files           map[string]FileEntry `yaml:"files"`
	FactTypes       []string        `yaml:"fact_types"`
	Coverage        map[string]interface{} `yaml:"coverage"`
	Validation      ValidationEntry `yaml:"validation"`
}

// SourceEntry contains source metadata
type SourceEntry struct {
	Organization    string `yaml:"organization"`
	URL             string `yaml:"url"`
	APIEndpoint     string `yaml:"api_endpoint"`
	License         string `yaml:"license"`
	Citation        string `yaml:"citation"`
	DataFormat      string `yaml:"data_format"`
	UpdateFrequency string `yaml:"update_frequency"`
}

// VersionEntry contains version metadata
type VersionEntry struct {
	Release       string `yaml:"release"`
	DownloadedAt  string `yaml:"downloaded_at"`
	DownloadedBy  string `yaml:"downloaded_by"`
	PublishedAt   string `yaml:"published_at"`
	ImplementedAt string `yaml:"implemented_at"`
	ImplementedBy string `yaml:"implemented_by"`
}

// FileEntry contains file metadata with checksums
type FileEntry struct {
	Filename         string `yaml:"filename"`
	ChecksumSHA256   string `yaml:"checksum_sha256"`
	RecordCount      int    `yaml:"record_count"`
	TransformScript  string `yaml:"transform_script"`
	TransformVersion string `yaml:"transform_version"`
}

// ValidationEntry contains validation rules
type ValidationEntry struct {
	RequiredFields []string `yaml:"required_fields"`
}

// CapabilitiesEntry contains runtime capability flags
type CapabilitiesEntry struct {
	AuthorityCoverage  map[string]bool `yaml:"authority_coverage"`
	FactTypeCoverage   map[string]bool `yaml:"fact_type_coverage"`
	CoverageLevel      string          `yaml:"coverage_level"`
	CoverageWarning    string          `yaml:"coverage_warning"`
}

// AuditTrailEntry contains audit information
type AuditTrailEntry struct {
	LastValidation   string   `yaml:"last_validation"`
	LastValidator    string   `yaml:"last_validator"`
	ValidationStatus string   `yaml:"validation_status"`
	ValidationErrors []string `yaml:"validation_errors"`
}

// SyncHistoryEntry contains sync history
type SyncHistoryEntry struct {
	Timestamp       string `yaml:"timestamp"`
	Authority       string `yaml:"authority"`
	Status          string `yaml:"status"`
	FactsSynced     int    `yaml:"facts_synced"`
	DurationSeconds int    `yaml:"duration_seconds"`
}

// =============================================================================
// VALIDATOR
// =============================================================================

// Validator validates and manages the authority manifest
type Validator struct {
	basePath     string
	manifestPath string
	manifest     *Manifest
	mu           sync.RWMutex
	lastLoaded   time.Time
}

// NewValidator creates a new manifest validator
func NewValidator(basePath string) (*Validator, error) {
	manifestPath := filepath.Join(basePath, "MANIFEST.yaml")

	v := &Validator{
		basePath:     basePath,
		manifestPath: manifestPath,
	}

	if err := v.Load(); err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	return v, nil
}

// Load loads the manifest from disk
func (v *Validator) Load() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := os.ReadFile(v.manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	v.manifest = &manifest
	v.lastLoaded = time.Now()
	return nil
}

// Save saves the manifest to disk
func (v *Validator) Save() error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.manifest == nil {
		return fmt.Errorf("no manifest loaded")
	}

	data, err := yaml.Marshal(v.manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(v.manifestPath, data, 0644)
}

// =============================================================================
// VALIDATION (Gap 1: Provenance Locking)
// =============================================================================

// ValidationResult contains the full validation result
type ValidationResult struct {
	Valid             bool                          `json:"valid"`
	ValidatedAt       time.Time                     `json:"validated_at"`
	ManifestVersion   string                        `json:"manifest_version"`
	AuthorityResults  map[string]AuthorityResult    `json:"authority_results"`
	Errors            []string                      `json:"errors"`
	Warnings          []string                      `json:"warnings"`
}

// AuthorityResult contains validation result for a single authority
type AuthorityResult struct {
	Authority        string   `json:"authority"`
	Valid            bool     `json:"valid"`
	FileExists       bool     `json:"file_exists"`
	ChecksumMatch    bool     `json:"checksum_match"`
	ExpectedChecksum string   `json:"expected_checksum"`
	ActualChecksum   string   `json:"actual_checksum"`
	RecordCount      int      `json:"record_count"`
	Errors           []string `json:"errors"`
}

// Validate validates all authority sources in the manifest
func (v *Validator) Validate(ctx context.Context) *ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := &ValidationResult{
		ValidatedAt:      time.Now(),
		ManifestVersion:  v.manifest.SchemaVersion,
		AuthorityResults: make(map[string]AuthorityResult),
		Errors:           []string{},
		Warnings:         []string{},
	}

	allValid := true

	for name, authority := range v.manifest.Authorities {
		ar := v.validateAuthority(ctx, name, authority)
		result.AuthorityResults[name] = ar

		if !ar.Valid {
			allValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Authority %s validation failed", name))
		}
	}

	result.Valid = allValid

	// Update audit trail
	v.mu.RUnlock()
	v.mu.Lock()
	v.manifest.AuditTrail.LastValidation = time.Now().Format(time.RFC3339)
	v.manifest.AuditTrail.LastValidator = "manifest-validator"
	if allValid {
		v.manifest.AuditTrail.ValidationStatus = "VALID"
		v.manifest.AuditTrail.ValidationErrors = []string{}
	} else {
		v.manifest.AuditTrail.ValidationStatus = "INVALID"
		v.manifest.AuditTrail.ValidationErrors = result.Errors
	}
	v.mu.Unlock()
	v.mu.RLock()

	return result
}

func (v *Validator) validateAuthority(ctx context.Context, name string, authority AuthorityEntry) AuthorityResult {
	result := AuthorityResult{
		Authority: name,
		Valid:     true,
		Errors:    []string{},
	}

	// Validate each file in the authority
	for fileType, file := range authority.Files {
		if file.Filename == "" || file.Filename == "embedded" {
			continue // Skip embedded data
		}

		filePath := filepath.Join(v.basePath, name, file.Filename)

		// Check file exists
		info, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			result.FileExists = false
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("File %s not found for %s", file.Filename, fileType))
			continue
		}
		result.FileExists = true

		// Validate checksum if specified
		if file.ChecksumSHA256 != "" {
			actualChecksum, err := v.computeFileChecksum(filePath)
			if err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to compute checksum for %s: %v", file.Filename, err))
				continue
			}

			result.ExpectedChecksum = file.ChecksumSHA256
			result.ActualChecksum = actualChecksum

			if actualChecksum != file.ChecksumSHA256 {
				result.ChecksumMatch = false
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Checksum mismatch for %s: expected %s, got %s",
					file.Filename, file.ChecksumSHA256, actualChecksum))
			} else {
				result.ChecksumMatch = true
			}
		}

		// Store file size as proxy for record count validation
		result.RecordCount = int(info.Size())
	}

	return result
}

func (v *Validator) computeFileChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// =============================================================================
// CAPABILITY EXPOSURE (Gap 3: Runtime Status)
// =============================================================================

// Capabilities represents the runtime authority capabilities
type Capabilities struct {
	// Authority availability
	Authorities map[string]AuthorityStatus `json:"authorities"`

	// Fact type coverage
	FactTypeCoverage map[string]bool `json:"fact_type_coverage"`

	// Overall status
	CoverageLevel   string `json:"coverage_level"`
	CoverageWarning string `json:"coverage_warning,omitempty"`

	// Timestamps
	LastUpdated time.Time `json:"last_updated"`
}

// AuthorityStatus represents status of a single authority
type AuthorityStatus struct {
	Name           string `json:"name"`
	Available      bool   `json:"available"`
	AuthorityLevel string `json:"authority_level"`
	LLMPolicy      string `json:"llm_policy"`
	SourceVersion  string `json:"source_version,omitempty"`
	LastSync       string `json:"last_sync,omitempty"`
	FactTypes      []string `json:"fact_types"`
}

// GetCapabilities returns the current authority capabilities for runtime exposure
func (v *Validator) GetCapabilities() *Capabilities {
	v.mu.RLock()
	defer v.mu.RUnlock()

	caps := &Capabilities{
		Authorities:      make(map[string]AuthorityStatus),
		FactTypeCoverage: make(map[string]bool),
		LastUpdated:      time.Now(),
	}

	// Build authority status from manifest
	availableCount := 0
	totalCount := len(v.manifest.Authorities)

	for name, authority := range v.manifest.Authorities {
		available := v.manifest.Capabilities.AuthorityCoverage[name]

		status := AuthorityStatus{
			Name:           name,
			Available:      available,
			AuthorityLevel: authority.AuthorityLevel,
			LLMPolicy:      authority.LLMPolicy,
			SourceVersion:  authority.Version.Release,
			LastSync:       authority.Version.DownloadedAt,
			FactTypes:      authority.FactTypes,
		}

		caps.Authorities[name] = status

		if available {
			availableCount++
		}

		// Update fact type coverage
		for _, ft := range authority.FactTypes {
			if available {
				caps.FactTypeCoverage[ft] = true
			} else if _, exists := caps.FactTypeCoverage[ft]; !exists {
				caps.FactTypeCoverage[ft] = false
			}
		}
	}

	// Determine coverage level
	switch {
	case availableCount == totalCount:
		caps.CoverageLevel = "FULL"
	case availableCount >= totalCount/2:
		caps.CoverageLevel = "PARTIAL"
	case availableCount > 0:
		caps.CoverageLevel = "MINIMAL"
	default:
		caps.CoverageLevel = "NONE"
		caps.CoverageWarning = "No authority sources are available. Clinical decision support is limited."
	}

	if caps.CoverageLevel != "FULL" && caps.CoverageWarning == "" {
		caps.CoverageWarning = fmt.Sprintf("Only %d of %d authority sources available. Some fact types may be unavailable.",
			availableCount, totalCount)
	}

	return caps
}

// =============================================================================
// CHECKSUM UPDATES (Gap 1: Auto-compute)
// =============================================================================

// UpdateChecksums computes and updates checksums for all authority files
func (v *Validator) UpdateChecksums(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	for name, authority := range v.manifest.Authorities {
		for fileType, file := range authority.Files {
			if file.Filename == "" || file.Filename == "embedded" {
				continue
			}

			filePath := filepath.Join(v.basePath, name, file.Filename)

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				continue // Skip non-existent files
			}

			checksum, err := v.computeFileChecksum(filePath)
			if err != nil {
				return fmt.Errorf("failed to compute checksum for %s/%s: %w", name, fileType, err)
			}

			// Update the file entry
			file.ChecksumSHA256 = checksum
			authority.Files[fileType] = file
		}
		v.manifest.Authorities[name] = authority
	}

	return nil
}

// =============================================================================
// SYNC HISTORY (Audit Trail)
// =============================================================================

// RecordSync records a sync operation in the manifest
func (v *Validator) RecordSync(authority string, status string, factsSynced int, duration time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()

	entry := SyncHistoryEntry{
		Timestamp:       time.Now().Format(time.RFC3339),
		Authority:       authority,
		Status:          status,
		FactsSynced:     factsSynced,
		DurationSeconds: int(duration.Seconds()),
	}

	// Prepend to history (newest first)
	v.manifest.SyncHistory = append([]SyncHistoryEntry{entry}, v.manifest.SyncHistory...)

	// Keep only last 100 entries
	if len(v.manifest.SyncHistory) > 100 {
		v.manifest.SyncHistory = v.manifest.SyncHistory[:100]
	}

	// Update capability flag
	if status == "SUCCESS" {
		v.manifest.Capabilities.AuthorityCoverage[authority] = true
	}
}

// SetAuthorityAvailable sets the availability of an authority
func (v *Validator) SetAuthorityAvailable(authority string, available bool) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.manifest.Capabilities.AuthorityCoverage[authority] = available
}
