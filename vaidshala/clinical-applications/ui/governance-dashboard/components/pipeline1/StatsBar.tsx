'use client';

import { useQuery } from '@tanstack/react-query';
import { pipeline1Api } from '@/lib/pipeline1-api';
import type { ExtractionJob, PageStats, ReviewTask } from '@/types/pipeline1';

interface StatsBarProps {
  jobId: string;
}

export function StatsBar({ jobId }: StatsBarProps) {
  const {
    data: job,
    isLoading: jobLoading,
  } = useQuery<ExtractionJob>({
    queryKey: ['pipeline1-job', jobId],
    queryFn: () => pipeline1Api.jobs.get(jobId),
  });

  const {
    data: pageStats,
    isLoading: statsLoading,
  } = useQuery<PageStats>({
    queryKey: ['pipeline1-page-stats', jobId],
    queryFn: () => pipeline1Api.pages.getStats(jobId),
  });

  const { data: tasks } = useQuery<ReviewTask[]>({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
  });

  if (jobLoading || statsLoading || !job || !pageStats) {
    return (
      <div className="bg-gray-900 text-white px-6 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-4 w-48 bg-white/10 rounded animate-pulse" />
            <div className="h-4 w-24 bg-white/10 rounded animate-pulse" />
          </div>
          <div className="flex items-center gap-2">
            <div className="h-9 w-28 bg-white/10 rounded animate-pulse" />
            <div className="h-9 w-28 bg-white/10 rounded animate-pulse" />
            <div className="h-9 w-28 bg-white/10 rounded animate-pulse" />
          </div>
        </div>
      </div>
    );
  }

  const fileName = job.sourcePdf.replace(/^.*\//, '');
  const truncatedId = job.jobId.slice(0, 12);
  const pagesReviewed = pageStats.totalPages - pageStats.pagesNoDecision;
  const totalTasks = tasks?.length ?? 0;
  const resolvedTasks = tasks?.filter((t) => t.status === 'RESOLVED').length ?? 0;

  return (
    <div className="bg-gray-900 text-white px-6 py-3">
      <div className="flex items-center justify-between">
        {/* Left side: guideline name, page range, job ID */}
        <div className="flex items-center gap-3 min-w-0">
          <span className="font-semibold text-sm truncate" title={fileName}>
            {fileName}
          </span>

          {job.pageRange && (
            <span className="inline-flex items-center text-xs font-medium bg-white/10 rounded px-2 py-0.5 whitespace-nowrap">
              pp. {job.pageRange}
            </span>
          )}

          <span
            className="text-xs text-gray-400 font-mono whitespace-nowrap"
            title={job.jobId}
          >
            {truncatedId}
          </span>
        </div>

        {/* Right side: metric tiles */}
        <div className="flex items-center gap-2 flex-shrink-0">
          <MetricTile label="Merged Spans" value={job.totalMergedSpans} />
          <MetricTile label="Sections" value={job.totalSections} />
          <MetricTile
            label="Pages Reviewed"
            value={`${pagesReviewed} / ${pageStats.totalPages}`}
          />
          {totalTasks > 0 && (
            <MetricTile
              label="Tasks"
              value={`${resolvedTasks} / ${totalTasks}`}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function MetricTile({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div className="bg-white/10 rounded px-3 py-1.5 text-center">
      <div className="text-[10px] uppercase tracking-wide text-gray-400 leading-tight">
        {label}
      </div>
      <div className="text-sm font-semibold leading-tight mt-0.5">{value}</div>
    </div>
  );
}
