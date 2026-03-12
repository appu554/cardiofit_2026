'use client';

import Link from 'next/link';
import {
  AlertTriangle,
  FileCheck,
  Clock,
  AlertCircle,
} from 'lucide-react';
import { formatRelativeTime, cn, getPriorityColor } from '@/lib/utils';
import type { QueueItem, ReviewPriority } from '@/types/governance';

interface RecentActivityProps {
  items?: QueueItem[];
  isLoading: boolean;
}

// Priority icon and color config
const priorityConfig: Record<
  ReviewPriority,
  { icon: React.ElementType; bgColor: string; textColor: string }
> = {
  CRITICAL: {
    icon: AlertTriangle,
    bgColor: 'bg-red-50',
    textColor: 'text-red-600',
  },
  HIGH: {
    icon: AlertCircle,
    bgColor: 'bg-orange-50',
    textColor: 'text-orange-600',
  },
  STANDARD: {
    icon: FileCheck,
    bgColor: 'bg-blue-50',
    textColor: 'text-blue-600',
  },
  LOW: {
    icon: Clock,
    bgColor: 'bg-gray-50',
    textColor: 'text-gray-600',
  },
};

export function RecentActivity({ items, isLoading }: RecentActivityProps) {
  return (
    <div className="card">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">Recent Activity</h2>
        <p className="text-sm text-gray-500">
          Latest governance actions (21 CFR Part 11 compliant)
        </p>
      </div>

      <div className="card-body">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-start space-x-4">
                <div className="skeleton h-10 w-10 rounded-full" />
                <div className="flex-1 space-y-2">
                  <div className="skeleton h-4 w-3/4" />
                  <div className="skeleton h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : items && items.length > 0 ? (
          <div className="flow-root">
            <ul className="-mb-8">
              {items.slice(0, 10).map((item, idx) => {
                const priority = item.reviewPriority || 'STANDARD';
                const config = priorityConfig[priority];
                const Icon = config.icon;
                const isLast = idx === Math.min(items.length - 1, 9);
                const content = item.content as Record<string, unknown>;

                return (
                  <li key={item.factId}>
                    <div className="relative pb-8">
                      {!isLast && (
                        <span
                          className="absolute left-5 top-10 -ml-px h-full w-0.5 bg-gray-200"
                          aria-hidden="true"
                        />
                      )}
                      <div className="relative flex items-start space-x-3">
                        <div className={cn('relative p-2 rounded-full', config.bgColor)}>
                          <Icon className={cn('h-5 w-5', config.textColor)} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm">
                            <Link
                              href={`/facts/${item.factId}`}
                              className="font-medium text-gray-900 hover:text-blue-600"
                            >
                              {item.drugName}
                            </Link>{' '}
                            <span className="text-gray-500">
                              {item.factType.toLowerCase()} - {priority.toLowerCase()} priority
                            </span>
                          </div>
                          {(content?.clinical_effect as string) && (
                            <p className="mt-0.5 text-sm text-gray-500 truncate max-w-md">
                              {String(content.clinical_effect)}
                            </p>
                          )}
                          <div className="mt-1 flex items-center space-x-2">
                            <span className="text-xs text-gray-400">
                              {formatRelativeTime(item.createdAt)}
                            </span>
                            <span className="text-gray-300">•</span>
                            <span className={cn(
                              'badge text-xs',
                              item.slaStatus === 'BREACHED' && 'badge-critical',
                              item.slaStatus === 'AT_RISK' && 'badge-high',
                              item.slaStatus === 'ON_TRACK' && 'badge-standard'
                            )}>
                              {item.slaStatus.replace('_', ' ')}
                            </span>
                            <span className="text-gray-300">•</span>
                            <span className="font-mono text-xs text-gray-400">
                              {item.sourceId}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </li>
                );
              })}
            </ul>
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500">
            No recent activity
          </div>
        )}
      </div>
    </div>
  );
}
