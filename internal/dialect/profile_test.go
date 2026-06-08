package dialect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"

	"iq/internal/dialect"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.profile.LoadOptions()
			assert.Equal(t, tt.want.AllowBooleanKeys, got.AllowBooleanKeys)
			assert.Equal(t, tt.want.AllowShadows, got.AllowShadows)
			assert.Equal(t, tt.want.IgnoreInlineComment, got.IgnoreInlineComment)
			assert.Equal(t, tt.want.UnescapeValueCommentSymbols, got.UnescapeValueCommentSymbols)
		})
	}
}
