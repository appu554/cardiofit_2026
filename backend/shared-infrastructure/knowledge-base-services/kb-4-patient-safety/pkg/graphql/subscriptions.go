package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"kb-patient-safety/pkg/analytics"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// SubscriptionType represents the type of subscription
type SubscriptionType string

const (
	SubTypeSafetyAlerts    SubscriptionType = "SAFETY_ALERTS"
	SubTypeSignalDetection SubscriptionType = "SIGNAL_DETECTION"
	SubTypeOverrides       SubscriptionType = "OVERRIDES"
	SubTypeServiceHealth   SubscriptionType = "SERVICE_HEALTH"
)

// Subscription represents a client subscription
type Subscription struct {
	ID           string
	Type         SubscriptionType
	Filters      map[string]interface{}
	Connection   *websocket.Conn
	CreatedAt    time.Time
	LastActivity time.Time
}

// SubscriptionManager manages GraphQL subscriptions
type SubscriptionManager struct {
	mu             sync.RWMutex
	subscriptions  map[string]*Subscription
	upgrader       websocket.Upgrader
	signalDetector *analytics.SignalDetector
	trendAnalyzer  *analytics.TrendAnalyzer

	// Channels for broadcasting updates
	alertChan  chan *SafetyAlertPayload
	signalChan chan *SignalDetectionPayload
	overrideChan chan *OverridePayload
	healthChan chan *ServiceHealthPayload

	// Health monitoring
	serviceHealth *ServiceHealth
}

