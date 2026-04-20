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

// InMemoryLedger is the Sprint 1 governance ledger. HMAC-SHA256 chain; each entry's
// hash is HMAC(key, prior_hash || entry_type || subject_id || payload_json || sequence || timestamp).
// Sprint 2 replaces storage with PostgreSQL persistence and adds Ed25519 per-entry signatures.
type InMemoryLedger struct {
	mu      sync.Mutex
	key     []byte
	entries []models.LedgerEntry
}

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

func NewInMemoryLedger(hmacKey []byte) *InMemoryLedger {
	if len(hmacKey) == 0 {
		hmacKey = []byte("sprint1-default-do-not-use-in-prod")
	}
	return &InMemoryLedger{key: hmacKey}
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

func (l *InMemoryLedger) computeEntryHash(prior, entryType, subjectID, payloadJSON string, seq int64, ts time.Time) string {
	m := hmac.New(sha256.New, l.key)
	fmt.Fprintf(m, "%s|%s|%s|%s|%d|%s", prior, entryType, subjectID, payloadJSON, seq, ts.Format(time.RFC3339Nano))
	return hex.EncodeToString(m.Sum(nil))
}
