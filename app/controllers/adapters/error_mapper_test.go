package adapters

import (
	"fmt"
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
			args:   []any{"Two Sum", "basicos_1", int64(17)},
			want:   "Problem `Two Sum` already exists on class `basicos_1` with the ID: `17`",
		},
		{
			name:   "not found template supports argument reordering",
			err:    serviceerr.NewNotFoundInfoError("", nil),
			action: ActionScoreByUserAndClass,
			args:   []any{"jorge", "basicos_1"},
			want:   "Not found info for class code `basicos_1` or user name `jorge`",
		},
		{
			name:   "wrapped not found is resolved via error chain",
			err:    fmt.Errorf("outer wrap: %w", serviceerr.NewNotFoundInfoError("", fmt.Errorf("inner"))),
			action: ActionGetClassesOfUsername,
			args:   []any{"jorge"},
			want:   "Username `jorge` not found",
			assertFunc: func(t *testing.T, got string) {
				assert.Contains(t, got, "jorge")
			},
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
