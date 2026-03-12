// Package explainability provides clinical decision explanation capabilities
// for KB-4 Patient Safety service. It answers "Why did this alert fire?"
// with traceable, audit-ready explanations.
package explainability

import (
	"fmt"
	"strings"
	"time"

	"kb-patient-safety/pkg/safety"
)

// ExplanationType categorizes the type of explanation
type ExplanationType string

const (
	ExplanationTypeBlackBox         ExplanationType = "BLACK_BOX_WARNING"
	ExplanationTypeContraindication ExplanationType = "CONTRAINDICATION"
	ExplanationTypeAgeLimit         ExplanationType = "AGE_LIMIT"
	ExplanationTypeDoseLimit        ExplanationType = "DOSE_LIMIT"
	ExplanationTypePregnancy        ExplanationType = "PREGNANCY"
	ExplanationTypeLactation        ExplanationType = "LACTATION"
	ExplanationTypeHighAlert        ExplanationType = "HIGH_ALERT"
	ExplanationTypeBeers            ExplanationType = "BEERS_CRITERIA"
	ExplanationTypeAnticholinergic  ExplanationType = "ANTICHOLINERGIC"
	ExplanationTypeLabRequired      ExplanationType = "LAB_REQUIRED"
)

// EvidenceChainLink represents a single link in the evidence chain
type EvidenceChainLink struct {
	Step        int       `json:"step"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ClinicalRationale provides detailed clinical reasoning
type ClinicalRationale struct {
	Summary          string   `json:"summary"`
	MechanismOfRisk  string   `json:"mechanismOfRisk,omitempty"`
	PatientFactors   []string `json:"patientFactors,omitempty"`
	DrugFactors      []string `json:"drugFactors,omitempty"`
	InteractionRisks []string `json:"interactionRisks,omitempty"`
}

// GovernanceTrace provides full audit trail of the knowledge source
type GovernanceTrace struct {
	PrimaryAuthority     string    `json:"primaryAuthority"`
	SourceDocument       string    `json:"sourceDocument"`
	SourceSection        string    `json:"sourceSection,omitempty"`
	SourceURL            string    `json:"sourceUrl,omitempty"`
	SourceVersion        string    `json:"sourceVersion,omitempty"`
	EvidenceLevel        string    `json:"evidenceLevel"`
	Jurisdiction         string    `json:"jurisdiction"`
	KnowledgeVersion     string    `json:"knowledgeVersion"`
	ApprovalStatus       string    `json:"approvalStatus"`
	ApprovedBy           string    `json:"approvedBy,omitempty"`
	ApprovedAt           string    `json:"approvedAt,omitempty"`
	EffectiveDate        string    `json:"effectiveDate"`
	ReviewDate           string    `json:"reviewDate,omitempty"`
	RegulatoryReferences []string  `json:"regulatoryReferences,omitempty"`
}

// AlertExplanation provides a complete explanation for why an alert fired
type AlertExplanation struct {
	// Alert identification
	AlertID   string          `json:"alertId"`
	AlertType ExplanationType `json:"alertType"`
	Severity  string          `json:"severity"`

	// Human-readable explanation
	Title           string `json:"title"`
	PlainEnglish    string `json:"plainEnglish"`    // Non-technical explanation
	ClinicalSummary string `json:"clinicalSummary"` // Technical clinical summary

	// Drug information
	DrugInfo DrugExplanation `json:"drugInfo"`

	// Patient context that triggered the alert
	PatientContext PatientExplanation `json:"patientContext,omitempty"`

	// Evidence chain (traceable reasoning)
	EvidenceChain []EvidenceChainLink `json:"evidenceChain"`

	// Clinical rationale
	ClinicalRationale ClinicalRationale `json:"clinicalRationale"`

	// Full governance trace (for audit)
	GovernanceTrace GovernanceTrace `json:"governanceTrace"`

	// Recommendations
	Recommendations        []string `json:"recommendations"`
	Alternatives           []string `json:"alternatives,omitempty"`
	MonitoringRequirements []string `json:"monitoringRequirements,omitempty"`

	// Override guidance
	CanOverride       bool     `json:"canOverride"`
	OverrideConditions []string `json:"overrideConditions,omitempty"`

	// Metadata
	GeneratedAt time.Time `json:"generatedAt"`
	RequestID   string    `json:"requestId,omitempty"`
}

// DrugExplanation provides drug-specific explanation details
type DrugExplanation struct {
	RxNormCode  string   `json:"rxnormCode"`
	DrugName    string   `json:"drugName"`
	DrugClass   string   `json:"drugClass,omitempty"`
	ATCCode     string   `json:"atcCode,omitempty"`
	TallManName string   `json:"tallManName,omitempty"`
	BrandNames  []string `json:"brandNames,omitempty"`
}

// PatientExplanation provides patient factors that contributed to the alert
type PatientExplanation struct {
	AgeYears    float64  `json:"ageYears,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	IsPregnant  bool     `json:"isPregnant,omitempty"`
	IsLactating bool     `json:"isLactating,omitempty"`
	Conditions  []string `json:"conditions,omitempty"`
	RiskFactors []string `json:"riskFactors,omitempty"`
}

