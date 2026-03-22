package abdm

import (
	"context"
	"time"

	"github.com/cardiofit/ingestion-service/internal/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresConsentStore persists and retrieves ABDM consent artifacts
// in PostgreSQL for consent verification during data exchange.
type PostgresConsentStore struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresConsentStore creates a consent store backed by the given
// PostgreSQL connection pool.
func NewPostgresConsentStore(pool *pgxpool.Pool, logger *zap.Logger) *PostgresConsentStore {
	return &PostgresConsentStore{
		pool:   pool,
		logger: logger,
	}
}

// GetConsent retrieves a consent artifact by its ABDM consent ID.
func (s *PostgresConsentStore) GetConsent(consentID string) (*crypto.ConsentArtifact, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var a crypto.ConsentArtifact
	err := s.pool.QueryRow(ctx, `
		SELECT consent_id, patient_id, hiu_request_id, purpose,
		       hi_types, date_from, date_to, expires_at, signature, status
		  FROM abdm_consent_artifacts
		 WHERE consent_id = $1
	`, consentID).Scan(
		&a.ConsentID,
		&a.PatientID,
		&a.HIURequestID,
		&a.Purpose,
		&a.HITypes,
		&a.DateFrom,
		&a.DateTo,
		&a.ExpiresAt,
		&a.Signature,
		&a.Status,
	)
	if err != nil {
		s.logger.Error("consent store: query failed",
			zap.String("consent_id", consentID),
			zap.Error(err),
		)
		return nil, err
	}

	return &a, nil
}

// StoreConsent upserts a consent artifact. On conflict (duplicate
// consent_id) the status and expiry are updated to reflect the latest
// state from the ABDM consent manager.
func (s *PostgresConsentStore) StoreConsent(artifact crypto.ConsentArtifact) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO abdm_consent_artifacts
			(consent_id, patient_id, hiu_request_id, purpose,
			 hi_types, date_from, date_to, expires_at, signature, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (consent_id) DO UPDATE SET
			status     = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			signature  = EXCLUDED.signature
	`,
		artifact.ConsentID,
		artifact.PatientID,
		artifact.HIURequestID,
		artifact.Purpose,
		artifact.HITypes,
		artifact.DateFrom,
		artifact.DateTo,
		artifact.ExpiresAt,
		artifact.Signature,
		artifact.Status,
	)
	if err != nil {
		s.logger.Error("consent store: upsert failed",
			zap.String("consent_id", artifact.ConsentID),
			zap.Error(err),
		)
		return err
	}

	return nil
}
