// prose_scanner.go: MedDRA Prose Scanner — finds ALL MedDRA terms in free text.
//
// This is the SCANNER, not the VALIDATOR. The difference:
//   - Validator (meddra.go): "You found 'Nausea' in a table cell. Is it real?" → Yes, PT 10028813.
//   - Scanner (this file):   "Here's 2,000 words of prose. Find every MedDRA term." → Nausea, Dizziness, Edema...
//
// Algorithm: N-gram sliding window over tokenized text.
//   1. Load all MedDRA LLT+PT terms (79K) into map[lowercase]→{ptCode, ptName, lltCode, socCode, socName}
//   2. For each word position, check 1-word, 2-word, ..., 8-word windows against map
//   3. Post-process: longest match wins (remove overlaps), negation detection, frequency extraction
//
// Performance: ~79K map entries × O(n*8) lookups for n words. A 2,000-word section ≈ 16K lookups, <5ms.
package factstore

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

// maxNgramWords is the maximum number of words in a MedDRA term we'll match.
// MedDRA's longest LLT/PT terms are ~7-8 words (e.g., "Neoplasm malignant of unspecified site").
const maxNgramWords = 8

// proseTerm holds the MedDRA codes for a matched term.
type proseTerm struct {
	PTCode  string // MedDRA Preferred Term code (e.g., "10028813")
	PTName  string // MedDRA PT name (official canonical name)
	LLTCode string // LLT code if matched at LLT level, else same as PTCode
	SOCCode string // System Organ Class code
	SOCName string // SOC name (e.g., "Gastrointestinal disorders")
}

// ProseMatch represents a single MedDRA term found in prose text.
type ProseMatch struct {
	// Term identification
	MatchedText string // Exact text matched in the prose (e.g., "nausea")
	PTCode      string // MedDRA PT code
	PTName      string // MedDRA PT canonical name (e.g., "Nausea")
	LLTCode     string // LLT code if matched via LLT
	SOCCode     string // SOC code
	SOCName     string // SOC name

	// Position in the prose
	StartWord int // Word index where the match starts
	EndWord   int // Word index where the match ends (exclusive)

	// Context extraction
	Frequency     string // Percentage or qualifier if found nearby (e.g., "3.2%", "common")
	FrequencyBand string // Standardized band: VERY_COMMON, COMMON, UNCOMMON, RARE, VERY_RARE
	IsNegated     bool   // True if negation detected (e.g., "no evidence of nausea")
}

// ProseScanResult holds all matches from scanning a prose section.
type ProseScanResult struct {
	Matches     []*ProseMatch // All non-negated, deduplicated matches
	NegatedCount int          // How many terms were found but negated
	TermsLoaded  int          // Total terms in the dictionary (for logging)
}

// MedDRAProseScanner scans free text for MedDRA terms using n-gram window matching.
type MedDRAProseScanner struct {
	terms map[string]*proseTerm // lowercase term → MedDRA codes
	log   *logrus.Entry
	stats scannerStats
}

type scannerStats struct {
	lltCount int
	ptCount  int
	totalTerms int
}

// negation patterns — words/phrases that negate a following medical term
var negationPrefixes = []string{
	"no ", "not ", "without ", "absence of ", "no evidence of ",
	"negative for ", "denies ", "denied ", "unlikely ",
	"no sign of ", "no signs of ", "no symptom of ", "no symptoms of ",
	"ruled out ", "rule out ", "free of ", "free from ",
	"no increase in ", "no significant ",
}

// frequencyPercent matches percentages like "3.2%", "(12%)", "0.1 %"
var frequencyPercent = regexp.MustCompile(`(\d+\.?\d*)\s*%`)

// qualitativeFrequency maps descriptor words to MedDRA-standard frequency bands.
var qualitativeFrequency = map[string]struct {
	band string
	rate string
}{
	"very common":   {band: "VERY_COMMON", rate: ">=10%"},
	"very commonly": {band: "VERY_COMMON", rate: ">=10%"},
	"common":        {band: "COMMON", rate: "1-10%"},
	"commonly":      {band: "COMMON", rate: "1-10%"},
	"frequently":    {band: "COMMON", rate: "1-10%"},
	"frequent":      {band: "COMMON", rate: "1-10%"},
	"uncommon":      {band: "UNCOMMON", rate: "0.1-1%"},
	"uncommonly":    {band: "UNCOMMON", rate: "0.1-1%"},
	"infrequent":    {band: "UNCOMMON", rate: "0.1-1%"},
	"infrequently":  {band: "UNCOMMON", rate: "0.1-1%"},
	"rare":          {band: "RARE", rate: "0.01-0.1%"},
	"rarely":        {band: "RARE", rate: "0.01-0.1%"},
	"very rare":     {band: "VERY_RARE", rate: "<0.01%"},
	"very rarely":   {band: "VERY_RARE", rate: "<0.01%"},
	"isolated":      {band: "VERY_RARE", rate: "<0.01%"},
}

