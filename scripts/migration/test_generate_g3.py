#!/usr/bin/env python3
"""Unit tests for deterministic G3 generation and fail-closed identities."""
from __future__ import annotations

import importlib.util
import json
import sys
import unittest
from pathlib import Path


HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("generate_g3_tested", HERE / "generate_g3.py")
assert SPEC is not None and SPEC.loader is not None
g3 = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = g3
SPEC.loader.exec_module(g3)

VERIFY_SPEC = importlib.util.spec_from_file_location("verify_g3_tested", HERE / "verify_g3.py")
assert VERIFY_SPEC is not None and VERIFY_SPEC.loader is not None
verify = importlib.util.module_from_spec(VERIFY_SPEC)
sys.modules[VERIFY_SPEC.name] = verify
VERIFY_SPEC.loader.exec_module(verify)


class GenerateG3Tests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.files_a, cls.a4_a = g3.expected_files()
        cls.files_b, cls.a4_b = g3.expected_files()

    def test_generation_is_byte_deterministic(self):
        self.assertEqual(self.files_a, self.files_b)
        self.assertEqual(self.a4_a, self.a4_b)

    def test_signed_manifest_uses_nested_table_shape(self):
        manifest = json.loads(g3.git_blob(g3.FROZEN_G2_COMMIT, g3.G2_MANIFEST_PATH))
        self.assertEqual(len(manifest["tables"]), 64)
        self.assertTrue(all(set(value) == {"columns", "constraints", "indexes", "triggers"} for value in manifest["tables"].values()))
        self.assertTrue({"columns", "constraints", "indexes", "triggers"}.isdisjoint(manifest))

    def test_legacy_archive_is_exactly_001_through_040(self):
        paths = g3.legacy_paths()
        self.assertEqual([int(Path(path).name[:3]) for path in paths], list(range(1, 41)))
        for path in paths:
            archived = self.files_a[g3.LEGACY_DIR / Path(path).name]
            self.assertEqual(archived, g3.git_blob(g3.FROZEN_G2_COMMIT, path))

    def test_baseline_emits_signed_object_counts(self):
        baseline = self.files_a[g3.BASELINE_SQL].decode()
        self.assertEqual(baseline.count("CREATE TABLE public."), 64)
        self.assertEqual(baseline.count("CREATE OR REPLACE FUNCTION public."), 10)
        self.assertEqual(baseline.count("CREATE SEQUENCE public."), 1)
        self.assertNotIn("REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public", baseline)
        self.assertIn("SET LOCAL search_path = pg_catalog, public;", baseline)

    def test_composite_rejects_each_single_component_tamper(self):
        components = [
            ("product_manifest_blob_sha256", g3.G2_MANIFEST_SHA256.encode("ascii")),
            ("control_manifest", self.files_a[g3.CONTROL_MANIFEST]),
            ("baseline_sql", self.files_a[g3.BASELINE_SQL]),
            ("seed_sql", self.files_a[g3.SEED_SQL]),
            ("role_bootstrap_sql", self.files_a[g3.ROLES_SQL]),
            ("legacy_checksum_inventory", self.files_a[g3.LEGACY_SUMS]),
        ]
        expected = g3.composite_digest(components)
        self.assertEqual(expected, self.a4_a["composite"]["sha256"])
        for index, (label, data) in enumerate(components):
            with self.subTest(component=label):
                tampered = list(components)
                tampered[index] = (label, data + b"tamper")
                self.assertNotEqual(g3.composite_digest(tampered), expected)

    def test_live_relations_ignore_only_unsigned_profiler_display_type(self):
        live = [{
            "schema": "public",
            "name": "work_items",
            "kind": "r",
            "persistence": "p",
            "type": "work_items",
        }]
        self.assertEqual(
            verify.signed_relation_projection(live),
            [{
                "schema": "public",
                "name": "work_items",
                "kind": "r",
                "persistence": "p",
            }],
        )


if __name__ == "__main__":
    unittest.main()
