// This file implements `shukra version`. It exists so users and operators can
// quickly confirm which CLI build they are running when debugging installs or
// release issues.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Shukra CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), formatVersion(version, commit, date))
			return nil
		},
	}
}
