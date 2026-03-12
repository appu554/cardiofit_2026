'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle,
  XCircle,
  AlertTriangle,
  Loader2,
  Lock,
  Flag,
  Pencil,
  GitMerge,
  Info,
  EyeOff,
} from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { cn } from '@/lib/utils';
import { generateReferences, checkApprovalGate } from '@/lib/references';
import { RejectionReasons } from './RejectionReasons';
import { useAuth } from '@/hooks/useAuth';
import type { ClinicalFact, ReviewRequest, OverrideType, RejectionReasonCode } from '@/types/governance';

interface ReviewActionsProps {
  fact: ClinicalFact;
  onActionComplete: () => void;
}

type ActionType = 'approve' | 'approve-suppress' | 'reject' | 'escalate' | null;

const SUPPRESSION_REASONS = [
  { code: 'INFORMATIONAL_ONLY', label: 'Informational only — not clinically actionable' },
  { code: 'NOT_ACTIONABLE', label: 'Not actionable in current clinical context' },
  { code: 'REDUNDANT', label: 'Redundant — covered by existing rule' },
  { code: 'LOCAL_POLICY', label: 'Local formulary / policy exclusion' },
] as const;

export function ReviewActions({ fact, onActionComplete }: ReviewActionsProps) {
  const queryClient = useQueryClient();
  const { user, canReviewFacts } = useAuth();
  const [activeAction, setActiveAction] = useState<ActionType>(null);
  const [reason, setReason] = useState('');
  const [rejectionCodes, setRejectionCodes] = useState<RejectionReasonCode[]>([]);
  const [otherText, setOtherText] = useState('');
  const [suppressionReason, setSuppressionReason] = useState('');
  const [error, setError] = useState<string | null>(null);

  const factId = fact.factId || fact.id;

  // Approval gate check
  const references = generateReferences(fact);
  const gate = checkApprovalGate(references);

  // Mutations
  const approveMutation = useMutation({
    mutationFn: (request: ReviewRequest) => governanceApi.facts.approve(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['queue'] });
      queryClient.invalidateQueries({ queryKey: ['fact', factId] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      onActionComplete();
    },
    onError: (err) => setError((err as Error).message),
  });

  const rejectMutation = useMutation({
    mutationFn: (request: ReviewRequest) => governanceApi.facts.reject(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['queue'] });
      queryClient.invalidateQueries({ queryKey: ['fact', factId] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      onActionComplete();
    },
    onError: (err) => setError((err as Error).message),
  });

  const escalateMutation = useMutation({
    mutationFn: (request: ReviewRequest) => governanceApi.facts.escalate(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['queue'] });
      queryClient.invalidateQueries({ queryKey: ['fact', factId] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      onActionComplete();
    },
    onError: (err) => setError((err as Error).message),
  });

  const isLoading =
    approveMutation.isPending ||
    rejectMutation.isPending ||
    escalateMutation.isPending;

  const handleSubmit = () => {
    if ((activeAction === 'approve' || activeAction === 'approve-suppress') && !reason.trim()) {
      setError('Comment is required for approval');
      return;
    }
    if (activeAction === 'approve-suppress' && !suppressionReason) {
      setError('Select a suppression reason');
      return;
    }
    if (activeAction === 'reject' && rejectionCodes.length === 0) {
      setError('Select at least one rejection reason');
      return;
    }

    const rejectionText = activeAction === 'reject'
      ? rejectionCodes.join(', ') + (otherText ? `: ${otherText}` : '')
      : undefined;

    const suppressText = activeAction === 'approve-suppress'
      ? `[SUPPRESSED: ${suppressionReason}] ${reason.trim()}`
      : undefined;

    const request: ReviewRequest = {
      factId: factId!,
      reviewerId: user?.sub || 'unknown',
      action: 'APPROVE',
      reason: activeAction === 'reject' ? rejectionText!
        : activeAction === 'approve-suppress' ? suppressText!
        : reason.trim(),
      ...(activeAction === 'approve-suppress' && { suppress: true, suppressionReason }),
    };

    switch (activeAction) {
      case 'approve':
      case 'approve-suppress':
        approveMutation.mutate(request);
        break;
      case 'reject':
        rejectMutation.mutate(request);
        break;
      case 'escalate':
        escalateMutation.mutate(request);
        break;
    }
  };

  const handleCancel = () => {
    setActiveAction(null);
    setReason('');
    setRejectionCodes([]);
    setOtherText('');
    setSuppressionReason('');
    setError(null);
  };

  return (
    <div className="card h-full">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">Decision Controls</h2>
      </div>

      <div className="card-body">
        {!canReviewFacts ? (
          <div className="flex flex-col items-center justify-center py-8 text-gray-400">
            <Lock className="h-8 w-8 mb-2" />
            <p className="text-sm font-medium">View-only access</p>
            <p className="text-xs mt-1">Reviewers and Admins can take actions</p>
          </div>
        ) : !activeAction ? (
          <div className="space-y-3">
            {/* Approve — with gate */}
            <div className="relative group">
              <button
                onClick={() => setActiveAction('approve')}
                disabled={!gate.canApprove}
                className={cn(
                  'w-full btn flex items-center justify-center',
                  gate.canApprove
                    ? 'bg-green-600 text-white hover:bg-green-700'
                    : 'bg-gray-200 text-gray-400 cursor-not-allowed'
                )}
              >
                {gate.canApprove ? (
                  <CheckCircle className="h-5 w-5 mr-2" />
                ) : (
                  <Lock className="h-5 w-5 mr-2" />
                )}
                Approve
              </button>
              {!gate.canApprove && (
                <div className="mt-1.5 p-2 bg-amber-50 border border-amber-200 rounded text-xs text-amber-800">
                  <Info className="h-3.5 w-3.5 inline mr-1" />
                  Approval requires{' '}
                  {!gate.hasPrimary && 'primary source (DailyMed)'}
                  {!gate.hasPrimary && !gate.hasTerminology && ' and '}
                  {!gate.hasTerminology && 'terminology reference (MedDRA/RxNorm)'}
                </div>
              )}
            </div>

            {/* Approve but Suppress */}
            <div className="relative group">
              <button
                onClick={() => setActiveAction('approve-suppress')}
                disabled={!gate.canApprove}
                className={cn(
                  'w-full btn flex items-center justify-center',
                  gate.canApprove
                    ? 'bg-gray-600 text-white hover:bg-gray-700'
                    : 'bg-gray-200 text-gray-400 cursor-not-allowed'
                )}
              >
                <EyeOff className="h-5 w-5 mr-2" />
                Approve &amp; Suppress
              </button>
              <p className="text-xs text-gray-500 mt-1 text-center">
                Approve as valid but hide from CDS alerts
              </p>
            </div>

            {/* Edit (placeholder) */}
            <button
              disabled
              className="w-full btn btn-outline flex items-center justify-center opacity-50 cursor-not-allowed"
            >
              <Pencil className="h-5 w-5 mr-2" />
              Edit (future)
            </button>

            {/* Merge (placeholder) */}
            <button
              disabled
              className="w-full btn btn-outline flex items-center justify-center opacity-50 cursor-not-allowed"
            >
              <GitMerge className="h-5 w-5 mr-2" />
              Merge (future)
            </button>

            {/* Reject */}
            <button
              onClick={() => setActiveAction('reject')}
              className="w-full btn bg-red-600 text-white hover:bg-red-700 flex items-center justify-center"
            >
              <XCircle className="h-5 w-5 mr-2" />
              Reject
            </button>

            {/* Escalate */}
            <button
              onClick={() => setActiveAction('escalate')}
              className="w-full btn bg-amber-600 text-white hover:bg-amber-700 flex items-center justify-center"
            >
              <Flag className="h-5 w-5 mr-2" />
              Escalate to SME
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Action header */}
            <div
              className={cn(
                'p-3 rounded-lg text-sm font-medium flex items-center',
                activeAction === 'approve' && 'bg-green-50 text-green-700',
                activeAction === 'approve-suppress' && 'bg-gray-100 text-gray-700',
                activeAction === 'reject' && 'bg-red-50 text-red-700',
                activeAction === 'escalate' && 'bg-amber-50 text-amber-700'
              )}
            >
              {activeAction === 'approve' && <><CheckCircle className="h-5 w-5 mr-2" />Approving Fact</>}
              {activeAction === 'approve-suppress' && <><EyeOff className="h-5 w-5 mr-2" />Approve &amp; Suppress</>}
              {activeAction === 'reject' && <><XCircle className="h-5 w-5 mr-2" />Rejecting Fact</>}
              {activeAction === 'escalate' && <><Flag className="h-5 w-5 mr-2" />Escalating to SME</>}
            </div>

            {/* Reject: structured reasons */}
            {activeAction === 'reject' && (
              <RejectionReasons
                selected={rejectionCodes}
                onSelectionChange={setRejectionCodes}
                otherText={otherText}
                onOtherTextChange={setOtherText}
              />
            )}

            {/* Approve-Suppress: suppression reason */}
            {activeAction === 'approve-suppress' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Suppression Reason <span className="text-red-500">*</span>
                </label>
                <div className="space-y-1.5">
                  {SUPPRESSION_REASONS.map((sr) => (
                    <label
                      key={sr.code}
                      className={cn(
                        'flex items-start p-2 rounded-lg border cursor-pointer transition-colors text-sm',
                        suppressionReason === sr.code
                          ? 'border-gray-600 bg-gray-50'
                          : 'border-gray-200 hover:border-gray-300'
                      )}
                    >
                      <input
                        type="radio"
                        name="suppression"
                        value={sr.code}
                        checked={suppressionReason === sr.code}
                        onChange={() => setSuppressionReason(sr.code)}
                        className="mt-0.5 mr-2"
                      />
                      <span>{sr.label}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* Approve/Escalate/Suppress: free-text comment */}
            {activeAction !== 'reject' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Comment <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  rows={3}
                  className="input w-full"
                  placeholder={
                    activeAction === 'approve'
                      ? 'Clinical evidence supports this fact...'
                      : 'Requires SME review due to...'
                  }
                />
              </div>
            )}

            {/* Error */}
            {error && (
              <div className="p-3 bg-red-50 border border-red-100 rounded-lg text-red-700 text-sm">
                {error}
              </div>
            )}

            {/* Action buttons */}
            <div className="flex space-x-3 pt-2">
              <button onClick={handleCancel} disabled={isLoading} className="flex-1 btn btn-outline">
                Cancel
              </button>
              <button
                onClick={handleSubmit}
                disabled={isLoading}
                className={cn(
                  'flex-1 btn flex items-center justify-center',
                  activeAction === 'approve' && 'bg-green-600 text-white hover:bg-green-700',
                  activeAction === 'approve-suppress' && 'bg-gray-600 text-white hover:bg-gray-700',
                  activeAction === 'reject' && 'bg-red-600 text-white hover:bg-red-700',
                  activeAction === 'escalate' && 'bg-amber-600 text-white hover:bg-amber-700',
                  isLoading && 'opacity-50 cursor-not-allowed'
                )}
              >
                {isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Confirm
              </button>
            </div>

            <p className="text-xs text-gray-500 text-center pt-2">
              Recorded per 21 CFR Part 11 audit trail
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
