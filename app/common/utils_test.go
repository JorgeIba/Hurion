package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractUserIDFromMention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mention    string
		assertFunc func(*testing.T, string, bool)
	}{
		{
			name:    "valid mention",
			mention: "<@1234567890>",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, "1234567890", id)
			},
		},
		{
			name:    "valid mention with !",
			mention: "<@!1234567890>",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, "1234567890", id)
			},
		},
		{
			name:    "invalid prefix",
			mention: "1234567890>",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.False(t, ok)
				assert.Empty(t, id)
			},
		},
		{
			name:    "invalid suffix",
			mention: "<@1234567890",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.False(t, ok)
				assert.Empty(t, id)
			},
		},
		{
			name:    "non-numeric ID",
			mention: "<@abc1234>",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.False(t, ok)
				assert.Empty(t, id)
			},
		},
		{
			name:    "empty ID",
			mention: "<@>",
			assertFunc: func(t *testing.T, id string, ok bool) {
				assert.False(t, ok)
				assert.Empty(t, id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := ExtractUserIDFromMention(tt.mention)
			tt.assertFunc(t, id, ok)
		})
	}
}

func TestIsUserMention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mention  string
		expected bool
	}{
		{
			name:     "valid standard mention",
			mention:  "<@1234567890>",
			expected: true,
		},
		{
			name:     "valid nickname mention",
			mention:  "<@!1234567890>",
			expected: true,
		},
		{
			name:     "role mention (should be false)",
			mention:  "<@&1234567890>",
			expected: false,
		},
		{
			name:     "channel mention",
			mention:  "<#1234567890>",
			expected: false,
		},
		{
			name:     "plain text",
			mention:  "1234567890",
			expected: false,
		},
		{
			name:     "empty brackets",
			mention:  "<@>",
			expected: true, // The regex/prefix check currently allows this, which is fine since it's just a shape check. ExtractUserID handles actual validation.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsUserMention(tt.mention)
			assert.Equal(t, tt.expected, result)
		})
	}
}
