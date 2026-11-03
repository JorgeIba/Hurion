# Stage 2: Judge Accounts and Self-Service Linking

**Overview:** [00-overview.md](00-overview.md) · **Previous:** [01-foundation.md](01-foundation.md) · **Next:** [03-sync.md](03-sync.md)

## 0. Context for a New Contributor

Hurion is a Go Discord bot for competitive-programming classes (see `README.md`, `docs/ARCHITECTURE.md`). Stage 1 added a judge-neutral `Judge` port and `JudgeRegistry` (`app/services/judges.go`), domain types (`app/services/service_model/judge.go`), and a Codeforces client (`app/providers/judges/codeforces`). None of them are wired into the bot yet.

This stage lets a Discord member link a Codeforces handle to their Discord identity. They prove ownership by submitting a **compilation error** to a given problem. Members can also unlink and look up linked handles. **No ACs are recorded in this stage.**

Read before starting:
- `.ai/CONTEXT.md`, `.ai/TESTING.md`, `docs/ARCHITECTURE.md`, ADR 0005
- **Schema and data access:** `app/data/schema.sql`, `app/data/datastore.go`, `app/data/queries.go`. Look at `EnsureUser`, `AddACBatch`, and `ExterminateUser`, and at how SQLite constraint errors are detected (`sqlite3.ErrConstraintForeignKey` around `queries.go:647`).
- **Existing slash-command patterns:**
  - `app/bot/discordcommands/ac.go`: command definitions
  - `app/bot/interactionrouter.go`: route maps
  - `app/bot/bot.go`: `HandleInteractionEvent`, `interactionHandler`, `RegisterHandlers`
  - `app/context/interaction.go`: `InteractionContext`
  - `app/controllers/interactiontransport/custom_ids.go`: `ComponentAction` allowlist and custom IDs
- **Controller patterns:**
  - `app/controllers/mentor/ac_proposal_interactions.go` and `ac_interactions.go`: `deferACInteractionResponse`, `editInteractionResponse`, `AllowedMentions`
  - `app/controllers/mentor/problem_draft_cache.go`: `otter` TTL cache with injectable clock
  - `app/controllers/common/helpers.go`: `GetMemberPrincipal`
- **Wiring:** `app/main.go`, `app/services/services.go`
- **Test harnesses:** `integration/setup_test.go`, `workflow/setup_test.go`, `workflow/problem_interactions_test.go` (`recordingDiscordInteractionClient`)

**Prerequisites:** stage 1 merged. Branch `feat/judge-sync-2-accounts` from `main`.

## 1. Goal

When this stage is done:
1. Existing and new databases have the `judge_accounts` table.
2. `/judge link`, the **Verify** button, `/judge unlink`, and `/judge whois` work end to end against a fake judge in workflow tests and against Codeforces in a test guild.
3. A handle can be linked only by the person who controls it, only once per judge, and only one handle per judge per user.

Out of scope: `/judge sync`, AC writes, and documentation for users (stage 4).

## 2. Data Layer

### 2.1 Schema

Append to `app/data/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS judge_accounts (
    user_id        TEXT NOT NULL,
    judge          TEXT NOT NULL,
    handle         TEXT NOT NULL COLLATE NOCASE,
    verified_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP,
    PRIMARY KEY (user_id, judge),
    UNIQUE (judge, handle),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
```

### 2.2 Model and queries

`app/data/models.go`:

```go
type JudgeAccount struct {
    UserID       string
    Judge        string
    Handle       string
    VerifiedAt   time.Time
    LastSyncedAt *time.Time
}
```

Add these to `QueryDatastore` and `UpdateDataStore`, then run `make generate`:

| Method | Behavior |
| --- | --- |
| `GetJudgeAccountsByUserID(userID string) ([]JudgeAccount, error)` | Ordered by `judge`. Empty slice when there are none |
| `GetJudgeAccount(userID, judge string) (JudgeAccount, error)` | `dataerr.NotFoundError` when absent |
| `GetJudgeAccountByHandle(judge, handle string) (JudgeAccount, error)` | Case-insensitive. `NotFoundError` when absent |
| `SaveJudgeAccount(account JudgeAccount, user User) error` | One transaction (see below) |
| `DeleteJudgeAccount(userID, judge string) (bool, error)` | `true` if a row was deleted |

`SaveJudgeAccount` runs one transaction:
1. Ensure the user exists, with the same "don't overwrite a real name with `Unknown`" rule as `EnsureUser`.
2. `INSERT ... ON CONFLICT(user_id, judge) DO UPDATE SET handle = excluded.handle, verified_at = CURRENT_TIMESTAMP, last_synced_at = NULL`.

A `UNIQUE (judge, handle)` violation, meaning another user owns the handle, returns `dataerr.RowConflictError`. Detect it through `sqlite3.ErrConstraintUnique`.

