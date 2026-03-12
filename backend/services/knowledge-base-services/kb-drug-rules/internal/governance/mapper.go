package governance

import (
	"regexp"
	"strings"
	"time"
)

const (
	// Dataset version for KB-1 drug rules
	KB1DatasetVersion = "KB1-2025.12.1"

	// Rule version
	KB1RuleVersion = "1.0.0"
)

// AlertType represents the type of clinical alert
type AlertType string

const (
	AlertTypeHighAlert      AlertType = "HIGH_ALERT"
	AlertTypeBlackBox       AlertType = "BLACK_BOX"
	AlertTypeNarrowTI       AlertType = "NARROW_TI"
	AlertTypeDoseExceeded   AlertType = "DOSE_EXCEEDED"
	AlertTypeRenalCaution   AlertType = "RENAL_CAUTION"
	AlertTypeHepaticCaution AlertType = "HEPATIC_CAUTION"
	AlertTypeGeriatric      AlertType = "GERIATRIC"
	AlertTypePediatric      AlertType = "PEDIATRIC"
	AlertTypePregnancy      AlertType = "PREGNANCY"
	AlertTypeMonitoring     AlertType = "MONITORING"
	AlertTypeInteraction    AlertType = "INTERACTION"
	AlertTypeGeneral        AlertType = "GENERAL"
)

// SeverityMapper maps KB-1 alerts/warnings to governance severities
type SeverityMapper struct {
	// Pattern-based mappings for warning text
	patternMappings []patternMapping

	// Direct mappings for alert types
	typeMappings map[AlertType]GovernanceSeverity

	// Clinical reference sources by drug class
	referenceSources map[string]string
}

type patternMapping struct {
	pattern  *regexp.Regexp
	severity GovernanceSeverity
}

// NewSeverityMapper creates a new severity mapper with default mappings
func NewSeverityMapper() *SeverityMapper {
	mapper := &SeverityMapper{
		typeMappings:     make(map[AlertType]GovernanceSeverity),
		referenceSources: make(map[string]string),
	}

	// Initialize pattern-based mappings
	mapper.initPatternMappings()

	// Initialize type-based mappings
	mapper.initTypeMappings()

	// Initialize reference sources
	mapper.initReferenceSources()

	return mapper
}

func (m *SeverityMapper) initPatternMappings() {
	patterns := []struct {
		regex    string
		severity GovernanceSeverity
	}{
		// Hard blocks - absolute contraindications
		{`(?i)contraindicated`, SeverityHardBlock},
		{`(?i)do not use`, SeverityHardBlock},
		{`(?i)absolute.*contraindication`, SeverityHardBlock},
		{`(?i)fatal.*risk`, SeverityHardBlock},
		{`(?i)life.?threatening`, SeverityMandatoryEscalation},

		// Mandatory escalation
		{`(?i)black.?box`, SeverityMandatoryEscalation},
		{`(?i)rems.?program`, SeverityMandatoryEscalation},
		{`(?i)risk.*evaluation`, SeverityMandatoryEscalation},

		// Supervisor required
		{`(?i)high.?alert`, SeverityOverrideWithSupervisor},
		{`(?i)narrow.*therapeutic`, SeverityOverrideWithSupervisor},
		{`(?i)requires.*independent.*check`, SeverityOverrideWithSupervisor},
		{`(?i)double.?check`, SeverityOverrideWithSupervisor},

		// Override with documentation
		{`(?i)exceeds.*max`, SeverityOverrideWithDocumentation},
		{`(?i)exceeds.*dose`, SeverityOverrideWithDocumentation},
		{`(?i)renal.*adjust`, SeverityOverrideWithDocumentation},
		{`(?i)hepatic.*adjust`, SeverityOverrideWithDocumentation},
		{`(?i)elderly.*lower`, SeverityOverrideWithDocumentation},
		{`(?i)geriatric`, SeverityOverrideWithDocumentation},
		{`(?i)reduce.*dose`, SeverityOverrideWithDocumentation},

		// Counseling required
		{`(?i)patient.*education`, SeverityCounselingRequired},
		{`(?i)counsel.*patient`, SeverityCounselingRequired},
		{`(?i)lifestyle`, SeverityCounselingRequired},
		{`(?i)dietary`, SeverityCounselingRequired},
		{`(?i)avoid.*alcohol`, SeverityCounselingRequired},
		{`(?i)avoid.*grapefruit`, SeverityCounselingRequired},

		// Notify only - monitoring and information
		{`(?i)monitor`, SeverityNotifyOnly},
		{`(?i)periodic.*check`, SeverityNotifyOnly},
		{`(?i)follow.?up`, SeverityNotifyOnly},
		{`(?i)titrat`, SeverityNotifyOnly},
	}

	for _, p := range patterns {
		compiled := regexp.MustCompile(p.regex)
		m.patternMappings = append(m.patternMappings, patternMapping{
			pattern:  compiled,
			severity: p.severity,
		})
	}
}

