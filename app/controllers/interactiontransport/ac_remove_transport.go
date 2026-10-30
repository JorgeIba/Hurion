package interactiontransport

import (
	"errors"
	"strconv"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/bwmarrin/discordgo"
)

// ParseACRemoveOptions reads the problem and optional target user without writing.
// An empty targetUserID means no user was specified.
func ParseACRemoveOptions(data discordgo.ApplicationCommandInteractionData) (problemID int64, targetUserID string, err error) {
	invalid := errors.New("Use `/ac remove` with a positive problem ID and an optional user.")
	if data.Name != discordcommands.ACCommandName ||
		len(data.Options) != 1 ||
		data.Options[0] == nil ||
		data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand ||
		data.Options[0].Name != discordcommands.ACRemoveCommandName {
		return 0, "", invalid
	}

	seen := map[string]bool{}
	for _, option := range data.Options[0].Options {
		if option == nil || seen[option.Name] {
			return 0, "", invalid
		}

		seen[option.Name] = true
		value, ok := option.Value.(string)
		if !ok {
			return 0, "", invalid
		}

		switch option.Name {
		case discordcommands.ACProblemIDOptionName:
			id, err := strconv.ParseInt(value, 10, 64)
			if option.Type != discordgo.ApplicationCommandOptionString || err != nil || id <= 0 {
				return 0, "", invalid
			}

			problemID = id
		case discordcommands.ACUserOptionName:
			id, err := strconv.ParseUint(value, 10, 64)
			if option.Type != discordgo.ApplicationCommandOptionUser || err != nil || id == 0 {
				return 0, "", invalid
			}

			targetUserID = value
		default:
			return 0, "", invalid
		}
	}

	if problemID == 0 {
		return 0, "", invalid
	}

	return problemID, targetUserID, nil
}
