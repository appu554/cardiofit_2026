import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { format, formatDistanceToNow, differenceInHours, isPast } from 'date-fns';
import type { ReviewPriority, FactStatus, FactType } from '@/types/governance';

// ============================================================================
// Class Name Utility
// ============================================================================

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// ============================================================================
// Date & Time Formatting
// ============================================================================

export function formatDate(date: string | Date): string {
  return format(new Date(date), 'MMM d, yyyy');
}

export function formatDateTime(date: string | Date): string {
  return format(new Date(date), 'MMM d, yyyy h:mm a');
}

export function formatRelativeTime(date: string | Date): string {
  return formatDistanceToNow(new Date(date), { addSuffix: true });
}

export function getHoursRemaining(dueDate: string): number {
  return differenceInHours(new Date(dueDate), new Date());
}

export function isOverdue(dueDate: string): boolean {
  return isPast(new Date(dueDate));
}

// ============================================================================
// SLA Status Helpers
// ============================================================================

export type SlaStatus = 'ON_TRACK' | 'AT_RISK' | 'BREACHED';

export function getSlaStatus(dueDate: string): SlaStatus {
  const hoursRemaining = getHoursRemaining(dueDate);

  if (hoursRemaining < 0) return 'BREACHED';
  if (hoursRemaining < 8) return 'AT_RISK';
  return 'ON_TRACK';
}

export function getSlaStatusColor(status: SlaStatus): string {
  switch (status) {
    case 'ON_TRACK':
      return 'text-green-600 bg-green-50';
    case 'AT_RISK':
      return 'text-amber-600 bg-amber-50';
    case 'BREACHED':
      return 'text-red-600 bg-red-50';
  }
}

// ============================================================================
// Priority Helpers
// ============================================================================

export function getPriorityColor(priority: ReviewPriority): string {
  switch (priority) {
    case 'CRITICAL':
      return 'text-red-700 bg-red-100 border-red-200';
    case 'HIGH':
      return 'text-orange-700 bg-orange-100 border-orange-200';
    case 'STANDARD':
      return 'text-blue-700 bg-blue-100 border-blue-200';
    case 'LOW':
      return 'text-gray-700 bg-gray-100 border-gray-200';
  }
}

export function getPriorityIcon(priority: ReviewPriority): string {
  switch (priority) {
    case 'CRITICAL':
      return '🚨';
    case 'HIGH':
      return '⚠️';
    case 'STANDARD':
      return '📋';
    case 'LOW':
      return '📝';
  }
}

export function getPrioritySlaHours(priority: ReviewPriority): number {
  switch (priority) {
    case 'CRITICAL':
      return 24;
    case 'HIGH':
      return 48;
    case 'STANDARD':
      return 168; // 7 days
    case 'LOW':
      return 336; // 14 days
  }
}

// ============================================================================
// Status Helpers
// ============================================================================

export function getStatusColor(status: FactStatus): string {
  switch (status) {
    case 'DRAFT':
      return 'text-gray-700 bg-gray-100';
    case 'PENDING_REVIEW':
      return 'text-amber-700 bg-amber-100';
    case 'APPROVED':
      return 'text-green-700 bg-green-100';
    case 'ACTIVE':
      return 'text-blue-700 bg-blue-100';
    case 'REJECTED':
      return 'text-red-700 bg-red-100';
    case 'SUPERSEDED':
      return 'text-purple-700 bg-purple-100';
    case 'RETIRED':
      return 'text-gray-500 bg-gray-50';
  }
}

export function getStatusLabel(status: FactStatus): string {
  switch (status) {
    case 'DRAFT':
      return 'Draft';
    case 'PENDING_REVIEW':
      return 'Pending Review';
    case 'APPROVED':
      return 'Approved';
    case 'ACTIVE':
      return 'Active';
    case 'REJECTED':
      return 'Rejected';
    case 'SUPERSEDED':
      return 'Superseded';
    case 'RETIRED':
      return 'Retired';
  }
}

