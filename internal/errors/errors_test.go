package errors_test

import (
	stderrors "errors"
	"testing"

	iqerr "iq/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      *iqerr.KeyNotFoundError
		targetIs error
		wantMsg  string
	}{
		{
			name:     "standard",
			err:      &iqerr.KeyNotFoundError{Section: "database", Key: "host"},
			targetIs: iqerr.ErrKeyNotFound,
			wantMsg:  `[database] key "host" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, stderrors.Is(tt.err, tt.targetIs))
			assert.Equal(t, tt.wantMsg, tt.err.Error())

			var target *iqerr.KeyNotFoundError
			require.True(t, stderrors.As(tt.err, &target))
			assert.Equal(t, tt.err.Section, target.Section)
			assert.Equal(t, tt.err.Key, target.Key)
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      *iqerr.ParseError
		targetIs error
		wantMsg  string
	}{
		{
			name:     "with line",
			err:      &iqerr.ParseError{Path: "config.ini", Line: 5},
			targetIs: iqerr.ErrFileParseFailed,
			wantMsg:  "failed to parse config.ini (line 5)",
		},
		{
			name:     "no line",
			err:      &iqerr.ParseError{Path: "config.ini", Line: 0},
			targetIs: iqerr.ErrFileParseFailed,
			wantMsg:  "failed to parse config.ini (line 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, stderrors.Is(tt.err, tt.targetIs))
			assert.Equal(t, tt.wantMsg, tt.err.Error())

			var target *iqerr.ParseError
			require.True(t, stderrors.As(tt.err, &target))
			assert.Equal(t, tt.err.Path, target.Path)
			assert.Equal(t, tt.err.Line, target.Line)
		})
	}
}

func TestPathInvalidError(t *testing.T) {
	tests := []struct {
		name     string
		err      *iqerr.PathInvalidError
		targetIs error
		wantMsg  string
	}{
		{
			name:     "invalid path",
			err:      &iqerr.PathInvalidError{Expr: ".foo..bar"},
			targetIs: iqerr.ErrPathInvalid,
			wantMsg:  `invalid path expression: ".foo..bar"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, stderrors.Is(tt.err, tt.targetIs))
			assert.Equal(t, tt.wantMsg, tt.err.Error())

			var target *iqerr.PathInvalidError
			require.True(t, stderrors.As(tt.err, &target))
			assert.Equal(t, tt.err.Expr, target.Expr)
		})
	}
}
