package services

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestValidationManager() *ValidationManager {
	log, _ := zap.NewDevelopment()
	return NewValidationManager(log)
}

// TestValidation_TimeOfDay_Flags — usual hour 7, reading hour 15 (diff=8 >2)
// → returns UNCONFIRMED with reason TIME_OF_DAY_INCONSISTENT.
func TestValidation_TimeOfDay_Flags(t *testing.T) {
	vm := newTestValidationManager()

	state, reason := vm.CheckWeightValidation(
		82.0,  // currentValue
		80.0,  // baselineMedian
		2.0,   // deviation
		"HIGH", // severity
		15,    // measurementHour (3 PM)
		7,     // usualHour (7 AM) — diff = 8
		false, // hasPriorDeviation
		false, // isCriticalHF
	)

	if state != ValidationUnconfirmed {
		t.Errorf("expected UNCONFIRMED, got %s", state)
	}
	if reason != "TIME_OF_DAY_INCONSISTENT" {
		t.Errorf("expected TIME_OF_DAY_INCONSISTENT, got %s", reason)
	}
}

// TestValidation_FirstDeviation_AwaitingConfirmation — weight deviation 2.2kg,
// severity HIGH, no prior deviation → returns AWAITING_CONFIRMATION.
func TestValidation_FirstDeviation_AwaitingConfirmation(t *testing.T) {
	vm := newTestValidationManager()

	state, reason := vm.CheckWeightValidation(
		82.2,  // currentValue
		80.0,  // baselineMedian
		2.2,   // deviation
		"HIGH", // severity
		7,     // measurementHour
		7,     // usualHour — consistent
		false, // hasPriorDeviation — first deviation
		false, // isCriticalHF
	)

	if state != ValidationAwaitingConfirmation {
		t.Errorf("expected AWAITING_CONFIRMATION, got %s", state)
	}
	if reason != "FIRST_DEVIATION_NEEDS_CONFIRMATION" {
		t.Errorf("expected FIRST_DEVIATION_NEEDS_CONFIRMATION, got %s", reason)
	}
}

// TestValidation_Confirmation_Within20Pct_Confirmed — original deviation 2.2kg,
// confirmation deviation 2.0kg (9% diff, <20%) → returns CONFIRMED.
func TestValidation_Confirmation_Within20Pct_Confirmed(t *testing.T) {
	vm := newTestValidationManager()

	state := vm.ProcessConfirmation(2.2, 2.0)

	if state != ValidationConfirmed {
		t.Errorf("expected CONFIRMED, got %s", state)
	}
}

// TestValidation_Confirmation_Over50Pct_Refuted — original 2.2kg,
// confirmation 0.5kg (77% diff, >50%) → returns REFUTED.
func TestValidation_Confirmation_Over50Pct_Refuted(t *testing.T) {
	vm := newTestValidationManager()

	state := vm.ProcessConfirmation(2.2, 0.5)

	if state != ValidationRefuted {
		t.Errorf("expected REFUTED, got %s", state)
	}
}

// TestValidation_Expired_NoConfirmation — pending created 25h ago,
// no confirmation → returns true from IsExpired.
func TestValidation_Expired_NoConfirmation(t *testing.T) {
	vm := newTestValidationManager()

	now := time.Now().UTC()
	expiresAt := now.Add(-1 * time.Hour) // expired 1h ago (was set 25h ago, 24h window)

	if !vm.IsExpired(expiresAt, now) {
		t.Error("expected pending validation to be expired")
	}

	// Verify non-expired case.
	futureExpiry := now.Add(1 * time.Hour)
	if vm.IsExpired(futureExpiry, now) {
		t.Error("expected pending validation to NOT be expired")
	}
}

// TestValidation_Critical_BypassesWaiting — weight gain 3.5kg in CKM 4c
// → returns UNCONFIRMED_CRITICAL (fires immediately but notes uncertainty).
func TestValidation_Critical_BypassesWaiting(t *testing.T) {
	vm := newTestValidationManager()

	state, reason := vm.CheckWeightValidation(
		83.5,      // currentValue
		80.0,      // baselineMedian
		3.5,       // deviation
		"CRITICAL", // severity
		7,         // measurementHour
		7,         // usualHour — consistent (time-of-day passes)
		false,     // hasPriorDeviation
		true,      // isCriticalHF — CKM 4c
	)

	if state != ValidationUnconfirmedCritical {
		t.Errorf("expected UNCONFIRMED_CRITICAL, got %s", state)
	}
	if reason != "CRITICAL_HF_BYPASS" {
		t.Errorf("expected CRITICAL_HF_BYPASS, got %s", reason)
	}
}
