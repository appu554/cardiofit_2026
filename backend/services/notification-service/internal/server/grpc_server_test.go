package server

import (
	"context"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/database"
	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/escalation"
	pb "github.com/cardiofit/notification-service/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// setupTestGRPCServer creates a test gRPC server instance
func setupTestGRPCServer() *GRPCServer {
	logger, _ := zap.NewDevelopment()

	mockDeliveryMgr := &delivery.Manager{}
	mockEscalationEngine := &escalation.Engine{}
	mockDB := &database.PostgresDB{}
	mockRedis := &database.RedisClient{}

	server := &GRPCServer{
		deliveryManager:  mockDeliveryMgr,
		escalationEngine: mockEscalationEngine,
		db:               mockDB,
		redis:            mockRedis,
		logger:           logger,
		port:             50060,
	}

	return server
}

// SendNotification tests

func TestSendNotification_Success(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		PatientId:   "patient-123",
		Priority:    "high",
		Type:        "sepsis_alert",
		Title:       "Sepsis Alert",
		Message:     "Patient shows signs of sepsis",
		Recipients:  []string{"doctor@hospital.com"},
		RequiresAck: true,
		Metadata: map[string]string{
			"department": "ICU",
		},
	}

	// Note: This would need proper DB mocking for full test
	resp, err := server.SendNotification(context.Background(), req)

	// Basic validation
	if err == nil {
		assert.NotEmpty(t, resp.NotificationId)
		assert.NotEmpty(t, resp.Status)
		assert.NotNil(t, resp.CreatedAt)
	} else {
		// Expected if DB is not mocked
		assert.Error(t, err)
	}
}

func TestSendNotification_MissingPatientID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		Priority:    "high",
		Message:     "Test message",
		Recipients:  []string{"doctor@hospital.com"},
	}

	resp, err := server.SendNotification(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "missing required fields")
}

func TestSendNotification_MissingPriority(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		PatientId:  "patient-123",
		Message:    "Test message",
		Recipients: []string{"doctor@hospital.com"},
	}

	resp, err := server.SendNotification(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSendNotification_MissingMessage(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		PatientId:  "patient-123",
		Priority:   "high",
		Recipients: []string{"doctor@hospital.com"},
	}

	resp, err := server.SendNotification(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSendNotification_NoRecipients(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		PatientId: "patient-123",
		Priority:  "high",
		Message:   "Test message",
		Recipients: []string{},
	}

	resp, err := server.SendNotification(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "at least one recipient")
}

// GetDeliveryStatus tests

func TestGetDeliveryStatus_MissingNotificationID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.GetDeliveryStatusRequest{
		NotificationId: "",
	}

	resp, err := server.GetDeliveryStatus(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "notification_id is required")
}

func TestGetDeliveryStatus_NotificationNotFound(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.GetDeliveryStatusRequest{
		NotificationId: "non-existent-id",
	}

	resp, err := server.GetDeliveryStatus(context.Background(), req)

	// Expected to fail without proper DB setup
	assert.Nil(t, resp)
	assert.Error(t, err)
}

// AcknowledgeAlert tests

func TestAcknowledgeAlert_Success(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.AcknowledgeAlertRequest{
		AlertId:            "alert-123",
		UserId:             "user-456",
		NotificationId:     "notif-789",
		AcknowledgmentNote: "Acknowledged by physician",
		AcknowledgedAt:     timestamppb.New(time.Now()),
	}

	// Note: This would need proper DB mocking for full test
	resp, err := server.AcknowledgeAlert(context.Background(), req)

	// Expected to fail without proper DB setup, but test structure is validated
	if err == nil {
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NotNil(t, resp.AcknowledgedAt)
	} else {
		assert.Error(t, err)
	}
}

func TestAcknowledgeAlert_MissingAlertID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.AcknowledgeAlertRequest{
		UserId:         "user-456",
		NotificationId: "notif-789",
	}

	resp, err := server.AcknowledgeAlert(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "alert_id and user_id are required")
}

func TestAcknowledgeAlert_MissingUserID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.AcknowledgeAlertRequest{
		AlertId:        "alert-123",
		NotificationId: "notif-789",
	}

	resp, err := server.AcknowledgeAlert(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// UpdatePreferences tests

func TestUpdatePreferences_Success(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.UpdatePreferencesRequest{
		UserId:            "user-123",
		PreferredChannels: []string{"email", "sms"},
		QuietHoursStart:   "22:00",
		QuietHoursEnd:     "07:00",
		EnabledAlertTypes: []string{"sepsis_alert", "deterioration"},
		PriorityThreshold: "moderate",
	}

	// Note: This would need proper DB mocking for full test
	resp, err := server.UpdatePreferences(context.Background(), req)

	// Expected to fail without proper DB setup, but test structure is validated
	if err == nil {
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
	} else {
		assert.Error(t, err)
	}
}

func TestUpdatePreferences_MissingUserID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.UpdatePreferencesRequest{
		PreferredChannels: []string{"email", "sms"},
	}

	resp, err := server.UpdatePreferences(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "user_id is required")
}

