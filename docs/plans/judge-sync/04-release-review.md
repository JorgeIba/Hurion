# Stage 4: Release Review — End-to-End Coverage, Documentation, and Live Acceptance

**Overview:** [00-overview.md](00-overview.md) · **Previous:** [03-sync.md](03-sync.md)

## 0. Context for a New Contributor

Hurion is a Go Discord bot for competitive-programming classes (see `README.md`). Stages 1–3 built judge account sync:
- a `Judge` port with a Codeforces client
- verified handles in `judge_accounts`, linked with `/judge link` and **Verify**
- `/judge unlink` and `/judge whois`
- `/judge sync`, which records ACs from Codeforces without mentor approval and posts a report

This stage makes the feature releasable:
1. one long end-to-end workflow test of the whole journey
2. complete user and architecture documentation
3. operational notes
4. a scripted acceptance run in a real test guild before production

**No new behavior is added.** Fix bugs found here only if they are small and covered by a new test. Otherwise, report them and stop.

Read before starting:
- the overview and all three previous stage plans
- ADR 0005
- `docs/COMMANDS.md`, `docs/ARCHITECTURE.md`, `README.md`, `.env.example`, `README.Docker.md`, `infra/README.md`
- `workflow/judge_link_test.go` and `workflow/judge_sync_test.go` from stages 2 and 3

**Prerequisites:** stage 3 merged. Branch `feat/judge-sync-4-release` from `main`.

## 1. End-to-End Workflow Test

Create `workflow/judge_journey_test.go` with one test, `TestWorkflow_JudgeJourney`. It drives the full lifecycle through bot entry points only (`handleInteractionEvent`, `sendMessage`, `sendReaction`). Static prerequisites (class, roles, problems) may be seeded; **all judge state must be created through commands**. It asserts the intermediate state after every step:

| # | Actor | Action | Assert |
| --- | --- | --- | --- |
| 1 | Student | `/judge sync` before linking | Private "link first" message. No judge calls |
| 2 | Student | `/judge link codeforces stud` | Challenge shown. No account row |
| 3 | Student | Verify without a CE | Not-found message. No account row |
| 4 | (fake judge) | Add a CE on the verification problem | none |
| 5 | Student | Verify | Account row with canonical handle |
| 6 | Student | `/ac submit` for problem P1 | Pending proposal exists |
| 7 | (fake judge) | Add ACs for P1 and P2 and a WA for P3 | none |
| 8 | Student | `/judge sync` | P1 and P2 solved. P1's pending proposal deleted. Public report lists P1 and P2 as new and counts P3 as still unsolved. It contains no verdict details |
| 9 | Mentor | Approve P1's old proposal card | "Already resolved". Exactly one P1 row |
| 10 | Mentor | `!undo_ac @student P2` | P2 unsolved |
| 11 | Mentor | `/judge sync user:@student` after advancing the gate clock | P2 re-recorded. Report lists only P2 |
| 12 | Other student | `/judge sync user:@student` | Refused. No judge call |
| 13 | Student | `/judge unlink codeforces` | Row removed. A later sync says "link first" |
| 14 | Admin | `!exterminate @student` (confirm 👍) | After relinking in a sub-step, the account row cascades away |

Run it with `-race -count=5`.

## 2. Regression Sweep

Prove that nothing outside the feature changed behavior:
- `make test` passes.
- `cd web && pnpm check` passes (the scoreboard JSON shape must be unchanged).
- Compare the scoreboard export before and after a sync. In a test, export a class with `jsonexport`, sync, and export again: only the expected solves differ, and the schema keys are identical.
- Run `go test -cover ./app/...` on this branch and on `main`. Report the totals; coverage must not drop. List per-package coverage for:
  - `app/services`
  - `app/controllers/judge`
  - `app/providers/judges/...`
  - `app/data`

## 3. Documentation

| File | Change |
| --- | --- |
| `docs/COMMANDS.md` | A new "Judge accounts" section with every `/judge` subcommand, who may use it, its options, and examples. Explain verification step by step, cooldowns, that synced ACs need no approval, that undo is overridden by the next sync, and the supported URL forms |
| `docs/ARCHITECTURE.md` | Add a Judges provider to the layer table and diagram. Update the schema section to 8 tables with `judge_accounts` in the ER diagram. Add a "Judge sync flow" subsection mirroring overview section 7. Update the Confirmation Flow section to note that judge-synced ACs bypass mentor approval |
| `README.md` | Add a student bullet: "Link your Codeforces account and sync accepted problems automatically with `/judge sync`." |
| `.env.example`, `README.Docker.md` | Document the optional `CODEFORCES_API_URL` (testing only) |
| `docs/adr/0005-judge-account-sync.md` | Set the status to Accepted and implemented. Record any decision changed during stages 1–3 |
| `docs/plans/judge-sync/00-overview.md` | Mark the stages complete. Link the PRs |

