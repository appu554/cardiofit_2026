// Package proto contains protobuf definitions for KB-1 gRPC service
// This is a simplified Go implementation of the protobuf messages
// In production, these would be generated from .proto files using protoc
package proto

import "encoding/json"

// DosingServiceServer defines the gRPC service interface
type DosingServiceServer interface {
	GetDosingRule(*GetDosingRuleRequest, DosingService_GetDosingRuleServer) error
	CalculateDose(*CalculateDoseRequest, DosingService_CalculateDoseServer) error
	BatchGetDosingRules(*BatchGetDosingRulesRequest, DosingService_BatchGetDosingRulesServer) error
	ValidateDosingRule(*ValidateDosingRuleRequest, DosingService_ValidateDosingRuleServer) error
	StreamDosingUpdates(*StreamDosingUpdatesRequest, DosingService_StreamDosingUpdatesServer) error
}

// UnimplementedDosingServiceServer provides default implementations
type UnimplementedDosingServiceServer struct{}

func (UnimplementedDosingServiceServer) GetDosingRule(*GetDosingRuleRequest, DosingService_GetDosingRuleServer) error {
	return nil
}
func (UnimplementedDosingServiceServer) CalculateDose(*CalculateDoseRequest, DosingService_CalculateDoseServer) error {
	return nil
}
func (UnimplementedDosingServiceServer) BatchGetDosingRules(*BatchGetDosingRulesRequest, DosingService_BatchGetDosingRulesServer) error {
	return nil
}
func (UnimplementedDosingServiceServer) ValidateDosingRule(*ValidateDosingRuleRequest, DosingService_ValidateDosingRuleServer) error {
	return nil
}
func (UnimplementedDosingServiceServer) StreamDosingUpdates(*StreamDosingUpdatesRequest, DosingService_StreamDosingUpdatesServer) error {
	return nil
}

// DosingService_GetDosingRuleServer is the stream server interface
type DosingService_GetDosingRuleServer interface {
	Send(*DosingRuleResponse) error
}

// DosingService_CalculateDoseServer is the stream server interface
type DosingService_CalculateDoseServer interface {
	Send(*CalculateDoseResponse) error
}

// DosingService_BatchGetDosingRulesServer is the stream server interface
type DosingService_BatchGetDosingRulesServer interface {
	Send(*BatchDosingRulesResponse) error
}

// DosingService_ValidateDosingRuleServer is the stream server interface
type DosingService_ValidateDosingRuleServer interface {
	Send(*ValidationResponse) error
}

// DosingService_StreamDosingUpdatesServer is the stream server interface for updates
type DosingService_StreamDosingUpdatesServer interface {
	Send(*DosingUpdate) error
	Context() interface{ Done() <-chan struct{} }
}

// RegisterDosingServiceServer registers the dosing service with a gRPC server
func RegisterDosingServiceServer(s interface{}, srv interface{}) {
	// In production, this would register with the actual gRPC server
	// grpcServer.RegisterService(&DosingService_ServiceDesc, srv)
}

// GetDosingRuleRequest is the request for getting a dosing rule
type GetDosingRuleRequest struct {
	DrugCode string `json:"drug_code"`
}

func (r *GetDosingRuleRequest) GetDrugCode() string {
	if r == nil {
		return ""
	}
	return r.DrugCode
}

// CalculateDoseRequest is the request for dose calculation
type CalculateDoseRequest struct {
	DrugCode string             `json:"drug_code"`
	Patient  *PatientParameters `json:"patient"`
}

func (r *CalculateDoseRequest) GetDrugCode() string {
	if r == nil {
		return ""
	}
	return r.DrugCode
}

func (r *CalculateDoseRequest) GetPatient() *PatientParameters {
	if r == nil {
		return nil
	}
	return r.Patient
}

// PatientParameters contains patient-specific dosing parameters
type PatientParameters struct {
	Age             int32   `json:"age"`
	WeightKg        float64 `json:"weight_kg"`
	HeightCm        float64 `json:"height_cm"`
	Gender          string  `json:"gender"`
	EGFR            float64 `json:"egfr"`
	SerumCreatinine float64 `json:"serum_creatinine"`
	ChildPughScore  string  `json:"child_pugh_score"`
	IsPregnant      bool    `json:"is_pregnant"`
	IsBreastfeeding bool    `json:"is_breastfeeding"`
}

