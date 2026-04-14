// This file implements AppEnvironment-focused CLI commands. It exists so users
// can generate, inspect, and update Shukra environments from one coherent UX.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/spf13/cobra"
)

func newEnvCommand(opts *RootOptions) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Manage AppEnvironment resources",
	}

	envCmd.AddCommand(
		newEnvInitCommand(opts),
		newEnvApplyCommand(opts),
		newEnvStatusCommand(opts),
		newEnvPauseCommand(opts, true),
		newEnvPauseCommand(opts, false),
		newEnvDeleteCommand(opts),
		newEnvMigrationCommand(opts),
		newEnvRestoreCommand(opts),
	)

	return envCmd
}

func newEnvInitCommand(opts *RootOptions) *cobra.Command {
	var (
		image         string
		replicas      int32
		containerPort int32
		servicePort   int32
		outputPath    string
		ingressHost   string
	)

	cmd := &cobra.Command{
		Use:   "init NAME",
		Short: "Generate a starter AppEnvironment manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			payload, err := renderStarterManifest(name, opts.Namespace, image, replicas, containerPort, servicePort, ingressHost)
			if err != nil {
				return fmt.Errorf("marshal starter AppEnvironment: %w", err)
			}

			if outputPath == "" {
				_, err = cmd.OutOrStdout().Write(payload)
				return err
			}

			return os.WriteFile(outputPath, payload, 0o644)
		},
	}

	cmd.Flags().StringVar(&image, "image", "", "Container image for the application.")
	cmd.Flags().Int32Var(&replicas, "replicas", 2, "Replica count for the starter manifest.")
	cmd.Flags().Int32Var(&containerPort, "container-port", 8080, "Container port exposed by the app.")
	cmd.Flags().Int32Var(&servicePort, "service-port", 80, "Service port exposed inside the cluster.")
	cmd.Flags().StringVar(&outputPath, "output", "", "Optional file path to write instead of stdout.")
	cmd.Flags().StringVar(&ingressHost, "ingress-host", "", "Optional ingress host to include in the starter manifest.")
	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func renderStarterManifest(name, namespace, image string, replicas, containerPort, servicePort int32, ingressHost string) ([]byte, error) {
	spec := map[string]any{
		"app": map[string]any{
			"image":         image,
			"replicas":      replicas,
			"containerPort": containerPort,
			"livenessProbe": map[string]any{
				"httpGet": map[string]any{
					"path": "/",
					"port": containerPort,
				},
				"initialDelaySeconds": 10,
			},
			"readinessProbe": map[string]any{
				"httpGet": map[string]any{
					"path": "/",
					"port": containerPort,
				},
				"initialDelaySeconds": 5,
			},
		},
		"service": map[string]any{
			"enabled":    true,
			"type":       string(corev1.ServiceTypeClusterIP),
			"port":       servicePort,
			"targetPort": containerPort,
		},
	}

	if ingressHost != "" {
		spec["ingress"] = map[string]any{
			"enabled":  true,
			"host":     ingressHost,
			"path":     "/",
			"pathType": string(networkingPathType()),
		}
	}

	return yaml.Marshal(map[string]any{
		"apiVersion": appsv1beta1.GroupVersion.String(),
		"kind":       "AppEnvironment",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": spec,
	})
}

func newEnvApplyCommand(opts *RootOptions) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply an AppEnvironment manifest with kubectl",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			kubectlArgs := appendKubectlConnectionArgs(opts, []string{"apply", "-f", filePath})
			printTitle(cmd.OutOrStdout(), "Applying AppEnvironment")
			printKV(cmd.OutOrStdout(), "File", filePath)
			if err := runCommand("kubectl", kubectlArgs...); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), success("OK  Manifest applied"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Manifest file to apply.")
	return cmd
}

