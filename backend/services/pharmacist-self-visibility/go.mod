module github.com/cardiofit/pharmacist-self-visibility

go 1.24.1

require (
	github.com/cardiofit/shared v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
)

require github.com/golang-jwt/jwt/v5 v5.2.0

replace github.com/cardiofit/shared => ../../shared-infrastructure/knowledge-base-services/shared
