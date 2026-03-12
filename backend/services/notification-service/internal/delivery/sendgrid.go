package delivery

import (
	"context"
	"fmt"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridProvider implements email delivery via SendGrid
type SendGridProvider struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(cfg config.EmailConfig) *SendGridProvider {
	return &SendGridProvider{
		client:    sendgrid.NewSendClient(cfg.SendGridAPIKey),
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
	}
}

// Send sends an email via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error {
	from := mail.NewEmail(p.fromName, p.fromEmail)
	subject := "Clinical Alert Notification"
	if title, ok := metadata["title"].(string); ok {
		subject = title
	}

	for _, recipient := range recipients {
		to := mail.NewEmail("", recipient)
		message := mail.NewSingleEmail(from, subject, to, content, content)

		response, err := p.client.Send(message)
		if err != nil {
			return fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}

		if response.StatusCode >= 400 {
			return fmt.Errorf("sendgrid returned error status %d", response.StatusCode)
		}
	}

	return nil
}
