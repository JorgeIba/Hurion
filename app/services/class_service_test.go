package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestClassServiceEnforcesClassCodeLimitForCreationAndSelection(t *testing.T) {
	t.Parallel()
	operations := []struct {
		name string
		call func(*ClassService, string) (int64, error)
	}{
		{name: "create", call: func(s *ClassService, code string) (int64, error) {
			return s.CreateClass(UserActor("actor"), code, "Algorithms")
		}},
		{name: "select class", call: func(s *ClassService, code string) (int64, error) {
			class, err := s.GetClassByClassCode(code)
			return class.ID, err
		}},
		{name: "select class ID", call: (*ClassService).GetClassIDByClassCode},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name  string
				code  string
				valid bool
			}{
				{name: "ASCII at limit", code: strings.Repeat("a", 50), valid: true},
				{name: "Unicode at limit", code: strings.Repeat("🚀", 50), valid: true},
				{name: "ASCII above limit", code: strings.Repeat("a", 51)},
				{name: "Unicode above limit", code: strings.Repeat("🚀", 51)},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					store := mock_data.NewMockDataStore(gomock.NewController(t))
					if test.valid {
						if operation.name == "create" {
							store.EXPECT().CreateClass(data.Class{ClassCode: test.code, Name: "Algorithms"}).Return(int64(7), nil)
						} else {
							store.EXPECT().GetClassByClassCode(test.code).Return(data.Class{ID: 7, ClassCode: test.code}, nil)
						}
					}
					auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
					if operation.name == "create" {
						auth.EXPECT().CanCreateClass(UserActor("actor")).Return(true, nil)
					}
					id, err := operation.call(NewClassService(store, auth), test.code)
					if test.valid {
						require.NoError(t, err)
						assert.Equal(t, int64(7), id)
					} else {
						require.Error(t, err)
						assert.IsType(t, &serviceerr.ValidationError{}, err)
						assert.ErrorContains(t, err, "class code must be at most 50 characters")
						assert.Zero(t, id)
					}
				})
			}
		})
	}
}

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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanCreateClass(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			id, err := svc.CreateClass(UserActor("actor"), tt.classCode, tt.className)
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanClearClass(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			err := svc.ClearClassContentsByClassCode(UserActor("actor"), tt.classCode)
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanDeleteClass(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			err := svc.DeleteClassByClassCode(UserActor("actor"), tt.classCode)
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

			svc := NewClassService(mockStore, nil)
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

			svc := NewClassService(mockStore, nil)
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

			svc := NewClassService(mockStore, nil)
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

			svc := NewClassService(mockStore, nil)
			class, err := svc.GetClassByClassID(tt.classID)
			tt.assertFunc(t, class, err)
		})
	}
}

