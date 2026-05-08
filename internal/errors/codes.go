package errors

// Code represents a machine-readable error code.
type Code string

const (
	AuthRequired                Code = "AUTH_REQUIRED"
	AuthSessionExpired          Code = "AUTH_SESSION_EXPIRED"
	AuthMFARequired             Code = "AUTH_MFA_REQUIRED"
	AuthMFAInvalid              Code = "AUTH_MFA_INVALID"
	NetworkUnreachable          Code = "NETWORK_UNREACHABLE"
	NetworkTimeout              Code = "NETWORK_TIMEOUT"
	APIError                    Code = "API_ERROR"
	APISchemaChanged            Code = "API_SCHEMA_CHANGED"
	ValidationFailed            Code = "VALIDATION_FAILED"
	ReadOnlyViolation           Code = "READ_ONLY_VIOLATION"
	ConfirmationRequired        Code = "CONFIRMATION_REQUIRED"
	DryRunRequired              Code = "DRY_RUN_REQUIRED"
	ResourceNotFound            Code = "RESOURCE_NOT_FOUND"
	PartialSuccess              Code = "PARTIAL_SUCCESS"
	ConfigInvalid               Code = "CONFIG_INVALID"
	SessionPermissionInsecure   Code = "SESSION_PERMISSION_INSECURE"
	InternalError               Code = "INTERNAL_ERROR"
	InvalidArguments            Code = "INVALID_ARGUMENTS"
)

// Category groups errors for higher-level handling.
type Category string

const (
	CatAuth       Category = "auth"
	CatNetwork    Category = "network"
	CatAPI        Category = "api"
	CatValidation Category = "validation"
	CatSafety     Category = "safety"
	CatConfig     Category = "config"
	CatInternal   Category = "internal"
)
