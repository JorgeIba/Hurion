package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSolutionService_AddACToUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		problemID  int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - AC added",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "already solved",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ACAlreadyExistsError{}, err)
			},
		},
		{
			name:      "AddAC generic error",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(errors.New("db fail"))
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

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AddACToUserID(tt.userID, tt.problemID)
			tt.assertFunc(t, err)
		})
	}
}

func TestSolutionService_AddACByPrincipal(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		roles   []string
		allowed bool
	}{
		{name: "records AC after validating access", roles: []string{"student"}, allowed: true},
		{name: "denies access without writing", roles: []string{"other"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
				{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}},
			}, nil)
			if test.allowed {
				gomock.InOrder(
					store.EXPECT().EnsureUser(data.User{UserID: "123", DisplayName: "Unknown"}).Return(nil),
					store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(false, nil),
					store.EXPECT().AddAC("123", int64(42)).Return(nil),
				)
			}
			err := (&SolutionService{Datastore: store}).AddACByPrincipal(service_model.Principal{UserID: "123", Roles: test.roles}, 42)
			if test.allowed {
				assert.NoError(t, err)
			} else {
				assert.IsType(t, &serviceerr.UserNotEnrolledError{}, err)
			}
		})
	}
}

func TestPrepareACCreationBatchNormalizesWithoutWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
		{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}},
	}, nil)
	student := service_model.Principal{UserID: "123", DisplayName: "Ana", Roles: []string{"student"}}
	prepared, issues, err := NewServices(store).Solution.PrepareCreationBatch(
		[]service_model.CreateACParams{{UserID: " 00123 ", ProblemID: 42}},
		map[string]service_model.Principal{"123": student},
	)
	require.Empty(t, issues)
	require.Len(t, prepared, 1)
	if assert.NoError(t, err) {
		assert.Equal(t, student.User, prepared[0].User)
		assert.Equal(t, int64(42), prepared[0].Problem.ID)
		assert.Equal(t, "basicos", prepared[0].Class.ClassCode)
		assert.Equal(t, 1, prepared[0].Entry)
	}
}

func TestPrepareACCreationBatchRejectsInvalidTargetsWithoutQueries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params service_model.CreateACParams
	}{
		{name: "missing user", params: service_model.CreateACParams{ProblemID: 42}},
		{name: "non-numeric user", params: service_model.CreateACParams{UserID: "abc", ProblemID: 42}},
		{name: "zero user", params: service_model.CreateACParams{UserID: "0", ProblemID: 42}},
		{name: "signed user", params: service_model.CreateACParams{UserID: "+123", ProblemID: 42}},
		{name: "missing problem", params: service_model.CreateACParams{UserID: "123"}},
		{name: "negative problem", params: service_model.CreateACParams{UserID: "123", ProblemID: -1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			prepared, issues, err := NewServices(store).Solution.PrepareCreationBatch([]service_model.CreateACParams{test.params}, nil)
			require.NoError(t, err)
			require.Len(t, issues, 1)
			assert.Equal(t, 1, issues[0].Entry)
			assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
			assert.Nil(t, prepared)
		})
	}
}

func TestSolutionService_UndoAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		problemID  int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - AC removed",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
				m.EXPECT().RemoveAC("u1", int64(100)).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "not solved",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ConflictError{}, err)
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

			svc := &SolutionService{Datastore: mockStore}
			err := svc.UndoAC(tt.userID, tt.problemID)
			tt.assertFunc(t, err)
		})
	}
}

