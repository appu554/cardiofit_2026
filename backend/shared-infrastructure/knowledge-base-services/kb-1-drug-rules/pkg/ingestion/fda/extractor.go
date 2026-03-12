package fda

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kb-1-drug-rules/internal/models"
)

// Extractor extracts dosing and safety information from SPL sections
type Extractor struct {
	parser *Parser
}

// NewExtractor creates a new dosing extractor
func NewExtractor() *Extractor {
	return &Extractor{
		parser: NewParser(),
	}
}

// =============================================================================
// MAIN EXTRACTION METHODS
// =============================================================================

// ExtractDosingRules extracts dosing rules from SPL document
func (e *Extractor) ExtractDosingRules(doc *SPLDocument) (*models.DosingRules, error) {
	rules := &models.DosingRules{
		PrimaryMethod: "FIXED", // Default, may be updated based on content
	}

	// Extract from Dosage and Administration section
	dosageSection := e.parser.GetSection(doc, SectionDosageAdmin)
	if dosageSection != nil {
		text := e.parser.GetSectionText(dosageSection)
		rules.Adult = e.extractAdultDosing(text)

		// Check for weight-based dosing indicators
		if e.containsWeightBasedDosing(text) {
			rules.PrimaryMethod = "WEIGHT_BASED"
			rules.WeightBased = e.extractWeightBasedDosing(text)
		}

		// Check for BSA-based dosing (common in oncology)
		if e.containsBSABasedDosing(text) {
			rules.PrimaryMethod = "BSA_BASED"
			rules.BSABased = e.extractBSABasedDosing(text)
		}

		// Extract titration schedule if present
		rules.Titration = e.extractTitrationSchedule(text)
	}

	// Extract from Use in Specific Populations section
	specificPopSection := e.parser.GetSection(doc, SectionUseSpecificPop)
	if specificPopSection != nil {
		text := e.parser.GetSectionText(specificPopSection)
		rules.Renal = e.extractRenalDosing(text)
		rules.Hepatic = e.extractHepaticDosing(text)
		rules.Geriatric = e.extractGeriatricDosing(text)
		rules.Pediatric = e.extractPediatricDosing(text)
	}

	// Check specific sections for organ impairment
	renalSection := e.parser.GetSection(doc, SectionRenalImpairment)
	if renalSection != nil && rules.Renal == nil {
		text := e.parser.GetSectionText(renalSection)
		rules.Renal = e.extractRenalDosing(text)
	}

	hepaticSection := e.parser.GetSection(doc, SectionHepaticImpairment)
	if hepaticSection != nil && rules.Hepatic == nil {
		text := e.parser.GetSectionText(hepaticSection)
		rules.Hepatic = e.extractHepaticDosing(text)
	}

	// Check pediatric use section
	pediatricSection := e.parser.GetSection(doc, SectionPediatricUse)
	if pediatricSection != nil && rules.Pediatric == nil {
		text := e.parser.GetSectionText(pediatricSection)
		rules.Pediatric = e.extractPediatricDosing(text)
	}

	// Check geriatric use section
	geriatricSection := e.parser.GetSection(doc, SectionGeriatricUse)
	if geriatricSection != nil && rules.Geriatric == nil {
		text := e.parser.GetSectionText(geriatricSection)
		rules.Geriatric = e.extractGeriatricDosing(text)
	}

	return rules, nil
}

// ExtractSafetyInfo extracts safety information from SPL document
func (e *Extractor) ExtractSafetyInfo(doc *SPLDocument) (*models.SafetyInfo, error) {
	safety := &models.SafetyInfo{}

	// Black Box Warning
	blackBoxSection := e.parser.GetSection(doc, SectionBlackBox)
	if blackBoxSection != nil {
		safety.BlackBoxWarning = true
		safety.BlackBoxText = e.parser.GetSectionText(blackBoxSection)
	}

	// Contraindications
	contraSection := e.parser.GetSection(doc, SectionContraindications)
	if contraSection != nil {
		safety.Contraindications = e.extractBulletPoints(
			e.parser.GetSectionText(contraSection),
		)
	}

	// Drug Interactions
	interactionsSection := e.parser.GetSection(doc, SectionDrugInteractions)
	if interactionsSection != nil {
		text := e.parser.GetSectionText(interactionsSection)
		safety.MajorInteractions = e.extractMajorInteractions(text)
	}

	// Warnings and Precautions
	warningsSection := e.parser.GetSection(doc, SectionWarnings)
	if warningsSection != nil {
		text := e.parser.GetSectionText(warningsSection)
		safety.Monitoring = e.extractMonitoringRequirements(text)

		// Check for high-alert drug indicators
		safety.HighAlertDrug = e.isHighAlertDrug(text)
		safety.NarrowTherapeuticIndex = e.isNarrowTherapeuticIndex(text)
	}

	return safety, nil
}

