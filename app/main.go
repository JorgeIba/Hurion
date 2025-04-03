package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalln("Error creating the session,", err)
	}

	dg.AddHandler(messageHandler)

	if err = dg.Open(); err != nil {
		log.Fatalln("Error opening connection,", err)
	}
	defer dg.Close()

	fmt.Println("Bot is running, CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	fmt.Println("Disconnecting bot...")
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Avoid self-checking
	if m.Author.Bot {
		return
	}

	args := strings.Split(m.Content, " ")
	if len(args) < 1 {
		return
	}

	switch args[0] {
	case "!AC":
		s.ChannelMessageSend(m.ChannelID, "I'm listening bro")
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		s.MessageReactionAdd(m.ChannelID, m.ID, "❌")
	default:
		return
	}
}