func TestSolutionService_AddPendingACByPrincipal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageID  string
		principal  service_model.Principal
		problemID  int64
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success - pending AC added",
			messageID: "m1",
			principal: service_model.Principal{
				User:  service_model.User{UserID: "u1"},
				Roles: []string{"role_student"},
			},
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 100}}).Return(map[data.ACKey]data.ACValidationData{
					{UserID: "u1", ProblemID: 100}: {Problem: &data.Problem{ID: 100, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "CS101"}, AccessRoleIDs: []string{"role_student"}},
				}, nil)
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddPendingConfirmation(gomock.Any()).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "already solved",
			messageID: "m1",
			principal: service_model.Principal{
				User:  service_model.User{UserID: "u1"},
				Roles: []string{"role_student"},
			},
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 100}}).Return(map[data.ACKey]data.ACValidationData{
					{UserID: "u1", ProblemID: 100}: {Problem: &data.Problem{ID: 100, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "CS101"}, AccessRoleIDs: []string{"role_student"}},
				}, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ACAlreadyExistsError{}, err)
			},
		},
		{
			name:      "user not enrolled",
			messageID: "m1",
			principal: service_model.Principal{
				User:  service_model.User{UserID: "u1"},
				Roles: []string{"wrong"},
			},
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 100}}).Return(map[data.ACKey]data.ACValidationData{
					{UserID: "u1", ProblemID: 100}: {Problem: &data.Problem{ID: 100, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "CS101"}, AccessRoleIDs: []string{"role_student"}},
				}, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.UserNotEnrolledError{}, err)
			},
		},
		{
			name:      "problem does not exist",
			messageID: "m1",
			principal: service_model.Principal{User: service_model.User{UserID: "u1"}},
			problemID: 404,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 404}}).Return(map[data.ACKey]data.ACValidationData{}, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &serviceerr.ProblemNotFoundError{}, err)
			},
		},
		{
			name:      "internal error - get problem fails",
			messageID: "m1",
			principal: service_model.Principal{User: service_model.User{UserID: "u1"}},
			problemID: 100,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 100}}).Return(nil, errors.New("db fail"))
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

			svc := NewServices(mockStore).Solution
			err := svc.AddPendingACByPrincipal(tt.messageID, tt.principal, tt.problemID)
			tt.assertFunc(t, err)
		})
	}
}

func TestSolutionService_ExistsPendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, bool, error)
	}{
		{
			name:      "exists",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().ExistsPendingConfirmation("m1").Return(true, nil)
			},
			assertFunc: func(t *testing.T, got bool, err error) {
				assert.NoError(t, err)
				assert.True(t, got)
			},
		},
		{
			name:      "error",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().ExistsPendingConfirmation("m1").Return(false, errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, got bool, err error) {
				assert.Error(t, err)
				assert.False(t, got)
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

			svc := &SolutionService{Datastore: mockStore}
			got, err := svc.ExistsPendingConfirmation(tt.messageID)
			tt.assertFunc(t, got, err)
		})
	}
}

func TestSolutionService_GetPendingConfirmationByMessageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, *service_model.PendingConfirmation, error)
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
			},
			assertFunc: func(t *testing.T, res *service_model.PendingConfirmation, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "m1", res.MessageID)
			},
		},
		{
			name:      "not found",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("", int64(0), &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, res *service_model.PendingConfirmation, err error) {
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

			svc := &SolutionService{Datastore: mockStore}
			res, err := svc.GetPendingConfirmationByMessageID(tt.messageID)
			tt.assertFunc(t, res, err)
		})
	}
}

func TestSolutionService_AcceptACProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
				m.EXPECT().RemovePendingConfirmation("m1").Return(nil)
				m.EXPECT().EnsureUser(gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "error GetPendingConfirmationByMessageID",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("", int64(0), errors.New("db fail"))
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:      "error RemovePendingConfirmation",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
				m.EXPECT().RemovePendingConfirmation("m1").Return(errors.New("db fail"))
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

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AcceptACProposal(tt.messageID)
			tt.assertFunc(t, err)
		})
	}
}

func TestSolutionService_RejectACProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageID  string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, error)
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemovePendingConfirmation("m1").Return(nil)
			},
			assertFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:      "error",
			messageID: "m1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().RemovePendingConfirmation("m1").Return(errors.New("db fail"))
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

			svc := &SolutionService{Datastore: mockStore}
			err := svc.RejectACProposal(tt.messageID)
			tt.assertFunc(t, err)
		})
	}
}

