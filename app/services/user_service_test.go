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
		name    string
		userID  string
		setup   func(*mock_data.MockDataStore)
		wantErr bool
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser("u1").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "datastore error",
			userID: "u1",
			setup: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser("u1").Return(errors.New("db fail"))
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
			tt.setup(mockStore)

			svc := &UserService{Datastore: mockStore}
			err := svc.EnsureUser(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_ExterminateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:   "success - user exterminated",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", UserName: "u1"}, nil)
				m.EXPECT().ExterminateUser("u1").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "not found - user does not exist",
			userID: "nonexistent",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("nonexistent").Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.UserNotFoundError{},
		},
		{
			name:   "internal error - get user fails",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u2").Return(data.User{}, errors.New("db error"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:   "internal error - extermination fails",
			userID: "u3",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u3").Return(data.User{UserID: "u3", UserName: "u3"}, nil)
				m.EXPECT().ExterminateUser("u3").Return(errors.New("deletion failed"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
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

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_GetUserByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		setupMock   func(*mock_data.MockDataStore)
		wantResult  service_model.User
		wantErr     bool
		wantErrType error
	}{
		{
			name:   "success",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u1").Return(data.User{UserID: "u1", UserName: "u1"}, nil)
			},
			wantResult: service_model.User{UserID: "u1", UserName: "u1"},
			wantErr:    false,
		},
		{
			name:   "not found",
			userID: "none",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("none").Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantResult:  service_model.User{},
			wantErr:     true,
			wantErrType: &serviceerr.UserNotFoundError{},
		},
		{
			name:   "internal error",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetUserByUserID("u2").Return(data.User{}, errors.New("db error"))
			},
			wantResult:  service_model.User{},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
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

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestUserService_GetClassesByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mock_data.MockDataStore)
		wantResult []service_model.Class
		wantErr    bool
	}{
		{
			name:   "success",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("u1").Return([]data.Class{{ID: 1, Name: "C1"}}, nil)
			},
			wantResult: []service_model.Class{{ID: 1, Name: "C1"}},
			wantErr:    false,
		},
		{
			name:   "error",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassesByUserID("u1").Return(nil, errors.New("db fail"))
			},
			wantResult: nil,
			wantErr:    true,
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

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}
