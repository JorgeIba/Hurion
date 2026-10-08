# ADR 0003: Mentor Problem Management through Slash Commands, Batches, and an API

* **Status**: Accepted
* **Date**: 2026-09-29
* **Implementation plan**: [Slash problem management](../plans/slash-problem-management.md)
* **Phase-2 implementation plan**: [Mentor command batches](../plans/mentor-command-batches.md)

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

Support problem creation and editing through Discord forms, with selection by problem ID and private results. Autocomplete is deferred. Existing Discord and web lists provide browsing. Destructive slash operations are outside this phase.

Class is selected through the optional `class` slash option before opening a form. Creation defaults it from the current channel when possible; editing defaults it from the saved problem. `/problem edit problem:<ID> [class:<code>]` opens five fields: Name, URL, Difficulty, Priority, and Topic. A destination class must exist. A failed edit draft is keyed by guild, author, problem ID, and destination class code. Editing preserves the problem ID and solved records; the modal context carries the target ID and destination class code and the service resolves the destination class code before saving.

The configured mentor or admin role permits problem management. Check the actor and guild at runtime and verify current authority before a write. Student access roles do not imply restrictions on which classes a mentor can manage. The same role policy applies to prefix problem creation and updates.

Keep command definitions aligned with the application during normal bot startup, limited to the configured guild and owned commands. Route incoming interactions to the appropriate application operation and respond within Discord's protocol constraints.

Forms preserve supplied values during correction. Optional difficulty and priority are text; blank or unspecified difficulty and blank priority or topic clear their optional values. Shared validation applies consistently across interfaces, including an explicit repair path for legacy values that cannot be saved or displayed under the agreed rules.

Use the same bounded, expiring Otter draft cache as problem creation. Failed edit forms expire after ten minutes and are capped at 100 entries. Successful form delivery consumes a recovered draft; failed delivery leaves it available. Duplicate-delivery suppression and in-flight session state are outside this increment.

Concurrent editors use **last successful write wins**, including unchanged prefilled fields and class membership. This accepted tradeoff permits an older form to overwrite a newer edit. Deleted targets remain deleted. Automatic replay of uncertain writes and durable editor recovery are outside phase 1.

### DiscordGo version and upgrade follow-up

Use native SDK support for Label-based forms and submission decoding. The accepted compatibility pin is DiscordGo `v0.29.1-0.20260214123928-f43dd94faaac`, referencing [upstream commit f43dd94faaac](https://github.com/bwmarrin/discordgo/blob/f43dd94faaacd5b163e9e783f14b5bd8be639fc9/components.go). This permits the selected form layout without a custom encoder or SDK fork; its tradeoff is an upstream commit rather than a tagged release.

Prefer a reviewed compatible tag when available. Dependency upgrades must preserve form encoding, submission decoding, and optional-field behavior, with application checks and live form verification before rollout. The dependency manifest records the version actually used. Dropdown support remains a separate decision.

### Phase 2: Explicit command batches

Support `/problem add_batch` for lines of `!add_problem` and `/ac batch` for explicit student `!ac` lines. Each batch accepts only its own operation; updates, removals, mixed batches, general scripts, and HTTP API access are outside this phase. Start with multiline forms and add `.txt` attachments as a separate increment using the same parsers.

Prepare without mutations: parse every line, retain original line numbers, validate values and current permissions, resolve targets, and show a private preview before confirmation. Report all detected AC duplicates, both within the input and against existing records, so the mentor can correct and resubmit. Do not check or remove duplicate problems, including identical entries. Preserve existing role policies: mentors or admins manage problems; mentor ACs require the mentor role and the student's access to the problem's class.

Store the prepared batch in a bounded in-process Otter cache, bound to an opaque batch ID, submitting actor, and guild. Pending batches expire five minutes after insertion; reads do not renew this deadline. Initial bounds are 100 operations per batch, 100 cached batches, and 256 KiB of UTF-8 input. Capacity eviction may remove a batch earlier, and restart loses pending batches.

Confirmation and cancellation claim and remove a matching, unexpired batch atomically. Only the successful claimant may execute, outside the cache lock. Recheck current authority and student roles before writing; perform database target and conflict checks in the same transaction as all mutations, including necessary user records. Discord lookups run outside the database transaction. All operations commit together or roll back together, replacing the earlier partial-execution decision. Preserve insertion order and return created problem IDs only after commit. Verify that scoreboard export reflects the committed result.

This is single-process confirmation consumption, not durable exactly-once execution. Do not reinsert consumed batches or automatically retry writes. Known failures with confirmed rollback require fresh validation and confirmation. Separate confirmed rollback, uncertain outcomes, and receipt-delivery failures. An absent cache entry does not establish that nothing was written: it may have been consumed and be executing or already committed. Log execution outcomes by batch ID; a failed Discord receipt never repeats the mutation. Durable recovery and persistent idempotency are outside this phase.

### Phase 3: Authenticated API

Expose structured problem operations using personal, revocable tokens bound to a verified owner and guild. A token's permissions cannot exceed its owner's current authority. Protect credentials and check permissions for every operation.

Define partial updates and retry behavior explicitly. Retryable writes must preserve their mutation and saved outcome atomically. Return results to the caller; optional Discord receipts report delivery separately from mutation success.

Reuse the application's problem operations and storage. Review hosting and network access before exposing the API remotely.

## 3. Alternatives and Tradeoffs

| Approach | Benefit | Cost or limitation |
| --- | --- | --- |
| Improve prefix commands only | Small change to an existing interface | Positional editing remains cumbersome |
| Slash forms first | Native workflow where mentors already work | Interaction timing, registration, and temporary state |
| Text batches | Several prepared changes without API credentials | Grammar, preview, atomic execution, and confirmation lifetime |
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
