// This file implements operator installation commands. It exists so users can
// install or remove Shukra through a stable CLI instead of hand-assembling helm
// commands every time.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInstallCommand(opts *RootOptions) *cobra.Command {
	var operatorNamespace string
	var chartRef string
	var chartVersion string
	var imageRepository string
	var imageTag string
	var useOCI bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install or upgrade the Shukra Operator with Helm",
		RunE: func(cmd *cobra.Command, args []string) error {
			releaseChart := chartRef
			if useOCI {
				releaseChart = "oci://ghcr.io/sandy001-kki/charts/shukra-operator"
			}

			helmArgs := []string{
				"upgrade", "--install", "shukra-operator", releaseChart,
				"-n", operatorNamespace,
				"--create-namespace",
				"--set", fmt.Sprintf("leaderElection.namespace=%s", operatorNamespace),
				"--wait",
				"--timeout", "10m",
			}

			if chartVersion != "" {
				helmArgs = append(helmArgs, "--version", chartVersion)
			}
			if imageRepository != "" {
				helmArgs = append(helmArgs, "--set", fmt.Sprintf("image.repository=%s", imageRepository))
			}
			if imageTag != "" {
				helmArgs = append(helmArgs, "--set", fmt.Sprintf("image.tag=%s", imageTag))
			}

			helmArgs = appendHelmConnectionArgs(opts, helmArgs)
			printTitle(cmd.OutOrStdout(), "Installing Shukra Operator")
			printKV(cmd.OutOrStdout(), "Namespace", operatorNamespace)
			printKV(cmd.OutOrStdout(), "Chart", releaseChart)
			if chartVersion != "" {
				printKV(cmd.OutOrStdout(), "Chart version", chartVersion)
			}

			if err := runCommand("helm", helmArgs...); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), success("OK  Shukra Operator is installed"))
			return nil
		},
	}

	cmd.Flags().StringVar(&operatorNamespace, "operator-namespace", "shukra-system", "Namespace where the operator is installed.")
	cmd.Flags().StringVar(&chartRef, "chart", "charts/shukra-operator", "Helm chart reference or local chart path.")
	cmd.Flags().StringVar(&chartVersion, "chart-version", "", "Helm chart version to install when using an OCI chart.")
	cmd.Flags().StringVar(&imageRepository, "image-repository", "", "Override the controller image repository.")
	cmd.Flags().StringVar(&imageTag, "image-tag", "", "Override the controller image tag.")
	cmd.Flags().BoolVar(&useOCI, "oci", false, "Install from the published OCI chart instead of the local chart directory.")
	_ = opts

	return cmd
}

func newUninstallCommand(opts *RootOptions) *cobra.Command {
	var operatorNamespace string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the Shukra Operator Helm release",
		RunE: func(cmd *cobra.Command, args []string) error {
			helmArgs := appendHelmConnectionArgs(opts, []string{"uninstall", "shukra-operator", "-n", operatorNamespace})
			printTitle(cmd.OutOrStdout(), "Uninstalling Shukra Operator")
			printKV(cmd.OutOrStdout(), "Namespace", operatorNamespace)
			if err := runCommand("helm", helmArgs...); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), success("OK  Shukra Operator is uninstalled"))
			return nil
		},
	}

	cmd.Flags().StringVar(&operatorNamespace, "operator-namespace", "shukra-system", "Namespace where the operator is installed.")
	return cmd
}
