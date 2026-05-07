// Package governance contains the KB-4 audit-chain verification logic
// used by the Layer 3 Pre-Wave Task 2 deliverable.
//
// The verifier walks every signed explicit-criteria rule across the 8
// criterion sets currently in scope (STOPP_V3, START_V3, BEERS_2023,
// BEERS_RENAL, ACB, PIMS_WANG, AU_APINCHS, AU_TGA_BLACKBOX) and
// asserts:
//
//  1. an Ed25519 signature is present for every rule;
//  2. the signature payload's content_sha matches the stored
//     content_sha column (no drift between signed payload and current
//     row contents);
//  3. dual-approval audit entries exist for the rule — at least one
//     reviewer with role 'CLINICAL_REVIEWER' and one with role
//     'MEDICAL_DIRECTOR' (the L6 dual-approval rule);
//  4. the Ed25519 signature itself verifies against the platform
//     public key.
//
// Any failure surfaces as a non-nil error from VerifyChain with the
// failing (criterion_set, criterion_id) prefixed in the error message.
// The CLI wrapper (cmd/verify-au-chain) maps a non-nil error to exit
// status 1 and a nil error to exit status 0.
//
// The verifier is intentionally written against the database/sql
// interface so a test can pass an in-memory or fake *sql.DB. The
// signature-verification logic is factored into VerifyRuleSignature
// for unit testing without a database.
package governance

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

// CriterionSetsInScope is the canonical list of criterion sets the
// L6 governance chain covers as of 2026-05-06. Adding to this list
// requires a Layer 3 governance review (per migration 007 comment).
var CriterionSetsInScope = []string{
	"STOPP_V3",
	"START_V3",
	"BEERS_2023",
	"BEERS_RENAL",
	"ACB",
	"PIMS_WANG",
	"AU_APINCHS",
	"AU_TGA_BLACKBOX",
}

// SignedRule captures the columns the verifier reads from the
// kb4_explicit_criteria + kb4_rule_signatures join.
type SignedRule struct {
	CriterionSet string
	CriterionID  string
	ContentSHA   string // hex-encoded SHA-256 of canonicalised rule body
	Signature    []byte // raw Ed25519 signature (64 bytes)
	SignedSHA    string // hex SHA-256 the signer captured at signing time
}

// ApprovalEntry is a single row of the dual-approval audit trail.
type ApprovalEntry struct {
	ReviewerRole string
	ReviewerID   string
}

// ChainStore is the minimal data-access surface the verifier needs.
// The production implementation is a thin wrapper over *sql.DB; tests
// can implement this interface in-memory.
type ChainStore interface {
	ListSignedRules(ctx context.Context, criterionSet string) ([]SignedRule, error)
	ListApprovals(ctx context.Context, criterionSet, criterionID string) ([]ApprovalEntry, error)
}

// SQLChainStore is the production implementation backed by *sql.DB.
type SQLChainStore struct {
	DB *sql.DB
}

