package workflow

import (
	"strconv"
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_ACProposal_CreatesPendingConfirmationAndRequestsMentorDecision(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.setRoles("student_1", []string{"role_enrolled"})

	env.sendMessage(t, "student_1", "Student One", "!ac "+strconv.FormatInt(problemID, 10))

	env.assertPendingConfirmation(t, "msg_student_1", "student_1", problemID)
	env.assertUserExists(t, "student_1", "Student One")

	last := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.Reaction, last.Type)
	assert.Equal(t, testChannelID, last.ChannelID)
	assert.Equal(t, "msg_student_1", last.ReplyTo)
	assert.ElementsMatch(t, []string{common.REACTION_YES, common.REACTION_NO}, last.Reactions)
	assert.Empty(t, last.ClearSpecificReactions, "proposal should request a mentor decision, not clear reactions yet")
}

func TestWorkflow_ACProposal_MentorApproval_GrantsProblemAndMarksProposalAccepted(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.setRoles("student_1", []string{"role_enrolled"})
	env.Discord.isMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == "mentor_1", nil
	}

	env.sendMessage(t, "student_1", "Student One", "!ac "+strconv.FormatInt(problemID, 10))
	beforeReactionMessages := len(env.Dispatcher.Messages())

	env.sendReaction(t, "mentor_1", "Mentor One", "msg_student_1", common.REACTION_YES)

	env.assertPendingConfirmationRemoved(t, "msg_student_1")
	env.assertProblemSolved(t, "student_1", problemID)
	env.assertUserExists(t, "mentor_1", "Mentor One")

	messages := env.Dispatcher.Messages()
	require.Len(t, messages, beforeReactionMessages+1)

	last := messages[len(messages)-1]
	assert.Equal(t, dispatcher.Reaction, last.Type)
	assert.Equal(t, "msg_student_1", last.ReplyTo)
	assert.ElementsMatch(t, []string{common.REACTION_YES, common.REACTION_NO}, last.ClearSpecificReactions)
	assert.Equal(t, []string{common.REACTION_ACCEPTED}, last.Reactions)
}

func TestWorkflow_ACProposal_MentorRejection_RemovesPendingWithoutGrantingProblem(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.setRoles("student_1", []string{"role_enrolled"})
	env.Discord.isMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == "mentor_1", nil
	}

	env.sendMessage(t, "student_1", "Student One", "!ac "+strconv.FormatInt(problemID, 10))
	beforeReactionMessages := len(env.Dispatcher.Messages())

	env.sendReaction(t, "mentor_1", "Mentor One", "msg_student_1", common.REACTION_NO)

	env.assertPendingConfirmationRemoved(t, "msg_student_1")
	env.assertProblemNotSolved(t, "student_1", problemID)

	messages := env.Dispatcher.Messages()
	require.Len(t, messages, beforeReactionMessages+1)

	last := messages[len(messages)-1]
	assert.Equal(t, dispatcher.Reaction, last.Type)
	assert.Equal(t, "msg_student_1", last.ReplyTo)
	assert.ElementsMatch(t, []string{common.REACTION_YES, common.REACTION_NO}, last.ClearSpecificReactions)
	assert.Equal(t, []string{common.REACTION_REJECTED}, last.Reactions)
}

func TestWorkflow_ACProposal_NonMentorReaction_IsIgnoredAndLeavesProposalPending(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.setRoles("student_1", []string{"role_enrolled"})
	env.Discord.isMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == "mentor_1", nil
	}

	env.sendMessage(t, "student_1", "Student One", "!ac "+strconv.FormatInt(problemID, 10))
	beforeReactionMessages := len(env.Dispatcher.Messages())

	env.sendReaction(t, "random_1", "Random User", "msg_student_1", common.REACTION_YES)

	env.assertPendingConfirmation(t, "msg_student_1", "student_1", problemID)
	env.assertProblemNotSolved(t, "student_1", problemID)
	env.assertUserExists(t, "random_1", "Random User")
	assert.Len(t, env.Dispatcher.Messages(), beforeReactionMessages, "non-mentor reactions should not dispatch an approval or rejection update")
}

func TestWorkflow_UndoAC_RemovesSolvedProblemAndReportsSuccess(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedUser(t, "student_1", "Student One")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.setRoles("student_1", []string{"role_enrolled"})
	require.NoError(t, env.DB.AddAC("student_1", problemID))

	env.sendMessage(t, "student_1", "Student One", "!undo_ac "+strconv.FormatInt(problemID, 10))

	env.assertProblemNotSolved(t, "student_1", problemID)

	last := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.SimpleMessage, last.Type)
	assert.Equal(t, testChannelID, last.ChannelID)
	assert.Contains(t, last.Content, "UnAC-ed successfully")
	assert.Contains(t, last.Content, "Student One")
	assert.Contains(t, last.Content, strconv.FormatInt(problemID, 10))
}

func TestWorkflow_MentorDirectAC_GrantsProblemAndAcknowledgesWithAcceptedReaction(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.Discord.setDisplayName("111222333444555666", "Student One")
	env.setRoles("111222333444555666", []string{"role_enrolled"})
	env.Discord.isMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == "mentor_1", nil
	}

	env.sendMessage(t, "mentor_1", "Mentor One", "!ac <@111222333444555666> "+strconv.FormatInt(problemID, 10))

	env.assertProblemSolved(t, "111222333444555666", problemID)
	env.assertUserExists(t, "111222333444555666", "Student One")

	last := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.Reaction, last.Type)
	assert.Equal(t, "msg_mentor_1", last.ReplyTo)
	assert.Equal(t, []string{common.REACTION_ACCEPTED}, last.Reactions)
	assert.Empty(t, last.ClearSpecificReactions)
}

func TestWorkflow_MentorDirectUndoAC_RemovesSolvedProblemAndMentionsTargetUser(t *testing.T) {
	t.Parallel()

	env := setupWorkflowEnv(t)
	classID := env.seedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.seedProblem(t, "Two Sum", "https://example.com/two-sum", classID, "Easy", "Arrays")
	env.seedUser(t, "111222333444555666", "Student One")
	require.NoError(t, env.DB.AddAC("111222333444555666", problemID))
	env.Discord.setDisplayName("111222333444555666", "Student One")
	env.Discord.isMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == "mentor_1", nil
	}

	env.sendMessage(t, "mentor_1", "Mentor One", "!undo_ac <@111222333444555666> "+strconv.FormatInt(problemID, 10))

	env.assertProblemNotSolved(t, "111222333444555666", problemID)

	last := env.Dispatcher.LastMessage(t)
	assert.Equal(t, dispatcher.SimpleMessage, last.Type)
	assert.Contains(t, last.Content, "UnAC-ed successfully")
	assert.Contains(t, last.Content, "<@111222333444555666>")
	assert.Contains(t, last.Content, strconv.FormatInt(problemID, 10))
}
