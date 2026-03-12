package safety

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// getTeratogenicEffectsAsStrings converts interface{} TeratogenicEffects to []string
// Handles both simple []string and complex []map structures from YAML
func getTeratogenicEffectsAsStrings(effects interface{}) []string {
	if effects == nil {
		return nil
	}

	// Try direct []string
	if strSlice, ok := effects.([]string); ok {
		return strSlice
	}

	// Try []interface{} (common YAML unmarshal result)
	if ifaceSlice, ok := effects.([]interface{}); ok {
		result := make([]string, 0, len(ifaceSlice))
		for _, item := range ifaceSlice {
			switch v := item.(type) {
			case string:
				result = append(result, v)
			case map[string]interface{}:
				// Complex structure - extract category or effects
				if cat, ok := v["category"].(string); ok {
					result = append(result, cat)
				}
				if effectsList, ok := v["effects"].([]interface{}); ok {
					for _, e := range effectsList {
						if s, ok := e.(string); ok {
							result = append(result, s)
						}
					}
				}
			}
		}
		return result
	}

	return nil
}

// getTrimesterRisk extracts trimester-specific risk from interface{} map
// Handles both map[string]string and map[string]interface{} from YAML
func getTrimesterRisk(risks interface{}, trimesterKey string) string {
	if risks == nil {
		return ""
	}

	// Try map[string]string
	if strMap, ok := risks.(map[string]string); ok {
		return strMap[trimesterKey]
	}

	// Try map[string]interface{} (common YAML unmarshal result)
	if ifaceMap, ok := risks.(map[string]interface{}); ok {
		if val, ok := ifaceMap[trimesterKey]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	return ""
}

// SafetyChecker performs comprehensive medication safety evaluation
type SafetyChecker struct {
	db             *SafetyDatabase
	governedStore  *KnowledgeStore
	useGoverned    bool
}

// NewSafetyChecker creates a new safety checker instance with hardcoded data
func NewSafetyChecker() *SafetyChecker {
	return &SafetyChecker{
		db:          GetDatabase(),
		useGoverned: false,
	}
}

// NewGovernedSafetyChecker creates a safety checker that uses governed YAML knowledge
// with fallback to hardcoded data when governed knowledge is not available
func NewGovernedSafetyChecker(knowledgePath string) (*SafetyChecker, error) {
	loader := NewKnowledgeLoader(knowledgePath)
	store, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load governed knowledge: %w", err)
	}

	return &SafetyChecker{
		db:            GetDatabase(),
		governedStore: store,
		useGoverned:   true,
	}, nil
}

// NewJurisdictionAwareSafetyChecker creates a safety checker that loads jurisdiction-specific
// knowledge with fallback to global knowledge. Supports US, AU, IN jurisdictions.
// Directory structure: knowledgePath/{us,au,in,global}/{knowledgeType}/*.yaml
func NewJurisdictionAwareSafetyChecker(knowledgePath string, jurisdiction Jurisdiction) (*SafetyChecker, error) {
	loader := NewJurisdictionAwareLoader(knowledgePath, jurisdiction)
	store, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load jurisdiction-aware knowledge: %w", err)
	}

	return &SafetyChecker{
		db:            GetDatabase(),
		governedStore: store,
		useGoverned:   true,
	}, nil
}

// GetKnowledgeStats returns statistics about loaded governed knowledge
func (sc *SafetyChecker) GetKnowledgeStats() map[string]int {
	if sc.governedStore != nil {
		return sc.governedStore.GetStats()
	}
	return map[string]int{"governed_knowledge": 0, "using_hardcoded": 1}
}

// IsUsingGovernedKnowledge returns true if the checker is using governed YAML knowledge
func (sc *SafetyChecker) IsUsingGovernedKnowledge() bool {
	return sc.useGoverned && sc.governedStore != nil
}

// =============================================================================
// PUBLIC ACCESSOR METHODS - Get governed knowledge with fallback to hardcoded
// =============================================================================

// GetBlackBoxWarning returns a black box warning for the given RxNorm code
// Uses governed knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) GetBlackBoxWarning(rxnormCode string) (*BlackBoxWarning, bool) {
	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gw, ok := sc.governedStore.GetBlackBoxWarning(rxnormCode); ok {
			return &gw, true
		}
	}
	// Fallback to hardcoded
	if w := sc.db.GetBlackBoxWarning(rxnormCode); w != nil {
		return w, true
	}
	return nil, false
}

// GetHighAlertMedication returns high-alert info for the given RxNorm code
func (sc *SafetyChecker) GetHighAlertMedication(rxnormCode string) (*HighAlertMedication, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if gm, ok := sc.governedStore.GetHighAlertMedication(rxnormCode); ok {
			return &gm, true
		}
	}
	if m := sc.db.GetHighAlertMedication(rxnormCode); m != nil {
		return m, true
	}
	return nil, false
}

// GetBeersEntry returns Beers Criteria entry for the given RxNorm code
func (sc *SafetyChecker) GetBeersEntry(rxnormCode string) (*BeersEntry, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if ge, ok := sc.governedStore.GetBeersEntry(rxnormCode); ok {
			return &ge, true
		}
	}
	if e := sc.db.GetBeersEntry(rxnormCode); e != nil {
		return e, true
	}
	return nil, false
}

// GetPregnancySafety returns pregnancy safety info for the given RxNorm code
func (sc *SafetyChecker) GetPregnancySafety(rxnormCode string) (*PregnancySafety, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if gp, ok := sc.governedStore.GetPregnancySafety(rxnormCode); ok {
			return &gp, true
		}
	}
	if p := sc.db.GetPregnancySafety(rxnormCode); p != nil {
		return p, true
	}
	return nil, false
}

// GetLactationSafety returns lactation safety info for the given RxNorm code
func (sc *SafetyChecker) GetLactationSafety(rxnormCode string) (*LactationSafety, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if gl, ok := sc.governedStore.GetLactationSafety(rxnormCode); ok {
			return &gl, true
		}
	}
	if l := sc.db.GetLactationSafety(rxnormCode); l != nil {
		return l, true
	}
	return nil, false
}

