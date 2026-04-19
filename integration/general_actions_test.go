package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_General_GetScoreByClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, classCode)
	env.SeedAC(t, student.UserID, problemID)

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!score", classCode})

	// 2. ACT
	general.HandleGetScoreByClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.NotEmpty(t, lastMsg.Files)
	assert.Equal(t, "table.png", lastMsg.Files[0].Name)
}

func TestIntegration_General_GetScoreByUserAndClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	p1 := env.SeedProblem(t, "P1", "url", classID, "Easy", "Arrays")
	p2 := env.SeedProblem(t, "P2", "url", classID, "Easy", "Arrays")

	// Seed 7 users to test neighborhood (2 before, 2 after + target = 5 total)
	// Rankings (by CountProblems desc, then UserID asc):
	// 1. U1 (2 AC) - UserID 1
	// 2. U2 (1 AC) - UserID 2
	// 3. U3 (1 AC) - UserID 3
	// 4. U4 (1 AC) - UserID 4  <-- TARGET (Middle)
	// 5. U5 (0 AC) - UserID 5
	// 6. U6 (0 AC) - UserID 6
	// 7. U7 (0 AC) - UserID 7

	userIDs := []string{"111", "222", "333", "444", "555", "666", "777"}
	for i, id := range userIDs {
		env.SeedUser(t, id, fmt.Sprintf("User %d", i+1))
		env.SeedEnrollment(t, id, classCode)
	}

	// Assign ACs
	targetUserID := "444"
	targetDisplayName := "User 4"

	env.SeedAC(t, "111", p1)
	env.SeedAC(t, "111", p2)
	env.SeedAC(t, "222", p1)
	env.SeedAC(t, "333", p1)
	env.SeedAC(t, targetUserID, p1) // Target has 1 AC

	// Stub fetcher for all users to resolve names in table
	env.Fetcher.GetCachedMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
		for i, id := range userIDs {
			if id == userID {
				name := fmt.Sprintf("User %d", i+1)
				return &discordgo.Member{
					User: &discordgo.User{ID: id, Username: name},
					Nick: name,
				}, nil
			}
		}
		return nil, fmt.Errorf("not found")
	}

	ctx := env.NewContext("999", "Exec", "guild_1", []string{"!score_user", classCode, fmt.Sprintf("<@%s>", targetUserID)})

	// 2. ACT
	general.HandleGetScoreByUserAndClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embeds[0].Title, targetDisplayName)

	// Since the controller renders the score to a PNG image, we verify the logical
	// correctness of the neighborhood and sorting by querying the service directly.
	score, err := env.Services.Score.GetNeighborsScoreForUserID(targetUserID, classCode)
	assert.NoError(t, err)

	// We expect 5 users in the neighborhood (2 before, 2 after + target)
	assert.Len(t, score.UserScores, 5)
	assert.Equal(t, "222", score.UserScores[0].User.UserID)
	assert.Equal(t, "333", score.UserScores[1].User.UserID)
	assert.Equal(t, "444", score.UserScores[2].User.UserID) // Target
	assert.Equal(t, "555", score.UserScores[3].User.UserID)
	assert.Equal(t, "666", score.UserScores[4].User.UserID)

	// Verify deterministic sorting: CountProblems desc, then UserID asc
	for i := 0; i < len(score.UserScores)-1; i++ {
		curr, next := score.UserScores[i], score.UserScores[i+1]
		if curr.CountProblems == next.CountProblems {
			assert.Less(t, curr.User.UserID, next.User.UserID, "Deterministic sort failed for UserIDs")
		} else {
			assert.Greater(t, curr.CountProblems, next.CountProblems, "Score sort failed")
		}
	}
}

func TestIntegration_General_GetUsersInClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	student1 := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	student2 := service_model.User{UserID: "222333444", DisplayName: "Student Two"}
	env.SeedUser(t, student1.UserID, student1.DisplayName)
	env.SeedUser(t, student2.UserID, student2.DisplayName)
	env.SeedEnrollment(t, student1.UserID, classCode)
	env.SeedEnrollment(t, student2.UserID, classCode)

	// Stub fetcher
	env.Fetcher.GetCachedMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
		if userID == student1.UserID {
			return &discordgo.Member{User: &discordgo.User{ID: student1.UserID}, Nick: student1.DisplayName}, nil
		}
		return &discordgo.Member{User: &discordgo.User{ID: student2.UserID}, Nick: student2.DisplayName}, nil
	}

	ctx := env.NewContext("555666777", "User One", "guild_1", []string{"!users", classCode})

	// 2. ACT
	general.HandleGetUsersInClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Description, student1.DisplayName)
	assert.Contains(t, lastMsg.Embed.Description, student2.DisplayName)
}

func TestIntegration_General_GetMyClasses(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "class_1", "Class One")
	env.SeedClass(t, "class_2", "Class Two")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "class_1")
	env.SeedEnrollment(t, student.UserID, "class_2")

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!my_classes"})
	// HandleGetMyClasses uses c.Message.Author.ID directly
	ctx.Message.Author.ID = student.UserID

	// 2. ACT
	general.HandleGetMyClasses(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Description, "class_1")
	assert.Contains(t, lastMsg.Embed.Description, "class_2")
}

func TestIntegration_General_GetClassesOfUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "class_1", "Class One")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "class_1")

	ctx := env.NewContext("444555666", "Another User", "guild_1", []string{"!user_classes", fmt.Sprintf("<@%s>", student.UserID)})

	// 2. ACT
	general.HandleGetClassesOfUser(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Title, student.UserID)
	assert.Contains(t, lastMsg.Embed.Description, "class_1")
}
