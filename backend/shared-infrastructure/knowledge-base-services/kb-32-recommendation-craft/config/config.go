// Package config provides runtime configuration for kb-32-recommendation-craft.
// Task 1 ships a minimal placeholder; Tasks 4–12 will extend this as each
// pipeline stage acquires its own tuneable parameters.
package config

import (
	"os"
	"strings"
)

// Config holds the resolved runtime configuration for the service.
// Fields are exported so they can be passed to handler constructors.
type Config struct {
	// Port is the TCP port the HTTP server listens on (default: "8150").
	Port string

	// VaidshalaDS is the PostgreSQL connection string for the Vaidshala schema.
	VaidshalaDS string

	// JWTSecret is used by the JWT middleware wired in Task 13.
	// Empty only when DevMode is true.
	JWTSecret string

	// DevMode disables JWT enforcement (KB32_DEV_MODE=true).
	DevMode bool
}

// Load reads all kb-32 configuration from environment variables.
// It does NOT validate — validation is performed in main.go before Load is
// called, so Load can assume required vars are present.
func Load() Config {
	return Config{
		Port:        getenv("PORT", "8150"),
		VaidshalaDS: os.Getenv("VAIDSHALA_DSN"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		DevMode:     strings.EqualFold(os.Getenv("KB32_DEV_MODE"), "true"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
