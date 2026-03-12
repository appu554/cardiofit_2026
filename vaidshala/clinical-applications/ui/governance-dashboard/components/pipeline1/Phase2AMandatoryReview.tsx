'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ChevronLeft,
  ChevronRight,
  CheckCircle,
  Pencil,
  XCircle,
  ShieldAlert,
  AlertTriangle,
  Loader2,
} from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { useEditDistance, getEditSeverity, EDIT_SEVERITY_CONFIG } from '@/hooks/useEditDistance';
import { SemanticText } from './SemanticText';
import { CoverageGuardAlertBanner } from './CoverageGuardAlert';
import { RejectModal } from './RejectModal';
import { ReviewShell } from './ReviewShell';
import { cn } from '@/lib/utils';
import { getChannelInfo, getChannelName, getConfidenceColor } from '@/lib/pipeline1-channels';

// =============================================================================
// Disagreement Detail Parser
// =============================================================================

interface ChannelEntry {
  channel: string;
  text: string;
}

/**
 * Parse the disagreement_detail string from the Signal Merger.
 * Format: "B:'text B saw' vs C:'text C saw' vs F:'text F saw'"
 * Also handles single-channel: "F:'text F saw'"
 */
function parseDisagreementDetail(detail: string): ChannelEntry[] {
  const entries: ChannelEntry[] = [];
  // Match patterns like: B:'some text' or F:'some text'
  const regex = /([A-Z][A-Z0-9_]*):'((?:[^'\\]|\\.)*)'/g;
  let match;
  while ((match = regex.exec(detail)) !== null) {
    entries.push({ channel: match[1], text: match[2] });
  }
  return entries;
}

/**
 * Determine a human-readable disagreement reason from channel data.
 */
function getDisagreementReason(
  channels: string[],
  entries: ChannelEntry[],
): { reason: string; severity: 'info' | 'warning' | 'critical' } {
  if (channels.length === 1) {
    const ch = channels[0];
    if (ch === 'F') {
      return {
        reason: 'LLM-only extraction — no structural channel confirmed this text. Verify against source PDF.',
        severity: 'warning',
      };
    }
    return {
      reason: `Single-channel extraction — only ${getChannelName(ch)} (${ch}) found this text.`,
      severity: 'info',
    };
  }

  // Check if channels extracted different key terms
  const uniqueTexts = new Set(entries.map((e) => e.text.trim()));
  if (uniqueTexts.size > 1) {
    return {
      reason: `${channels.length} channels extracted different content at this location. Compare each channel's extraction below.`,
      severity: 'critical',
    };
  }

  return {
    reason: `Flagged for review — ${channels.length} channels contributed but content may differ in context.`,
    severity: 'warning',
  };
}
import type {
  MergedSpan,
  ReviewTask,
  ExtractionJob,
  SpanReviewRequest,
  RejectReason,
  Tier1Checklist,
} from '@/types/pipeline1';
import { TIER1_CHECKLIST_LABELS } from '@/types/pipeline1';

// =============================================================================
// Props
// =============================================================================

interface Phase2AMandatoryReviewProps {
  jobId: string;
  job: ExtractionJob;
  onActionComplete: () => void;
  onPhaseComplete: () => void;
}

// =============================================================================
// Constants
// =============================================================================

const INITIAL_CHECKLIST: Tier1Checklist = {
  textMatchesSource: false,
  numericsVerified: false,
  negationsPreserved: false,
  scopeCorrect: false,
  noOmissions: false,
};

const CHECKLIST_KEYS = Object.keys(INITIAL_CHECKLIST) as (keyof Tier1Checklist)[];

// =============================================================================
// Phase2AMandatoryReview — Tier 1 Mandatory Review
//
// Task queue for DISAGREEMENT + PASSAGE_SPOT_CHECK tasks only.
// Sequential review with 5-point checklist gating and source-constrained edits.
// =============================================================================

