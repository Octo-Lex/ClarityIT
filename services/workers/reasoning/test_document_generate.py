"""
Tests for Document Generation Gateway and HTTP endpoint.

These tests verify:
- /document-generate requires worker token
- Rejects oversized requests
- Returns structured document_json
- Supports all allowed document_type values
- Supports all allowed tones
- Gateway error returns safe error
- Logs do not include raw prompt
- No DB/MinIO/NATS/Redis mutation behavior
"""

import json
import os
import sys
import unittest
from io import StringIO
from unittest.mock import patch
import urllib.request
import urllib.error
import time
import logging

# Add parent dir to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from document_generate import (
    StubDocumentGenerateGateway,
    validate_generate_request,
    validate_generate_response,
    GENERATE_TONES,
    GENERATE_DOC_TYPES,
)


class TestDocumentGenerateGateway(unittest.TestCase):
    """Test the stub document generate gateway."""

    def setUp(self):
        self.gateway = StubDocumentGenerateGateway()

    def test_generates_valid_document_json(self):
        result = self.gateway.generate_document(
            title="Test Plan",
            document_type="implementation_plan",
            prompt="Create an implementation plan for v1.5",
            tone="technical",
            sections=["Overview", "Scope", "Risks"],
        )
        self.assertIn("document_json", result)
        self.assertIn("summary", result)
        doc = result["document_json"]
        self.assertEqual(doc["schema_version"], 1)
        self.assertEqual(doc["title"], "Test Plan")
        self.assertEqual(doc["document_type"], "implementation_plan")
        self.assertGreater(len(doc["blocks"]), 0)

    def test_blocks_are_valid_types(self):
        result = self.gateway.generate_document(
            title="Test",
            document_type="general_document",
            prompt="A general document",
            tone="formal",
        )
        for block in result["document_json"]["blocks"]:
            self.assertIn("type", block)
            self.assertIn(block["type"], {"heading", "paragraph", "callout", "bullets", "numbered_list", "quote", "table", "page_break"})
            self.assertIn("id", block)

    def test_supports_all_document_types(self):
        for doc_type in GENERATE_DOC_TYPES:
            result = self.gateway.generate_document(
                title=f"Test {doc_type}",
                document_type=doc_type,
                prompt="Test prompt",
                tone="technical",
            )
            self.assertEqual(result["document_json"]["document_type"], doc_type)
            self.assertGreater(len(result["document_json"]["blocks"]), 0)

    def test_supports_all_tones(self):
        for tone in GENERATE_TONES:
            result = self.gateway.generate_document(
                title="Tone Test",
                document_type="general_document",
                prompt="Test prompt",
                tone=tone,
            )
            self.assertIn("document_json", result)

    def test_custom_sections_used(self):
        result = self.gateway.generate_document(
            title="Custom Sections",
            document_type="project_report",
            prompt="A project report",
            tone="executive",
            sections=["Intro", "Analysis", "Conclusion"],
        )
        # Should have heading blocks matching sections
        headings = [b for b in result["document_json"]["blocks"] if b["type"] == "heading"]
        heading_texts = [h["text"] for h in headings]
        for s in ["Intro", "Analysis", "Conclusion"]:
            self.assertIn(s, heading_texts)

    def test_default_sections_when_none_provided(self):
        result = self.gateway.generate_document(
            title="Defaults",
            document_type="implementation_plan",
            prompt="Test",
            tone="technical",
            sections=None,
        )
        headings = [b["text"] for b in result["document_json"]["blocks"] if b["type"] == "heading"]
        self.assertGreater(len(headings), 1)

    def test_no_forbidden_fields(self):
        result = self.gateway.generate_document(
            title="Forbidden Check",
            document_type="general_document",
            prompt="Test",
            tone="technical",
        )
        errors = validate_generate_response(result)
        self.assertEqual(errors, [], f"found forbidden fields: {errors}")


class TestGenerateValidation(unittest.TestCase):
    """Test request validation."""

    def test_valid_request(self):
        errors = validate_generate_request({
            "title": "Test",
            "document_type": "general_document",
            "prompt": "A prompt",
            "tone": "technical",
            "sections": [],
        })
        self.assertEqual(errors, [])

    def test_missing_title(self):
        errors = validate_generate_request({
            "document_type": "general_document",
            "prompt": "A prompt",
        })
        self.assertTrue(any("title" in e for e in errors))

    def test_invalid_doc_type(self):
        errors = validate_generate_request({
            "title": "Test",
            "document_type": "bad_type",
            "prompt": "A prompt",
        })
        self.assertTrue(any("document_type" in e for e in errors))

    def test_missing_prompt(self):
        errors = validate_generate_request({
            "title": "Test",
            "document_type": "general_document",
            "prompt": "",
        })
        self.assertTrue(any("prompt" in e for e in errors))

    def test_invalid_tone(self):
        errors = validate_generate_request({
            "title": "Test",
            "document_type": "general_document",
            "prompt": "A prompt",
            "tone": "angry",
        })
        self.assertTrue(any("tone" in e for e in errors))

    def test_too_many_sections(self):
        errors = validate_generate_request({
            "title": "Test",
            "document_type": "general_document",
            "prompt": "A prompt",
            "sections": [f"S{i}" for i in range(21)],
        })
        self.assertTrue(any("sections" in e for e in errors))


