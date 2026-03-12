'use client';

import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { cn, getPriorityColor, getPriorityIcon } from '@/lib/utils';
import type { QueueSummary as QueueSummaryType } from '@/types/governance';

interface QueueSummaryProps {
  summary?: QueueSummaryType;
  isLoading: boolean;
}

const priorityLabels = {
  critical: 'Critical',
  high: 'High',
  standard: 'Standard',
  low: 'Low',
};

export function QueueSummary({ summary, isLoading }: QueueSummaryProps) {
  const priorities = ['critical', 'high', 'standard', 'low'] as const;
  const total = summary
    ? summary.critical + summary.high + summary.standard + summary.low
    : 0;

  return (
    <div className="card">
      <div className="card-header flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900">Review Queue</h2>
          <p className="text-sm text-gray-500">
            {total} items pending review
          </p>
        </div>
        <Link
          href="/queue"
          className="flex items-center text-sm font-medium text-blue-600 hover:text-blue-700"
        >
          View All
          <ArrowRight className="ml-1 h-4 w-4" />
        </Link>
      </div>

      <div className="card-body">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="skeleton h-12 w-full" />
            ))}
          </div>
        ) : (
          <div className="space-y-4">
            {priorities.map((priority) => {
              const count = summary?.[priority] || 0;
              const percent = total > 0 ? (count / total) * 100 : 0;

              return (
                <Link
                  key={priority}
                  href={`/queue?priority=${priority.toUpperCase()}`}
                  className="block"
                >
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center">
                      <span className="mr-2">
                        {getPriorityIcon(priority.toUpperCase() as any)}
                      </span>
                      <span className="text-sm font-medium text-gray-700">
                        {priorityLabels[priority]}
                      </span>
                    </div>
                    <span
                      className={cn(
                        'badge',
                        getPriorityColor(priority.toUpperCase() as any)
                      )}
                    >
                      {count}
                    </span>
                  </div>
                  <div className="sla-bar">
                    <div
                      className={cn(
                        'sla-bar-fill',
                        priority === 'critical' && 'bg-red-500',
                        priority === 'high' && 'bg-orange-500',
                        priority === 'standard' && 'bg-blue-500',
                        priority === 'low' && 'bg-gray-400'
                      )}
                      style={{ width: `${percent}%` }}
                    />
                  </div>
                </Link>
              );
            })}

            {/* Overdue Section */}
            {summary && summary.overdue > 0 && (
              <div className="pt-4 border-t border-gray-100">
                <Link
                  href="/queue?slaStatus=BREACHED"
                  className="flex items-center justify-between text-red-600 hover:text-red-700"
                >
                  <div className="flex items-center">
                    <span className="mr-2">⏰</span>
                    <span className="text-sm font-medium">Overdue Items</span>
                  </div>
                  <span className="badge badge-critical">
                    {summary.overdue}
                  </span>
                </Link>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
