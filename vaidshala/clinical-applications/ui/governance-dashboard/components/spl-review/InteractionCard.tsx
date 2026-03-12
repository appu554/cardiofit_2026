'use client';

import {
  Activity,
  ArrowUp,
  ArrowDown,
  HelpCircle,
  Shield,
  AlertOctagon,
  Eye,
  Pill,
  FlaskConical,
  MessageSquare,
  ClipboardList,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { InteractionData } from '@/types/spl-review';

// ============================================================================
// InteractionCard — Type-specific content for INTERACTION facts
//
// Shows: objectDrug, clinicalEffect, direction, clinicalAction,
// mechanism/enzyme, source phrase.
//
// Rendered inside SPLFactCard as `children`.
// ============================================================================

interface InteractionCardProps {
  data: InteractionData;
}

// Clinical action severity config
const ACTION_CONFIG: Record<InteractionData['clinicalAction'], {
  color: string;
  bg: string;
  border: string;
  icon: React.ComponentType<{ className?: string }>;
}> = {
  CONTRAINDICATED: {
    color: 'text-red-700',
    bg: 'bg-red-50',
    border: 'border-red-200',
    icon: AlertOctagon,
  },
  AVOID: {
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    border: 'border-orange-200',
    icon: Shield,
  },
  MONITOR: {
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    border: 'border-yellow-200',
    icon: Eye,
  },
  DOSE_ADJUST: {
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    icon: Pill,
  },
  INFORM: {
    color: 'text-gray-700',
    bg: 'bg-gray-50',
    border: 'border-gray-200',
    icon: HelpCircle,
  },
};

// Direction config
const DIRECTION_CONFIG: Record<InteractionData['direction'], {
  icon: React.ComponentType<{ className?: string }>;
  color: string;
  label: string;
}> = {
  INCREASE: { icon: ArrowUp, color: 'text-red-600', label: 'Increases' },
  DECREASE: { icon: ArrowDown, color: 'text-blue-600', label: 'Decreases' },
  UNKNOWN: { icon: HelpCircle, color: 'text-gray-500', label: 'Unknown direction' },
};

// Extended data shape — normalizer may pass through extra pipeline fields
interface ExtendedInteractionData extends InteractionData {
  management?: string;
  subjectDrug?: string;
}

export function InteractionCard({ data }: InteractionCardProps) {
  const ext = data as ExtendedInteractionData;
  const action = data.clinicalAction || 'INFORM';
  const actionConfig = ACTION_CONFIG[action] || ACTION_CONFIG.INFORM;
  const dirConfig = DIRECTION_CONFIG[data.direction] || DIRECTION_CONFIG.UNKNOWN;
  const ActionIcon = actionConfig.icon;
  const DirIcon = dirConfig.icon;

  return (
    <div className="space-y-3">
      {/* ── Interaction Header ──────────────────────────────────── */}
      <div className="flex items-start gap-3">
        <Activity className="h-5 w-5 text-blue-500 mt-0.5 shrink-0" />
        <div>
          <h4 className="text-base font-semibold text-gray-900">
            {data.objectDrug || 'Unknown Drug'}
          </h4>
          {data.objectDrugClass && (
            <p className="text-xs text-gray-500 mt-0.5">
              Class: {data.objectDrugClass}
            </p>
          )}
          {ext.subjectDrug && (
            <p className="text-xs text-gray-400 mt-0.5">
              Interacts with: {ext.subjectDrug}
            </p>
          )}
        </div>
      </div>

      {/* ── Clinical Action Badge ──────────────────────────────── */}
      <div className={cn(
        'rounded-lg p-3 border flex items-center gap-3',
        actionConfig.bg,
        actionConfig.border,
      )}>
        <ActionIcon className={cn('h-5 w-5 shrink-0', actionConfig.color)} />
        <div>
          <p className={cn('text-sm font-semibold', actionConfig.color)}>
            {action.replace(/_/g, ' ')}
          </p>
          <p className="text-xs text-gray-600 mt-0.5">
            {data.clinicalEffect || 'See source text for details'}
          </p>
        </div>
      </div>

      {/* ── Direction + Mechanism Row ──────────────────────────── */}
      <div className="flex items-start gap-4">
        {/* Direction */}
        <div className="flex-1">
          <span className="text-[10px] text-gray-400 uppercase font-semibold">Direction</span>
          <div className="flex items-center gap-1.5 mt-1">
            <DirIcon className={cn('h-4 w-4', dirConfig.color)} />
            <span className={cn('text-sm font-medium', dirConfig.color)}>
              {dirConfig.label}
            </span>
          </div>
        </div>

        {/* Mechanism / Enzyme */}
        {(data.mechanism || data.enzyme) && (
          <div className="flex-1">
            <span className="text-[10px] text-gray-400 uppercase font-semibold">Mechanism</span>
            <div className="mt-1 space-y-1">
              {data.mechanism && (
                <p className="text-xs text-gray-700">{data.mechanism}</p>
              )}
              {data.enzyme && (
                <span className="inline-flex items-center gap-1 text-[10px] font-semibold text-purple-700 bg-purple-50 px-1.5 py-0.5 rounded">
                  <FlaskConical className="h-3 w-3" />
                  {data.enzyme}
                </span>
              )}
            </div>
          </div>
        )}
      </div>

      {/* ── Management / Clinical Guidance ─────────────────────── */}
      {ext.management && (
        <div>
          <div className="flex items-center gap-1.5 text-xs text-gray-500 mb-1">
            <ClipboardList className="h-3 w-3" />
            <span className="font-medium">Clinical Management</span>
          </div>
          <p className="text-xs text-gray-700 bg-amber-50/60 border-l-2 border-amber-300 px-3 py-2 rounded-r-md leading-relaxed">
            {ext.management}
          </p>
        </div>
      )}

      {/* ── Source Phrase ──────────────────────────────────────── */}
      {data.sourcePhrase && (
        <div>
          <div className="flex items-center gap-1.5 text-xs text-gray-500 mb-1">
            <MessageSquare className="h-3 w-3" />
            <span className="font-medium">Source Text</span>
          </div>
          <blockquote className="text-xs text-gray-700 bg-blue-50/50 border-l-2 border-blue-300 px-3 py-2 rounded-r-md italic leading-relaxed">
            &ldquo;{data.sourcePhrase}&rdquo;
          </blockquote>
        </div>
      )}
    </div>
  );
}
