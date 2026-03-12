'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Document, Page, pdfjs } from 'react-pdf';
import type { CustomTextRenderer } from 'react-pdf/dist/esm/shared/types.js';
import { Loader2, AlertTriangle, ChevronLeft, ChevronRight, Search, X, ArrowUp, ArrowDown } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { CHANNEL_HIGHLIGHT_COLORS } from '@/types/pipeline1';

import 'react-pdf/dist/esm/Page/TextLayer.css';
import 'react-pdf/dist/esm/Page/AnnotationLayer.css';

// PDF.js worker — CDN for reliable Next.js compatibility
pdfjs.GlobalWorkerOptions.workerSrc = `//unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

/** A span to be highlighted on the PDF page with channel-specific coloring. */
export interface PdfHighlightSpan {
  text: string;
  channels: string[];
  bbox?: [number, number, number, number];
  id: string;
  isSelected?: boolean;
}

interface PdfHighlightViewerProps {
  jobId: string;
  /** Physical PDF page number (1-indexed) */
  page?: number;
  /** Text to search for and highlight on the rendered page */
  highlightText?: string;
  /**
   * When true and no stored bbox, uses DOM-based full-text search to find
   * the exact span text in the text layer and draws a bounding-box overlay.
   * Falls back to "not found" if the text isn't on this page.
   */
  useBbox?: boolean;
  /**
   * Stored bounding box from pipeline extraction [x0, y0, x1, y1] in PDF
   * points (1 pt = 1/72 inch). When provided, bypasses text-layer search
   * entirely and draws a pixel-perfect overlay at the exact extraction
   * coordinates. Available for L1_RECOVERY spans (from PyMuPDF rawdict).
   */
  pdfBbox?: [number, number, number, number];
  /**
   * Multi-channel highlighting: render multiple overlay boxes with
   * channel-specific colors. When provided, overrides single-span behavior.
   * Each span with a bbox renders directly; spans without attempt text search.
   * Selected span gets thicker border and higher opacity.
   */
  highlightSpans?: PdfHighlightSpan[];
}

type HighlightStatus = 'pending' | 'found' | 'not-found' | 'stored-bbox';

// ─── Normalize text for comparison ──────────────────────────────────────
// Collapse whitespace, strip HTML entities, lowercase, trim
function normalizeForSearch(text: string): string {
  return text
    .replace(/[\r\n\t]+/g, ' ')
    .replace(/[""'']/g, '"')
    .replace(/[–—]/g, '-')
    .replace(/≥/g, '>=')
    .replace(/≤/g, '<=')
    .replace(/‡/g, '>=')    // KDIGO PDFs use ‡ as ≥ in text layer
    // Fix PDF ligature artifacts: "fi " → "fi", "fl " → "fl"
    .replace(/fi\s(?=[a-z])/g, 'fi')
    .replace(/fl\s(?=[a-z])/g, 'fl')
    .replace(/\s+/g, ' ')
    // Normalize spacing around comparison operators so "< 90" and "<90" both
    // become "<90". Critical for clinical text where pipelines may strip spaces.
    .replace(/\s*([<>]=?)\s*/g, '$1')
    .trim()
    .toLowerCase();
}

/**
 * Renders a single PDF page with text highlighting.
 *
 * Four highlight strategies (in priority order):
 * 1. **Stored bbox** (`pdfBbox`): pixel-perfect overlay from pipeline coords.
 *    No text-layer search needed. Works for tables/figures.
 * 2. **DOM full-text search** (`useBbox` without `pdfBbox`): after the text
 *    layer renders, searches for the EXACT span text (or progressively
 *    shorter prefixes) in the concatenated DOM text. Computes a tight bbox
 *    from the matching text nodes. Eliminates false positives from keyword
 *    matching on medical text.
 * 3. **Inline marks** (default, no `useBbox`): wraps matched keywords in
 *    amber <mark> for visual scanning.
 * 4. **Not-found fallback**: shows reviewer guidance when text isn't on page.
 */
export function PdfHighlightViewer({
  jobId,
  page,
  highlightText,
  useBbox,
  pdfBbox,
  highlightSpans,
}: PdfHighlightViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const pageWrapperRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [pageOriginalWidth, setPageOriginalWidth] = useState(0);
  const [highlightStatus, setHighlightStatus] = useState<HighlightStatus>('pending');
  const [bboxRect, setBboxRect] = useState<{
    left: number; top: number; width: number; height: number;
  } | null>(null);

  // Manual page navigation
  const [totalPages, setTotalPages] = useState(0);
  const [manualPage, setManualPage] = useState<number | null>(null);

  // Effective page: prop page → manual override → null (no page)
  const effectivePage = manualPage ?? page ?? null;

  // ─── User search bar state ─────────────────────────────────────────
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMatchCount, setSearchMatchCount] = useState(0);
  const [currentSearchIdx, setCurrentSearchIdx] = useState(-1);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchMarksRef = useRef<HTMLElement[]>([]);

  const pdfUrl = useMemo(
    () => pipeline1Api.context.getSourcePdfUrl(jobId),
    [jobId],
  );

  // Reset manual page override when span changes (new page prop)
  useEffect(() => {
    setManualPage(null);
  }, [page, highlightText]);

  // Track container width for responsive page sizing
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width;
      if (w && w > 0) setContainerWidth(w);
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Reset state when span or page changes
  useEffect(() => {
    setBboxRect(null);
    setPageOriginalWidth(0);
    setHighlightStatus(pdfBbox ? 'stored-bbox' : 'pending');
  }, [highlightText, page, pdfBbox]);

  // ─── Strategy 1: Stored bbox from pipeline ─────────────────────────
  const handlePageLoadSuccess = useCallback((pageData: any) => {
    const origW =
      pageData?.originalWidth ||
      pageData?.width ||
      (pageData?.view ? pageData.view[2] - pageData.view[0] : 0);
    if (origW > 0) setPageOriginalWidth(origW);
  }, []);

  // Compute overlay position from stored PyMuPDF coordinates
  useEffect(() => {
    if (!pdfBbox || pageOriginalWidth <= 0 || containerWidth <= 0) return;
    const scale = containerWidth / pageOriginalWidth;
    const [x0, y0, x1, y1] = pdfBbox;
    const pad = 6;
    setBboxRect({
      left: x0 * scale - pad,
      top: y0 * scale - pad,
      width: (x1 - x0) * scale + 2 * pad,
      height: (y1 - y0) * scale + 2 * pad,
    });
    setHighlightStatus('stored-bbox');
  }, [pdfBbox, pageOriginalWidth, containerWidth]);

  // Scroll to stored bbox overlay after it renders
  useEffect(() => {
    if (pdfBbox && bboxRect && containerRef.current) {
      const timer = setTimeout(() => {
        containerRef.current?.scrollTo({
          top: Math.max(0, bboxRect.top - 100),
          behavior: 'smooth',
        });
      }, 120);
      return () => clearTimeout(timer);
    }
  }, [pdfBbox, bboxRect]);

  // ─── Strategy 3: Inline marks (non-bbox mode only) ──────────────────
  // Build a keyword regex ONLY for inline mark mode (useBbox=false).
  // For useBbox=true, we use DOM-based full-text search instead.
  const highlightPattern = useMemo(() => {
    if (!highlightText || pdfBbox || useBbox) return null;
    const escape = (w: string) => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

    // Numeric anchors — specific to a span location
    const numericAnchors = (highlightText.match(/\d[\d.,%/]+/g) || [])
      .filter((n) => n.length >= 2)
      .map(escape)
      .slice(0, 2);

    // Distinctive words — longest first (longer words = fewer false positives)
    const words = highlightText
      .split(/[\s().,;:≥≤<>\/\-]+/)
      .filter((w) => w.length >= 6)
      .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
    const uniqueWords = Array.from(new Set(words))
      .sort((a, b) => b.length - a.length)
      .slice(0, 3);

    const allPatterns = Array.from(new Set([...numericAnchors, ...uniqueWords]));
    if (allPatterns.length === 0) {
      const shortWords = highlightText
        .split(/[\s().,;:≥≤<>\/\-]+/)
        .filter((w) => w.length >= 4)
        .map(escape);
      const few = Array.from(new Set(shortWords))
        .sort((a, b) => b.length - a.length)
        .slice(0, 3);
      if (few.length === 0) return null;
      return new RegExp(`(${few.join('|')})`, 'gi');
    }
    return new RegExp(`(${allPatterns.join('|')})`, 'gi');
  }, [highlightText, pdfBbox, useBbox]);

  // Inline mark renderer (only for non-bbox mode)
  const textRenderer = useCallback<CustomTextRenderer>(
    ({ str }) => {
      if (!highlightPattern) return str;
      return str.replace(
        highlightPattern,
        (match) =>
          `<mark style="background-color:rgba(251,191,36,0.5);padding:1px 2px;border-radius:2px;">${match}</mark>`,
      );
    },
    [highlightPattern],
  );

  // ─── Strategy 2: DOM-based full-text search (useBbox mode) ──────────
  // After the text layer renders, search the concatenated DOM text for the
  // span text. This eliminates false-positive keyword matches on medical
  // pages where terms like "potassium", "30", "eGFR" repeat in different
  // contexts. Finds the EXACT span text (or its longest matching prefix)
  // and computes a tight bounding box from the matching DOM elements.
  const domFullTextSearch = useCallback(() => {
    if (!pageWrapperRef.current || !highlightText) {
      setHighlightStatus('not-found');
      return;
    }

    const textLayer = pageWrapperRef.current.querySelector(
      '.react-pdf__Page__textContent',
    );
    if (!textLayer) {
      setHighlightStatus('not-found');
      return;
    }

    // Collect all text spans from the text layer
    const textSpans = Array.from(
      textLayer.querySelectorAll('span'),
    ) as HTMLSpanElement[];
    if (textSpans.length === 0) {
      setHighlightStatus('not-found');
      return;
    }

    // Build full page text by concatenating text items with spaces
    // (PDF text items are positioned absolutely; no explicit spaces)
    const fragments: { el: HTMLSpanElement; text: string; start: number }[] = [];
    let offset = 0;
    for (const el of textSpans) {
      const raw = el.textContent || '';
      if (!raw.trim()) {
        offset += 1; // account for empty items
        continue;
      }
      fragments.push({ el, text: raw, start: offset });
      offset += raw.length + 1; // +1 for implicit space between items
    }

    const fullText = fragments.map((f) => f.text).join(' ');
    const normalizedFull = normalizeForSearch(fullText);
    const normalizedSearch = normalizeForSearch(highlightText);

    // Try exact substring match first
    let matchIdx = normalizedFull.indexOf(normalizedSearch);
    let matchLen = normalizedSearch.length;

    // If exact match fails, try progressively shorter prefixes (min 20 chars)
    if (matchIdx === -1 && normalizedSearch.length > 20) {
      let tryLen = Math.floor(normalizedSearch.length * 0.75);
      while (matchIdx === -1 && tryLen >= 20) {
        const shorter = normalizedSearch.substring(0, tryLen);
        matchIdx = normalizedFull.indexOf(shorter);
        if (matchIdx !== -1) matchLen = tryLen;
        tryLen = Math.floor(tryLen * 0.75);
      }
    }

    // If prefix matching fails, try finding a distinctive middle segment
    if (matchIdx === -1 && normalizedSearch.length > 30) {
      // Try the middle 40% of the text (avoids common sentence starts/ends)
      const start = Math.floor(normalizedSearch.length * 0.3);
      const end = Math.floor(normalizedSearch.length * 0.7);
      const middle = normalizedSearch.substring(start, end);
      if (middle.length >= 15) {
        matchIdx = normalizedFull.indexOf(middle);
        if (matchIdx !== -1) matchLen = middle.length;
      }
    }

    if (matchIdx === -1) {
      setBboxRect(null);
      setHighlightStatus('not-found');
      return;
    }

    // Map match position back to DOM elements
    // We need to find which fragments overlap with [matchIdx, matchIdx+matchLen]
    // in the normalized text. Since normalization changes lengths, we work with
    // the unnormalized fullText and use a proportional mapping.
    const normRatio = fullText.length / normalizedFull.length;
    const approxStart = Math.floor(matchIdx * normRatio);
    const approxEnd = Math.floor((matchIdx + matchLen) * normRatio);

    const wrapperRect = pageWrapperRef.current!.getBoundingClientRect();
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    let found = false;

    for (const frag of fragments) {
      const fragStart = frag.start;
      const fragEnd = frag.start + frag.text.length;

      // Check overlap with matched region
      if (fragEnd > approxStart && fragStart < approxEnd) {
        const r = frag.el.getBoundingClientRect();
        if (r.width > 0 && r.height > 0) {
          minX = Math.min(minX, r.left - wrapperRect.left);
          minY = Math.min(minY, r.top - wrapperRect.top);
          maxX = Math.max(maxX, r.right - wrapperRect.left);
          maxY = Math.max(maxY, r.bottom - wrapperRect.top);
          found = true;
        }
      }
    }

    if (!found) {
      setBboxRect(null);
      setHighlightStatus('not-found');
      return;
    }

    // Sanity check: if bbox is unreasonably large (>60% of page), treat as not found
    const pageHeight = pageWrapperRef.current!.scrollHeight || 1;
    const bboxHeight = maxY - minY;
    if (bboxHeight > pageHeight * 0.6) {
      setBboxRect(null);
      setHighlightStatus('not-found');
      return;
    }

    const pad = 6;
    setBboxRect({
      left: minX - pad,
      top: minY - pad,
      width: (maxX - minX) + 2 * pad,
      height: (maxY - minY) + 2 * pad,
    });
    setHighlightStatus('found');

    // Scroll to the highlight
    containerRef.current?.scrollTo({
      top: Math.max(0, minY - 100),
      behavior: 'smooth',
    });
  }, [highlightText]);

  // ─── User search: inline DOM <mark> injection ──────────────────────
  // Instead of floating overlay boxes, this directly wraps matched text
  // inside the PDF text layer with <mark> elements — just like browser
  // Ctrl+F. Works with react-pdf's absolutely-positioned text spans.

  /** Remove all search <mark> elements and restore original text nodes. */
  const clearSearchHighlights = useCallback(() => {
    if (!pageWrapperRef.current) return;
    const marks = pageWrapperRef.current.querySelectorAll('mark[data-pdf-search]');
    for (const mark of Array.from(marks)) {
      const parent = mark.parentNode;
      if (!parent) continue;
      parent.replaceChild(document.createTextNode(mark.textContent || ''), mark);
      parent.normalize(); // merge adjacent text nodes back together
    }
    searchMarksRef.current = [];
    setSearchMatchCount(0);
    setCurrentSearchIdx(-1);
  }, []);

  /** Style the current match brighter, all others dimmer. */
  const styleSearchMarks = useCallback((activeIdx: number) => {
    const marks = searchMarksRef.current;
    for (let i = 0; i < marks.length; i++) {
      if (i === activeIdx) {
        marks[i].style.backgroundColor = 'rgba(59, 130, 246, 0.55)';
        marks[i].style.outline = '2px solid rgba(59, 130, 246, 0.9)';
        marks[i].style.outlineOffset = '1px';
      } else {
        marks[i].style.backgroundColor = 'rgba(59, 130, 246, 0.3)';
        marks[i].style.outline = 'none';
      }
    }
  }, []);

  /** Walk text layer, find all query matches, wrap in <mark>. */
  const executeSearch = useCallback(() => {
    clearSearchHighlights();

    if (!searchQuery.trim() || !pageWrapperRef.current) return;

    const textLayer = pageWrapperRef.current.querySelector(
      '.react-pdf__Page__textContent',
    );
    if (!textLayer) return;

    const query = searchQuery.trim().toLowerCase();
    const allMarks: HTMLElement[] = [];

    // Collect all text nodes in the text layer (handles both plain and
    // customTextRenderer spans that may already contain <mark> children)
    const textNodes: Text[] = [];
    const walker = document.createTreeWalker(textLayer, NodeFilter.SHOW_TEXT);
    let walkNode: Text | null;
    while ((walkNode = walker.nextNode() as Text | null)) {
      if (walkNode.textContent && walkNode.textContent.trim()) {
        textNodes.push(walkNode);
      }
    }

    // Process each text node: split at matches, wrap in <mark>
    for (const textNode of textNodes) {
      const text = textNode.textContent || '';
      const lowerText = text.toLowerCase();

      // Check if this text node has any match at all (fast path)
      if (lowerText.indexOf(query) === -1) continue;

      const frag = document.createDocumentFragment();
      let lastIdx = 0;
      let idx = lowerText.indexOf(query);

      while (idx !== -1) {
        // Text before match
        if (idx > lastIdx) {
          frag.appendChild(document.createTextNode(text.slice(lastIdx, idx)));
        }

        // Wrap matched text in a <mark>
        const mark = document.createElement('mark');
        mark.setAttribute('data-pdf-search', 'true');
        mark.style.backgroundColor = 'rgba(59, 130, 246, 0.3)';
        mark.style.borderRadius = '2px';
        mark.style.color = 'inherit';
        mark.style.padding = '0';
        mark.textContent = text.slice(idx, idx + query.length);
        frag.appendChild(mark);
        allMarks.push(mark);

        lastIdx = idx + query.length;
        idx = lowerText.indexOf(query, lastIdx);
      }

      // Remaining text after last match
      if (lastIdx < text.length) {
        frag.appendChild(document.createTextNode(text.slice(lastIdx)));
      }

      // Replace the original text node with our fragment
      textNode.parentNode?.replaceChild(frag, textNode);
    }

    searchMarksRef.current = allMarks;
    setSearchMatchCount(allMarks.length);

    if (allMarks.length > 0) {
      setCurrentSearchIdx(0);
      styleSearchMarks(0);
      allMarks[0].scrollIntoView({ behavior: 'smooth', block: 'center' });
    } else {
      setCurrentSearchIdx(-1);
    }
  }, [searchQuery, clearSearchHighlights, styleSearchMarks]);

  // Auto-search as user types (debounced 150ms to avoid thrashing on fast typing)
  useEffect(() => {
    if (!searchOpen) return;
    if (!searchQuery.trim()) {
      clearSearchHighlights();
      return;
    }
    const timer = setTimeout(() => executeSearch(), 150);
    return () => clearTimeout(timer);
  }, [searchQuery, searchOpen, executeSearch, clearSearchHighlights]);

  // Re-run search when page changes (text layer re-renders wipe DOM marks)
  const searchPendingRef = useRef(false);
  useEffect(() => {
    if (searchQuery.trim()) searchPendingRef.current = true;
    searchMarksRef.current = [];
    setSearchMatchCount(0);
    setCurrentSearchIdx(-1);
  }, [effectivePage]);

  // Scroll + style when navigating prev/next
  useEffect(() => {
    const marks = searchMarksRef.current;
    if (currentSearchIdx >= 0 && currentSearchIdx < marks.length) {
      styleSearchMarks(currentSearchIdx);
      marks[currentSearchIdx].scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [currentSearchIdx, styleSearchMarks]);

  // Clear search when toggling off
  useEffect(() => {
    if (!searchOpen) {
      setSearchQuery('');
      clearSearchHighlights();
    } else {
      setTimeout(() => searchInputRef.current?.focus(), 50);
    }
  }, [searchOpen, clearSearchHighlights]);

  // After text layer renders: choose strategy based on mode
  const handleTextLayerSuccess = useCallback(() => {
    if (pdfBbox) return;
    if (!containerRef.current) return;

    requestAnimationFrame(() => {
      if (useBbox) {
        // Strategy 2: DOM-based full-text search
        domFullTextSearch();
      } else {
        // Strategy 3: Inline mark mode — scroll to first visible <mark>
        const target = containerRef.current?.querySelector('mark');
        if (target) {
          setHighlightStatus('found');
          target.scrollIntoView({ behavior: 'smooth', block: 'center' });
        } else if (highlightPattern) {
          setHighlightStatus('not-found');
        }
      }

      // Re-run user search if pending after page change
      if (searchPendingRef.current) {
        searchPendingRef.current = false;
        executeSearch();
      }
    });
  }, [highlightPattern, useBbox, pdfBbox, domFullTextSearch, executeSearch]);

  // Determine whether to use customTextRenderer:
  // - useBbox mode: NO (we use DOM search instead, need clean text layer)
  // - pdfBbox mode: NO (stored bbox, no text search needed)
  // - inline mode: YES (keyword-based marks)
  const shouldUseTextRenderer = !useBbox && !pdfBbox && !!highlightPattern;

  // ─── Render ────────────────────────────────────────────────────────
  return (
    <div ref={containerRef} className="h-full overflow-auto bg-gray-100">
      <Document
        file={pdfUrl}
        onLoadSuccess={(pdf) => setTotalPages(pdf.numPages)}
        loading={
          <div className="flex items-center justify-center h-64">
            <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
            <span className="ml-2 text-sm text-gray-400">Loading PDF...</span>
          </div>
        }
        error={
          <div className="flex items-center justify-center h-64 text-red-500 text-sm">
            Failed to load PDF. Check backend connection.
          </div>
        }
      >
        {/* ── Page navigation bar ─────────────────────────────── */}
        {totalPages > 0 && (
          <div className="sticky top-0 z-20 flex items-center justify-center gap-2 py-1.5 px-3 bg-white/90 backdrop-blur border-b border-gray-200">
            <button
              onClick={() => setManualPage(Math.max(1, (effectivePage || 1) - 1))}
              disabled={!effectivePage || effectivePage <= 1}
              className="p-0.5 rounded hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4 text-gray-600" />
            </button>
            <input
              type="number"
              min={1}
              max={totalPages}
              value={effectivePage || ''}
              placeholder="—"
              onChange={(e) => {
                const n = parseInt(e.target.value, 10);
                if (n >= 1 && n <= totalPages) setManualPage(n);
              }}
              className="w-12 text-center text-xs border border-gray-200 rounded px-1 py-0.5 focus:outline-none focus:ring-1 focus:ring-blue-300 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
            />
            <span className="text-[10px] text-gray-400">/ {totalPages}</span>
            <button
              onClick={() => setManualPage(Math.min(totalPages, (effectivePage || 0) + 1))}
              disabled={!effectivePage || effectivePage >= totalPages}
              className="p-0.5 rounded hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronRight className="h-4 w-4 text-gray-600" />
            </button>
            {manualPage && page && manualPage !== page && (
              <button
                onClick={() => setManualPage(null)}
                className="text-[10px] text-blue-600 hover:text-blue-800 ml-1"
              >
                Reset to p.{page}
              </button>
            )}

            {/* Divider */}
            <div className="w-px h-4 bg-gray-200 mx-1" />

            {/* Search toggle / inline search bar */}
            {!searchOpen ? (
              <button
                onClick={() => setSearchOpen(true)}
                className="p-0.5 rounded hover:bg-gray-100"
                title="Search in PDF (Ctrl+F)"
              >
                <Search className="h-4 w-4 text-gray-500" />
              </button>
            ) : (
              <div className="flex items-center gap-1">
                <div className="relative">
                  <input
                    ref={searchInputRef}
                    type="text"
                    value={searchQuery}
                    placeholder="Search in PDF..."
                    onChange={(e) => setSearchQuery(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && searchMatchCount > 0) {
                        e.preventDefault();
                        if (e.shiftKey) {
                          setCurrentSearchIdx((i) => (i - 1 + searchMatchCount) % searchMatchCount);
                        } else {
                          setCurrentSearchIdx((i) => (i + 1) % searchMatchCount);
                        }
                      }
                      if (e.key === 'Escape') {
                        setSearchOpen(false);
                      }
                    }}
                    className="w-36 text-xs border border-gray-300 rounded pl-1.5 pr-6 py-0.5 focus:outline-none focus:ring-1 focus:ring-blue-400 focus:border-blue-400"
                  />
                  {searchQuery && (
                    <span className="absolute right-1.5 top-1/2 -translate-y-1/2 text-[9px] text-gray-400 pointer-events-none">
                      {searchMatchCount > 0
                        ? `${currentSearchIdx + 1}/${searchMatchCount}`
                        : '0/0'}
                    </span>
                  )}
                </div>
                <button
                  onClick={() => setCurrentSearchIdx((i) => (i - 1 + searchMatchCount) % searchMatchCount)}
                  disabled={searchMatchCount === 0}
                  className="p-0.5 rounded hover:bg-gray-100 disabled:opacity-30"
                  title="Previous match (Shift+Enter)"
                >
                  <ArrowUp className="h-3.5 w-3.5 text-gray-600" />
                </button>
                <button
                  onClick={() => setCurrentSearchIdx((i) => (i + 1) % searchMatchCount)}
                  disabled={searchMatchCount === 0}
                  className="p-0.5 rounded hover:bg-gray-100 disabled:opacity-30"
                  title="Next match (Enter)"
                >
                  <ArrowDown className="h-3.5 w-3.5 text-gray-600" />
                </button>
                <button
                  onClick={() => setSearchOpen(false)}
                  className="p-0.5 rounded hover:bg-gray-100"
                  title="Close search (Esc)"
                >
                  <X className="h-3.5 w-3.5 text-gray-500" />
                </button>
              </div>
            )}
          </div>
        )}

        {/* ── No page number state ────────────────────────────── */}
        {!effectivePage && containerWidth > 0 && (
          <div className="flex flex-col items-center justify-center py-16 px-8 text-center">
            <AlertTriangle className="h-8 w-8 text-amber-400 mb-3" />
            <div className="text-sm font-semibold text-gray-700 mb-1">Page not mapped</div>
            <p className="text-xs text-gray-500 mb-4 max-w-[280px]">
              This extraction does not have a page number. Use the page controls above to manually browse the PDF.
            </p>
            <button
              onClick={() => setManualPage(1)}
              className="text-xs text-blue-600 hover:text-blue-800 underline"
            >
              Browse from page 1
            </button>
          </div>
        )}

        {effectivePage && containerWidth > 0 && (
          <div ref={pageWrapperRef} style={{ position: 'relative' }}>
            <Page
              pageNumber={effectivePage}
              width={containerWidth}
              renderTextLayer={true}
              renderAnnotationLayer={false}
              customTextRenderer={shouldUseTextRenderer ? textRenderer : undefined}
              onLoadSuccess={handlePageLoadSuccess}
              onRenderTextLayerSuccess={handleTextLayerSuccess}
              loading={
                <div className="flex items-center justify-center h-64">
                  <Loader2 className="h-6 w-6 animate-spin text-gray-300" />
                </div>
              }
            />
            {/* Amber bbox overlay — from stored pipeline coords or DOM text search */}
            {bboxRect && !highlightSpans && (
              <div
                data-bbox-overlay
                style={{
                  position: 'absolute',
                  left: bboxRect.left,
                  top: bboxRect.top,
                  width: bboxRect.width,
                  height: bboxRect.height,
                  backgroundColor: 'rgba(251, 191, 36, 0.25)',
                  border: '2px solid rgba(251, 191, 36, 0.8)',
                  borderRadius: '4px',
                  pointerEvents: 'none',
                  zIndex: 10,
                  boxShadow: '0 0 8px rgba(251, 191, 36, 0.4)',
                }}
              />
            )}
            {/* Multi-channel bbox overlays — each span in its primary channel color */}
            {highlightSpans && pageOriginalWidth > 0 && containerWidth > 0 && (() => {
              const scale = containerWidth / pageOriginalWidth;
              const pad = 4;
              return highlightSpans
                .filter((s) => s.bbox)
                .map((s) => {
                  const [x0, y0, x1, y1] = s.bbox!;
                  const primaryChannel = s.channels[0] || 'B';
                  const colors = CHANNEL_HIGHLIGHT_COLORS[primaryChannel] || CHANNEL_HIGHLIGHT_COLORS.B;
                  const isSelected = s.isSelected;
                  return (
                    <div
                      key={s.id}
                      data-span-overlay={s.id}
                      style={{
                        position: 'absolute',
                        left: x0 * scale - pad,
                        top: y0 * scale - pad,
                        width: (x1 - x0) * scale + 2 * pad,
                        height: (y1 - y0) * scale + 2 * pad,
                        backgroundColor: isSelected
                          ? colors.bg.replace('0.15', '0.3')
                          : colors.bg,
                        border: `${isSelected ? '3px' : '2px'} solid ${colors.border}`,
                        borderRadius: '3px',
                        pointerEvents: 'none',
                        zIndex: isSelected ? 12 : 10,
                        boxShadow: isSelected
                          ? `0 0 12px ${colors.border}`
                          : `0 0 4px ${colors.bg}`,
                        transition: 'all 150ms ease',
                      }}
                    />
                  );
                });
            })()}
          </div>
        )}

        {/* Close the effectivePage conditional block */}
        </Document>

      {/* ── Not-found fallback: text not located on this page ──────────── */}
      {highlightStatus === 'not-found' && highlightText && (
        <div className="mx-4 my-3 p-4 bg-amber-50 border border-amber-300 rounded-lg">
          <div className="flex items-start gap-2.5">
            <Search className="h-5 w-5 text-amber-600 mt-0.5 shrink-0" />
            <div className="flex-1 min-w-0">
              <div className="text-sm font-semibold text-amber-900 mb-1">
                Text Not Found on Page {page ?? '—'}
              </div>
              <p className="text-xs text-amber-800 leading-relaxed mb-3">
                The extracted text could not be located in the PDF text layer on this page.
                This may happen when the text is inside a figure or table (vector-drawn content),
                or when the pipeline assigned a page number from a summary section.
              </p>
              <div className="text-xs text-amber-900 font-medium mb-2">
                Manual verification steps:
              </div>
              <ul className="text-xs text-amber-800 space-y-1 mb-3 list-none pl-0">
                <li className="flex gap-1.5">
                  <span className="shrink-0">1.</span>
                  <span>Scan page {page ?? '—'} above for figures, tables, or summary boxes</span>
                </li>
                <li className="flex gap-1.5">
                  <span className="shrink-0">2.</span>
                  <span>If not visible, use the page controls to browse nearby pages</span>
                </li>
                <li className="flex gap-1.5">
                  <span className="shrink-0">3.</span>
                  <span>Compare character-by-character against the extracted text below</span>
                </li>
              </ul>
              <div className="bg-white/60 border border-amber-200 rounded p-3 text-xs text-gray-800 font-mono leading-relaxed break-words">
                {'\u201C'}{highlightText}{'\u201D'}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
