'use client';

import { JobList } from '@/components/pipeline1/JobList';

export default function Pipeline1Page() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Span Review</h1>
        <p className="text-gray-500 mt-1">
          Review extracted text spans from Pipeline 1 guideline processing
        </p>
      </div>

      {/* Job List Table */}
      <div className="card overflow-hidden">
        <JobList />
      </div>
    </div>
  );
}
