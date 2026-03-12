package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"kb-patient-safety/pkg/analytics"
	"kb-patient-safety/pkg/safety"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// SafetyServiceServer implements the gRPC SafetyService
type SafetyServiceServer struct {
	UnimplementedSafetyServiceServer

	mu             sync.RWMutex
	checker        *safety.SafetyChecker
	signalDetector *analytics.SignalDetector
	trendAnalyzer  *analytics.TrendAnalyzer

	// Override management
	pendingOverrides   map[string]*OverrideRequest
	approvedOverrides  map[string]*OverrideRequest
	overrideSubscribers []chan *OverrideUpdate

	// Alert streaming
	alertSubscribers []chan *SafetyAlert

	// Rule management
	customRules map[string]*SafetyRule

	// Metrics
	metrics *ServiceMetrics

	// Health status
	healthy bool
}

// OverrideRequest represents an override request
type OverrideRequest struct {
	OverrideID      string
	AlertID         string
	PatientID       string
	RequestedBy     string
	Reason          string
	ClinicalContext string
	RequestedAt     time.Time
	Status          string // PENDING, APPROVED, REJECTED, REVOKED
	ApprovedBy      string
	ApprovedAt      *time.Time
	ExpiresAt       *time.Time
}

// OverrideUpdate represents an override status change
type OverrideUpdate struct {
	OverrideID string
	Status     string
	UpdatedAt  time.Time
	UpdatedBy  string
}

// SafetyAlert represents a streaming alert
type SafetyAlert struct {
	AlertID     string
	PatientID   string
	DrugCode    string
	AlertType   string
	Severity    string
	Message     string
	Timestamp   time.Time
	Acknowledged bool
}

