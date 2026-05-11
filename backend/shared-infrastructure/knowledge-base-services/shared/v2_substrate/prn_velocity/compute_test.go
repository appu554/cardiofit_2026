package prn_velocity

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	testResident = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherResident = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testNow       = time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
)

// admins builds a slice of Administration events at the given offsets
// (in days) before `testNow`, all for testResident + PRNBenzodiazepine.
func admins(offsetsDays ...float64) []Administration {
	out := make([]Administration, 0, len(offsetsDays))
	for _, d := range offsetsDays {
		out = append(out, Administration{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(time.Duration(-d * float64(24*time.Hour))),
		})
	}
	return out
}

func TestCompute_AllSeverityBuckets(t *testing.T) {
	// Baseline window holds 6 administrations over 90 days → avg = 2.0/30d.
	// Place 6 admins evenly spread across (now-120d, now-30d].
	baseline := admins(35, 50, 65, 80, 95, 110) // 6 admins, all in baseline window

	cases := []struct {
		name         string
		recentCount  int
		wantSeverity int
		// wantRatio is the expected ratio given baseline avg = 2.0
		wantRatio float64
	}{
		// recent=2 → ratio=1.0 → bucket 1 (NOT > 1.0)
		{"severity_1_no_increase", 2, 1, 1.0},
		// recent=3 → ratio=1.5 → bucket 2 (>1.0, not >1.5)
		{"severity_2_any_increase", 3, 2, 1.5},
		// recent=4 → ratio=2.0 → bucket 3 (>1.5, not >2.5)
		{"severity_3_150pct", 4, 3, 2.0},
		// recent=6 → ratio=3.0 → bucket 4 (>2.5, not >4.0)
		{"severity_4_250pct", 6, 4, 3.0},
		// recent=10 → ratio=5.0 → bucket 5 (>4.0)
		{"severity_5_400pct", 10, 5, 5.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Place recent admins evenly inside (now-30d, now].
			recent := make([]Administration, 0, tc.recentCount)
			for i := 0; i < tc.recentCount; i++ {
				// Spread admins between 1d and 29d before now.
				offset := 1.0 + float64(i)*(28.0/float64(maxInt(tc.recentCount-1, 1)))
				if tc.recentCount == 1 {
					offset = 15.0
				}
				recent = append(recent, Administration{
					ResidentID:     testResident,
					Class:          PRNBenzodiazepine,
					AdministeredAt: testNow.Add(time.Duration(-offset * float64(24*time.Hour))),
				})
			}

			all := append([]Administration{}, baseline...)
			all = append(all, recent...)

			r := Compute(all, testResident, PRNBenzodiazepine, testNow)

			if r.Recent30dCount != tc.recentCount {
				t.Errorf("Recent30dCount = %d; want %d", r.Recent30dCount, tc.recentCount)
			}
			if r.Baseline90dAvg != 2.0 {
				t.Errorf("Baseline90dAvg = %v; want 2.0", r.Baseline90dAvg)
			}
			if r.VelocityRatio != tc.wantRatio {
				t.Errorf("VelocityRatio = %v; want %v", r.VelocityRatio, tc.wantRatio)
			}
			if r.Severity != tc.wantSeverity {
				t.Errorf("Severity = %d; want %d (ratio=%v)", r.Severity, tc.wantSeverity, r.VelocityRatio)
			}
		})
	}
}

func TestCompute_NoBaseline_RecentPresent_CriticalEmergent(t *testing.T) {
	// No baseline admins; 3 recent. Ratio must be +Inf, severity 5.
	in := admins(2, 10, 25)
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)

	if r.Recent30dCount != 3 {
		t.Errorf("Recent30dCount = %d; want 3", r.Recent30dCount)
	}
	if r.Baseline90dAvg != 0 {
		t.Errorf("Baseline90dAvg = %v; want 0", r.Baseline90dAvg)
	}
	if !math.IsInf(r.VelocityRatio, +1) {
		t.Errorf("VelocityRatio = %v; want +Inf", r.VelocityRatio)
	}
	if r.Severity != 5 {
		t.Errorf("Severity = %d; want 5", r.Severity)
	}
}

func TestCompute_NoBaseline_NoRecent_Quiescent(t *testing.T) {
	r := Compute(nil, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 0 || r.Baseline90dAvg != 0 {
		t.Errorf("counts not zero: %+v", r)
	}
	if r.VelocityRatio != 0 {
		t.Errorf("VelocityRatio = %v; want 0", r.VelocityRatio)
	}
	if r.Severity != 1 {
		t.Errorf("Severity = %d; want 1", r.Severity)
	}
}

func TestCompute_BaselinePresent_NoRecent(t *testing.T) {
	in := admins(45, 60, 90) // all in baseline
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)

	if r.Recent30dCount != 0 {
		t.Errorf("Recent30dCount = %d; want 0", r.Recent30dCount)
	}
	if r.Baseline90dAvg != 1.0 {
		t.Errorf("Baseline90dAvg = %v; want 1.0", r.Baseline90dAvg)
	}
	if r.VelocityRatio != 0 {
		t.Errorf("VelocityRatio = %v; want 0", r.VelocityRatio)
	}
	if r.Severity != 1 {
		t.Errorf("Severity = %d; want 1", r.Severity)
	}
}