func (m *SeverityMapper) initTypeMappings() {
	m.typeMappings[AlertTypeHighAlert] = SeverityOverrideWithSupervisor
	m.typeMappings[AlertTypeBlackBox] = SeverityMandatoryEscalation
	m.typeMappings[AlertTypeNarrowTI] = SeverityOverrideWithSupervisor
	m.typeMappings[AlertTypeDoseExceeded] = SeverityOverrideWithDocumentation
	m.typeMappings[AlertTypeRenalCaution] = SeverityOverrideWithDocumentation
	m.typeMappings[AlertTypeHepaticCaution] = SeverityOverrideWithDocumentation
	m.typeMappings[AlertTypeGeriatric] = SeverityCounselingRequired
	m.typeMappings[AlertTypePediatric] = SeverityOverrideWithDocumentation
	m.typeMappings[AlertTypePregnancy] = SeverityOverrideWithSupervisor
	m.typeMappings[AlertTypeMonitoring] = SeverityNotifyOnly
	m.typeMappings[AlertTypeInteraction] = SeverityOverrideWithDocumentation
	m.typeMappings[AlertTypeGeneral] = SeverityNotifyOnly
}

func (m *SeverityMapper) initReferenceSources() {
	m.referenceSources["default"] = "FDA Label, Lexicomp, UpToDate 2024"
	m.referenceSources["anticoagulant"] = "CHEST Guidelines 2021, FDA Label"
	m.referenceSources["diabetes"] = "ADA Standards of Care 2024, FDA Label"
	m.referenceSources["cardiovascular"] = "ACC/AHA Guidelines 2023, FDA Label"
	m.referenceSources["antibiotic"] = "IDSA Guidelines, Sanford Guide 2024"
	m.referenceSources["opioid"] = "CDC Opioid Prescribing Guidelines 2022, FDA REMS"
	m.referenceSources["renal"] = "KDIGO Guidelines 2024, FDA Label"
}

// MapWarning maps a warning message to governance severity
func (m *SeverityMapper) MapWarning(message string) GovernanceSeverity {
	// Check pattern-based mappings in order (higher severity patterns first)
	for _, pm := range m.patternMappings {
		if pm.pattern.MatchString(message) {
			return pm.severity
		}
	}

	// Default to notify only for unmatched warnings
	return SeverityNotifyOnly
}

// MapAlertType maps an alert type to governance severity
func (m *SeverityMapper) MapAlertType(alertType AlertType) GovernanceSeverity {
	if severity, ok := m.typeMappings[alertType]; ok {
		return severity
	}
	return SeverityNotifyOnly
}

