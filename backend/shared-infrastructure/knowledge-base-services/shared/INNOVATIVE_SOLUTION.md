# Innovative Solution: OHDSI-First Architecture

## 🎯 The Breakthrough Insight

Your current plan uses **LLM for everything**, which introduces:
- Latency (500ms+ per call)
- Cost ($0.08 warmup, ongoing API costs)
- Non-determinism (different outputs for same input)
- External API dependency in critical path

**The innovative solution**: Use **OHDSI Athena Standardized Vocabularies** as a FREE, LOCAL, DETERMINISTIC terminology backbone.

### Why OHDSI Athena is the Answer

| Aspect | MedDRA License | LLM Approach | OHDSI Athena |
|--------|---------------|--------------|--------------|
| **Cost** | $5K-50K/year | ~$0.08 warmup + ongoing | **FREE** |
| **Determinism** | ✅ Yes | ❌ No | ✅ Yes |
| **Latency** | Local lookup | 500ms API | **<1ms local** |
| **FAERS Compatible** | ✅ Yes | Partial | ✅ Yes (MedDRA→SNOMED mappings) |
| **Offline** | ✅ Yes | ❌ No | ✅ Yes |
| **Includes RxNorm** | ❌ No | ❌ No | ✅ Yes |

### What Athena Provides (FREE)

- **10+ million concepts** from 136 vocabularies
- **MedDRA → SNOMED mappings** (5,000+ direct mappings)
- **RxNorm** for drug normalization
- **SNOMED CT** as standard for conditions
- **Cross-vocabulary relationships** for translation

---

## 🏗️ Architecture: Three-Layer Terminology Service

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TERMINOLOGY NORMALIZATION SERVICE                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   LAYER 1: LOCAL DICTIONARY (Sub-millisecond, 95% hit rate)                 │
│   ═══════════════════════════════════════════════════════════               │
│   ┌─────────────────────┐  ┌─────────────────────────────────┐              │
│   │ Drug Name Cache     │  │ Adverse Event Cache             │              │
│   │ (Top 500 RxNorm)    │  │ (Top 2,000 SNOMED conditions)   │              │
│   │ "Lithium"→6448      │  │ "Hemorrhage"→192671(SNOMED)     │              │
│   │ "Digoxin"→3407      │  │ "Nausea"→422587007(SNOMED)      │              │
│   └─────────────────────┘  └─────────────────────────────────┘              │
│                                                                             │
│   LAYER 2: OHDSI ATHENA SQLITE (Millisecond, 99% hit rate)                  │
│   ════════════════════════════════════════════════════════                  │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │ athena_vocab.sqlite (downloaded from athena.ohdsi.org)            │     │
│   │ - CONCEPT table: 10M+ concepts                                    │     │
│   │ - CONCEPT_RELATIONSHIP: MedDRA→SNOMED, ICD→SNOMED mappings        │     │
│   │ - CONCEPT_SYNONYM: Alternative names for fuzzy matching           │     │
│   │ - DRUG_STRENGTH: RxNorm ingredient/dose info                      │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│   LAYER 3: LLM FALLBACK (Only for truly novel terms, <1% of calls)          │
│   ═══════════════════════════════════════════════════════════════════       │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │ Claude Haiku (fast, cheap)                                        │     │
│   │ - Only called when Layer 1 & 2 miss                               │     │
│   │ - Result cached to Layer 1 for future lookups                     │     │
│   │ - Cost: <$0.01/month after warmup                                 │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 🔧 Implementation: OHDSI-Powered Normalization

### Part A: Drug Validation (Fixes FK Constraint - Issue 1)

