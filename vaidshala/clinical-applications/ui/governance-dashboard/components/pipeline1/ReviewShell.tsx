'use client';

import { ReactNode } from 'react';
import dynamic from 'next/dynamic';
import { Loader2 } from 'lucide-react';
import { PdfErrorBoundary } from './PdfErrorBoundary';

// Dynamic import: pdfjs-dist ESM module fails during SSR webpack bundling
const PdfHighlightViewer = dynamic(
  () => import('./PdfHighlightViewer').then((m) => m.PdfHighlightViewer),
  { ssr: false, loading: () => <div className="flex-1 flex items-center justify-center"><Loader2 className="h-6 w-6 animate-spin text-gray-300" /></div> },
);

// =============================================================================
// ReviewShell — Shared split-panel layout for Phase 2A, 2B, 3A, 3B
//
// Left 48%: children (phase-specific content — task cards, span lists, etc.)
// Right 52%: PdfHighlightViewer with the source PDF
//
// This prevents code duplication across the four review phases that all
// share the same "content + PDF side-by-side" layout pattern.
// =============================================================================

interface ReviewShellProps {
  jobId: string;
  /** Phase-specific header bar rendered above the left panel content */
  topBar?: ReactNode;
  /** Left panel content — task card, span inspector, etc. */
  children: ReactNode;
  /** Physical PDF page number (1-indexed) to display */
  pdfPage?: number;
  /** Text to search for and highlight on the rendered page */
  pdfHighlightText?: string;
  /** Stored bounding box from pipeline extraction [x0, y0, x1, y1] */
  pdfBbox?: [number, number, number, number];
  /** Use bbox-style overlay instead of inline marks */
  useBbox?: boolean;
}

export function ReviewShell({
  jobId,
  topBar,
  children,
  pdfPage,
  pdfHighlightText,
  pdfBbox,
  useBbox,
}: ReviewShellProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Optional top bar */}
      {topBar && (
        <div className="shrink-0 border-b border-gray-200 bg-white">
          {topBar}
        </div>
      )}

      {/* Split panel */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left panel: phase content */}
        <div className="w-[48%] border-r border-gray-200 overflow-y-auto">
          {children}
        </div>

        {/* Right panel: PDF viewer */}
        <div className="w-[52%] overflow-hidden">
          <PdfErrorBoundary>
            <PdfHighlightViewer
              jobId={jobId}
              page={pdfPage}
              highlightText={pdfHighlightText}
              pdfBbox={pdfBbox}
              useBbox={useBbox}
            />
          </PdfErrorBoundary>
        </div>
      </div>
    </div>
  );
}
