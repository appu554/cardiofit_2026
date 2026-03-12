'use client';

import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import dynamic from 'next/dynamic';
import {
  CheckCircle,
  XCircle,
  Pencil,
  Loader2,
  ChevronLeft,
  ChevronRight,
  Flag,
  ArrowUpRight,
  ShieldAlert,
  Check,
  AlertTriangle,
  Plus,
} from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import { SemanticText } from './SemanticText';
import { CoverageGuardAlertBanner } from './CoverageGuardAlert';
import { PdfErrorBoundary } from './PdfErrorBoundary';
import { RejectModal } from './RejectModal';

// Dynamic import: pdfjs-dist ESM module fails during SSR webpack bundling
const PdfHighlightViewer = dynamic(
  () => import('./PdfHighlightViewer').then((m) => m.PdfHighlightViewer),
  { ssr: false, loading: () => <div className="flex-1 flex items-center justify-center"><Loader2 className="h-6 w-6 animate-spin text-gray-300" /></div> },
);
import type {
  ExtractionJob,
  MergedSpan,
  PageInfo,
  PageStats,
  PageDecisionAction,
  SpanReviewRequest,
  RejectReason,
  RiskTier,
} from '@/types/pipeline1';

// =============================================================================
// Props
// =============================================================================

interface PageReviewModeProps {
  jobId: string;
  job: ExtractionJob;
  onBack: () => void;
  onActionComplete: () => void;
}

// =============================================================================
// Page cell colors
// =============================================================================

function pageRingColor(page: PageInfo): string {
  if (page.decision === 'ACCEPT') return 'ring-green-400 bg-green-50';
  if (page.decision === 'FLAG') return 'ring-amber-400 bg-amber-50';
  if (page.decision === 'ESCALATE') return 'ring-red-400 bg-red-50';
  if (page.risk === 'oracle') return 'ring-red-300 bg-red-50/50';
  if (page.risk === 'disagreement') return 'ring-amber-300 bg-amber-50/50';
  return 'ring-gray-200 bg-white';
}

function pageDecisionIcon(decision?: PageDecisionAction) {
  if (decision === 'ACCEPT') return <Check className="h-2.5 w-2.5 text-green-600" />;
  if (decision === 'FLAG') return <Flag className="h-2.5 w-2.5 text-amber-600" />;
  if (decision === 'ESCALATE') return <ArrowUpRight className="h-2.5 w-2.5 text-red-600" />;
  return null;
}

// =============================================================================
// Page Navigator Strip
// =============================================================================

type PageFilter = 'all' | 'undecided' | 'risk' | 'flagged';

