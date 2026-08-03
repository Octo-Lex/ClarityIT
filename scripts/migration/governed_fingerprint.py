#!/usr/bin/env python3
"""Governed target fingerprint for G3 fresh-install / adoption convergence.

The raw profiler fingerprint (``capture_schema.fingerprint_of``) is
environment-dependent: it hashes ``migration_state`` (a row count and a
max timestamp), every non-system role including the bootstrap superuser,
and an opaque ``grants_sha256`` that bakes in the superuser's implicit
owner ACL lines and database-level grants.  Two fresh installs reach the
same raw fingerprint only by accident of identical ledger timestamps and
identical bootstrap identities, and a P3-adopted database (whose
extension owner is ``legacy_ext_owner``, not ``g3bootstrap``) can never
match a fresh install on the raw fingerprint.

This module defines a *governed* projection that hashes only the signed
G2 contract: the governed product and platform objects, the five target
roles and their memberships, the closed-world grants over the governed
object inventory, projected ownership (including the database owner),
effective default privileges, and the extension-owner invariant.  It
deliberately excludes the bootstrap/extension-owner identity, ledger row
values, row counts, and capture metadata.

The fresh-install governed fingerprint is the convergence target: both
fresh installs and the P3-adopted database must reach exact equality on
this projection.
"""
from __future__ import annotations

import hashlib
import json
import re
import uuid

# Versioned algorithm identity.  Bump when the projection composition
# changes; the A4 manifest and receipt record this so a fingerprint is
# always interpretable alongside its algorithm.
GOVERNED_ALGORITHM = "clarityit-g3-governed-v1"
GOVERNED_DOMAIN = "clarityit-g3-governed-v1\0"

# The G1-approved P3 golden fingerprint (source-profile allowlist entry).
P3_GOLDEN_FINGERPRINT = "cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"
# The signed G2 manifest records P1 as its source authority.
P1_FINGERPRINT_FIELD = "p1_fingerprint"

# Fixed G3 artifact date (deterministic; not an execution timestamp).
G3_ARTIFACT_DATE = "2026-08-03T00:00:00+00:00"

# Deterministic source-profile identifiers.  profile_id is a UUIDv5 over a
# fixed namespace and the P3 golden fingerprint, so every generation
# produces the same id without consulting a clock or database.
SOURCE_PROFILE_NAMESPACE = uuid.UUID("a4d9c3f1-7e2b-4a6f-9d18-3b5c8e1f0a27")
P3_PROFILE_ID = str(uuid.uuid5(SOURCE_PROFILE_NAMESPACE, f"clarityit:g3:source-profile:{P3_GOLDEN_FINGERPRINT}"))

# G1 approval reference (the commit that approved the P1/P2/P3 profile pack).
G1_APPROVAL_REF = "3b4a6fdeb35473e5f73ca74bafa479bd2648fb10"
# P3 fixture source commit (where the P3 schema.sql/seed.sql originate).
P3_SOURCE_COMMIT = "29c4cdcb4c7bd9f13209f5627b55f4fabbd08a33"


def _normalize(value) -> str:
    """Canonical JSON serialization for a single governed element."""
    return json.dumps(value, sort_keys=True, ensure_ascii=True, separators=(",", ":"))


def roles_digest(role_rows: list, membership_rows: list) -> str:
    """SHA-256 over the canonical serialization of governed roles + memberships.

    ``role_rows`` is a list of ``{name, flags}`` dicts (flags = the 7 booleans).
    ``membership_rows`` is a list of ``{member, role_of, admin_option,
    inherit_option, set_option}`` dicts.  The digest is taken over the
    sorted JSON array of ``[roles, memberships]`` so ordering does not
    matter and the formula is explicit and reproducible.
    """
    payload = [_normalize(role_rows), _normalize(membership_rows)]
    return hashlib.sha256(_normalize(payload).encode("utf-8")).hexdigest()


