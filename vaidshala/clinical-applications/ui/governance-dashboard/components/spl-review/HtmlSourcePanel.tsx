'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Loader2,
  FileCode,
  ZoomIn,
  ZoomOut,
  Search,
  ChevronDown,
  ExternalLink,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { splReviewApi } from '@/lib/spl-api';
import { EXPECTED_SECTIONS } from '@/types/spl-review';

// ============================================================================
// HTML Source Panel — Renders SPL section HTML with term highlighting
//
// Unlike the guideline review (PDF-based), SPL labels are XML/HTML.
// This panel fetches the raw section HTML from the backend, sanitizes it,
// and renders it with the active fact's source phrase highlighted.
//
// Key differences from PdfHighlightViewer:
// - No PDF.js — uses dangerouslySetInnerHTML with DOMPurify-style sanitization
// - Highlights by text content matching, not bounding box
// - Supports LOINC section switching (Warnings, Drug Interactions, etc.)
// ============================================================================

interface HtmlSourcePanelProps {
  /** Source document ID (from derived_facts) */
  sourceDocumentId: string;
  /** LOINC section code to display (e.g., '34073-7') */
  sectionCode: string;
  /** Text to highlight in the rendered HTML */
  highlightText?: string;
  /** List of available LOINC sections for this drug */
  availableSections?: string[];
  /** Callback when section changes */
  onSectionChange?: (sectionCode: string) => void;
}

