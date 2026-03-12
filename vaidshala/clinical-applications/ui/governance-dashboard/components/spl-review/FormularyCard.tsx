'use client';

import {
  Package,
  Hash,
  Pill,
  Building2,
  Ruler,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { FormularyData } from '@/types/spl-review';

// ============================================================================
// FormularyCard — Type-specific for FORMULARY (How Supplied) facts
//
// Shows: NDC code, package form, strength, package size, manufacturer.
// ============================================================================

interface FormularyCardProps {
  data: FormularyData;
}

export function FormularyCard({ data }: FormularyCardProps) {
  const fields = [
    { icon: Hash, label: 'NDC Code', value: data.ndcCode },
    { icon: Pill, label: 'Package Form', value: data.packageForm },
    { icon: Ruler, label: 'Strength', value: data.strength },
    { icon: Package, label: 'Package Size', value: data.packageSize },
    { icon: Building2, label: 'Manufacturer', value: data.manufacturer },
  ].filter((f) => f.value);

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Package className="h-4 w-4 text-purple-500 shrink-0" />
        <h4 className="text-sm font-semibold text-gray-900">How Supplied</h4>
      </div>

      {/* Fields */}
      <div className="bg-gray-50 rounded-lg p-3 space-y-2.5">
        {fields.map((f) => {
          const Icon = f.icon;
          return (
            <div key={f.label} className="flex items-center gap-2.5">
              <Icon className="h-3.5 w-3.5 text-gray-400 shrink-0" />
              <div className="flex items-baseline gap-2 min-w-0">
                <span className="text-[10px] text-gray-400 uppercase shrink-0 w-20">{f.label}</span>
                <span className="text-xs text-gray-900 font-medium truncate">{f.value}</span>
              </div>
            </div>
          );
        })}
        {fields.length === 0 && (
          <p className="text-xs text-gray-400 italic">No product data extracted</p>
        )}
      </div>
    </div>
  );
}
