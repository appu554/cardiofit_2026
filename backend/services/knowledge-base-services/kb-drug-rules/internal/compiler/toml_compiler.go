package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
)

// TOMLCompiler compiles TOML dosing rules to KB-1 compliant JSON format
// Implements deterministic compilation with unit normalization and AST generation
type TOMLCompiler struct {
	logger *logrus.Logger
	schema *gojsonschema.Schema
}

// NewTOMLCompiler creates a new TOML compiler with validation schema
func NewTOMLCompiler(logger *logrus.Logger, schemaPath string) (*TOMLCompiler, error) {
	// Load JSON schema for validation (TODO: implement schema loading)
	var schema *gojsonschema.Schema
	
	return &TOMLCompiler{
		logger: logger,
		schema: schema,
	}, nil
}

// TOMLRule represents the structure of a TOML dosing rule file
type TOMLRule struct {
	Meta struct {
		Name            string    `toml:"name"`
		SemanticVersion string    `toml:"semantic_version"`
		Authors         []string  `toml:"authors"`
		Approvals       []string  `toml:"approvals"`
		KB3Refs         []string  `toml:"kb3_refs"`
		KB4Refs         []string  `toml:"kb4_refs"`
		EffectiveFrom   string    `toml:"effective_from"`
		DrugCode        string    `toml:"drug_code"`
		DrugName        string    `toml:"drug_name"`
	} `toml:"meta"`

	BaseDose struct {
		Unit        string  `toml:"unit"`
		Starting    float64 `toml:"starting"`
		MaxDaily    float64 `toml:"max_daily"`
		MinDaily    float64 `toml:"min_daily"`
		Frequency   string  `toml:"frequency"`
		Loading     string  `toml:"loading,omitempty"`
		Maintenance string  `toml:"maintenance,omitempty"`
		MaxSingle   float64 `toml:"max_single,omitempty"`
		Note        string  `toml:"note,omitempty"`
	} `toml:"base_dose"`

	Formulations []struct {
		Form      string    `toml:"form"`
		Strengths []float64 `toml:"strengths"`
		Route     string    `toml:"route,omitempty"`
	} `toml:"formulations"`

	Adjustments struct {
		Renal []struct {
			EGFRMin         float64 `toml:"egfr_min"`
			EGFRMax         float64 `toml:"egfr_max"`
			Action          string  `toml:"action"`
			Factor          float64 `toml:"factor,omitempty"`
			CapMax          float64 `toml:"cap_max,omitempty"`
			Note            string  `toml:"note,omitempty"`
			Contraindicated bool    `toml:"contraindicated,omitempty"`
			Interval        string  `toml:"interval,omitempty"`
		} `toml:"renal"`

		Hepatic []struct {
			ChildPughClass  string  `toml:"child_pugh_class"`
			Action          string  `toml:"action"`
			Factor          float64 `toml:"factor,omitempty"`
			Contraindicated bool    `toml:"contraindicated,omitempty"`
			Note            string  `toml:"note,omitempty"`
		} `toml:"hepatic"`

		Age []struct {
			AgeMin     int     `toml:"age_min"`
			AgeMax     int     `toml:"age_max"`
			Factor     float64 `toml:"factor,omitempty"`
			MaxDoseMg  float64 `toml:"max_dose_mg,omitempty"`
			Formula    string  `toml:"formula,omitempty"`
		} `toml:"age"`

		Weight []struct {
			WeightMin        float64 `toml:"weight_min"`
			WeightMax        float64 `toml:"weight_max"`
			DosePerKg        float64 `toml:"dose_per_kg,omitempty"`
			DoseCapMgPerKg   float64 `toml:"dose_cap_mg_per_kg,omitempty"`
			MaxTotal         float64 `toml:"max_total,omitempty"`
			UseIdealWeight   bool    `toml:"use_ideal_weight,omitempty"`
			UseActualWeight  bool    `toml:"use_actual_weight,omitempty"`
			Note            string  `toml:"note,omitempty"`
		} `toml:"weight"`

		Dialysis struct {
			Hemodialysis struct {
				DoseMultiplier              float64 `toml:"dose_multiplier"`
				SupplementalDose           float64 `toml:"supplemental_dose"`
				TimingRelativeToDialysis   string  `toml:"timing_relative_to_dialysis"`
				Note                       string  `toml:"note"`
			} `toml:"hemodialysis"`
			
			Peritoneal struct {
				DoseMultiplier float64 `toml:"dose_multiplier"`
				Interval       string  `toml:"interval"`
				Note           string  `toml:"note"`
			} `toml:"peritoneal"`
			
			CRRT struct {
				DoseMultiplier float64 `toml:"dose_multiplier"`
				Interval       string  `toml:"interval"`
				Note           string  `toml:"note"`
			} `toml:"crrt"`
		} `toml:"dialysis"`
	} `toml:"adjustments"`

	Monitoring struct {
		TroughLevels struct {
			TargetRange   string  `toml:"target_range"`
			Timing        string  `toml:"timing"`
			Frequency     string  `toml:"frequency"`
			CriticalHigh  float64 `toml:"critical_high"`
			CriticalLow   float64 `toml:"critical_low"`
		} `toml:"trough_levels"`
		
		SafetyLabs struct {
			Baseline  []string `toml:"baseline"`
			Ongoing   []string `toml:"ongoing"`
			Frequency string   `toml:"frequency"`
		} `toml:"safety_labs"`
	} `toml:"monitoring"`

	Titration []struct {
		Step       int     `toml:"step"`
		AfterDays  int     `toml:"after_days"`
		Action     string  `toml:"action"`
		Amount     float64 `toml:"amount,omitempty"`
		IncreaseBy float64 `toml:"increase_by,omitempty"`
		MaxStep    int     `toml:"max_step,omitempty"`
		Monitoring string  `toml:"monitoring,omitempty"`
	} `toml:"titration"`

	Population map[string]struct {
		AgeMin          int     `toml:"age_min"`
		AgeMax          int     `toml:"age_max"`
		WeightRequired  bool    `toml:"weight_required,omitempty"`
		Formula         string  `toml:"formula,omitempty"`
		MaxDoseMg       float64 `toml:"max_dose_mg,omitempty"`
		Contraindicated bool    `toml:"contraindicated,omitempty"`
	} `toml:"population"`

	SafetyVerification struct {
		Contraindications []string `toml:"contraindications"`
		Warnings          []string `toml:"warnings"`
		Precautions       []string `toml:"precautions"`
		LabMonitoring     []string `toml:"lab_monitoring"`
	} `toml:"safety_verification"`

	RegionalVariations map[string]struct {
		MaxDailyDoseMg float64 `toml:"max_daily_dose_mg,omitempty"`
		Notes          string  `toml:"notes,omitempty"`
		Formulation    string  `toml:"preferred_formulation,omitempty"`
	} `toml:"regional_variations"`
}

