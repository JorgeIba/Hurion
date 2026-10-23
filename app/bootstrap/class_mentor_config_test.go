package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassMentorConfiguration(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, setting string
		overridden    bool
		roles         []string
		invalid       bool
	}{
		{name: "absent inherits"},
		{name: "explicit empty replaces", setting: "mentor_role_ids: []", overridden: true, roles: []string{}},
		{name: "one custom role", setting: `mentor_role_ids: ["4444444444"]`, overridden: true, roles: []string{"4444444444"}},
		{name: "multiple custom roles", setting: `mentor_role_ids: ["4444444444", "5555555555"]`, overridden: true, roles: []string{"4444444444", "5555555555"}},
		{name: "empty role fails", setting: `mentor_role_ids: [" "]`, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte("admin_role_id: admin\nmentor_role_id: global\nclasses:\n  - code: advanced\n    name: Advanced\n    "+tt.setting+"\n"), 0600))
			cfg, err := LoadConfig(path)
			if tt.invalid {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			roles, present := cfg.MentorRolesByClass()["advanced"]
			assert.Equal(t, tt.overridden, present)
			assert.Equal(t, tt.roles, roles)
		})
	}
}
