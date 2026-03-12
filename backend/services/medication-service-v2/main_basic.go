package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting medication-service-v2 (basic version)")

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define a basic GraphQL schema for testing
	medicationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Medication",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"dosage": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"medication": &graphql.Field{
				Type: medicationType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id := params.Args["id"].(string)
					return map[string]interface{}{
						"id":     id,
						"name":   "Sample Medication",
						"dosage": "10mg",
					}, nil
				},
			},
			"_service": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "_Service",
					Fields: graphql.Fields{
						"sdl": &graphql.Field{
							Type: graphql.String,
						},
					},
				}),
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					sdl := `
						type Medication @key(fields: "id") {
							id: ID!
							name: String!
							dosage: String
						}

						type Query {
							medication(id: ID!): Medication
						}
					`
					return map[string]interface{}{
						"sdl": sdl,
					}, nil
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
	})
	if err != nil {
		logger.Fatal("Failed to create schema", zap.Error(err))
	}

	// Start GraphQL server for Apollo Federation
	go func() {
		router := gin.New()
		router.Use(gin.Logger())
		router.Use(gin.Recovery())

		// Enable CORS
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
		})

		// GraphQL endpoint
		router.POST("/graphql", func(c *gin.Context) {
			var request struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			result := graphql.Do(graphql.Params{
				Schema:         schema,
				RequestString:  request.Query,
				VariableValues: request.Variables,
				Context:        ctx,
			})

			c.JSON(200, result)
		})

		// Federation endpoint
		router.GET("/federation", func(c *gin.Context) {
			sdl := `
				type Medication @key(fields: "id") {
					id: ID!
					name: String!
					dosage: String
				}

				type Query {
					medication(id: ID!): Medication
				}
			`
			c.Header("Content-Type", "text/plain")
			c.String(200, sdl)
		})

		// Health check endpoint
		router.GET("/health", func(c *gin.Context) {
			status := map[string]interface{}{
				"service":   "medication-service-v2",
				"status":    "healthy",
				"timestamp": time.Now().Format(time.RFC3339),
				"version":   "2.0.0-basic",
			}
			c.JSON(200, status)
		})

		// GraphQL Playground
		router.GET("/graphql", func(c *gin.Context) {
			playground := `<!DOCTYPE html>
<html>
<head>
	<meta charset=utf-8/>
	<meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
	<title>GraphQL Playground</title>
	<link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
	<link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
	<script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
	<div id="root">
		<style>
			body { background-color: rgb(23, 42, 58); font-family: Open Sans, sans-serif; height: 90vh; }
			#root { height: 100%; width: 100%; display: flex; align-items: center; justify-content: center; }
			.loading { font-size: 32px; font-weight: 200; color: rgba(255, 255, 255, .6); margin-left: 20px; }
			img { width: 78px; height: 78px; }
			.title { font-weight: 400; }
		</style>
		<img src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png" alt="">
		<div class="loading"> Loading
			<span class="title">GraphQL Playground</span>
		</div>
	</div>
	<script>window.addEventListener('load', function (event) {
		GraphQLPlayground.init(document.getElementById('root'), {
			endpoint: '/graphql'
		})
	})</script>
</body>
</html>`
			c.Header("Content-Type", "text/html")
			c.String(200, playground)
		})

		// Start server
		port := getEnvInt("GRAPHQL_PORT", 8005)
		logger.Info("Starting GraphQL Federation server",
			zap.String("port", fmt.Sprintf("%d", port)),
			zap.String("federation_url", fmt.Sprintf("http://localhost:%d/federation", port)),
			zap.String("playground_url", fmt.Sprintf("http://localhost:%d/graphql", port)))

		srv := &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("GraphQL server failed", zap.Error(err))
		}
	}()

	logger.Info("medication-service-v2 started successfully!")
	logger.Info("🚀 GraphQL Federation endpoint: http://localhost:8005/federation")
	logger.Info("🎮 GraphQL Playground: http://localhost:8005/graphql")
	logger.Info("💚 Health check: http://localhost:8005/health")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down medication-service-v2...")
	logger.Info("Shutdown complete")
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}