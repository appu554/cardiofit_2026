// Package clients provides HTTP clients for KB service integrations
// This file provides type aliases for backward compatibility
package clients

// Type aliases for backward compatibility with pkg code
// These allow pkg code to use shorter names while the actual implementations
// have more descriptive names

// KB1Client is an alias for KB1DosingClient
type KB1Client = KB1DosingClient

// KB3Client is an alias for KB3TemporalClient
type KB3Client = KB3TemporalClient

// KB6Client is an alias for KB6FormularyClient
type KB6Client = KB6FormularyClient

// KB7Client is an alias for KB7TerminologyClient
type KB7Client = KB7TerminologyClient
