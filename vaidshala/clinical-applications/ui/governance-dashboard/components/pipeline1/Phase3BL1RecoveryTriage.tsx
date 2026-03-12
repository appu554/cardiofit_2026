'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, AlertTriangle, Loader2, Eye } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { useEditDistance, getEditSeverity, EDIT_SEVERITY_CONFIG } from '@/hooks/useEditDistance';
import { SemanticText } from './SemanticText';
import { RejectModal } from './RejectModal';
import { ReviewShell } from './ReviewShell';
import { cn } from '@/lib/utils';
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import type { MergedSpan, ReviewTask, ExtractionJob, SpanReviewRequest, RejectReason } from '@/types/pipeline1';

// =============================================================================
// Phase 3B — L1 Recovery Triage
//
// Reviews L1_RECOVERY tasks where OCR recovery spans need visual verification
// against the source PDF. These spans were recovered from vector-drawn
// tables/figures using PyMuPDF OCR — the primary verification method is
// visual comparison against the rendered PDF.
//
// Layout: ReviewShell with useBbox={true} for stored bbox overlay
// Workflow: visual comparison → confirm/edit/reject → auto-advance
// =============================================================================

// -----------------------------------------------------------------------------
// Props
// -----------------------------------------------------------------------------

interface Phase3BL1RecoveryTriageProps {
  jobId: string;
  job: ExtractionJob;
  onActionComplete: () => void;
  onPhaseComplete: () => void;
}

// =============================================================================
// Component
// =============================================================================

