'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Loader2, CheckCircle, Pencil, XCircle, Plus, ArrowUpRight, ChevronLeft, ChevronRight, ShieldAlert } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { SemanticText } from './SemanticText';
import { CoverageGuardAlertBanner } from './CoverageGuardAlert';
import { PdfErrorBoundary } from './PdfErrorBoundary';
import { RejectModal } from './RejectModal';
import { AddSpanModal } from './AddSpanModal';
import { PageReviewMode } from './PageReviewMode';
import { cn } from '@/lib/utils';

// Dynamic import: pdfjs-dist ESM module fails during SSR webpack bundling
const PdfHighlightViewer = dynamic(
  () => import('./PdfHighlightViewer').then((m) => m.PdfHighlightViewer),
  { ssr: false, loading: () => <div className="flex-1 flex items-center justify-center"><Loader2 className="h-6 w-6 animate-spin text-gray-300" /></div> },
);
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import type { MergedSpan, ReviewTask, ExtractionJob, SpanReviewRequest, RejectReason } from '@/types/pipeline1';

// =============================================================================
// Props
// =============================================================================

interface FactReviewLayoutProps {
  jobId: string;
  job: ExtractionJob;
  reviewTasks: ReviewTask[];
  onBack: () => void;
  onActionComplete: () => void;
  onRequestRevalidation?: () => void;
}

// =============================================================================
// FactReviewLayout — Full-screen mock-matching review experience
// =============================================================================

