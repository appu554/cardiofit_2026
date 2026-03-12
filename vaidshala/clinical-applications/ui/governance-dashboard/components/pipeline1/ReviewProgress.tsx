'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { pipeline1Api } from '@/lib/pipeline1-api';
import type { JobMetrics, PageStats } from '@/types/pipeline1';

interface ReviewProgressProps {
  jobId: string;
}

const STATUS_COLORS: Record<string, { bg: string; label: string }> = {
  confirmed: { bg: 'bg-green-500', label: 'Confirmed' },
  rejected:  { bg: 'bg-red-500',   label: 'Rejected' },
  edited:    { bg: 'bg-blue-500',  label: 'Edited' },
  added:     { bg: 'bg-purple-500', label: 'Added' },
  pending:   { bg: 'bg-gray-300',  label: 'Pending' },
};

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      className={`w-4 h-4 text-gray-400 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  );
}

export function ReviewProgress({ jobId }: ReviewProgressProps) {
  const [spanOpen, setSpanOpen] = useState(false);
  const [pageOpen, setPageOpen] = useState(false);

  const { data: metrics } = useQuery<JobMetrics>({
    queryKey: ['pipeline1-metrics', jobId],
    queryFn: () => pipeline1Api.jobs.getMetrics(jobId),
    refetchInterval: 5000,
  });

  const { data: pageStats } = useQuery<PageStats>({
    queryKey: ['pipeline1-page-stats', jobId],
    queryFn: () => pipeline1Api.pages.getStats(jobId),
    refetchInterval: 5000,
  });

  if (!metrics) {
    return (
      <div className="p-4">
        <div className="h-4 bg-gray-200 rounded animate-pulse" />
      </div>
    );
  }

  const total = metrics.totalSpans || 1;
  const segments = [
    { key: 'confirmed', count: metrics.confirmed },
    { key: 'rejected',  count: metrics.rejected },
    { key: 'edited',    count: metrics.edited },
    { key: 'added',     count: metrics.added },
    { key: 'pending',   count: metrics.pending },
  ];

  const pagesReviewed = pageStats
    ? pageStats.totalPages - pageStats.pagesNoDecision
    : 0;

  return (
    <div className="p-4 space-y-2">
      {/* Span Progress */}
      <div>
        <button
          onClick={() => setSpanOpen(!spanOpen)}
          className="w-full flex items-center justify-between py-1.5 hover:bg-gray-50 rounded -mx-1 px-1"
        >
          <span className="text-sm font-medium text-gray-700">Span Progress</span>
          <div className="flex items-center gap-2">
            <span className="text-sm font-bold text-gray-900">
              {metrics.completionPct.toFixed(1)}%
            </span>
            <ChevronIcon open={spanOpen} />
          </div>
        </button>

        {spanOpen && (
          <div className="mt-2">
            {/* Stacked bar */}
            <div className="w-full h-3 rounded-full overflow-hidden flex bg-gray-100">
              {segments.map(({ key, count }) => {
                if (count === 0) return null;
                const pct = (count / total) * 100;
                return (
                  <div
                    key={key}
                    className={`${STATUS_COLORS[key].bg} transition-all duration-300`}
                    style={{ width: `${pct}%` }}
                    title={`${STATUS_COLORS[key].label}: ${count}`}
                  />
                );
              })}
            </div>

            {/* Legend */}
            <div className="flex flex-wrap gap-3 mt-2">
              {segments.map(({ key, count }) => (
                <div key={key} className="flex items-center text-xs text-gray-600">
                  <div className={`w-2 h-2 rounded-full ${STATUS_COLORS[key].bg} mr-1`} />
                  {STATUS_COLORS[key].label}: {count}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Page Progress */}
      {pageStats && pageStats.totalPages > 0 && (
        <div>
          <button
            onClick={() => setPageOpen(!pageOpen)}
            className="w-full flex items-center justify-between py-1.5 hover:bg-gray-50 rounded -mx-1 px-1"
          >
            <span className="text-sm font-medium text-gray-700">Page Progress</span>
            <div className="flex items-center gap-2">
              <span className="text-sm font-bold text-gray-900">
                {pagesReviewed} / {pageStats.totalPages}
              </span>
              <ChevronIcon open={pageOpen} />
            </div>
          </button>

          {pageOpen && (
            <div className="mt-2">
              <div className="w-full h-2 rounded-full overflow-hidden flex bg-gray-100">
                {pageStats.pagesAccepted > 0 && (
                  <div
                    className="bg-green-500 transition-all duration-300"
                    style={{ width: `${(pageStats.pagesAccepted / pageStats.totalPages) * 100}%` }}
                    title={`Accepted: ${pageStats.pagesAccepted}`}
                  />
                )}
                {pageStats.pagesFlagged > 0 && (
                  <div
                    className="bg-amber-500 transition-all duration-300"
                    style={{ width: `${(pageStats.pagesFlagged / pageStats.totalPages) * 100}%` }}
                    title={`Flagged: ${pageStats.pagesFlagged}`}
                  />
                )}
                {pageStats.pagesEscalated > 0 && (
                  <div
                    className="bg-red-500 transition-all duration-300"
                    style={{ width: `${(pageStats.pagesEscalated / pageStats.totalPages) * 100}%` }}
                    title={`Escalated: ${pageStats.pagesEscalated}`}
                  />
                )}
              </div>

              <div className="flex flex-wrap gap-3 mt-2">
                <div className="flex items-center text-xs text-gray-600">
                  <div className="w-2 h-2 rounded-full bg-green-500 mr-1" />
                  Accepted: {pageStats.pagesAccepted}
                </div>
                <div className="flex items-center text-xs text-gray-600">
                  <div className="w-2 h-2 rounded-full bg-amber-500 mr-1" />
                  Flagged: {pageStats.pagesFlagged}
                </div>
                <div className="flex items-center text-xs text-gray-600">
                  <div className="w-2 h-2 rounded-full bg-red-500 mr-1" />
                  Escalated: {pageStats.pagesEscalated}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
