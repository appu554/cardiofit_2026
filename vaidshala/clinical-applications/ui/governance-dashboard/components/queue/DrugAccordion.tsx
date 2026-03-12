'use client';

import { useState, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Pill,
  ShieldAlert,
  Zap,
  FlaskConical,
  Activity,
  Check,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { governanceApi } from '@/lib/api';
import { useAuth } from '@/hooks/useAuth';
import type { QueueItem } from '@/types/governance';

// ============================================================================
// Types
// ============================================================================

interface DrugGroup {
  rxcui: string;
  drugName: string;
  items: QueueItem[];
  totalFacts: number;
  criticalCount: number;
  highCount: number;
  factTypeBreakdown: Record<string, number>;
}

interface DrugAccordionProps {
  items: QueueItem[];
  isLoading: boolean;
}

// ============================================================================
// Constants
// ============================================================================

const FACT_TYPE_CONFIG: Record<string, { color: string; icon: React.ElementType; label: string }> = {
  SAFETY_SIGNAL: { color: 'bg-red-100 text-red-800', icon: ShieldAlert, label: 'Safety Signals' },
  INTERACTION: { color: 'bg-purple-100 text-purple-800', icon: Zap, label: 'Interactions' },
  DRUG_INTERACTION: { color: 'bg-purple-100 text-purple-800', icon: Zap, label: 'Drug Interactions' },
  ORGAN_IMPAIRMENT: { color: 'bg-yellow-100 text-yellow-800', icon: AlertTriangle, label: 'Organ Impairment' },
  CONTRAINDICATION: { color: 'bg-orange-100 text-orange-800', icon: AlertTriangle, label: 'Contraindications' },
  DOSING_RULE: { color: 'bg-blue-100 text-blue-800', icon: FlaskConical, label: 'Dosing Rules' },
  LAB_REFERENCE: { color: 'bg-indigo-100 text-indigo-800', icon: FlaskConical, label: 'Lab References' },
  RENAL_ADJUSTMENT: { color: 'bg-cyan-100 text-cyan-800', icon: Activity, label: 'Renal Adjustments' },
  HEPATIC_ADJUSTMENT: { color: 'bg-teal-100 text-teal-800', icon: Activity, label: 'Hepatic Adjustments' },
  THERAPEUTIC_GUIDELINE: { color: 'bg-green-100 text-green-800', icon: CheckCircle2, label: 'Guidelines' },
  REPRODUCTIVE_SAFETY: { color: 'bg-pink-100 text-pink-800', icon: ShieldAlert, label: 'Reproductive Safety' },
  FORMULARY: { color: 'bg-emerald-100 text-emerald-800', icon: CheckCircle2, label: 'Formulary' },
  GERIATRIC_CONSIDERATION: { color: 'bg-amber-100 text-amber-800', icon: Activity, label: 'Geriatric' },
  PEDIATRIC_DOSING: { color: 'bg-pink-100 text-pink-800', icon: Activity, label: 'Pediatric' },
};

const SEVERITY_ORDER: Record<string, number> = {
  CRITICAL: 0,
  HIGH: 1,
  MEDIUM: 2,
  LOW: 3,
};

// ============================================================================
// Grouping Logic
// ============================================================================

function groupByDrug(items: QueueItem[]): DrugGroup[] {
  const groups: Record<string, DrugGroup> = {};

  for (const item of items) {
    const key = item.rxcui || item.drugName;
    if (!groups[key]) {
      groups[key] = {
        rxcui: item.rxcui,
        drugName: item.drugName,
        items: [],
        totalFacts: 0,
        criticalCount: 0,
        highCount: 0,
        factTypeBreakdown: {},
      };
    }
    groups[key].items.push(item);
    groups[key].totalFacts++;
    groups[key].factTypeBreakdown[item.factType] =
      (groups[key].factTypeBreakdown[item.factType] || 0) + 1;

    const severity = (item.content as Record<string, unknown>)?.severity as string;
    if (severity === 'CRITICAL') groups[key].criticalCount++;
    if (severity === 'HIGH') groups[key].highCount++;
  }

  // Sort groups: critical first, then by total facts descending
  return Object.values(groups).sort((a, b) => {
    if (a.criticalCount !== b.criticalCount) return b.criticalCount - a.criticalCount;
    return b.totalFacts - a.totalFacts;
  });
}

function groupByFactType(items: QueueItem[]): Record<string, QueueItem[]> {
  const groups: Record<string, QueueItem[]> = {};
  for (const item of items) {
    groups[item.factType] = groups[item.factType] || [];
    groups[item.factType].push(item);
  }
  // Sort items within each group by severity
  for (const key of Object.keys(groups)) {
    groups[key].sort((a, b) => {
      const sa = SEVERITY_ORDER[(a.content as any)?.severity || 'LOW'] ?? 3;
      const sb = SEVERITY_ORDER[(b.content as any)?.severity || 'LOW'] ?? 3;
      return sa - sb;
    });
  }
  return groups;
}

// ============================================================================
// Drug Card Component
// ============================================================================

function DrugCard({ group }: { group: DrugGroup }) {
  const [expanded, setExpanded] = useState(false);
  const [expandedTypes, setExpandedTypes] = useState<Set<string>>(new Set());
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [batchReason, setBatchReason] = useState('');
  const [showBatchModal, setShowBatchModal] = useState<'approve' | 'reject' | null>(null);

  const router = useRouter();
  const queryClient = useQueryClient();
  const { user, canReviewFacts } = useAuth();

  const factTypeGroups = useMemo(() => groupByFactType(group.items), [group.items]);

  // Batch approve mutation
  const batchApproveMutation = useMutation({
    mutationFn: async (factIds: string[]) => {
      const results = await Promise.allSettled(
        factIds.map((factId) =>
          governanceApi.facts.approve({
            factId,
            reviewerId: user?.sub || '',
            action: 'APPROVE',
            reason: batchReason || 'Batch approved — clinically verified',
          })
        )
      );
      const failed = results.filter((r) => r.status === 'rejected');
      if (failed.length > 0) {
        throw new Error(`${failed.length} of ${factIds.length} approvals failed`);
      }
      return results;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['queue'] });
      setSelectedIds(new Set());
      setShowBatchModal(null);
      setBatchReason('');
    },
  });

  // Batch reject mutation
  const batchRejectMutation = useMutation({
    mutationFn: async (factIds: string[]) => {
      const results = await Promise.allSettled(
        factIds.map((factId) =>
          governanceApi.facts.reject({
            factId,
            reviewerId: user?.sub || '',
            action: 'REJECT',
            reason: batchReason || 'Batch rejected',
          })
        )
      );
      const failed = results.filter((r) => r.status === 'rejected');
      if (failed.length > 0) {
        throw new Error(`${failed.length} of ${factIds.length} rejections failed`);
      }
      return results;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['queue'] });
      setSelectedIds(new Set());
      setShowBatchModal(null);
      setBatchReason('');
    },
  });

  const toggleType = (type: string) => {
    const next = new Set(expandedTypes);
    next.has(type) ? next.delete(type) : next.add(type);
    setExpandedTypes(next);
  };

  const toggleSelect = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const next = new Set(selectedIds);
    next.has(id) ? next.delete(id) : next.add(id);
    setSelectedIds(next);
  };

  const toggleSelectAllInType = (type: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const typeItems = factTypeGroups[type] || [];
    const allSelected = typeItems.every((item) => selectedIds.has(item.factId));
    const next = new Set(selectedIds);
    if (allSelected) {
      typeItems.forEach((item) => next.delete(item.factId));
    } else {
      typeItems.forEach((item) => next.add(item.factId));
    }
    setSelectedIds(next);
  };

  const isProcessing = batchApproveMutation.isPending || batchRejectMutation.isPending;

  return (
    <div className="card overflow-hidden">
      {/* Drug Card Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
      >
        <div className="flex items-center space-x-4">
          <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-blue-50">
            <Pill className="h-5 w-5 text-blue-600" />
          </div>
          <div className="text-left">
            <h3 className="text-lg font-semibold text-gray-900">
              {group.drugName}
            </h3>
            <p className="text-sm text-gray-500">
              RxCUI: {group.rxcui}
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          {/* Badges */}
          <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-700">
            {group.totalFacts} Facts
          </span>
          {group.criticalCount > 0 && (
            <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
              <AlertTriangle className="h-3 w-3 mr-1" />
              {group.criticalCount} Critical
            </span>
          )}
          {group.highCount > 0 && (
            <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
              {group.highCount} High
            </span>
          )}
          {/* Fact type mini badges */}
          {Object.entries(group.factTypeBreakdown).map(([type, count]) => {
            const cfg = FACT_TYPE_CONFIG[type];
            return (
              <span
                key={type}
                className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium', cfg?.color || 'bg-gray-100 text-gray-700')}
              >
                {count} {cfg?.label || type.replace(/_/g, ' ')}
              </span>
            );
          })}

          {expanded ? (
            <ChevronDown className="h-5 w-5 text-gray-400" />
          ) : (
            <ChevronRight className="h-5 w-5 text-gray-400" />
          )}
        </div>
      </button>

      {/* Expanded Content */}
      {expanded && (
        <div className="border-t border-gray-100">
          {/* Batch Action Bar */}
          {selectedIds.size > 0 && canReviewFacts && (
            <div className="px-6 py-3 bg-blue-50 border-b border-blue-100 flex items-center justify-between">
              <span className="text-sm font-medium text-blue-800">
                <Check className="h-4 w-4 inline mr-1" />
                {selectedIds.size} fact{selectedIds.size > 1 ? 's' : ''} selected
              </span>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setShowBatchModal('approve')}
                  disabled={isProcessing}
                  className="btn btn-sm bg-green-600 text-white hover:bg-green-700 disabled:opacity-50"
                >
                  <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                  Approve Selected
                </button>
                <button
                  onClick={() => setShowBatchModal('reject')}
                  disabled={isProcessing}
                  className="btn btn-sm bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
                >
                  <XCircle className="h-3.5 w-3.5 mr-1" />
                  Reject Selected
                </button>
                <button
                  onClick={() => setSelectedIds(new Set())}
                  className="btn btn-sm btn-outline"
                >
                  Clear
                </button>
              </div>
            </div>
          )}

          {/* Batch Confirmation Modal */}
          {showBatchModal && (
            <div className="px-6 py-4 bg-yellow-50 border-b border-yellow-100">
              <p className="text-sm font-medium text-yellow-800 mb-2">
                {showBatchModal === 'approve'
                  ? `Approve ${selectedIds.size} fact(s)?`
                  : `Reject ${selectedIds.size} fact(s)?`}
              </p>
              <input
                type="text"
                placeholder="Reason (required for reject, optional for approve)"
                value={batchReason}
                onChange={(e) => setBatchReason(e.target.value)}
                className="input w-full mb-2 text-sm"
              />
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => {
                    const ids = Array.from(selectedIds);
                    if (showBatchModal === 'approve') {
                      batchApproveMutation.mutate(ids);
                    } else {
                      if (!batchReason.trim()) return;
                      batchRejectMutation.mutate(ids);
                    }
                  }}
                  disabled={isProcessing || (showBatchModal === 'reject' && !batchReason.trim())}
                  className={cn(
                    'btn btn-sm text-white disabled:opacity-50',
                    showBatchModal === 'approve' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'
                  )}
                >
                  {isProcessing ? 'Processing...' : 'Confirm'}
                </button>
                <button
                  onClick={() => { setShowBatchModal(null); setBatchReason(''); }}
                  className="btn btn-sm btn-outline"
                >
                  Cancel
                </button>
              </div>
              {(batchApproveMutation.isError || batchRejectMutation.isError) && (
                <p className="mt-2 text-xs text-red-600">
                  {(batchApproveMutation.error || batchRejectMutation.error)?.message}
                </p>
              )}
            </div>
          )}

          {/* Fact Type Sections */}
          {Object.entries(factTypeGroups).map(([type, typeItems]) => {
            const cfg = FACT_TYPE_CONFIG[type] || { color: 'bg-gray-100 text-gray-700', icon: Activity, label: type.replace(/_/g, ' ') };
            const TypeIcon = cfg.icon;
            const isTypeExpanded = expandedTypes.has(type);
            const allSelected = typeItems.every((item) => selectedIds.has(item.factId));

            return (
              <div key={type} className="border-b border-gray-50 last:border-b-0">
                {/* Fact Type Header */}
                <button
                  onClick={() => toggleType(type)}
                  className="w-full px-6 py-3 flex items-center justify-between hover:bg-gray-50 transition-colors"
                >
                  <div className="flex items-center space-x-3">
                    {canReviewFacts && (
                      <input
                        type="checkbox"
                        checked={allSelected}
                        onChange={() => {}}
                        onClick={(e) => toggleSelectAllInType(type, e)}
                        className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                    )}
                    <TypeIcon className={cn('h-4 w-4', cfg.color.split(' ')[1])} />
                    <span className="font-medium text-gray-800">
                      {cfg.label}
                    </span>
                    <span className="text-sm text-gray-500">
                      ({typeItems.length})
                    </span>
                  </div>
                  {isTypeExpanded ? (
                    <ChevronDown className="h-4 w-4 text-gray-400" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-gray-400" />
                  )}
                </button>

                {/* Facts List */}
                {isTypeExpanded && (
                  <div className="bg-gray-50/50">
                    <table className="w-full">
                      <thead>
                        <tr className="text-xs text-gray-500 uppercase">
                          {canReviewFacts && <th className="px-6 py-2 w-8"></th>}
                          <th className="px-3 py-2 text-left">Concept</th>
                          <th className="px-3 py-2 text-left">Severity</th>
                          <th className="px-3 py-2 text-left">MedDRA PT</th>
                          <th className="px-3 py-2 text-left">Confidence</th>
                          <th className="px-3 py-2 text-left">Source</th>
                          <th className="px-3 py-2 text-left">SLA</th>
                          <th className="px-3 py-2"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {typeItems.map((item) => {
                          const content = item.content as Record<string, unknown>;
                          const concept = (content?.conditionName as string) || (content?.description as string) || '—';
                          const severity = (content?.severity as string) || '—';
                          const meddraPT = content?.meddraPT as string;
                          const isSelected = selectedIds.has(item.factId);

                          return (
                            <tr
                              key={item.factId}
                              className={cn(
                                'border-t border-gray-100 hover:bg-white cursor-pointer transition-colors',
                                isSelected && 'bg-blue-50/50'
                              )}
                              onClick={() => router.push(`/facts/${item.factId}`)}
                            >
                              {canReviewFacts && (
                                <td className="px-6 py-2.5">
                                  <input
                                    type="checkbox"
                                    checked={isSelected}
                                    onChange={() => {}}
                                    onClick={(e) => toggleSelect(item.factId, e)}
                                    className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                                  />
                                </td>
                              )}
                              <td className="px-3 py-2.5">
                                <span className="text-sm text-gray-900 font-medium">
                                  {concept}
                                </span>
                              </td>
                              <td className="px-3 py-2.5">
                                <span
                                  className={cn(
                                    'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
                                    severity === 'CRITICAL' && 'bg-red-100 text-red-800',
                                    severity === 'HIGH' && 'bg-orange-100 text-orange-800',
                                    severity === 'MEDIUM' && 'bg-amber-100 text-amber-800',
                                    severity === 'LOW' && 'bg-gray-100 text-gray-700',
                                    severity === '—' && 'bg-gray-100 text-gray-500'
                                  )}
                                >
                                  {severity}
                                </span>
                              </td>
                              <td className="px-3 py-2.5">
                                {meddraPT ? (
                                  <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded font-mono text-gray-600">
                                    {meddraPT}
                                  </code>
                                ) : (
                                  <span className="text-gray-400 text-xs">—</span>
                                )}
                              </td>
                              <td className="px-3 py-2.5">
                                <div className="flex items-center">
                                  <div className="w-12 h-1.5 bg-gray-200 rounded-full overflow-hidden mr-1.5">
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
                                  <span className="text-xs text-gray-500">
                                    {(item.confidenceScore * 100).toFixed(0)}%
                                  </span>
                                </div>
                              </td>
                              <td className="px-3 py-2.5">
                                <span className="text-xs text-gray-500">
                                  {item.sourceType === 'FDA_SPL' ? 'FDA SPL' : item.sourceType || '—'}
                                </span>
                              </td>
                              <td className="px-3 py-2.5">
                                <span
                                  className={cn(
                                    'text-xs font-medium',
                                    (item as any).slaStatus === 'ON_TRACK' && 'text-green-600',
                                    (item as any).slaStatus === 'AT_RISK' && 'text-amber-600',
                                    (item as any).slaStatus === 'BREACHED' && 'text-red-600',
                                  )}
                                >
                                  {(item as any).slaStatus?.replace(/_/g, ' ') || '—'}
                                </span>
                              </td>
                              <td className="px-3 py-2.5 text-right">
                                <span className="text-blue-600 text-xs font-medium">
                                  Review →
                                </span>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Main Export
// ============================================================================

export function DrugAccordion({ items, isLoading }: DrugAccordionProps) {
  const drugGroups = useMemo(() => groupByDrug(items), [items]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="card p-6">
            <div className="flex items-center space-x-4">
              <div className="skeleton h-10 w-10 rounded-lg" />
              <div className="flex-1 space-y-2">
                <div className="skeleton h-5 w-40" />
                <div className="skeleton h-4 w-24" />
              </div>
              <div className="flex space-x-2">
                <div className="skeleton h-6 w-16 rounded-full" />
                <div className="skeleton h-6 w-20 rounded-full" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (drugGroups.length === 0) {
    return (
      <div className="card">
        <div className="text-center py-16 text-gray-500">
          <Pill className="h-12 w-12 mx-auto mb-3 text-gray-300" />
          <p className="text-lg font-medium text-gray-700">Queue is empty</p>
          <p className="text-sm mt-1">All clinical facts have been reviewed</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Summary Bar */}
      <div className="flex items-center space-x-4 text-sm text-gray-600">
        <span className="font-medium">{drugGroups.length} drug{drugGroups.length > 1 ? 's' : ''}</span>
        <span className="text-gray-300">|</span>
        <span>{items.length} facts pending review</span>
      </div>

      {/* Drug Cards */}
      {drugGroups.map((group) => (
        <DrugCard key={group.rxcui} group={group} />
      ))}
    </div>
  );
}