// Explainer generates explanations for safety alerts
type Explainer struct {
	checker *safety.SafetyChecker
}

// NewExplainer creates a new Explainer instance
func NewExplainer(checker *safety.SafetyChecker) *Explainer {
	return &Explainer{
		checker: checker,
	}
}

// lookupGovernanceByAlertType retrieves governance metadata from the knowledge store
// based on the alert type and drug RxNorm code
func (e *Explainer) lookupGovernanceByAlertType(alertType safety.AlertType, rxnormCode string) GovernanceTrace {
	if e.checker == nil || rxnormCode == "" {
		return GovernanceTrace{}
	}

	switch alertType {
	case safety.AlertTypeBlackBox:
		if warning, ok := e.checker.GetBlackBoxWarning(rxnormCode); ok && warning != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(warning.Governance.SourceAuthority),
				SourceDocument:   warning.Governance.SourceDocument,
				SourceSection:    warning.Governance.SourceSection,
				SourceURL:        warning.Governance.SourceURL,
				SourceVersion:    warning.Governance.SourceVersion,
				EvidenceLevel:    string(warning.Governance.EvidenceLevel),
				Jurisdiction:     string(warning.Governance.Jurisdiction),
				KnowledgeVersion: warning.Governance.KnowledgeVersion,
				ApprovalStatus:   string(warning.Governance.ApprovalStatus),
				ApprovedBy:       warning.Governance.ApprovedBy,
				ApprovedAt:       warning.Governance.ApprovedAt,
				EffectiveDate:    warning.Governance.EffectiveDate,
				ReviewDate:       warning.Governance.ReviewDate,
			}
		}
	case safety.AlertTypeHighAlert:
		if med, ok := e.checker.GetHighAlertMedication(rxnormCode); ok && med != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(med.Governance.SourceAuthority),
				SourceDocument:   med.Governance.SourceDocument,
				SourceSection:    med.Governance.SourceSection,
				SourceURL:        med.Governance.SourceURL,
				SourceVersion:    med.Governance.SourceVersion,
				EvidenceLevel:    string(med.Governance.EvidenceLevel),
				Jurisdiction:     string(med.Governance.Jurisdiction),
				KnowledgeVersion: med.Governance.KnowledgeVersion,
				ApprovalStatus:   string(med.Governance.ApprovalStatus),
				ApprovedBy:       med.Governance.ApprovedBy,
				ApprovedAt:       med.Governance.ApprovedAt,
				EffectiveDate:    med.Governance.EffectiveDate,
				ReviewDate:       med.Governance.ReviewDate,
			}
		}
	case safety.AlertTypeBeers:
		if entry, ok := e.checker.GetBeersEntry(rxnormCode); ok && entry != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(entry.Governance.SourceAuthority),
				SourceDocument:   entry.Governance.SourceDocument,
				SourceSection:    entry.Governance.SourceSection,
				SourceURL:        entry.Governance.SourceURL,
				SourceVersion:    entry.Governance.SourceVersion,
				EvidenceLevel:    string(entry.Governance.EvidenceLevel),
				Jurisdiction:     string(entry.Governance.Jurisdiction),
				KnowledgeVersion: entry.Governance.KnowledgeVersion,
				ApprovalStatus:   string(entry.Governance.ApprovalStatus),
				ApprovedBy:       entry.Governance.ApprovedBy,
				ApprovedAt:       entry.Governance.ApprovedAt,
				EffectiveDate:    entry.Governance.EffectiveDate,
				ReviewDate:       entry.Governance.ReviewDate,
			}
		}
	case safety.AlertTypePregnancy:
		if preg, ok := e.checker.GetPregnancySafety(rxnormCode); ok && preg != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(preg.Governance.SourceAuthority),
				SourceDocument:   preg.Governance.SourceDocument,
				SourceSection:    preg.Governance.SourceSection,
				SourceURL:        preg.Governance.SourceURL,
				SourceVersion:    preg.Governance.SourceVersion,
				EvidenceLevel:    string(preg.Governance.EvidenceLevel),
				Jurisdiction:     string(preg.Governance.Jurisdiction),
				KnowledgeVersion: preg.Governance.KnowledgeVersion,
				ApprovalStatus:   string(preg.Governance.ApprovalStatus),
				ApprovedBy:       preg.Governance.ApprovedBy,
				ApprovedAt:       preg.Governance.ApprovedAt,
				EffectiveDate:    preg.Governance.EffectiveDate,
				ReviewDate:       preg.Governance.ReviewDate,
			}
		}
	case safety.AlertTypeLactation:
		if lact, ok := e.checker.GetLactationSafety(rxnormCode); ok && lact != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(lact.Governance.SourceAuthority),
				SourceDocument:   lact.Governance.SourceDocument,
				SourceSection:    lact.Governance.SourceSection,
				SourceURL:        lact.Governance.SourceURL,
				SourceVersion:    lact.Governance.SourceVersion,
				EvidenceLevel:    string(lact.Governance.EvidenceLevel),
				Jurisdiction:     string(lact.Governance.Jurisdiction),
				KnowledgeVersion: lact.Governance.KnowledgeVersion,
				ApprovalStatus:   string(lact.Governance.ApprovalStatus),
				ApprovedBy:       lact.Governance.ApprovedBy,
				ApprovedAt:       lact.Governance.ApprovedAt,
				EffectiveDate:    lact.Governance.EffectiveDate,
				ReviewDate:       lact.Governance.ReviewDate,
			}
		}
	case safety.AlertTypeLabRequired:
		if lab, ok := e.checker.GetLabRequirement(rxnormCode); ok && lab != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(lab.Governance.SourceAuthority),
				SourceDocument:   lab.Governance.SourceDocument,
				SourceSection:    lab.Governance.SourceSection,
				SourceURL:        lab.Governance.SourceURL,
				SourceVersion:    lab.Governance.SourceVersion,
				EvidenceLevel:    string(lab.Governance.EvidenceLevel),
				Jurisdiction:     string(lab.Governance.Jurisdiction),
				KnowledgeVersion: lab.Governance.KnowledgeVersion,
				ApprovalStatus:   string(lab.Governance.ApprovalStatus),
				ApprovedBy:       lab.Governance.ApprovedBy,
				ApprovedAt:       lab.Governance.ApprovedAt,
				EffectiveDate:    lab.Governance.EffectiveDate,
				ReviewDate:       lab.Governance.ReviewDate,
			}
		}
	case safety.AlertTypeAnticholinergic:
		if acb, ok := e.checker.GetAnticholinergicBurden(rxnormCode); ok && acb != nil {
			return GovernanceTrace{
				PrimaryAuthority: string(acb.Governance.SourceAuthority),
				SourceDocument:   acb.Governance.SourceDocument,
				SourceSection:    acb.Governance.SourceSection,
				SourceURL:        acb.Governance.SourceURL,
				SourceVersion:    acb.Governance.SourceVersion,
				EvidenceLevel:    string(acb.Governance.EvidenceLevel),
				Jurisdiction:     string(acb.Governance.Jurisdiction),
				KnowledgeVersion: acb.Governance.KnowledgeVersion,
				ApprovalStatus:   string(acb.Governance.ApprovalStatus),
				ApprovedBy:       acb.Governance.ApprovedBy,
				ApprovedAt:       acb.Governance.ApprovedAt,
				EffectiveDate:    acb.Governance.EffectiveDate,
				ReviewDate:       acb.Governance.ReviewDate,
			}
		}
	}

	return GovernanceTrace{}
}

