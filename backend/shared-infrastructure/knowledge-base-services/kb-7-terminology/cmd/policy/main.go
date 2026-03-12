package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/google/uuid"
	"database/sql"
	_ "github.com/lib/pq"
)

// PolicyRule represents a policy rule
type PolicyRule struct {
	ID                string                 `json:"id"`
	RuleID            string                 `json:"rule_id"`
	RuleName          string                 `json:"rule_name"`
	RuleVersion       string                 `json:"rule_version"`
	RuleType          string                 `json:"rule_type"`
	RuleCategory      string                 `json:"rule_category"`
	Priority          int                    `json:"priority"`
	RuleDescription   string                 `json:"rule_description"`
	RuleExpression    map[string]interface{} `json:"rule_expression"`
	RuleConditions    map[string]interface{} `json:"rule_conditions,omitempty"`
	TriggerEvents     []string               `json:"trigger_events"`
	ResourceTypes     []string               `json:"resource_types,omitempty"`
	ClinicalDomains   []string               `json:"clinical_domains,omitempty"`
	ActionType        string                 `json:"action_type"`
	ActionParameters  map[string]interface{} `json:"action_parameters,omitempty"`
	EscalationRules   map[string]interface{} `json:"escalation_rules,omitempty"`
	Status            string                 `json:"status"`
	EffectiveDate     time.Time              `json:"effective_date"`
	ExpiryDate        *time.Time             `json:"expiry_date,omitempty"`
	CreatedBy         string                 `json:"created_by"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyEvaluation represents a policy evaluation result
type PolicyEvaluation struct {
	ID                  string                 `json:"id"`
	EvaluationID        string                 `json:"evaluation_id"`
	EvaluationTimestamp time.Time              `json:"evaluation_timestamp"`
	RuleSetID           string                 `json:"rule_set_id"`
	RuleID              *string                `json:"rule_id,omitempty"`
	EventType           string                 `json:"event_type"`
	ResourceType        string                 `json:"resource_type"`
	ResourceID          *string                `json:"resource_id,omitempty"`
	ActorID             string                 `json:"actor_id"`
	InputData           map[string]interface{} `json:"input_data"`
	ContextData         map[string]interface{} `json:"context_data,omitempty"`
	EvaluationResult    string                 `json:"evaluation_result"`
	RuleMatched         bool                   `json:"rule_matched"`
	MatchConfidence     *float64               `json:"match_confidence,omitempty"`
	DecisionReason      *string                `json:"decision_reason,omitempty"`
	TriggeredActions    map[string]interface{} `json:"triggered_actions,omitempty"`
	EscalationTriggered bool                   `json:"escalation_triggered"`
	EvaluationDuration  *int                   `json:"evaluation_duration_ms,omitempty"`
	AuditEventID        *string                `json:"audit_event_id,omitempty"`
	AuditSessionID      *string                `json:"audit_session_id,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationContext represents the context for policy validation
type ValidationContext struct {
	EventType    string                 `json:"event_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ActorID      string                 `json:"actor_id"`
	Data         map[string]interface{} `json:"data"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

var (
	databaseURL string
	db          *sql.DB
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "policy",
		Short: "KB-7 Clinical Policy Engine",
		Long:  "Command-line tool for managing and validating clinical policies",
	}

	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", "", "Database connection URL")

	// Commands
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(listRulesCmd())
	rootCmd.AddCommand(evaluateCmd())
	rootCmd.AddCommand(createRuleCmd())
	rootCmd.AddCommand(reportCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func validateCmd() *cobra.Command {
	var (
		policySet string
		prFiles   string
		actorID   string
		eventType string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate against clinical policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			// Parse PR files if provided
			var files []string
			if prFiles != "" {
				files = strings.Split(prFiles, ",")
			}

			// Create validation context
			ctx := &ValidationContext{
				EventType:    eventType,
				ResourceType: "terminology_change",
				ActorID:      actorID,
				Data: map[string]interface{}{
					"changed_files": files,
					"policy_set":   policySet,
				},
				Context: map[string]interface{}{
					"validation_timestamp": time.Now(),
					"validation_source":    "cli",
				},
			}

			// Evaluate policies
			result, err := evaluatePolicies(policySet, ctx)
			if err != nil {
				return fmt.Errorf("policy evaluation failed: %w", err)
			}

			// Display results
			fmt.Printf("Policy Validation Results:\n")
			fmt.Printf("Policy Set: %s\n", policySet)
			fmt.Printf("Overall Result: %s\n", result.EvaluationResult)
			fmt.Printf("Rules Matched: %t\n", result.RuleMatched)

			if result.DecisionReason != nil {
				fmt.Printf("Decision Reason: %s\n", *result.DecisionReason)
			}

			if result.EscalationTriggered {
				fmt.Printf("⚠️  Escalation Triggered\n")
			}

			// Exit with appropriate code
			if result.EvaluationResult == "block" {
				fmt.Printf("\n❌ Validation FAILED - Changes blocked by policy\n")
				os.Exit(1)
			} else if result.EvaluationResult == "require_review" {
				fmt.Printf("\n🔍 Validation requires REVIEW\n")
			} else if result.EvaluationResult == "warn" {
				fmt.Printf("\n⚠️  Validation passed with WARNINGS\n")
			} else {
				fmt.Printf("\n✅ Validation PASSED\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&policySet, "policy-set", "clinical-safety", "Policy set to validate against")
	cmd.Flags().StringVar(&prFiles, "pr-files", "", "Comma-separated list of changed files")
	cmd.Flags().StringVar(&actorID, "actor", "unknown", "Actor performing the action")
	cmd.Flags().StringVar(&eventType, "event-type", "terminology_change", "Type of event being validated")

	return cmd
}

func listRulesCmd() *cobra.Command {
	var (
		ruleSet        string
		category       string
		clinicalDomain string
		activeOnly     bool
	)

	cmd := &cobra.Command{
		Use:   "list-rules",
		Short: "List policy rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			rules, err := listPolicyRules(ruleSet, category, clinicalDomain, activeOnly)
			if err != nil {
				return fmt.Errorf("failed to list rules: %w", err)
			}

			fmt.Printf("Policy Rules:\n")
			fmt.Printf("%-30s %-15s %-15s %-10s %-15s\n", "Rule ID", "Type", "Category", "Priority", "Action")
			fmt.Printf("%-30s %-15s %-15s %-10s %-15s\n", strings.Repeat("-", 30), strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 10), strings.Repeat("-", 15))

			for _, rule := range rules {
				fmt.Printf("%-30s %-15s %-15s %-10d %-15s\n",
					rule.RuleID,
					rule.RuleType,
					rule.RuleCategory,
					rule.Priority,
					rule.ActionType,
				)
			}

			fmt.Printf("\nTotal: %d rules\n", len(rules))
			return nil
		},
	}

	cmd.Flags().StringVar(&ruleSet, "rule-set", "", "Filter by rule set")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	cmd.Flags().StringVar(&clinicalDomain, "clinical-domain", "", "Filter by clinical domain")
	cmd.Flags().BoolVar(&activeOnly, "active-only", true, "Show only active rules")

	return cmd
}

func evaluateCmd() *cobra.Command {
	var (
		ruleSetID    string
		eventType    string
		resourceType string
		resourceID   string
		actorID      string
		inputData    string
	)

	cmd := &cobra.Command{
		Use:   "evaluate",
		Short: "Evaluate a specific scenario against policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			// Parse input data
			var data map[string]interface{}
			if inputData != "" {
				if err := json.Unmarshal([]byte(inputData), &data); err != nil {
					return fmt.Errorf("invalid input data JSON: %w", err)
				}
			} else {
				data = make(map[string]interface{})
			}

			ctx := &ValidationContext{
				EventType:    eventType,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				ActorID:      actorID,
				Data:         data,
			}

			result, err := evaluatePolicies(ruleSetID, ctx)
			if err != nil {
				return fmt.Errorf("evaluation failed: %w", err)
			}

			// Pretty print result
			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format result: %w", err)
			}

			fmt.Printf("Evaluation Result:\n%s\n", string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&ruleSetID, "rule-set", "", "Rule set ID to evaluate against")
	cmd.Flags().StringVar(&eventType, "event-type", "", "Event type")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Resource type")
	cmd.Flags().StringVar(&resourceID, "resource-id", "", "Resource ID")
	cmd.Flags().StringVar(&actorID, "actor", "", "Actor ID")
	cmd.Flags().StringVar(&inputData, "input-data", "", "Input data as JSON")

	cmd.MarkFlagRequired("rule-set")
	cmd.MarkFlagRequired("event-type")
	cmd.MarkFlagRequired("resource-type")
	cmd.MarkFlagRequired("actor")

	return cmd
}

func createRuleCmd() *cobra.Command {
	var (
		ruleID          string
		ruleName        string
		ruleType        string
		category        string
		priority        int
		description     string
		expression      string
		actionType      string
		triggerEvents   string
		clinicalDomains string
	)

	cmd := &cobra.Command{
		Use:   "create-rule",
		Short: "Create a new policy rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			// Parse expression
			var expr map[string]interface{}
			if err := json.Unmarshal([]byte(expression), &expr); err != nil {
				return fmt.Errorf("invalid expression JSON: %w", err)
			}

			// Parse trigger events
			var triggers []string
			if triggerEvents != "" {
				triggers = strings.Split(triggerEvents, ",")
			}

			// Parse clinical domains
			var domains []string
			if clinicalDomains != "" {
				domains = strings.Split(clinicalDomains, ",")
			}

			rule := &PolicyRule{
				ID:              uuid.New().String(),
				RuleID:          ruleID,
				RuleName:        ruleName,
				RuleVersion:     "1.0.0",
				RuleType:        ruleType,
				RuleCategory:    category,
				Priority:        priority,
				RuleDescription: description,
				RuleExpression:  expr,
				TriggerEvents:   triggers,
				ClinicalDomains: domains,
				ActionType:      actionType,
				Status:          "active",
				EffectiveDate:   time.Now(),
				CreatedBy:       "cli-user",
			}

			if err := createPolicyRule(rule); err != nil {
				return fmt.Errorf("failed to create rule: %w", err)
			}

			fmt.Printf("Created policy rule: %s\n", rule.RuleID)
			return nil
		},
	}

	cmd.Flags().StringVar(&ruleID, "rule-id", "", "Unique rule identifier")
	cmd.Flags().StringVar(&ruleName, "rule-name", "", "Rule name")
	cmd.Flags().StringVar(&ruleType, "rule-type", "", "Rule type (validation, approval, notification, blocking)")
	cmd.Flags().StringVar(&category, "category", "", "Rule category (safety, quality, compliance, operational)")
	cmd.Flags().IntVar(&priority, "priority", 100, "Rule priority (lower = higher priority)")
	cmd.Flags().StringVar(&description, "description", "", "Rule description")
	cmd.Flags().StringVar(&expression, "expression", "", "Rule expression as JSON")
	cmd.Flags().StringVar(&actionType, "action-type", "", "Action type (allow, warn, block, require_review)")
	cmd.Flags().StringVar(&triggerEvents, "trigger-events", "", "Comma-separated trigger events")
	cmd.Flags().StringVar(&clinicalDomains, "clinical-domains", "", "Comma-separated clinical domains")

	cmd.MarkFlagRequired("rule-id")
	cmd.MarkFlagRequired("rule-name")
	cmd.MarkFlagRequired("rule-type")
	cmd.MarkFlagRequired("category")
	cmd.MarkFlagRequired("description")
	cmd.MarkFlagRequired("expression")
	cmd.MarkFlagRequired("action-type")

	return cmd
}

func reportCmd() *cobra.Command {
	var (
		days       int
		outputFile string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate policy evaluation report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return err
			}
			defer db.Close()

			report, err := generatePolicyReport(days)
			if err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			var output []byte
			switch format {
			case "json":
				output, err = json.MarshalIndent(report, "", "  ")
			default:
				output = []byte(formatPolicyReportAsText(report))
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

	cmd.Flags().IntVar(&days, "days", 7, "Number of days to include in report")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json)")

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

func evaluatePolicies(ruleSetID string, ctx *ValidationContext) (*PolicyEvaluation, error) {
	evaluation := &PolicyEvaluation{
		ID:                  uuid.New().String(),
		EvaluationID:        uuid.New().String(),
		EvaluationTimestamp: time.Now(),
		RuleSetID:           ruleSetID,
		EventType:           ctx.EventType,
		ResourceType:        ctx.ResourceType,
		ActorID:             ctx.ActorID,
		InputData:           ctx.Data,
		ContextData:         ctx.Context,
		EvaluationResult:    "allow", // Default
		RuleMatched:         false,
		EscalationTriggered: false,
	}

	if ctx.ResourceID != "" {
		evaluation.ResourceID = &ctx.ResourceID
	}

	start := time.Now()

	// Get active rules for the rule set
	rules, err := getRuleSetRules(ruleSetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Evaluate each rule
	for _, rule := range rules {
		matched, confidence, reason := evaluateRule(rule, ctx)
		if matched {
			evaluation.RuleMatched = true
			evaluation.RuleID = &rule.RuleID
			evaluation.MatchConfidence = &confidence
			evaluation.DecisionReason = &reason

			// Apply rule action
			switch rule.ActionType {
			case "block":
				evaluation.EvaluationResult = "block"
				// Stop on first blocking rule
				break
			case "require_review":
				evaluation.EvaluationResult = "require_review"
			case "warn":
				if evaluation.EvaluationResult == "allow" {
					evaluation.EvaluationResult = "warn"
				}
			}

			// Check for escalation
			if rule.EscalationRules != nil {
				evaluation.EscalationTriggered = true
			}

			// If blocking rule matched, stop evaluation
			if rule.ActionType == "block" {
				break
			}
		}
	}

	duration := int(time.Since(start).Milliseconds())
	evaluation.EvaluationDuration = &duration

	// Save evaluation to database
	if err := savePolicyEvaluation(evaluation); err != nil {
		log.Printf("Failed to save policy evaluation: %v", err)
	}

	return evaluation, nil
}

func evaluateRule(rule *PolicyRule, ctx *ValidationContext) (bool, float64, string) {
	// Simple rule evaluation logic
	// In a production system, this would use a proper rules engine like JSONLogic

	// Check if event type matches trigger events
	eventMatches := false
	for _, trigger := range rule.TriggerEvents {
		if trigger == ctx.EventType {
			eventMatches = true
			break
		}
	}

	if !eventMatches {
		return false, 0.0, "Event type does not match trigger events"
	}

	// Check clinical domain if specified
	if len(rule.ClinicalDomains) > 0 {
		domainMatches := false
		if clinicalDomain, exists := ctx.Data["clinical_domain"]; exists {
			domainStr, ok := clinicalDomain.(string)
			if ok {
				for _, domain := range rule.ClinicalDomains {
					if domain == domainStr {
						domainMatches = true
						break
					}
				}
			}
		}
		if !domainMatches {
			return false, 0.0, "Clinical domain does not match rule requirements"
		}
	}

	// Simple expression evaluation
	// This is a simplified version - production would use JSONLogic or similar
	if expr, ok := rule.RuleExpression["simple_match"]; ok {
		if matchValue, ok := expr.(string); ok {
			if dataValue, exists := ctx.Data[matchValue]; exists {
				if dataValue != nil {
					return true, 1.0, fmt.Sprintf("Rule matched on field: %s", matchValue)
				}
			}
		}
	}

	// For medication safety rules
	if rule.RuleCategory == "safety" && strings.Contains(rule.RuleID, "medication") {
		if files, exists := ctx.Data["changed_files"]; exists {
			if fileList, ok := files.([]string); ok {
				for _, file := range fileList {
					if strings.Contains(file, "medication") || strings.Contains(file, "drug") {
						return true, 0.9, "Medication-related file changes detected"
					}
				}
			}
		}
	}

	// For allergy safety rules
	if rule.RuleCategory == "safety" && strings.Contains(rule.RuleID, "allergy") {
		if files, exists := ctx.Data["changed_files"]; exists {
			if fileList, ok := files.([]string); ok {
				for _, file := range fileList {
					if strings.Contains(file, "allergy") || strings.Contains(file, "adverse") {
						return true, 0.9, "Allergy-related file changes detected"
					}
				}
			}
		}
	}

	return false, 0.0, "No rule conditions matched"
}

func getRuleSetRules(ruleSetID string) ([]*PolicyRule, error) {
	query := `
		SELECT r.id, r.rule_id, r.rule_name, r.rule_version, r.rule_type,
			   r.rule_category, r.priority, r.rule_description, r.rule_expression,
			   r.rule_conditions, r.trigger_events, r.resource_types,
			   r.clinical_domains, r.action_type, r.action_parameters,
			   r.escalation_rules, r.status, r.effective_date, r.expiry_date,
			   r.created_by, r.metadata
		FROM policy_rules r
		JOIN policy_rule_set_rules rsr ON r.id = rsr.rule_id
		JOIN policy_rule_sets rs ON rsr.rule_set_id = rs.id
		WHERE rs.rule_set_id = $1
		  AND r.status = 'active'
		  AND (r.expiry_date IS NULL OR r.expiry_date > NOW())
		  AND rs.status = 'active'
		ORDER BY rsr.execution_order, r.priority`

	rows, err := db.Query(query, ruleSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*PolicyRule
	for rows.Next() {
		rule := &PolicyRule{}
		var ruleExpression, ruleConditions, triggerEvents, resourceTypes, clinicalDomains, actionParameters, escalationRules, metadata []byte

		err := rows.Scan(
			&rule.ID, &rule.RuleID, &rule.RuleName, &rule.RuleVersion, &rule.RuleType,
			&rule.RuleCategory, &rule.Priority, &rule.RuleDescription, &ruleExpression,
			&ruleConditions, &triggerEvents, &resourceTypes,
			&clinicalDomains, &rule.ActionType, &actionParameters,
			&escalationRules, &rule.Status, &rule.EffectiveDate, &rule.ExpiryDate,
			&rule.CreatedBy, &metadata,
		)
		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		if len(ruleExpression) > 0 {
			json.Unmarshal(ruleExpression, &rule.RuleExpression)
		}
		if len(ruleConditions) > 0 {
			json.Unmarshal(ruleConditions, &rule.RuleConditions)
		}
		if len(triggerEvents) > 0 {
			json.Unmarshal(triggerEvents, &rule.TriggerEvents)
		}
		if len(resourceTypes) > 0 {
			json.Unmarshal(resourceTypes, &rule.ResourceTypes)
		}
		if len(clinicalDomains) > 0 {
			json.Unmarshal(clinicalDomains, &rule.ClinicalDomains)
		}
		if len(actionParameters) > 0 {
			json.Unmarshal(actionParameters, &rule.ActionParameters)
		}
		if len(escalationRules) > 0 {
			json.Unmarshal(escalationRules, &rule.EscalationRules)
		}
		if len(metadata) > 0 {
			json.Unmarshal(metadata, &rule.Metadata)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func listPolicyRules(ruleSet, category, clinicalDomain string, activeOnly bool) ([]*PolicyRule, error) {
	query := `
		SELECT id, rule_id, rule_name, rule_version, rule_type,
			   rule_category, priority, rule_description, action_type,
			   status, effective_date, created_by
		FROM policy_rules
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if activeOnly {
		query += " AND status = 'active' AND (expiry_date IS NULL OR expiry_date > NOW())"
	}

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND rule_category = $%d", argCount)
		args = append(args, category)
	}

	if clinicalDomain != "" {
		argCount++
		query += fmt.Sprintf(" AND clinical_domains @> $%d", argCount)
		args = append(args, fmt.Sprintf("[\"%s\"]", clinicalDomain))
	}

	query += " ORDER BY priority ASC, rule_id"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*PolicyRule
	for rows.Next() {
		rule := &PolicyRule{}
		err := rows.Scan(
			&rule.ID, &rule.RuleID, &rule.RuleName, &rule.RuleVersion, &rule.RuleType,
			&rule.RuleCategory, &rule.Priority, &rule.RuleDescription, &rule.ActionType,
			&rule.Status, &rule.EffectiveDate, &rule.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func createPolicyRule(rule *PolicyRule) error {
	query := `
		INSERT INTO policy_rules (
			id, rule_id, rule_name, rule_version, rule_type, rule_category,
			priority, rule_description, rule_expression, rule_conditions,
			trigger_events, resource_types, clinical_domains, action_type,
			action_parameters, escalation_rules, status, effective_date,
			expiry_date, created_by, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	ruleExpression, _ := json.Marshal(rule.RuleExpression)
	ruleConditions, _ := json.Marshal(rule.RuleConditions)
	triggerEvents, _ := json.Marshal(rule.TriggerEvents)
	resourceTypes, _ := json.Marshal(rule.ResourceTypes)
	clinicalDomains, _ := json.Marshal(rule.ClinicalDomains)
	actionParameters, _ := json.Marshal(rule.ActionParameters)
	escalationRules, _ := json.Marshal(rule.EscalationRules)
	metadata, _ := json.Marshal(rule.Metadata)

	_, err := db.Exec(query,
		rule.ID, rule.RuleID, rule.RuleName, rule.RuleVersion, rule.RuleType, rule.RuleCategory,
		rule.Priority, rule.RuleDescription, ruleExpression, ruleConditions,
		triggerEvents, resourceTypes, clinicalDomains, rule.ActionType,
		actionParameters, escalationRules, rule.Status, rule.EffectiveDate,
		rule.ExpiryDate, rule.CreatedBy, metadata,
	)

	return err
}

func savePolicyEvaluation(evaluation *PolicyEvaluation) error {
	query := `
		INSERT INTO policy_evaluations (
			id, evaluation_id, evaluation_timestamp, rule_set_id, rule_id,
			event_type, resource_type, resource_id, actor_id, input_data,
			context_data, evaluation_result, rule_matched, match_confidence,
			decision_reason, triggered_actions, escalation_triggered,
			evaluation_duration_ms, audit_event_id, audit_session_id, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	inputData, _ := json.Marshal(evaluation.InputData)
	contextData, _ := json.Marshal(evaluation.ContextData)
	triggeredActions, _ := json.Marshal(evaluation.TriggeredActions)
	metadata, _ := json.Marshal(evaluation.Metadata)

	_, err := db.Exec(query,
		evaluation.ID, evaluation.EvaluationID, evaluation.EvaluationTimestamp,
		evaluation.RuleSetID, evaluation.RuleID, evaluation.EventType,
		evaluation.ResourceType, evaluation.ResourceID, evaluation.ActorID,
		inputData, contextData, evaluation.EvaluationResult, evaluation.RuleMatched,
		evaluation.MatchConfidence, evaluation.DecisionReason, triggeredActions,
		evaluation.EscalationTriggered, evaluation.EvaluationDuration,
		evaluation.AuditEventID, evaluation.AuditSessionID, metadata,
	)

	return err
}

// Report generation
type PolicyReport struct {
	Period           string                   `json:"period"`
	GeneratedAt      time.Time                `json:"generated_at"`
	EvaluationSummary map[string]int          `json:"evaluation_summary"`
	RuleMatchSummary map[string]int           `json:"rule_match_summary"`
	EscalationSummary map[string]interface{}  `json:"escalation_summary"`
	TopRules         []map[string]interface{} `json:"top_rules"`
}

func generatePolicyReport(days int) (*PolicyReport, error) {
	report := &PolicyReport{
		Period:      fmt.Sprintf("Last %d days", days),
		GeneratedAt: time.Now(),
	}

	// Evaluation summary
	evalQuery := `
		SELECT evaluation_result, COUNT(*)
		FROM policy_evaluations
		WHERE evaluation_timestamp >= NOW() - INTERVAL '%d days'
		GROUP BY evaluation_result`

	rows, err := db.Query(fmt.Sprintf(evalQuery, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report.EvaluationSummary = make(map[string]int)
	for rows.Next() {
		var result string
		var count int
		if err := rows.Scan(&result, &count); err == nil {
			report.EvaluationSummary[result] = count
		}
	}

	// Rule match summary
	matchQuery := `
		SELECT rule_matched, COUNT(*)
		FROM policy_evaluations
		WHERE evaluation_timestamp >= NOW() - INTERVAL '%d days'
		GROUP BY rule_matched`

	rows, err = db.Query(fmt.Sprintf(matchQuery, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report.RuleMatchSummary = make(map[string]int)
	for rows.Next() {
		var matched bool
		var count int
		if err := rows.Scan(&matched, &count); err == nil {
			key := "not_matched"
			if matched {
				key = "matched"
			}
			report.RuleMatchSummary[key] = count
		}
	}

	// Escalation summary
	escalationQuery := `
		SELECT
			COUNT(*) FILTER (WHERE escalation_triggered = true) as escalations,
			COUNT(*) as total_evaluations
		FROM policy_evaluations
		WHERE evaluation_timestamp >= NOW() - INTERVAL '%d days'`

	var escalations, totalEvaluations int
	err = db.QueryRow(fmt.Sprintf(escalationQuery, days)).Scan(&escalations, &totalEvaluations)
	if err != nil {
		return nil, err
	}

	report.EscalationSummary = map[string]interface{}{
		"escalations":       escalations,
		"total_evaluations": totalEvaluations,
	}

	if totalEvaluations > 0 {
		report.EscalationSummary["escalation_rate"] = float64(escalations) / float64(totalEvaluations) * 100
	}

	return report, nil
}

func formatPolicyReportAsText(report *PolicyReport) string {
	text := fmt.Sprintf("Policy Report - %s\nGenerated: %s\n\n", report.Period, report.GeneratedAt.Format("2006-01-02 15:04:05"))

	text += "Evaluation Summary:\n"
	for result, count := range report.EvaluationSummary {
		text += fmt.Sprintf("  %s: %d\n", result, count)
	}

	text += "\nRule Match Summary:\n"
	for match, count := range report.RuleMatchSummary {
		text += fmt.Sprintf("  %s: %d\n", match, count)
	}

	text += "\nEscalation Summary:\n"
	for metric, value := range report.EscalationSummary {
		text += fmt.Sprintf("  %s: %v\n", metric, value)
	}

	return text
}