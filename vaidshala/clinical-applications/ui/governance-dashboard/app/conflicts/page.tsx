'use client';

import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import {
  AlertTriangle,
  RefreshCw,
  GitMerge,
  ChevronRight,
  Shield,
  Check,
} from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { formatDateTime, cn, getConfidenceColor } from '@/lib/utils';
import type { ConflictGroup, ClinicalFact } from '@/types/governance';

// Resolution strategy labels
const resolutionLabels: Record<string, { label: string; description: string }> = {
  AUTHORITY_PRIORITY: {
    label: 'Authority Priority',
    description: 'Resolved by source authority ranking',
  },
  RECENCY: {
    label: 'Most Recent',
    description: 'Latest fact takes precedence',
  },
  MANUAL: {
    label: 'Manual Review Required',
    description: 'Human review needed to resolve conflict',
  },
};

// Fact type labels
const factTypeLabels: Record<string, string> = {
  INTERACTION: 'Drug Interaction',
  DRUG_INTERACTION: 'Drug-Drug Interaction',
  CONTRAINDICATION: 'Contraindication',
  DOSING_RULE: 'Dosing Rule',
  ALLERGY_CROSS_REACTIVITY: 'Allergy Cross-Reactivity',
  SAFETY_SIGNAL: 'Safety Signal',
  ORGAN_IMPAIRMENT: 'Organ Impairment',
  THERAPEUTIC_GUIDELINE: 'Therapeutic Guideline',
};

