package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Stage 4 Pathway definitions — loaded from YAML config files
// ---------------------------------------------------------------------------

// MedicationEntry represents a single medication rule in a pathway.
type MedicationEntry struct {
	Class     string `yaml:"class"     json:"class"`
	Intensity string `yaml:"intensity" json:"intensity,omitempty"`
	Condition string `yaml:"condition" json:"condition,omitempty"`
	Rationale string `yaml:"rationale" json:"rationale,omitempty"`
	Preferred string `yaml:"preferred" json:"preferred,omitempty"`
	Note      string `yaml:"note"      json:"note,omitempty"`
	Priority  string `yaml:"priority"  json:"priority,omitempty"`
}

// ContraindicatedEntry represents a contraindicated drug class.
type ContraindicatedEntry struct {
	Class  string `yaml:"class"  json:"class"`
	Reason string `yaml:"reason" json:"reason"`
}

// TargetAdjustments holds BP/lipid targets per substage.
type TargetAdjustments struct {
	SBPTarget int `yaml:"sbp_target" json:"sbp_target,omitempty"`
	LDLTarget int `yaml:"ldl_target" json:"ldl_target,omitempty"`
}

// SubstagePathway defines mandatory/recommended/contraindicated meds for a substage.
type SubstagePathway struct {
	Label                  string                 `yaml:"label"                   json:"label"`
	Strategy               string                 `yaml:"strategy"                json:"strategy"`
	MandatoryMedications   []MedicationEntry      `yaml:"mandatory_medications"   json:"mandatory_medications,omitempty"`
	RecommendedMedications []MedicationEntry      `yaml:"recommended_medications" json:"recommended_medications,omitempty"`
	Contraindicated        []ContraindicatedEntry `yaml:"contraindicated"         json:"contraindicated,omitempty"`
	TargetAdjustments      *TargetAdjustments     `yaml:"target_adjustments"      json:"target_adjustments,omitempty"`
}

// HFSubstagePathways holds the three HF-specific pathways for Stage 4c.
type HFSubstagePathways struct {
	HFrEF  *SubstagePathway `yaml:"hfref"  json:"hfref,omitempty"`
	HFmrEF *SubstagePathway `yaml:"hfmref" json:"hfmref,omitempty"`
	HFpEF  *SubstagePathway `yaml:"hfpef"  json:"hfpef,omitempty"`
}

// Stage4cPathway extends SubstagePathway with HF-specific sub-pathways.
type Stage4cPathway struct {
	SubstagePathway `yaml:",inline"`
	HFSubstages     HFSubstagePathways `yaml:",inline"`
}

// Stage4Pathways is the top-level YAML structure for ckm_stage4_pathways.yaml.
type Stage4Pathways struct {
	Stage4a SubstagePathway `yaml:"stage_4a" json:"stage_4a"`
	Stage4b SubstagePathway `yaml:"stage_4b" json:"stage_4b"`
	Stage4c Stage4cPathway  `yaml:"stage_4c" json:"stage_4c"`
}

// ---------------------------------------------------------------------------
// PathwayLoader — loads and caches pathway definitions from YAML
// ---------------------------------------------------------------------------

// PathwayLoader loads CKM Stage 4 pathway definitions from YAML files.
type PathwayLoader struct {
	mu        sync.RWMutex
	pathways  *Stage4Pathways
	configDir string
}

// NewPathwayLoader creates a loader pointed at the given config directory.
// configDir should be the market-configs root (containing shared/, india/, australia/).
func NewPathwayLoader(configDir string) *PathwayLoader {
	return &PathwayLoader{configDir: configDir}
}

// Load reads ckm_stage4_pathways.yaml from the shared config directory.
func (l *PathwayLoader) Load() error {
	path := filepath.Join(l.configDir, "shared", "ckm_stage4_pathways.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("PathwayLoader: read %s: %w", path, err)
	}

	var pw Stage4Pathways
	if err := yaml.Unmarshal(data, &pw); err != nil {
		return fmt.Errorf("PathwayLoader: parse %s: %w", path, err)
	}

	l.mu.Lock()
	l.pathways = &pw
	l.mu.Unlock()
	return nil
}

