// Package prefix provides chroot-style lexical resolution of filesystem path strings
// against an optional runtime root (install prefix).
//
// Resolve is not a real chroot(2): it only rewrites path strings. Symlinks inside
// the prefix tree can still point outside it; optional containment checks (e.g.
// EvalSymlinks on the prefix plus lexical HasPrefix after Clean) are left to callers.
//
// Lexical ".." segments in the input are handled by filepath.Clean before joining;
// the joined result may still escape the prefix (e.g. Join(prefix, "..")) unless
// callers add optional validation—that is out of scope for Resolve itself.
package prefix

import (
	"path/filepath"
	"strings"
)

// Resolve returns the effective path for p under runtime prefix.
//
// Prefix is considered empty if strings.TrimSpace(prefix) yields ""; in that case p
// is returned unchanged (legacy: relative stays as passed; absolute stays absolute).
//
// When prefix is non-empty: cleaned := filepath.Clean(p); leading path separators are
// removed from cleaned after the volume name (filepath.VolumeName), so absolute and
// relative inputs both map under prefix via filepath.Join(prefix, remainder). On
// Windows, volume roots (e.g. C:, UNC \\host\share) are stripped per filepath rules
// before removing leading separators.
func Resolve(prefix string, p string) string {
	pref := strings.TrimSpace(prefix)
	if pref == "" {
		return p
	}
	cleaned := filepath.Clean(p)
	vol := filepath.VolumeName(cleaned)
	rest := cleaned[len(vol):]
	rest = strings.TrimLeft(rest, `/\`)
	return filepath.Join(pref, rest)
}
