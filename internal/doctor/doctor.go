package doctor

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/version"
)

// Result represents the output of the doctor command.
type Result struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Config  Report `json:"config"`
	Session Report `json:"session"`
	Network Report `json:"network"`
	Safety  Report `json:"safety"`
}

// Report represents a specific component's check report.
type Report struct {
	Path          string `json:"path,omitempty"`
	Exists        bool   `json:"exists"`
	Valid         bool   `json:"valid,omitempty"`
	PermissionOK  bool   `json:"permission_ok,omitempty"`
	Authenticated bool   `json:"authenticated,omitempty"`
	APIReachable  bool   `json:"api_reachable,omitempty"`
}

// Check performs local system and configuration checks.
func Check(ctx context.Context, connect bool) *Result {
	res := &Result{
		Version: version.Version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}

	// Config check
	cfgPath := config.DefaultConfigPath()
	_, err := os.Stat(cfgPath)
	res.Config = Report{
		Path:   cfgPath,
		Exists: !os.IsNotExist(err),
	}

	// Session check
	sessPath := config.DefaultSessionPath()
	store := auth.NewStore(sessPath)
	sess, err := store.Load()
	res.Session = Report{
		Path:   sessPath,
		Exists: !os.IsNotExist(err),
	}

	if err == nil && sess != nil {
		res.Session.Authenticated = true
		// In a real implementation, we would check file permissions here for PermissionOK
		info, err := os.Stat(sessPath)
		if err == nil {
			res.Session.PermissionOK = (info.Mode()&0777 == 0600)
		}
	}

	if connect && res.Session.Authenticated {
		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, 10*time.Second)
		var identity interface{}
		err := client.Do(ctx, &graphql.Request{
			OperationName: "GetIdentity",
			Query:         graphql.GetIdentityQuery,
		}, &identity)

		if err == nil {
			res.Network.APIReachable = true
		}
	}

	return res
}
