// Package factstore provides the DDI Grammar Extractor for deterministic
// drug-drug interaction extraction from FDA SPL prose text.
//
// P2.1: Regex-based DDI prose patterns that capture standard FDA label
// interaction language. Each pattern produces a KBInteractionContent fact
// compatible with the table-based ParseInteraction output.
//
// This runs BEFORE the LLM fallback so that deterministic grammar-extracted
// interactions reduce LLM dependency from ~67% to ~7%.
package factstore

import (
	"regexp"
	"strings"
)

// DDIGrammar extracts drug-drug interaction facts from prose text using
// regex patterns that match standard FDA label language.
type DDIGrammar struct {
	// Compiled pattern families
	enzymePatterns        []*ddiPattern
	concomitantPatterns   []*ddiPattern
	avoidancePatterns     []*ddiPattern
	concentrationPatterns []*ddiPattern
	effectPatterns        []*ddiPattern

	// Drug name validation: reject common false-positive words
	falsePositiveDrugs *regexp.Regexp
}

// ddiPattern wraps a regex with metadata for structured extraction
type ddiPattern struct {
	re          *regexp.Regexp
	patternType string // ENZYME, CONCOMITANT, AVOIDANCE, CONCENTRATION, EFFECT
	severity    string // Default severity if not derivable from match
}

// DDIGrammarMatch represents a single grammar-extracted interaction
type DDIGrammarMatch struct {
	InteractantName string
	ClinicalEffect  string
	Mechanism       string
	Management      string
	Severity        string
	SourcePhrase    string
	PatternType     string
}

// NewDDIGrammar creates a DDI grammar extractor with precompiled patterns.
func NewDDIGrammar() *DDIGrammar {
	g := &DDIGrammar{
		// Reject single common words that regex might capture as "drug names"
		falsePositiveDrugs: regexp.MustCompile(`(?i)^(the|a|an|this|that|these|those|it|its|and|or|but|with|for|from|into|also|been|were|was|are|have|has|had|may|can|could|should|would|will|not|more|less|other|such|each|both|all|any|some|no|use|used|using|dose|doses|effect|effects|result|results|drug|drugs|patient|patients|treatment|therapy|study|studies|table|concomitant|clinical|significant|increased|decreased|reported)$`),
	}
	g.initPatterns()
	return g
}

