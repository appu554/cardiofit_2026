package signals

import "testing"

func TestValidateFBG_Accept(t *testing.T) {
	result := ValidateSignal(SignalFBG, 5.5)
	if result.Status != ValidationAccepted {
		t.Errorf("expected ACCEPTED, got %s", result.Status)
	}
}

func TestValidateFBG_Priority(t *testing.T) {
	result := ValidateSignal(SignalFBG, 3.5)
	if !result.Priority {
		t.Error("expected priority flag for FBG < 4.0")
	}
}

func TestValidateFBG_Reject(t *testing.T) {
	result := ValidateSignal(SignalFBG, -1.0)
	if result.Status != ValidationRejected {
		t.Errorf("expected REJECTED for negative FBG, got %s", result.Status)
	}
}

func TestValidateSBP_Priority(t *testing.T) {
	result := ValidateSignal(SignalSBP, 185.0)
	if !result.Priority {
		t.Error("expected priority flag for SBP > 180")
	}
}

func TestValidateHR_Priority_Low(t *testing.T) {
	result := ValidateSignal(SignalHR, 35.0)
	if !result.Priority {
		t.Error("expected priority flag for HR < 40")
	}
}

func TestValidateWeight_Normal(t *testing.T) {
	result := ValidateSignal(SignalWeight, 75.0)
	if result.Status != ValidationAccepted {
		t.Errorf("expected ACCEPTED, got %s", result.Status)
	}
	if result.Priority {
		t.Error("unexpected priority for normal weight")
	}
}
