package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Init ensures the git storage layout exists at the provided base path.
//
// Creates the base directory if missing, runs git init when the path is not
// already a git repository (idempotent), then creates:
// - <path>/templates
// - <path>/audit
// - <path>/audit/vm
// - <path>/audit/template
// - <path>/audit/wizard
func Init(path string) error {
	base := strings.TrimSpace(path)
	if base == "" {
		return fmt.Errorf("git base path is required")
	}

	if err := ensureGitRepo(base); err != nil {
		return err
	}

	directories := []string{
		base,
		filepath.Join(base, "templates"),
		filepath.Join(base, "audit"),
		filepath.Join(base, "audit", "vm"),
		filepath.Join(base, "audit", "template"),
		filepath.Join(base, "audit", "wizard"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}

	return nil
}

// ensureGitRepo runs git init when base is not already a git repository.
// Idempotent: no-op if base already contains a .git directory or worktree.
func ensureGitRepo(base string) error {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("create git base %q: %w", base, err)
	}

	check := exec.Command("git", "rev-parse", "--git-dir")
	check.Dir = base
	if err := check.Run(); err == nil {
		return nil
	}

	initCmd := exec.Command("git", "init")
	initCmd.Dir = base
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	return nil
}
