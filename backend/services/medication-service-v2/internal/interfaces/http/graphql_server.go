package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"go.uber.org/zap"

	"medication-service-v2/internal/graphql/resolvers"
	medicationschema "medication-service-v2/internal/graphql"
)

// GraphQLServer handles GraphQL requests for Apollo Federation
type GraphQLServer struct {
	schema   graphql.Schema
	resolver *resolvers.MedicationResolver
	logger   *zap.Logger
}

// NewGraphQLServer creates a new GraphQL server
func NewGraphQLServer(
	resolver *resolvers.MedicationResolver,
	logger *zap.Logger,
) (*GraphQLServer, error) {
	// Build the GraphQL schema
	schema, err := medicationschema.BuildSchema(resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to build GraphQL schema: %w", err)
	}

	return &GraphQLServer{
		schema:   schema,
		resolver: resolver,
		logger:   logger,
	}, nil
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}              `json:"data,omitempty"`
	Errors []GraphQLError           `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLErrorLocation represents the location of a GraphQL error
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// RegisterRoutes registers GraphQL routes with the Gin router
func (s *GraphQLServer) RegisterRoutes(router *gin.Engine) {
	// GraphQL endpoint for Apollo Federation
	router.POST("/federation", s.handleGraphQL)
	router.GET("/federation", s.handleGraphQL)

	// Schema SDL endpoint for federation introspection
	router.GET("/federation/sdl", s.handleSDL)

	// Playground endpoint for development
	router.GET("/graphql", s.handlePlayground)
	router.POST("/graphql", s.handleGraphQL)
}

// handleGraphQL handles GraphQL requests
func (s *GraphQLServer) handleGraphQL(c *gin.Context) {
	var req GraphQLRequest

	if c.Request.Method == http.MethodGet {
		// Handle GET requests with query parameters
		req.Query = c.Query("query")
		req.OperationName = c.Query("operationName")

		if variables := c.Query("variables"); variables != "" {
			if err := json.Unmarshal([]byte(variables), &req.Variables); err != nil {
				s.respondWithError(c, http.StatusBadRequest, "Invalid variables parameter")
				return
			}
		}
	} else {
		// Handle POST requests with JSON body
		if err := c.ShouldBindJSON(&req); err != nil {
			s.respondWithError(c, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	// Log the GraphQL request
	s.logger.Info("GraphQL request",
		zap.String("query", strings.ReplaceAll(req.Query, "\n", " ")),
		zap.String("operationName", req.OperationName),
		zap.Any("variables", req.Variables),
	)

	// Execute the GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         s.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        c.Request.Context(),
	})

	// Convert GraphQL result to our response format
	response := s.convertResult(result)

	// Set appropriate headers
	c.Header("Content-Type", "application/json")

	// Return the response
	c.JSON(http.StatusOK, response)
}

// handleSDL handles requests for the schema definition language
func (s *GraphQLServer) handleSDL(c *gin.Context) {
	sdl := medicationschema.GetFederationSDL(s.schema)

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, sdl)
}

// handlePlayground serves a GraphQL playground for development
func (s *GraphQLServer) handlePlayground(c *gin.Context) {
	playground := s.getPlaygroundHTML()
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, playground)
}

// convertResult converts a GraphQL result to our response format
func (s *GraphQLServer) convertResult(result *graphql.Result) GraphQLResponse {
	response := GraphQLResponse{
		Data: result.Data,
	}

	// Convert errors
	if len(result.Errors) > 0 {
		response.Errors = make([]GraphQLError, len(result.Errors))
		for i, err := range result.Errors {
			graphqlErr := GraphQLError{
				Message: err.Message,
			}

			// Convert locations
			if len(err.Locations) > 0 {
				graphqlErr.Locations = make([]GraphQLErrorLocation, len(err.Locations))
				for j, loc := range err.Locations {
					graphqlErr.Locations[j] = GraphQLErrorLocation{
						Line:   loc.Line,
						Column: loc.Column,
					}
				}
			}

			// Convert path
			if len(err.Path) > 0 {
				graphqlErr.Path = err.Path
			}

			// Convert extensions
			if len(err.Extensions) > 0 {
				graphqlErr.Extensions = err.Extensions
			}

			response.Errors[i] = graphqlErr
		}
	}

	// Add extensions
	if len(result.Extensions) > 0 {
		response.Extensions = result.Extensions
	}

	return response
}

// respondWithError sends an error response
func (s *GraphQLServer) respondWithError(c *gin.Context, statusCode int, message string) {
	response := GraphQLResponse{
		Errors: []GraphQLError{
			{
				Message: message,
			},
		},
	}

	c.JSON(statusCode, response)
}

// getPlaygroundHTML returns the HTML for GraphQL Playground
func (s *GraphQLServer) getPlaygroundHTML() string {
	return `
