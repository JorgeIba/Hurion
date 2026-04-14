package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserService_EnsureUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setup      func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser("u1").Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:   "datastore error",
			userID: "u1",
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser("u1").Return(errors.New("db fail"))
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

			svc := &UserService{Datastore: mockStore}
			err := svc.EnsureUser(tt.userID)
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
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", UserName: "u1"}, nil)
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
				m.EXPECT().GetUserByUserID("u3").Return(data.User{UserID: "u3", UserName: "u3"}, nil)
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

			svc := &UserService{Datastore: mockStore}
			err := svc.ExterminateUser(tt.userID)
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
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", UserName: "u1"}, nil)
			},
			assertFunc: func(t *testing.T, res service_model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, service_model.User{UserID: "u1", UserName: "u1"}, res)
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

			svc := &UserService{Datastore: mockStore}
			res, err := svc.GetUserByUserID(tt.userID)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestUserService_GetClassesByUserID(t *testing.T) {
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
				m.EXPECT().GetClassesByUserID("u1").Return([]data.Class{{ID: 1, Name: "C1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, []service_model.Class{{ID: 1, Name: "C1"}}, res)
			},
		},
		{
			name:   "error",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("u1").Return(nil, errors.New("db fail"))
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

			svc := &UserService{Datastore: mockStore}
			res, err := svc.GetClassesByUserID(tt.userID)
			tt.assertFunc(t, res, err)
		})
	}
}
