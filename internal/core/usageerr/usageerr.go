// Package usageerr defines the error type for caller mistakes (bad
// arguments, unknown profile, unknown subcommand). The CLI adapter maps it
// to exit code 2 per R1 in docs/acceptance.md; every other error maps to 1.
package usageerr

import (
	"errors"
	"fmt"
)

// Error marks a failure as the caller's mistake rather than a runtime
// failure.
type Error struct {
	msg     string
	wrapped error
}

func New(msg string) *Error {
	return &Error{msg: msg}
}

func Newf(format string, args ...any) *Error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

// Wrap marks an existing error as a usage error without changing its
// message.
func Wrap(err error) *Error {
	return &Error{msg: err.Error(), wrapped: err}
}

func (e *Error) Error() string { return e.msg }

func (e *Error) Unwrap() error { return e.wrapped }

// IsUsage reports whether err is (or wraps) a usage error.
func IsUsage(err error) bool {
	var u *Error
	return errors.As(err, &u)
}
