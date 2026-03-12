'use client';

import { useState, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Loader2, BarChart3, ChevronDown, ChevronUp, PanelLeftOpen, PanelLeftClose } from 'lucide-react';
import Link from 'next/link';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { StatsBar } from '@/components/pipeline1/StatsBar';
import { ReviewProgress } from '@/components/pipeline1/ReviewProgress';
import { PhaseStepper } from '@/components/pipeline1/PhaseStepper';
import { Phase1Report } from '@/components/pipeline1/Phase1Report';
import { Phase2AMandatoryReview } from '@/components/pipeline1/Phase2AMandatoryReview';
import { Phase2BPageBrowse } from '@/components/pipeline1/Phase2BPageBrowse';
import { Phase3AFOnlyCorroboration } from '@/components/pipeline1/Phase3AFOnlyCorroboration';
import { Phase3BL1RecoveryTriage } from '@/components/pipeline1/Phase3BL1RecoveryTriage';
import { Phase4Revalidation } from '@/components/pipeline1/Phase4Revalidation';
import { Phase5SignOff } from '@/components/pipeline1/Phase5SignOff';
import { usePhaseGates } from '@/hooks/usePhaseGates';
import type { ReviewSubPhase } from '@/types/pipeline1';

// =============================================================================
// Master Page — Unified 7-sub-phase layout with persistent sidebar stepper
//
// BEFORE: Phase 2/3 early-returned into FactReviewLayout (no stepper visible).
// AFTER:  All phases render inside a unified [Stepper Sidebar | Phase Content]
//         layout. The stepper is always visible for phase navigation.
// =============================================================================

