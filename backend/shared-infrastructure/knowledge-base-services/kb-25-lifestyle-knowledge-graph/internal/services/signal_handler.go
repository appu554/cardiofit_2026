package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kb-25-lifestyle-knowledge-graph/internal/graph"

	"go.uber.org/zap"
)

// SignalHandler processes signals routed by the consumer.
type SignalHandler struct {
	chainSvc    *ChainTraversalService
	graphClient graph.GraphClient
	kb26BaseURL string
	httpClient  *http.Client
	logger      *zap.Logger
}

// NewSignalHandler creates a handler for KB-25 lifestyle signals.
func NewSignalHandler(chainSvc *ChainTraversalService, graphClient graph.GraphClient, kb26BaseURL string, logger *zap.Logger) *SignalHandler {
	return &SignalHandler{
		chainSvc:    chainSvc,
		graphClient: graphClient,
		kb26BaseURL: kb26BaseURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		logger:      logger,
	}
}

// HandleSignal is the entry point called by the consumer.
func (h *SignalHandler) HandleSignal(ctx context.Context, action RouteAction, patientID string, payload json.RawMessage) error {
	switch action {
	case RouteMatchFood:
		return h.handleMealLog(ctx, patientID, payload)
	case RouteMatchExercise:
		return h.handleActivity(ctx, patientID, payload)
	case RouteUpdateWeight:
		return h.handleWeight(ctx, patientID, payload)
	case RouteUpdateWaist:
		return h.handleWaist(ctx, patientID, payload)
	}
	return nil
}

// mealPayload represents a meal log signal.
type mealPayload struct {
	FoodItems    []string `json:"food_items"`
	CaloriesKcal float64  `json:"calories_kcal"`
	ProteinG     float64  `json:"protein_g"`
	CarbsG       float64  `json:"carbs_g"`
	FatG         float64  `json:"fat_g"`
}

// activityPayload represents an activity signal.
type activityPayload struct {
	ActivityType string  `json:"activity_type"`
	DurationMin  float64 `json:"duration_min"`
	Steps        int     `json:"steps,omitempty"`
}

// calibrationRequest is the POST body for KB-26 /calibrate.
type calibrationRequest struct {
	PatientID        string  `json:"patient_id"`
	InterventionCode string  `json:"intervention_code"`
	TargetVariable   string  `json:"target_variable"`
	PopulationEffect float64 `json:"population_effect"`
	ObservedEffect   float64 `json:"observed_effect"`
	ObservationSD    float64 `json:"observation_sd"`
}

// defaultObservationSD is the assumed standard deviation for Bayesian calibration
// observations. Used as a prior uncertainty estimate when posting attribution to KB-26.
const defaultObservationSD = 2.0

// clinicalTargets are the clinical variables we compute attribution for.
var clinicalTargets = []string{"HBA1C", "SBP", "LDL"}

