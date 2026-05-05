package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	p1 := env.SeedProblem(t, "P1", "url", classID, "Easy", "Topic")
	p2 := env.SeedProblem(t, "P2", "url", classID, "Easy", "Topic")
	p3 := env.SeedProblem(t, "P3", "url", classID, "Easy", "Topic")

	// Seed 7 users with different activity levels
	// 111: 3 ACs (Top)
	// 222: 2 ACs
	// 333: 2 ACs (Tie-break by ID)
	// 444: 1 AC (Target)
	// 555: 1 AC
	// 666: 1 AC
	// 777: 0 ACs (Should NOT appear)
	userIDs := []string{"111", "222", "333", "444", "555", "666", "777"}
	for i, id := range userIDs {
		env.SeedUser(t, id, fmt.Sprintf("User %d", i+1))
	}

	env.SeedAC(t, "111", p1)
	env.SeedAC(t, "111", p2)
	env.SeedAC(t, "111", p3)

	env.SeedAC(t, "222", p1)
	env.SeedAC(t, "222", p2)

	env.SeedAC(t, "333", p1)
	env.SeedAC(t, "333", p2)

	env.SeedAC(t, "444", p1)
	env.SeedAC(t, "555", p1)
	env.SeedAC(t, "666", p1)

	targetUserID := "444"
	targetDisplayName := "User 4"

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

	score, err := env.Services.Score.GetNeighborsScoreForUserID(targetUserID, classCode)
	assert.NoError(t, err)

	// Active users are 111, 222, 333, 444, 555, 666 (6 total).
	// Sorted by Score (desc), then ID (asc):
	// 0: 111 (3 AC)
	// 1: 222 (2 AC)
	// 2: 333 (2 AC)
	// 3: 444 (1 AC) <-- Target (Index 3)
	// 4: 555 (1 AC)
	// 5: 666 (1 AC)
	// Window (2 before, 2 after): [1:222, 2:333, 3:444, 4:555, 5:666]
	assert.Len(t, score.UserScores, 5)
	assert.Equal(t, "222", score.UserScores[0].User.UserID)
	assert.Equal(t, "333", score.UserScores[1].User.UserID)
	assert.Equal(t, "444", score.UserScores[2].User.UserID) // Target
	assert.Equal(t, "555", score.UserScores[3].User.UserID)
	assert.Equal(t, "666", score.UserScores[4].User.UserID)

	// Verify User 7 is NOT in the active user list at all
	activeUsers, err := env.DB.GetActiveUsersByClassID(classID)
	require.NoError(t, err)
	assert.Len(t, activeUsers, 6)
	for _, u := range activeUsers {
		assert.NotEqual(t, "777", u.UserID)
	}

	for i := 0; i < len(score.UserScores)-1; i++ {
		curr, next := score.UserScores[i], score.UserScores[i+1]
		if curr.CountProblems == next.CountProblems {
			assert.Less(t, curr.User.UserID, next.User.UserID, "Deterministic sort failed for UserIDs")
		} else {
			assert.Greater(t, curr.CountProblems, next.CountProblems, "Score sort failed")
		}
	}
}

func TestIntegration_General_GetScoreByUserAndClass_TopRanked(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	p1 := env.SeedProblem(t, "P1", "url", classID, "Easy", "Arrays")

	// Seed 5 users
	userIDs := []string{"111", "222", "333", "444", "555"}
	for i, id := range userIDs {
		env.SeedUser(t, id, fmt.Sprintf("User %d", i+1))
		env.SeedAC(t, id, p1)
	}

	targetUserID := "111"

	ctx := env.NewContext("999", "Exec", "guild_1", []string{"!score_user", classCode, fmt.Sprintf("<@%s>", targetUserID)})

	// 2. ACT
	general.HandleGetScoreByUserAndClass(ctx)

	// 3. ASSERT
	score, err := env.Services.Score.GetNeighborsScoreForUserID(targetUserID, classCode)
	assert.NoError(t, err)

	assert.Len(t, score.UserScores, 3, "Top ranked user should have 0 neighbors above and 2 below")
	assert.Equal(t, "111", score.UserScores[0].User.UserID, "Target should be at index 0")
}

func TestIntegration_General_GetUsersInClass_DisplayNamePriority(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")

	userID := "111222333"
	env.SeedUser(t, userID, "DB Name") // DB has one name
	pID := env.SeedProblem(t, "P1", "url", classID, "Easy", "Topic")
	env.SeedAC(t, userID, pID)

	// Stub fetcher returns a different Nickname (higher priority)
	env.Fetcher.GetCachedMemberFunc = func(guildID, userIDArg string) (*discordgo.Member, error) {
		return &discordgo.Member{
			User: &discordgo.User{ID: userID, Username: "Discord Username", GlobalName: "Discord Global"},
			Nick: "Discord Nickname",
		}, nil
	}

	ctx := env.NewContext("555666777", "User One", "guild_1", []string{"!users", classCode})

	// 2. ACT
	general.HandleGetUsersInClass(ctx)

	// 3. ASSERT: Should use Nickname
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Embed.Description, "Discord Nickname")
}

func TestIntegration_General_GetUsersInClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")

	student1 := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	student2 := service_model.User{UserID: "222333444", DisplayName: "Student Two"}
	env.SeedUser(t, student1.UserID, student1.DisplayName)
	env.SeedUser(t, student2.UserID, student2.DisplayName)
	pID := env.SeedProblem(t, "P1", "url", classID, "Easy", "Topic")
	env.SeedAC(t, student1.UserID, pID)
	env.SeedAC(t, student2.UserID, pID)

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
	env.SeedClassAccessRole(t, "class_1", "role_1")
	env.SeedClassAccessRole(t, "class_2", "role_2")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	// Give roles for both classes
	env.SeedRoleToUser(student.UserID, "role_1")
	env.SeedRoleToUser(student.UserID, "role_2")

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!my_classes"})

	// 2. ACT
	general.HandleGetMyClasses(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
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
	env.SeedClassAccessRole(t, "class_1", "role_1")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	// Give role to target user
	env.SeedRoleToUser(student.UserID, "role_1")

	ctx := env.NewContext("444555666", "Another User", "guild_1", []string{"!user_classes", fmt.Sprintf("<@%s>", student.UserID)})

	// 2. ACT
	general.HandleGetClassesOfUser(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Title, student.UserID)
	assert.Contains(t, lastMsg.Embed.Description, "class_1")
}

func TestIntegration_General_GetRolesAndChannels(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos"
	env.SeedClass(t, classCode, "Basicos 2026")
	env.SeedClassAccessRole(t, classCode, "role_123")
	env.SeedClassroom(t, "chan_456", classCode)

	t.Run("get_roles explicit", func(t *testing.T) {
		ctx := env.NewContext("123", "User", "guild_1", []string{"!get_roles", classCode})
		general.HandleGetClassRoles(ctx)
		sentMsgs := env.Dispatcher.GetSentMessages()
		require.NotEmpty(t, sentMsgs)
		assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "role_123")
	})

	t.Run("get_channels contextual", func(t *testing.T) {
		// Run from the linked channel
		ctx := env.NewContext("123", "User", "guild_1", []string{"!get_channels"})
		ctx.Message.ChannelID = "chan_456"

		general.HandleGetClassRooms(ctx)
		sentMsgs := env.Dispatcher.GetSentMessages()
		require.NotEmpty(t, sentMsgs)
		assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "chan_456")
	})
}
