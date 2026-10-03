# Foundation Review: Problem Models, Mappings, and Planning

**Status:** Models and mappings reviewed and committed locally; documentation restored and updated for this delivery. Runtime implementation remains pending.

**Updated:** 2026-10-03.

**Branch:** `mentor-command-planning-models`, in the existing `mentor-command-planning` worktree.

**Parent:** [Implementation plan](slash-problem-management.md#foundation--current-models-mappings-and-planning) and [ADR 0003](../adr/0003-mentor-problem-management-interfaces.md).

## Purpose

Prepare the representations needed for mentor problem forms before implementing the Discord interaction flow. This delivery consists of the models commit, the mappings commit, and a documentation commit. The earlier larger draft is preserved separately on `wip/problem-modal-draft`.

## Read the Current Change in This Order

1. [Application request models](../../app/services/service_model/model.go): creation uses ClassCode; the new complete-replacement update request includes ProblemID and numeric ClassID. Nil optional values represent clearing when that request is eventually saved.
2. [Form models](../../app/presentation/discord/problem_modal.go): `CreateProblemModal` and `EditProblemModal` share raw `ProblemModalFields`; both include ClassCode. Edit ProblemID is display metadata, outside the editable fields.
3. [Service preparation](../../app/services/adapters.go): `MapProblemToUpdateProblemParams` copies saved values into a pending update request without lookup or validation.
4. [Presentation mappings](../../app/presentation/discord/problem_adapters.go): both directions of application params/form conversion live beside the form models. The edit reverse mapping uses the caller-supplied target ID and destination ClassID instead of trusting display metadata.
5. [Service mapping tests](../../app/services/problem_params_test.go) and [form mapping tests](../../app/presentation/discord/problem_adapters_test.go): preservation of distinct problem values, optional fields, prefills, raw correction text, and edit identity.
6. [ADR](../adr/0003-mentor-problem-management-interfaces.md) and [implementation plan](slash-problem-management.md): phased scope, accepted tradeoffs, and the next discussion.

## Mapping Walkthrough

Initial edit prefill follows:

```text
Saved Problem -> UpdateProblemParams -> EditProblemModal
```

The caller resolves ClassID to ClassCode before presenting the edit form. The mapper copies ProblemID into the form for display. No problem is updated by this preparation.

Submission mapping follows:

```text
Raw modal values + caller-owned target/resolved class -> pending request params
```

Saving will require actor/session checks and application validation outside these mappings. The current code defines no runtime session or submission handler and does not yet route the existing service update through UpdateProblemParams.

| Input or condition | Current mapping behavior |
| --- | --- |
| Saved nil difficulty/topic | Service preparation retains nil |
| Missing difficulty on prefill | Form shows the existing Not specified label |
| Missing topic on prefill | Form contains blank text |
| Explicit raw values on prefill | Copied without trimming or validation |
| Submitted blank optional text | Pending request receives nil |
| Submitted Not specified difficulty, ignoring case/outer whitespace | Pending request receives nil |
| Other submitted text | Retained raw for later application validation |
| Modal display ID differs from supplied target | Update request uses the supplied target |

A pending request is not a validated request. The models and mappers introduce no URL, length, difficulty, class-existence, or permission validation. Those shared rules are later application work. Field length constants still need agreement; the rejected 4,000-character class default is not a domain decision.

## Boundaries

Presentation owns Discord form models and their bidirectional mappings, and imports application model types. Services own saved-problem preparation and do not import presentation. Controllers will coordinate actor identity, class resolution, sessions, response delivery, and application operations. Public mapping names use `MapSourceToDestination`; private conversion helpers use `mapSourceToDestination`.

DiscordGo remains `v0.28.1` in this delivery. There are no Label builders, submission parsers, payload fixtures, slash registration, cache/session code, or shared manager yet. The ADR records the planned SDK pin and the tagged-release follow-up; it does not claim the dependency was upgraded here.

## Verification Recorded for the Models and Mappings

- `GOCACHE=/tmp/hurion-step-1-go-cache go test ./app/services ./app/presentation/discord` — passed after the final test-table revision.
- Race-enabled tests for services, Discord presentation, and controller adapters — passed during mapping placement review.
- Focused `go vet`, bot compilation, formatting, and whitespace checks — passed during the mapping/rename reviews.
- Independent mapping-placement and Create-naming reviews — passed.

These results cover the current models/mappings. They do not establish SDK Label compatibility or live Discord acceptance. Documentation restoration is checked separately for stale implementation claims and valid local links.

## Next Reviewable Work to Discuss

The user proposed upgrading DiscordGo and demonstrating a basic modal opened through a slash command. Before coding, agree the selected SDK version, test command name, form fields, registration/removal, and immediate response path. A private submission confirmation can prove entered values are decoded without introducing problem writes, validation retries, or an editor cache.

Discuss that proposal after publishing this delivery. Full problem-form transport, shared application validation, service-contract migration, Otter sessions, and complete slash workflows remain later reviewed work. Branch choices and live test-guild actions require explicit agreement.
