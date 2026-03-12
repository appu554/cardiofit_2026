'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle,
  XCircle,
  Pencil,
  ArrowUpRight,
  Loader2,
  Clock,
  Shield,
  ShieldCheck,
  ShieldX,
  Cpu,
  Brain,
  Hash,
  FileText,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { splReviewApi } from '@/lib/spl-api';
import type {
  SPLDerivedFact,
  ReviewAction,
  FactReviewDecision,
  ExtractionMethod,
  GovernanceStatus,
} from '@/types/spl-review';
import { EXTRACTION_METHOD_LABELS } from '@/types/spl-review';

// ============================================================================
// SPL Fact Card — Base wrapper for fact-level review
//
// Shows fact metadata (method, confidence, status), review action buttons,
// and delegates rendering of the fact-specific content to child components
// (SafetySignalCard, InteractionCard, etc.).
// ============================================================================

// Extraction method badge config
const METHOD_CONFIG: Record<ExtractionMethod, { icon: React.ComponentType<{ className?: string }>; color: string; bg: string }> = {
  STRUCTURED_PARSE: { icon: Cpu, color: 'text-emerald-700', bg: 'bg-emerald-50' },
  LLM_FALLBACK: { icon: Brain, color: 'text-amber-700', bg: 'bg-amber-50' },
  DDI_GRAMMAR: { icon: Hash, color: 'text-blue-700', bg: 'bg-blue-50' },
  PROSE_SCAN: { icon: FileText, color: 'text-purple-700', bg: 'bg-purple-50' },
  SPL_PRODUCT: { icon: Shield, color: 'text-gray-700', bg: 'bg-gray-50' },
};

// Status badge config
const STATUS_CONFIG: Record<GovernanceStatus, { icon: React.ComponentType<{ className?: string }>; color: string; bg: string; label: string }> = {
  PENDING_REVIEW: { icon: Clock, color: 'text-amber-700', bg: 'bg-amber-50', label: 'Pending' },
  APPROVED: { icon: ShieldCheck, color: 'text-emerald-700', bg: 'bg-emerald-50', label: 'Approved' },
  REJECTED: { icon: ShieldX, color: 'text-red-700', bg: 'bg-red-50', label: 'Rejected' },
  SUPERSEDED: { icon: Clock, color: 'text-gray-500', bg: 'bg-gray-50', label: 'Superseded' },
};

// ============================================================================
// Props
// ============================================================================

interface SPLFactCardProps {
  fact: SPLDerivedFact;
  /** The type-specific content rendered inside the card */
  children: React.ReactNode;
  /** Called after a review action completes */
  onActionComplete: () => void;
  /** Called when the user clicks the source highlight link */
  onHighlightSource?: (fact: SPLDerivedFact) => void;
  /** Whether this card is currently selected/focused */
  isActive?: boolean;
}

// ============================================================================
// Component
// ============================================================================

