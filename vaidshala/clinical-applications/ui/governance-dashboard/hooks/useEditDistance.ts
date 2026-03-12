import { useMemo } from 'react';
import type { EditDistanceInfo } from '@/types/pipeline1';

// ============================================================================
// Levenshtein Distance — Source-Constrained Edit Tracking
//
// Used by Phase 2A and Phase 3A to warn reviewers when edits deviate too far
// from the original extracted text. Thresholds:
//   0–15%  → green  (minor correction)
//   15–30% → amber  (significant change — verify against source)
//   >30%   → red    (major deviation — consider rejecting & adding new)
// ============================================================================

/**
 * Classic DP Levenshtein distance — single-row space optimization.
 * O(n*m) time, O(min(n,m)) space.
 */
export function levenshteinDistance(a: string, b: string): number {
  if (a === b) return 0;
  if (a.length === 0) return b.length;
  if (b.length === 0) return a.length;

  // Ensure `a` is the shorter string for O(min(n,m)) space
  if (a.length > b.length) {
    [a, b] = [b, a];
  }

  const aLen = a.length;
  const bLen = b.length;
  let prev = new Array(aLen + 1);
  let curr = new Array(aLen + 1);

  for (let i = 0; i <= aLen; i++) prev[i] = i;

  for (let j = 1; j <= bLen; j++) {
    curr[0] = j;
    for (let i = 1; i <= aLen; i++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      curr[i] = Math.min(
        prev[i] + 1,      // deletion
        curr[i - 1] + 1,  // insertion
        prev[i - 1] + cost // substitution
      );
    }
    [prev, curr] = [curr, prev];
  }

  return prev[aLen];
}

/**
 * Returns the edit distance severity level based on change percentage.
 */
export function getEditSeverity(changePercentage: number): 'green' | 'amber' | 'red' {
  if (changePercentage <= 15) return 'green';
  if (changePercentage <= 30) return 'amber';
  return 'red';
}

export const EDIT_SEVERITY_CONFIG = {
  green: {
    label: 'Minor correction',
    color: 'text-green-700',
    bg: 'bg-green-50',
    border: 'border-green-200',
  },
  amber: {
    label: 'Significant change — verify against source',
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    border: 'border-amber-200',
  },
  red: {
    label: 'Major deviation. Consider rejecting and adding a new fact.',
    color: 'text-red-700',
    bg: 'bg-red-50',
    border: 'border-red-200',
  },
} as const;

/**
 * React hook for computing edit distance between original and edited text.
 * Returns memoized EditDistanceInfo with change percentage.
 */
export function useEditDistance(original: string, edited: string): EditDistanceInfo {
  return useMemo(() => {
    const dist = levenshteinDistance(original, edited);
    const pct = original.length > 0 ? (dist / original.length) * 100 : 0;
    return {
      originalText: original,
      editedText: edited,
      levenshteinDistance: dist,
      changePercentage: Math.round(pct * 10) / 10,
    };
  }, [original, edited]);
}
