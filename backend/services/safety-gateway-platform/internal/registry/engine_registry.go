package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// EngineRegistry manages in-process safety engines
type EngineRegistry struct {
	engines       map[string]*EngineInfo
	config        *config.Config
	logger        *logger.Logger
	mutex         sync.RWMutex
	healthTicker  *time.Ticker
	stopHealth    chan struct{}
}

// EngineInfo contains information about a registered engine
type EngineInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Instance     types.SafetyEngine     `json:"-"` // In-process instance
	Capabilities []string               `json:"capabilities"`
	Tier         types.CriticalityTier  `json:"tier"`
	Priority     int                    `json:"priority"`
	Timeout      time.Duration          `json:"timeout"`
	Status       types.EngineStatus     `json:"status"`
	LastCheck    time.Time              `json:"last_check"`
	FailureCount int                    `json:"failure_count"`
	Config       types.EngineConfig     `json:"config"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	
	// Phase 2: Snapshot support tracking
	SupportsSnapshot     bool     `json:"supports_snapshot"`
	SnapshotRequirements []string `json:"snapshot_requirements,omitempty"`
}

// NewEngineRegistry creates a new engine registry
func NewEngineRegistry(cfg *config.Config, logger *logger.Logger) *EngineRegistry {
	registry := &EngineRegistry{
		engines:    make(map[string]*EngineInfo),
		config:     cfg,
		logger:     logger,
		stopHealth: make(chan struct{}),
	}

	// Start health monitoring
	registry.startHealthMonitoring()

	return registry
}

// RegisterEngine registers a new safety engine
func (r *EngineRegistry) RegisterEngine(engine types.SafetyEngine, tier types.CriticalityTier, priority int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	engineID := engine.ID()

	// Check if engine is already registered
	if _, exists := r.engines[engineID]; exists {
		return fmt.Errorf("engine %s is already registered", engineID)
	}

	// Get engine configuration
	engineConfig, exists := r.config.Engines[engineID]
	if !exists {
		r.logger.Warn("No configuration found for engine, using defaults", zap.String("engine_id", engineID))
		engineConfig = config.EngineConfig{
			Enabled:      true,
			TimeoutMs:    5000,
			Priority:     priority,
			Tier:         int(tier),
			Capabilities: engine.Capabilities(),
		}
	}

	// Skip if engine is disabled
	if !engineConfig.Enabled {
		r.logger.Info("Engine is disabled, skipping registration", zap.String("engine_id", engineID))
		return nil
	}

	// Initialize engine
	initConfig := types.EngineConfig{
		ID:           engineID,
		Name:         engine.Name(),
		Enabled:      engineConfig.Enabled,
		Timeout:      time.Duration(engineConfig.TimeoutMs) * time.Millisecond,
		Priority:     engineConfig.Priority,
		Tier:         types.CriticalityTier(engineConfig.Tier),
		Capabilities: engineConfig.Capabilities,
	}

	if err := engine.Initialize(initConfig); err != nil {
		return fmt.Errorf("failed to initialize engine %s: %w", engineID, err)
	}

	// Perform initial health check
	if err := engine.HealthCheck(); err != nil {
		r.logger.Error("Engine failed initial health check", zap.String("engine_id", engineID), zap.Error(err))
		return fmt.Errorf("engine %s failed initial health check: %w", engineID, err)
	}

	// Check if engine supports snapshot-based evaluation
	var supportsSnapshot bool
	var snapshotRequirements []string
	if snapshotEngine, ok := engine.(types.SnapshotAwareEngine); ok {
		supportsSnapshot = snapshotEngine.IsSnapshotCompatible()
		snapshotRequirements = snapshotEngine.GetSnapshotRequirements()
	}

	// Create engine info
	info := &EngineInfo{
		ID:           engineID,
		Name:         engine.Name(),
		Instance:     engine,
		Capabilities: engine.Capabilities(),
		Tier:         tier,
		Priority:     engineConfig.Priority,
		Timeout:      time.Duration(engineConfig.TimeoutMs) * time.Millisecond,
		Status:       types.EngineStatusHealthy,
		LastCheck:    time.Now(),
		FailureCount: 0,
		Config:       initConfig,
		Metadata:     make(map[string]interface{}),
		
		// Phase 2: Snapshot support
		SupportsSnapshot:     supportsSnapshot,
		SnapshotRequirements: snapshotRequirements,
	}

	r.engines[engineID] = info

	r.logger.Info("Engine registered successfully",
		zap.String("engine_id", engineID),
		zap.String("engine_name", engine.Name()),
		zap.Int("tier", int(tier)),
		zap.Int("priority", priority),
		zap.Strings("capabilities", engine.Capabilities()),
		zap.Bool("supports_snapshot", supportsSnapshot),
		zap.Strings("snapshot_requirements", snapshotRequirements),
	)

	return nil
}

// UnregisterEngine unregisters a safety engine
func (r *EngineRegistry) UnregisterEngine(engineID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	info, exists := r.engines[engineID]
	if !exists {
		return fmt.Errorf("engine %s is not registered", engineID)
	}

	// Shutdown engine
	if err := info.Instance.Shutdown(); err != nil {
		r.logger.Error("Error shutting down engine", zap.String("engine_id", engineID), zap.Error(err))
	}

	delete(r.engines, engineID)

	r.logger.Info("Engine unregistered", zap.String("engine_id", engineID))

	return nil
}

// GetEnginesForRequest returns engines applicable for a safety request
func (r *EngineRegistry) GetEnginesForRequest(req *types.SafetyRequest) []*EngineInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var selectedEngines []*EngineInfo

	for _, engine := range r.engines {
		if engine.Status == types.EngineStatusHealthy &&
			r.engineSupportsRequest(engine, req) {
			selectedEngines = append(selectedEngines, engine)
		}
	}

	// Sort by tier first (Tier 1 critical), then priority
	sort.Slice(selectedEngines, func(i, j int) bool {
		if selectedEngines[i].Tier != selectedEngines[j].Tier {
			return selectedEngines[i].Tier < selectedEngines[j].Tier
		}
		return selectedEngines[i].Priority > selectedEngines[j].Priority
	})

	return selectedEngines
}

// engineSupportsRequest checks if an engine supports a specific request
func (r *EngineRegistry) engineSupportsRequest(engine *EngineInfo, req *types.SafetyRequest) bool {
	// Check if engine has capabilities for this request type
	requiredCapabilities := r.getRequiredCapabilities(req)
	
	for _, required := range requiredCapabilities {
		hasCapability := false
		for _, capability := range engine.Capabilities {
			if capability == required {
				hasCapability = true
				break
			}
		}
		if hasCapability {
			return true // Engine has at least one required capability
		}
	}

	return false
}

// getRequiredCapabilities determines required capabilities based on request
func (r *EngineRegistry) getRequiredCapabilities(req *types.SafetyRequest) []string {
	var capabilities []string

	switch req.ActionType {
	case "medication_order", "prescription", "medication_administration":
		capabilities = append(capabilities, "drug_interaction", "contraindication", "dosing")
		if len(req.AllergyIDs) > 0 {
			capabilities = append(capabilities, "allergy_check")
		}
	case "procedure_order":
		capabilities = append(capabilities, "clinical_protocol", "contraindication")
	case "lab_order", "diagnostic_order":
		capabilities = append(capabilities, "clinical_protocol", "guideline_compliance")
	case "treatment_plan", "care_plan":
		capabilities = append(capabilities, "clinical_protocol", "guideline_compliance", "contraindication")
	}

	// Always include hard constraints for critical actions
	if req.Priority == "urgent" || req.Priority == "emergency" {
		capabilities = append(capabilities, "hard_constraints", "safety_limits")
	}

	return capabilities
}

// GetEngine returns a specific engine by ID
func (r *EngineRegistry) GetEngine(engineID string) (*EngineInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info, exists := r.engines[engineID]
	if !exists {
		return nil, fmt.Errorf("engine %s not found", engineID)
	}

	return info, nil
}

// GetAllEngines returns all registered engines
func (r *EngineRegistry) GetAllEngines() map[string]*EngineInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Return a copy to prevent external modification
	engines := make(map[string]*EngineInfo)
	for id, info := range r.engines {
		engines[id] = info
	}

	return engines
}

// GetHealthyEngines returns only healthy engines
func (r *EngineRegistry) GetHealthyEngines() map[string]*EngineInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	engines := make(map[string]*EngineInfo)
	for id, info := range r.engines {
		if info.Status == types.EngineStatusHealthy {
			engines[id] = info
		}
	}

	return engines
}

// UpdateEngineStatus updates the status of an engine
func (r *EngineRegistry) UpdateEngineStatus(engineID string, status types.EngineStatus) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if info, exists := r.engines[engineID]; exists {
		oldStatus := info.Status
		info.Status = status
		info.LastCheck = time.Now()

		if status != types.EngineStatusHealthy {
			info.FailureCount++
		} else {
			info.FailureCount = 0
		}

		r.logger.Info("Engine status updated",
			zap.String("engine_id", engineID),
			zap.String("old_status", string(oldStatus)),
			zap.String("new_status", string(status)),
			zap.Int("failure_count", info.FailureCount),
		)
	}
}

// startHealthMonitoring starts the health monitoring routine
func (r *EngineRegistry) startHealthMonitoring() {
	r.healthTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-r.healthTicker.C:
				r.performHealthChecks()
			case <-r.stopHealth:
				return
			}
		}
	}()
}

// performHealthChecks performs health checks on all engines
func (r *EngineRegistry) performHealthChecks() {
	r.mutex.RLock()
	engines := make([]*EngineInfo, 0, len(r.engines))
	for _, info := range r.engines {
		engines = append(engines, info)
	}
	r.mutex.RUnlock()

	for _, info := range engines {
		go r.checkEngineHealth(info)
	}
}

// checkEngineHealth checks the health of a single engine
func (r *EngineRegistry) checkEngineHealth(info *EngineInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Perform health check with timeout
	done := make(chan error, 1)
	go func() {
		done <- info.Instance.HealthCheck()
	}()

	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		err = fmt.Errorf("health check timeout")
	}

	// Update status based on health check result
	if err != nil {
		r.logger.Warn("Engine health check failed", zap.String("engine_id", info.ID), zap.Error(err))
		
		if info.FailureCount >= 3 {
			r.UpdateEngineStatus(info.ID, types.EngineStatusUnhealthy)
		} else {
			r.UpdateEngineStatus(info.ID, types.EngineStatusDegraded)
		}
	} else {
		r.UpdateEngineStatus(info.ID, types.EngineStatusHealthy)
	}
}

// Shutdown shuts down the engine registry
func (r *EngineRegistry) Shutdown() error {
	// Stop health monitoring
	if r.healthTicker != nil {
		r.healthTicker.Stop()
	}
	
	select {
	case r.stopHealth <- struct{}{}:
	default:
	}

	// Shutdown all engines
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for engineID, info := range r.engines {
		if err := info.Instance.Shutdown(); err != nil {
			r.logger.Error("Error shutting down engine", zap.String("engine_id", engineID), zap.Error(err))
		}
	}

	r.logger.Info("Engine registry shut down")
	return nil
}

// GetSnapshotCompatibleEngines returns engines that support snapshot-based evaluation
func (r *EngineRegistry) GetSnapshotCompatibleEngines() []*EngineInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var compatible []*EngineInfo
	for _, engine := range r.engines {
		if engine.Status == types.EngineStatusHealthy && engine.SupportsSnapshot {
			compatible = append(compatible, engine)
		}
	}

	// Sort by tier first (Tier 1 critical), then priority
	sort.Slice(compatible, func(i, j int) bool {
		if compatible[i].Tier != compatible[j].Tier {
			return compatible[i].Tier < compatible[j].Tier
		}
		return compatible[i].Priority > compatible[j].Priority
	})

	return compatible
}

// GetEnginesForRequestWithSnapshot returns engines applicable for a snapshot-based safety request
func (r *EngineRegistry) GetEnginesForRequestWithSnapshot(req *types.SafetyRequest, snapshot *types.ClinicalSnapshot) []*EngineInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var selectedEngines []*EngineInfo

	for _, engine := range r.engines {
		if engine.Status == types.EngineStatusHealthy &&
			engine.SupportsSnapshot &&
			r.engineSupportsRequest(engine, req) &&
			r.engineSupportsSnapshot(engine, snapshot) {
			selectedEngines = append(selectedEngines, engine)
		}
	}

	// Sort by tier first (Tier 1 critical), then priority
	sort.Slice(selectedEngines, func(i, j int) bool {
		if selectedEngines[i].Tier != selectedEngines[j].Tier {
			return selectedEngines[i].Tier < selectedEngines[j].Tier
		}
		return selectedEngines[i].Priority > selectedEngines[j].Priority
	})

	return selectedEngines
}

// engineSupportsSnapshot checks if an engine can work with the provided snapshot
func (r *EngineRegistry) engineSupportsSnapshot(engine *EngineInfo, snapshot *types.ClinicalSnapshot) bool {
	if snapshot.Data == nil {
		return false
	}

	// Check if snapshot contains required fields for this engine
	for _, requirement := range engine.SnapshotRequirements {
		if !r.snapshotHasField(snapshot, requirement) {
			// If snapshot is missing required fields and data completeness is low, reject
			if snapshot.DataCompleteness < 0.8 {
				return false
			}
			// Allow if live fetch is enabled for missing fields
			if !snapshot.AllowLiveFetch || !r.isLiveFetchAllowed(snapshot, requirement) {
				return false
			}
		}
	}

	return true
}

// snapshotHasField checks if snapshot contains a specific field
func (r *EngineRegistry) snapshotHasField(snapshot *types.ClinicalSnapshot, field string) bool {
	if snapshot.Data == nil {
		return false
	}

	switch field {
	case "demographics":
		return snapshot.Data.Demographics != nil
	case "active_medications":
		return len(snapshot.Data.ActiveMedications) > 0
	case "allergies":
		return len(snapshot.Data.Allergies) > 0
	case "conditions":
		return len(snapshot.Data.Conditions) > 0
	case "recent_vitals":
		return len(snapshot.Data.RecentVitals) > 0
	case "lab_results":
		return len(snapshot.Data.LabResults) > 0
	case "recent_encounters":
		return len(snapshot.Data.RecentEncounters) > 0
	}

	return false
}

// isLiveFetchAllowed checks if live fetch is allowed for a specific field
func (r *EngineRegistry) isLiveFetchAllowed(snapshot *types.ClinicalSnapshot, field string) bool {
	if !snapshot.AllowLiveFetch {
		return false
	}

	for _, allowedField := range snapshot.AllowedLiveFields {
		if allowedField == field {
			return true
		}
	}

	return false
}

// GetSnapshotEngineStats returns statistics about snapshot engine support
func (r *EngineRegistry) GetSnapshotEngineStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := map[string]interface{}{}
	totalEngines := len(r.engines)
	snapshotCompatible := 0
	healthySnaphotEngines := 0

	for _, engine := range r.engines {
		if engine.SupportsSnapshot {
			snapshotCompatible++
			if engine.Status == types.EngineStatusHealthy {
				healthySnaphotEngines++
			}
		}
	}

	stats["total_engines"] = totalEngines
	stats["snapshot_compatible_engines"] = snapshotCompatible  
	stats["healthy_snapshot_engines"] = healthySnaphotEngines
	if totalEngines > 0 {
		stats["snapshot_compatibility_rate"] = float64(snapshotCompatible) / float64(totalEngines) * 100
	}

	return stats
}
