package delivery

import (
	"context"
	"fmt"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.uber.org/zap"
)

// VoiceCallProvider interface for making voice calls
type VoiceCallProvider interface {
	MakeCall(ctx context.Context, phoneNumber string, message string, metadata map[string]interface{}) (string, error)
}

// TwilioVoiceProvider implements voice call delivery via Twilio
type TwilioVoiceProvider struct {
	client       *twilio.RestClient
	fromNumber   string
	twimlBaseURL string // Base URL for TwiML callback
	logger       *zap.Logger
}

// NewTwilioVoiceProvider creates a new Twilio voice provider
func NewTwilioVoiceProvider(cfg config.SMSConfig, logger *zap.Logger) *TwilioVoiceProvider {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioSID,
		Password: cfg.TwilioToken,
	})

	return &TwilioVoiceProvider{
		client:       client,
		fromNumber:   cfg.TwilioFromNumber,
		twimlBaseURL: "https://api.cardiofit.health/twiml", // Would be configurable
		logger:       logger,
	}
}

// MakeCall initiates a voice call with Twilio
func (p *TwilioVoiceProvider) MakeCall(ctx context.Context, phoneNumber string, message string, metadata map[string]interface{}) (string, error) {
	// Generate TwiML URL for the message
	// In production, this would point to an endpoint that generates TwiML XML
	// For now, we use Twilio's text-to-speech directly
	twimlURL := p.generateTwiMLURL(message, metadata)

	params := &twilioApi.CreateCallParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(p.fromNumber)
	params.SetUrl(twimlURL)

	// Optional: Set machine detection to avoid leaving voicemails
	// params.SetMachineDetection("Enable")

	// Optional: Set timeout and status callback
	// params.SetTimeout(30)
	// params.SetStatusCallback(p.twimlBaseURL + "/status")

	call, err := p.client.Api.CreateCall(params)
	if err != nil {
		p.logger.Error("Failed to initiate voice call",
			zap.String("phone_number", phoneNumber),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to initiate voice call: %w", err)
	}

	p.logger.Info("Voice call initiated",
		zap.String("phone_number", phoneNumber),
		zap.String("call_sid", *call.Sid),
	)

	return *call.Sid, nil
}

// generateTwiMLURL generates a TwiML URL for text-to-speech
// In production, this would be a proper endpoint that generates TwiML XML
func (p *TwilioVoiceProvider) generateTwiMLURL(message string, metadata map[string]interface{}) string {
	// For production use, this would point to your own TwiML endpoint:
	// return fmt.Sprintf("%s/escalation?alert_id=%s&message=%s",
	//     p.twimlBaseURL,
	//     url.QueryEscape(metadata["alert_id"].(string)),
	//     url.QueryEscape(message))

	// For testing/development, use Twilio's Echo TwiML demo
	// In production, replace with your own TwiML endpoint
	return fmt.Sprintf("http://demo.twilio.com/docs/voice.xml")
}

// MockVoiceProvider is a mock implementation for testing
type MockVoiceProvider struct {
	calls  []VoiceCall
	logger *zap.Logger
}

// VoiceCall represents a voice call record
type VoiceCall struct {
	PhoneNumber string
	Message     string
	Metadata    map[string]interface{}
	CallSID     string
}

// NewMockVoiceProvider creates a mock voice provider for testing
func NewMockVoiceProvider(logger *zap.Logger) *MockVoiceProvider {
	return &MockVoiceProvider{
		calls:  make([]VoiceCall, 0),
		logger: logger,
	}
}

// MakeCall records a voice call without actually making it
func (m *MockVoiceProvider) MakeCall(ctx context.Context, phoneNumber string, message string, metadata map[string]interface{}) (string, error) {
	callSID := fmt.Sprintf("CALL-%d", len(m.calls)+1)

	call := VoiceCall{
		PhoneNumber: phoneNumber,
		Message:     message,
		Metadata:    metadata,
		CallSID:     callSID,
	}

	m.calls = append(m.calls, call)

	m.logger.Info("Mock voice call recorded",
		zap.String("phone_number", phoneNumber),
		zap.String("call_sid", callSID),
		zap.String("message", message),
	)

	return callSID, nil
}

// GetCalls returns all recorded calls (for testing)
func (m *MockVoiceProvider) GetCalls() []VoiceCall {
	return m.calls
}

// Reset clears all recorded calls (for testing)
func (m *MockVoiceProvider) Reset() {
	m.calls = make([]VoiceCall, 0)
}