The existing `ExterminateUser` deletes the `users` row, so the cascade removes the user's accounts. Add a test proving this; don't change `ExterminateUser`.

## 3. Service Layer

### 3.1 Wiring

Add a `JudgeAccount *JudgeAccountService` field to `Services`. Keep `NewServices(datastore)` working unchanged, so the many existing tests are untouched. Add:

```go
func NewServicesWithJudges(datastore data.DataStore, judges *JudgeRegistry) *Services
```

`NewServices` calls this with a nil registry.

### 3.2 `JudgeAccountService` (`app/services/judge_account_service.go`)

```go
type JudgeAccountService struct {
    Datastore data.DataStore
    Judges    *JudgeRegistry
    Now       func() time.Time // defaults to time.Now
}

type LinkChallenge struct {
    UserID   string
    Judge    service_model.JudgeCode
    Handle   string                          // canonical, from LookupHandle
    Problem  service_model.VerificationProblem
    IssuedAt time.Time
}

const LinkChallengeTTL = 10 * time.Minute
const linkVerificationLookback = 20      // submissions fetched by CompleteLink
const linkClockSkew = 30 * time.Second   // tolerance before IssuedAt

func (s *JudgeAccountService) StartLink(ctx context.Context, user service_model.User, judge service_model.JudgeCode, handle string) (LinkChallenge, error)
func (s *JudgeAccountService) CompleteLink(ctx context.Context, user service_model.User, challenge LinkChallenge) (service_model.JudgeAccount, error)
func (s *JudgeAccountService) Unlink(userID string, judge service_model.JudgeCode) error
func (s *JudgeAccountService) GetAccounts(userID string) ([]service_model.JudgeAccount, error)
```

Add `service_model.JudgeAccount` mirroring the data model, with `Judge` typed as `JudgeCode`, plus mapping functions next to the existing mappers.

**`StartLink`:**
1. If `judge` isn't registered, return `ValidationError`.
2. Call `LookupHandle`. Map `judgeerr.InvalidHandleError` to `ValidationError`, `HandleNotFoundError` to `NotFoundInfoError`, and other errors to `InternalError` with a retry-later message.
3. If `GetJudgeAccountByHandle` finds the canonical handle owned by **another** user, return `ConflictError` ("already linked to another member").
4. If the user already has **this** handle on this judge, return `ConflictError` ("already linked").
5. Otherwise return a challenge with `IssuedAt = Now()`. **This method writes nothing.**

**`CompleteLink`:**
1. If `challenge.UserID != user.UserID`, return `InternalError`.
2. If `Now() - IssuedAt > LinkChallengeTTL`, return `ValidationError` ("expired, start again").
3. Fetch `Submissions(ctx, handle, linkVerificationLookback)`.
4. Succeed only if some submission has `Problem == challenge.Problem.Problem`, `Verdict == VerdictCompilationError`, and `SubmittedAt >= IssuedAt - linkClockSkew`.
5. If none matches, return `NotFoundInfoError` with an actionable message.
6. On success, call `SaveJudgeAccount`. A data `RowConflictError` maps to `ConflictError`. This covers the handle being claimed by someone else between start and completion.

**`Unlink`:** a missing account returns `NotFoundInfoError`.

## 4. Discord Layer

### 4.1 Command definition (`app/bot/discordcommands/judge.go`)

```
/judge
  link    judge:<string choice, required>  handle:<string, required, max 24>
  unlink  judge:<string choice, required>
  whois   user:<USER, optional>
```

- **Choices:** built from `service_model.KnownJudgeCodes`, with name `Codeforces` and value `codeforces`.
- **Constants:** add `JudgeCommandName`, the subcommand names, and the option names.
- **Action:** `VerifyJudgeLink Action = "huron:judge:link:verify"`.
- **Registration:** append the command to `SlashCommands(...)`.
- **Startup log:** update the startup log line in `main.go`, which currently names only `/problem` and `/ac`.

### 4.2 Routing

- **`interactionrouter.go`:**
  - add `interactionCommandMap["judge"]` with the three subcommands
  - add `interactionComponentMap[VerifyJudgeLink]`
  - add `VerifyJudgeLink` to `interactiontransport.ComponentAction`
- **`app/context/interaction.go`:** add a `JudgeInteractionController` interface and a `JudgeController` field to `InteractionContext`.
- **`bot.go`:** thread a `judgeInteractionController` parameter through `RegisterHandlers`, `HandleInteractionEvent`, and `interactionHandler`, the same way the problem and AC controllers are passed. Update the callers in `app/main.go` and `workflow/setup_test.go` (`handleInteractionEvent`).
- **Custom ID:** `huron:judge:link:verify|<judgeCode>`. Add a builder and a parser in `interactiontransport`. The parser rejects unknown judge codes.

