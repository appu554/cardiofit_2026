// Package api provides HTTP handlers for KB-13 Quality Measures Engine.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-13-quality-measures/internal/models"
)

// ListMeasuresResponse represents the response for listing measures.
type ListMeasuresResponse struct {
	Measures []*models.Measure `json:"measures"`
	Total    int               `json:"total"`
}

// ListMeasures handles GET /v1/measures - returns all measures.
func (s *Server) ListMeasures(c *gin.Context) {
	// Check for active-only filter
	activeOnly := c.Query("active") == "true"

	var measures []*models.Measure
	if activeOnly {
		measures = s.store.GetActiveMeasures()
	} else {
		measures = s.store.GetAllMeasures()
	}

	c.JSON(http.StatusOK, ListMeasuresResponse{
		Measures: measures,
		Total:    len(measures),
	})
}

// GetMeasure handles GET /v1/measures/:id - returns a single measure.
func (s *Server) GetMeasure(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Measure ID is required",
		})
		return
	}

	measure := s.store.GetMeasure(id)
	if measure == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Measure not found",
			"id":      id,
		})
		return
	}

	c.JSON(http.StatusOK, measure)
}

// SearchMeasures handles GET /v1/measures/search - searches measures by criteria.
func (s *Server) SearchMeasures(c *gin.Context) {
	// Build filter from query parameters
	filter := models.MeasureFilter{
		SearchQuery: c.Query("q"),
		ActiveOnly:  c.Query("active") == "true",
	}

	// Parse programs filter (comma-separated)
	if programs := c.Query("programs"); programs != "" {
		filter.Programs = parseQualityPrograms(programs)
	}

	// Parse domains filter (comma-separated)
	if domains := c.Query("domains"); domains != "" {
		filter.Domains = parseClinicalDomains(domains)
	}

	// Parse types filter (comma-separated)
	if types := c.Query("types"); types != "" {
		filter.Types = parseMeasureTypes(types)
	}

	// Perform search
	measures := s.store.FilterMeasures(filter)

	c.JSON(http.StatusOK, ListMeasuresResponse{
		Measures: measures,
		Total:    len(measures),
	})
}

// GetMeasuresByProgram handles GET /v1/measures/by-program/:program.
func (s *Server) GetMeasuresByProgram(c *gin.Context) {
	programStr := c.Param("program")
	program := models.QualityProgram(programStr)

	// Validate program
	if !isValidProgram(program) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_program",
			"message": "Invalid quality program",
			"program": programStr,
			"valid":   []string{"HEDIS", "CMS", "MIPS", "ACO", "PCMH", "NQF", "CUSTOM"},
		})
		return
	}

	measures := s.store.GetMeasuresByProgram(program)

	c.JSON(http.StatusOK, ListMeasuresResponse{
		Measures: measures,
		Total:    len(measures),
	})
}

// GetMeasuresByDomain handles GET /v1/measures/by-domain/:domain.
func (s *Server) GetMeasuresByDomain(c *gin.Context) {
	domainStr := c.Param("domain")
	domain := models.ClinicalDomain(domainStr)

	// Validate domain
	if !isValidDomain(domain) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_domain",
			"message": "Invalid clinical domain",
			"domain":  domainStr,
			"valid":   []string{"DIABETES", "CARDIOVASCULAR", "RESPIRATORY", "PREVENTIVE", "BEHAVIORAL_HEALTH", "MATERNAL", "PEDIATRIC", "PATIENT_SAFETY"},
		})
		return
	}

	measures := s.store.GetMeasuresByDomain(domain)

	c.JSON(http.StatusOK, ListMeasuresResponse{
		Measures: measures,
		Total:    len(measures),
	})
}

// GetBenchmarks handles GET /v1/benchmarks/:measureId - returns all benchmarks for a measure.
func (s *Server) GetBenchmarks(c *gin.Context) {
	measureID := c.Param("measureId")
	if measureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Measure ID is required",
		})
		return
	}

	// Get the latest benchmark (for now - will return all when storage is implemented)
	benchmark := s.store.GetLatestBenchmark(measureID)
	if benchmark == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "not_found",
			"message":   "No benchmarks found for measure",
			"measureId": measureID,
		})
		return
	}

	// Return as array for consistency
	c.JSON(http.StatusOK, gin.H{
		"measureId":  measureID,
		"benchmarks": []*models.Benchmark{benchmark},
	})
}

// GetBenchmarkByYear handles GET /v1/benchmarks/:measureId/:year.
func (s *Server) GetBenchmarkByYear(c *gin.Context) {
	measureID := c.Param("measureId")
	yearStr := c.Param("year")

	if measureID == "" || yearStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Measure ID and year are required",
		})
		return
	}

	// Parse year
	year := parseYear(yearStr)
	if year == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_year",
			"message": "Year must be a valid 4-digit number",
			"year":    yearStr,
		})
		return
	}

	benchmark := s.store.GetBenchmark(measureID, year)
	if benchmark == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "not_found",
			"message":   "Benchmark not found for measure and year",
			"measureId": measureID,
			"year":      year,
		})
		return
	}

	c.JSON(http.StatusOK, benchmark)
}

// --- Helper functions ---

// parseQualityPrograms parses comma-separated quality programs.
func parseQualityPrograms(s string) []models.QualityProgram {
	var result []models.QualityProgram
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				p := models.QualityProgram(s[start:i])
				if isValidProgram(p) {
					result = append(result, p)
				}
			}
			start = i + 1
		}
	}
	return result
}

// parseClinicalDomains parses comma-separated clinical domains.
func parseClinicalDomains(s string) []models.ClinicalDomain {
	var result []models.ClinicalDomain
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				d := models.ClinicalDomain(s[start:i])
				if isValidDomain(d) {
					result = append(result, d)
				}
			}
			start = i + 1
		}
	}
	return result
}

// parseMeasureTypes parses comma-separated measure types.
func parseMeasureTypes(s string) []models.MeasureType {
	var result []models.MeasureType
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				t := models.MeasureType(s[start:i])
				if isValidMeasureType(t) {
					result = append(result, t)
				}
			}
			start = i + 1
		}
	}
	return result
}

// parseYear parses a year string to int.
func parseYear(s string) int {
	if len(s) != 4 {
		return 0
	}
	year := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		year = year*10 + int(c-'0')
	}
	return year
}

// isValidProgram checks if a quality program is valid.
func isValidProgram(p models.QualityProgram) bool {
	switch p {
	case models.ProgramHEDIS, models.ProgramCMS, models.ProgramMIPS,
		models.ProgramACO, models.ProgramPCMH, models.ProgramNQF, models.ProgramCustom:
		return true
	}
	return false
}

// isValidDomain checks if a clinical domain is valid.
func isValidDomain(d models.ClinicalDomain) bool {
	switch d {
	case models.DomainDiabetes, models.DomainCardiovascular, models.DomainRespiratory,
		models.DomainPreventive, models.DomainBehavioralHealth, models.DomainMaternal,
		models.DomainPediatric, models.DomainPatientSafety:
		return true
	}
	return false
}

// isValidMeasureType checks if a measure type is valid.
func isValidMeasureType(t models.MeasureType) bool {
	switch t {
	case models.MeasureTypeProcess, models.MeasureTypeOutcome, models.MeasureTypeStructure,
		models.MeasureTypeEfficiency, models.MeasureTypeComposite, models.MeasureTypeIntermediate:
		return true
	}
	return false
}
