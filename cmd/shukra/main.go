// This file is the executable entrypoint for the Shukra CLI. It exists so end
// users can install, bootstrap, and manage AppEnvironment resources without
// needing to remember raw helm, kubectl, and Kubernetes API commands.
package main

import (
	"fmt"
	"os"

	"github.com/sandy001-kki/Shukra/pkg/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cli.Execute(version, commit, date); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
