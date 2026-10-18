package mentor

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
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
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			store.EXPECT().ExistsPendingConfirmation("msg_123").Return(true, nil)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			unavailable := dataerr.NewNotFoundError("proposal was replaced or resolved", nil)
			if emoji == common.REACTION_YES {
				store.EXPECT().AcceptPendingConfirmation("msg_123").Return(data.PendingConfirmation{}, unavailable)
			} else {
				store.EXPECT().RemovePendingConfirmation("msg_123").Return(data.PendingConfirmation{}, unavailable)
			}
			out := make(chan dispatcher.MessageResponse, 1)
			event := makeReactionContext("mentor", "guild", emoji, services.NewServices(store), validator, nil, out)
			HandlePendingConfirmation(event)
			require.Empty(t, out, "an obsolete proposal must not produce a success reaction or an unexpected-error message")
		})
	}
}
