package interactiontransport

import (
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblemModalIDsRoundTripEncodedContext(t *testing.T) {
	class := "basicos|á%"
	create, err := CreateProblemModalID(class)
	require.NoError(t, err)
	assert.Equal(t, discordcommands.CreateProblem, ModalAction(create))
	decoded, err := ParseCreateProblemModalID(create)
	require.NoError(t, err)
	assert.Equal(t, class, decoded)
	edit, err := EditProblemModalID(class, 123)
	require.NoError(t, err)
	assert.Equal(t, discordcommands.EditProblem, ModalAction(edit))
	decoded, problemID, err := ParseEditProblemModalID(edit)
	require.NoError(t, err)
	assert.Equal(t, class, decoded)
	assert.EqualValues(t, 123, problemID)
	_, err = ParseCreateProblemModalID(edit)
	assert.Error(t, err)
}

func TestActionsRecognizeMalformedContextWithoutAcceptingIt(t *testing.T) {
	assert.Equal(t, discordcommands.EditProblem, ModalAction("huron:problem:edit|%GG|abc"))
	_, _, err := ParseEditProblemModalID("huron:problem:edit|%GG|abc")
	assert.Error(t, err)
	assert.Equal(t, discordcommands.CancelACBatch, ComponentAction("huron:ac:batch:cancel|"))
	_, err = ParseACBatchCancelID("huron:ac:batch:cancel|")
	assert.Error(t, err)
	assert.Empty(t, ModalAction("huron:problem:edit-other|123"))
	assert.Empty(t, ModalAction(ACBatchCancelID("id")))
	assert.Empty(t, ComponentAction(ACBatchModalID()))
	assert.NoError(t, ValidateACBatchModalID(ACBatchModalID()))
	assert.Error(t, ValidateACBatchModalID(ProblemBatchModalID()))
	assert.Error(t, ValidateACBatchModalID(ACBatchModalID()+"|extra"))
}

func TestCancelIDsKeepFamiliesAndBatchContextDistinct(t *testing.T) {
	for _, test := range []struct {
		name   string
		build  func(string) string
		parse  func(string) (string, error)
		other  func(string) (string, error)
		action discordcommands.Action
	}{
		{"problem", ProblemBatchCancelID, ParseProblemBatchCancelID, ParseACBatchCancelID, discordcommands.CancelProblemBatch},
		{"AC", ACBatchCancelID, ParseACBatchCancelID, ParseProblemBatchCancelID, discordcommands.CancelACBatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := test.build("batch123")
			assert.Equal(t, test.action, ComponentAction(id))
			batch, err := test.parse(id)
			require.NoError(t, err)
			assert.Equal(t, "batch123", batch)
			_, err = test.other(id)
			assert.Error(t, err)
			for _, bad := range []string{test.build(""), test.build(" "), test.build("id|extra")} {
				_, err = test.parse(bad)
				assert.Error(t, err)
			}
		})
	}
}

func TestProblemIDsEnforceContextAndWireLength(t *testing.T) {
	capacity := 100 - len(string(discordcommands.CreateProblem)) - 1
	id, err := CreateProblemModalID(strings.Repeat("a", capacity))
	require.NoError(t, err)
	assert.Len(t, id, 100)
	_, err = CreateProblemModalID(strings.Repeat("a", capacity+1))
	assert.Error(t, err)
	_, err = CreateProblemModalID(" ")
	assert.Error(t, err)
	_, err = EditProblemModalID("class", 0)
	assert.Error(t, err)
}
