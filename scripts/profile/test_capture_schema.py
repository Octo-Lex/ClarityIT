#!/usr/bin/env python3
"""
Unit tests for the WP-00 schema capture fingerprint logic.

These tests do NOT require a live database. They validate:
  1. Self-consistency: a manifest's stored fingerprint reproduces itself.
  2. fingerprint_sha256 is excluded from its own computation.
  3. Ownership is excluded from the fingerprint (spec §4.3).
  4. Determinism: identical schema → identical fingerprint regardless of
     capture timestamp, label, row counts, or ownership.
  5. Sensitivity: any schema change changes the fingerprint.
  6. P1==P2: two captures of an identical schema (different label/timestamp)
     produce the same fingerprint.

Run: python -m unittest scripts.profile.test_capture_schema
     or: python scripts/profile/test_capture_schema.py
"""
import copy
import hashlib
import importlib.util
import json
import os
import sys
import unittest

# Import the profiler module by path (it's not a package)
_HERE = os.path.dirname(os.path.abspath(__file__))
_spec = importlib.util.spec_from_file_location(
    "capture_schema", os.path.join(_HERE, "capture_schema.py")
)
cs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cs)


def _base_manifest():
    """A minimal but representative manifest covering all fingerprinted sections."""
    return {
        "profiler_version": "3.2.0-p1p2",
        "captured_at_utc": "2026-07-31T00:00:00+00:00",
        "source_label": "test",
        "fingerprint_sha256": "",  # will be set; must NOT affect recompute
        "postgres": {
            "settings": {"TimeZone": "UTC"},
            "extensions": [{"name": "pgcrypto", "version": "1.3"}],
        },
        "pg_version_string": "PostgreSQL 16.4 on x86_64-pc-linux-musl",
        "schemas": ["public"],
        "relations": [
            {"schema": "public", "name": "users", "kind": "r",
             "persistence": "p", "type": "users"}
        ],
        "columns": {
            "public.users": [
                {"name": "id", "type": "uuid", "not_null": True,
                 "default": "gen_random_uuid()", "identity": ""},
                {"name": "email", "type": "text", "not_null": True,
                 "default": None, "identity": ""},
            ]
        },
        "constraints": {
            "public.users": [
                {"name": "users_pkey", "type": "p", "definition": "PRIMARY KEY (id)"}
            ]
        },
        "indexes": {
            "public.users": [
                {"name": "users_pkey",
                 "definition": "CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id)",
                 "unique": True, "primary": True}
            ]
        },
        "sequences": [
            {"schema": "public", "name": "users_id_seq", "type": "bigint",
             "start": 1, "increment": 1, "max": 9223372036854775807,
             "min": 1, "cache": 1, "cycle": False}
        ],
        "functions": [
            {"schema": "public", "name": "set_updated_at", "args": "",
             "body": "CREATE FUNCTION set_updated_at() ..."},
            {"schema": "public", "name": "do_thing", "args": "integer",
             "body": "CREATE FUNCTION do_thing(integer) ..."},
            {"schema": "public", "name": "do_thing", "args": "text",
             "body": "CREATE FUNCTION do_thing(text) ..."},
        ],
        "triggers": {},
        "views": [],
        "rls_policies": [],
        "rls_state": [],
        "migration_state": {"table": None, "note": "none"},
        "roles_and_grants": {
            "roles": [{"name": "clarityit", "superuser": True}],
            "memberships": [],
            "grants_sha256": "abc123",
            "default_privileges": [],
        },
        "ownership": [
            {"schema": "public", "relation": "users", "owner": "clarityit"}
        ],
        "integrity_checks": {"orphan_foreign_keys": [], "invalid_constraints": []},
        "row_counts": {"public.users": 100},
        "schema_dump_sha256": "deadbeef",
        "schema_dump_error": None,
    }


class TestSelfConsistency(unittest.TestCase):
    def test_stored_fingerprint_reproduces_itself(self):
        """The core fix: fingerprint_sha256 must be excluded so recompute matches."""
        m = _base_manifest()
        m["fingerprint_sha256"] = cs.fingerprint_of(m)
        recomputed = cs.fingerprint_of(m)
        self.assertEqual(
            m["fingerprint_sha256"], recomputed,
            "stored fingerprint must reproduce itself on recompute",
        )

    def test_fingerprint_excludes_itself(self):
        """Changing fingerprint_sha256 must NOT change the recomputed fingerprint."""
        m = _base_manifest()
        m["fingerprint_sha256"] = "aaa"
        fa = cs.fingerprint_of(m)
        m["fingerprint_sha256"] = "bbb"
        fb = cs.fingerprint_of(m)
        self.assertEqual(fa, fb, "fingerprint_sha256 must not affect the fingerprint")


