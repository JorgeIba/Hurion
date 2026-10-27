package workflow

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScopedMentorWorkflow(t *testing.T) *workflowEnv {
	t.Helper()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	scoped := authorization.New(authorization.NewDiscordRoleSource(env.Discord, testGuildID), "admin", "global", map[string]string{"advanced": "advanced-mentor"})
	env.Services = services.NewServices(env.DB, scoped)
	responder := dispatcher.NewDiscordInteractionResponder(env.Responder)
	env.Bot = appbot.NewBot(env.Services, env.Discord, env.Dispatcher, responder, appbot.BotConfig{GuildID: testGuildID})
	env.ProblemController = mentor.NewProblemInteractionController(env.Services, responder)
	env.ACController = mentor.NewACInteractionController(env.Services, env.Discord, responder)
	env.seedClass(t, "basics", "Basics")
	env.seedClass(t, "advanced", "Advanced")
	env.seedClassroom(t, testChannelID, "basics")
	env.seedClassAccessRole(t, "basics", "student")
	env.seedClassAccessRole(t, "advanced", "student")
	env.setRoles("123", []string{"student"})
	return env
}

func scopedProblem(t *testing.T, env *workflowEnv, code string) int64 {
	t.Helper()
	class, err := env.Services.Class.GetClassByClassCode(code)
	require.NoError(t, err)
	return env.seedProblem(t, "Original", "https://example.com/original", class.ID, "easy", "Arrays")
}

func TestWorkflow_ClassAuthorizationRejectsEventsFromOtherGuilds(t *testing.T) {
	t.Parallel()
	env := setupScopedMentorWorkflow(t)
	env.setRoles("mentor", []string{"admin"})
	id := scopedProblem(t, env, "advanced")
	env.sendMessage(t, "123", "Student", fmt.Sprintf("!ac %d", id))
	env.assertPendingConfirmation(t, "msg_123", "123", id)
	before := len(env.Dispatcher.Messages())
	env.Bot.HandleMessageEvent(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "foreign", GuildID: "other-guild", ChannelID: testChannelID,
		Author:  &discordgo.User{ID: "mentor"},
		Content: fmt.Sprintf("!update_problem %d advanced https://example.com/new easy - - Changed", id),
	}})
	env.Bot.HandleReactionEvent(&discordgo.MessageReactionAdd{MessageReaction: &discordgo.MessageReaction{
		GuildID: "other-guild", ChannelID: testChannelID, MessageID: "msg_123",
		UserID: "mentor", Emoji: discordgo.Emoji{Name: common.REACTION_YES},
	}, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}}})
	problem, err := env.Services.Problem.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, "Original", problem.Name)
	env.assertPendingConfirmation(t, "msg_123", "123", id)
	solved, err := env.DB.HasUserSolvedProblem("123", id)
	require.NoError(t, err)
	assert.False(t, solved)
	assert.Len(t, env.Dispatcher.Messages(), before)
}

