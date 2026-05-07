// Package parser is a thin convenience over dsl.ParseRule for ingesting
// directories of bundled ScopeRule YAML files.
package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"kb-scope-rules/internal/dsl"
)

// LoadedRule pairs a parsed ScopeRule with its on-disk path and raw YAML.
type LoadedRule struct {
	Path        string
	PayloadYAML []byte
	Rule        *dsl.ScopeRule
}

// LoadDir walks a directory recursively and returns one LoadedRule per
// .yaml / .yml file. Files whose YAML fails to parse are returned in
// errs alongside the successfully-parsed rules.
func LoadDir(root string) ([]LoadedRule, []error) {
	var loaded []LoadedRule
	var errs []error
	if _, err := os.Stat(root); err != nil {
		return nil, []error{fmt.Errorf("LoadDir: %w", err)}
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", path, err))
			return nil
		}
		rule, err := dsl.ParseRule(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", path, err))
			return nil
		}
		loaded = append(loaded, LoadedRule{Path: path, PayloadYAML: data, Rule: rule})
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return loaded, errs
}
