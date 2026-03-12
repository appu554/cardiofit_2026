'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle, Loader2, FileCheck, ShieldAlert, Activity, BookOpen, ChevronDown, ChevronRight } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { PageReviewMode } from './PageReviewMode';
import { cn } from '@/lib/utils';
import type { ExtractionJob, PageInfo, PageStats, TierReviewStats } from '@/types/pipeline1';

// =============================================================================
// Props
// =============================================================================

interface Phase2BPageBrowseProps {
  jobId: string;
  job: ExtractionJob;
  onActionComplete: () => void;
  onPhaseComplete: () => void;
}

// =============================================================================
// Tier Progress Bar — compact single-row bar for each tier
// =============================================================================

function TierProgressRow({
  label,
  icon,
  total,
  reviewed,
  threshold,
  colorClass,
  barColorClass,
}: {
  label: string;
  icon: React.ReactNode;
  total: number;
  reviewed: number;
  threshold: number; // 100 = must review all, 20 = sample ≥20%
  colorClass: string;
  barColorClass: string;
}) {
  const pct = total > 0 ? (reviewed / total) * 100 : 0;
  const met = total === 0 || pct >= threshold;

  return (
    <div className="flex items-center gap-2">
      {/* Icon + label */}
      <div className={cn('flex items-center gap-1 w-36 shrink-0', colorClass)}>
        {icon}
        <span className="text-[10px] font-semibold truncate">{label}</span>
      </div>

      {/* Progress bar */}
      <div className="flex-1 h-1.5 bg-gray-100 rounded-full overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all duration-500', barColorClass)}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>

      {/* Count + pct */}
      <span className="text-[10px] text-gray-500 w-20 text-right tabular-nums shrink-0">
        {reviewed}/{total}
        {total > 0 && (
          <span className="ml-1 font-semibold">{pct.toFixed(0)}%</span>
        )}
      </span>

      {/* Threshold indicator */}
      {met ? (
        <CheckCircle className="h-3 w-3 text-green-500 shrink-0" />
      ) : (
        <span className="text-[8px] text-gray-400 shrink-0 w-3">&ge;{threshold}%</span>
      )}
    </div>
  );
}

// =============================================================================
// Phase 2B — Page Browse Review (Tier-Based)
//
// Wraps the existing PageReviewMode component with phase-level tracking:
//   - 3-row tier progress dashboard (T1: 100%, T2: ≥20%, T3: info only)
//   - Page progress bar
//   - Bulk Accept Clean button (only for pages with T1 fully reviewed)
//   - Auto-fires onPhaseComplete when gate conditions met
// =============================================================================