export function FactReviewLayout({
  jobId,
  job,
  reviewTasks,
  onBack,
  onActionComplete,
  onRequestRevalidation,
}: FactReviewLayoutProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  // Filter to span-based tasks only (L1_RECOVERY + DISAGREEMENT)
  const spanTasks = useMemo(
    () => reviewTasks.filter((t) => t.spanId),
    [reviewTasks],
  );

  // ─── State ──────────────────────────────────────────────────────────
  const [currentIdx, setCurrentIdx] = useState(0);
  const [currentSpan, setCurrentSpan] = useState<MergedSpan | null>(null);
  const [spanLoading, setSpanLoading] = useState(false);
  const [decisions, setDecisions] = useState<Record<string, string>>({});
  const [showReject, setShowReject] = useState(false);
  const [showAddSpan, setShowAddSpan] = useState(false);
  const [showEscalate, setShowEscalate] = useState(false);
  const [showSummary, setShowSummary] = useState(false);
  const [note, setNote] = useState('');
  const [editMode, setEditMode] = useState(false);
  const [editedText, setEditedText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  // ─── Browse All mode ──────────────────────────────────────────────
  const [viewMode, setViewMode] = useState<'tasks' | 'all' | 'pages'>('tasks');
  const [channelFilter, setChannelFilter] = useState<string | null>(null);
  const [allSpans, setAllSpans] = useState<MergedSpan[]>([]);
  const [allSpansLoading, setAllSpansLoading] = useState(false);

  const filteredAllSpans = useMemo(() => {
    if (!channelFilter) return allSpans;
    return allSpans.filter((s) => s.contributingChannels.includes(channelFilter));
  }, [allSpans, channelFilter]);

  const channelCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    allSpans.forEach((s) => {
      s.contributingChannels.forEach((ch) => {
        counts[ch] = (counts[ch] || 0) + 1;
      });
    });
    return counts;
  }, [allSpans]);

  const currentTask = spanTasks[currentIdx];
  const decidedCount = Object.keys(decisions).length;
  const totalTasks = viewMode === 'tasks' ? spanTasks.length : filteredAllSpans.length;

  // Verdict: PASS only when all tasks have been decided
  const pendingCount = totalTasks - decidedCount;
  const verdict = pendingCount === 0 ? 'PASS' : 'BLOCK';

  // Alert-gated confirm
  const hasCriticalAlert = currentSpan?.coverageGuardAlert?.alertSeverity === 'critical';
  const alertBlocksConfirm =
    currentSpan?.coverageGuardAlert?.type === 'numeric_mismatch' ||
    currentSpan?.coverageGuardAlert?.type === 'branch_loss';

  // ─── Fetch span when task changes ───────────────────────────────────
  useEffect(() => {
    // Browse All mode: span comes from the fetched list directly
    if (viewMode === 'all') {
      if (filteredAllSpans.length > 0 && currentIdx < filteredAllSpans.length) {
        const s = filteredAllSpans[currentIdx];
        setCurrentSpan(s);
        if (s.reviewStatus !== 'PENDING') {
          setDecisions((d) => ({ ...d, [s.id]: s.reviewStatus.toLowerCase() }));
        }
      } else {
        setCurrentSpan(null);
      }
      setSpanLoading(false);
      return;
    }
    // Tasks mode: fetch span from API
    if (!currentTask?.spanId) {
      setCurrentSpan(null);
      return;
    }
    setSpanLoading(true);
    pipeline1Api.spans
      .get(jobId, currentTask.spanId)
      .then((span) => {
        setCurrentSpan(span);
        setSpanLoading(false);
        // Initialize decision from existing review status
        if (span.reviewStatus !== 'PENDING') {
          setDecisions((d) => ({ ...d, [span.id]: span.reviewStatus.toLowerCase() }));
        }
      })
      .catch(() => {
        setCurrentSpan(null);
        setSpanLoading(false);
      });
  }, [viewMode, jobId, currentTask?.spanId, currentTask?.id, currentIdx, filteredAllSpans]);

  // Reset form state when navigating
  useEffect(() => {
    setNote('');
    setEditMode(false);
    setEditedText('');
    setError(null);
    setShowEscalate(false);
  }, [currentIdx]);

  // ─── Fetch all spans for Browse All mode (paginated) ──────────────
  useEffect(() => {
    if (viewMode !== 'all') return;
    let cancelled = false;
    setAllSpansLoading(true);
    setAllSpans([]);

    async function fetchAll() {
      const PAGE_SIZE = 500;
      let page = 1;
      let accumulated: MergedSpan[] = [];
      let hasMore = true;

      while (hasMore && !cancelled) {
        try {
          const result = await pipeline1Api.spans.list(jobId, {}, page, PAGE_SIZE);
          accumulated = [...accumulated, ...result.items];
          hasMore = result.hasMore;
          page++;
          // Progressive update: show spans as they load
          if (!cancelled) {
            setAllSpans([...accumulated]);
          }
        } catch {
          break;
        }
      }
      if (!cancelled) {
        setAllSpansLoading(false);
      }
    }

    fetchAll();
    return () => { cancelled = true; };
  }, [jobId, viewMode]);

  // Reset index on mode/filter change
  useEffect(() => {
    setCurrentIdx(0);
    setShowSummary(false);
  }, [viewMode, channelFilter]);

  // ─── Cache invalidation ─────────────────────────────────────────────
  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans-all', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-review-tasks', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
  }, [queryClient, jobId]);

  // ─── Action handlers ────────────────────────────────────────────────
  const advance = useCallback(() => {
    if (currentIdx < totalTasks - 1) {
      setCurrentIdx((i) => i + 1);
    } else if (viewMode === 'tasks') {
      setShowSummary(true);
    }
    // In "all" mode, stay on last item after action
  }, [currentIdx, totalTasks, viewMode]);

  const handleConfirm = useCallback(async () => {
    if (!currentSpan) return;
    setActionLoading(true);
    setError(null);
    try {
      const req: SpanReviewRequest = {
        reviewerId: user?.sub || 'unknown',
        ...(note ? { note } : {}),
      };
      await pipeline1Api.spans.confirm(jobId, currentSpan.id, req);
      setDecisions((d) => ({ ...d, [currentSpan.id]: 'confirmed' }));
      invalidateAll();
      onActionComplete();
      advance();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionLoading(false);
    }
  }, [currentSpan, user, note, jobId, invalidateAll, onActionComplete, advance]);

  const handleEdit = useCallback(async () => {
    if (!currentSpan || !editedText.trim()) {
      setError('Edited text is required.');
      return;
    }
    setActionLoading(true);
    setError(null);
    try {
      const req: SpanReviewRequest = {
        reviewerId: user?.sub || 'unknown',
        editedText,
        ...(note ? { note } : {}),
      };
      await pipeline1Api.spans.edit(jobId, currentSpan.id, req);
      setDecisions((d) => ({ ...d, [currentSpan.id]: 'edited' }));
      invalidateAll();
      onActionComplete();
      setEditMode(false);
      advance();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionLoading(false);
    }
  }, [currentSpan, user, note, editedText, jobId, invalidateAll, onActionComplete, advance]);

  const handleRejectConfirm = useCallback(
    async (reason: RejectReason) => {
      if (!currentSpan) return;
      setActionLoading(true);
      setError(null);
      try {
        const req: SpanReviewRequest = {
          reviewerId: user?.sub || 'unknown',
          rejectReason: reason,
          ...(note ? { note } : {}),
        };
        await pipeline1Api.spans.reject(jobId, currentSpan.id, req);
        setDecisions((d) => ({ ...d, [currentSpan.id]: 'rejected' }));
        invalidateAll();
        onActionComplete();
        setShowReject(false);
        advance();
      } catch (err) {
        setError((err as Error).message);
      } finally {
        setActionLoading(false);
      }
    },
    [currentSpan, user, note, jobId, invalidateAll, onActionComplete, advance],
  );

  const handleEscalateConfirm = useCallback(async () => {
    if (!currentSpan) return;
    setActionLoading(true);
    setError(null);
    try {
      const req: SpanReviewRequest = {
        reviewerId: user?.sub || 'unknown',
        rejectReason: 'escalated_to_sme',
        note: note || 'Escalated to subject matter expert for review',
      };
      await pipeline1Api.spans.reject(jobId, currentSpan.id, req);
      setDecisions((d) => ({ ...d, [currentSpan.id]: 'rejected' }));
      invalidateAll();
      onActionComplete();
      setShowEscalate(false);
      advance();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionLoading(false);
    }
  }, [currentSpan, user, note, jobId, invalidateAll, onActionComplete, advance]);

  // ─── Navigation ─────────────────────────────────────────────────────
  const goTo = useCallback(
    (dir: 'prev' | 'next') => {
      if (dir === 'prev' && currentIdx > 0) {
        setCurrentIdx((i) => i - 1);
        setShowSummary(false);
      }
      if (dir === 'next') {
        if (currentIdx < totalTasks - 1) {
          setCurrentIdx((i) => i + 1);
          setShowSummary(false);
        } else if (viewMode === 'tasks') {
          setShowSummary(true);
        }
        // In "all" mode, don't go past the end
      }
    },
    [currentIdx, totalTasks, viewMode],
  );

  // ─── Keyboard shortcuts ─────────────────────────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      switch (e.key.toLowerCase()) {
        case 'c':
          if (!editMode && !alertBlocksConfirm) handleConfirm();
          break;
        case 'e':
          if (!editMode && currentSpan) {
            setEditMode(true);
            setEditedText(currentSpan.text);
          }
          break;
        case 'r':
          if (!editMode) setShowReject(true);
          break;
        case 'arrowleft':
          goTo('prev');
          break;
        case 'arrowright':
          goTo('next');
          break;
        case 'j':
          goTo('next');
          break;
        case 'k':
          goTo('prev');
          break;
        case 'escape':
          setEditMode(false);
          setShowReject(false);
          setShowEscalate(false);
          break;
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [editMode, alertBlocksConfirm, handleConfirm, goTo, currentSpan]);

  // ─── Derived display values ─────────────────────────────────────────
  const jobName = job.sourcePdf?.replace(/^.*\//, '').replace(/\.pdf$/i, '') || 'Guideline Review';
  const jobIdShort = job.jobId.slice(0, 8);

  // ─── Derived: effective task type for "all" mode ─────────────────────
  const effectiveTaskType = viewMode === 'tasks'
    ? currentTask?.taskType
    : (currentSpan?.contributingChannels.includes('L1_RECOVERY') || currentSpan?.contributingChannels.includes('L1'))
      ? 'L1_RECOVERY'
      : currentSpan?.hasDisagreement ? 'DISAGREEMENT' : undefined;

  const isL1Recovery =
    effectiveTaskType === 'L1_RECOVERY' ||
    currentSpan?.contributingChannels.includes('L1_RECOVERY') ||
    currentSpan?.contributingChannels.includes('L1');

  // ─── Empty state ────────────────────────────────────────────────────
  if (viewMode === 'tasks' && spanTasks.length === 0) {
    return (
      <div className="h-[calc(100vh-64px)] -m-6 flex flex-col">
        {/* TopBar */}
        <div className="h-[52px] bg-[#1B3A5C] flex items-center justify-between px-6 shrink-0">
          <div className="flex items-center gap-5">
            <span className="text-white text-sm font-semibold tracking-wide">{jobName}</span>
            <span className="text-white/30 text-xs">|</span>
            <span className="text-white/50 text-xs">Job: {jobIdShort}</span>
          </div>
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-white/70 hover:text-white text-xs transition-colors"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back to Dashboard
          </button>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <CheckCircle className="h-12 w-12 text-green-400 mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-gray-700">No review tasks</h3>
            <p className="text-sm text-gray-500 mt-1">All facts have been reviewed or no tasks require attention.</p>
          </div>
        </div>
      </div>
    );
  }

  // ─── Summary View (tasks mode only) ─────────────────────────────────
  if (showSummary && viewMode === 'tasks') {
    const counts = { confirmed: 0, edited: 0, rejected: 0, pending: 0 };
    spanTasks.forEach((t) => {
      const d = t.spanId ? decisions[t.spanId] : undefined;
      if (d === 'confirmed') counts.confirmed++;
      else if (d === 'edited') counts.edited++;
      else if (d === 'rejected') counts.rejected++;
      else counts.pending++;
    });

    return (
      <div className="h-[calc(100vh-64px)] -m-6 flex flex-col">
        {/* TopBar */}
        <div className="h-[52px] bg-[#1B3A5C] flex items-center justify-between px-6 shrink-0">
          <div className="flex items-center gap-5">
            <span className="text-white text-sm font-semibold tracking-wide">{jobName}</span>
            <span className="text-white/30 text-xs">|</span>
            <span className="text-white/50 text-xs">Job: {jobIdShort}</span>
          </div>
          <div className="flex items-center gap-4">
            <span
              className={cn(
                'text-[11px] font-semibold tracking-wider px-2.5 py-1 rounded',
                counts.pending === 0
                  ? 'bg-[#065F46] text-[#A7F3D0]'
                  : 'bg-[#7F1D1D] text-[#FECACA]',
              )}
            >
              {counts.pending === 0 ? '✓ PASS' : `⊘ BLOCK (${counts.pending})`}
            </span>
          </div>
        </div>
        {/* Summary content */}
        <div className="flex-1 flex items-center justify-center bg-[#F7F8FA]">
          <div className="bg-white rounded-xl p-10 w-[440px] shadow-sm border border-gray-200 text-center">
            <div className="text-xl font-bold text-[#1B3A5C] mb-2">Review Summary</div>
            <div className="text-sm text-gray-500 mb-7">
              Tier 1 Facts: {totalTasks - counts.pending}/{totalTasks} reviewed
            </div>
            <div className="flex flex-col gap-2 mb-7 text-left">
              {[
                { label: 'Confirmed', count: counts.confirmed, color: 'text-green-600' },
                { label: 'Edited', count: counts.edited, color: 'text-blue-600' },
                { label: 'Rejected', count: counts.rejected, color: 'text-red-600' },
                { label: 'Pending', count: counts.pending, color: 'text-gray-500' },
              ].map((row) => (
                <div
                  key={row.label}
                  className="flex justify-between items-center px-3.5 py-2 rounded-md bg-gray-50"
                >
                  <span className="text-sm text-gray-800">{row.label}</span>
                  <span className={cn('text-sm font-bold', row.color)}>{row.count}</span>
                </div>
              ))}
            </div>
            <div
              className={cn(
                'p-3 rounded-lg mb-5 text-xs font-semibold',
                counts.pending === 0
                  ? 'bg-green-50 border border-green-200 text-green-800'
                  : 'bg-red-50 border border-red-200 text-red-800',
              )}
            >
              {counts.pending === 0
                ? '✓ All facts reviewed — ready for Re-Validation'
                : `${counts.pending} facts remaining`}
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowSummary(false);
                  setCurrentIdx(0);
                }}
                className="flex-1 px-4 py-2.5 rounded-lg border border-gray-200 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
              >
                Review Again
              </button>
              {counts.pending === 0 && onRequestRevalidation ? (
                <button
                  onClick={onRequestRevalidation}
                  className="flex-1 px-4 py-2.5 rounded-lg bg-[#1B3A5C] text-white text-sm font-semibold hover:bg-[#2A5580] transition-colors"
                >
                  Run Final Validation →
                </button>
              ) : (
                <button
                  onClick={onBack}
                  className="flex-1 px-4 py-2.5 rounded-lg bg-[#1B3A5C] text-white text-sm font-semibold hover:bg-[#2A5580] transition-colors"
                >
                  Back to Dashboard
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // ─── Main Review Layout ─────────────────────────────────────────────
  const span = currentSpan;
  const channelLabel = span?.contributingChannels?.join(' + ') || '—';
  const corroboration = span?.mergedConfidence ?? 0;
  const scoreColor =
    corroboration >= 0.7 ? 'text-green-700 bg-green-50' : corroboration >= 0.5 ? 'text-amber-700 bg-amber-50' : 'text-red-700 bg-red-50';
  const tierBg =
    span?.tier === 1
      ? 'bg-[#7F1D1D] text-[#FECACA]'
      : span?.tier === 2
        ? 'bg-amber-700 text-amber-100'
        : 'bg-gray-600 text-gray-200';

  return (
    <div className="h-[calc(100vh-64px)] -m-6 flex flex-col font-sans bg-[#F7F8FA]">
      {/* ── TopBar ───────────────────────────────────────────────────── */}
      <div className="h-[52px] bg-[#1B3A5C] flex items-center justify-between px-6 shrink-0">
        <div className="flex items-center gap-5">
          <span className="text-white text-sm font-semibold tracking-wide">{jobName}</span>
          <span className="text-white/30 text-xs">|</span>
          <span className="text-white/50 text-xs">Job: {jobIdShort}</span>
          {/* ── View mode toggle ──────────────────────────────── */}
          <div className="flex items-center bg-white/10 rounded-md p-0.5 ml-2">
            <button
              onClick={() => setViewMode('tasks')}
              className={cn(
                'px-3 py-1 text-[11px] rounded font-medium transition-colors',
                viewMode === 'tasks'
                  ? 'bg-white text-[#1B3A5C]'
                  : 'text-white/60 hover:text-white/90',
              )}
            >
              Review Tasks ({spanTasks.length})
            </button>
            <button
              onClick={() => setViewMode('all')}
              className={cn(
                'px-3 py-1 text-[11px] rounded font-medium transition-colors',
                viewMode === 'all'
                  ? 'bg-white text-[#1B3A5C]'
                  : 'text-white/60 hover:text-white/90',
              )}
            >
              All Extractions{allSpans.length > 0 ? ` (${allSpans.length}${allSpansLoading ? '…' : ''})` : ''}
            </button>
            <button
              onClick={() => setViewMode('pages')}
              className={cn(
                'px-3 py-1 text-[11px] rounded font-medium transition-colors',
                viewMode === 'pages'
                  ? 'bg-white text-[#1B3A5C]'
                  : 'text-white/60 hover:text-white/90',
              )}
            >
              Page Review
            </button>
          </div>
        </div>
        <div className="flex items-center gap-4">
          {viewMode === 'tasks' && (
            <span
              className={cn(
                'text-[11px] font-semibold tracking-wider px-2.5 py-1 rounded',
                verdict === 'PASS'
                  ? 'bg-[#065F46] text-[#A7F3D0]'
                  : 'bg-[#7F1D1D] text-[#FECACA]',
              )}
            >
              {verdict === 'PASS' ? '✓ PASS' : `⊘ BLOCK (${pendingCount})`}
            </span>
          )}
          {onRequestRevalidation && (
            <button
              onClick={onRequestRevalidation}
              className="bg-white/10 border border-white/20 text-white text-[11px] px-3 py-1.5 rounded hover:bg-white/20 transition-colors font-medium"
            >
              Re-Validate
            </button>
          )}
          <button
            onClick={onBack}
            className="bg-white/5 border border-white/15 text-white/70 text-[11px] px-3 py-1.5 rounded hover:bg-white/10 hover:text-white transition-colors"
          >
            ← Dashboard
          </button>
        </div>
      </div>

      {/* ── Page Review Mode (full takeover below top bar) ──────────── */}
      {viewMode === 'pages' && (
        <PageReviewMode
          jobId={jobId}
          job={job}
          onBack={onBack}
          onActionComplete={onActionComplete}
        />
      )}

      {/* ── Channel Filter Bar (Browse All mode) ─────────────────────── */}
      {viewMode === 'all' && (
        <div className="px-6 py-2 bg-white border-b border-gray-200 flex items-center gap-2 shrink-0 overflow-x-auto">
          <span className="text-[10px] text-gray-500 uppercase tracking-wider font-semibold mr-1 shrink-0">
            Channel:
          </span>
          <button
            onClick={() => setChannelFilter(null)}
            className={cn(
              'text-[11px] px-2.5 py-1 rounded-full border font-medium transition-colors shrink-0',
              !channelFilter
                ? 'bg-gray-800 text-white border-gray-800'
                : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100',
            )}
          >
            All ({allSpans.length})
          </button>
          {(['B', 'C', 'D', 'E', 'F', 'L1_RECOVERY'] as const).map((ch) => {
            const count = channelCounts[ch] || 0;
            if (count === 0) return null;
            const info = getChannelInfo(ch);
            return (
              <button
                key={ch}
                onClick={() => setChannelFilter(channelFilter === ch ? null : ch)}
                className={cn(
                  'text-[11px] px-2.5 py-1 rounded-full border font-medium transition-colors shrink-0',
                  channelFilter === ch
                    ? cn(info.bg, info.color, 'border-current')
                    : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100',
                )}
              >
                {ch === 'L1_RECOVERY' ? 'L1' : ch} — {info.name} ({count})
              </button>
            );
          })}
          {allSpansLoading && (
            <span className="flex items-center gap-1.5 ml-2 shrink-0">
              <Loader2 className="h-3.5 w-3.5 animate-spin text-gray-400" />
              <span className="text-[10px] text-gray-400">Loading {allSpans.length}…</span>
            </span>
          )}
        </div>
      )}

      {/* ── Split Content (tasks + all modes only) ─────────────────── */}
      {viewMode !== 'pages' && <div className="flex-1 flex overflow-hidden">
        {/* LEFT — Fact Card (48%) */}
        <div className="w-[48%] flex flex-col px-6 pt-5 pb-4 overflow-y-auto">
          {(spanLoading || (viewMode === 'all' && allSpansLoading)) ? (
            <div className="flex-1 flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-gray-300" />
            </div>
          ) : viewMode === 'all' && filteredAllSpans.length === 0 ? (
            <div className="flex-1 flex items-center justify-center text-gray-400 text-sm text-center">
              {allSpansLoading ? 'Loading all extractions...' : channelFilter ? `No extractions for channel ${channelFilter}` : 'No extractions found'}
            </div>
          ) : !span ? (
            <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
              No span data available
            </div>
          ) : (
            <>
              {/* ── Meta Row ──────────────────────────────────────── */}
              <div className="flex items-center gap-2 mb-3 flex-wrap">
                {/* Task type / channel badge */}
                <span
                  className={cn(
                    'text-[10px] font-bold px-2 py-0.5 rounded tracking-wider',
                    effectiveTaskType === 'L1_RECOVERY'
                      ? 'bg-orange-100 text-orange-800 border border-orange-300'
                      : effectiveTaskType === 'DISAGREEMENT'
                        ? 'bg-purple-100 text-purple-800 border border-purple-300'
                        : 'bg-gray-100 text-gray-600 border border-gray-200',
                  )}
                >
                  {effectiveTaskType === 'L1_RECOVERY'
                    ? 'L1 RECOVERY'
                    : effectiveTaskType === 'DISAGREEMENT'
                      ? 'DISAGREEMENT'
                      : viewMode === 'all' ? 'EXTRACTION' : (currentTask?.taskType ?? '—')}
                </span>
                {span.tier != null && (
                  <span
                    className={cn(
                      'text-[10px] font-bold px-2 py-0.5 rounded tracking-wider',
                      tierBg,
                    )}
                  >
                    TIER {span.tier}
                  </span>
                )}
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  § {span.sectionId || '—'}
                </span>
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  p. {span.pageNumber ?? '—'}
                </span>
                <span className="text-[11px] text-indigo-700 bg-indigo-50 px-2 py-0.5 rounded font-medium">
                  Ch {channelLabel}
                </span>
                <span
                  className={cn(
                    'text-[11px] px-2 py-0.5 rounded font-semibold',
                    scoreColor,
                  )}
                >
                  {corroboration.toFixed(1)}
                </span>
                {/* Review status indicator for already-reviewed spans */}
                {span.reviewStatus !== 'PENDING' && (
                  <span
                    className={cn(
                      'text-[10px] font-semibold px-2 py-0.5 rounded-full border',
                      span.reviewStatus === 'CONFIRMED' && 'bg-green-50 text-green-700 border-green-200',
                      span.reviewStatus === 'REJECTED' && 'bg-red-50 text-red-700 border-red-200',
                      span.reviewStatus === 'EDITED' && 'bg-blue-50 text-blue-700 border-blue-200',
                    )}
                  >
                    {span.reviewStatus}
                  </span>
                )}
              </div>

              {/* ── Alert Banner ──────────────────────────────────── */}
              {span.coverageGuardAlert && (
                <div className="mb-4">
                  <CoverageGuardAlertBanner alert={span.coverageGuardAlert} />
                </div>
              )}

              {/* ── Fact Text Card ────────────────────────────────── */}
              {editMode ? (
                <div className="bg-white border border-blue-300 rounded-lg p-5 mb-4 flex-1">
                  <label className="block text-xs font-semibold text-blue-700 uppercase tracking-wide mb-2">
                    Editing Fact Text
                  </label>
                  <textarea
                    value={editedText}
                    onChange={(e) => setEditedText(e.target.value)}
                    rows={6}
                    className="w-full border border-gray-200 rounded-md p-3 text-[15px] leading-[1.75] text-gray-900 font-serif resize-none focus:outline-none focus:ring-2 focus:ring-blue-300"
                    autoFocus
                  />
                </div>
              ) : (
                <div className="bg-white border border-gray-200 rounded-lg p-5 mb-4 flex-1 leading-[1.75] text-[15px] text-gray-900 font-serif">
                  <SemanticText
                    text={span.text}
                    tokens={span.semanticTokens}
                  />
                </div>
              )}

              {/* ── Channel Confidence Breakdown (Browse All mode) ── */}
              {viewMode === 'all' && span.contributingChannels.length > 0 && (
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 mb-4">
                  <div className="text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-2">
                    Channel Breakdown
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {span.contributingChannels.map((ch) => {
                      const info = getChannelInfo(ch);
                      const conf = span.channelConfidences?.[ch];
                      return (
                        <div
                          key={ch}
                          className={cn(
                            'flex items-center gap-1.5 text-[11px] px-2.5 py-1.5 rounded-md border',
                            info.bg,
                            info.color,
                          )}
                        >
                          <span className="font-semibold">{ch === 'L1_RECOVERY' ? 'L1' : ch}</span>
                          <span className="font-normal opacity-80">{info.name}</span>
                          {conf != null && (
                            <span className="font-bold ml-1">{(conf * 100).toFixed(0)}%</span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* ── Reviewer Notes ────────────────────────────────── */}
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Reviewer notes (optional)..."
                className="w-full h-12 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 bg-[#FAFBFC] resize-none mb-4 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />

              {/* ── Alert-gated warning ──────────────────────────── */}
              {alertBlocksConfirm && (
                <div className="flex items-start gap-2 p-2.5 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700 mb-3">
                  <ShieldAlert className="h-4 w-4 text-red-500 mt-0.5 shrink-0" />
                  <span>
                    <strong>Confirm blocked</strong> — this fact has a critical CoverageGuard alert.
                    Review the alert, then Edit or Reject.
                  </span>
                </div>
              )}

              {/* ── Error ─────────────────────────────────────────── */}
              {error && (
                <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-red-700 text-xs mb-3">
                  {error}
                </div>
              )}

              {/* ── Action Bar ────────────────────────────────────── */}
              <div className="flex justify-between items-center pt-3 border-t border-gray-200">
                <div className="flex gap-2">
                  {editMode ? (
                    <>
                      <button
                        onClick={() => setEditMode(false)}
                        disabled={actionLoading}
                        className="px-4 py-2 rounded-md border border-gray-200 text-xs font-medium text-gray-600 hover:bg-gray-50 transition-colors"
                      >
                        Cancel
                      </button>
                      <button
                        onClick={handleEdit}
                        disabled={actionLoading}
                        className="px-5 py-2 rounded-md bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 transition-colors flex items-center gap-1.5 disabled:opacity-50"
                      >
                        {actionLoading && <Loader2 className="h-3 w-3 animate-spin" />}
                        Save Edit
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        onClick={handleConfirm}
                        disabled={alertBlocksConfirm || actionLoading}
                        title={alertBlocksConfirm ? 'Resolve alert before confirming' : 'Confirm (C)'}
                        className={cn(
                          'px-5 py-2 rounded-md text-white text-xs font-semibold flex items-center gap-1.5 transition-colors',
                          alertBlocksConfirm
                            ? 'bg-gray-300 cursor-not-allowed'
                            : 'bg-green-600 hover:bg-green-700',
                        )}
                      >
                        <CheckCircle className="h-3.5 w-3.5" /> Confirm
                      </button>
                      <button
                        onClick={() => {
                          setEditMode(true);
                          setEditedText(span.text);
                        }}
                        disabled={actionLoading}
                        className="px-5 py-2 rounded-md bg-blue-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-blue-700 transition-colors"
                      >
                        <Pencil className="h-3.5 w-3.5" /> Edit
                      </button>
                      <button
                        onClick={() => setShowReject(true)}
                        disabled={actionLoading}
                        className="px-5 py-2 rounded-md bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-red-700 transition-colors"
                      >
                        <XCircle className="h-3.5 w-3.5" /> Reject
                      </button>
                    </>
                  )}
                </div>
                {!editMode && (
                  <div className="flex gap-2">
                    <button
                      onClick={() => setShowAddSpan(true)}
                      className="px-4 py-2 rounded-md bg-purple-600 text-white text-xs font-semibold hover:bg-purple-700 transition-colors flex items-center gap-1.5"
                    >
                      <Plus className="h-3.5 w-3.5" /> Add Fact
                    </button>
                    <button
                      onClick={() => setShowEscalate(true)}
                      disabled={actionLoading}
                      className="px-4 py-2 rounded-md bg-amber-500 text-white text-xs font-semibold hover:bg-amber-600 transition-colors flex items-center gap-1.5"
                    >
                      <ArrowUpRight className="h-3.5 w-3.5" /> Escalate
                    </button>
                  </div>
                )}
              </div>

              {/* ── Navigation ────────────────────────────────────── */}
              <div className="flex justify-between items-center mt-3.5 pt-2.5">
                <button
                  onClick={() => goTo('prev')}
                  disabled={currentIdx === 0}
                  className={cn(
                    'flex items-center gap-1 px-3.5 py-1.5 rounded border text-xs transition-colors',
                    currentIdx === 0
                      ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                      : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700',
                  )}
                >
                  <ChevronLeft className="h-3 w-3" /> Prev
                </button>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-gray-500">
                    {viewMode === 'all' ? 'Extraction' : 'Fact'} {currentIdx + 1} of {totalTasks}
                  </span>
                  {viewMode === 'tasks' && (
                    <span className="text-[10px] text-gray-400">({decidedCount} decided)</span>
                  )}
                </div>
                <button
                  onClick={() => goTo('next')}
                  disabled={viewMode === 'all' && currentIdx >= totalTasks - 1}
                  className={cn(
                    'flex items-center gap-1 px-3.5 py-1.5 rounded border border-gray-200 text-xs transition-colors',
                    viewMode === 'all' && currentIdx >= totalTasks - 1
                      ? 'text-gray-300 cursor-not-allowed'
                      : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700',
                  )}
                >
                  {viewMode === 'tasks' && currentIdx === totalTasks - 1 ? 'Summary' : 'Next'}{' '}
                  <ChevronRight className="h-3 w-3" />
                </button>
              </div>

              {/* ── Keyboard hint ─────────────────────────────────── */}
              <div className="text-center mt-2 text-[10px] text-gray-300 tracking-wide">
                C confirm · E edit · R reject · ← → navigate
              </div>
            </>
          )}
        </div>

        {/* RIGHT — PDF Panel (52%) */}
        <div className="flex-1 bg-gray-50 border-l border-gray-200 flex flex-col overflow-hidden">
          {/* PDF header */}
          <div className="px-5 py-2.5 border-b border-gray-200 bg-white shrink-0 flex justify-between items-center">
            <span className="text-xs font-semibold text-[#1B3A5C]">
              Source PDF{(viewMode === 'tasks' ? (currentTask?.pageNumber ?? span?.pageNumber) : span?.pageNumber) ? ` — Page ${viewMode === 'tasks' ? (currentTask?.pageNumber ?? span?.pageNumber) : span?.pageNumber}` : ''}
            </span>
            <span className="text-[10px] text-gray-400">
              {span?.bbox ? 'Pipeline bbox overlay' : span?.pageNumber ? 'Auto-scrolled to match' : 'Manual page navigation'}
            </span>
          </div>
          {/* PDF viewer — react-pdf with text highlighting */}
          <div className="flex-1 min-h-0">
            <PdfErrorBoundary>
              <PdfHighlightViewer
                jobId={jobId}
                page={viewMode === 'tasks' ? (currentTask?.pageNumber ?? span?.pageNumber ?? undefined) : (span?.pageNumber ?? undefined)}
                highlightText={span?.text}
                useBbox={true}
                pdfBbox={span?.bbox}
              />
            </PdfErrorBoundary>
          </div>
        </div>
      </div>}

      {/* ── Reject Modal (tasks/all modes) ────────────────────────────── */}
      {viewMode !== 'pages' && showReject && (
        <RejectModal
          onClose={() => setShowReject(false)}
          onConfirm={handleRejectConfirm}
          isLoading={actionLoading}
        />
      )}

      {/* ── Escalate Confirmation Modal (tasks/all modes) ─────────────── */}
      {viewMode !== 'pages' && showEscalate && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
          onClick={() => setShowEscalate(false)}
        >
          <div
            className="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 mb-3">
              <ArrowUpRight className="h-5 w-5 text-amber-600" />
              <h3 className="text-lg font-semibold text-gray-900">Escalate to SME</h3>
            </div>
            <p className="text-sm text-gray-600 mb-5">
              This fact will be flagged for review by a subject matter expert.
              It will be removed from your review queue.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowEscalate(false)}
                className="px-4 py-2 rounded-lg border border-gray-200 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleEscalateConfirm}
                disabled={actionLoading}
                className="px-4 py-2 rounded-lg bg-amber-500 text-white text-sm font-semibold hover:bg-amber-600 transition-colors flex items-center gap-1.5 disabled:opacity-50"
              >
                {actionLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                Confirm Escalation
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Add Span Modal (tasks/all modes) ──────────────────────────── */}
      {viewMode !== 'pages' && showAddSpan && (
        <AddSpanModal
          jobId={jobId}
          defaultPageNumber={currentSpan?.pageNumber ?? undefined}
          onClose={() => setShowAddSpan(false)}
          onSuccess={() => {
            invalidateAll();
            onActionComplete();
          }}
        />
      )}
    </div>
  );
}
