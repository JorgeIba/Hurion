package common

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/customerr/controllerserr"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestResolveClassContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		channelID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, class service_model.Class, err error)
	}{
		{
			name:      "success - class from explicit argument",
			args:      []string{"!list", "basicos_1"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "basicos_1", class.ClassCode)
			},
		},
		{
			name:      "success - class resolved from channel",
			args:      []string{"!list"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("basicos_1", nil)
				m.EXPECT().GetClassByClassCode("basicos_1").Return(data.Class{ID: 1, ClassCode: "basicos_1"}, nil)
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "basicos_1", class.ClassCode)
			},
		},
		{
			name:      "error - channel has no class mapping",
			args:      []string{"!list"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.Error(t, err)
				assert.Equal(t, service_model.Class{}, class)
				assert.IsType(t, &serviceerr.NotFoundInfoError{}, err)
			},
		},
		{
			name:      "error - invalid class code fails immediately",
			args:      []string{"!list", "invalid_class"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				// Explicit lookup fails, no fallback
				m.EXPECT().GetClassByClassCode("invalid_class").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.Error(t, err)
				assert.Equal(t, service_model.Class{}, class)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:      "error - missing channel context",
			args:      []string{"!list"},
			channelID: "",
			setupMock: func(m *mock_data.MockDataStore) {},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.Error(t, err)
				assert.Equal(t, service_model.Class{}, class)
				assert.IsType(t, &controllerserr.ValidationError{}, err)
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

			ctx := &context.ContextMessage{
				Args:     tc.args,
				Services: services.NewServices(mockStore),
				Message:  &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: tc.channelID}},
			}

			class, err := ResolveClassContext(ctx, 1)
			tc.assertFunc(t, class, err)
		})
	}
}

func TestResolveClassCodeArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		channelID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, got any)
	}{
		{
			name:      "returns explicit class argument",
			args:      []string{"!score", "basicos_1"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {},
			assertFunc: func(t *testing.T, got any) {
				assert.Equal(t, "basicos_1", got)
			},
		},
		{
			name:      "returns room linked class code for contextual command",
			args:      []string{"!score"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("basicos_1", nil)
			},
			assertFunc: func(t *testing.T, got any) {
				assert.Equal(t, "basicos_1", got)
			},
		},
		{
			name:      "returns nil without channel context",
			args:      []string{"!score"},
			channelID: "",
			setupMock: func(m *mock_data.MockDataStore) {},
			assertFunc: func(t *testing.T, got any) {
				assert.Nil(t, got)
			},
		},
		{
			name:      "returns nil when room has no class mapping",
			args:      []string{"!score"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, got any) {
				assert.Nil(t, got)
			},
		},
		{
			name:      "returns nil when room lookup fails",
			args:      []string{"!score"},
			channelID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("", fmt.Errorf("db error"))
			},
			assertFunc: func(t *testing.T, got any) {
				assert.Nil(t, got)
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

			ctx := &context.ContextMessage{
				Args:     tc.args,
				Services: services.NewServices(mockStore),
				Message:  &discordgo.MessageCreate{Message: &discordgo.Message{ChannelID: tc.channelID}},
			}

			got := ResolveClassCodeArg(ctx, 1)
			tc.assertFunc(t, got)
		})
	}
}

func TestRequireClassAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		executorID string
		guildID    string
		class      service_model.Class
		setupMock  func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore)
		assertFunc func(t *testing.T, allowed bool, err error)
	}{
		{
			name:       "success - admin bypass",
			executorID: "admin_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin(gomock.Any(), "admin_1").Return(true, nil)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name:       "success - user has access role",
			executorID: "user_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin(gomock.Any(), "user_1").Return(false, nil)
				fetcher.EXPECT().GetCachedMember(gomock.Any(), "user_1").Return(&discordgo.Member{Roles: []string{"role_mentor"}}, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos_1").Return(
					[]data.ClassAccessRole{{ClassCode: "basicos_1", RoleID: "role_mentor"}},
					nil,
				)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name:       "success - user does not have access role",
			executorID: "user_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin(gomock.Any(), "user_1").Return(false, nil)
				fetcher.EXPECT().GetCachedMember(gomock.Any(), "user_1").Return(&discordgo.Member{Roles: []string{"role_student"}}, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos_1").Return(
					[]data.ClassAccessRole{{ClassCode: "basicos_1", RoleID: "role_mentor"}},
					nil,
				)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name:       "success - fallback to live fetch works",
			executorID: "user_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin(gomock.Any(), "user_1").Return(false, nil)
				// 1. Cache Miss
				fetcher.EXPECT().GetCachedMember(gomock.Any(), "user_1").Return(nil, fmt.Errorf("cache miss"))
				// 2. API Success
				fetcher.EXPECT().FetchMember(gomock.Any(), "user_1").Return(&discordgo.Member{Roles: []string{"role_mentor"}}, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos_1").Return(
					[]data.ClassAccessRole{{ClassCode: "basicos_1", RoleID: "role_mentor"}},
					nil,
				)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name:       "error - cached and live fetch both fail",
			executorID: "user_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin(gomock.Any(), "user_1").Return(false, nil)
				// 1. Cache Miss
				fetcher.EXPECT().GetCachedMember(gomock.Any(), "user_1").Return(nil, fmt.Errorf("cache miss"))
				// 2. API Fail
				fetcher.EXPECT().FetchMember(gomock.Any(), "user_1").Return(nil, fmt.Errorf("api fail"))
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.Error(t, err)
				assert.False(t, allowed)
				assert.IsType(t, &controllerserr.InternalError{}, err)
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
			mockFetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			mockValidator := mock_providers.NewMockDiscordValidator(ctrl)
			tc.setupMock(mockFetcher, mockValidator, mockStore)

			ctx := &context.ContextMessage{
				Executor:         service_model.User{UserID: tc.executorID},
				ExecutorGuildID:  tc.guildID,
				Services:         services.NewServices(mockStore),
				DiscordFetcher:   mockFetcher,
				DiscordValidator: mockValidator,
			}

			allowed, err := RequireClassAccess(ctx, tc.class)
			tc.assertFunc(t, allowed, err)
		})
	}
}