// GetAnticholinergicBurden returns ACB score for the given RxNorm code
func (sc *SafetyChecker) GetAnticholinergicBurden(rxnormCode string) (*AnticholinergicBurden, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if ga, ok := sc.governedStore.GetAnticholinergicBurden(rxnormCode); ok {
			return &ga, true
		}
	}
	if a := sc.db.GetAnticholinergicBurden(rxnormCode); a != nil {
		return a, true
	}
	return nil, false
}

// GetLabRequirement returns lab requirements for the given RxNorm code
func (sc *SafetyChecker) GetLabRequirement(rxnormCode string) (*LabRequirement, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if gl, ok := sc.governedStore.GetLabRequirement(rxnormCode); ok {
			return &gl, true
		}
	}
	if l := sc.db.GetLabRequirement(rxnormCode); l != nil {
		return l, true
	}
	return nil, false
}

// GetContraindications returns contraindications for the given RxNorm code
func (sc *SafetyChecker) GetContraindications(rxnormCode string) ([]Contraindication, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if gc, ok := sc.governedStore.GetContraindications(rxnormCode); ok {
			return gc, true
		}
	}
	// Convert []*Contraindication to []Contraindication for hardcoded fallback
	if c := sc.db.GetContraindications(rxnormCode); len(c) > 0 {
		result := make([]Contraindication, len(c))
		for i, ptr := range c {
			result[i] = *ptr
		}
		return result, true
	}
	return nil, false
}

// GetDoseLimit returns dose limits for the given RxNorm code
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) GetDoseLimit(rxnormCode string) (*DoseLimit, bool) {
	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if dl, ok := sc.governedStore.GetDoseLimit(rxnormCode); ok {
			return &dl, true
		}
	}
	// Fallback to hardcoded
	if d := sc.db.GetDoseLimit(rxnormCode); d != nil {
		return d, true
	}
	return nil, false
}

// GetAgeLimit returns age limits for the given RxNorm code
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) GetAgeLimit(rxnormCode string) (*AgeLimit, bool) {
	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if al, ok := sc.governedStore.GetAgeLimit(rxnormCode); ok {
			return &al, true
		}
	}
	// Fallback to hardcoded
	if a := sc.db.GetAgeLimit(rxnormCode); a != nil {
		return a, true
	}
	return nil, false
}