// initPatterns compiles all DDI grammar patterns.
func (g *DDIGrammar) initPatterns() {
	// drugRe matches a capitalized drug name (1-4 words, first word capitalized)
	// Examples: "Warfarin", "St. John's Wort", "Oral contraceptives"
	drugRe := `([A-Z][a-z]+(?:[\s\-'][A-Za-z]+){0,3})`

	// enzymeRe matches CYP enzymes, transporters, and metabolic pathways
	enzymeRe := `(CYP\s*\d[A-Z]\d{1,2}|P-?gp|P-?glycoprotein|OATP\d?[A-Z]?\d?|BCRP|UGT\d[A-Z]\d|MAO-?[AB]?|COMT|aldehyde\s+oxidase)`

	// =================================================================
	// Pattern Family 1: Enzyme substrate/inhibitor/inducer declarations
	// "Warfarin is a substrate of CYP2C9"
	// "Fluconazole is a strong inhibitor of CYP2C19"
	// =================================================================
	g.enzymePatterns = []*ddiPattern{
		{
			re:          regexp.MustCompile(`(?i)` + drugRe + `\s+(?:is|are)\s+(?:a\s+)?(?:major\s+|strong\s+|moderate\s+|weak\s+|potent\s+)?(?:substrate|inhibitor|inducer)s?\s+of\s+` + enzymeRe),
			patternType: "ENZYME",
			severity:    "MODERATE",
		},
		{
			re:          regexp.MustCompile(`(?i)(?:strong|potent|moderate|weak)\s+(?:inhibitor|inducer)s?\s+of\s+` + enzymeRe + `\s*(?:,|\(|such\s+as)\s*(?:e\.?g\.?,?\s*)?` + drugRe),
			patternType: "ENZYME",
			severity:    "MODERATE",
		},
		// "Metabolized primarily by CYP3A4"
		{
			re:          regexp.MustCompile(`(?i)(?:metabolized|cleared|eliminated)\s+(?:primarily|mainly|predominantly)?\s*(?:by|via|through)\s+` + enzymeRe),
			patternType: "ENZYME",
			severity:    "MODERATE",
		},
	}

	// =================================================================
	// Pattern Family 2: Concomitant use warnings
	// "Concomitant use of warfarin with aspirin may result in increased bleeding"
	// "Coadministration of X and Y can lead to increased exposure"
	// =================================================================
	g.concomitantPatterns = []*ddiPattern{
		{
			re:          regexp.MustCompile(`(?i)(?:concomitant|concurrent|simultaneous)\s+(?:use|administration|therapy)\s+(?:of\s+)?` + drugRe + `\s+(?:with|and)\s+` + drugRe + `\s+(?:may|can|could|might|will)\s+(?:result\s+in|cause|lead\s+to|produce)\s+([^.]{5,80})`),
			patternType: "CONCOMITANT",
			severity:    "MODERATE",
		},
		{
			re:          regexp.MustCompile(`(?i)(?:co-?administration|co-?prescribing)\s+(?:of\s+)?` + drugRe + `\s+(?:with|and)\s+` + drugRe + `\s+(?:may|can|could|might|will)\s+(?:result\s+in|cause|lead\s+to|produce)\s+([^.]{5,80})`),
			patternType: "CONCOMITANT",
			severity:    "MODERATE",
		},
		// "When [Drug] is used with [Drug], [effect]"
		{
			re:          regexp.MustCompile(`(?i)[Ww]hen\s+` + drugRe + `\s+is\s+(?:used|given|taken|administered|combined)\s+(?:with|together\s+with)\s+` + drugRe + `\s*,\s*([^.]{5,80})`),
			patternType: "CONCOMITANT",
			severity:    "MODERATE",
		},
	}

	// =================================================================
	// Pattern Family 3: Avoidance directives
	// "Avoid concomitant use with strong CYP3A4 inhibitors"
	// "Do not use with MAO inhibitors"
	// =================================================================
	g.avoidancePatterns = []*ddiPattern{
		{
			re:          regexp.MustCompile(`(?i)(?:avoid|do\s+not\s+use|should\s+not\s+be\s+used|is\s+contraindicated)\s+(?:concomitant\s+(?:use|administration)\s+)?(?:with|of)\s+` + drugRe),
			patternType: "AVOIDANCE",
			severity:    "SEVERE",
		},
		{
			re:          regexp.MustCompile(`(?i)(?:avoid|do\s+not\s+use|should\s+not\s+be\s+used)\s+(?:concomitant\s+(?:use|administration)\s+)?(?:with|of)\s+(?:strong\s+|potent\s+)?` + enzymeRe + `\s+(?:inhibitor|inducer)s?`),
			patternType: "AVOIDANCE",
			severity:    "SEVERE",
		},
		// "contraindicated with [Drug]"
		{
			re:          regexp.MustCompile(`(?i)contraindicated\s+(?:with|when\s+used\s+with|in\s+(?:combination|conjunction)\s+with)\s+` + drugRe),
			patternType: "AVOIDANCE",
			severity:    "SEVERE",
		},
	}

	// =================================================================
	// Pattern Family 4: Concentration/exposure change statements
	// "Increased plasma concentrations of warfarin"
	// "May decrease the serum levels of digoxin"
	// =================================================================
	g.concentrationPatterns = []*ddiPattern{
		{
			re:          regexp.MustCompile(`(?i)((?:increase|decrease|elevate|reduce|raise|lower)[sd]?)\s+(?:the\s+)?(?:plasma|serum|blood)?\s*(?:concentration|level|exposure)s?\s+of\s+` + drugRe),
			patternType: "CONCENTRATION",
			severity:    "MODERATE",
		},
		{
			re:          regexp.MustCompile(`(?i)` + drugRe + `\s+(?:plasma|serum|blood)?\s*(?:concentration|level|exposure)s?\s+(?:may\s+be|are|were|was)\s+((?:increase|decrease|elevate|reduce|raise|lower)[sd]?)`),
			patternType: "CONCENTRATION",
			severity:    "MODERATE",
		},
		// "May increase/decrease the AUC/Cmax of [Drug] by X%"
		{
			re:          regexp.MustCompile(`(?i)(?:may|can|could)\s+(increase|decrease)\s+(?:the\s+)?(?:AUC|C(?:max|min)|exposure|bioavailability)\s+of\s+` + drugRe),
			patternType: "CONCENTRATION",
			severity:    "MODERATE",
		},
	}

	// =================================================================
	// Pattern Family 5: Clinical effect sentences
	// "Warfarin may potentiate the effect of aspirin"
	// "May enhance the anticoagulant effect"
	// =================================================================
	g.effectPatterns = []*ddiPattern{
		{
			re:          regexp.MustCompile(`(?i)` + drugRe + `\s+(?:may|can|could|might)\s+(?:potentiate|enhance|augment|diminish|attenuate|antagonize|reduce)\s+(?:the\s+)?(?:effect|action|activity|efficacy)\s+of\s+` + drugRe),
			patternType: "EFFECT",
			severity:    "MODERATE",
		},
		// "May enhance the [therapeutic] effect of [drug class]"
		{
			re:          regexp.MustCompile(`(?i)(?:may|can|could)\s+(?:potentiate|enhance|augment|diminish|reduce)\s+(?:the\s+)?(?:anticoagulant|antihypertensive|hypoglycemic|hypotensive|sedative|CNS\s+depressant|nephrotoxic|hepatotoxic|myelosuppressive|serotonergic)\s+(?:effect|action|activity)s?\s+(?:of\s+)?` + drugRe),
			patternType: "EFFECT",
			severity:    "MODERATE",
		},
		// "[Drug] is expected to increase the risk of [effect]"
		{
			re:          regexp.MustCompile(`(?i)` + drugRe + `\s+(?:is\s+expected\s+to|may|can)\s+(?:increase|decrease)\s+(?:the\s+)?risk\s+of\s+([^.]{5,60})`),
			patternType: "EFFECT",
			severity:    "MODERATE",
		},
	}
}

