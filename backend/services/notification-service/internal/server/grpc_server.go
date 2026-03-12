package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/cardiofit/notification-service/internal/database"
	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/escalation"
	"github.com/cardiofit/notification-service/internal/models"
	pb "github.com/cardiofit/notification-service/pkg/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCServer handles gRPC requests for the notification service
type GRPCServer struct {
	pb.UnimplementedNotificationServiceServer
	server           *grpc.Server
	deliveryManager  *delivery.Manager
	escalationEngine *escalation.Engine
	db               *database.PostgresDB
	redis            *database.RedisClient
	logger           *zap.Logger
	port             int
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer(
	deliveryManager *delivery.Manager,
	escalationEngine *escalation.Engine,
	db *database.PostgresDB,
	redis *database.RedisClient,
	logger *zap.Logger,
	port int,
) *GRPCServer {
	s := &GRPCServer{
		deliveryManager:  deliveryManager,
		escalationEngine: escalationEngine,
		db:               db,
		redis:            redis,
		logger:           logger,
		port:             port,
	}

	// Create gRPC server with interceptors
	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			UnaryRecoveryInterceptor(logger),
			UnaryLoggingInterceptor(logger),
			UnaryMetricsInterceptor(),
			UnaryAuthInterceptor(),
		),
	)

	// Register service
	pb.RegisterNotificationServiceServer(s.server, s)

	// Register reflection service for debugging
	reflection.Register(s.server)

	return s
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	s.logger.Info("Starting gRPC server", zap.Int("port", s.port))

	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

// Shutdown performs graceful shutdown of the gRPC server
func (s *GRPCServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down gRPC server")
	s.server.GracefulStop()
	return nil
}

// RPC method implementations

// SendNotification sends a notification to specified users
func (s *GRPCServer) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	s.logger.Info("SendNotification called",
		zap.String("patient_id", req.PatientId),
		zap.String("priority", req.Priority),
		zap.String("type", req.Type),
	)

	// Validate request
	if req.PatientId == "" || req.Priority == "" || req.Message == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing required fields")
	}

	if len(req.Recipients) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one recipient is required")
	}

	// Create alert model
	alert := &models.ClinicalAlert{
		ID:          uuid.New().String(),
		PatientID:   req.PatientId,
		Priority:    req.Priority,
		Type:        req.Type,
		Title:       req.Title,
		Message:     req.Message,
		Recipients:  req.Recipients,
		Timestamp:   time.Now(),
		RequiresAck: req.RequiresAck,
	}

	// Convert metadata
	if req.Metadata != nil {
		alert.Metadata = make(map[string]interface{})
		for k, v := range req.Metadata {
			alert.Metadata[k] = v
		}
	}

	// Store notification in database
	notificationID, err := s.createNotification(ctx, alert)
	if err != nil {
		s.logger.Error("Failed to create notification", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create notification: %v", err)
	}

	// Create routing decision
	routingDecision := &models.RoutingDecision{
		AlertID:       alert.ID,
		Channel:       "email", // Default channel
		Recipients:    req.Recipients,
		Content:       req.Message,
		Priority:      req.Priority,
		Metadata:      alert.Metadata,
		RetryAttempts: 3,
	}

	// Deliver notification asynchronously
	go func() {
		deliveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := s.deliveryManager.Deliver(deliveryCtx, routingDecision)
		if err != nil {
			s.logger.Error("Failed to deliver notification",
				zap.String("notification_id", notificationID),
				zap.Error(err),
			)
			RecordNotificationDelivery(routingDecision.Channel, "failed")
			return
		}

		RecordNotificationDelivery(routingDecision.Channel, "success")

		// Update notification status
		status := models.StatusSent
		if result.Success {
			status = models.StatusDelivered
		} else {
			status = models.StatusFailed
		}

		s.updateNotificationStatusGRPC(context.Background(), notificationID, status, result.MessageID, result.Error)
	}()

	response := &pb.SendNotificationResponse{
		NotificationId: notificationID,
		Status:         string(models.StatusPending),
		CreatedAt:      timestamppb.New(time.Now()),
	}

	return response, nil
}

