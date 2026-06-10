package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProblemService_CreateProblemIntoClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     service_model.CreateProblemParams
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, int64, error)
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
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(100), id)
			},
		},
		{
			name: "class not found",
			params: service_model.CreateProblemParams{
				ClassCode: "UNKNOWN",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("UNKNOWN").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name: "generic error in IsClassCodeValid",
			params: service_model.CreateProblemParams{
				ClassCode: "CS101",
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
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
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.DuplicateProblemError{}, err)
			},
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
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
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

			svc := NewProblemService(mockStore)
			id, err := svc.CreateProblemIntoClass(tt.params)
			tt.assertFunc(t, id, err)
		})
	}
}

func TestProblemService_RemoveProblemFromClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		problemID  int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - removed",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().RemoveProblemByID(int64(100)).Return(int64(1), nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "not found",
			problemID: 404,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(404)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "generic error GetProblemByID",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "generic error RemoveProblemByID",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().RemoveProblemByID(int64(100)).Return(int64(0), errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, err error) {
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

			svc := NewProblemService(mockStore)
			err := svc.RemoveProblemFromClass(tt.problemID)
			tt.assertFunc(t, err)
		})
	}
}

func TestProblemService_UpdateProblemByProblemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		problemID  int64
		problem    service_model.Problem
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success",
			problemID: 1,
			problem:   service_model.Problem{Name: "New Name"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), data.Problem{Name: "New Name"}).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "not found",
			problemID: 2,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(2)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ProblemNotFoundError{}, err)
			},
		},
		{
			name:      "duplicate conflict on update",
			problemID: 1,
			problem:   service_model.Problem{Name: "Repeated Name"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), data.Problem{Name: "Repeated Name"}).Return(dataerr.NewRowConflictError("problem exists", nil))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.DuplicateProblemError{}, err)
			},
		},
		{
			name:      "generic error UpdateProblemByProblemID",
			problemID: 1,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), gomock.Any()).Return(errors.New("db fail"))
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

			svc := NewProblemService(mockStore)
			err := svc.UpdateProblemByProblemID(tt.problemID, tt.problem)
			tt.assertFunc(t, err)
		})
	}
}

func TestProblemService_GetProblemsGroupedByTopic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.ProblemsByTopic, error)
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
			assertFunc: func(t *testing.T, res []service_model.ProblemsByTopic, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 2)
				assert.Equal(t, "Math", res[0].Topic)
				assert.Len(t, res[0].Problems, 2)
				assert.Equal(t, int64(1), res[0].Problems[0].ID)
				assert.Equal(t, int64(2), res[0].Problems[1].ID)
				assert.Equal(t, "Strings", res[1].Topic)
				assert.Len(t, res[1].Problems, 1)
				assert.Equal(t, int64(3), res[1].Problems[0].ID)
			},
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, res []service_model.ProblemsByTopic, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, res []service_model.ProblemsByTopic, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "generic error GetProblemsByClass",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, res []service_model.ProblemsByTopic, err error) {
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

			svc := NewProblemService(mockStore)
			res, err := svc.GetProblemsGroupedByTopic(tt.classCode)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestProblemService_GetProblemByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		problemID  int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, service_model.Problem, error)
	}{
		{
			name:      "success",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, Name: "P100"}, nil)
			},
			assertFunc: func(t *testing.T, p service_model.Problem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(100), p.ID)
			},
		},
		{
			name:      "not found",
			problemID: 404,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(404)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, p service_model.Problem, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ProblemNotFoundError{}, err)
			},
		},
		{
			name:      "internal error",
			problemID: 500,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(500)).Return(data.Problem{}, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, p service_model.Problem, err error) {
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

			svc := NewProblemService(mockStore)
			p, err := svc.GetProblemByID(tt.problemID)
			tt.assertFunc(t, p, err)
		})
	}
}
