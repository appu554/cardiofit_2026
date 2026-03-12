'use client';

import {
  FileCheck,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  TrendingUp,
} from 'lucide-react';
import { cn, formatNumber } from '@/lib/utils';
import type { GovernanceMetrics } from '@/types/governance';

interface MetricsCardsProps {
  metrics?: GovernanceMetrics;
  isLoading: boolean;
}

export function MetricsCards({ metrics, isLoading }: MetricsCardsProps) {
  // Use pre-computed totalFacts if available, otherwise sum from backend metrics
  const totalFacts = metrics?.totalFacts ||
    ((metrics?.totalDraft || 0) + (metrics?.totalApproved || 0) +
     (metrics?.totalActive || 0) + (metrics?.totalSuperseded || 0) +
     (metrics?.pendingReview || 0));

  const cards = [
    {
      label: 'Total Facts',
      value: totalFacts,
      icon: FileCheck,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
    },
    {
      label: 'Pending Reviews',
      value: metrics?.pendingReview || 0,
      icon: Clock,
      color: 'text-amber-600',
      bgColor: 'bg-amber-50',
      highlight: (metrics?.pendingReview || 0) > 10,
    },
    {
      label: 'Approved',
      value: metrics?.totalApproved || 0,
      icon: CheckCircle,
      color: 'text-green-600',
      bgColor: 'bg-green-50',
    },
    {
      label: 'Active',
      value: metrics?.totalActive || 0,
      icon: CheckCircle,
      color: 'text-emerald-600',
      bgColor: 'bg-emerald-50',
    },
    {
      label: 'Overdue (SLA)',
      value: metrics?.breachedSLA || 0,
      icon: AlertTriangle,
      color: 'text-red-600',
      bgColor: 'bg-red-50',
      highlight: (metrics?.breachedSLA || 0) > 0,
    },
    {
      label: 'At Risk (SLA)',
      value: metrics?.atRiskSLA || 0,
      icon: Clock,
      color: 'text-orange-600',
      bgColor: 'bg-orange-50',
      highlight: (metrics?.atRiskSLA || 0) > 0,
    },
  ];

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
      {cards.map((card) => (
        <div
          key={card.label}
          className={cn(
            'metric-card',
            card.highlight && 'ring-2 ring-red-200 animate-pulse-slow'
          )}
        >
          <div className="flex items-center justify-between">
            <div className={cn('p-2 rounded-lg', card.bgColor)}>
              <card.icon className={cn('h-5 w-5', card.color)} />
            </div>
          </div>
          <div className="mt-4">
            {isLoading ? (
              <div className="skeleton h-8 w-16" />
            ) : (
              <p className="metric-value">
                {(card as any).isTime ? card.value : formatNumber(card.value as number)}
              </p>
            )}
            <p className="metric-label">{card.label}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
