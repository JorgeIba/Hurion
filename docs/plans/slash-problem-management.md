# Implementation Plan: Slash Problem Management

**Status:** Direction approved on 2026-09-30. The current branch delivers models, mappings, and planning. Discuss each next increment before coding, then implement and stop for user review.

**Date:** 2026-09-29.

**Updated:** 2026-10-03; reflects the smaller models/mappings delivery, presentation-owned bridges, Create naming, and the proposed next SDK/basic-modal proof. [Current delivery review guide](slash-problem-management-step-1.md).

**Parent decision:** [ADR 0003](../adr/0003-mentor-problem-management-interfaces.md).

**Baseline:** GitHub `main`, commit `c348537` (`fix(deploy): isolate production replica and request message content (#70)`).

## Current Delivery and Next Discussion

Branch `mentor-command-planning-models` contains two reviewed implementation commits and a third documentation commit: request/form models, their mappings and tests, and this ADR/plan/review guide. It implements no DiscordGo upgrade, form builders/parsers, session cache, shared validation, service-contract migration, slash registration, or runtime workflow. The active dependency remains DiscordGo `v0.28.1`.

Next, discuss a small SDK compatibility proof: upgrade to the reviewed version, open a basic modal from a test-guild slash command, and return a private submission confirmation. Agree its command name, fields, registration/removal, and transport placement before writing code. Automated fake-event tests and live test-guild validation are separate checks. This proof performs no problem writes and needs no editor cache or retry machinery.

The older larger implementation is preserved separately on `wip/problem-modal-draft`; its code and previous verification claims are not the current delivery. Branch/base choices for later work are discussed explicitly; this plan does not require another branch to be created automatically.


## Architecture Review Requirements

Include these requirements in each independent architecture review prompt. Provide the intended behavior and current code without previous review verdicts or an advocated outcome.

- Challenge the proposed design and nearby existing contracts; repository precedent is evidence, not a justification by itself.
- Inspect complete changed modules, callers, service signatures, and datastore behavior. Explicitly distinguish stored models, raw form values, and write parameters; check create/update asymmetries, identity ownership, null semantics, and class-update behavior.
- Compare the proposal with a simpler concrete arrangement, such as direct construction, private controller helpers, or consolidation. Explain which types, functions, and package boundaries earn their cost.
- Separate idiomatic Go requirements from architectural preferences. Consistency alone does not justify another abstraction, and passing tests do not establish good API design.
- Report evidence-backed findings, alternatives, tradeoffs, and unreviewed integration assumptions. Identify relevant existing design debt even when its correction belongs to a later increment; do not silently expand implementation scope.
- Give an explicit recommendation with its review scope and limitations. A review need not find a defect, but approval must explain why the design was retained after considering alternatives.

## 1. Phase-1 Scope

Implement `/problem add` and `/problem edit` in one configured guild, with problem autocomplete, forms, private results, and current permission checks. Preserve problem IDs and solved records on edit. Existing prefix commands continue working through shared problem-management authorization and validation.

Defer `/problem list`; existing `!list` and the deployed web Problems tab cover browsing. Autocomplete selects an edit target without adding another browser. Text batches, HTTP routes, API keys, MCP, a web editor, slash migration of all existing commands, and destructive slash commands are later work. The implementation should not refactor unrelated score, enrollment, deployment, or student-confirmation behavior.

## 2. Proposed Mentor Experience

| Command | Options | Result |
| --- | --- | --- |
| `/problem add` | No class option; class is a required form field | Private preparation message, then **Open form** button; prefill class from the current linked channel |
| `/problem edit` | Required `problem`, with autocomplete by ID/name | Private problem summary, then **Open editor** button |

`problem` is a string option whose selected value is a numeric problem ID. Suggestion labels include the name, ID, and class; the name is not an identity or a uniqueness guarantee. A mentor can enter an ID directly. Return at most 25 deterministic autocomplete choices, with labels of at most 100 characters using Unicode-safe truncation. Keep the ID visible and shorten the name/class display as needed. Prioritize an exact numeric ID, then matching name prefixes, then other name matches, with ID as the stable tie-breaker. Limit lookup time and database result count; autocomplete must not enumerate every problem through unbounded per-class reads. Unknown IDs/classes return an actionable private error.

Editing prefills the required Class field from the selected problem's current class, regardless of the channel where the command was invoked. The mentor may change it to move the same problem to another existing class; preserve the problem ID and solved records.

**Problem ID display approved on 2026-10-01:** show the immutable target in the edit modal title, for example `Edit problem #42`. `EditProblemModal.ProblemID` is display metadata copied from the saved problem. It is not an editable field and is not recovered from submitted text. The controller's owned session supplies the authoritative target for writes and restores the display ID when reopening decoded correction values. Keep all five component slots for the editable fields.

