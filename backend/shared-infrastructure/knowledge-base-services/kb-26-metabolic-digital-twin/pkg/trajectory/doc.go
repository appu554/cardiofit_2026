// Package trajectory provides public type aliases for the KB-26 domain
// trajectory models so that other modules (e.g. KB-23) can import them
// without violating Go's internal package visibility rule.
//
// All types are pure type aliases — they are identical to the originals
// in kb-26-metabolic-digital-twin/internal/models and can be used
// interchangeably.
//
// MAINTENANCE RULE: Do NOT define new types in this package. Add types to
// internal/models (the single source of truth) and re-export them here as
// aliases. This keeps the authoritative definitions private to KB-26
// while exposing exactly what cross-module consumers need.
package trajectory
