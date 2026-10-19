package discord

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	maxAutocompleteChoices    = 25
	maxAutocompleteNameLength = 100
	problemAutocompleteLabel  = "%s: %s"
)

// ProblemAutocompleteChoices filters an ID-ordered class catalog without changing it.
func ProblemAutocompleteChoices(problems []service_model.Problem, query string) []*discordgo.ApplicationCommandOptionChoice {
	query = strings.ToLower(strings.TrimSpace(query))

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, min(len(problems), maxAutocompleteChoices))
	for _, problem := range problems {
		id := strconv.FormatInt(problem.ID, 10)
		if !strings.HasPrefix(id, query) && !strings.Contains(strings.ToLower(problem.Name), query) {
			continue
		}

		label := fmt.Sprintf(problemAutocompleteLabel, id, problem.Name)

		units := utf16.Encode([]rune(label))
		if len(units) > maxAutocompleteNameLength {
			label = string(utf16.Decode(units[:maxAutocompleteNameLength]))
		}

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  label,
			Value: id,
		})

		if len(choices) == maxAutocompleteChoices {
			break
		}
	}
	return choices
}

// BuildAutocompleteResponse also represents unavailable suggestions as an empty list.
func BuildAutocompleteResponse(choices []*discordgo.ApplicationCommandOptionChoice) *discordgo.InteractionResponse {
	if choices == nil {
		choices = []*discordgo.ApplicationCommandOptionChoice{}
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	}
}
