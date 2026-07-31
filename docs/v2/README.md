# ClarityIT v2 Markdown Document Set

This directory is the portable GitHub-flavored Markdown edition of the current
ClarityIT v2 specification program, updated 1 August 2026.

## Documents

| Order | Document | Role |
|---:|---|---|
| 1 | [Product Definition](ClarityIT_v2_Product_Definition_v0.1.md) | Product and design authority: users, value, experience, scope, and release boundary |
| 2 | [Authoritative Execution Kernel](ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md) | Engineering contract: truth, authority, execution, verification, outcomes, and evidence |
| 3 | [v1-to-v2 Compatibility and Migration](ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md) | Migration contract: source profiles, coexistence, cutover, rollback, and historical truth |
| 4 | [WP-00 Migration Baseline and CI Stabilization Plan](ClarityIT_v2_WP-00_Migration_Baseline_and_CI_Stabilization_Plan_v0.1.md) | First delivery work package: migration phases 0-1 and gates G0-G6 |
| 5 | [Native Pattern Specification](ClarityIT_v2_Native_Pattern_Specification_v0.1.md) | Normative catalog of 18 reusable product and architecture patterns, including contracts, evidence, failure behavior, and conformance criteria |
| 6 | [Delivery Roadmap](ClarityIT_v2_Delivery_Roadmap_v0.2.md) | Migration-first WP-00-WP-10 implementation sequence, package gates, release boundaries, dependencies, metrics, and pattern ownership |
| 7 | [Layered System Architecture](ClarityIT-v2-Layered-System-Architecture.md) | Editable Mermaid representation of the target-state reference architecture |

Versioned DOCX distributions are retained under [`rendered/`](rendered/).
Markdown is the maintained source for future revisions; rendered distributions
must be regenerated from an approved, hash-pinned Markdown baseline.

## Authority and scope

The files retain the status stated by their current source documents. Conversion
to Markdown does not approve a draft, change a gate decision, or broaden the
first-release scope.

The Product Definition governs product scope. The Authoritative Execution Kernel
governs execution semantics. The migration specification governs source-to-target
transition. The Native Pattern Specification defines reusable pattern contracts,
and the Delivery Roadmap sequences their implementation without overriding those
higher-order authorities.

WP-00 remains the only formally named v2 delivery package and the hard
prerequisite. WP-01 through WP-10 are proposed package names until their package
plans are approved. The first release remains one provider-neutral
`compute.virtual_machine.start` lineage on one enrolled non-production virtual
machine. Site Runtime, multi-target execution, additional providers, and wider
mutation paths remain behind their stated gates.

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
- The architecture document uses Mermaid so it remains editable and renderable on
  GitHub.
- DOCX pagination, page headers, and page footers are presentation features and are
  not reproduced in Markdown.

## Integrity

- [`SOURCE-SHA256SUMS.txt`](SOURCE-SHA256SUMS.txt) preserves the source baseline
  for the original corrected specifications and architecture assets.
- [`SHA256SUMS.txt`](SHA256SUMS.txt) pins the complete repository document set,
  excluding the manifest itself.
