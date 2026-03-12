package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/metrics"
)

// AdherenceData holds KB-21 adherence information per patient.
type AdherenceData struct {
	InsulinAdherence float64 `json:"insulin_adherence"`
	GainFactor       float64 `json:"gain_factor"`
	Source           string  `json:"source"`
}

// HTNAdherenceData holds KB-21 antihypertensive adherence for HTN card gating.
type HTNAdherenceData struct {
	PatientID              string                       `json:"patient_id"`
	AggregateScore         float64                      `json:"aggregate_score"`
	AggregateScore7d       float64                      `json:"aggregate_score_7d"`
	AggregateTrend         string                       `json:"aggregate_trend"`
	PrimaryReason          string                       `json:"primary_reason"`
	PerClassAdherence      map[string]HTNClassAdherence  `json:"per_class_adherence"`
	ActiveHTNDrugClasses   int                          `json:"active_htn_drug_classes"`
	DietarySodiumEstimate  string                       `json:"dietary_sodium_estimate,omitempty"`
	SaltReductionPotential float64                      `json:"salt_reduction_potential"`
	Source                 string                       `json:"source"`
}

// HTNClassAdherence mirrors KB-21's per-drug-class adherence for one HTN drug.
type HTNClassAdherence struct {
	DrugClass      string  `json:"drug_class"`
	Score30d       float64 `json:"score_30d"`
	Score7d        float64 `json:"score_7d"`
	Trend          string  `json:"trend"`
	DataQuality    string  `json:"data_quality"`
	IsFDC          bool    `json:"is_fdc"`
	PrimaryBarrier string  `json:"primary_barrier,omitempty"`
}

// HTNAdherenceGate classifies the card behaviour based on HTN adherence.
type HTNAdherenceGate string

const (
	HTNGateStandardEscalation   HTNAdherenceGate = "STANDARD_ESCALATION"
	HTNGateAdherenceLead        HTNAdherenceGate = "ADHERENCE_LEAD"
	HTNGateAdherenceIntervention HTNAdherenceGate = "ADHERENCE_INTERVENTION"
	HTNGateSideEffectHPI        HTNAdherenceGate = "SIDE_EFFECT_HPI"
)

type KB21Client struct {
	cfg     *config.Config
	cache   *cache.CacheClient
	metrics *metrics.Collector
	log     *zap.Logger
	client  *http.Client
}

func NewKB21Client(cfg *config.Config, c *cache.CacheClient, m *metrics.Collector, log *zap.Logger) *KB21Client {
	return &KB21Client{
		cfg:     cfg,
		cache:   c,
		metrics: m,
		log:     log,
		client: &http.Client{
			Timeout: cfg.KB21Timeout(),
		},
	}
}

// FetchAdherence retrieves adherence score from cache or KB-21 (A-04).
func (c *KB21Client) FetchAdherence(ctx context.Context, patientID string) (*AdherenceData, error) {
	// Check cache first
	var cached AdherenceData
	if err := c.cache.GetAdherence(patientID, &cached); err == nil {
		return &cached, nil
	}

	// Fetch from KB-21
	start := time.Now()
	url := fmt.Sprintf("%s/patient/%s/adherence", c.cfg.KB21URL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-21 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	c.metrics.KB21FetchLatency.Observe(float64(time.Since(start).Milliseconds()))

	if err != nil {
		c.log.Warn("KB-21 unreachable", zap.Error(err))
		return nil, fmt.Errorf("KB-21 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-21 returned status %d: %s", resp.StatusCode, string(body))
	}

	var data AdherenceData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode KB-21 response: %w", err)
	}

	// Cache with 6h TTL (A-04)
	if err := c.cache.SetAdherence(patientID, data); err != nil {
		c.log.Warn("adherence cache write failed", zap.Error(err))
	}

	return &data, nil
}

// FetchHTNAdherence retrieves antihypertensive adherence from KB-21 (Amendment 4).
// Used by card_builder to gate HYPERTENSION_REVIEW cards based on adherence level.
func (c *KB21Client) FetchHTNAdherence(ctx context.Context, patientID string) (*HTNAdherenceData, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/patient/%s/adherence/htn", c.cfg.KB21URL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-21 HTN adherence request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	c.metrics.KB21FetchLatency.Observe(float64(time.Since(start).Milliseconds()))

	if err != nil {
		c.log.Warn("KB-21 unreachable for HTN adherence", zap.Error(err))
		return nil, fmt.Errorf("KB-21 HTN adherence fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-21 HTN adherence returned status %d: %s", resp.StatusCode, string(body))
	}

	var data HTNAdherenceData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode KB-21 HTN adherence response: %w", err)
	}

	return &data, nil
}

// EvaluateHTNAdherenceGate determines the card behaviour based on
// aggregate antihypertensive adherence and barrier type.
//
// Decision matrix (Amendment 4):
//
//	Adherence >= 0.85           → STANDARD_ESCALATION
//	Adherence 0.60-0.84        → ADHERENCE_LEAD (lead with adherence finding)
//	Adherence < 0.60           → ADHERENCE_INTERVENTION (no dose card)
//	Any class SIDE_EFFECT       → SIDE_EFFECT_HPI (route to KB-22)
func (c *KB21Client) EvaluateHTNAdherenceGate(ctx context.Context, patientID string) (HTNAdherenceGate, *HTNAdherenceData, error) {
	data, err := c.FetchHTNAdherence(ctx, patientID)
	if err != nil {
		// On failure, default to standard escalation (fail-open for clinical safety)
		c.log.Warn("HTN adherence fetch failed, defaulting to STANDARD_ESCALATION",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		return HTNGateStandardEscalation, nil, err
	}

	// Side effect override: any HTN drug class with SIDE_EFFECTS barrier → route to HPI
	for _, cls := range data.PerClassAdherence {
		if cls.PrimaryBarrier == "SIDE_EFFECTS" {
			return HTNGateSideEffectHPI, data, nil
		}
	}

	switch {
	case data.AggregateScore >= 0.85:
		return HTNGateStandardEscalation, data, nil
	case data.AggregateScore >= 0.60:
		return HTNGateAdherenceLead, data, nil
	default:
		return HTNGateAdherenceIntervention, data, nil
	}
}
