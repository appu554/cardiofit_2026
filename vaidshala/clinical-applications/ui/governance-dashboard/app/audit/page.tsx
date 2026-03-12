'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import {
  Shield,
  RefreshCw,
  Filter,
  Search,
  Calendar,
  CheckCircle,
  XCircle,
  AlertTriangle,
  FileCheck,
  UserPlus,
  ArrowUpCircle,
  GitMerge,
  Clock,
  ChevronRight,
} from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { formatDateTime, cn } from '@/lib/utils';
import type { AuditEvent, AuditEventType } from '@/types/governance';

// Event type configuration
const eventConfig: Record<
  AuditEventType,
  { icon: React.ElementType; color: string; bgColor: string; label: string }
> = {
  FACT_CREATED: {
    icon: FileCheck,
    color: 'text-blue-600',
    bgColor: 'bg-blue-50',
    label: 'Fact Created',
  },
  FACT_SUBMITTED_FOR_REVIEW: {
    icon: ArrowUpCircle,
    color: 'text-amber-600',
    bgColor: 'bg-amber-50',
    label: 'Submitted for Review',
  },
  REVIEWER_ASSIGNED: {
    icon: UserPlus,
    color: 'text-purple-600',
    bgColor: 'bg-purple-50',
    label: 'Reviewer Assigned',
  },
  FACT_APPROVED: {
    icon: CheckCircle,
    color: 'text-green-600',
    bgColor: 'bg-green-50',
    label: 'Fact Approved',
  },
  FACT_REJECTED: {
    icon: XCircle,
    color: 'text-red-600',
    bgColor: 'bg-red-50',
    label: 'Fact Rejected',
  },
  FACT_ESCALATED: {
    icon: AlertTriangle,
    color: 'text-orange-600',
    bgColor: 'bg-orange-50',
    label: 'Escalated to CMO',
  },
  FACT_ACTIVATED: {
    icon: CheckCircle,
    color: 'text-green-600',
    bgColor: 'bg-green-50',
    label: 'Fact Activated',
  },
  FACT_SUPERSEDED: {
    icon: GitMerge,
    color: 'text-purple-600',
    bgColor: 'bg-purple-50',
    label: 'Fact Superseded',
  },
  CONFLICT_DETECTED: {
    icon: AlertTriangle,
    color: 'text-amber-600',
    bgColor: 'bg-amber-50',
    label: 'Conflict Detected',
  },
  CONFLICT_RESOLVED: {
    icon: CheckCircle,
    color: 'text-green-600',
    bgColor: 'bg-green-50',
    label: 'Conflict Resolved',
  },
  OVERRIDE_APPLIED: {
    icon: Shield,
    color: 'text-orange-600',
    bgColor: 'bg-orange-50',
    label: 'Override Applied',
  },
  OVERRIDE_EXPIRED: {
    icon: Clock,
    color: 'text-gray-600',
    bgColor: 'bg-gray-50',
    label: 'Override Expired',
  },
};

