# G2 Decision — Migration 029 (Recommendation Evidence + Role Gap)

## Conflict

Migration 029 creates the `recommendation_evidence` table and then executes:
```sql
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;
```

No migration in the chain creates the `clarityit_app` role. The GRANT fails on fresh install (and likely failed in production, though the table itself was created successfully because DDL precedes the GRANT).

## P1 evidence

P1 inventory:
- **Roles:** Single role `clarityit` (superuser, canlogin, createdb, createrole, replication, bypassrls, inherit)
- **Memberships:** None
- **Ownership:** All 65 objects owned by `clarityit`
- **Default privileges:** None configured
- **`clarityit_app`:** Does NOT exist in P1
- **`recommendation_evidence`:** EXISTS in P1 with 16 columns (table creation succeeded; GRANT failed or was ignored)

The production database runs with the `clarityit` superuser role for all connections (confirmed by docker-compose: `POSTGRES_USER: clarityit`). There is no separate application role.

## Decision

**Separate privileged role bootstrap from application migration.** The reconciled baseline will:

### 1. Role inventory and posture

The P1 posture has a single `clarityit` superuser role. This is a development-only posture — production should use a least-privilege application role. The baseline will:

- **Define the runtime group role explicitly.** The baseline creates `clarityit_app` as a `NOLOGIN` group role (the intended grantee for table privileges).
- **Define the runtime login role explicitly.** The baseline creates `clarityit` as a `LOGIN` role that is a member of `clarityit_app`, with `INHERIT`.
- **Separate privileged bootstrap from application migration.** Role creation and privilege grants are in a privileged-bootstrap section that runs before the application schema. The application schema (migrations 001-040) assumes roles already exist.

### 2. The 029 GRANT

The reconciled baseline preserves migration 029's `GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app` — which now succeeds because `clarityit_app` is created in the bootstrap section.

### 3. Missing-role fail-closed behavior

The migration runner MUST verify that required roles exist before applying any migration that references them. If `clarityit_app` (or any other expected role) is missing, the runner fails with a clear diagnostic before attempting mutation.

### 4. Role/privilege posture verification

A post-baseline verification step will confirm:
- `clarityit_app` exists as a `NOLOGIN` group role
- `clarityit` exists as a `LOGIN` role with `INHERIT`
- `clarityit` is a member of `clarityit_app`
- All table grants reference existing roles

## Proof

A validation test will assert:
1. `clarityit_app` exists with `rolcanlogin = false`
2. `clarityit` exists with `rolcanlogin = true`, `rolinherit = true`
3. `clarityit` is a member of `clarityit_app` (via `pg_auth_members`)
4. Loading 029 without the bootstrap role fails with a role-not-found error
