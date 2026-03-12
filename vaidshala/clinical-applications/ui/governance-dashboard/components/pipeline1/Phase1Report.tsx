'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle, AlertTriangle, Shield } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import type { JobMetrics, ReviewTask, ExtractionJob } from '@/types/pipeline1';

// ============================================================================
// Phase1Report — Sprint 2 Governance Bookends
//
// Read-only renderer of the CoverageGuard report. The reviewer reads the
// gate-domain summary, span overview, and severity alerts, then acknowledges
// the report to unlock Phase 2.
// ============================================================================

interface Phase1ReportProps {
  jobId: string;
  job: ExtractionJob;
  onAcknowledge: () => void;
  acknowledged: boolean;
}

export function Phase1Report({ jobId, job, onAcknowledge, acknowledged }: Phase1ReportProps) {
  const [checked, setChecked] = useState(false);

  const { data: metrics } = useQuery<JobMetrics>({
    queryKey: ['pipeline1-metrics', jobId],
    queryFn: () => pipeline1Api.jobs.getMetrics(jobId),
  });

  const { data: tasks } = useQuery<ReviewTask[]>({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
  });

  const l1RecoveryCount = tasks?.filter((t) => t.taskType === 'L1_RECOVERY').length ?? 0;
  const disagreementCount = tasks?.filter((t) => t.taskType === 'DISAGREEMENT').length ?? 0;
  const spotCheckCount = tasks?.filter((t) => t.taskType === 'PASSAGE_SPOT_CHECK').length ?? 0;
  const criticalCount = tasks?.filter((t) => t.severity === 'critical').length ?? 0;
  const warningCount = tasks?.filter((t) => t.severity === 'warning').length ?? 0;

  // ---------------------------------------------------------------------------
  // Already acknowledged — show compact confirmation
  // ---------------------------------------------------------------------------

  if (acknowledged) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-8">
        <CheckCircle className="h-12 w-12 text-green-400 mb-4" />
        <h3 className="text-lg font-semibold text-gray-700 mb-2">Report Acknowledged</h3>
        <p className="text-sm text-gray-500">Proceed to Phase 2 — Fact Review</p>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Report view
  // ---------------------------------------------------------------------------

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-lg mx-auto px-6 py-10 space-y-6">
        {/* Header */}
        <div className="text-center space-y-2">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-blue-100 mb-2">
            <Shield className="h-6 w-6 text-blue-700" />
          </div>
          <h2 className="text-xl font-bold text-gray-900">CoverageGuard Report</h2>
          <p className="text-sm text-gray-500">
            {job.sourcePdf.replace(/^.*\//, '')} · {job.pipelineVersion} · {job.totalPages} pages
          </p>
        </div>

        {/* Gate Domain Summary */}
        <div className="rounded-lg border border-gray-200 overflow-hidden">
          <div className="px-4 py-2.5 bg-gray-50 border-b border-gray-200">
            <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wider">
              Gate Domain Summary
            </h3>
          </div>
          <div className="divide-y divide-gray-100">
            {[
              { label: 'C2 — L1 Recovery Spans', count: l1RecoveryCount, severity: l1RecoveryCount > 0 ? 'critical' as const : 'pass' as const },
              { label: 'Channel Disagreements', count: disagreementCount, severity: disagreementCount > 0 ? 'warning' as const : 'pass' as const },
              { label: 'Passage Spot-Checks', count: spotCheckCount, severity: spotCheckCount > 0 ? 'info' as const : 'pass' as const },
            ].map((row) => (
              <div key={row.label} className="flex items-center justify-between px-4 py-2.5">
                <span className="text-sm text-gray-700">{row.label}</span>
                <span
                  className={cn(
                    'text-sm font-bold',
                    row.severity === 'critical' && 'text-red-600',
                    row.severity === 'warning' && 'text-amber-600',
                    row.severity === 'info' && 'text-blue-600',
                    row.severity === 'pass' && 'text-green-600',
                  )}
                >
                  {row.count > 0 ? `${row.count} flagged` : 'PASS'}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Extraction Overview */}
        {metrics && (
          <div className="rounded-lg border border-gray-200 overflow-hidden">
            <div className="px-4 py-2.5 bg-gray-50 border-b border-gray-200">
              <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wider">
                Extraction Overview
              </h3>
            </div>
            <div className="px-4 py-3 space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Total Spans</span>
                <span className="font-bold text-gray-900">{metrics.totalSpans}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Critical Tasks</span>
                <span className={cn('font-bold', criticalCount > 0 ? 'text-red-600' : 'text-green-600')}>
                  {criticalCount}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Warning Tasks</span>
                <span className={cn('font-bold', warningCount > 0 ? 'text-amber-600' : 'text-green-600')}>
                  {warningCount}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Total Pages</span>
                <span className="font-bold text-gray-900">{job.totalPages}</span>
              </div>
            </div>
          </div>
        )}

        {/* Severity Alert */}
        {criticalCount > 0 && (
          <div className="flex items-start gap-2.5 p-3 rounded-lg border border-red-200 bg-red-50">
            <AlertTriangle className="h-4 w-4 text-red-600 mt-0.5 flex-shrink-0" />
            <p className="text-xs text-red-700">
              <strong>{criticalCount} critical task(s)</strong> require resolution before this job
              can be completed. L1 Recovery spans must be verified against the source PDF.
            </p>
          </div>
        )}

        {/* Acknowledge Checkbox + CTA */}
        <div className="space-y-3 pt-2">
          <label className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 bg-gray-50 cursor-pointer hover:bg-gray-100 transition-colors">
            <input
              type="checkbox"
              checked={checked}
              onChange={(e) => setChecked(e.target.checked)}
              className="mt-0.5 accent-blue-600"
            />
            <span className="text-sm text-gray-700">
              I have reviewed the CoverageGuard report and understand the flagged items
            </span>
          </label>

          <button
            onClick={onAcknowledge}
            disabled={!checked}
            className={cn(
              'w-full py-2.5 rounded-lg text-sm font-semibold transition-colors',
              checked
                ? 'bg-gray-900 text-white hover:bg-gray-800'
                : 'bg-gray-200 text-gray-400 cursor-not-allowed',
            )}
          >
            Proceed to Fact Review →
          </button>
        </div>
      </div>
    </div>
  );
}
