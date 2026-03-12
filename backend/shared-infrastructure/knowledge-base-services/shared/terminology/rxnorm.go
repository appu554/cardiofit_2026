// Package terminology provides ontology-grounded terminology normalization services.
//
// rxnorm.go: Drug validation and RxCUI correction using RxNav-in-a-Box.
//
// This file implements the fix for Issue 1: FK Constraint Failures.
//
// Problem:
//
//	SPL XML contains wrong/outdated RxCUI values that don't match drug_master.
//	Example: SPL says Lithium=5521, but drug_master has Lithium=6448.
//	Result: Facts can't project from derived_facts to clinical_facts (FK violation).
//
// Solution:
//
//	Validate every RxCUI via RxNav-in-a-Box before storage.
//	If RxCUI is wrong, look up correct one by drug name.
//	Store the CANONICAL RxCUI that matches drug_master.
package terminology

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources"
	"github.com/cardiofit/shared/datasources/rxnav"
)

// RxNormNormalizer implements DrugNormalizer using RxNav-in-a-Box.
// It validates RxCUIs and corrects them when they don't match the drug name.
type RxNormNormalizer struct {
	client *rxnav.Client
	log    *logrus.Entry
}

// RxNormNormalizerConfig contains configuration for the normalizer.
type RxNormNormalizerConfig struct {
	// RxNavConfig is the RxNav client configuration.
	// Use rxnav.LocalConfig() for RxNav-in-a-Box (localhost:4000).
	RxNavConfig rxnav.Config

	// Logger for logging operations.
	Logger *logrus.Entry
}

// DefaultRxNormConfig returns configuration for RxNav-in-a-Box Docker instance.
// This is the recommended configuration for Phase 3.
func DefaultRxNormConfig() RxNormNormalizerConfig {
	return RxNormNormalizerConfig{
		RxNavConfig: rxnav.LocalConfig(), // localhost:4000
	}
}

// NewRxNormNormalizer creates a new drug normalizer using RxNav.
func NewRxNormNormalizer(config RxNormNormalizerConfig) (*RxNormNormalizer, error) {
	client := rxnav.NewClient(config.RxNavConfig)

	log := config.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	return &RxNormNormalizer{
		client: client,
		log:    log.WithField("component", "rxnorm_normalizer"),
	}, nil
}

