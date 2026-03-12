// Package cache implements the local safety data cache (Phase 5.1).
//
// During titration cycles, Channel B and C read from this cache ONLY.
// Network calls happen during refresh, never during evaluation (SA-06).
//
// Cache entries:
//   - Raw labs per patient (from KB-20)
//   - Active medications per patient (from KB-20)
//   - Current MCU_GATE per patient (from KB-23 Redis)
//   - Dose history per patient (from local store)
//
// Refresh strategy: scheduled (default 60 min) + event-driven (KB-19).
package cache

import (
	"sync"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// RAASChangeRecency tracks the last ACEi/ARB dose change for RAAS tolerance (PG-14).
type RAASChangeRecency struct {
	DaysSinceChange       int
	InitiationOrTitration string // "INITIATION" | "TITRATION" | "NONE"
}

// PatientSafetyData holds all cached safety data for one patient.
type PatientSafetyData struct {
	PatientID         string
	RawLabs           *channel_b.RawPatientData
	ActiveMedications []string
	MCUGate           vt.ChannelAResult
	LastRefresh       time.Time
	HypoWithin7d      bool // from dose history analysis

	// ── HTN co-management cached fields (Wave 1) ──
	RAASChangeRecency *RAASChangeRecency // nil = no recent RAAS change
	SodiumCurrent     *float64           // mEq/L, nil if not available
	PotassiumCurrent  *float64           // mEq/L, nil if not available (redundant with RawLabs but explicit for Channel C)
	SBPCurrent        *float64           // mmHg, nil if not available
	CKDStage          string             // e.g., "3a", "3b", "4", "5"
	Season            string             // SUMMER|MONSOON|WINTER|AUTUMN|UNKNOWN
}

// SafetyCache is a thread-safe in-memory cache of patient safety data.
// Refreshed every RefreshInterval AND on KB-19 events.
type SafetyCache struct {
	mu              sync.RWMutex
	patients        map[string]*PatientSafetyData
	refreshInterval time.Duration
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	RefreshIntervalMinutes int // default: 60
}

// DefaultCacheConfig returns production-safe cache defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		RefreshIntervalMinutes: 60,
	}
}

// NewSafetyCache creates a new safety data cache.
func NewSafetyCache(cfg CacheConfig) *SafetyCache {
	interval := time.Duration(cfg.RefreshIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute
	}
	return &SafetyCache{
		patients:        make(map[string]*PatientSafetyData),
		refreshInterval: interval,
	}
}

// Get retrieves cached safety data for a patient.
// Returns nil if not cached or if cache entry has expired.
func (c *SafetyCache) Get(patientID string) *PatientSafetyData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.patients[patientID]
	if !ok {
		return nil
	}
	if time.Since(entry.LastRefresh) > c.refreshInterval {
		return nil // stale — caller should refresh
	}
	return entry
}

// Put stores or updates safety data for a patient.
func (c *SafetyCache) Put(data *PatientSafetyData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data.LastRefresh = time.Now()
	c.patients[data.PatientID] = data
}

// Invalidate removes a patient's cached data, forcing a refresh on next access.
func (c *SafetyCache) Invalidate(patientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.patients, patientID)
}

// InvalidateAll clears the entire cache.
func (c *SafetyCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.patients = make(map[string]*PatientSafetyData)
}

// IsStale returns true if the patient's cache entry needs refresh.
func (c *SafetyCache) IsStale(patientID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.patients[patientID]
	if !ok {
		return true
	}
	return time.Since(entry.LastRefresh) > c.refreshInterval
}

// Size returns the number of cached patients.
func (c *SafetyCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.patients)
}
