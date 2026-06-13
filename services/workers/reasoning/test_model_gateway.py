"""Tests for model_gateway module."""

import sys
import os
import pytest

sys.path.insert(0, os.path.dirname(__file__))

from model_gateway import (
    StubModelGateway,
    IntentionShape,
    validate_model_output,
    FORBIDDEN_FIELDS,
)


class TestStubModelGateway:
    def test_returns_valid_intention(self):
        gw = StubModelGateway()
        intention = gw.generate_intention(
            agent_run_id="run-123",
            context={},
            tool_grants=[],
        )
        assert isinstance(intention, IntentionShape)
        assert intention.agent_run_id == "run-123"
        assert intention.intention_type == "incidents.add_timeline"
        assert intention.requested_tool == "incidents.add_timeline"
        assert 0.0 <= intention.confidence <= 1.0
        assert intention.reasoning_summary != ""
        assert intention.validate() == []

    def test_to_api_payload(self):
        gw = StubModelGateway()
        intention = gw.generate_intention("run-456", {}, [])
        payload = intention.to_api_payload()
        assert "intention_type" in payload
        assert "reasoning_summary" in payload
        assert "chain_of_thought" not in payload
        assert "thinking" not in payload


class TestInvalidModelOutput:
    def test_chain_of_thought_rejected(self):
        raw = {"chain_of_thought": "Let me think step by step...", "intention_type": "test"}
        errors = validate_model_output(raw)
        assert any("forbidden" in e for e in errors)

    def test_thinking_rejected(self):
        raw = {"thinking": "I should consider...", "intention_type": "test"}
        errors = validate_model_output(raw)
        assert any("forbidden" in e for e in errors)

    def test_valid_output_passes(self):
        raw = {"intention_type": "test", "reasoning_summary": "valid"}
        errors = validate_model_output(raw)
        assert errors == []


class TestIntentionShapeValidation:
    def test_missing_fields(self):
        shape = IntentionShape(agent_run_id="", intention_type="", requested_tool="")
        errors = shape.validate()
        assert len(errors) >= 3  # at least agent_run_id, intention_type, requested_tool

    def test_invalid_confidence(self):
        shape = IntentionShape(
            agent_run_id="r1",
            intention_type="test",
            requested_tool="test",
            confidence=2.0,
            reasoning_summary="test",
        )
        errors = shape.validate()
        assert any("confidence" in e for e in errors)

    def test_invalid_autonomy(self):
        shape = IntentionShape(
            agent_run_id="r1",
            intention_type="test",
            requested_tool="test",
            autonomy_level="A6",
            reasoning_summary="test",
        )
        errors = shape.validate()
        assert any("autonomy" in e for e in errors)

    def test_no_reasoning_summary_rejected(self):
        shape = IntentionShape(
            agent_run_id="r1",
            intention_type="test",
            requested_tool="test",
            reasoning_summary="",
        )
        errors = shape.validate()
        assert any("reasoning_summary" in e for e in errors)
