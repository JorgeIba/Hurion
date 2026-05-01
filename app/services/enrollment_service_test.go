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

func TestEnrollmentService_GetListOfUsersByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.User, error)
	}{
		{
			name:      "success",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{{UserID: "u1", DisplayName: "User 1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.User, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, "u1", res[0].UserID)
			},
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, res []service_model.User, err error) {
				assert.Error(t, err)
				assert.Len(t, res, 0)
			},
		},
		{
			name:      "class generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, res []service_model.User, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "GetAllUsersByClassID generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, res []service_model.User, err error) {
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

			svc := &EnrollmentService{Datastore: mockStore}
			res, err := svc.GetListOfUsersByClassCode(tt.classCode)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestEnrollmentService_UserWithRolesHasAccessClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roles      []string
		class      service_model.Class
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, allowed bool, err error)
	}{
		{
			name:  "success - role intersection grants access",
			roles: []string{"role_student", "role_mentor"},
			class: service_model.Class{ClassCode: "CS101"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("CS101").Return([]data.ClassAccessRole{
					{ClassCode: "CS101", RoleID: "role_mentor"},
				}, nil)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name:  "success - no role intersection denies access",
			roles: []string{"role_student"},
			class: service_model.Class{ClassCode: "CS101"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("CS101").Return([]data.ClassAccessRole{
					{ClassCode: "CS101", RoleID: "role_mentor"},
				}, nil)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name:  "success - class with no configured access roles denies by default",
			roles: []string{"role_student"},
			class: service_model.Class{ClassCode: "CS101"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("CS101").Return([]data.ClassAccessRole{}, nil)
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name:  "error - class code is required",
			roles: []string{"role_student"},
			class: service_model.Class{},
			setupMock: func(m *mock_data.MockDataStore) {
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.Error(t, err)
				assert.False(t, allowed)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:  "error - datastore failure",
			roles: []string{"role_student"},
			class: service_model.Class{ClassCode: "CS101"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("CS101").Return(nil, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, allowed bool, err error) {
				assert.Error(t, err)
				assert.False(t, allowed)
				assert.IsType(t, &serviceerr.InternalError{}, err)
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

			svc := &EnrollmentService{Datastore: mockStore}
			allowed, err := svc.UserWithRolesHasAccessClass(tc.roles, tc.class)
			tc.assertFunc(t, allowed, err)
		})
	}
}
