package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cardiofit/notification-service/internal/database"
	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/escalation"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer handles HTTP requests for the notification service
type HTTPServer struct {
	router          *http.ServeMux
	server          *http.Server
	deliveryManager *delivery.Manager
	escalationEngine *escalation.Engine
	db              *database.PostgresDB
	redis           *database.RedisClient
	logger          *zap.Logger
	port            int
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(
	deliveryManager *delivery.Manager,
	escalationEngine *escalation.Engine,
	db *database.PostgresDB,
	redis *database.RedisClient,
	logger *zap.Logger,
	port int,
) *HTTPServer {
	s := &HTTPServer{
		router:          http.NewServeMux(),
		deliveryManager: deliveryManager,
		escalationEngine: escalationEngine,
		db:              db,
		redis:           redis,
		logger:          logger,
		port:            port,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *HTTPServer) setupRoutes() {
	// Health and monitoring endpoints
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/ready", s.handleReady)
	s.router.Handle("/metrics", promhttp.Handler())

	// API endpoints
	s.router.HandleFunc("/api/v1/notifications/acknowledge", s.handleAcknowledge)
	s.router.HandleFunc("/api/v1/notifications/", s.handleGetNotification)
	s.router.HandleFunc("/api/v1/escalations/", s.handleGetEscalations)
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Apply middleware chain
	handler := s.router
	handler = applyMiddleware(handler,
		RecoveryMiddleware(s.logger),
		LoggingMiddleware(s.logger),
		MetricsMiddleware(),
		CORSMiddleware(),
		TimeoutMiddleware(30*time.Second),
	)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server", zap.Int("port", s.port))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// Shutdown performs graceful shutdown of the HTTP server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

// Health check handlers

// handleHealth is a liveness probe that returns 200 if the service is running
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service": "notification-service",
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleReady is a readiness probe that checks service dependencies
func (s *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	ready := true

	// Check PostgreSQL
	if err := s.db.Ping(ctx); err != nil {
		checks["postgres"] = "unhealthy: " + err.Error()
		ready = false
	} else {
		checks["postgres"] = "healthy"
	}

	// Check Redis
	if err := s.redis.Ping(ctx); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
		ready = false
	} else {
		checks["redis"] = "healthy"
	}

	// Check Kafka (simulated - would need actual Kafka health check)
	checks["kafka"] = "healthy"

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    checks,
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	s.writeJSON(w, status, response)
}

// API handlers

// handleAcknowledge handles POST /api/v1/notifications/acknowledge
func (s *HTTPServer) handleAcknowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AlertID          string `json:"alert_id"`
		NotificationID   string `json:"notification_id"`
		UserID           string `json:"user_id"`
		AcknowledgmentNote string `json:"acknowledgment_note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.AlertID == "" || req.UserID == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required fields: alert_id and user_id")
		return
	}

	// Update notification status in database
	acknowledgedAt := time.Now()
	err := s.updateNotificationStatus(r.Context(), req.NotificationID, models.StatusAcknowledged, &acknowledgedAt)
	if err != nil {
		s.logger.Error("Failed to update notification status",
			zap.String("notification_id", req.NotificationID),
			zap.Error(err),
		)
		s.writeError(w, http.StatusInternalServerError, "Failed to acknowledge notification")
		return
	}

	response := map[string]interface{}{
		"success":          true,
		"message":          "Alert acknowledged successfully",
		"alert_id":         req.AlertID,
		"notification_id":  req.NotificationID,
		"acknowledged_at":  acknowledgedAt.Format(time.RFC3339),
		"acknowledged_by":  req.UserID,
	}

	s.logger.Info("Alert acknowledged",
		zap.String("alert_id", req.AlertID),
		zap.String("user_id", req.UserID),
		zap.String("notification_id", req.NotificationID),
	)

	s.writeJSON(w, http.StatusOK, response)
}

// handleGetNotification handles GET /api/v1/notifications/{id}
func (s *HTTPServer) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract notification ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/")
	notificationID := strings.TrimSpace(path)

	if notificationID == "" {
		s.writeError(w, http.StatusBadRequest, "Missing notification ID")
		return
	}

	// Query notification from database
	notification, err := s.getNotification(r.Context(), notificationID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "Notification not found")
			return
		}
		s.logger.Error("Failed to get notification",
			zap.String("notification_id", notificationID),
			zap.Error(err),
		)
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve notification")
		return
	}

	s.writeJSON(w, http.StatusOK, notification)
}

// handleGetEscalations handles GET /api/v1/escalations/{alertId}
func (s *HTTPServer) handleGetEscalations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract alert ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/escalations/")
	alertID := strings.TrimSpace(path)

	if alertID == "" {
		s.writeError(w, http.StatusBadRequest, "Missing alert ID")
		return
	}

	// Query escalation history from database
	escalations, err := s.getEscalationHistory(r.Context(), alertID)
	if err != nil {
		s.logger.Error("Failed to get escalation history",
			zap.String("alert_id", alertID),
			zap.Error(err),
		)
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve escalation history")
		return
	}

	response := map[string]interface{}{
		"alert_id":    alertID,
		"escalations": escalations,
		"count":       len(escalations),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// Database helper methods

// updateNotificationStatus updates the status of a notification
func (s *HTTPServer) updateNotificationStatus(ctx context.Context, notificationID string, status models.NotificationStatus, acknowledgedAt *time.Time) error {
	query := `
		UPDATE notifications
		SET status = $1, acknowledged_at = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err := s.db.ExecContext(ctx, query, status, acknowledgedAt, notificationID)
	return err
}

// getNotification retrieves a notification by ID
func (s *HTTPServer) getNotification(ctx context.Context, notificationID string) (*models.Notification, error) {
	query := `
		SELECT id, alert_id, user_id, channel, priority, message, status,
		       retry_count, external_id, created_at, sent_at, delivered_at,
		       acknowledged_at, error_message
		FROM notifications
		WHERE id = $1
	`

	notification := &models.Notification{}
	var sentAt, deliveredAt, acknowledgedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, notificationID).Scan(
		&notification.ID,
		&notification.AlertID,
		&notification.UserID,
		&notification.Channel,
		&notification.Priority,
		&notification.Message,
		&notification.Status,
		&notification.RetryCount,
		&notification.ExternalID,
		&notification.CreatedAt,
		&sentAt,
		&deliveredAt,
		&acknowledgedAt,
		&notification.ErrorMessage,
	)

	if err != nil {
		return nil, err
	}

	if sentAt.Valid {
		notification.SentAt = &sentAt.Time
	}
	if deliveredAt.Valid {
		notification.DeliveredAt = &deliveredAt.Time
	}
	if acknowledgedAt.Valid {
		notification.AcknowledgedAt = &acknowledgedAt.Time
	}

	return notification, nil
}

