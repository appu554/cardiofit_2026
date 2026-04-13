package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB21EngagementProfile is the subset of KB-21's engagement profile that
// KB-26 uses to derive BP-context engagement phenotype.
type KB21EngagementProfile struct {
	PatientID           string   `json:"patient_id"`
	Phenotype           string   `json:"phenotype"`
	EngagementComposite *float64 `json:"engagement_composite,omitempty"`
}

// KB21Client fetches engagement profile data from KB-21.
type KB21Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB21Client constructs a client.
func NewKB21Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB21Client {
	return &KB21Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// FetchEngagement retrieves a patient's engagement profile from KB-21.
func (c *KB21Client) FetchEngagement(ctx context.Context, patientID string) (*KB21EngagementProfile, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/engagement", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-21 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-21 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-21 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-21 returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile KB21EngagementProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode KB-21 response: %w", err)
	}
	return &profile, nil
}

// MapEngagementToBPPhenotype translates KB-21's BehavioralPhenotype enum
// into the BP-context engagement strings the classifier understands.
//
//	DORMANT, CHURNED           -> MEASUREMENT_AVOIDANT
//	SPORADIC + composite < 0.5 -> CRISIS_ONLY_MEASURER
//	anything else              -> "" (no flag, classifier treats as no bias)
//
// engagementComposite may be 0 if KB-21 has not computed it yet.
func MapEngagementToBPPhenotype(kb21Phenotype string, engagementComposite float64) string {
	switch kb21Phenotype {
	case "DORMANT", "CHURNED":
		return "MEASUREMENT_AVOIDANT"
	case "SPORADIC":
		if engagementComposite < 0.5 {
			return "CRISIS_ONLY_MEASURER"
		}
		return ""
	default:
		return ""
	}
}