func TestClassService_LinkClassroom(t *testing.T) {
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanLinkClassResources(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			linked, err := svc.LinkClassroom(UserActor("actor"), tt.classroom)
			assert.Equal(t, tt.name == "success - classroom registered", linked)
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanUnlinkClassResources(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			err := svc.RemoveClassroom(UserActor("actor"), tt.roomID)
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

			svc := NewClassService(mockStore, nil)
			res, err := svc.GetClassrooms(tt.classCode)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestClassService_LinkClassAccessRole(t *testing.T) {
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
				m.EXPECT().HasClassAccessRole("c1", "r1").Return(false, nil)
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanLinkClassResources(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			linked, err := svc.LinkClassAccessRole(UserActor("actor"), tt.role)
			assert.Equal(t, tt.name == "success", linked)
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

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanUnlinkClassResources(UserActor("actor")).Return(true, nil)
			svc := NewClassService(mockStore, auth)
			err := svc.RevokeAccessRole(UserActor("actor"), tt.role)
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

			svc := NewClassService(mockStore, nil)
			res, err := svc.GetAccessRoles(tt.classCode)
			tt.assertFunc(t, res, err)
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

			svc := NewClassService(mockStore, nil)
			classCode, found, err := svc.GetClassCodeFromRoom(tt.roomID)
			tt.assertFunc(t, classCode, found, err)
		})
	}
}

func TestClassMutationsRequirePermissionBeforeAccessingData(t *testing.T) {
	t.Parallel()

	operations := []struct {
		name             string
		expectPermission func(*mock_services.MockAuthorizer, service_model.Actor, error)
		call             func(*ClassService, service_model.Actor) error
	}{
		{
			name: "create",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanCreateClass(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				_, err := s.CreateClass(actor, "A", "Class A")
				return err
			},
		},
		{
			name: "clear",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanClearClass(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				return s.ClearClassContentsByClassCode(actor, "A")
			},
		},
		{
			name: "delete",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanDeleteClass(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				return s.DeleteClassByClassCode(actor, "A")
			},
		},
		{
			name: "register room",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanLinkClassResources(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				_, err := s.LinkClassroom(actor, service_model.Classroom{ClassCode: "A", RoomID: "room"})
				return err
			},
		},
		{
			name: "remove room",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanUnlinkClassResources(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				return s.RemoveClassroom(actor, "room")
			},
		},
		{
			name: "assign role",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanLinkClassResources(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				_, err := s.LinkClassAccessRole(actor, service_model.ClassAccessRole{ClassCode: "A", RoleID: "role"})
				return err
			},
		},
		{
			name: "revoke role",
			expectPermission: func(auth *mock_services.MockAuthorizer, actor service_model.Actor, err error) {
				auth.EXPECT().CanUnlinkClassResources(actor).Return(false, err)
			},
			call: func(s *ClassService, actor service_model.Actor) error {
				return s.RevokeAccessRole(actor, service_model.ClassAccessRole{ClassCode: "A", RoleID: "role"})
			},
		},
	}
	scenarios := []struct {
		name              string
		lookupErr         error
		missingActor      bool
		missingAuthorizer bool
	}{
		{name: "denied"},
		{name: "lookup failed", lookupErr: errors.New("role lookup unavailable")},
		{name: "missing actor", missingActor: true},
		{name: "missing authorizer", missingAuthorizer: true},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			t.Parallel()
			for _, scenario := range scenarios {
				t.Run(scenario.name, func(t *testing.T) {
					t.Parallel()
					ctrl := gomock.NewController(t)
					store := mock_data.NewMockDataStore(ctrl)
					auth := mock_services.NewMockAuthorizer(ctrl)
					actor := UserActor("actor")
					service := NewClassService(store, auth)
					if scenario.missingActor {
						actor = service_model.Actor{}
					} else if scenario.missingAuthorizer {
						service.Authorizer = nil
					} else {
						operation.expectPermission(auth, actor, scenario.lookupErr)
					}

					err := operation.call(service, actor)
					if scenario.lookupErr != nil || scenario.missingAuthorizer || scenario.missingActor {
						assert.IsType(t, &serviceerr.InternalError{}, err)
					} else {
						assert.IsType(t, &serviceerr.PermissionDeniedError{}, err)
					}
				})
			}
		})
	}
}

func TestLinkClassAccessRoleReportsExistingLinkWithoutWriting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		classCode string
		roleID    string
	}{
		{
			name:      "existing role link",
			classCode: "A",
			roleID:    "student",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetClassByClassCode(tt.classCode).Return(data.Class{ClassCode: tt.classCode}, nil)
			store.EXPECT().HasClassAccessRole(tt.classCode, tt.roleID).Return(true, nil)

			auth := mock_services.NewMockAuthorizer(gomock.NewController(t))
			auth.EXPECT().CanLinkClassResources(UserActor("actor")).Return(true, nil)
			linked, err := NewClassService(store, auth).LinkClassAccessRole(
				UserActor("actor"),
				service_model.ClassAccessRole{ClassCode: tt.classCode, RoleID: tt.roleID},
			)

			assert.NoError(t, err)
			assert.False(t, linked)
		})
	}
}

