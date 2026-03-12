// Package engine provides action execution for the rules engine
package engine

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/database"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ActionExecutor executes rule actions
type ActionExecutor struct {
	db     *database.PostgresDB
	logger *logrus.Logger
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(db *database.PostgresDB, logger *logrus.Logger) *ActionExecutor {
	return &ActionExecutor{
		db:     db,
		logger: logger,
	}
}

// ExecuteActions executes all actions for a triggered rule
func (e *ActionExecutor) ExecuteActions(ctx context.Context, rule *models.Rule, evalCtx *models.EvaluationContext) []models.ActionResult {
	results := make([]models.ActionResult, 0, len(rule.Actions))

	for _, action := range rule.Actions {
		result := e.executeAction(ctx, rule, &action, evalCtx)
		results = append(results, result)
	}

	return results
}

// executeAction executes a single action
func (e *ActionExecutor) executeAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	switch action.Type {
	case models.ActionTypeAlert:
		return e.executeAlertAction(ctx, rule, action, evalCtx)
	case models.ActionTypeEscalate:
		return e.executeEscalateAction(ctx, rule, action, evalCtx)
	case models.ActionTypeNotify:
		return e.executeNotifyAction(ctx, rule, action, evalCtx)
	case models.ActionTypeRecommend:
		return e.executeRecommendAction(ctx, rule, action, evalCtx)
	case models.ActionTypeInference:
		return e.executeInferenceAction(ctx, rule, action, evalCtx)
	case models.ActionTypeDerivation:
		return e.executeDerivationAction(ctx, rule, action, evalCtx)
	case models.ActionTypeSuppress:
		return e.executeSuppressAction(ctx, rule, action, evalCtx)
	case models.ActionTypeLog:
		return e.executeLogAction(ctx, rule, action, evalCtx)
	case models.ActionTypeWebhook:
		return e.executeWebhookAction(ctx, rule, action, evalCtx)
	default:
		return models.ActionResult{
			Type:    action.Type,
			Success: false,
			Error:   fmt.Sprintf("unknown action type: %s", action.Type),
		}
	}
}

// executeAlertAction creates an alert
func (e *ActionExecutor) executeAlertAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	// Render message template
	message, err := e.renderTemplate(action.Message, rule, evalCtx)
	if err != nil {
		e.logger.WithError(err).Warn("Failed to render alert message template")
		message = action.Message
	}

	// Create alert
	alert := &models.Alert{
		ID:          uuid.New().String(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		PatientID:   evalCtx.PatientID,
		EncounterID: evalCtx.EncounterID,
		Severity:    rule.Severity,
		Category:    rule.Category,
		Message:     message,
		Priority:    action.Priority,
		Status:      models.AlertStatusActive,
		Context: map[string]interface{}{
			"rule_id":     rule.ID,
			"rule_type":   rule.Type,
			"triggered_at": time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store alert in database (if database is configured)
	if e.db != nil {
		if err := e.db.CreateAlert(ctx, alert); err != nil {
			e.logger.WithError(err).Error("Failed to create alert")
			return models.ActionResult{
				Type:    action.Type,
				Success: false,
				Error:   err.Error(),
			}
		}
	}

	e.logger.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"rule_id":    rule.ID,
		"patient_id": evalCtx.PatientID,
		"severity":   rule.Severity,
	}).Info("Alert created")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
		AlertID: alert.ID,
		Data: map[string]interface{}{
			"alert_id": alert.ID,
			"severity": rule.Severity,
			"priority": action.Priority,
		},
	}
}

// executeEscalateAction handles escalation
func (e *ActionExecutor) executeEscalateAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	message, _ := e.renderTemplate(action.Message, rule, evalCtx)

	escalationLevel := "STANDARD"
	if level, ok := action.Parameters["level"]; ok {
		escalationLevel = level
	}

	urgency := "ROUTINE"
	if u, ok := action.Parameters["urgency"]; ok {
		urgency = u
	}

	e.logger.WithFields(logrus.Fields{
		"rule_id":    rule.ID,
		"patient_id": evalCtx.PatientID,
		"level":      escalationLevel,
		"urgency":    urgency,
		"recipients": action.Recipients,
		"channel":    action.Channel,
	}).Info("Escalation triggered")

	// In a full implementation, this would integrate with notification systems
	// For now, we log the escalation

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"level":      escalationLevel,
			"urgency":    urgency,
			"recipients": action.Recipients,
			"channel":    action.Channel,
		},
	}
}

// executeNotifyAction sends a notification
func (e *ActionExecutor) executeNotifyAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	message, _ := e.renderTemplate(action.Message, rule, evalCtx)

	e.logger.WithFields(logrus.Fields{
		"rule_id":    rule.ID,
		"patient_id": evalCtx.PatientID,
		"recipients": action.Recipients,
		"channel":    action.Channel,
	}).Info("Notification sent")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"recipients": action.Recipients,
			"channel":    action.Channel,
		},
	}
}

