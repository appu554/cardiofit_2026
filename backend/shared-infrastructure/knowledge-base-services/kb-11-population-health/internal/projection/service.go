// Package projection provides the core business logic for KB-11 Population Health Engine.
//
// ARCHITECTURE OVERVIEW:
// KB-11 is a "Population Intelligence Layer" that CONSUMES patient data from
// FHIR Store and KB-17, calculates risk scores (governed by KB-18), and provides
// population-level analytics. It is NOT a patient registry.
//
// North Star: "KB-11 answers population-level questions, NOT patient-level decisions."
package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/config"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Service provides the core projection and population health functionality.
type Service struct {
	repo       *database.ProjectionRepository
	fhirClient *clients.FHIRClient
	kb17Client *clients.KB17Client
	config     *config.Config
	logger     *logrus.Entry

	// Sync management
	syncMu     sync.Mutex
	syncActive map[models.SyncSource]bool
}

// NewService creates a new projection service.
func NewService(
	repo *database.ProjectionRepository,
	fhirClient *clients.FHIRClient,
	kb17Client *clients.KB17Client,
	cfg *config.Config,
	logger *logrus.Entry,
) *Service {
	return &Service{
		repo:       repo,
		fhirClient: fhirClient,
		kb17Client: kb17Client,
		config:     cfg,
		logger:     logger.WithField("service", "projection"),
		syncActive: make(map[models.SyncSource]bool),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Patient Projection Queries (READ-ONLY operations on cached data)
// ──────────────────────────────────────────────────────────────────────────────

// GetPatientByFHIRID retrieves a patient projection by FHIR ID.
func (s *Service) GetPatientByFHIRID(ctx context.Context, fhirID string) (*models.PatientProjection, error) {
	return s.repo.GetByFHIRID(ctx, fhirID)
}

// QueryPatients retrieves patient projections with filtering and pagination.
func (s *Service) QueryPatients(ctx context.Context, req *models.PatientQueryRequest) ([]*models.PatientProjection, int, error) {
	req.SetDefaults()
	return s.repo.Query(ctx, req)
}

// GetPopulationMetrics calculates population-level analytics.
// This is the CORE PURPOSE of KB-11 - answering population questions.
func (s *Service) GetPopulationMetrics(ctx context.Context, req *models.PopulationMetricsRequest) (*models.PopulationMetrics, error) {
	return s.repo.GetPopulationMetrics(ctx, req)
}

// ──────────────────────────────────────────────────────────────────────────────
// Attribution Management (KB-11 OWNS attribution data)
// ──────────────────────────────────────────────────────────────────────────────

// UpdateAttribution updates patient attribution (PCP, practice assignment).
// KB-11 owns this data - it's enrichment on top of synced patient data.
func (s *Service) UpdateAttribution(ctx context.Context, req *models.AttributionUpdateRequest) error {
	return s.repo.UpdateAttribution(ctx, req)
}

// BatchUpdateAttribution updates multiple patient attributions.
func (s *Service) BatchUpdateAttribution(ctx context.Context, req *models.BatchAttributionUpdateRequest) error {
	for _, update := range req.Updates {
		if err := s.repo.UpdateAttribution(ctx, &update); err != nil {
			s.logger.WithError(err).WithField("patient", update.PatientFHIRID).Warn("Failed to update attribution")
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Data Synchronization (READ-ONLY from upstream sources)
// ──────────────────────────────────────────────────────────────────────────────

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	Source        models.SyncSource `json:"source"`
	Status        models.SyncStatus `json:"status"`
	RecordsSynced int               `json:"records_synced"`
	Duration      time.Duration     `json:"duration"`
	Error         string            `json:"error,omitempty"`
}

// SyncFromFHIR synchronizes patient data from FHIR Store.
// IMPORTANT: This is a READ-ONLY operation. We CONSUME data, never write back.
func (s *Service) SyncFromFHIR(ctx context.Context, fullSync bool, maxRecords int) (*SyncResult, error) {
	if !s.acquireSyncLock(models.SyncSourceFHIR) {
		return nil, fmt.Errorf("FHIR sync already in progress")
	}
	defer s.releaseSyncLock(models.SyncSourceFHIR)

	start := time.Now()
	result := &SyncResult{
		Source: models.SyncSourceFHIR,
		Status: models.SyncStatusInProgress,
	}

	// Mark sync as started
	if err := s.repo.StartSync(ctx, models.SyncSourceFHIR); err != nil {
		s.logger.WithError(err).Warn("Failed to mark sync start")
	}

	s.logger.WithFields(logrus.Fields{
		"full_sync":   fullSync,
		"max_records": maxRecords,
	}).Info("Starting FHIR sync")

	// Sync patients in batches
	pageSize := 100
	if maxRecords > 0 && maxRecords < pageSize {
		pageSize = maxRecords
	}

	pageToken := ""
	totalSynced := 0

	for {
		bundle, err := s.fhirClient.SearchPatients(ctx, pageSize, pageToken)
		if err != nil {
			result.Status = models.SyncStatusFailed
			result.Error = err.Error()
			result.Duration = time.Since(start)
			s.repo.UpdateSyncStatus(ctx, models.SyncSourceFHIR, models.SyncStatusFailed, totalSynced, &result.Error)
			return result, err
		}

		// Process patients in parallel with bounded concurrency
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(s.config.Risk.MaxConcurrent)

		for _, entry := range bundle.Entry {
			entry := entry // Capture for closure
			g.Go(func() error {
				patient, ok := entry.Resource.(*clients.FHIRPatient)
				if !ok {
					// Try to unmarshal from map
					data, err := json.Marshal(entry.Resource)
					if err != nil {
						return nil
					}
					patient = &clients.FHIRPatient{}
					if err := json.Unmarshal(data, patient); err != nil {
						return nil
					}
				}

				proj := patient.ToPatientProjection()
				return s.repo.Upsert(gctx, proj)
			})
		}

		if err := g.Wait(); err != nil {
			s.logger.WithError(err).Warn("Error syncing batch")
		}

		totalSynced += len(bundle.Entry)

		// Check if we've hit the limit
		if maxRecords > 0 && totalSynced >= maxRecords {
			break
		}

		// Get next page
		pageToken = bundle.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}

	result.Status = models.SyncStatusSuccess
	result.RecordsSynced = totalSynced
	result.Duration = time.Since(start)

	s.repo.UpdateSyncStatus(ctx, models.SyncSourceFHIR, models.SyncStatusSuccess, totalSynced, nil)

	s.logger.WithFields(logrus.Fields{
		"records_synced": totalSynced,
		"duration":       result.Duration.String(),
	}).Info("FHIR sync completed")

	return result, nil
}

// SyncFromKB17 synchronizes patient data from KB-17 Registry.
// IMPORTANT: This is a READ-ONLY operation. We CONSUME data, never write back.
func (s *Service) SyncFromKB17(ctx context.Context, fullSync bool, maxRecords int) (*SyncResult, error) {
	if !s.acquireSyncLock(models.SyncSourceKB17) {
		return nil, fmt.Errorf("KB-17 sync already in progress")
	}
	defer s.releaseSyncLock(models.SyncSourceKB17)

	start := time.Now()
	result := &SyncResult{
		Source: models.SyncSourceKB17,
		Status: models.SyncStatusInProgress,
	}

	// Mark sync as started
	if err := s.repo.StartSync(ctx, models.SyncSourceKB17); err != nil {
		s.logger.WithError(err).Warn("Failed to mark sync start")
	}

	s.logger.WithFields(logrus.Fields{
		"full_sync":   fullSync,
		"max_records": maxRecords,
	}).Info("Starting KB-17 sync")

	// Sync patients in batches
	pageSize := 100
	if maxRecords > 0 && maxRecords < pageSize {
		pageSize = maxRecords
	}

	offset := 0
	totalSynced := 0

	for {
		list, err := s.kb17Client.ListPatients(ctx, pageSize, offset)
		if err != nil {
			result.Status = models.SyncStatusFailed
			result.Error = err.Error()
			result.Duration = time.Since(start)
			s.repo.UpdateSyncStatus(ctx, models.SyncSourceKB17, models.SyncStatusFailed, totalSynced, &result.Error)
			return result, err
		}

		// Process patients in parallel
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(s.config.Risk.MaxConcurrent)

		for _, patient := range list.Patients {
			patient := patient // Capture for closure
			g.Go(func() error {
				proj := patient.ToPatientProjection()
				return s.repo.Upsert(gctx, proj)
			})
		}

		if err := g.Wait(); err != nil {
			s.logger.WithError(err).Warn("Error syncing batch")
		}

		totalSynced += len(list.Patients)

		// Check if we've hit the limit or no more data
		if maxRecords > 0 && totalSynced >= maxRecords {
			break
		}
		if !list.HasMore {
			break
		}

		offset += pageSize
	}

	result.Status = models.SyncStatusSuccess
	result.RecordsSynced = totalSynced
	result.Duration = time.Since(start)

	s.repo.UpdateSyncStatus(ctx, models.SyncSourceKB17, models.SyncStatusSuccess, totalSynced, nil)

	s.logger.WithFields(logrus.Fields{
		"records_synced": totalSynced,
		"duration":       result.Duration.String(),
	}).Info("KB-17 sync completed")

	return result, nil
}

// GetSyncStatus retrieves the sync status for a source.
func (s *Service) GetSyncStatus(ctx context.Context, source models.SyncSource) (*models.SyncStatusRecord, error) {
	return s.repo.GetSyncStatus(ctx, source)
}

// GetAllSyncStatus retrieves sync status for all sources.
func (s *Service) GetAllSyncStatus(ctx context.Context) ([]*models.SyncStatusRecord, error) {
	sources := []models.SyncSource{
		models.SyncSourceFHIR,
		models.SyncSourceKB17,
		models.SyncSourceKB13,
	}

	statuses := []*models.SyncStatusRecord{}
	for _, source := range sources {
		status, err := s.repo.GetSyncStatus(ctx, source)
		if err != nil {
			return nil, err
		}
		if status != nil {
			statuses = append(statuses, status)
		}
	}

	return statuses, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Sync Lock Management
// ──────────────────────────────────────────────────────────────────────────────

func (s *Service) acquireSyncLock(source models.SyncSource) bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if s.syncActive[source] {
		return false
	}
	s.syncActive[source] = true
	return true
}

func (s *Service) releaseSyncLock(source models.SyncSource) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	delete(s.syncActive, source)
}

// IsSyncActive checks if a sync operation is currently in progress.
func (s *Service) IsSyncActive(source models.SyncSource) bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.syncActive[source]
}
