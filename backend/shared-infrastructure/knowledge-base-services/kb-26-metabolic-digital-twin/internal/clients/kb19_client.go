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

	"kb-26-metabolic-digital-twin/internal/metrics"
)

// KB19Event is the envelope KB-19 expects on POST /api/v1/events.
// Fields use omitempty so each event type only sends the fields it cares
// about. Mirrors KB-23's KB19Event struct so KB-19 sees one consistent shape.
type KB19Event struct {
	EventType    string    `json:"event_type"`
	PatientID    string    `json:"patient_id"`
	Timestamp    time.Time `json:"timestamp"`
	BPPhenotype  string    `json:"bp_phenotype,omitempty"`
	Urgency      string    `json:"urgency,omitempty"`
	OldPhenotype string    `json:"old_phenotype,omitempty"`
	NewPhenotype string    `json:"new_phenotype,omitempty"`
}

// KB19Client publishes events to KB-19 via HTTP POST.
type KB19Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
	metrics *metrics.Collector
}

// NewKB19Client constructs a client. The metrics collector is optional;
// pass nil in tests.
func NewKB19Client(baseURL string, timeout time.Duration, log *zap.Logger, metricsCollector *metrics.Collector) *KB19Client {
	return &KB19Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
		metrics: metricsCollector,
	}
}

// PublishMaskedHTNDetected announces a newly classified masked HTN patient.
func (c *KB19Client) PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error {
	return c.post(ctx, KB19Event{
		EventType:   "MASKED_HTN_DETECTED",
		PatientID:   patientID,
		Timestamp:   time.Now().UTC(),
		BPPhenotype: phenotype,
		Urgency:     urgency,
	})
}

// PublishPhenotypeChanged announces a transition from one phenotype to another.
func (c *KB19Client) PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error {
	return c.post(ctx, KB19Event{
		EventType:    "BP_PHENOTYPE_CHANGED",
		PatientID:    patientID,
		Timestamp:    time.Now().UTC(),
		OldPhenotype: oldPhenotype,
		NewPhenotype: newPhenotype,
	})
}

func (c *KB19Client) post(ctx context.Context, event KB19Event) error {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.KB19PublishLatency.Observe(time.Since(start).Seconds())
		}
	}()

	body, err := json.Marshal(event)
	if err != nil {
		if c.metrics != nil {
			c.metrics.KB19PublishErrors.Inc()
		}
		return fmt.Errorf("marshal KB-19 event: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		if c.metrics != nil {
			c.metrics.KB19PublishErrors.Inc()
		}
		return fmt.Errorf("build KB-19 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-19 publish failed", zap.String("url", url), zap.Error(err))
		if c.metrics != nil {
			c.metrics.KB19PublishErrors.Inc()
		}
		return fmt.Errorf("KB-19 POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		if c.metrics != nil {
			c.metrics.KB19PublishErrors.Inc()
		}
		return fmt.Errorf("KB-19 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Debug("KB-19 event published",
		zap.String("event_type", event.EventType),
		zap.String("patient_id", event.PatientID),
	)
	return nil
}
