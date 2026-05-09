package output

import (
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

// Envelope is the standard success response wrapper.
type Envelope struct {
	OK   bool        `json:"ok"`
	Data interface{} `json:"data,omitempty"`
	Meta Metadata    `json:"meta"`
}

// ErrorEnvelope is the standard error response wrapper.
type ErrorEnvelope struct {
	OK    bool          `json:"ok"`
	Error *errors.Error `json:"error"`
	Meta  Metadata      `json:"meta"`
}

// Metadata contains request/command metadata.
type Metadata struct {
	Command       string `json:"command"`
	Profile       string `json:"profile"`
	DurationMS    int64  `json:"duration_ms"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id,omitempty"`
}

// NewEnvelope creates a new success envelope.
func NewEnvelope(command, profile, schemaVersion, requestID string, data interface{}, duration time.Duration) *Envelope {
	return &Envelope{
		OK:   true,
		Data: data,
		Meta: Metadata{
			Command:       command,
			Profile:       profile,
			DurationMS:    duration.Milliseconds(),
			SchemaVersion: schemaVersion,
			RequestID:     requestID,
		},
	}
}

// NewErrorEnvelope creates a new error envelope.
func NewErrorEnvelope(command, profile, schemaVersion string, err *errors.Error, duration time.Duration) *ErrorEnvelope {
	return &ErrorEnvelope{
		OK:    false,
		Error: err,
		Meta: Metadata{
			Command:       command,
			Profile:       profile,
			DurationMS:    duration.Milliseconds(),
			SchemaVersion: schemaVersion,
		},
	}
}
