package cmd

import (
	"fmt"
	"io"

	"github.com/tszaks/codex-memory/internal/analysis"
	"github.com/tszaks/codex-memory/internal/output"
)

func runRisk(out io.Writer, args []string, jsonOutput bool) error {
	target, err := requireArg(args, "path")
	if err != nil {
		return err
	}
	repoPath := optionalRepoArg(args, 1)
	indexer, err := openIndexedStore(repoPath)
	if err != nil {
		return err
	}
	defer indexer.Store.Close()

	report, err := analysis.Risk(indexer.Store, target)
	if err != nil {
		return err
	}

	return output.Write(out, report, jsonOutput, func() string {
		return fmt.Sprintf("%s\nRisk: %s (%d)\nChurn: %d\nRecent touches: %d\nNeighbors: %d",
			report.Path,
			report.Level,
			report.Score,
			report.ChurnScore,
			report.RecentTouchCount,
			report.NeighborCount,
		)
	})
}
