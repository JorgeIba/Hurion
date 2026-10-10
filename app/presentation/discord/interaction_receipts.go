package discord

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// JSONInputIssue carries parser locations for a private Discord receipt.
type JSONInputIssue struct {
	Entry   int
	Line    int
	Column  int
	Message string
}

func FormatJSONParseResult(count int, issues []JSONInputIssue) string {
	if len(issues) > 0 {
		return formatJSONInputErrors(issues)
	}
	return fmt.Sprintf("JSON parsed. Entries: %d. No changes were applied.", count)
}

func formatJSONInputErrors(issues []JSONInputIssue) string {
	return formatInputIssues(fmt.Sprintf("Could not read JSON: %d issue(s).", len(issues)), issues)
}

func FormatBatchValidationErrors(issues []JSONInputIssue) string {
	return formatInputIssues(fmt.Sprintf("Batch validation failed: %d issue(s). No changes were applied.", len(issues)), issues)
}

func formatInputIssues(content string, issues []JSONInputIssue) string {
	for index, issue := range issues {
		location := "Input"
		if issue.Entry > 0 {
			location = fmt.Sprintf("Entry %d", issue.Entry)
		}
		if issue.Line > 0 {
			location += fmt.Sprintf(" (line %d, column %d)", issue.Line, issue.Column)
		}
		// One malformed value must not consume the whole receipt. Preserve the
		// entry/location and reserve room for the omitted-error count.
		message := truncateJSONInputError(issue.Message, 500)
		line := fmt.Sprintf("\n%s: %s", location, message)
		if utf16Count(content+line) > 1900 {
			content += fmt.Sprintf("\n%d more issue(s). Correct these and submit again.", len(issues)-index)
			break
		}
		content += line
	}
	return content
}

func truncateJSONInputError(message string, limit int) string {
	message = strings.ReplaceAll(message, "\n", " ")
	var result strings.Builder
	length := 0
	for _, char := range message {
		width := 1
		if char > 0xFFFF {
			width = 2
		}
		if length+width > limit {
			result.WriteString("…")
			break
		}
		result.WriteRune(char)
		length += width
	}
	return result.String()
}

// FormatCreateProblemError keeps mapped errors within Discord's message limit.
func FormatCreateProblemError(content, classCode string, recoverDraft bool) string {
	if recoverDraft {
		content += ". Try /problem add class:" + classCode + " again."
	}
	if utf16Count(content) <= maxMessageContentLength {
		return content
	}
	if !recoverDraft {
		return "Could not add the problem. Please try /problem add again."
	}
	// The modal identifier already bounds the selected class code's length.
	return "Could not add the problem. Please check the values and try /problem add class:" + classCode + " again."
}

// FormatCreatedProblemReceipt identifies the saved problem without truncating stored values.
func FormatCreatedProblemReceipt(problemID int64, classCode, problemName string) string {
	identity := fmt.Sprintf("Problem `%d` was added to class %q.", problemID, strings.TrimSpace(classCode))
	content := identity + fmt.Sprintf("\nName: %q", strings.TrimSpace(problemName))
	if utf16Count(content) > maxMessageContentLength {
		return identity
	}
	return content
}

func FormatEditProblemError(content string, problemID int64, classCode string, recoverDraft bool) string {
	if recoverDraft {
		content += fmt.Sprintf(" Try /problem edit problem:%d class:%s again.", problemID, classCode)
	}

	if utf16Count(content) <= maxMessageContentLength {
		return content
	}

	if !recoverDraft {
		return fmt.Sprintf("Could not update problem `%d`. Please try /problem edit again.", problemID)
	}

	retry := fmt.Sprintf("Could not update problem `%d`. Try /problem edit problem:%d class:%s again.", problemID, problemID, classCode)
	if utf16Count(retry) > maxMessageContentLength {
		return fmt.Sprintf("Could not update problem `%d`. Retry /problem edit with problem:%d and the same destination class code to recover your values.", problemID, problemID)
	}
	return retry
}

func FormatUpdatedProblemReceipt(problemID int64, classCode, name string) string {
	identity := fmt.Sprintf("Problem `%d` was updated in class %q.", problemID, classCode)
	content := identity + fmt.Sprintf("\nName: %q", strings.TrimSpace(name))
	if utf16Count(content) > maxMessageContentLength {
		return fmt.Sprintf("Problem `%d` was updated.", problemID)
	}
	return content
}

func FormatProblemBatchValidationResult(prepared []service_model.PreparedProblem) string {
	classes := make(map[int64]struct{})
	for _, entry := range prepared {
		classes[entry.Class.ID] = struct{}{}
	}
	return fmt.Sprintf("Problem batch validated. Entries: %d. Classes: %d. No changes were applied.", len(prepared), len(classes))
}

