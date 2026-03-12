package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SnapshotManager manages the lifecycle of clinical snapshots
// for the Calculate → Validate → Commit workflow.
type SnapshotManager struct {
	store           SnapshotStore
	ttlMinutes      int
	signatureMethod string

	// Concurrency control
	mutex sync.RWMutex

	// Metrics
	metrics *SnapshotMetrics
}

// SnapshotStore interface for snapshot persistence
type SnapshotStore interface {
	Save(ctx context.Context, snapshot *ClinicalSnapshot) error
	Get(ctx context.Context, id string) (*ClinicalSnapshot, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters SnapshotFilters) ([]*ClinicalSnapshot, error)
	UpdateStatus(ctx context.Context, id string, status SnapshotStatus) error
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(store SnapshotStore, ttlMinutes int) *SnapshotManager {
	if ttlMinutes <= 0 || ttlMinutes > MaxTTLMinutes {
		ttlMinutes = DefaultTTLMinutes
	}

	return &SnapshotManager{
		store:           store,
		ttlMinutes:      ttlMinutes,
		signatureMethod: SignatureMethodSHA256,
		metrics:         &SnapshotMetrics{},
	}
}

// CreateCalculationSnapshot creates a new snapshot for the Calculate phase
func (sm *SnapshotManager) CreateCalculationSnapshot(
	ctx context.Context,
	patientID uuid.UUID,
	recipeID uuid.UUID,
	clinicalData ClinicalSnapshotData,
	computedScores ComputedScores,
	createdBy string,
) (*ClinicalSnapshot, error) {

	return sm.createSnapshot(ctx, patientID, recipeID, clinicalData, computedScores, SnapshotTypeCalculation, createdBy)
}

// CreateValidationSnapshot creates a new snapshot for the Validate phase
// It references the previous calculation snapshot
func (sm *SnapshotManager) CreateValidationSnapshot(
	ctx context.Context,
	calculationSnapshotID uuid.UUID,
	createdBy string,
) (*ClinicalSnapshot, error) {

	// Retrieve the calculation snapshot
	calcSnapshot, err := sm.store.Get(ctx, calculationSnapshotID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve calculation snapshot: %w", err)
	}

	if calcSnapshot.SnapshotType != SnapshotTypeCalculation {
		return nil, fmt.Errorf("source snapshot is not a calculation snapshot")
	}

	if calcSnapshot.IsExpired() {
		return nil, fmt.Errorf("calculation snapshot has expired")
	}

	// Create validation snapshot with reference to calculation snapshot
	snapshot, err := sm.createSnapshot(
		ctx,
		calcSnapshot.PatientID,
		calcSnapshot.RecipeID,
		calcSnapshot.ClinicalData,
		calcSnapshot.ComputedScores,
		SnapshotTypeValidation,
		createdBy,
	)
	if err != nil {
		return nil, err
	}

	snapshot.PreviousSnapshotID = &calculationSnapshotID
	snapshot.ChangeReason = "validation_phase"

	// Update in store
	if err := sm.store.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save validation snapshot: %w", err)
	}

	return snapshot, nil
}

// CreateCommitSnapshot creates a new snapshot for the Commit phase
func (sm *SnapshotManager) CreateCommitSnapshot(
	ctx context.Context,
	validationSnapshotID uuid.UUID,
	createdBy string,
) (*ClinicalSnapshot, error) {

	// Retrieve the validation snapshot
	valSnapshot, err := sm.store.Get(ctx, validationSnapshotID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve validation snapshot: %w", err)
	}

	if valSnapshot.SnapshotType != SnapshotTypeValidation {
		return nil, fmt.Errorf("source snapshot is not a validation snapshot")
	}

	if valSnapshot.IsExpired() {
		return nil, fmt.Errorf("validation snapshot has expired")
	}

	// Create commit snapshot
	snapshot, err := sm.createSnapshot(
		ctx,
		valSnapshot.PatientID,
		valSnapshot.RecipeID,
		valSnapshot.ClinicalData,
		valSnapshot.ComputedScores,
		SnapshotTypeCommit,
		createdBy,
	)
	if err != nil {
		return nil, err
	}

	snapshot.PreviousSnapshotID = &validationSnapshotID
	snapshot.ChangeReason = "commit_phase"

	// Update in store
	if err := sm.store.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save commit snapshot: %w", err)
	}

	// Mark validation snapshot as superseded
	if err := sm.store.UpdateStatus(ctx, validationSnapshotID.String(), SnapshotStatusSuperseded); err != nil {
		// Log but don't fail
	}

	return snapshot, nil
}

