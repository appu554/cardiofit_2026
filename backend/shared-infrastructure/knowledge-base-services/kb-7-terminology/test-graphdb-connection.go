package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"kb-7-terminology/internal/semantic"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{})
	
	fmt.Println("=== KB-7 GraphDB Connection Test ===\n")
	
	// Create GraphDB client
	client := semantic.NewGraphDBClient(
		"http://localhost:7200",
		"kb7-terminology",
		logger,
	)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Test 1: Health Check
	fmt.Println("Test 1: GraphDB Health Check")
	err := client.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}
	fmt.Println("✅ GraphDB is healthy and accessible\n")
	
	// Test 2: Get Repository Info
	fmt.Println("Test 2: Repository Information")
	info, err := client.GetRepositoryInfo(ctx)
	if err != nil {
		fmt.Printf("ℹ️  Repository 'kb7-terminology' doesn't exist yet: %v\n", err)
		fmt.Println("   This is normal for a fresh GraphDB instance\n")
	} else {
		fmt.Printf("✅ Repository info: %+v\n\n", info)
	}
	
	// Test 3: Insert sample RDF data
	fmt.Println("Test 3: Insert Sample Clinical Concept")
	sampleTriples := []semantic.TripleData{
		{
			Subject:   "http://cardiofit.ai/kb7/concept/test-001",
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    "http://cardiofit.ai/kb7/ontology#ClinicalConcept",
		},
		{
			Subject:   "http://cardiofit.ai/kb7/concept/test-001",
			Predicate: "http://www.w3.org/2000/01/rdf-schema#label",
			Object:    "Test Clinical Concept",
		},
		{
			Subject:   "http://cardiofit.ai/kb7/concept/test-001",
			Predicate: "http://cardiofit.ai/kb7/ontology#code",
			Object:    "TEST-001",
		},
	}
	
	err = client.InsertTriples(ctx, sampleTriples)
	if err != nil {
		fmt.Printf("ℹ️  Cannot insert data (repository may not exist): %v\n", err)
		fmt.Println("   To create repository, visit: http://localhost:7200\n")
	} else {
		fmt.Println("✅ Sample data inserted successfully\n")
		
		// Test 4: Query the data back
		fmt.Println("Test 4: Query Sample Data")
		query := &semantic.SPARQLQuery{
			Query: `
				PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
				PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
				
				SELECT ?concept ?label ?code WHERE {
					?concept a kb7:ClinicalConcept ;
						rdfs:label ?label ;
						kb7:code ?code .
				}
				LIMIT 10
			`,
		}
		
		results, err := client.ExecuteSPARQL(ctx, query)
		if err != nil {
			fmt.Printf("⚠️  Query failed: %v\n", err)
		} else {
			fmt.Printf("✅ Query successful! Found %d results\n", len(results.Results.Bindings))
			for i, binding := range results.Results.Bindings {
				fmt.Printf("   Result %d: %+v\n", i+1, binding)
			}
		}
	}
	
	fmt.Println("\n=== Connection Test Complete ===")
	fmt.Println("\n📊 Next Steps:")
	fmt.Println("1. Create repository via Web UI: http://localhost:7200")
	fmt.Println("   - Repository ID: kb7-terminology")
	fmt.Println("   - Ruleset: OWL2-RL (Optimized)")
	fmt.Println("2. Or use GraphDB Workbench to import ontology files")
	fmt.Println("3. Then re-run this test to verify data insertion\n")
}
