'use client';

import { useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileText,
  BookOpen,
  Pill,
  Activity,
  Search,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ClinicalFact, FactReference } from '@/types/governance';
import { generateReferences } from '@/lib/references';
import { DailyMedViewer } from './DailyMedViewer';

interface EvidencePanelProps {
  fact: ClinicalFact;
}

const SECTION_CONFIG: Record<string, {
  icon: typeof FileText;
  borderColor: string;
  bgColor: string;
}> = {
  FDA_DAILYMED: { icon: FileText, borderColor: 'border-l-blue-500', bgColor: 'bg-blue-50' },
  MEDDRA: { icon: BookOpen, borderColor: 'border-l-green-500', bgColor: 'bg-green-50' },
  RXNORM: { icon: Pill, borderColor: 'border-l-orange-500', bgColor: 'bg-orange-50' },
  FDA_LABEL_PDF: { icon: FileText, borderColor: 'border-l-indigo-500', bgColor: 'bg-indigo-50' },
  FAERS: { icon: Activity, borderColor: 'border-l-red-500', bgColor: 'bg-red-50' },
  PUBMED: { icon: Search, borderColor: 'border-l-purple-500', bgColor: 'bg-purple-50' },
};

function AccordionSection({
  reference,
  fact,
  defaultOpen = false,
  onViewLabel,
}: {
  reference: FactReference;
  fact: ClinicalFact;
  defaultOpen?: boolean;
  onViewLabel?: () => void;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const config = SECTION_CONFIG[reference.system] || {
    icon: FileText,
    borderColor: 'border-l-gray-400',
    bgColor: 'bg-gray-50',
  };
  const Icon = config.icon;
  const content = fact.content as Record<string, unknown>;

  return (
    <div className={cn('border-l-4 rounded-lg overflow-hidden', config.borderColor)}>
      <button
        onClick={() => setOpen(!open)}
        className={cn(
          'w-full flex items-center justify-between px-4 py-3 text-left',
          config.bgColor,
          'hover:brightness-95 transition-all'
        )}
      >
        <div className="flex items-center space-x-3">
          <Icon className="h-4 w-4 shrink-0" />
          <span className="font-medium text-sm">{reference.label}</span>
          <span className="text-xs px-1.5 py-0.5 rounded bg-white/70 text-gray-600">
            {reference.type.replace(/_/g, ' ')}
          </span>
        </div>
        {open ? (
          <ChevronDown className="h-4 w-4 text-gray-500" />
        ) : (
          <ChevronRight className="h-4 w-4 text-gray-500" />
        )}
      </button>

      {open && (
        <div className="px-4 py-3 bg-white border-t border-gray-100 space-y-3">
          {/* System-specific details */}
          {reference.system === 'FDA_DAILYMED' && (
            <>
              {reference.anchor?.sectionLoinc && (
                <div className="text-sm">
                  <span className="text-gray-500">Section LOINC:</span>{' '}
                  <code className="bg-gray-100 px-1 rounded text-xs">{reference.anchor.sectionLoinc}</code>
                </div>
              )}
              {fact.evidenceSpans && fact.evidenceSpans.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-gray-500 uppercase mb-1">Evidence Span</p>
                  {fact.evidenceSpans.map((span, i) => (
                    <blockquote key={i} className="text-sm text-gray-700 italic border-l-2 border-blue-400 pl-3 mt-1 bg-blue-50/50 py-1 rounded-r">
                      &ldquo;{span}&rdquo;
                    </blockquote>
                  ))}
                </div>
              )}
              {onViewLabel && (
                <button
                  onClick={onViewLabel}
                  className="inline-flex items-center text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 px-3 py-1.5 rounded-lg transition-colors"
                >
                  <FileText className="h-3.5 w-3.5 mr-1.5" />
                  View Full Label
                </button>
              )}
            </>
          )}

          {reference.system === 'MEDDRA' && (
            <div className="space-y-1 text-sm">
              <div>
                <span className="text-gray-500">PT:</span>{' '}
                <code className="bg-gray-100 px-1 rounded text-xs">{content?.meddraPT as string}</code>
                {' — '}
                <span className="text-gray-800">{content?.meddraName as string}</span>
              </div>
              {(content?.meddraLLT as string) && (
                <div>
                  <span className="text-gray-500">LLT:</span>{' '}
                  <span className="text-gray-700">{String(content.meddraLLT)}</span>
                </div>
              )}
              {(content?.meddraSOCName as string) && (
                <div>
                  <span className="text-gray-500">SOC:</span>{' '}
                  <span className="text-gray-700">{String(content.meddraSOCName)}</span>
                </div>
              )}
              {content?.termConfidence != null && (
                <div>
                  <span className="text-gray-500">Term Confidence:</span>{' '}
                  <span className="text-gray-700">{((content.termConfidence as number) * 100).toFixed(0)}%</span>
                </div>
              )}
            </div>
          )}

          {reference.system === 'RXNORM' && (
            <div className="space-y-1 text-sm">
              <div>
                <span className="text-gray-500">RxCUI:</span>{' '}
                <code className="bg-gray-100 px-1 rounded text-xs">{fact.rxcui}</code>
              </div>
              <div>
                <span className="text-gray-500">Drug:</span>{' '}
                <span className="text-gray-800">{fact.drugName}</span>
              </div>
            </div>
          )}

          {reference.system === 'FAERS' && (
            <p className="text-sm text-gray-600">
              Search openFDA FAERS for post-market adverse event reports for{' '}
              <strong>{fact.drugName}</strong>.
            </p>
          )}

          {reference.system === 'PUBMED' && (
            <p className="text-sm text-gray-600">
              Search PubMed for literature on{' '}
              <strong>{fact.drugName}</strong>
              {(content?.conditionName as string) && (
                <> and <strong>{String(content.conditionName)}</strong></>
              )}.
            </p>
          )}

          {/* Link */}
          <a
            href={reference.url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center text-sm text-blue-600 hover:text-blue-700 font-medium"
          >
            <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
            {reference.system === 'FDA_DAILYMED' && 'View in DailyMed'}
            {reference.system === 'FDA_LABEL_PDF' && 'View FDA PDF'}
            {reference.system === 'MEDDRA' && 'Open MedDRA Browser'}
            {reference.system === 'RXNORM' && 'Verify in RxNav'}
            {reference.system === 'FAERS' && 'Search FAERS'}
            {reference.system === 'PUBMED' && 'Search PubMed'}
          </a>
        </div>
      )}
    </div>
  );
}

function deriveEvidenceQuality(fact: ClinicalFact, references: FactReference[]) {
  const content = fact.content as Record<string, unknown>;
  const sourceType = fact.sourceType || 'UNKNOWN';
  const confidence = fact.confidenceScore ?? 0;
  const hasDailyMed = references.some(r => r.system === 'FDA_DAILYMED');
  const hasMedDRA = references.some(r => r.system === 'MEDDRA');
  const hasRxNorm = references.some(r => r.system === 'RXNORM');
  const hasFAERS = references.some(r => r.system === 'FAERS');

  // Source tier
  const sourceTier = sourceType === 'FDA_SPL' ? 'Regulatory (FDA Label)'
    : sourceType === 'AUTHORITATIVE' ? 'Authoritative Source'
    : sourceType === 'LITERATURE' ? 'Published Literature'
    : 'Derived / Computed';

  // Study design proxy
  const studyDesign = hasDailyMed ? 'FDA-Approved Label (Phase III+ evidence)'
    : hasFAERS ? 'Post-Market Surveillance (FAERS)'
    : 'Extracted Clinical Knowledge';

  // Terminology coverage
  const termCoverage: string[] = [];
  if (hasMedDRA) termCoverage.push('MedDRA');
  if (hasRxNorm) termCoverage.push('RxNorm');
  if (content?.meddraSOCName) termCoverage.push('SOC mapped');

  // Regulatory weight
  const regulatoryWeight = sourceType === 'FDA_SPL' ? 'High — FDA-regulated content'
    : sourceType === 'AUTHORITATIVE' ? 'Medium — Expert consensus'
    : 'Low — Requires validation';

  const weightColor = sourceType === 'FDA_SPL' ? 'text-green-700 bg-green-50'
    : sourceType === 'AUTHORITATIVE' ? 'text-amber-700 bg-amber-50'
    : 'text-red-700 bg-red-50';

  return { sourceTier, studyDesign, termCoverage, regulatoryWeight, weightColor, confidence };
}

export function EvidencePanel({ fact }: EvidencePanelProps) {
  const references = generateReferences(fact);
  const [showViewer, setShowViewer] = useState(false);
  const quality = deriveEvidenceQuality(fact, references);

  return (
    <>
      <div className="card">
        <div className="card-header">
          <h2 className="text-lg font-semibold text-gray-900">Evidence Stack</h2>
          <p className="text-sm text-gray-500">
            {references.length} reference{references.length !== 1 ? 's' : ''} — read-only, frozen at extraction
          </p>
        </div>
        <div className="card-body space-y-2">
          {/* Evidence Quality Summary */}
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-2 space-y-2.5">
            <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Evidence Quality</h3>
            <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <div>
                <span className="text-gray-500 text-xs">Source Type</span>
                <p className="font-medium text-gray-800">{quality.sourceTier}</p>
              </div>
              <div>
                <span className="text-gray-500 text-xs">Study Design</span>
                <p className="font-medium text-gray-800">{quality.studyDesign}</p>
              </div>
              <div>
                <span className="text-gray-500 text-xs">Terminology Coverage</span>
                <p className="font-medium text-gray-800">
                  {quality.termCoverage.length > 0 ? quality.termCoverage.join(' · ') : 'None'}
                </p>
              </div>
              <div>
                <span className="text-gray-500 text-xs">Regulatory Weight</span>
                <span className={cn('inline-block text-xs font-medium px-2 py-0.5 rounded mt-0.5', quality.weightColor)}>
                  {quality.regulatoryWeight}
                </span>
              </div>
            </div>
          </div>

          {references.map((ref, i) => (
            <AccordionSection
              key={`${ref.system}-${i}`}
              reference={ref}
              fact={fact}
              defaultOpen={i === 0}
              onViewLabel={ref.system === 'FDA_DAILYMED' ? () => setShowViewer(true) : undefined}
            />
          ))}
        </div>
      </div>

      {showViewer && fact.sourceId && (
        <DailyMedViewer
          setId={fact.sourceId}
          drugName={fact.drugName}
          sectionLoinc={(fact.content as Record<string, unknown>)?.splSection as string | undefined}
          onClose={() => setShowViewer(false)}
        />
      )}
    </>
  );
}
