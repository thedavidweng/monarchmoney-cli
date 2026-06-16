package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected string
	}{
		{"AuthRequired", AuthRequired, "AUTH_REQUIRED"},
		{"AuthSessionExpired", AuthSessionExpired, "AUTH_SESSION_EXPIRED"},
		{"AuthMFARequired", AuthMFARequired, "AUTH_MFA_REQUIRED"},
		{"AuthMFAInvalid", AuthMFAInvalid, "AUTH_MFA_INVALID"},
		{"NetworkUnreachable", NetworkUnreachable, "NETWORK_UNREACHABLE"},
		{"NetworkTimeout", NetworkTimeout, "NETWORK_TIMEOUT"},
		{"APIError", APIError, "API_ERROR"},
		{"APISchemaChanged", APISchemaChanged, "API_SCHEMA_CHANGED"},
		{"FEATURE_UNAVAILABLE", FEATURE_UNAVAILABLE, "FEATURE_UNAVAILABLE"},
		{"ValidationFailed", ValidationFailed, "VALIDATION_FAILED"},
		{"ReadOnlyViolation", ReadOnlyViolation, "READ_ONLY_VIOLATION"},
		{"ConfirmationRequired", ConfirmationRequired, "CONFIRMATION_REQUIRED"},
		{"ResourceNotFound", ResourceNotFound, "RESOURCE_NOT_FOUND"},
		{"InternalError", InternalError, "INTERNAL_ERROR"},
		{"InvalidArguments", InvalidArguments, "INVALID_ARGUMENTS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.code))
		})
	}
}

func TestCategoryConstants(t *testing.T) {
	tests := []struct {
		name     string
		cat      Category
		expected string
	}{
		{"CatAuth", CatAuth, "auth"},
		{"CatNetwork", CatNetwork, "network"},
		{"CatAPI", CatAPI, "api"},
		{"CatValidation", CatValidation, "validation"},
		{"CatSafety", CatSafety, "safety"},
		{"CatInternal", CatInternal, "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.cat))
		})
	}
}
