package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/google/uuid"
	"database/sql"
	_ "github.com/lib/pq"
)

// AuditEvent represents an audit event
type AuditEvent struct {
	ID                  string                 `json:"id"`
	EventID             string                 `json:"event_id"`
	EventType           string                 `json:"event_type"`
	EventCategory       string                 `json:"event_category"`
	SeverityLevel       string                 `json:"severity_level"`
	EventTimestamp      time.Time              `json:"event_timestamp"`
	EventDescription    string                 `json:"event_description"`
	EventSource         string                 `json:"event_source"`
	EventOutcome        string                 `json:"event_outcome"`
	PrimaryActorID      string                 `json:"primary_actor_id"`
	PrimaryActorType    string                 `json:"primary_actor_type"`
	SecondaryActors     map[string]interface{} `json:"secondary_actors,omitempty"`
	ResourceType        string                 `json:"resource_type"`
	ResourceID          *string                `json:"resource_id,omitempty"`
	ResourceName        *string                `json:"resource_name,omitempty"`
	BeforeState         map[string]interface{} `json:"before_state,omitempty"`
	AfterState          map[string]interface{} `json:"after_state,omitempty"`
	ChangeSummary       *string                `json:"change_summary,omitempty"`
	ClinicalDomain      *string                `json:"clinical_domain,omitempty"`
	PatientSafetyFlag   bool                   `json:"patient_safety_flag"`
	ClinicalRiskLevel   *string                `json:"clinical_risk_level,omitempty"`
	ComplianceFlags     map[string]interface{} `json:"compliance_flags,omitempty"`
	ProvenanceChain     map[string]interface{} `json:"provenance_chain,omitempty"`
	CorrelationID       *string                `json:"correlation_id,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy           string                 `json:"created_by"`
}

// AuditSession represents an audit session
type AuditSession struct {
	ID                        string                 `json:"id"`
	SessionID                 string                 `json:"session_id"`
	SessionType               string                 `json:"session_type"`
	SessionStatus             string                 `json:"session_status"`
	InitiatedBy               string                 `json:"initiated_by"`
	InitiatedAt               time.Time              `json:"initiated_at"`
	ClinicalContext           map[string]interface{} `json:"clinical_context,omitempty"`
	SafetyAssessment          map[string]interface{} `json:"safety_assessment,omitempty"`
	RiskScore                 *int                   `json:"risk_score,omitempty"`
	RequiresClinicalReview    bool                   `json:"requires_clinical_review"`
	ClinicalReviewer          *string                `json:"clinical_reviewer,omitempty"`
	ClinicalReviewStatus      *string                `json:"clinical_review_status,omitempty"`
	ClinicalReviewNotes       *string                `json:"clinical_review_notes,omitempty"`
	TechnicalReviewer         *string                `json:"technical_reviewer,omitempty"`
	TechnicalReviewStatus     *string                `json:"technical_review_status,omitempty"`
	TechnicalReviewNotes      *string                `json:"technical_review_notes,omitempty"`
	SessionMetadata           map[string]interface{} `json:"session_metadata,omitempty"`
}

var (
	databaseURL string
	db          *sql.DB
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "audit",
		Short: "KB-7 Clinical Audit System",
		Long:  "Command-line tool for managing clinical audit events and sessions",
	}

	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", "", "Database connection URL")

	// Commands
	rootCmd.AddCommand(createPRAuditCmd())
	rootCmd.AddCommand(createEventCmd())
	rootCmd.AddCommand(createSessionCmd())
	rootCmd.AddCommand(listEventsCmd())
	rootCmd.AddCommand(listSessionsCmd())
	rootCmd.AddCommand(generateReportCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func createPRAuditCmd() *cobra.Command {
	var (
		prNumber        string
		author          string
		branch          string
		safetyScore     string
		affectedSystems string
		requiresReview  string
	)

	cmd := &cobra.Command{
		Use:   "create-pr-audit",
		Short: "Create audit entry for GitHub PR",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			// Parse parameters
			score, err := strconv.Atoi(safetyScore)
			if err != nil {
				score = 0
			}

			requiresReviewBool, _ := strconv.ParseBool(requiresReview)

			// Create audit session
			sessionID := fmt.Sprintf("pr-%s", prNumber)
			session := &AuditSession{
				ID:                     uuid.New().String(),
				SessionID:              sessionID,
				SessionType:            "pr_review",
				SessionStatus:          "active",
				InitiatedBy:            author,
				InitiatedAt:            time.Now(),
				RiskScore:              &score,
				RequiresClinicalReview: requiresReviewBool,
				ClinicalContext: map[string]interface{}{
					"pr_number":         prNumber,
					"branch":            branch,
					"affected_systems":  affectedSystems,
				},
				SafetyAssessment: map[string]interface{}{
					"safety_score":      score,
					"assessment_time":   time.Now(),
					"automated":         true,
				},
				SessionMetadata: map[string]interface{}{
					"source":           "github_workflow",
					"automation_level": "full",
				},
			}

			if err := createSession(session); err != nil {
				return fmt.Errorf("failed to create audit session: %w", err)
			}

			// Create audit event
			event := &AuditEvent{
				ID:                uuid.New().String(),
				EventID:           fmt.Sprintf("pr-audit-%s", prNumber),
				EventType:         "pr_safety_assessment",
				EventCategory:     "clinical",
				SeverityLevel:     getSeverityFromScore(score),
				EventTimestamp:    time.Now(),
				EventDescription:  fmt.Sprintf("Clinical safety assessment for PR #%s", prNumber),
				EventSource:       "github",
				EventOutcome:      "success",
				PrimaryActorID:    author,
				PrimaryActorType:  "user",
				ResourceType:      "pull_request",
				ResourceID:        &prNumber,
				ResourceName:      &branch,
				PatientSafetyFlag: score > 50,
				ClinicalRiskLevel: getClinicalRiskLevel(score),
				CreatedBy:         "audit-system",
				Metadata: map[string]interface{}{
					"pr_number":         prNumber,
					"branch":            branch,
					"safety_score":      score,
					"affected_systems":  affectedSystems,
					"requires_review":   requiresReviewBool,
				},
			}

			if err := createEvent(event); err != nil {
				return fmt.Errorf("failed to create audit event: %w", err)
			}

			// Link event to session
			if err := linkEventToSession(session.ID, event.ID, 1); err != nil {
				return fmt.Errorf("failed to link event to session: %w", err)
			}

			fmt.Printf("Created audit entry for PR #%s (Session: %s, Event: %s)\n", prNumber, session.SessionID, event.EventID)
			return nil
		},
	}

	cmd.Flags().StringVar(&prNumber, "pr-number", "", "Pull request number")
	cmd.Flags().StringVar(&author, "author", "", "PR author")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name")
	cmd.Flags().StringVar(&safetyScore, "safety-score", "0", "Safety score (0-100)")
	cmd.Flags().StringVar(&affectedSystems, "affected-systems", "", "Affected systems")
	cmd.Flags().StringVar(&requiresReview, "requires-review", "false", "Requires clinical review")

	cmd.MarkFlagRequired("pr-number")
	cmd.MarkFlagRequired("author")
	cmd.MarkFlagRequired("branch")

	return cmd
}

func createEventCmd() *cobra.Command {
	var (
		eventType        string
		eventCategory    string
		description      string
		actorID          string
		resourceType     string
		resourceID       string
		clinicalDomain   string
		safetyFlag       bool
	)

	cmd := &cobra.Command{
		Use:   "create-event",
		Short: "Create a new audit event",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			event := &AuditEvent{
				ID:                uuid.New().String(),
				EventID:           uuid.New().String(),
				EventType:         eventType,
				EventCategory:     eventCategory,
				SeverityLevel:     "medium",
				EventTimestamp:    time.Now(),
				EventDescription:  description,
				EventSource:       "manual",
				EventOutcome:      "success",
				PrimaryActorID:    actorID,
				PrimaryActorType:  "user",
				ResourceType:      resourceType,
				PatientSafetyFlag: safetyFlag,
				CreatedBy:         "manual-entry",
			}

			if resourceID != "" {
				event.ResourceID = &resourceID
			}

			if clinicalDomain != "" {
				event.ClinicalDomain = &clinicalDomain
			}

			if err := createEvent(event); err != nil {
				return fmt.Errorf("failed to create audit event: %w", err)
			}

			fmt.Printf("Created audit event: %s\n", event.EventID)
			return nil
		},
	}

	cmd.Flags().StringVar(&eventType, "type", "", "Event type")
	cmd.Flags().StringVar(&eventCategory, "category", "technical", "Event category")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&actorID, "actor", "", "Actor ID")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Resource type")
	cmd.Flags().StringVar(&resourceID, "resource-id", "", "Resource ID")
	cmd.Flags().StringVar(&clinicalDomain, "clinical-domain", "", "Clinical domain")
	cmd.Flags().BoolVar(&safetyFlag, "safety-flag", false, "Patient safety flag")

	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("description")
	cmd.MarkFlagRequired("actor")
	cmd.MarkFlagRequired("resource-type")

	return cmd
}

func createSessionCmd() *cobra.Command {
	var (
		sessionType string
		initiatedBy string
	)

	cmd := &cobra.Command{
		Use:   "create-session",
		Short: "Create a new audit session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			session := &AuditSession{
				ID:            uuid.New().String(),
				SessionID:     uuid.New().String(),
				SessionType:   sessionType,
				SessionStatus: "active",
				InitiatedBy:   initiatedBy,
				InitiatedAt:   time.Now(),
			}

			if err := createSession(session); err != nil {
				return fmt.Errorf("failed to create audit session: %w", err)
			}

			fmt.Printf("Created audit session: %s\n", session.SessionID)
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionType, "type", "", "Session type")
	cmd.Flags().StringVar(&initiatedBy, "initiated-by", "", "Initiated by")

	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("initiated-by")

	return cmd
}

func listEventsCmd() *cobra.Command {
	var (
		limit          int
		eventCategory  string
		clinicalDomain string
		safetyOnly     bool
	)

	cmd := &cobra.Command{
		Use:   "list-events",
		Short: "List audit events",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			events, err := listEvents(limit, eventCategory, clinicalDomain, safetyOnly)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}

			for _, event := range events {
				fmt.Printf("%s | %s | %s | %s | %s | %s\n",
					event.EventTimestamp.Format("2006-01-02 15:04:05"),
					event.EventID,
					event.EventType,
					event.EventCategory,
					event.PrimaryActorID,
					event.EventDescription,
				)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Limit number of results")
	cmd.Flags().StringVar(&eventCategory, "category", "", "Filter by event category")
	cmd.Flags().StringVar(&clinicalDomain, "clinical-domain", "", "Filter by clinical domain")
	cmd.Flags().BoolVar(&safetyOnly, "safety-only", false, "Show only patient safety events")

	return cmd
}

func listSessionsCmd() *cobra.Command {
	var (
		limit  int
		status string
	)

	cmd := &cobra.Command{
		Use:   "list-sessions",
		Short: "List audit sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			sessions, err := listSessions(limit, status)
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			for _, session := range sessions {
				reviewStatus := "No"
				if session.RequiresClinicalReview {
					reviewStatus = "Yes"
				}

				fmt.Printf("%s | %s | %s | %s | %s | Risk: %v\n",
					session.InitiatedAt.Format("2006-01-02 15:04:05"),
					session.SessionID,
					session.SessionType,
					session.SessionStatus,
					reviewStatus,
					session.RiskScore,
				)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Limit number of results")
	cmd.Flags().StringVar(&status, "status", "", "Filter by session status")

	return cmd
}

func generateReportCmd() *cobra.Command {
	var (
		outputFile string
		format     string
		days       int
	)

	cmd := &cobra.Command{
		Use:   "generate-report",
		Short: "Generate audit report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			report, err := generateAuditReport(days)
			if err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			var output []byte
			switch format {
			case "json":
				output, err = json.MarshalIndent(report, "", "  ")
			default:
				output = []byte(formatReportAsText(report))
			}

			if err != nil {
				return fmt.Errorf("failed to format report: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, output, 0644); err != nil {
					return fmt.Errorf("failed to write report file: %w", err)
				}
				fmt.Printf("Report written to %s\n", outputFile)
			} else {
				fmt.Print(string(output))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json)")
	cmd.Flags().IntVar(&days, "days", 7, "Number of days to include in report")

	return cmd
}

// Database functions
func initDB() error {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			return fmt.Errorf("database URL not provided")
		}
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	return db.Ping()
}

func createEvent(event *AuditEvent) error {
	query := `
		INSERT INTO audit_events (
			id, event_id, event_type, event_category, severity_level,
			event_timestamp, event_description, event_source, event_outcome,
			primary_actor_id, primary_actor_type, secondary_actors,
			resource_type, resource_id, resource_name,
			before_state, after_state, change_summary,
			clinical_domain, patient_safety_flag, clinical_risk_level,
			compliance_flags, provenance_chain, correlation_id,
			metadata, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)`

	secondaryActors, _ := json.Marshal(event.SecondaryActors)
	beforeState, _ := json.Marshal(event.BeforeState)
	afterState, _ := json.Marshal(event.AfterState)
	complianceFlags, _ := json.Marshal(event.ComplianceFlags)
	provenanceChain, _ := json.Marshal(event.ProvenanceChain)
	metadata, _ := json.Marshal(event.Metadata)

	_, err := db.Exec(query,
		event.ID, event.EventID, event.EventType, event.EventCategory, event.SeverityLevel,
		event.EventTimestamp, event.EventDescription, event.EventSource, event.EventOutcome,
		event.PrimaryActorID, event.PrimaryActorType, secondaryActors,
		event.ResourceType, event.ResourceID, event.ResourceName,
		beforeState, afterState, event.ChangeSummary,
		event.ClinicalDomain, event.PatientSafetyFlag, event.ClinicalRiskLevel,
		complianceFlags, provenanceChain, event.CorrelationID,
		metadata, event.CreatedBy,
	)

	return err
}

func createSession(session *AuditSession) error {
	query := `
		INSERT INTO audit_sessions (
			id, session_id, session_type, session_status,
			initiated_by, initiated_at, clinical_context,
			safety_assessment, risk_score, requires_clinical_review,
			clinical_reviewer, clinical_review_status, clinical_review_notes,
			technical_reviewer, technical_review_status, technical_review_notes,
			session_metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	clinicalContext, _ := json.Marshal(session.ClinicalContext)
	safetyAssessment, _ := json.Marshal(session.SafetyAssessment)
	sessionMetadata, _ := json.Marshal(session.SessionMetadata)

	_, err := db.Exec(query,
		session.ID, session.SessionID, session.SessionType, session.SessionStatus,
		session.InitiatedBy, session.InitiatedAt, clinicalContext,
		safetyAssessment, session.RiskScore, session.RequiresClinicalReview,
		session.ClinicalReviewer, session.ClinicalReviewStatus, session.ClinicalReviewNotes,
		session.TechnicalReviewer, session.TechnicalReviewStatus, session.TechnicalReviewNotes,
		sessionMetadata,
	)

	return err
}

func linkEventToSession(sessionID, eventID string, sequenceNumber int) error {
	query := `
		INSERT INTO audit_session_events (session_id, event_id, sequence_number)
		SELECT s.id, e.id, $3
		FROM audit_sessions s, audit_events e
		WHERE s.id = $1 AND e.id = $2`

	_, err := db.Exec(query, sessionID, eventID, sequenceNumber)
	return err
}

func listEvents(limit int, eventCategory, clinicalDomain string, safetyOnly bool) ([]*AuditEvent, error) {
	query := `
		SELECT id, event_id, event_type, event_category, severity_level,
			   event_timestamp, event_description, event_source, event_outcome,
			   primary_actor_id, primary_actor_type, resource_type,
			   resource_id, resource_name, clinical_domain,
			   patient_safety_flag, clinical_risk_level, created_by
		FROM audit_events
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if eventCategory != "" {
		argCount++
		query += fmt.Sprintf(" AND event_category = $%d", argCount)
		args = append(args, eventCategory)
	}

	if clinicalDomain != "" {
		argCount++
		query += fmt.Sprintf(" AND clinical_domain = $%d", argCount)
		args = append(args, clinicalDomain)
	}

	if safetyOnly {
		query += " AND patient_safety_flag = true"
	}

	query += " ORDER BY event_timestamp DESC"

	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		err := rows.Scan(
			&event.ID, &event.EventID, &event.EventType, &event.EventCategory, &event.SeverityLevel,
			&event.EventTimestamp, &event.EventDescription, &event.EventSource, &event.EventOutcome,
			&event.PrimaryActorID, &event.PrimaryActorType, &event.ResourceType,
			&event.ResourceID, &event.ResourceName, &event.ClinicalDomain,
			&event.PatientSafetyFlag, &event.ClinicalRiskLevel, &event.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func listSessions(limit int, status string) ([]*AuditSession, error) {
	query := `
		SELECT id, session_id, session_type, session_status,
			   initiated_by, initiated_at, risk_score,
			   requires_clinical_review, clinical_review_status,
			   technical_review_status
		FROM audit_sessions
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND session_status = $%d", argCount)
		args = append(args, status)
	}

	query += " ORDER BY initiated_at DESC"

	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*AuditSession
	for rows.Next() {
		session := &AuditSession{}
		err := rows.Scan(
			&session.ID, &session.SessionID, &session.SessionType, &session.SessionStatus,
			&session.InitiatedBy, &session.InitiatedAt, &session.RiskScore,
			&session.RequiresClinicalReview, &session.ClinicalReviewStatus,
			&session.TechnicalReviewStatus,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Helper functions
func getSeverityFromScore(score int) string {
	switch {
	case score >= 90:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

func getClinicalRiskLevel(score int) *string {
	var level string
	switch {
	case score >= 90:
		level = "critical"
	case score >= 70:
		level = "high"
	case score >= 40:
		level = "moderate"
	case score >= 20:
		level = "low"
	default:
		level = "minimal"
	}
	return &level
}

// Report generation
type AuditReport struct {
	Period        string                   `json:"period"`
	GeneratedAt   time.Time                `json:"generated_at"`
	EventSummary  map[string]int           `json:"event_summary"`
	SessionSummary map[string]int          `json:"session_summary"`
	SafetyMetrics map[string]interface{}   `json:"safety_metrics"`
	TopActors     []map[string]interface{} `json:"top_actors"`
}

func generateAuditReport(days int) (*AuditReport, error) {
	report := &AuditReport{
		Period:      fmt.Sprintf("Last %d days", days),
		GeneratedAt: time.Now(),
	}

	// Event summary
	eventQuery := `
		SELECT event_category, COUNT(*)
		FROM audit_events
		WHERE event_timestamp >= NOW() - INTERVAL '%d days'
		GROUP BY event_category`

	rows, err := db.Query(fmt.Sprintf(eventQuery, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report.EventSummary = make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err == nil {
			report.EventSummary[category] = count
		}
	}

	// Session summary
	sessionQuery := `
		SELECT session_status, COUNT(*)
		FROM audit_sessions
		WHERE initiated_at >= NOW() - INTERVAL '%d days'
		GROUP BY session_status`

	rows, err = db.Query(fmt.Sprintf(sessionQuery, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report.SessionSummary = make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			report.SessionSummary[status] = count
		}
	}

	// Safety metrics
	safetyQuery := `
		SELECT
			COUNT(*) FILTER (WHERE patient_safety_flag = true) as safety_events,
			COUNT(*) as total_events,
			COUNT(DISTINCT clinical_domain) as clinical_domains_affected
		FROM audit_events
		WHERE event_timestamp >= NOW() - INTERVAL '%d days'`

	var safetyEvents, totalEvents, domainsAffected int
	err = db.QueryRow(fmt.Sprintf(safetyQuery, days)).Scan(&safetyEvents, &totalEvents, &domainsAffected)
	if err != nil {
		return nil, err
	}

	report.SafetyMetrics = map[string]interface{}{
		"safety_events":           safetyEvents,
		"total_events":           totalEvents,
		"clinical_domains_affected": domainsAffected,
	}

	if totalEvents > 0 {
		report.SafetyMetrics["safety_event_percentage"] = float64(safetyEvents) / float64(totalEvents) * 100
	}

	return report, nil
}

func formatReportAsText(report *AuditReport) string {
	text := fmt.Sprintf("Audit Report - %s\nGenerated: %s\n\n", report.Period, report.GeneratedAt.Format("2006-01-02 15:04:05"))

	text += "Event Summary:\n"
	for category, count := range report.EventSummary {
		text += fmt.Sprintf("  %s: %d\n", category, count)
	}

	text += "\nSession Summary:\n"
	for status, count := range report.SessionSummary {
		text += fmt.Sprintf("  %s: %d\n", status, count)
	}

	text += "\nSafety Metrics:\n"
	for metric, value := range report.SafetyMetrics {
		text += fmt.Sprintf("  %s: %v\n", metric, value)
	}

	return text
}