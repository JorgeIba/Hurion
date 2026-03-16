package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScoreService_GetRawScoreByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		classCode  string
		setupMock  func(*mocks.MockDataStore)
		wantResult map[data.User][]data.SolvedProblem
		wantErr    bool
	}{
		{
			name:      "success - raw scores",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				u1 := data.User{UserID: "u1", UserName: "user1"}
				u2 := data.User{UserID: "u2", UserName: "user2"}
				u3 := data.User{UserID: "u3", UserName: "user3"}
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					u1: {{UserID: "u1", ProblemID: 100}},
					u3: {{UserID: "u3", ProblemID: 101}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{u1, u2, u3}, nil)
			},
			wantResult: map[data.User][]data.SolvedProblem{
				{UserID: "u1", UserName: "user1"}: {{UserID: "u1", ProblemID: 100}},
				{UserID: "u2", UserName: "user2"}: {},
				{UserID: "u3", UserName: "user3"}: {{UserID: "u3", ProblemID: 101}},
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantResult: map[data.User][]data.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantResult: map[data.User][]data.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetAllSolvedProblemsByClass",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: map[data.User][]data.SolvedProblem{},
			wantErr:    true,
		},
		{
			name:      "generic error GetAllUsersByClassID",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantResult: map[data.User][]data.SolvedProblem{},
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

			svc := &ScoreService{Datastore: mockStore}
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
		setupMock  func(*mocks.MockDataStore)
		assertFunc func(*testing.T, Score)
		wantErr    bool
	}{
		{
			name:      "success - 3 users with different solved counts",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				u1 := data.User{UserID: "u1", UserName: "user1"}
				u2 := data.User{UserID: "u2", UserName: "user2"}
				u3 := data.User{UserID: "u3", UserName: "user3"}
				
				problems := []data.Problem{
					{ID: 100, Topic: ptrStr("T1"), Name: "P1"},
					{ID: 101, Topic: ptrStr("T1"), Name: "P2"},
					{ID: 102, Topic: ptrStr("T1"), Name: "P3"},
				}

				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				
				// Mock solve status
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					u1: {{ProblemID: 100}, {ProblemID: 101}, {ProblemID: 102}}, // Solved 3
					u2: {{ProblemID: 100}, {ProblemID: 101}},                  // Solved 2
					u3: {{ProblemID: 100}},                                     // Solved 1
				}, nil)
				
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{u1, u2, u3}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(problems, nil)
			},
			assertFunc: func(t *testing.T, res Score) {
				assert.Len(t, res.UserScores, 3)
				
				// Sorting check (Desc by count)
				// User 1: 3 AC
				assert.Equal(t, "u1", res.UserScores[0].User.UserID)
				assert.Equal(t, 3, res.UserScores[0].CountProblems)
				
				// User 2: 2 AC
				assert.Equal(t, "u2", res.UserScores[1].User.UserID)
				assert.Equal(t, 2, res.UserScores[1].CountProblems)
				
				// User 3: 1 AC
				assert.Equal(t, "u3", res.UserScores[2].User.UserID)
				assert.Equal(t, 1, res.UserScores[2].CountProblems)

				// Check problem map for User 2 (Solved P1, P2; Not P3)
				// Note: ProblemsByTopic contains the ordered problems
				p1 := res.ProblemsByTopic[0].Problems[0]
				p2 := res.ProblemsByTopic[0].Problems[1]
				p3 := res.ProblemsByTopic[0].Problems[2]
				
				assert.True(t, res.UserScores[1].Problems[p1])
				assert.True(t, res.UserScores[1].Problems[p2])
				assert.False(t, res.UserScores[1].Problems[p3])
			},
			wantErr: false,
		},
		{
			name:      "error in GetRawScoreByClassCode",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error in GetProblemsGroupedByTopic",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, errors.New("db fail"))
			},
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

			// ScoreService needs ProblemService
			probSvc := &ProblemService{Datastore: mockStore}
			svc := &ScoreService{Datastore: mockStore, ProblemService: probSvc}

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

