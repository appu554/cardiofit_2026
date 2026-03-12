// Package models provides the rule store for in-memory rule management
package models

import (
	"sort"
	"sync"
	"time"
)

// RuleStore provides thread-safe in-memory storage for rules with multiple indexes
type RuleStore struct {
	rules       map[string]*Rule           // ID -> Rule
	byType      map[string][]*Rule         // Type -> Rules
	byCategory  map[string][]*Rule         // Category -> Rules
	bySeverity  map[string][]*Rule         // Severity -> Rules
	byTag       map[string][]*Rule         // Tag -> Rules
	byStatus    map[string][]*Rule         // Status -> Rules
	allRules    []*Rule                    // Priority-sorted list of all rules
	mu          sync.RWMutex
	lastReload  time.Time
	loadDuration time.Duration
}

// NewRuleStore creates a new rule store
func NewRuleStore() *RuleStore {
	return &RuleStore{
		rules:      make(map[string]*Rule),
		byType:     make(map[string][]*Rule),
		byCategory: make(map[string][]*Rule),
		bySeverity: make(map[string][]*Rule),
		byTag:      make(map[string][]*Rule),
		byStatus:   make(map[string][]*Rule),
		allRules:   make([]*Rule, 0),
	}
}

// Add adds a rule to the store
func (s *RuleStore) Add(rule *Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store by ID
	s.rules[rule.ID] = rule

	// Index by type
	s.byType[rule.Type] = append(s.byType[rule.Type], rule)

	// Index by category
	if rule.Category != "" {
		s.byCategory[rule.Category] = append(s.byCategory[rule.Category], rule)
	}

	// Index by severity
	if rule.Severity != "" {
		s.bySeverity[rule.Severity] = append(s.bySeverity[rule.Severity], rule)
	}

	// Index by status
	if rule.Status != "" {
		s.byStatus[rule.Status] = append(s.byStatus[rule.Status], rule)
	}

	// Index by tags
	for _, tag := range rule.Tags {
		s.byTag[tag] = append(s.byTag[tag], rule)
	}

	// Add to all rules list
	s.allRules = append(s.allRules, rule)
}

// Get retrieves a rule by ID
func (s *RuleStore) Get(id string) (*Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, exists := s.rules[id]
	return rule, exists
}

// GetAll returns all rules sorted by priority
func (s *RuleStore) GetAll() []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Rule, len(s.allRules))
	copy(result, s.allRules)
	return result
}

// GetActive returns all active rules sorted by priority
func (s *RuleStore) GetActive() []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := s.byStatus[StatusActive]
	result := make([]*Rule, len(rules))
	copy(result, rules)
	return result
}

// GetByType returns all rules of a specific type
func (s *RuleStore) GetByType(ruleType string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := s.byType[ruleType]
	result := make([]*Rule, len(rules))
	copy(result, rules)
	return result
}

// GetByCategory returns all rules in a specific category
func (s *RuleStore) GetByCategory(category string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := s.byCategory[category]
	result := make([]*Rule, len(rules))
	copy(result, rules)
	return result
}

// GetBySeverity returns all rules with a specific severity
func (s *RuleStore) GetBySeverity(severity string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := s.bySeverity[severity]
	result := make([]*Rule, len(rules))
	copy(result, rules)
	return result
}

// GetByTags returns all rules that have ALL specified tags
func (s *RuleStore) GetByTags(tags []string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(tags) == 0 {
		return nil
	}

	// Start with rules that have the first tag
	candidates := make(map[string]*Rule)
	for _, rule := range s.byTag[tags[0]] {
		candidates[rule.ID] = rule
	}

	// Filter by remaining tags
	for _, tag := range tags[1:] {
		tagRules := make(map[string]bool)
		for _, rule := range s.byTag[tag] {
			tagRules[rule.ID] = true
		}
		for id := range candidates {
			if !tagRules[id] {
				delete(candidates, id)
			}
		}
	}

	// Convert to slice
	result := make([]*Rule, 0, len(candidates))
	for _, rule := range candidates {
		result = append(result, rule)
	}

	// Sort by priority
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result
}

