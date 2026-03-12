package fda

import (
	"fmt"
	"strings"

	"kb-1-drug-rules/internal/models"
)

// =============================================================================
// EXTRACTION QUALITY FRAMEWORK
// =============================================================================
// This module provides confidence scoring, anomaly detection, and risk
// classification for FDA SPL extraction. Pattern-based NLP extraction is
// inherently risky - this framework makes those risks visible and actionable.

// ExtractionResult wraps extracted data with quality metrics
type ExtractionResult struct {
	// Extracted data
	Dosing *models.DosingRules `json:"dosing,omitempty"`
	Safety *models.SafetyInfo  `json:"safety,omitempty"`

	// Quality assessment
	Quality *ExtractionQuality `json:"quality"`

	// Risk classification
	RiskLevel     RiskLevel `json:"risk_level"`
	RiskFactors   []string  `json:"risk_factors,omitempty"`
	RequiresReview bool     `json:"requires_review"`

	// Raw context for manual review
	RawDosageText       string `json:"raw_dosage_text,omitempty"`
	RawSafetyText       string `json:"raw_safety_text,omitempty"`
	RawContraText       string `json:"raw_contra_text,omitempty"`
}

// ExtractionQuality contains per-field confidence scores and issues
type ExtractionQuality struct {
	// Overall confidence (0-100)
	OverallConfidence int `json:"overall_confidence"`

	// Per-field confidence
	DoseConfidence         int `json:"dose_confidence"`
	UnitConfidence         int `json:"unit_confidence"`
	FrequencyConfidence    int `json:"frequency_confidence"`
	MaxDoseConfidence      int `json:"max_dose_confidence"`
	RenalConfidence        int `json:"renal_confidence"`
	HepaticConfidence      int `json:"hepatic_confidence"`
	ContraConfidence       int `json:"contra_confidence"`

	// Missing mandatory fields
	MissingFields []string `json:"missing_fields,omitempty"`

	// Detected anomalies
	Anomalies []Anomaly `json:"anomalies,omitempty"`

	// Warnings (non-blocking issues)
	Warnings []string `json:"warnings,omitempty"`

	// Extraction statistics
	PatternsMatched   int `json:"patterns_matched"`
	PatternsAttempted int `json:"patterns_attempted"`
}