func TestWorkflow_ClassMentorTextOperationsUseProblemClass(t *testing.T) {
	t.Parallel()
	for _, actor := range []struct {
		name             string
		roles            []string
		advanced, basics bool
	}{
		{"global", []string{"global"}, false, true},
		{"custom", []string{"advanced-mentor"}, true, false},
		{"admin", []string{"admin"}, true, true},
		{"student", []string{"student"}, false, false},
	} {
		for _, code := range []string{"basics", "advanced"} {
			for _, operation := range []string{"create", "update", "remove", "grant", "undo"} {
				t.Run(actor.name+"/"+code+"/"+operation, func(t *testing.T) {
					t.Parallel()
					env := setupScopedMentorWorkflow(t)
					env.setRoles("mentor", actor.roles)
					id := scopedProblem(t, env, code)
					allowed := actor.basics
					if code == "advanced" {
						allowed = actor.advanced
					}
					command := ""
					switch operation {
					case "create":
						command = fmt.Sprintf("!add_problem %s https://example.com/new easy - - New", code)
					case "update":
						command = fmt.Sprintf("!update_problem %d %s https://example.com/new easy - - Changed", id, code)
					case "remove":
						command = fmt.Sprintf("!remove_problem %d", id)
					case "grant":
						command = fmt.Sprintf("!ac 123 %d", id)
					case "undo":
						env.seedUser(t, "123", "Student")
						require.NoError(t, env.DB.AddAC("123", id))
						command = fmt.Sprintf("!undo_ac 123 %d", id)
					}
					env.sendMessage(t, "mentor", "Mentor", command)
					if operation == "remove" {
						env.Discord.storeMessage(newConfirmationMessage("confirmation", "msg_mentor", []string{"!remove_problem", fmt.Sprint(id)}, "mentor"))
						env.Discord.storeMessage(&discordgo.Message{ID: "msg_mentor", ChannelID: testChannelID, GuildID: testGuildID, Author: &discordgo.User{ID: "mentor"}})
						env.sendReaction(t, "mentor", "Mentor", "confirmation", common.REACTION_THUMBS_UP)
					}
					if operation == "create" {
						problems, err := env.Services.Problem.GetProblemsByClass(code)
						require.NoError(t, err)
						want := 1
						if allowed {
							want = 2
						}
						assert.Len(t, problems, want)
					} else if operation == "remove" {
						_, err := env.DB.GetProblemByID(id)
						assert.Equal(t, allowed, err != nil)
					} else if operation == "update" {
						problem, err := env.DB.GetProblemByID(id)
						require.NoError(t, err)
						want := "Original"
						if allowed {
							want = "Changed"
						}
						assert.Equal(t, want, problem.Name)
					} else {
						solved, err := env.DB.HasUserSolvedProblem("123", id)
						require.NoError(t, err)
						want := allowed
						if operation == "undo" {
							want = !allowed
						}
						assert.Equal(t, want, solved)
					}
				})
			}
		}
	}
}

func TestWorkflow_ClassMentorMoveRequiresBothClasses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		roles   []string
		allowed bool
	}{
		{"source only", []string{"global"}, false},
		{"destination only", []string{"advanced-mentor"}, false},
		{"both", []string{"global", "advanced-mentor"}, true},
		{"admin", []string{"admin"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupScopedMentorWorkflow(t)
			env.setRoles("mentor", tt.roles)
			id := scopedProblem(t, env, "basics")
			env.sendMessage(t, "mentor", "Mentor", fmt.Sprintf("!update_problem %d advanced https://example.com/moved easy - - Moved", id))
			problem, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			class, err := env.Services.Class.GetClassByClassID(problem.ClassID)
			require.NoError(t, err)
			want := "basics"
			if tt.allowed {
				want = "advanced"
			}
			assert.Equal(t, want, class.ClassCode)
		})
	}
}

func TestWorkflow_ClassMentorProposalDecisions(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"reaction approve", "reaction reject", "button approve", "button reject"} {
		for _, actor := range []struct {
			name    string
			roles   []string
			allowed bool
		}{
			{"global", []string{"global"}, false},
			{"custom", []string{"advanced-mentor"}, true},
			{"admin", []string{"admin"}, true},
		} {
			t.Run(path+"/"+actor.name, func(t *testing.T) {
				t.Parallel()
				env := setupScopedMentorWorkflow(t)
				env.setRoles("mentor", actor.roles)
				id := scopedProblem(t, env, "advanced")
				// The ordinary student's proposal succeeds even from a different class's room.
				env.sendMessage(t, "123", "Student", fmt.Sprintf("!ac %d", id))
				env.assertPendingConfirmation(t, "msg_123", "123", id)
				approve := path == "reaction approve" || path == "button approve"
				if path == "reaction approve" || path == "reaction reject" {
					emoji := common.REACTION_NO
					if approve {
						emoji = common.REACTION_YES
					}
					env.sendReaction(t, "mentor", "Mentor", "msg_123", emoji)
				} else {
					customID := interactiontransport.ACProposalRejectID("msg_123")
					if approve {
						customID = interactiontransport.ACProposalApproveID("msg_123")
					}
					env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
						Type: discordgo.InteractionMessageComponent, GuildID: testGuildID, ChannelID: testChannelID,
						Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}}, Message: &discordgo.Message{ID: "card"},
						Data: discordgo.MessageComponentInteractionData{ComponentType: discordgo.ButtonComponent, CustomID: customID},
					}})
				}
				if actor.allowed {
					env.assertPendingConfirmationRemoved(t, "msg_123")
				} else {
					env.assertPendingConfirmation(t, "msg_123", "123", id)
				}
				solved, err := env.DB.HasUserSolvedProblem("123", id)
				require.NoError(t, err)
				assert.Equal(t, actor.allowed && approve, solved)
			})
		}
	}
}

