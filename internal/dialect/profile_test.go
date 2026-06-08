package dialect_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"iq/internal/dialect"
	iqerr "iq/internal/errors"
)

func TestProfile_LoadOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile dialect.Profile
		want    ini.LoadOptions
	}{
		{
			name:    "ProfileGeneric returns expected LoadOptions",
			profile: dialect.ProfileGeneric,
			want: ini.LoadOptions{
				AllowBooleanKeys:            true,
				AllowShadows:                false,
				IgnoreInlineComment:         false,
				UnescapeValueCommentSymbols: true,
			},
		},
		{
			name:    "ProfileSystemd has AllowShadows and IgnoreInlineComment",
			profile: dialect.ProfileSystemd,
			want: ini.LoadOptions{
				AllowBooleanKeys:    true,
				AllowShadows:        true,
				IgnoreInlineComment: true,
			},
		},
		{
			name:    "ProfileGitconfig has Insensitive",
			profile: dialect.ProfileGitconfig,
			want: ini.LoadOptions{
				Insensitive:      true,
				AllowBooleanKeys: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.profile.LoadOptions()
			assert.Equal(t, tt.want.AllowBooleanKeys, got.AllowBooleanKeys)
			assert.Equal(t, tt.want.AllowShadows, got.AllowShadows)
			assert.Equal(t, tt.want.IgnoreInlineComment, got.IgnoreInlineComment)
			assert.Equal(t, tt.want.UnescapeValueCommentSymbols, got.UnescapeValueCommentSymbols)
			assert.Equal(t, tt.want.Insensitive, got.Insensitive)
		})
	}
}

func TestParseProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    dialect.Profile
		wantErr bool
	}{
		{"generic lowercase", "generic", dialect.ProfileGeneric, false},
		{"systemd lowercase", "systemd", dialect.ProfileSystemd, false},
		{"gitconfig lowercase", "gitconfig", dialect.ProfileGitconfig, false},
		{"GENERIC uppercase", "GENERIC", dialect.ProfileGeneric, false},
		{"Systemd mixed case", "Systemd", dialect.ProfileSystemd, false},
		{"GitConfig mixed case", "GitConfig", dialect.ProfileGitconfig, false},
		{"unknown returns error", "bogus", dialect.ProfileGeneric, true},
		{"empty string returns error", "", dialect.ProfileGeneric, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := dialect.ParseProfile(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, iqerr.ErrDialectDetect))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
