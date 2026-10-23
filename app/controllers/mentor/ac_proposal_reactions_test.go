package mentor

import (
	"testing"

	authtest "github.com/JorgeIba/Hurion/app/authorization/authtest"
	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestACProposalReactionDoesNotReportSuccessIfConsumedAfterLookup(t *testing.T) {
	for _, emoji := range []string{common.REACTION_YES, common.REACTION_NO} {
		t.Run(emoji, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			validator := authtest.NewMockRoleChecker(ctrl)
			store.EXPECT().GetPendingConfirmation("msg_123").Return("123", int64(77), nil)
			store.EXPECT().GetProblemByID(int64(77)).Return(data.Problem{ID: 77, ClassID: 1}, nil)
			store.EXPECT().GetClassByClassID(int64(1)).Return(data.Class{ID: 1, ClassCode: "basics"}, nil)
			store.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			unavailable := dataerr.NewNotFoundError("proposal was replaced or resolved", nil)
			if emoji == common.REACTION_YES {
				store.EXPECT().AcceptPendingConfirmationInClass("msg_123", gomock.Any()).Return(data.PendingConfirmation{}, unavailable)
			} else {
				store.EXPECT().RemovePendingConfirmationInClass("msg_123", gomock.Any()).Return(data.PendingConfirmation{}, unavailable)
			}
			out := make(chan dispatcher.MessageResponse, 1)
			event := makeReactionContext("mentor", "guild", emoji, services.NewServices(store, authtest.Allow{}), validator, nil, out)
			HandlePendingConfirmation(event)
			require.Empty(t, out, "an obsolete proposal must not produce a success reaction or an unexpected-error message")
		})
	}
}
