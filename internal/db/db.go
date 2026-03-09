package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	conn     *sql.DB
	RepoRoot string
}

type RepoRecord struct {
	ID                int64
	Root              string
	Branch            string
	LastIndexedCommit string
	IndexedAt         time.Time
}

type FileStat struct {
	Path             string
	Extension        string
	ChurnScore       int
	RecentTouchCount int
	ExistsOnDisk     bool
}

type CommitRecord struct {
	SHA         string
	AuthorName  string
	AuthorEmail string
	CommittedAt time.Time
	Subject     string
	Body        string
}

type CochangeEdge struct {
	SourcePath    string
	RelatedPath   string
	CochangeCount int
	RecencyWeight float64
}

type DecisionNote struct {
	SourceType  string
	SourceRef   string
	Title       string
	Body        string
	CommittedAt time.Time
}

func Open(repoRoot string) (*Store, error) {
	dbDir := filepath.Join(repoRoot, ".codex-memory")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	return OpenPath(repoRoot, filepath.Join(dbDir, "codex-memory.sqlite"))
}

func OpenPath(repoRoot, dbPath string) (*Store, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{conn: conn, RepoRoot: repoRoot}
	if err := store.Init(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Init() error {
	if _, err := s.conn.Exec(schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) DB() *sql.DB {
	return s.conn
}

func (s *Store) UpsertRepo(branch, lastIndexedCommit string, indexedAt time.Time) (RepoRecord, error) {
	if _, err := s.conn.Exec(`
INSERT INTO repos (root, branch, last_indexed_commit, indexed_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(root) DO UPDATE SET
  branch = excluded.branch,
  last_indexed_commit = excluded.last_indexed_commit,
  indexed_at = excluded.indexed_at
`, s.RepoRoot, branch, lastIndexedCommit, indexedAt.UTC().Format(time.RFC3339)); err != nil {
		return RepoRecord{}, fmt.Errorf("upsert repo: %w", err)
	}

	return s.Repo()
}

func (s *Store) Repo() (RepoRecord, error) {
	row := s.conn.QueryRow(`SELECT id, root, branch, COALESCE(last_indexed_commit, ''), indexed_at FROM repos WHERE root = ?`, s.RepoRoot)
	var repo RepoRecord
	var indexedAt string
	if err := row.Scan(&repo.ID, &repo.Root, &repo.Branch, &repo.LastIndexedCommit, &indexedAt); err != nil {
		return RepoRecord{}, fmt.Errorf("read repo: %w", err)
	}
	repo.IndexedAt, _ = time.Parse(time.RFC3339, indexedAt)
	return repo, nil
}

func (s *Store) ResetRepoData(repoID int64) error {
	tables := []string{"files", "commits", "file_commits", "cochange_edges", "decision_notes"}
	for _, table := range tables {
		if _, err := s.conn.Exec(fmt.Sprintf("DELETE FROM %s WHERE repo_id = ?", table), repoID); err != nil {
			return fmt.Errorf("reset %s: %w", table, err)
		}
	}
	return nil
}

func (s *Store) InsertCommit(repoID int64, commit CommitRecord) error {
	_, err := s.conn.Exec(`
INSERT OR REPLACE INTO commits (repo_id, sha, author_name, author_email, committed_at, subject, body)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, repoID, commit.SHA, commit.AuthorName, commit.AuthorEmail, commit.CommittedAt.UTC().Format(time.RFC3339), commit.Subject, commit.Body)
	if err != nil {
		return fmt.Errorf("insert commit %s: %w", commit.SHA, err)
	}
	return nil
}

func (s *Store) InsertFileCommit(repoID int64, filePath, commitSHA string, committedAt time.Time) error {
	_, err := s.conn.Exec(`
INSERT OR REPLACE INTO file_commits (repo_id, file_path, commit_sha, committed_at)
VALUES (?, ?, ?, ?)
`, repoID, filePath, commitSHA, committedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert file commit %s -> %s: %w", filePath, commitSHA, err)
	}
	return nil
}

func (s *Store) UpsertFile(repoID int64, stat FileStat) error {
	exists := 0
	if stat.ExistsOnDisk {
		exists = 1
	}
	_, err := s.conn.Exec(`
INSERT OR REPLACE INTO files (repo_id, path, extension, churn_score, recent_touch_count, exists_on_disk)
VALUES (?, ?, ?, ?, ?, ?)
`, repoID, stat.Path, stat.Extension, stat.ChurnScore, stat.RecentTouchCount, exists)
	if err != nil {
		return fmt.Errorf("upsert file %s: %w", stat.Path, err)
	}
	return nil
}

func (s *Store) UpsertEdge(repoID int64, edge CochangeEdge) error {
	_, err := s.conn.Exec(`
INSERT OR REPLACE INTO cochange_edges (repo_id, source_path, related_path, cochange_count, recency_weight)
VALUES (?, ?, ?, ?, ?)
`, repoID, edge.SourcePath, edge.RelatedPath, edge.CochangeCount, edge.RecencyWeight)
	if err != nil {
		return fmt.Errorf("upsert edge %s -> %s: %w", edge.SourcePath, edge.RelatedPath, err)
	}
	return nil
}

func (s *Store) UpsertDecisionNote(repoID int64, note DecisionNote) error {
	_, err := s.conn.Exec(`
INSERT OR REPLACE INTO decision_notes (repo_id, source_type, source_ref, title, body, committed_at)
VALUES (?, ?, ?, ?, ?, ?)
`, repoID, note.SourceType, note.SourceRef, note.Title, note.Body, note.CommittedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert decision note %s: %w", note.SourceRef, err)
	}
	return nil
}
