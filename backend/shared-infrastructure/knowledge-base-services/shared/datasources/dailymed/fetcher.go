// Package dailymed provides a client for FDA DailyMed SPL (Structured Product Label) documents.
// DailyMed is the official provider of FDA-approved drug labeling.
//
// Phase 3a.3: DailyMed SPL Fetcher
// Key Feature: Fetch full SPL XML documents for clinical fact extraction
//
// API Documentation: https://dailymed.nlm.nih.gov/dailymed/webservices-help/v2/index-api.cfm
package dailymed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// SPL FETCHER
// =============================================================================

// SPLFetcher downloads and caches FDA Structured Product Labels
type SPLFetcher struct {
	baseURL     string
	splBaseURL  string // Direct SPL XML download URL
	httpClient  *http.Client
	cache       SPLCache
	rateLimiter chan struct{}
	mu          sync.Mutex
}

// Config holds configuration for the SPL fetcher
type Config struct {
	BaseURL            string        // DailyMed API URL (default: https://dailymed.nlm.nih.gov/dailymed)
	Timeout            time.Duration // HTTP timeout
	RateLimitPerSecond int           // API rate limit
	CacheTTL           time.Duration // Cache duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		BaseURL:            "https://dailymed.nlm.nih.gov/dailymed",
		Timeout:            60 * time.Second,
		RateLimitPerSecond: 5, // Be conservative with federal APIs
		CacheTTL:           24 * time.Hour,
	}
}

// NewSPLFetcher creates a new SPL fetcher
func NewSPLFetcher(config Config, cache SPLCache) *SPLFetcher {
	if config.BaseURL == "" {
		config.BaseURL = "https://dailymed.nlm.nih.gov/dailymed"
	}

	f := &SPLFetcher{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		splBaseURL: "https://dailymed.nlm.nih.gov/dailymed/services/v2/spls",
		httpClient: &http.Client{Timeout: config.Timeout},
		cache:      cache,
	}

	// Initialize rate limiter
	if config.RateLimitPerSecond > 0 {
		f.rateLimiter = make(chan struct{}, config.RateLimitPerSecond)
		for i := 0; i < config.RateLimitPerSecond; i++ {
			f.rateLimiter <- struct{}{}
		}
		go f.refillRateLimiter()
	}

	return f
}

// =============================================================================
// SPL DOCUMENT STRUCTURES
// =============================================================================

// SPLDocument represents a full SPL XML document
type SPLDocument struct {
	XMLName       xml.Name        `xml:"document"`
	ID            SPLID           `xml:"id"`
	SetID         SPLID           `xml:"setId"`
	VersionNumber SPLVersionNumber `xml:"versionNumber"`
	EffectiveTime SPLTime         `xml:"effectiveTime"`
	Title         string          `xml:"title"`
	Sections      []SPLSection    `xml:"component>structuredBody>component>section"`

	// Metadata (populated after parsing)
	RawXML      []byte    `xml:"-"`
	ContentHash string    `xml:"-"`
	FetchedAt   time.Time `xml:"-"`
}

// SPLVersionNumber represents a version number in SPL format
type SPLVersionNumber struct {
	Value int `xml:"value,attr"`
}

// SPLID represents an identifier in SPL format
type SPLID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

// SPLTime represents a time value in SPL format
type SPLTime struct {
	Value string `xml:"value,attr"`
}

// SPLSection represents a labeled section with LOINC code
type SPLSection struct {
	ID          SPLID        `xml:"id"`
	Code        SPLCode      `xml:"code"`
	Title       string       `xml:"title"`
	Text        SPLText      `xml:"text"`
	Subsections []SPLSection `xml:"component>section"`
}

// SPLCode represents a coded value (LOINC code for sections)
type SPLCode struct {
	Code           string `xml:"code,attr"`
	CodeSystem     string `xml:"codeSystem,attr"`     // Should be 2.16.840.1.113883.6.1 for LOINC
	CodeSystemName string `xml:"codeSystemName,attr"` // "LOINC"
	DisplayName    string `xml:"displayName,attr"`
}

// SPLText contains the narrative content (may include tables)
type SPLText struct {
	Content    string     `xml:",innerxml"`
	Tables     []SPLTable `xml:"table"`
	Paragraphs []string   `xml:"paragraph"`
	Lists      []SPLList  `xml:"list"`
}

