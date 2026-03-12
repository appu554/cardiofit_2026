package phase2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/models"
)

// EvidenceEnvelopeManager manages the Evidence Envelope for Phase 2 operations
type EvidenceEnvelopeManager struct {
	knowledgeBroker    KnowledgeBrokerClient
	currentEnvelope    *models.EvidenceEnvelope
	environment        string
	refreshInterval    time.Duration
	
	// Concurrency control
	mutex              sync.RWMutex
	refreshInProgress  bool
	
	// Monitoring
	logger             *logrus.Logger
	metrics            *EvidenceEnvelopeMetrics
	
	// Background refresh
	stopChan           chan struct{}
	refreshTicker      *time.Ticker
}

// EvidenceEnvelopeMetrics tracks Evidence Envelope performance and usage
type EvidenceEnvelopeMetrics struct {
	EnvelopeRefreshCount   int64
	VersionUsageCount      map[string]int64
	LastRefreshDuration    time.Duration
	CacheHitRate          float64
	ValidationFailures    int64
}

// NewEvidenceEnvelopeManager creates a new Evidence Envelope manager
func NewEvidenceEnvelopeManager(
	knowledgeBroker KnowledgeBrokerClient,
	environment string,
	refreshInterval time.Duration,
) *EvidenceEnvelopeManager {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &EvidenceEnvelopeManager{
		knowledgeBroker: knowledgeBroker,
		environment:     environment,
		refreshInterval: refreshInterval,
		logger:          logger,
		metrics: &EvidenceEnvelopeMetrics{
			VersionUsageCount: make(map[string]int64),
		},
		stopChan: make(chan struct{}),
	}
}

// Initialize initializes the Evidence Envelope manager
func (eem *EvidenceEnvelopeManager) Initialize(ctx context.Context) error {
	eem.logger.WithField("environment", eem.environment).Info("Initializing Evidence Envelope Manager")

	// Fetch initial active version set
	if err := eem.RefreshEnvelope(ctx); err != nil {
		return fmt.Errorf("failed to initialize evidence envelope: %w", err)
	}

	// Start background refresh if interval is configured
	if eem.refreshInterval > 0 {
		eem.startBackgroundRefresh()
	}

	eem.logger.Info("✅ Evidence Envelope Manager initialized successfully")
	return nil
}

// RefreshEnvelope fetches the latest active version set and updates the envelope
func (eem *EvidenceEnvelopeManager) RefreshEnvelope(ctx context.Context) error {
	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	if eem.refreshInProgress {
		return fmt.Errorf("refresh already in progress")
	}

	eem.refreshInProgress = true
	defer func() { eem.refreshInProgress = false }()

	startTime := time.Now()
	eem.logger.Info("🔄 Refreshing Evidence Envelope")

	// Fetch active version set from Knowledge Broker
	activeVersionSet, err := eem.knowledgeBroker.GetActiveVersionSet(ctx, eem.environment)
	if err != nil {
		eem.metrics.ValidationFailures++
		return fmt.Errorf("failed to fetch active version set: %w", err)
	}

	// Validate all KB versions are accessible
	if err := eem.knowledgeBroker.ValidateKBVersions(ctx, activeVersionSet.KBVersions); err != nil {
		eem.metrics.ValidationFailures++
		return fmt.Errorf("KB version validation failed: %w", err)
	}

	// Create new Evidence Envelope
	newEnvelope := &models.EvidenceEnvelope{
		VersionSetName: activeVersionSet.Name,
		KBVersions:     activeVersionSet.KBVersions,
		Environment:    activeVersionSet.Environment,
		ActivatedAt:    activeVersionSet.ActivatedAt,
		UsedVersions:   make(map[string]models.VersionUsage),
	}

	// Generate snapshot ID if not exists
	if newEnvelope.SnapshotID == "" {
		newEnvelope.SnapshotID = fmt.Sprintf("env-%s-%d", eem.environment, time.Now().Unix())
	}

	// Update current envelope
	eem.currentEnvelope = newEnvelope

	// Update metrics
	eem.metrics.EnvelopeRefreshCount++
	eem.metrics.LastRefreshDuration = time.Since(startTime)

	eem.logger.WithFields(logrus.Fields{
		"version_set": newEnvelope.VersionSetName,
		"kb_count":    len(newEnvelope.KBVersions),
		"duration_ms": eem.metrics.LastRefreshDuration.Milliseconds(),
	}).Info("✅ Evidence Envelope refreshed successfully")

	return nil
}

