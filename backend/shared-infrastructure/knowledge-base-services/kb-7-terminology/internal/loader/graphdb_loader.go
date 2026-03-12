// Package loader provides services for loading terminology data into GraphDB.
// This is the Phase 5 implementation that supports loading from GCS or local files.
package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// Config holds the configuration for the GraphDB loader
type Config struct {
	// GCS Configuration
	GCSBucket      string `json:"gcs_bucket"`
	GCSCredentials string `json:"gcs_credentials"`

	// GraphDB Configuration
	GraphDBURL      string `json:"graphdb_url"`
	GraphDBRepo     string `json:"graphdb_repo"`
	GraphDBUsername string `json:"graphdb_username"`
	GraphDBPassword string `json:"graphdb_password"`

	// Load Options
	Timeout    time.Duration `json:"timeout"`
	NamedGraph string        `json:"named_graph"`
}

// LoadResult represents the result of a load operation
type LoadResult struct {
	Success       bool          `json:"success"`
	TripleCount   int64         `json:"triple_count"`
	Duration      time.Duration `json:"duration"`
	Source        string        `json:"source"`
	Version       string        `json:"version"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	LoadTimestamp time.Time     `json:"load_timestamp"`
}

// RepositoryStatus represents the status of a GraphDB repository
type RepositoryStatus struct {
	Available    bool   `json:"available"`
	Repository   string `json:"repository"`
	TripleCount  int64  `json:"triple_count"`
	Writable     bool   `json:"writable"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// VerificationResult represents the result of data verification
type VerificationResult struct {
	TotalTriples   int64 `json:"total_triples"`
	SNOMEDConcepts int64 `json:"snomed_concepts"`
	RxNormConcepts int64 `json:"rxnorm_concepts"`
	LOINCConcepts  int64 `json:"loinc_concepts"`
	AllPassed      bool  `json:"all_passed"`
}

// GraphDBLoader handles loading terminology data into GraphDB
type GraphDBLoader struct {
	config *Config
	logger *logrus.Logger
	client *http.Client
}

// NewGraphDBLoader creates a new GraphDB loader instance
func NewGraphDBLoader(config *Config, logger *logrus.Logger) *GraphDBLoader {
	if logger == nil {
		logger = logrus.New()
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	return &GraphDBLoader{
		config: config,
		logger: logger,
		client: &http.Client{Timeout: timeout},
	}
}

// LoadFromGCS loads TTL data from GCS into GraphDB
func (l *GraphDBLoader) LoadFromGCS(ctx context.Context, version string) (*LoadResult, error) {
	result := &LoadResult{
		Source:        "gcs",
		Version:       version,
		LoadTimestamp: time.Now(),
	}
	startTime := time.Now()

	gcsPath := fmt.Sprintf("%s/kb7-kernel.ttl", version)
	l.logger.Infof("Loading from GCS: gs://%s/%s", l.config.GCSBucket, gcsPath)

	// Create GCS client
	var client *storage.Client
	var err error

	if l.config.GCSCredentials != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(l.config.GCSCredentials))
	} else {
		client, err = storage.NewClient(ctx)
	}
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create GCS client: %v", err)
		return result, err
	}
	defer client.Close()

	// Generate signed URL
	bucket := client.Bucket(l.config.GCSBucket)
	obj := bucket.Object(gcsPath)

	// Check if object exists
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("GCS object not found: %v", err)
		return result, err
	}
	l.logger.Infof("File size: %.2f GB", float64(attrs.Size)/(1024*1024*1024))

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(60 * time.Minute),
	}

	signedURL, err := bucket.SignedURL(gcsPath, opts)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to generate signed URL: %v", err)
		return result, err
	}

	// Execute SPARQL LOAD
	if err := l.executeSPARQLLoad(ctx, signedURL); err != nil {
		result.ErrorMessage = fmt.Sprintf("SPARQL LOAD failed: %v", err)
		return result, err
	}

	// Get triple count
	count, _ := l.GetTripleCount(ctx)
	result.TripleCount = count
	result.Success = true
	result.Duration = time.Since(startTime)

	return result, nil
}

// LoadFromFile loads TTL data from a local file into GraphDB
func (l *GraphDBLoader) LoadFromFile(ctx context.Context, filePath string) (*LoadResult, error) {
	result := &LoadResult{
		Source:        "file",
		Version:       filePath,
		LoadTimestamp: time.Now(),
	}
	startTime := time.Now()

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("file not found: %v", err)
		return result, err
	}
	l.logger.Infof("Loading file: %s (%.2f GB)", filePath, float64(info.Size())/(1024*1024*1024))

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to open file: %v", err)
		return result, err
	}
	defer file.Close()

	// Upload to GraphDB
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", l.config.GraphDBURL, l.config.GraphDBRepo)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, file)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return result, err
	}

	req.Header.Set("Content-Type", "text/turtle")
	if l.config.GraphDBUsername != "" {
		req.SetBasicAuth(l.config.GraphDBUsername, l.config.GraphDBPassword)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("upload failed: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		result.ErrorMessage = fmt.Sprintf("upload failed with status %d: %s", resp.StatusCode, string(body))
		return result, fmt.Errorf(result.ErrorMessage)
	}

	// Get triple count
	count, _ := l.GetTripleCount(ctx)
	result.TripleCount = count
	result.Success = true
	result.Duration = time.Since(startTime)

	return result, nil
}

