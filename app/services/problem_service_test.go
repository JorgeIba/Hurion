package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/helpers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProblemService_CreateProblemIntoClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		params      service_model.CreateProblemParams
		setupMock   func(*mock_data.MockDataStore)
		wantID      int64
		wantErr     bool
		wantErrType error
	}{
		{
			name: "success - problem created",
			params: service_model.CreateProblemParams{
				Name:       "Two Sum",
				URL:        "https://leetcode.com/problems/two-sum/",
				ClassCode:  "CS101",
				Difficulty: helpers.PtrStr("Easy"),
				Topic:      helpers.PtrStr("Arrays"),
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).Times(2)
				m.EXPECT().AddProblem(data.Problem{
					Name:       "Two Sum",
					URL:        "https://leetcode.com/problems/two-sum/",
					ClassID:    10,
					Difficulty: helpers.PtrStr("Easy"),
					Topic:      helpers.PtrStr("Arrays"),
				}).Return(int64(100), nil)
			},
			wantID:  100,
			wantErr: false,
		},
		{
			name: "class not found in IsClassCodeValid",
			params: service_model.CreateProblemParams{
				ClassCode: "UNKNOWN",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("UNKNOWN").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantID:      0,
			wantErr:     true,
			wantErrType: &serviceerr.ClassCodeNotFoundError{},
		},
		{
			name: "generic error in IsClassCodeValid",
			params: service_model.CreateProblemParams{
				ClassCode: "CS101",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantID:      0,
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name: "generic error in GetClassByClassCode",
			params: service_model.CreateProblemParams{
				ClassCode: "CS101",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantID:      0,
			wantErr:     true,
			wantErrType: &serviceerr.InternalError{},
		},
		{
			name: "conflict - duplicate problem",
			params: service_model.CreateProblemParams{
				ClassCode: "CS101",
				Name:      "Two Sum",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).Times(2)
				m.EXPECT().AddProblem(gomock.Any()).Return(int64(0), &dataerr.RowConflictError{})
			},
			wantID:      0,
			wantErr:     true,
			wantErrType: &serviceerr.DuplicateProblemError{},
		},
		{
			name: "generic error in AddProblem",
			params: service_model.CreateProblemParams{
				ClassCode: "CS101",
				Name:      "Two Sum",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).Times(2)
				m.EXPECT().AddProblem(gomock.Any()).Return(int64(0), errors.New("db fail"))
			},
			wantID:      0,
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

			svc := NewProblemService(mockStore)
			id, err := svc.CreateProblemIntoClass(tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestProblemService_RemoveProblemFromClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		problemID int64
		setupMock func(*mock_data.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success - removed",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().RemoveProblemByID(int64(100)).Return(int64(1), nil)
			},
			wantErr: false,
		},
		{
			name:      "not found",
			problemID: 404,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(404)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error GetProblemByID",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "generic error RemoveProblemByID",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().RemoveProblemByID(int64(100)).Return(int64(0), errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := NewProblemService(mockStore)
			err := svc.RemoveProblemFromClass(tt.problemID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProblemService_UpdateProblemByProblemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		problemID   int64
		problem     service_model.Problem
		setupMock   func(*mock_data.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success",
			problemID: 1,
			problem:   service_model.Problem{Name: "New Name"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), data.Problem{ID: 0, Name: "New Name", ClassID: 0, URL: "", Difficulty: nil, Topic: nil}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "not found",
			problemID: 2,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(2)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.ProblemNotFoundError{},
		},
		{
			name:      "duplicate conflict on update",
			problemID: 1,
			problem:   service_model.Problem{Name: "Repeated Name"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), data.Problem{ID: 0, Name: "Repeated Name", ClassID: 0, URL: "", Difficulty: nil, Topic: nil}).Return(dataerr.NewRowConflictError("problem exists", nil))
			},
			wantErr:     true,
			wantErrType: &serviceerr.DuplicateProblemError{},
		},
		{
			name:      "generic error UpdateProblemByProblemID",
			problemID: 1,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), gomock.Any()).Return(errors.New("db fail"))
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

			svc := NewProblemService(mockStore)
			err := svc.UpdateProblemByProblemID(tt.problemID, tt.problem)

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

func TestProblemService_GetProblemsGroupedByTopic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		wantResult []service_model.ProblemsByTopic
		wantErr    bool
	}{
		{
			name:      "success - grouped",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return([]data.Problem{
					{ID: 1, Topic: helpers.PtrStr("Math"), Difficulty: helpers.PtrStr("1")},
					{ID: 2, Topic: helpers.PtrStr("Math"), Difficulty: helpers.PtrStr("2")},
					{ID: 3, Topic: helpers.PtrStr("Strings"), Difficulty: helpers.PtrStr("1")},
				}, nil)
			},
			wantResult: []service_model.ProblemsByTopic{
				{
					Topic: "Math",
					Problems: []service_model.Problem{
						{ID: 1, Topic: helpers.PtrStr("Math"), Difficulty: helpers.PtrStr("1")},
						{ID: 2, Topic: helpers.PtrStr("Math"), Difficulty: helpers.PtrStr("2")},
					},
				},
				{
					Topic: "Strings",
					Problems: []service_model.Problem{
						{ID: 3, Topic: helpers.PtrStr("Strings"), Difficulty: helpers.PtrStr("1")},
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantResult: []service_model.ProblemsByTopic{},
			wantErr:    true,
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantResult: []service_model.ProblemsByTopic{},
			wantErr:    true,
		},
		{
			name:      "generic error GetProblemsByClass",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: []service_model.ProblemsByTopic{},
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

			svc := NewProblemService(mockStore)
			res, err := svc.GetProblemsGroupedByTopic(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}
