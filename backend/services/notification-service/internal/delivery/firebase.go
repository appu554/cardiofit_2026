package delivery

import (
	"context"
	"fmt"

	"github.com/cardiofit/notification-service/internal/config"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FirebaseProvider implements push notification delivery via Firebase
type FirebaseProvider struct {
	client *messaging.Client
}

// NewFirebaseProvider creates a new Firebase provider
func NewFirebaseProvider(cfg config.PushConfig) *FirebaseProvider {
	opt := option.WithCredentialsFile(cfg.FirebaseCredentials)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil
	}

	return &FirebaseProvider{
		client: client,
	}
}

// Send sends a push notification via Firebase
func (p *FirebaseProvider) Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error {
	title := "Clinical Alert"
	if t, ok := metadata["title"].(string); ok {
		title = t
	}

	for _, token := range recipients {
		message := &messaging.Message{
			Notification: &messaging.Notification{
				Title: title,
				Body:  content,
			},
			Token: token,
		}

		_, err := p.client.Send(ctx, message)
		if err != nil {
			return fmt.Errorf("failed to send push notification to %s: %w", token, err)
		}
	}

	return nil
}
