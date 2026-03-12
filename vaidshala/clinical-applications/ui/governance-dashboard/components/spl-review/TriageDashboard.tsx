'use client';

import { useMemo, useState } from 'react';
import {
  CheckCircle,
  XCircle,
  AlertTriangle,
  BarChart3,
  Pill,
  Cpu,
  Brain,
  Shield,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { DrugCard } from './DrugCard';
import { GradeBadge, VerdictBadge } from './GradeBadge';
import type { CompletenessReport, GateVerdict, CompletenessGrade } from '@/types/spl-review';

// ============================================================================
// Pipeline Run Summary Banner
// ============================================================================

interface RunSummaryProps {
  reports: CompletenessReport[];
}

function RunSummary({ reports }: RunSummaryProps) {
  const stats = useMemo(() => {
    const total = reports.reduce((s, r) => s + r.totalFacts, 0);
    const structured = reports.reduce((s, r) => s + r.structuredCount, 0);
    const llm = reports.reduce((s, r) => s + r.llmCount, 0);
    const grammar = reports.reduce((s, r) => s + r.grammarCount, 0);
    const pass = reports.filter((r) => r.gateVerdict === 'PASS').length;
    const warn = reports.filter((r) => r.gateVerdict === 'WARNING').length;
    const block = reports.filter((r) => r.gateVerdict === 'BLOCK').length;
    const avgMeddra = reports.filter((r) => r.totalFacts > 0).length > 0
      ? reports.filter((r) => r.totalFacts > 0).reduce((s, r) => s + r.meddraMatchRate, 0) /
        reports.filter((r) => r.totalFacts > 0).length
      : 0;
    const deterministicPct = total > 0 ? ((structured + grammar) / total) * 100 : 0;

    return { total, structured, llm, grammar, pass, warn, block, avgMeddra, deterministicPct };
  }, [reports]);

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-5">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900">Pipeline Run Summary</h2>
          <p className="text-xs text-gray-500 mt-0.5">
            {reports.length} drugs processed &middot; {stats.total.toLocaleString()} facts extracted
          </p>
        </div>
        <div className="flex items-center gap-2">
          <VerdictBadge verdict="PASS" size="sm" />
          <span className="text-xs text-gray-500">{stats.pass}</span>
          {stats.warn > 0 && (
            <>
              <VerdictBadge verdict="WARNING" size="sm" />
              <span className="text-xs text-gray-500">{stats.warn}</span>
            </>
          )}
          {stats.block > 0 && (
            <>
              <VerdictBadge verdict="BLOCK" size="sm" />
              <span className="text-xs text-gray-500">{stats.block}</span>
            </>
          )}
        </div>
      </div>

      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <KpiCard
          icon={BarChart3}
          label="Total Facts"
          value={stats.total.toLocaleString()}
          color="blue"
        />
        <KpiCard
          icon={Cpu}
          label="Deterministic"
          value={`${stats.deterministicPct.toFixed(1)}%`}
          subValue={`${stats.structured + stats.grammar} structured`}
          color={stats.deterministicPct >= 75 ? 'emerald' : 'amber'}
        />
        <KpiCard
          icon={Brain}
          label="LLM Fallback"
          value={`${stats.llm}`}
          subValue={`${stats.total > 0 ? ((stats.llm / stats.total) * 100).toFixed(1) : 0}%`}
          color={stats.llm / Math.max(stats.total, 1) < 0.3 ? 'emerald' : 'amber'}
        />
        <KpiCard
          icon={Shield}
          label="MedDRA Match"
          value={`${stats.avgMeddra.toFixed(1)}%`}
          subValue="avg across drugs"
          color={stats.avgMeddra >= 90 ? 'emerald' : stats.avgMeddra >= 70 ? 'amber' : 'red'}
        />
      </div>

      {/* Method Distribution Bar */}
      <div className="mt-4">
        <div className="flex items-center justify-between text-xs text-gray-500 mb-1">
          <span>Extraction Method Distribution</span>
          <span>
            {stats.structured} table &middot; {stats.grammar} grammar &middot; {stats.llm} LLM
          </span>
        </div>
        <div className="flex h-2.5 rounded-full overflow-hidden bg-gray-100">
          {stats.total > 0 && (
            <>
              <div
                className="bg-emerald-500 h-full"
                style={{ width: `${(stats.structured / stats.total) * 100}%` }}
                title={`Structured: ${stats.structured}`}
              />
              <div
                className="bg-blue-500 h-full"
                style={{ width: `${(stats.grammar / stats.total) * 100}%` }}
                title={`Grammar: ${stats.grammar}`}
              />
              <div
                className="bg-amber-500 h-full"
                style={{ width: `${(stats.llm / stats.total) * 100}%` }}
                title={`LLM: ${stats.llm}`}
              />
            </>
          )}
        </div>
        <div className="flex items-center gap-4 mt-1.5 text-[10px] text-gray-400">
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-emerald-500" /> Structured
          </span>
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-blue-500" /> Grammar
          </span>
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-amber-500" /> LLM
          </span>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// KPI Card
// ============================================================================

function KpiCard({
  icon: Icon,
  label,
  value,
  subValue,
  color,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
  subValue?: string;
  color: 'blue' | 'emerald' | 'amber' | 'red';
}) {
  const colorMap = {
    blue: { bg: 'bg-blue-50', icon: 'text-blue-500', value: 'text-blue-900' },
    emerald: { bg: 'bg-emerald-50', icon: 'text-emerald-500', value: 'text-emerald-900' },
    amber: { bg: 'bg-amber-50', icon: 'text-amber-500', value: 'text-amber-900' },
    red: { bg: 'bg-red-50', icon: 'text-red-500', value: 'text-red-900' },
  };
  const c = colorMap[color];

  return (
    <div className={cn('rounded-lg p-3', c.bg)}>
      <div className="flex items-center gap-2">
        <Icon className={cn('h-4 w-4', c.icon)} />
        <span className="text-xs text-gray-500">{label}</span>
      </div>
      <p className={cn('text-xl font-bold mt-1', c.value)}>{value}</p>
      {subValue && <p className="text-[10px] text-gray-400 mt-0.5">{subValue}</p>}
    </div>
  );
}

// ============================================================================
// Triage Drug Grid Filter
// ============================================================================

type FilterMode = 'all' | 'pass' | 'block' | 'warning';

function FilterBar({
  mode,
  setMode,
  counts,
}: {
  mode: FilterMode;
  setMode: (m: FilterMode) => void;
  counts: Record<FilterMode, number>;
}) {
  const tabs: { key: FilterMode; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { key: 'all', label: 'All Drugs', icon: Pill },
    { key: 'pass', label: 'PASS', icon: CheckCircle },
    { key: 'block', label: 'BLOCK', icon: XCircle },
    { key: 'warning', label: 'WARNING', icon: AlertTriangle },
  ];

  return (
    <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          onClick={() => setMode(tab.key)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
            mode === tab.key
              ? 'bg-white text-gray-900 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          )}
        >
          <tab.icon className="h-3.5 w-3.5" />
          {tab.label}
          <span className={cn(
            'ml-1 px-1.5 py-0.5 rounded-full text-[10px]',
            mode === tab.key ? 'bg-gray-100 text-gray-700' : 'bg-transparent text-gray-400'
          )}>
            {counts[tab.key]}
          </span>
        </button>
      ))}
    </div>
  );
}

