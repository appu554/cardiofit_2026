'use client';

import { useState } from 'react';
import {
  AlertCircle,
  Activity,
  Hash,
  Beaker,
  Pill,
  Tag,
  Lightbulb,
  Shield,
  FlaskConical,
  Building2,
  Barcode,
  Info,
  HelpCircle,
  Microscope,
  Stethoscope,
  FileWarning,
  TestTube,
  Eye,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ClinicalFact } from '@/types/governance';

interface FactContentProps {
  fact: ClinicalFact;
}

const SEVERITY_COLORS: Record<string, string> = {
  CRITICAL: 'bg-red-600 text-white',
  HIGH: 'bg-red-100 text-red-800',
  MAJOR: 'bg-red-100 text-red-800',
  MEDIUM: 'bg-amber-100 text-amber-800',
  MODERATE: 'bg-amber-100 text-amber-800',
  LOW: 'bg-green-100 text-green-800',
  MINOR: 'bg-green-100 text-green-800',
};

const CONFIDENCE_BAND_COLORS: Record<string, string> = {
  HIGH: 'text-green-700 bg-green-100',
  MEDIUM: 'text-amber-700 bg-amber-100',
  LOW: 'text-red-700 bg-red-100',
};

// ============================================================================
// RXNORM Drug Identity Lookup
// Known RxCUI → clinical drug description mapping
// ============================================================================

const RXNORM_IDENTITY: Record<string, { display: string; tty: string; components?: string }> = {
  '6809':    { display: 'Metformin Hydrochloride', tty: 'IN', components: 'Oral tablets: 500mg, 850mg, 1000mg' },
  '11289':   { display: 'Warfarin Sodium', tty: 'IN', components: 'Oral tablets: 1mg, 2mg, 2.5mg, 3mg, 4mg, 5mg, 6mg, 7.5mg, 10mg' },
  '1364430': { display: 'Apixaban', tty: 'IN', components: 'Oral tablets: 2.5mg, 5mg (Eliquis®)' },
  '1488564': { display: 'Dapagliflozin Propanediol', tty: 'IN', components: 'Oral tablets: 5mg, 10mg (Farxiga®)' },
};

// ============================================================================
// Clinical Interpretation Logic
// Derives what this fact actually represents for the pharmacist
// ============================================================================

interface ClinicalInterpretation {
  category: 'adverse_event' | 'lab_abnormality' | 'monitoring' | 'contraindication' | 'interaction' | 'dosing';
  label: string;
  description: string;
  actionability: 'requires_action' | 'monitoring_only' | 'informational';
  actionabilityLabel: string;
  icon: typeof AlertCircle;
  bgColor: string;
  textColor: string;
}

