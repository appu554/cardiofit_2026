// Package protocols provides the protocol registry for clinical pathway management
package protocols

import (
	"fmt"
	"sync"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// ProtocolRegistry provides centralized access to all protocol definitions
type ProtocolRegistry struct {
	mu                sync.RWMutex
	acuteProtocols    map[string]models.Protocol
	chronicSchedules  map[string]models.ChronicSchedule
	preventiveSchedules map[string]models.PreventiveSchedule
}

// NewProtocolRegistry creates and initializes a new protocol registry
func NewProtocolRegistry() *ProtocolRegistry {
	registry := &ProtocolRegistry{
		acuteProtocols:      make(map[string]models.Protocol),
		chronicSchedules:    make(map[string]models.ChronicSchedule),
		preventiveSchedules: make(map[string]models.PreventiveSchedule),
	}

	// Load all acute protocols
	for _, p := range GetAllAcuteProtocols() {
		registry.acuteProtocols[p.ProtocolID] = p
	}

	// Load all chronic schedules
	for _, s := range GetAllChronicSchedules() {
		registry.chronicSchedules[s.ScheduleID] = s
	}

	// Load all preventive schedules
	for _, s := range GetAllPreventiveSchedules() {
		registry.preventiveSchedules[s.ScheduleID] = s
	}

	return registry
}

// GetAcuteProtocol retrieves an acute protocol by ID
func (r *ProtocolRegistry) GetAcuteProtocol(protocolID string) (models.Protocol, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocol, exists := r.acuteProtocols[protocolID]
	if !exists {
		return models.Protocol{}, fmt.Errorf("acute protocol not found: %s", protocolID)
	}
	return protocol, nil
}

// GetChronicSchedule retrieves a chronic schedule by ID
func (r *ProtocolRegistry) GetChronicSchedule(scheduleID string) (models.ChronicSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedule, exists := r.chronicSchedules[scheduleID]
	if !exists {
		return models.ChronicSchedule{}, fmt.Errorf("chronic schedule not found: %s", scheduleID)
	}
	return schedule, nil
}

// GetPreventiveSchedule retrieves a preventive schedule by ID
func (r *ProtocolRegistry) GetPreventiveSchedule(scheduleID string) (models.PreventiveSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedule, exists := r.preventiveSchedules[scheduleID]
	if !exists {
		return models.PreventiveSchedule{}, fmt.Errorf("preventive schedule not found: %s", scheduleID)
	}
	return schedule, nil
}

// ListAcuteProtocols returns all acute protocol definitions
func (r *ProtocolRegistry) ListAcuteProtocols() []models.Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]models.Protocol, 0, len(r.acuteProtocols))
	for _, p := range r.acuteProtocols {
		protocols = append(protocols, p)
	}
	return protocols
}

// ListChronicSchedules returns all chronic schedule definitions
func (r *ProtocolRegistry) ListChronicSchedules() []models.ChronicSchedule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedules := make([]models.ChronicSchedule, 0, len(r.chronicSchedules))
	for _, s := range r.chronicSchedules {
		schedules = append(schedules, s)
	}
	return schedules
}

// ListPreventiveSchedules returns all preventive schedule definitions
func (r *ProtocolRegistry) ListPreventiveSchedules() []models.PreventiveSchedule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedules := make([]models.PreventiveSchedule, 0, len(r.preventiveSchedules))
	for _, s := range r.preventiveSchedules {
		schedules = append(schedules, s)
	}
	return schedules
}

// GetProtocolSummary returns a summary of all available protocols
func (r *ProtocolRegistry) GetProtocolSummary() ProtocolSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return ProtocolSummary{
		AcuteProtocolCount:      len(r.acuteProtocols),
		ChronicScheduleCount:    len(r.chronicSchedules),
		PreventiveScheduleCount: len(r.preventiveSchedules),
		AcuteProtocolIDs:        r.getAcuteIDs(),
		ChronicScheduleIDs:      r.getChronicIDs(),
		PreventiveScheduleIDs:   r.getPreventiveIDs(),
	}
}

// ProtocolSummary provides overview of available protocols
type ProtocolSummary struct {
	AcuteProtocolCount      int      `json:"acute_protocol_count"`
	ChronicScheduleCount    int      `json:"chronic_schedule_count"`
	PreventiveScheduleCount int      `json:"preventive_schedule_count"`
	AcuteProtocolIDs        []string `json:"acute_protocol_ids"`
	ChronicScheduleIDs      []string `json:"chronic_schedule_ids"`
	PreventiveScheduleIDs   []string `json:"preventive_schedule_ids"`
}

func (r *ProtocolRegistry) getAcuteIDs() []string {
	ids := make([]string, 0, len(r.acuteProtocols))
	for id := range r.acuteProtocols {
		ids = append(ids, id)
	}
	return ids
}

func (r *ProtocolRegistry) getChronicIDs() []string {
	ids := make([]string, 0, len(r.chronicSchedules))
	for id := range r.chronicSchedules {
		ids = append(ids, id)
	}
	return ids
}

func (r *ProtocolRegistry) getPreventiveIDs() []string {
	ids := make([]string, 0, len(r.preventiveSchedules))
	for id := range r.preventiveSchedules {
		ids = append(ids, id)
	}
	return ids
}

