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

func TestUserService_BindUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		userName    string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "success - username available and user bound",
			userID:   "u123",
			userName: "cool_user",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("cool_user", "u123").
					Return(true, nil)

				m.EXPECT().
					BindUser(data.User{UserID: "u123", UserName: "cool_user"}).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "conflict - username already taken",
			userID:   "u1",
			userName: "taken",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("taken", "u1").
					Return(false, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
		},
		{
			name:     "internal error - datastore check fails",
			userID:   "u1",
			userName: "anything",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("anything", "u1").
					Return(false, errors.New("db connection lost"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:     "internal error - bind operation fails",
			userID:   "u2",
			userName: "newuser",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("newuser", "u2").
					Return(true, nil)

				m.EXPECT().
					BindUser(gomock.Any()).
					Return(errors.New("db write failed"))
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

			err := svc.BindUser(tt.userID, tt.userName)

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

func TestUserService_ChangeUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		newUsername string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:        "success - username updated",
			userID:      "u1",
			newUsername: "new_name",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("new_name", "u1").
					Return(true, nil)
				m.EXPECT().
					BindUser(data.User{UserID: "u1", UserName: "new_name"}).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "conflict - username already taken",
			userID:      "u1",
			newUsername: "taken",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("taken", "u1").
					Return(false, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
		},
		{
			name:        "internal error - check fails",
			userID:      "u1",
			newUsername: "name",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("name", "u1").
					Return(false, errors.New("db error"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:        "internal error - update fails",
			userID:      "u1",
			newUsername: "name",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					IsUsernameAvailable("name", "u1").
					Return(true, nil)
				m.EXPECT().
					BindUser(gomock.Any()).
					Return(errors.New("db error"))
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
			err := svc.ChangeUsername(tt.userID, tt.newUsername)

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

func TestUserService_HasUsernameBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mock_data.MockDataStore)
		wantResult bool
		wantErr    bool
	}{
		{
			name:   "is bound",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					HasUsernameBound("u1").
					Return(true, nil)
			},
			wantResult: true,
			wantErr:    false,
		},
		{
			name:   "is not bound",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					HasUsernameBound("u2").
					Return(false, nil)
			},
			wantResult: false,
			wantErr:    false,
		},
		{
			name:   "internal error",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					HasUsernameBound("u1").
					Return(false, errors.New("db error"))
			},
			wantResult: false,
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
			result, err := svc.HasUsernameBound(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
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
				m.EXPECT().
					GetUserByUserID("u1").
					Return(data.User{UserID: "u1", UserName: "user1"}, nil)

				m.EXPECT().
					ExterminateUser("u1").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "not found - user does not exist",
			userID: "nonexistent",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("nonexistent").
					Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.UserNotFoundError{},
		},
		{
			name:   "internal error - get user fails",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("u2").
					Return(data.User{}, errors.New("db error"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:   "internal error - extermination fails",
			userID: "u3",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("u3").
					Return(data.User{UserID: "u3", UserName: "user3"}, nil)

				m.EXPECT().
					ExterminateUser("u3").
					Return(errors.New("deletion failed"))
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

func TestUserService_GetUsernameByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		setupMock   func(*mock_data.MockDataStore)
		wantResult  string
		wantErr     bool
		wantErrType error
	}{
		{
			name:   "success - username retrieved",
			userID: "u1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("u1").
					Return(data.User{UserID: "u1", UserName: "john_doe"}, nil)
			},
			wantResult: "john_doe",
			wantErr:    false,
		},
		{
			name:   "not found - user does not exist",
			userID: "nonexistent",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("nonexistent").
					Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantResult:  "",
			wantErr:     true,
			wantErrType: &serviceerr.UserNotFoundError{},
		},
		{
			name:   "internal error - datastore fails",
			userID: "u2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserID("u2").
					Return(data.User{}, errors.New("database connection lost"))
			},
			wantResult:  "",
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

			result, err := svc.GetUsernameByUserID(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestUserService_GetUserIDByUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		username    string
		setupMock   func(*mock_data.MockDataStore)
		wantResult  string
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "success - found",
			username: "john",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserName("john").
					Return(data.User{UserID: "u123", UserName: "john"}, nil)
			},
			wantResult: "u123",
			wantErr:    false,
		},
		{
			name:     "not found",
			username: "nobody",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserName("nobody").
					Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantResult:  "",
			wantErr:     true,
			wantErrType: &serviceerr.UserNotFoundError{},
		},
		{
			name:     "internal error",
			username: "error_user",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetUserByUserName("error_user").
					Return(data.User{}, errors.New("db error"))
			},
			wantResult:  "",
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
			result, err := svc.GetUserIDByUsername(tt.username)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
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
