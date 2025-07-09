package fault

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("resource not found")
	ErrUniqueViolation     = errors.New("unique violation")
	ErrForeignKeyViolation = errors.New("restricted for deletion")
)

type ErrorType int

const (
	ErrClient ErrorType = iota
	ErrInternal
)

type Fault struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *Fault) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.typeString(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.typeString(), e.Message)
}

// Unwrap allows errors.Is and errors.As to work.
func (e *Fault) Unwrap() error {
	return e.Err
}

// typeString returns a human-readable representation of the error type.
func (e *Fault) typeString() string {
	switch e.Type {
	case ErrClient:
		return "ClientError"
	case ErrInternal:
		return "InternalError"
	default:
		return "UnknownError"
	}
}

// NewClientError creates a new client error.
func NewClientError(msg string, err error) error {
	return &Fault{
		Type:    ErrClient,
		Message: msg,
		Err:     err,
	}
}

// NewInternalError creates a new internal server error.
func NewInternalError(msg string, err error) error {
	return &Fault{
		Type:    ErrInternal,
		Message: msg,
		Err:     err,
	}
}

// IsClientError checks if an error is a client error.
func IsClientError(err error) bool {
	var ce *Fault
	if errors.As(err, &ce) {
		return ce.Type == ErrClient
	}
	return false
}

// IsInternalError checks if an error is an internal error.
func IsInternalError(err error) bool {
	var ce *Fault
	if errors.As(err, &ce) {
		return ce.Type == ErrInternal
	}
	return false
}
