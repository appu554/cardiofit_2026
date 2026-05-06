// Package interfaces declares storage and transport contracts for the v2
// substrate. The canonical KB (kb-20 for actor entities) implements these
// interfaces; other KBs use them via clients.
package interfaces

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrNotFound is returned by stores when a requested entity does not exist.
// Handlers should check errors.Is(err, ErrNotFound) to choose 404 vs 500.
var ErrNotFound = errors.New("v2_substrate: entity not found")

// ResidentStore is the canonical storage contract for Resident entities.
// kb-20-patient-profile is the only KB expected to implement this.
type ResidentStore interface {
	GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error)
	UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error)
	// ListResidentsByFacility returns residents at the given facility, paginated.
	// limit must be > 0 (caller's responsibility); offset >= 0. The implementation
	// may apply a maximum cap (e.g. 1000) but caller should not rely on that.
	ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID, limit, offset int) ([]models.Resident, error)
}

// PersonStore is the canonical storage contract for Person entities.
type PersonStore interface {
	GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error)
	UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error)
	GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error)
}

// RoleStore is the canonical storage contract for Role entities.
type RoleStore interface {
	GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error)
	UpsertRole(ctx context.Context, r models.Role) (*models.Role, error)
	ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error)
	// ListActiveRolesByPersonAndFacility returns only roles where ValidFrom <= now <= ValidTo (or ValidTo is nil)
	// and (FacilityID is nil OR FacilityID == facilityID). Used by the future Authorisation evaluator.
	ListActiveRolesByPersonAndFacility(ctx context.Context, personID uuid.UUID, facilityID uuid.UUID) ([]models.Role, error)
}

// MedicineUseStore is the canonical storage contract for MedicineUse entities.
// kb-20-patient-profile is the only KB expected to implement this.
type MedicineUseStore interface {
	GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error)
	UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error)
	ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error)
}

// ObservationStore is the canonical storage contract for Observation entities.
// kb-20-patient-profile is the only KB expected to implement this. List
// methods take limit/offset; the implementation may apply a maximum cap
// (e.g. 1000) but caller should not rely on that.
//
// Implementations of UpsertObservation MUST compute Delta before insert via
// shared/v2_substrate/delta.ComputeDelta with an injected BaselineProvider;
// when the provider returns delta.ErrNoBaseline (or Value is nil or
// Kind=behavioural), the resulting Delta.DirectionalFlag must be
// DeltaFlagNoBaseline.
type ObservationStore interface {
	GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error)
	UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error)
	ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error)
	ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error)
}

// EventStore is the canonical storage contract for Event entities.
// kb-20-patient-profile is the only KB expected to implement this. List
// methods take limit/offset; the implementation may apply a maximum cap
// (e.g. 1000) but caller should not rely on that.
type EventStore interface {
	GetEvent(ctx context.Context, id uuid.UUID) (*models.Event, error)
	UpsertEvent(ctx context.Context, e models.Event) (*models.Event, error)
	ListEventsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Event, error)
	// ListEventsByType returns events of a given event_type whose occurred_at
	// falls inside [from, to). A zero `from` means no lower bound; a zero `to`
	// means no upper bound. Results are ordered by occurred_at DESC.
	ListEventsByType(ctx context.Context, eventType string, from, to time.Time, limit, offset int) ([]models.Event, error)
}
