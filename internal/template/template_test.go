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
