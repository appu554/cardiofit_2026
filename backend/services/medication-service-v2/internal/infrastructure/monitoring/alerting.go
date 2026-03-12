package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high" 
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

// AlertCategory represents the category of an alert
type AlertCategory string

const (
	CategoryPatientSafety AlertCategory = "patient_safety"
	CategorySystemHealth  AlertCategory = "system_health"
	CategoryPerformance   AlertCategory = "performance"
	CategorySecurity      AlertCategory = "security"
	CategoryCompliance    AlertCategory = "compliance"
	CategoryDataQuality   AlertCategory = "data_quality"
)

// Alert represents a system alert
type Alert struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Category    AlertCategory `json:"category"`
	Severity    AlertSeverity `json:"severity"`
	Status      AlertStatus   `json:"status"`
	Description string        `json:"description"`
	
	// Timing
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	LastUpdate   time.Time  `json:"last_update"`
	
	// Context
	ServiceName   string                 `json:"service_name"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	PatientID     string                 `json:"patient_id,omitempty"` // Hashed
	
	// Alert details
	TriggerValue  interface{}            `json:"trigger_value"`
	ThresholdValue interface{}           `json:"threshold_value"`
	CurrentValue  interface{}            `json:"current_value"`
	
	// Healthcare-specific
	ClinicalImpact string   `json:"clinical_impact,omitempty"`
	SafetyRisk     string   `json:"safety_risk,omitempty"`
	ActionRequired []string `json:"action_required,omitempty"`
	
	// Metadata
	Labels   map[string]string      `json:"labels"`
	Metadata map[string]interface{} `json:"metadata"`
	
	// Escalation
	EscalationLevel int       `json:"escalation_level"`
	NotifiedUsers   []string  `json:"notified_users"`
	AckBy           string    `json:"acknowledged_by,omitempty"`
	AckTime         *time.Time `json:"acknowledged_time,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID              string        `yaml:"id"`
	Name            string        `yaml:"name"`
	Category        AlertCategory `yaml:"category"`
	Severity        AlertSeverity `yaml:"severity"`
	Description     string        `yaml:"description"`
	
	// Rule conditions
	MetricName      string        `yaml:"metric_name"`
	Operator        string        `yaml:"operator"` // >, <, >=, <=, ==, !=
	Threshold       interface{}   `yaml:"threshold"`
	Duration        time.Duration `yaml:"duration"`
	
	// Healthcare-specific
	PatientSafety   bool          `yaml:"patient_safety"`
	ClinicalImpact  string        `yaml:"clinical_impact"`
	SafetyRisk      string        `yaml:"safety_risk"`
	ActionRequired  []string      `yaml:"action_required"`
	
	// Notification
	NotifyChannels  []string      `yaml:"notify_channels"`
	EscalationRules []EscalationRule `yaml:"escalation_rules"`
	
	// Rule metadata
	Labels          map[string]string `yaml:"labels"`
	Enabled         bool              `yaml:"enabled"`
}

// EscalationRule defines escalation behavior for alerts
type EscalationRule struct {
	Level       int           `yaml:"level"`
	Duration    time.Duration `yaml:"duration"`
	Channels    []string      `yaml:"channels"`
	Recipients  []string      `yaml:"recipients"`
	Message     string        `yaml:"message"`
}

// AlertingConfig holds configuration for the alerting system
type AlertingConfig struct {
	Rules                []AlertRule   `yaml:"rules"`
	EvaluationInterval   time.Duration `yaml:"evaluation_interval"`
	NotificationChannels map[string]NotificationChannelConfig `yaml:"notification_channels"`
	DefaultSeverity      AlertSeverity `yaml:"default_severity"`
	MaxAlerts            int           `yaml:"max_alerts"`
	RetentionDays        int           `yaml:"retention_days"`
}

