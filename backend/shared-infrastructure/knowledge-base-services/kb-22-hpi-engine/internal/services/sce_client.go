package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// SCEClient calls the standalone KB-24 Safety Constraint Engine via HTTP.
// CC-1 architecture: SCE runs as a sidecar on the same host (~1-2ms latency).
// If the SCE is unavailable, the client falls back to the in-process SCEService.
type SCEClient struct {
	baseURL  string
	client   *http.Client
	fallback *SCEService
	log      *zap.Logger
}

// SCEEvaluateRequest is the request body for POST /api/v1/evaluate on KB-24.
type SCEEvaluateRequest struct {
	SessionID  uuid.UUID       `json:"session_id"`
	NodeID     string          `json:"node_id"`
	QuestionID string          `json:"question_id"`
	Answer     string          `json:"answer"`
	FiredCMs   map[string]bool `json:"fired_cms,omitempty"`
}

// SCEEvaluateResponse mirrors KB-24's evaluate response.
type SCEEvaluateResponse struct {
	Clear              bool               `json:"clear"`
	Flags              []models.SafetyFlag `json:"flags,omitempty"`
	EscalationRequired bool               `json:"escalation_required"`
	ReasonCode         string             `json:"reason_code,omitempty"`
}

// NewSCEClient creates an HTTP client for the standalone SCE service.
// The fallback SCEService is used when the HTTP call fails (circuit breaker pattern).
func NewSCEClient(baseURL string, timeout time.Duration, fallback *SCEService, log *zap.Logger) *SCEClient {
	return &SCEClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		fallback: fallback,
		log:      log.With(zap.String("component", "sce-client")),
	}
}

// Evaluate sends an answer to the SCE for safety evaluation.
// On HTTP failure, falls back to the in-process SCEService.
// Includes timing instrumentation (V-03) to validate ~1-2ms sidecar latency.
func (c *SCEClient) Evaluate(
	ctx context.Context,
	sessionID uuid.UUID,
	nodeID string,
	questionID string,
	answer string,
	firedCMs map[string]bool,
) (*SCEResult, error) {
	totalStart := time.Now()

	reqBody := SCEEvaluateRequest{
		SessionID:  sessionID,
		NodeID:     nodeID,
		QuestionID: questionID,
		Answer:     answer,
		FiredCMs:   firedCMs,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal SCE request: %w", err)
	}

	url := c.baseURL + "/api/v1/evaluate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create SCE request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpStart := time.Now()
	resp, err := c.client.Do(httpReq)
	httpDuration := time.Since(httpStart)

	if err != nil {
		c.log.Warn("CC-1: SCE HTTP call failed, falling back to in-process",
			zap.String("session_id", sessionID.String()),
			zap.Duration("http_attempt_ms", httpDuration),
			zap.Error(err),
		)
		fallbackStart := time.Now()
		result, fbErr := c.fallback.EvaluateAnswer(ctx, sessionID, nodeID, questionID, answer, firedCMs)
		c.log.Info("CC-1: SCE fallback completed",
			zap.String("session_id", sessionID.String()),
			zap.Duration("fallback_ms", time.Since(fallbackStart)),
			zap.Duration("total_ms", time.Since(totalStart)),
		)
		return result, fbErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Warn("CC-1: SCE returned non-200, falling back to in-process",
			zap.String("session_id", sessionID.String()),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("http_ms", httpDuration),
		)
		return c.fallback.EvaluateAnswer(ctx, sessionID, nodeID, questionID, answer, firedCMs)
	}

	var sceResp SCEEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&sceResp); err != nil {
		c.log.Warn("CC-1: SCE response decode failed, falling back to in-process",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return c.fallback.EvaluateAnswer(ctx, sessionID, nodeID, questionID, answer, firedCMs)
	}

	totalDuration := time.Since(totalStart)
	c.log.Info("CC-1: SCE evaluate completed",
		zap.String("session_id", sessionID.String()),
		zap.Duration("http_ms", httpDuration),
		zap.Duration("total_ms", totalDuration),
		zap.Bool("clear", sceResp.Clear),
		zap.Bool("escalation_required", sceResp.EscalationRequired),
	)

	return &SCEResult{
		Clear:              sceResp.Clear,
		Flags:              sceResp.Flags,
		EscalationRequired: sceResp.EscalationRequired,
		ReasonCode:         sceResp.ReasonCode,
	}, nil
}

// ClearSession notifies KB-24 to release session state.
// Fire-and-forget — failure is logged but not propagated.
func (c *SCEClient) ClearSession(ctx context.Context, sessionID uuid.UUID) {
	url := fmt.Sprintf("%s/api/v1/sessions/%s/clear", c.baseURL, sessionID.String())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		c.log.Warn("CC-1: failed to create SCE clear request", zap.Error(err))
		return
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.log.Warn("CC-1: SCE clear session failed (non-fatal)", zap.Error(err))
		c.fallback.ClearSession(sessionID)
		return
	}
	resp.Body.Close()
}
