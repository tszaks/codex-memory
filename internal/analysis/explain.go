package analysis

import (
	"fmt"

	"github.com/tszaks/codex-memory/internal/db"
)

type CommitSummary struct {
	SHA         string `json:"sha"`
	Subject     string `json:"subject"`
	CommittedAt string `json:"committed_at"`
}

type ExplainReport struct {
	Path          string          `json:"path"`
	Risk          RiskReport      `json:"risk"`
	RecentCommits []CommitSummary `json:"recent_commits"`
	Decisions     []Decision      `json:"decisions"`
	Neighbors     []Neighbor      `json:"neighbors"`
}

func Explain(store *db.Store, targetPath string) (ExplainReport, error) {
	risk, err := Risk(store, targetPath)
	if err != nil {
		return ExplainReport{}, err
	}

	repo, err := store.Repo()
	if err != nil {
		return ExplainReport{}, err
	}

	rows, err := store.DB().Query(`
SELECT c.sha, c.subject, c.committed_at
FROM file_commits fc
JOIN commits c
  ON c.repo_id = fc.repo_id AND c.sha = fc.commit_sha
WHERE fc.repo_id = ? AND fc.file_path = ?
ORDER BY c.committed_at DESC
LIMIT 5
`, repo.ID, risk.Path)
	if err != nil {
		return ExplainReport{}, fmt.Errorf("query recent commits: %w", err)
	}
	defer rows.Close()

	commits := make([]CommitSummary, 0)
	for rows.Next() {
		var item CommitSummary
		if err := rows.Scan(&item.SHA, &item.Subject, &item.CommittedAt); err != nil {
			return ExplainReport{}, fmt.Errorf("scan recent commit: %w", err)
		}
		commits = append(commits, item)
	}

	decisions, err := Decisions(store, targetPath, 3)
	if err != nil {
		return ExplainReport{}, err
	}

	return ExplainReport{
		Path:          risk.Path,
		Risk:          risk,
		RecentCommits: commits,
		Decisions:     decisions,
		Neighbors:     risk.TopNeighbors,
	}, nil
}
