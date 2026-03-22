package ehr

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const defaultPollInterval = 15 * time.Minute

// SFTPSourceConfig describes an SFTP endpoint to poll for CSV lab/observation
// data from a hospital EHR.
type SFTPSourceConfig struct {
	HospitalID   string        `json:"hospital_id"`
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Username     string        `json:"username"`
	Password     string        `json:"password,omitempty"`
	KeyPath      string        `json:"key_path,omitempty"`
	RemoteDir    string        `json:"remote_dir"`
	FilePattern  string        `json:"file_pattern"`
	Template     string        `json:"template,omitempty"`
	PollInterval time.Duration `json:"poll_interval"`
}

// SFTPPoller abstracts the SFTP filesystem operations so the adapter can be
// tested without a real SFTP server.
type SFTPPoller interface {
	ListFiles(ctx context.Context, dir string) ([]string, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
	MoveToProcessed(ctx context.Context, src string) error
}

// PollResult carries the outcome of polling a single file.
type PollResult struct {
	HospitalID   string
	Filename     string
	Observations []canonical.CanonicalObservation
	Error        error
}

// SFTPAdapter polls configured SFTP servers for CSV files and converts them
// into canonical observations.
type SFTPAdapter struct {
	config SFTPSourceConfig
	poller SFTPPoller
	logger *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSFTPAdapter creates a new SFTPAdapter for the given config and poller.
func NewSFTPAdapter(config SFTPSourceConfig, poller SFTPPoller, logger *zap.Logger) *SFTPAdapter {
	if config.PollInterval <= 0 {
		config.PollInterval = defaultPollInterval
	}
	return &SFTPAdapter{
		config: config,
		poller: poller,
		logger: logger,
	}
}

// Start begins the background poll loop. The handler function is called for
// each successfully polled result.
func (a *SFTPAdapter) Start(ctx context.Context, handler func(PollResult)) {
	ctx, a.cancel = context.WithCancel(ctx)
	a.wg.Add(1)
	go a.pollLoop(ctx, handler)
}

// Stop signals the poll loop to exit and waits for it to finish.
func (a *SFTPAdapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
}

// pollLoop runs at the configured interval until the context is cancelled.
func (a *SFTPAdapter) pollLoop(ctx context.Context, handler func(PollResult)) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	// Run an immediate poll on start.
	results := a.pollOnce(ctx)
	for _, r := range results {
		handler(r)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			results := a.pollOnce(ctx)
			for _, r := range results {
				handler(r)
			}
		}
	}
}

// pollOnce lists files from the SFTP remote directory, filters by the
// configured pattern, parses matching CSV files, and moves them to processed.
func (a *SFTPAdapter) pollOnce(ctx context.Context) []PollResult {
	files, err := a.poller.ListFiles(ctx, a.config.RemoteDir)
	if err != nil {
		a.logger.Error("SFTP list files failed",
			zap.String("hospital_id", a.config.HospitalID),
			zap.Error(err),
		)
		return []PollResult{{
			HospitalID: a.config.HospitalID,
			Error:      fmt.Errorf("list files: %w", err),
		}}
	}

	var results []PollResult

	for _, file := range files {
		// Apply file pattern filter.
		if a.config.FilePattern != "" {
			matched, matchErr := filepath.Match(a.config.FilePattern, filepath.Base(file))
			if matchErr != nil || !matched {
				continue
			}
		}

		data, readErr := a.poller.ReadFile(ctx, file)
		if readErr != nil {
			a.logger.Error("SFTP read file failed",
				zap.String("hospital_id", a.config.HospitalID),
				zap.String("file", file),
				zap.Error(readErr),
			)
			results = append(results, PollResult{
				HospitalID: a.config.HospitalID,
				Filename:   file,
				Error:      readErr,
			})
			continue
		}

		observations, parseErr := a.parseCSV(data)
		if parseErr != nil {
			a.logger.Error("CSV parse failed",
				zap.String("hospital_id", a.config.HospitalID),
				zap.String("file", file),
				zap.Error(parseErr),
			)
			results = append(results, PollResult{
				HospitalID: a.config.HospitalID,
				Filename:   file,
				Error:      parseErr,
			})
			continue
		}

		// Move to processed.
		if moveErr := a.poller.MoveToProcessed(ctx, file); moveErr != nil {
			a.logger.Warn("SFTP move to processed failed",
				zap.String("file", file),
				zap.Error(moveErr),
			)
		}

		a.logger.Info("SFTP file processed",
			zap.String("hospital_id", a.config.HospitalID),
			zap.String("file", file),
			zap.Int("observation_count", len(observations)),
		)

		results = append(results, PollResult{
			HospitalID:   a.config.HospitalID,
			Filename:     file,
			Observations: observations,
		})
	}

	return results
}

// parseCSV reads a CSV file with required columns: test_code, value, unit,
// sample_date and converts each row into a CanonicalObservation.
func (a *SFTPAdapter) parseCSV(data []byte) ([]canonical.CanonicalObservation, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true

	// Read header row.
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	// Build column index.
	colIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		colIndex[strings.TrimSpace(strings.ToLower(h))] = i
	}

	// Validate required columns.
	required := []string{"test_code", "value", "unit", "sample_date"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required CSV column: %s", col)
		}
	}

	var observations []canonical.CanonicalObservation

	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read CSV row: %w", readErr)
		}

		testCode := strings.TrimSpace(record[colIndex["test_code"]])
		valueStr := strings.TrimSpace(record[colIndex["value"]])
		unit := strings.TrimSpace(record[colIndex["unit"]])
		dateStr := strings.TrimSpace(record[colIndex["sample_date"]])

		value, parseErr := strconv.ParseFloat(valueStr, 64)
		if parseErr != nil {
			a.logger.Warn("skipping CSV row: invalid value",
				zap.String("value", valueStr),
				zap.Error(parseErr),
			)
			continue
		}

		ts, parseErr := time.Parse("2006-01-02", dateStr)
		if parseErr != nil {
			// Try RFC3339.
			ts, parseErr = time.Parse(time.RFC3339, dateStr)
			if parseErr != nil {
				a.logger.Warn("skipping CSV row: invalid sample_date",
					zap.String("sample_date", dateStr),
					zap.Error(parseErr),
				)
				continue
			}
		}

		observations = append(observations, canonical.CanonicalObservation{
			ID:              uuid.New(),
			SourceType:      canonical.SourceEHR,
			SourceID:        "sftp_" + a.config.HospitalID,
			ObservationType: classifyByLOINC(testCode),
			LOINCCode:       testCode,
			Value:           value,
			Unit:            unit,
			Timestamp:       ts,
			QualityScore:    0.80,
		})
	}

	return observations, nil
}
