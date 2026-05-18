package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	controllers_common "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type ConfirmationHandler func(originalContext *context.ContextMessage, confirmationContext *context.ContextMessage)
type WarningMessageGenerator func(c *context.ContextMessage) string

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
	Exterminate.String(): {
		WarningMessage: func(c *context.ContextMessage) string {
			if len(c.Args) < 2 {
				return "⚠️ **WARNING**: Invalid exterminate command."
			}
			userParam := c.Args[1]
			userID, ok := controllers_common.ParseUserID(userParam)
			displayName := userParam
			if ok {
				displayName = controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)
			}
			return fmt.Sprintf("⚠️ **WARNING**: This will permanently delete user `%s` and ALL their data from ALL classes.\n\nAre you sure you want to exterminate this user?", displayName)
		},
		Handler: handleExterminateConfirmation,
	},
	ClearClass.String(): {
		WarningMessage: func(c *context.ContextMessage) string {
			classCode := controllers_common.ResolveClassCodeArg(c, 1)
			if classCode == nil {
				return "⚠️ **WARNING**: Invalid clear class command."
			}
			return fmt.Sprintf("⚠️ **WARNING**: This will RESET class `%v` by deleting all problems, solved records, pending confirmations, and score history.\n\nThe class, linked rooms, and linked roles will stay. Are you sure?", classCode)
		},
		Handler: handleClearClassConfirmation,
	},
	ExterminateClass.String(): {
		WarningMessage: func(c *context.ContextMessage) string {
			if len(c.Args) < 2 {
				return "⚠️ **WARNING**: Invalid exterminate class command."
			}
			classCode := c.Args[1]
			return fmt.Sprintf("⚠️ **WARNING**: This will COMPLETELY DELETE class `%s`, including its problems, solved records, linked rooms, and linked roles.\n\nThis action cannot be undone. Are you sure?", classCode)
		},
		Handler: handleExterminateClassConfirmation,
	},
	RemoveProblem.String(): {
		WarningMessage: func(c *context.ContextMessage) string {
			if len(c.Args) < 2 {
				return "⚠️ **WARNING**: Invalid remove problem command."
			}
			problemID := c.Args[1]
			return fmt.Sprintf("⚠️ **WARNING**: This will delete problem `%s` and ALL STUDENT ACs (solved_problems) for this problem.\n\nAre you sure you want to remove this problem?", problemID)
		},
		Handler: handleRemoveProblemConfirmation,
	},
}

// GetConfirmationMessage returns the warning message for a confirmable command
// Returns empty string if command is not confirmable
func GetConfirmationMessage(c *context.ContextMessage, command string) string {
	if cmd, exists := confirmableCommands[command]; exists {
		return cmd.WarningMessage(c)
	}
	return ""
}

// executeConfirmedCommand routes the confirmed command to the appropriate handler
func executeConfirmedCommand(originalContext *context.ContextMessage, confirmationContext *context.ContextMessage) {
	command := originalContext.Args[0]

	// Check if command exists in confirmable commands and execute its handler
	if cmd, exists := confirmableCommands[command]; exists {
		cmd.Handler(originalContext, confirmationContext)
	} else {
		setMessageAsError(confirmationContext)
	}
}

func handleExterminateConfirmation(originalContext *context.ContextMessage, c *context.ContextMessage) {
	if err := admin.HandleExterminate(originalContext); err != nil {
		setMessageAsError(c)
	} else {
		setMessageAsExecuted(c, fmt.Sprintf("User %s has been exterminated", originalContext.Args[1]))
	}
}

func handleClearClassConfirmation(originalContext *context.ContextMessage, c *context.ContextMessage) {
	if err := admin.HandleClearClass(originalContext); err != nil {
		setMessageAsError(c)
	} else {
		classCode := controllers_common.ResolveClassCodeArg(originalContext, 1)
		setMessageAsExecuted(c, fmt.Sprintf("Class %v has been cleared", classCode))
	}
}

func handleExterminateClassConfirmation(originalContext *context.ContextMessage, c *context.ContextMessage) {
	if err := admin.HandleExterminateClass(originalContext); err != nil {
		setMessageAsError(c)
	} else {
		setMessageAsExecuted(c, fmt.Sprintf("Class %s has been exterminated", originalContext.Args[1]))
	}
}

func handleRemoveProblemConfirmation(originalContext *context.ContextMessage, c *context.ContextMessage) {
	switch len(originalContext.Args) {
	case 2:
		if err := mentor.HandleRemoveProblem(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("Problem %s has been removed", originalContext.Args[1]))
		}
	default:
		setMessageAsError(c)
	}
}
