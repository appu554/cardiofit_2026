package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type SPARQLProxy struct {
	graphdbURL   string
	repositoryID string
	redisClient  *redis.Client
	cacheTTL     time.Duration
}

type SPARQLRequest struct {
	Query  string `json:"query" binding:"required"`
	Format string `json:"format,omitempty"`
}

type SPARQLResponse struct {
	Head    SPARQLHead           `json:"head"`
	Results SPARQLResults        `json:"results"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

type SPARQLHead struct {
	Vars []string `json:"vars"`
}

type SPARQLResults struct {
	Bindings []map[string]SPARQLBinding `json:"bindings"`
}

type SPARQLBinding struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	DataType string `json:"datatype,omitempty"`
	Lang     string `json:"xml:lang,omitempty"`
}

func NewSPARQLProxy() *SPARQLProxy {
	graphdbURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	repositoryID := getEnv("REPOSITORY_ID", "kb7-terminology")
	cacheTTL, _ := time.ParseDuration(getEnv("CACHE_TTL", "300s"))

	// Redis client for caching
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_URL", "redis-semantic:6379"),
		Password: "",
		DB:       0,
	})

	return &SPARQLProxy{
		graphdbURL:   graphdbURL,
		repositoryID: repositoryID,
		redisClient:  redisClient,
		cacheTTL:     cacheTTL,
	}
}

func (sp *SPARQLProxy) executeQuery(ctx context.Context, query string, format string) (*SPARQLResponse, error) {
	// Check cache first for read-only queries
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "ASK") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DESCRIBE") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "CONSTRUCT") {

		cacheKey := fmt.Sprintf("sparql:%s:%s", hashQuery(query), format)
		cached, err := sp.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var response SPARQLResponse
			if json.Unmarshal([]byte(cached), &response) == nil {
				log.Printf("Cache hit for query: %s", truncateQuery(query))
				return &response, nil
			}
		}
	}

	// Execute query against GraphDB
	endpoint := fmt.Sprintf("%s/repositories/%s", sp.graphdbURL, sp.repositoryID)

	// Prepare request
	data := url.Values{}
	data.Set("query", query)
	if format == "" {
		format = "application/sparql-results+json"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", format)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphDB error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var response SPARQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Add metadata
	response.Meta = map[string]interface{}{
		"executionTime": time.Now().Format(time.RFC3339),
		"repository":    sp.repositoryID,
		"cached":        false,
	}

	// Cache read-only queries
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "ASK") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DESCRIBE") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "CONSTRUCT") {

		cacheKey := fmt.Sprintf("sparql:%s:%s", hashQuery(query), format)
		cached, _ := json.Marshal(response)
		sp.redisClient.Set(ctx, cacheKey, cached, sp.cacheTTL)
	}

	log.Printf("Executed query: %s", truncateQuery(query))
	return &response, nil
}

func (sp *SPARQLProxy) setupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		// Test connection to GraphDB
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		healthURL := fmt.Sprintf("%s/rest/repositories", sp.graphdbURL)
		req, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		if err != nil || resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "GraphDB connection failed",
			})
			return
		}
		resp.Body.Close()

		c.JSON(http.StatusOK, gin.H{
			"status":     "healthy",
			"graphdb":    sp.graphdbURL,
			"repository": sp.repositoryID,
			"timestamp":  time.Now().Format(time.RFC3339),
		})
	})

	// SPARQL endpoint
	r.POST("/sparql", func(c *gin.Context) {
		var req SPARQLRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		response, err := sp.executeQuery(ctx, req.Query, req.Format)
		if err != nil {
			log.Printf("Query error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Clinical terminology queries
	r.GET("/terminology/concept/:id", func(c *gin.Context) {
		conceptID := c.Param("id")

		query := fmt.Sprintf(`
		PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

		SELECT ?property ?value WHERE {
			<%s> ?property ?value .
		}
		`, conceptID)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := sp.executeQuery(ctx, query, "application/sparql-results+json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Mapping queries
	r.GET("/terminology/mapping", func(c *gin.Context) {
		source := c.Query("source")
		target := c.Query("target")

		if source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source parameter required"})
			return
		}

		var query string
		if target != "" {
			query = fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

			SELECT ?mapping ?property ?confidence WHERE {
				?mapping a kb7:ConceptMapping ;
					kb7:sourceCode "%s" ;
					kb7:targetCode "%s" ;
					?property ?confidence .
			}
			`, source, target)
		} else {
			query = fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

			SELECT ?targetCode ?property ?confidence WHERE {
				?mapping a kb7:ConceptMapping ;
					kb7:sourceCode "%s" ;
					kb7:targetCode ?targetCode ;
					?property ?confidence .
			}
			`, source)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := sp.executeQuery(ctx, query, "application/sparql-results+json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	return r
}

func main() {
	proxy := NewSPARQLProxy()
	router := proxy.setupRoutes()

	port := getEnv("PORT", "8095")
	log.Printf("Starting SPARQL Proxy on port %s", port)
	log.Printf("GraphDB URL: %s", proxy.graphdbURL)
	log.Printf("Repository: %s", proxy.repositoryID)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// Utility functions
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func hashQuery(query string) string {
	// Simple hash for caching - in production use proper hash function
	return fmt.Sprintf("%x", len(query)+int(query[0]))
}

func truncateQuery(query string) string {
	if len(query) > 100 {
		return query[:100] + "..."
	}
	return query
}