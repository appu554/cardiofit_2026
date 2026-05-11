package audit

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestEnforcePDPRead_SameOwner_AllowsRead(t *testing.T) {
	pharmacist := uuid.New()
	if err := EnforcePDPRead(pharmacist, pharmacist); err != nil {
		t.Fatalf("same-pharmacist read should succeed, got %v", err)
	}
}

func TestEnforcePDPRead_CrossPharmacist_Blocked(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	err := EnforcePDPRead(a, b)
	if !errors.Is(err, ErrCrossPharmacistRead) {
		t.Fatalf("cross-pharmacist read should return ErrCrossPharmacistRead, got %v", err)
	}
}

func TestEnforcePDPRead_NilUUIDs_Blocked(t *testing.T) {
	if err := EnforcePDPRead(uuid.Nil, uuid.New()); !errors.Is(err, ErrCrossPharmacistRead) {
		t.Errorf("nil requester should be blocked, got %v", err)
	}
	if err := EnforcePDPRead(uuid.New(), uuid.Nil); !errors.Is(err, ErrCrossPharmacistRead) {
		t.Errorf("nil owner should be blocked, got %v", err)
	}
}

func TestEnforcePDPAggregateRead_AuthorisedRoles(t *testing.T) {
	for _, role := range []string{RoleClinicalInformatics, RoleEthicsSteeringCommittee} {
		if err := EnforcePDPAggregateRead(role); err != nil {
			t.Errorf("role %q should be authorised for aggregate read, got %v", role, err)
		}
	}
}

func TestEnforcePDPAggregateRead_UnauthorisedRoles_Blocked(t *testing.T) {
	for _, role := range []string{"employer", "manager", "pharmacist", "auditor", "", "admin"} {
		err := EnforcePDPAggregateRead(role)
		if !errors.Is(err, ErrSurveillanceAttempt) {
			t.Errorf("role %q should return ErrSurveillanceAttempt, got %v", role, err)
		}
	}
}

func TestVisibilityClassConstants(t *testing.T) {
	// Pin the four enum values — drift here would mean audit rows are
	// classified inconsistently with the pharmacist self-visibility module.
	want := map[VisibilityClass]string{
		VisibilityPDP: "PDP",
		VisibilityPEV: "PEV",
		VisibilityAD:  "AD",
		VisibilityPDF: "PDF",
	}
	for v, s := range want {
		if string(v) != s {
			t.Errorf("VisibilityClass %q drifted: want %q got %q", s, s, string(v))
		}
	}
}
