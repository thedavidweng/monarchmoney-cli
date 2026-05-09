package errors

import (
	"errors"
	"testing"
)

func TestErrorFormatting(t *testing.T) {
	err := New(APIError, "boom", CatAPI, true, errors.New("root cause"))
	if got, want := err.Error(), "[API_ERROR] boom: root cause"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	noCause := New(InternalError, "plain", CatInternal, false, nil)
	if got, want := noCause.Error(), "[INTERNAL_ERROR] plain"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNewPopulatesFields(t *testing.T) {
	cause := errors.New("cause")
	err := New(ValidationFailed, "invalid", CatValidation, false, cause)
	if err.Code != ValidationFailed || err.Message != "invalid" || err.Category != CatValidation || err.Retryable || err.Err != cause {
		t.Fatalf("New() returned %#v", err)
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		code Code
		want int
	}{
		{"auth required", AuthRequired, 3},
		{"session expired", AuthSessionExpired, 3},
		{"mfa required", AuthMFARequired, 3},
		{"mfa invalid", AuthMFAInvalid, 3},
		{"read only", ReadOnlyViolation, 4},
		{"network unreachable", NetworkUnreachable, 5},
		{"network timeout", NetworkTimeout, 5},
		{"api error", APIError, 6},
		{"schema changed", APISchemaChanged, 6},
		{"feature unavailable", FEATURE_UNAVAILABLE, 6},
		{"validation failed", ValidationFailed, 7},
		{"confirm required", ConfirmationRequired, 10},
		{"invalid args", InvalidArguments, 2},
		{"default", Code("SOMETHING_ELSE"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "msg", CatInternal, false, nil)
			if got := err.ExitCode(); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
