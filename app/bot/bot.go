package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators"
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

	dg.AddHandler(bot.messageHandler)
	dg.AddHandler(bot.reactionHandler)

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

func (bot *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	args := strings.Split(m.Content, " ")
	if len(args) < 1 {
		return
	}

	context := &context.ContextMessage{
		Session:           s,
		Message:           m,
		DatabaseValidator: validators.NewValidatorDB(bot.db),
		DiscordValidator:  validators.NewValidatorDiscord(s),
		Datastore:         bot.db,
		Args:              args,
	}
	context.Services, _ = services.NewServices(context.Datastore, context.DatabaseValidator)

	cmd := args[0]
	fmt.Println(cmd)
	if handler, exists := commandMap[cmd]; exists {
		handler(context)
	}
}

func (bot *Bot) reactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
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
