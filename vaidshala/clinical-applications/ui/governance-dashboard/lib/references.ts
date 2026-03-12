// ============================================================================
// Reference Link Generator
// Constructs typed, versioned references from ClinicalFact data
// References are read-only, immutable, frozen at extraction time
// ============================================================================

import type { ClinicalFact, FactReference } from '@/types/governance';

/**
 * Generate all applicable references for a clinical fact.
 * Used by the Evidence Stack panel and the Approval Gate check.
 */
export function generateReferences(fact: ClinicalFact): FactReference[] {
  const refs: FactReference[] = [];
  const content = fact.content as Record<string, unknown>;
  const meddraPT = content?.meddraPT as string | undefined;
  const conditionName = content?.conditionName as string | undefined;
  const interactantName = content?.interactantName as string | undefined;
  const sectionLoinc = (content?.splSection as string) || (content?.sectionLoinc as string) || undefined;

  // 1. PRIMARY: DailyMed (always — sourceId is the FDA SetID)
  if (fact.sourceId) {
    refs.push({
      type: 'PRIMARY_SOURCE',
      system: 'FDA_DAILYMED',
      label: 'FDA Drug Label (DailyMed)',
      url: `https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=${fact.sourceId}`,
      anchor: sectionLoinc ? { sectionLoinc: String(sectionLoinc) } : undefined,
    });
  }

  // 2. TERMINOLOGY: MedDRA (if meddraPT exists)
  if (meddraPT) {
    refs.push({
      type: 'TERMINOLOGY',
      system: 'MEDDRA',
      label: 'MedDRA Terminology',
      url: `https://tools.meddra.org/browser?term=${meddraPT}`,
    });
  }

  // 3. TERMINOLOGY: RxNorm (always, using rxcui)
  if (fact.rxcui) {
    refs.push({
      type: 'TERMINOLOGY',
      system: 'RXNORM',
      label: 'RxNorm Drug Identity',
      url: `https://mor.nlm.nih.gov/RxNav/search?searchBy=RXCUI&searchTerm=${fact.rxcui}`,
    });
  }

  // 4. REGULATORY: FDA Label PDF
  if (fact.sourceId) {
    refs.push({
      type: 'REGULATORY',
      system: 'FDA_LABEL_PDF',
      label: 'FDA Label PDF',
      url: `https://dailymed.nlm.nih.gov/dailymed/getFile.cfm?setid=${fact.sourceId}&type=pdf`,
    });
  }

  // 5. SECONDARY: openFDA FAERS (for SAFETY_SIGNAL facts)
  if (fact.factType === 'SAFETY_SIGNAL' && fact.drugName) {
    const searchTerm = conditionName || interactantName || '';
    refs.push({
      type: 'SECONDARY_AUTHORITY',
      system: 'FAERS',
      label: 'FDA FAERS (Post-Market)',
      url: `https://api.fda.gov/drug/event.json?search=patient.drug.medicinalproduct:"${encodeURIComponent(fact.drugName)}"&limit=10`,
    });
  }

  // 6. SECONDARY: PubMed
  if (fact.drugName) {
    const concept = conditionName || interactantName || '';
    const query = concept
      ? `${fact.drugName}+AND+${concept}`
      : fact.drugName;
    refs.push({
      type: 'SECONDARY_AUTHORITY',
      system: 'PUBMED',
      label: 'PubMed Literature',
      url: `https://pubmed.ncbi.nlm.nih.gov/?term=${encodeURIComponent(query)}`,
    });
  }

  return refs;
}

/**
 * Check if a fact passes the Approval Gate.
 * Requires ≥1 PRIMARY_SOURCE and ≥1 TERMINOLOGY reference.
 */
export function checkApprovalGate(refs: FactReference[]): {
  canApprove: boolean;
  hasPrimary: boolean;
  hasTerminology: boolean;
} {
  const hasPrimary = refs.some((r) => r.type === 'PRIMARY_SOURCE');
  const hasTerminology = refs.some((r) => r.type === 'TERMINOLOGY');
  return {
    canApprove: hasPrimary && hasTerminology,
    hasPrimary,
    hasTerminology,
  };
}
