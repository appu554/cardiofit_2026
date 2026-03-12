// Package dailymed provides delta synchronization for SPL documents.
//
// Phase 3a.3: Delta Syncer for DailyMed SPL
// Key Feature: Incrementally sync only changed SPL documents
//
// Sync Strategies:
// - DAILY: Sync documents published since yesterday
// - WEEKLY: Sync documents published in last 7 days
// - MONTHLY: Full comparison scan (hash-based)
// - FULL: Re-download and re-parse everything
package dailymed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// SYNC CONFIGURATION
// =============================================================================

// SyncStrategy defines the delta sync approach
type SyncStrategy string

const (
	// SyncDaily fetches documents published since yesterday
	SyncDaily SyncStrategy = "DAILY"

	// SyncWeekly fetches documents published in last 7 days
	SyncWeekly SyncStrategy = "WEEKLY"

	// SyncMonthly performs full hash comparison
	SyncMonthly SyncStrategy = "MONTHLY"

	// SyncFull re-downloads everything
	SyncFull SyncStrategy = "FULL"
)

// SyncConfig contains configuration for delta sync
type SyncConfig struct {
	Strategy          SyncStrategy
	BatchSize         int           // Documents per batch
	MaxConcurrency    int           // Parallel fetches
	SkipUnchanged     bool          // Skip if hash matches
	ProcessSections   bool          // Route sections after fetch
	EmitEvents        bool          // Emit Kafka events
	RetryFailures     bool          // Retry previously failed documents
	MaxRetries        int           // Max retries per document
	RequestTimeout    time.Duration // HTTP timeout per request
	ProgressCallback  func(SyncProgress)
}

// DefaultSyncConfig returns sensible defaults
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		Strategy:        SyncDaily,
		BatchSize:       100,
		MaxConcurrency:  5,
		SkipUnchanged:   true,
		ProcessSections: true,
		EmitEvents:      false,
		RetryFailures:   true,
		MaxRetries:      3,
		RequestTimeout:  60 * time.Second,
	}
}

// =============================================================================
// SYNC RESULT
// =============================================================================

// SyncResult contains the outcome of a sync operation
type SyncResult struct {
	SyncID            uuid.UUID
	Strategy          SyncStrategy
	StartedAt         time.Time
	CompletedAt       time.Time
	Duration          time.Duration

	// Counts
	TotalFound        int
	DocumentsSynced   int
	DocumentsSkipped  int
	DocumentsFailed   int
	SectionsProcessed int
	TablesExtracted   int

	// Details
	SyncedSetIDs      []string
	SkippedSetIDs     []string
	FailedDocuments   []FailedDocument
	Errors            []string
}

// FailedDocument contains information about a failed sync
type FailedDocument struct {
	SetID       string
	Version     int
	Error       string
	Attempts    int
	LastAttempt time.Time
}

// SyncProgress contains progress information during sync
type SyncProgress struct {
	Processed   int
	Total       int
	CurrentDoc  string
	Stage       string // "LISTING", "FETCHING", "PARSING", "STORING"
	StartedAt   time.Time
	EstimatedETA time.Duration
}

// =============================================================================
// DELTA SYNCER
// =============================================================================

// DeltaSyncer handles incremental synchronization of SPL documents
type DeltaSyncer struct {
	fetcher        *SPLFetcher
	storage        *StorageManager
	router         *SectionRouter
	config         SyncConfig
	httpClient     *http.Client
	baseURL        string
	mu             sync.Mutex
	currentSync    *SyncResult
}

// NewDeltaSyncer creates a new delta syncer
func NewDeltaSyncer(fetcher *SPLFetcher, storage *StorageManager, config SyncConfig) *DeltaSyncer {
	return &DeltaSyncer{
		fetcher:    fetcher,
		storage:    storage,
		router:     NewSectionRouter(),
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		baseURL:    "https://dailymed.nlm.nih.gov/dailymed",
	}
}

// =============================================================================
// SYNC OPERATIONS
// =============================================================================

// Sync performs delta synchronization based on the configured strategy
func (ds *DeltaSyncer) Sync(ctx context.Context) (*SyncResult, error) {
	ds.mu.Lock()
	result := &SyncResult{
		SyncID:    uuid.New(),
		Strategy:  ds.config.Strategy,
		StartedAt: time.Now(),
	}
	ds.currentSync = result
	ds.mu.Unlock()

	defer func() {
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
	}()

	// Get the time range based on strategy
	since, err := ds.getSyncSince(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("getting sync time: %v", err))
		return result, err
	}

	// List documents to sync
	documents, err := ds.listDocumentsToSync(ctx, since)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("listing documents: %v", err))
		return result, err
	}

	result.TotalFound = len(documents)
	ds.reportProgress(SyncProgress{
		Total:     result.TotalFound,
		Stage:     "LISTING",
		StartedAt: result.StartedAt,
	})

	// Process documents in batches
	for i := 0; i < len(documents); i += ds.config.BatchSize {
		end := i + ds.config.BatchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		ds.processBatch(ctx, batch, result)

		// Check for cancellation
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "sync cancelled")
			return result, ctx.Err()
		default:
		}
	}

	return result, nil
}

