"""
Model Gateway Abstraction for ClarityIT Agent Runtime.

Provides a unified interface for LLM-backed intention generation.
For Genesis, only StubModelGateway is implemented — real providers
(OpenAI, LiteLLM, Ollama) are placeholder stubs.

Key rules:
- Output must be a structured IntentionShape dict
- chain_of_thought field is ALWAYS rejected/stripped
- reasoning_summary is the ONLY narrative output
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from typing import Any, Protocol


@dataclass
class IntentionShape:
    """Structured intention output — the ONLY shape the reasoning worker produces."""

    agent_run_id: str
    intention_type: str
    requested_tool: str
    target: dict[str, Any] = field(default_factory=dict)
    confidence: float = 0.0
    risk_level: str = "low"
    autonomy_level: str = "A0"
    reasoning_summary: str = ""
    evidence_refs: list[dict[str, Any]] = field(default_factory=list)

    def to_api_payload(self) -> dict[str, Any]:
        return {
            "intention_type": self.intention_type,
            "requested_tool": self.requested_tool,
            "target": self.target,
            "confidence": self.confidence,
            "risk_level": self.risk_level,
            "autonomy_level": self.autonomy_level,
            "reasoning_summary": self.reasoning_summary,
            "evidence_refs": self.evidence_refs,
        }

    def validate(self) -> list[str]:
        errors: list[str] = []
        if not self.agent_run_id:
            errors.append("agent_run_id required")
        if not self.intention_type:
            errors.append("intention_type required")
        if not self.requested_tool:
            errors.append("requested_tool required")
        if not (0.0 <= self.confidence <= 1.0):
            errors.append("confidence must be 0.0–1.0")
        if self.risk_level not in ("low", "medium", "high", "critical"):
            errors.append("invalid risk_level")
        if not re.match(r"^A[0-5]$", self.autonomy_level):
            errors.append("invalid autonomy_level")
        if not self.reasoning_summary:
            errors.append("reasoning_summary required — no chain-of-thought allowed")
        return errors


class ModelGateway(Protocol):
    """Interface all model gateways must implement."""

    def generate_intention(
        self,
        agent_run_id: str,
        context: dict[str, Any],
        tool_grants: list[dict[str, Any]],
    ) -> IntentionShape:
        ...


# ─── Forbidden field detection ───

FORBIDDEN_FIELDS = {"chain_of_thought", "chain-of-thought", "thinking", "thought_process", "internal_reasoning"}


def validate_model_output(raw: dict[str, Any]) -> list[str]:
    """Check raw model output for forbidden fields and shape issues."""
    errors: list[str] = []
    for key in raw:
        if key.lower().replace("-", "_") in FORBIDDEN_FIELDS or key.lower() in FORBIDDEN_FIELDS:
            errors.append(f"forbidden field '{key}' — chain-of-thought is rejected")
    return errors


# ─── Stub Model Gateway ───


class StubModelGateway:
    """
    Returns a fixed, valid intention for testing.

    Never calls a real LLM. Produces deterministic output.
    """

    def __init__(
        self,
        intention_type: str = "incidents.add_timeline",
        requested_tool: str = "incidents.add_timeline",
        confidence: float = 0.75,
        risk_level: str = "low",
        autonomy_level: str = "A3",
        reasoning_summary: str = "Stub: incident requires a timeline update based on operational context.",
    ):
        self._intention_type = intention_type
        self._requested_tool = requested_tool
        self._confidence = confidence
        self._risk_level = risk_level
        self._autonomy_level = autonomy_level
        self._reasoning_summary = reasoning_summary

    def generate_intention(
        self,
        agent_run_id: str,
        context: dict[str, Any],
        tool_grants: list[dict[str, Any]],
    ) -> IntentionShape:
        return IntentionShape(
            agent_run_id=agent_run_id,
            intention_type=self._intention_type,
            requested_tool=self._requested_tool,
            confidence=self._confidence,
            risk_level=self._risk_level,
            autonomy_level=self._autonomy_level,
            reasoning_summary=self._reasoning_summary,
        )


# ─── Placeholder gateways (not yet implemented) ───


class OpenAICompatibleGateway:
    """Placeholder for OpenAI-compatible API integration."""

    def __init__(self, api_key: str = "", model: str = "gpt-4o-mini", base_url: str = ""):
        self._api_key = api_key
        self._model = model
        self._base_url = base_url

    def generate_intention(
        self,
        agent_run_id: str,
        context: dict[str, Any],
        tool_grants: list[dict[str, Any]],
    ) -> IntentionShape:
        raise NotImplementedError("OpenAICompatibleGateway not yet implemented")


class LiteLLMGateway:
    """Placeholder for LiteLLM proxy integration."""

    def __init__(self, base_url: str = ""):
        self._base_url = base_url

    def generate_intention(
        self,
        agent_run_id: str,
        context: dict[str, Any],
        tool_grants: list[dict[str, Any]],
    ) -> IntentionShape:
        raise NotImplementedError("LiteLLMGateway not yet implemented")


class LocalOllamaGateway:
    """Placeholder for local Ollama integration."""

    def __init__(self, base_url: str = "http://localhost:11434", model: str = "llama3"):
        self._base_url = base_url
        self._model = model

    def generate_intention(
        self,
        agent_run_id: str,
        context: dict[str, Any],
        tool_grants: list[dict[str, Any]],
    ) -> IntentionShape:
        raise NotImplementedError("LocalOllamaGateway not yet implemented")
