// Package store provides data storage and retrieval for lab results
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/types"
)

// ResultStore manages lab result storage and retrieval
type ResultStore struct {
	db    *gorm.DB
	cache *redis.Client
	log   *logrus.Entry
}

// NewResultStore creates a new result store
func NewResultStore(db *gorm.DB, cache *redis.Client, log *logrus.Entry) *ResultStore {
	return &ResultStore{
		db:    db,
		cache: cache,
		log:   log.WithField("component", "result_store"),
	}
}

// Store saves a new lab result
func (s *ResultStore) Store(ctx context.Context, req *types.StoreResultRequest) (*types.LabResult, error) {
	reportedAt := time.Now()
	if req.ReportedAt != nil {
		reportedAt = *req.ReportedAt
	}

	result := &types.LabResult{
		ID:             uuid.New(),
		PatientID:      req.PatientID,
		Code:           req.Code,
		Name:           req.Name,
		ValueNumeric:   req.ValueNumeric,
		ValueString:    req.ValueString,
		Unit:           req.Unit,
		ReferenceRange: req.ReferenceRange,
		CollectedAt:    req.CollectedAt,
		ReportedAt:     reportedAt,
		Status:         req.Status,
		Performer:      req.Performer,
		EncounterID:    req.EncounterID,
		SpecimenID:     req.SpecimenID,
		OrderID:        req.OrderID,
	}

	if result.Status == "" {
		result.Status = types.ResultStatusFinal
	}

	if err := s.db.WithContext(ctx).Create(result).Error; err != nil {
		return nil, fmt.Errorf("failed to store result: %w", err)
	}

	// Invalidate cache for this patient
	s.invalidatePatientCache(ctx, req.PatientID)

	s.log.WithFields(logrus.Fields{
		"result_id":  result.ID,
		"patient_id": result.PatientID,
		"code":       result.Code,
	}).Debug("Stored lab result")

	return result, nil
}

// StoreBatch saves multiple lab results
func (s *ResultStore) StoreBatch(ctx context.Context, requests []types.StoreResultRequest) ([]types.LabResult, error) {
	results := make([]types.LabResult, len(requests))

	for i, req := range requests {
		reportedAt := time.Now()
		if req.ReportedAt != nil {
			reportedAt = *req.ReportedAt
		}

		results[i] = types.LabResult{
			ID:             uuid.New(),
			PatientID:      req.PatientID,
			Code:           req.Code,
			Name:           req.Name,
			ValueNumeric:   req.ValueNumeric,
			ValueString:    req.ValueString,
			Unit:           req.Unit,
			ReferenceRange: req.ReferenceRange,
			CollectedAt:    req.CollectedAt,
			ReportedAt:     reportedAt,
			Status:         req.Status,
			Performer:      req.Performer,
			EncounterID:    req.EncounterID,
			SpecimenID:     req.SpecimenID,
			OrderID:        req.OrderID,
		}

		if results[i].Status == "" {
			results[i].Status = types.ResultStatusFinal
		}
	}

	if err := s.db.WithContext(ctx).CreateInBatches(results, 100).Error; err != nil {
		return nil, fmt.Errorf("failed to store batch: %w", err)
	}

	// Invalidate caches
	patientIDs := make(map[string]bool)
	for _, r := range results {
		patientIDs[r.PatientID] = true
	}
	for patientID := range patientIDs {
		s.invalidatePatientCache(ctx, patientID)
	}

	return results, nil
}

// GetByID retrieves a result by ID
func (s *ResultStore) GetByID(ctx context.Context, id string) (*types.LabResult, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("result:%s", id)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
			var result types.LabResult
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return &result, nil
			}
		}
	}

	var result types.LabResult
	if err := s.db.WithContext(ctx).First(&result, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("result not found: %w", err)
	}

	// Cache the result
	if s.cache != nil {
		if data, err := json.Marshal(result); err == nil {
			s.cache.Set(ctx, cacheKey, data, 30*time.Minute)
		}
	}

	return &result, nil
}

// GetByPatient retrieves results for a patient with pagination
func (s *ResultStore) GetByPatient(ctx context.Context, patientID string, limit, offset int) ([]types.LabResult, int, error) {
	var results []types.LabResult
	var total int64

	query := s.db.WithContext(ctx).Model(&types.LabResult{}).Where("patient_id = ?", patientID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("collected_at DESC").Limit(limit).Offset(offset).Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, int(total), nil
}

// GetByPatientAndCode retrieves results for a specific test
func (s *ResultStore) GetByPatientAndCode(ctx context.Context, patientID, code string, days int) ([]types.LabResult, error) {
	since := time.Now().AddDate(0, 0, -days)

	var results []types.LabResult
	err := s.db.WithContext(ctx).
		Where("patient_id = ? AND code = ? AND collected_at >= ?", patientID, code, since).
		Order("collected_at DESC").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetRecentByPatient retrieves recent results for a patient
func (s *ResultStore) GetRecentByPatient(ctx context.Context, patientID string, days int) ([]types.LabResult, error) {
	since := time.Now().AddDate(0, 0, -days)

	var results []types.LabResult
	err := s.db.WithContext(ctx).
		Where("patient_id = ? AND collected_at >= ?", patientID, since).
		Order("collected_at DESC").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetPreviousResult retrieves the most recent previous result for delta checking
func (s *ResultStore) GetPreviousResult(ctx context.Context, patientID, code string, beforeTime time.Time) (*types.LabResult, error) {
	var result types.LabResult
	err := s.db.WithContext(ctx).
		Where("patient_id = ? AND code = ? AND collected_at < ?", patientID, code, beforeTime).
		Order("collected_at DESC").
		First(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetDistinctCodes retrieves all unique test codes for a patient
func (s *ResultStore) GetDistinctCodes(ctx context.Context, patientID string) ([]string, error) {
	var codes []string
	err := s.db.WithContext(ctx).
		Model(&types.LabResult{}).
		Where("patient_id = ?", patientID).
		Distinct("code").
		Pluck("code", &codes).Error

	if err != nil {
		return nil, err
	}

	return codes, nil
}

// GetCriticalResults retrieves results with critical/panic flags
func (s *ResultStore) GetCriticalResults(ctx context.Context, since time.Time) ([]types.LabResult, error) {
	var results []types.LabResult

	err := s.db.WithContext(ctx).
		Joins("JOIN interpretations ON lab_results.id = interpretations.result_id").
		Where("(interpretations.is_critical = true OR interpretations.is_panic = true) AND lab_results.created_at >= ?", since).
		Order("lab_results.created_at DESC").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// invalidatePatientCache invalidates cached data for a patient
func (s *ResultStore) invalidatePatientCache(ctx context.Context, patientID string) {
	if s.cache == nil {
		return
	}

	pattern := fmt.Sprintf("patient:%s:*", patientID)
	iter := s.cache.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		s.cache.Del(ctx, iter.Val())
	}
}
