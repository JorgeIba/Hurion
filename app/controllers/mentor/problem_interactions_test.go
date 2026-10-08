package mentor

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProblemCreateOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		options   []*discordgo.ApplicationCommandInteractionDataOption
		want      string
		wantError bool
	}{
		{name: "omitted class allows channel default"},
		{name: "explicit class is trimmed", options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "class", Value: " basicos "}}, want: "basicos"},
		{name: "blank class allows channel default", options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "class", Value: " "}}},
		{name: "nontext class is rejected", options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "class", Value: 123}}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := discordgo.ApplicationCommandInteractionData{Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add", Options: test.options}}}
			got, err := parseProblemCreateOptions(command)
			assert.Equal(t, test.want, got)
			if test.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
	_, err := parseProblemCreateOptions(discordgo.ApplicationCommandInteractionData{})
	assert.Error(t, err)
}

func TestParseProblemEditOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		problemID any
		classCode any
		wantID    int64
		wantClass string
		wantError bool
	}{
		{name: "omitted class allows saved class default", problemID: "123", wantID: 123},
		{name: "explicit values are trimmed", problemID: " 123 ", classCode: " basicos ", wantID: 123, wantClass: "basicos"},
		{name: "blank class allows saved class default", problemID: "123", classCode: " ", wantID: 123},
		{name: "largest int64 ID", problemID: "9223372036854775807", wantID: 9223372036854775807},
		{name: "missing ID", wantError: true},
		{name: "blank ID", problemID: " ", wantError: true},
		{name: "zero ID", problemID: "0", wantError: true},
		{name: "negative ID", problemID: "-1", wantError: true},
		{name: "noninteger ID", problemID: "1.5", wantError: true},
		{name: "overflow ID", problemID: "9223372036854775808", wantError: true},
		{name: "nontext ID", problemID: 123, wantError: true},
		{name: "nontext class", problemID: "123", classCode: 456, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var options []*discordgo.ApplicationCommandInteractionDataOption
			if test.problemID != nil {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "problem", Value: test.problemID})
			}
			if test.classCode != nil {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Value: test.classCode})
			}
			command := discordgo.ApplicationCommandInteractionData{Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "edit", Options: options}}}
			id, classCode, err := parseProblemEditOptions(command)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.wantID, id)
			assert.Equal(t, test.wantClass, classCode)
		})
	}
	for _, options := range [][]*discordgo.ApplicationCommandInteractionDataOption{
		nil,
		{nil},
		{{Name: "edit"}, {Name: "add"}},
	} {
		_, _, err := parseProblemEditOptions(discordgo.ApplicationCommandInteractionData{Options: options})
		assert.Error(t, err)
	}
}