// getSyncSince returns the start time for delta sync based on strategy
func (ds *DeltaSyncer) getSyncSince(ctx context.Context) (time.Time, error) {
	switch ds.config.Strategy {
	case SyncDaily:
		return time.Now().AddDate(0, 0, -1), nil

	case SyncWeekly:
		return time.Now().AddDate(0, 0, -7), nil

	case SyncMonthly:
		return time.Now().AddDate(0, -1, 0), nil

	case SyncFull:
		return time.Time{}, nil // No filter

	default:
		// Use last sync time from database
		return ds.storage.GetLastSyncTime(ctx)
	}
}

// listDocumentsToSync fetches the list of documents updated since the given time
func (ds *DeltaSyncer) listDocumentsToSync(ctx context.Context, since time.Time) ([]SPLListItem, error) {
	var allDocuments []SPLListItem
	page := 1
	pageSize := 100

	for {
		items, hasMore, err := ds.fetchPage(ctx, since, page, pageSize)
		if err != nil {
			return nil, fmt.Errorf("fetching page %d: %w", page, err)
		}

		allDocuments = append(allDocuments, items...)

		if !hasMore {
			break
		}
		page++

		// Safety limit
		if page > 1000 {
			break
		}
	}

	return allDocuments, nil
}

// SPLListItem represents an item in the SPL list response
type SPLListItem struct {
	SetID         string `json:"setid"`
	Title         string `json:"title"`
	ProductNDC    string `json:"product_ndc"`
	PublishedDate string `json:"published_date"`
	ProductType   string `json:"product_type"`
	Labeler       string `json:"labeler"`
}

