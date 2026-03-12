// Package reference provides context-aware lab reference range selection
// Phase 3b.6: Range Selection Algorithm with specificity scoring
package reference

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/types"
)

// RangeSelector handles context-aware reference range selection
type RangeSelector struct {
	db *gorm.DB
}

// NewRangeSelector creates a new RangeSelector instance
func NewRangeSelector(db *gorm.DB) *RangeSelector {
	return &RangeSelector{db: db}
}

// SelectRange finds the most specific matching reference range for a patient and lab test
// Algorithm:
//   1. Get all active ranges for the LOINC code
//   2. Filter to ranges where ALL non-null conditions match the patient
//   3. Sort by specificity_score descending
//   4. Return highest scoring match, or default range if no match
func (s *RangeSelector) SelectRange(ctx context.Context, loincCode string, patient *types.PatientContext) (*ConditionalReferenceRange, error) {
	// 1. Get all active ranges for this LOINC code
	ranges, err := s.getRangesForLOINC(ctx, loincCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get ranges for LOINC %s: %w", loincCode, err)
	}

	if len(ranges) == 0 {
		return nil, fmt.Errorf("no reference ranges found for LOINC %s", loincCode)
	}

	// 2. Filter to ranges where ALL conditions match
	var matching []*ConditionalReferenceRange
	for _, r := range ranges {
		if s.conditionsMatch(&r.RangeConditions, patient) {
			matching = append(matching, r)
		}
	}

	// 3. If no specific match, look for default range (all conditions null)
	if len(matching) == 0 {
		defaultRange := s.findDefaultRange(ranges)
		if defaultRange != nil {
			return defaultRange, nil
		}
		return nil, fmt.Errorf("no matching reference range for LOINC %s with patient context", loincCode)
	}

	// 4. Sort by specificity score descending, return highest
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].SpecificityScore > matching[j].SpecificityScore
	})

	return matching[0], nil
}

// SelectRangeByTestID finds the most specific matching reference range by lab_test_id
func (s *RangeSelector) SelectRangeByTestID(ctx context.Context, labTestID uuid.UUID, patient *types.PatientContext) (*ConditionalReferenceRange, error) {
	// Get all active ranges for this lab test
	var ranges []*ConditionalReferenceRange
	err := s.db.WithContext(ctx).
		Where("lab_test_id = ? AND is_active = true", labTestID).
		Where("effective_date <= ? AND (expiration_date IS NULL OR expiration_date > ?)", time.Now(), time.Now()).
		Find(&ranges).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ranges for lab test %s: %w", labTestID, err)
	}

	if len(ranges) == 0 {
		return nil, fmt.Errorf("no reference ranges found for lab test %s", labTestID)
	}

	// Filter and sort (same logic as SelectRange)
	var matching []*ConditionalReferenceRange
	for _, r := range ranges {
		if s.conditionsMatch(&r.RangeConditions, patient) {
			matching = append(matching, r)
		}
	}

	if len(matching) == 0 {
		defaultRange := s.findDefaultRange(ranges)
		if defaultRange != nil {
			return defaultRange, nil
		}
		return nil, fmt.Errorf("no matching reference range for lab test %s with patient context", labTestID)
	}

	sort.Slice(matching, func(i, j int) bool {
		return matching[i].SpecificityScore > matching[j].SpecificityScore
	})

	return matching[0], nil
}

// getRangesForLOINC retrieves all active ranges for a LOINC code
func (s *RangeSelector) getRangesForLOINC(ctx context.Context, loincCode string) ([]*ConditionalReferenceRange, error) {
	var labTest LabTest
	err := s.db.WithContext(ctx).
		Where("loinc_code = ? AND is_active = true", loincCode).
		First(&labTest).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lab test with LOINC %s not found", loincCode)
		}
		return nil, err
	}

	var ranges []*ConditionalReferenceRange
	now := time.Now()
	err = s.db.WithContext(ctx).
		Where("lab_test_id = ? AND is_active = true", labTest.ID).
		Where("effective_date <= ? AND (expiration_date IS NULL OR expiration_date > ?)", now, now).
		Find(&ranges).Error
	if err != nil {
		return nil, err
	}

	return ranges, nil
}