<!DOCTYPE html>
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
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }
      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }
      img {
        width: 78px;
        height: 78px;
      }
      .title {
        font-weight: 400;
      }
    </style>
    <img src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png" alt="">
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '/graphql',
        settings: {
          'editor.theme': 'dark',
          'editor.cursorShape': 'line',
          'editor.reuseHeaders': true,
          'tracing.hideTracingResponse': true,
          'editor.fontSize': 14,
          'editor.fontFamily': '"Source Code Pro", "Consolas", "Inconsolata", "Droid Sans Mono", "Monaco", monospace',
          'request.credentials': 'omit',
        },
      })
    })</script>
</body>
</html>
`
}

// Health check handler specifically for GraphQL service
func (s *GraphQLServer) HandleHealthCheck(c *gin.Context) {
	// Perform a simple introspection query to verify schema is working
	result := graphql.Do(graphql.Params{
		Schema:        s.schema,
		RequestString: "{ __schema { types { name } } }",
		Context:       c.Request.Context(),
	})

	if len(result.Errors) > 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "GraphQL schema validation failed",
			"details": result.Errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "healthy",
		"service":    "medication-service-v2",
		"component":  "graphql",
		"federation": "enabled",
		"schema":     "valid",
	})
}

// Federation-specific endpoints

// HandleFederationCheck verifies federation capability
func (s *GraphQLServer) HandleFederationCheck(c *gin.Context) {
	// Test the _service query which is required for federation
	result := graphql.Do(graphql.Params{
		Schema:        s.schema,
		RequestString: "{ _service { sdl } }",
		Context:       c.Request.Context(),
	})

	if len(result.Errors) > 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Federation _service query failed",
			"details": result.Errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "healthy",
		"service":           "medication-service-v2",
		"federation":        "ready",
		"entities":          []string{"Medication", "MedicationRequest", "Patient"},
		"federationVersion": "2.0",
		"sdlAvailable":      true,
	})
}

// HandleIntrospection handles schema introspection for federation
func (s *GraphQLServer) HandleIntrospection(c *gin.Context) {
	introspectionQuery := `
	query IntrospectionQuery {
		__schema {
			queryType { name }
			mutationType { name }
			subscriptionType { name }
			types {
				...FullType
			}
			directives {
				name
				description
				locations
				args {
					...InputValue
				}
			}
		}
	}

	fragment FullType on __Type {
		kind
		name
		description
		fields(includeDeprecated: true) {
			name
			description
			args {
				...InputValue
			}
			type {
				...TypeRef
			}
			isDeprecated
			deprecationReason
		}
		inputFields {
			...InputValue
		}
		interfaces {
			...TypeRef
		}
		enumValues(includeDeprecated: true) {
			name
			description
			isDeprecated
			deprecationReason
		}
		possibleTypes {
			...TypeRef
		}
	}

	fragment InputValue on __InputValue {
		name
		description
		type { ...TypeRef }
		defaultValue
	}

	fragment TypeRef on __Type {
		kind
		name
		ofType {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
							ofType {
								kind
								name
								ofType {
									kind
									name
								}
							}
						}
					}
				}
			}
		}
	}
	`

	result := graphql.Do(graphql.Params{
		Schema:        s.schema,
		RequestString: introspectionQuery,
		Context:       c.Request.Context(),
	})

	response := s.convertResult(result)
	c.JSON(http.StatusOK, response)
}