// SafetyAlertPayload represents a safety alert subscription payload
type SafetyAlertPayload struct {
	AlertID     string                 `json:"alertId"`
	PatientID   string                 `json:"patientId"`
	DrugCode    string                 `json:"drugCode"`
	AlertType   string                 `json:"alertType"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SignalDetectionPayload represents a signal detection subscription payload
type SignalDetectionPayload struct {
	AnalysisID      string                   `json:"analysisId"`
	DrugCode        string                   `json:"drugCode"`
	SignalType      string                   `json:"signalType"`
	SignalStrength  float64                  `json:"signalStrength"`
	PValue          float64                  `json:"pValue"`
	DetectionMethod string                   `json:"detectionMethod"`
	Timestamp       time.Time                `json:"timestamp"`
	Summary         *analytics.StatisticalSummary `json:"summary,omitempty"`
}

// OverridePayload represents an override subscription payload
type OverridePayload struct {
	OverrideID  string    `json:"overrideId"`
	AlertID     string    `json:"alertId"`
	PatientID   string    `json:"patientId"`
	Status      string    `json:"status"`
	RequestedBy string    `json:"requestedBy"`
	ApprovedBy  string    `json:"approvedBy,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ServiceHealthPayload represents a service health subscription payload
type ServiceHealthPayload struct {
	ServiceName string    `json:"serviceName"`
	Status      string    `json:"status"`
	Uptime      int64     `json:"uptime"`
	Metrics     *ServiceMetricsPayload `json:"metrics"`
	Timestamp   time.Time `json:"timestamp"`
}

// ServiceMetricsPayload contains service performance metrics
type ServiceMetricsPayload struct {
	TotalEvaluations   int64   `json:"totalEvaluations"`
	AverageLatencyMs   float64 `json:"averageLatencyMs"`
	AlertsGenerated    int64   `json:"alertsGenerated"`
	ActiveConnections  int64   `json:"activeConnections"`
	ErrorRate          float64 `json:"errorRate"`
}

// ServiceHealth tracks overall service health
type ServiceHealth struct {
	mu            sync.RWMutex
	Status        string
	StartTime     time.Time
	TotalEvaluations int64
	ErrorCount    int64
	LastError     *string
	LastErrorTime *time.Time
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(signalDetector *analytics.SignalDetector, trendAnalyzer *analytics.TrendAnalyzer) *SubscriptionManager {
	sm := &SubscriptionManager{
		subscriptions:  make(map[string]*Subscription),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		signalDetector: signalDetector,
		trendAnalyzer:  trendAnalyzer,
		alertChan:      make(chan *SafetyAlertPayload, 100),
		signalChan:     make(chan *SignalDetectionPayload, 100),
		overrideChan:   make(chan *OverridePayload, 100),
		healthChan:     make(chan *ServiceHealthPayload, 10),
		serviceHealth: &ServiceHealth{
			Status:    "HEALTHY",
			StartTime: time.Now(),
		},
	}

	// Start broadcast workers
	go sm.broadcastWorker()
	go sm.healthMonitorWorker()

	return sm
}

// HandleWebSocket handles WebSocket connections for GraphQL subscriptions
func (sm *SubscriptionManager) HandleWebSocket(c *gin.Context) {
	// Upgrade to WebSocket
	conn, err := sm.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create subscription
	subID := uuid.New().String()
	subscription := &Subscription{
		ID:           subID,
		Connection:   conn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Wait for subscription message
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("WebSocket read error: %v", err)
		return
	}

	// Parse subscription request
	var subRequest struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := json.Unmarshal(message, &subRequest); err != nil {
		conn.WriteJSON(map[string]string{"error": "invalid subscription request"})
		return
	}

	// Determine subscription type
	switch subRequest.Type {
	case "safetyAlertStream":
		subscription.Type = SubTypeSafetyAlerts
	case "signalDetectionUpdates":
		subscription.Type = SubTypeSignalDetection
	case "overrideUpdates":
		subscription.Type = SubTypeOverrides
	case "serviceHealthUpdates":
		subscription.Type = SubTypeServiceHealth
	default:
		conn.WriteJSON(map[string]string{"error": "unknown subscription type"})
		return
	}

	subscription.Filters = subRequest.Payload

	// Register subscription
	sm.mu.Lock()
	sm.subscriptions[subID] = subscription
	sm.mu.Unlock()

	// Send acknowledgment
	conn.WriteJSON(map[string]interface{}{
		"type":           "subscription_ack",
		"subscriptionId": subID,
	})

	log.Printf("New subscription: %s (%s)", subID, subscription.Type)

	// Keep connection alive and handle messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket closed: %s", subID)
			break
		}

		// Handle ping/pong or control messages
		subscription.LastActivity = time.Now()

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg["type"] == "ping" {
				conn.WriteJSON(map[string]string{"type": "pong"})
			} else if msg["type"] == "unsubscribe" {
				break
			}
		}
	}

	// Cleanup
	sm.mu.Lock()
	delete(sm.subscriptions, subID)
	sm.mu.Unlock()
}

// broadcastWorker handles broadcasting updates to subscribers
func (sm *SubscriptionManager) broadcastWorker() {
	for {
		select {
		case alert := <-sm.alertChan:
			sm.broadcastToType(SubTypeSafetyAlerts, map[string]interface{}{
				"type": "safetyAlert",
				"data": alert,
			}, func(sub *Subscription, data interface{}) bool {
				return sm.matchesSafetyAlertFilter(sub, alert)
			})

		case signal := <-sm.signalChan:
			sm.broadcastToType(SubTypeSignalDetection, map[string]interface{}{
				"type": "signalDetection",
				"data": signal,
			}, nil)

		case override := <-sm.overrideChan:
			sm.broadcastToType(SubTypeOverrides, map[string]interface{}{
				"type": "overrideUpdate",
				"data": override,
			}, nil)

		case health := <-sm.healthChan:
			sm.broadcastToType(SubTypeServiceHealth, map[string]interface{}{
				"type": "serviceHealth",
				"data": health,
			}, nil)
		}
	}
}

// broadcastToType broadcasts a message to all subscribers of a specific type
func (sm *SubscriptionManager) broadcastToType(subType SubscriptionType, message interface{}, filter func(*Subscription, interface{}) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, sub := range sm.subscriptions {
		if sub.Type != subType {
			continue
		}

		// Apply filter if provided
		if filter != nil && !filter(sub, message) {
			continue
		}

		// Send message
		if err := sub.Connection.WriteJSON(message); err != nil {
			log.Printf("Failed to send to subscription %s: %v", sub.ID, err)
		}
	}
}

