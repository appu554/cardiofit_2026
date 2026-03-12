package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-23-decision-cards/internal/models"
)

// TemplateLoader reads CardTemplate YAML definitions from a directory tree,
// validates them against V-04 and N-04 requirements, and provides lookup by
// template_id, differential_id, and node_id. It supports hot-reload via Reload().
type TemplateLoader struct {
	dir       string
	log       *zap.Logger
	mu        sync.RWMutex
	templates map[string]*models.CardTemplate   // keyed by template_id
	byDiff    map[string][]*models.CardTemplate  // keyed by differential_id
	byNode    map[string][]*models.CardTemplate  // keyed by node_id
}

// NewTemplateLoader creates a TemplateLoader that reads from the given directory.
func NewTemplateLoader(dir string, log *zap.Logger) *TemplateLoader {
	return &TemplateLoader{
		dir:       dir,
		log:       log,
		templates: make(map[string]*models.CardTemplate),
		byDiff:    make(map[string][]*models.CardTemplate),
		byNode:    make(map[string][]*models.CardTemplate),
	}
}

// Load walks the templates directory, parsing all *.yaml / *.yml files.
// Invalid templates are logged but do not prevent other templates from loading.
func (l *TemplateLoader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	newTemplates := make(map[string]*models.CardTemplate)
	newByDiff := make(map[string][]*models.CardTemplate)
	newByNode := make(map[string][]*models.CardTemplate)

	err := filepath.Walk(l.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		// Skip vocabulary directory to avoid loading non-template YAML files.
		rel, _ := filepath.Rel(l.dir, path)
		if len(rel) > 10 && rel[:10] == "vocabulary" {
			return nil
		}

		tmpl, parseErr := l.parseTemplate(path)
		if parseErr != nil {
			l.log.Error("failed to parse template",
				zap.String("path", path),
				zap.Error(parseErr),
			)
			return nil // continue loading other templates
		}

		if valErr := l.validateTemplate(tmpl); valErr != nil {
			l.log.Error("template validation failed",
				zap.String("template_id", tmpl.TemplateID),
				zap.Error(valErr),
			)
			return nil
		}

		newTemplates[tmpl.TemplateID] = tmpl
		newByDiff[tmpl.DifferentialID] = append(newByDiff[tmpl.DifferentialID], tmpl)
		newByNode[tmpl.NodeID] = append(newByNode[tmpl.NodeID], tmpl)

		l.log.Debug("loaded template",
			zap.String("template_id", tmpl.TemplateID),
			zap.String("node_id", tmpl.NodeID),
			zap.String("differential_id", tmpl.DifferentialID),
		)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk templates dir: %w", err)
	}

	l.templates = newTemplates
	l.byDiff = newByDiff
	l.byNode = newByNode

	l.log.Info("templates loaded", zap.Int("count", len(l.templates)))
	return nil
}

// parseTemplate reads and unmarshals a single YAML template file, then
// computes derived fields (content hash, safety flags, threshold JSONB).
func (l *TemplateLoader) parseTemplate(path string) (*models.CardTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var tmpl models.CardTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}

	// Compute content hash for change detection.
	hash := sha256.Sum256(data)
	tmpl.ContentSHA256 = fmt.Sprintf("%x", hash)
	tmpl.LoadedAt = time.Now()
	tmpl.RecommendationsCount = len(tmpl.Recommendations)

	// Check for safety instructions among recommendations.
	for _, rec := range tmpl.Recommendations {
		if rec.RecType == models.RecSafetyInstruction {
			tmpl.HasSafetyInstructions = true
			break
		}
	}

	// Check if any gate rule requires dose_adjustment_notes.
	for _, rule := range tmpl.GateRules {
		if rule.Gate == models.GateModify && rule.AdjustmentNotes != "" {
			tmpl.RequiresDoseAdjustmentNotes = true
			break
		}
	}

	// Marshal thresholds to JSONB for database storage.
	emptyThresholds := models.TemplateThresholds{}
	if tmpl.Thresholds != emptyThresholds {
		threshData, marshalErr := json.Marshal(tmpl.Thresholds)
		if marshalErr == nil {
			tmpl.ConfidenceThresholds = models.JSONB(threshData)
		}
	}

	return &tmpl, nil
}

// validateTemplate enforces V-04 and N-04 requirements on a parsed template.
//
// V-04: SAFETY_INSTRUCTION recommendations must have trigger_condition_en and action_text_en.
// N-04: SAFETY_INSTRUCTION fragments must have patient_advocate_reviewed_by and reading_level_validated.
func (l *TemplateLoader) validateTemplate(tmpl *models.CardTemplate) error {
	if tmpl.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}
	if tmpl.NodeID == "" {
		return fmt.Errorf("node_id is required for template %s", tmpl.TemplateID)
	}
	if tmpl.DifferentialID == "" {
		return fmt.Errorf("differential_id is required for template %s", tmpl.TemplateID)
	}

	// V-04: SAFETY_INSTRUCTION recs must have trigger_condition + action_text.
	for i, rec := range tmpl.Recommendations {
		if rec.RecType == models.RecSafetyInstruction {
			if rec.TriggerConditionEn == "" {
				return fmt.Errorf("template %s: SAFETY_INSTRUCTION rec[%d] missing trigger_condition_en", tmpl.TemplateID, i)
			}
			if rec.ActionTextEn == "" {
				return fmt.Errorf("template %s: SAFETY_INSTRUCTION rec[%d] missing action_text_en", tmpl.TemplateID, i)
			}
		}
	}

	// N-04: SAFETY_INSTRUCTION fragments must have patient_advocate_reviewed_by and reading_level_validated.
	for i, frag := range tmpl.Fragments {
		if frag.FragmentType == models.FragSafetyInstruction {
			if frag.PatientAdvocateReviewedBy == "" {
				return fmt.Errorf("template %s: SAFETY_INSTRUCTION fragment[%d] missing patient_advocate_reviewed_by", tmpl.TemplateID, i)
			}
			if !frag.ReadingLevelValidated {
				return fmt.Errorf("template %s: SAFETY_INSTRUCTION fragment[%d] reading_level_validated must be true", tmpl.TemplateID, i)
			}
		}
	}

	return nil
}

// Get returns a template by its template_id.
func (l *TemplateLoader) Get(templateID string) (*models.CardTemplate, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.templates[templateID]
	return t, ok
}

// GetByDifferential returns all templates matching a differential_id.
func (l *TemplateLoader) GetByDifferential(differentialID string) []*models.CardTemplate {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.byDiff[differentialID]
}

// GetByNode returns all templates for a given node_id.
func (l *TemplateLoader) GetByNode(nodeID string) []*models.CardTemplate {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.byNode[nodeID]
}

// List returns all loaded templates.
func (l *TemplateLoader) List() []*models.CardTemplate {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]*models.CardTemplate, 0, len(l.templates))
	for _, t := range l.templates {
		result = append(result, t)
	}
	return result
}

// Count returns the number of loaded templates.
func (l *TemplateLoader) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.templates)
}

// Reload hot-reloads all templates from disk, replacing the in-memory index
// atomically under the write lock.
func (l *TemplateLoader) Reload() error {
	l.log.Info("hot-reloading templates...")
	return l.Load()
}
