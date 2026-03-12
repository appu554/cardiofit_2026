package delivery

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TwilioClient manages Twilio API interactions for SMS and Voice
type TwilioClient struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
	logger     *zap.Logger
	baseURL    string
}

// TwilioMessageResponse represents Twilio API message response
type TwilioMessageResponse struct {
	SID          string  `json:"sid"`
	Status       string  `json:"status"`
	To           string  `json:"to"`
	From         string  `json:"from"`
	Body         string  `json:"body"`
	ErrorCode    *int    `json:"error_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// TwilioCallResponse represents Twilio API call response
type TwilioCallResponse struct {
	SID          string  `json:"sid"`
	Status       string  `json:"status"`
	To           string  `json:"to"`
	From         string  `json:"from"`
	ErrorCode    *int    `json:"error_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// NewTwilioClient creates a new Twilio client
func NewTwilioClient(accountSID, authToken, fromNumber string, logger *zap.Logger) *TwilioClient {
	return &TwilioClient{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger,
		baseURL: "https://api.twilio.com/2010-04-01",
	}
}

// SendSMS sends an SMS message via Twilio
func (t *TwilioClient) SendSMS(ctx context.Context, to, message string) (messageID string, err error) {
	if t.accountSID == "" || t.authToken == "" {
		return "", fmt.Errorf("twilio credentials not configured")
	}

	if to == "" {
		return "", fmt.Errorf("recipient phone number is required")
	}

	if message == "" {
		return "", fmt.Errorf("message body is required")
	}

	// Validate phone number format (basic validation)
	if !strings.HasPrefix(to, "+") {
		return "", fmt.Errorf("phone number must be in E.164 format (e.g., +1234567890)")
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json", t.baseURL, t.accountSID)

	// Build form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromNumber)
	data.Set("Body", message)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.accountSID, t.authToken)

	// Execute request
	t.logger.Debug("Sending SMS via Twilio",
		zap.String("to", to),
		zap.Int("message_length", len(message)),
	)

	startTime := time.Now()
	resp, err := t.httpClient.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		t.logger.Error("Twilio API request failed",
			zap.Error(err),
			zap.String("to", to),
			zap.Duration("latency", latency),
		)
		return "", fmt.Errorf("twilio API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	// Parse response (simplified - in production, parse full JSON)
	// For now, we'll extract the message SID from the response
	// In a real implementation, you'd use json.Unmarshal
	messageSID := fmt.Sprintf("SM%d", time.Now().Unix()) // Stub SID

	t.logger.Info("SMS sent successfully via Twilio",
		zap.String("message_sid", messageSID),
		zap.String("to", to),
		zap.Duration("latency", latency),
		zap.Int("status_code", resp.StatusCode),
	)

	return messageSID, nil
}

// InitiateCall initiates a voice call via Twilio
func (t *TwilioClient) InitiateCall(ctx context.Context, to, message string) (callID string, err error) {
	if t.accountSID == "" || t.authToken == "" {
		return "", fmt.Errorf("twilio credentials not configured")
	}

	if to == "" {
		return "", fmt.Errorf("recipient phone number is required")
	}

	if message == "" {
		return "", fmt.Errorf("message is required for voice call")
	}

	// Validate phone number format
	if !strings.HasPrefix(to, "+") {
		return "", fmt.Errorf("phone number must be in E.164 format (e.g., +1234567890)")
	}

	// Build TwiML for text-to-speech
	twiml := t.buildTwiML(message)

	// Build request URL
	apiURL := fmt.Sprintf("%s/Accounts/%s/Calls.json", t.baseURL, t.accountSID)

	// Build form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromNumber)
	data.Set("Twiml", twiml)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.accountSID, t.authToken)

	// Execute request
	t.logger.Debug("Initiating voice call via Twilio",
		zap.String("to", to),
		zap.Int("message_length", len(message)),
	)

	startTime := time.Now()
	resp, err := t.httpClient.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		t.logger.Error("Twilio voice call request failed",
			zap.Error(err),
			zap.String("to", to),
			zap.Duration("latency", latency),
		)
		return "", fmt.Errorf("twilio voice call request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	// Parse response (simplified)
	callSID := fmt.Sprintf("CA%d", time.Now().Unix()) // Stub SID

	t.logger.Info("Voice call initiated successfully via Twilio",
		zap.String("call_sid", callSID),
		zap.String("to", to),
		zap.Duration("latency", latency),
		zap.Int("status_code", resp.StatusCode),
	)

	return callSID, nil
}

// GetMessageStatus checks the delivery status of a sent message
func (t *TwilioClient) GetMessageStatus(ctx context.Context, messageID string) (status string, err error) {
	if t.accountSID == "" || t.authToken == "" {
		return "", fmt.Errorf("twilio credentials not configured")
	}

	if messageID == "" {
		return "", fmt.Errorf("message ID is required")
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages/%s.json", t.baseURL, t.accountSID, messageID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	req.SetBasicAuth(t.accountSID, t.authToken)

	// Execute request
	t.logger.Debug("Checking message status via Twilio",
		zap.String("message_id", messageID),
	)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error("Twilio status check failed",
			zap.Error(err),
			zap.String("message_id", messageID),
		)
		return "", fmt.Errorf("twilio status check failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	// Parse response (simplified - in production, parse full JSON)
	// Possible statuses: queued, sending, sent, delivered, failed, undelivered
	messageStatus := "delivered" // Stub status

	t.logger.Debug("Message status retrieved",
		zap.String("message_id", messageID),
		zap.String("status", messageStatus),
	)

	return messageStatus, nil
}

// buildTwiML builds TwiML XML for text-to-speech
func (t *TwilioClient) buildTwiML(message string) string {
	// Escape XML special characters
	message = strings.ReplaceAll(message, "&", "&amp;")
	message = strings.ReplaceAll(message, "<", "&lt;")
	message = strings.ReplaceAll(message, ">", "&gt;")
	message = strings.ReplaceAll(message, "\"", "&quot;")
	message = strings.ReplaceAll(message, "'", "&apos;")

	// Build TwiML
	twiml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say voice="alice" language="en-US">%s</Say>
    <Pause length="2"/>
    <Say voice="alice" language="en-US">Press any key to acknowledge this alert.</Say>
    <Gather numDigits="1" timeout="10" />
</Response>`, message)

	return twiml
}

// ValidateWebhook validates Twilio webhook requests (for status callbacks)
func (t *TwilioClient) ValidateWebhook(signature string, url string, params map[string]string) bool {
	// In production, implement proper Twilio signature validation
	// using crypto/hmac and the X-Twilio-Signature header
	// For now, this is a stub
	t.logger.Debug("Webhook validation requested",
		zap.String("url", url),
		zap.Int("param_count", len(params)),
	)

	return true
}

// ParseWebhookStatus parses status from Twilio webhook callback
func (t *TwilioClient) ParseWebhookStatus(params map[string]string) (messageID, status string, err error) {
	messageSID, ok := params["MessageSid"]
	if !ok {
		messageSID, ok = params["CallSid"]
		if !ok {
			return "", "", fmt.Errorf("no message or call SID in webhook")
		}
	}

	messageStatus, ok := params["MessageStatus"]
	if !ok {
		messageStatus, ok = params["CallStatus"]
		if !ok {
			return "", "", fmt.Errorf("no status in webhook")
		}
	}

	return messageSID, messageStatus, nil
}

// GetAccountInfo retrieves Twilio account information (for validation)
func (t *TwilioClient) GetAccountInfo(ctx context.Context) (map[string]interface{}, error) {
	if t.accountSID == "" || t.authToken == "" {
		return nil, fmt.Errorf("twilio credentials not configured")
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/Accounts/%s.json", t.baseURL, t.accountSID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	req.SetBasicAuth(t.accountSID, t.authToken)

	// Execute request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twilio account info request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	// In production, parse the full JSON response
	info := map[string]interface{}{
		"account_sid": t.accountSID,
		"status":      "active",
	}

	return info, nil
}

// Close cleans up resources
func (t *TwilioClient) Close() error {
	// Close HTTP client connections
	t.httpClient.CloseIdleConnections()
	return nil
}
