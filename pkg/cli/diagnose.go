// This file implements explicit diagnose commands outside chat mode. It exists
// so users and automation can request structured diagnostics without going
// through the English chat interface.
package cli

import "github.com/spf13/cobra"

func newDiagnoseCommand(opts *RootOptions) *cobra.Command {
	diagnoseCmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose operator or AppEnvironment health",
	}

	diagnoseCmd.AddCommand(
		&cobra.Command{
			Use:   "env NAME",
			Short: "Diagnose an AppEnvironment",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return chatDiagnose(cmd, opts, chatIntent{
					Action:    "diagnose",
					Name:      args[0],
					Namespace: opts.Namespace,
				})
			},
		},
		&cobra.Command{
			Use:   "operator",
			Short: "Diagnose the Shukra Operator control plane",
			RunE: func(cmd *cobra.Command, args []string) error {
				return chatDiagnose(cmd, opts, chatIntent{
					Action:    "diagnose",
					Target:    "operator",
					Namespace: opts.Namespace,
				})
			},
		},
	)

	return diagnoseCmd
}