func (h *SignalHandler) handleMealLog(ctx context.Context, patientID string, rawMsg json.RawMessage) error {
	// Unwrap the outer signal envelope to get the inner payload.
	var env signalEnvelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		return fmt.Errorf("unmarshal signal envelope: %w", err)
	}

	var meal mealPayload
	if err := json.Unmarshal(env.Payload, &meal); err != nil {
		return fmt.Errorf("unmarshal meal payload: %w", err)
	}

	h.logger.Info("Processing meal log signal",
		zap.String("patient_id", patientID),
		zap.Int("food_item_count", len(meal.FoodItems)),
	)

	for _, foodCode := range meal.FoodItems {
		// Verify the food item exists in Neo4j.
		records, err := h.graphClient.Run(ctx, graph.CypherGetFoodByCode, map[string]any{
			"code": foodCode,
		})
		if err != nil {
			h.logger.Warn("Failed to query food node",
				zap.String("food_code", foodCode),
				zap.Error(err),
			)
			continue
		}
		if len(records) == 0 {
			h.logger.Debug("Food code not found in graph, skipping",
				zap.String("food_code", foodCode),
			)
			continue
		}

		// Traverse causal chains for each clinical target.
		for _, target := range clinicalTargets {
			chains, err := h.chainSvc.GetChainsToTarget(ctx, target)
			if err != nil {
				h.logger.Warn("Chain traversal failed",
					zap.String("target", target),
					zap.Error(err),
				)
				continue
			}

			// Filter chains that originate from this food code.
			for _, chain := range chains {
				if chain.Source != foodCode {
					continue
				}

				// Compute attribution.
				attr := AttributeOutcome(patientID, target, chain.NetEffect.EffectSize)

				// POST calibration to KB-26.
				if err := h.postCalibration(ctx, calibrationRequest{
					PatientID:        patientID,
					InterventionCode: foodCode,
					TargetVariable:   target,
					PopulationEffect: chain.NetEffect.EffectSize,
					ObservedEffect:   chain.NetEffect.EffectSize * attr.LifestyleFrac,
					ObservationSD:    defaultObservationSD,
				}); err != nil {
					h.logger.Warn("Failed to post calibration to KB-26",
						zap.String("food_code", foodCode),
						zap.String("target", target),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

func (h *SignalHandler) handleActivity(ctx context.Context, patientID string, rawMsg json.RawMessage) error {
	// Unwrap the outer signal envelope to get the inner payload.
	var env signalEnvelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		return fmt.Errorf("unmarshal signal envelope: %w", err)
	}

	var activity activityPayload
	if err := json.Unmarshal(env.Payload, &activity); err != nil {
		return fmt.Errorf("unmarshal activity payload: %w", err)
	}

	h.logger.Info("Processing activity signal",
		zap.String("patient_id", patientID),
		zap.String("activity_type", activity.ActivityType),
		zap.Float64("duration_min", activity.DurationMin),
	)

	exerciseCode := activity.ActivityType

	// Verify the exercise exists in Neo4j.
	records, err := h.graphClient.Run(ctx, graph.CypherGetExerciseByCode, map[string]any{
		"code": exerciseCode,
	})
	if err != nil {
		return fmt.Errorf("failed to query exercise node: %w", err)
	}
	if len(records) == 0 {
		h.logger.Debug("Exercise code not found in graph, skipping",
			zap.String("exercise_code", exerciseCode),
		)
		return nil
	}

	// Traverse causal chains for each clinical target.
	for _, target := range clinicalTargets {
		chains, err := h.chainSvc.GetChainsToTarget(ctx, target)
		if err != nil {
			h.logger.Warn("Chain traversal failed",
				zap.String("target", target),
				zap.Error(err),
			)
			continue
		}

		// Filter chains that originate from this exercise code.
		for _, chain := range chains {
			if chain.Source != exerciseCode {
				continue
			}

			// Compute attribution.
			attr := AttributeOutcome(patientID, target, chain.NetEffect.EffectSize)

			// POST calibration to KB-26.
			if err := h.postCalibration(ctx, calibrationRequest{
				PatientID:        patientID,
				InterventionCode: exerciseCode,
				TargetVariable:   target,
				PopulationEffect: chain.NetEffect.EffectSize,
				ObservedEffect:   chain.NetEffect.EffectSize * attr.LifestyleFrac,
				ObservationSD:    defaultObservationSD,
			}); err != nil {
				h.logger.Warn("Failed to post calibration to KB-26",
					zap.String("exercise_code", exerciseCode),
					zap.String("target", target),
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

func (h *SignalHandler) handleWeight(_ context.Context, patientID string, rawMsg json.RawMessage) error {
	var env signalEnvelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		return fmt.Errorf("unmarshal signal envelope: %w", err)
	}

	var payload struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal weight payload: %w", err)
	}

	h.logger.Info("Weight signal received",
		zap.String("patient_id", patientID),
		zap.Float64("weight_value", payload.Value),
		zap.String("unit", payload.Unit),
	)

	return nil
}

func (h *SignalHandler) handleWaist(_ context.Context, patientID string, rawMsg json.RawMessage) error {
	var env signalEnvelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		return fmt.Errorf("unmarshal signal envelope: %w", err)
	}

	var payload struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal waist payload: %w", err)
	}

	h.logger.Info("Waist signal received",
		zap.String("patient_id", patientID),
		zap.Float64("waist_value", payload.Value),
		zap.String("unit", payload.Unit),
	)

	return nil
}

// postCalibration sends a calibration request to KB-26.
func (h *SignalHandler) postCalibration(ctx context.Context, req calibrationRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal calibration request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/calibrate", h.kb26BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create calibration request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("KB-26 calibration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-26 calibration returned status %d", resp.StatusCode)
	}

	h.logger.Debug("Calibration posted to KB-26",
		zap.String("patient_id", req.PatientID),
		zap.String("intervention", req.InterventionCode),
		zap.String("target", req.TargetVariable),
	)

	return nil
}
