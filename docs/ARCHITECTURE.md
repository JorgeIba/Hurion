# Architecture

## Layers

The bot follows a clean **layered architecture** with clear separation of concerns:

| Layer | Responsibility | Key Packages |
|---|---|---|
| **Discord (Entry Point)** | Receives messages and reactions from Discord via `discordgo` session handlers. | `bot/` |
| **Context** | Data structure encapsulating the Discord event, dependencies, and executor information. | `context/` |
| **Command Router** | Parses `!commands`, resolves argument arity, and routes to the correct controller. | `bot/commandrouter.go` |
| **Reaction Router** | Handles emoji reactions for AC approvals and confirmation flows. | `bot/reactionrouter.go` |
| **Controllers** | Orchestrate permission checks, call services, and format Discord responses. Organized by role: `admin/`, `enrolled/`, `general/`, `mentor/`. | `controllers/` |
| **Adapters** | Transform data between layers (service models → presentation models). | `controllers/adapters/` |
| **Providers** | Verify Discord roles and interact with External APIs (`DiscordProvider`). | `providers/` |
| **Services** | Business logic layer. Six services: `UserService`, `EnrollmentService`, `ClassService`, `ProblemService`, `SolutionService`, `ScoreService`. | `services/` |
| **Data (Repository)** | Framework-agnostic database operations via the `DataStore` interface. | `data/` |
| **Presentation** | Rendering layer: generates score images (`image/`) and Excel exports (`excelizer/`). | `presentation/` |
| **Dispatcher** | Buffered, async message queue that handles sending all Discord responses (simple messages, embeds, reactions, files, paginated embeds, confirmation messages). | `dispatcher/` |
| **Logger** | Structured logging via `logrus`. | `logger/` |
| **Custom Errors** | Typed errors per layer: `dataerr`, `serviceerr`, `discorderr`, `controllerserr`, `presentationerr`. | `customerr/` |

---

## Diagram

```mermaid
graph TB
    subgraph BotLayer["Bot Layer"]
        Discord[Discord Client]
        Commands[Command Router]
        Reactions[Reaction Router]
    end

    subgraph ControllerLayer["Controller Layer"]
        AdminCtrl[Admin Controller]
        MentorCtrl[Mentor Controller]
        EnrolledCtrl[Enrolled Controller]
        GeneralCtrl[General Controller]
    end

    subgraph ProviderLayer["Provider Layer"]
        DiscordProvider[Discord Provider]
    end

    subgraph ServiceLayer["Service Layer"]
        UserService[User Service]
        EnrollmentService[Enrollment Service]
        ClassService[Class Service]
        ProblemService[Problem Service]
        SolutionService[Solution Service]
        ScoreService[Score Service]
    end

    subgraph DataLayer["Data Layer"]
        Queries[Database Queries]
        DB[(SQLite)]
    end

    subgraph PresentationLayer["Presentation Layer"]
        ImageRenderer[Score Image Renderer]
        Excelizer[Excel Exporter]
    end

    %% Internal connections
    Discord --> Context
    Context --> Commands
    Context --> Reactions
    Queries --> DB

    %% Flow between layers
    Commands --> ControllerLayer
    Reactions --> ControllerLayer
    ControllerLayer --> ProviderLayer
    ControllerLayer --> ServiceLayer
    ControllerLayer --> PresentationLayer
    ServiceLayer --> DataLayer
```

---

## Domain Concepts

To ensure clarity across the system, the bot utilizes the following domain concepts:
- **Enrolled User**: A user is considered "enrolled" or to have "access" to a class if they possess a Discord Role that has been linked to that class via the `class_access_roles` table (managed by `!link_roles`).
- **Active Student**: A user who has solved at least 1 problem (AC) in a specific class. This drives the `!users` and `!score` logic.

---

## Database Schema

The bot uses **SQLite** with 7 tables. 

```mermaid
erDiagram
    users ||--o{ solved_problems : "has"
    users ||--o{ pending_confirmations : "proposes"
    classes ||--o{ classrooms : "has"
    classes ||--o{ class_access_roles : "grants access via"
    classes ||--o{ problems : "contains"
    problems ||--o{ solved_problems : "is solved in"
    problems ||--o{ pending_confirmations : "awaits confirmation for"

    users {
        TEXT user_id PK
        TEXT display_name
    }
    classes {
        INTEGER id PK
        TEXT class_code UK
        TEXT name
    }
    classrooms {
        TEXT room_id PK
        TEXT class_code FK
    }
    class_access_roles {
        TEXT class_code PK, FK
        TEXT role_id PK
    }
    problems {
        INTEGER id PK
        TEXT name
        TEXT url
        INTEGER class_id FK
        TEXT difficulty
        TEXT topic
    }
    solved_problems {
        TEXT user_id PK, FK
        INTEGER problem_id PK, FK
        TIMESTAMP solved_at
    }
    pending_confirmations {
        TEXT message_id
        TEXT user_id PK, FK
        INTEGER problem_id PK, FK
    }
```

The DDL source of truth is [app/data/schema.sql](../app/data/schema.sql).

---

## Parameter Naming Conventions

| Format | Rule | Examples |
|---|---|---|
| `lowercase_snake` | Must be a single word (no spaces) | `class_code` → `basicos_1` |
| `CamelUpperCase` | Can contain multiple words (quotes not needed) | `ProblemName` → `Two Sum`, `ClassName` → `Basicos 2025` |

The `difficulty` parameter accepts: `Easy`, `Medium`, `Hard`, or `-` to skip.
The `topic` parameter accepts any single word, or `-` to skip.

---

## Confirmation Flow

Dangerous commands (`!remove_problem`, `!clear_class`, `!exterminate`) trigger a **two-step confirmation flow**:

1. The bot sends an embedded warning message with 👍 and 👎 reactions.
2. The original command author reacts:
   - **👍** → The operation is executed and the embed is updated to "EXECUTED".
   - **👎** → The operation is cancelled and the embed is updated to "CANCELLED".

For **AC proposals** (`!ac <problem_id>` by enrolled users):

1. The bot adds ✅ and ❌ reactions to the user's message and records the proposal in `pending_confirmations`.
2. A **mentor** reacts:
   - **✅** → The AC is approved and recorded in `solved_problems`. The ✅/❌ reactions are replaced by 🎈 (**Accepted**).
   - **❌** → The proposal is rejected. The ✅/❌ reactions are replaced by 🗣️ (**Rejected**).
