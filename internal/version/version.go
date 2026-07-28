package version

import (
	"fmt"
	"runtime/debug"
	"sync"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

var (
	once  sync.Once
	build *debug.BuildInfo
)

func resolve() {
	once.Do(func() {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		build = info
		if Commit == "none" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && setting.Value != "" {
					Commit = setting.Value
					break
				}
			}
		}
		if Date == "unknown" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.time" && setting.Value != "" {
					Date = setting.Value
					break
				}
			}
		}
		if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	})
}

func GetVersion() string {
	resolve()
	return Version
}

func GetCommit() string {
	resolve()
	if len(Commit) > 12 {
		return Commit[:12]
	}
	return Commit
}

func GetDate() string {
	resolve()
	return Date
}

func GetBuiltBy() string {
	resolve()
	return BuiltBy
}

func GetBuildInfo() string {
	resolve()
	if build == nil {
		return "build info unavailable"
	}
	return fmt.Sprintf("go %s, module %s, path %s", build.GoVersion, build.Main.Path, build.Path)
}
