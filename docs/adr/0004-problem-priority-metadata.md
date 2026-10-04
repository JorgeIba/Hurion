# ADR 0004: Optional Problem Priority and Class-Specific Presentation

* **Status**: Proposed
* **Date**: 2026-10-03

---

## 1. Context

Mentors need to record how a problem should be prioritized independently of its difficulty. A priority is optional; when it has a value, the stored value should have the same meaning across classes and interfaces. Labels, descriptions, and symbols may differ by class, so they should not become part of the data model.

Problem lists also need a stable order when priorities match. The current problem record has an auto-incremented ID, but no creation timestamp. IDs indicate insertion order in the current database, but do not expose the time a problem was added.

The current database may be recreated before this feature is used. Existing production data does not need to be migrated for this change.

## 2. Decision

### Stored contract

Add a nullable `priority` field to problems. Store one of these stable codes:

| Code | General meaning |
| --- | --- |
| `essential` | Essential for the class or track |
| `recommended` | Recommended |
| `supplementary` | Additional practice |
| `not_required` | Explicitly not required |
| `discouraged` | Should not be attempted or recommended |

An absent priority is represented by SQL `NULL` and by an absent optional value in Go. It means no priority has been assigned; it is not another priority code. Do not store `unassigned`, an empty string, or a separate `unreviewed` state for the same condition. Difficulty remains a separate optional field.

These codes describe priority categories. They do not change AC values, student scores, or leaderboard calculations. Their fixed category order is used only for problem-list sorting.

### Presentation contract

Keep each stable code separate from its displayed label, description, and symbol. Provide general English presentation defaults and allow per-class overrides. A class may use its own wording or symbols to explain the categories; those choices are configuration, not part of the universal priority contract. Presentation overrides cannot change the stored code or its ordering.

An absent priority has a general presentation meaning of “No priority assigned.” A class may choose different wording or a symbol for that state, without introducing another stored priority value.

### Problem creation time and ordering

Add `created_at` to each problem. SQLite generates it at insertion time using a UTC `CURRENT_TIMESTAMP` default. The application does not supply or change it during an update. For imported records, this is the time they are imported into Hurion; historical source-file dates are not inferred.

When a problem list is sorted by priority, preserve the existing topic grouping and sort problems within each group in this order: `essential`, `recommended`, `supplementary`, `not_required`, no priority, `discouraged`. Use `created_at` ascending (oldest first) to break ties, then problem ID ascending for timestamps that match. Other views retain their current ordering unless their presentation phase explicitly adopts the priority sort.

### Persistence and rollout

Persist and read both fields through the existing data layer and carry them through service, presentation, and exported scoreboard models. The schema is rebuilt through the existing clean-database workflow before use; do not add a historical migration for this release.

## 3. Alternatives Considered

| Alternative | Reason not selected |
| --- | --- |
| Store translated labels or emojis in `priority` | Couples persisted data to language and one class's convention; changing presentation would require rewriting records. |
| Store `unassigned` as a priority value | Duplicates the meaning of an absent optional field and requires special-case handling throughout the domain. |
| Use difficulty as priority | Difficulty answers how hard a problem is; priority answers whether or how strongly it is recommended. |
| Use problem ID alone as the tie-breaker | ID gives deterministic insertion order but cannot expose or communicate when the problem was added. |

## 4. Consequences

* Priority codes remain stable as labels and class-specific symbols evolve.
* An absent priority remains distinguishable from `not_required` and `discouraged`.
* `created_at` provides a human-readable time for deterministic tie-breaking; IDs remain a final deterministic fallback.
* Class presentation configuration must be available to both Discord outputs and the web/export pipeline.
* Existing database contents are intentionally outside this rollout because the database will be recreated before the feature is enabled.

## 5. Incremental Implementation Plan

1. **Contract and models:** review the canonical priority values, optional representation, and model fields for priority and creation time. Stop here for review and approval before implementation proceeds.
2. **Persistence and commands:** add SQLite fields and data access, generate `created_at` in SQLite, carry priority through service mappings, and design how create/edit commands set or clear it.
3. **Presentation and sorting:** display priority in problem lists, score views, web, images, and Excel exports; add priority sorting with timestamp and ID tie-breaks.
4. **Class overrides:** add general display defaults and optional class-specific label, explanation, and symbol overrides; configure the advanced class.

Keep the stages independently reviewable. Do not combine the stages into a broad refactor.
