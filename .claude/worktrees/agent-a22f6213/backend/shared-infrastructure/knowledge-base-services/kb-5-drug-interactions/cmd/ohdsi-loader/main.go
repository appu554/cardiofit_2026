// =============================================================================
// OHDSI Streaming ETL Loader
// =============================================================================
// Purpose: Filter OHDSI vocabulary files to minimal DDI-execution subset
// Strategy: Stream-read CSV → Apply hard filters → Write filtered TSV
//
// This loader implements the "Minimal Required Subset" principle:
// - Keep only ~5% of total vocabulary (not 45M rows)
// - Filter by Vocabulary ID whitelist
// - Filter by Relationship ID whitelist
// - Output lean, execution-grade substrate
//
// Usage:
//   cat CONCEPT.csv | ./ohdsi-loader -type=concept > filtered_concept.tsv
//   cat CONCEPT_RELATIONSHIP.csv | ./ohdsi-loader -type=rel > filtered_rel.tsv
// =============================================================================

package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// =============================================================================
// AUTHORITATIVE WHITELIST (Lock These)
// =============================================================================

// allowedVocabularies defines the vocabulary IDs to KEEP
// These are the only vocabularies needed for DDI Class Expansion
var allowedVocabularies = map[string]bool{
	// Drug concepts (execution targets)
	"RxNorm":           true, // The "Spine" - Ingredients, Brand Names, NDCs
	"RxNorm Extension": true, // International drugs (non-US brands)

	// Drug class concepts (execution anchors)
	"ATC":      true, // WHO standard class hierarchy
	"VA Class": true, // Veterans Affairs NDF-RT - mechanism-based (ONC alignment)
	"MED-RT":   true, // MED-RT successor

	// Context Engine support
	"LOINC": true, // Lab codes for Context Engine (KB-16 reference)
	"UCUM":  true, // Units for drug strength normalization
}

// allowedRelationships defines the relationship IDs to KEEP
// These are the only relationships needed for Class Expansion
var allowedRelationships = map[string]bool{
	// PRIMARY: Drug → Class execution edge
	"Drug has drug class": true,

	// HIERARCHY: Class → Class resolution
	"Is a":     true, // Standard hierarchy
	"Subsumes": true, // Inverse hierarchy

	// MAPPING: Translation layer
	"Maps to":     true, // Hospital Local Code → Standard RxNorm
	"RxNorm - ATC": true, // Explicit NLM mappings
	"ATC - RxNorm": true, // Inverse mapping
}

// =============================================================================
// Column Index Mapping (OHDSI Standard Schema)
// =============================================================================

// CONCEPT.csv columns (tab-delimited):
// 0: concept_id
// 1: concept_name
// 2: domain_id
// 3: vocabulary_id
// 4: concept_class_id
// 5: standard_concept
// 6: concept_code
// 7: valid_start_date
// 8: valid_end_date
// 9: invalid_reason

const (
	conceptIdIdx         = 0
	conceptNameIdx       = 1
	domainIdIdx          = 2
	vocabularyIdIdx      = 3
	conceptClassIdIdx    = 4
	standardConceptIdx   = 5
	conceptCodeIdx       = 6
	validStartDateIdx    = 7
	validEndDateIdx      = 8
	invalidReasonIdx     = 9
)

// CONCEPT_RELATIONSHIP.csv columns (tab-delimited):
// 0: concept_id_1
// 1: concept_id_2
// 2: relationship_id
// 3: valid_start_date
// 4: valid_end_date
// 5: invalid_reason

const (
	conceptId1Idx     = 0
	conceptId2Idx     = 1
	relationshipIdIdx = 2
	relValidStartIdx  = 3
	relValidEndIdx    = 4
	relInvalidIdx     = 5
)

// =============================================================================
// Loader Statistics
// =============================================================================

type LoaderStats struct {
	TotalRead    int64
	TotalWritten int64
	Skipped      int64
	Errors       int64
}

func (s *LoaderStats) Summary() string {
	return fmt.Sprintf(
		"Total Read: %d | Written: %d | Skipped: %d | Errors: %d | Filter Rate: %.2f%%",
		s.TotalRead, s.TotalWritten, s.Skipped, s.Errors,
		float64(s.TotalWritten)/float64(s.TotalRead)*100,
	)
}

// =============================================================================
// Main Entry Point
// =============================================================================

