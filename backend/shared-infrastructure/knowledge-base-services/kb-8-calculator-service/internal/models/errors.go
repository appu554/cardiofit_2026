package models

import "errors"

// Validation errors
var (
	// Parameter validation errors
	ErrInvalidCreatinine     = errors.New("invalid serum creatinine value")
	ErrInvalidAge            = errors.New("invalid age: must be between 1 and 120 years")
	ErrInvalidSex            = errors.New("invalid sex: must be 'male' or 'female'")
	ErrInvalidWeight         = errors.New("invalid weight: must be between 1 and 500 kg")
	ErrInvalidHeight         = errors.New("invalid height: must be between 1 and 300 cm")
	ErrMissingRequiredParams = errors.New("missing required parameters")

	// Calculator errors
	ErrCalculatorNotFound    = errors.New("calculator not found")
	ErrCalculatorUnavailable = errors.New("calculator temporarily unavailable")
	ErrInsufficientData      = errors.New("insufficient data for calculation")
)
