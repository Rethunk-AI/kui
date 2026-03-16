package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// slugRe replaces non-alphanumeric with hyphen.
var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a human-readable name to a template_id (lowercase, alphanumeric + hyphen).
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "template"
	}
	return s
}

// BaseImage holds pool + path or pool + volume for template disk reference.
type BaseImage struct {
	Pool   string `yaml:"pool"`
	Path   string `yaml:"path,omitempty"`
	Volume string `yaml:"volume,omitempty"`
}

// Meta is the meta.yaml schema for a template.
type Meta struct {
	Name      string    `yaml:"name"`
	BaseImage BaseImage `yaml:"base_image"`
	CPU       int       `yaml:"cpu,omitempty"`
	RAMMB     int       `yaml:"ram_mb,omitempty"`
	Network   string    `yaml:"network,omitempty"`
}

// TemplateInfo is a list item for GET /api/templates.
type TemplateInfo struct {
	TemplateID string    `json:"template_id"`
	Name       string    `json:"name"`
	BaseImage  BaseImage `json:"base_image"`
	CPU        int       `json:"cpu"`
	RAMMB      int       `json:"ram_mb"`
	Network    string    `json:"network"`
	CreatedAt  string    `json:"created_at"`
}

// ParseMeta parses meta.yaml content into Meta.
func ParseMeta(yamlContent []byte) (*Meta, error) {
	var m Meta
	if err := yaml.Unmarshal(yamlContent, &m); err != nil {
		return nil, fmt.Errorf("parse meta.yaml: %w", err)
	}
	if strings.TrimSpace(m.Name) == "" {
		return nil, fmt.Errorf("meta.yaml: name is required")
	}
	if strings.TrimSpace(m.BaseImage.Pool) == "" {
		return nil, fmt.Errorf("meta.yaml: base_image.pool is required")
	}
	hasPath := strings.TrimSpace(m.BaseImage.Path) != ""
	hasVol := strings.TrimSpace(m.BaseImage.Volume) != ""
	if hasPath == hasVol {
		return nil, fmt.Errorf("meta.yaml: exactly one of base_image.path or base_image.volume is required")
	}
	return &m, nil
}

// WriteMeta writes Meta to a file path.
func WriteMeta(path string, m *Meta) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	return nil
}

// TemplateExists returns true if the template directory exists.
func TemplateExists(gitBase string, templateID string) (bool, error) {
	p := filepath.Join(strings.TrimSpace(gitBase), "templates", templateID)
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// CreateTemplateDir creates the template directory and returns its path.
func CreateTemplateDir(gitBase string, templateID string) (string, error) {
	base := strings.TrimSpace(gitBase)
	if base == "" {
		return "", fmt.Errorf("git base path is required")
	}
	if strings.TrimSpace(templateID) == "" {
		return "", fmt.Errorf("template_id is required")
	}
	p := filepath.Join(base, "templates", templateID)
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", fmt.Errorf("create template dir: %w", err)
	}
	return p, nil
}

// ListTemplates enumerates template directories under git_base/templates/ and parses meta.yaml.
// Malformed templates are skipped; the list continues.
func ListTemplates(gitBase string) ([]TemplateInfo, error) {
	base := strings.TrimSpace(gitBase)
	if base == "" {
		return nil, fmt.Errorf("git base path is required")
	}
	templatesDir := filepath.Join(base, "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read templates dir: %w", err)
	}
	var out []TemplateInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		templateID := e.Name()
		metaPath := filepath.Join(templatesDir, templateID, "meta.yaml")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		meta, err := ParseMeta(data)
		if err != nil {
			continue
		}
		createdAt := ""
		if fi, err := os.Stat(metaPath); err == nil {
			createdAt = fi.ModTime().UTC().Format(time.RFC3339)
		}
		out = append(out, TemplateInfo{
			TemplateID: templateID,
			Name:       meta.Name,
			BaseImage:  meta.BaseImage,
			CPU:        meta.CPU,
			RAMMB:      meta.RAMMB,
			Network:    meta.Network,
			CreatedAt:  createdAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].CreatedAt, out[j].CreatedAt
		if a == "" && b == "" {
			return out[i].TemplateID < out[j].TemplateID
		}
		if a == "" {
			return false
		}
		if b == "" {
			return true
		}
		return a > b
	})
	return out, nil
}