// NotificationChannelConfig defines notification channel configuration
type NotificationChannelConfig struct {
	Type     string                 `yaml:"type"` // email, slack, pagerduty, webhook
	Settings map[string]interface{} `yaml:"settings"`
	Enabled  bool                   `yaml:"enabled"`
}

// AlertManager manages the alerting system
type AlertManager struct {
	config          *AlertingConfig
	rules           map[string]*AlertRule
	activeAlerts    map[string]*Alert
	metrics         *Metrics
	logger          *Logger
	notificationMgr *NotificationManager
	
	// State management
	mu              sync.RWMutex
	evaluationTicker *time.Ticker
	shutdownCh       chan struct{}
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *AlertingConfig, metrics *Metrics, logger *Logger) *AlertManager {
	am := &AlertManager{
		config:          config,
		rules:           make(map[string]*AlertRule),
		activeAlerts:    make(map[string]*Alert),
		metrics:         metrics,
		logger:          logger,
		notificationMgr: NewNotificationManager(config.NotificationChannels, logger),
		shutdownCh:      make(chan struct{}),
	}

	// Load rules
	for _, rule := range config.Rules {
		am.rules[rule.ID] = &rule
	}

	return am
}

// Start starts the alert manager
func (am *AlertManager) Start(ctx context.Context) error {
	am.logger.Info("Starting alert manager",
		zap.Int("rules_count", len(am.rules)),
		zap.Duration("evaluation_interval", am.config.EvaluationInterval),
	)

	// Start evaluation loop
	am.evaluationTicker = time.NewTicker(am.config.EvaluationInterval)
	go am.evaluationLoop(ctx)

	// Load default healthcare alert rules
	am.loadHealthcareAlertRules()

	return nil
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	close(am.shutdownCh)
	if am.evaluationTicker != nil {
		am.evaluationTicker.Stop()
	}
	am.logger.Info("Alert manager stopped")
}

