-- kb-26-metabolic-digital-twin/migrations/008_bp_context_raw_phenotype.sql
-- Phase 5 P5-1: Add raw_phenotype column to bp_context_history.
--
-- The stability engine can damp a proposed transition during the dwell
-- window (default 14 days). Before P5-1, when dampening fired, the raw
-- classifier output was lost — only the held-stable phenotype was
-- persisted, with confidence='DAMPED'. This made it impossible to detect
-- "the algorithm has consistently disagreed with the held phenotype for
-- N days" — exactly the signal that should yield the dwell.
--
-- This migration adds raw_phenotype to capture the un-dampened classifier
-- output. The orchestrator now writes both: phenotype is the post-engine
-- stable state, raw_phenotype is the pre-engine proposal. The stability
-- engine reads recent raw_phenotype values via stability.History.RawMatchRate
-- and overrides the dwell when the in-window agreement rate meets
-- Policy.MaxDwellOverrideRate (default 0.7).
--
-- Backwards compat: column is nullable. Snapshots written before this
-- migration leave raw_phenotype = NULL, and the engine treats NULL/empty
-- raw entries as "no signal" (excluded from the rate calculation).

ALTER TABLE bp_context_history
    ADD COLUMN IF NOT EXISTS raw_phenotype VARCHAR(30);
