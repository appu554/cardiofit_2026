package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Override Governance Framework
// Implements hierarchical override levels with role-based authority validation

// OverrideLevel represents the hierarchical levels of clinical override authority
type OverrideLevel int

const (
	OverrideLevelNone OverrideLevel = iota
	OverrideLevelClinicalJudgment
	OverrideLevelPeerReview
	OverrideLevelSupervisory
	OverrideLevelEmergency
)

// String returns the string representation of override level
func (ol OverrideLevel) String() string {
	switch ol {
	case OverrideLevelClinicalJudgment:
		return "CLINICAL_JUDGMENT"
	case OverrideLevelPeerReview:
		return "PEER_REVIEW"
	case OverrideLevelSupervisory:
		return "SUPERVISORY"
	case OverrideLevelEmergency:
		return "EMERGENCY"
	default:
		return "NONE"
	}
}

// OverrideLevelFromString converts string to OverrideLevel
func OverrideLevelFromString(s string) OverrideLevel {
	switch s {
	case "CLINICAL_JUDGMENT":
		return OverrideLevelClinicalJudgment
	case "PEER_REVIEW":
		return OverrideLevelPeerReview
	case "SUPERVISORY":
		return OverrideLevelSupervisory
	case "EMERGENCY":
		return OverrideLevelEmergency
	default:
		return OverrideLevelNone
	}
}

// AuthorityLevel represents clinician authority levels
type AuthorityLevel int

const (
	AuthorityLevelNone AuthorityLevel = iota
	AuthorityLevelResident
	AuthorityLevelAttending
	AuthorityLevelSpecialist
	AuthorityLevelDepartmentHead
	AuthorityLevelChiefMedicalOfficer
)

// String returns the string representation of authority level
func (al AuthorityLevel) String() string {
	switch al {
	case AuthorityLevelResident:
		return "RESIDENT"
	case AuthorityLevelAttending:
		return "ATTENDING"
	case AuthorityLevelSpecialist:
		return "SPECIALIST"
	case AuthorityLevelDepartmentHead:
		return "DEPARTMENT_HEAD"
	case AuthorityLevelChiefMedicalOfficer:
		return "CHIEF_MEDICAL_OFFICER"
	default:
		return "NONE"
	}
}

// AuthorityLevelFromString converts string to AuthorityLevel
func AuthorityLevelFromString(s string) AuthorityLevel {
	switch s {
	case "RESIDENT":
		return AuthorityLevelResident
	case "ATTENDING":
		return AuthorityLevelAttending
	case "SPECIALIST":
		return AuthorityLevelSpecialist
	case "DEPARTMENT_HEAD":
		return AuthorityLevelDepartmentHead
	case "CHIEF_MEDICAL_OFFICER":
		return AuthorityLevelChiefMedicalOfficer
	default:
		return AuthorityLevelNone
	}
}

// OverrideGovernanceRule defines the governance rules for override levels
type OverrideGovernanceRule struct {
	Level                   OverrideLevel     `json:"level"`
	RequiredAuthority       []AuthorityLevel  `json:"required_authority"`
	RequiresCoSignature     bool              `json:"requires_co_signature"`
	RequiresPeerReview      bool              `json:"requires_peer_review"`
	MaxDuration             time.Duration     `json:"max_duration"`
	AuditLevel              AuditLevel        `json:"audit_level"`
	ReviewPeriod            time.Duration     `json:"review_period"`
	AutoEscalationTimeout   time.Duration     `json:"auto_escalation_timeout"`
	AllowedReasons          []OverrideReason  `json:"allowed_reasons"`
	EmergencyBypass         bool              `json:"emergency_bypass"`
}

// AuditLevel represents the level of audit required for overrides
type AuditLevel int

const (
	AuditLevelStandard AuditLevel = iota
	AuditLevelEnhanced
	AuditLevelRealTime
	AuditLevelCommitteeReview
)

// String returns the string representation of audit level
func (al AuditLevel) String() string {
	switch al {
	case AuditLevelStandard:
		return "STANDARD"
	case AuditLevelEnhanced:
		return "ENHANCED"
	case AuditLevelRealTime:
		return "REAL_TIME"
	case AuditLevelCommitteeReview:
		return "COMMITTEE_REVIEW"
	default:
		return "STANDARD"
	}
}

