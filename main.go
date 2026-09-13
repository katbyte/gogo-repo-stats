package main

import (
	"os"

	c "github.com/gookit/color"
	"github.com/katbyte/go-kt/clog"
	"github.com/katbyte/gogo-repo-stats/cli"
)

const cmdName = "gogo-repo-stats"

func main() {
	// the log level comes from GOGO_REPO_STATS_LOG; read it once here, before anything logs
	clog.SetLevelFromEnv("GOGO_REPO_STATS_LOG")

	cmd, err := cli.Make(cmdName)
	if err != nil {
		clog.Log.Error(c.Sprintf("<red>%s: building cmd</> %v", cmdName, err))

		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		clog.Log.Error(c.Sprintf("<red>%s:</> %v", cmdName, err))

		os.Exit(1)
	}

	os.Exit(0)
}
