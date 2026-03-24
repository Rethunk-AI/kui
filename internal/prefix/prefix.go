// Package prefix provides chroot-style path resolution for a runtime filesystem root.
//
// # Mental model
//
// When a non-empty prefix is set, Resolve treats it like the root of a virtual tree
// (similar in spirit to chroot(8), but without any kernel namespace or syscall): path
// strings that look “absolute” (e.g. /var/lib/kui/db.sqlite) are not opened on the host
// root; they are opened as filepath.Join(prefix, "var", "lib", "kui", "db.sqlite").
//
// # Symlinks, containment, and ".."
//
// Resolve is a pure string operation: it does not touch the filesystem, does not
// evaluate symlinks, and does not enforce that the result stays lexically inside
// prefix. A cleaned path containing ".." segments can still produce a joined path
// that escapes prefix (e.g. Resolve("/tmp/pfx", "../etc/passwd")). Callers that need
// hard containment should validate after resolution (for example lexical checks or
// filepath.EvalSymlinks on prefix at startup, as described in project docs).
//
// Relative inputs with a non-empty prefix are joined under prefix (prefix-relative),
// not relative to the process current working directory.
package prefix

import (
	"path/filepath"
	"strings"
)

// Resolve maps the logical path p under an optional runtime prefix.
//
// If prefix is empty after strings.TrimSpace, p is returned unchanged. This preserves
// legacy behavior: relative paths stay relative as provided (CWD-relative at call
// sites), and absolute paths stay as given. No filepath.Clean is applied in that case.
//
// If prefix is non-empty after trim, filepath.Clean(p) is computed, the platform root
// is removed (leading separators after the volume on Windows, leading separators on
// Unix), and the remainder is joined with the trimmed prefix. Both relative and
// absolute p end up under prefix.
func Resolve(prefix, p string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return p
	}
	cleaned := filepath.Clean(p)
	remainder := stripLeadingRoot(cleaned)
	return filepath.Join(prefix, remainder)
}

// stripLeadingRoot removes the leading volume (if any) and root separators from a
// Clean path so the result is suitable as a suffix for filepath.Join(prefix, …).
func stripLeadingRoot(cleaned string) string {
	vol := filepath.VolumeName(cleaned)
	s := cleaned[len(vol):]
	s = strings.TrimLeft(s, `/\`)
	return s
}
