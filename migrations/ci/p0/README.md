# Repository-derived backend test schema

This directory contains a deterministic **P0 test fixture** used only by
GitHub Actions to provision PostgreSQL for backend tests.

It is not an approved P1 production profile, P2 restored-backup profile, or P3
fixture matched to P1. It is not the WP-00 G2 target manifest, the G3
reconciled baseline, a supported fresh-install path, or an upgrade path.

The fixture is generated from the repository migrations at the checked-out
commit with three narrowly scoped repairs:

- skip the earlier migration 005 agent schema and retain the later migration
  018 shape used by the current application;
- replace migration 016 with deterministic permission-name reconciliation;
- create the environment-level `clarityit_app` group role before migration 029
  grants privileges to it.

These choices make the repository-shaped database testable. They do not claim
that it matches any live database. G2/G3 remain blocked until the required
P1/P2 evidence and approvals exist.