// GetByAnyTag returns all rules that have ANY of the specified tags
func (s *RuleStore) GetByAnyTag(tags []string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]*Rule, 0)

	for _, tag := range tags {
		for _, rule := range s.byTag[tag] {
			if !seen[rule.ID] {
				seen[rule.ID] = true
				result = append(result, rule)
			}
		}
	}

	// Sort by priority
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result
}

// Remove removes a rule from the store
func (s *RuleStore) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, exists := s.rules[id]
	if !exists {
		return false
	}

	// Remove from main map
	delete(s.rules, id)

	// Remove from type index
	s.byType[rule.Type] = removeFromSlice(s.byType[rule.Type], id)

	// Remove from category index
	if rule.Category != "" {
		s.byCategory[rule.Category] = removeFromSlice(s.byCategory[rule.Category], id)
	}

	// Remove from severity index
	if rule.Severity != "" {
		s.bySeverity[rule.Severity] = removeFromSlice(s.bySeverity[rule.Severity], id)
	}

	// Remove from status index
	if rule.Status != "" {
		s.byStatus[rule.Status] = removeFromSlice(s.byStatus[rule.Status], id)
	}

	// Remove from tag indexes
	for _, tag := range rule.Tags {
		s.byTag[tag] = removeFromSlice(s.byTag[tag], id)
	}

	// Remove from allRules
	s.allRules = removeFromSlice(s.allRules, id)

	return true
}

// Clear removes all rules from the store
func (s *RuleStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules = make(map[string]*Rule)
	s.byType = make(map[string][]*Rule)
	s.byCategory = make(map[string][]*Rule)
	s.bySeverity = make(map[string][]*Rule)
	s.byTag = make(map[string][]*Rule)
	s.byStatus = make(map[string][]*Rule)
	s.allRules = make([]*Rule, 0)
}

// SortByPriority sorts all internal rule lists by priority
func (s *RuleStore) SortByPriority() {
	s.mu.Lock()
	defer s.mu.Unlock()

	sortByPriority := func(rules []*Rule) {
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Priority < rules[j].Priority
		})
	}

	sortByPriority(s.allRules)

	for _, rules := range s.byType {
		sortByPriority(rules)
	}
	for _, rules := range s.byCategory {
		sortByPriority(rules)
	}
	for _, rules := range s.bySeverity {
		sortByPriority(rules)
	}
	for _, rules := range s.byTag {
		sortByPriority(rules)
	}
	for _, rules := range s.byStatus {
		sortByPriority(rules)
	}
}

// SetReloadMetadata sets the reload timestamp and duration
func (s *RuleStore) SetReloadMetadata(loadTime time.Time, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastReload = loadTime
	s.loadDuration = duration
}

// GetStats returns statistics about the rule store
func (s *RuleStore) GetStats() *StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StoreStats{
		TotalRules:      len(s.rules),
		ActiveRules:     len(s.byStatus[StatusActive]),
		RulesByType:     make(map[string]int),
		RulesByCategory: make(map[string]int),
		RulesBySeverity: make(map[string]int),
		LastReloadAt:    s.lastReload,
		LoadDurationMs:  float64(s.loadDuration.Microseconds()) / 1000,
	}

	for t, rules := range s.byType {
		stats.RulesByType[t] = len(rules)
	}
	for c, rules := range s.byCategory {
		stats.RulesByCategory[c] = len(rules)
	}
	for sev, rules := range s.bySeverity {
		stats.RulesBySeverity[sev] = len(rules)
	}

	return stats
}

// Count returns the total number of rules
func (s *RuleStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules)
}

// HasRule checks if a rule exists
func (s *RuleStore) HasRule(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.rules[id]
	return exists
}

