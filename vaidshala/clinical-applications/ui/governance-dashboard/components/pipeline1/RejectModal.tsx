'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { RejectReason } from '@/types/pipeline1';
import { REJECT_REASON_LABELS } from '@/types/pipeline1';

interface RejectModalProps {
  onClose: () => void;
  onConfirm: (reason: RejectReason) => void;
  isLoading?: boolean;
}

export function RejectModal({ onClose, onConfirm, isLoading }: RejectModalProps) {
  const [reason, setReason] = useState<RejectReason | null>(null);

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[100]">
      <div className="bg-white rounded-xl p-7 w-[380px] shadow-2xl">
        <div className="text-[15px] font-semibold text-gray-900 mb-4">
          Reject — Select Reason
        </div>
        <div className="flex flex-col gap-1.5 mb-5">
          {(Object.entries(REJECT_REASON_LABELS) as [RejectReason, string][]).map(
            ([key, label]) => (
              <label
                key={key}
                className={cn(
                  'flex items-center gap-2.5 px-3 py-2 rounded-md cursor-pointer transition-colors text-[13px]',
                  reason === key
                    ? 'bg-red-50 border border-red-200 text-gray-900'
                    : 'bg-gray-50 border border-gray-200 text-gray-700 hover:bg-gray-100',
                )}
              >
                <input
                  type="radio"
                  name="rejectReason"
                  value={key}
                  checked={reason === key}
                  onChange={() => setReason(key)}
                  className="accent-red-600"
                />
                {label}
              </label>
            ),
          )}
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="px-4 py-2 rounded-md border border-gray-200 bg-white text-xs text-gray-500 hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => {
              if (reason) onConfirm(reason);
            }}
            disabled={!reason || isLoading}
            className={cn(
              'px-4 py-2 rounded-md border-none text-white text-xs font-semibold transition-colors flex items-center gap-1.5',
              reason && !isLoading
                ? 'bg-red-600 hover:bg-red-700 cursor-pointer'
                : 'bg-gray-300 cursor-not-allowed',
            )}
          >
            {isLoading && <Loader2 className="h-3 w-3 animate-spin" />}
            Confirm Reject
          </button>
        </div>
      </div>
    </div>
  );
}