// MapSafetyFlags maps KB-1 safety flags to governance result
func (m *SeverityMapper) MapSafetyFlags(isHighAlert, hasBlackBox, isNarrowTI bool, drugClass string) []GovernanceResult {
	var results []GovernanceResult

	if hasBlackBox {
		results = append(results, GovernanceResult{
			OriginalType:    string(AlertTypeBlackBox),
			OriginalMessage: "BLACK BOX WARNING - review specific warnings before prescribing",
			Severity:        SeverityMandatoryEscalation,
			Action:          GovernanceActions[SeverityMandatoryEscalation],
			Provenance:      m.createProvenance(drugClass, "FDA Black Box Warning Review"),
			OverrideReasonOptions: []string{
				"Benefit outweighs risk for this patient",
				"No alternative therapy available",
				"Specialist consultation obtained",
				"Patient informed consent documented",
			},
		})
	}

	if isHighAlert {
		results = append(results, GovernanceResult{
			OriginalType:    string(AlertTypeHighAlert),
			OriginalMessage: "HIGH-ALERT medication - requires independent double-check",
			Severity:        SeverityOverrideWithSupervisor,
			Action:          GovernanceActions[SeverityOverrideWithSupervisor],
			Provenance:      m.createProvenance(drugClass, "ISMP High-Alert Medication List"),
			OverrideReasonOptions: []string{
				"Independent double-check completed",
				"Dose verified by clinical pharmacist",
				"Protocol-based dosing approved",
			},
		})
	}

	if isNarrowTI {
		results = append(results, GovernanceResult{
			OriginalType:    string(AlertTypeNarrowTI),
			OriginalMessage: "NARROW THERAPEUTIC INDEX - monitor levels closely",
			Severity:        SeverityOverrideWithSupervisor,
			Action:          GovernanceActions[SeverityOverrideWithSupervisor],
			Provenance:      m.createProvenance(drugClass, "FDA NTI Drug Classification"),
			OverrideReasonOptions: []string{
				"Therapeutic drug monitoring ordered",
				"Baseline levels documented",
				"Dosing protocol established",
			},
		})
	}

	return results
}

// MapWarningsAndErrors maps KB-1 warnings/errors arrays to governance results
func (m *SeverityMapper) MapWarningsAndErrors(warnings, errors, safetyAlerts []string, drugClass string) []GovernanceResult {
	var results []GovernanceResult

	// Map errors (highest priority)
	for _, err := range errors {
		severity := m.MapWarning(err)
		// Errors are at least override with documentation
		if severity == SeverityNotifyOnly || severity == SeverityCounselingRequired {
			severity = SeverityOverrideWithDocumentation
		}
		results = append(results, GovernanceResult{
			OriginalType:    "ERROR",
			OriginalMessage: err,
			Severity:        severity,
			Action:          GovernanceActions[severity],
			Provenance:      m.createProvenance(drugClass, "Dose Limit Validation"),
		})
	}

	// Map safety alerts
	for _, alert := range safetyAlerts {
		severity := m.MapWarning(alert)
		results = append(results, GovernanceResult{
			OriginalType:    "SAFETY_ALERT",
			OriginalMessage: alert,
			Severity:        severity,
			Action:          GovernanceActions[severity],
			Provenance:      m.createProvenance(drugClass, "Clinical Safety Engine"),
		})
	}

	// Map warnings
	for _, warning := range warnings {
		severity := m.MapWarning(warning)
		results = append(results, GovernanceResult{
			OriginalType:    "WARNING",
			OriginalMessage: warning,
			Severity:        severity,
			Action:          GovernanceActions[severity],
			Provenance:      m.createProvenance(drugClass, "Dosing Rule Evaluation"),
		})
	}

	return results
}

// createProvenance generates evidence provenance for a governance result
func (m *SeverityMapper) createProvenance(drugClass, calculationMethod string) EvidenceProvenance {
	source := m.referenceSources["default"]
	if classSource, ok := m.referenceSources[strings.ToLower(drugClass)]; ok {
		source = classSource
	}

	return EvidenceProvenance{
		ClinicalReferenceSource:     source,
		CalculationMethodVersion:    calculationMethod,
		DatasetVersion:              KB1DatasetVersion,
		GovernanceBinding:           "FDA/21CFR",
		RequiresSecondaryValidation: false,
		EvaluatedAt:                 time.Now().UTC(),
		RuleVersion:                 KB1RuleVersion,
	}
}

