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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := dialect.Detect(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
