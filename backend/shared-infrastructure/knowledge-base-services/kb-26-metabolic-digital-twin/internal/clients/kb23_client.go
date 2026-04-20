package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB23Client is a minimal HTTP client for the subset of KB-23 endpoints
// KB-26 needs to call. Phase 4 P9 only uses the composite synthesis
// trigger — expand as needed.
type KB23Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB23Client constructs a client against the given KB-23 base URL.
func NewKB23Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB23Client {
	return &KB23Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// TriggerCompositeSynthesize asks KB-23 to fold the patient's active cards
// (created in the last 72 hours) into a single CompositeCardSignal. The
// call is best-effort: KB-23 returns 200 OK with {composite_created:false}
// when no active cards exist, and any transport error is surfaced so the
// caller can decide whether to log-and-continue or retry.
func (c *KB23Client) TriggerCompositeSynthesize(ctx context.Context, patientID string) error {
	if patientID == "" {
		return fmt.Errorf("patient_id required")
	}

	url := fmt.Sprintf("%s/api/v1/composite-cards/synthesize/%s", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("build KB-23 composite request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-23 composite trigger failed",
			zap.String("url", url), zap.Error(err))
		return fmt.Errorf("KB-23 POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KB-23 composite returned status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Debug("KB-23 composite synthesise triggered",
		zap.String("patient_id", patientID),
		zap.Int("status", resp.StatusCode))
	return nil
}

// ResolveLifecycle tells KB-23 that an outcome signal has been observed for
// a patient (Gap 19 T4). KB-23 attributes the outcome to the most-recent
// actioned-but-unresolved DetectionLifecycle and closes it.
//
// Best-effort: a 404 from KB-23 means "nothing to resolve" and is not an
// error from KB-26's perspective; a transport error is logged and returned
// so the caller can decide whether to continue or retry.
func (c *KB23Client) ResolveLifecycle(
	ctx context.Context,
	patientID, detectionType, outcomeDescription string,
	resolvedAt time.Time,
) error {
	if patientID == "" {
		return fmt.Errorf("patient_id required")
	}

	body := map[string]interface{}{
		"patient_id":          patientID,
		"resolved_at":         resolvedAt.UTC().Format(time.RFC3339Nano),
		"outcome_description": outcomeDescription,
	}
	if detectionType != "" {
		body["detection_type"] = detectionType
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal resolve body: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/tracking/resolve", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("build KB-23 resolve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-23 resolve-lifecycle failed", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("KB-23 POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // no matching lifecycle — not an error
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KB-23 resolve returned status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Debug("KB-23 lifecycle resolved",
		zap.String("patient_id", patientID),
		zap.String("detection_type", detectionType),
		zap.Int("status", resp.StatusCode))
	return nil
}
