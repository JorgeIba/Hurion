package authorization

import (
	"errors"
	"slices"
	"strings"

	"github.com/JorgeIba/Hurion/app/providers"
)

// DiscordRoleSource fetches current role IDs from the configured Discord server.
// It does not interpret roles or decide application permissions.
type DiscordRoleSource struct {
	fetcher providers.DiscordFetcher
	guildID string
}

func NewDiscordRoleSource(fetcher providers.DiscordFetcher, guildID string) *DiscordRoleSource {
	return &DiscordRoleSource{
		fetcher: fetcher,
		guildID: guildID,
	}
}

func (s *DiscordRoleSource) FetchRoles(userID string) ([]string, error) {
	if s.guildID == "" || strings.TrimSpace(userID) == "" {
		return nil, errors.New("server and user are required")
	}

	member, err := s.fetcher.FetchMember(s.guildID, userID)
	if err != nil {
		return nil, err
	}

	if member == nil || member.User == nil || member.User.ID != userID {
		return nil, errors.New("could not verify Discord member identity")
	}

	return slices.Clone(member.Roles), nil
}
