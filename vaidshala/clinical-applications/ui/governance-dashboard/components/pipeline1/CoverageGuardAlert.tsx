'use client';

import { AlertTriangle, Info } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  COVERAGE_GUARD_ALERT_CONFIG,
  COVERAGE_GUARD_SEVERITY_STYLE,
} from '@/lib/pipeline1-channels';
import type { CoverageGuardAlert } from '@/types/pipeline1';

// ============================================================================
// CoverageGuardAlertBanner — Sprint 1 CoverageGuard Safety Verification
//
// Surfaces CoverageGuard findings (numeric mismatch, branch loss, LLM-only,
// negation flip) as colored alert banners at point of decision in SpanInspector.
// Follows the exact structural pattern of the existing L1 Recovery and Channel
// Disagreement alerts already in SpanInspector.
// ============================================================================

interface CoverageGuardAlertProps {
  alert: CoverageGuardAlert | undefined;
}

const ICON_MAP = {
  AlertTriangle,
  Info,
} as const;

export function CoverageGuardAlertBanner({ alert }: CoverageGuardAlertProps) {
  if (!alert) return null;

  const config = COVERAGE_GUARD_ALERT_CONFIG[alert.type];
  const style = COVERAGE_GUARD_SEVERITY_STYLE[alert.alertSeverity];
  const Icon = ICON_MAP[config.iconName];

  return (
    <div className={cn('rounded-lg border overflow-hidden', style.border, style.bg)}>
      {/* Header */}
      <div
        className={cn(
          'flex items-center gap-2 px-3 py-2 border-b',
          style.headerBg,
          style.headerBorder,
        )}
      >
        <Icon className={cn('h-4 w-4 flex-shrink-0', style.textColor)} />
        <p className={cn('text-xs font-bold uppercase tracking-wider', style.textColor)}>
          {config.label}
        </p>
      </div>

      {/* Body — content varies by alert type */}
      <div className="px-3 py-2.5 space-y-1.5">
        {alert.type === 'numeric_mismatch' && (
          <>
            <p className="text-xs text-gray-700">
              <span className="text-gray-500">Source: </span>
              <code className="font-semibold text-gray-900">{alert.sourceValue}</code>
              <span className="text-gray-400 mx-1.5">&rarr;</span>
              <span className="text-gray-500">Extracted: </span>
              <code className="font-semibold text-red-600 line-through">
                {alert.extractedValue}
              </code>
            </p>
            <p className="text-xs text-gray-500">{alert.detail}</p>
          </>
        )}

        {alert.type === 'branch_loss' && (
          <>
            <p className="text-xs text-gray-700">
              <span className="text-gray-500">Source thresholds: </span>
              <span className="font-semibold text-gray-900">
                {alert.sourceThresholds}
              </span>
              <span className="text-gray-400 mx-1.5">&rarr;</span>
              <span className="text-gray-500">Extracted: </span>
              <span className="font-semibold text-red-600">
                {alert.extractedThresholds}
              </span>
            </p>
            <p className="text-xs text-gray-500">{alert.detail}</p>
          </>
        )}

        {alert.type === 'llm_only' && (
          <p className="text-xs text-gray-700">{alert.detail}</p>
        )}

        {alert.type === 'negation_flip' && (
          <>
            <p className="text-xs text-gray-700">
              <span className="text-gray-500">Source: </span>
              <code className="font-semibold text-gray-900">{alert.sourceValue}</code>
              <span className="text-gray-400 mx-1.5">&rarr;</span>
              <span className="text-gray-500">Extracted: </span>
              <code className="font-semibold text-red-600">
                {alert.extractedValue}
              </code>
            </p>
            <p className="text-xs text-gray-500">{alert.detail}</p>
          </>
        )}
      </div>
    </div>
  );
}