func TestScoreService_GetNeighborsScoreForUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		username   string
		classCode  string
		setupMock  func(*mocks.MockDataStore)
		assertFunc func(*testing.T, Score)
		wantErr    bool
	}{
		{
			name:      "success - middle of list (7 users, target at index 3)",
			username:  "u4",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("u4").Return(data.User{UserID: "id4", UserName: "u4"}, nil)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()

				users := []data.User{
					{UserID: "id1", UserName: "u1"},
					{UserID: "id2", UserName: "u2"},
					{UserID: "id3", UserName: "u3"},
					{UserID: "id4", UserName: "u4"},
					{UserID: "id5", UserName: "u5"},
					{UserID: "id6", UserName: "u6"},
					{UserID: "id7", UserName: "u7"},
				}

				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					users[0]: make([]data.SolvedProblem, 7),
					users[1]: make([]data.SolvedProblem, 6),
					users[2]: make([]data.SolvedProblem, 5),
					users[3]: make([]data.SolvedProblem, 4), // target u4
					users[4]: make([]data.SolvedProblem, 3),
					users[5]: make([]data.SolvedProblem, 2),
					users[6]: make([]data.SolvedProblem, 1),
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(users, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, nil)
			},
			assertFunc: func(t *testing.T, res Score) {
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
			username:  "u1",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("u1").Return(data.User{UserID: "id1", UserName: "u1"}, nil)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()

				users := []data.User{
					{UserID: "id1", UserName: "u1"},
					{UserID: "id2", UserName: "u2"},
					{UserID: "id3", UserName: "u3"},
					{UserID: "id4", UserName: "u4"},
				}

				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{
					users[0]: make([]data.SolvedProblem, 4),
					users[1]: make([]data.SolvedProblem, 3),
					users[2]: make([]data.SolvedProblem, 2),
					users[3]: make([]data.SolvedProblem, 1),
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(users, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, nil)
			},
			assertFunc: func(t *testing.T, res Score) {
				// Index 0 -> max(0, 0-2) to min(3, 0+2) => 0 to 2
				assert.Len(t, res.UserScores, 3)
				assert.Equal(t, "id1", res.UserScores[0].User.UserID)
				assert.Equal(t, "id2", res.UserScores[1].User.UserID)
				assert.Equal(t, "id3", res.UserScores[2].User.UserID)
			},
			wantErr: false,
		},
		{
			name:      "user not found in DB",
			username:  "ghost",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("ghost").Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error GetUserByUserName",
			username:  "u1",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("u1").Return(data.User{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error in GetScoreByClassCode",
			username:  "u1",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("u1").Return(data.User{UserID: "u1"}, nil)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "user not found in scores",
			username:  "ghost",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("ghost").Return(data.User{UserID: "ghostID"}, nil)
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(nil, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return(nil, nil)
			},
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

			probSvc := &ProblemService{Datastore: mockStore}
			svc := &ScoreService{Datastore: mockStore, ProblemService: probSvc}

			res, err := svc.GetNeighborsScoreForUsername(tt.username, tt.classCode)

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
		setupMock  func(*mocks.MockDataStore)
		assertFunc func(*testing.T, map[string]Score)
		wantErr    bool
	}{
		{
			name: "success - scores for multiple classes",
			setupMock: func(m *mocks.MockDataStore) {
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
				m.EXPECT().GetProblemsByClass(int64(1)).Return([]data.Problem{{ID: 100}}, nil)

				// Mock Class 2
				m.EXPECT().GetClassByClassCode("C2").Return(data.Class{ID: 2, ClassCode: "C2"}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(2)).Return(map[data.User][]data.SolvedProblem{
					{UserID: "u2"}: {{ProblemID: 200}},
				}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(2)).Return([]data.User{{UserID: "u2"}}, nil)
				m.EXPECT().GetProblemsByClass(int64(2)).Return([]data.Problem{{ID: 200}}, nil)
			},
			assertFunc: func(t *testing.T, res map[string]Score) {
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
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetAllClasses().Return(nil, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name: "error in GetScoreByClassCode for a class",
			setupMock: func(m *mocks.MockDataStore) {
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
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			probSvc := &ProblemService{Datastore: mockStore}
			svc := &ScoreService{Datastore: mockStore, ProblemService: probSvc}

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
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10, ClassCode: "CS101"}, nil).AnyTimes()
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return([]data.User{{UserID: "u1"}}, nil).AnyTimes()
				m.EXPECT().GetAllSolvedProblemsByClass(int64(10)).Return(map[data.User][]data.SolvedProblem{}, nil)
				m.EXPECT().GetProblemsByClass(int64(10)).Return([]data.Problem{}, nil)
			},
			wantErr: false,
		},
		{
			name:      "class not found",
			classCode: "NONE",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("NONE").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error GetClassByClassCode",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{}, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "generic error GetAllUsersByClassID",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetClassByClassCode("CS101").Return(data.Class{ID: 10}, nil)
				m.EXPECT().GetAllUsersByClassID(int64(10)).Return(nil, errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error in GetScoreByClassCode",
			classCode: "CS101",
			setupMock: func(m *mocks.MockDataStore) {
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
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			probSvc := &ProblemService{Datastore: mockStore}
			svc := &ScoreService{Datastore: mockStore, ProblemService: probSvc}

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