// GetPathways returns the loaded pathway definitions, or an error if not loaded.
func (l *PathwayLoader) GetPathways() (*Stage4Pathways, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.pathways == nil {
		return nil, fmt.Errorf("PathwayLoader: pathways not loaded — call Load() first")
	}
	return l.pathways, nil
}

// QueryMandatory returns the mandatory medications for a given CKM substage.
// For Stage 4c, hfType selects the HF-specific pathway (HFrEF/HFmrEF/HFpEF).
func (l *PathwayLoader) QueryMandatory(ckmStage, hfType string) ([]MedicationEntry, error) {
	pw, err := l.GetPathways()
	if err != nil {
		return nil, err
	}

	switch ckmStage {
	case "4a":
		return pw.Stage4a.MandatoryMedications, nil
	case "4b":
		return pw.Stage4b.MandatoryMedications, nil
	case "4c":
		return l.queryHFMandatory(pw, hfType)
	default:
		return nil, nil
	}
}

// QueryContraindicated returns contraindicated medications for a CKM substage.
func (l *PathwayLoader) QueryContraindicated(ckmStage, hfType string) ([]ContraindicatedEntry, error) {
	pw, err := l.GetPathways()
	if err != nil {
		return nil, err
	}

	switch ckmStage {
	case "4a":
		return pw.Stage4a.Contraindicated, nil
	case "4b":
		return pw.Stage4b.Contraindicated, nil
	case "4c":
		return l.queryHFContraindicated(pw, hfType)
	default:
		return nil, nil
	}
}

// QueryRecommended returns recommended medications for a CKM substage.
func (l *PathwayLoader) QueryRecommended(ckmStage, hfType string) ([]MedicationEntry, error) {
	pw, err := l.GetPathways()
	if err != nil {
		return nil, err
	}

	switch ckmStage {
	case "4a":
		return pw.Stage4a.RecommendedMedications, nil
	case "4b":
		return pw.Stage4b.RecommendedMedications, nil
	case "4c":
		return l.queryHFRecommended(pw, hfType)
	default:
		return nil, nil
	}
}

func (l *PathwayLoader) queryHFMandatory(pw *Stage4Pathways, hfType string) ([]MedicationEntry, error) {
	switch hfType {
	case "HFrEF":
		if pw.Stage4c.HFSubstages.HFrEF != nil {
			return pw.Stage4c.HFSubstages.HFrEF.MandatoryMedications, nil
		}
	case "HFmrEF":
		if pw.Stage4c.HFSubstages.HFmrEF != nil {
			return pw.Stage4c.HFSubstages.HFmrEF.MandatoryMedications, nil
		}
	case "HFpEF":
		if pw.Stage4c.HFSubstages.HFpEF != nil {
			return pw.Stage4c.HFSubstages.HFpEF.MandatoryMedications, nil
		}
	}
	return nil, nil
}

func (l *PathwayLoader) queryHFContraindicated(pw *Stage4Pathways, hfType string) ([]ContraindicatedEntry, error) {
	switch hfType {
	case "HFrEF":
		if pw.Stage4c.HFSubstages.HFrEF != nil {
			return pw.Stage4c.HFSubstages.HFrEF.Contraindicated, nil
		}
	case "HFmrEF":
		if pw.Stage4c.HFSubstages.HFmrEF != nil {
			return pw.Stage4c.HFSubstages.HFmrEF.Contraindicated, nil
		}
	case "HFpEF":
		if pw.Stage4c.HFSubstages.HFpEF != nil {
			return pw.Stage4c.HFSubstages.HFpEF.Contraindicated, nil
		}
	}
	return nil, nil
}

func (l *PathwayLoader) queryHFRecommended(pw *Stage4Pathways, hfType string) ([]MedicationEntry, error) {
	switch hfType {
	case "HFrEF":
		// HFrEF pathway has no recommended section in YAML (only mandatory + contraindicated)
		return nil, nil
	case "HFmrEF":
		return nil, nil
	case "HFpEF":
		if pw.Stage4c.HFSubstages.HFpEF != nil {
			return pw.Stage4c.HFSubstages.HFpEF.RecommendedMedications, nil
		}
	}
	return nil, nil
}