// ExtractFromProse scans section prose text for DDI patterns and returns
// structured interaction facts. The indexDrugName is the drug whose label
// we're reading (precipitant drug).
func (g *DDIGrammar) ExtractFromProse(proseText string, indexDrugName string) []DDIGrammarMatch {
	if len(proseText) < 50 {
		return nil
	}

	var matches []DDIGrammarMatch
	seen := make(map[string]bool) // Dedup by lowercase interactant

	// Split into sentences for better pattern matching
	sentences := splitIntoSentences(proseText)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 20 {
			continue
		}

		// Try each pattern family in priority order
		g.matchEnzyme(sentence, indexDrugName, &matches, seen)
		g.matchConcomitant(sentence, indexDrugName, &matches, seen)
		g.matchAvoidance(sentence, indexDrugName, &matches, seen)
		g.matchConcentration(sentence, indexDrugName, &matches, seen)
		g.matchEffect(sentence, indexDrugName, &matches, seen)
	}

	return matches
}

// matchEnzyme extracts enzyme-mediated DDI declarations
func (g *DDIGrammar) matchEnzyme(sentence, indexDrugName string, matches *[]DDIGrammarMatch, seen map[string]bool) {
	for _, pat := range g.enzymePatterns {
		allMatches := pat.re.FindAllStringSubmatch(sentence, -1)
		for _, m := range allMatches {
			if len(m) < 3 {
				continue
			}
			// For enzyme patterns: group 1 = drug, group 2 = enzyme
			drug := strings.TrimSpace(m[1])
			enzyme := strings.TrimSpace(m[2])

			if g.isFalsePositiveDrug(drug) || enzyme == "" {
				continue
			}

			key := strings.ToLower(drug + "|" + enzyme)
			if seen[key] {
				continue
			}
			seen[key] = true

			mechanism := extractMechanismFromSentence(sentence, enzyme)

			*matches = append(*matches, DDIGrammarMatch{
				InteractantName: drug,
				Mechanism:       mechanism,
				ClinicalEffect:  "Metabolic interaction via " + enzyme,
				Severity:        g.deriveSeverity(sentence, pat.severity),
				SourcePhrase:    truncatePhrase(sentence, 200),
				PatternType:     pat.patternType,
			})
		}
	}
}

