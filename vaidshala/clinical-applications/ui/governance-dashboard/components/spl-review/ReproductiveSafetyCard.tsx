'use client';

import {
  Baby,
  AlertTriangle,
  ShieldCheck,
  Tag,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ReproductiveSafetyData } from '@/types/spl-review';

// ============================================================================
// ReproductiveSafetyCard — Type-specific for REPRODUCTIVE_SAFETY facts
//
// Shows: category (PREGNANCY/LACTATION/FERTILITY), risk level,
// FDA category, RID%, PLLR summary, population.
// ============================================================================

interface ReproductiveSafetyCardProps {
  data: ReproductiveSafetyData;
}

const CATEGORY_CONFIG: Record<ReproductiveSafetyData['category'], {
  color: string;
  bg: string;
  label: string;
}> = {
  PREGNANCY: { color: 'text-pink-700', bg: 'bg-pink-50', label: 'Pregnancy' },
  LACTATION: { color: 'text-purple-700', bg: 'bg-purple-50', label: 'Lactation' },
  FERTILITY: { color: 'text-blue-700', bg: 'bg-blue-50', label: 'Fertility' },
};

export function ReproductiveSafetyCard({ data }: ReproductiveSafetyCardProps) {
  const catConfig = CATEGORY_CONFIG[data.category] || CATEGORY_CONFIG.PREGNANCY;

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Baby className="h-4 w-4 text-pink-500 shrink-0" />
        <span className={cn('text-xs font-semibold px-2 py-0.5 rounded', catConfig.bg, catConfig.color)}>
          {catConfig.label}
        </span>
      </div>

      {/* Risk Level */}
      <div className="bg-gray-50 rounded-lg p-3">
        <span className="text-[10px] text-gray-400 uppercase font-semibold">Risk Level</span>
        <p className="text-sm font-semibold text-gray-900 mt-1">{data.riskLevel}</p>
      </div>

      {/* Details Grid */}
      <div className="grid grid-cols-2 gap-3">
        {data.fdaCategory && (
          <div>
            <span className="text-[10px] text-gray-400 uppercase">FDA Category</span>
            <p className="text-xs font-medium text-gray-900 mt-0.5">{data.fdaCategory}</p>
          </div>
        )}
        {data.ridPercent && (
          <div>
            <span className="text-[10px] text-gray-400 uppercase">RID%</span>
            <p className="text-xs font-medium text-gray-900 mt-0.5">{data.ridPercent}</p>
          </div>
        )}
        {data.population && (
          <div className="col-span-2">
            <span className="text-[10px] text-gray-400 uppercase">Population</span>
            <p className="text-xs text-gray-700 mt-0.5">{data.population}</p>
          </div>
        )}
      </div>

      {/* PLLR Summary */}
      {data.pllrSummary && (
        <div>
          <span className="text-[10px] text-gray-400 uppercase font-semibold">PLLR Summary</span>
          <p className="text-xs text-gray-700 mt-1 leading-relaxed bg-gray-50 rounded-lg p-3">
            {data.pllrSummary}
          </p>
        </div>
      )}
    </div>
  );
}