// US→UK spelling bridge for loading: we store both US and UK forms in the dictionary
// so that American English prose matches MedDRA's British English terms.
var ukToUSBridge = []struct {
	uk string
	us string
}{
	// These are reversed from the normalizer's US→UK bridge:
	// MedDRA stores "Diarrhoea" — we also index "diarrhea" pointing to the same PT.
	{uk: "rrhoea", us: "rrhea"},    // Diarrhoea→Diarrhea
	{uk: "aemia", us: "emia"},      // Anaemia→Anemia
	{uk: "haem", us: "hem"},        // Haemorrhage→Hemorrhage
	{uk: "leuc", us: "leuk"},       // Leucopenia→Leukopenia
	{uk: "oesophag", us: "esophag"},// Oesophagitis→Esophagitis
	{uk: "oedema", us: "edema"},    // Oedema→Edema
	{uk: "tumour", us: "tumor"},    // Tumour→Tumor
	{uk: "colour", us: "color"},    // Discolouration→Discoloration
	{uk: "paed", us: "ped"},        // Paediatric→Pediatric
	{uk: "faec", us: "fec"},        // Faeces→Feces
	{uk: "foet", us: "fet"},        // Foetal→Fetal
	{uk: "gynaec", us: "gynec"},    // Gynaecological→Gynecological
	{uk: "ischaem", us: "ischem"},  // Ischaemia→Ischemia (note: both have 'sch')
	{uk: "caecum", us: "cecum"},    // Caecum→Cecum
	{uk: "leukaem", us: "leukem"}, // Leukaemia→Leukemia (after leuc→leuk, this catches the combined form)
}

// NewMedDRAProseScanner creates a scanner by loading all LLT+PT terms from the MedDRA SQLite DB.
// The db parameter should come from meddraLoader.DB() — the same SQLite used by the validator.
func NewMedDRAProseScanner(db *sql.DB, log *logrus.Entry) (*MedDRAProseScanner, error) {
	if db == nil {
		return nil, fmt.Errorf("MedDRA database is nil")
	}

	scanner := &MedDRAProseScanner{
		terms: make(map[string]*proseTerm, 100000), // pre-allocate for ~79K terms + US variants
		log:   log.WithField("component", "prose-scanner"),
	}

	if err := scanner.loadTerms(db); err != nil {
		return nil, fmt.Errorf("failed to load MedDRA terms: %w", err)
	}

	scanner.log.WithFields(logrus.Fields{
		"llt_loaded":    scanner.stats.lltCount,
		"pt_loaded":     scanner.stats.ptCount,
		"total_entries":  scanner.stats.totalTerms,
		"map_size":       len(scanner.terms),
	}).Info("MedDRA prose scanner initialized")

	return scanner, nil
}

