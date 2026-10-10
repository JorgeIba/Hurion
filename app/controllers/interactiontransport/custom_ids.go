// Package interactiontransport owns Discord interaction forms, their IDs, and input decoding.
package interactiontransport

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
)

const (
	problemModalSeparator = "|"
	batchActionSeparator  = "|"
	maxCustomIDLength     = 100
)

// ModalAction recognizes the action even if context is malformed, allowing its
// controller to report the input error. Context validation happens when parsing.
func ModalAction(customID string) discordcommands.Action {
	route, _, _ := strings.Cut(customID, problemModalSeparator)

	switch discordcommands.Action(route) {
	case discordcommands.CreateProblem, discordcommands.EditProblem, discordcommands.ProblemBatch, discordcommands.ACBatch:
		return discordcommands.Action(route)
	default:
		return ""
	}
}

func ComponentAction(customID string) discordcommands.Action {
	route, _, _ := strings.Cut(customID, batchActionSeparator)

	switch discordcommands.Action(route) {
	case discordcommands.CancelProblemBatch, discordcommands.CancelACBatch:
		return discordcommands.Action(route)
	default:
		return ""
	}
}

func ProblemBatchModalID() string { return string(discordcommands.ProblemBatch) }
func ACBatchModalID() string      { return string(discordcommands.ACBatch) }
func ValidateProblemBatchModalID(customID string) error {
	return validateBatchModalID(customID, discordcommands.ProblemBatch)
}

func ValidateACBatchModalID(customID string) error {
	return validateBatchModalID(customID, discordcommands.ACBatch)
}
func validateBatchModalID(customID string, action discordcommands.Action) error {
	if customID != string(action) {
		return fmt.Errorf("invalid batch form ID")
	}

	return nil
}

func CreateProblemModalID(classCode string) (string, error) {
	if strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("missing problem modal class code")
	}

	customID := fmt.Sprintf("%s%s%s", string(discordcommands.CreateProblem), problemModalSeparator, url.PathEscape(classCode))

	if len(customID) > maxCustomIDLength {
		return "", fmt.Errorf("problem modal custom ID exceeds Discord's %d-character limit", maxCustomIDLength)
	}
	return customID, nil
}

func EditProblemModalID(classCode string, problemID int64) (string, error) {
	if strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("missing problem modal class code")
	}

	if problemID <= 0 {
		return "", fmt.Errorf("invalid problem modal context")
	}

	customID := fmt.Sprintf("%s%s%s%s%d", string(discordcommands.EditProblem), problemModalSeparator, url.PathEscape(classCode), problemModalSeparator, problemID)
	if len(customID) > maxCustomIDLength {
		return "", fmt.Errorf("problem modal custom ID exceeds Discord's %d-character limit", maxCustomIDLength)
	}
	return customID, nil
}

func ParseCreateProblemModalID(customID string) (string, error) {
	route, encodedClass, found := strings.Cut(customID, problemModalSeparator)
	if route != string(discordcommands.CreateProblem) || !found || len(customID) > maxCustomIDLength {
		return "", fmt.Errorf("invalid problem modal ID")
	}

	classCode, err := url.PathUnescape(encodedClass)
	if err != nil || strings.TrimSpace(classCode) == "" {
		return "", fmt.Errorf("invalid problem modal class code")
	}
	return classCode, nil
}

func ParseEditProblemModalID(customID string) (string, int64, error) {
	parts := strings.Split(customID, problemModalSeparator)
	if len(parts) != 3 || parts[0] != string(discordcommands.EditProblem) || len(customID) > maxCustomIDLength {
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

func ProblemBatchCancelID(batchID string) string {
	return string(discordcommands.CancelProblemBatch) + batchActionSeparator + batchID
}

func ACBatchCancelID(batchID string) string {
	return string(discordcommands.CancelACBatch) + batchActionSeparator + batchID
}

func ParseProblemBatchCancelID(customID string) (string, error) {
	return parseBatchCancelID(customID, discordcommands.CancelProblemBatch)
}

func ParseACBatchCancelID(customID string) (string, error) {
	return parseBatchCancelID(customID, discordcommands.CancelACBatch)
}

func parseBatchCancelID(customID string, action discordcommands.Action) (string, error) {
	route, batchID, found := strings.Cut(customID, batchActionSeparator)
	if !found || route != string(action) || batchID == "" || strings.TrimSpace(batchID) != batchID || strings.Contains(batchID, batchActionSeparator) {
		return "", fmt.Errorf("invalid batch cancellation button")
	}
	return batchID, nil
}
