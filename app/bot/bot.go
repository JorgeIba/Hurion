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
	messageSender        dispatcher.MessageSender
	interactionResponder dispatcher.InteractionResponder
	discordProvider      providers.DiscordProvider
	services             *services.Services
	testMode             bool
	whitelistUsers       map[string]struct{}
	whitelistRoles       map[string]struct{}
	guildID              string
}

type BotConfig struct {
	TestMode         bool
	WhitelistUserIDs []string
	WhitelistRoleIDs []string
	GuildID          string
}

// NewBot accepts a nil responder only when no interaction events will be handled.
func NewBot(svcs *services.Services, provider providers.DiscordProvider, sender dispatcher.MessageSender, responder dispatcher.InteractionResponder, cfg BotConfig) *Bot {
	userMap := make(map[string]struct{})
	for _, id := range cfg.WhitelistUserIDs {
		userMap[id] = struct{}{}
	}

	roleMap := make(map[string]struct{})
	for _, id := range cfg.WhitelistRoleIDs {
		roleMap[id] = struct{}{}
	}

	return &Bot{
		messageSender:        sender,
		interactionResponder: responder,
		discordProvider:      provider,
		services:             svcs,
		testMode:             cfg.TestMode,
		whitelistUsers:       userMap,
		whitelistRoles:       roleMap,
		guildID:              cfg.GuildID,
	}
}

func (bot *Bot) isWhitelisted(userID string, userRoles []string) bool {
	if _, ok := bot.whitelistUsers[userID]; ok {
		return true
	}
	for _, roleID := range userRoles {
		if _, ok := bot.whitelistRoles[roleID]; ok {
			return true
		}
	}

	return false
}

func (bot *Bot) resolveUserRoles(m *discordgo.Member, guildID, userID string) []string {
	if m != nil && len(m.Roles) > 0 {
		return m.Roles
	}
	if guildID != "" && bot.discordProvider != nil {
		if mem, err := bot.discordProvider.GetCachedMember(guildID, userID); err == nil && mem != nil {
			return mem.Roles
		}
	}
	return nil
}

func (bot *Bot) ConfigureSession(s *discordgo.Session) {
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsMessageContent
	s.StateEnabled = true
}

func (bot *Bot) RegisterHandlers(s *discordgo.Session, problemInteractionController context.ProblemInteractionController) {
	if s == nil {
		return
	}

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		bot.HandleMessageEvent(m)
	})
	s.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		bot.HandleReactionEvent(r)
	})
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.HandleInteractionEvent(s, i, problemInteractionController)
	})
}

func recoverHandlerPanic(eventName string) {
	if recovered := recover(); recovered != nil {
		logger.Log.Errorf("Panic recovered in %s handler: %v", eventName, recovered)
	}
}

func (bot *Bot) HandleMessageEvent(m *discordgo.MessageCreate) {
	defer recoverHandlerPanic("message event")

	if bot.testMode {
		roles := bot.resolveUserRoles(m.Member, m.GuildID, m.Author.ID)
		if !bot.isWhitelisted(m.Author.ID, roles) {
			return
		}
	}

	bot.messageHandler(m)
}

func (bot *Bot) messageHandler(m *discordgo.MessageCreate) {
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
		BotUserID:              bot.discordProvider.BotUserID(),
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

func (bot *Bot) HandleReactionEvent(r *discordgo.MessageReactionAdd) {
	defer recoverHandlerPanic("reaction event")

	if bot.testMode {
		roles := bot.resolveUserRoles(r.Member, r.GuildID, r.UserID)
		if !bot.isWhitelisted(r.UserID, roles) {
			return
		}
	}

	bot.reactionHandler(r)
}

func (bot *Bot) reactionHandler(r *discordgo.MessageReactionAdd) {
	if r.UserID == bot.discordProvider.BotUserID() || r.GuildID == "" {
		return
	}

	executor := controllers_common.MapMemberToUser(r.Member)
	if executor.UserID == "" {
		// Try to find the member in the bot's memory (state) to get the DisplayName via provider
		if m, err := bot.discordProvider.GetCachedMember(r.GuildID, r.UserID); err == nil {
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
		BotUserID:              bot.discordProvider.BotUserID(),
	}

	if router, exists := reactionRouterMap[r.Emoji.Name]; exists {
		// Systematic safeguard: ensures every user involved exists in the DB before logic runs
		controllers_common.SyncParticipants(context)

		router(context)
	}
}

func (bot *Bot) HandleInteractionEvent(s *discordgo.Session, event *discordgo.InteractionCreate, problemInteractionController context.ProblemInteractionController) {
	defer recoverHandlerPanic("interaction event")

	if event == nil || event.Interaction == nil {
		return
	}
	if !hasInteractionRoute(event) {
		return
	}

	if bot.testMode {
		userID := ""
		if event.Member != nil && event.Member.User != nil {
			userID = event.Member.User.ID
		}
		if userID == "" && event.User != nil {
			userID = event.User.ID
		}

		roles := bot.resolveUserRoles(event.Member, event.GuildID, userID)
		if !bot.isWhitelisted(userID, roles) {
			bot.interactionResponder.RespondPrivately(event.Interaction, "This development interaction is not enabled for your account.")
			return
		}
	}

	bot.interactionHandler(event, problemInteractionController)
}

func (bot *Bot) interactionHandler(event *discordgo.InteractionCreate, problemInteractionController context.ProblemInteractionController) {
	if event.GuildID != bot.guildID {
		bot.interactionResponder.RespondPrivately(event.Interaction, "This interaction is only available in the configured server.")
		return
	}

	c := &context.InteractionContext{
		Interaction:       event,
		Dispatcher:        bot.interactionResponder,
		ProblemController: problemInteractionController,
	}
	switch event.Type {
	case discordgo.InteractionApplicationCommand:
		routeInteractionCommand(c)
	case discordgo.InteractionModalSubmit:
		routeInteractionModalSubmit(c)
	}
}