// OverrideReason represents categorized reasons for overrides
type OverrideReason struct {
	Code        string `json:"code"`
	Category    string `json:"category"`
	Description string `json:"description"`
	FreeText    string `json:"free_text,omitempty"`
}

// Clinician represents a healthcare provider with authority information
type Clinician struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Role            string         `json:"role"`
	Department      string         `json:"department"`
	AuthorityLevel  AuthorityLevel `json:"authority_level"`
	Specialties     []string       `json:"specialties"`
	Available       bool           `json:"available"`
	ContactMethods  []string       `json:"contact_methods"`
}

// OverrideRequest represents a request for clinical override
type OverrideRequest struct {
	ID               string            `json:"id"`
	WorkflowID       string            `json:"workflow_id"`
	ValidationID     string            `json:"validation_id"`
	Verdict          string            `json:"verdict"`
	Findings         []ValidationFinding `json:"findings"`
	RequiredLevel    OverrideLevel     `json:"required_level"`
	RequestedBy      Clinician         `json:"requested_by"`
	RequestedAt      time.Time         `json:"requested_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	Urgency          ReviewUrgency     `json:"urgency"`
	Status           OverrideStatus    `json:"status"`
	CoSignRequired   bool              `json:"co_sign_required"`
	PeerReviewID     *string           `json:"peer_review_id,omitempty"`
	Evidence         interface{}       `json:"evidence,omitempty"`
}

// OverrideDecision represents the clinician's decision on an override request
type OverrideDecision struct {
	ID                    string            `json:"id"`
	RequestID             string            `json:"request_id"`
	SessionID             string            `json:"session_id"`
	WorkflowID            string            `json:"workflow_id"`
	Decision              OverrideAction    `json:"decision"`
	DecidedBy             Clinician         `json:"decided_by"`
	DecidedAt             time.Time         `json:"decided_at"`
	OverrideLevel         OverrideLevel     `json:"override_level"`
	Reason                OverrideReason    `json:"reason"`
	ClinicalJustification string            `json:"clinical_justification"`
	CoSignature           *CoSignature      `json:"co_signature,omitempty"`
	AlternativeAction     *AlternativeAction `json:"alternative_action,omitempty"`
	RiskAcceptance        *RiskAcceptance   `json:"risk_acceptance,omitempty"`
	MonitoringPlan        *MonitoringPlan   `json:"monitoring_plan,omitempty"`
	Conditions            []string          `json:"conditions,omitempty"`
}

// OverrideAction represents possible override actions
type OverrideAction string

const (
	OverrideActionApprove  OverrideAction = "OVERRIDE"
	OverrideActionModify   OverrideAction = "MODIFY"
	OverrideActionCancel   OverrideAction = "CANCEL"
	OverrideActionDefer    OverrideAction = "DEFER"
	OverrideActionEscalate OverrideAction = "ESCALATE"
)

// OverrideStatus represents the current status of an override request
type OverrideStatus string

const (
	OverrideStatusPending   OverrideStatus = "PENDING"
	OverrideStatusInReview  OverrideStatus = "IN_REVIEW"
	OverrideStatusApproved  OverrideStatus = "APPROVED"
	OverrideStatusRejected  OverrideStatus = "REJECTED"
	OverrideStatusExpired   OverrideStatus = "EXPIRED"
	OverrideStatusCancelled OverrideStatus = "CANCELLED"
	OverrideStatusEscalated OverrideStatus = "ESCALATED"
)

// ReviewUrgency represents the urgency level for override review
type ReviewUrgency string

const (
	UrgencyRoutine   ReviewUrgency = "ROUTINE"   // 4 hours
	UrgencyUrgent    ReviewUrgency = "URGENT"    // 1 hour
	UrgencyStat      ReviewUrgency = "STAT"      // 15 minutes
	UrgencyEmergency ReviewUrgency = "EMERGENCY" // Immediate
)

// CoSignature represents a co-signature from another clinician
type CoSignature struct {
	ClinicianID   string    `json:"clinician_id"`
	Clinician     Clinician `json:"clinician"`
	Signature     string    `json:"signature"`
	Timestamp     time.Time `json:"timestamp"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
}

