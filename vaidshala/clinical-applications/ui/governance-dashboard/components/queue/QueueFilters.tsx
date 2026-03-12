'use client';

import { X } from 'lucide-react';
import type { QueueFilters as QueueFiltersType, ReviewPriority, FactStatus, FactType } from '@/types/governance';

interface QueueFiltersProps {
  filters: QueueFiltersType;
  onFiltersChange: (filters: QueueFiltersType) => void;
  onReset: () => void;
}

const priorities: ReviewPriority[] = ['CRITICAL', 'HIGH', 'STANDARD', 'LOW'];
const statuses: FactStatus[] = ['DRAFT', 'PENDING_REVIEW'];
const slaStatuses = ['ON_TRACK', 'AT_RISK', 'BREACHED'] as const;
const factTypes: FactType[] = [
  'SAFETY_SIGNAL',
  'INTERACTION',
  'ORGAN_IMPAIRMENT',
  'CONTRAINDICATION',
  'DOSING_RULE',
  'LAB_REFERENCE',
  'ALLERGY_CROSS_REACTIVITY',
];

export function QueueFilters({ filters, onFiltersChange, onReset }: QueueFiltersProps) {
  const togglePriority = (priority: ReviewPriority) => {
    const current = filters.priority || [];
    const updated = current.includes(priority)
      ? current.filter((p) => p !== priority)
      : [...current, priority];
    onFiltersChange({ ...filters, priority: updated.length ? updated : undefined });
  };

  const toggleFactType = (factType: FactType) => {
    const current = filters.factType || [];
    const updated = current.includes(factType)
      ? current.filter((t) => t !== factType)
      : [...current, factType];
    onFiltersChange({ ...filters, factType: updated.length ? updated : undefined });
  };

  const hasFilters = Object.values(filters).some(Boolean);

  return (
    <div className="space-y-4">
      {/* Priority Filter */}
      <div>
        <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
          Priority
        </label>
        <div className="flex flex-wrap gap-2">
          {priorities.map((priority) => (
            <button
              key={priority}
              onClick={() => togglePriority(priority)}
              className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
                filters.priority?.includes(priority)
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {priority}
            </button>
          ))}
        </div>
      </div>

      {/* SLA Status Filter */}
      <div>
        <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
          SLA Status
        </label>
        <div className="flex flex-wrap gap-2">
          {slaStatuses.map((status) => (
            <button
              key={status}
              onClick={() =>
                onFiltersChange({
                  ...filters,
                  slaStatus: filters.slaStatus === status ? undefined : status,
                })
              }
              className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
                filters.slaStatus === status
                  ? status === 'BREACHED'
                    ? 'bg-red-600 text-white'
                    : status === 'AT_RISK'
                    ? 'bg-amber-600 text-white'
                    : 'bg-green-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {status.replace('_', ' ')}
            </button>
          ))}
        </div>
      </div>

      {/* Fact Type Filter */}
      <div>
        <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
          Fact Type
        </label>
        <div className="flex flex-wrap gap-2">
          {factTypes.map((factType) => (
            <button
              key={factType}
              onClick={() => toggleFactType(factType)}
              className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
                filters.factType?.includes(factType)
                  ? 'bg-purple-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {factType.replace(/_/g, ' ')}
            </button>
          ))}
        </div>
      </div>

      {/* Conflicts Filter */}
      <div>
        <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
          Conflicts
        </label>
        <div className="flex gap-2">
          <button
            onClick={() =>
              onFiltersChange({
                ...filters,
                hasConflicts: filters.hasConflicts === true ? undefined : true,
              })
            }
            className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
              filters.hasConflicts === true
                ? 'bg-red-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            With Conflicts
          </button>
          <button
            onClick={() =>
              onFiltersChange({
                ...filters,
                hasConflicts: filters.hasConflicts === false ? undefined : false,
              })
            }
            className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
              filters.hasConflicts === false
                ? 'bg-green-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            No Conflicts
          </button>
        </div>
      </div>

      {/* Reset Button */}
      {hasFilters && (
        <div className="pt-2">
          <button
            onClick={onReset}
            className="flex items-center text-sm text-gray-500 hover:text-gray-700"
          >
            <X className="h-4 w-4 mr-1" />
            Clear all filters
          </button>
        </div>
      )}
    </div>
  );
}
