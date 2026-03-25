package adapters

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
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
			name:   "nil error returns fallback",
			err:    nil,
			action: ActionBindUser,
			args:   []any{"jorge"},
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
			err:    serviceerr.NewConflictError("username already taken from service", nil),
			action: ActionBindUser,
			args:   []any{"jorge"},
			want:   "Username `jorge` is already taken by another user. Please choose a different username.",
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
			name:   "semantic user not binded for register action",
			err:    serviceerr.NewUserNotBindedError("", nil),
			action: ActionRegisterInClass,
			args:   []any{"basicos_1", "jorge"},
			want:   "User `jorge` is not binded yet. Use `!bind <username>` first.",
		},
		{
			name:   "semantic problem not found for enrolled propose AC",
			err:    serviceerr.NewProblemNotFoundError("", nil),
			action: ActionEnrolledProposeAC,
			args:   []any{"jorge", int64(404)},
			want:   "Problem `404` not found for user `jorge`",
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
			args:   []any{"jorge", "basicos_404"},
			want:   "Class code `basicos_404` not found for username `jorge`",
		},
		{
			name:   "semantic user not found for mentor register action",
			err:    serviceerr.NewUserNotFoundError("", nil),
			action: ActionMentorRegisterUser,
			args:   []any{"jorge_not_found"},
			want:   "Username `jorge_not_found` not found",
		},
		{
			name:   "semantic problem not found for mentor undo AC",
			err:    serviceerr.NewProblemNotFoundError("", nil),
			action: ActionMentorUndoAC,
			args:   []any{"jorge", int64(77)},
			want:   "Problem id `77` not found for username `jorge`",
		},
		{
			name:   "missing template args falls back to action default",
			err:    serviceerr.NewConflictError("", nil),
			action: ActionMentorAddProblem,
			args:   []any{"basicos_1"}, // Not enough args for fmt.Sprintf
			want:   "Unexpected error while adding problem.",
		},
		{
			name:   "unmapped error kind falls back to action default",
			err:    serviceerr.NewInternalError("", nil),
			action: ActionRegisterInClass,
			args:   []any{"basicos_1", "jorge"},
			want:   "Unexpected internal error bro.",
		},
		{
			name:   "wrapped semantic error is resolved via error chain",
			err:    fmt.Errorf("outer wrap: %w", serviceerr.NewClassCodeNotFoundError("", fmt.Errorf("inner"))),
			action: ActionListProblemsByClass,
			args:   []any{"basicos_1"},
			want:   "Class code `basicos_1` not found",
		},
		{
			name:   "unexpected non service error uses action default",
			err:    fmt.Errorf("plain error"),
			action: ActionExterminateUser,
			args:   []any{"123"},
			want:   "Unexpected error while exterminating user.",
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

	for action := ActionBindUser; action < ActionSentinel; action++ {
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
