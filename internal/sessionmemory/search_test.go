package sessionmemory

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSearchHandlesPunctuationFiltersAndCompactResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	matching := testParsedSession("matching-session", strings.Repeat("large outcome ", 5000))
	matching.Session.Title = "Billing migration recovery"
	matching.Session.Source = "claude"
	matching.Session.CWD = "/repo/service"
	matching.Session.UpdatedAt = "2026-08-07T12:00:00Z"
	matching.Session.FilesTouched = []string{"src/db.go"}
	matching.Session.Commands = []string{strings.Repeat("go test ", 1000)}
	matching.SearchBlob = "billing migration recovery database"
	if err := store.upsert(matching, nil); err != nil {
		t.Fatal(err)
	}
	other := testParsedSession("other-session", "unrelated")
	other.Session.Title = "Billing migration in another source"
	other.Session.Source = "codex"
	other.Session.CWD = "/elsewhere"
	other.Session.UpdatedAt = "2026-08-07T12:00:00Z"
	other.SearchBlob = "billing migration"
	if err := store.upsert(other, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	results, err := SearchWithOptions(context.Background(), SessionSearchOptions{
		DBPath:   path,
		Query:    `billing: (migration?)`,
		Limit:    10,
		Source:   "claude",
		RepoRoot: "/repo",
		Files:    []string{"src/db.go"},
		After:    time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "matching-session" {
		t.Fatalf("unexpected filtered results: %+v", results)
	}
	result := results[0]
	if result.FirstUserMessage != "" || result.LastAgentMessage != "" || len(result.Snippet) > 700 {
		t.Fatalf("search result is not compact: %+v", result)
	}
	if result.Citation.SessionID != result.ID || result.Citation.Source != "claude" || result.Coverage.Mode != "full" {
		t.Fatalf("missing citation or coverage: %+v", result)
	}
	payload, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > 20_000 {
		t.Fatalf("compact search emitted %d bytes, want <= 20000", len(payload))
	}
}

func TestHybridSearchFusesLexicalAndSemanticRanks(t *testing.T) {
	t.Setenv("PALLIUM_EMBED_PROVIDER", "test-local")
	path := filepath.Join(t.TempDir(), "sessions.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id, title, blob string
		vector          []float64
	}{
		{id: "both", title: "Billing retry fix", blob: "billing retry fix", vector: []float64{1, 0}},
		{id: "semantic-only", title: "Database investigation", blob: "database investigation", vector: []float64{0.95, 0.05}},
		{id: "irrelevant", title: "UI polish", blob: "ui polish", vector: []float64{0, 1}},
	} {
		parsed := testParsedSession(item.id, item.blob)
		parsed.Session.Title = item.title
		parsed.Session.Source = "codex"
		parsed.Session.CWD = "/repo"
		parsed.SearchBlob = item.blob
		if err := store.upsert(parsed, nil); err != nil {
			t.Fatal(err)
		}
		rows, err := store.db.Query(`SELECT id,text_sha256 FROM codex_session_chunks WHERE session_id=?`, item.id)
		if err != nil {
			t.Fatal(err)
		}
		var chunks [][2]string
		for rows.Next() {
			var chunkID, sha string
			if err := rows.Scan(&chunkID, &sha); err != nil {
				t.Fatal(err)
			}
			chunks = append(chunks, [2]string{chunkID, sha})
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		for _, chunk := range chunks {
			if _, err := store.db.Exec(`INSERT INTO codex_session_embeddings(chunk_id,provider,model,dim,vector_blob,text_sha256,embedded_at) VALUES(?,?,?,?,?,?,?)`, chunk[0], "test-local", "test-model", 2, packVector(item.vector), chunk[1], "2026-08-07T12:00:00Z"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	originalEmbedTexts := embedTexts
	t.Cleanup(func() { embedTexts = originalEmbedTexts })
	embedTexts = func(context.Context, string, []string) ([][]float64, error) {
		return [][]float64{{1, 0}}, nil
	}
	results, err := SearchWithOptions(context.Background(), SessionSearchOptions{DBPath: path, Query: "billing retry", Limit: 3, Hybrid: true, Model: "test-model"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 || results[0].ID != "both" {
		t.Fatalf("unexpected fused ranking: %+v", results)
	}
	if results[0].LexicalScore == 0 || results[0].SemanticScore == 0 || !slices.Contains(results[0].Signals, "lexical") || !slices.Contains(results[0].Signals, "semantic") {
		t.Fatalf("top result was not fused from both retrievers: %+v", results[0])
	}
	if !slices.ContainsFunc(results, func(result SearchResult) bool { return result.ID == "semantic-only" && result.SemanticScore > 0 }) {
		t.Fatalf("semantic-only candidate was not included: %+v", results)
	}
}

func TestHybridSearchFallsBackToLexicalWithoutEmbeddings(t *testing.T) {
	t.Setenv("PALLIUM_EMBED_PROVIDER", "empty-provider")
	path := filepath.Join(t.TempDir(), "sessions.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := testParsedSession("lexical-fallback", "checkout recovery")
	parsed.Session.Title = "Checkout recovery"
	parsed.SearchBlob = "checkout recovery"
	if err := store.upsert(parsed, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	results, err := SearchWithOptions(context.Background(), SessionSearchOptions{DBPath: path, Query: "checkout", Limit: 3, Hybrid: true, Model: "missing-model"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Warnings) == 0 || !strings.Contains(results[0].Warnings[len(results[0].Warnings)-1], "lexical results") {
		t.Fatalf("missing explicit lexical fallback: %+v", results)
	}
}

func TestGrepTreatsPunctuationAsTextInsteadOfFTSSyntax(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	parsed := testParsedSession("grep-safe", "The auth retry failed in checkout.")
	if err := store.upsert(parsed, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	results, err := Grep(`auth: retry (failed?)`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0]["session_id"] != "grep-safe" {
		t.Fatalf("unexpected grep results: %+v", results)
	}
}
