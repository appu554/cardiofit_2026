package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kb-patient-profile/internal/models"
)

func TestValidateLab_Creatinine(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 0.19, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 0.2, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 0.25, models.ValidationFlagged},
		{"At flag low — ACCEPTED", 0.3, models.ValidationAccepted},
		{"Normal value", 1.0, models.ValidationAccepted},
		{"Normal high", 5.0, models.ValidationAccepted},
		{"At flag high — ACCEPTED", 10.0, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 15.0, models.ValidationFlagged},
		{"At max — FLAGGED (max > FlagHigh)", 20.0, models.ValidationFlagged},
		{"Above max — REJECTED", 20.1, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeCreatinine, tc.value)
			assert.Equal(t, tc.status, result.Status,
				"Creatinine %.2f: expected %s, got %s (%s)", tc.value, tc.status, result.Status, result.FlagReason)
		})
	}
}

func TestValidateLab_EGFR(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Negative — REJECTED", -1, models.ValidationRejected},
		{"Zero — FLAGGED (min < FlagLow)", 0, models.ValidationFlagged},
		{"Very low — FLAGGED", 3, models.ValidationFlagged},
		{"At flag low — ACCEPTED", 5, models.ValidationAccepted},
		{"Normal", 90, models.ValidationAccepted},
		{"At flag high — ACCEPTED", 150, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 180, models.ValidationFlagged},
		{"At max — FLAGGED (max > FlagHigh)", 200, models.ValidationFlagged},
		{"Above max — REJECTED", 201, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeEGFR, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_FBG(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 29, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 30, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 35, models.ValidationFlagged},
		{"At flag low — ACCEPTED", 40, models.ValidationAccepted},
		{"Normal fasting", 100, models.ValidationAccepted},
		{"Diabetic range", 250, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 550, models.ValidationFlagged},
		{"Above max — REJECTED", 601, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeFBG, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_HbA1c(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 2.9, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 3.0, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 3.2, models.ValidationFlagged},
		{"Normal", 5.6, models.ValidationAccepted},
		{"Diabetic", 8.5, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 16.0, models.ValidationFlagged},
		{"Above max — REJECTED", 18.1, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeHbA1c, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_SBP(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 59, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 60, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 65, models.ValidationFlagged},
		{"Normal", 120, models.ValidationAccepted},
		{"Hypertensive", 160, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 260, models.ValidationFlagged},
		{"Above max — REJECTED", 281, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeSBP, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_DBP(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 29, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 30, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 35, models.ValidationFlagged},
		{"At flag low — ACCEPTED", 40, models.ValidationAccepted},
		{"Normal", 80, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 160, models.ValidationFlagged},
		{"Above max — REJECTED", 181, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeDBP, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_Potassium(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 1.4, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 1.5, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 2.0, models.ValidationFlagged},
		{"Normal", 4.5, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 7.0, models.ValidationFlagged},
		{"Above max — REJECTED", 9.1, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypePotassium, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_TotalCholesterol(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name   string
		value  float64
		status string
	}{
		{"Below min — REJECTED", 49, models.ValidationRejected},
		{"At min — FLAGGED (min < FlagLow)", 50, models.ValidationFlagged},
		{"Below flag low — FLAGGED", 60, models.ValidationFlagged},
		{"Normal", 200, models.ValidationAccepted},
		{"Above flag high — FLAGGED", 450, models.ValidationFlagged},
		{"Above max — REJECTED", 601, models.ValidationRejected},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(models.LabTypeTotalCholesterol, tc.value)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestValidateLab_UnknownType(t *testing.T) {
	v := NewLabValidator()

	result := v.Validate("UNKNOWN_LAB_TYPE", 999)
	assert.Equal(t, models.ValidationAccepted, result.Status,
		"Unknown lab types should be accepted without validation")
}

func TestValidateBPPair(t *testing.T) {
	v := NewLabValidator()

	tests := []struct {
		name       string
		sbp        float64
		dbp        float64
		wantReject bool
	}{
		{"Normal BP — SBP > DBP", 120, 80, false},
		{"Wide pulse pressure", 180, 60, false},
		{"SBP equals DBP — REJECTED", 100, 100, true},
		{"SBP less than DBP — REJECTED", 80, 90, true},
		{"Minimal difference", 81, 80, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateBPPair(tc.sbp, tc.dbp)
			if tc.wantReject {
				assert.NotNil(t, result, "should return validation result")
				assert.Equal(t, models.ValidationRejected, result.Status)
			} else {
				assert.Nil(t, result, "should return nil for valid BP pair")
			}
		})
	}
}

func TestValidateLab_FlagReasonIncludesLabType(t *testing.T) {
	v := NewLabValidator()

	result := v.Validate(models.LabTypeCreatinine, 0.1)
	assert.Equal(t, models.ValidationRejected, result.Status)
	assert.Contains(t, result.FlagReason, models.LabTypeCreatinine,
		"Flag reason should contain the lab type")
}
