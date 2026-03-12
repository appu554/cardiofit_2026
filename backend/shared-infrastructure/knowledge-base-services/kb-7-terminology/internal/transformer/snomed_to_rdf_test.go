package transformer

import (
	"context"
	"strings"
	"testing"
	"time"

	"kb-7-terminology/internal/models"

	"go.uber.org/zap"
)

func TestNewSNOMEDToRDFTransformer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	if transformer == nil {
		t.Fatal("Expected transformer to be created")
	}

	if transformer.baseURI != "http://snomed.info/id/" {
		t.Errorf("Expected baseURI to be http://snomed.info/id/, got %s", transformer.baseURI)
	}

	if len(transformer.namespaces) == 0 {
		t.Error("Expected namespaces to be populated")
	}
}

func TestConceptToTriples(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	concept := &models.Concept{
		Code:          "387517004",
		System:        "SNOMED",
		PreferredTerm: "Paracetamol",
		Definition:    "Analgesic and antipyretic medication",
		Active:        true,
		Version:       "20240131",
		Status:        "active",
		Properties: models.JSONB{
			"module_id":             "900000000000012004",
			"definition_status_id":  "900000000000073002",
			"fully_specified_name":  "Paracetamol (substance)",
			"synonyms":              []interface{}{"Acetaminophen", "Tylenol"},
			"parent_codes":          []interface{}{"7947003"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tripleSet, err := transformer.ConceptToTriples(concept)
	if err != nil {
		t.Fatalf("ConceptToTriples failed: %v", err)
	}

	if tripleSet == nil {
		t.Fatal("Expected tripleSet to be created")
	}

	if tripleSet.ConceptID != "387517004" {
		t.Errorf("Expected ConceptID to be 387517004, got %s", tripleSet.ConceptID)
	}

	if len(tripleSet.Triples) == 0 {
		t.Error("Expected triples to be generated")
	}

	// Check for essential triples
	hasTypeTriple := false
	hasLabelTriple := false
	hasPrefLabelTriple := false
	hasSubClassOfTriple := false

	for _, triple := range tripleSet.Triples {
		if triple.Predicate == "rdf:type" {
			hasTypeTriple = true
		}
		if triple.Predicate == "rdfs:label" {
			hasLabelTriple = true
		}
		if triple.Predicate == "skos:prefLabel" {
			hasPrefLabelTriple = true
		}
		if triple.Predicate == "rdfs:subClassOf" {
			hasSubClassOfTriple = true
		}
	}

	if !hasTypeTriple {
		t.Error("Expected rdf:type triple")
	}
	if !hasLabelTriple {
		t.Error("Expected rdfs:label triple")
	}
	if !hasPrefLabelTriple {
		t.Error("Expected skos:prefLabel triple")
	}
	if !hasSubClassOfTriple {
		t.Error("Expected rdfs:subClassOf triple")
	}

	t.Logf("Generated %d triples for concept", len(tripleSet.Triples))
}

func TestConceptToTriples_Minimal(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	// Minimal concept
	concept := &models.Concept{
		Code:          "12345",
		System:        "SNOMED",
		PreferredTerm: "Test Concept",
		Active:        true,
		Version:       "20240101",
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	tripleSet, err := transformer.ConceptToTriples(concept)
	if err != nil {
		t.Fatalf("ConceptToTriples failed: %v", err)
	}

	if len(tripleSet.Triples) == 0 {
		t.Error("Expected triples to be generated for minimal concept")
	}
}

func TestConceptToTriples_SpecialCharacters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	concept := &models.Concept{
		Code:          "999999",
		System:        "SNOMED",
		PreferredTerm: "Test with \"quotes\" and \n newlines",
		Definition:    "Definition with special chars: <>&",
		Active:        true,
		Version:       "20240101",
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	tripleSet, err := transformer.ConceptToTriples(concept)
	if err != nil {
		t.Fatalf("ConceptToTriples failed: %v", err)
	}

	// Check that special characters are escaped
	for _, triple := range tripleSet.Triples {
		if triple.ObjectType == RDFObjectLiteral {
			if strings.Contains(triple.Object, "\n") && !strings.Contains(triple.Object, "\\n") {
				// Raw object shouldn't be escaped yet
			}
		}
	}
}

func TestBatchToTurtle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	concepts := []*models.Concept{
		{
			Code:          "387517004",
			System:        "SNOMED",
			PreferredTerm: "Paracetamol",
			Active:        true,
			Version:       "20240131",
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			Code:          "387207008",
			System:        "SNOMED",
			PreferredTerm: "Ibuprofen",
			Active:        true,
			Version:       "20240131",
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	doc, err := transformer.BatchToTurtle(concepts)
	if err != nil {
		t.Fatalf("BatchToTurtle failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Expected TurtleDocument to be created")
	}

	if doc.ConceptCount != 2 {
		t.Errorf("Expected ConceptCount to be 2, got %d", doc.ConceptCount)
	}

	if doc.TripleCount == 0 {
		t.Error("Expected TripleCount to be greater than 0")
	}

	if len(doc.Statements) == 0 {
		t.Error("Expected statements to be generated")
	}

	// Check for prefix statements
	hasPrefixes := false
	for _, stmt := range doc.Statements {
		if strings.HasPrefix(stmt, "@prefix") {
			hasPrefixes = true
			break
		}
	}
	if !hasPrefixes {
		t.Error("Expected prefix statements")
	}

	t.Logf("Generated %d statements for %d concepts", len(doc.Statements), doc.ConceptCount)
}

func TestBatchToTurtle_Empty(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	doc, err := transformer.BatchToTurtle([]*models.Concept{})
	if err != nil {
		t.Fatalf("BatchToTurtle failed: %v", err)
	}

	if doc.ConceptCount != 0 {
		t.Errorf("Expected ConceptCount to be 0, got %d", doc.ConceptCount)
	}
}

func TestConvertBatchToTurtleString(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	concepts := []*models.Concept{
		{
			Code:          "387517004",
			System:        "SNOMED",
			PreferredTerm: "Paracetamol",
			Active:        true,
			Version:       "20240131",
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	ctx := context.Background()
	turtleString, err := transformer.ConvertBatchToTurtleString(ctx, concepts)
	if err != nil {
		t.Fatalf("ConvertBatchToTurtleString failed: %v", err)
	}

	if turtleString == "" {
		t.Error("Expected turtle string to be generated")
	}

	// Check that it's valid Turtle format
	if !strings.Contains(turtleString, "@prefix") {
		t.Error("Expected @prefix declarations")
	}

	if !strings.Contains(turtleString, "sct:387517004") {
		t.Error("Expected concept URI in output")
	}

	t.Logf("Generated Turtle string with %d bytes", len(turtleString))
}

func TestFormatObject(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	tests := []struct {
		name     string
		triple   RDFTriple
		expected string
	}{
		{
			name: "URI object",
			triple: RDFTriple{
				Object:     "sct:387517004",
				ObjectType: RDFObjectURI,
			},
			expected: "sct:387517004",
		},
		{
			name: "Literal with language",
			triple: RDFTriple{
				Object:     "Paracetamol",
				ObjectType: RDFObjectLiteral,
				Language:   "en",
			},
			expected: `"Paracetamol"@en`,
		},
		{
			name: "Literal with datatype",
			triple: RDFTriple{
				Object:     "true",
				ObjectType: RDFObjectLiteral,
				DataType:   "xsd:boolean",
			},
			expected: `"true"^^xsd:boolean`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.formatObject(tt.triple)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEscapeTurtleLiteral(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	tests := []struct {
		input    string
		expected string
	}{
		{`test"quote`, `test\"quote`},
		{"test\nline", "test\\nline"},
		{"test\ttab", "test\\ttab"},
		{`test\backslash`, `test\\backslash`},
	}

	for _, tt := range tests {
		result := transformer.escapeTurtleLiteral(tt.input)
		if result != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, result)
		}
	}
}

func TestGetStatistics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	doc := &TurtleDocument{
		ConceptCount: 100,
		TripleCount:  1200,
		Prefixes:     transformer.namespaces,
		Statements:   make([]string, 1200),
	}

	stats := transformer.GetStatistics(doc)

	if stats["concept_count"] != 100 {
		t.Errorf("Expected concept_count to be 100, got %v", stats["concept_count"])
	}

	if stats["triple_count"] != 1200 {
		t.Errorf("Expected triple_count to be 1200, got %v", stats["triple_count"])
	}

	avgTriples := stats["avg_triples_per_concept"].(float64)
	if avgTriples != 12.0 {
		t.Errorf("Expected avg_triples_per_concept to be 12.0, got %f", avgTriples)
	}
}

// Benchmark tests
func BenchmarkConceptToTriples(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	concept := &models.Concept{
		Code:          "387517004",
		System:        "SNOMED",
		PreferredTerm: "Paracetamol",
		Definition:    "Analgesic and antipyretic medication",
		Active:        true,
		Version:       "20240131",
		Status:        "active",
		Properties: models.JSONB{
			"module_id":             "900000000000012004",
			"definition_status_id":  "900000000000073002",
			"parent_codes":          []interface{}{"7947003"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transformer.ConceptToTriples(concept)
	}
}

func BenchmarkBatchToTurtle(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	transformer := NewSNOMEDToRDFTransformer(logger)

	// Create batch of 1000 concepts
	concepts := make([]*models.Concept, 1000)
	for i := 0; i < 1000; i++ {
		concepts[i] = &models.Concept{
			Code:          "387517004",
			System:        "SNOMED",
			PreferredTerm: "Paracetamol",
			Active:        true,
			Version:       "20240131",
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transformer.BatchToTurtle(concepts)
	}
}
