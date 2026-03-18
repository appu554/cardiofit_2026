package unit

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/services"
)

func TestNormalPosterior_SingleObservation(t *testing.T) {
	post := services.NormalPosterior(-12, 3, -10, 2)
	if post.Mean >= -10 || post.Mean <= -12 {
		t.Errorf("posterior mean should be between -12 and -10, got %f", post.Mean)
	}
	if post.SD >= 3 || post.SD >= 2 {
		t.Errorf("posterior SD should be < min(3,2), got %f", post.SD)
	}
}

func TestNormalPosterior_ManyObservations(t *testing.T) {
	mean := -12.0
	sd := 3.0
	for i := 0; i < 5; i++ {
		post := services.NormalPosterior(mean, sd, -8, 2)
		mean = post.Mean
		sd = post.SD
	}
	if math.Abs(mean-(-8)) > 1.5 {
		t.Errorf("after 5 observations of -8, mean should approach -8, got %f", mean)
	}
}

func TestNormalPosterior_ZeroSD(t *testing.T) {
	post := services.NormalPosterior(-12, 0, -10, 2)
	if post.Mean != -10 {
		t.Errorf("with zero priorSD, mean should equal obs, got %f", post.Mean)
	}
	if post.SD != 1.0 {
		t.Errorf("with zero priorSD, SD should be 1.0, got %f", post.SD)
	}
}

func TestCalibrationConfidence(t *testing.T) {
	c0 := services.CalibrationConfidence(0)
	c1 := services.CalibrationConfidence(1)
	c5 := services.CalibrationConfidence(5)
	c10 := services.CalibrationConfidence(10)

	if c0 >= c1 || c1 >= c5 || c5 >= c10 {
		t.Errorf("confidence should increase with observations: c0=%f c1=%f c5=%f c10=%f", c0, c1, c5, c10)
	}
	if c10 > 1.0 {
		t.Errorf("confidence should not exceed 1.0, got %f", c10)
	}
}
