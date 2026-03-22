package coding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type LabCodeRegistry struct {
	db     *pgxpool.Pool
	cache  map[string]codeMapping
	mu     sync.RWMutex
	logger *zap.Logger
}

type codeMapping struct {
	LOINCCode   string
	DisplayName string
	Unit        string
	CachedAt    time.Time
}

func NewLabCodeRegistry(db *pgxpool.Pool, logger *zap.Logger) *LabCodeRegistry {
	r := &LabCodeRegistry{
		db:     db,
		cache:  make(map[string]codeMapping),
		logger: logger,
	}
	return r
}

func (r *LabCodeRegistry) LookupLOINC(labID, labCode string) (loincCode, displayName, unit string, err error) {
	key := labID + ":" + labCode

	r.mu.RLock()
	if m, ok := r.cache[key]; ok && time.Since(m.CachedAt) < 5*time.Minute {
		r.mu.RUnlock()
		return m.LOINCCode, m.DisplayName, m.Unit, nil
	}
	r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = r.db.QueryRow(ctx,
		`SELECT loinc_code, COALESCE(display_name, ''), COALESCE(unit, '')
		 FROM lab_code_mappings
		 WHERE lab_id = $1 AND lab_code = $2`,
		labID, labCode,
	).Scan(&loincCode, &displayName, &unit)

	if err != nil {
		r.logger.Debug("lab code not found in registry",
			zap.String("lab_id", labID),
			zap.String("lab_code", labCode),
		)
		return "", "", "", fmt.Errorf("no LOINC mapping for %s:%s", labID, labCode)
	}

	r.mu.Lock()
	r.cache[key] = codeMapping{
		LOINCCode:   loincCode,
		DisplayName: displayName,
		Unit:        unit,
		CachedAt:    time.Now(),
	}
	r.mu.Unlock()

	return loincCode, displayName, unit, nil
}

func (r *LabCodeRegistry) Preload(labID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := r.db.Query(ctx,
		`SELECT lab_code, loinc_code, COALESCE(display_name, ''), COALESCE(unit, '')
		 FROM lab_code_mappings WHERE lab_id = $1`,
		labID,
	)
	if err != nil {
		return fmt.Errorf("preload %s mappings: %w", labID, err)
	}
	defer rows.Close()

	count := 0
	r.mu.Lock()
	defer r.mu.Unlock()

	for rows.Next() {
		var labCode, loincCode, displayName, unit string
		if err := rows.Scan(&labCode, &loincCode, &displayName, &unit); err != nil {
			continue
		}
		r.cache[labID+":"+labCode] = codeMapping{
			LOINCCode:   loincCode,
			DisplayName: displayName,
			Unit:        unit,
			CachedAt:    time.Now(),
		}
		count++
	}

	r.logger.Info("preloaded lab code mappings",
		zap.String("lab_id", labID),
		zap.Int("count", count),
	)
	return nil
}

func (r *LabCodeRegistry) UpsertMapping(labID, labCode, loincCode, displayName, unit string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.Exec(ctx,
		`INSERT INTO lab_code_mappings (lab_id, lab_code, loinc_code, display_name, unit)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (lab_id, lab_code)
		 DO UPDATE SET loinc_code = $3, display_name = $4, unit = $5`,
		labID, labCode, loincCode, displayName, unit,
	)
	if err != nil {
		return fmt.Errorf("upsert mapping: %w", err)
	}

	r.mu.Lock()
	delete(r.cache, labID+":"+labCode)
	r.mu.Unlock()

	return nil
}

func (r *LabCodeRegistry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return map[string]interface{}{
		"cache_size": len(r.cache),
	}
}
