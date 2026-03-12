'use client';

import { Check, Lock } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SUB_PHASE_CONFIG, SUB_PHASES } from '@/types/pipeline1';
import type { ReviewSubPhase } from '@/types/pipeline1';
import type { PhaseGates } from '@/hooks/usePhaseGates';
import { isSubPhaseUnlocked, isSubPhaseComplete } from '@/hooks/usePhaseGates';

// ============================================================================
// PhaseStepper — Full 5-Phase Workflow (7 sub-phases with group headers)
//
// Vertical stepper showing all 7 sub-phases. Sub-phases 2A/2B are grouped
// under "Fact Review", 3A/3B under "Low-Confidence". Each step is locked,
// available, active, or completed based on the 6 boolean gates.
// ============================================================================

interface PhaseStepperProps {
  activePhase: ReviewSubPhase;
  onSelectPhase: (phase: ReviewSubPhase) => void;
  gates: PhaseGates;
}

// Track which group labels have been rendered to show headers only once
function getGroupHeader(phase: ReviewSubPhase): string | null {
  const config = SUB_PHASE_CONFIG[phase];
  return config.groupLabel ?? null;
}

export function PhaseStepper({ activePhase, onSelectPhase, gates }: PhaseStepperProps) {
  // Track rendered group headers to avoid duplicates
  const renderedGroups = new Set<number>();

  return (
    <div className="px-2 py-1 space-y-0.5">
      {SUB_PHASES.map((phase) => {
        const config = SUB_PHASE_CONFIG[phase];
        const unlocked = isSubPhaseUnlocked(phase, gates);
        const completed = isSubPhaseComplete(phase, gates);
        const isActive = phase === activePhase;
        const isSubStep = phase === '2b' || phase === '3b';
        const groupHeader = getGroupHeader(phase);
        const showGroupHeader = groupHeader && !renderedGroups.has(config.group);

        if (showGroupHeader) {
          renderedGroups.add(config.group);
        }

        return (
          <div key={phase}>
            {/* Group divider for 2A/3A — first sub-phase in a multi-phase group */}
            {showGroupHeader && (
              <div className="flex items-center gap-2 px-2 pt-3 pb-1">
                <div className="h-px flex-1 bg-gray-200" />
                <span className="text-[9px] font-semibold text-gray-400 uppercase tracking-wider whitespace-nowrap">
                  {groupHeader}
                </span>
                <div className="h-px flex-1 bg-gray-200" />
              </div>
            )}

            <button
              onClick={() => unlocked && onSelectPhase(phase)}
              disabled={!unlocked}
              className={cn(
                'w-full flex items-center gap-2.5 py-2 rounded-lg text-left transition-colors',
                // Sub-steps (2B, 3B) indent slightly
                isSubStep ? 'pl-5 pr-2.5' : 'px-2.5',
                isActive && 'bg-blue-50 border border-blue-200',
                !isActive && unlocked && 'hover:bg-gray-100 border border-transparent',
                !unlocked && 'opacity-50 cursor-not-allowed border border-transparent',
              )}
            >
              {/* Step indicator circle */}
              <div
                className={cn(
                  'flex-shrink-0 rounded-full flex items-center justify-center text-xs font-bold',
                  // Sub-steps get smaller circles
                  isSubStep ? 'w-5 h-5' : 'w-6 h-6',
                  completed && 'bg-green-100 text-green-700',
                  isActive && !completed && 'bg-blue-600 text-white',
                  !isActive && !completed && unlocked && 'bg-gray-200 text-gray-600',
                  !unlocked && 'bg-gray-100 text-gray-400',
                )}
              >
                {completed ? (
                  <Check className={cn(isSubStep ? 'h-3 w-3' : 'h-3.5 w-3.5')} />
                ) : !unlocked ? (
                  <Lock className="h-3 w-3" />
                ) : (
                  <span className={cn(isSubStep && 'text-[10px]')}>
                    {phase.toUpperCase()}
                  </span>
                )}
              </div>

              {/* Label + description */}
              <div className="min-w-0 flex-1">
                <p
                  className={cn(
                    'text-xs font-semibold truncate',
                    isActive ? 'text-blue-800' : unlocked ? 'text-gray-800' : 'text-gray-400',
                  )}
                >
                  {config.label}
                </p>
                <p
                  className={cn(
                    'text-[10px] truncate',
                    isActive ? 'text-blue-600' : 'text-gray-400',
                  )}
                >
                  {config.description}
                </p>
              </div>
            </button>
          </div>
        );
      })}
    </div>
  );
}