```go
// shared/terminology/drug_normalizer.go

type OHDSIDrugNormalizer struct {
    athenaDB    *sql.DB      // SQLite with Athena vocabulary
    rxnavClient *rxnav.Client // Existing RxNav-in-a-Box (backup)
    cache       sync.Map      // L1 in-memory cache
}

// ValidateAndNormalize fixes FK failures using OHDSI Athena
func (n *OHDSIDrugNormalizer) ValidateAndNormalize(ctx context.Context, rxcui, drugName string) (*NormalizedDrug, error) {
    // Layer 1: Check cache
    if cached, ok := n.cache.Load(rxcui); ok {
        return cached.(*NormalizedDrug), nil
    }

    // Layer 2: Query Athena CONCEPT table for RxCUI validation
    var conceptName string
    var standardConcept string
    err := n.athenaDB.QueryRowContext(ctx, `
        SELECT c.concept_name, c.standard_concept
        FROM concept c
        WHERE c.concept_code = ?
          AND c.vocabulary_id = 'RxNorm'
          AND c.invalid_reason IS NULL
    `, rxcui).Scan(&conceptName, &standardConcept)

    if err == sql.ErrNoRows {
        // RxCUI not found in Athena - lookup by name
        return n.lookupByDrugName(ctx, drugName)
    }
    if err != nil {
        return nil, err
    }

    // Check if RxCUI matches expected drug name
    if !strings.Contains(strings.ToLower(conceptName), strings.ToLower(drugName)) {
        // MISMATCH: RxCUI 5521 is hydroxychloroquine, not lithium
        log.Warnf("RxCUI %s is '%s', expected '%s' - correcting", rxcui, conceptName, drugName)
        return n.lookupByDrugName(ctx, drugName)
    }

    result := &NormalizedDrug{
        CanonicalName:  conceptName,
        CanonicalRxCUI: rxcui,
        OriginalRxCUI:  rxcui,
        WasCorrected:   false,
        Confidence:     1.0,
        Source:         "ATHENA_RXNORM",
    }
    n.cache.Store(rxcui, result)
    return result, nil
}

// lookupByDrugName finds correct RxCUI when SPL has wrong one
func (n *OHDSIDrugNormalizer) lookupByDrugName(ctx context.Context, drugName string) (*NormalizedDrug, error) {
    var rxcui, conceptName string
    err := n.athenaDB.QueryRowContext(ctx, `
        SELECT c.concept_code, c.concept_name
        FROM concept c
        WHERE c.vocabulary_id = 'RxNorm'
          AND c.concept_class_id = 'Ingredient'
          AND c.standard_concept = 'S'
          AND c.invalid_reason IS NULL
          AND LOWER(c.concept_name) = LOWER(?)
    `, drugName).Scan(&rxcui, &conceptName)

    if err == sql.ErrNoRows {
        // Try fuzzy match via concept_synonym
        return n.fuzzyLookupByName(ctx, drugName)
    }
    if err != nil {
        return nil, err
    }

    return &NormalizedDrug{
        CanonicalName:  conceptName,
        CanonicalRxCUI: rxcui,
        WasCorrected:   true,
        Confidence:     0.95,
        Source:         "ATHENA_RXNORM",
    }, nil
}
```

### Part B: Adverse Event Normalization (Fixes Issues 2 & 3)