// matchConcomitant extracts concomitant use warnings
func (g *DDIGrammar) matchConcomitant(sentence, indexDrugName string, matches *[]DDIGrammarMatch, seen map[string]bool) {
	for _, pat := range g.concomitantPatterns {
		allMatches := pat.re.FindAllStringSubmatch(sentence, -1)
		for _, m := range allMatches {
			if len(m) < 4 {
				continue
			}
			// Groups: drug1, drug2, effect
			drug1 := strings.TrimSpace(m[1])
			drug2 := strings.TrimSpace(m[2])
			effect := strings.TrimSpace(m[3])

			// The interactant is whichever drug is NOT the index drug
			interactant := drug2
			if strings.EqualFold(drug2, indexDrugName) || isSubstringMatch(indexDrugName, drug2) {
				interactant = drug1
			}

			if g.isFalsePositiveDrug(interactant) {
				continue
			}

			key := strings.ToLower(interactant)
			if seen[key] {
				continue
			}
			seen[key] = true

			*matches = append(*matches, DDIGrammarMatch{
				InteractantName: interactant,
				ClinicalEffect:  effect,
				Mechanism:       extractMechanismFromSentence(sentence, ""),
				Severity:        g.deriveSeverity(sentence, pat.severity),
				SourcePhrase:    truncatePhrase(sentence, 200),
				PatternType:     pat.patternType,
			})
		}
	}
}

// matchAvoidance extracts avoidance directives
func (g *DDIGrammar) matchAvoidance(sentence, indexDrugName string, matches *[]DDIGrammarMatch, seen map[string]bool) {
	for _, pat := range g.avoidancePatterns {
		allMatches := pat.re.FindAllStringSubmatch(sentence, -1)
		for _, m := range allMatches {
			if len(m) < 2 {
				continue
			}
			interactant := strings.TrimSpace(m[len(m)-1]) // Last capture group is always the drug

			if g.isFalsePositiveDrug(interactant) {
				continue
			}

			key := strings.ToLower(interactant)
			if seen[key] {
				continue
			}
			seen[key] = true

			*matches = append(*matches, DDIGrammarMatch{
				InteractantName: interactant,
				ClinicalEffect:  "Concomitant use should be avoided",
				Management:      "Avoid concomitant use",
				Severity:        "SEVERE", // Avoidance directives are always severe
				SourcePhrase:    truncatePhrase(sentence, 200),
				PatternType:     pat.patternType,
			})
		}
	}
}

