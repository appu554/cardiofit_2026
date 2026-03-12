'use client';

import { useMemo, useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Loader2, Search, Copy, Check } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import { getChannelInfo, getConfidenceColor } from '@/lib/pipeline1-channels';
import type { MergedSpan } from '@/types/pipeline1';

interface TextViewerProps {
  jobId: string;
  spans: MergedSpan[];
  selectedSpan: MergedSpan | null;
  onSelectSpan: (span: MergedSpan) => void;
  /** When set, shows ONLY this page's text slice (not the full document) */
  activePageNumber?: number | null;
  /** When set, dims all spans except this one (task-driven focus) */
  focusSpanId?: string | null;
}

const STATUS_HIGHLIGHT: Record<string, string> = {
  PENDING:   'bg-yellow-100 hover:bg-yellow-200 border-b-2 border-yellow-300',
  CONFIRMED: 'bg-green-100 hover:bg-green-200 border-b-2 border-green-300',
  REJECTED:  'bg-red-100 hover:bg-red-200 border-b-2 border-red-300',
  EDITED:    'bg-blue-100 hover:bg-blue-200 border-b-2 border-blue-300',
  ADDED:     'bg-purple-100 hover:bg-purple-200 border-b-2 border-purple-300',
};

const STATUS_BADGE: Record<string, string> = {
  PENDING:   'bg-yellow-100 text-yellow-800',
  CONFIRMED: 'bg-green-100 text-green-800',
  REJECTED:  'bg-red-100 text-red-800',
  EDITED:    'bg-blue-100 text-blue-800',
  ADDED:     'bg-purple-100 text-purple-800',
};

interface TextSegment {
  text: string;
  span?: MergedSpan;
}

