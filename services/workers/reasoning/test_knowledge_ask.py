"""Tests for the knowledge_ask module."""

import pytest
from unittest.mock import MagicMock
from knowledge_ask import handle_knowledge_ask, validate_knowledge_ask, _sanitize_response, _extract_citations


class TestHandleKnowledgeAsk:
    def test_returns_answer_with_citations(self):
        """Test 1: knowledge-ask returns answer with citations."""
        gateway = MagicMock()
        gateway.generate.return_value = "Based on [src-0], the backup runs daily. Also see [src-1] for details."

        payload = {
            "question": "What is the backup process?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Backup Guide", "snippet": "Backups run daily at 2am."},
                {"source_key": "src-1", "source_type": "incident", "source_id": "inc-1", "title": "Backup Incident", "snippet": "Backup failed on June 10."},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        assert "answer" in result
        assert len(result["answer"]) > 0
        assert "src-0" in result["citations"]
        assert "src-1" in result["citations"]

    def test_cites_only_provided_source_keys(self):
        """Test 2: cites only provided source keys."""
        gateway = MagicMock()
        gateway.generate.return_value = "According to [src-0], the answer is yes."

        payload = {
            "question": "Is the backup verified?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Doc", "snippet": "Verification runs daily."},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        for citation in result["citations"]:
            assert citation in ["src-0"]

    def test_no_sources_returns_low_confidence(self):
        """Test 3: no sources returns low confidence."""
        gateway = MagicMock()
        payload = {"question": "What is our policy?", "sources": []}

        result = handle_knowledge_ask(payload, gateway)

        assert result["confidence"] == "low"
        assert len(result["missing_info"]) > 0
        gateway.generate.assert_not_called()

    def test_rejects_chain_of_thought_fields(self):
        """Test 4: rejects/omits chain-of-thought fields."""
        gateway = MagicMock()
        gateway.generate.return_value = "The answer is yes."

        payload = {
            "question": "What is the policy?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Doc", "snippet": "The policy is documented."},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        assert "chain_of_thought" not in result
        assert "thinking" not in result
        assert "internal_reasoning" not in result

    def test_handles_missing_snippets_safely(self):
        """Test 5: handles missing snippets safely."""
        gateway = MagicMock()
        gateway.generate.return_value = "Limited information available."

        payload = {
            "question": "What is the policy?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Doc", "snippet": ""},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        assert "answer" in result
        # Should not crash

    def test_bounded_answer_length(self):
        """Test 6: bounded answer length."""
        gateway = MagicMock()
        gateway.generate.return_value = "A" * 20000  # Way too long

        payload = {
            "question": "What is the policy?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Doc", "snippet": "Policy content here."},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        assert len(result["answer"]) <= 8000

    def test_model_failure_returns_safe_response(self):
        """Test 7: model failure returns safe response."""
        gateway = MagicMock()
        gateway.generate.side_effect = Exception("API error")

        payload = {
            "question": "What is the policy?",
            "sources": [
                {"source_key": "src-0", "source_type": "clarity_document", "source_id": "doc-1", "title": "Doc", "snippet": "Policy content."},
            ],
        }

        result = handle_knowledge_ask(payload, gateway)

        assert result["confidence"] == "low"
        assert "could not generate" in result["answer"].lower()


class TestValidateKnowledgeAsk:
    def test_valid_payload(self):
        errors = validate_knowledge_ask({"question": "valid question here", "sources": []})
        assert len(errors) == 0

    def test_short_question(self):
        errors = validate_knowledge_ask({"question": "hi", "sources": []})
        assert len(errors) > 0

    def test_long_question(self):
        errors = validate_knowledge_ask({"question": "x" * 1001, "sources": []})
        assert len(errors) > 0

    def test_invalid_sources_type(self):
        errors = validate_knowledge_ask({"question": "valid question", "sources": "not a list"})
        assert len(errors) > 0


class TestSanitizeResponse:
    def test_removes_forbidden_fields(self):
        resp = {
            "answer": "test",
            "chain_of_thought": "secret",
            "thinking": "hidden",
            "internal_reasoning": "internal",
            "tool_calls": [],
            "prompt": "system prompt",
            "raw_prompt": "raw",
        }
        clean = _sanitize_response(resp)
        assert "answer" in clean
        assert "chain_of_thought" not in clean
        assert "thinking" not in clean
        assert "internal_reasoning" not in clean
        assert "tool_calls" not in clean
        assert "prompt" not in clean
        assert "raw_prompt" not in clean


class TestExtractCitations:
    def test_finds_all_keys(self):
        answer = "Based on [src-0] and [src-2], the answer is yes."
        keys = ["src-0", "src-1", "src-2"]
        cited = _extract_citations(answer, keys)
        assert "src-0" in cited
        assert "src-1" not in cited
        assert "src-2" in cited

    def test_no_keys_found(self):
        answer = "The answer is yes."
        keys = ["src-0", "src-1"]
        cited = _extract_citations(answer, keys)
        assert len(cited) == 0
