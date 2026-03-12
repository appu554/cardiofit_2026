'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Play, Square, Activity, AlertCircle } from 'lucide-react';
import { governanceApi } from '@/lib/api';
import { cn, formatRelativeTime } from '@/lib/utils';

export function ExecutorStatus() {
  const queryClient = useQueryClient();
  const [isToggling, setIsToggling] = useState(false);

  const { data: status, isLoading } = useQuery({
    queryKey: ['executor-status'],
    queryFn: () => governanceApi.executor.getStatus(),
    refetchInterval: 10000, // 10 seconds
  });

  const startMutation = useMutation({
    mutationFn: () => governanceApi.executor.start(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executor-status'] });
      setIsToggling(false);
    },
    onError: () => setIsToggling(false),
  });

  const stopMutation = useMutation({
    mutationFn: () => governanceApi.executor.stop(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executor-status'] });
      setIsToggling(false);
    },
    onError: () => setIsToggling(false),
  });

  const handleToggle = () => {
    setIsToggling(true);
    if (status?.running) {
      stopMutation.mutate();
    } else {
      startMutation.mutate();
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center space-x-2 px-4 py-2 bg-gray-100 rounded-lg">
        <div className="skeleton h-4 w-24" />
      </div>
    );
  }

  return (
    <div
      className={cn(
        'flex items-center space-x-4 px-4 py-2 rounded-lg border',
        status?.running
          ? 'bg-green-50 border-green-200'
          : 'bg-gray-50 border-gray-200'
      )}
    >
      {/* Status Indicator */}
      <div className="flex items-center space-x-2">
        <div
          className={cn(
            'h-2 w-2 rounded-full',
            status?.running ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
          )}
        />
        <span
          className={cn(
            'text-sm font-medium',
            status?.running ? 'text-green-700' : 'text-gray-600'
          )}
        >
          {status?.running ? 'Executor Running' : 'Executor Stopped'}
        </span>
      </div>

      {/* Stats */}
      {status?.running && (
        <div className="flex items-center space-x-4 text-xs text-gray-500 border-l border-gray-200 pl-4">
          <div className="flex items-center">
            <Activity className="h-3 w-3 mr-1" />
            {status.factsProcessed} processed
          </div>
          {status.errors > 0 && (
            <div className="flex items-center text-red-600">
              <AlertCircle className="h-3 w-3 mr-1" />
              {status.errors} errors
            </div>
          )}
          {status.lastProcessedAt && (
            <span>Last: {formatRelativeTime(status.lastProcessedAt)}</span>
          )}
        </div>
      )}

      {/* Toggle Button */}
      <button
        onClick={handleToggle}
        disabled={isToggling}
        className={cn(
          'flex items-center px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
          status?.running
            ? 'bg-red-100 text-red-700 hover:bg-red-200'
            : 'bg-green-100 text-green-700 hover:bg-green-200',
          isToggling && 'opacity-50 cursor-not-allowed'
        )}
      >
        {status?.running ? (
          <>
            <Square className="h-3 w-3 mr-1" />
            Stop
          </>
        ) : (
          <>
            <Play className="h-3 w-3 mr-1" />
            Start
          </>
        )}
      </button>
    </div>
  );
}
