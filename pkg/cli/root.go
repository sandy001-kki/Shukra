// This file defines the root command for the Shukra CLI. It exists to give all
// subcommands a shared place for kubeconfig, namespace, and shell execution
// behavior so the CLI remains consistent across features.
package cli

import "github.com/spf13/cobra"

// RootOptions holds shared CLI state that subcommands need to talk to a
// Kubernetes cluster or invoke local tooling like helm and kubectl.
type RootOptions struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

// Execute builds and runs the CLI tree. It accepts version metadata so release
// assets can embed the actual git tag and commit into `shukra version`.
func Execute(version, commit, date string) error {
	opts := &RootOptions{}
	rootCmd := &cobra.Command{
		Use:           "shukra",
		Short:         "Shukra CLI for installing and managing AppEnvironment resources",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  shukra install --operator-namespace shukra-system
  shukra env init demo-app --image nginx:1.27 --output demo-app.yaml
  shukra env apply -f demo-app.yaml
  shukra env status demo-app -n default`,
		Long: `Shukra is the human-friendly CLI companion to the Shukra Operator.

It helps users install the operator, bootstrap local clusters, generate starter
AppEnvironment manifests, and manage environment lifecycle actions without
memorizing raw kubectl and helm commands.`,
	}

	rootCmd.PersistentFlags().StringVar(&opts.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use.")
	rootCmd.PersistentFlags().StringVar(&opts.Context, "context", "", "Kubernetes context override for this command.")
	rootCmd.PersistentFlags().StringVarP(&opts.Namespace, "namespace", "n", "default", "Namespace used for AppEnvironment operations.")

	rootCmd.AddCommand(
		newVersionCommand(version, commit, date),
		newDoctorCommand(opts),
		newDiagnoseCommand(opts),
		newConsoleCommand(opts),
		newInstallCommand(opts),
		newUninstallCommand(opts),
		newBootstrapCommand(),
		newAskCommand(),
		newChatCommand(opts, version, commit, date),
		newEnvCommand(opts),
		newCompletionCommand(rootCmd),
	)

	return rootCmd.Execute()
}