// matchConcentration extracts concentration/exposure change statements
func (g *DDIGrammar) matchConcentration(sentence, indexDrugName string, matches *[]DDIGrammarMatch, seen map[string]bool) {
	for _, pat := range g.concentrationPatterns {
		allMatches := pat.re.FindAllStringSubmatch(sentence, -1)
		for _, m := range allMatches {
			if len(m) < 3 {
				continue
			}

			// Extract direction and drug — order depends on which pattern matched
			var direction, drug string
			for i := 1; i < len(m); i++ {
				trimmed := strings.TrimSpace(m[i])
				if trimmed == "" {
					continue
				}
				lower := strings.ToLower(trimmed)
				if strings.HasPrefix(lower, "increase") || strings.HasPrefix(lower, "decrease") ||
					strings.HasPrefix(lower, "elevate") || strings.HasPrefix(lower, "reduce") ||
					strings.HasPrefix(lower, "raise") || strings.HasPrefix(lower, "lower") {
					direction = trimmed
				} else if len(trimmed) > 2 && trimmed[0] >= 'A' && trimmed[0] <= 'Z' {
					drug = trimmed
				}
			}

			if drug == "" || g.isFalsePositiveDrug(drug) {
				continue
			}

			key := strings.ToLower(drug)
			if seen[key] {
				continue
			}
			seen[key] = true

			lowerDir := strings.ToLower(direction)
			effect := strings.ToUpper(lowerDir[:1]) + lowerDir[1:] + " " + drug + " levels"

			*matches = append(*matches, DDIGrammarMatch{
				InteractantName: drug,
				ClinicalEffect:  effect,
				Mechanism:       extractMechanismFromSentence(sentence, ""),
				Severity:        g.deriveSeverity(sentence, pat.severity),
				SourcePhrase:    truncatePhrase(sentence, 200),
				PatternType:     pat.patternType,
			})
		}
	}
}

// matchEffect extracts clinical effect sentences
func (g *DDIGrammar) matchEffect(sentence, indexDrugName string, matches *[]DDIGrammarMatch, seen map[string]bool) {
	for _, pat := range g.effectPatterns {
		allMatches := pat.re.FindAllStringSubmatch(sentence, -1)
		for _, m := range allMatches {
			if len(m) < 3 {
				continue
			}

			// Extract drug and effect from capture groups
			var drug, effect string
			for i := 1; i < len(m); i++ {
				trimmed := strings.TrimSpace(m[i])
				if trimmed == "" {
					continue
				}
				if len(trimmed) > 2 && trimmed[0] >= 'A' && trimmed[0] <= 'Z' && !g.isFalsePositiveDrug(trimmed) {
					if drug == "" {
						drug = trimmed
					} else if effect == "" {
						effect = trimmed
					}
				} else if len(trimmed) > 5 {
					effect = trimmed
				}
			}

			// If first captured drug is the index drug, try second
			if strings.EqualFold(drug, indexDrugName) || isSubstringMatch(indexDrugName, drug) {
				if effect != "" && len(effect) > 2 && effect[0] >= 'A' && effect[0] <= 'Z' {
					drug = effect
					effect = ""
				} else {
					continue
				}
			}

			if drug == "" || g.isFalsePositiveDrug(drug) {
				continue
			}

			key := strings.ToLower(drug)
			if seen[key] {
				continue
			}
			seen[key] = true

			if effect == "" {
				effect = extractEffectContext(sentence)
			}

			*matches = append(*matches, DDIGrammarMatch{
				InteractantName: drug,
				ClinicalEffect:  effect,
				Mechanism:       extractMechanismFromSentence(sentence, ""),
				Severity:        g.deriveSeverity(sentence, pat.severity),
				SourcePhrase:    truncatePhrase(sentence, 200),
				PatternType:     pat.patternType,
			})
		}
	}
}

// ToInteractionContents converts grammar matches to KBInteractionContent structs
// for direct use with the existing pipeline fact creation.
func (g *DDIGrammar) ToInteractionContents(matches []DDIGrammarMatch, indexDrugName string) []KBInteractionContent {
	var contents []KBInteractionContent

	for _, m := range matches {
		content := KBInteractionContent{
			InteractionType: "DRUG_DRUG",
			PrecipitantDrug: indexDrugName,
			ObjectDrug:      m.InteractantName,
			InteractantName: m.InteractantName,
			Severity:        m.Severity,
			Mechanism:       m.Mechanism,
			ClinicalEffect:  m.ClinicalEffect,
			Management:      m.Management,
			SourcePhrase:    m.SourcePhrase,
		}
		contents = append(contents, content)
	}

	return contents
}

