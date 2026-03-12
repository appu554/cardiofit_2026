# Phase 3: Clinical Truth Arbitration - Implementation Plan

> **Timeline**: Weeks 7-17 (11 weeks) *(Updated to include Phase 3b.5 + 3b.6)*
> **Status**: PENDING
> **Philosophy**: *"Freeze meaning. Fluidly replace intelligence."*

---

## Executive Summary

Phase 3 transforms KB-1 from an "extraction engine" into a **Clinical Truth Arbitration Platform**. The key insight is that LLM should be a **gap filler of last resort**, not a primary source of clinical facts.

### What Already Exists (Leverage These)

| Component | Location | Status |
|-----------|----------|--------|
| `DataSource` interface | `shared/datasources/interfaces.go` | ✅ Complete |
| `RxNavClient` interface | `shared/datasources/interfaces.go` | ✅ Complete |
| `RxClassClient` interface | `shared/datasources/interfaces.go` | ✅ Complete |
| FDA DailyMed Client | `kb-1-drug-rules/pkg/ingestion/fda/client.go` | ✅ Partial |
| Redis Cache | `shared/datasources/cache/redis.go` | ✅ Complete |
| MED-RT Client | `shared/datasources/medrt/client.go` | ✅ Complete |
| FactStore | `kb-1-drug-rules/internal/database/postgres.go` | ✅ Complete |

### What Needs to Be Built

| Component | Priority | Effort | Phase |
|-----------|----------|--------|-------|
| **Source-Centric Data Model** | P0 | 2 days | 3a |
| Navigation Rules Engine | P0 | 2 days | 3d |
| DailyMed SPL Fetcher (full XML) | P0 | 3 days | 3a |
| LOINC Section Parser + KB Router | P0 | 3 days | 3a |
| **Tabular Harvester** (table → JSON) | P0 | 2 days | 3a |
| Ground Truth Authority Clients | P0 | 12 days | 3b |
| **DraftRule Schema** | P0 | 1 day | 3b.5 |
| **Unit/Table Normalizers** | P0 | 2.5 days | 3b.5 |
| **Condition-Action Generator** | P0 | 1.5 days | 3b.5 |
| **Rule Translator + Fingerprint** | P0 | 2.5 days | 3b.5 |
| **Untranslatable Queue** | P0 | 1 day | 3b.5 |
| **Conditional Reference Ranges** | P0 | 2 days | 3b.6 |
| **Range Selection Engine** | P0 | 1.5 days | 3b.6 |
| **Pregnancy-Specific Ranges** | P0 | 2 days | 3b.6 |
| **Renal Function Ranges (KDIGO)** | P0 | 1 day | 3b.6 |
| **Neonatal Bilirubin Nomogram** | P0 | 1 day | 3b.6 |
| **KB-16 Context Integration** | P0 | 1 day | 3b.6 |
| Shadow Renbase Extractor | P1 | 3 days | 3c |
| LLM Provider Interface | P1 | 2 days | 3c |
| Race-to-Consensus Engine | P1 | 3 days | 3c |
| Governance Pipeline | P2 | 5 days | 3d |

> **Phase 3b.5 Total**: 8.5 days (critical path for computable rules)
> **Phase 3b.6 Total**: 12 days (KB-16 conditional reference ranges)

---

## Source-Centric Architecture (NEW)

> **Core Insight**: Parse each source ONCE, extract to MULTIPLE KBs. Full lineage from production fact back to regulatory label.

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SOURCE-CENTRIC DATA FLOW                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                  │
│  │ FDA DailyMed │    │   CPIC API   │    │ CredibleMeds │                  │
│  │  SPL (XML)   │    │   (JSON)     │    │    (CSV)     │                  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘                  │
│         │                   │                   │                          │
│         ▼                   ▼                   ▼                          │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │                    source_documents                              │       │
│  │  • id (UUID)           • fetched_at                             │       │
│  │  • source_type         • raw_content_hash                       │       │
│  │  • document_id (SetID) • version_number                         │       │
│  └──────────────────────────┬──────────────────────────────────────┘       │
│                             │                                              │
│                             ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │                    source_sections                               │       │
│  │  • id (UUID)              • section_code (LOINC)                │       │
│  │  • source_document_id     • target_kbs[] (KB-1, KB-4...)        │       │
│  │  • parsed_tables[]        • extraction_method                   │       │
│  └──────────────────────────┬──────────────────────────────────────┘       │
│                             │                                              │
│                             ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │                    derived_facts                                 │       │
│  │  • source_document_id (FK)  • extraction_confidence             │       │
│  │  • source_section_id (FK)   • governance_status (DRAFT)         │       │
│  │  • target_kb                • evidence_spans (quotes)           │       │
│  └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
│  INVARIANT: All facts have source_document_id + source_section_id          │
│             Full audit trail from production fact to regulatory label      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### LOINC Section → KB Routing Table

| LOINC Code | Section Name | Target KBs | Extraction Method |
|------------|--------------|------------|-------------------|
| **34066-1** | Boxed Warning | KB-4 (Safety) | TABLE_PARSE |
| **34068-7** | Dosage & Administration | KB-1, KB-6, KB-16 | TABLE_PARSE + LLM_GAP |
| **34070-3** | Contraindications | KB-4, KB-5 | TABLE_PARSE |
| **34073-7** | Drug Interactions | KB-5 | TABLE_PARSE |
| **34077-8** | Pregnancy/Nursing | KB-4 | AUTHORITY (LactMed) |
| **34090-1** | Clinical Pharmacology | KB-1 (PK) | TABLE_PARSE |
| **43685-7** | Warnings & Precautions | KB-4 | TABLE_PARSE |

### LLM Input Rules (STRICT)

| Input Type | Allowed? | Rationale |
|------------|----------|-----------|
| Entire SPL document | ❌ **NEVER** | Too much noise, hallucination risk |
| LOINC-bounded section | ✅ Yes | Focused, relevant context |
| Parsed dosing table JSON | ✅ Yes | Structured, verifiable |
| Ontology-linked fragment | ✅ Yes | RxCUI/ATC anchored |
| Free-text prose | ⚠️ Only with consensus | Requires 2-of-3 agreement |

---

## Phase 3a: Foundation Infrastructure (Weeks 7-8)

### Goal
Establish the **Source-Centric** data model and LOINC section parsing infrastructure.

### 3a.1 Deploy RxNav-in-a-Box (2 days)

**Why**: Unlimited local RxNorm API calls without rate limits. Provides **deterministic NDC → SPL SetID** mapping via RxNorm SPL Links.

**Implementation**:

```bash
# docker/rxnav/docker-compose.yml
version: '3.8'
services:
  rxnav:
    image: lhncbc/rxnav-in-a-box:latest
    ports:
      - "4000:4000"
    volumes:
      - rxnav-data:/data
    environment:
      - RXNAV_DATA_URL=https://download.nlm.nih.gov/rxnorm/RxNorm_full_current.zip
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4000/REST/version"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  rxnav-data:
```

**File**: `backend/shared-infrastructure/docker/rxnav/docker-compose.yml`

**Key Feature - RxNorm SPL Links**:
```go
// Use RxNorm's SPL link for deterministic NDC → SetID mapping
// Maintained by National Library of Medicine
func (c *RxNavClient) GetSPLSetID(ctx context.Context, ndc string) (string, error) {
    // GET /REST/rxcui.json?idtype=NDC&id={ndc}
    // Then: GET /REST/rxcui/{rxcui}/related?tty=SBD+SCD
    // Finally: GET /REST/rxcui/{rxcui}/property?propName=SPL_SET_ID
}
```

---

### 3a.2 Create Source-Centric Data Model (2 days)

**Why**: Single Parse → Multi-KB distribution with full audit trail.

**File**: `backend/shared-infrastructure/knowledge-base-services/migrations/005_source_centric_model.sql`

```sql
-- Source-Centric Data Model for Phase 3
-- Enables: Single parse → Multi-KB distribution

CREATE TABLE source_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source identification
    source_type VARCHAR(50) NOT NULL,     -- 'FDA_SPL', 'CPIC', 'CREDIBLEMEDS', etc.
    document_id VARCHAR(255) NOT NULL,    -- SetID for SPL, PMID for CPIC
    version_number VARCHAR(50),

    -- Content tracking
    raw_content_hash VARCHAR(64) NOT NULL, -- SHA-256 for change detection
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Metadata
    drug_name VARCHAR(255),
    rxcui VARCHAR(50),
    ndc_codes TEXT[],

    UNIQUE(source_type, document_id, version_number)
);

CREATE TABLE source_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_document_id UUID NOT NULL REFERENCES source_documents(id),

    -- Section identification (LOINC-based for SPL)
    section_code VARCHAR(50) NOT NULL,      -- LOINC code: '34068-7'
    section_name VARCHAR(255),

    -- Routing configuration
    target_kbs TEXT[] NOT NULL,             -- ['KB-1', 'KB-4', 'KB-6']

    -- Parsed content
    raw_text TEXT,
    parsed_tables JSONB,                    -- Tabular Harvester output
    extraction_method VARCHAR(50),          -- 'TABLE_PARSE', 'LLM_GAP', 'AUTHORITY'
    extraction_confidence DECIMAL(5,4),

    UNIQUE(source_document_id, section_code)
);

CREATE TABLE derived_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Full Lineage (Source-Centric)
    source_document_id UUID NOT NULL REFERENCES source_documents(id),
    source_section_id UUID REFERENCES source_sections(id),

    -- Fact content
    target_kb VARCHAR(20) NOT NULL,         -- 'KB-1', 'KB-4', etc.
    fact_type VARCHAR(100) NOT NULL,
    fact_data JSONB NOT NULL,

    -- Extraction metadata
    extraction_method VARCHAR(50) NOT NULL, -- 'AUTHORITY', 'TABLE_PARSE', 'LLM_CONSENSUS'
    extraction_confidence DECIMAL(5,4),
    evidence_spans JSONB,                   -- Quoted source text

    -- Governance
    governance_status VARCHAR(20) DEFAULT 'DRAFT',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_source_docs_rxcui ON source_documents(rxcui);
CREATE INDEX idx_source_docs_type ON source_documents(source_type);
CREATE INDEX idx_source_sections_code ON source_sections(section_code);
CREATE INDEX idx_source_sections_target ON source_sections USING GIN(target_kbs);
CREATE INDEX idx_derived_facts_kb ON derived_facts(target_kb);
CREATE INDEX idx_derived_facts_status ON derived_facts(governance_status);

-- CRITICAL CONSTRAINT: Every fact must have source lineage
ALTER TABLE derived_facts ADD CONSTRAINT fact_must_have_source
    CHECK (source_document_id IS NOT NULL);
```

---

### 3a.3 Build DailyMed SPL Fetcher (3 days)

**Why**: Full SPL XML download with delta sync for drug labels.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/dailymed/fetcher.go`

```go
package dailymed

import (
    "context"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "time"
)

// SPLFetcher downloads and caches FDA Structured Product Labels
type SPLFetcher struct {
    baseURL     string
    httpClient  *http.Client
    cache       SPLCache
    deltaURL    string // For incremental updates
}

// SPLDocument represents a full SPL XML document
type SPLDocument struct {
    SetID         string    `xml:"setId>root,attr"`
    VersionNumber string    `xml:"versionNumber,attr"`
    EffectiveTime time.Time `xml:"effectiveTime,attr"`

    // Key sections by LOINC code
    Sections []SPLSection `xml:"component>structuredBody>component>section"`
}

// SPLSection represents a labeled section with LOINC code
type SPLSection struct {
    ID        string       `xml:"id,attr"`
    Code      string       `xml:"code>code,attr"`        // LOINC code
    CodeSystem string      `xml:"code>codeSystem,attr"` // Should be 2.16.840.1.113883.6.1
    Title     string       `xml:"title"`
    Text      SPLText      `xml:"text"`
    Subsections []SPLSection `xml:"component>section"`
}

// SPLText contains the narrative content (may include tables)
type SPLText struct {
    Content    string     `xml:",innerxml"`
    Tables     []SPLTable `xml:"table"`
    Paragraphs []string   `xml:"paragraph"`
}

// SPLTable represents a structured table in SPL
type SPLTable struct {
    ID      string        `xml:"ID,attr"`
    Headers []string      `xml:"thead>tr>th"`
    Rows    []SPLTableRow `xml:"tbody>tr"`
}

type SPLTableRow struct {
    Cells []string `xml:"td"`
}

// FetchBySetID retrieves SPL by its unique set identifier
func (f *SPLFetcher) FetchBySetID(ctx context.Context, setID string) (*SPLDocument, error) {
    // Check cache first
    if cached, found := f.cache.Get(setID); found {
        return cached, nil
    }

    url := fmt.Sprintf("%s/spls/%s.xml", f.baseURL, setID)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }

    resp, err := f.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetching SPL: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("SPL fetch failed: %s", resp.Status)
    }

    var doc SPLDocument
    if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
        return nil, fmt.Errorf("parsing SPL XML: %w", err)
    }

    // Cache for future use
    f.cache.Set(setID, &doc)

    return &doc, nil
}

// FetchByRxCUI retrieves SPL for a drug by its RxCUI
func (f *SPLFetcher) FetchByRxCUI(ctx context.Context, rxcui string) (*SPLDocument, error) {
    // First, resolve RxCUI to SetID via DailyMed API
    setID, err := f.resolveSetID(ctx, rxcui)
    if err != nil {
        return nil, fmt.Errorf("resolving SetID for RxCUI %s: %w", rxcui, err)
    }

    return f.FetchBySetID(ctx, setID)
}

// GetSection extracts a specific section by LOINC code
func (doc *SPLDocument) GetSection(loincCode string) *SPLSection {
    for _, section := range doc.Sections {
        if section.Code == loincCode {
            return &section
        }
        // Check subsections
        for _, sub := range section.Subsections {
            if sub.Code == loincCode {
                return &sub
            }
        }
    }
    return nil
}

// Key LOINC codes for clinical decision support
const (
    LOINCDosageAdministration = "34068-7" // Dosage and Administration
    LOINCBoxedWarning         = "34066-1" // Boxed Warning
    LOINCContraindications    = "34070-3" // Contraindications
    LOINCWarningsPrecautions  = "43685-7" // Warnings and Precautions
    LOINCPregnancy            = "34077-8" // Pregnancy
    LOINCNursing              = "34080-2" // Nursing Mothers
    LOINCPediatricUse         = "34081-0" // Pediatric Use
    LOINCGeriatricUse         = "34082-8" // Geriatric Use
    LOINCClinicalPharm        = "34090-1" // Clinical Pharmacology
    LOINCDrugInteractions     = "34073-7" // Drug Interactions
)
```

---

### 3a.4 Build LOINC Section Parser (3 days)

**Why**: Extract structured data from specific SPL sections using LOINC codes.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/spl/loinc_parser.go`

```go
package spl

import (
    "regexp"
    "strings"

    "github.com/cardiofit/shared/datasources/dailymed"
)

// LOINCSectionParser extracts clinical facts from SPL sections
type LOINCSectionParser struct {
    tableExtractor *TableExtractor
    gfrPattern     *regexp.Regexp
    childPughPattern *regexp.Regexp
}

// NewLOINCSectionParser creates a parser with clinical regex patterns
func NewLOINCSectionParser() *LOINCSectionParser {
    return &LOINCSectionParser{
        tableExtractor: NewTableExtractor(),
        // GFR patterns: "CrCl < 30", "eGFR 30-60", "GFR less than 15"
        gfrPattern: regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine clearance)\s*[<>≤≥]?\s*(\d+)(?:\s*-\s*(\d+))?`),
        // Child-Pugh patterns: "Child-Pugh A", "moderate hepatic impairment"
        childPughPattern: regexp.MustCompile(`(?i)(Child-Pugh\s*([ABC]))|((mild|moderate|severe)\s+hepatic\s+impairment)`),
    }
}

// ParsedSection contains extracted clinical data from an SPL section
type ParsedSection struct {
    LOINCCode       string
    HasStructuredTable bool
    Tables          []ParsedTable
    GFRThresholds   []GFRThreshold
    ChildPughClass  []ChildPughClassification
    RawText         string
    ExtractionType  string // "TABLE_PARSE" or "NEEDS_LLM"
}

type GFRThreshold struct {
    Operator   string  // "<", ">", "<=", ">=", "range"
    LowerBound float64
    UpperBound float64 // Only for range
    Action     string  // Will be extracted: "reduce dose", "avoid", "contraindicated"
}

