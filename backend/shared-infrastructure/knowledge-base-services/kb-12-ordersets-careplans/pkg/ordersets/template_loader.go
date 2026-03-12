// Package ordersets provides template loading and management functionality
package ordersets

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kb-12-ordersets-careplans/internal/cache"
	"kb-12-ordersets-careplans/internal/models"
	"gorm.io/gorm"
)

// TemplateLoader manages loading and caching of order set templates
type TemplateLoader struct {
	db           *gorm.DB
	cache        *cache.RedisCache
	templates    map[string]*models.OrderSetTemplate
	mu           sync.RWMutex
	lastRefresh  time.Time
	refreshTTL   time.Duration
}

// NewTemplateLoader creates a new template loader instance
func NewTemplateLoader(db *gorm.DB, redisCache *cache.RedisCache) *TemplateLoader {
	return &TemplateLoader{
		db:          db,
		cache:       redisCache,
		templates:   make(map[string]*models.OrderSetTemplate),
		refreshTTL:  15 * time.Minute,
	}
}

// LoadAllTemplates loads all templates from database or defaults
func (tl *TemplateLoader) LoadAllTemplates(ctx context.Context) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// Try loading from database first
	if tl.db != nil {
		var dbTemplates []models.OrderSetTemplate
		if err := tl.db.WithContext(ctx).Where("active = ?", true).Find(&dbTemplates).Error; err == nil && len(dbTemplates) > 0 {
			for i := range dbTemplates {
				tl.templates[dbTemplates[i].TemplateID] = &dbTemplates[i]
			}
			tl.lastRefresh = time.Now()
			return nil
		}
	}

	// Fall back to default templates
	return tl.loadDefaultTemplates()
}

// loadDefaultTemplates loads all hardcoded default templates
func (tl *TemplateLoader) loadDefaultTemplates() error {
	// Load Admission Order Sets (15 templates across 7 files)
	admissionTemplates := GetAllAdmissionOrderSets()
	for _, t := range admissionTemplates {
		tl.templates[t.TemplateID] = t
	}

	// Load Procedure Order Sets (10 templates across 4 files)
	procedureTemplates := GetAllProcedureOrderSets()
	for _, t := range procedureTemplates {
		tl.templates[t.TemplateID] = t
	}

	// Load Emergency Protocols (4 templates)
	emergencyTemplates := GetAllEmergencyProtocols()
	for _, t := range emergencyTemplates {
		tl.templates[t.TemplateID] = t
	}

	tl.lastRefresh = time.Now()
	return nil
}

// GetTemplate retrieves a template by ID
func (tl *TemplateLoader) GetTemplate(ctx context.Context, templateID string) (*models.OrderSetTemplate, error) {
	// Lazy-load templates if not yet initialized
	tl.mu.RLock()
	isEmpty := len(tl.templates) == 0
	tl.mu.RUnlock()
	if isEmpty {
		if err := tl.LoadAllTemplates(ctx); err != nil {
			return nil, fmt.Errorf("failed to load templates: %w", err)
		}
	}

	// Check cache first
	if tl.cache != nil {
		cacheKey := fmt.Sprintf("orderset:template:%s", templateID)
		cached, err := tl.cache.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var template models.OrderSetTemplate
			if err := json.Unmarshal([]byte(cached), &template); err == nil {
				return &template, nil
			}
		}
	}

	// Check in-memory templates
	tl.mu.RLock()
	template, exists := tl.templates[templateID]
	tl.mu.RUnlock()

	if exists {
		// Cache the result
		if tl.cache != nil {
			cacheKey := fmt.Sprintf("orderset:template:%s", templateID)
			if data, err := json.Marshal(template); err == nil {
				_ = tl.cache.Set(ctx, cacheKey, string(data), 30*time.Minute)
			}
		}
		return template, nil
	}

	// Try database
	if tl.db != nil {
		var dbTemplate models.OrderSetTemplate
		if err := tl.db.WithContext(ctx).Where("template_id = ? AND active = ?", templateID, true).First(&dbTemplate).Error; err == nil {
			tl.mu.Lock()
			tl.templates[templateID] = &dbTemplate
			tl.mu.Unlock()
			return &dbTemplate, nil
		}
	}

	return nil, fmt.Errorf("template not found: %s", templateID)
}

