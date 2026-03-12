'use client';

import {
  CheckCircle,
  XCircle,
  AlertTriangle,
  FileCheck,
  UserPlus,
  ArrowUpCircle,
  GitMerge,
  Clock,
  Shield,
} from 'lucide-react';
import { formatDateTime, cn } from '@/lib/utils';
import type { AuditEvent, AuditEventType } from '@/types/governance';

interface AuditHistoryProps {
  history?: AuditEvent[];
  isLoading: boolean;
}

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

export function AuditHistory({ history, isLoading }: AuditHistoryProps) {
  return (
    <div className="card">
      <div className="card-header">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Audit History</h2>
            <p className="text-sm text-gray-500">
              21 CFR Part 11 compliant audit trail
            </p>
          </div>
          <Shield className="h-5 w-5 text-gray-400" />
        </div>
      </div>

      <div className="card-body">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-start space-x-3">
                <div className="skeleton h-8 w-8 rounded-full" />
                <div className="flex-1 space-y-2">
                  <div className="skeleton h-4 w-3/4" />
                  <div className="skeleton h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : history && history.length > 0 ? (
          <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200" />

            <ul className="space-y-6">
              {history.map((event, idx) => {
                const config = eventConfig[event.eventType] || {
                  icon: FileCheck,
                  color: 'text-gray-600',
                  bgColor: 'bg-gray-50',
                  label: event.eventType,
                };
                const Icon = config.icon;
                // Handle field name differences: backend uses actorType, frontend uses actorRole
                const actorRole = (event as any).actorType || event.actorRole || '';
                // Extract reason from metadata if not directly available
                const reason = event.reason || (event.metadata as any)?.reason || '';

                return (
                  <li key={event.id} className="relative pl-10">
                    {/* Timeline dot */}
                    <div
                      className={cn(
                        'absolute left-0 p-1.5 rounded-full',
                        config.bgColor
                      )}
                    >
                      <Icon className={cn('h-4 w-4', config.color)} />
                    </div>

                    {/* Content */}
                    <div className="bg-gray-50 rounded-lg p-4">
                      <div className="flex items-start justify-between">
                        <div>
                          <p className="font-medium text-gray-900">
                            {config.label}
                          </p>
                          <p className="text-sm text-gray-600 mt-0.5">
                            by{' '}
                            <span className="font-medium">
                              {event.actorName || event.actorId}
                            </span>
                            <span className="text-gray-400 mx-1">•</span>
                            <span className="text-gray-500">
                              {actorRole}
                            </span>
                          </p>
                        </div>
                        <span className="text-xs text-gray-500">
                          {formatDateTime(event.createdAt)}
                        </span>
                      </div>

                      {/* Reason */}
                      {reason && (
                        <p className="mt-2 text-sm text-gray-700 bg-white p-2 rounded border border-gray-100">
                          "{reason}"
                        </p>
                      )}

                      {/* State Change */}
                      {event.previousState && event.newState && (
                        <div className="mt-2 flex items-center text-xs">
                          <span className="text-gray-500">
                            {event.previousState}
                          </span>
                          <span className="mx-2 text-gray-400">→</span>
                          <span className="font-medium text-gray-700">
                            {event.newState}
                          </span>
                        </div>
                      )}

                      {/* Digital Signature */}
                      {event.signature && (
                        <div className="mt-3 pt-2 border-t border-gray-100 flex items-center text-xs text-gray-400">
                          <Shield className="h-3 w-3 mr-1" />
                          <span className="font-mono">
                            Signature: {event.signature.slice(0, 16)}...
                          </span>
                        </div>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500">
            No audit history available
          </div>
        )}
      </div>
    </div>
  );
}
