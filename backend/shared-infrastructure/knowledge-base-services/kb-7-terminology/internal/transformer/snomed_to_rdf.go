package transformer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/models"

	"go.uber.org/zap"
)

// SNOMEDToRDFTransformer converts SNOMED CT concepts to RDF triples
type SNOMEDToRDFTransformer struct {
	baseURI    string
	namespaces map[string]string
	logger     *zap.Logger
}

// RDFTriple represents a single RDF triple (subject-predicate-object)
type RDFTriple struct {
	Subject    string        `json:"subject"`
	Predicate  string        `json:"predicate"`
	Object     string        `json:"object"`
	ObjectType RDFObjectType `json:"object_type"`
	DataType   string        `json:"data_type"`
	Language   string        `json:"language"`
	Graph      string        `json:"graph"`
}

// RDFObjectType defines the type of RDF object
type RDFObjectType string

const (
	RDFObjectURI       RDFObjectType = "uri"
	RDFObjectLiteral   RDFObjectType = "literal"
	RDFObjectBlankNode RDFObjectType = "blank_node"
)

// RDFTripleSet represents a batch of triples for a single concept
type RDFTripleSet struct {
	ConceptID string      `json:"concept_id"`
	System    string      `json:"system"`
	Triples   []RDFTriple `json:"triples"`
	CreatedAt time.Time   `json:"created_at"`
}

// TurtleDocument represents a complete Turtle RDF document
type TurtleDocument struct {
	Prefixes     map[string]string `json:"prefixes"`
	Statements   []string          `json:"statements"`
	TripleCount  int               `json:"triple_count"`
	ConceptCount int               `json:"concept_count"`
}

// NewSNOMEDToRDFTransformer creates a new SNOMED to RDF transformer
func NewSNOMEDToRDFTransformer(logger *zap.Logger) *SNOMEDToRDFTransformer {
	return &SNOMEDToRDFTransformer{
		baseURI: "http://snomed.info/id/",
		namespaces: map[string]string{
			"sct":  "http://snomed.info/id/",
			"kb7":  "http://cardiofit.ai/kb7/ontology#",
			"rdfs": "http://www.w3.org/2000/01/rdf-schema#",
			"skos": "http://www.w3.org/2004/02/skos/core#",
			"owl":  "http://www.w3.org/2002/07/owl#",
			"xsd":  "http://www.w3.org/2001/XMLSchema#",
			"rdf":  "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
		},
		logger: logger,
	}
}

