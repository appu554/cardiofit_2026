package services

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

// eligibilityCriterionYAML mirrors models.EligibilityCriterion with yaml tags so
// gopkg.in/yaml.v3 maps snake_case YAML keys correctly (the model struct only has
// json tags, which yaml.v3 does not fall back to).
type eligibilityCriterionYAML struct {
	FeatureKey string   `yaml:"feature_key"`
	Operator   string   `yaml:"operator"`
	Threshold  float64  `yaml:"threshold"`
	Set        []string `yaml:"set"`
}

// contraindicationYAML is the YAML-parse-time twin of models.Contraindication.
type contraindicationYAML struct {
	FeatureKey string   `yaml:"feature_key"`
	Operator   string   `yaml:"operator"`
	Threshold  float64  `yaml:"threshold"`
	Set        []string `yaml:"set"`
	Reason     string   `yaml:"reason"`
}

type interventionYAML struct {
	CohortID                  string `yaml:"cohort_id"`
	Version                   string `yaml:"version"`
	PrimaryCATEHorizonDays    int    `yaml:"primary_cate_horizon_days"`
	RecommendationCardinality int    `yaml:"recommendation_cardinality"`
	Interventions             []struct {
		ID                string                     `yaml:"id"`
		Category          string                     `yaml:"category"`
		Name              string                     `yaml:"name"`
		ClinicianLanguage string                     `yaml:"clinician_language"`
		CoolDownHours     int                        `yaml:"cool_down_hours"`
		ResourceCost      float64                    `yaml:"resource_cost"`
		FeatureSignature  []string                   `yaml:"feature_signature"`
		Eligibility       []eligibilityCriterionYAML `yaml:"eligibility"`
		Contraindications []contraindicationYAML     `yaml:"contraindications"`
	} `yaml:"interventions"`
}

// InterventionRegistry loads YAML-defined intervention taxonomies into the DB and
// answers eligibility queries for CATE scoring. Sprint 1 sits entirely in Go; the
// eligibility predicates are evaluated in-memory against a feature map.
type InterventionRegistry struct {
	db *gorm.DB
}

func NewInterventionRegistry(db *gorm.DB) *InterventionRegistry {
	return &InterventionRegistry{db: db}
}

// LoadFromYAML parses a single market-config YAML file and upserts each definition.
// Idempotent — safe to call on every service start.
func (r *InterventionRegistry) LoadFromYAML(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml: %w", err)
	}
	var cfg interventionYAML
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	for _, iv := range cfg.Interventions {
		// Convert YAML-parsed structs (with yaml tags) to model types (with json tags)
		// before marshaling to JSON for storage. This ensures feature_key round-trips
		// correctly — yaml.v3 cannot use json struct tags for unmarshaling.
		eligibility := make([]models.EligibilityCriterion, len(iv.Eligibility))
		for i, e := range iv.Eligibility {
			eligibility[i] = models.EligibilityCriterion{
				FeatureKey: e.FeatureKey,
				Operator:   e.Operator,
				Threshold:  e.Threshold,
				Set:        e.Set,
			}
		}
		contraindications := make([]models.Contraindication, len(iv.Contraindications))
		for i, c := range iv.Contraindications {
			contraindications[i] = models.Contraindication{
				FeatureKey: c.FeatureKey,
				Operator:   c.Operator,
				Threshold:  c.Threshold,
				Set:        c.Set,
				Reason:     c.Reason,
			}
		}
		eligJSON, err := json.Marshal(eligibility)
		if err != nil {
			return fmt.Errorf("marshal eligibility for %s: %w", iv.ID, err)
		}
		contraJSON, err := json.Marshal(contraindications)
		if err != nil {
			return fmt.Errorf("marshal contraindications for %s: %w", iv.ID, err)
		}
		def := models.InterventionDefinition{
			ID:                    iv.ID,
			CohortID:              cfg.CohortID,
			Category:              iv.Category,
			Name:                  iv.Name,
			ClinicianLanguage:     iv.ClinicianLanguage,
			CoolDownHours:         iv.CoolDownHours,
			ResourceCost:          iv.ResourceCost,
			FeatureSignature:      iv.FeatureSignature,
			EligibilityJSON:       string(eligJSON),
			ContraindicationsJSON: string(contraJSON),
			Version:               cfg.Version,
			SourceYAMLPath:        path,
		}
		if err := def.Validate(); err != nil {
			return fmt.Errorf("validate %s: %w", iv.ID, err)
		}
		if err := r.db.Save(&def).Error; err != nil {
			return fmt.Errorf("persist %s: %w", iv.ID, err)
		}
	}
	return nil
}

// ListEligible returns interventions whose cohort matches and whose eligibility
// criteria hold and whose contraindications do NOT hold for the given feature vector.
// Order of evaluation per intervention: contraindication → eligibility. Either failure
// excludes the intervention.
func (r *InterventionRegistry) ListEligible(cohortID string, features map[string]float64) ([]models.InterventionDefinition, error) {
	var all []models.InterventionDefinition
	if err := r.db.Where("cohort_id = ?", cohortID).Find(&all).Error; err != nil {
		return nil, err
	}
	out := make([]models.InterventionDefinition, 0, len(all))
	for _, d := range all {
		var contra []models.Contraindication
		if d.ContraindicationsJSON != "" {
			_ = json.Unmarshal([]byte(d.ContraindicationsJSON), &contra)
		}
		if anyContraindicationMatches(contra, features) {
			continue
		}
		var elig []models.EligibilityCriterion
		if d.EligibilityJSON != "" {
			_ = json.Unmarshal([]byte(d.EligibilityJSON), &elig)
		}
		if !allEligibilityMatches(elig, features) {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func anyContraindicationMatches(contra []models.Contraindication, f map[string]float64) bool {
	for _, c := range contra {
		if predicateHolds(c.Operator, f[c.FeatureKey], c.Threshold) {
			return true
		}
	}
	return false
}

func allEligibilityMatches(elig []models.EligibilityCriterion, f map[string]float64) bool {
	for _, e := range elig {
		if !predicateHolds(e.Operator, f[e.FeatureKey], e.Threshold) {
			return false
		}
	}
	return true
}

// predicateHolds evaluates one criterion against a feature value. Missing features
// default to 0 (the zero value of float64), which means an eligibility predicate like
// "gte 5" correctly fails for a patient without that feature recorded.
func predicateHolds(op string, value, threshold float64) bool {
	switch op {
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}
