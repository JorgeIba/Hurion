package interactiontransport

import "github.com/bwmarrin/discordgo"

// FocusedAutocompleteOption finds the field being edited, including nested subcommands.
// Autocomplete payloads can omit other required options while the user is typing.
func FocusedAutocompleteOption(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.ApplicationCommandInteractionDataOption {
	for _, option := range options {
		if option == nil {
			continue
		}

		if option.Focused {
			return option
		}

		if focused := FocusedAutocompleteOption(option.Options); focused != nil {
			return focused
		}
	}

	return nil
}