// SPLTable represents a structured table in SPL
type SPLTable struct {
	ID       string        `xml:"ID,attr"`
	Width    string        `xml:"width,attr"`
	THead    SPLTableHead  `xml:"thead"`
	Rows     []SPLTableRow `xml:"tbody>tr"`
	Caption  string        `xml:"caption"`
	// Headers is populated from THead.Row.Cells after parsing
	Headers []string `xml:"-"`
}

// SPLTableHead represents the table head section
type SPLTableHead struct {
	Row SPLTableRow `xml:"tr"`
}

// SPLTableRow represents a row in an SPL table
type SPLTableRow struct {
	Cells       []SPLTableCell `xml:"td"`
	HeaderCells []SPLTableCell `xml:"th"`
}

// SPLTableCell represents a cell in an SPL table
type SPLTableCell struct {
	Content string `xml:",innerxml"`
	Colspan string `xml:"colspan,attr"`
	Rowspan string `xml:"rowspan,attr"`
}

// GetHeaders returns the table headers as strings
func (t *SPLTable) GetHeaders() []string {
	// If already populated, return cached
	if len(t.Headers) > 0 {
		return t.Headers
	}
	// Extract from THead header cells
	for _, cell := range t.THead.Row.HeaderCells {
		t.Headers = append(t.Headers, stripXMLTags(cell.Content))
	}
	return t.Headers
}

// stripXMLTags removes XML tags from content, keeping only text
func stripXMLTags(s string) string {
	// Simple regex-free approach to strip XML tags
	result := ""
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result += string(r)
		}
	}
	return strings.TrimSpace(result)
}

// SPLList represents a list element
type SPLList struct {
	ListType string   `xml:"listType,attr"`
	Items    []string `xml:"item"`
}

// =============================================================================
// LOINC SECTION CODES (Key for Phase 3)
// =============================================================================

// LOINC section codes for clinical decision support
const (
	LOINCDosageAdministration = "34068-7" // Dosage and Administration
	LOINCBoxedWarning         = "34066-1" // Boxed Warning
	LOINCContraindications    = "34070-3" // Contraindications
	LOINCWarningsPrecautions  = "43685-7" // Warnings and Precautions
	LOINCPregnancy            = "34077-8" // Pregnancy
	LOINCNursing              = "34080-2" // Nursing Mothers
	LOINCPediatricUse         = "34081-0" // Pediatric Use
	LOINCGeriatricUse         = "34082-8" // Geriatric Use
	LOINCClinicalPharm        = "34090-1" // Clinical Pharmacology
	LOINCDrugInteractions     = "34073-7" // Drug Interactions
	LOINCAdverseReactions     = "34084-4" // Adverse Reactions
	LOINCOverdosage           = "34088-5" // Overdosage
	LOINCHowSupplied          = "34069-5" // How Supplied
	LOINCRenalImpairment      = "42232-0" // Renal Impairment (subsection)
	LOINCHepaticImpairment    = "42229-5" // Hepatic Impairment (subsection)
)

// =============================================================================
// FETCH OPERATIONS
// =============================================================================

// FetchBySetID retrieves SPL by its unique set identifier
func (f *SPLFetcher) FetchBySetID(ctx context.Context, setID string) (*SPLDocument, error) {
	// Check cache first
	if f.cache != nil {
		if cached, found := f.cache.Get(setID); found {
			return cached, nil
		}
	}

	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Fetch the SPL XML using REST API v2
	splURL := fmt.Sprintf("%s/%s.xml", f.splBaseURL, url.PathEscape(setID))

	req, err := http.NewRequestWithContext(ctx, "GET", splURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	// Note: DailyMed REST API v2 doesn't accept Accept headers - use URL extension

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching SPL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SPL fetch failed for SetID %s: HTTP %d", setID, resp.StatusCode)
	}

	// Read the raw XML
	rawXML, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading SPL response: %w", err)
	}

	// Parse the XML
	var doc SPLDocument
	if err := xml.Unmarshal(rawXML, &doc); err != nil {
		return nil, fmt.Errorf("parsing SPL XML: %w", err)
	}

	// Add metadata
	doc.RawXML = rawXML
	doc.ContentHash = hashContent(rawXML)
	doc.FetchedAt = time.Now()

	// Cache for future use
	if f.cache != nil {
		f.cache.Set(setID, &doc)
	}

	return &doc, nil
}

