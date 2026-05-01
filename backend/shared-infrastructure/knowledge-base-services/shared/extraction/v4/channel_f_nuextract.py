"""
Channel F: NuExtract Proposition Extractor via Ollama Sidecar (V4.1).

Uses NuExtract 2.0-8B (Qwen2.5 architecture) running as a native Ollama
service on Apple Silicon, NOT inside the Docker container.

V4.1 change: Channel F is an HTTP client, not a model host.

Why Ollama sidecar?
- Metal GPU acceleration on Apple Silicon (3-4 tok/s vs 0.3 tok/s on Docker CPU)
- GGUF Q4_K_M quantization (8B model in ~6GB VRAM)
- Native Qwen2.5 architecture support (no transformers compatibility issues)
- Process isolation (no RAM competition with extraction pipeline)

Operational contract (UNCHANGED from V4):
- Temperature: 0 (extraction, not generative)
- Invocation threshold: Prose blocks only, >15 words
- Block types processed: paragraph, list_item only (NOT heading, NOT table_cell)
- Passthrough: Elements under 15 words skip LLM processing

Deployment requirement:
- Mac Mini M4 with 24GB unified memory (MINIMUM)
- Ollama installed and running: `ollama serve`
- Model pulled: `ollama pull nuextract` (or custom GGUF import)

Pipeline Position:
    Channel A (GuidelineTree) -> Channel F (THIS, parallel with B-E)
    Requires: Channel A's GuidelineTree (for prose block identification)
"""

from __future__ import annotations

import json
import logging
import os
import re
import time
import unicodedata
from typing import Optional

from .models import ChannelOutput, GuidelineSection, GuidelineTree, RawSpan
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled

logger = logging.getLogger(__name__)


def _channel_f_model_version() -> str:
    """Channel F model version, pinned to NUEXTRACT_MODEL env if set."""
    env_model = os.environ.get("NUEXTRACT_MODEL")
    if env_model:
        return f"nuextract@{env_model}"
    try:
        return f"nuextract@{ChannelFNuExtract.VERSION}"
    except Exception:
        return "nuextract@v1.0"


def _channel_f_provenance(
    bbox,
    page_number,
    confidence,
    profile,
    notes: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel F (NuExtract via Ollama sidecar).

    Returns None when V5_BBOX_PROVENANCE is off or bbox is missing. Bbox is
    inherited from the parent prose passage. NuExtract does not expose
    per-proposition probabilities, so confidence is heuristic (0.85 default).
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="F",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_f_model_version(),
        notes=notes,
    )

# NuExtract extraction template — defines the JSON schema the model fills
EXTRACTION_TEMPLATE = json.dumps({
    "atomic_facts": [
        {
            "statement": "verbatim-string",
            "drug": "verbatim-string",
            "threshold": "verbatim-string",
            "action": "verbatim-string",
            "condition": "verbatim-string",
        }
    ]
})


