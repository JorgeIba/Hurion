package discord_test

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestProblemModalAdaptersPreserveOptionalValuesInBothDirections(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                        string
		difficulty, topic, priority *string
	}{
		{name: "missing optional values remain nil"},
		{name: "canonical codes remain codes", difficulty: new(service_model.DifficultyMedium), topic: new("Binary search"), priority: new(service_model.PriorityNotRequired)},
		{name: "explicit empty values remain unchanged", difficulty: new(""), topic: new(""), priority: new("")},
		{name: "raw values remain available for service validation", difficulty: new(" Very hard "), topic: new(" Binary search "), priority: new(" Custom priority ")},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fields := discord.ProblemModalFields{
				Name: " Two Sum con números ", URL: " fix this URL ",
				Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
			}
			t.Run("create request and form", func(t *testing.T) {
				params := service_model.CreateProblemParams{
					ClassCode: "basicos_1", Name: fields.Name, URL: fields.URL,
					Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
				}
				modal := discord.MapCreateProblemParamsToCreateProblemModal(params)
				assert.Equal(t, discord.CreateProblemModal{ClassCode: "basicos_1", ProblemModalFields: fields}, modal)
				assert.Equal(t, params, discord.MapCreateProblemModalToCreateProblemParams(modal))
			})
			t.Run("update request and form", func(t *testing.T) {
				params := service_model.UpdateProblemParams{
					ProblemID: 42, ClassCode: "basicos_1", Name: fields.Name, URL: fields.URL,
					Difficulty: test.difficulty, Topic: test.topic, Priority: test.priority,
				}
				modal := discord.MapUpdateProblemParamsToEditProblemModal(params)
				assert.Equal(t, discord.EditProblemModal{ProblemID: 42, ClassCode: "basicos_1", ProblemModalFields: fields}, modal)
				assert.Equal(t, params, discord.MapEditProblemModalToUpdateProblemParams(modal))
			})
		})
	}
}
