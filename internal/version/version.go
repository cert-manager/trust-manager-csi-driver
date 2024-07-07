package version

import (
	"fmt"
	"os"
	"runtime"
)

func init() {
	if AppVersion == "" || GitCommit == "" {
		fmt.Fprintf(os.Stderr, "warning: AppVersion or GitCommit not set, this binary was built without -ldflags \"-X internal/version.AppVersion=... -X internal/version.GitCommit=...\"\n")
	}
}

type Version struct {
	AppVersion string `json:"appVersion"`
	GitCommit  string `json:"gitCommit"`
	GoVersion  string `json:"goVersion"`
	Compiler   string `json:"compiler"`
	Platform   string `json:"platform"`
}

// This variable block holds information used to build up the version string
var (
	AppVersion = ""
	GitCommit  = ""
)

func VersionInfo() Version {
	return Version{
		AppVersion: AppVersion,
		GitCommit:  GitCommit,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
