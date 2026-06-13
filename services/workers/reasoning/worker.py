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


def main() -> None:
    base_url = os.environ.get("API_BASE_URL", "http://clarityit-api:8765")
    token = os.environ.get("WORKER_TOKEN", "")
    poll_interval = float(os.environ.get("POLL_INTERVAL", "5"))

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

    # Graceful shutdown
    def handle_signal(signum: int, frame: Any) -> None:
        log.info("Received signal %d, shutting down", signum)
        worker.stop()

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    worker.run()


if __name__ == "__main__":
    main()
