package sshtunnel

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseQemuSSH(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantNil bool
		wantErr bool
		host    string
		port    int
		user    string
		keyfile string
	}{
		{
			name:    "not qemu+ssh",
			uri:     "qemu:///system",
			wantNil: true,
		},
		{
			name:    "empty",
			uri:     "",
			wantNil: true,
		},
		{
			name:    "qemu+ssh with keyfile",
			uri:     "qemu+ssh://root@host.example/system?keyfile=/path/to/key",
			host:    "host.example",
			port:    22,
			user:    "root",
			keyfile: "/path/to/key",
		},
		{
			name:    "qemu+ssh with port",
			uri:     "qemu+ssh://user@host:2222/system?keyfile=/key",
			host:    "host",
			port:    2222,
			user:    "user",
			keyfile: "/key",
		},
		{
			name:    "qemu+ssh default user",
			uri:     "qemu+ssh://host/system?keyfile=/key",
			host:    "host",
			port:    22,
			user:    "root",
			keyfile: "/key",
		},
		{
			name:    "invalid uri",
			uri:     "qemu+ssh://[invalid",
			wantErr: true,
		},
		{
			name:    "missing host",
			uri:     "qemu+ssh:///system",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			uri:     "   ",
			wantNil: true,
		},
		{
			name:    "port 0 uses default 22",
			uri:     "qemu+ssh://host:0/system?keyfile=/key",
			host:    "host",
			port:    22,
			user:    "root",
			keyfile: "/key",
		},
		{
			name:    "invalid port returns error",
			uri:     "qemu+ssh://host:abc/system?keyfile=/key",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseQemuSSH(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseQemuSSH() expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseQemuSSH() error = %v", err)
				return
			}
			if tt.wantNil {
				if cfg != nil {
					t.Errorf("ParseQemuSSH() expected nil, got %+v", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatal("ParseQemuSSH() expected non-nil config")
			}
			if cfg.Host != tt.host {
				t.Errorf("Host = %q, want %q", cfg.Host, tt.host)
			}
			if cfg.Port != tt.port {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.port)
			}
			if cfg.User != tt.user {
				t.Errorf("User = %q, want %q", cfg.User, tt.user)
			}
			if cfg.Keyfile != tt.keyfile {
				t.Errorf("Keyfile = %q, want %q", cfg.Keyfile, tt.keyfile)
			}
		})
	}
}

func TestDialRemote_NilConfig_Error(t *testing.T) {
	_, err := DialRemote(context.Background(), nil, "tcp", "127.0.0.1:5900")
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestDialRemote_EmptyKeyfile_Error(t *testing.T) {
	cfg := &SSHConfig{Host: "host", Port: 22, User: "root", Keyfile: ""}
	_, err := DialRemote(context.Background(), cfg, "tcp", "127.0.0.1:5900")
	if err == nil {
		t.Fatal("expected error for empty keyfile")
	}
}

func TestDialRemote_WhitespaceKeyfile_Error(t *testing.T) {
	cfg := &SSHConfig{Host: "host", Port: 22, User: "root", Keyfile: "   "}
	_, err := DialRemote(context.Background(), cfg, "tcp", "127.0.0.1:5900")
	if err == nil {
		t.Fatal("expected error for whitespace keyfile")
	}
}

func TestDialRemote_KeyfileNotFound_Error(t *testing.T) {
	cfg := &SSHConfig{Host: "host", Port: 22, User: "root", Keyfile: filepath.Join(t.TempDir(), "nonexistent")}
	_, err := DialRemote(context.Background(), cfg, "tcp", "127.0.0.1:5900")
	if err == nil {
		t.Fatal("expected error for nonexistent keyfile")
	}
}

func TestDialRemote_InvalidKeyContent_Error(t *testing.T) {
	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "key")
	if err := os.WriteFile(keyPath, []byte("not a valid key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &SSHConfig{Host: "host", Port: 22, User: "root", Keyfile: keyPath}
	_, err := DialRemote(context.Background(), cfg, "tcp", "127.0.0.1:5900")
	if err == nil {
		t.Fatal("expected error for invalid key content")
	}
}
