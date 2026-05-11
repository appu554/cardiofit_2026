// Package structural — no_surveillance_reader_test.go is the load-bearing
// safeguard enforcing the Phase 1 LOG-ONLY commitment for pharmacist
// cognitive escalation events per S2 Adaptive Cognition Architectural
// Commitment Addendum Part 5.5.
//
// The test scans the audit package source code for any function or SQL
// pattern that would constitute a "give me pharmacist X's escalation
// patterns" reader. Such a reader would be surveillance per Addendum
// Part 5.2 and is forbidden until Phase 4 (≥12 months evidence + ESC
// approval + external clinical informatics review + pharmacist self-
// visibility operational).
//
// If anyone adds a surveillance reader without those preconditions, this
// test fails the build. The test cannot be bypassed without explicit
// removal of the test itself, which would be visible in code review.
package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenIdentifiers are function-name / type-name fragments that
// constitute a surveillance reader. The list is exhaustive of the
// patterns Addendum Part 5.2 prohibits and Task 7 enumerates.
var forbiddenIdentifiers = []string{
	"GetEscalationPattern",
	"GetEscalationPatterns",
	"QueryEscalationsForPharmacist",
	"EscalationsByPharmacist",
	"EscalationsForPharmacist",
	"ListEscalationsByPharmacist",
	"FetchEscalationPatterns",
	"PharmacistEscalationProfile",
	"PharmacistEscalationHistory",
}

// forbiddenSQLFragments are SQL substrings that would constitute the
// database-layer equivalent of a surveillance reader. The combination
// "FROM s2_audit_events" + "cognitive_escalation" + "pharmacist_id"
// in proximity is the canonical surveillance read.
var forbiddenSQLFragments = []string{
	"FROM s2_audit_events WHERE event_type = 'cognitive_escalation' AND pharmacist_id",
	"from s2_audit_events where event_type = 'cognitive_escalation' and pharmacist_id",
	"WHERE event_type = 'cognitive_escalation' AND pharmacist_id",
}

// auditPackagePath resolves to the audit package directory relative to
// the test binary's working directory. go test runs tests from the
// package directory, so we navigate up two levels (tests/structural →
// s2-aggregator) then down into internal/audit.
func auditPackagePath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	// wd is .../s2-aggregator/tests/structural
	root := filepath.Dir(filepath.Dir(wd))
	pkg := filepath.Join(root, "internal", "audit")
	if _, err := os.Stat(pkg); err != nil {
		t.Fatalf("audit package not found at %s: %v", pkg, err)
	}
	return pkg
}

// scanAuditPackage returns the concatenated source contents of every
// non-test .go file in the audit package. Test files are excluded so
// test helpers in this package's *_test.go files do not trip the gate.
func scanAuditPackage(t *testing.T) string {
	t.Helper()
	pkg := auditPackagePath(t)
	entries, err := os.ReadDir(pkg)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", pkg, err)
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(pkg, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		b.WriteString("// FILE: ")
		b.WriteString(name)
		b.WriteString("\n")
		b.Write(data)
		b.WriteString("\n")
	}
	return b.String()
}

// declaredIdentifiers parses every non-test .go file in the audit
// package via go/ast and returns the set of declared identifier names
// (functions, methods, types, type-method receivers' methods). Comments
// and doc-strings are intentionally excluded — the no-surveillance
// commitment is about what the package EXPORTS, and citing a
// forbidden-pattern name in a doc-comment that explains why it MUST
// NOT exist is the correct documentation discipline.
func declaredIdentifiers(t *testing.T) map[string]string {
	t.Helper()
	pkg := auditPackagePath(t)
	entries, err := os.ReadDir(pkg)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", pkg, err)
	}
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fullPath := filepath.Join(pkg, name)
		f, err := parser.ParseFile(fset, fullPath, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", fullPath, err)
		}
		for _, d := range f.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				out[decl.Name.Name] = name
			case *ast.GenDecl:
				for _, s := range decl.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok {
						out[ts.Name.Name] = name
					}
				}
			}
		}
	}
	return out
}

// TestNoSurveillanceReader_FunctionNames asserts that the audit package
// declares zero functions / types matching the surveillance-reader
// patterns. This is the load-bearing safeguard for Addendum Part 5.5's
// Phase 1 log-only commitment.
//
// The test parses Go declarations via go/ast (not raw text grep) so the
// MAY-NOT-USE doc-comment in escalation_event.go — which names the
// forbidden patterns precisely so reviewers know what to look for — is
// allowed.
//
// If this test fails, do NOT silence it. The failure indicates someone
// has added a declaration that constitutes surveillance per Addendum
// Part 5.2. Removal requires explicit Ethics Steering Committee
// approval per Part 5.5.
func TestNoSurveillanceReader_FunctionNames(t *testing.T) {
	decls := declaredIdentifiers(t)
	for _, forbidden := range forbiddenIdentifiers {
		for name, file := range decls {
			if strings.Contains(name, forbidden) {
				t.Errorf(
					"FORBIDDEN: audit package declares identifier %q in %s (matches forbidden pattern %q).\n"+
						"This would constitute a surveillance reader per Addendum Part 5.2.\n"+
						"Phase 1 commitment per Addendum Part 5.5 is LOG-ONLY for cognitive escalation.\n"+
						"Adding this requires explicit Ethics Steering Committee approval.",
					name, file, forbidden)
			}
		}
	}
}

// TestNoSurveillanceReader_SQLFragments asserts the audit package
// contains no SQL that queries s2_audit_events for cognitive_escalation
// rows filtered by pharmacist_id. This is the database-layer mirror of
// the function-name gate above.
func TestNoSurveillanceReader_SQLFragments(t *testing.T) {
	src := scanAuditPackage(t)
	for _, frag := range forbiddenSQLFragments {
		if strings.Contains(src, frag) {
			t.Errorf(
				"FORBIDDEN: audit package contains SQL fragment %q.\n"+
					"This is the database-layer equivalent of a surveillance reader per Addendum Part 5.2.\n"+
					"Per-pharmacist cognitive-escalation queries are forbidden in Phase 1 (Addendum Part 5.5).",
				frag)
		}
	}
}

// TestNoSurveillanceReader_MayNotUseHeaderPresent asserts the
// escalation_event.go file carries the MAY-NOT-USE doc-comment block
// citing Addendum Part 5.2. The header is the in-source declaration of
// the Phase 1 commitment; removing it without process should fail the
// build.
func TestNoSurveillanceReader_MayNotUseHeaderPresent(t *testing.T) {
	src := scanAuditPackage(t)
	requiredFragments := []string{
		"MAY-NOT-USE",
		"Addendum Part 5.2",
		"Performance evaluation",
		"Productivity surveillance",
		"Comparative pharmacist ranking",
		"Decisions affecting pharmacist employment",
		"Differential treatment of pharmacists",
	}
	for _, f := range requiredFragments {
		if !strings.Contains(src, f) {
			t.Errorf("audit package missing required MAY-NOT-USE header fragment %q (Addendum Part 5.2 commitment)", f)
		}
	}
}