func TestClassResourceLinksAuthorizeOnceAndPreservePartialResults(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		setup func(*mock_data.MockDataStore)
		link  func(*ClassService, service_model.Actor, string, ...string) ([]service_model.ClassResourceLinkResult, error)
	}{
		{
			name: "roles",
			setup: func(store *mock_data.MockDataStore) {
				store.EXPECT().HasClassAccessRole("A", "existing").Return(true, nil)
				store.EXPECT().HasClassAccessRole("A", "failed").Return(false, nil)
				store.EXPECT().AddClassAccessRole(data.ClassAccessRole{ClassCode: "A", RoleID: "failed"}).Return(errors.New("write failed"))
				store.EXPECT().HasClassAccessRole("A", "new").Return(false, nil)
				store.EXPECT().AddClassAccessRole(data.ClassAccessRole{ClassCode: "A", RoleID: "new"}).Return(nil)
			},
			link: (*ClassService).LinkClassAccessRoles,
		},
		{
			name: "rooms",
			setup: func(store *mock_data.MockDataStore) {
				store.EXPECT().IsRoomRegistered("existing").Return("A", true, nil)
				store.EXPECT().IsRoomRegistered("failed").Return("B", true, nil)
				store.EXPECT().IsRoomRegistered("new").Return("", false, nil)
				store.EXPECT().AddClassroom(data.Classroom{ClassCode: "A", RoomID: "new"}).Return(nil)
			},
			link: (*ClassService).LinkClassrooms,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			auth := mock_services.NewMockAuthorizer(ctrl)
			actor := UserActor("admin")
			auth.EXPECT().CanLinkClassResources(actor).Return(true, nil).Times(1)
			store.EXPECT().GetClassByClassCode("A").Return(data.Class{ClassCode: "A"}, nil).Times(1)
			tt.setup(store)

			results, err := tt.link(NewClassService(store, auth), actor, "A", "existing", "failed", "new")

			assert.NoError(t, err)
			if assert.Len(t, results, 3) {
				assert.Equal(t, service_model.ClassResourceLinkResult{ResourceID: "existing"}, results[0])
				assert.Equal(t, "failed", results[1].ResourceID)
				assert.Error(t, results[1].Err)
				assert.False(t, results[1].Linked)
				assert.Equal(t, service_model.ClassResourceLinkResult{ResourceID: "new", Linked: true}, results[2])
			}
		})
	}
}

func TestClassResourceLinkBatchesRejectBeforeWriting(t *testing.T) {
	t.Parallel()

	for _, operation := range []struct {
		name string
		link func(*ClassService, service_model.Actor, string, ...string) ([]service_model.ClassResourceLinkResult, error)
	}{
		{name: "roles", link: (*ClassService).LinkClassAccessRoles},
		{name: "rooms", link: (*ClassService).LinkClassrooms},
	} {
		for _, tt := range []struct {
			name     string
			allowed  bool
			authErr  error
			classErr error
			wantErr  error
		}{
			{name: "denied", wantErr: &serviceerr.PermissionDeniedError{}},
			{name: "authorization unavailable", authErr: errors.New("Discord unavailable"), wantErr: &serviceerr.InternalError{}},
			{name: "missing class", allowed: true, classErr: &dataerr.NotFoundError{}, wantErr: &serviceerr.ClassCodeNotFoundError{}},
			{name: "class lookup failure", allowed: true, classErr: errors.New("database unavailable"), wantErr: &serviceerr.InternalError{}},
		} {
			t.Run(operation.name+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				ctrl := gomock.NewController(t)
				store := mock_data.NewMockDataStore(ctrl)
				auth := mock_services.NewMockAuthorizer(ctrl)
				actor := UserActor("admin")
				auth.EXPECT().CanLinkClassResources(actor).Return(tt.allowed, tt.authErr)
				if tt.allowed {
					store.EXPECT().GetClassByClassCode("A").Return(data.Class{}, tt.classErr)
				}

				results, err := operation.link(NewClassService(store, auth), actor, "A", "one", "two")

				assert.Nil(t, results)
				assert.IsType(t, tt.wantErr, err)
			})
		}
	}
}
