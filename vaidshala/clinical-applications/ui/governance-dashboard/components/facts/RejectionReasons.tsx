'use client';

import { REJECTION_REASONS, type RejectionReasonCode } from '@/types/governance';

interface RejectionReasonsProps {
  selected: RejectionReasonCode[];
  onSelectionChange: (codes: RejectionReasonCode[]) => void;
  otherText: string;
  onOtherTextChange: (text: string) => void;
}

export function RejectionReasons({
  selected,
  onSelectionChange,
  otherText,
  onOtherTextChange,
}: RejectionReasonsProps) {
  const toggle = (code: RejectionReasonCode) => {
    if (selected.includes(code)) {
      onSelectionChange(selected.filter((c) => c !== code));
    } else {
      onSelectionChange([...selected, code]);
    }
  };

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-gray-700">Rejection Reason(s) <span className="text-red-500">*</span></p>
      {REJECTION_REASONS.map((reason) => (
        <label
          key={reason.code}
          className="flex items-start space-x-3 p-2 rounded hover:bg-gray-50 cursor-pointer"
        >
          <input
            type="checkbox"
            checked={selected.includes(reason.code)}
            onChange={() => toggle(reason.code)}
            className="mt-0.5 h-4 w-4 rounded border-gray-300 text-red-600 focus:ring-red-500"
          />
          <div>
            <span className="text-sm font-medium text-gray-800">{reason.label}</span>
            <p className="text-xs text-gray-500">{reason.description}</p>
          </div>
        </label>
      ))}
      {selected.includes('OTHER') && (
        <textarea
          value={otherText}
          onChange={(e) => onOtherTextChange(e.target.value)}
          rows={2}
          className="input w-full mt-1"
          placeholder="Describe the rejection reason..."
        />
      )}
    </div>
  );
}
