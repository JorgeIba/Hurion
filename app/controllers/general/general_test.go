package general

import (
	"fmt"
	"os"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logger.Init()

	// Set standard test roles
	os.Setenv("DISCORD_MENTOR_ROLE_ID", "mentor_role_id")
	os.Setenv("DISCORD_ADMIN_ROLE_ID", "admin_role_id")

	code := m.Run()

	os.Unsetenv("DISCORD_MENTOR_ROLE_ID")
	os.Unsetenv("DISCORD_ADMIN_ROLE_ID")

	os.Exit(code)
}

func ptrStr(s string) *string { return &s }

func TestHandleBindUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		authorID   string
		authorNick string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful bind",
			args:       []string{"!bind", "jorge"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().IsUsernameAvailable("jorge", "discord_123").Return(true, nil)
				m.EXPECT().BindUser(data.User{UserID: "discord_123", UserName: "jorge"}).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, "User `Jorge Nick` binded to the user name `jorge`", msg.Content)
			},
		},
		{
			name:       "username already taken",
			args:       []string{"!bind", "jorge_taken"},
			authorID:   "discord_456",
			authorNick: "Other Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().IsUsernameAvailable("jorge_taken", "discord_456").Return(false, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				// The error message comes from adapters.MapError
				assert.Contains(t, msg.Content, "is already taken")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)

			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author: &discordgo.User{
							ID:         tt.authorID,
							GlobalName: tt.authorNick,
						},
					},
				},
			}

			HandleBindUser(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleRegisterUserInClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		authorID   string
		authorNick string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful registration",
			args:       []string{"!register", "basicos_1"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().IsUserBinded("discord_123").Return(true, nil)
				m.EXPECT().IsUserRegisteredInClass("discord_123", int64(1)).Return(false, nil)
				m.EXPECT().RegisterUserIntoClass("discord_123", int64(1)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, "User `Jorge Nick` has registered in the class `basicos_1`", msg.Content)
			},
		},
		{
			name:       "class not found",
			args:       []string{"!register", "invalid_class"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("invalid_class").Return(data.Class{}, fmt.Errorf("class not found"))
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
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)

			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author: &discordgo.User{
							ID:         tt.authorID,
							GlobalName: tt.authorNick,
						},
					},
				},
			}

			HandleRegisterUserInClass(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetListOfProblemsByClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful list problems",
			args: []string{"!list_problems", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{
					{ID: 1, Name: "Two Sum", Topic: ptrStr("Basics"), Difficulty: ptrStr("Easy")},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
				assert.NotEmpty(t, msg.Embeds)
				assert.Equal(t, "List of problems for class basicos_1", msg.Embeds[0].Title)
			},
		},
		{
			name: "no problems found",
			args: []string{"!list_problems", "empty_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("empty_class").Return(data.Class{ID: 2, ClassCode: "empty_class"}, nil)
				m.EXPECT().GetProblemsByClass(int64(2)).Return([]data.Problem{}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "No problems found")
			},
		},
		{
			name: "service error",
			args: []string{"!list_problems", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)

			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
					},
				},
			}

			HandleGetListOfProblemsByClass(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreByClass(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get score",
			args: []string{"!score", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "jorge"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "jorge"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.Equal(t, "table.png", msg.Files[0].Name)
			},
		},
		{
			name: "service error",
			args: []string{"!score", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetScoreByClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreByUserAndClass(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get user score",
			args: []string{"!score", "jorge", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "123", UserName: "jorge"}, nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "jorge"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "jorge"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
			},
		},
		{
			name: "service error",
			args: []string{"!score", "jorge", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetScoreByUserAndClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetUsersInClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get users",
			args: []string{"!users", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{
					{UserID: "123", UserName: "jorge"},
					{UserID: "456", UserName: "jose"},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "<@123>: jorge")
				assert.Contains(t, msg.Embed.Description, "<@456>: jose")
			},
		},
		{
			name: "service error",
			args: []string{"!users", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return(nil, fmt.Errorf("db error"))
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
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}

			HandleGetUsersInClass(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreAllClasses(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get score all classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return([]data.Class{
					{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"},
				}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "jorge"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "jorge"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
			},
		},
		{
			name: "service error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetScoreAllClasses(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetAllClasses(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get all classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return([]data.Class{
					{ClassCode: "basicos_1", Name: "Basics 1"},
					{ClassCode: "basicos_2", Name: "Basics 2"},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "`basicos_1`: Basics 1")
				assert.Contains(t, msg.Embed.Description, "`basicos_2`: Basics 2")
			},
		},
		{
			name: "service error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetAllClasses(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetMyClasses(t *testing.T) {
	tests := []struct {
		name       string
		authorID   string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:     "successful get my classes",
			authorID: "discord_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("discord_123").Return([]data.Class{
					{ClassCode: "basicos_1", Name: "Basics 1"},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "`basicos_1`: Basics 1")
			},
		},
		{
			name:     "no classes found",
			authorID: "discord_456",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("discord_456").Return([]data.Class{}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "You are not registered in any class")
			},
		},
		{
			name:     "service error",
			authorID: "discord_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("discord_123").Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author:    &discordgo.User{ID: tt.authorID},
					},
				},
			}
			HandleGetMyClasses(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetClassesOfUsername(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get classes of username",
			args: []string{"!classes", "jorge"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "123", UserName: "jorge"}, nil)
				m.EXPECT().GetClassesByUserID("123").Return([]data.Class{
					{ClassCode: "basicos_1", Name: "Basics 1"},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "`basicos_1`: Basics 1")
			},
		},
		{
			name: "user not found",
			args: []string{"!classes", "unknown"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserName("unknown").Return(data.User{}, fmt.Errorf("not found"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name: "classes service error",
			args: []string{"!classes", "jorge"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserName("jorge").Return(data.User{UserID: "123"}, nil)
				m.EXPECT().GetClassesByUserID("123").Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "Unexpected error.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetClassesOfUsername(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetMyUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		authorID   string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:     "successful get username",
			authorID: "discord_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("discord_123").Return(data.User{UserID: "discord_123", UserName: "jorge"}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, "Your username is `jorge`", msg.Content)
			},
		},
		{
			name:     "service error",
			authorID: "discord_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("discord_123").Return(data.User{}, fmt.Errorf("db error"))
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
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author:    &discordgo.User{ID: tt.authorID},
					},
				},
			}

			HandleGetMyUsername(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleChangeUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		authorID   string
		authorNick string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:       "successful change username",
			args:       []string{"!change", "new_jorge"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().HasUsernameBound("discord_123").Return(true, nil)
				m.EXPECT().IsUsernameAvailable("new_jorge", "discord_123").Return(true, nil)
				m.EXPECT().BindUser(data.User{UserID: "discord_123", UserName: "new_jorge"}).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "changed username to `new_jorge`")
			},
		},
		{
			name:       "unexpected error",
			args:       []string{"!change", "new_jorge"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().HasUsernameBound("discord_123").Return(false, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Equal(t, "Unexpected error.", msg.Content)
			},
		},
		{
			name:       "successful bind (no previous username)",
			args:       []string{"!change", "new_jorge"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().HasUsernameBound("discord_123").Return(false, nil)
				m.EXPECT().IsUsernameAvailable("new_jorge", "discord_123").Return(true, nil)
				m.EXPECT().BindUser(data.User{UserID: "discord_123", UserName: "new_jorge"}).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := <-out
				assert.Contains(t, msg.Content, "bound to the username `new_jorge` (no previous username found)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			ctx := &context.ContextMessage{
				Args:       tt.args,
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author: &discordgo.User{
							ID:         tt.authorID,
							GlobalName: tt.authorNick,
						},
					},
				},
			}

			HandleChangeUsername(ctx)

			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}