// AlternativeAction represents an alternative approach suggested by the clinician
type AlternativeAction struct {
	Type             string      `json:"type"`
	ModifiedProposal interface{} `json:"modified_proposal,omitempty"`
	DeferralPeriod   string      `json:"deferral_period,omitempty"`
	Reason           string      `json:"reason"`
}

// RiskAcceptance represents explicit acknowledgment of risks
type RiskAcceptance struct {
	RisksAcknowledged []string  `json:"risks_acknowledged"`
	ClinicianID       string    `json:"clinician_id"`
	Timestamp         time.Time `json:"timestamp"`
	Signature         string    `json:"signature"`
}

// MonitoringPlan represents a plan for monitoring after override
type MonitoringPlan struct {
	Parameters       []string  `json:"parameters"`
	Frequency        string    `json:"frequency"`
	Duration         string    `json:"duration"`
	AlertThresholds  []string  `json:"alert_thresholds"`
	ResponsibleParty string    `json:"responsible_party"`
	FollowUpDate     time.Time `json:"follow_up_date"`
}

// ValidationFinding represents a validation finding from the safety gateway
type ValidationFinding struct {
	FindingID            string      `json:"finding_id"`
	Severity             string      `json:"severity"`
	Category             string      `json:"category"`
	Description          string      `json:"description"`
	ClinicalSignificance string      `json:"clinical_significance"`
	Recommendation       string      `json:"recommendation"`
	Overridable          bool        `json:"overridable"`
	Evidence             interface{} `json:"evidence,omitempty"`
	RiskScore            float64     `json:"risk_score"`
	Source               string      `json:"source"`
}

// OverrideGovernance provides governance rules and validation for clinical overrides
type OverrideGovernance struct {
	rules map[OverrideLevel]OverrideGovernanceRule
}

// NewOverrideGovernance creates a new override governance instance with default rules
func NewOverrideGovernance() *OverrideGovernance {
	return &OverrideGovernance{
		rules: getDefaultGovernanceRules(),
	}
}

// getDefaultGovernanceRules returns the default governance rules for override levels
func getDefaultGovernanceRules() map[OverrideLevel]OverrideGovernanceRule {
	return map[OverrideLevel]OverrideGovernanceRule{
		OverrideLevelClinicalJudgment: {
			Level: OverrideLevelClinicalJudgment,
			RequiredAuthority: []AuthorityLevel{
				AuthorityLevelAttending,
				AuthorityLevelSpecialist,
				AuthorityLevelDepartmentHead,
				AuthorityLevelChiefMedicalOfficer,
			},
			RequiresCoSignature:     false,
			RequiresPeerReview:      false,
			MaxDuration:             4 * time.Hour,
			AuditLevel:              AuditLevelStandard,
			ReviewPeriod:            24 * time.Hour,
			AutoEscalationTimeout:   4 * time.Hour,
			EmergencyBypass:         false,
		},
		OverrideLevelPeerReview: {
			Level: OverrideLevelPeerReview,
			RequiredAuthority: []AuthorityLevel{
				AuthorityLevelAttending,
				AuthorityLevelSpecialist,
				AuthorityLevelDepartmentHead,
				AuthorityLevelChiefMedicalOfficer,
			},
			RequiresCoSignature:     true,
			RequiresPeerReview:      true,
			MaxDuration:             2 * time.Hour,
			AuditLevel:              AuditLevelEnhanced,
			ReviewPeriod:            12 * time.Hour,
			AutoEscalationTimeout:   2 * time.Hour,
			EmergencyBypass:         false,
		},
		OverrideLevelSupervisory: {
			Level: OverrideLevelSupervisory,
			RequiredAuthority: []AuthorityLevel{
				AuthorityLevelDepartmentHead,
				AuthorityLevelChiefMedicalOfficer,
			},
			RequiresCoSignature:     true,
			RequiresPeerReview:      false,
			MaxDuration:             1 * time.Hour,
			AuditLevel:              AuditLevelRealTime,
			ReviewPeriod:            6 * time.Hour,
			AutoEscalationTimeout:   1 * time.Hour,
			EmergencyBypass:         false,
		},
		OverrideLevelEmergency: {
			Level: OverrideLevelEmergency,
			RequiredAuthority: []AuthorityLevel{
				AuthorityLevelResident,
				AuthorityLevelAttending,
				AuthorityLevelSpecialist,
				AuthorityLevelDepartmentHead,
				AuthorityLevelChiefMedicalOfficer,
			},
			RequiresCoSignature:     false,
			RequiresPeerReview:      false,
			MaxDuration:             15 * time.Minute,
			AuditLevel:              AuditLevelCommitteeReview,
			ReviewPeriod:            24 * time.Hour,
			AutoEscalationTimeout:   15 * time.Minute,
			EmergencyBypass:         true,
		},
	}
}

