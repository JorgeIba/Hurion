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

func TestClassService_CreateClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		className  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, int64, error)
	}{
		{
			name:      "success - class created",
			classCode: "CS101",
			className: "Intro to CS",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					CreateClass(data.Class{ClassCode: "CS101", Name: "Intro to CS"}).
					Return(int64(1), nil)
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), id)
			},
		},
		{
			name:      "conflict - class already exists",
			classCode: "CS101",
			className: "Intro to CS",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					CreateClass(gomock.Any()).
					Return(int64(0), &dataerr.RowConflictError{})
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ConflictError{}, err)
			},
		},
		{
			name:      "internal error - datastore fails",
			classCode: "CS101",
			className: "Intro to CS",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					CreateClass(gomock.Any()).
					Return(int64(0), errors.New("db error"))
			},
			assertFunc: func(t *testing.T, id int64, err error) {
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

			svc := &ClassService{Datastore: mockStore}
			id, err := svc.CreateClass(tt.classCode, tt.className)
			tt.assertFunc(t, id, err)
		})
	}
}

func TestClassService_ClearClassContentsByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - class contents cleared",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					ClearClassContents(int64(1)).
					Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "not found - class doesn't exist",
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:      "internal error - GetClassByClassCode fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{}, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
		{
			name:      "internal error - ClearClassContents fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					ClearClassContents(int64(1)).
					Return(errors.New("db error"))
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.ClearClassContentsByClassCode(tt.classCode)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_DeleteClassByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - class deleted",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					DeleteClass(int64(1)).
					Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "not found - class doesn't exist",
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:      "internal error - GetClassByClassCode fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{}, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
		{
			name:      "internal error - DeleteClass fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					DeleteClass(int64(1)).
					Return(errors.New("db error"))
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.DeleteClassByClassCode(tt.classCode)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_GetAllClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.Class, error)
	}{
		{
			name: "success - retrieved classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetAllClasses().
					Return([]data.Class{{ID: 1, ClassCode: "C1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, "C1", res[0].ClassCode)
			},
		},
		{
			name: "internal error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetAllClasses().
					Return(nil, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, res []service_model.Class, err error) {
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

			svc := &ClassService{Datastore: mockStore}
			res, err := svc.GetAllClasses()
			tt.assertFunc(t, res, err)
		})
	}
}

func TestClassService_GetClassByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, service_model.Class, error)
	}{
		{
			name:      "success",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 1, ClassCode: "CS101", Name: "Intro"}, nil)
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "CS101", class.ClassCode)
			},
		},
		{
			name:      "not found",
			classCode: "VOID",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("VOID").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:      "internal error",
			classCode: "FAIL",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("FAIL").Return(data.Class{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
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

			svc := &ClassService{Datastore: mockStore}
			class, err := svc.GetClassByClassCode(tt.classCode)
			tt.assertFunc(t, class, err)
		})
	}
}

func TestClassService_GetClassIDByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, int64, error)
	}{
		{
			name:      "success - id retrieved",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 42, ClassCode: "CS101"}, nil)
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(42), id)
			},
		},
		{
			name:      "not found",
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:      "generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{}, errors.New("db error"))
			},
			assertFunc: func(t *testing.T, id int64, err error) {
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

			svc := &ClassService{Datastore: mockStore}
			id, err := svc.GetClassIDByClassCode(tt.classCode)
			tt.assertFunc(t, id, err)
		})
	}
}

func TestClassService_GetClassByClassID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classID    int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, service_model.Class, error)
	}{
		{
			name:    "success",
			classID: 42,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassID(int64(42)).Return(data.Class{ID: 42, ClassCode: "C42"}, nil)
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "C42", class.ClassCode)
			},
		},
		{
			name:    "not found",
			classID: 404,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassID(int64(404)).Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name:    "internal error",
			classID: 500,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassID(int64(500)).Return(data.Class{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, class service_model.Class, err error) {
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

			svc := &ClassService{Datastore: mockStore}
			class, err := svc.GetClassByClassID(tt.classID)
			tt.assertFunc(t, class, err)
		})
	}
}