export function Phase2BPageBrowse({
  jobId,
  job,
  onActionComplete,
  onPhaseComplete,
}: Phase2BPageBrowseProps) {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const reviewerId = user?.sub || 'unknown';

  const [bulkLoading, setBulkLoading] = useState(false);
  const [bulkResult, setBulkResult] = useState<{ accepted: number; skipped: number } | null>(null);
  const [panelOpen, setPanelOpen] = useState(false);

  // ---------------------------------------------------------------------------
  // Fetch page stats for progress tracking + phase completion detection
  // ---------------------------------------------------------------------------

  const { data: pageStats, refetch: refetchStats } = useQuery<PageStats>({
    queryKey: ['pipeline1-page-stats', jobId],
    queryFn: () => pipeline1Api.pages.getStats(jobId),
    enabled: !!jobId,
  });

  // ---------------------------------------------------------------------------
  // Fetch full page list for bulk accept filtering
  // ---------------------------------------------------------------------------

  const { data: pages } = useQuery<PageInfo[]>({
    queryKey: ['pipeline1-pages', jobId],
    queryFn: () => pipeline1Api.pages.list(jobId),
    enabled: !!jobId,
  });

  // ---------------------------------------------------------------------------
  // Derived values
  // ---------------------------------------------------------------------------

  const totalPages = pageStats?.totalPages ?? 0;
  const decidedPages = totalPages - (pageStats?.pagesNoDecision ?? 0);
  const progressPct = totalPages > 0 ? (decidedPages / totalPages) * 100 : 0;
  const allDecided = pageStats ? pageStats.pagesNoDecision === 0 : false;
  const ts = pageStats?.tierStats;

  // Clean pages eligible for bulk accept:
  //   - risk === 'clean' (no oracle/disagreement)
  //   - no decision yet
  //   - all Tier 1 spans on the page are already reviewed
  const cleanUndecidedPages = useMemo(() => {
    if (!pages) return [];
    return pages.filter(
      (p) => p.risk === 'clean' && !p.decision && p.tier1Total === p.tier1Reviewed,
    );
  }, [pages]);

  // Pages that were skipped by bulk accept due to unreviewed T1 spans
  const skippedByT1 = useMemo(() => {
    if (!pages) return 0;
    return pages.filter(
      (p) => p.risk === 'clean' && !p.decision && p.tier1Total !== p.tier1Reviewed,
    ).length;
  }, [pages]);

  const hasCleanPages = cleanUndecidedPages.length > 0;

  // Phase completion: all pages decided + T1 100% + T2 ≥20%
  const phaseComplete = allDecided
    && (ts
      ? ts.tier1Total === ts.tier1Reviewed && ts.tier2Pct >= 20
      : true);

  // ---------------------------------------------------------------------------
  // Phase completion detection
  // ---------------------------------------------------------------------------

  useEffect(() => {
    if (phaseComplete && totalPages > 0) {
      onPhaseComplete();
    }
  }, [phaseComplete, totalPages, onPhaseComplete]);

  // ---------------------------------------------------------------------------
  // Bulk Accept Clean Pages
  // ---------------------------------------------------------------------------

  const handleBulkAcceptClean = useCallback(async () => {
    if (cleanUndecidedPages.length === 0) return;

    setBulkLoading(true);
    setBulkResult(null);

    let accepted = 0;
    try {
      for (const page of cleanUndecidedPages) {
        try {
          await pipeline1Api.pages.decide(jobId, page.pageNumber, {
            action: 'ACCEPT',
            reviewerId,
          });
          accepted++;
        } catch {
          // Backend 409 = T1 guard blocked this page; skip and continue
        }
      }

      setBulkResult({ accepted, skipped: cleanUndecidedPages.length - accepted });

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] }),
        queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] }),
        queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] }),
      ]);

      onActionComplete();
    } catch {
      await refetchStats();
    } finally {
      setBulkLoading(false);
    }
  }, [cleanUndecidedPages, jobId, reviewerId, queryClient, onActionComplete, refetchStats]);

  // ---------------------------------------------------------------------------
  // Wrap onActionComplete to also refetch phase-level stats
  // ---------------------------------------------------------------------------

  const handleActionComplete = useCallback(() => {
    refetchStats();
    queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
    onActionComplete();
  }, [refetchStats, queryClient, jobId, onActionComplete]);

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Phase Header Bar — collapsible, closed by default */}
      <div className="shrink-0 bg-white border-b border-gray-200 px-4 py-1.5">
        <div className="flex items-center justify-between gap-4">
          {/* Left: Phase title — click to toggle panel */}
          <button
            onClick={() => setPanelOpen(!panelOpen)}
            className="flex items-center gap-1.5 hover:opacity-80 transition-opacity"
          >
            {panelOpen ? (
              <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5 text-gray-400" />
            )}
            <FileCheck className="h-4 w-4 text-[#1B3A5C] shrink-0" />
            <span className="text-xs font-bold text-[#1B3A5C] whitespace-nowrap">
              Page Browse — Tier-Based Review
            </span>
            {/* Inline summary when collapsed */}
            {!panelOpen && ts && (
              <span className="text-[10px] text-gray-400 ml-2">
                T1:{ts.tier1Reviewed}/{ts.tier1Total} · T2:{ts.tier2Reviewed}/{ts.tier2Total} · Pages:{decidedPages}/{totalPages}
              </span>
            )}
          </button>

          {/* Right: Bulk Accept Clean button — always visible */}
          <div className="flex items-center gap-2 shrink-0">
            {bulkResult && (
              <span className="text-[10px] text-green-700 font-semibold flex items-center gap-1">
                <CheckCircle className="h-3 w-3" />
                {bulkResult.accepted} accepted
                {bulkResult.skipped > 0 && (
                  <span className="text-amber-600 ml-1">({bulkResult.skipped} blocked by T1)</span>
                )}
              </span>
            )}

            <button
              onClick={handleBulkAcceptClean}
              disabled={!hasCleanPages || bulkLoading}
              className={cn(
                'px-3 py-1 rounded text-[11px] font-semibold flex items-center gap-1.5 transition-colors',
                hasCleanPages && !bulkLoading
                  ? 'bg-green-600 text-white hover:bg-green-700'
                  : 'bg-gray-200 text-gray-400 cursor-not-allowed',
              )}
              title={skippedByT1 > 0
                ? `${skippedByT1} clean page(s) have unreviewed T1 spans and cannot be bulk-accepted`
                : undefined
              }
            >
              {bulkLoading ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Accepting...
                </>
              ) : (
                <>
                  <CheckCircle className="h-3 w-3" />
                  Bulk Accept Clean
                  {hasCleanPages && (
                    <span className="bg-white/20 rounded px-1 py-0.5 text-[9px] font-bold">
                      {cleanUndecidedPages.length}
                    </span>
                  )}
                </>
              )}
            </button>
          </div>
        </div>

        {/* Collapsible content: Tier Progress + Page Bar */}
        {panelOpen && (
          <>
            {/* Tier Progress Dashboard */}
            {ts && (
              <div className="mt-2 space-y-1">
                <TierProgressRow
                  label="T1 Patient Safety"
                  icon={<ShieldAlert className="h-3 w-3" />}
                  total={ts.tier1Total}
                  reviewed={ts.tier1Reviewed}
                  threshold={100}
                  colorClass="text-red-700"
                  barColorClass={ts.tier1Total === ts.tier1Reviewed ? 'bg-green-500' : 'bg-red-500'}
                />
                <TierProgressRow
                  label="T2 Clinical Accuracy"
                  icon={<Activity className="h-3 w-3" />}
                  total={ts.tier2Total}
                  reviewed={ts.tier2Reviewed}
                  threshold={20}
                  colorClass="text-amber-700"
                  barColorClass={ts.tier2Pct >= 20 ? 'bg-green-500' : 'bg-amber-500'}
                />
                <TierProgressRow
                  label="T3 Informational"
                  icon={<BookOpen className="h-3 w-3" />}
                  total={ts.tier3Total}
                  reviewed={ts.tier3Reviewed}
                  threshold={0}
                  colorClass="text-blue-700"
                  barColorClass="bg-blue-400"
                />
              </div>
            )}

            {/* Page progress bar */}
            <div className="mt-2">
              <div className="flex items-center justify-between mb-0.5">
                <span className="text-[10px] text-gray-500">
                  Pages:{' '}
                  <span className="font-bold text-gray-700">{decidedPages}/{totalPages}</span>
                </span>
              </div>
              <div className="h-1 bg-gray-100 rounded-full overflow-hidden">
                <div
                  className={cn(
                    'h-full rounded-full transition-all duration-500',
                    allDecided ? 'bg-green-500' : 'bg-[#1B3A5C]',
                  )}
                  style={{ width: `${progressPct}%` }}
                />
              </div>
            </div>

            {/* Completion badge */}
            {phaseComplete && totalPages > 0 && (
              <div className="flex items-center gap-1.5 mt-1.5">
                <CheckCircle className="h-3 w-3 text-green-600" />
                <span className="text-[10px] font-semibold text-green-700">
                  All tier thresholds met &amp; all pages decided — phase complete
                </span>
              </div>
            )}
          </>
        )}
      </div>

      {/* Wrapped PageReviewMode — fills remaining space */}
      <div className="flex-1 min-h-0 overflow-hidden">
        <PageReviewMode
          jobId={jobId}
          job={job}
          onBack={() => {
            // Phase wrapper manages navigation; no-op here since the phase
            // stepper handles backward navigation at the ReviewShell level.
          }}
          onActionComplete={handleActionComplete}
        />
      </div>
    </div>
  );
}
