# Implementation Plan: Judge Account Sync (Overview)

**Parent decision:** [ADR 0005](../../adr/0005-judge-account-sync.md) (written in stage 1).

This is the overview for the judge sync feature. Each review stage has its own self-contained plan in this directory:

| # | Plan | Delivers |
| --- | --- | --- |
| 1 | [01-foundation.md](01-foundation.md) | ADR 0005, judge port and registry, Codeforces URL parser and API client |
| 2 | [02-accounts-and-linking.md](02-accounts-and-linking.md) | `judge_accounts` table, `/judge link`, **Verify**, `/judge unlink`, `/judge whois` |
| 3 | [03-sync.md](03-sync.md) | `/judge sync`, AC recording, per-judge isolation, report |
| 4 | [04-release-review.md](04-release-review.md) | End-to-end workflow coverage, documentation, live acceptance in a test guild |

Each stage is one PR. A stage starts only after the previous PR is merged into `main`. Each stage plan lists its prerequisites, what to build, and how to prove it works.

## 1. Scope

Students link verified online-judge accounts to their Discord identity and sync their accepted submissions into Hurion. A synced AC is recorded without mentor approval because the judge is the source of truth. The sync replies with a report of the ACs recorded by that call.

The first iteration supports **Codeforces** only. The account model, judge abstraction, and commands are judge-agnostic, so AtCoder, CodeChef, and others can be added later by implementing one `Judge` and registering it.

Out of scope for the first iteration:
- mentor-driven linking
- class-wide or scheduled sync
- Codeforces group contests
- matching Div. 1/Div. 2 twin problems published under different contest IDs
- prefix (`!`) versions of these commands and script-runner support (the script runner only executes prefix commands)

## 2. Decisions

| Topic | Decision |
| --- | --- |
| Account storage | A judge-agnostic `judge_accounts` table; one account per user per judge; a handle is claimable once per judge |
| Linking | Self-service only, with proof of handle ownership (a compilation error on a given problem) |
| Sync permissions | Any server member can sync themselves; syncing another member requires the mentor or admin role, the same policy as `canManageACs` |
| Judge selection | Optional `judge` option; when omitted, sync every linked judge |
| Failure handling | Each judge succeeds or fails independently and commits separately |
| Problem matching | Derived from each problem's URL by the judge's parser; no schema change to `problems` |
| Report | Lists the ACs actually inserted by this call, per judge |
| Solve time | `solved_at` is the sync time, matching other AC paths |
| Undo | A mentor-removed AC that is still accepted on the judge is recorded again by the next sync |

## 3. Codeforces API Facts

