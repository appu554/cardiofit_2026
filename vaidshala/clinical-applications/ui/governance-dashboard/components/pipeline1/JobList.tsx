'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { FileText, Loader2, CheckCircle, Clock, Columns } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import type { ExtractionJob, JobStatus } from '@/types/pipeline1';

const STATUS_BADGE: Record<JobStatus, { color: string; label: string }> = {
  PENDING_REVIEW: { color: 'bg-amber-100 text-amber-800',   label: 'Pending Review' },
  IN_PROGRESS:    { color: 'bg-blue-100 text-blue-800',     label: 'In Progress' },
  COMPLETED:      { color: 'bg-green-100 text-green-800',   label: 'Completed' },
  ARCHIVED:       { color: 'bg-gray-100 text-gray-600',     label: 'Archived' },
};

export function JobList() {
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['pipeline1-jobs', page, pageSize],
    queryFn: () => pipeline1Api.jobs.list(page, pageSize),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!data?.items?.length) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-gray-400">
        <FileText className="h-12 w-12 mb-3" />
        <p className="text-lg font-medium">No extraction jobs found</p>
        <p className="text-sm mt-1">Run Pipeline 1 and ingest results to see jobs here</p>
      </div>
    );
  }

  return (
    <div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="text-left py-3 px-4 font-medium text-gray-500">Source PDF</th>
              <th className="text-left py-3 px-4 font-medium text-gray-500">Version</th>
              <th className="text-center py-3 px-4 font-medium text-gray-500">Spans</th>
              <th className="text-left py-3 px-4 font-medium text-gray-500 w-48">Progress</th>
              <th className="text-center py-3 px-4 font-medium text-gray-500">Status</th>
              <th className="text-left py-3 px-4 font-medium text-gray-500">Created</th>
              <th className="text-center py-3 px-4 font-medium text-gray-500">Actions</th>
            </tr>
          </thead>
          <tbody>
            {data.items.map((job) => (
              <JobRow key={job.jobId} job={job} />
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {data.total > pageSize && (
        <div className="flex items-center justify-between p-4 border-t border-gray-200">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page <= 1}
            className="btn btn-outline text-sm disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-gray-500">
            Page {page} of {Math.ceil(data.total / pageSize)}
          </span>
          <button
            onClick={() => setPage(page + 1)}
            disabled={!data.hasMore}
            className="btn btn-outline text-sm disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}

function JobRow({ job }: { job: ExtractionJob }) {
  const badge = STATUS_BADGE[job.status] || STATUS_BADGE.PENDING_REVIEW;
  const reviewed = job.spansConfirmed + job.spansRejected + job.spansEdited;
  const total = job.totalMergedSpans + job.spansAdded;
  const pct = total > 0 ? (reviewed / total) * 100 : 0;

  return (
    <tr className="border-b border-gray-100 hover:bg-gray-50 transition-colors">
      <td className="py-3 px-4">
        <Link
          href={`/pipeline1/${job.jobId}`}
          className="text-blue-600 hover:text-blue-800 font-medium hover:underline"
        >
          {job.sourcePdf.replace(/^.*\//, '')}
        </Link>
        {job.l1Tag && <span className="ml-2 text-xs text-gray-400">{job.l1Tag}</span>}
      </td>
      <td className="py-3 px-4 text-gray-600">{job.pipelineVersion}</td>
      <td className="py-3 px-4 text-center text-gray-600">{total}</td>
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          <div className="flex-1 h-2 bg-gray-100 rounded-full overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                pct >= 100 ? 'bg-green-500' : pct > 0 ? 'bg-blue-500' : 'bg-gray-200'
              )}
              style={{ width: `${Math.min(pct, 100)}%` }}
            />
          </div>
          <span className="text-xs text-gray-500 w-10 text-right">{pct.toFixed(0)}%</span>
        </div>
      </td>
      <td className="py-3 px-4 text-center">
        <span className={cn('text-xs font-medium px-2 py-1 rounded-full', badge.color)}>
          {badge.label}
        </span>
      </td>
      <td className="py-3 px-4 text-gray-500">
        <div className="flex items-center text-xs">
          <Clock className="h-3.5 w-3.5 mr-1" />
          {new Date(job.createdAt).toLocaleDateString()}
        </div>
      </td>
      <td className="py-3 px-4 text-center">
        <Link
          href={`/pipeline1/${job.jobId}/compare`}
          className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-600 hover:text-gray-900 bg-gray-100 hover:bg-gray-200 rounded transition-colors"
        >
          <Columns className="h-3 w-3" />
          Compare
        </Link>
      </td>
    </tr>
  );
}
