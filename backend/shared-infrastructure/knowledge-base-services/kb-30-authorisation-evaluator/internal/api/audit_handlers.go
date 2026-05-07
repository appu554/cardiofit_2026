package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/audit"
)

// outputFormat returns the requested format ("json", "csv", "fhir") via
// ?format= query parameter; default is "fhir" (FHIR Bundle).
func outputFormat(r *http.Request) string {
	f := strings.ToLower(r.URL.Query().Get("format"))
	switch f {
	case "csv", "json", "fhir":
		return f
	}
	return "fhir"
}

func (s *Server) handleAuditResident(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, "/v1/audit/resident/"))
	if err != nil {
		http.Error(w, "invalid resident id", http.StatusBadRequest)
		return
	}
	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	records := s.Audit.QueryByResident(id, from, to)
	emitRecords(w, r, records)
}

func (s *Server) handleAuditCredential(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, "/v1/audit/credential/"))
	if err != nil {
		http.Error(w, "invalid credential id", http.StatusBadRequest)
		return
	}
	emitRecords(w, r, s.Audit.QueryByCredential(id))
}

func (s *Server) handleAuditJurisdiction(w http.ResponseWriter, r *http.Request) {
	// Path shape: /v1/audit/jurisdiction/{juri}/medications/{schedule}
	// jurisdiction may contain a slash (e.g. AU/VIC) — find the
	// /medications/ marker rather than splitting blindly.
	rest := strings.TrimPrefix(r.URL.Path, "/v1/audit/jurisdiction/")
	if rest == "" {
		http.Error(w, "missing jurisdiction", http.StatusBadRequest)
		return
	}
	juri := rest
	schedule := ""
	if idx := strings.Index(rest, "/medications/"); idx >= 0 {
		juri = rest[:idx]
		schedule = strings.TrimPrefix(rest[idx:], "/medications/")
		// Strip trailing slash if any.
		schedule = strings.TrimSuffix(schedule, "/")
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	records := s.Audit.QueryByJurisdictionSchedule(juri, schedule, time.Duration(days)*24*time.Hour)
	emitRecords(w, r, records)
}

func (s *Server) handleAuditChain(w http.ResponseWriter, r *http.Request) {
	// Path shape: /v1/audit/authorisation/{id}/chain
	rest := strings.TrimPrefix(r.URL.Path, "/v1/audit/authorisation/")
	parts := strings.Split(rest, "/")
	if len(parts) < 1 {
		http.Error(w, "missing authorisation id", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		http.Error(w, "invalid authorisation id", http.StatusBadRequest)
		return
	}
	chain, ok := s.Audit.QueryAuthorisationChain(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, chain)
}

func emitRecords(w http.ResponseWriter, r *http.Request, records []audit.EvaluationRecord) {
	switch outputFormat(r) {
	case "json":
		writeJSON(w, http.StatusOK, records)
	case "csv":
		emitCSV(w, records)
	default:
		emitFHIRBundle(w, records)
	}
}

func emitCSV(w http.ResponseWriter, records []audit.EvaluationRecord) {
	w.Header().Set("Content-Type", "text/csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"evaluation_id", "evaluated_at", "jurisdiction", "role", "action_class",
		"medication_schedule", "medication_class", "decision", "rule_id", "legislative_reference",
	})
	for _, rec := range records {
		_ = cw.Write([]string{
			rec.ID.String(),
			rec.EvaluatedAt.Format(time.RFC3339),
			rec.Query.Jurisdiction,
			rec.Query.Role,
			string(rec.Query.ActionClass),
			rec.Query.MedicationSchedule,
			rec.Query.MedicationClass,
			string(rec.Result.Decision),
			rec.Result.RuleID,
			rec.Result.LegislativeReference,
		})
	}
	cw.Flush()
}

// emitFHIRBundle returns a minimal FHIR R4 Bundle of type "searchset"
// where each entry is an AuditEvent resource modelling the authorisation
// evaluation. This is regulator-defensible and machine-parseable.
func emitFHIRBundle(w http.ResponseWriter, records []audit.EvaluationRecord) {
	type entry struct {
		FullURL  string         `json:"fullUrl"`
		Resource map[string]any `json:"resource"`
	}
	entries := make([]entry, 0, len(records))
	for _, rec := range records {
		entries = append(entries, entry{
			FullURL: fmt.Sprintf("urn:uuid:%s", rec.ID),
			Resource: map[string]any{
				"resourceType": "AuditEvent",
				"id":           rec.ID.String(),
				"recorded":     rec.EvaluatedAt.UTC().Format(time.RFC3339),
				"action":       string(rec.Result.Decision),
				"type": map[string]any{
					"system":  "http://terminology.hl7.org/CodeSystem/audit-event-type",
					"code":    "rest",
					"display": "RESTful Operation",
				},
				"agent": []map[string]any{{
					"who":  map[string]any{"identifier": map[string]any{"value": rec.Query.ActorRef.String()}},
					"role": []map[string]any{{"text": rec.Query.Role}},
				}},
				"source": map[string]any{
					"site":     "kb-30-authorisation-evaluator",
					"observer": map[string]any{"display": "AuthorisationEvaluator"},
				},
				"entity": []map[string]any{
					{
						"what": map[string]any{"display": rec.Result.RuleID},
						"name": "AuthorisationRule",
					},
					{
						"what": map[string]any{"identifier": map[string]any{"value": rec.Query.ResidentRef.String()}},
						"name": "Resident",
					},
				},
				"outcomeDesc": fmt.Sprintf("%s: %s (%s)", rec.Result.Decision, rec.Result.Reason, rec.Result.LegislativeReference),
			},
		})
	}
	bundle := map[string]any{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(entries),
		"entry":        entries,
	}
	w.Header().Set("Content-Type", "application/fhir+json")
	_ = json.NewEncoder(w).Encode(bundle)
}
