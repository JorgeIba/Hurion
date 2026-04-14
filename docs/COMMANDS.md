# Commands Reference

Use `!help` in Discord to see all commands, or `!help <command>` for details on a specific one.

---

## 🌐 General Commands

Available to all users.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!help` | `!help [command]` | Show info for all commands or details for a specific command | No |
| `!register` | `!register <class_code>` | Register yourself in a class | No |
| `!list` | `!list <class_code>` | Show the list of problems available for a class (paginated embed) | No |
| `!score` | `!score` | Show scores for **all classes** as image tables | No |
| `!score` | `!score <class_code>` | Show scores for a **specific class** as an image table | No |
| `!score` | `!score <class_code> <@user>` | Show a **specific user's** neighborhood ranking in a class | No |
| `!users` | `!users <class_code>` | List all users registered in a class | No |
| `!classes` | `!classes` | List all available classes | No |
| `!classes_of` | `!classes_of <@user>` | Show all classes a specific user is enrolled in | No |

---

## 👤 Enrolled User Commands

Available to users registered in at least one class.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!ac` | `!ac <problem_id>` | Propose a problem as solved (requires mentor approval via ✅/❌) | Mentor Reaction |
| `!undo_ac` | `!undo_ac <problem_id>` | Remove a problem from your own solved list | No |
| `!deregister` | `!deregister <class_code>` | Remove yourself from a class (deletes all your ACs and pending confirmations in that class) | Yes 👍/👎 |
| `!my_classes` | `!my_classes` | Show all classes you're currently enrolled in | No |

---

## 👨‍🏫 Mentor Commands

Available to users with the Mentor Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!register` | `!register <@user> <class_code>` | Register another user in a class | No |
| `!deregister` | `!deregister <@user> <class_code>` | Remove another user from a class | Yes 👍/👎 |
| `!ac` | `!ac <@user> <problem_id>` | Directly mark a problem as solved for a user (no approval needed) | No |
| `!undo_ac` | `!undo_ac <@user> <problem_id>` | Remove a solved problem from another user's list | No |
| `!add_problem` | `!add_problem <class_code> <url> [difficulty] [topic] <ProblemName>` | Add a new problem to a class | No |
| `!remove_problem` | `!remove_problem <problem_id>` | Remove a problem, all associated ACs and pending confirmations | Yes 👍/👎 |
| `!update_problem` | `!update_problem <problem_id> <class_code> <url> [difficulty] [topic] <ProblemName>` | Update a problem's details while preserving its ID | No |

---

## ⚙️ Admin Commands

Available to users with the Admin Discord role.

| Command | Usage | Description | Confirmation |
|---|---|---|:---:|
| `!create_class` | `!create_class <class_code> <ClassName>` | Create a new class with a code and display name | No |
| `!clear_class` | `!clear_class <class_code>` | Remove all enrollments, problems, and scores from a class | Yes 👍/👎 |
| `!exterminate` | `!exterminate <@user>` | Permanently delete a user and all their data from the system | Yes 👍/👎 |
| `!export` | `!export <class_code>` | Export class data (users, problems, scores) to an Excel file | No |