// Anomaly represents a detected data quality issue
type Anomaly struct {
	Type        AnomalyType `json:"type"`
	Severity    string      `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Description string      `json:"description"`
	Field       string      `json:"field"`
	Value       string      `json:"value,omitempty"`
	Expected    string      `json:"expected,omitempty"`
}

// AnomalyType categorizes anomaly types
type AnomalyType string

const (
	AnomalyDoseExceedsMax       AnomalyType = "DOSE_EXCEEDS_MAX"
	AnomalyPediatricExceedsAdult AnomalyType = "PEDIATRIC_EXCEEDS_ADULT"
	AnomalyRenalExceedsNormal   AnomalyType = "RENAL_EXCEEDS_NORMAL"
	AnomalyUnitMismatch         AnomalyType = "UNIT_MISMATCH"
	AnomalyMissingMaxDose       AnomalyType = "MISSING_MAX_DOSE"
	AnomalyNoDoseExtracted      AnomalyType = "NO_DOSE_EXTRACTED"
	AnomalyConflictingDoses     AnomalyType = "CONFLICTING_DOSES"
	AnomalySuspiciousValue      AnomalyType = "SUSPICIOUS_VALUE"
)

// RiskLevel for clinical risk classification
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "CRITICAL" // Requires CMO + Pharmacist sign-off
	RiskLevelHigh     RiskLevel = "HIGH"     // Requires Pharmacist sign-off
	RiskLevelStandard RiskLevel = "STANDARD" // Auto-approval possible
	RiskLevelLow      RiskLevel = "LOW"      // Auto-approval
)

// =============================================================================
// HIGH-RISK DRUG CLASSIFICATION
// =============================================================================

// HighRiskDrugClasses defines drug classes that require mandatory manual review
var HighRiskDrugClasses = map[string]RiskLevel{
	// CRITICAL - Can kill quickly with small errors
	"anticoagulant":         RiskLevelCritical,
	"warfarin":              RiskLevelCritical,
	"heparin":               RiskLevelCritical,
	"insulin":               RiskLevelCritical,
	"chemotherapy":          RiskLevelCritical,
	"antineoplastic":        RiskLevelCritical,
	"opioid":                RiskLevelCritical,
	"narcotic":              RiskLevelCritical,
	"digoxin":               RiskLevelCritical,
	"potassium chloride":    RiskLevelCritical,
	"neuromuscular blocker": RiskLevelCritical,

	// HIGH - Narrow therapeutic index or serious consequences
	"immunosuppressant": RiskLevelHigh,
	"antiepileptic":     RiskLevelHigh,
	"lithium":           RiskLevelHigh,
	"aminoglycoside":    RiskLevelHigh,
	"vancomycin":        RiskLevelHigh,
	"methotrexate":      RiskLevelHigh,
	"theophylline":      RiskLevelHigh,
	"phenytoin":         RiskLevelHigh,
	"carbamazepine":     RiskLevelHigh,
	"valproic":          RiskLevelHigh,
}

// HighRiskDrugNames maps specific drug names to risk levels
var HighRiskDrugNames = map[string]RiskLevel{
	// Anticoagulants
	"warfarin":      RiskLevelCritical,
	"heparin":       RiskLevelCritical,
	"enoxaparin":    RiskLevelCritical,
	"rivaroxaban":   RiskLevelCritical,
	"apixaban":      RiskLevelCritical,
	"dabigatran":    RiskLevelCritical,
	"edoxaban":      RiskLevelCritical,

	// Insulins
	"insulin":       RiskLevelCritical,
	"glargine":      RiskLevelCritical,
	"lispro":        RiskLevelCritical,
	"aspart":        RiskLevelCritical,
	"detemir":       RiskLevelCritical,

	// Opioids
	"morphine":      RiskLevelCritical,
	"fentanyl":      RiskLevelCritical,
	"oxycodone":     RiskLevelCritical,
	"hydromorphone": RiskLevelCritical,
	"methadone":     RiskLevelCritical,

	// Cardiac
	"digoxin":       RiskLevelCritical,
	"amiodarone":    RiskLevelHigh,
	"sotalol":       RiskLevelHigh,
	"flecainide":    RiskLevelHigh,

	// Narrow TI
	"lithium":       RiskLevelHigh,
	"phenytoin":     RiskLevelHigh,
	"carbamazepine": RiskLevelHigh,
	"valproate":     RiskLevelHigh,
	"theophylline":  RiskLevelHigh,
	"cyclosporine":  RiskLevelHigh,
	"tacrolimus":    RiskLevelHigh,
	"sirolimus":     RiskLevelHigh,
}

// =============================================================================
// QUALITY VALIDATOR
// =============================================================================

// QualityValidator validates extraction quality and detects anomalies
type QualityValidator struct {
	extractor *Extractor
}

// NewQualityValidator creates a new quality validator
func NewQualityValidator() *QualityValidator {
	return &QualityValidator{
		extractor: NewExtractor(),
	}
}

// ValidateExtraction performs comprehensive validation on extracted data
func (v *QualityValidator) ValidateExtraction(
	dosing *models.DosingRules,
	safety *models.SafetyInfo,
	drugName string,
	drugClass string,
) *ExtractionQuality {
	quality := &ExtractionQuality{
		PatternsAttempted: 12, // Total extraction patterns we try
	}

	// Count successful extractions
	if dosing != nil {
		if dosing.Adult != nil && len(dosing.Adult.Standard) > 0 {
			quality.PatternsMatched++
			quality.DoseConfidence = 70
		}
		if dosing.Adult != nil && dosing.Adult.MaxDaily > 0 {
			quality.PatternsMatched++
			quality.MaxDoseConfidence = 80
		}
		if dosing.Renal != nil && len(dosing.Renal.Adjustments) > 0 {
			quality.PatternsMatched++
			quality.RenalConfidence = 60
		}
		if dosing.Hepatic != nil {
			quality.PatternsMatched++
			quality.HepaticConfidence = 60
		}
	}

	if safety != nil {
		if len(safety.Contraindications) > 0 {
			quality.PatternsMatched++
			quality.ContraConfidence = 70
		}
	}

	// Check for missing mandatory fields
	v.checkMissingFields(quality, dosing, safety)

	// Detect anomalies
	v.detectAnomalies(quality, dosing, safety)

	// Calculate overall confidence
	quality.OverallConfidence = v.calculateOverallConfidence(quality)

	return quality
}

// checkMissingFields identifies missing mandatory fields
func (v *QualityValidator) checkMissingFields(
	quality *ExtractionQuality,
	dosing *models.DosingRules,
	safety *models.SafetyInfo,
) {
	// Dose is mandatory
	if dosing == nil || dosing.Adult == nil || len(dosing.Adult.Standard) == 0 {
		if dosing == nil || dosing.WeightBased == nil || dosing.WeightBased.DosePerKg == 0 {
			if dosing == nil || dosing.BSABased == nil || dosing.BSABased.DosePerM2 == 0 {
				quality.MissingFields = append(quality.MissingFields, "dosing")
				quality.Warnings = append(quality.Warnings, "No dosing information extracted - requires manual review")
			}
		}
	}

	// Unit validation
	if dosing != nil && dosing.Adult != nil {
		for _, dose := range dosing.Adult.Standard {
			if dose.Unit == "" {
				quality.MissingFields = append(quality.MissingFields, "dose_unit")
				quality.Warnings = append(quality.Warnings, "Dose extracted without unit - DANGEROUS")
			}
			if dose.Frequency == "" {
				quality.MissingFields = append(quality.MissingFields, "dose_frequency")
				quality.Warnings = append(quality.Warnings, "Dose extracted without frequency")
			}
		}
	}

	// Max dose strongly recommended
	if dosing != nil && dosing.Adult != nil {
		if dosing.Adult.MaxDaily == 0 && dosing.Adult.MaxSingle == 0 {
			quality.Warnings = append(quality.Warnings, "No maximum dose extracted - manual verification recommended")
		}
	}

	// Weight-based drugs MUST have max dose
	if dosing != nil && dosing.WeightBased != nil && dosing.WeightBased.DosePerKg > 0 {
		if dosing.WeightBased.MaxDose == 0 {
			quality.MissingFields = append(quality.MissingFields, "weight_based_max_dose")
			quality.Warnings = append(quality.Warnings, "Weight-based dose without max cap - CRITICAL for safety")
		}
	}
}

// detectAnomalies identifies data quality issues
func (v *QualityValidator) detectAnomalies(
	quality *ExtractionQuality,
	dosing *models.DosingRules,
	safety *models.SafetyInfo,
) {
	if dosing == nil {
		return
	}

	// Check: Adult dose should not exceed max daily
	if dosing.Adult != nil && dosing.Adult.MaxDaily > 0 {
		for _, dose := range dosing.Adult.Standard {
			if dose.Dose > dosing.Adult.MaxDaily {
				quality.Anomalies = append(quality.Anomalies, Anomaly{
					Type:        AnomalyDoseExceedsMax,
					Severity:    "CRITICAL",
					Description: fmt.Sprintf("Extracted dose (%.2f) exceeds max daily (%.2f)", dose.Dose, dosing.Adult.MaxDaily),
					Field:       "adult.standard.dose",
					Value:       fmt.Sprintf("%.2f", dose.Dose),
					Expected:    fmt.Sprintf("<= %.2f", dosing.Adult.MaxDaily),
				})
			}
		}
	}

	// Check: Pediatric dose should generally not exceed adult dose
	if dosing.Pediatric != nil && dosing.Adult != nil && len(dosing.Adult.Standard) > 0 {
		// This check would need age range specific validation
		// Flagging for awareness
		quality.Warnings = append(quality.Warnings, "Pediatric dosing present - verify age-appropriate limits")
	}

	// Check: Renal impaired dose should not exceed normal dose
	if dosing.Renal != nil && len(dosing.Renal.Adjustments) > 0 {
		for _, adj := range dosing.Renal.Adjustments {
			if adj.DosePercent > 100 {
				quality.Anomalies = append(quality.Anomalies, Anomaly{
					Type:        AnomalyRenalExceedsNormal,
					Severity:    "HIGH",
					Description: fmt.Sprintf("Renal adjustment (%.0f%%) exceeds normal dose", adj.DosePercent),
					Field:       "renal.adjustment",
					Value:       fmt.Sprintf("%.0f%%", adj.DosePercent),
					Expected:    "<= 100%",
				})
			}
		}
	}

	// Check: Suspicious values (likely extraction errors)
	if dosing.Adult != nil {
		for _, dose := range dosing.Adult.Standard {
			// Doses > 10000 mg are suspicious
			if dose.Unit == "mg" && dose.Dose > 10000 {
				quality.Anomalies = append(quality.Anomalies, Anomaly{
					Type:        AnomalySuspiciousValue,
					Severity:    "HIGH",
					Description: fmt.Sprintf("Unusually high dose: %.0f mg - verify extraction", dose.Dose),
					Field:       "adult.standard.dose",
					Value:       fmt.Sprintf("%.0f mg", dose.Dose),
				})
			}
			// Doses < 0.001 mg are suspicious (except mcg)
			if dose.Unit == "mg" && dose.Dose > 0 && dose.Dose < 0.001 {
				quality.Anomalies = append(quality.Anomalies, Anomaly{
					Type:        AnomalySuspiciousValue,
					Severity:    "HIGH",
					Description: fmt.Sprintf("Unusually low dose: %.6f mg - possibly mcg?", dose.Dose),
					Field:       "adult.standard.dose",
					Value:       fmt.Sprintf("%.6f mg", dose.Dose),
				})
			}
		}
	}

	// Check: Multiple conflicting doses without indication
	if dosing.Adult != nil && len(dosing.Adult.Standard) > 3 {
		hasIndications := false
		for _, dose := range dosing.Adult.Standard {
			if dose.Indication != "" {
				hasIndications = true
				break
			}
		}
		if !hasIndications {
			quality.Anomalies = append(quality.Anomalies, Anomaly{
				Type:        AnomalyConflictingDoses,
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Multiple doses extracted (%d) without indications - unclear which to use", len(dosing.Adult.Standard)),
				Field:       "adult.standard",
			})
		}
	}
}

// calculateOverallConfidence computes overall extraction confidence
func (v *QualityValidator) calculateOverallConfidence(quality *ExtractionQuality) int {
	// Start with base confidence based on pattern success rate
	if quality.PatternsAttempted == 0 {
		return 0
	}
	base := (quality.PatternsMatched * 100) / quality.PatternsAttempted

	// Reduce for missing mandatory fields
	base -= len(quality.MissingFields) * 15

	// Reduce for anomalies
	for _, anomaly := range quality.Anomalies {
		switch anomaly.Severity {
		case "CRITICAL":
			base -= 30
		case "HIGH":
			base -= 20
		case "MEDIUM":
			base -= 10
		case "LOW":
			base -= 5
		}
	}

	// Reduce for warnings
	base -= len(quality.Warnings) * 5

	// Clamp to 0-100
	if base < 0 {
		base = 0
	}
	if base > 100 {
		base = 100
	}

	return base
}

// =============================================================================
// RISK CLASSIFIER
// =============================================================================

// ClassifyRisk determines the clinical risk level for a drug
func ClassifyRisk(drugName, drugClass string, safety *models.SafetyInfo) (RiskLevel, []string) {
	var riskFactors []string
	maxRisk := RiskLevelLow

	drugNameLower := strings.ToLower(drugName)
	drugClassLower := strings.ToLower(drugClass)

	// Check drug name against high-risk list
	for name, risk := range HighRiskDrugNames {
		if strings.Contains(drugNameLower, name) {
			if riskGreater(risk, maxRisk) {
				maxRisk = risk
			}
			riskFactors = append(riskFactors, fmt.Sprintf("High-risk drug: %s", name))
		}
	}

	// Check drug class against high-risk list
	for class, risk := range HighRiskDrugClasses {
		if strings.Contains(drugClassLower, class) {
			if riskGreater(risk, maxRisk) {
				maxRisk = risk
			}
			riskFactors = append(riskFactors, fmt.Sprintf("High-risk class: %s", class))
		}
	}

	// Safety flags increase risk
	if safety != nil {
		if safety.BlackBoxWarning {
			if riskGreater(RiskLevelHigh, maxRisk) {
				maxRisk = RiskLevelHigh
			}
			riskFactors = append(riskFactors, "Black box warning present")
		}
		if safety.NarrowTherapeuticIndex {
			if riskGreater(RiskLevelHigh, maxRisk) {
				maxRisk = RiskLevelHigh
			}
			riskFactors = append(riskFactors, "Narrow therapeutic index")
		}
		if safety.HighAlertDrug {
			if riskGreater(RiskLevelHigh, maxRisk) {
				maxRisk = RiskLevelHigh
			}
			riskFactors = append(riskFactors, "High-alert drug indicator")
		}
	}

	// Default to STANDARD if no specific risks but still needs some review
	if maxRisk == RiskLevelLow && len(riskFactors) == 0 {
		maxRisk = RiskLevelStandard
	}

	return maxRisk, riskFactors
}

// riskGreater returns true if a > b in terms of risk severity
func riskGreater(a, b RiskLevel) bool {
	order := map[RiskLevel]int{
		RiskLevelLow:      0,
		RiskLevelStandard: 1,
		RiskLevelHigh:     2,
		RiskLevelCritical: 3,
	}
	return order[a] > order[b]
}

// RequiresManualReview determines if a drug requires manual pharmacist review
func RequiresManualReview(risk RiskLevel, quality *ExtractionQuality) bool {
	// CRITICAL and HIGH always require review
	if risk == RiskLevelCritical || risk == RiskLevelHigh {
		return true
	}

	// Low confidence requires review
	if quality != nil && quality.OverallConfidence < 50 {
		return true
	}

	// Any critical anomaly requires review
	if quality != nil {
		for _, anomaly := range quality.Anomalies {
			if anomaly.Severity == "CRITICAL" {
				return true
			}
		}
	}

	// Missing mandatory fields requires review
	if quality != nil && len(quality.MissingFields) > 0 {
		return true
	}

	return false
}

// =============================================================================
// EXTRACTION WITH QUALITY
// =============================================================================

// ExtractWithQuality performs extraction with full quality assessment
func (v *QualityValidator) ExtractWithQuality(
	doc *SPLDocument,
	drugName string,
	drugClass string,
) (*ExtractionResult, error) {
	result := &ExtractionResult{}

	// Perform extraction
	dosing, err := v.extractor.ExtractDosingRules(doc)
	if err != nil {
		return nil, fmt.Errorf("dosing extraction failed: %w", err)
	}
	result.Dosing = dosing

	safety, err := v.extractor.ExtractSafetyInfo(doc)
	if err != nil {
		return nil, fmt.Errorf("safety extraction failed: %w", err)
	}
	result.Safety = safety

	// Store raw text for manual review
	dosageSection := v.extractor.parser.GetSection(doc, SectionDosageAdmin)
	if dosageSection != nil {
		text := v.extractor.parser.GetSectionText(dosageSection)
		if len(text) > 5000 {
			text = text[:5000] + "...[truncated]"
		}
		result.RawDosageText = text
	}

	contraSection := v.extractor.parser.GetSection(doc, SectionContraindications)
	if contraSection != nil {
		text := v.extractor.parser.GetSectionText(contraSection)
		if len(text) > 2000 {
			text = text[:2000] + "...[truncated]"
		}
		result.RawContraText = text
	}

	// Validate extraction quality
	result.Quality = v.ValidateExtraction(dosing, safety, drugName, drugClass)

	// Classify risk
	result.RiskLevel, result.RiskFactors = ClassifyRisk(drugName, drugClass, safety)

	// Determine if manual review required
	result.RequiresReview = RequiresManualReview(result.RiskLevel, result.Quality)

	return result, nil
}
