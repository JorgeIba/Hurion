package discord

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// CreateProblemModalCustomID identifies the route before the encoded class context.
const CreateProblemModalCustomID = "huron:problem:create"

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
	required, optional := true, false

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    "Add problem",
			Components: []discordgo.MessageComponent{
				discordgo.Label{
					Label: "Name",
					Component: discordgo.TextInput{
						CustomID: problemNameFieldID, Style: discordgo.TextInputShort,
						Value: modal.Name, Required: &required, Placeholder: "Two Sum",
					},
				},
				discordgo.Label{
					Label: "URL",
					Component: discordgo.TextInput{
						CustomID: problemURLFieldID, Style: discordgo.TextInputShort,
						Value: modal.URL, Required: &required, Placeholder: "https://example.com/problem",
					},
				},
				discordgo.Label{
					Label:       "Difficulty",
					Description: "Optional: Easy, Medium, Hard, or Not specified",
					Component: discordgo.TextInput{
						CustomID: problemDifficultyFieldID, Style: discordgo.TextInputShort,
						Value: modal.Difficulty, Required: &optional,
					},
				},
				discordgo.Label{
					Label:       "Priority",
					Description: "Optional: Essential, Recommended, Supplementary, Not required, or Discouraged",
					Component: discordgo.TextInput{
						CustomID: problemPriorityFieldID, Style: discordgo.TextInputShort,
						Value: modal.Priority, Required: &optional,
					},
				},
				discordgo.Label{
					Label: "Topic",
					Component: discordgo.TextInput{
						CustomID: problemTopicFieldID, Style: discordgo.TextInputShort,
						Value: modal.Topic, Required: &optional, Placeholder: "Binary search",
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
	customID := CreateProblemModalCustomID + modalCustomIDSeparator + url.PathEscape(classCode)
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

// ParseCreateProblemModal reads the SDK-decoded label inputs by their identity,
// preserving text exactly and recovering the selected class code from its context.
// Shape checks do not validate problem domain rules.
func ParseCreateProblemModal(data discordgo.ModalSubmitInteractionData) (CreateProblemModal, error) {
	classCode, err := parseCreateProblemModalCustomID(data.CustomID)
	if err != nil {
		return CreateProblemModal{}, err
	}

	modal := CreateProblemModal{ClassCode: classCode}
	fields := map[string]*string{
		problemNameFieldID:       &modal.Name,
		problemURLFieldID:        &modal.URL,
		problemDifficultyFieldID: &modal.Difficulty,
		problemPriorityFieldID:   &modal.Priority,
		problemTopicFieldID:      &modal.Topic,
	}

	for _, component := range data.Components {
		label, ok := component.(*discordgo.Label)
		if !ok || label == nil {
			return CreateProblemModal{}, fmt.Errorf("expected a problem form label")
		}
		input, ok := label.Component.(*discordgo.TextInput)
		if !ok || input == nil {
			return CreateProblemModal{}, fmt.Errorf("expected a problem form text input")
		}
		field, ok := fields[input.CustomID]
		if !ok {
			return CreateProblemModal{}, fmt.Errorf("unknown or repeated problem field %q", input.CustomID)
		}
		*field = input.Value
		delete(fields, input.CustomID)
	}
	if len(fields) != 0 {
		return CreateProblemModal{}, fmt.Errorf("incomplete problem form")
	}

	return modal, nil
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
