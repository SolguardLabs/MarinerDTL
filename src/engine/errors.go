package engine

import "fmt"

type ErrorKind string

const (
	ErrInvalid  ErrorKind = "invalid"
	ErrNotFound ErrorKind = "not_found"
	ErrState    ErrorKind = "state"
	ErrPolicy   ErrorKind = "policy"
	ErrSolvency ErrorKind = "solvency"
)

type ServiceError struct {
	Kind    ErrorKind
	Message string
}

func (e ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func invalid(format string, args ...any) error {
	return ServiceError{Kind: ErrInvalid, Message: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...any) error {
	return ServiceError{Kind: ErrNotFound, Message: fmt.Sprintf(format, args...)}
}

func stateError(format string, args ...any) error {
	return ServiceError{Kind: ErrState, Message: fmt.Sprintf(format, args...)}
}

func policyError(format string, args ...any) error {
	return ServiceError{Kind: ErrPolicy, Message: fmt.Sprintf(format, args...)}
}

func solvencyError(format string, args ...any) error {
	return ServiceError{Kind: ErrSolvency, Message: fmt.Sprintf(format, args...)}
}
