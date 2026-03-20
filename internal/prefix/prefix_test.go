package prefix

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_emptyPrefixLegacyNoOp(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		p    string
	}{
		{"empty", ""},
		{"relative", "foo/bar"},
		{"absolute_unix", "/var/lib/kui/db.sqlite"},
		{"dots", "../x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Resolve("", tc.p)
			if got != tc.p {
				t.Fatalf("Resolve(\"\"): got %q want %q", got, tc.p)
			}
		})
	}
}

func TestResolve_whitespaceOnlyPrefixNoOp(t *testing.T) {
	t.Parallel()
	p := "/etc/kui/config.yaml"
	got := Resolve("   \t  ", p)
	if got != p {
		t.Fatalf("got %q want %q", got, p)
	}
}

func TestResolve_nonEmptyAbsoluteUnixStyle(t *testing.T) {
	t.Parallel()
	root := filepath.FromSlash("/tmp/kui-prefix")
	cases := []struct {
		p    string
		want string
	}{
		{"/var/lib/kui/db.sqlite", filepath.Join(root, "var", "lib", "kui", "db.sqlite")},
		{"/etc/kui/config.yaml", filepath.Join(root, "etc", "kui", "config.yaml")},
		{"/", root},
	}
	for _, tc := range cases {
		t.Run(strings.ReplaceAll(tc.p, "/", "_"), func(t *testing.T) {
			t.Parallel()
			got := Resolve(root, tc.p)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestResolve_relativeUnderPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := Resolve(root, "var/lib/kui")
	want := filepath.Join(root, "var", "lib", "kui")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_leadingSeparatorsStripped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Multiple leading slashes Clean to one; all must be stripped before join.
	got := Resolve(root, "///var/lib/kui")
	want := filepath.Join(root, "var", "lib", "kui")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_cleanTrailingDotsAndSeparators(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := Resolve(root, "/foo/bar/../baz/")
	want := filepath.Join(root, "foo", "baz")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_dotPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := Resolve(root, ".")
	want := filepath.Clean(root)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_doubleDotLexicalUnderPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Clean collapses a/b/.. -> a; result stays under prefix lexically.
	got := Resolve(root, "a/b/../c")
	want := filepath.Join(root, "a", "c")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_planRepresentativeConfigPath(t *testing.T) {
	t.Parallel()
	root := filepath.FromSlash("/tmp/kui-run")
	// Plan example: --config /etc/kui/config.yaml -> under prefix.
	got := Resolve(root, "/etc/kui/config.yaml")
	want := filepath.Join(root, "etc", "kui", "config.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_prefixTrimmed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	padded := "  " + root + "  "
	got := Resolve(padded, "/x/y")
	want := filepath.Join(root, "x", "y")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
