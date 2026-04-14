// This file exposes local environment bootstrap commands. It exists so the
// PowerShell bootstrap workflow becomes discoverable through the `shukra` CLI.
package cli

import "github.com/spf13/cobra"

func newBootstrapCommand() *cobra.Command {
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap local Shukra environments",
	}

	bootstrapCmd.AddCommand(&cobra.Command{
		Use:   "local",
		Short: "Run the one-command local bootstrap workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand("powershell", "-ExecutionPolicy", "Bypass", "-File", ".\\hack\\bootstrap-local.ps1")
		},
	})

	return bootstrapCmd
}
