// Package transform provides versioned transformation scripts for authority data.
//
// This addresses Gap 2: Manual Transformations Are Not Versioned
//
// Each transformation script must:
//   1. Have a unique version identifier (e.g., "lactmed_v2026_01")
//   2. Be immutable once used in production
//   3. Have its hash recorded in the MANIFEST.yaml
//   4. Produce deterministic output from the same input
//
// When source data formats change:
//   1. Create a NEW transform script (e.g., "lactmed_v2026_02")
//   2. Update MANIFEST.yaml to reference the new script
//   3. Re-process all data with the new transform
//   4. Keep old scripts for audit reproducibility
package transform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

// =============================================================================
// TRANSFORM REGISTRY
// =============================================================================

// TransformFunc is the signature for all transform functions
type TransformFunc func(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error)

// TransformResult contains the result of a transformation
type TransformResult struct {
	// Identification
	TransformName    string    `json:"transform_name"`
	TransformVersion string    `json:"transform_version"`

	// Input metadata
	InputChecksum    string    `json:"input_checksum"`
	InputRecords     int       `json:"input_records"`

	// Output metadata
	OutputChecksum   string    `json:"output_checksum"`
	OutputRecords    int       `json:"output_records"`

	// Processing stats
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	RecordsSkipped   int       `json:"records_skipped"`
	RecordsFailed    int       `json:"records_failed"`
	Warnings         []string  `json:"warnings,omitempty"`
	Errors           []string  `json:"errors,omitempty"`

	// For audit
	Success          bool      `json:"success"`
}

// TransformMetadata describes a transform script
type TransformMetadata struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	SourceAuthority  string   `json:"source_authority"`
	SourceFormat     string   `json:"source_format"`
	OutputFormat     string   `json:"output_format"`
	RequiredColumns  []string `json:"required_columns"`
	CreatedAt        string   `json:"created_at"`
	Author           string   `json:"author"`
}

// Registry holds all registered transforms
var Registry = make(map[string]RegisteredTransform)

// RegisteredTransform holds a transform and its metadata
type RegisteredTransform struct {
	Metadata  TransformMetadata
	Transform TransformFunc
}

// Register registers a transform in the registry
func Register(name string, metadata TransformMetadata, fn TransformFunc) {
	Registry[name] = RegisteredTransform{
		Metadata:  metadata,
		Transform: fn,
	}
}

// Get retrieves a transform by name
func Get(name string) (RegisteredTransform, bool) {
	t, ok := Registry[name]
	return t, ok
}

// =============================================================================
// TRANSFORM UTILITIES
// =============================================================================

// ComputeChecksum computes SHA-256 checksum of a file
func ComputeChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeReaderChecksum computes SHA-256 checksum from a reader
func ComputeReaderChecksum(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// =============================================================================
// SAMPLE TRANSFORMS (Templates)
// =============================================================================

func init() {
	// Register sample transforms - actual implementations in separate files

	Register("lactmed_v2026_01", TransformMetadata{
		Name:            "lactmed_v2026_01",
		Version:         "1.0.0",
		Description:     "Transform LactMed XML to JSON format",
		SourceAuthority: "lactmed",
		SourceFormat:    "XML",
		OutputFormat:    "JSON",
		RequiredColumns: []string{"drug_name", "rid_percentage", "aap_rating"},
		CreatedAt:       "2026-01-25",
		Author:          "cardiofit-clinical-team",
	}, lactmedTransformV2026_01)

	Register("cpic_v2026_q1", TransformMetadata{
		Name:            "cpic_v2026_q1",
		Version:         "1.0.0",
		Description:     "Transform CPIC API response to internal format",
		SourceAuthority: "cpic",
		SourceFormat:    "JSON_API",
		OutputFormat:    "JSON",
		RequiredColumns: []string{"gene", "drug_rxcui", "phenotype", "recommendation"},
		CreatedAt:       "2026-01-25",
		Author:          "cardiofit-clinical-team",
	}, cpicTransformV2026_Q1)

	Register("crediblemeds_v2026_01", TransformMetadata{
		Name:            "crediblemeds_v2026_01",
		Version:         "1.0.0",
		Description:     "Transform CredibleMeds CSV to internal format",
		SourceAuthority: "crediblemeds",
		SourceFormat:    "CSV",
		OutputFormat:    "JSON",
		RequiredColumns: []string{"drug_name", "qt_risk_category"},
		CreatedAt:       "2026-01-25",
		Author:          "cardiofit-clinical-team",
	}, crediblemedsTransformV2026_01)

	Register("livertox_v2026_01", TransformMetadata{
		Name:            "livertox_v2026_01",
		Version:         "1.0.0",
		Description:     "Transform LiverTox XML to internal format",
		SourceAuthority: "livertox",
		SourceFormat:    "XML",
		OutputFormat:    "JSON",
		RequiredColumns: []string{"drug_name", "likelihood_score", "pattern"},
		CreatedAt:       "2026-01-25",
		Author:          "cardiofit-clinical-team",
	}, livertoxTransformV2026_01)

	Register("drugbank_v5_1_11", TransformMetadata{
		Name:            "drugbank_v5_1_11",
		Version:         "1.0.0",
		Description:     "Transform DrugBank XML to internal format",
		SourceAuthority: "drugbank",
		SourceFormat:    "XML",
		OutputFormat:    "JSON",
		RequiredColumns: []string{"drugbank_id", "name", "rxcui"},
		CreatedAt:       "2026-01-25",
		Author:          "cardiofit-clinical-team",
	}, drugbankTransformV5_1_11)
}

// =============================================================================
// TRANSFORM STUBS (Implementations in authority-specific files)
// =============================================================================

func lactmedTransformV2026_01(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error) {
	// TODO: Implement actual LactMed XML parsing
	return &TransformResult{
		TransformName:    "lactmed_v2026_01",
		TransformVersion: "1.0.0",
		Success:          true,
	}, fmt.Errorf("not implemented - use lactmed/ingest.go directly")
}

func cpicTransformV2026_Q1(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error) {
	// TODO: Implement actual CPIC API response parsing
	return &TransformResult{
		TransformName:    "cpic_v2026_q1",
		TransformVersion: "1.0.0",
		Success:          true,
	}, fmt.Errorf("not implemented - use cpic/client.go directly")
}

func crediblemedsTransformV2026_01(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error) {
	// TODO: Implement actual CredibleMeds CSV parsing
	return &TransformResult{
		TransformName:    "crediblemeds_v2026_01",
		TransformVersion: "1.0.0",
		Success:          true,
	}, fmt.Errorf("not implemented - use crediblemeds/client.go directly")
}

func livertoxTransformV2026_01(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error) {
	// TODO: Implement actual LiverTox XML parsing
	return &TransformResult{
		TransformName:    "livertox_v2026_01",
		TransformVersion: "1.0.0",
		Success:          true,
	}, fmt.Errorf("not implemented - use livertox/ingest.go directly")
}

func drugbankTransformV5_1_11(ctx context.Context, input io.Reader, output io.Writer) (*TransformResult, error) {
	// TODO: Implement actual DrugBank XML parsing
	return &TransformResult{
		TransformName:    "drugbank_v5_1_11",
		TransformVersion: "1.0.0",
		Success:          true,
	}, fmt.Errorf("not implemented - use drugbank/loader.go directly")
}
