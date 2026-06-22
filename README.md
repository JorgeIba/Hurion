<p align="center">
  <img src="docs/logo_readme.png" alt="Hurion Logo" width="200">
</p>

<h1 align="center">Hurion</h1>

<p align="center">
  <strong>The course assistant for algorithms clubs.</strong><br>
  Manage classes, verify solutions, and track progress — all without leaving Discord.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Database-SQLite-003B57?style=flat-square&logo=sqlite" alt="SQLite">
  <img src="https://img.shields.io/badge/Deployment-Docker-2496ED?style=flat-square&logo=docker" alt="Docker">
  <img src="https://img.shields.io/badge/Platform-Discord-5865F2?style=flat-square&logo=discord" alt="Discord">
</p>

---

## 🎯 What does it do?

Hurion transforms a Discord server into a full-scale classroom management system. It's built on a **stateless, role-based architecture** that maps a server's structure directly to its courses.

### 👤 For Students
*   **Submit with Ease:** Use `!ac <id>` to propose a solved problem.
*   **Real-time Leaderboards:** Check standings and class scores with beautiful image-based tables.
*   **Personal Tracking:** Keep a history of solved problems across different classes.

### 👨‍🏫 For Mentors
*   **Verification Loop:** Approve solutions with a single ✅ reaction. No more manual spreadsheets.
*   **Content Management:** Add, update, or remove problems in seconds.
*   **Class Oversight:** See who is active and who needs support with `!users`.

### ⚙️ For Admins
*   **Infrastructure Linking:** Connect Discord channels to classes and roles to access-levels.
*   **Data Portability:** Export entire class results to Excel with `!export`.
*   **System Control:** Manage classes and users with high-integrity confirmation flows.

---

## 💬 See it in action

> **Student:** `!ac 101`
>
> **Hurion:** *Reacts with ✅ and ❌*
> 
> **Mentor:** *Reacts with ✅ on the same message*
>
> **Hurion:** 🎈 **Accepted!** @Student has solved **"Two Sum"**. Their score has been updated.

---

## 📚 Documentation

Detailed guides for every part of the system:

- 🏗️ **[Architecture & Design](docs/ARCHITECTURE.md)** — Layered structure and DB schema.
- 📜 **[Commands Reference](docs/COMMANDS.md)** — Full list of commands by role.
- 🤖 **[AI Hub](.ai/CONTEXT.md)** — Rules for AI-assisted contributions.
- 🧪 **[Testing Guidelines](.ai/TESTING.md)** — Patterns for verified changes.

---

## 🚀 Quick Start

### Prerequisites
*   **Go 1.24.1+**
*   **GCC / C Compiler** (For SQLite CGo)
*   **Discord Bot Token** with *Member* and *Message Content* intents enabled.

### Local Setup
1.  **Configure:** Create a `.env` with your `DISCORD_BOT_TOKEN`.
2.  **Initialize:** Define classes in `config.yaml`.
3.  **Seed:** `make run-seed` to bootstrap the database.
4.  **Launch:** `make run`.

### Runtime Flags & Configuration Modes
The bot's behavior is controlled by a combination of CLI flags and your `config.yaml` file:

*   **Standard Mode (No flags):** The bot simply boots up and connects to Discord. It strictly uses whatever data is currently in the SQLite database.
*   **Seed Mode (CLI Flag: `--seed`):** When passed at startup, the bot performs an **additive bootstrap** against the database before connecting to Discord. It safely creates any missing classes, rooms, or roles defined in your `config.yaml`. Because it is strictly additive, it will *not* delete or overwrite existing data in the database.
*   **Test Mode (Config: `test_mode: true`):** Controlled via your `config.yaml`. When true, the bot bypasses strict Discord channel validations and only responds to users and roles explicitly listed in the config's whitelists. It also prefixes database interactions with `test_`, allowing safe local development without corrupting production data.

### Docker (Recommended)
```bash
docker compose up --build -d
```
For persistence, replication (Litestream), and Docker flag configuration details, see **[README.Docker.md](README.Docker.md)**.

---

## 🌟 Why Hurion?

The name **Hurion** is the fusion of two identities:

*   **The Huron (Ferret):** The agile mascot of the algorithms club at **ESCOM** (Escuela Superior de Cómputo). Known for its curiosity and its ability to "ferret out" solutions in complex environments.
*   **Orion:** The legendary Greek hunter and celestial guide. Just as the constellation guides travelers, Hurion guides students through their journey in competitive programming.

**Hurion** is the "Hunter of Solutions"—a centralized assistant designed to handle the heavy administrative lifting so mentors can focus on teaching and students can focus on coding.

> **🎨 Note on the Logo:** The current logo was generated with the help of AI based on our club's original mascot. If you're a designer and would like to help us create a proper, handcrafted vector version, we would love your contribution!