function deriveClinicalInterpretation(fact: ClinicalFact): ClinicalInterpretation {
  const content = fact.content as Record<string, unknown>;
  const signalType = (content?.signalType as string) || '';
  const conditionName = (content?.conditionName as string) || '';
  const severity = (content?.severity as string) || '';
  const requiresMonitor = content?.requiresMonitor as boolean;
  const meddraName = (content?.meddraName as string) || '';

  // Lab test names typically indicate lab monitoring, not adverse events
  const LAB_PATTERNS = /\b(blood|serum|plasma|urine|level|ratio|count|INR|eGFR|creatinine|triglycerides|glucose|HbA1c|ALT|AST|WBC|platelet|hemoglobin|hematocrit|potassium|sodium|calcium|bilirubin|albumin|weight|BMI)\b/i;
  const isLabRelated = LAB_PATTERNS.test(conditionName) || LAB_PATTERNS.test(meddraName);

  // Interaction types
  if (fact.factType === 'INTERACTION' || fact.factType === 'DRUG_INTERACTION') {
    return {
      category: 'interaction',
      label: 'Drug-Drug Interaction',
      description: `Potential interaction between ${fact.drugName} and ${content?.interactantName || 'another drug'}.`,
      actionability: severity === 'CRITICAL' || severity === 'HIGH' ? 'requires_action' : 'monitoring_only',
      actionabilityLabel: severity === 'CRITICAL' || severity === 'HIGH' ? 'May require intervention' : 'Monitor if co-prescribed',
      icon: AlertCircle,
      bgColor: 'bg-purple-50',
      textColor: 'text-purple-800',
    };
  }

  // Contraindications
  if (fact.factType === 'CONTRAINDICATION') {
    return {
      category: 'contraindication',
      label: 'Contraindication',
      description: `Use of ${fact.drugName} is contraindicated in this context.`,
      actionability: 'requires_action',
      actionabilityLabel: 'Do not prescribe in this condition',
      icon: FileWarning,
      bgColor: 'bg-red-50',
      textColor: 'text-red-800',
    };
  }

  // Dosing rules
  if (fact.factType === 'DOSING_RULE' || fact.factType === 'RENAL_ADJUSTMENT' || fact.factType === 'HEPATIC_ADJUSTMENT') {
    return {
      category: 'dosing',
      label: 'Dosing Consideration',
      description: `Dosage adjustment information for ${fact.drugName}.`,
      actionability: 'requires_action',
      actionabilityLabel: 'Adjust dose per guidelines',
      icon: TestTube,
      bgColor: 'bg-blue-50',
      textColor: 'text-blue-800',
    };
  }

  // Safety signals — distinguish lab vs adverse event
  if (isLabRelated) {
    return {
      category: 'lab_abnormality',
      label: 'Laboratory Finding',
      description: `A laboratory parameter change observed during ${fact.drugName} treatment. This represents a measurable lab value, not a diagnosed disease or boxed warning.`,
      actionability: requiresMonitor ? 'monitoring_only' : 'informational',
      actionabilityLabel: requiresMonitor ? 'Monitor lab values during treatment' : 'Informational — routine monitoring sufficient',
      icon: Microscope,
      bgColor: 'bg-cyan-50',
      textColor: 'text-cyan-800',
    };
  }

  // General adverse event
  if (signalType === 'ADVERSE_REACTION' || signalType === 'WARNING') {
    const isSevere = severity === 'CRITICAL' || severity === 'HIGH';
    return {
      category: 'adverse_event',
      label: 'Adverse Reaction',
      description: `An adverse reaction reported during ${fact.drugName} treatment. ${isSevere ? 'This is a clinically significant finding.' : 'Clinical significance should be assessed in context.'}`,
      actionability: isSevere ? 'requires_action' : 'monitoring_only',
      actionabilityLabel: isSevere ? 'Clinically significant — requires awareness' : 'Monitor for this adverse reaction',
      icon: Stethoscope,
      bgColor: 'bg-amber-50',
      textColor: 'text-amber-800',
    };
  }

  // Default
  return {
    category: 'adverse_event',
    label: 'Clinical Finding',
    description: `A clinical observation related to ${fact.drugName} treatment.`,
    actionability: 'informational',
    actionabilityLabel: 'Review in clinical context',
    icon: Activity,
    bgColor: 'bg-gray-50',
    textColor: 'text-gray-800',
  };
}

// ============================================================================
// Extraction Reason Tooltip
// ============================================================================

function deriveExtractionReason(fact: ClinicalFact): string[] {
  const reasons: string[] = [];
  const content = fact.content as Record<string, unknown>;

  if (fact.sourceType === 'ETL' || fact.extractionMethod === 'STRUCTURED_PARSE') {
    reasons.push('Appeared in FDA-approved drug label');
  }
  if (content?.meddraPT) {
    reasons.push('Valid MedDRA Preferred Term mapped');
  }
  if (content?.termConfidence && (content.termConfidence as number) >= 0.8) {
    reasons.push(`Term confidence: ${((content.termConfidence as number) * 100).toFixed(0)}%`);
  }
  if (fact.confidenceScore >= 0.9) {
    reasons.push('High extraction confidence');
  }
  if (content?.signalType === 'ADVERSE_REACTION') {
    reasons.push('Section: Adverse Reactions');
  } else if (content?.signalType === 'WARNING') {
    reasons.push('Section: Warnings & Precautions');
  }
  if (reasons.length === 0) {
    reasons.push('Extracted from structured clinical data source');
  }
  return reasons;
}

// ============================================================================
// Component
// ============================================================================

