-- Migration 004: Add pipeline highlight HTML + source PDF path
-- These enable the reviewer UI to show the original PDF and channel-colored
-- HTML highlights as reference views alongside the span-overlay workspace.

-- Store the self-contained HTML highlight output (70K per job, small enough for TEXT)
ALTER TABLE l2_guideline_tree ADD COLUMN IF NOT EXISTS highlight_html TEXT;

-- Store the absolute filesystem path to the source PDF pages
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS source_pdf_path TEXT;
