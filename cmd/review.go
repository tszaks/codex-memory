package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runReview(out io.Writer, args []string, jsonOutput bool) error {
	baseRef := "HEAD~1"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		baseRef = strings.TrimSpace(args[0])
	}
	repoPath := optionalRepoArg(args, 1)
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
				lines = append(lines, fmt.Sprintf("- %s (%s)", file.Path, file.RiskLevel))
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