// ValidateAndNormalize checks RxCUI against RxNav and corrects if wrong.
//
// This is the KEY FUNCTION that fixes Issue 1: FK Constraint Failures.
//
// Algorithm:
//  1. Try to validate the given RxCUI via RxNav
//  2. If valid, check if the drug name matches what we expect
//  3. If mismatch (or invalid), look up correct RxCUI by drug name
//  4. Return the canonical RxCUI that will match drug_master
//
// Example:
//
//	Input:  rxcui="5521", drugName="Lithium"
//	Step 1: RxNav says 5521 = "hydroxychloroquine" (WRONG!)
//	Step 2: Name doesn't match "Lithium"
//	Step 3: Look up "Lithium" → RxCUI 6448
//	Output: CanonicalRxCUI="6448", WasCorrected=true
func (n *RxNormNormalizer) ValidateAndNormalize(ctx context.Context, rxcui, drugName string) (*NormalizedDrug, error) {
	result := &NormalizedDrug{
		OriginalRxCUI: rxcui,
		Source:        "RXNAV_LOCAL",
	}

	// Clean up drug name for comparison
	cleanedName := cleanDrugName(drugName)

	n.log.WithFields(logrus.Fields{
		"rxcui":     rxcui,
		"drug_name": cleanedName,
	}).Debug("Validating RxCUI")

	// Step 1: Try to validate the given RxCUI via RxNav.
	// If the RxCUI is valid in RxNav, we trust it — even if the SPL title was garbage.
	// This handles cases where extractDrugNameFromTitle returns noise but the
	// pipeline_runner already has the correct ingredient RxCUI from InitialScopeDrugs.
	if rxcui != "" {
		drug, err := n.client.GetDrugByRxCUI(ctx, rxcui)
		if err == nil && drug != nil && drug.Name != "" {
			// RxCUI exists in RxNav — check if drug name matches
			if cleanedName == "" || n.drugNameMatches(drug.Name, cleanedName, drug.Ingredients) {
				// MATCH (or drug name was garbage so we trust RxCUI)
				n.log.WithFields(logrus.Fields{
					"rxcui":         rxcui,
					"drug_name":     drug.Name,
					"was_corrected": false,
				}).Debug("RxCUI validated successfully")

				result.CanonicalName = drug.Name
				result.CanonicalRxCUI = rxcui
				result.WasCorrected = false
				result.Confidence = 1.0
				result.TTY = drug.TTY
				result.GenericName = n.extractGenericName(drug)
				return result, nil
			}

			// MISMATCH: RxCUI is valid but wrong drug
			// Example: 5521 is hydroxychloroquine, not lithium
			n.log.WithFields(logrus.Fields{
				"rxcui":       rxcui,
				"actual_drug": drug.Name,
				"expected":    cleanedName,
			}).Warn("RxCUI valid but wrong drug - will lookup correct RxCUI")
		}
	}

	// Step 2: RxCUI invalid or wrong - lookup by drug name.
	// Need a valid drug name to search.
	if cleanedName == "" {
		return nil, fmt.Errorf("drug name is empty or invalid: %q", drugName)
	}

	correctRxCUI, err := n.lookupRxCUIByName(ctx, cleanedName)
	if err != nil {
		return nil, fmt.Errorf("cannot find RxCUI for %q: %w", cleanedName, err)
	}

	// Step 3: Get canonical drug info for the correct RxCUI
	correctDrug, err := n.client.GetDrugByRxCUI(ctx, correctRxCUI)
	if err != nil {
		return nil, fmt.Errorf("cannot get drug info for RxCUI %s: %w", correctRxCUI, err)
	}

	// Step 4: Resolve to ingredient-level RxCUI (TTY=IN) for drug_master FK compatibility.
	// The name lookup may return SCD/SBD/SCDFP/GPCK codes, but drug_master uses IN codes.
	// Example: "lithium" lookup → SCD 92678 ("lithium Oral Tablet"), but drug_master has IN 6448 ("lithium")
	ingredientRxCUI := correctRxCUI
	ingredientName := correctDrug.Name
	if correctDrug.TTY != "IN" && correctDrug.TTY != "MIN" {
		resolved := false

		// Strategy 1: Direct ingredient lookup (works for SCD/SBD)
		ingredients, ingErr := n.client.GetIngredients(ctx, correctRxCUI)
		if ingErr == nil && len(ingredients) > 0 {
			ingredientRxCUI = ingredients[0].RxCUI
			ingredientName = ingredients[0].Name
			resolved = true
		}

		// Strategy 2: For SCDFP/SBDFP, go through SCD first, then get ingredient
		if !resolved && (correctDrug.TTY == "SCDFP" || correctDrug.TTY == "SBDFP" || correctDrug.TTY == "SBDG" || correctDrug.TTY == "SCDG") {
			related, relErr := n.client.GetRelatedByType(ctx, correctRxCUI, "SCD")
			if relErr == nil && len(related) > 0 {
				scdIngredients, scdErr := n.client.GetIngredients(ctx, related[0].RxCUI)
				if scdErr == nil && len(scdIngredients) > 0 {
					ingredientRxCUI = scdIngredients[0].RxCUI
					ingredientName = scdIngredients[0].Name
					resolved = true
				}
			}
		}

		// Strategy 3: Extract base ingredient name and look up directly
		// e.g., "amiodarone hydrochloride Oral Tablet" → "amiodarone" → RxCUI 703
		if !resolved {
			baseName := extractBaseName(ingredientName)
			if baseName != "" {
				inRxCUI, inErr := n.client.GetRxCUIByName(ctx, baseName)
				if inErr == nil && inRxCUI != "" {
					inDrug, inDrugErr := n.client.GetDrugByRxCUI(ctx, inRxCUI)
					if inDrugErr == nil && inDrug != nil && (inDrug.TTY == "IN" || inDrug.TTY == "MIN") {
						ingredientRxCUI = inRxCUI
						ingredientName = inDrug.Name
						resolved = true
					}
				}
			}
		}

		if resolved {
			n.log.WithFields(logrus.Fields{
				"product_rxcui":    correctRxCUI,
				"product_tty":      correctDrug.TTY,
				"ingredient_rxcui": ingredientRxCUI,
				"ingredient_name":  ingredientName,
			}).Debug("Resolved to ingredient-level RxCUI for drug_master FK")
		}
	}

	result.CanonicalName = ingredientName
	result.CanonicalRxCUI = ingredientRxCUI
	result.WasCorrected = true
	result.Confidence = 0.95
	result.TTY = correctDrug.TTY
	result.GenericName = ingredientName

	n.log.WithFields(logrus.Fields{
		"original_rxcui": rxcui,
		"correct_rxcui":  ingredientRxCUI,
		"drug_name":      ingredientName,
	}).Info("RxCUI CORRECTED")

	return result, nil
}