// FetchByRxCUI retrieves SPL for a drug by its RxCUI (requires RxNav lookup first)
// Note: Use RxNavClient.GetSPLSetIDFromRxCUI() first, then call FetchBySetID()
func (f *SPLFetcher) FetchByRxCUI(ctx context.Context, rxcui string, rxnavClient RxNavSPLResolver) (*SPLDocument, error) {
	// Step 1: Get SetID from RxNav
	setID, err := rxnavClient.GetSPLSetID(ctx, rxcui)
	if err != nil {
		return nil, fmt.Errorf("resolving SetID for RxCUI %s: %w", rxcui, err)
	}

	// Step 2: Fetch by SetID
	return f.FetchBySetID(ctx, setID)
}

// SearchByDrugName searches DailyMed for SPL documents by drug name
func (f *SPLFetcher) SearchByDrugName(ctx context.Context, drugName string) ([]SPLSearchResult, error) {
	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	searchURL := fmt.Sprintf("%s/services/v2/spls.json?drug_name=%s", f.baseURL, url.QueryEscape(drugName))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searching DailyMed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}

	var searchResp struct {
		Data []SPLSearchResult `json:"data"`
	}

	if err := parseJSONResponse(resp.Body, &searchResp); err != nil {
		return nil, err
	}

	return searchResp.Data, nil
}

// SPLSearchResult represents a search result from DailyMed
type SPLSearchResult struct {
	SetID       string `json:"setid"`
	Title       string `json:"title"`
	ProductNDC  string `json:"product_ndc"`
	PublishedAt string `json:"published_date"`
	ProductType string `json:"product_type"`
	Labeler     string `json:"labeler"`
}

// =============================================================================
// SECTION EXTRACTION
// =============================================================================

// GetSection extracts a specific section by LOINC code
func (doc *SPLDocument) GetSection(loincCode string) *SPLSection {
	for i := range doc.Sections {
		if doc.Sections[i].Code.Code == loincCode {
			return &doc.Sections[i]
		}
		// Check subsections
		for j := range doc.Sections[i].Subsections {
			if doc.Sections[i].Subsections[j].Code.Code == loincCode {
				return &doc.Sections[i].Subsections[j]
			}
		}
	}
	return nil
}

// GetAllSections returns all sections with LOINC codes
func (doc *SPLDocument) GetAllSections() map[string]*SPLSection {
	sections := make(map[string]*SPLSection)

	var collectSections func(s []SPLSection)
	collectSections = func(s []SPLSection) {
		for i := range s {
			if s[i].Code.Code != "" {
				sections[s[i].Code.Code] = &s[i]
			}
			collectSections(s[i].Subsections)
		}
	}

	collectSections(doc.Sections)
	return sections
}

// GetDosageSection returns the Dosage and Administration section
func (doc *SPLDocument) GetDosageSection() *SPLSection {
	return doc.GetSection(LOINCDosageAdministration)
}

// GetBoxedWarning returns the Boxed Warning section
func (doc *SPLDocument) GetBoxedWarning() *SPLSection {
	return doc.GetSection(LOINCBoxedWarning)
}

// GetContraindications returns the Contraindications section
func (doc *SPLDocument) GetContraindications() *SPLSection {
	return doc.GetSection(LOINCContraindications)
}

// GetDrugInteractions returns the Drug Interactions section
func (doc *SPLDocument) GetDrugInteractions() *SPLSection {
	return doc.GetSection(LOINCDrugInteractions)
}

// HasTables returns true if the section or any subsection contains structured tables
func (s *SPLSection) HasTables() bool {
	if len(s.Text.Tables) > 0 {
		return true
	}
	for i := range s.Subsections {
		if len(s.Subsections[i].Text.Tables) > 0 {
			return true
		}
	}
	return false
}

// GetTables returns all tables in the section
func (s *SPLSection) GetTables() []SPLTable {
	return s.Text.Tables
}

