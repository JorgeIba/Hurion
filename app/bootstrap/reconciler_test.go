package bootstrap

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/bootstrap/mock_bootstrap"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestReconciler_Reconcile(t *testing.T) {
	logger.Init()
	t.Parallel()

	actor := services.ConfigReconcilerActor()
	class := service_model.Class{ClassCode: "c1", Name: "Class 1"}
	room := service_model.Classroom{ClassCode: "c1", RoomID: "room1"}
	role := service_model.ClassAccessRole{ClassCode: "c1", RoleID: "role1"}
	cfg := Config{Classes: []ClassConfig{{
		Code: "c1", Name: "Class 1", RoomIDs: []string{"room1"}, RoleIDs: []string{"role1"},
	}}}

	tests := []struct {
		name      string
		setupMock func(*mock_bootstrap.MockClassServiceInterface)
		wantError string
	}{
		{
			name: "creates missing class and links resources",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{}, serviceerr.NewClassCodeNotFoundError("missing", nil))
				m.EXPECT().CreateClass(actor, "c1", "Class 1").Return(int64(1), nil)
				m.EXPECT().LinkClassroom(actor, room).Return(true, nil)
				m.EXPECT().LinkClassAccessRole(actor, role).Return(true, nil)
			},
		},
		{
			name: "services report existing links without rewriting",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(class, nil)
				m.EXPECT().LinkClassroom(actor, room).Return(false, nil)
				m.EXPECT().LinkClassAccessRole(actor, role).Return(false, nil)
			},
		},
		{
			name: "class lookup failure stops reconciliation",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{}, serviceerr.NewInternalError("database unavailable", nil))
			},
			wantError: "failed to check class",
		},
		{
			name: "class creation failure stops before links",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(service_model.Class{}, serviceerr.NewClassCodeNotFoundError("missing", nil))
				m.EXPECT().CreateClass(actor, "c1", "Class 1").Return(int64(0), errors.New("database unavailable"))
			},
			wantError: "failed to create class",
		},
		{
			name: "service rejects room owned by another class",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(class, nil)
				m.EXPECT().LinkClassroom(actor, room).Return(false, serviceerr.NewConflictError("room belongs to another class", nil))
			},
			wantError: "room belongs to another class",
		},
		{
			name: "room failure stops before role links",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(class, nil)
				m.EXPECT().LinkClassroom(actor, room).Return(false, errors.New("database unavailable"))
			},
			wantError: "failed to link room",
		},
		{
			name: "role failure is reported",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(class, nil)
				m.EXPECT().LinkClassroom(actor, room).Return(true, nil)
				m.EXPECT().LinkClassAccessRole(actor, role).Return(false, errors.New("database unavailable"))
			},
			wantError: "failed to link role",
		},
		{
			name: "service permission denial stops before writes",
			setupMock: func(m *mock_bootstrap.MockClassServiceInterface) {
				m.EXPECT().GetClassByClassCode("c1").Return(class, nil)
				m.EXPECT().LinkClassroom(actor, room).Return(false, serviceerr.NewPermissionDeniedError())
			},
			wantError: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mockClassSvc := mock_bootstrap.NewMockClassServiceInterface(ctrl)
			tt.setupMock(mockClassSvc)

			err := NewReconciler(mockClassSvc, actor).Reconcile(cfg)

			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
