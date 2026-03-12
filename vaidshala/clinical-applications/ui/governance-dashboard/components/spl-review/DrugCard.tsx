'use client';

import Link from 'next/link';
import { cn } from '@/lib/utils';
import {
  AlertTriangle,
  ShieldCheck,
  Pill,
  Beaker,
  Baby,
  Package,
  Activity,
  Heart,
} from 'lucide-react';
import { GradeBadge, VerdictBadge, MetricBar } from './GradeBadge';
import type { CompletenessReport, SPLFactType, GateVerdict } from '@/types/spl-review';
import { FACT_TYPE_LABELS } from '@/types/spl-review';

// ============================================================================
// Fact Type Icon — returns the appropriate icon for each SPL fact type
// ============================================================================

const FACT_TYPE_ICONS: Record<SPLFactType, React.ComponentType<{ className?: string }>> = {
  SAFETY_SIGNAL: AlertTriangle,
  INTERACTION: Activity,
  REPRODUCTIVE_SAFETY: Baby,
  FORMULARY: Package,
  LAB_REFERENCE: Beaker,
  ORGAN_IMPAIRMENT: Heart,
};

// ============================================================================
// Drug Card — shows a single drug's completeness on the triage dashboard
// ============================================================================

interface DrugCardProps {
  report: CompletenessReport;
  pendingCount: number;
  approvedCount: number;
  reviewProgress: number; // 0-100
  isSelected?: boolean;
}

export function DrugCard({
  report,
  pendingCount,
  approvedCount,
  reviewProgress,
  isSelected,
}: DrugCardProps) {
  const isBlock = report.gateVerdict === 'BLOCK';
  const isEmpty = report.totalFacts === 0;

  return (
    <Link
      href={`/spl-review/${encodeURIComponent(report.drugName)}`}
      className={cn(
        'block rounded-xl border transition-all hover:shadow-md',
        isSelected
          ? 'ring-2 ring-blue-500 border-blue-300 shadow-md'
          : isBlock
            ? 'border-red-200 bg-red-50/30 hover:border-red-300'
            : 'border-gray-200 bg-white hover:border-gray-300'
      )}
    >
      {/* Header: Drug name + Grade + Verdict */}
      <div className="p-4 pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-gray-900 capitalize truncate">
              {report.drugName}
            </h3>
            <p className="text-xs text-gray-400 mt-0.5">
              RxCUI {report.rxcui}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <GradeBadge grade={report.grade} size="md" />
            <VerdictBadge verdict={report.gateVerdict} size="sm" />
          </div>
        </div>

        {/* Fact count summary */}
        <div className="mt-3 flex items-center gap-4 text-xs">
          <span className="text-gray-900 font-medium">
            {report.totalFacts} facts
          </span>
          {pendingCount > 0 && (
            <span className="text-amber-600 font-medium">
              {pendingCount} pending
            </span>
          )}
          {approvedCount > 0 && (
            <span className="text-emerald-600">
              {approvedCount} approved
            </span>
          )}
        </div>
      </div>

      {/* Fact Type Distribution — mini bar chart */}
      {!isEmpty && (
        <div className="px-4 pb-3">
          <div className="flex gap-0.5 h-2 rounded-full overflow-hidden bg-gray-100">
            {Object.entries(report.factCounts)
              .sort(([, a], [, b]) => b - a)
              .map(([type, count]) => {
                const pct = (count / report.totalFacts) * 100;
                const colors: Record<string, string> = {
                  SAFETY_SIGNAL: 'bg-amber-500',
                  INTERACTION: 'bg-blue-500',
                  REPRODUCTIVE_SAFETY: 'bg-pink-500',
                  FORMULARY: 'bg-purple-500',
                  LAB_REFERENCE: 'bg-cyan-500',
                  ORGAN_IMPAIRMENT: 'bg-red-500',
                };
                return (
                  <div
                    key={type}
                    className={cn('h-full', colors[type] || 'bg-gray-400')}
                    style={{ width: `${pct}%` }}
                    title={`${FACT_TYPE_LABELS[type as SPLFactType] || type}: ${count}`}
                  />
                );
              })}
          </div>
          {/* Legend (compact) */}
          <div className="flex flex-wrap gap-x-3 gap-y-1 mt-2">
            {Object.entries(report.factCounts)
              .sort(([, a], [, b]) => b - a)
              .map(([type, count]) => {
                const Icon = FACT_TYPE_ICONS[type as SPLFactType] || Pill;
                return (
                  <span key={type} className="flex items-center gap-1 text-[10px] text-gray-500">
                    <Icon className="h-3 w-3" />
                    {count}
                  </span>
                );
              })}
          </div>
        </div>
      )}

      {/* Empty state for BLOCK drugs */}
      {isEmpty && (
        <div className="px-4 pb-3">
          <p className="text-xs text-red-600 italic">
            No facts extracted — investigate source label
          </p>
        </div>
      )}

      {/* Quality Metrics */}
      {!isEmpty && (
        <div className="px-4 pb-3 space-y-2">
          <MetricBar
            label="Deterministic"
            value={report.deterministicPct}
            thresholds={{ good: 85, fair: 50 }}
          />
          <MetricBar
            label="MedDRA Match"
            value={report.meddraMatchRate}
            thresholds={{ good: 90, fair: 70 }}
          />
          <MetricBar
            label="Frequency"
            value={report.frequencyCovRate}
            thresholds={{ good: 50, fair: 20 }}
          />
        </div>
      )}

      {/* Section Coverage */}
      {!isEmpty && (
        <div className="px-4 pb-3">
          <div className="flex items-center justify-between text-xs">
            <span className="text-gray-500">Sections</span>
            <span className="text-gray-700 font-medium">
              {report.sectionsCovered.length}/7
            </span>
          </div>
          <div className="flex gap-1 mt-1">
            {['34084-4', '34071-1', '43685-7', '34073-7', '34068-7', '34088-5', '34069-5'].map(
              (loinc) => (
                <div
                  key={loinc}
                  className={cn(
                    'h-1.5 flex-1 rounded-full',
                    report.sectionsCovered.includes(loinc)
                      ? 'bg-emerald-400'
                      : 'bg-gray-200'
                  )}
                  title={loinc}
                />
              )
            )}
          </div>
        </div>
      )}

      {/* Review Progress Bar (bottom) */}
      <div className="px-4 pb-4">
        <div className="flex items-center justify-between text-xs mb-1">
          <span className="text-gray-500">Review</span>
          <span className={cn(
            'font-medium',
            reviewProgress >= 100 ? 'text-emerald-600' :
            reviewProgress > 0 ? 'text-blue-600' :
            'text-gray-400'
          )}>
            {reviewProgress.toFixed(0)}%
          </span>
        </div>
        <div className="h-1 bg-gray-100 rounded-full overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              reviewProgress >= 100 ? 'bg-emerald-500' : 'bg-blue-500'
            )}
            style={{ width: `${Math.min(100, reviewProgress)}%` }}
          />
        </div>
      </div>

      {/* Warnings */}
      {report.warnings.length > 0 && (
        <div className="px-4 pb-3 border-t border-gray-100">
          <div className="mt-2 flex items-start gap-1.5 text-[10px] text-amber-600">
            <AlertTriangle className="h-3 w-3 mt-0.5 shrink-0" />
            <span className="line-clamp-2">{report.warnings[0]}</span>
          </div>
        </div>
      )}
    </Link>
  );
}