type ParsedTable struct {
    Headers []string
    Rows    [][]string
    Type    string // "GFR_DOSE_TABLE", "HEPATIC_TABLE", "INTERACTION_TABLE", "UNKNOWN"
}

// ParseDosageSection extracts renal/hepatic dosing from LOINC 34068-7
func (p *LOINCSectionParser) ParseDosageSection(section *dailymed.SPLSection) (*ParsedSection, error) {
    result := &ParsedSection{
        LOINCCode: section.Code,
        RawText:   section.Text.Content,
    }

    // Priority 1: Extract structured tables
    if len(section.Text.Tables) > 0 {
        result.HasStructuredTable = true
        result.ExtractionType = "TABLE_PARSE"

        for _, table := range section.Text.Tables {
            parsed := p.parseTable(table)
            result.Tables = append(result.Tables, parsed)

            // Extract GFR thresholds from table
            if parsed.Type == "GFR_DOSE_TABLE" {
                thresholds := p.extractGFRFromTable(parsed)
                result.GFRThresholds = append(result.GFRThresholds, thresholds...)
            }
        }

        return result, nil
    }

    // Priority 2: Extract from prose using regex
    result.ExtractionType = "NEEDS_LLM" // May need LLM for complex prose

    // Try regex extraction first
    gfrMatches := p.gfrPattern.FindAllStringSubmatch(section.Text.Content, -1)
    for _, match := range gfrMatches {
        threshold := p.parseGFRMatch(match)
        if threshold != nil {
            result.GFRThresholds = append(result.GFRThresholds, *threshold)
        }
    }

    // If we found GFR thresholds via regex, upgrade extraction type
    if len(result.GFRThresholds) > 0 {
        result.ExtractionType = "REGEX_PARSE"
    }

    return result, nil
}

// parseTable analyzes table structure and classifies it
func (p *LOINCSectionParser) parseTable(table dailymed.SPLTable) ParsedTable {
    parsed := ParsedTable{
        Headers: table.Headers,
        Type:    "UNKNOWN",
    }

    // Convert rows
    for _, row := range table.Rows {
        parsed.Rows = append(parsed.Rows, row.Cells)
    }

    // Classify table type by headers
    headerStr := strings.ToLower(strings.Join(table.Headers, " "))

    switch {
    case strings.Contains(headerStr, "creatinine") ||
         strings.Contains(headerStr, "gfr") ||
         strings.Contains(headerStr, "renal"):
        parsed.Type = "GFR_DOSE_TABLE"

    case strings.Contains(headerStr, "child-pugh") ||
         strings.Contains(headerStr, "hepatic"):
        parsed.Type = "HEPATIC_TABLE"

    case strings.Contains(headerStr, "interaction") ||
         strings.Contains(headerStr, "concomitant"):
        parsed.Type = "INTERACTION_TABLE"
    }

    return parsed
}

// CanExtractWithoutLLM returns true if we have enough structured data
func (p *ParsedSection) CanExtractWithoutLLM() bool {
    return p.ExtractionType == "TABLE_PARSE" || p.ExtractionType == "REGEX_PARSE"
}

// RouteToKBs determines which KBs should receive facts from this section
func (p *LOINCSectionParser) RouteToKBs(loincCode string) []string {
    routing := map[string][]string{
        "34066-1": {"KB-4"},                    // Boxed Warning → Safety
        "34068-7": {"KB-1", "KB-6", "KB-16"},  // Dosage → Dosing, Formulary, Lab
        "34070-3": {"KB-4", "KB-5"},           // Contraindications → Safety, DDI
        "34073-7": {"KB-5"},                    // Drug Interactions → DDI
        "34077-8": {"KB-4"},                    // Pregnancy → Safety (also LactMed)
        "34090-1": {"KB-1"},                    // Clinical Pharm → PK params
        "43685-7": {"KB-4"},                    // Warnings → Safety
    }
    if targets, ok := routing[loincCode]; ok {
        return targets
    }
    return []string{} // Unknown section, don't route
}
```

---

### 3a.5 Build Tabular Harvester (2 days)

**Why**: Parse `<table>` blocks from SPL XML and output structured JSON. This becomes the **renal dosing spine**.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/spl/tabular_harvester.go`

```go
package spl

import (
    "encoding/json"
    "strings"

    "golang.org/x/net/html"
)

// TabularHarvester extracts structured data from SPL HTML tables
// Output: JSON, not prose. Preserves headers, units, footnotes.
type TabularHarvester struct {
    unitNormalizer *UnitNormalizer
}

// HarvestedTable represents a fully parsed table in JSON format
type HarvestedTable struct {
    ID          string           `json:"id"`
    Type        TableType        `json:"type"`        // GFR_DOSE, HEPATIC_DOSE, INTERACTION, etc.
    Headers     []ColumnHeader   `json:"headers"`
    Rows        []TableRow       `json:"rows"`
    Footnotes   []string         `json:"footnotes,omitempty"`
    SourceLOINC string           `json:"source_loinc"` // Which section it came from
}

type ColumnHeader struct {
    Name string `json:"name"`
    Unit string `json:"unit,omitempty"` // e.g., "mL/min", "mg", "%"
}

type TableRow struct {
    Condition  string            `json:"condition"`           // e.g., "CrCl 30-60 mL/min"
    Values     map[string]string `json:"values"`              // Column name → value
    Parsed     *ParsedDoseRule   `json:"parsed,omitempty"`    // Structured interpretation
}

type ParsedDoseRule struct {
    GFRMin       *float64 `json:"gfr_min,omitempty"`
    GFRMax       *float64 `json:"gfr_max,omitempty"`
    ChildPugh    string   `json:"child_pugh,omitempty"`    // "A", "B", "C"
    Action       string   `json:"action"`                   // "REDUCE", "AVOID", "CONTRAINDICATED"
    DoseAdjust   string   `json:"dose_adjust,omitempty"`   // "50%", "250mg BID"
    MaxDose      string   `json:"max_dose,omitempty"`
}

type TableType string

const (
    TableTypeGFRDose      TableType = "GFR_DOSE"
    TableTypeHepaticDose  TableType = "HEPATIC_DOSE"
    TableTypeInteraction  TableType = "INTERACTION"
    TableTypeAdverseEvent TableType = "ADVERSE_EVENT"
    TableTypeUnknown      TableType = "UNKNOWN"
)

// Harvest extracts all tables from an SPL section and converts to JSON
func (h *TabularHarvester) Harvest(htmlContent string, loincCode string) ([]HarvestedTable, error) {
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        return nil, err
    }

    var tables []HarvestedTable
    h.extractTables(doc, loincCode, &tables)
    return tables, nil
}

// ToJSON serializes a table for storage in source_sections.parsed_tables
func (t *HarvestedTable) ToJSON() ([]byte, error) {
    return json.Marshal(t)
}

// ClassifyTable determines the table type based on headers and context
func (h *TabularHarvester) ClassifyTable(headers []string) TableType {
    headerStr := strings.ToLower(strings.Join(headers, " "))

    switch {
    case strings.Contains(headerStr, "creatinine") ||
         strings.Contains(headerStr, "gfr") ||
         strings.Contains(headerStr, "renal") ||
         strings.Contains(headerStr, "clcr"):
        return TableTypeGFRDose

    case strings.Contains(headerStr, "child-pugh") ||
         strings.Contains(headerStr, "hepatic") ||
         strings.Contains(headerStr, "liver"):
        return TableTypeHepaticDose

    case strings.Contains(headerStr, "interaction") ||
         strings.Contains(headerStr, "concomitant") ||
         strings.Contains(headerStr, "co-administration"):
        return TableTypeInteraction

    case strings.Contains(headerStr, "adverse") ||
         strings.Contains(headerStr, "side effect"):
        return TableTypeAdverseEvent

    default:
        return TableTypeUnknown
    }
}
```

**Key Principle**:
- Input: SPL HTML with `<table>` elements
- Output: **JSON, not prose** - Structured, verifiable, LLM-safe
- Preserve: headers, units, footnotes
- This becomes the **KB-1 renal dosing spine**

---

### 3a Exit Criteria Checklist

- [ ] RxNav-in-a-Box running locally on port 4000
- [ ] **RxNorm SPL Links working** (NDC → SetID deterministic mapping)
- [ ] **source_documents table** created and populated
- [ ] **source_sections table** created with LOINC routing
- [ ] Can fetch SPL XML for any drug by SetID or RxCUI
- [ ] Can parse LOINC sections 34068-7, 34066-1, 34077-8
- [ ] **Tabular Harvester** outputs JSON for dosing tables
- [ ] **LOINC → KB routing** working (34068-7 → KB-1, KB-6, KB-16)
- [ ] Can identify if section has structured tables vs prose
- [ ] Unit tests for SPL parser with sample drug labels (metformin, lisinopril)

---

## Phase 3b: Ground Truth Ingestion (Weeks 9-10)

### Goal
Connect to and ingest data from authoritative clinical knowledge sources.

### 3b.1 CPIC API Client (2 days)

**Authority Level**: DEFINITIVE (LLM = ❌ NEVER)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/cpic/client.go`

```go
package cpic

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// Client provides access to CPIC pharmacogenomics guidelines
// API: https://api.cpicpgx.org/
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// GenedrugPair represents a gene-drug interaction from CPIC
type GenedrugPair struct {
    DrugName       string   `json:"drugName"`
    Gene           string   `json:"gene"`
    CPICLevel      string   `json:"cpicLevel"`      // "A", "A/B", "B", "C", "D"
    PGxOnFDALabel  bool     `json:"pgxOnFdaLabel"`
    Guideline      *Guideline `json:"guideline,omitempty"`
    Recommendations []Recommendation `json:"recommendations,omitempty"`
}

type Guideline struct {
    URL         string `json:"url"`
    Publication string `json:"publication"`
    PMID        string `json:"pmid"`
}

type Recommendation struct {
    Phenotype    string `json:"phenotype"`    // "Poor Metabolizer", "Intermediate Metabolizer"
    Implication  string `json:"implication"`
    Recommendation string `json:"recommendation"`
    Classification string `json:"classification"` // "Strong", "Moderate", "Optional"
}

// GetGenedrugPairs retrieves all gene-drug pairs for a drug
func (c *Client) GetGenedrugPairs(ctx context.Context, drugName string) ([]GenedrugPair, error) {
    url := fmt.Sprintf("%s/v1/gene_drug?drug=%s", c.baseURL, drugName)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var pairs []GenedrugPair
    if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
        return nil, err
    }

    return pairs, nil
}

// GetRecommendationsForGenotype returns dosing recommendations for a specific genotype
func (c *Client) GetRecommendationsForGenotype(ctx context.Context, drugName, gene, phenotype string) (*Recommendation, error) {
    pairs, err := c.GetGenedrugPairs(ctx, drugName)
    if err != nil {
        return nil, err
    }

    for _, pair := range pairs {
        if pair.Gene == gene {
            for _, rec := range pair.Recommendations {
                if rec.Phenotype == phenotype {
                    return &rec, nil
                }
            }
        }
    }

    return nil, fmt.Errorf("no recommendation found for %s/%s/%s", drugName, gene, phenotype)
}
```

---

### 3b.2 CredibleMeds API Client (2 days)

**Authority Level**: DEFINITIVE (LLM = ❌ NEVER)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/crediblemeds/client.go`

```go
package crediblemeds

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// Client provides access to CredibleMeds QT risk database
// Note: Requires academic/clinical license
type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

// QTRiskCategory represents CredibleMeds QT risk classification
type QTRiskCategory string

const (
    QTRiskKnown          QTRiskCategory = "KNOWN_RISK"          // Known risk of TdP
    QTRiskPossible       QTRiskCategory = "POSSIBLE_RISK"       // Possible risk of TdP
    QTRiskConditional    QTRiskCategory = "CONDITIONAL_RISK"    // Conditional risk (hypokalemia, etc.)
    QTRiskAvoidWithSQT   QTRiskCategory = "AVOID_WITH_SQT"      // Avoid in short QT syndrome
    QTRiskNone           QTRiskCategory = "NONE"                // Not on the list
)

// DrugQTRisk represents a drug's QT prolongation risk
type DrugQTRisk struct {
    DrugName       string         `json:"drugName"`
    GenericName    string         `json:"genericName"`
    RiskCategory   QTRiskCategory `json:"riskCategory"`
    LastReviewed   string         `json:"lastReviewed"`
    Evidence       []Evidence     `json:"evidence,omitempty"`
    Contraindications []string    `json:"contraindications,omitempty"`
}

type Evidence struct {
    PMID       string `json:"pmid"`
    Summary    string `json:"summary"`
    StudyType  string `json:"studyType"` // "Case Report", "Clinical Trial", "Meta-analysis"
}

// GetQTRisk retrieves QT risk classification for a drug
func (c *Client) GetQTRisk(ctx context.Context, drugName string) (*DrugQTRisk, error) {
    url := fmt.Sprintf("%s/drugs/search?name=%s", c.baseURL, drugName)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("X-API-Key", c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        // Drug not on CredibleMeds list = no known QT risk
        return &DrugQTRisk{
            DrugName:     drugName,
            RiskCategory: QTRiskNone,
        }, nil
    }

    var risk DrugQTRisk
    if err := json.NewDecoder(resp.Body).Decode(&risk); err != nil {
        return nil, err
    }

    return &risk, nil
}

// IsHighQTRisk returns true if drug has known or possible QT risk
func (risk *DrugQTRisk) IsHighQTRisk() bool {
    return risk.RiskCategory == QTRiskKnown || risk.RiskCategory == QTRiskPossible
}
```

---

### 3b.3 LiverTox XML Ingestion (2 days)

**Authority Level**: DEFINITIVE (LLM = ❌ NEVER)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/livertox/ingest.go`

```go
package livertox

import (
    "context"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
)

// LiverToxDB represents the full LiverTox database
// Download: https://www.ncbi.nlm.nih.gov/books/NBK547852/
type LiverToxDB struct {
    Drugs map[string]*DrugHepatotoxicity
}

// DrugHepatotoxicity contains hepatotoxicity data for a drug
type DrugHepatotoxicity struct {
    DrugName          string            `xml:"drugName"`
    PubChemCID        string            `xml:"pubChemCid"`
    RxCUI             string            `xml:"rxcui,omitempty"`

    // Likelihood scores (A-E)
    LikelihoodScore   string            `xml:"likelihoodScore"`   // A=Well-established, E=Unproven
    LikelihoodCategory string           `xml:"likelihoodCategory"`

    // Clinical patterns
    LatencyRange      string            `xml:"latencyRange"`      // "1-8 weeks"
    Pattern           string            `xml:"pattern"`           // "Hepatocellular", "Cholestatic", "Mixed"
    Severity          string            `xml:"severity"`          // "Mild", "Moderate", "Severe"

    // Mechanism and features
    Mechanism         string            `xml:"mechanism"`
    ImmunologicFeatures bool            `xml:"immunologicFeatures"`
    Autoimmune        bool              `xml:"autoimmune"`

    // References
    References        []Reference       `xml:"references>reference"`
}

type Reference struct {
    PMID    string `xml:"pmid"`
    Title   string `xml:"title"`
    Authors string `xml:"authors"`
    Year    string `xml:"year"`
}

// HepatotoxicityRisk severity levels
type HepatotoxicityRisk string

const (
    HepatoRiskHigh     HepatotoxicityRisk = "HIGH"     // Likelihood A-B
    HepatoRiskModerate HepatotoxicityRisk = "MODERATE" // Likelihood C
    HepatoRiskLow      HepatotoxicityRisk = "LOW"      // Likelihood D
    HepatoRiskUnknown  HepatotoxicityRisk = "UNKNOWN"  // Likelihood E or not in DB
)

// GetRiskLevel converts LiverTox likelihood to risk level
func (d *DrugHepatotoxicity) GetRiskLevel() HepatotoxicityRisk {
    switch d.LikelihoodScore {
    case "A", "B":
        return HepatoRiskHigh
    case "C":
        return HepatoRiskModerate
    case "D":
        return HepatoRiskLow
    default:
        return HepatoRiskUnknown
    }
}

// Ingestor handles LiverTox database updates
type Ingestor struct {
    downloadURL string
    db          *LiverToxDB
    factStore   FactStoreWriter
}