func (p *PatientParameters) GetAge() int32 {
	if p == nil {
		return 0
	}
	return p.Age
}

func (p *PatientParameters) GetWeightKg() float64 {
	if p == nil {
		return 0
	}
	return p.WeightKg
}

func (p *PatientParameters) GetHeightCm() float64 {
	if p == nil {
		return 0
	}
	return p.HeightCm
}

func (p *PatientParameters) GetEgfr() float64 {
	if p == nil {
		return 0
	}
	return p.EGFR
}

func (p *PatientParameters) GetSerumCreatinine() float64 {
	if p == nil {
		return 0
	}
	return p.SerumCreatinine
}

func (p *PatientParameters) GetGender() string {
	if p == nil {
		return ""
	}
	return p.Gender
}

// BatchGetDosingRulesRequest is the request for batch rule retrieval
type BatchGetDosingRulesRequest struct {
	DrugCodes []string `json:"drug_codes"`
}

func (r *BatchGetDosingRulesRequest) GetDrugCodes() []string {
	if r == nil {
		return nil
	}
	return r.DrugCodes
}

// ValidateDosingRuleRequest is the request for rule validation
type ValidateDosingRuleRequest struct {
	Content string `json:"content"`
	Format  string `json:"format"`
}

func (r *ValidateDosingRuleRequest) GetContent() string {
	if r == nil {
		return ""
	}
	return r.Content
}

func (r *ValidateDosingRuleRequest) GetFormat() string {
	if r == nil {
		return ""
	}
	return r.Format
}

// StreamDosingUpdatesRequest is the request for streaming updates
type StreamDosingUpdatesRequest struct {
	DrugCodes []string `json:"drug_codes"`
}

func (r *StreamDosingUpdatesRequest) GetDrugCodes() []string {
	if r == nil {
		return nil
	}
	return r.DrugCodes
}

// DosingRuleResponse is the KB-1 gRPC response containing complete dosing ruleset
type DosingRuleResponse struct {
	// Transaction tracking
	TransactionId   string `json:"transaction_id"`

	// Drug identification
	DrugCode        string `json:"drug_code"`
	DrugName        string `json:"drug_name"`
	SemanticVersion string `json:"semantic_version"`
	Version         string `json:"version"`
	Status          string `json:"status"`

	// Base dosing
	DefaultDose     float64 `json:"default_dose"`
	Unit            string  `json:"unit"`
	MinDose         float64 `json:"min_dose"`
	MaxDose         float64 `json:"max_dose"`
	Frequency       string  `json:"frequency"`
	Route           string  `json:"route"`
	DosingMethod    string  `json:"dosing_method"`

	// Safety flags
	RequiresRenal   bool `json:"requires_renal"`
	RequiresHepatic bool `json:"requires_hepatic"`
	IsHighAlert     bool `json:"is_high_alert"`

	// Compiled content
	CompiledJson json.RawMessage `json:"compiled_json,omitempty"`
	Checksum     string          `json:"checksum"`

	// Clinical warnings
	Warnings []string `json:"warnings"`

	// Dose adjustments
	Adjustments []*DoseAdjustment `json:"adjustments,omitempty"`

	// Titration schedule
	Schedule *TitrationSchedule `json:"schedule,omitempty"`

	// Population-specific dosing
	Population *PopulationDosing `json:"population,omitempty"`

	// Provenance and metadata
	ProvenanceJson string        `json:"provenance_json,omitempty"`
	Metadata       *RuleMetadata `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CalculateDoseResponse is the response for dose calculation
type CalculateDoseResponse struct {
	DrugCode         string   `json:"drug_code"`
	DrugName         string   `json:"drug_name"`
	CalculatedDose   float64  `json:"calculated_dose"`
	Unit             string   `json:"unit"`
	Frequency        string   `json:"frequency"`
	Route            string   `json:"route"`
	Warnings         []string `json:"warnings"`
	Adjustments      []string `json:"adjustments"`
	CalculatedAt     string   `json:"calculated_at"`
	ConfidenceLevel  string   `json:"confidence_level"`
	RequiresReview   bool     `json:"requires_review"`
}

// BatchDosingRulesResponse is the response for batch rule retrieval
type BatchDosingRulesResponse struct {
	Rules []*DosingRuleResponse `json:"rules"`
}

// ValidationResponse is the response for rule validation
type ValidationResponse struct {
	Valid          bool     `json:"valid"`
	Errors         []string `json:"errors"`
	Warnings       []string `json:"warnings"`
	RequiredFields []string `json:"required_fields,omitempty"`
}

// DosingUpdate represents a streaming update for a dosing rule
type DosingUpdate struct {
	DrugCode   string `json:"drug_code"`
	DrugName   string `json:"drug_name"`
	Version    string `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	ChangeType string `json:"change_type"`
}

