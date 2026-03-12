'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, AlertTriangle, Loader2, Sparkles } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { useEditDistance, getEditSeverity, EDIT_SEVERITY_CONFIG } from '@/hooks/useEditDistance';
import { SemanticText } from './SemanticText';
import { CoverageGuardAlertBanner } from './CoverageGuardAlert';
import { RejectModal } from './RejectModal';
import { ReviewShell } from './ReviewShell';
import { cn } from '@/lib/utils';
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import type { MergedSpan, ExtractionJob, SpanReviewRequest, RejectReason } from '@/types/pipeline1';

// =============================================================================
// Phase 3A — F-Only Corroboration
//
// Reviews spans extracted only by Channel F (NuExtract LLM) with low
// confidence (<=0.5). These are LLM-only extractions with no corroboration
// from deterministic channels — higher hallucination risk.
//
// Layout: ReviewShell split-panel (left: span card, right: PDF)
// Workflow: 3-item checklist → confirm/edit/reject → auto-advance
// =============================================================================

// -----------------------------------------------------------------------------
// Props
// -----------------------------------------------------------------------------

interface Phase3AFOnlyCorroborationProps {
  jobId: string;
  job: ExtractionJob;
  onActionComplete: () => void;
  onPhaseComplete: () => void;
}

// -----------------------------------------------------------------------------
// Checklist definition
// -----------------------------------------------------------------------------

interface FOnlyChecklist {
  textPresentInSource: boolean;
  assignedCorrectSection: boolean;
  consistentWithOtherChannels: boolean;
}

const CHECKLIST_LABELS: Record<keyof FOnlyChecklist, string> = {
  textPresentInSource: 'Text present in source document',
  assignedCorrectSection: 'Assigned to correct section',
  consistentWithOtherChannels: 'Consistent with other channel extractions',
};

const INITIAL_CHECKLIST: FOnlyChecklist = {
  textPresentInSource: false,
  assignedCorrectSection: false,
  consistentWithOtherChannels: false,
};

// =============================================================================
// Component
// =============================================================================

