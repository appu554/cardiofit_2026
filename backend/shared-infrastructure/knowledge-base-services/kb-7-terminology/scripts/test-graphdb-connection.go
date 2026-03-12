package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"
)

// Import the semantic package (adjust path as needed)
// Note: This assumes the semantic package is in the parent directory structure
// For standalone testing, you may need to adjust the import path

type GraphDBClient struct {
	baseURL    string
	repository string
}

func main() {
	fmt.Println("=== GraphDB Connectivity Test ===\n")

	// Test 1: Repository health check
	fmt.Println("Test 1: Repository Health Check")
	testHealthCheck()

	// Test 2: Simple SPARQL query
	fmt.Println("\nTest 2: Simple SPARQL Query")
	testSPARQLQuery()

	// Test 3: Count triples
	fmt.Println("\nTest 3: Count Existing Triples")
	testTripleCount()

	fmt.Println("\n✅ All connectivity tests passed!")
}

func testHealthCheck() {
	// Simple HTTP GET to repository endpoint
	resp, err := httpGet("http://localhost:7200/rest/repositories/kb7-terminology")
	if err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}
	fmt.Printf("✅ Repository accessible (Status: %d)\n", resp.StatusCode)
}

func testSPARQLQuery() {
	// Execute a simple ASK query
	query := "ASK { ?s ?p ?o }"
	resp, err := httpPost("http://localhost:7200/repositories/kb7-terminology",
		"query="+url.QueryEscape(query))
	if err != nil {
		log.Fatalf("❌ SPARQL query failed: %v", err)
	}
	fmt.Printf("✅ SPARQL endpoint responsive (Status: %d)\n", resp.StatusCode)
}

func testTripleCount() {
	// Count triples in repository
	query := "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
	resp, err := httpPost("http://localhost:7200/repositories/kb7-terminology",
		"query="+url.QueryEscape(query))
	if err != nil {
		log.Fatalf("❌ Triple count query failed: %v", err)
	}
	fmt.Printf("✅ Triple count query successful (Status: %d)\n", resp.StatusCode)
}

// Helper functions (simplified for testing)
func httpGet(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Get(url)
}

func httpPost(url, data string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
}