### 4.3 Controller (`app/controllers/judge/`)

```go
type InteractionController struct {
    services   *services.Services
    responder  dispatcher.InteractionResponder
    challenges *linkChallengeCache // otter: key {guildID, userID, judge}; TTL services.LinkChallengeTTL; capacity 500; injectable clock
}
func NewInteractionController(svcs *services.Services, responder dispatcher.InteractionResponder) *InteractionController
```

All replies in this stage are **private** (ephemeral), with `AllowedMentions` parse set to empty.

| Handler | Behavior |
| --- | --- |
| `HandleLinkInteraction` | 1. Require an identified member.<br>2. Parse options.<br>3. Defer privately.<br>4. Call `StartLink`.<br>5. Store the challenge under `{guild, invoker, judge}`, replacing any older one.<br>6. Edit the response with: the canonical handle, the problem link, "submit any code that fails to compile, then press Verify within 10 minutes", and a **Verify** button. |
| `HandleVerifyLinkInteraction` | 1. Parse the judge from the custom ID.<br>2. Respond with a deferred update.<br>3. Load the challenge **under the clicking member's own key**. Another member finds nothing and gets "You have no pending verification", so the original challenge is unaffected.<br>4. Call `CompleteLink`.<br>5. On success, invalidate the challenge and edit the message to "Linked Codeforces handle **X**", removing the button.<br>6. On `NotFoundInfoError`, keep the challenge and send a private follow-up explaining that no recent compilation error was found.<br>7. On expiry, invalidate the challenge and say to start again. |
| `HandleUnlinkInteraction` | Call `Unlink` and reply privately with the result. |
| `HandleWhoisInteraction` | The target is the `user` option or the invoker. Read with `GetAccounts` and reply privately with the list, or "No linked judge accounts". |

Map service errors to user messages in the controller, following `adapters.MapError` or a local equivalent. Never show raw error strings.

### 4.4 Presentation (`app/presentation/discord/judge_accounts.go`)

Pure builders for the link instructions (with the button component), the linked confirmation, and the whois list. Escape handles with the existing markdown escaping helper if one exists in `presentation/discord`; otherwise add one and test it.

### 4.5 Wiring (`app/main.go`)

- Build `services.NewJudgeRegistry(codeforces.New(codeforces.Options{BaseURL: os.Getenv("CODEFORCES_API_URL")}))`. An empty value uses the default URL.
- Use `services.NewServicesWithJudges(db, registry)`.
- Construct `judge.NewInteractionController` and pass it to `RegisterHandlers`.
- Add `CODEFORCES_API_URL` (commented, optional) to `.env.example`.
- Mock and script modes must start without network access. Constructing the client must not make requests.

## 5. Verification

### 5.1 Data unit tests (`app/data/judge_accounts_test.go`, in-memory SQLite)

| Test | Assertions |
| --- | --- |
| Schema applies twice | Calling `NewDB` on the same file path twice (use `t.TempDir()`) succeeds, which proves `IF NOT EXISTS` |
| Existing database gains table | Create a database file with **only** the pre-stage schema (copy of the old DDL as a test constant). Open it with `NewDB`. `judge_accounts` exists and the existing rows are intact |
| Save and get | Round-trip all fields. `LastSyncedAt` is nil. The user is created if missing |
| Case-insensitive handle | `GetJudgeAccountByHandle("codeforces", "TOURIST")` finds `"tourist"` |
| Handle unique per judge | A second user saving the same handle with different case gets `RowConflictError`. The first account is unchanged |
| Same handle, different judge | Allowed (seed `judge = "atcoder"` directly) |
| Relink replaces | Saving a new handle for the same user and judge replaces the handle, refreshes `verified_at`, and clears `last_synced_at` |
| Delete | Returns `true`, then `false` |
| Cascade | `ExterminateUser` removes the user's accounts |
| Name preservation | Saving with `DisplayName "Unknown"` doesn't overwrite an existing real name |

### 5.2 Service unit tests (`app/services/judge_account_service_test.go`)

Use `mock_data.MockDataStore`, `mock_services.MockJudge`, and a fixed `Now`. Write table-driven cases with an `assertFunc` each:
- **`StartLink` happy path:** the canonical handle is from the judge, `IssuedAt == Now()`, the problem is the judge's `VerificationProblem`, and **no** datastore writes happen (gomock fails on unexpected calls).
- **`StartLink` errors:**
  - unknown judge returns `ValidationError`
  - invalid handle returns `ValidationError`
  - handle not found returns `NotFoundInfoError`
  - judge unavailable returns `InternalError`
  - handle owned by another user returns `ConflictError`
  - handle already linked to self returns `ConflictError`

  Use `assert.IsType` for each.