// fetchPage fetches a page of SPL documents from the API
func (ds *DeltaSyncer) fetchPage(ctx context.Context, since time.Time, page, pageSize int) ([]SPLListItem, bool, error) {
	// Build URL with parameters
	u, _ := url.Parse(ds.baseURL + "/services/v2/spls.json")
	q := u.Query()
	q.Set("pagesize", strconv.Itoa(pageSize))
	q.Set("page", strconv.Itoa(page))

	if !since.IsZero() {
		q.Set("published_date", since.Format("2006-01-02"))
		q.Set("published_date_comparison", "gte")
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data     []SPLListItem `json:"data"`
		Metadata struct {
			TotalElements int  `json:"total_elements"`
			Elements      int  `json:"elements"`
			TotalPages    int  `json:"total_pages"`
			CurrentPage   int  `json:"current_page"`
			NextPageURL   string `json:"next_page_url"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, fmt.Errorf("parsing response: %w", err)
	}

	hasMore := result.Metadata.NextPageURL != ""
	return result.Data, hasMore, nil
}

// processBatch processes a batch of documents concurrently
func (ds *DeltaSyncer) processBatch(ctx context.Context, batch []SPLListItem, result *SyncResult) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, ds.config.MaxConcurrency)
	var mu sync.Mutex

	for _, item := range batch {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(item SPLListItem) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			synced, skipped, failed := ds.processDocument(ctx, item)

			mu.Lock()
			if synced {
				result.DocumentsSynced++
				result.SyncedSetIDs = append(result.SyncedSetIDs, item.SetID)
			}
			if skipped {
				result.DocumentsSkipped++
				result.SkippedSetIDs = append(result.SkippedSetIDs, item.SetID)
			}
			if failed != nil {
				result.DocumentsFailed++
				result.FailedDocuments = append(result.FailedDocuments, *failed)
			}
			mu.Unlock()

			// Report progress
			ds.reportProgress(SyncProgress{
				Processed:  result.DocumentsSynced + result.DocumentsSkipped + result.DocumentsFailed,
				Total:      result.TotalFound,
				CurrentDoc: item.SetID,
				Stage:      "FETCHING",
				StartedAt:  result.StartedAt,
			})

		}(item)
	}

	wg.Wait()
}

// processDocument processes a single document
func (ds *DeltaSyncer) processDocument(ctx context.Context, item SPLListItem) (synced bool, skipped bool, failed *FailedDocument) {
	// Check if document already exists with same hash
	if ds.config.SkipUnchanged {
		existingHash, exists, err := ds.storage.GetDocumentHashBySetID(ctx, item.SetID, 0) // 0 = latest
		if err == nil && exists {
			// Fetch just the header to compare
			doc, err := ds.fetcher.FetchBySetID(ctx, item.SetID)
			if err == nil && doc.ContentHash == existingHash {
				return false, true, nil // Skip unchanged
			}
		}
	}

	// Fetch the full document
	var lastErr error
	for attempt := 1; attempt <= ds.config.MaxRetries; attempt++ {
		doc, err := ds.fetcher.FetchBySetID(ctx, item.SetID)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second) // Backoff
			continue
		}

		// Store the document
		sourceDoc, err := ds.storage.SaveDocument(ctx, doc, doc.RawXML, "DELTA")
		if err != nil {
			lastErr = err
			continue
		}

		// Process sections if enabled
		if ds.config.ProcessSections {
			routedSections := ds.router.RouteDocument(doc)

			err = ds.storage.SaveRoutedSections(ctx, sourceDoc.ID, routedSections)
			if err != nil {
				lastErr = err
				continue
			}

			// Update document status
			ds.storage.UpdateDocumentStatus(ctx, sourceDoc.ID, "PARSED", "")
		}

		return true, false, nil // Success
	}

	// All retries failed
	return false, false, &FailedDocument{
		SetID:       item.SetID,
		Error:       lastErr.Error(),
		Attempts:    ds.config.MaxRetries,
		LastAttempt: time.Now(),
	}
}

// reportProgress calls the progress callback if configured
func (ds *DeltaSyncer) reportProgress(progress SyncProgress) {
	if ds.config.ProgressCallback != nil {
		// Calculate ETA
		if progress.Processed > 0 {
			elapsed := time.Since(progress.StartedAt)
			avgTime := elapsed / time.Duration(progress.Processed)
			remaining := progress.Total - progress.Processed
			progress.EstimatedETA = avgTime * time.Duration(remaining)
		}

		ds.config.ProgressCallback(progress)
	}
}

// =============================================================================
// TARGETED SYNC OPERATIONS
// =============================================================================

// SyncBySetID syncs a specific document by SetID
func (ds *DeltaSyncer) SyncBySetID(ctx context.Context, setID string) (*SyncResult, error) {
	result := &SyncResult{
		SyncID:    uuid.New(),
		Strategy:  "TARGETED",
		StartedAt: time.Now(),
		TotalFound: 1,
	}

	doc, err := ds.fetcher.FetchBySetID(ctx, setID)
	if err != nil {
		result.FailedDocuments = append(result.FailedDocuments, FailedDocument{
			SetID:       setID,
			Error:       err.Error(),
			LastAttempt: time.Now(),
		})
		result.DocumentsFailed = 1
		return result, err
	}

	// Store document
	sourceDoc, err := ds.storage.SaveDocument(ctx, doc, doc.RawXML, "TARGETED")
	if err != nil {
		result.FailedDocuments = append(result.FailedDocuments, FailedDocument{
			SetID:       setID,
			Error:       err.Error(),
			LastAttempt: time.Now(),
		})
		result.DocumentsFailed = 1
		return result, err
	}

	// Route sections
	if ds.config.ProcessSections {
		routedSections := ds.router.RouteDocument(doc)
		err = ds.storage.SaveRoutedSections(ctx, sourceDoc.ID, routedSections)
		if err != nil {
			return result, err
		}
		ds.storage.UpdateDocumentStatus(ctx, sourceDoc.ID, "PARSED", "")
		result.SectionsProcessed = len(routedSections)
	}

	result.DocumentsSynced = 1
	result.SyncedSetIDs = append(result.SyncedSetIDs, setID)
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// SyncByRxCUI syncs documents for a specific RxCUI
func (ds *DeltaSyncer) SyncByRxCUI(ctx context.Context, rxcui string, rxnavResolver RxNavSPLResolver) (*SyncResult, error) {
	result := &SyncResult{
		SyncID:    uuid.New(),
		Strategy:  "TARGETED_RXCUI",
		StartedAt: time.Now(),
	}

	// Get SetID from RxCUI
	setID, err := rxnavResolver.GetSPLSetID(ctx, rxcui)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resolving RxCUI: %v", err))
		return result, err
	}

	return ds.SyncBySetID(ctx, setID)
}

// SyncByNDC syncs documents for a specific NDC
func (ds *DeltaSyncer) SyncByNDC(ctx context.Context, ndc string) (*SyncResult, error) {
	result := &SyncResult{
		SyncID:    uuid.New(),
		Strategy:  "TARGETED_NDC",
		StartedAt: time.Now(),
	}

	// Search for SPL by NDC
	documents, err := ds.searchByNDC(ctx, ndc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("searching by NDC: %v", err))
		return result, err
	}

	result.TotalFound = len(documents)

	// Sync each found document
	for _, doc := range documents {
		synced, skipped, failed := ds.processDocument(ctx, doc)
		if synced {
			result.DocumentsSynced++
			result.SyncedSetIDs = append(result.SyncedSetIDs, doc.SetID)
		}
		if skipped {
			result.DocumentsSkipped++
		}
		if failed != nil {
			result.DocumentsFailed++
			result.FailedDocuments = append(result.FailedDocuments, *failed)
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// searchByNDC searches for SPL documents by NDC
func (ds *DeltaSyncer) searchByNDC(ctx context.Context, ndc string) ([]SPLListItem, error) {
	u, _ := url.Parse(ds.baseURL + "/services/v2/spls.json")
	q := u.Query()
	q.Set("ndc", ndc)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []SPLListItem `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Data, nil
}

// =============================================================================
// RETRY OPERATIONS
// =============================================================================

// RetryFailed retries previously failed documents
func (ds *DeltaSyncer) RetryFailed(ctx context.Context, failed []FailedDocument) (*SyncResult, error) {
	result := &SyncResult{
		SyncID:     uuid.New(),
		Strategy:   "RETRY",
		StartedAt:  time.Now(),
		TotalFound: len(failed),
	}

	for _, doc := range failed {
		synced, skipped, failure := ds.processDocument(ctx, SPLListItem{SetID: doc.SetID})
		if synced {
			result.DocumentsSynced++
			result.SyncedSetIDs = append(result.SyncedSetIDs, doc.SetID)
		}
		if skipped {
			result.DocumentsSkipped++
		}
		if failure != nil {
			failure.Attempts = doc.Attempts + ds.config.MaxRetries
			result.DocumentsFailed++
			result.FailedDocuments = append(result.FailedDocuments, *failure)
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// =============================================================================
// VERSION HISTORY
// =============================================================================

// VersionInfo contains version information for a SetID
type VersionInfo struct {
	SetID         string
	Version       int
	PublishedDate time.Time
	Title         string
	IsCurrent     bool
}

// GetVersionHistory retrieves the version history for a SetID
func (ds *DeltaSyncer) GetVersionHistory(ctx context.Context, setID string) ([]VersionInfo, error) {
	u, _ := url.Parse(fmt.Sprintf("%s/services/v2/spls/%s/history.json", ds.baseURL, setID))

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Version       int    `json:"spl_version"`
			PublishedDate string `json:"published_date"`
			Title         string `json:"title"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var versions []VersionInfo
	for i, v := range result.Data {
		pubDate, _ := time.Parse("2006-01-02", v.PublishedDate)
		versions = append(versions, VersionInfo{
			SetID:         setID,
			Version:       v.Version,
			PublishedDate: pubDate,
			Title:         v.Title,
			IsCurrent:     i == 0, // First is most recent
		})
	}

	return versions, nil
}

// =============================================================================
// SCHEDULED SYNC
// =============================================================================

// ScheduledSync represents a scheduled sync job
type ScheduledSync struct {
	syncer      *DeltaSyncer
	schedule    SyncSchedule
	stopChan    chan struct{}
	runningLock sync.Mutex
	isRunning   bool
}

// SyncSchedule defines when to run scheduled syncs
type SyncSchedule struct {
	DailyTime  string // "02:00" format
	WeeklyDay  time.Weekday
	WeeklyTime string
	Enabled    bool
}

// NewScheduledSync creates a scheduled sync job
func NewScheduledSync(syncer *DeltaSyncer, schedule SyncSchedule) *ScheduledSync {
	return &ScheduledSync{
		syncer:   syncer,
		schedule: schedule,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduled sync
func (ss *ScheduledSync) Start(ctx context.Context) {
	if !ss.schedule.Enabled {
		return
	}

	go ss.runLoop(ctx)
}

// Stop stops the scheduled sync
func (ss *ScheduledSync) Stop() {
	close(ss.stopChan)
}

// runLoop runs the scheduling loop
func (ss *ScheduledSync) runLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ss.stopChan:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if ss.shouldRun(now) {
				ss.runningLock.Lock()
				if !ss.isRunning {
					ss.isRunning = true
					go func() {
						defer func() {
							ss.runningLock.Lock()
							ss.isRunning = false
							ss.runningLock.Unlock()
						}()
						ss.syncer.Sync(ctx)
					}()
				}
				ss.runningLock.Unlock()
			}
		}
	}
}

// shouldRun checks if sync should run at this time
func (ss *ScheduledSync) shouldRun(now time.Time) bool {
	timeStr := now.Format("15:04")

	// Check daily schedule
	if timeStr == ss.schedule.DailyTime {
		return true
	}

	// Check weekly schedule
	if now.Weekday() == ss.schedule.WeeklyDay && timeStr == ss.schedule.WeeklyTime {
		return true
	}

	return false
}
