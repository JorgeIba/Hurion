package integration

import (
	"errors"
	"sync"

	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/bwmarrin/discordgo"
)

// StubDiscordProvider implements Discord provider interfaces for integration tests.
type StubDiscordProvider struct {
	IsAdminFunc           func(guildID, userID string) (bool, error)
	IsMentorFunc          func(guildID, userID string) (bool, error)
	GetCachedMemberFunc   func(guildID, userID string) (*discordgo.Member, error)
	FetchMemberFunc       func(guildID, userID string) (*discordgo.Member, error)
	FetchMessageFunc      func(channelID, messageID string) (*discordgo.Message, error)
	EditEmbedFunc         func(channelID, messageID string, embed *discordgo.MessageEmbed) error
	RemoveBotReactionFunc func(channelID, messageID, emojiID string) error

	// UserRoles stores roles for specific users to avoid clobbering in tests.
	UserRoles map[string][]string
}

func (s *StubDiscordProvider) IsAdmin(guildID, userID string) (bool, error) {
	if s.IsAdminFunc != nil {
		return s.IsAdminFunc(guildID, userID)
	}
	return false, nil
}

func (s *StubDiscordProvider) IsMentor(guildID, userID string) (bool, error) {
	if s.IsMentorFunc != nil {
		return s.IsMentorFunc(guildID, userID)
	}
	return false, nil
}

func (s *StubDiscordProvider) GetCachedMember(guildID, userID string) (*discordgo.Member, error) {
	if s.GetCachedMemberFunc != nil {
		return s.GetCachedMemberFunc(guildID, userID)
	}

	roles := []string{}
	if s.UserRoles != nil {
		if r, ok := s.UserRoles[userID]; ok {
			roles = r
		}
	}

	return &discordgo.Member{
		User:  &discordgo.User{ID: userID},
		Roles: roles,
	}, nil
}

func (s *StubDiscordProvider) FetchMember(guildID, userID string) (*discordgo.Member, error) {
	if s.FetchMemberFunc != nil {
		return s.FetchMemberFunc(guildID, userID)
	}
	return s.GetCachedMember(guildID, userID)
}

func (s *StubDiscordProvider) FetchMessage(channelID, messageID string) (*discordgo.Message, error) {
	if s.FetchMessageFunc != nil {
		return s.FetchMessageFunc(channelID, messageID)
	}
	return nil, errors.New("StubDiscordProvider.FetchMessage not configured")
}

func (s *StubDiscordProvider) EditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed) error {
	if s.EditEmbedFunc != nil {
		return s.EditEmbedFunc(channelID, messageID, embed)
	}
	return errors.New("StubDiscordProvider.EditEmbed not configured")
}

func (s *StubDiscordProvider) RemoveBotReaction(channelID, messageID, emojiID string) error {
	if s.RemoveBotReactionFunc != nil {
		return s.RemoveBotReactionFunc(channelID, messageID, emojiID)
	}
	return errors.New("StubDiscordProvider.RemoveBotReaction not configured")
}

// FakeDispatcher captures messages instead of sending them to Discord.
type FakeDispatcher struct {
	mu           sync.Mutex
	SentMessages []dispatcher.MessageResponse
}

func (f *FakeDispatcher) Send(msg dispatcher.MessageResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.SentMessages = append(f.SentMessages, msg)
}

func (f *FakeDispatcher) GetSentMessages() []dispatcher.MessageResponse {
	f.mu.Lock()
	defer f.mu.Unlock()
	msgs := make([]dispatcher.MessageResponse, len(f.SentMessages))
	copy(msgs, f.SentMessages)
	return msgs
}
