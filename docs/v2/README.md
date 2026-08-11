# ClarityIT v2 Markdown Document Set

This directory is the portable GitHub-flavored Markdown edition of the current
ClarityIT v2 specification and delivery program. The design authorities are
dated 30 July through 1 August 2026; WP-00 closed on 11 August 2026 and WP-01
package planning was authorized on 12 August 2026.

## Start here

The [Project Completion Authority](PROJECT-COMPLETION-AUTHORITY.md) is the
operational entry point for current gate status, frozen identities, authorized
continuation, and blocked scope. It does not replace the semantic authority of
the Product Definition, execution kernel, migration specification, or signed
evidence records.

WP-01 planning authority is recorded in
[`wp01/WP01-AUTHORIZATION.md`](wp01/WP01-AUTHORIZATION.md). The formal package
boundary is defined by the
[WP-01 Authoritative Kernel Foundation Plan](ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md).

The packaging provenance for the original v2 document-set integration onto
`main` is recorded in
[`docs/migration/wp00/AUTHORITY-RECONCILIATION.md`](../migration/wp00/AUTHORITY-RECONCILIATION.md).

## Documents

| Order | Document | Role |
|---:|---|---|
| 1 | [Project Completion Authority](PROJECT-COMPLETION-AUTHORITY.md) | Current gate ledger, frozen identities, authorization boundary, and future-session handoff |
| 2 | [WP-01 Authorization](wp01/WP01-AUTHORIZATION.md) | `WP01-AUTH-2026-08-12`, exact WP-01 authority-set ratification and activation boundary |
| 3 | [WP-01 Authoritative Kernel Foundation Plan](ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md) | Formal WP-01 workstreams, internal gates, AC-01 criteria, evidence and RG-01 exit contract |
| 4 | [Product Definition](ClarityIT_v2_Product_Definition_v0.1.md) | Product and design authority: users, value, experience, scope, and release boundary |
| 5 | [Authoritative Execution Kernel](ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md) | Engineering contract: truth, authority, execution, verification, outcomes, and evidence |
| 6 | [v1-to-v2 Compatibility and Migration](ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md) | Migration contract: source profiles, coexistence, cutover, rollback, and historical truth |
| 7 | [WP-00 Migration Baseline and CI Stabilization Plan](ClarityIT_v2_WP-00_Migration_Baseline_and_CI_Stabilization_Plan_v0.1.md) | Completed migration-foundation package: migration phases 0–1 and gates G0–G6 |
| 8 | [Delivery Roadmap](ClarityIT_v2_Delivery_Roadmap_v0.2.md) | WP-01–WP-10, RG-01–RG-10, and R1–R5 completion sequence; later packages remain separately gated |
| 9 | [Native Pattern Specification](ClarityIT_v2_Native_Pattern_Specification_v0.1.md) | Normative pattern set ratified for the WP-01-owned patterns by `WP01-AUTH-2026-08-12`; later package implementation remains separately gated |
| 10 | [Environment Trust and Evidence Custody Profile](ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md) | Adopted development exception and normative production trust/custody exit criteria |
| 11 | [Layered System Architecture](ClarityIT-v2-Layered-System-Architecture.md) | Editable Mermaid executive overview of the seven-plane target-state architecture |
| 12 | [Authoritative Operation Sequence](ClarityIT-v2-Authoritative-Operation-Sequence.md) | Physical outbox dispatch, evidence sealing, independent verification, and accountable outcome decision |
| 13 | [Trust and Deployment Topology](ClarityIT-v2-Trust-and-Deployment-Topology.md) | Workload identity, credentials, policy, runtime compatibility, isolation, and environment placement |
| 14 | [Signals and Routines](ClarityIT-v2-Signals-and-Routines.md) | Proposed separation of human intake, Signals, routine firing, deduplication, destinations, and exceptions |

Corrected DOCX distributions for the original four long-form specifications are
retained under [`rendered/v0.1/`](rendered/v0.1/). Markdown is the maintained
source for future revisions; rendered distributions must be regenerated from an
approved, hash-pinned Markdown baseline. The Native Pattern Specification,
environment profile, roadmap, completion ledger, and package plans are
Markdown-only repository authorities unless a separately approved rendered
release is produced.

## Authority and scope

The files retain the status stated by their source documents except where a later
repository authorization explicitly ratifies an exact version for a bounded
package. Conversion to Markdown by itself does not approve a draft, change a gate
decision, or broaden the first-release scope.

WP-00 is complete. WP-01 is explicitly authorized under
`WP01-AUTH-2026-08-12`; its implementation authority activates when the formal
WP-01 plan is integrated to `main`. WP-02 through WP-10 remain separately gated
and require their own authorization/package plans. RG-01 is the only WP-01 exit
gate that can authorize the foundation as ready for a separately authorized
WP-02 package.

The WP-01 authorization ratifies the exact Product, Kernel, Compatibility,
Layered Architecture, Native Pattern and Delivery Roadmap repository versions
listed in `wp01/WP01-AUTHORIZATION.md` **for WP-01 only**. That bounded
ratification does not automatically authorize later patterns or packages.

The architecture suite is target state. WP-01 may implement the bounded P-05
context-overlay, P-09 trust-foundation and other pattern contracts explicitly
owned by its plan. P-12 Site Runtime and P-15 Signals/Routines remain outside
WP-01. The first live provider-neutral VM-start mutation remains WP-02; WP-01 is
provider-mutation-free and uses deterministic fake/no-op execution fixtures only.

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
