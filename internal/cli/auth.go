package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"golang.org/x/term"
)

var (
	email     string
	password  string
	mfaCode   string
	mfaSecret string
)

var readPassword = term.ReadPassword
var scanInput = fmt.Scanln
var authenticateSession = auth.Authenticate
var newSessionStore = auth.NewStore
var defaultSessionPath = config.DefaultSessionPath
var exitFunc = os.Exit

type identityResult struct {
	Email string
}

var fetchIdentity = func(ctx context.Context, token string) (*identityResult, error) {
	client := graphql.NewClient("https://api.monarch.com/graphql", token, timeout)
	var resp struct {
		Me struct {
			Email string `json:"email"`
		} `json:"me"`
	}
	if err := client.Do(ctx, &graphql.Request{
		OperationName: "GetIdentity",
		Query:         graphql.GetIdentityQuery,
	}, &resp); err != nil {
		return nil, err
	}

	return &identityResult{Email: resp.Me.Email}, nil
}

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

		// Priority: Flags > Env Vars > Prompt
		email := viper.GetString("email")
		password := viper.GetString("password")
		mfaCode := viper.GetString("mfa-code")
		mfaSecret := viper.GetString("mfa-secret")

		if email == "" {
			fmt.Print("Email: ")
			scanInput(&email)
		}

		if password == "" {
			fmt.Print("Password: ")
			bytePassword, err := readPassword(int(os.Stdin.Fd()))
			fmt.Println() // New line after password input
			if err != nil {
				handleError(renderer, "login", errors.New(errors.InternalError, "failed to read password", errors.CatInternal, false, err), start)
				return
			}
			password = string(bytePassword)
		}

		sess, err := authenticateSession(email, password, mfaCode, mfaSecret)

		// Handle MFA requirement if not already provided
		if err != nil {
			if e, ok := err.(*errors.Error); ok && e.Code == errors.AuthMFARequired && !jsonMode {
				fmt.Print("MFA Code: ")
				scanInput(&mfaCode)
				sess, err = authenticateSession(email, password, mfaCode, mfaSecret)
			}
		}

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
		store := newSessionStore(defaultSessionPath())
		if err := store.Save(sess); err != nil {
			handleError(renderer, "auth.login", errors.New(errors.InternalError, "failed to save session", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("auth.login", profile, output.SchemaVersion, "", map[string]interface{}{
				"status":       "logged in",
				"email":        sess.Email,
				"profile":      sess.Profile,
				"created_at":   sess.CreatedAt,
				"updated_at":   sess.UpdatedAt,
				"session_path": defaultSessionPath(),
			}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully logged in as %s.\n", sess.Email)
			fmt.Printf("Logged in at: %s\n", sess.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Session token saved to: %s\n", defaultSessionPath())
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := newSessionStore(defaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "auth.status", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		identity, err := fetchIdentity(cmd.Context(), sess.Token)
		if err != nil {
			cliErr, ok := err.(*errors.Error)
			if !ok {
				cliErr = errors.New(errors.InternalError, "failed to verify session", errors.CatInternal, false, err)
			}
			handleError(renderer, "auth.status", cliErr, start)
			return
		}

		displayEmail := sess.Email
		if displayEmail == "" && identity != nil && identity.Email != "" {
			displayEmail = identity.Email
		}

		data := map[string]interface{}{
			"authenticated": true,
			"session_valid": true,
			"email":         displayEmail,
			"profile":       sess.Profile,
			"created_at":    sess.CreatedAt,
			"updated_at":    sess.UpdatedAt,
			"session_path":  defaultSessionPath(),
		}

		if jsonMode {
			env := output.NewEnvelope("auth.status", profile, output.SchemaVersion, "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Authenticated: yes\n")
			fmt.Printf("Email: %s\n", displayEmail)
			fmt.Printf("Profile: %s\n", sess.Profile)
			fmt.Printf("Logged in at: %s\n", sess.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Session valid: yes\n")
			fmt.Printf("Session path: %s\n", defaultSessionPath())
		}
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove local session",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := newSessionStore(defaultSessionPath())
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
		fmt.Println(defaultSessionPath())
	},
}

func init() {
	loginCmd.Flags().StringVar(&email, "email", "", "email address")
	loginCmd.Flags().StringVar(&password, "password", "", "password")
	loginCmd.Flags().StringVar(&mfaCode, "mfa-code", "", "6-digit MFA code")
	loginCmd.Flags().StringVar(&mfaSecret, "mfa-secret", "", "TOTP secret key for automatic MFA")

	viper.BindPFlag("email", loginCmd.Flags().Lookup("email"))
	viper.BindPFlag("password", loginCmd.Flags().Lookup("password"))
	viper.BindPFlag("mfa-code", loginCmd.Flags().Lookup("mfa-code"))
	viper.BindPFlag("mfa-secret", loginCmd.Flags().Lookup("mfa-secret"))

	sessionCmd.AddCommand(sessionPathCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(sessionCmd)
	RootCmd.AddCommand(authCmd)
}

func handleError(r *output.Renderer, command string, err *errors.Error, start time.Time) {
	if err != nil && err.Code == errors.AuthSessionExpired {
		store := newSessionStore(defaultSessionPath())
		if sess, loadErr := store.Load(); loadErr == nil {
			if sess.Email != "" {
				err = errors.New(err.Code, fmt.Sprintf("session token for %s stored at %s expired or invalid; run `monarch auth login` again", sess.Email, defaultSessionPath()), err.Category, err.Retryable, err.Err)
			} else {
				err = errors.New(err.Code, fmt.Sprintf("session token stored at %s expired or invalid; run `monarch auth login` again", defaultSessionPath()), err.Category, err.Retryable, err.Err)
			}
		}
	}

	env := output.NewErrorEnvelope(command, profile, output.SchemaVersion, err, time.Since(start))
	r.RenderError(env)
	exitFunc(err.ExitCode())
}