func main() {
	// Parse command line flags
	filterType := flag.String("type", "", "Filter type: 'concept' or 'rel'")
	verbose := flag.Bool("v", false, "Verbose output (stats to stderr)")
	flag.Parse()

	if *filterType == "" {
		log.Fatal("Usage: ohdsi-loader -type=concept|rel [-v]")
	}

	stats := &LoaderStats{}

	switch *filterType {
	case "concept":
		filterConcepts(os.Stdin, os.Stdout, stats)
	case "rel":
		filterRelationships(os.Stdin, os.Stdout, stats)
	default:
		log.Fatalf("Unknown type: %s. Use 'concept' or 'rel'", *filterType)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "\n[OHDSI Loader] %s\n", stats.Summary())
	}
}

// =============================================================================
// Concept Filter (CONCEPT.csv)
// =============================================================================

func filterConcepts(input io.Reader, output io.Writer, stats *LoaderStats) {
	// Use buffered reader for large files
	reader := csv.NewReader(bufio.NewReaderSize(input, 1024*1024)) // 1MB buffer
	reader.Comma = '\t'
	reader.LazyQuotes = true // Handle messy medical strings
	reader.FieldsPerRecord = -1 // Variable fields (some rows may be malformed)

	writer := csv.NewWriter(output)
	writer.Comma = '\t'
	defer writer.Flush()

	// Read and write header
	header, err := reader.Read()
	if err != nil {
		log.Fatal("Failed to read header:", err)
	}
	writer.Write(header)

	// Streaming filter loop
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			stats.Errors++
			continue
		}

		stats.TotalRead++

		// Safety check for malformed rows
		if len(record) < 10 {
			stats.Skipped++
			continue
		}

		// Apply vocabulary filter
		vocabularyId := strings.TrimSpace(record[vocabularyIdIdx])

		if allowedVocabularies[vocabularyId] {
			// Additional filter: For Drug domain, prefer standard concepts
			domain := strings.TrimSpace(record[domainIdIdx])
			standardConcept := strings.TrimSpace(record[standardConceptIdx])

			// Keep if:
			// 1. It's a class vocabulary (ATC, VA Class, MED-RT) - always keep
			// 2. It's a drug vocabulary (RxNorm) AND is standard concept
			// 3. It's LOINC/UCUM - always keep (for context engine)
			isClassVocab := vocabularyId == "ATC" || vocabularyId == "VA Class" || vocabularyId == "MED-RT"
			isContextVocab := vocabularyId == "LOINC" || vocabularyId == "UCUM"
			isDrugVocab := vocabularyId == "RxNorm" || vocabularyId == "RxNorm Extension"

			shouldKeep := false
			if isClassVocab || isContextVocab {
				shouldKeep = true
			} else if isDrugVocab {
				// For drugs: keep standard concepts OR ingredients
				// standard_concept = 'S' means standard
				// Also keep 'C' (classification) concepts
				shouldKeep = standardConcept == "S" || standardConcept == "C" ||
					domain == "Drug" // Keep all drug domain for now (can tighten later)
			}

			if shouldKeep {
				writer.Write(record)
				stats.TotalWritten++
			} else {
				stats.Skipped++
			}
		} else {
			stats.Skipped++
		}

		// Progress indicator (every 500k rows)
		if stats.TotalRead%500000 == 0 {
			fmt.Fprintf(os.Stderr, "[Progress] Processed %dk rows, kept %dk\n",
				stats.TotalRead/1000, stats.TotalWritten/1000)
		}
	}
}

// =============================================================================
// Relationship Filter (CONCEPT_RELATIONSHIP.csv)
// =============================================================================

func filterRelationships(input io.Reader, output io.Writer, stats *LoaderStats) {
	// Use buffered reader for large files
	reader := csv.NewReader(bufio.NewReaderSize(input, 1024*1024)) // 1MB buffer
	reader.Comma = '\t'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	writer := csv.NewWriter(output)
	writer.Comma = '\t'
	defer writer.Flush()

	// Read and write header
	header, err := reader.Read()
	if err != nil {
		log.Fatal("Failed to read header:", err)
	}
	writer.Write(header)

	// Streaming filter loop
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			stats.Errors++
			continue
		}

		stats.TotalRead++

		// Safety check for malformed rows
		if len(record) < 6 {
			stats.Skipped++
			continue
		}

		// Apply relationship filter
		relationshipId := strings.TrimSpace(record[relationshipIdIdx])

		if allowedRelationships[relationshipId] {
			writer.Write(record)
			stats.TotalWritten++
		} else {
			stats.Skipped++
		}

		// Progress indicator (every 1M rows for relationships - they're bigger)
		if stats.TotalRead%1000000 == 0 {
			fmt.Fprintf(os.Stderr, "[Progress] Processed %dM rows, kept %dk\n",
				stats.TotalRead/1000000, stats.TotalWritten/1000)
		}
	}
}
