package integration

import (
	"fmt"
	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/bootstrap"
	"github.com/JorgeIba/Hurion/app/services"
	"testing"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/admin"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ClearClass_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", classCode)
	env.SeedClassAccessRole(t, classCode, "role_1")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	studentID := "student_1"
	env.SeedUser(t, studentID, "Student")
	env.SeedPendingConfirmation(t, "pending_1", studentID, problemID)
	env.SeedAC(t, studentID, problemID)

	// Stub Admin role
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	// 2. ACT: Call Controller directly for the final action
	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!clear_class", classCode})
	ctx.Message.ID = "msg_original_123"

	err := admin.HandleClearClass(ctx)
	require.NoError(t, err)

	// 3. ASSERT: Class remains, but its contents and score-driving data are gone
	class, err := env.DB.GetClassByClassCode(classCode)
	require.NoError(t, err)
	assert.Equal(t, classCode, class.ClassCode)
	env.AssertProblemNotExists(t, problemID)
	env.AssertProblemNotSolved(t, studentID, problemID)

	rooms, err := env.DB.GetClassroomsByClassCode(classCode)
	require.NoError(t, err)
	assert.Len(t, rooms, 1)

	roles, err := env.DB.GetClassAccessRolesByClassCode(classCode)
	require.NoError(t, err)
	assert.Len(t, roles, 1)

	_, _, err = env.DB.GetPendingConfirmation("pending_1")
	var notFoundErr *dataerr.NotFoundError
	require.ErrorAs(t, err, &notFoundErr)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Contains(t, lastMsg.Content, "has been cleared successfully")
	}
}

func TestIntegration_ClearClass_ContextAware(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", classCode)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!clear_class"})

	err := admin.HandleClearClass(ctx)
	require.NoError(t, err)
	env.AssertProblemNotExists(t, problemID)

	class, err := env.DB.GetClassByClassCode(classCode)
	require.NoError(t, err)
	assert.Equal(t, classCode, class.ClassCode)
}

func TestIntegration_ClearClass_ExplicitTargetSurvivesRoomRelink(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	classACode := "class_a"
	classBCode := "class_b"
	classAID := env.SeedClass(t, classACode, "Class A")
	classBID := env.SeedClass(t, classBCode, "Class B")
	problemAID := env.SeedProblem(t, "Problem A", "url-a", classAID, "Easy", "Arrays")
	problemBID := env.SeedProblem(t, "Problem B", "url-b", classBID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", classACode)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	// Confirmation payload capture converts contextual clear_class into an explicit class argument.
	env.SeedClassroom(t, "channel_1", classBCode)
	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!clear_class", classACode})

	err := admin.HandleClearClass(ctx)
	require.NoError(t, err)

	env.AssertProblemNotExists(t, problemAID)
	_, err = env.DB.GetProblemByID(problemBID)
	require.NoError(t, err)

	classA, err := env.DB.GetClassByClassCode(classACode)
	require.NoError(t, err)
	assert.Equal(t, classACode, classA.ClassCode)

	classB, err := env.DB.GetClassByClassCode(classBCode)
	require.NoError(t, err)
	assert.Equal(t, classBCode, classB.ClassCode)
}

func TestIntegration_ClearClass_MissingContext(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!clear_class"})

	err := admin.HandleClearClass(ctx)
	require.Error(t, err)

	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "couldn't identify the class")
}

func TestIntegration_ExterminateClass_Logic(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", classCode)
	env.SeedClassAccessRole(t, classCode, "role_1")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	studentID := "student_1"
	env.SeedUser(t, studentID, "Student")
	env.SeedAC(t, studentID, problemID)

	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!exterminate_class", classCode})

	err := admin.HandleExterminateClass(ctx)
	require.NoError(t, err)

	env.AssertClassNotExists(t, classCode)
	env.AssertProblemNotExists(t, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has been removed successfully")
}

func TestIntegration_ExterminateUser_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	class1ID := env.SeedClass(t, "class_1", "Class One")
	class2ID := env.SeedClass(t, "class_2", "Class Two")
	p1 := env.SeedProblem(t, "P1", "url", class1ID, "Easy", "Arrays")
	p2 := env.SeedProblem(t, "P2", "url", class2ID, "Easy", "Arrays")

	testUserID := "999888777"
	env.SeedUser(t, testUserID, "Dead Man Walking")
	env.SeedAC(t, testUserID, p1)
	env.SeedAC(t, testUserID, p2)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!exterminate", fmt.Sprintf("<@%s>", testUserID)})

	// 2. ACT
	admin.HandleExterminate(ctx)

	// 3. ASSERT: Deep wipe verification
	env.AssertUserNotExists(t, testUserID)
	env.AssertProblemNotSolved(t, testUserID, p1)
	env.AssertProblemNotSolved(t, testUserID, p2)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "completely removed from the system")
}

