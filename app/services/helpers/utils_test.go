package helpers

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func ptrStr(s string) *string { return &s }

func TestExistsProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "exists",
			problemID: 1,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "not found",
			problemID: 2,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(2)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "db error",
			problemID: 3,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(3)).Return(data.Problem{}, errors.New("db error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := ExistsProblem(mockStore, tt.problemID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsClassCodeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		classCode string
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "valid",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 1}, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "invalid",
			classCode: "NONE",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "db error",
			classCode: "ERR",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("ERR").Return(data.Class{}, errors.New("db fail"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := IsClassCodeValid(mockStore, tt.classCode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsUserIDEnrolledByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		classCode string
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "success - enrolled",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "class not found",
			userID:    "u1",
			classCode: "NONE",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: true,
		},
		{
			name:      "generic error GetClassByClassCode",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := IsUserIDEnrolledByClassCode(mockStore, tt.userID, tt.classCode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCanBroACTheProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "can AC",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{ID: 10, ClassID: 1}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(10)).Return(false, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "problem not found",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: true, // It returns error if problem not found according to implementation
		},
		{
			name:      "not enrolled",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{ID: 10, ClassID: 1}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(1)).Return(false, nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "already solved",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{ID: 10, ClassID: 1}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(10)).Return(true, nil)
			},
			want:    false,
			wantErr: true, // It returns serviceerr.ConflictError
		},
		{
			name:      "IsUserIDEnrolledByClassID generic error",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{ID: 10, ClassID: 1}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(1)).Return(false, errors.New("db fail"))
			},
			want:    false,
			wantErr: true,
		},
		{
			name:      "HasUserSolvedProblem generic error",
			userID:    "u1",
			problemID: 10,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(10)).Return(data.Problem{ID: 10, ClassID: 1}, nil)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(1)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(10)).Return(false, errors.New("db fail"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := CanBroACTheProblem(mockStore, tt.userID, tt.problemID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestHasUserSolvedProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "solved",
			userID:    "u1",
			problemID: 1,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().HasUserSolvedProblem("u1", int64(1)).Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "not solved",
			userID:    "u1",
			problemID: 1,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().HasUserSolvedProblem("u1", int64(1)).Return(false, nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "db error",
			userID:    "u1",
			problemID: 1,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().HasUserSolvedProblem("u1", int64(1)).Return(false, errors.New("db fail"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := HasUserSolvedProblem(mockStore, tt.userID, tt.problemID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
