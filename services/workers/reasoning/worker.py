"""
ClarityIT Reasoning Worker.

Polls the Go API for pending agent runs, fetches context,
uses a ModelGateway to generate structured intentions, and
POSTs them back to the API.

NEVER writes directly to PostgreSQL, NATS, Redis, MinIO, Git, or Proxmox.
Communicates ONLY through the Go API HTTP interface.
"""

from __future__ import annotations

import json
import logging
import os
import signal
import sys
import time
from typing import Any

import urllib.request
import urllib.error

from model_gateway import (
    StubModelGateway,
    ModelGateway,
    IntentionShape,
    validate_model_output,
)
from document_assist import (
    StubDocumentAssistGateway,
    validate_assist_mode,
    validate_assist_response,
    ASSIST_MODES,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("reasoning-worker")


class APIClient:
    """HTTP client for the Go control plane API. No database access."""

    def __init__(self, base_url: str, token: str):
        self._base_url = base_url.rstrip("/")
        self._token = token

    def _request(self, method: str, path: str, body: dict[str, Any] | None = None) -> dict[str, Any] | list[Any] | None:
        url = f"{self._base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Authorization", f"Bearer {self._token}")
        req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                raw = resp.read().decode()
                if not raw:
                    return None
                return json.loads(raw)
        except urllib.error.HTTPError as e:
            log.warning("API %s %s → %d: %s", method, path, e.code, e.read().decode()[:200])
            return None
        except Exception as e:
            log.error("API error %s %s: %s", method, path, e)
            return None

    def get(self, path: str) -> dict | list | None:
        return self._request("GET", path)

    def post(self, path: str, body: dict[str, Any], idempotency_key: str = "") -> dict | None:
        if idempotency_key:
            # Add via custom header hack — we'll use query param instead
            path = path  # Idempotency-Key passed in body header is tricky with urllib
        return self._request("POST", path, body)


class ReasoningWorker:
    """
    Main worker loop.

    1. Poll pending agent runs from API
    2. For each run, generate an intention via ModelGateway
    3. POST the intention back to the API
    """

    def __init__(self, api: APIClient, gateway: ModelGateway, poll_interval: float = 5.0):
        self._api = api
        self._gateway = gateway
        self._poll_interval = poll_interval
        self._running = False

    def run(self) -> None:
        self._running = True
        log.info("Reasoning worker started (poll_interval=%.1fs)", self._poll_interval)

        while self._running:
            try:
                self._poll_cycle()
            except Exception:
                log.exception("Poll cycle error")
            time.sleep(self._poll_interval)

        log.info("Reasoning worker stopped")

    def stop(self) -> None:
        self._running = False

    def _poll_cycle(self) -> None:
        # Fetch teams this worker token has access to
        teams = self._api.get("/api/auth/me")
        if not teams:
            return

        # Get team memberships
        me = self._api.get("/api/auth/me")
        if not me or not isinstance(me, dict):
            return

        # We don't have a "list pending runs" endpoint that's team-agnostic,
        # so we list runs per team. For the stub, we just iterate known teams.
        # In production, the API would expose /api/agent-runs?status=pending
        # For now, we demonstrate the loop structure.

        team_id = os.environ.get("TEAM_ID", "")
        if not team_id:
            return

        # List agent runs
        runs = self._api.get(f"/api/teams/{team_id}/agent-runs")
        if not runs or not isinstance(runs, list):
            return

        for run in runs:
            if run.get("status") not in ("pending", "running"):
                continue
            self._process_run(team_id, run)

    def _process_run(self, team_id: str, run: dict[str, Any]) -> None:
        run_id = run.get("id", "")
        agent_id = run.get("agent_id", "")
        log.info("Processing run %s for agent %s", run_id[:8], agent_id[:8])

        # Fetch tool grants for this agent
        grants = self._api.get(f"/api/teams/{team_id}/agents/{agent_id}/grants")
        if not grants or not isinstance(grants, list):
            grants = []

        # Use the model gateway to generate an intention
        try:
            intention = self._gateway.generate_intention(
                agent_run_id=run_id,
                context={"run": run},
                tool_grants=grants,
            )
        except Exception:
            log.exception("Model gateway error for run %s", run_id[:8])
            return

        # Validate
        errors = intention.validate()
        if errors:
            log.warning("Invalid intention for run %s: %s", run_id[:8], errors)
            return

        # POST intention to API
        payload = intention.to_api_payload()
        result = self._api.post(
            f"/api/teams/{team_id}/agent-runs/{run_id}/intentions",
            payload,
        )

        if result:
            log.info("Intention created for run %s: %s", run_id[:8], result.get("id", "?"))
        else:
            log.warning("Failed to create intention for run %s", run_id[:8])


# ─── Document Assist HTTP Server ───

