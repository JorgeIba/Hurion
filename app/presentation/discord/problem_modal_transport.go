package discord

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// CreateProblemModalCustomID identifies the route before the encoded class context.
const CreateProblemModalCustomID = "huron:problem:create"

const EditProblemModalCustomID = "huron:problem:edit"

const (
	maxMessageContentLength  = 2000
	modalCustomIDSeparator   = "|"
	modalCustomIDMaxLength   = 100
	problemNameFieldID       = "name"
	problemURLFieldID        = "url"
	problemDifficultyFieldID = "difficulty"
	problemPriorityFieldID   = "priority"
	problemTopicFieldID      = "topic"
)

// BuildCreateProblemModalResponse presents raw values using native Discord labels.
// Required inputs guide entry; application validation belongs to the write flow.
func BuildCreateProblemModalResponse(modal CreateProblemModal) (*discordgo.InteractionResponse, error) {
	customID, err := buildCreateProblemModalCustomID(modal.ClassCode)
	if err != nil {
		return nil, err
	}
	return buildProblemModalResponse(modal.ProblemModalFields, customID, "Add problem")
}

// BuildEditProblemModalResponse carries the problem ID and destination class code as submit context.
func BuildEditProblemModalResponse(modal EditProblemModal) (*discordgo.InteractionResponse, error) {
	customID, err := buildEditProblemModalCustomID(modal.ClassCode, modal.ProblemID)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("Edit #%d · %s", modal.ProblemID, modal.ClassCode)
	for utf16Count(title) > 45 {
		runes := []rune(title)
		title = string(runes[:len(runes)-1])
	}

	response, err := buildProblemModalResponse(modal.ProblemModalFields, customID, title)
	if err != nil {
		return nil, fmt.Errorf("%w; repair it using !update_problem", err)
	}
	return response, nil
}

func buildProblemModalResponse(fields ProblemModalFields, customID, title string) (*discordgo.InteractionResponse, error) {
	for _, value := range []string{fields.Name, fields.URL, fields.Difficulty, fields.Priority, fields.Topic} {
		if utf16Count(value) > 4000 {
			return nil, fmt.Errorf("a field exceeds Discord's form limit")
		}
	}
	required, optional := true, false

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    title,
			Components: []discordgo.MessageComponent{
				discordgo.Label{
					Label: "Name",
					Component: discordgo.TextInput{
						CustomID: problemNameFieldID, Style: discordgo.TextInputShort,
						Value: fields.Name, Required: &required, Placeholder: "Two Sum",
					},
				},
				discordgo.Label{
					Label: "URL",
					Component: discordgo.TextInput{
						CustomID: problemURLFieldID, Style: discordgo.TextInputShort,
						Value: fields.URL, Required: &required, Placeholder: "https://example.com/problem",
					},
				},
				discordgo.Label{
					Label:       "Difficulty",
					Description: "Optional: Easy, Medium, Hard, or Not specified",
					Component: discordgo.TextInput{
						CustomID: problemDifficultyFieldID, Style: discordgo.TextInputShort,
						Value: fields.Difficulty, Required: &optional,
					},
				},
				discordgo.Label{
					Label:       "Priority",
					Description: "Optional: Essential, Recommended, Supplementary, Not required, or Discouraged",
					Component: discordgo.TextInput{
						CustomID: problemPriorityFieldID, Style: discordgo.TextInputShort,
						Value: fields.Priority, Required: &optional,
					},
				},
				discordgo.Label{
					Label: "Topic",
					Component: discordgo.TextInput{
						CustomID: problemTopicFieldID, Style: discordgo.TextInputShort,
						Value: fields.Topic, Required: &optional, Placeholder: "Binary search",
					},
				},
			},
		},
	}, nil
}

// ModalRouteFromCustomID recognizes a route even when its context is malformed,
// so the matching controller can respond with a form error.
func ModalRouteFromCustomID(customID string) string {
	route, _, _ := strings.Cut(customID, modalCustomIDSeparator)
	return route
}

func buildCreateProblemModalCustomID(classCode string) (string, error) {
	if strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("missing problem modal class code")
	}

	customID := fmt.Sprintf("%s%s%s", CreateProblemModalCustomID, modalCustomIDSeparator, url.PathEscape(classCode))

	if len(customID) > modalCustomIDMaxLength {
		return "", fmt.Errorf("problem modal custom ID exceeds Discord's %d-character limit", modalCustomIDMaxLength)
	}
	return customID, nil
}