// GetTemplatesByCategory retrieves all templates of a specific category
func (tl *TemplateLoader) GetTemplatesByCategory(ctx context.Context, category models.OrderSetCategory) ([]*models.OrderSetTemplate, error) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	var result []*models.OrderSetTemplate
	for _, t := range tl.templates {
		if t.Category == category {
			result = append(result, t)
		}
	}

	return result, nil
}

// GetAllTemplates returns all loaded templates
func (tl *TemplateLoader) GetAllTemplates(ctx context.Context) ([]*models.OrderSetTemplate, error) {
	// Refresh if stale
	if time.Since(tl.lastRefresh) > tl.refreshTTL {
		if err := tl.LoadAllTemplates(ctx); err != nil {
			// Continue with stale data if refresh fails
		}
	}

	tl.mu.RLock()
	defer tl.mu.RUnlock()

	result := make([]*models.OrderSetTemplate, 0, len(tl.templates))
	for _, t := range tl.templates {
		result = append(result, t)
	}

	return result, nil
}

// SearchTemplates searches templates by name or description
func (tl *TemplateLoader) SearchTemplates(ctx context.Context, query string) ([]*models.OrderSetTemplate, error) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	var results []*models.OrderSetTemplate
	queryLower := toLower(query)

	for _, t := range tl.templates {
		if containsIgnoreCase(t.Name, queryLower) || containsIgnoreCase(t.Description, queryLower) {
			results = append(results, t)
		}
	}

	return results, nil
}

// GetTimeCriticalTemplates returns templates with time constraints
func (tl *TemplateLoader) GetTimeCriticalTemplates(ctx context.Context) ([]*models.OrderSetTemplate, error) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	var results []*models.OrderSetTemplate
	for _, t := range tl.templates {
		if t.IsTimeCritical() {
			results = append(results, t)
		}
	}

	return results, nil
}