// EscalationRecord represents an escalation event record
type EscalationRecord struct {
	ID            string    `json:"id"`
	AlertID       string    `json:"alert_id"`
	Level         int       `json:"level"`
	FromChannel   string    `json:"from_channel"`
	ToChannel     string    `json:"to_channel"`
	Reason        string    `json:"reason"`
	EscalatedAt   time.Time `json:"escalated_at"`
	TargetUsers   []string  `json:"target_users"`
	Acknowledged  bool      `json:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// getEscalationHistory retrieves escalation history for an alert
func (s *HTTPServer) getEscalationHistory(ctx context.Context, alertID string) ([]EscalationRecord, error) {
	query := `
		SELECT id, alert_id, level, from_channel, to_channel, reason,
		       escalated_at, target_users, acknowledged, acknowledged_at
		FROM escalations
		WHERE alert_id = $1
		ORDER BY escalated_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var escalations []EscalationRecord

	for rows.Next() {
		var e EscalationRecord
		var targetUsers string
		var acknowledgedAt sql.NullTime

		err := rows.Scan(
			&e.ID,
			&e.AlertID,
			&e.Level,
			&e.FromChannel,
			&e.ToChannel,
			&e.Reason,
			&e.EscalatedAt,
			&targetUsers,
			&e.Acknowledged,
			&acknowledgedAt,
		)

		if err != nil {
			return nil, err
		}

		// Parse target users JSON array
		if targetUsers != "" {
			json.Unmarshal([]byte(targetUsers), &e.TargetUsers)
		}

		if acknowledgedAt.Valid {
			e.AcknowledgedAt = &acknowledgedAt.Time
		}

		escalations = append(escalations, e)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return escalations, nil
}

// Helper methods for JSON responses

// writeJSON writes a JSON response
func (s *HTTPServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// writeError writes an error JSON response
func (s *HTTPServer) writeError(w http.ResponseWriter, status int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	s.writeJSON(w, status, response)
}

// applyMiddleware applies middleware in reverse order
func applyMiddleware(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
