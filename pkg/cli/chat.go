// This file implements an English-first interactive chat mode for the Shukra
// CLI. It exists so PowerShell users can manage the operator and environments
// through conversational commands instead of memorizing many subcommands.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/spf13/cobra"
)

var errChatExit = errors.New("shukra chat exit requested")

// chatIntent is a normalized action extracted from natural-language input.
// Keeping the parser output small makes the REPL easier to evolve over time.
type chatIntent struct {
	Action        string
	Name          string
	File          string
	Namespace     string
	MigrationID   string
	Image         string
	Source        string
	TriggerNonce  string
	RestoreMode   string
	ChartVersion  string
	UseOCI        bool
	UnknownReason string
}

func newChatCommand(opts *RootOptions, version, commit, date string) *cobra.Command {
	var oneShot string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Open an English-first interactive Shukra assistant in your terminal",
		Long: `Shukra chat opens an assistant-style terminal session for common
operator and AppEnvironment tasks.

You can type plain English commands such as:
  status basic-app
  apply examples/basic.yaml
  install operator from oci version 0.2.2
  show operator logs
  pause basic-app
  resume basic-app
  delete basic-app
  bootstrap local`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if oneShot != "" {
				return executeChatInput(cmd, opts, version, commit, date, oneShot)
			}
			if len(args) > 0 {
				return executeChatInput(cmd, opts, version, commit, date, strings.Join(args, " "))
			}
			return runChatREPL(cmd, opts, version, commit, date)
		},
	}

	cmd.Flags().StringVar(&oneShot, "message", "", "Run one English command and exit without opening the interactive prompt.")
	return cmd
}

func runChatREPL(cmd *cobra.Command, opts *RootOptions, version, commit, date string) error {
	in := cmd.InOrStdin()
	out := cmd.OutOrStdout()

	printTitle(out, "Shukra Chat")
	fmt.Fprintln(out, "  Your English-first assistant for the Shukra Operator.")
	fmt.Fprintln(out)
	printTitle(out, "Try saying")
	printNote(out, "-", "status basic-app")
	printNote(out, "-", "apply examples/basic.yaml")
	printNote(out, "-", "show operator logs")
	printNote(out, "-", "pause basic-app")
	printNote(out, "-", "bootstrap local")
	printNote(out, "-", "quit")
	fmt.Fprintln(out)
	printKV(out, "Namespace", opts.Namespace)
	printKV(out, "Version", displayVersion(version, commit, date))
	fmt.Fprintln(out)

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, header("shukra> "))
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			fmt.Fprintln(out)
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := executeChatInput(cmd, opts, version, commit, date, line); err != nil {
			if errors.Is(err, errChatExit) {
				return nil
			}
			return err
		}
	}
}

