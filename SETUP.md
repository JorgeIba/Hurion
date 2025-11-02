# Huron Problem Checker Bot - Setup Guide

## Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose (for containerized deployment)
- A Discord Bot Token
- Discord Server with appropriate roles configured

## Configuration

### 1. Environment Variables Setup

The bot requires the following environment variables to be set:

#### Required Variables:

| Variable | Description | How to Obtain |
|----------|-------------|---------------|
| `DISCORD_BOT_TOKEN` | Your Discord bot token | Get from [Discord Developer Portal](https://discord.com/developers/applications) |
| `DISCORD_ADMIN_ROLE_ID` | Role ID for administrators | Enable Developer Mode, right-click role, "Copy ID" |
| `DISCORD_MENTOR_ROLE_ID` | Role ID for mentors | Enable Developer Mode, right-click role, "Copy ID" |

#### Steps to Configure:

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit the `.env` file:**
   ```bash
   nano .env  # or use your preferred editor
   ```

3. **Fill in all required values:**
   ```env
   DISCORD_BOT_TOKEN=your_actual_bot_token_here
   DISCORD_ADMIN_ROLE_ID=12345678910111213
   DISCORD_MENTOR_ROLE_ID=12345678910111213
   ```

### 2. How to Get Discord IDs

#### Enable Developer Mode:
1. Open Discord
2. Go to User Settings (gear icon)
3. Navigate to Advanced
4. Enable "Developer Mode"

#### Get Role IDs:
1. Go to Server Settings > Roles
2. Right-click on the desired role
3. Select "Copy ID"

## Running the Bot

### Option 1: Docker (Recommended)

```bash
# Build and run
docker compose up --build

# Run in background
docker compose up -d

# View logs
docker compose logs -f

# Stop the bot
docker compose down
```

### Option 2: Local Development

```bash
# Install dependencies
go mod download

# Run the bot
go run app/main.go
```

### Option 3: Build Binary

```bash
# Build
go build -o huron-bot app/main.go

# Run
./huron-bot
```

## Role Permissions

The bot has three permission levels:

### 👤 All Users (General)
- `!bind <user_name>` - Bind Discord account to username
- `!register <class_code>` - Register in a class
- `!list <class_code>` - View problems
- `!score [class_code] [user_name]` - View scores
- `!classes` - List all classes
- `!me` - Show your username

### 📚 Enrolled Users
All general commands plus:
- `!ac <problem_id>` - Mark problem as solved (requires mentor approval)
- `!undo_ac <problem_id>` - Undo solved problem
- `!deregister <class_code>` - Leave a class
- `!my_classes` - View your classes

### 👨‍🏫 Mentors
All enrolled commands plus:
- `!add_problem <class_code> <url> [difficulty] [topic] <ProblemName>` - Add problem
- `!update_problem <problem_id> <class_code> <url> [difficulty] [topic] <ProblemName>` - Update problem
- `!remove_problem <problem_id>` - Remove problem
- `!users <class_code>` - List class users
- `!classes_of <user_name>` - View user's classes
- `!register <user_name> <class_code>` - Register another user
- `!deregister <user_name> <class_code>` - Deregister another user
- Approve/reject problem solutions with ✅/❌ reactions

### ⚙️ Administrators
All mentor commands plus:
- `!create_class <class_code> <ClassName>` - Create new class
- `!clear_class <class_code>` - Clear all class data
- `!exterminate <user_id>` - Remove user from system
- `!export <class_code>` - Export class data to Excel

## Database

The bot uses SQLite for data persistence:
- Location: `./app/data/db.sqlite`
- Schema: `./app/data/schema.sql`
- To reset: Delete `db.sqlite` and restart the bot

## Troubleshooting

### Bot doesn't start:
- Verify all environment variables are set in `.env`
- Check that the bot token is valid
- Ensure role IDs are correct (18-19 digit numbers)

### Permission errors:
- Verify the user has the correct Discord role
- Check that role IDs in `.env` match your server roles
- Ensure the bot has appropriate permissions in Discord

### Database errors:
- Check that `./app/data/` directory exists
- Verify write permissions for the data directory
- Try deleting `db.sqlite` to reset the database

## Development

### Running Tests:
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./app/validators/...

# Run with verbose output
go test ./... -v
```

### Project Structure:
```
app/
├── bot/              # Discord bot setup and command routing
├── controllers/      # Command handlers by permission level
├── services/         # Business logic layer
├── data/             # Database operations
├── validators/       # Permission and validation logic
├── customerr/        # Custom error types
└── main.go          # Entry point
```

## Contributing

When adding new features that require role-based permissions:
1. Use `validator.IsAdmin()` or `validator.IsMentor()` in controllers
2. Update the help command in `app/bot/helpCommand.go`
3. Add route in `app/bot/commandrouter.go`
4. Follow the layer architecture (Controller → Service → Data)

## Support

For issues or questions:
- Check existing issues on GitHub
- Review Discord bot permissions
- Verify environment variable configuration
