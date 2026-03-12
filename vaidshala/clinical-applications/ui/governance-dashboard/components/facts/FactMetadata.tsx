'use client';

import {
  formatDateTime,
  getConfidenceColor,
  getConfidenceLabel,
} from '@/lib/utils';
import type { ClinicalFact } from '@/types/governance';

interface FactMetadataProps {
  fact: ClinicalFact;
}

export function FactMetadata({ fact }: FactMetadataProps) {
  // Use flat structure from KB-0 backend
  const confidence = fact.confidenceScore ?? fact.confidence ?? 0;
  const content = fact.content as Record<string, unknown>;
  const evidenceLevel: string = String((content?.evidence_level as string) || fact.confidenceBand || 'UNKNOWN');
  const targetDrug = content?.target_drug as Record<string, unknown> | undefined;

  return (
    <div className="card">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">Metadata</h2>
      </div>

      <div className="card-body">
        <dl className="space-y-4">
          {/* Confidence Score */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Confidence Score
            </dt>
            <dd className="mt-1">
              <div className="flex items-center">
                <div className="flex-1 h-3 bg-gray-200 rounded-full overflow-hidden mr-3">
                  <div
                    className={`h-full rounded-full ${
                      confidence >= 0.95
                        ? 'bg-green-500'
                        : confidence >= 0.85
                        ? 'bg-blue-500'
                        : confidence >= 0.65
                        ? 'bg-amber-500'
                        : 'bg-red-500'
                    }`}
                    style={{ width: `${confidence * 100}%` }}
                  />
                </div>
                <span className="text-lg font-bold text-gray-900">
                  {(confidence * 100).toFixed(0)}%
                </span>
              </div>
              <p className={`mt-1 text-sm ${getConfidenceColor(confidence).split(' ')[0]}`}>
                {getConfidenceLabel(confidence)} Confidence
              </p>
            </dd>
          </div>

          {/* Evidence Level */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Evidence Level
            </dt>
            <dd className="mt-1 flex items-center">
              <span
                className={`inline-flex items-center justify-center h-8 w-8 rounded-full font-bold text-white ${
                  evidenceLevel === 'HIGH' || evidenceLevel === 'A'
                    ? 'bg-green-500'
                    : evidenceLevel === 'MEDIUM' || evidenceLevel === 'B'
                    ? 'bg-blue-500'
                    : evidenceLevel === 'LOW' || evidenceLevel === 'C'
                    ? 'bg-amber-500'
                    : 'bg-gray-500'
                }`}
              >
                {evidenceLevel.charAt(0)}
              </span>
              <span className="ml-2 text-sm text-gray-600">
                {evidenceLevel === 'HIGH' ? 'High-quality evidence'
                  : evidenceLevel === 'MEDIUM' ? 'Moderate evidence'
                  : evidenceLevel === 'LOW' ? 'Limited evidence'
                  : String(evidenceLevel)}
              </span>
            </dd>
          </div>

          {/* Source ID */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Source ID
            </dt>
            <dd className="mt-1">
              <span className="inline-flex items-center px-3 py-1 rounded-full bg-purple-100 text-purple-700 font-medium text-sm">
                {fact.sourceId}
              </span>
            </dd>
          </div>

          {/* Source Type */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Source Type
            </dt>
            <dd className="mt-1 text-gray-900">{fact.sourceType}</dd>
          </div>

          {/* RxCUI */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Drug RxCUI
            </dt>
            <dd className="mt-1 font-mono text-gray-900">{fact.rxcui}</dd>
          </div>

          {(targetDrug?.rxcui as string) && (
            <div>
              <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
                Interacting Drug RxCUI
              </dt>
              <dd className="mt-1 font-mono text-gray-900">
                {String(targetDrug?.rxcui)} ({String(targetDrug?.name)})
              </dd>
            </div>
          )}

          {/* Confidence Band */}
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Confidence Band
            </dt>
            <dd className="mt-1 text-gray-900">{fact.confidenceBand}</dd>
          </div>

          {/* Timestamps */}
          <div className="pt-4 border-t border-gray-100 space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Created</span>
              <span className="text-gray-900">{formatDateTime(fact.createdAt)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Updated</span>
              <span className="text-gray-900">{formatDateTime(fact.updatedAt)}</span>
            </div>
            {fact.activatedAt && (
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Activated</span>
                <span className="text-gray-900">
                  {formatDateTime(fact.activatedAt)}
                </span>
              </div>
            )}
          </div>
        </dl>
      </div>
    </div>
  );
}
