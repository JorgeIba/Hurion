package interactiontransport

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	problemNameFieldID            = "name"
	problemURLFieldID             = "url"
	problemDifficultyFieldID      = "difficulty"
	problemPriorityFieldID        = "priority"
	problemTopicFieldID           = "topic"
	unassignedMetadataSelectValue = "unset"
)

// BuildCreateProblemModalResponse presents text and fixed metadata choices using Discord labels.
// Required inputs guide entry; application validation belongs to the write flow.
func BuildCreateProblemModalResponse(modal CreateProblemModal) (*discordgo.InteractionResponse, error) {
	customID, err := CreateProblemModalID(modal.ClassCode)
	if err != nil {
		return nil, err
	}
	return buildProblemModalResponse(modal.ProblemModalFields, customID, "Add problem")
}

// BuildEditProblemModalResponse carries the problem ID and destination class code as submit context.
func BuildEditProblemModalResponse(modal EditProblemModal) (*discordgo.InteractionResponse, error) {
	customID, err := EditProblemModalID(modal.ClassCode, modal.ProblemID)
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
	for _, value := range []string{fields.Name, fields.URL, problemModalTextValue(fields.Topic)} {
		if utf16Count(value) > 4000 {
			return nil, fmt.Errorf("a field exceeds Discord's form limit")
		}
	}
	required, optional := true, false

	// Length limits are enforced by the service. Omit custom input limits so
	// oversized drafts can be prefilled for correction without truncation.
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    title,
			Components: []discordgo.MessageComponent{
				discordgo.Label{
					Label:       "Name",
					Description: fmt.Sprintf("Up to %d characters", service_model.ProblemNameMaxLength),
					Component: discordgo.TextInput{
						CustomID: problemNameFieldID, Style: discordgo.TextInputShort,
						Value: fields.Name, Required: &required, Placeholder: "Two Sum",
					},
				},
				discordgo.Label{
					Label:       "URL",
					Description: fmt.Sprintf("HTTP or HTTPS; up to %d characters", service_model.ProblemURLMaxLength),
					Component: discordgo.TextInput{
						CustomID: problemURLFieldID, Style: discordgo.TextInputShort,
						Value: fields.URL, Required: &required, Placeholder: "https://example.com/problem",
					},
				},
				discordgo.Label{
					Label:       "Difficulty",
					Description: "Optional: Easy, Medium, Hard, or Not specified",
					Component:   problemDifficultySelect(fields.Difficulty),
				},
				discordgo.Label{
					Label:       "Priority",
					Description: "Optional: Essential, Recommended, Supplementary, Not required, or Discouraged",
					Component:   problemPrioritySelect(fields.Priority),
				},
				discordgo.Label{
					Label:       "Topic",
					Description: fmt.Sprintf("Optional; up to %d characters", service_model.ProblemTopicMaxLength),
					Component: discordgo.TextInput{
						CustomID: problemTopicFieldID, Style: discordgo.TextInputShort,
						Value: problemModalTextValue(fields.Topic), Required: &optional, Placeholder: "Binary search",
					},
				},
			},
		},
	}, nil
}

func problemModalTextValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func problemDifficultySelect(current *string) modalStringSelect {
	return problemMetadataSelect(problemDifficultyFieldID, "Choose a difficulty", current, problemDifficultyOptions())
}

func problemDifficultyOptions() []modalStringSelectOption {
	return []modalStringSelectOption{
		{Label: "Easy", Value: service_model.DifficultyEasy},
		{Label: "Medium", Value: service_model.DifficultyMedium},
		{Label: "Hard", Value: service_model.DifficultyHard},
		{Label: presentation.DifficultyUnspecifiedLabel, Value: unassignedMetadataSelectValue},
	}
}

func problemPrioritySelect(current *string) modalStringSelect {
	return problemMetadataSelect(problemPriorityFieldID, "Choose a priority", current, problemPriorityOptions())
}

func problemPriorityOptions() []modalStringSelectOption {
	return []modalStringSelectOption{
		{Label: "Essential", Value: service_model.PriorityEssential},
		{Label: "Recommended", Value: service_model.PriorityRecommended},
		{Label: "Supplementary", Value: service_model.PrioritySupplementary},
		{Label: "Not required", Value: service_model.PriorityNotRequired},
		{Label: "Discouraged", Value: service_model.PriorityDiscouraged},
		{Label: presentation.PriorityUnspecifiedLabel, Value: unassignedMetadataSelectValue},
	}
}

