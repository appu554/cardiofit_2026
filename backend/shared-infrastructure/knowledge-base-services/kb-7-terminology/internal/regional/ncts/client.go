package ncts

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// NCTSClient handles interaction with Australia's National Clinical Terminology Service
type NCTSClient struct {
	BaseURL     string
	Username    string
	Password    string
	DownloadDir string
	Browser     *rod.Browser
	HTTPClient  *http.Client
	Logger      Logger
}

// Logger interface for NCTS operations
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// TerminologyAsset represents a downloadable terminology asset
type TerminologyAsset struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Type             string            `json:"type"` // "snomed-ct-au", "amt", "shrimp"
	Description      string            `json:"description"`
	ReleaseDate      time.Time         `json:"release_date"`
	Format           string            `json:"format"` // "rf2", "json", "xml"
	URL              string            `json:"url"`
	Size             int64             `json:"size"`
	Checksum         string            `json:"checksum"`
	ChecksumType     string            `json:"checksum_type"` // "sha256", "md5"
	RequiresAuth     bool              `json:"requires_auth"`
	AccessLevel      string            `json:"access_level"` // "public", "institutional", "licensed"
	Dependencies     []string          `json:"dependencies"`
	Metadata         map[string]string `json:"metadata"`
	DownloadAttempts int               `json:"download_attempts"`
	LastDownloaded   *time.Time        `json:"last_downloaded,omitempty"`
	LocalPath        string            `json:"local_path,omitempty"`
	VerificationHash string            `json:"verification_hash,omitempty"`
}

// DownloadResult represents the result of a terminology download
type DownloadResult struct {
	Asset           *TerminologyAsset `json:"asset"`
	Success         bool              `json:"success"`
	LocalPath       string            `json:"local_path"`
	ActualChecksum  string            `json:"actual_checksum"`
	VerificationOK  bool              `json:"verification_ok"`
	DownloadTime    time.Duration     `json:"download_time"`
	FileSize        int64             `json:"file_size"`
	Error           error             `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata"`
}

