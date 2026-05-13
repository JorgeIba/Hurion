# Commands Reference

The Huron bot uses a **stateless, contextual engine**. Many commands will automatically infer the `class_code` based on the Discord channel where they are executed.


---

## 🌐 General Commands

Available to all users.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!help` | `!help [command]` | Show info for all commands or details for a specific command | No |
| `!list` | `!list [class_code]` | Show problems for a class. Inferred from channel if `class_code` is missing. | No |
| `!score` | `!score` | Show scores for **all classes** as image tables. | No |
| `!score` | `!score <class_code>` | Show scores for a **specific class**. | No |
| `!score` | `!score <class_code> <@user>` | Show a **specific user's** ranking in a class. | No |
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
| `!undo_ac` | `!undo_ac <@user> <problem_id>` | Remove a solved problem from another user's list. | No |
| `!add_problem` | `!add_problem <class_code> <url> [difficulty] [topic] <Name>` | Add a new problem to a class. | No |
| `!update_problem` | `!update_problem <id> <class> <url> [diff] [topic] <Name>` | Update a problem's details (preserves ID). | No |
| `!remove_problem` | `!remove_problem <problem_id>` | Remove a problem and its ACs. | Yes 👍/👎 |

---

## ⚙️ Admin Commands

Available to users with the Admin Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!create_class` | `!create_class <class_code> <ClassName>` | Create a new class. | No |
| `!link_roles` | `!link_roles <class_code> <@role1> ...` | Link Discord roles to a class. | No |
| `!link_rooms` | `!link_rooms <class_code> <#channel1> ...` | Link Discord channels as classrooms. | No |
| `!clear_class` | `!clear_class <class_code>` | Wipe all class data (problems/scores). | Yes 👍/👎 |
| `!exterminate` | `!exterminate <@user>` | Permanently delete a user and all their data. | Yes 👍/👎 |
| `!export` | `!export <class_code>` | Export class data to an Excel file. | No |

---

> ℹ️ **Note on Parameters**: `CamelUpperCase` parameters (like Problem Name) can contain spaces. `lowercase_snake` parameters (like class_code) must be a single word.
