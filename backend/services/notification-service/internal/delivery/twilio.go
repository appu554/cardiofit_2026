package delivery

import (
	"context"
	"fmt"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioProvider implements SMS delivery via Twilio
type TwilioProvider struct {
	client     *twilio.RestClient
	fromNumber string
}

// NewTwilioProvider creates a new Twilio provider
func NewTwilioProvider(cfg config.SMSConfig) *TwilioProvider {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioSID,
		Password: cfg.TwilioToken,
	})

	return &TwilioProvider{
		client:     client,
		fromNumber: cfg.TwilioFromNumber,
	}
}

// Send sends an SMS via Twilio
func (p *TwilioProvider) Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error {
	for _, recipient := range recipients {
		params := &twilioApi.CreateMessageParams{}
		params.SetTo(recipient)
		params.SetFrom(p.fromNumber)
		params.SetBody(content)

		_, err := p.client.Api.CreateMessage(params)
		if err != nil {
			return fmt.Errorf("failed to send SMS to %s: %w", recipient, err)
		}
	}

	return nil
}
