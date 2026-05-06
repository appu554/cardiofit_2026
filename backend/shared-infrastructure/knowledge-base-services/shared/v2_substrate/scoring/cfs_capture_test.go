package scoring

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestCFSScoreLabel(t *testing.T) {
	if got := CFSScoreLabel(7); got != "Living with severe frailty" {
		t.Errorf("CFSScoreLabel(7) = %q", got)
	}
	if got := CFSScoreLabel(0); got != "" {
		t.Errorf("CFSScoreLabel(0) = %q; expected empty", got)
	}
	if got := CFSScoreLabel(10); got != "" {
		t.Errorf("CFSScoreLabel(10) = %q; expected empty", got)
	}
}

func TestValidateCFSCapture_DelegatesToValidation(t *testing.T) {
	good := models.CFSScore{
		ResidentRef:       uuid.New(),
		AssessedAt:        time.Now().UTC(),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: CFSInstrumentVersionCurrent,
		Score:             4,
	}
	if err := ValidateCFSCapture(good); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
	bad := good
	bad.Score = 0
	if err := ValidateCFSCapture(bad); err == nil {
		t.Errorf("expected error for score=0")
	}
}
