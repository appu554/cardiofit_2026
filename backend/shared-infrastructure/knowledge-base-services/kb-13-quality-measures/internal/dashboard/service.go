// Package dashboard provides quality measure analytics and visualization data.
package dashboard

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/repository"
)

// Service provides dashboard analytics functionality.
type Service struct {
	db           *sql.DB
	resultRepo   *repository.ResultRepository
	careGapRepo  *repository.CareGapRepository
	measureStore *models.MeasureStore
	logger       *zap.Logger
}

// NewService creates a new dashboard service.
func NewService(
	db *sql.DB,
	resultRepo *repository.ResultRepository,
	careGapRepo *repository.CareGapRepository,
	measureStore *models.MeasureStore,
	logger *zap.Logger,
) *Service {
	return &Service{
		db:           db,
		resultRepo:   resultRepo,
		careGapRepo:  careGapRepo,
		measureStore: measureStore,
		logger:       logger,
	}
}

// OverviewMetrics provides top-level dashboard metrics.
type OverviewMetrics struct {
	TotalMeasures    int        `json:"total_measures"`
	ActiveMeasures   int        `json:"active_measures"`
	AverageScore     float64    `json:"average_score"`
	TotalCareGaps    int        `json:"total_care_gaps"`
	OpenCareGaps     int        `json:"open_care_gaps"`
	OverdueCareGaps  int        `json:"overdue_care_gaps"`
	LastCalculatedAt *time.Time `json:"last_calculated_at,omitempty"`
	TrendDirection   string     `json:"trend_direction"` // improving, declining, stable
}

// GetOverview returns high-level dashboard metrics.
func (s *Service) GetOverview(ctx context.Context) (*OverviewMetrics, error) {
	metrics := &OverviewMetrics{
		TotalMeasures:  s.measureStore.Count(),
		ActiveMeasures: s.countActiveMeasures(),
	}

	// Get care gap stats
	gapStats, err := s.getCareGapStats(ctx)
	if err != nil {
		s.logger.Warn("Failed to get care gap stats", zap.Error(err))
	} else {
		metrics.TotalCareGaps = gapStats.Total
		metrics.OpenCareGaps = gapStats.Open
		metrics.OverdueCareGaps = gapStats.Overdue
	}

	// Get average score and last calculation time
	scoreStats, err := s.getScoreStats(ctx)
	if err != nil {
		s.logger.Warn("Failed to get score stats", zap.Error(err))
	} else {
		metrics.AverageScore = scoreStats.Average
		metrics.LastCalculatedAt = scoreStats.LastCalculated
		metrics.TrendDirection = scoreStats.Trend
	}

	return metrics, nil
}

// MeasurePerformance shows performance for a specific measure.
type MeasurePerformance struct {
	MeasureID       string    `json:"measure_id"`
	MeasureTitle    string    `json:"measure_title"`
	Domain          string    `json:"domain"`
	CurrentScore    float64   `json:"current_score"`
	TargetScore     float64   `json:"target_score"`
	Benchmark       float64   `json:"benchmark"`
	ScoreGap        float64   `json:"score_gap"`
	Trend           string    `json:"trend"`
	TrendPercentage float64   `json:"trend_percentage"`
	CareGapCount    int       `json:"care_gap_count"`
	LastCalculated  time.Time `json:"last_calculated"`
}

// GetMeasurePerformance returns performance metrics for all measures.
func (s *Service) GetMeasurePerformance(ctx context.Context) ([]*MeasurePerformance, error) {
	measures := s.measureStore.GetAllMeasures()
	performances := make([]*MeasurePerformance, 0, len(measures))

	for _, measure := range measures {
		perf, err := s.getMeasurePerformance(ctx, measure)
		if err != nil {
			s.logger.Warn("Failed to get measure performance",
				zap.String("measure_id", measure.ID),
				zap.Error(err),
			)
			continue
		}
		performances = append(performances, perf)
	}

	return performances, nil
}

// GetMeasurePerformanceByID returns performance for a specific measure.
func (s *Service) GetMeasurePerformanceByID(ctx context.Context, measureID string) (*MeasurePerformance, error) {
	measure := s.measureStore.GetMeasure(measureID)
	if measure == nil {
		return nil, fmt.Errorf("measure not found: %s", measureID)
	}

	return s.getMeasurePerformance(ctx, measure)
}

