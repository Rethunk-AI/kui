package prefix

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Parallel()
	tmp := filepath.Join("tmp", "kui-prefix")
	prefix := filepath.FromSlash(tmp)

	tests := []struct {
		name   string
		prefix string
		p      string
		want   string
	}{
		{
			name:   "empty prefix no-op empty p",
			prefix: "",
			p:      "",
			want:   "",
		},
		{
			name:   "empty prefix no-op relative",
			prefix: "",
			p:      "foo/bar",
			want:   "foo/bar",
		},
		{
			name:   "empty prefix no-op double-dot relative",
			prefix: "",
			p:      "../x",
			want:   "../x",
		},
		{
			name:   "whitespace-only prefix no-op",
			prefix: "   \t",
			p:      "/abs",
			want:   "/abs",
		},
		{
			name:   "non-empty relative under prefix",
			prefix: prefix,
			p:      "foo/bar",
			want:   filepath.Join(prefix, "foo", "bar"),
		},
		{
			name:   "non-empty absolute unix-style under prefix",
			prefix: prefix,
			p:      "/var/lib/kui/db.sqlite",
			want:   filepath.Join(prefix, "var", "lib", "kui", "db.sqlite"),
		},
		{
			name:   "plan representative /etc/kui config",
			prefix: filepath.FromSlash("/tmp/kui-run"),
			p:      "/etc/kui/config.yaml",
			want:   filepath.Join(filepath.FromSlash("/tmp/kui-run"), "etc", "kui", "config.yaml"),
		},
		{
			name:   "absolute root only maps to prefix",
			prefix: prefix,
			p:      "/",
			want:   filepath.Clean(prefix),
		},
		{
			name:   "multiple leading slashes cleaned then stripped",
			prefix: prefix,
			p:      "///var/lib/kui",
			want:   filepath.Join(prefix, "var", "lib", "kui"),
		},
		{
			name:   "clean collapses dot and separator",
			prefix: prefix,
			p:      "./foo//bar/../baz",
			want:   filepath.Join(prefix, "foo", "baz"),
		},
		{
			name:   "clean trailing slash and double-dot",
			prefix: prefix,
			p:      "/foo/bar/../baz/",
			want:   filepath.Join(prefix, "foo", "baz"),
		},
		{
			name:   "dot-only becomes prefix",
			prefix: prefix,
			p:      ".",
			want:   filepath.Clean(prefix),
		},
		{
			name:   "parent segments after clean",
			prefix: prefix,
			p:      filepath.Join("a", "b", "..", "c"),
			want:   filepath.Join(prefix, "a", "c"),
		},
		{
			name:   "trimmed prefix used in join",
			prefix: "  " + prefix + "  ",
			p:      "x",
			want:   filepath.Join(prefix, "x"),
		},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name   string
			prefix string
			p      string
			want   string
		}{
			{
				name:   "windows drive absolute under prefix",
				prefix: `D:\prefix`,
				p:      `C:\var\lib\kui`,
				want:   filepath.Join(`D:\prefix`, "var", "lib", "kui"),
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Resolve(tt.prefix, tt.p)
			if got != tt.want {
				t.Fatalf("Resolve(%q, %q) = %q; want %q", tt.prefix, tt.p, got, tt.want)
			}
		})
	}
}

func TestResolve_emptyPathUnderPrefix(t *testing.T) {
	t.Parallel()
	prefix := filepath.Join("tmp", "pfx")
	got := Resolve(prefix, "")
	want := filepath.Clean(prefix)
	if got != want {
		t.Fatalf("Resolve(%q, %q) = %q; want %q", prefix, "", got, want)
	}
}

func TestResolve_rootOnlyUnderPrefix(t *testing.T) {
	t.Parallel()
	prefix := filepath.Join("tmp", "pfx")
	got := Resolve(prefix, string(filepath.Separator))
	if got != filepath.Clean(prefix) {
		t.Fatalf("Resolve root-only: got %q want %q", got, filepath.Clean(prefix))
	}
}

func TestResolve_preservesLegacyWhenPrefixEmpty(t *testing.T) {
	t.Parallel()
	abs := filepath.FromSlash("/etc/kui/config.yaml")
	if runtime.GOOS == "windows" {
		abs = `C:\etc\kui\config.yaml`
	}
	if got := Resolve("", abs); got != abs {
		t.Fatalf("empty prefix: got %q want %q", got, abs)
	}
	rel := filepath.Join("..", "data", "x")
	if got := Resolve("", rel); got != rel {
		t.Fatalf("empty prefix relative: got %q want %q", got, rel)
	}
}

func TestResolve_multipleLeadingSeparators(t *testing.T) {
	t.Parallel()
	prefix := filepath.Join("p", "fx")
	p := string(filepath.Separator) + string(filepath.Separator) + filepath.Join("a", "b")
	got := Resolve(prefix, p)
	want := filepath.Join(prefix, "a", "b")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_relativeWithRealTempDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := Resolve(root, "var/lib/kui")
	want := filepath.Join(root, "var", "lib", "kui")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_prefixTrimmedWithRealTempDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	padded := "  " + root + "  "
	got := Resolve(padded, "/x/y")
	want := filepath.Join(root, "x", "y")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_uncStyleOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("UNC paths are Windows-specific")
	}
	t.Parallel()
	prefix := `D:\pfx`
	p := `\\server\share\folder\file.txt`
	got := Resolve(prefix, p)
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("expected under prefix: %q", got)
	}
	if !strings.Contains(got, "folder") {
		t.Fatalf("expected folder segment in %q", got)
	}
}
