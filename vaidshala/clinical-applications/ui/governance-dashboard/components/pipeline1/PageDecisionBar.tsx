'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Check, Flag, ArrowUpCircle, ArrowRight, Loader2, AlertTriangle } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import type { PageDecisionAction } from '@/types/pipeline1';

// ============================================================================
// Props
// ============================================================================

interface PageDecisionBarProps {
  jobId: string;
  activePage: number | null;
  onNextPage: () => void;
  currentDecision?: PageDecisionAction;
  /** Number of unresolved L1_RECOVERY tasks on this page — blocks Accept */
  unresolvedCriticalCount?: number;
}

// ============================================================================
// Decision Badge
// ============================================================================

const decisionStyles: Record<PageDecisionAction, string> = {
  ACCEPT: 'bg-green-100 text-green-700 border border-green-200',
  FLAG: 'bg-amber-100 text-amber-700 border border-amber-200',
  ESCALATE: 'bg-red-100 text-red-700 border border-red-200',
};

const decisionLabels: Record<PageDecisionAction, string> = {
  ACCEPT: 'Accepted',
  FLAG: 'Flagged',
  ESCALATE: 'Escalated',
};

function DecisionBadge({ decision }: { decision?: PageDecisionAction }) {
  if (!decision) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-500 border border-gray-200">
        No Decision
      </span>
    );
  }

  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        decisionStyles[decision],
      )}
    >
      {decisionLabels[decision]}
    </span>
  );
}

// ============================================================================
// Component
// ============================================================================

export function PageDecisionBar({
  jobId,
  activePage,
  onNextPage,
  currentDecision,
  unresolvedCriticalCount = 0,
}: PageDecisionBarProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const reviewerId = user?.sub || 'unknown';

  const decideMutation = useMutation({
    mutationFn: (action: PageDecisionAction) =>
      pipeline1Api.pages.decide(jobId, activePage!, { action, reviewerId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline1-pages', jobId] });
      queryClient.invalidateQueries({ queryKey: ['pipeline1-page-stats', jobId] });
    },
  });

  const isLoading = decideMutation.isPending;
  const acceptBlocked = unresolvedCriticalCount > 0;

  // ---- Disabled state: no page selected ----
  if (activePage === null) {
    return (
      <div className="bg-white px-6 py-3 flex items-center justify-between">
        <p className="text-sm text-gray-400">Select a page to make a decision</p>

        <div className="flex items-center gap-2">
          <button disabled className="btn btn-outline opacity-40 cursor-not-allowed">
            <Check className="h-4 w-4 mr-1.5" />
            Accept Page
          </button>
          <button disabled className="btn btn-outline opacity-40 cursor-not-allowed">
            <Flag className="h-4 w-4 mr-1.5" />
            Flag for Follow-up
          </button>
          <button disabled className="btn btn-outline opacity-40 cursor-not-allowed">
            <ArrowUpCircle className="h-4 w-4 mr-1.5" />
            Escalate
          </button>
          <div className="h-8 w-px bg-gray-300" aria-hidden="true" />
          <button disabled className="btn bg-gray-900 text-white opacity-40 cursor-not-allowed">
            Save &amp; Next
            <ArrowRight className="h-4 w-4 ml-1.5" />
          </button>
        </div>
      </div>
    );
  }

  // ---- Active state ----
  return (
    <div className="bg-white px-6 py-3 flex items-center justify-between">
      {/* Left side: page label + current decision badge + critical warning */}
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium text-gray-700">
          Page {activePage} Decision:
        </span>
        <DecisionBadge decision={currentDecision} />
        {acceptBlocked && (
          <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600">
            <AlertTriangle className="h-3.5 w-3.5" />
            {unresolvedCriticalCount} critical task{unresolvedCriticalCount > 1 ? 's' : ''} unresolved
          </span>
        )}
      </div>

      {/* Right side: action buttons */}
      <div className="flex items-center gap-2">
        {/* Accept Page — blocked when critical tasks are unresolved */}
        <button
          type="button"
          disabled={isLoading || acceptBlocked}
          onClick={() => decideMutation.mutate('ACCEPT')}
          className={cn(
            'inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
            'border-green-300 text-green-700 hover:bg-green-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 focus-visible:ring-offset-2',
            currentDecision === 'ACCEPT' && 'bg-green-50 ring-1 ring-green-400',
            (isLoading || acceptBlocked) && 'opacity-50 cursor-not-allowed',
          )}
          aria-label={acceptBlocked ? 'Accept blocked — resolve critical tasks first' : 'Accept page'}
          aria-pressed={currentDecision === 'ACCEPT'}
          title={acceptBlocked ? `Resolve ${unresolvedCriticalCount} critical L1 recovery task${unresolvedCriticalCount > 1 ? 's' : ''} before accepting` : undefined}
        >
          {isLoading && decideMutation.variables === 'ACCEPT' ? (
            <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
          ) : (
            <Check className="h-4 w-4 mr-1.5" />
          )}
          Accept Page
        </button>

        {/* Flag for Follow-up */}
        <button
          type="button"
          disabled={isLoading}
          onClick={() => decideMutation.mutate('FLAG')}
          className={cn(
            'inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
            'border-amber-300 text-amber-700 hover:bg-amber-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2',
            currentDecision === 'FLAG' && 'bg-amber-50 ring-1 ring-amber-400',
            isLoading && 'opacity-50 cursor-not-allowed',
          )}
          aria-label="Flag page for follow-up"
          aria-pressed={currentDecision === 'FLAG'}
        >
          {isLoading && decideMutation.variables === 'FLAG' ? (
            <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
          ) : (
            <Flag className="h-4 w-4 mr-1.5" />
          )}
          Flag for Follow-up
        </button>

        {/* Escalate */}
        <button
          type="button"
          disabled={isLoading}
          onClick={() => decideMutation.mutate('ESCALATE')}
          className={cn(
            'inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
            'border-red-300 text-red-700 hover:bg-red-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2',
            currentDecision === 'ESCALATE' && 'bg-red-50 ring-1 ring-red-400',
            isLoading && 'opacity-50 cursor-not-allowed',
          )}
          aria-label="Escalate page"
          aria-pressed={currentDecision === 'ESCALATE'}
        >
          {isLoading && decideMutation.variables === 'ESCALATE' ? (
            <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
          ) : (
            <ArrowUpCircle className="h-4 w-4 mr-1.5" />
          )}
          Escalate
        </button>

        {/* Vertical divider */}
        <div className="h-8 w-px bg-gray-300" aria-hidden="true" />

        {/* Save & Next */}
        <button
          type="button"
          onClick={onNextPage}
          disabled={isLoading}
          className={cn(
            'inline-flex items-center px-4 py-1.5 rounded-md text-sm font-medium transition-colors',
            'bg-gray-900 text-white hover:bg-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900 focus-visible:ring-offset-2',
            isLoading && 'opacity-50 cursor-not-allowed',
          )}
        >
          Save &amp; Next
          <ArrowRight className="h-4 w-4 ml-1.5" />
        </button>
      </div>
    </div>
  );
}
