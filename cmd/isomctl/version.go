package main

import (
	"runtime"
)

// Version is replaced by the Makefile.
var Version = "git"

func version(e *env, args ...string) error {
	set := e.newFlagSet("version")
	set.Description = "Show version information."
	if err := set.Parse(args); err != nil {
		return err
	}

	e.stdout.Logf("isomctl version: %s (go runtime %s %s/%s)\n", Version,
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}