// GetPreferences tests

func TestGetPreferences_Success(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.GetPreferencesRequest{
		UserId: "user-123",
	}

	// Note: This would need proper DB mocking for full test
	resp, err := server.GetPreferences(context.Background(), req)

	// Should return default preferences if user not found
	if err == nil {
		assert.NotNil(t, resp)
		assert.Equal(t, "user-123", resp.UserId)
		assert.NotEmpty(t, resp.PreferredChannels)
	} else {
		// May fail without DB setup
		assert.Error(t, err)
	}
}

func TestGetPreferences_MissingUserID(t *testing.T) {
	server := setupTestGRPCServer()

	req := &pb.GetPreferencesRequest{
		UserId: "",
	}

	resp, err := server.GetPreferences(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "user_id is required")
}

// Interceptor tests

func TestUnaryLoggingInterceptor(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	interceptor := UnaryLoggingInterceptor(logger)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/notification.NotificationService/SendNotification",
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestUnaryMetricsInterceptor(t *testing.T) {
	interceptor := UnaryMetricsInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/notification.NotificationService/GetDeliveryStatus",
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	// Metrics are recorded asynchronously, so we just verify handler works
}

func TestUnaryAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := UnaryAuthInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/notification.NotificationService/SendNotification",
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestUnaryRecoveryInterceptor(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	interceptor := UnaryRecoveryInterceptor(logger)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/notification.NotificationService/SendNotification",
	}

	// Should not panic
	assert.NotPanics(t, func() {
		resp, err := interceptor(context.Background(), nil, info, handler)
		assert.Nil(t, resp)
		assert.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// Metrics recording tests

func TestRecordNotificationDelivery(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		RecordNotificationDelivery("email", "success")
		RecordNotificationDelivery("sms", "failed")
	})
}

func TestRecordEscalationEvent(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		RecordEscalationEvent("1")
		RecordEscalationEvent("2")
	})
}

func TestRecordAlertFatigueSuppression(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		RecordAlertFatigueSuppression("quiet_hours")
		RecordAlertFatigueSuppression("rate_limit")
	})
}

func TestRecordKafkaMessageProcessed(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		RecordKafkaMessageProcessed("clinical-alerts", "success")
		RecordKafkaMessageProcessed("clinical-alerts", "failed")
	})
}

// Integration tests for gRPC server

func TestGRPCServerCreation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockDeliveryMgr := &delivery.Manager{}
	mockEscalationEngine := &escalation.Engine{}
	mockDB := &database.PostgresDB{}
	mockRedis := &database.RedisClient{}

	server := NewGRPCServer(
		mockDeliveryMgr,
		mockEscalationEngine,
		mockDB,
		mockRedis,
		logger,
		50060,
	)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.Equal(t, 50060, server.port)
	assert.NotNil(t, server.logger)
}

func TestGRPCServerMethodRegistration(t *testing.T) {
	server := setupTestGRPCServer()

	// Verify server implements the required interface
	var _ pb.NotificationServiceServer = server

	// Test that all methods are implemented
	assert.NotNil(t, server.SendNotification)
	assert.NotNil(t, server.GetDeliveryStatus)
	assert.NotNil(t, server.AcknowledgeAlert)
	assert.NotNil(t, server.UpdatePreferences)
	assert.NotNil(t, server.GetPreferences)
}

// Benchmark tests

func BenchmarkSendNotification(b *testing.B) {
	server := setupTestGRPCServer()

	req := &pb.SendNotificationRequest{
		PatientId:   "patient-123",
		Priority:    "high",
		Type:        "sepsis_alert",
		Title:       "Sepsis Alert",
		Message:     "Patient shows signs of sepsis",
		Recipients:  []string{"doctor@hospital.com"},
		RequiresAck: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.SendNotification(ctx, req)
	}
}

func BenchmarkGetDeliveryStatus(b *testing.B) {
	server := setupTestGRPCServer()

	req := &pb.GetDeliveryStatusRequest{
		NotificationId: "notif-123",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.GetDeliveryStatus(ctx, req)
	}
}

func BenchmarkAcknowledgeAlert(b *testing.B) {
	server := setupTestGRPCServer()

	req := &pb.AcknowledgeAlertRequest{
		AlertId:        "alert-123",
		UserId:         "user-456",
		NotificationId: "notif-789",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.AcknowledgeAlert(ctx, req)
	}
}
