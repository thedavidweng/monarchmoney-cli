package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"golang.org/x/term"
)

var (
	email     string
	password  string
	mfaCode   string
	mfaSecret string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and session",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Monarch Money",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if email == "" {
			fmt.Print("Email: ")
			fmt.Scanln(&email)
		}

		if password == "" {
			fmt.Print("Password: ")
			bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println() // New line after password input
			if err != nil {
				handleError(renderer, "login", errors.New(errors.InternalError, "failed to read password", errors.CatInternal, false, err), start)
				return
			}
			password = string(bytePassword)
		}

		sess, err := auth.Authenticate(email, password, mfaCode, mfaSecret)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.InternalError, "authentication failed", errors.CatInternal, false, err)
			}
			handleError(renderer, "auth.login", cliErr, start)
			return
		}

		sess.Profile = profile
		store := auth.NewStore(config.DefaultSessionPath())
		if err := store.Save(sess); err != nil {
			handleError(renderer, "auth.login", errors.New(errors.InternalError, "failed to save session", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("auth.login", profile, output.SchemaVersion, "", map[string]string{"status": "logged in"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Successfully logged in.")
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "auth.status", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		// In Phase 2.5, we'll add a connected check to GetIdentity.
		// For now, just check if local session exists.

		data := map[string]interface{}{
			"authenticated": true,
			"profile":       sess.Profile,
			"created_at":    sess.CreatedAt,
		}

		if jsonMode {
			env := output.NewEnvelope("auth.status", profile, output.SchemaVersion, "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Authenticated as profile: %s\n", sess.Profile)
			fmt.Printf("Session created: %s\n", sess.CreatedAt.Format(time.RFC3339))
		}
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove local session",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		if err := store.Delete(); err != nil && !os.IsNotExist(err) {
			handleError(renderer, "auth.logout", errors.New(errors.InternalError, "failed to delete session", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("auth.logout", profile, output.SchemaVersion, "", map[string]string{"status": "logged out"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Successfully logged out.")
		}
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage local session",
}

var sessionPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the path to the session file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.DefaultSessionPath())
	},
}

func init() {
	loginCmd.Flags().StringVar(&email, "email", "", "email address")
	loginCmd.Flags().StringVar(&password, "password", "", "password")
	loginCmd.Flags().StringVar(&mfaCode, "mfa-code", "", "6-digit MFA code")

	sessionCmd.AddCommand(sessionPathCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(sessionCmd)
	RootCmd.AddCommand(authCmd)
}

func handleError(r *output.Renderer, command string, err *errors.Error, start time.Time) {
	env := output.NewErrorEnvelope(command, profile, output.SchemaVersion, err, time.Since(start))
	r.RenderError(env)
	os.Exit(err.ExitCode())
}
