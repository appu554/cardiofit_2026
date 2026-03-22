package labs

import (
	"context"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
)

type LabAdapter interface {
	LabID() string
	Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error)
	ValidateWebhookAuth(apiKey string) bool
}

type LabResult struct {
	LabTestCode  string    `json:"lab_test_code"`
	TestName     string    `json:"test_name"`
	Value        float64   `json:"value"`
	ValueString  string    `json:"value_string,omitempty"`
	Unit         string    `json:"unit"`
	ReferenceMin *float64  `json:"reference_min,omitempty"`
	ReferenceMax *float64  `json:"reference_max,omitempty"`
	IsAbnormal   bool      `json:"is_abnormal"`
	SampleType   string    `json:"sample_type,omitempty"`
	CollectedAt  time.Time `json:"collected_at"`
	ReportedAt   time.Time `json:"reported_at"`
}

type LabReport struct {
	ReportID     string      `json:"report_id"`
	LabID        string      `json:"lab_id"`
	PatientID    *uuid.UUID  `json:"patient_id,omitempty"`
	PatientPhone string      `json:"patient_phone,omitempty"`
	PatientName  string      `json:"patient_name,omitempty"`
	ABHANumber   string      `json:"abha_number,omitempty"`
	Results      []LabResult `json:"results"`
	OrderID      string      `json:"order_id,omitempty"`
	CollectedAt  time.Time   `json:"collected_at"`
	ReportedAt   time.Time   `json:"reported_at"`
	RawPayload   []byte      `json:"raw_payload,omitempty"`
}