// GetCurrentEnvelope returns the current Evidence Envelope (thread-safe)
func (eem *EvidenceEnvelopeManager) GetCurrentEnvelope() *models.EvidenceEnvelope {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()

	if eem.currentEnvelope == nil {
		return nil
	}

	// Return a copy to prevent external modification
	envelope := *eem.currentEnvelope
	envelope.UsedVersions = make(map[string]models.VersionUsage)
	for k, v := range eem.currentEnvelope.UsedVersions {
		envelope.UsedVersions[k] = v
	}

	return &envelope
}

// RecordVersionUsage records usage of a specific KB version
func (eem *EvidenceEnvelopeManager) RecordVersionUsage(kbName string, cacheHit bool) error {
	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	if eem.currentEnvelope == nil {
		return fmt.Errorf("no active evidence envelope")
	}

	version, ok := eem.currentEnvelope.KBVersions[kbName]
	if !ok {
		return fmt.Errorf("KB %s not found in evidence envelope", kbName)
	}

	// Update usage tracking
	usage := eem.currentEnvelope.UsedVersions[kbName]
	usage.Version = version
	usage.AccessedAt = time.Now()
	usage.QueryCount++
	if cacheHit {
		usage.CacheHits++
	}
	eem.currentEnvelope.UsedVersions[kbName] = usage

	// Update metrics
	eem.metrics.VersionUsageCount[kbName]++

	return nil
}

// GetKBVersion returns the version for a specific knowledge base
func (eem *EvidenceEnvelopeManager) GetKBVersion(kbName string) (string, error) {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()

	if eem.currentEnvelope == nil {
		return "", fmt.Errorf("no active evidence envelope")
	}

	version, ok := eem.currentEnvelope.KBVersions[kbName]
	if !ok {
		return "", fmt.Errorf("KB %s not found in evidence envelope", kbName)
	}

	return version, nil
}

// ValidateEnvelopeConsistency validates that all used versions are still consistent
func (eem *EvidenceEnvelopeManager) ValidateEnvelopeConsistency(ctx context.Context) error {
	eem.mutex.RLock()
	envelope := eem.currentEnvelope
	eem.mutex.RUnlock()

	if envelope == nil {
		return fmt.Errorf("no active evidence envelope")
	}

	eem.logger.Info("🔍 Validating Evidence Envelope consistency")

	// Fetch current active version set
	activeVersionSet, err := eem.knowledgeBroker.GetActiveVersionSet(ctx, eem.environment)
	if err != nil {
		return fmt.Errorf("failed to fetch current active version set: %w", err)
	}

	// Check if our envelope is still current
	if envelope.VersionSetName != activeVersionSet.Name {
		eem.logger.WithFields(logrus.Fields{
			"current_envelope": envelope.VersionSetName,
			"active_version_set": activeVersionSet.Name,
		}).Warn("⚠️ Evidence Envelope is outdated")
		
		return fmt.Errorf("evidence envelope is outdated: current=%s, active=%s", 
			envelope.VersionSetName, activeVersionSet.Name)
	}

	// Validate individual KB versions
	for kbName, currentVersion := range envelope.KBVersions {
		activeVersion, ok := activeVersionSet.KBVersions[kbName]
		if !ok {
			return fmt.Errorf("KB %s no longer exists in active version set", kbName)
		}
		
		if currentVersion != activeVersion {
			eem.logger.WithFields(logrus.Fields{
				"kb_name": kbName,
				"envelope_version": currentVersion,
				"active_version": activeVersion,
			}).Warn("⚠️ KB version mismatch detected")
			
			return fmt.Errorf("KB %s version mismatch: envelope=%s, active=%s", 
				kbName, currentVersion, activeVersion)
		}
	}

	eem.logger.Info("✅ Evidence Envelope consistency validated")
	return nil
}

