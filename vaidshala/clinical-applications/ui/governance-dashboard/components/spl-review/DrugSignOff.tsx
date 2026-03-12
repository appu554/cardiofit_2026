'use client';

import { useState, useMemo } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ShieldCheck,
  CheckCircle2,
  Pencil,
  XCircle,
  Plus,
  AlertTriangle,
  Loader2,
  FileSignature,
  Lock,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { splReviewApi } from '@/lib/spl-api';
import type {
  DrugSignOff as DrugSignOffType,
  CompletenessReport,
  SPLFactType,
} from '@/types/spl-review';
import { FACT_TYPE_LABELS } from '@/types/spl-review';

// ============================================================================
// Drug Sign-Off — Mode 3 attestation
//
// The pharmacist reviews aggregate stats and signs off on the entire drug's
// extracted facts. This is the final gate before facts flow to downstream KBs.
//
// Requirements for sign-off:
// - All PENDING_REVIEW facts must be resolved (confirmed/edited/rejected)
// - Auto-approved sample check must be complete (5-10% spot-check)
// - Pharmacist provides attestation text
// ============================================================================

interface DrugSignOffProps {
  drugName: string;
  rxcui: string;
  report: CompletenessReport;
  /** Counts from the review queue */
  reviewStats: {
    confirmed: number;
    edited: number;
    rejected: number;
    added: number;
    pendingReview: number;
    autoApproved: number;
    sampleChecked: number;
    sampleErrors: number;
  };
  /** Existing sign-off (if previously signed) */
  existingSignOff?: DrugSignOffType | null;
  onComplete: () => void;
}

