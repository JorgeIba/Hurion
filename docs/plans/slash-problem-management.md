# Implementation Plan: Slash Problem Management

**Parent decision:** [ADR 0003](../adr/0003-mentor-problem-management-interfaces.md).

This plan describes the intended capabilities, boundaries, and review stages. Implementation details and verification results belong in the code and the review of each increment.

## 1. Phase-1 Scope

Provide problem creation and editing through Discord slash commands and forms, including edit autocomplete and private results. Preserve identity and solved records when editing or moving a problem between classes.

Keep existing prefix problem commands available and apply the shared application policies to both interfaces. Browsing remains with the existing Discord and web lists. Batch execution, API access, MCP, and destructive slash commands follow separate designs.

## 2. Mentor Experience

| Operation | Expected behavior |
| --- | --- |
| Create | Open a form with class defaulted from the channel when available |
| Edit | Select a problem by ID or autocomplete and open its prefilled form |
| Correct invalid input | Explain the error and reopen the submitted values for correction |
| Save | Return a private result identifying the problem and its class |

Both forms contain Class, Name, URL, Difficulty, and Topic. Class, name, and URL are required. Difficulty and topic are optional. Class is editable; a destination class must exist. The edit target is visible and immutable, with its authoritative identity held by the editor session.

Forms allow multiword names and topics. Blank optional fields clear their stored values, and unspecified difficulty is normalized consistently. Use the shared application rules for required fields, accepted difficulty values, URL validity, and length limits. Transport limits and domain limits have separate purposes.

Retain recognized submitted values when validation fails. Existing invalid data must have an actionable repair path; do not silently truncate it. Receipt rendering must handle user-supplied text without triggering mentions.

## 3. Interaction Lifecycle and Timing

Keep command registration aligned with the application during normal bot startup and limit it to the configured guild and owned commands. Incoming-event callbacks route interactions by their type and command or component identity. Mock and script modes remain independent of live registration.

Open the form directly from the slash invocation when preparation can finish within Discord's response deadline. Use the interaction's role snapshot for initial access checks. Keep the accepted timing limitation visible at the preparation boundary; if preparation becomes too slow, introduce a separately acknowledged form-opening action.

Acknowledge form submission promptly before live authorization and application work. Verify current permissions before saving. Autocomplete returns bounded suggestions promptly and does not grant write authority.

Interaction response delivery must meet the protocol's deadlines independently of the ordinary message queue. Handle delivery failures separately from operation outcomes so a failed receipt cannot cause another write.

## 4. Temporary Editors and Retries

Use the editor-storage decision in the ADR. Bound session lifetime, total capacity, and per-actor usage. Apply bounded request admission before expensive work; cache capacity alone does not limit request volume. Concrete limits are chosen and reviewed with the implementation.

Bind each editor and submission attempt to its owner, guild, and target. Claim a submission atomically so concurrent or repeated delivery cannot execute the same attempt twice within the process. Protect an in-progress save from expiration or eviction and retain its outcome briefly for duplicate responses.

Validation failures retain the proposed fields and permit a fresh submission attempt. Successful or uncertain writes are not automatically replayed. Expired sessions and sessions lost on restart require starting again.

Distinct editors follow the ADR's accepted last-successful-write-wins policy, including prefilled values and class moves. An edit preserves problem identity and solved records; a deleted target is not recreated.

## 5. Application Boundaries

Keep Discord event parsing and form presentation at the transport boundary. Adapt their values to typed application requests rather than constructing a positional message command.

Share problem-management authorization, validation, and writes across slash and prefix interfaces. Keep identity, class resolution, and optional-value semantics explicit. Pure mappings perform no lookup, authorization, or validation.

Services remain independent of Discord presentation types. Controllers coordinate application work and temporary editor state. Reuse the existing persistence and exporter behavior for problem writes.

Use native SDK form support according to the [ADR's compatibility decision](../adr/0003-mentor-problem-management-interfaces.md#discordgo-version-and-upgrade-follow-up). Concrete SDK calls and model-to-form mappings are reviewed with the implementation.

## 6. Reviewable Stages

Implement one cohesive increment, let the user review and question it, apply revisions, and publish only after approval. Resolve PR comments and wait for merge before dependent work. Agree branch and base choices with the user.

| Stage | Purpose | Main review question |
| --- | --- | --- |
| Foundation | Request and form models, mappings, and architectural direction | Are representations and boundaries clear? |
| Command integration | Command definitions, registration, and basic interaction routing | Does the bot receive and dispatch the intended command? |
| Form transport | Native form display, prefilling, and submission parsing | Can mentors enter data and can the bot read it correctly? |
| Problem operations | Shared authorization, validation, and prefix integration | Do both interfaces enforce the agreed application rules? |
| Editor lifecycle | Temporary drafts, bounded admission, and duplicate prevention | Are editor identity and execution state managed correctly? |
| Add workflow | Connect form entry, correction, saving, and results | Can a mentor create and correct a problem? |
| Edit workflow | Load and update an existing problem, including class changes | Are identity, solved records, and overwrite semantics preserved? |
| Autocomplete | Bounded problem selection by ID and name | Can mentors find a target without unbounded work? |
| Release review | Operational guidance and real Discord validation | Is the complete workflow ready for rollout? |

### Foundation — Current models, mappings, and planning

Review the problem representations and conversion boundaries together. Preserve identity and optional values, and distinguish raw form input from application write parameters. The [foundation review guide](slash-problem-management-step-1.md) provides context for this boundary.

Stages may be split into smaller increments when needed for a focused review. Specify exact functions, files, data structures, and checks when discussing the increment being implemented.

## 7. Review and Verification

Architecture reviews should challenge the proposal and nearby existing contracts, compare a credible simpler alternative, and explain which abstractions earn their cost. Repository precedent and passing tests are evidence, not proof of sound design.

Inspect callers and related create/edit flows together, including identity ownership, class moves, optional clearing, and errors. Report grounded concerns and accepted tradeoffs without expanding implementation scope.

Choose verification that matches the changed behavior and the repository's [testing guidelines](../../.ai/TESTING.md). Report actual evidence and limitations in the increment review. Automated checks and live Discord acceptance provide different confidence.

Before rollout, verify the complete mentor journey, permission changes during editing, correction and expiry behavior, duplicate handling, class moves, preservation of solved records, and prefix compatibility. Review registration behavior and rollback in a test guild before production deployment.
