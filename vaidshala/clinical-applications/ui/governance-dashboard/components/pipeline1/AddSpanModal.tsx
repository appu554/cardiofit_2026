'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Plus, Loader2 } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { useAuth } from '@/hooks/useAuth';
import { cn } from '@/lib/utils';
import type { AddSpanRequest } from '@/types/pipeline1';

// =============================================================================
// Props
// =============================================================================

interface AddSpanModalProps {
  jobId: string;
  /** Pre-fill the page number from the current active page */
  defaultPageNumber?: number;
  onClose: () => void;
  onSuccess: () => void;
}

// =============================================================================
// Component
// =============================================================================

export function AddSpanModal({
  jobId,
  defaultPageNumber,
  onClose,
  onSuccess,
}: AddSpanModalProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  const [text, setText] = useState('');
  const [startOffset, setStartOffset] = useState(0);
  const [endOffset, setEndOffset] = useState(0);
  const [pageNumber, setPageNumber] = useState<string>(
    defaultPageNumber != null ? String(defaultPageNumber) : '',
  );
  const [sectionId, setSectionId] = useState('');
  const [note, setNote] = useState('');
  const [error, setError] = useState<string | null>(null);

  const addMutation = useMutation({
    mutationFn: (req: AddSpanRequest) => pipeline1Api.spans.add(jobId, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
      queryClient.invalidateQueries({ queryKey: ['pipeline1-spans-all', jobId] });
      queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
      queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
      onSuccess();
      onClose();
    },
    onError: (err: unknown) => {
      setError((err as Error).message);
    },
  });

  const handleSubmit = () => {
    setError(null);

    if (!text.trim()) {
      setError('Span text is required.');
      return;
    }

    const req: AddSpanRequest = {
      text: text.trim(),
      startOffset,
      endOffset: endOffset || text.trim().length,
      reviewerId: user?.sub || 'unknown',
      ...(pageNumber ? { pageNumber: Number(pageNumber) } : {}),
      ...(sectionId.trim() ? { sectionId: sectionId.trim() } : {}),
      ...(note.trim() ? { note: note.trim() } : {}),
    };

    addMutation.mutate(req);
  };

  return (
    // Backdrop
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={onClose}
    >
      {/* Modal */}
      <div
        className="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-200">
          <div className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-purple-600" />
            <h2 className="text-lg font-semibold text-gray-900">Add Span</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="px-5 py-4 space-y-4">
          {/* Text */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Span Text <span className="text-red-500">*</span>
            </label>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={4}
              placeholder="Paste or type the text that the pipeline missed..."
              className="input w-full text-sm"
              autoFocus
            />
          </div>

          {/* Page + Section (side by side) */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Page Number</label>
              <input
                type="number"
                value={pageNumber}
                onChange={(e) => setPageNumber(e.target.value)}
                placeholder="e.g. 5"
                className="input w-full text-sm"
                min={1}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Section ID</label>
              <input
                type="text"
                value={sectionId}
                onChange={(e) => setSectionId(e.target.value)}
                placeholder="e.g. 4.2.1"
                className="input w-full text-sm"
              />
            </div>
          </div>

          {/* Offsets (collapsed — advanced, rarely needed) */}
          <details className="text-xs text-gray-500">
            <summary className="cursor-pointer hover:text-gray-700">Advanced: Text Offsets</summary>
            <div className="grid grid-cols-2 gap-3 mt-2">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">Start Offset</label>
                <input
                  type="number"
                  value={startOffset}
                  onChange={(e) => setStartOffset(Number(e.target.value))}
                  className="input w-full text-xs"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">End Offset</label>
                <input
                  type="number"
                  value={endOffset}
                  onChange={(e) => setEndOffset(Number(e.target.value))}
                  className="input w-full text-xs"
                />
              </div>
            </div>
          </details>

          {/* Note */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Note</label>
            <input
              type="text"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Why this span was added..."
              className="input w-full text-sm"
            />
          </div>

          {/* Error */}
          {error && (
            <div className="p-3 bg-red-50 border border-red-100 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-5 py-4 border-t border-gray-200">
          <button onClick={onClose} className="btn btn-outline">
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={addMutation.isPending}
            className={cn(
              'btn bg-purple-600 text-white hover:bg-purple-700 flex items-center gap-2',
              addMutation.isPending && 'opacity-50 cursor-not-allowed',
            )}
          >
            {addMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Add Span
          </button>
        </div>
      </div>
    </div>
  );
}
