'use client';

import { useState, useCallback, useEffect } from 'react';
import { X, Maximize2, Minimize2, ExternalLink } from 'lucide-react';
import { cn } from '@/lib/utils';

interface DailyMedViewerProps {
  setId: string;
  drugName: string;
  sectionLoinc?: string;
  onClose: () => void;
}

export function DailyMedViewer({ setId, drugName, sectionLoinc, onClose }: DailyMedViewerProps) {
  const [fullscreen, setFullscreen] = useState(false);
  const [loading, setLoading] = useState(true);

  const url = `https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=${setId}`;

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    },
    [onClose]
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [handleKeyDown]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* Modal */}
      <div
        className={cn(
          'relative bg-white rounded-xl shadow-2xl flex flex-col transition-all duration-200',
          fullscreen ? 'w-screen h-screen rounded-none' : 'w-[90vw] h-[85vh] max-w-6xl'
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-gray-50 rounded-t-xl shrink-0">
          <div>
            <h3 className="font-semibold text-gray-900">
              FDA Drug Label — {drugName}
            </h3>
            {sectionLoinc && (
              <p className="text-xs text-gray-500 mt-0.5">
                Section LOINC: <code className="bg-gray-200 px-1 rounded">{sectionLoinc}</code>
              </p>
            )}
          </div>
          <div className="flex items-center space-x-2">
            <a
              href={url}
              target="_blank"
              rel="noopener noreferrer"
              className="p-1.5 rounded hover:bg-gray-200 text-gray-500 hover:text-gray-700 transition-colors"
              title="Open in new tab"
            >
              <ExternalLink className="h-4 w-4" />
            </a>
            <button
              onClick={() => setFullscreen(!fullscreen)}
              className="p-1.5 rounded hover:bg-gray-200 text-gray-500 hover:text-gray-700 transition-colors"
              title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
            >
              {fullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            </button>
            <button
              onClick={onClose}
              className="p-1.5 rounded hover:bg-red-100 text-gray-500 hover:text-red-600 transition-colors"
              title="Close (Esc)"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Iframe */}
        <div className="flex-1 relative">
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center bg-white">
              <div className="text-center">
                <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto" />
                <p className="text-sm text-gray-500 mt-3">Loading DailyMed label…</p>
              </div>
            </div>
          )}
          <iframe
            src={url}
            className="w-full h-full border-0"
            onLoad={() => setLoading(false)}
            title={`DailyMed label for ${drugName}`}
            sandbox="allow-same-origin allow-scripts allow-popups"
          />
        </div>
      </div>
    </div>
  );
}