func newEnvStatusCommand(opts *RootOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "status NAME",
		Short: "Show a concise AppEnvironment status summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			kubeClient, _, err := buildClient(ctx, opts)
			if err != nil {
				return err
			}

			appEnv := &appsv1beta1.AppEnvironment{}
			if err := kubeClient.Get(ctx, types.NamespacedName{Name: args[0], Namespace: opts.Namespace}, appEnv); err != nil {
				return fmt.Errorf("get AppEnvironment %s/%s: %w", opts.Namespace, args[0], err)
			}

			switch output {
			case "json":
				payload, err := json.MarshalIndent(appEnv, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			case "yaml":
				payload, err := yaml.Marshal(appEnv)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			default:
				printStatusSummary(cmd, appEnv)
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "summary", "Output format: summary, yaml, or json.")
	return cmd
}

func newEnvPauseCommand(opts *RootOptions, pause bool) *cobra.Command {
	use := "resume"
	short := "Resume AppEnvironment reconciliation"
	if pause {
		use = "pause"
		short = "Pause AppEnvironment reconciliation"
	}

	return &cobra.Command{
		Use:   use + " NAME",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			kubeClient, _, err := buildClient(ctx, opts)
			if err != nil {
				return err
			}

			appEnv := &appsv1beta1.AppEnvironment{}
			key := types.NamespacedName{Name: args[0], Namespace: opts.Namespace}
			if err := kubeClient.Get(ctx, key, appEnv); err != nil {
				return fmt.Errorf("get AppEnvironment %s/%s: %w", opts.Namespace, args[0], err)
			}

			appEnv.Spec.Paused = pause
			if err := kubeClient.Update(ctx, appEnv); err != nil {
				return fmt.Errorf("update pause flag for %s/%s: %w", opts.Namespace, args[0], err)
			}

			state := "resumed"
			if pause {
				state = "paused"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "AppEnvironment %s/%s %s\n", opts.Namespace, args[0], state)
			return nil
		},
	}
}

func newEnvDeleteCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete an AppEnvironment resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			kubeClient, _, err := buildClient(ctx, opts)
			if err != nil {
				return err
			}

			appEnv := &appsv1beta1.AppEnvironment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      args[0],
					Namespace: opts.Namespace,
				},
			}
			if err := kubeClient.Delete(ctx, appEnv); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("delete AppEnvironment %s/%s: %w", opts.Namespace, args[0], err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "AppEnvironment %s/%s delete requested\n", opts.Namespace, args[0])
			return nil
		},
	}
}

func newEnvMigrationCommand(opts *RootOptions) *cobra.Command {
	var migrationID string
	var image string

	cmd := &cobra.Command{
		Use:   "migrate NAME",
		Short: "Update an AppEnvironment with a new migration request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			kubeClient, _, err := buildClient(ctx, opts)
			if err != nil {
				return err
			}

			appEnv := &appsv1beta1.AppEnvironment{}
			key := types.NamespacedName{Name: args[0], Namespace: opts.Namespace}
			if err := kubeClient.Get(ctx, key, appEnv); err != nil {
				return fmt.Errorf("get AppEnvironment %s/%s: %w", opts.Namespace, args[0], err)
			}

			appEnv.Spec.Migration.Enabled = true
			appEnv.Spec.Migration.MigrationID = migrationID
			if image != "" {
				appEnv.Spec.Migration.Image = image
			}

			if err := kubeClient.Update(ctx, appEnv); err != nil {
				return fmt.Errorf("update migration for %s/%s: %w", opts.Namespace, args[0], err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "AppEnvironment %s/%s migration set to %s\n", opts.Namespace, args[0], migrationID)
			return nil
		},
	}

	cmd.Flags().StringVar(&migrationID, "migration-id", "", "New migration idempotency key.")
	cmd.Flags().StringVar(&image, "image", "", "Optional override for the migration image.")
	_ = cmd.MarkFlagRequired("migration-id")
	return cmd
}

