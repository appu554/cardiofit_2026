package unit

import (
	"context"
	"testing"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/cardiofit/notification-service/internal/routing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRoutingEngine_Route(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := config.RoutingConfig{
		DefaultChannel: "email",
		RetryAttempts:  3,
	}

	engine := routing.NewEngine(cfg, nil, logger)

	tests := []struct {
		name           string
		alert          *models.ClinicalAlert
		expectedChannel string
	}{
		{
			name: "Critical alert routes to SMS",
			alert: &models.ClinicalAlert{
				ID:         "alert-1",
				Priority:   "critical",
				Recipients: []string{"user1@example.com"},
			},
			expectedChannel: "sms",
		},
		{
			name: "High priority routes to push",
			alert: &models.ClinicalAlert{
				ID:         "alert-2",
				Priority:   "high",
				Recipients: []string{"user2@example.com"},
			},
			expectedChannel: "push",
		},
		{
			name: "Medium priority routes to email",
			alert: &models.ClinicalAlert{
				ID:         "alert-3",
				Priority:   "medium",
				Recipients: []string{"user3@example.com"},
			},
			expectedChannel: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Route(context.Background(), tt.alert)
			assert.NoError(t, err)
			assert.NotNil(t, decision)
			assert.Equal(t, tt.expectedChannel, decision.Channel)
			assert.Equal(t, tt.alert.ID, decision.AlertID)
			assert.Equal(t, tt.alert.Recipients, decision.Recipients)
		})
	}
}