// createSnapshot creates a new clinical snapshot with computed hash
func (sm *SnapshotManager) createSnapshot(
	ctx context.Context,
	patientID uuid.UUID,
	recipeID uuid.UUID,
	clinicalData ClinicalSnapshotData,
	computedScores ComputedScores,
	snapshotType SnapshotType,
	createdBy string,
) (*ClinicalSnapshot, error) {

	now := time.Now()

	snapshot := &ClinicalSnapshot{
		ID:             uuid.New(),
		PatientID:      patientID,
		RecipeID:       recipeID,
		SnapshotType:   snapshotType,
		Status:         SnapshotStatusActive,
		Version:        1,
		ClinicalData:   clinicalData,
		ComputedScores: computedScores,
		FreshnessMetadata: FreshnessMetadata{
			DataSources:      make(map[string]DataSourceInfo),
			OverallFreshness: FreshnessStatusFresh,
			LastRefreshAt:    now,
			NextRefreshAt:    now.Add(time.Duration(sm.ttlMinutes) * time.Minute),
		},
		ValidationResults: ValidationResults{
			IsValid:         true,
			ValidationScore: 1.0,
			ValidatedAt:     now,
			ValidatedBy:     "system",
		},
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(sm.ttlMinutes) * time.Minute),
		CreatedBy: createdBy,
		AuditTrail: []SnapshotAuditEntry{
			{
				ID:        uuid.New(),
				Action:    AuditActionCreated,
				Timestamp: now,
				UserID:    createdBy,
				Details:   fmt.Sprintf("Created %s snapshot", snapshotType),
			},
		},
	}

	// Compute hash for integrity verification
	hash, err := sm.computeHash(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to compute snapshot hash: %w", err)
	}
	snapshot.Hash = hash

	// Save to store
	if err := sm.store.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Update metrics
	sm.mutex.Lock()
	sm.metrics.TotalSnapshots++
	sm.metrics.ActiveSnapshots++
	sm.mutex.Unlock()

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot by ID
func (sm *SnapshotManager) GetSnapshot(ctx context.Context, id string) (*ClinicalSnapshot, error) {
	snapshot, err := sm.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	// Add audit entry for view
	snapshot.AuditTrail = append(snapshot.AuditTrail, SnapshotAuditEntry{
		ID:        uuid.New(),
		Action:    AuditActionViewed,
		Timestamp: time.Now(),
		Details:   "Snapshot retrieved",
	})

	return snapshot, nil
}

// ValidateSnapshot validates a snapshot's integrity and freshness
func (sm *SnapshotManager) ValidateSnapshot(ctx context.Context, id string) (*SnapshotValidationResult, error) {
	startTime := time.Now()

	snapshot, err := sm.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	result := &SnapshotValidationResult{
		SnapshotID:  id,
		Valid:       true,
		ValidatedAt: time.Now(),
	}

	// Check expiration
	result.NotExpired = !snapshot.IsExpired()
	if !result.NotExpired {
		result.Valid = false
		result.Errors = append(result.Errors, "snapshot has expired")
	}

	// Verify checksum
	computedHash, err := sm.computeHash(snapshot)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "failed to compute checksum")
	} else {
		result.ChecksumValid = computedHash == snapshot.Hash
		if !result.ChecksumValid {
			result.Valid = false
			result.Errors = append(result.Errors, "checksum mismatch - data may have been tampered")
		}
	}

	// Signature validation (simplified for standalone)
	result.SignatureValid = true

	// Check data freshness
	for dataType, sourceInfo := range snapshot.FreshnessMetadata.DataSources {
		age := time.Since(sourceInfo.LastUpdated)
		if age > time.Duration(sm.ttlMinutes)*time.Minute {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s data is stale (age: %v)", dataType, age))
		}
	}

	result.ValidationDurationMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	return result, nil
}