// GetCanonicalRxCUI looks up the correct RxCUI by drug name only.
func (n *RxNormNormalizer) GetCanonicalRxCUI(ctx context.Context, drugName string) (string, error) {
	cleanedName := cleanDrugName(drugName)
	if cleanedName == "" {
		return "", fmt.Errorf("drug name is empty or invalid: %q", drugName)
	}

	return n.lookupRxCUIByName(ctx, cleanedName)
}

// lookupRxCUIByName tries multiple strategies to find the RxCUI for a drug name.
func (n *RxNormNormalizer) lookupRxCUIByName(ctx context.Context, drugName string) (string, error) {
	// Strategy 1: Exact name lookup
	rxcui, err := n.client.GetRxCUIByName(ctx, drugName)
	if err == nil && rxcui != "" {
		return rxcui, nil
	}

	// Strategy 2: Approximate term search
	// This handles variations like "Lithium Carbonate" vs "Lithium"
	drugs, err := n.client.SearchDrugs(ctx, drugName, 5)
	if err == nil && len(drugs) > 0 {
		// Find best match by name similarity
		for _, drug := range drugs {
			if n.drugNameMatches(drug.Name, drugName, nil) {
				n.log.WithFields(logrus.Fields{
					"query":     drugName,
					"match":     drug.Name,
					"rxcui":     drug.RxCUI,
				}).Debug("Found via approximate search")
				return drug.RxCUI, nil
			}
		}
		// If no exact match, use first result
		if drugs[0].RxCUI != "" {
			return drugs[0].RxCUI, nil
		}
	}

	// Strategy 3: Try common variations
	variations := n.generateNameVariations(drugName)
	for _, variant := range variations {
		rxcui, err := n.client.GetRxCUIByName(ctx, variant)
		if err == nil && rxcui != "" {
			n.log.WithFields(logrus.Fields{
				"original":  drugName,
				"variant":   variant,
				"rxcui":     rxcui,
			}).Debug("Found via name variation")
			return rxcui, nil
		}
	}

	return "", fmt.Errorf("no RxCUI found for drug name: %s", drugName)
}

// drugNameMatches checks if two drug names refer to the same drug.
// Handles variations like brand vs generic, with/without salt forms.
func (n *RxNormNormalizer) drugNameMatches(actualName, expectedName string, ingredients []string) bool {
	actual := strings.ToLower(actualName)
	expected := strings.ToLower(expectedName)

	// Exact match
	if actual == expected {
		return true
	}

	// Contains match (handles "Lithium Carbonate" matching "Lithium")
	if strings.Contains(actual, expected) || strings.Contains(expected, actual) {
		return true
	}

	// Check against ingredients
	for _, ing := range ingredients {
		ingLower := strings.ToLower(ing)
		if strings.Contains(ingLower, expected) || strings.Contains(expected, ingLower) {
			return true
		}
	}

	// Extract base name (remove common suffixes like "HCl", "Sodium", etc.)
	actualBase := extractBaseName(actual)
	expectedBase := extractBaseName(expected)
	if actualBase != "" && expectedBase != "" && actualBase == expectedBase {
		return true
	}

	return false
}

