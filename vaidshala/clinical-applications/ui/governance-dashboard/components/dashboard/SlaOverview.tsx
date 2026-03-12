'use client';

import { CheckCircle, AlertTriangle, XCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { GovernanceMetrics } from '@/types/governance';

interface SlaOverviewProps {
  metrics?: GovernanceMetrics;
  isLoading: boolean;
}

export function SlaOverview({ metrics, isLoading }: SlaOverviewProps) {
  // Compute SLA compliance from available metrics
  // Compliance = (pending - breached) / pending * 100
  const pending = metrics?.pendingReview || 0;
  const breached = metrics?.breachedSLA || 0;
  const compliance = pending > 0 ? Math.round(((pending - breached) / pending) * 100) : 100;
  const isHealthy = compliance >= 90;
  const isWarning = compliance >= 70 && compliance < 90;
  const isCritical = compliance < 70;

  return (
    <div className="card">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">SLA Compliance</h2>
        <p className="text-sm text-gray-500">Review time performance</p>
      </div>

      <div className="card-body">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center h-40">
            <div className="skeleton h-24 w-24 rounded-full" />
          </div>
        ) : (
          <div className="flex flex-col items-center">
            {/* Circular Progress */}
            <div className="relative w-32 h-32">
              <svg className="w-full h-full transform -rotate-90">
                {/* Background circle */}
                <circle
                  cx="64"
                  cy="64"
                  r="56"
                  fill="none"
                  stroke="#e5e7eb"
                  strokeWidth="12"
                />
                {/* Progress circle */}
                <circle
                  cx="64"
                  cy="64"
                  r="56"
                  fill="none"
                  stroke={
                    isHealthy
                      ? '#059669'
                      : isWarning
                      ? '#d97706'
                      : '#dc2626'
                  }
                  strokeWidth="12"
                  strokeLinecap="round"
                  strokeDasharray={`${(compliance / 100) * 352} 352`}
                  className="transition-all duration-500"
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center">
                <div className="text-center">
                  <span
                    className={cn(
                      'text-2xl font-bold',
                      isHealthy && 'text-green-600',
                      isWarning && 'text-amber-600',
                      isCritical && 'text-red-600'
                    )}
                  >
                    {compliance.toFixed(0)}%
                  </span>
                </div>
              </div>
            </div>

            {/* Status */}
            <div
              className={cn(
                'flex items-center mt-4 px-3 py-1 rounded-full text-sm font-medium',
                isHealthy && 'bg-green-50 text-green-700',
                isWarning && 'bg-amber-50 text-amber-700',
                isCritical && 'bg-red-50 text-red-700'
              )}
            >
              {isHealthy && <CheckCircle className="h-4 w-4 mr-1" />}
              {isWarning && <AlertTriangle className="h-4 w-4 mr-1" />}
              {isCritical && <XCircle className="h-4 w-4 mr-1" />}
              {isHealthy && 'Healthy'}
              {isWarning && 'At Risk'}
              {isCritical && 'Critical'}
            </div>

            {/* SLA Targets */}
            <div className="w-full mt-6 space-y-2 text-sm">
              <div className="flex justify-between text-gray-600">
                <span>Critical</span>
                <span className="font-medium">24h</span>
              </div>
              <div className="flex justify-between text-gray-600">
                <span>High</span>
                <span className="font-medium">48h</span>
              </div>
              <div className="flex justify-between text-gray-600">
                <span>Standard</span>
                <span className="font-medium">7 days</span>
              </div>
              <div className="flex justify-between text-gray-600">
                <span>Low</span>
                <span className="font-medium">14 days</span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
