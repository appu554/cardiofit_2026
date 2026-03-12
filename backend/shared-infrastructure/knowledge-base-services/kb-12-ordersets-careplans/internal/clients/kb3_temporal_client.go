// Package clients provides HTTP clients for KB service integrations
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-12-ordersets-careplans/internal/config"
)

// KB3TemporalClient provides HTTP client for KB-3 Guidelines/Temporal service
type KB3TemporalClient struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// Protocol represents a clinical protocol from KB-3
type Protocol struct {
	ProtocolID      string           `json:"protocol_id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"` // acute, chronic, preventive
	GuidelineSource string           `json:"guideline_source"`
	Version         string           `json:"version"`
	Description     string           `json:"description"`
	Stages          []Stage          `json:"stages"`
	Constraints     []TimeConstraint `json:"constraints"`
	EntryConditions []Condition      `json:"entry_conditions,omitempty"`
	Active          bool             `json:"active"`
}

// Stage represents a stage within a protocol
type Stage struct {
	StageID     string   `json:"stage_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Order       int      `json:"order"`
	Actions     []Action `json:"actions"`
}

// Action represents an action within a stage
type Action struct {
	ActionID    string        `json:"action_id"`
	Name        string        `json:"name"`
	Type        string        `json:"type"` // assessment, lab, medication, procedure, notification
	Required    bool          `json:"required"`
	Deadline    time.Duration `json:"deadline,omitempty"`
	Description string        `json:"description,omitempty"`
}

// TimeConstraint represents a time-critical constraint
type TimeConstraint struct {
	ConstraintID   string        `json:"constraint_id"`
	Action         string        `json:"action"`
	Deadline       time.Duration `json:"deadline"`
	AlertThreshold time.Duration `json:"alert_threshold,omitempty"`
	Severity       string        `json:"severity"` // critical, major, minor
	Reference      string        `json:"reference"`
	Description    string        `json:"description,omitempty"`
}

// Condition represents a condition for protocol entry
type Condition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// ConstraintValidationRequest represents a request to validate temporal constraints
// NOTE: KB-3 requires deadline field for constraint validation
type ConstraintValidationRequest struct {
	ProtocolID    string        `json:"protocol_id"`
	ConstraintID  string        `json:"constraint_id"`
	ActionTime    time.Time     `json:"action_time"`
	ReferenceTime time.Time     `json:"reference_time"`
	Deadline      time.Duration `json:"deadline"`             // Required by KB-3
	GracePeriod   time.Duration `json:"grace_period,omitempty"`
}

// ConstraintValidationResponse represents the result of constraint validation
type ConstraintValidationResponse struct {
	Valid            bool          `json:"valid"`
	Status           string        `json:"status"` // on_time, warning, overdue, critical
	TimeRemaining    time.Duration `json:"time_remaining"`
	TimeElapsed      time.Duration `json:"time_elapsed"`
	Deadline         time.Duration `json:"deadline"`
	AlertThreshold   time.Duration `json:"alert_threshold"`
	PercentComplete  float64       `json:"percent_complete"`
	Message          string        `json:"message"`
	RecommendedAction string       `json:"recommended_action,omitempty"`
}

// RecurrencePattern represents a recurrence schedule from KB-3
type RecurrencePattern struct {
	PatternID   string         `json:"pattern_id"`
	Type        string         `json:"type"` // daily, weekly, monthly, custom
	Frequency   int            `json:"frequency"`
	Interval    string         `json:"interval"` // day, week, month
	DaysOfWeek  []string       `json:"days_of_week,omitempty"`
	DayOfMonth  int            `json:"day_of_month,omitempty"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     time.Time      `json:"end_date,omitempty"`
	Occurrences int            `json:"occurrences,omitempty"`
	Exceptions  []time.Time    `json:"exceptions,omitempty"`
}

// ScheduleRequest represents a request to generate a schedule
type ScheduleRequest struct {
	Pattern       RecurrencePattern `json:"pattern"`
	StartDate     time.Time         `json:"start_date"`
	EndDate       time.Time         `json:"end_date"`
	MaxOccurrences int              `json:"max_occurrences,omitempty"`
}

// NextOccurrenceRequest represents KB-3's actual API format for next-occurrence
// NOTE: This matches the actual KB-3 /v1/temporal/next-occurrence endpoint
type NextOccurrenceRequest struct {
	FromTime   time.Time          `json:"from_time"`
	Recurrence RecurrenceConfig   `json:"recurrence"`
}

// RecurrenceConfig matches KB-3's recurrence format
type RecurrenceConfig struct {
	Frequency string `json:"frequency"` // DAILY, WEEKLY, MONTHLY
	Interval  int    `json:"interval"`
}

// NextOccurrenceResponse represents KB-3's response
type NextOccurrenceResponse struct {
	NextOccurrence time.Time `json:"next_occurrence"`
	FromTime       time.Time `json:"from_time"`
	Frequency      string    `json:"frequency"`
	Interval       int       `json:"interval"`
}

