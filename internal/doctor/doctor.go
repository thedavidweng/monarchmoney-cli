package doctor

import (
	"os"
	"runtime"

	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/version"
)

// Result represents the output of the doctor command.
type Result struct {
	Version      string `json:"version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Config       Report `json:"config"`
	Session      Report `json:"session"`
	Network      Report `json:"network"`
	Safety       Report `json:"safety"`
}

// Report represents a specific component's check report.
type Report struct {
	Path         string `json:"path,omitempty"`
	Exists       bool   `json:"exists"`
	Valid        bool   `json:"valid,omitempty"`
	PermissionOK bool   `json:"permission_ok,omitempty"`
	Authenticated bool  `json:"authenticated,omitempty"`
}

// Check performs local system and configuration checks.
func Check() *Result {
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
	_, err = os.Stat(sessPath)
	res.Session = Report{
		Path:   sessPath,
		Exists: !os.IsNotExist(err),
	}

	return res
}
