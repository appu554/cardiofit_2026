// Package factstore provides the Completeness Checker for post-extraction
// quality assessment of SPL pipeline output.
//
// P3: Per-drug quality report that flags drugs with missing or insufficient output.
// Checks section coverage, minimum counts (soft warnings), MedDRA normalization
// rates, frequency coverage, and row extraction ratios.
package factstore

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// CompletenessChecker evaluates extraction quality for a drug's output.
type CompletenessChecker struct {
	log *logrus.Entry
}

// NewCompletenessChecker creates a completeness checker.
func NewCompletenessChecker(log *logrus.Entry) *CompletenessChecker {
	return &CompletenessChecker{
		log: log.WithField("component", "completeness-checker"),
	}
}

// CompletenessReport contains per-drug quality metrics.
type CompletenessReport struct {
	DrugName string `json:"drugName"`
	RxCUI    string `json:"rxcui"`

	// Section coverage
	SectionsCovered    []string `json:"sectionsCovered"`    // LOINC codes that produced facts
	SectionsMissing    []string `json:"sectionsMissing"`    // Expected sections with no facts
	SectionCoveragePct float64  `json:"sectionCoveragePct"` // % of expected sections covered

	// Fact counts by type
	FactCounts       map[string]int `json:"factCounts"`       // FactType → count
	TotalFacts       int            `json:"totalFacts"`
	FactTypesCovered int            `json:"factTypesCovered"` // How many of 6 types are present

	// Quality metrics
	MedDRAMatchRate  float64 `json:"meddraMatchRate"`  // % of SAFETY_SIGNAL facts with MedDRA PT code
	FrequencyCovRate float64 `json:"frequencyCovRate"` // % of SAFETY_SIGNAL facts with frequency data
	InteractionQual  float64 `json:"interactionQual"`  // % of INTERACTION facts with clinicalEffect

	// Row extraction
	TotalSourceRows     int     `json:"totalSourceRows"`
	ExtractedRows       int     `json:"extractedRows"`
	SkippedRows         int     `json:"skippedRows"`
	RowCoveragePct      float64 `json:"rowCoveragePct"`
	SkipReasonBreakdown map[string]int `json:"skipReasonBreakdown"` // reason → count

	// Method distribution
	StructuredCount int     `json:"structuredCount"`
	LLMCount        int     `json:"llmCount"`
	GrammarCount    int     `json:"grammarCount"`
	DeterministicPct float64 `json:"deterministicPct"` // (structured+grammar) / total

	// Warnings
	Warnings []string `json:"warnings"`
	Grade    string   `json:"grade"` // A, B, C, D, F
}

// expectedSections are the LOINC codes we expect to produce facts for most drugs.
// Not all drugs have all sections, so missing sections are soft warnings.
var expectedSections = map[string]string{
	"34084-4": "DOSAGE AND ADMINISTRATION",
	"34071-1": "WARNINGS",
	"43685-7": "WARNINGS AND PRECAUTIONS",
	"34073-7": "DRUG INTERACTIONS",
	"34068-7": "BOXED WARNING",
	"34088-5": "OVERDOSAGE",
	"34069-5": "HOW SUPPLIED",
}

// expectedSafetyLOINCs are LOINC codes expected to produce SAFETY_SIGNAL facts.
var expectedSafetyLOINCs = map[string]bool{
	"34084-4": true, // Dosage
	"34071-1": true, // Warnings
	"43685-7": true, // Warnings and Precautions
	"34068-7": true, // Boxed Warning
	"34072-9": true, // Contraindications
}

