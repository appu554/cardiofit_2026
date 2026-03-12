package graphql

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
)

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []string    `json:"errors,omitempty"`
}

// ServeHTTP handles GraphQL HTTP requests
func (h *GraphQLHandler) ServeHTTP(c *gin.Context) {
	// Handle OPTIONS for CORS
	if c.Request.Method == http.MethodOptions {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Status(http.StatusOK)
		return
	}

	// Only allow POST for GraphQL
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "GraphQL only accepts POST requests",
		})
		return
	}

	// Parse request body
	var request GraphQLRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON in request body",
		})
		return
	}

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         h.schema,
		RequestString:  request.Query,
		VariableValues: request.Variables,
		OperationName:  request.OperationName,
		Context:        c.Request.Context(),
	})

	// Build response
	response := GraphQLResponse{
		Data: result.Data,
	}

	// Add errors if any
	if result.HasErrors() {
		errors := make([]string, len(result.Errors))
		for i, err := range result.Errors {
			errors[i] = err.Error()
		}
		response.Errors = errors
	}

	// Set CORS headers
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Return response
	c.JSON(http.StatusOK, response)
}

// HandleIntrospection handles GraphQL introspection queries
func (h *GraphQLHandler) HandleIntrospection(c *gin.Context) {
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
		Schema:        h.schema,
		RequestString: introspectionQuery,
		Context:       c.Request.Context(),
	})

	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, result)
}

// HandleSDL handles federation SDL requests
func (h *GraphQLHandler) HandleSDL(c *gin.Context) {
	sdlQuery := `
		query {
			_service {
				sdl
			}
		}
	`

	result := graphql.Do(graphql.Params{
		Schema:        h.schema,
		RequestString: sdlQuery,
		Context:       c.Request.Context(),
	})

	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, result)
}