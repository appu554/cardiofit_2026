'use client';

import {
  Beaker,
  Clock,
  FileText,
  Ruler,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { LabReferenceData } from '@/types/spl-review';

// ============================================================================
// LabReferenceCard — Type-specific for LAB_REFERENCE facts
//
// Shows: lab test name, reference range, monitoring frequency,
// clinical context.
// ============================================================================

interface LabReferenceCardProps {
  data: LabReferenceData;
}

export function LabReferenceCard({ data }: LabReferenceCardProps) {
  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Beaker className="h-4 w-4 text-cyan-500 shrink-0" />
        <h4 className="text-base font-semibold text-gray-900">{data.labTest}</h4>
      </div>

      {/* Details */}
      <div className="bg-gray-50 rounded-lg p-3 space-y-2.5">
        {data.referenceRange && (
          <div className="flex items-start gap-2.5">
            <Ruler className="h-3.5 w-3.5 text-gray-400 mt-0.5 shrink-0" />
            <div>
              <span className="text-[10px] text-gray-400 uppercase">Reference Range</span>
              <p className="text-xs text-gray-900 font-medium mt-0.5">{data.referenceRange}</p>
            </div>
          </div>
        )}

        {data.monitoringFrequency && (
          <div className="flex items-start gap-2.5">
            <Clock className="h-3.5 w-3.5 text-gray-400 mt-0.5 shrink-0" />
            <div>
              <span className="text-[10px] text-gray-400 uppercase">Monitoring Frequency</span>
              <p className="text-xs text-gray-900 font-medium mt-0.5">{data.monitoringFrequency}</p>
            </div>
          </div>
        )}

        {data.clinicalContext && (
          <div className="flex items-start gap-2.5">
            <FileText className="h-3.5 w-3.5 text-gray-400 mt-0.5 shrink-0" />
            <div>
              <span className="text-[10px] text-gray-400 uppercase">Clinical Context</span>
              <p className="text-xs text-gray-700 mt-0.5 leading-relaxed">{data.clinicalContext}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
