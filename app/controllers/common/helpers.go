package common

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/controllerserr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

// ResolveClassContext determines the active class for a command.
// Logic:
// 1. If an argument exists at the specified index, try to use it as a class code.
// 2. If it's a valid class, return it.
// 3. If it failed with a "Not Found" error, proceed to step 4 (fallback).
// 4. If it failed with any other error, return that error (abort).
// 5. Fallback: check if the current channel is a registered Classroom.
func ResolveClassContext(c *context.ContextMessage, argIndex int) (service_model.Class, error) {
	// 1. Try to resolve from argument first
	if len(c.Args) > argIndex {
		arg := c.Args[argIndex]
		class, err := c.Services.Class.GetClassByClassCode(arg)
		if err == nil {
			return class, nil
		}

		// If it's a hard error (not just "not found"), we stop here.
		var notFoundErr *serviceerr.ClassCodeNotFoundError
		if !errors.As(err, &notFoundErr) {
			return service_model.Class{}, err
		}
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

// RequireClassAccess verifies if the user has permission to access a class.
func RequireClassAccess(c *context.ContextMessage, class service_model.Class) (bool, error) {
	// Admin Bypass
	isAdmin, err := c.DiscordValidator.IsAdmin(c.ExecutorGuildID, c.Executor.UserID)
	if err == nil && isAdmin {
		return true, nil
	}

	// Get member roles
	member, err := c.DiscordFetcher.GetCachedMember(c.ExecutorGuildID, c.Executor.UserID)
	if err != nil {
		return false, controllerserr.NewInternalError("error fetching member from discord", err)
	}

	if member == nil {
		return false, controllerserr.NewInternalError("could not fetch member roles", fmt.Errorf("member is nil"))
	}

	return c.Services.Enrollment.UserWithRolesHasAccessClass(member.Roles, class)
}
