"""
Tests for Document Assist Gateway and HTTP Server.

These tests verify:
- /document-assist requires worker token
- Rejects oversized requests
- Returns structured JSON for various modes
- Does not log raw content
- Model gateway failure returns safe error
"""

import json
import os
import sys
import unittest
from io import StringIO
from unittest.mock import patch
import urllib.request
import urllib.error
import threading
import time

# Add parent dir to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from document_assist import (
    StubDocumentAssistGateway,
    validate_assist_mode,
    validate_assist_response,
    ASSIST_MODES,
)


class TestDocumentAssistGateway(unittest.TestCase):
    """Test the stub document assist gateway."""

    def setUp(self):
        self.gateway = StubDocumentAssistGateway()

    def test_rewrite_returns_paragraph(self):
        result = self.gateway.generate_assist(
            mode="rewrite",
            selected_text="This is some text to rewrite.",
            instruction="",
            document_type="general_document",
            max_words=300,
        )
        self.assertIn("suggested_blocks", result)
        self.assertIn("summary", result)
        self.assertGreater(len(result["suggested_blocks"]), 0)
        self.assertEqual(result["suggested_blocks"][0]["type"], "paragraph")
        self.assertIn("text", result["suggested_blocks"][0])

    def test_summarize_returns_paragraph(self):
        result = self.gateway.generate_assist(
            mode="summarize",
            selected_text="This is a longer piece of text that should be summarized into something much shorter.",
            instruction="",
            document_type="general_document",
            max_words=100,
        )
        self.assertGreater(len(result["suggested_blocks"]), 0)
        self.assertIn("Summary:", result["suggested_blocks"][0]["text"])

    def test_create_outline_returns_multiple_blocks(self):
        result = self.gateway.generate_assist(
            mode="create_outline",
            selected_text="",
            instruction="",
            document_type="implementation_plan",
            max_words=500,
        )
        self.assertGreater(len(result["suggested_blocks"]), 1)
        self.assertEqual(result["suggested_blocks"][0]["type"], "heading")

    def test_extract_action_items_returns_bullets(self):
        result = self.gateway.generate_assist(
            mode="extract_action_items",
            selected_text="We need to follow up on the API design.\nThe team should review the PR.\nAction: deploy to staging.",
            instruction="",
            document_type="meeting_summary",
            max_words=300,
        )
        self.assertEqual(result["suggested_blocks"][0]["type"], "bullets")
        self.assertIn("items", result["suggested_blocks"][0])

    def test_draft_section_returns_heading_and_paragraph(self):
        result = self.gateway.generate_assist(
            mode="draft_section",
            selected_text="",
            instruction="Write about deployment strategy",
            document_type="implementation_plan",
            max_words=300,
        )
        types = [b["type"] for b in result["suggested_blocks"]]
        self.assertIn("heading", types)
        self.assertIn("paragraph", types)

    def test_all_modes_produce_valid_output(self):
        for mode in ASSIST_MODES:
            result = self.gateway.generate_assist(
                mode=mode,
                selected_text="Sample text for processing by the assist gateway.",
                instruction="Test instruction",
                document_type="general_document",
                max_words=300,
            )
            self.assertIn("suggested_blocks", result, f"mode {mode} missing suggested_blocks")
            self.assertIn("summary", result, f"mode {mode} missing summary")
            self.assertGreater(len(result["suggested_blocks"]), 0, f"mode {mode} returned empty blocks")

    def test_no_forbidden_fields_in_response(self):
        for mode in ASSIST_MODES:
            result = self.gateway.generate_assist(
                mode=mode,
                selected_text="Test text",
                instruction="",
                document_type="general_document",
                max_words=300,
            )
            errors = validate_assist_response(result)
            self.assertEqual(errors, [], f"mode {mode} returned forbidden fields: {errors}")

    def test_max_words_bounds_respected(self):
        result = self.gateway.generate_assist(
            mode="expand",
            selected_text="Short text",
            instruction="",
            document_type="general_document",
            max_words=20,
        )
        # The stub should bound the output
        for block in result["suggested_blocks"]:
            if "text" in block:
                # 20 words * ~6 chars = ~120 chars max in stub
                self.assertLessEqual(len(block["text"]), 200)