// ScheduleResponse represents generated schedule dates
type ScheduleResponse struct {
	Success     bool        `json:"success"`
	Dates       []time.Time `json:"dates"`
	Count       int         `json:"count"`
	NextDate    time.Time   `json:"next_date,omitempty"`
	PatternID   string      `json:"pattern_id"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// IntervalRelation represents Allen's Interval Algebra relations
type IntervalRelation struct {
	Relation     string    `json:"relation"` // before, after, meets, overlaps, during, starts, finishes, equals
	Interval1    Interval  `json:"interval_1"`
	Interval2    Interval  `json:"interval_2"`
	Valid        bool      `json:"valid"`
	Description  string    `json:"description"`
}

// Interval represents a time interval
type Interval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewKB3TemporalClient creates a new KB-3 Temporal HTTP client
func NewKB3TemporalClient(cfg config.KBClientConfig) *KB3TemporalClient {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   false,
	}

	return &KB3TemporalClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb3-temporal"),
	}
}

// IsEnabled returns whether the KB-3 client is enabled
func (c *KB3TemporalClient) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-3 service is healthy
func (c *KB3TemporalClient) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-3 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-3 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// GetProtocol retrieves a protocol by ID
func (c *KB3TemporalClient) GetProtocol(ctx context.Context, protocolID string) (*Protocol, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning empty protocol")
		return nil, nil
	}

	var resp *Protocol
	// KB-3 uses /v1/protocols/:type/:id format, try acute first since most are acute protocols
	endpoint := fmt.Sprintf("/v1/protocols/acute/%s", protocolID)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetProtocolsByType retrieves protocols by type (acute, chronic, preventive)
func (c *KB3TemporalClient) GetProtocolsByType(ctx context.Context, protocolType string) ([]Protocol, error) {
	if !c.config.Enabled {
		return []Protocol{}, nil
	}

	var resp struct {
		Protocols []Protocol `json:"protocols"`
	}
	// KB-3 uses /v1/protocols/:type endpoint (e.g., /v1/protocols/acute)
	endpoint := fmt.Sprintf("/v1/protocols/%s", protocolType)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Protocols, nil
}

// GetAcuteProtocols retrieves all acute protocols (Sepsis, Stroke, STEMI, etc.)
func (c *KB3TemporalClient) GetAcuteProtocols(ctx context.Context) ([]Protocol, error) {
	return c.GetProtocolsByType(ctx, "acute")
}

// ValidateConstraint validates a temporal constraint
func (c *KB3TemporalClient) ValidateConstraint(ctx context.Context, req *ConstraintValidationRequest) (*ConstraintValidationResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning default validation")
		return &ConstraintValidationResponse{
			Valid:  true,
			Status: "unknown",
			Message: "KB-3 temporal validation disabled",
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *ConstraintValidationResponse
	err = c.doRequest(ctx, "POST", "/v1/temporal/validate-constraint", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ValidateConstraintTiming is a convenience method for quick constraint validation
func (c *KB3TemporalClient) ValidateConstraintTiming(ctx context.Context, actionTime, referenceTime time.Time, deadline, gracePeriod time.Duration) (*ConstraintValidationResponse, error) {
	if !c.config.Enabled {
		elapsed := actionTime.Sub(referenceTime)
		valid := elapsed <= deadline
		status := "on_time"
		if elapsed > deadline {
			status = "overdue"
		} else if elapsed > deadline-gracePeriod {
			status = "warning"
		}
		return &ConstraintValidationResponse{
			Valid:           valid,
			Status:          status,
			TimeElapsed:     elapsed,
			TimeRemaining:   deadline - elapsed,
			Deadline:        deadline,
			PercentComplete: float64(elapsed) / float64(deadline) * 100,
		}, nil
	}

	req := &ConstraintValidationRequest{
		ActionTime:    actionTime,
		ReferenceTime: referenceTime,
		Deadline:      deadline,
		GracePeriod:   gracePeriod,
	}

	return c.ValidateConstraint(ctx, req)
}

// GenerateSchedule generates a schedule based on recurrence pattern
// NOTE: Converts internal ScheduleRequest to KB-3's NextOccurrenceRequest format
func (c *KB3TemporalClient) GenerateSchedule(ctx context.Context, req *ScheduleRequest) (*ScheduleResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, generating local schedule")
		return c.generateLocalSchedule(req), nil
	}

	// Convert pattern type to KB-3 frequency format
	frequency := "DAILY"
	switch req.Pattern.Type {
	case "daily":
		frequency = "DAILY"
	case "weekly":
		frequency = "WEEKLY"
	case "monthly":
		frequency = "MONTHLY"
	}

	// Convert to KB-3's actual API format
	kb3Req := &NextOccurrenceRequest{
		FromTime: req.StartDate,
		Recurrence: RecurrenceConfig{
			Frequency: frequency,
			Interval:  req.Pattern.Frequency,
		},
	}

	body, err := json.Marshal(kb3Req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var kb3Resp *NextOccurrenceResponse
	err = c.doRequest(ctx, "POST", "/v1/temporal/next-occurrence", body, &kb3Resp)
	if err != nil {
		return nil, err
	}

	// Convert KB-3 response to our ScheduleResponse format
	return &ScheduleResponse{
		Success:   true,
		Dates:     []time.Time{kb3Resp.NextOccurrence},
		Count:     1,
		NextDate:  kb3Resp.NextOccurrence,
		PatternID: req.Pattern.PatternID,
	}, nil
}

// generateLocalSchedule generates a basic schedule locally when KB-3 is disabled
func (c *KB3TemporalClient) generateLocalSchedule(req *ScheduleRequest) *ScheduleResponse {
	var dates []time.Time
	current := req.StartDate

	for current.Before(req.EndDate) && (req.MaxOccurrences == 0 || len(dates) < req.MaxOccurrences) {
		dates = append(dates, current)

		switch req.Pattern.Interval {
		case "day":
			current = current.AddDate(0, 0, req.Pattern.Frequency)
		case "week":
			current = current.AddDate(0, 0, req.Pattern.Frequency*7)
		case "month":
			current = current.AddDate(0, req.Pattern.Frequency, 0)
		default:
			current = current.AddDate(0, 0, 1)
		}
	}

	var nextDate time.Time
	if len(dates) > 0 {
		nextDate = dates[0]
		for _, d := range dates {
			if d.After(time.Now()) {
				nextDate = d
				break
			}
		}
	}

	return &ScheduleResponse{
		Success:   true,
		Dates:     dates,
		Count:     len(dates),
		NextDate:  nextDate,
		PatternID: req.Pattern.PatternID,
	}
}

// ValidateIntervalRelation validates Allen's Interval Algebra relations
func (c *KB3TemporalClient) ValidateIntervalRelation(ctx context.Context, relation string, interval1, interval2 Interval) (*IntervalRelation, error) {
	if !c.config.Enabled {
		return c.validateLocalInterval(relation, interval1, interval2), nil
	}

	req := map[string]interface{}{
		"relation":   relation,
		"interval_1": interval1,
		"interval_2": interval2,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *IntervalRelation
	err = c.doRequest(ctx, "POST", "/v1/temporal/evaluate", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateLocalInterval performs local interval validation
func (c *KB3TemporalClient) validateLocalInterval(relation string, i1, i2 Interval) *IntervalRelation {
	var valid bool
	var description string

	switch relation {
	case "before":
		valid = i1.End.Before(i2.Start)
		description = "Interval 1 ends before Interval 2 starts"
	case "after":
		valid = i1.Start.After(i2.End)
		description = "Interval 1 starts after Interval 2 ends"
	case "meets":
		valid = i1.End.Equal(i2.Start)
		description = "Interval 1 end equals Interval 2 start"
	case "overlaps":
		valid = i1.Start.Before(i2.Start) && i1.End.After(i2.Start) && i1.End.Before(i2.End)
		description = "Intervals overlap with I1 starting first"
	case "during":
		valid = i1.Start.After(i2.Start) && i1.End.Before(i2.End)
		description = "Interval 1 occurs during Interval 2"
	case "starts":
		valid = i1.Start.Equal(i2.Start) && i1.End.Before(i2.End)
		description = "Intervals start at same time"
	case "finishes":
		valid = i1.Start.After(i2.Start) && i1.End.Equal(i2.End)
		description = "Intervals end at same time"
	case "equals":
		valid = i1.Start.Equal(i2.Start) && i1.End.Equal(i2.End)
		description = "Intervals are equal"
	default:
		description = "Unknown relation"
	}

	return &IntervalRelation{
		Relation:    relation,
		Interval1:   i1,
		Interval2:   i2,
		Valid:       valid,
		Description: description,
	}
}

// GetTimeConstraints retrieves time constraints for a protocol
func (c *KB3TemporalClient) GetTimeConstraints(ctx context.Context, protocolID string) ([]TimeConstraint, error) {
	protocol, err := c.GetProtocol(ctx, protocolID)
	if err != nil {
		return nil, err
	}
	if protocol == nil {
		return []TimeConstraint{}, nil
	}
	return protocol.Constraints, nil
}

// doRequest performs an HTTP request with retry logic
func (c *KB3TemporalClient) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
			if waitTime > c.config.RetryWaitMax {
				waitTime = c.config.RetryWaitMax
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		var req *http.Request
		var err error

		if body != nil {
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewReader(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, nil)
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Client-Service", "kb-12-ordersets")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.log.WithError(err).WithField("attempt", attempt+1).Warn("KB-3 request failed, retrying")
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("KB-3 server error: %d - %s", resp.StatusCode, string(respBody))
			c.log.WithField("status", resp.StatusCode).WithField("attempt", attempt+1).Warn("KB-3 server error, retrying")
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("KB-3 client error: %d - %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("KB-3 request failed after %d retries: %w", c.config.MaxRetries+1, lastErr)
}

// RegisterTemporalEvent registers a temporal event for tracking time constraints
func (c *KB3TemporalClient) RegisterTemporalEvent(ctx context.Context, eventData []byte) error {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, skipping temporal event registration")
		return nil
	}

	var result map[string]interface{}
	// KB-3 handles temporal events through the alerts processing endpoint
	return c.doRequest(ctx, "POST", "/v1/alerts/process", eventData, &result)
}
