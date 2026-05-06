package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// TestUpsertResidentRoundTrip exercises Resident upsert + read against a
// live kb-20 Postgres. Skipped when KB20_TEST_DATABASE_URL is unset so
// the unit-test pass on CI/local-without-DB stays green.
func TestUpsertResidentRoundTrip(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping integration test")
	}
	store, err := NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("NewV2SubstrateStore: %v", err)
	}
	defer store.Close()

	in := models.Resident{
		ID:            uuid.New(),
		IHI:           "8003608000000570",
		GivenName:     "Margaret",
		FamilyName:    "Brown",
		DOB:           time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		Sex:           "female",
		FacilityID:    uuid.New(),
		CareIntensity: models.CareIntensityActive,
		Status:        models.ResidentStatusActive,
	}

	out, err := store.UpsertResident(context.Background(), in)
	if err != nil {
		t.Fatalf("UpsertResident: %v", err)
	}
	if out.IHI != in.IHI {
		t.Errorf("IHI mismatch: got %q want %q", out.IHI, in.IHI)
	}

	fetched, err := store.GetResident(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("GetResident: %v", err)
	}
	if fetched.GivenName != in.GivenName {
		t.Errorf("GivenName mismatch: got %q want %q", fetched.GivenName, in.GivenName)
	}
	if fetched.FamilyName != in.FamilyName {
		t.Errorf("FamilyName mismatch: got %q want %q", fetched.FamilyName, in.FamilyName)
	}
}

// TestUpsertPersonAndRoleRoundTrip exercises Person + Role upsert + list.
// Skipped when KB20_TEST_DATABASE_URL is unset.
func TestUpsertPersonAndRoleRoundTrip(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping integration test")
	}
	store, err := NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("NewV2SubstrateStore: %v", err)
	}
	defer store.Close()

	person := models.Person{
		ID:         uuid.New(),
		GivenName:  "Sarah",
		FamilyName: "Chen",
		HPII:       "8003614900000000",
	}
	pOut, err := store.UpsertPerson(context.Background(), person)
	if err != nil {
		t.Fatalf("UpsertPerson: %v", err)
	}
	if pOut.HPII != person.HPII {
		t.Errorf("HPII mismatch: got %q want %q", pOut.HPII, person.HPII)
	}

	role := models.Role{
		ID:        uuid.New(),
		PersonID:  person.ID,
		Kind:      models.RoleEN,
		ValidFrom: time.Now().UTC(),
	}
	rOut, err := store.UpsertRole(context.Background(), role)
	if err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	if rOut.Kind != role.Kind {
		t.Errorf("Kind mismatch: got %q want %q", rOut.Kind, role.Kind)
	}

	roles, err := store.ListRolesByPerson(context.Background(), person.ID)
	if err != nil {
		t.Fatalf("ListRolesByPerson: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
}
