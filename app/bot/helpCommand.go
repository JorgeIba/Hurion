package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func getCommandDataByName(name string) (CommandData, bool) {
	if cmd, ok := CommandByName(name); ok {
		return cmd.Data(), true
	}
	return CommandData{}, false
}

func buildGeneralHelpMessage() *discordgo.MessageEmbed {
	allCommands := []string{}
	enrolledCommands := []string{}
	mentorCommands := []string{}
	adminCommands := []string{}

	for _, data := range Commands {
		cmdEntry := fmt.Sprintf("• `%s` - %s", data.Cmd, data.Description)

		if data.UserRoles.Contains(RoleAll) {
			allCommands = append(allCommands, cmdEntry)
		}
		if data.UserRoles.Contains(RoleEnrolled) {
			enrolledCommands = append(enrolledCommands, cmdEntry)
		}
		if data.UserRoles.Contains(RoleMentor) {
			mentorCommands = append(mentorCommands, cmdEntry)
		}
		if data.UserRoles.Contains(RoleAdmin) {
			adminCommands = append(adminCommands, cmdEntry)
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

func buildCommandHelpMessage(data CommandData) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Command: %s", data.Cmd),
		Description: data.Description,
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📋 Usage",
				Value:  fmt.Sprintf("`%s`", data.Usage),
				Inline: false,
			},
			{
				Name:   "👥 User Type",
				Value:  data.UserRoles.String(),
				Inline: true,
			},
		},
	}

	if data.ExtraInfo != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "ℹ️ Additional Information",
			Value:  data.ExtraInfo,
			Inline: false,
		})
	}

	return embed
}
