#!/usr/bin/env python3
"""Phase 0 golden-vector generator for the G4 canonical JSON serializer.

Generates and validates candidate golden bytes that the Go
``CanonicalJSON`` implementation must reproduce. These vectors govern the
JSON canonicalization contract used by exactly two of the three G4
fingerprint algorithms:

  * the **source-profiler fingerprint** (capture_schema.fingerprint_of), and
  * the **governed fingerprint** (governed_fingerprint.governed_fingerprint).

Both use::

    json.dumps(value, sort_keys=True, ensure_ascii=True, separators=(",", ":"))

The **composite installation digest** is NOT governed by these vectors: it
is a binary-framed SHA-256 (domain prefix + length-prefixed labeled
components) with no JSON canonicalization step, and is covered by a
separate unit test in the digest port.

The two governed projections (source-profiler, governed) hash only ints,
strings, bools, arrays and objects. No float field is in scope; if one is
ever introduced it is a G4 stop, not a serialization fallback (see
``negative/README.md``).

Outputs, resolved relative to this file's parent directory:

* ``<name>.input.json``      pretty, UTF-8/LF, ``sort_keys=False`` (preserves
                             the visibly out-of-order construction).
* ``<name>.expected.bytes``  authoritative canonical bytes, ASCII, no
                             trailing newline.

Generation order (every vector, before any promotion):
  1. build input + canonical bytes in memory;
  2. re-serialize the in-memory input and assert byte-equality (self-consistency);
  3. stage both files;
  4. after the full set is written, reread every ``expected.bytes`` from disk,
     re-serialize its sibling ``.input.json`` value, and assert byte-equality
     (on-disk validation); only then is the set "promoted" (left in place).

The script makes no git commit. The produced files are generated golden
candidates, not repository-frozen artifacts, until committed later.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Environment report (the generator asserts CPython + stdlib json).
# ---------------------------------------------------------------------------
print(f"sys.version = {sys.version}")
print(f"sys.implementation.name = {sys.implementation.name}")
if sys.implementation.name != "cpython":
    raise SystemExit("STOP: golden vectors must be produced by CPython stdlib json")
print(f"json.__file__ = {json.__file__}")
# stdlib json check: a stdlib json must not carry a non-empty __package__ that
# points elsewhere; the file path itself is the strongest signal and is echoed
# above. We additionally guard against a stray third-party shadow by asserting
# the module is the one Python ships.
import importlib.util
spec = importlib.util.find_spec("json")
if spec is None or spec.origin is None or "Lib" not in str(spec.origin).replace("\\", "/"):
    # On Windows the stdlib path contains "Lib/json/__init__.py"; if a third
    # party shadowed json it would not. Report and stop rather than emit.
    raise SystemExit(f"STOP: json origin does not look like stdlib: {spec.origin if spec else None}")
print(f"json origin (validated stdlib) = {spec.origin}")


HERE = Path(__file__).resolve().parent
NEG = HERE / "negative"


def canonical_bytes(value) -> bytes:
    """The single canonicalization contract; ASCII bytes, no trailing newline."""
    return json.dumps(value, sort_keys=True, ensure_ascii=True, separators=(",", ":")).encode("ascii")


def pretty_input_bytes(value) -> bytes:
    """Pretty UTF-8/LF; sort_keys=False preserves constructed order."""
    return json.dumps(value, indent=2, sort_keys=False, ensure_ascii=False).encode("utf-8")


# ---------------------------------------------------------------------------
# Vector inputs. Positive vectors cover every dimension raised across the
# three plan rounds; negative vectors are informational-only float reprs.
# ---------------------------------------------------------------------------
POSITIVE: dict[str, object] = {
    # 1. key ordering: nested, out of order -> lexicographic by code point.
    "key_order": {
        "zeta": 1,
        "alpha": {"yankee": 2, "bravo": 3, "alpha": 4},
        "mike": [9, 8, 7],
        "Alpha": 5,  # uppercase code point < lowercase
    },
    # 2. non-ASCII keys (not just values).
    "non_ascii_keys": {
        "日本語": 1,
        "café": 2,
        "über": 3,
    },
    # 3. JSON-incompatible whitespace U+2028 / U+2029.
    "u2028_u2029": {
        "ls": "a\u2028b",
        "ps": "c\u2029d",
    },
    # 4. control characters \x00..\x1f.
    "control_chars": {
        "nul": "\x00",
        "bel": "\x07",
        "tab": "\t",
        "lf": "\n",
        "cr": "\r",
        "esc": "\x1b",
        "us": "\x1f",
    },
    # 5. quotes, backslashes, forward slash.
    "quotes_backslashes": {
        "quote": '"',
        "backslash": "\\",
        "slash": "/",
        "mixed": 'a"b\\c/d',
    },
    # 6. astral plane (emoji) -> surrogate pair \ud83d\ude00.
    "astral_plane": {"value": "\U0001f600"},
    # 7. int64 max and min (exact integer values).
    "int64_max_min": {
        "max": 9223372036854775807,
        "min": -9223372036854775808,
    },
    # 8. booleans and null.
    "booleans_null": {
        "t": True,
        "f": False,
        "n": None,
    },
    # 9. HTML chars: Python default does NOT escape -> Go must SetEscapeHTML(false).
    "html_chars": {"value": "<>&"},
    # 10. documents that no trailing newline is emitted (empty-string value).
    "trailing_newline": {"value": ""},
    # 11. end-to-end composite.
    "mixed_nested": {
        "schema": "public",
        "Relations": [{"Name": "z", "id": 2}, {"Name": "a", "id": 1}],
        "flags": {"nullable": True, "pk": False, "note": None},
        "count": 7,
        "tag": "café \U0001f600 <tag>",
    },
}

NEGATIVE: dict[str, object] = {
    # Informational only. No frozen projection contains a float; if one is
    # introduced it is a G4 stop. These record CPython float repr so the
    # decision can be made with evidence. These are BARE SCALAR JSON values
    # (not wrapped in objects): expected bytes are 1.0, 0.1, and 1e+100.
    "float_repr_1p0": 1.0,
    "float_repr_0p1": 0.1,
    "float_repr_1e100": 1e100,
}


def stage_vector(name: str, value: object, dest: Path) -> tuple[bytes, bytes]:
    """Build in memory, self-check, stage to disk. Returns (expected, input_pretty)."""
    expected = canonical_bytes(value)
    input_pretty = pretty_input_bytes(value)
    # (2) in-memory self-consistency: re-serialize the same value and compare.
    again = canonical_bytes(value)
    if again != expected:
        raise SystemExit(f"STOP: in-memory self-consistency failed for {name}")
    if expected.endswith(b"\n"):
        raise SystemExit(f"STOP: canonical bytes unexpectedly end with newline for {name}")
    # (3) stage both files.
    (dest / f"{name}.input.json").write_bytes(input_pretty)
    (dest / f"{name}.expected.bytes").write_bytes(expected)
    return expected, input_pretty


def validate_on_disk(name: str, value: object, dest: Path) -> bytes:
    """(4) reread the staged expected.bytes, re-serialize the input value,
    and assert byte-equality against both the reread file and the in-memory
    canonical bytes."""
    on_disk_expected = (dest / f"{name}.expected.bytes").read_bytes()
    on_disk_input = (dest / f"{name}.input.json").read_bytes()
    # Re-derive canonical from the pretty input's VALUE (not bytes) to mirror
    # how the Go test will load the JSON value then canonicalize.
    reloaded_value = json.loads(on_disk_input.decode("utf-8"))
    rederived = canonical_bytes(reloaded_value)
    canonical_value = canonical_bytes(value)
    if on_disk_expected != rederived:
        raise SystemExit(f"STOP: on-disk expected.bytes != rederived for {name}")
    if on_disk_expected != canonical_value:
        raise SystemExit(f"STOP: on-disk expected.bytes != canonical(value) for {name}")
    return on_disk_expected


def main() -> int:
    HERE.mkdir(parents=True, exist_ok=True)
    NEG.mkdir(parents=True, exist_ok=True)

    produced: dict[str, bytes] = {}

    # Stage positive vectors.
    for name, value in POSITIVE.items():
        expected, _ = stage_vector(name, value, HERE)
        produced[name] = expected

    # Stage negative vectors.
    for name, value in NEGATIVE.items():
        expected, _ = stage_vector(name, value, NEG)
        produced[name] = expected

    # (4) on-disk validation across the full set, before promotion.
    for name, value in POSITIVE.items():
        validate_on_disk(name, value, HERE)
    for name, value in NEGATIVE.items():
        validate_on_disk(name, value, NEG)

    # Informational notice for negative vectors (separate file; never prepended
    # to .input.json / .expected.bytes).
    (NEG / "README.md").write_bytes(
        (
            "# Informational negative vectors\n\n"
            "These record CPython `json` float formatting for evidence only. They are "
            "NOT to be matched by the Go serializer. The two JSON-canonicalizing "
            "projections (source-profiler fingerprint, governed fingerprint) use only "
            "ints, strings, bools, arrays and objects. If a frozen projection ever "
            "introduces a float, that is a G4 stop condition, not a serialization "
            "fallback. (The composite installation digest is binary-framed and not a "
            "JSON projection; it is covered by a separate test.)\n"
        ).encode("utf-8")
    )

    # Promotion report: highlighted expectations + count summary.
    print("")
    print("=== highlighted expected bytes ===")
    print(f"astral_plane   = {produced['astral_plane'].decode('ascii')}")
    print(f"html_chars     = {produced['html_chars'].decode('ascii')}")
    print(f"float_repr_1e100 = {produced['float_repr_1e100'].decode('ascii')}")
    print("")
    print("=== vector counts ===")
    print(f"positive vectors: {len(POSITIVE)}")
    print(f"negative vectors: {len(NEGATIVE)}")
    print(f"pair files (28 expected): {2 * (len(POSITIVE) + len(NEGATIVE))}")
    total_artifacts = 2 * (len(POSITIVE) + len(NEGATIVE)) + 2  # +gen_vectors.py +negative/README.md
    print(f"total Phase 0 artifacts (30 expected): {total_artifacts}")
    print("")
    print("Phase 0 generation complete. Files are generated and validated candidates (no commit made).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
