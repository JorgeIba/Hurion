package services_test

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/helpers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/mock_services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScoreService_GetRawScoreByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore, *mock_services.MockProblemService)
		wantResult map[service_model.User][]service_model.SolvedProblem
		wantErr    bool
	}{
		{
			name:      "success - raw scores",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				u1 := data.User{UserID: "u1", DisplayName: "user1"}
				u2 := data.User{UserID: "u2", DisplayName: "user2"}
				u3 := data.User{UserID: "u3", DisplayName: "user3"}
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					u1: {{UserID: "u1", ProblemID: 100}},
					u3: {{UserID: "u3", ProblemID: 101}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{u1, u2, u3}, nil)
			},
			wantResult: map[service_model.User][]service_model.SolvedProblem{
				{UserID: "u1", DisplayName: "user1"}: {{UserID: "u1", ProblemID: 100}},
				{UserID: "u2", DisplayName: "user2"}: {},
				{UserID: "u3", DisplayName: "user3"}: {{UserID: "u3", ProblemID: 101}},
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantResult: map[service_model.User][]service_model.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantResult: map[service_model.User][]service_model.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetAllSolvedProblemsByClass",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: map[service_model.User][]service_model.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetAllUsersByClassID",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: map[service_model.User][]service_model.SolvedProblem{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockProbSvc := mock_services.NewMockProblemService(ctrl)
			tt.setupMock(mockStore, mockProbSvc)

			svc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProbSvc}
			res, err := svc.GetRawScoreByClassCode(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)
			}
		})
	}
}

func TestScoreService_GetScoreByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mock_data.MockDataStore, *mock_services.MockProblemService)
		assertFunc func(*testing.T, service_model.Score)
		wantErr    bool
	}{
		{
			name:      "success - 3 users with different solved counts",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				u1 := data.User{UserID: "u1", DisplayName: "user1"}
				u2 := data.User{UserID: "u2", DisplayName: "user2"}
				u3 := data.User{UserID: "u3", DisplayName: "user3"}
				
				problems := []service_model.Problem{
					{ID: 100, Topic: helpers.PtrStr("T1"), Name: "P1"},
					{ID: 101, Topic: helpers.PtrStr("T1"), Name: "P2"},
					{ID: 102, Topic: helpers.PtrStr("T1"), Name: "P3"},
				}

				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				
				// Mock solve status
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					u1: {{ProblemID: 100}, {ProblemID: 101}, {ProblemID: 102}}, // Solved 3
					u2: {{ProblemID: 100}, {ProblemID: 101}},                  // Solved 2
					u3: {{ProblemID: 100}},                                     // Solved 1
				}, nil)
				
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{u1, u2, u3}, nil)
				
				// Mock ProblemService instead of DataStore for problems
				groupedProblems := []service_model.ProblemsByTopic{
					{
						Topic:    "T1",
						Problems: problems,
					},
				}
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return(groupedProblems, nil)
			},
			assertFunc: func(t *testing.T, res service_model.Score) {
				assert.Len(t, res.UserScores, 3)
				
				// Sorting check (Desc by count)
				// User 1: 3 AC
				assert.Equal(t, "u1", res.UserScores[0].User.UserID)
				assert.Equal(t, "user1", res.UserScores[0].User.DisplayName)
				assert.Equal(t, 3, res.UserScores[0].CountProblems)
				
				// User 2: 2 AC
				assert.Equal(t, "u2", res.UserScores[1].User.UserID)
				assert.Equal(t, "user2", res.UserScores[1].User.DisplayName)
				assert.Equal(t, 2, res.UserScores[1].CountProblems)
				
				// User 3: 1 AC
				assert.Equal(t, "u3", res.UserScores[2].User.UserID)
				assert.Equal(t, "user3", res.UserScores[2].User.DisplayName)
				assert.Equal(t, 1, res.UserScores[2].CountProblems)

				// Check problem map for User 2 (Solved P1, P2; Not P3)
				// Note: ProblemsByTopic contains the ordered problems
				p1 := res.ProblemsByTopic[0].Problems[0]
				p2 := res.ProblemsByTopic[0].Problems[1]
				p3 := res.ProblemsByTopic[0].Problems[2]
				
				assert.True(t, res.UserScores[1].Problems[p1.ID])
				assert.True(t, res.UserScores[1].Problems[p2.ID])
				assert.False(t, res.UserScores[1].Problems[p3.ID])
			},
			wantErr: false,
		},
		{
			name:      "error in GetRawScoreByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error in GetProblemsGroupedByTopic",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{}, nil)
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return(nil, errors.New("db fail"))
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
			mockProbSvc := mock_services.NewMockProblemService(ctrl)
			tt.setupMock(mockStore, mockProbSvc)

			// ScoreService needs ProblemService
			svc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProbSvc}

			res, err := svc.GetScoreByClassCode(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.assertFunc != nil {
					tt.assertFunc(t, res)
				}
			}
		})
	}
}

