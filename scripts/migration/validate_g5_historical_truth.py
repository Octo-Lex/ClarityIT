#!/usr/bin/env python3
"""Validate the frozen WP-00 historical-truth fixture contract.

G5 does not perform a v2 historical backfill. This validator proves that the
committed P3 synthetic fixture still carries the five classification cases that
WP-00 freezes for later migration work, and that none of those cases is labeled
as authoritative provider completion, Verification, Accepted outcome, or an
AuthorityGrant.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

SEED = Path("migrations/profiles/p3/seed.sql")

EXPECTED = {
    1: "legacy_unverified",
    2: "legacy_submitted_unverified",
    3: "legacy_decision_evidence",
    4: "legacy_outcome_unknown",
    5: "legacy_operator_assessment",
}

REQUIRED_FRAGMENTS = {
    1: ("agent_effect_results", "'succeeded'"),
    2: ("asset_actions", "proxmox_task_id", "'succeeded'"),
    3: ("approval_requests", "'approved'"),
    4: ("asset_actions", "'executing'"),
    5: ("action_outcomes", "operator_feedback"),
}

FORBIDDEN_CLASSIFICATIONS = {
    "provider_completed",
    "verified",
    "accepted",
    "authoritygrant",
    "verification",
}


def fail(message: str) -> None:
    print(f"HISTORICAL-TRUTH FAIL: {message}", file=sys.stderr)
    raise SystemExit(1)


def main() -> int:
    if not SEED.is_file():
        fail(f"missing fixture: {SEED}")

    text = SEED.read_text(encoding="utf-8")
    pattern = re.compile(
        r"^-- Legacy-truth case\s+(\d+):[^\n]*\(→\s*([A-Za-z0-9_]+)\)\s*$",
        re.MULTILINE,
    )
    found = {int(case): classification for case, classification in pattern.findall(text)}

    if set(found) != set(EXPECTED):
        fail(f"case set mismatch: found={sorted(found)} expected={sorted(EXPECTED)}")

    for case, expected in EXPECTED.items():
        actual = found[case]
        if actual != expected:
            fail(f"case {case} classification={actual!r}, expected={expected!r}")
        if actual.lower() in FORBIDDEN_CLASSIFICATIONS:
            fail(f"case {case} uses forbidden authoritative classification {actual!r}")

    # Check each case section still contains the synthetic evidence shape that
    # makes the classification meaningful, rather than only preserving a stale
    # comment. Sections end at the next case marker or EOF.
    markers = list(re.finditer(r"^-- Legacy-truth case\s+(\d+):", text, re.MULTILINE))
    for index, marker in enumerate(markers):
        case = int(marker.group(1))
        end = markers[index + 1].start() if index + 1 < len(markers) else len(text)
        section = text[marker.start():end]
        for fragment in REQUIRED_FRAGMENTS[case]:
            if fragment not in section:
                fail(f"case {case} missing required fixture fragment {fragment!r}")

    # The fixture is deliberately synthetic. A regression that removes this
    # statement weakens the evidence-hygiene guarantee and must fail G5.
    if "NO production data" not in text:
        fail("fixture no longer declares the no-production-data boundary")

    for case in sorted(EXPECTED):
        print(f"HISTORICAL-TRUTH-{case} {EXPECTED[case]} PASS")
    print("HISTORICAL-TRUTH PASS: 5 / 5; zero authoritative promotions")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
