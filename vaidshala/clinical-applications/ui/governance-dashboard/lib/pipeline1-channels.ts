// ============================================================================
// Pipeline 1 Channel & Risk Constants
// Maps extraction pipeline channels to display metadata for the reviewer UI.
// ============================================================================

import type { ChannelKey, ChannelInfo, PageRisk, CoverageGuardAlertType } from '@/types/pipeline1';

// Channel B–F are extraction channels; L1 is oracle recovery
export const CHANNEL_MAP: Record<ChannelKey, ChannelInfo> = {
  B: { name: 'Drug Dictionary', color: 'text-blue-700', bg: 'bg-blue-100', border: 'border-l-blue-500', bgTint: 'bg-blue-50/40', icon: 'BookOpen' },
  C: { name: 'Grammar / Regex', color: 'text-green-700', bg: 'bg-green-100', border: 'border-l-green-500', bgTint: 'bg-green-50/40', icon: 'Code' },
  D: { name: 'Table Decomp', color: 'text-purple-700', bg: 'bg-purple-100', border: 'border-l-purple-500', bgTint: 'bg-purple-50/40', icon: 'Table' },
  E: { name: 'GLiNER NER', color: 'text-orange-700', bg: 'bg-orange-100', border: 'border-l-orange-500', bgTint: 'bg-orange-50/40', icon: 'Zap' },
  F: { name: 'NuExtract LLM', color: 'text-pink-700', bg: 'bg-pink-100', border: 'border-l-pink-500', bgTint: 'bg-pink-50/40', icon: 'Sparkles' },
  L1: { name: 'Oracle Recovery', color: 'text-red-700', bg: 'bg-red-100', border: 'border-l-red-500', bgTint: 'bg-red-50/40', icon: 'AlertTriangle' },
  L1_RECOVERY: { name: 'Oracle Recovery', color: 'text-red-700', bg: 'bg-red-100', border: 'border-l-red-500', bgTint: 'bg-red-50/40', icon: 'AlertTriangle' },
};

export const RISK_CONFIG: Record<PageRisk, { label: string; color: string; bg: string; border: string }> = {
  clean: { label: 'Clean', color: 'text-green-700', bg: 'bg-green-50', border: 'border-green-400' },
  oracle: { label: 'Oracle', color: 'text-red-700', bg: 'bg-red-50', border: 'border-red-400' },
  disagreement: { label: 'Disagreement', color: 'text-amber-700', bg: 'bg-amber-50', border: 'border-amber-400' },
};

const ALL_CHANNELS = Object.keys(CHANNEL_MAP) as ChannelKey[];

export function getChannelInfo(channel: string): ChannelInfo {
  const key = channel as ChannelKey;
  return CHANNEL_MAP[key] || { name: channel, color: 'text-gray-700', bg: 'bg-gray-100', icon: 'Circle' };
}

export function getChannelColor(channel: string): string {
  return getChannelInfo(channel).color;
}

export function getChannelName(channel: string): string {
  return getChannelInfo(channel).name;
}

export function getChannelBg(channel: string): string {
  return getChannelInfo(channel).bg;
}

// Confidence color thresholds
export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.9) return 'text-green-600';
  if (confidence >= 0.7) return 'text-blue-600';
  if (confidence >= 0.5) return 'text-amber-600';
  return 'text-red-600';
}

export function getConfidenceBg(confidence: number): string {
  if (confidence >= 0.9) return 'bg-green-50';
  if (confidence >= 0.7) return 'bg-blue-50';
  if (confidence >= 0.5) return 'bg-amber-50';
  return 'bg-red-50';
}

// ============================================================================
// Task Type Configuration (Review Task Queue)
// ============================================================================

import type { ReviewTaskType, ReviewTaskSeverity } from '@/types/pipeline1';

export const TASK_TYPE_CONFIG: Record<
  ReviewTaskType,
  { label: string; color: string; bg: string; border: string; icon: string }
> = {
  L1_RECOVERY: {
    label: 'L1 Recovery',
    color: 'text-red-700',
    bg: 'bg-red-50',
    border: 'border-red-300',
    icon: 'AlertTriangle',
  },
  DISAGREEMENT: {
    label: 'Disagreement',
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    border: 'border-amber-300',
    icon: 'GitBranch',
  },
  PASSAGE_SPOT_CHECK: {
    label: 'Passage Check',
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    border: 'border-blue-300',
    icon: 'FileSearch',
  },
};

export const SEVERITY_CONFIG: Record<
  ReviewTaskSeverity,
  { label: string; color: string; bg: string }
> = {
  critical: { label: 'Critical', color: 'text-red-700', bg: 'bg-red-100' },
  warning: { label: 'Warning', color: 'text-amber-700', bg: 'bg-amber-100' },
  info: { label: 'Info', color: 'text-blue-700', bg: 'bg-blue-100' },
};

// ============================================================================
// CoverageGuard Alert Configuration (Sprint 1)
// ============================================================================

export const COVERAGE_GUARD_ALERT_CONFIG: Record<
  CoverageGuardAlertType,
  { label: string; iconName: 'AlertTriangle' | 'Info' }
> = {
  numeric_mismatch: { label: 'NUMERIC MISMATCH', iconName: 'AlertTriangle' },
  branch_loss: { label: 'BRANCH INCOMPLETE', iconName: 'AlertTriangle' },
  llm_only: { label: 'LLM-ONLY EXTRACTION', iconName: 'Info' },
  negation_flip: { label: 'NEGATION INTEGRITY', iconName: 'AlertTriangle' },
};

// Severity → banner styling. Mirrors the structural pattern of the existing
// L1 Recovery alert (SpanInspector:289) and Channel Disagreement (SpanInspector:318).
export const COVERAGE_GUARD_SEVERITY_STYLE: Record<
  ReviewTaskSeverity,
  { border: string; bg: string; headerBg: string; textColor: string; headerBorder: string }
> = {
  critical: {
    border: 'border-red-300',
    bg: 'bg-red-50',
    headerBg: 'bg-red-100',
    textColor: 'text-red-800',
    headerBorder: 'border-red-200',
  },
  warning: {
    border: 'border-amber-300',
    bg: 'bg-amber-50',
    headerBg: 'bg-amber-100',
    textColor: 'text-amber-800',
    headerBorder: 'border-amber-200',
  },
  info: {
    border: 'border-blue-300',
    bg: 'bg-blue-50',
    headerBg: 'bg-blue-100',
    textColor: 'text-blue-800',
    headerBorder: 'border-blue-200',
  },
};

export { ALL_CHANNELS };
