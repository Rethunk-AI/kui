package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kui/kui/internal/db"
)

// TimestampFormat is the canonical audit timestamp format (UTC).
const TimestampFormat = "20060102T150405Z"

// Event describes an audit event for SQLite persistence.
type Event struct {
	EventType  string
	EntityType string
	EntityID   string
	UserID     *string
	Payload    interface{}
}

// RecordEvent writes an SQLite-only audit event (no Git diff).
// Use for auth and vm_lifecycle events.
func RecordEvent(ctx context.Context, database *db.DB, ev Event) error {
	if database == nil || database.SQL == nil {
		return fmt.Errorf("database not initialized")
	}
	payloadJSON, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var userID interface{}
	if ev.UserID != nil {
		userID = *ev.UserID
	} else {
		userID = nil
	}
	_, err = database.SQL.ExecContext(ctx,
		`INSERT INTO audit_events (event_type, entity_type, entity_id, user_id, payload, git_commit, created_at) VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		ev.EventType, ev.EntityType, ev.EntityID, userID, string(payloadJSON), now,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// RecordEventWithDiff writes a diff file to Git, commits it, then inserts an audit_events row with git_commit.
// Use for wizard_complete, vm_config_change, template_create.
// Order per spec: write diff → git add → commit → insert audit_events.
func RecordEventWithDiff(ctx context.Context, database *db.DB, gitBasePath string, ev Event, diffPath string, diffContent string) error {
	if database == nil || database.SQL == nil {
		return fmt.Errorf("database not initialized")
	}
	base := strings.TrimSpace(gitBasePath)
	if base == "" {
		return fmt.Errorf("git base path required")
	}

	// Ensure audit directory exists
	diffDir := filepath.Dir(diffPath)
	if err := os.MkdirAll(filepath.Join(base, diffDir), 0o755); err != nil {
		return fmt.Errorf("create audit dir: %w", err)
	}

	fullPath := filepath.Join(base, diffPath)
	if err := os.WriteFile(fullPath, []byte(diffContent), 0o644); err != nil {
		return fmt.Errorf("write diff file: %w", err)
	}

	repo, err := openOrInitRepo(base)
	if err != nil {
		return fmt.Errorf("open git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	if _, err := wt.Add(diffPath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	scope := ev.EntityType
	if scope == "" {
		scope = "wizard"
	}
	commitMsg := fmt.Sprintf("audit(%s): %s", scope, ev.EventType)
	if ev.EntityID != "" {
		commitMsg += " " + ev.EntityID
	}

	commit, err := wt.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "KUI",
			Email: "kui@local",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	sha := commit.String()

	payloadJSON, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var userID interface{}
	if ev.UserID != nil {
		userID = *ev.UserID
	} else {
		userID = nil
	}
	_, err = database.SQL.ExecContext(ctx,
		`INSERT INTO audit_events (event_type, entity_type, entity_id, user_id, payload, git_commit, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.EventType, ev.EntityType, ev.EntityID, userID, string(payloadJSON), sha, now,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func openOrInitRepo(path string) (*git.Repository, error) {
	r, err := git.PlainOpen(path)
	if err == nil {
		return r, nil
	}
	if err != git.ErrRepositoryAlreadyExists && err != git.ErrRepositoryNotExists {
		return nil, err
	}
	r, err = git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// WizardDiff produces a unified diff from empty (before) to config YAML (after).
func WizardDiff(afterContent string) string {
	lines := strings.Split(afterContent, "\n")
	var sb strings.Builder
	sb.WriteString("--- /dev/null\n")
	sb.WriteString("+++ config.yaml\n")
	sb.WriteString("@@ -0,0 +1,")
	sb.WriteString(fmt.Sprintf("%d", len(lines)))
	sb.WriteString(" @@\n")
	for _, line := range lines {
		sb.WriteString("+")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}