export function FactContent({ fact }: FactContentProps) {
  const [showExtractionTooltip, setShowExtractionTooltip] = useState(false);
  const content = fact.content as Record<string, unknown>;

  const conditionName =
    (content?.conditionName as string) ||
    (content?.interactantName as string) ||
    (content?.condition as string) ||
    null;
  const meddraPT = content?.meddraPT as string | undefined;
  const meddraName = content?.meddraName as string | undefined;
  const meddraSOCName = content?.meddraSOCName as string | undefined;
  const severity =
    (content?.severity as string) ||
    (fact.severity as string) ||
    null;
  const signalType = content?.signalType as string | undefined;
  const interactionType = content?.interactionType as string | undefined;
  const recommendation = content?.recommendation as string | undefined;
  const extractionMethod = fact.extractionMethod || 'UNKNOWN';

  // Canonical key
  const canonicalKey = [fact.rxcui, fact.factType, meddraPT || '—'].join(' | ');

  // Clinical interpretation
  const interpretation = deriveClinicalInterpretation(fact);
  const InterpIcon = interpretation.icon;

  // Drug identity from RxNorm
  const rxnormIdentity = RXNORM_IDENTITY[fact.rxcui];

  // Extraction reasons
  const extractionReasons = deriveExtractionReason(fact);

  return (
    <div className="card h-full">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">The Fact</h2>
      </div>

      <div className="card-body space-y-4">
        {/* ── Clinical Interpretation Banner ── */}
        <div className={cn('rounded-lg p-3 border', interpretation.bgColor, `border-${interpretation.textColor.replace('text-', '')}/20`)}>
          <div className="flex items-start space-x-2">
            <InterpIcon className={cn('h-4 w-4 mt-0.5 shrink-0', interpretation.textColor)} />
            <div>
              <p className={cn('text-sm font-semibold', interpretation.textColor)}>
                {interpretation.label}
              </p>
              <p className="text-xs text-gray-600 mt-0.5">
                {interpretation.description}
              </p>
              <div className="mt-1.5 flex items-center">
                <span className={cn(
                  'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
                  interpretation.actionability === 'requires_action' && 'bg-red-100 text-red-700',
                  interpretation.actionability === 'monitoring_only' && 'bg-amber-100 text-amber-700',
                  interpretation.actionability === 'informational' && 'bg-gray-100 text-gray-600',
                )}>
                  {interpretation.actionability === 'requires_action' && '● '}
                  {interpretation.actionability === 'monitoring_only' && '○ '}
                  {interpretation.actionability === 'informational' && '○ '}
                  {interpretation.actionabilityLabel}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* ── Drug Identity (Enhanced with RxNorm) ── */}
        {fact.drugName && (
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 space-y-2">
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
              <Pill className="h-3.5 w-3.5 mr-1" />
              Drug Identity
            </p>
            <p className="text-lg font-bold text-gray-900">
              {rxnormIdentity ? rxnormIdentity.display : fact.drugName}
            </p>
            {rxnormIdentity && (
              <>
                <div className="flex items-center space-x-2">
                  <span className="text-xs text-gray-500">Source:</span>
                  <code className="text-xs bg-white px-1.5 py-0.5 rounded border border-slate-200">
                    RxNorm ({rxnormIdentity.tty})
                  </code>
                  <code className="text-xs bg-white px-1.5 py-0.5 rounded border border-slate-200">
                    RxCUI: {fact.rxcui}
                  </code>
                </div>
                {rxnormIdentity.components && (
                  <div>
                    <span className="text-xs text-gray-500">Forms:</span>{' '}
                    <span className="text-xs text-gray-700">{rxnormIdentity.components}</span>
                  </div>
                )}
              </>
            )}
            {!rxnormIdentity && (
              <div className="flex items-center space-x-2">
                <span className="text-xs text-gray-500">RxCUI:</span>
                <code className="text-xs bg-white px-1.5 py-0.5 rounded border border-slate-200">{fact.rxcui}</code>
              </div>
            )}
            {fact.genericName && fact.genericName !== fact.drugName && (
              <div>
                <span className="text-xs text-gray-500">Generic:</span>{' '}
                <span className="text-sm font-medium text-gray-900">{fact.genericName}</span>
              </div>
            )}
            {fact.manufacturer && (
              <div className="flex items-start">
                <Building2 className="h-3 w-3 mr-1 mt-0.5 text-gray-400 shrink-0" />
                <span className="text-xs text-gray-600">{fact.manufacturer}</span>
              </div>
            )}
            {fact.atcCodes && fact.atcCodes.length > 0 && (
              <div>
                <span className="text-xs text-gray-500">ATC:</span>{' '}
                {fact.atcCodes.map((code, i) => (
                  <code key={i} className="text-xs bg-white px-1.5 py-0.5 rounded border border-slate-200 mr-1">{code}</code>
                ))}
              </div>
            )}
            {fact.ndcCodes && fact.ndcCodes.length > 0 && (
              <div>
                <span className="text-xs text-gray-500 flex items-center">
                  <Barcode className="h-3 w-3 mr-1" />
                  NDC:
                </span>
                <div className="flex flex-wrap gap-1 mt-0.5">
                  {fact.ndcCodes.slice(0, 5).map((code, i) => (
                    <code key={i} className="text-xs bg-white px-1.5 py-0.5 rounded border border-slate-200">{code}</code>
                  ))}
                  {fact.ndcCodes.length > 5 && (
                    <span className="text-xs text-gray-400">+{fact.ndcCodes.length - 5} more</span>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Condition / Concept ── */}
        {conditionName && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
              <Activity className="h-3.5 w-3.5 mr-1" />
              Condition / Concept
            </p>
            <p className="text-lg font-semibold text-gray-900 mt-1">{conditionName}</p>
          </div>
        )}

        {/* ── MedDRA PT ── */}
        {meddraPT && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
              <Hash className="h-3.5 w-3.5 mr-1" />
              MedDRA PT
            </p>
            <div className="mt-1 flex items-center space-x-2">
              <code className="text-sm bg-gray-100 px-2 py-0.5 rounded font-mono">{meddraPT}</code>
              {meddraName && <span className="text-sm text-gray-700">{meddraName}</span>}
            </div>
          </div>
        )}

        {/* ── MedDRA SOC ── */}
        {meddraSOCName && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider">SOC</p>
            <p className="text-sm text-gray-700 mt-1">{meddraSOCName}</p>
          </div>
        )}

        {/* ── Severity ── */}
        {severity && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
              <AlertCircle className="h-3.5 w-3.5 mr-1" />
              Severity
            </p>
            <span
              className={cn(
                'inline-flex items-center mt-1 px-2.5 py-0.5 rounded text-xs font-bold',
                SEVERITY_COLORS[severity.toUpperCase()] || 'bg-gray-100 text-gray-700'
              )}
            >
              {severity}
            </span>
          </div>
        )}

        {/* ── Confidence ── */}
        <div>
          <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
            <Shield className="h-3.5 w-3.5 mr-1" />
            Confidence
          </p>
          <div className="mt-1 flex items-center space-x-3">
            <div className="flex-1 h-2.5 bg-gray-200 rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full',
                  fact.confidenceScore >= 0.9 && 'bg-green-500',
                  fact.confidenceScore >= 0.7 && fact.confidenceScore < 0.9 && 'bg-amber-500',
                  fact.confidenceScore < 0.7 && 'bg-red-500'
                )}
                style={{ width: `${fact.confidenceScore * 100}%` }}
              />
            </div>
            <span className="text-sm font-semibold text-gray-700">
              {(fact.confidenceScore * 100).toFixed(0)}%
            </span>
            {fact.confidenceBand && (
              <span
                className={cn(
                  'text-xs px-1.5 py-0.5 rounded font-medium',
                  CONFIDENCE_BAND_COLORS[fact.confidenceBand] || 'bg-gray-100 text-gray-600'
                )}
              >
                {fact.confidenceBand}
              </span>
            )}
          </div>
        </div>

        {/* ── Canonical Key ── */}
        <div>
          <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
            <Tag className="h-3.5 w-3.5 mr-1" />
            Canonical Key
          </p>
          <code className="text-xs bg-gray-100 px-2 py-1 rounded font-mono text-gray-600 block mt-1 break-all">
            {canonicalKey}
          </code>
        </div>

        {/* ── Signal / Interaction Type ── */}
        {(signalType || interactionType) && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              {signalType ? 'Signal Type' : 'Interaction Type'}
            </p>
            <span className="inline-flex mt-1 px-2 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-800">
              {signalType || interactionType}
            </span>
          </div>
        )}

        {/* ── Recommendation ── */}
        {recommendation && (
          <div>
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
              <Lightbulb className="h-3.5 w-3.5 mr-1" />
              Recommendation
            </p>
            <p className="text-sm text-gray-800 mt-1 bg-blue-50 border border-blue-100 p-3 rounded-lg">
              {recommendation}
            </p>
          </div>
        )}

        {/* ── Extraction Method + Why Tooltip ── */}
        <div className="relative">
          <p className="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center">
            <Beaker className="h-3.5 w-3.5 mr-1" />
            Extraction Method
            <button
              onClick={() => setShowExtractionTooltip(!showExtractionTooltip)}
              className="ml-1.5 text-gray-400 hover:text-gray-600"
              title="Why was this extracted?"
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          </p>
          <span className="inline-flex mt-1 px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
            {extractionMethod.replace(/_/g, ' ')}
          </span>

          {/* Extraction Tooltip */}
          {showExtractionTooltip && (
            <div className="mt-2 p-3 bg-slate-800 text-white rounded-lg text-xs space-y-1">
              <p className="font-medium flex items-center">
                <Eye className="h-3 w-3 mr-1" />
                Why was this extracted?
              </p>
              <ul className="space-y-0.5 ml-4">
                {extractionReasons.map((reason, i) => (
                  <li key={i} className="list-disc text-slate-300">{reason}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
