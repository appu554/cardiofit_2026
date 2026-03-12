'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import type { SectionPassage } from '@/types/pipeline1';

interface SectionPassageViewProps {
  jobId: string;
  pageNumber: number | null;
}

// ---------------------------------------------------------------------------
// JSON field rendering helpers
// ---------------------------------------------------------------------------

function JsonKey({ name }: { name: string }) {
  return <span className="text-blue-400">&quot;{name}&quot;</span>;
}

function JsonString({ value }: { value: string }) {
  return <span className="text-green-400">&quot;{value}&quot;</span>;
}

function JsonNumber({ value }: { value: number }) {
  return <span className="text-amber-400">{value}</span>;
}

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max) + '...';
}

// ---------------------------------------------------------------------------
// Single passage card
// ---------------------------------------------------------------------------

function PassageBlock({ passage }: { passage: SectionPassage }) {
  const prose = passage.proseText ? truncate(passage.proseText, 200) : null;

  return (
    <div className="bg-gray-900 text-gray-100 rounded-lg p-4 font-mono text-xs leading-relaxed">
      <div>{'{'}</div>

      {/* section_id */}
      <div className="ml-4">
        <JsonKey name="section_id" />
        <span className="text-gray-400">: </span>
        <JsonString value={passage.sectionId} />
        <span className="text-gray-400">,</span>
      </div>

      {/* heading */}
      <div className="ml-4">
        <JsonKey name="heading" />
        <span className="text-gray-400">: </span>
        <JsonString value={passage.heading} />
        <span className="text-gray-400">,</span>
      </div>

      {/* span_count */}
      <div className="ml-4">
        <JsonKey name="span_count" />
        <span className="text-gray-400">: </span>
        <JsonNumber value={passage.spanCount} />
        <span className="text-gray-400">,</span>
      </div>

      {/* prose_text */}
      <div className="ml-4">
        <JsonKey name="prose_text" />
        <span className="text-gray-400">: </span>
        {prose !== null ? (
          <JsonString value={prose} />
        ) : (
          <span className="text-gray-500 italic">null</span>
        )}
      </div>

      <div>{'}'}</div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Loading skeleton
// ---------------------------------------------------------------------------

function PassageSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      {[1, 2, 3].map((i) => (
        <div key={i} className="bg-gray-900/60 rounded-lg p-4 space-y-2">
          <div className="h-3 bg-gray-700 rounded w-3/4" />
          <div className="h-3 bg-gray-700 rounded w-1/2" />
          <div className="h-3 bg-gray-700 rounded w-1/3" />
          <div className="h-3 bg-gray-700 rounded w-5/6" />
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function SectionPassageView({ jobId, pageNumber }: SectionPassageViewProps) {
  const { data: passages, isLoading } = useQuery<SectionPassage[]>({
    queryKey: ['pipeline1-passages', jobId],
    queryFn: () => pipeline1Api.context.getPassages(jobId),
  });

  // Filter passages to those matching the selected page
  const filteredPassages = useMemo(() => {
    if (!passages) return [];
    if (pageNumber === null) return [];
    return passages.filter((p) => p.pageNumber === pageNumber);
  }, [passages, pageNumber]);

  // Total span count across filtered passages
  const totalSpanCount = useMemo(
    () => filteredPassages.reduce((sum, p) => sum + p.spanCount, 0),
    [filteredPassages]
  );

  // -- No page selected --
  if (pageNumber === null) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400 text-sm">
        Select a page to view section passages
      </div>
    );
  }

  // -- Loading state --
  if (isLoading) {
    return (
      <div className="p-6">
        <PassageSkeleton />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      {/* Passage blocks */}
      {filteredPassages.map((passage) => (
        <PassageBlock key={passage.sectionId} passage={passage} />
      ))}

      {/* L3 Readiness card */}
      {filteredPassages.length > 0 ? (
        <div className="flex items-center gap-2 rounded-lg border border-green-700 bg-green-950 px-4 py-3 text-sm text-green-300">
          <CheckCircle className="h-5 w-5 flex-shrink-0 text-green-400" aria-hidden="true" />
          <span>
            Ready &mdash; {totalSpanCount} span{totalSpanCount !== 1 ? 's' : ''} across{' '}
            {filteredPassages.length} section{filteredPassages.length !== 1 ? 's' : ''}
          </span>
        </div>
      ) : (
        <div className="flex items-center justify-center rounded-lg border border-gray-700 bg-gray-800 px-4 py-3 text-sm text-gray-400">
          No sections for this page
        </div>
      )}
    </div>
  );
}
