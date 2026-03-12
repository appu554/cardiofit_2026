package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "medication-service-v2/proto"
	"medication-service-v2/flow2-go-engine/internal/flow2"
)

// Flow2GRPCServer implements the Flow2Engine gRPC service
type Flow2GRPCServer struct {
	pb.UnimplementedFlow2EngineServer
	orchestrator *flow2.Orchestrator
	port         int
}

// NewFlow2GRPCServer creates a new gRPC server instance
func NewFlow2GRPCServer(orchestrator *flow2.Orchestrator, port int) *Flow2GRPCServer {
	return &Flow2GRPCServer{
		orchestrator: orchestrator,
		port:         port,
	}
}

// Start starts the gRPC server
func (s *Flow2GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingInterceptor),
	)

	// Register Flow2Engine service
	pb.RegisterFlow2EngineServer(grpcServer, s)

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("flow2.Flow2Engine", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection for gRPC debugging
	reflection.Register(grpcServer)

	log.Printf("🚀 Flow2 Go Engine gRPC server starting on port %d", s.port)

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	return nil
}

// ExecuteRecipe executes a medication recipe with clinical context
func (s *Flow2GRPCServer) ExecuteRecipe(ctx context.Context, req *pb.RecipeExecutionRequest) (*pb.RecipeExecutionResponse, error) {
	startTime := time.Now()

	log.Printf("📋 Executing recipe: %s for patient: %s", req.RecipeId, req.PatientId)

	// Convert protobuf request to internal format
	clinicalContext := structToMap(req.ClinicalContext)

	// Execute recipe through orchestrator
	result, err := s.orchestrator.ExecuteRecipe(ctx, flow2.RecipeRequest{
		RequestID:       req.RequestId,
		RecipeID:        req.RecipeId,
		PatientID:       req.PatientId,
		MedicationCode:  req.MedicationCode,
		ClinicalContext: clinicalContext,
		SnapshotID:      req.SnapshotId,
		Metadata:        req.Metadata,
	})

	if err != nil {
		return &pb.RecipeExecutionResponse{
			RequestId:      req.RequestId,
			Success:        false,
			Errors:         []string{err.Error()},
			Timestamp:      timestamppb.Now(),
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Convert result to protobuf response
	return &pb.RecipeExecutionResponse{
		RequestId: req.RequestId,
		Success:   true,
		Result: &pb.RecipeResult{
			RecipeId:     result.RecipeID,
			RecipeName:   result.RecipeName,
			SafetyStatus: convertSafetyStatus(result.SafetyStatus),
			Alerts:       convertSafetyAlerts(result.Alerts),
			Recommendations: convertRecommendations(result.Recommendations),
			ClinicalEvidence: mapToStruct(result.ClinicalEvidence),
		},
		Timestamp:       timestamppb.Now(),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// OptimizeDose optimizes medication dosing based on patient parameters
func (s *Flow2GRPCServer) OptimizeDose(ctx context.Context, req *pb.DoseOptimizationRequest) (*pb.DoseOptimizationResponse, error) {
	startTime := time.Now()

	log.Printf("💊 Optimizing dose for medication: %s, patient: %s", req.MedicationCode, req.PatientId)

	// Execute dose optimization through orchestrator
	result, err := s.orchestrator.OptimizeDose(ctx, flow2.DoseRequest{
		RequestID:      req.RequestId,
		PatientID:      req.PatientId,
		MedicationCode: req.MedicationCode,
		PatientContext: convertPatientContext(req.PatientContext),
		Purpose:        req.Purpose.String(),
		Parameters:     req.Parameters,
	})

	if err != nil {
		return &pb.DoseOptimizationResponse{
			RequestId: req.RequestId,
			Success:   false,
			Errors:    []string{err.Error()},
		}, nil
	}

	// Convert result to protobuf response
	return &pb.DoseOptimizationResponse{
		RequestId: req.RequestId,
		Success:   true,
		Recommendation: &pb.DoseRecommendation{
			DoseValue:          result.DoseValue,
			DoseUnit:           result.DoseUnit,
			Route:              result.Route,
			Frequency:          result.Frequency,
			DurationDays:       int32(result.DurationDays),
			CalculationMethod:  result.CalculationMethod,
			CalculationFactors: result.CalculationFactors,
			Adjustments:        convertDoseAdjustments(result.Adjustments),
		},
		SafetyAlerts: convertSafetyAlerts(result.SafetyAlerts),
	}, nil
}

// AnalyzeMedication performs comprehensive medication intelligence analysis
func (s *Flow2GRPCServer) AnalyzeMedication(ctx context.Context, req *pb.MedicationIntelligenceRequest) (*pb.MedicationIntelligenceResponse, error) {
	log.Printf("🔬 Analyzing medications for patient: %s", req.PatientId)

	// Execute medication analysis through orchestrator
	result, err := s.orchestrator.AnalyzeMedications(ctx, flow2.MedicationAnalysisRequest{
		RequestID:         req.RequestId,
		PatientID:         req.PatientId,
		MedicationCodes:   req.MedicationCodes,
		IntelligenceType:  req.IntelligenceType.String(),
		AnalysisDepth:     req.AnalysisDepth.String(),
		PatientContext:    convertPatientContext(req.PatientContext),
	})

	if err != nil {
		return &pb.MedicationIntelligenceResponse{
			RequestId: req.RequestId,
			Success:   false,
			Errors:    []string{err.Error()},
		}, nil
	}

	// Convert result to protobuf response
	return &pb.MedicationIntelligenceResponse{
		RequestId:    req.RequestId,
		Success:      true,
		Analysis:     convertMedicationAnalysis(result.Analysis),
		Interactions: convertDrugInteractions(result.Interactions),
		Alerts:       convertSafetyAlerts(result.Alerts),
	}, nil
}

// ExecuteFlow2 executes Flow2 workflow for complex clinical decisions
func (s *Flow2GRPCServer) ExecuteFlow2(ctx context.Context, req *pb.Flow2Request) (*pb.Flow2Response, error) {
	log.Printf("⚙️ Executing Flow2 workflow: %s for patient: %s", req.ActionType, req.PatientId)

	// Convert protobuf request to internal format
	parameters := structToMap(req.Parameters)
	clinicalData := structToMap(req.ClinicalData)

	// Execute Flow2 workflow through orchestrator
	result, err := s.orchestrator.ExecuteFlow2(ctx, flow2.Flow2Request{
		RequestID:    req.RequestId,
		PatientID:    req.PatientId,
		ActionType:   req.ActionType,
		Parameters:   parameters,
		ClinicalData: clinicalData,
	})

	if err != nil {
		return &pb.Flow2Response{
			RequestId: req.RequestId,
			Success:   false,
			Errors:    []string{err.Error()},
		}, nil
	}

	// Convert result data to Any
	dataAny, err := anypb.New(structpb.NewStringValue(fmt.Sprintf("%v", result.Data)))
	if err != nil {
		return &pb.Flow2Response{
			RequestId: req.RequestId,
			Success:   false,
			Errors:    []string{fmt.Sprintf("failed to marshal result data: %v", err)},
		}, nil
	}

	return &pb.Flow2Response{
		RequestId: req.RequestId,
		Success:   true,
		Data:      dataAny,
		Metadata:  result.Metadata,
	}, nil
}

// HealthCheck returns the health status of the service
func (s *Flow2GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:  pb.ServiceStatus_SERVICE_STATUS_HEALTHY,
		Version: "1.0.0",
		Capabilities: map[string]string{
			"recipe_execution":        "enabled",
			"dose_optimization":       "enabled",
			"medication_intelligence": "enabled",
			"flow2_workflows":        "enabled",
			"streaming":              "enabled",
		},
		Timestamp: timestamppb.Now(),
	}, nil
}

// StreamPatientUpdates streams patient updates for real-time monitoring
func (s *Flow2GRPCServer) StreamPatientUpdates(stream pb.Flow2Engine_StreamPatientUpdatesServer) error {
	log.Printf("📡 Starting patient update stream")

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("Stream ended: %v", err)
			return err
		}

		log.Printf("Received update for patient: %s, type: %s", req.PatientId, req.UpdateType)

		// Process the update and generate alerts
		alert := s.processPatientUpdate(req)

		// Send alert back to client
		if err := stream.Send(alert); err != nil {
			log.Printf("Failed to send alert: %v", err)
			return err
		}
	}
}

// processPatientUpdate processes a patient update and generates clinical alerts
func (s *Flow2GRPCServer) processPatientUpdate(req *pb.PatientUpdateRequest) *pb.ClinicalAlert {
	// This would contain real clinical logic
	return &pb.ClinicalAlert{
		AlertId:   fmt.Sprintf("alert-%d", time.Now().Unix()),
		PatientId: req.PatientId,
		Timestamp: timestamppb.Now(),
		Severity:  pb.AlertSeverity_ALERT_SEVERITY_INFO,
		Message:   fmt.Sprintf("Update processed for patient %s", req.PatientId),
		Actions:   []string{"Review patient chart", "Consider medication adjustment"},
	}
}

// loggingInterceptor logs all gRPC calls
func (s *Flow2GRPCServer) loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()

	// Call the handler
	resp, err := handler(ctx, req)

	// Log the call
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("❌ gRPC call %s failed after %v: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("✅ gRPC call %s completed in %v", info.FullMethod, duration)
	}

	return resp, err
}

