package ehr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockSFTPPoller implements SFTPPoller for testing without a real SFTP server.
type mockSFTPPoller struct {
	files map[string][]byte
	moved []string
}

func (m *mockSFTPPoller) ListFiles(_ context.Context, dir string) ([]string, error) {
	var names []string
	for name := range m.files {
		names = append(names, dir+"/"+name)
	}
	return names, nil
}

func (m *mockSFTPPoller) ReadFile(_ context.Context, path string) ([]byte, error) {
	// Extract the filename from the full path.
	for name, data := range m.files {
		if len(path) >= len(name) && path[len(path)-len(name):] == name {
			return data, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (m *mockSFTPPoller) MoveToProcessed(_ context.Context, src string) error {
	m.moved = append(m.moved, src)
	return nil
}

func TestSFTPAdapter_PollOnce(t *testing.T) {
	csv := `test_code,value,unit,sample_date
4548-4,7.2,%,2026-03-20
33914-3,62,mL/min/1.73m2,2026-03-20
14771-0,5.8,mmol/L,2026-03-20
`
	poller := &mockSFTPPoller{
		files: map[string][]byte{
			"results_2026-03-20.csv": []byte(csv),
		},
	}

	config := SFTPSourceConfig{
		HospitalID:   "hospital-alpha",
		RemoteDir:    "/data/outbound",
		FilePattern:  "*.csv",
		PollInterval: 1 * time.Minute,
	}

	adapter := NewSFTPAdapter(config, poller, zap.NewNop())
	results := adapter.pollOnce(context.Background())

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if r.HospitalID != "hospital-alpha" {
		t.Errorf("expected hospital-alpha, got %s", r.HospitalID)
	}
	if len(r.Observations) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(r.Observations))
	}

	// Verify first observation.
	obs := r.Observations[0]
	if obs.LOINCCode != "4548-4" {
		t.Errorf("expected LOINC 4548-4, got %s", obs.LOINCCode)
	}
	if obs.Value != 7.2 {
		t.Errorf("expected value 7.2, got %f", obs.Value)
	}
}

func TestSFTPAdapter_PollOnce_MissingColumn(t *testing.T) {
	// CSV missing "unit" and "sample_date" columns.
	csv := `test_code,value
4548-4,7.2
`
	poller := &mockSFTPPoller{
		files: map[string][]byte{
			"bad.csv": []byte(csv),
		},
	}

	config := SFTPSourceConfig{
		HospitalID:   "hospital-beta",
		RemoteDir:    "/data/outbound",
		FilePattern:  "*.csv",
		PollInterval: 1 * time.Minute,
	}

	adapter := NewSFTPAdapter(config, poller, zap.NewNop())
	results := adapter.pollOnce(context.Background())

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error == nil {
		t.Fatal("expected error for missing columns")
	}
}

func TestSFTPAdapter_FilePatternFilter(t *testing.T) {
	poller := &mockSFTPPoller{
		files: map[string][]byte{
			"results.csv": []byte("test_code,value,unit,sample_date\n4548-4,7.2,%,2026-03-20\n"),
			"notes.txt":   []byte("some notes"),
			"data.csv":    []byte("test_code,value,unit,sample_date\n33914-3,62,mL/min,2026-03-20\n"),
		},
	}

	config := SFTPSourceConfig{
		HospitalID:   "hospital-gamma",
		RemoteDir:    "/data/outbound",
		FilePattern:  "*.csv",
		PollInterval: 1 * time.Minute,
	}

	adapter := NewSFTPAdapter(config, poller, zap.NewNop())
	results := adapter.pollOnce(context.Background())

	// Only .csv files should be processed (notes.txt skipped).
	totalObs := 0
	for _, r := range results {
		if r.Error != nil {
			t.Errorf("unexpected error for file %s: %v", r.Filename, r.Error)
			continue
		}
		totalObs += len(r.Observations)
	}

	if totalObs != 2 {
		t.Errorf("expected 2 observations from CSV files only, got %d", totalObs)
	}

	// Verify notes.txt was not processed (should not appear in results).
	for _, r := range results {
		if r.Filename != "" && r.Filename[len(r.Filename)-4:] == ".txt" {
			t.Error("txt file should not have been processed")
		}
	}
}
