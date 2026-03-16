package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	SQL *sql.DB
}

var ddl = []string{
	`CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    created_at TEXT NOT NULL,
    updated_at TEXT NULL
);`,
	`CREATE TABLE preferences (
    user_id TEXT PRIMARY KEY,
    default_host_id TEXT NULL,
    list_view_options TEXT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT fk_preferences_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`,
	`CREATE TABLE vm_metadata (
    host_id TEXT NOT NULL,
    libvirt_uuid TEXT NOT NULL,
    claimed INTEGER NOT NULL DEFAULT 0,
    display_name TEXT NULL,
    console_preference TEXT NULL,
    last_access TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (host_id, libvirt_uuid)
);`,
	`CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    entity_type TEXT NULL,
    entity_id TEXT NULL,
    user_id TEXT NULL,
    payload TEXT NULL,
    git_commit TEXT NULL,
    created_at TEXT NOT NULL,
    CONSTRAINT fk_audit_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);`,
}

func Open(path string) (*DB, error) {
	dsn := strings.TrimSpace(path)
	if dsn == "" {
		return nil, fmt.Errorf("db path is required")
	}

	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}

	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", dsn, err)
	}

	if err := initSchema(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return &DB{SQL: sqlDB}, nil
}

func (d *DB) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}

	return d.SQL.Close()
}

// VMMetadataRow represents a row from vm_metadata.
type VMMetadataRow struct {
	HostID            string
	LibvirtUUID       string
	Claimed           bool
	DisplayName       sql.NullString
	ConsolePreference sql.NullString
	LastAccess        sql.NullString
	CreatedAt         string
	UpdatedAt         string
}

// ListVMMetadata returns all vm_metadata rows.
func (d *DB) ListVMMetadata(ctx context.Context) ([]VMMetadataRow, error) {
	rows, err := d.SQL.QueryContext(ctx, `SELECT host_id, libvirt_uuid, claimed, display_name, console_preference, last_access, created_at, updated_at FROM vm_metadata`)
	if err != nil {
		return nil, fmt.Errorf("list vm_metadata: %w", err)
	}
	defer rows.Close()

	var out []VMMetadataRow
	for rows.Next() {
		var r VMMetadataRow
		var claimed int
		if err := rows.Scan(&r.HostID, &r.LibvirtUUID, &claimed, &r.DisplayName, &r.ConsolePreference, &r.LastAccess, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm_metadata: %w", err)
		}
		r.Claimed = claimed != 0
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vm_metadata: %w", err)
	}
	return out, nil
}

func initSchema(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("enable sqlite foreign_keys: %w", err)
	}

	for _, statement := range ddl {
		if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS ` + strings.TrimPrefix(statement, `CREATE TABLE `)); err != nil {
			return fmt.Errorf("apply schema statement: %w", err)
		}
	}

	return nil
}
