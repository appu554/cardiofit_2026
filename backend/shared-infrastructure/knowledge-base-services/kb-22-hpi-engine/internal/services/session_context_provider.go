package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// SessionContextProvider fetches patient context from KB-20 (Patient Profile),
// KB-21 (Behavioral Intelligence), and KB-23 (Treatment Perturbations) using
// 4 parallel goroutines (F-03, R-03). KB-20 is required; KB-21 and KB-23 are
// optional with graceful degradation to defaults.
type SessionContextProvider struct {
	config  *config.Config
	log     *zap.Logger
	metrics *metrics.Collector
	client  *http.Client
}

// SessionContext holds the aggregated context fetched from upstream KBs,
// required to initialise an HPI session.
type SessionContext struct {
	StratumLabel        string
	CKDSubstage         *string
	ActiveModifiers     []ContextModifier
	SafetyOverrides     []models.SafetyTriggerDef
	AdherenceWeights        map[string]float64
	AdherenceGainFactor     float64
	ReliabilityModifier     float64
	TreatmentPerturbations []TreatmentPerturbation

	// G2: Patient demographics for sex-modifier evaluation.
	// Populated from KB-20 patient profile.
	PatientSex string // "Male", "Female", or "" if unknown
	PatientAge int    // age in years, 0 if unknown
}

// kb20StratumResponse is the expected response from KB-20 stratum endpoint.
type kb20StratumResponse struct {
	StratumLabel    string                  `json:"stratum_label"`
	CKDSubstage     *string                 `json:"ckd_substage,omitempty"`
	ActiveModifiers []ContextModifier       `json:"active_modifiers,omitempty"`
	SafetyOverrides []models.SafetyTriggerDef `json:"safety_overrides,omitempty"`
	PatientSex      string                  `json:"patient_sex,omitempty"` // G2: "Male" or "Female"
	PatientAge      int                     `json:"patient_age,omitempty"` // G2: age in years
}

// kb21AdherenceResponse is the expected response from KB-21 adherence endpoint.
type kb21AdherenceResponse struct {
	Weights map[string]float64 `json:"weights"`
}

// kb21ReliabilityResponse is the expected response from KB-21 reliability endpoint.
type kb21ReliabilityResponse struct {
	ReliabilityModifier float64 `json:"reliability_modifier"`
}

// TreatmentPerturbation describes a single treatment-induced perturbation that
// may shift prior probabilities or trigger safety overrides during an HPI session.
type TreatmentPerturbation struct {
	TreatmentID   string  `json:"treatment_id"`
	DrugClass     string  `json:"drug_class"`
	PerturbedNode string  `json:"perturbed_node"`
	Direction     string  `json:"direction"`     // "increase" | "decrease"
	Magnitude     float64 `json:"magnitude"`     // effect size [0,1]
	EvidenceLevel string  `json:"evidence_level"` // e.g. "A", "B", "C"
}

// kb23PerturbationResponse is the expected response from KB-23 treatment-perturbations endpoint.
type kb23PerturbationResponse struct {
	Perturbations []TreatmentPerturbation `json:"perturbations"`
}

// NewSessionContextProvider creates a new SessionContextProvider with a shared
// HTTP client configured for upstream timeouts.
func NewSessionContextProvider(cfg *config.Config, log *zap.Logger, m *metrics.Collector) *SessionContextProvider {
	// Use the maximum of KB-20/KB-21/KB-23 timeouts for the shared transport,
	// individual requests set their own context deadlines.
	maxTimeout := cfg.KB20Timeout()
	if cfg.KB21Timeout() > maxTimeout {
		maxTimeout = cfg.KB21Timeout()
	}
	if cfg.KB23Timeout() > maxTimeout {
		maxTimeout = cfg.KB23Timeout()
	}

	return &SessionContextProvider{
		config:  cfg,
		log:     log,
		metrics: m,
		client: &http.Client{
			Timeout: maxTimeout + 10*time.Millisecond, // small buffer over per-request deadline
		},
	}
}

