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

PostgreSQL 16 stores three independent membership options per `pg_auth_members` row: `admin_option` (delegate the membership), `inherit_option` (use the role's privileges without `SET ROLE`), and `set_option` (execute `SET ROLE` to the role). Any option left unspecified on `GRANT` defaults to `TRUE`. See [`pg_auth_members`](https://www.postgresql.org/docs/16/catalog-pg-auth-members.html) and [`GRANT`](https://www.postgresql.org/docs/16/sql-grant.html). Because the unspecified default is `TRUE`, every target membership below states all three options explicitly.

| Member | Role | ADMIN | INHERIT | SET | SQL |
|---|---|---|---|---|---|
| `clarityit` | `clarityit_app` | FALSE | TRUE | FALSE | `GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE` |
| `clarityit_migrator` | `clarityit_owner` | FALSE | FALSE | TRUE | `GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE` |
| `clarityit_admin` | — | — | — | — | No memberships on app/owner roles (no ambient ACL) |

Rationale per row:

- **`clarityit` → `clarityit_app`: INHERIT TRUE, ADMIN FALSE, SET FALSE.** The application login inherits the app grant group's table/function/sequence privileges directly (no `SET ROLE` needed at runtime). It must not delegate `clarityit_app` membership (ADMIN FALSE) and has no operational reason to `SET ROLE clarityit_app` (SET FALSE).
- **`clarityit_migrator` → `clarityit_owner`: SET TRUE, ADMIN FALSE, INHERIT FALSE.** The migrator runs migrations by `SET ROLE clarityit_owner` (SET TRUE is the PG16 option that authorizes this — not ADMIN). It must not delegate owner membership (ADMIN FALSE): granting `ADMIN TRUE` would let the migrator hand out owner authority. `INHERIT FALSE` prevents ambient owner privileges so the migrator only holds owner powers during an explicit `SET ROLE`.
- **`clarityit_admin`: no app/owner membership.** Role administration is scoped by `CREATEROLE`; it must not acquire ambient application ACL.

### Ownership

All objects owned by `clarityit_owner` in the production target. `NOLOGIN` prevents direct connection; `clarityit_migrator` activates owner authority via `SET ROLE clarityit_owner`.

In the **development exception** (CT 150), `clarityit` remains the superuser owner for compatibility.

### Missing-role fail-closed

The migration runner MUST verify all required roles exist **before any table, index, grant, or bootstrap mutation**. If any role is missing or has incorrect flags, the runner fails with a clear diagnostic. A pre-existing superuser `clarityit` fails the production-target validator.

### PostgreSQL default PUBLIC EXECUTE on functions

All functions have implicit `PUBLIC EXECUTE` by default. The target posture revokes PUBLIC EXECUTE **per application-function signature** — never `REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC`, which would also strip PUBLIC EXECUTE from the 81 extension functions (pgcrypto, citext, pg_trgm) and break their operator classes and casts. The closed-world revocation set is the 10 application functions enumerated in the manifest.

For functions created **after** bootstrap, `ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC` ensures future application functions do not inherit the PUBLIC default. The default-privileges rule is owner-scoped (`clarityit_owner`), so it does not affect extension functions (owned by the extension installer). EXECUTE is then granted to `clarityit_app` per signature, and via default privileges for future functions created by `clarityit_owner`.

### Sequence privileges

Sequence privileges (`USAGE`, `SELECT`) are separate from table privileges. The target must grant `USAGE` on `audit_logs_id_seq` to `clarityit_app` if the application inserts into `audit_logs`.
