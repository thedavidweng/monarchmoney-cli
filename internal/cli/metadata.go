package cli

import (
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

func envelopeWithWarnings(command string, data any, start time.Time, warnings ...string) *output.Envelope {
	env := output.NewEnvelope(command, profile, output.SchemaVersion, "", data, time.Since(start))
	if len(warnings) > 0 {
		env.Meta.Warnings = append([]string(nil), warnings...)
	}
	return env
}
