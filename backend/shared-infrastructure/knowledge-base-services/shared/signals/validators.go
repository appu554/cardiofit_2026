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
	reason := ""
	if rule.priorityCheck != nil {
		priority = rule.priorityCheck(value)
		if priority {
			reason = "priority threshold breached"
		}
	}
	return ValidationResult{
		Status:   ValidationAccepted,
		Priority: priority,
		Reason:   reason,
	}
}

type validationRule struct {
	minPlausible  float64
	maxPlausible  float64
	priorityCheck func(float64) bool
}

var validationRules = map[SignalType]validationRule{
	SignalFBG:        {minPlausible: 0.5, maxPlausible: 50.0, priorityCheck: func(v float64) bool { return v < 4.0 || v > 20.0 }},
	SignalPPBG:       {minPlausible: 0.5, maxPlausible: 50.0, priorityCheck: func(v float64) bool { return v < 4.0 }},
	SignalHbA1c:      {minPlausible: 3.0, maxPlausible: 20.0},
	SignalSBP:        {minPlausible: 40, maxPlausible: 300, priorityCheck: func(v float64) bool { return v > 180 || v < 90 }},
	SignalDBP:        {minPlausible: 20, maxPlausible: 200},
	SignalHR:         {minPlausible: 20, maxPlausible: 300, priorityCheck: func(v float64) bool { return v < 40 || v > 150 }},
	SignalCreatinine: {minPlausible: 0.1, maxPlausible: 30.0, priorityCheck: func(v float64) bool { return v > 10.0 }},
	SignalACR:        {minPlausible: 0, maxPlausible: 5000, priorityCheck: func(v float64) bool { return v > 300 }},
	SignalPotassium:  {minPlausible: 1.0, maxPlausible: 10.0, priorityCheck: func(v float64) bool { return v > 5.5 || v < 3.0 }},
	SignalWeight:     {minPlausible: 1.0, maxPlausible: 500.0},
	SignalWaist:      {minPlausible: 20.0, maxPlausible: 250.0},
}
