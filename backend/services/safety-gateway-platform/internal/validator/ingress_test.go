package validator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

func TestIngressValidator_ValidateRequest(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			MaxRequestSizeMB: 10,
		},
		Security: config.SecurityConfig{
			RateLimiting: config.RateLimitingConfig{
				Enabled:           false, // Disable for testing
				RequestsPerMinute: 1000,
				BurstSize:         100,
			},
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	validator, err := NewIngressValidator(cfg, testLogger)
	require.NoError(t, err)

	tests := []struct {
		name    string
		request *types.SafetyRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: &types.SafetyRequest{
				RequestID:     "550e8400-e29b-41d4-a716-446655440000",
				PatientID:     "550e8400-e29b-41d4-a716-446655440001",
				ClinicianID:   "550e8400-e29b-41d4-a716-446655440002",
				ActionType:    "medication_order",
				Priority:      "normal",
				MedicationIDs: []string{"550e8400-e29b-41d4-a716-446655440003"},
				Timestamp:     time.Now(),
				Source:        "test",
			},
			wantErr: false,
		},
		{
			name: "missing request ID",
			request: &types.SafetyRequest{
				PatientID:   "550e8400-e29b-41d4-a716-446655440001",
				ClinicianID: "550e8400-e29b-41d4-a716-446655440002",
				ActionType:  "medication_order",
			},
			wantErr: true,
			errMsg:  "request_id is required",
		},
		{
			name: "missing patient ID",
			request: &types.SafetyRequest{
				RequestID:   "550e8400-e29b-41d4-a716-446655440000",
				ClinicianID: "550e8400-e29b-41d4-a716-446655440002",
				ActionType:  "medication_order",
			},
			wantErr: true,
			errMsg:  "patient_id is required",
		},
		{
			name: "invalid action type",
			request: &types.SafetyRequest{
				RequestID:   "550e8400-e29b-41d4-a716-446655440000",
				PatientID:   "550e8400-e29b-41d4-a716-446655440001",
				ClinicianID: "550e8400-e29b-41d4-a716-446655440002",
				ActionType:  "invalid_action",
			},
			wantErr: true,
			errMsg:  "invalid action_type",
		},
		{
			name: "invalid UUID format",
			request: &types.SafetyRequest{
				RequestID:   "invalid-uuid",
				PatientID:   "550e8400-e29b-41d4-a716-446655440001",
				ClinicianID: "550e8400-e29b-41d4-a716-446655440002",
				ActionType:  "medication_order",
			},
			wantErr: true,
			errMsg:  "must be a valid UUID format",
		},
		{
			name: "malicious content",
			request: &types.SafetyRequest{
				RequestID:   "550e8400-e29b-41d4-a716-446655440000",
				PatientID:   "550e8400-e29b-41d4-a716-446655440001",
				ClinicianID: "550e8400-e29b-41d4-a716-446655440002",
				ActionType:  "medication_order",
				Source:      "<script>alert('xss')</script>",
			},
			wantErr: true,
			errMsg:  "potentially malicious content detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := validator.ValidateRequest(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIngressValidator_ValidateUUID(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			RateLimiting: config.RateLimitingConfig{
				Enabled: false,
			},
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	validator, err := NewIngressValidator(cfg, testLogger)
	require.NoError(t, err)

	tests := []struct {
		name      string
		uuid      string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "valid UUID",
			uuid:      "550e8400-e29b-41d4-a716-446655440000",
			fieldName: "test_field",
			wantErr:   false,
		},
		{
			name:      "empty UUID",
			uuid:      "",
			fieldName: "test_field",
			wantErr:   true,
		},
		{
			name:      "invalid length",
			uuid:      "550e8400-e29b-41d4-a716",
			fieldName: "test_field",
			wantErr:   true,
		},
		{
			name:      "missing hyphens",
			uuid:      "550e8400e29b41d4a716446655440000",
			fieldName: "test_field",
			wantErr:   true,
		},
		{
			name:      "wrong hyphen positions",
			uuid:      "550e8400-e29b41d4-a716-446655440000",
			fieldName: "test_field",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateUUID(tt.uuid, tt.fieldName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIngressValidator_SanitizeRequest(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			RateLimiting: config.RateLimitingConfig{
				Enabled: false,
			},
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	validator, err := NewIngressValidator(cfg, testLogger)
	require.NoError(t, err)

	request := &types.SafetyRequest{
		RequestID:     "  550e8400-e29b-41d4-a716-446655440000  ",
		PatientID:     "  550e8400-e29b-41d4-a716-446655440001  ",
		ClinicianID:   "  550e8400-e29b-41d4-a716-446655440002  ",
		ActionType:    "  medication_order  ",
		Priority:      "", // Should be set to default
		MedicationIDs: []string{"  med1  ", "", "  med2  "},
		Context: map[string]string{
			"  key1  ": "  value1  ",
			"key2":     "  value2  ",
		},
		Source: "", // Should be set to default
	}

	validator.sanitizeRequest(request)

	// Check trimmed fields
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", request.RequestID)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", request.PatientID)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440002", request.ClinicianID)
	assert.Equal(t, "medication_order", request.ActionType)

	// Check defaults
	assert.Equal(t, "normal", request.Priority)
	assert.Equal(t, "unknown", request.Source)

	// Check sanitized arrays (empty strings removed)
	assert.Equal(t, []string{"med1", "med2"}, request.MedicationIDs)

	// Check sanitized context
	assert.Equal(t, "value1", request.Context["  key1  "])
	assert.Equal(t, "value2", request.Context["key2"])
}

func BenchmarkIngressValidator_ValidateRequest(b *testing.B) {
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			MaxRequestSizeMB: 10,
		},
		Security: config.SecurityConfig{
			RateLimiting: config.RateLimitingConfig{
				Enabled: false,
			},
		},
	}

	testLogger, _ := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})

	validator, _ := NewIngressValidator(cfg, testLogger)

	request := &types.SafetyRequest{
		RequestID:     "550e8400-e29b-41d4-a716-446655440000",
		PatientID:     "550e8400-e29b-41d4-a716-446655440001",
		ClinicianID:   "550e8400-e29b-41d4-a716-446655440002",
		ActionType:    "medication_order",
		Priority:      "normal",
		MedicationIDs: []string{"550e8400-e29b-41d4-a716-446655440003"},
		Timestamp:     time.Now(),
		Source:        "test",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateRequest(ctx, request)
	}
}
