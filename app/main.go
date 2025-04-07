package main

import (
	"log"
	"os"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bot"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatalf("There is not a token")
	}

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