// generateNameVariations creates common variations of a drug name.
func (n *RxNormNormalizer) generateNameVariations(name string) []string {
	var variations []string
	lower := strings.ToLower(name)

	// Common salt forms
	salts := []string{"", " hydrochloride", " hcl", " sodium", " potassium", " sulfate", " carbonate"}
	for _, salt := range salts {
		if salt == "" {
			continue
		}
		// Add salt if not present
		if !strings.HasSuffix(lower, salt) {
			variations = append(variations, name+salt)
		}
		// Remove salt if present
		if strings.HasSuffix(lower, salt) {
			variations = append(variations, strings.TrimSuffix(name, salt))
		}
	}

	return variations
}

// extractGenericName extracts the generic/ingredient name from drug info.
func (n *RxNormNormalizer) extractGenericName(drug *datasources.RxNormDrug) string {
	if drug == nil {
		return ""
	}
	if len(drug.Ingredients) > 0 {
		return drug.Ingredients[0]
	}
	return drug.Name
}

// cleanDrugName cleans up a drug name for comparison.
// Removes common noise patterns from SPL titles.
func cleanDrugName(name string) string {
	if name == "" {
		return ""
	}

	// Remove common SPL preamble noise
	noisePatterns := []string{
		"these highlights do not include all information",
		"highlights of prescribing information",
		"full prescribing information",
		"WARNING:",
		"BOXED WARNING",
	}

	lower := strings.ToLower(name)
	for _, pattern := range noisePatterns {
		if strings.Contains(lower, pattern) {
			// This is garbage text, not a drug name
			return ""
		}
	}

	// Extract just the drug name part (before any dashes, parentheses, etc.)
	// Handle formats like "LITHIUM CARBONATE- lithium carbonate tablet"
	if idx := strings.Index(name, "-"); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}
	if idx := strings.Index(name, "("); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}

	// Remove dosage information
	// Handle "Aspirin 81mg" → "Aspirin"
	words := strings.Fields(name)
	var cleanWords []string
	for _, word := range words {
		// Skip if it looks like a dosage (contains numbers followed by mg/ml/etc.)
		if containsDosagePattern(word) {
			continue
		}
		cleanWords = append(cleanWords, word)
	}

	return strings.TrimSpace(strings.Join(cleanWords, " "))
}

// extractBaseName removes common suffixes and dose forms to get the base drug name.
// Example: "amiodarone hydrochloride Oral Tablet" → "amiodarone"
func extractBaseName(name string) string {
	result := strings.ToLower(strings.TrimSpace(name))

	// Remove dose forms first (these appear at the end)
	doseForms := []string{
		" oral tablet", " oral capsule", " oral solution", " oral suspension",
		" injectable solution", " injection", " tablet", " capsule",
		" solution", " suspension", " cream", " ointment", " patch",
		" extended release", " delayed release",
	}
	for _, df := range doseForms {
		result = strings.TrimSuffix(result, df)
	}

	// Remove salt forms
	salts := []string{
		" hydrochloride", " hcl", " sodium", " potassium",
		" sulfate", " carbonate", " acetate", " phosphate",
		" tartrate", " citrate", " maleate", " fumarate",
		" mesylate", " besylate", " succinate", " lactate",
	}
	for _, salt := range salts {
		result = strings.TrimSuffix(result, salt)
	}

	// Remove dosage numbers (e.g., "200 mg")
	words := strings.Fields(result)
	var clean []string
	for _, w := range words {
		if containsDosagePattern(w) {
			continue
		}
		// Skip standalone units
		if w == "mg" || w == "ml" || w == "mcg" || w == "g" {
			continue
		}
		clean = append(clean, w)
	}

	return strings.TrimSpace(strings.Join(clean, " "))
}

// containsDosagePattern checks if a word looks like a dosage.
func containsDosagePattern(word string) bool {
	lower := strings.ToLower(word)
	dosageUnits := []string{"mg", "ml", "mcg", "g", "iu", "units", "%"}
	for _, unit := range dosageUnits {
		if strings.HasSuffix(lower, unit) {
			// Check if there's a number before the unit
			for _, c := range lower[:len(lower)-len(unit)] {
				if c >= '0' && c <= '9' {
					return true
				}
			}
		}
	}
	return false
}

// Ensure RxNormNormalizer implements DrugNormalizer
var _ DrugNormalizer = (*RxNormNormalizer)(nil)