export default function JobReviewPage() {
  const params = useParams();
  const jobId = params.jobId as string;

  // Core state — defaults to Phase 2B (Page Browse) with report pre-acknowledged
  const [showProgress, setShowProgress] = useState(false);
  const [activePhase, setActivePhase] = useState<ReviewSubPhase>('2b');
  const [reportAcknowledged, setReportAcknowledged] = useState(true);
  const [revalidationRequired, setRevalidationRequired] = useState(false);

  // UI state — header hidden by default, sidebar open by default
  const [headerVisible, setHeaderVisible] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // Fetch job details
  const { data: job, isLoading: jobLoading, error } = useQuery({
    queryKey: ['pipeline1-job', jobId],
    queryFn: () => pipeline1Api.jobs.get(jobId),
  });

  // Phase gates (6 gates for 7 sub-phases)
  const gates = usePhaseGates(jobId, reportAcknowledged, revalidationRequired);

  // ─── Phase transition callbacks ─────────────────────────────────────

  const handleAcknowledgeReport = useCallback(() => {
    setReportAcknowledged(true);
    setActivePhase('2a');
  }, []);

  const handlePhase2AComplete = useCallback(() => {
    setActivePhase('2b');
  }, []);

  const handlePhase2BComplete = useCallback(() => {
    setActivePhase('3a');
  }, []);

  const handlePhase3AComplete = useCallback(() => {
    setActivePhase('3b');
  }, []);

  const handlePhase3BComplete = useCallback(() => {
    setActivePhase('4');
  }, []);

  const handleRevalidationComplete = useCallback((passed: boolean) => {
    if (passed) {
      setRevalidationRequired(false);
      setActivePhase('5');
    }
  }, []);

  const handleSelectPhase = useCallback((phase: ReviewSubPhase) => {
    setActivePhase(phase);
  }, []);

  const handleActionComplete = useCallback(() => {
    // Any span-set modification (ADD, EDIT, REJECT) invalidates the current
    // re-validation. Phase 5 stays locked until re-validation runs again.
    setRevalidationRequired(true);
  }, []);

  // ─── Error / Loading ────────────────────────────────────────────────

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64">
        <p className="text-red-600 font-medium">Failed to load job</p>
        <p className="text-gray-500 text-sm mt-1">{(error as Error).message}</p>
        <Link href="/pipeline1" className="mt-4 btn btn-outline">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Jobs
        </Link>
      </div>
    );
  }

  if (jobLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  // ─── Unified Layout — Stepper always visible ────────────────────────

  return (
    <div className="-m-6 flex flex-col h-[calc(100vh-64px)] overflow-hidden">
      {/* Top: Stats Bar — collapsible, hidden by default */}
      {headerVisible && <StatsBar jobId={jobId} />}

      {/* Back nav (slim) */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200 bg-white">
        <div className="flex items-center gap-2">
          <Link
            href="/pipeline1"
            className="flex items-center text-gray-600 hover:text-gray-900 text-sm"
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            Jobs
          </Link>

          {/* Header toggle button */}
          <button
            onClick={() => setHeaderVisible(!headerVisible)}
            className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors"
            title={headerVisible ? 'Hide stats header' : 'Show stats header'}
          >
            {headerVisible ? (
              <ChevronUp className="h-3 w-3" />
            ) : (
              <ChevronDown className="h-3 w-3" />
            )}
            Stats
          </button>
        </div>
        <div className="flex items-center gap-3">
          {job && (
            <div className="text-xs text-gray-400">
              <span className="font-medium text-gray-600">{job.sourcePdf.replace(/^.*\//, '')}</span>
              {' · '}
              <span className="font-mono">{job.jobId.slice(0, 12)}...</span>
            </div>
          )}
          <div className="relative">
            <button
              onClick={() => setShowProgress(!showProgress)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-gray-100 hover:bg-gray-200 text-gray-700 transition-colors"
            >
              <BarChart3 className="h-3 w-3" />
              Progress
            </button>
            {showProgress && (
              <>
                <div className="fixed inset-0 z-40" onClick={() => setShowProgress(false)} />
                <div className="absolute right-0 top-full mt-1 z-50 w-72 bg-white rounded-lg shadow-lg border border-gray-200">
                  <ReviewProgress jobId={jobId} />
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Main content area: Sidebar + Phase content */}
      <div className="flex-1 overflow-hidden flex">
        {/* LEFT: Phase Stepper sidebar — collapsible */}
        {sidebarOpen ? (
          <div className="w-56 shrink-0 border-r border-gray-200 bg-gray-50 overflow-y-auto">
            <div className="flex items-center justify-between px-3 pt-2">
              <span className="text-[10px] font-bold text-gray-400 uppercase tracking-widest">
                Phases
              </span>
              <button
                onClick={() => setSidebarOpen(false)}
                className="p-0.5 rounded hover:bg-gray-200 text-gray-400 hover:text-gray-600 transition-colors"
                title="Close sidebar"
              >
                <PanelLeftClose className="h-3.5 w-3.5" />
              </button>
            </div>
            <PhaseStepper
              activePhase={activePhase}
              onSelectPhase={handleSelectPhase}
              gates={gates}
            />
          </div>
        ) : (
          <div className="w-8 shrink-0 border-r border-gray-200 bg-gray-50 flex flex-col items-center pt-2">
            <button
              onClick={() => setSidebarOpen(true)}
              className="p-1 rounded hover:bg-gray-200 text-gray-400 hover:text-gray-600 transition-colors"
              title="Open sidebar"
            >
              <PanelLeftOpen className="h-3.5 w-3.5" />
            </button>
          </div>
        )}

        {/* RIGHT: Phase content (fills remaining width) */}
        <div className="flex-1 min-w-0 overflow-hidden">
          {activePhase === '1' && job && (
            <Phase1Report
              jobId={jobId}
              job={job}
              onAcknowledge={handleAcknowledgeReport}
              acknowledged={reportAcknowledged}
            />
          )}

          {activePhase === '2a' && job && (
            <Phase2AMandatoryReview
              jobId={jobId}
              job={job}
              onActionComplete={handleActionComplete}
              onPhaseComplete={handlePhase2AComplete}
            />
          )}

          {activePhase === '2b' && job && (
            <Phase2BPageBrowse
              jobId={jobId}
              job={job}
              onActionComplete={handleActionComplete}
              onPhaseComplete={handlePhase2BComplete}
            />
          )}

          {activePhase === '3a' && job && (
            <Phase3AFOnlyCorroboration
              jobId={jobId}
              job={job}
              onActionComplete={handleActionComplete}
              onPhaseComplete={handlePhase3AComplete}
            />
          )}

          {activePhase === '3b' && job && (
            <Phase3BL1RecoveryTriage
              jobId={jobId}
              job={job}
              onActionComplete={handleActionComplete}
              onPhaseComplete={handlePhase3BComplete}
            />
          )}

          {activePhase === '4' && (
            <Phase4Revalidation
              jobId={jobId}
              onRevalidationComplete={handleRevalidationComplete}
            />
          )}

          {activePhase === '5' && job && (
            <Phase5SignOff
              jobId={jobId}
              job={job}
              gates={gates}
            />
          )}
        </div>
      </div>
    </div>
  );
}