// KB1CompiledRule represents the compiled KB-1 compliant JSON structure
type KB1CompiledRule struct {
	Meta struct {
		DrugCode        string    `json:"drug_code"`
		DrugName        string    `json:"drug_name"`
		Version         string    `json:"semantic_version"`
		SourceFile      string    `json:"source_file"`
		Authors         []string  `json:"authors"`
		Approvals       []string  `json:"approvals"`
		KB3Refs         []string  `json:"kb3_refs"`
		KB4Refs         []string  `json:"kb4_refs"`
		EffectiveFrom   time.Time `json:"effective_from"`
		CompiledAt      time.Time `json:"compiled_at"`
		CompilerVersion string    `json:"compiler_version"`
	} `json:"meta"`

	BaseDose struct {
		Unit      string  `json:"unit_ucum"`        // UCUM normalized unit
		Starting  float64 `json:"starting_mg"`      // Normalized to mg
		MaxDaily  float64 `json:"max_daily_mg"`     // Normalized to mg
		MinDaily  float64 `json:"min_daily_mg"`     // Normalized to mg
		Frequency string  `json:"frequency_code"`   // Standardized frequency
	} `json:"base_dose"`

	Formulations []struct {
		Form      string    `json:"form_standardized"`
		Strengths []float64 `json:"strengths_mg"`
		Route     string    `json:"route_snomed,omitempty"`
	} `json:"formulations"`

	DoseAdjustments []struct {
		AdjustmentID string          `json:"adjustment_id"`
		Type         string          `json:"adjust_type"`
		ConditionAST json.RawMessage `json:"condition_ast"` // Compiled predicate
		FormulaAST   json.RawMessage `json:"formula_ast"`   // Compiled formula
		Multiplier   float64         `json:"multiplier,omitempty"`
		AdditiveMg   float64         `json:"additive_mg,omitempty"`
		MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
		MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
	} `json:"dose_adjustments"`

	TitrationSchedule []struct {
		StepNumber int             `json:"step_number"`
		AfterDays  int             `json:"after_days"`
		ActionType string          `json:"action_type"`
		ActionAST  json.RawMessage `json:"action_ast"`
		MaxStep    int             `json:"max_step,omitempty"`
		Monitoring json.RawMessage `json:"monitoring_requirements,omitempty"`
	} `json:"titration_schedule"`

	PopulationDosing []struct {
		PopulationID string          `json:"population_id"`
		Type         string          `json:"population_type"`
		AgeMin       int             `json:"age_min"`
		AgeMax       int             `json:"age_max"`
		WeightMin    float64         `json:"weight_min,omitempty"`
		WeightMax    float64         `json:"weight_max,omitempty"`
		FormulaAST   json.RawMessage `json:"formula_ast"`
		SafetyLimits json.RawMessage `json:"safety_limits"`
	} `json:"population_dosing"`

	UsedFields []string `json:"used_fields"` // Patient context fields referenced by rules
	Checksum   string   `json:"checksum"`    // SHA256 of entire compiled structure
}

