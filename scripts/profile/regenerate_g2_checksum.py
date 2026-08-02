#!/usr/bin/env python3
"""
regenerate_g2_checksum.py — recompute the detached G2 manifest checksum.

Reads the committed Git BLOB (via `git cat-file blob HEAD:<path>`) — the
repository artifact CI tests — NOT the working-tree file, whose line endings
vary by platform. Writes the SHA-256 and byte size to the detached checksum
file that validate_g2.sh Step 0 asserts against.

Run this whenever the manifest content changes, then commit BOTH the manifest
and the checksum together. Step 0 will fail if only one is committed.
"""
import hashlib
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
MANIFEST = REPO / "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"
CHECKSUM = REPO / "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.sha256"
REL = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"


def main():
    # Read the committed blob, not the working-tree file.
    blob = subprocess.run(
        ["git", "cat-file", "blob", f"HEAD:{REL}"],
        capture_output=True, check=True,
    ).stdout
    sha = hashlib.sha256(blob).hexdigest()
    size = len(blob)

    # Also sanity-check the working-tree file matches the blob. If it doesn't,
    # the user likely has uncommitted manifest edits — warn but still write the
    # checksum for the committed blob (what CI will test).
    wt = MANIFEST.read_bytes()
    if hashlib.sha256(wt).hexdigest() != sha:
        print(f"WARNING: working-tree {REL} != committed blob", file=sys.stderr)
        print(f"  working-tree: {hashlib.sha256(wt).hexdigest()} ({len(wt)} bytes)", file=sys.stderr)
        print(f"  committed:    {sha} ({size} bytes)", file=sys.stderr)
        print("  Commit the manifest change first, then re-run this script.", file=sys.stderr)
        print("  The checksum below is for the COMMITTED blob (CI's view).", file=sys.stderr)

    CHECKSUM.write_text(
        "# Detached checksum for the G2 target manifest.\n"
        "# CI (validate_g2.sh Step 0) ASSERTS the committed Git blob's SHA-256 against\n"
        "# the value below — read via `git cat-file blob`, so it is the repository\n"
        "# artifact, not a platform-specific working-tree representation. Fail-closed.\n"
        "#\n"
        "# This is the authoritative identity. The receipt (G2-APPROVALS.md) must cite\n"
        "# exactly this value; a mismatch fails CI.\n"
        f"{MANIFEST.name}:{sha}\n"
        f"{MANIFEST.name}:size:{size}\n",
        encoding="utf-8",
    )
    print(f"checksum written: {CHECKSUM.relative_to(REPO)}")
    print(f"  {MANIFEST.name}:{sha}")
    print(f"  {MANIFEST.name}:size:{size}")


if __name__ == "__main__":
    main()