// SaveTemplate saves or updates a template in the database
func (tl *TemplateLoader) SaveTemplate(ctx context.Context, template *models.OrderSetTemplate) error {
	if tl.db == nil {
		return fmt.Errorf("database not configured")
	}

	// Upsert the template
	result := tl.db.WithContext(ctx).Save(template)
	if result.Error != nil {
		return result.Error
	}

	// Update in-memory cache
	tl.mu.Lock()
	tl.templates[template.TemplateID] = template
	tl.mu.Unlock()

	// Invalidate Redis cache
	if tl.cache != nil {
		cacheKey := fmt.Sprintf("orderset:template:%s", template.TemplateID)
		_ = tl.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// DeleteTemplate deactivates a template (soft delete)
func (tl *TemplateLoader) DeleteTemplate(ctx context.Context, templateID string) error {
	if tl.db == nil {
		return fmt.Errorf("database not configured")
	}

	// Soft delete by setting active = false
	result := tl.db.WithContext(ctx).Model(&models.OrderSetTemplate{}).
		Where("template_id = ?", templateID).
		Update("active", false)
	if result.Error != nil {
		return result.Error
	}

	// Remove from in-memory cache
	tl.mu.Lock()
	delete(tl.templates, templateID)
	tl.mu.Unlock()

	// Invalidate Redis cache
	if tl.cache != nil {
		cacheKey := fmt.Sprintf("orderset:template:%s", templateID)
		_ = tl.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// SeedDatabase seeds the database with all default templates
func (tl *TemplateLoader) SeedDatabase(ctx context.Context) error {
	if tl.db == nil {
		return fmt.Errorf("database not configured")
	}

	// Load default templates
	if err := tl.loadDefaultTemplates(); err != nil {
		return err
	}

	// Insert all templates
	tl.mu.RLock()
	templates := make([]*models.OrderSetTemplate, 0, len(tl.templates))
	for _, t := range tl.templates {
		templates = append(templates, t)
	}
	tl.mu.RUnlock()

	for _, t := range templates {
		// Use upsert to avoid duplicates
		result := tl.db.WithContext(ctx).
			Where("template_id = ?", t.TemplateID).
			Assign(t).
			FirstOrCreate(&models.OrderSetTemplate{})
		if result.Error != nil {
			return fmt.Errorf("failed to seed template %s: %w", t.TemplateID, result.Error)
		}
	}

	return nil
}

// GetTemplateStats returns statistics about loaded templates
func (tl *TemplateLoader) GetTemplateStats(ctx context.Context) map[string]interface{} {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	stats := map[string]interface{}{
		"total_templates": len(tl.templates),
		"last_refresh":    tl.lastRefresh,
		"categories":      make(map[string]int),
	}

	categories := stats["categories"].(map[string]int)
	timeCriticalCount := 0

	for _, t := range tl.templates {
		categories[string(t.Category)]++
		if t.IsTimeCritical() {
			timeCriticalCount++
		}
	}

	stats["time_critical_count"] = timeCriticalCount

	return stats
}

// ValidateTemplate validates a template's structure and orders
func (tl *TemplateLoader) ValidateTemplate(template *models.OrderSetTemplate) []string {
	var errors []string

	if template.TemplateID == "" {
		errors = append(errors, "template_id is required")
	}
	if template.Name == "" {
		errors = append(errors, "name is required")
	}
	if template.Category == "" {
		errors = append(errors, "category is required")
	}
	if len(template.Orders) == 0 {
		errors = append(errors, "orders cannot be empty")
	}

	// Validate orders structure
	orders, err := template.GetOrders()
	if err != nil {
		errors = append(errors, fmt.Sprintf("invalid orders JSON: %v", err))
	} else {
		for i, order := range orders {
			if order.OrderID == "" {
				errors = append(errors, fmt.Sprintf("order[%d]: order_id is required", i))
			}
			if order.Name == "" {
				errors = append(errors, fmt.Sprintf("order[%d]: name is required", i))
			}
			if order.OrderType == "" {
				errors = append(errors, fmt.Sprintf("order[%d]: order_type is required", i))
			}
		}
	}

	// Validate time constraints if present
	if template.TimeConstraints != nil {
		constraints, err := template.GetTimeConstraints()
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid time_constraints JSON: %v", err))
		} else {
			for i, c := range constraints {
				if c.ConstraintID == "" {
					errors = append(errors, fmt.Sprintf("constraint[%d]: constraint_id is required", i))
				}
				if c.Deadline == 0 {
					errors = append(errors, fmt.Sprintf("constraint[%d]: deadline is required", i))
				}
			}
		}
	}

	return errors
}

// Helper functions
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	return len(sLower) >= len(substr) && findSubstring(sLower, substr) >= 0
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TemplateRegistry provides a global registry of all template IDs and metadata
type TemplateRegistry struct {
	Admission   []TemplateInfo `json:"admission"`
	Procedure   []TemplateInfo `json:"procedure"`
	Emergency   []TemplateInfo `json:"emergency"`
	CarePlan    []TemplateInfo `json:"care_plan"`
}

// TemplateInfo contains metadata about a template
type TemplateInfo struct {
	TemplateID      string `json:"template_id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	GuidelineSource string `json:"guideline_source"`
	TimeCritical    bool   `json:"time_critical"`
}

// GetTemplateRegistry returns a registry of all available templates
func GetTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		Admission: []TemplateInfo{
			// Cardiac
			{TemplateID: "ADM-CHF-001", Name: "CHF Exacerbation", Category: "cardiac", GuidelineSource: "ACC/AHA Heart Failure Guidelines"},
			{TemplateID: "ADM-MI-001", Name: "Acute MI/STEMI", Category: "cardiac", GuidelineSource: "ACC/AHA STEMI Guidelines", TimeCritical: true},
			{TemplateID: "ADM-CP-001", Name: "Chest Pain Evaluation", Category: "cardiac", GuidelineSource: "ACC/AHA Chest Pain Guidelines"},
			// Respiratory
			{TemplateID: "ADM-PNA-001", Name: "Community-Acquired Pneumonia", Category: "respiratory", GuidelineSource: "IDSA/ATS CAP Guidelines"},
			{TemplateID: "ADM-COPD-001", Name: "COPD Exacerbation", Category: "respiratory", GuidelineSource: "GOLD Guidelines"},
			// Metabolic
			{TemplateID: "ADM-SEP-001", Name: "Sepsis/Septic Shock (SEP-1)", Category: "metabolic", GuidelineSource: "Surviving Sepsis Campaign", TimeCritical: true},
			{TemplateID: "ADM-DKA-001", Name: "Diabetic Ketoacidosis", Category: "metabolic", GuidelineSource: "ADA DKA Guidelines"},
			// Neuro
			{TemplateID: "ADM-STROKE-001", Name: "Acute Ischemic Stroke", Category: "neuro", GuidelineSource: "AHA/ASA Stroke Guidelines", TimeCritical: true},
			{TemplateID: "ADM-SYNC-001", Name: "Syncope Evaluation", Category: "neuro", GuidelineSource: "ACC/AHA Syncope Guidelines"},
			// GI
			{TemplateID: "ADM-GIB-001", Name: "GI Bleeding", Category: "gi", GuidelineSource: "ACG GI Bleeding Guidelines"},
			{TemplateID: "ADM-PANC-001", Name: "Acute Pancreatitis", Category: "gi", GuidelineSource: "ACG Pancreatitis Guidelines"},
			// Infectious
			{TemplateID: "ADM-CELL-001", Name: "Cellulitis", Category: "infectious", GuidelineSource: "IDSA Skin Infection Guidelines"},
			{TemplateID: "ADM-UTI-001", Name: "UTI/Pyelonephritis", Category: "infectious", GuidelineSource: "IDSA UTI Guidelines"},
			// Other
			{TemplateID: "ADM-ETOH-001", Name: "Alcohol Withdrawal (CIWA)", Category: "other", GuidelineSource: "ASAM Guidelines"},
			{TemplateID: "ADM-GEN-001", Name: "General Medicine Admission", Category: "other", GuidelineSource: "SHM Hospital Medicine Guidelines"},
		},
		Procedure: []TemplateInfo{
			// Surgical
			{TemplateID: "PROC-PREOP-001", Name: "Pre-Operative Assessment", Category: "surgical", GuidelineSource: "ASA Pre-Anesthesia Guidelines"},
			{TemplateID: "PROC-POSTOP-001", Name: "Post-Operative Care", Category: "surgical", GuidelineSource: "ERAS Guidelines"},
			{TemplateID: "PROC-SURGADM-001", Name: "Surgical Admission", Category: "surgical", GuidelineSource: "ACS NSQIP Guidelines"},
			// GI
			{TemplateID: "PROC-COLON-001", Name: "Colonoscopy Preparation", Category: "gi", GuidelineSource: "ASGE Guidelines"},
			{TemplateID: "PROC-EGD-001", Name: "Upper Endoscopy (EGD)", Category: "gi", GuidelineSource: "ASGE Guidelines"},
			// Cardiac
			{TemplateID: "PROC-CATH-001", Name: "Cardiac Catheterization/PCI", Category: "cardiac", GuidelineSource: "ACC/AHA/SCAI Guidelines", TimeCritical: true},
			{TemplateID: "PROC-DCCV-001", Name: "Elective Cardioversion", Category: "cardiac", GuidelineSource: "ACC/AHA/HRS AF Guidelines"},
			// Bedside
			{TemplateID: "PROC-LP-001", Name: "Lumbar Puncture", Category: "bedside", GuidelineSource: "IDSA Meningitis Guidelines"},
			{TemplateID: "PROC-PARA-001", Name: "Paracentesis", Category: "bedside", GuidelineSource: "AASLD Ascites Guidelines"},
			{TemplateID: "PROC-THORA-001", Name: "Thoracentesis", Category: "bedside", GuidelineSource: "BTS Pleural Guidelines"},
		},
		Emergency: []TemplateInfo{
			{TemplateID: "EMERG-CODE-001", Name: "Code Blue - Cardiac Arrest", Category: "emergency", GuidelineSource: "AHA ACLS Guidelines", TimeCritical: true},
			{TemplateID: "EMERG-RRT-001", Name: "Rapid Response Team", Category: "emergency", GuidelineSource: "IHI RRT Guidelines", TimeCritical: true},
			{TemplateID: "EMERG-MH-001", Name: "Malignant Hyperthermia", Category: "emergency", GuidelineSource: "MHAUS Guidelines", TimeCritical: true},
			{TemplateID: "EMERG-ANAPH-001", Name: "Anaphylaxis", Category: "emergency", GuidelineSource: "WAO/AAAAI Guidelines", TimeCritical: true},
		},
		CarePlan: []TemplateInfo{
			// Will be populated when care plans are implemented
		},
	}

	return registry
}

// GetTemplateCount returns the total count of available templates
func GetTemplateCount() map[string]int {
	return map[string]int{
		"admission":  15,
		"procedure":  10,
		"emergency":  4,
		"care_plan":  0, // To be implemented
		"total":      29,
	}
}
