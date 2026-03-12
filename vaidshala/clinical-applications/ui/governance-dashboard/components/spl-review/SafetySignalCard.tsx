'use client';

import {
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Activity,
  Tag,
  BarChart3,
  Gauge,
  MessageSquare,
  ShieldAlert,
  Lightbulb,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { SafetySignalData } from '@/types/spl-review';

// ============================================================================
// SafetySignalCard — Type-specific content for SAFETY_SIGNAL facts
//
// Shows: conditionName, MedDRA PT + SOC, frequency band, severity,
// source phrase (with MedDRA validation indicator).
//
// This component is rendered INSIDE SPLFactCard as the `children` prop.
// The SPLFactCard handles meta (method, confidence, status) + review actions.
// ============================================================================

interface SafetySignalCardProps {
  data: SafetySignalData;
  /** Whether the term was MedDRA-validated (from fact-level flag) */
  meddraValidated: boolean;
  /** Signal type from pipeline (BOXED_WARNING, ADVERSE_REACTION, etc.) */
  signalType?: string;
  /** Clinical recommendation from pipeline */
  recommendation?: string;
  /** Description from pipeline */
  description?: string;
}

export function SafetySignalCard({ data, meddraValidated, signalType, recommendation, description }: SafetySignalCardProps) {
  return (
    <div className="space-y-3">
      {/* ── Condition Name + Signal Type (primary display) ──────────── */}
      <div>
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
          <h4 className="text-base font-semibold text-gray-900">
            {data.conditionName}
          </h4>
          {signalType && (
            <SignalTypeBadge type={signalType} />
          )}
        </div>
      </div>

      {/* ── MedDRA Mapping ──────────────────────────────────────────── */}
      <div className="bg-gray-50 rounded-lg p-3 space-y-2">
        <div className="flex items-center gap-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wider">
          <Tag className="h-3 w-3" />
          MedDRA Mapping
        </div>

        <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
          {/* PT (Preferred Term) */}
          <div>
            <span className="text-[10px] text-gray-400 uppercase">PT</span>
            <p className={cn(
              'text-xs font-medium',
              data.meddraPT ? 'text-gray-900' : 'text-gray-400 italic'
            )}>
              {data.meddraPT || 'Not mapped'}
            </p>
          </div>

          {/* PT Code */}
          <div>
            <span className="text-[10px] text-gray-400 uppercase">PT Code</span>
            <p className={cn(
              'text-xs font-mono',
              data.meddraPTCode ? 'text-gray-700' : 'text-gray-400 italic'
            )}>
              {data.meddraPTCode || '—'}
            </p>
          </div>

          {/* SOC (System Organ Class) */}
          <div className="col-span-2">
            <span className="text-[10px] text-gray-400 uppercase">SOC</span>
            <p className={cn(
              'text-xs',
              data.meddraSOC ? 'text-gray-700' : 'text-gray-400 italic'
            )}>
              {data.meddraSOC || 'Pending — requires MedDRA dictionary activation'}
            </p>
          </div>
        </div>

        {/* MedDRA validation indicator */}
        <div className="flex items-center gap-1.5 pt-1 border-t border-gray-100">
          {meddraValidated ? (
            <span className="inline-flex items-center gap-1 text-[10px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded">
              <CheckCircle2 className="h-3 w-3" />
              MedDRA Validated
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 text-[10px] font-semibold text-amber-700 bg-amber-50 px-2 py-0.5 rounded">
              <XCircle className="h-3 w-3" />
              Not in MedDRA — manual review required
            </span>
          )}
        </div>
      </div>

      {/* ── Frequency + Severity Row ────────────────────────────────── */}
      <div className="flex items-start gap-4">
        {/* Frequency */}
        <div className="flex-1">
          <div className="flex items-center gap-1.5 text-xs text-gray-500 mb-1">
            <BarChart3 className="h-3 w-3" />
            <span className="font-medium">Frequency</span>
          </div>
          {data.frequency ? (
            <div>
              <p className="text-sm font-medium text-gray-900">{data.frequency}</p>
              {data.frequencyBand && (
                <FrequencyBadge band={data.frequencyBand} />
              )}
            </div>
          ) : (
            <p className="text-xs text-gray-400 italic">Not reported</p>
          )}
        </div>

        {/* Severity */}
        <div className="flex-1">
          <div className="flex items-center gap-1.5 text-xs text-gray-500 mb-1">
            <Gauge className="h-3 w-3" />
            <span className="font-medium">Severity</span>
          </div>
          {data.severity ? (
            <SeverityBadge severity={data.severity} />
          ) : (
            <p className="text-xs text-gray-400 italic">Not specified</p>
          )}
        </div>
      </div>

      {/* ── Source Phrase ────────────────────────────────────────────── */}
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

      {/* ── Recommendation / Description ───────────────────────────── */}
      {(recommendation || description) && (
        <div className="bg-blue-50/60 rounded-lg p-3 space-y-1.5">
          <div className="flex items-center gap-1.5 text-xs font-semibold text-blue-600 uppercase tracking-wider">
            <Lightbulb className="h-3 w-3" />
            Clinical Context
          </div>
          {description && (
            <p className="text-xs text-gray-700 leading-relaxed">{description}</p>
          )}
          {recommendation && (
            <p className="text-xs text-blue-800 font-medium leading-relaxed">{recommendation}</p>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Frequency Band Badge
// ============================================================================

function FrequencyBadge({ band }: { band: string }) {
  const bandConfig: Record<string, { color: string; bg: string }> = {
    VERY_COMMON: { color: 'text-red-700', bg: 'bg-red-50' },
    COMMON: { color: 'text-orange-700', bg: 'bg-orange-50' },
    UNCOMMON: { color: 'text-yellow-700', bg: 'bg-yellow-50' },
    RARE: { color: 'text-blue-700', bg: 'bg-blue-50' },
    VERY_RARE: { color: 'text-gray-700', bg: 'bg-gray-100' },
  };

  const config = bandConfig[band] || { color: 'text-gray-700', bg: 'bg-gray-50' };

  return (
    <span className={cn('inline-flex items-center text-[10px] font-semibold px-1.5 py-0.5 rounded mt-1', config.bg, config.color)}>
      {band.replace(/_/g, ' ')}
    </span>
  );
}

// ============================================================================
// Signal Type Badge — Shows BOXED_WARNING, ADVERSE_REACTION, etc.
// ============================================================================

function SignalTypeBadge({ type }: { type: string }) {
  const typeConfig: Record<string, { color: string; bg: string; border: string; label: string }> = {
    BOXED_WARNING: { color: 'text-red-800', bg: 'bg-red-100', border: 'border-red-300', label: 'Boxed Warning' },
    ADVERSE_REACTION: { color: 'text-orange-700', bg: 'bg-orange-50', border: 'border-orange-200', label: 'Adverse Reaction' },
    WARNING: { color: 'text-amber-700', bg: 'bg-amber-50', border: 'border-amber-200', label: 'Warning' },
    PRECAUTION: { color: 'text-yellow-700', bg: 'bg-yellow-50', border: 'border-yellow-200', label: 'Precaution' },
    CONTRAINDICATION: { color: 'text-red-700', bg: 'bg-red-50', border: 'border-red-200', label: 'Contraindication' },
  };

  const config = typeConfig[type] || { color: 'text-gray-600', bg: 'bg-gray-50', border: 'border-gray-200', label: type.replace(/_/g, ' ') };

  return (
    <span className={cn(
      'inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded border uppercase tracking-wider',
      config.bg, config.color, config.border
    )}>
      <ShieldAlert className="h-2.5 w-2.5" />
      {config.label}
    </span>
  );
}

// ============================================================================
// Severity Badge
// ============================================================================

function SeverityBadge({ severity }: { severity: string }) {
  const lower = severity.toLowerCase();
  const config =
    lower.includes('fatal') || lower.includes('death') || lower.includes('life-threatening')
      ? { color: 'text-red-700', bg: 'bg-red-50', border: 'border-red-200' }
      : lower.includes('serious') || lower.includes('severe')
        ? { color: 'text-orange-700', bg: 'bg-orange-50', border: 'border-orange-200' }
        : lower.includes('moderate')
          ? { color: 'text-yellow-700', bg: 'bg-yellow-50', border: 'border-yellow-200' }
          : { color: 'text-gray-700', bg: 'bg-gray-50', border: 'border-gray-200' };

  return (
    <span className={cn('inline-flex items-center text-xs font-medium px-2 py-0.5 rounded border', config.bg, config.color, config.border)}>
      {severity}
    </span>
  );
}
