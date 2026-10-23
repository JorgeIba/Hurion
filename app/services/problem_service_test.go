package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	authtest "github.com/JorgeIba/Hurion/app/authorization/authtest"
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
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
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
			name: "class lookup fails",
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
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().AddProblem(gomock.Any()).Return(int64(0), &dataerr.RowConflictError{})
			},
			assertFunc: func(t *testing.T, id int64, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.DuplicateProblemError{}, err)
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
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
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
			mockStore.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			tt.setupMock(mockStore)

			svc := NewProblemService(mockStore, authtest.Allow{})
			id, err := svc.CreateProblemIntoClass(authtest.Actor(), tt.params)
			tt.assertFunc(t, id, err)
		})
	}
}

func TestPrepareCreationDoesNotWrite(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
	prepared, err := NewProblemService(store, authtest.Allow{}).PrepareCreation(authtest.Actor(), service_model.CreateProblemParams{ClassCode: " basicos ", Name: " Two Sum ", URL: " https://example.com/problem "})
	if assert.NoError(t, err) {
		assert.Equal(t, "Two Sum", prepared.Params.Name)
		assert.Equal(t, "basicos", prepared.Params.ClassCode)
		assert.Equal(t, int64(10), prepared.Class.ID)
		assert.Zero(t, prepared.Entry)
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
			store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			id, err := NewProblemService(store, authtest.Allow{}).CreateProblemIntoClass(authtest.Actor(), params)
			assert.Zero(t, id)
			assert.IsType(t, &serviceerr.ValidationError{}, err)
			assert.ErrorContains(t, err, tt.reason)
			assert.True(t, serviceerr.IsServiceError(err))
		})
	}
}

func TestReplaceProblemByProblemIDRejectsOversizedFieldsBeforeAccessingDatastore(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*service_model.ReplaceProblemParams)
		want   string
	}{
		{name: "name", mutate: func(p *service_model.ReplaceProblemParams) { p.Name = " " + strings.Repeat("🚀", 201) + " " }, want: "problem name must be at most 200 characters"},
		{name: "URL", mutate: func(p *service_model.ReplaceProblemParams) {
			p.URL = " https://example.com/" + strings.Repeat("a", 2000) + " "
		}, want: "URL must be at most 2000 characters"},
		{name: "topic", mutate: func(p *service_model.ReplaceProblemParams) {
			p.Topic = helpers.PtrStr(" " + strings.Repeat("é", 101) + " ")
		}, want: "topic must be at most 100 characters"},
		{name: "class code", mutate: func(p *service_model.ReplaceProblemParams) {
			p.ClassCode = " " + strings.Repeat("🚀", 51) + " "
		}, want: "class code must be at most 50 characters"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			params := service_model.ReplaceProblemParams{ProblemID: 42, ClassCode: "basicos_1", Name: "Two Sum", URL: "https://example.com/problem"}
			test.mutate(&params)
			err := NewProblemService(store, authtest.Allow{}).ReplaceProblemByProblemID(authtest.Actor(), params)
			require.Error(t, err)
			assert.IsType(t, &serviceerr.ValidationError{}, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestReplaceProblemByProblemIDPreservesValuesAtTextLimits(t *testing.T) {
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
			params := service_model.ReplaceProblemParams{ProblemID: 42, ClassCode: " " + classCode + " ", Name: " " + strings.Repeat("🚀", 200) + " ", URL: " " + prefix + strings.Repeat("a", 2000-len(prefix)) + " ", Topic: test.topic}
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42}, nil)
			store.EXPECT().GetClassByClassCode(classCode).Return(data.Class{ID: 7, ClassCode: classCode}, nil)
			store.EXPECT().ReplaceProblemByProblemID(int64(42), data.Problem{ID: 42, ClassID: 7, Name: strings.Repeat("🚀", 200), URL: prefix + strings.Repeat("a", 2000-len(prefix)), Topic: test.topic}).Return(nil)
			require.NoError(t, NewProblemService(store, authtest.Allow{}).ReplaceProblemByProblemID(authtest.Actor(), params))
		})
	}
}