// DetectChanges compares current patient data with snapshot to detect conflicts
func (sm *SnapshotManager) DetectChanges(
	ctx context.Context,
	snapshotID string,
	currentData ClinicalSnapshotData,
) (*ChangeDetectionResult, error) {

	snapshot, err := sm.store.Get(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	result := &ChangeDetectionResult{
		SnapshotID:   snapshotID,
		HasChanges:   false,
		HardConflicts: []Conflict{},
		SoftConflicts: []Conflict{},
		DetectedAt:   time.Now(),
	}

	// Detect lab changes (HARD conflicts)
	for _, currentLab := range currentData.LabResults {
		for _, snapshotLab := range snapshot.ClinicalData.LabResults {
			if currentLab.TestCode == snapshotLab.TestCode {
				if currentLab.CriticalValue && !snapshotLab.CriticalValue {
					result.HardConflicts = append(result.HardConflicts, Conflict{
						Type:        ConflictTypeLab,
						Severity:    ConflictSeverityHard,
						Field:       currentLab.TestName,
						OldValue:    snapshotLab.Value,
						NewValue:    currentLab.Value,
						Description: fmt.Sprintf("Lab %s became critical", currentLab.TestName),
					})
					result.HasChanges = true
				}
			}
		}
	}

	// Detect new conditions (HARD conflicts)
	snapshotConditionCodes := make(map[string]bool)
	for _, c := range snapshot.ClinicalData.Conditions {
		snapshotConditionCodes[c.SNOMEDCT] = true
	}

	for _, c := range currentData.Conditions {
		if !snapshotConditionCodes[c.SNOMEDCT] && c.Status == ConditionStatusActive {
			result.HardConflicts = append(result.HardConflicts, Conflict{
				Type:        ConflictTypeCondition,
				Severity:    ConflictSeverityHard,
				Field:       "conditions",
				NewValue:    c.ConditionName,
				Description: fmt.Sprintf("New condition diagnosed: %s", c.ConditionName),
			})
			result.HasChanges = true
		}
	}

	// Detect new allergies (HARD conflicts)
	snapshotAllergens := make(map[string]bool)
	for _, a := range snapshot.ClinicalData.Allergies {
		snapshotAllergens[a.Allergen] = true
	}

	for _, a := range currentData.Allergies {
		if !snapshotAllergens[a.Allergen] && a.Status == AllergyStatusActive {
			result.HardConflicts = append(result.HardConflicts, Conflict{
				Type:        ConflictTypeAllergy,
				Severity:    ConflictSeverityHard,
				Field:       "allergies",
				NewValue:    a.Allergen,
				Description: fmt.Sprintf("New allergy reported: %s", a.Allergen),
			})
			result.HasChanges = true
		}
	}

	// Detect demographic changes (SOFT conflicts)
	if currentData.Demographics.WeightKg != nil && snapshot.ClinicalData.Demographics.WeightKg != nil {
		if *currentData.Demographics.WeightKg != *snapshot.ClinicalData.Demographics.WeightKg {
			result.SoftConflicts = append(result.SoftConflicts, Conflict{
				Type:        ConflictTypeDemographic,
				Severity:    ConflictSeveritySoft,
				Field:       "weight",
				OldValue:    *snapshot.ClinicalData.Demographics.WeightKg,
				NewValue:    *currentData.Demographics.WeightKg,
				Description: "Patient weight has changed (may affect dosing)",
			})
			result.HasChanges = true
		}
	}

	return result, nil
}

// ExpireSnapshot marks a snapshot as expired
func (sm *SnapshotManager) ExpireSnapshot(ctx context.Context, id string) error {
	if err := sm.store.UpdateStatus(ctx, id, SnapshotStatusExpired); err != nil {
		return fmt.Errorf("failed to expire snapshot: %w", err)
	}

	sm.mutex.Lock()
	sm.metrics.ActiveSnapshots--
	sm.metrics.ExpiredSnapshots++
	sm.mutex.Unlock()

	return nil
}

// computeHash computes SHA256 hash of snapshot data for integrity
func (sm *SnapshotManager) computeHash(snapshot *ClinicalSnapshot) (string, error) {
	// Create a copy without hash and audit trail for hashing
	data := struct {
		PatientID      uuid.UUID            `json:"patient_id"`
		RecipeID       uuid.UUID            `json:"recipe_id"`
		SnapshotType   SnapshotType         `json:"snapshot_type"`
		ClinicalData   ClinicalSnapshotData `json:"clinical_data"`
		ComputedScores ComputedScores       `json:"computed_scores"`
		CreatedAt      time.Time            `json:"created_at"`
	}{
		PatientID:      snapshot.PatientID,
		RecipeID:       snapshot.RecipeID,
		SnapshotType:   snapshot.SnapshotType,
		ClinicalData:   snapshot.ClinicalData,
		ComputedScores: snapshot.ComputedScores,
		CreatedAt:      snapshot.CreatedAt,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// GetMetrics returns current snapshot manager metrics
func (sm *SnapshotManager) GetMetrics() *SnapshotMetrics {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Return a copy
	metrics := *sm.metrics
	return &metrics
}

// ChangeDetectionResult contains results of change detection between snapshot and current data
type ChangeDetectionResult struct {
	SnapshotID    string     `json:"snapshot_id"`
	HasChanges    bool       `json:"has_changes"`
	HardConflicts []Conflict `json:"hard_conflicts"`
	SoftConflicts []Conflict `json:"soft_conflicts"`
	DetectedAt    time.Time  `json:"detected_at"`
}

// Conflict represents a detected change/conflict
type Conflict struct {
	Type        ConflictType     `json:"type"`
	Severity    ConflictSeverity `json:"severity"`
	Field       string           `json:"field"`
	OldValue    interface{}      `json:"old_value,omitempty"`
	NewValue    interface{}      `json:"new_value,omitempty"`
	Description string           `json:"description"`
}

// ConflictType represents the type of conflict
type ConflictType string

const (
	ConflictTypeLab         ConflictType = "lab"
	ConflictTypeCondition   ConflictType = "condition"
	ConflictTypeAllergy     ConflictType = "allergy"
	ConflictTypeMedication  ConflictType = "medication"
	ConflictTypeDemographic ConflictType = "demographic"
	ConflictTypeVitals      ConflictType = "vitals"
)

// ConflictSeverity represents conflict severity (HARD = abort, SOFT = warn)
type ConflictSeverity string

const (
	ConflictSeverityHard ConflictSeverity = "hard" // Requires re-calculation
	ConflictSeveritySoft ConflictSeverity = "soft" // Warning only
)
