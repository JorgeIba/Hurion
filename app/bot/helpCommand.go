package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
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
	"!change_username": {
		Command:     "!change_username",
		Usage:       "!change_username <user_name>",
		Description: "Change your current username to a new one",
		UserType:    "All",
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\"). If you don't have a username bound yet, this will bind the new username to your account.",
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
		Usage:       "!add_problem <class_code> <url> [difficulty] [topic] <ProblemName>",
		Description: "Add a new problem to a class with URL, optional difficulty and topic, and name",
		UserType:    "Mentor",
		ExtraInfo: "The `class_code` should be one word, url should have 'http://' prefix (Discord policies I guess), " +
			"`difficulty` can be: Easy, Medium, Hard, or use '-' to skip (No case sensitive). " +
			"`topic` can be any text, one word, use '-' to skip (No case sensitive). " +
			"`ProblemName` can include multiple words.\n" +
			"Examples:\n" +
			"  !add_problem basicos http://example.com - - Two Sum (no difficulty, no topic)\n" +
			"  !add_problem basicos http://example.com Easy - Two Sum (only difficulty)\n" +
			"  !add_problem basicos http://example.com - DP Two Sum (only topic)\n" +
			"  !add_problem basicos http://example.com Medium Graphs Binary Tree Problem (both)",
	},
	"!remove_problem": {
		Command:     "!remove_problem",
		Usage:       "!remove_problem <problem_id>",
		Description: "Remove a problem from a class by its ID",
		UserType:    "Mentor",
	},
	"!update_problem": {
		Command:     "!update_problem",
		Usage:       "!update_problem <problem_id> <class_code> <url> [difficulty] [topic] <ProblemName>",
		Description: "Update a problem's details while preserving its ID for AC records",
		UserType:    "Mentor",
		ExtraInfo: "The `problem_id` and `class_code` should be one word, url should have 'http://' prefix (Discord policies I guess), " +
			"`difficulty` can be: Easy, Medium, Hard, or use '-' to skip (No case sensitive). " +
			"`topic` can be any text, one word, use '-' to skip (No case sensitive). " +
			"`ProblemName` can include multiple words.\n" +
			"Examples:\n" +
			"  !update_problem 3 basicos http://example.com - - Updated Name (no difficulty, no topic)\n" +
			"  !update_problem 3 basicos http://example.com Hard - Updated Name (only difficulty)\n" +
			"  !update_problem 3 basicos http://example.com - DP Updated Name (only topic)\n" +
			"  !update_problem 3 basicos http://example.com Easy Graphs Updated Problem Name (both)",
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
	}, "!exterminate": {
		Command:     "!exterminate",
		Usage:       "!exterminate <user_id>",
		Description: "Remove user from all system data including bindings, enrollments, solved problems and pending confirmations",
		UserType:    "Admin",
		ExtraInfo:   "The `user_id` should be the Discord user ID (e.g., \"123456789012345678\"). This action cannot be undone.",
	}, "!export": {
		Command:     "!export",
		Usage:       "!export <class_code>",
		Description: "Export all class data (users, problems, scores) to an Excel file",
		UserType:    "Admin",
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\"). The Excel file will contain three sheets: Users, Problems, and Scores.",
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
func buildGeneralHelpMessage() *discordgo.MessageEmbed {
	allCommands := []string{}
	enrolledCommands := []string{}
	mentorCommands := []string{}
	adminCommands := []string{}

	for cmd, info := range commandInfoMap {
		switch info.UserType {
		case "All":
			allCommands = append(allCommands, fmt.Sprintf("• `%s` - %s", cmd, info.Description))
		case "Enrolled":
			enrolledCommands = append(enrolledCommands, fmt.Sprintf("• `%s` - %s", cmd, info.Description))
		case "Mentor":
			mentorCommands = append(mentorCommands, fmt.Sprintf("• `%s` - %s", cmd, info.Description))
		case "Admin":
			adminCommands = append(adminCommands, fmt.Sprintf("• `%s` - %s", cmd, info.Description))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📚 Huron Problem Checker Bot - Available Commands",
		Description: "Use `!help <command>` for detailed information about a specific command",
		Color:       0x5865F2, // Discord Blurple
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// General commands field
	if len(allCommands) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🌐 General Commands",
			Value:  strings.Join(allCommands, "\n"),
			Inline: false,
		})
	}

	// Enrolled commands field
	if len(enrolledCommands) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👤 Enrolled User Commands",
			Value:  strings.Join(enrolledCommands, "\n"),
			Inline: false,
		})
	}

	// Mentor commands field
	if len(mentorCommands) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👨‍🏫 Mentor Commands",
			Value:  strings.Join(mentorCommands, "\n"),
			Inline: false,
		})
	}

	// Admin commands field
	if len(adminCommands) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚙️ Admin Commands",
			Value:  strings.Join(adminCommands, "\n"),
			Inline: false,
		})
	}

	// Parameter naming convention footer
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "📝 Parameter Naming Conventions",
		Value: "• `CamelUpperCase` parameters can include spaces (e.g., \"Problem Name 4\")\n" +
			"• lowercase parameters should be one word (e.g., `class_code`, `user_name`)",
		Inline: false,
	})

	return embed
}

// buildCommandHelpMessage creates a detailed help message for a specific command
func buildCommandHelpMessage(info commandInfo) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Command: %s", info.Command),
		Description: info.Description,
		Color:       0x57F287, // Discord Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📋 Usage",
				Value:  fmt.Sprintf("`%s`", info.Usage),
				Inline: false,
			},
			{
				Name:   "👥 User Type",
				Value:  info.UserType,
				Inline: true,
			},
		},
	}

	if info.ExtraInfo != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "ℹ️ Additional Information",
			Value:  info.ExtraInfo,
			Inline: false,
		})
	}

	return embed
}
