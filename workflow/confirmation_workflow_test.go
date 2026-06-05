package workflow

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_ConfirmationReaction_CancelUpdatesConfirmationMessage(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	confirmation := newConfirmationMessage(
		"confirmation_1",
		"original_1",
		[]string{"!clear_class", "basicos_1"},
		"123456",
	)
	env.Discord.storeMessage(confirmation)

	env.sendReaction(t, "123456", "Student One", "confirmation_1", common.REACTION_THUMBS_DOWN)

	updated, err := env.Discord.FetchMessage(testChannelID, "confirmation_1")
	require.NoError(t, err)
	require.NotEmpty(t, updated.Embeds)
	assert.Equal(t, "CANCELLED", updated.Embeds[0].Title)
	assert.Contains(t, updated.Embeds[0].Description, "Student One")
	assert.ElementsMatch(t, []string{common.REACTION_THUMBS_UP, common.REACTION_THUMBS_DOWN}, env.Discord.removedReactionEmojis())
}
