// This file wraps external tool execution used by install/bootstrap flows. It
// exists so helm, kubectl, and PowerShell invocations are assembled in one
// place, which keeps command behavior predictable and debuggable.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func appendHelmConnectionArgs(opts *RootOptions, args []string) []string {
	if opts == nil {
		return args
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	if opts.Context != "" {
		args = append(args, "--kube-context", opts.Context)
	}
	return args
}

func appendKubectlConnectionArgs(opts *RootOptions, args []string) []string {
	if opts == nil {
		return args
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	if opts.Context != "" {
		args = append(args, "--context", opts.Context)
	}
	return args
}