// =============================================================================
// ADULT DOSING EXTRACTION
// =============================================================================

// extractAdultDosing extracts adult dosing from text
func (e *Extractor) extractAdultDosing(text string) *models.AdultDosing {
	dosing := &models.AdultDosing{}

	// Extract standard doses using regex patterns
	doses := e.extractDosePatterns(text)
	if len(doses) > 0 {
		dosing.Standard = doses
	}

	// Extract max daily dose
	maxDailyPattern := regexp.MustCompile(`(?i)(?:maximum|max)\s+(?:daily\s+)?(?:dose|dosage)[:\s]+(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)(?:\s+per\s+day)?`)
	if match := maxDailyPattern.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.MaxDaily = val
			dosing.MaxUnit = normalizeUnit(match[2])
		}
	}

	// Extract max single dose
	maxSinglePattern := regexp.MustCompile(`(?i)(?:maximum|max)\s+single\s+(?:dose|dosage)[:\s]+(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)`)
	if match := maxSinglePattern.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.MaxSingle = val
		}
	}

	// Alternative max dose pattern
	if dosing.MaxDaily == 0 {
		altMaxPattern := regexp.MustCompile(`(?i)(?:not\s+(?:to\s+)?exceed|should\s+not\s+exceed)\s+(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)(?:\s+(?:per\s+)?(?:day|daily))?`)
		if match := altMaxPattern.FindStringSubmatch(text); len(match) > 1 {
			if val, err := strconv.ParseFloat(match[1], 64); err == nil {
				dosing.MaxDaily = val
				dosing.MaxUnit = normalizeUnit(match[2])
			}
		}
	}

	return dosing
}