function PageNavigator({
  pages,
  selectedPage,
  onSelect,
  filter,
  onFilterChange,
  stats,
}: {
  pages: PageInfo[];
  selectedPage: number;
  onSelect: (page: number) => void;
  filter: PageFilter;
  onFilterChange: (f: PageFilter) => void;
  stats: PageStats | null;
}) {
  const stripRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to selected page
  useEffect(() => {
    if (!stripRef.current) return;
    const el = stripRef.current.querySelector(`[data-page="${selectedPage}"]`);
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
  }, [selectedPage]);

  const filteredPages = useMemo(() => {
    switch (filter) {
      case 'undecided': return pages.filter((p) => !p.decision);
      case 'risk': return pages.filter((p) => p.risk !== 'clean');
      case 'flagged': return pages.filter((p) => p.decision === 'FLAG' || p.decision === 'ESCALATE');
      default: return pages;
    }
  }, [pages, filter]);

  const decided = stats ? stats.pagesAccepted + stats.pagesFlagged + stats.pagesEscalated : 0;
  const total = stats?.totalPages ?? pages.length;

  return (
    <div className="px-4 py-2 bg-white border-b border-gray-200 shrink-0">
      {/* Filter row + progress */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-gray-500 uppercase tracking-wider font-semibold mr-1">
            Pages:
          </span>
          {(['all', 'undecided', 'risk', 'flagged'] as PageFilter[]).map((f) => (
            <button
              key={f}
              onClick={() => onFilterChange(f)}
              className={cn(
                'text-[10px] px-2 py-0.5 rounded-full border font-medium transition-colors capitalize',
                filter === f
                  ? 'bg-gray-800 text-white border-gray-800'
                  : 'bg-gray-50 text-gray-500 border-gray-200 hover:bg-gray-100',
              )}
            >
              {f === 'all' ? `All (${pages.length})` :
               f === 'undecided' ? `Undecided (${pages.filter(p => !p.decision).length})` :
               f === 'risk' ? `Has Risk (${pages.filter(p => p.risk !== 'clean').length})` :
               `Flagged (${pages.filter(p => p.decision === 'FLAG' || p.decision === 'ESCALATE').length})`}
            </button>
          ))}
        </div>
        {stats && (
          <div className="flex items-center gap-2 text-[10px] text-gray-500">
            <span className="font-semibold text-green-700">{stats.pagesAccepted} accepted</span>
            <span>|</span>
            <span className="font-semibold text-amber-700">{stats.pagesFlagged} flagged</span>
            <span>|</span>
            <span className="font-bold text-gray-700">{decided}/{total} decided</span>
          </div>
        )}
      </div>

      {/* Page strip */}
      <div ref={stripRef} className="flex gap-1 overflow-x-auto pb-1 scrollbar-thin">
        {filteredPages.map((p) => (
          <button
            key={p.pageNumber}
            data-page={p.pageNumber}
            onClick={() => onSelect(p.pageNumber)}
            className={cn(
              'shrink-0 w-10 h-10 rounded-md ring-2 flex flex-col items-center justify-center text-[9px] font-medium transition-all relative',
              pageRingColor(p),
              selectedPage === p.pageNumber
                ? 'ring-[#1B3A5C] ring-[3px] scale-110 shadow-md z-10'
                : 'hover:scale-105',
            )}
          >
            <span className="font-bold text-[10px] text-gray-800">{p.pageNumber}</span>
            <span className="text-[8px] text-gray-400">{p.spanCount}s</span>
            {p.decision && (
              <span className="absolute -top-0.5 -right-0.5">
                {pageDecisionIcon(p.decision)}
              </span>
            )}
          </button>
        ))}
        {filteredPages.length === 0 && (
          <span className="text-[10px] text-gray-400 py-2">No pages match filter</span>
        )}
      </div>
    </div>
  );
}

// =============================================================================
// Compact Span Card (for the per-page span list)
// =============================================================================

function SpanCard({
  span,
  isExpanded,
  onClick,
  onConfirm,
  onEdit,
  onReject,
  actionLoading,
  alertBlocksConfirm,
}: {
  span: MergedSpan;
  isExpanded: boolean;
  onClick: () => void;
  onConfirm: () => void;
  onEdit: () => void;
  onReject: () => void;
  actionLoading: boolean;
  alertBlocksConfirm: boolean;
}) {
  const confidence = span.mergedConfidence ?? 0;
  const confColor = confidence >= 0.7 ? 'text-green-700' : confidence >= 0.5 ? 'text-amber-700' : 'text-red-700';
  const statusBg =
    span.reviewStatus === 'CONFIRMED' ? 'bg-green-100 text-green-700 border-green-200' :
    span.reviewStatus === 'REJECTED' ? 'bg-red-100 text-red-700 border-red-200' :
    span.reviewStatus === 'EDITED' ? 'bg-blue-100 text-blue-700 border-blue-200' :
    span.reviewStatus === 'ADDED' ? 'bg-purple-100 text-purple-700 border-purple-200' :
    'bg-gray-100 text-gray-500 border-gray-200';

  // Channel-colored left border + subtle background tint
  const primaryChannel = span.contributingChannels[0];
  const channelInfo = primaryChannel ? getChannelInfo(primaryChannel) : null;
  const channelBorder = channelInfo?.border || 'border-l-gray-300';
  const channelBgTint = channelInfo?.bgTint || '';

  return (
    <div
      className={cn(
        'border rounded-lg transition-all mb-2 border-l-[3px]',
        channelBorder,
        isExpanded
          ? 'border-r-[#1B3A5C] border-t-[#1B3A5C] border-b-[#1B3A5C] shadow-sm'
          : 'border-r-gray-200 border-t-gray-200 border-b-gray-200 hover:border-r-gray-300 hover:border-t-gray-300 hover:border-b-gray-300',
        isExpanded ? 'bg-white' : channelBgTint || 'bg-white',
        span.reviewStatus !== 'PENDING' && !isExpanded && 'opacity-70',
      )}
    >
      {/* Compact header — always visible */}
      <button
        onClick={onClick}
        className="w-full text-left px-3 py-2 flex items-center gap-2"
      >
        {/* Channel badges */}
        <div className="flex gap-0.5 shrink-0">
          {span.contributingChannels.map((ch) => {
            const info = getChannelInfo(ch);
            return (
              <span
                key={ch}
                className={cn('text-[9px] font-bold px-1 py-0.5 rounded', info.bg, info.color)}
              >
                {ch === 'L1_RECOVERY' ? 'L1' : ch}
              </span>
            );
          })}
        </div>

        {/* Confidence */}
        <span className={cn('text-[10px] font-semibold', confColor)}>
          {(confidence * 100).toFixed(0)}%
        </span>

        {/* Text preview */}
        <span className="text-xs text-gray-700 truncate flex-1">
          {span.text.slice(0, 120)}{span.text.length > 120 ? '...' : ''}
        </span>

        {/* Status badge */}
        <span className={cn('text-[9px] font-semibold px-1.5 py-0.5 rounded border shrink-0', statusBg)}>
          {span.reviewStatus === 'PENDING' ? 'PENDING' : span.reviewStatus}
        </span>

        {/* Risk indicators */}
        {span.hasDisagreement && (
          <AlertTriangle className="h-3 w-3 text-amber-500 shrink-0" />
        )}
        {span.tier != null && span.tier <= 2 && (
          <span className={cn(
            'text-[8px] font-bold px-1 py-0.5 rounded',
            span.tier === 1 ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700',
          )}>
            T{span.tier}
          </span>
        )}
      </button>

      {/* Expanded detail — shown when selected */}
      {isExpanded && (
        <div className="px-3 pb-3 border-t border-gray-100 pt-2">
          {/* CoverageGuard Alert */}
          {span.coverageGuardAlert && (
            <div className="mb-2">
              <CoverageGuardAlertBanner alert={span.coverageGuardAlert} />
            </div>
          )}

          {/* Full text with semantic highlighting */}
          <div className="bg-gray-50 rounded-md p-3 mb-2 text-sm leading-relaxed text-gray-900 font-serif">
            <SemanticText text={span.text} tokens={span.semanticTokens} />
          </div>

          {/* Disagreement detail */}
          {span.hasDisagreement && span.disagreementDetail && (
            <div className="flex items-start gap-1.5 p-2 rounded border border-amber-200 bg-amber-50 text-xs text-amber-800 mb-2">
              <AlertTriangle className="h-3.5 w-3.5 text-amber-500 mt-0.5 shrink-0" />
              <span>{span.disagreementDetail}</span>
            </div>
          )}

          {/* Channel confidence breakdown */}
          {span.contributingChannels.length > 1 && (
            <div className="flex flex-wrap gap-1.5 mb-2">
              {span.contributingChannels.map((ch) => {
                const info = getChannelInfo(ch);
                const conf = span.channelConfidences?.[ch];
                return (
                  <span key={ch} className={cn('text-[10px] px-1.5 py-0.5 rounded', info.bg, info.color)}>
                    {ch === 'L1_RECOVERY' ? 'L1' : ch}: {info.name}
                    {conf != null && ` ${(conf * 100).toFixed(0)}%`}
                  </span>
                );
              })}
            </div>
          )}

          {/* Alert-gated warning */}
          {alertBlocksConfirm && (
            <div className="flex items-start gap-2 p-2 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700 mb-2">
              <ShieldAlert className="h-3.5 w-3.5 text-red-500 mt-0.5 shrink-0" />
              <span>
                <strong>Confirm blocked</strong> — critical CoverageGuard alert. Edit or Reject instead.
              </span>
            </div>
          )}

          {/* Action buttons */}
          <div className="flex gap-1.5">
            <button
              onClick={(e) => { e.stopPropagation(); onConfirm(); }}
              disabled={alertBlocksConfirm || actionLoading || !['PENDING', 'ADDED'].includes(span.reviewStatus)}
              className={cn(
                'px-3 py-1.5 rounded text-[11px] font-semibold flex items-center gap-1 transition-colors',
                alertBlocksConfirm || !['PENDING', 'ADDED'].includes(span.reviewStatus)
                  ? 'bg-gray-200 text-gray-400 cursor-not-allowed'
                  : 'bg-green-600 text-white hover:bg-green-700',
              )}
            >
              <CheckCircle className="h-3 w-3" /> Confirm
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onEdit(); }}
              disabled={actionLoading || !['PENDING', 'ADDED'].includes(span.reviewStatus)}
              className={cn(
                'px-3 py-1.5 rounded text-[11px] font-semibold flex items-center gap-1 transition-colors',
                !['PENDING', 'ADDED'].includes(span.reviewStatus)
                  ? 'bg-gray-200 text-gray-400 cursor-not-allowed'
                  : 'bg-blue-600 text-white hover:bg-blue-700',
              )}
            >
              <Pencil className="h-3 w-3" /> Edit
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onReject(); }}
              disabled={actionLoading || !['PENDING', 'ADDED'].includes(span.reviewStatus)}
              className={cn(
                'px-3 py-1.5 rounded text-[11px] font-semibold flex items-center gap-1 transition-colors',
                !['PENDING', 'ADDED'].includes(span.reviewStatus)
                  ? 'bg-gray-200 text-gray-400 cursor-not-allowed'
                  : 'bg-red-600 text-white hover:bg-red-700',
              )}
            >
              <XCircle className="h-3 w-3" /> Reject
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// =============================================================================
// Page Decision Bar
// =============================================================================

function PageDecisionBar({
  page,
  onDecide,
  loading,
  decisionError,
}: {
  page: PageInfo;
  onDecide: (action: PageDecisionAction) => void;
  loading: boolean;
  decisionError?: string | null;
}) {
  const t1Complete = page.tier1Total === page.tier1Reviewed;
  const t1Remaining = page.tier1Total - page.tier1Reviewed;
  // ACCEPT blocked if T1 spans are still pending
  const acceptBlocked = !t1Complete && page.tier1Total > 0;

  return (
    <div className="sticky bottom-0 bg-white border-t border-gray-200 px-4 py-3">
      <div className="flex items-center justify-between">
        <div className="text-xs text-gray-500">
          <span className="font-semibold text-gray-700">Page {page.pageNumber}</span>
          {' '}&mdash;{' '}
          {page.reviewedSpans}/{page.spanCount} spans reviewed
          {page.pendingSpans > 0 && (
            <span className="text-amber-600 font-semibold ml-1">({page.pendingSpans} pending)</span>
          )}
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => onDecide('ACCEPT')}
            disabled={loading || acceptBlocked}
            className={cn(
              'px-4 py-1.5 rounded-md text-[11px] font-semibold flex items-center gap-1 transition-colors',
              page.decision === 'ACCEPT'
                ? 'bg-green-100 text-green-800 border border-green-300 ring-2 ring-green-400'
                : acceptBlocked
                  ? 'bg-gray-200 text-gray-400 cursor-not-allowed'
                  : 'bg-green-600 text-white hover:bg-green-700',
            )}
            title={acceptBlocked ? `${t1Remaining} Tier 1 safety span(s) must be reviewed` : undefined}
          >
            {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : <Check className="h-3 w-3" />}
            {page.decision === 'ACCEPT' ? 'Accepted' : 'Accept Page'}
          </button>
          <button
            onClick={() => onDecide('FLAG')}
            disabled={loading}
            className={cn(
              'px-4 py-1.5 rounded-md text-[11px] font-semibold flex items-center gap-1 transition-colors',
              page.decision === 'FLAG'
                ? 'bg-amber-100 text-amber-800 border border-amber-300 ring-2 ring-amber-400'
                : 'bg-amber-500 text-white hover:bg-amber-600',
            )}
          >
            <Flag className="h-3 w-3" />
            {page.decision === 'FLAG' ? 'Flagged' : 'Flag Page'}
          </button>
          <button
            onClick={() => onDecide('ESCALATE')}
            disabled={loading}
            className={cn(
              'px-4 py-1.5 rounded-md text-[11px] font-semibold flex items-center gap-1 transition-colors',
              page.decision === 'ESCALATE'
                ? 'bg-red-100 text-red-800 border border-red-300 ring-2 ring-red-400'
                : 'bg-red-600 text-white hover:bg-red-700',
            )}
          >
            <ArrowUpRight className="h-3 w-3" />
            {page.decision === 'ESCALATE' ? 'Escalated' : 'Escalate'}
          </button>
        </div>
      </div>
      {/* T1 warning bar */}
      {acceptBlocked && (
        <div className="mt-1.5 flex items-center gap-1.5 text-[10px] text-red-600 font-semibold">
          <ShieldAlert className="h-3 w-3" />
          {t1Remaining} Tier 1 (patient safety) span{t1Remaining !== 1 ? 's' : ''} must be reviewed before ACCEPT
        </div>
      )}
      {/* Backend 409 error display */}
      {decisionError && (
        <div className="mt-1.5 flex items-center gap-1.5 text-[10px] text-red-600 font-semibold">
          <AlertTriangle className="h-3 w-3" />
          {decisionError}
        </div>
      )}
    </div>
  );
}