export function HtmlSourcePanel({
  sourceDocumentId,
  sectionCode,
  highlightText,
  availableSections,
  onSectionChange,
}: HtmlSourcePanelProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [zoom, setZoom] = useState(100);
  const [showSectionPicker, setShowSectionPicker] = useState(false);

  // Fetch section HTML from backend
  const { data: html, isLoading, error } = useQuery({
    queryKey: ['spl-section-html', sourceDocumentId, sectionCode],
    queryFn: () => splReviewApi.facts.getSectionHtml(sourceDocumentId, sectionCode),
    enabled: !!sourceDocumentId && !!sectionCode,
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  });

  // Sanitize and highlight the HTML
  const processedHtml = useMemo(() => {
    if (!html) return '';

    // Basic sanitization: strip script tags and event handlers
    let sanitized = html
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/\son\w+="[^"]*"/gi, '')
      .replace(/\son\w+='[^']*'/gi, '')
      .replace(/javascript:/gi, '');

    // Highlight the target text
    if (highlightText && highlightText.length > 3) {
      const escaped = highlightText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const regex = new RegExp(`(${escaped})`, 'gi');
      sanitized = sanitized.replace(
        regex,
        '<mark class="spl-highlight" style="background: #FDE68A; padding: 2px 4px; border-radius: 3px; border-bottom: 2px solid #F59E0B;">$1</mark>'
      );
    }

    return sanitized;
  }, [html, highlightText]);

  // Scroll to highlighted text when it changes
  useEffect(() => {
    if (!containerRef.current || !highlightText) return;

    const timer = setTimeout(() => {
      const mark = containerRef.current?.querySelector('.spl-highlight');
      if (mark) {
        mark.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }, 100);

    return () => clearTimeout(timer);
  }, [processedHtml, highlightText]);

  // Section name lookup
  const sectionName = EXPECTED_SECTIONS[sectionCode] || sectionCode;

  // Zoom controls
  const handleZoom = useCallback((delta: number) => {
    setZoom((z) => Math.max(60, Math.min(200, z + delta)));
  }, []);

  return (
    <div className="flex flex-col h-full bg-white">
      {/* ── Header Bar ─────────────────────────────────────────────── */}
      <div className="px-4 py-2.5 border-b border-gray-200 flex items-center justify-between shrink-0 bg-gray-50">
        <div className="flex items-center gap-3">
          <FileCode className="h-4 w-4 text-gray-500" />
          <div className="relative">
            <button
              onClick={() => setShowSectionPicker(!showSectionPicker)}
              className="flex items-center gap-1.5 text-xs font-semibold text-gray-900 hover:text-blue-700 transition-colors"
            >
              {sectionName}
              <span className="text-gray-400 font-mono text-[10px]">({sectionCode})</span>
              {availableSections && availableSections.length > 1 && (
                <ChevronDown className="h-3 w-3 text-gray-400" />
              )}
            </button>

            {/* Section picker dropdown */}
            {showSectionPicker && availableSections && availableSections.length > 1 && (
              <div className="absolute top-full left-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg z-20 min-w-[240px]">
                {availableSections.map((code) => (
                  <button
                    key={code}
                    onClick={() => {
                      onSectionChange?.(code);
                      setShowSectionPicker(false);
                    }}
                    className={cn(
                      'w-full text-left px-3 py-2 text-xs hover:bg-blue-50 transition-colors flex items-center justify-between',
                      code === sectionCode
                        ? 'bg-blue-50 text-blue-700 font-medium'
                        : 'text-gray-700'
                    )}
                  >
                    <span>{EXPECTED_SECTIONS[code] || code}</span>
                    <span className="text-gray-400 font-mono">{code}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Zoom controls */}
        <div className="flex items-center gap-2">
          {highlightText && (
            <span className="text-[10px] text-amber-600 bg-amber-50 px-2 py-0.5 rounded flex items-center gap-1">
              <Search className="h-3 w-3" />
              Highlighting match
            </span>
          )}
          <div className="flex items-center gap-1 bg-gray-100 rounded-md px-1.5 py-0.5">
            <button
              onClick={() => handleZoom(-10)}
              className="p-0.5 text-gray-500 hover:text-gray-700 transition-colors"
              title="Zoom out"
            >
              <ZoomOut className="h-3.5 w-3.5" />
            </button>
            <span className="text-[10px] text-gray-600 font-medium w-8 text-center">
              {zoom}%
            </span>
            <button
              onClick={() => handleZoom(10)}
              className="p-0.5 text-gray-500 hover:text-gray-700 transition-colors"
              title="Zoom in"
            >
              <ZoomIn className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </div>

      {/* ── HTML Content ───────────────────────────────────────────── */}
      <div
        ref={containerRef}
        className="flex-1 overflow-auto p-6"
        style={{ fontSize: `${zoom}%` }}
      >
        {isLoading && (
          <div className="flex items-center justify-center h-full">
            <div className="flex flex-col items-center gap-3">
              <Loader2 className="h-6 w-6 text-blue-500 animate-spin" />
              <p className="text-xs text-gray-500">Loading section HTML...</p>
            </div>
          </div>
        )}

        {error && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <p className="text-red-600 text-sm font-medium">Failed to load section HTML</p>
              <p className="text-gray-500 text-xs mt-1">
                Ensure the SPL source documents are available in the database
              </p>
            </div>
          </div>
        )}

        {!isLoading && !error && !html && (
          <div className="flex items-center justify-center h-full text-gray-400 text-sm">
            No HTML content available for this section
          </div>
        )}

        {processedHtml && (
          <div
            className="spl-source-content prose prose-sm max-w-none
              prose-table:border-collapse prose-table:border prose-table:border-gray-300
              prose-th:border prose-th:border-gray-300 prose-th:bg-gray-50 prose-th:p-2 prose-th:text-xs prose-th:font-semibold
              prose-td:border prose-td:border-gray-300 prose-td:p-2 prose-td:text-xs
              prose-p:text-sm prose-p:text-gray-700 prose-p:leading-relaxed
              prose-li:text-sm prose-li:text-gray-700
              prose-h1:text-base prose-h1:font-bold prose-h1:text-gray-900
              prose-h2:text-sm prose-h2:font-bold prose-h2:text-gray-900
              prose-h3:text-sm prose-h3:font-semibold prose-h3:text-gray-800"
            dangerouslySetInnerHTML={{ __html: processedHtml }}
          />
        )}
      </div>
    </div>
  );
}
