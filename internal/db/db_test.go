package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpen_EmptyPath_Error(t *testing.T) {
	_, err := Open("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpen_InvalidPath_Error(t *testing.T) {
	_, err := Open("/dev/null/not-a-dir/kui.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOpen_WhitespacePath_Error(t *testing.T) {
	_, err := Open("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only path")
	}
}

func TestClose_NilDB_NoOp(t *testing.T) {
	var d *DB
	if err := d.Close(); err != nil {
		t.Errorf("Close on nil DB should be no-op, got %v", err)
	}
}

func TestClose_NilSQL_NoOp(t *testing.T) {
	d := &DB{SQL: nil}
	if err := d.Close(); err != nil {
		t.Errorf("Close on nil SQL should be no-op, got %v", err)
	}
}

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

func TestVMMetadata_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kui.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// Insert
	if err := database.InsertVMMetadata(ctx, "h1", "uuid1", false, nil); err != nil {
		t.Fatalf("insert: %v", err)
	}
	disp := "my-vm"
	if err := database.InsertVMMetadata(ctx, "h1", "uuid2", true, &disp); err != nil {
		t.Fatalf("insert with display: %v", err)
	}

	// List
	list, err := database.ListVMMetadata(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}

	// Get
	row, err := database.GetVMMetadata(ctx, "h1", "uuid1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row == nil {
		t.Fatal("expected row")
	}
	if row.Claimed {
		t.Error("expected claimed=false")
	}
	if !row.DisplayName.Valid {
		t.Error("expected display_name null")
	}

	row2, err := database.GetVMMetadata(ctx, "h1", "uuid2")
	if err != nil {
		t.Fatalf("get uuid2: %v", err)
	}
	if row2 == nil || !row2.Claimed || !row2.DisplayName.Valid || row2.DisplayName.String != "my-vm" {
		t.Errorf("unexpected row2: %+v", row2)
	}

	// Get not found
	row3, err := database.GetVMMetadata(ctx, "h1", "nonexistent")
	if err != nil || row3 != nil {
		t.Errorf("expected nil for nonexistent, got err=%v row=%v", err, row3)
	}

	// Update display_name only
	dn := "updated"
	if err := database.UpdateVMMetadata(ctx, "h1", "uuid1", &dn, nil); err != nil {
		t.Fatalf("update display: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid1")
	if !row.DisplayName.Valid || row.DisplayName.String != "updated" {
		t.Errorf("expected display_name updated, got %v", row.DisplayName)
	}

	// Update console_preference only
	cp := "vnc"
	if err := database.UpdateVMMetadata(ctx, "h1", "uuid1", nil, &cp); err != nil {
		t.Fatalf("update console: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid1")
	if !row.ConsolePreference.Valid || row.ConsolePreference.String != "vnc" {
		t.Errorf("expected console_preference vnc, got %v", row.ConsolePreference)
	}

	// Update both
	dn2 := "both"
	cp2 := "serial"
	if err := database.UpdateVMMetadata(ctx, "h1", "uuid1", &dn2, &cp2); err != nil {
		t.Fatalf("update both: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid1")
	if row.DisplayName.String != "both" || row.ConsolePreference.String != "serial" {
		t.Errorf("expected both updated, got %+v", row)
	}

	// Update neither (last_access only)
	if err := database.UpdateVMMetadata(ctx, "h1", "uuid1", nil, nil); err != nil {
		t.Fatalf("update neither: %v", err)
	}

	// UpdateVMMetadataLastAccess
	if err := database.UpdateVMMetadataLastAccess(ctx, "h1", "uuid1"); err != nil {
		t.Fatalf("update last access: %v", err)
	}

	// UpsertVMMetadataClaim - update existing
	if err := database.UpsertVMMetadataClaim(ctx, "h1", "uuid1", "claimed-name"); err != nil {
		t.Fatalf("upsert claim existing: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid1")
	if !row.Claimed || row.DisplayName.String != "claimed-name" {
		t.Errorf("expected claimed with name, got %+v", row)
	}

	// UpsertVMMetadataClaim - insert new
	if err := database.UpsertVMMetadataClaim(ctx, "h1", "uuid3", "new-claim"); err != nil {
		t.Fatalf("upsert claim new: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid3")
	if row == nil || !row.Claimed || row.DisplayName.String != "new-claim" {
		t.Errorf("expected new claimed row, got %+v", row)
	}

	// Delete
	if err := database.DeleteVMMetadata(ctx, "h1", "uuid1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	row, _ = database.GetVMMetadata(ctx, "h1", "uuid1")
	if row != nil {
		t.Error("expected row deleted")
	}
}

func TestInsertVMMetadata_Duplicate_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kui.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := database.InsertVMMetadata(ctx, "h1", "uuid1", false, nil); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := database.InsertVMMetadata(ctx, "h1", "uuid1", false, nil); err == nil {
		t.Fatal("expected error for duplicate insert")
	}
}
