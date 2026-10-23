package services

import (
	authtest "github.com/JorgeIba/Hurion/app/authorization/authtest"

	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserService_EnsureUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		user       service_model.User
		setup      func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name: "success",
			user: service_model.User{UserID: "u1", DisplayName: "u1"},
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "u1"}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "datastore error",
			user: service_model.User{UserID: "u1", DisplayName: "u1"},
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "u1"}).Return(errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setup(mockStore)

			svc := &UserService{Datastore: mockStore, Authorizer: authtest.Allow{}}
			err := svc.EnsureUser(tt.user)
			tt.assertFunc(t, err)
		})
	}
}

func TestUserService_ExterminateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:   "success - user exterminated",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", DisplayName: "u1"}, nil)
				m.EXPECT().ExterminateUser("u1").Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:   "not found - user does not exist",
			userID: "nonexistent",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("nonexistent").Return(data.User{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.UserNotFoundError{}, err)
			},
		},
		{
			name:   "internal error - get user fails",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u2").Return(data.User{}, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
		{
			name:   "internal error - extermination fails",
			userID: "u3",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u3").Return(data.User{UserID: "u3", DisplayName: "u3"}, nil)
				m.EXPECT().ExterminateUser("u3").Return(errors.New("deletion failed"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
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

			svc := &UserService{Datastore: mockStore, Authorizer: authtest.Allow{}}
			err := svc.ExterminateUser(authtest.Actor(), tt.userID)
			tt.assertFunc(t, err)
		})
	}
}

func TestUserService_GetUserByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, service_model.User, error)
	}{
		{
			name:   "success",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", DisplayName: "u1"}, nil)
			},
			assertFunc: func(t *testing.T, res service_model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, service_model.User{UserID: "u1", DisplayName: "u1"}, res)
			},
		},
		{
			name:   "not found",
			userID: "none",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("none").Return(data.User{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, res service_model.User, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.UserNotFoundError{}, err)
			},
		},
		{
			name:   "internal error",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u2").Return(data.User{}, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, res service_model.User, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
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

			svc := &UserService{Datastore: mockStore, Authorizer: authtest.Allow{}}
			res, err := svc.GetUserByUserID(tt.userID)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestUserService_GetAccessibleClassesByRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roles      []string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.Class, error)
	}{
		{
			name:  "success - one accessible class",
			roles: []string{"role_1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByAccessRoles([]string{"role_1"}).Return([]data.Class{{ClassCode: "c1", Name: "Class 1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, "c1", res[0].ClassCode)
			},
		},
		{
			name:  "success - multiple accessible classes",
			roles: []string{"role_a", "role_b"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByAccessRoles([]string{"role_a", "role_b"}).Return([]data.Class{
					{ClassCode: "c1"},
					{ClassCode: "c2"},
				}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 2)
			},
		},
		{
			name:  "no access to any class",
			roles: []string{"other"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByAccessRoles([]string{"other"}).Return([]data.Class{}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 0)
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

			svc := &UserService{Datastore: mockStore, Authorizer: authtest.Allow{}}
			res, err := svc.GetAccessibleClassesByRoles(tt.roles)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestUserService_GetClassesWhereUserIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.Class, error)
	}{
		{
			name:   "success",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesWhereUserIsActive("u1").Return([]data.Class{{ClassCode: "c1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, "c1", res[0].ClassCode)
			},
		},
		{
			name:   "datastore error",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesWhereUserIsActive("u1").Return(nil, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.Error(t, err)
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

			svc := &UserService{Datastore: mockStore, Authorizer: authtest.Allow{}}
			res, err := svc.GetClassesWhereUserIsActive(tt.userID)
			tt.assertFunc(t, res, err)
		})
	}
}