// ConceptToTriples converts a PostgreSQL Concept to RDF triples
func (t *SNOMEDToRDFTransformer) ConceptToTriples(concept *models.Concept) (*RDFTripleSet, error) {
	if concept == nil {
		return nil, fmt.Errorf("concept is nil")
	}

	triples := &RDFTripleSet{
		ConceptID: concept.Code,
		System:    concept.System,
		Triples:   make([]RDFTriple, 0, 24),
		CreatedAt: time.Now(),
	}

	subjectURI := t.buildConceptURI(concept.Code)

	// 1. Type declaration (owl:Class)
	triples.Triples = append(triples.Triples, RDFTriple{
		Subject:    subjectURI,
		Predicate:  "rdf:type",
		Object:     "owl:Class",
		ObjectType: RDFObjectURI,
	})

	// 2. Preferred term (rdfs:label + skos:prefLabel)
	if concept.PreferredTerm != "" {
		triples.Triples = append(triples.Triples,
			RDFTriple{
				Subject:    subjectURI,
				Predicate:  "rdfs:label",
				Object:     concept.PreferredTerm,
				ObjectType: RDFObjectLiteral,
				Language:   "en",
			},
			RDFTriple{
				Subject:    subjectURI,
				Predicate:  "skos:prefLabel",
				Object:     concept.PreferredTerm,
				ObjectType: RDFObjectLiteral,
				Language:   "en",
			},
		)
	}

	// 3. Definition (skos:definition)
	if concept.Definition != "" {
		triples.Triples = append(triples.Triples, RDFTriple{
			Subject:    subjectURI,
			Predicate:  "skos:definition",
			Object:     concept.Definition,
			ObjectType: RDFObjectLiteral,
			Language:   "en",
		})
	}

	// 4. Metadata attributes
	triples.Triples = append(triples.Triples,
		t.createLiteralTriple(subjectURI, "kb7:conceptId", concept.Code, "xsd:string"),
		t.createLiteralTriple(subjectURI, "kb7:system", concept.System, "xsd:string"),
		t.createLiteralTriple(subjectURI, "kb7:version", concept.Version, "xsd:string"),
		t.createLiteralTriple(subjectURI, "kb7:active", fmt.Sprintf("%t", concept.Active), "xsd:boolean"),
	)

	// Add status if available
	if concept.Status != "" {
		triples.Triples = append(triples.Triples,
			t.createLiteralTriple(subjectURI, "kb7:status", concept.Status, "xsd:string"))
	}

	// 5. Parent relationships (rdfs:subClassOf) from properties
	if concept.Properties != nil {
		// Handle parent_codes from JSONB properties
		if parentCodes, ok := concept.Properties["parent_codes"].([]interface{}); ok {
			for _, parentCode := range parentCodes {
				if pc, ok := parentCode.(string); ok && pc != "" {
					parentURI := t.buildConceptURI(pc)
					triples.Triples = append(triples.Triples, RDFTriple{
						Subject:    subjectURI,
						Predicate:  "rdfs:subClassOf",
						Object:     parentURI,
						ObjectType: RDFObjectURI,
					})
				}
			}
		}

		// 6. SNOMED-specific properties from JSONB
		if moduleID, ok := concept.Properties["module_id"].(string); ok && moduleID != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:moduleId", moduleID, "xsd:string"))
		}

		if defStatusID, ok := concept.Properties["definition_status_id"].(string); ok && defStatusID != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:definitionStatusId", defStatusID, "xsd:string"))
		}

		// 7. Synonyms (skos:altLabel)
		if synonyms, ok := concept.Properties["synonyms"].([]interface{}); ok {
			for _, synonym := range synonyms {
				if syn, ok := synonym.(string); ok && syn != "" {
					triples.Triples = append(triples.Triples, RDFTriple{
						Subject:    subjectURI,
						Predicate:  "skos:altLabel",
						Object:     syn,
						ObjectType: RDFObjectLiteral,
						Language:   "en",
					})
				}
			}
		}

		// 8. Fully Specified Name (FSN)
		if fsn, ok := concept.Properties["fully_specified_name"].(string); ok && fsn != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:fullySpecifiedName", fsn, "xsd:string"))
		}

		// 9. Additional SNOMED attributes
		if effectiveTime, ok := concept.Properties["effective_time"].(string); ok && effectiveTime != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:effectiveTime", effectiveTime, "xsd:dateTime"))
		}

		// 10. Description type
		if descType, ok := concept.Properties["description_type"].(string); ok && descType != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:descriptionType", descType, "xsd:string"))
		}

		// 11. Case significance
		if caseSign, ok := concept.Properties["case_significance"].(string); ok && caseSign != "" {
			triples.Triples = append(triples.Triples,
				t.createLiteralTriple(subjectURI, "kb7:caseSignificance", caseSign, "xsd:string"))
		}
	}

	// 12. Timestamps
	triples.Triples = append(triples.Triples,
		t.createLiteralTriple(subjectURI, "kb7:createdAt", concept.CreatedAt.Format(time.RFC3339), "xsd:dateTime"),
		t.createLiteralTriple(subjectURI, "kb7:updatedAt", concept.UpdatedAt.Format(time.RFC3339), "xsd:dateTime"))

	return triples, nil
}

// BatchToTurtle converts multiple concepts to a single Turtle document
func (t *SNOMEDToRDFTransformer) BatchToTurtle(concepts []*models.Concept) (*TurtleDocument, error) {
	doc := &TurtleDocument{
		Prefixes:     t.namespaces,
		Statements:   make([]string, 0, len(concepts)*15),
		ConceptCount: len(concepts),
	}

	// Write prefix declarations
	prefixStatements := t.generatePrefixStatements()
	doc.Statements = append(doc.Statements, prefixStatements...)
	doc.Statements = append(doc.Statements, "") // Empty line after prefixes

	// Convert each concept to triples
	for _, concept := range concepts {
		tripleSet, err := t.ConceptToTriples(concept)
		if err != nil {
			t.logger.Warn("Failed to convert concept",
				zap.String("code", concept.Code),
				zap.Error(err))
			continue
		}

		// Convert triples to Turtle statements
		statements := t.triplesToTurtle(tripleSet.Triples)
		doc.Statements = append(doc.Statements, statements...)
		doc.Statements = append(doc.Statements, "") // Empty line between concepts
		doc.TripleCount += len(tripleSet.Triples)
	}

	return doc, nil
}

