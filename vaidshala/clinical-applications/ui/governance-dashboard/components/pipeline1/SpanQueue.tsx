'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search, Filter, Loader2 } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { SpanReviewCard } from './SpanReviewCard';
import type { MergedSpan, SpanFilters, SpanReviewStatus } from '@/types/pipeline1';

interface SpanQueueProps {
  jobId: string;
  sectionFilter: string | null;
  selectedSpan: MergedSpan | null;
  onSelectSpan: (span: MergedSpan) => void;
}

const STATUS_OPTIONS: { value: SpanReviewStatus | ''; label: string }[] = [
  { value: '',          label: 'All' },
  { value: 'PENDING',   label: 'Pending' },
  { value: 'CONFIRMED', label: 'Confirmed' },
  { value: 'REJECTED',  label: 'Rejected' },
  { value: 'EDITED',    label: 'Edited' },
  { value: 'ADDED',     label: 'Added' },
];

export function SpanQueue({ jobId, sectionFilter, selectedSpan, onSelectSpan }: SpanQueueProps) {
  const [statusFilter, setStatusFilter] = useState<SpanReviewStatus | ''>('');
  const [search, setSearch] = useState('');
  const [disagreementOnly, setDisagreementOnly] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 30;

  const filters: SpanFilters = {
    ...(statusFilter ? { status: statusFilter as SpanReviewStatus } : {}),
    ...(sectionFilter ? { sectionId: sectionFilter } : {}),
    ...(search ? { search } : {}),
    ...(disagreementOnly ? { hasDisagreement: true } : {}),
  };

  const { data, isLoading } = useQuery({
    queryKey: ['pipeline1-spans', jobId, filters, page, pageSize],
    queryFn: () => pipeline1Api.spans.list(jobId, filters, page, pageSize),
  });

  return (
    <div className="flex flex-col h-full">
      {/* Filters */}
      <div className="p-3 border-b border-gray-200 space-y-2">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            placeholder="Search span text..."
            className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-blue-400"
          />
        </div>

        <div className="flex items-center gap-2">
          {/* Status filter */}
          <div className="flex items-center gap-1">
            <Filter className="h-3.5 w-3.5 text-gray-400" />
            <select
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value as SpanReviewStatus | ''); setPage(1); }}
              className="text-xs border border-gray-200 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-400"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>

          {/* Disagreement toggle */}
          <label className="flex items-center text-xs text-gray-600 cursor-pointer">
            <input
              type="checkbox"
              checked={disagreementOnly}
              onChange={(e) => { setDisagreementOnly(e.target.checked); setPage(1); }}
              className="mr-1.5"
            />
            Disagreements
          </label>

          {/* Count */}
          {data && (
            <span className="ml-auto text-xs text-gray-400">{data.total} spans</span>
          )}
        </div>
      </div>

      {/* Span list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
          </div>
        ) : !data?.items?.length ? (
          <div className="text-center py-12 text-sm text-gray-400">
            No spans match filters
          </div>
        ) : (
          data.items.map((span) => (
            <SpanReviewCard
              key={span.id}
              span={span}
              isSelected={selectedSpan?.id === span.id}
              onSelect={onSelectSpan}
            />
          ))
        )}
      </div>

      {/* Pagination */}
      {data && data.total > pageSize && (
        <div className="p-3 border-t border-gray-200 flex items-center justify-between">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page <= 1}
            className="text-xs px-3 py-1 border rounded disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-xs text-gray-500">
            Page {page} of {Math.ceil(data.total / pageSize)}
          </span>
          <button
            onClick={() => setPage(page + 1)}
            disabled={!data.hasMore}
            className="text-xs px-3 py-1 border rounded disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