Docs verification: every command name, option name, and limit in `COMMANDS.md` must match the code. Add a unit test in `app/bot/discordcommands` that collects `/judge` subcommand and option names and checks each appears in `docs/COMMANDS.md`. It reads the file relative to the repo root; `schema.sql` is the only other file tests may read, per `.ai/TESTING.md`, so if reviewers object, record this check as a manual one instead.

## 4. Operational Notes (add to `docs/ARCHITECTURE.md` or a new `docs/JUDGE_SYNC.md`)

- **External dependency:** `codeforces.com/api`, called anonymously, at most one request per 2.1 seconds process-wide. An outage only affects `/judge` commands.
- **Logging:** confirm and document the log lines for link start, link success, sync start, per-judge results (judge, target, new AC count, duration, error), and gate skips. Add any that are missing (small change, with a test only where a test helper already captures logs).
- **Rollback:**
  - The table is additive and unused by other features.
  - To disable the feature, revert the command registration commit, or remove `JudgeSlashCommand` from `SlashCommands` and redeploy; Discord drops the command on the next registration.
  - Data in `judge_accounts` can stay.
  - Synced ACs are ordinary `solved_problems` rows. They have no provenance column, so they can't be selectively reverted. State this explicitly.
- **Restarts:** challenges and cooldowns live in memory and are lost on restart; that is acceptable.

## 5. Live Acceptance in a Test Guild

Use a test guild, a fresh database copy, and two Discord accounts (one student and one mentor), plus one Codeforces account you control. Record each step's result as pass/fail with a screenshot or text in the PR:

1. **Startup:** start with `make run`. The log shows `/judge` registered.
2. **Linking:** student linking flow, including a wrong-problem CE (rejected) and then the right problem (accepted).
3. **Duplicate handle:** linking the same handle from the mentor's account is refused.
4. **Seed problems:** the mentor adds four problems:
   - an AC'd contest URL
   - an AC'd problemset URL
   - a gym URL (if stage 1 confirmed gym support)
   - a never-attempted problem
   - plus one non-Codeforces URL
5. **First sync:** the student runs `/judge sync`. The report numbers match exactly, and `!score` and the web scoreboard update within the exporter's debounce window.
6. **Second sync:** an immediate second sync hits the cooldown. After 5 minutes, it reports 0 new.
7. **Pending proposal:** the student proposes the never-attempted problem with `/ac submit`, then gets AC on Codeforces and syncs. The proposal card can no longer be approved, and the problem is solved once.
8. **Mentor sync:** the mentor syncs the student. The student cannot sync the mentor.
9. **Failure path:** restart with `CODEFORCES_API_URL=http://127.0.0.1:9`. A sync fails privately and nothing changes.
10. **Rate limit:** two accounts sync at the same moment. Both succeed, and the logs show spaced requests.
11. **Rollback rehearsal:** remove the command registration locally, restart, and confirm `/judge` disappears from the guild. Restore it.

## 6. Commands to Run and Report

```
make generate
go fmt ./... && go vet ./...
make test
go test -race -count=5 -run TestWorkflow_JudgeJourney -v ./workflow/
go test -cover ./app/...            # on this branch and on main
(cd web && pnpm check)
go test -tags live -run TestLive -v ./app/providers/judges/codeforces/
```

## 7. Definition of Done

- [ ] `TestWorkflow_JudgeJourney` exists, asserts every row in section 1, and passes under `-race -count=5`.
- [ ] The regression sweep (section 2) is done, with coverage numbers in the PR and no drop.
- [ ] All documentation in section 3 is updated. The docs-versus-code check passes, or is recorded as manual.
- [ ] The operational notes (section 4) are written, including the no-provenance caveat.
- [ ] Every live acceptance step (section 5) passed, with evidence in the PR.
- [ ] Bugs found are either fixed with a covering test or listed as known issues in the PR and in ADR 0005.