// ============================================================================
// Main Triage Dashboard Component
// ============================================================================

interface TriageDashboardProps {
  reports: CompletenessReport[];
  /** Per-drug pending review counts (drugName → count) */
  pendingCounts: Record<string, number>;
  /** Per-drug approved counts (drugName → count) */
  approvedCounts: Record<string, number>;
}

export function TriageDashboard({
  reports,
  pendingCounts,
  approvedCounts,
}: TriageDashboardProps) {
  const [filterMode, setFilterMode] = useState<FilterMode>('all');

  // Sort: BLOCK first (they need attention), then by fact count descending
  const sortedReports = useMemo(() => {
    const verdictOrder: Record<GateVerdict, number> = { BLOCK: 0, WARNING: 1, PASS: 2 };
    return [...reports].sort((a, b) => {
      const va = verdictOrder[a.gateVerdict] ?? 3;
      const vb = verdictOrder[b.gateVerdict] ?? 3;
      if (va !== vb) return va - vb;
      return b.totalFacts - a.totalFacts;
    });
  }, [reports]);

  // Filter
  const filteredReports = useMemo(() => {
    if (filterMode === 'all') return sortedReports;
    const verdictMap: Record<string, GateVerdict> = {
      pass: 'PASS',
      block: 'BLOCK',
      warning: 'WARNING',
    };
    return sortedReports.filter((r) => r.gateVerdict === verdictMap[filterMode]);
  }, [sortedReports, filterMode]);

  // Counts for filter bar
  const filterCounts = useMemo(() => ({
    all: reports.length,
    pass: reports.filter((r) => r.gateVerdict === 'PASS').length,
    block: reports.filter((r) => r.gateVerdict === 'BLOCK').length,
    warning: reports.filter((r) => r.gateVerdict === 'WARNING').length,
  }), [reports]);

  return (
    <div className="space-y-6">
      {/* Pipeline Summary */}
      <RunSummary reports={reports} />

      {/* Filter + Drug Grid */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Drug Triage</h2>
          <FilterBar mode={filterMode} setMode={setFilterMode} counts={filterCounts} />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
          {filteredReports.map((report) => {
            const pending = pendingCounts[report.drugName] || 0;
            const approved = approvedCounts[report.drugName] || 0;
            const total = pending + approved;
            const progress = total > 0 ? ((approved) / report.totalFacts) * 100 : 0;

            return (
              <DrugCard
                key={report.drugName}
                report={report}
                pendingCount={pending}
                approvedCount={approved}
                reviewProgress={report.totalFacts > 0 ? progress : 0}
              />
            );
          })}
        </div>

        {filteredReports.length === 0 && (
          <div className="text-center py-12 text-gray-400">
            <Pill className="h-8 w-8 mx-auto mb-2" />
            <p className="text-sm">No drugs match the selected filter</p>
          </div>
        )}
      </div>
    </div>
  );
}
