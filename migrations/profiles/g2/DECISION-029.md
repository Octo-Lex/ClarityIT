# G2 Decision — Migration 029 (Recommendation Evidence + Role Gap) — Corrected

## Conflict

Migration 029 creates `recommendation_evidence` and executes:
```sql
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;
```

PostgreSQL raises an error (not silent failure) when GRANT targets a nonexistent role. No migration creates `clarityit_app`. Fresh install fails at 029.

## P1 evidence

P1 role posture:
- **Single role:** `clarityit` — `superuser=true, canlogin=true, createdb=true, createrole=true, replication=true, bypassrls=true, inherit=true`
- **Memberships:** none
- **Ownership:** all 65 objects owned by `clarityit`
- **Default privileges:** none
- **`clarityit_app`:** does NOT exist

**Critical:** If `clarityit` remains superuser and object owner, membership in `clarityit_app` provides no least-privilege containment — a superuser bypasses all GRANT-based access control.

## Decision

**Separate privileged role bootstrap from application migration with explicit least-privilege posture.**

### 1. Target role posture

| Role | Flags | Purpose |
|---|---|---|
| `clarityit_app` | `NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Application runtime group role (grantee of table privileges) |
| `clarityit` | `LOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Application login role, member of `clarityit_app` |
| `clarityit_owner` | `LOGIN CREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Migration/schema owner; owns all objects; NOT a member of `clarityit_app` |
| `clarityit_admin` | `LOGIN NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT` | Break-glass admin for role/policy management only |

**Memberships:**
- `clarityit` → member of `clarityit_app` (inherits table privileges)
- `clarityit_owner` → NOT a member of `clarityit_app` (cannot read/write app data via inheritance; must use SET ROLE if needed)
- `clarityit_admin` → NOT a member of `clarityit_app` (cannot read/write app data)

**Ownership:** All 65 objects owned by `clarityit_owner` (not `clarityit`).

### 2. Migration 029 GRANT

With `clarityit_app` created in the bootstrap, the GRANT succeeds:
```sql
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;
```

### 3. Missing-role fail-closed

The migration runner MUST verify required roles exist before applying any migration:
```sql
SELECT rolname FROM pg_roles WHERE rolname = 'clarityit_app';
-- If absent: FAIL with "Required role clarityit_app not found. Run bootstrap first."
```

### 4. Least-privilege note

In the **development exception** (CT 150), the current `clarityit` role remains a superuser for compatibility with the running application. The target posture above is the **production target** that G3 will implement. The reconciled baseline's bootstrap section defines the production posture; the development-exception waiver allows running with the existing superuser until production deployment.

### 5. Privilege review (for Security)

Security must review:
- `clarityit_app` has only the explicit GRANTs from migrations (SELECT/INSERT/UPDATE on specific tables)
- `clarityit_owner` owns objects but does NOT inherit app data access
- `clarityit_admin` can manage roles but NOT access application data
- No role has `SUPERUSER` in the production target

## Proof

A validation test will assert:
1. `clarityit_app` exists with `rolcanlogin = false`, `rolsuper = false`
2. `clarityit` exists with `rolcanlogin = true`, `rolsuper = false`, `rolinherit = true`
3. `clarityit` is a member of `clarityit_app`
4. Loading 029 without the bootstrap role fails with a role-not-found error
5. `clarityit_owner` is NOT a member of `clarityit_app`
