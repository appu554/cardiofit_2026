'use client';

import { useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  ChevronUp,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { QueueItem, SortOptions } from '@/types/governance';

interface QueueTableProps {
  items: QueueItem[];
  isLoading: boolean;
  sort: SortOptions;
  onSortChange: (sort: SortOptions) => void;
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
}

// Fact type badge colors
const FACT_TYPE_COLORS: Record<string, string> = {
  SAFETY_SIGNAL: 'bg-red-100 text-red-800',
  INTERACTION: 'bg-purple-100 text-purple-800',
  DRUG_INTERACTION: 'bg-purple-100 text-purple-800',
  CONTRAINDICATION: 'bg-orange-100 text-orange-800',
  DOSING_RULE: 'bg-blue-100 text-blue-800',
  LAB_REFERENCE: 'bg-indigo-100 text-indigo-800',
  RENAL_ADJUSTMENT: 'bg-cyan-100 text-cyan-800',
  HEPATIC_ADJUSTMENT: 'bg-teal-100 text-teal-800',
  ORGAN_IMPAIRMENT: 'bg-yellow-100 text-yellow-800',
  REPRODUCTIVE_SAFETY: 'bg-pink-100 text-pink-800',
  THERAPEUTIC_GUIDELINE: 'bg-green-100 text-green-800',
  FORMULARY: 'bg-emerald-100 text-emerald-800',
  GERIATRIC_CONSIDERATION: 'bg-amber-100 text-amber-800',
  PEDIATRIC_DOSING: 'bg-pink-100 text-pink-800',
};

const STATUS_COLORS: Record<string, string> = {
  DRAFT: 'bg-gray-100 text-gray-700',
  PENDING_REVIEW: 'bg-amber-100 text-amber-800',
  APPROVED: 'bg-green-100 text-green-800',
  REJECTED: 'bg-red-100 text-red-800',
};

export function QueueTable({
  items,
  isLoading,
  sort,
  onSortChange,
  page,
  pageSize,
  total,
  onPageChange,
}: QueueTableProps) {
  const router = useRouter();
  const totalPages = Math.ceil(total / pageSize);

  const handleSort = (field: SortOptions['field']) => {
    if (sort.field === field) {
      onSortChange({ field, direction: sort.direction === 'asc' ? 'desc' : 'asc' });
    } else {
      onSortChange({ field, direction: 'desc' });
    }
  };

  const SortIcon = ({ field }: { field: SortOptions['field'] }) => {
    if (sort.field !== field) return null;
    return sort.direction === 'asc' ? (
      <ChevronUp className="h-4 w-4" />
    ) : (
      <ChevronDown className="h-4 w-4" />
    );
  };

  return (
    <div className="card">
      <div className="table-container">
        <table className="table">
          <thead>
            <tr>
              <th>Drug</th>
              <th>Fact Type</th>
              <th>Extracted Concept</th>
              <th>MedDRA PT</th>
              <th
                className="cursor-pointer hover:bg-gray-100"
                onClick={() => handleSort('confidence')}
              >
                <div className="flex items-center">
                  Confidence
                  <SortIcon field="confidence" />
                </div>
              </th>
              <th>Sources</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {Array.from({ length: 8 }).map((_, j) => (
                    <td key={j}><div className="skeleton h-6 w-20" /></td>
                  ))}
                </tr>
              ))
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={8} className="text-center py-12 text-gray-500">
                  No items in queue
                </td>
              </tr>
            ) : (
              items.map((item) => {
                const content = item.content as Record<string, unknown>;
                const concept =
                  (content?.conditionName as string) ||
                  (content?.interactantName as string) ||
                  (content?.condition as string) ||
                  '—';
                const meddraPT = content?.meddraPT as string | undefined;
                const sourceLabel =
                  item.sourceType === 'FDA_SPL' ? 'FDA SPL' :
                  item.sourceType === 'AUTHORITATIVE' ? 'Authoritative' :
                  item.sourceType || '—';

                return (
                  <tr
                    key={item.factId}
                    className="hover:bg-gray-50 cursor-pointer transition-colors"
                    onClick={() => router.push(`/facts/${item.factId}`)}
                  >
                    {/* Drug */}
                    <td>
                      <span className="font-semibold text-gray-900">
                        {item.drugName}
                      </span>
                    </td>

                    {/* Fact Type */}
                    <td>
                      <span
                        className={cn(
                          'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
                          FACT_TYPE_COLORS[item.factType] || 'bg-gray-100 text-gray-700'
                        )}
                      >
                        {item.factType.replace(/_/g, ' ')}
                      </span>
                    </td>

                    {/* Extracted Concept */}
                    <td>
                      <span className="text-gray-800 text-sm max-w-[200px] truncate block">
                        {concept}
                      </span>
                    </td>

                    {/* MedDRA PT */}
                    <td>
                      {meddraPT ? (
                        <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded font-mono text-gray-700">
                          {meddraPT}
                        </code>
                      ) : (
                        <span className="text-gray-400 text-xs">—</span>
                      )}
                    </td>

                    {/* Confidence */}
                    <td>
                      <div className="flex items-center">
                        <div className="w-14 h-2 bg-gray-200 rounded-full overflow-hidden mr-2">
                          <div
                            className={cn(
                              'h-full rounded-full',
                              item.confidenceScore >= 0.9 && 'bg-green-500',
                              item.confidenceScore >= 0.7 && item.confidenceScore < 0.9 && 'bg-amber-500',
                              item.confidenceScore < 0.7 && 'bg-red-500'
                            )}
                            style={{ width: `${item.confidenceScore * 100}%` }}
                          />
                        </div>
                        <span className="text-sm text-gray-600">
                          {(item.confidenceScore * 100).toFixed(0)}%
                        </span>
                      </div>
                    </td>

                    {/* Sources */}
                    <td>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                        {sourceLabel}
                      </span>
                    </td>

                    {/* Status */}
                    <td>
                      <div className="flex items-center space-x-1">
                        <span
                          className={cn(
                            'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
                            STATUS_COLORS[item.status] || 'bg-gray-100 text-gray-700'
                          )}
                        >
                          {item.status.replace(/_/g, ' ')}
                        </span>
                        {item.hasConflict && (
                          <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
                        )}
                      </div>
                    </td>

                    {/* Actions */}
                    <td>
                      <Link
                        href={`/facts/${item.factId}`}
                        className="text-blue-600 hover:text-blue-700 font-medium text-sm whitespace-nowrap"
                      >
                        Review →
                      </Link>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {total > 0 && (
        <div className="flex items-center justify-between px-6 py-4 border-t border-gray-100">
          <p className="text-sm text-gray-500">
            Showing {(page - 1) * pageSize + 1} to{' '}
            {Math.min(page * pageSize, total)} of {total} results
          </p>
          <div className="flex items-center space-x-2">
            <button
              onClick={() => onPageChange(page - 1)}
              disabled={page === 1}
              className="btn btn-outline btn-sm disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-sm text-gray-600">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => onPageChange(page + 1)}
              disabled={page === totalPages}
              className="btn btn-outline btn-sm disabled:opacity-50"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