// GetDeliveryStatus retrieves the delivery status of a notification
func (s *GRPCServer) GetDeliveryStatus(ctx context.Context, req *pb.GetDeliveryStatusRequest) (*pb.GetDeliveryStatusResponse, error) {
	s.logger.Info("GetDeliveryStatus called", zap.String("notification_id", req.NotificationId))

	if req.NotificationId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "notification_id is required")
	}

	// Query notification from database
	notification, err := s.getNotificationByID(ctx, req.NotificationId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "notification not found")
		}
		s.logger.Error("Failed to get notification", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to retrieve notification")
	}

	// Query delivery attempts
	attempts, err := s.getDeliveryAttempts(ctx, req.NotificationId)
	if err != nil {
		s.logger.Error("Failed to get delivery attempts", zap.Error(err))
		// Continue without attempts - not critical
	}

	response := &pb.GetDeliveryStatusResponse{
		NotificationId: notification.ID,
		Status:         string(notification.Status),
		Channel:        string(notification.Channel),
		Attempts:       attempts,
		LastUpdated:    timestamppb.New(notification.CreatedAt),
	}

	return response, nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *GRPCServer) AcknowledgeAlert(ctx context.Context, req *pb.AcknowledgeAlertRequest) (*pb.AcknowledgeAlertResponse, error) {
	s.logger.Info("AcknowledgeAlert called",
		zap.String("alert_id", req.AlertId),
		zap.String("user_id", req.UserId),
	)

	if req.AlertId == "" || req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "alert_id and user_id are required")
	}

	acknowledgedAt := time.Now()
	if req.AcknowledgedAt != nil {
		acknowledgedAt = req.AcknowledgedAt.AsTime()
	}

	// Update notification status
	err := s.acknowledgeAlert(ctx, req.AlertId, req.UserId, req.NotificationId, req.AcknowledgmentNote, acknowledgedAt)
	if err != nil {
		s.logger.Error("Failed to acknowledge alert", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to acknowledge alert: %v", err)
	}

	response := &pb.AcknowledgeAlertResponse{
		Success:        true,
		Message:        "Alert acknowledged successfully",
		AcknowledgedAt: timestamppb.New(acknowledgedAt),
	}

	return response, nil
}

// UpdatePreferences updates user notification preferences
func (s *GRPCServer) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.UpdatePreferencesResponse, error) {
	s.logger.Info("UpdatePreferences called", zap.String("user_id", req.UserId))

	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	// Create preferences model
	preferences := &models.NotificationPreference{
		UserID:            req.UserId,
		PreferredChannels: req.PreferredChannels,
		QuietHoursStart:   req.QuietHoursStart,
		QuietHoursEnd:     req.QuietHoursEnd,
		EnabledAlertTypes: req.EnabledAlertTypes,
		Priority:          req.PriorityThreshold,
	}

	// Store preferences in database
	err := s.updateUserPreferences(ctx, preferences)
	if err != nil {
		s.logger.Error("Failed to update preferences", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update preferences: %v", err)
	}

	response := &pb.UpdatePreferencesResponse{
		Success: true,
		Message: "Preferences updated successfully",
	}

	return response, nil
}

// GetPreferences retrieves user notification preferences
func (s *GRPCServer) GetPreferences(ctx context.Context, req *pb.GetPreferencesRequest) (*pb.GetPreferencesResponse, error) {
	s.logger.Info("GetPreferences called", zap.String("user_id", req.UserId))

	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	// Query preferences from database
	preferences, err := s.getUserPreferences(ctx, req.UserId)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			return &pb.GetPreferencesResponse{
				UserId:            req.UserId,
				PreferredChannels: []string{"email", "push"},
				QuietHoursStart:   "22:00",
				QuietHoursEnd:     "07:00",
				EnabledAlertTypes: []string{},
				PriorityThreshold: "moderate",
			}, nil
		}
		s.logger.Error("Failed to get preferences", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to retrieve preferences")
	}

	response := &pb.GetPreferencesResponse{
		UserId:            preferences.UserID,
		PreferredChannels: preferences.PreferredChannels,
		QuietHoursStart:   preferences.QuietHoursStart,
		QuietHoursEnd:     preferences.QuietHoursEnd,
		EnabledAlertTypes: preferences.EnabledAlertTypes,
		PriorityThreshold: preferences.Priority,
	}

	return response, nil
}

// Database helper methods

