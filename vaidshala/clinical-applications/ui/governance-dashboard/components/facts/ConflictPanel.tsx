'use client';

import { AlertTriangle, Check, GitMerge } from 'lucide-react';
import {
  cn,
  getConfidenceColor,
  formatDateTime,
} from '@/lib/utils';
import type { ConflictGroup } from '@/types/governance';

interface ConflictPanelProps {
  conflicts?: ConflictGroup;
  isLoading: boolean;
  currentFactId: string;
}

export function ConflictPanel({
  conflicts,
  isLoading,
  currentFactId,
}: ConflictPanelProps) {
  if (isLoading) {
    return (
      <div className="card">
        <div className="card-header">
          <h2 className="text-lg font-semibold text-gray-900">
            Conflict Resolution
          </h2>
        </div>
        <div className="card-body">
          <div className="skeleton h-32 w-full" />
        </div>
      </div>
    );
  }

  if (!conflicts) {
    return null;
  }

  const resolutionLabel = {
    AUTHORITY_PRIORITY: 'Authority Priority',
    RECENCY: 'Most Recent',
    MANUAL: 'Manual Review Required',
  };

  return (
    <div className="card border-amber-200">
      <div className="card-header bg-amber-50 border-amber-100">
        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <AlertTriangle className="h-5 w-5 text-amber-600 mr-2" />
            <h2 className="text-lg font-semibold text-amber-900">
              Conflict Detected
            </h2>
          </div>
          <span className="badge bg-amber-100 text-amber-700">
            {conflicts.facts.length} conflicting facts
          </span>
        </div>
        <p className="text-sm text-amber-700 mt-1">
          Multiple facts exist for {conflicts.drugName} ({conflicts.factType})
        </p>
      </div>

      <div className="card-body">
        {/* Resolution Strategy */}
        <div className="mb-4 p-3 bg-gray-50 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <span className="text-sm text-gray-500">Resolution Strategy:</span>
              <span className="ml-2 font-medium text-gray-900">
                {resolutionLabel[conflicts.resolutionStrategy]}
              </span>
            </div>
            {conflicts.suggestedWinner && (
              <span className="text-sm text-green-600 flex items-center">
                <Check className="h-4 w-4 mr-1" />
                Suggested winner identified
              </span>
            )}
          </div>
          {conflicts.resolutionReason && (
            <p className="text-sm text-gray-600 mt-2">
              {conflicts.resolutionReason}
            </p>
          )}
        </div>

        {/* Conflicting Facts */}
        <div className="space-y-3">
          {conflicts.facts.map((fact) => {
            const isCurrent = fact.id === currentFactId;
            const isSuggested = fact.id === conflicts.suggestedWinner;

            return (
              <div
                key={fact.id}
                className={cn(
                  'p-4 rounded-lg border-2 transition-colors',
                  isCurrent
                    ? 'border-blue-300 bg-blue-50'
                    : isSuggested
                    ? 'border-green-300 bg-green-50'
                    : 'border-gray-200 bg-white'
                )}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      {isCurrent && (
                        <span className="badge bg-blue-100 text-blue-700">
                          Current
                        </span>
                      )}
                      {isSuggested && (
                        <span className="badge bg-green-100 text-green-700 flex items-center">
                          <GitMerge className="h-3 w-3 mr-1" />
                          Suggested Winner
                        </span>
                      )}
                      <span className="badge bg-purple-100 text-purple-700">
                        {fact.sourceAuthority}
                      </span>
                    </div>

                    <p className="text-sm text-gray-900 mt-2 line-clamp-2">
                      {String(fact.content.description ?? '')}
                    </p>

                    <div className="flex items-center space-x-4 mt-2 text-xs text-gray-500">
                      <span>
                        Confidence:{' '}
                        <span
                          className={cn(
                            'font-medium',
                            getConfidenceColor(fact.confidence ?? 0).split(' ')[0]
                          )}
                        >
                          {((fact.confidence ?? 0) * 100).toFixed(0)}%
                        </span>
                      </span>
                      <span>Created: {formatDateTime(fact.createdAt)}</span>
                      <span className="font-mono">{(fact.id ?? fact.factId).slice(0, 8)}...</span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Manual Resolution Notice */}
        {conflicts.resolutionStrategy === 'MANUAL' && (
          <div className="mt-4 p-3 bg-amber-50 border border-amber-100 rounded-lg">
            <p className="text-sm text-amber-800">
              <strong>Manual review required:</strong> The system could not
              automatically determine a winner. Please review all conflicting
              facts and approve the correct one. Approving one fact will
              automatically supersede the others.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
