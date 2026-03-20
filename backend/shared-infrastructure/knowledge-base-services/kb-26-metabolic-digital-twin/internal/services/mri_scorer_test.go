package services

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestComputeGlucoseDomain_Optimal(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)
	domain := scorer.ComputeGlucoseDomain(85, 115, 0)
	// All signals below mean → domain score should be negative
	if domain.Score > 0 {
		t.Errorf("optimal glucose domain should be negative, got %f", domain.Score)
	}
}

func TestComputeGlucoseDomain_HighRisk(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)
	domain := scorer.ComputeGlucoseDomain(160, 220, 0.6)
	// All signals high → domain score should be strongly positive
	if domain.Score < 1.5 {
		t.Errorf("high-risk glucose domain should be > 1.5, got %f", domain.Score)
	}
}

func TestComputeBodyCompDomain(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)
	domain := scorer.ComputeBodyCompDomain(100, 1.5, 0.3, "M", 25.0)
	// Waist 100cm male (above threshold) + weight gaining + low muscle, normal BMI
	if domain.Score < 0.5 {
		t.Errorf("elevated body comp domain should be > 0.5, got %f", domain.Score)
	}
}

func TestComputeCardioDomain(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)
	domain := scorer.ComputeCardioDomain(145, 10, "REVERSE_DIPPER")
	// High SBP + rising trend + reverse dipper
	if domain.Score < 1.5 {
		t.Errorf("high-risk cardio domain should be > 1.5, got %f", domain.Score)
	}
}

func TestComputeBehavioralDomain_Good(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)
	domain := scorer.ComputeBehavioralDomain(8500, 1.1, 0.9)
	// Good steps + good protein + good sleep → negative score
	if domain.Score > 0 {
		t.Errorf("good behavioral domain should be negative, got %f", domain.Score)
	}
}

func TestScaleToRange_Midpoint(t *testing.T) {
	// Raw z=0 → should map to ~50 (midpoint of 0-100)
	scaled := ScaleToRange(0)
	if math.Abs(scaled-50) > 1 {
		t.Errorf("z=0 should scale to ~50, got %f", scaled)
	}
}

func TestScaleToRange_Extremes(t *testing.T) {
	// Very negative z → near 0
	low := ScaleToRange(-4)
	if low > 5 {
		t.Errorf("z=-4 should scale near 0, got %f", low)
	}
	// Very positive z → near 100
	high := ScaleToRange(4)
	if high < 95 {
		t.Errorf("z=4 should scale near 100, got %f", high)
	}
}

func TestCategorizeMRI(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{10, models.MRICategoryOptimal},
		{25, models.MRICategoryOptimal},
		{26, models.MRICategoryMildDysregulation},
		{50, models.MRICategoryMildDysregulation},
		{51, models.MRICategoryModerateDeterioration},
		{75, models.MRICategoryModerateDeterioration},
		{76, models.MRICategoryHighDeterioration},
		{100, models.MRICategoryHighDeterioration},
	}
	for _, tc := range tests {
		cat := CategorizeMRI(tc.score)
		if cat != tc.expected {
			t.Errorf("CategorizeMRI(%f) = %s, want %s", tc.score, cat, tc.expected)
		}
	}
}

func TestComputeTrend_Improving(t *testing.T) {
	// Decreasing scores = IMPROVING
	history := []float64{60, 55, 50, 45}
	trend := ComputeMRITrend(42, history)
	if trend != "IMPROVING" {
		t.Errorf("decreasing MRI scores should be IMPROVING, got %s", trend)
	}
}

func TestComputeTrend_Worsening(t *testing.T) {
	history := []float64{40, 45, 50, 55}
	trend := ComputeMRITrend(60, history)
	if trend != "WORSENING" {
		t.Errorf("increasing MRI scores should be WORSENING, got %s", trend)
	}
}

func TestComputeTrend_Stable(t *testing.T) {
	history := []float64{50, 51, 50, 49}
	trend := ComputeMRITrend(50, history)
	if trend != "STABLE" {
		t.Errorf("flat MRI scores should be STABLE, got %s", trend)
	}
}

