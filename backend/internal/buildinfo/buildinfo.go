package buildinfo

import "runtime"

// Set by -ldflags; VERSION at the repository root is the release source of truth.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	AppVersion             string   `json:"app_version"`
	Commit                 string   `json:"commit"`
	BuildTime              string   `json:"build_time"`
	GoVersion              string   `json:"go_version"`
	APIVersions            []string `json:"api_versions"`
	MinSupportedAppVersion string   `json:"min_supported_app_version"`
}

func Current(minVersion string) Info {
	return Info{Version, Commit, BuildTime, runtime.Version(), []string{"v1"}, minVersion}
}
