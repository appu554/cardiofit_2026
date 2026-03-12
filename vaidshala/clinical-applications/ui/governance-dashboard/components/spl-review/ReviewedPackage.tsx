'use client';

import { useMemo } from 'react';
import {
  Package,
  CheckCircle2,
  XCircle,
  Pencil,
  Plus,
  ArrowRight,
  Download,
  ShieldCheck,
  AlertTriangle,
  Clock,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { GradeBadge, VerdictBadge } from './GradeBadge';
import type {
  CompletenessReport,
  DrugSignOff,
  SPLFactType,
} from '@/types/spl-review';
import { FACT_TYPE_LABELS } from '@/types/spl-review';

// ============================================================================
// SPL Reviewed Package — Output contract summary
//
// After the pharmacist signs off, this component shows the complete
// "reviewed package" that will be projected to downstream KBs (KB-1..KB-6).
//
// Contents:
// - Drug identity + completeness grade + sign-off status
// - Fact disposition summary (confirmed/edited/rejected/added)
// - Per-KB projection targets (which facts go where)
// - Quality gate results
// - Export/download action
// ============================================================================

// KB target mapping
const KB_TARGETS: Record<string, { name: string; factTypes: SPLFactType[] }> = {
  'KB-1': { name: 'Drug Rules', factTypes: ['SAFETY_SIGNAL'] },
  'KB-4': { name: 'Patient Safety', factTypes: ['REPRODUCTIVE_SAFETY'] },
  'KB-5': { name: 'Drug Interactions', factTypes: ['INTERACTION'] },
  'KB-6': { name: 'Formulary', factTypes: ['FORMULARY'] },
  'KB-16': { name: 'Lab Interpretation', factTypes: ['LAB_REFERENCE'] },
};

interface ReviewedPackageProps {
  report: CompletenessReport;
  signOff: DrugSignOff;
  /** Per-type fact counts that were approved (will be projected) */
  approvedByType: Record<string, number>;
  /** Per-type fact counts that were rejected (excluded) */
  rejectedByType: Record<string, number>;
}

export function ReviewedPackage({
  report,
  signOff,
  approvedByType,
  rejectedByType,
}: ReviewedPackageProps) {
  // Compute projection targets
  const projections = useMemo(() => {
    return Object.entries(KB_TARGETS).map(([kb, config]) => {
      const factsToProject = config.factTypes.reduce(
        (sum, ft) => sum + (approvedByType[ft] || 0),
        0
      );
      const factsRejected = config.factTypes.reduce(
        (sum, ft) => sum + (rejectedByType[ft] || 0),
        0
      );
      return {
        kb,
        name: config.name,
        factTypes: config.factTypes,
        factsToProject,
        factsRejected,
      };
    }).filter((p) => p.factsToProject > 0 || p.factsRejected > 0);
  }, [approvedByType, rejectedByType]);

  const totalApproved = Object.values(approvedByType).reduce((s, c) => s + c, 0);
  const totalRejected = Object.values(rejectedByType).reduce((s, c) => s + c, 0);

  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-emerald-50 to-blue-50 px-5 py-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Package className="h-6 w-6 text-emerald-600" />
            <div>
              <h3 className="text-base font-semibold text-gray-900">
                SPL Reviewed Package
              </h3>
              <p className="text-xs text-gray-500 mt-0.5 capitalize">
                {report.drugName} &middot; RxCUI {report.rxcui}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <GradeBadge grade={report.grade} size="md" />
            <VerdictBadge verdict={report.gateVerdict} size="md" />
          </div>
        </div>
      </div>

      <div className="p-5 space-y-5">
        {/* ── Sign-Off Status ─────────────────────────────────────── */}
        <div className="flex items-center gap-3 bg-emerald-50 rounded-lg p-3 border border-emerald-200">
          <ShieldCheck className="h-5 w-5 text-emerald-600 shrink-0" />
          <div>
            <p className="text-sm font-medium text-emerald-800">
              Signed off by {signOff.reviewerId}
            </p>
            <p className="text-xs text-emerald-600">
              {new Date(signOff.signedAt).toLocaleString()}
            </p>
          </div>
        </div>

        {/* ── Fact Disposition Summary ─────────────────────────────── */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 mb-3">Fact Disposition</h4>
          <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
            {[
              { label: 'Total', count: report.totalFacts, icon: Package, color: 'text-gray-700', bg: 'bg-gray-50' },
              { label: 'Confirmed', count: signOff.confirmed, icon: CheckCircle2, color: 'text-emerald-600', bg: 'bg-emerald-50' },
              { label: 'Edited', count: signOff.edited, icon: Pencil, color: 'text-blue-600', bg: 'bg-blue-50' },
              { label: 'Rejected', count: signOff.rejected, icon: XCircle, color: 'text-red-600', bg: 'bg-red-50' },
              { label: 'Added', count: signOff.added, icon: Plus, color: 'text-purple-600', bg: 'bg-purple-50' },
            ].map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.label} className={cn('rounded-lg p-3', item.bg)}>
                  <div className="flex items-center gap-1.5">
                    <Icon className={cn('h-3.5 w-3.5', item.color)} />
                    <span className="text-[10px] text-gray-500 uppercase">{item.label}</span>
                  </div>
                  <p className={cn('text-lg font-bold mt-1', item.color)}>{item.count}</p>
                </div>
              );
            })}
          </div>
        </div>

        {/* ── KB Projection Targets ───────────────────────────────── */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 mb-3">
            KB Projection Targets
          </h4>
          <div className="space-y-2">
            {projections.map((proj) => (
              <div
                key={proj.kb}
                className="flex items-center justify-between bg-gray-50 rounded-lg px-4 py-3 border border-gray-100"
              >
                <div className="flex items-center gap-3">
                  <span className="text-xs font-bold text-gray-500 bg-white px-2 py-1 rounded border border-gray-200">
                    {proj.kb}
                  </span>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{proj.name}</p>
                    <p className="text-[10px] text-gray-400">
                      {proj.factTypes.map((ft) => FACT_TYPE_LABELS[ft]).join(', ')}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <p className="text-sm font-bold text-emerald-600">
                      {proj.factsToProject}
                    </p>
                    <p className="text-[10px] text-gray-400">to project</p>
                  </div>
                  {proj.factsRejected > 0 && (
                    <div className="text-right">
                      <p className="text-sm font-bold text-red-500">
                        {proj.factsRejected}
                      </p>
                      <p className="text-[10px] text-gray-400">rejected</p>
                    </div>
                  )}
                  <ArrowRight className="h-4 w-4 text-gray-300" />
                </div>
              </div>
            ))}
            {projections.length === 0 && (
              <p className="text-xs text-gray-400 italic text-center py-4">
                No facts approved for projection
              </p>
            )}
          </div>
        </div>

        {/* ── Spot-Check Results ───────────────────────────────────── */}
        <div className="bg-blue-50 rounded-lg p-4 border border-blue-200">
          <h4 className="text-sm font-semibold text-blue-900 mb-2">
            Auto-Approve Spot-Check
          </h4>
          <div className="flex items-center gap-6 text-xs">
            <div>
              <span className="text-blue-600">Sample Size:</span>
              <span className="ml-1 text-blue-900 font-bold">
                {signOff.autoApprovedSampleSize}
              </span>
            </div>
            <div>
              <span className="text-blue-600">Errors:</span>
              <span className={cn(
                'ml-1 font-bold',
                signOff.autoApprovedSampleErrors > 0 ? 'text-red-600' : 'text-emerald-600'
              )}>
                {signOff.autoApprovedSampleErrors}
              </span>
            </div>
            <div>
              <span className="text-blue-600">Error Rate:</span>
              <span className={cn(
                'ml-1 font-bold',
                signOff.autoApprovedSampleSize > 0 && signOff.autoApprovedSampleErrors / signOff.autoApprovedSampleSize > 0.05
                  ? 'text-red-600'
                  : 'text-emerald-600'
              )}>
                {signOff.autoApprovedSampleSize > 0
                  ? ((signOff.autoApprovedSampleErrors / signOff.autoApprovedSampleSize) * 100).toFixed(1)
                  : 0}%
              </span>
            </div>
          </div>
        </div>

        {/* ── Quality Gate ─────────────────────────────────────────── */}
        <div className={cn(
          'rounded-lg p-4 border',
          totalRejected / Math.max(report.totalFacts, 1) > 0.2
            ? 'bg-amber-50 border-amber-200'
            : 'bg-emerald-50 border-emerald-200'
        )}>
          <div className="flex items-center gap-2">
            {totalRejected / Math.max(report.totalFacts, 1) > 0.2 ? (
              <AlertTriangle className="h-5 w-5 text-amber-600" />
            ) : (
              <ShieldCheck className="h-5 w-5 text-emerald-600" />
            )}
            <div>
              <p className={cn(
                'text-sm font-semibold',
                totalRejected / Math.max(report.totalFacts, 1) > 0.2
                  ? 'text-amber-800'
                  : 'text-emerald-800'
              )}>
                {totalRejected / Math.max(report.totalFacts, 1) > 0.2
                  ? `High rejection rate (${((totalRejected / Math.max(report.totalFacts, 1)) * 100).toFixed(0)}%) — review pipeline quality`
                  : `Quality gate passed — ${totalApproved} facts ready for projection`}
              </p>
              <p className="text-xs text-gray-500 mt-0.5">
                Acceptance rate: {report.totalFacts > 0 ? ((totalApproved / report.totalFacts) * 100).toFixed(1) : 0}%
              </p>
            </div>
          </div>
        </div>

        {/* ── Export Action ─────────────────────────────────────────── */}
        <button className="w-full flex items-center justify-center gap-2 py-3 rounded-lg bg-blue-600 text-white text-sm font-semibold hover:bg-blue-700 transition-colors">
          <Download className="h-4 w-4" />
          Export Reviewed Package (JSON)
        </button>
      </div>
    </div>
  );
}
