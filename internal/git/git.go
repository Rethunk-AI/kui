package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init ensures the git storage layout exists at the provided base path.
//
// It currently creates the required directories only:
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
