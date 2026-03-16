package db

import (
	"testing"

	"path/filepath"
)

func TestOpenAppliesSchema(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "kui.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}()

	rows, err := database.SQL.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name IN ('users','preferences','vm_metadata','audit_events') ORDER BY name`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	observed := map[string]struct{}{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table row: %v", err)
		}
		observed[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table rows: %v", err)
	}

	expected := []string{"audit_events", "preferences", "users", "vm_metadata"}
	for _, table := range expected {
		if _, ok := observed[table]; !ok {
			t.Fatalf("missing table %q", table)
		}
	}

	if _, err := database.SQL.Exec(`INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		"user-1",
		"admin",
		"hash",
		"2026-03-16T00:00:00Z"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := database.SQL.Exec(`INSERT INTO preferences (user_id, updated_at) VALUES (?, ?)`, "user-1", "2026-03-16T00:00:01Z"); err != nil {
		t.Fatalf("insert preferences: %v", err)
	}

	if _, err := database.SQL.Exec(`INSERT INTO vm_metadata (host_id, libvirt_uuid, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"host-1",
		"uuid-1",
		"2026-03-16T00:00:02Z",
		"2026-03-16T00:00:03Z"); err != nil {
		t.Fatalf("insert vm_metadata: %v", err)
	}

	if _, err := database.SQL.Exec(`INSERT INTO audit_events (event_type, created_at, user_id) VALUES (?, ?, ?)`,
		"vm_lifecycle",
		"2026-03-16T00:00:04Z",
		"user-1"); err != nil {
		t.Fatalf("insert audit_events: %v", err)
	}
}