// IngestFull downloads and processes the complete LiverTox database
func (i *Ingestor) IngestFull(ctx context.Context) error {
    // Download LiverTox XML dump
    resp, err := http.Get(i.downloadURL)
    if err != nil {
        return fmt.Errorf("downloading LiverTox: %w", err)
    }
    defer resp.Body.Close()

    // Parse XML
    var db LiverToxDB
    if err := xml.NewDecoder(resp.Body).Decode(&db); err != nil {
        return fmt.Errorf("parsing LiverTox XML: %w", err)
    }

    // Store each drug's hepatotoxicity data
    for drugName, drug := range db.Drugs {
        fact := i.toFact(drugName, drug)
        if err := i.factStore.StoreFact(ctx, fact); err != nil {
            return fmt.Errorf("storing fact for %s: %w", drugName, err)
        }
    }

    return nil
}
```

---

### 3b.4 LactMed XML Ingestion (2 days)

**Authority Level**: DEFINITIVE (LLM = ❌ NEVER)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/lactmed/ingest.go`

```go
package lactmed

// DrugLactation contains breastfeeding safety data from LactMed
type DrugLactation struct {
    DrugName            string    `xml:"drugName"`
    RxCUI               string    `xml:"rxcui,omitempty"`

    // Relative Infant Dose (RID) - key metric
    RIDPercent          *float64  `xml:"ridPercent,omitempty"`      // <10% generally acceptable
    RIDCategory         string    `xml:"ridCategory"`               // "LOW", "MODERATE", "HIGH"

    // Safety classification
    SafetyCategory      string    `xml:"safetyCategory"`            // "Compatible", "Use with caution", "Contraindicated"
    AAPrecommendation   string    `xml:"aapRecommendation"`         // American Academy of Pediatrics

    // Clinical details
    ExcretedInMilk      bool      `xml:"excretedInMilk"`
    OralBioavailability string    `xml:"oralBioavailability"`       // Affects infant exposure
    MilkPlasmaRatio     *float64  `xml:"milkPlasmaRatio,omitempty"` // M/P ratio

    // Effects observed
    InfantEffects       []string  `xml:"infantEffects>effect"`
    MaternalEffects     []string  `xml:"maternalEffects>effect"`

    // Recommendations
    MonitoringRequired  []string  `xml:"monitoring>item"`
    Alternatives        []string  `xml:"alternatives>drug"`

    LastUpdated         string    `xml:"lastUpdated"`
}

// LactationSafetyLevel represents breastfeeding safety
type LactationSafetyLevel string

const (
    LactationSafe         LactationSafetyLevel = "SAFE"           // RID <2%, no concerns
    LactationProbablySafe LactationSafetyLevel = "PROBABLY_SAFE"  // RID 2-10%, monitor
    LactationUseWithCaution LactationSafetyLevel = "USE_CAUTION"  // RID 10-25%, risks
    LactationAvoid        LactationSafetyLevel = "AVOID"          // RID >25% or contraindicated
)

// GetSafetyLevel calculates breastfeeding safety from RID
func (d *DrugLactation) GetSafetyLevel() LactationSafetyLevel {
    if d.RIDPercent == nil {
        // No RID data - use category
        switch d.SafetyCategory {
        case "Compatible":
            return LactationSafe
        case "Contraindicated":
            return LactationAvoid
        default:
            return LactationUseWithCaution
        }
    }

    rid := *d.RIDPercent
    switch {
    case rid < 2:
        return LactationSafe
    case rid < 10:
        return LactationProbablySafe
    case rid < 25:
        return LactationUseWithCaution
    default:
        return LactationAvoid
    }
}
```

---

### 3b.5 DrugBank Structured Loader (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/drugbank/loader.go`

### 3b.6 OHDSI Beers/STOPP Loader (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/datasources/ohdsi/beers.go`

---

### 3b Exit Criteria Checklist

- [ ] CPIC API returns pharmacogenomics guidelines
- [ ] CredibleMeds API returns QT risk categories
- [ ] LiverTox data loaded into FactStore
- [ ] LactMed data loaded into FactStore
- [ ] DrugBank PK parameters accessible
- [ ] OHDSI Beers/STOPP concept sets loaded
- [ ] All facts have provenance tracking

---

## Phase 3b.5: Canonical Rule Generation (Week 10.5-11)

### Goal
Transform classified tables into **computable IF/THEN rules** using a canonical schema. This bridges the gap between table classification (which exists) and actionable clinical decision rules.

> **Core Insight**: Classification without translation = metadata without actionable rules.

### 3b.5.1 DraftRule Contract (1 day)

**Why**: Every extracted rule needs a canonical, computable representation with full provenance.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/rules/draft_rule.go`

```go
package rules

import (
    "crypto/sha256"
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// DraftRule represents a canonical, computable clinical rule
// Invariant: Every rule has a semantic fingerprint and full provenance
type DraftRule struct {
    RuleID              uuid.UUID         `json:"rule_id" db:"rule_id"`
    Domain              string            `json:"domain" db:"domain"`                // KB-1, KB-4, KB-5
    RuleType            RuleType          `json:"rule_type" db:"rule_type"`          // DOSING, CONTRAINDICATION, INTERACTION

    // Computable IF/THEN structure
    Condition           Condition         `json:"condition" db:"condition"`
    Action              Action            `json:"action" db:"action"`

    // Full lineage
    Provenance          Provenance        `json:"provenance" db:"provenance"`

    // Semantic deduplication
    SemanticFingerprint Fingerprint       `json:"semantic_fingerprint" db:"semantic_fingerprint"`

    // Governance
    GovernanceStatus    GovernanceStatus  `json:"governance_status" db:"governance_status"`
    ReviewedBy          *string           `json:"reviewed_by,omitempty" db:"reviewed_by"`
    ReviewedAt          *time.Time        `json:"reviewed_at,omitempty" db:"reviewed_at"`

    CreatedAt           time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt           time.Time         `json:"updated_at" db:"updated_at"`
}

// RuleType classifies the clinical rule category
type RuleType string

const (
    RuleTypeDosing          RuleType = "DOSING"
    RuleTypeContraindication RuleType = "CONTRAINDICATION"
    RuleTypeInteraction     RuleType = "INTERACTION"
    RuleTypeMonitoring      RuleType = "MONITORING"
    RuleTypeWarning         RuleType = "WARNING"
)

// Condition represents the IF part of the rule
// Schema: {variable, operator, value, unit}
type Condition struct {
    Variable    string    `json:"variable"`     // renal_function.crcl, hepatic.child_pugh, age
    Operator    Operator  `json:"operator"`     // <, >, <=, >=, BETWEEN, ==, IN
    Value       *float64  `json:"value,omitempty"`
    MinValue    *float64  `json:"min_value,omitempty"`   // For BETWEEN operator
    MaxValue    *float64  `json:"max_value,omitempty"`   // For BETWEEN operator
    StringValue *string   `json:"string_value,omitempty"` // For categorical: "A", "B", "C"
    Unit        string    `json:"unit"`          // ml/min, mg, percent
}

// Operator defines the comparison type
type Operator string

const (
    OpLessThan          Operator = "<"
    OpGreaterThan       Operator = ">"
    OpLessOrEqual       Operator = "<="
    OpGreaterOrEqual    Operator = ">="
    OpBetween           Operator = "BETWEEN"
    OpEquals            Operator = "=="
    OpNotEquals         Operator = "!="
    OpIn                Operator = "IN"
)

// Action represents the THEN part of the rule
// Schema: {effect, adjustment}
type Action struct {
    Effect      Effect           `json:"effect"`             // CONTRAINDICATED, DOSE_ADJUST, AVOID, MONITOR
    Adjustment  *DoseAdjustment  `json:"adjustment,omitempty"`
    Message     string           `json:"message,omitempty"`  // Human-readable recommendation
}

// Effect defines the clinical action type
type Effect string

const (
    EffectContraindicated  Effect = "CONTRAINDICATED"
    EffectDoseAdjust       Effect = "DOSE_ADJUST"
    EffectAvoid            Effect = "AVOID"
    EffectMonitor          Effect = "MONITOR"
    EffectUseWithCaution   Effect = "USE_WITH_CAUTION"
    EffectNoChange         Effect = "NO_CHANGE"
)

// DoseAdjustment contains specific dosing modifications
type DoseAdjustment struct {
    Type        AdjustmentType `json:"type"`                // PERCENTAGE, ABSOLUTE, INTERVAL
    Percentage  *float64       `json:"percentage,omitempty"` // 50 = 50% of normal dose
    AbsoluteDose *string       `json:"absolute_dose,omitempty"` // "250mg BID"
    MaxDose     *string        `json:"max_dose,omitempty"`   // "500mg daily"
    Interval    *string        `json:"interval,omitempty"`   // "Every 48 hours"
}

// AdjustmentType defines how the dose is modified
type AdjustmentType string

const (
    AdjustmentPercentage AdjustmentType = "PERCENTAGE"
    AdjustmentAbsolute   AdjustmentType = "ABSOLUTE"
    AdjustmentInterval   AdjustmentType = "INTERVAL"
    AdjustmentMaxDose    AdjustmentType = "MAX_DOSE"
)

// Provenance tracks the complete source lineage
type Provenance struct {
    SourceDocumentID  uuid.UUID  `json:"source_document_id"`
    SourceSectionID   *uuid.UUID `json:"source_section_id,omitempty"`
    SourceType        string     `json:"source_type"`        // FDA_SPL, CPIC, etc.
    DocumentID        string     `json:"document_id"`        // SetID for SPL
    SectionCode       string     `json:"section_code"`       // LOINC code
    ExtractionMethod  string     `json:"extraction_method"`  // TABLE_PARSE, REGEX_PARSE
    EvidenceSpan      string     `json:"evidence_span"`      // Quoted source text
    Confidence        float64    `json:"confidence"`         // 0.0 - 1.0
}

// Fingerprint provides semantic deduplication
type Fingerprint struct {
    Hash      string    `json:"hash"`       // SHA256 of canonical JSON
    Version   int       `json:"version"`    // Schema version for hash compatibility
    CreatedAt time.Time `json:"created_at"`
}

// GovernanceStatus tracks the rule lifecycle
type GovernanceStatus string

const (
    GovernanceDraft     GovernanceStatus = "DRAFT"
    GovernanceReview    GovernanceStatus = "PENDING_REVIEW"
    GovernanceApproved  GovernanceStatus = "APPROVED"
    GovernanceRejected  GovernanceStatus = "REJECTED"
    GovernanceActive    GovernanceStatus = "ACTIVE"
    GovernanceSuperseded GovernanceStatus = "SUPERSEDED"
)

// ComputeFingerprint generates a semantic fingerprint for deduplication
func (r *DraftRule) ComputeFingerprint() Fingerprint {
    // Create canonical representation (domain + condition + action only)
    canonical := struct {
        Domain    string    `json:"domain"`
        Condition Condition `json:"condition"`
        Action    Action    `json:"action"`
    }{
        Domain:    r.Domain,
        Condition: r.Condition,
        Action:    r.Action,
    }

    jsonBytes, _ := json.Marshal(canonical)
    hash := sha256.Sum256(jsonBytes)

    return Fingerprint{
        Hash:      fmt.Sprintf("%x", hash),
        Version:   1,
        CreatedAt: time.Now(),
    }
}
```

---

### 3b.5.2 Unit Normalizer (1 day)

**Why**: Clinical values must be standardized before comparison (CrCl → ml/min, eGFR → ml/min/1.73m², Child-Pugh → A/B/C).

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/unit_normalizer.go`

```go
package extraction

import (
    "fmt"
    "regexp"
    "strings"
)

// UnitNormalizer standardizes clinical units to canonical forms
type UnitNormalizer struct {
    unitMappings      map[string]string
    variableMappings  map[string]string
    gfrPattern        *regexp.Regexp
    childPughPattern  *regexp.Regexp
}

// NewUnitNormalizer creates a normalizer with standard clinical mappings
func NewUnitNormalizer() *UnitNormalizer {
    return &UnitNormalizer{
        unitMappings: map[string]string{
            // Renal function units → ml/min
            "ml/min":            "mL/min",
            "mL/min":            "mL/min",
            "ml/min/1.73m2":     "mL/min/1.73m²",
            "ml/min/1.73m²":     "mL/min/1.73m²",
            "mL/min/1.73m2":     "mL/min/1.73m²",
            "mL/min/1.73m²":     "mL/min/1.73m²",

            // Dose units
            "mg":    "mg",
            "mcg":   "mcg",
            "μg":    "mcg",
            "ug":    "mcg",
            "g":     "g",
            "mg/kg": "mg/kg",

            // Time units
            "hr":    "hour",
            "hrs":   "hour",
            "hour":  "hour",
            "hours": "hour",
            "day":   "day",
            "days":  "day",
            "week":  "week",
            "weeks": "week",

            // Percentage
            "%":       "percent",
            "percent": "percent",
        },
        variableMappings: map[string]string{
            // Renal function
            "crcl":                  "renal_function.crcl",
            "creatinine clearance":  "renal_function.crcl",
            "clcr":                  "renal_function.crcl",
            "egfr":                  "renal_function.egfr",
            "gfr":                   "renal_function.gfr",
            "creatinine":            "renal_function.creatinine",

            // Hepatic function
            "child-pugh":            "hepatic.child_pugh",
            "child pugh":            "hepatic.child_pugh",
            "hepatic impairment":    "hepatic.impairment_level",

            // Age
            "age":                   "patient.age",
            "pediatric":             "patient.age_category",
            "geriatric":             "patient.age_category",

            // Weight
            "weight":                "patient.weight",
            "body weight":           "patient.weight",
            "bsa":                   "patient.bsa",
        },
        gfrPattern:       regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance|ClCr)\s*[<>≤≥]?\s*(\d+(?:\.\d+)?)`),
        childPughPattern: regexp.MustCompile(`(?i)(Child-Pugh\s*([ABC]))|((mild|moderate|severe)\s+hepatic\s+impairment)`),
    }
}

// NormalizedValue represents a standardized clinical value
type NormalizedValue struct {
    Variable       string   `json:"variable"`        // Canonical variable name
    NumericValue   *float64 `json:"numeric_value,omitempty"`
    StringValue    *string  `json:"string_value,omitempty"`
    Unit           string   `json:"unit"`            // Canonical unit
    OriginalText   string   `json:"original_text"`   // Source text for audit
}

// NormalizeUnit converts a unit string to canonical form
func (n *UnitNormalizer) NormalizeUnit(unit string) string {
    normalized := strings.TrimSpace(strings.ToLower(unit))
    if canonical, ok := n.unitMappings[normalized]; ok {
        return canonical
    }
    // Check case-insensitive
    for k, v := range n.unitMappings {
        if strings.EqualFold(k, normalized) {
            return v
        }
    }
    return unit // Return original if no mapping found
}

// NormalizeVariable converts a variable name to canonical form
func (n *UnitNormalizer) NormalizeVariable(variable string) string {
    normalized := strings.TrimSpace(strings.ToLower(variable))
    if canonical, ok := n.variableMappings[normalized]; ok {
        return canonical
    }
    return variable
}

// NormalizeChildPugh converts hepatic impairment descriptions to A/B/C
func (n *UnitNormalizer) NormalizeChildPugh(text string) string {
    lower := strings.ToLower(text)

    // Direct Child-Pugh class
    if strings.Contains(lower, "child-pugh a") || strings.Contains(lower, "child pugh a") {
        return "A"
    }
    if strings.Contains(lower, "child-pugh b") || strings.Contains(lower, "child pugh b") {
        return "B"
    }
    if strings.Contains(lower, "child-pugh c") || strings.Contains(lower, "child pugh c") {
        return "C"
    }

    // Severity-based mapping
    if strings.Contains(lower, "mild hepatic") {
        return "A"
    }
    if strings.Contains(lower, "moderate hepatic") {
        return "B"
    }
    if strings.Contains(lower, "severe hepatic") {
        return "C"
    }

    return text // Return original if no mapping
}

// ParseGFRThreshold extracts GFR threshold from text
func (n *UnitNormalizer) ParseGFRThreshold(text string) (*NormalizedValue, error) {
    matches := n.gfrPattern.FindStringSubmatch(text)
    if len(matches) < 3 {
        return nil, fmt.Errorf("no GFR threshold found in: %s", text)
    }

    var value float64
    fmt.Sscanf(matches[2], "%f", &value)

    return &NormalizedValue{
        Variable:     n.NormalizeVariable(matches[1]),
        NumericValue: &value,
        Unit:         "mL/min",
        OriginalText: matches[0],
    }, nil
}
```

---

### 3b.5.3 Table Normalizer (1.5 days)

**Why**: Detect column roles (condition vs action) and standardize table structure before rule extraction.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/table_normalizer.go`

