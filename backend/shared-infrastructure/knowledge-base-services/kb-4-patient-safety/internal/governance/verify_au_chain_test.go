package governance

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// fakeStore is an in-memory ChainStore used by the unit tests so the
// signature-verification logic can be exercised without a database.
type fakeStore struct {
	rulesByCS    map[string][]SignedRule
	approvalsBy  map[string][]ApprovalEntry // key = "cs/id"
}

func (f *fakeStore) ListSignedRules(_ context.Context, cs string) ([]SignedRule, error) {
	return f.rulesByCS[cs], nil
}
func (f *fakeStore) ListApprovals(_ context.Context, cs, id string) ([]ApprovalEntry, error) {
	return f.approvalsBy[cs+"/"+id], nil
}

// signedRuleFor builds a SignedRule whose signature verifies under
// privKey, with content "body".
func signedRuleFor(t *testing.T, priv ed25519.PrivateKey, cs, id, body string) SignedRule {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	hexSum := hex.EncodeToString(sum[:])
	sig := ed25519.Sign(priv, sum[:])
	return SignedRule{
		CriterionSet: cs,
		CriterionID:  id,
		ContentSHA:   hexSum,
		SignedSHA:    hexSum,
		Signature:    sig,
	}
}

func dualApproval() []ApprovalEntry {
	return []ApprovalEntry{
		{ReviewerRole: "CLINICAL_REVIEWER", ReviewerID: "rev-1"},
		{ReviewerRole: "MEDICAL_DIRECTOR", ReviewerID: "md-1"},
	}
}

func newKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

func TestVerifyRuleSignature_Happy(t *testing.T) {
	pub, priv := newKey(t)
	r := signedRuleFor(t, priv, "STOPP_V3", "A1", "rule-body-canonicalised")
	if err := VerifyRuleSignature(r, pub); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVerifyRuleSignature_Drift(t *testing.T) {
	pub, priv := newKey(t)
	r := signedRuleFor(t, priv, "STOPP_V3", "A1", "rule-body")
	r.ContentSHA = strings.Repeat("0", 64) // mutate to simulate drift
	err := VerifyRuleSignature(r, pub)
	if err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("expected drift error, got %v", err)
	}
}

func TestVerifyRuleSignature_BadSignature(t *testing.T) {
	pub, priv := newKey(t)
	r := signedRuleFor(t, priv, "STOPP_V3", "A1", "rule-body")
	// flip a bit in the signature
	r.Signature[0] ^= 0xFF
	err := VerifyRuleSignature(r, pub)
	if err == nil || !strings.Contains(err.Error(), "did not verify") {
		t.Fatalf("expected verify failure, got %v", err)
	}
}

func TestVerifyRuleSignature_WrongPubKey(t *testing.T) {
	_, priv := newKey(t)
	otherPub, _ := newKey(t)
	r := signedRuleFor(t, priv, "STOPP_V3", "A1", "rule-body")
	if err := VerifyRuleSignature(r, otherPub); err == nil {
		t.Fatalf("expected failure under wrong pub key")
	}
}

func TestVerifyDualApproval_Happy(t *testing.T) {
	if err := VerifyDualApproval(dualApproval()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestVerifyDualApproval_MissingMD(t *testing.T) {
	entries := []ApprovalEntry{{ReviewerRole: "CLINICAL_REVIEWER", ReviewerID: "rev-1"}}
	err := VerifyDualApproval(entries)
	if err == nil || !strings.Contains(err.Error(), "MEDICAL_DIRECTOR") {
		t.Fatalf("expected missing MD, got %v", err)
	}
}

func TestVerifyChain_HappyAllSets(t *testing.T) {
	pub, priv := newKey(t)
	store := &fakeStore{
		rulesByCS:   map[string][]SignedRule{},
		approvalsBy: map[string][]ApprovalEntry{},
	}
	for _, cs := range CriterionSetsInScope {
		r := signedRuleFor(t, priv, cs, "001", "body-"+cs)
		store.rulesByCS[cs] = []SignedRule{r}
		store.approvalsBy[cs+"/001"] = dualApproval()
	}
	if err := VerifyChain(context.Background(), store, pub); err != nil {
		t.Fatalf("expected clean chain, got %v", err)
	}
}

func TestVerifyChain_EmptyCriterionSetFails(t *testing.T) {
	pub, _ := newKey(t)
	store := &fakeStore{rulesByCS: map[string][]SignedRule{}}
	err := VerifyChain(context.Background(), store, pub)
	if err == nil || !strings.Contains(err.Error(), "no signed rules") {
		t.Fatalf("expected empty-set failure, got %v", err)
	}
}

func TestVerifyChain_PrefixesCriterionSetAndID(t *testing.T) {
	pub, priv := newKey(t)
	store := &fakeStore{
		rulesByCS:   map[string][]SignedRule{},
		approvalsBy: map[string][]ApprovalEntry{},
	}
	for _, cs := range CriterionSetsInScope {
		r := signedRuleFor(t, priv, cs, "001", "body-"+cs)
		store.rulesByCS[cs] = []SignedRule{r}
		store.approvalsBy[cs+"/001"] = dualApproval()
	}
	// Corrupt the START_V3/001 signature so the failure message is
	// expected to include "[START_V3/001]" prefix.
	bad := store.rulesByCS["START_V3"][0]
	bad.Signature[0] ^= 0xFF
	store.rulesByCS["START_V3"] = []SignedRule{bad}
	err := VerifyChain(context.Background(), store, pub)
	if err == nil || !strings.Contains(err.Error(), "[START_V3/001]") {
		t.Fatalf("expected START_V3/001 prefix in error, got %v", err)
	}
}

func TestVerifyChain_RejectsBadPubKeyLength(t *testing.T) {
	store := &fakeStore{}
	err := VerifyChain(context.Background(), store, ed25519.PublicKey([]byte{0x00, 0x01}))
	if err == nil || !strings.Contains(err.Error(), "invalid Ed25519 public key length") {
		t.Fatalf("expected pubkey length error, got %v", err)
	}
}
