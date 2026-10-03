package discord_test

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestProblemParamsToModalsPreserveRawValuesAndPrefillMissingOptionalValues(t *testing.T) {
	t.Parallel()
	difficulty, topic, empty := " Very hard ", "Binary search", ""
	for _, test := range []struct {
		name       string
		difficulty *string
		topic      *string
		want       discord.ProblemModalFields
	}{
		{
			name: "missing difficulty shows unspecified and missing topic stays blank",
			want: discord.ProblemModalFields{Difficulty: service_model.DifficultyDefault},
		},
		{
			name: "explicit empty values stay empty", difficulty: &empty, topic: &empty,
		},
		{
			name: "raw difficulty and multiword topic are preserved", difficulty: &difficulty, topic: &topic,
			want: discord.ProblemModalFields{Difficulty: " Very hard ", Topic: "Binary search"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			want := test.want
			want.Name, want.URL = "Two Sum con números", "https://example.com/two-sum"
			t.Run("create request prefills create form", func(t *testing.T) {
				params := service_model.CreateProblemParams{
					ClassCode: "basicos_1", Name: want.Name, URL: want.URL,
					Difficulty: test.difficulty, Topic: test.topic,
				}
				assert.Equal(t, discord.CreateProblemModal{ClassCode: "basicos_1", ProblemModalFields: want}, discord.MapCreateProblemParamsToCreateProblemModal(params))
			})
			t.Run("update request prefills edit form and display ID", func(t *testing.T) {
				params := service_model.UpdateProblemParams{
					ProblemID: 42, ClassID: 7, Name: want.Name, URL: want.URL,
					Difficulty: test.difficulty, Topic: test.topic,
				}
				assert.Equal(t, discord.EditProblemModal{ProblemID: 42, ClassCode: "basicos_1", ProblemModalFields: want}, discord.MapUpdateProblemParamsToEditProblemModal(params, "basicos_1"))
			})
		})
	}
}

func TestProblemModalsToParamsClearOptionalValuesAndPreserveOtherRawText(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		difficulty     string
		topic          string
		wantDifficulty *string
		wantTopic      *string
	}{
		{name: "empty optional fields clear stored values"},
		{name: "whitespace-only optional fields clear stored values", difficulty: " \t ", topic: " \n "},
		{name: "unspecified difficulty clears stored value", difficulty: "Not specified"},
		{name: "unspecified marker ignores case and surrounding whitespace", difficulty: " nOt SpEcIfIeD "},
		{
			name: "difficulty and topic remain raw for application normalization", difficulty: " mEDium ", topic: " Binary search ",
			wantDifficulty: stringPointerForProblemModalTest(" mEDium "), wantTopic: stringPointerForProblemModalTest(" Binary search "),
		},
		{
			name: "invalid difficulty remains available for application validation", difficulty: "Insane", topic: "Not specified",
			wantDifficulty: stringPointerForProblemModalTest("Insane"), wantTopic: stringPointerForProblemModalTest("Not specified"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fields := discord.ProblemModalFields{
				Name: " Two Sum con números ", URL: " fix this URL ", Difficulty: test.difficulty, Topic: test.topic,
			}
			t.Run("create form becomes a pending create request", func(t *testing.T) {
				modal := discord.CreateProblemModal{ClassCode: " basicos_2 ", ProblemModalFields: fields}
				assert.Equal(t, service_model.CreateProblemParams{
					Name: fields.Name, URL: fields.URL, ClassCode: " basicos_2 ",
					Difficulty: test.wantDifficulty, Topic: test.wantTopic,
				}, discord.MapCreateProblemModalToCreateProblemParams(modal))
				assert.Equal(t, fields, modal.ProblemModalFields, "mapping leaves correction text unchanged")
			})
			t.Run("edit form uses session target and resolved destination class", func(t *testing.T) {
				modal := discord.EditProblemModal{ProblemID: 999, ClassCode: " basicos_2 ", ProblemModalFields: fields}
				assert.Equal(t, service_model.UpdateProblemParams{
					ProblemID: 42, ClassID: 9, Name: fields.Name, URL: fields.URL,
					Difficulty: test.wantDifficulty, Topic: test.wantTopic,
				}, discord.MapEditProblemModalToUpdateProblemParams(modal, 42, 9))
				assert.Equal(t, int64(999), modal.ProblemID, "display metadata is neither trusted nor modified")
				assert.Equal(t, fields, modal.ProblemModalFields, "mapping leaves correction text unchanged")
			})
		})
	}
}

func stringPointerForProblemModalTest(value string) *string {
	return &value
}