func TestScoreService_GetNeighborsScoreForUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		classCode  string
		setupMock  func(*mock_data.MockDataStore, *mock_services.MockProblemService)
		assertFunc func(*testing.T, service_model.Score)
		wantErr    bool
	}{
		{
			name:      "success - middle of list (7 users, target at index 3)",
			userID:    "id4",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()

				users := []data.User{
					{UserID: "id1", DisplayName: "u1"},
					{UserID: "id2", DisplayName: "u2"},
					{UserID: "id3", DisplayName: "u3"},
					{UserID: "id4", DisplayName: "u4"},
					{UserID: "id5", DisplayName: "u5"},
					{UserID: "id6", DisplayName: "u6"},
					{UserID: "id7", DisplayName: "u7"},
				}

				// Crucial: define problems so CountProblems works and sorting is stable
				problems := []service_model.Problem{
					{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}, {ID: 6}, {ID: 7},
				}
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return([]service_model.ProblemsByTopic{
					{Topic: "T", Problems: problems},
				}, nil)

				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					users[0]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}, {ProblemID: 4}, {ProblemID: 5}, {ProblemID: 6}, {ProblemID: 7}},
					users[1]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}, {ProblemID: 4}, {ProblemID: 5}, {ProblemID: 6}},
					users[2]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}, {ProblemID: 4}, {ProblemID: 5}},
					users[3]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}, {ProblemID: 4}}, // target u4 at index 3
					users[4]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}},
					users[5]: {{ProblemID: 1}, {ProblemID: 2}},
					users[6]: {{ProblemID: 1}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(users, nil)
			},
			assertFunc: func(t *testing.T, res service_model.Score) {
				// Window should be: [id2, id3, id4, id5, id6]
				assert.Len(t, res.UserScores, 5)
				assert.Equal(t, "id2", res.UserScores[0].User.UserID)
				assert.Equal(t, "id3", res.UserScores[1].User.UserID)
				assert.Equal(t, "id4", res.UserScores[2].User.UserID)
				assert.Equal(t, "id5", res.UserScores[3].User.UserID)
				assert.Equal(t, "id6", res.UserScores[4].User.UserID)
			},
			wantErr: false,
		},
		{
			name:      "success - target at start (index 0)",
			userID:    "id1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()

				users := []data.User{
					{UserID: "id1", DisplayName: "u1"},
					{UserID: "id2", DisplayName: "u2"},
					{UserID: "id3", DisplayName: "u3"},
					{UserID: "id4", DisplayName: "u4"},
				}

				problems := []service_model.Problem{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return([]service_model.ProblemsByTopic{
					{Topic: "T", Problems: problems},
				}, nil)

				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					users[0]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}, {ProblemID: 4}},
					users[1]: {{ProblemID: 1}, {ProblemID: 2}, {ProblemID: 3}},
					users[2]: {{ProblemID: 1}, {ProblemID: 2}},
					users[3]: {{ProblemID: 1}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(users, nil)
			},
			assertFunc: func(t *testing.T, res service_model.Score) {
				// Index 0 -> max(0, 0-2) to min(3, 0+2) => 0 to 2
				assert.Len(t, res.UserScores, 3)
				assert.Equal(t, "id1", res.UserScores[0].User.UserID)
				assert.Equal(t, "u1", res.UserScores[0].User.DisplayName)
				assert.Equal(t, "id2", res.UserScores[1].User.UserID)
				assert.Equal(t, "u2", res.UserScores[1].User.DisplayName)
				assert.Equal(t, "id3", res.UserScores[2].User.UserID)
				assert.Equal(t, "u3", res.UserScores[2].User.DisplayName)
			},
			wantErr: false,
		},
		{
			name:      "error in GetScoreByClassCode",
			userID:    "u1",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "user not found in scores",
			userID:    "ghostID",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(nil, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{}, nil)
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return(nil, nil)
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
			mockProbSvc := mock_services.NewMockProblemService(ctrl)
			tt.setupMock(mockStore, mockProbSvc)

			svc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProbSvc}

			res, err := svc.GetNeighborsScoreForUserID(tt.userID, tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.assertFunc != nil {
					tt.assertFunc(t, res)
				}
			}
		})
	}
}