func TestCompute_BoundaryRatio_Exactly_4_0_Falls_To_Severity_4(t *testing.T) {
	// Baseline avg = 2.0 (6 admins / 3). Recent = 8 → ratio = 4.0 → severity 4
	// per `> 4.0` semantics in the CQL.
	baseline := admins(35, 50, 65, 80, 95, 110)
	recent := admins(1, 3, 6, 10, 14, 18, 22, 26)
	all := append([]Administration{}, baseline...)
	all = append(all, recent...)

	r := Compute(all, testResident, PRNBenzodiazepine, testNow)
	if r.VelocityRatio != 4.0 {
		t.Fatalf("setup failure: VelocityRatio = %v; want exactly 4.0", r.VelocityRatio)
	}
	if r.Severity != 4 {
		t.Errorf("Severity for ratio=4.0 = %d; want 4 (CQL uses strict `>`)", r.Severity)
	}
}

func TestCompute_BoundaryRatio_Exactly_1_0_Falls_To_Severity_1(t *testing.T) {
	// Baseline avg = 2.0; recent = 2 → ratio = 1.0 → severity 1
	// (NOT severity 2 — per `> 1.0`).
	baseline := admins(35, 50, 65, 80, 95, 110)
	recent := admins(5, 20)
	all := append([]Administration{}, baseline...)
	all = append(all, recent...)

	r := Compute(all, testResident, PRNBenzodiazepine, testNow)
	if r.VelocityRatio != 1.0 {
		t.Fatalf("setup failure: VelocityRatio = %v; want exactly 1.0", r.VelocityRatio)
	}
	if r.Severity != 1 {
		t.Errorf("Severity for ratio=1.0 = %d; want 1", r.Severity)
	}
}

func TestCompute_BoundaryAdmin_Exactly_30d_Falls_In_Baseline(t *testing.T) {
	// An administration exactly at now-30d falls in the baseline window
	// per half-open semantics: recent = (now-30d, now], baseline = (now-120d, now-30d].
	in := []Administration{
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-30 * 24 * time.Hour),
		},
	}
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 0 {
		t.Errorf("admin at exactly now-30d: Recent30dCount = %d; want 0 (falls in baseline)", r.Recent30dCount)
	}
	if r.Baseline90dAvg == 0 {
		t.Errorf("admin at exactly now-30d should be counted in baseline; avg = %v", r.Baseline90dAvg)
	}
	// 1 admin in baseline window → avg = 1/3
	wantAvg := 1.0 / 3.0
	if math.Abs(r.Baseline90dAvg-wantAvg) > 1e-12 {
		t.Errorf("Baseline90dAvg = %v; want %v", r.Baseline90dAvg, wantAvg)
	}
}

func TestCompute_BoundaryAdmin_Exactly_Now_Falls_In_Recent(t *testing.T) {
	in := []Administration{
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow, // exactly `now`
		},
	}
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 1 {
		t.Errorf("admin at exactly now: Recent30dCount = %d; want 1 (recent window is right-closed)", r.Recent30dCount)
	}
}

func TestCompute_OutOfWindow_Ignored(t *testing.T) {
	in := []Administration{
		// Older than 120d — outside baseline.
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-150 * 24 * time.Hour),
		},
		// Exactly at now-120d — outside baseline (lower bound exclusive).
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-120 * 24 * time.Hour),
		},
		// Future timestamp — outside recent.
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(1 * time.Hour),
		},
	}
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 0 {
		t.Errorf("Recent30dCount = %d; want 0", r.Recent30dCount)
	}
	if r.Baseline90dAvg != 0 {
		t.Errorf("Baseline90dAvg = %v; want 0", r.Baseline90dAvg)
	}
	if r.Severity != 1 {
		t.Errorf("Severity = %d; want 1", r.Severity)
	}
}

func TestCompute_WrongClass_Ignored(t *testing.T) {
	in := []Administration{
		// Antipsychotic — should be ignored when querying benzodiazepine.
		{
			ResidentID:     testResident,
			Class:          PRNAntipsychotic,
			AdministeredAt: testNow.Add(-5 * 24 * time.Hour),
		},
		// Benzo in recent window — should be counted.
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-10 * 24 * time.Hour),
		},
	}
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 1 {
		t.Errorf("Recent30dCount = %d; want 1 (wrong-class admin must be ignored)", r.Recent30dCount)
	}
}

func TestCompute_WrongResident_Ignored(t *testing.T) {
	in := []Administration{
		{
			ResidentID:     otherResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-5 * 24 * time.Hour),
		},
		{
			ResidentID:     testResident,
			Class:          PRNBenzodiazepine,
			AdministeredAt: testNow.Add(-10 * 24 * time.Hour),
		},
	}
	r := Compute(in, testResident, PRNBenzodiazepine, testNow)
	if r.Recent30dCount != 1 {
		t.Errorf("Recent30dCount = %d; want 1 (other-resident admin must be ignored)", r.Recent30dCount)
	}
}

func TestCompute_ResultFieldsPopulated(t *testing.T) {
	r := Compute(nil, testResident, PRNAnalgesic, testNow)
	if r.ResidentID != testResident {
		t.Errorf("ResidentID not propagated: %v", r.ResidentID)
	}
	if r.Class != PRNAnalgesic {
		t.Errorf("Class not propagated: %v", r.Class)
	}
	if !r.EvaluatedAt.Equal(testNow) {
		t.Errorf("EvaluatedAt not propagated: %v", r.EvaluatedAt)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
