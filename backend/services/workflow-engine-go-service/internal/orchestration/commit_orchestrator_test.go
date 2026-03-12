package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// MockMedicationClient implements the MedicationServiceClient interface for testing
type MockMedicationClient struct {
	CommitFunc func(ctx context.Context, request interface{}) (interface{}, error)
}

func (m *MockMedicationClient) Commit(ctx context.Context, request interface{}) (interface{}, error) {
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockMedicationClient) HealthCheck(ctx context.Context) error {
	return nil
}

// TestCommitOrchestrator_NewInstance tests creation of CommitOrchestrator
func TestCommitOrchestrator_NewInstance(t *testing.T) {
	// Create mock dependencies
	mockClient := &MockMedicationClient{}
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	logger := zap.NewNop()

	// Create CommitOrchestrator instance
	orchestrator := NewCommitOrchestrator(mockClient, redisClient, logger)

	// Verify it was created successfully
	if orchestrator == nil {
		t.Fatal("Expected CommitOrchestrator instance, got nil")
	}

	if orchestrator.medicationClient == nil {
		t.Error("Expected medication client to be set")
	}

	if orchestrator.redisClient == nil {
		t.Error("Expected Redis client to be set")
	}

	if orchestrator.logger == nil {
		t.Error("Expected logger to be set")
	}
}

// TestCommitRequest_Validation tests commit request validation
func TestCommitRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request *CommitRequest
		valid   bool
	}{
		{
			name: "Valid commit request",
			request: &CommitRequest{
				ProposalID:    "proposal-123",
				PatientID:     "patient-456",
				WorkflowID:    "workflow-789",
				CorrelationID: "corr-123",
				ValidationResult: &CommitValidationResult{
					ValidationID: "val-123",
					Verdict:      "SAFE",
					RiskScore:    0.1,
				},
				SelectedProposal: &CommitProposal{
					ProposalID:     "proposal-123",
					MedicationCode: "MED001",
					MedicationName: "Test Medication",
					Dosage:         "10mg",
					Frequency:      "BID",
				},
				ProviderContext: &CommitProviderContext{
					ProviderID: "provider-123",
					Timestamp:  time.Now(),
				},
			},
			valid: true,
		},
		{
			name: "Missing proposal ID",
			request: &CommitRequest{
				PatientID:  "patient-456",
				WorkflowID: "workflow-789",
			},
			valid: false,
		},
		{
			name: "Missing validation result",
			request: &CommitRequest{
				ProposalID: "proposal-123",
				PatientID:  "patient-456",
				WorkflowID: "workflow-789",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommitRequest(tt.request)
			if tt.valid && err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestSafetyVerdictHandling tests different safety verdict handling
func TestSafetyVerdictHandling(t *testing.T) {
	tests := []struct {
		name           string
		verdict        string
		expectedAction string
	}{
		{
			name:           "SAFE verdict should trigger immediate commit",
			verdict:        "SAFE",
			expectedAction: "IMMEDIATE_COMMIT",
		},
		{
			name:           "UNSAFE verdict should trigger UI interaction",
			verdict:        "UNSAFE",
			expectedAction: "UI_INTERACTION",
		},
		{
			name:           "WARNING verdict should trigger UI interaction",
			verdict:        "WARNING",
			expectedAction: "UI_INTERACTION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := determineActionForVerdict(tt.verdict)
			if action != tt.expectedAction {
				t.Errorf("Expected action %s for verdict %s, got %s", tt.expectedAction, tt.verdict, action)
			}
		})
	}
}

// Helper functions for testing

func validateCommitRequest(request *CommitRequest) error {
	if request == nil {
		return fmt.Errorf("commit request cannot be nil")
	}
	if request.ProposalID == "" {
		return fmt.Errorf("proposal ID is required")
	}
	if request.PatientID == "" {
		return fmt.Errorf("patient ID is required")
	}
	if request.WorkflowID == "" {
		return fmt.Errorf("workflow ID is required")
	}
	if request.ValidationResult == nil {
		return fmt.Errorf("validation result is required")
	}
	return nil
}

func determineActionForVerdict(verdict string) string {
	switch verdict {
	case "SAFE":
		return "IMMEDIATE_COMMIT"
	case "UNSAFE", "WARNING":
		return "UI_INTERACTION"
	default:
		return "UNKNOWN"
	}
}