// GetHighestSeverity returns the highest severity from a list of results
func GetHighestSeverity(results []GovernanceResult) GovernanceSeverity {
	severityOrder := map[GovernanceSeverity]int{
		SeverityNotifyOnly:                0,
		SeverityCounselingRequired:        1,
		SeverityOverrideWithDocumentation: 2,
		SeverityOverrideWithSupervisor:    3,
		SeverityMandatoryEscalation:       4,
		SeverityHardBlock:                 5,
	}

	highest := SeverityNotifyOnly
	highestOrder := 0

	for _, result := range results {
		if order, ok := severityOrder[result.Severity]; ok && order > highestOrder {
			highest = result.Severity
			highestOrder = order
		}
	}

	return highest
}

// CanProceed determines if the action can proceed based on governance results
func CanProceed(results []GovernanceResult) bool {
	for _, result := range results {
		if result.Severity == SeverityHardBlock {
			return false
		}
	}
	return true
}

// GetRequiredSteps returns the list of required steps before proceeding
func GetRequiredSteps(results []GovernanceResult) []string {
	stepsMap := make(map[string]bool)
	var steps []string

	for _, result := range results {
		action := result.Action

		if action.DocumentationNeeded && !stepsMap["documentation"] {
			steps = append(steps, "Document clinical rationale")
			stepsMap["documentation"] = true
		}

		if action.RequiresSupervisor && !stepsMap["supervisor"] {
			steps = append(steps, "Obtain supervisor approval")
			stepsMap["supervisor"] = true
		}

		if action.RequiresEscalation && !stepsMap["escalation"] {
			steps = append(steps, "Escalate to clinical review board")
			stepsMap["escalation"] = true
		}

		if result.Severity == SeverityCounselingRequired && !stepsMap["counseling"] {
			steps = append(steps, "Provide and document patient counseling")
			stepsMap["counseling"] = true
		}
	}

	return steps
}

// CreateEnhancedResponse wraps any response with governance metadata
func (m *SeverityMapper) CreateEnhancedResponse(
	data interface{},
	warnings, errors, safetyAlerts []string,
	isHighAlert, hasBlackBox, isNarrowTI bool,
	drugClass, calculationMethod string,
) GovernanceEnhancedResponse {

	var allResults []GovernanceResult

	// Add safety flag results
	flagResults := m.MapSafetyFlags(isHighAlert, hasBlackBox, isNarrowTI, drugClass)
	allResults = append(allResults, flagResults...)

	// Add warning/error results
	messageResults := m.MapWarningsAndErrors(warnings, errors, safetyAlerts, drugClass)
	allResults = append(allResults, messageResults...)

	// Create provenance for the overall calculation
	provenance := EvidenceProvenance{
		ClinicalReferenceSource:     m.referenceSources["default"],
		CalculationMethodVersion:    calculationMethod,
		DatasetVersion:              KB1DatasetVersion,
		GovernanceBinding:           "FDA/21CFR",
		RequiresSecondaryValidation: isHighAlert || hasBlackBox || isNarrowTI,
		EvaluatedAt:                 time.Now().UTC(),
		RuleVersion:                 KB1RuleVersion,
		EvidenceLevel:               "Level 1A - FDA Approved Labeling",
	}

	if classSource, ok := m.referenceSources[strings.ToLower(drugClass)]; ok {
		provenance.ClinicalReferenceSource = classSource
	}

	return GovernanceEnhancedResponse{
		Data:            data,
		Governance:      allResults,
		HighestSeverity: GetHighestSeverity(allResults),
		CanProceed:      CanProceed(allResults),
		RequiredSteps:   GetRequiredSteps(allResults),
		Provenance:      provenance,
	}
}