func TestIndividualAndBatchACValidationShareAccessRules(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		roles, accessRoles []string
		allowed            bool
	}{
		{name: "matching class access", roles: []string{"student"}, accessRoles: []string{"student"}, allowed: true},
		{name: "no matching student role", roles: []string{"other"}, accessRoles: []string{"student"}},
		{name: "class has no access roles", roles: []string{"student"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			student := service_model.Principal{UserID: "123", DisplayName: "Ana", Roles: test.roles}
			problem := data.Problem{ID: 42, ClassID: 10, Name: "Two Sum"}
			class := data.Class{ID: 10, ClassCode: "basicos"}
			store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
				{UserID: "123", ProblemID: 42}: {Problem: &problem, Class: &class, AccessRoleIDs: test.accessRoles},
			}, nil).Times(3)
			if test.allowed {
				store.EXPECT().EnsureUser(data.User{UserID: "123", DisplayName: "Unknown"}).Return(nil)
				store.EXPECT().EnsureUser(data.User{UserID: "123", DisplayName: "Ana"}).Return(nil)
				store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(false, nil).Times(2)
				store.EXPECT().AddAC("123", int64(42)).Return(nil)
				store.EXPECT().AddPendingConfirmation(data.PendingConfirmation{MessageID: "message", UserID: "123", ProblemID: 42}).Return(nil)
			}
			svcs := NewServices(store)
			individualErr := svcs.Solution.AddACByPrincipal(student, 42)
			pendingErr := svcs.Solution.AddPendingACByPrincipal("message", student, 42)
			batch, issues, err := svcs.Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: " 00123 ", ProblemID: 42}}, map[string]service_model.Principal{"123": student})
			require.NoError(t, err)
			if !test.allowed {
				assert.IsType(t, &serviceerr.UserNotEnrolledError{}, individualErr)
				assert.IsType(t, &serviceerr.UserNotEnrolledError{}, pendingErr)
				assert.Nil(t, batch)
				require.Len(t, issues, 1)
				assert.Equal(t, 1, issues[0].Entry)
				assert.IsType(t, &serviceerr.UserNotEnrolledError{}, issues[0].Err)
				assert.ErrorContains(t, issues[0].Err, "user 123 has no access to class basicos")
				return
			}
			require.NoError(t, individualErr)
			require.NoError(t, pendingErr)
			require.Empty(t, issues)
			require.Len(t, batch, 1)
			assert.Equal(t, student.User, batch[0].User)
			assert.Equal(t, int64(42), batch[0].Problem.ID)
			assert.Equal(t, int64(10), batch[0].Class.ID)
			assert.Equal(t, 1, batch[0].Entry)
		})
	}
}

func TestIndividualACRejectsInvalidTargetsBeforeReads(t *testing.T) {
	t.Parallel()
	for _, input := range []service_model.CreateACParams{
		{ProblemID: 42}, {UserID: "u1"}, {UserID: "u1", ProblemID: -1},
	} {
		t.Run(input.UserID+"/"+fmt.Sprint(input.ProblemID), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			service := NewServices(store).Solution
			student := service_model.Principal{UserID: input.UserID}
			assert.IsType(t, &serviceerr.ValidationError{}, service.AddACByPrincipal(student, input.ProblemID))
			assert.IsType(t, &serviceerr.ValidationError{}, service.AddPendingACByPrincipal("message", student, input.ProblemID))
		})
	}
}

func TestIndividualACRejectsMissingClassBeforeWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "u1", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
		{UserID: "u1", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}},
	}, nil).Times(2)
	service := NewServices(store).Solution
	student := service_model.Principal{UserID: "u1", Roles: []string{"student"}}
	assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, service.AddACByPrincipal(student, 42))
	assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, service.AddPendingACByPrincipal("message", student, 42))
}
