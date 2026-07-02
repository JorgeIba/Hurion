package script

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// FakeDiscordProvider is a mock implementation of DiscordProvider
// used for the offline and script runner modes. It bypasses real Discord API calls.
type FakeDiscordProvider struct {
	UserRoles    map[string][]string
	AdminRoleID  string
	MentorRoleID string
}

func NewFakeDiscordProvider(adminRoleID, mentorRoleID string) *FakeDiscordProvider {
	return &FakeDiscordProvider{
		UserRoles:    make(map[string][]string),
		AdminRoleID:  adminRoleID,
		MentorRoleID: mentorRoleID,
	}
}

func (f *FakeDiscordProvider) SetUserRoles(userID string, roles []string) {
	if f.UserRoles == nil {
		f.UserRoles = make(map[string][]string)
	}
	f.UserRoles[userID] = roles
}

// -- DiscordValidator --

func (f *FakeDiscordProvider) IsAdmin(guildID, userID string) (bool, error) {
	roles := f.UserRoles[userID]
	for _, role := range roles {
		if role == f.AdminRoleID {
			return true, nil
		}
	}
	return false, nil
}

func (f *FakeDiscordProvider) IsMentor(guildID, userID string) (bool, error) {
	roles := f.UserRoles[userID]
	for _, role := range roles {
		if role == f.MentorRoleID {
			return true, nil
		}
	}
	return false, nil
}

// -- DiscordFetcher --

func (f *FakeDiscordProvider) FetchMessage(channelID, messageID string) (*discordgo.Message, error) {
	// Return a dummy message
	return &discordgo.Message{
		ID:        messageID,
		ChannelID: channelID,
		Content:   "Fake message content",
	}, nil
}

func (f *FakeDiscordProvider) GetCachedMember(guildID, userID string) (*discordgo.Member, error) {
	return f.FetchMember(guildID, userID)
}

func (f *FakeDiscordProvider) FetchMember(guildID, userID string) (*discordgo.Member, error) {
	// Synthesize a valid member so GetMemberSafe and other user checks succeed offline.
	if userID == "" {
		return nil, fmt.Errorf("empty user ID")
	}

	roles, ok := f.UserRoles[userID]
	if !ok {
		roles = []string{}
	}

	return &discordgo.Member{
		User: &discordgo.User{
			ID:       userID,
			Username: "User_" + userID,
		},
		Roles: roles,
	}, nil
}

// -- DiscordMessageModifier --

func (f *FakeDiscordProvider) EditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed) error {
	// Silently succeed
	return nil
}

func (f *FakeDiscordProvider) RemoveBotReaction(channelID, messageID, emojiID string) error {
	// Silently succeed
	return nil
}

func (f *FakeDiscordProvider) BotUserID() string {
	return "mock_bot_user"
}