class DocumentAssistServer:
    """
    Lightweight HTTP server for document assist requests.

    - Internal network only (port 9100, not exposed)
    - Requires WORKER_TOKEN in Authorization header
    - No DB/MinIO/NATS/Redis access
    - No logging of raw content
    """

    MAX_BODY_SIZE = 100_000  # 100KB max request body

    def __init__(self, port: int, token: str, gateway: StubDocumentAssistGateway):
        self._port = port
        self._token = token
        self._gateway = gateway
        self._server: ThreadingHTTPServer | None = None
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        handler = self._make_handler()
        self._server = ThreadingHTTPServer(("0.0.0.0", self._port), handler)
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)
        self._thread.start()
        log.info("Document assist server listening on port %d", self._port)

    def stop(self) -> None:
        if self._server:
            self._server.shutdown()
            self._server.server_close()
            log.info("Document assist server stopped")

    def _make_handler(self):
        token = self._token
        gateway = self._gateway
        max_body = self.MAX_BODY_SIZE

        class Handler(BaseHTTPRequestHandler):
            def _check_auth(self) -> bool:
                auth = self.headers.get("Authorization", "")
                if not auth.startswith("Bearer "):
                    return False
                return auth[7:] == token

            def _send_json(self, status: int, body: dict) -> None:
                data = json.dumps(body).encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            def do_POST(self) -> None:
                if self.path != "/document-assist":
                    self._send_json(404, {"error": "Not found"})
                    return

                if not self._check_auth():
                    self._send_json(401, {"error": "Unauthorized"})
                    return

                # Check body size
                content_length = int(self.headers.get("Content-Length", 0))
                if content_length > max_body:
                    self._send_json(413, {"error": "Request body too large"})
                    return

                try:
                    raw = self.rfile.read(content_length)
                    payload = json.loads(raw)
                except (json.JSONDecodeError, Exception):
                    self._send_json(400, {"error": "Invalid JSON"})
                    return

                # Validate required fields
                mode = payload.get("mode", "")
                mode_errors = validate_assist_mode(mode)
                if mode_errors:
                    self._send_json(400, {"error": mode_errors[0]})
                    return

                selected_text = payload.get("selected_text", "")
                instruction = payload.get("instruction", "")
                document_type = payload.get("document_type", "general_document")
                max_words = payload.get("max_words", 300)

                # Bounds validation
                if len(selected_text) > 20_000:
                    self._send_json(400, {"error": "selected_text exceeds 20000 chars"})
                    return
                if len(instruction) > 2_000:
                    self._send_json(400, {"error": "instruction exceeds 2000 chars"})
                    return
                if not (20 <= max_words <= 2_000):
                    self._send_json(400, {"error": "max_words must be 20-2000"})
                    return

                # Generate suggestions
                try:
                    result = gateway.generate_assist(
                        mode=mode,
                        selected_text=selected_text,
                        instruction=instruction,
                        document_type=document_type,
                        max_words=max_words,
                    )
                except Exception:
                    # Do NOT log raw content
                    log.error("Document assist gateway error for mode=%s", mode)
                    self._send_json(500, {"error": "Generation failed"})
                    return

                # Validate no forbidden fields
                forbidden = validate_assist_response(result)
                if forbidden:
                    self._send_json(500, {"error": "Invalid response"})
                    return

                # Do NOT log suggested text or raw content
                log.info("Document assist completed: mode=%s", mode)
                self._send_json(200, result)

            def do_GET(self) -> None:
                if self.path == "/health":
                    self._send_json(200, {"status": "ok"})
                else:
                    self._send_json(404, {"error": "Not found"})

            def log_message(self, format: str, *args: Any) -> None:
                # Suppress default access logs to prevent content leakage
                pass

        return Handler


# ─── Imports for HTTP server ───
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def main() -> None:
    base_url = os.environ.get("API_BASE_URL", "http://clarityit-api:8765")
    token = os.environ.get("WORKER_TOKEN", "")
    poll_interval = float(os.environ.get("POLL_INTERVAL", "5"))
    assist_port = int(os.environ.get("ASSIST_PORT", "9100"))

    if not token:
        log.error("WORKER_TOKEN environment variable is required")
        sys.exit(1)

    # Verify no forbidden credentials
    for forbidden in ("DATABASE_URL", "NATS_URL", "REDIS_URL", "MINIO_ENDPOINT"):
        if os.environ.get(forbidden):
            log.error("Reasoning worker must NOT have %s — violates isolation boundary", forbidden)
            sys.exit(1)

    api = APIClient(base_url, token)
    gateway = StubModelGateway()
    worker = ReasoningWorker(api, gateway, poll_interval)

    # Start document assist HTTP server in background thread
    assist_gateway = StubDocumentAssistGateway()
    assist_server = DocumentAssistServer(assist_port, token, assist_gateway)
    assist_server.start()

    # Graceful shutdown
    def handle_signal(signum: int, frame: Any) -> None:
        log.info("Received signal %d, shutting down", signum)
        worker.stop()
        assist_server.stop()

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    worker.run()


if __name__ == "__main__":
    main()
