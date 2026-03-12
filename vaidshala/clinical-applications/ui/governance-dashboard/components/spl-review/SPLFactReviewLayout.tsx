'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Loader2,
  AlertTriangle,
  Activity,
  Baby,
  Package,
  Beaker,
  Heart,
  Clock,
  CheckCircle2,
  Filter,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { SPLFactCard } from './SPLFactCard';
import { SafetySignalCard } from './SafetySignalCard';
import { InteractionCard } from './InteractionCard';
import { ReproductiveSafetyCard } from './ReproductiveSafetyCard';
import { FormularyCard } from './FormularyCard';
import { LabReferenceCard } from './LabReferenceCard';
import { HtmlSourcePanel } from './HtmlSourcePanel';
import { splReviewApi } from '@/lib/spl-api';
import type {
  SPLDerivedFact,
  SPLFactType,
  GovernanceStatus,
  SafetySignalData,
  InteractionData,
  ReproductiveSafetyData,
  FormularyData,
  LabReferenceData,
} from '@/types/spl-review';
import { FACT_TYPE_LABELS } from '@/types/spl-review';

// ============================================================================
// Pipeline → UI field normalization
// ============================================================================
// The Go pipeline stores facts with its own field naming convention.
// These helpers map pipeline fields → the shape each card component expects.

/** Map pipeline severity to UI clinicalAction */
function severityToAction(severity: string | undefined): InteractionData['clinicalAction'] {
  switch (severity?.toUpperCase()) {
    case 'CONTRAINDICATED': return 'CONTRAINDICATED';
    case 'MAJOR': return 'AVOID';
    case 'MODERATE': return 'MONITOR';
    case 'MINOR': return 'INFORM';
    default: return 'INFORM';
  }
}

/** Normalize INTERACTION factData from pipeline shape → InteractionData */
function normalizeInteraction(raw: Record<string, unknown>): InteractionData {
  return {
    // Pipeline uses interactantName for the interacting drug; objectDrug is the subject
    objectDrug: (raw.interactantName as string) || (raw.precipitantDrug as string) || (raw.objectDrug as string) || 'Unknown Drug',
    objectDrugClass: (raw.objectDrugClass as string) || (raw.drugClass as string) || undefined,
    clinicalEffect: (raw.clinicalEffect as string) || 'See source text',
    direction: (raw.direction as InteractionData['direction']) || 'UNKNOWN',
    clinicalAction: (raw.clinicalAction as InteractionData['clinicalAction']) || severityToAction(raw.severity as string),
    mechanism: (raw.mechanism as string) || undefined,
    enzyme: (raw.enzyme as string) || undefined,
    sourcePhrase: (raw.sourcePhrase as string) || undefined,
    // Extra fields passed through for display
    ...(raw.management ? { management: raw.management } : {}),
    ...(raw.objectDrug ? { subjectDrug: raw.objectDrug } : {}),
  } as InteractionData;
}

/** Normalize LAB_REFERENCE factData from pipeline shape → LabReferenceData */
function normalizeLabReference(raw: Record<string, unknown>): LabReferenceData {
  // Pipeline uses labName; card expects labTest
  const rangeLow = raw.referenceRangeLow as string | number | null;
  const rangeHigh = raw.referenceRangeHigh as string | number | null;
  const rangeStr = (rangeLow != null && rangeHigh != null)
    ? `${rangeLow} – ${rangeHigh}${raw.unit ? ' ' + raw.unit : ''}`
    : (raw.referenceRange as string) || undefined;

  return {
    labTest: (raw.labName as string) || (raw.labTest as string) || 'Unknown Lab',
    referenceRange: rangeStr,
    monitoringFrequency: (raw.monitoringFrequency as string) || undefined,
    clinicalContext: (raw.clinicalContext as string) || undefined,
  };
}