export function Phase2AMandatoryReview({
  jobId,
  job,
  onActionComplete,
  onPhaseComplete,
}: Phase2AMandatoryReviewProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  // -- Data fetching: review tasks ----------------------------------------
  const {
    data: allTasks = [],
    isLoading: tasksLoading,
  } = useQuery({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
  });

  // Filter to mandatory review types only
  const tasks = useMemo(
    () =>
      allTasks.filter(
        (t) =>
          t.taskType === 'DISAGREEMENT' || t.taskType === 'PASSAGE_SPOT_CHECK',
      ),
    [allTasks],
  );

  // -- State ---------------------------------------------------------------
  const [currentIdx, setCurrentIdx] = useState(0);
  const [currentSpan, setCurrentSpan] = useState<MergedSpan | null>(null);
  const [spanLoading, setSpanLoading] = useState(false);
  const [checklist, setChecklist] = useState<Tier1Checklist>(INITIAL_CHECKLIST);
  const [editMode, setEditMode] = useState(false);
  const [editedText, setEditedText] = useState('');
  const [note, setNote] = useState('');
  const [showReject, setShowReject] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [resolvedIds, setResolvedIds] = useState<Set<string>>(new Set());

  const currentTask = tasks[currentIdx] ?? null;
  const totalTasks = tasks.length;
  const resolvedCount = tasks.filter(
    (t) => t.status === 'RESOLVED' || resolvedIds.has(t.id),
  ).length;
  const allResolved = totalTasks > 0 && resolvedCount >= totalTasks;

  // -- Derived: alert-gated confirm ----------------------------------------
  const hasCriticalAlert =
    currentSpan?.coverageGuardAlert?.alertSeverity === 'critical';
  const alertBlocksConfirm =
    currentSpan?.coverageGuardAlert?.type === 'numeric_mismatch' ||
    currentSpan?.coverageGuardAlert?.type === 'branch_loss';

  // -- Derived: checklist complete -----------------------------------------
  const checklistComplete = CHECKLIST_KEYS.every((k) => checklist[k]);
  const confirmEnabled = checklistComplete && !alertBlocksConfirm && !actionLoading;

  // -- Edit distance tracking ----------------------------------------------
  const editDistanceInfo = useEditDistance(
    currentSpan?.text ?? '',
    editedText,
  );
  const editSeverity = getEditSeverity(editDistanceInfo.changePercentage);
  const severityStyle = EDIT_SEVERITY_CONFIG[editSeverity];

  // -- Fetch span for current task -----------------------------------------
  useEffect(() => {
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
      })
      .catch(() => {
        setCurrentSpan(null);
        setSpanLoading(false);
      });
  }, [jobId, currentTask?.spanId, currentTask?.id]);

  // -- Reset form state on navigation --------------------------------------
  useEffect(() => {
    setChecklist(INITIAL_CHECKLIST);
    setEditMode(false);
    setEditedText('');
    setNote('');
    setError(null);
    setShowReject(false);
  }, [currentIdx]);

  // -- Auto-advance: phase complete when all tasks resolved ----------------
  useEffect(() => {
    if (allResolved && totalTasks > 0) {
      onPhaseComplete();
    }
  }, [allResolved, totalTasks, onPhaseComplete]);

  // -- Cache invalidation (same pattern as FactReviewLayout) ---------------
  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans-all', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-review-tasks', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
  }, [queryClient, jobId]);

  // -- Navigation ----------------------------------------------------------
  const advance = useCallback(() => {
    if (currentIdx < totalTasks - 1) {
      setCurrentIdx((i) => i + 1);
    }
  }, [currentIdx, totalTasks]);

  const goTo = useCallback(
    (dir: 'prev' | 'next') => {
      if (dir === 'prev' && currentIdx > 0) {
        setCurrentIdx((i) => i - 1);
      }
      if (dir === 'next' && currentIdx < totalTasks - 1) {
        setCurrentIdx((i) => i + 1);
      }
    },
    [currentIdx, totalTasks],
  );

  // -- Action handlers -----------------------------------------------------
  const markResolved = useCallback(
    (taskId: string) => {
      setResolvedIds((prev) => new Set(prev).add(taskId));
    },
    [],
  );

  const handleConfirm = useCallback(async () => {
    if (!currentSpan || !currentTask) return;
    if (!checklistComplete) return;
    setActionLoading(true);
    setError(null);
    try {
      const req: SpanReviewRequest = {
        reviewerId: user?.sub || 'unknown',
        ...(note ? { note } : {}),
      };
      await pipeline1Api.spans.confirm(jobId, currentSpan.id, req);
      markResolved(currentTask.id);
      invalidateAll();
      onActionComplete();
      advance();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionLoading(false);
    }
  }, [currentSpan, currentTask, user, note, jobId, checklistComplete, invalidateAll, onActionComplete, advance, markResolved]);

  const handleEdit = useCallback(async () => {
    if (!currentSpan || !currentTask || !editedText.trim()) {
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
      markResolved(currentTask.id);
      invalidateAll();
      onActionComplete();
      setEditMode(false);
      advance();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionLoading(false);
    }
  }, [currentSpan, currentTask, user, note, editedText, jobId, invalidateAll, onActionComplete, advance, markResolved]);

  const handleRejectConfirm = useCallback(
    async (reason: RejectReason) => {
      if (!currentSpan || !currentTask) return;
      setActionLoading(true);
      setError(null);
      try {
        const req: SpanReviewRequest = {
          reviewerId: user?.sub || 'unknown',
          rejectReason: reason,
          ...(note ? { note } : {}),
        };
        await pipeline1Api.spans.reject(jobId, currentSpan.id, req);
        markResolved(currentTask.id);
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
    [currentSpan, currentTask, user, note, jobId, invalidateAll, onActionComplete, advance, markResolved],
  );

  // -- Checklist toggle ----------------------------------------------------
  const toggleChecklistItem = useCallback((key: keyof Tier1Checklist) => {
    setChecklist((prev) => ({ ...prev, [key]: !prev[key] }));
  }, []);

  // -- Keyboard shortcuts --------------------------------------------------
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      switch (e.key.toLowerCase()) {
        case 'c':
          if (!editMode && confirmEnabled) handleConfirm();
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
        case 'escape':
          setEditMode(false);
          setShowReject(false);
          break;
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [editMode, confirmEnabled, handleConfirm, goTo, currentSpan]);

  // -- Derived display values ----------------------------------------------
  const span = currentSpan;
  const corroboration = span?.mergedConfidence ?? 0;
  const confidenceColor = getConfidenceColor(corroboration);
  const tierBg =
    span?.tier === 1
      ? 'bg-[#7F1D1D] text-[#FECACA]'
      : span?.tier === 2
        ? 'bg-amber-700 text-amber-100'
        : 'bg-gray-600 text-gray-200';

  // -- Loading state -------------------------------------------------------
  if (tasksLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-gray-300" />
      </div>
    );
  }

  // -- Empty state ---------------------------------------------------------
  if (tasks.length === 0) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <CheckCircle className="h-12 w-12 text-green-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-gray-700">
            No mandatory review tasks
          </h3>
          <p className="text-sm text-gray-500 mt-1">
            No DISAGREEMENT or PASSAGE_SPOT_CHECK tasks for this job.
          </p>
          <button
            onClick={onPhaseComplete}
            className="mt-5 px-5 py-2.5 rounded-lg bg-[#1B3A5C] text-white text-sm font-semibold hover:bg-[#2A5580] transition-colors"
          >
            Continue to Next Phase
          </button>
        </div>
      </div>
    );
  }

  // -- Progress bar width --------------------------------------------------
  const progressPct = totalTasks > 0 ? (resolvedCount / totalTasks) * 100 : 0;

  // -- Top bar with progress and counter -----------------------------------
  const topBar = (
    <div className="px-5 py-3 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h2 className="text-sm font-semibold text-[#1B3A5C] tracking-wide">
          Phase 2A — Tier 1 Mandatory Review
        </h2>
        <span className="text-[10px] text-gray-500 uppercase tracking-wider font-medium">
          Disagreement + Spot-Check
        </span>
      </div>
      <div className="flex items-center gap-4">
        {/* Progress counter */}
        <span className="text-xs text-gray-500 font-medium">
          {resolvedCount}/{totalTasks} resolved
        </span>
        {allResolved && (
          <span className="text-[10px] font-semibold tracking-wider px-2.5 py-1 rounded bg-[#065F46] text-[#A7F3D0]">
            ALL RESOLVED
          </span>
        )}
      </div>
    </div>
  );

  // -- Render: split-panel via ReviewShell ---------------------------------
  return (
    <ReviewShell
      jobId={jobId}
      topBar={topBar}
      pdfPage={currentTask?.pageNumber ?? span?.pageNumber ?? undefined}
      pdfHighlightText={span?.text}
      pdfBbox={span?.bbox}
      useBbox={true}
    >
      <div className="flex flex-col h-full">
        {/* ── Progress Bar ────────────────────────────────────────────── */}
        <div className="px-5 pt-4 pb-2">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-[10px] font-medium text-gray-400 uppercase tracking-wider">
              Progress
            </span>
            <span className="text-[10px] text-gray-400">
              {Math.round(progressPct)}%
            </span>
          </div>
          <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all duration-500 ease-out',
                allResolved ? 'bg-green-500' : 'bg-[#1B3A5C]',
              )}
              style={{ width: `${progressPct}%` }}
            />
          </div>
        </div>

        {/* ── Navigation Bar ──────────────────────────────────────────── */}
        <div className="px-5 py-2.5 flex items-center justify-between border-b border-gray-100">
          <button
            onClick={() => goTo('prev')}
            disabled={currentIdx === 0}
            className={cn(
              'flex items-center gap-1 px-3 py-1.5 rounded border text-xs transition-colors',
              currentIdx === 0
                ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700',
            )}
          >
            <ChevronLeft className="h-3 w-3" /> Prev
          </button>
          <span className="text-xs text-gray-500 font-medium">
            Task {currentIdx + 1} of {totalTasks}
          </span>
          <button
            onClick={() => goTo('next')}
            disabled={currentIdx >= totalTasks - 1}
            className={cn(
              'flex items-center gap-1 px-3 py-1.5 rounded border text-xs transition-colors',
              currentIdx >= totalTasks - 1
                ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700',
            )}
          >
            Next <ChevronRight className="h-3 w-3" />
          </button>
        </div>

        {/* ── Main Content (scrollable) ───────────────────────────────── */}
        <div className="flex-1 overflow-y-auto px-5 pt-4 pb-5">
          {spanLoading ? (
            <div className="flex-1 flex items-center justify-center py-20">
              <Loader2 className="h-8 w-8 animate-spin text-gray-300" />
            </div>
          ) : !span ? (
            <div className="flex-1 flex flex-col items-center justify-center py-20 text-gray-400 text-sm gap-2">
              <span>No span data available for this task</span>
              {currentTask?.taskType === 'PASSAGE_SPOT_CHECK' && (
                <span className="text-xs text-gray-300">
                  This passage section has no linked spans. You may skip it.
                </span>
              )}
            </div>
          ) : (
            <>
              {/* ── Meta Row ──────────────────────────────────────── */}
              <div className="flex items-center gap-2 mb-3 flex-wrap">
                {/* Task type badge */}
                <span
                  className={cn(
                    'text-[10px] font-bold px-2 py-0.5 rounded tracking-wider',
                    currentTask?.taskType === 'DISAGREEMENT'
                      ? 'bg-purple-100 text-purple-800 border border-purple-300'
                      : 'bg-blue-100 text-blue-800 border border-blue-300',
                  )}
                >
                  {currentTask?.taskType === 'DISAGREEMENT'
                    ? 'DISAGREEMENT'
                    : 'PASSAGE SPOT CHECK'}
                </span>
                {/* Tier badge */}
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
                {/* Section */}
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  &sect; {span.sectionId || '--'}
                </span>
                {/* Page */}
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  p. {span.pageNumber ?? '--'}
                </span>
                {/* Channel badges */}
                {span.contributingChannels.map((ch) => {
                  const info = getChannelInfo(ch);
                  return (
                    <span
                      key={ch}
                      className={cn(
                        'text-[10px] font-semibold px-2 py-0.5 rounded',
                        info.bg,
                        info.color,
                      )}
                    >
                      {ch === 'L1_RECOVERY' ? 'L1' : ch}
                    </span>
                  );
                })}
                {/* Confidence score */}
                <span
                  className={cn(
                    'text-[11px] px-2 py-0.5 rounded font-semibold',
                    confidenceColor,
                    corroboration >= 0.7
                      ? 'bg-green-50'
                      : corroboration >= 0.5
                        ? 'bg-amber-50'
                        : 'bg-red-50',
                  )}
                >
                  {corroboration.toFixed(2)}
                </span>
                {/* Already-resolved indicator */}
                {(span.reviewStatus !== 'PENDING' || resolvedIds.has(currentTask?.id ?? '')) && (
                  <span
                    className={cn(
                      'text-[10px] font-semibold px-2 py-0.5 rounded-full border',
                      span.reviewStatus === 'CONFIRMED' && 'bg-green-50 text-green-700 border-green-200',
                      span.reviewStatus === 'REJECTED' && 'bg-red-50 text-red-700 border-red-200',
                      span.reviewStatus === 'EDITED' && 'bg-blue-50 text-blue-700 border-blue-200',
                      span.reviewStatus === 'PENDING' && resolvedIds.has(currentTask?.id ?? '') && 'bg-gray-50 text-gray-600 border-gray-200',
                    )}
                  >
                    {resolvedIds.has(currentTask?.id ?? '') && span.reviewStatus === 'PENDING'
                      ? 'DECIDED'
                      : span.reviewStatus}
                  </span>
                )}
              </div>

              {/* ── Task Description ──────────────────────────────── */}
              {currentTask?.description && (
                <div className="text-xs text-gray-600 bg-gray-50 border border-gray-200 rounded-md px-3 py-2 mb-3 leading-relaxed">
                  {currentTask.description}
                </div>
              )}

              {/* ── Disagreement Breakdown ─────────────────────────── */}
              {(span.hasDisagreement || currentTask?.taskType === 'DISAGREEMENT') && (
                (() => {
                  const entries = span.disagreementDetail
                    ? parseDisagreementDetail(span.disagreementDetail)
                    : [];
                  const { reason, severity } = getDisagreementReason(
                    span.contributingChannels,
                    entries,
                  );
                  const severityStyles = {
                    info: { border: 'border-blue-300', bg: 'bg-blue-50', headerBg: 'bg-blue-100', text: 'text-blue-800', icon: 'text-blue-600' },
                    warning: { border: 'border-amber-300', bg: 'bg-amber-50', headerBg: 'bg-amber-100', text: 'text-amber-800', icon: 'text-amber-600' },
                    critical: { border: 'border-red-300', bg: 'bg-red-50', headerBg: 'bg-red-100', text: 'text-red-800', icon: 'text-red-600' },
                  }[severity];

                  return (
                    <div className={cn('rounded-lg border overflow-hidden mb-4', severityStyles.border, severityStyles.bg)}>
                      {/* Header */}
                      <div className={cn('flex items-center gap-2 px-3 py-2 border-b', severityStyles.headerBg, severityStyles.border)}>
                        <AlertTriangle className={cn('h-4 w-4 flex-shrink-0', severityStyles.icon)} />
                        <span className={cn('text-xs font-semibold uppercase tracking-wide', severityStyles.text)}>
                          Why Flagged
                        </span>
                        <span className={cn('text-[10px] ml-auto font-medium px-2 py-0.5 rounded', severityStyles.text, severityStyles.headerBg)}>
                          {span.contributingChannels.length === 1
                            ? 'SINGLE CHANNEL'
                            : `${span.contributingChannels.length} CHANNELS`}
                        </span>
                      </div>

                      {/* Reason */}
                      <div className="px-3 py-2.5">
                        <p className={cn('text-xs leading-relaxed', severityStyles.text)}>
                          {reason}
                        </p>
                      </div>

                      {/* Fact text — displayed inside Why Flagged for context */}
                      <div className="px-3 pb-3">
                        <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-2">
                          Extracted Fact Text
                        </p>
                        <div className={cn(
                          'rounded-md p-3 leading-[1.75] text-[15px] text-gray-900 font-serif',
                          span.reviewerText && span.reviewStatus === 'EDITED'
                            ? 'bg-blue-50/50 border border-blue-200'
                            : 'bg-white border border-gray-200',
                        )}>
                          {span.reviewerText && span.reviewStatus === 'EDITED' && (
                            <div className="flex items-center gap-1.5 mb-2 text-[10px] font-semibold text-blue-600 uppercase tracking-wide">
                              <Pencil className="h-3 w-3" />
                              Edited Text
                            </div>
                          )}
                          <SemanticText
                            text={span.reviewerText && span.reviewStatus === 'EDITED' ? span.reviewerText : span.text}
                            tokens={span.reviewStatus === 'EDITED' ? undefined : span.semanticTokens}
                          />
                          {span.reviewerText && span.reviewStatus === 'EDITED' && (
                            <details className="mt-3 text-xs">
                              <summary className="text-gray-400 cursor-pointer hover:text-gray-600">
                                Show original text
                              </summary>
                              <div className="mt-1.5 text-gray-500 line-through text-[13px] leading-relaxed">
                                {span.text}
                              </div>
                            </details>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })()
              )}

              {/* ── Fact Text for non-disagreement tasks ─────────── */}
              {!editMode && !(span.hasDisagreement || currentTask?.taskType === 'DISAGREEMENT') && (
                <div className={cn(
                  'rounded-lg p-5 mb-4 leading-[1.75] text-[15px] text-gray-900 font-serif',
                  span.reviewerText && span.reviewStatus === 'EDITED'
                    ? 'bg-blue-50/50 border border-blue-200'
                    : 'bg-white border border-gray-200',
                )}>
                  {span.reviewerText && span.reviewStatus === 'EDITED' && (
                    <div className="flex items-center gap-1.5 mb-2 text-[10px] font-semibold text-blue-600 uppercase tracking-wide">
                      <Pencil className="h-3 w-3" />
                      Edited Text
                    </div>
                  )}
                  <SemanticText
                    text={span.reviewerText && span.reviewStatus === 'EDITED' ? span.reviewerText : span.text}
                    tokens={span.reviewStatus === 'EDITED' ? undefined : span.semanticTokens}
                  />
                  {span.reviewerText && span.reviewStatus === 'EDITED' && (
                    <details className="mt-3 text-xs">
                      <summary className="text-gray-400 cursor-pointer hover:text-gray-600">
                        Show original text
                      </summary>
                      <div className="mt-1.5 text-gray-500 line-through text-[13px] leading-relaxed">
                        {span.text}
                      </div>
                    </details>
                  )}
                </div>
              )}

              {/* ── CoverageGuard Alert Banner ────────────────────── */}
              {span.coverageGuardAlert && (
                <div className="mb-4">
                  <CoverageGuardAlertBanner alert={span.coverageGuardAlert} />
                </div>
              )}

              {/* ── Edit Mode (replaces fact text in Why Flagged box) ─ */}
              {editMode && (
                <div className="bg-white border border-blue-300 rounded-lg p-5 mb-4">
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

                  {/* Edit distance warning bar */}
                  {editedText && editedText !== (span?.text ?? '') && (
                    <div
                      className={cn(
                        'mt-3 flex items-center gap-2 px-3 py-2 rounded-md border text-xs',
                        severityStyle.bg,
                        severityStyle.border,
                        severityStyle.color,
                      )}
                    >
                      <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
                      <span className="font-medium">
                        {editDistanceInfo.changePercentage.toFixed(1)}% changed
                      </span>
                      <span className="mx-1 opacity-40">|</span>
                      <span>{severityStyle.label}</span>
                    </div>
                  )}
                </div>
              )}

              {/* ── 5-Point Tier 1 Checklist ──────────────────────── */}
              {!editMode && (
                <div className="bg-[#FAFBFC] border border-gray-200 rounded-lg p-4 mb-4">
                  <div className="flex items-center gap-2 mb-3">
                    <ShieldAlert className="h-4 w-4 text-[#1B3A5C]" />
                    <span className="text-xs font-semibold text-[#1B3A5C] uppercase tracking-wide">
                      Tier 1 Verification Checklist
                    </span>
                    <span className="text-[10px] text-gray-400 ml-auto">
                      {CHECKLIST_KEYS.filter((k) => checklist[k]).length}/5 checked
                    </span>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    {CHECKLIST_KEYS.map((key) => (
                      <label
                        key={key}
                        className={cn(
                          'flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors text-[13px]',
                          checklist[key]
                            ? 'bg-green-50 border border-green-200 text-green-800'
                            : 'bg-white border border-gray-200 text-gray-700 hover:bg-gray-50',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={checklist[key]}
                          onChange={() => toggleChecklistItem(key)}
                          className="accent-green-600 h-3.5 w-3.5 flex-shrink-0"
                        />
                        <span className={cn(checklist[key] && 'font-medium')}>
                          {TIER1_CHECKLIST_LABELS[key]}
                        </span>
                        {checklist[key] && (
                          <CheckCircle className="h-3.5 w-3.5 text-green-600 ml-auto flex-shrink-0" />
                        )}
                      </label>
                    ))}
                  </div>
                  {!checklistComplete && (
                    <p className="text-[10px] text-gray-400 mt-2 pl-1">
                      All 5 items must be checked before confirming.
                    </p>
                  )}
                </div>
              )}

              {/* ── Reviewer Notes ────────────────────────────────── */}
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Reviewer notes (optional)..."
                className="w-full h-12 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 bg-[#FAFBFC] resize-none mb-4 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />

              {/* ── Alert-gated warning ────────────────────────────── */}
              {alertBlocksConfirm && (
                <div className="flex items-start gap-2 p-2.5 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700 mb-3">
                  <ShieldAlert className="h-4 w-4 text-red-500 mt-0.5 shrink-0" />
                  <span>
                    <strong>Confirm blocked</strong> -- this fact has a critical CoverageGuard alert
                    ({currentSpan?.coverageGuardAlert?.type === 'numeric_mismatch'
                      ? 'numeric mismatch'
                      : 'branch loss'}).
                    Review the alert, then Edit or Reject.
                  </span>
                </div>
              )}

              {/* ── Error ──────────────────────────────────────────── */}
              {error && (
                <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-red-700 text-xs mb-3">
                  {error}
                </div>
              )}

              {/* ── Action Bar ─────────────────────────────────────── */}
              <div className="flex items-center gap-2 pt-3 border-t border-gray-200">
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
                      disabled={actionLoading || !editedText.trim()}
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
                      disabled={!confirmEnabled}
                      title={
                        alertBlocksConfirm
                          ? 'Resolve alert before confirming'
                          : !checklistComplete
                            ? 'Complete all checklist items first'
                            : 'Confirm (C)'
                      }
                      className={cn(
                        'px-5 py-2 rounded-md text-white text-xs font-semibold flex items-center gap-1.5 transition-colors',
                        confirmEnabled
                          ? 'bg-green-600 hover:bg-green-700'
                          : 'bg-gray-300 cursor-not-allowed',
                      )}
                    >
                      <CheckCircle className="h-3.5 w-3.5" /> Confirm
                    </button>
                    <button
                      onClick={() => {
                        if (span) {
                          setEditMode(true);
                          setEditedText(span.reviewerText && span.reviewStatus === 'EDITED' ? span.reviewerText : span.text);
                        }
                      }}
                      disabled={actionLoading}
                      className="px-5 py-2 rounded-md bg-blue-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-blue-700 transition-colors disabled:opacity-50"
                    >
                      <Pencil className="h-3.5 w-3.5" /> Edit
                    </button>
                    <button
                      onClick={() => setShowReject(true)}
                      disabled={actionLoading}
                      className="px-5 py-2 rounded-md bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-red-700 transition-colors disabled:opacity-50"
                    >
                      <XCircle className="h-3.5 w-3.5" /> Reject
                    </button>
                  </>
                )}
              </div>

              {/* ── Keyboard hint ──────────────────────────────────── */}
              <div className="text-center mt-3 text-[10px] text-gray-300 tracking-wide">
                C confirm &middot; E edit &middot; R reject &middot; &larr; &rarr; navigate
              </div>
            </>
          )}
        </div>
      </div>

      {/* ── Reject Modal ──────────────────────────────────────────────── */}
      {showReject && (
        <RejectModal
          onClose={() => setShowReject(false)}
          onConfirm={handleRejectConfirm}
          isLoading={actionLoading}
        />
      )}
    </ReviewShell>
  );
}