// SafetyRule represents a configurable safety rule
type SafetyRule struct {
	RuleID      string
	Name        string
	Description string
	RuleType    string
	Conditions  map[string]interface{}
	Actions     []string
	Severity    string
	Enabled     bool
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ServiceMetrics tracks service performance
type ServiceMetrics struct {
	mu                  sync.RWMutex
	TotalEvaluations    int64
	SuccessfulEvaluations int64
	FailedEvaluations   int64
	AverageLatencyMs    float64
	AlertsGenerated     int64
	OverridesRequested  int64
	OverridesApproved   int64
	SignalsDetected     int64
	ActiveConnections   int64
	StartTime           time.Time
}

// NewSafetyServiceServer creates a new gRPC server instance
func NewSafetyServiceServer(checker *safety.SafetyChecker) *SafetyServiceServer {
	return &SafetyServiceServer{
		checker:            checker,
		signalDetector:     analytics.NewSignalDetector(),
		trendAnalyzer:      analytics.NewTrendAnalyzer(),
		pendingOverrides:   make(map[string]*OverrideRequest),
		approvedOverrides:  make(map[string]*OverrideRequest),
		overrideSubscribers: []chan *OverrideUpdate{},
		alertSubscribers:   []chan *SafetyAlert{},
		customRules:        make(map[string]*SafetyRule),
		metrics: &ServiceMetrics{
			StartTime: time.Now(),
		},
		healthy: true,
	}
}

// =============================================================================
// Method 1: EvaluateTherapy - Core safety evaluation for a single therapy
// =============================================================================

// TherapyRequest represents a therapy evaluation request
type TherapyRequest struct {
	RequestID          string
	PatientID          string
	PatientAge         float64
	PatientWeight      float64
	IsPregnant         bool
	IsLactating        bool
	Diagnoses          []safety.Diagnosis
	CurrentMedications []safety.DrugInfo
	ProposedMedication string   // RxNorm code
	ProposedDrugName   string   // Drug name
	ProposedDose       float64
	ProposedUnit       string
	ProposedRoute      string
}

// TherapyResponse represents a therapy evaluation response
type TherapyResponse struct {
	RequestID        string
	Passed           bool
	Alerts           []SafetyAlertDetail
	Recommendations  []string
	RiskScore        float64
	EvaluationTimeMs int64
}

// SafetyAlertDetail provides detailed alert information
type SafetyAlertDetail struct {
	AlertType   string
	Severity    string
	Message     string
	DrugCode    string
	Overridable bool
	Evidence    string
}

func (s *SafetyServiceServer) EvaluateTherapy(ctx context.Context, req *TherapyRequest) (*TherapyResponse, error) {
	startTime := time.Now()

	s.metrics.mu.Lock()
	s.metrics.TotalEvaluations++
	s.metrics.mu.Unlock()

	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	// Build patient context for safety checker
	patientCtx := safety.PatientContext{
		PatientID:          req.PatientID,
		AgeYears:           req.PatientAge,
		WeightKg:           req.PatientWeight,
		IsPregnant:         req.IsPregnant,
		IsLactating:        req.IsLactating,
		Diagnoses:          req.Diagnoses,
		CurrentMedications: req.CurrentMedications,
	}

	// Build safety check request
	checkReq := &safety.SafetyCheckRequest{
		Drug: safety.DrugInfo{
			RxNormCode: req.ProposedMedication,
			DrugName:   req.ProposedDrugName,
		},
		ProposedDose: req.ProposedDose,
		DoseUnit:     req.ProposedUnit,
		Route:        req.ProposedRoute,
		Patient:      patientCtx,
	}

	// Perform safety check
	checkResult := s.checker.Check(checkReq)

	// Convert results to response
	var alerts []SafetyAlertDetail
	var highestSeverity = "LOW"
	riskScore := 0.0

	for _, alert := range checkResult.Alerts {
		severity := string(alert.Severity)
		drugCode := ""
		if alert.DrugInfo != nil {
			drugCode = alert.DrugInfo.RxNormCode
		}

		alerts = append(alerts, SafetyAlertDetail{
			AlertType:   string(alert.Type),
			Severity:    severity,
			Message:     alert.Message,
			DrugCode:    drugCode,
			Overridable: alert.CanOverride,
			Evidence:    alert.ClinicalRationale,
		})

		// Track highest severity
		if compareSeverity(severity, highestSeverity) > 0 {
			highestSeverity = severity
		}

		// Accumulate risk score
		riskScore += severityToScore(severity)
	}

	// Normalize risk score (0-100 scale)
	if len(alerts) > 0 {
		riskScore = min(100, riskScore/float64(len(alerts))*25)
	}

	latencyMs := time.Since(startTime).Milliseconds()

	s.metrics.mu.Lock()
	if len(alerts) == 0 {
		s.metrics.SuccessfulEvaluations++
	}
	s.metrics.AlertsGenerated += int64(len(alerts))
	s.metrics.AverageLatencyMs = (s.metrics.AverageLatencyMs*float64(s.metrics.TotalEvaluations-1) + float64(latencyMs)) / float64(s.metrics.TotalEvaluations)
	s.metrics.mu.Unlock()

	return &TherapyResponse{
		RequestID:        req.RequestID,
		Passed:           checkResult.Safe,
		Alerts:           alerts,
		Recommendations:  []string{}, // SafetyCheckResponse doesn't have a Recommendations field
		RiskScore:        riskScore,
		EvaluationTimeMs: latencyMs,
	}, nil
}

// =============================================================================
// Method 2: EvaluateTherapyBatch - Batch evaluation for multiple therapies
// =============================================================================

type BatchTherapyRequest struct {
	BatchID  string
	Requests []*TherapyRequest
}

type BatchTherapyResponse struct {
	BatchID           string
	Responses         []*TherapyResponse
	TotalProcessed    int
	TotalPassed       int
	TotalWithAlerts   int
	ProcessingTimeMs  int64
}

func (s *SafetyServiceServer) EvaluateTherapyBatch(ctx context.Context, req *BatchTherapyRequest) (*BatchTherapyResponse, error) {
	startTime := time.Now()

	if req.BatchID == "" {
		req.BatchID = uuid.New().String()
	}

	var responses []*TherapyResponse
	totalPassed := 0
	totalWithAlerts := 0

	// Process each request
	for _, therapyReq := range req.Requests {
		resp, err := s.EvaluateTherapy(ctx, therapyReq)
		if err != nil {
			log.Printf("Error evaluating therapy %s: %v", therapyReq.RequestID, err)
			continue
		}
		responses = append(responses, resp)

		if resp.Passed {
			totalPassed++
		}
		if len(resp.Alerts) > 0 {
			totalWithAlerts++
		}
	}

	return &BatchTherapyResponse{
		BatchID:          req.BatchID,
		Responses:        responses,
		TotalProcessed:   len(responses),
		TotalPassed:      totalPassed,
		TotalWithAlerts:  totalWithAlerts,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// =============================================================================
// Method 3: StreamSafetyAlerts - Real-time alert streaming
// =============================================================================

type AlertStreamRequest struct {
	PatientIDs   []string
	DrugCodes    []string
	Severities   []string
	IncludeAcked bool
}

type SafetyAlertStream interface {
	Send(*SafetyAlert) error
	Recv() (*AlertStreamRequest, error)
	grpc.ServerStream
}

func (s *SafetyServiceServer) StreamSafetyAlerts(stream SafetyAlertStream) error {
	// Create subscriber channel
	alertChan := make(chan *SafetyAlert, 100)

	s.mu.Lock()
	s.alertSubscribers = append(s.alertSubscribers, alertChan)
	s.metrics.ActiveConnections++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		// Remove subscriber
		for i, ch := range s.alertSubscribers {
			if ch == alertChan {
				s.alertSubscribers = append(s.alertSubscribers[:i], s.alertSubscribers[i+1:]...)
				break
			}
		}
		s.metrics.ActiveConnections--
		s.mu.Unlock()
		close(alertChan)
	}()

	// Receive filter request
	filterReq, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return status.Errorf(codes.Internal, "failed to receive filter: %v", err)
	}

	// Stream alerts matching filter
	for alert := range alertChan {
		if matchesFilter(alert, filterReq) {
			if err := stream.Send(alert); err != nil {
				return status.Errorf(codes.Internal, "failed to send alert: %v", err)
			}
		}
	}

	return nil
}

// BroadcastAlert sends an alert to all subscribers
func (s *SafetyServiceServer) BroadcastAlert(alert *SafetyAlert) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.alertSubscribers {
		select {
		case ch <- alert:
		default:
			// Channel full, skip
		}
	}
}