// =============================================================================
// Edit Modal (inline for page review)
// =============================================================================

function EditModal({
  span,
  onSave,
  onClose,
  loading,
}: {
  span: MergedSpan;
  onSave: (text: string, note?: string) => void;
  onClose: () => void;
  loading: boolean;
}) {
  const [text, setText] = useState(span.text);
  const [note, setNote] = useState('');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-5" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-sm font-semibold text-gray-900 mb-3">Edit Fact Text</h3>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={6}
          className="w-full border border-gray-200 rounded-md p-3 text-sm leading-relaxed text-gray-900 font-serif resize-none mb-3 focus:outline-none focus:ring-2 focus:ring-blue-300"
          autoFocus
        />
        <textarea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Reviewer note (optional)..."
          rows={2}
          className="w-full border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-700 resize-none mb-3 focus:outline-none focus:ring-2 focus:ring-blue-200"
        />
        <div className="flex justify-end gap-2">
          <button onClick={onClose} className="px-4 py-2 rounded-md border border-gray-200 text-xs text-gray-600 hover:bg-gray-50">
            Cancel
          </button>
          <button
            onClick={() => onSave(text, note || undefined)}
            disabled={loading || !text.trim()}
            className="px-4 py-2 rounded-md bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"
          >
            {loading && <Loader2 className="h-3 w-3 animate-spin" />}
            Save Edit
          </button>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Add Span Modal (reviewer adds missed content)
// =============================================================================

function AddSpanModal({
  pageNumber,
  onSave,
  onClose,
  loading,
}: {
  pageNumber: number;
  onSave: (text: string, note?: string) => void;
  onClose: () => void;
  loading: boolean;
}) {
  const [text, setText] = useState('');
  const [note, setNote] = useState('');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-5" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2 mb-3">
          <Plus className="h-4 w-4 text-purple-600" />
          <h3 className="text-sm font-semibold text-gray-900">Add Missing Fact</h3>
          <span className="text-[10px] text-gray-400 ml-auto">Page {pageNumber}</span>
        </div>
        <p className="text-[11px] text-gray-500 mb-3">
          Add content that was missed by the extraction pipeline. This will be recorded as a REVIEWER-added span.
        </p>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Enter the exact text from the source document..."
          rows={6}
          className="w-full border border-gray-200 rounded-md p-3 text-sm leading-relaxed text-gray-900 font-serif resize-none mb-3 focus:outline-none focus:ring-2 focus:ring-purple-300"
          autoFocus
        />
        <textarea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Reviewer note — why was this missed? (optional)"
          rows={2}
          className="w-full border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-700 resize-none mb-3 focus:outline-none focus:ring-2 focus:ring-purple-200"
        />
        <div className="flex justify-end gap-2">
          <button onClick={onClose} className="px-4 py-2 rounded-md border border-gray-200 text-xs text-gray-600 hover:bg-gray-50">
            Cancel
          </button>
          <button
            onClick={() => onSave(text, note || undefined)}
            disabled={loading || !text.trim()}
            className="px-4 py-2 rounded-md bg-purple-600 text-white text-xs font-semibold hover:bg-purple-700 disabled:opacity-50 flex items-center gap-1"
          >
            {loading && <Loader2 className="h-3 w-3 animate-spin" />}
            <Plus className="h-3 w-3" />
            Add Fact
          </button>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// PageReviewMode — Main Component
// =============================================================================

export function PageReviewMode({
  jobId,
  job,
  onBack,
  onActionComplete,
}: PageReviewModeProps) {
  const { user } = useAuth();

  // ─── State ──────────────────────────────────────────────────────────
  const [pages, setPages] = useState<PageInfo[]>([]);
  const [pagesLoading, setPagesLoading] = useState(true);
  const [stats, setStats] = useState<PageStats | null>(null);
  const [selectedPage, setSelectedPage] = useState<number>(0);
  const [pageFilter, setPageFilter] = useState<PageFilter>('all');

  const [pageSpans, setPageSpans] = useState<MergedSpan[]>([]);
  const [spansLoading, setSpansLoading] = useState(false);
  const [expandedSpanId, setExpandedSpanId] = useState<string | null>(null);

  const [actionLoading, setActionLoading] = useState(false);
  const [showReject, setShowReject] = useState(false);
  const [rejectTarget, setRejectTarget] = useState<MergedSpan | null>(null);
  const [showEdit, setShowEdit] = useState(false);
  const [editTarget, setEditTarget] = useState<MergedSpan | null>(null);
  const [tierFilter, setTierFilter] = useState<RiskTier | null>(null);
  const [decisionError, setDecisionError] = useState<string | null>(null);
  const [showAddSpan, setShowAddSpan] = useState(false);

  const reviewerId = user?.sub || 'unknown';

  // ─── Load pages list + stats ─────────────────────────────────────────
  const fetchPages = useCallback(async () => {
    setPagesLoading(true);
    try {
      const [pageList, pageStats] = await Promise.all([
        pipeline1Api.pages.list(jobId),
        pipeline1Api.pages.getStats(jobId),
      ]);
      setPages(pageList);
      setStats(pageStats);
      // Auto-select first undecided page (or first page)
      if (selectedPage === 0) {
        const firstUndecided = pageList.find((p) => !p.decision);
        setSelectedPage(firstUndecided?.pageNumber ?? pageList[0]?.pageNumber ?? 1);
      }
    } catch {
      // silently handle
    } finally {
      setPagesLoading(false);
    }
  }, [jobId, selectedPage]);

  useEffect(() => { fetchPages(); }, [fetchPages]);

  // ─── Load spans for selected page ────────────────────────────────────
  useEffect(() => {
    if (!selectedPage) return;
    let cancelled = false;
    setSpansLoading(true);
    setExpandedSpanId(null);
    setDecisionError(null);
    setTierFilter(null);

    async function load() {
      try {
        // Fetch all spans for this page (up to 500 — tables can be large)
        const result = await pipeline1Api.spans.list(jobId, { pageNumber: selectedPage }, 1, 500);
        if (!cancelled) {
          setPageSpans(result.items);
        }
      } catch {
        // silently handle
      } finally {
        if (!cancelled) setSpansLoading(false);
      }
    }

    load();
    return () => { cancelled = true; };
  }, [jobId, selectedPage]);

  // ─── Current page info ──────────────────────────────────────────────
  const currentPageInfo = useMemo(
    () => pages.find((p) => p.pageNumber === selectedPage),
    [pages, selectedPage],
  );

  // ─── Span actions ───────────────────────────────────────────────────
  const refreshAfterAction = useCallback(async () => {
    // Reload page spans and pages (decisions may update)
    const [result, pageList, pageStats] = await Promise.all([
      pipeline1Api.spans.list(jobId, { pageNumber: selectedPage }, 1, 500),
      pipeline1Api.pages.list(jobId),
      pipeline1Api.pages.getStats(jobId),
    ]);
    setPageSpans(result.items);
    setPages(pageList);
    setStats(pageStats);
    onActionComplete();
  }, [jobId, selectedPage, onActionComplete]);

  const handleConfirmSpan = useCallback(async (span: MergedSpan) => {
    setActionLoading(true);
    try {
      await pipeline1Api.spans.confirm(jobId, span.id, { reviewerId });
      await refreshAfterAction();
    } catch {
      // silently handle
    } finally {
      setActionLoading(false);
    }
  }, [jobId, reviewerId, refreshAfterAction]);

  const handleRejectSpan = useCallback(async (reason: RejectReason) => {
    if (!rejectTarget) return;
    setActionLoading(true);
    try {
      await pipeline1Api.spans.reject(jobId, rejectTarget.id, { reviewerId, rejectReason: reason });
      setShowReject(false);
      setRejectTarget(null);
      await refreshAfterAction();
    } catch {
      // silently handle
    } finally {
      setActionLoading(false);
    }
  }, [jobId, reviewerId, rejectTarget, refreshAfterAction]);

  const handleEditSpan = useCallback(async (text: string, note?: string) => {
    if (!editTarget) return;
    setActionLoading(true);
    try {
      await pipeline1Api.spans.edit(jobId, editTarget.id, { reviewerId, editedText: text, note });
      setShowEdit(false);
      setEditTarget(null);
      await refreshAfterAction();
    } catch {
      // silently handle
    } finally {
      setActionLoading(false);
    }
  }, [jobId, reviewerId, editTarget, refreshAfterAction]);

  // ─── Add missing span ─────────────────────────────────────────────
  const handleAddSpan = useCallback(async (text: string, note?: string) => {
    setActionLoading(true);
    try {
      await pipeline1Api.spans.add(jobId, {
        text,
        startOffset: 0,
        endOffset: text.length,
        pageNumber: selectedPage,
        reviewerId,
        note,
      });
      setShowAddSpan(false);
      await refreshAfterAction();
    } catch {
      // silently handle
    } finally {
      setActionLoading(false);
    }
  }, [jobId, selectedPage, reviewerId, refreshAfterAction]);

  // ─── Page decision ──────────────────────────────────────────────────
  const [pageDecisionLoading, setPageDecisionLoading] = useState(false);

  const handlePageDecision = useCallback(async (action: PageDecisionAction) => {
    setPageDecisionLoading(true);
    setDecisionError(null);
    try {
      await pipeline1Api.pages.decide(jobId, selectedPage, {
        action,
        reviewerId,
      });
      await refreshAfterAction();
    } catch (err: unknown) {
      // Surface 409 Conflict (T1 guard) error to the user
      const msg = err instanceof Error ? err.message : 'Page decision failed';
      setDecisionError(msg);
    } finally {
      setPageDecisionLoading(false);
    }
  }, [jobId, selectedPage, reviewerId, refreshAfterAction]);

  // ─── Navigate pages ──────────────────────────────────────────────────
  const goToPage = useCallback((dir: 'prev' | 'next') => {
    const idx = pages.findIndex((p) => p.pageNumber === selectedPage);
    if (dir === 'prev' && idx > 0) setSelectedPage(pages[idx - 1].pageNumber);
    if (dir === 'next' && idx < pages.length - 1) setSelectedPage(pages[idx + 1].pageNumber);
  }, [pages, selectedPage]);

  // ─── Keyboard shortcuts ──────────────────────────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
      if (e.key === 'ArrowLeft' || e.key === 'k') goToPage('prev');
      if (e.key === 'ArrowRight' || e.key === 'j') goToPage('next');
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [goToPage]);

  // ─── Derived ─────────────────────────────────────────────────────────
  const jobName = job.sourcePdf?.replace(/^.*\//, '').replace(/\.pdf$/i, '') || 'Guideline Review';
  const expandedSpan = pageSpans.find((s) => s.id === expandedSpanId) ?? null;

  // ─── Span stats for current page ────────────────────────────────────
  const pendingOnPage = pageSpans.filter((s) => s.reviewStatus === 'PENDING').length;
  const reviewedOnPage = pageSpans.length - pendingOnPage;

  // ─── Loading state ──────────────────────────────────────────────────
  if (pagesLoading) {
    return (
      <div className="h-[calc(100vh-64px)] -m-6 flex items-center justify-center bg-[#F7F8FA]">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
          <span className="text-sm text-gray-500">Loading page data...</span>
        </div>
      </div>
    );
  }

  // ─── Render ──────────────────────────────────────────────────────────
  return (
    <div className="flex flex-col h-full bg-[#F7F8FA]">
      {/* Page Navigator Strip */}
      <PageNavigator
        pages={pages}
        selectedPage={selectedPage}
        onSelect={setSelectedPage}
        filter={pageFilter}
        onFilterChange={setPageFilter}
        stats={stats}
      />

      {/* Split content: Spans (left) + PDF (right) */}
      <div className="flex-1 flex overflow-hidden">
        {/* LEFT — Span list for selected page (48%) */}
        <div className="w-[48%] flex flex-col overflow-hidden">
          {/* Page header */}
          <div className="px-4 py-2.5 bg-white border-b border-gray-200 flex items-center justify-between shrink-0">
            <div className="flex items-center gap-3">
              <button
                onClick={() => goToPage('prev')}
                disabled={pages.findIndex(p => p.pageNumber === selectedPage) <= 0}
                className="p-1 rounded hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <ChevronLeft className="h-4 w-4 text-gray-600" />
              </button>
              <div>
                <span className="text-sm font-semibold text-[#1B3A5C]">
                  Page {selectedPage}
                </span>
                <span className="text-xs text-gray-400 ml-2">
                  {pageSpans.length} extractions
                </span>
                {currentPageInfo?.risk !== 'clean' && (
                  <span className={cn(
                    'text-[9px] font-bold px-1.5 py-0.5 rounded ml-2 uppercase',
                    currentPageInfo?.risk === 'oracle' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700',
                  )}>
                    {currentPageInfo?.risk}
                  </span>
                )}
              </div>
              <button
                onClick={() => goToPage('next')}
                disabled={pages.findIndex(p => p.pageNumber === selectedPage) >= pages.length - 1}
                className="p-1 rounded hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <ChevronRight className="h-4 w-4 text-gray-600" />
              </button>
            </div>
            <div className="flex items-center gap-2">
              <div className="text-[10px] text-gray-500">
                {reviewedOnPage}/{pageSpans.length} reviewed
                {pendingOnPage > 0 && (
                  <span className="text-amber-600 font-semibold ml-1">
                    ({pendingOnPage} pending)
                  </span>
                )}
              </div>
              <button
                onClick={() => setShowAddSpan(true)}
                className="px-2 py-1 rounded-md text-[10px] font-semibold flex items-center gap-1 bg-purple-50 text-purple-700 border border-purple-200 hover:bg-purple-100 transition-colors"
              >
                <Plus className="h-3 w-3" />
                Add Fact
              </button>
            </div>
          </div>

          {/* Channel legend */}
          {!spansLoading && pageSpans.length > 0 && (
            <div className="px-3 pt-2 pb-1 flex flex-wrap gap-1.5 border-b border-gray-100 bg-white shrink-0">
              {Array.from(new Set(pageSpans.flatMap((s) => s.contributingChannels))).map((ch) => {
                const info = getChannelInfo(ch);
                return (
                  <span
                    key={ch}
                    className={cn('text-[9px] font-semibold px-1.5 py-0.5 rounded flex items-center gap-1', info.bg, info.color)}
                  >
                    <span className={cn('w-2 h-2 rounded-sm', info.border?.replace('border-l-', 'bg-') || 'bg-gray-400')} />
                    {ch === 'L1_RECOVERY' ? 'L1' : ch} {info.name}
                  </span>
                );
              })}
            </div>
          )}

          {/* Tier filter tabs */}
          {!spansLoading && pageSpans.length > 0 && (
            <div className="px-3 py-1.5 flex items-center gap-1 border-b border-gray-100 bg-white shrink-0">
              {([null, 1, 2, 3] as (RiskTier | null)[]).map((t) => {
                const count = t === null
                  ? pageSpans.length
                  : pageSpans.filter((s) => s.tier === t).length;
                const reviewed = t === null
                  ? pageSpans.filter((s) => s.reviewStatus !== 'PENDING').length
                  : pageSpans.filter((s) => s.tier === t && s.reviewStatus !== 'PENDING').length;
                const allDone = count > 0 && reviewed === count;
                const label = t === null ? 'All' : t === 1 ? 'T1 Safety' : t === 2 ? 'T2 Clinical' : 'T3 Info';
                return (
                  <button
                    key={t ?? 'all'}
                    onClick={() => setTierFilter(t)}
                    className={cn(
                      'text-[10px] px-2 py-1 rounded-md border font-semibold transition-colors flex items-center gap-1',
                      tierFilter === t
                        ? t === 1 ? 'bg-red-600 text-white border-red-600'
                          : t === 2 ? 'bg-amber-600 text-white border-amber-600'
                          : t === 3 ? 'bg-blue-600 text-white border-blue-600'
                          : 'bg-gray-800 text-white border-gray-800'
                        : allDone
                          ? 'bg-green-50 text-green-700 border-green-200'
                          : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100',
                    )}
                  >
                    {label} ({count})
                    {allDone && tierFilter !== t && <CheckCircle className="h-2.5 w-2.5" />}
                  </button>
                );
              })}
            </div>
          )}

          {/* Span list — filtered or grouped by tier */}
          <div className="flex-1 overflow-y-auto px-3 py-2">
            {spansLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-6 w-6 animate-spin text-gray-300" />
              </div>
            ) : pageSpans.length === 0 ? (
              <div className="flex items-center justify-center py-12 text-sm text-gray-400">
                No extractions on this page
              </div>
            ) : tierFilter !== null ? (
              /* Filtered view: show only selected tier */
              pageSpans.filter((s) => s.tier === tierFilter).length === 0 ? (
                <div className="flex items-center justify-center py-12 text-sm text-gray-400">
                  No Tier {tierFilter} spans on this page
                </div>
              ) : (
                pageSpans.filter((s) => s.tier === tierFilter).map((span) => {
                  const alertBlocks =
                    span.coverageGuardAlert?.type === 'numeric_mismatch' ||
                    span.coverageGuardAlert?.type === 'branch_loss';
                  return (
                    <SpanCard
                      key={span.id}
                      span={span}
                      isExpanded={expandedSpanId === span.id}
                      onClick={() => setExpandedSpanId(expandedSpanId === span.id ? null : span.id)}
                      onConfirm={() => handleConfirmSpan(span)}
                      onEdit={() => { setEditTarget(span); setShowEdit(true); }}
                      onReject={() => { setRejectTarget(span); setShowReject(true); }}
                      actionLoading={actionLoading}
                      alertBlocksConfirm={!!alertBlocks}
                    />
                  );
                })
              )
            ) : (
              /* Grouped view: all tiers with section headers */
              ([1, 2, 3, null] as (RiskTier | null)[]).map((tier) => {
                const tierSpans = pageSpans.filter((s) =>
                  tier === null ? s.tier == null || (s.tier !== 1 && s.tier !== 2 && s.tier !== 3) : s.tier === tier
                );
                if (tierSpans.length === 0) return null;
                const reviewed = tierSpans.filter((s) => s.reviewStatus !== 'PENDING').length;
                const tierLabel = tier === 1 ? 'Tier 1: Patient Safety'
                  : tier === 2 ? 'Tier 2: Clinical Accuracy'
                  : tier === 3 ? 'Tier 3: Informational'
                  : 'Untiered';
                const tierColor = tier === 1 ? 'text-red-700 border-red-200'
                  : tier === 2 ? 'text-amber-700 border-amber-200'
                  : tier === 3 ? 'text-blue-700 border-blue-200'
                  : 'text-gray-500 border-gray-200';
                return (
                  <div key={tier ?? 'untiered'} className="mb-3">
                    <div className={cn(
                      'flex items-center gap-2 py-1 mb-1 border-b text-[10px] font-bold uppercase tracking-wider',
                      tierColor,
                    )}>
                      <span>{tierLabel}</span>
                      <span className="font-semibold text-gray-400">
                        ({reviewed}/{tierSpans.length} reviewed)
                      </span>
                      {reviewed === tierSpans.length && <CheckCircle className="h-2.5 w-2.5 text-green-500" />}
                    </div>
                    {tierSpans.map((span) => {
                      const alertBlocks =
                        span.coverageGuardAlert?.type === 'numeric_mismatch' ||
                        span.coverageGuardAlert?.type === 'branch_loss';
                      return (
                        <SpanCard
                          key={span.id}
                          span={span}
                          isExpanded={expandedSpanId === span.id}
                          onClick={() => setExpandedSpanId(expandedSpanId === span.id ? null : span.id)}
                          onConfirm={() => handleConfirmSpan(span)}
                          onEdit={() => { setEditTarget(span); setShowEdit(true); }}
                          onReject={() => { setRejectTarget(span); setShowReject(true); }}
                          actionLoading={actionLoading}
                          alertBlocksConfirm={!!alertBlocks}
                        />
                      );
                    })}
                  </div>
                );
              })
            )}
          </div>

          {/* Page Decision Bar */}
          {currentPageInfo && (
            <PageDecisionBar
              page={currentPageInfo}
              onDecide={handlePageDecision}
              loading={pageDecisionLoading}
              decisionError={decisionError}
            />
          )}
        </div>

        {/* RIGHT — PDF Panel (52%) */}
        <div className="flex-1 bg-gray-50 border-l border-gray-200 flex flex-col overflow-hidden">
          {/* PDF header */}
          <div className="px-5 py-2.5 border-b border-gray-200 bg-white shrink-0 flex justify-between items-center">
            <span className="text-xs font-semibold text-[#1B3A5C]">
              Source PDF — Page {selectedPage}
            </span>
            <span className="text-[10px] text-gray-400">
              {expandedSpan ? 'Highlighting selected span' : 'Showing full page'}
            </span>
          </div>
          {/* PDF viewer pinned to page */}
          <div className="flex-1 min-h-0">
            <PdfErrorBoundary>
              <PdfHighlightViewer
                jobId={jobId}
                page={selectedPage}
                highlightText={expandedSpan?.text}
                pdfBbox={expandedSpan?.bbox}
                useBbox={true}
              />
            </PdfErrorBoundary>
          </div>
        </div>
      </div>

      {/* Modals */}
      {showReject && rejectTarget && (
        <RejectModal
          onClose={() => { setShowReject(false); setRejectTarget(null); }}
          onConfirm={handleRejectSpan}
          isLoading={actionLoading}
        />
      )}
      {showEdit && editTarget && (
        <EditModal
          span={editTarget}
          onSave={handleEditSpan}
          onClose={() => { setShowEdit(false); setEditTarget(null); }}
          loading={actionLoading}
        />
      )}
      {showAddSpan && (
        <AddSpanModal
          pageNumber={selectedPage}
          onSave={handleAddSpan}
          onClose={() => setShowAddSpan(false)}
          loading={actionLoading}
        />
      )}
    </div>
  );
}
