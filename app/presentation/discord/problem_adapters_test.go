package discord_test

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestProblemParamsToModalsPreserveRawValuesAndPrefillMissingOptionalValues(t *testing.T) {
	t.Parallel()
	difficulty, topic, priority, empty := " Very hard ", "Binary search", " Custom priority ", ""
	for _, test := range []struct {
		name       string
		difficulty *string
		topic      *string
		priority   *string
		want       discord.ProblemModalFields
	}{
		{
			name: "missing metadata shows unassigned labels and missing topic stays blank",
			want: discord.ProblemModalFields{Difficulty: presentation.DifficultyUnspecifiedLabel, Priority: presentation.PriorityUnspecifiedLabel},
		},
		{
			name: "explicit empty values stay empty", difficulty: &empty, topic: &empty, priority: &empty,
		},
		{
			name: "unknown metadata and multiword topic are preserved", difficulty: &difficulty, topic: &topic, priority: &priority,
			want: discord.ProblemModalFields{Difficulty: " Very hard ", Topic: "Binary search", Priority: " Custom priority "},
		},
		{
			name: "canonical metadata codes use display labels", difficulty: new(service_model.DifficultyMedium), priority: new(service_model.PriorityNotRequired),
			want: discord.ProblemModalFields{Difficulty: "Medium", Priority: "Not required"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			want := test.want
			want.Name, want.URL = "Two Sum con números", "https://example.com/two-sum"
			t.Run("create request prefills create form", func(t *testing.T) {
				params := service_model.CreateProblemParams{
					ClassCode: "basicos_1", Name: want.Name, URL: want.URL,
					Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
				}
				assert.Equal(t, discord.CreateProblemModal{ClassCode: "basicos_1", ProblemModalFields: want}, discord.MapCreateProblemParamsToCreateProblemModal(params))
			})
			t.Run("update request prefills edit form and display ID", func(t *testing.T) {
				params := service_model.UpdateProblemParams{
					ProblemID: 42, ClassID: 7, Name: want.Name, URL: want.URL,
					Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
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
		priority       string
		wantDifficulty *string
		wantTopic      *string
		wantPriority   *string
	}{
		{name: "empty optional fields clear stored values"},
		{name: "whitespace-only optional fields clear stored values", difficulty: " \t ", topic: " \n ", priority: " \t "},
		{name: "unspecified difficulty clears stored value", difficulty: "Not specified"},
		{name: "unspecified marker ignores case and surrounding whitespace", difficulty: " nOt SpEcIfIeD "},
		{name: "unassigned priority clears stored value", priority: presentation.PriorityUnspecifiedLabel},
		{name: "unassigned priority marker ignores case and surrounding whitespace", priority: " nO PrIoRiTy AsSiGnEd "},
		{
			name: "metadata and topic remain raw for application normalization", difficulty: " mEDium ", topic: " Binary search ", priority: " Recommended ",
			wantDifficulty: stringPointerForProblemModalTest(" mEDium "), wantTopic: stringPointerForProblemModalTest(" Binary search "),
			wantPriority: stringPointerForProblemModalTest(" Recommended "),
		},
		{
			name: "invalid metadata remains available for application validation", difficulty: "Insane", topic: "Not specified", priority: "Urgent",
			wantDifficulty: stringPointerForProblemModalTest("Insane"), wantTopic: stringPointerForProblemModalTest("Not specified"),
			wantPriority: stringPointerForProblemModalTest("Urgent"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fields := discord.ProblemModalFields{
				Name: " Two Sum con números ", URL: " fix this URL ", Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
			}
			t.Run("create form becomes a pending create request", func(t *testing.T) {
				modal := discord.CreateProblemModal{ClassCode: " basicos_2 ", ProblemModalFields: fields}
				assert.Equal(t, service_model.CreateProblemParams{
					Name: fields.Name, URL: fields.URL, ClassCode: " basicos_2 ",
					Difficulty: test.wantDifficulty, Topic: test.wantTopic, Priority: test.wantPriority,
				}, discord.MapCreateProblemModalToCreateProblemParams(modal))
				assert.Equal(t, fields, modal.ProblemModalFields, "mapping leaves correction text unchanged")
			})
			t.Run("edit form uses session target and resolved destination class", func(t *testing.T) {
				modal := discord.EditProblemModal{ProblemID: 999, ClassCode: " basicos_2 ", ProblemModalFields: fields}
				assert.Equal(t, service_model.UpdateProblemParams{
					ProblemID: 42, ClassID: 9, Name: fields.Name, URL: fields.URL,
					Difficulty: test.wantDifficulty, Topic: test.wantTopic, Priority: test.wantPriority,
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
