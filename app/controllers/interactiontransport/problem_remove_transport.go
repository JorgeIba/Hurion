package interactiontransport

import (
	"errors"
	"strconv"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/bwmarrin/discordgo"
)

// ParseProblemRemoveOptions reads the selected problem without performing any writes.
func ParseProblemRemoveOptions(data discordgo.ApplicationCommandInteractionData) (int64, error) {
	invalid := errors.New("Use /problem remove problem:<positive ID>.")
	if data.Name != discordcommands.ProblemCommandName ||
		len(data.Options) != 1 ||
		data.Options[0] == nil ||
		data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand ||
		data.Options[0].Name != discordcommands.ProblemRemoveCommandName {
		return 0, invalid
	}

	options := data.Options[0].Options
	if len(options) != 1 ||
		options[0] == nil ||
		options[0].Name != discordcommands.ProblemIDOptionName ||
		options[0].Type != discordgo.ApplicationCommandOptionString {
		return 0, invalid
	}

	value, ok := options[0].Value.(string)
	if !ok {
		return 0, invalid
	}

	problemID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || problemID <= 0 {
		return 0, invalid
	}

	return problemID, nil
}