class TestOwnershipExclusion(unittest.TestCase):
    def test_ownership_excluded_from_fingerprint(self):
        """Spec §4.3: ownership must not affect the fingerprint."""
        m = _base_manifest()
        fa = cs.fingerprint_of(m)
        m2 = copy.deepcopy(m)
        m2["ownership"][0]["owner"] = "different_owner"
        fb = cs.fingerprint_of(m2)
        self.assertEqual(fa, fb, "ownership must not affect the fingerprint")


class TestPgVersionStringExclusion(unittest.TestCase):
    def test_pg_version_string_excluded_from_fingerprint(self):
        """The build-specific version string must not affect the fingerprint.
        This is what caused the CI golden mismatch (16.4 local vs 16.x CI)."""
        m = _base_manifest()
        m["pg_version_string"] = "PostgreSQL 16.4 on x86_64-pc-linux-musl"
        fa = cs.fingerprint_of(m)
        m["pg_version_string"] = "PostgreSQL 16.99 on x86_64-pc-linux-musl, different compiler"
        fb = cs.fingerprint_of(m)
        self.assertEqual(fa, fb, "pg_version_string must not affect the fingerprint")


class TestDeterminism(unittest.TestCase):
    def test_timestamp_label_rowcounts_irrelevant(self):
        """Two captures of the same schema with different volatile fields match."""
        a = _base_manifest()
        a["captured_at_utc"] = "2026-07-31T10:00:00+00:00"
        a["source_label"] = "P1-production"
        a["row_counts"] = {"public.users": 42}

        b = _base_manifest()
        b["captured_at_utc"] = "2026-12-31T23:59:59+00:00"
        b["source_label"] = "P2-restored"
        b["row_counts"] = {"public.users": 999}

        self.assertEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))

    def test_p1_equals_p2_on_identical_schema(self):
        """The whole point: P1 vs P2 of the same schema share a fingerprint."""
        p1 = _base_manifest()
        p1["source_label"] = "P1-production"
        p1["fingerprint_sha256"] = cs.fingerprint_of(p1)

        p2 = copy.deepcopy(p1)
        p2["source_label"] = "P2-restored"
        p2["captured_at_utc"] = "2026-07-31T12:00:00+00:00"
        p2["fingerprint_sha256"] = ""  # will recompute
        p2["fingerprint_sha256"] = cs.fingerprint_of(p2)

        self.assertEqual(p1["fingerprint_sha256"], p2["fingerprint_sha256"])


class TestSensitivity(unittest.TestCase):
    def test_column_change_detected(self):
        a = _base_manifest()
        b = copy.deepcopy(a)
        b["columns"]["public.users"][1]["type"] = "varchar(255)"
        self.assertNotEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))

    def test_grant_change_detected(self):
        a = _base_manifest()
        b = copy.deepcopy(a)
        b["roles_and_grants"]["grants_sha256"] = "different"
        self.assertNotEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))

    def test_rls_policy_change_detected(self):
        a = _base_manifest()
        b = copy.deepcopy(a)
        b["rls_policies"].append({
            "schema": "public", "table": "users", "name": "p",
            "cmd": "*", "permissive": True, "using": "true",
            "with_check": None, "roles": [],
        })
        self.assertNotEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))

    def test_sequence_property_change_detected(self):
        a = _base_manifest()
        b = copy.deepcopy(a)
        b["sequences"][0]["increment"] = 2
        self.assertNotEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))

    def test_overloaded_function_body_change_detected(self):
        """Overloads are totally ordered by args; body changes are detected."""
        a = _base_manifest()
        b = copy.deepcopy(a)
        # Change only the (text) overload's body, not the (integer) one
        for f in b["functions"]:
            if f["name"] == "do_thing" and f["args"] == "text":
                f["body"] = "DIFFERENT BODY"
        self.assertNotEqual(cs.fingerprint_of(a), cs.fingerprint_of(b))


class TestCanonicalization(unittest.TestCase):
    def test_canonical_is_compact_sorted(self):
        m = _base_manifest()
        canon = cs.canonicalize(m)
        self.assertNotIn(b", ", canon)
        self.assertNotIn(b": ", canon)

    def test_excluded_fields_not_in_canonical(self):
        m = _base_manifest()
        m["fingerprint_sha256"] = "SHOULD_NOT_APPEAR"
        m["pg_version_string"] = "SHOULD_NOT_APPEAR_EITHER"
        canon = cs.canonicalize(m)
        self.assertNotIn(b"fingerprint_sha256", canon)
        self.assertNotIn(b"SHOULD_NOT_APPEAR", canon)
        self.assertNotIn(b"ownership", canon)
        self.assertNotIn(b"row_counts", canon)
        self.assertNotIn(b"pg_version_string", canon)


if __name__ == "__main__":
    unittest.main(verbosity=2)