// DetermineRequiredOverrideLevel determines the required override level based on validation findings
func (og *OverrideGovernance) DetermineRequiredOverrideLevel(verdict string, findings []ValidationFinding) OverrideLevel {
	criticalCount := 0
	highCount := 0
	totalRiskScore := 0.0

	for _, finding := range findings {
		switch finding.Severity {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		}
		totalRiskScore += finding.RiskScore
	}

	avgRiskScore := totalRiskScore / float64(len(findings))

	// Determine override level based on findings severity and risk
	switch verdict {
	case "UNSAFE":
		if criticalCount > 2 || avgRiskScore > 8.0 {
			return OverrideLevelSupervisory
		}
		if criticalCount > 0 || avgRiskScore > 6.0 {
			return OverrideLevelPeerReview
		}
		return OverrideLevelClinicalJudgment

	case "WARNING":
		if criticalCount > 0 || avgRiskScore > 7.0 {
			return OverrideLevelPeerReview
		}
		if highCount > 3 || avgRiskScore > 5.0 {
			return OverrideLevelClinicalJudgment
		}
		return OverrideLevelClinicalJudgment

	default:
		return OverrideLevelNone
	}
}

// ValidateOverrideAuthority validates if a clinician has authority for the required override level
func (og *OverrideGovernance) ValidateOverrideAuthority(clinician Clinician, requiredLevel OverrideLevel) error {
	rule, exists := og.rules[requiredLevel]
	if !exists {
		return fmt.Errorf("no governance rule found for override level: %s", requiredLevel.String())
	}

	// Check if clinician's authority level is sufficient
	hasAuthority := false
	for _, authorizedLevel := range rule.RequiredAuthority {
		if clinician.AuthorityLevel == authorizedLevel {
			hasAuthority = true
			break
		}
	}

	if !hasAuthority {
		return fmt.Errorf("clinician authority level %s insufficient for override level %s",
			clinician.AuthorityLevel.String(), requiredLevel.String())
	}

	return nil
}

// ValidateOverrideDecision validates an override decision against governance rules
func (og *OverrideGovernance) ValidateOverrideDecision(request *OverrideRequest, decision *OverrideDecision) error {
	rule, exists := og.rules[decision.OverrideLevel]
	if !exists {
		return fmt.Errorf("no governance rule found for override level: %s", decision.OverrideLevel.String())
	}

	// Validate authority
	if err := og.ValidateOverrideAuthority(decision.DecidedBy, decision.OverrideLevel); err != nil {
		return err
	}

	// Validate co-signature requirement
	if rule.RequiresCoSignature && decision.CoSignature == nil {
		return fmt.Errorf("co-signature required for override level: %s", decision.OverrideLevel.String())
	}

	// Validate timing
	if time.Now().After(request.ExpiresAt) {
		return fmt.Errorf("override request has expired")
	}

	// Validate clinical justification
	if decision.ClinicalJustification == "" {
		return fmt.Errorf("clinical justification required for all overrides")
	}

	return nil
}

// GetOverrideGovernanceRule returns the governance rule for a specific override level
func (og *OverrideGovernance) GetOverrideGovernanceRule(level OverrideLevel) (OverrideGovernanceRule, error) {
	rule, exists := og.rules[level]
	if !exists {
		return OverrideGovernanceRule{}, fmt.Errorf("no governance rule found for override level: %s", level.String())
	}
	return rule, nil
}

