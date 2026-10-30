package bot

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type loggingProvider struct{ providers.DiscordProvider }

func (loggingProvider) BotUserID() string { return "bot" }

func TestUnknownPrefixDoesNotLogInvocation(t *testing.T) {
	logger.Init()
	previousHooks := logger.Log.ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logger.Log.ReplaceHooks(previousHooks) })
	hook := logrustest.NewLocal(logger.Log)
	bot := NewBot(nil, loggingProvider{}, nil, nil, BotConfig{GuildID: "guild"})
	bot.HandleMessageEvent(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "unknown-message", GuildID: "guild", Content: "!hola", Author: &discordgo.User{ID: "actor"},
	}})
	assert.Empty(t, hook.AllEntries())
}

func TestSlashInvocationLogsOnlyRecognizedSubcommand(t *testing.T) {
	logger.Init()
	previousHooks := logger.Log.ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logger.Log.ReplaceHooks(previousHooks) })
	hook := logrustest.NewLocal(logger.Log)
	for _, test := range []struct {
		name  string
		count int
	}{{"submit", 1}, {"hola", 0}} {
		t.Run(test.name, func(t *testing.T) {
			hook.Reset()
			probe := &autocompleteCommandProbe{}
			event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "logging-interaction", Type: discordgo.InteractionApplicationCommand, GuildID: "guild",
				Member: &discordgo.Member{User: &discordgo.User{ID: "actor"}},
				Data: discordgo.ApplicationCommandInteractionData{Name: "ac", Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: test.name, Type: discordgo.ApplicationCommandOptionSubCommand},
				}},
			}}
			routeInteractionCommand(&context.InteractionContext{Interaction: event, ACController: probe, Dispatcher: &autocompleteResponder{}})
			assert.Len(t, hook.AllEntries(), test.count)
			assert.Equal(t, test.count, probe.calls)
			if test.count > 0 {
				if entry := hook.LastEntry(); assert.NotNil(t, entry) {
					assert.Equal(t, "Command invoked: /ac submit", entry.Message)
				}
			}
		})
	}
}