// ConvertBatchToTurtleString converts a batch to a Turtle string
func (t *SNOMEDToRDFTransformer) ConvertBatchToTurtleString(ctx context.Context, concepts []*models.Concept) (string, error) {
	doc, err := t.BatchToTurtle(concepts)
	if err != nil {
		return "", err
	}

	return strings.Join(doc.Statements, "\n"), nil
}

// Helper methods

func (t *SNOMEDToRDFTransformer) buildConceptURI(code string) string {
	return fmt.Sprintf("sct:%s", code)
}

func (t *SNOMEDToRDFTransformer) createLiteralTriple(subject, predicate, value, datatype string) RDFTriple {
	return RDFTriple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     value,
		ObjectType: RDFObjectLiteral,
		DataType:   datatype,
	}
}

func (t *SNOMEDToRDFTransformer) generatePrefixStatements() []string {
	statements := make([]string, 0, len(t.namespaces))
	for prefix, uri := range t.namespaces {
		statements = append(statements, fmt.Sprintf("@prefix %s: <%s> .", prefix, uri))
	}
	return statements
}

func (t *SNOMEDToRDFTransformer) triplesToTurtle(triples []RDFTriple) []string {
	if len(triples) == 0 {
		return nil
	}

	// Group triples by subject for compact Turtle syntax
	grouped := make(map[string][]RDFTriple)
	for _, triple := range triples {
		grouped[triple.Subject] = append(grouped[triple.Subject], triple)
	}

	statements := make([]string, 0)
	for subject, subjectTriples := range grouped {
		var builder strings.Builder
		builder.WriteString(subject)
		builder.WriteString(" ")

		predicates := make([]string, 0)
		for i, triple := range subjectTriples {
			object := t.formatObject(triple)

			if i == 0 {
				predicates = append(predicates, fmt.Sprintf("%s %s", triple.Predicate, object))
			} else {
				predicates = append(predicates, fmt.Sprintf("    %s %s", triple.Predicate, object))
			}
		}

		builder.WriteString(strings.Join(predicates, " ;\n"))
		builder.WriteString(" .")
		statements = append(statements, builder.String())
	}

	return statements
}

func (t *SNOMEDToRDFTransformer) formatObject(triple RDFTriple) string {
	switch triple.ObjectType {
	case RDFObjectURI:
		// URI objects are already prefixed
		return triple.Object
	case RDFObjectLiteral:
		// Escape special characters
		escaped := t.escapeTurtleLiteral(triple.Object)

		// Add language tag if present
		if triple.Language != "" {
			return fmt.Sprintf(`"%s"@%s`, escaped, triple.Language)
		}

		// Add datatype if present
		if triple.DataType != "" {
			return fmt.Sprintf(`"%s"^^%s`, escaped, triple.DataType)
		}

		// Plain literal
		return fmt.Sprintf(`"%s"`, escaped)
	case RDFObjectBlankNode:
		return triple.Object
	default:
		return fmt.Sprintf(`"%s"`, triple.Object)
	}
}

func (t *SNOMEDToRDFTransformer) escapeTurtleLiteral(value string) string {
	// Escape special characters for Turtle format
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}

// GetNamespaces returns the namespace map
func (t *SNOMEDToRDFTransformer) GetNamespaces() map[string]string {
	return t.namespaces
}

// GetStatistics returns transformation statistics
func (t *SNOMEDToRDFTransformer) GetStatistics(doc *TurtleDocument) map[string]interface{} {
	return map[string]interface{}{
		"concept_count":       doc.ConceptCount,
		"triple_count":        doc.TripleCount,
		"avg_triples_per_concept": float64(doc.TripleCount) / float64(doc.ConceptCount),
		"statement_count":     len(doc.Statements),
		"namespace_count":     len(doc.Prefixes),
	}
}
