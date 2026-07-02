# Script Runner Developer Guide

The script runner allows developers to execute a sequence of bot commands defined in a JSON Lines (JSONL) file. This is highly useful for two main use cases:
1. **Mock Script Mode (Offline)**: Test bot workflows, scale limits, or image rendering completely offline without a connection to Discord.
2. **Online Script Mode**: Execute commands against real Discord IDs and roles (such as for migrations or seed scripting), bypassing verification flows by executing commands as a Mentor.

---

## Operating Modes

| Mode | Flags | Network Layer | Message Output |
|---|---|---|---|
| **Mock Script** | `--mock-discord --script app/script/test.jsonl` | Fake Mock Providers | Printed to Console (`stdout`) |
| **Online Script** | `--script commands.jsonl` | Real Discord Session | Printed to Console (`stdout`) |

*Note: In both script modes, the bot will execute the instructions sequentially, print all dispatcher outputs to the console, and immediately exit once finished.*

---

## JSONL File Schema

Each line of a script file must contain exactly one JSON object representing an instruction. There are four instruction types:

### 1. `setup_alias` (Only valid in Mock Script Mode)
Registers a friendly username string and binds it to a mock numeric Discord ID.
```json
{"type": "setup_alias", "alias": "student_maria", "user_id": "4000"}
```

### 2. `setup_roles` (Only valid in Mock Script Mode)
Configures mock roles for a user. It maps user IDs (supports registered aliases) to an array of role IDs.
```json
{"type": "setup_roles", "user_id": "mentor_pedro", "roles": ["mentor_role_id"]}
```

### 3. `setup_channel` (Only valid in Mock Script Mode)
Binds a friendly channel alias to a mock numeric Discord channel ID.
```json
{"type": "setup_channel", "alias": "basicos", "channel_id": "123456"}
```

### 4. `command`
Instructs the bot to execute a command as a specific executor.
```json
{
  "type": "command",
  "command": "!ac @student_maria 1",
  "executor_id": "mentor_pedro",
  "channel_id": "#basicos"
}
```

#### Fields:
* `command` (string, required): The raw command to execute (e.g. `!ac`, `!score`).
* `executor_id` (string, required): The Discord ID (or registered alias) of the user executing the command. **Strict Alias Enforcement**: To ensure logs are readable and prevent typos, the `executor_id` must be registered via `setup_alias` (or have an explicit `executor_name` field in the command) or the script runner will reject the instruction.
* `channel_id` (string, optional): The Discord Channel ID (supports `#channel_alias` prefix).
* `roles` (array of strings, optional): In-line roles for the executor. (Note: Using `setup_roles` once at the top of the file is the recommended way to configure static roles instead of repeating this field).

*Note: For **Online Script Mode**, the script runner reads your target Discord Server (Guild) ID from the `guild_id` parameter inside `config.yaml` to successfully resolve permissions and role memberships against the real Discord API. Under **Mock Script Mode**, if `guild_id` is empty, it automatically defaults to `"guild_1"`.*

---

## Alias Resolution

To avoid typing long numeric Discord IDs inside command strings, you can use the `@` and `#` prefixes.

When a command is executed:
1. The script runner splits the command string by spaces into separate tokens.
2. **User Mentions**: If any token starts with `@`, it extracts the user alias (e.g. `@student_maria` -> `student_maria`). If registered, it replaces the token with a standard Discord mention tag: `<@user_id>`.
3. **Channel IDs**: If any token starts with `#`, it extracts the channel alias (e.g. `#basicos` -> `basicos`). If registered, it replaces the token with the raw channel ID string (e.g. `!link_rooms advanced_go #basicos` -> `!link_rooms advanced_go 123456`).

This tokenized matching ensures that shorter prefixes (like `@bob` or `#chan`) will never incorrectly match or shadow longer aliases (like `@bobby` or `#channel`).

---

## Console Output Reference

When the bot dispatches messages during script execution, it intercepts them and logs them in a readable stdout format:

* **Simple Message**:
  `[BOT MESSAGE -> #123456]: User student_luis has already solved problem 1`
* **Embed**:
  `[BOT EMBED -> #123456]: [Scores] Maria (3 ACs), Luis (1 AC)`
* **Pagination Embed**:
  `[BOT PAGINATION -> #123456]: Pagination embed sent`
* **File Attachment**:
  `[BOT FILE -> #123456]: 1 files attached (table.png)`
* **Confirmation Prompt**:
  `[BOT CONFIRMATION -> msg:mock_msg_14 in #123456]: ⚠️ WARNING: This will delete class...`
