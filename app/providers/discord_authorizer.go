package providers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JorgeIba/Hurion/app/authorization"
)

// DiscordAuthorizer evaluates domain permissions using current guild roles and
// immutable configuration. It does not resolve database resources.
type DiscordAuthorizer struct {
	fetcher                   DiscordFetcher
	adminRoleID, mentorRoleID string
	classRoles                map[string][]string
}

// NewDiscordAuthorizer copies the policy; absent class overrides inherit the
// global mentor role and explicit empty overrides allow administrators only.
func NewDiscordAuthorizer(fetcher DiscordFetcher, adminRoleID, mentorRoleID string, overrides map[string][]string) *DiscordAuthorizer {
	roles := make(map[string][]string, len(overrides))
	for code, ids := range overrides {
		copied := slices.Clone(ids)
		for i := range copied {
			copied[i] = strings.TrimSpace(copied[i])
		}
		roles[strings.TrimSpace(code)] = copied
	}
	return &DiscordAuthorizer{fetcher: fetcher, adminRoleID: adminRoleID, mentorRoleID: mentorRoleID, classRoles: roles}
}

// Authorize evaluates ownership before role lookup for self-service AC removal.
// A role-lookup failure always fails closed; no permissions are cached across requests.
func (a *DiscordAuthorizer) Authorize(actor authorization.Actor, permission authorization.Permission, resource authorization.Resource) error {
	if actor.UserID == "" || actor.GuildID == "" {
		return authorization.ErrDenied
	}
	if permission == authorization.UndoAC && resource.OwnerUserID == actor.UserID {
		return nil
	}
	if a.fetcher == nil {
		return fmt.Errorf("Discord member fetcher is not configured")
	}
	member, err := a.fetcher.FetchMember(actor.GuildID, actor.UserID)
	if err != nil {
		return err
	}
	if member == nil {
		return fmt.Errorf("Discord member response is missing")
	}
	has := func(ids []string) bool {
		for _, id := range ids {
			if id != "" && slices.Contains(member.Roles, id) {
				return true
			}
		}
		return false
	}
	// Unknown permissions must not accidentally be granted to administrators.
	switch permission {
	case authorization.Admin, authorization.ManageProblems, authorization.ManageACs, authorization.UndoAC, authorization.MentorIntake, authorization.AccessClass:
	default:
		return authorization.ErrDenied
	}
	if has([]string{a.adminRoleID}) {
		return nil
	}
	switch permission {
	case authorization.ManageProblems, authorization.ManageACs, authorization.UndoAC:
		code := strings.TrimSpace(resource.ClassCode)
		if code == "" {
			return authorization.ErrDenied
		}
		ids, found := a.classRoles[code]
		if !found {
			ids = []string{a.mentorRoleID}
		}
		if has(ids) {
			return nil
		}
	case authorization.MentorIntake:
		if has([]string{a.mentorRoleID}) {
			return nil
		}
		for _, ids := range a.classRoles {
			if has(ids) {
				return nil
			}
		}
	case authorization.AccessClass:
		if has(resource.AccessRoleIDs) {
			return nil
		}
	}
	return authorization.ErrDenied
}