// Fetch retrieves patient stratum, adherence weights, reliability modifier, and
// treatment perturbations from KB-20, KB-21, and KB-23 in parallel using errgroup.
//
// Goroutines:
//  1. KB-20: GET /api/v1/patient/{id}/stratum/{node_id}          (required, 40ms timeout)
//  2. KB-21: GET /api/v1/patient/{id}/adherence-weights           (optional, 40ms timeout)
//  3. KB-21: GET /api/v1/patient/{id}/answer-reliability          (optional, 40ms timeout)
//  4. KB-23: GET /api/v1/patient/{id}/treatment-perturbations     (optional, 40ms timeout)
//
// If KB-20 fails, the entire fetch returns an error (session cannot be initialised
// without stratum). If KB-21 or KB-23 calls fail, defaults are used (adherence=1.0
// for all drug classes, reliability=1.0, empty perturbations).
func (p *SessionContextProvider) Fetch(ctx context.Context, patientID uuid.UUID, nodeID string) (*SessionContext, error) {
	result := &SessionContext{
		AdherenceWeights:    make(map[string]float64),
		AdherenceGainFactor: 1.0, // default if KB-21 is unavailable
		ReliabilityModifier: 1.0, // default if KB-21 is unavailable
	}

	g, gCtx := errgroup.WithContext(ctx)

	// Goroutine 1: KB-20 stratum (required)
	g.Go(func() error {
		start := time.Now()
		defer func() {
			p.metrics.KB20FetchDuration.Observe(float64(time.Since(start).Milliseconds()))
		}()

		url := fmt.Sprintf("%s/api/v1/patient/%s/stratum/%s", p.config.KB20URL, patientID.String(), nodeID)

		reqCtx, cancel := context.WithTimeout(gCtx, p.config.KB20Timeout())
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("kb-20 request build failed: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			return fmt.Errorf("kb-20 stratum fetch failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("kb-20 returned status %d: %s", resp.StatusCode, string(body))
		}

		var stratumResp kb20StratumResponse
		if err := json.NewDecoder(resp.Body).Decode(&stratumResp); err != nil {
			return fmt.Errorf("kb-20 response decode failed: %w", err)
		}

		if stratumResp.StratumLabel == "" {
			return fmt.Errorf("kb-20 returned empty stratum_label for patient %s node %s", patientID, nodeID)
		}

		result.StratumLabel = stratumResp.StratumLabel
		result.CKDSubstage = stratumResp.CKDSubstage
		result.ActiveModifiers = stratumResp.ActiveModifiers
		result.SafetyOverrides = stratumResp.SafetyOverrides
		result.PatientSex = stratumResp.PatientSex
		result.PatientAge = stratumResp.PatientAge

		p.log.Debug("kb-20 stratum fetched",
			zap.String("patient_id", patientID.String()),
			zap.String("stratum", stratumResp.StratumLabel),
			zap.Duration("latency", time.Since(start)),
		)

		return nil
	})

	// Goroutine 2: KB-21 adherence weights (optional)
	g.Go(func() error {
		start := time.Now()
		defer func() {
			p.metrics.KB21FetchDuration.Observe(float64(time.Since(start).Milliseconds()))
		}()

		url := fmt.Sprintf("%s/api/v1/patient/%s/adherence-weights", p.config.KB21URL, patientID.String())

		reqCtx, cancel := context.WithTimeout(gCtx, p.config.KB21Timeout())
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			p.log.Warn("kb-21 adherence request build failed, using defaults",
				zap.Error(err),
			)
			return nil // optional, do not propagate
		}
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			p.log.Warn("kb-21 adherence fetch failed, using defaults",
				zap.String("patient_id", patientID.String()),
				zap.Error(err),
			)
			return nil // optional
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			p.log.Warn("kb-21 adherence returned non-200, using defaults",
				zap.String("patient_id", patientID.String()),
				zap.Int("status", resp.StatusCode),
			)
			return nil
		}

		var adherenceResp kb21AdherenceResponse
		if err := json.NewDecoder(resp.Body).Decode(&adherenceResp); err != nil {
			p.log.Warn("kb-21 adherence decode failed, using defaults",
				zap.Error(err),
			)
			return nil
		}

		result.AdherenceWeights = adherenceResp.Weights

		p.log.Debug("kb-21 adherence weights fetched",
			zap.String("patient_id", patientID.String()),
			zap.Int("weight_count", len(adherenceResp.Weights)),
			zap.Duration("latency", time.Since(start)),
		)

		return nil
	})

	// Goroutine 3: KB-21 answer reliability (optional)
	g.Go(func() error {
		start := time.Now()

		url := fmt.Sprintf("%s/api/v1/patient/%s/answer-reliability", p.config.KB21URL, patientID.String())

		reqCtx, cancel := context.WithTimeout(gCtx, p.config.KB21Timeout())
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			p.log.Warn("kb-21 reliability request build failed, using default 1.0",
				zap.Error(err),
			)
			return nil
		}
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			p.log.Warn("kb-21 reliability fetch failed, using default 1.0",
				zap.String("patient_id", patientID.String()),
				zap.Error(err),
			)
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			p.log.Warn("kb-21 reliability returned non-200, using default 1.0",
				zap.String("patient_id", patientID.String()),
				zap.Int("status", resp.StatusCode),
			)
			return nil
		}

		var reliabilityResp kb21ReliabilityResponse
		if err := json.NewDecoder(resp.Body).Decode(&reliabilityResp); err != nil {
			p.log.Warn("kb-21 reliability decode failed, using default 1.0",
				zap.Error(err),
			)
			return nil
		}

		// Clamp reliability to [0.1, 1.0] to prevent zeroing out all LR updates
		if reliabilityResp.ReliabilityModifier < 0.1 {
			reliabilityResp.ReliabilityModifier = 0.1
		}
		if reliabilityResp.ReliabilityModifier > 1.0 {
			reliabilityResp.ReliabilityModifier = 1.0
		}

		result.ReliabilityModifier = reliabilityResp.ReliabilityModifier

		p.log.Debug("kb-21 reliability fetched",
			zap.String("patient_id", patientID.String()),
			zap.Float64("reliability", reliabilityResp.ReliabilityModifier),
			zap.Duration("latency", time.Since(start)),
		)

		return nil
	})

	// Goroutine 4: KB-23 treatment perturbations (optional)
	g.Go(func() error {
		start := time.Now()
		defer func() {
			p.metrics.KB23FetchDuration.Observe(float64(time.Since(start).Milliseconds()))
		}()

		url := fmt.Sprintf("%s/api/v1/patient/%s/treatment-perturbations", p.config.KB23URL, patientID.String())

		reqCtx, cancel := context.WithTimeout(gCtx, p.config.KB23Timeout())
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			p.log.Warn("kb-23 perturbation request build failed, using empty perturbations",
				zap.Error(err),
			)
			return nil // optional, do not propagate
		}
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			p.log.Warn("kb-23 perturbation fetch failed, using empty perturbations",
				zap.String("patient_id", patientID.String()),
				zap.Error(err),
			)
			return nil // optional
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			p.log.Warn("kb-23 perturbation returned non-200, using empty perturbations",
				zap.String("patient_id", patientID.String()),
				zap.Int("status", resp.StatusCode),
			)
			return nil
		}

		var perturbResp kb23PerturbationResponse
		if err := json.NewDecoder(resp.Body).Decode(&perturbResp); err != nil {
			p.log.Warn("kb-23 perturbation decode failed, using empty perturbations",
				zap.Error(err),
			)
			return nil
		}

		result.TreatmentPerturbations = perturbResp.Perturbations

		p.log.Debug("kb-23 treatment perturbations fetched",
			zap.String("patient_id", patientID.String()),
			zap.Int("perturbation_count", len(perturbResp.Perturbations)),
			zap.Duration("latency", time.Since(start)),
		)

		return nil
	})

	// Wait for all goroutines; errgroup cancels remaining on first error
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("session context fetch failed: %w", err)
	}

	// Compute adherence gain factor from KB-21 adherence weights.
	// Maps overall adherence score to a tier-based gain:
	//   HIGH   (>0.8): 1.0 — full observation weight
	//   MEDIUM (0.5–0.8): 0.7
	//   LOW    (<0.5): 0.4
	// Default remains 1.0 when no adherence data is available.
	result.AdherenceGainFactor = computeAdherenceGain(result.AdherenceWeights)

	p.log.Info("session context fetch complete",
		zap.String("patient_id", patientID.String()),
		zap.String("node_id", nodeID),
		zap.String("stratum", result.StratumLabel),
		zap.Float64("reliability", result.ReliabilityModifier),
		zap.Float64("adherence_gain", result.AdherenceGainFactor),
		zap.Int("modifier_count", len(result.ActiveModifiers)),
		zap.Int("adherence_weight_count", len(result.AdherenceWeights)),
		zap.Int("perturbation_count", len(result.TreatmentPerturbations)),
	)

	return result, nil
}

// computeAdherenceGain derives a tier-based gain factor from the KB-21
// adherence weight map. The overall score is the arithmetic mean of all
// drug-class weights. Tier mapping:
//
//	score > 0.8  → 1.0 (HIGH adherence, full observation weight)
//	0.5 ≤ score ≤ 0.8 → 0.7 (MEDIUM adherence)
//	score < 0.5  → 0.4 (LOW adherence)
//
// Returns 1.0 when the weight map is empty (no data from KB-21).
func computeAdherenceGain(weights map[string]float64) float64 {
	if len(weights) == 0 {
		return 1.0
	}

	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	score := sum / float64(len(weights))

	switch {
	case score > 0.8:
		return 1.0
	case score >= 0.5:
		return 0.7
	default:
		return 0.4
	}
}
