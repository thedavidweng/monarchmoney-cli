package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var marshalJSON = json.Marshal
var marshalJSONIndent = json.MarshalIndent

// Renderer handles writing data to the appropriate output streams.
type Renderer struct {
	Stdout io.Writer
	Stderr io.Writer
	JSON   bool
	Pretty bool
}

// NewRenderer returns a new Renderer.
func NewRenderer(stdout, stderr io.Writer, jsonMode, pretty bool) *Renderer {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &Renderer{
		Stdout: stdout,
		Stderr: stderr,
		JSON:   jsonMode,
		Pretty: pretty,
	}
}

// RenderSuccess writes a successful result to stdout.
func (r *Renderer) RenderSuccess(env *Envelope) error {
	if r.JSON {
		var data []byte
		var err error
		if r.Pretty {
			data, err = marshalJSONIndent(env, "", "  ")
		} else {
			data, err = marshalJSON(env)
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(r.Stdout, string(data))
		return nil
	}

	// Non-JSON output should be implemented per command, 
	// but for now we provide a fallback or just do nothing if only JSON is expected.
	// For this task, stdout should ideally only contain result data.
	return nil
}

// RenderError writes an error to stdout (as JSON) or stderr (as text).
func (r *Renderer) RenderError(env *ErrorEnvelope) error {
	if r.JSON {
		var data []byte
		var err error
		if r.Pretty {
			data, err = marshalJSONIndent(env, "", "  ")
		} else {
			data, err = marshalJSON(env)
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(r.Stdout, string(data))
		return nil
	}

	fmt.Fprintf(r.Stderr, "Error: %s\n", env.Error.Message)
	return nil
}

// PrintDiagnostic writes a diagnostic message to stderr.
func (r *Renderer) PrintDiagnostic(msg string) {
	fmt.Fprintln(r.Stderr, msg)
}
