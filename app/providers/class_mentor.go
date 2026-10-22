package providers

import (
	"fmt"
	"slices"
	"strings"
)

// ClassMentorValidator is implemented when a validator has class-specific policy.
// Validators without it retain the legacy, globally scoped mentor policy.
type ClassMentorValidator interface {
	CanManageClass(guildID, userID, classCode string) (bool, error)
}

type classMentorProvider struct {
	DiscordProvider
	roles map[string][]string
}

// WithClassMentorRoles applies configured overrides without persisting Discord
// roles in SQLite. A missing map key inherits the global role; an empty list
// allows only administrators. The configuration is copied and remains immutable.
func WithClassMentorRoles(provider DiscordProvider, roles map[string][]string) DiscordProvider {
	copied := make(map[string][]string, len(roles))
	for code, ids := range roles {
		normalized := slices.Clone(ids)
		for i, id := range normalized {
			normalized[i] = strings.TrimSpace(id)
		}
		copied[strings.TrimSpace(code)] = normalized
	}
	return &classMentorProvider{DiscordProvider: provider, roles: copied}
}

// IsMentor is only an intake gate: membership in any mentor role permits opening
// a form. Every target must subsequently pass CanManageClass before mutation.
func (p *classMentorProvider) IsMentor(guildID, userID string) (bool, error) {
	allowed, err := p.DiscordProvider.IsMentor(guildID, userID)
	if err != nil || allowed {
		return allowed, err
	}
	var roles []string
	for _, ids := range p.roles {
		roles = append(roles, ids...)
	}
	return p.hasAnyRole(guildID, userID, roles)
}

func (p *classMentorProvider) CanManageClass(guildID, userID, classCode string) (bool, error) {
	classCode = strings.TrimSpace(classCode)
	if guildID == "" || userID == "" || classCode == "" {
		return false, fmt.Errorf("server, actor and class are required")
	}
	admin, err := p.IsAdmin(guildID, userID)
	if err != nil || admin {
		return admin, err
	}
	roles, overridden := p.roles[classCode]
	if !overridden {
		return p.DiscordProvider.IsMentor(guildID, userID)
	}
	return p.hasAnyRole(guildID, userID, roles)
}

func (p *classMentorProvider) hasAnyRole(guildID, userID string, roles []string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}
	member, err := p.FetchMember(guildID, userID)
	if err != nil {
		return false, err
	}
	if member == nil {
		return false, fmt.Errorf("could not resolve server member")
	}
	for _, role := range roles {
		if slices.Contains(member.Roles, role) {
			return true, nil
		}
	}
	return false, nil
}
