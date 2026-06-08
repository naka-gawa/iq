package dialect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"iq/internal/dialect"
)

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want dialect.Profile
	}{
		{
			name: "empty string returns ProfileGeneric",
			path: "",
			want: dialect.ProfileGeneric,
		},
		{
			name: "ini extension returns ProfileGeneric",
			path: "config.ini",
			want: dialect.ProfileGeneric,
		},
		{
			name: "cfg extension returns ProfileGeneric",
			path: "settings.cfg",
			want: dialect.ProfileGeneric,
		},
		{
			name: "conf extension returns ProfileGeneric",
			path: "app.conf",
			want: dialect.ProfileGeneric,
		},
		{
			name: "long absolute path returns ProfileGeneric",
			path: "/very/long/path/to/some/deeply/nested/directory/config.ini",
			want: dialect.ProfileGeneric,
		},
		{
			name: "no extension returns ProfileGeneric",
			path: "noextension",
			want: dialect.ProfileGeneric,
		},
		{
			name: "unknown extension returns ProfileGeneric",
			path: "file.toml",
			want: dialect.ProfileGeneric,
		},
		// Systemd extensions
		{
			name: ".service returns ProfileSystemd",
			path: "nginx.service",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".target returns ProfileSystemd",
			path: "multi-user.target",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".socket returns ProfileSystemd",
			path: "docker.socket",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".mount returns ProfileSystemd",
			path: "home.mount",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".timer returns ProfileSystemd",
			path: "backup.timer",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".path returns ProfileSystemd",
			path: "watch.path",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".scope returns ProfileSystemd",
			path: "session.scope",
			want: dialect.ProfileSystemd,
		},
		{
			name: ".slice returns ProfileSystemd",
			path: "system.slice",
			want: dialect.ProfileSystemd,
		},
		{
			name: "absolute path .service returns ProfileSystemd",
			path: "/etc/systemd/system/myapp.service",
			want: dialect.ProfileSystemd,
		},
		// Gitconfig basenames
		{
			name: ".gitconfig returns ProfileGitconfig",
			path: ".gitconfig",
			want: dialect.ProfileGitconfig,
		},
		{
			name: "absolute .gitconfig path returns ProfileGitconfig",
			path: "/home/user/.gitconfig",
			want: dialect.ProfileGitconfig,
		},
		{
			name: "config inside .git dir returns ProfileGitconfig",
			path: "/repo/.git/config",
			want: dialect.ProfileGitconfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := dialect.Detect(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
