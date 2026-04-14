package services

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CGMActivePatientLister enumerates patients who have CGM data available
// for period report computation. Production wiring is a Phase 6 follow-up
// that connects this to a KB-20 query returning patients with cgm_active=true
// and a timestamp of their most recent CGMPeriodReport (so the batch can
// check the 14-day gate).
type CGMActivePatientLister interface {
	ListCGMActivePatientIDs(ctx context.Context) ([]CGMActivePatient, error)
}

// CGMActivePatient is one row returned by the lister. LastReportAt is nil
// when the patient has never had a period report computed — in that case
// the batch treats them as immediately eligible.
type CGMActivePatient struct {
	PatientID    string
	LastReportAt *time.Time
}

// CGMReadingFetcher fetches raw glucose readings for a patient within a
// time window. Production wiring is a Phase 6 follow-up — the current
// raw CGM reading store (likely a time-series table) is not yet part of
// KB-26's repository layer.
type CGMReadingFetcher interface {
	FetchCGMReadings(ctx context.Context, patientID string, start, end time.Time) ([]GlucoseReading, error)
}

// CGMDailyBatch runs every day at 01:00 UTC and, for each patient whose
// last period report is ≥14 days old (or who has never had one),
// computes a fresh 14-day CGMPeriodReport. Phase 6 P6-4 — first KB-26
// BatchScheduler consumer beyond BPContextDailyBatch.
//
// Scope note — both CGMActivePatientLister and CGMReadingFetcher are
// optional. When either is nil, the batch runs in heartbeat mode and
// logs the number of patients that would have been evaluated without
// fetching readings. The computation (ComputePeriodReport) is fully
// built + tested; the missing piece is the data source. When the KB-20
// CGM patient query + raw reading fetcher land (Phase 6 follow-up),
// wiring them here activates full per-patient report computation
// without changing the batch's cadence or scheduler integration.
type CGMDailyBatch struct {
	repo    CGMActivePatientLister
	fetcher CGMReadingFetcher
	log     *zap.Logger
}

// NewCGMDailyBatch wires the dependencies.
func NewCGMDailyBatch(repo CGMActivePatientLister, fetcher CGMReadingFetcher, log *zap.Logger) *CGMDailyBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &CGMDailyBatch{repo: repo, fetcher: fetcher, log: log}
}

// Name implements BatchJob.
func (j *CGMDailyBatch) Name() string { return "cgm_daily" }

// ShouldRun implements BatchJob — fires only at 01:00 UTC. The KB-26
// scheduler ticks hourly (Phase 5 P5-3); ShouldRun filters to one fire
// per day per ticker.
func (j *CGMDailyBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Hour() == 1
}

// Run iterates CGM-active patients and computes a new period report for
// any patient whose last report is ≥14 days old. In heartbeat mode
// (repo or fetcher nil) it logs the candidate count without fetching
// readings or computing reports.
func (j *CGMDailyBatch) Run(ctx context.Context) error {
	if j.repo == nil {
		j.log.Warn("cgm daily batch: repo nil, skipping")
		return nil
	}
	patients, err := j.repo.ListCGMActivePatientIDs(ctx)
	if err != nil {
		return err
	}

	// Identify patients whose last report is stale (≥14 days old or never).
	now := time.Now().UTC()
	stale := make([]CGMActivePatient, 0, len(patients))
	for _, p := range patients {
		if p.LastReportAt == nil || now.Sub(*p.LastReportAt) >= 14*24*time.Hour {
			stale = append(stale, p)
		}
	}

	if j.fetcher == nil {
		j.log.Info("cgm daily batch heartbeat",
			zap.Int("cgm_active_patient_count", len(patients)),
			zap.Int("due_for_report", len(stale)),
			zap.String("note", "raw reading fetcher pending Phase 6 follow-up — needs KB-20 CGM reading endpoint"))
		return nil
	}

	// Full per-patient evaluation. Errors are logged and isolated.
	computed := 0
	for _, p := range stale {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		periodEnd := now
		periodStart := periodEnd.Add(-14 * 24 * time.Hour)
		readings, fetchErr := j.fetcher.FetchCGMReadings(ctx, p.PatientID, periodStart, periodEnd)
		if fetchErr != nil {
			j.log.Warn("cgm daily batch: fetch failed",
				zap.String("patient_id", p.PatientID),
				zap.Error(fetchErr))
			continue
		}
		report := ComputePeriodReport(readings, periodStart, periodEnd)
		// Persistence of the report is a Phase 6 follow-up — the model
		// and table already exist (migration 005_cgm_tables.sql) but the
		// repository write-path isn't wired to this batch yet.
		j.log.Debug("cgm period report computed",
			zap.String("patient_id", p.PatientID),
			zap.Float64("tir_pct", report.TIRPct),
			zap.Float64("gri", report.GRI),
			zap.String("gri_zone", report.GRIZone),
			zap.Bool("sufficient", report.SufficientData))
		computed++
	}
	j.log.Info("cgm daily batch complete",
		zap.Int("cgm_active_patient_count", len(patients)),
		zap.Int("due_for_report", len(stale)),
		zap.Int("computed", computed))
	return nil
}
