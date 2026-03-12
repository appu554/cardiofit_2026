import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { pipeline1Api } from '@/lib/pipeline1-api';
import type { JobMetrics, ReviewTask, PageStats, ReviewSubPhase } from '@/types/pipeline1';

// ============================================================================
// Phase Gate Logic — Full 5-Phase Workflow (7 sub-phases)
//
// Six boolean gates derived from existing query data. No new API calls except
// pageStats (already available). Each gate controls a phase transition:
//   Phase 1  → 2A: reportAcknowledged (local state)
//   Phase 2A → 2B: tier1Complete (all DISAGREEMENT + SPOT_CHECK tasks resolved)
//   Phase 2B → 3A: pageBrowseComplete (all pages have decisions)
//   Phase 3A/3B → 4: lowConfComplete (F-only reviewed + L1_RECOVERY resolved)
//   Phase 4  → 5:  revalidationPassed (CoverageGuard delta check passed)
// ============================================================================

export interface PhaseGates {
  reportAcknowledged: boolean;
  tier1Complete: boolean;
  pageBrowseComplete: boolean;
  lowConfComplete: boolean;
  revalidationPassed: boolean;
  allGatesPassed: boolean;
}

export function usePhaseGates(
  jobId: string,
  reportAcknowledged: boolean,
  revalidationRequired = false,
): PhaseGates {
  const { data: metrics } = useQuery<JobMetrics>({
    queryKey: ['pipeline1-metrics', jobId],
    queryFn: () => pipeline1Api.jobs.getMetrics(jobId),
  });

  const { data: tasks } = useQuery<ReviewTask[]>({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
  });

  const { data: pageStats } = useQuery<PageStats>({
    queryKey: ['pipeline1-page-stats', jobId],
    queryFn: () => pipeline1Api.pages.getStats(jobId),
  });

  return useMemo(() => {
    // Gate 1: reviewer acknowledged the CoverageGuard report
    const gate1 = reportAcknowledged;

    // Gate 2: all Tier-1 fact review tasks (DISAGREEMENT + SPOT_CHECK) are resolved
    const tier1Tasks = tasks?.filter(
      (t) => t.taskType === 'DISAGREEMENT' || t.taskType === 'PASSAGE_SPOT_CHECK',
    ) ?? [];
    const gate2 = tier1Tasks.length === 0 || tier1Tasks.every((t) => t.status === 'RESOLVED');

    // Gate 3: all pages have decisions AND tier review thresholds met
    // - 100% of Tier 1 (patient safety) spans must be reviewed
    // - ≥20% of Tier 2 (clinical accuracy) spans must be reviewed
    const gate3 = pageStats
      ? pageStats.pagesNoDecision === 0
        && (pageStats.tierStats
          ? pageStats.tierStats.tier1Total === pageStats.tierStats.tier1Reviewed
            && pageStats.tierStats.tier2Pct >= 20
          : true) // graceful fallback if tierStats not yet populated
      : false;

    // Gate 4: all remaining tasks (L1_RECOVERY + any pending) are resolved
    const pendingTasks = tasks?.filter((t) => t.status === 'PENDING') ?? [];
    const gate4 = pendingTasks.length === 0;

    // Gate 5: re-validation passed on the final span set
    const gate5 = revalidationRequired
      ? false
      : (metrics ? metrics.edited === 0 && metrics.rejected === 0 : false) || gate4;

    const allGates = gate1 && gate2 && gate3 && gate4 && gate5;

    return {
      reportAcknowledged: gate1,
      tier1Complete: gate2,
      pageBrowseComplete: gate3,
      lowConfComplete: gate4,
      revalidationPassed: gate5,
      allGatesPassed: allGates,
    };
  }, [reportAcknowledged, tasks, pageStats, metrics, revalidationRequired]);
}

/**
 * Determines whether a given sub-phase is unlocked based on the current gates.
 * Each sub-phase requires all gates of prior phases to be passed.
 */
export function isSubPhaseUnlocked(phase: ReviewSubPhase, gates: PhaseGates): boolean {
  switch (phase) {
    case '1':  return true;
    case '2a': return gates.reportAcknowledged;
    case '2b': return gates.reportAcknowledged && gates.tier1Complete;
    case '3a': return gates.reportAcknowledged && gates.tier1Complete && gates.pageBrowseComplete;
    case '3b': return gates.reportAcknowledged && gates.tier1Complete && gates.pageBrowseComplete;
    case '4':  return gates.reportAcknowledged && gates.tier1Complete
                   && gates.pageBrowseComplete && gates.lowConfComplete;
    case '5':  return gates.reportAcknowledged && gates.tier1Complete
                   && gates.pageBrowseComplete && gates.lowConfComplete
                   && gates.revalidationPassed;
    default:   return false;
  }
}

/**
 * Determines whether a given sub-phase is complete (all its own work is done).
 */
export function isSubPhaseComplete(phase: ReviewSubPhase, gates: PhaseGates): boolean {
  switch (phase) {
    case '1':  return gates.reportAcknowledged;
    case '2a': return gates.tier1Complete;
    case '2b': return gates.pageBrowseComplete;
    case '3a': return gates.lowConfComplete; // shared gate with 3b
    case '3b': return gates.lowConfComplete;
    case '4':  return gates.revalidationPassed;
    case '5':  return false; // sign-off is terminal — never auto-complete
    default:   return false;
  }
}