- **`CompleteLink` success:** a matching CE at `IssuedAt + 1m` calls `SaveJudgeAccount` once with the canonical handle.
- **`CompleteLink` fails without saving:**
  - the CE is on another problem
  - the verdict is WA on the right problem
  - the CE is at `IssuedAt - 31s`
  - no submissions
- **Clock skew:** a CE at `IssuedAt - 29s` succeeds.
- **Expired:** `Now = IssuedAt + 10m + 1s` returns `ValidationError` with **no** judge call.
- **Wrong user:** a challenge for another user returns `InternalError` with no judge call.
- **Race:** `SaveJudgeAccount` returning `RowConflictError` becomes `ConflictError`.
- **`Unlink`:** a missing account returns `NotFoundInfoError`.

### 5.3 Transport and controller unit tests

- **Custom ID round trip:** builder then parser. The parser rejects a malformed ID, an unknown judge, and an ID over 100 characters.
- **`ComponentAction`:** recognizes `VerifyJudgeLink`.
- **Command definition:** `/judge` has exactly the subcommands `link`, `unlink`, `whois`. The `judge` choices equal `KnownJudgeCodes`. `handle` has max length 24. `whois.user` is type USER.
- **Challenge cache:** with an `otter` fake clock (see `problem_draft_cache_test.go`), an entry expires after the TTL, and keys for different guilds, users, or judges don't collide.

### 5.4 Workflow tests (`workflow/judge_link_test.go`)

Add a fake judge to the workflow harness: `fakeJudge` implementing `services.Judge`, with mutable handles and submissions, guarded by a mutex. Build services with `NewServicesWithJudges`. The tests drive real `discordgo.InteractionCreate` events through `env.handleInteractionEvent`.

| Test | Steps and assertions |
| --- | --- |
| `TestWorkflow_JudgeLink_VerifiesOwnershipAndStoresAccount` | 1. Member runs `/judge link codeforces tourist`.<br>2. The response is a private deferral, then an edit containing the problem URL and a Verify button.<br>3. **Intermediate:** no row in `judge_accounts`.<br>4. Press Verify with no CE: private "not found" follow-up, still no row.<br>5. Add a CE to the fake judge and press Verify again: a row exists with canonical handle `Tourist`, and the message is edited to the success text without components. |
| `TestWorkflow_JudgeLink_OtherMemberCannotCompleteChallenge` | Member A starts. Member B presses A's Verify button: B gets "no pending verification" and no row is written. A can still complete. |
| `TestWorkflow_JudgeLink_HandleClaimedByAnotherMember` | A links `tourist`. B runs `/judge link codeforces TOURIST` and gets a conflict message. B's deferred response never shows a Verify button. |
| `TestWorkflow_JudgeLink_UnknownHandle` | The fake judge returns not-found. The user sees the message and no challenge is cached: pressing a forged Verify ID says there is no pending verification. |
| `TestWorkflow_JudgeUnlinkAndWhois` | Link, then `whois` (self) lists the handle. A mentor's `whois user:@A` lists it. `unlink` removes it. `whois` then says none. A second `unlink` gives a not-linked message. |
| `TestWorkflow_JudgeLink_RelinkReplacesHandle` | Link `a`, then a complete link for `b`: one row with handle `b`. |
| Existing workflow tests | All pass with the new `HandleInteractionEvent` parameter. |

The `recordingDiscordInteractionClient` isn't goroutine-safe. That's fine in this stage because the handlers are synchronous; note it for stage 3.

### 5.5 Manual verification in a test guild (record results in the PR)

Run the bot against a **test guild** with `make run` and a real Codeforces account you control:
1. `/judge link codeforces <your handle in different casing>`. The private message shows the canonical handle and the Watermelon (4A) link.
2. Press Verify **before** submitting: you get the "not found" message.
3. Submit `int main( {` to 4A on Codeforces. Wait for the CE verdict, then press Verify: the account is linked.
4. Run `sqlite3 data/db.sqlite "select * from judge_accounts;"` and check the row.
5. From a second Discord account, run `/judge link codeforces <same handle>`: it's refused.
6. Run `/judge whois`, then `/judge unlink codeforces`, then `/judge whois`.
7. Start a link, wait more than 10 minutes, then press Verify: you're told to start again.

### 5.6 Commands to run and report

```
make generate
go fmt ./... && go vet ./...
make test
go test -race -run 'Judge' -v ./app/... ./workflow/...
```

### 5.7 Definition of done

- [ ] All tests in sections 5.1–5.4 exist and pass under `-race`.
- [ ] The manual steps in section 5.5 were performed, with results (screenshots or text) in the PR.
- [ ] No AC tables are written by any code added in this stage: `git diff main -- app/ | grep -n solved_problems` is empty.
- [ ] Existing tests are unchanged except for the mechanical controller-parameter update.
- [ ] `/judge` is registered on startup, and the startup log names it.
