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

func TestEnrollmentService_RegisterUserIDIntoClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		user        service_model.User
		classCode   string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - registered",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "User 1"}).Return(nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, nil)
				m.EXPECT().RegisterUserIntoClass("u1", int64(10)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("UNKNOWN").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.ClassCodeNotFoundError{},
		},
		{
			name:      "class check generic error",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:      "ensure user generic error",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "User 1"}).Return(errors.New("db fail"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:      "already registered",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "User 1"}).Return(nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
		},
		{
			name:      "IsUserRegisteredInClass generic error",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "User 1"}).Return(nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, errors.New("db fail"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:      "internal error during registration",
			user:      service_model.User{UserID: "u1", DisplayName: "User 1"},
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().EnsureUser(data.User{UserID: "u1", DisplayName: "User 1"}).Return(nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, nil)
				m.EXPECT().RegisterUserIntoClass("u1", int64(10)).Return(errors.New("db error"))
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

			svc := &EnrollmentService{Datastore: mockStore}
			err := svc.RegisterUserIDIntoClass(tt.user, tt.classCode)

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

func TestEnrollmentService_DeregisterUserIDFromClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		classCode   string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - deregistered",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				// From helpers.IsUserIDEnrolledByClassCode
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				// Re-call in DeregisterUserIDFromClass (service logic)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().DeregisterUserFromClass("u1", int64(10)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "not enrolled",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.UserNotEnrolledError{},
		},
		{
			name:      "error in IsUserIDEnrolledByClassCode helper",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "class not found in DeregisterUserIDFromClass service logic",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				// helper pass
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				// service logic fails
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error in DeregisterUserIDFromClass service logic",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				// helper pass
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				// service logic fails
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "DeregisterUserFromClass fail",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().DeregisterUserFromClass("u1", int64(10)).Return(errors.New("db fail"))
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

			svc := &EnrollmentService{Datastore: mockStore}
			err := svc.DeregisterUserIDFromClass(tt.userID, tt.classCode)

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

func TestEnrollmentService_GetListOfUsersByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		wantResult []service_model.User
		wantErr    bool
	}{
		{
			name:      "success",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{{UserID: "u1", DisplayName: "User 1"}}, nil)
			},
			wantResult: []service_model.User{{UserID: "u1", DisplayName: "User 1"}},
			wantErr:    false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantResult: []service_model.User{},
			wantErr:    true,
		},
		{
			name:      "class generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantResult: []service_model.User{},
			wantErr:    true,
		},
		{
			name:      "GetAllUsersByClassID generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: []service_model.User{},
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

			svc := &EnrollmentService{Datastore: mockStore}
			res, err := svc.GetListOfUsersByClassCode(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}
