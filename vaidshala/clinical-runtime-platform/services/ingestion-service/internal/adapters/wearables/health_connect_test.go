package wearables

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func TestHealthConnectAdapter_BloodPressure(t *testing.T) {
	adapter := &HealthConnectAdapter{}

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "hc-device-001",
		Records: []HealthConnectRecord{
			{
				RecordType:  "BloodPressure",
				PackageName: "com.google.android.apps.fitness",
				DeviceModel: "Pixel 8",
				StartTime:   time.Now(),
				EndTime:     time.Now(),
				Values:      map[string]float64{"systolic": 120, "diastolic": 80},
				Unit:        "mmHg",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(obs))
	}

	// Systolic
	if obs[0].Value != 120 {
		t.Errorf("expected systolic value 120, got %f", obs[0].Value)
	}
	if obs[0].LOINCCode != "8480-6" {
		t.Errorf("expected systolic LOINC 8480-6, got %s", obs[0].LOINCCode)
	}
	if obs[0].Unit != "mmHg" {
		t.Errorf("expected unit mmHg, got %s", obs[0].Unit)
	}

	// Diastolic
	if obs[1].Value != 80 {
		t.Errorf("expected diastolic value 80, got %f", obs[1].Value)
	}
	if obs[1].LOINCCode != "8462-4" {
		t.Errorf("expected diastolic LOINC 8462-4, got %s", obs[1].LOINCCode)
	}
}

func TestHealthConnectAdapter_HeartRate(t *testing.T) {
	adapter := &HealthConnectAdapter{}

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "hc-device-002",
		Records: []HealthConnectRecord{
			{
				RecordType:  "HeartRate",
				PackageName: "com.samsung.health",
				DeviceModel: "Galaxy Watch 6",
				StartTime:   time.Now(),
				EndTime:     time.Now(),
				Values:      map[string]float64{"value": 72},
				Unit:        "beats/min",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}

	if obs[0].LOINCCode != "8867-4" {
		t.Errorf("expected LOINC 8867-4, got %s", obs[0].LOINCCode)
	}
	if obs[0].Unit != "beats/min" {
		t.Errorf("expected unit beats/min, got %s", obs[0].Unit)
	}
	if obs[0].Value != 72 {
		t.Errorf("expected value 72, got %f", obs[0].Value)
	}
}

func TestHealthConnectAdapter_Steps(t *testing.T) {
	adapter := &HealthConnectAdapter{}

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "hc-device-003",
		Records: []HealthConnectRecord{
			{
				RecordType:  "Steps",
				PackageName: "com.google.android.apps.fitness",
				DeviceModel: "Pixel Watch 2",
				StartTime:   time.Now(),
				EndTime:     time.Now(),
				Values:      map[string]float64{"value": 8500},
				Unit:        "steps",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}

	if obs[0].Value != 8500 {
		t.Errorf("expected 8500 steps, got %f", obs[0].Value)
	}
	if obs[0].SourceType != canonical.SourceWearable {
		t.Errorf("expected source WEARABLE, got %s", obs[0].SourceType)
	}
}

func TestHealthConnectAdapter_UnsupportedType(t *testing.T) {
	adapter := &HealthConnectAdapter{}

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "hc-device-004",
		Records: []HealthConnectRecord{
			{
				RecordType: "UnsupportedMetric",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				Values:     map[string]float64{"value": 42},
				Unit:       "unknown",
			},
		},
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Fatal("expected error for unsupported record type, got nil")
	}
}

func TestHealthConnectAdapter_EmptyPayload(t *testing.T) {
	adapter := &HealthConnectAdapter{}

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "hc-device-005",
		Records:   []HealthConnectRecord{},
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Fatal("expected error for empty payload, got nil")
	}
}