// CreateOverrideRequest creates a new override request
func (og *OverrideGovernance) CreateOverrideRequest(
	workflowID, validationID, verdict string,
	findings []ValidationFinding,
	requestedBy Clinician,
	urgency ReviewUrgency,
) (*OverrideRequest, error) {

	requiredLevel := og.DetermineRequiredOverrideLevel(verdict, findings)

	if requiredLevel == OverrideLevelNone {
		return nil, fmt.Errorf("no override required for verdict: %s", verdict)
	}

	rule := og.rules[requiredLevel]

	request := &OverrideRequest{
		ID:               fmt.Sprintf("override_%s", uuid.New().String()),
		WorkflowID:       workflowID,
		ValidationID:     validationID,
		Verdict:          verdict,
		Findings:         findings,
		RequiredLevel:    requiredLevel,
		RequestedBy:      requestedBy,
		RequestedAt:      time.Now(),
		ExpiresAt:        time.Now().Add(rule.MaxDuration),
		Urgency:          urgency,
		Status:           OverrideStatusPending,
		CoSignRequired:   rule.RequiresCoSignature,
	}

	// Adjust expiration based on urgency
	request.ExpiresAt = og.calculateExpirationTime(urgency, rule.MaxDuration)

	return request, nil
}

// calculateExpirationTime calculates expiration time based on urgency and max duration
func (og *OverrideGovernance) calculateExpirationTime(urgency ReviewUrgency, maxDuration time.Duration) time.Time {
	var urgencyDuration time.Duration

	switch urgency {
	case UrgencyEmergency:
		urgencyDuration = 5 * time.Minute
	case UrgencyStat:
		urgencyDuration = 15 * time.Minute
	case UrgencyUrgent:
		urgencyDuration = 1 * time.Hour
	case UrgencyRoutine:
		urgencyDuration = 4 * time.Hour
	default:
		urgencyDuration = maxDuration
	}

	// Use the shorter of urgency duration or max rule duration
	if urgencyDuration < maxDuration {
		return time.Now().Add(urgencyDuration)
	}

	return time.Now().Add(maxDuration)
}

// IsEmergencyBypassAllowed checks if emergency bypass is allowed for the override level
func (og *OverrideGovernance) IsEmergencyBypassAllowed(level OverrideLevel) bool {
	rule, exists := og.rules[level]
	return exists && rule.EmergencyBypass
}

// GetEscalationPath returns the escalation path for an override level
func (og *OverrideGovernance) GetEscalationPath(level OverrideLevel) []AuthorityLevel {
	var path []AuthorityLevel

	switch level {
	case OverrideLevelClinicalJudgment:
		path = []AuthorityLevel{AuthorityLevelSpecialist, AuthorityLevelDepartmentHead}
	case OverrideLevelPeerReview:
		path = []AuthorityLevel{AuthorityLevelDepartmentHead, AuthorityLevelChiefMedicalOfficer}
	case OverrideLevelSupervisory:
		path = []AuthorityLevel{AuthorityLevelChiefMedicalOfficer}
	case OverrideLevelEmergency:
		path = []AuthorityLevel{} // No escalation for emergency
	}

	return path
}

// JSON marshaling support

func (or *OverrideRequest) MarshalJSON() ([]byte, error) {
	type Alias OverrideRequest
	return json.Marshal(&struct {
		RequiredLevel string `json:"required_level"`
		Urgency       string `json:"urgency"`
		Status        string `json:"status"`
		*Alias
	}{
		RequiredLevel: or.RequiredLevel.String(),
		Urgency:       string(or.Urgency),
		Status:        string(or.Status),
		Alias:         (*Alias)(or),
	})
}

func (od *OverrideDecision) MarshalJSON() ([]byte, error) {
	type Alias OverrideDecision
	return json.Marshal(&struct {
		Decision      string `json:"decision"`
		OverrideLevel string `json:"override_level"`
		*Alias
	}{
		Decision:      string(od.Decision),
		OverrideLevel: od.OverrideLevel.String(),
		Alias:         (*Alias)(od),
	})
}