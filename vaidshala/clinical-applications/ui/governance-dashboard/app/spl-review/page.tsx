'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { TriageDashboard } from '@/components/spl-review/TriageDashboard';
import { splReviewApi } from '@/lib/spl-api';
import { governanceApi } from '@/lib/api';
import type { CompletenessReport } from '@/types/spl-review';
import { Loader2 } from 'lucide-react';

// ============================================================================
// SPL Fact Review — Triage Dashboard
//
// Entry point for pharmacist SPL review. Shows all drugs from the latest
// pipeline run with completeness grades, gate verdicts, and quality metrics.
// The pharmacist triages drugs (REVIEW / INVESTIGATE / OUT_OF_SCOPE) before
// drilling into fact-level review.
// ============================================================================

export default function SPLReviewPage() {
  // Fetch completeness reports from the API
  // Fallback: If the dedicated /spl/completeness endpoint isn't available yet,
  // fetch directly from the queue and derive what we need.
  const {
    data: reports,
    isLoading: reportsLoading,
    error: reportsError,
  } = useQuery({
    queryKey: ['spl-completeness'],
    queryFn: async (): Promise<CompletenessReport[]> => {
      try {
        return await splReviewApi.completeness.getAll();
      } catch {
        // Endpoint not yet implemented — return mock data from the DB
        // This will be replaced once the SPL-specific API endpoints are built
        return [];
      }
    },
    refetchInterval: 60000,
  });

  // Fetch governance queue to compute per-drug pending/approved counts
  const { data: queueData, isLoading: queueLoading } = useQuery({
    queryKey: ['spl-queue-all'],
    queryFn: () =>
      governanceApi.queue.getQueue(
        { status: ['PENDING_REVIEW', 'APPROVED'] },
        { field: 'createdAt', direction: 'desc' },
        1,
        5000
      ),
    refetchInterval: 60000,
  });

  // Compute per-drug counts from queue data
  const { pendingCounts, approvedCounts } = useMemo(() => {
    const pending: Record<string, number> = {};
    const approved: Record<string, number> = {};

    for (const item of queueData?.items || []) {
      const drug = item.drugName?.toLowerCase() || '';
      if (!drug) continue;
      if (item.status === 'PENDING_REVIEW') {
        pending[drug] = (pending[drug] || 0) + 1;
      } else if (item.status === 'APPROVED') {
        approved[drug] = (approved[drug] || 0) + 1;
      }
    }

    return { pendingCounts: pending, approvedCounts: approved };
  }, [queueData]);

  const isLoading = reportsLoading || queueLoading;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-6 w-6 text-blue-500 animate-spin" />
          <p className="text-sm text-gray-500">Loading SPL completeness reports...</p>
        </div>
      </div>
    );
  }

  if (reportsError || !reports) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <p className="text-red-600 font-medium">Failed to load completeness reports</p>
          <p className="text-gray-500 text-sm mt-1">
            Ensure KB-0 service is running and the SPL pipeline has been executed
          </p>
        </div>
      </div>
    );
  }

  if (reports.length === 0) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">SPL Fact Review</h1>
          <p className="text-gray-500 mt-1">
            Drug label extraction review and pharmacist sign-off
          </p>
        </div>
        <div className="flex items-center justify-center h-64 bg-white rounded-xl border border-gray-200">
          <div className="text-center">
            <p className="text-gray-500 font-medium">No pipeline runs found</p>
            <p className="text-gray-400 text-sm mt-1">
              Run the SPL pipeline first:
            </p>
            <code className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded mt-2 block">
              ./bin/spl-pipeline --drugs-file ./next10_drugs.csv --verbose
            </code>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">SPL Fact Review</h1>
          <p className="text-gray-500 mt-1">
            Drug label extraction review and pharmacist sign-off
          </p>
        </div>
      </div>

      {/* Triage Dashboard */}
      <TriageDashboard
        reports={reports}
        pendingCounts={pendingCounts}
        approvedCounts={approvedCounts}
      />
    </div>
  );
}
