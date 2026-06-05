package workflow

import (
	"strconv"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_HelpCommand_DispatchesEmbedAndSyncsExecutor(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)

	env.sendMessage(t, "123456", "Student One", "!help")

	lastMsg := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	require.NotNil(t, lastMsg.Embed)
	assert.NotEmpty(t, lastMsg.Embed.Title)

	env.assertUserExists(t, "123456", "Student One")
}

func TestWorkflow_ListCommand_UsesContextualClassAndSeededProblem(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	env.seedClassroom(t, testChannelID, "basicos_1")
	problemID := env.seedProblem(t, "Two Sum", "http://example.com/two-sum", classID, "Easy", "Arrays")

	env.sendMessage(t, "123456", "Student One", "!list")

	lastMsg := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.PaginationEmbed, lastMsg.Type)
	assert.Equal(t, testChannelID, lastMsg.ChannelID)
	require.NotEmpty(t, lastMsg.Embeds)

	embed := lastMsg.Embeds[0]
	assert.Equal(t, "List of problems for class basicos_1", embed.Title)
	require.Len(t, embed.Fields, 3)
	assert.Contains(t, embed.Fields[0].Value, strconv.FormatInt(problemID, 10))
	assert.Contains(t, embed.Fields[2].Value, "[Two Sum](http://example.com/two-sum)")
}

func TestWorkflow_MyClassesCommand_UsesRoleAccessMapping(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	env.seedClass(t, "basicos_1", "Basicos 2026")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.seedUser(t, "123456", "Student One")
	env.setRoles("123456", []string{"role_enrolled"})

	env.sendMessage(t, "123456", "Student One", "!my_classes")

	lastMsg := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	require.NotNil(t, lastMsg.Embed)
	assert.Equal(t, "Your classes:", lastMsg.Embed.Title)
	assert.Contains(t, lastMsg.Embed.Description, "`basicos_1`: Basicos 2026")
}
