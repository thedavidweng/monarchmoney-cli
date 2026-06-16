package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

func TestNewEnvelope(t *testing.T) {
	t.Run("sets ok to true and populates all fields", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		dur := 150 * time.Millisecond

		env := NewEnvelope("get-accounts", "default", "1.0", "req-123", data, dur)

		require.NotNil(t, env)
		assert.True(t, env.OK)
		assert.Equal(t, data, env.Data)
		assert.Equal(t, "get-accounts", env.Meta.Command)
		assert.Equal(t, "default", env.Meta.Profile)
		assert.Equal(t, "1.0", env.Meta.SchemaVersion)
		assert.Equal(t, "req-123", env.Meta.RequestID)
		assert.Equal(t, int64(150), env.Meta.DurationMS)
	})

	t.Run("with nil data", func(t *testing.T) {
		env := NewEnvelope("cmd", "p", "1", "r", nil, 0)

		require.NotNil(t, env)
		assert.True(t, env.OK)
		assert.Nil(t, env.Data)
	})

	t.Run("with empty strings", func(t *testing.T) {
		env := NewEnvelope("", "", "", "", "something", 0)

		require.NotNil(t, env)
		assert.True(t, env.OK)
		assert.Equal(t, "", env.Meta.Command)
		assert.Equal(t, "", env.Meta.Profile)
		assert.Equal(t, "", env.Meta.SchemaVersion)
		assert.Equal(t, "", env.Meta.RequestID)
	})

	t.Run("with zero duration", func(t *testing.T) {
		env := NewEnvelope("cmd", "p", "1", "r", "d", 0)

		assert.Equal(t, int64(0), env.Meta.DurationMS)
	})

	t.Run("converts duration to milliseconds correctly", func(t *testing.T) {
		tests := []struct {
			name     string
			dur      time.Duration
			expected int64
		}{
			{"1ms", 1 * time.Millisecond, 1},
			{"1s", 1 * time.Second, 1000},
			{"250ms", 250 * time.Millisecond, 250},
			{"1m", 1 * time.Minute, 60000},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := NewEnvelope("cmd", "p", "1", "r", nil, tt.dur)
				assert.Equal(t, tt.expected, env.Meta.DurationMS)
			})
		}
	})
}

func TestNewErrorEnvelope(t *testing.T) {
	t.Run("sets ok to false and populates all fields", func(t *testing.T) {
		err := errors.New(errors.AuthRequired, "login needed", errors.CatAuth, false, nil)
		dur := 50 * time.Millisecond

		env := NewErrorEnvelope("sync", "work", "1.0", err, dur)

		require.NotNil(t, env)
		assert.False(t, env.OK)
		assert.Equal(t, err, env.Error)
		assert.Equal(t, "sync", env.Meta.Command)
		assert.Equal(t, "work", env.Meta.Profile)
		assert.Equal(t, "1.0", env.Meta.SchemaVersion)
		assert.Equal(t, int64(50), env.Meta.DurationMS)
	})

	t.Run("with nil error", func(t *testing.T) {
		env := NewErrorEnvelope("cmd", "p", "1", nil, 0)

		require.NotNil(t, env)
		assert.False(t, env.OK)
		assert.Nil(t, env.Error)
	})

	t.Run("with empty strings and zero duration", func(t *testing.T) {
		err := errors.New(errors.InternalError, "", errors.CatInternal, true, nil)
		env := NewErrorEnvelope("", "", "", err, 0)

		require.NotNil(t, env)
		assert.False(t, env.OK)
		assert.Equal(t, "", env.Meta.Command)
		assert.Equal(t, "", env.Meta.Profile)
		assert.Equal(t, "", env.Meta.SchemaVersion)
		assert.Equal(t, int64(0), env.Meta.DurationMS)
	})

	t.Run("converts duration to milliseconds correctly", func(t *testing.T) {
		err := errors.New(errors.NetworkTimeout, "timeout", errors.CatNetwork, true, nil)
		env := NewErrorEnvelope("cmd", "p", "1", err, 3*time.Second+250*time.Millisecond)

		assert.Equal(t, int64(3250), env.Meta.DurationMS)
	})
}