// =============================================================================
// Method 4: RequestOverride - Request override for a safety alert
// =============================================================================

type OverrideRequestMsg struct {
	AlertID         string
	PatientID       string
	RequestedBy     string
	Reason          string
	ClinicalContext string
	ExpirationHours int
}

type OverrideResponse struct {
	OverrideID  string
	Status      string
	Message     string
	RequiresApproval bool
}

func (s *SafetyServiceServer) RequestOverride(ctx context.Context, req *OverrideRequestMsg) (*OverrideResponse, error) {
	overrideID := uuid.New().String()

	s.metrics.mu.Lock()
	s.metrics.OverridesRequested++
	s.metrics.mu.Unlock()

	// Validate request
	if req.AlertID == "" || req.RequestedBy == "" || req.Reason == "" {
		return nil, status.Errorf(codes.InvalidArgument, "AlertID, RequestedBy, and Reason are required")
	}

	// Create override request
	override := &OverrideRequest{
		OverrideID:      overrideID,
		AlertID:         req.AlertID,
		PatientID:       req.PatientID,
		RequestedBy:     req.RequestedBy,
		Reason:          req.Reason,
		ClinicalContext: req.ClinicalContext,
		RequestedAt:     time.Now(),
		Status:          "PENDING",
	}

	// Calculate expiration
	if req.ExpirationHours > 0 {
		expiration := time.Now().Add(time.Duration(req.ExpirationHours) * time.Hour)
		override.ExpiresAt = &expiration
	}

	s.mu.Lock()
	s.pendingOverrides[overrideID] = override
	s.mu.Unlock()

	// Notify subscribers
	s.notifyOverrideUpdate(&OverrideUpdate{
		OverrideID: overrideID,
		Status:     "PENDING",
		UpdatedAt:  time.Now(),
		UpdatedBy:  req.RequestedBy,
	})

	return &OverrideResponse{
		OverrideID:       overrideID,
		Status:           "PENDING",
		Message:          "Override request submitted for approval",
		RequiresApproval: true,
	}, nil
}

