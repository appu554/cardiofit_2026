'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { Columns, FileText, Loader2, Clock } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import type { ExtractionJob, JobStatus } from '@/types/pipeline1';

const STATUS_BADGE: Record<JobStatus, { color: string; label: string }> = {
  PENDING_REVIEW: { color: 'bg-amber-100 text-amber-800', label: 'Pending Review' },
  IN_PROGRESS: { color: 'bg-blue-100 text-blue-800', label: 'In Progress' },
  COMPLETED: { color: 'bg-green-100 text-green-800', label: 'Completed' },
  ARCHIVED: { color: 'bg-gray-100 text-gray-600', label: 'Archived' },
};

export default function CompareLandingPage() {
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

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Compare View</h1>
        <p className="text-gray-500 mt-1">
          Select a job to compare extracted passages against the source document
        </p>
      </div>

      <div className="card overflow-hidden">
        {!data?.items?.length ? (
          <div className="flex flex-col items-center justify-center py-24 text-gray-400">
            <FileText className="h-12 w-12 mb-3" />
            <p className="text-lg font-medium">No extraction jobs found</p>
            <p className="text-sm mt-1">Run Pipeline 1 and ingest results to see jobs here</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {data.items.map((job) => (
              <JobCard key={job.jobId} job={job} />
            ))}
          </div>
        )}

        {data && data.total > pageSize && (
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
    </div>
  );
}

function JobCard({ job }: { job: ExtractionJob }) {
  const badge = STATUS_BADGE[job.status] || STATUS_BADGE.PENDING_REVIEW;
  const filename = job.sourcePdf.replace(/^.*\//, '');

  return (
    <div className="flex items-center justify-between px-5 py-4 hover:bg-gray-50 transition-colors">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-3">
          <p className="text-sm font-medium text-gray-900 truncate">{filename}</p>
          <span className={cn('text-[10px] font-medium px-2 py-0.5 rounded-full', badge.color)}>
            {badge.label}
          </span>
        </div>
        <div className="flex items-center gap-4 mt-1 text-xs text-gray-500">
          <span>v{job.pipelineVersion}</span>
          <span>{job.totalMergedSpans.toLocaleString()} spans</span>
          <span>{job.totalSections} sections</span>
          <span className="flex items-center">
            <Clock className="h-3 w-3 mr-1" />
            {new Date(job.createdAt).toLocaleDateString()}
          </span>
        </div>
      </div>
      <Link
        href={`/pipeline1/${job.jobId}/compare`}
        className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors shrink-0 ml-4"
      >
        <Columns className="h-4 w-4" />
        Open Compare
      </Link>
    </div>
  );
}