func TestReplaceProblemByProblemIDRejectsInvalidTargetClassesBeforeWriting(t *testing.T) {
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
			store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42}, nil)
			store.EXPECT().GetClassByClassCode("basicos_1").Return(test.class, test.lookupErr)
			err := NewProblemService(store, authtest.Allow{}).ReplaceProblemByProblemID(authtest.Actor(), service_model.ReplaceProblemParams{ProblemID: 42, ClassCode: "basicos_1", Name: "Two Sum", URL: "https://example.com/problem"})
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
			mockStore.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			tt.setupMock(mockStore)

			svc := NewProblemService(mockStore, authtest.Allow{})
			err := svc.RemoveProblemFromClass(authtest.Actor(), tt.problemID)
			tt.assertFunc(t, err)
		})
	}
}

func TestProblemService_ReplaceProblemByProblemID(t *testing.T) {
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
				m.EXPECT().ReplaceProblemByProblemID(int64(1), data.Problem{
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
				m.EXPECT().ReplaceProblemByProblemID(int64(1), gomock.Any()).Return(dataerr.NewNotFoundError("deleted problem", nil))
			},
			wantErr: &serviceerr.ProblemNotFoundError{},
		},
		{
			name: "duplicate conflict on update",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().ReplaceProblemByProblemID(int64(1), gomock.Any()).Return(dataerr.NewRowConflictError("problem exists", nil))
			},
			wantErr: &serviceerr.DuplicateProblemError{},
		},
		{
			name: "update error",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
				m.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10}, nil)
				m.EXPECT().ReplaceProblemByProblemID(int64(1), gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: &serviceerr.InternalError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockStore := mock_data.NewMockDataStore(gomock.NewController(t))
			mockStore.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			tt.setupMock(mockStore)
			err := NewProblemService(mockStore, authtest.Allow{}).ReplaceProblemByProblemID(authtest.Actor(), service_model.ReplaceProblemParams{
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
		mutate func(*service_model.ReplaceProblemParams)
		reason string
	}{
		{"zero problem ID", func(p *service_model.ReplaceProblemParams) { p.ProblemID = 0 }, "problem ID must be positive"},
		{"negative problem ID", func(p *service_model.ReplaceProblemParams) { p.ProblemID = -1 }, "problem ID must be positive"},
		{"missing destination class code", func(p *service_model.ReplaceProblemParams) { p.ClassCode = "" }, "class code is required"},
		{"blank destination class code", func(p *service_model.ReplaceProblemParams) { p.ClassCode = " " }, "class code is required"},
		{"blank name", func(p *service_model.ReplaceProblemParams) { p.Name = " " }, "problem name is required"},
		{"relative URL", func(p *service_model.ReplaceProblemParams) { p.URL = "/problem" }, "URL must be an absolute HTTP or HTTPS URL"},
		{"unsupported URL", func(p *service_model.ReplaceProblemParams) { p.URL = "ftp://example.com/problem" }, "URL must be an absolute HTTP or HTTPS URL"},
		{"invalid difficulty", func(p *service_model.ReplaceProblemParams) { p.Difficulty = helpers.PtrStr("extreme") }, "invalid difficulty"},
		{"invalid priority", func(p *service_model.ReplaceProblemParams) { p.Priority = helpers.PtrStr("urgent") }, "invalid priority"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := service_model.ReplaceProblemParams{ProblemID: 1, ClassCode: "basicos", Name: "Two Sum", URL: "https://example.com/problem"}
			tt.mutate(&params)
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			err := NewProblemService(store, authtest.Allow{}).ReplaceProblemByProblemID(authtest.Actor(), params)
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
			mockStore.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			tt.setupMock(mockStore)

			svc := NewProblemService(mockStore, authtest.Allow{})
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
			mockStore.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
			tt.setupMock(mockStore)

			svc := NewProblemService(mockStore, authtest.Allow{})
			p, err := svc.GetProblemByID(tt.problemID)
			tt.assertFunc(t, p, err)
		})
	}
}