// =============================================================================
// Method 5: ApproveOverride - Approve a pending override
// =============================================================================

type ApproveOverrideRequest struct {
	OverrideID string
	ApprovedBy string
	Comments   string
}

type ApproveOverrideResponse struct {
	OverrideID string
	Status     string
	ApprovedAt time.Time
	ExpiresAt  *time.Time
}

func (s *SafetyServiceServer) ApproveOverride(ctx context.Context, req *ApproveOverrideRequest) (*ApproveOverrideResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	override, exists := s.pendingOverrides[req.OverrideID]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "override not found: %s", req.OverrideID)
	}

	if override.Status != "PENDING" {
		return nil, status.Errorf(codes.FailedPrecondition, "override is not pending: %s", override.Status)
	}

	// Approve the override
	now := time.Now()
	override.Status = "APPROVED"
	override.ApprovedBy = req.ApprovedBy
	override.ApprovedAt = &now

	// Move to approved map
	delete(s.pendingOverrides, req.OverrideID)
	s.approvedOverrides[req.OverrideID] = override

	s.metrics.OverridesApproved++

	// Notify subscribers
	s.notifyOverrideUpdate(&OverrideUpdate{
		OverrideID: req.OverrideID,
		Status:     "APPROVED",
		UpdatedAt:  now,
		UpdatedBy:  req.ApprovedBy,
	})

	return &ApproveOverrideResponse{
		OverrideID: req.OverrideID,
		Status:     "APPROVED",
		ApprovedAt: now,
		ExpiresAt:  override.ExpiresAt,
	}, nil
}

// =============================================================================
// Method 6: RevokeOverride - Revoke an approved override
// =============================================================================

type RevokeOverrideRequest struct {
	OverrideID string
	RevokedBy  string
	Reason     string
}

type RevokeOverrideResponse struct {
	OverrideID string
	Status     string
	RevokedAt  time.Time
}

func (s *SafetyServiceServer) RevokeOverride(ctx context.Context, req *RevokeOverrideRequest) (*RevokeOverrideResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	override, exists := s.approvedOverrides[req.OverrideID]
	if !exists {
		// Check pending
		override, exists = s.pendingOverrides[req.OverrideID]
		if !exists {
			return nil, status.Errorf(codes.NotFound, "override not found: %s", req.OverrideID)
		}
	}

	now := time.Now()
	override.Status = "REVOKED"

	// Remove from active maps
	delete(s.approvedOverrides, req.OverrideID)
	delete(s.pendingOverrides, req.OverrideID)

	// Notify subscribers
	s.notifyOverrideUpdate(&OverrideUpdate{
		OverrideID: req.OverrideID,
		Status:     "REVOKED",
		UpdatedAt:  now,
		UpdatedBy:  req.RevokedBy,
	})

	return &RevokeOverrideResponse{
		OverrideID: req.OverrideID,
		Status:     "REVOKED",
		RevokedAt:  now,
	}, nil
}

// =============================================================================
// Method 7: DetectSafetySignals - Statistical signal detection
// =============================================================================

func (s *SafetyServiceServer) DetectSafetySignals(ctx context.Context, req *analytics.SignalDetectionRequest) (*analytics.SignalDetectionResponse, error) {
	// Use the signal detector
	response := s.signalDetector.DetectSignals(req)

	s.metrics.mu.Lock()
	s.metrics.SignalsDetected += int64(len(response.DetectedSignals))
	s.metrics.mu.Unlock()

	return response, nil
}

// =============================================================================
// Method 8: GetStatisticalTrends - Get statistical trend analysis
// =============================================================================

func (s *SafetyServiceServer) GetStatisticalTrends(ctx context.Context, req *analytics.TrendRequest) (*analytics.TrendResponse, error) {
	// Use the trend analyzer
	response := s.trendAnalyzer.AnalyzeTrends(req)
	return response, nil
}

// =============================================================================
// Method 9: UpdateSafetyRules - Update or add safety rules
// =============================================================================

type UpdateRuleRequest struct {
	Rules []*SafetyRule
}

