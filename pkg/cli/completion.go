// This file exposes shell completion generation for the Shukra CLI. It exists
// so users can integrate the CLI with PowerShell, Bash, Zsh, and Fish shells
// instead of remembering every command or flag manually.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
	}

	completionCmd.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generate Bash completions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rootCmd.GenBashCompletion(os.Stdout)
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generate Zsh completions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rootCmd.GenZshCompletion(os.Stdout)
			},
		},
		&cobra.Command{
			Use:   "fish",
			Short: "Generate Fish completions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rootCmd.GenFishCompletion(os.Stdout, true)
			},
		},
		&cobra.Command{
			Use:   "powershell",
			Short: "Generate PowerShell completions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			},
		},
	)

	return completionCmd
}
