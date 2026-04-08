// Package apperror defines stable API error codes and helpers.
package apperror

import (
	"errors"
	"fmt"
)

// Kind classifies errors for HTTP mapping.
type Kind string

const (
	KindInvalid      Kind = "invalid"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindInternal     Kind = "internal"
)

// Error is an application error with a stable code for clients.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// New builds an Error.
func New(kind Kind, code, msg string) *Error {
	return &Error{Kind: kind, Code: code, Message: msg}
}

// Wrap adds context and optional underlying error.
func Wrap(kind Kind, code, msg string, err error) *Error {
	return &Error{Kind: kind, Code: code, Message: msg, Err: err}
}

// AsError unwraps to *Error.
func AsError(err error) (*Error, bool) {
	var ae *Error
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