func executeChatInput(cmd *cobra.Command, opts *RootOptions, version, commit, date, input string) error {
	intent := parseChatIntent(input, opts.Namespace)

	switch intent.Action {
	case "quit":
		fmt.Fprintln(cmd.OutOrStdout(), success("Bye. Your Shukra session is complete."))
		return errChatExit
	case "help":
		printChatHelp(cmd.OutOrStdout())
		return nil
	case "version":
		printTitle(cmd.OutOrStdout(), "Shukra CLI Version")
		printKV(cmd.OutOrStdout(), "Version", displayVersion(version, commit, date))
		return nil
	case "status":
		return chatStatus(cmd, opts, intent)
	case "apply":
		return chatApply(cmd, opts, intent)
	case "pause":
		return chatPauseResume(cmd, opts, intent, true)
	case "resume":
		return chatPauseResume(cmd, opts, intent, false)
	case "delete":
		return chatDelete(cmd, opts, intent)
	case "migrate":
		return chatMigrate(cmd, opts, intent)
	case "restore":
		return chatRestore(cmd, opts, intent)
	case "logs":
		return chatLogs(opts)
	case "install":
		return chatInstall(cmd, opts, intent)
	case "uninstall":
		return chatUninstall(cmd, opts)
	case "bootstrap":
		return chatBootstrap(cmd)
	default:
		printTitle(cmd.OutOrStdout(), "I couldn't understand that yet")
		if intent.UnknownReason != "" {
			printNote(cmd.OutOrStdout(), "-", intent.UnknownReason)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		printChatHelp(cmd.OutOrStdout())
		return nil
	}
}

func parseChatIntent(input, defaultNamespace string) chatIntent {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.Join(strings.Fields(normalized), " ")
	intent := chatIntent{Namespace: defaultNamespace}

	switch normalized {
	case "exit", "quit", "bye":
		intent.Action = "quit"
		return intent
	case "help", "what can you do", "commands":
		intent.Action = "help"
		return intent
	case "version", "show version":
		intent.Action = "version"
		return intent
	case "bootstrap", "bootstrap local", "start local", "setup local":
		intent.Action = "bootstrap"
		return intent
	case "install", "install operator", "install shukra", "install shukra operator":
		intent.Action = "install"
		return intent
	case "uninstall", "uninstall operator", "remove operator":
		intent.Action = "uninstall"
		return intent
	case "logs", "show operator logs", "operator logs", "show logs":
		intent.Action = "logs"
		return intent
	}

	if strings.HasPrefix(normalized, "apply ") {
		intent.Action = "apply"
		intent.File = strings.TrimSpace(input[len("apply "):])
		return intent
	}
	if strings.HasPrefix(normalized, "status ") {
		intent.Action = "status"
		intent.Name = firstWord(strings.TrimSpace(input[len("status "):]))
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		return intent
	}
	if strings.HasPrefix(normalized, "pause ") {
		intent.Action = "pause"
		intent.Name = firstWord(strings.TrimSpace(input[len("pause "):]))
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		return intent
	}
	if strings.HasPrefix(normalized, "resume ") {
		intent.Action = "resume"
		intent.Name = firstWord(strings.TrimSpace(input[len("resume "):]))
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		return intent
	}
	if strings.HasPrefix(normalized, "delete ") {
		intent.Action = "delete"
		intent.Name = firstWord(strings.TrimSpace(input[len("delete "):]))
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		return intent
	}
	if strings.HasPrefix(normalized, "show status for ") {
		intent.Action = "status"
		intent.Name = firstWord(strings.TrimSpace(input[len("show status for "):]))
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		return intent
	}
	if strings.HasPrefix(normalized, "install operator from oci") || strings.Contains(normalized, "install") && strings.Contains(normalized, "oci") {
		intent.Action = "install"
		intent.UseOCI = true
		intent.ChartVersion = parseVersionToken(normalized)
		return intent
	}
	if strings.HasPrefix(normalized, "migrate ") || strings.Contains(normalized, " migration") {
		intent.Action = "migrate"
		intent.Name = detectNameFromAction(normalized, "migrate")
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		intent.MigrationID = parseAfterKeyword(input, "id")
		intent.Image = parseAfterKeyword(input, "image")
		return intent
	}
	if strings.HasPrefix(normalized, "restore ") || strings.Contains(normalized, " restore") {
		intent.Action = "restore"
		intent.Name = detectNameFromAction(normalized, "restore")
		intent.Namespace = parseNamespace(normalized, defaultNamespace)
		intent.TriggerNonce = parseAfterKeyword(input, "nonce")
		intent.Image = parseAfterKeyword(input, "image")
		intent.Source = parseAfterKeyword(input, "source")
		intent.RestoreMode = parseAfterKeyword(input, "mode")
		return intent
	}

	intent.UnknownReason = "Try a plain instruction like `status basic-app`, `apply examples/basic.yaml`, or `show operator logs`."
	return intent
}

func parseNamespace(normalized, fallback string) string {
	re := regexp.MustCompile(`(?:in|namespace)\s+([a-z0-9-]+)`)
	match := re.FindStringSubmatch(normalized)
	if len(match) == 2 {
		return match[1]
	}
	return fallback
}

func parseVersionToken(normalized string) string {
	re := regexp.MustCompile(`version\s+([0-9]+\.[0-9]+\.[0-9]+)`)
	match := re.FindStringSubmatch(normalized)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func parseAfterKeyword(input, keyword string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\s+([^\s]+)`)
	match := re.FindStringSubmatch(input)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func firstWord(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func detectNameFromAction(normalized, action string) string {
	fields := strings.Fields(normalized)
	for idx, field := range fields {
		if field == action && idx+1 < len(fields) {
			next := fields[idx+1]
			switch next {
			case "for", "with", "using":
				continue
			default:
				return next
			}
		}
	}
	return ""
}

func printChatHelp(out io.Writer) {
	printTitle(out, "What you can say")
	printNote(out, "-", "status basic-app")
	printNote(out, "-", "show status for basic-app in default")
	printNote(out, "-", "apply examples/basic.yaml")
	printNote(out, "-", "pause basic-app")
	printNote(out, "-", "resume basic-app")
	printNote(out, "-", "delete basic-app")
	printNote(out, "-", "show operator logs")
	printNote(out, "-", "install operator from oci version 0.2.2")
	printNote(out, "-", "bootstrap local")
	printNote(out, "-", "quit")
}

func displayVersion(version, commit, date string) string {
	parts := []string{}
	if version != "" {
		parts = append(parts, version)
	}
	if commit != "" {
		parts = append(parts, commit)
	}
	if date != "" {
		parts = append(parts, date)
	}
	if len(parts) == 0 {
		return "dev"
	}
	return strings.Join(parts, " | ")
}

func chatStatus(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	if intent.Name == "" {
		return fmt.Errorf("please tell me which AppEnvironment you want, for example: status basic-app")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chatOpts := *opts
	chatOpts.Namespace = intent.Namespace
	kubeClient, _, err := buildClient(ctx, &chatOpts)
	if err != nil {
		return err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, appEnv); err != nil {
		return fmt.Errorf("get AppEnvironment %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	printStatusSummary(cmd, appEnv)
	return nil
}

func chatApply(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	if intent.File == "" {
		return fmt.Errorf("please tell me which file to apply, for example: apply examples/basic.yaml")
	}

	resolvedPath := intent.File
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Clean(resolvedPath)
	}
	kubectlArgs := appendKubectlConnectionArgs(opts, []string{"apply", "-f", resolvedPath})
	printTitle(cmd.OutOrStdout(), "Applying AppEnvironment")
	printKV(cmd.OutOrStdout(), "File", resolvedPath)
	if err := runCommand("kubectl", kubectlArgs...); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), success("OK  Manifest applied"))
	return nil
}

func chatPauseResume(cmd *cobra.Command, opts *RootOptions, intent chatIntent, pause bool) error {
	if intent.Name == "" {
		action := "pause"
		if !pause {
			action = "resume"
		}
		return fmt.Errorf("please tell me which AppEnvironment to %s, for example: %s basic-app", action, action)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chatOpts := *opts
	chatOpts.Namespace = intent.Namespace
	kubeClient, _, err := buildClient(ctx, &chatOpts)
	if err != nil {
		return err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	key := types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}
	if err := kubeClient.Get(ctx, key, appEnv); err != nil {
		return fmt.Errorf("get AppEnvironment %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	appEnv.Spec.Paused = pause
	if err := kubeClient.Update(ctx, appEnv); err != nil {
		return fmt.Errorf("update pause flag for %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	state := "resumed"
	if pause {
		state = "paused"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", success(fmt.Sprintf("OK  AppEnvironment %s/%s %s", intent.Namespace, intent.Name, state)))
	return nil
}

func chatDelete(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	if intent.Name == "" {
		return fmt.Errorf("please tell me which AppEnvironment to delete, for example: delete basic-app")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chatOpts := *opts
	chatOpts.Namespace = intent.Namespace
	kubeClient, _, err := buildClient(ctx, &chatOpts)
	if err != nil {
		return err
	}

	appEnv := &appsv1beta1.AppEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	if err := kubeClient.Delete(ctx, appEnv); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete AppEnvironment %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", success(fmt.Sprintf("OK  Delete requested for %s/%s", intent.Namespace, intent.Name)))
	return nil
}

func chatMigrate(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	if intent.Name == "" || intent.MigrationID == "" {
		return fmt.Errorf("please say something like: migrate basic-app with id v2")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chatOpts := *opts
	chatOpts.Namespace = intent.Namespace
	kubeClient, _, err := buildClient(ctx, &chatOpts)
	if err != nil {
		return err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	key := types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}
	if err := kubeClient.Get(ctx, key, appEnv); err != nil {
		return fmt.Errorf("get AppEnvironment %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	appEnv.Spec.Migration.Enabled = true
	appEnv.Spec.Migration.MigrationID = intent.MigrationID
	if intent.Image != "" {
		appEnv.Spec.Migration.Image = intent.Image
	}

	if err := kubeClient.Update(ctx, appEnv); err != nil {
		return fmt.Errorf("update migration for %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", success(fmt.Sprintf("OK  Migration %s requested for %s/%s", intent.MigrationID, intent.Namespace, intent.Name)))
	return nil
}

func chatRestore(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	if intent.Name == "" || intent.TriggerNonce == "" || intent.Image == "" || intent.Source == "" {
		return fmt.Errorf("please say something like: restore basic-app with nonce restore-001 image busybox:1.36 source s3://bucket/backup")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	chatOpts := *opts
	chatOpts.Namespace = intent.Namespace
	kubeClient, _, err := buildClient(ctx, &chatOpts)
	if err != nil {
		return err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	key := types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}
	if err := kubeClient.Get(ctx, key, appEnv); err != nil {
		return fmt.Errorf("get AppEnvironment %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	appEnv.Spec.Restore.Enabled = true
	appEnv.Spec.Restore.TriggerNonce = intent.TriggerNonce
	appEnv.Spec.Restore.Image = intent.Image
	appEnv.Spec.Restore.Source = intent.Source
	if intent.RestoreMode != "" {
		appEnv.Spec.Restore.Mode = intent.RestoreMode
	}

	if err := kubeClient.Update(ctx, appEnv); err != nil {
		return fmt.Errorf("update restore for %s/%s: %w", intent.Namespace, intent.Name, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", success(fmt.Sprintf("OK  Restore %s requested for %s/%s", intent.TriggerNonce, intent.Namespace, intent.Name)))
	return nil
}

func chatLogs(opts *RootOptions) error {
	kubectlArgs := appendKubectlConnectionArgs(opts, []string{"logs", "-n", "shukra-system", "deploy/shukra-operator", "--tail=100"})
	return runCommand("kubectl", kubectlArgs...)
}

func chatInstall(cmd *cobra.Command, opts *RootOptions, intent chatIntent) error {
	releaseChart := "charts/shukra-operator"
	if intent.UseOCI {
		releaseChart = "oci://ghcr.io/sandy001-kki/charts/shukra-operator"
	}

	helmArgs := []string{
		"upgrade", "--install", "shukra-operator", releaseChart,
		"-n", "shukra-system",
		"--create-namespace",
		"--set", "leaderElection.namespace=shukra-system",
		"--wait",
		"--timeout", "10m",
	}
	if intent.ChartVersion != "" {
		helmArgs = append(helmArgs, "--version", intent.ChartVersion)
	}
	helmArgs = appendHelmConnectionArgs(opts, helmArgs)

	printTitle(cmd.OutOrStdout(), "Installing Shukra Operator")
	printKV(cmd.OutOrStdout(), "Chart", releaseChart)
	if intent.ChartVersion != "" {
		printKV(cmd.OutOrStdout(), "Chart version", intent.ChartVersion)
	}
	if err := runCommand("helm", helmArgs...); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), success("OK  Shukra Operator is installed"))
	return nil
}

func chatUninstall(cmd *cobra.Command, opts *RootOptions) error {
	helmArgs := appendHelmConnectionArgs(opts, []string{"uninstall", "shukra-operator", "-n", "shukra-system"})
	printTitle(cmd.OutOrStdout(), "Uninstalling Shukra Operator")
	if err := runCommand("helm", helmArgs...); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), success("OK  Shukra Operator is uninstalled"))
	return nil
}

func chatBootstrap(cmd *cobra.Command) error {
	printTitle(cmd.OutOrStdout(), "Bootstrapping Local Shukra Environment")
	return runCommand("powershell", "-ExecutionPolicy", "Bypass", "-File", ".\\hack\\bootstrap-local.ps1")
}
