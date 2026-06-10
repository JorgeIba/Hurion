package bot

import (
	"strings"

	"github.com/JorgeIba/Hurion/app/context"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	messageSender   dispatcher.MessageSender
	discordProvider providers.DiscordProvider
	services        *services.Services
}

func NewBot(svcs *services.Services, provider providers.DiscordProvider, sender dispatcher.MessageSender) *Bot {
	return &Bot{
		messageSender:   sender,
		discordProvider: provider,
		services:        svcs,
	}
}

func (bot *Bot) ConfigureSession(s *discordgo.Session) {
	s.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuildMembers
	s.StateEnabled = true
}

func (bot *Bot) RegisterHandlers(s *discordgo.Session) {
	s.AddHandler(bot.HandleMessageEvent)
	s.AddHandler(bot.HandleReactionEvent)
}

func (bot *Bot) HandleMessageEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("Panic recovered in messageHandler: %v", r)
		}
	}()
	bot.messageHandler(s, m)
}

func (bot *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.GuildID == "" {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) < 1 {
		return
	}

	executor := controllers_common.MapMemberToUser(m.Member)
	if executor.UserID == "" {
		executor = controllers_common.MapDiscordUserToUser(m.Author)
	}

	context := &context.ContextMessage{
		Message:                m,
		DiscordValidator:       bot.discordProvider,
		DiscordFetcher:         bot.discordProvider,
		DiscordMessageModifier: bot.discordProvider,
		Args:                   args,
		Services:               bot.services,
		Dispatcher:             bot.messageSender,
		Executor:               executor,  // Who is executing
		ExecutorGuildID:        m.GuildID, // Where it's executed
		BotUserID:              botUserIDFromSession(s),
	}

	cmd := args[0]
	if len(cmd) > 0 && cmd[0] == '!' {
		logger.Log.Info(cmd)
	}

	if router, exists := commandMap[cmd]; exists {
		// Systematic safeguard: ensures every user mentioned exists in the DB before logic runs
		controllers_common.SyncParticipants(context)
		router(context)
	}
}

func (bot *Bot) HandleReactionEvent(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Log.Errorf("Panic recovered in reactionHandler: %v", rec)
		}
	}()
	bot.reactionHandler(s, r)
}

func (bot *Bot) reactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID || r.GuildID == "" {
		return
	}

	executor := controllers_common.MapMemberToUser(r.Member)
	if executor.UserID == "" {
		// Try to find the member in the bot's memory (state) to get the DisplayName
		if m, err := s.State.Member(r.GuildID, r.UserID); err == nil {
			executor = controllers_common.MapMemberToUser(m)
		} else {
			// Failsafe: if not in memory, at least keep the ID and set a fallback name
			executor.UserID = r.UserID
			executor.DisplayName = "Unknown"
		}
	}

	context := &context.ContextMessage{
		MessageReaction:        r,
		DiscordValidator:       bot.discordProvider,
		DiscordFetcher:         bot.discordProvider,
		DiscordMessageModifier: bot.discordProvider,
		Services:               bot.services,
		Dispatcher:             bot.messageSender,
		Executor:               executor,  // Who is executing
		ExecutorGuildID:        r.GuildID, // Where it's executed
		BotUserID:              botUserIDFromSession(s),
	}

	if router, exists := reactionRouterMap[r.Emoji.Name]; exists {
		// Systematic safeguard: ensures every user involved exists in the DB before logic runs
		controllers_common.SyncParticipants(context)

		router(context)
	}
}

func botUserIDFromSession(s *discordgo.Session) string {
	if s != nil && s.State != nil && s.State.User != nil {
		return s.State.User.ID
	}

	return ""
}
