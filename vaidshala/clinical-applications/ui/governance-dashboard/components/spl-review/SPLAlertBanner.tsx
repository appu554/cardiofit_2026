'use client';

import {
  AlertTriangle,
  AlertOctagon,
  Brain,
  Tag,
  Activity,
  PackageX,
  Sparkles,
  Info,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { SPLAlert, SPLAlertType } from '@/types/spl-review';

// ============================================================================
// SPL Alert Banner — Shows SPL-specific quality alerts
//
// Alert types from the design spec:
// - FREQUENCY_MISMATCH: Table says 5%, LLM says "rare"
// - LLM_ONLY: No structured corroboration
// - MEDDRA_UNRESOLVED: Term not in MedDRA dictionary
// - DIRECTION_CONFLICT: Grammar says INCREASE, LLM says DECREASE
// - MISSING_FACT_TYPE: Expected fact type has 0 extractions
// - AUTO_APPROVE_SAMPLE: Spot-check alert for auto-approved fact
// ============================================================================

const ALERT_TYPE_CONFIG: Record<SPLAlertType, {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  description: string;
}> = {
  FREQUENCY_MISMATCH: {
    icon: AlertTriangle,
    label: 'Frequency Mismatch',
    description: 'Structured and LLM extraction disagree on frequency/incidence',
  },
  LLM_ONLY: {
    icon: Brain,
    label: 'LLM-Only Extraction',
    description: 'No deterministic corroboration — fact extracted only by LLM',
  },
  MEDDRA_UNRESOLVED: {
    icon: Tag,
    label: 'MedDRA Unresolved',
    description: 'Adverse event term not found in MedDRA dictionary',
  },
  DIRECTION_CONFLICT: {
    icon: Activity,
    label: 'Direction Conflict',
    description: 'DDI grammar and LLM disagree on interaction direction',
  },
  MISSING_FACT_TYPE: {
    icon: PackageX,
    label: 'Missing Fact Type',
    description: 'Expected fact type has zero extractions for this drug',
  },
  AUTO_APPROVE_SAMPLE: {
    icon: Sparkles,
    label: 'Auto-Approve Sample',
    description: 'This auto-approved fact was selected for pharmacist spot-check',
  },
};

const SEVERITY_CONFIG = {
  CRITICAL: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    iconColor: 'text-red-600',
    textColor: 'text-red-800',
    badgeBg: 'bg-red-100',
    badgeText: 'text-red-700',
  },
  HIGH: {
    bg: 'bg-orange-50',
    border: 'border-orange-200',
    iconColor: 'text-orange-600',
    textColor: 'text-orange-800',
    badgeBg: 'bg-orange-100',
    badgeText: 'text-orange-700',
  },
  MEDIUM: {
    bg: 'bg-yellow-50',
    border: 'border-yellow-200',
    iconColor: 'text-yellow-600',
    textColor: 'text-yellow-800',
    badgeBg: 'bg-yellow-100',
    badgeText: 'text-yellow-700',
  },
  LOW: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    iconColor: 'text-blue-600',
    textColor: 'text-blue-800',
    badgeBg: 'bg-blue-100',
    badgeText: 'text-blue-700',
  },
};

// ============================================================================
// Single Alert Banner
// ============================================================================

interface SPLAlertBannerProps {
  alert: SPLAlert;
}

export function SPLAlertBanner({ alert }: SPLAlertBannerProps) {
  const typeConfig = ALERT_TYPE_CONFIG[alert.type];
  const sevConfig = SEVERITY_CONFIG[alert.severity];
  const Icon = typeConfig.icon;

  return (
    <div className={cn('rounded-lg border p-3', sevConfig.bg, sevConfig.border)}>
      <div className="flex items-start gap-2.5">
        <Icon className={cn('h-4 w-4 mt-0.5 shrink-0', sevConfig.iconColor)} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={cn('text-xs font-semibold', sevConfig.textColor)}>
              {typeConfig.label}
            </span>
            <span className={cn('text-[10px] font-bold px-1.5 py-0.5 rounded', sevConfig.badgeBg, sevConfig.badgeText)}>
              {alert.severity}
            </span>
          </div>
          <p className={cn('text-xs mt-1', sevConfig.textColor)}>
            {alert.message}
          </p>
          {alert.drugName && (
            <p className="text-[10px] text-gray-500 mt-1">
              Drug: {alert.drugName}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Alert List — Shows all alerts for a drug/fact
// ============================================================================

interface SPLAlertListProps {
  alerts: SPLAlert[];
  className?: string;
}

export function SPLAlertList({ alerts, className }: SPLAlertListProps) {
  if (alerts.length === 0) return null;

  // Sort by severity: CRITICAL first
  const severityOrder = { CRITICAL: 0, HIGH: 1, MEDIUM: 2, LOW: 3 };
  const sorted = [...alerts].sort(
    (a, b) => (severityOrder[a.severity] ?? 4) - (severityOrder[b.severity] ?? 4)
  );

  const criticalCount = alerts.filter((a) => a.severity === 'CRITICAL').length;
  const highCount = alerts.filter((a) => a.severity === 'HIGH').length;

  return (
    <div className={cn('space-y-2', className)}>
      {/* Summary header */}
      <div className="flex items-center gap-2">
        <AlertOctagon className="h-4 w-4 text-amber-600" />
        <span className="text-xs font-semibold text-gray-900">
          {alerts.length} Alert{alerts.length !== 1 ? 's' : ''}
        </span>
        {criticalCount > 0 && (
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-red-100 text-red-700">
            {criticalCount} critical
          </span>
        )}
        {highCount > 0 && (
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-orange-100 text-orange-700">
            {highCount} high
          </span>
        )}
      </div>

      {/* Alert items */}
      {sorted.map((alert, i) => (
        <SPLAlertBanner key={`${alert.type}-${alert.factId || i}`} alert={alert} />
      ))}
    </div>
  );
}

// ============================================================================
// Inline Alert Chip (for compact display within fact cards)
// ============================================================================

interface SPLAlertChipProps {
  alert: SPLAlert;
}

export function SPLAlertChip({ alert }: SPLAlertChipProps) {
  const typeConfig = ALERT_TYPE_CONFIG[alert.type];
  const sevConfig = SEVERITY_CONFIG[alert.severity];
  const Icon = typeConfig.icon;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full border',
        sevConfig.badgeBg,
        sevConfig.badgeText,
        sevConfig.border,
      )}
      title={alert.message}
    >
      <Icon className="h-3 w-3" />
      {typeConfig.label}
    </span>
  );
}