export function TextViewer({ jobId, spans, selectedSpan, onSelectSpan, activePageNumber, focusSpanId }: TextViewerProps) {
  const [spanSearch, setSpanSearch] = useState('');
  const [copied, setCopied] = useState(false);

  const { data: normalizedText, isLoading } = useQuery({
    queryKey: ['pipeline1-text', jobId],
    queryFn: () => pipeline1Api.context.getText(jobId),
  });

  // Separate positioned vs unpositioned spans
  const { positionedSpans, hasPositionedSpans } = useMemo(() => {
    const positioned = spans.filter((s) => s.startOffset >= 0 && s.endOffset >= 0);
    return { positionedSpans: positioned, hasPositionedSpans: positioned.length > 0 };
  }, [spans]);

  // ── Parse page delimiters: {N}----...---- markers in the normalized text ──
  const pageDelimiters = useMemo(() => {
    if (!normalizedText) return [];
    const delims: { pageNum: number; markerStart: number; contentStart: number }[] = [];
    // Match patterns like {0}---...--- or {12}---...---
    const regex = /\{(\d+)\}-{2,}/g;
    let match: RegExpExecArray | null;
    while ((match = regex.exec(normalizedText)) !== null) {
      const pageNum = parseInt(match[1], 10);
      const markerStart = match.index;
      // Content starts after the marker + any trailing newline
      let contentStart = markerStart + match[0].length;
      if (normalizedText[contentStart] === '\n') contentStart++;
      delims.push({ pageNum, markerStart, contentStart });
    }
    return delims;
  }, [normalizedText]);

  // ── Page-scoped text: use page delimiters to get the FULL page text ──
  // Delimiters are 0-indexed ({0}, {1}, ...) while API pageNumber is 1-indexed (1, 2, ...)
  // Mapping: API page N → delimiter {N-1}
  const pageTextSlice = useMemo(() => {
    if (activePageNumber == null || !normalizedText || pageDelimiters.length === 0) return null;

    const delimPage = activePageNumber - 1; // 0-indexed
    const delimIdx = pageDelimiters.findIndex((d) => d.pageNum === delimPage);
    if (delimIdx === -1) return null;

    const start = pageDelimiters[delimIdx].contentStart;
    // End at the next page's marker start, or end of text
    const end = delimIdx + 1 < pageDelimiters.length
      ? pageDelimiters[delimIdx + 1].markerStart
      : normalizedText.length;

    return { start, end, text: normalizedText.slice(start, end).trimEnd() };
  }, [activePageNumber, normalizedText, pageDelimiters]);

  // ── Build segments: page-scoped when page selected, full-document otherwise ──
  const segments = useMemo(() => {
    if (!normalizedText || !hasPositionedSpans) return null;

    // Page-scoped: only spans on the selected page, only that text range
    if (pageTextSlice && activePageNumber != null) {
      const { start, end } = pageTextSlice;
      const pageSpans = positionedSpans
        .filter((s) => s.pageNumber === activePageNumber && s.startOffset < end && s.endOffset > start)
        .sort((a, b) => a.startOffset - b.startOffset);

      const result: TextSegment[] = [];
      let cursor = start;

      for (const span of pageSpans) {
        const spanStart = Math.max(span.startOffset, start);
        const spanEnd = Math.min(span.endOffset, end);

        if (spanStart > cursor) {
          result.push({ text: normalizedText.slice(cursor, spanStart) });
        }

        result.push({
          text: normalizedText.slice(spanStart, spanEnd),
          span,
        });

        cursor = spanEnd;
      }

      if (cursor < end) {
        result.push({ text: normalizedText.slice(cursor, end) });
      }

      return result;
    }

    // Full-document: all positioned spans across all pages
    const sorted = [...positionedSpans].sort((a, b) => a.startOffset - b.startOffset);
    const result: TextSegment[] = [];
    let cursor = 0;

    for (const span of sorted) {
      if (span.startOffset >= normalizedText.length) continue;
      const end = Math.min(span.endOffset, normalizedText.length);

      if (span.startOffset > cursor) {
        result.push({ text: normalizedText.slice(cursor, span.startOffset) });
      }

      result.push({
        text: normalizedText.slice(span.startOffset, end),
        span,
      });

      cursor = end;
    }

    if (cursor < normalizedText.length) {
      result.push({ text: normalizedText.slice(cursor) });
    }

    return result;
  }, [normalizedText, positionedSpans, hasPositionedSpans, pageTextSlice, activePageNumber]);

  // Filter spans for the card list (search + page filter)
  const filteredSpans = useMemo(() => {
    let result = spans;
    if (activePageNumber != null) {
      result = result.filter((s) => s.pageNumber === activePageNumber);
    }
    if (spanSearch.trim()) {
      const q = spanSearch.toLowerCase();
      result = result.filter((s) => s.text.toLowerCase().includes(q));
    }
    return result;
  }, [spans, activePageNumber, spanSearch]);

  // Copy page text to clipboard
  const handleCopy = useCallback(() => {
    if (!pageTextSlice) return;
    navigator.clipboard.writeText(pageTextSlice.text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [pageTextSlice]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!normalizedText) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400 text-sm">
        No normalized text available for this job
      </div>
    );
  }

  // ── Positioned spans: overlay on text ──
  if (segments) {
    const isPageScoped = pageTextSlice != null && activePageNumber != null;
    const pageSpanCount = isPageScoped
      ? positionedSpans.filter((s) => s.pageNumber === activePageNumber).length
      : 0;

    return (
      <div className="flex flex-col h-full">
        {/* Page header with copy button (only when page-scoped) */}
        {isPageScoped && (
          <div className="shrink-0 flex items-center justify-between px-6 py-2.5 border-b border-gray-200 bg-gray-50">
            <div className="flex items-center gap-2 text-xs text-gray-600">
              <span className="font-semibold text-gray-800">Page {activePageNumber}</span>
              <span className="text-gray-300">|</span>
              <span>{pageSpanCount} spans</span>
            </div>
            <button
              onClick={handleCopy}
              className={cn(
                'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors',
                copied
                  ? 'bg-green-100 text-green-700'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 hover:text-gray-800',
              )}
            >
              {copied ? (
                <>
                  <Check className="h-3 w-3" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  Copy Page Text
                </>
              )}
            </button>
          </div>
        )}

        {/* Text content */}
        <div className="flex-1 overflow-y-auto p-6 whitespace-pre-wrap text-sm text-gray-700 font-mono leading-relaxed">
          {segments.map((seg, i) => {
            if (!seg.span) {
              return <span key={i}>{seg.text}</span>;
            }

            const isSelected = selectedSpan?.id === seg.span.id;
            // In page-scoped mode, no dimming needed (only page spans shown)
            // In full-document mode, use focusSpanId or page-based dimming
            const isDimmed = isPageScoped
              ? false
              : focusSpanId
                ? seg.span.id !== focusSpanId
                : (activePageNumber != null && seg.span.pageNumber !== activePageNumber);

            return (
              <span
                key={seg.span.id}
                id={`span-${seg.span.id}`}
                onClick={() => onSelectSpan(seg.span!)}
                className={cn(
                  'cursor-pointer rounded-sm transition-colors inline',
                  isDimmed
                    ? 'bg-gray-100 text-gray-400 border-b-2 border-gray-200'
                    : STATUS_HIGHLIGHT[seg.span.reviewStatus] || STATUS_HIGHLIGHT.PENDING,
                  isSelected && 'ring-2 ring-blue-500 ring-offset-1'
                )}
                title={`${seg.span.reviewStatus} | ${(seg.span.mergedConfidence * 100).toFixed(0)}% | ${seg.span.contributingChannels.map(ch => getChannelInfo(ch).name).join(', ')}`}
              >
                {seg.text}
              </span>
            );
          })}
        </div>
      </div>
    );
  }

  // ── Unpositioned spans: card list view ──
  return (
    <div className="flex flex-col h-full">
      {/* Search + filter header */}
      <div className="p-3 border-b border-gray-200 bg-white shrink-0">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-gray-400" />
          <input
            type="text"
            placeholder="Search spans..."
            value={spanSearch}
            onChange={(e) => setSpanSearch(e.target.value)}
            className="w-full pl-8 pr-3 py-1.5 text-xs border border-gray-200 rounded-md focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center justify-between mt-2 text-xs text-gray-500">
          <span>
            {filteredSpans.length} of {spans.length} spans
            {activePageNumber != null && ` (page ${activePageNumber})`}
          </span>
          <span className="text-gray-400">Offsets not available — showing card view</span>
        </div>
      </div>

      {/* Span cards */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1.5">
        {filteredSpans.length === 0 ? (
          <div className="text-center py-8 text-gray-400 text-xs">
            {spanSearch ? 'No spans match your search' : 'No spans available'}
          </div>
        ) : (
          filteredSpans.map((span) => {
            const isSelected = selectedSpan?.id === span.id;
            const channels = span.contributingChannels || [];

            return (
              <div
                key={span.id}
                id={`span-${span.id}`}
                onClick={() => onSelectSpan(span)}
                className={cn(
                  'p-2.5 rounded-md border cursor-pointer transition-all text-xs',
                  isSelected
                    ? 'border-blue-500 ring-2 ring-blue-200 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300 bg-white hover:bg-gray-50'
                )}
              >
                {/* Header: channels + confidence + status */}
                <div className="flex items-center justify-between mb-1.5">
                  <div className="flex items-center gap-1">
                    {channels.map((ch) => {
                      const info = getChannelInfo(ch);
                      return (
                        <span key={ch} className={cn('px-1.5 py-0.5 rounded text-[10px] font-medium', info.bg, info.color)}>
                          {info.name}
                        </span>
                      );
                    })}
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className={cn('font-mono text-[10px]', getConfidenceColor(span.mergedConfidence))}>
                      {(span.mergedConfidence * 100).toFixed(0)}%
                    </span>
                    <span className={cn('px-1.5 py-0.5 rounded text-[10px] font-medium', STATUS_BADGE[span.reviewStatus] || STATUS_BADGE.PENDING)}>
                      {span.reviewStatus}
                    </span>
                  </div>
                </div>

                {/* Span text */}
                <p className="text-gray-700 leading-relaxed line-clamp-3">
                  {span.reviewerText || span.text}
                </p>

                {/* Footer: page + disagreement */}
                <div className="flex items-center gap-2 mt-1.5 text-[10px] text-gray-400">
                  {span.pageNumber != null && <span>p.{span.pageNumber}</span>}
                  {span.hasDisagreement && (
                    <span className="text-amber-600 font-medium">Disagreement</span>
                  )}
                  {span.sectionId && <span className="truncate max-w-[120px]">{span.sectionId}</span>}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
