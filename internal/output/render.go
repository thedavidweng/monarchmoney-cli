package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Renderer struct {
	Stdout io.Writer
	Stderr io.Writer
	JSON   bool
	Pretty bool
}

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

func (r *Renderer) RenderSuccess(env *Envelope) {
	if r.JSON {
		fmt.Fprintln(r.Stdout, r.marshal(env))
	}
}

func (r *Renderer) RenderError(env *ErrorEnvelope) {
	if r.JSON {
		fmt.Fprintln(r.Stdout, r.marshal(env))
		return
	}
	fmt.Fprintf(r.Stderr, "Error: %s\n", env.Error.Message)
}

func (r *Renderer) PrintDiagnostic(msg string) {
	fmt.Fprintln(r.Stderr, msg)
}

func (r *Renderer) marshal(v any) string {
	var data []byte
	if r.Pretty {
		data, _ = json.MarshalIndent(v, "", "  ")
	} else {
		data, _ = json.Marshal(v)
	}
	return string(data)
}
