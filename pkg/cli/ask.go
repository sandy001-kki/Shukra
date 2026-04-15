// This file implements a grounded `shukra ask` command. It exists so users can
// ask Shukra project questions and receive retrieval-based answers from the
// local documentation set even when no generative model runtime is available.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type askSource struct {
	Path  string `json:"path"`
	Score int    `json:"score"`
}

type askResponse struct {
	Question string      `json:"question"`
	Summary  string      `json:"summary"`
	Sources  []askSource `json:"sources"`
}

func newAskCommand() *cobra.Command {
	var top int
	var output string

	cmd := &cobra.Command{
		Use:   "ask QUESTION",
		Short: "Answer Shukra questions from local docs and examples",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")
			response, err := answerQuestion(question, top)
			if err != nil {
				return err
			}

			if output == "json" {
				payload, err := json.MarshalIndent(response, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			}

			printTitle(cmd.OutOrStdout(), "Shukra Ask")
			printKV(cmd.OutOrStdout(), "Question", response.Question)
			fmt.Fprintln(cmd.OutOrStdout())
			printTitle(cmd.OutOrStdout(), "Answer")
			printNote(cmd.OutOrStdout(), "-", response.Summary)
			fmt.Fprintln(cmd.OutOrStdout())
			printTitle(cmd.OutOrStdout(), "Sources")
			for _, source := range response.Sources {
				printNote(cmd.OutOrStdout(), "-", fmt.Sprintf("%s (score=%d)", source.Path, source.Score))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&top, "top", 3, "Number of matching sources to return.")
	cmd.Flags().StringVarP(&output, "output", "o", "summary", "Output format: summary or json.")
	return cmd
}

func answerQuestion(question string, top int) (*askResponse, error) {
	if top <= 0 {
		top = 3
	}
	sources := []string{
		"README.md",
		"docs/beginner-guide.md",
		"docs/getting-started.md",
		"docs/bring-your-own-cluster.md",
		"docs/helm-values.md",
		"docs/cloud-eks.md",
		"docs/cloud-gke.md",
		"docs/cloud-aks.md",
		"docs/gitops-argocd.md",
		"docs/gitops-flux.md",
		"docs/observability.md",
		"docs/cli.md",
		"docs/troubleshooting.md",
		"docs/tenancy.md",
		"docs/architecture.md",
		"docs/migration-restore-walkthrough.md",
		"examples/basic.yaml",
		"examples/ingress.yaml",
		"examples/autoscaling.yaml",
		"examples/migration.yaml",
		"examples/restore.yaml",
		"examples/paused.yaml",
		"examples/production-web.yaml",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, err
	}

	type scoredDoc struct {
		Path    string
		Score   int
		Snippet string
	}

	queryTerms := uniqueTerms(question)
	matches := make([]scoredDoc, 0, len(sources))
	for _, source := range sources {
		content, err := os.ReadFile(filepath.Join(repoRoot, source))
		if err != nil {
			continue
		}
		text := string(content)
		score := scoreText(text, queryTerms)
		if score == 0 {
			continue
		}
		matches = append(matches, scoredDoc{
			Path:    filepath.ToSlash(source),
			Score:   score,
			Snippet: bestSnippet(text, queryTerms),
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Score > matches[j].Score
	})

	if len(matches) == 0 {
		return &askResponse{
			Question: question,
			Summary:  "I could not find a grounded answer in the local Shukra docs or examples. Try rephrasing with terms like install, ingress, migration, restore, helm, or AppEnvironment.",
		}, nil
	}

	if len(matches) > top {
		matches = matches[:top]
	}

	summaryParts := make([]string, 0, len(matches))
	responseSources := make([]askSource, 0, len(matches))
	for _, match := range matches {
		summaryParts = append(summaryParts, fmt.Sprintf("%s: %s", match.Path, match.Snippet))
		responseSources = append(responseSources, askSource{Path: match.Path, Score: match.Score})
	}

	return &askResponse{
		Question: question,
		Summary:  strings.Join(summaryParts, " "),
		Sources:  responseSources,
	}, nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := wd
	for {
		if fileExists(filepath.Join(current, "go.mod")) && fileExists(filepath.Join(current, "README.md")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not locate Shukra repository root from %s", wd)
		}
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniqueTerms(question string) []string {
	normalized := strings.ToLower(question)
	replacer := strings.NewReplacer(",", " ", ".", " ", ":", " ", ";", " ", "/", " ", "\\", " ", "-", " ", "_", " ")
	normalized = replacer.Replace(normalized)
	parts := strings.Fields(normalized)
	seen := map[string]struct{}{}
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		terms = append(terms, part)
	}
	return terms
}

func scoreText(text string, terms []string) int {
	lower := strings.ToLower(text)
	score := 0
	for _, term := range terms {
		score += strings.Count(lower, term)
	}
	return score
}

func bestSnippet(text string, terms []string) string {
	lines := strings.Split(text, "\n")
	bestLine := ""
	bestScore := 0
	for _, line := range lines {
		score := scoreText(line, terms)
		if score > bestScore {
			bestScore = score
			bestLine = strings.TrimSpace(line)
		}
	}
	if bestLine == "" {
		bestLine = strings.TrimSpace(text)
	}
	if len(bestLine) > 220 {
		bestLine = bestLine[:220] + "..."
	}
	return bestLine
}
