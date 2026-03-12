'use client';

import { useQuery } from '@tanstack/react-query';
import { Loader2, Check, Flag, ArrowUpCircle } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { RISK_CONFIG } from '@/lib/pipeline1-channels';
import { cn } from '@/lib/utils';
import type { PageInfo, PageDecisionAction } from '@/types/pipeline1';

// ---------------------------------------------------------------------------
// Decision badge config
// ---------------------------------------------------------------------------

const DECISION_BADGE: Record<
  PageDecisionAction,
  { icon: typeof Check; color: string; label: string }
> = {
  ACCEPT: {
    icon: Check,
    color: 'text-green-700 bg-green-100 border-green-300',
    label: 'Accepted',
  },
  FLAG: {
    icon: Flag,
    color: 'text-amber-700 bg-amber-100 border-amber-300',
    label: 'Flagged',
  },
  ESCALATE: {
    icon: ArrowUpCircle,
    color: 'text-red-700 bg-red-100 border-red-300',
    label: 'Escalated',
  },
};

// ---------------------------------------------------------------------------
// Risk to left-bar color mapping (Tailwind border-l classes)
// ---------------------------------------------------------------------------

const RISK_BAR_COLOR: Record<string, string> = {
  clean: 'border-l-green-500',
  oracle: 'border-l-red-500',
  disagreement: 'border-l-amber-500',
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface PageNavigatorProps {
  jobId: string;
  activePage: number | null;
  onSelectPage: (pageNumber: number) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function PageNavigator({ jobId, activePage, onSelectPage }: PageNavigatorProps) {
  const { data: pages, isLoading } = useQuery<PageInfo[]>({
    queryKey: ['pipeline1-pages', jobId],
    queryFn: () => pipeline1Api.pages.list(jobId),
  });

  // Loading skeleton
  if (isLoading) {
    return (
      <div className="p-3 space-y-2" role="status" aria-label="Loading pages">
        <Loader2 className="h-4 w-4 animate-spin text-gray-400 mx-auto mb-2" />
        {[...Array(8)].map((_, i) => (
          <div
            key={i}
            className="h-16 bg-gray-200 rounded-lg animate-pulse"
            style={{ opacity: 1 - i * 0.08 }}
          />
        ))}
      </div>
    );
  }

  // Empty state
  if (!pages || pages.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No pages found for this job.
      </div>
    );
  }

  return (
    <nav
      className="overflow-y-auto max-h-[calc(100vh-12rem)]"
      aria-label="PDF page navigation"
    >
      <p className="px-3 py-2 text-xs font-semibold text-gray-400 uppercase tracking-wider">
        Pages ({pages.length})
      </p>

      <div className="px-2 pb-2 space-y-1">
        {pages.map((page) => (
          <PageButton
            key={page.pageNumber}
            page={page}
            isActive={activePage === page.pageNumber}
            onSelect={onSelectPage}
          />
        ))}
      </div>
    </nav>
  );
}

// ---------------------------------------------------------------------------
// Individual page button
// ---------------------------------------------------------------------------

interface PageButtonProps {
  page: PageInfo;
  isActive: boolean;
  onSelect: (pageNumber: number) => void;
}

function PageButton({ page, isActive, onSelect }: PageButtonProps) {
  const riskCfg = RISK_CONFIG[page.risk] || RISK_CONFIG.clean;
  const barColor = RISK_BAR_COLOR[page.risk] || 'border-l-gray-300';
  const decision = page.decision ? DECISION_BADGE[page.decision] : null;
  const DecisionIcon = decision?.icon;

  const allReviewed = page.spanCount > 0 && page.pendingSpans === 0;

  return (
    <button
      onClick={() => onSelect(page.pageNumber)}
      aria-current={isActive ? 'page' : undefined}
      aria-label={`Page ${page.pageNumber}, ${riskCfg.label} risk, ${page.spanCount} spans, ${page.pendingSpans} pending`}
      className={cn(
        'w-full text-left rounded-lg border-l-4 border border-gray-200 p-2.5 transition-all',
        barColor,
        isActive
          ? 'ring-2 ring-blue-500 bg-blue-50 border-blue-200'
          : 'hover:bg-gray-50 hover:border-gray-300'
      )}
    >
      {/* Row 1: Page number + risk label */}
      <div className="flex items-center justify-between mb-1">
        <span className={cn('text-sm font-semibold', isActive ? 'text-blue-800' : 'text-gray-800')}>
          Page {page.pageNumber}
        </span>
        <span
          className={cn(
            'inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded-full border',
            riskCfg.color,
            riskCfg.bg,
            riskCfg.border
          )}
        >
          {riskCfg.label}
        </span>
      </div>

      {/* Row 2: Section ID badges */}
      {page.sectionIds.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-1.5">
          {page.sectionIds.map((sid) => (
            <span
              key={sid}
              className="text-[10px] text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded truncate max-w-[7rem]"
              title={sid}
            >
              {sid}
            </span>
          ))}
        </div>
      )}

      {/* Row 3: Span counts + decision badge */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-gray-500">
          <span>{page.spanCount} spans</span>
          <span className="text-gray-300">|</span>
          {allReviewed ? (
            <span className="text-green-600 font-medium">
              {page.reviewedSpans} reviewed
            </span>
          ) : (
            <>
              <span className="text-amber-600">{page.pendingSpans} pending</span>
              {page.reviewedSpans > 0 && (
                <span className="text-green-600">{page.reviewedSpans} done</span>
              )}
            </>
          )}
        </div>

        {decision && DecisionIcon && (
          <span
            className={cn(
              'inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded-full border',
              decision.color
            )}
          >
            <DecisionIcon className="h-3 w-3 mr-0.5" />
            {decision.label}
          </span>
        )}
      </div>
    </button>
  );
}