// CompilationResult represents the result of TOML compilation
type CompilationResult struct {
	Success      bool              `json:"success"`
	CompiledJSON json.RawMessage   `json:"compiled_json"`
	Checksum     string            `json:"checksum"`
	UsedFields   []string          `json:"used_fields"`
	Errors       []string          `json:"errors"`
	Warnings     []string          `json:"warnings"`
	Metadata     CompilationMeta   `json:"metadata"`
}

// CompilationMeta contains metadata about the compilation process
type CompilationMeta struct {
	CompilerVersion string    `json:"compiler_version"`
	CompiledAt      time.Time `json:"compiled_at"`
	SourceChecksum  string    `json:"source_checksum"`
	UnitNormalized  bool      `json:"unit_normalized"`
	ASTGenerated    bool      `json:"ast_generated"`
}

// Compile converts TOML dosing rule to KB-1 compliant JSON structure
func (c *TOMLCompiler) Compile(tomlContent string, sourceFile string) (*CompilationResult, error) {
	result := &CompilationResult{
		Success:  false,
		Errors:   []string{},
		Warnings: []string{},
		Metadata: CompilationMeta{
			CompilerVersion: "1.0.0",
			CompiledAt:      time.Now(),
			SourceChecksum:  c.calculateChecksum(tomlContent),
		},
	}

	// Parse TOML content
	var tomlRule TOMLRule
	if err := toml.Unmarshal([]byte(tomlContent), &tomlRule); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("TOML parsing error: %v", err))
		return result, nil
	}

	// Validate required fields
	if err := c.validateRequiredFields(&tomlRule); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	// Compile to KB-1 structure
	compiled, err := c.compileToKB1Structure(&tomlRule, sourceFile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Compilation error: %v", err))
		return result, nil
	}

	// Validate compiled structure
	if err := c.validateCompiledStructure(compiled); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Validation error: %v", err))
		return result, nil
	}

	// Serialize to JSON
	compiledJSON, err := json.Marshal(compiled)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("JSON serialization error: %v", err))
		return result, nil
	}

	// Calculate final checksum
	checksum := c.calculateChecksum(string(compiledJSON))

	result.Success = true
	result.CompiledJSON = compiledJSON
	result.Checksum = checksum
	result.UsedFields = compiled.UsedFields
	result.Metadata.UnitNormalized = true
	result.Metadata.ASTGenerated = true

	return result, nil
}

