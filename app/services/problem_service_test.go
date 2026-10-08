package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name: "normalizes values before creating the problem",
			params: service_model.CreateProblemParams{
				Name:       " Two Sum ",
				URL:        " https://leetcode.com/problems/two-sum/ ",
				ClassCode:  " CS101 ",
				Difficulty: helpers.PtrStr(" Easy "),
				Topic:      helpers.PtrStr(" Arrays "),
				Priority:   helpers.PtrStr(" RECOMMENDED "),
			},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).Times(2)
				m.EXPECT().AddProblem(data.Problem{
					Name:       "Two Sum",
					URL:        "https://leetcode.com/problems/two-sum/",
					ClassID:    10,
					Difficulty: helpers.PtrStr(service_model.DifficultyEasy),
					Topic:      helpers.PtrStr("Arrays"),
					Priority:   helpers.PtrStr(service_model.PriorityRecommended),
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
				Name:      "Two Sum",
				URL:       "https://example.com/problem",
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
				Name:      "Two Sum",
				URL:       "https://example.com/problem",
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
				URL:       "https://example.com/problem",
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
				Name:      "Two Sum",
				URL:       "https://example.com/problem",
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
				URL:       "https://example.com/problem",
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

func TestCreateProblemIntoClassRejectsInvalidValuesBeforeAccessingDatastore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*service_model.CreateProblemParams)
		reason string
	}{
		{
			name:   "blank problem name",
			mutate: func(p *service_model.CreateProblemParams) { p.Name = " " },
			reason: "problem name is required",
		},
		{
			name:   "blank class code",
			mutate: func(p *service_model.CreateProblemParams) { p.ClassCode = " " },
			reason: "class code is required",
		},
		{
			name:   "unsupported URL scheme",
			mutate: func(p *service_model.CreateProblemParams) { p.URL = "ftp://example.com/problem" },
			reason: "URL must be an absolute HTTP or HTTPS URL",
		},
		{
			name:   "invalid difficulty",
			mutate: func(p *service_model.CreateProblemParams) { p.Difficulty = helpers.PtrStr("impossible") },
			reason: "difficulty",
		},
		{
			name:   "invalid priority",
			mutate: func(p *service_model.CreateProblemParams) { p.Priority = helpers.PtrStr("urgent") },
			reason: "priority",
		},
		{
			name:   "name above the limit",
			mutate: func(p *service_model.CreateProblemParams) { p.Name = strings.Repeat("a", 201) },
			reason: "problem name must be at most 200 characters",
		},
		{
			name:   "URL above the limit",
			mutate: func(p *service_model.CreateProblemParams) { p.URL = "https://example.com/" + strings.Repeat("a", 2000) },
			reason: "URL must be at most 2000 characters",
		},
		{
			name:   "topic above the limit",
			mutate: func(p *service_model.CreateProblemParams) { p.Topic = helpers.PtrStr(strings.Repeat("a", 101)) },
			reason: "topic must be at most 100 characters",
		},
		{
			name:   "class code above the limit",
			mutate: func(p *service_model.CreateProblemParams) { p.ClassCode = strings.Repeat("a", 51) },
			reason: "class code must be at most 50 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := service_model.CreateProblemParams{
				Name: "Two Sum", URL: "https://example.com/problem", ClassCode: "CS101",
			}
			tt.mutate(&params)
			// No datastore calls are expected for invalid input.
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			id, err := NewProblemService(store).CreateProblemIntoClass(params)
			assert.Zero(t, id)
			assert.IsType(t, &serviceerr.ValidationError{}, err)
			assert.ErrorContains(t, err, tt.reason)
			assert.True(t, serviceerr.IsServiceError(err))
		})
	}
}