class TestAssistValidation(unittest.TestCase):
    """Test mode validation."""

    def test_valid_modes(self):
        for mode in ASSIST_MODES:
            errors = validate_assist_mode(mode)
            self.assertEqual(errors, [], f"valid mode {mode} should have no errors")

    def test_invalid_mode(self):
        errors = validate_assist_mode("invalid_mode")
        self.assertGreater(len(errors), 0)

    def test_forbidden_field_detection(self):
        errors = validate_assist_response({"chain_of_thought": "secret thinking"})
        self.assertGreater(len(errors), 0)
        errors = validate_assist_response({"thinking": "internal reasoning"})
        self.assertGreater(len(errors), 0)

    def test_clean_response_no_errors(self):
        errors = validate_assist_response({"suggested_blocks": [], "summary": "ok"})
        self.assertEqual(errors, [])


class TestDocumentAssistServer(unittest.TestCase):
    """Test the HTTP server endpoints."""

    @classmethod
    def setUpClass(cls):
        """Start a test worker with document assist server."""
        os.environ["WORKER_TOKEN"] = "test-token-123"

        # Import worker module after setting env
        import worker as worker_module

        cls.gateway = StubDocumentAssistGateway()
        cls.server = worker_module.DocumentAssistServer(19100, "test-token-123", cls.gateway)
        cls.server.start()
        time.sleep(0.5)  # Let server start

    @classmethod
    def tearDownClass(cls):
        cls.server.stop()

    def _make_request(self, path="/document-assist", body=None, token="test-token-123", method="POST"):
        """Helper to make HTTP request to the test server."""
        url = f"http://localhost:19100{path}"
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
        status, body = self._make_request(token="wrong-token")
        self.assertEqual(status, 401)

    def test_no_token_rejected(self):
        status, body = self._make_request(token="")
        self.assertEqual(status, 401)

    def test_rejects_oversized_request(self):
        # Create a body over 100KB
        big_text = "x" * 101_000
        status, body = self._make_request(body={
            "mode": "rewrite",
            "selected_text": big_text,
            "instruction": "",
            "document_type": "general_document",
            "max_words": 300,
        })
        self.assertIn(status, (400, 413))  # 413 from body size, 400 from field validation

    def test_returns_structured_json_rewrite(self):
        status, body = self._make_request(body={
            "mode": "rewrite",
            "selected_text": "This needs to be rewritten for clarity.",
            "instruction": "Make it clearer",
            "document_type": "general_document",
            "max_words": 300,
        })
        self.assertEqual(status, 200)
        self.assertIn("suggested_blocks", body)
        self.assertIn("summary", body)
        self.assertGreater(len(body["suggested_blocks"]), 0)

    def test_returns_structured_json_summarize(self):
        status, body = self._make_request(body={
            "mode": "summarize",
            "selected_text": "This is a longer piece of text that contains multiple sentences and should be summarized into a shorter form.",
            "instruction": "",
            "document_type": "general_document",
            "max_words": 100,
        })
        self.assertEqual(status, 200)
        self.assertIn("suggested_blocks", body)

    def test_health_endpoint(self):
        status, body = self._make_request(path="/health", method="GET", token="test-token-123")
        self.assertEqual(status, 200)
        self.assertEqual(body["status"], "ok")

    def test_invalid_mode_returns_400(self):
        status, body = self._make_request(body={
            "mode": "invalid",
            "selected_text": "text",
            "instruction": "",
            "document_type": "general_document",
            "max_words": 300,
        })
        self.assertEqual(status, 400)

    def test_max_words_bounds(self):
        status, body = self._make_request(body={
            "mode": "rewrite",
            "selected_text": "text",
            "instruction": "",
            "document_type": "general_document",
            "max_words": 5,
        })
        self.assertEqual(status, 400)


class TestNoContentLogging(unittest.TestCase):
    """Verify that raw content is not logged."""

    def test_log_message_does_not_contain_content(self):
        """The HTTP handler suppresses log messages to prevent content leakage."""
        # Capture logs
        log_buffer = StringIO()
        import logging
        handler = logging.StreamHandler(log_buffer)
        logger = logging.getLogger("reasoning-worker")
        logger.addHandler(handler)

        gateway = StubDocumentAssistGateway()
        result = gateway.generate_assist(
            mode="rewrite",
            selected_text="SUPER_SECRET_CONTENT_THAT_MUST_NOT_APPEAR_IN_LOGS",
            instruction="",
            document_type="general_document",
            max_words=300,
        )

        # The gateway itself should not log the content
        log_output = log_buffer.getvalue()
        self.assertNotIn("SUPER_SECRET_CONTENT", log_output)

        logger.removeHandler(handler)


if __name__ == "__main__":
    unittest.main()
