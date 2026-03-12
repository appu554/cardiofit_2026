'use client';

import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { SPLFactReviewLayout } from '@/components/spl-review/SPLFactReviewLayout';
import type { SPLFactType, GovernanceStatus } from '@/types/spl-review';

// ============================================================================
// SPL Fact Review Page — /spl-review/[drug]/review
//
// Split-pane review experience for a single drug's facts.
// Supports query parameters:
//   ?type=SAFETY_SIGNAL — filter to specific fact type
//   ?status=PENDING_REVIEW — filter to specific governance status
// ============================================================================

export default function SPLReviewPage() {
  const params = useParams<{ drug: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();

  const drugName = decodeURIComponent(params.drug);
  const typeParam = searchParams.get('type') as SPLFactType | null;
  const statusParam = searchParams.get('status') as GovernanceStatus | null;

  return (
    <SPLFactReviewLayout
      drugName={drugName}
      initialFactType={typeParam || undefined}
      initialStatus={statusParam || undefined}
      onBack={() => router.push(`/spl-review/${encodeURIComponent(drugName)}`)}
    />
  );
}
