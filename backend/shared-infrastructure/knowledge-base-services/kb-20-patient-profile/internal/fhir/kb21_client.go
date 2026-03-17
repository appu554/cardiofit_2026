package fhir

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
)

// FestivalStatus represents the festival state returned by KB-21's /festival-status endpoint.
type FestivalStatus struct {
	Active      bool   `json:"active"`
	Name        string `json:"name,omitempty"`
	FastingType string `json:"fasting_type,omitempty"`
	Start       string `json:"start,omitempty"`
	End         string `json:"end,omitempty"`
	CoreStart   string `json:"core_start,omitempty"`
	CoreEnd     string `json:"core_end,omitempty"`
}

// KB21Client queries the KB-21 Behavioral Intelligence Service for festival calendar data.
// Used by KB-20's perturbation system (P4: festival fasting) to populate
// PerturbationEvalInput.FestivalActive/FestivalEndDate/FastingType.
//
// Cached in-memory (TTL 1h) because festival status changes at most daily.
type KB21Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger

	cache    map[string]*festivalCacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

type festivalCacheEntry struct {
	status    *FestivalStatus
	fetchedAt time.Time
}

// NewKB21Client creates a client for KB-21 festival status lookups.
func NewKB21Client(cfg config.KB21Config, logger *zap.Logger) *KB21Client {
	return &KB21Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:   logger,
		cache:    make(map[string]*festivalCacheEntry),
		cacheTTL: 1 * time.Hour,
	}
}

// GetFestivalStatus queries KB-21 for the active festival in the given region.
// Returns nil (no error) if KB-21 is unreachable — perturbation system treats
// nil as "no festival active" (graceful degradation).
func (c *KB21Client) GetFestivalStatus(region string) *FestivalStatus {
	if region == "" {
		region = "ALL"
	}

	// Check cache
	c.cacheMu.RLock()
	if entry, ok := c.cache[region]; ok {
		if time.Since(entry.fetchedAt) < c.cacheTTL {
			c.cacheMu.RUnlock()
			return entry.status
		}
	}
	c.cacheMu.RUnlock()

	reqURL := fmt.Sprintf("%s/festival-status?region=%s", c.baseURL, url.QueryEscape(region))

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		c.logger.Warn("KB-21 festival status request failed — P4 data unavailable",
			zap.String("region", region),
			zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("KB-21 festival status returned non-200",
			zap.String("region", region),
			zap.Int("status", resp.StatusCode))
		return nil
	}

	var status FestivalStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		c.logger.Warn("KB-21 festival status decode failed",
			zap.String("region", region),
			zap.Error(err))
		return nil
	}

	// Cache result
	c.cacheMu.Lock()
	c.cache[region] = &festivalCacheEntry{
		status:    &status,
		fetchedAt: time.Now(),
	}
	c.cacheMu.Unlock()

	if status.Active {
		c.logger.Debug("KB-21 active festival detected",
			zap.String("region", region),
			zap.String("name", status.Name),
			zap.String("fasting_type", status.FastingType))
	}

	return &status
}

// MapFestivalToPerturbationFastingType converts KB-21's FestivalType
// to KB-20's perturbation fasting type used by evalFestivalFasting().
//
// Mapping rationale:
//   - FASTING → COMPLETE_FAST (full glucose suppression, hypo risk)
//   - MIXED → FRUIT_ONLY (intermediate — fasting+feasting alternation)
//   - SWEET_HEAVY → ONE_MEAL (dampened — eating but high glycemic)
//   - MEAT_FEAST → DIETARY_RESTRICTION (minimal glucose impact → no suppression)
func MapFestivalToPerturbationFastingType(festivalType string) string {
	switch festivalType {
	case "FASTING":
		return "COMPLETE_FAST"
	case "MIXED":
		return "FRUIT_ONLY"
	case "SWEET_HEAVY":
		return "ONE_MEAL"
	case "MEAT_FEAST":
		return "DIETARY_RESTRICTION"
	default:
		return "DIETARY_RESTRICTION" // unknown → minimal suppression
	}
}

// HealthCheck verifies KB-21 is reachable.
func (c *KB21Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("KB-21 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-21 health check returned %d", resp.StatusCode)
	}
	return nil
}