// Converter functions

func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return make(map[string]interface{})
	}
	return s.AsMap()
}

func mapToStruct(m map[string]interface{}) *structpb.Struct {
	s, _ := structpb.NewStruct(m)
	return s
}

func convertSafetyStatus(status string) pb.SafetyStatus {
	switch status {
	case "SAFE":
		return pb.SafetyStatus_SAFETY_STATUS_SAFE
	case "CAUTION":
		return pb.SafetyStatus_SAFETY_STATUS_CAUTION
	case "WARNING":
		return pb.SafetyStatus_SAFETY_STATUS_WARNING
	case "CONTRAINDICATED":
		return pb.SafetyStatus_SAFETY_STATUS_CONTRAINDICATED
	default:
		return pb.SafetyStatus_SAFETY_STATUS_UNKNOWN
	}
}

func convertSafetyAlerts(alerts []flow2.SafetyAlert) []*pb.SafetyAlert {
	var pbAlerts []*pb.SafetyAlert
	for _, alert := range alerts {
		pbAlerts = append(pbAlerts, &pb.SafetyAlert{
			AlertId:              alert.ID,
			Severity:             convertAlertSeverity(alert.Severity),
			Category:             alert.Category,
			Message:              alert.Message,
			ClinicalSignificance: alert.ClinicalSignificance,
			RecommendedActions:   alert.RecommendedActions,
			EvidenceReferences:   convertReferences(alert.References),
		})
	}
	return pbAlerts
}

