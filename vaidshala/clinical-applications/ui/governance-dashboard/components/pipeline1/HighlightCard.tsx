'use client';

import { AlertTriangle } from 'lucide-react';
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import { cn } from '@/lib/utils';
import type { MergedSpan, SpanReviewStatus } from '@/types/pipeline1';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface HighlightCardProps {
  span: MergedSpan;
  isSelected: boolean;
  onClick: () => void;
}

// ---------------------------------------------------------------------------
// Review status badge configuration
// ---------------------------------------------------------------------------

const REVIEW_STATUS_STYLE: Record<SpanReviewStatus, { label: string; className: string }> = {
  PENDING:   { label: 'Pending',   className: 'text-yellow-700 bg-yellow-100 border-yellow-300' },
  CONFIRMED: { label: 'Confirmed', className: 'text-green-700 bg-green-100 border-green-300' },
  REJECTED:  { label: 'Rejected',  className: 'text-red-700 bg-red-100 border-red-300' },
  EDITED:    { label: 'Edited',    className: 'text-blue-700 bg-blue-100 border-blue-300' },
  ADDED:     { label: 'Added',     className: 'text-purple-700 bg-purple-100 border-purple-300' },
};

// ---------------------------------------------------------------------------
// Left-border color mapping (Tailwind border-l colors matching channel bg)
// ---------------------------------------------------------------------------

function getLeftBorderColor(channel: string): string {
  const info = getChannelInfo(channel);
  // Map bg-{color}-100 to border-l-{color}-400 for a visible left accent
  const colorMatch = info.bg.match(/bg-(\w+)-/);
  if (colorMatch) {
    return `border-l-${colorMatch[1]}-400`;
  }
  return 'border-l-gray-400';
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function HighlightCard({ span, isSelected, onClick }: HighlightCardProps) {
  const firstChannel = span.contributingChannels[0] ?? '';
  const leftBorder = getLeftBorderColor(firstChannel);
  const statusStyle = REVIEW_STATUS_STYLE[span.reviewStatus] ?? REVIEW_STATUS_STYLE.PENDING;

  const hasOracle = span.contributingChannels.includes('L1');

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full text-left p-3 rounded-lg border border-l-4 transition-colors cursor-pointer',
        leftBorder,
        isSelected
          ? 'ring-2 ring-blue-500 bg-blue-50 border-gray-200'
          : 'border-gray-200 hover:bg-gray-50',
      )}
    >
      {/* ---- Top row: channel badges + confidence ---- */}
      <div className="flex items-center justify-between mb-1.5">
        {/* Channel badges */}
        <div className="flex items-center gap-1">
          {span.contributingChannels.map((ch) => {
            const info = getChannelInfo(ch);
            return (
              <span
                key={ch}
                className={cn(
                  'inline-flex items-center justify-center text-[10px] font-semibold leading-none px-1.5 py-0.5 rounded-full',
                  info.bg,
                  info.color,
                )}
              >
                {ch}
              </span>
            );
          })}
        </div>

        {/* Confidence number */}
        <span
          className={cn(
            'text-lg font-bold tabular-nums leading-none',
            getConfidenceColor(span.mergedConfidence),
          )}
        >
          {(span.mergedConfidence * 100).toFixed(0)}
        </span>
      </div>

      {/* ---- Middle: text preview (2-line clamp) ---- */}
      <p className="text-sm text-gray-800 line-clamp-2 my-1.5">
        {span.reviewerText ?? span.text}
      </p>

      {/* ---- Bottom row: status badge + warning icons ---- */}
      <div className="flex items-center justify-between mt-1.5">
        {/* Review status badge */}
        <span
          className={cn(
            'inline-flex items-center text-[10px] font-medium leading-none px-2 py-0.5 rounded-full border',
            statusStyle.className,
          )}
        >
          {statusStyle.label}
        </span>

        {/* Warning icons */}
        <div className="flex items-center gap-1.5">
          {hasOracle && (
            <AlertTriangle
              className="h-3.5 w-3.5 text-red-500"
              aria-label="Oracle recovery span"
            />
          )}
          {span.hasDisagreement && (
            <AlertTriangle
              className="h-3.5 w-3.5 text-amber-500"
              aria-label="Channel disagreement"
            />
          )}
        </div>
      </div>
    </button>
  );
}
