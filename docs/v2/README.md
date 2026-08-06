# ClarityIT v2 Markdown Document Set

This directory is the portable GitHub-flavored Markdown edition of the current
ClarityIT v2 specification and delivery program. The design authorities are
dated 30 July through 1 August 2026; the current completion ledger is dated
5 August 2026.

## Start here

The [Project Completion Authority](PROJECT-COMPLETION-AUTHORITY.md) is the
operational entry point for current gate status, frozen identities, authorized
continuation, and blocked scope. It does not replace the semantic authority of
the Product Definition, execution kernel, migration specification, or signed
evidence records.

The packaging provenance for this integration onto `main` is recorded in
[`docs/migration/wp00/AUTHORITY-RECONCILIATION.md`](../migration/wp00/AUTHORITY-RECONCILIATION.md).

## Documents

| Order | Document | Role |
|---:|---|---|
| 1 | [Project Completion Authority](PROJECT-COMPLETION-AUTHORITY.md) | Current gate ledger, frozen identities, authorization boundary, and future-session handoff |
| 2 | [Product Definition](ClarityIT_v2_Product_Definition_v0.1.md) | Product and design authority: users, value, experience, scope, and release boundary |
| 3 | [Authoritative Execution Kernel](ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md) | Engineering contract: truth, authority, execution, verification, outcomes, and evidence |
| 4 | [v1-to-v2 Compatibility and Migration](ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md) | Migration contract: source profiles, coexistence, cutover, rollback, and historical truth |
| 5 | [WP-00 Migration Baseline and CI Stabilization Plan](ClarityIT_v2_WP-00_Migration_Baseline_and_CI_Stabilization_Plan_v0.1.md) | First delivery work package: migration phases 0–1 and gates G0–G6 |
| 6 | [Delivery Roadmap](ClarityIT_v2_Delivery_Roadmap_v0.2.md) | Draft WP-01–WP-10, RG-01–RG-10, and R1–R5 completion sequence |
| 7 | [Native Pattern Specification](ClarityIT_v2_Native_Pattern_Specification_v0.1.md) | Draft normative patterns for governed work, execution, knowledge, routines, and scale |
| 8 | [Environment Trust and Evidence Custody Profile](ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md) | Adopted development exception and normative production trust/custody exit criteria |
| 9 | [Layered System Architecture](ClarityIT-v2-Layered-System-Architecture.md) | Editable Mermaid executive overview of the seven-plane target-state architecture |
| 10 | [Authoritative Operation Sequence](ClarityIT-v2-Authoritative-Operation-Sequence.md) | Physical outbox dispatch, evidence sealing, independent verification, and accountable outcome decision |
| 11 | [Trust and Deployment Topology](ClarityIT-v2-Trust-and-Deployment-Topology.md) | Workload identity, credentials, policy, runtime compatibility, isolation, and environment placement |
| 12 | [Signals and Routines](ClarityIT-v2-Signals-and-Routines.md) | Proposed separation of human intake, Signals, routine firing, deduplication, destinations, and exceptions |

Corrected DOCX distributions for the original four long-form specifications are
retained under [`rendered/v0.1/`](rendered/v0.1/). Markdown is the maintained
source for future revisions; rendered distributions must be regenerated from an
approved, hash-pinned Markdown baseline. The Native Pattern Specification,
environment profile, roadmap, and completion ledger are currently Markdown-only
repository authorities.

## Authority and scope

The files retain the status stated by their current source documents. Conversion
to Markdown does not approve a draft, change a gate decision, or broaden the
first-release scope.

Only WP-00 is currently a formal delivery package. WP-01 through WP-10 and
RG-01 through RG-10 remain proposed names and boundaries until separately
approved package plans exist. The Project Completion Authority is authoritative
for current status and continuation; it does not override the semantic authority
of the specifications or signed receipts.

The architecture suite is target state. The layered overview and authoritative
operation sequence are constrained by the Product and Kernel v0.1 baselines.
Dashed-border P-05/P-09/P-12/P-15 content is explicitly proposed under the
unapproved Native Pattern draft; incorporation in a diagram does not approve it.
Site Runtime, Native Enforcement, and the optional Host Sensor remain outside
WP-00 and the initial central-route Proxmox slice.

> **Execution truth invariant:** Provider, worker and agent outputs remain
> source-attributed claims after persistence. Only independent verification can
> establish a verified result, and only a separate outcome decision can accept it.

## Conversion notes

- DOCX headings were shifted one level so each Markdown file has one document-level
  `#` heading.
- Cover metadata was normalized into readable Markdown without changing the
  substantive body text.
- Tables were preserved as GitHub pipe tables.
- Embedded figures were extracted to [`images/`](images/), given descriptive
  filenames, and referenced with relative paths.
- The four architecture documents use Mermaid so the source remains editable and
  renderable on GitHub. Version-stamped PNGs are generated from the committed
  Mermaid source and retained under [`images/`](images/).
- DOCX pagination, page headers, and page footers are presentation features and are
  not reproduced in Markdown.

## Integrity

- [`SOURCE-SHA256SUMS.txt`](SOURCE-SHA256SUMS.txt) pins the corrected DOCX
  distributions and architecture assets used for this edition.
- [`SHA256SUMS.txt`](SHA256SUMS.txt) pins the complete repository document set,
  excluding the manifest itself.
