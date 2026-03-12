'use client';

import { useState, useCallback, useImperativeHandle, forwardRef } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CheckCircle, XCircle, Pencil, Loader2, AlertTriangle, Info, ShieldAlert } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { getChannelInfo, ALL_CHANNELS, getConfidenceColor } from '@/lib/pipeline1-channels';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import type { MergedSpan, SpanReviewRequest, RejectReason } from '@/types/pipeline1';
import { REJECT_REASON_LABELS } from '@/types/pipeline1';
import { SemanticText } from './SemanticText';
import { CoverageGuardAlertBanner } from './CoverageGuardAlert';

// =============================================================================
// Types
// =============================================================================

interface SpanInspectorProps {
  span: MergedSpan | null;
  jobId: string;
  onActionComplete: () => void;
}

type ActionType = 'confirm' | 'reject' | 'edit' | null;

/** Imperative handle for keyboard shortcuts from parent */
export interface SpanInspectorHandle {
  triggerConfirm: () => void;
  triggerEdit: () => void;
  triggerReject: () => void;
  triggerCancel: () => void;
  triggerSubmit: () => void;
}

// =============================================================================
// Review status badge map (consistent with SpanReviewCard)
// =============================================================================

const STATUS_BADGE: Record<string, { icon: typeof CheckCircle; color: string; label: string }> = {
  PENDING:   { icon: AlertTriangle, color: 'text-amber-600 bg-amber-50 border-amber-200', label: 'Pending' },
  CONFIRMED: { icon: CheckCircle,   color: 'text-green-600 bg-green-50 border-green-200', label: 'Confirmed' },
  REJECTED:  { icon: XCircle,       color: 'text-red-600 bg-red-50 border-red-200',       label: 'Rejected' },
  EDITED:    { icon: Pencil,        color: 'text-blue-600 bg-blue-50 border-blue-200',    label: 'Edited' },
  ADDED:     { icon: Info,          color: 'text-purple-600 bg-purple-50 border-purple-200', label: 'Added' },
};

// =============================================================================
// Component
// =============================================================================