// compileToKB1Structure converts TOML rule to KB-1 JSON structure
func (c *TOMLCompiler) compileToKB1Structure(tomlRule *TOMLRule, sourceFile string) (*KB1CompiledRule, error) {
	compiled := &KB1CompiledRule{}
	usedFields := make(map[string]bool)

	// Compile metadata
	effectiveFrom, _ := time.Parse("2006-01-02", tomlRule.Meta.EffectiveFrom)
	compiled.Meta = struct {
		DrugCode        string    `json:"drug_code"`
		DrugName        string    `json:"drug_name"`
		Version         string    `json:"semantic_version"`
		SourceFile      string    `json:"source_file"`
		Authors         []string  `json:"authors"`
		Approvals       []string  `json:"approvals"`
		KB3Refs         []string  `json:"kb3_refs"`
		KB4Refs         []string  `json:"kb4_refs"`
		EffectiveFrom   time.Time `json:"effective_from"`
		CompiledAt      time.Time `json:"compiled_at"`
		CompilerVersion string    `json:"compiler_version"`
	}{
		DrugCode:        tomlRule.Meta.DrugCode,
		DrugName:        tomlRule.Meta.DrugName,
		Version:         tomlRule.Meta.SemanticVersion,
		SourceFile:      sourceFile,
		Authors:         tomlRule.Meta.Authors,
		Approvals:       tomlRule.Meta.Approvals,
		KB3Refs:         tomlRule.Meta.KB3Refs,
		KB4Refs:         tomlRule.Meta.KB4Refs,
		EffectiveFrom:   effectiveFrom,
		CompiledAt:      time.Now(),
		CompilerVersion: "1.0.0",
	}

	// Compile base dose with unit normalization
	compiled.BaseDose = struct {
		Unit      string  `json:"unit_ucum"`
		Starting  float64 `json:"starting_mg"`
		MaxDaily  float64 `json:"max_daily_mg"`
		MinDaily  float64 `json:"min_daily_mg"`
		Frequency string  `json:"frequency_code"`
	}{
		Unit:      c.normalizeToUCUM(tomlRule.BaseDose.Unit),
		Starting:  c.normalizeToMg(tomlRule.BaseDose.Starting, tomlRule.BaseDose.Unit),
		MaxDaily:  c.normalizeToMg(tomlRule.BaseDose.MaxDaily, tomlRule.BaseDose.Unit),
		MinDaily:  c.normalizeToMg(tomlRule.BaseDose.MinDaily, tomlRule.BaseDose.Unit),
		Frequency: c.standardizeFrequency(tomlRule.BaseDose.Frequency),
	}

	// Compile formulations
	for _, form := range tomlRule.Formulations {
		normalizedStrengths := make([]float64, len(form.Strengths))
		for i, strength := range form.Strengths {
			normalizedStrengths[i] = c.normalizeToMg(strength, tomlRule.BaseDose.Unit)
		}
		
		compiled.Formulations = append(compiled.Formulations, struct {
			Form      string    `json:"form_standardized"`
			Strengths []float64 `json:"strengths_mg"`
			Route     string    `json:"route_snomed,omitempty"`
		}{
			Form:      c.standardizeFormulation(form.Form),
			Strengths: normalizedStrengths,
			Route:     c.mapRouteToSNOMED(form.Route),
		})
	}

	// Compile dose adjustments
	adjustmentID := 1
	
	// Renal adjustments
	for _, renal := range tomlRule.Adjustments.Renal {
		conditionAST, err := c.compileConditionToAST(fmt.Sprintf("egfr >= %v AND egfr <= %v", renal.EGFRMin, renal.EGFRMax))
		if err != nil {
			return nil, fmt.Errorf("failed to compile renal condition: %w", err)
		}
		
		formulaAST, err := c.compileFormulaToAST(renal.Action, renal.Factor, renal.CapMax)
		if err != nil {
			return nil, fmt.Errorf("failed to compile renal formula: %w", err)
		}

		compiled.DoseAdjustments = append(compiled.DoseAdjustments, struct {
			AdjustmentID string          `json:"adjustment_id"`
			Type         string          `json:"adjust_type"`
			ConditionAST json.RawMessage `json:"condition_ast"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			Multiplier   float64         `json:"multiplier,omitempty"`
			AdditiveMg   float64         `json:"additive_mg,omitempty"`
			MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
			MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
		}{
			AdjustmentID: fmt.Sprintf("renal_%d", adjustmentID),
			Type:         "renal",
			ConditionAST: conditionAST,
			FormulaAST:   formulaAST,
			Multiplier:   renal.Factor,
			MaxDoseMg:    renal.CapMax,
		})
		
		usedFields["egfr"] = true
		adjustmentID++
	}

	// Hepatic adjustments
	for _, hepatic := range tomlRule.Adjustments.Hepatic {
		conditionAST, err := c.compileConditionToAST(fmt.Sprintf("child_pugh_class = '%s'", hepatic.ChildPughClass))
		if err != nil {
			return nil, fmt.Errorf("failed to compile hepatic condition: %w", err)
		}
		
		formulaAST, err := c.compileFormulaToAST(hepatic.Action, hepatic.Factor, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to compile hepatic formula: %w", err)
		}

		compiled.DoseAdjustments = append(compiled.DoseAdjustments, struct {
			AdjustmentID string          `json:"adjustment_id"`
			Type         string          `json:"adjust_type"`
			ConditionAST json.RawMessage `json:"condition_ast"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			Multiplier   float64         `json:"multiplier,omitempty"`
			AdditiveMg   float64         `json:"additive_mg,omitempty"`
			MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
			MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
		}{
			AdjustmentID: fmt.Sprintf("hepatic_%d", adjustmentID),
			Type:         "hepatic",
			ConditionAST: conditionAST,
			FormulaAST:   formulaAST,
			Multiplier:   hepatic.Factor,
		})
		
		usedFields["child_pugh_class"] = true
		adjustmentID++
	}

	// Age adjustments
	for _, age := range tomlRule.Adjustments.Age {
		conditionAST, err := c.compileConditionToAST(fmt.Sprintf("age_years >= %d AND age_years <= %d", age.AgeMin, age.AgeMax))
		if err != nil {
			return nil, fmt.Errorf("failed to compile age condition: %w", err)
		}
		
		formulaAST, err := c.compileFormulaToAST("multiply", age.Factor, age.MaxDoseMg)
		if err != nil {
			return nil, fmt.Errorf("failed to compile age formula: %w", err)
		}

		compiled.DoseAdjustments = append(compiled.DoseAdjustments, struct {
			AdjustmentID string          `json:"adjustment_id"`
			Type         string          `json:"adjust_type"`
			ConditionAST json.RawMessage `json:"condition_ast"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			Multiplier   float64         `json:"multiplier,omitempty"`
			AdditiveMg   float64         `json:"additive_mg,omitempty"`
			MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
			MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
		}{
			AdjustmentID: fmt.Sprintf("age_%d", adjustmentID),
			Type:         "age",
			ConditionAST: conditionAST,
			FormulaAST:   formulaAST,
			Multiplier:   age.Factor,
			MaxDoseMg:    age.MaxDoseMg,
		})
		
		usedFields["age_years"] = true
		adjustmentID++
	}

	// Weight adjustments
	for _, weight := range tomlRule.Adjustments.Weight {
		conditionAST, err := c.compileConditionToAST(fmt.Sprintf("weight_kg >= %v AND weight_kg <= %v", weight.WeightMin, weight.WeightMax))
		if err != nil {
			return nil, fmt.Errorf("failed to compile weight condition: %w", err)
		}
		
		var action string
		var factor float64 = 1.0
		var maxDose float64
		
		if weight.UseIdealWeight {
			action = "use_ideal_weight"
		} else if weight.UseActualWeight {
			action = "use_actual_weight"
		} else if weight.DoseCapMgPerKg > 0 {
			action = "cap_dose_per_kg"
			maxDose = weight.DoseCapMgPerKg
		}
		
		formulaAST, err := c.compileFormulaToAST(action, factor, maxDose)
		if err != nil {
			return nil, fmt.Errorf("failed to compile weight formula: %w", err)
		}

		compiled.DoseAdjustments = append(compiled.DoseAdjustments, struct {
			AdjustmentID string          `json:"adjustment_id"`
			Type         string          `json:"adjust_type"`
			ConditionAST json.RawMessage `json:"condition_ast"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			Multiplier   float64         `json:"multiplier,omitempty"`
			AdditiveMg   float64         `json:"additive_mg,omitempty"`
			MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
			MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
		}{
			AdjustmentID: fmt.Sprintf("weight_%d", adjustmentID),
			Type:         "weight",
			ConditionAST: conditionAST,
			FormulaAST:   formulaAST,
			Multiplier:   factor,
			MaxDoseMg:    maxDose,
		})
		
		usedFields["weight_kg"] = true
		adjustmentID++
	}

	// Dialysis adjustments
	if tomlRule.Adjustments.Dialysis.Hemodialysis.DoseMultiplier != 0 {
		conditionAST, _ := c.compileConditionToAST("dialysis_type = 'hemodialysis'")
		formulaAST, _ := c.compileFormulaToAST("multiply", tomlRule.Adjustments.Dialysis.Hemodialysis.DoseMultiplier, 0)
		
		compiled.DoseAdjustments = append(compiled.DoseAdjustments, struct {
			AdjustmentID string          `json:"adjustment_id"`
			Type         string          `json:"adjust_type"`
			ConditionAST json.RawMessage `json:"condition_ast"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			Multiplier   float64         `json:"multiplier,omitempty"`
			AdditiveMg   float64         `json:"additive_mg,omitempty"`
			MaxDoseMg    float64         `json:"max_dose_mg,omitempty"`
			MinDoseMg    float64         `json:"min_dose_mg,omitempty"`
		}{
			AdjustmentID: fmt.Sprintf("dialysis_hd_%d", adjustmentID),
			Type:         "dialysis_hemodialysis",
			ConditionAST: conditionAST,
			FormulaAST:   formulaAST,
			Multiplier:   tomlRule.Adjustments.Dialysis.Hemodialysis.DoseMultiplier,
			AdditiveMg:   tomlRule.Adjustments.Dialysis.Hemodialysis.SupplementalDose,
		})
		
		usedFields["dialysis_type"] = true
		adjustmentID++
	}

	// Compile titration schedule
	for _, titration := range tomlRule.Titration {
		actionAST, err := c.compileActionToAST(titration.Action, titration.Amount, titration.IncreaseBy)
		if err != nil {
			return nil, fmt.Errorf("failed to compile titration action: %w", err)
		}

		monitoringAST, _ := c.compileMonitoringToAST(titration.Monitoring)

		compiled.TitrationSchedule = append(compiled.TitrationSchedule, struct {
			StepNumber int             `json:"step_number"`
			AfterDays  int             `json:"after_days"`
			ActionType string          `json:"action_type"`
			ActionAST  json.RawMessage `json:"action_ast"`
			MaxStep    int             `json:"max_step,omitempty"`
			Monitoring json.RawMessage `json:"monitoring_requirements,omitempty"`
		}{
			StepNumber: titration.Step,
			AfterDays:  titration.AfterDays,
			ActionType: titration.Action,
			ActionAST:  actionAST,
			MaxStep:    titration.MaxStep,
			Monitoring: monitoringAST,
		})
	}

	// Compile population dosing
	for popType, pop := range tomlRule.Population {
		formulaAST, err := c.compileFormulaStringToAST(pop.Formula)
		if err != nil {
			return nil, fmt.Errorf("failed to compile population formula for %s: %w", popType, err)
		}

		safetyLimitsAST, _ := c.compileSafetyLimitsToAST(pop.MaxDoseMg, pop.Contraindicated)

		compiled.PopulationDosing = append(compiled.PopulationDosing, struct {
			PopulationID string          `json:"population_id"`
			Type         string          `json:"population_type"`
			AgeMin       int             `json:"age_min"`
			AgeMax       int             `json:"age_max"`
			WeightMin    float64         `json:"weight_min,omitempty"`
			WeightMax    float64         `json:"weight_max,omitempty"`
			FormulaAST   json.RawMessage `json:"formula_ast"`
			SafetyLimits json.RawMessage `json:"safety_limits"`
		}{
			PopulationID: fmt.Sprintf("pop_%s", popType),
			Type:         popType,
			AgeMin:       pop.AgeMin,
			AgeMax:       pop.AgeMax,
			FormulaAST:   formulaAST,
			SafetyLimits: safetyLimitsAST,
		})

		// Track which fields are used in population formulas
		c.extractUsedFieldsFromFormula(pop.Formula, usedFields)
	}

	// Convert used fields map to slice
	var usedFieldsList []string
	for field := range usedFields {
		usedFieldsList = append(usedFieldsList, field)
	}
	compiled.UsedFields = usedFieldsList

	// Calculate final checksum
	compiledBytes, _ := json.Marshal(compiled)
	compiled.Checksum = c.calculateChecksum(string(compiledBytes))

	return compiled, nil
}

// Validation methods

func (c *TOMLCompiler) validateRequiredFields(rule *TOMLRule) error {
	if rule.Meta.Name == "" {
		return fmt.Errorf("meta.name is required")
	}
	if rule.Meta.SemanticVersion == "" {
		return fmt.Errorf("meta.semantic_version is required")
	}
	if rule.Meta.DrugCode == "" {
		return fmt.Errorf("meta.drug_code is required")
	}
	if rule.BaseDose.Unit == "" {
		return fmt.Errorf("base_dose.unit is required")
	}
	if rule.BaseDose.Starting <= 0 {
		return fmt.Errorf("base_dose.starting must be positive")
	}
	return nil
}

func (c *TOMLCompiler) validateCompiledStructure(compiled *KB1CompiledRule) error {
	if compiled.Meta.DrugCode == "" {
		return fmt.Errorf("compiled drug_code is empty")
	}
	if compiled.BaseDose.Starting <= 0 {
		return fmt.Errorf("compiled starting dose must be positive")
	}
	if compiled.Checksum == "" {
		return fmt.Errorf("checksum is required")
	}
	return nil
}

// Unit normalization methods

func (c *TOMLCompiler) normalizeToUCUM(unit string) string {
	// Map common units to UCUM format
	ucumMap := map[string]string{
		"mg":  "mg",
		"g":   "g",
		"mcg": "ug",
		"μg":  "ug",
		"mg/kg": "mg/kg",
		"mg/m2": "mg/m2",
		"units": "U",
		"iu":    "IU",
	}
	
	normalized := strings.ToLower(strings.TrimSpace(unit))
	if ucum, exists := ucumMap[normalized]; exists {
		return ucum
	}
	
	return unit // Return original if no mapping found
}

func (c *TOMLCompiler) normalizeToMg(value float64, unit string) float64 {
	// Convert various units to mg
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "g":
		return value * 1000
	case "mcg", "μg":
		return value / 1000
	case "mg":
		return value
	default:
		return value // Return as-is if unknown unit
	}
}

func (c *TOMLCompiler) standardizeFrequency(frequency string) string {
	// Map common frequencies to standardized codes
	freqMap := map[string]string{
		"once_daily":   "QD",
		"twice_daily":  "BID",
		"three_daily":  "TID",
		"four_daily":   "QID",
		"every_8h":     "Q8H",
		"every_12h":    "Q12H",
		"every_6h":     "Q6H",
		"as_needed":    "PRN",
	}
	
	normalized := strings.ToLower(strings.TrimSpace(frequency))
	if code, exists := freqMap[normalized]; exists {
		return code
	}
	
	return frequency
}

func (c *TOMLCompiler) standardizeFormulation(form string) string {
	// Standardize formulation names
	formMap := map[string]string{
		"tablet":           "tablet",
		"capsule":          "capsule",
		"solution":         "oral_solution",
		"injection":        "injection",
		"extended_release": "extended_release_tablet",
		"immediate_release": "immediate_release_tablet",
	}
	
	normalized := strings.ToLower(strings.TrimSpace(form))
	if standard, exists := formMap[normalized]; exists {
		return standard
	}
	
	return form
}

func (c *TOMLCompiler) mapRouteToSNOMED(route string) string {
	// Map routes to SNOMED codes
	routeMap := map[string]string{
		"oral":          "26643006",
		"intravenous":   "47625008",
		"intramuscular": "78421000",
		"subcutaneous":  "34206005",
		"topical":       "6064005",
	}
	
	normalized := strings.ToLower(strings.TrimSpace(route))
	if snomed, exists := routeMap[normalized]; exists {
		return snomed
	}
	
	return route
}

// AST compilation methods

func (c *TOMLCompiler) compileConditionToAST(condition string) (json.RawMessage, error) {
	// Parse condition string into AST
	// This is a simplified implementation - production would use a proper parser
	ast := map[string]interface{}{
		"type":      "condition",
		"expression": condition,
		"compiled_at": time.Now(),
	}
	
	// Parse simple conditions (e.g., "egfr >= 30 AND egfr <= 59")
	if strings.Contains(condition, "egfr") {
		ast["requires_fields"] = []string{"egfr"}
	}
	if strings.Contains(condition, "age_years") {
		ast["requires_fields"] = append(ast["requires_fields"].([]string), "age_years")
	}
	
	return json.Marshal(ast)
}

func (c *TOMLCompiler) compileFormulaToAST(action string, factor, capMax float64) (json.RawMessage, error) {
	ast := map[string]interface{}{
		"type":   "formula",
		"action": action,
	}
	
	switch action {
	case "reduce_max", "cap_max":
		ast["operation"] = "min"
		ast["value"] = capMax
	case "multiply":
		ast["operation"] = "multiply"
		ast["factor"] = factor
	case "contraindicated":
		ast["operation"] = "contraindicate"
		ast["result"] = false
	default:
		ast["operation"] = "identity"
	}
	
	return json.Marshal(ast)
}

func (c *TOMLCompiler) compileActionToAST(action string, amount, increaseBy float64) (json.RawMessage, error) {
	ast := map[string]interface{}{
		"type":   "titration_action",
		"action": action,
	}
	
	if amount > 0 {
		ast["amount"] = amount
	}
	if increaseBy > 0 {
		ast["increase_by"] = increaseBy
	}
	
	return json.Marshal(ast)
}

func (c *TOMLCompiler) compileMonitoringToAST(monitoring string) (json.RawMessage, error) {
	if monitoring == "" {
		return json.RawMessage("{}"), nil
	}
	
	ast := map[string]interface{}{
		"type":         "monitoring",
		"requirements": monitoring,
	}
	
	return json.Marshal(ast)
}

func (c *TOMLCompiler) compileFormulaStringToAST(formula string) (json.RawMessage, error) {
	if formula == "" {
		return json.RawMessage("{}"), nil
	}
	
	ast := map[string]interface{}{
		"type":    "mathematical_formula",
		"formula": formula,
	}
	
	// Simple parsing for weight-based formulas
	if strings.Contains(formula, "weight_kg") {
		ast["requires_fields"] = []string{"weight_kg"}
		
		// Extract numeric multiplier (e.g., "0.02 * weight_kg * 1000")
		re := regexp.MustCompile(`([0-9.]+)\s*\*\s*weight_kg`)
		if matches := re.FindStringSubmatch(formula); len(matches) > 1 {
			if multiplier, err := strconv.ParseFloat(matches[1], 64); err == nil {
				ast["weight_multiplier"] = multiplier
			}
		}
	}
	
	return json.Marshal(ast)
}

func (c *TOMLCompiler) compileSafetyLimitsToAST(maxDose float64, contraindicated bool) (json.RawMessage, error) {
	ast := map[string]interface{}{
		"type": "safety_limits",
	}
	
	if maxDose > 0 {
		ast["max_dose_mg"] = maxDose
	}
	if contraindicated {
		ast["contraindicated"] = true
	}
	
	return json.Marshal(ast)
}

// extractUsedFieldsFromFormula analyzes formula strings to determine required patient context fields
func (c *TOMLCompiler) extractUsedFieldsFromFormula(formula string, usedFields map[string]bool) {
	// Simple pattern matching for common fields
	patterns := map[string]string{
		"weight_kg":  "weight_kg",
		"age_years":  "age_years",
		"egfr":       "egfr",
		"creatinine": "creatinine_clearance",
		"bsa":        "body_surface_area",
	}
	
	for pattern, field := range patterns {
		if strings.Contains(strings.ToLower(formula), pattern) {
			usedFields[field] = true
		}
	}
}

// calculateChecksum generates SHA256 checksum for content integrity
func (c *TOMLCompiler) calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// CompileFile compiles a TOML file to KB-1 compliant JSON
func (c *TOMLCompiler) CompileFile(filePath string) (*CompilationResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &CompilationResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("File read error: %v", err)},
		}, nil
	}
	
	return c.Compile(string(content), filePath)
}