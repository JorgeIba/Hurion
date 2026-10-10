package common

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/customerr/controllerserr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ResolveClassContext determines the active class for a command.
// Logic:
// 1. If an argument exists at the specified index, try to use it as a class code.
// 2. If an argument is provided, return its result immediately (success or failure). Do NOT fall back.
// 3. Fallback: If NO argument is provided, check if the current channel is a registered Classroom.
func ResolveClassContext(c *context.ContextMessage, argIndex int) (service_model.Class, error) {
	// 1. Try to resolve from argument first
	if len(c.Args) > argIndex {
		arg := c.Args[argIndex]
		return c.Services.Class.GetClassByClassCode(arg)
	}

	// 2. Fallback to channel context
	channelID := GetMessageChannelID(c)
	if channelID == "" {
		return service_model.Class{}, controllerserr.NewValidationError("channel context is not available", nil)
	}

	code, found, err := c.Services.Class.GetClassCodeFromRoom(channelID)
	if err != nil {
		return service_model.Class{}, err
	}

	if found {
		return c.Services.Class.GetClassByClassCode(code)
	}

	// 3. Fail: No valid class found anywhere
	return service_model.Class{}, serviceerr.NewNotFoundInfoError("could not identify the target class", nil)
}

// ResolveClassCodeArg returns the explicit class argument when present, or the
// class code linked to the current room when the command is using channel
// context. It is intended for error formatting paths that still need the class
// identity even if class resolution failed.
func ResolveClassCodeArg(c *context.ContextMessage, argIndex int) any {
	if len(c.Args) > argIndex {
		return c.Args[argIndex]
	}

	channelID := GetMessageChannelID(c)
	if channelID == "" {
		return nil
	}

	code, found, err := c.Services.Class.GetClassCodeFromRoom(channelID)
	if err != nil || !found {
		return nil
	}

	return code
}

// GetMemberSafe attempts to get a member from cache first, then falls back to live fetch.
func GetMemberSafe(c *context.ContextMessage, userID string) (*service_model.Principal, error) {
	return GetMemberPrincipal(c.DiscordFetcher, c.ExecutorGuildID, userID)
}

// GetMemberPrincipal resolves a member from the SDK cache, falling back to the
// Discord API. It returns a role snapshot usable by message and interaction controllers.
func GetMemberPrincipal(fetcher providers.DiscordFetcher, guildID, userID string) (*service_model.Principal, error) {
	if guildID == "" {
		return nil, controllerserr.NewInternalError("this command can only be used in a server", fmt.Errorf("missing guild context"))
	}

	// 1. Try cache
	member, err := fetcher.GetCachedMember(guildID, userID)
	if err != nil || member == nil {
		member, err = fetcher.FetchMember(guildID, userID)
		if err != nil {
			return nil, controllerserr.NewInternalError("could not fetch user roles from Discord API", err)
		}
	}
	if member == nil || member.User == nil || member.User.ID != userID {
		return nil, controllerserr.NewInternalError("could not verify Discord member identity", nil)
	}

	return &service_model.Principal{
		UserID:      userID,
		DisplayName: GetDisplayNameFromMember(member),
		Roles:       append([]string(nil), member.Roles...),
	}, nil
}

// GetMemberPrincipals resolves distinct IDs once per call. A failure returns no partial map.
func GetMemberPrincipals(fetcher providers.DiscordFetcher, guildID string, userIDs []string) (map[string]service_model.Principal, error) {
	principals := make(map[string]service_model.Principal, len(userIDs))
	for _, userID := range userIDs {
		if _, found := principals[userID]; found {
			continue
		}
		principal, err := GetMemberPrincipal(fetcher, guildID, userID)
		if err != nil {
			return nil, fmt.Errorf("could not resolve Discord member %s: %w", userID, err)
		}
		principals[userID] = *principal
	}
	return principals, nil
}

// RequireClassAccess verifies if the user has permission to access a class.
func RequireClassAccess(c *context.ContextMessage, class service_model.Class) (bool, error) {
	// Admin Bypass
	isAdmin, err := c.DiscordValidator.IsAdmin(c.ExecutorGuildID, c.Executor.UserID)
	if err != nil {
		return false, controllerserr.NewInternalError("error verifying admin status", err)
	}
	if isAdmin {
		return true, nil
	}

	// Get member roles
	member, err := GetMemberSafe(c, c.Executor.UserID)
	if err != nil {
		return false, err
	}

	return c.Services.Enrollment.UserWithRolesHasAccessClass(member.Roles, class)
}

// SyncParticipants scans the command arguments for user participants and ensures they exist in the DB.
func SyncParticipants(c *context.ContextMessage) {
	// Always sync the executor first
	if err := c.Services.User.EnsureUser(service_model.User{
		UserID:      c.Executor.UserID,
		DisplayName: c.Executor.DisplayName,
	}); err != nil {
		logger.Log.Errorf("System Safeguard: Failed to sync executor %s: %v", c.Executor.UserID, err)
	}

	const MIN_LENGTH_FOR_USER_ID = 13 // Arbitrary threshold to skip short args that can't be user IDs
	// Scan arguments for other potential users (mentions or long numeric IDs)
	for _, arg := range c.Args {
		// Only check if it's a mention or looks like a Discord Snowflake (numeric and length > MIN_LENGTH)
		isMention := common.IsUserMention(arg)
		isSnowflake := !isMention && len(arg) > MIN_LENGTH_FOR_USER_ID && isNumeric(arg)

		if !isMention && !isSnowflake {
			continue
		}

		userID, ok := ParseUserID(arg)
		if !ok || userID == c.Executor.UserID {
			continue
		}

		// Fetch real info for this participant and sync
		member, err := GetMemberSafe(c, userID)
		displayName := "Unknown"
		if err == nil && member != nil {
			displayName = member.User.DisplayName
		}

		if err := c.Services.User.EnsureUser(service_model.User{
			UserID:      userID,
			DisplayName: displayName,
		}); err != nil {
			logger.Log.Errorf("System Safeguard: Failed to sync participant %s: %v", userID, err)
		}
	}
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