// evaluationLoop runs the main evaluation loop
func (am *AlertManager) evaluationLoop(ctx context.Context) {
	for {
		select {
		case <-am.evaluationTicker.C:
			am.evaluateRules(ctx)
		case <-am.shutdownCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// evaluateRules evaluates all alert rules
func (am *AlertManager) evaluateRules(ctx context.Context) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		am.evaluateRule(ctx, rule)
	}

	// Clean up resolved alerts
	am.cleanupResolvedAlerts()
}

// evaluateRule evaluates a single alert rule
func (am *AlertManager) evaluateRule(ctx context.Context, rule *AlertRule) {
	// Get current metric value
	currentValue := am.getMetricValue(rule.MetricName)
	if currentValue == nil {
		return
	}

	// Check if rule condition is met
	conditionMet := am.evaluateCondition(currentValue, rule.Operator, rule.Threshold)
	
	alertID := fmt.Sprintf("%s_%s", rule.ID, rule.MetricName)
	existingAlert, exists := am.activeAlerts[alertID]

	if conditionMet {
		if !exists {
			// Create new alert
			alert := &Alert{
				ID:             alertID,
				Name:           rule.Name,
				Category:       rule.Category,
				Severity:       rule.Severity,
				Status:         StatusFiring,
				Description:    rule.Description,
				StartTime:      time.Now(),
				LastUpdate:     time.Now(),
				ServiceName:    "medication-service-v2",
				TriggerValue:   currentValue,
				ThresholdValue: rule.Threshold,
				CurrentValue:   currentValue,
				ClinicalImpact: rule.ClinicalImpact,
				SafetyRisk:     rule.SafetyRisk,
				ActionRequired: rule.ActionRequired,
				Labels:         rule.Labels,
				Metadata:       make(map[string]interface{}),
				EscalationLevel: 0,
				NotifiedUsers:  []string{},
			}

			am.activeAlerts[alertID] = alert
			am.fireAlert(ctx, alert)
			
		} else if existingAlert.Status == StatusResolved {
			// Re-fire resolved alert
			existingAlert.Status = StatusFiring
			existingAlert.LastUpdate = time.Now()
			existingAlert.CurrentValue = currentValue
			existingAlert.EndTime = nil
			
			am.fireAlert(ctx, existingAlert)
		} else {
			// Update existing firing alert
			existingAlert.CurrentValue = currentValue
			existingAlert.LastUpdate = time.Now()
			
			// Check for escalation
			am.checkEscalation(ctx, existingAlert, rule)
		}
	} else if exists && existingAlert.Status == StatusFiring {
		// Resolve alert
		existingAlert.Status = StatusResolved
		existingAlert.LastUpdate = time.Now()
		now := time.Now()
		existingAlert.EndTime = &now
		
		am.resolveAlert(ctx, existingAlert)
	}
}

// fireAlert fires a new alert
func (am *AlertManager) fireAlert(ctx context.Context, alert *Alert) {
	am.logger.Warn("Alert fired",
		zap.String("alert_id", alert.ID),
		zap.String("alert_name", alert.Name),
		zap.String("category", string(alert.Category)),
		zap.String("severity", string(alert.Severity)),
		zap.Any("current_value", alert.CurrentValue),
		zap.Any("threshold_value", alert.ThresholdValue),
	)

	// Send notifications
	am.notificationMgr.SendAlert(ctx, alert)

	// Record metric
	am.metrics.RecordCounter("alerts_fired", 1, map[string]string{
		"category": string(alert.Category),
		"severity": string(alert.Severity),
	})

	// Log audit event for patient safety alerts
	if alert.Category == CategoryPatientSafety {
		am.logPatientSafetyAlert(ctx, alert)
	}
}

// resolveAlert resolves an alert
func (am *AlertManager) resolveAlert(ctx context.Context, alert *Alert) {
	am.logger.Info("Alert resolved",
		zap.String("alert_id", alert.ID),
		zap.String("alert_name", alert.Name),
		zap.Duration("duration", alert.LastUpdate.Sub(alert.StartTime)),
	)

	// Send resolution notification
	am.notificationMgr.SendResolution(ctx, alert)

	// Record metric
	am.metrics.RecordCounter("alerts_resolved", 1, map[string]string{
		"category": string(alert.Category),
		"severity": string(alert.Severity),
	})
}

// checkEscalation checks if an alert should be escalated
func (am *AlertManager) checkEscalation(ctx context.Context, alert *Alert, rule *AlertRule) {
	duration := time.Since(alert.StartTime)
	
	for _, escalationRule := range rule.EscalationRules {
		if escalationRule.Level > alert.EscalationLevel && duration >= escalationRule.Duration {
			am.escalateAlert(ctx, alert, &escalationRule)
			break
		}
	}
}

// escalateAlert escalates an alert to the next level
func (am *AlertManager) escalateAlert(ctx context.Context, alert *Alert, escalationRule *EscalationRule) {
	alert.EscalationLevel = escalationRule.Level
	alert.LastUpdate = time.Now()

	am.logger.Warn("Alert escalated",
		zap.String("alert_id", alert.ID),
		zap.Int("escalation_level", escalationRule.Level),
		zap.Strings("channels", escalationRule.Channels),
	)

	// Send escalation notification
	am.notificationMgr.SendEscalation(ctx, alert, escalationRule)

	// Record metric
	am.metrics.RecordCounter("alerts_escalated", 1, map[string]string{
		"category": string(alert.Category),
		"level":    fmt.Sprintf("%d", escalationRule.Level),
	})
}

// Manual alert operations

// FireManualAlert fires a manual alert
func (am *AlertManager) FireManualAlert(ctx context.Context, alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert.ID = fmt.Sprintf("manual_%d", time.Now().UnixNano())
	alert.StartTime = time.Now()
	alert.LastUpdate = time.Now()
	alert.Status = StatusFiring

	am.activeAlerts[alert.ID] = alert
	am.fireAlert(ctx, alert)
}

// AcknowledgeAlert acknowledges an alert
func (am *AlertManager) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.AckBy = userID
	now := time.Now()
	alert.AckTime = &now
	alert.LastUpdate = now

	am.logger.Info("Alert acknowledged",
		zap.String("alert_id", alertID),
		zap.String("acknowledged_by", userID),
	)

	return nil
}

