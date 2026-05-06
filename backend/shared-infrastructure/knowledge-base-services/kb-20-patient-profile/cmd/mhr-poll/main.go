// Command mhr-poll is the Wave 3 SKELETON CLI for polling the MHR
// SOAP/CDA gateway for new pathology documents and ingesting them
// into the kb-20 substrate.
//
// Usage (planned):
//
//	mhr-poll --ihi <16-digit> --since <RFC3339> [--mode soap_cda|fhir_gateway|dual]
//
// Wave 3 status: the CLI structure exists, flag parsing works, and
// the wiring to MHRSOAPClient + MHRFHIRClient is in place — but the
// production clients are stub implementations that return
// ErrMHRWiringDeferred / ErrMHRFHIRWiringDeferred. Running this CLI
// against a real MHR endpoint will panic with a deferred-wiring
// message; running it in dry-run mode against the stubs surfaces
// the expected deferred error and exits non-zero. V1 will replace
// the stub constructors with production implementations and the
// CLI will run unchanged.
//
// See docs/adr/2026-05-06-mhr-integration-strategy.md for the
// deferral rationale and V1 sequencing.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cardiofit/shared/v2_substrate/ingestion"
)

func main() {
	var (
		ihi      = flag.String("ihi", "", "16-digit Individual Healthcare Identifier (required)")
		sinceRaw = flag.String("since", "", "RFC3339 watermark; only documents authored on/after this point are fetched (required)")
		mode     = flag.String("mode", "soap_cda", "gateway mode: soap_cda | fhir_gateway | dual")
	)
	flag.Parse()

	if *ihi == "" || *sinceRaw == "" {
		fmt.Fprintln(os.Stderr, "mhr-poll: --ihi and --since are required")
		flag.Usage()
		os.Exit(2)
	}
	since, err := time.Parse(time.RFC3339, *sinceRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mhr-poll: invalid --since: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()

	soapClient := ingestion.NewStubMHRSOAPClient()
	fhirClient := ingestion.NewStubMHRFHIRClient()

	// Wave 3 skeleton: invoke the configured client(s); they return
	// ErrMHRWiringDeferred / ErrMHRFHIRWiringDeferred. V1 replaces
	// the stub constructors with production wiring.
	switch *mode {
	case "soap_cda":
		_, err := soapClient.GetPathologyDocumentList(ctx, *ihi, since)
		emitDeferredAndPanicIfReal(err, "soap_cda")
	case "fhir_gateway":
		_, err := fhirClient.GetDiagnosticReports(ctx, *ihi, since)
		emitDeferredAndPanicIfReal(err, "fhir_gateway")
	case "dual":
		_, errSoap := soapClient.GetPathologyDocumentList(ctx, *ihi, since)
		_, errFhir := fhirClient.GetDiagnosticReports(ctx, *ihi, since)
		emitDeferredAndPanicIfReal(errSoap, "soap_cda")
		emitDeferredAndPanicIfReal(errFhir, "fhir_gateway")
	default:
		fmt.Fprintf(os.Stderr, "mhr-poll: unknown --mode %q (expected soap_cda | fhir_gateway | dual)\n", *mode)
		os.Exit(2)
	}

	// Reaching here means at least one path didn't return a deferred-
	// wiring error — i.e. production wiring landed since this CLI was
	// last reviewed. Update the CLI to walk the real result set.
	fmt.Fprintln(os.Stderr, "mhr-poll: production wiring detected — CLI needs the V1 result-walker implementation")
	os.Exit(2)
}

// emitDeferredAndPanicIfReal prints the deferred-wiring banner and
// returns when err is the expected deferred-wiring sentinel; panics
// otherwise. The panic is intentional: a non-deferred error from the
// stub means the constructor was swapped with a partial production
// implementation that still has bugs — fail loudly rather than mask.
func emitDeferredAndPanicIfReal(err error, mode string) {
	if errors.Is(err, ingestion.ErrMHRWiringDeferred) || errors.Is(err, ingestion.ErrMHRFHIRWiringDeferred) {
		fmt.Fprintf(os.Stderr, "mhr-poll[%s]: production wiring deferred to V1 (%v)\n", mode, err)
		return
	}
	panic(fmt.Sprintf("mhr-poll[%s]: unexpected error from stub client (production wiring partially landed?): %v", mode, err))
}
