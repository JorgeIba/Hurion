package bot

import (
	"log"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	botToken        string
	db              *data.DB
	dispatcher      *dispatcher.Dispatcher
	discordProvider providers.DiscordProvider
	services        *services.Services
	session         *discordgo.Session
}

func CreateBot(botToken string, db *data.DB) (*Bot, error) {
	return &Bot{
		botToken:        botToken,
		db:              db,
		dispatcher:      nil,
		discordProvider: nil,
		services:        nil,
	}, nil
}

func (bot *Bot) Run() error {
	dg, err := discordgo.New("Bot " + bot.botToken)
	if err != nil {
		log.Fatalln("Error creating the session,", err)
		return err
	}

	const bufferSize = 5
	bot.dispatcher = dispatcher.NewDispatcher(dg, bufferSize)
	bot.dispatcher.Start()

	// Initialize provider and services
	bot.discordProvider = providers.NewProviderDiscord(dg)
	bot.services = services.NewServices(bot.db)

	dg.AddHandler(bot.safeMessageHandler)
	dg.AddHandler(bot.safeReactionHandler)

	if err = dg.Open(); err != nil {
		log.Fatalln("Error opening connection,", err)
		return err
	}

	bot.session = dg
	logger.Log.Info("Bot is running, CTRL+C to exit.")
	return nil
}

func (bot *Bot) Stop() {
	logger.Log.Info("Shutting down the bot...")

	if bot.session != nil {
		bot.session.Close()
	}

	if bot.dispatcher != nil {
		bot.dispatcher.Stop()
	}

	if bot.db != nil {
		bot.db.CloseDB()
	}

	logger.Log.Info("Bot shut down successfully.")
}

func (bot *Bot) safeMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("Panic recovered in messageHandler: %v", r)
		}
	}()
	bot.messageHandler(s, m)
}

func (bot *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) < 1 {
		return
	}

	context := &context.ContextMessage{
		Session:          s,
		Message:          m,
		DiscordValidator: bot.discordProvider,
		DiscordFetcher:   bot.discordProvider,
		Args:             args,
		Services:         bot.services,
		Dispatcher:       bot.dispatcher,
		ExecutorUserID:   m.Author.ID, // Who is executing
		ExecutorGuildID:  m.GuildID,   // Where it's executed
	}

	cmd := args[0]
	if len(cmd) > 0 && cmd[0] == '!' {
		logger.Log.Info(cmd)
	}
	if router, exists := commandMap[cmd]; exists {
		router(context)
	}
}

func (bot *Bot) safeReactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Log.Errorf("Panic recovered in reactionHandler: %v", rec)
		}
	}()
	bot.reactionHandler(s, r)
}

func (bot *Bot) reactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}

	context := &context.ContextMessage{
		Session:          s,
		MessageReaction:  r,
		DiscordValidator: bot.discordProvider,
		DiscordFetcher:   bot.discordProvider,
		Services:         bot.services,
		Dispatcher:       bot.dispatcher,
		ExecutorUserID:   r.UserID,  // Who is executing
		ExecutorGuildID:  r.GuildID, // Where it's executed
	}

	if router, exists := reactionRouterMap[r.Emoji.Name]; exists {
		router(context)
	}
}
