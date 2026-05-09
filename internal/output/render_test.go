package output

import (
	"bytes"
	stderrors "errors"
	"testing"
	"time"

	clierrors "github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestRenderer_RenderSuccess(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, false)

	env := NewEnvelope("test", "default", SchemaVersion, "req-123", map[string]string{"foo": "bar"}, 10*time.Millisecond)
	err := renderer.RenderSuccess(env)

	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), `"ok":true`)
	assert.Contains(t, stdout.String(), `"data":{"foo":"bar"}`)
	assert.Contains(t, stdout.String(), `"request_id":"req-123"`)
}

func TestRenderer_RenderSuccessPretty(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, true)
	env := NewEnvelope("test", "default", SchemaVersion, "", map[string]string{"foo": "bar"}, time.Second)

	assert.NoError(t, renderer.RenderSuccess(env))
	assert.Contains(t, stdout.String(), "\n  \"ok\": true")
}

func TestRenderer_RenderSuccessNonJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, false, false)

	assert.NoError(t, renderer.RenderSuccess(NewEnvelope("test", "default", SchemaVersion, "", "value", 0)))
	assert.Equal(t, "", stdout.String())
}

type brokenJSON struct{}

func (brokenJSON) MarshalJSON() ([]byte, error) {
	return nil, stderrors.New("marshal failed")
}

func TestRenderer_RenderSuccessMarshalError(t *testing.T) {
	originalMarshal := marshalJSON
	defer func() { marshalJSON = originalMarshal }()
	marshalJSON = func(any) ([]byte, error) { return nil, stderrors.New("marshal failed") }

	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, false)
	env := NewEnvelope("test", "default", SchemaVersion, "", brokenJSON{}, 0)

	if err := renderer.RenderSuccess(env); err == nil {
		t.Fatal("RenderSuccess() error = nil, want failure")
	}
}

func TestRenderer_RenderErrorJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, false)
	env := NewErrorEnvelope("test", "default", SchemaVersion, &clierrors.Error{Message: "boom"}, 10*time.Millisecond)

	assert.NoError(t, renderer.RenderError(env))
	assert.Contains(t, stdout.String(), `"ok":false`)
	assert.Contains(t, stdout.String(), `"message":"boom"`)
}

func TestRenderer_RenderErrorTextAndDiagnostic(t *testing.T) {
	stderr := &bytes.Buffer{}
	renderer := NewRenderer(nil, stderr, false, false)
	env := NewErrorEnvelope("test", "default", SchemaVersion, &clierrors.Error{Message: "boom"}, 0)

	assert.NoError(t, renderer.RenderError(env))
	renderer.PrintDiagnostic("hello")
	assert.Contains(t, stderr.String(), "Error: boom")
	assert.Contains(t, stderr.String(), "hello")
}

func TestRenderer_RenderErrorPretty(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, true)
	env := NewErrorEnvelope("test", "default", SchemaVersion, &clierrors.Error{Message: "boom"}, 0)

	assert.NoError(t, renderer.RenderError(env))
	assert.Contains(t, stdout.String(), "\n  \"ok\": false")
}

func TestRenderer_RenderErrorMarshalError(t *testing.T) {
	originalMarshal := marshalJSON
	defer func() { marshalJSON = originalMarshal }()
	marshalJSON = func(any) ([]byte, error) { return nil, stderrors.New("marshal failed") }

	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, false)
	env := NewErrorEnvelope("test", "default", SchemaVersion, &clierrors.Error{Message: "boom"}, 0)

	if err := renderer.RenderError(env); err == nil {
		t.Fatal("RenderError() error = nil, want failure")
	}
}

func TestNewRendererDefaults(t *testing.T) {
	renderer := NewRenderer(nil, nil, false, false)
	assert.NotNil(t, renderer.Stdout)
	assert.NotNil(t, renderer.Stderr)
}