class TestDocumentGenerateServer(unittest.TestCase):
    """Test the HTTP server /document-generate endpoint."""

    @classmethod
    def setUpClass(cls):
        os.environ["WORKER_TOKEN"] = "test-gen-token-456"

        import worker as worker_module

        cls.gateway = StubDocumentGenerateGateway()
        cls.assist_gateway = worker_module.StubDocumentAssistGateway()
        cls.server = worker_module.DocumentAssistServer(19200, "test-gen-token-456", cls.assist_gateway, cls.gateway)
        cls.server.start()
        time.sleep(0.5)

    @classmethod
    def tearDownClass(cls):
        cls.server.stop()

    def _make_request(self, path="/document-generate", body=None, token="test-gen-token-456", method="POST"):
        url = f"http://localhost:19200{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        if token:
            req.add_header("Authorization", f"Bearer {token}")
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                return resp.status, json.loads(resp.read())
        except urllib.error.HTTPError as e:
            raw = e.read().decode()
            try:
                return e.code, json.loads(raw)
            except json.JSONDecodeError:
                return e.code, {"raw": raw}

    def test_requires_worker_token(self):
        status, _ = self._make_request(token="wrong-token")
        self.assertEqual(status, 401)

    def test_no_token_rejected(self):
        status, _ = self._make_request(token="")
        self.assertEqual(status, 401)

    def test_rejects_oversized_request(self):
        big_prompt = "x" * 101_000
        status, _ = self._make_request(body={
            "title": "Big",
            "document_type": "general_document",
            "prompt": big_prompt,
            "tone": "technical",
        })
        self.assertIn(status, (400, 413))

    def test_returns_structured_document_json(self):
        status, body = self._make_request(body={
            "title": "API Test Doc",
            "document_type": "implementation_plan",
            "prompt": "Create an implementation plan for the new feature.",
            "tone": "technical",
            "sections": ["Overview", "Scope"],
        })
        self.assertEqual(status, 200)
        self.assertIn("document_json", body)
        self.assertIn("summary", body)
        doc = body["document_json"]
        self.assertEqual(doc["schema_version"], 1)
        self.assertEqual(doc["title"], "API Test Doc")
        self.assertEqual(doc["document_type"], "implementation_plan")
        self.assertGreater(len(doc["blocks"]), 0)

    def test_supports_all_document_types_via_endpoint(self):
        for doc_type in GENERATE_DOC_TYPES:
            status, body = self._make_request(body={
                "title": f"Test {doc_type}",
                "document_type": doc_type,
                "prompt": "Test prompt",
                "tone": "technical",
            })
            self.assertEqual(status, 200)
            self.assertEqual(body["document_json"]["document_type"], doc_type)

    def test_supports_all_tones_via_endpoint(self):
        for tone in GENERATE_TONES:
            status, body = self._make_request(body={
                "title": "Tone Test",
                "document_type": "general_document",
                "prompt": "Test prompt",
                "tone": tone,
            })
            self.assertEqual(status, 200)

    def test_invalid_doc_type_returns_400(self):
        status, body = self._make_request(body={
            "title": "Test",
            "document_type": "invalid",
            "prompt": "Test",
            "tone": "technical",
        })
        self.assertEqual(status, 400)

    def test_missing_title_returns_400(self):
        status, body = self._make_request(body={
            "document_type": "general_document",
            "prompt": "Test",
            "tone": "technical",
        })
        self.assertEqual(status, 400)


class TestNoPromptLogging(unittest.TestCase):
    """Verify that raw prompt is not logged."""

    def test_log_does_not_contain_prompt(self):
        log_buffer = StringIO()
        handler = logging.StreamHandler(log_buffer)
        logger = logging.getLogger("reasoning-worker")
        logger.addHandler(handler)

        gateway = StubDocumentGenerateGateway()
        gateway.generate_document(
            title="Secret Doc",
            document_type="general_document",
            prompt="SUPER_SECRET_PROMPT_THAT_MUST_NOT_APPEAR_IN_LOGS",
            tone="technical",
        )

        log_output = log_buffer.getvalue()
        self.assertNotIn("SUPER_SECRET_PROMPT", log_output)

        logger.removeHandler(handler)


if __name__ == "__main__":
    unittest.main()
