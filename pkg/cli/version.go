// This file implements `shukra version`. It exists so users and operators can
// quickly confirm which CLI build they are running when debugging installs or
// release issues.
package cli

import "github.com/spf13/cobra"

func newVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Shukra CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			printTitle(cmd.OutOrStdout(), "Shukra CLI")
			printKV(cmd.OutOrStdout(), "Version", version)
			printKV(cmd.OutOrStdout(), "Commit", commit)
			printKV(cmd.OutOrStdout(), "Build date", date)
			return nil
		},
	}
}
