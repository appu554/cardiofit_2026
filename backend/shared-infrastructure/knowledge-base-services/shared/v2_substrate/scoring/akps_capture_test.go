package scoring

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestAKPSScoreLabel(t *testing.T) {
	if got := AKPSScoreLabel(40); got != "In bed >50% of time" {
		t.Errorf("AKPSScoreLabel(40) = %q", got)
	}
	if got := AKPSScoreLabel(15); got != "" {
		t.Errorf("AKPSScoreLabel(15) = %q; expected empty", got)
	}
}

func TestValidateAKPSCapture_DelegatesToValidation(t *testing.T) {
	good := models.AKPSScore{
		ResidentRef:       uuid.New(),
		AssessedAt:        time.Now().UTC(),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: AKPSInstrumentVersionCurrent,
		Score:             60,
	}
	if err := ValidateAKPSCapture(good); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
	bad := good
	bad.Score = 33
	if err := ValidateAKPSCapture(bad); err == nil {
		t.Errorf("expected error for non-multiple-of-10")
	}
}
