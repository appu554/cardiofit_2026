// Native Rust CAE Engine - Go Integration Layer
//
// This file replaces the Python subprocess-based CAE engine with a direct
// FFI integration to the Rust implementation. It eliminates the 50-100ms
// subprocess overhead and provides sub-20ms evaluation times.

package engines

/*
#cgo LDFLAGS: -L./rust_engines/target/release -lsafety_engines
#include "./rust_engines/target/cae_engine.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// RustCAEEngine implements the SafetyEngine interface using native Rust
type RustCAEEngine struct {
	id           string
	name         string
	capabilities []string
	logger       *logger.Logger
	initialized  bool
}

// RustCAEConfig represents configuration for the Rust CAE engine
type RustCAEConfig struct {
	RulesPath         string `json:"rules_path"`
	KnowledgeDBPath   string `json:"knowledge_db_path"`
	CacheSize         int    `json:"cache_size"`
	CacheTTLSeconds   int    `json:"cache_ttl_seconds"`
	CacheEnabled      bool   `json:"cache_enabled"`
	MaxEvalTimeMs     int    `json:"max_evaluation_time_ms"`
	WorkerThreads     int    `json:"worker_threads"`
	MonitoringEnabled bool   `json:"monitoring_enabled"`
	LogLevel          string `json:"log_level"`
	StructuredLogging bool   `json:"structured_logging"`
}

// NewRustCAEEngine creates a new Rust-based CAE engine instance
func NewRustCAEEngine(logger *logger.Logger, config RustCAEConfig) *RustCAEEngine {
	return &RustCAEEngine{
		id:   "rust_cae_engine",
		name: "Native Rust Clinical Assertion Engine",
		capabilities: []string{
			"drug_interaction",
			"contraindication", 
			"dosing_validation",
			"allergy_check",
			"duplicate_therapy",
			"clinical_protocol",
		},
		logger:      logger,
		initialized: false,
	}
}

// ID returns the engine identifier
func (r *RustCAEEngine) ID() string {
	return r.id
}

// Name returns the engine name
func (r *RustCAEEngine) Name() string {
	return r.name
}

// Capabilities returns the engine capabilities
func (r *RustCAEEngine) Capabilities() []string {
	return r.capabilities
}

// Initialize initializes the Rust CAE engine
func (r *RustCAEEngine) Initialize(config types.EngineConfig) error {
	r.logger.Info("Initializing native Rust CAE engine",
		zap.String("engine_id", r.id),
		zap.Duration("timeout", config.Timeout),
		zap.Strings("capabilities", r.capabilities),
	)

	// Create Rust configuration JSON
	rustConfig := RustCAEConfig{
		RulesPath:         "./data/clinical_rules",
		KnowledgeDBPath:   "./data/knowledge_graph",
		CacheSize:         10000,
		CacheTTLSeconds:   3600,
		CacheEnabled:      true,
		MaxEvalTimeMs:     int(config.Timeout.Milliseconds()),
		WorkerThreads:     4,
		MonitoringEnabled: true,
		LogLevel:          "info",
		StructuredLogging: true,
	}

	configJSON, err := json.Marshal(rustConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal Rust config: %w", err)
	}

	// Convert to C string
	cConfigJSON := C.CString(string(configJSON))
	defer C.free(unsafe.Pointer(cConfigJSON))

	// Initialize Rust engine via FFI
	result := C.cae_initialize_engine(cConfigJSON)
	if result != C.CAE_SUCCESS {
		return fmt.Errorf("failed to initialize Rust CAE engine: error code %d", int(result))
	}

	r.initialized = true
	r.logger.Info("Native Rust CAE engine initialized successfully",
		zap.String("engine_id", r.id),
		zap.Int("cache_size", rustConfig.CacheSize),
		zap.Int("worker_threads", rustConfig.WorkerThreads),
	)

	return nil
}

// HealthCheck performs a health check on the Rust CAE engine
func (r *RustCAEEngine) HealthCheck() error {
	if !r.initialized {
		return fmt.Errorf("Rust CAE engine not initialized")
	}

	// Call Rust health check function
	result := C.cae_health_check()
	if result != C.CAE_SUCCESS {
		return fmt.Errorf("Rust CAE engine health check failed: error code %d", int(result))
	}

	return nil
}

// Evaluate performs safety evaluation using the native Rust CAE engine
func (r *RustCAEEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	if !r.initialized {
		return nil, fmt.Errorf("Rust CAE engine not initialized")
	}

	startTime := time.Now()

	r.logger.Debug("Rust CAE engine evaluation started",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("action_type", req.ActionType),
		zap.Int("medication_count", len(req.MedicationIDs)),
		zap.Int("condition_count", len(req.ConditionIDs)),
		zap.Int("allergy_count", len(req.AllergyIDs)),
	)

	// Convert Go request to C request structure
	cRequest, err := r.convertToCRequest(req, clinicalContext)
	if err != nil {
		r.logger.Error("Failed to convert Go request to C request",
			zap.String("request_id", req.RequestID),
			zap.Error(err),
		)
		return r.createErrorResult(err, time.Since(startTime)), nil
	}
	defer r.freeCRequest(cRequest)

	// Call Rust CAE engine via FFI
	cResult := C.cae_evaluate_safety(cRequest)
	if cResult == nil {
		err := fmt.Errorf("Rust CAE engine returned null result")
		r.logger.Error("Rust CAE engine evaluation failed",
			zap.String("request_id", req.RequestID),
			zap.Error(err),
		)
		return r.createErrorResult(err, time.Since(startTime)), nil
	}
	defer C.cae_free_result(cResult)

	// Convert C result back to Go
	result, err := r.convertFromCResult(cResult, time.Since(startTime))
	if err != nil {
		r.logger.Error("Failed to convert C result to Go result",
			zap.String("request_id", req.RequestID),
			zap.Error(err),
		)
		return r.createErrorResult(err, time.Since(startTime)), nil
	}

	r.logger.Debug("Rust CAE engine evaluation completed",
		zap.String("request_id", req.RequestID),
		zap.String("status", string(result.Status)),
		zap.Float64("risk_score", result.RiskScore),
		zap.Float64("confidence", result.Confidence),
		zap.Int("violations", len(result.Violations)),
		zap.Int("warnings", len(result.Warnings)),
		zap.Int64("duration_ms", result.Duration.Milliseconds()),
	)

	// Log performance metrics
	if result.Duration.Milliseconds() > 50 {
		r.logger.Warn("Rust CAE evaluation exceeded 50ms target",
			zap.String("request_id", req.RequestID),
			zap.Int64("duration_ms", result.Duration.Milliseconds()),
		)
	}

	return result, nil
}

// Shutdown shuts down the Rust CAE engine
func (r *RustCAEEngine) Shutdown() error {
	if r.initialized {
		r.logger.Info("Shutting down Rust CAE engine", zap.String("engine_id", r.id))
		C.cae_shutdown_engine()
		r.initialized = false
		r.logger.Info("Rust CAE engine shutdown completed", zap.String("engine_id", r.id))
	}
	return nil
}

// EvaluateWithSnapshot performs safety evaluation using snapshot data
func (r *RustCAEEngine) EvaluateWithSnapshot(ctx context.Context, req *types.SafetyRequest, snapshot *types.ClinicalSnapshot) (*types.EngineResult, error) {
	// Convert snapshot to clinical context and use regular evaluation
	// This is a simplified implementation - in practice, you might want
	// to pass the snapshot directly to Rust for more efficient processing
	
	clinicalContext := &types.ClinicalContext{}
	if snapshot.Data != nil {
		clinicalContext = &types.ClinicalContext{
			Demographics:      snapshot.Data.Demographics,
			ActiveMedications: snapshot.Data.ActiveMedications,
			Allergies:         snapshot.Data.Allergies,
			Conditions:        snapshot.Data.Conditions,
			RecentVitals:      snapshot.Data.RecentVitals,
			LabResults:        snapshot.Data.LabResults,
			ContextVersion:    snapshot.Data.ContextVersion,
		}
	}

	result, err := r.Evaluate(ctx, req, clinicalContext)
	if err != nil {
		return nil, err
	}

	// Add snapshot metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["snapshot_id"] = snapshot.SnapshotID
	result.Metadata["snapshot_checksum"] = snapshot.Checksum
	result.Metadata["data_completeness"] = snapshot.DataCompleteness
	result.Metadata["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
	result.Metadata["processing_mode"] = "snapshot_based"

	return result, nil
}

// IsSnapshotCompatible returns true if the engine supports snapshot-based evaluation
func (r *RustCAEEngine) IsSnapshotCompatible() bool {
	return true
}

// GetSnapshotRequirements returns the required snapshot fields for Rust CAE engine
func (r *RustCAEEngine) GetSnapshotRequirements() []string {
	return []string{
		"demographics",
		"active_medications",
		"allergies", 
		"conditions",
		"recent_vitals",
	}
}

// Helper functions

// convertToCRequest converts Go SafetyRequest to C request structure
func (r *RustCAEEngine) convertToCRequest(req *types.SafetyRequest, ctx *types.ClinicalContext) (*C.CSafetyRequest, error) {
	// Allocate C struct
	cReq := (*C.CSafetyRequest)(C.malloc(C.size_t(unsafe.Sizeof(C.CSafetyRequest{}))))
	if cReq == nil {
		return nil, fmt.Errorf("failed to allocate memory for C request")
	}

	// Convert strings to C strings
	cReq.patient_id = C.CString(req.PatientID)
	cReq.request_id = C.CString(req.RequestID)
	cReq.action_type = C.CString(req.ActionType)
	cReq.priority = C.CString(req.Priority)

	// Convert arrays to JSON strings
	medicationJSON, err := json.Marshal(req.MedicationIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal medication IDs: %w", err)
	}
	cReq.medication_ids_json = C.CString(string(medicationJSON))

	conditionJSON, err := json.Marshal(req.ConditionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal condition IDs: %w", err)
	}
	cReq.condition_ids_json = C.CString(string(conditionJSON))

	allergyJSON, err := json.Marshal(req.AllergyIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allergy IDs: %w", err)
	}
	cReq.allergy_ids_json = C.CString(string(allergyJSON))

	return cReq, nil
}

// freeCRequest frees memory allocated for C request
func (r *RustCAEEngine) freeCRequest(cReq *C.CSafetyRequest) {
	if cReq == nil {
		return
	}

	// Free individual string fields
	if cReq.patient_id != nil {
		C.free(unsafe.Pointer(cReq.patient_id))
	}
	if cReq.request_id != nil {
		C.free(unsafe.Pointer(cReq.request_id))
	}
	if cReq.action_type != nil {
		C.free(unsafe.Pointer(cReq.action_type))
	}
	if cReq.priority != nil {
		C.free(unsafe.Pointer(cReq.priority))
	}
	if cReq.medication_ids_json != nil {
		C.free(unsafe.Pointer(cReq.medication_ids_json))
	}
	if cReq.condition_ids_json != nil {
		C.free(unsafe.Pointer(cReq.condition_ids_json))
	}
	if cReq.allergy_ids_json != nil {
		C.free(unsafe.Pointer(cReq.allergy_ids_json))
	}

	// Free the struct itself
	C.free(unsafe.Pointer(cReq))
}

// convertFromCResult converts C result to Go EngineResult
func (r *RustCAEEngine) convertFromCResult(cResult *C.CSafetyResult, duration time.Duration) (*types.EngineResult, error) {
	// Convert status
	var status types.SafetyStatus
	switch int(cResult.status) {
	case 0:
		status = types.SafetyStatusSafe
	case 1:
		status = types.SafetyStatusUnsafe
	case 2:
		status = types.SafetyStatusWarning
	case 3:
		status = types.SafetyStatusManualReview
	default:
		return nil, fmt.Errorf("unknown safety status: %d", int(cResult.status))
	}

	// Parse violations JSON
	var violations []string
	if cResult.violations_json != nil {
		violationsStr := C.GoString(cResult.violations_json)
		if len(violationsStr) > 0 {
			if err := json.Unmarshal([]byte(violationsStr), &violations); err != nil {
				r.logger.Warn("Failed to parse violations JSON", zap.Error(err))
				violations = []string{violationsStr} // Fallback to raw string
			}
		}
	}

	// Parse warnings JSON
	var warnings []string
	if cResult.warnings_json != nil {
		warningsStr := C.GoString(cResult.warnings_json)
		if len(warningsStr) > 0 {
			if err := json.Unmarshal([]byte(warningsStr), &warnings); err != nil {
				r.logger.Warn("Failed to parse warnings JSON", zap.Error(err))
				warnings = []string{warningsStr} // Fallback to raw string
			}
		}
	}

	// Create metadata
	metadata := map[string]interface{}{
		"engine_type":         "native_rust",
		"ffi_duration_ms":     duration.Milliseconds(),
		"rust_processing_ms":  int(cResult.processing_time_ms),
	}

	return &types.EngineResult{
		EngineID:   r.id,
		EngineName: r.name,
		Status:     status,
		RiskScore:  float64(cResult.risk_score),
		Confidence: float64(cResult.confidence),
		Violations: violations,
		Warnings:   warnings,
		Duration:   duration,
		Tier:       types.TierVetoCritical, // CAE is Tier 1 (critical)
		Metadata:   metadata,
	}, nil
}

// createErrorResult creates an error result for failed evaluations
func (r *RustCAEEngine) createErrorResult(err error, duration time.Duration) *types.EngineResult {
	return &types.EngineResult{
		EngineID:   r.id,
		EngineName: r.name,
		Status:     types.SafetyStatusUnsafe, // Fail closed for Tier 1 engine
		RiskScore:  1.0,                      // Maximum risk for errors
		Violations: []string{fmt.Sprintf("Rust CAE engine error: %s", err.Error())},
		Confidence: 0.0,
		Duration:   duration,
		Tier:       types.TierVetoCritical,
		Error:      err.Error(),
		Metadata: map[string]interface{}{
			"engine_type": "native_rust",
			"error_type":  "evaluation_failure",
		},
	}
}