func TestComputeTrend_NoHistory(t *testing.T) {
	trend := ComputeMRITrend(50, nil)
	if trend != "STABLE" {
		t.Errorf("no history should default to STABLE, got %s", trend)
	}
}

func TestMRIScorer_GetHistoryScores_NoData(t *testing.T) {
	// Without a DB, GetHistoryScores should return empty slice, no error
	scorer := NewMRIScorer(nil, nil)
	scores := scorer.GetHistoryScores(uuid.New())
	if len(scores) != 0 {
		t.Errorf("expected empty history without DB, got %d", len(scores))
	}
}

// Integration test: the "borderline everything" patient from spec §5.1
func TestBorderlineEverythingPatient(t *testing.T) {
	scorer := NewMRIScorer(nil, nil)

	input := MRIScorerInput{
		FBG:         118,
		PPBG:        172,
		HbA1cTrend:  0.1,
		WaistCm:     92,
		WeightTrend: 0.3,
		MuscleSTS:   11,
		SBP:         138,
		SBPTrend:    3,
		BPDipping:   "NON_DIPPER",
		Steps:       3800,
		ProteinGKg:  0.7,
		SleepScore:  0.5,
		Sex:         "M",
	}

	result := scorer.ComputeMRI(input, nil)

	// Spec §5.1: this patient should be MODERATE_DETERIORATION (MRI ~58-71 with Indian population params)
	if result.Category != models.MRICategoryModerateDeterioration {
		t.Errorf("borderline patient should be MODERATE_DETERIORATION, got %s (score=%f)", result.Category, result.Score)
	}
	if result.Score < 45 || result.Score > 75 {
		t.Errorf("borderline patient MRI should be 45-75, got %f", result.Score)
	}
}

func TestMRIScorer_PersistScore_UpdatesNadir(t *testing.T) {
	db := setupTestDB(t)
	// mri_scores table needs to be created for PersistScore
	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE IF NOT EXISTS mri_scores (
		id TEXT PRIMARY KEY,
		patient_id TEXT NOT NULL,
		score REAL NOT NULL,
		category TEXT NOT NULL,
		trend TEXT,
		top_driver TEXT,
		glucose_domain REAL,
		body_comp_domain REAL,
		cardio_domain REAL,
		behavioral_domain REAL,
		signal_z_scores TEXT,
		twin_state_id TEXT,
		computed_at DATETIME NOT NULL
	)`)

	det := NewRelapseDetector(db, zap.NewNop())
	scorer := NewMRIScorer(db, zap.NewNop())
	scorer.SetRelapseDetector(det)

	pid := uuid.New()

	// First score
	result1 := models.MRIResult{
		Score: 55.0, Category: "MODERATE_DETERIORATION", Trend: "STABLE", TopDriver: "Glucose Control",
		Domains: []models.DomainScore{
			{Name: "Glucose Control", Scaled: 55, Signals: map[string]float64{"FBG": 0.5}},
			{Name: "Body Composition", Scaled: 50, Signals: map[string]float64{"waist": 0.3}},
			{Name: "Cardiovascular Regulation", Scaled: 50, Signals: map[string]float64{"SBP": 0.4}},
			{Name: "Behavioral Metabolism", Scaled: 50, Signals: map[string]float64{"activity": 0.2}},
		},
	}
	if _, err := scorer.PersistScore(pid, result1, nil); err != nil {
		t.Fatalf("PersistScore(1): %v", err)
	}

	// Second score (lower — should become the nadir)
	result2 := result1
	result2.Score = 42.0
	if _, err := scorer.PersistScore(pid, result2, nil); err != nil {
		t.Fatalf("PersistScore(2): %v", err)
	}

	nadir, err := det.GetNadir(pid)
	if err != nil {
		t.Fatalf("GetNadir: %v", err)
	}
	if nadir.NadirScore != 42.0 {
		t.Errorf("nadir = %.1f, want 42.0", nadir.NadirScore)
	}
}
