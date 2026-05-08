// Package portability implements cross-employer data portability transitions
// and account closure per Self-Visibility Guidelines Part 10.
//
// VisibilityClass: pharmacist-controlled — all data movements are initiated
// by and preserve rights for the pharmacist as the data subject.
package portability

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TransitionPlan records the outcome of a cross-employer portability transition.
// Per Guidelines §10: POA, PDP, and own PFA data follow the pharmacist.
// Active recommendations remain bound to the prior employer's deployment.
// VisibilityClass: pharmacist-controlled.
type TransitionPlan struct {
	ID                             uuid.UUID
	PharmacistID                   uuid.UUID
	PriorEmployerID                uuid.UUID
	NewEmployerID                  *uuid.UUID
	PreservesReflectiveEntries     bool
	PreservesPortfolio             bool
	PreservesOwnPFA                bool
	PreservesActiveRecommendations bool
	RevertsToFreeTier              bool
	InitiatedAt                    time.Time
}

// Carrier moves pharmacist-controlled data between employer contexts.
// Each method is idempotent; newEmployerID == nil signals free-tier reversion.
type Carrier interface {
	MovePOA(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
	MovePortfolio(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
	MoveOwnPFA(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
}

// Handler orchestrates portability transitions.
type Handler struct{ carrier Carrier }

// NewHandler returns a Handler backed by the given Carrier.
func NewHandler(c Carrier) *Handler { return &Handler{carrier: c} }

// Initiate performs a cross-employer transition for the given pharmacist.
// When newEmployerID is nil the account reverts to the free tier, preserving
// portfolio, CPD record, and RPL pack capability per Guidelines §10.3.
func (h *Handler) Initiate(ctx context.Context, pharmacistID, priorEmployerID uuid.UUID, newEmployerID *uuid.UUID) (TransitionPlan, error) {
	if err := ctx.Err(); err != nil {
		return TransitionPlan{}, err
	}
	if err := h.carrier.MovePOA(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	if err := h.carrier.MovePortfolio(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	if err := h.carrier.MoveOwnPFA(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	return TransitionPlan{
		ID:                             uuid.New(),
		PharmacistID:                   pharmacistID,
		PriorEmployerID:                priorEmployerID,
		NewEmployerID:                  newEmployerID,
		PreservesReflectiveEntries:     true,
		PreservesPortfolio:             true,
		PreservesOwnPFA:                true,
		PreservesActiveRecommendations: false, // stays with prior employer's deployment
		RevertsToFreeTier:              newEmployerID == nil,
		InitiatedAt:                    time.Now().UTC(),
	}, nil
}
