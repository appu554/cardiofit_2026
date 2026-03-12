'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import {
  Search,
  RefreshCw,
  Filter,
  FileText,
  CheckCircle,
  Clock,
  XCircle,
  AlertTriangle,
} from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { formatDateTime, cn } from '@/lib/utils';
import type { ClinicalFact, FactStatus, FactType } from '@/types/governance';

// Status badge configuration
const statusConfig: Record<FactStatus, { icon: React.ElementType; color: string; bgColor: string }> = {
  DRAFT: { icon: FileText, color: 'text-gray-600', bgColor: 'bg-gray-100' },
  PENDING_REVIEW: { icon: Clock, color: 'text-amber-600', bgColor: 'bg-amber-100' },
  APPROVED: { icon: CheckCircle, color: 'text-blue-600', bgColor: 'bg-blue-100' },
  ACTIVE: { icon: CheckCircle, color: 'text-green-600', bgColor: 'bg-green-100' },
  REJECTED: { icon: XCircle, color: 'text-red-600', bgColor: 'bg-red-100' },
  SUPERSEDED: { icon: AlertTriangle, color: 'text-purple-600', bgColor: 'bg-purple-100' },
  RETIRED: { icon: XCircle, color: 'text-gray-500', bgColor: 'bg-gray-100' },
};

// Fact type labels
const factTypeLabels: Record<string, string> = {
  INTERACTION: 'Drug Interaction',
  DRUG_INTERACTION: 'Drug-Drug Interaction',
  CONTRAINDICATION: 'Contraindication',
  DOSING_RULE: 'Dosing Rule',
  ALLERGY_CROSS_REACTIVITY: 'Allergy Cross-Reactivity',
  SAFETY_SIGNAL: 'Safety Signal',
  ORGAN_IMPAIRMENT: 'Organ Impairment',
  REPRODUCTIVE_SAFETY: 'Reproductive Safety',
  THERAPEUTIC_GUIDELINE: 'Therapeutic Guideline',
  LAB_REFERENCE: 'Lab Reference',
  LAB_DRUG_INTERACTION: 'Lab-Drug Interaction',
  FOOD_DRUG_INTERACTION: 'Food-Drug Interaction',
  PREGNANCY_CATEGORY: 'Pregnancy Category',
  RENAL_ADJUSTMENT: 'Renal Adjustment',
  HEPATIC_ADJUSTMENT: 'Hepatic Adjustment',
  GERIATRIC_CONSIDERATION: 'Geriatric Consideration',
  PEDIATRIC_DOSING: 'Pediatric Dosing',
  FORMULARY: 'Formulary',
};