// ============================================================================
// Fact Type Helpers
// ============================================================================

export function getFactTypeLabel(factType: FactType): string {
  const labels: Record<string, string> = {
    INTERACTION: 'Drug Interaction',
    DRUG_INTERACTION: 'Drug-Drug Interaction',
    CONTRAINDICATION: 'Contraindication',
    DOSING_RULE: 'Dosing Rule',
    ALLERGY_CROSS_REACTIVITY: 'Allergy Cross-Reactivity',
    SAFETY_SIGNAL: 'Safety Signal',
    ORGAN_IMPAIRMENT: 'Organ Impairment',
    REPRODUCTIVE_SAFETY: 'Reproductive Safety',
    THERAPEUTIC_GUIDELINE: 'Therapeutic Guideline',
    LAB_REFERENCE: 'Lab Reference',
    LAB_DRUG_INTERACTION: 'Lab-Drug Interaction',
    FOOD_DRUG_INTERACTION: 'Food-Drug Interaction',
    PREGNANCY_CATEGORY: 'Pregnancy Category',
    RENAL_ADJUSTMENT: 'Renal Adjustment',
    HEPATIC_ADJUSTMENT: 'Hepatic Adjustment',
    GERIATRIC_CONSIDERATION: 'Geriatric Consideration',
    PEDIATRIC_DOSING: 'Pediatric Dosing',
    FORMULARY: 'Formulary',
  };
  return labels[factType] || factType;
}

export function getFactTypeIcon(factType: FactType): string {
  const icons: Record<string, string> = {
    INTERACTION: '⚡',
    DRUG_INTERACTION: '💊',
    CONTRAINDICATION: '🚫',
    DOSING_RULE: '📐',
    ALLERGY_CROSS_REACTIVITY: '🤧',
    SAFETY_SIGNAL: '⚠️',
    ORGAN_IMPAIRMENT: '🫁',
    REPRODUCTIVE_SAFETY: '🤰',
    THERAPEUTIC_GUIDELINE: '📋',
    LAB_REFERENCE: '🔬',
    LAB_DRUG_INTERACTION: '🧪',
    FOOD_DRUG_INTERACTION: '🍽️',
    PREGNANCY_CATEGORY: '🤰',
    RENAL_ADJUSTMENT: '🫘',
    HEPATIC_ADJUSTMENT: '🫀',
    GERIATRIC_CONSIDERATION: '👴',
    PEDIATRIC_DOSING: '👶',
    FORMULARY: '📦',
  };
  return icons[factType] || '📄';
}

// ============================================================================
// Confidence & Severity Helpers
// ============================================================================

export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.95) return 'text-green-700 bg-green-100';
  if (confidence >= 0.85) return 'text-blue-700 bg-blue-100';
  if (confidence >= 0.65) return 'text-amber-700 bg-amber-100';
  return 'text-red-700 bg-red-100';
}

export function getConfidenceLabel(confidence: number): string {
  if (confidence >= 0.95) return 'Very High';
  if (confidence >= 0.85) return 'High';
  if (confidence >= 0.65) return 'Moderate';
  return 'Low';
}

export function getSeverityColor(severity: string): string {
  switch (severity) {
    case 'CRITICAL':
      return 'text-red-700 bg-red-100';
    case 'HIGH':
      return 'text-orange-700 bg-orange-100';
    case 'MODERATE':
      return 'text-amber-700 bg-amber-100';
    case 'LOW':
      return 'text-green-700 bg-green-100';
    default:
      return 'text-gray-700 bg-gray-100';
  }
}

// ============================================================================
// Number Formatting
// ============================================================================

export function formatNumber(num: number): string {
  return new Intl.NumberFormat().format(num);
}

export function formatPercent(num: number): string {
  return `${(num * 100).toFixed(1)}%`;
}
