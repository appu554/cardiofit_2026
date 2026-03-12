// ============================================================================
// Semantic Text Highlighting — Sprint 1 CoverageGuard Safety Verification
// ============================================================================
//
// Splits span text into segments tagged by clinical token type so the reviewer
// can visually verify that numerics, conditions, and negations were preserved
// faithfully during extraction.
//
// Algorithm: sequential case-insensitive splitting with priority ordering.
// Negations first (longest phrases), then numerics, then conditions.
// Earlier passes "claim" text — later passes skip already-classified segments.
// ============================================================================

import type { SemanticTokens } from '@/types/pipeline1';

export type HighlightSegmentType = 'normal' | 'numeric' | 'condition' | 'negation';

export interface HighlightSegment {
  text: string;
  type: HighlightSegmentType;
}

/**
 * Split `text` into semantically tagged segments using token lists from
 * CoverageGuard. Returns a single normal segment when no tokens are provided.
 */
export function highlightText(
  text: string,
  tokens?: SemanticTokens,
): HighlightSegment[] {
  if (!text) return [];

  // No tokens → entire text is normal
  if (
    !tokens ||
    (tokens.negations.length === 0 &&
      tokens.numerics.length === 0 &&
      tokens.conditions.length === 0)
  ) {
    return [{ text, type: 'normal' }];
  }

  let segments: HighlightSegment[] = [{ text, type: 'normal' }];

  // Negations first — longest phrases, prevents shorter condition words
  // (e.g. "with") from splitting negation phrases (e.g. "not treated with dialysis")
  if (tokens.negations.length > 0) {
    segments = splitOn(segments, tokens.negations, 'negation');
  }
  if (tokens.numerics.length > 0) {
    segments = splitOn(segments, tokens.numerics, 'numeric');
  }
  if (tokens.conditions.length > 0) {
    segments = splitOn(segments, tokens.conditions, 'condition');
  }

  return segments;
}

// ---------------------------------------------------------------------------
// Internal helper — splits normal segments on pattern matches
// ---------------------------------------------------------------------------

function splitOn(
  segments: HighlightSegment[],
  patterns: string[],
  type: HighlightSegmentType,
): HighlightSegment[] {
  const result: HighlightSegment[] = [];

  for (const seg of segments) {
    // Already classified — pass through untouched
    if (seg.type !== 'normal') {
      result.push(seg);
      continue;
    }

    let remaining = seg.text;

    for (const pattern of patterns) {
      if (!pattern) continue;
      const idx = remaining.toLowerCase().indexOf(pattern.toLowerCase());
      if (idx !== -1) {
        // Text before the match
        if (idx > 0) {
          result.push({ text: remaining.slice(0, idx), type: 'normal' });
        }
        // The matched token
        result.push({ text: remaining.slice(idx, idx + pattern.length), type });
        // Advance past the match
        remaining = remaining.slice(idx + pattern.length);
      }
    }

    // Whatever is left after all patterns
    if (remaining) {
      result.push({ text: remaining, type: 'normal' });
    }
  }

  return result;
}