func problemMetadataSelect(customID, placeholder string, current *string, options []modalStringSelectOption) modalStringSelect {
	currentValue := unassignedMetadataSelectValue
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	for i := range options {
		options[i].Default = strings.EqualFold(currentValue, options[i].Value)
	}
	required := false
	return modalStringSelect{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
		Required:    &required,
	}
}

func parseProblemModalFields(data discordgo.ModalSubmitInteractionData) (ProblemModalFields, error) {
	var result ProblemModalFields
	pendingFields := map[string]bool{
		problemNameFieldID:       true,
		problemURLFieldID:        true,
		problemDifficultyFieldID: true,
		problemPriorityFieldID:   true,
		problemTopicFieldID:      true,
	}

	for _, component := range data.Components {
		label, ok := component.(*discordgo.Label)
		if !ok || label == nil {
			return ProblemModalFields{}, fmt.Errorf("expected a problem form label")
		}
		customID, value, err := parseProblemModalField(label.Component)
		if err != nil {
			return ProblemModalFields{}, err
		}
		if !pendingFields[customID] {
			return ProblemModalFields{}, fmt.Errorf("unknown or repeated problem field %q", customID)
		}
		switch customID {
		case problemNameFieldID:
			result.Name = *value
		case problemURLFieldID:
			result.URL = *value
		case problemDifficultyFieldID:
			result.Difficulty = value
		case problemPriorityFieldID:
			result.Priority = value
		case problemTopicFieldID:
			result.Topic = value
		}
		delete(pendingFields, customID)
	}
	if len(pendingFields) != 0 {
		return ProblemModalFields{}, fmt.Errorf("incomplete problem form")
	}

	return result, nil
}

func parseProblemModalField(component discordgo.MessageComponent) (string, *string, error) {
	switch field := component.(type) {
	case *discordgo.TextInput:
		if field == nil {
			return "", nil, fmt.Errorf("expected a problem form component")
		}
		if field.CustomID == problemDifficultyFieldID || field.CustomID == problemPriorityFieldID {
			return "", nil, fmt.Errorf("expected a select for problem field %q", field.CustomID)
		}
		if field.CustomID == problemTopicFieldID && strings.TrimSpace(field.Value) == "" {
			return field.CustomID, nil, nil
		}
		return field.CustomID, &field.Value, nil
	case *discordgo.SelectMenu:
		if field == nil {
			return "", nil, fmt.Errorf("expected a problem form component")
		}
		if field.Type() != discordgo.SelectMenuComponent {
			return "", nil, fmt.Errorf("expected a string select for problem field %q", field.CustomID)
		}
		if len(field.Values) > 1 {
			return "", nil, fmt.Errorf("expected one value for problem select %q", field.CustomID)
		}

		var options []modalStringSelectOption
		switch field.CustomID {
		case problemDifficultyFieldID:
			options = problemDifficultyOptions()
		case problemPriorityFieldID:
			options = problemPriorityOptions()
		default:
			return "", nil, fmt.Errorf("unexpected select field %q", field.CustomID)
		}
		if len(field.Values) == 0 || field.Values[0] == unassignedMetadataSelectValue {
			return field.CustomID, nil, nil
		}
		for _, option := range options {
			if field.Values[0] == option.Value {
				return field.CustomID, &option.Value, nil
			}
		}
		return field.CustomID, nil, nil
	default:
		return "", nil, fmt.Errorf("expected a problem form text input or select")
	}
}

func BuildProblemBatchModalResponse() *discordgo.InteractionResponse {
	return buildJSONModalResponse(ProblemBatchModalID(), "Problem batch", `[{"class":"basicos","name":"Two Sum","url":"https://example.com/problem"}]`)
}

func ParseCreateProblemModal(data discordgo.ModalSubmitInteractionData) (CreateProblemModal, error) {
	classCode, err := ParseCreateProblemModalID(data.CustomID)
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
	classCode, problemID, err := ParseEditProblemModalID(data.CustomID)
	if err != nil {
		return EditProblemModal{}, err
	}
	fields, err := parseProblemModalFields(data)
	if err != nil {
		return EditProblemModal{}, err
	}
	return EditProblemModal{
		ClassCode:          classCode,
		ProblemID:          problemID,
		ProblemModalFields: fields,
	}, nil
}

func ParseProblemBatchModalSource(data discordgo.ModalSubmitInteractionData) (JSONModalSource, error) {
	if err := ValidateProblemBatchModalID(data.CustomID); err != nil {
		return JSONModalSource{}, err
	}
	return parseJSONModalSource(data)
}
