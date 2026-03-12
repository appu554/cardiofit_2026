'use client';

import { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  Loader2,
  ChevronDown,
  ChevronRight,
  ChevronsUpDown,
  FileText,
} from 'lucide-react';
import Link from 'next/link';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { PdfViewer } from '@/components/pipeline1/PdfViewer';
import { cn } from '@/lib/utils';
import type { SectionPassage } from '@/types/pipeline1';

export default function ComparisonReviewPage() {
  const params = useParams();
  const jobId = params.jobId as string;

  const [activePdfPage, setActivePdfPage] = useState<number | undefined>();
  const [activeLogicalPage, setActiveLogicalPage] = useState<number | undefined>();
  const [expandedPages, setExpandedPages] = useState<Set<number> | 'all'>('all');
  const scrollRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();
  const isManualNavRef = useRef(false);

  // Fetch job info
  const { data: job } = useQuery({
    queryKey: ['pipeline1-job', jobId],
    queryFn: () => pipeline1Api.jobs.get(jobId),
  });

  // Fetch passages
  const { data: passages, isLoading: passagesLoading } = useQuery({
    queryKey: ['pipeline1-passages', jobId],
    queryFn: () => pipeline1Api.context.getPassages(jobId),
    enabled: !!jobId,
  });

  // Group passages by page number
  const pageGroups = useMemo(() => {
    if (!passages) return [];
    const groups = new Map<number, SectionPassage[]>();
    for (const p of passages) {
      const page = p.pageNumber ?? 0;
      if (!groups.has(page)) groups.set(page, []);
      groups.get(page)!.push(p);
    }
    return Array.from(groups.entries())
      .sort(([a], [b]) => a - b)
      .map(([page, passages]) => ({ page, passages }));
  }, [passages]);

  const isPageExpanded = (page: number) =>
    expandedPages === 'all' || expandedPages.has(page);

  const togglePage = (page: number) => {
    setExpandedPages((prev) => {
      if (prev === 'all') {
        const next = new Set(pageGroups.map((g) => g.page));
        next.delete(page);
        return next;
      }
      const next = new Set(prev);
      if (next.has(page)) next.delete(page);
      else next.add(page);
      return next;
    });
  };

  const toggleAll = () => {
    if (expandedPages === 'all') {
      setExpandedPages(new Set());
    } else {
      setExpandedPages('all');
    }
  };

  /** Navigate the right-panel PDF to a specific page.
   *  Pipeline page numbers are physical PDF pages (1-indexed) — no offset needed. */
  const goToPage = useCallback((page: number) => {
    // Suppress observer-triggered nav for 500ms after a manual click
    isManualNavRef.current = true;
    clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => { isManualNavRef.current = false; }, 500);
    setActivePdfPage(page);
    setActiveLogicalPage(page);
  }, []);

  // Scroll-sync: observe which page group header is visible in the left panel
  useEffect(() => {
    const container = scrollRef.current;
    if (!container || pageGroups.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (isManualNavRef.current) return; // skip during manual click nav
        for (const entry of entries) {
          if (entry.isIntersecting) {
            const page = Number(entry.target.getAttribute('data-page'));
            if (!isNaN(page) && page !== activeLogicalPage) {
              clearTimeout(debounceRef.current);
              debounceRef.current = setTimeout(() => {
                setActivePdfPage(page);
                setActiveLogicalPage(page);
              }, 300);
            }
          }
        }
      },
      { root: container, threshold: 0.5 }
    );

    const headers = container.querySelectorAll('[data-page]');
    headers.forEach((el) => observer.observe(el));

    return () => observer.disconnect();
  }, [pageGroups, activeLogicalPage]);

  const totalSpans = useMemo(() => {
    if (!passages) return 0;
    return passages.reduce((sum, p) => sum + p.spanCount, 0);
  }, [passages]);

  if (passagesLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  return (
    <div className="-m-6 flex flex-col h-[calc(100vh-64px)] overflow-hidden">
      {/* Top bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200 bg-white shrink-0">
        <Link
          href="/pipeline1/compare"
          className="flex items-center text-gray-600 hover:text-gray-900 text-sm"
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back
        </Link>
        {job && (
          <div className="text-xs text-gray-500">
            <span className="font-medium text-gray-700">
              {job.sourcePdf.replace(/^.*\//, '')}
            </span>
            {' · '}
            {passages?.length ?? 0} passages · {totalSpans} spans
          </div>
        )}
      </div>

      {/* Two-panel split */}
      <div className="flex-1 flex min-h-0 overflow-hidden">
        {/* ─── LEFT: Extracted Data (Pipeline Output) ─── */}
        <div className="w-[45%] border-r border-gray-200 flex flex-col min-h-0 bg-white">
          <div className="px-4 py-2.5 bg-gray-50 border-b border-gray-200 shrink-0 flex items-center justify-between">
            <div>
              <h2 className="text-sm font-semibold text-gray-700">Extracted Data</h2>
              <p className="text-[11px] text-gray-400 mt-0.5">
                {pageGroups.length} pages · {passages?.length ?? 0} passages
              </p>
            </div>
            <button
              onClick={toggleAll}
              className="flex items-center gap-1 px-2 py-1 text-[10px] font-medium text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded transition-colors"
              title={expandedPages === 'all' ? 'Collapse all' : 'Expand all'}
            >
              <ChevronsUpDown className="h-3 w-3" />
              {expandedPages === 'all' ? 'Collapse' : 'Expand'}
            </button>
          </div>

          <div ref={scrollRef} className="flex-1 overflow-y-auto">
            {pageGroups.map(({ page, passages: pagePassages }) => (
              <div key={page} className="border-b border-gray-100">
                {/* Page header — click chevron to expand/collapse, click page label to navigate PDF */}
                <div data-page={page} className="flex items-center gap-0 hover:bg-gray-50 transition-colors">
                  <button
                    onClick={() => togglePage(page)}
                    className="flex items-center pl-4 py-2.5 pr-1"
                    title={isPageExpanded(page) ? 'Collapse' : 'Expand'}
                  >
                    {isPageExpanded(page) ? (
                      <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5 text-gray-400" />
                    )}
                  </button>
                  <button
                    onClick={() => goToPage(page)}
                    className={cn(
                      'flex-1 flex items-center gap-2 py-2.5 pr-4 text-left',
                      activeLogicalPage === page && 'bg-blue-50'
                    )}
                    title={`View page ${page} in source PDF`}
                  >
                    <FileText className={cn(
                      'h-3.5 w-3.5 shrink-0',
                      activeLogicalPage === page ? 'text-blue-500' : 'text-gray-300'
                    )} />
                    <span className={cn(
                      'text-xs font-semibold',
                      activeLogicalPage === page ? 'text-blue-700' : 'text-gray-600'
                    )}>
                      Page {page}
                    </span>
                    <span className="text-[10px] text-gray-400 ml-auto">
                      {pagePassages.length} section{pagePassages.length !== 1 ? 's' : ''}
                      {' · '}
                      {pagePassages.reduce((sum, p) => sum + p.spanCount, 0)} spans
                    </span>
                  </button>
                </div>

                {/* Passages under this page */}
                {isPageExpanded(page) && (
                  <div className="pb-2 px-3 space-y-2">
                    {pagePassages.map((passage) => (
                      <button
                        key={passage.sectionId}
                        onClick={() => goToPage(page)}
                        className={cn(
                          'w-full text-left rounded-md border overflow-hidden transition-colors',
                          activeLogicalPage === page
                            ? 'border-blue-200 bg-blue-50/30'
                            : 'border-gray-200 bg-white hover:border-gray-300'
                        )}
                      >
                        {/* Passage header */}
                        <div className="px-3 py-2 bg-gray-50 border-b border-gray-100">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-medium text-gray-800 truncate">
                              {passage.heading}
                            </span>
                            <span className="text-[10px] text-gray-400 shrink-0 ml-2">
                              {passage.spanCount} spans
                            </span>
                          </div>
                          <span className="text-[10px] text-gray-400 font-mono">
                            {passage.sectionId}
                          </span>
                        </div>

                        {/* Prose text */}
                        {passage.proseText && (
                          <div className="px-3 py-2 text-xs text-gray-700 leading-relaxed max-h-48 overflow-y-auto whitespace-pre-wrap">
                            {passage.proseText}
                          </div>
                        )}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}

            {pageGroups.length === 0 && (
              <div className="flex items-center justify-center h-32 text-gray-400 text-sm">
                No passages found for this job
              </div>
            )}
          </div>
        </div>

        {/* ─── RIGHT: Original Source PDF ─── */}
        <div className="flex-1 flex flex-col">
          <div className="px-4 py-2.5 bg-gray-50 border-b border-gray-200 shrink-0 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-700">Original Source PDF</h2>
            {activeLogicalPage && (
              <span className="text-[11px] text-gray-400">
                Viewing page {activeLogicalPage}
              </span>
            )}
          </div>
          <div className="flex-1 min-h-0">
            <PdfViewer jobId={jobId} page={activePdfPage} />
          </div>
        </div>
      </div>
    </div>
  );
}
