package ingestion

import "sync"

// VendorAdapter applies per-vendor transformations to a parsed
// observation before it is added to the result. Common quirks the
// adapter shape supports:
//   - vendor uses non-standard OBX-3 system value (e.g. "VENDOR-X-CODES"
//     instead of "LN") — the adapter rewrites LOINCCode to the standard
//     LOINC after a lookup
//   - vendor reports units in vendor-specific abbreviation rather than
//     UCUM — the adapter remaps Unit
//   - vendor's abnormal flag uses non-HL7 codes — the adapter
//     normalises AbnormalFlag
//
// The default genericVendorAdapter is a pass-through; per-vendor
// adapters are V1 work as agreements are signed (see ADR for
// deferral rationale).
type VendorAdapter interface {
	Adapt(po ParsedObservation) ParsedObservation
}

// genericVendorAdapter passes through ParsedObservation unchanged.
// It is the fallback for unknown vendor names AND the default
// adapter for vendors whose feed is already standards-conformant
// (LOINC + UCUM + HL7 abnormal flags) and needs no remapping.
type genericVendorAdapter struct{}

// Adapt returns its input unchanged.
func (g *genericVendorAdapter) Adapt(po ParsedObservation) ParsedObservation { return po }

// adapterRegistry guards the per-vendor adapter table. Concurrent reads
// happen on the parse path; writes happen at process startup (when
// vendors register their adapters) so a sync.RWMutex is the right
// shape here.
var (
	adapterRegistry   = map[string]VendorAdapter{}
	adapterRegistryMu sync.RWMutex
	defaultAdapter    VendorAdapter = &genericVendorAdapter{}
)

// RegisterVendorAdapter registers an adapter under the given vendor
// name. Subsequent ParseORUR01 calls with vendorName == name will
// route their observations through the registered adapter. Registration
// is process-global; tests that need isolation should use distinct
// vendor names.
//
// Re-registering the same name overwrites the prior adapter. This is
// intentional — V1 may want to swap in an updated adapter without
// process restart.
func RegisterVendorAdapter(name string, a VendorAdapter) {
	if name == "" || a == nil {
		return
	}
	adapterRegistryMu.Lock()
	defer adapterRegistryMu.Unlock()
	adapterRegistry[name] = a
}

// lookupVendorAdapter returns the registered adapter for the given
// vendor name, or the default pass-through adapter when no registration
// exists. Empty / "generic" vendor names always return the default.
func lookupVendorAdapter(name string) VendorAdapter {
	if name == "" || name == "generic" {
		return defaultAdapter
	}
	adapterRegistryMu.RLock()
	defer adapterRegistryMu.RUnlock()
	if a, ok := adapterRegistry[name]; ok {
		return a
	}
	return defaultAdapter
}
