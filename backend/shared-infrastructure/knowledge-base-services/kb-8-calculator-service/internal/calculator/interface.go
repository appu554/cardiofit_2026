// Package calculator provides clinical score calculators.
//
// The calculator package implements the ATOMIC pattern - all calculations
// are pure mathematical operations with no I/O. Input data must be
// pre-fetched before calling calculators.
//
// Supported calculators:
//   - eGFR (CKD-EPI 2021, race-free)
//   - CrCl (Cockcroft-Gault)
//   - BMI (with Asian cutoffs)
//   - SOFA (ICU mortality)
//   - qSOFA (sepsis screening)
//   - CHA₂DS₂-VASc (stroke risk)
//   - HAS-BLED (bleeding risk)
//   - ASCVD 10-Year (cardiovascular risk)
package calculator

import (
	"context"

	"kb-8-calculator-service/internal/models"
)

// Calculator is the interface that all clinical calculators must implement.
type Calculator interface {
	// Type returns the calculator type identifier
	Type() models.CalculatorType

	// Name returns a human-readable name
	Name() string

	// Version returns the formula/algorithm version
	Version() string

	// Reference returns the clinical citation
	Reference() string
}

// EGFRCalculatorInterface defines the eGFR calculator contract.
type EGFRCalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.EGFRParams) (*models.EGFRResult, error)
}

// CrClCalculatorInterface defines the CrCl calculator contract.
type CrClCalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.CrClParams) (*models.CrClResult, error)
}

// BMICalculatorInterface defines the BMI calculator contract.
type BMICalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.BMIParams) (*models.BMIResult, error)
}

// SOFACalculatorInterface defines the SOFA calculator contract.
type SOFACalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.SOFAParams) (*models.SOFAResult, error)
}

// QSOFACalculatorInterface defines the qSOFA calculator contract.
type QSOFACalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.QSOFAParams) (*models.QSOFAResult, error)
}

// CHA2DS2VAScCalculatorInterface defines the CHA2DS2-VASc calculator contract.
type CHA2DS2VAScCalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.CHA2DS2VAScParams) (*models.CHA2DS2VAScResult, error)
}

// HASBLEDCalculatorInterface defines the HAS-BLED calculator contract.
type HASBLEDCalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.HASBLEDParams) (*models.HASBLEDResult, error)
}

// ASCVDCalculatorInterface defines the ASCVD calculator contract.
type ASCVDCalculatorInterface interface {
	Calculator
	Calculate(ctx context.Context, params *models.ASCVDParams) (*models.ASCVDResult, error)
}