// =============================================================================
// HELPERS
// =============================================================================

// deriveSeverity upgrades/downgrades default severity based on sentence keywords
func (g *DDIGrammar) deriveSeverity(sentence, defaultSeverity string) string {
	lower := strings.ToLower(sentence)

	if strings.Contains(lower, "contraindicated") || strings.Contains(lower, "avoid") ||
		strings.Contains(lower, "do not use") || strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "life-threatening") {
		return "SEVERE"
	}
	if strings.Contains(lower, "caution") || strings.Contains(lower, "monitor") ||
		strings.Contains(lower, "dose adjustment") || strings.Contains(lower, "reduce dose") {
		return "MODERATE"
	}
	if strings.Contains(lower, "no clinically significant") || strings.Contains(lower, "unlikely") ||
		strings.Contains(lower, "not expected") {
		return "MILD"
	}
	return defaultSeverity
}

// isFalsePositiveDrug checks if a captured "drug name" is actually a common word
func (g *DDIGrammar) isFalsePositiveDrug(name string) bool {
	if name == "" || len(name) < 3 {
		return true
	}
	return g.falsePositiveDrugs.MatchString(name)
}

// splitIntoSentences does basic sentence splitting on period boundaries
// while respecting abbreviations and decimal numbers.
func splitIntoSentences(text string) []string {
	// Replace common abbreviations to prevent false splits
	replacer := strings.NewReplacer(
		"e.g.", "eg",
		"i.e.", "ie",
		"vs.", "vs",
		"Dr.", "Dr",
		"Mr.", "Mr",
		"Mrs.", "Mrs",
		"St.", "St",
		"U.S.", "US",
	)
	cleaned := replacer.Replace(text)

	// Split on sentence boundaries: period/semicolon followed by space and uppercase.
	// Go's regexp (RE2) doesn't support lookaheads, so we match the full boundary
	// and reconstruct sentences by prepending the captured uppercase letter.
	sentenceRe := regexp.MustCompile(`([.;])\s+([A-Z])`)
	// Replace boundary with a unique delimiter, keeping the uppercase letter
	delimited := sentenceRe.ReplaceAllString(cleaned, "$1\x00$2")
	parts := strings.Split(delimited, "\x00")

	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 15 {
			sentences = append(sentences, p)
		}
	}

	return sentences
}

// extractMechanismFromSentence tries to extract a pharmacological mechanism
func extractMechanismFromSentence(sentence, enzyme string) string {
	if enzyme != "" {
		lower := strings.ToLower(sentence)
		if strings.Contains(lower, "inhibit") {
			return "Inhibition of " + enzyme
		}
		if strings.Contains(lower, "induc") {
			return "Induction of " + enzyme
		}
		if strings.Contains(lower, "substrate") {
			return "Substrate of " + enzyme
		}
		return enzyme + " pathway"
	}

	// Try to extract mechanism keywords
	mechRe := regexp.MustCompile(`(?i)((?:inhibit|induc|compet|displac|alter|block|stimulat)[a-z]*\s+(?:of\s+)?(?:CYP\s*\w+|P-?gp|renal\s+(?:clearance|tubular|secretion)|hepatic\s+metabolism|protein\s+binding|absorption))`)
	if m := mechRe.FindStringSubmatch(sentence); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	return ""
}

// extractEffectContext pulls clinical effect context from the sentence
func extractEffectContext(sentence string) string {
	effectRe := regexp.MustCompile(`(?i)(?:result\s+in|cause|lead\s+to|produce|increase[sd]?\s+risk\s+of)\s+([^.]{5,80})`)
	if m := effectRe.FindStringSubmatch(sentence); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// isSubstringMatch checks if either string contains the other (case-insensitive)
func isSubstringMatch(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	return strings.Contains(la, lb) || strings.Contains(lb, la)
}

// truncatePhrase limits a source phrase to maxLen characters
func truncatePhrase(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
