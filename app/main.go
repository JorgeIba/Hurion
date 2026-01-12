package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bot"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func main() {
	// Validate required environment variables
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatalf("DISCORD_BOT_TOKEN environment variable is required")
	}

	adminRoleID := os.Getenv("DISCORD_ADMIN_ROLE_ID")
	if adminRoleID == "" {
		log.Fatalf("DISCORD_ADMIN_ROLE_ID environment variable is required")
	}

	mentorRoleID := os.Getenv("DISCORD_MENTOR_ROLE_ID")
	if mentorRoleID == "" {
		log.Fatalf("DISCORD_MENTOR_ROLE_ID environment variable is required")
	}

	logger.Init()

	// Get the directory where the executable is located
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// Build paths relative to executable directory
	dbPath := filepath.Join(execDir, "data", "db.sqlite")
	schemaPath := filepath.Join(execDir, "data", "schema.sql")

	db, err := data.NewDB(
		&data.DBConfig{
			DBPath:     dbPath,
			SchemaPath: schemaPath,
			DBType:     data.Prod,
		},
	)

	if err != nil {
		log.Fatalf("Error creating the DB: %v", err)
	}

	bot, err := bot.CreateBot(token, db)
	if err != nil {
		log.Fatalf("Error creating the bot: %v", err)
	}

	if err := bot.Run(); err != nil {
		log.Fatalf("Error running the bot: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	bot.Stop()
}
