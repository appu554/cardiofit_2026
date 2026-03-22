package labs

import (
	"context"
	"testing"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"go.uber.org/zap"
)

func TestGenericCSVAdapter_Parse(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewGenericCSVAdapter("test_lab", &mockCodeRegistry{}, logger)

	csv := `test_code,test_name,value,unit,sample_date
TSH,TSH,2.5,mIU/L,2026-03-15
HBA1C,HbA1c,7.2,%,2026-03-15
FBG,Fasting Glucose,145,mg/dL,2026-03-15`

	observations, err := adapter.Parse(context.Background(), []byte(csv))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(observations) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(observations))
	}

	for _, obs := range observations {
		if obs.SourceType != canonical.SourceLab {
			t.Errorf("expected LAB source, got %s", obs.SourceType)
		}
		if obs.SourceID != "test_lab" {
			t.Errorf("expected test_lab source ID, got %s", obs.SourceID)
		}
	}
}

func TestGenericCSVAdapter_MissingColumn(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewGenericCSVAdapter("test_lab", &mockCodeRegistry{}, logger)

	csv := `test_code,test_name,value
TSH,TSH,2.5`

	_, err := adapter.Parse(context.Background(), []byte(csv))
	if err == nil {
		t.Error("expected error for missing columns")
	}
}

func TestGenericCSVAdapter_EmptyCSV(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewGenericCSVAdapter("test_lab", &mockCodeRegistry{}, logger)

	csv := `test_code,test_name,value,unit,sample_date`

	_, err := adapter.Parse(context.Background(), []byte(csv))
	if err == nil {
		t.Error("expected error for empty CSV")
	}
}