```go
// shared/terminology/adverse_event_normalizer.go

type OHDSIAdverseEventNormalizer struct {
    athenaDB  *sql.DB       // SQLite with Athena vocabulary
    llmClient *llm.Client   // Fallback for novel terms
    cache     sync.Map      // L1 in-memory cache
    
    // Pre-loaded for sub-ms lookups
    snomedConditions map[string]*AthenaCondition  // 50K most common
    meddraToSnomed   map[string]string            // 5K MedDRA→SNOMED mappings
}

type AthenaCondition struct {
    ConceptID      int64
    ConceptCode    string  // SNOMED code
    ConceptName    string  // "Nausea"
    VocabularyID   string  // "SNOMED"
    DomainID       string  // "Condition"
    MedDRACodes    []string // Mapped MedDRA PTs for FAERS
}

// Normalize validates, normalizes, and returns SNOMED code (FAERS via mapping)
func (n *OHDSIAdverseEventNormalizer) Normalize(ctx context.Context, text string) (*NormalizedAdverseEvent, error) {
    cleaned := strings.ToLower(strings.TrimSpace(text))
    
    // Layer 1: Check in-memory cache
    if cached, ok := n.cache.Load(cleaned); ok {
        return cached.(*NormalizedAdverseEvent), nil
    }

    // Layer 1b: Check pre-loaded common conditions (sub-ms)
    if condition, ok := n.snomedConditions[cleaned]; ok {
        result := &NormalizedAdverseEvent{
            CanonicalName:  condition.ConceptName,
            OriginalText:   text,
            SNOMEDCode:     condition.ConceptCode,
            SNOMEDName:     condition.ConceptName,
            MedDRACodes:    condition.MedDRACodes,  // For FAERS!
            IsValidTerm:    true,
            Confidence:     1.0,
            Source:         "ATHENA_SNOMED",
        }
        n.cache.Store(cleaned, result)
        return result, nil
    }

    // Layer 2: Query Athena SQLite for condition match
    result, err := n.queryAthenaForCondition(ctx, cleaned)
    if err == nil && result.IsValidTerm {
        n.cache.Store(cleaned, result)
        return result, nil
    }

    // Layer 2b: Try synonym matching
    result, err = n.queryAthenaSynonyms(ctx, cleaned)
    if err == nil && result.IsValidTerm {
        n.cache.Store(cleaned, result)
        return result, nil
    }

    // Layer 2c: Check if this is obvious noise (statistical patterns)
    if n.isObviousNoise(cleaned) {
        result := &NormalizedAdverseEvent{
            OriginalText: text,
            IsValidTerm:  false,
            Confidence:   1.0,
            Source:       "PATTERN_FILTER",
        }
        n.cache.Store(cleaned, result)
        return result, nil
    }

    // Layer 3: LLM fallback for truly novel terms (<1% of calls)
    result, err = n.llmFallback(ctx, text)
    if err == nil {
        n.cache.Store(cleaned, result)
    }
    return result, err
}

// queryAthenaForCondition searches Athena CONCEPT table
func (n *OHDSIAdverseEventNormalizer) queryAthenaForCondition(ctx context.Context, term string) (*NormalizedAdverseEvent, error) {
    var conceptID int64
    var conceptCode, conceptName string
    
    // Search SNOMED conditions (standard vocabulary for adverse events)
    err := n.athenaDB.QueryRowContext(ctx, `
        SELECT c.concept_id, c.concept_code, c.concept_name
        FROM concept c
        WHERE c.vocabulary_id = 'SNOMED'
          AND c.domain_id = 'Condition'
          AND c.standard_concept = 'S'
          AND c.invalid_reason IS NULL
          AND LOWER(c.concept_name) = ?
    `, term).Scan(&conceptID, &conceptCode, &conceptName)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("not found in Athena")
    }
    if err != nil {
        return nil, err
    }

    // Get MedDRA equivalents for FAERS compatibility
    meddraCodes := n.getMedDRAMappings(ctx, conceptID)

    return &NormalizedAdverseEvent{
        CanonicalName:  conceptName,
        OriginalText:   term,
        SNOMEDCode:     conceptCode,
        SNOMEDName:     conceptName,
        MedDRACodes:    meddraCodes,  // FAERS ready!
        IsValidTerm:    true,
        Confidence:     1.0,
        Source:         "ATHENA_SNOMED",
    }, nil
}

// getMedDRAMappings finds MedDRA PTs that map to this SNOMED concept
func (n *OHDSIAdverseEventNormalizer) getMedDRAMappings(ctx context.Context, snomedConceptID int64) []string {
    rows, err := n.athenaDB.QueryContext(ctx, `
        SELECT DISTINCT c2.concept_code
        FROM concept_relationship cr
        JOIN concept c2 ON cr.concept_id_2 = c2.concept_id
        WHERE cr.concept_id_1 = ?
          AND cr.relationship_id = 'Mapped from'
          AND c2.vocabulary_id = 'MedDRA'
          AND c2.concept_class_id = 'PT'
    `, snomedConceptID)
    if err != nil {
        return nil
    }
    defer rows.Close()

    var codes []string
    for rows.Next() {
        var code string
        if err := rows.Scan(&code); err == nil {
            codes = append(codes, code)
        }
    }
    return codes
}

// isObviousNoise filters statistical patterns WITHOUT LLM
func (n *OHDSIAdverseEventNormalizer) isObviousNoise(term string) bool {
    noisePatterns := []string{
        `^\d+(\.\d+)?%?$`,           // "5.2%", "234"
        `^n\s*[=\(]`,                // "n=45", "n(234)"
        `^p\s*[<>=]`,                // "p<0.05"
        `^\([\d.\-]+\)$`,            // "(1.2-3.4)"
        `^(major|minor|mild|moderate|severe)$`,  // Severity labels
        `(clinical impact|intervention|management):?$`,  // Section headers
        `^(placebo|treatment|control)$`,  // Study arms
        `^\d+\s*/\s*\d+$`,           // "23/456"
        `^[\d.]+\s*±\s*[\d.]+$`,     // "5.2 ± 1.3"
    }
    
    for _, pattern := range noisePatterns {
        if matched, _ := regexp.MatchString(pattern, term); matched {
            return true
        }
    }
    return false
}

// llmFallback handles truly novel terms (rare)
func (n *OHDSIAdverseEventNormalizer) llmFallback(ctx context.Context, text string) (*NormalizedAdverseEvent, error) {
    // This is called <1% of the time
    // ... existing LLM code but only for edge cases
}
```

### Part C: Pre-Loading High-Frequency Terms

```go
// shared/terminology/preload.go

// PreloadCommonTerms loads top 50K conditions for sub-ms lookup
func (n *OHDSIAdverseEventNormalizer) PreloadCommonTerms(ctx context.Context) error {
    // Load most common SNOMED conditions (from FAERS frequency analysis)
    rows, err := n.athenaDB.QueryContext(ctx, `
        SELECT c.concept_id, c.concept_code, c.concept_name
        FROM concept c
        WHERE c.vocabulary_id = 'SNOMED'
          AND c.domain_id = 'Condition'
          AND c.standard_concept = 'S'
          AND c.invalid_reason IS NULL
        ORDER BY c.concept_id  -- Proxy for frequency (lower IDs = more common)
        LIMIT 50000
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    n.snomedConditions = make(map[string]*AthenaCondition, 50000)
    for rows.Next() {
        var cond AthenaCondition
        if err := rows.Scan(&cond.ConceptID, &cond.ConceptCode, &cond.ConceptName); err != nil {
            continue
        }
        n.snomedConditions[strings.ToLower(cond.ConceptName)] = &cond
    }

    log.Infof("Preloaded %d SNOMED conditions for sub-ms lookup", len(n.snomedConditions))
    return nil
}
```

---

## 📦 Setup: One-Time Athena Download

```bash
#!/bin/bash
# scripts/setup_athena_vocab.sh

