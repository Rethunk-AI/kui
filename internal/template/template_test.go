package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Ubuntu 22.04 Base", "ubuntu-22-04-base"},
		{"my-vm", "my-vm"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"", "template"},
		{"---", "template"},
	}
	for _, tt := range tests {
		got := Slugify(tt.in)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseMeta_InvalidYAML(t *testing.T) {
	_, err := ParseMeta([]byte("invalid: yaml: content: ["))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseMeta_EmptyName(t *testing.T) {
	_, err := ParseMeta([]byte(`name: ""
base_image:
  pool: default
  volume: v
`))
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestParseMeta_EmptyPool(t *testing.T) {
	_, err := ParseMeta([]byte(`name: x
base_image:
  pool: ""
  volume: v
`))
	if err == nil {
		t.Error("expected error for empty pool")
	}
}

func TestParseMeta_BothPathAndVolume(t *testing.T) {
	_, err := ParseMeta([]byte(`name: x
base_image:
  pool: p
  path: /a
  volume: v
`))
	if err == nil {
		t.Error("expected error when both path and volume")
	}
}

func TestParseMeta_PathOnly(t *testing.T) {
	meta, err := ParseMeta([]byte(`name: x
base_image:
  pool: p
  path: /a/b
`))
	if err != nil {
		t.Fatalf("ParseMeta: %v", err)
	}
	if meta.BaseImage.Path != "/a/b" || meta.BaseImage.Volume != "" {
		t.Errorf("expected path only: %+v", meta.BaseImage)
	}
}

func TestParseMeta(t *testing.T) {
	valid := []byte(`name: Ubuntu 22.04
base_image:
  pool: default
  volume: ubuntu-2204.qcow2
cpu: 2
ram_mb: 2048
network: default
`)
	meta, err := ParseMeta(valid)
	if err != nil {
		t.Fatalf("ParseMeta: %v", err)
	}
	if meta.Name != "Ubuntu 22.04" {
		t.Errorf("Name = %q", meta.Name)
	}
	if meta.BaseImage.Pool != "default" || meta.BaseImage.Volume != "ubuntu-2204.qcow2" {
		t.Errorf("BaseImage = %+v", meta.BaseImage)
	}
	if meta.CPU != 2 || meta.RAMMB != 2048 || meta.Network != "default" {
		t.Errorf("cpu/ram/network = %d/%d/%q", meta.CPU, meta.RAMMB, meta.Network)
	}

	invalid := []byte(`name: x
base_image:
  pool: default
`)
	_, err = ParseMeta(invalid)
	if err == nil {
		t.Error("ParseMeta should fail when neither path nor volume")
	}
}

func TestListTemplates(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates", "t1"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := &Meta{
		Name:      "Test",
		BaseImage: BaseImage{Pool: "default", Volume: "t1.qcow2"},
	}
	if err := WriteMeta(filepath.Join(dir, "templates", "t1", "meta.yaml"), meta); err != nil {
		t.Fatal(err)
	}
	list, err := ListTemplates(dir)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].TemplateID != "t1" || list[0].Name != "Test" {
		t.Errorf("list[0] = %+v", list[0])
	}
}

func TestTemplateExists(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "git")
	if err := os.MkdirAll(filepath.Join(base, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "templates", "t1"), 0o755); err != nil {
		t.Fatal(err)
	}

	exists, err := TemplateExists(base, "t1")
	if err != nil {
		t.Fatalf("TemplateExists: %v", err)
	}
	if !exists {
		t.Error("expected t1 to exist")
	}

	exists, err = TemplateExists(base, "nonexistent")
	if err != nil {
		t.Fatalf("TemplateExists: %v", err)
	}
	if exists {
		t.Error("expected nonexistent to not exist")
	}

	// Path exists but is a file, not a directory
	if err := os.WriteFile(filepath.Join(base, "templates", "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	exists, err = TemplateExists(base, "file")
	if err != nil {
		t.Fatalf("TemplateExists file: %v", err)
	}
	if exists {
		t.Error("expected file (not dir) to return false")
	}
}

func TestCreateTemplateDir(t *testing.T) {
	dir := t.TempDir()

	_, err := CreateTemplateDir("", "t1")
	if err == nil {
		t.Error("expected error for empty base")
	}

	_, err = CreateTemplateDir(dir, "  ")
	if err == nil {
		t.Error("expected error for empty templateID")
	}

	p, err := CreateTemplateDir(dir, "new-tpl")
	if err != nil {
		t.Fatalf("CreateTemplateDir: %v", err)
	}
	if p != filepath.Join(dir, "templates", "new-tpl") {
		t.Errorf("path = %q", p)
	}
}

func TestWriteMeta(t *testing.T) {
	meta := &Meta{Name: "x", BaseImage: BaseImage{Pool: "p", Volume: "v"}}
	path := filepath.Join(t.TempDir(), "meta.yaml")
	if err := WriteMeta(path, meta); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseMeta(data)
	if err != nil {
		t.Fatalf("ParseMeta: %v", err)
	}
	if parsed.Name != meta.Name {
		t.Errorf("parsed.Name = %q", parsed.Name)
	}
}

func TestListTemplates_EmptyBase_Error(t *testing.T) {
	_, err := ListTemplates("")
	if err == nil {
		t.Error("expected error for empty base")
	}
}

func TestListTemplates_NoTemplatesDir_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	list, err := ListTemplates(dir)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil when templates dir missing, got %v", list)
	}
}

func TestListTemplates_SkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(templatesDir, "bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "bad", "meta.yaml"), []byte("invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(templatesDir, "good"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := &Meta{Name: "Good", BaseImage: BaseImage{Pool: "p", Volume: "v"}}
	if err := WriteMeta(filepath.Join(templatesDir, "good", "meta.yaml"), meta); err != nil {
		t.Fatal(err)
	}

	list, err := ListTemplates(dir)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 1 || list[0].TemplateID != "good" {
		t.Errorf("expected one good template, got %v", list)
	}
}

func TestListTemplates_SkipsDirWithoutMeta(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "no-meta"), 0o755); err != nil {
		t.Fatal(err)
	}
	list, err := ListTemplates(dir)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

func TestListTemplates_SkipsNonDir(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := ListTemplates(dir)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list when file in templates, got %v", list)
	}
}
