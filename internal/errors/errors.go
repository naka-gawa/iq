// Package errors defines sentinel errors and rich error types for iq.
//
// Sentinel errors are used for exit code mappiong via [errors.Is].
// Rich error types carry contextual fields and implement [errors.Unwrap].
// so that both errors.Is and errors.As work correctly.
//
// Exit code mapping:
//
//	ErrKeyNotFound     -> exit 2
//	ErrPathInvalid     -> exit 1
//	ErrFileParseFailed -> exit 1
//	ErrDialectDetect   -> exit 1
//
// Callers should import this package with an alias to avoid collision
// with the standard library errors package:
//
//	import iqerr "iq/internal/errors"
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for exit code mapping.
var (
	// ErrKeyNotFound is the only error that maps to exit code 2.
	// See docs/spec/PRD.md for the full exit code table.
	ErrKeyNotFound     = errors.New("key not found")
	ErrPathInvalid     = errors.New("invalid path expression")
	ErrFileParseFailed = errors.New("failed to parse INI file")
	ErrDialectDetect   = errors.New("failed to detect dialect")
)

// KeyNotFoundError unwraps to [ErrKeyNotFound].
type KeyNotFoundError struct {
	Section string
	Key     string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("[%s] key %q not found", e.Section, e.Key)
}

func (e *KeyNotFoundError) Unwrap() error { return ErrKeyNotFound }

// ParseError unwraps to [ErrFileParseFailed].
type ParseError struct {
	Path string
	Line int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s (line %d)", e.Path, e.Line)
}

func (e *ParseError) Unwrap() error { return ErrFileParseFailed }

// PathInvalidError unwraps to [ErrPathInvalid].
type PathInvalidError struct {
	Expr string
}

func (e *PathInvalidError) Error() string {
	return fmt.Sprintf("invalid path expression: %q", e.Expr)
}

func (e *PathInvalidError) Unwrap() error { return ErrPathInvalid }
