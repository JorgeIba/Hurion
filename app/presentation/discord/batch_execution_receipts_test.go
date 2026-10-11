package discord

import (
	"fmt"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"math"
	"strings"
	"testing"
)

func TestBatchExecutionReceiptsSummarizeCountsAndClearActions(t *testing.T) {
	t.Parallel()
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			t.Parallel()
			ids := make([]int64, 100)
			for index := range ids {
				ids[index] = math.MaxInt64 - int64(index)
			}
			result := BuildProblemBatchExecutionResult(service_model.ProblemCreateBatchResult{CreatedProblemIDs: ids}, failed)
			assert.LessOrEqual(t, utf16Count(*result.Content), maxMessageContentLength)
			assert.Empty(t, *result.Components)
			assert.Empty(t, result.AllowedMentions.Parse)
			assert.Empty(t, result.Files)
			assert.NotContains(t, *result.Content, "Problem IDs")
			assert.Equal(t, failed, strings.Contains(*result.Content, "batch outcome could not be confirmed"))
			assert.NotContains(t, *result.Content, "No changes were applied")
		})
	}
	short := BuildProblemBatchExecutionResult(service_model.ProblemCreateBatchResult{CreatedProblemIDs: []int64{23, 42}}, false)
	assert.Contains(t, *short.Content, "Problems created: 2")
	assert.NotContains(t, *short.Content, "23, 42")
	assert.Empty(t, short.Files)
	ac := BuildACBatchExecutionResult(service_model.ACBatchResult{Recorded: 2, AlreadyRecorded: 3}, false)
	assert.Contains(t, *ac.Content, "ACs recorded: 2. Already recorded: 3.")
	assert.Empty(t, *ac.Components)
	assert.Empty(t, ac.AllowedMentions.Parse)
	failedAC := BuildACBatchExecutionResult(service_model.ACBatchResult{}, true)
	assert.Contains(t, *failedAC.Content, "Confirmed AC registrations: 0")
	assert.Contains(t, *failedAC.Content, "batch outcome could not be confirmed")
	assert.NotContains(t, *failedAC.Content, "Earlier writes were not rolled back")
	assert.NotContains(t, *failedAC.Content, "No changes were applied")
}

func TestBatchConfirmationValidationReceiptFitsDiscord(t *testing.T) {
	t.Parallel()
	issues := make([]JSONInputIssue, 100)
	for index := range issues {
		issues[index] = JSONInputIssue{Entry: index + 1, Message: strings.Repeat("🚀", 1000)}
	}
	receipt := BuildBatchValidationFailure(FormatBatchValidationErrors(issues))
	assert.LessOrEqual(t, utf16Count(*receipt.Content), maxMessageContentLength)
	assert.Contains(t, *receipt.Content, "No changes were applied")
	assert.Contains(t, *receipt.Content, "Submit the batch again")
	assert.Empty(t, *receipt.Components)
	assert.Empty(t, receipt.AllowedMentions.Parse)
}

func TestBatchValidationInstructionFitsAfterOmittedIssueFooter(t *testing.T) {
	t.Parallel()
	var issues []JSONInputIssue
	for index, size := range []int{500, 500, 500, 297, 500} {
		issues = append(issues, JSONInputIssue{Entry: index + 1, Message: strings.Repeat("x", size)})
	}
	receipt := BuildBatchValidationFailure(FormatBatchValidationErrors(issues))
	assert.LessOrEqual(t, utf16Count(*receipt.Content), maxMessageContentLength)
	assert.Contains(t, *receipt.Content, "more issue(s)")
	assert.Contains(t, *receipt.Content, "Submit the batch again.")
}
