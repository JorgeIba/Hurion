package common

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/controllerserr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers/mock_providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
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
				validator.EXPECT().IsAdmin("guild_1", "admin_1").Return(true, nil)
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
				validator.EXPECT().IsAdmin("guild_1", "user_1").Return(false, nil)
				fetcher.EXPECT().GetCachedMember("guild_1", "user_1").Return(&discordgo.Member{Roles: []string{"role_mentor"}}, nil)
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
				validator.EXPECT().IsAdmin("guild_1", "user_1").Return(false, nil)
				fetcher.EXPECT().GetCachedMember("guild_1", "user_1").Return(&discordgo.Member{Roles: []string{"role_student"}}, nil)
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
			name:       "error - cached member fetch fails",
			executorID: "user_1",
			guildID:    "guild_1",
			class:      service_model.Class{ClassCode: "basicos_1"},
			setupMock: func(fetcher *mock_providers.MockDiscordFetcher, validator *mock_providers.MockDiscordValidator, store *mock_data.MockDataStore) {
				validator.EXPECT().IsAdmin("guild_1", "user_1").Return(false, nil)
				fetcher.EXPECT().GetCachedMember("guild_1", "user_1").Return(nil, fmt.Errorf("cache miss"))
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