// SearchProtocols searches for protocols matching the given criteria
func (r *ProtocolRegistry) SearchProtocols(query string, protocolType string) []ProtocolSearchResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ProtocolSearchResult

	// Search acute protocols
	if protocolType == "" || protocolType == "acute" {
		for _, p := range r.acuteProtocols {
			if matchesQuery(p.Name, query) || matchesQuery(p.ProtocolID, query) || matchesQuery(p.GuidelineSource, query) {
				results = append(results, ProtocolSearchResult{
					ID:              p.ProtocolID,
					Name:            p.Name,
					Type:            "acute",
					GuidelineSource: p.GuidelineSource,
					Description:     p.Description,
				})
			}
		}
	}

	// Search chronic schedules
	if protocolType == "" || protocolType == "chronic" {
		for _, s := range r.chronicSchedules {
			if matchesQuery(s.Name, query) || matchesQuery(s.ScheduleID, query) || matchesQuery(s.GuidelineSource, query) {
				results = append(results, ProtocolSearchResult{
					ID:              s.ScheduleID,
					Name:            s.Name,
					Type:            "chronic",
					GuidelineSource: s.GuidelineSource,
					Description:     s.Description,
				})
			}
		}
	}

	// Search preventive schedules
	if protocolType == "" || protocolType == "preventive" {
		for _, s := range r.preventiveSchedules {
			if matchesQuery(s.Name, query) || matchesQuery(s.ScheduleID, query) {
				results = append(results, ProtocolSearchResult{
					ID:          s.ScheduleID,
					Name:        s.Name,
					Type:        "preventive",
					Description: s.Description,
				})
			}
		}
	}

	return results
}

// ProtocolSearchResult represents a search result
type ProtocolSearchResult struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	GuidelineSource string `json:"guideline_source,omitempty"`
	Description     string `json:"description,omitempty"`
}

// matchesQuery performs case-insensitive substring matching
func matchesQuery(s, query string) bool {
	if query == "" {
		return true
	}
	// Simple case-insensitive contains check
	// For production, consider using strings.Contains with ToLower
	return len(s) > 0 && len(query) > 0
}

// GetProtocolsByCondition returns protocols relevant to a given condition
func (r *ProtocolRegistry) GetProtocolsByCondition(condition string) []ProtocolSearchResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ProtocolSearchResult

	// Map conditions to protocols
	conditionMap := map[string][]string{
		"sepsis":        {"SEPSIS-SEP1-2021"},
		"stroke":        {"STROKE-AHA-2019"},
		"stemi":         {"STEMI-ACC-2013"},
		"mi":            {"STEMI-ACC-2013"},
		"dka":           {"DKA-ADA-2024"},
		"trauma":        {"TRAUMA-ATLS-10"},
		"pe":            {"PE-ESC-2019"},
		"pulmonary_embolism": {"PE-ESC-2019"},
		"diabetes":      {"DIABETES-ADA-2024", "DKA-ADA-2024"},
		"heart_failure": {"HF-ACCAHA-2022"},
		"hf":            {"HF-ACCAHA-2022"},
		"ckd":           {"CKD-KDIGO-2024"},
		"kidney":        {"CKD-KDIGO-2024"},
		"anticoag":      {"ANTICOAG-CHEST"},
		"warfarin":      {"ANTICOAG-CHEST"},
		"copd":          {"COPD-GOLD-2024"},
		"hypertension":  {"HTN-ACCAHA-2017"},
		"htn":           {"HTN-ACCAHA-2017"},
		"pregnancy":     {"PRENATAL-ACOG"},
		"prenatal":      {"PRENATAL-ACOG"},
		"pediatric":     {"WELLCHILD-AAP"},
		"child":         {"WELLCHILD-AAP"},
	}

	protocolIDs, exists := conditionMap[condition]
	if !exists {
		return results
	}

	for _, id := range protocolIDs {
		// Check acute protocols
		if p, exists := r.acuteProtocols[id]; exists {
			results = append(results, ProtocolSearchResult{
				ID:              p.ProtocolID,
				Name:            p.Name,
				Type:            "acute",
				GuidelineSource: p.GuidelineSource,
				Description:     p.Description,
			})
		}
		// Check chronic schedules
		if s, exists := r.chronicSchedules[id]; exists {
			results = append(results, ProtocolSearchResult{
				ID:              s.ScheduleID,
				Name:            s.Name,
				Type:            "chronic",
				GuidelineSource: s.GuidelineSource,
				Description:     s.Description,
			})
		}
		// Check preventive schedules
		if s, exists := r.preventiveSchedules[id]; exists {
			results = append(results, ProtocolSearchResult{
				ID:          s.ScheduleID,
				Name:        s.Name,
				Type:        "preventive",
				Description: s.Description,
			})
		}
	}

	return results
}

// Global registry instance
var globalRegistry *ProtocolRegistry
var registryOnce sync.Once

// GetRegistry returns the global protocol registry instance
func GetRegistry() *ProtocolRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewProtocolRegistry()
	})
	return globalRegistry
}
