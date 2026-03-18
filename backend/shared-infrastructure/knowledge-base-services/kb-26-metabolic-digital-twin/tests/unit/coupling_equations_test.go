package unit

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

func TestCouplingStep_ISIncreaseReducesVF(t *testing.T) {
	state := models.SimState{IS: 0.5, VF: 0.6, HGO: 0.5, MM: 0.5, VR: 0.5, RR: 0.8}
	intervention := models.Intervention{ISEffect: 0.01}

	next := services.CouplingStep(state, intervention)
	if next.VF >= state.VF {
		t.Errorf("IS↑ should reduce VF: before=%f after=%f", state.VF, next.VF)
	}
}

func TestCouplingStep_VFIncreaseReducesIS(t *testing.T) {
	state := models.SimState{IS: 0.5, VF: 0.6, HGO: 0.5, MM: 0.5, VR: 0.5, RR: 0.8}
	intervention := models.Intervention{VFEffect: 0.01}

	next := services.CouplingStep(state, intervention)
	if next.IS >= state.IS {
		t.Errorf("VF↑ should reduce IS: before=%f after=%f", state.IS, next.IS)
	}
}

func TestCouplingStep_Clamped(t *testing.T) {
	state := models.SimState{IS: 0.99, VF: 0.01, HGO: 0.01, MM: 0.99, VR: 0.01, RR: 0.99}
	intervention := models.Intervention{ISEffect: 0.1, MMEffect: 0.1}

	next := services.CouplingStep(state, intervention)
	if next.IS > 1.0 || next.VF < 0.0 || next.MM > 1.0 {
		t.Errorf("values should be clamped to [0,1]: IS=%f VF=%f MM=%f", next.IS, next.VF, next.MM)
	}
}
