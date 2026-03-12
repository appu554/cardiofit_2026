// Package api provides GraphQL handlers for KB-9 Care Gaps Service.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-9-care-gaps/internal/models"
)

// GraphQLRequest represents a GraphQL query request.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   interface{}     `json:"data,omitempty"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []string               `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// handleGraphQLPlayground serves the GraphQL playground UI.
func (s *Server) handleGraphQLPlayground(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html>
<head>
  <title>KB-9 Care Gaps - GraphQL Playground</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root"></div>
  <script>
    window.addEventListener('load', function() {
      GraphQLPlayground.init(document.getElementById('root'), { endpoint: '/graphql' })
    })
  </script>
</body>
</html>
`
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// handleGraphQL handles GraphQL queries with a simple resolver.
func (s *Server) handleGraphQL(c *gin.Context) {
	// Parse request
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.sendGraphQLError(c, "Failed to read request body", err)
		return
	}

	var req GraphQLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.sendGraphQLError(c, "Invalid JSON", err)
		return
	}

	s.logger.Debug("GraphQL request",
		zap.String("operation", req.OperationName),
		zap.Int("query_length", len(req.Query)),
	)

	// Route to appropriate resolver based on query
	response := s.resolveGraphQL(c, &req)
	c.JSON(http.StatusOK, response)
}

// resolveGraphQL routes queries to the appropriate resolver.
func (s *Server) resolveGraphQL(c *gin.Context, req *GraphQLRequest) *GraphQLResponse {
	query := strings.TrimSpace(req.Query)

	// Handle Federation introspection
	if strings.Contains(query, "_service") {
		return s.resolveServiceSDL()
	}

	// Handle availableMeasures / measures query
	if strings.Contains(query, "availableMeasures") || (strings.Contains(query, "measures") && !strings.Contains(query, "Input")) {
		return s.resolveMeasures()
	}

	// Handle getMeasureInfo query
	if strings.Contains(query, "getMeasureInfo") {
		measureType := s.extractVariable(req.Variables, "measure")
		return s.resolveMeasureInfo(measureType)
	}

	// Handle getPatientCareGaps / careGaps query
	if strings.Contains(query, "getPatientCareGaps") || strings.Contains(query, "careGaps") {
		return s.resolveCareGaps(c, req.Variables)
	}

	// Handle evaluateMeasure query
	if strings.Contains(query, "evaluateMeasure") {
		return s.resolveEvaluateMeasure(c, req.Variables)
	}

	// Handle evaluatePopulation query
	if strings.Contains(query, "evaluatePopulation") {
		return s.resolveEvaluatePopulation(c, req.Variables)
	}

	// Handle careGapsServiceHealth query
	if strings.Contains(query, "careGapsServiceHealth") || strings.Contains(query, "health") {
		return s.resolveHealthCheck(c)
	}

	// Handle mutations
	if strings.Contains(query, "mutation") {
		if strings.Contains(query, "recordGapAddressed") || strings.Contains(query, "addressGap") {
			return s.resolveAddressGap(c, req.Variables)
		}
		if strings.Contains(query, "dismissGap") {
			return s.resolveDismissGap(c, req.Variables)
		}
		if strings.Contains(query, "snoozeGap") {
			return s.resolveSnoozeGap(c, req.Variables)
		}
	}

	// Unknown query
	return &GraphQLResponse{
		Errors: []GraphQLError{
			{Message: "Unknown query or mutation"},
		},
	}
}

// resolveServiceSDL returns the SDL for Apollo Federation.
func (s *Server) resolveServiceSDL() *GraphQLResponse {
	sdl := `
extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable"])

type Query {
  getPatientCareGaps(input: PatientCareGapsInput!): CareGapReport!
  evaluateMeasure(input: MeasureEvaluationInput!): MeasureReport!
  availableMeasures: [MeasureInfo!]!
  careGapsServiceHealth: ServiceHealth!
}
`
	return &GraphQLResponse{
		Data: map[string]interface{}{
			"_service": map[string]string{
				"sdl": sdl,
			},
		},
	}
}

// resolveMeasures returns all available measures.
func (s *Server) resolveMeasures() *GraphQLResponse {
	measures := s.careGapsService.GetAvailableMeasures()
	return &GraphQLResponse{
		Data: map[string]interface{}{
			"availableMeasures": convertMeasuresToGraphQL(measures),
		},
	}
}

// resolveMeasureInfo returns info for a specific measure.
func (s *Server) resolveMeasureInfo(measureType string) *GraphQLResponse {
	if measureType == "" {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: "measure type is required"}},
		}
	}

	measure, err := s.careGapsService.GetMeasureInfo(models.MeasureType(measureType))
	if err != nil {
		return &GraphQLResponse{
			Data: map[string]interface{}{
				"getMeasureInfo": nil,
			},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"getMeasureInfo": convertMeasureToGraphQL(*measure),
		},
	}
}

// resolveCareGaps handles care gaps queries.
func (s *Server) resolveCareGaps(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	// Extract input from variables
	input := s.extractInputObject(vars, "input")
	if input == nil {
		// Try direct variables
		input = vars
	}

	patientID := s.extractString(input, "patientId")
	if patientID == "" {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: "patientId is required"}},
		}
	}

	// Parse optional parameters
	includeClosedGaps := s.extractBool(input, "includeClosedGaps")
	includeEvidence := s.extractBool(input, "includeEvidence")

	// Parse period
	period := s.extractPeriod(input)

	// Parse measures
	var measures []models.MeasureType
	if measuresRaw, ok := input["measures"].([]interface{}); ok {
		for _, m := range measuresRaw {
			if ms, ok := m.(string); ok {
				measures = append(measures, models.MeasureType(ms))
			}
		}
	}

	// Get care gaps
	report, err := s.careGapsService.GetPatientCareGaps(
		c.Request.Context(),
		patientID,
		measures,
		period,
		includeClosedGaps,
		includeEvidence,
	)
	if err != nil {
		s.logger.Error("Failed to get care gaps", zap.Error(err))
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: fmt.Sprintf("Failed to get care gaps: %v", err)}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"getPatientCareGaps": convertCareGapReportToGraphQL(report),
		},
	}
}

// resolveEvaluateMeasure handles measure evaluation queries.
func (s *Server) resolveEvaluateMeasure(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	input := s.extractInputObject(vars, "input")
	if input == nil {
		input = vars
	}

	patientID := s.extractString(input, "patientId")
	measureType := s.extractString(input, "measure")
	periodStart := s.extractString(input, "periodStart")
	periodEnd := s.extractString(input, "periodEnd")

	if patientID == "" || measureType == "" {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: "patientId and measure are required"}},
		}
	}

	// Parse period
	start, _ := time.Parse("2006-01-02", periodStart)
	end, _ := time.Parse("2006-01-02", periodEnd)
	if start.IsZero() {
		start = time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}

	report, err := s.careGapsService.EvaluateMeasure(
		c.Request.Context(),
		patientID,
		models.MeasureType(measureType),
		models.Period{Start: start, End: end},
	)
	if err != nil {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: fmt.Sprintf("Failed to evaluate measure: %v", err)}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"evaluateMeasure": convertMeasureReportToGraphQL(report),
		},
	}
}

// resolveEvaluatePopulation handles population evaluation queries.
func (s *Server) resolveEvaluatePopulation(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	input := s.extractInputObject(vars, "input")
	if input == nil {
		input = vars
	}

	measureType := s.extractString(input, "measure")
	periodStart := s.extractString(input, "periodStart")
	periodEnd := s.extractString(input, "periodEnd")
	limit := 1000

	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	// Extract patient IDs
	var patientIDs []string
	if pids, ok := input["patientIds"].([]interface{}); ok {
		for _, pid := range pids {
			if ps, ok := pid.(string); ok {
				patientIDs = append(patientIDs, ps)
			}
		}
	}

	if len(patientIDs) == 0 || measureType == "" {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: "patientIds and measure are required"}},
		}
	}

	// Parse period
	start, _ := time.Parse("2006-01-02", periodStart)
	end, _ := time.Parse("2006-01-02", periodEnd)

	report, err := s.careGapsService.EvaluatePopulation(
		c.Request.Context(),
		patientIDs,
		models.MeasureType(measureType),
		models.Period{Start: start, End: end},
		limit,
	)
	if err != nil {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: fmt.Sprintf("Failed to evaluate population: %v", err)}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"evaluatePopulation": convertPopulationReportToGraphQL(report),
		},
	}
}

// resolveHealthCheck returns service health.
func (s *Server) resolveHealthCheck(c *gin.Context) *GraphQLResponse {
	status := "healthy"
	fhirStatus := "connected"
	cqlStatus := "available"

	if err := s.careGapsService.HealthCheck(c.Request.Context()); err != nil {
		status = "degraded"
		fhirStatus = "error: " + err.Error()
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"careGapsServiceHealth": map[string]interface{}{
				"status":               status,
				"cqlEngineStatus":      cqlStatus,
				"fhirServerStatus":     fhirStatus,
				"measureLibraryStatus": "loaded",
				"version":              "1.0.0",
				"uptimeSeconds":        int(time.Since(s.startTime).Seconds()),
				"measuresLoaded":       len(s.careGapsService.GetAvailableMeasures()),
			},
		},
	}
}

// resolveAddressGap handles gap addressed mutations.
func (s *Server) resolveAddressGap(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	input := s.extractInputObject(vars, "input")
	if input == nil {
		input = vars
	}

	patientID := s.extractString(input, "patientId")
	gapID := s.extractString(input, "gapId")
	intervention := s.extractString(input, "intervention")
	notes := s.extractString(input, "notes")

	gap, err := s.careGapsService.RecordGapAddressed(
		c.Request.Context(),
		patientID,
		gapID,
		models.InterventionType(intervention),
		notes,
	)
	if err != nil {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: err.Error()}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"recordGapAddressed": convertCareGapToGraphQL(*gap),
		},
	}
}

// resolveDismissGap handles gap dismissal mutations.
func (s *Server) resolveDismissGap(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	input := s.extractInputObject(vars, "input")
	if input == nil {
		input = vars
	}

	patientID := s.extractString(input, "patientId")
	gapID := s.extractString(input, "gapId")
	reason := s.extractString(input, "reason")

	gap, err := s.careGapsService.DismissGap(
		c.Request.Context(),
		patientID,
		gapID,
		reason,
	)
	if err != nil {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: err.Error()}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"dismissGap": convertCareGapToGraphQL(*gap),
		},
	}
}

// resolveSnoozeGap handles gap snooze mutations.
func (s *Server) resolveSnoozeGap(c *gin.Context, vars map[string]interface{}) *GraphQLResponse {
	input := s.extractInputObject(vars, "input")
	if input == nil {
		input = vars
	}

	patientID := s.extractString(input, "patientId")
	gapID := s.extractString(input, "gapId")
	snoozeUntilStr := s.extractString(input, "snoozeUntil")
	reason := s.extractString(input, "reason")

	snoozeUntil, _ := time.Parse("2006-01-02", snoozeUntilStr)

	gap, err := s.careGapsService.SnoozeGap(
		c.Request.Context(),
		patientID,
		gapID,
		snoozeUntil,
		reason,
	)
	if err != nil {
		return &GraphQLResponse{
			Errors: []GraphQLError{{Message: err.Error()}},
		}
	}

	return &GraphQLResponse{
		Data: map[string]interface{}{
			"snoozeGap": convertCareGapToGraphQL(*gap),
		},
	}
}

// ========== Helper Functions ==========

func (s *Server) sendGraphQLError(c *gin.Context, message string, err error) {
	c.JSON(http.StatusBadRequest, GraphQLResponse{
		Errors: []GraphQLError{{Message: fmt.Sprintf("%s: %v", message, err)}},
	})
}

func (s *Server) extractVariable(vars map[string]interface{}, key string) string {
	if v, ok := vars[key].(string); ok {
		return v
	}
	return ""
}

func (s *Server) extractInputObject(vars map[string]interface{}, key string) map[string]interface{} {
	if v, ok := vars[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func (s *Server) extractString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func (s *Server) extractBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func (s *Server) extractPeriod(m map[string]interface{}) models.Period {
	now := time.Now()
	year := now.Year()

	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := now

	if ps := s.extractString(m, "periodStart"); ps != "" {
		if parsed, err := time.Parse("2006-01-02", ps); err == nil {
			start = parsed
		}
	}
	if pe := s.extractString(m, "periodEnd"); pe != "" {
		if parsed, err := time.Parse("2006-01-02", pe); err == nil {
			end = parsed
		}
	}

	return models.Period{Start: start, End: end}
}

// ========== Conversion Functions ==========

func convertMeasuresToGraphQL(measures []models.MeasureInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, len(measures))
	for i, m := range measures {
		result[i] = convertMeasureToGraphQL(m)
	}
	return result
}

func convertMeasureToGraphQL(m models.MeasureInfo) map[string]interface{} {
	return map[string]interface{}{
		"type":        m.Type,
		"cmsId":       m.CMSID,
		"name":        m.Name,
		"description": m.Description,
		"domain":      m.Domain,
		"steward":     m.Steward,
		"version":     m.Version,
		"cqlLibrary":  m.CQLLibrary,
	}
}

func convertCareGapReportToGraphQL(r *models.CareGapReport) map[string]interface{} {
	openGaps := make([]map[string]interface{}, len(r.OpenGaps))
	for i, g := range r.OpenGaps {
		openGaps[i] = convertCareGapToGraphQL(g)
	}

	closedGaps := make([]map[string]interface{}, len(r.ClosedGaps))
	for i, g := range r.ClosedGaps {
		closedGaps[i] = convertCareGapToGraphQL(g)
	}

	upcomingDue := make([]map[string]interface{}, len(r.UpcomingDue))
	for i, g := range r.UpcomingDue {
		upcomingDue[i] = convertCareGapToGraphQL(g)
	}

	return map[string]interface{}{
		"patientId":  r.PatientID,
		"reportDate": r.ReportDate.Format(time.RFC3339),
		"measurementPeriod": map[string]string{
			"start": r.MeasurementPeriod.Start.Format("2006-01-02"),
			"end":   r.MeasurementPeriod.End.Format("2006-01-02"),
		},
		"openGaps":         openGaps,
		"closedGaps":       closedGaps,
		"upcomingDue":      upcomingDue,
		"summary":          convertSummaryToGraphQL(r.Summary),
		"dataCompleteness": r.DataCompleteness,
		"warnings":         r.Warnings,
	}
}

func convertCareGapToGraphQL(g models.CareGap) map[string]interface{} {
	result := map[string]interface{}{
		"id":             g.ID,
		"measure":        convertMeasureToGraphQL(g.Measure),
		"status":         g.Status,
		"priority":       g.Priority,
		"reason":         g.Reason,
		"recommendation": g.Recommendation,
		"identifiedDate": g.IdentifiedDate.Format("2006-01-02"),
	}

	if g.ClosedDate != nil {
		result["closedDate"] = g.ClosedDate.Format("2006-01-02")
	}
	if g.DueDate != nil {
		result["dueDate"] = g.DueDate.Format("2006-01-02")
	}
	if g.DaysUntilDue != nil {
		result["daysUntilDue"] = *g.DaysUntilDue
	}
	if g.Evidence != nil {
		result["evidence"] = convertEvidenceToGraphQL(g.Evidence)
	}
	if g.TemporalContext != nil {
		result["temporalContext"] = convertTemporalInfoToGraphQL(g.TemporalContext)
	}
	if len(g.Interventions) > 0 {
		interventions := make([]map[string]interface{}, len(g.Interventions))
		for i, intv := range g.Interventions {
			interventions[i] = map[string]interface{}{
				"type":        intv.Type,
				"description": intv.Description,
				"code":        intv.Code,
				"codeSystem":  intv.CodeSystem,
				"priority":    intv.Priority,
			}
		}
		result["interventions"] = interventions
	}

	return result
}

func convertSummaryToGraphQL(s models.CareGapSummary) map[string]interface{} {
	gapsByDomain := make([]map[string]interface{}, len(s.GapsByDomain))
	for i, d := range s.GapsByDomain {
		gapsByDomain[i] = map[string]interface{}{
			"domain": d.Domain,
			"count":  d.Count,
		}
	}

	result := map[string]interface{}{
		"totalOpenGaps":    s.TotalOpenGaps,
		"urgentGaps":       s.UrgentGaps,
		"highPriorityGaps": s.HighPriorityGaps,
		"gapsByDomain":     gapsByDomain,
	}

	if s.QualityScore != nil {
		result["qualityScore"] = *s.QualityScore
	}

	return result
}

func convertEvidenceToGraphQL(e *models.CQLEvidence) map[string]interface{} {
	populations := make([]map[string]interface{}, len(e.Populations))
	for i, p := range e.Populations {
		populations[i] = map[string]interface{}{
			"population": p.Population,
			"isMember":   p.IsMember,
			"reason":     p.Reason,
		}
	}

	dataElements := make([]map[string]interface{}, len(e.DataElements))
	for i, d := range e.DataElements {
		elem := map[string]interface{}{
			"name":             d.Name,
			"contributedToGap": d.ContributedToGap,
		}
		if d.Value != nil {
			elem["value"] = *d.Value
		}
		if d.ValueDate != nil {
			elem["valueDate"] = d.ValueDate.Format("2006-01-02")
		}
		dataElements[i] = elem
	}

	return map[string]interface{}{
		"libraryId":      e.LibraryID,
		"libraryVersion": e.LibraryVersion,
		"populations":    populations,
		"dataElements":   dataElements,
		"evaluatedAt":    e.EvaluatedAt.Format(time.RFC3339),
	}
}

func convertTemporalInfoToGraphQL(t *models.TemporalInfo) map[string]interface{} {
	result := map[string]interface{}{
		"status":           t.Status,
		"gracePeriodDays":  t.GracePeriodDays,
		"daysUntilDue":     t.DaysUntilDue,
		"daysOverdue":      t.DaysOverdue,
		"isRecurring":      t.IsRecurring,
		"recurrenceMonths": t.RecurrenceMonths,
		"sourcedFromKB3":   t.SourcedFromKB3,
	}

	if t.DueDate != nil {
		result["dueDate"] = t.DueDate.Format("2006-01-02")
	}
	if t.OverdueDate != nil {
		result["overdueDate"] = t.OverdueDate.Format("2006-01-02")
	}
	if t.LastCompletedDate != nil {
		result["lastCompletedDate"] = t.LastCompletedDate.Format("2006-01-02")
	}
	if t.NextDueDate != nil {
		result["nextDueDate"] = t.NextDueDate.Format("2006-01-02")
	}

	return result
}

func convertMeasureReportToGraphQL(r *models.MeasureReport) map[string]interface{} {
	populations := make([]map[string]interface{}, len(r.Populations))
	for i, p := range r.Populations {
		populations[i] = map[string]interface{}{
			"population": p.Population,
			"count":      p.Count,
		}
	}

	return map[string]interface{}{
		"id":          r.ID,
		"measure":     convertMeasureToGraphQL(r.Measure),
		"patientId":   r.PatientID,
		"period": map[string]string{
			"start": r.Period.Start.Format("2006-01-02"),
			"end":   r.Period.End.Format("2006-01-02"),
		},
		"status":      r.Status,
		"type":        r.Type,
		"populations": populations,
		"generatedAt": r.GeneratedAt.Format(time.RFC3339),
	}
}

func convertPopulationReportToGraphQL(r *models.PopulationMeasureReport) map[string]interface{} {
	populations := make([]map[string]interface{}, len(r.Populations))
	for i, p := range r.Populations {
		populations[i] = map[string]interface{}{
			"population": p.Population,
			"count":      p.Count,
		}
	}

	patientsWithGaps := make([]map[string]interface{}, len(r.PatientsWithGaps))
	for i, p := range r.PatientsWithGaps {
		patientsWithGaps[i] = map[string]interface{}{
			"patientId":      p.PatientID,
			"status":         p.Status,
			"recommendation": p.Recommendation,
		}
	}

	result := map[string]interface{}{
		"id":            r.ID,
		"measure":       convertMeasureToGraphQL(r.Measure),
		"period": map[string]string{
			"start": r.Period.Start.Format("2006-01-02"),
			"end":   r.Period.End.Format("2006-01-02"),
		},
		"totalPatients":    r.TotalPatients,
		"populations":      populations,
		"patientsWithGaps": patientsWithGaps,
		"generatedAt":      r.GeneratedAt.Format(time.RFC3339),
		"processingTimeMs": r.ProcessingTimeMs,
	}

	if r.PerformanceRate != nil {
		result["performanceRate"] = *r.PerformanceRate
	}

	return result
}