// Check evaluates completeness of extraction for a single drug.
// processedSectionCodes: LOINC codes of sections the pipeline processed (from routed sections).
// skipReasons: accumulated skip reasons from extractFromTables (reason → count).
func (cc *CompletenessChecker) Check(drugName, rxcui string, facts []*DerivedFact, totalSourceRows int, processedSectionCodes []string, skipReasons map[string]int) *CompletenessReport {
	report := &CompletenessReport{
		DrugName:            drugName,
		RxCUI:               rxcui,
		FactCounts:          make(map[string]int),
		SkipReasonBreakdown: make(map[string]int),
		TotalFacts:          len(facts),
		TotalSourceRows:     totalSourceRows,
		Warnings:            make([]string, 0),
	}

	if len(facts) == 0 {
		report.Warnings = append(report.Warnings, "CRITICAL: No facts extracted for drug")
		report.Grade = "F"
		return report
	}

	// Count facts by type and method
	sectionsSeen := make(map[string]bool)
	var safetyFacts, interactionFacts int
	var meddraMatched, withFrequency, withClinicalEffect int

	for _, f := range facts {
		report.FactCounts[f.FactType]++
		sectionsSeen[f.SourceSectionID] = true

		switch f.ExtractionMethod {
		case "STRUCTURED_PARSE":
			report.StructuredCount++
		case "LLM_FALLBACK":
			report.LLMCount++
		}

		// Parse fact data for quality metrics
		data := make(map[string]interface{})
		if err := json.Unmarshal(f.FactData, &data); err == nil {
			switch f.FactType {
			case "SAFETY_SIGNAL":
				safetyFacts++
				if pt, ok := data["meddraPT"].(string); ok && pt != "" {
					meddraMatched++
				}
				if freq, ok := data["frequency"].(string); ok && freq != "" {
					withFrequency++
				}
			case "INTERACTION":
				interactionFacts++
				if ce, ok := data["clinicalEffect"].(string); ok && ce != "" {
					withClinicalEffect++
				}
				// Check if this was grammar-extracted (via sourcePhrase containing "Grammar:")
				if sp, ok := data["sourcePhrase"].(string); ok && strings.HasPrefix(sp, "Grammar:") {
					report.GrammarCount++
					report.StructuredCount++ // Grammar counts as structured
				}
			}
		}
	}

	// Section coverage: compare processed LOINC codes against expected sections
	report.FactTypesCovered = len(report.FactCounts)
	processedSet := make(map[string]bool, len(processedSectionCodes))
	for _, code := range processedSectionCodes {
		processedSet[code] = true
	}
	for loinc := range expectedSections {
		if processedSet[loinc] {
			report.SectionsCovered = append(report.SectionsCovered, loinc)
		} else {
			report.SectionsMissing = append(report.SectionsMissing, loinc)
		}
	}
	if len(expectedSections) > 0 {
		report.SectionCoveragePct = float64(len(report.SectionsCovered)) / float64(len(expectedSections)) * 100
	}

	// Skip reason breakdown from pipeline extraction
	if skipReasons != nil {
		for reason, count := range skipReasons {
			report.SkipReasonBreakdown[reason] = count
		}
	}

	// Quality metrics
	if safetyFacts > 0 {
		report.MedDRAMatchRate = float64(meddraMatched) / float64(safetyFacts) * 100
		report.FrequencyCovRate = float64(withFrequency) / float64(safetyFacts) * 100
	}
	if interactionFacts > 0 {
		report.InteractionQual = float64(withClinicalEffect) / float64(interactionFacts) * 100
	}

	// Row coverage
	report.ExtractedRows = len(facts) // Simplified: 1 fact ≈ 1 row
	if totalSourceRows > 0 {
		report.RowCoveragePct = float64(report.ExtractedRows) / float64(totalSourceRows) * 100
		report.SkippedRows = totalSourceRows - report.ExtractedRows
	}

	// Deterministic ratio
	deterministicTotal := report.StructuredCount // Grammar already counted in StructuredCount
	if report.TotalFacts > 0 {
		report.DeterministicPct = float64(deterministicTotal) / float64(report.TotalFacts) * 100
	}

	// Generate warnings
	cc.generateWarnings(report, safetyFacts, interactionFacts)

	// Assign grade
	report.Grade = cc.assignGrade(report)

	// Log the report
	cc.log.WithFields(logrus.Fields{
		"drug":             drugName,
		"totalFacts":       report.TotalFacts,
		"factTypes":        report.FactTypesCovered,
		"sectionCoverage":  fmt.Sprintf("%.0f%% (%d/%d)", report.SectionCoveragePct, len(report.SectionsCovered), len(expectedSections)),
		"meddraMatchRate":  fmt.Sprintf("%.1f%%", report.MedDRAMatchRate),
		"frequencyCovRate": fmt.Sprintf("%.1f%%", report.FrequencyCovRate),
		"deterministicPct": fmt.Sprintf("%.1f%%", report.DeterministicPct),
		"grade":            report.Grade,
	}).Info("Completeness check complete")

	return report
}

// generateWarnings creates soft warnings based on thresholds.
func (cc *CompletenessChecker) generateWarnings(report *CompletenessReport, safetyFacts, interactionFacts int) {
	// Minimum count warnings (soft — some drugs legitimately have few)
	if safetyFacts < 5 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Below typical range: only %d SAFETY_SIGNAL facts (typical: 5-50)", safetyFacts))
	}
	if interactionFacts < 2 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Below typical range: only %d INTERACTION facts (typical: 2-20)", interactionFacts))
	}

	// Quality warnings
	if report.MedDRAMatchRate < 50 && safetyFacts > 0 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Low MedDRA match rate: %.1f%% — dictionary gap may need expansion", report.MedDRAMatchRate))
	}
	if report.FrequencyCovRate < 20 && safetyFacts > 0 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Low frequency coverage: %.1f%% — prose annotation may not be reaching AE terms", report.FrequencyCovRate))
	}
	if report.DeterministicPct < 50 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("High LLM dependency: only %.1f%% deterministic extraction", report.DeterministicPct))
	}

	// Fact type coverage
	if report.FactTypesCovered < 2 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Limited fact type coverage: only %d/6 types extracted", report.FactTypesCovered))
	}
}

// assignGrade based on overall quality.
func (cc *CompletenessChecker) assignGrade(report *CompletenessReport) string {
	score := 0.0

	// MedDRA match rate (0-25 points)
	if report.MedDRAMatchRate >= 80 {
		score += 25
	} else if report.MedDRAMatchRate >= 50 {
		score += 15
	} else if report.MedDRAMatchRate > 0 {
		score += 5
	}

	// Frequency coverage (0-25 points)
	if report.FrequencyCovRate >= 50 {
		score += 25
	} else if report.FrequencyCovRate >= 20 {
		score += 15
	} else if report.FrequencyCovRate > 0 {
		score += 5
	}

	// Deterministic ratio (0-25 points)
	if report.DeterministicPct >= 85 {
		score += 25
	} else if report.DeterministicPct >= 50 {
		score += 15
	} else if report.DeterministicPct > 0 {
		score += 5
	}

	// Fact count (0-25 points)
	if report.TotalFacts >= 20 {
		score += 25
	} else if report.TotalFacts >= 10 {
		score += 15
	} else if report.TotalFacts > 0 {
		score += 5
	}

	switch {
	case score >= 90:
		return "A"
	case score >= 70:
		return "B"
	case score >= 50:
		return "C"
	case score >= 30:
		return "D"
	default:
		return "F"
	}
}
