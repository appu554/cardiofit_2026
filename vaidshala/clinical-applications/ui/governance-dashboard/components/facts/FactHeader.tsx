'use client';

import {
  cn,
  getPriorityColor,
  getPriorityIcon,
  getStatusColor,
  getStatusLabel,
  getFactTypeLabel,
  getFactTypeIcon,
  getSlaStatus,
  getSlaStatusColor,
  getHoursRemaining,
  formatDateTime,
} from '@/lib/utils';
import type { ClinicalFact } from '@/types/governance';

interface FactHeaderProps {
  fact: ClinicalFact;
}

export function FactHeader({ fact }: FactHeaderProps) {
  // Use flat structure from KB-0 backend
  const reviewDueAt = fact.reviewDueAt || fact.slaDueDate;
  const slaStatus = reviewDueAt ? getSlaStatus(reviewDueAt) : null;
  const hoursRemaining = reviewDueAt ? getHoursRemaining(reviewDueAt) : null;

  // Extract target drug from content for DDI interactions
  const content = fact.content as Record<string, unknown>;
  const targetDrug = content?.target_drug as Record<string, unknown> | undefined;
  const interactingDrugName = targetDrug?.name as string | undefined || fact.interactingDrugName;

  // Get risk level from content
  const riskLevel = (content?.risk_level as string) || fact.severity || 'UNKNOWN';

  return (
    <div className="card p-6">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          {/* Drug Names */}
          <div className="flex items-center space-x-3 mb-2">
            <h1 className="text-2xl font-bold text-gray-900">
              {fact.drugName}
            </h1>
            {interactingDrugName && (
              <>
                <span className="text-gray-400">↔</span>
                <h1 className="text-2xl font-bold text-gray-900">
                  {interactingDrugName}
                </h1>
              </>
            )}
          </div>

          {/* Fact Type & Description */}
          <div className="flex items-center space-x-3 text-gray-600">
            <span className="text-lg">
              {getFactTypeIcon(fact.factType as any)}
            </span>
            <span className="font-medium">
              {getFactTypeLabel(fact.factType as any)}
            </span>
            <span className="text-gray-300">|</span>
            <span>Source: {fact.sourceId}</span>
          </div>
        </div>

        {/* Badges */}
        <div className="flex flex-col items-end space-y-2">
          {/* Status */}
          <span className={cn('badge text-sm', getStatusColor(fact.status))}>
            {getStatusLabel(fact.status)}
          </span>

          {/* Priority */}
          {fact.reviewPriority && (
            <span
              className={cn(
                'badge border text-sm',
                getPriorityColor(fact.reviewPriority)
              )}
            >
              {getPriorityIcon(fact.reviewPriority)} {fact.reviewPriority}
            </span>
          )}

          {/* Risk Level / Severity */}
          <span
            className={cn(
              'badge text-sm',
              riskLevel === 'CRITICAL' || riskLevel === 'HIGH' ? 'bg-red-100 text-red-700' :
              riskLevel === 'MODERATE' || riskLevel === 'MEDIUM' ? 'bg-amber-100 text-amber-700' :
              'bg-blue-100 text-blue-700'
            )}
          >
            {riskLevel} Risk
          </span>
        </div>
      </div>

      {/* SLA Bar */}
      {slaStatus && hoursRemaining !== null && (
        <div className="mt-4 pt-4 border-t border-gray-100">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-gray-600">SLA Deadline</span>
            <span
              className={cn(
                'text-sm font-medium px-2 py-0.5 rounded',
                getSlaStatusColor(slaStatus)
              )}
            >
              {hoursRemaining < 0
                ? `${Math.abs(hoursRemaining)}h overdue`
                : `${hoursRemaining}h remaining`}
            </span>
          </div>
          <div className="sla-bar">
            <div
              className={cn(
                'sla-bar-fill',
                slaStatus === 'ON_TRACK' && 'on-track',
                slaStatus === 'AT_RISK' && 'at-risk',
                slaStatus === 'BREACHED' && 'breached'
              )}
              style={{
                width: `${Math.max(0, Math.min(100, (1 - hoursRemaining / 168) * 100))}%`,
              }}
            />
          </div>
          {reviewDueAt && (
            <p className="text-xs text-gray-500 mt-1">
              Due: {formatDateTime(reviewDueAt)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
