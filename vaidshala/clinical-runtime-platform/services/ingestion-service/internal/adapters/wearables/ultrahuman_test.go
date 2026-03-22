package wearables

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// generateCGMReadings produces a sinusoidal glucose pattern centred on
// baseMgDL with the given amplitude, one reading every 5 minutes.
func generateCGMReadings(count int, baseMgDL, amplitude float64) []CGMReading {
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	readings := make([]CGMReading, count)
	for i := 0; i < count; i++ {
		t := start.Add(time.Duration(i) * 5 * time.Minute)
		// Sinusoidal pattern: base + amplitude * sin(2*pi*i/count)
		glucose := baseMgDL + amplitude*math.Sin(2*math.Pi*float64(i)/float64(count))
		readings[i] = CGMReading{
			Timestamp:   t,
			GlucoseMgDL: glucose,
		}
	}
	return readings
}

func TestAggregateCGM_WellControlled(t *testing.T) {
	// 288 readings (24h), centred at 110 mg/dL with small amplitude.
	readings := generateCGMReadings(288, 110, 20)

	agg := AggregateCGM(readings)

	if agg.ReadingCount != 288 {
		t.Errorf("expected 288 readings, got %d", agg.ReadingCount)
	}
	if agg.TIR < 90 {
		t.Errorf("expected TIR > 90%%, got %.2f%%", agg.TIR)
	}
	if agg.TBR > 5 {
		t.Errorf("expected TBR < 5%%, got %.2f%%", agg.TBR)
	}
	if agg.CV > CGMCVTarget {
		t.Errorf("expected CV < %.0f%%, got %.2f%%", CGMCVTarget, agg.CV)
	}
	if agg.MeanGlucose < 100 || agg.MeanGlucose > 120 {
		t.Errorf("expected mean glucose ~110, got %.2f", agg.MeanGlucose)
	}
}

func TestAggregateCGM_PoorlyControlled(t *testing.T) {
	// High base glucose with large swings → high TAR and mean.
	readings := generateCGMReadings(288, 200, 40)

	agg := AggregateCGM(readings)

	if agg.TAR < 30 {
		t.Errorf("expected TAR > 30%%, got %.2f%%", agg.TAR)
	}
	if agg.MeanGlucose < 150 {
		t.Errorf("expected mean glucose > 150, got %.2f", agg.MeanGlucose)
	}
}

func TestAggregateCGM_HypoglycemiaRisk(t *testing.T) {
	// Low base glucose → significant time below range.
	readings := generateCGMReadings(288, 75, 15)

	agg := AggregateCGM(readings)

	if agg.TBR < 5 {
		t.Errorf("expected TBR > 5%%, got %.2f%%", agg.TBR)
	}
}

func TestAggregateCGM_MAGCalculation(t *testing.T) {
	// 12 readings with known oscillation → MAG > 0.
	readings := generateCGMReadings(12, 120, 30)

	agg := AggregateCGM(readings)

	if agg.MAG <= 0 {
		t.Errorf("expected MAG > 0, got %.2f", agg.MAG)
	}
	if agg.ReadingCount != 12 {
		t.Errorf("expected 12 readings, got %d", agg.ReadingCount)
	}
}

func TestUltrahumanAdapter_Convert(t *testing.T) {
	adapter := &UltrahumanAdapter{}

	readings := generateCGMReadings(288, 110, 20)
	now := time.Now()

	payload := UltrahumanCGMPayload{
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		DeviceID:    "uh-device-001",
		SensorID:    "sensor-abc",
		Readings:    readings,
		PeriodStart: now.Add(-24 * time.Hour),
		PeriodEnd:   now,
	}

	obs, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs) != 7 {
		t.Fatalf("expected 7 observations, got %d", len(obs))
	}

	for _, o := range obs {
		if o.SourceType != canonical.SourceWearable {
			t.Errorf("expected source WEARABLE, got %s", o.SourceType)
		}
		if o.DeviceContext == nil {
			t.Error("expected device context to be set")
			continue
		}
		if o.DeviceContext.Manufacturer != "Ultrahuman" {
			t.Errorf("expected manufacturer Ultrahuman, got %s", o.DeviceContext.Manufacturer)
		}
	}
}

func TestUltrahumanAdapter_InsufficientReadings(t *testing.T) {
	adapter := &UltrahumanAdapter{}

	payload := UltrahumanCGMPayload{
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		DeviceID:    "uh-device-002",
		SensorID:    "sensor-xyz",
		Readings:    generateCGMReadings(5, 110, 10),
		PeriodStart: time.Now().Add(-1 * time.Hour),
		PeriodEnd:   time.Now(),
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Fatal("expected error for insufficient readings, got nil")
	}
}

func TestQualityFromReadingCount(t *testing.T) {
	tests := []struct {
		count    int
		expected float64
	}{
		{288, 0.95},
		{200, 0.85},
		{100, 0.75},
		{50, 0.60},
	}

	for _, tt := range tests {
		got := qualityFromReadingCount(tt.count)
		if got != tt.expected {
			t.Errorf("qualityFromReadingCount(%d) = %.2f, want %.2f", tt.count, got, tt.expected)
		}
	}
}
