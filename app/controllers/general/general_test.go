package general

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers/mock_providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
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

func TestHandleGetUsersInClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get users",
			args: []string{"!users", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes()
				m.EXPECT().GetActiveUsersByClassID(int64(1)).Return([]data.User{
					{UserID: "123", DisplayName: "Jorge"},
					{UserID: "456", DisplayName: "Luis"},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "Jorge"}, nil).AnyTimes()
				f.EXPECT().GetCachedMember(gomock.Any(), "456").Return(&discordgo.Member{User: &discordgo.User{ID: "456"}, Nick: "Luis"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Description, "Jorge")
				assert.Contains(t, msg.Embed.Description, "Luis")
			},
		},
		{
			name: "service error",
			args: []string{"!users", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
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
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetUsersInClass(ctx)
			if tt.assertFunc != nil {
				tt.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetListOfProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful list problems (explicit)",
			args: []string{"!list", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes().AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
				assert.NotEmpty(t, msg.Embeds)
			},
		},
		{
			name: "successful list problems (contextual)",
			args: []string{"!list"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos_1", nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil).AnyTimes().AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
				assert.NotEmpty(t, msg.Embeds)
			},
		},
		{
			name: "no problems found",
			args: []string{"!list", "empty_class"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("empty_class").Return(data.Class{ID: 2, ClassCode: "empty_class"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(2)).Return([]data.Problem{}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "No problems found")
			},
		},
		{
			name: "service error",
			args: []string{"!list", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				// Internal error during class lookup
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Unexpected internal error bro.")
			},
		},
		{
			name: "controller error",
			args: []string{"!list"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				// No expectations needed as it fails context validation before calling services
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Something wrong with the controller, bro")
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
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}

			// Context without Message to trigger controller validation error in contextual commands
			ctx := &context.ContextMessage{
				Args:             tt.args,
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Executor:         service_model.User{UserID: "u1"},
			}

			if tt.name != "controller error" {
				ctx.Message = &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}}
			}

			HandleGetListOfProblems(ctx)
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get score by class",
			args: []string{"!score", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
			},
		},
		{
			name: "successful get score by contextual class",
			args: []string{"!score"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos_1", nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
			},
		},
		{
			name: "service error",
			args: []string{"!score", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
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
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get user score (mention)",
			args: []string{"!score_of", "<@123>", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
				assert.Contains(t, msg.Embeds[0].Title, "name_123")
			},
		},
		{
			name: "successful get user score (raw ID)",
			args: []string{"!score_of", "123", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
				assert.Contains(t, msg.Embeds[0].Title, "name_123")
			},
		},
		{
			name: "successful get user score by contextual class",
			args: []string{"!score_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos_1", nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
				assert.Contains(t, msg.Embeds[0].Title, "name_123")
			},
		},
		{
			name: "user has no solved problems in contextual class",
			args: []string{"!score_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos_1", nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "doesn't have any solved problems in class `basicos_1` yet")
			},
		},
		{
			name: "service error",
			args: []string{"!score_of", "<@123>", "basicos_1"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{}, fmt.Errorf("db error"))
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
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get score all classes",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetAllClasses().Return([]data.Class{{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}}, nil).AnyTimes()
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1", Name: "Basics 1"}, nil).AnyTimes()
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 1, Name: "Two Sum"}}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "123", DisplayName: "name_123"}: {{ProblemID: 1}},
				}, nil)
				f.EXPECT().GetCachedMember(gomock.Any(), "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "name_123"}, nil).AnyTimes()
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.FileMessage, msg.Type)
				assert.NotEmpty(t, msg.Files)
				assert.NotEmpty(t, msg.Embeds)
			},
		},
		{
			name: "service error",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetAllClasses().Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
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
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetScoreAllClasses(ctx)
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get classes (mention)",
			args: []string{"!classes_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{Roles: []string{"role_1"}}, nil)
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return([]data.Class{{ClassCode: "basicos_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Title, "Classes of user")
				assert.Contains(t, msg.Embed.Description, "basicos_1")
			},
		},
		{
			name: "successful get classes (raw ID)",
			args: []string{"!classes_of", "123"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{Roles: []string{"role_1"}}, nil)
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return([]data.Class{{ClassCode: "basicos_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Title, "Classes of user")
				assert.Contains(t, msg.Embed.Description, "basicos_1")
			},
		},
		{
			name: "service error",
			args: []string{"!classes_of", "<@123>"},
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{Roles: []string{"role_1"}}, nil)
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return(nil, errors.New("db fail"))
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
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:             tt.args,
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
			}
			HandleGetClassesOfUser(ctx)
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name: "successful get all classes",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetAllClasses().Return([]data.Class{{ClassCode: "basicos_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Title, "All classes I found:")
				assert.Contains(t, msg.Embed.Description, "basicos_1")
			},
		},
		{
			name: "service error",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				m.EXPECT().GetAllClasses().Return(nil, fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
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
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Message:          &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: "chan_123"}},
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
		setupMock  func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:     "successful get my classes",
			authorID: "123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{Roles: []string{"role_1"}}, nil)
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return([]data.Class{{ClassCode: "basicos_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Equal(t, dispatcher.EmbedMessage, msg.Type)
				assert.Contains(t, msg.Embed.Title, "Your classes:")
				assert.Contains(t, msg.Embed.Description, "basicos_1")
			},
		},
		{
			name:     "service error",
			authorID: "123",
			setupMock: func(m *mock_data.MockDataStore, v *mock_providers.MockDiscordValidator, f *mock_providers.MockDiscordFetcher) {
				f.EXPECT().GetCachedMember("guild_123", "123").Return(&discordgo.Member{Roles: []string{"role_1"}}, nil)
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return(nil, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
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
			mockVal := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetch := mock_providers.NewMockDiscordFetcher(ctrl)

			tt.setupMock(mockStore, mockVal, mockFetch)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Services:         svc,
				Dispatcher:       disp,
				DiscordValidator: mockVal,
				DiscordFetcher:   mockFetch,
				ExecutorGuildID:  "guild_123",
				Executor: service_model.User{
					UserID: tt.authorID,
				},
				Message: &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: "chan_123",
						Author: &discordgo.User{
							ID: tt.authorID,
						},
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

func TestHandleGetClassRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		channelID  string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:      "success - explicit class code",
			args:      []string{"!get_roles", "basicos"},
			channelID: "chan_other",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 1, ClassCode: "basicos"}, nil)
				m.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "role_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Roles linked to class `basicos`")
				assert.Contains(t, msg.Content, "role_1")
			},
		},
		{
			name:      "success - contextual (no args)",
			args:      []string{"!get_roles"},
			channelID: "chan_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos", nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 1, ClassCode: "basicos"}, nil)
				m.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "role_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Roles linked to class `basicos`")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)

			tc.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:       tc.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: tc.channelID}},
			}
			HandleGetClassRoles(ctx)
			if tc.assertFunc != nil {
				tc.assertFunc(t, out)
			}
		})
	}
}

