'use client';

import { cn } from '@/lib/utils';
import { CheckCircle, AlertTriangle, XCircle } from 'lucide-react';
import type { CompletenessGrade, GateVerdict } from '@/types/spl-review';
import { GRADE_CONFIG, VERDICT_CONFIG } from '@/types/spl-review';

// ============================================================================
// Grade Badge — shows A/B/C/D/F with color coding
// ============================================================================

export function GradeBadge({
  grade,
  size = 'md',
}: {
  grade: CompletenessGrade;
  size?: 'sm' | 'md' | 'lg';
}) {
  const config = GRADE_CONFIG[grade];
  const sizeClasses = {
    sm: 'h-6 w-6 text-xs',
    md: 'h-8 w-8 text-sm',
    lg: 'h-12 w-12 text-xl',
  };

  return (
    <div
      className={cn(
        'inline-flex items-center justify-center rounded-full font-bold border',
        config.bg,
        config.color,
        config.border,
        sizeClasses[size]
      )}
      title={`Grade ${grade} — ${config.label}`}
    >
      {grade}
    </div>
  );
}

// ============================================================================
// Verdict Badge — shows PASS/WARNING/BLOCK
// ============================================================================

export function VerdictBadge({
  verdict,
  size = 'md',
}: {
  verdict: GateVerdict;
  size?: 'sm' | 'md' | 'lg';
}) {
  const config = VERDICT_CONFIG[verdict];
  const IconComponent =
    verdict === 'PASS' ? CheckCircle :
    verdict === 'WARNING' ? AlertTriangle :
    XCircle;

  const sizeClasses = {
    sm: 'text-xs px-1.5 py-0.5 gap-1',
    md: 'text-xs px-2 py-1 gap-1.5',
    lg: 'text-sm px-3 py-1.5 gap-2',
  };

  const iconSizes = { sm: 'h-3 w-3', md: 'h-3.5 w-3.5', lg: 'h-4 w-4' };

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full font-semibold',
        config.bg,
        config.color,
        sizeClasses[size]
      )}
    >
      <IconComponent className={iconSizes[size]} />
      {config.label}
    </span>
  );
}

// ============================================================================
// Metric Bar — horizontal bar for percentages
// ============================================================================

export function MetricBar({
  label,
  value,
  suffix = '%',
  thresholds,
}: {
  label: string;
  value: number;
  suffix?: string;
  thresholds?: { good: number; fair: number }; // above good=green, above fair=yellow, else red
}) {
  const t = thresholds || { good: 80, fair: 50 };
  const barColor =
    value >= t.good ? 'bg-emerald-500' :
    value >= t.fair ? 'bg-yellow-500' :
    'bg-red-500';

  const textColor =
    value >= t.good ? 'text-emerald-700' :
    value >= t.fair ? 'text-yellow-700' :
    'text-red-700';

  const clampedWidth = Math.min(100, Math.max(0, value));

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-gray-500">{label}</span>
        <span className={cn('font-medium', textColor)}>
          {value.toFixed(1)}{suffix}
        </span>
      </div>
      <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', barColor)}
          style={{ width: `${clampedWidth}%` }}
        />
      </div>
    </div>
  );
}