// ProgramSummary aggregates metrics by quality program.
type ProgramSummary struct {
	Program       string  `json:"program"`
	MeasureCount  int     `json:"measure_count"`
	AverageScore  float64 `json:"average_score"`
	MeetingTarget int     `json:"meeting_target"`
	BelowTarget   int     `json:"below_target"`
	TotalGaps     int     `json:"total_gaps"`
}

// GetProgramSummaries returns metrics grouped by quality program.
func (s *Service) GetProgramSummaries(ctx context.Context) ([]*ProgramSummary, error) {
	// Group measures by program
	programMeasures := make(map[models.QualityProgram][]*models.Measure)
	for _, measure := range s.measureStore.GetAllMeasures() {
		programMeasures[measure.Program] = append(programMeasures[measure.Program], measure)
	}

	summaries := make([]*ProgramSummary, 0, len(programMeasures))
	for program, measures := range programMeasures {
		summary := &ProgramSummary{
			Program:      string(program),
			MeasureCount: len(measures),
		}

		var totalScore float64
		var scoredCount int
		for _, measure := range measures {
			result, err := s.resultRepo.GetLatestByMeasure(ctx, measure.ID)
			if err != nil {
				continue
			}

			totalScore += result.Score
			scoredCount++

			// Default target is 0.75 (75%) if not specified
			target := 0.75
			if result.Score >= target {
				summary.MeetingTarget++
			} else {
				summary.BelowTarget++
			}

			gapSummary, err := s.careGapRepo.GetSummaryByMeasure(ctx, measure.ID)
			if err == nil {
				summary.TotalGaps += gapSummary.OpenGaps
			}
		}

		if scoredCount > 0 {
			summary.AverageScore = totalScore / float64(scoredCount)
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// DomainSummary aggregates metrics by clinical domain.
type DomainSummary struct {
	Domain         string  `json:"domain"`
	MeasureCount   int     `json:"measure_count"`
	AverageScore   float64 `json:"average_score"`
	TotalGaps      int     `json:"total_gaps"`
	TopPerformer   string  `json:"top_performer"`
	NeedsAttention string  `json:"needs_attention"`
}

// GetDomainSummaries returns metrics grouped by clinical domain.
func (s *Service) GetDomainSummaries(ctx context.Context) ([]*DomainSummary, error) {
	// Group measures by domain
	domainMeasures := make(map[models.ClinicalDomain][]*models.Measure)
	for _, measure := range s.measureStore.GetAllMeasures() {
		domainMeasures[measure.Domain] = append(domainMeasures[measure.Domain], measure)
	}

	summaries := make([]*DomainSummary, 0, len(domainMeasures))
	for domain, measures := range domainMeasures {
		summary := &DomainSummary{
			Domain:       string(domain),
			MeasureCount: len(measures),
		}

		var totalScore float64
		var scoredCount int
		var topScore float64
		var lowScore float64 = 1.0
		var topMeasure, lowMeasure string

		for _, measure := range measures {
			result, err := s.resultRepo.GetLatestByMeasure(ctx, measure.ID)
			if err != nil {
				continue
			}

			totalScore += result.Score
			scoredCount++

			if result.Score > topScore {
				topScore = result.Score
				topMeasure = measure.Title
			}
			if result.Score < lowScore {
				lowScore = result.Score
				lowMeasure = measure.Title
			}

			gapSummary, err := s.careGapRepo.GetSummaryByMeasure(ctx, measure.ID)
			if err == nil {
				summary.TotalGaps += gapSummary.OpenGaps
			}
		}

		if scoredCount > 0 {
			summary.AverageScore = totalScore / float64(scoredCount)
		}
		summary.TopPerformer = topMeasure
		summary.NeedsAttention = lowMeasure

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// TrendData provides historical performance data for charting.
type TrendData struct {
	MeasureID string       `json:"measure_id"`
	Points    []TrendPoint `json:"points"`
}

// TrendPoint is a single data point in a trend.
type TrendPoint struct {
	Date  time.Time `json:"date"`
	Score float64   `json:"score"`
}

// GetTrendData returns historical score data for a measure.
func (s *Service) GetTrendData(ctx context.Context, measureID string, months int) (*TrendData, error) {
	if months <= 0 {
		months = 12
	}

	query := `
		SELECT period_end, score
		FROM calculation_results
		WHERE measure_id = $1
			AND created_at >= NOW() - INTERVAL '1 month' * $2
		ORDER BY period_end ASC
	`

	rows, err := s.db.QueryContext(ctx, query, measureID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to query trend data: %w", err)
	}
	defer rows.Close()

	trend := &TrendData{
		MeasureID: measureID,
		Points:    make([]TrendPoint, 0),
	}

	for rows.Next() {
		var point TrendPoint
		if err := rows.Scan(&point.Date, &point.Score); err != nil {
			return nil, fmt.Errorf("failed to scan trend point: %w", err)
		}
		trend.Points = append(trend.Points, point)
	}

	return trend, nil
}

// CareGapDashboard provides care gap focused analytics.
type CareGapDashboard struct {
	TotalOpen      int               `json:"total_open"`
	TotalOverdue   int               `json:"total_overdue"`
	ByPriority     map[string]int    `json:"by_priority"`
	ByMeasure      []MeasureGapCount `json:"by_measure"`
	RecentlyAdded  int               `json:"recently_added"`
	ClosedThisWeek int               `json:"closed_this_week"`
}

// MeasureGapCount shows gap count for a measure.
type MeasureGapCount struct {
	MeasureID    string `json:"measure_id"`
	MeasureTitle string `json:"measure_title"`
	GapCount     int    `json:"gap_count"`
}

// GetCareGapDashboard returns care gap analytics.
func (s *Service) GetCareGapDashboard(ctx context.Context) (*CareGapDashboard, error) {
	dashboard := &CareGapDashboard{
		ByPriority: make(map[string]int),
		ByMeasure:  make([]MeasureGapCount, 0),
	}

	// Query aggregate stats
	statsQuery := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'open') as open_count,
			COUNT(*) FILTER (WHERE status = 'open' AND due_date < NOW()) as overdue_count,
			COUNT(*) FILTER (WHERE status = 'open' AND priority = 'high') as high_priority,
			COUNT(*) FILTER (WHERE status = 'open' AND priority = 'medium') as medium_priority,
			COUNT(*) FILTER (WHERE status = 'open' AND priority = 'low') as low_priority,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days') as recently_added,
			COUNT(*) FILTER (WHERE status = 'closed' AND created_at >= NOW() - INTERVAL '7 days') as closed_this_week
		FROM care_gaps
	`

	var highPriority, mediumPriority, lowPriority int
	err := s.db.QueryRowContext(ctx, statsQuery).Scan(
		&dashboard.TotalOpen,
		&dashboard.TotalOverdue,
		&highPriority,
		&mediumPriority,
		&lowPriority,
		&dashboard.RecentlyAdded,
		&dashboard.ClosedThisWeek,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query care gap stats: %w", err)
	}

	// Assign to map after scanning
	dashboard.ByPriority["high"] = highPriority
	dashboard.ByPriority["medium"] = mediumPriority
	dashboard.ByPriority["low"] = lowPriority

	// Query by measure
	measureQuery := `
		SELECT measure_id, COUNT(*) as gap_count
		FROM care_gaps
		WHERE status = 'open'
		GROUP BY measure_id
		ORDER BY gap_count DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, measureQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query measure gaps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mgc MeasureGapCount
		if err := rows.Scan(&mgc.MeasureID, &mgc.GapCount); err != nil {
			continue
		}
		if measure := s.measureStore.GetMeasure(mgc.MeasureID); measure != nil {
			mgc.MeasureTitle = measure.Title
		}
		dashboard.ByMeasure = append(dashboard.ByMeasure, mgc)
	}

	return dashboard, nil
}

// Helper methods

func (s *Service) countActiveMeasures() int {
	count := 0
	for _, measure := range s.measureStore.GetAllMeasures() {
		if measure.Active {
			count++
		}
	}
	return count
}

type careGapStats struct {
	Total   int
	Open    int
	Overdue int
}

func (s *Service) getCareGapStats(ctx context.Context) (*careGapStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'open') as open_count,
			COUNT(*) FILTER (WHERE status = 'open' AND due_date < NOW()) as overdue_count
		FROM care_gaps
	`

	var stats careGapStats
	err := s.db.QueryRowContext(ctx, query).Scan(
		&stats.Total,
		&stats.Open,
		&stats.Overdue,
	)
	if err == sql.ErrNoRows {
		return &stats, nil
	}
	return &stats, err
}

type scoreStats struct {
	Average        float64
	LastCalculated *time.Time
	Trend          string
}

func (s *Service) getScoreStats(ctx context.Context) (*scoreStats, error) {
	query := `
		SELECT
			AVG(score) as avg_score,
			MAX(created_at) as last_calc
		FROM calculation_results
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`

	var stats scoreStats
	var avgScore sql.NullFloat64
	var lastCalc sql.NullTime

	err := s.db.QueryRowContext(ctx, query).Scan(&avgScore, &lastCalc)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if avgScore.Valid {
		stats.Average = avgScore.Float64
	}
	if lastCalc.Valid {
		stats.LastCalculated = &lastCalc.Time
	}

	// Determine trend by comparing current vs previous period
	stats.Trend = s.calculateTrend(ctx)

	return &stats, nil
}

func (s *Service) calculateTrend(ctx context.Context) string {
	query := `
		SELECT
			AVG(CASE WHEN created_at >= NOW() - INTERVAL '15 days' THEN score END) as recent,
			AVG(CASE WHEN created_at < NOW() - INTERVAL '15 days' AND created_at >= NOW() - INTERVAL '30 days' THEN score END) as previous
		FROM calculation_results
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`

	var recent, previous sql.NullFloat64
	err := s.db.QueryRowContext(ctx, query).Scan(&recent, &previous)
	if err != nil || !recent.Valid || !previous.Valid {
		return "stable"
	}

	diff := recent.Float64 - previous.Float64
	if diff > 0.02 {
		return "improving"
	} else if diff < -0.02 {
		return "declining"
	}
	return "stable"
}

func (s *Service) getMeasurePerformance(ctx context.Context, measure *models.Measure) (*MeasurePerformance, error) {
	perf := &MeasurePerformance{
		MeasureID:    measure.ID,
		MeasureTitle: measure.Title,
		Domain:       string(measure.Domain),
		TargetScore:  0.75, // Default target 75%
		Benchmark:    0.70, // Default benchmark
	}

	// Get latest result
	result, err := s.resultRepo.GetLatestByMeasure(ctx, measure.ID)
	if err == nil {
		perf.CurrentScore = result.Score
		perf.LastCalculated = result.CreatedAt
		perf.ScoreGap = perf.TargetScore - result.Score
	}

	// Get care gap count
	summary, err := s.careGapRepo.GetSummaryByMeasure(ctx, measure.ID)
	if err == nil {
		perf.CareGapCount = summary.OpenGaps
	}

	// Calculate trend
	trend, err := s.GetTrendData(ctx, measure.ID, 3)
	if err == nil && len(trend.Points) >= 2 {
		first := trend.Points[0].Score
		last := trend.Points[len(trend.Points)-1].Score
		if last > first {
			perf.Trend = "improving"
			if first > 0 {
				perf.TrendPercentage = ((last - first) / first) * 100
			}
		} else if last < first {
			perf.Trend = "declining"
			if first > 0 {
				perf.TrendPercentage = ((first - last) / first) * -100
			}
		} else {
			perf.Trend = "stable"
		}
	}

	return perf, nil
}

// ComparisonReport provides period-over-period or benchmark comparison data.
type ComparisonReport struct {
	GeneratedAt       time.Time              `json:"generated_at"`
	ComparisonType    string                 `json:"comparison_type"` // period, benchmark, year_over_year
	CurrentPeriod     *PeriodData            `json:"current_period"`
	PriorPeriod       *PeriodData            `json:"prior_period,omitempty"`
	BenchmarkData     *BenchmarkComparison   `json:"benchmark_data,omitempty"`
	MeasureComparisons []MeasureComparison   `json:"measure_comparisons"`
	Summary           *ComparisonSummary     `json:"summary"`
}

// PeriodData contains aggregate data for a specific time period.
type PeriodData struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Label        string    `json:"label"`
	AverageScore float64   `json:"average_score"`
	TotalGaps    int       `json:"total_gaps"`
	MeasureCount int       `json:"measure_count"`
}

// BenchmarkComparison shows comparison against benchmarks.
type BenchmarkComparison struct {
	Source      string  `json:"source"` // HEDIS, CMS, NQF
	Year        int     `json:"year"`
	Percentile  int     `json:"percentile,omitempty"`
	National    float64 `json:"national_average,omitempty"`
	TopDecile   float64 `json:"top_decile,omitempty"`
}

// MeasureComparison shows performance change for a single measure.
type MeasureComparison struct {
	MeasureID       string  `json:"measure_id"`
	MeasureTitle    string  `json:"measure_title"`
	Program         string  `json:"program"`
	CurrentScore    float64 `json:"current_score"`
	PriorScore      float64 `json:"prior_score,omitempty"`
	BenchmarkScore  float64 `json:"benchmark_score,omitempty"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"change_percent"`
	Direction       string  `json:"direction"` // improved, declined, stable
	GapToBenchmark  float64 `json:"gap_to_benchmark,omitempty"`
	GapToTarget     float64 `json:"gap_to_target"`
	TargetScore     float64 `json:"target_score"`
}

// ComparisonSummary aggregates comparison statistics.
type ComparisonSummary struct {
	TotalMeasures     int     `json:"total_measures"`
	Improved          int     `json:"improved"`
	Declined          int     `json:"declined"`
	Stable            int     `json:"stable"`
	AboveBenchmark    int     `json:"above_benchmark"`
	BelowBenchmark    int     `json:"below_benchmark"`
	AverageChange     float64 `json:"average_change"`
	BestImprover      string  `json:"best_improver,omitempty"`
	NeedsAttention    string  `json:"needs_attention,omitempty"`
}

// ComparisonRequest specifies the comparison parameters.
type ComparisonRequest struct {
	Type        string     `json:"type"` // period, benchmark, year_over_year
	MeasureIDs  []string   `json:"measure_ids,omitempty"` // empty = all measures
	Program     string     `json:"program,omitempty"` // filter by program
	CurrentEnd  *time.Time `json:"current_end,omitempty"`
	PriorMonths int        `json:"prior_months,omitempty"` // default 3
}

// GetComparison generates a comparison report based on the request.
func (s *Service) GetComparison(ctx context.Context, req *ComparisonRequest) (*ComparisonReport, error) {
	// Set defaults
	if req.Type == "" {
		req.Type = "period"
	}
	if req.PriorMonths == 0 {
		req.PriorMonths = 3
	}

	now := time.Now().UTC()
	currentEnd := now
	if req.CurrentEnd != nil {
		currentEnd = *req.CurrentEnd
	}

	report := &ComparisonReport{
		GeneratedAt:       now,
		ComparisonType:    req.Type,
		MeasureComparisons: make([]MeasureComparison, 0),
		Summary:           &ComparisonSummary{},
	}

	// Get measures to compare
	var measures []*models.Measure
	if len(req.MeasureIDs) > 0 {
		for _, id := range req.MeasureIDs {
			if m := s.measureStore.GetMeasure(id); m != nil {
				measures = append(measures, m)
			}
		}
	} else if req.Program != "" {
		measures = s.measureStore.GetMeasuresByProgram(models.QualityProgram(req.Program))
	} else {
		measures = s.measureStore.GetAllMeasures()
	}

	// Calculate period boundaries
	currentStart := currentEnd.AddDate(0, -req.PriorMonths, 0)
	priorEnd := currentStart
	priorStart := priorEnd.AddDate(0, -req.PriorMonths, 0)

	report.CurrentPeriod = &PeriodData{
		Start: currentStart,
		End:   currentEnd,
		Label: fmt.Sprintf("%s - %s", currentStart.Format("Jan 2006"), currentEnd.Format("Jan 2006")),
	}
	report.PriorPeriod = &PeriodData{
		Start: priorStart,
		End:   priorEnd,
		Label: fmt.Sprintf("%s - %s", priorStart.Format("Jan 2006"), priorEnd.Format("Jan 2006")),
	}

	var totalCurrentScore, totalPriorScore float64
	var scoredCurrentCount, scoredPriorCount int
	var bestImproverScore float64
	var worstDeclinerScore float64

	for _, measure := range measures {
		comp := MeasureComparison{
			MeasureID:    measure.ID,
			MeasureTitle: measure.Title,
			Program:      string(measure.Program),
			TargetScore:  0.75, // Default target
		}

		// Get current period score
		currentResult, err := s.getScoreForPeriod(ctx, measure.ID, currentStart, currentEnd)
		if err == nil && currentResult > 0 {
			comp.CurrentScore = currentResult
			totalCurrentScore += currentResult
			scoredCurrentCount++
		}

		// Get prior period score
		priorResult, err := s.getScoreForPeriod(ctx, measure.ID, priorStart, priorEnd)
		if err == nil && priorResult > 0 {
			comp.PriorScore = priorResult
			totalPriorScore += priorResult
			scoredPriorCount++
		}

		// Calculate change
		if comp.PriorScore > 0 {
			comp.Change = comp.CurrentScore - comp.PriorScore
			comp.ChangePercent = (comp.Change / comp.PriorScore) * 100
		}

		// Determine direction
		if comp.Change > 0.02 {
			comp.Direction = "improved"
			report.Summary.Improved++
			if comp.Change > bestImproverScore {
				bestImproverScore = comp.Change
				report.Summary.BestImprover = measure.Title
			}
		} else if comp.Change < -0.02 {
			comp.Direction = "declined"
			report.Summary.Declined++
			if comp.Change < worstDeclinerScore {
				worstDeclinerScore = comp.Change
				report.Summary.NeedsAttention = measure.Title
			}
		} else {
			comp.Direction = "stable"
			report.Summary.Stable++
		}

		// Calculate gap to target
		comp.GapToTarget = comp.TargetScore - comp.CurrentScore

		report.MeasureComparisons = append(report.MeasureComparisons, comp)
	}

	// Calculate summary statistics
	report.Summary.TotalMeasures = len(measures)
	if scoredCurrentCount > 0 {
		report.CurrentPeriod.AverageScore = totalCurrentScore / float64(scoredCurrentCount)
		report.CurrentPeriod.MeasureCount = scoredCurrentCount
	}
	if scoredPriorCount > 0 {
		report.PriorPeriod.AverageScore = totalPriorScore / float64(scoredPriorCount)
		report.PriorPeriod.MeasureCount = scoredPriorCount
	}
	if report.Summary.TotalMeasures > 0 {
		var totalChange float64
		for _, mc := range report.MeasureComparisons {
			totalChange += mc.Change
		}
		report.Summary.AverageChange = totalChange / float64(report.Summary.TotalMeasures)
	}

	// Get care gap counts for current period
	currentGaps, _ := s.getGapCountForPeriod(ctx, currentStart, currentEnd)
	priorGaps, _ := s.getGapCountForPeriod(ctx, priorStart, priorEnd)
	report.CurrentPeriod.TotalGaps = currentGaps
	report.PriorPeriod.TotalGaps = priorGaps

	return report, nil
}

// getScoreForPeriod retrieves the average score for a measure within a time period.
func (s *Service) getScoreForPeriod(ctx context.Context, measureID string, start, end time.Time) (float64, error) {
	query := `
		SELECT AVG(score)
		FROM calculation_results
		WHERE measure_id = $1
			AND period_end >= $2
			AND period_end <= $3
	`

	var avgScore sql.NullFloat64
	err := s.db.QueryRowContext(ctx, query, measureID, start, end).Scan(&avgScore)
	if err != nil {
		return 0, err
	}

	if avgScore.Valid {
		return avgScore.Float64, nil
	}
	return 0, fmt.Errorf("no scores found for period")
}

// getGapCountForPeriod counts care gaps created within a time period.
func (s *Service) getGapCountForPeriod(ctx context.Context, start, end time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM care_gaps
		WHERE created_at >= $1 AND created_at <= $2
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, start, end).Scan(&count)
	return count, err
}
