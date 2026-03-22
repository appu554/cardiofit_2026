package wearables

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func TestAppleHealthAdapter_HeartRate(t *testing.T) {
	adapter := &AppleHealthAdapter{}

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "apple-watch-001",
		Samples: []AppleHealthSample{
			{
				SampleType:  "HKQuantityTypeIdentifierHeartRate",
				Value:       68,
				Unit:        "beats/min",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Apple Watch",
				DeviceModel: "Apple Watch Series 9",
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

	if obs[0].Value != 68 {
		t.Errorf("expected value 68, got %f", obs[0].Value)
	}
	if obs[0].LOINCCode != "8867-4" {
		t.Errorf("expected LOINC 8867-4, got %s", obs[0].LOINCCode)
	}
	if obs[0].QualityScore != 0.90 {
		t.Errorf("expected quality score 0.90, got %f", obs[0].QualityScore)
	}
	if obs[0].SourceType != canonical.SourceWearable {
		t.Errorf("expected source WEARABLE, got %s", obs[0].SourceType)
	}
}

func TestAppleHealthAdapter_BloodGlucose_MmolConversion(t *testing.T) {
	adapter := &AppleHealthAdapter{}

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "iphone-001",
		Samples: []AppleHealthSample{
			{
				SampleType:  "HKQuantityTypeIdentifierBloodGlucose",
				Value:       6.5,
				Unit:        "mmol/L",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Dexcom G7",
				DeviceModel: "iPhone 15",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 6.5 mmol/L * 18.0182 ≈ 117.12 mg/dL
	expected := 6.5 * 18.0182
	if math.Abs(obs[0].Value-expected) > 0.5 {
		t.Errorf("expected ~%.1f mg/dL, got %.2f", expected, obs[0].Value)
	}
	if obs[0].Unit != "mg/dL" {
		t.Errorf("expected unit mg/dL, got %s", obs[0].Unit)
	}
}

func TestAppleHealthAdapter_WeightLbsConversion(t *testing.T) {
	adapter := &AppleHealthAdapter{}

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "iphone-002",
		Samples: []AppleHealthSample{
			{
				SampleType:  "HKQuantityTypeIdentifierBodyMass",
				Value:       176,
				Unit:        "lbs",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Withings Body+",
				DeviceModel: "iPhone 15 Pro",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 176 lbs / 2.20462 ≈ 79.83 kg
	expected := 176.0 / 2.20462
	if math.Abs(obs[0].Value-expected) > 0.5 {
		t.Errorf("expected ~%.1f kg, got %.2f", expected, obs[0].Value)
	}
	if obs[0].Unit != "kg" {
		t.Errorf("expected unit kg, got %s", obs[0].Unit)
	}
}

func TestAppleHealthAdapter_MultipleSamples(t *testing.T) {
	adapter := &AppleHealthAdapter{}

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "apple-watch-002",
		Samples: []AppleHealthSample{
			{
				SampleType:  "HKQuantityTypeIdentifierHeartRate",
				Value:       72,
				Unit:        "beats/min",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Apple Watch",
				DeviceModel: "Apple Watch Ultra 2",
			},
			{
				SampleType:  "HKQuantityTypeIdentifierStepCount",
				Value:       10500,
				Unit:        "steps",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Apple Watch",
				DeviceModel: "Apple Watch Ultra 2",
			},
			{
				SampleType:  "HKQuantityTypeIdentifierOxygenSaturation",
				Value:       98,
				Unit:        "%",
				StartDate:   time.Now(),
				EndDate:     time.Now(),
				SourceName:  "Apple Watch",
				DeviceModel: "Apple Watch Ultra 2",
			},
		},
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(obs))
	}
}

func TestAppleHealthAdapter_UnsupportedType(t *testing.T) {
	adapter := &AppleHealthAdapter{}

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "iphone-003",
		Samples: []AppleHealthSample{
			{
				SampleType: "HKQuantityTypeIdentifierUnsupported",
				Value:      42,
				Unit:       "unknown",
				StartDate:  time.Now(),
				EndDate:    time.Now(),
			},
		},
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Fatal("expected error for unsupported sample type, got nil")
	}
}

func TestConvertAppleHealthUnit_Fahrenheit(t *testing.T) {
	// 98.6°F → 37°C
	value, unit := convertAppleHealthUnit(98.6, "degF")

	if unit != "°C" {
		t.Errorf("expected unit °C, got %s", unit)
	}

	expected := (98.6 - 32) * 5.0 / 9.0
	if math.Abs(value-expected) > 0.01 {
		t.Errorf("expected ~%.2f°C, got %.2f", expected, value)
	}
}