// ListSignedRules joins kb4_explicit_criteria with kb4_rule_signatures
// (signature table assumed present per the L6 governance design).
func (s *SQLChainStore) ListSignedRules(ctx context.Context, criterionSet string) ([]SignedRule, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.criterion_set, c.criterion_id, c.content_sha,
		       s.signature_bytes, s.signed_content_sha
		  FROM kb4_explicit_criteria c
		  JOIN kb4_rule_signatures  s
		    ON s.criterion_set = c.criterion_set
		   AND s.criterion_id  = c.criterion_id
		 WHERE c.criterion_set = $1
		 ORDER BY c.criterion_id`, criterionSet)
	if err != nil {
		return nil, fmt.Errorf("query signed rules for %s: %w", criterionSet, err)
	}
	defer rows.Close()
	out := []SignedRule{}
	for rows.Next() {
		var r SignedRule
		if err := rows.Scan(&r.CriterionSet, &r.CriterionID, &r.ContentSHA, &r.Signature, &r.SignedSHA); err != nil {
			return nil, fmt.Errorf("scan signed rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListApprovals returns the dual-approval entries for a given rule.
func (s *SQLChainStore) ListApprovals(ctx context.Context, criterionSet, criterionID string) ([]ApprovalEntry, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT reviewer_role, reviewer_id
		  FROM kb4_rule_approvals
		 WHERE criterion_set = $1 AND criterion_id = $2`, criterionSet, criterionID)
	if err != nil {
		return nil, fmt.Errorf("query approvals %s/%s: %w", criterionSet, criterionID, err)
	}
	defer rows.Close()
	out := []ApprovalEntry{}
	for rows.Next() {
		var a ApprovalEntry
		if err := rows.Scan(&a.ReviewerRole, &a.ReviewerID); err != nil {
			return nil, fmt.Errorf("scan approval: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// requiredRoles are the L6 dual-approval roles that must each appear
// at least once in the approval entries for a rule.
var requiredRoles = []string{"CLINICAL_REVIEWER", "MEDICAL_DIRECTOR"}

// VerifyChain is the entry point. It iterates every criterion set in
// CriterionSetsInScope and applies VerifyRuleSignature + dual-approval
// checks. The first failure is returned with the failing
// (criterion_set, criterion_id) prefixed.
func VerifyChain(ctx context.Context, store ChainStore, pubKey ed25519.PublicKey) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid Ed25519 public key length: got %d, want %d", len(pubKey), ed25519.PublicKeySize)
	}
	for _, cs := range CriterionSetsInScope {
		rules, err := store.ListSignedRules(ctx, cs)
		if err != nil {
			return fmt.Errorf("[%s]: %w", cs, err)
		}
		if len(rules) == 0 {
			return fmt.Errorf("[%s]: no signed rules found — chain incomplete", cs)
		}
		for _, r := range rules {
			if err := VerifyRuleSignature(r, pubKey); err != nil {
				return fmt.Errorf("[%s/%s]: signature failure: %w", r.CriterionSet, r.CriterionID, err)
			}
			approvals, err := store.ListApprovals(ctx, r.CriterionSet, r.CriterionID)
			if err != nil {
				return fmt.Errorf("[%s/%s]: %w", r.CriterionSet, r.CriterionID, err)
			}
			if err := VerifyDualApproval(approvals); err != nil {
				return fmt.Errorf("[%s/%s]: dual-approval failure: %w", r.CriterionSet, r.CriterionID, err)
			}
		}
	}
	return nil
}

// VerifyRuleSignature checks that the recorded signature verifies
// against the public key over the recorded payload digest, and that
// the signed payload digest matches the current content_sha column
// (no drift since signing).
func VerifyRuleSignature(r SignedRule, pubKey ed25519.PublicKey) error {
	if r.ContentSHA == "" {
		return errors.New("missing content_sha")
	}
	if r.SignedSHA == "" {
		return errors.New("missing signed_content_sha")
	}
	if r.ContentSHA != r.SignedSHA {
		return fmt.Errorf("content_sha drift: stored=%s signed=%s", r.ContentSHA, r.SignedSHA)
	}
	if len(r.Signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: %d", len(r.Signature))
	}
	digest, err := hex.DecodeString(r.ContentSHA)
	if err != nil {
		return fmt.Errorf("content_sha not valid hex: %w", err)
	}
	if len(digest) != sha256.Size {
		return fmt.Errorf("content_sha decoded length %d, want %d", len(digest), sha256.Size)
	}
	if !ed25519.Verify(pubKey, digest, r.Signature) {
		return errors.New("Ed25519 signature did not verify")
	}
	return nil
}

// VerifyDualApproval checks that every required role appears at least
// once in the approval entries.
func VerifyDualApproval(entries []ApprovalEntry) error {
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.ReviewerRole] = true
	}
	for _, role := range requiredRoles {
		if !seen[role] {
			return fmt.Errorf("missing required approval role %q (have %v)", role, roleSet(entries))
		}
	}
	return nil
}

func roleSet(entries []ApprovalEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ReviewerRole)
	}
	return out
}
