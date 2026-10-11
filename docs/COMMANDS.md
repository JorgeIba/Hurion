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
| `!undo_ac` | `!undo_ac <problem_id>` | Remove a problem from your own solved list. | No |
| `!my_classes` | `!my_classes` | Show all classes you currently have access to. | No |

---

## 👨‍🏫 Mentor Commands

Available to users with the Mentor Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!ac` | `!ac <@user> <problem_id>` | Directly mark a problem as solved for a user. | No |
| `/problem add_batch` | `/problem add_batch` | Open a form to read a problem batch from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `/problem edit_batch` | `/problem edit_batch` | Open a form to update supplied fields of existing problems from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `/ac add_batch` | `/ac add_batch` | Open a form to read explicit student ACs from JSON. Validates the batch and reports issues; requires Confirm before applying it. | Confirm / Cancel |
| `!undo_ac` | `!undo_ac <@user> <problem_id>` | Remove a solved problem from another user's list. | No |
| `!add_problem` | `!add_problem <class_code> <url> <difficulty or -> <topic or -> <priority or -> <Name>` | Add a new problem to a class. | No |
| `!update_problem` | `!update_problem <problem_id> <class_code> <url> <difficulty or -> <topic or -> <priority or -> <Name>` | Update a problem's details (preserves ID). | No |
| `!remove_problem` | `!remove_problem <problem_id>` | Remove a problem and its ACs. | Yes 👍/👎 |

---

Batch forms are available to mentors and admins. Paste one JSON array (up to Discord's 4,000-character form limit) or upload one `.json` file (up to 5 MiB), The file takes precedence when both sources are supplied. Problem objects use `class`, `name`, `url`, and optional `difficulty`, `topic`, and `priority`; Problem edit objects require a positive integer `problem_id` and at least one editable field. Only supplied fields are updated; omitted fields retain their current values. Explicit `null` clears `difficulty`, `topic`, or `priority`; `class`, `name`, and `url` must contain valid nonempty values when supplied. Problem IDs and recorded ACs are preserved. For example, `[{"problem_id":42,"difficulty":"hard","topic":null}]` changes only the difficulty and clears the topic. Repeated target IDs are applied in input order. AC objects use a string `user_id` and integer `problem_id`. Format errors identify the entry or JSON syntax location. Domain validation checks problem destinations and student access before reporting a valid batch. Already recorded ACs remain valid and are counted in the summary. Valid batches are retained for up to five minutes in separate in-process caches of at most 50 validated batches per command. The private preview includes Confirm and Cancel; only the submitting actor in the same server with current management permissions can use them. Confirm consumes the batch once and revalidates it before applying changes. ACs already recorded are skipped and counted; created and updated problems are reported with their IDs. Expiration, capacity eviction, or restarting the bot makes a pending batch unavailable and requires a new submission. Validation failures apply no changes. Storage failures stop subsequent operations without rolling back earlier writes; the result reports acknowledged progress and warns that the failed operation may also have been applied. A consumed batch cannot be retried, including when delivering its result fails.

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
