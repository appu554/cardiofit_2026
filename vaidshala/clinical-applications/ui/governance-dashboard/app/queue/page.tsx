'use client';

import { useState, Suspense } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';
import { Filter, Search, RefreshCw, LayoutList, LayoutGrid } from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { QueueTable } from '@/components/queue/QueueTable';
import { DrugAccordion } from '@/components/queue/DrugAccordion';
import { QueueFilters } from '@/components/queue/QueueFilters';
import type { QueueFilters as QueueFiltersType, SortOptions } from '@/types/governance';

export default function QueuePageWrapper() {
  return (
    <Suspense fallback={<div className="p-6">Loading queue...</div>}>
      <QueuePage />
    </Suspense>
  );
}

function QueuePage() {
  const searchParams = useSearchParams();
  const initialPriority = searchParams.get('priority');
  const initialSlaStatus = searchParams.get('slaStatus');

  const [filters, setFilters] = useState<QueueFiltersType>({
    priority: initialPriority ? [initialPriority as any] : undefined,
    slaStatus: initialSlaStatus as any || undefined,
  });
  const [sort, setSort] = useState<SortOptions>({
    field: 'priority',
    direction: 'desc',
  });
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'drug' | 'list'>('drug');

  // For drug view: fetch ALL items to group client-side
  const { data: allData, isLoading: allLoading, refetch: refetchAll, isFetching: allFetching } = useQuery({
    queryKey: ['queue-all', filters, searchQuery],
    queryFn: () =>
      governanceApi.queue.getQueue(
        { ...filters, search: searchQuery || undefined },
        { field: 'priority', direction: 'desc' },
        1,
        1000 // fetch all for grouping
      ),
    enabled: viewMode === 'drug',
    refetchInterval: 30000,
  });

  // For list view: paginated fetch
  const { data: pageData, isLoading: pageLoading, refetch: refetchPage, isFetching: pageFetching } = useQuery({
    queryKey: ['queue', filters, sort, page, searchQuery],
    queryFn: () =>
      governanceApi.queue.getQueue(
        { ...filters, search: searchQuery || undefined },
        sort,
        page,
        20
      ),
    enabled: viewMode === 'list',
    refetchInterval: 30000,
  });

  const isLoading = viewMode === 'drug' ? allLoading : pageLoading;
  const isFetching = viewMode === 'drug' ? allFetching : pageFetching;
  const refetch = viewMode === 'drug' ? refetchAll : refetchPage;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Review Queue</h1>
          <p className="text-gray-500 mt-1">
            Clinical facts pending pharmacist review
          </p>
        </div>
        <div className="flex items-center space-x-3">
          {/* View Toggle */}
          <div className="flex items-center bg-gray-100 rounded-lg p-0.5">
            <button
              onClick={() => setViewMode('drug')}
              className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'drug'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              <LayoutGrid className="h-4 w-4 inline mr-1.5" />
              By Drug
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'list'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              <LayoutList className="h-4 w-4 inline mr-1.5" />
              List
            </button>
          </div>

          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="btn btn-outline btn-sm"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Search and Filters Bar */}
      <div className="card">
        <div className="p-4">
          <div className="flex items-center space-x-4">
            {/* Search */}
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search by drug name, RxCUI, or concept..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input pl-10 w-full"
              />
            </div>

            {/* Filter Toggle */}
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`btn ${showFilters ? 'btn-primary' : 'btn-outline'} btn-sm`}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
              {Object.values(filters).filter(Boolean).length > 0 && (
                <span className="ml-2 bg-white text-blue-600 rounded-full px-2 py-0.5 text-xs font-medium">
                  {Object.values(filters).filter(Boolean).length}
                </span>
              )}
            </button>
          </div>

          {/* Expanded Filters */}
          {showFilters && (
            <div className="mt-4 pt-4 border-t border-gray-100">
              <QueueFilters
                filters={filters}
                onFiltersChange={setFilters}
                onReset={() => setFilters({})}
              />
            </div>
          )}
        </div>
      </div>

      {/* Content */}
      {viewMode === 'drug' ? (
        <DrugAccordion
          items={allData?.items || []}
          isLoading={isLoading}
        />
      ) : (
        <QueueTable
          items={pageData?.items || []}
          isLoading={isLoading}
          sort={sort}
          onSortChange={setSort}
          page={page}
          pageSize={20}
          total={pageData?.total || 0}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}
