// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

// ProtocolDescriptor represents metadata about a clinical protocol.
// This is NOT the protocol logic itself (which lives in CQL), but rather
// the metadata needed to determine applicability and priority.
type ProtocolDescriptor struct {
	// Unique identifier for this protocol
	ID string `json:"id" yaml:"id"`

	// Human-readable name
	Name string `json:"name" yaml:"name"`

	// Brief description of what this protocol addresses
	Description string `json:"description" yaml:"description"`

	// Protocol category for grouping and filtering
	Category ProtocolCategory `json:"category" yaml:"category"`

	// Priority class determines order of dominance in conflict resolution
	PriorityClass PriorityClass `json:"priority_class" yaml:"priority_class"`

	// CQL fact IDs that must be true for this protocol to be considered
	// These are evaluated against PatientContext.CQLTruthFlags
	TriggerCriteria []string `json:"trigger_criteria" yaml:"trigger_criteria"`

	// CQL fact IDs that, if true, block this protocol
	ContraindicationRules []string `json:"contraindication_rules" yaml:"contraindication_rules"`

	// KB-8 calculator IDs required for this protocol
	RequiredCalculators []string `json:"required_calculators" yaml:"required_calculators"`

	// Clinical guideline source (e.g., ACC/AHA, SSC, KDIGO)
	GuidelineSource string `json:"guideline_source" yaml:"guideline_source"`

	// Version of the guideline (e.g., "2021", "v2.1")
	GuidelineVersion string `json:"guideline_version" yaml:"guideline_version"`

	// DOI or citation reference
	CitationReference string `json:"citation_reference" yaml:"citation_reference"`

	// Clinical setting applicability
	ApplicableSettings []ClinicalSetting `json:"applicable_settings" yaml:"applicable_settings"`

	// Target population (e.g., adult, pediatric, geriatric)
	TargetPopulation []string `json:"target_population" yaml:"target_population"`

	// Whether this protocol is currently active
	IsActive bool `json:"is_active" yaml:"is_active"`

	// Version of this protocol descriptor
	Version string `json:"version" yaml:"version"`
}

// ProtocolCategory represents the type of clinical protocol.
type ProtocolCategory string

const (
	// CategoryEmergency for life-threatening emergencies (cardiac arrest, anaphylaxis)
	CategoryEmergency ProtocolCategory = "EMERGENCY"

	// CategoryAcute for acute conditions requiring immediate intervention (sepsis, MI)
	CategoryAcute ProtocolCategory = "ACUTE"

	// CategoryICU for ICU-specific protocols (ventilation, sedation)
	CategoryICU ProtocolCategory = "ICU"

	// CategoryChronic for chronic disease management (diabetes, heart failure)
	CategoryChronic ProtocolCategory = "CHRONIC"

	// CategoryPreventive for preventive care (immunizations, screenings)
	CategoryPreventive ProtocolCategory = "PREVENTIVE"

	// CategoryPalliative for palliative and comfort care
	CategoryPalliative ProtocolCategory = "PALLIATIVE"
)

// PriorityClass determines the order of dominance when protocols conflict.
// Lower numbers = higher priority.
type PriorityClass int

const (
	// PriorityEmergency - Life-preserving / resuscitation (highest priority)
	PriorityEmergency PriorityClass = 1

	// PriorityAcute - Organ-failure stabilization
	PriorityAcute PriorityClass = 2

	// PriorityMorbidity - Immediate morbidity prevention
	PriorityMorbidity PriorityClass = 3

	// PriorityChronic - Long-term chronic optimization (lowest priority)
	PriorityChronic PriorityClass = 4
)

// ClinicalSetting represents where the protocol can be applied.
type ClinicalSetting string

const (
	SettingED           ClinicalSetting = "ED"
	SettingICU          ClinicalSetting = "ICU"
	SettingInpatient    ClinicalSetting = "INPATIENT"
	SettingOutpatient   ClinicalSetting = "OUTPATIENT"
	SettingAmbulatory   ClinicalSetting = "AMBULATORY"
	SettingHomeCare     ClinicalSetting = "HOME_CARE"
	SettingLongTermCare ClinicalSetting = "LONG_TERM_CARE"
)

// String returns the string representation of PriorityClass.
func (p PriorityClass) String() string {
	switch p {
	case PriorityEmergency:
		return "EMERGENCY"
	case PriorityAcute:
		return "ACUTE"
	case PriorityMorbidity:
		return "MORBIDITY"
	case PriorityChronic:
		return "CHRONIC"
	default:
		return "UNKNOWN"
	}
}

// IsHigherPriority returns true if p has higher priority than other.
func (p PriorityClass) IsHigherPriority(other PriorityClass) bool {
	return p < other
}

// IsApplicableTo checks if the protocol is applicable to the given setting.
func (pd *ProtocolDescriptor) IsApplicableTo(setting ClinicalSetting) bool {
	if len(pd.ApplicableSettings) == 0 {
		return true // No restrictions means applicable everywhere
	}
	for _, s := range pd.ApplicableSettings {
		if s == setting {
			return true
		}
	}
	return false
}

// RequiresTrigger checks if a specific CQL fact is required for this protocol.
func (pd *ProtocolDescriptor) RequiresTrigger(factID string) bool {
	for _, trigger := range pd.TriggerCriteria {
		if trigger == factID {
			return true
		}
	}
	return false
}

// IsContraindicated checks if a specific CQL fact would contraindicate this protocol.
func (pd *ProtocolDescriptor) IsContraindicated(factID string) bool {
	for _, rule := range pd.ContraindicationRules {
		if rule == factID {
			return true
		}
	}
	return false
}
