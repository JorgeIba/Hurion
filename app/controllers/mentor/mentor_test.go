package mentor

import (
	"fmt"
	"os"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators/mock_validators"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	// Initialize logger for all tests in this package
	logger.Init()
	os.Exit(m.Run())
}

func TestHandleRegisterUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful registration",
			args:       []string{"!register_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "user_jorge", UserName: "jorge"}, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().IsUserBinded("user_jorge").Return(true, nil)
				m.EXPECT().IsUserRegisteredInClass("user_jorge", int64(1)).Return(false, nil)
				m.EXPECT().RegisterUserIntoClass("user_jorge", int64(1)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, "User `jorge` has registered in the class `basicos_1`", msg.Content)
			},
		},
		{
			name:       "not a mentor",
			args:       []string{"!register_user", "jorge", "basicos_1"},
			executorID: "user_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "user_123").Return(false, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrNotPermissions, msg.Content)
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!register_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
		{
			name:       "service error",
			args:       []string{"!register_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleRegisterUser(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleDeregisterUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
		wantErr    bool
	}{
		{
			name:       "successful deregistration",
			args:       []string{"!deregister_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "user_jorge", UserName: "jorge"}, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().IsUserRegisteredInClass("user_jorge", int64(1)).Return(true, nil)
				m.EXPECT().DeregisterUserFromClass("user_jorge", int64(1)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "has been deregistered")
			},
			wantErr: false,
		},
		{
			name:       "service error",
			args:       []string{"!deregister_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
			wantErr: true,
		},
		{
			name:       "permission check failure",
			args:       []string{"!deregister_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
			wantErr: true,
		},
		{
			name:       "deregister service error",
			args:       []string{"!deregister_user", "jorge", "basicos_1"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "user_jorge"}, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().IsUserRegisteredInClass("user_jorge", int64(1)).Return(true, nil)
				m.EXPECT().DeregisterUserFromClass("user_jorge", int64(1)).Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			err := HandleDeregisterUser(ctx)
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

func TestHandleAddAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		executorID string
		guildID    string
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful add AC",
			args:       []string{"!add_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "user_jorge", UserName: "jorge"}, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 77, Name: "Two Sum", ClassID: 1}, nil).AnyTimes()
				m.EXPECT().IsUserRegisteredInClass("user_jorge", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("user_jorge", int64(77)).Return(false, nil)
				m.EXPECT().AddAC("user_jorge", int64(77)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.ReplyMessage, msg.Type)
				assert.Contains(t, msg.Content, "Problem `77` has been AC-ed")
			},
		},
		{
			name:       "parsing error",
			args:       []string{"!add_ac", "jorge", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` to an integer")
			},
		},
		{
			name:       "service error",
			args:       []string{"!add_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!add_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ID: "msg_123", ChannelID: "chan_123"}},
			}
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful undo AC",
			args:       []string{"!undo_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "user_jorge", UserName: "jorge"}, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 77, Name: "Two Sum"}, nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("user_jorge", int64(77)).Return(true, nil)
				m.EXPECT().RemoveAC("user_jorge", int64(77)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "has been UnAC-ed successfully")
			},
		},
		{
			name:       "service error",
			args:       []string{"!undo_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!undo_ac", "jorge", "77"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
		{
			name:       "parsing error",
			args:       []string{"!undo_ac", "jorge", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` to an integer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ID: "msg_123", ChannelID: "chan_123"}},
			}
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful add problem (full args)",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).Times(2)
				m.EXPECT().AddProblem(gomock.Any()).DoAndReturn(func(p data.Problem) (int64, error) {
					assert.Equal(t, "Two Sum", p.Name)
					return 100, nil
				})
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "added correctly")
			},
		},
		{
			name:       "service error",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!add_problem", "basicos_1", "https://url.com", "Easy", "Loops", "Two", "Sum"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
		wantErr    bool
	}{
		{
			name:       "successful removal",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 123}, nil).AnyTimes()
				m.EXPECT().RemoveProblemByID(int64(123)).Return(int64(1), nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "has been removed")
			},
		},
		{
			name:       "service error",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{}, fmt.Errorf("db error")).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
			wantErr: true,
		},
		{
			name:       "parsing error",
			args:       []string{"!remove_problem", "not_an_int"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` to an integer")
			},
			wantErr: true,
		},
		{
			name:       "permission check failure",
			args:       []string{"!remove_problem", "123"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		want       bool
		wantErr    bool
	}{
		{
			name:       "missing executor user ID",
			executorID: "",
			guildID:    "guild_123",
			setupMock:  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {},
			want:       false,
			wantErr:    true,
		},
		{
			name:       "DM execution (missing guild ID)",
			executorID: "user_123",
			guildID:    "",
			setupMock:  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {},
			want:       false,
			wantErr:    true,
		},
		{
			name:       "validator error",
			executorID: "user_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "user_123").Return(false, fmt.Errorf("discord error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			ctx := &context.ContextMessage{
				ExecutorUserID:   tt.executorID,
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "accept AC proposal",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("user_123", int64(77), nil).AnyTimes()
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 77, ClassID: 1}, nil).AnyTimes()
				m.EXPECT().IsUserRegisteredInClass("user_123", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("user_123", int64(77)).Return(false, nil)
				m.EXPECT().AddAC("user_123", int64(77)).Return(nil)
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.ReplyMessage, msg.Type)
				assert.Contains(t, msg.Content, "Problem `77` has been AC-ed")
			},
		},
		{
			name:       "reject AC proposal",
			emoji:      common.REACTION_NO,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("user_123", int64(77), nil).AnyTimes()
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.ReplyMessage, msg.Type)
				assert.Contains(t, msg.Content, "Keep trying lil bro")
			},
		},
		{
			name:       "confirmation not found",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
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
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("", int64(0), fmt.Errorf("db error")).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrUnexpectec, msg.Content)
			},
		},
		{
			name:       "accept proposal service error",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("user_123", int64(77), nil).AnyTimes()
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 77, ClassID: 1}, nil).AnyTimes()
				m.EXPECT().IsUserRegisteredInClass("user_123", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("user_123", int64(77)).Return(false, nil)
				m.EXPECT().AddAC("user_123", int64(77)).Return(fmt.Errorf("db error"))
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrUnexpectec, msg.Content)
			},
		},
		{
			name:       "exists check error",
			emoji:      common.REACTION_YES,
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
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
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
				m.EXPECT().GetPendingConfirmation("msg_123").Return("user_123", int64(77), nil).AnyTimes()
				m.EXPECT().RemovePendingConfirmation("msg_123").Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrUnexpectec, msg.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				MessageReaction: &discordgo.MessageReactionAdd{
					MessageReaction: &discordgo.MessageReaction{
						MessageID: "msg_123",
						ChannelID: "chan_123",
						Emoji:     discordgo.Emoji{Name: tt.emoji},
					},
				},
			}
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful update problem",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 100}, nil).AnyTimes()
				m.EXPECT().UpdateProblemByProblemID(int64(100), gomock.Any()).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "updated successfully")
			},
		},
		{
			name:       "class not found",
			args:       []string{"!update_problem", "100", "invalid_class", "https://new-url.com", "Medium", "Strings", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetClassByClassCode("invalid_class").Return(data.Class{}, fmt.Errorf("class not found"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name:       "permission check failure",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(false, fmt.Errorf("discord error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, messageErrVerifying, msg.Content)
			},
		},
		{
			name:       "parsing error (ID)",
			args:       []string{"!update_problem", "not_an_int", "basicos_1", "https://new-url.com", "Medium", "Strings", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "I couldn't parse `not_an_int` to an integer")
			},
		},
		{
			name:       "update service error",
			args:       []string{"!update_problem", "100", "basicos_1", "https://new-url.com", "Medium", "Strings", "New", "Name"},
			executorID: "mentor_123",
			guildID:    "guild_123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_validators.MockDiscordValidator) {
				v.EXPECT().IsMentor("guild_123", "mentor_123").Return(true, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1}, nil)
				m.EXPECT().GetProblemByID(gomock.Any()).Return(data.Problem{ID: 100}, nil).AnyTimes()
				m.EXPECT().UpdateProblemByProblemID(int64(100), gomock.Any()).Return(fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
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
			mockVal := mock_validators.NewMockDiscordValidator(ctrl)
			tt.setupMock(mockStore, mockVal)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				ExecutorUserID:   tt.executorID,
				ExecutorGuildID:  tt.guildID,
				Services:         svc,
				DiscordValidator: mockVal,
				Dispatcher:       disp,
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleUpdateProblem(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}
