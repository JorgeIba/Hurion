package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestClassService_CreateClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		classCode   string
		className   string
		setupMock   func(*mocks.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - class created",
			classCode: "CS101",
			className: "Intro to CS",
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
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

			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &ClassService{Datastore: mockStore}
			err := svc.CreateClass(tt.classCode, tt.className)

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
		setupMock   func(*mocks.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - class deleted",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().
					GetClassByClassCode("UNKNOWN").
					Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.NotFoundInfoError{},
		},
		{
			name:      "internal error - GetClassByClassCode fails",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
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

			mockStore := mocks.NewMockDataStore(ctrl)
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
		setupMock  func(*mocks.MockDataStore)
		wantResult []data.Class
		wantErr    bool
	}{
		{
			name: "success - retrieved classes",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().
					GetAllClasses().
					Return([]data.Class{{ID: 1, ClassCode: "C1"}}, nil)
			},
			wantResult: []data.Class{{ID: 1, ClassCode: "C1"}},
			wantErr:    false,
		},
		{
			name: "internal error",
			setupMock: func(m *mocks.MockDataStore) {
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

			mockStore := mocks.NewMockDataStore(ctrl)
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
		setupMock  func(*mocks.MockDataStore)
		wantResult int64
		wantErr    bool
	}{
		{
			name:      "success - id retrieved",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
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
			setupMock: func(m *mocks.MockDataStore) {
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

			mockStore := mocks.NewMockDataStore(ctrl)
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