// conditionsMatch checks if ALL non-null conditions in the range match the patient
// Key principle: NULL condition = matches any patient for that condition
func (s *RangeSelector) conditionsMatch(cond *RangeConditions, patient *types.PatientContext) bool {
	// Gender check
	if cond.Gender != nil && *cond.Gender != patient.Sex {
		return false
	}

	// Age in years check
	if cond.AgeMinYears != nil && float64(patient.Age) < *cond.AgeMinYears {
		return false
	}
	if cond.AgeMaxYears != nil && float64(patient.Age) >= *cond.AgeMaxYears {
		return false
	}

	// Age in days check (for neonates)
	if cond.AgeMinDays != nil && patient.AgeInDays < *cond.AgeMinDays {
		return false
	}
	if cond.AgeMaxDays != nil && patient.AgeInDays >= *cond.AgeMaxDays {
		return false
	}

	// Pregnancy check
	if cond.IsPregnant != nil {
		if *cond.IsPregnant != patient.IsPregnant {
			return false
		}
		// If pregnant, check trimester
		if *cond.IsPregnant && cond.Trimester != nil {
			if *cond.Trimester != patient.Trimester {
				return false
			}
		}
	}

	// Postpartum check
	if cond.IsPostpartum != nil && *cond.IsPostpartum != patient.IsPostpartum {
		return false
	}

	// Lactation check
	if cond.IsLactating != nil && *cond.IsLactating != patient.IsLactating {
		return false
	}

	// CKD stage check
	if cond.CKDStage != nil && *cond.CKDStage != patient.CKDStage {
		return false
	}

	// Dialysis check
	if cond.IsOnDialysis != nil && *cond.IsOnDialysis != patient.IsOnDialysis {
		return false
	}

	// eGFR range check
	if cond.EGFRMin != nil && patient.EGFR < *cond.EGFRMin {
		return false
	}
	if cond.EGFRMax != nil && patient.EGFR >= *cond.EGFRMax {
		return false
	}

	// Neonatal gestational age check
	if cond.GestationalAgeWeeksMin != nil {
		if patient.GestationalAgeAtBirth < *cond.GestationalAgeWeeksMin {
			return false
		}
	}
	if cond.GestationalAgeWeeksMax != nil {
		if patient.GestationalAgeAtBirth > *cond.GestationalAgeWeeksMax {
			return false
		}
	}

	// Hours of life check (for neonatal ranges)
	if cond.HoursOfLifeMin != nil && patient.HoursOfLife < *cond.HoursOfLifeMin {
		return false
	}
	if cond.HoursOfLifeMax != nil && patient.HoursOfLife >= *cond.HoursOfLifeMax {
		return false
	}

	// All conditions match
	return true
}

// findDefaultRange finds the most generic range (all conditions null or minimal)
func (s *RangeSelector) findDefaultRange(ranges []*ConditionalReferenceRange) *ConditionalReferenceRange {
	// Look for ranges with specificity_score = 0 or 1 (most generic)
	for _, r := range ranges {
		if r.SpecificityScore <= 1 {
			// Verify it's truly a default (minimal conditions)
			if s.isDefaultRange(&r.RangeConditions) {
				return r
			}
		}
	}

	// If no true default, return the lowest specificity score
	if len(ranges) > 0 {
		sort.Slice(ranges, func(i, j int) bool {
			return ranges[i].SpecificityScore < ranges[j].SpecificityScore
		})
		return ranges[0]
	}

	return nil
}

// isDefaultRange checks if a range has minimal/no conditions
func (s *RangeSelector) isDefaultRange(cond *RangeConditions) bool {
	// Check if most conditions are null (only age/gender might be set)
	if cond.IsPregnant != nil ||
		cond.Trimester != nil ||
		cond.CKDStage != nil ||
		cond.IsOnDialysis != nil ||
		cond.GestationalAgeWeeksMin != nil ||
		cond.HoursOfLifeMin != nil {
		return false
	}
	return true
}

// GetAllMatchingRanges returns all ranges that match the patient (sorted by specificity)
// Useful for debugging or showing alternative interpretations
func (s *RangeSelector) GetAllMatchingRanges(ctx context.Context, loincCode string, patient *types.PatientContext) ([]*ConditionalReferenceRange, error) {
	ranges, err := s.getRangesForLOINC(ctx, loincCode)
	if err != nil {
		return nil, err
	}

	var matching []*ConditionalReferenceRange
	for _, r := range ranges {
		if s.conditionsMatch(&r.RangeConditions, patient) {
			matching = append(matching, r)
		}
	}

	// Sort by specificity descending
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].SpecificityScore > matching[j].SpecificityScore
	})

	return matching, nil
}