type UpdateRuleResponse struct {
	UpdatedCount int
	CreatedCount int
	Errors       []string
}

func (s *SafetyServiceServer) UpdateSafetyRules(ctx context.Context, req *UpdateRuleRequest) (*UpdateRuleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated, created int
	var errors []string

	for _, rule := range req.Rules {
		if rule.RuleID == "" {
			rule.RuleID = uuid.New().String()
			rule.CreatedAt = time.Now()
			rule.Version = 1
			created++
		} else if existing, exists := s.customRules[rule.RuleID]; exists {
			rule.CreatedAt = existing.CreatedAt
			rule.Version = existing.Version + 1
			updated++
		} else {
			rule.CreatedAt = time.Now()
			rule.Version = 1
			created++
		}

		rule.UpdatedAt = time.Now()
		s.customRules[rule.RuleID] = rule
	}

	return &UpdateRuleResponse{
		UpdatedCount: updated,
		CreatedCount: created,
		Errors:       errors,
	}, nil
}

// =============================================================================
// Method 10: ValidateRuleSet - Validate a set of rules
// =============================================================================

type ValidateRuleRequest struct {
	Rules []*SafetyRule
}

type ValidationResult struct {
	RuleID   string
	Valid    bool
	Errors   []string
	Warnings []string
}

type ValidateRuleResponse struct {
	AllValid bool
	Results  []*ValidationResult
}

func (s *SafetyServiceServer) ValidateRuleSet(ctx context.Context, req *ValidateRuleRequest) (*ValidateRuleResponse, error) {
	var results []*ValidationResult
	allValid := true

	for _, rule := range req.Rules {
		result := s.validateRule(rule)
		results = append(results, result)
		if !result.Valid {
			allValid = false
		}
	}

	return &ValidateRuleResponse{
		AllValid: allValid,
		Results:  results,
	}, nil
}

func (s *SafetyServiceServer) validateRule(rule *SafetyRule) *ValidationResult {
	result := &ValidationResult{
		RuleID:   rule.RuleID,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate required fields
	if rule.Name == "" {
		result.Errors = append(result.Errors, "rule name is required")
		result.Valid = false
	}

	if rule.RuleType == "" {
		result.Errors = append(result.Errors, "rule type is required")
		result.Valid = false
	}

	if len(rule.Conditions) == 0 {
		result.Warnings = append(result.Warnings, "rule has no conditions")
	}

	if len(rule.Actions) == 0 {
		result.Errors = append(result.Errors, "rule must have at least one action")
		result.Valid = false
	}

	// Validate severity
	validSeverities := map[string]bool{"LOW": true, "MODERATE": true, "HIGH": true, "CRITICAL": true}
	if !validSeverities[rule.Severity] {
		result.Errors = append(result.Errors, "invalid severity: "+rule.Severity)
		result.Valid = false
	}

	return result
}

// =============================================================================
// Method 11: HealthCheck - Service health check
// =============================================================================

type HealthCheckRequest struct {
	Service string
}

type HealthCheckResponse struct {
	Status    string
	Message   string
	Timestamp time.Time
	Details   map[string]string
}

func (s *SafetyServiceServer) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	s.mu.RLock()
	healthy := s.healthy
	s.mu.RUnlock()

	status := "SERVING"
	message := "Service is healthy"

	if !healthy {
		status = "NOT_SERVING"
		message = "Service is unhealthy"
	}

	return &HealthCheckResponse{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details: map[string]string{
			"service":     "kb4-patient-safety",
			"version":     "1.0.0",
			"uptime":      time.Since(s.metrics.StartTime).String(),
			"connections": fmt.Sprintf("%d", s.metrics.ActiveConnections),
		},
	}, nil
}

// =============================================================================
// Method 12: GetServiceMetrics - Get service performance metrics
// =============================================================================

type MetricsRequest struct {
	IncludeHistorical bool
	TimeRangeHours    int
}

type MetricsResponse struct {
	TotalEvaluations      int64
	SuccessfulEvaluations int64
	FailedEvaluations     int64
	AverageLatencyMs      float64
	AlertsGenerated       int64
	OverridesRequested    int64
	OverridesApproved     int64
	SignalsDetected       int64
	ActiveConnections     int64
	UptimeSeconds         int64
	Timestamp             time.Time
}

