package general

import (
	"fmt"
	"os"
	"testing"
	"time"

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
	code := m.Run()
	os.Exit(code)
}

const (
	channelTimeout = 1 * time.Second
)

func ptrStr(s string) *string { return &s }

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
				m.EXPECT().EnsureUser("discord_123").Return(nil)
				m.EXPECT().IsUserRegisteredInClass("discord_123", int64(1)).Return(false, nil)
				m.EXPECT().RegisterUserIntoClass("discord_123", int64(1)).Return(nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, "User `Jorge Nick` has registered in the class `basicos_1`", msg.Content)
			},
		},
		{
			name:       "service error",
			args:       []string{"!register", "invalid_class"},
			authorID:   "discord_123",
			authorNick: "Jorge Nick",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("invalid_class").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
					{UserID: "123", UserName: "Nick1"},
					{UserID: "456", UserName: "Nick2"},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "<@123>")
				assert.Contains(t, msg.Embed.Description, "<@456>")
			},
		},
		{
			name: "service error",
			args: []string{"!users", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
			args: []string{"!list", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{
					ID:         1,
					Name:       "Two Sum",
					Topic:      ptrStr("Basics"),
					Difficulty: ptrStr("Easy"),
				}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
				assert.NotEmpty(t, msg.Embeds)
				assert.Contains(t, msg.Embeds[0].Title, "basicos_1")
			},
		},
		{
			name: "no problems found",
			args: []string{"!list", "empty_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("empty_class").Return(data.Class{ID: 2, ClassCode: "empty_class"}, nil)
				m.EXPECT().GetProblemsByClass(int64(2)).Return([]data.Problem{}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Contains(t, msg.Content, "No problems found")
			},
		},
		{
			name: "service error",
			args: []string{"!list", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
			HandleGetListOfProblemsByClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreByClass(t *testing.T) {
	t.Parallel()

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
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "123"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "123"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
			HandleGetScoreByClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreByUserAndClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get user score",
			args: []string{"!score", "basicos_1", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "123"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "123"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
				assert.Contains(t, msg.Embeds[0].Title, "<@123>")
			},
		},
		{
			name: "service error",
			args: []string{"!score", "basicos_1", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
			HandleGetScoreByUserAndClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetScoreAllClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get score all classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return([]data.Class{{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "123", UserName: "123"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", UserName: "123"}: {{ProblemID: 1}},
				}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Contains(t, msg.Content, "Unexpected internal error.")
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
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get all classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetAllClasses().Return([]data.Class{{ClassCode: "basicos_1", Name: "Basics 1"}, {ClassCode: "basicos_2", Name: "Basics 2"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
	t.Parallel()

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
				m.EXPECT().GetClassesByUserID("discord_123").Return([]data.Class{{ClassCode: "basicos_1", Name: "Basics 1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
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
				Services:   svc,
				Dispatcher: disp,
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{ChannelID: "chan_123", Author: &discordgo.User{ID: tt.authorID}},
				},
			}
			HandleGetMyClasses(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetClassesOfUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get classes of user",
			args: []string{"!classes_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("123").Return([]data.Class{{ClassCode: "basicos_1", Name: "Basics 1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "`basicos_1`: Basics 1")
			},
		},
		{
			name: "no classes found",
			args: []string{"!classes_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("123").Return([]data.Class{}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Contains(t, msg.Content, "User <@123> not registered in any class")
			},
		},
		{
			name: "service error",
			args: []string{"!classes_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("123").Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				var msg dispatcher.MessageResponse
				select {
				case msg = <-out:
				case <-time.After(channelTimeout):
					t.Fatal("timeout")
				}
				assert.Equal(t, "Unexpected error.", msg.Content)
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
			HandleGetClassesOfUser(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}