func TestSyncParticipants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		executor  service_model.User
		setupMock func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore)
	}{
		{
			name:     "success - sync executor only (no other args)",
			args:     []string{"!help"},
			executor: service_model.User{UserID: "exec_1", DisplayName: "Exec"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore) {
				store.EXPECT().EnsureUser(data.User{UserID: "exec_1", DisplayName: "Exec"}).Return(nil)
			},
		},
		{
			name:     "success - sync executor and mentioned user",
			args:     []string{"!ac", "<@111111111111111111>", "101"},
			executor: service_model.User{UserID: "exec_1", DisplayName: "Exec"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore) {
				store.EXPECT().EnsureUser(data.User{UserID: "exec_1", DisplayName: "Exec"}).Return(nil)

				fetcher.EXPECT().GetCachedMember("guild_1", "111111111111111111").Return(&discordgo.Member{
					User: &discordgo.User{Username: "Target User"},
				}, nil)
				store.EXPECT().EnsureUser(data.User{UserID: "111111111111111111", DisplayName: "Target User"}).Return(nil)
			},
		},
		{
			name:     "success - sync long numeric ID",
			args:     []string{"!exterminate", "123456789012345678"},
			executor: service_model.User{UserID: "exec_1", DisplayName: "Exec"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore) {
				store.EXPECT().EnsureUser(data.User{UserID: "exec_1", DisplayName: "Exec"}).Return(nil)

				fetcher.EXPECT().GetCachedMember("guild_1", "123456789012345678").Return(&discordgo.Member{
					User: &discordgo.User{Username: "snowflake_user"},
				}, nil)
				store.EXPECT().EnsureUser(data.User{UserID: "123456789012345678", DisplayName: "snowflake_user"}).Return(nil)
			},
		},
		{
			name:     "skip - short problem ID",
			args:     []string{"!ac", "101"},
			executor: service_model.User{UserID: "exec_1", DisplayName: "Exec"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore) {
				store.EXPECT().EnsureUser(data.User{UserID: "exec_1", DisplayName: "Exec"}).Return(nil)
			},
		},
		{
			name:     "fallback - user left server",
			args:     []string{"!ac", "<@222222222222222222>", "101"},
			executor: service_model.User{UserID: "exec_1", DisplayName: "Exec"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, store *mock_data.MockDataStore) {
				store.EXPECT().EnsureUser(data.User{UserID: "exec_1", DisplayName: "Exec"}).Return(nil)

				fetcher.EXPECT().GetCachedMember("guild_1", "222222222222222222").Return(nil, fmt.Errorf("not found"))
				fetcher.EXPECT().FetchMember("guild_1", "222222222222222222").Return(nil, fmt.Errorf("not found"))

				store.EXPECT().EnsureUser(data.User{UserID: "222222222222222222", DisplayName: "Unknown"}).Return(nil)
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
			mockFetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			tc.setupMock(mockFetcher, mockStore)

			ctx := &context.ContextMessage{
				Args:            tc.args,
				Executor:        tc.executor,
				ExecutorGuildID: "guild_1",
				Services:        services.NewServices(mockStore),
				DiscordFetcher:  mockFetcher,
			}

			SyncParticipants(ctx)
		})
	}
}

func TestIsValidClassCode(t *testing.T) {
	t.Parallel()

	assert.True(t, IsValidClassCode("basicos_1"))
	assert.True(t, IsValidClassCode("power-class"))
	assert.True(t, IsValidClassCode("Class123"))

	assert.False(t, IsValidClassCode("basic/1"))
	assert.False(t, IsValidClassCode("basic\\1"))
	assert.False(t, IsValidClassCode("basic 1"))
	assert.False(t, IsValidClassCode("basic@1"))
	assert.False(t, IsValidClassCode(""))
}