func TestPrepareProblemBatchNormalizesAndKeepsRepeatedProblems(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos", Name: "Basics"}}, nil)
	difficulty := " HARD "
	input := service_model.CreateProblemParams{ClassCode: " basicos ", Name: " Two Sum ", URL: " https://example.com/problem ", Difficulty: &difficulty}
	prepared, issues, err := NewServices(store, authtest.Allow{}).Problem.PrepareCreationBatch(authtest.Actor(), []service_model.CreateProblemParams{input, input})
	require.NoError(t, err)
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 2) {
		assert.Equal(t, 1, prepared[0].Entry)
		assert.Equal(t, 2, prepared[1].Entry)
		assert.Equal(t, int64(10), prepared[0].Class.ID)
		assert.Equal(t, "basicos", prepared[0].Params.ClassCode)
		assert.Equal(t, "Two Sum", prepared[0].Params.Name)
		assert.Equal(t, "https://example.com/problem", prepared[0].Params.URL)
		assert.Equal(t, "hard", *prepared[0].Params.Difficulty)
		assert.Equal(t, prepared[0].Params, prepared[1].Params)
	}
	assert.Equal(t, " HARD ", difficulty)
	assert.Equal(t, " basicos ", input.ClassCode)
}

func TestPrepareProblemBatchReportsEveryInvalidEntry(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"missing"}).Return(nil, nil)
	input := []service_model.CreateProblemParams{
		{},
		{ClassCode: "missing", Name: "Valid", URL: "https://example.com/problem"},
		{Name: "Other", ClassCode: "basicos", URL: "javascript:alert(1)"},
	}
	prepared, issues, err := NewServices(store, authtest.Allow{}).Problem.PrepareCreationBatch(authtest.Actor(), input)
	require.NoError(t, err)
	assert.Nil(t, prepared)
	if assert.Len(t, issues, 3) {
		assert.Equal(t, 1, issues[0].Entry)
		assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
		assert.Contains(t, issues[0].Err.Error(), "problem name is required")
		assert.Contains(t, issues[0].Err.Error(), "class code is required")
		assert.Equal(t, 2, issues[1].Entry)
		assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, issues[1].Err)
		assert.Equal(t, 3, issues[2].Entry)
	}
}

func TestPrepareProblemCreationBatchReturnsDatastoreFailuresSeparately(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return(nil, errors.New("database unavailable"))

	prepared, issues, err := NewProblemService(store, authtest.Allow{}).PrepareCreationBatch(authtest.Actor(), []service_model.CreateProblemParams{
		{ClassCode: "basicos", Name: "Valid", URL: "https://example.com/problem"},
	})

	assert.Nil(t, prepared)
	assert.Empty(t, issues)
	assert.IsType(t, &serviceerr.InternalError{}, err)
}

func TestPrepareProblemUpdateBatchValidatesTargetsWithoutWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetProblemsByIDs([]int64{42}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	prepared, issues := NewServices(store, authtest.Allow{}).Problem.PrepareProblemPatchBatch(authtest.Actor(), []service_model.PatchProblemParams{{ProblemID: 42, ClassCode: service_model.PatchField[string]{Set: true, Value: " basicos "}, Name: service_model.PatchField[string]{Set: true, Value: " Updated "}, URL: service_model.PatchField[string]{Set: true, Value: " https://example.com/new "}}})
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 1) {
		assert.Equal(t, 1, prepared[0].Entry)
		assert.EqualValues(t, 42, prepared[0].Params.ProblemID)
		assert.Equal(t, "basicos", prepared[0].Params.ClassCode.Value)
		assert.Equal(t, "Updated", prepared[0].Params.Name.Value)
		assert.Equal(t, "https://example.com/new", prepared[0].Params.URL.Value)
		assert.EqualValues(t, 10, prepared[0].Class.ID)
	}
}

