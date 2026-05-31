package cli

import (
	"time"

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