export function DrugSignOffPanel({
  drugName,
  rxcui,
  report,
  reviewStats,
  existingSignOff,
  onComplete,
}: DrugSignOffProps) {
  const queryClient = useQueryClient();
  const [attestation, setAttestation] = useState(
    existingSignOff?.attestation ||
    `I, the reviewing pharmacist, attest that I have reviewed the extracted clinical facts for ${drugName} and confirm that the approved facts are accurate, complete, and suitable for use in clinical decision support.`
  );
  const [error, setError] = useState<string | null>(null);

  // Check sign-off readiness
  const canSignOff = useMemo(() => {
    // All pending must be resolved
    if (reviewStats.pendingReview > 0) return false;
    // Attestation must be provided
    if (!attestation.trim()) return false;
    return true;
  }, [reviewStats.pendingReview, attestation]);

  // Blocking reasons
  const blockingReasons = useMemo(() => {
    const reasons: string[] = [];
    if (reviewStats.pendingReview > 0) {
      reasons.push(`${reviewStats.pendingReview} facts still pending review`);
    }
    if (!attestation.trim()) {
      reasons.push('Attestation text is required');
    }
    return reasons;
  }, [reviewStats.pendingReview, attestation]);

  // Fact type coverage
  const factTypeCoverage = useMemo(() => {
    const expected: SPLFactType[] = ['SAFETY_SIGNAL', 'INTERACTION'];
    const coverage: Record<SPLFactType, boolean> = {} as Record<SPLFactType, boolean>;
    expected.forEach((ft) => {
      coverage[ft] = (report.factCounts[ft] || 0) > 0;
    });
    return coverage;
  }, [report]);

  // Submit sign-off
  const mutation = useMutation({
    mutationFn: () =>
      splReviewApi.signOff.submit({
        drugName,
        rxcui,
        totalFacts: report.totalFacts,
        confirmed: reviewStats.confirmed,
        edited: reviewStats.edited,
        rejected: reviewStats.rejected,
        added: reviewStats.added,
        autoApprovedSampleSize: reviewStats.sampleChecked,
        autoApprovedSampleErrors: reviewStats.sampleErrors,
        factTypeCoverage,
        reviewerId: 'pharmacist',
        attestation: attestation.trim(),
        signedAt: new Date().toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spl-signoff', drugName] });
      queryClient.invalidateQueries({ queryKey: ['spl-completeness'] });
      onComplete();
    },
    onError: (err) => setError((err as Error).message),
  });

  const isAlreadySigned = !!existingSignOff;

  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      {/* Header */}
      <div className={cn(
        'px-5 py-4 border-b',
        isAlreadySigned ? 'bg-emerald-50 border-emerald-200' : 'bg-gray-50 border-gray-200'
      )}>
        <div className="flex items-center gap-3">
          {isAlreadySigned ? (
            <ShieldCheck className="h-6 w-6 text-emerald-600" />
          ) : (
            <FileSignature className="h-6 w-6 text-gray-600" />
          )}
          <div>
            <h3 className="text-base font-semibold text-gray-900">
              {isAlreadySigned ? 'Drug Signed Off' : 'Drug Sign-Off'}
            </h3>
            <p className="text-xs text-gray-500 mt-0.5 capitalize">
              {drugName} &middot; RxCUI {rxcui}
            </p>
          </div>
        </div>
      </div>

      <div className="p-5 space-y-5">
        {/* ── Review Summary ──────────────────────────────────────── */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 mb-3">Review Summary</h4>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {[
              { label: 'Confirmed', count: reviewStats.confirmed, icon: CheckCircle2, color: 'text-emerald-600' },
              { label: 'Edited', count: reviewStats.edited, icon: Pencil, color: 'text-blue-600' },
              { label: 'Rejected', count: reviewStats.rejected, icon: XCircle, color: 'text-red-600' },
              { label: 'Added', count: reviewStats.added, icon: Plus, color: 'text-purple-600' },
            ].map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.label} className="bg-gray-50 rounded-lg p-3">
                  <div className="flex items-center gap-1.5">
                    <Icon className={cn('h-3.5 w-3.5', item.color)} />
                    <span className="text-[10px] text-gray-500 uppercase">{item.label}</span>
                  </div>
                  <p className={cn('text-xl font-bold mt-1', item.color)}>{item.count}</p>
                </div>
              );
            })}
          </div>
        </div>

        {/* ── Spot-Check Summary ──────────────────────────────────── */}
        <div className="bg-blue-50 rounded-lg p-4">
          <h4 className="text-sm font-semibold text-blue-900 mb-2">Auto-Approved Spot-Check</h4>
          <div className="flex items-center gap-6 text-xs">
            <div>
              <span className="text-blue-600 font-medium">Total Auto-Approved:</span>
              <span className="ml-1 text-blue-900 font-bold">{reviewStats.autoApproved}</span>
            </div>
            <div>
              <span className="text-blue-600 font-medium">Sample Checked:</span>
              <span className="ml-1 text-blue-900 font-bold">{reviewStats.sampleChecked}</span>
            </div>
            <div>
              <span className="text-blue-600 font-medium">Errors Found:</span>
              <span className={cn(
                'ml-1 font-bold',
                reviewStats.sampleErrors > 0 ? 'text-red-600' : 'text-emerald-600'
              )}>
                {reviewStats.sampleErrors}
              </span>
            </div>
          </div>
          {reviewStats.sampleErrors > 0 && (
            <div className="mt-2 flex items-start gap-1.5 text-xs text-amber-700">
              <AlertTriangle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
              <span>
                Errors found in spot-check. Consider expanding sample size before sign-off.
              </span>
            </div>
          )}
        </div>

        {/* ── Blocking Reasons ────────────────────────────────────── */}
        {blockingReasons.length > 0 && !isAlreadySigned && (
          <div className="bg-red-50 rounded-lg p-4 border border-red-200">
            <h4 className="text-sm font-semibold text-red-800 flex items-center gap-2 mb-2">
              <Lock className="h-4 w-4" />
              Sign-Off Blocked
            </h4>
            <ul className="space-y-1">
              {blockingReasons.map((reason, i) => (
                <li key={i} className="text-xs text-red-700 flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">&#x2022;</span>
                  {reason}
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* ── Attestation ─────────────────────────────────────────── */}
        {!isAlreadySigned && (
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              Pharmacist Attestation
            </label>
            <textarea
              value={attestation}
              onChange={(e) => setAttestation(e.target.value)}
              rows={4}
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm text-gray-700 resize-none focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
            <p className="text-[10px] text-gray-400 mt-1">
              This attestation will be recorded per 21 CFR Part 11 audit trail requirements.
            </p>
          </div>
        )}

        {/* ── Previously signed info ──────────────────────────────── */}
        {isAlreadySigned && existingSignOff && (
          <div className="bg-emerald-50 rounded-lg p-4 border border-emerald-200">
            <p className="text-sm text-emerald-800 font-medium">
              Signed by {existingSignOff.reviewerId} on{' '}
              {new Date(existingSignOff.signedAt).toLocaleString()}
            </p>
            <p className="text-xs text-emerald-700 mt-2 italic">
              &ldquo;{existingSignOff.attestation}&rdquo;
            </p>
          </div>
        )}

        {/* ── Error ───────────────────────────────────────────────── */}
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
            {error}
          </div>
        )}

        {/* ── Submit Button ───────────────────────────────────────── */}
        {!isAlreadySigned && (
          <button
            onClick={() => mutation.mutate()}
            disabled={!canSignOff || mutation.isPending}
            className={cn(
              'w-full flex items-center justify-center gap-2 py-3 rounded-lg text-sm font-semibold transition-colors',
              canSignOff
                ? 'bg-emerald-600 text-white hover:bg-emerald-700'
                : 'bg-gray-200 text-gray-400 cursor-not-allowed'
            )}
          >
            {mutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <FileSignature className="h-4 w-4" />
            )}
            Sign Off on {drugName}
          </button>
        )}
      </div>
    </div>
  );
}