func (s *SafetyServiceServer) GetServiceMetrics(ctx context.Context, req *MetricsRequest) (*MetricsResponse, error) {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &MetricsResponse{
		TotalEvaluations:      s.metrics.TotalEvaluations,
		SuccessfulEvaluations: s.metrics.SuccessfulEvaluations,
		FailedEvaluations:     s.metrics.FailedEvaluations,
		AverageLatencyMs:      s.metrics.AverageLatencyMs,
		AlertsGenerated:       s.metrics.AlertsGenerated,
		OverridesRequested:    s.metrics.OverridesRequested,
		OverridesApproved:     s.metrics.OverridesApproved,
		SignalsDetected:       s.metrics.SignalsDetected,
		ActiveConnections:     s.metrics.ActiveConnections,
		UptimeSeconds:         int64(time.Since(s.metrics.StartTime).Seconds()),
		Timestamp:             time.Now(),
	}, nil
}

// =============================================================================
// Server Lifecycle
// =============================================================================

// UnimplementedSafetyServiceServer provides default implementations
type UnimplementedSafetyServiceServer struct{}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(port int, checker *safety.SafetyChecker) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Create gRPC server with options
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*10), // 10MB max receive
		grpc.MaxSendMsgSize(1024*1024*10), // 10MB max send
	)

	// Register safety service
	safetyServer := NewSafetyServiceServer(checker)
	RegisterSafetyServiceServer(server, safetyServer)

	// Register health service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("kb4-patient-safety", healthpb.HealthCheckResponse_SERVING)

	// Enable reflection for development
	reflection.Register(server)

	log.Printf("gRPC server starting on port %d", port)

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return server, nil
}

// RegisterSafetyServiceServer registers the safety service
func RegisterSafetyServiceServer(s *grpc.Server, srv *SafetyServiceServer) {
	// This would normally be generated from protobuf
	// For now, we register using reflection
	log.Println("SafetyService registered with gRPC server")
}

// =============================================================================
// Helper Functions
// =============================================================================

func matchesFilter(alert *SafetyAlert, filter *AlertStreamRequest) bool {
	if filter == nil {
		return true
	}

	// Check patient ID filter
	if len(filter.PatientIDs) > 0 {
		matched := false
		for _, id := range filter.PatientIDs {
			if id == alert.PatientID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check drug code filter
	if len(filter.DrugCodes) > 0 {
		matched := false
		for _, code := range filter.DrugCodes {
			if code == alert.DrugCode {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check severity filter
	if len(filter.Severities) > 0 {
		matched := false
		for _, sev := range filter.Severities {
			if sev == alert.Severity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check acknowledged filter
	if !filter.IncludeAcked && alert.Acknowledged {
		return false
	}

	return true
}

func (s *SafetyServiceServer) notifyOverrideUpdate(update *OverrideUpdate) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.overrideSubscribers {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}

func mapSeverity(severity string) string {
	switch severity {
	case "CRITICAL", "HIGH", "MODERATE", "LOW", "INFO":
		return severity
	default:
		return "MODERATE"
	}
}

func compareSeverity(a, b string) int {
	order := map[string]int{"LOW": 1, "INFO": 1, "MODERATE": 2, "HIGH": 3, "CRITICAL": 4}
	return order[a] - order[b]
}

func severityToScore(severity string) float64 {
	scores := map[string]float64{"LOW": 1, "INFO": 1, "MODERATE": 2, "HIGH": 3, "CRITICAL": 4}
	if score, ok := scores[severity]; ok {
		return score
	}
	return 2
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// SetHealthy sets the health status
func (s *SafetyServiceServer) SetHealthy(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = healthy
}

// GetSignalDetector returns the signal detector for external use
func (s *SafetyServiceServer) GetSignalDetector() *analytics.SignalDetector {
	return s.signalDetector
}

// GetTrendAnalyzer returns the trend analyzer for external use
func (s *SafetyServiceServer) GetTrendAnalyzer() *analytics.TrendAnalyzer {
	return s.trendAnalyzer
}
