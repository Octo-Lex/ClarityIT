"""
Document Assist Gateway for ClarityIT Reasoning Worker.

Provides text generation for agent-assisted document editing.
This module generates suggestion blocks only — it does NOT:
- Access the database
- Access MinIO
- Access NATS or Redis
- Call the Tool Gateway
- Mutate any persistent state

The Go API remains the validator and policy boundary.
"""

from __future__ import annotations

import json
import re
from typing import Any


# ─── Modes ───

ASSIST_MODES = {
    "rewrite",
    "summarize",
    "expand",
    "make_concise",
    "make_executive",
    "make_technical",
    "draft_section",
    "create_outline",
    "extract_action_items",
    "improve_headings",
}

FORBIDDEN_FIELDS = {"chain_of_thought", "chain-of-thought", "thinking", "thought_process", "internal_reasoning"}


def validate_assist_mode(mode: str) -> list[str]:
    """Validate the assist mode."""
    errors: list[str] = []
    if mode not in ASSIST_MODES:
        errors.append(f"invalid mode '{mode}' — must be one of: {', '.join(sorted(ASSIST_MODES))}")
    return errors


def validate_assist_response(raw: dict[str, Any]) -> list[str]:
    """Validate the response has no forbidden fields."""
    errors: list[str] = []
    for key in raw:
        key_lower = key.lower().replace("-", "_")
        if key_lower in FORBIDDEN_FIELDS or key.lower() in FORBIDDEN_FIELDS:
            errors.append(f"forbidden field '{key}' — chain-of-thought is rejected")
    return errors


class StubDocumentAssistGateway:
    """
    Stub document assist gateway for testing.

    Returns deterministic suggestions based on the mode.
    Never calls a real LLM.
    """

    def generate_assist(
        self,
        mode: str,
        selected_text: str,
        instruction: str,
        document_type: str,
        max_words: int,
    ) -> dict[str, Any]:
        """Generate suggestion blocks for the given mode."""

        text = selected_text.strip()
        if not text and mode not in ("draft_section", "create_outline"):
            text = "(empty)"

        if mode == "rewrite":
            rewritten = f"[Rewritten] {text}" if len(text) > 100 else f"[Rewritten for clarity] {text}"
            return {
                "suggested_blocks": [{"type": "paragraph", "text": rewritten[:max_words * 6]}],
                "summary": "Rewrote selected text for improved clarity.",
            }

        if mode == "summarize":
            # Simple word-based summary
            words = text.split()
            if len(words) <= 10:
                summary_text = text
            else:
                summary_text = " ".join(words[:10]) + "..."
            return {
                "suggested_blocks": [{"type": "paragraph", "text": f"Summary: {summary_text}"}],
                "summary": "Summarized the selected text into a concise overview.",
            }

        if mode == "expand":
            expanded = f"{text}\n\nAdditional context: This section provides further detail on the topics mentioned above, including relevant background information and supporting evidence."
            return {
                "suggested_blocks": [{"type": "paragraph", "text": expanded[:max_words * 6]}],
                "summary": "Expanded the selected text with additional context.",
            }

        if mode == "make_concise":
            words = text.split()
            half = max(5, len(words) // 2)
            concise = " ".join(words[:half])
            return {
                "suggested_blocks": [{"type": "paragraph", "text": concise}],
                "summary": "Made the text more concise.",
            }

        if mode == "make_executive":
            exec_text = f"**Executive Summary:** {text}" if not text.startswith("**") else text
            return {
                "suggested_blocks": [{"type": "paragraph", "text": exec_text[:max_words * 6]}],
                "summary": "Reframed the text for executive audience.",
            }

        if mode == "make_technical":
            tech_text = f"**Technical Details:** {text}\n\nKey implementation considerations and technical specifications apply."
            return {
                "suggested_blocks": [{"type": "paragraph", "text": tech_text[:max_words * 6]}],
                "summary": "Added technical depth to the text.",
            }

        if mode == "draft_section":
            section_title = instruction[:60] if instruction else "Drafted Section"
            return {
                "suggested_blocks": [
                    {"type": "heading", "level": 2, "text": section_title},
                    {"type": "paragraph", "text": f"This section covers: {instruction or 'the requested topic'}. Detailed content should be added here based on the specific requirements and context."},
                ],
                "summary": f"Drafted a new section: {section_title}",
            }

        if mode == "create_outline":
            sections = ["Overview", "Scope", "Architecture", "Implementation", "Risks", "Timeline", "Next Steps"]
            blocks = [{"type": "heading", "level": 1, "text": "Document Outline"}]
            for s in sections:
                blocks.append({"type": "bullets", "items": [s]})
            return {
                "suggested_blocks": blocks,
                "summary": "Generated a standard document outline with 7 sections.",
            }

        if mode == "extract_action_items":
            # Find lines that look like action items
            lines = text.split("\n")
            items = [line.strip().lstrip("- ").lstrip("* ") for line in lines if line.strip() and any(kw in line.lower() for kw in ["action", "todo", "task", "follow", "should", "must", "will", "need"])]
            if not items:
                items = ["Review and assign action items from the text above"]
            return {
                "suggested_blocks": [{"type": "bullets", "items": items[:20]}],
                "summary": f"Extracted {len(items)} action item(s).",
            }

        if mode == "improve_headings":
            lines = text.split("\n")
            headings = [line.lstrip("# ").strip() for line in lines if line.strip().startswith("#")]
            if not headings:
                headings = ["Overview", "Details", "Summary"]
            blocks = [{"type": "heading", "level": i + 1, "text": f"Improved: {h}"} for i, h in enumerate(headings[:6])]
            return {
                "suggested_blocks": blocks,
                "summary": f"Improved {len(blocks)} heading(s).",
            }

        # Fallback (should never reach here due to validation)
        return {
            "suggested_blocks": [{"type": "paragraph", "text": text}],
            "summary": "Returned original text.",
        }
