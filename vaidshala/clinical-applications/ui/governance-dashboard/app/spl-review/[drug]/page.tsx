'use client';

import { useMemo } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  AlertTriangle,
  Activity,
  Baby,
  Package,
  Beaker,
  Heart,
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
  FileText,
  Eye,
  Sparkles,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { GradeBadge, VerdictBadge, MetricBar } from '@/components/spl-review/GradeBadge';
import { splReviewApi } from '@/lib/spl-api';
import { governanceApi } from '@/lib/api';
import type {
  CompletenessReport,
  SPLFactType,
  SPLDerivedFact,
  GovernanceStatus,
  ExtractionMethod,
} from '@/types/spl-review';
import {
  FACT_TYPE_LABELS,
  EXTRACTION_METHOD_LABELS,
  EXPECTED_SECTIONS,
} from '@/types/spl-review';

// ============================================================================
// SPL Drug Detail Page
//
// Mode 2 entry — drills into a single drug's extracted facts. Shows:
// - Drug-level completeness summary (grade, verdict, metrics)
// - Fact type tabs with counts + status breakdown
// - Quick action cards: Review Pending, Spot-Check Approved
// The pharmacist selects a fact type tab to enter the fact-level review view.
// ============================================================================

// Fact type icons + color mapping
const FACT_TYPE_CONFIG: Record<
  SPLFactType,
  {
    icon: React.ComponentType<{ className?: string }>;
    color: string;
    bg: string;
    border: string;
    barColor: string;
  }
> = {
  SAFETY_SIGNAL: {
    icon: AlertTriangle,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    border: 'border-amber-200',
    barColor: 'bg-amber-500',
  },
  INTERACTION: {
    icon: Activity,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    barColor: 'bg-blue-500',
  },
  REPRODUCTIVE_SAFETY: {
    icon: Baby,
    color: 'text-pink-700',
    bg: 'bg-pink-50',
    border: 'border-pink-200',
    barColor: 'bg-pink-500',
  },
  FORMULARY: {
    icon: Package,
    color: 'text-purple-700',
    bg: 'bg-purple-50',
    border: 'border-purple-200',
    barColor: 'bg-purple-500',
  },
  LAB_REFERENCE: {
    icon: Beaker,
    color: 'text-cyan-700',
    bg: 'bg-cyan-50',
    border: 'border-cyan-200',
    barColor: 'bg-cyan-500',
  },
  ORGAN_IMPAIRMENT: {
    icon: Heart,
    color: 'text-red-700',
    bg: 'bg-red-50',
    border: 'border-red-200',
    barColor: 'bg-red-500',
  },
};

// ============================================================================
// Fact Type Tab Card
// ============================================================================

