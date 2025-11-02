package main

import (
	"log"
	"os"

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

	db, err := data.NewDB(
		&data.DBConfig{
			DBPath:     "./data/db.sqlite",
			SchemaPath: "./data/schema.sql",
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

	bot.Run()
}
