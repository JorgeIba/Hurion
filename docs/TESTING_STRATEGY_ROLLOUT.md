# Testing Strategy Rollout

This document is the working reference for the workflow-testing initiative. It explains why the new suite exists, what each test layer is responsible for, and how the rollout proceeds across PRs.

This rollout is **incremental** and follows a **stop-and-wait** model:
- after each PR is opened, implementation pauses
- the next chunk starts only after review feedback is resolved and approval is given

---

## Why add a `workflow/` suite

The repo already has strong unit coverage and a substantial `integration/` suite, but those suites answer different questions:
- **Unit** tests tell us whether one component's logic is correct in isolation.
- **Integration** tests tell us whether adjacent real layers collaborate correctly.

What they do not fully cover is the confidence boundary above them:
- real user journeys
- multi-step transitions
- actor handoffs across roles
- event-driven behavior that starts at bot entrypoints

The `workflow/` suite exists to validate those journeys directly.

---

## Suite philosophy

The repo classifies tests by **confidence boundary**, not by setup mechanics.

### Unit
- Validates isolated logic and local behavior.
- Failures should point to a narrow rule, transformation, or branch.

### Integration
- Validates contracts between adjacent real application layers.
- Typical examples in this repo:
  - controller -> service -> repository -> DB
  - controller -> presentation -> dispatcher output
  - permission and contextual resolution using real services plus fake Discord collaborators
- Integration tests are **not** defined by whether they seed data.

### Workflow
- Validates multi-step user or business journeys across actions, roles, and state transitions.
- Typical examples in this repo:
  - student proposes AC -> mentor reacts -> pending state becomes accepted or rejected
  - admin triggers `!clear_class` -> confirmation is emitted -> author confirms/cancels -> state mutates or stays intact

---

## Repo rules for this rollout

- Existing tests in `integration/` stay where they are.
- No existing `integration/` tests should be moved or removed as part of this initiative.
- New workflow confidence should be added through a new top-level `workflow/` suite.
- Static prerequisites may still be seeded directly in workflow tests when helpful.
- Dynamic workflow state should be created by earlier actions in the test whenever practical.

---

## Planned PR sequence

### PR 1: Philosophy and rollout docs
- Update `.ai/TESTING.md` to define unit, integration, and workflow tests by purpose.
- Add this rollout document.
- Make the stop-and-wait process explicit.

### PR 2: Workflow harness foundation
- Create `workflow/`.
- Add the shared harness for message-driven and reaction-driven tests.
- Add one very small proof test that validates the harness shape.

### PR 3: AC workflow coverage
- Add enrolled user `!ac` proposal flow.
- Add mentor approval flow.
- Add mentor rejection flow.
- Add unauthorized or unrelated reaction coverage.

### PR 4: Dangerous command confirmation workflows
- Add `!clear_class` workflow coverage.
- Assert confirmation emission, confirm path, cancel path, and non-author reaction ignore path.

### PR 5: Broader workflow expansion
- Add the next highest-value scenarios that provide distinct confidence, likely from:
  - `!remove_problem` confirmation flow
  - contextual class resolution through channel mapping
  - role-divergent flows for student, mentor, and admin behavior
  - one read-heavy workflow if bot-entry coverage adds value beyond existing integration tests

---

## Initial workflow target scenarios

The initial workflow suite should cover these first:
- student proposes AC and the bot records a pending confirmation
- mentor approves the proposal and the student gains the AC
- mentor rejects the proposal and the student does not gain the AC
- unrelated user reaction does nothing
- `!clear_class` produces a confirmation request
- author confirmation clears class contents
- cancellation leaves class contents intact
- non-author confirmation reaction is ignored

---

## Status checklist

- [x] PR 1 defined and documented
- [ ] PR 1 implemented and reviewed
- [ ] PR 2 workflow harness created
- [ ] PR 3 AC workflows added
- [ ] PR 4 confirmation workflows added
- [ ] PR 5 broader workflow expansion added

Update this checklist as each PR is opened, reviewed, and completed so the repo always contains the current rollout state.