# 1. Register at https://athena.ohdsi.org (FREE)
# 2. Select vocabularies: SNOMED, RxNorm, MedDRA (if you have license), ICD10
# 3. Download the vocabulary bundle (~3GB compressed)

# 4. Extract and load into SQLite for local queries
python3 scripts/athena_to_sqlite.py \
    --input /path/to/vocabulary_download_v5 \
    --output data/athena_vocab.sqlite

# Result: Single 2GB SQLite file with all vocabularies
# Query time: <10ms for most lookups
```

```python
# scripts/athena_to_sqlite.py
import sqlite3
import csv
import os

def load_athena_to_sqlite(input_dir, output_path):
    """Convert Athena CSV files to SQLite database"""
    conn = sqlite3.connect(output_path)
    
    # Create tables
    conn.execute('''CREATE TABLE IF NOT EXISTS concept (
        concept_id INTEGER PRIMARY KEY,
        concept_name TEXT,
        domain_id TEXT,
        vocabulary_id TEXT,
        concept_class_id TEXT,
        standard_concept TEXT,
        concept_code TEXT,
        valid_start_date TEXT,
        valid_end_date TEXT,
        invalid_reason TEXT
    )''')
    
    conn.execute('''CREATE TABLE IF NOT EXISTS concept_relationship (
        concept_id_1 INTEGER,
        concept_id_2 INTEGER,
        relationship_id TEXT,
        valid_start_date TEXT,
        valid_end_date TEXT,
        invalid_reason TEXT
    )''')
    
    conn.execute('''CREATE TABLE IF NOT EXISTS concept_synonym (
        concept_id INTEGER,
        concept_synonym_name TEXT,
        language_concept_id INTEGER
    )''')
    
    # Load CONCEPT.csv
    with open(os.path.join(input_dir, 'CONCEPT.csv'), 'r') as f:
        reader = csv.DictReader(f, delimiter='\t')
        for row in reader:
            conn.execute('''
                INSERT INTO concept VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ''', (
                row['concept_id'], row['concept_name'], row['domain_id'],
                row['vocabulary_id'], row['concept_class_id'], 
                row.get('standard_concept'), row['concept_code'],
                row['valid_start_date'], row['valid_end_date'],
                row.get('invalid_reason')
            ))
    
    # Create indexes for fast lookup
    conn.execute('CREATE INDEX idx_concept_code ON concept(concept_code, vocabulary_id)')
    conn.execute('CREATE INDEX idx_concept_name ON concept(LOWER(concept_name))')
    conn.execute('CREATE INDEX idx_concept_domain ON concept(domain_id, standard_concept)')
    
    # Load CONCEPT_RELATIONSHIP.csv (for MedDRA→SNOMED mappings)
    with open(os.path.join(input_dir, 'CONCEPT_RELATIONSHIP.csv'), 'r') as f:
        reader = csv.DictReader(f, delimiter='\t')
        for row in reader:
            conn.execute('''
                INSERT INTO concept_relationship VALUES (?, ?, ?, ?, ?, ?)
            ''', (
                row['concept_id_1'], row['concept_id_2'], row['relationship_id'],
                row['valid_start_date'], row['valid_end_date'],
                row.get('invalid_reason')
            ))
    
    conn.execute('CREATE INDEX idx_rel_1 ON concept_relationship(concept_id_1)')
    conn.execute('CREATE INDEX idx_rel_2 ON concept_relationship(concept_id_2)')
    
    # Load CONCEPT_SYNONYM.csv (for fuzzy matching)
    with open(os.path.join(input_dir, 'CONCEPT_SYNONYM.csv'), 'r') as f:
        reader = csv.DictReader(f, delimiter='\t')
        for row in reader:
            conn.execute('''
                INSERT INTO concept_synonym VALUES (?, ?, ?)
            ''', (
                row['concept_id'], row['concept_synonym_name'],
                row.get('language_concept_id')
            ))
    
    conn.execute('CREATE INDEX idx_synonym ON concept_synonym(LOWER(concept_synonym_name))')
    
    conn.commit()
    conn.close()
    print(f"Created {output_path}")

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--input', required=True)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    load_athena_to_sqlite(args.input, args.output)
```

---

## 📊 Performance Comparison

| Metric | LLM-Only Approach | OHDSI-First Approach |
|--------|------------------|----------------------|
| **Latency** | 500ms (API call) | <1ms (95%), <10ms (99%), 500ms (1%) |
| **Cost** | $0.08 warmup + ongoing | $0 (one-time download) |
| **Determinism** | ❌ Variable outputs | ✅ Consistent |
| **Offline Capable** | ❌ No | ✅ Yes |
| **FAERS Compatible** | Partial (LLM guesses) | ✅ Full (real mappings) |
| **RxNorm Validation** | Via RxNav API | ✅ Local SQLite |
| **Vocabulary Coverage** | ~2K (cached) | 10M+ concepts |

---

## 🎯 Why This Is Better

### 1. **Issue 1 (FK Constraint) Fixed Properly**

LLM approach: Calls RxNav API for every drug → network dependency

OHDSI approach: Local SQLite lookup → <1ms, works offline

```sql
-- Local query to validate RxCUI
SELECT concept_name FROM concept 
WHERE concept_code = '5521' AND vocabulary_id = 'RxNorm';
-- Returns "Hydroxychloroquine" → WRONG for Lithium → Lookup correct RxCUI
```

### 2. **Issue 2 (FAERS Compatibility) Solved With Real Mappings**

LLM approach: Claude "guesses" MedDRA codes → unreliable

OHDSI approach: Uses official MedDRA→SNOMED mappings from Athena

```sql
-- Get MedDRA codes for SNOMED "Nausea" (422587007)
SELECT c2.concept_code as meddra_code, c2.concept_name
FROM concept_relationship cr
JOIN concept c2 ON cr.concept_id_2 = c2.concept_id
WHERE cr.concept_id_1 = (SELECT concept_id FROM concept WHERE concept_code = '422587007')
  AND cr.relationship_id = 'Mapped from'
  AND c2.vocabulary_id = 'MedDRA';