func convertAlertSeverity(severity string) pb.AlertSeverity {
	switch severity {
	case "INFO":
		return pb.AlertSeverity_ALERT_SEVERITY_INFO
	case "LOW":
		return pb.AlertSeverity_ALERT_SEVERITY_LOW
	case "MEDIUM":
		return pb.AlertSeverity_ALERT_SEVERITY_MEDIUM
	case "HIGH":
		return pb.AlertSeverity_ALERT_SEVERITY_HIGH
	case "CRITICAL":
		return pb.AlertSeverity_ALERT_SEVERITY_CRITICAL
	default:
		return pb.AlertSeverity_ALERT_SEVERITY_UNKNOWN
	}
}

func convertRecommendations(recs []flow2.ClinicalRecommendation) []*pb.ClinicalRecommendation {
	var pbRecs []*pb.ClinicalRecommendation
	for _, rec := range recs {
		pbRecs = append(pbRecs, &pb.ClinicalRecommendation{
			RecommendationId: rec.ID,
			Type:            rec.Type,
			Description:     rec.Description,
			Priority:        convertPriority(rec.Priority),
			Rationale:       rec.Rationale,
			ActionItems:     rec.ActionItems,
		})
	}
	return pbRecs
}

func convertPriority(priority string) pb.Priority {
	switch priority {
	case "LOW":
		return pb.Priority_PRIORITY_LOW
	case "MEDIUM":
		return pb.Priority_PRIORITY_MEDIUM
	case "HIGH":
		return pb.Priority_PRIORITY_HIGH
	case "URGENT":
		return pb.Priority_PRIORITY_URGENT
	default:
		return pb.Priority_PRIORITY_UNKNOWN
	}
}