function ConflictCard({ conflict }: { conflict: ConflictGroup }) {
  const router = useRouter();
  const resolution = resolutionLabels[conflict.resolutionStrategy] || resolutionLabels.MANUAL;

  const handleViewFact = (factId: string) => {
    router.push(`/facts/${factId}`);
  };

  return (
    <div className="card border-amber-200 bg-amber-50/50">
      <div className="card-header bg-amber-50 border-amber-100">
        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <AlertTriangle className="h-5 w-5 text-amber-600 mr-2" />
            <div>
              <h3 className="font-semibold text-amber-900">{conflict.drugName}</h3>
              <p className="text-sm text-amber-700">
                {factTypeLabels[conflict.factType] || conflict.factType}
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <span className="badge bg-amber-100 text-amber-700">
              {conflict.facts.length} conflicting facts
            </span>
            <span
              className={cn(
                'badge',
                conflict.resolutionStrategy === 'MANUAL'
                  ? 'bg-red-100 text-red-700'
                  : 'bg-green-100 text-green-700'
              )}
            >
              {resolution.label}
            </span>
          </div>
        </div>
      </div>

      <div className="card-body">
        {/* Resolution Info */}
        <div className="mb-4 p-3 bg-white rounded-lg border border-amber-100">
          <div className="flex items-center justify-between">
            <div>
              <span className="text-sm text-gray-500">Resolution Strategy:</span>
              <span className="ml-2 font-medium text-gray-900">{resolution.label}</span>
              <p className="text-xs text-gray-500 mt-0.5">{resolution.description}</p>
            </div>
            {conflict.suggestedWinner && (
              <span className="text-sm text-green-600 flex items-center">
                <Check className="h-4 w-4 mr-1" />
                Winner suggested
              </span>
            )}
          </div>
          {conflict.resolutionReason && (
            <p className="text-sm text-gray-600 mt-2 pt-2 border-t border-gray-100">
              {conflict.resolutionReason}
            </p>
          )}
        </div>

        {/* Conflicting Facts List */}
        <div className="space-y-3">
          {conflict.facts.map((fact) => {
            const isSuggested = fact.id === conflict.suggestedWinner || fact.factId === conflict.suggestedWinner;
            const content = fact.content as Record<string, unknown>;
            const description = (content?.clinical_effect || content?.description || 'No description') as string;

            return (
              <div
                key={fact.id || fact.factId}
                onClick={() => handleViewFact(fact.factId || fact.id || '')}
                className={cn(
                  'p-4 rounded-lg border-2 cursor-pointer transition-all hover:shadow-md',
                  isSuggested
                    ? 'border-green-300 bg-green-50'
                    : 'border-gray-200 bg-white hover:border-blue-200'
                )}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2 mb-2">
                      {isSuggested && (
                        <span className="badge bg-green-100 text-green-700 flex items-center">
                          <GitMerge className="h-3 w-3 mr-1" />
                          Suggested Winner
                        </span>
                      )}
                      <span className="badge bg-purple-100 text-purple-700">
                        {fact.sourceId || fact.sourceAuthority}
                      </span>
                      <span className="badge bg-gray-100 text-gray-600">
                        {fact.status}
                      </span>
                    </div>

                    <p className="text-sm text-gray-900 line-clamp-2">{description}</p>

                    <div className="flex items-center space-x-4 mt-2 text-xs text-gray-500">
                      <span>
                        Confidence:{' '}
                        <span
                          className={cn(
                            'font-medium',
                            getConfidenceColor(fact.confidenceScore || fact.confidence || 0).split(' ')[0]
                          )}
                        >
                          {((fact.confidenceScore || fact.confidence || 0) * 100).toFixed(0)}%
                        </span>
                      </span>
                      <span>Created: {formatDateTime(fact.createdAt)}</span>
                      <span className="font-mono">{(fact.factId || fact.id || '').slice(0, 8)}...</span>
                    </div>
                  </div>
                  <ChevronRight className="h-5 w-5 text-gray-400 ml-2 flex-shrink-0" />
                </div>
              </div>
            );
          })}
        </div>

        {/* Manual Review Notice */}
        {conflict.resolutionStrategy === 'MANUAL' && (
          <div className="mt-4 p-3 bg-amber-100 border border-amber-200 rounded-lg">
            <div className="flex items-start">
              <Shield className="h-5 w-5 text-amber-600 mr-2 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-amber-800">
                <strong>Manual review required:</strong> The system could not automatically
                determine a winner. Please review all conflicting facts and approve the
                correct one. Approving one fact will automatically supersede the others.
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default function ConflictsPage() {
  const { data: conflicts, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['conflicts'],
    queryFn: () => governanceApi.facts.getAllConflicts(),
    refetchInterval: 60000,
  });

  // Count stats
  const manualReviewCount = conflicts?.filter((c) => c.resolutionStrategy === 'MANUAL').length || 0;
  const autoResolvedCount = conflicts?.filter((c) => c.resolutionStrategy !== 'MANUAL').length || 0;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Conflict Resolution</h1>
          <p className="text-gray-500 mt-1">
            Review and resolve conflicting clinical facts
          </p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="btn btn-outline btn-sm"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card p-4">
          <div className="flex items-center">
            <div className="p-2 bg-amber-100 rounded-lg mr-3">
              <AlertTriangle className="h-5 w-5 text-amber-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{conflicts?.length || 0}</p>
              <p className="text-sm text-gray-500">Total Conflicts</p>
            </div>
          </div>
        </div>
        <div className="card p-4">
          <div className="flex items-center">
            <div className="p-2 bg-red-100 rounded-lg mr-3">
              <Shield className="h-5 w-5 text-red-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{manualReviewCount}</p>
              <p className="text-sm text-gray-500">Needs Manual Review</p>
            </div>
          </div>
        </div>
        <div className="card p-4">
          <div className="flex items-center">
            <div className="p-2 bg-green-100 rounded-lg mr-3">
              <Check className="h-5 w-5 text-green-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{autoResolvedCount}</p>
              <p className="text-sm text-gray-500">Auto-Resolved</p>
            </div>
          </div>
        </div>
      </div>

      {/* Conflicts List */}
      {isLoading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card p-6">
              <div className="skeleton h-6 w-48 mb-4" />
              <div className="skeleton h-24 w-full" />
            </div>
          ))}
        </div>
      ) : conflicts && conflicts.length > 0 ? (
        <div className="space-y-6">
          {/* Manual Review Required - Show First */}
          {manualReviewCount > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
                <Shield className="h-5 w-5 text-red-600 mr-2" />
                Requires Manual Review ({manualReviewCount})
              </h2>
              <div className="space-y-4">
                {conflicts
                  .filter((c) => c.resolutionStrategy === 'MANUAL')
                  .map((conflict) => (
                    <ConflictCard key={conflict.groupId} conflict={conflict} />
                  ))}
              </div>
            </div>
          )}

          {/* Auto-Resolved */}
          {autoResolvedCount > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
                <GitMerge className="h-5 w-5 text-green-600 mr-2" />
                Auto-Resolved ({autoResolvedCount})
              </h2>
              <div className="space-y-4">
                {conflicts
                  .filter((c) => c.resolutionStrategy !== 'MANUAL')
                  .map((conflict) => (
                    <ConflictCard key={conflict.groupId} conflict={conflict} />
                  ))}
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="card p-12 text-center">
          <GitMerge className="h-16 w-16 mx-auto mb-4 text-gray-300" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">
            No Conflicts Found
          </h3>
          <p className="text-gray-500">
            All clinical facts are currently in agreement. Conflicts will appear here
            when multiple facts provide different guidance for the same clinical scenario.
          </p>
        </div>
      )}
    </div>
  );
}
