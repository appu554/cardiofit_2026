package fhirclient

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// doWithRetry executes an HTTP request with exponential backoff on 429/5xx.
func doWithRetry(
	client *http.Client,
	method, url string,
	bodyFactory func() io.Reader,
	headers map[string]string,
	logger *zap.Logger,
) (*http.Response, error) {
	var lastErr error
	for attempt, delay := range retryDelays {
		var body io.Reader
		if bodyFactory != nil {
			body = bodyFactory()
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logger.Warn("FHIR request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("FHIR Store returned %d", resp.StatusCode)
			logger.Warn("FHIR Store returned retryable status",
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt+1),
			)
			time.Sleep(delay)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("all %d retries exhausted: %w", len(retryDelays), lastErr)
}