// InterpretLabValue interprets a lab value using context-aware range selection
func (s *RangeSelector) InterpretLabValue(ctx context.Context, loincCode string, value float64, unit string, patient *types.PatientContext) (*RangeInterpretation, error) {
	// Select the most specific matching range
	selectedRange, err := s.SelectRange(ctx, loincCode, patient)
	if err != nil {
		return nil, fmt.Errorf("failed to select range: %w", err)
	}

	// Interpret the value against the selected range
	interpretation := selectedRange.Interpret(value, unit)
	return interpretation, nil
}

// CalculateSpecificityScore computes the specificity score for a set of conditions
// Higher score = more specific condition set
// Used when creating new conditional ranges
func CalculateSpecificityScore(cond *RangeConditions) int {
	score := 0

	// Demographics: +1 each
	if cond.Gender != nil {
		score++
	}
	if cond.AgeMinYears != nil || cond.AgeMaxYears != nil {
		score++
	}
	if cond.AgeMinDays != nil || cond.AgeMaxDays != nil {
		score++
	}

	// Pregnancy: +2 for pregnant, +1 more for trimester
	if cond.IsPregnant != nil && *cond.IsPregnant {
		score += 2
		if cond.Trimester != nil {
			score++
		}
	}

	// CKD: +2 for stage
	if cond.CKDStage != nil {
		score += 2
	}

	// Dialysis: +3 (most specific renal condition)
	if cond.IsOnDialysis != nil && *cond.IsOnDialysis {
		score += 3
	}

	// Neonatal: +2 for GA, +1 more for hours of life
	if cond.GestationalAgeWeeksMin != nil || cond.GestationalAgeWeeksMax != nil {
		score += 2
		if cond.HoursOfLifeMin != nil || cond.HoursOfLifeMax != nil {
			score++
		}
	}

	// Postpartum/Lactation: +1 each
	if cond.IsPostpartum != nil && *cond.IsPostpartum {
		score++
	}
	if cond.IsLactating != nil && *cond.IsLactating {
		score++
	}

	return score
}

// ValidateRangeConsistency checks that range values are logically consistent
func ValidateRangeConsistency(r *ConditionalReferenceRange) error {
	// Low must be less than High
	if r.LowNormal != nil && r.HighNormal != nil {
		if *r.LowNormal >= *r.HighNormal {
			return fmt.Errorf("low_normal (%v) must be less than high_normal (%v)", *r.LowNormal, *r.HighNormal)
		}
	}

	// Critical must be outside normal
	if r.CriticalLow != nil && r.LowNormal != nil {
		if *r.CriticalLow >= *r.LowNormal {
			return fmt.Errorf("critical_low (%v) must be less than low_normal (%v)", *r.CriticalLow, *r.LowNormal)
		}
	}
	if r.CriticalHigh != nil && r.HighNormal != nil {
		if *r.CriticalHigh <= *r.HighNormal {
			return fmt.Errorf("critical_high (%v) must be greater than high_normal (%v)", *r.CriticalHigh, *r.HighNormal)
		}
	}

	// Panic must be outside critical
	if r.PanicLow != nil && r.CriticalLow != nil {
		if *r.PanicLow >= *r.CriticalLow {
			return fmt.Errorf("panic_low (%v) must be less than critical_low (%v)", *r.PanicLow, *r.CriticalLow)
		}
	}
	if r.PanicHigh != nil && r.CriticalHigh != nil {
		if *r.PanicHigh <= *r.CriticalHigh {
			return fmt.Errorf("panic_high (%v) must be greater than critical_high (%v)", *r.PanicHigh, *r.CriticalHigh)
		}
	}

	// Trimester must be 1, 2, or 3
	if r.Trimester != nil && (*r.Trimester < 1 || *r.Trimester > 3) {
		return fmt.Errorf("trimester must be 1, 2, or 3, got %d", *r.Trimester)
	}

	// CKD stage must be 1-5
	if r.CKDStage != nil && (*r.CKDStage < 1 || *r.CKDStage > 5) {
		return fmt.Errorf("ckd_stage must be 1-5, got %d", *r.CKDStage)
	}

	return nil
}