// GetRawText returns the text content without HTML markup.
// Always aggregates parent text + subsection text so that nested FDA label
// structures (e.g. Adverse Reactions 34084-4 with child 6.1, 6.2) are captured.
func (s *SPLSection) GetRawText() string {
	var parts []string

	parentText := stripHTMLTags(s.Text.Content)
	if parentText != "" {
		parts = append(parts, parentText)
	}

	for i := range s.Subsections {
		sub := stripHTMLTags(s.Subsections[i].Text.Content)
		if sub != "" {
			if s.Subsections[i].Title != "" {
				parts = append(parts, s.Subsections[i].Title+": "+sub)
			} else {
				parts = append(parts, sub)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

// =============================================================================
// CACHE INTERFACE
// =============================================================================

// SPLCache defines the caching interface for SPL documents
type SPLCache interface {
	Get(setID string) (*SPLDocument, bool)
	Set(setID string, doc *SPLDocument)
	Delete(setID string)
	Clear()
}

// RxNavSPLResolver defines the interface for RxNav SPL lookups
type RxNavSPLResolver interface {
	GetSPLSetID(ctx context.Context, rxcui string) (string, error)
}

// =============================================================================
// IN-MEMORY CACHE IMPLEMENTATION
// =============================================================================

// MemorySPLCache provides a simple in-memory cache for SPL documents
type MemorySPLCache struct {
	cache map[string]*cachedSPL
	ttl   time.Duration
	mu    sync.RWMutex
}

type cachedSPL struct {
	doc       *SPLDocument
	expiresAt time.Time
}

// NewMemorySPLCache creates a new in-memory cache
func NewMemorySPLCache(ttl time.Duration) *MemorySPLCache {
	c := &MemorySPLCache{
		cache: make(map[string]*cachedSPL),
		ttl:   ttl,
	}
	go c.cleanupLoop()
	return c
}

func (c *MemorySPLCache) Get(setID string) (*SPLDocument, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache[setID]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.doc, true
		}
	}
	return nil, false
}

func (c *MemorySPLCache) Set(setID string, doc *SPLDocument) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[setID] = &cachedSPL{
		doc:       doc,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *MemorySPLCache) Delete(setID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, setID)
}

func (c *MemorySPLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cachedSPL)
}