```go
package extraction

import (
    "strings"
)

// TableNormalizer detects column roles and standardizes table structure
type TableNormalizer struct {
    unitNormalizer    *UnitNormalizer
    conditionPatterns []columnPattern
    actionPatterns    []columnPattern
}

type columnPattern struct {
    keywords []string
    role     ColumnRole
}

// ColumnRole identifies the semantic role of a table column
type ColumnRole string

const (
    RoleCondition    ColumnRole = "CONDITION"     // IF part: GFR, Child-Pugh, age
    RoleAction       ColumnRole = "ACTION"        // THEN part: dose, recommendation
    RoleDrugName     ColumnRole = "DRUG_NAME"     // Drug identifier
    RoleUnknown      ColumnRole = "UNKNOWN"       // Needs manual classification
)

// NormalizedTable represents a table with classified columns
type NormalizedTable struct {
    ID              string                `json:"id"`
    OriginalHeaders []string              `json:"original_headers"`
    NormalizedCols  []NormalizedColumn    `json:"normalized_columns"`
    Rows            []NormalizedRow       `json:"rows"`
    Translatable    bool                  `json:"translatable"`
    UntranslatableReason string           `json:"untranslatable_reason,omitempty"`
}

// NormalizedColumn contains column metadata
type NormalizedColumn struct {
    Index           int        `json:"index"`
    OriginalHeader  string     `json:"original_header"`
    NormalizedName  string     `json:"normalized_name"`
    Role            ColumnRole `json:"role"`
    Unit            string     `json:"unit,omitempty"`
    Confidence      float64    `json:"confidence"` // How confident in role assignment
}

// NormalizedRow contains normalized cell values
type NormalizedRow struct {
    Index  int                  `json:"index"`
    Cells  []NormalizedCell     `json:"cells"`
}

// NormalizedCell contains a normalized value
type NormalizedCell struct {
    ColumnIndex  int               `json:"column_index"`
    OriginalText string            `json:"original_text"`
    Normalized   *NormalizedValue  `json:"normalized,omitempty"`
}

// NewTableNormalizer creates a normalizer with clinical column patterns
func NewTableNormalizer() *TableNormalizer {
    return &TableNormalizer{
        unitNormalizer: NewUnitNormalizer(),
        conditionPatterns: []columnPattern{
            {keywords: []string{"crcl", "gfr", "egfr", "creatinine clearance", "renal"}, role: RoleCondition},
            {keywords: []string{"child-pugh", "hepatic", "liver"}, role: RoleCondition},
            {keywords: []string{"age", "pediatric", "geriatric"}, role: RoleCondition},
            {keywords: []string{"weight", "bsa", "body surface"}, role: RoleCondition},
        },
        actionPatterns: []columnPattern{
            {keywords: []string{"dose", "dosage", "dosing"}, role: RoleAction},
            {keywords: []string{"recommendation", "recommended"}, role: RoleAction},
            {keywords: []string{"adjustment", "adjust"}, role: RoleAction},
            {keywords: []string{"maximum", "max dose"}, role: RoleAction},
            {keywords: []string{"contraindicated", "avoid", "do not use"}, role: RoleAction},
        },
    }
}

// Normalize processes a raw table into a normalized structure
func (n *TableNormalizer) Normalize(headers []string, rows [][]string) (*NormalizedTable, error) {
    result := &NormalizedTable{
        OriginalHeaders: headers,
        Translatable:    true,
    }

    // Step 1: Classify each column
    hasCondition := false
    hasAction := false

    for i, header := range headers {
        col := n.classifyColumn(i, header)
        result.NormalizedCols = append(result.NormalizedCols, col)

        if col.Role == RoleCondition {
            hasCondition = true
        }
        if col.Role == RoleAction {
            hasAction = true
        }
    }

    // Step 2: Check if table is translatable
    if !hasCondition || !hasAction {
        result.Translatable = false
        if !hasCondition {
            result.UntranslatableReason = "NO_CONDITION_COLUMN: Cannot identify IF part"
        } else {
            result.UntranslatableReason = "NO_ACTION_COLUMN: Cannot identify THEN part"
        }
        return result, nil
    }

    // Step 3: Normalize row values
    for rowIdx, row := range rows {
        normalizedRow := NormalizedRow{Index: rowIdx}

        for colIdx, cell := range row {
            normalizedCell := NormalizedCell{
                ColumnIndex:  colIdx,
                OriginalText: cell,
            }

            // Apply normalization based on column role
            if colIdx < len(result.NormalizedCols) {
                col := result.NormalizedCols[colIdx]
                normalizedCell.Normalized = n.normalizeCell(cell, col)
            }

            normalizedRow.Cells = append(normalizedRow.Cells, normalizedCell)
        }

        result.Rows = append(result.Rows, normalizedRow)
    }

    return result, nil
}

// classifyColumn determines the semantic role of a column
func (n *TableNormalizer) classifyColumn(index int, header string) NormalizedColumn {
    headerLower := strings.ToLower(header)

    col := NormalizedColumn{
        Index:          index,
        OriginalHeader: header,
        Role:           RoleUnknown,
        Confidence:     0.0,
    }

    // Check condition patterns
    for _, pattern := range n.conditionPatterns {
        for _, keyword := range pattern.keywords {
            if strings.Contains(headerLower, keyword) {
                col.Role = pattern.role
                col.NormalizedName = n.unitNormalizer.NormalizeVariable(header)
                col.Confidence = 0.9
                return col
            }
        }
    }

    // Check action patterns
    for _, pattern := range n.actionPatterns {
        for _, keyword := range pattern.keywords {
            if strings.Contains(headerLower, keyword) {
                col.Role = pattern.role
                col.NormalizedName = "action." + strings.ReplaceAll(headerLower, " ", "_")
                col.Confidence = 0.9
                return col
            }
        }
    }

    // Unknown column
    col.NormalizedName = strings.ReplaceAll(headerLower, " ", "_")
    col.Confidence = 0.3
    return col
}

func (n *TableNormalizer) normalizeCell(cell string, col NormalizedColumn) *NormalizedValue {
    // Normalize based on column role
    switch col.Role {
    case RoleCondition:
        // Try to parse as GFR threshold
        if nv, err := n.unitNormalizer.ParseGFRThreshold(cell); err == nil {
            return nv
        }
        // Try Child-Pugh normalization
        if childPugh := n.unitNormalizer.NormalizeChildPugh(cell); childPugh != cell {
            return &NormalizedValue{
                Variable:     col.NormalizedName,
                StringValue:  &childPugh,
                OriginalText: cell,
            }
        }
    case RoleAction:
        // Keep as string for action columns
        return &NormalizedValue{
            Variable:     col.NormalizedName,
            StringValue:  &cell,
            OriginalText: cell,
        }
    }

    return &NormalizedValue{
        OriginalText: cell,
    }
}
```

---

### 3b.5.4 Condition-Action Generator (1.5 days)

**Why**: Extract IF/THEN pairs from normalized tables and generate computable conditions.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/condition_action.go`

```go
package extraction

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"

    "github.com/cardiofit/shared/rules"
)

// ConditionActionGenerator creates Condition/Action pairs from normalized tables
type ConditionActionGenerator struct {
    rangePattern    *regexp.Regexp
    operatorPattern *regexp.Regexp
}

// NewConditionActionGenerator creates a generator with extraction patterns
func NewConditionActionGenerator() *ConditionActionGenerator {
    return &ConditionActionGenerator{
        // Matches: "30-60", "30 - 60", "30 to 60"
        rangePattern: regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:-|to)\s*(\d+(?:\.\d+)?)`),
        // Matches: "< 30", "> 60", "<= 15", ">= 90"
        operatorPattern: regexp.MustCompile(`([<>]=?)\s*(\d+(?:\.\d+)?)`),
    }
}

// GeneratedRule represents a rule extracted from a table row
type GeneratedRule struct {
    Condition rules.Condition `json:"condition"`
    Action    rules.Action    `json:"action"`
    RowIndex  int             `json:"row_index"`
    Confidence float64        `json:"confidence"`
}

// GenerateFromTable extracts IF/THEN rules from a normalized table
func (g *ConditionActionGenerator) GenerateFromTable(table *NormalizedTable) ([]GeneratedRule, error) {
    if !table.Translatable {
        return nil, fmt.Errorf("table is not translatable: %s", table.UntranslatableReason)
    }

    var generatedRules []GeneratedRule

    // Find condition and action columns
    var conditionCols, actionCols []int
    for _, col := range table.NormalizedCols {
        if col.Role == RoleCondition {
            conditionCols = append(conditionCols, col.Index)
        }
        if col.Role == RoleAction {
            actionCols = append(actionCols, col.Index)
        }
    }

    // Generate rule for each row
    for _, row := range table.Rows {
        condition, condConf := g.extractCondition(row, table.NormalizedCols, conditionCols)
        action, actConf := g.extractAction(row, table.NormalizedCols, actionCols)

        if condition != nil && action != nil {
            generatedRules = append(generatedRules, GeneratedRule{
                Condition:  *condition,
                Action:     *action,
                RowIndex:   row.Index,
                Confidence: (condConf + actConf) / 2,
            })
        }
    }

    return generatedRules, nil
}

// extractCondition builds a Condition from row cells
func (g *ConditionActionGenerator) extractCondition(row NormalizedRow, cols []NormalizedColumn, conditionIdxs []int) (*rules.Condition, float64) {
    for _, colIdx := range conditionIdxs {
        if colIdx >= len(row.Cells) {
            continue
        }

        cell := row.Cells[colIdx]
        col := cols[colIdx]

        // Try to parse the cell value
        condition := g.parseConditionCell(cell.OriginalText, col.NormalizedName)
        if condition != nil {
            return condition, col.Confidence
        }
    }

    return nil, 0
}

// parseConditionCell converts cell text to a Condition
func (g *ConditionActionGenerator) parseConditionCell(text string, variable string) *rules.Condition {
    text = strings.TrimSpace(text)

    // Check for range: "30-60", "30 to 60"
    if matches := g.rangePattern.FindStringSubmatch(text); len(matches) == 3 {
        min, _ := strconv.ParseFloat(matches[1], 64)
        max, _ := strconv.ParseFloat(matches[2], 64)
        return &rules.Condition{
            Variable: variable,
            Operator: rules.OpBetween,
            MinValue: &min,
            MaxValue: &max,
            Unit:     "mL/min", // Default for GFR
        }
    }

    // Check for operator: "< 30", "> 60"
    if matches := g.operatorPattern.FindStringSubmatch(text); len(matches) == 3 {
        value, _ := strconv.ParseFloat(matches[2], 64)
        op := g.parseOperator(matches[1])
        return &rules.Condition{
            Variable: variable,
            Operator: op,
            Value:    &value,
            Unit:     "mL/min",
        }
    }

    // Check for Child-Pugh class
    lower := strings.ToLower(text)
    if strings.Contains(lower, "child-pugh") || strings.Contains(lower, "hepatic") {
        var class string
        switch {
        case strings.Contains(lower, "a") || strings.Contains(lower, "mild"):
            class = "A"
        case strings.Contains(lower, "b") || strings.Contains(lower, "moderate"):
            class = "B"
        case strings.Contains(lower, "c") || strings.Contains(lower, "severe"):
            class = "C"
        }
        if class != "" {
            return &rules.Condition{
                Variable:    "hepatic.child_pugh",
                Operator:    rules.OpEquals,
                StringValue: &class,
            }
        }
    }

    return nil
}

func (g *ConditionActionGenerator) parseOperator(op string) rules.Operator {
    switch op {
    case "<":
        return rules.OpLessThan
    case ">":
        return rules.OpGreaterThan
    case "<=":
        return rules.OpLessOrEqual
    case ">=":
        return rules.OpGreaterOrEqual
    default:
        return rules.OpEquals
    }
}

// extractAction builds an Action from row cells
func (g *ConditionActionGenerator) extractAction(row NormalizedRow, cols []NormalizedColumn, actionIdxs []int) (*rules.Action, float64) {
    for _, colIdx := range actionIdxs {
        if colIdx >= len(row.Cells) {
            continue
        }

        cell := row.Cells[colIdx]
        col := cols[colIdx]

        action := g.parseActionCell(cell.OriginalText)
        if action != nil {
            return action, col.Confidence
        }
    }

    return nil, 0
}

// parseActionCell converts cell text to an Action
func (g *ConditionActionGenerator) parseActionCell(text string) *rules.Action {
    lower := strings.ToLower(text)

    // Contraindicated
    if strings.Contains(lower, "contraindicated") || strings.Contains(lower, "do not use") {
        return &rules.Action{
            Effect:  rules.EffectContraindicated,
            Message: text,
        }
    }

    // Avoid
    if strings.Contains(lower, "avoid") || strings.Contains(lower, "not recommended") {
        return &rules.Action{
            Effect:  rules.EffectAvoid,
            Message: text,
        }
    }

    // Dose reduction percentage
    if percentMatch := regexp.MustCompile(`(\d+)\s*%`).FindStringSubmatch(text); len(percentMatch) == 2 {
        pct, _ := strconv.ParseFloat(percentMatch[1], 64)
        return &rules.Action{
            Effect: rules.EffectDoseAdjust,
            Adjustment: &rules.DoseAdjustment{
                Type:       rules.AdjustmentPercentage,
                Percentage: &pct,
            },
            Message: text,
        }
    }

    // Specific dose
    if doseMatch := regexp.MustCompile(`(\d+)\s*(mg|mcg|g)`).FindStringSubmatch(lower); len(doseMatch) == 3 {
        return &rules.Action{
            Effect: rules.EffectDoseAdjust,
            Adjustment: &rules.DoseAdjustment{
                Type:         rules.AdjustmentAbsolute,
                AbsoluteDose: &text,
            },
            Message: text,
        }
    }

    // No change / standard dose
    if strings.Contains(lower, "no adjustment") || strings.Contains(lower, "normal dose") || strings.Contains(lower, "no change") {
        return &rules.Action{
            Effect:  rules.EffectNoChange,
            Message: text,
        }
    }

    // Monitoring
    if strings.Contains(lower, "monitor") {
        return &rules.Action{
            Effect:  rules.EffectMonitor,
            Message: text,
        }
    }

    // Generic use with caution
    if strings.Contains(lower, "caution") || strings.Contains(lower, "use with") {
        return &rules.Action{
            Effect:  rules.EffectUseWithCaution,
            Message: text,
        }
    }

    // Default: treat as dose adjustment with message
    return &rules.Action{
        Effect:  rules.EffectDoseAdjust,
        Message: text,
    }
}
```

---

### 3b.5.5 Rule Translator Orchestrator (1.5 days)

**Why**: Coordinate the full pipeline: Classified Table → Normalized Table → Condition/Action → DraftRule.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/rules/rule_translator.go`

```go
package rules

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/cardiofit/shared/extraction"
    "github.com/cardiofit/shared/datasources/dailymed"
)

// RuleTranslator orchestrates the table → rule pipeline
type RuleTranslator struct {
    tableNormalizer   *extraction.TableNormalizer
    conditionActionGen *extraction.ConditionActionGenerator
    fingerprintReg    *FingerprintRegistry
    untranslatableQ   *UntranslatableQueue
}

// NewRuleTranslator creates a configured translator
func NewRuleTranslator(fingerprintReg *FingerprintRegistry, untranslatableQ *UntranslatableQueue) *RuleTranslator {
    return &RuleTranslator{
        tableNormalizer:   extraction.NewTableNormalizer(),
        conditionActionGen: extraction.NewConditionActionGenerator(),
        fingerprintReg:    fingerprintReg,
        untranslatableQ:   untranslatableQ,
    }
}

// TranslationResult contains the outcome of rule translation
type TranslationResult struct {
    Rules            []DraftRule            `json:"rules"`
    UntranslatableTables []UntranslatableTable `json:"untranslatable_tables,omitempty"`
    Stats            TranslationStats       `json:"stats"`
}

