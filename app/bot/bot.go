package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	botToken string
	db       *data.DB
}

func CreateBot(botToken string, db *data.DB) (*Bot, error) {
	return &Bot{botToken, db}, nil
}

func (bot *Bot) Run() {
	dg, err := discordgo.New("Bot " + bot.botToken)
	if err != nil {
		log.Fatalln("Error creating the session,", err)
	}

	dg.AddHandler(messageHandler)
	dg.AddHandler(reactionHandler)

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
	if m.Author.Bot {
		return
	}

	args := strings.Split(m.Content, " ")
	if len(args) < 1 {
		return
	}

	cmd := args[0]
	if handler, exists := commandMap[cmd]; exists {
		handler(s, m, args)
	}
}

func reactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		fmt.Println("Something happened when trying to get the message:", err)
		return
	}

	if message.Author.ID != s.State.User.ID {
		return
	}

	// TODO: check here If messageID corresponds to a pending reaction
}