func TestWorkflow_ClassMentorFormsRecheckRolesAndNormalizeClasses(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "edit"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			env := setupScopedMentorWorkflow(t)
			env.setRoles("mentor", []string{"global", "advanced-mentor"})
			id := scopedProblem(t, env, "advanced")
			var response *discordgo.InteractionResponse
			if operation == "create" {
				var err error
				response, err = buildCreateProblemModalForTest(interactiontransport.CreateProblemModal{ClassCode: " advanced ", ProblemModalFields: interactiontransport.ProblemModalFields{Name: "New", URL: "https://example.com/new"}})
				require.NoError(t, err)
			} else {
				response = env.openProblemEditor(t, "mentor", id, "basics")
			}
			env.setRoles("mentor", []string{"global"}) // Global intake still passes, class authorization must fail.
			if operation == "edit" {
				env.submitProblemEditor(t, "mentor", response, interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/changed"})
			} else {
				env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}}, Data: submittedProblemFormData(t, response),
				}})
			}
			require.NotEmpty(t, env.Responder.edits)
			assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "No changes were applied")
			problem, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			assert.Equal(t, "Original", problem.Name)
			problems, err := env.Services.Problem.GetProblemsByClass("advanced")
			require.NoError(t, err)
			assert.Len(t, problems, 1)
		})
	}
}

func TestWorkflow_ClassMentorBatchesRejectAllUnauthorizedWrites(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"create", "edit", "AC"} {
		for _, phase := range []string{"preview", "confirm", "cancel"} {
			t.Run(family+"/"+phase, func(t *testing.T) {
				t.Parallel()
				env := setupScopedMentorWorkflow(t)
				env.setRoles("mentor", []string{"global", "advanced-mentor"})
				basicsID := scopedProblem(t, env, "basics")
				advancedID := scopedProblem(t, env, "advanced")
				command, subcommand, route := "problem", "add_batch", string(discordcommands.ProblemBatch)
				input := `[{"class":"basics","name":"New basic","url":"https://example.com/new","topic":"Arrays"},{"class":"advanced","name":"New advanced","url":"https://example.com/new","topic":"Arrays"}]`
				if family == "edit" {
					subcommand, route = "edit_batch", string(discordcommands.ProblemEditBatch)
					input = fmt.Sprintf(`[{"problem_id":%d,"name":"Changed basic"},{"problem_id":%d,"name":"Changed advanced"}]`, basicsID, advancedID)
				} else if family == "AC" {
					command, route = "ac", string(discordcommands.ACBatch)
					input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d},{"user_id":"123","problem_id":%d}]`, basicsID, advancedID)
				}
				if phase == "preview" {
					env.setRoles("mentor", []string{"global"})
					env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
						Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
						Data: discordgo.ModalSubmitInteractionData{CustomID: route, Components: []discordgo.MessageComponent{
							&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
							&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
						}},
					}})
					require.Len(t, env.Responder.edits, 1)
					assert.Contains(t, *env.Responder.edits[0].response.Content, "No changes were applied")
				} else {
					preview := submitBatchCommandPreview(t, env, command, subcommand, input)
					env.setRoles("mentor", []string{"global"})
					label := "Confirm"
					if phase == "cancel" {
						label = "Cancel"
					}
					env.handleInteractionEvent(batchButtonEvent(t, preview, label))
					require.Len(t, env.Responder.edits, 2)
					if phase == "cancel" {
						assert.Contains(t, *env.Responder.edits[1].response.Content, "cancel")
					} else {
						assert.Contains(t, *env.Responder.edits[1].response.Content, "No changes were applied")
					}
				}
				for _, code := range []string{"basics", "advanced"} {
					problems, err := env.Services.Problem.GetProblemsByClass(code)
					require.NoError(t, err)
					require.Len(t, problems, 1, "no partial creation")
					assert.Equal(t, "Original", problems[0].Name, "no partial edits")
					solved, err := env.DB.HasUserSolvedProblem("123", problems[0].ID)
					require.NoError(t, err)
					assert.False(t, solved, "no partial AC grants")
				}
			})
		}
	}
}

func TestWorkflow_ClassMentorBatchConfirmationUsesCurrentProblemClass(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"edit", "AC"} {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			env := setupScopedMentorWorkflow(t)
			env.setRoles("mentor", []string{"global"})
			first := scopedProblem(t, env, "basics")
			second := scopedProblem(t, env, "basics")
			command, subcommand := "problem", "edit_batch"
			input := fmt.Sprintf(`[{"problem_id":%d,"name":"Changed first"},{"problem_id":%d,"name":"Changed second"}]`, first, second)
			if family == "AC" {
				command, subcommand = "ac", "add_batch"
				input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d},{"user_id":"123","problem_id":%d}]`, first, second)
			}
			preview := submitBatchCommandPreview(t, env, command, subcommand, input)
			// A different administrator moves one target after the preview was issued.
			env.setRoles("admin", []string{"admin"})
			env.sendMessage(t, "admin", "Admin", fmt.Sprintf("!update_problem %d advanced https://example.com/moved easy - - Original", second))
			env.handleInteractionEvent(batchButtonEvent(t, preview, "Confirm"))
			require.Len(t, env.Responder.edits, 2)
			assert.Contains(t, *env.Responder.edits[1].response.Content, "No changes were applied")
			for _, id := range []int64{first, second} {
				problem, err := env.DB.GetProblemByID(id)
				require.NoError(t, err)
				assert.Equal(t, "Original", problem.Name)
				solved, err := env.DB.HasUserSolvedProblem("123", id)
				require.NoError(t, err)
				assert.False(t, solved)
			}
		})
	}
}

