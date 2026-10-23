// Package authtest provides explicit authorizers for isolated test fixtures.
package authtest

import "github.com/JorgeIba/Hurion/app/authorization"

// Allow is only for tests whose subject is not authorization.
type Allow struct{}

func (Allow) Authorize(authorization.Actor, authorization.Permission, authorization.Resource) error {
	return nil
}

// RoleChecker adapts legacy test doubles while authorization moves into services.
// Production authorization uses DiscordAuthorizer and current member roles.
//
//go:generate go tool mockgen -destination=mock_role_checker.go -package=authtest . RoleChecker
type RoleChecker interface {
	IsAdmin(string, string) (bool, error)
	IsMentor(string, string) (bool, error)
}
type Roles struct{ Checker RoleChecker }

func (a Roles) Authorize(actor authorization.Actor, p authorization.Permission, r authorization.Resource) error {
	if p == authorization.UndoAC && actor.UserID == r.OwnerUserID {
		return nil
	}
	if p != authorization.Admin && p != authorization.AccessClass {
		allowed, err := a.Checker.IsMentor(actor.GuildID, actor.UserID)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	allowed, err := a.Checker.IsAdmin(actor.GuildID, actor.UserID)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	return authorization.ErrDenied
}

// IntakeRoles isolates transport tests: only form intake uses the role checker;
// target policy is replaced by an explicit allow stub and tested at the service boundary.
type IntakeRoles struct{ Checker RoleChecker }

func (a IntakeRoles) Authorize(actor authorization.Actor, p authorization.Permission, r authorization.Resource) error {
	if p == authorization.MentorIntake {
		return (Roles{Checker: a.Checker}).Authorize(actor, p, r)
	}
	return nil
}

// Actor is a valid executor for fixtures that are unrelated to transport identity.
func Actor() authorization.Actor { return authorization.Actor{GuildID: "test", UserID: "actor"} }
