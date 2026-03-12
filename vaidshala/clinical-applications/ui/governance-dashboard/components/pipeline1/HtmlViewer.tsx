'use client';

import { pipeline1Api } from '@/lib/pipeline1-api';

interface HtmlViewerProps {
  jobId: string;
}

/**
 * Renders the pipeline-generated channel-colored highlight HTML in an iframe.
 * The HTML is self-contained (embedded CSS + vanilla JS tooltips) with a dark
 * GitHub-inspired theme — the iframe provides full style isolation from Tailwind.
 */
export function HtmlViewer({ jobId }: HtmlViewerProps) {
  const src = pipeline1Api.context.getHighlightHtmlUrl(jobId);

  return (
    <iframe
      src={src}
      title="Pipeline Highlight HTML"
      className="w-full h-full border-0"
      sandbox="allow-scripts"
    />
  );
}