// ExplainAlert generates a complete explanation for a safety alert
func (e *Explainer) ExplainAlert(alert *safety.SafetyAlert, patientCtx *safety.PatientContext) *AlertExplanation {
	explanation := &AlertExplanation{
		AlertID:     alert.ID,
		AlertType:   ExplanationType(alert.Type),
		Severity:    string(alert.Severity),
		Title:       alert.Title,
		GeneratedAt: time.Now().UTC(),
	}

	// Extract drug info from alert
	if alert.DrugInfo != nil {
		explanation.DrugInfo = DrugExplanation{
			RxNormCode: alert.DrugInfo.RxNormCode,
			DrugName:   alert.DrugInfo.DrugName,
			DrugClass:  alert.DrugInfo.DrugClass,
		}
	}

	// Build explanation based on alert type
	switch alert.Type {
	case safety.AlertTypeBlackBox:
		e.explainBlackBox(explanation, alert)
	case safety.AlertTypeContraindication:
		e.explainContraindication(explanation, alert)
	case safety.AlertTypeAgeLimit:
		e.explainAgeLimit(explanation, alert, patientCtx)
	case safety.AlertTypeDoseLimit:
		e.explainDoseLimit(explanation, alert)
	case safety.AlertTypePregnancy:
		e.explainPregnancy(explanation, alert, patientCtx)
	case safety.AlertTypeLactation:
		e.explainLactation(explanation, alert, patientCtx)
	case safety.AlertTypeHighAlert:
		e.explainHighAlert(explanation, alert)
	case safety.AlertTypeBeers:
		e.explainBeers(explanation, alert, patientCtx)
	case safety.AlertTypeAnticholinergic:
		e.explainAnticholinergic(explanation, alert, patientCtx)
	case safety.AlertTypeLabRequired:
		e.explainLabRequired(explanation, alert)
	default:
		e.explainGeneric(explanation, alert)
	}

	// Set patient context if provided
	if patientCtx != nil {
		explanation.PatientContext = PatientExplanation{
			AgeYears:    patientCtx.AgeYears,
			Gender:      patientCtx.Gender,
			IsPregnant:  patientCtx.IsPregnant,
			IsLactating: patientCtx.IsLactating,
		}
	}

	// Add override guidance
	explanation.CanOverride = alert.CanOverride

	return explanation
}

