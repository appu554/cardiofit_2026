package signals

// ValidationStatus is the outcome of signal plausibility validation.
type ValidationStatus string

const (
	ValidationAccepted ValidationStatus = "ACCEPTED"
	ValidationFlagged  ValidationStatus = "FLAGGED"
	ValidationRejected ValidationStatus = "REJECTED"
)

// ValidationResult carries the outcome of a signal validation.
type ValidationResult struct {
	Status   ValidationStatus
	Priority bool
	Reason   string
}

// ValidateSignal checks plausibility and priority for a given signal type and value.
func ValidateSignal(st SignalType, value float64) ValidationResult {
	rule, ok := validationRules[st]
	if !ok {
		return ValidationResult{Status: ValidationAccepted}
	}
	if value < rule.minPlausible || value > rule.maxPlausible {
		return ValidationResult{
			Status: ValidationRejected,
			Reason: "value outside plausible range",
		}
	}
	priority := false
	if rule.priorityCheck != nil {
		priority = rule.priorityCheck(value)
	}
	return ValidationResult{
		Status:   ValidationAccepted,
		Priority: priority,
	}
}

type validationRule struct {
	minPlausible  float64
	maxPlausible  float64
	priorityCheck func(float64) bool
}

var validationRules = map[SignalType]validationRule{
	SignalFBG:        {0.5, 50.0, func(v float64) bool { return v < 4.0 || v > 20.0 }},
	SignalPPBG:       {0.5, 50.0, func(v float64) bool { return v < 4.0 }},
	SignalHbA1c:      {3.0, 20.0, nil},
	SignalSBP:        {40, 300, func(v float64) bool { return v > 180 || v < 90 }},
	SignalDBP:        {20, 200, nil},
	SignalHR:         {20, 300, func(v float64) bool { return v < 40 || v > 150 }},
	SignalCreatinine: {0.1, 30.0, func(v float64) bool { return v > 10.0 }},
	SignalACR:        {0, 5000, func(v float64) bool { return v > 300 }},
	SignalPotassium:  {1.0, 10.0, func(v float64) bool { return v > 5.5 || v < 3.0 }},
	SignalWeight:     {1.0, 500.0, nil},
	SignalWaist:      {20.0, 250.0, nil},
}