func TestPrepareProblemUpdateBatchReportsIssuesWithoutPartialResults(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetProblemsByIDs([]int64{42, 99}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	prepared, issues := NewServices(store, authtest.Allow{}).Problem.PrepareProblemPatchBatch(authtest.Actor(), []service_model.PatchProblemParams{
		{ProblemID: 42, ClassCode: service_model.PatchField[string]{Set: true, Value: "basicos"}, Name: service_model.PatchField[string]{Set: true, Value: "Valid"}},
		{},
		{ProblemID: 99, ClassCode: service_model.PatchField[string]{Set: true, Value: "basicos"}, Name: service_model.PatchField[string]{Set: true, Value: "Missing"}},
	})
	assert.Nil(t, prepared)
	if assert.Len(t, issues, 2) {
		assert.Equal(t, 2, issues[0].Entry)
		assert.Equal(t, 3, issues[1].Entry)
		assert.IsType(t, &serviceerr.ProblemNotFoundError{}, issues[1].Err)
	}
}

func TestProblemPatchBatchWithoutClassChangesDoesNotResolveClasses(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetProblemsByIDs([]int64{42}).Return([]data.Problem{{ID: 42, ClassID: 10}}, nil).Times(2)
	store.EXPECT().PatchProblemByProblemID(int64(42), data.ProblemPatch{
		Difficulty: data.PatchField[*string]{Set: true, Value: new("easy")},
	}).Return(nil)
	store.EXPECT().PatchProblemByProblemID(int64(42), data.ProblemPatch{
		Topic: data.PatchField[*string]{Set: true},
	}).Return(nil)
	service := NewProblemService(store, authtest.Allow{})
	prepared, issues := service.PrepareProblemPatchBatch(authtest.Actor(), []service_model.PatchProblemParams{
		{ProblemID: 42, Difficulty: service_model.PatchField[*string]{Set: true, Value: new("easy")}},
		{ProblemID: 42, Topic: service_model.PatchField[*string]{Set: true}},
	})
	require.Empty(t, issues)
	require.Len(t, prepared, 2)
	assert.Zero(t, prepared[0].Class)
	assert.Zero(t, prepared[1].Class)
	result, issues, err := service.ExecuteProblemPatchBatch(authtest.Actor(), prepared)
	require.NoError(t, err)
	assert.Empty(t, issues)
	assert.Equal(t, []int64{42, 42}, result.PatchedProblemIDs)
}

func TestProblemBatchUsesBulkLookupsForPreparationAndConfirmation(t *testing.T) {
	for _, size := range []int{1, service_model.MaxBatchOperations} {
		t.Run(fmt.Sprintf("%d entries", size), func(t *testing.T) {
			t.Parallel()

			t.Run("create", func(t *testing.T) {
				t.Parallel()
				store := mock_data.NewMockDataStore(gomock.NewController(t))
				store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
				classes := make([]data.Class, size)
				inputs := make([]service_model.CreateProblemParams, size)
				for index := range size {
					classCode := fmt.Sprintf("class-%03d", index)
					classes[index] = data.Class{ID: int64(index + 1), ClassCode: classCode}
					inputs[index] = service_model.CreateProblemParams{
						ClassCode: classCode,
						ProblemFields: service_model.ProblemFields{
							Name: fmt.Sprintf("Problem %d", index),
							URL:  "https://example.com/problem",
						},
					}
				}

				store.EXPECT().GetClassesByClassCodes(gomock.Any()).DoAndReturn(
					func(classCodes []string) ([]data.Class, error) {
						assert.Len(t, classCodes, size)
						return classes, nil
					},
				).Times(2)
				store.EXPECT().AddProblemsBatch(gomock.Any()).DoAndReturn(func(problems []data.Problem) ([]int64, error) {
					assert.Len(t, problems, size)
					problemIDs := make([]int64, size)
					for index := range problemIDs {
						problemIDs[index] = int64(index + 1)
					}
					return problemIDs, nil
				})

				service := NewServices(store, authtest.Allow{})
				prepared, issues, err := service.Problem.PrepareCreationBatch(authtest.Actor(), inputs)
				require.NoError(t, err)
				require.Empty(t, issues)
				require.Len(t, prepared, size)
				result, issues, err := service.Problem.ExecuteCreationBatch(authtest.Actor(), prepared)
				require.NoError(t, err)
				require.Empty(t, issues)
				assert.Len(t, result.CreatedProblemIDs, size)
			})

			t.Run("update", func(t *testing.T) {
				t.Parallel()
				store := mock_data.NewMockDataStore(gomock.NewController(t))
				store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
				problems := make([]data.Problem, size)
				classes := make([]data.Class, size)
				inputs := make([]service_model.PatchProblemParams, size)
				for index := range size {
					id := int64(index + 1)
					classCode := fmt.Sprintf("class-%03d", index)
					problems[index] = data.Problem{ID: id}
					classes[index] = data.Class{ID: id, ClassCode: classCode}
					inputs[index] = service_model.PatchProblemParams{
						ProblemID: id,
						PatchProblemFields: service_model.PatchProblemFields{
							ClassCode: service_model.PatchField[string]{Set: true, Value: classCode},
							Name:      service_model.PatchField[string]{Set: true, Value: fmt.Sprintf("Updated problem %d", index)},
							URL:       service_model.PatchField[string]{Set: true, Value: "https://example.com/problem"},
						},
					}
				}

				store.EXPECT().GetProblemsByIDs(gomock.Any()).DoAndReturn(
					func(problemIDs []int64) ([]data.Problem, error) {
						assert.Len(t, problemIDs, size)
						return problems, nil
					},
				).Times(2)
				store.EXPECT().GetClassesByClassCodes(gomock.Any()).DoAndReturn(
					func(classCodes []string) ([]data.Class, error) {
						assert.Len(t, classCodes, size)
						return classes, nil
					},
				).Times(2)
				store.EXPECT().PatchProblemByProblemID(gomock.Any(), gomock.Any()).Return(nil).Times(size)

				service := NewServices(store, authtest.Allow{})
				prepared, issues := service.Problem.PrepareProblemPatchBatch(authtest.Actor(), inputs)
				require.Empty(t, issues)
				require.Len(t, prepared, size)
				result, issues, err := service.Problem.ExecuteProblemPatchBatch(authtest.Actor(), prepared)
				require.NoError(t, err)
				require.Empty(t, issues)
				assert.Len(t, result.PatchedProblemIDs, size)
			})
		})
	}
}

func TestPrepareProblemUpdateBatchPreservesMissingTargetsWhenClassLookupFails(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetProblemsByIDs([]int64{42, 99}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return(nil, errors.New("database unavailable"))

	prepared, issues := NewServices(store, authtest.Allow{}).Problem.PrepareProblemPatchBatch(authtest.Actor(), []service_model.PatchProblemParams{
		{ProblemID: 42, ClassCode: service_model.PatchField[string]{Set: true, Value: "basicos"}},
		{ProblemID: 99, ClassCode: service_model.PatchField[string]{Set: true, Value: "basicos"}},
	})

	assert.Nil(t, prepared)
	if assert.Len(t, issues, 2) {
		assert.Equal(t, 1, issues[0].Entry)
		assert.IsType(t, &serviceerr.InternalError{}, issues[0].Err)
		assert.Equal(t, 2, issues[1].Entry)
		assert.IsType(t, &serviceerr.ProblemNotFoundError{}, issues[1].Err)
	}
}

func TestProblemUpdateBatchRejectsInvalidSizeBeforeReadingOrWriting(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		store := mock_data.NewMockDataStore(gomock.NewController(t))
		store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
		svc := NewServices(store, authtest.Allow{})
		prepared, issues := svc.Problem.PrepareProblemPatchBatch(authtest.Actor(), make([]service_model.PatchProblemParams, size))
		assert.Nil(t, prepared)
		if assert.Len(t, issues, 1) {
			assert.Zero(t, issues[0].Entry)
		}
		result, issues, err := svc.Problem.ExecuteProblemPatchBatch(authtest.Actor(), make([]service_model.PreparedProblemPatch, size))
		assert.NoError(t, err)
		assert.Empty(t, result.PatchedProblemIDs)
		assert.Len(t, issues, 1)
	}
}

func TestExecuteProblemBatchRejectsReplacedClassBeforeWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 11, ClassCode: "basicos"}}, nil)
	params := service_model.CreateProblemParams{ClassCode: "basicos", Name: "New", URL: "https://example.com"}
	result, issues, err := NewServices(store, authtest.Allow{}).Problem.ExecuteCreationBatch(authtest.Actor(), []service_model.PreparedProblem{
		{Entry: 1, Params: params, Class: service_model.Class{ID: 10, ClassCode: "basicos"}},
	})
	require.NoError(t, err)
	assert.Empty(t, result.CreatedProblemIDs)
	require.Len(t, issues, 1)
	assert.Equal(t, 1, issues[0].Entry)
	assert.IsType(t, &serviceerr.ConflictError{}, issues[0].Err)
}