func convertPatientContext(pc *pb.PatientContext) flow2.PatientContext {
	if pc == nil {
		return flow2.PatientContext{}
	}

	return flow2.PatientContext{
		PatientID:            pc.PatientId,
		AgeYears:            pc.AgeYears,
		WeightKg:            pc.WeightKg,
		HeightCm:            pc.HeightCm,
		Gender:              pc.Gender,
		CreatinineClearance: pc.CreatinineClearance,
		EGFR:                pc.Egfr,
		HepaticFunction:     pc.HepaticFunction,
		PregnancyStatus:     pc.PregnancyStatus,
		ActiveConditions:    pc.ActiveConditions,
		CurrentMedications:  pc.CurrentMedications,
		Allergies:           convertAllergies(pc.Allergies),
		LabValues:          pc.LabValues,
	}
}

func convertAllergies(allergies []*pb.Allergy) []flow2.Allergy {
	var result []flow2.Allergy
	for _, a := range allergies {
		result = append(result, flow2.Allergy{
			AllergenCode: a.AllergenCode,
			AllergenName: a.AllergenName,
			ReactionType: a.ReactionType,
			Severity:     a.Severity,
		})
	}
	return result
}

func convertDoseAdjustments(adjustments []flow2.DoseAdjustment) []*pb.DoseAdjustment {
	var pbAdjustments []*pb.DoseAdjustment
	for _, adj := range adjustments {
		pbAdjustments = append(pbAdjustments, &pb.DoseAdjustment{
			Reason:            adj.Reason,
			AdjustmentType:    adj.Type,
			Factor:            adj.Factor,
			ClinicalRationale: adj.ClinicalRationale,
		})
	}
	return pbAdjustments
}

func convertMedicationAnalysis(analysis flow2.MedicationAnalysis) *pb.MedicationAnalysis {
	return &pb.MedicationAnalysis{
		MedicationCode:        analysis.MedicationCode,
		MedicationName:        analysis.MedicationName,
		SafetyConsiderations: convertSafetyConsiderations(analysis.SafetyConsiderations),
		PkProfile:            convertPKProfile(analysis.PKProfile),
		Guidelines:           convertGuidelines(analysis.Guidelines),
		Monitoring:           convertMonitoring(analysis.Monitoring),
	}
}

func convertSafetyConsiderations(considerations []flow2.SafetyConsideration) []*pb.SafetyConsideration {
	var pbConsiderations []*pb.SafetyConsideration
	for _, c := range considerations {
		pbConsiderations = append(pbConsiderations, &pb.SafetyConsideration{
			Type:                 c.Type,
			Description:         c.Description,
			RiskLevel:           convertRiskLevel(c.RiskLevel),
			MitigationStrategies: c.MitigationStrategies,
		})
	}
	return pbConsiderations
}

