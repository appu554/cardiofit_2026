package services

import "kb-26-metabolic-digital-twin/internal/models"

// Coupling coefficients (spec §6.3 — tuned from physiological literature)
const (
	kIS_VF  = -0.15 // VF↑ → IS↓
	kIS_MM  = 0.10  // MM↑ → IS↑
	kVF_IS  = -0.08 // IS↑ → insulin demand↓ → VF↓
	kVF_MM  = -0.05 // MM↑ → BMR↑ → VF↓
	kHGO_VF = 0.12  // VF↑ → HGO↑
	kHGO_IS = -0.10 // IS↑ → hepatic sensitivity↑ → HGO↓
	kMM_IS  = 0.05  // IS↑ → anabolic environment → MM↑
	kVR_VF  = 0.08  // VF↑ → inflammation → VR↑
	kVR_IS  = -0.06 // IS↑ → endothelial function → VR↓
	kRR_VF  = -0.03 // VF↑ → nephron damage → RR↓
)

// CouplingStep advances the simulation by one day using coupled equations.
func CouplingStep(state models.SimState, intervention models.Intervention) models.SimState {
	next := state

	dIS := intervention.ISEffect + kIS_VF*state.VF + kIS_MM*state.MM
	next.IS = clamp(state.IS+dIS, 0, 1)

	dVF := intervention.VFEffect + kVF_IS*state.IS + kVF_MM*state.MM
	next.VF = clamp(state.VF+dVF, 0, 1)

	dHGO := intervention.HGOEffect + kHGO_VF*state.VF + kHGO_IS*state.IS
	next.HGO = clamp(state.HGO+dHGO, 0, 1)

	dMM := intervention.MMEffect + kMM_IS*state.IS
	next.MM = clamp(state.MM+dMM, 0, 1)

	dVR := intervention.VREffect + kVR_VF*state.VF + kVR_IS*state.IS
	next.VR = clamp(state.VR+dVR, 0, 1)

	dRR := intervention.RREffect + kRR_VF*state.VF
	next.RR = clamp(state.RR+dRR, 0, 1)

	return next
}