// SilenceAlert silences an alert
func (am *AlertManager) SilenceAlert(ctx context.Context, alertID string, duration time.Duration) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Status = StatusSilenced
	alert.LastUpdate = time.Now()
	
	// Set silence expiration
	alert.Metadata["silence_until"] = time.Now().Add(duration)

	am.logger.Info("Alert silenced",
		zap.String("alert_id", alertID),
		zap.Duration("duration", duration),
	)

	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		if alert.Status == StatusFiring {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// Healthcare-specific alert rules

// loadHealthcareAlertRules loads default healthcare alert rules
func (am *AlertManager) loadHealthcareAlertRules() {
	healthcareRules := []*AlertRule{
		{
			ID:          "patient_safety_violation",
			Name:        "Patient Safety Violation Detected",
			Category:    CategoryPatientSafety,
			Severity:    SeverityCritical,
			Description: "Critical patient safety violation detected",
			MetricName:  "medication_service_safety_violations_total",
			Operator:    ">",
			Threshold:   0,
			Duration:    0,
			PatientSafety: true,
			ClinicalImpact: "High - Immediate patient risk",
			SafetyRisk:    "Critical - Potential patient harm",
			ActionRequired: []string{
				"Stop medication process",
				"Review patient case immediately", 
				"Notify attending physician",
				"Document safety incident",
			},
			NotifyChannels: []string{"critical_alerts", "safety_team"},
			Enabled:       true,
		},
		{
			ID:          "dosage_calculation_accuracy_low",
			Name:        "Dosage Calculation Accuracy Below Threshold",
			Category:    CategoryPatientSafety,
			Severity:    SeverityHigh,
			Description: "Dosage calculation accuracy has fallen below acceptable threshold",
			MetricName:  "medication_service_dosage_calculation_accuracy",
			Operator:    "<",
			Threshold:   0.95, // 95% accuracy threshold
			Duration:    5 * time.Minute,
			ClinicalImpact: "High - Potential dosing errors",
			SafetyRisk:    "High - Incorrect dosing",
			ActionRequired: []string{
				"Review calculation engine",
				"Validate recent calculations",
				"Check input data quality",
			},
			Enabled: true,
		},
		{
			ID:          "response_time_high",
			Name:        "High Response Time Detected", 
			Category:    CategoryPerformance,
			Severity:    SeverityMedium,
			Description: "API response time exceeds acceptable threshold",
			MetricName:  "medication_service_http_request_duration_seconds",
			Operator:    ">",
			Threshold:   0.25, // 250ms threshold
			Duration:    2 * time.Minute,
			ClinicalImpact: "Medium - Delayed clinical decisions",
			ActionRequired: []string{
				"Check system resources",
				"Review database performance",
				"Monitor external dependencies",
			},
			Enabled: true,
		},
		{
			ID:          "error_rate_high",
			Name:        "Error Rate Above Threshold",
			Category:    CategorySystemHealth,
			Severity:    SeverityHigh,
			Description: "System error rate has exceeded acceptable threshold",
			MetricName:  "medication_service_errors_total",
			Operator:    ">",
			Threshold:   10, // 10 errors per evaluation interval
			Duration:    1 * time.Minute,
			ClinicalImpact: "High - Service disruption",
			ActionRequired: []string{
				"Investigate error patterns",
				"Check service health",
				"Review recent deployments",
			},
			Enabled: true,
		},
		{
			ID:          "database_connection_issues",
			Name:        "Database Connection Issues",
			Category:    CategorySystemHealth,
			Severity:    SeverityCritical,
			Description: "Database connectivity issues detected",
			MetricName:  "medication_service_database_connections_active",
			Operator:    "<",
			Threshold:   1,
			Duration:    30 * time.Second,
			ClinicalImpact: "Critical - No data access",
			SafetyRisk:    "Critical - Cannot access patient data",
			ActionRequired: []string{
				"Check database health",
				"Verify network connectivity",
				"Activate backup systems if available",
			},
			Enabled: true,
		},
	}

	// Add healthcare rules to the rule map
	for _, rule := range healthcareRules {
		am.rules[rule.ID] = rule
	}

	am.logger.Info("Healthcare alert rules loaded",
		zap.Int("healthcare_rules_count", len(healthcareRules)),
	)
}

// Helper methods

// getMetricValue retrieves the current value for a metric
func (am *AlertManager) getMetricValue(metricName string) interface{} {
	// This would query the actual metrics registry
	// For now, return mock values based on metric name
	switch metricName {
	case "medication_service_safety_violations_total":
		return 0 // No safety violations
	case "medication_service_dosage_calculation_accuracy":
		return 0.98 // 98% accuracy
	case "medication_service_http_request_duration_seconds":
		return 0.12 // 120ms average
	case "medication_service_errors_total":
		return 2 // 2 errors
	case "medication_service_database_connections_active":
		return 5 // 5 active connections
	default:
		return nil
	}
}

// evaluateCondition evaluates an alert condition
func (am *AlertManager) evaluateCondition(currentValue interface{}, operator string, threshold interface{}) bool {
	// Convert values to float64 for comparison
	current, ok1 := convertToFloat64(currentValue)
	thresh, ok2 := convertToFloat64(threshold)
	
	if !ok1 || !ok2 {
		return false
	}

	switch operator {
	case ">":
		return current > thresh
	case "<":
		return current < thresh
	case ">=":
		return current >= thresh
	case "<=":
		return current <= thresh
	case "==":
		return current == thresh
	case "!=":
		return current != thresh
	default:
		return false
	}
}

// convertToFloat64 converts interface{} to float64
func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// cleanupResolvedAlerts removes old resolved alerts
func (am *AlertManager) cleanupResolvedAlerts() {
	cutoff := time.Now().Add(-time.Duration(am.config.RetentionDays) * 24 * time.Hour)
	
	for id, alert := range am.activeAlerts {
		if alert.Status == StatusResolved && 
		   alert.EndTime != nil && 
		   alert.EndTime.Before(cutoff) {
			delete(am.activeAlerts, id)
		}
	}
}

// logPatientSafetyAlert logs patient safety alerts to audit trail
func (am *AlertManager) logPatientSafetyAlert(ctx context.Context, alert *Alert) {
	auditEvent := &AuditEvent{
		EventID:       fmt.Sprintf("safety_alert_%s", alert.ID),
		EventType:     "patient_safety_alert",
		EventCategory: "safety_violation",
		PatientID:     alert.PatientID,
		Operation:     "safety_alert_fired",
		Outcome:       "alert_created",
		Description:   alert.Description,
		ClinicalContext: map[string]interface{}{
			"alert_severity":    alert.Severity,
			"clinical_impact":   alert.ClinicalImpact,
			"safety_risk":       alert.SafetyRisk,
			"action_required":   alert.ActionRequired,
			"trigger_value":     alert.TriggerValue,
			"threshold_value":   alert.ThresholdValue,
		},
		SafetyImpact:    alert.ClinicalImpact,
		ComplianceFlags: []string{"hipaa", "patient_safety", "audit_trail"},
		Metadata: map[string]interface{}{
			"alert_id":          alert.ID,
			"alert_category":    alert.Category,
			"escalation_level":  alert.EscalationLevel,
		},
	}

	am.logger.LogAuditEvent(ctx, auditEvent)
}