// createNotification creates a new notification record
func (s *GRPCServer) createNotification(ctx context.Context, alert *models.ClinicalAlert) (string, error) {
	notificationID := uuid.New().String()

	query := `
		INSERT INTO notifications (id, alert_id, user_id, channel, priority, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	// Use first recipient as user_id (in production, create multiple notifications)
	userID := alert.Recipients[0]
	channel := "email"
	priority := 3 // Default medium priority

	_, err := s.db.ExecContext(ctx, query,
		notificationID,
		alert.ID,
		userID,
		channel,
		priority,
		alert.Message,
		models.StatusPending,
	)

	return notificationID, err
}

// updateNotificationStatusGRPC updates notification status and metadata
func (s *GRPCServer) updateNotificationStatusGRPC(ctx context.Context, notificationID string, notifStatus models.NotificationStatus, externalID, errorMsg string) error {
	query := `
		UPDATE notifications
		SET status = $1, external_id = $2, error_message = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err := s.db.ExecContext(ctx, query, notifStatus, externalID, errorMsg, notificationID)
	return err
}

// getNotificationByID retrieves a notification by ID
func (s *GRPCServer) getNotificationByID(ctx context.Context, notificationID string) (*models.Notification, error) {
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

// getDeliveryAttempts retrieves delivery attempts for a notification
func (s *GRPCServer) getDeliveryAttempts(ctx context.Context, notificationID string) ([]*pb.DeliveryAttempt, error) {
	query := `
		SELECT channel, success, error, attempted_at
		FROM delivery_attempts
		WHERE notification_id = $1
		ORDER BY attempted_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, notificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*pb.DeliveryAttempt

	for rows.Next() {
		var channel, errorMsg string
		var success bool
		var attemptedAt time.Time

		err := rows.Scan(&channel, &success, &errorMsg, &attemptedAt)
		if err != nil {
			return nil, err
		}

		attempts = append(attempts, &pb.DeliveryAttempt{
			Channel:     channel,
			Success:     success,
			Error:       errorMsg,
			AttemptedAt: timestamppb.New(attemptedAt),
		})
	}

	return attempts, rows.Err()
}

// acknowledgeAlert updates alert acknowledgment status
func (s *GRPCServer) acknowledgeAlert(ctx context.Context, alertID, userID, notificationID, note string, acknowledgedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update notification status
	query1 := `
		UPDATE notifications
		SET status = $1, acknowledged_at = $2, updated_at = NOW()
		WHERE alert_id = $3 AND user_id = $4
	`

	_, err = tx.ExecContext(ctx, query1, models.StatusAcknowledged, acknowledgedAt, alertID, userID)
	if err != nil {
		return err
	}

	// Insert acknowledgment record
	query2 := `
		INSERT INTO alert_acknowledgments (id, alert_id, user_id, notification_id, note, acknowledged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	ackID := uuid.New().String()
	_, err = tx.ExecContext(ctx, query2, ackID, alertID, userID, notificationID, note, acknowledgedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// updateUserPreferences updates user notification preferences
func (s *GRPCServer) updateUserPreferences(ctx context.Context, prefs *models.NotificationPreference) error {
	query := `
		INSERT INTO user_preferences (user_id, preferred_channels, quiet_hours_start, quiet_hours_end, enabled_alert_types, priority_threshold, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET preferred_channels = $2, quiet_hours_start = $3, quiet_hours_end = $4,
		    enabled_alert_types = $5, priority_threshold = $6, updated_at = NOW()
	`

	channelsJSON, _ := json.Marshal(prefs.PreferredChannels)
	alertTypesJSON, _ := json.Marshal(prefs.EnabledAlertTypes)

	_, err := s.db.ExecContext(ctx, query,
		prefs.UserID,
		channelsJSON,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		alertTypesJSON,
		prefs.Priority,
	)

	return err
}

// getUserPreferences retrieves user notification preferences
func (s *GRPCServer) getUserPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	query := `
		SELECT user_id, preferred_channels, quiet_hours_start, quiet_hours_end, enabled_alert_types, priority_threshold
		FROM user_preferences
		WHERE user_id = $1
	`

	prefs := &models.NotificationPreference{}
	var channelsJSON, alertTypesJSON string

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.UserID,
		&channelsJSON,
		&prefs.QuietHoursStart,
		&prefs.QuietHoursEnd,
		&alertTypesJSON,
		&prefs.Priority,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON arrays
	json.Unmarshal([]byte(channelsJSON), &prefs.PreferredChannels)
	json.Unmarshal([]byte(alertTypesJSON), &prefs.EnabledAlertTypes)

	return prefs, nil
}
