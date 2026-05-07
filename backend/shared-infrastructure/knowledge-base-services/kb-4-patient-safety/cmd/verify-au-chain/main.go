// Command verify-au-chain is the CLI wrapper around the L6 governance
// chain verifier defined in
// kb-4-patient-safety/internal/governance/verify_au_chain.go.
//
// Pre-Wave Task 2 of the Layer 3 v2 rule encoding plan.
//
// Usage:
//   verify-au-chain  [-pubkey-hex <hex>]  [-pubkey-env KB4_VERIFY_PUBKEY]
//
// Environment:
//   KB4_DATABASE_URL    PostgreSQL DSN for the kb-4 database (required
//                       unless -dry-run is set).
//   KB4_VERIFY_PUBKEY   Hex-encoded Ed25519 verification public key.
//                       Overridden by -pubkey-hex if both supplied.
//
// Exit codes:
//   0  every signed rule in every criterion set verifies cleanly
//      under the dual-approval contract.
//   1  one or more failures; failing (criterion_set, criterion_id) is
//      printed to stderr.
//   2  configuration / connectivity error before any rule was checked.
//
// See the runbook at:
//   claudedocs/audits/2026-05-PreWave-l6-governance-verification.md
package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kb-patient-safety/internal/governance"
)

func main() {
	pubkeyHex := flag.String("pubkey-hex", "", "Ed25519 verification public key (hex)")
	pubkeyEnv := flag.String("pubkey-env", "KB4_VERIFY_PUBKEY", "env var to read pubkey from when -pubkey-hex is empty")
	dbEnv := flag.String("db-env", "KB4_DATABASE_URL", "env var holding the kb-4 PG DSN")
	timeoutSec := flag.Int("timeout", 60, "overall verification timeout, seconds")
	dryRun := flag.Bool("dry-run", false, "do not connect to DB; only validate the public key parses")
	flag.Parse()

	pubKey, err := loadPubKey(*pubkeyHex, *pubkeyEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify-au-chain: pubkey load: %v\n", err)
		os.Exit(2)
	}
	if *dryRun {
		fmt.Println("verify-au-chain: dry-run OK — pubkey parsed.")
		return
	}

	dsn := os.Getenv(*dbEnv)
	if dsn == "" {
		fmt.Fprintf(os.Stderr, "verify-au-chain: %s is not set\n", *dbEnv)
		os.Exit(2)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify-au-chain: open db: %v\n", err)
		os.Exit(2)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "verify-au-chain: ping db: %v\n", err)
		os.Exit(2)
	}
	store := &governance.SQLChainStore{DB: db}
	if err := governance.VerifyChain(ctx, store, pubKey); err != nil {
		fmt.Fprintf(os.Stderr, "verify-au-chain: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("verify-au-chain: OK — all criterion sets verified.")
}

func loadPubKey(hexFlag, envName string) (ed25519.PublicKey, error) {
	src := hexFlag
	if src == "" {
		src = os.Getenv(envName)
	}
	if src == "" {
		return nil, errors.New("no public key supplied via -pubkey-hex or env")
	}
	raw, err := hex.DecodeString(src)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("decoded length %d, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}
