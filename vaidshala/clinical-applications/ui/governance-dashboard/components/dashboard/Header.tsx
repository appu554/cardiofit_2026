'use client';

import { Bell, Search, RefreshCw } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { governanceApi } from '@/lib/api';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import { getUserInitials, getRoleDisplayName } from '@/lib/auth';

export function Header() {
  const { user } = useAuth();
  const { data: metrics, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['governance-metrics'],
    queryFn: async () => {
      try {
        return await governanceApi.dashboard.getMetrics();
      } catch {
        // Endpoint not yet implemented — fail silently, show fallback values
        return null;
      }
    },
    refetchInterval: 60000, // 60s (reduced frequency since endpoint isn't live yet)
    retry: false,
  });

  return (
    <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
      {/* Search */}
      <div className="flex items-center flex-1 max-w-lg">
        <div className="relative w-full">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search facts, drugs, or reviewers..."
            className="input pl-10 w-full"
          />
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center space-x-4">
        {/* Refresh Button */}
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className={cn(
            'p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors',
            isFetching && 'animate-spin'
          )}
        >
          <RefreshCw className="h-5 w-5" />
        </button>

        {/* Notifications */}
        <button className="relative p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors">
          <Bell className="h-5 w-5" />
          {metrics && (metrics.overdueReviews ?? 0) > 0 && (
            <span className="absolute top-1 right-1 h-4 w-4 bg-red-500 text-white text-xs rounded-full flex items-center justify-center">
              {(metrics.overdueReviews ?? 0) > 9 ? '9+' : metrics.overdueReviews}
            </span>
          )}
        </button>

        {/* Quick Stats */}
        <div className="hidden md:flex items-center space-x-4 pl-4 border-l border-gray-200">
          <div className="text-right">
            <p className="text-xs text-gray-500">Pending Reviews</p>
            <p className="text-lg font-semibold text-amber-600">
              {isLoading ? '—' : metrics?.pendingReviews || 0}
            </p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-500">SLA Compliance</p>
            <p className={cn(
              'text-lg font-semibold',
              (metrics?.slaCompliancePercent || 0) >= 90
                ? 'text-green-600'
                : 'text-red-600'
            )}>
              {isLoading ? '—' : `${(metrics?.slaCompliancePercent || 0).toFixed(0)}%`}
            </p>
          </div>
        </div>

        {/* User Avatar */}
        {user && (
          <div className="flex items-center space-x-2 pl-4 border-l border-gray-200">
            {user.picture ? (
              <img src={user.picture} alt={user.name} className="h-8 w-8 rounded-full" />
            ) : (
              <div className="h-8 w-8 rounded-full bg-blue-600 text-white flex items-center justify-center text-sm font-medium">
                {getUserInitials(user)}
              </div>
            )}
            <div className="hidden lg:block text-right">
              <p className="text-sm font-medium text-gray-700">{user.name}</p>
              <p className="text-xs text-gray-500">{getRoleDisplayName(user)}</p>
            </div>
          </div>
        )}
      </div>
    </header>
  );
}
