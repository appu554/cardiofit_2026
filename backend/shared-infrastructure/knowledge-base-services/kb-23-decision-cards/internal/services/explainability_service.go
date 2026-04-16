package services

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/models"
)

// ExplainabilityEntry is one link in the evidence trail for a
// decision card. Each entry represents one reasoning step that
// contributed to the card's generation. Phase 10 Gap 10.
type ExplainabilityEntry struct {
	Step        int       `json:"step"`
	Source      string    `json:"source"`       // "KB-22_HPI", "MCU_GATE", "SAFETY_CHECK", "CONFOUNDER", "TEMPLATE"
	Summary     string    `json:"summary"`
	Detail      string    `json:"detail,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ExplainabilityTrail is the complete evidence trail for a single
// decision card, assembled from the card's stored fields. Phase 10
// Gap 10.
type ExplainabilityTrail struct {
	CardID               string                `json:"card_id"`
	PatientID            string                `json:"patient_id"`
	TemplateID           string                `json:"template_id"`
	GeneratedAt          time.Time             `json:"generated_at"`
	Entries              []ExplainabilityEntry `json:"entries"`
	OverallConfidence    string                `json:"overall_confidence"`
	ChainIntegrity       string                `json:"chain_integrity"` // "COMPLETE" or "PARTIAL" (some sources missing)
}

// ExplainabilityService builds evidence trails from stored decision
// cards. It reads the card's existing fields (MCUGateRationale,
// SafetyCheckSummary, ReasoningChain, ClinicianSummary) and
// assembles them into a chronologically-ordered trail that answers
// "why did the system generate this card?" Phase 10 Gap 10.
type ExplainabilityService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewExplainabilityService constructs the service.
func NewExplainabilityService(db *gorm.DB, logger *zap.Logger) *ExplainabilityService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ExplainabilityService{db: db, logger: logger}
}

// BuildTrail assembles the complete evidence trail for a decision
// card identified by its card_id (UUID string). Returns nil if the
// card doesn't exist. Phase 10 Gap 10.
func (s *ExplainabilityService) BuildTrail(cardID string) (*ExplainabilityTrail, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var card models.DecisionCard
	if err := s.db.Where("card_id = ?", cardID).First(&card).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	trail := &ExplainabilityTrail{
		CardID:      card.CardID.String(),
		PatientID:   card.PatientID.String(),
		TemplateID:  card.TemplateID,
		GeneratedAt: card.CreatedAt,
	}

	step := 1

	// Step 1: Template selection — why this card type?
	trail.Entries = append(trail.Entries, ExplainabilityEntry{
		Step:      step,
		Source:    "TEMPLATE",
		Summary:   "Card template selected: " + card.TemplateID,
		Detail:    "Differential ID: " + card.PrimaryDifferentialID + "; Node: " + card.NodeID,
		Timestamp: card.CreatedAt,
	})
	step++

	// Step 2: Confidence tier — how confident is the diagnosis?
	trail.Entries = append(trail.Entries, ExplainabilityEntry{
		Step:      step,
		Source:    "CONFIDENCE",
		Summary:   "Diagnostic confidence: " + string(card.DiagnosticConfidenceTier),
		Timestamp: card.CreatedAt,
	})
	trail.OverallConfidence = string(card.DiagnosticConfidenceTier)
	step++

	// Step 3: MCU gate evaluation — what gate was assigned and why?
	if card.MCUGateRationale != "" {
		trail.Entries = append(trail.Entries, ExplainabilityEntry{
			Step:      step,
			Source:    "MCU_GATE",
			Summary:   "MCU gate: " + string(card.MCUGate),
			Detail:    card.MCUGateRationale,
			Timestamp: card.CreatedAt,
		})
		step++
	}

	// Step 4: Safety check summary — what safety evaluations ran?
	if len(card.SafetyCheckSummary) > 0 {
		trail.Entries = append(trail.Entries, ExplainabilityEntry{
			Step:      step,
			Source:    "SAFETY_CHECK",
			Summary:   "Safety evaluation completed",
			Detail:    string(card.SafetyCheckSummary),
			Timestamp: card.CreatedAt,
		})
		step++
	}

	// Step 5: Reasoning chain from KB-22 (if populated)
	if len(card.ReasoningChain) > 0 {
		trail.Entries = append(trail.Entries, ExplainabilityEntry{
			Step:      step,
			Source:    "KB22_HPI",
			Summary:   "Bayesian reasoning chain from KB-22 HPI engine",
			Detail:    string(card.ReasoningChain),
			Timestamp: card.CreatedAt,
		})
		step++
	}

	// Step 6: Patient state snapshot (if populated)
	if len(card.PatientStateSnapshot) > 0 {
		var snapshot map[string]interface{}
		if json.Unmarshal(card.PatientStateSnapshot, &snapshot) == nil {
			snapshotStr, _ := json.Marshal(snapshot)
			trail.Entries = append(trail.Entries, ExplainabilityEntry{
				Step:      step,
				Source:    "PATIENT_STATE",
				Summary:   "Patient state at card generation",
				Detail:    string(snapshotStr),
				Timestamp: card.CreatedAt,
			})
			step++
		}
	}

	// Step 7: Clinician summary — the final output
	trail.Entries = append(trail.Entries, ExplainabilityEntry{
		Step:      step,
		Source:    "CLINICIAN_SUMMARY",
		Summary:   card.ClinicianSummary,
		Timestamp: card.CreatedAt,
	})

	// Determine chain integrity
	trail.ChainIntegrity = "PARTIAL"
	if len(card.ReasoningChain) > 0 && len(card.SafetyCheckSummary) > 0 {
		trail.ChainIntegrity = "COMPLETE"
	}

	return trail, nil
}