// NCTSConfig holds configuration for NCTS client
type NCTSConfig struct {
	BaseURL              string        `json:"base_url"`
	Username             string        `json:"username"`
	Password             string        `json:"password"`
	DownloadDirectory    string        `json:"download_directory"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	RequestTimeout       time.Duration `json:"request_timeout"`
	EnableHeadlessBrowser bool         `json:"enable_headless_browser"`
	BrowserTimeout       time.Duration `json:"browser_timeout"`
	UserAgent            string        `json:"user_agent"`
	VerifyChecksums      bool          `json:"verify_checksums"`
	KeepDownloads        bool          `json:"keep_downloads"`
	LogLevel             string        `json:"log_level"`
}

// NewNCTSClient creates a new NCTS client with the provided configuration
func NewNCTSClient(config *NCTSConfig, logger Logger) (*NCTSClient, error) {
	if config.BaseURL == "" {
		config.BaseURL = "https://www.healthterminologies.gov.au"
	}

	if config.DownloadDirectory == "" {
		config.DownloadDirectory = "./downloads/ncts"
	}

	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Minute
	}

	if config.BrowserTimeout == 0 {
		config.BrowserTimeout = 5 * time.Minute
	}

	if config.UserAgent == "" {
		config.UserAgent = "KB-7-Terminology-Service/1.0.0 (Clinical-System; Australia)"
	}

	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 10 * time.Second
	}

	// Create download directory
	if err := os.MkdirAll(config.DownloadDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Initialize HTTP client
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	client := &NCTSClient{
		BaseURL:     config.BaseURL,
		Username:    config.Username,
		Password:    config.Password,
		DownloadDir: config.DownloadDirectory,
		HTTPClient:  httpClient,
		Logger:      logger,
	}

	// Initialize browser for authenticated downloads if needed
	if config.EnableHeadlessBrowser {
		if err := client.initBrowser(config.BrowserTimeout); err != nil {
			logger.Warn("Failed to initialize browser, authenticated downloads may not work", "error", err)
		}
	}

	return client, nil
}

// initBrowser initializes the headless browser for authenticated downloads
func (c *NCTSClient) initBrowser(timeout time.Duration) error {
	launcher := launcher.New().
		Headless(true).
		NoSandbox(true).
		Devtools(false)

	url, err := launcher.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url).Timeout(timeout)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	c.Browser = browser
	return nil
}

// Close closes the NCTS client and cleans up resources
func (c *NCTSClient) Close() error {
	if c.Browser != nil {
		c.Browser.Close()
	}
	return nil
}

// ListAvailableTerminologies retrieves list of available terminology assets from NCTS
func (c *NCTSClient) ListAvailableTerminologies(ctx context.Context) ([]*TerminologyAsset, error) {
	c.Logger.Info("Retrieving available terminologies from NCTS")

	// NCTS API endpoints (these would need to be updated based on actual NCTS API)
	endpoints := []string{
		"/api/v1/terminologies/snomed-ct-au",
		"/api/v1/terminologies/amt",
		"/api/v1/terminologies/shrimp",
	}

	var allAssets []*TerminologyAsset

	for _, endpoint := range endpoints {
		assets, err := c.fetchTerminologyAssets(ctx, endpoint)
		if err != nil {
			c.Logger.Error("Failed to fetch terminologies from endpoint", "endpoint", endpoint, "error", err)
			continue
		}
		allAssets = append(allAssets, assets...)
	}

	// Add hardcoded known assets for demonstration (would be replaced with actual API calls)
	knownAssets := c.getKnownTerminologyAssets()
	allAssets = append(allAssets, knownAssets...)

	c.Logger.Info("Retrieved terminology assets", "count", len(allAssets))
	return allAssets, nil
}

// fetchTerminologyAssets fetches terminology assets from a specific endpoint
func (c *NCTSClient) fetchTerminologyAssets(ctx context.Context, endpoint string) ([]*TerminologyAsset, error) {
	fullURL := c.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "KB-7-Terminology-Service/1.0.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger.Debug("Non-200 response from NCTS", "status", resp.StatusCode, "endpoint", endpoint)
		return nil, nil // Return empty list, don't fail completely
	}

	var assets []*TerminologyAsset
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return assets, nil
}

// getKnownTerminologyAssets returns hardcoded known terminology assets
// This would be replaced with actual NCTS API integration
func (c *NCTSClient) getKnownTerminologyAssets() []*TerminologyAsset {
	now := time.Now()

	return []*TerminologyAsset{
		{
			ID:           "snomed-ct-au-20240531",
			Name:         "SNOMED CT-AU",
			Version:      "20240531",
			Type:         "snomed-ct-au",
			Description:  "SNOMED CT Australian Extension",
			ReleaseDate:  time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC),
			Format:       "rf2",
			URL:          "https://www.healthterminologies.gov.au/access/download/snomed-ct-au/20240531",
			Size:         245760000, // ~234 MB
			Checksum:     "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			ChecksumType: "sha256",
			RequiresAuth: true,
			AccessLevel:  "institutional",
			Dependencies: []string{},
			Metadata: map[string]string{
				"country":          "AU",
				"namespace":        "1000036",
				"effective_date":   "2024-05-31",
				"module_id":        "32506021000036107",
				"release_type":     "full",
				"rf2_format":       "20190731",
			},
		},
		{
			ID:           "amt-20240531",
			Name:         "Australian Medicines Terminology (AMT)",
			Version:      "20240531",
			Type:         "amt",
			Description:  "Australian Medicines Terminology - complete dataset",
			ReleaseDate:  time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC),
			Format:       "rf2",
			URL:          "https://www.healthterminologies.gov.au/access/download/amt/20240531",
			Size:         52428800, // ~50 MB
			Checksum:     "b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef1234567",
			ChecksumType: "sha256",
			RequiresAuth: true,
			AccessLevel:  "institutional",
			Dependencies: []string{"snomed-ct-au-20240531"},
			Metadata: map[string]string{
				"country":          "AU",
				"namespace":        "1000036",
				"effective_date":   "2024-05-31",
				"module_id":        "32506021000036107",
				"amt_version":      "v3.1.8",
				"includes_pbs":     "true",
				"includes_artg":    "true",
			},
		},
		{
			ID:           "shrimp-20240531",
			Name:         "SHRIMP (Secure Hash for Rapid Identification of Medication Products)",
			Version:      "20240531",
			Type:         "shrimp",
			Description:  "Secure Hash for Rapid Identification of Medication Products",
			ReleaseDate:  time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC),
			Format:       "json",
			URL:          "https://www.healthterminologies.gov.au/access/download/shrimp/20240531",
			Size:         10485760, // ~10 MB
			Checksum:     "c3d4e5f6789012345678901234567890abcdef1234567890abcdef12345678",
			ChecksumType: "sha256",
			RequiresAuth: true,
			AccessLevel:  "institutional",
			Dependencies: []string{"amt-20240531"},
			Metadata: map[string]string{
				"country":        "AU",
				"effective_date": "2024-05-31",
				"hash_algorithm": "SHA-256",
				"encoding":       "base64",
			},
		},
	}
}

// DownloadTerminology downloads a specific terminology asset
func (c *NCTSClient) DownloadTerminology(ctx context.Context, asset *TerminologyAsset) (*DownloadResult, error) {
	c.Logger.Info("Starting terminology download", "asset", asset.ID, "type", asset.Type)

	result := &DownloadResult{
		Asset:    asset,
		Success:  false,
		Metadata: make(map[string]string),
	}

	startTime := time.Now()

	// Create asset-specific download directory
	assetDir := filepath.Join(c.DownloadDir, asset.Type, asset.Version)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create asset directory: %w", err)
		return result, result.Error
	}

	// Determine filename
	filename := fmt.Sprintf("%s-%s.%s", asset.Type, asset.Version, c.getFileExtension(asset.Format))
	localPath := filepath.Join(assetDir, filename)

	// Check if file already exists and is valid
	if c.isFileValid(localPath, asset) {
		c.Logger.Info("File already exists and is valid", "path", localPath)
		result.Success = true
		result.LocalPath = localPath
		result.VerificationOK = true
		result.DownloadTime = 0

		if stat, err := os.Stat(localPath); err == nil {
			result.FileSize = stat.Size()
		}

		return result, nil
	}

	// Download the file
	var err error
	if asset.RequiresAuth {
		err = c.downloadWithAuthentication(ctx, asset, localPath)
	} else {
		err = c.downloadDirect(ctx, asset, localPath)
	}

	if err != nil {
		result.Error = fmt.Errorf("download failed: %w", err)
		return result, result.Error
	}

	result.DownloadTime = time.Since(startTime)
	result.LocalPath = localPath

	// Get file size
	if stat, err := os.Stat(localPath); err == nil {
		result.FileSize = stat.Size()
	}

	// Verify checksum if provided
	if asset.Checksum != "" {
		actualChecksum, err := c.calculateChecksum(localPath, asset.ChecksumType)
		if err != nil {
			c.Logger.Warn("Failed to calculate checksum", "path", localPath, "error", err)
		} else {
			result.ActualChecksum = actualChecksum
			result.VerificationOK = strings.EqualFold(actualChecksum, asset.Checksum)

			if !result.VerificationOK {
				c.Logger.Error("Checksum verification failed",
					"expected", asset.Checksum,
					"actual", actualChecksum,
					"file", localPath)
				result.Error = fmt.Errorf("checksum verification failed")
				return result, result.Error
			}
		}
	}

	result.Success = true
	asset.LocalPath = localPath
	asset.LastDownloaded = &startTime
	asset.VerificationHash = result.ActualChecksum

	c.Logger.Info("Successfully downloaded terminology",
		"asset", asset.ID,
		"size", result.FileSize,
		"duration", result.DownloadTime)

	return result, nil
}

// downloadDirect downloads a file directly without authentication
func (c *NCTSClient) downloadDirect(ctx context.Context, asset *TerminologyAsset, localPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", asset.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "KB-7-Terminology-Service/1.0.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// downloadWithAuthentication downloads a file using browser automation for authentication
func (c *NCTSClient) downloadWithAuthentication(ctx context.Context, asset *TerminologyAsset, localPath string) error {
	if c.Browser == nil {
		return fmt.Errorf("browser not available for authenticated download")
	}

	// Create a page
	page := c.Browser.MustPage()
	defer page.Close()

	// Navigate to login page
	loginURL := c.BaseURL + "/access/login"
	c.Logger.Debug("Navigating to login page", "url", loginURL)

	err := page.Navigate(loginURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// Wait for page to load
	page.MustWaitLoad()

	// Fill in login form (adjust selectors based on actual NCTS login page)
	usernameSelector := "input[name='username'], input[type='email'], #username, #email"
	passwordSelector := "input[name='password'], input[type='password'], #password"
	submitSelector := "button[type='submit'], input[type='submit'], .login-button"

	// Enter credentials
	if username := page.MustElement(usernameSelector); username != nil {
		username.MustInput(c.Username)
	} else {
		return fmt.Errorf("username field not found")
	}

	if password := page.MustElement(passwordSelector); password != nil {
		password.MustInput(c.Password)
	} else {
		return fmt.Errorf("password field not found")
	}

	// Submit login form
	if submit := page.MustElement(submitSelector); submit != nil {
		submit.MustClick()
	} else {
		return fmt.Errorf("submit button not found")
	}

	// Wait for login to complete
	page.MustWaitLoad()

	// Navigate to download URL
	c.Logger.Debug("Navigating to download URL", "url", asset.URL)

	err = page.Navigate(asset.URL)
	if err != nil {
		return fmt.Errorf("failed to navigate to download URL: %w", err)
	}

	// Set download behavior
	page.MustEvaluate(&rod.EvalOptions{
		JS: `() => {
			// Override download behavior to capture the download
			const originalDownload = HTMLAnchorElement.prototype.click;
			HTMLAnchorElement.prototype.click = function() {
				window.downloadTriggered = this.href;
				return originalDownload.call(this);
			};
		}`,
	})

	// Trigger download (this would need to be customized based on NCTS interface)
	downloadButton := page.MustElement("a[href*='download'], button[data-download], .download-button")
	downloadButton.MustClick()

	// Wait for download to be triggered
	page.MustWait(`() => window.downloadTriggered`)

	// Get the actual download URL
	downloadURL := page.MustEval("() => window.downloadTriggered").String()

	// Use the authenticated session to download the file
	return c.downloadFileWithSession(ctx, page, downloadURL, localPath)
}

// downloadFileWithSession downloads a file using an authenticated browser session
func (c *NCTSClient) downloadFileWithSession(ctx context.Context, page *rod.Page, downloadURL, localPath string) error {
	// Get cookies from the browser session
	cookies := page.MustCookies()

	// Create HTTP request with cookies
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	// Add cookies to request
	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	req.Header.Set("User-Agent", "KB-7-Terminology-Service/1.0.0")

	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// calculateChecksum calculates the checksum of a file
func (c *NCTSClient) calculateChecksum(filePath, checksumType string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	switch strings.ToLower(checksumType) {
	case "sha256":
		hash := sha256.New()
		_, err := io.Copy(hash, file)
		if err != nil {
			return "", fmt.Errorf("failed to calculate SHA256: %w", err)
		}
		return fmt.Sprintf("%x", hash.Sum(nil)), nil
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checksumType)
	}
}

// isFileValid checks if a local file exists and matches the expected characteristics
func (c *NCTSClient) isFileValid(localPath string, asset *TerminologyAsset) bool {
	stat, err := os.Stat(localPath)
	if err != nil {
		return false
	}

	// Check file size if available
	if asset.Size > 0 && stat.Size() != asset.Size {
		c.Logger.Debug("File size mismatch", "expected", asset.Size, "actual", stat.Size())
		return false
	}

	// Check checksum if available
	if asset.Checksum != "" {
		actualChecksum, err := c.calculateChecksum(localPath, asset.ChecksumType)
		if err != nil {
			c.Logger.Debug("Failed to calculate checksum for validation", "error", err)
			return false
		}

		if !strings.EqualFold(actualChecksum, asset.Checksum) {
			c.Logger.Debug("Checksum mismatch", "expected", asset.Checksum, "actual", actualChecksum)
			return false
		}
	}

	return true
}

// getFileExtension returns the appropriate file extension for a format
func (c *NCTSClient) getFileExtension(format string) string {
	switch strings.ToLower(format) {
	case "rf2":
		return "zip"
	case "json":
		return "json"
	case "xml":
		return "xml"
	default:
		return "dat"
	}
}

// GetDownloadManifest creates a manifest of all downloaded terminology assets
func (c *NCTSClient) GetDownloadManifest() (*DownloadManifest, error) {
	manifest := &DownloadManifest{
		GeneratedAt: time.Now(),
		DownloadDir: c.DownloadDir,
		Assets:      make([]*TerminologyAsset, 0),
	}

	// Walk through download directory to find all assets
	err := filepath.Walk(c.DownloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Try to identify the asset type from path structure
		relPath, _ := filepath.Rel(c.DownloadDir, path)
		pathParts := strings.Split(relPath, string(filepath.Separator))

		if len(pathParts) >= 2 {
			assetType := pathParts[0]
			version := pathParts[1]

			asset := &TerminologyAsset{
				ID:      fmt.Sprintf("%s-%s", assetType, version),
				Name:    assetType,
				Version: version,
				Type:    assetType,
				Format:  c.inferFormatFromExtension(filepath.Ext(path)),
				Size:    info.Size(),
				LocalPath: path,
				LastDownloaded: &info.ModTime(),
			}

			// Calculate current checksum
			if checksum, err := c.calculateChecksum(path, "sha256"); err == nil {
				asset.VerificationHash = checksum
			}

			manifest.Assets = append(manifest.Assets, asset)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk download directory: %w", err)
	}

	manifest.TotalAssets = len(manifest.Assets)
	manifest.TotalSize = c.calculateTotalSize(manifest.Assets)

	return manifest, nil
}

// DownloadManifest represents a manifest of downloaded terminology assets
type DownloadManifest struct {
	GeneratedAt   time.Time            `json:"generated_at"`
	DownloadDir   string               `json:"download_dir"`
	TotalAssets   int                  `json:"total_assets"`
	TotalSize     int64                `json:"total_size"`
	Assets        []*TerminologyAsset  `json:"assets"`
}

func (c *NCTSClient) inferFormatFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".zip":
		return "rf2"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	default:
		return "unknown"
	}
}

func (c *NCTSClient) calculateTotalSize(assets []*TerminologyAsset) int64 {
	var total int64
	for _, asset := range assets {
		total += asset.Size
	}
	return total
}