// loadTerms populates the terms map from SQLite.
func (s *MedDRAProseScanner) loadTerms(db *sql.DB) error {
	// Load PTs first (they are the canonical terms)
	ptRows, err := db.Query(`
		SELECT pt_code, pt_name, COALESCE(pt_soc_code, '')
		FROM meddra_pt
		WHERE pt_name IS NOT NULL AND pt_name != ''
	`)
	if err != nil {
		return fmt.Errorf("query meddra_pt: %w", err)
	}
	defer ptRows.Close()

	// Build PT code → SOC lookup for enrichment
	ptSOCMap := make(map[string]string, 25000)
	for ptRows.Next() {
		var ptCode, ptName, socCode string
		if err := ptRows.Scan(&ptCode, &ptName, &socCode); err != nil {
			continue
		}
		ptSOCMap[ptCode] = socCode

		lower := strings.ToLower(strings.TrimSpace(ptName))
		if lower == "" || len(lower) < 3 {
			continue
		}

		term := &proseTerm{
			PTCode:  ptCode,
			PTName:  ptName,
			LLTCode: ptCode, // PT matched directly
			SOCCode: socCode,
		}
		s.terms[lower] = term
		s.stats.ptCount++

		// Add US spelling variant
		if usForm := bridgeUKtoUS(lower); usForm != lower {
			s.terms[usForm] = term
		}
	}

	// Load SOC names for enrichment
	socNames := make(map[string]string, 30)
	socRows, err := db.Query(`SELECT soc_code, soc_name FROM meddra_soc WHERE soc_name IS NOT NULL`)
	if err == nil {
		defer socRows.Close()
		for socRows.Next() {
			var code, name string
			if err := socRows.Scan(&code, &name); err == nil {
				socNames[code] = name
			}
		}
	}

	// Enrich PT terms with SOC names
	for _, term := range s.terms {
		if term.SOCCode != "" && term.SOCName == "" {
			if name, ok := socNames[term.SOCCode]; ok {
				term.SOCName = name
			}
		}
	}

	// Load LLTs — these map to parent PTs
	lltRows, err := db.Query(`
		SELECT llt_code, llt_name, pt_code
		FROM meddra_llt
		WHERE llt_name IS NOT NULL AND llt_name != '' AND llt_currency = 'Y'
	`)
	if err != nil {
		// LLT table might not exist if loaded from ValueSet JSON (flat structure)
		s.log.WithError(err).Debug("LLT query failed — may be ValueSet-only load (PTs still usable)")
		s.stats.totalTerms = len(s.terms)
		return nil
	}
	defer lltRows.Close()

	for lltRows.Next() {
		var lltCode, lltName, ptCode string
		if err := lltRows.Scan(&lltCode, &lltName, &ptCode); err != nil {
			continue
		}

		lower := strings.ToLower(strings.TrimSpace(lltName))
		if lower == "" || len(lower) < 3 {
			continue
		}

		// Look up parent PT info
		parentSOC := ptSOCMap[ptCode]
		parentSOCName := socNames[parentSOC]

		// Find the PT name from our already-loaded PT terms
		ptName := lltName // fallback
		for _, existing := range s.terms {
			if existing.PTCode == ptCode && existing.LLTCode == existing.PTCode {
				ptName = existing.PTName
				break
			}
		}

		term := &proseTerm{
			PTCode:  ptCode,
			PTName:  ptName,
			LLTCode: lltCode,
			SOCCode: parentSOC,
			SOCName: parentSOCName,
		}

		// Only add if we don't already have a PT-level entry (PT takes priority)
		if _, exists := s.terms[lower]; !exists {
			s.terms[lower] = term
		}
		s.stats.lltCount++

		// Add US spelling variant
		if usForm := bridgeUKtoUS(lower); usForm != lower {
			if _, exists := s.terms[usForm]; !exists {
				s.terms[usForm] = term
			}
		}
	}

	s.stats.totalTerms = len(s.terms)
	return nil
}

// bridgeUKtoUS converts British English medical terms to American English.
// This is the reverse of the normalizer's bridgeSpellingVariant.
// Returns the input unchanged if no bridge applies.
func bridgeUKtoUS(text string) string {
	result := text
	for _, rule := range ukToUSBridge {
		if strings.Contains(result, rule.uk) {
			result = strings.Replace(result, rule.uk, rule.us, 1)
		}
	}
	return result
}

// ScanText scans prose text for all MedDRA terms using n-gram sliding window.
//
// Algorithm:
//  1. Tokenize text into words (preserving order)
//  2. For each word position i, check windows of size 1..maxNgramWords
//  3. Longest match at each position wins
//  4. Remove overlapping matches (keep longest)
//  5. Check negation context for each match
//  6. Extract frequency from surrounding context
func (s *MedDRAProseScanner) ScanText(text string) *ProseScanResult {
	if len(s.terms) == 0 || len(text) < 10 {
		return &ProseScanResult{TermsLoaded: s.stats.totalTerms}
	}

	// Tokenize into words, preserving positions for context extraction
	words, wordStarts := tokenizeForScanning(text)
	if len(words) == 0 {
		return &ProseScanResult{TermsLoaded: s.stats.totalTerms}
	}

	// Phase 1: Find all matches using n-gram window
	var rawMatches []*ProseMatch
	lowerText := strings.ToLower(text)

	for i := 0; i < len(words); i++ {
		var bestMatch *ProseMatch
		bestLen := 0

		// Try n-grams from longest to shortest (greedy: longest match wins)
		maxN := maxNgramWords
		if i+maxN > len(words) {
			maxN = len(words) - i
		}

		for n := maxN; n >= 1; n-- {
			// Build the n-gram
			ngram := strings.Join(words[i:i+n], " ")
			ngramLower := strings.ToLower(ngram)

			if term, ok := s.terms[ngramLower]; ok {
				// Skip very short single-word matches that are common English words
				if n == 1 && isScannerCommonWord(ngramLower) {
					continue
				}
				bestMatch = &ProseMatch{
					MatchedText: ngram,
					PTCode:      term.PTCode,
					PTName:      term.PTName,
					LLTCode:     term.LLTCode,
					SOCCode:     term.SOCCode,
					SOCName:     term.SOCName,
					StartWord:   i,
					EndWord:     i + n,
				}
				bestLen = n
				break // Longest match found, stop trying shorter
			}
		}

		if bestMatch != nil {
			rawMatches = append(rawMatches, bestMatch)
			i += bestLen - 1 // Skip past the matched words (loop will i++ again)
		}
	}

	// Phase 2: Remove overlapping matches (already handled by greedy + skip)
	// Phase 3: Check negation and extract frequency
	result := &ProseScanResult{
		TermsLoaded: s.stats.totalTerms,
	}

	seen := make(map[string]bool) // Dedup by PT code within this scan
	for _, m := range rawMatches {
		// Check negation
		charPos := 0
		if m.StartWord < len(wordStarts) {
			charPos = wordStarts[m.StartWord]
		}
		if isNegatedInContext(lowerText, charPos) {
			m.IsNegated = true
			result.NegatedCount++
			continue // Skip negated terms
		}

		// Dedup: keep first occurrence of each PT
		if seen[m.PTCode] {
			continue
		}
		seen[m.PTCode] = true

		// Extract frequency from nearby context
		extractFrequencyContext(lowerText, charPos, m)

		result.Matches = append(result.Matches, m)
	}

	return result
}

