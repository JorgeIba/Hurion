package mentor

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACBatchAccessErrorsUseVerifiedDisplayName(t *testing.T) {
	t.Parallel()
	entries := batchInput[acInput]{{UserID: "123", ProblemID: 13}}
	issues := []batchInputIssue{{entry: 1, err: serviceerr.NewUserNotEnrolledError("no access", nil)}}
	for _, name := range []string{"Jorge Luis", "", "Unknown", "name`\n@everyone"} {
		t.Run(name, func(t *testing.T) {
			students := map[string]service_model.UserWithRoles{"123": {User: service_model.User{UserID: "123", DisplayName: name}}}
			mapped := mapACBatchValidationIssues(entries, issues, students)
			require.Len(t, mapped, 1)
			assert.Equal(t, 1, mapped[0].Entry)
			want := name
			if name == "" || name == "Unknown" {
				want = "123"
			}
			if name == "name`\n@everyone" {
				want = "name' @everyone"
			}
			assert.Equal(t, "User `"+want+"` doesn't have access to the class of problem `13`", mapped[0].Message)
		})
	}
	mapped := mapACBatchValidationIssues(entries, issues, nil)
	assert.Contains(t, mapped[0].Message, "`123`")
}

func TestACBatchConfirmationErrorsUseCurrentDiscordName(t *testing.T) {
	t.Parallel()
	batch := []service_model.PreparedAC{{User: service_model.User{UserID: "123", DisplayName: "Old Name"}, Problem: service_model.Problem{ID: 13}}}
	students := map[string]service_model.UserWithRoles{"123": {User: service_model.User{UserID: "123", DisplayName: "Current Name"}}}
	issues := []service_model.BatchEntryError{{Entry: 1, Err: serviceerr.NewUserNotEnrolledError("no access", nil)}}
	mapped := mapACBatchExecutionIssues(batch, issues, students)
	assert.Contains(t, mapped[0].Message, "`Current Name`")
	assert.NotContains(t, mapped[0].Message, "Old Name")
}
