package adapters

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/stretchr/testify/assert"
)

func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		action     Action
		args       []any
		want       string
		assertFunc func(t *testing.T, got string)
	}{
		{
			name:   "validation reasons are shown without technical error metadata",
			err:    serviceerr.NewValidationError("invalid problem values", errors.Join(errors.New("problem name is required"), errors.New("class code is required"))),
			action: ActionMentorAddProblem,
			want:   "Please correct the values: problem name is required\nclass code is required",
		},
		{
			name:   "wrapped validation errors still show the correction reason",
			err:    fmt.Errorf("create failed: %w", serviceerr.NewValidationError("invalid problem values", errors.New("URL must be an absolute HTTP or HTTPS URL"))),
			action: ActionMentorAddProblem,
			want:   "Please correct the values: URL must be an absolute HTTP or HTTPS URL",
		},
		{
			name:   "validation without a cause uses its summary",
			err:    serviceerr.NewValidationError("problem name is required", nil),
			action: ActionMentorAddProblem,
			want:   "Please correct the values: problem name is required",
		},
		{
			name:   "nil error returns fallback",
			err:    nil,
			action: ActionRegisterInClass,
			args:   []any{"basicos_1", "jorge"},
			want:   "Unexpected error.",
		},
		{
			name:   "internal error is always hidden",
			err:    serviceerr.NewInternalError("db crashed with details", nil),
			action: ActionScoreByClass,
			args:   []any{"basicos_1"},
			want:   "Unexpected internal error bro.",
		},
		{
			name:   "service message is ignored, template is used instead",
			err:    serviceerr.NewConflictError("from service", nil),
			action: ActionMentorAddProblem,
			args:   []any{"basicos_1", "https://leetcode.com/problems/two-sum", "Two Sum"},
			want:   "Problem `Two Sum` already exists on class `basicos_1` with URL `https://leetcode.com/problems/two-sum`",
		},
		{
			name:   "template supports multiple args",
			err:    serviceerr.NewConflictError("", nil),
			action: ActionMentorAddProblem,
			args:   []any{"basicos_1", "https://leetcode.com/problems/two-sum", "Two Sum"},
			want:   "Problem `Two Sum` already exists on class `basicos_1` with URL `https://leetcode.com/problems/two-sum`",
		},
		{
			name:   "semantic class code not found for register action",
			err:    serviceerr.NewClassCodeNotFoundError("", nil),
			action: ActionRegisterInClass,
			args:   []any{"basicos_404", "jorge"},
			want:   "Class code `basicos_404` not found for user `jorge`.",
		},
		{
			name:   "semantic problem not found for enrolled propose AC",
			err:    serviceerr.NewProblemNotFoundError("", nil),
			action: ActionEnrolledProposeAC,
			args:   []any{"<@123>", int64(404)},
			want:   "Problem `404` not found for user `<@123>`",
		},
		{
			name:   "semantic duplicate for mentor add problem",
			err:    serviceerr.NewDuplicateProblemError("", nil),
			action: ActionMentorAddProblem,
			args:   []any{"basicos_1", "https://leetcode.com/problems/two-sum", "Two Sum"},
			want:   "Problem `Two Sum` already exists on class `basicos_1` (with URL `https://leetcode.com/problems/two-sum`)",
		},
		{
			name:   "semantic class not found for mentor register action",
			err:    serviceerr.NewClassCodeNotFoundError("", nil),
			action: ActionMentorRegisterUser,
			args:   []any{"<@123>", "basicos_404"},
			want:   "Class code `basicos_404` not found for user `<@123>`",
		},
		{
			name:   "semantic user not found for mentor register",
			err:    serviceerr.NewUserNotFoundError("", nil),
			action: ActionMentorRegisterUser,
			args:   []any{"<@123>", "basicos_1"},
			want:   "User `<@123>` not found",
		},
		{
			name:   "semantic class code not found for score by user and class",
			err:    serviceerr.NewClassCodeNotFoundError("", nil),
			action: ActionScoreByUserAndClass,
			args:   []any{"<@123>", "basicos_404"},
			want:   "Class code `basicos_404` not found",
		},
		{
			name:   "missing context for score by class",
			err:    serviceerr.NewNotFoundInfoError("", nil),
			action: ActionScoreByClass,
			args:   []any{nil},
			want:   "I couldn't identify the class, bro. Try specifying the class code (e.g., `!score basicos_1`) or run this command inside a class channel.",
		},
		{
			name:   "semantic user not active for score by user and class",
			err:    serviceerr.NewUserNotEnrolledError("", nil),
			action: ActionScoreByUserAndClass,
			args:   []any{"Jorge", "basicos_1"},
			want:   "Bro, `Jorge` doesn't have any solved problems in class `basicos_1` yet.",
		},
		{
			name:   "missing context for score by user and class",
			err:    serviceerr.NewNotFoundInfoError("", nil),
			action: ActionScoreByUserAndClass,
			args:   []any{"<@123>", nil},
			want:   "I couldn't identify the class, bro. Try specifying the class code (e.g., `!score_of @user basicos_1`) or run this command inside a class channel.",
		},
		{
			name:   "semantic problem not found for mentor undo AC",
			err:    serviceerr.NewProblemNotFoundError("", nil),
			action: ActionMentorUndoAC,
			args:   []any{"<@123>", int64(77)},
			want:   "Problem id `77` not found for user `<@123>`",
		},
		{
			name:   "semantic conflict for mentor undo AC",
			err:    serviceerr.NewConflictError("any", nil),
			action: ActionMentorUndoAC,
			args:   []any{"<@123>", int64(77)},
			want:   "User `<@123>` has not solved problem `77` yet",
		},
		{
			name:   "wrapped class not found is mapped correctly",
			err:    fmt.Errorf("outer: %w", serviceerr.NewClassCodeNotFoundError("", nil)),
			action: ActionListProblemsByClass,
			args:   []any{"basicos_1"},
			want:   "Class code `basicos_1` not found",
		},
		{
			name:   "missing template args falls back to action default",
			err:    serviceerr.NewConflictError("", nil),
			action: ActionMentorAddProblem,
			args:   []any{"basicos_1"},
			want:   "Unexpected error while adding problem.",
		},
		{
			name:   "unexpected non service error uses action default",
			err:    fmt.Errorf("plain error"),
			action: ActionExterminateUser,
			args:   []any{"123"},
			want:   "Unexpected error while exterminating user.",
		},
		{
			name:   "missing context for clear class",
			err:    serviceerr.NewNotFoundInfoError("", nil),
			action: ActionClearClass,
			args:   []any{nil},
			want:   "I couldn't identify the class, bro. Try specifying the class code (e.g., `!clear_class basicos_1`) or run this command inside a class channel.",
		},
		{
			name:   "unexpected non service error uses exterminate class default",
			err:    fmt.Errorf("plain error"),
			action: ActionExterminateClass,
			args:   []any{"basicos_1"},
			want:   "Unexpected error while exterminating class.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MapError(tt.err, tt.action, tt.args...)

			assert.Equal(t, tt.want, got)
			if tt.assertFunc != nil {
				tt.assertFunc(t, got)
			}
		})
	}
}

func TestTemplates_AllActionsHaveDefault(t *testing.T) {
	t.Parallel()

	for action := ActionRegisterInClass; action < ActionSentinel; action++ {
		action := action
		t.Run(fmt.Sprintf("action_%d_has_default", action), func(t *testing.T) {
			t.Parallel()

			byType, ok := templates[action]
			if !assert.True(t, ok, "missing template map for action %d", action) {
				return
			}

			msg, ok := byType[errorKindDefault]
			if !assert.True(t, ok, "missing default template for action %d", action) {
				return
			}

			assert.NotEmpty(t, strings.TrimSpace(msg), "default template for action %d is empty", action)
		})
	}
}
