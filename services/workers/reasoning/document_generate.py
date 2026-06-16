"""
Document Generation Gateway for ClarityIT Reasoning Worker.

Generates complete native ClarityDocs documents from a user prompt.
This module generates structured document_json only — it does NOT:
- Access the database
- Access MinIO
- Access NATS or Redis
- Call the Tool Gateway
- Mutate any persistent state

The Go API remains the validator and policy boundary.
"""

from __future__ import annotations

from typing import Any


# ─── Tones ───

GENERATE_TONES = {
    "technical",
    "executive",
    "casual",
    "formal",
}

# ─── Document types (must match Go-side validDocumentTypes) ───

GENERATE_DOC_TYPES = {
    "general_document",
    "decision_memo",
    "implementation_plan",
    "incident_summary",
    "training_doc",
    "architecture_doc",
    "project_report",
    "status_report",
    "meeting_summary",
    "executive_brief",
}

FORBIDDEN_FIELDS = {"chain_of_thought", "chain-of-thought", "thinking", "thought_process", "internal_reasoning"}


def validate_generate_request(payload: dict[str, Any]) -> list[str]:
    """Validate the document generation request."""
    errors: list[str] = []

    title = payload.get("title", "")
    if not title or not title.strip():
        errors.append("title is required")
    elif len(title) > 200:
        errors.append("title exceeds 200 chars")

    doc_type = payload.get("document_type", "")
    if doc_type not in GENERATE_DOC_TYPES:
        errors.append(f"invalid document_type '{doc_type}'")

    prompt = payload.get("prompt", "")
    if not prompt or not prompt.strip():
        errors.append("prompt is required")
    elif len(prompt) > 2000:
        errors.append("prompt exceeds 2000 chars")

    tone = payload.get("tone", "technical")
    if tone not in GENERATE_TONES:
        errors.append(f"invalid tone '{tone}'")

    sections = payload.get("sections", [])
    if len(sections) > 20:
        errors.append("sections exceeds maximum of 20")
    for i, s in enumerate(sections):
        if not isinstance(s, str) or not s.strip():
            errors.append(f"section {i} is empty")
        elif len(s) > 100:
            errors.append(f"section {i} exceeds 100 chars")

    return errors


def validate_generate_response(raw: dict[str, Any]) -> list[str]:
    """Validate the response has no forbidden fields."""
    errors: list[str] = []
    for key in raw:
        key_lower = key.lower().replace("-", "_")
        if key_lower in FORBIDDEN_FIELDS or key.lower() in FORBIDDEN_FIELDS:
            errors.append(f"forbidden field '{key}' — chain-of-thought is rejected")
    # Check nested document_json for forbidden fields
    doc_json = raw.get("document_json", {})
    if isinstance(doc_json, dict):
        for key in doc_json:
            key_lower = key.lower().replace("-", "_")
            if key_lower in FORBIDDEN_FIELDS or key.lower() in FORBIDDEN_FIELDS:
                errors.append(f"forbidden field 'document_json.{key}' — chain-of-thought is rejected")
    return errors


class StubDocumentGenerateGateway:
    """
    Stub document generation gateway for testing.

    Generates deterministic structured documents based on the prompt,
    document type, tone, and requested sections.
    Never calls a real LLM.
    """

    def generate_document(
        self,
        title: str,
        document_type: str,
        prompt: str,
        tone: str = "technical",
        sections: list[str] | None = None,
    ) -> dict[str, Any]:
        """Generate a complete document_json from the given parameters."""

        if sections is None or len(sections) == 0:
            sections = self._default_sections(document_type)

        blocks: list[dict[str, Any]] = []

        # Document title heading
        blocks.append({
            "id": "blk_001",
            "type": "heading",
            "level": 1,
            "text": title,
        })

        # Generate one heading + paragraph per section
        for i, section_name in enumerate(sections):
            block_idx = (i + 1) * 10
            blocks.append({
                "id": f"blk_{block_idx + 1:03d}",
                "type": "heading",
                "level": 2,
                "text": section_name,
            })

            para_text = self._section_content(section_name, document_type, tone, prompt)
            blocks.append({
                "id": f"blk_{block_idx + 2:03d}",
                "type": "paragraph",
                "text": para_text,
            })

        # Add a summary/closing block for certain types
        if document_type in ("implementation_plan", "project_report", "executive_brief"):
            blocks.append({
                "id": "blk_999",
                "type": "callout",
                "variant": "info",
                "text": self._closing_note(document_type, tone),
            })

        document_json = {
            "schema_version": 1,
            "title": title,
            "document_type": document_type,
            "blocks": blocks,
        }

        word_count = sum(
            len(b.get("text", "").split()) + sum(len(it.split()) for it in b.get("items", []))
            for b in blocks
        )

        summary = f"Generated {document_type} with {len(sections)} sections ({tone} tone, ~{word_count} words)."

        return {
            "document_json": document_json,
            "summary": summary,
        }

    def _default_sections(self, doc_type: str) -> list[str]:
        """Return default section names per document type."""
        defaults = {
            "general_document": ["Overview", "Background", "Discussion", "Conclusion"],
            "decision_memo": ["Context", "Options Considered", "Recommendation", "Next Steps"],
            "implementation_plan": ["Overview", "Scope", "Architecture", "Risks", "Timeline", "Next Steps"],
            "incident_summary": ["Incident Overview", "Impact", "Root Cause", "Resolution", "Lessons Learned"],
            "training_doc": ["Introduction", "Prerequisites", "Steps", "Best Practices", "Troubleshooting"],
            "architecture_doc": ["Overview", "System Context", "Components", "Data Flow", "Security", "Scalability"],
            "project_report": ["Executive Summary", "Objectives", "Progress", "Challenges", "Next Steps"],
            "status_report": ["Current Status", "Completed Work", "In Progress", "Risks", "Upcoming"],
            "meeting_summary": ["Attendees", "Agenda", "Discussion Points", "Decisions", "Action Items"],
            "executive_brief": ["Summary", "Key Findings", "Strategic Implications", "Recommendations"],
        }
        return defaults.get(doc_type, ["Overview", "Details", "Summary"])

    def _section_content(self, section_name: str, doc_type: str, tone: str, prompt: str) -> str:
        """Generate deterministic placeholder content for a section."""
        # Tone adjustments
        tone_prefix = {
            "technical": "From a technical perspective,",
            "executive": "From a strategic standpoint,",
            "casual": "So,",
            "formal": "It is noted that,",
        }.get(tone, "")

        prompt_ref = prompt[:80] if prompt else "the specified requirements"

        return (
            f"{tone_prefix} this section addresses {section_name.lower()} "
            f"in the context of: {prompt_ref}. "
            f"The content here should be expanded with specific details, "
            f"relevant data, and appropriate references for this {doc_type.replace('_', ' ')}."
        )

    def _closing_note(self, doc_type: str, tone: str) -> str:
        """Generate a closing callout note."""
        return (
            f"This {doc_type.replace('_', ' ')} was generated as a draft. "
            f"Review and refine the content before finalizing."
        )