// TranslationStats provides metrics on the translation
type TranslationStats struct {
    TablesProcessed     int     `json:"tables_processed"`
    TablesTranslated    int     `json:"tables_translated"`
    TablesUntranslatable int    `json:"tables_untranslatable"`
    RulesGenerated      int     `json:"rules_generated"`
    DuplicatesSkipped   int     `json:"duplicates_skipped"`
    AverageConfidence   float64 `json:"average_confidence"`
}

// UntranslatableTable records a table that couldn't be translated
type UntranslatableTable struct {
    TableID    string   `json:"table_id"`
    Headers    []string `json:"headers"`
    Reason     string   `json:"reason"`
    SourceInfo string   `json:"source_info"`
}

// TranslateClassifiedTables converts classified tables to DraftRules
func (t *RuleTranslator) TranslateClassifiedTables(
    ctx context.Context,
    tables []dailymed.HarvestedTable,
    provenance Provenance,
    domain string,
) (*TranslationResult, error) {
    result := &TranslationResult{}
    var totalConfidence float64

    for _, table := range tables {
        // Skip tables that aren't clinical dosing tables
        if table.Type == dailymed.TableTypeUnknown || table.Type == dailymed.TableTypeAdverseEvent {
            continue
        }

        result.Stats.TablesProcessed++

        // Step 1: Normalize the table
        headers := make([]string, len(table.Headers))
        for i, h := range table.Headers {
            headers[i] = h.Name
        }

        rows := make([][]string, len(table.Rows))
        for i, row := range table.Rows {
            cells := make([]string, 0)
            for _, v := range row.Values {
                cells = append(cells, v)
            }
            rows[i] = cells
        }

        normalized, err := t.tableNormalizer.Normalize(headers, rows)
        if err != nil {
            return nil, fmt.Errorf("normalizing table %s: %w", table.ID, err)
        }

        // Step 2: Check if translatable
        if !normalized.Translatable {
            result.Stats.TablesUntranslatable++

            untranslatable := UntranslatableTable{
                TableID: table.ID,
                Headers: headers,
                Reason:  normalized.UntranslatableReason,
                SourceInfo: fmt.Sprintf("%s/%s", provenance.DocumentID, provenance.SectionCode),
            }
            result.UntranslatableTables = append(result.UntranslatableTables, untranslatable)

            // Queue for human review
            if t.untranslatableQ != nil {
                t.untranslatableQ.Enqueue(ctx, untranslatable, provenance)
            }
            continue
        }

        result.Stats.TablesTranslated++

        // Step 3: Generate condition/action pairs
        generatedRules, err := t.conditionActionGen.GenerateFromTable(normalized)
        if err != nil {
            return nil, fmt.Errorf("generating rules from table %s: %w", table.ID, err)
        }

        // Step 4: Convert to DraftRules with fingerprinting
        for _, generated := range generatedRules {
            draftRule := DraftRule{
                RuleID:           uuid.New(),
                Domain:           domain,
                RuleType:         t.mapTableTypeToRuleType(table.Type),
                Condition:        generated.Condition,
                Action:           generated.Action,
                Provenance:       provenance,
                GovernanceStatus: GovernanceDraft,
                CreatedAt:        time.Now(),
                UpdatedAt:        time.Now(),
            }

            // Compute semantic fingerprint
            draftRule.SemanticFingerprint = draftRule.ComputeFingerprint()

            // Check for duplicates
            if t.fingerprintReg != nil {
                exists, err := t.fingerprintReg.Exists(ctx, draftRule.SemanticFingerprint.Hash)
                if err != nil {
                    return nil, fmt.Errorf("checking fingerprint: %w", err)
                }
                if exists {
                    result.Stats.DuplicatesSkipped++
                    continue
                }

                // Register new fingerprint
                if err := t.fingerprintReg.Register(ctx, draftRule); err != nil {
                    return nil, fmt.Errorf("registering fingerprint: %w", err)
                }
            }

            result.Rules = append(result.Rules, draftRule)
            result.Stats.RulesGenerated++
            totalConfidence += generated.Confidence
        }
    }

    if result.Stats.RulesGenerated > 0 {
        result.Stats.AverageConfidence = totalConfidence / float64(result.Stats.RulesGenerated)
    }

    return result, nil
}

// mapTableTypeToRuleType converts table classification to rule type
func (t *RuleTranslator) mapTableTypeToRuleType(tableType dailymed.TableType) RuleType {
    switch tableType {
    case dailymed.TableTypeGFRDose, dailymed.TableTypeHepaticDose:
        return RuleTypeDosing
    case dailymed.TableTypeInteraction:
        return RuleTypeInteraction
    default:
        return RuleTypeDosing // Default
    }
}
```

---

### 3b.5.6 Rule Fingerprint Engine (1 day)

**Why**: Prevent duplicate rules through semantic hashing. Same clinical meaning = same fingerprint.

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/governance/fingerprint_registry/registry.go`

```go
package fingerprint_registry

import (
    "context"
    "database/sql"
    "time"

    "github.com/cardiofit/shared/rules"
)

// FingerprintRegistry stores and checks semantic fingerprints for deduplication
type FingerprintRegistry struct {
    db *sql.DB
}

// NewFingerprintRegistry creates a registry with database connection
func NewFingerprintRegistry(db *sql.DB) *FingerprintRegistry {
    return &FingerprintRegistry{db: db}
}

// Exists checks if a fingerprint already exists in the registry
func (r *FingerprintRegistry) Exists(ctx context.Context, hash string) (bool, error) {
    var exists bool
    err := r.db.QueryRowContext(ctx,
        "SELECT EXISTS(SELECT 1 FROM fingerprint_registry WHERE hash = $1)",
        hash,
    ).Scan(&exists)
    return exists, err
}

// Register adds a new fingerprint to the registry
func (r *FingerprintRegistry) Register(ctx context.Context, rule rules.DraftRule) error {
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO fingerprint_registry (hash, rule_id, domain, rule_type, created_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (hash) DO NOTHING
    `, rule.SemanticFingerprint.Hash, rule.RuleID, rule.Domain, rule.RuleType, time.Now())
    return err
}

// GetRuleByFingerprint retrieves the rule ID for a fingerprint
func (r *FingerprintRegistry) GetRuleByFingerprint(ctx context.Context, hash string) (*string, error) {
    var ruleID string
    err := r.db.QueryRowContext(ctx,
        "SELECT rule_id FROM fingerprint_registry WHERE hash = $1",
        hash,
    ).Scan(&ruleID)

    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &ruleID, nil
}
```

---

### 3b.5.7 UNTRANSLATABLE Handling (1 day)

**Why**: Tables that can't be translated must go to human review, NOT to LLM (Rule 4).

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/rules/untranslatable.go`

```go
package rules

import (
    "context"
    "database/sql"
    "time"

    "github.com/google/uuid"
)

// UntranslatableQueue handles tables that cannot be automatically translated
// Per Rule 4: Provenance unclear → Draft only, human review required
type UntranslatableQueue struct {
    db *sql.DB
}

// NewUntranslatableQueue creates a queue with database connection
func NewUntranslatableQueue(db *sql.DB) *UntranslatableQueue {
    return &UntranslatableQueue{db: db}
}

// UntranslatableEntry represents a table pending human review
type UntranslatableEntry struct {
    ID               uuid.UUID  `db:"id"`
    TableID          string     `db:"table_id"`
    Headers          []string   `db:"headers"`
    RowCount         int        `db:"row_count"`
    Reason           string     `db:"reason"`          // Why it couldn't be translated
    SourceDocumentID uuid.UUID  `db:"source_document_id"`
    SourceSectionID  *uuid.UUID `db:"source_section_id"`
    SourceInfo       string     `db:"source_info"`     // Human-readable source

    // Review status
    Status           string     `db:"status"`          // PENDING, IN_REVIEW, RESOLVED, DEFERRED
    AssignedTo       *string    `db:"assigned_to"`
    AssignedAt       *time.Time `db:"assigned_at"`

    // Resolution
    Resolution       *string    `db:"resolution"`      // MANUAL_RULES, NOT_CLINICAL, AMBIGUOUS
    ResolutionNotes  *string    `db:"resolution_notes"`
    ResolvedBy       *string    `db:"resolved_by"`
    ResolvedAt       *time.Time `db:"resolved_at"`

    // SLA tracking
    SLADeadline      time.Time  `db:"sla_deadline"`
    SLABreached      bool       `db:"sla_breached"`

    CreatedAt        time.Time  `db:"created_at"`
    UpdatedAt        time.Time  `db:"updated_at"`
}

// Enqueue adds an untranslatable table to the human review queue
func (q *UntranslatableQueue) Enqueue(ctx context.Context, table UntranslatableTable, provenance Provenance) error {
    entry := UntranslatableEntry{
        ID:               uuid.New(),
        TableID:          table.TableID,
        Headers:          table.Headers,
        Reason:           table.Reason,
        SourceDocumentID: provenance.SourceDocumentID,
        SourceSectionID:  provenance.SourceSectionID,
        SourceInfo:       table.SourceInfo,
        Status:           "PENDING",
        SLADeadline:      time.Now().Add(72 * time.Hour), // 72-hour SLA
        CreatedAt:        time.Now(),
        UpdatedAt:        time.Now(),
    }

    _, err := q.db.ExecContext(ctx, `
        INSERT INTO untranslatable_queue
        (id, table_id, headers, reason, source_document_id, source_section_id,
         source_info, status, sla_deadline, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, entry.ID, entry.TableID, entry.Headers, entry.Reason,
       entry.SourceDocumentID, entry.SourceSectionID, entry.SourceInfo,
       entry.Status, entry.SLADeadline, entry.CreatedAt, entry.UpdatedAt)

    return err
}

