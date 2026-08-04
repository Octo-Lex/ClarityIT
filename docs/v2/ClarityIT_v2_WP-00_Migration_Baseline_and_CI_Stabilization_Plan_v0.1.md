# ClarityIT v2 — WP-00: Migration Baseline and CI Stabilization

*Migration Baseline and CI Stabilization*

**Document type:** Implementation Work Package

**Version:** 0.1

**Status:** Draft for execution approval

**Date:** 30 July 2026

**Repository:** Octo-Lex/ClarityIT

**Specified source baseline:** main at b9d15877583e84c45ab5478dfaa6087966926fc5

**Revision starting baseline:** main@318b4eba6ee298013fa8e949ba5d34d5f7d7dc30 (history: `b9d1587` → `995d9e0` [PR#2 CPU-loop fix] → `318b4eb` [PR#3 v2 docs]). Merging the durable-disposition follow-up advances main again.

> **Revision note (2026-07-30):** This document was originally authored against `main=b9d1587` with PR#2 (`cad4230`) open. Since then PR#2 merged as squash `995d9e0` (bounded termination, but **without** the durable poison-event disposition its own AC-00-03 required) and PR#3 merged as `318b4eb` (v2 docs). References to `b9d1587`/`cad4230`/PR#2 below are retained as historical provenance where they describe the original baseline; where they describe *current* state they are corrected in place. The durable-disposition debt is discharged by the `fix/context-worker-durable-dlq` follow-up, which is a precondition for G0 acceptance.

**Phase boundary:** Compatibility Specification phases 0-1 only

**Exit decision:** Migration foundation accepted or blocked; no partial pass

> **Work-package definition** WP-00 establishes one reproducible source and database baseline, a fail-closed migration mechanism, and blocking green backend CI. It removes uncertainty from the foundation before any v2 execution-kernel, adapter, Site Runtime, host-agent, or experience implementation begins.

## Decision snapshot

- Start from the specified source implementation at b9d1587 *(historical baseline; revision starting baseline is `main@318b4eb` per the revision note above)*. Do not declare the operational freeze until every deployed byte is reconciled into one tagged source artifact. PR \#2 has merged as squash `995d9e0`.

- Do not edit and replay legacy migrations 001-040. Preserve them unchanged as historical provenance; fresh installations use one reconciled baseline followed by immutable checksummed forward migrations.

- Treat the approved production schema profile, not migration filenames or README counts, as the authority for upgrading an existing database.

- Support only explicitly approved source profiles. An unknown fingerprint fails before any DDL, data write, or automatic repair.

- Resolve migrations 016, 018, and 029 through recorded schema decisions grounded in P1/P2 evidence; do not guess the production shape.

- Implement one Go migration command as the only supported schema-change path. It owns locking, checksums, ledger records, adoption, apply, restart, and verification.

- Make backend build, fresh install, approved-profile adoption, restart, negative-profile, and existing Go tests blocking. Remove continue-on-error only after the complete matrix passes.

- ~~Merge or supersede PR \#2 only after failed poison messages are durably recorded before terminal acknowledgement.~~ **(Historical — PR#2 merged as `995d9e0` with bounded termination but without durable disposition; the `fix/context-worker-durable-dlq` follow-up supplies the missing durable poison-event disposition that this requirement originally mandated before G0 acceptance.)**

- Preserve legacy execution records as claims. WP-00 must establish a test fixture proving no legacy value maps to provider_completed, verified, or accepted.

- WP-00 passes as one package. Missing production profile evidence, an unreconciled deployment delta, a manual SQL correction, or a non-blocking backend job is a release blocker.

## 1. Authority, purpose, and outcomes

This work package implements phases 0 and 1 of the ClarityIT v2 v1-to-v2 Compatibility and Migration Specification v0.1. The Product Definition governs scope and outcome; the Authoritative Execution Kernel Specification governs later execution semantics; the Compatibility and Migration Specification governs source adoption, historical truth, baseline construction, and release gates.

> **Primary outcome** A reviewer can reproduce the exact WP-00 result from a frozen release artifact: profile an approved PostgreSQL 16 source, restore its backup, adopt or install the reconciled baseline without replaying migrations 001-040, apply immutable forward revisions, verify the target fingerprint, and obtain a blocking green CI result without manual database correction.

### 1.1 Binding outcomes

1.  One source freeze manifest identifies the repository commit, operational delta, release tag, dependency locks, container image digests, and configuration schema used by WP-00.

2.  P1, P2, and P3 source profiles are captured, normalized, fingerprinted, approved, and tied to a restore proof.

3.  Conflicts in legacy migrations 016, 018, and 029 are resolved by recorded decisions against the approved live shape.

4.  A clean PostgreSQL 16 installation produces the exact reconciled target schema from a single baseline plus forward revisions.

5.  An approved existing source is adopted by fingerprint and upgraded without replaying legacy files or editing source data by hand.

6.  The migration runner is fail-closed, checksummed, locked, restartable, observable, and unable to invoke providers.

7.  Backend CI is blocking and green; the current non-blocking exception is removed.

8.  The evidence pack proves provenance, restore, schema equivalence, failure recovery, CI results, and final acceptance.

### 1.2 Definition of done

| **ID**     | **Required result**                                                | **Proof**                                                                                       |
|------------|--------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| **DOD-01** | Empty PostgreSQL 16 installs reproducibly.                         | Two clean runs yield the same schema fingerprint and revision set.                              |
| **DOD-02** | P2 restored backup upgrades without manual data or DDL correction. | Recorded runner transcript, before/after profiles, and reconciliation report.                   |
| **DOD-03** | Unknown or drifted source fails before DDL.                        | Negative test shows unchanged pre/post schema fingerprint.                                      |
| **DOD-04** | Interrupted application resumes safely.                            | Fault-injection tests show no duplicate revision, partial target object, or checksum drift.     |
| **DOD-05** | Legacy truth is not inflated.                                      | Golden classification fixture contains zero provider_completed, verified, or accepted mappings. |
| **DOD-06** | Operational delta is represented in source.                        | Production-byte comparison and resolved PR \#2 evidence are in the freeze manifest.             |
| **DOD-07** | Backend CI is blocking and green.                                  | No continue-on-error; all required checks succeed on the final commit.                          |
| **DOD-08** | No unresolved severity 1 or 2 foundation defect remains.           | Signed defect disposition and release decision.                                                 |

### 1.3 Scope boundary

| **Included**                                                | **Explicitly excluded**                                             |
|-------------------------------------------------------------|---------------------------------------------------------------------|
| Repository/deployment freeze and PR \#2 reconciliation      | Execution-kernel domain tables and live state machines              |
| Read-only schema profiling and backup-restore rehearsal     | Generic compute adapter or Proxmox v2 implementation                |
| Legacy schema conflict decisions and clean baseline         | Site Runtime, host sensors, or target-side AI agents                |
| Migration ledger, lock, checksum, adoption, apply, verify   | Backfill, journal catch-up, production cutover, or contract cleanup |
| P3 fixtures, CI matrix, failure injection, release evidence | Frontend redesign, new product surfaces, or provider mutations      |
| Legacy-truth classification contract fixture                | Promotion of historical claims into v2 Verification or outcomes     |

> **Follow-on boundary:** No Native Pattern implementation or WP-01 through
> WP-10 feature work is part of WP-00. G6 remains the prerequisite for those
> packages. CT 150 evidence custody may satisfy only the signed development
> exception; it is not production-readiness evidence and cannot be promoted in
> place.

## 2. Confirmed starting baseline

The source implementation was rechecked on 30 July 2026. *(Historical: main resolved to `b9d1587` at original authoring. Current revision starting baseline is `main@318b4eb`.)* The repository contains 40 legacy SQL files and 64 nominal unique table definitions. The compatibility specification remains authoritative where it deliberately supersedes older repository guidance.

| **Dimension**             | **Confirmed condition**                                  | **WP-00 consequence**                                                         |
|---------------------------|----------------------------------------------------------|-------------------------------------------------------------------------------|
| **Repository**            | Octo-Lex/ClarityIT; revision starting baseline `main@318b4eb` (was `b9d1587`)                         | Use as specified logical source reference.                                    |
| **Database**              | PostgreSQL 16                                            | All baseline, restore, and CI scenarios target PostgreSQL 16.                 |
| **Legacy migration path** | Sorted psql loop; no durable ledger/checksum/lock policy | Retire as a supported execution path.                                         |
| **CI**                    | Backend job has continue-on-error: true                  | Replace with required blocking checks.                                        |
| **Fresh install defect**  | Issue \#1 remains open                                   | Cannot close until WP-00 gates pass.                                          |
| **Operational delta**     | PR \#2 merged as squash `995d9e0` (was open at `cad4230`, reported deployed)          | Reconcile before source freeze.                                               |
| **PR \#2 defect**         | Terminal retry lost durable poison-event visibility     | Persist sanitized dead-letter record before Term.                             |
| **Existing tests**        | Go tests expect postgres:5432 inside Docker network      | Preserve the network alias during WP-00; broad test refactor is not required. |

### 2.1 Legacy migration conflicts

| **File** | **Conflict**                                                                                   | **WP-00 decision rule**                                                                                                                  |
|----------|------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **016**  | Four incidents/docs \*.edit permissions remain while the migration asserts none remain.        | Canonical product permission names are update; reconcile P1 data and role grants through the new baseline/forward path.                  |
| **018**  | Agent tables duplicate 005 with conflicting columns, defaults, nullability, and status values. | Capture the actual P1/P2 shape; approve one supported source shape or explicitly version multiple profiles. Never infer from file order. |
| **029**  | GRANT targets clarityit_app although the chain never creates the role.                         | Record runtime-role ownership in the profile and make privileged role bootstrap/preflight explicit.                                      |

> **Normative correction** Issue \#1 suggested repairing legacy files in place. WP-00 follows the later compatibility contract: migrations 001-040 remain immutable provenance. Their defects are resolved in the reconciled baseline and, where required for adopted databases, in new forward corrective revisions.

### 2.2 Entry prerequisites

- Read-only access sufficient to capture the P1 production schema, extensions, roles/grants digest, row counts, and declared data-quality checks.

- A recent production backup that can be restored into an isolated PostgreSQL 16 environment without target-system credentials.

- Named Product, Architecture, Database, Backend, Operations, Security, and Quality owners.

- A secure evidence location for sensitive profile and restore artifacts; only sanitized manifests and references enter the repository or CI.

- A temporary schema-change freeze for v1 while source profiles and the reconciled target are approved.

> **Entry gate** WP-00 may begin repository work before P1 access is ready, but baseline approval cannot pass and no source profile may be guessed. P1 absence is a blocker, not a reason to elevate the repository chain into production truth.

## 3. Delivery model and dependency gates

> G0 Source freeze  
> -\> G1 Approved P1/P2/P3 profiles and restore proof  
> -\> G2 Recorded 016/018/029 decisions and target manifest  
> -\> G3 Reconciled baseline and immutable legacy archive  
> -\> G4 Migration runner accepted  
> -\> G5 Blocking CI matrix green  
> -\> G6 WP-00 evidence accepted

Gates are sequential authority boundaries, not presentation milestones. Work may overlap where it cannot change an upstream decision, but no downstream artifact is accepted against an unapproved input.

| **Track**                  | **Accountable owner**   | **Primary output**                          | **Exit gate** |
|----------------------------|-------------------------|---------------------------------------------|---------------|
| **WS0 - Freeze**           | Operations              | A1 freeze manifest and source tag           | G0            |
| **WS1 - Profile/restore**  | Database                | A2 source-profile pack; A3 restore proof    | G1            |
| **WS2 - Schema decisions** | Architecture + Database | Decision records and target manifest        | G2            |
| **WS3 - Baseline**         | Database + Backend      | A4 reconciled baseline and legacy archive   | G3            |
| **WS4 - Runner**           | Backend                 | A5 forward-migration pack and migration CLI | G4            |
| **WS5 - CI**               | Quality + Backend       | Blocking CI matrix and test evidence        | G5            |
| **WS6 - Acceptance**       | Product + Architecture  | WP-00 release evidence manifest             | G6            |

## 4. Detailed work breakdown

### 4.1 WS0 - Freeze repository and operational state

**WS0-01.** Record `b9d1587` as the specified logical source *(historical; revision starting baseline is `main@318b4eb`)* and capture the current main branch, PRs, dependency lockfiles, workflow files, container definitions, and deploy scripts.

**WS0-02.** Inventory production-deployed API and worker binaries/images, file-copy deltas, migration history evidence, feature flags, and configuration schema; compare hashes against source-built artifacts.

**WS0-03.** ~~Rework PR \#2 or create a superseding change that retains the object-to-comment edge fix and bounded backoff while writing a sanitized durable dead-letter record before terminal acknowledgement.~~ **(Discharged: PR#2 merged as `995d9e0` carrying the self-loop fix and bounded termination; the `fix/context-worker-durable-dlq` follow-up added the sanitized durable dead-letter record written before Term.)**

**WS0-04.** Add regression tests for self-loop prevention, bounded retry, unparseable message handling, durable poison-event visibility, and disposition behavior. *(Replay/redrive remains out of scope — future work — so "replay" is dropped from the test scope.)*

**WS0-05.** Build the source-freeze candidate, reproduce its artifacts, verify that every deployed semantic change is represented, and tag the approved freeze commit. The tag name and SHA are recorded in A1; this plan does not invent the final SHA.

**G0 pass.** The release artifact reproduces production behavior, PR \#2 is merged (as squash `995d9e0`) with durable poison-event disposition in place via the follow-up, no deployed code delta remains outside source control, and A1 is approved.

### 4.2 WS1 - Capture source profiles and prove restore

**WS1-01.** Implement a read-only profiler that emits a canonical schema manifest and SHA-256 fingerprint. Include PostgreSQL version, extensions, schemas, relations, columns, defaults, nullability, constraints, indexes, triggers, migration-used functions, and relevant roles/grants.

**WS1-02.** Exclude volatile OIDs, physical file locations, statistics, dump timestamps, and secret values from canonicalization. Version the canonicalization algorithm.

**WS1-03.** Capture P1 from the authoritative production database and require Database, Operations, and Security approval.

**WS1-04.** Restore a recent backup into an isolated PostgreSQL 16 environment; capture P2 and prove that its fingerprint and declared data conditions match the approved migration source.

**WS1-05.** Create P3 as a deterministic sanitized fixture that reproduces P1 constraints, conflict shapes, key relationships, and legacy truth cases without production secrets.

**WS1-06.** Store profile approvals in an allowlist consumed by the migration runner. Unknown or drifted profiles remain blocked.

**G1 pass.** P1, P2, and P3 are approved; the P2 restore is repeatable; the profiler is deterministic; sensitive evidence is access-controlled.

### 4.3 WS2 - Resolve the target schema

**WS2-01.** For migration 016, reconcile existing permission rows and role grants, approve the canonical incidents.update and docs.update names, and define deterministic collision handling.

**WS2-02.** For migration 018, compare P1/P2 tables with 005, 018, and current application queries. Approve the canonical columns, constraints, defaults, indexes, and compatibility aliases in a signed decision record.

**WS2-03.** For migration 029, inventory the effective runtime login and group roles. Define privileged bootstrap, ownership, grants, and preflight behavior; missing required roles fail explicitly.

**WS2-04.** Produce the normalized target-schema manifest, including object ownership and grants, then verify it against an independently created clean database.

**WS2-05.** Record whether any additional P1 drift is accepted, corrected, quarantined, or declared unsupported. No unexplained drift enters the baseline.

**G2 pass.** Every conflict has one approved disposition and the target manifest is complete enough to generate and verify a clean database.

### 4.4 WS3 - Create the reconciled baseline

**WS3-01.** Move or copy migrations 001-040 into a versioned legacy-provenance area without changing their bytes; publish their ordered checksums.

**WS3-02.** Generate a deterministic clean-install SQL baseline from the approved target manifest. Remove environment-specific owners, volatile metadata, and data samples not explicitly part of the seed contract.

**WS3-03.** Separate privileged role/bootstrap operations from application-owned schema revisions and make both preconditions explicit.

**WS3-04.** Create the minimal platform migration-control structures needed by WP-00: source profiles, runs, schema revisions, and reconciliation results.

**WS3-05.** Test the empty-database path twice and compare logical dumps, normalized manifests, fingerprints, revision rows, constraints, indexes, triggers, functions, and grants.

**WS3-06.** Define approved-source adoption: record the source profile and baseline adoption revision without replaying legacy SQL, then apply only forward revisions.

**G3 pass.** Fresh install and approved-source adoption converge on the exact target fingerprint; legacy files are immutable and are never selected by the supported runner.

### 4.5 WS4 - Implement the migration runner

**WS4-01.** Add a Go migration command within the existing backend codebase. It is the only supported path for profile validation, baseline install/adoption, planning, application, status, and verification.

**WS4-02.** Embed or package baseline and forward migration bytes with their manifest so a release artifact cannot silently select different SQL.

**WS4-03.** Acquire a PostgreSQL advisory lock before any migration-control or target-schema change. Reject concurrent mutation attempts deterministically.

**WS4-04.** Validate source profile, application compatibility range, PostgreSQL version, required extensions/roles, backup/restore evidence reference, and migration checksums before DDL.

**WS4-05.** Execute transactional revisions atomically. Isolate any unavoidable non-transactional step with explicit pre/post checks and restart state.

**WS4-06.** Record source commit, revision, checksum, actor, run, duration, result, target fingerprint, and evidence reference. A succeeded checksum is immutable.

**WS4-07.** Provide deterministic failpoints for CI and a read-only verify mode suitable for deployment startup and post-restore checks.

**WS4-08.** Give the runner no provider credentials, target-system access, or ability to invoke a ClarityIT effect.

**G4 pass.** The runner passes install, adoption, checksum, lock, restart, verification, and privilege-boundary tests and fully replaces the Makefile psql loop.

### 4.6 WS5 - Make backend CI blocking and green

**WS5-01.** Replace the legacy fresh-migration loop with the supported migration runner and reconciled baseline.

**WS5-02.** Split static/build, fresh install, approved-profile adoption, negative-profile/checksum, restart/lock, context poison-event, and backend integration tests into clear required checks.

**WS5-03.** Retain the Docker network alias postgres for existing Go integration tests during WP-00; do not expand scope into a broad test-harness rewrite.

**WS5-04.** Upload sanitized manifests, revision ledgers, fingerprints, JUnit/test summaries, and failure diagnostics as CI evidence; never upload production dumps or secret-bearing profile content.

**WS5-05.** Remove continue-on-error and the '\[non-blocking\]' job label only after every required check succeeds on the exact final commit.

**WS5-06.** Configure main-branch protection to require the WP-00 backend checks in addition to the already blocking frontend and worker checks.

**G5 pass.** All required checks are blocking and green from a clean runner; rerun produces the same database fingerprints and evidence digests.

### 4.7 WS6 - Rehearse, review, and accept

**WS6-01.** Rehearse the full P2 restore, fingerprint, adoption, forward apply, restart, verify, and evidence flow using only release artifacts and the approved runbook.

**WS6-02.** Inject failures before DDL, during a transactional revision, after a committed revision, during lock contention, and during verification; prove the required recovery behavior.

**WS6-03.** Run the historical-truth fixture and confirm that synthetic success, operator outcome text, and provider submission references remain weaker classifications.

**WS6-04.** Complete schema, identity, constraint, grant, checksum, secret-scan, and artifact-provenance review.

**WS6-05.** Publish the WP-00 release evidence manifest, record approval or block, and close issue \#1 only if its fresh-install and CI claims are demonstrably resolved.

**G6 pass.** Every acceptance criterion is evidenced, every required owner signs, and no severity 1 or 2 foundation defect remains open.

## 5. Technical implementation contract

### 5.1 Recommended repository layout

> migrations/  
> legacy/v1/001-040/ \# byte-preserved provenance; never replayed by v2 runner  
> v2/baseline/0001_reconciled.sql  
> v2/forward/ \# immutable checksummed revisions  
> manifests/ \# ordered checksums and target fingerprint  
> profiles/p3/ \# sanitized deterministic CI fixture  
> services/api/cmd/clarity-migrate/  
> services/api/internal/migrate/  
> docs/migration/wp00/ \# non-sensitive decisions, runbooks, evidence manifest  
> .github/workflows/ci.yml

*The implementation may adjust exact folder names to fit repository conventions, but the separation of immutable legacy provenance, one reconciled baseline, and forward-only revisions is binding.*

### 5.2 Supported runner operations

| **Operation** | **Behavior**                                                                     | **Mutation**        |
|---------------|----------------------------------------------------------------------------------|---------------------|
| **profile**   | Emit canonical manifest and fingerprint; compare to allowlist.                   | Read-only           |
| **plan**      | Resolve empty/install or approved/adopt path and list exact revisions/checksums. | Read-only           |
| **apply**     | Lock, validate, install/adopt, apply forward revisions, verify.                  | Controlled DDL/data |
| **status**    | Show run, revision, checksum, profile, and compatibility state.                  | Read-only           |
| **verify**    | Compare live manifest, revision ledger, grants, and target fingerprint.          | Read-only           |

### 5.3 Baseline selection algorithm

> capture current normalized profile  
> if database is empty:  
> create and record the fixed migration-control bootstrap  
> apply reconciled baseline 0001  
> else:  
> require exact approved source-profile fingerprint  
> require restore-proof reference and compatible application release  
> record baseline adoption; do not execute migrations 001-040  
> validate every pending revision checksum  
> acquire advisory lock  
> apply forward revisions in order with durable run state  
> verify exact target manifest and fingerprint  
> release lock and seal run evidence

### 5.4 Fail-closed conditions

- Unknown source fingerprint, PostgreSQL major version, required extension, role/grant posture, or application compatibility range.

- Unreconciled production code delta or release artifact whose source commit cannot be proven.

- Changed checksum for an already successful revision or different bytes for a declared revision.

- Missing or stale backup/restore proof when the approved run policy requires it.

- Concurrent runner, unresolved previous run, uncertain non-transactional checkpoint, or target fingerprint mismatch.

- Manual source-data edits, ignored SQL errors, skipped rows, or operator override without a recorded approved decision.

### 5.5 Historical-truth safeguard

WP-00 does not perform the v2 historical backfill. It does freeze the classification contract and fixture consumed by the later migration. The fixture must include agent_effect_results, remediation results, asset actions with and without provider task identifiers, approval requests, and action_outcomes.

| **Legacy evidence**                             | **Strongest allowed WP-00 classification** | **Forbidden promotion**                |
|-------------------------------------------------|--------------------------------------------|----------------------------------------|
| Synthetic succeeded effect/result               | legacy_unverified                          | provider_completed, verified, accepted |
| Provider task identifier without terminal proof | legacy_submitted_unverified                | provider_completed, verified           |
| Executing/ambiguous action                      | legacy_outcome_unknown                     | automatic retry, completed             |
| Legacy approval                                 | legacy decision evidence                   | AuthorityGrant                         |
| Operator outcome text                           | legacy annotation/claim                    | Verification or Accepted outcome       |

## 6. CI and test plan

### 6.1 Required CI checks

| **Check**               | **Required coverage**                                 | **Pass condition**                                             |
|-------------------------|-------------------------------------------------------|----------------------------------------------------------------|
| **backend-static**      | gofmt check, go vet, build                            | No warning converted to success; exact source commit recorded. |
| **db-fresh**            | Empty PostgreSQL 16; install twice                    | Exact expected fingerprint; second apply is a no-op.           |
| **db-adopt-p3**         | Restore sanitized P3; profile/adopt/apply             | No legacy replay or manual SQL; target fingerprint exact.      |
| **db-negative**         | Unknown profile, changed checksum, missing role       | Fails before prohibited mutation with stable reason code.      |
| **db-restart-lock**     | Failpoints and two concurrent runners                 | Safe resume; one writer; no duplicate revision.                |
| **context-dead-letter** | Self-loop, retry, terminal failure, replay visibility | Poison event durably visible before Term.                      |
| **backend-integration** | Existing Go test suite in Docker network              | All tests pass with bounded timeout.                           |
| **artifact-audit**      | Manifest, checksums, secret scan, required evidence   | Complete sanitized evidence; no secret finding.                |

> **Blocking rule** A green frontend or worker job cannot compensate for a failed or non-blocking backend foundation check. The merge gate is the conjunction of all required checks.

### 6.2 Mandatory WP-00 scenarios

| **ID**       | **Scenario**                                                                                                                          |
|--------------|---------------------------------------------------------------------------------------------------------------------------------------|
| **MT-00-01** | Two fresh installs produce the exact target schema fingerprint and revision set.                                                      |
| **MT-00-02** | P2/P3 approved source adopts and upgrades without executing any legacy migration file.                                                |
| **MT-00-03** | Unknown fingerprint exits before DDL; pre/post normalized manifests are identical.                                                    |
| **MT-00-04** | A modified checksum for a successful revision is rejected.                                                                            |
| **MT-00-05** | Two concurrent apply commands cannot mutate the database concurrently.                                                                |
| **MT-00-06** | Failure inside a transactional revision leaves no partial target change and resumes safely.                                           |
| **MT-00-07** | Failure after commit recognizes the completed revision and does not execute it twice.                                                 |
| **MT-00-08** | Migration 016 canonical permission rows and role grants match the approved target.                                                    |
| **MT-00-09** | The approved 005/018 agent schema source shape converges to the exact target.                                                         |
| **MT-00-10** | Missing or incorrect runtime role posture fails with an explicit preflight result.                                                    |
| **MT-00-11** | P2 restore proof and P1/P2/P3 profile lineage are reconstructable.                                                                    |
| **MT-00-12** | Self-loop event does not fail ingestion; poison events use bounded retry and durable dead-lettering.                                  |
| **MT-00-13** | All existing backend tests pass with a bounded timeout and stable PostgreSQL network alias.                                           |
| **MT-00-14** | Legacy synthetic success maps only to legacy_unverified.                                                                              |
| **MT-00-15** | Legacy provider submission ambiguity maps to legacy_submitted_unverified or legacy_outcome_unknown and is never auto-retried.         |
| **MT-00-16** | Legacy approvals and outcome annotations create no authority, Verification, or acceptance.                                            |
| **MT-00-17** | Evidence artifacts contain no production secrets or raw sensitive values.                                                             |
| **MT-00-18** | A reviewer reproduces the source, profile, baseline, apply, verification, and CI evidence chain from A1-A5 plus the release manifest. |

## 7. Deliverables and evidence

| **ID** | **Artifact**           | **Minimum content**                                                                                       | **Approval**              |
|--------|------------------------|-----------------------------------------------------------------------------------------------------------|---------------------------|
| **A1** | Freeze manifest        | Source/freeze commit and tag; deployed-byte comparison; image and lockfile digests; PR \#2 disposition.   | Operations + Architecture |
| **A2** | Source-profile pack    | P1/P2/P3 manifests, fingerprints, canonicalizer version, extensions, role digest, data checks, approvals. | Database + Security       |
| **A3** | Restore proof          | Backup reference, isolated restore log, P2 fingerprint, elapsed time, validation result.                  | Database + Operations     |
| **A4** | Reconciled baseline    | Baseline SQL, target manifest/fingerprint, legacy checksum inventory, generation provenance.              | Database + Architecture   |
| **A5** | Migration pack         | Runner artifact, control-schema DDL, forward revisions, checksums, compatibility range, recovery posture. | Backend + Database        |
| **A6** | CI evidence            | Required-check results, sanitized manifests, test summaries, fault-injection and rerun evidence.          | Quality                   |
| **A7** | WP-00 release manifest | Artifact digests, gate results, open-defect statement, approvals, acceptance or block decision.           | Product + Architecture    |

Sensitive P1/P2 bytes remain outside the repository and ordinary CI. Repository artifacts contain normalized metadata, digests, sanitized fixtures, decisions, and evidence references sufficient for audit without exposing secrets.

## 8. Schedule and ownership

### 8.1 Target execution window

Target critical path: 15 working days from access to P1 and the restorable backup. The schedule assumes one backend owner and one database/operations owner working in parallel, with Quality and Security engaged at their gates. If P1 access or restore proof is late, the acceptance date moves; the gate is not weakened.

| **Window**     | **Primary work**                                                   | **Gate** |
|----------------|--------------------------------------------------------------------|----------|
| **Days 1-2**   | WS0 freeze, deployment-byte inventory, PR \#2 correction and tests | G0       |
| **Days 2-4**   | Profiler, P1 capture, P2 restore, P3 fixture                       | G1       |
| **Days 4-6**   | 016/018/029 decisions and target manifest                          | G2       |
| **Days 6-9**   | Reconciled baseline, legacy archive, target equivalence            | G3       |
| **Days 7-11**  | Go runner, ledger, locks, adoption, restart and verify             | G4       |
| **Days 10-13** | Blocking CI matrix, negative and failure-injection tests           | G5       |
| **Days 14-15** | P2 rehearsal, evidence review, final acceptance                    | G6       |

### 8.2 Responsibility matrix

| **Role**               | **Accountability in WP-00**                                           | **Approval** |
|------------------------|-----------------------------------------------------------------------|--------------|
| **Product owner**      | Scope, exclusions, final release decision                             | G6           |
| **Architecture owner** | Authority hierarchy, source/target boundaries, no legacy replay       | G0, G2, G6   |
| **Database owner**     | P1/P2/P3, target DDL, restore, fingerprints, runner DB behavior       | G1-G4        |
| **Backend owner**      | PR \#2 correction, migration command, ledger, tests, workflow changes | G0, G4, G5   |
| **Operations owner**   | Deployment inventory, freeze, backup access, isolated rehearsal       | G0, G1, G6   |
| **Security owner**     | Role boundary, sensitive evidence, secret scan, artifact provenance   | G1, G5, G6   |
| **Quality owner**      | Fixture quality, CI matrix, fault injection, evidence completeness    | G5, G6       |

## 9. Risks and controls

| **Risk**                                     | **Early indicator**                           | **Required control**                                                                                 |
|----------------------------------------------|-----------------------------------------------|------------------------------------------------------------------------------------------------------|
| **Production shape differs from repository** | P1 fingerprint or object inventory mismatch   | Fail closed; approve a new profile and update the target decision before baseline generation.        |
| **018 has more than one live shape**         | P1/P2/other environment disagreement          | Support only named profiles with explicit convergent paths; do not use permissive SQL.               |
| **Operational code is outside Git**          | Binary/image hash has no source match         | Block G0 until reconstructed, reviewed, and tagged.                                                  |
| **Poison event becomes invisible**           | Term without durable record                   | Persist sanitized dead-letter evidence before terminal acknowledgement.                              |
| **Role bootstrap needs elevated privilege**  | clarityit_app absent or ownership differs     | Separate privileged bootstrap from app migration; preflight exact required role/grants.              |
| **Baseline generation is nondeterministic**  | Clean-run fingerprints differ                 | Version canonicalization; normalize dump inputs; compare two independent fresh installs.             |
| **CI hides state leakage**                   | Rerun passes only on reused volume/cache      | Use clean PostgreSQL per job; rerun apply and compare digests.                                       |
| **Sensitive data leaks into evidence**       | Raw dump/sample or credential in artifact     | Sanitized P3 only in CI; secret scan; restricted P1/P2 storage; digest/reference in public evidence. |
| **Scope expands into v2 semantics**          | Kernel/adapter/UI changes appear in WP-00 PRs | Reject from WP-00 and queue for follow-on work after G6.                                             |

## 10. Acceptance criteria

### 10.1 Source freeze

**AC-00-01.** A1 identifies `b9d1587` as the specified historical source reference and the revision starting baseline (`main@318b4eb`, advanced by the durable-disposition follow-up) as the approved freeze commit/tag containing every deployed semantic delta.

**AC-00-02.** Production binaries/images/configuration schema are compared to reproducible source-built artifacts; every mismatch is resolved or blocks G0.

**AC-00-03.** ~~PR \#2 is merged or superseded only after durable dead-letter visibility is implemented and tested.~~ **(Historically accurate rewording: PR#2 merged as `995d9e0` with bounded termination; this follow-up (`fix/context-worker-durable-dlq`) supplies the missing durable poison-event disposition the original AC-00-03 required before G0 acceptance. Do not read this as "PR#2 merged only after durable DLQ" — that ordering did not occur.)**

**AC-00-04.** No file-copy-only production change remains outside the freeze artifact.

### 10.2 Profiles and restore

**AC-00-05.** The profiler emits the same manifest and fingerprint for two unchanged captures.

**AC-00-06.** P1, P2, and P3 include PostgreSQL version, extensions, schema objects, constraints, indexes, triggers/functions, and roles/grants digest.

**AC-00-07.** P2 is produced from a recorded isolated restore and matches the approved source conditions.

**AC-00-08.** P3 is sanitized and contains deterministic cases for 016, 018, 029, relationships, and legacy truth.

**AC-00-09.** An unapproved fingerprint exits before DDL and produces a stable diagnostic.

### 10.3 Reconciled target and baseline

**AC-00-10.** 016, 018, and 029 each have an approved decision grounded in P1/P2 and current application behavior.

**AC-00-11.** Legacy migrations 001-040 retain their original bytes and ordered checksums.

**AC-00-12.** The supported runner never selects or replays a legacy migration file.

**AC-00-13.** Two clean baseline installations produce the exact approved target fingerprint.

**AC-00-14.** Approved-source adoption converges to the same target without manual DDL or data edits.

**AC-00-15.** Schema ownership and runtime-role grants are explicit and verified.

### 10.4 Migration runner

**AC-00-16.** A PostgreSQL advisory lock prevents concurrent migration writers.

**AC-00-17.** Every successful revision records immutable version, checksum, source commit, actor, run, duration, and result.

**AC-00-18.** A changed checksum for a successful revision is rejected.

**AC-00-19.** Transactional failure leaves no partial change and rerun resumes without duplication.

**AC-00-20.** Any non-transactional step has explicit pre/post conditions and restart evidence.

**AC-00-21.** Verify mode detects missing/extra objects, constraint/index/grant drift, revision mismatch, and target fingerprint mismatch.

**AC-00-22.** The runner has no provider credential, target access, or effect-execution path.

### 10.5 CI, truth, security, and evidence

**AC-00-23.** Backend CI contains no continue-on-error and no non-blocking label.

**AC-00-24.** All checks in section 6.1 are required and green on the final commit.

**AC-00-25.** Fresh install and P3 adoption run from clean PostgreSQL instances and are reproducible.

**AC-00-26.** The historical-truth fixture produces zero provider_completed, verified, or accepted mappings from legacy claims.

**AC-00-27.** Poison-event tests prove bounded retry, durable terminal disposition, and operator/replay visibility.

**AC-00-28.** Sanitized CI and release artifacts pass secret scanning; production dumps and secret values are absent.

**AC-00-29.** A reviewer reconstructs every gate from A1-A7 and verifies all artifact digests.

**AC-00-30.** No unresolved severity 1 or 2 migration, data-integrity, security, retry, or CI defect remains open.

## 11. Exit and follow-on boundary

> **Acceptance decision** WP-00 is Accepted only when AC-00-01 through AC-00-30 are evidenced and G6 is signed. Otherwise it is Blocked with named failed criteria, owner, corrective work, and preserved evidence. Conditional or partial acceptance is not permitted.

Acceptance authorizes the next implementation-ready contract: Generic Compute Adapter Specification v0.1, followed by the Proxmox VE conformance profile and the verified virtual-machine recovery slice. It does not itself authorize provider mutation, Site Runtime deployment, host agents, additional adapters, or broader UI work.

The [Project Completion Authority](PROJECT-COMPLETION-AUTHORITY.md) records the
current gate and exact integrated identities. It is a status ledger, not a
substitute for G6 evidence or signatures.

## Appendix A. First-day execution checklist

- Confirm named owners and schedule the G0-G2 approval sessions.

- Create the WP-00 integration branch and protect main from direct schema changes.

- Capture the revision starting baseline (`main@318b4eb`; historically `main=b9d1587`, PR \#2 now merged as `995d9e0`), current workflow, Makefile, dependency locks, and deployment inventory.

- ~~Open the PR \#2 superseding task with durable dead-letter acceptance criteria.~~ **(Discharged: the durable-disposition follow-up `fix/context-worker-durable-dlq` implements this.)**

- Provision the isolated PostgreSQL 16 restore environment and secure evidence location.

- Run the profiler first against P2/P3 during development, then perform the approved read-only P1 capture.

- Do not generate the reconciled baseline until G2 approves the target manifest.

## Appendix B. Source basis and authority

| **Source**                                                  | **Use in WP-00**                                        | **Authority**         |
|-------------------------------------------------------------|---------------------------------------------------------|-----------------------|
| **ClarityIT v2 Product Definition v0.1**                    | Product boundary and delivery sequence                  | Product               |
| **Authoritative Execution Kernel Specification v0.1**       | Truth and separation invariants carried into fixtures   | Engineering semantics |
| **v1-to-v2 Compatibility and Migration Specification v0.1** | Profiles, baseline, migration control, CI, and gates    | Migration             |
| **Updated ClarityIT v2 reference architecture**             | Component and persistence boundaries                    | Architecture          |
| **Octo-Lex/ClarityIT b9d1587** *(historical)*               | Specified v1 source implementation (revision starting baseline now `main@318b4eb`) | Source implementation |
| **GitHub issue \#1**                                        | Fresh-install defect record                             | Known defect          |
| **GitHub PR \#2 / cad4230** *(historical)*                  | Operational retry delta — merged as squash `995d9e0`; durable disposition discharged by `fix/context-worker-durable-dlq` | Operational delta     |
| **Approved P1/P2/P3 profile pack**                          | Actual source database authority                        | Upgrade source        |
| **This work package**                                       | Execution order, tasks, gates, evidence, and acceptance | WP-00 delivery        |
