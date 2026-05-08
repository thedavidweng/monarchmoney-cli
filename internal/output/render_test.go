package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderer_RenderSuccess(t *testing.T) {
	stdout := &bytes.Buffer{}
	renderer := NewRenderer(stdout, nil, true, false)

	env := NewEnvelope("test", "default", "2026-05-08", "req-123", map[string]string{"foo": "bar"}, 10*time.Millisecond)
	err := renderer.RenderSuccess(env)

	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), `"ok":true`)
	assert.Contains(t, stdout.String(), `"data":{"foo":"bar"}`)
	assert.Contains(t, stdout.String(), `"request_id":"req-123"`)
}