func convertRiskLevel(level string) pb.RiskLevel {
	switch level {
	case "LOW":
		return pb.RiskLevel_RISK_LEVEL_LOW
	case "MODERATE":
		return pb.RiskLevel_RISK_LEVEL_MODERATE
	case "HIGH":
		return pb.RiskLevel_RISK_LEVEL_HIGH
	case "VERY_HIGH":
		return pb.RiskLevel_RISK_LEVEL_VERY_HIGH
	default:
		return pb.RiskLevel_RISK_LEVEL_UNKNOWN
	}
}

func convertPKProfile(profile flow2.PharmacokineticProfile) *pb.PharmacokineticProfile {
	return &pb.PharmacokineticProfile{
		HalfLifeHours:      profile.HalfLifeHours,
		Clearance:          profile.Clearance,
		VolumeDistribution: profile.VolumeDistribution,
		MetabolismPathway:  profile.MetabolismPathway,
		CypInteractions:    profile.CYPInteractions,
		Bioavailability:    profile.Bioavailability,
	}
}

func convertGuidelines(guidelines []flow2.TherapeuticGuideline) []*pb.TherapeuticGuideline {
	var pbGuidelines []*pb.TherapeuticGuideline
	for _, g := range guidelines {
		pbGuidelines = append(pbGuidelines, &pb.TherapeuticGuideline{
			Condition: g.Condition,
			Population: g.Population,
			DoseRange: &pb.DoseRange{
				MinDose:   g.MinDose,
				MaxDose:   g.MaxDose,
				Unit:      g.Unit,
				Frequency: g.Frequency,
			},
			MonitoringParameters: g.MonitoringParameters,
			EvidenceLevel:       g.EvidenceLevel,
		})
	}
	return pbGuidelines
}

func convertMonitoring(monitoring []flow2.MonitoringRequirement) []*pb.MonitoringRequirement {
	var pbMonitoring []*pb.MonitoringRequirement
	for _, m := range monitoring {
		pbMonitoring = append(pbMonitoring, &pb.MonitoringRequirement{
			Parameter:            m.Parameter,
			Frequency:           m.Frequency,
			TargetRange:         m.TargetRange,
			ClinicalSignificance: m.ClinicalSignificance,
		})
	}
	return pbMonitoring
}

func convertDrugInteractions(interactions []flow2.DrugInteraction) []*pb.DrugInteraction {
	var pbInteractions []*pb.DrugInteraction
	for _, i := range interactions {
		pbInteractions = append(pbInteractions, &pb.DrugInteraction{
			Drug1Code:      i.Drug1Code,
			Drug1Name:      i.Drug1Name,
			Drug2Code:      i.Drug2Code,
			Drug2Name:      i.Drug2Name,
			Severity:       convertInteractionSeverity(i.Severity),
			Mechanism:      i.Mechanism,
			ClinicalEffect: i.ClinicalEffect,
			Management:     i.Management,
			References:     convertReferences(i.References),
		})
	}
	return pbInteractions
}

func convertInteractionSeverity(severity string) pb.InteractionSeverity {
	switch severity {
	case "MINOR":
		return pb.InteractionSeverity_INTERACTION_SEVERITY_MINOR
	case "MODERATE":
		return pb.InteractionSeverity_INTERACTION_SEVERITY_MODERATE
	case "MAJOR":
		return pb.InteractionSeverity_INTERACTION_SEVERITY_MAJOR
	case "CONTRAINDICATED":
		return pb.InteractionSeverity_INTERACTION_SEVERITY_CONTRAINDICATED
	default:
		return pb.InteractionSeverity_INTERACTION_SEVERITY_UNKNOWN
	}
}

func convertReferences(refs []flow2.Reference) []*pb.Reference {
	var pbRefs []*pb.Reference
	for _, r := range refs {
		pbRefs = append(pbRefs, &pb.Reference{
			Source:        r.Source,
			Title:         r.Title,
			Url:           r.URL,
			EvidenceLevel: r.EvidenceLevel,
		})
	}
	return pbRefs
}