export const SpanInspector = forwardRef<SpanInspectorHandle, SpanInspectorProps>(
  function SpanInspector({ span, jobId, onActionComplete }, ref) {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [activeAction, setActiveAction] = useState<ActionType>(null);
  const [note, setNote] = useState('');
  const [editedText, setEditedText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState<RejectReason | null>(null);

  // Alert-gated confirm: block direct confirm when critical CoverageGuard alert present
  const hasCriticalAlert = span?.coverageGuardAlert?.alertSeverity === 'critical';

  // ---------------------------------------------------------------------------
  // Mutations
  // ---------------------------------------------------------------------------

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans-all', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-review-tasks', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
  };

  const handleReset = useCallback(() => {
    setActiveAction(null);
    setNote('');
    setEditedText('');
    setError(null);
    setRejectReason(null);
  }, []);

  const onMutationSuccess = () => {
    invalidateAll();
    handleReset();
    onActionComplete();
  };

  const onMutationError = (err: unknown) => {
    setError((err as Error).message);
  };

  const confirmMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.confirm(jobId, span!.id, req),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });

  const rejectMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.reject(jobId, span!.id, req),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });

  const editMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.edit(jobId, span!.id, req),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });

  const isLoading =
    confirmMutation.isPending || rejectMutation.isPending || editMutation.isPending;

  // ---------------------------------------------------------------------------
  // Submit handler
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(() => {
    if (!span) return;
    const reviewerId = user?.sub || 'unknown';
    const req: SpanReviewRequest = {
      reviewerId,
      ...(note ? { note } : {}),
    };

    switch (activeAction) {
      case 'confirm':
        confirmMutation.mutate(req);
        break;
      case 'reject':
        if (!rejectReason) {
          setError('Select a rejection reason.');
          return;
        }
        if (rejectReason === 'other' && !note.trim()) {
          setError('A note is required when "Other" is selected.');
          return;
        }
        rejectMutation.mutate({ ...req, rejectReason });
        break;
      case 'edit':
        if (!editedText.trim()) {
          setError('Edited text is required.');
          return;
        }
        editMutation.mutate({ ...req, editedText });
        break;
    }
  }, [span, user, note, activeAction, rejectReason, editedText, confirmMutation, rejectMutation, editMutation]);

  // ---------------------------------------------------------------------------
  // Imperative handle for keyboard shortcuts
  // ---------------------------------------------------------------------------

  useImperativeHandle(ref, () => ({
    triggerConfirm: () => {
      if (span && !hasCriticalAlert && !activeAction) setActiveAction('confirm');
    },
    triggerEdit: () => {
      if (span && !activeAction) {
        setActiveAction('edit');
        setEditedText(span.text);
      }
    },
    triggerReject: () => {
      if (span && !activeAction) setActiveAction('reject');
    },
    triggerCancel: () => handleReset(),
    triggerSubmit: () => {
      if (activeAction) handleSubmit();
    },
  }), [span, activeAction, hasCriticalAlert, handleSubmit, handleReset]);

  // ---------------------------------------------------------------------------
  // Empty state
  // ---------------------------------------------------------------------------

  if (!span) {
    return (
      <div className="card h-full">
        <div className="card-header">
          <h2 className="text-lg font-semibold text-gray-900">Span Inspector</h2>
        </div>
        <div className="card-body flex items-center justify-center py-16 text-gray-400">
          <p className="text-sm">Select a span to inspect</p>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Derived values
  // ---------------------------------------------------------------------------

  const badge = STATUS_BADGE[span.reviewStatus] || STATUS_BADGE.PENDING;
  const BadgeIcon = badge.icon;
  const confidencePct = (span.mergedConfidence * 100).toFixed(0);
  const hasOracleChannel = span.contributingChannels.includes('L1');
  const isRecommendation = /^recommendation/i.test(span.text);

  // ---------------------------------------------------------------------------
  // Provenance chain text
  // ---------------------------------------------------------------------------

  const isL1Recovery = span.contributingChannels.includes('L1_RECOVERY') || span.contributingChannels.includes('L1');
  const isUnpositioned = span.startOffset < 0;

  const provenanceLines = [
    `section: ${span.sectionId || '\u2014'}`,
    `  \u2514\u2500 merged_span: ${span.id.slice(0, 12)}`,
    `     channels: [${span.contributingChannels.join(', ')}]`,
    `     confidence: ${span.mergedConfidence}`,
    `     offsets: [${span.startOffset}..${span.endOffset}]`,
  ].join('\n');

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="card h-full flex flex-col">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">Span Inspector</h2>
      </div>

      <div className="card-body flex-1 overflow-y-auto space-y-5">
        {/* ----------------------------------------------------------------- */}
        {/* Header: channel badges + span text                                */}
        {/* ----------------------------------------------------------------- */}
        <div>
          <div className="flex flex-wrap gap-1.5 mb-2">
            {span.contributingChannels.map((ch) => {
              const info = getChannelInfo(ch);
              return (
                <span
                  key={ch}
                  className={cn(
                    'inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full',
                    info.bg,
                    info.color,
                  )}
                >
                  {ch} &mdash; {info.name}
                </span>
              );
            })}
          </div>
          {/* CoverageGuard Alert Banner */}
          {span.coverageGuardAlert && (
            <div className="mb-2">
              <CoverageGuardAlertBanner alert={span.coverageGuardAlert} />
            </div>
          )}
          <div className="max-h-32 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-3">
            <SemanticText
              text={span.text}
              tokens={span.semanticTokens}
              className="text-sm text-gray-800 whitespace-pre-wrap break-words"
            />
          </div>
        </div>

        {/* ----------------------------------------------------------------- */}
        {/* Detail rows                                                       */}
        {/* ----------------------------------------------------------------- */}
        <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <div className="text-gray-500">Type</div>
          <div className="text-gray-900 font-medium">
            {span.sectionId || '\u2014'}
          </div>

          <div className="text-gray-500">Confidence</div>
          <div className={cn('font-medium', getConfidenceColor(span.mergedConfidence))}>
            {confidencePct}%
          </div>

          <div className="text-gray-500">Status</div>
          <div>
            <span
              className={cn(
                'inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full border',
                badge.color,
              )}
            >
              <BadgeIcon className="h-3 w-3 mr-1" />
              {badge.label}
            </span>
          </div>

          {span.tier != null && (
            <>
              <div className="text-gray-500">Risk Tier</div>
              <div>
                <span
                  className={cn(
                    'inline-flex items-center text-xs font-bold px-2 py-0.5 rounded-full',
                    span.tier === 1 && 'bg-red-100 text-red-800',
                    span.tier === 2 && 'bg-amber-100 text-amber-800',
                    span.tier === 3 && 'bg-gray-100 text-gray-700',
                  )}
                >
                  TIER {span.tier}
                </span>
              </div>
            </>
          )}

          <div className="text-gray-500">Page</div>
          <div className="text-gray-900 font-medium">
            {span.pageNumber != null ? span.pageNumber : '\u2014'}
          </div>
        </div>

        {/* ----------------------------------------------------------------- */}
        {/* Provenance Chain                                                   */}
        {/* ----------------------------------------------------------------- */}
        <div>
          <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
            Provenance Chain
          </h3>
          <pre className="bg-gray-900 text-green-400 font-mono text-xs p-3 rounded whitespace-pre overflow-x-auto">
            {provenanceLines}
          </pre>
        </div>

        {/* ----------------------------------------------------------------- */}
        {/* Channel Corroboration Matrix                                       */}
        {/* ----------------------------------------------------------------- */}
        <div>
          <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
            Channel Corroboration
          </h3>
          <div className="flex gap-2">
            {ALL_CHANNELS.map((ch) => {
              const present = span.contributingChannels.includes(ch);
              return (
                <div key={ch} className="flex flex-col items-center gap-0.5">
                  {present ? (
                    <CheckCircle className="h-4 w-4 text-green-500" aria-label={`Channel ${ch} present`} />
                  ) : (
                    <span className="h-4 w-4 flex items-center justify-center text-gray-300" aria-label={`Channel ${ch} absent`}>
                      &mdash;
                    </span>
                  )}
                  <span className="text-[10px] font-medium text-gray-500">{ch}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* ----------------------------------------------------------------- */}
        {/* Recovery / Disagreement Context (the diff the reviewer needs)     */}
        {/* ----------------------------------------------------------------- */}
        {isL1Recovery && (
          <div className="rounded-lg border border-red-300 bg-red-50 overflow-hidden">
            <div className="flex items-center gap-2 px-3 py-2 bg-red-100 border-b border-red-200">
              <AlertTriangle className="h-4 w-4 text-red-700 flex-shrink-0" />
              <p className="text-sm font-semibold text-red-800">L1 Oracle Recovery</p>
            </div>
            <div className="px-3 py-2.5 space-y-2">
              <p className="text-xs text-red-700">
                This text was <strong>silently dropped</strong> by Marker OCR.
                Recovered by PyMuPDF rawdict fallback. Compare with Source PDF to verify accuracy.
              </p>
              {isUnpositioned && (
                <p className="text-xs text-red-600 italic">
                  No text offsets available &mdash; this span cannot be overlaid on normalized text.
                  Use the Source PDF tab for ground-truth comparison.
                </p>
              )}
              <div className="rounded border border-red-200 bg-white p-2.5">
                <p className="text-[10px] font-semibold text-red-500 uppercase tracking-wide mb-1">
                  Recovered Text
                </p>
                <SemanticText
                  text={span.text}
                  tokens={span.semanticTokens}
                  className="text-sm text-gray-900 whitespace-pre-wrap break-words"
                />
              </div>
            </div>
          </div>
        )}

        {span.hasDisagreement && span.disagreementDetail && !isL1Recovery && (
          <div className="rounded-lg border border-amber-300 bg-amber-50 overflow-hidden">
            <div className="flex items-center gap-2 px-3 py-2 bg-amber-100 border-b border-amber-200">
              <AlertTriangle className="h-4 w-4 text-amber-700 flex-shrink-0" />
              <p className="text-sm font-semibold text-amber-800">Channel Disagreement</p>
            </div>
            <div className="px-3 py-2.5">
              <p className="text-xs text-amber-700 whitespace-pre-wrap">
                {span.disagreementDetail}
              </p>
            </div>
          </div>
        )}

        {/* ----------------------------------------------------------------- */}
        {/* Recommendation Info (conditional)                                  */}
        {/* ----------------------------------------------------------------- */}
        {isRecommendation && (
          <div className="flex items-start gap-2 p-3 rounded-lg border border-blue-200 bg-blue-50">
            <Info className="h-4 w-4 text-blue-600 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm font-semibold text-blue-700">Guideline Recommendation</p>
              <p className="text-xs text-blue-600 mt-0.5">
                Verify grading and evidence level.
              </p>
            </div>
          </div>
        )}

        {/* ----------------------------------------------------------------- */}
        {/* Action Buttons                                                     */}
        {/* ----------------------------------------------------------------- */}
        {!activeAction ? (
          <div className="space-y-2 pt-2">
            {/* Alert-gated Confirm: disabled when critical CoverageGuard alert */}
            {hasCriticalAlert && (
              <div className="flex items-start gap-2 p-2.5 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700 mb-1">
                <ShieldAlert className="h-4 w-4 text-red-500 mt-0.5 flex-shrink-0" />
                <span>
                  <strong>Confirm blocked</strong> — this span has a critical CoverageGuard alert.
                  Review the alert, then Edit or Reject.
                </span>
              </div>
            )}
            <button
              onClick={() => setActiveAction('confirm')}
              disabled={hasCriticalAlert}
              className={cn(
                'w-full btn flex items-center justify-center',
                hasCriticalAlert
                  ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                  : 'bg-green-600 text-white hover:bg-green-700',
              )}
            >
              <CheckCircle className="h-5 w-5 mr-2" />
              Confirm
            </button>
            <button
              onClick={() => {
                setActiveAction('edit');
                setEditedText(span.text);
              }}
              className="w-full btn bg-blue-600 text-white hover:bg-blue-700 flex items-center justify-center"
            >
              <Pencil className="h-5 w-5 mr-2" />
              Edit
            </button>
            <button
              onClick={() => setActiveAction('reject')}
              className="w-full btn bg-red-600 text-white hover:bg-red-700 flex items-center justify-center"
            >
              <XCircle className="h-5 w-5 mr-2" />
              Reject
            </button>
          </div>
        ) : (
          <div className="space-y-4 pt-2">
            {/* Active action banner */}
            <div
              className={cn(
                'p-3 rounded-lg text-sm font-medium flex items-center',
                activeAction === 'confirm' && 'bg-green-50 text-green-700',
                activeAction === 'edit'    && 'bg-blue-50 text-blue-700',
                activeAction === 'reject'  && 'bg-red-50 text-red-700',
              )}
            >
              {activeAction === 'confirm' && (
                <><CheckCircle className="h-5 w-5 mr-2" />Confirming Span</>
              )}
              {activeAction === 'edit' && (
                <><Pencil className="h-5 w-5 mr-2" />Editing Span</>
              )}
              {activeAction === 'reject' && (
                <><XCircle className="h-5 w-5 mr-2" />Rejecting Span</>
              )}
            </div>

            {/* Edit textarea */}
            {activeAction === 'edit' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Corrected Text <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={editedText}
                  onChange={(e) => setEditedText(e.target.value)}
                  rows={4}
                  className="input w-full text-sm"
                />
              </div>
            )}

            {/* Structured Reject Reasons */}
            {activeAction === 'reject' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Rejection Reason <span className="text-red-500">*</span>
                </label>
                <div className="space-y-1">
                  {(Object.entries(REJECT_REASON_LABELS) as [RejectReason, string][]).map(
                    ([key, label]) => (
                      <label
                        key={key}
                        className={cn(
                          'flex items-center gap-2 px-3 py-1.5 rounded-md border text-sm cursor-pointer transition-colors',
                          rejectReason === key
                            ? 'bg-red-50 border-red-300 text-red-800'
                            : 'bg-gray-50 border-gray-200 text-gray-700 hover:bg-gray-100',
                        )}
                      >
                        <input
                          type="radio"
                          name="rejectReason"
                          value={key}
                          checked={rejectReason === key}
                          onChange={() => setRejectReason(key)}
                          className="accent-red-600"
                        />
                        {label}
                      </label>
                    ),
                  )}
                </div>
              </div>
            )}

            {/* Additional Notes */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                {activeAction === 'reject' ? 'Additional Notes' : 'Note'}{' '}
                {activeAction === 'reject' && rejectReason === 'other' && (
                  <span className="text-red-500">*</span>
                )}
              </label>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={2}
                className="input w-full text-sm"
                placeholder={
                  activeAction === 'reject'
                    ? rejectReason === 'other'
                      ? 'Describe the issue...'
                      : 'Optional additional context...'
                    : 'Optional note...'
                }
              />
            </div>

            {/* Error message */}
            {error && (
              <div className="p-3 bg-red-50 border border-red-100 rounded-lg text-red-700 text-sm">
                {error}
              </div>
            )}

            {/* Submit / Cancel */}
            <div className="flex space-x-3">
              <button
                onClick={handleReset}
                disabled={isLoading}
                className="flex-1 btn btn-outline"
              >
                Cancel
              </button>
              <button
                onClick={handleSubmit}
                disabled={isLoading}
                className={cn(
                  'flex-1 btn flex items-center justify-center',
                  activeAction === 'confirm' && 'bg-green-600 text-white hover:bg-green-700',
                  activeAction === 'edit'    && 'bg-blue-600 text-white hover:bg-blue-700',
                  activeAction === 'reject'  && 'bg-red-600 text-white hover:bg-red-700',
                  isLoading && 'opacity-50 cursor-not-allowed',
                )}
              >
                {isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Submit
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
});