func TestScoreService_GetScoreAllClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*mock_data.MockDataStore, *mock_services.MockProblemService)
		assertFunc func(*testing.T, map[string]service_model.Score)
		wantErr    bool
	}{
		{
			name: "success - scores for multiple classes",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetAllClasses().Return([]data.Class{
					{ID: 1, ClassCode: "C1", Name: "Class 1"},
					{ID: 2, ClassCode: "C2", Name: "Class 2"},
				}, nil)

				// Mock Class 1
				m.EXPECT().GetClassByClassCode("C1").Return(data.Class{ID: 1, ClassCode: "C1"}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "u1"}: {{ProblemID: 100}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(1)).Return([]data.User{{UserID: "u1"}}, nil)
				ps.EXPECT().GetProblemsGroupedByTopic("C1").Return([]service_model.ProblemsByTopic{
					{Problems: []service_model.Problem{{ID: 100}}},
				}, nil)

				// Mock Class 2
				m.EXPECT().GetClassByClassCode("C2").Return(data.Class{ID: 2, ClassCode: "C2"}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(2)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "u2"}: {{ProblemID: 200}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(2)).Return([]data.User{{UserID: "u2"}}, nil)
				ps.EXPECT().GetProblemsGroupedByTopic("C2").Return([]service_model.ProblemsByTopic{
					{Problems: []service_model.Problem{{ID: 200}}},
				}, nil)
			},
			assertFunc: func(t *testing.T, res map[string]service_model.Score) {
				assert.Len(t, res, 2)
				assert.Contains(t, res, "Class 1")
				assert.Contains(t, res, "Class 2")
				
				assert.Equal(t, "u1", res["Class 1"].UserScores[0].User.UserID)
				assert.Equal(t, "u2", res["Class 2"].UserScores[0].User.UserID)
			},
			wantErr: false,
		},
		{
			name: "error GetAllClasses",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetAllClasses().Return(nil, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name: "error in GetScoreByClassCode for a class",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetAllClasses().Return([]data.Class{{ID: 1, ClassCode: "C1"}}, nil)
				m.EXPECT().GetClassByClassCode("C1").Return(data.Class{}, errors.New("db fail"))
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
			mockProbSvc := mock_services.NewMockProblemService(ctrl)
			tt.setupMock(mockStore, mockProbSvc)

			svc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProbSvc}

			res, err := svc.GetScoreAllClasses()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.assertFunc != nil {
					tt.assertFunc(t, res)
				}
			}
		})
	}
}

func TestScoreService_GetClassExportData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		classCode string
		setupMock func(*mock_data.MockDataStore, *mock_services.MockProblemService)
		wantErr   bool
	}{
		{
			name:      "success",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10, ClassCode: "CS101"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{{UserID: "u1"}}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				ps.EXPECT().GetProblemsGroupedByTopic("CS101").Return([]service_model.ProblemsByTopic{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "generic error GetAllUsersByClassID",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error in GetScoreByClassCode",
			classCode: "CS101",
			setupMock: func(m *mock_data.MockDataStore, ps *mock_services.MockProblemService) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10, ClassCode: "CS101"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{{UserID: "u1"}}, nil)
				// GetScoreByClassCode will call GetRawScoreByClassCode -> GetAllSolvedProblemsByClass
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
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
			mockProbSvc := mock_services.NewMockProblemService(ctrl)
			tt.setupMock(mockStore, mockProbSvc)

			svc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProbSvc}

			users, problems, score, err := svc.GetClassExportData(tt.classCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, users)
				assert.NotNil(t, problems)
				assert.NotNil(t, score)
			}
		})
	}
}