export default function AuditPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['audit', eventTypeFilter, dateFrom, dateTo, page],
    queryFn: () =>
      governanceApi.facts.getAuditLog(
        {
          eventType: eventTypeFilter || undefined,
          fromDate: dateFrom || undefined,
          toDate: dateTo || undefined,
        },
        page,
        50
      ),
    refetchInterval: 30000,
  });

  const events = data?.items || [];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / 50);

  const handleViewFact = (factId: string) => {
    router.push(`/facts/${factId}`);
  };

  // Group events by date
  const groupedEvents = events.reduce((acc, event) => {
    const date = new Date(event.createdAt).toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
    if (!acc[date]) acc[date] = [];
    acc[date].push(event);
    return acc;
  }, {} as Record<string, AuditEvent[]>);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Audit Log</h1>
          <p className="text-gray-500 mt-1">
            21 CFR Part 11 compliant audit trail for all governance actions
          </p>
        </div>
        <div className="flex items-center space-x-3">
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

      {/* Compliance Banner */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-center">
          <Shield className="h-6 w-6 text-blue-600 mr-3" />
          <div>
            <h3 className="font-medium text-blue-900">21 CFR Part 11 Compliance</h3>
            <p className="text-sm text-blue-700">
              All audit events include digital signatures, timestamps, and actor identification
              to meet FDA electronic records requirements.
            </p>
          </div>
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
                placeholder="Search by actor, fact ID, or reason..."
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
            </button>
          </div>

          {/* Expanded Filters */}
          {showFilters && (
            <div className="mt-4 pt-4 border-t border-gray-100 flex flex-wrap gap-4">
              {/* Event Type Filter */}
              <div className="flex-1 min-w-[200px]">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Event Type
                </label>
                <select
                  value={eventTypeFilter}
                  onChange={(e) => setEventTypeFilter(e.target.value)}
                  className="input w-full"
                >
                  <option value="">All Event Types</option>
                  {Object.entries(eventConfig).map(([key, config]) => (
                    <option key={key} value={key}>
                      {config.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Date From */}
              <div className="min-w-[180px]">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  From Date
                </label>
                <div className="relative">
                  <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="date"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                    className="input pl-10 w-full"
                  />
                </div>
              </div>

              {/* Date To */}
              <div className="min-w-[180px]">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  To Date
                </label>
                <div className="relative">
                  <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="date"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                    className="input pl-10 w-full"
                  />
                </div>
              </div>

              {/* Clear Filters */}
              <div className="flex items-end">
                <button
                  onClick={() => {
                    setEventTypeFilter('');
                    setDateFrom('');
                    setDateTo('');
                    setSearchQuery('');
                  }}
                  className="btn btn-outline btn-sm"
                >
                  Clear All
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Audit Timeline */}
      <div className="card">
        <div className="card-header">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Event Timeline</h2>
            <span className="text-sm text-gray-500">{total} total events</span>
          </div>
        </div>

        <div className="card-body">
          {isLoading ? (
            <div className="space-y-6">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="flex items-start space-x-4">
                  <div className="skeleton h-10 w-10 rounded-full" />
                  <div className="flex-1 space-y-2">
                    <div className="skeleton h-5 w-48" />
                    <div className="skeleton h-4 w-full" />
                    <div className="skeleton h-3 w-32" />
                  </div>
                </div>
              ))}
            </div>
          ) : events.length > 0 ? (
            <div className="space-y-8">
              {Object.entries(groupedEvents).map(([date, dateEvents]) => (
                <div key={date}>
                  <h3 className="text-sm font-medium text-gray-500 mb-4 sticky top-0 bg-white py-2">
                    {date}
                  </h3>
                  <div className="relative">
                    {/* Timeline line */}
                    <div className="absolute left-5 top-0 bottom-0 w-0.5 bg-gray-200" />

                    <ul className="space-y-4">
                      {dateEvents.map((event) => {
                        const config = eventConfig[event.eventType] || {
                          icon: FileCheck,
                          color: 'text-gray-600',
                          bgColor: 'bg-gray-50',
                          label: event.eventType,
                        };
                        const Icon = config.icon;

                        return (
                          <li
                            key={event.id}
                            className="relative pl-14 cursor-pointer group"
                            onClick={() => handleViewFact(event.factId)}
                          >
                            {/* Timeline dot */}
                            <div
                              className={cn(
                                'absolute left-0 p-2 rounded-full transition-transform group-hover:scale-110',
                                config.bgColor
                              )}
                            >
                              <Icon className={cn('h-5 w-5', config.color)} />
                            </div>

                            {/* Content */}
                            <div className="bg-gray-50 rounded-lg p-4 group-hover:bg-gray-100 transition-colors">
                              <div className="flex items-start justify-between">
                                <div className="flex-1">
                                  <div className="flex items-center space-x-2 mb-1">
                                    <span className="font-medium text-gray-900">
                                      {config.label}
                                    </span>
                                    <span className="text-xs text-gray-400 font-mono">
                                      {event.factId.slice(0, 8)}...
                                    </span>
                                  </div>
                                  <p className="text-sm text-gray-600">
                                    by{' '}
                                    <span className="font-medium">{event.actorName}</span>
                                    <span className="text-gray-400 mx-1">•</span>
                                    <span className="text-gray-500">{event.actorRole}</span>
                                  </p>

                                  {/* Reason */}
                                  {event.reason && (
                                    <p className="mt-2 text-sm text-gray-700 bg-white p-2 rounded border border-gray-100">
                                      "{event.reason}"
                                    </p>
                                  )}

                                  {/* State Change */}
                                  {event.previousState && event.newState && (
                                    <div className="mt-2 flex items-center text-xs">
                                      <span className="badge bg-gray-100 text-gray-600">
                                        {event.previousState}
                                      </span>
                                      <span className="mx-2 text-gray-400">→</span>
                                      <span className="badge bg-blue-100 text-blue-700">
                                        {event.newState}
                                      </span>
                                    </div>
                                  )}
                                </div>

                                <div className="flex flex-col items-end ml-4">
                                  <span className="text-xs text-gray-500">
                                    {formatDateTime(event.createdAt)}
                                  </span>
                                  <ChevronRight className="h-4 w-4 text-gray-400 mt-2 opacity-0 group-hover:opacity-100 transition-opacity" />
                                </div>
                              </div>

                              {/* Digital Signature */}
                              <div className="mt-3 pt-2 border-t border-gray-100 flex items-center text-xs text-gray-400">
                                <Shield className="h-3 w-3 mr-1" />
                                <span className="font-mono">
                                  Signature: {event.signature.slice(0, 20)}...
                                </span>
                              </div>
                            </div>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <Shield className="h-16 w-16 mx-auto mb-4 text-gray-300" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">No Audit Events Found</h3>
              <p className="text-gray-500">
                Audit events will appear here as governance actions are taken on clinical facts.
              </p>
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="bg-white px-4 py-3 flex items-center justify-between border-t border-gray-200 sm:px-6">
            <div className="flex-1 flex justify-between sm:hidden">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="btn btn-outline btn-sm"
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="btn btn-outline btn-sm"
              >
                Next
              </button>
            </div>
            <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
              <div>
                <p className="text-sm text-gray-700">
                  Showing{' '}
                  <span className="font-medium">{(page - 1) * 50 + 1}</span> to{' '}
                  <span className="font-medium">{Math.min(page * 50, total)}</span> of{' '}
                  <span className="font-medium">{total}</span> events
                </p>
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="btn btn-outline btn-sm"
                >
                  Previous
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                  className="btn btn-outline btn-sm"
                >
                  Next
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
