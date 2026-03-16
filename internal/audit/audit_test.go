package audit

import (
	"context"
	"os"
	"path/filepath"
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
