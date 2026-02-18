# Huron Problem Checker Bot

A Discord bot built in **Go** for managing algorithms courses. It handles user registration, class management, problem assignments, solution tracking, and score visualization — all orchestrated through Discord commands with role-based access control.

Built with [`discordgo`](https://github.com/bwmarrin/discordgo) and backed by **SQLite**.

---

## Documentation

- [Architecture & Design](docs/ARCHITECTURE.md) — Layered architecture, diagram, DB schema, confirmation flow, parameter conventions.
- [Commands Reference](docs/COMMANDS.md) — Full list of commands by role (General, Enrolled, Mentor, Admin).

---

## Setup & Installation

### Prerequisites

| Requirement | Version |
|---|---|
| **Go** | 1.24.1+ |
| **SQLite3** | Any recent version (bundled via `go-sqlite3`, requires CGo) |
| **GCC / C Compiler** | Required by `go-sqlite3` (e.g., `gcc` on Linux, MinGW on Windows) |
| **Docker** *(optional)* | For containerized deployment |

### Environment Variables

Create a `.env` file in the project root:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token_here
DISCORD_ADMIN_ROLE_ID=123456789012345678
DISCORD_MENTOR_ROLE_ID=123456789012345679
```

| Variable | Required | Description |
|---|:---:|---|
| `DISCORD_BOT_TOKEN` | ✅ | Bot token from the [Discord Developer Portal](https://discord.com/developers/applications) |
| `DISCORD_ADMIN_ROLE_ID` | ✅ | Discord role ID for admin-level commands |
| `DISCORD_MENTOR_ROLE_ID` | ✅ | Discord role ID for mentor-level commands |

### Running Locally

```bash
# 1. Clone the repository
git clone https://github.com/JorgeIba/Huron-Problem-Checker-Bot.git
cd Huron-Problem-Checker-Bot

# 2. Install dependencies
go mod download

# 3. Generate mocks (required for tests)
make generate

# 4. Run tests
make test

# 5. Build the binary
make build
# Binary created at ./bin/huron-bot

# 6. Run the bot
make run
```

> **Note:** The bot resolves the database and schema paths relative to the executable location. When using `make run`, the binary is at `./bin/huron-bot`, so it expects `./bin/data/schema.sql`. Make sure to copy or symlink the schema file appropriately, or run from source:
>
> ```bash
> go run app/main.go
> ```

### Running with Docker

```bash
# Build and start with Docker Compose
docker compose up --build -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

The Docker setup uses a multi-stage build (Go builder → Debian slim runtime) and includes automatic backup scripts. Volumes are mounted at `./local-backup` for database backups.