export function SPLFactCard({
  fact,
  children,
  onActionComplete,
  onHighlightSource,
  isActive,
}: SPLFactCardProps) {
  const queryClient = useQueryClient();
  const [activeAction, setActiveAction] = useState<ReviewAction | null>(null);
  const [note, setNote] = useState('');
  const [editedText, setEditedText] = useState('');
  const [rejectionReason, setRejectionReason] = useState('');
  const [error, setError] = useState<string | null>(null);

  const methodConfig = METHOD_CONFIG[fact.extractionMethod] || METHOD_CONFIG.STRUCTURED_PARSE;
  const statusConfig = STATUS_CONFIG[fact.governanceStatus] || STATUS_CONFIG.PENDING_REVIEW;
  const MethodIcon = methodConfig.icon;
  const StatusIcon = statusConfig.icon;

  const isPending = fact.governanceStatus === 'PENDING_REVIEW';

  // ── Submit review decision ──────────────────────────────────────────
  const mutation = useMutation({
    mutationFn: (decision: FactReviewDecision) =>
      splReviewApi.facts.submitDecision(decision),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spl-facts'] });
      queryClient.invalidateQueries({ queryKey: ['spl-completeness'] });
      queryClient.invalidateQueries({ queryKey: ['spl-queue-all'] });
      setActiveAction(null);
      setNote('');
      setEditedText('');
      setRejectionReason('');
      setError(null);
      onActionComplete();
    },
    onError: (err) => setError((err as Error).message),
  });

  const handleSubmit = (action: ReviewAction) => {
    if (action === 'REJECT' && !rejectionReason.trim()) {
      setError('Rejection reason is required');
      return;
    }

    const decision: FactReviewDecision = {
      factId: fact.id,
      action,
      reviewerId: 'pharmacist', // Will be replaced by Auth0 user
      notes: note || undefined,
      editedText: action === 'EDIT' ? editedText : undefined,
      rejectionReason: action === 'REJECT' ? rejectionReason : undefined,
      timestamp: new Date().toISOString(),
    };

    mutation.mutate(decision);
  };

  const handleCancel = () => {
    setActiveAction(null);
    setNote('');
    setEditedText('');
    setRejectionReason('');
    setError(null);
  };

  // ── Confidence color ────────────────────────────────────────────────
  const confidence = fact.extractionConfidence;
  const confColor =
    confidence >= 0.85 ? 'text-emerald-700 bg-emerald-50' :
    confidence >= 0.65 ? 'text-amber-700 bg-amber-50' :
    'text-red-700 bg-red-50';

  return (
    <div
      className={cn(
        'rounded-xl border transition-all',
        isActive
          ? 'ring-2 ring-blue-400 border-blue-300 shadow-md'
          : 'border-gray-200 bg-white hover:border-gray-300'
      )}
    >
      {/* ── Meta Header ──────────────────────────────────────────────── */}
      <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2">
          {/* Extraction method badge */}
          <span className={cn('inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded', methodConfig.bg, methodConfig.color)}>
            <MethodIcon className="h-3 w-3" />
            {EXTRACTION_METHOD_LABELS[fact.extractionMethod]}
          </span>

          {/* Confidence score */}
          <span className={cn('text-[10px] font-bold px-2 py-0.5 rounded', confColor)}>
            {(confidence * 100).toFixed(0)}%
          </span>

          {/* Status badge */}
          <span className={cn('inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded', statusConfig.bg, statusConfig.color)}>
            <StatusIcon className="h-3 w-3" />
            {statusConfig.label}
          </span>
        </div>

        {/* Source highlight link */}
        {onHighlightSource && (
          <button
            onClick={() => onHighlightSource(fact)}
            className="text-[10px] text-blue-600 hover:text-blue-800 font-medium flex items-center gap-1 transition-colors"
          >
            <FileText className="h-3 w-3" />
            View in source
          </button>
        )}
      </div>

      {/* ── Fact-Specific Content (delegated) ────────────────────────── */}
      <div className="p-4">
        {children}
      </div>

      {/* ── Fact Key ─────────────────────────────────────────────────── */}
      <div className="px-4 py-2 border-t border-gray-50 text-[10px] text-gray-400 font-mono truncate">
        {fact.factKey}
      </div>

      {/* ── Review Actions ───────────────────────────────────────────── */}
      {isPending && (
        <div className="px-4 py-3 border-t border-gray-100 bg-gray-50/50">
          {!activeAction ? (
            <div className="flex items-center gap-2">
              <button
                onClick={() => handleSubmit('CONFIRM')}
                disabled={mutation.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-semibold hover:bg-emerald-700 transition-colors disabled:opacity-50"
              >
                {mutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <CheckCircle className="h-3.5 w-3.5" />}
                Confirm
              </button>
              <button
                onClick={() => setActiveAction('EDIT')}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 transition-colors"
              >
                <Pencil className="h-3.5 w-3.5" />
                Edit
              </button>
              <button
                onClick={() => setActiveAction('REJECT')}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700 transition-colors"
              >
                <XCircle className="h-3.5 w-3.5" />
                Reject
              </button>
              <button
                onClick={() => setActiveAction('ESCALATE')}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-amber-500 text-white text-xs font-semibold hover:bg-amber-600 transition-colors ml-auto"
              >
                <ArrowUpRight className="h-3.5 w-3.5" />
                Escalate
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {/* Active action header */}
              <div className={cn(
                'p-2 rounded-lg text-xs font-medium flex items-center gap-1.5',
                activeAction === 'EDIT' && 'bg-blue-50 text-blue-700',
                activeAction === 'REJECT' && 'bg-red-50 text-red-700',
                activeAction === 'ESCALATE' && 'bg-amber-50 text-amber-700',
              )}>
                {activeAction === 'EDIT' && <><Pencil className="h-3.5 w-3.5" /> Editing Fact</>}
                {activeAction === 'REJECT' && <><XCircle className="h-3.5 w-3.5" /> Rejecting Fact</>}
                {activeAction === 'ESCALATE' && <><ArrowUpRight className="h-3.5 w-3.5" /> Escalating to SME</>}
              </div>

              {/* Edit: text area */}
              {activeAction === 'EDIT' && (
                <textarea
                  value={editedText}
                  onChange={(e) => setEditedText(e.target.value)}
                  rows={3}
                  className="w-full border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 resize-none focus:outline-none focus:ring-2 focus:ring-blue-200"
                  placeholder="Enter corrected fact data..."
                  autoFocus
                />
              )}

              {/* Reject: reason */}
              {activeAction === 'REJECT' && (
                <select
                  value={rejectionReason}
                  onChange={(e) => setRejectionReason(e.target.value)}
                  className="w-full border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 focus:outline-none focus:ring-2 focus:ring-red-200"
                >
                  <option value="">Select rejection reason...</option>
                  <option value="NOISE">Noise / garbage extraction</option>
                  <option value="MISCLASSIFICATION">Wrong fact type classification</option>
                  <option value="INVALID_MEDDRA">Invalid MedDRA mapping</option>
                  <option value="DUPLICATE">Duplicate of another fact</option>
                  <option value="INCORRECT_DATA">Incorrect data extraction</option>
                  <option value="OUT_OF_SCOPE">Out of scope for this drug</option>
                </select>
              )}

              {/* Notes */}
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={2}
                className="w-full border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-800 bg-gray-50 resize-none focus:outline-none focus:ring-2 focus:ring-gray-200"
                placeholder="Reviewer notes (optional)..."
              />

              {/* Error */}
              {error && (
                <div className="p-2 bg-red-50 border border-red-200 rounded-lg text-red-700 text-xs">
                  {error}
                </div>
              )}

              {/* Action buttons */}
              <div className="flex gap-2">
                <button
                  onClick={handleCancel}
                  disabled={mutation.isPending}
                  className="px-3 py-1.5 rounded-lg border border-gray-200 text-xs font-medium text-gray-600 hover:bg-gray-100 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleSubmit(activeAction)}
                  disabled={mutation.isPending}
                  className={cn(
                    'px-4 py-1.5 rounded-lg text-white text-xs font-semibold flex items-center gap-1.5 transition-colors disabled:opacity-50',
                    activeAction === 'EDIT' && 'bg-blue-600 hover:bg-blue-700',
                    activeAction === 'REJECT' && 'bg-red-600 hover:bg-red-700',
                    activeAction === 'ESCALATE' && 'bg-amber-500 hover:bg-amber-600',
                  )}
                >
                  {mutation.isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                  Submit
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Already-reviewed indicator ────────────────────────────────── */}
      {!isPending && fact.reviewedBy && (
        <div className="px-4 py-2 border-t border-gray-100 bg-gray-50/50 text-[10px] text-gray-500">
          Reviewed by {fact.reviewedBy}
          {fact.reviewedAt && ` on ${new Date(fact.reviewedAt).toLocaleDateString()}`}
          {fact.reviewNotes && ` — "${fact.reviewNotes}"`}
        </div>
      )}
    </div>
  );
}