func TestUpdateProblemByProblemIDRejectsOversizedFieldsBeforeAccessingDatastore(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*service_model.UpdateProblemParams)
		want   string
	}{
		{name: "name", mutate: func(p *service_model.UpdateProblemParams) { p.Name = " " + strings.Repeat("🚀", 201) + " " }, want: "problem name must be at most 200 characters"},
		{name: "URL", mutate: func(p *service_model.UpdateProblemParams) {
			p.URL = " https://example.com/" + strings.Repeat("a", 2000) + " "
		}, want: "URL must be at most 2000 characters"},
		{name: "topic", mutate: func(p *service_model.UpdateProblemParams) {
			p.Topic = helpers.PtrStr(" " + strings.Repeat("é", 101) + " ")
		}, want: "topic must be at most 100 characters"},
		{name: "class code", mutate: func(p *service_model.UpdateProblemParams) {
			p.ClassCode = " " + strings.Repeat("🚀", 51) + " "
		}, want: "class code must be at most 50 characters"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			params := service_model.UpdateProblemParams{ProblemID: 42, ClassCode: "basicos_1", Name: "Two Sum", URL: "https://example.com/problem"}
			test.mutate(&params)
			err := NewProblemService(store).UpdateProblemByProblemID(params)
			require.Error(t, err)
			assert.IsType(t, &serviceerr.ValidationError{}, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestUpdateProblemByProblemIDPreservesValuesAtTextLimits(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		topic *string
	}{
		{name: "unset topic"},
		{name: "topic at the limit", topic: helpers.PtrStr(strings.Repeat("é", 100))},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			prefix := "https://example.com/"
			classCode := strings.Repeat("🚀", 50)
			params := service_model.UpdateProblemParams{ProblemID: 42, ClassCode: " " + classCode + " ", Name: " " + strings.Repeat("🚀", 200) + " ", URL: " " + prefix + strings.Repeat("a", 2000-len(prefix)) + " ", Topic: test.topic}
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42}, nil)
			store.EXPECT().GetClassByClassCode(classCode).Return(data.Class{ID: 7, ClassCode: classCode}, nil)
			store.EXPECT().UpdateProblemByProblemID(int64(42), data.Problem{ID: 42, ClassID: 7, Name: strings.Repeat("🚀", 200), URL: prefix + strings.Repeat("a", 2000-len(prefix)), Topic: test.topic}).Return(nil)
			require.NoError(t, NewProblemService(store).UpdateProblemByProblemID(params))
		})
	}
}

func TestUpdateProblemByProblemIDRejectsInvalidTargetClassesBeforeWriting(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		class       data.Class
		lookupErr   error
		wantErr     any
		wantMessage string
	}{
		{name: "class code above limit", class: data.Class{ID: 7, ClassCode: strings.Repeat("🚀", 51)}, wantErr: &serviceerr.ValidationError{}, wantMessage: "class code must be at most 50 characters"},
		{name: "missing class", lookupErr: &dataerr.NotFoundError{}, wantErr: &serviceerr.ClassCodeNotFoundError{}},
		{name: "class lookup fails", lookupErr: errors.New("db error"), wantErr: &serviceerr.InternalError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42}, nil)
			store.EXPECT().GetClassByClassCode("basicos_1").Return(test.class, test.lookupErr)
			err := NewProblemService(store).UpdateProblemByProblemID(service_model.UpdateProblemParams{ProblemID: 42, ClassCode: "basicos_1", Name: "Two Sum", URL: "https://example.com/problem"})
			require.Error(t, err)
			assert.IsType(t, test.wantErr, err)
			if test.wantMessage != "" {
				assert.ErrorContains(t, err, test.wantMessage)
			}
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
		name      string
		setupMock func(*mock_data.MockDataStore)
		wantErr   error
	}{
		{
			name: "normalizes replacement values before saving",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), data.Problem{
					ID: 1, Name: "New Name", URL: "https://example.com/problem", ClassID: 10,
					Difficulty: helpers.PtrStr("hard"), Topic: helpers.PtrStr("Binary search"), Priority: helpers.PtrStr("recommended"),
				}).Return(nil)
			},
		},
		{
			name: "problem not found",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{}, dataerr.NewNotFoundError("missing problem", nil))
			},
			wantErr: &serviceerr.ProblemNotFoundError{},
		},
		{
			name: "problem lookup error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{}, errors.New("db fail"))
			},
			wantErr: &serviceerr.InternalError{},
		},
		{
			name: "destination class not found",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{}, dataerr.NewNotFoundError("missing class", nil))
			},
			wantErr: &serviceerr.ClassCodeNotFoundError{},
		},
		{
			name: "destination class lookup error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: &serviceerr.InternalError{},
		},
		{
			name: "problem deleted between lookup and update",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), gomock.Any()).Return(dataerr.NewNotFoundError("deleted problem", nil))
			},
			wantErr: &serviceerr.ProblemNotFoundError{},
		},
		{
			name: "duplicate conflict on update",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), gomock.Any()).Return(dataerr.NewRowConflictError("problem exists", nil))
			},
			wantErr: &serviceerr.DuplicateProblemError{},
		},
		{
			name: "update error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().UpdateProblemByProblemID(int64(1), gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: &serviceerr.InternalError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockStore := mock_data.NewMockDataStore(gomock.NewController(t))
			tt.setupMock(mockStore)
			err := NewProblemService(mockStore).UpdateProblemByProblemID(service_model.UpdateProblemParams{
				ProblemID: 1, ClassCode: "basicos", Name: " New Name ", URL: " https://example.com/problem ",
				Difficulty: helpers.PtrStr(" HARD "), Topic: helpers.PtrStr(" Binary search "), Priority: helpers.PtrStr(" RECOMMENDED "),
			})
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.IsType(t, tt.wantErr, err)
			}
		})
	}
}

