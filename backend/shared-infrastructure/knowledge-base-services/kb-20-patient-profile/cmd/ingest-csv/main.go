// Command ingest-csv runs a single CSV ingestion pass against a kb-20 server.
//
// Usage:
//
//	ingest-csv --file <path.csv> --facility <uuid> --source <label>
//	          [--kb20-base-url http://localhost:8131] [--dry-run]
//
// Exits 0 if at least one row was ingested and no rows errored.
// Exits 1 on any error or zero ingested rows.
//
// The matcher used here is a thin adapter over KB20Client.MatchIdentity so
// the CLI shares the kb-20 service's identity decisions. Normaliser uses
// no-op AMT/SNOMED lookups for MVP — kb-7-terminology integration is a
// future increment per Layer 2 plan §3.2.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/client"
	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/ingestion"
)

type clientMatcher struct {
	c *client.KB20Client
}

func (m *clientMatcher) Match(ctx context.Context, in identity.IncomingIdentifier) (identity.MatchResult, error) {
	res, err := m.c.MatchIdentity(ctx, in)
	if err != nil || res == nil {
		return identity.MatchResult{Confidence: identity.ConfidenceNone, Path: identity.MatchPathNoMatch, RequiresReview: true}, err
	}
	return res.Match, nil
}

type stubAMT struct{}

func (stubAMT) LookupByName(_ context.Context, _, _, _ string) (string, float64, error) {
	return "", 0, nil
}

type stubSNOMED struct{}

func (stubSNOMED) LookupIndication(_ context.Context, _ string) (string, float64, error) {
	return "", 0, nil
}

func main() {
	var (
		filePath    = flag.String("file", "", "path to CSV file (required)")
		facilityRaw = flag.String("facility", "", "facility UUID (required)")
		source      = flag.String("source", "", "source label written to EvidenceTrace (required)")
		baseURL     = flag.String("kb20-base-url", "http://localhost:8131", "KB20 service base URL")
		dryRun      = flag.Bool("dry-run", false, "parse + match + normalise but do not write")
	)
	flag.Parse()

	if *filePath == "" || *facilityRaw == "" || *source == "" {
		fmt.Fprintln(os.Stderr, "ingest-csv: --file, --facility, --source are required")
		flag.Usage()
		os.Exit(2)
	}
	facilityID, err := uuid.Parse(*facilityRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest-csv: invalid --facility UUID: %v\n", err)
		os.Exit(2)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest-csv: open %s: %v\n", *filePath, err)
		os.Exit(1)
	}
	defer f.Close()

	kb20 := client.NewKB20Client(*baseURL)

	cfg := ingestion.RunnerConfig{
		FacilityID:  facilityID,
		SourceLabel: *source,
		Client:      kb20,
		Matcher:     &clientMatcher{c: kb20},
		Normaliser:  &ingestion.Normaliser{AMT: stubAMT{}, SNOMED: stubSNOMED{}},
		DryRun:      *dryRun,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	res, err := ingestion.Run(ctx, f, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest-csv: run failed: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))

	if res.RowsErrored > 0 || res.RowsIngested == 0 {
		os.Exit(1)
	}
}
