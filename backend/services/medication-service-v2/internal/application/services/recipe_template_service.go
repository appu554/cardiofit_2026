package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
)

// RecipeTemplate represents a template for creating recipes
type RecipeTemplate struct {
	ID                  uuid.UUID                    `json:"id"`
	Name                string                       `json:"name"`
	ProtocolID          string                       `json:"protocol_id"`
	Category            string                       `json:"category"`
	Description         string                       `json:"description"`
	Version             string                       `json:"version"`
	ContextRequirements entities.ContextRequirements `json:"context_requirements"`
	CalculationRules    []entities.CalculationRule   `json:"calculation_rules"`
	SafetyRules         []entities.SafetyRule        `json:"safety_rules"`
	MonitoringRules     []entities.MonitoringRule    `json:"monitoring_rules"`
	ConditionalRules    []entities.ConditionalRule   `json:"conditional_rules"`
	DefaultTTL          time.Duration                `json:"default_ttl"`
	Tags                []string                     `json:"tags"`
	ClinicalEvidence    *entities.ClinicalEvidence   `json:"clinical_evidence,omitempty"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	CreatedBy           string                       `json:"created_by"`
	Status              entities.RecipeStatus        `json:"status"`
	IsActive            bool                         `json:"is_active"`
}

// RecipeTemplateService manages recipe templates
type RecipeTemplateService interface {
	CreateTemplate(ctx context.Context, template *RecipeTemplate) error
	GetTemplate(ctx context.Context, id uuid.UUID) (*RecipeTemplate, error)
	GetTemplatesByProtocol(ctx context.Context, protocolID string) ([]*RecipeTemplate, error)
	UpdateTemplate(ctx context.Context, template *RecipeTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	ListTemplates(ctx context.Context, filters TemplateFilters) ([]*RecipeTemplate, error)
	CreateRecipeFromTemplate(ctx context.Context, templateID uuid.UUID, customization RecipeCustomization) (*entities.Recipe, error)
	ValidateTemplate(ctx context.Context, template *RecipeTemplate) error
}

// TemplateFilters defines filters for template listing
type TemplateFilters struct {
	ProtocolID string   `json:"protocol_id,omitempty"`
	Category   string   `json:"category,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Status     entities.RecipeStatus `json:"status,omitempty"`
	IsActive   *bool    `json:"is_active,omitempty"`
	CreatedBy  string   `json:"created_by,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

// RecipeCustomization allows customizing recipes created from templates
type RecipeCustomization struct {
	Name                string                       `json:"name,omitempty"`
	Description         string                       `json:"description,omitempty"`
	Indication          string                       `json:"indication"`
	ContextOverrides    map[string]interface{}       `json:"context_overrides,omitempty"`
	RuleOverrides       []RuleOverride               `json:"rule_overrides,omitempty"`
	AdditionalRules     AdditionalRules              `json:"additional_rules,omitempty"`
	TTLOverride         *time.Duration               `json:"ttl_override,omitempty"`
	CreatedBy           string                       `json:"created_by"`
}

// RuleOverride allows overriding specific rules
type RuleOverride struct {
	RuleID      uuid.UUID              `json:"rule_id"`
	RuleType    string                 `json:"rule_type"` // calculation, safety, monitoring
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Priority    *int                   `json:"priority,omitempty"`
	Disabled    bool                   `json:"disabled,omitempty"`
}

// AdditionalRules allows adding extra rules
type AdditionalRules struct {
	CalculationRules []entities.CalculationRule `json:"calculation_rules,omitempty"`
	SafetyRules      []entities.SafetyRule      `json:"safety_rules,omitempty"`
	MonitoringRules  []entities.MonitoringRule  `json:"monitoring_rules,omitempty"`
}

// RecipeTemplateServiceImpl implements RecipeTemplateService
type RecipeTemplateServiceImpl struct {
	recipeRepository repositories.RecipeRepository
	templateRepo     TemplateRepository
	validationRules  []TemplateValidationRule
}

// TemplateRepository defines template storage operations
type TemplateRepository interface {
	Save(ctx context.Context, template *RecipeTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*RecipeTemplate, error)
	GetByProtocol(ctx context.Context, protocolID string) ([]*RecipeTemplate, error)
	List(ctx context.Context, filters TemplateFilters) ([]*RecipeTemplate, error)
	Update(ctx context.Context, template *RecipeTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TemplateValidationRule defines template validation logic
type TemplateValidationRule interface {
	Validate(ctx context.Context, template *RecipeTemplate) error
	GetRuleName() string
}

// NewRecipeTemplateService creates a new recipe template service
func NewRecipeTemplateService(recipeRepo repositories.RecipeRepository, templateRepo TemplateRepository) *RecipeTemplateServiceImpl {
	service := &RecipeTemplateServiceImpl{
		recipeRepository: recipeRepo,
		templateRepo:     templateRepo,
		validationRules:  make([]TemplateValidationRule, 0),
	}

	// Register default validation rules
	service.AddValidationRule(&BasicTemplateValidationRule{})
	service.AddValidationRule(&ProtocolConsistencyValidationRule{})
	service.AddValidationRule(&RuleConsistencyValidationRule{})

	return service
}

// AddValidationRule adds a validation rule
func (s *RecipeTemplateServiceImpl) AddValidationRule(rule TemplateValidationRule) {
	s.validationRules = append(s.validationRules, rule)
}

// CreateTemplate creates a new recipe template
func (s *RecipeTemplateServiceImpl) CreateTemplate(ctx context.Context, template *RecipeTemplate) error {
	// Generate ID if not provided
	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	// Set creation timestamp
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	// Validate template
	if err := s.ValidateTemplate(ctx, template); err != nil {
		return errors.Wrap(err, "template validation failed")
	}

	// Save template
	if err := s.templateRepo.Save(ctx, template); err != nil {
		return errors.Wrap(err, "failed to save template")
	}

	return nil
}

// GetTemplate retrieves a template by ID
func (s *RecipeTemplateServiceImpl) GetTemplate(ctx context.Context, id uuid.UUID) (*RecipeTemplate, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template")
	}

	return template, nil
}

// GetTemplatesByProtocol retrieves templates by protocol
func (s *RecipeTemplateServiceImpl) GetTemplatesByProtocol(ctx context.Context, protocolID string) ([]*RecipeTemplate, error) {
	templates, err := s.templateRepo.GetByProtocol(ctx, protocolID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get templates by protocol")
	}

	return templates, nil
}

// UpdateTemplate updates an existing template
func (s *RecipeTemplateServiceImpl) UpdateTemplate(ctx context.Context, template *RecipeTemplate) error {
	// Set update timestamp
	template.UpdatedAt = time.Now()

	// Validate template
	if err := s.ValidateTemplate(ctx, template); err != nil {
		return errors.Wrap(err, "template validation failed")
	}

	// Update template
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return errors.Wrap(err, "failed to update template")
	}

	return nil
}

// DeleteTemplate deletes a template
func (s *RecipeTemplateServiceImpl) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if err := s.templateRepo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "failed to delete template")
	}

	return nil
}

// ListTemplates lists templates with filters
func (s *RecipeTemplateServiceImpl) ListTemplates(ctx context.Context, filters TemplateFilters) ([]*RecipeTemplate, error) {
	templates, err := s.templateRepo.List(ctx, filters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list templates")
	}

	return templates, nil
}

// CreateRecipeFromTemplate creates a recipe from a template
func (s *RecipeTemplateServiceImpl) CreateRecipeFromTemplate(ctx context.Context, templateID uuid.UUID, customization RecipeCustomization) (*entities.Recipe, error) {
	// Get template
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template")
	}

	if !template.IsActive {
		return nil, errors.New("template is not active")
	}

	// Create recipe from template
	recipe := &entities.Recipe{
		ID:                uuid.New(),
		ProtocolID:        template.ProtocolID,
		Name:              template.Name,
		Version:           "1.0.0",
		Description:       template.Description,
		Indication:        customization.Indication,
		ContextRequirements: template.ContextRequirements,
		CalculationRules:  make([]entities.CalculationRule, len(template.CalculationRules)),
		SafetyRules:       make([]entities.SafetyRule, len(template.SafetyRules)),
		MonitoringRules:   make([]entities.MonitoringRule, len(template.MonitoringRules)),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		CreatedBy:         customization.CreatedBy,
		Status:            entities.RecipeStatusDraft,
		TTL:               template.DefaultTTL,
		ClinicalEvidence:  template.ClinicalEvidence,
	}

	// Apply customizations
	if customization.Name != "" {
		recipe.Name = customization.Name
	}
	if customization.Description != "" {
		recipe.Description = customization.Description
	}
	if customization.TTLOverride != nil {
		recipe.TTL = *customization.TTLOverride
	}

	// Copy rules with overrides
	copy(recipe.CalculationRules, template.CalculationRules)
	copy(recipe.SafetyRules, template.SafetyRules)
	copy(recipe.MonitoringRules, template.MonitoringRules)

	// Apply rule overrides
	if err := s.applyRuleOverrides(recipe, customization.RuleOverrides); err != nil {
		return nil, errors.Wrap(err, "failed to apply rule overrides")
	}

	// Add additional rules
	recipe.CalculationRules = append(recipe.CalculationRules, customization.AdditionalRules.CalculationRules...)
	recipe.SafetyRules = append(recipe.SafetyRules, customization.AdditionalRules.SafetyRules...)
	recipe.MonitoringRules = append(recipe.MonitoringRules, customization.AdditionalRules.MonitoringRules...)

	// Apply context overrides
	if err := s.applyContextOverrides(recipe, customization.ContextOverrides); err != nil {
		return nil, errors.Wrap(err, "failed to apply context overrides")
	}

	// Validate recipe
	if err := recipe.Validate(); err != nil {
		return nil, errors.Wrap(err, "generated recipe is invalid")
	}

	// Save recipe
	if err := s.recipeRepository.Save(ctx, recipe); err != nil {
		return nil, errors.Wrap(err, "failed to save recipe")
	}

	return recipe, nil
}

// ValidateTemplate validates a template
func (s *RecipeTemplateServiceImpl) ValidateTemplate(ctx context.Context, template *RecipeTemplate) error {
	for _, rule := range s.validationRules {
		if err := rule.Validate(ctx, template); err != nil {
			return errors.Wrapf(err, "validation rule '%s' failed", rule.GetRuleName())
		}
	}

	return nil
}

// Helper methods

func (s *RecipeTemplateServiceImpl) applyRuleOverrides(recipe *entities.Recipe, overrides []RuleOverride) error {
	for _, override := range overrides {
		switch override.RuleType {
		case "calculation":
			if err := s.applyCalculationRuleOverride(recipe, override); err != nil {
				return err
			}
		case "safety":
			if err := s.applySafetyRuleOverride(recipe, override); err != nil {
				return err
			}
		case "monitoring":
			if err := s.applyMonitoringRuleOverride(recipe, override); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown rule type: %s", override.RuleType)
		}
	}

	return nil
}

func (s *RecipeTemplateServiceImpl) applyCalculationRuleOverride(recipe *entities.Recipe, override RuleOverride) error {
	for i, rule := range recipe.CalculationRules {
		if rule.ID == override.RuleID {
			if override.Disabled {
				// Remove rule by creating new slice without this element
				recipe.CalculationRules = append(recipe.CalculationRules[:i], recipe.CalculationRules[i+1:]...)
				return nil
			}

			// Apply parameter overrides
			if override.Parameters != nil {
				for key, value := range override.Parameters {
					rule.Parameters[key] = value
				}
			}

			// Apply priority override
			if override.Priority != nil {
				rule.Priority = *override.Priority
			}

			recipe.CalculationRules[i] = rule
			return nil
		}
	}

	return fmt.Errorf("calculation rule with ID %s not found", override.RuleID)
}

func (s *RecipeTemplateServiceImpl) applySafetyRuleOverride(recipe *entities.Recipe, override RuleOverride) error {
	for i, rule := range recipe.SafetyRules {
		if rule.ID == override.RuleID {
			if override.Disabled {
				// Remove rule
				recipe.SafetyRules = append(recipe.SafetyRules[:i], recipe.SafetyRules[i+1:]...)
				return nil
			}

			// Apply priority override
			if override.Priority != nil {
				rule.Priority = *override.Priority
			}

			recipe.SafetyRules[i] = rule
			return nil
		}
	}

	return fmt.Errorf("safety rule with ID %s not found", override.RuleID)
}

func (s *RecipeTemplateServiceImpl) applyMonitoringRuleOverride(recipe *entities.Recipe, override RuleOverride) error {
	for i, rule := range recipe.MonitoringRules {
		if rule.ID == override.RuleID {
			if override.Disabled {
				// Remove rule
				recipe.MonitoringRules = append(recipe.MonitoringRules[:i], recipe.MonitoringRules[i+1:]...)
				return nil
			}

			recipe.MonitoringRules[i] = rule
			return nil
		}
	}

	return fmt.Errorf("monitoring rule with ID %s not found", override.RuleID)
}

func (s *RecipeTemplateServiceImpl) applyContextOverrides(recipe *entities.Recipe, overrides map[string]interface{}) error {
	// Apply overrides to context requirements
	// This would depend on specific implementation requirements
	return nil
}

// Template Validation Rules

// BasicTemplateValidationRule validates basic template structure
type BasicTemplateValidationRule struct{}

func (r *BasicTemplateValidationRule) Validate(ctx context.Context, template *RecipeTemplate) error {
	if template.Name == "" {
		return errors.New("template name is required")
	}

	if template.ProtocolID == "" {
		return errors.New("template protocol_id is required")
	}

	if len(template.CalculationRules) == 0 {
		return errors.New("template must have at least one calculation rule")
	}

	return nil
}

func (r *BasicTemplateValidationRule) GetRuleName() string {
	return "basic_template_validation"
}

// ProtocolConsistencyValidationRule validates protocol consistency
type ProtocolConsistencyValidationRule struct{}

func (r *ProtocolConsistencyValidationRule) Validate(ctx context.Context, template *RecipeTemplate) error {
	// Validate that all rules are consistent with the protocol
	for _, rule := range template.ConditionalRules {
		if rule.Protocol != template.ProtocolID {
			return fmt.Errorf("conditional rule protocol '%s' does not match template protocol '%s'", rule.Protocol, template.ProtocolID)
		}
	}

	return nil
}

func (r *ProtocolConsistencyValidationRule) GetRuleName() string {
	return "protocol_consistency_validation"
}

// RuleConsistencyValidationRule validates rule consistency
type RuleConsistencyValidationRule struct{}

func (r *RuleConsistencyValidationRule) Validate(ctx context.Context, template *RecipeTemplate) error {
	// Validate calculation rules
	for _, rule := range template.CalculationRules {
		if err := rule.Validate(); err != nil {
			return errors.Wrapf(err, "invalid calculation rule '%s'", rule.Name)
		}
	}

	// Validate safety rules
	for _, rule := range template.SafetyRules {
		if err := rule.Validate(); err != nil {
			return errors.Wrapf(err, "invalid safety rule '%s'", rule.Name)
		}
	}

	return nil
}

func (r *RuleConsistencyValidationRule) GetRuleName() string {
	return "rule_consistency_validation"
}

// Validate validates a recipe template
func (rt *RecipeTemplate) Validate() error {
	if rt.Name == "" {
		return errors.New("template name is required")
	}

	if rt.ProtocolID == "" {
		return errors.New("template protocol_id is required")
	}

	if rt.Category == "" {
		return errors.New("template category is required")
	}

	if len(rt.CalculationRules) == 0 {
		return errors.New("template must have at least one calculation rule")
	}

	return nil
}