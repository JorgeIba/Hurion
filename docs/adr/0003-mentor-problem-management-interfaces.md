# ADR 0003: Mentor Problem Management through Slash Commands, Batches, and an API

* **Status**: Accepted
* **Date**: 2026-09-29
* **Implementation plan**: [Slash problem management](../plans/slash-problem-management.md)

## 1. Context

Mentors need a convenient way to create and correct problems, including changes prepared by an agent. Positional text commands are useful for simple operations but become cumbersome when several fields must be supplied or edited together.

Discord provides an existing place for interactive work. Batches and an API can support larger changes and integrations once the application operations are established. Remote API access also requires a hosting decision consistent with [ADR 0002](0002-container-delivery-and-cicd-pipeline.md).

## 2. Decision

Introduce the interfaces in phases, sharing typed problem operations, authorization, and validation:

1. **Slash commands and forms** for interactive mentor editing.
2. **Explicit text batches** for preparing several changes together.
3. **An authenticated HTTP API** for integrations and direct automation.

An MCP adapter may expose the same application operations when there is demand from AI clients.

### Application and presentation boundaries

Presentation adapts transport-specific input to application requests and formats results. Application operations enforce permissions and problem rules before calling domain services. Domain services remain independent of Discord and HTTP.

Mappings preserve identity and optional-value semantics. Controllers coordinate contextual lookups, temporary editor state, and response delivery. Existing prefix problem commands use the same application policies while retaining their text interface.

Build the responsibilities required by the active phase. Review concrete types, function names, file placement, and test arrangements with each implementation increment.

### Phase 1: Slash commands

Support problem creation and editing through Discord forms, with autocomplete for selecting an edit target and private results. Existing Discord and web lists provide browsing. Destructive slash operations are outside this phase.

Class is a required, editable form field. Creation defaults it from the current channel when possible; editing defaults it from the saved problem. Mentors can correct the class, and the destination must exist. Editing preserves the problem ID and solved records. Display the target identity without making it editable; the owned editor session determines the target when saving.

The configured mentor or admin role permits problem management. Check the actor and guild at runtime and verify current authority before a write. Student access roles do not imply restrictions on which classes a mentor can manage. The same role policy applies to prefix problem creation and updates.

Keep command definitions aligned with the application during normal bot startup, limited to the configured guild and owned commands. Route incoming interactions to the appropriate application operation and respond within Discord's protocol constraints.

Forms preserve supplied values during correction. Optional difficulty is text; blank or unspecified difficulty and blank topic clear their optional values. Shared validation applies consistently across interfaces, including an explicit repair path for legacy values that cannot be saved or displayed under the agreed rules.

Use bounded, expiring editor sessions in memory, backed by Otter v2. Bind them to the actor and guild, prevent duplicate execution of a submission, and reject new editors when full rather than evict active work. A save already in progress remains protected until completion. Expired sessions and process restarts require a new invocation.

Concurrent editors use **last successful write wins**, including unchanged prefilled fields and class membership. This accepted tradeoff permits an older form to overwrite a newer edit. Deleted targets remain deleted. Automatic replay of uncertain writes and durable editor recovery are outside phase 1.

### DiscordGo version and upgrade follow-up

Use native SDK support for Label-based forms and submission decoding. The accepted compatibility pin is DiscordGo `v0.29.1-0.20260214123928-f43dd94faaac`, referencing [upstream commit f43dd94faaac](https://github.com/bwmarrin/discordgo/blob/f43dd94faaacd5b163e9e783f14b5bd8be639fc9/components.go). This permits the selected form layout without a custom encoder or SDK fork; its tradeoff is an upstream commit rather than a tagged release.

Prefer a reviewed compatible tag when available. Dependency upgrades must preserve form encoding, submission decoding, and optional-field behavior, with application checks and live form verification before rollout. The dependency manifest records the version actually used. Dropdown support remains a separate decision.

### Phase 2: Explicit command batches

Accept structured multiline input or a text attachment through a dedicated batch workflow. Define and review the grammar, preview the proposed changes, and bind approval to the submitting actor and guild. Recheck authority before execution.

Execute operations in order and report completed, failed, and unexecuted work. Earlier successful changes remain applied when a later operation fails. Resume and durable retry guarantees require explicit design.

### Phase 3: Authenticated API

Expose structured problem operations using personal, revocable tokens bound to a verified owner and guild. A token's permissions cannot exceed its owner's current authority. Protect credentials and check permissions for every operation.

Define partial updates and retry behavior explicitly. Retryable writes must preserve their mutation and saved outcome atomically. Return results to the caller; optional Discord receipts report delivery separately from mutation success.

Reuse the application's problem operations and storage. Review hosting and network access before exposing the API remotely.

## 3. Alternatives and Tradeoffs

| Approach | Benefit | Cost or limitation |
| --- | --- | --- |
| Improve prefix commands only | Small change to an existing interface | Positional editing remains cumbersome |
| Slash forms first | Native workflow where mentors already work | Interaction timing, registration, and temporary state |
| Text batches | Several prepared changes without API credentials | Grammar, preview, and partial failure handling |
| API first | Flexible integrations | Authentication, hosting, and retry guarantees |
| MCP first | Direct access from AI clients | Requires a stable operation and authentication contract |
| Web administration UI | Rich browsing and editing | Separate interface and authenticated write backend |

Start with slash forms, then introduce batches and the API as their use cases justify the work.

## 4. Review and Rollout

Implement in small, independently reviewable increments. Review behavior, boundaries, and relevant verification before proceeding. Validate Discord behavior in a test guild before production rollout.

Keep existing prefix commands available during migration. Review batch and API contracts in their own phases. Publishing and production deployment follow explicit user approval.

## 5. References

- [Discord application commands](https://docs.discord.com/developers/interactions/application-commands)
- [Discord interaction response lifecycle](https://docs.discord.com/developers/interactions/receiving-and-responding)
- [Discord modal components](https://docs.discord.com/developers/components/using-modal-components)
- [Discord component reference](https://docs.discord.com/developers/components/reference)
- [DiscordGo releases](https://github.com/bwmarrin/discordgo/releases)
- [Otter v2 expiration and pinning](https://maypok86.github.io/otter/user-guide/v2/features/eviction/)