// GetTypes returns all rule types in the store
func (s *RuleStore) GetTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, 0, len(s.byType))
	for t := range s.byType {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// GetCategories returns all categories in the store
func (s *RuleStore) GetCategories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]string, 0, len(s.byCategory))
	for c := range s.byCategory {
		categories = append(categories, c)
	}
	sort.Strings(categories)
	return categories
}

// GetTags returns all tags in the store
func (s *RuleStore) GetTags() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tags := make([]string, 0, len(s.byTag))
	for t := range s.byTag {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// removeFromSlice removes a rule by ID from a slice
func removeFromSlice(rules []*Rule, id string) []*Rule {
	result := make([]*Rule, 0, len(rules))
	for _, r := range rules {
		if r.ID != id {
			result = append(result, r)
		}
	}
	return result
}

// Filter represents filter criteria for querying rules
type Filter struct {
	IDs        []string `json:"ids,omitempty"`
	Types      []string `json:"types,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Severities []string `json:"severities,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	TagLogic   string   `json:"tag_logic,omitempty"` // AND or OR
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

// Query returns rules matching the filter criteria
func (s *RuleStore) Query(filter *Filter) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if filter == nil {
		return s.GetAll()
	}

	candidates := make(map[string]*Rule)

	// If specific IDs are provided, start with those
	if len(filter.IDs) > 0 {
		for _, id := range filter.IDs {
			if rule, exists := s.rules[id]; exists {
				candidates[id] = rule
			}
		}
	} else {
		// Start with all rules
		for id, rule := range s.rules {
			candidates[id] = rule
		}
	}

	// Filter by types
	if len(filter.Types) > 0 {
		typeSet := make(map[string]bool)
		for _, t := range filter.Types {
			typeSet[t] = true
		}
		for id, rule := range candidates {
			if !typeSet[rule.Type] {
				delete(candidates, id)
			}
		}
	}

	// Filter by categories
	if len(filter.Categories) > 0 {
		catSet := make(map[string]bool)
		for _, c := range filter.Categories {
			catSet[c] = true
		}
		for id, rule := range candidates {
			if !catSet[rule.Category] {
				delete(candidates, id)
			}
		}
	}

	// Filter by severities
	if len(filter.Severities) > 0 {
		sevSet := make(map[string]bool)
		for _, sev := range filter.Severities {
			sevSet[sev] = true
		}
		for id, rule := range candidates {
			if !sevSet[rule.Severity] {
				delete(candidates, id)
			}
		}
	}

	// Filter by statuses
	if len(filter.Statuses) > 0 {
		statusSet := make(map[string]bool)
		for _, status := range filter.Statuses {
			statusSet[status] = true
		}
		for id, rule := range candidates {
			if !statusSet[rule.Status] {
				delete(candidates, id)
			}
		}
	}

	// Filter by tags
	if len(filter.Tags) > 0 {
		if filter.TagLogic == LogicOR {
			// Rule must have ANY of the tags
			for id, rule := range candidates {
				hasTag := false
				ruleTagSet := make(map[string]bool)
				for _, t := range rule.Tags {
					ruleTagSet[t] = true
				}
				for _, t := range filter.Tags {
					if ruleTagSet[t] {
						hasTag = true
						break
					}
				}
				if !hasTag {
					delete(candidates, id)
				}
			}
		} else {
			// Default: Rule must have ALL of the tags
			for id, rule := range candidates {
				ruleTagSet := make(map[string]bool)
				for _, t := range rule.Tags {
					ruleTagSet[t] = true
				}
				hasAll := true
				for _, t := range filter.Tags {
					if !ruleTagSet[t] {
						hasAll = false
						break
					}
				}
				if !hasAll {
					delete(candidates, id)
				}
			}
		}
	}

	// Convert to slice and sort by priority
	result := make([]*Rule, 0, len(candidates))
	for _, rule := range candidates {
		result = append(result, rule)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []*Rule{}
		}
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result
}