function FactTypeCard({
  factType,
  count,
  pendingCount,
  approvedCount,
  rejectedCount,
  drugName,
}: {
  factType: SPLFactType;
  count: number;
  pendingCount: number;
  approvedCount: number;
  rejectedCount: number;
  drugName: string;
}) {
  const config = FACT_TYPE_CONFIG[factType];
  const Icon = config.icon;
  const isEmpty = count === 0;
  const reviewedCount = approvedCount + rejectedCount;
  const progressPct = count > 0 ? (reviewedCount / count) * 100 : 0;

  return (
    <div
      className={cn(
        'rounded-xl border transition-all',
        isEmpty
          ? 'border-gray-100 bg-gray-50/50 opacity-60'
          : cn(config.border, 'bg-white hover:shadow-md')
      )}
    >
      <div className="p-4">
        {/* Header: Icon + Type Name + Count */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className={cn('p-2 rounded-lg', config.bg)}>
              <Icon className={cn('h-4 w-4', config.color)} />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-gray-900">
                {FACT_TYPE_LABELS[factType]}
              </h3>
              <p className="text-xs text-gray-400 mt-0.5">
                {count} fact{count !== 1 ? 's' : ''}
              </p>
            </div>
          </div>
          <span className={cn('text-2xl font-bold', isEmpty ? 'text-gray-300' : config.color)}>
            {count}
          </span>
        </div>

        {/* Status Breakdown */}
        {!isEmpty && (
          <div className="mt-3 space-y-2">
            <div className="flex items-center gap-3 text-xs">
              {pendingCount > 0 && (
                <span className="flex items-center gap-1 text-amber-600">
                  <Clock className="h-3 w-3" />
                  {pendingCount} pending
                </span>
              )}
              {approvedCount > 0 && (
                <span className="flex items-center gap-1 text-emerald-600">
                  <CheckCircle2 className="h-3 w-3" />
                  {approvedCount} approved
                </span>
              )}
              {rejectedCount > 0 && (
                <span className="flex items-center gap-1 text-red-600">
                  <XCircle className="h-3 w-3" />
                  {rejectedCount} rejected
                </span>
              )}
            </div>

            {/* Progress bar */}
            <div className="h-1 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={cn('h-full rounded-full transition-all', config.barColor)}
                style={{ width: `${Math.min(100, progressPct)}%` }}
              />
            </div>
          </div>
        )}

        {/* Empty state */}
        {isEmpty && (
          <p className="mt-2 text-xs text-gray-400 italic">
            Not extracted from this label
          </p>
        )}
      </div>

      {/* Action Footer */}
      {!isEmpty && (
        <div className="border-t border-gray-100 px-4 py-2.5 flex items-center justify-between">
          {pendingCount > 0 ? (
            <span className="flex items-center gap-1.5 text-xs text-amber-700 font-medium">
              <Eye className="h-3.5 w-3.5" />
              Review {pendingCount} pending
            </span>
          ) : (
            <span className="flex items-center gap-1.5 text-xs text-emerald-600">
              <CheckCircle2 className="h-3.5 w-3.5" />
              All reviewed
            </span>
          )}
          <span className="text-xs text-gray-400">
            {progressPct.toFixed(0)}% complete
          </span>
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Section Coverage Detail
// ============================================================================

function SectionCoverage({ report }: { report: CompletenessReport }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <h3 className="text-sm font-semibold text-gray-900 mb-3">
        LOINC Section Coverage
      </h3>
      <div className="space-y-2">
        {Object.entries(EXPECTED_SECTIONS).map(([loinc, name]) => {
          const covered = report.sectionsCovered.includes(loinc);
          return (
            <div
              key={loinc}
              className="flex items-center justify-between text-xs"
            >
              <div className="flex items-center gap-2">
                <div
                  className={cn(
                    'h-2 w-2 rounded-full',
                    covered ? 'bg-emerald-400' : 'bg-gray-200'
                  )}
                />
                <span className={covered ? 'text-gray-700' : 'text-gray-400'}>
                  {name}
                </span>
              </div>
              <span className="text-gray-400 font-mono">{loinc}</span>
            </div>
          );
        })}
      </div>
      <div className="mt-3 pt-3 border-t border-gray-100 flex items-center justify-between text-xs">
        <span className="text-gray-500">Coverage</span>
        <span className="font-medium text-gray-900">
          {report.sectionsCovered.length} / {Object.keys(EXPECTED_SECTIONS).length} sections
        </span>
      </div>
    </div>
  );
}

// ============================================================================
// Method Distribution Card
// ============================================================================

function MethodDistribution({ report }: { report: CompletenessReport }) {
  const methods = [
    { key: 'structured', label: 'Structured Parse', count: report.structuredCount, color: 'bg-emerald-500' },
    { key: 'llm', label: 'LLM Fallback', count: report.llmCount, color: 'bg-amber-500' },
    { key: 'grammar', label: 'DDI Grammar', count: report.grammarCount, color: 'bg-blue-500' },
  ].filter((m) => m.count > 0);

  const total = report.totalFacts;

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <h3 className="text-sm font-semibold text-gray-900 mb-3">
        Extraction Methods
      </h3>
      <div className="space-y-3">
        {methods.map((m) => (
          <div key={m.key}>
            <div className="flex items-center justify-between text-xs mb-1">
              <span className="text-gray-600">{m.label}</span>
              <span className="text-gray-900 font-medium">
                {m.count} ({total > 0 ? ((m.count / total) * 100).toFixed(1) : 0}%)
              </span>
            </div>
            <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={cn('h-full rounded-full', m.color)}
                style={{ width: `${total > 0 ? (m.count / total) * 100 : 0}%` }}
              />
            </div>
          </div>
        ))}
      </div>
      <div className="mt-3 pt-3 border-t border-gray-100">
        <MetricBar
          label="Deterministic Ratio"
          value={report.deterministicPct}
          thresholds={{ good: 85, fair: 50 }}
        />
      </div>
    </div>
  );
}

// ============================================================================
// Quick Actions Panel
// ============================================================================

function QuickActions({
  drugName,
  pendingCount,
  approvedCount,
  totalFacts,
}: {
  drugName: string;
  pendingCount: number;
  approvedCount: number;
  totalFacts: number;
}) {
  const reviewedPct = totalFacts > 0 ? ((totalFacts - pendingCount) / totalFacts) * 100 : 0;
  const encodedDrug = encodeURIComponent(drugName);

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <h3 className="text-sm font-semibold text-gray-900 mb-3">Review Actions</h3>
      <div className="space-y-3">
        {/* Review Pending */}
        {pendingCount > 0 ? (
          <Link
            href={`/spl-review/${encodedDrug}/review?status=PENDING_REVIEW`}
            className="block rounded-lg p-3 border border-amber-200 bg-amber-50 hover:bg-amber-100 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Eye className="h-4 w-4 text-amber-600" />
              <span className="text-sm font-medium text-amber-800">
                Review Pending Facts
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1">
              {pendingCount} facts await pharmacist review
            </p>
          </Link>
        ) : (
          <div className="rounded-lg p-3 border border-gray-100 bg-gray-50">
            <div className="flex items-center gap-2">
              <Eye className="h-4 w-4 text-gray-400" />
              <span className="text-sm font-medium text-gray-500">
                Review Pending Facts
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1">No pending facts</p>
          </div>
        )}

        {/* Spot-Check */}
        {approvedCount > 0 ? (
          <Link
            href={`/spl-review/${encodedDrug}/review?status=APPROVED`}
            className="block rounded-lg p-3 border border-blue-200 bg-blue-50 hover:bg-blue-100 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-blue-600" />
              <span className="text-sm font-medium text-blue-800">
                Spot-Check Approved
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1">
              Verify 5-10% sample of {approvedCount} auto-approved
            </p>
          </Link>
        ) : (
          <div className="rounded-lg p-3 border border-gray-100 bg-gray-50">
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-gray-400" />
              <span className="text-sm font-medium text-gray-500">
                Spot-Check Approved
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1">No auto-approved facts</p>
          </div>
        )}

        {/* Progress */}
        <div className="pt-2">
          <div className="flex items-center justify-between text-xs mb-1">
            <span className="text-gray-500">Overall Progress</span>
            <span className={cn('font-medium', reviewedPct >= 100 ? 'text-emerald-600' : 'text-blue-600')}>
              {reviewedPct.toFixed(0)}%
            </span>
          </div>
          <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                reviewedPct >= 100 ? 'bg-emerald-500' : 'bg-blue-500'
              )}
              style={{ width: `${Math.min(100, reviewedPct)}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Warnings Card
// ============================================================================

function WarningsCard({ warnings }: { warnings: string[] }) {
  if (warnings.length === 0) return null;

  return (
    <div className="bg-amber-50 rounded-xl border border-amber-200 p-4">
      <h3 className="text-sm font-semibold text-amber-800 mb-2 flex items-center gap-2">
        <AlertTriangle className="h-4 w-4" />
        Quality Warnings
      </h3>
      <ul className="space-y-1.5">
        {warnings.map((w, i) => (
          <li key={i} className="text-xs text-amber-700 flex items-start gap-2">
            <span className="text-amber-400 mt-0.5">&#x2022;</span>
            {w}
          </li>
        ))}
      </ul>
    </div>
  );
}

// ============================================================================
// Main Drug Detail Page
// ============================================================================

export default function DrugDetailPage() {
  const params = useParams<{ drug: string }>();
  const drugName = decodeURIComponent(params.drug);

  // Fetch completeness report for this drug
  const {
    data: report,
    isLoading: reportLoading,
    error: reportError,
  } = useQuery({
    queryKey: ['spl-completeness', drugName],
    queryFn: () => splReviewApi.completeness.getByDrug(drugName),
    refetchInterval: 60000,
  });

  // Fetch all derived facts for this drug to compute per-type/per-status counts
  const { data: factsData, isLoading: factsLoading } = useQuery({
    queryKey: ['spl-facts', drugName],
    queryFn: () =>
      governanceApi.queue.getQueue(
        { status: ['PENDING_REVIEW', 'APPROVED', 'REJECTED'] },
        { field: 'createdAt', direction: 'desc' },
        1,
        5000
      ),
    refetchInterval: 60000,
  });

  // Compute per-type and per-status counts
  const factBreakdown = useMemo(() => {
    const items = (factsData?.items || []).filter(
      (item) => item.drugName?.toLowerCase() === drugName.toLowerCase()
    );

    const byType: Record<string, { total: number; pending: number; approved: number; rejected: number }> = {};
    let totalPending = 0;
    let totalApproved = 0;

    for (const item of items) {
      const type = item.factType || 'UNKNOWN';
      if (!byType[type]) {
        byType[type] = { total: 0, pending: 0, approved: 0, rejected: 0 };
      }
      byType[type].total++;
      if (item.status === 'PENDING_REVIEW') {
        byType[type].pending++;
        totalPending++;
      } else if (item.status === 'APPROVED') {
        byType[type].approved++;
        totalApproved++;
      } else if (item.status === 'REJECTED') {
        byType[type].rejected++;
      }
    }

    return { byType, totalPending, totalApproved, totalItems: items.length };
  }, [factsData, drugName]);

  const isLoading = reportLoading || factsLoading;

  // ── Loading State ──────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-6 w-6 text-blue-500 animate-spin" />
          <p className="text-sm text-gray-500">
            Loading {drugName} review data...
          </p>
        </div>
      </div>
    );
  }

  // ── Error State ────────────────────────────────────────────────────────
  if (reportError || !report) {
    return (
      <div className="space-y-4">
        <Link
          href="/spl-review"
          className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Triage
        </Link>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <p className="text-red-600 font-medium">
              No completeness report found for &ldquo;{drugName}&rdquo;
            </p>
            <p className="text-gray-500 text-sm mt-1">
              This drug may not have been processed by the pipeline yet
            </p>
          </div>
        </div>
      </div>
    );
  }

  // ── All Fact Types ─────────────────────────────────────────────────────
  const ALL_FACT_TYPES: SPLFactType[] = [
    'SAFETY_SIGNAL',
    'INTERACTION',
    'REPRODUCTIVE_SAFETY',
    'FORMULARY',
    'LAB_REFERENCE',
    'ORGAN_IMPAIRMENT',
  ];

  // ── Main Render ────────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* Back Link + Drug Header */}
      <div>
        <Link
          href="/spl-review"
          className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 transition-colors mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Triage Dashboard
        </Link>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 capitalize">
              {report.drugName}
            </h1>
            <p className="text-gray-500 mt-1">
              RxCUI {report.rxcui} &middot; {report.totalFacts} facts extracted
            </p>
          </div>
          <div className="flex items-center gap-3">
            <GradeBadge grade={report.grade} size="lg" />
            <VerdictBadge verdict={report.gateVerdict} size="lg" />
          </div>
        </div>
      </div>

      {/* Quality Warnings */}
      <WarningsCard warnings={report.warnings} />

      {/* Quality Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <MetricBar
            label="MedDRA Match"
            value={report.meddraMatchRate}
            thresholds={{ good: 90, fair: 70 }}
          />
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <MetricBar
            label="Frequency Coverage"
            value={report.frequencyCovRate}
            thresholds={{ good: 50, fair: 20 }}
          />
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <MetricBar
            label="Interaction Quality"
            value={report.interactionQual}
            thresholds={{ good: 80, fair: 50 }}
          />
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <MetricBar
            label="Section Coverage"
            value={report.sectionCoveragePct}
            thresholds={{ good: 80, fair: 50 }}
          />
        </div>
      </div>

      {/* Fact Type Cards Grid */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-3">
          Facts by Type
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {ALL_FACT_TYPES.map((ft) => {
            const typeData = factBreakdown.byType[ft];
            const count = report.factCounts[ft] || 0;
            return (
              <FactTypeCard
                key={ft}
                factType={ft}
                count={count}
                pendingCount={typeData?.pending || 0}
                approvedCount={typeData?.approved || 0}
                rejectedCount={typeData?.rejected || 0}
                drugName={drugName}
              />
            );
          })}
        </div>
      </div>

      {/* Bottom Row: Section Coverage + Method Distribution + Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <SectionCoverage report={report} />
        <MethodDistribution report={report} />
        <QuickActions
          drugName={drugName}
          pendingCount={factBreakdown.totalPending}
          approvedCount={factBreakdown.totalApproved}
          totalFacts={report.totalFacts}
        />
      </div>
    </div>
  );
}
