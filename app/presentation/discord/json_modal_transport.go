package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	jsonTextFieldID = "json"
	jsonFileFieldID = "file"
)

// JSONModalSource selects the uploaded file when present, otherwise pasted text.
type JSONModalSource struct {
	Text       string
	Attachment *discordgo.MessageAttachment
}

func buildJSONModalResponse(customID, title, example string) *discordgo.InteractionResponse {
	optional := false
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID, Title: title,
			Components: []discordgo.MessageComponent{
				discordgo.Label{
					Label: "JSON", Description: "Paste a JSON array or upload a .json file. Uploaded files take precedence.",
					Component: discordgo.TextInput{CustomID: jsonTextFieldID, Style: discordgo.TextInputParagraph, Required: &optional, MaxLength: 4000, Placeholder: example},
				},
				discordgo.Label{
					Label: "JSON file", Description: "One .json file, up to 5 MiB. If provided, pasted JSON is ignored.",
					Component: discordgo.FileUpload{CustomID: jsonFileFieldID, Required: &optional, MinValues: new(0), MaxValues: 1},
				},
			},
		},
	}
}

// parseJSONModalSource verifies the form structure and resolves the selected
// attachment from Discord's payload; downloading and JSON parsing are separate.
func parseJSONModalSource(data discordgo.ModalSubmitInteractionData, customID string) (JSONModalSource, error) {
	if data.CustomID != customID {
		return JSONModalSource{}, fmt.Errorf("invalid batch form ID")
	}
	var source JSONModalSource
	seenText, seenFile := false, false
	var fileIDs []string
	for _, component := range data.Components {
		label, ok := component.(*discordgo.Label)
		if !ok || label == nil {
			return JSONModalSource{}, fmt.Errorf("expected a batch form label")
		}
		switch field := label.Component.(type) {
		case *discordgo.TextInput:
			if field == nil || field.CustomID != jsonTextFieldID || seenText {
				return JSONModalSource{}, fmt.Errorf("unknown or repeated JSON field")
			}
			seenText, source.Text = true, field.Value
		case *discordgo.FileUpload:
			if field == nil || field.CustomID != jsonFileFieldID || seenFile {
				return JSONModalSource{}, fmt.Errorf("unknown or repeated file field")
			}
			seenFile, fileIDs = true, field.Values
		default:
			return JSONModalSource{}, fmt.Errorf("expected JSON text or a file upload")
		}
	}
	if !seenText || !seenFile {
		return JSONModalSource{}, fmt.Errorf("incomplete batch form")
	}
	if len(fileIDs) > 1 {
		return JSONModalSource{}, fmt.Errorf("upload only one .json file")
	}
	hasText := strings.TrimSpace(source.Text) != ""
	if !hasText && len(fileIDs) == 0 {
		return JSONModalSource{}, fmt.Errorf("paste JSON or upload one .json file")
	}
	if len(fileIDs) == 1 {
		attachment := data.Resolved.Attachments[fileIDs[0]]
		if attachment == nil || attachment.ID != fileIDs[0] {
			return JSONModalSource{}, fmt.Errorf("could not resolve the uploaded file; upload it again")
		}
		source.Text, source.Attachment = "", attachment
	} else if utf16Count(source.Text) > 4000 {
		return JSONModalSource{}, fmt.Errorf("pasted JSON exceeds Discord's 4000-character form limit; upload a .json file")
	}
	return source, nil
}

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
