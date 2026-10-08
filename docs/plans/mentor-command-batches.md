# Implementation Plan: Mentor Command Batches (Phase 2)

**Parent decision:** [ADR 0003](../adr/0003-mentor-problem-management-interfaces.md#phase-2-explicit-command-batches).

Deliver one reviewable PR at a time. Review and merge each increment before implementing its dependent PR. Enable a slash command only when its complete preparation, confirmation, and atomic write workflow is ready. This plan does not authorize deployment.

## 1. Input Contract

`/problem add_batch` accepts one existing prefix-style creation command per nonempty line:

```text
!add_problem <class_code> <url> <difficulty or -> <topic or -> <priority or -> <ProblemName...>
```

Class is explicit on every line; one batch may target several classes. Difficulty and priority use the existing case-insensitive codes. `-` leaves an optional value unset. Topic is one whitespace-delimited token, as in the prefix command; the remaining tokens form the multiword name. Whitespace is handled like the existing prefix interface, without quoting or escaping rules.

`/ac batch` accepts only explicit student AC commands:

```text
!ac <Discord user mention or numeric ID> <positive problem ID>
```

Accept `<@ID>`, `<@!ID>`, or the raw numeric ID using the existing user parser. Do not resolve usernames or script aliases. Problem IDs must fit a positive signed 64-bit integer. Self-AC syntax, extra arguments, and other commands are rejected.

Input must be UTF-8 and at most 256 KiB (262,144 bytes). Ignore blank lines, support LF and CRLF, and retain the original one-based line numbers. At most 100 nonempty operation lines are allowed. Comments, Markdown fences, and script instructions are not part of the grammar. Reject empty or oversized input explicitly; never truncate it.

Parsers are pure transport helpers. Problem parsing reuses existing normalization and value validation. Neither parser resolves classes, queries Discord or storage, checks stored duplicates, or writes. Collect line errors and return no usable entries if any line fails. An input-wide error uses line zero. Parsing success is not full batch validation.

Preserve repeated problem entries, including identical ones. Preserve repeated AC entries during parsing so preparation can report every duplicate with its line number; never silently deduplicate. No executable routing through prefix handlers or the JSONL script runner.

## 2. Preparation and Confirmation

Preparation performs read-only class/problem, authorization, enrollment, and AC duplicate checks. Avoid `SyncParticipants` and AC write methods: these can create users. Collect errors by line and require correction and a fresh submission before confirmation.

Keep the prepared, normalized operations and resolved target identities in an Otter session with a random batch ID, owner, guild, and fixed five-minute deadline. Use a capacity of 100 sessions. Explain expiry/eviction/restart honestly; five minutes is the maximum pending lifetime, not guaranteed retention. Show a private preview with problems and classes, or AC counts by distinct students, problems, and classes, with enough detail to review the proposed changes.

Use Confirm and Cancel buttons. Acknowledge confirmation promptly; an acknowledgement failure stops before claiming or writing. Verify current permissions, then atomically verify owner/guild/deadline and remove the session. Cancellation uses the same claim rule. A mismatched actor cannot consume the session. Only the winning handler proceeds; database work and Discord requests never execute inside Otter's `Compute` callback. Once claimed, expiry cannot interrupt execution.

Read current roles directly from Discord, once per unique student per preparation/confirmation. Do this outside the SQLite transaction. Bind database targets to the prepared identities; reject deleted, moved, or replaced targets that invalidate the preview. Recheck database conditions and conflicts using the transaction's connection, then apply all mutations within that same transaction. Preserve existing policies for mentor/admin problem management and mentor ACs.

Do not add a global worker, operation registry, transaction framework, persistent outcome store, or automatic retries. Do not put a consumed batch back in the cache. Pending state is local to one bot process.

## 3. Results and Failure Semantics

Success receipts report actual committed counts and, for created problems, the line-to-ID mapping. An AC insertion that is skipped by a database conflict must abort the batch rather than count as success. Necessary user creation belongs to the same AC transaction and must roll back with it.

Report a confirmed rollback as no changes from this batch. Distinguish that from an outcome that could not be confirmed. Keep receipt delivery separate from writes; failure to deliver success never retries an operation. Log the batch ID with execution outcome, counts, and created IDs so a failed receipt can be investigated.

An unavailable session may be expired, evicted, cancelled, executing, committed, or lost after restart. Do not claim that nothing was applied or recommend automatic resubmission based only on a cache miss. Preventing concurrent claims protects one confirmation session; submitting a new problem batch remains a new creation request.

Correct errors by resubmitting revised text. Do not introduce a separate draft state machine. Add `.txt` input and complete detail attachments in the final transport increment; both paths use the same parser and preparation rules.

## 4. Separate PRs

| PR | Scope | Required evidence |
| --- | --- | --- |
| 1: Contract and parsers | Amend the ADR and document this plan; pure problem/AC text parsers and bounded input | Original line numbers, blank lines/CRLF, normalization, malformed input, size/count boundaries, preservation of repeated problems and AC lines, no usable partial result |
| 2: Atomic problem operations | Read-only problem preparation and all-or-nothing problem persistence using existing validation | Invalid/deleted/replaced classes; intermediate write failure rolls back all rows; repeated problems accepted; IDs returned only after commit; exporter reaches committed state |
| 3: Pending batch cache | Minimal shared owner/guild-bound Otter cache and atomic confirmation/cancellation | Fake-clock expiry; concurrent claims have one winner; wrong owner/guild cannot consume; eviction/missing state; execution survives pending deadline |
| 4: Problem slash workflow | Register and route `/problem add_batch`; multiline form, private preview, Confirm/Cancel and problem receipt | Bot-entry workflow; permission loss; no writes before confirmation; acknowledgement failure; double click/cancel; receipt failure never repeats writes; existing add/edit compatibility |
| 5: Atomic AC persistence | Transactional database checks, user synchronization, and strict AC insertion | Existing/concurrent conflicts abort; all rows and new users roll back; truthful inserted counts; exporter reflects committed results |
| 6: AC preparation/application | Read-only preflight, line diagnostics, duplicate checks, and current student roles | Duplicates within input/existing records; unauthorized students; moved/deleted problems; changed role mappings; validation has no mutations; no network lookups inside transaction |
| 7: AC slash workflow | Register and route `/ac batch`, reusing concrete cache/form/preview components | Full mentor journey; distinct student/problem/class counts; changed permissions/conflicts after preview; private results; existing AC behavior preserved |
| 8: Attachments and acceptance | `.txt` input and long result details for both commands; usage documentation and test-guild acceptance | Bounded downloads, explicit rejection without truncation, same parser/preparation behavior for pasted/file input, long receipts and real Discord interaction checks |

If an increment grows beyond a focused review, split it before implementation; do not compensate with a broad refactor. Preserve unrelated user changes. No schema migration should be assumed merely to support batches.

## 5. Verification Boundaries

Follow the repository's [testing guidelines](../../.ai/TESTING.md). Parser tests are unit tests and need no database or Discord mocks. Transaction checks use real SQLite and assert saved state after success and rollback, including concurrent conflicts. Cache concurrency tests use `-race` and controllable clocks. Workflow tests enter through bot interactions and assert intermediate transitions as well as final storage/output.

Verify scoreboard notification with the real transaction path. Existing per-row hooks are not a post-commit signal; add only the notification needed to avoid a stale export, without replacing the exporter architecture.

Generate mocks when changing interfaces and run the checks appropriate to each PR. Before completion, run the full existing suite and verify both mentor journeys in a test guild. Passing automated tests does not claim live Discord acceptance or deployment.