// executeSPARQLLoad executes a SPARQL LOAD command
func (l *GraphDBLoader) executeSPARQLLoad(ctx context.Context, sourceURL string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", l.config.GraphDBURL, l.config.GraphDBRepo)

	var sparqlUpdate string
	if l.config.NamedGraph != "" {
		sparqlUpdate = fmt.Sprintf("LOAD <%s> INTO GRAPH <%s>", sourceURL, l.config.NamedGraph)
	} else {
		sparqlUpdate = fmt.Sprintf("LOAD <%s>", sourceURL)
	}

	l.logger.Info("Executing SPARQL LOAD (this may take 5-15 minutes)...")

	data := url.Values{}
	data.Set("update", sparqlUpdate)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if l.config.GraphDBUsername != "" {
		req.SetBasicAuth(l.config.GraphDBUsername, l.config.GraphDBPassword)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("SPARQL LOAD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SPARQL LOAD failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CheckConnection verifies connectivity to GraphDB
func (l *GraphDBLoader) CheckConnection(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/rest/repositories/%s", l.config.GraphDBURL, l.config.GraphDBRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}

	if l.config.GraphDBUsername != "" {
		req.SetBasicAuth(l.config.GraphDBUsername, l.config.GraphDBPassword)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("repository not accessible: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ClearRepository removes all triples from the repository
func (l *GraphDBLoader) ClearRepository(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", l.config.GraphDBURL, l.config.GraphDBRepo)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	if l.config.GraphDBUsername != "" {
		req.SetBasicAuth(l.config.GraphDBUsername, l.config.GraphDBPassword)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("clear request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("clear failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetTripleCount returns the number of triples in the repository
func (l *GraphDBLoader) GetTripleCount(ctx context.Context) (int64, error) {
	return l.runCountQuery(ctx, "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }")
}

// GetStatus returns the current status of the GraphDB repository
func (l *GraphDBLoader) GetStatus(ctx context.Context) *RepositoryStatus {
	status := &RepositoryStatus{
		Repository: l.config.GraphDBRepo,
	}

	if err := l.CheckConnection(ctx); err != nil {
		status.ErrorMessage = err.Error()
		return status
	}
	status.Available = true
	status.Writable = true

	count, err := l.GetTripleCount(ctx)
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("could not get triple count: %v", err)
	} else {
		status.TripleCount = count
	}

	return status
}

// Verify runs verification queries against the loaded data
func (l *GraphDBLoader) Verify(ctx context.Context) (*VerificationResult, error) {
	result := &VerificationResult{}

	// Total triples
	count, err := l.GetTripleCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get triple count: %w", err)
	}
	result.TotalTriples = count

	// SNOMED concepts
	snomedCount, err := l.runCountQuery(ctx, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://snomed.info/id/"))
		}
	`)
	if err != nil {
		l.logger.WithError(err).Warn("SNOMED count query failed")
	} else {
		result.SNOMEDConcepts = snomedCount
	}

	// RxNorm concepts
	rxnormCount, err := l.runCountQuery(ctx, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://purl.bioontology.org/ontology/RXNORM/"))
		}
	`)
	if err != nil {
		l.logger.WithError(err).Warn("RxNorm count query failed")
	} else {
		result.RxNormConcepts = rxnormCount
	}

	// LOINC concepts
	loincCount, err := l.runCountQuery(ctx, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://loinc.org/"))
		}
	`)
	if err != nil {
		l.logger.WithError(err).Warn("LOINC count query failed")
	} else {
		result.LOINCConcepts = loincCount
	}

	// All passed if we have substantial data
	result.AllPassed = result.TotalTriples > 1_000_000

	return result, nil
}

// runCountQuery executes a COUNT query and returns the result
func (l *GraphDBLoader) runCountQuery(ctx context.Context, query string) (int64, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", l.config.GraphDBURL, l.config.GraphDBRepo)

	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	if l.config.GraphDBUsername != "" {
		req.SetBasicAuth(l.config.GraphDBUsername, l.config.GraphDBPassword)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results struct {
			Bindings []struct {
				Count struct {
					Value string `json:"value"`
				} `json:"count"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Results.Bindings) == 0 {
		return 0, nil
	}

	var count int64
	fmt.Sscanf(result.Results.Bindings[0].Count.Value, "%d", &count)
	return count, nil
}