func TestWorkflow_ClassMentorSlashCommandsCheckSourceAndDestination(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, command, source, destination string
		roles                              []string
		allowed                            bool
	}{
		{"global cannot open advanced add", "add", "", "advanced", []string{"global"}, false},
		{"custom can open advanced add", "add", "", "advanced", []string{"advanced-mentor"}, true},
		{"source-only cannot move", "edit", "basics", "advanced", []string{"global"}, false},
		{"destination-only cannot move", "edit", "basics", "advanced", []string{"advanced-mentor"}, false},
		{"both can move", "edit", "basics", "advanced", []string{"global", "advanced-mentor"}, true},
		{"custom editor uses actual class", "edit", "advanced", "", []string{"advanced-mentor"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupScopedMentorWorkflow(t)
			env.setRoles("mentor", tt.roles)
			options := []*discordgo.ApplicationCommandInteractionDataOption{}
			if tt.destination != "" {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Value: tt.destination})
			}
			if tt.source != "" {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "problem", Value: fmt.Sprint(scopedProblem(t, env, tt.source))})
			}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
				Data: discordgo.ApplicationCommandInteractionData{Name: "problem", Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: tt.command, Type: discordgo.ApplicationCommandOptionSubCommand, Options: options}}},
			}})
			require.Len(t, env.Responder.responses, 1)
			response := env.Responder.responses[0].response
			require.Equal(t, discordgo.InteractionResponseModal, response.Type, "opening a form does not authorize a write")
			fields := interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/changed"}
			if tt.command == "edit" {
				env.submitProblemEditor(t, "mentor", response, fields)
			} else {
				modal, err := interactiontransport.ParseCreateProblemModal(submittedProblemFormData(t, response))
				require.NoError(t, err)
				modal.ProblemModalFields = fields
				env.submitProblemForm(t, "mentor", modal)
			}
			require.NotEmpty(t, env.Responder.edits)
			receipt := *env.Responder.edits[len(env.Responder.edits)-1].response.Content
			if !tt.allowed {
				assert.Contains(t, receipt, "mentor or admin")
			} else {
				assert.NotContains(t, receipt, "mentor or admin")
			}
		})
	}
}

func TestWorkflow_ClassMentorDoesNotGrantAdminOrChangeStudentUndo(t *testing.T) {
	t.Parallel()
	env := setupScopedMentorWorkflow(t)
	env.setRoles("mentor", []string{"advanced-mentor"})
	env.sendMessage(t, "mentor", "Mentor", "!create_class unauthorized Unauthorized")
	_, err := env.Services.Class.GetClassByClassCode("unauthorized")
	require.Error(t, err, "class mentors must not gain admin-only commands")
	id := scopedProblem(t, env, "advanced")
	env.seedUser(t, "123", "Student")
	require.NoError(t, env.DB.AddAC("123", id))
	env.sendMessage(t, "123", "Student", fmt.Sprintf("!undo_ac %d", id))
	env.assertProblemNotSolved(t, "123", id)
}
