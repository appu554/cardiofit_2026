'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, GitBranch, FileSearch, Loader2, CheckCircle } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { TASK_TYPE_CONFIG, SEVERITY_CONFIG } from '@/lib/pipeline1-channels';
import { cn } from '@/lib/utils';
import type { ReviewTask, ReviewTaskType, ReviewPhase } from '@/types/pipeline1';

// =============================================================================
// Phase → task-type filter map
// =============================================================================
// Phase 2: Tier-1 fact-by-fact — disagreements + spot-checks (normal review)
// Phase 3: Low-confidence review — L1 recovery items (pipeline failures)
// Other phases (1, 4, 5): show all tasks (no filtering)

const PHASE_TASK_FILTER: Partial<Record<ReviewPhase, ReviewTaskType[]>> = {
  2: ['DISAGREEMENT', 'PASSAGE_SPOT_CHECK'],
  3: ['L1_RECOVERY'],
};

// =============================================================================
// Props
// =============================================================================

interface ReviewTaskQueueProps {
  jobId: string;
  selectedTaskId: string | null;
  onSelectTask: (task: ReviewTask) => void;
  /** Active review phase — filters visible task types (phase 2/3 only) */
  activePhase?: ReviewPhase;
}

// =============================================================================
// Icon map (string → component)
// =============================================================================

const TASK_ICONS: Record<ReviewTaskType, typeof AlertTriangle> = {
  L1_RECOVERY: AlertTriangle,
  DISAGREEMENT: GitBranch,
  PASSAGE_SPOT_CHECK: FileSearch,
};

// =============================================================================
// Task group header
// =============================================================================

function TaskGroupHeader({
  taskType,
  count,
  resolved,
}: {
  taskType: ReviewTaskType;
  count: number;
  resolved: number;
}) {
  const config = TASK_TYPE_CONFIG[taskType];
  const Icon = TASK_ICONS[taskType];

  return (
    <div className="flex items-center justify-between px-3 py-2 bg-gray-100 border-b border-gray-200">
      <div className="flex items-center gap-1.5">
        <Icon className={cn('h-3.5 w-3.5', config.color)} />
        <span className={cn('text-xs font-semibold uppercase tracking-wider', config.color)}>
          {config.label}
        </span>
      </div>
      <span className="text-xs text-gray-500">
        {resolved}/{count}
      </span>
    </div>
  );
}

// =============================================================================
// Single task card
// =============================================================================

function TaskCard({
  task,
  isSelected,
  onClick,
}: {
  task: ReviewTask;
  isSelected: boolean;
  onClick: () => void;
}) {
  const sevConfig = SEVERITY_CONFIG[task.severity];
  const isResolved = task.status === 'RESOLVED';

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full text-left p-2.5 rounded-lg border transition-colors',
        isSelected
          ? 'ring-2 ring-blue-500 bg-blue-50 border-blue-200'
          : 'border-gray-200 hover:bg-gray-50',
        isResolved && 'opacity-60',
      )}
    >
      {/* Row 1: severity + page + resolved check */}
      <div className="flex items-center justify-between mb-1">
        <span
          className={cn(
            'inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded-full',
            sevConfig.bg,
            sevConfig.color,
          )}
        >
          {sevConfig.label}
        </span>
        <div className="flex items-center gap-1.5">
          {task.pageNumber != null && (
            <span className="text-[10px] text-gray-400 font-mono">p.{task.pageNumber}</span>
          )}
          {isResolved && (
            <CheckCircle className="h-3.5 w-3.5 text-green-500" aria-label="Resolved" />
          )}
        </div>
      </div>

      {/* Row 2: title (2-line clamp) */}
      <p className="text-sm text-gray-800 line-clamp-2 mb-1">{task.title}</p>

      {/* Row 3: description */}
      <p className="text-xs text-gray-500 line-clamp-1">{task.description}</p>
    </button>
  );
}

// =============================================================================
// Main component
// =============================================================================

export function ReviewTaskQueue({ jobId, selectedTaskId, onSelectTask, activePhase }: ReviewTaskQueueProps) {
  const { data: tasks, isLoading } = useQuery<ReviewTask[]>({
    queryKey: ['pipeline1-review-tasks', jobId],
    queryFn: () => pipeline1Api.reviewTasks.list(jobId),
  });

  // Apply phase filter, then group tasks by type (L1 → Disagreement → Spot-Check)
  const grouped = useMemo(() => {
    if (!tasks) return null;

    // Phase-based filtering: only show task types relevant to the active phase
    const allowedTypes = activePhase ? PHASE_TASK_FILTER[activePhase] : undefined;
    const filtered = allowedTypes
      ? tasks.filter((t) => allowedTypes.includes(t.taskType))
      : tasks;

    const groups: { type: ReviewTaskType; items: ReviewTask[] }[] = [
      { type: 'L1_RECOVERY', items: [] },
      { type: 'DISAGREEMENT', items: [] },
      { type: 'PASSAGE_SPOT_CHECK', items: [] },
    ];

    for (const task of filtered) {
      const group = groups.find((g) => g.type === task.taskType);
      if (group) group.items.push(task);
    }

    return groups.filter((g) => g.items.length > 0);
  }, [tasks, activePhase]);

  // Summary counts (based on filtered view, not total)
  const filteredTasks = useMemo(() => {
    if (!tasks) return [];
    const allowedTypes = activePhase ? PHASE_TASK_FILTER[activePhase] : undefined;
    return allowedTypes ? tasks.filter((t) => allowedTypes.includes(t.taskType)) : tasks;
  }, [tasks, activePhase]);
  const totalTasks = filteredTasks.length;
  const resolvedTasks = filteredTasks.filter((t) => t.status === 'RESOLVED').length;

  // Loading skeleton
  if (isLoading) {
    return (
      <div className="p-3 space-y-2" role="status" aria-label="Loading tasks">
        <Loader2 className="h-4 w-4 animate-spin text-gray-400 mx-auto mb-2" />
        {[...Array(5)].map((_, i) => (
          <div key={i} className="h-20 bg-gray-200 rounded-lg animate-pulse" />
        ))}
      </div>
    );
  }

  // Empty state
  if (!grouped || grouped.length === 0) {
    return (
      <div className="p-4 text-center">
        <CheckCircle className="h-8 w-8 text-green-400 mx-auto mb-2" />
        <p className="text-sm font-medium text-gray-700">No flagged items</p>
        <p className="text-xs text-gray-500 mt-1">Pipeline output is clean</p>
      </div>
    );
  }

  return (
    <div className="overflow-y-auto">
      {/* Progress counter */}
      <div className="px-3 py-2 border-b border-gray-200 flex items-center justify-between">
        <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
          Tasks
        </span>
        <span className="text-xs font-medium text-gray-700">
          {resolvedTasks} of {totalTasks} resolved
        </span>
      </div>

      {/* Grouped task list */}
      {grouped.map((group) => {
        const groupResolved = group.items.filter((t) => t.status === 'RESOLVED').length;

        return (
          <div key={group.type}>
            <TaskGroupHeader
              taskType={group.type}
              count={group.items.length}
              resolved={groupResolved}
            />
            <div className="p-2 space-y-1.5">
              {group.items.map((task) => (
                <TaskCard
                  key={task.id}
                  task={task}
                  isSelected={selectedTaskId === task.id}
                  onClick={() => onSelectTask(task)}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
