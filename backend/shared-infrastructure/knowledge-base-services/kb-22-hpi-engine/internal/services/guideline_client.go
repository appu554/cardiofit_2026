package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/config"
)

// GuidelineClient integrates with KB-3 (Guidelines Repository) to fetch
// guideline-based prior adjustments (N-01) and management summaries for safety
// flags. All calls degrade gracefully: on error, nil is returned rather than
// propagating the failure, because KB-3 enrichment is optional.
type GuidelineClient struct {
	config *config.Config
	log    *zap.Logger
	client *http.Client
}

// GuidelineAdjustment holds the legacy response from KB-3 guideline query.
// Retained for backward compatibility with FetchAdjustments.
type GuidelineAdjustment struct {
	GuidelineRef  string             `json:"guideline_ref"`
	Adjustments   map[string]float64 `json:"adjustments"` // differential_id -> log-odds delta
	EvidenceLevel string             `json:"evidence_level,omitempty"`
}

// GuidelinePriorResponse contains log-odds adjustments to apply to the
// differential state vector, sourced from clinical guidelines (N-01).
type GuidelinePriorResponse struct {
	DifferentialAdjustments map[string]float64 `json:"differential_adjustments"`
	GuidelineRefs           []string           `json:"guideline_refs"`
}

// ManagementSummaryResponse contains a plain-language management summary
// for a fired safety flag, enriched by KB-3 guideline evidence.
type ManagementSummaryResponse struct {
	Summary       string   `json:"summary"`
	GuidelineRefs []string `json:"guideline_refs"`
}

// NewGuidelineClient creates a new GuidelineClient with an HTTP client
// configured for the KB-3 timeout.
func NewGuidelineClient(cfg *config.Config, log *zap.Logger) *GuidelineClient {
	return &GuidelineClient{
		config: cfg,
		log:    log,
		client: &http.Client{
			Timeout: cfg.KB3Timeout() + 10*time.Millisecond,
		},
	}
}

// GetPriorAdjustments fetches guideline-based prior adjustments from KB-3 for
// a given node and stratum combination (N-01). The adjustments are additive
// log-odds values that shift the prior probability of certain differentials
// based on guideline evidence.
//
// Returns nil on any error (graceful degradation). The session initialisation
// proceeds without guideline injection if KB-3 is unavailable.
func (c *GuidelineClient) GetPriorAdjustments(
	ctx context.Context,
	nodeID string,
	stratum string,
	ckdSubstage *string,
) (*GuidelinePriorResponse, error) {
	if c.config.KB3URL == "" {
		return nil, nil
	}

	// Build URL with query parameters
	baseURL := fmt.Sprintf("%s/api/v1/guidelines/prior-adjustments", c.config.KB3URL)

	params := url.Values{}
	params.Set("node_id", nodeID)
	params.Set("stratum", stratum)
	if ckdSubstage != nil && *ckdSubstage != "" {
		params.Set("ckd_substage", *ckdSubstage)
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	reqCtx, cancel := context.WithTimeout(ctx, c.config.KB3Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fullURL, nil)
	if err != nil {
		c.log.Warn("kb-3 prior adjustments request build failed",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return nil, nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("kb-3 prior adjustments fetch failed, proceeding without guideline priors",
			zap.String("node_id", nodeID),
			zap.String("stratum", stratum),
			zap.Error(err),
		)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.log.Debug("kb-3 has no prior adjustments for this node/stratum",
			zap.String("node_id", nodeID),
			zap.String("stratum", stratum),
		)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("kb-3 prior adjustments returned non-200",
			zap.String("node_id", nodeID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 256)),
		)
		return nil, nil
	}

	var result GuidelinePriorResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.log.Warn("kb-3 prior adjustments decode failed",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return nil, nil
	}

	c.log.Info("kb-3 prior adjustments fetched",
		zap.String("node_id", nodeID),
		zap.String("stratum", stratum),
		zap.Int("adjustment_count", len(result.DifferentialAdjustments)),
		zap.Int("guideline_ref_count", len(result.GuidelineRefs)),
	)

	return &result, nil
}

// GetManagementSummary fetches a guideline-backed management summary for a
// specific safety flag from KB-3. This enriches the safety flag with
// evidence-based recommended actions for the KB-23 Decision Card.
//
// Returns nil on any error (graceful degradation). The safety flag is still
// surfaced with its YAML-defined action if KB-3 enrichment fails.
func (c *GuidelineClient) GetManagementSummary(
	ctx context.Context,
	flagID string,
	stratum string,
) (*ManagementSummaryResponse, error) {
	if c.config.KB3URL == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf("%s/api/v1/guidelines/management-summary", c.config.KB3URL)

	params := url.Values{}
	params.Set("flag_id", flagID)
	params.Set("stratum", stratum)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	reqCtx, cancel := context.WithTimeout(ctx, c.config.KB3Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fullURL, nil)
	if err != nil {
		c.log.Warn("kb-3 management summary request build failed",
			zap.String("flag_id", flagID),
			zap.Error(err),
		)
		return nil, nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("kb-3 management summary fetch failed",
			zap.String("flag_id", flagID),
			zap.String("stratum", stratum),
			zap.Error(err),
		)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.log.Debug("kb-3 has no management summary for this flag",
			zap.String("flag_id", flagID),
			zap.String("stratum", stratum),
		)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("kb-3 management summary returned non-200",
			zap.String("flag_id", flagID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 256)),
		)
		return nil, nil
	}

	var result ManagementSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.log.Warn("kb-3 management summary decode failed",
			zap.String("flag_id", flagID),
			zap.Error(err),
		)
		return nil, nil
	}

	c.log.Debug("kb-3 management summary fetched",
		zap.String("flag_id", flagID),
		zap.String("stratum", stratum),
		zap.Int("guideline_ref_count", len(result.GuidelineRefs)),
	)

	return &result, nil
}

// FetchAdjustments is the legacy entry point for KB-3 guideline prior injection.
// Retained for backward compatibility. New code should use GetPriorAdjustments.
//
// Returns nil adjustments (not an error) if:
//   - KB-3 URL is not configured
//   - guidelineSource is empty
//   - KB-3 returns no adjustments
//   - KB-3 request fails (non-blocking enrichment)
func (c *GuidelineClient) FetchAdjustments(ctx context.Context, guidelineSource string, stratum string) (*GuidelineAdjustment, error) {
	if c.config.KB3URL == "" || guidelineSource == "" {
		return nil, nil
	}

	reqURL := fmt.Sprintf("%s/api/v1/guidelines/%s/prior-adjustments?stratum=%s",
		c.config.KB3URL, guidelineSource, stratum)

	reqCtx, cancel := context.WithTimeout(ctx, c.config.KB3Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		c.log.Warn("failed to create KB-3 request",
			zap.String("guideline_source", guidelineSource),
			zap.Error(err),
		)
		return nil, nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-3 request failed, proceeding without guideline adjustments",
			zap.String("guideline_source", guidelineSource),
			zap.Error(err),
		)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-3 returned non-OK status",
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 256)),
		)
		return nil, nil
	}

	var result GuidelineAdjustment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.log.Warn("failed to decode KB-3 response",
			zap.Error(err),
		)
		return nil, nil
	}

	c.log.Info("fetched guideline adjustments",
		zap.String("guideline_ref", result.GuidelineRef),
		zap.Int("adjustment_count", len(result.Adjustments)),
	)

	return &result, nil
}

