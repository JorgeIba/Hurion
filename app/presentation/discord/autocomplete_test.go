package discord

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblemAutocompleteChoicesFilterBeforeLimiting(t *testing.T) {
	t.Parallel()
	catalog := make([]service_model.Problem, 40)
	for i := range catalog {
		catalog[i] = service_model.Problem{ID: int64(i + 1), Name: "Practice"}
	}
	catalog[1].Name = "Two Sum"
	catalog[29].Name = "Sum Challenge"
	catalog[39].Name = "Needle"
	first25 := make([]string, 25)
	for i := range first25 {
		first25[i] = strconv.Itoa(i + 1)
	}

	for _, test := range []struct {
		name    string
		query   string
		wantIDs []string
	}{
		{name: "empty input shows first 25 IDs", wantIDs: first25},
		{name: "ID prefix", query: "1", wantIDs: []string{"1", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19"}},
		{name: "name ignores case and surrounding spaces", query: "  SUM ", wantIDs: []string{"2", "30"}},
		{name: "match beyond first 25 catalog entries", query: "needle", wantIDs: []string{"40"}},
		{name: "no matches", query: "not present", wantIDs: []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			before := append([]service_model.Problem(nil), catalog...)
			choices := ProblemAutocompleteChoices(catalog, test.query)
			ids := make([]string, 0, len(choices))
			for _, choice := range choices {
				id, ok := choice.Value.(string)
				require.True(t, ok, "the submitted value is a string ID, not the label")
				ids = append(ids, id)
				problemID, err := strconv.Atoi(id)
				require.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("%s: %s", id, catalog[problemID-1].Name), choice.Name)
			}
			assert.Equal(t, test.wantIDs, ids)
			assert.Equal(t, before, catalog, "presentation must not change the catalog")
		})
	}
}

func TestProblemAutocompleteLabelLimitPreservesID(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		problemName string
		wantLabel   string
	}{
		{name: "short label", problemName: "Tree Sum", wantLabel: "5: Tree Sum"},
		{name: "long ASCII label", problemName: strings.Repeat("A", 200), wantLabel: "5: " + strings.Repeat("A", 97)},
		{name: "long Unicode label", problemName: strings.Repeat("🚀", 100)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			choices := ProblemAutocompleteChoices([]service_model.Problem{{ID: 5, Name: test.problemName}}, "5")
			require.Len(t, choices, 1)
			assert.Equal(t, "5", choices[0].Value)
			assert.True(t, utf8.ValidString(choices[0].Name))
			assert.LessOrEqual(t, len(utf16.Encode([]rune(choices[0].Name))), 100)
			assert.True(t, strings.HasPrefix(choices[0].Name, "5: "))
			if test.wantLabel != "" {
				assert.Equal(t, test.wantLabel, choices[0].Name)
			}
		})
	}
}

func TestBuildAutocompleteResponse(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		choices []*discordgo.ApplicationCommandOptionChoice
	}{
		{name: "nil suggestions"},
		{name: "empty suggestions", choices: []*discordgo.ApplicationCommandOptionChoice{}},
		{name: "suggestions", choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "5: Tree Sum", Value: "5"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := BuildAutocompleteResponse(test.choices)
			assert.Equal(t, discordgo.InteractionApplicationCommandAutocompleteResult, response.Type)
			require.NotNil(t, response.Data)
			assert.NotNil(t, response.Data.Choices)
			assert.Len(t, response.Data.Choices, len(test.choices))
			assert.Empty(t, response.Data.Content)
			if len(test.choices) > 0 {
				assert.Equal(t, test.choices, response.Data.Choices)
			}
		})
	}
}