func TestExecuteProblemCreationBatchWrapsDatastoreFailure(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	store.EXPECT().AddProblemsBatch([]data.Problem{{Name: "New", URL: "https://example.com", ClassID: 10}}).Return(nil, errors.New("database unavailable"))

	result, issues, err := NewProblemService(store, authtest.Allow{}).ExecuteCreationBatch(authtest.Actor(), []service_model.PreparedProblem{{
		Entry:  1,
		Params: service_model.CreateProblemParams{ClassCode: "basicos", Name: "New", URL: "https://example.com"},
		Class:  service_model.Class{ID: 10, ClassCode: "basicos"},
	}})

	assert.Empty(t, result.CreatedProblemIDs)
	assert.Empty(t, issues)
	assert.IsType(t, &serviceerr.InternalError{}, err)
}

func TestProblemCreationBatchRejectsInvalidSizeBeforeReadingOrWriting(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			t.Parallel()
			service := NewProblemService(mock_data.NewMockDataStore(gomock.NewController(t)), authtest.Allow{})
			prepared, preparationIssues, preparationErr := service.PrepareCreationBatch(authtest.Actor(), make([]service_model.CreateProblemParams, size))
			assert.NoError(t, preparationErr)
			assert.Nil(t, prepared)
			result, executionIssues, err := service.ExecuteCreationBatch(authtest.Actor(), make([]service_model.PreparedProblem, size))
			require.NoError(t, err)
			assert.Empty(t, result.CreatedProblemIDs)
			for _, issues := range [][]service_model.BatchEntryError{preparationIssues, executionIssues} {
				require.Len(t, issues, 1)
				assert.Zero(t, issues[0].Entry)
				assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
			}
		})
	}
}

func TestPrepareProblemBatchDoesNotReturnPartialSuccess(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassByClassID(gomock.Any()).Return(data.Class{ID: 1, ClassCode: "source"}, nil).AnyTimes()
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	prepared, issues, err := NewProblemService(store, authtest.Allow{}).PrepareCreationBatch(authtest.Actor(), []service_model.CreateProblemParams{
		{ClassCode: "basicos", Name: "Valid", URL: "https://example.com/problem"}, {},
	})
	require.NoError(t, err)
	assert.Nil(t, prepared)
	require.Len(t, issues, 1)
	assert.Equal(t, 2, issues[0].Entry)
}
