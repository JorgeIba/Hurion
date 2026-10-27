package mentor

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestACProposalReactionDoesNotReportSuccessIfConsumedAfterLookup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, emoji string }{{"approve", common.REACTION_YES}, {"reject", common.REACTION_NO}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			validator := mock_services.NewMockAuthorizer(ctrl)
			validator.EXPECT().CanManageClasses(services.UserActor("mentor"), "A").Return(true, nil)
			store.EXPECT().GetPendingConfirmation("msg_123").Return("student", int64(77), nil)
			store.EXPECT().GetProblemByID(int64(77)).Return(data.Problem{ID: 77, ClassID: 1}, nil)
			store.EXPECT().GetClassByClassID(int64(1)).Return(data.Class{ID: 1, ClassCode: "A"}, nil)
			unavailable := dataerr.NewNotFoundError("proposal was replaced or resolved", nil)
			if tc.emoji == common.REACTION_YES {
				store.EXPECT().AcceptPendingConfirmation("msg_123").Return(data.PendingConfirmation{}, unavailable)
			} else {
				store.EXPECT().RemovePendingConfirmation("msg_123").Return(data.PendingConfirmation{}, unavailable)
			}
			out := make(chan dispatcher.MessageResponse, 1)
			event := makeReactionContext("mentor", "guild", tc.emoji, services.NewServices(store, validator), nil, out)
			HandlePendingConfirmation(event)
			assert.Empty(t, out, "an obsolete proposal must not produce a success reaction or an unexpected-error message")
		})
	}
}