// DosingRuleRequest is the request for KB-1 gRPC endpoint
type DosingRuleRequest struct {
	TransactionId        string          `json:"transaction_id"`
	DrugCode             string          `json:"drug_code"`
	VersionPin           string          `json:"version_pin"`
	PatientContext       *PatientContext `json:"patient_context"`
	IncludeCompiledBundle bool           `json:"include_compiled_bundle"`
}

// PatientContext contains patient-specific context for dosing decisions
type PatientContext struct {
	WeightKg     float64            `json:"weight_kg"`
	Egfr         float64            `json:"egfr"`
	AgeYears     int32              `json:"age_years"`
	Sex          string             `json:"sex"`
	Pregnant     bool               `json:"pregnant"`
	ExtraNumeric map[string]float64 `json:"extra_numeric"`
}

// RuleMetadataRequest is the request for rule metadata
type RuleMetadataRequest struct {
	DrugCode string `json:"drug_code"`
}

// RuleMetadataResponse contains rule metadata
type RuleMetadataResponse struct {
	DrugCode          string        `json:"drug_code"`
	AvailableVersions []string      `json:"available_versions"`
	LatestVersion     string        `json:"latest_version"`
	Metadata          *RuleMetadata `json:"metadata"`
	SupportedRegions  []string      `json:"supported_regions"`
}

// RuleMetadata contains metadata about a rule
type RuleMetadata struct {
	SourceFile string   `json:"source_file"`
	CreatedAt  string   `json:"created_at"`
	Authors    []string `json:"authors"`
}

// AvailabilityRequest checks rule availability
type AvailabilityRequest struct {
	DrugCode string `json:"drug_code"`
}

// AvailabilityResponse indicates rule availability
type AvailabilityResponse struct {
	Available        bool     `json:"available"`
	LatestVersion    string   `json:"latest_version"`
	SupportedRegions []string `json:"supported_regions"`
	Message          string   `json:"message"`
}

// Dose adjustment types for KB-1 gRPC

// DoseAdjustment represents a dosing adjustment
type DoseAdjustment struct {
	AdjId         string `json:"adj_id"`
	AdjustType    string `json:"adjust_type"`
	Description   string `json:"description"`
	ConditionJson string `json:"condition_json"`
	FormulaJson   string `json:"formula_json"`
}

// TitrationSchedule represents a titration schedule
type TitrationSchedule struct {
	ScheduleJson json.RawMessage  `json:"schedule_json"`
	Steps        []*TitrationStep `json:"steps"`
}

// TitrationStep represents a step in titration
type TitrationStep struct {
	StepNumber         int32   `json:"step_number"`
	AfterDays          int32   `json:"after_days"`
	ActionType         string  `json:"action_type"`
	ActionValue        float64 `json:"action_value"`
	MaxStep            int32   `json:"max_step"`
	MonitoringRequired string  `json:"monitoring_required"`
}

// PopulationDosing represents population-specific dosing rules
type PopulationDosing struct {
	PopulationJson json.RawMessage   `json:"population_json"`
	Rules          []*PopulationRule `json:"rules"`
}

// PopulationRule represents a rule for a specific population
type PopulationRule struct {
	PopId            string  `json:"pop_id"`
	PopulationType   string  `json:"population_type"`
	AgeMin           int32   `json:"age_min"`
	AgeMax           int32   `json:"age_max"`
	WeightMin        float64 `json:"weight_min"`
	WeightMax        float64 `json:"weight_max"`
	FormulaJson      string  `json:"formula_json"`
	SafetyLimitsJson string  `json:"safety_limits_json"`
}
