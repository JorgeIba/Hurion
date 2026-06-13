package workflow

import (
	"errors"
	"strings"
	"sync"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBotUserID = "bot_1"
	testGuildID   = "guild_1"
	testChannelID = "channel_1"
)

type workflowEnv struct {
	Bot        *appbot.Bot
	DB         *data.DB
	Services   *services.Services
	Discord    *stubDiscordProvider
	Dispatcher *fakeSender
	Session    *discordgo.Session
}

func setupWorkflowEnv(t *testing.T, configs ...appbot.BotConfig) *workflowEnv {
	t.Helper()

	var cfg appbot.BotConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	logger.Init()

	db, err := data.NewDB(&data.DBConfig{
		DBPath: ":memory:",
		DBType: data.Test,
	})
	require.NoError(t, err, "failed to initialize workflow test DB")
	t.Cleanup(db.CloseDB)

	svcs := services.NewServices(db)
	discord := newStubDiscordProvider()
	sender := &fakeSender{}
	session := &discordgo.Session{
		State: &discordgo.State{
			Ready: discordgo.Ready{
				User: &discordgo.User{ID: testBotUserID, Username: "Workflow Bot"},
			},
		},
	}

	return &workflowEnv{
		Bot:        appbot.NewBot(svcs, discord, sender, cfg),
		DB:         db,
		Services:   svcs,
		Discord:    discord,
		Dispatcher: sender,
		Session:    session,
	}
}

func (env *workflowEnv) sendMessage(t *testing.T, userID, displayName, content string) {
	t.Helper()
	env.Discord.setDisplayName(userID, displayName)

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg_" + userID,
			Content:   content,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			Author:    &discordgo.User{ID: userID, Username: displayName},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: userID, Username: displayName},
				Nick: displayName,
			},
		},
	}

	env.Bot.HandleMessageEvent(env.Session, msg)
}

func (env *workflowEnv) sendReaction(t *testing.T, userID, displayName, messageID, emoji string) {
	t.Helper()
	env.Discord.setDisplayName(userID, displayName)

	reaction := &discordgo.MessageReactionAdd{
		MessageReaction: &discordgo.MessageReaction{
			UserID:    userID,
			MessageID: messageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			Emoji:     discordgo.Emoji{Name: emoji},
		},
		Member: &discordgo.Member{
			User: &discordgo.User{ID: userID, Username: displayName},
			Nick: displayName,
		},
	}

	env.Bot.HandleReactionEvent(env.Session, reaction)
}

func (env *workflowEnv) seedClass(t *testing.T, classCode, name string) int64 {
	t.Helper()

	id, err := env.DB.CreateClass(data.Class{
		ClassCode: classCode,
		Name:      name,
	})
	require.NoError(t, err)
	return id
}

