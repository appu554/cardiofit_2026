package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

// InMemoryLedger is the Sprint 1 governance ledger. HMAC-SHA256 chain over
// length-prefixed fields: each entry's hash covers prior_hash, entry_type,
// subject_id, payload_json (all length-prefixed via computeEntryHash), plus
// the unprefixed sequence and RFC3339Nano timestamp. See computeEntryHash
// for the exact encoding.
// Sprint 2b replaces storage with PostgreSQL persistence and adds Ed25519
// per-entry signatures on top of the HMAC chain.
type InMemoryLedger struct {
	mu            sync.Mutex
	key           []byte
	entries       []models.LedgerEntry
	seededLastSeq int64
	seeded        bool
}

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

func NewInMemoryLedger(hmacKey []byte) *InMemoryLedger {
	if len(hmacKey) == 0 {
		hmacKey = []byte("sprint1-default-do-not-use-in-prod")
	}
	return &InMemoryLedger{key: hmacKey}
}

// SeedSequence primes the in-memory counter so the next AppendEntry produces
// sequence = startSeq. Call once at startup after restoring from a persistent
// store (DB) to avoid collisions with already-persisted LedgerEntry rows.
//
// Idempotent: if already seeded, or if live entries exist, this is a no-op.
// This prevents a second call (e.g., during a test helper or a botched
// restart sequence) from silently overwriting the first seed.
//
// Note: this method seeds only the sequence counter. The first entry's
// PriorHash will be the genesis hash regardless, which means each process
// lifetime starts a new HMAC chain segment. Sprint 2b's durable ledger
// will add a companion SeedPriorHash method for full cross-process chain
// continuity.
func (l *InMemoryLedger) SeedSequence(startSeq int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.seeded || len(l.entries) > 0 {
		return
	}
	l.seededLastSeq = startSeq - 1
	l.seeded = true
}

// AppendEntry appends a new entry and returns it with EntryHash and Sequence set.
func (l *InMemoryLedger) AppendEntry(entryType, subjectID, payloadJSON string) (models.LedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prior := genesisHash
	seq := int64(0)
	if n := len(l.entries); n > 0 {
		prior = l.entries[n-1].EntryHash
		seq = l.entries[n-1].Sequence + 1
	} else if l.seeded {
		seq = l.seededLastSeq + 1
	}
	now := time.Now().UTC()
	hash := l.computeEntryHash(prior, entryType, subjectID, payloadJSON, seq, now)

	entry := models.LedgerEntry{
		ID:          uuid.New(),
		Sequence:    seq,
		EntryType:   entryType,
		SubjectID:   subjectID,
		PayloadJSON: payloadJSON,
		PriorHash:   prior,
		EntryHash:   hash,
		CreatedAt:   now,
	}
	l.entries = append(l.entries, entry)
	return entry, nil
}

// VerifyChain walks every entry and recomputes its hash against the recorded prior_hash.
// Returns (true, -1, nil) if valid; (false, first_broken_index, nil) if tampered.
func (l *InMemoryLedger) VerifyChain() (bool, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ok, idx := l.verifyChainLocked()
	return ok, idx, nil
}

// tamperForTest mutates an entry's payload without updating its hash — used by the
// tamper-detection test only. Never call this in production code paths.
func (l *InMemoryLedger) tamperForTest(index int, newPayload string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if index >= 0 && index < len(l.entries) {
		l.entries[index].PayloadJSON = newPayload
	}
}

// Entries returns a copy of the current ledger entries.
func (l *InMemoryLedger) Entries() []models.LedgerEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]models.LedgerEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Snapshot returns a consistent view of the ledger: a defensive copy of all
// entries PLUS the chain-validity status computed over the same snapshot,
// under a single mutex acquisition. Use this in preference to separate
// Entries() + VerifyChain() calls when the caller needs a coherent view
// (e.g., the governance HTTP endpoint).
func (l *InMemoryLedger) Snapshot() (entries []models.LedgerEntry, chainValid bool, brokenIdx int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entries = make([]models.LedgerEntry, len(l.entries))
	copy(entries, l.entries)
	chainValid, brokenIdx = l.verifyChainLocked()
	return
}

// verifyChainLocked is the body of VerifyChain without mutex acquisition.
// Callers must already hold l.mu.
func (l *InMemoryLedger) verifyChainLocked() (bool, int) {
	prior := genesisHash
	for i, e := range l.entries {
		expected := l.computeEntryHash(prior, e.EntryType, e.SubjectID, e.PayloadJSON, e.Sequence, e.CreatedAt)
		if !hmac.Equal([]byte(expected), []byte(e.EntryHash)) || e.PriorHash != prior {
			return false, i
		}
		prior = e.EntryHash
	}
	return true, -1
}

// computeEntryHash produces an HMAC-SHA256 over length-prefixed fields.
// Each variable-length field is written as "<byte_length>:<bytes>|" so no
// payload value can collide with a neighbouring field's contents. The length
// prefix is Go's len(s) — UTF-8 byte count, NOT rune count. Cross-language
// verifiers (Sprint 2b+) must use byte length, not character count.
// Fixed-width fields (seq, timestamp) are written without length prefix
// because their structure is already unambiguous.
func (l *InMemoryLedger) computeEntryHash(prior, entryType, subjectID, payloadJSON string, seq int64, ts time.Time) string {
	m := hmac.New(sha256.New, l.key)
	writeLP := func(s string) {
		fmt.Fprintf(m, "%d:%s|", len(s), s)
	}
	writeLP(prior)
	writeLP(entryType)
	writeLP(subjectID)
	writeLP(payloadJSON)
	fmt.Fprintf(m, "%d|%s", seq, ts.Format(time.RFC3339Nano))
	return hex.EncodeToString(m.Sum(nil))
}
