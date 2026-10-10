package mentor

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

const (
	channelTimeout = 1 * time.Second
)

// newMockDiscordFetcher creates a fetcher with default behavior for GetCachedMember
func newMockDiscordFetcher(ctrl *gomock.Controller) *mock_providers.MockDiscordFetcher {
	fetcher := mock_providers.NewMockDiscordFetcher(ctrl)
	return fetcher
}

func awaitMessage(t *testing.T, out chan dispatcher.MessageResponse) dispatcher.MessageResponse {
	t.Helper()

	select {
	case msg := <-out:
		return msg
	case <-time.After(channelTimeout):
		t.Fatal("timeout")
		return dispatcher.MessageResponse{}
	}
}

func makeCommandContext(
	args []string,
	executorID string,
	guildID string,
	messageID string,
	svc *services.Services,
	validator *mock_providers.MockDiscordValidator,
	fetcher *mock_providers.MockDiscordFetcher,
	out chan dispatcher.MessageResponse,
) *context.ContextMessage {

	if messageID == "" {
		messageID = "msg_123"
	}

	return &context.ContextMessage{
		Args:             args,
		Executor:         service_model.User{UserID: executorID},
		ExecutorGuildID:  guildID,
		Services:         svc,
		DiscordValidator: validator,
		DiscordFetcher:   fetcher,
		Dispatcher:       &dispatcher.Dispatcher{Output: out},
		Message: &discordgo.MessageCreate{Message: &discordgo.Message{
			ID:        messageID,
			ChannelID: "chan_123",
		}},
	}
}

func makeReactionContext(
	executorID string,
	guildID string,
	emoji string,
	svc *services.Services,
	validator *mock_providers.MockDiscordValidator,
	fetcher *mock_providers.MockDiscordFetcher,
	out chan dispatcher.MessageResponse,
) *context.ContextMessage {

	return &context.ContextMessage{
		Executor:         service_model.User{UserID: executorID},
		ExecutorGuildID:  guildID,
		Services:         svc,
		DiscordValidator: validator,
		DiscordFetcher:   fetcher,
		Dispatcher:       &dispatcher.Dispatcher{Output: out},
		MessageReaction: &discordgo.MessageReactionAdd{MessageReaction: &discordgo.MessageReaction{
			MessageID: "msg_123",
			ChannelID: "chan_123",
			Emoji:     discordgo.Emoji{Name: emoji},
			GuildID:   guildID,
		}},
	}
}

func TestHandleAddAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful AC added",
			args:       []string{"!ac", "<@123>", "100"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)

				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role_student"}}, nil).AnyTimes()
				m.EXPECT().GetUserByUserID("123").Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()

				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 100}}).Return(map[data.ACKey]data.ACValidationData{
					{UserID: "123", ProblemID: 100}: {Problem: &data.Problem{ID: 100, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "CS101"}, AccessRoleIDs: []string{"role_student"}},
				}, nil)

				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("123", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("123", int64(100)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Reactions, common.REACTION_ACCEPTED)
			},
		},
		{
			name:       "parsing error",
			args:       []string{"!ac", "<@123>", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` as a problem ID")
			},
		},
		{
			name:       "service error",
			args:       []string{"!ac", "<@123>", "100"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role_student"}}, nil).AnyTimes()
				m.EXPECT().GetUserByUserID("123").Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()

				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 100}}).Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, "Unexpected internal error bro.", msg.Content)
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!ac", "<@123>", "100"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := makeCommandContext(tt.args, tt.executorID, tt.guildID, "msg_123", svc, mockVal, mockFetch, out)
			ctx.Dispatcher = disp
			HandleAddAC(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleUndoAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful undo AC",
			args:       []string{"!undo_ac", "<@123>", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role_student"}}, nil).AnyTimes()
				m.EXPECT().GetUserByUserID("123").Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("123", int64(77)).Return(true, nil)
				m.EXPECT().RemoveAC("123", int64(77)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "has been UnAC-ed successfully")
			},
		},
		{
			name:       "service error",
			args:       []string{"!undo_ac", "<@123>", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"role_student"}}, nil).AnyTimes()
				m.EXPECT().GetUserByUserID("123").Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("123", int64(77)).Return(false, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, "Unexpected internal error bro.", msg.Content)
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!undo_ac", "<@123>", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
		{
			name:       "parsing error",
			args:       []string{"!undo_ac", "<@123>", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` as a problem ID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := makeCommandContext(tt.args, tt.executorID, tt.guildID, "msg_123", svc, mockVal, mockFetch, out)
			ctx.Dispatcher = disp
			HandleUndoAC(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleAddProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful add problem (full args)",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "recommended", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().AddProblem(gomock.Any()).DoAndReturn(func(p data.Problem) (int64, error) {
					assert.Equal(t, "Two Sum", p.Name)
					if assert.NotNil(t, p.Difficulty) {
						assert.Equal(t, "easy", *p.Difficulty)
					}
					if assert.NotNil(t, p.Priority) {
						assert.Equal(t, "recommended", *p.Priority)
					}
					return 100, nil
				})
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "added correctly")
			},
		},
		{
			name:       "invalid difficulty",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Insane", "Loops", "-", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "invalid difficulty")
			},
		},
		{
			name:       "invalid priority",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "maybe", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "invalid priority")
			},
		},
		{
			name:       "service error",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "-", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "-", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := makeCommandContext(tt.args, tt.executorID, tt.guildID, "", svc, mockVal, mockFetch, out)
			ctx.Dispatcher = disp
			HandleAddProblem(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleRemoveProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
		wantErr    bool
	}{
		{
			name:       "successful removal",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetProblemByID(int64(123)).Return(data.Problem{ID: 123}, nil)
				m.EXPECT().RemoveProblemByID(int64(123)).Return(int64(1), nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "has been removed")
			},
		},
		{
			name:       "service error",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetProblemByID(int64(123)).Return(data.Problem{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
			wantErr: true,
		},
		{
			name:       "parsing error",
			args:       []string{"!remove_problem", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` as a problem ID")
			},
			wantErr: true,
		},
		{
			name:       "permission check failure",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := makeCommandContext(tt.args, tt.executorID, tt.guildID, "", svc, mockVal, mockFetch, out)
			ctx.Dispatcher = disp
			err := HandleRemoveProblem(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestCheckIfUserIsMentor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		executorID string
		guildID    string
		setupMock  func(v *mock_providers.MockDiscordValidator)
		want       bool
		wantErr    bool
	}{
		{
			name:       "missing executor user ID",
			executorID: "",
			guildID:    "guild_123",
			setupMock:  func(v *mock_providers.MockDiscordValidator) {},
			want:       false,
			wantErr:    true,
		},
		{
			name:       "DM execution (missing guild ID)",
			executorID: "user_123",
			guildID:    "",
			setupMock:  func(v *mock_providers.MockDiscordValidator) {},
			want:       false,
			wantErr:    true,
		},
		{
			name:       "validator error",
			executorID: "user_123",
			guildID:    "guild_123",
			setupMock: func(v *mock_providers.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "user_123").Return(false, fmt.Errorf("discord error"))
			},
			want:    false,
			wantErr: true,
		},
		{
			name:       "is mentor",
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(v *mock_providers.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockVal)

			ctx := &context.ContextMessage{
				Executor:         service_model.User{UserID: tt.executorID},
				ExecutorGuildID:  tt.guildID,
				DiscordValidator: mockVal,
			}

			got, err := checkIfUserIsMentor(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandlePendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		emoji      string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "accept AC proposal",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("123", int64(77), nil).Times(2)
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil)

				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("123", int64(77)).Return(false, nil)
				m.EXPECT().AddAC("123", int64(77)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.Reaction, msg.Type)
				assert.ElementsMatch(t, []string{common.REACTION_YES, common.REACTION_NO}, msg.ClearSpecificReactions)
				assert.Contains(t, msg.Reactions, common.REACTION_ACCEPTED)
			},
		},
		{
			name:       "reject AC proposal",
			emoji:      common.REACTION_NO,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("123", int64(77), nil)
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.Reaction, msg.Type)
				assert.ElementsMatch(t, []string{common.REACTION_YES, common.REACTION_NO}, msg.ClearSpecificReactions)
				assert.Contains(t, msg.Reactions, common.REACTION_REJECTED)
			},
		},
		{
			name:       "confirmation not found",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(false, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				assert.Equal(t, 0, len(out))
			},
		},
		{
			name:       "error fetching confirmation",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("", int64(0), fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrUnexpected, msg.Content)
			},
		},
		{
			name:       "accept proposal service error",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("123", int64(77), nil).Times(2)
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil)

				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("123", int64(77)).Return(false, nil)
				m.EXPECT().AddAC("123", int64(77)).Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, "Unexpected error.", msg.Content)
			},
		},
		{
			name:       "exists check error",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(false, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				assert.Equal(t, 0, len(out))
			},
		},
		{
			name:       "reject proposal service error",
			emoji:      common.REACTION_NO,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("123", int64(77), nil)
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrUnexpected, msg.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)

			ctx := makeReactionContext(tt.executorID, tt.guildID, tt.emoji, svc, mockVal, mockFetch, out)
			HandlePendingConfirmation(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleUpdateProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful update problem",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "essential", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, Priority: helpers.PtrStr("recommended")}, nil)
				m.EXPECT().ReplaceProblemByProblemID(int64(100), gomock.Any()).DoAndReturn(func(_ int64, updated data.Problem) error {
					if assert.NotNil(t, updated.Priority) {
						assert.Equal(t, "essential", *updated.Priority)
					}
					return nil
				})
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "updated successfully")
			},
		},
		{
			name:       "class not found",
			args:       []string{"!update_problem", "100", "invalid_class", "https://new-url.com", "Medium", "Strings", "-", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().GetClassByClassCode("invalid_class").Return(data.Class{}, fmt.Errorf("class not found"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "clear priority",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "-", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().ReplaceProblemByProblemID(int64(100), gomock.Any()).DoAndReturn(func(_ int64, updated data.Problem) error {
					assert.Nil(t, updated.Priority)
					return nil
				})
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "updated successfully")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "-", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
		{
			name:       "parsing error (ID)",
			args:       []string{"!update_problem", "not_an_int", "basicos_1", "https://new-url.com", "Medium", "Strings", "-", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` as a problem ID")
			},
		},
		{
			name:       "update service error",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "-", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(t *testing.T, m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				f.EXPECT().GetCachedMember("guild_123", gomock.Any()).Return(nil, fmt.Errorf("cache miss")).AnyTimes()
				m.EXPECT().GetUserByUserID(gomock.Any()).Return(data.User{UserID: "123", DisplayName: "Resolved Name"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1}, nil)
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().ReplaceProblemByProblemID(int64(100), gomock.Any()).Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := newMockDiscordFetcher(ctrl)
			tt.setupMock(t, mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := makeCommandContext(tt.args, tt.executorID, tt.guildID, "", svc, mockVal, mockFetch, out)
			ctx.Dispatcher = disp
			HandleUpdateProblem(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestParseDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		wantDifficulty string
		wantError      bool
	}{
		{name: "dash leaves difficulty unset", input: "-"},
		{name: "difficulty is normalized to canonical form", input: "MeDiUm", wantDifficulty: "medium"},
		{name: "unknown difficulty rejected", input: "expert", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotDifficulty, err := parseDifficulty(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.wantDifficulty == "" {
				assert.Nil(t, gotDifficulty)
			} else if assert.NotNil(t, gotDifficulty) {
				assert.Equal(t, tt.wantDifficulty, *gotDifficulty)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantPriority string
		wantError    bool
	}{
		{
			name:  "dash leaves priority unset",
			input: "-",
		},
		{
			name:         "priority is normalized",
			input:        "RECOMMENDED",
			wantPriority: "recommended",
		},
		{
			name:      "unknown priority rejected",
			input:     "maybe",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotPriority, err := parsePriority(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.wantPriority == "" {
				assert.Nil(t, gotPriority)
			} else if assert.NotNil(t, gotPriority) {
				assert.Equal(t, tt.wantPriority, *gotPriority)
			}
		})
	}
}
