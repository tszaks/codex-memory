package analysis

import (
	"fmt"

	"github.com/tszaks/codex-memory/internal/db"
)

type RiskReport struct {
	Path             string     `json:"path"`
	Score            int        `json:"score"`
	Level            string     `json:"level"`
	ChurnScore       int        `json:"churn_score"`
	RecentTouchCount int        `json:"recent_touch_count"`
	NeighborCount    int        `json:"neighbor_count"`
	TopNeighbors     []Neighbor `json:"top_neighbors"`
}

func Risk(store *db.Store, targetPath string) (RiskReport, error) {
	repo, err := store.Repo()
	if err != nil {
		return RiskReport{}, err
	}
	normalized, err := normalizeRepoPath(store.RepoRoot, targetPath)
	if err != nil {
		return RiskReport{}, err
	}

	row := store.DB().QueryRow(`
SELECT churn_score, recent_touch_count
FROM files
WHERE repo_id = ? AND path = ?
`, repo.ID, normalized)

	var churnScore, recentTouchCount int
	if err := row.Scan(&churnScore, &recentTouchCount); err != nil {
		return RiskReport{}, fmt.Errorf("read file risk data: %w", err)
	}

	neighborCount, err := NeighborCount(store, normalized)
	if err != nil {
		return RiskReport{}, err
	}

	neighbors, err := Neighbors(store, normalized, 5)
	if err != nil {
		return RiskReport{}, err
	}

	score := churnScore + (recentTouchCount * 3) + min(neighborCount, 10)*2
	level := "low"
	switch {
	case score >= 18:
		level = "high"
	case score >= 8:
		level = "medium"
	}

	return RiskReport{
		Path:             normalized,
		Score:            score,
		Level:            level,
		ChurnScore:       churnScore,
		RecentTouchCount: recentTouchCount,
		NeighborCount:    neighborCount,
		TopNeighbors:     neighbors,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