func FormatACBatchValidationResult(prepared []service_model.PreparedAC) string {
	students := make(map[string]struct{})
	alreadyRecorded := 0
	for _, entry := range prepared {
		students[entry.User.UserID] = struct{}{}
		if entry.AlreadyRecorded {
			alreadyRecorded++
		}
	}

	return fmt.Sprintf("AC batch validated. Entries: %d. Students: %d. Already recorded entries: %d. No changes were applied.", len(prepared), len(students), alreadyRecorded)
}

const BatchUnavailableMessage = "This batch is no longer available. Submit it again to validate a new batch."

func BuildValidatedBatchPreview(content, confirmID, cancelID string) *discordgo.WebhookEdit {
	content += "\nConfirm to apply this batch?"
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: confirmID, Label: "Confirm", Style: discordgo.SuccessButton},
		discordgo.Button{CustomID: cancelID, Label: "Cancel", Style: discordgo.SecondaryButton},
	}}}

	return &discordgo.WebhookEdit{Content: &content, Components: &components, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}
}

func BuildBatchCancellationResult(cancelled bool) *discordgo.WebhookEdit {
	content := BatchUnavailableMessage
	if cancelled {
		content = "Batch cancelled. No changes were applied."
	}

	components := []discordgo.MessageComponent{}
	return &discordgo.WebhookEdit{
		Content:    &content,
		Components: &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	}
}

// FormatJSONReadError preserves document locations when available.
func FormatJSONReadError(issue JSONInputIssue) string {
	if issue.Line > 0 {
		return FormatJSONParseResult(0, []JSONInputIssue{issue})
	}
	return "Could not read the batch form: " + issue.Message
}

func buildCompletedBatchResponse(content string) *discordgo.WebhookEdit {
	components := []discordgo.MessageComponent{}
	return &discordgo.WebhookEdit{Content: &content, Components: &components, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}
}

func BuildBatchValidationFailure(content string) *discordgo.WebhookEdit {
	return buildCompletedBatchResponse(content + "\nSubmit the batch again.")
}

func BuildProblemBatchExecutionResult(result service_model.ProblemCreateBatchResult, failed bool) *discordgo.WebhookEdit {
	content := fmt.Sprintf("Batch completed. Problems created: %d.", len(result.CreatedProblemIDs))
	if failed {
		content = fmt.Sprintf("Batch stopped. Confirmed problem creations: %d.\nEarlier writes were not rolled back. The failed operation may also have been applied. Check the class before submitting again.", len(result.CreatedProblemIDs))
	}

	return buildProblemBatchIDResponse(content, result.CreatedProblemIDs, "created-problem-ids.txt")
}

func buildProblemBatchIDResponse(content string, problemIDs []int64, filename string) *discordgo.WebhookEdit {
	ids := make([]string, len(problemIDs))
	for index, id := range problemIDs {
		ids[index] = strconv.FormatInt(id, 10)
	}

	response := buildCompletedBatchResponse(content)
	if len(ids) > 0 {
		list := "\nProblem IDs (in batch order): " + strings.Join(ids, ", ")
		if utf16Count(content+list) <= maxMessageContentLength {
			content += list
		} else {
			content += "\nProblem IDs (in batch order) are attached."
			response.Files = []*discordgo.File{{Name: filename, ContentType: "text/plain", Reader: strings.NewReader(strings.Join(ids, "\n"))}}
		}
	}
	response.Content = &content
	return response
}

func BuildACBatchExecutionResult(result service_model.ACBatchResult, failed bool) *discordgo.WebhookEdit {
	content := fmt.Sprintf("Batch completed. ACs recorded: %d. Already recorded: %d.", result.Recorded, result.AlreadyRecorded)
	if failed {
		content = fmt.Sprintf("Batch stopped. Confirmed AC registrations: %d. Already recorded: %d.\nThe batch outcome could not be confirmed; it may have been applied. Check the students' ACs before submitting again.", result.Recorded, result.AlreadyRecorded)
	}
	return buildCompletedBatchResponse(content)
}

func FormatProblemEditBatchValidationResult(prepared []service_model.PreparedProblemUpdate) string {
	classes := make(map[int64]struct{})
	for _, entry := range prepared {
		classes[entry.Class.ID] = struct{}{}
	}
	return fmt.Sprintf("Problem edit batch validated. Entries: %d. Destination classes: %d. No changes were applied.", len(prepared), len(classes))
}
func BuildProblemEditBatchExecutionResult(result service_model.ProblemUpdateBatchResult, failed bool) *discordgo.WebhookEdit {
	content := fmt.Sprintf("Batch completed. Problem updates applied: %d.", len(result.UpdatedProblemIDs))
	if failed {
		content = fmt.Sprintf("Batch stopped. Confirmed problem updates: %d.\nEarlier writes were not rolled back. The failed operation may also have been applied. Check the problems before submitting again.", len(result.UpdatedProblemIDs))
	}
	return buildProblemBatchIDResponse(content, result.UpdatedProblemIDs, "updated-problem-ids.txt")
}
