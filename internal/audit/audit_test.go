package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kui/kui/internal/db"
)

func TestRecordEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	err = RecordEvent(ctx, database, Event{
		EventType:  "auth",
		EntityType: "auth",
		EntityID:   "user-1",
		UserID:     nil,
		Payload: map[string]string{
			"action": "login",
			"result": "success",
		},
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	var count int
	err = database.SQL.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'auth'`).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 audit row, got %d", count)
	}
}

func TestRecordEventWithDiff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	gitPath := filepath.Join(dir, "git")
	if err := os.MkdirAll(gitPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	diffPath := "audit/wizard/20260102T150405Z.diff"
	diffContent := WizardDiff("hosts:\n  - id: local\n")
	err = RecordEventWithDiff(ctx, database, gitPath, Event{
		EventType:  "wizard_complete",
		EntityType: "wizard",
		EntityID:   "latest",
		UserID:     nil,
		Payload: map[string]string{
			"action": "wizard_complete",
			"result": "success",
		},
	}, diffPath, diffContent)
	if err != nil {
		t.Fatalf("RecordEventWithDiff: %v", err)
	}

	var gitCommit string
	err = database.SQL.QueryRow(`SELECT git_commit FROM audit_events WHERE event_type = 'wizard_complete'`).Scan(&gitCommit)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if gitCommit == "" {
		t.Error("expected git_commit to be set")
	}

	fullPath := filepath.Join(gitPath, diffPath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("diff file not found: %v", err)
	}
}

func TestWizardDiff(t *testing.T) {
	got := WizardDiff("a\nb")
	want := "--- /dev/null\n+++ config.yaml\n@@ -0,0 +1,2 @@\n+a\n+b\n"
	if got != want {
		t.Errorf("WizardDiff:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestRecordEvent_NilDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := RecordEvent(ctx, nil, Event{
		EventType: "auth",
		EntityType: "auth",
		EntityID:   "user-1",
		Payload:    map[string]string{"action": "login"},
	})
	if err == nil {
		t.Fatal("expected error for nil database")
	}
	if !strings.Contains(err.Error(), "database not initialized") {
		t.Errorf("expected database not initialized, got %v", err)
	}
}

func TestRecordEventWithDiff_NilDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	_ = os.MkdirAll(gitPath, 0o755)

	err := RecordEventWithDiff(ctx, nil, gitPath, Event{
		EventType:  "wizard_complete",
		EntityType: "wizard",
		EntityID:   "latest",
		Payload:    map[string]string{},
	}, "audit/wizard/test.diff", "diff content")
	if err == nil {
		t.Fatal("expected error for nil database")
	}
	if !strings.Contains(err.Error(), "database not initialized") {
		t.Errorf("expected database not initialized, got %v", err)
	}
}

func TestRecordEventWithDiff_EmptyGitPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	err = RecordEventWithDiff(ctx, database, "", Event{
		EventType:  "wizard_complete",
		EntityType: "wizard",
		EntityID:   "latest",
		Payload:    map[string]string{},
	}, "audit/wizard/test.diff", "diff content")
	if err == nil {
		t.Fatal("expected error for empty git path")
	}
	if !strings.Contains(err.Error(), "git base path required") {
		t.Errorf("expected git base path required, got %v", err)
	}
}

func TestRecordEventWithDiff_OpenOrInitRepo_ExistingRepo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	gitPath := filepath.Join(dir, "git")
	if err := os.MkdirAll(gitPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// First call creates repo via openOrInitRepo (init path)
	diffPath := "audit/wizard/20260102T150405Z.diff"
	diffContent := WizardDiff("hosts:\n  - id: local\n")
	err = RecordEventWithDiff(ctx, database, gitPath, Event{
		EventType:  "wizard_complete",
		EntityType: "wizard",
		EntityID:   "latest",
		UserID:     nil,
		Payload:    map[string]string{"action": "wizard_complete"},
	}, diffPath, diffContent)
	if err != nil {
		t.Fatalf("RecordEventWithDiff: %v", err)
	}

	// Second call uses existing repo (open path)
	diffPath2 := "audit/wizard/20260102T150406Z.diff"
	diffContent2 := WizardDiff("hosts:\n  - id: local\n  - id: remote\n")
	err = RecordEventWithDiff(ctx, database, gitPath, Event{
		EventType:  "wizard_complete",
		EntityType: "wizard",
		EntityID:   "latest",
		UserID:     nil,
		Payload:    map[string]string{"action": "wizard_complete"},
	}, diffPath2, diffContent2)
	if err != nil {
		t.Fatalf("RecordEventWithDiff (existing repo): %v", err)
	}
}

func TestCommitPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	if err := os.MkdirAll(gitPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a file to commit
	auditDir := filepath.Join(gitPath, "audit", "test")
	if err := os.MkdirAll(auditDir, 0o755); err != nil {
		t.Fatalf("mkdir audit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitPath, "audit", "test", "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sha, err := CommitPaths(gitPath, []string{"audit/test/file.txt"}, "test commit")
	if err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}
	if sha == "" {
		t.Error("expected non-empty commit SHA")
	}
}

func TestCommitPaths_EmptyBase(t *testing.T) {
	t.Parallel()
	_, err := CommitPaths("", []string{"file.txt"}, "msg")
	if err == nil {
		t.Fatal("expected error for empty base")
	}
	if !strings.Contains(err.Error(), "git base path required") {
		t.Errorf("expected git base path required, got %v", err)
	}
}

func TestRecordEvent_WithUserID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// Create user for foreign key
	now := "2026-03-16T00:00:00Z"
	_, err = database.SQL.Exec(`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-123", "testuser", "hash", "admin", now, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	userID := "user-123"
	err = RecordEvent(ctx, database, Event{
		EventType:  "auth",
		EntityType: "auth",
		EntityID:   "user-1",
		UserID:     &userID,
		Payload:    map[string]string{"action": "login", "result": "success"},
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	var storedUserID string
	err = database.SQL.QueryRow(`SELECT user_id FROM audit_events WHERE event_type = 'auth'`).Scan(&storedUserID)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if storedUserID != userID {
		t.Errorf("user_id = %q, want %q", storedUserID, userID)
	}
}
