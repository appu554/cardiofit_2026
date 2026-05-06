package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// TestKB20ClientUpsertAndGetResident exercises the client against an
// httptest-backed fake kb-20. No real DB required.
func TestKB20ClientUpsertAndGetResident(t *testing.T) {
	in := models.Resident{
		ID:            uuid.New(),
		GivenName:     "Margaret",
		FamilyName:    "Brown",
		DOB:           time.Now().UTC(),
		Sex:           "female",
		FacilityID:    uuid.New(),
		CareIntensity: models.CareIntensityActive,
		Status:        models.ResidentStatusActive,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/residents":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/residents/"+in.ID.String():
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(in)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewKB20Client(server.URL)
	out, err := c.UpsertResident(context.Background(), in)
	if err != nil {
		t.Fatalf("UpsertResident: %v", err)
	}
	if out.GivenName != in.GivenName {
		t.Errorf("GivenName mismatch: got %q want %q", out.GivenName, in.GivenName)
	}

	fetched, err := c.GetResident(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("GetResident: %v", err)
	}
	if fetched.ID != in.ID {
		t.Errorf("ID mismatch: got %s want %s", fetched.ID, in.ID)
	}
}

// TestKB20ClientPersonAndRole gives some coverage to the Person/Role
// branches as well, again using only httptest.
func TestKB20ClientPersonAndRole(t *testing.T) {
	person := models.Person{
		ID:         uuid.New(),
		GivenName:  "Sarah",
		FamilyName: "Chen",
		HPII:       "8003614900000000",
	}
	role := models.Role{
		ID:        uuid.New(),
		PersonID:  person.ID,
		Kind:      models.RoleEN,
		ValidFrom: time.Now().UTC(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/persons":
			_ = json.NewEncoder(w).Encode(person)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/persons/"+person.ID.String():
			_ = json.NewEncoder(w).Encode(person)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/persons" && r.URL.Query().Get("hpii") == person.HPII:
			_ = json.NewEncoder(w).Encode(person)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/roles":
			_ = json.NewEncoder(w).Encode(role)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/persons/"+person.ID.String()+"/roles":
			_ = json.NewEncoder(w).Encode([]models.Role{role})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewKB20Client(server.URL)

	if _, err := c.UpsertPerson(context.Background(), person); err != nil {
		t.Fatalf("UpsertPerson: %v", err)
	}
	if got, err := c.GetPerson(context.Background(), person.ID); err != nil || got.ID != person.ID {
		t.Fatalf("GetPerson: got=%v err=%v", got, err)
	}
	if got, err := c.GetPersonByHPII(context.Background(), person.HPII); err != nil || got.HPII != person.HPII {
		t.Fatalf("GetPersonByHPII: got=%v err=%v", got, err)
	}
	if _, err := c.UpsertRole(context.Background(), role); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	roles, err := c.ListRolesByPerson(context.Background(), person.ID)
	if err != nil || len(roles) != 1 {
		t.Fatalf("ListRolesByPerson: roles=%v err=%v", roles, err)
	}
}

// TestKB20ClientMedicineUseRoundTrip exercises the MedicineUse client paths
// against an httptest-backed fake kb-20. No real DB required.
func TestKB20ClientMedicineUseRoundTrip(t *testing.T) {
	residentID := uuid.New()
	spec, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90})
	mu := models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  residentID,
		DisplayName: "Perindopril 4mg",
		Intent: models.Intent{
			Category:   models.IntentTherapeutic,
			Indication: "Hypertension",
		},
		Target: models.Target{
			Kind: models.TargetKindBPThreshold,
			Spec: spec,
		},
		StopCriteria: models.StopCriteria{
			Triggers: []string{models.StopTriggerAdverseEvent},
		},
		StartedAt: time.Now().UTC(),
		Status:    models.MedicineUseStatusActive,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/medicine_uses":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mu)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/medicine_uses/"+mu.ID.String():
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mu)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/residents/"+residentID.String()+"/medicine_uses":
			if r.URL.Query().Get("limit") != "50" || r.URL.Query().Get("offset") != "0" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]models.MedicineUse{mu})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewKB20Client(server.URL)

	out, err := c.UpsertMedicineUse(context.Background(), mu)
	if err != nil {
		t.Fatalf("UpsertMedicineUse: %v", err)
	}
	if out.DisplayName != mu.DisplayName {
		t.Errorf("DisplayName mismatch: got %q want %q", out.DisplayName, mu.DisplayName)
	}

	fetched, err := c.GetMedicineUse(context.Background(), mu.ID)
	if err != nil {
		t.Fatalf("GetMedicineUse: %v", err)
	}
	if fetched.ID != mu.ID {
		t.Errorf("ID mismatch: got %s want %s", fetched.ID, mu.ID)
	}

	list, err := c.ListMedicineUsesByResident(context.Background(), residentID, 50, 0)
	if err != nil {
		t.Fatalf("ListMedicineUsesByResident: %v", err)
	}
	if len(list) != 1 || list[0].ID != mu.ID {
		t.Errorf("ListMedicineUsesByResident: got %+v want 1 entry with id=%s", list, mu.ID)
	}
}