// ExplainBlackBoxWarning generates explanation for a black box warning by RxNorm code
func (e *Explainer) ExplainBlackBoxWarning(rxnormCode string) *AlertExplanation {
	warning, found := e.checker.GetBlackBoxWarning(rxnormCode)
	if !found {
		return nil
	}

	explanation := &AlertExplanation{
		AlertType:   ExplanationTypeBlackBox,
		Severity:    string(warning.Severity),
		Title:       fmt.Sprintf("Black Box Warning: %s", warning.DrugName),
		GeneratedAt: time.Now().UTC(),
	}

	explanation.DrugInfo = DrugExplanation{
		RxNormCode: warning.RxNormCode,
		DrugName:   warning.DrugName,
		DrugClass:  warning.DrugClass,
		ATCCode:    warning.ATCCode,
	}

	// Plain English explanation
	explanation.PlainEnglish = fmt.Sprintf(
		"%s has the FDA's strongest safety warning (Black Box Warning) because it can cause serious or life-threatening side effects. "+
			"This warning is displayed prominently on the drug's packaging and prescribing information.",
		warning.DrugName,
	)

	// Clinical summary
	explanation.ClinicalSummary = warning.WarningText

	// Build evidence chain
	explanation.EvidenceChain = []EvidenceChainLink{
		{
			Step:        1,
			Description: fmt.Sprintf("Drug identified: %s (RxNorm: %s)", warning.DrugName, warning.RxNormCode),
			Source:      "RxNorm Database",
			Timestamp:   time.Now().UTC(),
		},
		{
			Step:        2,
			Description: fmt.Sprintf("Black Box Warning detected with risk categories: %s", strings.Join(warning.RiskCategories, ", ")),
			Source:      warning.Governance.SourceDocument,
			SourceURL:   warning.Governance.SourceURL,
			Timestamp:   time.Now().UTC(),
		},
		{
			Step:        3,
			Description: "Alert generated based on FDA's strongest safety classification",
			Source:      "KB-4 Patient Safety Engine",
			Timestamp:   time.Now().UTC(),
		},
	}

	// Clinical rationale
	explanation.ClinicalRationale = ClinicalRationale{
		Summary:         "Black Box Warnings are the FDA's strongest warning for drugs with serious safety risks. These warnings indicate that the drug has risks that are significant enough to require prominent display.",
		MechanismOfRisk: fmt.Sprintf("Risk categories: %s", strings.Join(warning.RiskCategories, ", ")),
		DrugFactors:     warning.RiskCategories,
	}

	// Governance trace
	explanation.GovernanceTrace = GovernanceTrace{
		PrimaryAuthority: string(warning.Governance.SourceAuthority),
		SourceDocument:   warning.Governance.SourceDocument,
		SourceSection:    warning.Governance.SourceSection,
		SourceURL:        warning.Governance.SourceURL,
		SourceVersion:    warning.Governance.SourceVersion,
		EvidenceLevel:    string(warning.Governance.EvidenceLevel),
		Jurisdiction:     string(warning.Governance.Jurisdiction),
		KnowledgeVersion: warning.Governance.KnowledgeVersion,
		ApprovalStatus:   string(warning.Governance.ApprovalStatus),
		ApprovedBy:       warning.Governance.ApprovedBy,
		ApprovedAt:       warning.Governance.ApprovedAt,
		EffectiveDate:    warning.Governance.EffectiveDate,
		ReviewDate:       warning.Governance.ReviewDate,
	}

	// Recommendations
	explanation.Recommendations = []string{
		"Review the full Black Box Warning in the prescribing information",
		"Assess patient's individual risk factors",
		"Consider risk-benefit ratio for this patient",
		"Document informed consent discussion",
		"Plan appropriate monitoring",
	}

	explanation.CanOverride = true
	explanation.OverrideConditions = []string{
		"Clinical benefit outweighs documented risks",
		"No suitable alternative therapy available",
		"Patient/guardian informed consent obtained",
		"Appropriate monitoring plan in place",
	}

	return explanation
}

