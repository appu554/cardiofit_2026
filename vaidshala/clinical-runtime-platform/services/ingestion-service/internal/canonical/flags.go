package canonical

// Flag represents a quality or clinical annotation attached to a
// CanonicalObservation during pipeline processing. Multiple flags can
// be present on a single observation.
type Flag string

const (
	FlagCriticalValue Flag = "CRITICAL_VALUE"
	FlagImplausible   Flag = "IMPLAUSIBLE"
	FlagLowQuality    Flag = "LOW_QUALITY"
	FlagUnmappedCode  Flag = "UNMAPPED_CODE"
	FlagStale         Flag = "STALE"
	FlagDuplicate     Flag = "DUPLICATE"
	FlagManualEntry   Flag = "MANUAL_ENTRY"
)
