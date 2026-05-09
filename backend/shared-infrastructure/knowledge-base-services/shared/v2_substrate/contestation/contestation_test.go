package contestation

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newValidContestation returns a fully-populated Contestation that passes Validate().
func newValidContestation() Contestation {
	return Contestation{
		ID:                 uuid.New(),
		PharmacistID:       uuid.New(),
		EmployerID:         uuid.New(),
		KPIType:            "dispensing_accuracy",
		KPISnapshot:        map[string]any{"rate": 0.94, "period": "2026-Q1"},
		PharmacistArgument: "The system counted a near-miss as a dispensing error; clinical review confirms no patient harm.",
		EmployerResponse:   "",
		Status:             StatusOpen,
		FiledAt:            time.Now().UTC(),
	}
}

func TestContestation_Validate_RejectsEmptyKPIType(t *testing.T) {
	c := newValidContestation()
	c.KPIType = ""
	if err := c.Validate(); !errors.Is(err, ErrEmptyKPIType) {
		t.Errorf("want ErrEmptyKPIType, got %v", err)
	}
}

func TestContestation_Validate_RejectsEmptyArgument(t *testing.T) {
	c := newValidContestation()
	c.PharmacistArgument = ""
	if err := c.Validate(); !errors.Is(err, ErrEmptyPharmacistArgument) {
		t.Errorf("want ErrEmptyPharmacistArgument, got %v", err)
	}
}

func TestContestation_Validate_RejectsInvalidStatus(t *testing.T) {
	c := newValidContestation()
	c.Status = "pending" // not in the allowed set
	if err := c.Validate(); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("want ErrInvalidStatus, got %v", err)
	}
}

func TestContestation_Validate_HappyPath(t *testing.T) {
	c := newValidContestation()
	if err := c.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestIsValidStatus_AllFour(t *testing.T) {
	valid := []string{StatusOpen, StatusResponded, StatusResolved, StatusWithdrawn}
	for _, s := range valid {
		if !IsValidStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if IsValidStatus("bogus") {
		t.Error("expected \"bogus\" to be invalid")
	}
}
