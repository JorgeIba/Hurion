package bootstrap

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bootstrap/mock_bootstrap"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestReconciler_Reconcile(t *testing.T) {
	logger.Init()
	t.Parallel()

	notFoundClassErr := &serviceerr.ClassCodeNotFoundError{}

	tests := []struct {
		name       string
		cfg        Config
		setupMock  func(*mock_bootstrap.MockClassServiceInterface)
		assertFunc func(*testing.T, error)
	}{
		{
			name: "success - all missing, creates everything",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code:    "c1",
						Name:    "Class 1",
						RoomIDs: []string{"room1"},
						RoleIDs: []string{"role1"},
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{}, notFoundClassErr)
				m.EXPECT().CreateClass("c1", "Class 1").Return(int64(1), nil)

				m.EXPECT().GetClassCodeFromRoom("room1").Return("", false, nil)
				m.EXPECT().RegisterClassroom(service_model.Classroom{
					ClassCode: "c1",
					RoomID:    "room1",
				}).Return(nil)

				m.EXPECT().GetAccessRoles("c1").Return([]service_model.ClassAccessRole{}, nil)
				m.EXPECT().AssignAccessRole(service_model.ClassAccessRole{
					ClassCode: "c1",
					RoleID:    "role1",
				}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "success - everything already exists, skips everything",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code:    "c1",
						Name:    "Class 1",
						RoomIDs: []string{"room1"},
						RoleIDs: []string{"role1"},
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{ClassCode: "c1"}, nil)

				m.EXPECT().GetClassCodeFromRoom("room1").Return("c1", true, nil)

				m.EXPECT().GetAccessRoles("c1").Return([]service_model.ClassAccessRole{
					{ClassCode: "c1", RoleID: "role1"},
				}, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error - room linked to different class",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code:    "c1",
						Name:    "Class 1",
						RoomIDs: []string{"room1"},
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{ClassCode: "c1"}, nil)
				m.EXPECT().GetClassCodeFromRoom("room1").Return("other_class", true, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "room \"room1\" is already linked to a different class")
			},
		},
		{
			name: "error - create class fails (db error)",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code: "c1",
						Name: "Class 1",
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{}, notFoundClassErr)
				m.EXPECT().CreateClass("c1", "Class 1").Return(int64(0), errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create class")
			},
		},
		{
			name: "error - link room fails",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code:    "c1",
						Name:    "Class 1",
						RoomIDs: []string{"room1"},
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{ClassCode: "c1"}, nil)
				m.EXPECT().GetClassCodeFromRoom("room1").Return("", false, nil)
				m.EXPECT().RegisterClassroom(gomock.Any()).Return(errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to link room")
			},
		},
		{
			name: "error - link role fails",
			cfg: Config{
				Classes: []ClassConfig{
					{
						Code:    "c1",
						Name:    "Class 1",
						RoleIDs: []string{"role1"},
					},
				},
			},
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{ClassCode: "c1"}, nil)
				m.EXPECT().GetAccessRoles("c1").Return([]service_model.ClassAccessRole{}, nil)
				m.EXPECT().AssignAccessRole(gomock.Any()).Return(errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to link role")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClassSvc := mock_bootstrap.NewMockClassServiceInterface(ctrl)
			tc.setupMock(mockClassSvc)

			r := NewReconciler(mockClassSvc)
			err := r.Reconcile(tc.cfg)
			tc.assertFunc(t, err)
		})
	}
}