def _grant_material(cur, governed_schemas: tuple[str, ...], target_role_names: set[str]) -> list[str]:
    """Closed-world ACL material over the governed object inventory.

    Returns sorted ``object|schema|name|grantor_excluded|grantee|privilege|is_grantable``
    strings for relations, functions, schemas, and sequences in the
    governed schemas, restricted to grantees in the target role set plus
    PUBLIC.  The environment-dependent *grantor* is deliberately excluded:
    only object identity, grantee, privilege, and grantability are retained,
    so two databases with the same governed posture but different
    bootstrap grantors reach the same material.
    """
    material: list[str] = []
    schema_list = ",".join("'%s'" % s for s in governed_schemas)
    role_list = ",".join("'%s'" % r for r in (target_role_names | {"PUBLIC"}))

    # Relation grants (tables, sequences, views) in governed schemas.
    cur.execute(
        "SELECT n.nspname, c.relname, c.relkind, "
        "pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable "
        "FROM pg_class c "
        "JOIN pg_namespace n ON n.oid = c.relnamespace, "
        "aclexplode(c.relacl) a "
        "WHERE n.nspname IN (%s) AND c.relkind IN ('r','S','v','m','f','p') "
        "AND pg_get_userbyid(a.grantee) IN (%s) "
        "ORDER BY 1,2,3,4,5" % (schema_list, role_list)
    )
    for nsp, rel, kind, grantee, priv, grantable in cur.fetchall():
        material.append(f"rel|{nsp}|{rel}|{kind}|{grantee}|{priv}|{grantable}")

    # Function grants in governed schemas (application functions only;
    # extension objects are excluded by filtering to the target grantee
    # set, which never includes the extension installer).
    cur.execute(
        "SELECT n.nspname, p.proname, "
        "pg_get_function_identity_arguments(p.oid), "
        "pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable "
        "FROM pg_proc p "
        "JOIN pg_namespace n ON n.oid = p.pronamespace, "
        "aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a "
        "WHERE n.nspname IN (%s) AND p.prokind IN ('f','p','w') "
        "AND pg_get_userbyid(a.grantee) IN (%s) "
        "ORDER BY 1,2,3,4,5" % (schema_list, role_list)
    )
    for nsp, proname, args, grantee, priv, grantable in cur.fetchall():
        material.append(f"func|{nsp}|{proname}|{args}|{grantee}|{priv}|{grantable}")

    # Schema grants.
    cur.execute(
        "SELECT n.nspname, pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable "
        "FROM pg_namespace n, aclexplode(n.nspacl) a "
        "WHERE n.nspname IN (%s) "
        "AND pg_get_userbyid(a.grantee) IN (%s) "
        "ORDER BY 1,2,3" % (schema_list, role_list)
    )
    for nsp, grantee, priv, grantable in cur.fetchall():
        material.append(f"schema|{nsp}|{grantee}|{priv}|{grantable}")

    return sorted(set(material))


def _default_privileges_effective(cur, governed_schemas: tuple[str, ...], target_creators: set[str]) -> list[str]:
    """Effective default-privilege posture for governed creators/schemas.

    Returns sorted ``creator|schema|objtype|grantee|privilege`` strings,
    normalized so that the same effective posture yields the same material
    regardless of declaration order.  Restricted to default ACLs created by
    the governed owner role in the governed schemas.
    """
    creator_list = ",".join("'%s'" % c for c in target_creators)
    schema_list = ",".join("'%s'" % s for s in governed_schemas)
    cur.execute(
        "SELECT pg_get_userbyid(d.defaclrole), n.nspname, d.defaclobjtype, "
        "pg_get_userbyid(a.grantee), a.privilege_type "
        "FROM pg_default_acl d "
        "LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace, "
        "aclexplode(d.defaclacl) a "
        "WHERE pg_get_userbyid(d.defaclrole) IN (%s) "
        "AND n.nspname IN (%s) "
        "ORDER BY 1,2,3,4,5" % (creator_list, schema_list)
    )
    rows = []
    for creator, nsp, objtype, grantee, priv in cur.fetchall():
        rows.append(f"{creator}|{nsp}|{objtype}|{grantee}|{priv}")
    return sorted(set(rows))