/** Normalize SAFETY_SIGNAL factData from pipeline shape → SafetySignalData */
function normalizeSafetySignal(raw: Record<string, unknown>): SafetySignalData & { signalType?: string; recommendation?: string; description?: string } {
  // Pipeline stores meddraPT as numeric code (e.g. "10036556") and meddraName as the
  // human-readable term (e.g. "Pregnancy"). The card expects meddraPT as the name and
  // meddraPTCode as the code. termConfidence >= 0.9 indicates MedDRA validation succeeded.
  const termConf = typeof raw.termConfidence === 'number' ? raw.termConfidence : 0;
  const hasValidMedDRA = termConf >= 0.9 && !!(raw.meddraPT || raw.meddraLLT);

  return {
    conditionName: (raw.conditionName as string) || 'Unknown Condition',
    // meddraName is the human-readable PT name; fall back to conditionName
    meddraPT: (raw.meddraName as string) || (raw.conditionName as string) || undefined,
    // meddraPT from pipeline is actually the numeric code
    meddraPTCode: (raw.meddraPT as string) || (raw.meddraLLT as string) || undefined,
    // Pipeline stores SOC as meddraSOCName (human-readable) or meddraSOC (code)
    meddraSOC: (raw.meddraSOCName as string) || (raw.meddraSOC as string) || (raw.soc as string) || undefined,
    frequency: (raw.frequency as string) || undefined,
    frequencyBand: (raw.frequencyBand as string) || undefined,
    severity: (raw.severity as string) || undefined,
    sourcePhrase: (raw.sourcePhrase as string) || undefined,
    meddraValidated: hasValidMedDRA,
    // Extra fields for enriched display
    signalType: (raw.signalType as string) || undefined,
    recommendation: (raw.recommendation as string) || undefined,
    description: (raw.description as string) || undefined,
  };
}

/** Normalize REPRODUCTIVE_SAFETY factData from pipeline shape → ReproductiveSafetyData */
function normalizeReproductive(raw: Record<string, unknown>): ReproductiveSafetyData {
  return {
    category: (raw.category as ReproductiveSafetyData['category']) || 'PREGNANCY',
    riskLevel: (raw.riskLevel as string) || 'Unknown',
    fdaCategory: (raw.fdaCategory as string) || undefined,
    ridPercent: (raw.ridPercent as string) || (raw.relativeInfantDose != null ? String(raw.relativeInfantDose) : undefined),
    pllrSummary: (raw.pllrSummary as string) || undefined,
    population: (raw.population as string) || undefined,
  };
}

// ============================================================================
// SPL Fact Review Layout — Split-pane review experience
//
// Left pane: Scrollable fact card list with review actions
// Right pane: HTML source panel showing the SPL section where the fact
//             was extracted, with the source phrase highlighted.
//
// Used from the drug detail page when the pharmacist clicks
// "Review Pending Facts" or selects a specific fact type.
// ============================================================================

// Fact type icons
const FACT_TYPE_ICONS: Record<SPLFactType, React.ComponentType<{ className?: string }>> = {
  SAFETY_SIGNAL: AlertTriangle,
  INTERACTION: Activity,
  REPRODUCTIVE_SAFETY: Baby,
  FORMULARY: Package,
  LAB_REFERENCE: Beaker,
  ORGAN_IMPAIRMENT: Heart,
};

// ============================================================================
// Props
// ============================================================================

interface SPLFactReviewLayoutProps {
  drugName: string;
  /** Filter to a specific fact type (optional) */
  initialFactType?: SPLFactType;
  /** Filter to a specific status (optional) */
  initialStatus?: GovernanceStatus;
  /** Callback to go back to the drug detail page */
  onBack: () => void;
}

// ============================================================================
// Status filter options
// ============================================================================

type StatusFilter = 'ALL' | GovernanceStatus;

// ============================================================================
// Component
// ============================================================================

