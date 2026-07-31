#!/usr/bin/env python3
"""
Integration test for migration_state() — proves the catalog-driven column
detection + savepoint fix works against a real database with a ledger table
that uses `applied_at` (not `created_at`).

Also proves the transaction is not left aborted after a failed probe.

Run inside the Docker network where postgres:5432 resolves:
    PGPASSWORD=clarityit python scripts/profile/test_migration_state.py
"""
from __future__ import annotations

import os
import sys
import unittest

import psycopg2  # type: ignore

_HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _HERE)
import importlib.util

_spec = importlib.util.spec_from_file_location(
    "capture_schema", os.path.join(_HERE, "capture_schema.py")
)
cs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cs)

DB_HOST = os.environ.get("DB_HOST", "postgres")
DB_PORT = int(os.environ.get("DB_PORT", "5432"))
DB_USER = os.environ.get("DB_USER", "clarityit")
DB_NAME = os.environ.get("DB_NAME", "clarityit")
DB_PASS = os.environ.get("PGPASSWORD", "clarityit")


@unittest.skipUnless(
    os.environ.get("DB_AVAILABLE") or os.environ.get("PGPASSWORD"),
    "requires PGPASSWORD or DB_AVAILABLE env (run in Docker network)",
)
class TestMigrationState(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # Use a writable connection for setup (CREATE TABLE etc.)
        cls.wconn = psycopg2.connect(
            host=DB_HOST, port=DB_PORT, dbname=DB_NAME, user=DB_USER, password=DB_PASS
        )
        cls.wconn.autocommit = True
        cls.wcur = cls.wconn.cursor()

    @classmethod
    def tearDownClass(cls):
        cls.wcur.close()
        cls.wconn.close()

    def setUp(self):
        """Create throwaway ledger tables for each test."""
        self.wcur.execute("DROP SCHEMA IF EXISTS platform CASCADE")
        self.wcur.execute("DROP TABLE IF EXISTS schema_migrations")
        self.wcur.execute("CREATE SCHEMA platform")

    def _ro_cursor(self):
        """A read-only cursor (as the profiler uses) for migration_state calls."""
        conn = psycopg2.connect(
            host=DB_HOST, port=DB_PORT, dbname=DB_NAME, user=DB_USER, password=DB_PASS
        )
        conn.autocommit = False
        cur = conn.cursor()
        cur.execute("SET TRANSACTION READ ONLY")
        return conn, cur

    def test_detects_applied_at_column(self):
        """The fix must detect `applied_at` (not hardcoded created_at)."""
        self.wcur.execute("""
            CREATE TABLE platform.schema_revisions (
                version INT PRIMARY KEY, name TEXT, checksum TEXT,
                applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )
        """)
        self.wcur.execute("""
            INSERT INTO platform.schema_revisions (version, name, checksum, applied_at)
            VALUES (1, 'test_rev', 'abc123', '2026-01-01T00:00:00Z')
        """)
        conn, cur = self._ro_cursor()
        try:
            result = cs.migration_state(cur)
            self.assertEqual(result["table"], "platform.schema_revisions")
            self.assertEqual(result["row_count"], 1)
            self.assertEqual(result["latest_column"], "applied_at")
            self.assertIn("2026-01-01", str(result["latest_recorded_at"]))
        finally:
            conn.rollback()
            conn.close()

    def test_transaction_not_aborted_after_probe(self):
        """After migration_state runs, the transaction must still be usable."""
        self.wcur.execute("CREATE TABLE schema_migrations (version INT PRIMARY KEY, created_at TIMESTAMPTZ)")
        self.wcur.execute("INSERT INTO schema_migrations (version, created_at) VALUES (1, NOW())")
        conn, cur = self._ro_cursor()
        try:
            cs.migration_state(cur)
            # If aborted, this SELECT raises InFailedSqlTransaction
            cur.execute("SELECT 1")
            self.assertEqual(cur.fetchone()[0], 1)
        finally:
            conn.rollback()
            conn.close()

    def test_handles_table_without_timestamp(self):
        """A ledger with no timestamp column should not crash or abort."""
        self.wcur.execute("CREATE TABLE schema_migrations (version INT PRIMARY KEY)")
        self.wcur.execute("INSERT INTO schema_migrations (version) VALUES (42)")
        conn, cur = self._ro_cursor()
        try:
            result = cs.migration_state(cur)
            self.assertEqual(result["table"], "schema_migrations")
            self.assertEqual(result["row_count"], 1)  # one row inserted
            cur.execute("SELECT 1")  # still alive
            self.assertEqual(cur.fetchone()[0], 1)
        finally:
            conn.rollback()
            conn.close()


if __name__ == "__main__":
    unittest.main(verbosity=2)
