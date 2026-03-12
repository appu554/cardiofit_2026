'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { governanceApi } from '@/lib/api';
import { MetricsCards } from '@/components/dashboard/MetricsCards';
import { QueueSummary } from '@/components/dashboard/QueueSummary';
import { RecentActivity } from '@/components/dashboard/RecentActivity';
import { SlaOverview } from '@/components/dashboard/SlaOverview';
import { ExecutorStatus } from '@/components/dashboard/ExecutorStatus';
import type { GovernanceMetrics, QueueSummary as QueueSummaryType } from '@/types/governance';

export default function DashboardPage() {
  const { data: dashboard, isLoading: dashLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => governanceApi.dashboard.getDashboard(),
    refetchInterval: 30000,
  });

  // Also fetch full queue to compute real metrics (backend metrics count clinical_facts only,
  // but pipeline data lives in derived_facts)
  const { data: queueData, isLoading: queueLoading } = useQuery({
    queryKey: ['queue-all'],
    queryFn: () => governanceApi.queue.getQueue({}, { field: 'priority', direction: 'desc' }, 1, 1000),
    refetchInterval: 30000,
  });

  const isLoading = dashLoading || queueLoading;

  // Compute real metrics from queue items when backend metrics are all zeros
  const metrics = useMemo<GovernanceMetrics | undefined>(() => {
    const backendMetrics = dashboard?.metrics;
    const items = queueData?.items || dashboard?.recentItems || [];

    const backendTotal = (backendMetrics?.totalDraft || 0) + (backendMetrics?.totalApproved || 0) +
                         (backendMetrics?.totalActive || 0) + (backendMetrics?.totalSuperseded || 0) +
                         (backendMetrics?.pendingReview || 0);

    // If backend has real data, use it
    if (backendTotal > 0 && backendMetrics) return backendMetrics;

    // Otherwise compute from queue items
    if (items.length === 0) return backendMetrics;

    const draft = items.filter(i => i.status === 'DRAFT').length;
    const approved = items.filter(i => i.status === 'APPROVED').length;
    const active = items.filter(i => i.status === 'ACTIVE').length;
    const pending = items.filter(i => i.status === 'PENDING_REVIEW').length;
    const rejected = items.filter(i => i.status === 'REJECTED').length;
    const breached = items.filter(i => i.slaStatus === 'BREACHED').length;
    const atRisk = items.filter(i => i.slaStatus === 'AT_RISK').length;
    const critical = items.filter(i => i.reviewPriority === 'CRITICAL' && i.status === 'PENDING_REVIEW').length;
    const conflicts = items.filter(i => i.hasConflict).length;

    return {
      totalDraft: draft,
      totalApproved: approved,
      totalActive: active,
      totalSuperseded: rejected,
      pendingReview: pending,
      criticalPending: critical,
      breachedSLA: breached,
      atRiskSLA: atRisk,
      withConflicts: conflicts,
      generatedAt: new Date().toISOString(),
      totalFacts: items.length,
    };
  }, [dashboard?.metrics, queueData?.items, dashboard?.recentItems]);

  // Compute queue summary from all items
  const queueSummary = useMemo<QueueSummaryType | undefined>(() => {
    const items = queueData?.items || dashboard?.recentItems;
    if (!items) return undefined;

    return {
      critical: items.filter(i => i.reviewPriority === 'CRITICAL').length,
      high: items.filter(i => i.reviewPriority === 'HIGH').length,
      standard: items.filter(i => i.reviewPriority === 'STANDARD').length,
      low: items.filter(i => i.reviewPriority === 'LOW').length,
      overdue: items.filter(i => i.slaStatus === 'BREACHED').length,
    };
  }, [queueData?.items, dashboard?.recentItems]);

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <p className="text-red-600 font-medium">Failed to load dashboard</p>
          <p className="text-gray-500 text-sm mt-1">
            Please check if KB-0 service is running
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            Governance Dashboard
          </h1>
          <p className="text-gray-500 mt-1">
            Canonical Fact Store clinical knowledge governance
          </p>
        </div>
        <ExecutorStatus />
      </div>

      {/* Metrics Overview */}
      <MetricsCards metrics={metrics} isLoading={isLoading} />

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Queue Summary - Takes 2 columns */}
        <div className="lg:col-span-2">
          <QueueSummary summary={queueSummary} isLoading={isLoading} />
        </div>

        {/* SLA Overview */}
        <div>
          <SlaOverview metrics={metrics} isLoading={isLoading} />
        </div>
      </div>

      {/* Recent Activity - show recent queue items instead of audit events */}
      <RecentActivity
        items={queueData?.items?.slice(0, 10) || dashboard?.recentItems}
        isLoading={isLoading}
      />
    </div>
  );
}