class ChannelFNuExtract:
    """NuExtract 2.0-8B proposition extractor via Ollama sidecar.

    Also exported as ``ChannelF`` for pipeline convenience.

    V4.1: Model runs as a native Ollama service on Apple Silicon,
    NOT inside the Docker container. Channel F is an HTTP client.

    Decomposes complex clinical prose into atomic propositions.
    Falls back gracefully if Ollama is unreachable — this channel
    is supplementary to Channels B+C+D.
    """

    VERSION = "4.2.1"
    WORD_THRESHOLD = 15  # blocks under this skip LLM
    PROSE_BLOCK_TYPES = {"paragraph", "list_item"}

    # V4.2.1: NFKC unicode normalization — maps compatibility characters to
    # canonical forms (ligatures→components, superscripts→digits, fullwidth→ASCII).
    # Applied to all span text before storage to ensure consistent downstream matching.
    @staticmethod
    def _normalize_span_text(text: str) -> str:
        """Normalize span text to NFKC form.

        Handles exotic unicode from NuExtract/Ollama output:
        - Ligatures: ﬁ→fi, ﬂ→fl
        - Superscripts: ² → 2, ³ → 3
        - Fullwidth: ０→0, Ａ→A
        - Compatibility: ℃→°C, ℉→°F
        """
        return unicodedata.normalize('NFKC', text).strip()

    # V4.2.1: Artifact patterns to filter before prose extraction
    # Page breaks: {N}--- (Marker page-break markers)
    _PAGE_BREAK_RE = re.compile(r'^\{?\d+\}?\s*-{3,}\s*$')
    # Image references: ![alt](path) or ![](path)
    _IMAGE_REF_RE = re.compile(r'^!\[.*?\]\(.*?\)\s*$')
    # Header/footer lines: short lines with typical header/footer content
    _HEADER_FOOTER_RE = re.compile(
        r'^(?:www\.\S+|chapter\s+\d+|KDIGO\s+\d{4}|Kidney\s+International)'
        r'(?:\s+(?:www\.\S+|chapter\s+\d+|KDIGO\s+\d{4}|Kidney\s+International))*\s*$',
        re.IGNORECASE,
    )

    # Configurable via environment variables
    DEFAULT_OLLAMA_URL = "http://host.docker.internal:11434"  # Docker Desktop macOS
    DEFAULT_MODEL_NAME = "nuextract"

    def __init__(
        self,
        ollama_url: Optional[str] = None,
        model_name: Optional[str] = None,
    ) -> None:
        """Initialize Ollama connection (NOT model loading).

        Args:
            ollama_url: Ollama API endpoint.
                        Default from OLLAMA_URL env var or host.docker.internal:11434
            model_name: Ollama model name.
                        Default from NUEXTRACT_MODEL env var or "nuextract"
        """
        self.ollama_url = (
            ollama_url
            or os.environ.get("OLLAMA_URL", self.DEFAULT_OLLAMA_URL)
        )
        self.model_name = (
            model_name
            or os.environ.get("NUEXTRACT_MODEL", self.DEFAULT_MODEL_NAME)
        )
        self._available = False
        self._init_error: Optional[str] = None

        try:
            self._check_ollama()
            self._available = True
        except Exception as e:
            self._init_error = str(e)
            logger.warning("Channel F (NuExtract) unavailable: %s", e)

    # ═══════════════════════════════════════════════════════════════════════
    # OLLAMA CONNECTION (V4.1 NEW — replaces _load_model)
    # ═══════════════════════════════════════════════════════════════════════

    def _check_ollama(self) -> None:
        """Verify Ollama service is running and model is loaded.

        Checks:
        1. Ollama API is reachable (GET /api/tags)
        2. Target model is available in Ollama's model list
        """
        import requests

        try:
            response = requests.get(
                f"{self.ollama_url}/api/tags",
                timeout=5,
            )
            response.raise_for_status()

            models = response.json().get("models", [])
            model_names = [m.get("name", "").split(":")[0] for m in models]

            if self.model_name not in model_names:
                available = ", ".join(model_names) or "(none)"
                raise ConnectionError(
                    f"Model '{self.model_name}' not found in Ollama. "
                    f"Available: {available}. "
                    f"Pull with: ollama pull {self.model_name}"
                )
        except requests.ConnectionError:
            raise ConnectionError(
                f"Cannot reach Ollama at {self.ollama_url}. "
                "Ensure Ollama is running: `ollama serve`"
            )

    @property
    def available(self) -> bool:
        """Whether the Ollama service and NuExtract model are reachable."""
        return self._available

    # ═══════════════════════════════════════════════════════════════════════
    # EXTRACTION ENTRYPOINT (signature unchanged from V4)
    # ═══════════════════════════════════════════════════════════════════════

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
    ) -> ChannelOutput:
        """Extract atomic propositions from prose blocks.

        Only processes paragraph and list_item blocks with >15 words.
        Short blocks pass through as-is (already atomic).

        Args:
            text: Normalized text (Channel 0 output)
            tree: GuidelineTree from Channel A

        Returns:
            ChannelOutput with proposition RawSpans
        """
        start_time = time.monotonic()

        if not self._available:
            return ChannelOutput(
                channel="F",
                spans=[],
                error=f"NuExtract not available: {self._init_error}",
                elapsed_ms=0.0,
            )

        spans: list[RawSpan] = []
        blocks_processed = 0
        blocks_passthrough = 0
        blocks_failed = 0

        # Collect all leaf sections for processing
        all_sections = self._collect_all_sections(tree.sections)

        for section in all_sections:
            section_text = text[section.start_offset:section.end_offset]

            # Extract prose lines (skip headings and table lines)
            prose_blocks = self._extract_prose_blocks(
                section_text, section.start_offset, tree
            )

            for block_text, block_start, block_end in prose_blocks:
                word_count = len(block_text.split())

                if word_count <= self.WORD_THRESHOLD:
                    # Passthrough: already atomic, no LLM needed
                    if block_text.strip():
                        spans.append(RawSpan(
                            channel="F",
                            text=self._normalize_span_text(block_text),
                            start=block_start,
                            end=block_end,
                            confidence=0.90,
                            page_number=tree.get_page_for_offset(block_start),
                            section_id=section.section_id,
                            source_block_type="paragraph",
                            channel_metadata={
                                "extraction_method": "passthrough",
                                "word_count": word_count,
                            },
                        ))
                    blocks_passthrough += 1
                else:
                    # NuExtract: decompose into atomic propositions
                    propositions = self._extract_propositions(block_text)

                    if propositions is None:
                        # Inference failure — skip block, don't lose data
                        blocks_failed += 1
                        continue

                    for prop in propositions:
                        statement = prop.get("statement", "")
                        if not statement:
                            continue

                        # V4.2.1: NFKC normalize LLM output
                        statement = self._normalize_span_text(statement)
                        if not statement:
                            continue

                        # Find the statement in the original text
                        prop_start, prop_end = self._find_span_offset(
                            text, statement, block_start, block_end
                        )

                        spans.append(RawSpan(
                            channel="F",
                            text=statement,
                            start=prop_start,
                            end=prop_end,
                            confidence=0.85,
                            page_number=tree.get_page_for_offset(prop_start),
                            section_id=section.section_id,
                            source_block_type="paragraph",
                            channel_metadata={
                                "extraction_method": "nuextract_ollama",
                                "drug": prop.get("drug"),
                                "threshold": prop.get("threshold"),
                                "action": prop.get("action"),
                                "condition": prop.get("condition"),
                            },
                        ))
                    blocks_processed += 1

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel="F",
            spans=spans,
            metadata={
                "blocks_processed_by_llm": blocks_processed,
                "blocks_passthrough": blocks_passthrough,
                "blocks_failed": blocks_failed,
                "total_propositions": len(spans),
                "ollama_url": self.ollama_url,
                "model_name": self.model_name,
            },
            elapsed_ms=elapsed_ms,
        )

    # ═══════════════════════════════════════════════════════════════════════
    # LLM INFERENCE via Ollama (V4.1 NEW — replaces _run_inference)
    # ═══════════════════════════════════════════════════════════════════════

    def _extract_propositions(self, block_text: str) -> Optional[list[dict]]:
        """Run NuExtract on a prose block to extract atomic propositions.

        Returns:
            list of dicts with keys: statement, drug, threshold, action, condition.
            None if inference failed (network error, timeout).
        """
        try:
            prompt = self._build_prompt(block_text)
            response = self._run_inference(prompt)
            return self._parse_response(response)
        except Exception as e:
            logger.warning("NuExtract inference failed: %s", e)
            return None

    def _build_prompt(self, text: str) -> str:
        """Build NuExtract prompt with extraction template.

        Format follows numind's NuExtract prompt convention:
        <|input|> / <|template|> / <|output|> delimiters.
        """
        return (
            f"<|input|>\n{text}\n"
            f"<|template|>\n{EXTRACTION_TEMPLATE}\n"
            f"<|output|>\n"
        )

    def _run_inference(self, prompt: str) -> str:
        """Run NuExtract inference via Ollama HTTP API.

        Uses temperature=0 for deterministic clinical extraction.
        Context size 8192 for long KDIGO prose blocks.
        Stream disabled for simpler response handling.
        """
        import requests

        response = requests.post(
            f"{self.ollama_url}/api/generate",
            json={
                "model": self.model_name,
                "prompt": prompt,
                "temperature": 0,
                "stream": False,
                "options": {
                    "num_ctx": 8192,
                    "num_predict": 1024,
                },
            },
            timeout=300,  # 5 min per block (generous for CPU fallback)
        )
        response.raise_for_status()
        return response.json().get("response", "")

    # ═══════════════════════════════════════════════════════════════════════
    # RESPONSE PARSING (unchanged from V4)
    # ═══════════════════════════════════════════════════════════════════════

    def _parse_response(self, response: str) -> list[dict]:
        """Parse NuExtract JSON response, handling truncated output.

        NuExtract outputs JSON matching the EXTRACTION_TEMPLATE schema.
        Truncated responses are common with long inputs — we attempt recovery.
        """
        # Try to find a complete JSON object in the response
        try:
            json_match = re.search(r'\{[\s\S]*\}', response)
            if json_match:
                data = json.loads(json_match.group())
                facts = data.get("atomic_facts", [])
                if isinstance(facts, list):
                    return facts
        except json.JSONDecodeError:
            pass

        # Try to recover truncated JSON by adding closing brackets
        try:
            cleaned = response.strip()
            if cleaned and not cleaned.endswith('}'):
                cleaned += '"}]}'
            data = json.loads(cleaned)
            facts = data.get("atomic_facts", [])
            if isinstance(facts, list):
                return facts
        except (json.JSONDecodeError, TypeError):
            pass

        return []

    # ═══════════════════════════════════════════════════════════════════════
    # PROSE BLOCK EXTRACTION (unchanged from V4)
    # ═══════════════════════════════════════════════════════════════════════

    def _is_artifact_line(self, stripped: str) -> bool:
        """Check if a line is a non-clinical artifact that should be filtered.

        V4.2.1: Detects page breaks, image references, and header/footer lines
        that leak through Marker's normalization. These produce ~9/32 noise spans
        in Channel F output.
        """
        if self._PAGE_BREAK_RE.match(stripped):
            return True
        if self._IMAGE_REF_RE.match(stripped):
            return True
        if self._HEADER_FOOTER_RE.match(stripped):
            return True
        return False

    def _extract_prose_blocks(
        self, section_text: str, base_offset: int, tree: GuidelineTree
    ) -> list[tuple[str, int, int]]:
        """Extract prose blocks from section text, skipping non-prose content.

        Segments section text into contiguous prose paragraphs by detecting
        and splitting on headings (#), table lines (|...|), blank lines,
        and V4.2.1 artifact lines (page breaks, image refs, headers/footers).
        """
        blocks: list[tuple[str, int, int]] = []
        lines = section_text.split('\n')
        offset = base_offset

        current_block: list[str] = []
        block_start = offset

        for line in lines:
            stripped = line.strip()
            line_end = offset + len(line)

            # Skip headings
            if stripped.startswith('#'):
                if current_block:
                    block_text = '\n'.join(current_block)
                    if block_text.strip():
                        blocks.append((block_text.strip(), block_start, offset))
                    current_block = []
                offset = line_end + 1
                block_start = offset
                continue

            # Skip table lines
            if stripped.startswith('|') and stripped.endswith('|'):
                if current_block:
                    block_text = '\n'.join(current_block)
                    if block_text.strip():
                        blocks.append((block_text.strip(), block_start, offset))
                    current_block = []
                offset = line_end + 1
                block_start = offset
                continue

            # V4.2.1: Skip artifact lines (page breaks, images, headers/footers)
            if stripped and self._is_artifact_line(stripped):
                if current_block:
                    block_text = '\n'.join(current_block)
                    if block_text.strip():
                        blocks.append((block_text.strip(), block_start, offset))
                    current_block = []
                offset = line_end + 1
                block_start = offset
                continue

            # Blank line = paragraph boundary
            if not stripped:
                if current_block:
                    block_text = '\n'.join(current_block)
                    if block_text.strip():
                        blocks.append((block_text.strip(), block_start, offset))
                    current_block = []
                offset = line_end + 1
                block_start = offset
                continue

            if not current_block:
                block_start = offset
            current_block.append(line)
            offset = line_end + 1

        # Flush remaining block
        if current_block:
            block_text = '\n'.join(current_block)
            if block_text.strip():
                blocks.append((block_text.strip(), block_start, offset))

        return blocks

    # ═══════════════════════════════════════════════════════════════════════
    # OFFSET RESOLUTION (unchanged from V4)
    # ═══════════════════════════════════════════════════════════════════════

    def _find_span_offset(
        self, full_text: str, span_text: str, search_start: int, search_end: int
    ) -> tuple[int, int]:
        """Find the offset of a proposition's text in the full document.

        Search strategy:
        1. Exact match within block range
        2. Case-insensitive match within block range
        3. Fallback: use block boundaries
        """
        # Exact match within the block's range
        idx = full_text.find(span_text, search_start, search_end)
        if idx >= 0:
            return idx, idx + len(span_text)

        # Case-insensitive search in block range
        lower_text = full_text[search_start:search_end].lower()
        lower_span = span_text.lower()
        idx = lower_text.find(lower_span)
        if idx >= 0:
            return search_start + idx, search_start + idx + len(span_text)

        # Last resort: use block boundaries
        return search_start, min(search_start + len(span_text), search_end)

    # ═══════════════════════════════════════════════════════════════════════
    # SECTION TRAVERSAL (unchanged from V4)
    # ═══════════════════════════════════════════════════════════════════════

    def _collect_all_sections(
        self, sections: list[GuidelineSection]
    ) -> list[GuidelineSection]:
        """Collect all leaf sections (no children) for processing.

        Leaf sections contain actual text content. Parent sections
        just define structure — their text is covered by children.
        """
        result: list[GuidelineSection] = []
        for section in sections:
            if section.children:
                result.extend(self._collect_all_sections(section.children))
            else:
                result.append(section)
        return result


# Short alias for pipeline imports
ChannelF = ChannelFNuExtract
