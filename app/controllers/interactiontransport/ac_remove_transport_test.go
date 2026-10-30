package interactiontransport

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func TestParseACRemoveOptions(t *testing.T) {
	for _, value := range []any{"", "Two Sum", "0", "-1", "1.5", "9223372036854775808", float64(5)} {
		t.Run("invalid problem", func(t *testing.T) {
			command := discordgo.ApplicationCommandInteractionData{
				Name: "ac",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{{
					Name: "remove",
					Type: discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandInteractionDataOption{{
						Name:  "id",
						Type:  discordgo.ApplicationCommandOptionString,
						Value: value,
					}},
				}},
			}
			_, _, err := ParseACRemoveOptions(command)
			require.Error(t, err)
		})
	}
	for _, value := range []any{"", "not-a-user", "0", "-1", float64(123)} {
		t.Run("invalid user", func(t *testing.T) {
			command := discordgo.ApplicationCommandInteractionData{
				Name: "ac",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{{
					Name: "remove",
					Type: discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{
							Name:  "id",
							Type:  discordgo.ApplicationCommandOptionString,
							Value: "5",
						},
						{
							Name:  "user",
							Type:  discordgo.ApplicationCommandOptionUser,
							Value: value,
						},
					},
				}},
			}
			_, _, err := ParseACRemoveOptions(command)
			require.Error(t, err)
		})
	}
}
