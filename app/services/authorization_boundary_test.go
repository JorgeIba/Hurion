package services_test

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/authorization/mock_authorization"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/script"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type authFixture struct {
	db                      *data.DB
	svcs                    *services.Services
	raw                     *script.FakeDiscordProvider
	basic, advanced, p1, p2 int64
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
	require.NoError(t, err)
	t.Cleanup(db.CloseDB)
	basic, err := db.CreateClass(data.Class{ClassCode: "basics", Name: "Basics"})
	require.NoError(t, err)
	advanced, err := db.CreateClass(data.Class{ClassCode: "advanced", Name: "Advanced"})
	require.NoError(t, err)
	first, err := db.AddProblem(data.Problem{ClassID: basic, Name: "First", URL: "https://example.com/1"})
	require.NoError(t, err)
	second, err := db.AddProblem(data.Problem{ClassID: advanced, Name: "Second", URL: "https://example.com/2"})
	require.NoError(t, err)
	require.NoError(t, db.EnsureUser(data.User{UserID: "123", DisplayName: "Student"}))
	for _, code := range []string{"basics", "advanced"} {
		require.NoError(t, db.AddClassAccessRole(data.ClassAccessRole{ClassCode: code, RoleID: "student-role"}))
	}
	raw := script.NewFakeDiscordProvider()
	authorizer := providers.NewDiscordAuthorizer(raw, "admin", "global", map[string][]string{"advanced": {"custom"}})
	return &authFixture{db: db, svcs: services.NewServices(db, authorizer), raw: raw, basic: basic, advanced: advanced, p1: first, p2: second}
}
func actor(id string) authorization.Actor { return authorization.Actor{GuildID: "guild", UserID: id} }

func TestServiceUndoACSeparatesActorFromOwner(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, executor, code string
		roles                []string
		allowed              bool
	}{
		{"own solve", "123", "advanced", nil, true},
		{"other student", "other", "advanced", []string{"student-role"}, false},
		{"global mentor replaced", "other", "advanced", []string{"global"}, false},
		{"class mentor", "other", "advanced", []string{"custom"}, true},
		{"custom mentor cannot undo another class", "other", "basics", []string{"custom"}, false},
		{"global mentor inherited", "other", "basics", []string{"global"}, true},
		{"admin", "other", "advanced", []string{"admin"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture(t)
			f.raw.SetUserRoles(tt.executor, tt.roles)
			id := f.p2
			if tt.code == "basics" {
				id = f.p1
			}
			require.NoError(t, f.db.AddAC("123", id))
			err := f.svcs.Solution.UndoAC(actor(tt.executor), "123", id)
			if tt.allowed {
				assert.NoError(t, err)
			} else {
				assert.IsType(t, &serviceerr.ForbiddenError{}, err)
			}
			solved, e := f.db.HasUserSolvedProblem("123", id)
			require.NoError(t, e)
			assert.Equal(t, !tt.allowed, solved)
		})
	}
}

func TestServiceProblemMoveRequiresBothClasses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		roles   []string
		allowed bool
	}{
		{"source only", []string{"global"}, false}, {"destination only", []string{"custom"}, false},
		{"both", []string{"global", "custom"}, true}, {"admin", []string{"admin"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture(t)
			f.raw.SetUserRoles("actor", tt.roles)
			err := f.svcs.Problem.PatchProblemByProblemID(actor("actor"), service_model.PatchProblemParams{ProblemID: f.p1, PatchProblemFields: service_model.PatchProblemFields{ClassCode: service_model.PatchField[string]{Set: true, Value: "advanced"}}})
			if tt.allowed {
				assert.NoError(t, err)
			} else {
				assert.IsType(t, &serviceerr.ForbiddenError{}, err)
			}
			problem, e := f.db.GetProblemByID(f.p1)
			require.NoError(t, e)
			want := f.basic
			if tt.allowed {
				want = f.advanced
			}
			assert.Equal(t, want, problem.ClassID)
		})
	}
}

func TestServiceAuthorizationCannotBeBypassedByCallingWritesDirectly(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(*authFixture) error
	}{
		{"create problem", func(f *authFixture) error {
			_, err := f.svcs.Problem.CreateProblemIntoClass(actor("actor"), service_model.CreateProblemParams{ClassCode: "advanced", Name: "Unauthorized", URL: "https://example.com/new"})
			return err
		}},
		{"remove problem", func(f *authFixture) error { return f.svcs.Problem.RemoveProblemFromClass(actor("actor"), f.p2) }},
		{"direct grant", func(f *authFixture) error { return f.svcs.Solution.AddACToUserID(actor("actor"), "123", f.p2) }},
		{"grant enrolled student", func(f *authFixture) error {
			return f.svcs.Solution.AddACByPrincipal(actor("actor"), service_model.Principal{UserID: "123", Roles: []string{"student-role"}}, f.p2)
		}},
		{"create class", func(f *authFixture) error {
			_, err := f.svcs.Class.CreateClass(actor("actor"), "new", "New")
			return err
		}},
		{"delete class", func(f *authFixture) error { return f.svcs.Class.DeleteClassByClassCode(actor("actor"), "advanced") }},
		{"clear class", func(f *authFixture) error {
			return f.svcs.Class.ClearClassContentsByClassCode(actor("actor"), "advanced")
		}},
		{"delete user", func(f *authFixture) error { return f.svcs.User.ExterminateUser(actor("actor"), "123") }},
		{"link role", func(f *authFixture) error {
			return f.svcs.Class.AssignAccessRole(actor("actor"), service_model.ClassAccessRole{ClassCode: "advanced", RoleID: "new"})
		}},
		{"link room", func(f *authFixture) error {
			return f.svcs.Class.RegisterClassroom(actor("actor"), service_model.Classroom{ClassCode: "advanced", RoomID: "new"})
		}},
		{"export", func(f *authFixture) error {
			_, _, _, err := f.svcs.Score.GetClassExportData(actor("actor"), "advanced")
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture(t)
			assert.IsType(t, &serviceerr.ForbiddenError{}, tt.run(f))
			_, err := f.db.GetProblemByID(f.p2)
			assert.NoError(t, err)
			_, err = f.db.GetUserByUserID("123")
			assert.NoError(t, err)
			solved, err := f.db.HasUserSolvedProblem("123", f.p2)
			require.NoError(t, err)
			assert.False(t, solved)
			all, err := f.db.GetAllClasses()
			require.NoError(t, err)
			assert.Len(t, all, 2)
		})
	}
}

