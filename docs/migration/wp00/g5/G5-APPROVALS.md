# WP-00 G5 — Acceptance Receipt

**Authorization:** `G5-AUTH-2026-08-10`  
**Decision date:** 2026-08-11  
**Decision:** **ACCEPTED upon integration of this receipt-only change**  
**Scope:** WP-00 G5 blocking CI matrix only. No G6 acceptance, provider, Site Runtime, or WP-01+ implementation is authorized by this receipt.

## 1. Accepted implementation and proof chain

| Item | Exact identity / result |
|---|---|
| G4 accepted prerequisite | `ecb0ea48eb67bc07371b72e11517a77ad802d465` |
| G5 authorization integration | `ea231810ba3b858a78cdb25850ab3e0fd407a3f1` |
| G5 implementation PR #20 candidate | `241cececbb74d2261cc696ade5c3aea09b6d2f8a` |
| G5 implementation squash on `main` | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| Current exact-main proof baseline | `d39c44fe942a786be43c1931f4047bf6a57df36e` |
| PostgreSQL image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |
| G5 workflow | `.github/workflows/g5-foundation.yml` |
| Ordinary CI workflow | `.github/workflows/ci.yml` |

The G5 implementation is additive CI-governance work. No accepted G4 migration-runner semantic was changed.

## 2. Exact candidate proof

The final pre-merge candidate `241cececbb74d2261cc696ade5c3aea09b6d2f8a` was proven by GitHub Actions:

- `WP-00 G5 Foundation Gate` run ID `31428517939` — **SUCCESS**.
- `CI` run ID `31428520644` — **SUCCESS**.
- `Backend (Go)` was part of the successful ordinary CI run and remained an unconditional blocking job.
- All four G5 fan-in dependencies passed: `backend-static`, database foundation matrix, historical truth, and artifact audit.
- The database foundation job retained the accepted G4 11-row matrix and required `ALL G4 EVIDENCE ROWS PASS`.

Retained candidate evidence digests:

| Evidence | SHA-256 |
|---|---|
| `g5-results` | `5eaf5f616d306bb066f94ff96ce5d3cd9721663bf6c4595c5945deec0c05efe5` |
| `g5-db-foundation` | `4b283b2e92d47b10b3496597da5f358b90e03b46b30621fab5f7defc4b38966b` |
| `g5-artifact-audit` | `2ac469f78fdc333596ae2cf676b03093f6b9ba08db7c8732a4180af0e14e4649` |
| `g5-historical-truth` | `cd98e3da5062d920b4eba3d8a9c7db62d26b01c207300b0a4594ffbd14852944` |

## 3. Exact-main proof

After implementation integration, the exact current `main` proof baseline `d39c44fe942a786be43c1931f4047bf6a57df36e` was observed green in GitHub Actions:

- `WP-00 G5 Foundation Gate` **run #11** — commit `d39c44f`, branch `main`, **SUCCESS**.
- `CI` **run #136** — commit `d39c44f`, branch `main`, **SUCCESS**.

The Actions UI evidence was supplied directly by the repository owner. The installed GitHub connector cannot enumerate push-triggered workflow runs, so the receipt records the visible immutable workflow/run numbers, exact commit, branch, and conclusion rather than inventing unavailable run IDs.

The ordinary `ci.yml` defines `Backend (Go)` as an unconditional job with no `continue-on-error`; therefore successful CI #136 includes successful `Backend (Go)` execution.

## 4. Required-status enforcement

Repository ruleset evidence is preserved as [`G5-RULESET-EVIDENCE.json`](G5-RULESET-EVIDENCE.json).

The exported GitHub ruleset has:

- ruleset ID `20672081`;
- name `WP-00 G5 Required Checks`;
- target `branch`;
- source `Octo-Lex/ClarityIT`;
- enforcement `active`;
- target condition `~DEFAULT_BRANCH` (repository default branch is `main`);
- empty bypass actor list;
- exactly one rule type: `required_status_checks`;
- exactly two required contexts:
  - `Backend (Go)`;
  - `G5 Foundation Gate`.