func TestClassService_RegisterClassroom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classroom  service_model.Classroom
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - classroom registered",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("", false, nil)
				m.EXPECT().AddClassroom(data.Classroom{RoomID: "r1", ClassCode: "c1"}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "conflict - room already registered to different class",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("different-class", true, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ConflictError{}, err)
			},
		},
		{
			name:      "success - room already registered to same class",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("c1", true, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "error - class not found",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "missing"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("missing").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.RegisterClassroom(tt.classroom)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_RemoveClassroom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roomID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:   "success",
			roomID: "r1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemoveClassroomByRoomID("r1").Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:   "success - non-existent room",
			roomID: "unknown",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemoveClassroomByRoomID("unknown").Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:   "internal error",
			roomID: "fail",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemoveClassroomByRoomID("fail").Return(errors.New("db fail"))
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.RemoveClassroom(tt.roomID)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_GetClassrooms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.Classroom, error)
	}{
		{
			name:      "success",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassroomsByClassCode("c1").Return([]data.Classroom{{RoomID: "r1", ClassCode: "c1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.Classroom, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
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

			svc := &ClassService{Datastore: mockStore}
			res, err := svc.GetClassrooms(tt.classCode)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestClassService_AssignAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		role       service_model.ClassAccessRole
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name: "success",
			role: service_model.ClassAccessRole{RoleID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().AddClassAccessRole(data.ClassAccessRole{RoleID: "r1", ClassCode: "c1"}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.AssignAccessRole(tt.role)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_RevokeAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		role       service_model.ClassAccessRole
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name: "success",
			role: service_model.ClassAccessRole{RoleID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemoveClassAccessRole(data.ClassAccessRole{RoleID: "r1", ClassCode: "c1"}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.RevokeAccessRole(tt.role)
			tt.assertFunc(t, err)
		})
	}
}

func TestClassService_GetAccessRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.ClassAccessRole, error)
	}{
		{
			name:      "success",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("c1").Return([]data.ClassAccessRole{{RoleID: "r1", ClassCode: "c1"}}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.ClassAccessRole, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 1)
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

			svc := &ClassService{Datastore: mockStore}
			res, err := svc.GetAccessRoles(tt.classCode)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestClassService_HasAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		roleID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, bool, error)
	}{
		{
			name:      "success - role is linked",
			classCode: "c1",
			roleID:    "r1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().HasClassAccessRole("c1", "r1").Return(true, nil)
			},
			assertFunc: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name:      "error - datastore fails",
			classCode: "c1",
			roleID:    "r1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().HasClassAccessRole("c1", "r1").Return(false, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, exists bool, err error) {
				assert.Error(t, err)
				assert.False(t, exists)
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

			svc := &ClassService{Datastore: mockStore}
			exists, err := svc.HasAccessRole(tt.classCode, tt.roleID)
			tt.assertFunc(t, exists, err)
		})
	}
}

func TestClassService_GetClassCodeFromRoom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roomID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, string, bool, error)
	}{
		{
			name:   "success - room is mapped",
			roomID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("c1", nil)
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, "c1", classCode)
			},
		},
		{
			name:   "success - room is not mapped",
			roomID: "room_unknown",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_unknown").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.NoError(t, err)
				assert.False(t, found)
				assert.Equal(t, "", classCode)
			},
		},
		{
			name:   "error - datastore failure",
			roomID: "room_fail",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_fail").Return("", errors.New("db error"))
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.Error(t, err)
				assert.False(t, found)
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

			svc := &ClassService{Datastore: mockStore}
			classCode, found, err := svc.GetClassCodeFromRoom(tt.roomID)
			tt.assertFunc(t, classCode, found, err)
		})
	}
}