export function Phase3BL1RecoveryTriage({
  jobId,
  job,
  onActionComplete,
  onPhaseComplete,
}: Phase3BL1RecoveryTriageProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------
  const [currentIdx, setCurrentIdx] = useState(0);
  const [editMode, setEditMode] = useState(false);
  const [editedText, setEditedText] = useState('');
  const [note, setNote] = useState('');
  const [showReject, setShowReject] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentSpan, setCurrentSpan] = useState<MergedSpan | null>(null);
  const [spanLoading, setSpanLoading] = useState(false);

  // ---------------------------------------------------------------------------
  // Data fetching: review tasks filtered to L1_RECOVERY
  // ---------------------------------------------------------------------------
  const { data: allTasks, isLoading: tasksLoading } = useQuery<ReviewTask[]>({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
    enabled: !!jobId,
  });

  const l1Tasks = useMemo(() => {
    if (!allTasks) return [];
    return allTasks.filter((t) => t.taskType === 'L1_RECOVERY');
  }, [allTasks]);

  const pendingTasks = useMemo(
    () => l1Tasks.filter((t) => t.status !== 'RESOLVED'),
    [l1Tasks],
  );

  const currentTask: ReviewTask | null = l1Tasks[currentIdx] ?? null;
  const totalTasks = l1Tasks.length;

  // ---------------------------------------------------------------------------
  // Fetch span for current task
  // ---------------------------------------------------------------------------
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

  // ---------------------------------------------------------------------------
  // Edit distance tracking (source-constrained edits)
  // ---------------------------------------------------------------------------
  const editDistanceInfo = useEditDistance(currentSpan?.text ?? '', editedText);
  const editSeverity = editMode ? getEditSeverity(editDistanceInfo.changePercentage) : null;
  const editSeverityConfig = editSeverity ? EDIT_SEVERITY_CONFIG[editSeverity] : null;

  // ---------------------------------------------------------------------------
  // Reset form state when navigating
  // ---------------------------------------------------------------------------
  useEffect(() => {
    setEditMode(false);
    setEditedText('');
    setNote('');
    setError(null);
  }, [currentIdx]);

  // ---------------------------------------------------------------------------
  // Auto-complete: when all L1_RECOVERY tasks are RESOLVED
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!tasksLoading && l1Tasks.length > 0 && pendingTasks.length === 0) {
      onPhaseComplete();
    }
  }, [tasksLoading, l1Tasks.length, pendingTasks.length, onPhaseComplete]);

  // ---------------------------------------------------------------------------
  // Cache invalidation
  // ---------------------------------------------------------------------------
  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-review-tasks', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
  }, [queryClient, jobId]);

  // ---------------------------------------------------------------------------
  // Navigation
  // ---------------------------------------------------------------------------
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

  // ---------------------------------------------------------------------------
  // Action handlers
  // ---------------------------------------------------------------------------
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

  // ---------------------------------------------------------------------------
  // Keyboard shortcuts: C/E/R + arrow keys
  // ---------------------------------------------------------------------------
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      switch (e.key.toLowerCase()) {
        case 'c':
          if (!editMode) handleConfirm();
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
  }, [editMode, handleConfirm, goTo, currentSpan]);

  // ---------------------------------------------------------------------------
  // Loading state
  // ---------------------------------------------------------------------------
  if (tasksLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-red-400" />
          <p className="text-sm text-gray-500">Loading L1 Recovery tasks...</p>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Empty state: no L1_RECOVERY tasks
  // ---------------------------------------------------------------------------
  if (l1Tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center px-8">
          <Eye className="h-12 w-12 text-red-300 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-gray-700 mb-2">
            No L1 Recovery Tasks
          </h3>
          <p className="text-sm text-gray-500 max-w-md">
            No OCR recovery spans require visual verification in this job.
            This phase requires no action.
          </p>
          <button
            onClick={onPhaseComplete}
            className="mt-6 px-6 py-2.5 rounded-lg bg-red-600 text-white text-sm font-semibold hover:bg-red-700 transition-colors"
          >
            Continue to Next Phase
          </button>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Derived display values
  // ---------------------------------------------------------------------------
  const span = currentSpan;
  const isTaskResolved = currentTask?.status === 'RESOLVED';
  const isPending = span?.reviewStatus === 'PENDING';

  // Format bbox coordinates for display
  const bboxLabel = span?.bbox
    ? `[${span.bbox.map((v) => v.toFixed(1)).join(', ')}]`
    : null;

  // ---------------------------------------------------------------------------
  // Render: ReviewShell with bbox overlay and left panel content
  // ---------------------------------------------------------------------------
  return (
    <>
      <ReviewShell
        jobId={jobId}
        pdfPage={currentTask?.pageNumber ?? span?.pageNumber ?? undefined}
        pdfHighlightText={span?.text}
        pdfBbox={span?.bbox}
        useBbox={true}
        topBar={
          <div className="flex items-center justify-between px-6 py-2.5">
            <div className="flex items-center gap-3">
              <Eye className="h-4 w-4 text-red-600" />
              <span className="text-sm font-semibold text-gray-900">
                Phase 3B: L1 Recovery Triage
              </span>
              <span className="text-xs text-gray-400">|</span>
              <span className="text-xs text-gray-500">
                OCR recovery visual verification
              </span>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-xs text-gray-500">
                {pendingTasks.length} pending of {totalTasks}
              </span>
            </div>
          </div>
        }
      >
        <div className="px-6 pt-5 pb-4 space-y-4">
          {/* ── Loading span ────────────────────────────────────────────── */}
          {spanLoading && (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-gray-300" />
            </div>
          )}

          {!spanLoading && !span && (
            <div className="flex items-center justify-center py-16 text-gray-400 text-sm">
              No span data available for this task
            </div>
          )}

          {!spanLoading && span && (
            <>
              {/* ── L1 Recovery Context Panel ──────────────────────────── */}
              <div className="rounded-lg border border-red-300 bg-red-50 overflow-hidden">
                <div className="flex items-center gap-2 px-4 py-2.5 bg-red-100 border-b border-red-200">
                  <AlertTriangle className="h-4 w-4 text-red-700 shrink-0" />
                  <span className="text-xs font-bold text-red-800 uppercase tracking-wider">
                    L1 Recovery — OCR Extraction
                  </span>
                  {isTaskResolved && (
                    <span className="ml-auto text-[10px] font-semibold text-green-700 bg-green-100 px-2 py-0.5 rounded-full">
                      RESOLVED
                    </span>
                  )}
                </div>
                <div className="px-4 py-3 space-y-3">
                  <p className="text-xs text-red-800 leading-relaxed">
                    This text was recovered from a vector-drawn table/figure using PyMuPDF
                    OCR. Visual verification against the source PDF is required.
                  </p>

                  {/* Bbox coordinates */}
                  {bboxLabel && (
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] font-semibold text-red-600 uppercase tracking-wide">
                        Bbox:
                      </span>
                      <code className="text-[11px] font-mono text-red-700 bg-red-100 px-2 py-0.5 rounded">
                        {bboxLabel}
                      </code>
                    </div>
                  )}

                  {/* Surrounding context */}
                  {span.surroundingContext && (
                    <div className="rounded border border-red-200 bg-white p-3">
                      <p className="text-[10px] font-semibold text-red-500 uppercase tracking-wide mb-1.5">
                        Surrounding Context
                      </p>
                      <p className="text-xs text-gray-700 leading-relaxed whitespace-pre-wrap">
                        {span.surroundingContext}
                      </p>
                    </div>
                  )}
                </div>
              </div>

              {/* ── Meta Row ────────────────────────────────────────────── */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="inline-flex items-center text-[10px] font-bold px-2 py-0.5 rounded tracking-wider bg-red-100 text-red-800 border border-red-300">
                  <AlertTriangle className="h-3 w-3 mr-1" />
                  L1 RECOVERY
                </span>
                {span.tier != null && (
                  <span
                    className={cn(
                      'text-[10px] font-bold px-2 py-0.5 rounded tracking-wider',
                      span.tier === 1 && 'bg-red-900 text-red-100',
                      span.tier === 2 && 'bg-amber-700 text-amber-100',
                      span.tier === 3 && 'bg-gray-600 text-gray-200',
                    )}
                  >
                    TIER {span.tier}
                  </span>
                )}
                {currentTask?.severity && (
                  <span
                    className={cn(
                      'text-[10px] font-bold px-2 py-0.5 rounded-full',
                      currentTask.severity === 'critical' && 'bg-red-100 text-red-700',
                      currentTask.severity === 'warning' && 'bg-amber-100 text-amber-700',
                      currentTask.severity === 'info' && 'bg-blue-100 text-blue-700',
                    )}
                  >
                    {currentTask.severity.toUpperCase()}
                  </span>
                )}
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  p. {span.pageNumber ?? currentTask?.pageNumber ?? '\u2014'}
                </span>
                <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                  {'\u00A7'} {span.sectionId || '\u2014'}
                </span>
                <span
                  className={cn(
                    'text-[11px] px-2 py-0.5 rounded font-semibold',
                    getConfidenceColor(span.mergedConfidence),
                    'bg-opacity-10',
                  )}
                >
                  {(span.mergedConfidence * 100).toFixed(0)}%
                </span>
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

              {/* ── Task Description ────────────────────────────────────── */}
              {currentTask?.title && (
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3">
                  <p className="text-xs font-semibold text-gray-700 mb-1">
                    {currentTask.title}
                  </p>
                  {currentTask.description && (
                    <p className="text-xs text-gray-500">{currentTask.description}</p>
                  )}
                </div>
              )}

              {/* ── Extracted Text (mono-font for character comparison) ── */}
              {editMode ? (
                <div className="bg-white border border-blue-300 rounded-lg p-5">
                  <label className="block text-xs font-semibold text-blue-700 uppercase tracking-wide mb-2">
                    Editing Recovered Text
                  </label>
                  <textarea
                    value={editedText}
                    onChange={(e) => setEditedText(e.target.value)}
                    rows={6}
                    className="w-full border border-gray-200 rounded-md p-3 text-sm leading-[1.75] text-gray-900 font-mono resize-none focus:outline-none focus:ring-2 focus:ring-blue-300"
                    autoFocus
                  />

                  {/* Edit distance indicator */}
                  {editedText !== span.text && editSeverityConfig && (
                    <div
                      className={cn(
                        'mt-3 px-3 py-2 rounded-md border text-xs flex items-center justify-between',
                        editSeverityConfig.bg,
                        editSeverityConfig.border,
                        editSeverityConfig.color,
                      )}
                    >
                      <span>{editSeverityConfig.label}</span>
                      <span className="font-mono font-semibold">
                        {editDistanceInfo.changePercentage.toFixed(1)}% changed
                        ({editDistanceInfo.levenshteinDistance} edits)
                      </span>
                    </div>
                  )}
                </div>
              ) : (
                <div className="rounded-lg border border-gray-300 bg-gray-900 overflow-hidden">
                  <div className="px-4 py-2 bg-gray-800 border-b border-gray-700 flex items-center justify-between">
                    <span className="text-[10px] font-semibold text-gray-400 uppercase tracking-wide">
                      Recovered Text — Character-by-Character Comparison
                    </span>
                    <Eye className="h-3.5 w-3.5 text-gray-500" />
                  </div>
                  <div className="p-4">
                    <pre className="text-sm text-green-400 font-mono whitespace-pre-wrap break-words leading-relaxed">
                      {span.text}
                    </pre>
                  </div>
                </div>
              )}

              {/* ── Contributing Channels ──────────────────────────────── */}
              {span.contributingChannels.length > 0 && (
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3">
                  <div className="text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-2">
                    Contributing Channels
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
                          <span className="font-semibold">{ch}</span>
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

              {/* ── Reviewer Notes ──────────────────────────────────────── */}
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Reviewer notes (optional)..."
                className="w-full h-12 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 bg-[#FAFBFC] resize-none focus:outline-none focus:ring-2 focus:ring-blue-200"
              />

              {/* ── Error ──────────────────────────────────────────────── */}
              {error && (
                <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-red-700 text-xs">
                  {error}
                </div>
              )}

              {/* ── Action Bar ──────────────────────────────────────────── */}
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
                        disabled={actionLoading || !isPending}
                        title="Confirm (C)"
                        className={cn(
                          'px-5 py-2 rounded-md text-white text-xs font-semibold flex items-center gap-1.5 transition-colors',
                          !isPending
                            ? 'bg-gray-300 cursor-not-allowed'
                            : 'bg-green-600 hover:bg-green-700',
                        )}
                      >
                        {actionLoading && <Loader2 className="h-3 w-3 animate-spin" />}
                        Confirm
                      </button>
                      <button
                        onClick={() => {
                          setEditMode(true);
                          setEditedText(span.text);
                        }}
                        disabled={actionLoading || !isPending}
                        className="px-5 py-2 rounded-md bg-blue-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-blue-700 transition-colors disabled:opacity-50"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => setShowReject(true)}
                        disabled={actionLoading || !isPending}
                        className="px-5 py-2 rounded-md bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 hover:bg-red-700 transition-colors disabled:opacity-50"
                      >
                        Reject
                      </button>
                    </>
                  )}
                </div>
              </div>

              {/* ── Navigation ──────────────────────────────────────────── */}
              <div className="flex justify-between items-center pt-2.5">
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
                <span className="text-xs text-gray-500">
                  Task {currentIdx + 1} of {totalTasks}
                </span>
                <button
                  onClick={() => goTo('next')}
                  disabled={currentIdx >= totalTasks - 1}
                  className={cn(
                    'flex items-center gap-1 px-3.5 py-1.5 rounded border text-xs transition-colors',
                    currentIdx >= totalTasks - 1
                      ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                      : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700',
                  )}
                >
                  Next <ChevronRight className="h-3 w-3" />
                </button>
              </div>

              {/* ── Keyboard hint ──────────────────────────────────────── */}
              <div className="text-center text-[10px] text-gray-300 tracking-wide">
                C confirm · E edit · R reject · {'\u2190'} {'\u2192'} navigate
              </div>
            </>
          )}
        </div>
      </ReviewShell>

      {/* ── Reject Modal ──────────────────────────────────────────────── */}
      {showReject && (
        <RejectModal
          onClose={() => setShowReject(false)}
          onConfirm={handleRejectConfirm}
          isLoading={actionLoading}
        />
      )}
    </>
  );
}