func TestServiceBatchesAuthorizeEveryEntryBeforeWriting(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"problems", "edits", "ACs"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture(t)
			f.raw.SetUserRoles("actor", []string{"global", "custom"})
			var execute func() ([]service_model.BatchEntryError, error)
			switch kind {
			case "problems":
				prepared, issues, err := f.svcs.Problem.PrepareCreationBatch(actor("actor"), []service_model.CreateProblemParams{{ClassCode: "basics", Name: "New1", URL: "https://example.com/new1"}, {ClassCode: "advanced", Name: "New2", URL: "https://example.com/new2"}})
				require.NoError(t, err)
				require.Empty(t, issues)
				execute = func() ([]service_model.BatchEntryError, error) {
					_, issues, err := f.svcs.Problem.ExecuteCreationBatch(actor("actor"), prepared)
					return issues, err
				}
			case "edits":
				prepared, issues := f.svcs.Problem.PrepareProblemPatchBatch(actor("actor"), []service_model.PatchProblemParams{{ProblemID: f.p1, PatchProblemFields: service_model.PatchProblemFields{Name: service_model.PatchField[string]{Set: true, Value: "Changed1"}}}, {ProblemID: f.p2, PatchProblemFields: service_model.PatchProblemFields{Name: service_model.PatchField[string]{Set: true, Value: "Changed2"}}}})
				require.Empty(t, issues)
				execute = func() ([]service_model.BatchEntryError, error) {
					_, issues, err := f.svcs.Problem.ExecuteProblemPatchBatch(actor("actor"), prepared)
					return issues, err
				}
			case "ACs":
				students := map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student-role"}}}
				prepared, issues, err := f.svcs.Solution.PrepareCreationBatch(actor("actor"), []service_model.CreateACParams{{UserID: "123", ProblemID: f.p1}, {UserID: "123", ProblemID: f.p2}}, students)
				require.NoError(t, err)
				require.Empty(t, issues)
				execute = func() ([]service_model.BatchEntryError, error) {
					_, issues, err := f.svcs.Solution.ExecuteCreationBatch(actor("actor"), prepared, students)
					return issues, err
				}
			}
			f.raw.SetUserRoles("actor", []string{"global"})
			issues, err := execute()
			require.NoError(t, err)
			require.Len(t, issues, 1)
			assert.IsType(t, &serviceerr.ForbiddenError{}, issues[0].Err)
			for _, entry := range []struct {
				id   int64
				name string
			}{{f.p1, "First"}, {f.p2, "Second"}} {
				problem, err := f.db.GetProblemByID(entry.id)
				require.NoError(t, err)
				assert.Equal(t, entry.name, problem.Name)
				solved, err := f.db.HasUserSolvedProblem("123", entry.id)
				require.NoError(t, err)
				assert.False(t, solved)
			}
			problems, err := f.db.GetProblemsByClass(f.basic)
			require.NoError(t, err)
			assert.Len(t, problems, 1)
		})
	}
}

func TestServiceProposalDecisionDoesNotConsumeAfterClassChangesDuringAuthorization(t *testing.T) {
	t.Parallel()
	for _, accepted := range []bool{false, true} {
		t.Run(map[bool]string{true: "approve", false: "reject"}[accepted], func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture(t)
			_, err := f.db.AddPendingConfirmation(data.PendingConfirmation{MessageID: "proposal", UserID: "123", ProblemID: f.p1})
			require.NoError(t, err)
			authorizer := mock_authorization.NewMockAuthorizer(gomock.NewController(t))
			authorizer.EXPECT().Authorize(actor("actor"), authorization.ManageACs, authorization.Resource{ClassCode: "basics", OwnerUserID: "123"}).DoAndReturn(func(authorization.Actor, authorization.Permission, authorization.Resource) error {
				return f.db.PatchProblemByProblemID(f.p1, data.ProblemPatch{ClassID: data.PatchField[int64]{Set: true, Value: f.advanced}})
			})
			svc := services.NewServices(f.db, authorizer)
			if accepted {
				_, err = svc.Solution.AcceptACProposal(actor("actor"), "proposal")
			} else {
				_, err = svc.Solution.RejectACProposal(actor("actor"), "proposal")
			}
			assert.IsType(t, &serviceerr.AuthorizationTargetChangedError{}, err)
			exists, err := f.db.ExistsPendingConfirmation("proposal")
			require.NoError(t, err)
			assert.True(t, exists)
			solved, err := f.db.HasUserSolvedProblem("123", f.p1)
			require.NoError(t, err)
			assert.False(t, solved)
		})
	}
}

func TestServiceMissingAuthorizerFailsClosed(t *testing.T) {
	t.Parallel()
	f := newAuthFixture(t)
	svc := services.NewServices(f.db, nil)
	err := svc.Solution.UndoAC(actor("123"), "123", f.p1)
	assert.IsType(t, &serviceerr.AuthorizationUnavailableError{}, err)
}