func TestUpdateProblemRejectsInvalidValuesBeforeAccessingDatastore(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		mutate func(*service_model.UpdateProblemParams)
		reason string
	}{
		{"zero problem ID", func(p *service_model.UpdateProblemParams) { p.ProblemID = 0 }, "problem ID must be positive"},
		{"negative problem ID", func(p *service_model.UpdateProblemParams) { p.ProblemID = -1 }, "problem ID must be positive"},
		{"missing destination class code", func(p *service_model.UpdateProblemParams) { p.ClassCode = "" }, "class code is required"},
		{"blank destination class code", func(p *service_model.UpdateProblemParams) { p.ClassCode = " " }, "class code is required"},
		{"blank name", func(p *service_model.UpdateProblemParams) { p.Name = " " }, "problem name is required"},
		{"relative URL", func(p *service_model.UpdateProblemParams) { p.URL = "/problem" }, "URL must be an absolute HTTP or HTTPS URL"},
		{"unsupported URL", func(p *service_model.UpdateProblemParams) { p.URL = "ftp://example.com/problem" }, "URL must be an absolute HTTP or HTTPS URL"},
		{"invalid difficulty", func(p *service_model.UpdateProblemParams) { p.Difficulty = helpers.PtrStr("extreme") }, "invalid difficulty"},
		{"invalid priority", func(p *service_model.UpdateProblemParams) { p.Priority = helpers.PtrStr("urgent") }, "invalid priority"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := service_model.UpdateProblemParams{ProblemID: 1, ClassCode: "basicos", Name: "Two Sum", URL: "https://example.com/problem"}
			tt.mutate(&params)
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			err := NewProblemService(store).UpdateProblemByProblemID(params)
			assert.IsType(t, &serviceerr.ValidationError{}, err)
			assert.ErrorContains(t, err, tt.reason)
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
					{ID: 4, Topic: helpers.PtrStr("Strings"), Difficulty: new("easy")},
					{ID: 2, Topic: helpers.PtrStr("Math")},
					{ID: 3},
					{ID: 1, Topic: helpers.PtrStr("Strings"), Difficulty: new("hard")},
				}, nil)
			},
			assertFunc: func(t *testing.T, res []service_model.ProblemsByTopic, err error) {
				assert.NoError(t, err)
				assert.Len(t, res, 3)
				assert.Equal(t, "Strings", res[0].Topic)
				assert.Len(t, res[0].Problems, 2)
				assert.Equal(t, int64(1), res[0].Problems[0].ID)
				assert.Equal(t, int64(4), res[0].Problems[1].ID)
				assert.Equal(t, "Math", res[1].Topic)
				assert.Len(t, res[1].Problems, 1)
				assert.Equal(t, int64(2), res[1].Problems[0].ID)
				assert.Equal(t, service_model.DefaultTopic, res[2].Topic)
				assert.Len(t, res[2].Problems, 1)
				assert.Equal(t, int64(3), res[2].Problems[0].ID)
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
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

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
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{
					ID:        100,
					Name:      "P100",
					Priority:  helpers.PtrStr("recommended"),
					CreatedAt: createdAt,
				}, nil)
			},
			assertFunc: func(t *testing.T, p service_model.Problem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(100), p.ID)
				if assert.NotNil(t, p.Priority) {
					assert.Equal(t, service_model.PriorityRecommended, *p.Priority)
				}
				assert.Equal(t, createdAt, p.CreatedAt)
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