// TermCount returns the number of terms loaded in the scanner dictionary.
func (s *MedDRAProseScanner) TermCount() int {
	return s.stats.totalTerms
}

// tokenizeForScanning splits text into words and records each word's character offset.
func tokenizeForScanning(text string) (words []string, wordStarts []int) {
	words = make([]string, 0, len(text)/5) // rough estimate: avg word length ~5
	wordStarts = make([]int, 0, len(text)/5)

	inWord := false
	wordStart := 0

	for i, r := range text {
		if isWordChar(r) {
			if !inWord {
				wordStart = i
				inWord = true
			}
		} else {
			if inWord {
				word := text[wordStart:i]
				if len(word) > 0 {
					words = append(words, word)
					wordStarts = append(wordStarts, wordStart)
				}
				inWord = false
			}
		}
	}
	// Handle last word
	if inWord {
		word := text[wordStart:]
		if len(word) > 0 {
			words = append(words, word)
			wordStarts = append(wordStarts, wordStart)
		}
	}

	return
}

// isWordChar returns true for characters that are part of a word (letters, digits, hyphens).
// Hyphens are included because MedDRA terms contain them (e.g., "Drug-induced liver injury").
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '\''
}

// isNegatedInContext checks if the character position in text is preceded by a negation phrase.
// Looks backwards up to 60 characters for negation patterns.
func isNegatedInContext(lowerText string, charPos int) bool {
	// Look at the 60 characters before this position
	start := charPos - 60
	if start < 0 {
		start = 0
	}
	prefix := lowerText[start:charPos]

	for _, neg := range negationPrefixes {
		if strings.HasSuffix(prefix, neg) {
			return true
		}
		// Also check with trailing spaces trimmed
		trimmed := strings.TrimRight(prefix, " ")
		if strings.HasSuffix(trimmed, strings.TrimSpace(neg)) {
			return true
		}
	}
	return false
}

// extractFrequencyContext looks for percentage or qualitative frequency near the matched term.
// Searches a ±120 character window around the term.
func extractFrequencyContext(lowerText string, charPos int, m *ProseMatch) {
	// Define window
	windowStart := charPos - 120
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := charPos + len(m.MatchedText) + 120
	if windowEnd > len(lowerText) {
		windowEnd = len(lowerText)
	}
	window := lowerText[windowStart:windowEnd]

	// Try percentage first (more specific)
	if matches := frequencyPercent.FindStringSubmatch(window); len(matches) > 0 {
		m.Frequency = matches[0]
		return
	}

	// Try qualitative descriptors
	for qualifier, info := range qualitativeFrequency {
		if strings.Contains(window, qualifier) {
			m.Frequency = qualifier + " (" + info.rate + ")"
			m.FrequencyBand = info.band
			return
		}
	}
}

// isCommonWord filters out single-word matches that are too generic to be medical terms
// even though they appear in MedDRA. These cause massive false positives in prose scanning.
var commonWordSet = map[string]bool{
	// Body parts that appear as normal English
	"pain": false, // Keep — "pain" IS a valid AE
	// Common words that happen to be MedDRA LLTs but aren't useful as standalone
	"fall":      true,
	"falls":     true,
	"feeling":   true,
	"death":     true, // Too broad without context
	"injury":    true,
	"disease":   true,
	"disorder":  true,
	"condition": true,
	"drug":      true,
	"effect":    true,
	"reaction":  true,
	"product":   true,
	"therapy":   true,
	"treatment": true,
	"dose":      true,
	"test":      true,
	"stress":    true,
	"shock":     true, // Too broad — "cardiogenic shock" is valid as 2-gram
	"mass":      true,
	"rash":      false, // Keep — "rash" is specific enough
	"cough":     false, // Keep
	"fever":     false, // Keep
	"fatigue":   false, // Keep
	"anxiety":   false, // Keep
	"nausea":    false, // Keep
}

func isScannerCommonWord(word string) bool {
	if val, ok := commonWordSet[word]; ok {
		return val // true = filter out, false = keep
	}
	// Filter single words under 4 chars that aren't known medical terms
	if len(word) < 4 {
		return true
	}
	return false
}
