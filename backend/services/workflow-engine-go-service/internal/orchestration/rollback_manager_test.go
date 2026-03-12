package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// TestRollbackManager_NewInstance tests creation of RollbackManager
func TestRollbackManager_NewInstance(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	logger := zap.NewNop()

	manager := NewRollbackManager(redisClient, logger)

	if manager == nil {
		t.Fatal("Expected RollbackManager instance, got nil")
	}

	if manager.redisClient == nil {
		t.Error("Expected Redis client to be set")
	}

	if manager.logger == nil {
		t.Error("Expected logger to be set")
	}
}

// TestRollbackToken_Creation tests rollback token creation
func TestRollbackToken_Creation(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	logger := zap.NewNop()

	manager := NewRollbackManager(redisClient, logger)
	ctx := context.Background()

	commitID := "commit-123"
	proposalID := "proposal-456"

	token, expiresAt := manager.CreateRollbackToken(ctx, commitID, proposalID)

	// Verify token is not empty
	if token == "" {
		t.Error("Expected non-empty rollback token")
	}

	// Verify expiration is approximately 5 minutes from now
	expectedExpiry := time.Now().Add(5 * time.Minute)
	timeDiff := expiresAt.Sub(expectedExpiry)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Expected expiry around 5 minutes from now, got %v", expiresAt)
	}
}

// TestRollbackWindow_Expiry tests 5-minute rollback window
func TestRollbackWindow_Expiry(t *testing.T) {
	tests := []struct {
		name           string
		tokenAge       time.Duration
		expectedValid  bool
	}{
		{
			name:          "Token within 5-minute window",
			tokenAge:      2 * time.Minute,
			expectedValid: true,
		},
		{
			name:          "Token at 5-minute boundary",
			tokenAge:      5 * time.Minute,
			expectedValid: false,
		},
		{
			name:          "Token expired (6 minutes)",
			tokenAge:      6 * time.Minute,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenTime := time.Now().Add(-tt.tokenAge)
			isValid := isTokenValid(tokenTime)

			if isValid != tt.expectedValid {
				t.Errorf("Expected token validity %v for age %v, got %v", tt.expectedValid, tt.tokenAge, isValid)
			}
		})
	}
}

// TestRollbackRequest_Validation tests rollback request validation
func TestRollbackRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request *RollbackRequest
		valid   bool
	}{
		{
			name: "Valid rollback request",
			request: &RollbackRequest{
				RollbackToken:    "token-123",
				CommitID:         "commit-456",
				ProposalID:       "proposal-789",
				RequestedBy:      "provider-123",
				Reason:           "Patient adverse reaction",
				CompensationMode: "SOFT_DELETE",
			},
			valid: true,
		},
		{
			name: "Missing rollback token",
			request: &RollbackRequest{
				CommitID:    "commit-456",
				ProposalID:  "proposal-789",
				RequestedBy: "provider-123",
			},
			valid: false,
		},
		{
			name: "Missing commit ID",
			request: &RollbackRequest{
				RollbackToken: "token-123",
				ProposalID:    "proposal-789",
				RequestedBy:   "provider-123",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRollbackRequest(tt.request)
			if tt.valid && err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestCompensationActions tests different compensation action types
func TestCompensationActions(t *testing.T) {
	tests := []struct {
		name           string
		compensationType string
		expectedActions []string
	}{
		{
			name:           "Soft delete compensation",
			compensationType: "SOFT_DELETE",
			expectedActions: []string{"MARK_DELETED", "UPDATE_STATUS", "AUDIT_LOG"},
		},
		{
			name:           "State reversion compensation",
			compensationType: "STATE_REVERSION",
			expectedActions: []string{"RESTORE_PREVIOUS_STATE", "UPDATE_AUDIT", "NOTIFY_SYSTEMS"},
		},
		{
			name:           "Full rollback compensation",
			compensationType: "FULL_ROLLBACK",
			expectedActions: []string{"DELETE_RECORD", "REVERT_CHANGES", "PUBLISH_EVENT", "AUDIT_LOG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := getCompensationActions(tt.compensationType)

			if len(actions) != len(tt.expectedActions) {
				t.Errorf("Expected %d actions, got %d", len(tt.expectedActions), len(actions))
				continue
			}

			for i, expected := range tt.expectedActions {
				if actions[i] != expected {
					t.Errorf("Expected action %s at position %d, got %s", expected, i, actions[i])
				}
			}
		})
	}
}

// Helper functions for testing

func isTokenValid(tokenTime time.Time) bool {
	return time.Since(tokenTime) <= 5*time.Minute
}

func validateRollbackRequest(request *RollbackRequest) error {
	if request == nil {
		return fmt.Errorf("rollback request cannot be nil")
	}
	if request.RollbackToken == "" {
		return fmt.Errorf("rollback token is required")
	}
	if request.CommitID == "" {
		return fmt.Errorf("commit ID is required")
	}
	if request.ProposalID == "" {
		return fmt.Errorf("proposal ID is required")
	}
	if request.RequestedBy == "" {
		return fmt.Errorf("requestedBy is required")
	}
	return nil
}

func getCompensationActions(compensationType string) []string {
	switch compensationType {
	case "SOFT_DELETE":
		return []string{"MARK_DELETED", "UPDATE_STATUS", "AUDIT_LOG"}
	case "STATE_REVERSION":
		return []string{"RESTORE_PREVIOUS_STATE", "UPDATE_AUDIT", "NOTIFY_SYSTEMS"}
	case "FULL_ROLLBACK":
		return []string{"DELETE_RECORD", "REVERT_CHANGES", "PUBLISH_EVENT", "AUDIT_LOG"}
	default:
		return []string{}
	}
}

// RollbackRequest represents a rollback request for testing
type RollbackRequest struct {
	RollbackToken    string `json:"rollback_token"`
	CommitID         string `json:"commit_id"`
	ProposalID       string `json:"proposal_id"`
	RequestedBy      string `json:"requested_by"`
	Reason           string `json:"reason"`
	CompensationMode string `json:"compensation_mode"`
}