// executeRecommendAction creates a clinical recommendation
func (e *ActionExecutor) executeRecommendAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	message, _ := e.renderTemplate(action.Message, rule, evalCtx)

	e.logger.WithFields(logrus.Fields{
		"rule_id":     rule.ID,
		"patient_id":  evalCtx.PatientID,
		"recommendation": message,
	}).Info("Recommendation generated")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"recommendation": message,
			"evidence":       rule.Evidence,
			"parameters":     action.Parameters,
		},
	}
}

// executeInferenceAction derives new clinical facts
func (e *ActionExecutor) executeInferenceAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	message, _ := e.renderTemplate(action.Message, rule, evalCtx)

	inferredCondition := ""
	if cond, ok := action.Parameters["inferred_condition"]; ok {
		inferredCondition = cond
	}

	confidence := "0.5"
	if conf, ok := action.Parameters["confidence"]; ok {
		confidence = conf
	}

	e.logger.WithFields(logrus.Fields{
		"rule_id":           rule.ID,
		"patient_id":        evalCtx.PatientID,
		"inferred_condition": inferredCondition,
		"confidence":        confidence,
	}).Info("Inference generated")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"inferred_condition": inferredCondition,
			"confidence":         confidence,
		},
	}
}

// executeDerivationAction calculates derived values
func (e *ActionExecutor) executeDerivationAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	outputField := ""
	if field, ok := action.Parameters["output_field"]; ok {
		outputField = field
	}

	calculation := ""
	if calc, ok := action.Parameters["calculation"]; ok {
		calculation = calc
	}

	// In a full implementation, this would execute the calculation
	// For now, we just log the derivation request

	e.logger.WithFields(logrus.Fields{
		"rule_id":      rule.ID,
		"patient_id":   evalCtx.PatientID,
		"output_field": outputField,
		"calculation":  calculation,
	}).Info("Derivation calculated")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Data: map[string]interface{}{
			"output_field": outputField,
			"calculation":  calculation,
		},
	}
}

// executeSuppressAction suppresses other rules/alerts
func (e *ActionExecutor) executeSuppressAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	suppressedRules := []string{}
	if rules, ok := action.Parameters["suppress_rules"]; ok {
		// Parse comma-separated rule IDs
		suppressedRules = append(suppressedRules, rules)
	}

	e.logger.WithFields(logrus.Fields{
		"rule_id":          rule.ID,
		"patient_id":       evalCtx.PatientID,
		"suppressed_rules": suppressedRules,
	}).Info("Suppression applied")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Data: map[string]interface{}{
			"suppressed_rules": suppressedRules,
		},
	}
}

// executeLogAction logs the rule execution
func (e *ActionExecutor) executeLogAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	message, _ := e.renderTemplate(action.Message, rule, evalCtx)

	logLevel := "INFO"
	if level, ok := action.Parameters["level"]; ok {
		logLevel = level
	}

	switch logLevel {
	case "DEBUG":
		e.logger.Debug(message)
	case "WARN":
		e.logger.Warn(message)
	case "ERROR":
		e.logger.Error(message)
	default:
		e.logger.Info(message)
	}

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Message: message,
	}
}

// executeWebhookAction calls an external webhook
func (e *ActionExecutor) executeWebhookAction(ctx context.Context, rule *models.Rule, action *models.Action, evalCtx *models.EvaluationContext) models.ActionResult {
	webhookURL := ""
	if url, ok := action.Parameters["url"]; ok {
		webhookURL = url
	}

	if webhookURL == "" {
		return models.ActionResult{
			Type:    action.Type,
			Success: false,
			Error:   "webhook URL not specified",
		}
	}

	// In a full implementation, this would make an HTTP call
	// For now, we just log the webhook request

	e.logger.WithFields(logrus.Fields{
		"rule_id":     rule.ID,
		"patient_id":  evalCtx.PatientID,
		"webhook_url": webhookURL,
	}).Info("Webhook called")

	return models.ActionResult{
		Type:    action.Type,
		Success: true,
		Data: map[string]interface{}{
			"url": webhookURL,
		},
	}
}

// renderTemplate renders a message template with context data
func (e *ActionExecutor) renderTemplate(templateStr string, rule *models.Rule, evalCtx *models.EvaluationContext) (string, error) {
	if templateStr == "" {
		return "", nil
	}

	// Create template data
	data := map[string]interface{}{
		"Rule": map[string]interface{}{
			"ID":       rule.ID,
			"Name":     rule.Name,
			"Type":     rule.Type,
			"Category": rule.Category,
			"Severity": rule.Severity,
		},
		"Context": map[string]interface{}{
			"PatientID":   evalCtx.PatientID,
			"EncounterID": evalCtx.EncounterID,
			"Labs":        evalCtx.Labs,
			"Vitals":      evalCtx.Vitals,
			"Medications": evalCtx.Medications,
			"Conditions":  evalCtx.Conditions,
			"Patient":     evalCtx.Patient,
		},
		"Timestamp": time.Now().Format(time.RFC3339),
	}

	tmpl, err := template.New("message").Parse(templateStr)
	if err != nil {
		return templateStr, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return templateStr, err
	}

	return buf.String(), nil
}