export function SPLFactReviewLayout({
  drugName,
  initialFactType,
  initialStatus,
  onBack,
}: SPLFactReviewLayoutProps) {
  const queryClient = useQueryClient();
  const [factTypeFilter, setFactTypeFilter] = useState<SPLFactType | 'ALL'>(initialFactType || 'ALL');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(initialStatus || 'ALL');
  const [currentIdx, setCurrentIdx] = useState(0);
  const [activeSectionCode, setActiveSectionCode] = useState<string>('');

  // Fetch all facts for this drug via the SPL-specific API
  const { data: factsData, isLoading } = useQuery({
    queryKey: ['spl-review-facts', drugName],
    queryFn: () =>
      splReviewApi.facts.getByDrug(drugName, undefined, 1, 5000),
    refetchInterval: 60000,
  });

  // The SPL API already returns SPLDerivedFact[] — just apply client-side filters
  const allFacts: SPLDerivedFact[] = useMemo(() => factsData?.items || [], [factsData]);

  const filteredFacts: SPLDerivedFact[] = useMemo(() => {
    return allFacts.filter((f) => {
      if (factTypeFilter !== 'ALL' && f.factType !== factTypeFilter) return false;
      if (statusFilter !== 'ALL' && f.governanceStatus !== statusFilter) return false;
      return true;
    });
  }, [allFacts, factTypeFilter, statusFilter]);

  // Current fact
  const currentFact = filteredFacts[currentIdx] || null;

  // Reset index when filter changes
  useEffect(() => {
    setCurrentIdx(0);
  }, [factTypeFilter, statusFilter]);

  // Update section code when fact changes
  useEffect(() => {
    if (currentFact?.sectionCode) {
      setActiveSectionCode(currentFact.sectionCode);
    }
  }, [currentFact]);

  // Get available section codes for section picker
  const availableSections = useMemo(() => {
    const sections = new Set<string>();
    filteredFacts.forEach((f) => {
      if (f.sectionCode) sections.add(f.sectionCode);
    });
    return Array.from(sections);
  }, [filteredFacts]);

  // Fact type counts for filter bar
  const factTypeCounts = useMemo(() => {
    const counts: Record<string, number> = { ALL: allFacts.length };
    allFacts.forEach((f) => {
      const ft = f.factType || 'UNKNOWN';
      counts[ft] = (counts[ft] || 0) + 1;
    });
    return counts;
  }, [allFacts]);

  // Status counts
  const statusCounts = useMemo(() => {
    const counts: Record<string, number> = { ALL: allFacts.length };
    allFacts.forEach((f) => {
      counts[f.governanceStatus] = (counts[f.governanceStatus] || 0) + 1;
    });
    return counts;
  }, [allFacts]);

  // Navigation
  const goTo = useCallback(
    (dir: 'prev' | 'next') => {
      if (dir === 'prev' && currentIdx > 0) setCurrentIdx((i) => i - 1);
      if (dir === 'next' && currentIdx < filteredFacts.length - 1) setCurrentIdx((i) => i + 1);
    },
    [currentIdx, filteredFacts.length]
  );

  // Handle highlight source — scroll source panel to fact's source phrase
  const handleHighlightSource = useCallback((fact: SPLDerivedFact) => {
    if (fact.sectionCode) {
      setActiveSectionCode(fact.sectionCode);
    }
  }, []);

  // Action complete — invalidate queries
  const handleActionComplete = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['spl-review-facts', drugName] });
    queryClient.invalidateQueries({ queryKey: ['spl-completeness'] });
    queryClient.invalidateQueries({ queryKey: ['spl-queue-all'] });
  }, [queryClient, drugName]);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      switch (e.key) {
        case 'ArrowLeft':
        case 'k':
          goTo('prev');
          break;
        case 'ArrowRight':
        case 'j':
          goTo('next');
          break;
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [goTo]);

  // Get source phrase from current fact for HTML highlighting
  const sourcePhrase = useMemo(() => {
    if (!currentFact) return '';
    const data = currentFact.factData as Record<string, unknown>;
    return (data?.sourcePhrase as string) || (data?.conditionName as string) || '';
  }, [currentFact]);

  // ── Loading State ────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="h-[calc(100vh-64px)] -m-6 flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-6 w-6 text-blue-500 animate-spin" />
          <p className="text-sm text-gray-500">Loading facts for {drugName}...</p>
        </div>
      </div>
    );
  }

  // ── Main Layout ──────────────────────────────────────────────────
  return (
    <div className="h-[calc(100vh-64px)] -m-6 flex flex-col font-sans bg-[#F7F8FA]">
      {/* ── Top Bar ───────────────────────────────────────────────── */}
      <div className="h-[52px] bg-[#1B3A5C] flex items-center justify-between px-6 shrink-0">
        <div className="flex items-center gap-4">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-white/70 hover:text-white text-xs transition-colors"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back
          </button>
          <span className="text-white/30">|</span>
          <span className="text-white text-sm font-semibold capitalize">{drugName}</span>
          <span className="text-white/50 text-xs">
            SPL Fact Review &middot; {filteredFacts.length} facts
          </span>
        </div>

        {/* Navigation */}
        <div className="flex items-center gap-3">
          <span className="text-white/60 text-xs">
            {filteredFacts.length > 0 ? `${currentIdx + 1} / ${filteredFacts.length}` : '0 facts'}
          </span>
          <div className="flex gap-1">
            <button
              onClick={() => goTo('prev')}
              disabled={currentIdx === 0}
              className="p-1.5 rounded text-white/50 hover:text-white hover:bg-white/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <button
              onClick={() => goTo('next')}
              disabled={currentIdx >= filteredFacts.length - 1}
              className="p-1.5 rounded text-white/50 hover:text-white hover:bg-white/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* ── Filter Bar ────────────────────────────────────────────── */}
      <div className="px-6 py-2 bg-white border-b border-gray-200 flex items-center gap-4 shrink-0 overflow-x-auto">
        {/* Fact type filter */}
        <div className="flex items-center gap-1.5">
          <Filter className="h-3.5 w-3.5 text-gray-400" />
          <span className="text-[10px] text-gray-500 uppercase tracking-wider font-semibold">Type:</span>
          <button
            onClick={() => setFactTypeFilter('ALL')}
            className={cn(
              'text-[11px] px-2.5 py-1 rounded-full border font-medium transition-colors',
              factTypeFilter === 'ALL'
                ? 'bg-gray-800 text-white border-gray-800'
                : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100'
            )}
          >
            All ({factTypeCounts.ALL || 0})
          </button>
          {(['SAFETY_SIGNAL', 'INTERACTION', 'REPRODUCTIVE_SAFETY', 'FORMULARY', 'LAB_REFERENCE', 'ORGAN_IMPAIRMENT'] as SPLFactType[]).map((ft) => {
            const count = factTypeCounts[ft] || 0;
            if (count === 0) return null;
            const Icon = FACT_TYPE_ICONS[ft];
            return (
              <button
                key={ft}
                onClick={() => setFactTypeFilter(factTypeFilter === ft ? 'ALL' : ft)}
                className={cn(
                  'text-[11px] px-2.5 py-1 rounded-full border font-medium transition-colors flex items-center gap-1',
                  factTypeFilter === ft
                    ? 'bg-blue-600 text-white border-blue-600'
                    : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100'
                )}
              >
                <Icon className="h-3 w-3" />
                {FACT_TYPE_LABELS[ft].split(' ')[0]} ({count})
              </button>
            );
          })}
        </div>

        <div className="w-px h-5 bg-gray-200" />

        {/* Status filter */}
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-gray-500 uppercase tracking-wider font-semibold">Status:</span>
          {[
            { key: 'ALL' as StatusFilter, label: 'All', icon: Filter },
            { key: 'PENDING_REVIEW' as StatusFilter, label: 'Pending', icon: Clock },
            { key: 'APPROVED' as StatusFilter, label: 'Approved', icon: CheckCircle2 },
          ].map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              onClick={() => setStatusFilter(key)}
              className={cn(
                'text-[11px] px-2.5 py-1 rounded-full border font-medium transition-colors flex items-center gap-1',
                statusFilter === key
                  ? 'bg-gray-800 text-white border-gray-800'
                  : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100'
              )}
            >
              <Icon className="h-3 w-3" />
              {label} ({statusCounts[key === 'ALL' ? 'ALL' : key] || 0})
            </button>
          ))}
        </div>
      </div>

      {/* ── Split Pane Content ────────────────────────────────────── */}
      <div className="flex-1 flex overflow-hidden">
        {/* LEFT — Fact Cards (48%) */}
        <div className="w-[48%] flex flex-col px-6 pt-5 pb-4 overflow-y-auto">
          {filteredFacts.length === 0 ? (
            <div className="flex-1 flex items-center justify-center text-gray-400 text-sm text-center">
              <div>
                <Filter className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No facts match the current filters</p>
                <button
                  onClick={() => { setFactTypeFilter('ALL'); setStatusFilter('ALL'); }}
                  className="mt-2 text-blue-600 hover:text-blue-800 text-xs font-medium"
                >
                  Clear filters
                </button>
              </div>
            </div>
          ) : currentFact ? (
            <SPLFactCard
              fact={currentFact}
              onActionComplete={handleActionComplete}
              onHighlightSource={handleHighlightSource}
              isActive
            >
              {/* Render type-specific content based on factType */}
              {currentFact.factType === 'SAFETY_SIGNAL' && (() => {
                const normalized = normalizeSafetySignal(currentFact.factData as Record<string, unknown>);
                return (
                  <SafetySignalCard
                    data={normalized}
                    meddraValidated={normalized.meddraValidated}
                    signalType={normalized.signalType}
                    recommendation={normalized.recommendation}
                    description={normalized.description}
                  />
                );
              })()}
              {currentFact.factType === 'INTERACTION' && (
                <InteractionCard
                  data={normalizeInteraction(currentFact.factData as Record<string, unknown>)}
                />
              )}
              {currentFact.factType === 'REPRODUCTIVE_SAFETY' && (
                <ReproductiveSafetyCard
                  data={normalizeReproductive(currentFact.factData as Record<string, unknown>)}
                />
              )}
              {currentFact.factType === 'FORMULARY' && (
                <FormularyCard
                  data={currentFact.factData as unknown as FormularyData}
                />
              )}
              {currentFact.factType === 'LAB_REFERENCE' && (
                <LabReferenceCard
                  data={normalizeLabReference(currentFact.factData as Record<string, unknown>)}
                />
              )}
              {/* Fallback for ORGAN_IMPAIRMENT or unknown types */}
              {!['SAFETY_SIGNAL', 'INTERACTION', 'REPRODUCTIVE_SAFETY', 'FORMULARY', 'LAB_REFERENCE'].includes(currentFact.factType) && (
                <div className="space-y-2">
                  <h4 className="text-sm font-semibold text-gray-700">
                    {FACT_TYPE_LABELS[currentFact.factType] || currentFact.factType}
                  </h4>
                  <pre className="text-[11px] text-gray-600 bg-gray-50 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
                    {JSON.stringify(currentFact.factData, null, 2)}
                  </pre>
                </div>
              )}
            </SPLFactCard>
          ) : null}

          {/* ── Navigation footer ─────────────────────────────────── */}
          {filteredFacts.length > 0 && (
            <div className="flex justify-between items-center mt-4 pt-3 border-t border-gray-200">
              <button
                onClick={() => goTo('prev')}
                disabled={currentIdx === 0}
                className={cn(
                  'flex items-center gap-1 px-3.5 py-1.5 rounded border text-xs transition-colors',
                  currentIdx === 0
                    ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                    : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700'
                )}
              >
                <ChevronLeft className="h-3 w-3" /> Prev
              </button>
              <span className="text-xs text-gray-500">
                Fact {currentIdx + 1} of {filteredFacts.length}
              </span>
              <button
                onClick={() => goTo('next')}
                disabled={currentIdx >= filteredFacts.length - 1}
                className={cn(
                  'flex items-center gap-1 px-3.5 py-1.5 rounded border text-xs transition-colors',
                  currentIdx >= filteredFacts.length - 1
                    ? 'border-gray-200 text-gray-300 cursor-not-allowed'
                    : 'border-gray-200 text-gray-500 hover:bg-gray-100 hover:text-gray-700'
                )}
              >
                Next <ChevronRight className="h-3 w-3" />
              </button>
            </div>
          )}

          {/* Keyboard hint */}
          <div className="text-center mt-2 text-[10px] text-gray-300 tracking-wide">
            ← → navigate &middot; j/k prev/next
          </div>
        </div>

        {/* RIGHT — HTML Source Panel (52%) */}
        <div className="flex-1 border-l border-gray-200 flex flex-col overflow-hidden">
          {currentFact?.sourceDocumentId && activeSectionCode ? (
            <HtmlSourcePanel
              sourceDocumentId={currentFact.sourceDocumentId}
              sectionCode={activeSectionCode}
              highlightText={sourcePhrase}
              availableSections={availableSections}
              onSectionChange={setActiveSectionCode}
            />
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
              <div className="text-center">
                <p>Select a fact to view its source section</p>
                <p className="text-xs mt-1 text-gray-300">
                  The original SPL HTML will display here with the extracted text highlighted
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
