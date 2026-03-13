package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runReview(out io.Writer, args []string, jsonOutput bool) error {
	baseRef, repoPath := parseReviewArgs(args)
	indexer, err := openIndexedStore(repoPath)
	if err != nil {
		return err
	}
	defer indexer.Store.Close()

	report, err := analysis.Review(indexer.Store, baseRef)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{
			fmt.Sprintf("Review base: %s", report.BaseRef),
			fmt.Sprintf("Summary: %s", report.Summary),
		}
		if len(report.RequiredTests) > 0 {
			lines = append(lines, "", "Required tests:")
			for _, test := range report.RequiredTests {
				lines = append(lines, "- "+test)
			}
		}
		if len(report.ChangedFiles) > 0 {
			lines = append(lines, "", "Changed files:")
			for _, file := range report.ChangedFiles {
				lines = append(lines, fmt.Sprintf("- %s (%s, %s)", file.Path, file.RiskLevel, file.ChangeSource))
				for _, reason := range file.TopReasons {
					lines = append(lines, "  reason: "+reason)
				}
				for _, test := range file.SuggestedTests {
					lines = append(lines, "  test: "+test)
				}
				for _, path := range file.BlastRadius {
					lines = append(lines, "  blast: "+path)
				}
			}
		}
		if len(report.Notes) > 0 {
			lines = append(lines, "", "Notes:")
			for _, note := range report.Notes {
				lines = append(lines, "- "+note)
			}
		}
		return strings.Join(lines, "\n")
	})
}

func parseReviewArgs(args []string) (string, string) {
	baseRef := "HEAD~1"
	repoPath := "."
	if len(args) == 0 {
		return baseRef, repoPath
	}

	first := strings.TrimSpace(args[0])
	if first == "" {
		return baseRef, repoPath
	}

	if looksLikePath(first) {
		repoPath = first
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			baseRef = strings.TrimSpace(args[1])
		}
		return baseRef, repoPath
	}

	baseRef = first
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		repoPath = strings.TrimSpace(args[1])
	}
	return baseRef, repoPath
}

func looksLikePath(value string) bool {
	if value == "." || value == ".." || strings.HasPrefix(value, "/") {
		return true
	}
	if info, err := os.Stat(value); err == nil && info.IsDir() {
		return true
	}
	return false
}
