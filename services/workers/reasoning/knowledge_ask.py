"""
Knowledge Ask handler for the Python reasoning worker.

This module handles POST /knowledge-ask requests.

Rules:
- No DB access
- No MinIO access
- No NATS access
- No Redis access
- No Tool Gateway
- No Proxmox
- No operational actions
- No source mutation
- No document mutation
- No long-term memory
- No external search
- No vector DB
- No chain-of-thought
- No hidden reasoning fields
- No raw prompt logging
"""

import json
import logging
from typing import Any

log = logging.getLogger(__name__)

MAX_ANSWER_LENGTH = 8000


def handle_knowledge_ask(payload: dict, gateway: Any) -> dict:
    """
    Handle a knowledge-ask request.

    Args:
        payload: {
            "question": "...",
            "sources": [
                {"source_key": "...", "source_type": "...", "source_id": "...", "title": "...", "snippet": "..."}
            ]
        }
        gateway: ModelGateway for LLM calls

    Returns:
        {
            "answer": "...",
            "citations": ["source_key_1", "source_key_2"],
            "confidence": "low|medium|high",
            "missing_info": []
        }
    """
    question = payload.get("question", "").strip()
    sources = payload.get("sources", [])

    # Validate question
    if not question or len(question) < 5:
        return {
            "answer": "Question is too short to answer.",
            "citations": [],
            "confidence": "low",
            "missing_info": ["Question was too short."],
        }

    # Handle no sources
    if not sources:
        return {
            "answer": "There is not enough indexed knowledge to answer this question.",
            "citations": [],
            "confidence": "low",
            "missing_info": ["No sources were provided."],
        }

    # Build context from snippets — no raw content logging
    context_parts = []
    source_keys = []
    for src in sources:
        key = src.get("source_key", "")
        title = src.get("title", "Untitled")
        snippet = src.get("snippet", "")
        if snippet:
            context_parts.append(f"[{key}] {title}:\n{snippet}")
            source_keys.append(key)

    if not context_parts:
        return {
            "answer": "The available knowledge sources do not contain enough detail to answer this question.",
            "citations": [],
            "confidence": "low",
            "missing_info": ["All source snippets were empty."],
        }

    context = "\n\n".join(context_parts)

    # Build the system prompt — no chain-of-thought, no hidden reasoning
    system_prompt = (
        "You are ClarityIT's knowledge assistant. "
        "Answer the user's question using ONLY the provided source documents. "
        "Cite sources using their source_key in the format [src-N]. "
        "If the sources do not contain enough information, say so clearly. "
        "Do not include chain-of-thought, thinking, or internal reasoning. "
        "Respond with only the answer."
    )

    user_prompt = f"Source documents:\n\n{context}\n\nQuestion: {question}\n\nAnswer:"

    # Call the model gateway
    try:
        answer = gateway.generate(system_prompt, user_prompt, max_tokens=2000, temperature=0.3)
    except Exception as e:
        log.error("Knowledge ask generation failed: %s", type(e).__name__)
        return {
            "answer": "The knowledge assistant could not generate an answer at this time.",
            "citations": [],
            "confidence": "low",
            "missing_info": ["Model generation failed."],
        }

    # Bound answer length
    if len(answer) > MAX_ANSWER_LENGTH:
        answer = answer[:MAX_ANSWER_LENGTH]

    # Determine confidence based on answer content
    confidence = _assess_confidence(answer, source_keys)

    # Extract citations from answer — find which source_keys are referenced
    citations = _extract_citations(answer, source_keys)

    # Missing info assessment
    missing_info = []
    lower_answer = answer.lower()
    if any(phrase in lower_answer for phrase in ["not enough", "insufficient", "cannot determine", "don't have enough"]):
        missing_info.append("The available sources may not fully answer this question.")
    if len(citations) == 0:
        missing_info.append("No specific sources were cited in the answer.")

    # Ensure no forbidden fields leak
    return _sanitize_response({
        "answer": answer.strip(),
        "citations": citations,
        "confidence": confidence,
        "missing_info": missing_info,
    })


def _assess_confidence(answer: str, all_keys: list[str]) -> str:
    """Assess confidence level based on answer characteristics."""
    lower = answer.lower()
    if len(answer) < 50:
        return "low"
    if any(w in lower for w in ["cannot", "unable", "not enough", "insufficient", "unclear"]):
        return "low"
    if len(answer) > 200 and any(key in answer for key in all_keys):
        return "high"
    return "medium"


def _extract_citations(answer: str, all_keys: list[str]) -> list[str]:
    """Extract cited source keys from the answer text."""
    cited = []
    for key in all_keys:
        if key in answer:
            cited.append(key)
    return cited


def _sanitize_response(resp: dict) -> dict:
    """Ensure no forbidden fields are present in the response."""
    forbidden = [
        "chain_of_thought", "thinking", "internal_reasoning",
        "tool_calls", "action", "mutation", "execute",
        "prompt", "system_prompt", "raw_prompt",
    ]
    clean = {}
    for k, v in resp.items():
        if k not in forbidden:
            clean[k] = v
    return clean


def validate_knowledge_ask(payload: dict) -> list[str]:
    """Validate the knowledge-ask request payload. Returns list of errors."""
    errors = []
    question = payload.get("question", "")
    if not question or len(question) < 5:
        errors.append("question must be at least 5 characters")
    if len(question) > 1000:
        errors.append("question must be at most 1000 characters")

    sources = payload.get("sources", [])
    if not isinstance(sources, list):
        errors.append("sources must be a list")

    return errors