func TestIntegration_LinkRoles_AlreadyLinkedMessage(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")
	env.SeedClassAccessRole(t, classCode, "1111111111")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!link_roles", classCode, "1111111111", "2222222222"})

	err := admin.HandleLinkRoles(ctx)
	require.NoError(t, err)

	roles, err := env.DB.GetClassAccessRolesByClassCode(classCode)
	require.NoError(t, err)
	assert.Len(t, roles, 2)

	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "Successfully linked 1 roles to class `basicos_1`")
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "1 already linked")
}

func TestIntegration_LinkRooms_AlreadyLinkedMessage(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")
	env.SeedClassroom(t, "1111111111", classCode)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!link_rooms", classCode, "1111111111", "2222222222"})

	err := admin.HandleLinkRooms(ctx)
	require.NoError(t, err)

	rooms, err := env.DB.GetClassroomsByClassCode(classCode)
	require.NoError(t, err)
	assert.Len(t, rooms, 2)

	sentMsgs := env.Dispatcher.GetSentMessages()
	require.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "Successfully linked 1 rooms to class `basicos_1`")
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "1 already linked")
}

func TestIntegration_CreateClass_AlreadyExists(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!create_class", classCode, "Duplicate Class"})

	// 2. ACT
	admin.HandleCreateClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	// The error mapper should handle the "already exists" error
	assert.Contains(t, lastMsg.Content, "already exists")
}

func TestIntegration_ExportClass_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "export_class"
	classID := env.SeedClass(t, classCode, "Export Class")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedAC(t, student.UserID, problemID)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!export", classCode})

	// 2. ACT
	err := admin.HandleExportClass(ctx)
	assert.NoError(t, err)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.NotEmpty(t, lastMsg.Files)
	assert.Contains(t, lastMsg.Files[0].Name, classCode)
}

func TestIntegration_Admin_UnauthorizedAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(*context.ContextMessage)
		args    []string
	}{
		{
			name: "CreateClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleCreateClass(c)
			},
			args: []string{"!create_class", "new_class", "New Class"},
		},
		{
			name: "ClearClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleClearClass(c)
			},
			args: []string{"!clear_class", "some_class"},
		},
		{
			name: "Exterminate unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleExterminate(c)
			},
			args: []string{"!exterminate", "<@123>"},
		},
		{
			name: "ExterminateClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleExterminateClass(c)
			},
			args: []string{"!exterminate_class", "some_class"},
		},
		{
			name: "ExportClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleExportClass(c)
			},
			args: []string{"!export", "some_class"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			student := service_model.User{UserID: "111", DisplayName: "Student"}

			// Seed something to verify it is NOT deleted/modified
			classCode := "some_class"
			env.SeedClass(t, classCode, "Protected")
			env.SeedUser(t, "123", "Protected user")

			// No administrator role is assigned.
			ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", tt.args)

			// ACT
			tt.handler(ctx)

			// ASSERT
			sentMsgs := env.Dispatcher.GetSentMessages()
			assert.NotEmpty(t, sentMsgs)
			assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "is not admin")

			// Verify DB side-effects: class should still exist
			_, err := env.DB.GetClassByClassCode(classCode)
			assert.NoError(t, err, "DB state should not change after unauthorized access")
		})
	}
}

func TestIntegration_ConfigReconcilerUsesClassService(t *testing.T) {
	t.Parallel()
	env := setupIntegrationEnv(t)
	svcs := services.NewServices(env.DB, authorization.New(nil, "", "", nil))
	reconciler := bootstrap.NewReconciler(svcs.Class, services.ConfigReconcilerActor())
	cfg := bootstrap.Config{Classes: []bootstrap.ClassConfig{{
		Code: "seeded", Name: "Seeded class", RoomIDs: []string{"seed-room"}, RoleIDs: []string{"student-role"},
	}}}
	for _, pass := range []struct{ name string }{{"initial seed"}, {"repeated seed"}} {
		t.Run(pass.name, func(t *testing.T) {
			assert.NoError(t, reconciler.Reconcile(cfg))
			class, err := env.DB.GetClassByClassCode("seeded")
			assert.NoError(t, err)
			assert.Equal(t, "Seeded class", class.Name)
			rooms, err := env.DB.GetClassroomsByClassCode("seeded")
			assert.NoError(t, err)
			if assert.Len(t, rooms, 1) {
				assert.Equal(t, "seed-room", rooms[0].RoomID)
			}
			roles, err := env.DB.GetClassAccessRolesByClassCode("seeded")
			assert.NoError(t, err)
			if assert.Len(t, roles, 1) {
				assert.Equal(t, "student-role", roles[0].RoleID)
			}
		})
	}
}