def _projected_ownership(cur, governed_schemas: tuple[str, ...], app_signatures: set) -> dict:
    """Projected ownership of governed objects + database owner.

    Returns ``{database_owner, schemas: {schema: owner}, relations:
    {schema.relkind.name: owner}, functions: {schema.name(args): owner}}``.
    The database owner is included (correction #3: adopted P3 must end
    with ``clarityit_owner`` owning the database, not legacy_ext_owner).
    Database-level ACL grants remain excluded from the grant material.

    Only the **closed governed object inventory** is projected: product
    tables, the product sequence, and the signed application functions
    (plus platform tables/functions).  Extension-provided objects are
    excluded because their owner intentionally differs between fresh
    (``g3bootstrap``) and adopted (``legacy_ext_owner``) installations.
    """
    cur.execute("SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = current_database()")
    db_owner = cur.fetchone()[0]

    schema_list = ",".join("'%s'" % s for s in governed_schemas)
    cur.execute(
        "SELECT n.nspname, pg_get_userbyid(n.nspowner) "
        "FROM pg_namespace n WHERE n.nspname IN (%s) ORDER BY 1" % schema_list
    )
    schema_owners = {nsp: owner for nsp, owner in cur.fetchall()}

    cur.execute(
        "SELECT n.nspname, c.relname, c.relkind, pg_get_userbyid(c.relowner) "
        "FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace "
        "WHERE n.nspname IN (%s) AND c.relkind IN ('r','S','i','v','m','f','p') "
        "ORDER BY 1,2,3" % schema_list
    )
    relation_owners = {}
    for nsp, rel, kind, owner in cur.fetchall():
        relation_owners[f"{nsp}.{kind}.{rel}"] = owner

    cur.execute(
        "SELECT n.nspname, p.proname, "
        "pg_get_function_identity_arguments(p.oid), pg_get_userbyid(p.proowner) "
        "FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace "
        "WHERE n.nspname IN (%s) AND p.prokind IN ('f','p','w') "
        "ORDER BY 1,2,3" % schema_list
    )
    function_owners = {}
    for nsp, proname, args, owner in cur.fetchall():
        sig = (nsp, proname, args)
        # Include only governed functions: signed application functions
        # (public) or platform functions.  Extension-provided functions
        # are excluded (their owner intentionally differs fresh vs adopted).
        if sig in app_signatures or nsp == "platform":
            function_owners[f"{nsp}.{proname}({args})"] = owner

    return {
        "database_owner": db_owner,
        "schemas": schema_owners,
        "relations": relation_owners,
        "functions": function_owners,
    }


def _extension_owner_invariant(cur, required_extensions: tuple[str, ...], target_role_names: set[str]) -> dict:
    """Invariant: no required extension is owned by a target role.

    Returns ``{extname: owned_by_target_role}`` (a boolean per extension).
    The projection encodes ONLY the invariant — that no extension is owned
    by a target role — not the specific owner name.  Fresh installs and
    adopted P3 databases intentionally have different extension owners
    (``g3bootstrap`` vs ``legacy_ext_owner``); both are non-target
    identities, so the invariant (not the name) is what converges.
    """
    ext_list = ",".join("'%s'" % e for e in required_extensions)
    cur.execute(
        "SELECT e.extname, pg_get_userbyid(e.extowner) "
        "FROM pg_extension e WHERE e.extname IN (%s) ORDER BY 1" % ext_list
    )
    return {name: (owner in target_role_names) for name, owner in cur.fetchall()}


