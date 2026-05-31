package cli

import (
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

// CommandDeps bundles the dependencies every command handler needs.
type CommandDeps struct {
	Start    time.Time
	Renderer *output.Renderer
	Service  *monarch.Service
}

// newDeps constructs a CommandDeps by loading the session, creating the GraphQL
// client, and building the monarch.Service. On failure it renders the error and
// returns ok=false; callers should return immediately.
func newDeps(renderer *output.Renderer, command string, start time.Time) (CommandDeps, bool) {
	cfg, err := config.Load()
	if err != nil {
		handleError(renderer, command, errors.New(errors.InternalError, "failed to load config", errors.CatInternal, false, err), start)
		return CommandDeps{}, false
	}

	store := auth.NewStore(defaultSessionPath())
	sess, err := store.Load()
	if err != nil {
		handleError(renderer, command, errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
		return CommandDeps{}, false
	}

	client := graphql.NewClient(cfg.APIEndpoint, sess.Token, cfg.Timeout)
	return CommandDeps{
		Start:    start,
		Renderer: renderer,
		Service:  monarch.NewService(client),
	}, true
}

// wrapError converts a generic error into a structured *errors.Error.
// If err is already an *errors.Error, it is returned as-is.
func wrapError(err error, message string) *errors.Error {
	if e, ok := err.(*errors.Error); ok {
		return e
	}
	return errors.New(errors.APIError, message, errors.CatAPI, false, err)
}

// Mutate executes a write operation with audit logging. It runs fn, logs the
// result to the audit log, and either returns the result or renders the error.
// Callers handle success output (JSON envelope or human-readable text).
// Returns nil if the mutation failed (error already rendered).
func (d CommandDeps) Mutate(command, resourceID string, fn func() (interface{}, error), failMsg string) (interface{}, error) {
	result, err := fn()

	resultStr := "success"
	var errCode string
	if err != nil {
		resultStr = "failure"
		if e, ok := err.(*errors.Error); ok {
			errCode = string(e.Code)
		}
	}

	audit.NewLogger().Log(&audit.Record{
		Command:    command,
		ResourceID: resourceID,
		DryRun:     false,
		Confirmed:  confirm,
		Profile:    profile,
		Result:     resultStr,
		ErrorCode:  errCode,
	})

	if err != nil {
		handleError(d.Renderer, command, wrapError(err, failMsg), d.Start)
		return nil, err
	}

	return result, nil
}