export default function FactsPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['facts', statusFilter, typeFilter, searchQuery, page],
    queryFn: () =>
      governanceApi.facts.getAllFacts(
        {
          status: statusFilter || undefined,
          factType: typeFilter || undefined,
          search: searchQuery || undefined,
        },
        page,
        20
      ),
    refetchInterval: 60000,
  });

  const facts = data?.items || [];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / 20);

  const handleRowClick = (factId: string) => {
    router.push(`/facts/${factId}`);
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Clinical Facts</h1>
          <p className="text-gray-500 mt-1">
            Browse and manage all clinical knowledge facts
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

      {/* Search and Filters Bar */}
      <div className="card">
        <div className="p-4">
          <div className="flex items-center space-x-4">
            {/* Search */}
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search by drug name, RxCUI, or content..."
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
              {/* Status Filter */}
              <div className="flex-1 min-w-[200px]">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Status
                </label>
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="input w-full"
                >
                  <option value="">All Statuses</option>
                  <option value="DRAFT">Draft</option>
                  <option value="PENDING_REVIEW">Pending Review</option>
                  <option value="APPROVED">Approved</option>
                  <option value="ACTIVE">Active</option>
                  <option value="REJECTED">Rejected</option>
                  <option value="SUPERSEDED">Superseded</option>
                  <option value="RETIRED">Retired</option>
                </select>
              </div>

              {/* Fact Type Filter */}
              <div className="flex-1 min-w-[200px]">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Fact Type
                </label>
                <select
                  value={typeFilter}
                  onChange={(e) => setTypeFilter(e.target.value)}
                  className="input w-full"
                >
                  <option value="">All Types</option>
                  {Object.entries(factTypeLabels).map(([key, label]) => (
                    <option key={key} value={key}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Clear Filters */}
              <div className="flex items-end">
                <button
                  onClick={() => {
                    setStatusFilter('');
                    setTypeFilter('');
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

      {/* Facts Table */}
      <div className="card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Drug / Fact
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Confidence
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Source
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Updated
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {isLoading ? (
                // Loading skeleton
                [...Array(5)].map((_, i) => (
                  <tr key={i}>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-40" />
                    </td>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-28" />
                    </td>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-24" />
                    </td>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-16" />
                    </td>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-20" />
                    </td>
                    <td className="px-6 py-4">
                      <div className="skeleton h-5 w-24" />
                    </td>
                  </tr>
                ))
              ) : facts.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-gray-500">
                    <FileText className="h-12 w-12 mx-auto mb-3 text-gray-300" />
                    <p className="text-lg font-medium">No clinical facts found</p>
                    <p className="text-sm mt-1">
                      Try adjusting your filters or search query
                    </p>
                  </td>
                </tr>
              ) : (
                facts.map((fact) => {
                  const status = statusConfig[fact.status] || statusConfig.DRAFT;
                  const StatusIcon = status.icon;
                  const content = fact.content as Record<string, unknown>;
                  const targetDrug = content?.target_drug as Record<string, unknown> | undefined;
                  const interactingName = targetDrug?.name as string || fact.interactingDrugName;

                  return (
                    <tr
                      key={fact.factId || fact.id}
                      onClick={() => handleRowClick(fact.factId || fact.id || '')}
                      className="cursor-pointer hover:bg-gray-50 transition-colors"
                    >
                      <td className="px-6 py-4">
                        <div className="flex items-center">
                          <div>
                            <p className="font-medium text-gray-900">
                              {fact.drugName}
                              {interactingName && (
                                <span className="text-gray-500"> + {interactingName}</span>
                              )}
                            </p>
                            <p className="text-sm text-gray-500 font-mono">
                              {fact.rxcui || fact.drugRxcui}
                            </p>
                          </div>
                          {fact.hasConflict && (
                            <AlertTriangle className="h-4 w-4 text-amber-500 ml-2" />
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span className="badge bg-purple-100 text-purple-700">
                          {factTypeLabels[fact.factType] || fact.factType}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={cn(
                            'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
                            status.bgColor,
                            status.color
                          )}
                        >
                          <StatusIcon className="h-3 w-3 mr-1" />
                          {fact.status}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center">
                          <div className="w-16 bg-gray-200 rounded-full h-2 mr-2">
                            <div
                              className={cn(
                                'h-2 rounded-full',
                                (fact.confidenceScore || fact.confidence || 0) >= 0.9
                                  ? 'bg-green-500'
                                  : (fact.confidenceScore || fact.confidence || 0) >= 0.7
                                  ? 'bg-blue-500'
                                  : 'bg-amber-500'
                              )}
                              style={{
                                width: `${((fact.confidenceScore || fact.confidence || 0) * 100).toFixed(0)}%`,
                              }}
                            />
                          </div>
                          <span className="text-sm text-gray-600">
                            {((fact.confidenceScore || fact.confidence || 0) * 100).toFixed(0)}%
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-sm text-gray-600">
                          {fact.sourceId || fact.sourceAuthority || 'Unknown'}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {formatDateTime(fact.updatedAt || fact.createdAt)}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
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
                  <span className="font-medium">{(page - 1) * 20 + 1}</span> to{' '}
                  <span className="font-medium">{Math.min(page * 20, total)}</span> of{' '}
                  <span className="font-medium">{total}</span> facts
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
