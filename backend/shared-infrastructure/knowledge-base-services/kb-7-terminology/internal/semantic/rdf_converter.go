package semantic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RDFConverter handles conversion between PostgreSQL data and RDF/Turtle format
type RDFConverter struct {
	logger *logrus.Logger
	config *RDFConfig
}

// RDFConfig contains configuration for RDF conversion
type RDFConfig struct {
	BaseURI       string
	DefaultGraph  string
	Namespaces    map[string]string
	OutputFormat  string // "turtle", "rdf-xml", "n-triples"
	IncludeProvenance bool
}

// ConceptData represents concept data from PostgreSQL
type ConceptData struct {
	ID           int64             `json:"id"`
	ConceptID    string            `json:"concept_id"`
	System       string            `json:"system"`
	Code         string            `json:"code"`
	Display      string            `json:"display"`
	Definition   string            `json:"definition"`
	Status       string            `json:"status"`
	Properties   map[string]string `json:"properties"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// MappingData represents mapping data from PostgreSQL
type MappingData struct {
	ID               int64             `json:"id"`
	SourceSystem     string            `json:"source_system"`
	SourceCode       string            `json:"source_code"`
	TargetSystem     string            `json:"target_system"`
	TargetCode       string            `json:"target_code"`
	MappingType      string            `json:"mapping_type"`
	Confidence       float64           `json:"confidence"`
	Status           string            `json:"status"`
	PolicyFlags      map[string]interface{} `json:"policy_flags"`
	ClinicalJustification string        `json:"clinical_justification"`
	CreatedBy        string            `json:"created_by"`
	ReviewedBy       string            `json:"reviewed_by"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// NewRDFConverter creates a new RDF converter
func NewRDFConverter(config *RDFConfig, logger *logrus.Logger) *RDFConverter {
	if config == nil {
		config = &RDFConfig{
			BaseURI:       "http://cardiofit.ai/kb7/",
			DefaultGraph:  "http://cardiofit.ai/kb7/graph/default",
			OutputFormat:  "turtle",
			IncludeProvenance: true,
			Namespaces: map[string]string{
				"kb7":    "http://cardiofit.ai/kb7/ontology#",
				"sct":    "http://snomed.info/id/",
				"rxnorm": "http://purl.bioontology.org/ontology/RXNORM/",
				"loinc":  "http://loinc.org/",
				"icd10":  "http://purl.bioontology.org/ontology/ICD10CM/",
				"amt":    "http://nehta.gov.au/amt#",
				"prov":   "http://www.w3.org/ns/prov#",
				"pav":    "http://purl.org/pav/",
				"rdfs":   "http://www.w3.org/2000/01/rdf-schema#",
				"skos":   "http://www.w3.org/2004/02/skos/core#",
				"owl":    "http://www.w3.org/2002/07/owl#",
				"xsd":    "http://www.w3.org/2001/XMLSchema#",
			},
		}
	}

	return &RDFConverter{
		logger: logger,
		config: config,
	}
}

// ConvertConceptsToTurtle converts PostgreSQL concept data to Turtle format
func (r *RDFConverter) ConvertConceptsToTurtle(ctx context.Context, concepts []ConceptData, writer io.Writer) error {
	r.logger.WithField("conceptCount", len(concepts)).Info("Converting concepts to Turtle format")

	// Write prefixes
	if err := r.writePrefixes(writer); err != nil {
		return fmt.Errorf("writing prefixes: %w", err)
	}

	// Write concepts
	for _, concept := range concepts {
		if err := r.writeConceptAsTurtle(writer, concept); err != nil {
			return fmt.Errorf("writing concept %s: %w", concept.ConceptID, err)
		}
	}

	r.logger.Info("Concepts converted to Turtle successfully")
	return nil
}

// ConvertMappingsToTurtle converts PostgreSQL mapping data to Turtle format
func (r *RDFConverter) ConvertMappingsToTurtle(ctx context.Context, mappings []MappingData, writer io.Writer) error {
	r.logger.WithField("mappingCount", len(mappings)).Info("Converting mappings to Turtle format")

	// Write prefixes
	if err := r.writePrefixes(writer); err != nil {
		return fmt.Errorf("writing prefixes: %w", err)
	}

	// Write mappings
	for _, mapping := range mappings {
		if err := r.writeMappingAsTurtle(writer, mapping); err != nil {
			return fmt.Errorf("writing mapping %d: %w", mapping.ID, err)
		}
	}

	r.logger.Info("Mappings converted to Turtle successfully")
	return nil
}

// ConvertFileToTurtle converts various RDF formats to Turtle
func (r *RDFConverter) ConvertFileToTurtle(ctx context.Context, inputPath, outputPath string) error {
	inputExt := strings.ToLower(filepath.Ext(inputPath))

	// Check if input is already in Turtle format
	if inputExt == ".ttl" {
		r.logger.Info("Input file is already in Turtle format, copying...")
		return r.copyFile(inputPath, outputPath)
	}

	// Handle different input formats
	switch inputExt {
	case ".owl", ".rdf":
		return r.convertOWLToTurtle(inputPath, outputPath)
	case ".nt":
		return r.convertNTriplesToTurtle(inputPath, outputPath)
	case ".jsonld":
		return r.convertJSONLDToTurtle(inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported input format: %s", inputExt)
	}
}

// ValidateTurtleFile validates Turtle syntax
func (r *RDFConverter) ValidateTurtleFile(ctx context.Context, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return r.ValidateTurtleStream(ctx, file)
}

// ValidateTurtleStream validates Turtle syntax from a stream
func (r *RDFConverter) ValidateTurtleStream(ctx context.Context, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	// Basic Turtle syntax validation patterns
	prefixPattern := regexp.MustCompile(`^@prefix\s+\w*:\s+<[^>]+>\s+\.$`)
	triplePattern := regexp.MustCompile(`^[^#]*\s+[^#]*\s+[^#]*\s*\.$`)
	commentPattern := regexp.MustCompile(`^\s*#.*$`)
	emptyLinePattern := regexp.MustCompile(`^\s*$`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if emptyLinePattern.MatchString(line) || commentPattern.MatchString(line) {
			continue
		}

		// Check prefix declarations
		if strings.HasPrefix(line, "@prefix") {
			if !prefixPattern.MatchString(line) {
				return fmt.Errorf("invalid prefix syntax at line %d: %s", lineNum, line)
			}
			continue
		}

		// Check base declarations
		if strings.HasPrefix(line, "@base") {
			continue
		}

		// Check triple statements
		if strings.Contains(line, " ") && strings.HasSuffix(line, ".") {
			if !triplePattern.MatchString(line) {
				r.logger.WithFields(logrus.Fields{
					"line":    lineNum,
					"content": line,
				}).Warn("Potentially invalid triple syntax")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning file: %w", err)
	}

	r.logger.WithField("totalLines", lineNum).Info("Turtle validation completed")
	return nil
}

// Private helper methods

func (r *RDFConverter) writePrefixes(writer io.Writer) error {
	// Write standard prefixes
	for prefix, uri := range r.config.Namespaces {
		if _, err := fmt.Fprintf(writer, "@prefix %s: <%s> .\n", prefix, uri); err != nil {
			return err
		}
	}

	// Add base URI
	if _, err := fmt.Fprintf(writer, "@base <%s> .\n\n", r.config.BaseURI); err != nil {
		return err
	}

	return nil
}

func (r *RDFConverter) writeConceptAsTurtle(writer io.Writer, concept ConceptData) error {
	// Generate concept URI
	conceptURI := r.generateConceptURI(concept.System, concept.Code)

	// Write concept as RDF
	if _, err := fmt.Fprintf(writer, "%s a kb7:ClinicalConcept ;\n", conceptURI); err != nil {
		return err
	}

	// Basic properties
	if _, err := fmt.Fprintf(writer, "    kb7:conceptId \"%s\" ;\n", r.escapeString(concept.ConceptID)); err != nil {
		return err
	}

	if concept.Display != "" {
		if _, err := fmt.Fprintf(writer, "    rdfs:label \"%s\" ;\n", r.escapeString(concept.Display)); err != nil {
			return err
		}
	}

	if concept.Definition != "" {
		if _, err := fmt.Fprintf(writer, "    rdfs:comment \"%s\" ;\n", r.escapeString(concept.Definition)); err != nil {
			return err
		}
	}

	// Status and system
	if _, err := fmt.Fprintf(writer, "    kb7:status \"%s\" ;\n", concept.Status); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "    kb7:terminologySystem \"%s\" ;\n", concept.System); err != nil {
		return err
	}

	// Additional properties
	for key, value := range concept.Properties {
		if _, err := fmt.Fprintf(writer, "    kb7:%s \"%s\" ;\n", key, r.escapeString(value)); err != nil {
			return err
		}
	}

	// Provenance if enabled
	if r.config.IncludeProvenance {
		if _, err := fmt.Fprintf(writer, "    pav:createdOn \"%s\"^^xsd:dateTime ;\n", concept.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(writer, "    pav:lastUpdatedOn \"%s\"^^xsd:dateTime ;\n", concept.UpdatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}

	// End the concept (remove last semicolon, add period)
	if _, err := fmt.Fprint(writer, "    .\n\n"); err != nil {
		return err
	}

	return nil
}

func (r *RDFConverter) writeMappingAsTurtle(writer io.Writer, mapping MappingData) error {
	// Generate mapping URI
	mappingURI := fmt.Sprintf("kb7:mapping_%s_%s_to_%s_%s",
		r.sanitizeForURI(mapping.SourceSystem),
		r.sanitizeForURI(mapping.SourceCode),
		r.sanitizeForURI(mapping.TargetSystem),
		r.sanitizeForURI(mapping.TargetCode))

	// Determine mapping class based on type
	mappingClass := "kb7:ConceptMapping"
	switch strings.ToLower(mapping.MappingType) {
	case "equivalent":
		mappingClass = "kb7:EquivalentMapping"
	case "approximate":
		mappingClass = "kb7:ApproximateMapping"
	case "hierarchical":
		mappingClass = "kb7:HierarchicalMapping"
	}

	// Write mapping as RDF
	if _, err := fmt.Fprintf(writer, "%s a %s ;\n", mappingURI, mappingClass); err != nil {
		return err
	}

	// Source and target
	if _, err := fmt.Fprintf(writer, "    kb7:sourceCode \"%s\" ;\n", r.escapeString(mapping.SourceCode)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "    kb7:targetCode \"%s\" ;\n", r.escapeString(mapping.TargetCode)); err != nil {
		return err
	}

	// Source and target URIs
	sourceURI := r.generateConceptURI(mapping.SourceSystem, mapping.SourceCode)
	targetURI := r.generateConceptURI(mapping.TargetSystem, mapping.TargetCode)

	if _, err := fmt.Fprintf(writer, "    kb7:mapsTo %s, %s ;\n", sourceURI, targetURI); err != nil {
		return err
	}

	// Mapping properties
	if _, err := fmt.Fprintf(writer, "    kb7:mappingConfidence \"%.3f\"^^xsd:decimal ;\n", mapping.Confidence); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "    kb7:validationStatus \"%s\" ;\n", mapping.Status); err != nil {
		return err
	}

	// Clinical justification
	if mapping.ClinicalJustification != "" {
		if _, err := fmt.Fprintf(writer, "    rdfs:comment \"%s\" ;\n", r.escapeString(mapping.ClinicalJustification)); err != nil {
			return err
		}
	}

	// Policy flags
	for key, value := range mapping.PolicyFlags {
		if boolVal, ok := value.(bool); ok {
			if _, err := fmt.Fprintf(writer, "    kb7:%s \"%t\"^^xsd:boolean ;\n", key, boolVal); err != nil {
				return err
			}
		} else if strVal, ok := value.(string); ok {
			if _, err := fmt.Fprintf(writer, "    kb7:%s \"%s\" ;\n", key, r.escapeString(strVal)); err != nil {
				return err
			}
		}
	}

	// Provenance
	if r.config.IncludeProvenance {
		if mapping.CreatedBy != "" {
			if _, err := fmt.Fprintf(writer, "    pav:createdBy <%s/users/%s> ;\n", r.config.BaseURI, mapping.CreatedBy); err != nil {
				return err
			}
		}

		if mapping.ReviewedBy != "" {
			if _, err := fmt.Fprintf(writer, "    kb7:reviewedBy <%s/reviewers/%s> ;\n", r.config.BaseURI, mapping.ReviewedBy); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(writer, "    pav:createdOn \"%s\"^^xsd:dateTime ;\n", mapping.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(writer, "    pav:lastUpdatedOn \"%s\"^^xsd:dateTime ;\n", mapping.UpdatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}

	// End the mapping
	if _, err := fmt.Fprint(writer, "    .\n\n"); err != nil {
		return err
	}

	return nil
}

func (r *RDFConverter) generateConceptURI(system, code string) string {
	// Map common systems to their standard URIs
	switch strings.ToLower(system) {
	case "snomed", "snomed-ct", "snomedct":
		return fmt.Sprintf("sct:%s", code)
	case "rxnorm":
		return fmt.Sprintf("rxnorm:%s", code)
	case "loinc":
		return fmt.Sprintf("loinc:%s", code)
	case "icd10", "icd-10":
		return fmt.Sprintf("icd10:%s", code)
	case "icd10am", "icd-10-am":
		return fmt.Sprintf("icd10am:%s", code)
	case "amt":
		return fmt.Sprintf("amt:%s", code)
	default:
		return fmt.Sprintf("kb7:concept_%s_%s", r.sanitizeForURI(system), r.sanitizeForURI(code))
	}
}

func (r *RDFConverter) escapeString(s string) string {
	// Escape quotes and special characters for Turtle literals
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (r *RDFConverter) sanitizeForURI(s string) string {
	// Replace spaces and special characters for URI components
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	return reg.ReplaceAllString(s, "_")
}

func (r *RDFConverter) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// Placeholder methods for format conversion (would need external libraries like rdflib or jena)
func (r *RDFConverter) convertOWLToTurtle(inputPath, outputPath string) error {
	return fmt.Errorf("OWL to Turtle conversion not implemented - use ROBOT tool")
}

func (r *RDFConverter) convertNTriplesToTurtle(inputPath, outputPath string) error {
	return fmt.Errorf("N-Triples to Turtle conversion not implemented - use ROBOT tool")
}

func (r *RDFConverter) convertJSONLDToTurtle(inputPath, outputPath string) error {
	return fmt.Errorf("JSON-LD to Turtle conversion not implemented - use ROBOT tool")
}