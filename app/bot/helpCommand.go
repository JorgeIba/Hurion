package bot

import (
	"fmt"
	"strings"
)

type commandInfo struct {
	Command     string
	Usage       string
	Description string
	UserType    string
	ExtraInfo   string
}

var commandInfoMap = map[string]commandInfo{
	"!help": {
		Command:     "!help",
		Usage:       "!help [command]",
		Description: "Show info for all commands or details for a specific command",
		UserType:    "All",
	},
	"!bind": {
		Command:     "!bind",
		Usage:       "!bind <user_name>",
		Description: "Bind your Discord account to a username for tracking progress",
		UserType:    "All",
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\")",
	},
	"!register": {
		Command:     "!register",
		Usage:       "!register <class_code>",
		Description: "Register yourself in a class with the specified class code",
		UserType:    "All",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\" or \"intermedios\")",
	},
	"!deregister": {
		Command:     "!deregister",
		Usage:       "!deregister <class_code>",
		Description: "Remove yourself from a class you're enrolled in",
		UserType:    "Enrolled",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\" or \"avanzados\")",
	},
	"!list": {
		Command:     "!list",
		Usage:       "!list <class_code>",
		Description: "Show the list of problems available for the specified class",
		UserType:    "All",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	"!score": {
		Command:     "!score",
		Usage:       "!score [class_code] [user_name]",
		Description: "Show scores for all classes, a specific class, or a specific user in a class",
		UserType:    "All",
		ExtraInfo:   "Both `class_code` and `user_name` parameters should be one word. You can use `!score` to see all scores, `!score class_code` to see scores for a class, or `!score user_name class_code` to see a specific user's scores in a class.",
	},
	"!ac": {
		Command:     "!ac",
		Usage:       "!ac <problem_id>",
		Description: "Mark a problem as solved (waiting for mentor approval with ✅ or ❌)",
		UserType:    "Enrolled",
	},
	"!undo_ac": {
		Command:     "!undo_ac",
		Usage:       "!undo_ac <problem_id>",
		Description: "Remove a previously solved problem from your solved list",
		UserType:    "Enrolled",
	},
	"!add_problem": {
		Command:     "!add_problem",
		Usage:       "!add_problem <class_code> <url> <ProblemName>",
		Description: "Add a new problem to a class with a URL and name",
		UserType:    "Mentor",
		ExtraInfo: "The `class_code` should be one word, url should have 'http://' prefix (Discord policies I guess), " +
			"and `ProblemName` can include multiple words (e.g., \"basicos http://example.com Gosha Is Hunting\")",
	},
	"!remove_problem": {
		Command:     "!remove_problem",
		Usage:       "!remove_problem <problem_id>",
		Description: "Remove a problem from a class by its ID",
		UserType:    "Mentor",
	},
	"!update_problem": {
		Command:     "!update_problem",
		Usage:       "!update_problem <problem_id> <class_code> <url> <ProblemName>",
		Description: "Update a problem's details while preserving its ID for AC records",
		UserType:    "Mentor",
		ExtraInfo:   "The `problem_id` and `class_code` should be one word, while `ProblemName` can include multiple words (e.g., \"3 basicos_1 http://updated_url.com Gosha Is Hunting 2\")",
	}, "!clear_class": {
		Command:     "!clear_class",
		Usage:       "!clear_class <class_code>",
		Description: "Remove all problems, enrollments, and scores from a class",
		UserType:    "Admin",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	}, "!create_class": {
		Command:     "!create_class",
		Usage:       "!create_class <class_code> <ClassName>",
		Description: "Create a new class with the specified code and name",
		UserType:    "Admin",
		ExtraInfo:   "The `class_code` should be one word, while `ClassName` can include multiple words (e.g., \"basicos_1 Basicos 2025\")",
	}, "!users": {
		Command:     "!users",
		Usage:       "!users <class_code>",
		Description: "Show a list of all users registered in the specified class",
		UserType:    "Mentor",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	"!classes": {
		Command:     "!classes",
		Usage:       "!classes",
		Description: "Show a list of all available classes",
		UserType:    "All",
	},
	"!my_classes": {
		Command:     "!my_classes",
		Usage:       "!my_classes",
		Description: "Show a list of all classes you're currently enrolled in",
		UserType:    "Enrolled",
	},
	"!classes_of": {
		Command:     "!classes_of",
		Usage:       "!classes_of <user_name>",
		Description: "Show a list of all classes a specific user is enrolled in",
		UserType:    "Mentor",
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\")",
	},
	"!me": {
		Command:     "!me",
		Usage:       "!me",
		Description: "Show the username bound to your Discord account",
		UserType:    "All",
	},
}

// buildGeneralHelpMessage creates the general help message listing all commands
func buildGeneralHelpMessage() string {
	var sb strings.Builder
	sb.WriteString("**📚 Huron Problem Checker Bot - Available Commands**\n\n")

	allCommands := []string{}
	enrolledCommands := []string{}
	mentorCommands := []string{}
	adminCommands := []string{}

	for cmd, info := range commandInfoMap {
		switch info.UserType {
		case "All":
			allCommands = append(allCommands, fmt.Sprintf("`%s` - %s", cmd, info.Description))
		case "Enrolled":
			enrolledCommands = append(enrolledCommands, fmt.Sprintf("`%s` - %s", cmd, info.Description))
		case "Mentor":
			mentorCommands = append(mentorCommands, fmt.Sprintf("`%s` - %s", cmd, info.Description))
		case "Admin":
			adminCommands = append(adminCommands, fmt.Sprintf("`%s` - %s", cmd, info.Description))
		}
	}

	// Basic commands
	sb.WriteString("**General Commands:**\n")
	for _, cmd := range allCommands {
		sb.WriteString("• " + cmd + "\n")
	}

	// Enrolled user commands
	sb.WriteString("\n**Enrolled User Commands:**\n")
	for _, cmd := range enrolledCommands {
		sb.WriteString("• " + cmd + "\n")
	}

	// Mentor commands
	sb.WriteString("\n**Mentor Commands:**\n")
	for _, cmd := range mentorCommands {
		sb.WriteString("• " + cmd + "\n")
	}

	// Admin commands
	sb.WriteString("\n**Admin Commands:**\n")
	for _, cmd := range adminCommands {
		sb.WriteString("• " + cmd + "\n")
	}
	sb.WriteString("\nFor detailed help on a specific command, use `!help <command>`")

	// Extra information
	sb.WriteString("\n\n**📝 Parameter Naming Conventions:**\n")
	sb.WriteString("• `CamelUpperCase` parameters (like `ProblemName`) can include spaces (e.g., \"Problem Name 4\")\n")
	sb.WriteString("• Parameters starting with lowercase (like `class_code`, `user_name`) should be one word\n")

	return sb.String()
}

// buildCommandHelpMessage creates a detailed help message for a specific command
func buildCommandHelpMessage(info commandInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**Command: %s**\n\n", info.Command))
	sb.WriteString(fmt.Sprintf("**Usage:** `%s`\n", info.Usage))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n", info.Description))
	sb.WriteString(fmt.Sprintf("**User Type:** %s\n", info.UserType))

	if info.ExtraInfo != "" {
		sb.WriteString(fmt.Sprintf("\n**Format Notes:** %s\n", info.ExtraInfo))
	}

	return sb.String()
}