func (c *MemorySPLCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, cached := range c.cache {
			if now.After(cached.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func (f *SPLFetcher) refillRateLimiter() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		select {
		case f.rateLimiter <- struct{}{}:
		default:
		}
	}
}

func hashContent(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func stripHTMLTags(s string) string {
	// Simple HTML tag stripping
	// In production, use a proper HTML parser
	result := s
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}

func parseJSONResponse(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}

// =============================================================================
// ADDITIONAL FETCH OPERATIONS
// =============================================================================

// FetchByNDC retrieves SPL by National Drug Code
func (f *SPLFetcher) FetchByNDC(ctx context.Context, ndc string) (*SPLDocument, error) {
	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Search for SPL by NDC
	searchURL := fmt.Sprintf("%s/services/v2/spls.json?ndc=%s", f.baseURL, url.QueryEscape(ndc))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searching by NDC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NDC search failed: HTTP %d", resp.StatusCode)
	}

	var searchResp struct {
		Data []SPLSearchResult `json:"data"`
	}

	if err := parseJSONResponse(resp.Body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	if len(searchResp.Data) == 0 {
		return nil, fmt.Errorf("no SPL found for NDC: %s", ndc)
	}

	// Fetch the first (most relevant) result by SetID
	return f.FetchBySetID(ctx, searchResp.Data[0].SetID)
}

// SPLVersionInfo contains information about a specific SPL version
type SPLVersionInfo struct {
	SetID         string
	Version       int
	PublishedDate string
	Title         string
	IsCurrent     bool
}

// GetVersionHistory retrieves the version history for a SetID
func (f *SPLFetcher) GetVersionHistory(ctx context.Context, setID string) ([]SPLVersionInfo, error) {
	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	historyURL := fmt.Sprintf("%s/services/v2/spls/%s/history.json", f.baseURL, url.QueryEscape(setID))

	req, err := http.NewRequestWithContext(ctx, "GET", historyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating history request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching version history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version history failed: HTTP %d", resp.StatusCode)
	}

	var historyResp struct {
		Data []struct {
			Version       int    `json:"spl_version"`
			PublishedDate string `json:"published_date"`
			Title         string `json:"title"`
		} `json:"data"`
	}

	if err := parseJSONResponse(resp.Body, &historyResp); err != nil {
		return nil, fmt.Errorf("parsing history response: %w", err)
	}

	var versions []SPLVersionInfo
	for i, v := range historyResp.Data {
		versions = append(versions, SPLVersionInfo{
			SetID:         setID,
			Version:       v.Version,
			PublishedDate: v.PublishedDate,
			Title:         v.Title,
			IsCurrent:     i == 0, // First is most recent
		})
	}

	return versions, nil
}

// FetchUpdates retrieves SPL documents updated since a given time
func (f *SPLFetcher) FetchUpdates(ctx context.Context, since time.Time) ([]*SPLDocument, error) {
	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Format date for API
	dateStr := since.Format("2006-01-02")

	// Search for updated SPLs
	searchURL := fmt.Sprintf("%s/services/v2/spls.json?published_date=%s&published_date_comparison=gte&pagesize=100",
		f.baseURL, url.QueryEscape(dateStr))

	var allDocs []*SPLDocument
	page := 1

	for {
		pageURL := searchURL + fmt.Sprintf("&page=%d", page)

		req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := f.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching updates: %w", err)
		}

		var searchResp struct {
			Data     []SPLSearchResult `json:"data"`
			Metadata struct {
				NextPageURL string `json:"next_page_url"`
			} `json:"metadata"`
		}

		if err := parseJSONResponse(resp.Body, &searchResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		resp.Body.Close()

		// Fetch each document
		for _, result := range searchResp.Data {
			doc, err := f.FetchBySetID(ctx, result.SetID)
			if err != nil {
				// Log error but continue
				continue
			}
			allDocs = append(allDocs, doc)
		}

		// Check if there are more pages
		if searchResp.Metadata.NextPageURL == "" {
			break
		}
		page++

		// Safety limit
		if page > 100 {
			break
		}
	}

	return allDocs, nil
}

// FetchSpecificVersion retrieves a specific version of an SPL
func (f *SPLFetcher) FetchSpecificVersion(ctx context.Context, setID string, version int) (*SPLDocument, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s_v%d", setID, version)
	if f.cache != nil {
		if cached, found := f.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Rate limiting
	if f.rateLimiter != nil {
		select {
		case <-f.rateLimiter:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Fetch the specific version using REST API v2
	splURL := fmt.Sprintf("%s/%s/%d.xml", f.splBaseURL, url.PathEscape(setID), version)

	req, err := http.NewRequestWithContext(ctx, "GET", splURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	// Note: DailyMed REST API v2 doesn't accept Accept headers - use URL extension

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching SPL version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SPL version fetch failed: HTTP %d", resp.StatusCode)
	}

	// Read the raw XML
	rawXML, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading SPL response: %w", err)
	}

	// Parse the XML
	var doc SPLDocument
	if err := xml.Unmarshal(rawXML, &doc); err != nil {
		return nil, fmt.Errorf("parsing SPL XML: %w", err)
	}

	// Add metadata
	doc.RawXML = rawXML
	doc.ContentHash = hashContent(rawXML)
	doc.FetchedAt = time.Now()

	// Cache for future use
	if f.cache != nil {
		f.cache.Set(cacheKey, &doc)
	}

	return &doc, nil
}

// =============================================================================
// BULK DOWNLOAD SUPPORT
// =============================================================================

// BulkDownloadConfig contains configuration for bulk downloads
type BulkDownloadConfig struct {
	DownloadDir   string
	ProductType   string // "human_rx", "human_otc", "animal"
	UpdateType    string // "full", "daily", "weekly", "monthly"
	Concurrency   int
}

// BulkDownloadURL returns the FTP URL for bulk downloads
func BulkDownloadURL(config BulkDownloadConfig) string {
	base := "ftp://public.nlm.nih.gov/nlmdata/.dailymed/"

	switch config.UpdateType {
	case "full":
		return base + fmt.Sprintf("dm_spl_release_%s.zip", config.ProductType)
	case "daily":
		return base + fmt.Sprintf("dm_spl_daily_update_%s.zip", time.Now().Format("20060102"))
	case "weekly":
		return base + fmt.Sprintf("dm_spl_weekly_%s.zip", time.Now().Format("20060102"))
	case "monthly":
		return base + fmt.Sprintf("dm_spl_monthly_%s.zip", time.Now().Format("200601"))
	default:
		return base + fmt.Sprintf("dm_spl_release_%s.zip", config.ProductType)
	}
}