export function Phase3AFOnlyCorroboration({
  jobId,
  job,
  onActionComplete,
  onPhaseComplete,
}: Phase3AFOnlyCorroborationProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------
  const [currentIdx, setCurrentIdx] = useState(0);
  const [checklist, setChecklist] = useState<FOnlyChecklist>(INITIAL_CHECKLIST);
  const [editMode, setEditMode] = useState(false);
  const [editedText, setEditedText] = useState('');
  const [note, setNote] = useState('');
  const [showReject, setShowReject] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ---------------------------------------------------------------------------
  // Data fetching: low-confidence spans, then client-side F-only filter
  // ---------------------------------------------------------------------------
  const { data: spansData, isLoading: spansLoading } = useQuery({
    queryKey: ['pipeline1-spans', jobId, 'low-confidence-f-only'],
    queryFn: () =>
      pipeline1Api.spans.list(jobId, { minConfidence: 0, maxConfidence: 0.5 }, 1, 500),
    enabled: !!jobId,
  });

  // Client-side filter: only spans where contributingChannels includes 'F'
  const fOnlySpans = useMemo(() => {
    if (!spansData?.items) return [];
    return spansData.items.filter(
      (s) => s.contributingChannels.includes('F'),
    );
  }, [spansData]);

  // Further filter to PENDING only for auto-complete detection
  const pendingFOnlySpans = useMemo(
    () => fOnlySpans.filter((s) => s.reviewStatus === 'PENDING'),
    [fOnlySpans],
  );

  const currentSpan: MergedSpan | null = fOnlySpans[currentIdx] ?? null;
  const totalSpans = fOnlySpans.length;

  // ---------------------------------------------------------------------------
  // Edit distance tracking (source-constrained edits)
  // ---------------------------------------------------------------------------
  const editDistanceInfo = useEditDistance(currentSpan?.text ?? '', editedText);
  const editSeverity = editMode ? getEditSeverity(editDistanceInfo.changePercentage) : null;
  const editSeverityConfig = editSeverity ? EDIT_SEVERITY_CONFIG[editSeverity] : null;

  // ---------------------------------------------------------------------------
  // Checklist completion gate
  // ---------------------------------------------------------------------------
  const allChecked = checklist.textPresentInSource
    && checklist.assignedCorrectSection
    && checklist.consistentWithOtherChannels;

  // ---------------------------------------------------------------------------
  // Channel F confidence for the current span
  // ---------------------------------------------------------------------------
  const channelFConfidence = currentSpan?.channelConfidences?.['F'] ?? currentSpan?.mergedConfidence ?? 0;

  // ---------------------------------------------------------------------------
  // Reset form state when navigating
  // ---------------------------------------------------------------------------
  useEffect(() => {
    setChecklist(INITIAL_CHECKLIST);
    setEditMode(false);
    setEditedText('');
    setNote('');
    setError(null);
  }, [currentIdx]);

  // ---------------------------------------------------------------------------
  // Auto-complete: when no more F-only low-confidence PENDING spans remain
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!spansLoading && fOnlySpans.length > 0 && pendingFOnlySpans.length === 0) {
      onPhaseComplete();
    }
  }, [spansLoading, fOnlySpans.length, pendingFOnlySpans.length, onPhaseComplete]);

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
    if (currentIdx < totalSpans - 1) {
      setCurrentIdx((i) => i + 1);
    }
  }, [currentIdx, totalSpans]);

  const goTo = useCallback(
    (dir: 'prev' | 'next') => {
      if (dir === 'prev' && currentIdx > 0) {
        setCurrentIdx((i) => i - 1);
      }
      if (dir === 'next' && currentIdx < totalSpans - 1) {
        setCurrentIdx((i) => i + 1);
      }
    },
    [currentIdx, totalSpans],
  );

  // ---------------------------------------------------------------------------
  // Action handlers
  // ---------------------------------------------------------------------------
  const handleConfirm = useCallback(async () => {
    if (!currentSpan || !allChecked) return;
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
  }, [currentSpan, allChecked, user, note, jobId, invalidateAll, onActionComplete, advance]);

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
          if (!editMode && allChecked) handleConfirm();
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
  }, [editMode, allChecked, handleConfirm, goTo, currentSpan]);

  // ---------------------------------------------------------------------------
  // Loading state
  // ---------------------------------------------------------------------------
  if (spansLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-pink-400" />
          <p className="text-sm text-gray-500">Loading F-Only spans...</p>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Empty state: no F-only low-confidence spans
  // ---------------------------------------------------------------------------
  if (fOnlySpans.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center px-8">
          <Sparkles className="h-12 w-12 text-pink-300 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-gray-700 mb-2">
            No F-Only Low-Confidence Spans
          </h3>
          <p className="text-sm text-gray-500 max-w-md">
            No spans were extracted solely by Channel F (NuExtract LLM) with confidence
            at or below 0.5. This phase requires no action.
          </p>
          <button
            onClick={onPhaseComplete}
            className="mt-6 px-6 py-2.5 rounded-lg bg-pink-600 text-white text-sm font-semibold hover:bg-pink-700 transition-colors"
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
  const fInfo = getChannelInfo('F');
  const span = currentSpan!;
  const isPending = span.reviewStatus === 'PENDING';
  const confidenceColor = getConfidenceColor(span.mergedConfidence);

  // ---------------------------------------------------------------------------
  // Render: ReviewShell with left panel content
  // ---------------------------------------------------------------------------
  return (
    <>
      <ReviewShell
        jobId={jobId}
        pdfPage={span.pageNumber ?? undefined}
        pdfHighlightText={span.text}
        topBar={
          <div className="flex items-center justify-between px-6 py-2.5">
            <div className="flex items-center gap-3">
              <Sparkles className="h-4 w-4 text-pink-600" />
              <span className="text-sm font-semibold text-gray-900">
                Phase 3A: F-Only Corroboration
              </span>
              <span className="text-xs text-gray-400">|</span>
              <span className="text-xs text-gray-500">
                Low-confidence LLM-only extractions
              </span>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-xs text-gray-500">
                {pendingFOnlySpans.length} pending of {totalSpans}
              </span>
            </div>
          </div>
        }
      >
        <div className="px-6 pt-5 pb-4 space-y-4">
          {/* ── LLM-Only Extraction Warning Banner ──────────────────────── */}
          <div className="rounded-lg border border-pink-300 bg-pink-50 overflow-hidden">
            <div className="flex items-center gap-2 px-4 py-2.5 bg-pink-100 border-b border-pink-200">
              <AlertTriangle className="h-4 w-4 text-pink-700 shrink-0" />
              <span className="text-xs font-bold text-pink-800 uppercase tracking-wider">
                LLM-Only Extraction
              </span>
              <span className="ml-auto text-xs text-pink-600 font-medium">
                Channel F confidence: {channelFConfidence.toFixed(2)}
              </span>
            </div>
            <div className="px-4 py-3">
              <p className="text-xs text-pink-800 leading-relaxed">
                This fact was extracted solely by the NuExtract LLM without corroboration
                from deterministic channels. Extra verification against the source PDF is
                required.
              </p>
            </div>
          </div>

          {/* ── Meta Row ────────────────────────────────────────────────── */}
          <div className="flex items-center gap-2 flex-wrap">
            <span
              className={cn(
                'inline-flex items-center text-[10px] font-bold px-2 py-0.5 rounded tracking-wider',
                fInfo.bg,
                fInfo.color,
                'border border-pink-300',
              )}
            >
              <Sparkles className="h-3 w-3 mr-1" />
              F — NuExtract LLM
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
            <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
              p. {span.pageNumber ?? '\u2014'}
            </span>
            <span className="text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
              \u00A7 {span.sectionId || '\u2014'}
            </span>
            <span className={cn('text-[11px] px-2 py-0.5 rounded font-semibold', confidenceColor, 'bg-opacity-10')}>
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

          {/* ── CoverageGuard Alert (if any) ──────────────────────────── */}
          {span.coverageGuardAlert && (
            <CoverageGuardAlertBanner alert={span.coverageGuardAlert} />
          )}

          {/* ── Fact Text / Edit Mode ──────────────────────────────────── */}
          {editMode ? (
            <div className="bg-white border border-blue-300 rounded-lg p-5">
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
            <div className="bg-white border border-gray-200 rounded-lg p-5 leading-[1.75] text-[15px] text-gray-900 font-serif">
              <SemanticText text={span.text} tokens={span.semanticTokens} />
            </div>
          )}

          {/* ── 3-Item Verification Checklist ──────────────────────────── */}
          {isPending && !editMode && (
            <div className="bg-gray-50 border border-gray-200 rounded-lg overflow-hidden">
              <div className="px-4 py-2.5 bg-gray-100 border-b border-gray-200">
                <span className="text-xs font-semibold text-gray-700 uppercase tracking-wide">
                  Verification Checklist
                </span>
                <span className="text-[10px] text-gray-400 ml-2">
                  All items required before confirm
                </span>
              </div>
              <div className="px-4 py-3 space-y-2">
                {(Object.entries(CHECKLIST_LABELS) as [keyof FOnlyChecklist, string][]).map(
                  ([key, label]) => (
                    <label
                      key={key}
                      className={cn(
                        'flex items-center gap-3 px-3 py-2 rounded-md border cursor-pointer transition-colors',
                        checklist[key]
                          ? 'bg-green-50 border-green-200 text-green-800'
                          : 'bg-white border-gray-200 text-gray-700 hover:bg-gray-50',
                      )}
                    >
                      <input
                        type="checkbox"
                        checked={checklist[key]}
                        onChange={(e) =>
                          setChecklist((prev) => ({ ...prev, [key]: e.target.checked }))
                        }
                        className="h-4 w-4 rounded border-gray-300 text-green-600 focus:ring-green-500 accent-green-600"
                      />
                      <span className="text-sm">{label}</span>
                    </label>
                  ),
                )}
              </div>
              {!allChecked && (
                <div className="px-4 py-2 border-t border-gray-200 bg-amber-50">
                  <p className="text-[10px] text-amber-700 font-medium">
                    Complete all checklist items to enable the Confirm button
                  </p>
                </div>
              )}
            </div>
          )}

          {/* ── Contributing Channels ──────────────────────────────────── */}
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

          {/* ── Reviewer Notes ──────────────────────────────────────────── */}
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Reviewer notes (optional)..."
            className="w-full h-12 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 bg-[#FAFBFC] resize-none focus:outline-none focus:ring-2 focus:ring-blue-200"
          />

          {/* ── Error ──────────────────────────────────────────────────── */}
          {error && (
            <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-red-700 text-xs">
              {error}
            </div>
          )}

          {/* ── Action Bar ──────────────────────────────────────────────── */}
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
                    disabled={!allChecked || actionLoading || !isPending}
                    title={!allChecked ? 'Complete checklist first' : 'Confirm (C)'}
                    className={cn(
                      'px-5 py-2 rounded-md text-white text-xs font-semibold flex items-center gap-1.5 transition-colors',
                      !allChecked || !isPending
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

          {/* ── Navigation ──────────────────────────────────────────────── */}
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
              Span {currentIdx + 1} of {totalSpans}
            </span>
            <button
              onClick={() => goTo('next')}
              disabled={currentIdx >= totalSpans - 1}
              className={cn(
                'flex items-center gap-1 px-3.5 py-1.5 rounded border text-xs transition-colors',
                currentIdx >= totalSpans - 1
                  ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                  : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700',
              )}
            >
              Next <ChevronRight className="h-3 w-3" />
            </button>
          </div>

          {/* ── Keyboard hint ──────────────────────────────────────────── */}
          <div className="text-center text-[10px] text-gray-300 tracking-wide">
            C confirm · E edit · R reject · \u2190 \u2192 navigate
          </div>
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