func TestHandleGetClassRooms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		channelID  string
		setupMock  func(m *mock_data.MockDataStore)
		assertFunc func(t *testing.T, out chan dispatcher.MessageResponse)
	}{
		{
			name:      "success - explicit class code",
			args:      []string{"!get_channels", "basicos"},
			channelID: "chan_other",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 1, ClassCode: "basicos"}, nil)
				m.EXPECT().GetClassroomsByClassCode("basicos").Return([]data.Classroom{{RoomID: "chan_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Rooms linked to class `basicos`")
				assert.Contains(t, msg.Content, "chan_1")
			},
		},
		{
			name:      "success - contextual (no args)",
			args:      []string{"!get_channels"},
			channelID: "chan_123",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("chan_123").Return("basicos", nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 1, ClassCode: "basicos"}, nil)
				m.EXPECT().GetClassroomsByClassCode("basicos").Return([]data.Classroom{{RoomID: "chan_1"}}, nil)
			},
			assertFunc: func(t *testing.T, out chan dispatcher.MessageResponse) {
				msg := awaitMessage(t, out)
				assert.Contains(t, msg.Content, "Rooms linked to class `basicos`")
				assert.Contains(t, msg.Content, "chan_1")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)

			tc.setupMock(mockStore)
			svc := services.NewServices(mockStore)
			out := make(chan dispatcher.MessageResponse, 1)
			disp := &dispatcher.Dispatcher{Output: out}
			ctx := &context.ContextMessage{
				Args:       tc.args,
				Services:   svc,
				Dispatcher: disp,
				Message:    &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: tc.channelID}},
			}
			HandleGetClassRooms(ctx)
			if tc.assertFunc != nil {
				tc.assertFunc(t, out)
			}
		})
	}
}
