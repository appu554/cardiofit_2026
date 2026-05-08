package contestation

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Contestation records a pharmacist's challenge to an algorithmic KPI feeding
// any employment-affecting decision. Per v3 §9 line 514: the algorithmic
// determination cannot be the sole basis for an adverse employment decision.
//
// Visibility: an active contestation record is visible to both pharmacist
// (subject) and employer (challenged party) per the dual-disclosure principle.
type Contestation struct {
	ID                 uuid.UUID
	PharmacistID       uuid.UUID
	EmployerID         uuid.UUID
	KPIType            string
	KPISnapshot        map[string]any // JSON-marshalable
	PharmacistArgument string
	EmployerResponse   string
	Status             string
	FiledAt            time.Time
	ResolvedAt         *time.Time
}

// Status enum per migration 028 CHECK constraint.
const (
	StatusOpen      = "open"
	StatusResponded = "responded"
	StatusResolved  = "resolved"
	StatusWithdrawn = "withdrawn"
)

// ValidStatuses is the complete list of accepted status values.
var ValidStatuses = []string{StatusOpen, StatusResponded, StatusResolved, StatusWithdrawn}

// IsValidStatus reports whether s is a recognised Contestation status.
func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Sentinel errors at package scope (matches Task 1 + 2.5 conventions).
var (
	ErrEmptyKPIType            = errors.New("contestation: kpi_type required")
	ErrEmptyPharmacistArgument = errors.New("contestation: pharmacist_argument required")
	ErrInvalidStatus           = errors.New("contestation: invalid status")
)

// Validate enforces the entity invariants. Returns one of the above sentinels.
func (c Contestation) Validate() error {
	if c.KPIType == "" {
		return ErrEmptyKPIType
	}
	if c.PharmacistArgument == "" {
		return ErrEmptyPharmacistArgument
	}
	if !IsValidStatus(c.Status) {
		return ErrInvalidStatus
	}
	return nil
}