// matchesSafetyAlertFilter checks if an alert matches subscription filters
func (sm *SubscriptionManager) matchesSafetyAlertFilter(sub *Subscription, alert *SafetyAlertPayload) bool {
	if sub.Filters == nil {
		return true
	}

	// Check patient ID filter
	if patientIDs, ok := sub.Filters["patientIds"].([]interface{}); ok && len(patientIDs) > 0 {
		matched := false
		for _, id := range patientIDs {
			if fmt.Sprintf("%v", id) == alert.PatientID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check severity filter
	if severities, ok := sub.Filters["severities"].([]interface{}); ok && len(severities) > 0 {
		matched := false
		for _, sev := range severities {
			if fmt.Sprintf("%v", sev) == alert.Severity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check drug code filter
	if drugCodes, ok := sub.Filters["drugCodes"].([]interface{}); ok && len(drugCodes) > 0 {
		matched := false
		for _, code := range drugCodes {
			if fmt.Sprintf("%v", code) == alert.DrugCode {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// healthMonitorWorker periodically broadcasts health updates
func (sm *SubscriptionManager) healthMonitorWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.RLock()
		activeConnections := int64(len(sm.subscriptions))
		sm.mu.RUnlock()

		sm.serviceHealth.mu.RLock()
		errorRate := float64(0)
		if sm.serviceHealth.TotalEvaluations > 0 {
			errorRate = float64(sm.serviceHealth.ErrorCount) / float64(sm.serviceHealth.TotalEvaluations) * 100
		}

		payload := &ServiceHealthPayload{
			ServiceName: "kb4-patient-safety",
			Status:      sm.serviceHealth.Status,
			Uptime:      int64(time.Since(sm.serviceHealth.StartTime).Seconds()),
			Metrics: &ServiceMetricsPayload{
				TotalEvaluations:  sm.serviceHealth.TotalEvaluations,
				AverageLatencyMs:  0, // Would need to track this
				AlertsGenerated:   0, // Would need to track this
				ActiveConnections: activeConnections,
				ErrorRate:         errorRate,
			},
			Timestamp: time.Now(),
		}
		sm.serviceHealth.mu.RUnlock()

		// Send to channel (non-blocking)
		select {
		case sm.healthChan <- payload:
		default:
			// Channel full, skip
		}
	}
}

// PublishSafetyAlert publishes a safety alert to subscribers
func (sm *SubscriptionManager) PublishSafetyAlert(alert *SafetyAlertPayload) {
	select {
	case sm.alertChan <- alert:
	default:
		log.Println("Alert channel full, dropping alert")
	}
}

// PublishSignalDetection publishes signal detection results to subscribers
func (sm *SubscriptionManager) PublishSignalDetection(signal *SignalDetectionPayload) {
	select {
	case sm.signalChan <- signal:
	default:
		log.Println("Signal channel full, dropping signal")
	}
}

// PublishOverrideUpdate publishes override status updates to subscribers
func (sm *SubscriptionManager) PublishOverrideUpdate(override *OverridePayload) {
	select {
	case sm.overrideChan <- override:
	default:
		log.Println("Override channel full, dropping update")
	}
}

// UpdateServiceHealth updates the service health status
func (sm *SubscriptionManager) UpdateServiceHealth(status string, err error) {
	sm.serviceHealth.mu.Lock()
	defer sm.serviceHealth.mu.Unlock()

	sm.serviceHealth.Status = status
	sm.serviceHealth.TotalEvaluations++

	if err != nil {
		sm.serviceHealth.ErrorCount++
		errStr := err.Error()
		sm.serviceHealth.LastError = &errStr
		now := time.Now()
		sm.serviceHealth.LastErrorTime = &now
	}
}

// GetActiveSubscriptions returns the count of active subscriptions by type
func (sm *SubscriptionManager) GetActiveSubscriptions() map[string]int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	counts := make(map[string]int)
	for _, sub := range sm.subscriptions {
		counts[string(sub.Type)]++
	}
	return counts
}

// CleanupStaleConnections removes connections that haven't been active recently
func (sm *SubscriptionManager) CleanupStaleConnections(maxAge time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, sub := range sm.subscriptions {
		if sub.LastActivity.Before(cutoff) {
			sub.Connection.Close()
			delete(sm.subscriptions, id)
			removed++
		}
	}

	return removed
}

// =============================================================================
// GraphQL Query Resolvers
// =============================================================================

// SafetyDashboardResolver resolves the safetyDashboard query
type SafetyDashboardResolver struct {
	sm *SubscriptionManager
}

// SafetyDashboard represents the dashboard data
type SafetyDashboard struct {
	TotalAlerts        int64                    `json:"totalAlerts"`
	CriticalAlerts     int64                    `json:"criticalAlerts"`
	PendingOverrides   int64                    `json:"pendingOverrides"`
	ActiveSignals      int64                    `json:"activeSignals"`
	AlertsByType       map[string]int64         `json:"alertsByType"`
	AlertsBySeverity   map[string]int64         `json:"alertsBySeverity"`
	TrendSummary       *analytics.TrendSummary  `json:"trendSummary,omitempty"`
	LastUpdated        time.Time                `json:"lastUpdated"`
}

// ResolveSafetyDashboard resolves the safety dashboard query
func (r *SafetyDashboardResolver) ResolveSafetyDashboard(ctx context.Context, timeRange string) (*SafetyDashboard, error) {
	// This would typically query the database
	// For now, return sample data structure
	return &SafetyDashboard{
		TotalAlerts:      0,
		CriticalAlerts:   0,
		PendingOverrides: 0,
		ActiveSignals:    0,
		AlertsByType:     make(map[string]int64),
		AlertsBySeverity: make(map[string]int64),
		LastUpdated:      time.Now(),
	}, nil
}

// DrugSafetyProfile represents a drug's safety profile
type DrugSafetyProfile struct {
	DrugCode          string                 `json:"drugCode"`
	DrugName          string                 `json:"drugName"`
	BlackBoxWarnings  []string               `json:"blackBoxWarnings"`
	Contraindications []string               `json:"contraindications"`
	HighAlertStatus   bool                   `json:"highAlertStatus"`
	BeersListStatus   bool                   `json:"beersListStatus"`
	ACBScore          int                    `json:"acbScore"`
	DoseLimit         *DoseLimitInfo         `json:"doseLimit,omitempty"`
	PregnancyCategory string                 `json:"pregnancyCategory,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// DoseLimitInfo represents dose limit information
type DoseLimitInfo struct {
	MaxSingleDose float64 `json:"maxSingleDose"`
	MaxDailyDose  float64 `json:"maxDailyDose"`
	Unit          string  `json:"unit"`
}

// ResolveDrugSafetyProfile resolves a drug safety profile query
func (sm *SubscriptionManager) ResolveDrugSafetyProfile(ctx context.Context, rxnorm string) (*DrugSafetyProfile, error) {
	// This would query the safety data store
	return &DrugSafetyProfile{
		DrugCode: rxnorm,
		DrugName: "Unknown",
		Metadata: make(map[string]interface{}),
	}, nil
}

// ClinicalInsights represents clinical insights
type ClinicalInsights struct {
	Recommendations    []string               `json:"recommendations"`
	RiskFactors        []string               `json:"riskFactors"`
	PopulationAnalysis *PopulationAnalysis    `json:"populationAnalysis,omitempty"`
	GeneratedAt        time.Time              `json:"generatedAt"`
}

// PopulationAnalysis represents population-level analysis
type PopulationAnalysis struct {
	TotalPatients       int64            `json:"totalPatients"`
	AtRiskPatients      int64            `json:"atRiskPatients"`
	AlertDistribution   map[string]int64 `json:"alertDistribution"`
	CommonInterventions []string         `json:"commonInterventions"`
}

// ResolveClinicalInsights resolves clinical insights query
func (sm *SubscriptionManager) ResolveClinicalInsights(ctx context.Context, drugCodes []string, patientPopulation string) (*ClinicalInsights, error) {
	return &ClinicalInsights{
		Recommendations: []string{},
		RiskFactors:     []string{},
		GeneratedAt:     time.Now(),
	}, nil
}

// OverrideAnalytics represents override analytics
type OverrideAnalytics struct {
	TotalRequested      int64              `json:"totalRequested"`
	TotalApproved       int64              `json:"totalApproved"`
	TotalRejected       int64              `json:"totalRejected"`
	TotalExpired        int64              `json:"totalExpired"`
	ApprovalRate        float64            `json:"approvalRate"`
	AverageTimeToApprove int64             `json:"averageTimeToApprove"` // in seconds
	ByReason            map[string]int64   `json:"byReason"`
	BySeverity          map[string]int64   `json:"bySeverity"`
}

// ResolveOverrideAnalytics resolves override analytics query
func (sm *SubscriptionManager) ResolveOverrideAnalytics(ctx context.Context, timeRange string) (*OverrideAnalytics, error) {
	return &OverrideAnalytics{
		TotalRequested: 0,
		TotalApproved:  0,
		TotalRejected:  0,
		TotalExpired:   0,
		ApprovalRate:   0,
		ByReason:       make(map[string]int64),
		BySeverity:     make(map[string]int64),
	}, nil
}

// RegisterGraphQLRoutes registers GraphQL-related routes
func (sm *SubscriptionManager) RegisterGraphQLRoutes(router *gin.Engine) {
	// WebSocket endpoint for subscriptions
	router.GET("/graphql/subscriptions", sm.HandleWebSocket)

	// GraphQL queries endpoint (simplified)
	router.POST("/graphql", sm.handleGraphQLQuery)

	// Dashboard endpoint
	router.GET("/v1/analytics/dashboard", sm.handleDashboard)

	// Drug safety profile endpoint
	router.GET("/v1/analytics/drug/:rxnorm", sm.handleDrugProfile)

	// Signal detection endpoint
	router.POST("/v1/analytics/signals", sm.handleSignalDetection)

	// Trend analysis endpoint
	router.POST("/v1/analytics/trends", sm.handleTrendAnalysis)

	// Override analytics endpoint
	router.GET("/v1/analytics/overrides", sm.handleOverrideAnalytics)
}

func (sm *SubscriptionManager) handleGraphQLQuery(c *gin.Context) {
	var req struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simplified query handling - in production, use a proper GraphQL library
	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"message": "GraphQL queries supported. Use specific REST endpoints for now.",
		},
	})
}

func (sm *SubscriptionManager) handleDashboard(c *gin.Context) {
	resolver := &SafetyDashboardResolver{sm: sm}
	timeRange := c.Query("timeRange")
	if timeRange == "" {
		timeRange = "24h"
	}

	dashboard, err := resolver.ResolveSafetyDashboard(c.Request.Context(), timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (sm *SubscriptionManager) handleDrugProfile(c *gin.Context) {
	rxnorm := c.Param("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rxnorm is required"})
		return
	}

	profile, err := sm.ResolveDrugSafetyProfile(c.Request.Context(), rxnorm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (sm *SubscriptionManager) handleSignalDetection(c *gin.Context) {
	var req analytics.SignalDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := sm.signalDetector.DetectSignals(&req)
	c.JSON(http.StatusOK, response)
}

func (sm *SubscriptionManager) handleTrendAnalysis(c *gin.Context) {
	var req analytics.TrendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := sm.trendAnalyzer.AnalyzeTrends(&req)
	c.JSON(http.StatusOK, response)
}

func (sm *SubscriptionManager) handleOverrideAnalytics(c *gin.Context) {
	timeRange := c.Query("timeRange")
	if timeRange == "" {
		timeRange = "7d"
	}

	analytics, err := sm.ResolveOverrideAnalytics(c.Request.Context(), timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}
