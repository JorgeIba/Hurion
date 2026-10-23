package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/authorization/mock_authorization"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestServiceAuthorizationErrors(t *testing.T) {
	t.Parallel()
	unavailable := errors.New("role source unavailable")
	for _, tt := range []struct {
		name     string
		actor    authorization.Actor
		decision error
		want     error
	}{
		{name: "allowed", actor: authorization.Actor{GuildID: "guild", UserID: "actor"}},
		{name: "denied", actor: authorization.Actor{GuildID: "guild", UserID: "actor"}, decision: authorization.ErrDenied, want: &serviceerr.ForbiddenError{}},
		{name: "lookup failed", actor: authorization.Actor{GuildID: "guild", UserID: "actor"}, decision: unavailable, want: &serviceerr.AuthorizationUnavailableError{}},
		{name: "missing actor", want: &serviceerr.ForbiddenError{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			authorizer := mock_authorization.NewMockAuthorizer(gomock.NewController(t))
			resource := authorization.Resource{ClassCode: "advanced"}
			if tt.actor.UserID != "" {
				authorizer.EXPECT().Authorize(tt.actor, authorization.ManageACs, resource).Return(tt.decision)
			}
			err := requireAuthorization(authorizer, tt.actor, authorization.ManageACs, resource)
			if tt.want == nil {
				assert.NoError(t, err)
			} else {
				assert.IsType(t, tt.want, err)
			}
			if tt.decision == unavailable {
				assert.ErrorIs(t, err, unavailable)
			}
		})
	}
}

func TestBatchAuthorizationResolvesAndChecksEachSourceClassOnce(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	authorizer := mock_authorization.NewMockAuthorizer(gomock.NewController(t))
	actor := authorization.Actor{GuildID: "guild", UserID: "mentor"}
	var problems []data.Problem
	var inputs []service_model.PatchProblemParams
	for id := int64(1); id <= 100; id++ {
		problems = append(problems, data.Problem{ID: id, ClassID: 7})
		inputs = append(inputs, service_model.PatchProblemParams{ProblemID: id, PatchProblemFields: service_model.PatchProblemFields{Name: service_model.PatchField[string]{Set: true, Value: "Changed"}}})
	}
	store.EXPECT().GetProblemsByIDs(gomock.Any()).Return(problems, nil)
	store.EXPECT().GetClassByClassID(int64(7)).Return(data.Class{ID: 7, ClassCode: "advanced"}, nil)
	authorizer.EXPECT().Authorize(actor, authorization.ManageProblems, authorization.Resource{ClassCode: "advanced"}).Return(nil)
	prepared, issues := NewProblemService(store, authorizer).PrepareProblemPatchBatch(actor, inputs)
	require.Empty(t, issues)
	assert.Len(t, prepared, 100)
}