Source: the Codeforces API documentation (https://codeforces.com/apiHelp).

- **Requests:** anonymous `GET https://codeforces.com/api/{method}`.
- **Responses:** `{"status":"OK","result":...}` or `{"status":"FAILED","comment":"..."}`.
- **Rate limit:** at most **one request every two seconds**. Excess requests fail with the comment `Call limit exceeded`.
- **`user.info?handles=<h>`:** returns `User` objects. It resolves renamed handles by default (`checkHistoricHandles=true`). `handles` is semicolon-separated, so a handle must be validated before it is sent.
- **`user.status?handle=<h>[&from=&count=]`:** returns the user's `Submission` objects, newest first. Each has:
  - `problem.contestId` (absent for some problemsets)
  - `problem.index`
  - `verdict` (absent while judging)
  - `creationTimeSeconds`
- **No per-problem lookup:** no method reports AC status for a list of problems. Sync therefore fetches the submission history once and filters locally.
- **ACs:** `verdict == "OK"` is an AC. Practice, virtual, and contest submissions all count.

## 4. Data Model

```sql
CREATE TABLE IF NOT EXISTS judge_accounts (
    user_id        TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    judge          TEXT NOT NULL,
    handle         TEXT NOT NULL COLLATE NOCASE,
    verified_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP,
    PRIMARY KEY (user_id, judge),
    UNIQUE (judge, handle)
);
```

- **Why a new table:** `schema.sql` has no migration mechanism. A new table reaches existing databases through `CREATE TABLE IF NOT EXISTS`; a new column on an existing table would not.
- **`judge`:** stores a canonical code (`codeforces`) validated in `service_model`. There is no `CHECK` constraint, following ADR 0004.
- **`handle`:** stores the canonical casing returned by the judge.
- **Verification in progress:** only verified accounts are stored. Challenges live in a bounded in-memory TTL cache and are lost on restart.
- **AC writes:** `AddACBatch` changes to return the inserted pairs (`RETURNING`). A new transactional write records a judge sync: insert ACs, delete stale pending proposals, and update `last_synced_at`.

## 5. Judge Abstraction

The port lives in `app/services/judges.go`, so services depend on an interface, not on HTTP. Implementations live under `app/providers/judges/`.

```go
type Judge interface {
    Code() service_model.JudgeCode
    DisplayName() string
    ParseProblemURL(rawURL string) (service_model.ProblemRef, bool)
    VerificationProblem() service_model.VerificationProblem
    LookupHandle(ctx context.Context, handle string) (string, error)
    Submissions(ctx context.Context, handle string, limit int) ([]service_model.Submission, error)
}
```

- **Registry:** `JudgeRegistry` holds the registered judges in registration order and maps a problem URL to at most one judge.
- **Verdicts:** submission verdicts are normalized (`Accepted`, `CompilationError`, `WrongAnswer`, ...), so services never see judge-specific enums.
- **Codeforces implementation:** it has a process-wide rate limiter (one request per 2.1 seconds), a request timeout, one retry on `Call limit exceeded`, and typed errors.

## 6. Commands

| Command | Who | Behavior |
| --- | --- | --- |
| `/judge link judge:<choice> handle:<h>` | Any server member | Starts verification; private reply with instructions and a **Verify** button |
| **Verify** button | The member who started linking | Completes verification and stores the account |
| `/judge unlink judge:<choice>` | Any linked member | Removes their own account |
| `/judge whois [user:@member]` | Any server member | Privately shows linked handles, defaulting to the invoker |
| `/judge sync [judge:<choice>] [user:@member]` | Members for themselves; mentors and admins for others | Syncs ACs from the selected judge, or all linked judges |

`judge` is a static choice list of registered judge codes. `user` uses Discord's native user picker.

## 7. Sync Behavior Summary

1. **Respond:** defer a private response, which shows progress.
2. **Select judges:** use the requested judge or every linked one. Skip judges that are in cooldown.
3. **Run judges concurrently:** each judge has its own deadline and its own transaction. A failure doesn't affect the others.
4. **Show progress:** the private response is updated as each judge finishes.
5. **Publish:** if at least one judge succeeded, the full report is posted as a public follow-up and the private response is removed. If every judge failed, the private response shows each reason.

Discord sets a response's visibility when it is deferred, so a single response can't switch from private to public. That is why progress and the final report are separate messages.

## 8. Report Contents

| Section | Source |
| --- | --- |
| New ACs | Pairs inserted by this sync, grouped by class, with links |
| Still unsolved | Candidates without an AC on the judge, as a count |
| Already solved / not on this judge | Counts |
| Failed or skipped judges | Reason and how to retry |

## 9. Process Rules for Every Stage

- **Before starting:** read `.ai/CONTEXT.md`, `.ai/TESTING.md`, and `docs/ARCHITECTURE.md`. Follow the layer rules: business rules in services, Discord types only in controllers, providers, and presentation.
- **Branch:** branch from an up-to-date `main` named `feat/judge-sync-<n>-<slug>`. Keep changes surgical to the stage.
- **After interface changes:** run `make generate`.
- **Before opening the PR:** run `go fmt ./...`, `go vet ./...`, and `make test`. They must pass.
- **Tests:**
  - table-driven with anonymous structs
  - `t.Parallel()` at the top of each test and inside each `t.Run`
  - `testify/assert`
  - `gomock` for interfaces
  - an `assertFunc` per case when outputs need inspection
- **Network:** no test may reach the network unless it is behind the `live` build tag.
- **PR description:** it must contain a **Verification** section with the commands run, their summarized output, and the manual checks performed. Anything not verified is listed explicitly.
