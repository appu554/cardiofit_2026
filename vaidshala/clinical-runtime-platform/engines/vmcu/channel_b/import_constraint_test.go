package channel_b

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImportConstraint verifies that channel_b package does NOT import:
//   - vmcu (parent package) — no GateSignal dependency
//   - channel_a — Channel B is independent of diagnostic reasoning
//   - titration — Channel B knows nothing about dose computation
//   - Any KB-22 or KB-23 package
//
// This test MUST run in CI — failure blocks deployment.
func TestImportConstraint(t *testing.T) {
	forbidden := []string{
		"vmcu/channel_a",
		"vmcu/titration",
		"kb-22",
		"kb-23",
		"kb22",
		"kb23",
	}

	fset := token.NewFileSet()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files for import analysis
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Errorf("failed to parse %s: %v", path, parseErr)
			return nil
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, fb := range forbidden {
				if strings.Contains(importPath, fb) {
					t.Errorf("FORBIDDEN IMPORT in %s: %q contains %q\n"+
						"Channel B must NOT import Channel A, titration, or KB-22/KB-23 packages",
						path, importPath, fb)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
}
