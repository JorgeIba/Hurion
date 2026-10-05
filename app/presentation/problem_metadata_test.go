package presentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code *string
		want string
	}{
		{name: "essential", code: new("essential"), want: "Essential"},
		{name: "recommended", code: new("recommended"), want: "Recommended"},
		{name: "supplementary", code: new("supplementary"), want: "Supplementary"},
		{name: "not required", code: new("not_required"), want: "Not required"},
		{name: "discouraged", code: new("discouraged"), want: "Discouraged"},
		{name: "unassigned", want: "No priority assigned"},
		{name: "unknown code stays visible", code: new("future_code"), want: "future_code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PriorityLabel(tt.code))
		})
	}
	code := "recommended"
	PriorityLabel(&code)
	assert.Equal(t, "recommended", code, "formatting must not overwrite the canonical code")
}
