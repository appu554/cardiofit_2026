// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/models"
)

// NotificationService handles sending notifications (stub implementation)
// In production, this would integrate with email, SMS, push notification services
type NotificationService struct {
	log     *logrus.Entry
	enabled bool
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(log *logrus.Entry) *NotificationService {
	return &NotificationService{
		log:     log.WithField("service", "notification"),
		enabled: true, // In production, this would be configurable
	}
}

// Send sends a notification through configured channels
func (s *NotificationService) Send(ctx context.Context, req *models.NotificationRequest) *models.NotificationResponse {
	if !s.enabled {
		s.log.Debug("Notification service disabled, skipping send")
		return &models.NotificationResponse{
			Success: true,
			Results: []models.NotificationResult{},
		}
	}

	var results []models.NotificationResult
	allSuccess := true

	// Process each configured channel
	for _, channel := range req.Channels {
		result := s.sendToChannel(ctx, req, channel)
		results = append(results, result)
		if !result.Success {
			allSuccess = false
		}
	}

	response := &models.NotificationResponse{
		Success: allSuccess,
		Results: results,
	}

	// Log notification
	s.log.WithFields(logrus.Fields{
		"recipient_id":   req.RecipientID,
		"recipient_type": req.RecipientType,
		"priority":       req.Priority,
		"channels":       len(req.Channels),
		"success":        allSuccess,
	}).Info("Notification sent")

	return response
}

// sendToChannel sends a notification to a specific channel
func (s *NotificationService) sendToChannel(ctx context.Context, req *models.NotificationRequest, channel models.NotificationChannel) models.NotificationResult {
	now := time.Now().UTC()

	// In production, this would dispatch to actual notification providers
	// For now, we just log the notification
	result := models.NotificationResult{
		Channel:   channel,
		Success:   true,
		SentAt:    &now,
		MessageID: generateMessageID(channel),
	}

	switch channel {
	case models.NotificationChannelInApp:
		s.log.WithFields(logrus.Fields{
			"channel":      "in_app",
			"recipient":    req.RecipientID,
			"title":        req.Title,
			"priority":     req.Priority,
		}).Debug("In-app notification logged")

	case models.NotificationChannelEmail:
		s.log.WithFields(logrus.Fields{
			"channel":      "email",
			"recipient":    req.RecipientID,
			"title":        req.Title,
			"priority":     req.Priority,
		}).Debug("Email notification logged (stub)")

	case models.NotificationChannelSMS:
		s.log.WithFields(logrus.Fields{
			"channel":      "sms",
			"recipient":    req.RecipientID,
			"message_len":  len(req.Message),
			"priority":     req.Priority,
		}).Debug("SMS notification logged (stub)")

	case models.NotificationChannelPush:
		s.log.WithFields(logrus.Fields{
			"channel":      "push",
			"recipient":    req.RecipientID,
			"title":        req.Title,
			"priority":     req.Priority,
		}).Debug("Push notification logged (stub)")

	case models.NotificationChannelPager:
		s.log.WithFields(logrus.Fields{
			"channel":      "pager",
			"recipient":    req.RecipientID,
			"priority":     req.Priority,
		}).Debug("Pager notification logged (stub)")

	default:
		result.Success = false
		result.Error = "unknown channel: " + string(channel)
	}

	return result
}

// SendTaskAssigned sends a notification for task assignment
func (s *NotificationService) SendTaskAssigned(ctx context.Context, task *models.Task, assigneeName string) *models.NotificationResponse {
	req := models.BuildTaskAssignedNotification(task, assigneeName)
	return s.Send(ctx, &req)
}

// SendTaskEscalated sends a notification for task escalation
func (s *NotificationService) SendTaskEscalated(ctx context.Context, task *models.Task, escalation *models.Escalation, recipientID string, recipientName string) *models.NotificationResponse {
	req := models.BuildEscalationNotification(task, escalation, recipientID, recipientName)
	return s.Send(ctx, &req)
}

// SendTaskDueSoon sends a notification for tasks due soon
func (s *NotificationService) SendTaskDueSoon(ctx context.Context, task *models.Task, minutesRemaining int) *models.NotificationResponse {
	req := models.BuildDueSoonNotification(task, minutesRemaining)
	return s.Send(ctx, &req)
}

// SendBulkNotification sends notifications to multiple recipients
func (s *NotificationService) SendBulkNotification(ctx context.Context, requests []*models.NotificationRequest) []*models.NotificationResponse {
	var responses []*models.NotificationResponse

	for _, req := range requests {
		resp := s.Send(ctx, req)
		responses = append(responses, resp)
	}

	return responses
}

// generateMessageID generates a unique message ID for tracking
func generateMessageID(channel models.NotificationChannel) string {
	return string(channel) + "-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string for IDs
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalSent     int64                   `json:"total_sent"`
	SuccessCount  int64                   `json:"success_count"`
	FailureCount  int64                   `json:"failure_count"`
	ByChannel     map[string]int64        `json:"by_channel"`
	ByPriority    map[string]int64        `json:"by_priority"`
}

// GetStats returns notification statistics (stub)
func (s *NotificationService) GetStats(ctx context.Context) *NotificationStats {
	// In production, this would query a database or metrics store
	return &NotificationStats{
		TotalSent:    0,
		SuccessCount: 0,
		FailureCount: 0,
		ByChannel:    make(map[string]int64),
		ByPriority:   make(map[string]int64),
	}
}