The client-supplied exported JSON SHA-256 is:

`e905241b71f259c83aa232d458d9851542ce058bb3649e37b3172ca6cb49634f`

No mandatory reviewer count, CODEOWNERS enforcement, signed-commit rule, linear-history rule, deployment approval, deletion restriction, force-push restriction, or other unrelated G5 control is represented in the exported ruleset.

`strict_required_status_checks_policy` is `false`; G5 did not require a branch to be updated with the latest `main` before merge. The frozen property is the fail-closed conjunction of the two required status contexts.

## 5. Frozen G5 CI classes

| Required class | Result | Evidence |
|---|---|---|
| `backend-static` | **PASS** | G5 fan-in; exact candidate checkout; changed-Go gofmt; full `go vet ./...` and `go build ./...`; CLI build |
| `db-fresh` | **PASS** | accepted G4 matrix G4-01 plus fan-in marker |
| `db-adopt-p3` | **PASS** | accepted G4 matrix G4-02 plus fan-in marker |
| `db-negative` | **PASS** | G4-03/G4-04 fail-closed coverage plus fan-in marker |
| `db-restart-lock` | **PASS** | G4-05/G4-06/G4-07 plus fan-in marker |
| `context-dead-letter` | **PASS** | separately required successful `Backend (Go)` suite |
| `backend-integration` | **PASS** | separately required successful `Backend (Go)` suite |
| `artifact-audit` | **PASS** | G3 static/generated checks, legacy checksum verification, G2 grant inventory and three-way receipt/checksum/blob binding |

Historical-truth safeguard: **PASS — 5/5; zero authoritative promotions**.

Accepted G4 regression oracle: **PASS — 11/11**.

Evidence hygiene: **PASS**; no production credentials or raw sensitive P1/P2 bytes were introduced into ordinary CI or this receipt.

## 6. Enforcement predicate

The effective `main` merge predicate is now:

```text
Backend (Go) == SUCCESS
AND
G5 Foundation Gate == SUCCESS
```

A green frontend or worker result cannot compensate for either required context failing or being absent.

## 7. Acceptance decisions

The authorization requested one transparent delegated assessment across the required G5 roles. These are role-based decisions, not three independent human attestations.

| Role | Decision | Basis |
|---|---|---|
| Backend | **APPROVE** | workflow implementation is deterministic, exact-SHA bound, and does not alter accepted G4 runner semantics |
| Quality | **APPROVE** | eight frozen CI classes, G4 11-row regression oracle, exact-main proof, and fail-closed fan-in are evidenced |
| Security | **APPROVE** | historical-truth guard, secret/evidence hygiene, artifact binding, and no-bypass required-status enforcement are evidenced |

## 8. Final G5 decision

Every frozen G5 requirement is satisfied:

1. G4 remained the accepted prerequisite.
2. G5 authorization was integrated before implementation.
3. Exact candidate proof passed both G5 and ordinary CI.
4. G5 implementation was integrated on `main`.
5. Exact-main G5 and ordinary CI proof passed on `d39c44fe942a786be43c1931f4047bf6a57df36e`.
6. The required-status ruleset is active on `main`, has no bypass actors, and requires exactly `Backend (Go)` and `G5 Foundation Gate`.
7. Historical-truth and evidence-hygiene safeguards are blocking.
8. No accepted G1-G4 artifact or identity was rewritten.

**Decision: G5 ACCEPTED.**

This decision becomes durable when this receipt and the project completion authority update are integrated together. G6 is separately authorized by `G6-AUTH-2026-08-11` and becomes active from that accepted G5 integration boundary. G6 itself remains unaccepted until AC-00-01 through AC-00-30 and the frozen WS6 evidence contract pass.
