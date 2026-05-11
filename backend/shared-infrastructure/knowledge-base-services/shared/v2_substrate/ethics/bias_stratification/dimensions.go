// Package bias_stratification produces per-stratum metric means across the six
// equity-audit dimensions named in Ethical Architecture Implementation
// Guidelines v1.0 §7.2 Mechanism 1. The output map[string]float64 shape is
// the input contract of pattern_detection.DetectBiasDisparity.
package bias_stratification

// Dimension identifies one of the six demographic axes the platform audits
// for outcome disparity.
type Dimension string

const (
	DimAgeBand     Dimension = "age_band"
	DimSex         Dimension = "sex"
	DimFrailtyTier Dimension = "frailty_tier"
	DimCALD        Dimension = "cald_background"
	DimSocioecon   Dimension = "socioeconomic_indicator"
	DimFacility    Dimension = "facility_geography"
)

// AllDimensions enumerates the six audit dimensions in stable order: age,
// sex, frailty, CALD, socioeconomic, facility/geography.
var AllDimensions = []Dimension{
	DimAgeBand,
	DimSex,
	DimFrailtyTier,
	DimCALD,
	DimSocioecon,
	DimFacility,
}

// AgeBand classifies a chronological age into one of four canonical bands
// (under_65, 65-74, 75-84, 85+). Boundaries are inclusive on the lower
// bound: 65 → 65-74, 75 → 75-84, 85 → 85+.
func AgeBand(age int) string {
	switch {
	case age < 65:
		return "under_65"
	case age < 75:
		return "65-74"
	case age < 85:
		return "75-84"
	default:
		return "85+"
	}
}
