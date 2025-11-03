package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type ConfirmationHandler func(originalContext *context.ContextMessage, args []string, confirmationContext *context.ContextMessage)
type WarningMessageGenerator func(args []string) string

// ConfirmableCommand defines a command that requires confirmation before execution
type ConfirmableCommand struct {
	WarningMessage WarningMessageGenerator // Function to generate the warning message with placeholders
	Handler        ConfirmationHandler     // Handler function to execute when confirmed
}

// confirmableCommands stores all commands that require confirmation
// To add a new confirmable command:
// 1. Add entry here with the warning message generator and handler
// 2. Implement the handler function if it doesn't exist
var confirmableCommands = map[string]ConfirmableCommand{
	"!exterminate": {
		WarningMessage: func(args []string) string {
			userName := args[1]
			return fmt.Sprintf("⚠️ **WARNING**: This will permanently delete user `%s` and ALL their data from ALL classes.\n\nAre you sure you want to exterminate this user?", userName)
		},
		Handler: handleExterminateConfirmation,
	},
	"!clear_class": {
		WarningMessage: func(args []string) string {
			classCode := args[1]
			return fmt.Sprintf("⚠️ **WARNING**: This will delete ALL student enrollments, solved problems, and pending confirmations for class `%s`.\n\nThe class itself will remain but will be empty. Are you sure?", classCode)
		},
		Handler: handleClearClassConfirmation,
	},
	"!deregister": {
		WarningMessage: func(args []string) string {
			if len(args) == 2 {
				// Enrolled deregistering themselves
				classCode := args[1]
				return fmt.Sprintf("⚠️ **WARNING**: This will remove you from class `%s` and delete ALL your solved problems and pending confirmations in this class.\n\nAre you sure you want to deregister?", classCode)
			} else if len(args) == 3 {
				// Mentor deregistering a user
				userName := args[1]
				classCode := args[2]
				return fmt.Sprintf("⚠️ **WARNING**: This will remove user `%s` from class `%s` and delete ALL their solved problems and pending confirmations in this class.\n\nAre you sure you want to deregister?", userName, classCode)
			}
			return "⚠️ **WARNING**: Invalid deregister command."
		},
		Handler: handleDeregisterConfirmation,
	},
	"!remove_problem": {
		WarningMessage: func(args []string) string {
			problemID := args[1]
			return fmt.Sprintf("⚠️ **WARNING**: This will delete problem `%s` and ALL STUDENT ACs (solved_problems) for this problem.\n\nAre you sure you want to remove this problem?", problemID)
		},
		Handler: handleRemoveProblemConfirmation,
	},
}

// GetConfirmationMessage returns the warning message for a confirmable command
// Returns empty string if command is not confirmable
func GetConfirmationMessage(command string, args []string) string {
	if cmd, exists := confirmableCommands[command]; exists {
		return cmd.WarningMessage(args)
	}
	return ""
}

// executeConfirmedCommand routes the confirmed command to the appropriate handler
func executeConfirmedCommand(originalContext *context.ContextMessage, args []string, confirmationContext *context.ContextMessage) {
	command := args[0]

	// Check if command exists in confirmable commands and execute its handler
	if cmd, exists := confirmableCommands[command]; exists {
		cmd.Handler(originalContext, args, confirmationContext)
	} else {
		setMessageAsError(confirmationContext)
	}
}

func handleExterminateConfirmation(originalContext *context.ContextMessage, args []string, c *context.ContextMessage) {
	if err := admin.HandleExterminate(originalContext); err != nil {
		setMessageAsError(c)
	} else {
		setMessageAsExecuted(c, fmt.Sprintf("User %s has been exterminated", args[1]))
	}
}

func handleClearClassConfirmation(originalContext *context.ContextMessage, args []string, c *context.ContextMessage) {
	if err := admin.HandleClearClass(originalContext); err != nil {
		setMessageAsError(c)
	} else {
		setMessageAsExecuted(c, fmt.Sprintf("Class %s has been cleared", args[1]))
	}
}

func handleDeregisterConfirmation(originalContext *context.ContextMessage, args []string, c *context.ContextMessage) {
	switch len(args) {
	case 2:
		// Enrolled user deregistering themselves
		if err := enrolled.HandleDeregisterUser(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("You have been deregistered from class %s", args[1]))
		}
	case 3:
		// Mentor deregistering a user
		if err := mentor.HandleDeregisterUser(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("User %s has been deregistered from class %s", args[1], args[2]))
		}
	default:
		setMessageAsError(c)
	}
}

func handleRemoveProblemConfirmation(originalContext *context.ContextMessage, args []string, c *context.ContextMessage) {
	switch len(args) {
	case 2:
		if err := mentor.HandleRemoveProblem(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("Problem %s has been removed", args[1]))
		}
	default:
		setMessageAsError(c)
	}
}