func buildEditProblemModalCustomID(classCode string, problemID int64) (string, error) {
	if strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("missing problem modal class code")
	}
	if problemID <= 0 {
		return "", fmt.Errorf("invalid problem modal context")
	}
	customID := fmt.Sprintf("%s%s%s%s%d", EditProblemModalCustomID, modalCustomIDSeparator, url.PathEscape(classCode), modalCustomIDSeparator, problemID)
	if len(customID) > modalCustomIDMaxLength {
		return "", fmt.Errorf("problem modal custom ID exceeds Discord's %d-character limit", modalCustomIDMaxLength)
	}
	return customID, nil
}

func parseCreateProblemModalCustomID(customID string) (string, error) {
	route, encodedClass, found := strings.Cut(customID, modalCustomIDSeparator)
	if route != CreateProblemModalCustomID || !found || len(customID) > modalCustomIDMaxLength {
		return "", fmt.Errorf("invalid problem modal ID")
	}

	classCode, err := url.PathUnescape(encodedClass)
	if err != nil || strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("invalid problem modal class code")
	}
	return classCode, nil
}

func parseEditProblemModalCustomID(customID string) (string, int64, error) {
	parts := strings.Split(customID, modalCustomIDSeparator)
	if len(parts) != 3 || parts[0] != EditProblemModalCustomID || len(customID) > modalCustomIDMaxLength {
		return "", 0, fmt.Errorf("invalid problem modal context")
	}

	classCode, err := url.PathUnescape(parts[1])
	if err != nil || strings.TrimSpace(classCode) == "" {
		return "", 0, fmt.Errorf("invalid problem modal class code")
	}

	problemID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || problemID <= 0 {
		return "", 0, fmt.Errorf("invalid problem modal problem ID")
	}
	return classCode, problemID, nil
}

// ParseCreateProblemModal reads the SDK-decoded label inputs by their identity,
// preserving text exactly and recovering the selected class code from its context.
// Shape checks do not validate problem domain rules.
func ParseCreateProblemModal(data discordgo.ModalSubmitInteractionData) (CreateProblemModal, error) {
	classCode, err := parseCreateProblemModalCustomID(data.CustomID)
	if err != nil {
		return CreateProblemModal{}, err
	}

	fields, err := parseProblemModalFields(data)
	if err != nil {
		return CreateProblemModal{}, err
	}
	return CreateProblemModal{ClassCode: classCode, ProblemModalFields: fields}, nil
}

func ParseEditProblemModal(data discordgo.ModalSubmitInteractionData) (EditProblemModal, error) {
	classCode, problemID, err := parseEditProblemModalCustomID(data.CustomID)
	if err != nil {
		return EditProblemModal{}, err
	}

	fields, err := parseProblemModalFields(data)
	if err != nil {
		return EditProblemModal{}, err
	}
	return EditProblemModal{ProblemID: problemID, ClassCode: classCode, ProblemModalFields: fields}, nil
}

func parseProblemModalFields(data discordgo.ModalSubmitInteractionData) (ProblemModalFields, error) {
	var result ProblemModalFields
	fields := map[string]*string{
		problemNameFieldID:       &result.Name,
		problemURLFieldID:        &result.URL,
		problemDifficultyFieldID: &result.Difficulty,
		problemPriorityFieldID:   &result.Priority,
		problemTopicFieldID:      &result.Topic,
	}

	for _, component := range data.Components {
		label, ok := component.(*discordgo.Label)
		if !ok || label == nil {
			return ProblemModalFields{}, fmt.Errorf("expected a problem form label")
		}
		input, ok := label.Component.(*discordgo.TextInput)
		if !ok || input == nil {
			return ProblemModalFields{}, fmt.Errorf("expected a problem form text input")
		}
		field, ok := fields[input.CustomID]
		if !ok {
			return ProblemModalFields{}, fmt.Errorf("unknown or repeated problem field %q", input.CustomID)
		}
		*field = input.Value
		delete(fields, input.CustomID)
	}
	if len(fields) != 0 {
		return ProblemModalFields{}, fmt.Errorf("incomplete problem form")
	}

	return result, nil
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
