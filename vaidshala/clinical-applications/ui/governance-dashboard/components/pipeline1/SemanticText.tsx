'use client';

import { highlightText } from '@/lib/pipeline1-highlights';
import { cn } from '@/lib/utils';
import type { SemanticTokens } from '@/types/pipeline1';

// ============================================================================
// SemanticText — Sprint 1 CoverageGuard Safety Verification
//
// Renders extracted fact text with visual highlighting of clinically critical
// tokens: numerics (blue), conditions (amber), negations (red wavy underline).
// Falls back to plain text when no tokens are provided.
// ============================================================================

interface SemanticTextProps {
  text: string;
  tokens?: SemanticTokens;
  className?: string;
}

// Segment type → Tailwind classes
const SEGMENT_STYLES = {
  numeric:
    'bg-blue-100 text-blue-800 font-semibold font-mono text-[0.9em] px-0.5 rounded',
  condition:
    'bg-amber-100 text-amber-800 font-medium italic px-0.5 rounded',
  negation:
    'bg-red-100 text-red-800 font-bold underline decoration-wavy decoration-red-400 px-0.5 rounded',
} as const;

export function SemanticText({ text, tokens, className }: SemanticTextProps) {
  const segments = highlightText(text, tokens);

  const hasTokens =
    tokens &&
    (tokens.numerics.length > 0 ||
      tokens.conditions.length > 0 ||
      tokens.negations.length > 0);

  return (
    <div className={className}>
      {/* Highlighted text */}
      <span>
        {segments.map((seg, i) => {
          if (seg.type === 'normal') {
            return <span key={i}>{seg.text}</span>;
          }
          return (
            <span key={i} className={SEGMENT_STYLES[seg.type]}>
              {seg.text}
            </span>
          );
        })}
      </span>

      {/* Token legend — only when semantic tokens are present */}
      {hasTokens && (
        <div className="flex flex-wrap gap-3 mt-2 text-[10px] text-gray-500 select-none">
          <span>
            <span className="bg-blue-100 text-blue-800 font-semibold font-mono px-1 rounded">
              123
            </span>{' '}
            Numeric
          </span>
          <span>
            <span className="bg-amber-100 text-amber-800 font-medium italic px-1 rounded">
              if
            </span>{' '}
            Condition
          </span>
          <span>
            <span className="bg-red-100 text-red-800 font-bold underline decoration-wavy decoration-red-400 px-1 rounded">
              not
            </span>{' '}
            Negation
          </span>
        </div>
      )}
    </div>
  );
}