// Helper methods for different alert types

func (e *Explainer) explainBlackBox(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = fmt.Sprintf(
		"This medication (%s) has the FDA's strongest safety warning (Black Box Warning). "+
			"This warning indicates serious or life-threatening risks that require careful consideration before prescribing.",
		explanation.DrugInfo.DrugName,
	)
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "FDA Drug Label"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Drug identified with Black Box Warning", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 2, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "FDA's Black Box Warning indicates the highest level of drug safety concern.",
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainContraindication(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = fmt.Sprintf(
		"This medication should not be used in this patient due to a contraindication. "+
			"A contraindication means there is a medical reason why this drug could be harmful.",
	)
	explanation.ClinicalSummary = alert.Message

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Drug contraindication identified", Source: "FDA Drug Label Section 4", Timestamp: time.Now().UTC()},
		{Step: 2, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "Contraindications indicate situations where a drug should not be used due to potential harm.",
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainAgeLimit(explanation *AlertExplanation, alert *safety.SafetyAlert, patientCtx *safety.PatientContext) {
	ageText := "unknown age"
	if patientCtx != nil {
		ageText = fmt.Sprintf("%.1f years old", patientCtx.AgeYears)
	}

	explanation.PlainEnglish = fmt.Sprintf(
		"This medication has age restrictions. The patient is %s, which is outside the approved age range for this medication.",
		ageText,
	)
	explanation.ClinicalSummary = alert.Message

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: fmt.Sprintf("Patient age: %s", ageText), Source: "Patient Record", Timestamp: time.Now().UTC()},
		{Step: 2, Description: "Age restriction identified", Source: "FDA Pediatric/Geriatric Guidelines", Timestamp: time.Now().UTC()},
		{Step: 3, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "Age restrictions exist because drug safety and efficacy may differ across age groups.",
	}
	if patientCtx != nil {
		explanation.ClinicalRationale.PatientFactors = []string{
			fmt.Sprintf("Age: %.1f years", patientCtx.AgeYears),
		}
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainDoseLimit(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = "The proposed dose exceeds the maximum recommended dose for this medication. " +
		"Higher doses may increase the risk of adverse effects."
	explanation.ClinicalSummary = alert.Message

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Proposed dose evaluated", Source: "Prescription Order", Timestamp: time.Now().UTC()},
		{Step: 2, Description: "Dose exceeds maximum limit", Source: "FDA Dosing Guidelines", Timestamp: time.Now().UTC()},
		{Step: 3, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "Maximum dose limits are established to prevent toxicity and adverse effects.",
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainPregnancy(explanation *AlertExplanation, alert *safety.SafetyAlert, patientCtx *safety.PatientContext) {
	explanation.PlainEnglish = "This medication may pose risks during pregnancy. " +
		"The safety of this drug during pregnancy has been evaluated, and caution is recommended."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "FDA PLLR Label"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Patient identified as pregnant", Source: "Patient Record", Timestamp: time.Now().UTC()},
		{Step: 2, Description: "Pregnancy safety information retrieved", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 3, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary:        "Pregnancy safety categories help assess potential risks to the fetus.",
		PatientFactors: []string{"Pregnancy status: Positive"},
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainLactation(explanation *AlertExplanation, alert *safety.SafetyAlert, patientCtx *safety.PatientContext) {
	explanation.PlainEnglish = "This medication may pass into breast milk and could affect a nursing infant. " +
		"The safety during breastfeeding has been evaluated."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "NIH LactMed Database"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Patient identified as lactating", Source: "Patient Record", Timestamp: time.Now().UTC()},
		{Step: 2, Description: "Lactation safety information retrieved", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 3, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary:        "Lactation safety assessment considers drug transfer to breast milk and potential infant effects.",
		PatientFactors: []string{"Lactation status: Active"},
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainHighAlert(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = "This is a high-alert medication, which means it has a higher risk of causing " +
		"significant patient harm if used incorrectly. Extra safety measures are recommended."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "ISMP High-Alert Medications List"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Drug identified as high-alert medication", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 2, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "High-alert medications bear a heightened risk of causing significant patient harm when used in error. " +
			"They require additional safeguards to reduce the risk of errors.",
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainBeers(explanation *AlertExplanation, alert *safety.SafetyAlert, patientCtx *safety.PatientContext) {
	explanation.PlainEnglish = "This medication is listed in the Beers Criteria as potentially inappropriate for older adults. " +
		"Alternative medications may be safer for elderly patients."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "AGS Beers Criteria 2023"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Patient age ≥65 years identified", Source: "Patient Record", Timestamp: time.Now().UTC()},
		{Step: 2, Description: "Drug listed in AGS Beers Criteria", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 3, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "The Beers Criteria identifies medications that are potentially inappropriate for older adults " +
			"due to increased risks of adverse effects in this population.",
	}
	if patientCtx != nil && patientCtx.AgeYears >= 65 {
		explanation.ClinicalRationale.PatientFactors = []string{
			fmt.Sprintf("Age: %.0f years (≥65, Beers Criteria applies)", patientCtx.AgeYears),
		}
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainAnticholinergic(explanation *AlertExplanation, alert *safety.SafetyAlert, patientCtx *safety.PatientContext) {
	explanation.PlainEnglish = "This medication has anticholinergic effects, which can cause cognitive impairment, " +
		"dry mouth, constipation, and other side effects, especially in older adults."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "ACB Scale"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Drug anticholinergic burden assessed", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 2, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "Anticholinergic burden is cumulative across medications. High burden is associated with " +
			"increased risk of cognitive impairment, falls, and delirium, particularly in elderly patients.",
	}

	// Add patient age context for geriatric relevance
	if patientCtx != nil && patientCtx.AgeYears >= 65 {
		explanation.ClinicalRationale.PatientFactors = []string{
			fmt.Sprintf("Age: %.0f years (≥65, increased ACB sensitivity)", patientCtx.AgeYears),
		}
	}

	explanation.Recommendations = alert.Recommendations
}

func (e *Explainer) explainLabRequired(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = "This medication requires laboratory monitoring to ensure safe use. " +
		"Regular lab tests help detect potential problems early."
	explanation.ClinicalSummary = alert.Message

	// Lookup governance metadata from knowledge store
	govTrace := e.lookupGovernanceByAlertType(alert.Type, explanation.DrugInfo.RxNormCode)
	if govTrace.PrimaryAuthority != "" {
		explanation.GovernanceTrace = govTrace
	}

	sourceDoc := "FDA Drug Label"
	if govTrace.SourceDocument != "" {
		sourceDoc = govTrace.SourceDocument
	}

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Lab monitoring requirements identified", Source: sourceDoc, SourceURL: govTrace.SourceURL, Timestamp: time.Now().UTC()},
		{Step: 2, Description: alert.ClinicalRationale, Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: "Laboratory monitoring is required to detect adverse effects early and adjust therapy as needed.",
	}

	explanation.Recommendations = alert.Recommendations
	explanation.MonitoringRequirements = alert.Recommendations
}

func (e *Explainer) explainGeneric(explanation *AlertExplanation, alert *safety.SafetyAlert) {
	explanation.PlainEnglish = "A safety concern has been identified for this medication."
	explanation.ClinicalSummary = alert.Message

	explanation.EvidenceChain = []EvidenceChainLink{
		{Step: 1, Description: "Safety alert generated", Source: "KB-4 Safety Knowledge Base", Timestamp: time.Now().UTC()},
	}

	explanation.ClinicalRationale = ClinicalRationale{
		Summary: alert.ClinicalRationale,
	}

	explanation.Recommendations = alert.Recommendations
}

// ExplainSafetyCheckResult generates explanations for all alerts in a safety check result
func (e *Explainer) ExplainSafetyCheckResult(result *safety.SafetyCheckResponse, patientCtx *safety.PatientContext) *SafetyCheckExplanation {
	explanation := &SafetyCheckExplanation{
		RequestID:           result.RequestID,
		CheckedAt:           result.CheckedAt,
		IsSafe:              result.Safe,
		RequiresAction:      result.RequiresAction,
		BlocksPrescribing:   result.BlockPrescribing,
		TotalAlerts:         result.TotalAlerts,
		CriticalAlertCount:  result.CriticalAlerts,
		HighAlertCount:      result.HighAlerts,
		ModerateAlertCount:  result.ModerateAlerts,
		LowAlertCount:       result.LowAlerts,
		AlertExplanations:   make([]*AlertExplanation, 0, len(result.Alerts)),
		GeneratedAt:         time.Now().UTC(),
	}

	// Generate summary
	if result.Safe {
		explanation.Summary = "No significant safety concerns identified for this prescription."
	} else if result.BlockPrescribing {
		explanation.Summary = "CRITICAL: This prescription has safety concerns that should block prescribing without review."
	} else if result.RequiresAction {
		explanation.Summary = "Safety alerts require clinician review before proceeding."
	}

	// Explain each alert
	for i := range result.Alerts {
		alertExplanation := e.ExplainAlert(&result.Alerts[i], patientCtx)
		explanation.AlertExplanations = append(explanation.AlertExplanations, alertExplanation)
	}

	return explanation
}

// SafetyCheckExplanation provides explanation for a complete safety check
type SafetyCheckExplanation struct {
	RequestID           string              `json:"requestId"`
	CheckedAt           time.Time           `json:"checkedAt"`
	GeneratedAt         time.Time           `json:"generatedAt"`
	Summary             string              `json:"summary"`
	IsSafe              bool                `json:"isSafe"`
	RequiresAction      bool                `json:"requiresAction"`
	BlocksPrescribing   bool                `json:"blocksPrescribing"`
	TotalAlerts         int                 `json:"totalAlerts"`
	CriticalAlertCount  int                 `json:"criticalAlertCount"`
	HighAlertCount      int                 `json:"highAlertCount"`
	ModerateAlertCount  int                 `json:"moderateAlertCount"`
	LowAlertCount       int                 `json:"lowAlertCount"`
	AlertExplanations   []*AlertExplanation `json:"alertExplanations"`
}
