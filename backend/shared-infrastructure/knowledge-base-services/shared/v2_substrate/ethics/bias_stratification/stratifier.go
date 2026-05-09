package bias_stratification

import (
	"context"
	"fmt"
)

// Sample is a single resident-level metric observation tagged with the
// demographic attributes needed for stratified disparity analysis.
type Sample struct {
	ResidentID   string
	Value        float64
	Demographics map[Dimension]string
}

// MetricSource streams Sample values for a named metric. Implementations
// are expected to close the returned channel when the stream is exhausted
// or the context is cancelled.
type MetricSource interface {
	StreamMetrics(ctx context.Context, metric string) (<-chan Sample, error)
}

// Stratifier consumes samples from an injected MetricSource and produces
// per-stratum means suitable for pattern_detection.DetectBiasDisparity.
type Stratifier struct {
	src MetricSource
}

// NewStratifier wires a Stratifier around the given MetricSource. The
// stratifier holds no state of its own and is safe to reuse across calls.
func NewStratifier(src MetricSource) *Stratifier {
	return &Stratifier{src: src}
}

// StratifyByDimension drains the metric stream and returns the mean value
// of the metric within each stratum of dim. Samples whose Demographics map
// has no value (or an empty value) for dim are dropped as un-classified.
//
// Cancellation: if ctx is cancelled mid-drain the function returns
// ctx.Err() and the partial result is discarded — callers see the
// truncation rather than silent partial output.
func (s *Stratifier) StratifyByDimension(ctx context.Context, metric string, dim Dimension) (map[string]float64, error) {
	ch, err := s.src.StreamMetrics(ctx, metric)
	if err != nil {
		return nil, fmt.Errorf("stream metrics %q: %w", metric, err)
	}
	sums := make(map[string]float64)
	counts := make(map[string]int)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case sample, ok := <-ch:
			if !ok {
				return finalizeMeans(sums, counts), nil
			}
			stratum := sample.Demographics[dim]
			if stratum == "" {
				continue
			}
			sums[stratum] += sample.Value
			counts[stratum]++
		}
	}
}

// StratifyAll drains the metric stream once and fans out across all six
// dimensions, returning a per-dimension map of stratum means. Equivalent
// to six calls to StratifyByDimension but consumes the source only once.
func (s *Stratifier) StratifyAll(ctx context.Context, metric string) (map[Dimension]map[string]float64, error) {
	ch, err := s.src.StreamMetrics(ctx, metric)
	if err != nil {
		return nil, fmt.Errorf("stream metrics %q: %w", metric, err)
	}
	sums := make(map[Dimension]map[string]float64, len(AllDimensions))
	counts := make(map[Dimension]map[string]int, len(AllDimensions))
	for _, d := range AllDimensions {
		sums[d] = make(map[string]float64)
		counts[d] = make(map[string]int)
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case sample, ok := <-ch:
			if !ok {
				out := make(map[Dimension]map[string]float64, len(AllDimensions))
				for _, d := range AllDimensions {
					out[d] = finalizeMeans(sums[d], counts[d])
				}
				return out, nil
			}
			for _, d := range AllDimensions {
				stratum := sample.Demographics[d]
				if stratum == "" {
					continue
				}
				sums[d][stratum] += sample.Value
				counts[d][stratum]++
			}
		}
	}
}

func finalizeMeans(sums map[string]float64, counts map[string]int) map[string]float64 {
	out := make(map[string]float64, len(sums))
	for k, sum := range sums {
		c := counts[k]
		if c == 0 {
			continue
		}
		out[k] = sum / float64(c)
	}
	return out
}