// Check performs a comprehensive safety evaluation
func (sc *SafetyChecker) Check(req *SafetyCheckRequest) *SafetyCheckResponse {
	response := &SafetyCheckResponse{
		Safe:            true,
		RequiresAction:  false,
		BlockPrescribing: false,
		Alerts:          []SafetyAlert{},
		CheckedAt:       time.Now(),
		RequestID:       uuid.New().String(),
	}

	rxnormCode := req.Drug.RxNormCode

	// Determine which checks to run
	checkAll := len(req.CheckTypes) == 0
	checkTypes := make(map[AlertType]bool)
	for _, t := range req.CheckTypes {
		checkTypes[t] = true
	}

	// Run all applicable safety checks
	if checkAll || checkTypes[AlertTypeBlackBox] {
		sc.checkBlackBoxWarning(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeContraindication] {
		sc.checkContraindications(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeAgeLimit] {
		sc.checkAgeLimits(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeDoseLimit] {
		sc.checkDoseLimits(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypePregnancy] {
		sc.checkPregnancySafety(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeLactation] {
		sc.checkLactationSafety(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeHighAlert] {
		sc.checkHighAlertStatus(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeBeers] {
		sc.checkBeersCriteria(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeAnticholinergic] {
		sc.checkAnticholinergicBurden(rxnormCode, req, response)
	}

	if checkAll || checkTypes[AlertTypeLabRequired] {
		sc.checkLabRequirements(rxnormCode, req, response)
	}

	// Calculate summary statistics
	sc.calculateSummary(response)

	return response
}

// checkBlackBoxWarning evaluates FDA black box warnings
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkBlackBoxWarning(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var warning *BlackBoxWarning
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gw, ok := sc.governedStore.GetBlackBoxWarning(rxnormCode); ok {
			warning = &gw
			governanceSource = fmt.Sprintf("[%s %s]", gw.Governance.SourceAuthority, gw.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found in governed store
	if warning == nil {
		warning = sc.db.GetBlackBoxWarning(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if warning == nil {
		return
	}

	alert := SafetyAlert{
		ID:                     uuid.New().String(),
		Type:                   AlertTypeBlackBox,
		Severity:               SeverityHigh,
		Title:                  fmt.Sprintf("Black Box Warning: %s", strings.Join(warning.RiskCategories, ", ")),
		Message:                warning.WarningText,
		RequiresAcknowledgment: true,
		CanOverride:            true,
		ClinicalRationale:      fmt.Sprintf("FDA's strongest warning indicating serious or life-threatening risks. %s", governanceSource),
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	if warning.HasREMS {
		alert.Title = fmt.Sprintf("Black Box Warning + REMS: %s", warning.REMSProgram)
		alert.Recommendations = []string{
			fmt.Sprintf("Ensure enrollment in %s program", warning.REMSProgram),
			"Verify all REMS requirements are met before dispensing",
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkContraindications evaluates drug contraindications against patient conditions
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkContraindications(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var governedContraindications []Contraindication
	var hardcodedContraindications []*Contraindication
	var governanceSource string
	useGoverned := false

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gci, ok := sc.governedStore.GetContraindications(rxnormCode); ok && len(gci) > 0 {
			governedContraindications = gci
			useGoverned = true
			// Use governance from first entry for source annotation
			governanceSource = fmt.Sprintf("[%s %s]", gci[0].Governance.SourceAuthority, gci[0].Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found in governed store
	if !useGoverned {
		hardcodedContraindications = sc.db.GetContraindications(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if len(governedContraindications) == 0 && len(hardcodedContraindications) == 0 {
		return
	}

	patientConditions := make(map[string]bool)
	for _, dx := range req.Patient.Diagnoses {
		// Normalize codes (handle ICD-10 format variations)
		code := strings.ToUpper(strings.ReplaceAll(dx.Code, ".", ""))
		patientConditions[code] = true
		// Also match on first 3 characters for category matching
		if len(code) >= 3 {
			patientConditions[code[:3]] = true
		}
	}

	// Helper function to process a contraindication
	processContraindication := func(ci *Contraindication, ciGovernanceSource string) {
		for _, condCode := range ci.ConditionCodes {
			normalizedCode := strings.ToUpper(strings.ReplaceAll(condCode, ".", ""))
			prefix := normalizedCode
			if len(normalizedCode) >= 3 {
				prefix = normalizedCode[:3]
			}

			if patientConditions[normalizedCode] || patientConditions[prefix] {
				severity := ci.Severity
				canOverride := ci.Type == "relative"

				alert := SafetyAlert{
					ID:                     uuid.New().String(),
					Type:                   AlertTypeContraindication,
					Severity:               severity,
					Title:                  fmt.Sprintf("Contraindication: %s", strings.Join(ci.ConditionDescriptions, ", ")),
					Message:                fmt.Sprintf("%s %s", ci.ClinicalRationale, ciGovernanceSource),
					RequiresAcknowledgment: true,
					CanOverride:            canOverride,
					ClinicalRationale:      ci.ClinicalRationale,
					DrugInfo:               &req.Drug,
					CreatedAt:              time.Now(),
				}

				if ci.AlternativeConsiderations != "" {
					alert.Recommendations = []string{ci.AlternativeConsiderations}
				}

				if ci.Type == "absolute" {
					response.BlockPrescribing = true
				}

				response.Alerts = append(response.Alerts, alert)
				break // Only add once per contraindication
			}
		}
	}

	// Process governed contraindications
	if useGoverned {
		for i := range governedContraindications {
			ci := &governedContraindications[i]
			ciGovernanceSource := governanceSource
			if ci.Governance.SourceAuthority != "" {
				ciGovernanceSource = fmt.Sprintf("[%s %s]", ci.Governance.SourceAuthority, ci.Governance.SourceDocument)
			}
			processContraindication(ci, ciGovernanceSource)
		}
	} else {
		// Process hardcoded contraindications
		for _, ci := range hardcodedContraindications {
			processContraindication(ci, governanceSource)
		}
	}
}

// checkAgeLimits evaluates age restrictions
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkAgeLimits(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var limit *AgeLimit
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gl, ok := sc.governedStore.GetAgeLimit(rxnormCode); ok {
			limit = &gl
			governanceSource = fmt.Sprintf(" [%s %s]", gl.Governance.SourceAuthority, gl.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found in governed store
	if limit == nil {
		limit = sc.db.GetAgeLimit(rxnormCode)
		governanceSource = " [Built-in Reference]"
	}

	if limit == nil {
		return
	}

	patientAge := req.Patient.AgeYears
	if req.Patient.AgeMonths > 0 && patientAge == 0 {
		patientAge = req.Patient.AgeMonths / 12
	}

	if limit.MinAgeYears > 0 && patientAge < limit.MinAgeYears {
		alert := SafetyAlert{
			ID:                     uuid.New().String(),
			Type:                   AlertTypeAgeLimit,
			Severity:               limit.Severity,
			Title:                  fmt.Sprintf("Age Restriction: Minimum age %.0f years", limit.MinAgeYears),
			Message:                fmt.Sprintf("Patient age (%.1f years) is below minimum age (%.0f years). %s%s", patientAge, limit.MinAgeYears, limit.Rationale, governanceSource),
			RequiresAcknowledgment: true,
			CanOverride:            limit.Severity != SeverityCritical,
			ClinicalRationale:      limit.Rationale,
			DrugInfo:               &req.Drug,
			CreatedAt:              time.Now(),
		}

		if limit.Severity == SeverityCritical {
			response.BlockPrescribing = true
		}

		response.Alerts = append(response.Alerts, alert)
	}

	if limit.MaxAgeYears > 0 && patientAge > limit.MaxAgeYears {
		alert := SafetyAlert{
			ID:                     uuid.New().String(),
			Type:                   AlertTypeAgeLimit,
			Severity:               limit.Severity,
			Title:                  fmt.Sprintf("Age Restriction: Maximum age %.0f years", limit.MaxAgeYears),
			Message:                fmt.Sprintf("Patient age (%.1f years) exceeds maximum age (%.0f years). %s%s", patientAge, limit.MaxAgeYears, limit.Rationale, governanceSource),
			RequiresAcknowledgment: true,
			CanOverride:            limit.Severity != SeverityCritical,
			ClinicalRationale:      limit.Rationale,
			DrugInfo:               &req.Drug,
			CreatedAt:              time.Now(),
		}

		response.Alerts = append(response.Alerts, alert)
	}
}

// checkDoseLimits evaluates proposed dose against maximum limits
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkDoseLimits(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var limit *DoseLimit
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gl, ok := sc.governedStore.GetDoseLimit(rxnormCode); ok {
			limit = &gl
			governanceSource = fmt.Sprintf(" [%s %s]", gl.Governance.SourceAuthority, gl.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found in governed store
	if limit == nil {
		limit = sc.db.GetDoseLimit(rxnormCode)
		governanceSource = " [Built-in Reference]"
	}

	if limit == nil || req.ProposedDose == 0 {
		return
	}

	// Determine applicable max based on patient age
	maxSingleDose := limit.MaxSingleDose
	isGeriatric := req.Patient.AgeYears >= 65

	if isGeriatric && limit.GeriatricMaxDose > 0 {
		maxSingleDose = limit.GeriatricMaxDose
	}

	if req.ProposedDose > maxSingleDose {
		alert := SafetyAlert{
			ID:       uuid.New().String(),
			Type:     AlertTypeDoseLimit,
			Severity: SeverityHigh,
			Title:    "Exceeds Maximum Single Dose",
			Message: fmt.Sprintf("Proposed dose %.1f %s exceeds maximum single dose %.1f %s%s%s",
				req.ProposedDose, req.DoseUnit, maxSingleDose, limit.MaxSingleDoseUnit,
				func() string {
					if isGeriatric {
						return " (geriatric-adjusted limit)"
					}
					return ""
				}(), governanceSource),
			RequiresAcknowledgment: true,
			CanOverride:            true,
			ClinicalRationale:      "Doses exceeding maximum may increase risk of adverse effects.",
			Recommendations: []string{
				fmt.Sprintf("Consider reducing to %.1f %s or less", maxSingleDose, limit.MaxSingleDoseUnit),
			},
			DrugInfo:  &req.Drug,
			CreatedAt: time.Now(),
		}

		response.Alerts = append(response.Alerts, alert)
	}

	// Add renal/hepatic adjustment alerts if applicable
	if req.Patient.RenalFunction != nil && limit.RenalAdjustment != "" {
		if req.Patient.RenalFunction.EGFR < 30 || req.Patient.RenalFunction.CrCl < 30 {
			alert := SafetyAlert{
				ID:                     uuid.New().String(),
				Type:                   AlertTypeDoseLimit,
				Severity:               SeverityModerate,
				Title:                  "Renal Dose Adjustment Required",
				Message:                fmt.Sprintf("%s%s", limit.RenalAdjustment, governanceSource),
				RequiresAcknowledgment: true,
				CanOverride:            true,
				DrugInfo:               &req.Drug,
				CreatedAt:              time.Now(),
			}
			response.Alerts = append(response.Alerts, alert)
		}
	}

	if req.Patient.HepaticFunction != nil && limit.HepaticAdjustment != "" {
		if req.Patient.HepaticFunction.ChildPughClass == "B" || req.Patient.HepaticFunction.ChildPughClass == "C" {
			alert := SafetyAlert{
				ID:                     uuid.New().String(),
				Type:                   AlertTypeDoseLimit,
				Severity:               SeverityModerate,
				Title:                  "Hepatic Dose Adjustment Required",
				Message:                fmt.Sprintf("%s%s", limit.HepaticAdjustment, governanceSource),
				RequiresAcknowledgment: true,
				CanOverride:            true,
				DrugInfo:               &req.Drug,
				CreatedAt:              time.Now(),
			}
			response.Alerts = append(response.Alerts, alert)
		}
	}
}

// checkPregnancySafety evaluates pregnancy safety
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkPregnancySafety(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	if !req.Patient.IsPregnant {
		return
	}

	var safety *PregnancySafety
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gs, ok := sc.governedStore.GetPregnancySafety(rxnormCode); ok {
			safety = &gs
			governanceSource = fmt.Sprintf("[%s %s]", gs.Governance.SourceAuthority, gs.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found
	if safety == nil {
		safety = sc.db.GetPregnancySafety(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if safety == nil {
		return
	}

	var severity Severity
	var canOverride bool
	var alertTitle string

	// Handle both legacy categories (X, D, C) and PLLR risk categories
	switch safety.Category {
	case PregnancyCategoryX:
		severity = SeverityCritical
		canOverride = false
		response.BlockPrescribing = true
		alertTitle = fmt.Sprintf("Pregnancy Category %s (CONTRAINDICATED)", safety.Category)
	case PregnancyCategoryD:
		severity = SeverityHigh
		canOverride = true
		alertTitle = fmt.Sprintf("Pregnancy Category %s (HIGH RISK)", safety.Category)
	case PregnancyCategoryC:
		severity = SeverityModerate
		canOverride = true
		alertTitle = fmt.Sprintf("Pregnancy Category %s (USE WITH CAUTION)", safety.Category)
	default:
		// Check PLLR-style RiskCategory if legacy Category not set
		switch strings.ToUpper(safety.RiskCategory) {
		case "CONTRAINDICATED":
			severity = SeverityCritical
			canOverride = false
			response.BlockPrescribing = true
			alertTitle = "Pregnancy: CONTRAINDICATED"
		case "HIGH_RISK", "AVOID":
			severity = SeverityHigh
			canOverride = true
			alertTitle = "Pregnancy: HIGH RISK"
		case "USE_WITH_CAUTION", "CAUTION":
			severity = SeverityModerate
			canOverride = true
			alertTitle = "Pregnancy: USE WITH CAUTION"
		case "COMPATIBLE", "PROBABLY_SAFE", "SAFE":
			return // Generally safe - no alert needed
		default:
			// If neither Category nor RiskCategory triggers, skip
			if safety.Category == "" && safety.RiskCategory == "" {
				return
			}
			// Unknown category - generate informational alert
			severity = SeverityLow
			canOverride = true
			alertTitle = fmt.Sprintf("Pregnancy Safety Information: %s", safety.DrugName)
		}
	}

	// Build recommendation message from available data
	recommendation := safety.Recommendation
	if recommendation == "" && safety.PLLRRiskSummary != "" {
		recommendation = safety.PLLRRiskSummary
	}

	alert := SafetyAlert{
		ID:                     uuid.New().String(),
		Type:                   AlertTypePregnancy,
		Severity:               severity,
		Title:                  alertTitle,
		Message:                fmt.Sprintf("%s %s", recommendation, governanceSource),
		RequiresAcknowledgment: severity != SeverityLow,
		CanOverride:            canOverride,
		ClinicalRationale:      safety.Recommendation,
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	// Handle TeratogenicEffects - supports both []string and complex structures
	if effects := getTeratogenicEffectsAsStrings(safety.TeratogenicEffects); len(effects) > 0 {
		alert.Message = fmt.Sprintf("%s Teratogenic effects: %s",
			alert.Message, strings.Join(effects, ", "))
	}

	// Handle alternatives from both old and new field names
	alternatives := safety.AlternativeDrugs
	if len(alternatives) == 0 && len(safety.Alternatives) > 0 {
		alternatives = safety.Alternatives
	}
	if len(alternatives) > 0 {
		alert.Recommendations = []string{
			fmt.Sprintf("Consider alternatives: %s", strings.Join(alternatives, ", ")),
		}
	}

	// Add monitoring recommendations from governed data
	if len(safety.MonitoringInPregnancy) > 0 {
		for _, mon := range safety.MonitoringInPregnancy {
			alert.Recommendations = append(alert.Recommendations, mon)
		}
	}

	// Add trimester-specific warnings - handle interface{} type
	if req.Patient.Trimester > 0 && safety.TrimesterRisks != nil {
		trimesterKey := map[int]string{1: "first", 2: "second", 3: "third"}[req.Patient.Trimester]
		if risk := getTrimesterRisk(safety.TrimesterRisks, trimesterKey); risk != "" {
			alert.Message = fmt.Sprintf("%s. Trimester %d specific risk: %s",
				alert.Message, req.Patient.Trimester, risk)
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkLactationSafety evaluates lactation safety
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkLactationSafety(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	if !req.Patient.IsLactating {
		return
	}

	var safety *LactationSafety
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gs, ok := sc.governedStore.GetLactationSafety(rxnormCode); ok {
			safety = &gs
			governanceSource = fmt.Sprintf("[%s %s]", gs.Governance.SourceAuthority, gs.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found
	if safety == nil {
		safety = sc.db.GetLactationSafety(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if safety == nil {
		return
	}

	var severity Severity
	var canOverride bool

	switch safety.Risk {
	case LactationContraindicated:
		severity = SeverityCritical
		canOverride = false
		response.BlockPrescribing = true
	case LactationUseWithCaution:
		severity = SeverityModerate
		canOverride = true
	case LactationProbablyCompatible, LactationCompatible:
		severity = SeverityLow
		canOverride = true
	default:
		severity = SeverityModerate
		canOverride = true
	}

	alert := SafetyAlert{
		ID:                     uuid.New().String(),
		Type:                   AlertTypeLactation,
		Severity:               severity,
		Title:                  fmt.Sprintf("Lactation Risk: %s", safety.Risk),
		Message:                fmt.Sprintf("%s %s", safety.Recommendation, governanceSource),
		RequiresAcknowledgment: severity == SeverityCritical || severity == SeverityHigh,
		CanOverride:            canOverride,
		ClinicalRationale:      safety.Recommendation,
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	if len(safety.InfantEffects) > 0 {
		alert.Message = fmt.Sprintf("%s Potential infant effects: %s",
			alert.Message, strings.Join(safety.InfantEffects, ", "))
	}

	if len(safety.AlternativeDrugs) > 0 {
		alert.Recommendations = []string{
			fmt.Sprintf("Preferred alternatives: %s", strings.Join(safety.AlternativeDrugs, ", ")),
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkHighAlertStatus checks if drug is on ISMP high-alert list
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkHighAlertStatus(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var med *HighAlertMedication
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gm, ok := sc.governedStore.GetHighAlertMedication(rxnormCode); ok {
			med = &gm
			governanceSource = fmt.Sprintf("[%s %s]", gm.Governance.SourceAuthority, gm.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found
	if med == nil {
		med = sc.db.GetHighAlertMedication(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if med == nil {
		return
	}

	response.IsHighAlertDrug = true

	alert := SafetyAlert{
		ID:       uuid.New().String(),
		Type:     AlertTypeHighAlert,
		Severity: SeverityModerate,
		Title:    fmt.Sprintf("High-Alert Medication: %s", med.Category),
		Message: fmt.Sprintf("%s is classified as a high-alert medication. "+
			"Requirements: %s. %s", med.DrugName, strings.Join(med.Requirements, ", "), governanceSource),
		RequiresAcknowledgment: true,
		CanOverride:            true,
		ClinicalRationale:      "High-alert medications bear heightened risk of causing significant patient harm when used in error.",
		Recommendations:        med.Safeguards,
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	if med.DoubleCheck {
		alert.Recommendations = append(alert.Recommendations, "Independent double-check required")
	}
	if med.SmartPump {
		alert.Recommendations = append(alert.Recommendations, "Smart pump required for IV administration")
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkBeersCriteria evaluates Beers Criteria for geriatric patients
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkBeersCriteria(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	// Beers Criteria applies to patients 65+
	if req.Patient.AgeYears < 65 {
		return
	}

	var entry *BeersEntry
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if ge, ok := sc.governedStore.GetBeersEntry(rxnormCode); ok {
			entry = &ge
			governanceSource = fmt.Sprintf("[%s %s]", ge.Governance.SourceAuthority, ge.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found
	if entry == nil {
		entry = sc.db.GetBeersEntry(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if entry == nil {
		return
	}

	var severity Severity
	switch entry.Recommendation {
	case BeersAvoid:
		severity = SeverityModerate
	case BeersAvoidInCondition:
		severity = SeverityModerate
	case BeersUseWithCaution:
		severity = SeverityLow
	}

	alert := SafetyAlert{
		ID:       uuid.New().String(),
		Type:     AlertTypeBeers,
		Severity: severity,
		Title:    fmt.Sprintf("Beers Criteria: %s", entry.Recommendation),
		Message: fmt.Sprintf("%s - %s. Quality of Evidence: %s, Strength: %s. %s",
			entry.DrugClass, entry.Rationale,
			entry.QualityOfEvidence, entry.StrengthOfRecommendation, governanceSource),
		RequiresAcknowledgment: true,
		CanOverride:            true,
		ClinicalRationale:      entry.Rationale,
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	if len(entry.AlternativeDrugs) > 0 {
		alert.Recommendations = []string{
			fmt.Sprintf("Consider alternatives: %s", strings.Join(entry.AlternativeDrugs, ", ")),
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkAnticholinergicBurden evaluates anticholinergic burden
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkAnticholinergicBurden(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var burden *AnticholinergicBurden
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gb, ok := sc.governedStore.GetAnticholinergicBurden(rxnormCode); ok {
			burden = &gb
			governanceSource = fmt.Sprintf("[%s]", gb.Governance.SourceAuthority)
		}
	}

	// Fallback to hardcoded if not found
	if burden == nil {
		burden = sc.db.GetAnticholinergicBurden(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if burden == nil || burden.ACBScore == 0 {
		return
	}

	// Calculate total burden including current medications
	totalScore := burden.ACBScore
	for _, med := range req.Patient.CurrentMedications {
		// Check governed store first for each medication
		var otherBurden *AnticholinergicBurden
		if sc.useGoverned && sc.governedStore != nil {
			if gb, ok := sc.governedStore.GetAnticholinergicBurden(med.RxNormCode); ok {
				otherBurden = &gb
			}
		}
		if otherBurden == nil {
			otherBurden = sc.db.GetAnticholinergicBurden(med.RxNormCode)
		}
		if otherBurden != nil {
			totalScore += otherBurden.ACBScore
		}
	}

	response.AnticholinergicBurdenTotal = totalScore

	var severity Severity
	var riskLevel string
	switch {
	case totalScore >= 6:
		severity = SeverityHigh
		riskLevel = "Very High"
	case totalScore >= 4:
		severity = SeverityModerate
		riskLevel = "High"
	case totalScore >= 2:
		severity = SeverityModerate
		riskLevel = "Moderate"
	default:
		severity = SeverityLow
		riskLevel = "Low"
	}

	alert := SafetyAlert{
		ID:       uuid.New().String(),
		Type:     AlertTypeAnticholinergic,
		Severity: severity,
		Title:    fmt.Sprintf("Anticholinergic Burden: ACB Score %d (%s Risk)", totalScore, riskLevel),
		Message: fmt.Sprintf("%s has ACB score of %d. Total burden with current medications: %d. "+
			"Effects: %s. %s", burden.DrugName, burden.ACBScore, totalScore, strings.Join(burden.Effects, ", "), governanceSource),
		RequiresAcknowledgment: totalScore >= 3,
		CanOverride:            true,
		ClinicalRationale:      "High anticholinergic burden increases risk of cognitive impairment, falls, and delirium, especially in elderly patients.",
		DrugInfo:               &req.Drug,
		CreatedAt:              time.Now(),
	}

	if totalScore >= 3 {
		alert.Recommendations = []string{
			"Consider medication review to reduce anticholinergic burden",
			"Monitor for cognitive changes and falls",
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// checkLabRequirements adds informational alerts for required monitoring
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) checkLabRequirements(rxnormCode string, req *SafetyCheckRequest, response *SafetyCheckResponse) {
	var labReq *LabRequirement
	var governanceSource string

	// Try governed knowledge first
	if sc.useGoverned && sc.governedStore != nil {
		if gl, ok := sc.governedStore.GetLabRequirement(rxnormCode); ok {
			labReq = &gl
			governanceSource = fmt.Sprintf("[%s %s]", gl.Governance.SourceAuthority, gl.Governance.SourceDocument)
		}
	}

	// Fallback to hardcoded if not found
	if labReq == nil {
		labReq = sc.db.GetLabRequirement(rxnormCode)
		governanceSource = "[Built-in Reference]"
	}

	if labReq == nil {
		return
	}

	// Extract lab names - prefer complex Labs array, fallback to RequiredLabs
	var labNames []string
	var frequency string
	var labDetails []string

	if len(labReq.Labs) > 0 {
		// Extract from complex Labs structure (governed YAML)
		for _, lab := range labReq.Labs {
			if lab.LabName != "" {
				labNames = append(labNames, lab.LabName)
				// Build detailed lab info
				detail := lab.LabName
				if lab.LOINCCode != "" {
					detail += fmt.Sprintf(" (LOINC: %s)", lab.LOINCCode)
				}
				labDetails = append(labDetails, detail)
			}
		}
		// Extract frequency from first lab entry or use monitoring fields
		if len(labReq.Labs) > 0 {
			if freq, ok := labReq.Labs[0].Frequency.(string); ok && freq != "" {
				frequency = freq
			}
		}
	} else if len(labReq.RequiredLabs) > 0 {
		// Fallback to simple RequiredLabs array
		labNames = labReq.RequiredLabs
		labDetails = labReq.RequiredLabs
	}

	// Fallback frequency from legacy fields
	if frequency == "" {
		if labReq.Frequency != "" {
			frequency = labReq.Frequency
		} else if labReq.InitialMonitoring != "" {
			frequency = fmt.Sprintf("Initial: %s", labReq.InitialMonitoring)
			if labReq.OngoingMonitoring != "" {
				frequency += fmt.Sprintf(", Ongoing: %s", labReq.OngoingMonitoring)
			}
		} else if labReq.OngoingMonitoring != "" {
			frequency = labReq.OngoingMonitoring
		} else {
			frequency = "Per clinical guidelines"
		}
	}

	// Build comprehensive message
	labList := strings.Join(labNames, ", ")
	if labList == "" {
		labList = "See prescribing information"
	}

	alert := SafetyAlert{
		ID:       uuid.New().String(),
		Type:     AlertTypeLabRequired,
		Severity: SeverityLow,
		Title:    fmt.Sprintf("Laboratory Monitoring Required: %s", labReq.DrugName),
		Message: fmt.Sprintf("Required labs: %s. Monitoring schedule: %s. %s %s",
			labList,
			frequency,
			labReq.Rationale,
			governanceSource),
		RequiresAcknowledgment: false,
		CanOverride:            true,
		ClinicalRationale:      labReq.Rationale,
		Recommendations: []string{
			fmt.Sprintf("Order baseline labs before starting therapy: %s", labList),
			fmt.Sprintf("Monitoring schedule: %s", frequency),
		},
		DrugInfo:  &req.Drug,
		CreatedAt: time.Now(),
	}

	if labReq.BaselineRequired {
		alert.Recommendations = append([]string{"BASELINE LABS REQUIRED before initiation"}, alert.Recommendations...)
	}

	// Add critical monitoring flag if applicable
	if labReq.CriticalMonitoring {
		alert.Severity = SeverityModerate
		alert.RequiresAcknowledgment = true
		alert.Title = fmt.Sprintf("CRITICAL Laboratory Monitoring Required: %s", labReq.DrugName)
	}

	// Add critical threshold info to recommendations if available
	if len(labReq.Labs) > 0 {
		for _, lab := range labReq.Labs {
			if lab.CriticalValues != nil {
				if critMap, ok := lab.CriticalValues.(map[string]interface{}); ok {
					for key, val := range critMap {
						alert.Recommendations = append(alert.Recommendations,
							fmt.Sprintf("Critical %s for %s: %v - requires immediate action", key, lab.LabName, val))
					}
				}
			}
		}
	}

	response.Alerts = append(response.Alerts, alert)
}

// calculateSummary computes response summary statistics
func (sc *SafetyChecker) calculateSummary(response *SafetyCheckResponse) {
	for _, alert := range response.Alerts {
		switch alert.Severity {
		case SeverityCritical:
			response.CriticalAlerts++
		case SeverityHigh:
			response.HighAlerts++
		case SeverityModerate:
			response.ModerateAlerts++
		case SeverityLow:
			response.LowAlerts++
		}
	}

	response.TotalAlerts = len(response.Alerts)
	response.Safe = response.CriticalAlerts == 0 && response.HighAlerts == 0
	response.RequiresAction = response.CriticalAlerts > 0 || response.HighAlerts > 0

	// Block prescribing if any critical alert that can't be overridden
	for _, alert := range response.Alerts {
		if alert.Severity == SeverityCritical && !alert.CanOverride {
			response.BlockPrescribing = true
			break
		}
	}
}

// ValidateDose checks if a proposed dose is within limits
func (sc *SafetyChecker) ValidateDose(drug DrugInfo, proposedDose float64, doseUnit string, patient PatientContext) *DoseLimitValidation {
	result := &DoseLimitValidation{
		Drug:         drug,
		ProposedDose: proposedDose,
		DoseUnit:     doseUnit,
		Patient:      patient,
		IsValid:      true,
	}

	limit := sc.db.GetDoseLimit(drug.RxNormCode)
	if limit == nil {
		result.Message = "No dose limits on file for this medication"
		return result
	}

	maxDose := limit.MaxSingleDose
	if patient.AgeYears >= 65 && limit.GeriatricMaxDose > 0 {
		maxDose = limit.GeriatricMaxDose
	}

	if proposedDose > maxDose {
		result.IsValid = false
		result.ExceedsSingle = true
		result.MaxAllowed = maxDose
		result.Message = fmt.Sprintf("Proposed dose %.1f %s exceeds maximum %.1f %s",
			proposedDose, doseUnit, maxDose, limit.MaxSingleDoseUnit)
	} else {
		result.MaxAllowed = maxDose
		result.Message = fmt.Sprintf("Dose is within limits (max: %.1f %s)", maxDose, limit.MaxSingleDoseUnit)
	}

	return result
}

// CalculateAnticholinergicBurden calculates total ACB for a medication list
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) CalculateAnticholinergicBurden(medications []DrugInfo) *AnticholinergicBurdenCalculation {
	result := &AnticholinergicBurdenCalculation{
		TotalScore:  0,
		Medications: []AnticholinergicBurden{},
	}

	for _, med := range medications {
		var burden *AnticholinergicBurden

		// Try governed knowledge first
		if sc.useGoverned && sc.governedStore != nil {
			if gb, ok := sc.governedStore.GetAnticholinergicBurden(med.RxNormCode); ok {
				burden = &gb
			}
		}

		// Fallback to hardcoded
		if burden == nil {
			burden = sc.db.GetAnticholinergicBurden(med.RxNormCode)
		}

		if burden != nil {
			result.TotalScore += burden.ACBScore
			result.Medications = append(result.Medications, *burden)
		}
	}

	switch {
	case result.TotalScore >= 6:
		result.RiskLevel = "Very High"
		result.CognitiveRisk = "Significant risk of cognitive impairment and delirium"
		result.Recommendation = "Immediate medication review recommended. Consider deprescribing or switching medications."
	case result.TotalScore >= 4:
		result.RiskLevel = "High"
		result.CognitiveRisk = "Elevated risk of cognitive effects"
		result.Recommendation = "Medication review recommended. Monitor for cognitive changes and falls."
	case result.TotalScore >= 2:
		result.RiskLevel = "Moderate"
		result.CognitiveRisk = "Moderate risk of anticholinergic effects"
		result.Recommendation = "Monitor for anticholinergic side effects."
	default:
		result.RiskLevel = "Low"
		result.CognitiveRisk = "Low risk"
		result.Recommendation = "Routine monitoring sufficient."
	}

	return result
}

// CheckBeers performs Beers Criteria check for a medication list
// Uses governed YAML knowledge first, falls back to hardcoded data
func (sc *SafetyChecker) CheckBeers(medications []DrugInfo, patientAge float64) []SafetyAlert {
	if patientAge < 65 {
		return nil
	}

	var alerts []SafetyAlert
	for _, med := range medications {
		var entry *BeersEntry
		var governanceSource string

		// Try governed knowledge first
		if sc.useGoverned && sc.governedStore != nil {
			if ge, ok := sc.governedStore.GetBeersEntry(med.RxNormCode); ok {
				entry = &ge
				governanceSource = fmt.Sprintf(" [%s]", ge.Governance.SourceAuthority)
			}
		}

		// Fallback to hardcoded
		if entry == nil {
			entry = sc.db.GetBeersEntry(med.RxNormCode)
			governanceSource = ""
		}

		if entry != nil {
			alerts = append(alerts, SafetyAlert{
				ID:       uuid.New().String(),
				Type:     AlertTypeBeers,
				Severity: SeverityModerate,
				Title:    fmt.Sprintf("Beers Criteria: %s - %s%s", med.DrugName, entry.Recommendation, governanceSource),
				Message:  entry.Rationale,
				DrugInfo: &med,
			})
		}
	}

	return alerts
}

// =============================================================================
// STOPP/START CRITERIA (European Geriatric Prescribing)
// =============================================================================

// CheckStoppCriteria evaluates STOPP criteria for potentially inappropriate prescribing
// STOPP: Screening Tool of Older Persons' Prescriptions
// Applies to patients ≥65 years (unless end-of-life or symptom control priority)
func (sc *SafetyChecker) CheckStoppCriteria(medications []DrugInfo, patientAge float64, patientConditions []string) []StoppViolation {
	// STOPP criteria apply to patients aged 65 and older
	if patientAge < 65 {
		return nil
	}

	if !sc.useGoverned || sc.governedStore == nil {
		return nil // STOPP/START only available in governed knowledge
	}

	var violations []StoppViolation

	// Get all STOPP entries
	stoppEntries := sc.governedStore.GetAllStoppEntries()

	for _, med := range medications {
		for _, entry := range stoppEntries {
			matched := false
			var matchedCondition string

			// Check if medication matches STOPP criterion by RxNorm code
			for _, rxnorm := range entry.RxNormCodes {
				if rxnorm == med.RxNormCode {
					matched = true
					break
				}
			}

			// Check if medication matches by drug class (case-insensitive)
			if !matched && entry.DrugClass != "" && med.DrugClass != "" {
				if strings.EqualFold(entry.DrugClass, med.DrugClass) {
					matched = true
				}
			}

			// For condition-specific STOPP criteria, check if patient has the condition
			if matched && len(entry.ConditionICD10) > 0 {
				conditionMatched := false
				for _, patientCondition := range patientConditions {
					for _, criteriaCondition := range entry.ConditionICD10 {
						if strings.HasPrefix(patientCondition, criteriaCondition) {
							conditionMatched = true
							matchedCondition = entry.Condition
							break
						}
					}
					if conditionMatched {
						break
					}
				}
				// If criterion requires condition but patient doesn't have it, no violation
				if !conditionMatched {
					matched = false
				}
			}

			if matched {
				violations = append(violations, StoppViolation{
					Entry:            &entry,
					CurrentDrug:      med,
					MatchedCondition: matchedCondition,
					Message: fmt.Sprintf("STOPP %s: %s - %s [EUGMS v3]",
						entry.ID, entry.Criteria, entry.Rationale),
					Severity: SeverityModerate,
				})
			}
		}
	}

	return violations
}

// CheckStartCriteria evaluates START criteria for potential prescribing omissions
// START: Screening Tool to Alert to Right Treatment
// Applies to patients ≥65 years (unless end-of-life or symptom control priority)
func (sc *SafetyChecker) CheckStartCriteria(currentMedications []DrugInfo, patientAge float64, patientConditions []string) []StartRecommendation {
	// START criteria apply to patients aged 65 and older
	if patientAge < 65 {
		return nil
	}

	if !sc.useGoverned || sc.governedStore == nil {
		return nil // STOPP/START only available in governed knowledge
	}

	var recommendations []StartRecommendation

	// Get all START entries
	startEntries := sc.governedStore.GetAllStartEntries()

	// Build a set of current medication RxNorm codes for quick lookup
	currentMedSet := make(map[string]bool)
	for _, med := range currentMedications {
		currentMedSet[med.RxNormCode] = true
	}

	for _, entry := range startEntries {
		// Check if patient has the condition that warrants starting treatment
		conditionMatched := false
		var matchedCondition string

		for _, patientCondition := range patientConditions {
			for _, criteriaCondition := range entry.ConditionICD10 {
				if strings.HasPrefix(patientCondition, criteriaCondition) {
					conditionMatched = true
					matchedCondition = entry.Condition
					break
				}
			}
			if conditionMatched {
				break
			}
		}

		if !conditionMatched {
			continue // Patient doesn't have the condition, START doesn't apply
		}

		// Check if patient is already receiving any of the recommended drugs
		alreadyReceiving := false
		for _, rxnorm := range entry.RxNormCodes {
			if currentMedSet[rxnorm] {
				alreadyReceiving = true
				break
			}
		}

		if !alreadyReceiving {
			// Patient has condition but is NOT receiving recommended treatment
			recommendations = append(recommendations, StartRecommendation{
				Entry:            &entry,
				MatchedCondition: matchedCondition,
				RecommendedDrugs: entry.RecommendedDrugs,
				Message: fmt.Sprintf("START %s: %s - Consider: %s [EUGMS v3]",
					entry.ID, entry.Criteria, strings.Join(entry.RecommendedDrugs, ", ")),
				Severity: SeverityLow,
			})
		}
	}

	return recommendations
}

// GetStoppEntry returns a specific STOPP criterion by ID
func (sc *SafetyChecker) GetStoppEntry(criterionID string) (*StoppEntry, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if entry, ok := sc.governedStore.GetStoppEntry(criterionID); ok {
			return &entry, true
		}
	}
	return nil, false
}

// GetStartEntry returns a specific START criterion by ID
func (sc *SafetyChecker) GetStartEntry(criterionID string) (*StartEntry, bool) {
	if sc.useGoverned && sc.governedStore != nil {
		if entry, ok := sc.governedStore.GetStartEntry(criterionID); ok {
			return &entry, true
		}
	}
	return nil, false
}

// GetAllStoppEntries returns all STOPP criteria
func (sc *SafetyChecker) GetAllStoppEntries() []StoppEntry {
	if sc.useGoverned && sc.governedStore != nil {
		return sc.governedStore.GetAllStoppEntries()
	}
	return nil
}

// GetAllStartEntries returns all START criteria
func (sc *SafetyChecker) GetAllStartEntries() []StartEntry {
	if sc.useGoverned && sc.governedStore != nil {
		return sc.governedStore.GetAllStartEntries()
	}
	return nil
}
