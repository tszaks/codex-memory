package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runChangedNow(out io.Writer, args []string, jsonOutput bool) error {
	repoPath := optionalRepoArg(args, 0)
	indexer, err := openIndexedStore(repoPath)
	if err != nil {
		return err
	}
	defer indexer.Store.Close()

	report, err := analysis.ChangedNow(indexer.Store)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		lines := []string{report.Summary}
		for _, file := range report.Files {
			lines = append(lines, fmt.Sprintf("- %s (%s, %s)", file.Path, file.RiskLevel, file.WorkingTreeStatus))
		}
		return strings.Join(lines, "\n")
	})
}
