package output

import (
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

type Envelope struct {
	OK   bool     `json:"ok"`
	Data any      `json:"data,omitempty"`
	Meta Metadata `json:"meta"`
}

type ErrorEnvelope struct {
	OK    bool          `json:"ok"`
	Error *errors.Error `json:"error"`
	Meta  Metadata      `json:"meta"`
}

type Metadata struct {
	Command       string   `json:"command"`
	Profile       string   `json:"profile"`
	DurationMS    int64    `json:"duration_ms"`
	SchemaVersion string   `json:"schema_version"`
	RequestID     string   `json:"request_id,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

func NewEnvelope(command, profile, schemaVersion, requestID string, data any, duration time.Duration) *Envelope {
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