// GetPending retrieves all pending entries ordered by SLA deadline
func (q *UntranslatableQueue) GetPending(ctx context.Context) ([]UntranslatableEntry, error) {
    rows, err := q.db.QueryContext(ctx, `
        SELECT id, table_id, headers, reason, source_document_id, source_section_id,
               source_info, status, sla_deadline, created_at
        FROM untranslatable_queue
        WHERE status = 'PENDING'
        ORDER BY sla_deadline ASC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var entries []UntranslatableEntry
    for rows.Next() {
        var e UntranslatableEntry
        if err := rows.Scan(&e.ID, &e.TableID, &e.Headers, &e.Reason,
            &e.SourceDocumentID, &e.SourceSectionID, &e.SourceInfo,
            &e.Status, &e.SLADeadline, &e.CreatedAt); err != nil {
            return nil, err
        }
        entries = append(entries, e)
    }

    return entries, nil
}

// Resolve marks an entry as resolved with the given resolution
func (q *UntranslatableQueue) Resolve(ctx context.Context, id uuid.UUID, resolution, notes, resolvedBy string) error {
    now := time.Now()
    _, err := q.db.ExecContext(ctx, `
        UPDATE untranslatable_queue
        SET status = 'RESOLVED', resolution = $1, resolution_notes = $2,
            resolved_by = $3, resolved_at = $4, updated_at = $5
        WHERE id = $6
    `, resolution, notes, resolvedBy, now, now, id)
    return err
}
```

---

### 3b.5.8 Database Migration (0.5 day)

**File**: `backend/shared-infrastructure/knowledge-base-services/migrations/023_draft_rules.sql`

```sql
-- ============================================================================
-- Phase 3b.5: Canonical Rule Generation Tables
-- ============================================================================
-- Supports: DraftRule storage, fingerprint deduplication, untranslatable queue

-- ============================================================================
-- DRAFT RULES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS draft_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Classification
    domain VARCHAR(20) NOT NULL,              -- KB-1, KB-4, KB-5
    rule_type VARCHAR(50) NOT NULL,           -- DOSING, CONTRAINDICATION, INTERACTION

    -- Computable rule structure (JSONB for flexibility)
    condition JSONB NOT NULL,                 -- {variable, operator, value, unit}
    action JSONB NOT NULL,                    -- {effect, adjustment, message}

    -- Full provenance
    source_document_id UUID NOT NULL REFERENCES source_documents(id),
    source_section_id UUID REFERENCES source_sections(id),
    provenance JSONB NOT NULL,                -- Complete extraction lineage

    -- Semantic deduplication
    semantic_fingerprint VARCHAR(64) NOT NULL UNIQUE,  -- SHA256 hash
    fingerprint_version INTEGER DEFAULT 1,

    -- Governance
    governance_status VARCHAR(20) DEFAULT 'DRAFT',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,

    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    superseded_by UUID REFERENCES draft_rules(id),
    supersedes UUID REFERENCES draft_rules(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for draft_rules
CREATE INDEX IF NOT EXISTS idx_draft_rules_domain ON draft_rules(domain);
CREATE INDEX IF NOT EXISTS idx_draft_rules_type ON draft_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_draft_rules_status ON draft_rules(governance_status);
CREATE INDEX IF NOT EXISTS idx_draft_rules_fingerprint ON draft_rules(semantic_fingerprint);
CREATE INDEX IF NOT EXISTS idx_draft_rules_source ON draft_rules(source_document_id);
CREATE INDEX IF NOT EXISTS idx_draft_rules_condition ON draft_rules USING GIN(condition);
CREATE INDEX IF NOT EXISTS idx_draft_rules_action ON draft_rules USING GIN(action);

-- ============================================================================
-- FINGERPRINT REGISTRY
-- ============================================================================
CREATE TABLE IF NOT EXISTS fingerprint_registry (
    hash VARCHAR(64) PRIMARY KEY,             -- SHA256 semantic fingerprint
    rule_id UUID NOT NULL REFERENCES draft_rules(id),
    domain VARCHAR(20) NOT NULL,
    rule_type VARCHAR(50) NOT NULL,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fingerprint_rule ON fingerprint_registry(rule_id);
CREATE INDEX IF NOT EXISTS idx_fingerprint_domain ON fingerprint_registry(domain);

-- ============================================================================
-- UNTRANSLATABLE QUEUE
-- ============================================================================
CREATE TABLE IF NOT EXISTS untranslatable_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Table identification
    table_id VARCHAR(255) NOT NULL,
    headers TEXT[] NOT NULL,
    row_count INTEGER,

    -- Failure reason
    reason VARCHAR(255) NOT NULL,             -- NO_CONDITION_COLUMN, NO_ACTION_COLUMN, etc.

    -- Source lineage
    source_document_id UUID NOT NULL REFERENCES source_documents(id),
    source_section_id UUID REFERENCES source_sections(id),
    source_info TEXT,

    -- Review status
    status VARCHAR(20) DEFAULT 'PENDING',     -- PENDING, IN_REVIEW, RESOLVED, DEFERRED
    assigned_to VARCHAR(100),
    assigned_at TIMESTAMPTZ,

    -- Resolution
    resolution VARCHAR(50),                   -- MANUAL_RULES, NOT_CLINICAL, AMBIGUOUS
    resolution_notes TEXT,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,

    -- SLA tracking
    sla_deadline TIMESTAMPTZ NOT NULL,
    sla_breached BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_untranslatable_status ON untranslatable_queue(status);
CREATE INDEX IF NOT EXISTS idx_untranslatable_sla ON untranslatable_queue(sla_deadline);
CREATE INDEX IF NOT EXISTS idx_untranslatable_source ON untranslatable_queue(source_document_id);

-- ============================================================================
-- HELPER VIEW: Pending Untranslatable with SLA Status
-- ============================================================================
CREATE OR REPLACE VIEW v_pending_untranslatable AS
SELECT
    uq.*,
    sd.drug_name,
    sd.source_type,
    ss.section_name,
    CASE
        WHEN uq.sla_deadline < NOW() THEN TRUE
        ELSE FALSE
    END AS is_sla_breached,
    EXTRACT(EPOCH FROM (uq.sla_deadline - NOW()))/3600 AS hours_until_sla
FROM untranslatable_queue uq
JOIN source_documents sd ON uq.source_document_id = sd.id
LEFT JOIN source_sections ss ON uq.source_section_id = ss.id
WHERE uq.status = 'PENDING'
ORDER BY uq.sla_deadline ASC;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE draft_rules IS 'Canonical computable rules extracted from clinical tables';
COMMENT ON TABLE fingerprint_registry IS 'Semantic fingerprints for rule deduplication';
COMMENT ON TABLE untranslatable_queue IS 'Tables that could not be automatically translated - pending human review';
```

---

### 3b.5 Exit Criteria Checklist

- [ ] **DraftRule schema** implemented with Condition/Action/Provenance
- [ ] **Unit Normalizer** converts CrCl → ml/min, Child-Pugh → A/B/C
- [ ] **Table Normalizer** detects condition vs action columns
- [ ] **Condition-Action Generator** extracts IF/THEN pairs
- [ ] **Rule Translator** orchestrates full pipeline
- [ ] **Fingerprint Registry** prevents duplicate rules
- [ ] **Untranslatable Queue** routes failures to human review (NOT LLM)
- [ ] **Migration 023** creates draft_rules, fingerprint_registry, untranslatable_queue tables
- [ ] Unit tests for metformin renal dosing table → DraftRule conversion
- [ ] Integration test: Classified table → Normalized → DraftRule with fingerprint

---

## Phase 3b.6: KB-16 Lab Reference Ranges Ingestion (Weeks 11.5-13)

### Goal
Transform KB-16 from a basic lab interpretation service into a **Context-Aware Clinical Reference Engine** with conditional reference ranges based on patient context (pregnancy, CKD stage, age, gestational age).

> **Core Insight**: Lab values must be interpreted based on PATIENT CONTEXT, not just standard ranges.
>
> Example: Hemoglobin 11 g/dL
> - Standard range (12-16): **ABNORMAL** ❌
> - Pregnancy T3 range (10.5-14): **NORMAL** ✅
>
> Using wrong range = **dangerous clinical decisions**

### What Already Exists in KB-16

| Component | File | Status |
|-----------|------|--------|
| Basic reference database | `pkg/reference/database.go` | ✅ 40+ tests |
| Authority registry | `pkg/reference/authorities.go` | ✅ 30+ authorities |
| ICMR India-specific ranges | `pkg/reference/icmr_ranges.go` | ✅ Complete |
| Interpretation engine | `pkg/interpretation/engine.go` | ✅ Context-aware |
| Type definitions | `pkg/types/types.go` | ✅ 650+ lines |

### 3b.6.1 Conditional Reference Range Schema (2 days)

**File**: `kb-16-lab-interpretation/pkg/reference/conditional_ranges.go`

```go
// ConditionalReferenceRange represents a reference range with patient conditions
type ConditionalReferenceRange struct {
    ID              uuid.UUID       `json:"id" gorm:"type:uuid;primary_key"`
    LabTestID       uuid.UUID       `json:"lab_test_id" gorm:"type:uuid;not null"`

    // CONDITIONS (all must match for this range to apply)
    Conditions      RangeConditions `json:"conditions" gorm:"embedded"`

    // Reference values
    LowNormal       *float64        `json:"low_normal,omitempty"`
    HighNormal      *float64        `json:"high_normal,omitempty"`
    CriticalLow     *float64        `json:"critical_low,omitempty"`
    CriticalHigh    *float64        `json:"critical_high,omitempty"`

    // Governance
    Authority       string          `json:"authority" gorm:"not null"`
    AuthorityRef    string          `json:"authority_ref" gorm:"not null"`
    SpecificityScore int            `json:"specificity_score" gorm:"default:0"`
}

// RangeConditions defines patient context variables for range selection
type RangeConditions struct {
    Gender          *string   `json:"gender,omitempty"`           // M, F, null=any
    AgeMinYears     *float64  `json:"age_min_years,omitempty"`
    AgeMaxYears     *float64  `json:"age_max_years,omitempty"`
    IsPregnant      *bool     `json:"is_pregnant,omitempty"`
    Trimester       *int      `json:"trimester,omitempty"`        // 1, 2, 3
    CKDStage        *int      `json:"ckd_stage,omitempty"`        // 1-5
    IsOnDialysis    *bool     `json:"is_on_dialysis,omitempty"`
    HoursOfLifeMin  *int      `json:"hours_of_life_min,omitempty"` // For neonates
    HoursOfLifeMax  *int      `json:"hours_of_life_max,omitempty"`
}
```

### 3b.6.2 Range Selection Algorithm (1.5 days)

**File**: `kb-16-lab-interpretation/pkg/reference/range_selector.go`

```go
// SelectRange chooses the most specific matching range for a patient
func (e *ReferenceEngine) SelectRange(ctx context.Context,
    loincCode string, patient *PatientContext) (*ConditionalReferenceRange, error) {

    // 1. Get all ranges for this LOINC code
    ranges, err := e.getRangesForLOINC(ctx, loincCode)

    // 2. Filter to ranges where ALL conditions match patient
    var matching []*ConditionalReferenceRange
    for _, r := range ranges {
        if e.conditionsMatch(&r.Conditions, patient) {
            matching = append(matching, r)
        }
    }

    // 3. If no match, return default (conditions all null)
    if len(matching) == 0 {
        return e.getDefaultRange(ctx, loincCode)
    }

    // 4. Select MOST SPECIFIC (highest SpecificityScore)
    sort.Slice(matching, func(i, j int) bool {
        return matching[i].SpecificityScore > matching[j].SpecificityScore
    })

    return matching[0], nil
}
```

### 3b.6.3 Pregnancy-Specific Ranges (ACOG, ATA) (2 days)

| Test | Unit | T1 (1-13w) | T2 (14-27w) | T3 (28-40w) | Authority |
|------|------|------------|-------------|-------------|-----------|
| Hemoglobin | g/dL | 11.0-14.0 | 10.5-14.0 | 10.5-14.0 | WHO, ACOG |
| Platelets | k/µL | >150 | >100 | >100 | ACOG |
| Creatinine | mg/dL | 0.4-0.7 | 0.4-0.8 | 0.4-0.9 | ACOG |
| TSH | mIU/L | 0.1-2.5 | 0.2-3.0 | 0.3-3.0 | ATA 2017 |
| Fibrinogen | mg/dL | 300-500 | 350-550 | 400-600 | ACOG |

**Clinical Alert**: AST/ALT ≥2× ULN in pregnancy → **HELLP syndrome evaluation**

### 3b.6.4 Renal Function Ranges (KDIGO) (1 day)

| Test | Unit | CKD 1-2 (eGFR ≥60) | CKD 3-4 (eGFR 15-59) | CKD 5 / Dialysis |
|------|------|---------------------|----------------------|------------------|
| Potassium | mEq/L | 3.5-5.0 | 3.5-5.5 | 3.5-6.0 |
| Phosphate | mg/dL | 2.5-4.5 | 2.5-4.5 | 3.5-5.5 |
| Hgb Target | g/dL | 12-16 (F) | 10-12 | 10-11.5 |
| PTH | pg/mL | 15-65 | 35-150 | 150-600 |

### 3b.6.5 Neonatal Bilirubin Nomogram (AAP 2022) (1 day)

| Hours of Life | Low Risk (≥38w) | Medium Risk (35-37w) | High Risk (<35w) |
|---------------|-----------------|----------------------|------------------|
| 24h | 12 mg/dL | 10 mg/dL | 8 mg/dL |
| 48h | 15 mg/dL | 13 mg/dL | 11 mg/dL |
| 72h | 18 mg/dL | 16 mg/dL | 14 mg/dL |
| 96h+ | 20 mg/dL | 18 mg/dL | 15 mg/dL |

**File**: `kb-16-lab-interpretation/pkg/reference/neonatal_bilirubin.go`

```go
// GetBilirubinThreshold returns phototherapy threshold with hour-of-life interpolation
func (e *ReferenceEngine) GetBilirubinThreshold(
    gestationalAge int,
    hoursOfLife int,
    riskFactors []string,
) (*BilirubinThreshold, error) {

    riskCategory := e.determineRiskCategory(gestationalAge, riskFactors)

    // Interpolate between known hour thresholds
    threshold := e.interpolateThreshold(riskCategory, hoursOfLife)

    return threshold, nil
}
```

### 3b.6.6 Database Migration (1 day)

**File**: `kb-16-lab-interpretation/migrations/003_conditional_reference_ranges.sql`

```sql
-- Lab test definitions (LOINC-based)
CREATE TABLE lab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loinc_code VARCHAR(20) NOT NULL UNIQUE,
    test_name VARCHAR(200) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    category VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE
);

-- Conditional reference ranges
CREATE TABLE conditional_reference_ranges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_test_id UUID NOT NULL REFERENCES lab_tests(id),

    -- CONDITIONS
    gender VARCHAR(1),
    age_min_years DECIMAL(5,2),
    age_max_years DECIMAL(5,2),
    is_pregnant BOOLEAN,
    trimester INTEGER,
    ckd_stage INTEGER,
    is_on_dialysis BOOLEAN,
    hours_of_life_min INTEGER,
    hours_of_life_max INTEGER,

    -- REFERENCE VALUES
    low_normal DECIMAL(10,4),
    high_normal DECIMAL(10,4),
    critical_low DECIMAL(10,4),
    critical_high DECIMAL(10,4),

    -- GOVERNANCE
    authority VARCHAR(50) NOT NULL,
    authority_reference TEXT NOT NULL,
    effective_date DATE NOT NULL,
    specificity_score INTEGER DEFAULT 0
);

-- Neonatal bilirubin thresholds (Bhutani nomogram)
CREATE TABLE neonatal_bilirubin_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gestational_age_weeks_min INTEGER NOT NULL,
    gestational_age_weeks_max INTEGER NOT NULL,
    risk_category VARCHAR(20) NOT NULL,
    hour_of_life INTEGER NOT NULL,
    photo_threshold DECIMAL(5,2) NOT NULL,
    exchange_threshold DECIMAL(5,2),
    authority VARCHAR(50) DEFAULT 'AAP',
    UNIQUE(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life)
);
```

### 3b.6 Implementation Tasks

| # | Task | Deliverable | Days | Priority |
|---|------|-------------|------|----------|
| 1 | Database Migration | `003_conditional_reference_ranges.sql` | 1 | P0 |
| 2 | Go Structs & Models | `pkg/reference/conditional_ranges.go` | 1 | P0 |
| 3 | Pregnancy Ranges Ingestion | TSH, Cr, Hgb, Plt, Fibrinogen | 2 | P0 |
| 4 | Renal Function Ranges | K, Phos, PTH by CKD stage | 1 | P0 |
| 5 | Age-Specific Ranges | Neonate, Peds, Adult, Geriatric | 1.5 | P0 |
| 6 | Neonatal Bilirubin Nomogram | Bhutani curves | 1 | P0 |
| 7 | Range Selection Engine | `pkg/reference/range_selector.go` | 1.5 | P0 |
| 8 | Integration with Interpreter | Update `engine.go` | 1 | P0 |
| 9 | Unit + Integration Tests | >85% coverage | 2 | P1 |

**Total Estimated Effort**: 12 days (can be parallelized to ~7 days)

### 3b.6 Exit Criteria Checklist

- [ ] **Conditional reference range schema** deployed and populated
- [ ] **Pregnancy-specific ranges** for TSH, Cr, Hgb, Plt, Fibrinogen (by trimester)
- [ ] **CKD-stage-specific targets** for K, Phos, PTH, Hgb
- [ ] **Age-stratified ranges** (neonate, pediatric, adult, geriatric)
- [ ] **Neonatal bilirubin nomogram** with hour-of-life interpolation
- [ ] **Range selection engine** correctly picks most-specific matching range
- [ ] **All ranges have governance** (authority, reference, effective date)
- [ ] **>85% test coverage** for range selection logic
- [ ] **NO LLM used** — all ranges from structured guideline tables

> **Key Principle**: KB-16 ingestion is FULLY DETERMINISTIC. All reference ranges come from structured tables in authoritative guidelines — NO LLM extraction needed.

---

## Phase 3c: Consensus Grid & Gap-Filling (Weeks 14-15)

### Goal
Build the LLM extraction layer with strict consensus requirements.

### 3c.1 LLM Provider Interface (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/llm/providers.go`

```go
package llm

import (
    "context"
)

// Provider defines the interface for LLM providers
type Provider interface {
    Name() string
    Extract(ctx context.Context, req *ExtractionRequest) (*ExtractionResult, error)
    SupportsStructuredOutput() bool
}

// ExtractionRequest defines what to extract from source text
type ExtractionRequest struct {
    FactType    string            // "RENAL_DOSE_ADJUST", "HEPATIC_DOSE_ADJUST", etc.
    SourceText  string            // The SPL section text
    Schema      *ExtractionSchema // Expected output structure
    DrugContext *DrugContext      // Drug info for context
}

type DrugContext struct {
    RxCUI       string
    DrugName    string
    GenericName string
    DrugClass   string
}

// ExtractionResult contains the LLM's extraction
type ExtractionResult struct {
    Provider    string
    FactType    string
    ExtractedData interface{}      // Matches schema
    Confidence  float64           // Provider's confidence
    Citations   []Citation        // Where in source text
    RawResponse string            // For debugging
    Latency     time.Duration
}

type Citation struct {
    StartOffset int
    EndOffset   int
    QuotedText  string
}

// ClaudeProvider implements Provider for Anthropic Claude
type ClaudeProvider struct {
    apiKey     string
    model      string // "claude-3-opus", "claude-3-sonnet"
    httpClient *http.Client
}

// GPT4Provider implements Provider for OpenAI GPT-4
type GPT4Provider struct {
    apiKey     string
    model      string // "gpt-4-turbo", "gpt-4o"
    httpClient *http.Client
}
```

---

### 3c.2 Shadow Renbase Extractor (3 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/renal/shadow_renbase.go`

```go
package renal

import (
    "context"
    "fmt"

    "github.com/cardiofit/shared/datasources/dailymed"
    "github.com/cardiofit/shared/extraction/spl"
    "github.com/cardiofit/shared/extraction/llm"
    "github.com/cardiofit/shared/extraction/consensus"
)

// ShadowRenbaseExtractor builds renal dosing from FDA SPL
type ShadowRenbaseExtractor struct {
    splFetcher       *dailymed.SPLFetcher
    loincParser      *spl.LOINCSectionParser
    consensusEngine  *consensus.Engine
    factStore        FactStoreWriter
}

// RenalDoseAdjustment represents extracted renal dosing information
type RenalDoseAdjustment struct {
    RxCUI           string
    DrugName        string
    GFRBands        []GFRBand
    DialysisGuidance *DialysisGuidance
    ExtractionType  ExtractionType
    Confidence      float64
    SourceSetID     string
    SourceSection   string // LOINC code
}

type GFRBand struct {
    MinGFR          float64  // inclusive
    MaxGFR          float64  // exclusive, 999 = no upper limit
    RecommendedDose string   // "50% of normal", "250mg BID"
    Action          string   // "REDUCE", "AVOID", "CONTRAINDICATED", "NO_CHANGE"
}

type DialysisGuidance struct {
    Hemodialysis     string // "Supplement dose post-HD", "No supplement needed"
    PeritonealDialysis string
    CRRT             string // Continuous renal replacement therapy
}

type ExtractionType string

const (
    ExtractionTable  ExtractionType = "TABLE_PARSE"     // High confidence
    ExtractionRegex  ExtractionType = "REGEX_PARSE"     // Medium confidence
    ExtractionLLM    ExtractionType = "LLM_CONSENSUS"   // Requires 2-of-3
    ExtractionNone   ExtractionType = "NO_DATA"         // Not available
)

// Extract retrieves renal dosing for a drug
func (e *ShadowRenbaseExtractor) Extract(ctx context.Context, rxcui string) (*RenalDoseAdjustment, error) {
    // Step 1: Fetch SPL document
    splDoc, err := e.splFetcher.FetchByRxCUI(ctx, rxcui)
    if err != nil {
        return nil, fmt.Errorf("fetching SPL: %w", err)
    }

    // Step 2: Get Dosage & Administration section (LOINC 34068-7)
    dosageSection := splDoc.GetSection(dailymed.LOINCDosageAdministration)
    if dosageSection == nil {
        return &RenalDoseAdjustment{
            RxCUI:          rxcui,
            ExtractionType: ExtractionNone,
        }, nil
    }

    // Step 3: Parse section
    parsed, err := e.loincParser.ParseDosageSection(dosageSection)
    if err != nil {
        return nil, fmt.Errorf("parsing dosage section: %w", err)
    }

    // Step 4: Decide extraction path based on Navigation Rules
    if parsed.CanExtractWithoutLLM() {
        // Rule 2: Table exists → PARSE, don't interpret
        return e.buildFromParsed(rxcui, splDoc, parsed)
    }

    // Step 5: LLM gap-filling required (Rule 3: Consensus required)
    return e.extractWithLLM(ctx, rxcui, splDoc, dosageSection)
}

func (e *ShadowRenbaseExtractor) extractWithLLM(ctx context.Context, rxcui string, splDoc *dailymed.SPLDocument, section *dailymed.SPLSection) (*RenalDoseAdjustment, error) {
    req := &llm.ExtractionRequest{
        FactType:   "RENAL_DOSE_ADJUST",
        SourceText: section.Text.Content,
        Schema:     renalDoseSchema,
        DrugContext: &llm.DrugContext{
            RxCUI: rxcui,
        },
    }

    // Run Race-to-Consensus
    result, err := e.consensusEngine.Extract(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("consensus extraction: %w", err)
    }

    if !result.Achieved {
        // Rule 3: LLMs disagree → HUMAN first
        return &RenalDoseAdjustment{
            RxCUI:          rxcui,
            ExtractionType: ExtractionLLM,
            Confidence:     result.MaxConfidence,
            // Mark for human review
        }, nil
    }

    return e.buildFromConsensus(rxcui, splDoc, result)
}
```

---

### 3c.3 Race-to-Consensus Engine (3 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/consensus/engine.go`

```go
package consensus

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/cardiofit/shared/extraction/llm"
)

// Engine coordinates multi-LLM extraction with consensus requirements
type Engine struct {
    providers     []llm.Provider
    minAgreement  int           // Minimum providers that must agree (default: 2)
    timeout       time.Duration
    diffThreshold float64       // Max difference in numeric values to consider "agreement"
}

// NewEngine creates a consensus engine with default settings
func NewEngine(providers []llm.Provider) *Engine {
    return &Engine{
        providers:     providers,
        minAgreement:  2,
        timeout:       30 * time.Second,
        diffThreshold: 0.05, // 5% difference allowed for numeric agreement
    }
}

// Result contains the consensus extraction outcome
type Result struct {
    Achieved       bool
    AgreementCount int
    WinningValue   interface{}
    Confidence     float64
    MaxConfidence  float64
    Disagreements  []Disagreement
    RequiresHuman  bool
    ProviderResults []llm.ExtractionResult
}

type Disagreement struct {
    Field     string
    Provider1 string
    Value1    interface{}
    Provider2 string
    Value2    interface{}
    Severity  string // "CRITICAL", "MINOR"
}

// Extract runs parallel extraction across all providers and checks consensus
func (e *Engine) Extract(ctx context.Context, req *llm.ExtractionRequest) (*Result, error) {
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()

    // Run providers in parallel
    results := make([]llm.ExtractionResult, len(e.providers))
    errors := make([]error, len(e.providers))
    var wg sync.WaitGroup

    for i, provider := range e.providers {
        wg.Add(1)
        go func(idx int, p llm.Provider) {
            defer wg.Done()
            result, err := p.Extract(ctx, req)
            if err != nil {
                errors[idx] = err
                return
            }
            results[idx] = *result
        }(i, provider)
    }

    wg.Wait()

    // Check how many succeeded
    var successResults []llm.ExtractionResult
    for i, err := range errors {
        if err == nil {
            successResults = append(successResults, results[i])
        }
    }

    if len(successResults) < e.minAgreement {
        return &Result{
            Achieved:       false,
            AgreementCount: len(successResults),
            RequiresHuman:  true,
        }, fmt.Errorf("insufficient provider responses: %d/%d", len(successResults), e.minAgreement)
    }

    // Check consensus
    return e.checkConsensus(successResults)
}

func (e *Engine) checkConsensus(results []llm.ExtractionResult) (*Result, error) {
    // Compare extracted values across providers
    agreementGroups := e.groupByAgreement(results)

    // Find largest agreement group
    var largestGroup []llm.ExtractionResult
    for _, group := range agreementGroups {
        if len(group) > len(largestGroup) {
            largestGroup = group
        }
    }

    consensus := &Result{
        Achieved:        len(largestGroup) >= e.minAgreement,
        AgreementCount:  len(largestGroup),
        ProviderResults: results,
    }

    if consensus.Achieved {
        // Use highest confidence result from agreeing providers
        var maxConf float64
        for _, r := range largestGroup {
            if r.Confidence > maxConf {
                maxConf = r.Confidence
                consensus.WinningValue = r.ExtractedData
                consensus.Confidence = r.Confidence
            }
        }
        consensus.MaxConfidence = maxConf
    } else {
        // Identify disagreements for human review
        consensus.Disagreements = e.findDisagreements(results)
        consensus.RequiresHuman = true
    }

    return consensus, nil
}

func (e *Engine) groupByAgreement(results []llm.ExtractionResult) [][]llm.ExtractionResult {
    // Group results by semantic equivalence
    // Implementation: Compare JSON representations with tolerance for numeric values
    // ... (detailed comparison logic)
    return nil // Placeholder
}
```

---

### 3c Exit Criteria Checklist

- [ ] Claude and GPT-4 providers implemented
- [ ] Shadow Renbase extracts GFR-based dosing from SPL
- [ ] Consensus engine enforces 2-of-3 agreement
- [ ] Disagreements trigger human escalation queue
- [ ] Confidence scoring reflects extraction method
- [ ] End-to-end test: Extract renal dosing for metformin

---

## Phase 3d: Governance Integration & Production (Weeks 14-15)

### Goal
Integrate with KB-0 governance and deploy to production.

### 3d.1 Navigation Rules Implementation (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/governance/navigation_rules.go`

```go
package governance

import (
    "context"
    "fmt"
)

// NavigationRuleEngine enforces the 4 non-negotiable rules
type NavigationRuleEngine struct {
    authorityRegistry *AuthorityRegistry
    factStore         FactStoreReader
}

// Rule represents a navigation rule
type Rule interface {
    ID() int
    Name() string
    Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error)
}

// RuleDecision indicates what action to take
type RuleDecision struct {
    RuleID       int
    Action       Action
    Reason       string
    AuthorityHit *AuthoritySource // If fact found in authority
    AllowLLM     bool
}

type Action string

const (
    ActionBlockLLM    Action = "BLOCK_LLM"     // Rule 1: Authority exists
    ActionParseTable  Action = "PARSE_TABLE"   // Rule 2: Structured data exists
    ActionHumanReview Action = "HUMAN_REVIEW"  // Rule 3: LLMs disagree
    ActionDraftOnly   Action = "DRAFT_ONLY"    // Rule 4: Unclear provenance
    ActionAllowLLM    Action = "ALLOW_LLM"     // All rules pass, LLM permitted
)

// Rule1 checks if fact exists in a curated authority
type Rule1AuthorityCheck struct {
    registry *AuthorityRegistry
}

func (r *Rule1AuthorityCheck) ID() int { return 1 }
func (r *Rule1AuthorityCheck) Name() string { return "Authority Existence Check" }

func (r *Rule1AuthorityCheck) Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
    // Check if this fact type has an authoritative source
    authority, found := r.registry.GetAuthorityForFactType(req.FactType)
    if !found {
        return &RuleDecision{
            RuleID:   1,
            Action:   ActionAllowLLM,
            AllowLLM: true,
            Reason:   "No authority defined for this fact type",
        }, nil
    }

    // Check if authority has data for this drug
    fact, err := authority.GetFact(ctx, req.RxCUI, req.FactType)
    if err != nil {
        return nil, fmt.Errorf("checking authority: %w", err)
    }

    if fact != nil {
        return &RuleDecision{
            RuleID:       1,
            Action:       ActionBlockLLM,
            AllowLLM:     false,
            Reason:       fmt.Sprintf("Fact exists in authority: %s", authority.Name()),
            AuthorityHit: &AuthoritySource{Name: authority.Name(), FactID: fact.ID},
        }, nil
    }

    return &RuleDecision{
        RuleID:   1,
        Action:   ActionAllowLLM,
        AllowLLM: true,
        Reason:   "Fact not found in authority, LLM gap-filling permitted",
    }, nil
}

// Evaluate runs all 4 rules in order
func (e *NavigationRuleEngine) Evaluate(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
    rules := []Rule{
        &Rule1AuthorityCheck{registry: e.authorityRegistry},
        &Rule2TableCheck{},
        &Rule3ConsensusCheck{},
        &Rule4ProvenanceCheck{},
    }

    for _, rule := range rules {
        decision, err := rule.Check(ctx, req)
        if err != nil {
            return nil, fmt.Errorf("rule %d failed: %w", rule.ID(), err)
        }

        // If rule blocks LLM, stop and return
        if !decision.AllowLLM {
            return decision, nil
        }
    }

    // All rules passed, LLM is permitted
    return &RuleDecision{
        Action:   ActionAllowLLM,
        AllowLLM: true,
        Reason:   "All navigation rules passed",
    }, nil
}
```

---

### 3d.2 6-Gate Governance Pipeline (3 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/governance/pipeline.go`

### 3d.3 NeMo Guardrails Integration (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/guardrails/nemo.go`

### 3d.4 Drift Monitoring Dashboard (2 days)

**File**: `backend/shared-infrastructure/knowledge-base-services/shared/monitoring/drift_dashboard.go`

---

### 3d Exit Criteria Checklist

- [ ] Navigation Rules enforce authority-first extraction
- [ ] 6-Gate Pipeline integrates with KB-0
- [ ] NeMo Guardrails validate LLM outputs
- [ ] Drift monitoring tracks confidence distributions
- [ ] All facts have complete audit trail
- [ ] Staging deployment validated
- [ ] Production deployment complete

---

## Dependency Graph

```
Phase 3a (Foundation) - Weeks 7-8
├── RxNav-in-a-Box ──────────────────────────────────────────┐
├── MedXN ─────────────────────────────────────────────────────┤
├── DailyMed SPL Fetcher ──────────────────────────────────────┤
├── LOINC Section Parser ──────────────────────────────────────┤
└── Tabular Harvester ─────────────────────────────────────────┘
                                                                  │
Phase 3b (Ground Truth) - Weeks 9-10                              │
├── CPIC Client ──────────────────────────────────────────────────┤
├── CredibleMeds Client ──────────────────────────────────────────┤
├── LiverTox Ingestion ───────────────────────────────────────────┤
├── LactMed Ingestion ────────────────────────────────────────────┤
├── DrugBank Loader ──────────────────────────────────────────────┤
└── OHDSI Loader ─────────────────────────────────────────────────┘
                                                                  │
                                                                  ▼
Phase 3b.5 (Canonical Rules) - Weeks 10.5-11 ◄────────────────────┤
├── DraftRule Schema ─────────────────────────────────────────────┤
├── Unit Normalizer ──────────────────────────────────────────────┤
├── Table Normalizer ─────────────────────────────────────────────┤
├── Condition-Action Generator ───────────────────────────────────┤
├── Rule Translator ──────────────────────────────────────────────┤
├── Fingerprint Registry ─────────────────────────────────────────┤
└── Untranslatable Queue ─────────────────────────────────────────┘
                                                                  │
                                                                  ▼
Phase 3c (Consensus) - Weeks 12-13                                │
├── LLM Providers ────────────────────────────────────────────────┤
├── Shadow Renbase ───────────────────────────────────────────────┤
├── Consensus Engine ─────────────────────────────────────────────┤
├── Confidence Calibrator ────────────────────────────────────────┤
└── Human Escalation ─────────────────────────────────────────────┘
                                                                  │
                                                                  ▼
Phase 3d (Governance) - Weeks 14-15                               │
├── Navigation Rules ◄────────────────────────────────────────────┤
├── 6-Gate Pipeline ◄─────────────────────────────────────────────┤
├── NeMo Guardrails ◄─────────────────────────────────────────────┤
└── Drift Monitoring ◄────────────────────────────────────────────┘
```

### Key Dependencies

| Component | Depends On |
|-----------|------------|
| Rule Translator | Table Normalizer, Condition-Action Generator |
| Fingerprint Registry | DraftRule Schema |
| Shadow Renbase | Rule Translator, Consensus Engine |
| Navigation Rules | All Phase 3b.5 components |

---

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| CredibleMeds API access denied | Medium | High | Fallback to literature-based QT lists |
| LLM rate limits during extraction | High | Medium | Implement retry with exponential backoff |
| Consensus never achieved | Low | High | Human escalation queue with SLA |
| SPL format changes | Low | Medium | Version detection in parser |
| Authority data staleness | Medium | Medium | Delta sync with change detection |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Authority coverage | >85% of facts from authorities | FactStore provenance analysis |
| LLM usage rate | <15% of total extractions | Extraction type distribution |
| Consensus achievement | >90% when LLM used | Consensus engine logs |
| Human escalation rate | <5% of extractions | Escalation queue metrics |
| Confidence calibration | Predicted vs actual accuracy within 5% | Validation sample review |
| End-to-end latency | <2s for single drug lookup | API response times |

---

## Appendix: File Structure

```
backend/shared-infrastructure/knowledge-base-services/
├── shared/
│   ├── datasources/
│   │   ├── interfaces.go           # ✅ Exists
│   │   ├── dailymed/
│   │   │   ├── fetcher.go          # 🔴 NEW - SPL fetcher
│   │   │   ├── cache.go            # 🔴 NEW - SPL cache
│   │   │   └── table_classifier.go # ✅ Exists - Table classification
│   │   ├── cpic/
│   │   │   └── client.go           # 🔴 NEW - CPIC API
│   │   ├── crediblemeds/
│   │   │   └── client.go           # 🔴 NEW - CredibleMeds API
│   │   ├── livertox/
│   │   │   └── ingest.go           # 🔴 NEW - LiverTox XML
│   │   ├── lactmed/
│   │   │   └── ingest.go           # 🔴 NEW - LactMed XML
│   │   ├── drugbank/
│   │   │   └── loader.go           # 🔴 NEW - DrugBank loader
│   │   └── ohdsi/
│   │       └── beers.go            # 🔴 NEW - Beers/STOPP
│   ├── extraction/
│   │   ├── spl/
│   │   │   ├── loinc_parser.go     # 🔴 NEW - LOINC section parser
│   │   │   └── tabular_harvester.go # ✅ Exists - Table → JSON (partial)
│   │   ├── unit_normalizer.go      # 🟡 Phase 3b.5 - Unit standardization
│   │   ├── table_normalizer.go     # 🟡 Phase 3b.5 - Column role detection
│   │   ├── condition_action.go     # 🟡 Phase 3b.5 - IF/THEN extraction
│   │   ├── llm/
│   │   │   ├── providers.go        # 🔴 NEW - Claude, GPT-4 adapters
│   │   │   └── schemas.go          # 🔴 NEW - Extraction schemas
│   │   ├── renal/
│   │   │   └── shadow_renbase.go   # 🔴 NEW - Shadow Renbase
│   │   ├── consensus/
│   │   │   └── engine.go           # 🔴 NEW - 2-of-3 consensus
│   │   ├── calibration/
│   │   │   └── scorer.go           # 🔴 NEW - Confidence calibration
│   │   └── guardrails/
│   │       └── nemo.go             # 🔴 NEW - NeMo Guardrails
│   ├── rules/                      # 🟡 Phase 3b.5 - NEW DIRECTORY
│   │   ├── draft_rule.go           # 🟡 Phase 3b.5 - DraftRule schema
│   │   ├── rule_translator.go      # 🟡 Phase 3b.5 - Table → Rule orchestrator
│   │   └── untranslatable.go       # 🟡 Phase 3b.5 - Failure handling
│   ├── governance/
│   │   ├── navigation_rules.go     # ✅ Exists - 4 rules implementation
│   │   ├── pipeline.go             # ✅ Exists - 6-gate pipeline
│   │   ├── authority_router.go     # ✅ Exists - Fact-type routing
│   │   ├── fingerprint_registry/   # 🟡 Phase 3b.5 - NEW DIRECTORY
│   │   │   ├── registry.go         # 🟡 Phase 3b.5 - Semantic deduplication
│   │   │   └── provenance.go       # 🟡 Phase 3b.5 - Provenance tracking
│   │   └── escalation/
│   │       └── queue.go            # 🔴 NEW - Human review queue
│   └── monitoring/
│       └── drift_dashboard.go      # 🔴 NEW - Drift monitoring
├── migrations/
│   ├── 021_source_centric_model.sql  # ✅ Exists - source_documents, source_sections, derived_facts
│   └── 023_draft_rules.sql           # 🟡 Phase 3b.5 - draft_rules, fingerprint_registry, untranslatable_queue
├── docker/
│   ├── rxnav/
│   │   └── docker-compose.yml      # 🔴 NEW - RxNav-in-a-Box
│   └── medxn/
│       └── Dockerfile              # 🔴 NEW - MedXN container
└── kb-1-drug-rules/
    └── pkg/ingestion/fda/
        └── client.go               # ✅ Exists - enhance for SPL
```

### Legend
- ✅ **Exists**: Already implemented
- 🟡 **Phase 3b.5**: New files for Canonical Rule Generation
- 🔴 **NEW**: Other new files (not yet implemented)

---

*Document Version: 1.1*
*Created: 2026-01-24*
*Updated: 2026-01-25*
*Based on: Phase 3 Truth Sourcing Manifest + Phase3b5_Canonical_Rule_Generation.docx*

### Change Log

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-24 | Initial Phase 3 Implementation Plan |
| 1.1 | 2026-01-25 | Added Phase 3b.5 Canonical Rule Generation (DraftRule, Normalizers, Translator, Fingerprint, Untranslatable Queue). Extended timeline from 8 to 9 weeks. |

---

## Phase 3d: Truth Arbitration Engine (NEW)

> **Timeline**: Days 1-14
> **Status**: ✅ IMPLEMENTED
> **Philosophy**: *"When truths collide, precedence decides."*

Phase 3d implements the **Truth Arbitration Engine** - a deterministic conflict resolution system that reconciles disagreements between Phase 3b.5 (Canonical Rules), Phase 3b.6 (Lab Interpretations), and Authority Facts.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    TRUTH ARBITRATION ENGINE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INPUT SOURCES                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ REGULATORY   │  │ AUTHORITY    │  │ LAB          │           │
│  │ (FDA BBW)    │  │ (CPIC)       │  │ (KB-16)      │           │
│  │ Trust: 1.00  │  │ Trust: 1.00  │  │ Trust: 0.95  │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│         │                 │                 │                    │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │ RULE         │  │ LOCAL        │                             │
│  │ (Phase 3b.5) │  │ (Hospital)   │                             │
│  │ Trust: 0.90  │  │ Trust: 0.80  │                             │
│  └──────┬───────┘  └──────┬───────┘                             │
│         │                 │                                      │
│         └────────────┬────┴────────────────┘                    │
│                      ▼                                           │
│         ┌────────────────────────┐                              │
│         │   CONFLICT DETECTOR    │                              │
│         │  • 6 conflict types    │                              │
│         │  • Severity scoring    │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│         ┌────────────────────────┐                              │
│         │   PRECEDENCE ENGINE    │                              │
│         │  • P0-P7 rule ladder   │                              │
│         │  • Resolution matrix   │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│         ┌────────────────────────┐                              │
│         │  DECISION SYNTHESIZER  │                              │
│         │  • Final verdict       │                              │
│         │  • Confidence score    │                              │
│         │  • Audit trail         │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│              ┌──────────────┐                                   │
│              │   DECISION   │                                   │
│              │  ACCEPT │ BLOCK │ OVERRIDE │ DEFER │ ESCALATE   │
│              └──────────────┘                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Precedence Rules (P0-P7)

> **P0 Physiology Supremacy**: *"No dosing or safety rule may override a KB-16 CRITICAL or PANIC classification."*

| Rule | Description | Winner | Rationale |
|------|-------------|--------|-----------|
| **P0** | CRITICAL/PANIC lab values = immediate BLOCK | LAB (KB-16) | Physiology supersedes all |
| **P1** | REGULATORY_BLOCK always wins | REGULATORY | Legal requirement |
| **P2** | DEFINITIVE authority > PRIMARY authority | Higher level | Evidence hierarchy |
| **P3** | AUTHORITY_FACT > CANONICAL_RULE (same drug) | AUTHORITY | Curated > extracted |
| **P4** | LAB critical + RULE triggered = ESCALATE | ESCALATE | Real-time validation |
| **P5** | More provenance sources > fewer sources | Higher count | Consensus strength |
| **P6** | LOCAL_POLICY can override rules, NOT authorities | Conditional | Site autonomy + safety |
| **P7** | More restrictive action wins ties | Stricter | Fail-safe default |

#### P0: Physiology Supremacy (NEW - 2026-01-26)

KB-16 Lab Interpretation is elevated to **P0 Authority** status. When a lab value reaches CRITICAL or PANIC classification, this represents immediate physiological danger that supersedes ALL other rules - including authorities, canonical rules, and local policies.

**Classification Hierarchy:**
| Classification | Action | Example |
|----------------|--------|---------|
| `PANIC_HIGH` / `PANIC_LOW` | Immediate BLOCK | K+ > 6.5 mmol/L, INR > 5.0 |
| `CRITICAL` | Immediate BLOCK | AST > 70 U/L (Pregnancy T3), eGFR < 15 |
| `ABNORMAL` / `HIGH` / `LOW` | Proceed to P1-P7 | Standard arbitration |

**Clinical Accuracy Thresholds:**
- **eGFR**: CRITICAL/PANIC = < 15 (Stage 5/ESRD), ABNORMAL = 15-59 (Stage 3-4 CKD)
- **INR**: PANIC = > 5.0-6.0, HIGH = 3.0-5.0, THERAPEUTIC = 2.0-3.0
- **Potassium**: PANIC_HIGH = > 6.5 mmol/L, HIGH = 5.1-6.5, NORMAL = 3.5-5.0

**Override Policy:**
| Level | Override Capability |
|-------|---------------------|
| P0 (Physiology) | Only by attending physician attestation with documented rationale |
| P1 (Regulatory) | Cannot be overridden programmatically (legal/FDA requirement) |
| P2-P7 | Standard arbitration hierarchy applies |

### Decision Outcomes

| Decision | Meaning | Clinical Action |
|----------|---------|-----------------|
| **ACCEPT** | All sources agree or no conflicts | Proceed |
| **BLOCK** | Hard constraint violated | Cannot proceed |
| **OVERRIDE** | Soft conflict, can proceed with acknowledgment | Warning + documentation |
| **DEFER** | Insufficient data | Request more information |
| **ESCALATE** | Complex conflict | Route to expert review |

### Conflict Types

| Conflict Type | Example | Frequency | Severity |
|---------------|---------|-----------|----------|
| `RULE_VS_AUTHORITY` | SPL "avoid" vs CPIC "contraindicated" | Common | MEDIUM |
| `RULE_VS_LAB` | Rule: CrCl < 30, Lab: eGFR = 28 | Common | HIGH |
| `AUTHORITY_VS_LAB` | CPIC: eGFR < 30, Lab: normal for pregnancy | Rare | CRITICAL |
| `AUTHORITY_VS_AUTHORITY` | CPIC vs CredibleMeds on same drug | Rare | HIGH |
| `RULE_VS_RULE` | Two SPLs with different thresholds | Common | LOW |
| `LOCAL_VS_ANY` | Hospital policy overrides guideline | Common | MEDIUM |

### Implementation Files

| File | Location | Purpose |
|------|----------|---------|
| `004_truth_arbitration.sql` | `migrations/` | Database schema + seed data |
| `types.go` | `pkg/arbitration/` | Source, Decision, Conflict enums |
| `schemas.go` | `pkg/arbitration/` | Input/Output schemas |
| `precedence_engine.go` | `pkg/arbitration/` | P0-P7 rule implementation |
| `conflict_detector.go` | `pkg/arbitration/` | Pairwise conflict detection |
| `decision_synthesizer.go` | `pkg/arbitration/` | Final decision logic |
| `engine.go` | `pkg/arbitration/` | Main orchestration + P0 physiology check |
| `decision_explainability.go` | `pkg/arbitration/` | Human-readable explanation engine |
| `arbitration_test.go` | `tests/` | Comprehensive unit tests |
| `arbitration_scenarios_test.go` | `tests/` | Clinical scenarios including P0 tests |
| `arbitration_additional_test.go` | `tests/` | Benign drug tests + P4 lab escalation tests |
| `IMPLEMENTATION_SUMMARY.md` | `docs/` | Complete implementation documentation |

### Test Scenario: Metformin + Renal Impairment

```
Patient: 68yo female, eGFR = 28
Intent: PRESCRIBE Metformin 500mg BID

Inputs:
- FDA SPL: IF CrCl < 30 THEN Avoid (Trust: 0.90)
- CPIC: eGFR < 30 = Contraindicated (Trust: 1.00)
- KB-16: eGFR = 28 (ABNORMAL - Stage 4 CKD, NOT CRITICAL*)
- Hospital: Allow if eGFR 25-30 with monitoring (Trust: 0.80)

*Clinical Note: eGFR 28 is Stage 4 CKD. CRITICAL/PANIC would be <15 (Stage 5/ESRD).

Conflicts:
- C1: SPL vs CPIC → CPIC wins (P3)
- C2: CPIC vs Hospital → CPIC wins (P6: Local ≠ Authority)
- C3: Lab (ABNORMAL) validates rule → Standard P4 consideration

Final Decision: BLOCK
Confidence: 1.00
Precedence Rule: P3
Winning Source: AUTHORITY (CPIC)
Rationale: CPIC DEFINITIVE contraindication cannot be overridden by local policy
```

### Test Scenario: P0 Physiology Supremacy (NEW - 2026-01-26)

```
Patient: 68yo female, CKD Stage 4, eGFR = 22
Intent: PRESCRIBE Lisinopril 10mg QD
Lab: K+ = 6.8 mmol/L (PANIC_HIGH)

Inputs:
- CPIC: ACE inhibitors allowed in CKD Stage 4 (Trust: 1.00)
- Canonical Rule: Monitor potassium in CKD (Trust: 0.90)
- KB-16: K+ = 6.8 (PANIC_HIGH interpretation)
- Hospital: Allow with close monitoring (Trust: 0.80)

P0 Check (BEFORE P1):
- Lab interpretation = PANIC_HIGH → Immediate BLOCK

Final Decision: BLOCK
Confidence: 1.00
Precedence Rule: P0
Winning Source: LAB (KB-16)
Rationale: P0 PHYSIOLOGY SUPREMACY - Potassium = 6.8 mmol/L is PANIC_HIGH.
           No dosing or safety rule may override this physiological finding.
           Clinical intervention required before proceeding with drug therapy.
```

### P0 Test Scenarios (Implemented)

| Scenario | Lab Value | Classification | Expected Decision | Pass |
|----------|-----------|----------------|-------------------|------|
| Panic Potassium + ACE-I | K+ 6.8 mmol/L | PANIC_HIGH | BLOCK (P0) | ✅ |
| Critical AST + Pregnancy | AST 85 U/L | CRITICAL | BLOCK (P0) | ✅ |
| INR Panic + Warfarin | INR 8.5 | PANIC_HIGH | BLOCK (P0) | ✅ |
| Normal Lab Allows Proceeding | K+ 4.5 mmol/L | NORMAL | ACCEPT | ✅ |

### Decision Explainability Engine (NEW - 2026-01-26)

The Decision Explainability Engine generates human-readable explanations for all arbitration decisions, supporting regulatory compliance, clinical trust, and audit requirements.

**Key Components:**

```go
type DecisionExplainer struct {
    VerboseMode       bool  // Include technical details
    IncludeSourceRefs bool  // Include source references
}

type DecisionExplanation struct {
    Decision         DecisionType  // ACCEPT, BLOCK, OVERRIDE, DEFER, ESCALATE
    PrecedenceRule   string        // P0-P7
    Confidence       float64       // 0.0-1.0
    Summary          string        // Human-readable explanation
    ActionRequired   string        // Clinical action needed
    RiskLevel        string        // LOW, MEDIUM, HIGH, CRITICAL
}
```

**Example Outputs:**

| Decision | Example Output |
|----------|----------------|
| **ACCEPT** | "Prescription approved: No conflicts detected for Amoxicillin 500mg. Confidence: 95%. Standard dosing guidelines apply." |
| **BLOCK (P0)** | "BLOCKED because lab Potassium (6.8 mmol/L) exceeded CKD Stage 4 critical threshold. No dosing rule may override this physiological finding." |
| **BLOCK (P1)** | "BLOCKED by FDA Black Box Warning for [Drug]. This regulatory requirement cannot be overridden programmatically." |
| **OVERRIDE** | "Override requires dual sign-off because this action contradicts P2 authority guidance. Documenting override reason is mandatory." |
| **DEFER** | "Decision deferred: Insufficient data. Missing: Renal function (eGFR/CrCl), Patient age. Please provide and resubmit." |
| **ESCALATE** | "Escalated due to conflicting authority guidance (CPIC vs FDA) in presence of abnormal labs. Routing to expert review." |

**Output Formats:**
- `ForClinician()`: Clinical-friendly summary with risk level
- `ForAudit()`: Audit-ready format with all decision details

### Benign Drug Tests (NEW - 2026-01-26)

Trust-building tests that prove the system doesn't over-block safe medications.

**Purpose:** Demonstrate that the arbitration engine correctly allows benign drugs to proceed when no conflicts exist, preventing "alert fatigue" from false positives.

| Test | Scenario | Expected | Pass |
|------|----------|----------|------|
| Benign Antibiotic | Amoxicillin 500mg, healthy adult | ACCEPT | ✅ |
| Pregnancy Category B | Amoxicillin, pregnant patient | ACCEPT | ✅ |
| Abnormal Lab, No Conflicts | Cr 1.5 (HIGH) + Tylenol | ACCEPT | ✅ |
| P4 Lab Isolated | INR 4.8 alone, no conflicting rules | ACCEPT | ✅ |

**Test Scenario: Benign Antibiotic**
```
Patient: 35yo male, healthy
Intent: PRESCRIBE Amoxicillin 500mg TID
Labs: All normal

Inputs:
- FDA SPL: No contraindications (Trust: 0.90)
- CPIC: No pharmacogenomic issues (Trust: 1.00)
- KB-16: Labs normal
- Hospital: Standard antibiotic (Trust: 0.80)

Conflicts: NONE

Final Decision: ACCEPT
Confidence: 0.95
Rationale: No conflicts detected, standard dosing applies
```

### P4 Lab Escalation Tests (NEW - 2026-01-26)

P4 rule: When a lab interpretation is HIGH/LOW/ABNORMAL (NOT CRITICAL/PANIC) AND there is an actual conflict between sources, ESCALATE for expert review.

**Key Distinction:**
- **P0**: CRITICAL/PANIC → Immediate BLOCK (physiology supersedes all)
- **P4**: HIGH/ABNORMAL + conflict → ESCALATE (requires expert arbitration)
- **No conflict**: HIGH lab alone doesn't trigger ESCALATE

| Test | Lab Level | Has Conflict? | Expected | Pass |
|------|-----------|---------------|----------|------|
| P4 Escalation With Conflict | K+ 6.2 (HIGH) | Yes (AVOID vs REDUCE_DOSE) | ESCALATE | ✅ |
| P4 High With Agreement | K+ 5.5 (HIGH) | No (all say CAUTION) | ACCEPT | ✅ |
| P4 Isolated Lab | INR 4.8 (HIGH) | No rules triggered | ACCEPT | ✅ |
| Abnormal Lab No Conflicts | Cr 1.5 (HIGH) | No | ACCEPT | ✅ |

**Test Scenario: P4 Lab Escalation With Conflict**
```
Patient: 68yo female, CKD Stage 4
Intent: PRESCRIBE Lisinopril 10mg QD
Lab: K+ = 6.2 mmol/L (HIGH - not PANIC)

Inputs:
- Authority: eGFR < 30 = AVOID (Trust: 1.00)
- Canonical Rule: Monitor K+ if CKD = REDUCE_DOSE (Trust: 0.90)
- KB-16: K+ = 6.2 (HIGH interpretation)

Conflicts:
- C1: AVOID vs REDUCE_DOSE → Conflict detected

P0 Check: HIGH (not PANIC/CRITICAL) → Continue to P1-P7
P4 Applies: Abnormal lab + conflict → ESCALATE

Final Decision: ESCALATE
Confidence: 0.80
Precedence Rule: P4
Rationale: Abnormal lab + conflicting guidance requires expert review
```

### Summary Metrics

| Metric | Value |
|--------|-------|
| Total Tests | 30+ |
| Precedence Rules | 8 (P0-P7) |
| Decision Types | 5 |
| Conflict Types | 6 |
| Source Types | 5 |
| Code Coverage | ~85% |

---

*Phase 3d Implementation Completed: 2026-01-26*
*P0 Physiology Supremacy Added: 2026-01-26*
*Decision Explainability Engine Added: 2026-01-26*
*Benign Drug Tests Added: 2026-01-26*
*P4 Lab Escalation Tests Added: 2026-01-26*

