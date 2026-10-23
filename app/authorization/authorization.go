// Package authorization defines the domain contract for authorizing operations.
package authorization

import "errors"

// Actor identifies the executor, independently of an operation's target user.
type Actor struct{ GuildID, UserID string }

// Permission describes an operation, not a Discord role.
type Permission string

const (
	Admin          Permission = "admin"
	ManageProblems Permission = "manage_problems"
	ManageACs      Permission = "manage_acs"
	UndoAC         Permission = "undo_ac"
	MentorIntake   Permission = "mentor_intake"
	AccessClass    Permission = "access_class"
)

// Resource contains domain facts resolved by the service, never Discord objects.
type Resource struct {
	ClassCode     string
	OwnerUserID   string
	AccessRoleIDs []string
}

// ErrDenied distinguishes a denied operation from an unavailable identity source.
var ErrDenied = errors.New("permission denied")

// Authorizer resolves policy for an actor and operation. Implementations must not
// trust caller-supplied role snapshots or authorize missing actor identities.
//
//go:generate go tool mockgen -destination=mock_authorization/mock_authorizer.go -package=mock_authorization . Authorizer
type Authorizer interface {
	Authorize(Actor, Permission, Resource) error
}