def governed_capture(cur, signed: dict, control: dict) -> dict:
    """Capture the governed projection from a live read-only cursor.

    ``signed`` is the parsed G2 target manifest; ``control`` is the parsed
    control-schema manifest.  The returned dict is the canonical input to
    :func:`governed_fingerprint`.
    """
    target_role_names = {r["name"] for r in signed["target_roles"]}
    governed_schemas = ("public", "platform")
    required_extensions = ("pgcrypto", "citext", "pg_trgm")

    # Governed roles (5) + memberships (2), projected onto the signed contract.
    cur.execute(
        "SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, "
        "rolcanlogin, rolreplication, rolbypassrls FROM pg_roles "
        "WHERE rolname = ANY(%s) ORDER BY rolname",
        (sorted(target_role_names),),
    )
    role_rows = [
        {
            "name": r[0],
            "flags": {
                "superuser": r[1], "inherit": r[2], "createrole": r[3],
                "createdb": r[4], "canlogin": r[5], "replication": r[6],
                "bypassrls": r[7],
            },
        }
        for r in cur.fetchall()
    ]

    cur.execute(
        "SELECT member.rolname, granted.rolname, am.admin_option, "
        "am.inherit_option, am.set_option "
        "FROM pg_auth_members am "
        "JOIN pg_roles member ON member.oid = am.member "
        "JOIN pg_roles granted ON granted.oid = am.roleid "
        "WHERE member.rolname = ANY(%s) AND granted.rolname = ANY(%s) "
        "ORDER BY 1,2",
        (sorted(target_role_names), sorted(target_role_names)),
    )
    membership_rows = [
        {
            "member": r[0], "role_of": r[1], "admin_option": r[2],
            "inherit_option": r[3], "set_option": r[4],
        }
        for r in cur.fetchall()
    ]

    # Product object shape (reuse the raw capture functions, scoped to
    # governed schemas; they carry no owner/ACL).
    import importlib.util
    from pathlib import Path
    profiler_path = Path(__file__).resolve().parents[1] / "profile" / "capture_schema.py"
    spec = importlib.util.spec_from_file_location("capture_schema_governed", profiler_path)
    cs = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(cs)

    relations = [r for r in cs.relations(cur, list(governed_schemas)) if r["schema"] in governed_schemas]
    # Project relations to the 4-field signed contract (drop composite type).
    relations_projected = [
        {k: r[k] for k in ("schema", "name", "kind", "persistence")} for r in relations
    ]
    columns = {k: v for k, v in cs.columns(cur, list(governed_schemas)).items()
               if k.split(".", 1)[0] in governed_schemas}
    constraints = {k: v for k, v in cs.constraints(cur, list(governed_schemas)).items()
                   if k.split(".", 1)[0] in governed_schemas}
    indexes = {k: v for k, v in cs.indexes(cur, list(governed_schemas)).items()
               if k.split(".", 1)[0] in governed_schemas}
    triggers = {k: v for k, v in cs.triggers(cur, list(governed_schemas)).items()
                if k.split(".", 1)[0] in governed_schemas}
    sequences = [s for s in cs.sequences(cur, list(governed_schemas)) if s["schema"] in governed_schemas]

    # Application functions (10), filtered to the signed signature set.
    app_signatures = {
        (row["schema"], row["name"], row["args"])
        for row in signed["target_grants"]["application_functions"]
    }
    all_functions = cs.functions(cur, list(governed_schemas))
    app_functions = [f for f in all_functions if (f["schema"], f["name"], f["args"]) in app_signatures]

    # Governed grants, projected ownership, default privileges, extension invariant.
    grants = _grant_material(cur, governed_schemas, target_role_names)
    ownership = _projected_ownership(cur, governed_schemas, app_signatures)
    default_privs = _default_privileges_effective(cur, governed_schemas, {"clarityit_owner"})
    ext_invariant = _extension_owner_invariant(cur, required_extensions, target_role_names)

    return {
        "algorithm": GOVERNED_ALGORITHM,
        "schemas": sorted(governed_schemas),
        "relations": relations_projected,
        "columns": columns,
        "constraints": constraints,
        "indexes": indexes,
        "triggers": triggers,
        "sequences": sequences,
        "application_functions": app_functions,
        "roles": role_rows,
        "memberships": membership_rows,
        "roles_digest": roles_digest(role_rows, membership_rows),
        "grants": grants,
        "default_privileges": default_privs,
        "ownership": ownership,
        "extension_owners": ext_invariant,
    }


def governed_fingerprint(capture: dict) -> str:
    """SHA-256 over the canonical serialization of the governed projection.

    The domain separator and algorithm version prefix the payload so the
    digest is bound to its definition and cannot be confused with any
    other SHA-256 in the G3 identity set.
    """
    payload = json.dumps(capture, sort_keys=True, ensure_ascii=True, separators=(",", ":")).encode("utf-8")
    h = hashlib.sha256()
    h.update(GOVERNED_DOMAIN.encode("utf-8"))
    h.update(payload)
    return h.hexdigest()


def assert_extension_owner_invariant(capture: dict, target_role_names: set[str]) -> None:
    """Raise AssertionError if any required extension is owned by a target role."""
    for extname, owned_by_target in capture["extension_owners"].items():
        if owned_by_target:
            raise AssertionError(
                f"governed extension-owner invariant violated: extension {extname} "
                f"is owned by a target role; extensions must be owned by a "
                f"non-target installer/legacy identity"
            )


def deterministic_source_profile(profile_id: str = P3_PROFILE_ID) -> dict:
    """The deterministic platform.source_profiles row for the P3 adoption.

    Every value is a generator constant or a deterministic derivation:
    ``profile_id`` is UUIDv5 over a fixed namespace and the P3 golden
    fingerprint; ``approved_at`` is the fixed G3 artifact date; the G1
    approval reference and P3 source commit are frozen constants.
    """
    return {
        "profile_id": profile_id,
        "schema_fingerprint": P3_GOLDEN_FINGERPRINT,
        "postgres_major": 16,
        "extensions": ["pgcrypto", "citext", "pg_trgm"],
        "roles_digest": None,  # filled at generation from the signed target roles
        "source_commit": P3_SOURCE_COMMIT,
        "approved_by": G1_APPROVAL_REF,
        "approved_at": G3_ARTIFACT_DATE,
    }