func newEnvRestoreCommand(opts *RootOptions) *cobra.Command {
	var triggerNonce string
	var image string
	var source string
	var mode string

	cmd := &cobra.Command{
		Use:   "restore NAME",
		Short: "Update an AppEnvironment with a new restore request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			kubeClient, _, err := buildClient(ctx, opts)
			if err != nil {
				return err
			}

			appEnv := &appsv1beta1.AppEnvironment{}
			key := types.NamespacedName{Name: args[0], Namespace: opts.Namespace}
			if err := kubeClient.Get(ctx, key, appEnv); err != nil {
				return fmt.Errorf("get AppEnvironment %s/%s: %w", opts.Namespace, args[0], err)
			}

			appEnv.Spec.Restore.Enabled = true
			appEnv.Spec.Restore.TriggerNonce = triggerNonce
			appEnv.Spec.Restore.Image = image
			appEnv.Spec.Restore.Source = source
			if mode != "" {
				appEnv.Spec.Restore.Mode = mode
			}

			if err := kubeClient.Update(ctx, appEnv); err != nil {
				return fmt.Errorf("update restore for %s/%s: %w", opts.Namespace, args[0], err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "AppEnvironment %s/%s restore set to %s\n", opts.Namespace, args[0], triggerNonce)
			return nil
		},
	}

	cmd.Flags().StringVar(&triggerNonce, "trigger-nonce", "", "New restore idempotency key.")
	cmd.Flags().StringVar(&image, "image", "", "Restore image used by the operator.")
	cmd.Flags().StringVar(&source, "source", "", "Restore source, such as a backup identifier or command string.")
	cmd.Flags().StringVar(&mode, "mode", "", "Optional restore mode, such as full or schema-only.")
	_ = cmd.MarkFlagRequired("trigger-nonce")
	_ = cmd.MarkFlagRequired("image")
	_ = cmd.MarkFlagRequired("source")
	return cmd
}

func printStatusSummary(cmd *cobra.Command, appEnv *appsv1beta1.AppEnvironment) {
	printTitle(cmd.OutOrStdout(), "AppEnvironment Status")
	printKV(cmd.OutOrStdout(), "Name", appEnv.Name)
	printKV(cmd.OutOrStdout(), "Namespace", appEnv.Namespace)
	printKV(cmd.OutOrStdout(), "Phase", appEnv.Status.Phase)
	printKV(cmd.OutOrStdout(), "Observed gen", fmt.Sprintf("%d", appEnv.Status.ObservedGeneration))
	printKV(cmd.OutOrStdout(), "URL", emptyDash(appEnv.Status.URL))
	printKV(cmd.OutOrStdout(), "Last error", emptyDash(appEnv.Status.LastError))
	printKV(cmd.OutOrStdout(), "Failure count", fmt.Sprintf("%d", appEnv.Status.FailureCount))
	printKV(cmd.OutOrStdout(), "Migration ID", emptyDash(appEnv.Status.LastAppliedMigrationID))
	printKV(cmd.OutOrStdout(), "Restore nonce", emptyDash(appEnv.Status.LastProcessedRestoreNonce))

	childResources := []string{
		appEnv.Status.ChildResources.ConfigMapName,
		appEnv.Status.ChildResources.ServiceName,
		appEnv.Status.ChildResources.DeploymentName,
		appEnv.Status.ChildResources.HPAName,
		appEnv.Status.ChildResources.IngressName,
		appEnv.Status.ChildResources.MigrationJobName,
		appEnv.Status.ChildResources.RestoreJobName,
		appEnv.Status.ChildResources.BackupCronJobName,
		appEnv.Status.ChildResources.NetworkPolicyName,
		appEnv.Status.ChildResources.PDBName,
	}
	childResources = filterEmpty(childResources)
	sort.Strings(childResources)
	fmt.Fprintln(cmd.OutOrStdout())
	printTitle(cmd.OutOrStdout(), "Child Resources")
	printNote(cmd.OutOrStdout(), "-", emptyDash(strings.Join(childResources, ", ")))

	fmt.Fprintln(cmd.OutOrStdout())
	printTitle(cmd.OutOrStdout(), "Conditions")
	for _, condition := range appEnv.Status.Conditions {
		line := fmt.Sprintf("%s=%s reason=%s message=%q", condition.Type, condition.Status, condition.Reason, condition.Message)
		printNote(cmd.OutOrStdout(), "-", line)
	}
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