// GetUsageStatistics returns usage statistics for the current envelope
func (eem *EvidenceEnvelopeManager) GetUsageStatistics() map[string]interface{} {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()

	stats := make(map[string]interface{})
	
	if eem.currentEnvelope != nil {
		stats["version_set_name"] = eem.currentEnvelope.VersionSetName
		stats["kb_count"] = len(eem.currentEnvelope.KBVersions)
		stats["activated_at"] = eem.currentEnvelope.ActivatedAt
		
		// Usage statistics
		totalQueries := int64(0)
		totalCacheHits := int64(0)
		kbUsage := make(map[string]interface{})
		
		for kbName, usage := range eem.currentEnvelope.UsedVersions {
			totalQueries += int64(usage.QueryCount)
			totalCacheHits += int64(usage.CacheHits)
			
			kbUsage[kbName] = map[string]interface{}{
				"version":      usage.Version,
				"query_count":  usage.QueryCount,
				"cache_hits":   usage.CacheHits,
				"last_access":  usage.AccessedAt,
			}
		}
		
		stats["kb_usage"] = kbUsage
		stats["total_queries"] = totalQueries
		stats["total_cache_hits"] = totalCacheHits
		
		if totalQueries > 0 {
			stats["cache_hit_rate"] = float64(totalCacheHits) / float64(totalQueries)
		}
	}

	// Metrics
	stats["metrics"] = map[string]interface{}{
		"refresh_count":        eem.metrics.EnvelopeRefreshCount,
		"last_refresh_duration": eem.metrics.LastRefreshDuration.Milliseconds(),
		"validation_failures":   eem.metrics.ValidationFailures,
	}

	return stats
}

// startBackgroundRefresh starts the background refresh goroutine
func (eem *EvidenceEnvelopeManager) startBackgroundRefresh() {
	eem.refreshTicker = time.NewTicker(eem.refreshInterval)
	
	go func() {
		defer eem.refreshTicker.Stop()
		
		for {
			select {
			case <-eem.refreshTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := eem.RefreshEnvelope(ctx); err != nil {
					eem.logger.WithError(err).Error("❌ Background Evidence Envelope refresh failed")
				}
				cancel()
			case <-eem.stopChan:
				eem.logger.Info("Background Evidence Envelope refresh stopped")
				return
			}
		}
	}()

	eem.logger.WithField("interval", eem.refreshInterval).Info("Started background Evidence Envelope refresh")
}

// Stop stops the Evidence Envelope manager and cleanup resources
func (eem *EvidenceEnvelopeManager) Stop() {
	eem.logger.Info("Stopping Evidence Envelope Manager")
	
	if eem.refreshTicker != nil {
		close(eem.stopChan)
		eem.refreshTicker.Stop()
	}
	
	eem.logger.Info("✅ Evidence Envelope Manager stopped")
}

// IsHealthy returns the health status of the Evidence Envelope manager
func (eem *EvidenceEnvelopeManager) IsHealthy() bool {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()
	
	return eem.currentEnvelope != nil
}

// GetHealthStatus returns detailed health status
func (eem *EvidenceEnvelopeManager) GetHealthStatus() map[string]interface{} {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()
	
	status := map[string]interface{}{
		"healthy": eem.currentEnvelope != nil,
		"environment": eem.environment,
	}
	
	if eem.currentEnvelope != nil {
		status["version_set"] = eem.currentEnvelope.VersionSetName
		status["kb_count"] = len(eem.currentEnvelope.KBVersions)
		status["activated_at"] = eem.currentEnvelope.ActivatedAt
		
		// Check if envelope is getting old
		age := time.Since(eem.currentEnvelope.ActivatedAt)
		status["envelope_age_hours"] = age.Hours()
		
		if age > 24*time.Hour {
			status["warning"] = "Evidence envelope is more than 24 hours old"
		}
	}
	
	return status
}