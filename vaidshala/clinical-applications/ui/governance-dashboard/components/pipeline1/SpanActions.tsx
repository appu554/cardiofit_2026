'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle,
  XCircle,
  Pencil,
  Loader2,
  AlertTriangle,
} from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import type { MergedSpan, SpanReviewRequest } from '@/types/pipeline1';

interface SpanActionsProps {
  span: MergedSpan | null;
  jobId: string;
  onActionComplete: () => void;
}

type ActionType = 'confirm' | 'reject' | 'edit' | null;

export function SpanActions({ span, jobId, onActionComplete }: SpanActionsProps) {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [activeAction, setActiveAction] = useState<ActionType>(null);
  const [note, setNote] = useState('');
  const [editedText, setEditedText] = useState('');
  const [error, setError] = useState<string | null>(null);

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ['pipeline1-spans', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-metrics', jobId] });
    queryClient.invalidateQueries({ queryKey: ['pipeline1-jobs'] });
  };

  const confirmMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.confirm(jobId, span!.id, req),
    onSuccess: () => { invalidateAll(); handleReset(); onActionComplete(); },
    onError: (err) => setError((err as Error).message),
  });

  const rejectMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.reject(jobId, span!.id, req),
    onSuccess: () => { invalidateAll(); handleReset(); onActionComplete(); },
    onError: (err) => setError((err as Error).message),
  });

  const editMutation = useMutation({
    mutationFn: (req: SpanReviewRequest) => pipeline1Api.spans.edit(jobId, span!.id, req),
    onSuccess: () => { invalidateAll(); handleReset(); onActionComplete(); },
    onError: (err) => setError((err as Error).message),
  });

  const isLoading = confirmMutation.isPending || rejectMutation.isPending || editMutation.isPending;

  const handleReset = () => {
    setActiveAction(null);
    setNote('');
    setEditedText('');
    setError(null);
  };

  const handleSubmit = () => {
    if (!span) return;
    const reviewerId = user?.sub || 'unknown';
    const req: SpanReviewRequest = {
      reviewerId,
      ...(note ? { note } : {}),
    };

    switch (activeAction) {
      case 'confirm':
        confirmMutation.mutate(req);
        break;
      case 'reject':
        if (!note.trim()) { setError('Note required for rejection'); return; }
        rejectMutation.mutate(req);
        break;
      case 'edit':
        if (!editedText.trim()) { setError('Edited text required'); return; }
        editMutation.mutate({ ...req, editedText });
        break;
    }
  };

  // No span selected
  if (!span) {
    return (
      <div className="card h-full">
        <div className="card-header">
          <h2 className="text-lg font-semibold text-gray-900">Decision Controls</h2>
        </div>
        <div className="card-body flex items-center justify-center py-16 text-gray-400">
          <p className="text-sm">Select a span to review</p>
        </div>
      </div>
    );
  }

  return (
    <div className="card h-full">
      <div className="card-header">
        <h2 className="text-lg font-semibold text-gray-900">Decision Controls</h2>
      </div>

      <div className="card-body">
        {/* Span info */}
        <div className="mb-4 p-3 bg-gray-50 rounded-lg">
          <p className="text-sm text-gray-800 line-clamp-3">{span.text}</p>
          <div className="flex gap-2 mt-2 text-xs text-gray-500">
            <span>Confidence: {(span.mergedConfidence * 100).toFixed(0)}%</span>
            {span.hasDisagreement && (
              <span className="text-amber-600 flex items-center">
                <AlertTriangle className="h-3 w-3 mr-0.5" />
                Disagreement
              </span>
            )}
          </div>
          {span.contributingChannels.length > 0 && (
            <div className="flex gap-1 mt-1.5">
              {span.contributingChannels.map((ch) => (
                <span key={ch} className="text-xs bg-gray-200 px-1.5 py-0.5 rounded">{ch}</span>
              ))}
            </div>
          )}
        </div>

        {!activeAction ? (
          /* Action buttons */
          <div className="space-y-3">
            <button
              onClick={() => setActiveAction('confirm')}
              className="w-full btn bg-green-600 text-white hover:bg-green-700 flex items-center justify-center"
            >
              <CheckCircle className="h-5 w-5 mr-2" />
              Confirm
            </button>
            <button
              onClick={() => { setActiveAction('edit'); setEditedText(span.text); }}
              className="w-full btn bg-blue-600 text-white hover:bg-blue-700 flex items-center justify-center"
            >
              <Pencil className="h-5 w-5 mr-2" />
              Edit
            </button>
            <button
              onClick={() => setActiveAction('reject')}
              className="w-full btn bg-red-600 text-white hover:bg-red-700 flex items-center justify-center"
            >
              <XCircle className="h-5 w-5 mr-2" />
              Reject
            </button>
          </div>
        ) : (
          /* Active action form */
          <div className="space-y-4">
            <div
              className={cn(
                'p-3 rounded-lg text-sm font-medium flex items-center',
                activeAction === 'confirm' && 'bg-green-50 text-green-700',
                activeAction === 'edit'    && 'bg-blue-50 text-blue-700',
                activeAction === 'reject'  && 'bg-red-50 text-red-700',
              )}
            >
              {activeAction === 'confirm' && <><CheckCircle className="h-5 w-5 mr-2" />Confirming Span</>}
              {activeAction === 'edit'    && <><Pencil className="h-5 w-5 mr-2" />Editing Span</>}
              {activeAction === 'reject'  && <><XCircle className="h-5 w-5 mr-2" />Rejecting Span</>}
            </div>

            {/* Edit textarea */}
            {activeAction === 'edit' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Corrected Text <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={editedText}
                  onChange={(e) => setEditedText(e.target.value)}
                  rows={4}
                  className="input w-full text-sm"
                />
              </div>
            )}

            {/* Note */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Note {activeAction === 'reject' && <span className="text-red-500">*</span>}
              </label>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={2}
                className="input w-full text-sm"
                placeholder={activeAction === 'reject' ? 'Reason for rejection...' : 'Optional note...'}
              />
            </div>

            {/* Error */}
            {error && (
              <div className="p-3 bg-red-50 border border-red-100 rounded-lg text-red-700 text-sm">{error}</div>
            )}

            {/* Submit / Cancel */}
            <div className="flex space-x-3">
              <button onClick={handleReset} disabled={isLoading} className="flex-1 btn btn-outline">
                Cancel
              </button>
              <button
                onClick={handleSubmit}
                disabled={isLoading}
                className={cn(
                  'flex-1 btn flex items-center justify-center',
                  activeAction === 'confirm' && 'bg-green-600 text-white hover:bg-green-700',
                  activeAction === 'edit'    && 'bg-blue-600 text-white hover:bg-blue-700',
                  activeAction === 'reject'  && 'bg-red-600 text-white hover:bg-red-700',
                  isLoading && 'opacity-50 cursor-not-allowed',
                )}
              >
                {isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Confirm
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
