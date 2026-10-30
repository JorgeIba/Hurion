# Commands Reference

Hurion uses a **stateless, contextual engine**. Many commands will automatically infer the `class_code` based on the Discord channel where they are executed.


---

## 🌐 General Commands

Available to all users.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!help` | `!help [command]` | Show info for all commands or details for a specific command | No |
| `!list` | `!list [class_code]` | Show problems for a class. Inferred from channel if `class_code` is missing. | No |
| `!scores` | `!scores` | Show scores for **all classes** as image tables. | No |
| `!score` | `!score [class_code]` | Show scores for a **specific class**. Inferred from channel if `class_code` is missing. | No |
| `!score_of` | `!score_of <@user> [class_code]` | Show a user's score in a class. Inferred from channel if `class_code` is missing. | No |
| `!users` | `!users [class_code]` | List active users. Inferred from channel if `class_code` is missing. | No |
| `!classes` | `!classes` | List all available classes. | No |
| `!classes_of` | `!classes_of <@user>` | Show all classes a specific user has access to. | No |
| `!get_roles` | `!get_roles [class_code]` | Show linked Discord roles. Inferred from channel if missing. | No |
| `!get_channels` | `!get_channels [class_code]` | Show linked Discord channels. Inferred from channel if missing. | No |

---

## 👤 Enrolled User Commands

Available to users with a valid class role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!ac` | `!ac <problem_id>` | Propose a problem as solved. | Mentor Reaction |
| `/ac submit` | `/ac submit id:<problem_id> [screenshot:<image>]` | Propose a solved problem with optional screenshot evidence. Search the ID field by ID or name, then select a suggestion or enter a numeric ID. | Mentor Buttons |
| `/ac remove` | `/ac remove id:<problem_id> [user:<user>]` | Remove your recorded AC. Removing another user’s AC requires permission to manage the problem’s class (its configured mentor role or admin). Does not cancel pending proposals. | None |
| `!undo_ac` | `!undo_ac <problem_id>` | Remove a problem from your own solved list. | No |
| `!my_classes` | `!my_classes` | Show all classes you currently have access to. | No |

---

## 👨‍🏫 Mentor Commands

Available to users with the configured Mentor or Admin Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!ac` | `!ac <@user> <problem_id>` | Directly mark a problem as solved for a user. | No |
| `/problem add` | `/problem add [class:<class_code>]` | Open a form to create a problem. The class defaults to the one linked to the current channel. | Submit form |
| `/problem edit` | `/problem edit problem:<problem_id> [class:<class_code>]` | Open a form to replace a problem's fields. The class defaults to its current class; its ID and recorded ACs are preserved. | Submit form |
| `/problem remove` | `/problem remove problem:<problem_id>` | Preview a problem deletion, then confirm to permanently delete it and its recorded ACs. Requires permission to manage its class. Confirmations expire after five minutes and belong to the requesting member. | Delete / Cancel |
| `/problem add_batch` | `/problem add_batch` | Open a form to read a problem batch from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `/problem edit_batch` | `/problem edit_batch` | Open a form to update supplied fields of existing problems from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `/ac add_batch` | `/ac add_batch` | Open a form to read explicit student ACs from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `!undo_ac` | `!undo_ac <@user> <problem_id>` | Remove a solved problem from another user's list. | No |
| `!add_problem` | `!add_problem <class_code> <url> <difficulty or -> <topic or -> <priority or -> <Name>` | Add a new problem to a class. | No |
| `!update_problem` | `!update_problem <problem_id> <class_code> <url> <difficulty or -> <topic or -> <priority or -> <Name>` | Update a problem's details (preserves ID). | No |
| `!remove_problem` | `!remove_problem <problem_id>` | Remove a problem and its ACs. | Yes 👍/👎 |

---

For JSON formats, examples, limits, confirmation behavior, and failure handling, see the [Mentor JSON Batch Guide](MENTOR_BATCHES.md).

### Problem autocomplete

The problem fields in `/ac submit`, `/ac remove` and `/problem edit`/`/problem remove` suggest problems from the class linked to the current channel. Type an ID prefix or part of a name; suggestions are ordered by ID and limited to 25 matches. Selecting `5: Tree Sum` supplies only ID `5`. If the channel has no linked class, suggestions are empty; a numeric ID can still be entered manually. The optional `class` field in `/problem edit` selects the destination class and does not change the source of problem suggestions.

Suggestions read the current class catalog without a cache. Command execution keeps its existing permission and validation rules.

## ⚙️ Admin Commands

Available to users with the Admin Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!create_class` | `!create_class <class_code> <ClassName>` | Create a new class. | No |
| `!link_roles` | `!link_roles <class_code> <@role1> ...` | Link Discord roles to a class. | No |
| `!link_rooms` | `!link_rooms <class_code> <#channel1> ...` | Link Discord channels as classrooms. | No |
| `!clear_class` | `!clear_class [class_code]` | Reset class contents while keeping the class, linked rooms, and linked roles. | Yes 👍/👎 |
| `!exterminate_class` | `!exterminate_class <class_code>` | Permanently delete a class and all its linked data/settings. | Yes 👍/👎 |
| `!exterminate` | `!exterminate <@user>` | Permanently delete a user and all their data. | Yes 👍/👎 |
| `!export` | `!export <class_code>` | Export class data to an Excel file. | No |

---

> ℹ️ **Note on Parameters**: 
> - `CamelUpperCase` parameters (like Problem Name) can contain spaces. 
> - `lowercase_snake` parameters (like class_code) must be a single word. 
> - **Optional Positional Parameters**: Arguments like `difficulty`, `topic`, and `priority` cannot be entirely omitted. Use a dash (`-`) to skip one (e.g., `!add_problem basicos_1 http://link.com - - - My Problem`).
> - **Priority**: Priority is a positional argument, like difficulty and topic. Use `-` to leave it unset. Valid codes are `essential`, `recommended`, `supplementary`, `not_required`, and `discouraged` (case-insensitive).