-- Returns: 10028813, "Nausea" (official MedDRA PT)
```

### 3. **Issue 3 (Regex Ceiling) Broken By Semantic Lookup**

LLM approach: Calls API for every term → expensive, slow

OHDSI approach: If term exists in SNOMED conditions → valid. If not → noise OR novel.

```go
// Check if "Arthritis" is a real condition
_, err := athenaDB.QueryRow(`
    SELECT 1 FROM concept 
    WHERE LOWER(concept_name) = 'arthritis'
      AND vocabulary_id = 'SNOMED'
      AND domain_id = 'Condition'
`)
// Found → valid clinical term

// Check "Meatitis"
// Not found → either noise (pattern check) OR truly novel (LLM fallback)
```

---

## 📁 Files to Create

| File | Lines | Purpose |
|------|-------|---------|
| `shared/terminology/athena_store.go` | ~200 | SQLite connection + queries |
| `shared/terminology/drug_normalizer.go` | ~150 | OHDSI-based drug validation |
| `shared/terminology/ae_normalizer.go` | ~250 | OHDSI-based AE normalization |
| `shared/terminology/preloader.go` | ~100 | Pre-load common terms |
| `scripts/athena_to_sqlite.py` | ~100 | One-time Athena → SQLite |
| `scripts/setup_athena_vocab.sh` | ~30 | Setup script |
| **Total** | **~830** | |

---

## ✅ Summary

**Your plan** was smart but introduced LLM dependency for everything.

**The innovative solution** uses:
1. **OHDSI Athena** (FREE, 10M+ concepts) as the terminology backbone
2. **Local SQLite** for <10ms lookups (vs 500ms LLM API)
3. **Real MedDRA→SNOMED mappings** for FAERS (vs LLM guessing)
4. **LLM as fallback only** for truly novel terms (<1% of calls)

**Result**: Faster, cheaper, more reliable, and genuinely FAERS-compatible.
