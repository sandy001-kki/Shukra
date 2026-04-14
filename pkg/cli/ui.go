// This file provides small terminal presentation helpers for the Shukra CLI.
// It exists so command output feels structured and intentional instead of being
// a stream of unrelated print statements.
package cli

import (
	"fmt"
	"io"
	"os"
)

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiCyan  = "\x1b[36m"
	ansiGreen = "\x1b[32m"
	ansiDim   = "\x1b[2m"
)

func useANSI() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func header(text string) string {
	if !useANSI() {
		return text
	}
	return ansiBold + ansiCyan + text + ansiReset
}

func success(text string) string {
	if !useANSI() {
		return text
	}
	return ansiGreen + text + ansiReset
}

func muted(text string) string {
	if !useANSI() {
		return text
	}
	return ansiDim + text + ansiReset
}

func printTitle(out io.Writer, title string) {
	fmt.Fprintln(out, header(title))
}

func printKV(out io.Writer, key, value string) {
	fmt.Fprintf(out, "  %-18s %s\n", key, value)
}

func printNote(out io.Writer, label, value string) {
	fmt.Fprintf(out, "%s %s\n", muted(label), value)
}
