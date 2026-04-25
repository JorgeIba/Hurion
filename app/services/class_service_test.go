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
		name        string
		classCode   string
		className   string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
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
			wantErr: false,
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
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
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

			svc := &ClassService{Datastore: mockStore}
			_, err := svc.CreateClass(tt.classCode, tt.className)

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

func TestClassService_DeleteClassByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		classCode   string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - class deleted",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					ClearClass(int64(1)).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "not found - class doesn't exist",
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.ClassCodeNotFoundError{},
		},
		{
			name:      "internal error - GetClassByClassCode fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{}, errors.New("db error"))
			},
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name:      "internal error - ClearClass fails",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 1, ClassCode: "CS101"}, nil)
				m.EXPECT().
					ClearClass(int64(1)).
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

			svc := &ClassService{Datastore: mockStore}
			err := svc.DeleteClassByClassCode(tt.classCode)

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

func TestClassService_GetAllClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*mock_data.MockDataStore)
		wantResult []service_model.Class
		wantErr    bool
	}{
		{
			name: "success - retrieved classes",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetAllClasses().
					Return([]data.Class{{ID: 1, ClassCode: "C1"}}, nil)
			},
			wantResult: []service_model.Class{{ID: 1, ClassCode: "C1"}},
			wantErr:    false,
		},
		{
			name: "internal error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetAllClasses().
					Return(nil, errors.New("db error"))
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

			svc := &ClassService{Datastore: mockStore}
			res, err := svc.GetAllClasses()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestClassService_GetClassIDByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		wantResult int64
		wantErr    bool
	}{
		{
			name:      "success - id retrieved",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{ID: 123}, nil)
			},
			wantResult: 123,
			wantErr:    false,
		},
		{
			name:      "not found",
			classCode: "UNKNOWN",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantResult: 0,
			wantErr:    true,
		},
		{
			name:      "generic error",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("CS101").
					Return(data.Class{}, errors.New("db error"))
			},
			wantResult: 0,
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

			svc := &ClassService{Datastore: mockStore}
			res, err := svc.GetClassIDByClassCode(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestClassService_RegisterClassroom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		classroom   service_model.Classroom
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - classroom registered",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("", false, nil)
				m.EXPECT().AddClassroom(data.Classroom{RoomID: "r1", ClassCode: "c1"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "conflict - room already registered to different class",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("different-class", true, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
		},
		{
			name:      "conflict - room already registered to same class",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().IsRoomRegistered("r1").Return("c1", true, nil)
			},
			wantErr: false,
		},
		{
			name:      "error - class not found",
			classroom: service_model.Classroom{RoomID: "r1", ClassCode: "missing"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("missing").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.ClassCodeNotFoundError{},
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

func TestClassService_GetClassrooms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		wantResult []service_model.Classroom
		wantErr    bool
	}{
		{
			name:      "success - retrieved classrooms",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassroomsByClassCode("c1").Return([]data.Classroom{
					{RoomID: "r1", ClassCode: "c1"},
				}, nil)
			},
			wantResult: []service_model.Classroom{{RoomID: "r1", ClassCode: "c1"}},
			wantErr:    false,
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

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestClassService_AssignAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		role        service_model.ClassAccessRole
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name: "success - role assigned",
			role: service_model.ClassAccessRole{RoleID: "r1", ClassCode: "c1"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
				m.EXPECT().AddClassAccessRole(data.ClassAccessRole{RoleID: "r1", ClassCode: "c1"}).Return(nil)
			},
			wantErr: false,
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

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClassService_GetClassCodeFromRoom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roomID     string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, classCode string, found bool, err error)
	}{
		{
			name:   "success - room is mapped",
			roomID: "room_1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_1").Return("CS101", nil)
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, "CS101", classCode)
			},
		},
		{
			name:   "success - room is not mapped",
			roomID: "room_2",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_2").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.NoError(t, err)
				assert.False(t, found)
				assert.Equal(t, "", classCode)
			},
		},
		{
			name:   "error - datastore failure",
			roomID: "room_3",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("room_3").Return("", errors.New("db error"))
			},
			assertFunc: func(t *testing.T, classCode string, found bool, err error) {
				assert.Error(t, err)
				assert.False(t, found)
				assert.Equal(t, "", classCode)
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

			svc := &ClassService{Datastore: mockStore}
			classCode, found, err := svc.GetClassCodeFromRoom(tc.roomID)
			tc.assertFunc(t, classCode, found, err)
		})
	}
}

func TestClassService_GetAccessRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		classCode   string
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - retrieved roles",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("c1").Return([]data.ClassAccessRole{
					{ClassCode: "c1", RoleID: "r1"},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:      "success - no roles for class returns empty list",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("c1").Return([]data.ClassAccessRole{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "internal error",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassAccessRolesByClassCode("c1").Return(nil, errors.New("db error"))
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

			svc := &ClassService{Datastore: mockStore}
			_, err := svc.GetAccessRoles(tt.classCode)

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
