'use client';

import { useMemo } from 'react';
import { pipeline1Api } from '@/lib/pipeline1-api';

interface PdfViewerProps {
  jobId: string;
  /** Navigate to this page in the PDF. Uses the #page=N fragment. */
  page?: number;
  /** Text to search/highlight on the page. Uses the #search=TEXT fragment. */
  searchText?: string;
}

/**
 * Extracts a short, distinctive search phrase from the span text.
 * Prioritises numeric/clinical terms that are most identifiable on a page.
 * Chrome's PDF viewer uses this to yellow-highlight matching text.
 */
function extractSearchPhrase(text: string): string {
  if (!text) return '';

  // Clean: collapse whitespace, trim
  const clean = text.replace(/\s+/g, ' ').trim();

  // Take first ~50 chars, trim to a word boundary
  const cut = clean.length <= 50 ? clean : clean.slice(0, 50).replace(/\s\S*$/, '');

  // Remove chars that break the PDF viewer search
  return cut.replace(/[[\]{}()]/g, '');
}

/**
 * Renders the original source PDF in the browser's native PDF viewer via iframe.
 * Uses #page=N for page navigation and #search=TEXT for text highlighting.
 *
 * The backend serves the PDF with Cache-Control: max-age=86400, so the browser
 * caches the 8MB file. Iframe remounts (via key) cost only DOM recreation, not
 * a network roundtrip after the first load.
 */
export function PdfViewer({ jobId, page, searchText }: PdfViewerProps) {
  const src = useMemo(() => {
    const base = pipeline1Api.context.getSourcePdfUrl(jobId);
    const fragments: string[] = [];

    if (page) fragments.push(`page=${page}`);

    if (searchText) {
      const phrase = extractSearchPhrase(searchText);
      if (phrase) fragments.push(`search=${encodeURIComponent(phrase)}`);
    }

    return fragments.length > 0 ? `${base}#${fragments.join('&')}` : base;
  }, [jobId, page, searchText]);

  return (
    <iframe
      key={src}
      src={src}
      title="Source PDF"
      className="w-full h-full border-0"
    />
  );
}