**Class entry agreed on 2026-09-30:** the add form always requires a class. Use the existing channel-to-class mapping to prefill its text field when available. If the channel is not linked, open the form with an empty required class field for the mentor to fill in. A lookup failure is an actionable preparation error, not an inferred class or an empty-success result. Mentors may change the prefilled code before adding; validate the submitted code against existing classes at submission. Channel inference supplies a default only. **Edit-class support approved on 2026-10-01:** editing displays the existing class in the preparation message and prefills it in a required editable Class field. Resolve the problem's ClassID to its class code during preparation; a lookup failure is a preparation error. Resolve and validate the submitted destination class again before writing. Correction forms retain the proposed class code. Discord slash option definitions have no per-invocation default-value field, so this prefill belongs in the modal.

### Form

| Field | Add | Edit | Proposed validation |
| --- | --- | --- | --- |
| Class | Required text field; prefilled from linked channel or empty | Required text field; prefilled from the problem's existing class and editable | Trimmed, nonempty, existing class code; length limit to be agreed |
| Name | Required, empty initially | Prefilled | Trimmed, nonempty, single line; length limit to be agreed |
| URL | Required, empty initially | Prefilled | Absolute HTTP/HTTPS URL with host; length limit to be agreed |
| Difficulty | Optional text, prefilled Not specified | Existing value prefilled as text | Trimmed, case-insensitive Easy/Medium/Hard/Not specified; blank means unspecified |
| Topic | Optional, empty initially | Existing value prefilled | Trimmed, single line; spaces allowed; length limit to be agreed |

The table describes future application validation, not rules implemented by the current mappings. Agree shared class/name/URL/topic limits before coding them; use those constants for both form constraints and application validation. Discord's input ceiling is a transport constraint, not the proposed class-code limit.

Use Discord's native Cancel and Submit controls. Cancel produces no write. Blank or Not specified maps to null difficulty; an empty topic maps to null. Normalize difficulty text in the shared manager (step 2), accepting case-insensitive Easy/Medium/Hard/Not specified and rejecting other values with an actionable error. The raw form decoder preserves bounded text so a validation retry can reopen what the mentor typed. Prefix input keeps its existing placeholder syntax, translating into these same typed values. Do not fetch or scrape the submitted URL.

Show the created or edited problem ID, class, and meaningful changed fields in the private result. Suppress mentions and escape user-supplied text. Routine form submission is the write action; it does not need another confirmation button. Validation failures include an **Edit again** button that reopens the proposed values, because a modal cannot be returned directly from a modal-submit response.

**Compatibility proposal to review:** shared field validation intentionally tightens prefix add/update as well as slash writes. Prefix grammar remains unchanged: names already allow spaces; topics remain one token there, while forms allow multiword topics. Existing records remain readable. A write must supply conforming values; a legacy invalid URL must be corrected before that edit saves. If an existing value cannot fit the reviewed form limits, fail preparation with a field-specific repair instruction using the existing prefix update; never silently truncate it. Add tests for these deliberate behavior changes rather than presenting them as unchanged compatibility. Reject missing, duplicate, and unexpected submitted fields; do not trust client-side required-field checks. Retain only bounded recognized fields in drafts, and sanitize/bound receipt and error rendering.

## 3. Interaction Lifecycle and Timing

Discord requires an initial response within three seconds. A modal cannot be opened after that interaction has already been deferred. Use this explicit flow for both add and edit:

1. Receive the slash event through the existing gateway session; immediately acknowledge with a private deferred response.
2. Check guild, actor, test-mode whitelist, and current management permission. For add, look up the channel's class default, allowing an unlinked channel to produce an empty required class field. For edit, resolve the selected problem and its class context. Prepare a short-lived form session.
3. Edit the private response with the target summary and **Open form** or **Open editor** button.
4. The button interaction performs only local session/actor checks and responds immediately with the prepared modal. It performs no Discord member lookup on this acknowledgement path.
5. Modal submission is a new interaction. Immediately acknowledge privately, then recheck current permissions, validate the values (including either form's submitted destination class code), and perform the write.
6. Edit that private response with success or an actionable failure. If response delivery fails after a successful write, record the write outcome and do not execute it again.

This adds one button click before the form compared with the concept preview. It makes permission checks and data preparation independent of the initial response deadline. Unauthorized users never receive a prepared editor; losing a role while the editor is open prevents submission.

Autocomplete responds immediately with bounded choices rather than deferring. Gate suggestions using the authenticated interaction member's roles and test-mode whitelist; enforce live authorization before preparing an editor and before writing. Empty/failed lookups return an empty choice set and a log entry. Suggestion visibility is not write authorization.

Keep response delivery in `app/dispatcher` and the Discord provider/session adapter. Add a narrow interaction responder there for immediate acknowledgement, autocomplete, modal opening, and private response editing. It returns delivery errors to the controller. Do not queue initial acknowledgements behind ordinary channel messages. This is an explicit interaction-specific extension to the queued messaging rule in [.ai/CONTEXT.md](../../.ai/CONTEXT.md); services still perform no Discord I/O.

## 4. Form Sessions, Replay, and Concurrent Edits

Use an in-memory session store in the interaction adapter, backed by Otter v2 behind a small `EditorStore` wrapper. The user selected Otter on 2026-09-30. Pin a reviewed stable version during implementation. Store an opaque random session ID, actor ID, guild ID, operation, target problem ID for edits, prefilled/proposed class code and other field values, expiry, and execution state. Do not put the submitted name, URL, or credentials in component IDs or logs.

Proposed defaults: ten-minute expiry from creation, a maximum of five unfinished sessions per actor/guild, and 200 total records including retained completed outcomes. Reads and validation retries do not extend the original draft deadline. Expired sessions and sessions lost on restart require rerunning the slash command. Pending form sessions do not require a database migration in phase 1. State and capacity transitions are atomic under the wrapper lock; an approximate cache-size estimate is not an admission guarantee. Cleanup releases capacity for expired records, without evicting valid drafts to admit new ones.

Otter supplies cache operations and expiration maintenance; `EditorStore` owns identity binding, atomic attempt transitions, and admission limits. Protect multi-step checks/transitions with a mutex; cache thread safety alone does not make them atomic. Keep Otter's size-based eviction disabled for editor records and reject new editors at the wrapper's capacity limit rather than evicting active drafts. Use a fixed absolute draft deadline when configuring expiration. An already-claimed save must remain protected from expiration until it finishes, followed by brief result retention. Otter documents custom expiration and pinning for stateful entries; verify those behaviors against the pinned version. Hold no store lock across Discord or database I/O. Keep a lightweight request throttle before expensive permission/database work; cache capacity does not limit request volume. No loaders, automatic refresh, persistence, or general caching framework are needed for this store.

Bind every button and submit to the original actor and guild. Atomically claim a session for execution so concurrent or repeated submits cannot perform the same write twice during the process lifetime. Keep the result until expiry. A claimed session with an uncertain write outcome is not automatically replayed; a mentor checks the problem list before intentionally creating a new request. This is not a durable exactly-once guarantee across process failure.

### Minimal session lifecycle

One session is created per slash invocation and keyed by an opaque session ID. It holds one current attempt ID and three states; keep no attempt-history map.

| Event | Required state | Transition and effect |
| --- | --- | --- |
| Prepare command | New | Store initial fields, owner/guild, fixed expiry, and current attempt; enter Open |
| Open form / Edit again | Open, matching owner/guild/attempt | Copy current draft and show modal; retain the same state and attempt |
| Submit | Open, matching owner/guild/attempt | Atomically claim Processing; run permission checks, validation, and write outside the store lock |
| Invalid recognized fields | Processing | Preserve bounded submitted values, generate a new attempt ID, return Open, and render Edit again bound to the new attempt |
| Successful write | Processing | Store the result and enter Finished before sending the receipt |
| Permission denial, deleted target, or uncertain write | Processing | Enter Finished with an actionable outcome; never automatically replay |
| Old attempt / foreign actor / missing session | Any | Reject without changing the current draft or performing a write |
| Repeated current submit | Processing or Finished | Report processing or retained result; do not claim another write |

Canceling or failing to deliver a modal leaves Open unchanged. Several opened copies share one attempt; the first claimed submit wins. Unsubmitted client edits are not available to the bot. A rejected submit's new attempt is prepared before its retry button is rendered, so opening/canceling does not consume that button. A post-write delivery failure leaves Finished intact.

Reject activity after the draft deadline. Let a save claimed before that deadline finish; never expire or reopen its Processing record. Retain a terminal outcome until the later of the original deadline or one minute after completion, still within global capacity. If a validation failure finishes after expiry, return an expired-draft outcome rather than reopen it. Missing sessions always reject; never reconstruct executable state from component IDs.

### Admission controls

Check configured guild, actor presence, owned namespace, interaction role snapshot, and test-mode whitelist locally before expensive work. Current live authorization remains required before editor preparation and writing. Apply a bounded in-memory limiter before Discord member requests and database work, without waiting in an unbounded queue. Initial proposed settings are ten expensive command/submit actions per minute per actor (burst five), three autocomplete lookups per second per actor (burst five), an aggregate budget of ten expensive requests per second, and four simultaneous expensive operations. Treat these as tunable starting values, not measured capacity claims. Local modal opening uses only session checks and immediate responses.

Limiter keys are actor/guild pairs, with idle expiry and a hard entry cap (proposed 200); throttle new keys when full. Autocomplete allocates no session and returns an empty choice set promptly when limited. Other owned interactions receive a private busy message. A submission refused before processing should retain its bounded values in the owned draft where feasible and offer another attempt; it performs no write. Never hold a store lock while sending a response or acquiring network/database resources.

**Noted and accepted by the user on 2026-09-30:** phase 1 uses last successful write wins for editable problem fields. If A opens an editor, B saves changes, and A then submits, A's submitted name, URL, difficulty, and topic replace the currently stored values. With edit-class support approved on 2026-10-01, the submitted class also follows this policy: an unchanged prefilled class can move the problem back after another mentor moved it. This includes prefilled fields that A did not change. The same behavior applies when B used a prefix update. Concurrent edits can therefore overwrite another mentor's changes; this tradeoff is accepted to simplify the initial implementation.

Do not implement expected-snapshot matching, revision fields, conflict rejection, or a conflict/reload workflow in phase 1. A conditional update is feasible, but its datastore contract and recovery flow add work beyond the accepted overwrite behavior. Slash updates match by problem ID and replace the editable values, including class membership resolved from the submitted class code, preserving the row's ID and solved records. Return not found if the target was deleted; do not recreate it. Authorization, input validation, and prevention of duplicate execution of the same submission remain required. Conflict protection can be revisited if actual usage warrants it.

## 5. Shared Problem-Management Path

Add one small actor-aware manager around existing problem and class services. Proposed methods cover preparing an add/edit request, creating a problem, updating editable fields by ID, and searching for edit targets. Names and file placement can follow the existing service conventions; do not introduce a generic command bus.

The manager receives plain user/guild IDs, obtains current authorization through a narrow provider interface, normalizes input (including case-insensitive difficulty text and blank/Not specified to null), and calls the existing domain services. It returns typed results/errors. Controllers adapt requests and format responses; the session store owns the temporary form lifecycle. Domain services and SQLite models receive no Discord objects.

Proposed role policy: the configured mentor **or** admin role permits problem management across configured classes. Tokens and additional class scopes belong to later phases. Current add/update controllers check only mentor role, so adopting this policy intentionally enables admin-only users for those prefix operations too. Document and test that change. Leave unrelated command role policies intact.

Route existing add/update prefix controllers through the same policy and validation. Their positional parser continues to translate `-` placeholders into optional values; the form adapter does not manufacture a message or rebuild a positional command string. Prefix updates retain their current operation semantics, including their explicit class argument. Slash forms likewise accept an explicit destination class code and can move a problem between classes.

Preserve the existing exporter path: problem writes use the existing datastore/update-hook wiring. No separate exporter invocation or scoreboard API is needed for slash results.

## 6. SDK Compatibility Decision

**Revised during user review on 2026-09-30:** defer the dropdown. Difficulty is a prefilled text field: show the existing value on edit, or Not specified for a new/unspecified problem. The placeholder lists accepted values. Clearing the optional field means unspecified. Shared normalization/validation happens in step 2 and preserves the same stored difficulty enum/null values.

**SDK decision approved on 2026-09-30:** use DiscordGo `v0.29.1-0.20260214123928-f43dd94faaac`, pinned to upstream commit `f43dd94faaacd5b163e9e783f14b5bd8be639fc9`. Its [component implementation](https://github.com/bwmarrin/discordgo/blob/f43dd94faaacd5b163e9e783f14b5bd8be639fc9/components.go) provides native Label encoding and submission decoding. Each field uses one Label containing one TextInput; the display label belongs to the wrapper, and the input carries its custom ID and raw value. Required flags are explicit boolean pointers, including false for optional fields.

[Discord's component reference](https://docs.discord.com/developers/components/reference#text-input) recommends Labels and deprecates Action Rows containing text inputs in modals. Use that structure when implementing the form builders. The tradeoff is a reviewed upstream commit rather than a tagged SDK release. Pinning gives reproducible resolution; full repository regression/build checks cover the dependency upgrade. Offline fixtures do not establish live Discord acceptance; verify forms and desktop/mobile behavior in an approved test guild before rollout.

The earlier tagged-SDK/Action Row proposal is superseded. Difficulty remains text; the dropdown is deferred independently of Label support. No custom encoder, compatibility framework, or local SDK fork is needed.

**Current ownership, reviewed on 2026-10-03:** modal models and both mapping directions belong to `app/presentation/discord`. Initial pending-update preparation belongs to services. Presentation imports `service_model`; services do not import presentation. Public conversions use `MapSourceToDestination`, private conversions `mapSourceToDestination`. Controllers coordinate the operation. Future builders/parsers stay with the Discord presentation code; shared application validation stays outside the mapping functions. No generic adapter or validation framework is required by this decision.

The SDK upgrade and native protocol checks remain future work. The branch still uses `v0.28.1`; recheck available tagged releases before implementing the previously approved pin. Start with the basic slash/modal proof after its scope is discussed, then implement the complete problem-form encode/decode contract in another reviewable change.

**Follow-up:** replace the pseudo-version with a compatible tagged release when available, through a separate reviewed dependency change. Follow the [ADR's SDK upgrade criteria](../adr/0003-mentor-problem-management-interfaces.md#discordgo-version-and-upgrade-follow-up), including protocol tests, full regression/build checks, and test-guild verification before deployment. Record the new tag and results; the upgrade preserves text difficulty and does not add the deferred dropdown.

## 7. Registration and Runtime Integration

- Add an opt-in daemon flag, proposed as `--slash-commands`, and require a configured `guild_id` when enabled. Keep mock/online script modes independent of gateway registration.
- Add explicit operator actions, proposed as `--register-slash-commands` and `--unregister-slash-commands`. Registration validates application/guild identity, upserts the single owned `/problem` command for that guild, reports its ID, and exits. Unregistration removes only the owned guild command. Dispatch these standalone actions before DB initialization, seeding, exporter startup, or gateway opening. Reject combinations with script/mock, seed, exporter, the daemon enable flag, or each other. Registration must not run automatically on every restart or reconnect.
- Register an `InteractionCreate` handler alongside current message/reaction handlers. Restrict slash, autocomplete, button, and modal events to the configured guild and owned command/component namespace. Route unrelated interactions without claiming their response.
- Configure guild-only command availability. Do not use Discord's administrator permission bit as a substitute for Hurion's configured mentor/admin roles. Visibility settings are a convenience; handler authorization is authoritative.
- Upsert only managed command names/IDs. Do not bulk overwrite the application's command list or delete unrelated commands.
- Registration and disabling/unregistering commands are explicit operator actions. A stopped/disabled handler leaves registered commands visible but unavailable until the operator removes them. Document that behavior and rollback instructions.

Proposed flag names are part of this review, not features already implemented. No live guild registration occurs during planning.

## 8. Implementation Sequence

Treat each increment as one cohesive commit or small PR. Every increment includes the tests for its own behavior and must compile with the feature disabled. Do not wait until the final increment to test earlier work. The numbers below are delivery order, not existing commits or PRs. The user authorizes implementation/publication per delivery; this plan alone does not authorize future coding or registration.

| Increment | Review question | Depends on | User-visible effect |
| --- | --- | --- | --- |
| 0. Models, mappings, planning | Are the representations and boundaries clear? | Reviewed models/bridges | Current delivery; no runtime feature |
| 1a. SDK/basic modal proof | Does the selected SDK open and read a basic modal? | Scope discussion | Explicit test-guild proof only |
| 1b. Full form transport | Can the SDK encode/decode the complete problem forms? | 1a and reviewed field limits | Offline contract tests; no problem writes |
| 2. Problem operations | Are authorization, validation, and writes shared correctly? | Approved role/compatibility choices | Deliberate prefix policy/validation changes |
| 3. Editor store | Are drafts isolated, bounded, and executable once? | 2 | None; store/admission tests |
| 4. Interaction transport | Do owned events get the right immediate response? | 1b, 3 | Opt-in handler; no automatic registration |
| 5. Command lifecycle | Can an operator register/remove only our command? | 4 | Explicit registration actions |
| 6. Add workflow | Can a mentor add, correct, and save a problem? | 2–5 | Complete add flow when enabled |
| 7. Edit workflow | Can a mentor edit by ID with the agreed overwrite policy? | 6 | Complete edit flow by numeric ID |
| 8. Edit autocomplete | Can a mentor find an edit target by ID/name? | 7 | Suggestions in the edit option |
| 9. Release review | Are regressions, docs, and rollback ready? | 1–8 | Test-guild validation, then separate production rollout |

### Foundation — Current models, mappings, and planning

**Delivered:** `UpdateProblemParams`, `CreateProblemModal`, `EditProblemModal`, shared raw `ProblemModalFields`, a service mapper from saved Problem to update params, and presentation mappings in both directions. Tests cover different saved problems, raw values, independent missing optionals, explicit empty values, prefill defaults, session-owned target identity, and supplied destination class identity. Creation uses Create consistently in code.

**Boundaries:** mappings perform no I/O or domain validation. Edit params use numeric ClassID, while the form uses ClassCode. Callers will resolve the submitted class and supply the owned target. The current service interface and prefix execution remain as before; the new update request is not wired to writes yet.

### Increment 1a — SDK upgrade and basic slash/modal proof

**Proposed next work, to discuss before coding:** use the approved upstream DiscordGo commit unless a compatible tagged release is reviewed instead. Add a small test-guild slash command that opens a basic Label/TextInput modal and responds privately to its submission. Decide its name, fields, enablement, registration/removal, and handler/provider boundaries with the user. Keep the proof focused on rendering and decoding; it performs no problem mutation and needs no editor store. Do not carry forward the production preparation/session machinery merely to demonstrate a basic form.

**Review evidence:** native outgoing and incoming protocol tests, fake-session event tests, dependency regression checks, and bot compilation. Record the exact version and why it was selected. Actual command registration and live desktop/mobile form acceptance require a separately agreed test-guild run; automated tests do not prove that run happened. Agree how the demonstration command is removed or replaced when full workflows arrive.

### Increment 1b — Complete problem-form encoding and decoding

**Implement after its scope review:** build `CreateProblemModal` and `EditProblemModal` responses using five Labels/TextInputs for Class, Name, URL, Difficulty, and Topic. Accept caller-supplied prefills and display edit identity in the title. Parse submissions by child custom ID, reject missing/duplicate/unexpected fields and incorrect component types, and retain bounded raw correction text. Adopt the reviewed shared field limits without silent truncation. Use independently specified incoming/outgoing fixtures.

**Files:** builders/parsers and adjacent tests/testdata under `app/presentation/discord`. Existing mappings remain in `problem_adapters.go`; no service or controller adapter duplication is needed.

**Review evidence:** required/optional flags, Unicode-safe bounds, prefill preservation, field-order independence, null/display defaults, ID display without an editable ID, malformed payload handling, and native encoding/decoding. No production write flow follows from these helpers.

### Increment 2 — Shared problem operations and prefix bridge

**Implement:** add plain actor/request/result types and a small manager. Reuse existing role-provider checks; allow a configured mentor OR admin and fail closed on unresolvable authority. Keep Discord objects out of service/data layers. For initial edit preparation, fetch the saved problem, construct pending params with `MapProblemToUpdateProblemParams`, resolve their ClassID to a class code, and return the prepared values for presentation. This preparation is not a write or a guarantee of valid legacy field values. At submission, validate and resolve the submitted class code for both add and edit; retain raw modal values on validation failure. For updates, construct `UpdateProblemParams` with the session-bound ProblemID, resolved destination ClassID, and normalized replacement fields, then introduce `ProblemService.UpdateProblem(UpdateProblemParams)` and reuse the existing datastore operation. This replaces the current `UpdateProblemByProblemID(problemID, Problem)` service contract; update its prefix caller and affected mocks in that reviewed increment. No separate editable-fields-only operation is needed now that slash edits permit class moves. Preserve solved rows and exporter hooks. Route only prefix add/update through the manager; retain prefix parsing and explicit class-update semantics. Update their help metadata for the approved policy.

**Likely files:** `app/services`, `app/services/service_model`, `app/data/datastore.go`, `app/data/queries.go`, `app/controllers/mentor/mentor.go`, `app/bot/commands.go`, and regenerated affected mocks. Add focused provider functionality only if needed to check the combined role policy without redundant member requests.

**Review evidence:** real SQLite integration tests cover mentor/admin/denied/error paths, required fields, case-insensitive difficulty text/blank/Not specified and invalid-value errors, optional null clearing, missing/deleted target, prefix and form class moves, invalid destination classes, and preservation of IDs and AC records. Demonstrate the intentional validation changes and legacy repair behavior. No optimistic-concurrency schema or version column.

### Increment 3 — Otter editor store and admission

**Implement:** pin stable Otter v2 and add `EditorStore` plus a small local admission helper. Implement only Create, Open/read, Claim, Retry-with-draft, and Finish behavior needed by the three-state contract. Store copies of bounded field values; do not expose mutable session pointers to handlers. Enforce actor/guild/attempt checks and exact capacity under the store's lock. Use fixed deadlines and protect Processing from expiry; retain terminal outcomes as specified. Limit expensive request admission and limiter bookkeeping.

**Likely files:** `go.mod`/`go.sum`, new `editor_store.go` and `editor_limits.go` under `app/controllers/mentor`, with adjacent tests. No global cache abstraction or session table.

**Review evidence:** fake-clock tests for expiry and cleanup; race-enabled concurrent admission/claim tests; independent users and multiple drafts for one user; cancel/reopen without attempt consumption; validation retry preserving fields; stale attempt rejection; full-cache refusal; Processing across expiry; terminal result retention; bounded limiter keys. Prove Otter configuration honors these wrapper rules. Avoid tests based on wall-clock sleeps.

### Increment 4 — Immediate interaction transport and routing

**Implement:** add a narrow interaction responder within the dispatcher/provider boundary for private defer, response edit, modal response, and autocomplete response. Extend session adapters/mocks only for those operations. Add the guild-scoped owned-event handler and opt-in daemon wiring. Parse actor identity from the event and session/attempt IDs from the owned namespace; never take actor identity from component IDs. For slash commands and modal submissions, validate locally and privately defer before live authorization/database work. For autocomplete, run local checks/admission, perform the bounded search, and send the immediate choices response without deferring. For Open form/Edit again buttons, perform local checks and immediately return the prepared modal without deferring. Unrelated events remain unclaimed. Start/stop store maintenance with the enabled daemon.

**Likely files:** `app/dispatcher`, `app/providers`, `app/bot/bot.go` plus a new interaction router, `app/bootstrap`, `app/main.go`, and affected mocks/tests. Keep the existing channel-message queue intact.

**Review evidence:** fake-session call-order tests prove slash/submit deferrals precede live role/DB work and bypass the ordinary queue; autocomplete uses bounded lookup followed by one choices response; buttons use local checks followed by one modal response. Cover guild/DM/member guards, owned/unrelated IDs, whitelist routes, busy responses, disabled feature, and dispatcher errors. Exercise slow dependencies without relying on live Discord timing. No registered command or write flow is required yet.

### Increment 5 — Explicit registration and removal

**Implement:** define the guild `/problem` schema: add has no options; edit requires a string `problem` option, initially accepting IDs and ready for autocomplete. Add standalone register/unregister actions, with flag validation and early dispatch. Upsert only `/problem`; never bulk replace other commands. Do not touch DB files, seed classes, start exporters, or open the gateway in these operator actions.

**Likely files:** new application-command definitions/registration helper under `app/bot` or `app/providers`, `app/bootstrap`, `app/main.go`, and adjacent tests. Add operator documentation alongside the change.

**Review evidence:** fake registration APIs preserve unrelated command IDs; conflicting flags fail before side effects; absent/foreign guild identity fails; repeat registration is an upsert; unregister is limited to the owned guild command. No real registration during automated checks.

### Increment 6 — Complete add and correction flow

**Implement:** slash add -> private preparation -> initial session -> Open form -> modal submit -> manager create -> terminal result -> private receipt. Prefill required Class from the channel; unlinked channels allow required manual entry, while lookup errors fail preparation. Implement invalid-field draft retention and Edit again using the current fresh attempt. Render bounded, escaped, mention-safe messages with no extra approval step.

**Likely files:** new slash controller/rendering helpers under `app/controllers/mentor`, `app/bot` routing, and `workflow`/`integration` fixtures or test helpers. Use the earlier store/responder/manager directly.

**Review evidence:** event-level workflows assert private preparation, class prefill/override, unlinked entry, invalid URL -> retained fields -> corrected success, lost role, capacity/rate denial, canceled/reopened form, other-user attempts, duplicate submissions, and failed success delivery causing exactly one in-process mutation. A restart/expired session rejects rather than reconstructs a write.

### Increment 7 — Complete edit by ID

**Implement:** resolve a numeric problem ID, prefill existing fields including required Class, display the original class as context, and save all submitted replacement fields including the validated destination class. Reuse add's lifecycle, session store, form decoder, and retry renderer. Keep last successful write wins; do not add a conflict warning/merge/reload system. Missing/deleted targets get actionable errors and no replacement record.

**Likely files:** slash mentor controller/helpers, problem-manager preparation, and integration/workflow tests.

**Review evidence:** changing one field preserves ID/AC rows; explicit optional-field clearing; two users editing the same problem hold independent drafts; the last submitted successful write replaces earlier field values; valid class changes preserve the problem ID and solved rows, invalid classes produce a retained correction draft, and an older prefilled class follows accepted last-write-wins behavior after an intervening prefix move; deleted target stays deleted; a legacy value gets the documented repair outcome; replay does not write again.

### Increment 8 — Bounded edit autocomplete

**Implement:** add a limited datastore search joined to class context and expose it through the typed manager/service query path. Use exact-ID priority and deterministic matching/order described in section 2. Specify and test the database's case-matching behavior, including non-ASCII examples; do not imply Unicode case folding the selected SQL does not provide. Add suggestion rendering and wire the edit option's autocomplete flag. Operators who registered an earlier increment must explicitly rerun `--register-slash-commands` after deploying this schema change; a fresh registration uses the final schema. No automatic startup registration, browsing pagination, or editor sessions for suggestions.

**Likely files:** `app/data` query/interface, `app/services` search method, autocomplete controller/presentation, command schema, mocks, and tests.

**Review evidence:** IDs, duplicate names, name matches, long/multibyte labels, choice count/length limits, single-guild restriction, role snapshot/whitelist gates, lookup failures/timeouts, and throttling. Prove lookups are bounded and valid typed IDs still work without selecting a suggestion. Verify re-registration updates the owned edit option while preserving unrelated commands.

### Increment 9 — Release documentation and verification

**Implement:** complete command/help/runtime documentation, reviewer checklist, and explicit rollback procedure. Update the ADR status only after actual approval. Keep API/batch/MCP contracts as later phases.

**Review evidence:** affected mocks regenerated; `make test` with default race detection and bot compilation pass; no unrelated changes in generated files; fake external I/O in automated suites; SDK/CLI choices recorded. Use an explicitly approved test guild to verify native desktop/mobile Label-form acceptance, text decoding, private responses, retries, role revocation, and expiration. Production registration/deployment is a separate explicit rollout action after those results are reviewed.

### Review packet for every increment

Include its purpose, touched boundaries, behavior changes, exact verification results, a relevant fixture or fake-event walkthrough, and remaining limitations. Keep generated mocks with the interface change that requires them. Review the increment before beginning dependent work; the next SDK/basic-modal scope is discussed before implementation. Keep a passing build and the daemon opt-in default off throughout. Commit and PR creation follow later user authorization; this document does not invent branch names or publish anything.

## 9. Verification and Acceptance

Follow [.ai/TESTING.md](../../.ai/TESTING.md) and the existing confidence boundaries.

**Unit and transport checks:** request decoding; problem autocomplete; native SDK Label/text payloads and independent incoming fixtures; every initial event response type; malformed and unrelated component IDs; actor/guild binding; session expiry/capacity; atomic replay claiming; retry draft retention; bounded admission and limiter storage; normalized required/optional fields; mention suppression; registration/removal without unrelated-command deletion.

**Integration checks:** real services and in-memory SQLite; mentor and admin writes; unauthorized and failed-role-lookup denial; linked-channel class prefill and unlinked-channel required entry; edited class default and invalid submitted class; class/problem not found; optional field clearing; accepted last-write-wins behavior including class membership, validated destination class changes; preserved IDs, solved records, and exporter-trigger wiring; legacy prefix operations through the shared path.

**Workflow checks:** slash add -> prepared form -> submit -> created record/private receipt; slash edit -> existing values -> changed field -> preserved solved records; role revoked between preparation/submission; two editors submit different changes with the last successful write winning; prefix update before form submit under the same overwrite policy; target deleted before submission; foreign actor/guild reuse; expired/replayed form; post-write response failure without a second mutation; test-mode whitelist enforcement on every interaction route.

After code changes: regenerate affected mocks, run `make test` with the repository's default race detection, and compile the bot. Any changed SDK payload fixture must verify decoding as well as encoding. Use fake external I/O for automated suites. An explicitly agreed live test-guild check may establish the basic SDK/modal proof in increment 1a. Full workflow smoke testing belongs to the release review; production registration and deployment remain separate rollout actions.

Acceptance requires successful add/edit workflows and edit autocomplete, the documented last-write-wins behavior, no repeat mutation from duplicate execution of one form submission in one process, validated class moves with preserved problem IDs and solved records, current permissions at write time, bounded session/request resources, private actionable results, explicit prefix regression/change coverage, and registration/rollback documentation. Preventing overwrites between distinct editors is explicitly outside phase-1 acceptance.

## 10. Approved Decisions and Increment Reviews

Approved on 2026-09-30:

1. The refined add/edit scope with edit autocomplete; defer `/problem list` to keep existing browsing separate.
2. The mentor-or-admin management policy and its intentional effect on existing prefix add/update.
3. The private preparation message and one-click form opening needed for predictable response timing.
4. Revised during step-1 review: pin the approved upstream DiscordGo commit, use Labels with prefilled difficulty text, and defer the dropdown. Otter's exact stable version is selected in increment 3.
5. Stricter shared field validation and the explicit legacy repair policy; prefix grammar is preserved, but accepted values deliberately tighten.
6. The proposed TTL/capacity/request limits and the broader delivery sequence, now split into smaller compatibility steps. Otter, required class prefill, and last-write-wins are already selected; their implementation still needs verification.

For each increment: implement, stop for user review and questions, apply requested revisions, then publish only after that increment is approved. Resolve PR comments and wait for merge. Reuse this worktree directory. Discuss the next scope and branch/base with the user rather than automatically creating another branch. Production registration and deployment remain later explicit rollout actions.

## 11. References

- [Discord application commands and autocomplete](https://docs.discord.com/developers/interactions/application-commands)
- [Discord interaction response lifecycle](https://docs.discord.com/developers/interactions/receiving-and-responding)
- [Discord modal components](https://docs.discord.com/developers/components/using-modal-components)
- [Discord component reference](https://docs.discord.com/developers/components/reference)
- [DiscordGo releases](https://github.com/bwmarrin/discordgo/releases)
- [Pinned DiscordGo component definitions](https://github.com/bwmarrin/discordgo/blob/f43dd94faaacd5b163e9e783f14b5bd8be639fc9/components.go)
- [Otter v2 expiration and pinning](https://maypok86.github.io/otter/user-guide/v2/features/eviction/)