func (env *workflowEnv) seedProblem(t *testing.T, name, url string, classID int64, difficulty, topic string) int64 {
	t.Helper()

	id, err := env.DB.AddProblem(data.Problem{
		Name:       name,
		URL:        url,
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	require.NoError(t, err)
	return id
}

func (env *workflowEnv) seedClassroom(t *testing.T, roomID, classCode string) {
	t.Helper()
	require.NoError(t, env.DB.AddClassroom(data.Classroom{RoomID: roomID, ClassCode: classCode}))
}

func (env *workflowEnv) seedClassAccessRole(t *testing.T, classCode, roleID string) {
	t.Helper()
	require.NoError(t, env.DB.AddClassAccessRole(data.ClassAccessRole{ClassCode: classCode, RoleID: roleID}))
}

func (env *workflowEnv) seedUser(t *testing.T, userID, displayName string) {
	t.Helper()
	require.NoError(t, env.DB.EnsureUser(data.User{UserID: userID, DisplayName: displayName}))
}

func (env *workflowEnv) assertUserExists(t *testing.T, userID, expectedName string) {
	t.Helper()

	user, err := env.DB.GetUserByUserID(userID)
	require.NoError(t, err)
	require.Equal(t, expectedName, user.DisplayName)
}

func (env *workflowEnv) assertPendingConfirmation(t *testing.T, messageID, expectedUserID string, expectedProblemID int64) {
	t.Helper()

	pending, err := env.Services.Solution.GetPendingConfirmationByMessageID(messageID)
	require.NoError(t, err, "expected pending confirmation for message %q", messageID)
	require.Equal(t, &service_model.PendingConfirmation{
		MessageID: messageID,
		UserID:    expectedUserID,
		ProblemID: expectedProblemID,
	}, pending)
}

func (env *workflowEnv) assertPendingConfirmationRemoved(t *testing.T, messageID string) {
	t.Helper()

	exists, err := env.Services.Solution.ExistsPendingConfirmation(messageID)
	require.NoError(t, err, "failed checking pending confirmation %q", messageID)
	assert.False(t, exists, "expected pending confirmation %q to be removed", messageID)
}

func (env *workflowEnv) assertProblemSolved(t *testing.T, userID string, problemID int64) {
	t.Helper()

	isSolved, err := env.DB.HasUserSolvedProblem(userID, problemID)
	require.NoError(t, err, "failed to query solved state")
	assert.True(t, isSolved, "expected problem %d to be solved by user %q", problemID, userID)
}

func (env *workflowEnv) assertProblemNotSolved(t *testing.T, userID string, problemID int64) {
	t.Helper()

	isSolved, err := env.DB.HasUserSolvedProblem(userID, problemID)
	require.NoError(t, err, "failed to query solved state")
	assert.False(t, isSolved, "expected problem %d to remain unsolved by user %q", problemID, userID)
}

func (env *workflowEnv) setRoles(userID string, roles []string) {
	env.Discord.setRoles(userID, roles)
}

func newStubDiscordProvider() *stubDiscordProvider {
	return &stubDiscordProvider{
		userRoles:    make(map[string][]string),
		displayNames: make(map[string]string),
		messages:     make(map[string]*discordgo.Message),
	}
}

type stubDiscordProvider struct {
	isAdminFunc           func(guildID, userID string) (bool, error)
	isMentorFunc          func(guildID, userID string) (bool, error)
	getCachedMemberFunc   func(guildID, userID string) (*discordgo.Member, error)
	fetchMemberFunc       func(guildID, userID string) (*discordgo.Member, error)
	editEmbedFunc         func(channelID, messageID string, embed *discordgo.MessageEmbed) error
	removeBotReactionFunc func(channelID, messageID, emojiID string) error

	userRoles        map[string][]string
	displayNames     map[string]string
	messages         map[string]*discordgo.Message
	removedReactions []string
}

func (s *stubDiscordProvider) IsAdmin(guildID, userID string) (bool, error) {
	if s.isAdminFunc != nil {
		return s.isAdminFunc(guildID, userID)
	}
	return false, nil
}

func (s *stubDiscordProvider) IsMentor(guildID, userID string) (bool, error) {
	if s.isMentorFunc != nil {
		return s.isMentorFunc(guildID, userID)
	}
	return false, nil
}

func (s *stubDiscordProvider) GetCachedMember(guildID, userID string) (*discordgo.Member, error) {
	if s.getCachedMemberFunc != nil {
		return s.getCachedMemberFunc(guildID, userID)
	}

	displayName := userID
	if name, ok := s.displayNames[userID]; ok {
		displayName = name
	}

	roles := append([]string{}, s.userRoles[userID]...)

	return &discordgo.Member{
		User:  &discordgo.User{ID: userID, Username: displayName},
		Nick:  displayName,
		Roles: roles,
	}, nil
}

func (s *stubDiscordProvider) FetchMember(guildID, userID string) (*discordgo.Member, error) {
	if s.fetchMemberFunc != nil {
		return s.fetchMemberFunc(guildID, userID)
	}
	return s.GetCachedMember(guildID, userID)
}

func (s *stubDiscordProvider) FetchMessage(channelID, messageID string) (*discordgo.Message, error) {
	msg, ok := s.messages[messageID]
	if !ok {
		return nil, errors.New("workflow stub message not found")
	}
	return msg, nil
}

func (s *stubDiscordProvider) EditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed) error {
	if s.editEmbedFunc != nil {
		return s.editEmbedFunc(channelID, messageID, embed)
	}

	msg, ok := s.messages[messageID]
	if !ok {
		return errors.New("workflow stub message not found")
	}
	msg.Embeds = []*discordgo.MessageEmbed{embed}
	return nil
}

func (s *stubDiscordProvider) RemoveBotReaction(channelID, messageID, emojiID string) error {
	if s.removeBotReactionFunc != nil {
		return s.removeBotReactionFunc(channelID, messageID, emojiID)
	}
	s.removedReactions = append(s.removedReactions, emojiID)
	return nil
}

func (s *stubDiscordProvider) setRoles(userID string, roles []string) {
	s.userRoles[userID] = append([]string{}, roles...)
}

func (s *stubDiscordProvider) setDisplayName(userID, displayName string) {
	s.displayNames[userID] = displayName
}

func (s *stubDiscordProvider) storeMessage(msg *discordgo.Message) {
	s.messages[msg.ID] = msg
}

func (s *stubDiscordProvider) removedReactionEmojis() []string {
	return append([]string{}, s.removedReactions...)
}

type fakeSender struct {
	mu       sync.Mutex
	messages []dispatcher.MessageResponse
}

func (f *fakeSender) Send(msg dispatcher.MessageResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msg)
}

func (f *fakeSender) Messages() []dispatcher.MessageResponse {
	f.mu.Lock()
	defer f.mu.Unlock()
	msgs := make([]dispatcher.MessageResponse, len(f.messages))
	copy(msgs, f.messages)
	return msgs
}

func (f *fakeSender) LastMessage(t *testing.T) dispatcher.MessageResponse {
	t.Helper()

	msgs := f.Messages()
	require.NotEmpty(t, msgs, "expected at least one dispatched message")
	return msgs[len(msgs)-1]
}

func newConfirmationMessage(id, replyTo string, args []string, originalAuthorID string) *discordgo.Message {
	embed := &discordgo.MessageEmbed{
		Title: "CONFIRMATION",
		Footer: &discordgo.MessageEmbedFooter{
			Text: dispatcher.ConfirmationCommandFooterPrefix + strings.Join(args, " "),
		},
	}

	return &discordgo.Message{
		ID:        id,
		ChannelID: testChannelID,
		Author:    &discordgo.User{ID: testBotUserID},
		Embeds:    []*discordgo.MessageEmbed{embed},
		ReferencedMessage: &discordgo.Message{
			ID:        replyTo,
			ChannelID: testChannelID,
			Author:    &discordgo.User{ID: originalAuthorID},
		},
	}
}
