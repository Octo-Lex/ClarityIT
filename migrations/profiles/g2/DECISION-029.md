# G2 Decision — Migration 029 (Recommendation Evidence + Role Gap) — Corrected v3

## Conflict

Migration 029 creates `recommendation_evidence` and executes:
```sql
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;
```

PostgreSQL raises an error (not silent) when GRANT targets a nonexistent role. No migration creates `clarityit_app`. Fresh install fails at 029.

## P1 evidence

- Single role: `clarityit` (superuser, canlogin, all flags)
- No memberships, no default privileges, all objects owned by `clarityit`
- `clarityit_app` does NOT exist

## Decision: five-role owner/migrator separation

**PostgreSQL ownership rules:** An object owner inherently controls its objects (SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER). `NOLOGIN` on `clarityit_owner` prevents direct connection; authorized `SET ROLE` by `clarityit_migrator` activates owner authority. This is correct behavior per [PostgreSQL ddl-priv](https://www.postgresql.org/docs/16/ddl-priv.html).

### Target role posture

| Role | Flags | Purpose |
|---|---|---|
| `clarityit_app` | `NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Application runtime grant group; owns nothing |
| `clarityit` | `LOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Application login identity; inherits only `clarityit_app` |
| `clarityit_owner` | `NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS` | Object owner; no administrative attributes; inherently controls owned objects |
| `clarityit_migrator` | `LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS` | May explicitly `SET ROLE clarityit_owner` to run migrations |
| `clarityit_admin` | `LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS` | Controlled role-administration identity; no ambient application ACL |

### Memberships

| Member | Role | Options |
|---|---|---|
| `clarityit` | `clarityit_app` | `INHERIT` (inherits app table privileges) |
| `clarityit_migrator` | `clarityit_owner` | `ADMIN` (can SET ROLE; NOINHERIT prevents ambient owner access) |
| `clarityit_admin` | — | No memberships on app/owner roles (no ambient ACL) |

### Ownership

All objects owned by `clarityit_owner` in the production target. `NOLOGIN` prevents direct connection; `clarityit_migrator` activates owner authority via `SET ROLE clarityit_owner`.

In the **development exception** (CT 150), `clarityit` remains the superuser owner for compatibility.

### Missing-role fail-closed

The migration runner MUST verify all required roles exist **before any table, index, grant, or bootstrap mutation**. If any role is missing or has incorrect flags, the runner fails with a clear diagnostic. A pre-existing superuser `clarityit` fails the production-target validator.

### PostgreSQL default PUBLIC EXECUTE on functions

All functions have implicit `PUBLIC EXECUTE` by default. The target posture must explicitly `REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC` and grant `EXECUTE` only to `clarityit_app` (or leave public for read-only utility functions if explicitly justified).

### Sequence privileges

Sequence privileges (`USAGE`, `SELECT`) are separate from table privileges. The target must grant `USAGE` on `audit_logs_id_seq` to `clarityit_app` if the application inserts into `audit_logs`.