// extractDosePatterns extracts dose patterns from text
func (e *Extractor) extractDosePatterns(text string) []models.StandardDose {
	var doses []models.StandardDose
	text = strings.ToLower(text)

	// Pattern definitions with their frequencies
	patterns := []struct {
		regex *regexp.Regexp
		freq  string
	}{
		// Fixed dose patterns
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+(?:once\s+)?daily`), "DAILY"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+twice\s+daily`), "BID"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+two\s+times\s+(?:a\s+)?day`), "BID"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+three\s+times\s+(?:a\s+)?day`), "TID"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+four\s+times\s+(?:a\s+)?day`), "QID"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+every\s+(\d+)\s+hours?`), "Q${3}H"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+(?:orally|by\s+mouth|po)`), "DAILY"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+at\s+bedtime`), "QHS"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+once\s+weekly`), "WEEKLY"},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+every\s+week`), "WEEKLY"},
	}

	seen := make(map[string]bool)

	for _, p := range patterns {
		matches := p.regex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				if val, err := strconv.ParseFloat(match[1], 64); err == nil {
					unit := normalizeUnit(match[2])
					key := fmt.Sprintf("%.2f-%s-%s", val, unit, p.freq)

					if !seen[key] {
						seen[key] = true
						doses = append(doses, models.StandardDose{
							Dose:      val,
							Unit:      unit,
							Frequency: p.freq,
						})
					}
				}
			}
		}
	}

	// Also try to extract dose ranges (e.g., "10-20 mg daily")
	rangePattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:to|-)\s*(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?)\s+(?:once\s+)?daily`)
	rangeMatches := rangePattern.FindAllStringSubmatch(text, -1)
	for _, match := range rangeMatches {
		if len(match) >= 4 {
			minVal, _ := strconv.ParseFloat(match[1], 64)
			maxVal, _ := strconv.ParseFloat(match[2], 64)
			unit := normalizeUnit(match[3])

			key := fmt.Sprintf("%.2f-%.2f-%s-DAILY", minVal, maxVal, unit)
			if !seen[key] {
				seen[key] = true
				doses = append(doses, models.StandardDose{
					DoseMin:   minVal,
					DoseMax:   maxVal,
					Unit:      unit,
					Frequency: "DAILY",
				})
			}
		}
	}

	return doses
}

// =============================================================================
// WEIGHT-BASED DOSING EXTRACTION
// =============================================================================

// containsWeightBasedDosing checks if text contains weight-based dosing indicators
func (e *Extractor) containsWeightBasedDosing(text string) bool {
	patterns := []string{
		`mg/kg`,
		`mg per kg`,
		`milligrams? per kilogram`,
		`based on (?:body )?weight`,
		`per kilogram`,
		`mcg/kg`,
	}

	text = strings.ToLower(text)
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, text); matched {
			return true
		}
	}
	return false
}

// extractWeightBasedDosing extracts weight-based dosing
func (e *Extractor) extractWeightBasedDosing(text string) *models.WeightBasedDosing {
	dosing := &models.WeightBasedDosing{}
	text = strings.ToLower(text)

	// Pattern: "X mg/kg" or "X mg per kg"
	patterns := []string{
		`(\d+(?:\.\d+)?)\s*(mg|mcg)/kg`,
		`(\d+(?:\.\d+)?)\s*(mg|mcg)\s+per\s+kg`,
		`(\d+(?:\.\d+)?)\s*(mg|mcg)\s+per\s+kilogram`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 2 {
			if val, err := strconv.ParseFloat(match[1], 64); err == nil {
				dosing.DosePerKg = val
				dosing.Unit = normalizeUnit(match[2]) + "/kg"
				break
			}
		}
	}

	// Extract max dose cap
	maxPattern := regexp.MustCompile(`(?i)(?:maximum|max|not\s+to\s+exceed)[:\s]+(\d+(?:\.\d+)?)\s*(mg|g|mcg)`)
	if match := maxPattern.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.MaxDose = val
		}
	}

	// Check for ideal body weight usage
	if strings.Contains(text, "ideal body weight") || strings.Contains(text, "ibw") {
		dosing.UseIdealWeight = true
	}

	// Check for adjusted body weight usage
	if strings.Contains(text, "adjusted body weight") || strings.Contains(text, "abw") {
		dosing.UseAdjustedWeight = true
	}

	// Extract frequency if mentioned
	freqPatterns := map[string]string{
		"once daily":        "DAILY",
		"twice daily":       "BID",
		"three times daily": "TID",
		"every 12 hours":    "Q12H",
		"every 8 hours":     "Q8H",
		"every 6 hours":     "Q6H",
	}

	for pattern, freq := range freqPatterns {
		if strings.Contains(text, pattern) {
			dosing.Frequency = freq
			break
		}
	}

	return dosing
}

// =============================================================================
// BSA-BASED DOSING EXTRACTION
// =============================================================================

// containsBSABasedDosing checks for BSA-based dosing indicators
func (e *Extractor) containsBSABasedDosing(text string) bool {
	patterns := []string{
		`mg/m[²2]`,
		`per m[²2]`,
		`body surface area`,
		`bsa`,
	}

	text = strings.ToLower(text)
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, text); matched {
			return true
		}
	}
	return false
}

// extractBSABasedDosing extracts BSA-based dosing
func (e *Extractor) extractBSABasedDosing(text string) *models.BSABasedDosing {
	dosing := &models.BSABasedDosing{}
	text = strings.ToLower(text)

	// Pattern: "X mg/m²" or "X mg per m²"
	patterns := []string{
		`(\d+(?:\.\d+)?)\s*(mg|mcg)/m[²2]`,
		`(\d+(?:\.\d+)?)\s*(mg|mcg)\s+per\s+m[²2]`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 2 {
			if val, err := strconv.ParseFloat(match[1], 64); err == nil {
				dosing.DosePerM2 = val
				dosing.Unit = normalizeUnit(match[2]) + "/m²"
				break
			}
		}
	}

	// Extract max absolute dose
	maxPattern := regexp.MustCompile(`(?:maximum|max|cap(?:ped)?)\s+(?:at\s+)?(\d+(?:\.\d+)?)\s*(mg|g)`)
	if match := maxPattern.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.MaxAbsoluteDose = val
		}
	}

	// Check for BSA cap
	bsaCapPattern := regexp.MustCompile(`bsa\s+(?:cap(?:ped)?|limit(?:ed)?)\s+(?:at|to)\s+(\d+(?:\.\d+)?)\s*m[²2]`)
	if match := bsaCapPattern.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.CappedAtBSA = val
		}
	}

	return dosing
}

// =============================================================================
// ORGAN IMPAIRMENT EXTRACTION
// =============================================================================

// extractRenalDosing extracts renal adjustment information
func (e *Extractor) extractRenalDosing(text string) *models.RenalDosing {
	text = strings.ToLower(text)

	// Check if renal impairment is mentioned
	if !strings.Contains(text, "renal") && !strings.Contains(text, "kidney") &&
		!strings.Contains(text, "creatinine") && !strings.Contains(text, "gfr") &&
		!strings.Contains(text, "crcl") {
		return nil
	}

	dosing := &models.RenalDosing{
		AdjustmentBasis: "eGFR",
	}

	// Determine basis (GFR vs CrCl)
	if strings.Contains(text, "creatinine clearance") || strings.Contains(text, "crcl") {
		dosing.AdjustmentBasis = "CrCl"
	}

	// Extract GFR/CrCl-based adjustments
	adjustments := []models.RenalAdjustmentTier{}

	// Common GFR threshold patterns
	gfrPatterns := []struct {
		pattern string
		minGFR  float64
		maxGFR  float64
	}{
		{`gfr\s*(?:>|≥|>=|greater\s+than)\s*90`, 90, 999},
		{`gfr\s*60\s*(?:to|-)\s*89`, 60, 89},
		{`gfr\s*(?:>|≥|>=|greater\s+than)\s*60`, 60, 999},
		{`gfr\s*45\s*(?:to|-)\s*59`, 45, 59},
		{`gfr\s*30\s*(?:to|-)\s*(?:59|60|44|45)`, 30, 59},
		{`gfr\s*30\s*(?:to|-)\s*(?:44|45)`, 30, 44},
		{`gfr\s*15\s*(?:to|-)\s*(?:29|30)`, 15, 29},
		{`gfr\s*(?:<|≤|<=|less\s+than)\s*30`, 0, 30},
		{`gfr\s*(?:<|≤|<=|less\s+than)\s*15`, 0, 15},
		{`(?:mild|moderate|severe)\s+renal\s+impairment`, -1, -1}, // Will be handled separately
	}

	for _, p := range gfrPatterns {
		if p.minGFR < 0 {
			continue // Skip qualitative patterns
		}
		if matched, _ := regexp.MatchString(p.pattern, text); matched {
			tier := models.RenalAdjustmentTier{
				MinGFR: p.minGFR,
				MaxGFR: p.maxGFR,
			}

			// Try to extract dose adjustment for this tier
			contextPattern := regexp.MustCompile(p.pattern + `[^.]*?(\d+)%`)
			if match := contextPattern.FindStringSubmatch(text); len(match) > 1 {
				if pct, err := strconv.ParseFloat(match[1], 64); err == nil {
					tier.DosePercent = pct
				}
			}

			adjustments = append(adjustments, tier)
		}
	}

	// Check for qualitative descriptions
	if strings.Contains(text, "mild renal impairment") {
		adjustments = append(adjustments, models.RenalAdjustmentTier{
			MinGFR: 60,
			MaxGFR: 89,
			Notes:  "Mild renal impairment",
		})
	}
	if strings.Contains(text, "moderate renal impairment") {
		adjustments = append(adjustments, models.RenalAdjustmentTier{
			MinGFR: 30,
			MaxGFR: 59,
			Notes:  "Moderate renal impairment",
		})
	}
	if strings.Contains(text, "severe renal impairment") {
		tier := models.RenalAdjustmentTier{
			MinGFR: 15,
			MaxGFR: 29,
			Notes:  "Severe renal impairment",
		}
		if strings.Contains(text, "contraindicated") || strings.Contains(text, "not recommended") {
			tier.Avoid = true
		}
		adjustments = append(adjustments, tier)
	}

	// Check for dialysis
	if strings.Contains(text, "dialysis") || strings.Contains(text, "esrd") ||
		strings.Contains(text, "end-stage renal") {
		tier := models.RenalAdjustmentTier{
			MinGFR: 0,
			MaxGFR: 15,
			Notes:  "ESRD/Dialysis",
		}
		if strings.Contains(text, "dialyzable") {
			tier.Dialyzable = true
		}
		if strings.Contains(text, "supplement") {
			// Try to extract supplement dose
			suppPattern := regexp.MustCompile(`supplement(?:ary|al)?\s+dose[:\s]+(\d+(?:\.\d+)?)\s*(mg|mcg)`)
			if match := suppPattern.FindStringSubmatch(text); len(match) > 1 {
				if val, err := strconv.ParseFloat(match[1], 64); err == nil {
					tier.SupplementDose = val
				}
			}
		}
		adjustments = append(adjustments, tier)
	}

	// Check for contraindication
	if (strings.Contains(text, "contraindicated") || strings.Contains(text, "not recommended")) &&
		strings.Contains(text, "renal") {
		// Find which level is contraindicated
		if strings.Contains(text, "severe") {
			for i := range adjustments {
				if adjustments[i].MinGFR < 30 {
					adjustments[i].Avoid = true
				}
			}
		}
	}

	if len(adjustments) == 0 {
		return nil
	}

	dosing.Adjustments = adjustments
	return dosing
}

// extractHepaticDosing extracts hepatic adjustment information
func (e *Extractor) extractHepaticDosing(text string) *models.HepaticDosing {
	text = strings.ToLower(text)

	if !strings.Contains(text, "hepatic") && !strings.Contains(text, "liver") &&
		!strings.Contains(text, "child-pugh") && !strings.Contains(text, "cirrhosis") {
		return nil
	}

	dosing := &models.HepaticDosing{}

	// Check for Child-Pugh class mentions
	if strings.Contains(text, "child-pugh a") || strings.Contains(text, "mild hepatic") {
		dosing.ChildPughA = &models.HepaticAdjustmentTier{
			Notes: "Child-Pugh A / Mild hepatic impairment",
		}
		if strings.Contains(text, "no adjustment") || strings.Contains(text, "no dosage adjustment") {
			dosing.ChildPughA.DosePercent = 100
		}
	}

	if strings.Contains(text, "child-pugh b") || strings.Contains(text, "moderate hepatic") {
		dosing.ChildPughB = &models.HepaticAdjustmentTier{
			Notes: "Child-Pugh B / Moderate hepatic impairment",
		}
		// Try to extract dose reduction
		if strings.Contains(text, "50%") || strings.Contains(text, "half") {
			dosing.ChildPughB.DosePercent = 50
		}
		if strings.Contains(text, "reduce") {
			pctPattern := regexp.MustCompile(`reduce[^.]*?(\d+)%`)
			if match := pctPattern.FindStringSubmatch(text); len(match) > 1 {
				if pct, err := strconv.ParseFloat(match[1], 64); err == nil {
					dosing.ChildPughB.DosePercent = 100 - pct
				}
			}
		}
	}

	if strings.Contains(text, "child-pugh c") || strings.Contains(text, "severe hepatic") {
		dosing.ChildPughC = &models.HepaticAdjustmentTier{
			Notes: "Child-Pugh C / Severe hepatic impairment",
		}
		if strings.Contains(text, "contraindicated") || strings.Contains(text, "not recommended") ||
			strings.Contains(text, "avoid") {
			dosing.ChildPughC.Avoid = true
		}
	}

	// If no specific class mentioned but hepatic impairment discussed
	if dosing.ChildPughA == nil && dosing.ChildPughB == nil && dosing.ChildPughC == nil {
		if strings.Contains(text, "caution") {
			dosing.Notes = "Use with caution in hepatic impairment"
		}
		if strings.Contains(text, "contraindicated") {
			dosing.ChildPughC = &models.HepaticAdjustmentTier{
				Avoid: true,
				Notes: "Contraindicated in hepatic impairment",
			}
		}
	}

	return dosing
}

// extractGeriatricDosing extracts geriatric dosing information
func (e *Extractor) extractGeriatricDosing(text string) *models.GeriatricDosing {
	text = strings.ToLower(text)

	if !strings.Contains(text, "geriatric") && !strings.Contains(text, "elderly") &&
		!strings.Contains(text, "older") && !strings.Contains(text, "65 years") {
		return nil
	}

	dosing := &models.GeriatricDosing{}

	// Check for start low, go slow
	if strings.Contains(text, "start") && (strings.Contains(text, "low") || strings.Contains(text, "lowest")) {
		dosing.StartLow = true
		dosing.Notes = "Start at lowest effective dose in elderly patients"
	}

	// Check for dose reduction
	reductionPattern := regexp.MustCompile(`(?:reduce|decrease)[^.]*?(\d+)%`)
	if match := reductionPattern.FindStringSubmatch(text); len(match) > 1 {
		if pct, err := strconv.ParseFloat(match[1], 64); err == nil {
			dosing.DoseReduction = pct
		}
	}

	// Check for avoidance
	if strings.Contains(text, "avoid") && strings.Contains(text, "elderly") {
		dosing.AvoidInElderly = true
	}

	// Check for Beers criteria mention
	if strings.Contains(text, "beers") {
		dosing.BeersListStatus = "Listed"
	}

	// General caution
	if strings.Contains(text, "caution") && dosing.Notes == "" {
		dosing.Notes = "Use with caution in elderly patients"
	}

	return dosing
}

// extractPediatricDosing extracts pediatric dosing information
func (e *Extractor) extractPediatricDosing(text string) *models.PediatricDosing {
	text = strings.ToLower(text)

	if !strings.Contains(text, "pediatric") && !strings.Contains(text, "children") &&
		!strings.Contains(text, "child") && !strings.Contains(text, "adolescent") {
		return nil
	}

	dosing := &models.PediatricDosing{}

	// Check if contraindicated
	if strings.Contains(text, "not recommended") || strings.Contains(text, "contraindicated") ||
		strings.Contains(text, "not established") || strings.Contains(text, "not indicated") {
		dosing.Contraindicated = true
		dosing.Notes = "Safety and efficacy not established in pediatric patients"
	}

	// Extract minimum age
	agePatterns := []struct {
		pattern string
		months  int
	}{
		{`(?:≥|>=|age|aged?)\s*18\s*years?`, 216},
		{`(?:≥|>=|age|aged?)\s*12\s*years?`, 144},
		{`(?:≥|>=|age|aged?)\s*6\s*years?`, 72},
		{`(?:≥|>=|age|aged?)\s*2\s*years?`, 24},
		{`(?:≥|>=|age|aged?)\s*1\s*year`, 12},
		{`(?:≥|>=|age|aged?)\s*6\s*months?`, 6},
		{`(?:≥|>=|age|aged?)\s*(\d+)\s*months?`, -1}, // Will extract value
	}

	for _, ap := range agePatterns {
		if ap.months >= 0 {
			if matched, _ := regexp.MatchString(ap.pattern, text); matched {
				dosing.MinAgeMonths = ap.months
				break
			}
		} else {
			re := regexp.MustCompile(ap.pattern)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				if months, err := strconv.Atoi(match[1]); err == nil {
					dosing.MinAgeMonths = months
					break
				}
			}
		}
	}

	// Check if weight-based
	if e.containsWeightBasedDosing(text) {
		dosing.UseWeight = true
	}

	return dosing
}

// =============================================================================
// TITRATION EXTRACTION
// =============================================================================

// extractTitrationSchedule extracts titration schedule if present
func (e *Extractor) extractTitrationSchedule(text string) *models.TitrationSchedule {
	text = strings.ToLower(text)

	if !strings.Contains(text, "titrat") && !strings.Contains(text, "increase") &&
		!strings.Contains(text, "escalat") && !strings.Contains(text, "adjust") {
		return nil
	}

	schedule := &models.TitrationSchedule{}

	// Try to extract target
	targetPattern := regexp.MustCompile(`target\s+(?:dose\s+)?(?:of\s+)?(\d+(?:\.\d+)?)\s*(mg|mcg)`)
	if match := targetPattern.FindStringSubmatch(text); len(match) > 1 {
		schedule.Target = match[1] + " " + match[2]
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			schedule.MaxDose = val
		}
	}

	// Extract steps if mentioned
	stepPattern := regexp.MustCompile(`(?:step|week|day)\s*(\d+)[:\s]+(\d+(?:\.\d+)?)\s*(mg|mcg)`)
	matches := stepPattern.FindAllStringSubmatch(text, -1)
	for i, match := range matches {
		if len(match) >= 4 {
			if val, err := strconv.ParseFloat(match[2], 64); err == nil {
				step := models.TitrationStep{
					StepNumber: i + 1,
					Dose:       val,
					Unit:       normalizeUnit(match[3]),
				}
				schedule.Steps = append(schedule.Steps, step)
			}
		}
	}

	// Look for interval information
	if strings.Contains(text, "weekly") || strings.Contains(text, "each week") {
		for i := range schedule.Steps {
			schedule.Steps[i].DurationDays = 7
		}
	} else if strings.Contains(text, "every 2 weeks") || strings.Contains(text, "biweekly") {
		for i := range schedule.Steps {
			schedule.Steps[i].DurationDays = 14
		}
	}

	if len(schedule.Steps) == 0 && schedule.Target == "" {
		return nil
	}

	return schedule
}

// =============================================================================
// SAFETY EXTRACTION
// =============================================================================

// extractMajorInteractions extracts major drug interactions
func (e *Extractor) extractMajorInteractions(text string) []string {
	var interactions []string

	// Look for contraindicated combinations
	contraPatterns := []string{
		`contraindicated with ([^.]+)`,
		`do not use with ([^.]+)`,
		`avoid (?:concomitant|concurrent) use (?:with|of) ([^.]+)`,
	}

	text = strings.ToLower(text)

	for _, pattern := range contraPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				interaction := strings.TrimSpace(match[1])
				if interaction != "" && len(interaction) < 100 {
					interactions = append(interactions, interaction)
				}
			}
		}
	}

	return interactions
}

// extractMonitoringRequirements extracts monitoring requirements
func (e *Extractor) extractMonitoringRequirements(text string) []string {
	var monitoring []string
	text = strings.ToLower(text)

	// Common monitoring requirements
	monitorPatterns := map[string]string{
		"renal function":    "Monitor renal function",
		"liver function":    "Monitor liver function tests",
		"hepatic function":  "Monitor hepatic function",
		"blood pressure":    "Monitor blood pressure",
		"potassium":         "Monitor serum potassium",
		"electrolytes":      "Monitor electrolytes",
		"blood glucose":     "Monitor blood glucose",
		"complete blood":    "Monitor complete blood count",
		"cbc":               "Monitor CBC",
		"inr":               "Monitor INR",
		"ecg":               "Monitor ECG",
		"qt interval":       "Monitor QT interval",
		"drug level":        "Monitor drug levels",
		"serum concentration": "Monitor serum concentrations",
	}

	for pattern, description := range monitorPatterns {
		if strings.Contains(text, pattern) {
			monitoring = append(monitoring, description)
		}
	}

	return monitoring
}

// isHighAlertDrug checks for high-alert drug indicators
func (e *Extractor) isHighAlertDrug(text string) bool {
	text = strings.ToLower(text)

	highAlertIndicators := []string{
		"high-alert",
		"high alert",
		"narrow therapeutic",
		"significant toxicity",
		"life-threatening",
		"serious adverse",
		"fatal",
		"death",
		"black box",
	}

	for _, indicator := range highAlertIndicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}
	return false
}

// isNarrowTherapeuticIndex checks for narrow therapeutic index indicators
func (e *Extractor) isNarrowTherapeuticIndex(text string) bool {
	text = strings.ToLower(text)

	ntiIndicators := []string{
		"narrow therapeutic",
		"therapeutic drug monitoring",
		"tdm",
		"serum concentration",
		"blood level",
		"drug level",
	}

	for _, indicator := range ntiIndicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}
	return false
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// extractBulletPoints extracts bullet points from text
func (e *Extractor) extractBulletPoints(text string) []string {
	var points []string

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "•") || strings.HasPrefix(line, "-") ||
			strings.HasPrefix(line, "*") {
			point := strings.TrimLeft(line, "•-* ")
			point = strings.TrimSpace(point)
			if point != "" && len(point) < 200 {
				points = append(points, point)
			}
		}
	}

	return points
}

// normalizeUnit normalizes dose unit strings
func normalizeUnit(unit string) string {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch unit {
	case "mg", "milligram", "milligrams":
		return "mg"
	case "g", "gram", "grams":
		return "g"
	case "mcg", "microgram", "micrograms", "µg":
		return "mcg"
	case "unit", "units", "iu":
		return "units"
	case "ml", "milliliter", "milliliters":
		return "mL"
	default:
		return unit
	}
}

