'use client';

import { AlertTriangle, CheckCircle, XCircle, Pencil, Plus } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { MergedSpan } from '@/types/pipeline1';

interface SpanReviewCardProps {
  span: MergedSpan;
  isSelected: boolean;
  onSelect: (span: MergedSpan) => void;
}

const STATUS_BADGE: Record<string, { icon: typeof CheckCircle; color: string; label: string }> = {
  PENDING:   { icon: AlertTriangle, color: 'text-amber-600 bg-amber-50 border-amber-200', label: 'Pending' },
  CONFIRMED: { icon: CheckCircle,   color: 'text-green-600 bg-green-50 border-green-200', label: 'Confirmed' },
  REJECTED:  { icon: XCircle,       color: 'text-red-600 bg-red-50 border-red-200',       label: 'Rejected' },
  EDITED:    { icon: Pencil,        color: 'text-blue-600 bg-blue-50 border-blue-200',    label: 'Edited' },
  ADDED:     { icon: Plus,          color: 'text-purple-600 bg-purple-50 border-purple-200', label: 'Added' },
};

export function SpanReviewCard({ span, isSelected, onSelect }: SpanReviewCardProps) {
  const badge = STATUS_BADGE[span.reviewStatus] || STATUS_BADGE.PENDING;
  const BadgeIcon = badge.icon;

  return (
    <button
      onClick={() => onSelect(span)}
      className={cn(
        'w-full text-left p-3 rounded-lg border transition-all',
        isSelected
          ? 'border-blue-400 bg-blue-50 ring-1 ring-blue-200'
          : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
      )}
    >
      {/* Header: status badge + confidence */}
      <div className="flex items-center justify-between mb-1.5">
        <span className={cn('inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full border', badge.color)}>
          <BadgeIcon className="h-3 w-3 mr-1" />
          {badge.label}
        </span>
        <span className={cn(
          'text-xs font-medium px-1.5 py-0.5 rounded',
          span.mergedConfidence >= 0.85 ? 'text-green-700 bg-green-100' :
          span.mergedConfidence >= 0.65 ? 'text-amber-700 bg-amber-100' :
          'text-red-700 bg-red-100'
        )}>
          {(span.mergedConfidence * 100).toFixed(0)}%
        </span>
      </div>

      {/* Text preview */}
      <p className="text-sm text-gray-800 line-clamp-2">{span.text}</p>

      {/* Footer: channels + disagreement */}
      <div className="flex items-center justify-between mt-2">
        <div className="flex gap-1">
          {span.contributingChannels.map((ch) => (
            <span key={ch} className="text-xs text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">
              {ch}
            </span>
          ))}
        </div>
        {span.hasDisagreement && (
          <span className="text-xs text-amber-600 flex items-center">
            <AlertTriangle className="h-3 w-3 mr-0.5" />
            Disagree
          </span>
        )}
      </div>

      {/* Page / section info */}
      {(span.pageNumber != null || span.sectionId) && (
        <div className="mt-1.5 text-xs text-gray-400">
          {span.pageNumber != null && <span>p.{span.pageNumber}</span>}
          {span.pageNumber != null && span.sectionId && <span> | </span>}
          {span.sectionId && <span>{span.sectionId}</span>}
        </div>
      )}
    </button>
  );
}
