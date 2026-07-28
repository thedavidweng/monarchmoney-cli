package errors

import (
	"fmt"
	"time"
)

type Error struct {
	Code         Code     `json:"code"`
	Message      string   `json:"message"`
	Category     Category `json:"category"`
	Retryable    bool     `json:"retryable"`
	RetryAfterMS int64    `json:"retry_after_ms,omitempty"`
	Err          error    `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code Code, message string, category Category, retryable bool, err error) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Category:  category,
		Retryable: retryable,
		Err:       err,
	}
}

func NewWithRetryAfter(code Code, message string, category Category, retryable bool, retryAfter time.Duration, err error) *Error {
	return &Error{
		Code:         code,
		Message:      message,
		Category:     category,
		Retryable:    retryable,
		RetryAfterMS: retryAfter.Milliseconds(),
		Err:          err,
	}
}

func (e *Error) ExitCode() int {
	switch e.Code {
	case AuthRequired, AuthSessionExpired, AuthMFARequired, AuthMFAInvalid:
		return 3
	case ReadOnlyViolation:
		return 4
	case RateLimited:
		return 5
	case NetworkUnreachable, NetworkTimeout:
		return 5
	case APIError, APISchemaChanged, FEATURE_UNAVAILABLE:
		return 6
	case ValidationFailed:
		return 7
	case ConfirmationRequired:
		return 10
	case InvalidArguments:
		return 2
	default:
		return 1
	}
}
