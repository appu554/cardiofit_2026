'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft } from 'lucide-react';
import Link from 'next/link';
import { governanceApi } from '@/lib/api';
import { FactContent } from '@/components/facts/FactContent';
import { EvidencePanel } from '@/components/facts/EvidencePanel';
import { ReviewActions } from '@/components/facts/ReviewActions';
import { AuditHistory } from '@/components/facts/AuditHistory';

export default function FactDetailPage() {
  const params = useParams();
  const router = useRouter();
  const factId = params.id as string;

  const { data: fact, isLoading: factLoading, error } = useQuery({
    queryKey: ['fact', factId],
    queryFn: () => governanceApi.facts.getFact(factId),
  });

  const { data: history, isLoading: historyLoading } = useQuery({
    queryKey: ['fact-history', factId],
    queryFn: () => governanceApi.facts.getHistory(factId),
  });

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64">
        <p className="text-red-600 font-medium">Failed to load fact</p>
        <p className="text-gray-500 text-sm mt-1">{(error as Error).message}</p>
        <Link href="/queue" className="mt-4 btn btn-outline">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Queue
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Back nav + Fact ID */}
      <div className="flex items-center justify-between">
        <Link
          href="/queue"
          className="flex items-center text-gray-600 hover:text-gray-900"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Queue
        </Link>
        {fact && (
          <div className="text-sm text-gray-500">
            <span className="font-medium">{fact.drugName}</span>
            {' · '}
            <span className="font-mono text-xs">{(fact.factId || fact.id)?.slice(0, 12)}…</span>
          </div>
        )}
      </div>

      {factLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-[280px_1fr_300px] gap-4 h-[calc(100vh-160px)]">
          <div className="card p-6"><div className="skeleton h-full w-full" /></div>
          <div className="card p-6"><div className="skeleton h-full w-full" /></div>
          <div className="card p-6"><div className="skeleton h-full w-full" /></div>
        </div>
      ) : fact ? (
        <>
          {/* 3-Panel Layout */}
          <div className="grid grid-cols-1 lg:grid-cols-[280px_1fr_300px] gap-4">
            {/* LEFT: The Fact (sticky) */}
            <div className="lg:sticky lg:top-6 lg:self-start">
              <FactContent fact={fact} />
            </div>

            {/* MIDDLE: Evidence Stack (scrollable) */}
            <div className="space-y-4 overflow-y-auto">
              <EvidencePanel fact={fact} />

              {/* Audit History below evidence */}
              <AuditHistory
                history={history}
                isLoading={historyLoading}
              />
            </div>

            {/* RIGHT: Decision Controls (sticky) */}
            <div className="lg:sticky lg:top-6 lg:self-start">
              {(fact.status === 'DRAFT' || fact.status === 'PENDING_REVIEW') ? (
                <ReviewActions
                  fact={fact}
                  onActionComplete={() => router.push('/queue')}
                />
              ) : (
                <div className="card">
                  <div className="card-header">
                    <h2 className="text-lg font-semibold text-gray-900">Status</h2>
                  </div>
                  <div className="card-body">
                    <p className="text-sm text-gray-600">
                      This fact has been <strong>{fact.status.toLowerCase().replace(/_/g, ' ')}</strong>.
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </>
      ) : null}
    </div>
  );
}
