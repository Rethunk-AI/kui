// Package prefix provides chroot-style path resolution for a runtime install root.
//
// Resolve maps logical paths (including absolute-looking strings such as
// "/var/lib/kui/db.sqlite") under a single directory prefix. This is analogous
// to changing the filesystem root (chroot) for application opens only—the
// process is not placed in a real chroot(2) jail.
//
// Semantics use path/filepath so separators and volume roots follow the host OS
// (Unix: single root; Windows: drive letters and UNC \\server\share prefixes).
//
// Optional containment hardening (e.g. rejecting ".." that lexically escapes the
// prefix, or comparing against EvalSymlinks(prefix)) is not performed here;
// callers may add startup checks. Symlinks inside the prefix tree can still
// point outside it unless separately validated.
package prefix

import (
	"path/filepath"
	"strings"
)

// Resolve returns p unchanged when prefix is empty (after strings.TrimSpace);
// otherwise it returns filepath.Join(trimmedPrefix, remainder), where remainder
// is filepath.Clean(p) with its volume name (if any) and all leading path
// separators removed. Relative inputs therefore become prefix-relative; Unix
// absolute paths lose the leading slash; Windows absolute paths lose the drive
// or UNC root so the path is rooted under prefix.
func Resolve(prefix, p string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return p
	}
	cleaned := filepath.Clean(p)
	vol := filepath.VolumeName(cleaned)
	rest := cleaned[len(vol):]
	rest = strings.TrimLeft(rest, `/\`)
	return filepath.Join(prefix, rest)
}
