package bot

import "strings"

type UserRole string

const (
	RoleAll      UserRole = "All"
	RoleEnrolled UserRole = "Enrolled"
	RoleMentor   UserRole = "Mentor"
	RoleAdmin    UserRole = "Admin"
)

type UserRoleSet map[UserRole]bool

func NewUserRoleSet(roles ...UserRole) UserRoleSet {
	set := make(UserRoleSet)
	for _, role := range roles {
		set[role] = true
	}
	return set
}

func (s UserRoleSet) Contains(role UserRole) bool {
	return s[role]
}

func (s UserRoleSet) String() string {
	roles := []string{}
	orderedRoles := []UserRole{RoleAll, RoleEnrolled, RoleMentor, RoleAdmin}
	for _, role := range orderedRoles {
		if s[role] {
			roles = append(roles, string(role))
		}
	}
	return strings.Join(roles, "/")
}
