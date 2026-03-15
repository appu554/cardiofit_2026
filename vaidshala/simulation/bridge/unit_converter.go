package bridge

const (
	GlucoseMgDLToMmolL    = 18.0
	CreatinineMgDLToUmolL = 88.4
)

func GlucoseToProduction(simValue float64) float64  { return simValue }
func GlucoseToSimulation(prodValue float64) float64  { return prodValue }
func CreatinineToProduction(simValue float64) float64 { return simValue }
func CreatinineToSimulation(prodValue float64) float64 { return prodValue }
