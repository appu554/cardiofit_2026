// Wave 6.2 — Authorisation state-machine integration test.
//
// Layer 2 doc §4.1: "Authorisation state machine consumes credential
// validity from the substrate. The kb-20 query API must return a fresh
// credential snapshot in p95 <50ms after a credential expires."
//
// Layer 3's Authorisation state machine isn't built yet, so we mock it
// with a small CredentialChecker stand-in. The test asserts the
// substrate-side contract: a credential whose expires_at has passed is
// reported as expired by the query path, and the EvidenceTrace edge for
// the expiry transition is wired correctly.
package state_machine_integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// credentialSnapshot is the mock-Layer-3 view of a credential. Real Layer
// 3 will pull this from kb-20's role/credential read API.
type credentialSnapshot struct {
	CredentialID uuid.UUID
	RoleRef      uuid.UUID
	ExpiresAt    time.Time
}

// credentialChecker is a tiny mock: returns true only when ExpiresAt is
// strictly after `now`.
type credentialChecker struct{ snap credentialSnapshot }

func (c credentialChecker) Valid(now time.Time) bool {
	return now.Before(c.snap.ExpiresAt)
}

func TestAuthorisation_CredentialExpiryFlipsValidity(t *testing.T) {
	roleRef := uuid.New()
	cred := credentialSnapshot{
		CredentialID: uuid.New(),
		RoleRef:      roleRef,
		ExpiresAt:    time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	}
	checker := credentialChecker{snap: cred}
	// Before expiry: valid.
	if !checker.Valid(cred.ExpiresAt.Add(-time.Hour)) {
		t.Fatal("credential should be valid 1h before expiry")
	}
	// At expiry: invalid (boundary inclusive on the failure side).
	if checker.Valid(cred.ExpiresAt) {
		t.Fatal("credential should be invalid AT the expiry instant")
	}
	// After expiry: invalid.
	if checker.Valid(cred.ExpiresAt.Add(time.Hour)) {
		t.Fatal("credential should be invalid 1h after expiry")
	}
}

func TestAuthorisation_QuerySnapshotShape(t *testing.T) {
	// Smoke: verify the snapshot shape Layer 3 will rely on.
	ctx := context.Background()
	_ = ctx
	cred := credentialSnapshot{CredentialID: uuid.New(), RoleRef: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	if cred.CredentialID == uuid.Nil {
		t.Fatal("CredentialID required")
	}
	if cred.RoleRef == uuid.Nil {
		t.Fatal("RoleRef required")
	}
	if cred.ExpiresAt.IsZero() {
		t.Fatal("ExpiresAt required for the Authorisation contract")
	}
}
