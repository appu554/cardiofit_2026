// Package services provides business logic for KB-17 Population Registry
package services

import (
	"context"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/cache"
	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/models"
)

// AnalyticsService handles registry statistics and analytics
type AnalyticsService struct {
	repo   *database.Repository
	cache  *cache.RedisCache
	logger *logrus.Entry
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	repo *database.Repository,
	cache *cache.RedisCache,
	logger *logrus.Entry,
) *AnalyticsService {
	return &AnalyticsService{
		repo:   repo,
		cache:  cache,
		logger: logger.WithField("service", "analytics"),
	}
}

// GetRegistryStats retrieves statistics for a specific registry
func (s *AnalyticsService) GetRegistryStats(ctx context.Context, registryCode models.RegistryCode) (*models.RegistryStats, error) {
	s.logger.WithField("registry_code", registryCode).Debug("Getting registry stats")

	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetStats(ctx, registryCode); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get stats from repository
	stats, err := s.repo.GetRegistryStats(registryCode)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetStats(ctx, stats)
	}

	return stats, nil
}

// GetAllRegistryStats retrieves statistics for all registries
func (s *AnalyticsService) GetAllRegistryStats(ctx context.Context) ([]models.RegistryStats, *models.StatsSummary, error) {
	s.logger.Debug("Getting all registry stats")

	registries, err := s.repo.ListRegistries(true) // active only
	if err != nil {
		return nil, nil, err
	}

	allStats := make([]models.RegistryStats, 0, len(registries))
	summary := &models.StatsSummary{
		TotalRegistries: len(registries),
	}

	for _, registry := range registries {
		stats, err := s.GetRegistryStats(ctx, registry.Code)
		if err != nil {
			s.logger.WithError(err).WithField("registry", registry.Code).Warn("Failed to get stats")
			continue
		}

		allStats = append(allStats, *stats)

		// Aggregate to summary
		summary.TotalEnrollments += stats.TotalEnrolled
		summary.ActiveEnrollments += stats.ActiveCount
		summary.HighRiskPatients += stats.HighRiskCount + stats.CriticalCount
		summary.PatientsWithGaps += stats.CareGapCount
	}

	return allStats, summary, nil
}

// GetHighRiskSummary retrieves a summary of high-risk patients
func (s *AnalyticsService) GetHighRiskSummary(ctx context.Context, limit, offset int) (*models.HighRiskResponse, error) {
	s.logger.Debug("Getting high-risk summary")

	patients, total, err := s.repo.GetHighRiskPatients(limit, offset)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.HighRiskPatientSummary, 0, len(patients))
	byTier := make(map[models.RiskTier]int64)

	for _, p := range patients {
		summaries = append(summaries, models.HighRiskPatientSummary{
			PatientID:       p.PatientID,
			RegistryCode:    p.RegistryCode,
			RiskTier:        p.RiskTier,
			CareGapCount:    len(p.CareGaps),
			EnrolledAt:      p.EnrolledAt,
			LastEvaluatedAt: p.LastEvaluatedAt,
		})
		byTier[p.RiskTier]++
	}

	return &models.HighRiskResponse{
		Success: true,
		Data:    summaries,
		Total:   total,
		ByTier:  byTier,
	}, nil
}

// GetCareGapsSummary retrieves care gaps summary
func (s *AnalyticsService) GetCareGapsSummary(ctx context.Context, limit, offset int) (*models.CareGapResponse, error) {
	s.logger.Debug("Getting care gaps summary")

	patients, total, err := s.repo.GetPatientsWithCareGaps(limit, offset)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.CareGapSummary, 0, len(patients))
	byRegistry := make(map[models.RegistryCode]int64)

	for _, p := range patients {
		summaries = append(summaries, models.CareGapSummary{
			PatientID:    p.PatientID,
			RegistryCode: p.RegistryCode,
			CareGaps:     p.CareGaps,
			RiskTier:     p.RiskTier,
			EnrolledAt:   p.EnrolledAt,
		})
		byRegistry[p.RegistryCode]++
	}

	return &models.CareGapResponse{
		Success:    true,
		Data:       summaries,
		Total:      total,
		ByRegistry: byRegistry,
	}, nil
}

// RefreshAllStats forces a refresh of all cached statistics
func (s *AnalyticsService) RefreshAllStats(ctx context.Context) error {
	s.logger.Info("Refreshing all registry statistics")

	registries, err := s.repo.ListRegistries(true)
	if err != nil {
		return err
	}

	for _, registry := range registries {
		// Invalidate cache
		if s.cache != nil {
			_ = s.cache.InvalidateStats(ctx, registry.Code)
		}

		// Recalculate stats
		_, err := s.GetRegistryStats(ctx, registry.Code)
		if err != nil {
			s.logger.WithError(err).WithField("registry", registry.Code).Warn("Failed to refresh stats")
		}
	}

	s.logger.Info("All registry statistics refreshed")
	return nil
}
